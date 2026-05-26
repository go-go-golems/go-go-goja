package jsverbscli

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"

	noderequire "github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/engine"
	databasemod "github.com/go-go-golems/go-go-goja/modules/database"
	"github.com/go-go-golems/go-go-goja/modules/uidsl"
	"github.com/go-go-golems/go-go-goja/pkg/jsverbs"
	_ "github.com/mattn/go-sqlite3"
)

type RuntimeSettings struct {
	DBPath      string
	ReadOnly    bool
	AllowWrites bool
}

func runtimeInvokerFactory(settings *RuntimeSettings) InvokerFactory {
	return func(repo ScannedRepository, _ *jsverbs.VerbSpec) jsverbs.VerbInvoker {
		return func(ctx context.Context, registry *jsverbs.Registry, verb *jsverbs.VerbSpec, parsedValues *values.Values) (interface{}, error) {
			factory, cleanup, err := newRuntimeFactory(repo, settings)
			if err != nil {
				return nil, err
			}
			defer cleanup()

			rt, err := factory.NewRuntime(engine.WithStartupContext(ctx), engine.WithLifetimeContext(ctx))
			if err != nil {
				return nil, err
			}
			defer func() { _ = rt.Close(context.Background()) }()

			return registry.InvokeInRuntime(ctx, rt, verb, parsedValues)
		}
	}
}

func newRuntimeFactory(repo ScannedRepository, settings *RuntimeSettings) (*engine.Factory, func(), error) {
	if repo.Registry == nil {
		return nil, nil, fmt.Errorf("repository %s has no jsverbs registry", describeRepository(repo))
	}
	if settings == nil {
		settings = &RuntimeSettings{ReadOnly: true}
	}

	cleanup := func() {}
	moduleSpecs := []engine.RuntimeModuleSpec{}
	if settings.DBPath != "" {
		db, err := sql.Open("sqlite3", settings.DBPath)
		if err != nil {
			return nil, nil, fmt.Errorf("open sqlite database %s: %w", settings.DBPath, err)
		}
		if err := db.Ping(); err != nil {
			_ = db.Close()
			return nil, nil, fmt.Errorf("ping sqlite database %s: %w", settings.DBPath, err)
		}
		guarded := &guardedDB{db: db, allowWrites: settings.AllowWrites && !settings.ReadOnly}
		databaseModule := databasemod.New(
			databasemod.WithPreconfiguredDB(guarded),
			databasemod.WithConfigureEnabled(false),
		)
		dbAliasModule := databasemod.New(
			databasemod.WithName("db"),
			databasemod.WithPreconfiguredDB(guarded),
			databasemod.WithConfigureEnabled(false),
		)
		moduleSpecs = append(moduleSpecs,
			engine.NativeModuleSpec{ModuleID: "database:configured", ModuleName: databaseModule.Name(), Loader: databaseModule.Loader},
			engine.NativeModuleSpec{ModuleID: "database:db-alias", ModuleName: dbAliasModule.Name(), Loader: dbAliasModule.Loader},
		)
		cleanup = func() { _ = db.Close() }
	}

	builder := engine.NewBuilder(runtimeOptions(repo)...).
		WithRequireOptions(noderequire.WithLoader(repo.Registry.RequireLoader())).
		UseModuleMiddleware(engine.MiddlewareOnly("fs", "path", "time", "timer", "yaml")).
		WithModules(moduleSpecs...).
		WithModules(uidsl.NewRegistrar())
	factory, err := builder.Build()
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	return factory, cleanup, nil
}

func runtimeOptions(repo ScannedRepository) []engine.Option {
	if repo.Repository.Embedded {
		return nil
	}
	folders := []string{repo.Repository.RootDir, filepath.Join(repo.Repository.RootDir, "node_modules")}
	parent := filepath.Dir(repo.Repository.RootDir)
	if parent != repo.Repository.RootDir {
		folders = append(folders, parent, filepath.Join(parent, "node_modules"))
	}
	return []engine.Option{engine.WithRequireOptions(noderequire.WithGlobalFolders(folders...))}
}

type guardedDB struct {
	db          *sql.DB
	allowWrites bool
}

func (g *guardedDB) Query(query string, args ...any) (*sql.Rows, error) {
	return g.db.Query(query, args...)
}

func (g *guardedDB) Exec(query string, args ...any) (sql.Result, error) {
	if !g.allowWrites {
		return nil, fmt.Errorf("database writes are disabled; rerun with --readonly=false --allow-writes")
	}
	return g.db.Exec(query, args...)
}
