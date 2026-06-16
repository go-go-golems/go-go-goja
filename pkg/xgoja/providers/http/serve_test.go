package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	stdhttp "net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
	"github.com/go-go-golems/go-go-goja/pkg/jsverbs"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/app"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/hostauth"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/hotreload"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

func TestNewServeCommandSetRequiresJSVerbSources(t *testing.T) {
	_, ok := providerapi.NewProviderRegistry().ResolveCommandSetProvider(PackageID, "serve")
	if ok {
		t.Fatal("empty registry unexpectedly resolved serve provider")
	}
	_, err := newServeCommandSet(providerapi.CommandSetContext{RuntimeFactory: fakeRuntimeFactory{}})
	if err == nil {
		t.Fatal("expected missing jsverb source error")
	}
}

func TestNewServeCommandSetBuildsVerbCommandsWithHTTPSection(t *testing.T) {
	registry := scanServeTestRegistry(t)
	capability := newHTTPCapability()
	set, err := newServeCommandSet(providerapi.CommandSetContext{
		Name:           "serve",
		RuntimeFactory: fakeRuntimeFactory{},
		SelectedModules: []providerapi.ModuleDescriptor{{
			PackageID:           PackageID,
			ModuleID:            "express",
			As:                  "express",
			PackageCapabilities: []providerapi.PackageCapability{capability},
		}},
		Sources: fakeSourceRegistry{jsverbs: fakeJSVerbSourceSet{registries: []*jsverbs.Registry{registry}}},
	})
	if err != nil {
		t.Fatalf("new serve command set: %v", err)
	}
	if len(set.Commands) != 1 {
		t.Fatalf("commands = %d, want 1", len(set.Commands))
	}
	desc := set.Commands[0].Description()
	if desc.Name != "demo" {
		t.Fatalf("command name = %q, want demo", desc.Name)
	}
	if desc.Parents[0] != "sites" {
		t.Fatalf("parents = %#v, want sites", desc.Parents)
	}
	if _, ok := desc.Schema.Get("http"); !ok {
		t.Fatalf("expected http section on serve command; schema=%#v", desc.Schema)
	}
	hotReloadSection, ok := desc.Schema.Get(serveHotReloadSectionSlug)
	if !ok {
		t.Fatalf("expected hot reload section on serve command; schema=%#v", desc.Schema)
	}
	for _, name := range []string{"hot-reload", "hot-reload-watch-root", "hot-reload-watch-ext", "hot-reload-smoke-path", "hot-reload-poll", "hot-reload-debounce", "hot-reload-close-grace", "hot-reload-status-path"} {
		if _, ok := hotReloadSection.GetDefinitions().Get(name); !ok {
			t.Fatalf("missing hot reload field %q", name)
		}
	}
}

func TestNewServeCommandSetAddsAuthSectionWhenHostAuthFactoryPresent(t *testing.T) {
	registry := scanServeTestRegistry(t)
	hostServices := app.HostServices{}
	if err := hostServices.SetHostService(hostauth.ServiceFactoryKey, hostauth.NewServiceFactory(hostauth.BuilderOptions{Config: hostauth.Config{
		Mode: hostauth.ModeDev,
		Stores: hostauth.StoresConfig{Default: hostauth.StoreConfig{
			Driver: string(hostauth.StoreDriverSQLite),
			DSN:    "file:auth.db",
		}},
	}})); err != nil {
		t.Fatalf("SetHostService: %v", err)
	}
	set, err := newServeCommandSet(providerapi.CommandSetContext{
		Name:           "serve",
		Host:           hostServices,
		RuntimeFactory: fakeRuntimeFactory{},
		Sources:        fakeSourceRegistry{jsverbs: fakeJSVerbSourceSet{registries: []*jsverbs.Registry{registry}}},
	})
	if err != nil {
		t.Fatalf("new serve command set: %v", err)
	}
	if len(set.Commands) != 1 {
		t.Fatalf("commands = %d, want 1", len(set.Commands))
	}
	authSection, ok := set.Commands[0].Description().Schema.Get(hostauth.SectionSlug)
	if !ok {
		t.Fatalf("expected auth section on serve command; schema=%#v", set.Commands[0].Description().Schema)
	}
	for _, name := range []string{"auth-mode", "auth-session-cookie-allow-insecure-http", "auth-default-store-driver", "auth-default-store-dsn", "auth-session-store-driver", "auth-audit-store-driver", "auth-appauth-store-driver", "auth-capability-store-driver"} {
		if _, ok := authSection.GetDefinitions().Get(name); !ok {
			t.Fatalf("missing auth field %q", name)
		}
	}
	mode, ok := authSection.GetDefinitions().Get("auth-mode")
	if !ok || mode.Default == nil || *mode.Default != string(hostauth.ModeDev) {
		t.Fatalf("mode default = %#v, want %q", mode.Default, hostauth.ModeDev)
	}
	dsn, ok := authSection.GetDefinitions().Get("auth-default-store-dsn")
	if !ok || dsn.Default == nil || *dsn.Default != "file:auth.db" {
		t.Fatalf("default-store-dsn default = %#v, want file:auth.db", dsn.Default)
	}
}

