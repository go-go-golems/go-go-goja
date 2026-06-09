---
Title: External Go HTTP Host Integration Implementation Guide
Ticket: XGOJA-EXTERNAL-HTTP-HOST
Status: active
Topics:
    - xgoja
    - goja
    - architecture
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/modules/express/express.go
      Note: Express module accepts gojahttp.Host and registers routes
    - Path: go-go-goja/pkg/gojahttp/host.go
      Note: Go HTTP handler that dispatches requests into JavaScript routes
    - Path: go-go-goja/pkg/xgoja/app/assets.go
      Note: Concrete HostServices service map and AssetStore
    - Path: go-go-goja/pkg/xgoja/app/factory.go
      Note: RuntimeFactory passes services to module setup and collects provider contributions
    - Path: go-go-goja/pkg/xgoja/app/host.go
      Note: HostOptions and generated app host construction
    - Path: go-go-goja/pkg/xgoja/providerapi/module.go
      Note: ModuleSetupContext and HostServices provider-facing API
    - Path: go-go-goja/pkg/xgoja/providers/http/http.go
      Note: HTTP provider express module and listener lifecycle
ExternalSources:
    - https://github.com/go-go-golems/go-go-goja/issues/65
Summary: Non-invasive implementation guide for letting generated xgoja runtime packages register JavaScript HTTP routes into a Go-owned gojahttp.Host, with runtime-manager validation guidance.
LastUpdated: 2026-06-09T01:25:00-04:00
WhatFor: Use this when implementing external Go HTTP host integration for generated xgoja runtime packages without renaming HostServices APIs yet.
WhenToUse: Read before changing xgoja HostOptions, generated package templates, the HTTP provider, gojahttp route introspection, or runtime-manager prototypes.
---


# External Go HTTP Host Integration Implementation Guide

## Executive summary

This ticket is about one practical integration gap in `go-go-goja`: generated xgoja runtime packages can be imported by a Go program, and the low-level Express module can already register routes into a caller-supplied `*gojahttp.Host`, but the generated xgoja provider path does not yet connect those two facts. The HTTP provider currently hides its `gojahttp.Host` inside provider state and can start its own listener. That is right for generated CLI/server binaries, but it is not enough for a hybrid application where Go owns the `net/http` server and only delegates selected routes to JavaScript.

The planned near-term approach is intentionally non-invasive. We will not rename `providerapi.HostServices`, `HostServiceContributionCapability`, or `ModuleSetupContext.Host` in this ticket. That larger API cleanup is tracked separately in GitHub issue #65. Instead, this ticket adds a small, current-name-compatible service injection hook to generated packages, teaches the HTTP provider to consume an externally supplied `*gojahttp.Host`, and adds enough route introspection/tests to prove that JavaScript routes register into the Go-owned host without the provider binding a TCP listener.

The target developer experience is this:

```go
jsHost := gojahttp.NewHost(gojahttp.HostOptions{Dev: true})

bundle, err := xgojaruntime.NewBundle(xgojaruntime.Options{
    ConfigureServices: func(s *app.HostServices) {
        _ = s.SetHostService(httpprovider.HostServiceKey, httpprovider.ExternalHostService{
            Host:       jsHost,
            OwnsListen: false,
        })
    },
})
if err != nil { return err }

rt, err := bundle.NewRuntime(ctx)
if err != nil { return err }
defer rt.Close(ctx)

_, err = rt.Owner.Call(ctx, "bootstrap-js-routes", func(ctx context.Context, vm *goja.Runtime) (any, error) {
    _, err := vm.RunString(`require("./routes").register()`)
    return nil, err
})
if err != nil { return err }

mux := http.NewServeMux()
mux.Handle("/pages/", jsHost)
mux.HandleFunc("POST /upload", uploadHandler)
```

The key invariant is: **the outer Go application owns the listener and mux; JavaScript only registers route handlers into a `gojahttp.Host` that Go mounts.**

---

## 1. Problem statement and scope

### 1.1 Problem

Generated xgoja packages are good for embedding a configured JavaScript runtime in a Go application, but they do not currently provide a clean service-injection hook for live Go host objects. The `runtime_package.go.tmpl` generated `Options` type only contains output and Cobra middleware fields, and `NewBundle` constructs `app.HostOptions` without any custom service configuration. Evidence: `cmd/xgoja/internal/generate/templates/runtime_package.go.tmpl:43-46` defines `Options`, and `runtime_package.go.tmpl:83-95` constructs `app.NewHostWithOptions(...)` with embedded resources, output, and middleware only.

The HTTP provider also currently ignores `ModuleSetupContext` in its `express` module factory and creates its own `gojahttp.Host` lazily. Evidence: `pkg/xgoja/providers/http/http.go:28-34` registers `express`, but the module factory does not inspect `ctx.Host`; `http.go:109-122` creates `gojahttp.NewHost(...)` when `entry.host == nil`; and `http.go:136-162` starts a TCP listener when HTTP is enabled.

That means a Go application cannot easily say: "here is my `*gojahttp.Host`; please let JavaScript register routes into it, but do not start a listener because my Go mux owns the server."

### 1.2 Scope

In scope for this ticket:

- Add a non-breaking service configuration hook to `app.HostOptions` and generated package/source-fragment `Options`.
- Add helper methods on `app.HostServices` so embedding code can add keyed services without manually editing `map[string][]any`.
- Add an HTTP provider service key and typed payload for externally supplied `*gojahttp.Host`.
- Update the HTTP provider's `express` module factory to inspect `ModuleSetupContext.Host` and use external-host mode when configured.
- Ensure external-host mode does not bind a TCP listener when `OwnsListen == false`.
- Add route introspection to `gojahttp.Registry` / `gojahttp.Host` for tests and debug/status endpoints.
- Document and test a runtime-manager validation pattern: reload candidate runtime, bootstrap JS routes, smoke-test host, atomically swap, keep last-known-good on failure.

