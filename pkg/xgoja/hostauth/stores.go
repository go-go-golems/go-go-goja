package hostauth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/appauth"
	appauthsql "github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/appauth/sqlstore"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/audit"
	auditsql "github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/audit/sqlstore"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/capability"
	capabilitysql "github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/capability/sqlstore"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/sessionauth"
	sessionauthsql "github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/sessionauth/sqlstore"
)

// StoreBundle contains the concrete stores built from ResolvedStoresConfig.
type StoreBundle struct {
	Session    sessionauth.Store
	Audit      audit.Store
	AppAuth    AppAuthStores
	Capability capability.Store

	Closers []func(context.Context) error
}

// Close closes all resources owned by the bundle.
func (b *StoreBundle) Close(ctx context.Context) error {
	if b == nil {
		return nil
	}
	return closeAll(ctx, b.Closers)
}

// BuildStores creates all host auth stores described by cfg. SQL DB handles are
// shared when store configs resolve to the same driver and DSN.
func BuildStores(ctx context.Context, cfg ResolvedStoresConfig) (*StoreBundle, error) {
	builder := storeBuilder{dbs: map[sqlDBKey]*sql.DB{}}
	bundle, err := builder.build(ctx, cfg)
	if err != nil {
		_ = closeAll(ctx, builder.closers)
		return nil, err
	}
	return bundle, nil
}

type storeBuilder struct {
	dbs     map[sqlDBKey]*sql.DB
	closers []func(context.Context) error
}

type sqlDBKey struct {
	driver StoreDriver
	dsn    string
}

func (b *storeBuilder) build(ctx context.Context, cfg ResolvedStoresConfig) (*StoreBundle, error) {
	sessionStore, err := b.buildSessionStore(ctx, cfg.Session)
	if err != nil {
		return nil, err
	}
	auditStore, err := b.buildAuditStore(ctx, cfg.Audit)
	if err != nil {
		return nil, err
	}
	appAuthStores, err := b.buildAppAuthStores(ctx, cfg.AppAuth)
	if err != nil {
		return nil, err
	}
	capabilityStore, err := b.buildCapabilityStore(ctx, cfg.Capability)
	if err != nil {
		return nil, err
	}
	return &StoreBundle{Session: sessionStore, Audit: auditStore, AppAuth: appAuthStores, Capability: capabilityStore, Closers: append([]func(context.Context) error(nil), b.closers...)}, nil
}

func (b *storeBuilder) buildSessionStore(ctx context.Context, cfg ResolvedStoreConfig) (sessionauth.Store, error) {
	switch cfg.Driver {
	case StoreDriverMemory:
		return sessionauth.NewMemoryStore(), nil
	case StoreDriverSQLite, StoreDriverPostgres:
		db, err := b.openDB(cfg)
		if err != nil {
			return nil, fmt.Errorf("build session store: %w", err)
		}
		store, err := sessionauthsql.New(sessionauthsql.Config{DB: db, Dialect: sessionDialect(cfg.Driver)})
		if err != nil {
			return nil, fmt.Errorf("build session store: %w", err)
		}
		if cfg.ApplySchema {
			if err := store.ApplySchema(ctx); err != nil {
				return nil, err
			}
		}
		return store, nil
	default:
		return nil, fmt.Errorf("build session store: unsupported driver %q", cfg.Driver)
	}
}

func (b *storeBuilder) buildAuditStore(ctx context.Context, cfg ResolvedStoreConfig) (audit.Store, error) {
	switch cfg.Driver {
	case StoreDriverMemory:
		return &audit.MemoryStore{}, nil
	case StoreDriverSQLite, StoreDriverPostgres:
		db, err := b.openDB(cfg)
		if err != nil {
			return nil, fmt.Errorf("build audit store: %w", err)
		}
		store, err := auditsql.New(auditsql.Config{DB: db, Dialect: auditDialect(cfg.Driver)})
		if err != nil {
			return nil, fmt.Errorf("build audit store: %w", err)
		}
		if cfg.ApplySchema {
			if err := store.ApplySchema(ctx); err != nil {
				return nil, err
			}
		}
		return store, nil
	default:
		return nil, fmt.Errorf("build audit store: unsupported driver %q", cfg.Driver)
	}
}

