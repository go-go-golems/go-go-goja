package databasemod_test

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/dop251/goja"
	databasemod "github.com/go-go-golems/go-go-goja/modules/database"
	gggengine "github.com/go-go-golems/go-go-goja/pkg/engine"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestDefaultDatabaseModuleConfigure(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "default.db")

	factory, err := gggengine.NewRuntimeFactoryBuilder().
		UseModuleMiddleware(gggengine.MiddlewareOnly("database")).
		Build()
	require.NoError(t, err)

	rt, err := factory.NewRuntime(gggengine.WithStartupContext(context.Background()), gggengine.WithLifetimeContext(context.Background()))
	require.NoError(t, err)
	defer func() { require.NoError(t, rt.Close(context.Background())) }()

	ret, err := rt.Owner.Call(context.Background(), "database.default", func(_ context.Context, vm *goja.Runtime) (any, error) {
		value, err := rt.VM.RunString(`
			const db = require("database");
			db.configure("sqlite3", ` + "`" + dbPath + "`" + `);
			db.exec("CREATE TABLE IF NOT EXISTS users (name TEXT)");
			db.exec("INSERT INTO users(name) VALUES (?)", "Ada");
			JSON.stringify(db.query("SELECT name FROM users ORDER BY name"));
		`)
		if err != nil {
			return nil, err
		}
		return value.Export(), nil
	})
	require.NoError(t, err)
	require.Equal(t, `[{"name":"Ada"}]`, ret)
}

func TestPreconfiguredModuleRequireByName(t *testing.T) {
	db := openSQLiteDB(t)
	_, err := db.Exec(`CREATE TABLE widgets (name TEXT NOT NULL)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO widgets(name) VALUES (?)`, "from-site-db")
	require.NoError(t, err)

	module := databasemod.New(
		databasemod.WithName("site-db"),
		databasemod.WithPreconfiguredDB(db),
	)

	factory, err := gggengine.NewRuntimeFactoryBuilder().
		WithModules(gggengine.NativeModuleRegistrar{
			ModuleID:   "test-site-db",
			ModuleName: module.Name(),
			Loader:     module.Loader,
		}).
		Build()
	require.NoError(t, err)

	rt, err := factory.NewRuntime(gggengine.WithStartupContext(context.Background()), gggengine.WithLifetimeContext(context.Background()))
	require.NoError(t, err)
	defer func() { require.NoError(t, rt.Close(context.Background())) }()

	ret, err := rt.Owner.Call(context.Background(), "database.preconfigured", func(_ context.Context, vm *goja.Runtime) (any, error) {
		value, err := rt.VM.RunString(`
			const siteDB = require("site-db");
			JSON.stringify(siteDB.query("SELECT name FROM widgets ORDER BY name"));
		`)
		if err != nil {
			return nil, err
		}
		return value.Export(), nil
	})
	require.NoError(t, err)
	require.Equal(t, `[{"name":"from-site-db"}]`, ret)
}

func TestPreconfiguredModuleRejectsConfigure(t *testing.T) {
	db := openSQLiteDB(t)
	module := databasemod.New(
		databasemod.WithName("site-db"),
		databasemod.WithPreconfiguredDB(db),
	)

	err := module.Configure("sqlite3", ":memory:")
	require.Error(t, err)
	require.Contains(t, err.Error(), "preconfigured")
}

func TestPreconfiguredModuleExecReceivesOwnerCallContext(t *testing.T) {
	type contextKey string
	const key contextKey = "request-id"

	db := &contextRecordingDB{}
	module := databasemod.New(
		databasemod.WithName("site-db"),
		databasemod.WithPreconfiguredDB(db),
	)

	factory, err := gggengine.NewRuntimeFactoryBuilder().
		WithModules(gggengine.NativeModuleRegistrar{
			ModuleID:   "test-site-db-context",
			ModuleName: module.Name(),
			Loader:     module.Loader,
		}).
		Build()
	require.NoError(t, err)

	rt, err := factory.NewRuntime(gggengine.WithStartupContext(context.Background()), gggengine.WithLifetimeContext(context.Background()))
	require.NoError(t, err)
	defer func() { require.NoError(t, rt.Close(context.Background())) }()

	ctx := context.WithValue(context.Background(), key, "from-request")
	_, err = rt.Owner.Call(ctx, "database.context", func(_ context.Context, vm *goja.Runtime) (any, error) {
		value, err := rt.VM.RunString(`
			const siteDB = require("site-db");
			siteDB.exec("INSERT INTO widgets(name) VALUES (?)", "Ada").success;
		`)
		if err != nil {
			return nil, err
		}
		return value.Export(), nil
	})
	require.NoError(t, err)
	require.Equal(t, "from-request", db.got.Value(key))
}