Out of scope for this ticket:

- The breaking `HostService*` to `RuntimeService*` rename. That is tracked in issue #65.
- A full generic runtime manager package in `go-go-goja` unless an app-local prototype proves the API shape first.
- Rewriting all generated HTTP serve commands.
- Moving application-specific upload/session/minitrace logic out of ClubMedMeetup.
- Making goja a secure multi-tenant sandbox. The JavaScript here is trusted application code.

### 1.3 Implementation attitude

This ticket should be small enough to review in phases. It should preserve existing generated binary behavior: when no external host service is configured, the HTTP provider should continue to create its own `gojahttp.Host` and start its own listener on first Express use, using the lazy-binding behavior already introduced in `f16430e`.

---

## 2. Vocabulary for a new intern

This section defines terms before using them in the rest of the guide.

### xgoja

`xgoja` is the generated-runtime layer in `go-go-goja`. It reads an `xgoja.yaml` build spec and produces either a standalone binary or an importable Go package. The generated runtime selects provider modules, embeds optional assets/help/jsverbs, creates a `providerapi.ProviderRegistry`, and builds `engine.Runtime` instances.

### Generated runtime package

A generated runtime package is output from `target.kind: package` or source-fragment generation. It exposes functions such as `DecodeSpec`, `RegisterProviders`, `NewBundle`, `Bundle.NewRuntime`, and `Bundle.NewRuntimeFromSections`. Evidence: the package template defines `Options`, `Bundle`, `NewBundle`, and runtime creation methods in `cmd/xgoja/internal/generate/templates/runtime_package.go.tmpl:43-118`. Fragmented source mode mirrors the bundle API in `cmd/xgoja/internal/generate/templates/bundle_fragment.go.tmpl:18-73`.

### app.Host

`app.Host` is the xgoja app-side object that ties together a provider registry, runtime spec, runtime factory, embedded resources, and generated Cobra commands. Evidence: `pkg/xgoja/app/host.go:12-22` defines the host fields; `host.go:36-52` builds a `HostServices` value and a `RuntimeFactory`.

### providerapi.ModuleSetupContext

`ModuleSetupContext` is passed to provider module factories while xgoja creates a runtime. It contains the module alias, JSON config, host services, runtime owner, and closer registration callback. Evidence: `pkg/xgoja/providerapi/module.go:14-21`.

For this ticket we keep the existing field name:

```go
type ModuleSetupContext struct {
    Context      context.Context
    Name         string
    As           string
    Config       json.RawMessage
    Host         HostServices
    RuntimeOwner runtimeowner.RuntimeOwner
    AddCloser    func(func(context.Context) error) error
}
```

A future breaking rename may make this `Services` instead of `Host`, but this ticket deliberately does not do that.

### HostServices and HostServiceLookup

`providerapi.HostServices` is the provider-facing interface currently used for built-in host resources such as embedded asset resolution. Evidence: `pkg/xgoja/providerapi/module.go:24-37` defines `AssetResolver`, `HostServices`, and `HostServiceLookup`.

`app.HostServices` is the concrete app implementation. It already contains an `Assets *AssetStore` and `Services map[string][]any`. Evidence: `pkg/xgoja/app/assets.go:20-23`. It already implements `HostService` and `HostServiceValues`. Evidence: `assets.go:56-79`.

### HostServiceContributionCapability

`HostServiceContributionCapability` is the existing provider capability that lets selected providers contribute runtime-scoped services before module setup. Evidence: `pkg/xgoja/providerapi/capabilities.go:70-98`. The runtime factory collects those contributions before registering module loaders. Evidence: `pkg/xgoja/app/factory.go:143-173`.

This ticket does not rename or redesign that capability. It only ensures embedding applications can supply services into the same final service bag.

### gojahttp.Host

`gojahttp.Host` is an `http.Handler` that stores JavaScript routes/static mounts and dispatches HTTP requests into the runtime owner. Evidence: `pkg/gojahttp/host.go:35-40` constructs the host and registers routes, while `host.go:88-146` implements `ServeHTTP`. Request handlers enter JavaScript through `h.owner.Call(...)` at `host.go:123-133`.

### Express module

The low-level `modules/express` package already accepts a caller-supplied `*gojahttp.Host`. Evidence: `modules/express/express.go:27-35` defines `NewRegistrar(host, opts...)`, and `express.go:68-78` defines `NewLoader(host, opts...)`. Route methods call the optional start hook and register callbacks into `r.host`. Evidence: `express.go:132-188`.

### HTTP provider

The xgoja HTTP provider exposes the `express` module and HTTP-related runtime configuration. Its package ID is `go-go-goja-http`. Evidence: `pkg/xgoja/providers/http/http.go:23-45`. Today it owns a per-runtime `runtimeEntry` containing `host` and `server`, as shown in `http.go:53-58`.

---

## 3. Current-state architecture

### 3.1 Runtime construction today

The current runtime construction path looks like this:

```text
xgojaruntime.NewBundle(opts)
  -> providerapi.NewProviderRegistry()
  -> RegisterProviders(registry)
  -> DecodeSpec()
  -> app.NewHostWithOptions(registry, runtimeSpec, app.HostOptions{...})
      -> app.HostServices{Assets: NewAssetStore(...)}
      -> app.NewRuntimeFactory(providers, runtimeSpec, services)

Bundle.NewRuntime(ctx)
  -> Host.Factory.NewRuntime(ctx)
      -> RuntimeFactory.NewRuntimeFromSections(ctx, nil)
      -> collect provider HostServiceContributionCapability values
      -> for each selected module:
          providerRuntimeModuleRegistrar.RegisterRuntimeModule(...)
          module.NewModuleFactory(ModuleSetupContext{Host: runtimeServices})
```

