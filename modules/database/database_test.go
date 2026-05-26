package databasemod_test

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/dop251/goja"
	gggengine "github.com/go-go-golems/go-go-goja/engine"
	databasemod "github.com/go-go-golems/go-go-goja/modules/database"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestDefaultDatabaseModuleConfigure(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "default.db")

	factory, err := gggengine.NewBuilder().
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

	factory, err := gggengine.NewBuilder().
		WithModules(gggengine.NativeModuleSpec{
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

	factory, err := gggengine.NewBuilder().
		WithModules(gggengine.NativeModuleSpec{
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

	factory, err := gggengine.NewBuilder().
		WithModules(gggengine.NativeModuleSpec{
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