func TestPreconfiguredModuleExecAfterAwaitReceivesOriginalCallContext(t *testing.T) {
	type contextKey string
	const key contextKey = "request-id"

	db := &contextRecordingDB{}
	module := databasemod.New(
		databasemod.WithName("site-db"),
		databasemod.WithPreconfiguredDB(db),
	)

	factory, err := gggengine.NewRuntimeFactoryBuilder().
		WithModules(gggengine.NativeModuleRegistrar{
			ModuleID:   "test-site-db-async-context",
			ModuleName: module.Name(),
			Loader:     module.Loader,
		}).
		Build()
	require.NoError(t, err)

	rt, err := factory.NewRuntime(gggengine.WithStartupContext(context.Background()), gggengine.WithLifetimeContext(context.Background()))
	require.NoError(t, err)
	defer func() { require.NoError(t, rt.Close(context.Background())) }()

	ctx := context.WithValue(context.Background(), key, "from-request")
	ret, err := rt.Owner.Call(ctx, "database.context.async-start", func(_ context.Context, vm *goja.Runtime) (any, error) {
		value, err := rt.VM.RunString(`
			(async () => {
				const timer = require("timer");
				const siteDB = require("site-db");
				await timer.sleep(1);
				return siteDB.exec("INSERT INTO widgets(name) VALUES (?)", "Ada").success;
			})();
		`)
		if err != nil {
			return nil, err
		}
		return value.Export(), nil
	})
	require.NoError(t, err)
	promise, ok := ret.(*goja.Promise)
	require.True(t, ok, "async IIFE should return a Promise")

	var result any
	deadline := time.After(time.Second)
	for result == nil {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for async database exec promise")
		default:
		}

		result, err = rt.Owner.Call(context.Background(), "database.context.async-poll", func(_ context.Context, _ *goja.Runtime) (any, error) {
			switch promise.State() {
			case goja.PromiseStatePending:
				return nil, nil
			case goja.PromiseStateRejected:
				return nil, fmt.Errorf("promise rejected: %s", promise.Result().String())
			case goja.PromiseStateFulfilled:
				return promise.Result().Export(), nil
			default:
				return nil, fmt.Errorf("unknown promise state: %v", promise.State())
			}
		})
		require.NoError(t, err)
		if result == nil {
			time.Sleep(5 * time.Millisecond)
		}
	}

	require.Equal(t, true, result)
	require.Equal(t, "from-request", db.got.Value(key))
}

func TestDBModuleExecContextFallsBackToLegacyQueryExecer(t *testing.T) {
	db := &legacyRecordingDB{}
	module := databasemod.New(databasemod.WithPreconfiguredDB(db))

	ret, err := module.ExecContext(context.Background(), "CREATE TABLE widgets(name TEXT)")
	require.NoError(t, err)
	require.Equal(t, true, ret["success"])
	require.Equal(t, int64(1), db.execCalls)
}

func TestDatabaseTransactionCommitPersistsWrites(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "tx-commit.db")

	factory, err := gggengine.NewRuntimeFactoryBuilder().
		UseModuleMiddleware(gggengine.MiddlewareOnly("database")).
		Build()
	require.NoError(t, err)

	rt, err := factory.NewRuntime(gggengine.WithStartupContext(context.Background()), gggengine.WithLifetimeContext(context.Background()))
	require.NoError(t, err)
	defer func() { require.NoError(t, rt.Close(context.Background())) }()

	ret, err := rt.Owner.Call(context.Background(), "database.tx.commit", func(_ context.Context, vm *goja.Runtime) (any, error) {
		value, err := rt.VM.RunString(`
			const db = require("database");
			db.configure("sqlite3", ` + "`" + dbPath + "`" + `);
			db.exec("CREATE TABLE users (name TEXT NOT NULL)");
			const tx = db.begin();
			tx.exec("INSERT INTO users(name) VALUES (?)", "Ada");
			tx.exec("INSERT INTO users(name) VALUES (?)", "Grace");
			const commit = tx.commit();
			JSON.stringify({ commit, rows: db.query("SELECT name FROM users ORDER BY name") });
		`)
		if err != nil {
			return nil, err
		}
		return value.Export(), nil
	})
	require.NoError(t, err)
	require.Equal(t, `{"commit":{"success":true},"rows":[{"name":"Ada"},{"name":"Grace"}]}`, ret)
}