Evidence:

- `app.NewHostWithOptions` creates `HostServices` and passes it to `NewRuntimeFactory` in `pkg/xgoja/app/host.go:36-45`.
- `providerRuntimeModuleRegistrar.RegisterRuntimeModule` passes `Host: s.services` into `ModuleSetupContext` in `pkg/xgoja/app/factory.go:46-54`.
- `RuntimeFactory.hostServicesForRuntime` gathers provider contributions before module setup in `pkg/xgoja/app/factory.go:143-173`.

### 3.2 Why generated package mode is almost enough

Generated package mode already exposes a host application API:

```go
bundle, err := xgojaruntime.NewBundle(xgojaruntime.Options{})
rt, err := bundle.NewRuntime(ctx)
```

The problem is not that generated packages cannot create runtimes. They can. The problem is that `Options` has no service-injection hook. Evidence: `runtime_package.go.tmpl:43-46` and `bundle_fragment.go.tmpl:18-21` define only `Out` and `MiddlewaresFunc`.

### 3.3 Why the low-level Express module is already suitable

The Express module is not the problem. It is already designed around a supplied `*gojahttp.Host`:

```go
func NewRegistrar(host *gojahttp.Host, opts ...Option) *Registrar
func NewLoader(host *gojahttp.Host, opts ...Option) require.ModuleLoader
```

Routes call `r.host.Register(...)` after the start hook succeeds. Evidence: `modules/express/express.go:136-145`.

The missing bridge is in the HTTP provider. The provider must use `ctx.Host` from `ModuleSetupContext` to see whether the embedding Go application supplied an external host.

### 3.4 What changed recently and why it matters

The runtime polish work already made `require("express")` side-effect-light. The provider no longer binds an HTTP listener during `require("express")`; route/static registration or `app.listen()` triggers the start hook. Evidence: `modules/express/express.go:141-144`, `express.go:152-155`, `express.go:162-169`, `express.go:176-184`, and `express.go:187` call `r.start(vm)` before app use, while `pkg/xgoja/providers/http/http.go:119-121` passes the start hook into `express.NewLoader`.

This is important because external-host mode can now be expressed as a start hook that no-ops when the outer Go host owns the listener.

---

## 4. Gap analysis

### Gap 1: app.HostOptions cannot configure services

`app.HostOptions` currently has embedded filesystem fields, output, and middleware configuration, but no service configuration hook. Evidence: `pkg/xgoja/app/host.go:24-30`.

Consequence: direct callers of `app.NewHostWithOptions` and generated package callers cannot inject live Go services in a standard way.

### Gap 2: generated package Options cannot configure services

The generated `Options` types in `runtime_package.go.tmpl` and `bundle_fragment.go.tmpl` do not expose a hook for services. Evidence: `runtime_package.go.tmpl:43-46` and `bundle_fragment.go.tmpl:18-21`.

Consequence: a host application importing a generated package cannot pass `*gojahttp.Host`, DB handles, stores, or other services without forking the generated template.

### Gap 3: HTTP provider ignores ModuleSetupContext.Host

The HTTP provider currently registers its module like this:

```go
NewModuleFactory: func(providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
    return capability.NewExpressLoader(), nil
},
```

Evidence: `pkg/xgoja/providers/http/http.go:28-34`.

Consequence: even if the service bag contains an external host, the provider will not see it.

### Gap 4: gojahttp has no route introspection

`gojahttp.Registry` can add and match routes, but it does not expose a copy-safe route list. Evidence: `pkg/gojahttp/route_registry.go:21-44` defines `Add` and `Match`, and there is no `Routes()` method.

Consequence: tests can prove behavior by issuing HTTP requests, but runtime-manager status/debug endpoints cannot easily report registered JS routes. Hot reload is easier to validate when route metadata is visible.

---

## 5. Proposed non-invasive architecture

### 5.1 High-level diagram

```text
Go application
  │
  ├── owns net/http.Server and ServeMux
  │
  ├── creates jsHost := gojahttp.NewHost(...)
  │
  ├── imports generated xgoja runtime package
  │       └── NewBundle(Options{ConfigureServices: ...})
  │              └── app.HostServices receives ExternalHostService
  │
  ├── creates runtime through Bundle.NewRuntime(ctx)
  │       └── HTTP provider sees ctx.Host service
  │              └── express loader uses jsHost
  │
  ├── bootstraps JS route registration
  │       └── JS calls require("express").app().get(...)
  │              └── route registers into jsHost
  │
  └── mounts jsHost under selected prefixes
          ├── mux.Handle("/pages/", jsHost)
          └── mux.Handle("/js/", http.StripPrefix("/js", jsHost))
```

### 5.2 Host service injection without renaming APIs

Add a service configuration hook while keeping the current names:

```go
// pkg/xgoja/app/host.go

type HostOptions struct {
    EmbeddedJSVerbs fs.FS
    EmbeddedHelp    fs.FS
    EmbeddedAssets  fs.FS
    Out             io.Writer
    MiddlewaresFunc cli.CobraMiddlewaresFunc

    ConfigureServices func(*HostServices)
}
```

Add helper methods to `app.HostServices`:

```go
// pkg/xgoja/app/assets.go or host_services.go

func (s *HostServices) SetHostService(key string, value any) error {
    key = strings.TrimSpace(key)
    if key == "" {
        return fmt.Errorf("host service key is required")
    }
    if value == nil {
        return fmt.Errorf("host service %q value is nil", key)
    }
    if s.Services == nil {
        s.Services = map[string][]any{}
    }
    s.Services[key] = []any{value}
    return nil
}

func (s *HostServices) AddHostService(key string, value any) error {
    key = strings.TrimSpace(key)
    if key == "" {
        return fmt.Errorf("host service key is required")
    }
    if value == nil {
        return fmt.Errorf("host service %q value is nil", key)
    }
    if s.Services == nil {
        s.Services = map[string][]any{}
    }
    s.Services[key] = append(s.Services[key], value)
    return nil
}
```