func TestNewServeCommandSetRejectsMalformedHostAuthServiceFactory(t *testing.T) {
	registry := scanServeTestRegistry(t)
	hostServices := app.HostServices{}
	if err := hostServices.SetHostService(hostauth.ServiceFactoryKey, "not a factory"); err != nil {
		t.Fatalf("SetHostService: %v", err)
	}
	_, err := newServeCommandSet(providerapi.CommandSetContext{
		Name:           "serve",
		Host:           hostServices,
		RuntimeFactory: fakeRuntimeFactory{},
		Sources:        fakeSourceRegistry{jsverbs: fakeJSVerbSourceSet{registries: []*jsverbs.Registry{registry}}},
	})
	if err == nil || !strings.Contains(err.Error(), hostauth.ServiceFactoryKey) {
		t.Fatalf("new serve command set error = %v, want malformed hostauth factory", err)
	}
}

func TestServeVerbLoadsIncludedHelperModulesWithoutHelperCommands(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "site.js"), []byte(`
__package__({ name: "site" });
function start() {
  const express = require("express");
  const app = express.app();
  require("./server.js").register(app);
}
__verb__("start", { name: "start", short: "Serve site", output: "text" });
`), 0o644); err != nil {
		t.Fatalf("write site.js: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "server.js"), []byte(`
function register(app) {
  app.get("/healthz").public().handle((_ctx, res) => res.json({ ok: true, source: "helper" }));
}
function helperThatMustNotBecomeACommand() {
  return "helper";
}
module.exports = { register };
`), 0o644); err != nil {
		t.Fatalf("write server.js: %v", err)
	}

	registry, err := jsverbs.ScanDir(dir)
	if err != nil {
		t.Fatalf("scan dir: %v", err)
	}
	if verbs := registry.Verbs(); len(verbs) != 1 || verbs[0].FullPath() != "site start" {
		t.Fatalf("verbs = %#v, want only site start", verbs)
	}
	set, err := newServeCommandSet(providerapi.CommandSetContext{
		Name:           "serve",
		RuntimeFactory: fakeRuntimeFactory{},
		Sources:        fakeSourceRegistry{jsverbs: fakeJSVerbSourceSet{registries: []*jsverbs.Registry{registry}}},
	})
	if err != nil {
		t.Fatalf("new serve command set: %v", err)
	}
	if len(set.Commands) != 1 {
		t.Fatalf("serve commands = %d, want only the explicit start command", len(set.Commands))
	}

	providers := providerapi.NewProviderRegistry()
	if err := Register(providers); err != nil {
		t.Fatalf("register http provider: %v", err)
	}
	capabilities, ok := providers.ResolvePackageCapabilities(PackageID)
	if !ok || len(capabilities) != 1 {
		t.Fatalf("http capabilities = %#v", capabilities)
	}
	runtimePlan := &app.RuntimePlan{Runtime: app.RuntimeSection{Modules: []app.RuntimeModulePlan{{Provider: PackageID, Name: "express", As: "express"}}}}
	factory := app.NewRuntimeFactory(providers, runtimePlan, app.HostServices{})
	addr := freeServeTestAddr(t)
	parsedValues := serveHotReloadTestValues(t, addr, map[string]any{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		verb, _ := registry.Verb("site start")
		_, err := serveVerb(ctx, providerapi.CommandSetContext{
			RuntimeFactory: factory,
			SelectedModules: []providerapi.ModuleDescriptor{{
				PackageID:           PackageID,
				ModuleID:            "express",
				As:                  "express",
				PackageCapabilities: capabilities,
			}},
		}, registry, verb, parsedValues)
		done <- err
	}()

	if body := waitForServeTestBody(t, "http://"+addr+"/healthz", done); !strings.Contains(body, `"source":"helper"`) {
		t.Fatalf("health body = %s", body)
	}
	cancel()
	select {
	case err := <-done:
		if err != nil && !strings.Contains(err.Error(), "context canceled") {
			t.Fatalf("serve returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("serve did not stop after cancel")
	}
}

func TestServeVerbStartsCommandOwnedServerWithoutExpressListen(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "site.js"), []byte(`
__package__({ name: "site" });
function start() {
  require("express");
}
__verb__("start", { name: "start", short: "Serve empty Express site", output: "text" });
`), 0o644); err != nil {
		t.Fatalf("write site.js: %v", err)
	}
	registry, err := jsverbs.ScanDir(dir)
	if err != nil {
		t.Fatalf("scan dir: %v", err)
	}
	providers := providerapi.NewProviderRegistry()
	if err := Register(providers); err != nil {
		t.Fatalf("register http provider: %v", err)
	}
	capabilities, ok := providers.ResolvePackageCapabilities(PackageID)
	if !ok || len(capabilities) != 1 {
		t.Fatalf("http capabilities = %#v", capabilities)
	}
	runtimePlan := &app.RuntimePlan{Runtime: app.RuntimeSection{Modules: []app.RuntimeModulePlan{{Provider: PackageID, Name: "express", As: "express"}}}}
	factory := app.NewRuntimeFactory(providers, runtimePlan, app.HostServices{})
	addr := freeServeTestAddr(t)
	parsedValues := serveHotReloadTestValues(t, addr, map[string]any{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		verb, _ := registry.Verb("site start")
		_, err := serveVerb(ctx, providerapi.CommandSetContext{
			RuntimeFactory: factory,
			SelectedModules: []providerapi.ModuleDescriptor{{
				PackageID:           PackageID,
				ModuleID:            "express",
				As:                  "express",
				PackageCapabilities: capabilities,
			}},
		}, registry, verb, parsedValues)
		done <- err
	}()

	_ = waitForServeTestStatusBody(t, "http://"+addr+"/__no_route_registered", done, stdhttp.StatusNotFound)
	cancel()
	select {
	case err := <-done:
		if err != nil && !strings.Contains(err.Error(), "context canceled") {
			t.Fatalf("serve returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("serve did not stop after cancel")
	}
}

func TestServeVerbUsesHostAuthServiceFactory(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "site.js"), []byte(`
__package__({ name: "site" });
function start() {
  const express = require("express");
  const app = express.app();
  app.get("/me")
    .auth(express.user().required())
    .allow("user.self.read")
    .handle((ctx, res) => res.json({ actor: ctx.actor.id }));
}
__verb__("start", { name: "start", short: "Serve protected site", output: "text" });
`), 0o644); err != nil {
		t.Fatalf("write site.js: %v", err)
	}
	registry, err := jsverbs.ScanDir(dir)
	if err != nil {
		t.Fatalf("scan dir: %v", err)
	}
	providers := providerapi.NewProviderRegistry()
	if err := Register(providers); err != nil {
		t.Fatalf("register http provider: %v", err)
	}
	capabilities, ok := providers.ResolvePackageCapabilities(PackageID)
	if !ok || len(capabilities) != 1 {
		t.Fatalf("http capabilities = %#v", capabilities)
	}
	hostServices := app.HostServices{}
	if err := hostServices.SetHostService(hostauth.ServiceFactoryKey, hostauth.NewServiceFactory(hostauth.BuilderOptions{Config: hostauth.Config{
		Mode:    hostauth.ModeDev,
		Session: hostauth.SessionConfig{Cookie: hostauth.CookieConfig{AllowInsecureHTTP: true}},
	}})); err != nil {
		t.Fatalf("SetHostService: %v", err)
	}
	runtimePlan := &app.RuntimePlan{Runtime: app.RuntimeSection{Modules: []app.RuntimeModulePlan{{Provider: PackageID, Name: "express", As: "express"}}}}
	factory := app.NewRuntimeFactory(providers, runtimePlan, hostServices)
	addr := freeServeTestAddr(t)
	parsedValues := serveHotReloadTestValues(t, addr, map[string]any{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		verb, _ := registry.Verb("site start")
		_, err := serveVerb(ctx, providerapi.CommandSetContext{
			Host:           hostServices,
			RuntimeFactory: factory,
			SelectedModules: []providerapi.ModuleDescriptor{{
				PackageID:           PackageID,
				ModuleID:            "express",
				As:                  "express",
				PackageCapabilities: capabilities,
			}},
		}, registry, verb, parsedValues)
		done <- err
	}()

	_ = waitForServeTestStatusBody(t, "http://"+addr+"/me", done, stdhttp.StatusUnauthorized)
	cancel()
	select {
	case err := <-done:
		if err != nil && !strings.Contains(err.Error(), "context canceled") {
			t.Fatalf("serve returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("serve did not stop after cancel")
	}
}

func TestServeVerbAppliesHostAuthToExternalHost(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "site.js"), []byte(`
__package__({ name: "site" });
function start() {
  const express = require("express");
  const app = express.app();
  app.get("/external-auth")
    .auth(express.user().required())
    .allow("user.self.read")
    .handle((ctx, res) => res.type("text/plain").send("external host " + ctx.actor.id));
}
__verb__("start", { name: "start", short: "Serve external host", output: "text" });
`), 0o644); err != nil {
		t.Fatalf("write site.js: %v", err)
	}
	registry, err := jsverbs.ScanDir(dir)
	if err != nil {
		t.Fatalf("scan dir: %v", err)
	}
	providers := providerapi.NewProviderRegistry()
	if err := Register(providers); err != nil {
		t.Fatalf("register http provider: %v", err)
	}
	capabilities, ok := providers.ResolvePackageCapabilities(PackageID)
	if !ok || len(capabilities) != 1 {
		t.Fatalf("http capabilities = %#v", capabilities)
	}
	jsHost := gojahttp.NewHost(gojahttp.HostOptions{Dev: true})
	hostServices := app.HostServices{}
	if err := hostServices.SetHostService(HostServiceKey, ExternalHostService{Host: jsHost, OwnsListen: false}); err != nil {
		t.Fatalf("SetHostService external host: %v", err)
	}
	if err := hostServices.SetHostService(hostauth.ServiceFactoryKey, hostauth.NewServiceFactory(hostauth.BuilderOptions{Config: hostauth.Config{
		Mode:    hostauth.ModeDev,
		Session: hostauth.SessionConfig{Cookie: hostauth.CookieConfig{AllowInsecureHTTP: true}},
	}})); err != nil {
		t.Fatalf("SetHostService auth factory: %v", err)
	}
	runtimePlan := &app.RuntimePlan{Runtime: app.RuntimeSection{Modules: []app.RuntimeModulePlan{{Provider: PackageID, Name: "express", As: "express"}}}}
	factory := app.NewRuntimeFactory(providers, runtimePlan, hostServices)
	parsedValues := serveHotReloadTestValues(t, freeServeTestAddr(t), map[string]any{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		verb, _ := registry.Verb("site start")
		_, err := serveVerb(ctx, providerapi.CommandSetContext{
			Host:           hostServices,
			RuntimeFactory: factory,
			SelectedModules: []providerapi.ModuleDescriptor{{
				PackageID:           PackageID,
				ModuleID:            "express",
				As:                  "express",
				PackageCapabilities: capabilities,
			}},
		}, registry, verb, parsedValues)
		done <- err
	}()

	body := waitForServeTestHostBody(t, jsHost, "/external-auth", done, stdhttp.StatusUnauthorized)
	if body == "" {
		t.Fatalf("external host body is empty")
	}
	cancel()
	select {
	case err := <-done:
		if err != nil && !strings.Contains(err.Error(), "context canceled") {
			t.Fatalf("serve returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("serve did not stop after cancel")
	}
}

func TestHostOptionsWithAuthPreservesHTTPSettings(t *testing.T) {
	opts := hostOptionsWithAuth(settings{DevErrors: true, RejectRawRoutes: false}, &hostauth.Services{})
	if !opts.Dev {
		t.Fatalf("Dev = false, want true")
	}
	if opts.RejectRawRoutes {
		t.Fatalf("RejectRawRoutes = true, want false")
	}
	opts = hostOptionsWithAuth(settings{RejectRawRoutes: true}, &hostauth.Services{})
	if !opts.RejectRawRoutes {
		t.Fatalf("RejectRawRoutes = false, want true")
	}
}

func TestServeVerbHotReloadServesStatusAndReloadsChangedSource(t *testing.T) {
	dir := t.TempDir()
	verbPath := filepath.Join(dir, "sites.js")
	writeServeHotReloadVerb(t, verbPath, 1)
	registry, err := jsverbs.ScanDir(dir)
	if err != nil {
		t.Fatalf("scan dir: %v", err)
	}
	verb, ok := registry.Verb("sites demo")
	if !ok {
		t.Fatalf("missing serve verb")
	}

	providers := providerapi.NewProviderRegistry()
	if err := Register(providers); err != nil {
		t.Fatalf("register http provider: %v", err)
	}
	runtimePlan := &app.RuntimePlan{Runtime: app.RuntimeSection{Modules: []app.RuntimeModulePlan{{Provider: PackageID, Name: "express", As: "express"}}}}
	factory := app.NewRuntimeFactory(providers, runtimePlan, app.HostServices{})
	addr := freeServeTestAddr(t)
	parsedValues := serveHotReloadTestValues(t, addr, map[string]any{
		"hot-reload":             true,
		"hot-reload-watch-root":  []string{dir},
		"hot-reload-poll":        "20ms",
		"hot-reload-debounce":    "20ms",
		"hot-reload-smoke-path":  "/healthz",
		"hot-reload-status-path": "/__xgoja/status",
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := serveVerb(ctx, providerapi.CommandSetContext{
			RuntimeFactory: factory,
			Sources:        fakeSourceRegistry{jsverbs: fakeJSVerbSourceSet{path: dir}},
		}, registry, verb, parsedValues)
		done <- err
	}()

	healthURL := "http://" + addr + "/healthz"
	if body := waitForServeTestBody(t, healthURL, done); !strings.Contains(body, `"version":1`) {
		t.Fatalf("initial health body = %s", body)
	}
	statusURL := "http://" + addr + "/__xgoja/status"
	status := waitForServeTestStatus(t, statusURL, done, 1)
	if !status.Ready || status.ActiveVersion != 1 || len(status.Routes) == 0 {
		t.Fatalf("initial status = %#v", status)
	}

	time.Sleep(50 * time.Millisecond)
	writeServeHotReloadVerb(t, verbPath, 2)
	status = waitForServeTestStatus(t, statusURL, done, 2)
	if !status.Ready || status.ActiveVersion < 2 || status.LastError != "" {
		t.Fatalf("reloaded status = %#v", status)
	}
	if body := waitForServeTestBody(t, healthURL, done); !strings.Contains(body, `"version":2`) {
		t.Fatalf("reloaded health body = %s", body)
	}

	time.Sleep(50 * time.Millisecond)
	if err := os.WriteFile(verbPath, []byte(`__package__({ name: "sites" }); function demo( {`), 0o644); err != nil {
		t.Fatalf("write broken verb: %v", err)
	}
	brokenStatus := waitForServeTestStatusError(t, statusURL, done, status.ActiveVersion)
	if brokenStatus.ActiveVersion != status.ActiveVersion {
		t.Fatalf("broken reload changed active version: before=%d after=%d", status.ActiveVersion, brokenStatus.ActiveVersion)
	}
	if body := waitForServeTestBody(t, healthURL, done); !strings.Contains(body, `"version":2`) {
		t.Fatalf("last-known-good health body = %s", body)
	}
	cancel()
	select {
	case err := <-done:
		if err != nil && !strings.Contains(err.Error(), "context canceled") {
			t.Fatalf("serve returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("serve did not stop after cancel")
	}
}

func TestServeVerbHotReloadUsesHostAuthServiceFactory(t *testing.T) {
	dir := t.TempDir()
	verbPath := filepath.Join(dir, "sites.js")
	if err := os.WriteFile(verbPath, []byte(`
__package__({ name: "sites" });
__verb__("demo", { name: "demo", short: "Serve demo", output: "text" });
function demo() {
  const express = require("express");
  const app = express.app();
  app.get("/me")
    .auth(express.user().required())
    .allow("user.self.read")
    .handle((ctx, res) => res.json({ actor: ctx.actor.id }));
}
`), 0o644); err != nil {
		t.Fatalf("write serve verb: %v", err)
	}
	registry, err := jsverbs.ScanDir(dir)
	if err != nil {
		t.Fatalf("scan dir: %v", err)
	}
	verb, ok := registry.Verb("sites demo")
	if !ok {
		t.Fatalf("missing serve verb")
	}

	providers := providerapi.NewProviderRegistry()
	if err := Register(providers); err != nil {
		t.Fatalf("register http provider: %v", err)
	}
	hostServices := app.HostServices{}
	if err := hostServices.SetHostService(hostauth.ServiceFactoryKey, hostauth.NewServiceFactory(hostauth.BuilderOptions{Config: hostauth.Config{
		Mode:    hostauth.ModeDev,
		Session: hostauth.SessionConfig{Cookie: hostauth.CookieConfig{AllowInsecureHTTP: true}},
	}})); err != nil {
		t.Fatalf("SetHostService auth factory: %v", err)
	}
	runtimePlan := &app.RuntimePlan{Runtime: app.RuntimeSection{Modules: []app.RuntimeModulePlan{{Provider: PackageID, Name: "express", As: "express"}}}}
	factory := app.NewRuntimeFactory(providers, runtimePlan, hostServices)
	addr := freeServeTestAddr(t)
	parsedValues := serveHotReloadTestValues(t, addr, map[string]any{
		"hot-reload":            true,
		"hot-reload-watch-root": []string{dir},
		"hot-reload-poll":       "20ms",
		"hot-reload-debounce":   "20ms",
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := serveVerb(ctx, providerapi.CommandSetContext{
			Host:           hostServices,
			RuntimeFactory: factory,
			Sources:        fakeSourceRegistry{jsverbs: fakeJSVerbSourceSet{path: dir}},
		}, registry, verb, parsedValues)
		done <- err
	}()

	_ = waitForServeTestStatusBody(t, "http://"+addr+"/me", done, stdhttp.StatusUnauthorized)
	cancel()
	select {
	case err := <-done:
		if err != nil && !strings.Contains(err.Error(), "context canceled") {
			t.Fatalf("serve returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("serve did not stop after cancel")
	}
}

func scanServeTestRegistry(t *testing.T) *jsverbs.Registry {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "sites.js")
	source := `
__package__({ name: "sites" });
__verb__("demo", { name: "demo", short: "Serve demo", output: "text" });
function demo() {}
`
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatalf("write verb: %v", err)
	}
	registry, err := jsverbs.ScanDir(dir)
	if err != nil {
		t.Fatalf("scan dir: %v", err)
	}
	return registry
}

type fakeRuntimeFactory struct{}

func (fakeRuntimeFactory) NewRuntime(context.Context, ...require.Option) (*engine.Runtime, error) {
	return nil, fmt.Errorf("not implemented")
}

func (fakeRuntimeFactory) NewRuntimeFromSections(context.Context, *values.Values, ...require.Option) (*engine.Runtime, error) {
	return nil, fmt.Errorf("not implemented")
}

func writeServeHotReloadVerb(t *testing.T, path string, version int) {
	t.Helper()
	source := fmt.Sprintf(`
__package__({ name: "sites" });
__verb__("demo", { name: "demo", short: "Serve demo", output: "text" });
function demo() {
  const express = require("express");
  const app = express.app();
  app.get("/healthz").public().handle((_ctx, res) => res.json({ ok: true, version: %d }));
}
`, version)
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatalf("write serve verb: %v", err)
	}
}

func freeServeTestAddr(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}
	return addr
}

func serveHotReloadTestValues(t *testing.T, listen string, hotReloadOverrides map[string]any) *values.Values {
	t.Helper()
	httpSection := httpSectionValues(t, map[string]any{"listen": listen})
	hotReloadSection, err := serveHotReloadSection()
	if err != nil {
		t.Fatalf("hot reload section: %v", err)
	}
	hotReloadValues := sectionValuesWithDefaults(t, hotReloadSection, hotReloadOverrides)
	return values.New(
		values.WithSectionValues("http", httpSection),
		values.WithSectionValues(serveHotReloadSectionSlug, hotReloadValues),
	)
}

func httpSectionValues(t *testing.T, overrides map[string]any) *values.SectionValues {
	t.Helper()
	capability := newHTTPCapability()
	sections, err := capability.GlazedConfigSections(providerapi.SectionRequest{})
	if err != nil {
		t.Fatalf("http sections: %v", err)
	}
	return sectionValuesWithDefaults(t, sections[0], overrides)
}

func sectionValuesWithDefaults(t *testing.T, section schema.Section, overrides map[string]any) *values.SectionValues {
	t.Helper()
	fieldValues := fields.NewFieldValues()
	for _, definition := range section.GetDefinitions().ToList() {
		if definition.Default != nil {
			fieldValues.Set(definition.Name, &fields.FieldValue{Definition: definition, Value: *definition.Default})
		}
	}
	for name, value := range overrides {
		definition, ok := section.GetDefinitions().Get(name)
		if !ok {
			t.Fatalf("unknown field %q", name)
		}
		fieldValues.Set(name, &fields.FieldValue{Definition: definition, Value: value})
	}
	sectionValues, err := values.NewSectionValues(section, values.WithFields(fieldValues))
	if err != nil {
		t.Fatalf("section values: %v", err)
	}
	return sectionValues
}

func waitForServeTestBody(t *testing.T, url string, done <-chan error) string {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case err := <-done:
			t.Fatalf("serve exited early: %v", err)
		default:
		}
		resp, err := stdhttp.Get(url)
		if err == nil {
			data, readErr := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if readErr == nil && resp.StatusCode == stdhttp.StatusOK {
				return string(data)
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", url)
	return ""
}

func waitForServeTestStatusBody(t *testing.T, url string, done <-chan error, statusCode int) string {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case err := <-done:
			t.Fatalf("serve exited early: %v", err)
		default:
		}
		resp, err := stdhttp.Get(url)
		if err == nil {
			data, readErr := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if readErr == nil && resp.StatusCode == statusCode {
				return string(data)
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s status %d", url, statusCode)
	return ""
}

func waitForServeTestHostBody(t *testing.T, host stdhttp.Handler, path string, done <-chan error, statusCode int) string {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case err := <-done:
			t.Fatalf("serve exited early: %v", err)
		default:
		}
		recorder := httptest.NewRecorder()
		host.ServeHTTP(recorder, httptest.NewRequest(stdhttp.MethodGet, path, nil))
		if recorder.Code == statusCode {
			return recorder.Body.String()
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for host path %s status %d", path, statusCode)
	return ""
}

func waitForServeTestStatus(t *testing.T, url string, done <-chan error, version int64) hotreload.Status {
	t.Helper()
	return waitForServeTestStatusWhere(t, url, done, func(status hotreload.Status) bool {
		return status.ActiveVersion >= version
	}, fmt.Sprintf("version %d", version))
}

func waitForServeTestStatusError(t *testing.T, url string, done <-chan error, activeVersion int64) hotreload.Status {
	t.Helper()
	return waitForServeTestStatusWhere(t, url, done, func(status hotreload.Status) bool {
		return status.ActiveVersion == activeVersion && status.LastError != ""
	}, fmt.Sprintf("last error with active version %d", activeVersion))
}

func waitForServeTestStatusWhere(t *testing.T, url string, done <-chan error, accept func(hotreload.Status) bool, label string) hotreload.Status {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case err := <-done:
			t.Fatalf("serve exited early: %v", err)
		default:
		}
		resp, err := stdhttp.Get(url)
		if err == nil {
			data, readErr := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if readErr == nil && resp.StatusCode == stdhttp.StatusOK {
				var status hotreload.Status
				if err := json.Unmarshal(data, &status); err == nil && accept(status) {
					return status
				}
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for status %s from %s", label, url)
	return hotreload.Status{}
}

func TestAppendTypeScriptWatchExtensions(t *testing.T) {
	got := appendTypeScriptWatchExtensions([]string{".js", ".ts"})
	want := []string{".js", ".ts", ".tsx", ".mts", ".cts"}
	if len(got) != len(want) {
		t.Fatalf("extensions = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("extensions = %#v, want %#v", got, want)
		}
	}
}

func TestSourceSetHasTypeScript(t *testing.T) {
	if sourceSetHasTypeScript(fakeJSVerbSourceSet{}) {
		t.Fatalf("empty fake source set unexpectedly has TypeScript")
	}
	if !sourceSetHasTypeScript(fakeJSVerbSourceSet{typescript: true}) {
		t.Fatalf("TypeScript-enabled fake source set was not detected")
	}
}

type fakeSourceRegistry struct {
	jsverbs fakeJSVerbSourceSet
}

func (r fakeSourceRegistry) ListSources() []providerapi.RuntimeSourceDescriptor {
	return r.ListSourcesByKind(providerapi.RuntimeSourceKindJSVerbs)
}

func (r fakeSourceRegistry) ListSourcesByKind(kind providerapi.RuntimeSourceKind) []providerapi.RuntimeSourceDescriptor {
	if kind != providerapi.RuntimeSourceKindJSVerbs {
		return nil
	}
	out := make([]providerapi.RuntimeSourceDescriptor, 0, len(r.jsverbs.ListJSVerbSources()))
	for _, source := range r.jsverbs.ListJSVerbSources() {
		out = append(out, providerapi.RuntimeSourceDescriptor{ID: source.ID, Kind: providerapi.RuntimeSourceKindJSVerbs, Path: source.Path, Embed: source.Embed, Provider: source.Provider, Source: source.Source, TypeScript: source.TypeScript})
	}
	return out
}

func (r fakeSourceRegistry) SourceByID(id string) (providerapi.RuntimeSourceDescriptor, bool) {
	for _, source := range r.ListSources() {
		if source.ID == id {
			return source, true
		}
	}
	return providerapi.RuntimeSourceDescriptor{}, false
}

func (r fakeSourceRegistry) JSVerbs() providerapi.JSVerbSourceSet {
	return r.jsverbs
}

type fakeJSVerbSourceSet struct {
	path       string
	registries []*jsverbs.Registry
	typescript bool
}

func (s fakeJSVerbSourceSet) ListJSVerbSources() []providerapi.JSVerbSourceDescriptor {
	descriptor := providerapi.JSVerbSourceDescriptor{ID: "fake", Path: s.path}
	if s.typescript {
		descriptor.TypeScript = &providerapi.TypeScriptDescriptor{Enabled: true}
	}
	return []providerapi.JSVerbSourceDescriptor{descriptor}
}

func (s fakeJSVerbSourceSet) ScanJSVerbSource(id string) (*jsverbs.Registry, error) {
	if id != "fake" {
		return nil, fmt.Errorf("unknown fake source %q", id)
	}
	if s.path != "" {
		return jsverbs.ScanDir(s.path)
	}
	if len(s.registries) == 0 {
		return nil, nil
	}
	return s.registries[0], nil
}

func (s fakeJSVerbSourceSet) ScanAllJSVerbSources() ([]*jsverbs.Registry, error) {
	if s.path != "" {
		registry, err := jsverbs.ScanDir(s.path)
		if err != nil {
			return nil, err
		}
		return []*jsverbs.Registry{registry}, nil
	}
	return s.registries, nil
}