func TestDatabaseTransactionRollbackDiscardsWrites(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "tx-rollback.db")

	factory, err := gggengine.NewRuntimeFactoryBuilder().
		UseModuleMiddleware(gggengine.MiddlewareOnly("database")).
		Build()
	require.NoError(t, err)

	rt, err := factory.NewRuntime(gggengine.WithStartupContext(context.Background()), gggengine.WithLifetimeContext(context.Background()))
	require.NoError(t, err)
	defer func() { require.NoError(t, rt.Close(context.Background())) }()

	ret, err := rt.Owner.Call(context.Background(), "database.tx.rollback", func(_ context.Context, vm *goja.Runtime) (any, error) {
		value, err := rt.VM.RunString(`
			const db = require("database");
			db.configure("sqlite3", ` + "`" + dbPath + "`" + `);
			db.exec("CREATE TABLE users (name TEXT NOT NULL)");
			const tx = db.begin();
			tx.exec("INSERT INTO users(name) VALUES (?)", "Ada");
			const rollback = tx.rollback();
			JSON.stringify({ rollback, rows: db.query("SELECT name FROM users ORDER BY name") });
		`)
		if err != nil {
			return nil, err
		}
		return value.Export(), nil
	})
	require.NoError(t, err)
	require.Equal(t, `{"rollback":{"success":true},"rows":[]}`, ret)
}

func TestDatabaseTransactionRejectsUseAfterCommit(t *testing.T) {
	db := openSQLiteDB(t)
	_, err := db.Exec(`CREATE TABLE users (name TEXT NOT NULL)`)
	require.NoError(t, err)

	module := databasemod.New(databasemod.WithPreconfiguredDB(db))
	tx, err := module.BeginContext(context.Background())
	require.NoError(t, err)
	_, err = tx.Commit()
	require.NoError(t, err)

	_, err = tx.ExecContext(context.Background(), `INSERT INTO users(name) VALUES (?)`, "Ada")
	require.Error(t, err)
	require.Contains(t, err.Error(), "transaction is closed")

	_, err = tx.Rollback()
	require.Error(t, err)
	require.Contains(t, err.Error(), "transaction is closed")
}

func TestDatabaseTransactionBeginRequiresConfiguredTransactionalDB(t *testing.T) {
	module := databasemod.New()
	_, err := module.BeginContext(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "database not configured")

	module = databasemod.New(databasemod.WithPreconfiguredDB(&legacyRecordingDB{}))
	_, err = module.BeginContext(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not support transactions")
}

func TestDatabaseTransactionReceivesOwnerCallContextAfterAwait(t *testing.T) {
	type contextKey string
	const key contextKey = "request-id"

	db := &transactionContextRecordingDB{}
	module := databasemod.New(
		databasemod.WithName("site-db"),
		databasemod.WithPreconfiguredDB(db),
	)

	factory, err := gggengine.NewRuntimeFactoryBuilder().
		WithModules(gggengine.NativeModuleRegistrar{
			ModuleID:   "test-site-db-transaction-context",
			ModuleName: module.Name(),
			Loader:     module.Loader,
		}).
		Build()
	require.NoError(t, err)

	rt, err := factory.NewRuntime(gggengine.WithStartupContext(context.Background()), gggengine.WithLifetimeContext(context.Background()))
	require.NoError(t, err)
	defer func() { require.NoError(t, rt.Close(context.Background())) }()

	ctx := context.WithValue(context.Background(), key, "from-request")
	ret, err := rt.Owner.Call(ctx, "database.tx.context.async-start", func(_ context.Context, vm *goja.Runtime) (any, error) {
		value, err := rt.VM.RunString(`
			(async () => {
				const timer = require("timer");
				const siteDB = require("site-db");
				await timer.sleep(1);
				const tx = siteDB.begin();
				const ok = tx.exec("INSERT INTO widgets(name) VALUES (?)", "Ada").success;
				tx.rollback();
				return ok;
			})();
		`)
		if err != nil {
			return nil, err
		}
		return value.Export(), nil
	})
	require.NoError(t, err)
	promise, ok := ret.(*goja.Promise)
	require.True(t, ok, "async IIFE should return a Promise")

	var result any
	deadline := time.After(time.Second)
	for result == nil {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for async database transaction promise")
		default:
		}

		result, err = rt.Owner.Call(context.Background(), "database.tx.context.async-poll", func(_ context.Context, _ *goja.Runtime) (any, error) {
			switch promise.State() {
			case goja.PromiseStatePending:
				return nil, nil
			case goja.PromiseStateRejected:
				return nil, fmt.Errorf("promise rejected: %s", promise.Result().String())
			case goja.PromiseStateFulfilled:
				return promise.Result().Export(), nil
			default:
				return nil, fmt.Errorf("unknown promise state: %v", promise.State())
			}
		})
		require.NoError(t, err)
		if result == nil {
			time.Sleep(5 * time.Millisecond)
		}
	}

	require.Equal(t, true, result)
	require.Equal(t, "from-request", db.beginCtx.Value(key))
	require.Equal(t, "from-request", db.tx.execCtx.Value(key))
}