Then wire the callback:

```go
func NewHostWithOptions(providers *providerapi.ProviderRegistry, runtimeSpec *RuntimeSpec, opts HostOptions) *Host {
    services := HostServices{Assets: NewAssetStore(opts.EmbeddedAssets, runtimeSpec)}
    if opts.ConfigureServices != nil {
        opts.ConfigureServices(&services)
    }
    // existing MiddlewaresFunc and Host construction
}
```

The callback is intentionally `func(*HostServices)` rather than `func(*HostServices) error` because `NewHostWithOptions` currently returns `*Host`, not `(*Host, error)`. This keeps the change additive and avoids forcing a broad signature change. Provider setup still validates service type/contents and returns normal runtime construction errors.

### 5.3 Generated package options

Update generated package and bundle-fragment templates:

```go
type Options struct {
    Out             io.Writer
    MiddlewaresFunc cli.CobraMiddlewaresFunc
    ConfigureServices func(*app.HostServices)
}
```

Pass it through:

```go
host := app.NewHostWithOptions(registry, runtimeSpec, app.HostOptions{
    EmbeddedJSVerbs: embeddedJSVerbs,
    EmbeddedHelp:    embeddedHelp,
    EmbeddedAssets:  embeddedAssets,
    Out: opts.Out,
    MiddlewaresFunc: opts.MiddlewaresFunc,
    ConfigureServices: opts.ConfigureServices,
})
```

This must be applied to both:

- `cmd/xgoja/internal/generate/templates/runtime_package.go.tmpl`
- `cmd/xgoja/internal/generate/templates/bundle_fragment.go.tmpl`

because package and source-fragment generation must expose the same embedding hook.

### 5.4 HTTP provider external host service

Add a typed payload to `pkg/xgoja/providers/http`:

```go
const HostServiceKey = "go-go-goja-http.host"

type ExternalHostService struct {
    Host       *gojahttp.Host
    OwnsListen bool
}
```

Semantics:

- `Host` is the `gojahttp.Host` supplied by the embedding Go application.
- `OwnsListen == false` means xgoja must not bind a TCP listener; the outer Go application owns the listener/mux.
- `OwnsListen == true` is reserved for advanced cases where the provider may use the supplied host but still own the listener. The first implementation should primarily test and support `false`.

The provider's module factory should become context-aware:

```go
providerapi.Module{
    Name: "express",
    DefaultAs: "express",
    Description: "Express-style HTTP route registration backed by gojahttp",
    NewModuleFactory: func(ctx providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
        return capability.NewExpressLoader(ctx.Host)
    },
}
```

Then the loader factory can resolve services once during module setup:

```go
func (c *capability) NewExpressLoader(hostServices providerapi.HostServices) (require.ModuleLoader, error) {
    external, err := externalHostService(hostServices)
    if err != nil {
        return nil, err
    }
    return func(vm *goja.Runtime, moduleObj *goja.Object) {
        entry := c.entry(vm)
        entry.mu.Lock()
        if entry.host == nil {
            if external.Host != nil {
                entry.host = external.Host
                entry.external = true
                entry.ownsListen = external.OwnsListen
            } else {
                entry.host = gojahttp.NewHost(gojahttp.HostOptions{})
                entry.external = false
                entry.ownsListen = true
            }
        }
        host := entry.host
        entry.mu.Unlock()

        express.NewLoader(host, express.WithOnUse(func(vm *goja.Runtime) error {
            return c.start(vm, entry)
        }))(vm, moduleObj)
    }, nil
}
```

Service resolution should validate type errors clearly:

```go
func externalHostService(host providerapi.HostServices) (ExternalHostService, error) {
    lookup, ok := host.(providerapi.HostServiceLookup)
    if !ok || lookup == nil {
        return ExternalHostService{}, nil
    }
    raw, ok := lookup.HostService(HostServiceKey)
    if !ok {
        return ExternalHostService{}, nil
    }
    svc, ok := raw.(ExternalHostService)
    if !ok {
        return ExternalHostService{}, fmt.Errorf("http host service %q must be ExternalHostService, got %T", HostServiceKey, raw)
    }
    if svc.Host == nil {
        return ExternalHostService{}, fmt.Errorf("http host service %q has nil Host", HostServiceKey)
    }
    return svc, nil
}
```

### 5.5 Start behavior matrix

The HTTP provider `start` method should make listener ownership explicit:

| Mode | `entry.host` | `entry.external` | `entry.ownsListen` | `start` behavior |
|---|---|---:|---:|---|
| Existing generated binary | provider-created host | false | true | bind configured listener |
| Go-owned external host | app-supplied host | true | false | no-op; routes already register into external host |
| Advanced external host with provider listener | app-supplied host | true | true | bind configured listener using supplied host |
| HTTP disabled | either | any | any | no-op |

Pseudocode:

```go
func (c *capability) start(vm *goja.Runtime, entry *runtimeEntry) error {
    entry.mu.Lock()
    defer entry.mu.Unlock()

    cfg := normalizeSettings(entry.settings)
    entry.settings = cfg
    if !cfg.Enabled {
        return nil
    }
    if entry.host == nil {
        entry.host = gojahttp.NewHost(gojahttp.HostOptions{})
        entry.external = false
        entry.ownsListen = true
    }
    if entry.external && !entry.ownsListen {
        return nil
    }
    if entry.server != nil {
        return nil
    }
    // existing net.Listen + server.Serve path
}
```

