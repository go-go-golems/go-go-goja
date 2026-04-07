package databasemod_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/dop251/goja"
	gggengine "github.com/go-go-golems/go-go-goja/engine"
	databasemod "github.com/go-go-golems/go-go-goja/modules/database"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestDefaultDatabaseModuleConfigure(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "default.db")

	factory, err := gggengine.NewBuilder().
		WithModules(gggengine.DefaultRegistryModules()).
		Build()
	require.NoError(t, err)

	rt, err := factory.NewRuntime(context.Background())
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

	rt, err := factory.NewRuntime(context.Background())
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

func openSQLiteDB(t *testing.T) *sql.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "site.db")
	db, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	return db
}