type transactionContextRecordingDB struct {
	beginCtx context.Context
	tx       *contextRecordingTx
}

func (db *transactionContextRecordingDB) Query(string, ...any) (*sql.Rows, error) {
	return nil, fmt.Errorf("unexpected legacy Query call")
}

func (db *transactionContextRecordingDB) Exec(string, ...any) (sql.Result, error) {
	return nil, fmt.Errorf("unexpected legacy Exec call")
}

func (db *transactionContextRecordingDB) BeginTransactionContext(ctx context.Context, _ *sql.TxOptions) (databasemod.Transaction, error) {
	db.beginCtx = ctx
	db.tx = &contextRecordingTx{}
	return db.tx, nil
}

type contextRecordingTx struct {
	execCtx context.Context
}

func (tx *contextRecordingTx) Query(string, ...any) (*sql.Rows, error) {
	return nil, fmt.Errorf("unexpected transaction Query call")
}

func (tx *contextRecordingTx) Exec(string, ...any) (sql.Result, error) {
	return nil, fmt.Errorf("unexpected transaction Exec call")
}

func (tx *contextRecordingTx) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	return nil, fmt.Errorf("unexpected transaction QueryContext call")
}

func (tx *contextRecordingTx) ExecContext(ctx context.Context, _ string, _ ...any) (sql.Result, error) {
	tx.execCtx = ctx
	return fakeResult{rowsAffected: 1}, nil
}

func (tx *contextRecordingTx) Commit() error   { return nil }
func (tx *contextRecordingTx) Rollback() error { return nil }

type contextRecordingDB struct {
	got context.Context
}

func (db *contextRecordingDB) Query(string, ...any) (*sql.Rows, error) {
	return nil, fmt.Errorf("unexpected legacy Query call")
}

func (db *contextRecordingDB) Exec(string, ...any) (sql.Result, error) {
	return nil, fmt.Errorf("unexpected legacy Exec call")
}

func (db *contextRecordingDB) QueryContext(ctx context.Context, _ string, _ ...any) (*sql.Rows, error) {
	db.got = ctx
	return nil, fmt.Errorf("unexpected QueryContext call")
}

func (db *contextRecordingDB) ExecContext(ctx context.Context, _ string, _ ...any) (sql.Result, error) {
	db.got = ctx
	return fakeResult{rowsAffected: 1}, nil
}

type legacyRecordingDB struct {
	execCalls int64
}

func (db *legacyRecordingDB) Query(string, ...any) (*sql.Rows, error) {
	return nil, fmt.Errorf("unexpected Query call")
}

func (db *legacyRecordingDB) Exec(string, ...any) (sql.Result, error) {
	db.execCalls++
	return fakeResult{rowsAffected: 1}, nil
}

type fakeResult struct {
	lastInsertID int64
	rowsAffected int64
}

func (r fakeResult) LastInsertId() (int64, error) { return r.lastInsertID, nil }
func (r fakeResult) RowsAffected() (int64, error) { return r.rowsAffected, nil }

func openSQLiteDB(t *testing.T) *sql.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "site.db")
	db, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	return db
}