This preserves self-listening behavior while supporting Go-owned listeners.

### 5.6 Route introspection

Add copy-safe route descriptors:

```go
type RouteDescriptor struct {
    Method  string `json:"method"`
    Pattern string `json:"pattern"`
}

func (r *Registry) Routes() []RouteDescriptor {
    r.mu.RLock()
    defer r.mu.RUnlock()
    out := make([]RouteDescriptor, 0, len(r.routes))
    for _, route := range r.routes {
        out = append(out, RouteDescriptor{Method: route.Method, Pattern: route.Pattern})
    }
    return out
}

func (h *Host) Routes() []RouteDescriptor {
    if h == nil || h.registry == nil {
        return nil
    }
    return h.registry.Routes()
}
```

This intentionally does not expose Goja callables. It is for debugging, tests, and runtime-manager status pages.

---

## 6. Runtime manager: what to validate now

A full generic runtime manager should not be the first implementation step in `go-go-goja`. The first proof should be app-local, because real applications will teach us which hooks need to be generic. However, the external-host work should be designed so a manager is easy to write.

### 6.1 Runtime manager responsibilities

A runtime manager for a hybrid Go host should:

- create a fresh `gojahttp.Host` for each candidate runtime;
- create a fresh xgoja runtime with `ExternalHostService{Host: candidateHost, OwnsListen: false}`;
- run JavaScript bootstrap code that registers routes;
- smoke-test the candidate host or candidate runtime;
- atomically swap the active handle on success;
- keep the last-known-good runtime on bootstrap/smoke failure;
- close replaced runtimes after a grace period;
- expose status: active version, last reload time, last error, registered routes.

### 6.2 Minimal runtime manager shape

```go
type RuntimeBundle interface {
    NewRuntime(context.Context, ...require.Option) (*engine.Runtime, error)
}

type Manager struct {
    bundle RuntimeBundle
    opts   Options
    active atomic.Pointer[Handle]
    mu     sync.Mutex
    seq    atomic.Int64
}

type Handle struct {
    ID        string
    Version   int64
    CreatedAt time.Time
    Runtime   *engine.Runtime
    HTTPHost  *gojahttp.Host
    Routes    []gojahttp.RouteDescriptor
}

type Options struct {
    NewHTTPHost func() *gojahttp.Host
    NewBundle   func(*gojahttp.Host) (RuntimeBundle, error)
    Bootstrap   func(context.Context, *engine.Runtime, *gojahttp.Host) error
    SmokeTest   func(context.Context, *engine.Runtime, *gojahttp.Host) error
    CloseGrace  time.Duration
}
```

The `NewBundle` hook is where generated-package service injection happens:

```go
NewBundle: func(jsHost *gojahttp.Host) (RuntimeBundle, error) {
    return xgojaruntime.NewBundle(xgojaruntime.Options{
        ConfigureServices: func(s *app.HostServices) {
            _ = s.SetHostService(httpprovider.HostServiceKey, httpprovider.ExternalHostService{
                Host:       jsHost,
                OwnsListen: false,
            })
        },
    })
}
```

### 6.3 Reload pseudocode

```go
func (m *Manager) Reload(ctx context.Context) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    jsHost := m.newHTTPHost()
    bundle, err := m.opts.NewBundle(jsHost)
    if err != nil { return err }

    rt, err := bundle.NewRuntime(ctx)
    if err != nil { return err }
    candidate := &Handle{
        ID:        newID(),
        Version:   m.seq.Add(1),
        CreatedAt: time.Now(),
        Runtime:   rt,
        HTTPHost:  jsHost,
    }

    success := false
    defer func() {
        if !success {
            _ = rt.Close(ctx)
        }
    }()

    if err := m.opts.Bootstrap(ctx, rt, jsHost); err != nil {
        return fmt.Errorf("bootstrap JS routes: %w", err)
    }
    if err := m.opts.SmokeTest(ctx, rt, jsHost); err != nil {
        return fmt.Errorf("smoke-test candidate runtime: %w", err)
    }

    candidate.Routes = jsHost.Routes()
    old := m.active.Swap(candidate)
    success = true

    if old != nil {
        m.closeOldLater(old)
    }
    return nil
}
```

### 6.4 Active handler pseudocode

```go
func (m *Manager) ActiveHandler() http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        h := m.active.Load()
        if h == nil || h.HTTPHost == nil {
            http.Error(w, "JavaScript runtime not ready", http.StatusServiceUnavailable)
            return
        }
        h.HTTPHost.ServeHTTP(w, r)
    })
}
```

This handler captures the active handle at request start. A reload can swap the active pointer while the request continues using the old handle. The old runtime should not be closed immediately; use a small grace period or request tracking if needed.

---

## 7. Implementation phases

### Phase 1: app.HostServices configuration helpers

Files:

- `pkg/xgoja/app/assets.go`
- `pkg/xgoja/app/host.go`
- `pkg/xgoja/app/assets_test.go` or new `host_services_external_test.go`

Tasks:

1. Add `SetHostService` and `AddHostService` helper methods to `app.HostServices`.
2. Add `ConfigureServices func(*HostServices)` to `app.HostOptions`.
3. Call `ConfigureServices` after the `AssetStore` is created and before `NewRuntimeFactory` is called.
4. Add a test provider whose module factory reads a configured service from `ctx.Host`.
5. Ensure embedded asset resolution still works.

Test sketch:

```go
func TestHostOptionsConfigureServicesVisibleToModuleSetup(t *testing.T) {
    registry := providerapi.NewProviderRegistry()
    seen := ""
    registry.Package("fixture", providerapi.Module{
        Name: "mod",
        NewModuleFactory: func(ctx providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
            lookup := ctx.Host.(providerapi.HostServiceLookup)
            raw, _ := lookup.HostService("demo")
            seen = raw.(string)
            return func(*goja.Runtime, *goja.Object) {}, nil
        },
    })
    host := app.NewHostWithOptions(registry, spec, app.HostOptions{
        ConfigureServices: func(s *app.HostServices) {
            _ = s.SetHostService("demo", "from-host")
        },
    })
    rt, err := host.Factory.NewRuntime(context.Background())
    require.NoError(t, err)
    defer rt.Close(context.Background())
    require.Equal(t, "from-host", seen)
}
```

### Phase 2: generated package/source-fragment service hook

Files:

- `cmd/xgoja/internal/generate/templates/runtime_package.go.tmpl`
- `cmd/xgoja/internal/generate/templates/bundle_fragment.go.tmpl`
- `cmd/xgoja/internal/generate/generate_test.go`
- `examples/xgoja/14-generated-runtime-package/internal/xgojaruntime/xgoja_runtime.gen.go` if committed generated fixtures are maintained.

Tasks:

1. Add `ConfigureServices func(*app.HostServices)` to generated `Options`.
2. Pass it through to `app.HostOptions`.
3. Add template rendering assertions.
4. Add generated package compile/smoke test proving a host app can pass a service visible to a selected module.

Validation:

```bash
go test ./cmd/xgoja/internal/generate -run 'Generated.*Package|RuntimePackage' -count=1
```

### Phase 3: HTTP provider external-host mode

Files:

- `pkg/xgoja/providers/http/http.go`
- `pkg/xgoja/providers/http/http_test.go`

Tasks:

1. Add `HostServiceKey` and `ExternalHostService`.
2. Change `NewModuleFactory` to pass `ctx.Host` into loader construction.
3. Resolve and validate the external host service during module setup.
4. Extend `runtimeEntry` with `external` and `ownsListen` booleans.
5. Make `start` no-op when `external && !ownsListen`.
6. Keep default provider-created host/listener behavior unchanged.

Test sketch:

```go
func TestExpressProviderRegistersIntoExternalHost(t *testing.T) {
    jsHost := gojahttp.NewHost(gojahttp.HostOptions{Dev: true})
    registry := providerapi.NewProviderRegistry()
    require.NoError(t, httpprovider.Register(registry))

    spec := &app.RuntimeSpec{Modules: []app.ModuleInstanceSpec{{
        Package: httpprovider.PackageID,
        Name:    "express",
        As:      "express",
    }}}

    host := app.NewHostWithOptions(registry, spec, app.HostOptions{
        ConfigureServices: func(s *app.HostServices) {
            _ = s.SetHostService(httpprovider.HostServiceKey, httpprovider.ExternalHostService{
                Host:       jsHost,
                OwnsListen: false,
            })
        },
    })

    rt, err := host.Factory.NewRuntime(context.Background())
    require.NoError(t, err)
    defer rt.Close(context.Background())

    _, err = rt.Owner.Call(context.Background(), "register", func(_ context.Context, vm *goja.Runtime) (any, error) {
        _, err := vm.RunString(`
            const express = require("express");
            const app = express.app();
            app.get("/hello/:name", (req, res) => res.json({ hello: req.params.name }));
        `)
        return nil, err
    })
    require.NoError(t, err)

    rr := httptest.NewRecorder()
    jsHost.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/hello/goja", nil))
    require.Equal(t, http.StatusOK, rr.Code)
    require.Contains(t, rr.Body.String(), `"hello":"goja"`)
}
```

Add a second test where the listen address is occupied and route registration still succeeds in external no-listen mode. That proves the provider did not bind.

### Phase 4: route introspection

Files:

- `pkg/gojahttp/route_registry.go`
- `pkg/gojahttp/host.go`
- `pkg/gojahttp/*_test.go`

Tasks:

1. Add `RouteDescriptor`.
2. Add `Registry.Routes()`.
3. Add `Host.Routes()`.
4. Test that returned slices are copy-safe and do not expose callables.
5. Use `Host.Routes()` in HTTP external-host tests or runtime-manager status tests.

### Phase 5: app-local RuntimeManager prototype and validation

Recommended initial location for an application proof:

```text
ClubMedMeetup/minitrace-viz/internal/jsruntime/manager.go
ClubMedMeetup/minitrace-viz/internal/jsruntime/watch.go
ClubMedMeetup/minitrace-viz/internal/jsruntime/status.go
```

Do not extract to `go-go-goja/pkg/xgoja/runtimehost` until this proof clarifies the stable API.

Validation scenarios:

1. Initial reload registers `/version` and serves `version: 1`.
2. A broken reload keeps serving old `version: 1`.
3. A successful reload swaps to `version: 2`.
4. A long request on old runtime completes while a new runtime becomes active.
5. Runtime status exposes current version, last error, and route list.

---

## 8. Test and validation matrix

| Layer | Test | What it proves |
|---|---|---|
| `app.HostServices` | configured service visible in module setup | embedding hook reaches provider modules |
| generated package template | rendered `Options` includes `ConfigureServices` | generated package users can configure services |
| generated package smoke | temp host imports generated package and passes service | generated code compiles and the hook works outside unit tests |
| HTTP provider | JS route registers into external `gojahttp.Host` | provider consumes external host service |
| HTTP provider | occupied listen address does not fail external no-listen route registration | provider does not bind when `OwnsListen=false` |
| HTTP provider | no service configured still self-listens | existing generated server behavior preserved |
| gojahttp | route descriptors are copy-safe | introspection does not leak callables or mutable internals |
| runtime manager | broken reload keeps old route response | last-known-good fallback works |
| runtime manager | concurrent old request survives swap | active handle capture and close grace are safe |

Focused commands:

```bash
cd /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja

go test ./pkg/xgoja/app ./pkg/xgoja/providers/http ./pkg/gojahttp ./modules/express -count=1

go test ./cmd/xgoja/internal/generate -count=1
```

Broader pre-PR validation:

```bash
go test ./pkg/xgoja/... ./pkg/gojahttp ./modules/express -count=1
go test ./cmd/xgoja/internal/buildspec ./cmd/xgoja/internal/generate -count=1
go test ./... -count=1
```

---

## 9. Decision records

### Decision: Use current HostServices names for this ticket

- **Context:** We identified confusing naming around `HostServices`, `HostServiceContributionCapability`, and `ModuleSetupContext.Host`. However, several packages already depend on the current provider API.
- **Options considered:** Perform a breaking rename now; add compatibility aliases; keep current names and track a later rename.
- **Decision:** Keep current names in this ticket. Track the breaking rename separately in GitHub issue #65.
- **Rationale:** External HTTP host mode is an additive feature. A broad provider API rename would make the implementation harder to review and could block downstream packages.
- **Consequences:** The guide must be explicit that `ctx.Host` is the service bag. Later cleanup will rename it to a more accurate runtime-service API.
- **Status:** accepted for this ticket.

### Decision: Add `ConfigureServices` to generated package Options

- **Context:** Generated package users need to inject live Go services without editing generated code or custom templates.
- **Options considered:** Require custom templates; expose `HostServices` directly as an `Options` field; add a callback; add a new `NewBundleWithServices` function.
- **Decision:** Add `ConfigureServices func(*app.HostServices)` to `Options` and `HostOptions`.
- **Rationale:** A callback is additive, simple, and works for both generated package and source-fragment generation. It keeps asset initialization in xgoja while allowing the host to add services.
- **Consequences:** The callback should remain small and deterministic. Provider module setup remains responsible for validating service payload types.
- **Status:** proposed.

### Decision: Use typed HTTP service payload

- **Context:** The HTTP provider needs exactly one externally supplied `*gojahttp.Host` for external-host mode.
- **Options considered:** Store the raw `*gojahttp.Host` directly under a key; create a typed payload with listener ownership; configure external mode through xgoja YAML.
- **Decision:** Use `ExternalHostService{Host *gojahttp.Host, OwnsListen bool}` under `go-go-goja-http.host`.
- **Rationale:** The typed payload documents ownership semantics and leaves room for advanced provider-owned-listener use without changing the key.
- **Consequences:** The provider must validate service type and nil host early.
- **Status:** proposed.

### Decision: RuntimeManager starts app-local

- **Context:** A generic manager API is tempting, but hot reload policy depends on app bootstrap, route prefixes, status endpoints, close grace, and smoke tests.
- **Options considered:** Build generic `pkg/xgoja/runtimehost` immediately; build app-local manager first; skip manager and only expose external-host mode.
- **Decision:** Implement external-host support in `go-go-goja`, then prototype RuntimeManager app-local first.
- **Rationale:** The external-host feature is a reusable primitive. The manager should be extracted after one real app proves the hooks.
- **Consequences:** This ticket's manager section is a validation design, not necessarily a go-go-goja package implementation mandate.
- **Status:** proposed.

---

## 10. Risks and mitigations

### Risk: Hidden listener ownership bugs

If external-host mode accidentally binds a listener, it can conflict with the outer Go server.

Mitigation:

- Add an occupied-port test for external mode.
- Store `external` and `ownsListen` on `runtimeEntry`.
- Make `start` explicitly return before any `net.Listen` call when `external && !ownsListen`.

### Risk: Service type errors show up too late

If the host injects the wrong type, errors could appear only when JavaScript requires `express`.

Mitigation:

- Resolve and validate the service in the module factory, not inside route registration.
- Return a clear setup error: `http host service "go-go-goja-http.host" must be ExternalHostService, got T`.

### Risk: Callback cannot return errors

A non-breaking `ConfigureServices func(*HostServices)` cannot return an error through `NewHostWithOptions`.

Mitigation:

- Keep service helpers returning errors for callers who want to handle them inside the callback.
- Keep provider payload validation in runtime construction.
- Revisit an error-returning API only if real integrations need it.

### Risk: Route registry races during reload/status

Route registration happens on runtime setup, while status/debug endpoints may read routes.

Mitigation:

- `Registry.Routes()` must take `RLock` and return a copy.
- Runtime manager should snapshot routes after bootstrap/smoke success.

### Risk: Closing old runtime while requests still use it

If a reload closes the old runtime immediately, in-flight requests can fail.

Mitigation:

- Active handler captures one handle per request.
- Close old runtime after a grace period in the first implementation.
- Consider reference-counted handles later if long requests are common.

---

## 11. Alternatives considered

### Alternative A: Build a custom generated template for hybrid hosts

This would work, but it makes every app own internal xgoja template details. It also duplicates the generated package API. Reject for the first implementation.

### Alternative B: Make JavaScript own the full server

This is the existing generated-server model. It is still valid for simple generated binaries, but it is the wrong ownership boundary for apps with uploads, sessions, data stores, and native Go APIs.

### Alternative C: Expose `gojahttp.Host` through a global provider singleton

This would be simpler in the HTTP provider but unsafe for multiple runtimes and hot reload. Reject because runtime isolation requires per-runtime hosts.

### Alternative D: Rename all HostService APIs first

This would clean up naming but is broad and affects sibling packages. Reject for this ticket; track in issue #65.

---

## 12. File reference map

### xgoja provider API

