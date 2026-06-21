package host

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/app"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

func TestRegisterHostProvider(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	if err := Register(registry); err != nil {
		t.Fatalf("register host provider: %v", err)
	}
	for _, name := range []string{"fs", "node:fs", "fetch", "exec", "database", "db"} {
		mod, ok := registry.ResolveModule(PackageID, name)
		if !ok {
			t.Fatalf("expected host module %q", name)
		}
		if mod.TypeScript == nil {
			t.Fatalf("expected host module %q to carry TypeScript descriptor", name)
		}
	}
}

func TestFetchRequiresExplicitAllow(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	if err := Register(registry); err != nil {
		t.Fatalf("register host provider: %v", err)
	}
	mod, ok := registry.ResolveModule(PackageID, "fetch")
	if !ok {
		t.Fatal("expected fetch module")
	}
	_, err := mod.NewModuleFactory(providerapi.ModuleSetupContext{Context: context.Background(), Name: "fetch", As: "fetch"})
	if err == nil || !strings.Contains(err.Error(), "config.allow=true") {
		t.Fatalf("expected allow error, got %v", err)
	}
}

func TestFetchProviderHonorsAllowedOrigins(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	if err := Register(registry); err != nil {
		t.Fatalf("register host provider: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()
	runtimePlan := &app.RuntimePlan{Runtime: app.RuntimeSection{Modules: []app.RuntimeModulePlan{{
		Provider: PackageID,
		Name:     "fetch",
		As:       "fetch",
		Config: map[string]any{
			"allow":          true,
			"allowedOrigins": []any{server.URL},
			"timeout":        "2s",
		},
	}}}}
	host := app.NewHostWithOptions(registry, runtimePlan, app.HostOptions{})
	rt, err := host.Factory.NewRuntime(context.Background())
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	defer func() { _ = rt.Close(context.Background()) }()

	_, err = rt.Owner.Call(context.Background(), "host.fetch.setup", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, runErr := vm.RunString(`
			globalThis.__fetchProviderSmoke = { done: false };
			(async () => {
				const fetch = require("fetch");
				const allowed = await fetch.fetch(` + strconv.Quote(server.URL) + `);
				let denied = "";
				try { await fetch.fetch("http://example.invalid/"); }
				catch (e) { denied = String(e); }
				globalThis.__fetchProviderSmoke = { done: true, error: "", allowed: allowed.status, denied };
			})().catch(e => { globalThis.__fetchProviderSmoke = { done: true, error: String(e) }; });
		`)
		return nil, runErr
	})
	if err != nil {
		t.Fatalf("run fetch setup: %v", err)
	}
	state := ""
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		ret, err := rt.Owner.Call(context.Background(), "host.fetch.state", func(_ context.Context, vm *goja.Runtime) (any, error) {
			value, runErr := vm.RunString(`JSON.stringify(globalThis.__fetchProviderSmoke || { done: false })`)
			if runErr != nil {
				return nil, runErr
			}
			return value.String(), nil
		})
		if err != nil {
			t.Fatalf("read fetch state: %v", err)
		}
		state = ret.(string)
		if strings.Contains(state, `"done":true`) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !strings.Contains(state, `"error":""`) || !strings.Contains(state, `"allowed":200`) || !strings.Contains(state, "not allowed") {
		t.Fatalf("fetch provider state = %s", state)
	}
}

func TestFSRequiresExplicitAllow(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	if err := Register(registry); err != nil {
		t.Fatalf("register host provider: %v", err)
	}
	mod, ok := registry.ResolveModule(PackageID, "fs")
	if !ok {
		t.Fatal("expected fs module")
	}
	_, err := mod.NewModuleFactory(providerapi.ModuleSetupContext{Context: context.Background(), Name: "fs", As: "fs"})
	if err == nil || !strings.Contains(err.Error(), "config.allow=true") {
		t.Fatalf("expected allow error, got %v", err)
	}
}

func TestFSHostAndEmbeddedAliases(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	if err := Register(registry); err != nil {
		t.Fatalf("register host provider: %v", err)
	}
	dir := t.TempDir()
	outPath := filepath.ToSlash(filepath.Join(dir, "out.txt"))
	assetFS := fstest.MapFS{
		"xgoja_embed/assets/app/config/default.json": &fstest.MapFile{Data: []byte(`{"ok":true}`)},
	}
	runtimePlan := &app.RuntimePlan{
		Sources: []app.SourcePlan{{ID: "app-assets", Kind: app.SourceKindAssets, Path: "xgoja_embed/assets/app", Embed: true}},
		Runtime: app.RuntimeSection{Modules: []app.RuntimeModulePlan{
			{
				Provider: PackageID,
				Name:     "fs",
				As:       "fs:assets",
				Config: map[string]any{
					"embedded": map[string]any{
						"allow":  true,
						"mounts": []any{map[string]any{"asset": "app-assets", "mount": "/app"}},
					},
				},
			},
			{
				Provider: PackageID,
				Name:     "fs",
				As:       "fs:host",
				Config:   map[string]any{"allow": true},
			},
		}},
	}
	host := app.NewHostWithOptions(registry, runtimePlan, app.HostOptions{EmbeddedAssets: assetFS})
	rt, err := host.Factory.NewRuntime(context.Background())
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	defer func() { _ = rt.Close(context.Background()) }()

	ret, err := rt.Owner.Call(context.Background(), "host.fs-aliases", func(_ context.Context, vm *goja.Runtime) (any, error) {
		value, runErr := vm.RunString(`
			const assets = require("fs:assets");
			const host = require("fs:host");
			let plain = "";
			try { require("fs"); } catch (e) { plain = "missing"; }
			const text = assets.readFileSync("/app/config/default.json", "utf8");
			host.writeFileSync(` + strconv.Quote(outPath) + `, text, "utf8");
			JSON.stringify({ text, plain, wrote: host.existsSync(` + strconv.Quote(outPath) + `) });
		`)
		if runErr != nil {
			return nil, runErr
		}
		return value.String(), nil
	})
	if err != nil {
		t.Fatalf("run aliases: %v", err)
	}
	state := ret.(string)
	for _, want := range []string{`"text":"{\"ok\":true}"`, `"plain":"missing"`, `"wrote":true`} {
		if !strings.Contains(state, want) {
			t.Fatalf("alias state missing %s: %s", want, state)
		}
	}
}

func TestFSRootEmbeddedMount(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	if err := Register(registry); err != nil {
		t.Fatalf("register host provider: %v", err)
	}
	assetFS := fstest.MapFS{
		"xgoja_embed/assets/app/config/default.json": &fstest.MapFile{Data: []byte(`{"ok":true}`)},
	}
	runtimePlan := &app.RuntimePlan{
		Sources: []app.SourcePlan{{ID: "app-assets", Kind: app.SourceKindAssets, Path: "xgoja_embed/assets/app", Embed: true}},
		Runtime: app.RuntimeSection{Modules: []app.RuntimeModulePlan{{
			Provider: PackageID,
			Name:     "fs",
			As:       "fs:assets",
			Config: map[string]any{
				"embedded": map[string]any{
					"allow":  true,
					"mounts": []any{map[string]any{"asset": "app-assets", "mount": "/"}},
				},
			},
		}}},
	}
	host := app.NewHostWithOptions(registry, runtimePlan, app.HostOptions{EmbeddedAssets: assetFS})
	rt, err := host.Factory.NewRuntime(context.Background())
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	defer func() { _ = rt.Close(context.Background()) }()

	ret, err := rt.Owner.Call(context.Background(), "host.fs-root-mount", func(_ context.Context, vm *goja.Runtime) (any, error) {
		value, runErr := vm.RunString(`require("fs:assets").readFileSync("/config/default.json", "utf8")`)
		if runErr != nil {
			return nil, runErr
		}
		return value.String(), nil
	})
	if err != nil {
		t.Fatalf("read root-mounted asset: %v", err)
	}
	if ret.(string) != `{"ok":true}` {
		t.Fatalf("root-mounted asset = %q", ret)
	}
}

func TestFSRejectsCombinedHostAndEmbeddedConfig(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	if err := Register(registry); err != nil {
		t.Fatalf("register host provider: %v", err)
	}
	mod, ok := registry.ResolveModule(PackageID, "fs")
	if !ok {
		t.Fatal("expected fs module")
	}
	cfg, err := json.Marshal(FSConfig{Allow: true, Embedded: EmbeddedFSConfig{Allow: true, Mounts: []AssetMount{{Asset: "app", Mount: "/app"}}}})
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	_, err = mod.NewModuleFactory(providerapi.ModuleSetupContext{Context: context.Background(), Name: "fs", As: "fs", Config: cfg})
	if err == nil || !strings.Contains(err.Error(), "separate aliases") {
		t.Fatalf("expected separate aliases error, got %v", err)
	}
}

func TestDatabasePreconfiguredFromProviderConfig(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	if err := Register(registry); err != nil {
		t.Fatalf("register host provider: %v", err)
	}
	dbPath := filepath.ToSlash(filepath.Join(t.TempDir(), "site.db"))
	runtimePlan := &app.RuntimePlan{
		Runtime: app.RuntimeSection{Modules: []app.RuntimeModulePlan{{
			Provider: PackageID,
			Name:     "db",
			As:       "db",
			Config: map[string]any{
				"driverName":     "sqlite3",
				"dataSourceName": dbPath,
			},
		}}},
	}
	host := app.NewHostWithOptions(registry, runtimePlan, app.HostOptions{})
	rt, err := host.Factory.NewRuntime(context.Background())
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	defer func() { _ = rt.Close(context.Background()) }()

	ret, err := rt.Owner.Call(context.Background(), "host.db-preconfigured", func(_ context.Context, vm *goja.Runtime) (any, error) {
		value, runErr := vm.RunString(`
			const db = require("db");
			db.exec("CREATE TABLE widgets (name TEXT NOT NULL)");
			db.exec("INSERT INTO widgets(name) VALUES (?)", "from-provider-config");
			let configureError = "";
			try { db.configure("sqlite3", ":memory:"); } catch (e) { configureError = String(e); }
			JSON.stringify({
				rows: db.query("SELECT name FROM widgets ORDER BY name"),
				configureError,
			});
		`)
		if runErr != nil {
			return nil, runErr
		}
		return value.String(), nil
	})
	if err != nil {
		t.Fatalf("run preconfigured db script: %v", err)
	}
	state := ret.(string)
	for _, want := range []string{`"name":"from-provider-config"`, `"configureError":"GoError: database module`, `is preconfigured and does not allow configure()`} {
		if !strings.Contains(state, want) {
			t.Fatalf("preconfigured db state missing %s: %s", want, state)
		}
	}
}

func TestDatabasePreconfiguredRejectsPartialConfig(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	if err := Register(registry); err != nil {
		t.Fatalf("register host provider: %v", err)
	}
	mod, ok := registry.ResolveModule(PackageID, "db")
	if !ok {
		t.Fatal("expected db module")
	}
	cfg, err := json.Marshal(DatabaseConfig{DriverName: "sqlite3"})
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	_, err = mod.NewModuleFactory(providerapi.ModuleSetupContext{Context: context.Background(), Name: "db", As: "db", Config: cfg})
	if err == nil || !strings.Contains(err.Error(), "requires both driverName and dataSourceName") {
		t.Fatalf("expected partial preconfig error, got %v", err)
	}
}

func TestDatabaseAllowConfigureModeStillWorks(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	if err := Register(registry); err != nil {
		t.Fatalf("register host provider: %v", err)
	}
	dbPath := filepath.ToSlash(filepath.Join(t.TempDir(), "configured-by-js.db"))
	runtimePlan := &app.RuntimePlan{
		Runtime: app.RuntimeSection{Modules: []app.RuntimeModulePlan{{
			Provider: PackageID,
			Name:     "db",
			As:       "db",
			Config:   map[string]any{"allowConfigure": true},
		}}},
	}
	host := app.NewHostWithOptions(registry, runtimePlan, app.HostOptions{})
	rt, err := host.Factory.NewRuntime(context.Background())
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	defer func() { _ = rt.Close(context.Background()) }()

	ret, err := rt.Owner.Call(context.Background(), "host.db-allow-configure", func(_ context.Context, vm *goja.Runtime) (any, error) {
		value, runErr := vm.RunString(`
			const db = require("db");
			db.configure("sqlite3", ` + strconv.Quote(dbPath) + `);
			db.exec("CREATE TABLE widgets (name TEXT NOT NULL)");
			db.exec("INSERT INTO widgets(name) VALUES (?)", "from-js-configure");
			JSON.stringify(db.query("SELECT name FROM widgets ORDER BY name"));
		`)
		if runErr != nil {
			return nil, runErr
		}
		return value.String(), nil
	})
	if err != nil {
		t.Fatalf("run allow-configure db script: %v", err)
	}
	if ret.(string) != `[{"name":"from-js-configure"}]` {
		t.Fatalf("allow-configure result = %s", ret)
	}
}

func TestExecAllowedCommands(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	if err := Register(registry); err != nil {
		t.Fatalf("register host provider: %v", err)
	}
	mod, ok := registry.ResolveModule(PackageID, "exec")
	if !ok {
		t.Fatal("expected exec module")
	}
	cfg, err := json.Marshal(ExecConfig{Allow: true, AllowedCommands: []string{"echo"}})
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	loader, err := mod.NewModuleFactory(providerapi.ModuleSetupContext{Context: context.Background(), Name: "exec", As: "exec", Config: cfg})
	if err != nil {
		t.Fatalf("new exec loader: %v", err)
	}
	vm := goja.New()
	moduleObj := vm.NewObject()
	exports := vm.NewObject()
	if err := moduleObj.Set("exports", exports); err != nil {
		t.Fatalf("set exports: %v", err)
	}
	loader(vm, moduleObj)
	run, ok := goja.AssertFunction(exports.Get("run"))
	if !ok {
		t.Fatal("exec.run is not a function")
	}
	if _, err := run(goja.Undefined(), vm.ToValue("sh"), vm.ToValue([]string{"-c", "echo bad"})); err == nil {
		t.Fatal("expected disallowed command error")
	}
}