func (b *storeBuilder) buildAppAuthStores(ctx context.Context, cfg ResolvedStoreConfig) (AppAuthStores, error) {
	switch cfg.Driver {
	case StoreDriverMemory:
		store := appauth.NewMemoryStore()
		return AppAuthStores{Users: store, Memberships: store, Resources: store}, nil
	case StoreDriverSQLite, StoreDriverPostgres:
		db, err := b.openDB(cfg)
		if err != nil {
			return AppAuthStores{}, fmt.Errorf("build appauth store: %w", err)
		}
		store, err := appauthsql.New(appauthsql.Config{DB: db, Dialect: appAuthDialect(cfg.Driver)})
		if err != nil {
			return AppAuthStores{}, fmt.Errorf("build appauth store: %w", err)
		}
		if cfg.ApplySchema {
			if err := store.ApplySchema(ctx); err != nil {
				return AppAuthStores{}, err
			}
		}
		return AppAuthStores{Users: store, Memberships: store, Resources: store}, nil
	default:
		return AppAuthStores{}, fmt.Errorf("build appauth store: unsupported driver %q", cfg.Driver)
	}
}

func (b *storeBuilder) buildCapabilityStore(ctx context.Context, cfg ResolvedStoreConfig) (capability.Store, error) {
	switch cfg.Driver {
	case StoreDriverMemory:
		return capability.NewMemoryStore(), nil
	case StoreDriverSQLite, StoreDriverPostgres:
		db, err := b.openDB(cfg)
		if err != nil {
			return nil, fmt.Errorf("build capability store: %w", err)
		}
		store, err := capabilitysql.New(capabilitysql.Config{DB: db, Dialect: capabilityDialect(cfg.Driver)})
		if err != nil {
			return nil, fmt.Errorf("build capability store: %w", err)
		}
		if cfg.ApplySchema {
			if err := store.ApplySchema(ctx); err != nil {
				return nil, err
			}
		}
		return store, nil
	default:
		return nil, fmt.Errorf("build capability store: unsupported driver %q", cfg.Driver)
	}
}

func (b *storeBuilder) openDB(cfg ResolvedStoreConfig) (*sql.DB, error) {
	key := sqlDBKey{driver: cfg.Driver, dsn: cfg.DSN}
	if db, ok := b.dbs[key]; ok {
		return db, nil
	}
	driverName, err := sqlDriverName(cfg.Driver)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open(driverName, cfg.DSN)
	if err != nil {
		return nil, err
	}
	b.dbs[key] = db
	b.closers = append(b.closers, func(context.Context) error { return db.Close() })
	return db, nil
}

func sqlDriverName(driver StoreDriver) (string, error) {
	switch driver {
	case StoreDriverSQLite:
		return "sqlite3", nil
	case StoreDriverPostgres:
		return "postgres", nil
	case StoreDriverMemory:
		return "", fmt.Errorf("memory store does not use a SQL driver")
	default:
		return "", fmt.Errorf("unsupported SQL driver %q", driver)
	}
}

func sessionDialect(driver StoreDriver) sessionauthsql.Dialect {
	if driver == StoreDriverSQLite {
		return sessionauthsql.DialectSQLite
	}
	return sessionauthsql.DialectPostgres
}

func auditDialect(driver StoreDriver) auditsql.Dialect {
	if driver == StoreDriverSQLite {
		return auditsql.DialectSQLite
	}
	return auditsql.DialectPostgres
}

func appAuthDialect(driver StoreDriver) appauthsql.Dialect {
	if driver == StoreDriverSQLite {
		return appauthsql.DialectSQLite
	}
	return appauthsql.DialectPostgres
}

func capabilityDialect(driver StoreDriver) capabilitysql.Dialect {
	if driver == StoreDriverSQLite {
		return capabilitysql.DialectSQLite
	}
	return capabilitysql.DialectPostgres
}

func closeAll(ctx context.Context, closers []func(context.Context) error) error {
	var errs []error
	for i := len(closers) - 1; i >= 0; i-- {
		if closers[i] == nil {
			continue
		}
		if err := closers[i](ctx); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