| File | Relevant lines | Why it matters |
|---|---:|---|
| `pkg/xgoja/providerapi/module.go` | 14-21 | `ModuleSetupContext` carries `Host` into provider module setup. |
| `pkg/xgoja/providerapi/module.go` | 24-37 | `HostServices` and `HostServiceLookup` are the current non-invasive service interfaces. |
| `pkg/xgoja/providerapi/capabilities.go` | 70-98 | Existing provider contribution API; keep unchanged in this ticket. |

### xgoja app/runtime construction

| File | Relevant lines | Why it matters |
|---|---:|---|
| `pkg/xgoja/app/assets.go` | 20-23 | Concrete `HostServices` already has a service map. |
| `pkg/xgoja/app/assets.go` | 56-79 | `HostService` and `HostServiceValues` already read service values. |
| `pkg/xgoja/app/host.go` | 24-30 | `HostOptions` lacks `ConfigureServices`. |
| `pkg/xgoja/app/host.go` | 36-45 | `NewHostWithOptions` creates services and passes them to `NewRuntimeFactory`. |
| `pkg/xgoja/app/factory.go` | 46-54 | Module factories receive `ModuleSetupContext{Host: s.services}`. |
| `pkg/xgoja/app/factory.go` | 143-173 | Provider service contributions are collected before module setup. |

### Generated package templates

| File | Relevant lines | Why it matters |
|---|---:|---|
| `cmd/xgoja/internal/generate/templates/runtime_package.go.tmpl` | 43-46 | Generated package `Options` lacks service injection. |
| `cmd/xgoja/internal/generate/templates/runtime_package.go.tmpl` | 83-95 | Generated package passes host options into `app.NewHostWithOptions`. |
| `cmd/xgoja/internal/generate/templates/bundle_fragment.go.tmpl` | 18-21 | Source-fragment `Options` also lacks service injection. |
| `cmd/xgoja/internal/generate/templates/bundle_fragment.go.tmpl` | 38-50 | Source-fragment `NewBundle` must pass the hook too. |

### HTTP/Express bridge

| File | Relevant lines | Why it matters |
|---|---:|---|
| `pkg/xgoja/providers/http/http.go` | 23-45 | HTTP provider registers `express` and `serve`. |
| `pkg/xgoja/providers/http/http.go` | 53-58 | `runtimeEntry` owns host/server state and needs ownership flags. |
| `pkg/xgoja/providers/http/http.go` | 109-122 | Current express loader creates an internal host if needed. |
| `pkg/xgoja/providers/http/http.go` | 136-162 | Current start path binds TCP listener. |
| `modules/express/express.go` | 27-35 | Low-level Express registrar already accepts external host. |
| `modules/express/express.go` | 68-78 | Express loader binds runtime owner into the host. |
| `modules/express/express.go` | 132-188 | Route/static registration calls start hook and registers into host. |

### gojahttp

| File | Relevant lines | Why it matters |
|---|---:|---|
| `pkg/gojahttp/host.go` | 35-40 | Constructs host and registers routes. |
| `pkg/gojahttp/host.go` | 88-146 | `ServeHTTP` dispatches requests into JavaScript through runtime owner. |
| `pkg/gojahttp/route_registry.go` | 21-44 | Registry supports add/match but lacks introspection. |

---

## 13. New intern checklist

Before coding:

1. Read `pkg/xgoja/providerapi/module.go` and understand `ModuleSetupContext`.
2. Read `pkg/xgoja/app/host.go`, `assets.go`, `host_services.go`, and `factory.go` to understand where services are created, contributed, and passed into module factories.
3. Read `pkg/xgoja/providers/http/http.go` and identify where the provider currently creates `gojahttp.Host` and starts listeners.
4. Read `modules/express/express.go` and confirm the raw module already accepts an external `gojahttp.Host`.
5. Read `pkg/gojahttp/host.go` and `route_registry.go` to understand how HTTP requests reach JS handlers.
6. Remember that issue #65 is deferred; do not rename public provider APIs in this ticket.

Coding order:

1. Add host service helper methods and `ConfigureServices`.
2. Update generated package and bundle-fragment templates.
3. Add HTTP external-host service and provider behavior.
4. Add route introspection.
5. Add tests in the same order.
6. Only then prototype runtime-manager behavior in an app or a narrow test harness.

Definition of done:

- A generated package caller can inject `ExternalHostService` through `NewBundle(Options{ConfigureServices: ...})`.
- JS route registration through `require("express")` registers into the supplied `gojahttp.Host`.
- External no-listen mode does not bind the provider listener.
- Existing self-listening HTTP provider behavior still works when no external host service is configured.
- Route introspection returns method/pattern descriptors without exposing callables.
- Focused and generated-package tests pass.

---

## 14. References

- Future breaking rename tracking issue: https://github.com/go-go-golems/go-go-goja/issues/65
- Broad hybrid host design: `ClubMedMeetup/ttmp/2026/06/08/xgoja-modules-improvement--improve-xgoja-goja-modules-based-on-clubmedmeetup-usage-patterns/design-doc/03-hybrid-go-http-host-and-xgoja-runtime-manager-design.md`
- Ecosystem improvement roadmap: `ClubMedMeetup/ttmp/2026/06/08/xgoja-modules-improvement--improve-xgoja-goja-modules-based-on-clubmedmeetup-usage-patterns/design-doc/04-goja-ecosystem-improvement-guide-after-minitrace-viz.md`
- Generated package implementation ticket: `go-go-goja/ttmp/2026/06/04/GOJA-065--flexible-xgoja-code-generation-for-embedding-runtimes-in-existing-applications`
- HTTP serve support ticket: `go-go-goja/ttmp/2026/06/04/GOJA-064--http-serve-support-for-xgoja-generated-verbs`
- Runtime polish ticket: `go-go-goja/ttmp/2026/06/08/XGOJA-RUNTIME-POLISH--polish-xgoja-runtime-ergonomics-for-express-fs-assets-and-module-inventory`
