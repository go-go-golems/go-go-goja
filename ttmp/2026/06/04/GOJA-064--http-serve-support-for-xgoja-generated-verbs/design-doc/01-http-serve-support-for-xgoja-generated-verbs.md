---
Title: HTTP Serve Support for xgoja Generated Verbs
Ticket: GOJA-064
Status: active
Topics:
    - goja
    - xgoja
    - http
    - verbs
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../../code/wesen/2026-05-03--goja-hosting-site/pkg/app/server.go
      Note: External reference for single-site goja runtime plus HTTP server lifecycle
    - Path: cmd/xgoja/internal/generate/generate_test.go
      Note: Generated-binary HTTP serve verb smoke test
    - Path: examples/xgoja/13-http-serve-jsverbs/verbs/sites.js
      Note: Example JavaScript site setup verb
    - Path: examples/xgoja/13-http-serve-jsverbs/xgoja.yaml
      Note: Runnable xgoja buildspec for HTTP serve jsverbs example
    - Path: modules/express/express.go
      Note: JavaScript express.app API and route/static registration surface
    - Path: pkg/xgoja/app/command_providers.go
      Note: |-
        Provider command set mounting and CommandSetContext construction
        Passes jsverb source set into command provider context
    - Path: pkg/xgoja/app/jsverb_sources.go
      Note: New app-side jsverb source scanner exposed to command providers
    - Path: pkg/xgoja/app/root.go
      Note: Existing generated root and jsverbs command construction that currently closes runtimes after invocation
    - Path: pkg/xgoja/app/run.go
      Note: Existing keep-alive runtime lifecycle to model for serve
    - Path: pkg/xgoja/providerapi/commands.go
      Note: |-
        Command provider API that needs jsverb source access for HTTP serve
        Added JSVerbSourceSet to CommandSetContext for serve provider implementation
    - Path: pkg/xgoja/providers/http/http.go
      Note: |-
        HTTP provider express module
        Registers serve command provider with HTTP package
    - Path: pkg/xgoja/providers/http/serve.go
      Note: New HTTP serve command provider implementation
ExternalSources: []
Summary: Design and implementation guide for serving HTTP sites from xgoja-generated JavaScript verbs using the express/gojahttp provider stack.
LastUpdated: 2026-06-04T22:43:00-04:00
WhatFor: Use when implementing or reviewing first-class HTTP serve support for generated xgoja verb commands.
WhenToUse: Before changing pkg/xgoja, pkg/jsverbs, modules/express, pkg/gojahttp, or the go-go-goja HTTP provider.
---




# HTTP Serve Support for xgoja Generated Verbs

## Executive summary

Generated xgoja binaries already have most of the building blocks needed to serve HTTP sites from JavaScript: a single generated runtime module list, provider modules, JavaScript verb scanning, an Express-style module, a `gojahttp.Host`, and a `run --keep-alive` command. The missing product shape is a first-class generated command that treats a JavaScript verb as a **site setup function**: build one long-lived runtime, invoke the selected verb once so it registers Express routes, keep the runtime and HTTP server alive, and shut everything down on Ctrl-C/SIGTERM.

The recommended design is a provider-backed `serve` command set supplied by `pkg/xgoja/providers/http`. Generated applications opt into it through `commandProviders`, and the command provider dynamically mirrors configured JavaScript verbs under a `serve` command tree:

```text
./my-app serve <package-path> <verb-name> [verb flags] --http-listen 127.0.0.1:8787
```

This keeps HTTP-specific command behavior close to the HTTP provider while reusing the same xgoja single-runtime and jsverbs machinery as ordinary generated verb commands. It also avoids the most tempting but weaker design, `express.serve()` alone, which would make JavaScript block manually and would not give generated binaries a discoverable CLI command shape.

### Update: single-runtime xgoja.yaml simplification

This plan now assumes the newer xgoja schema where `xgoja.yaml` has one top-level `modules:` list and no `runtimes:` map. That simplifies the HTTP serve design substantially:

- `serve` no longer has to choose or validate a runtime profile.
- `commandProviders[].runtimeProfile` should not be used in examples or new implementation work.
- `RuntimeFactory.NewRuntimeFromSections(ctx, vals, ...)` can create the one configured runtime directly.
- Module sections such as `--http-listen` are collected from the single configured module list.
- Future multi-site serving still means multiple **runtime instances**, but not multiple YAML runtime profiles. Each site instance should use the same generated module set unless a later schema introduces per-site module overrides.

For a new intern, the short version is:

1. `xgoja.yaml` describes provider packages, one generated runtime's module list, generated commands, JavaScript verb sources, and command providers.
2. xgoja generates a normal Go binary that embeds a normalized `app.RuntimeSpec`.
3. At startup, the generated root command builds an `app.Host`, attaches built-in commands, and attaches provider-owned command sets.
4. The existing `verbs` command scans JavaScript files and creates one Cobra/Glazed command per `__verb__` declaration.
5. The existing HTTP provider exposes `require("express")`, which creates `express.app()` and backs it with `gojahttp.Host`.
6. A new HTTP `serve` command provider should scan the same verb sources, create one long-lived runtime per selected site, invoke the selected verb once, and wait for shutdown instead of closing the runtime immediately.

## Problem statement and scope

### User-visible problem

Today, an xgoja-generated binary can run ordinary JavaScript verbs as short-lived CLI commands. It can also run an HTTP setup script through the built-in `run` command with `--keep-alive`:

```bash
./generated-app run scripts/server.js --http-listen 127.0.0.1:8787 --keep-alive
```

That works for script files, but it is not a good fit for generated JavaScript verb repositories. If a verb registers routes through `require("express")`, the runtime is closed when the verb invocation returns. The server either shuts down immediately or loses the runtime owner needed by request handlers. What users want is closer to:

```bash
./generated-app serve sites kanban --http-listen 127.0.0.1:8787
```

where `sites kanban` is an ordinary discovered JavaScript verb whose body registers routes.

### In scope

This design covers:

- serving one JavaScript verb-backed site from one generated xgoja process;
- integrating with the configured xgoja runtime module list and module sections;
- preserving existing `run`, `eval`, `repl`, and `verbs` behavior;
- making `go-go-goja-http` provide an opt-in `serve` command provider;
- preparing an explicit path for future multi-site serving, similar to `goja-site serve-multi`.

### Out of scope for the first implementation

The first implementation should not try to become the full `goja-site` application framework. In particular, it should not immediately implement:

- database policy management from `goja-site`;
- production observability and tracing wrappers from `goja-site`;
- automatic virtual-host dispatch across many separately-created site runtimes;
- hot reload;
- untrusted JavaScript sandboxing.

Those are follow-up layers. The first milestone is to keep the runtime alive and make route-registration verbs serve real HTTP requests.

## Current-state architecture

### xgoja generated root commands

`pkg/xgoja/app/root.go` is the generated application runtime entry point. `NewRootCommand` decodes embedded runtime JSON, constructs an `app.Host`, creates a Cobra root, and asks the host to attach default commands (`pkg/xgoja/app/root.go:36-53`). The host then installs the root framework and attaches enabled command families in a stable order: `eval`, `run`, `repl`, `modules`, `verbs`, and provider command sets (`pkg/xgoja/app/host.go:55-76`).

The important consequence is that a provider command set is already a first-class extension point. A provider does not need xgoja code generation changes to add a command, as long as the generated binary imports the provider and the buildspec includes `commandProviders:`.

```text
xgoja.yaml
  -> generated main.go imports providers
  -> providerapi.ProviderRegistry
  -> app.NewRootCommand(...)
  -> Host.AttachDefaultCommands(root)
       ├─ built-in run/eval/repl/modules/verbs
       └─ provider command sets
```

### Buildspec and runtime spec command-provider schema

At build time, `cmd/xgoja/internal/buildspec/build_spec.go` defines the declarative YAML schema. The current simplified schema has a top-level `modules:` list instead of a `runtimes:` map, and `CommandProviderInstanceSpec` carries `id`, `package`, `name`, `mount`, optional module filtering, static config, and a `lazy` flag (`cmd/xgoja/internal/buildspec/build_spec.go:16-29`, `cmd/xgoja/internal/buildspec/build_spec.go:100-109`). At runtime, the generated binary uses the reduced `app.RuntimeSpec`, which keeps the same single-runtime `modules` and command-provider fields (`pkg/xgoja/app/runtime_spec.go:15-27`, `pkg/xgoja/app/runtime_spec.go:76-87`).

Validation now rejects/flags legacy command-provider `runtimeProfile` fields and validates command-provider IDs, package IDs, and provider names against the single-runtime buildspec. That means an HTTP serve command provider can be introduced without inventing a new top-level command schema, as long as users opt in:

```yaml
commandProviders:
  - id: http-serve
    package: go-go-goja-http
    name: serve
    mount: serve
```

### How provider command sets are attached

`pkg/xgoja/app/command_providers.go` resolves each configured provider by `(package, name)`, applies the configured mount, creates a `providerapi.CommandSetContext`, and adds returned Glazed commands to Cobra (`pkg/xgoja/app/command_providers.go:19-56`). The context still includes a compatibility `RuntimeProfile` value (`main`), static provider config, selected runtime modules, provider registry, and an xgoja `RuntimeFactory` (`pkg/xgoja/app/command_providers.go:59-86`; `pkg/xgoja/providerapi/commands.go:24-37`).

The context is almost sufficient for HTTP serve support. The missing piece is access to configured JavaScript verb sources. Built-in jsverbs support can scan `runtimeSpec.JSVerbs`, but provider command sets currently cannot.

### Current JavaScript verb command flow

Built-in `verbs` support lives in `pkg/xgoja/app/root.go`. It does three things:

1. Use the single generated runtime module list for `commands.jsverbs`.
2. Scan configured JavaScript verb sources.
3. For every discovered `VerbSpec`, build a Glazed command whose invoker creates a fresh runtime, invokes the verb, then closes the runtime.

The core invocation is visible in `buildVerbCommands`: the invoker calls `factory.NewRuntimeFromSections(...)` with the registry's source loader, defers `rt.Close(...)`, initializes selected module capabilities, and calls `registry.InvokeInRuntime(...)` (`pkg/xgoja/app/root.go:233-275`, especially `pkg/xgoja/app/root.go:251-263`).

That `defer rt.Close(...)` is correct for ordinary CLI verbs. It is wrong for a route-registration verb that should continue serving requests after the setup function returns.

### Current `run --keep-alive` flow

The `run` command already models the lifetime behavior that HTTP serving needs. It accepts `--keep-alive`, creates a runtime, runs a JavaScript file as a module, and then waits for Ctrl-C/SIGTERM before closing the runtime (`pkg/xgoja/app/run.go:35-59`, `pkg/xgoja/app/run.go:87-129`).

This is the closest existing implementation to copy. The serve command should behave like `run --keep-alive`, except the startup unit is a selected jsverb rather than a file path.

### HTTP provider and Express module

The first-party HTTP provider is `pkg/xgoja/providers/http`. Its `Register` function exposes one provider package, `go-go-goja-http`, with one module named `express` and one package capability (`pkg/xgoja/providers/http/http.go:22-36`). That capability contributes a public Glazed `http` section with `--http-enabled` and `--http-listen` flags (`pkg/xgoja/providers/http/http.go:62-75`). Runtime initialization stores the parsed settings per Goja runtime and registers a closer (`pkg/xgoja/providers/http/http.go:78-97`).

The provider's loader lazily creates a `gojahttp.Host`, starts an HTTP server, and then installs the Express module loader (`pkg/xgoja/providers/http/http.go:100-114`). The server start path creates `net/http.Server{Addr: cfg.Listen, Handler: entry.host}` and runs `ListenAndServe` in a goroutine (`pkg/xgoja/providers/http/http.go:128-150`). Shutdown is tied to runtime close (`pkg/xgoja/providers/http/http.go:153-168`).

`modules/express` is the JavaScript-facing API. It exports `app()`, and each app object supports HTTP method registration, host-directory static mounts, and embedded-asset static mounts (`modules/express/express.go:89-125`):

```js
const express = require("express")
const app = express.app()
app.get("/", (_req, res) => res.send("hello"))
app.static("/assets", "./assets")
app.staticFromAssetsModule("/static", require("fs:assets"), "/app/public")
```

### gojahttp request dispatch

`pkg/gojahttp/host.go` is the bridge from Go `net/http` to JavaScript route callbacks. `ServeHTTP` checks static mounts, matches dynamic routes, creates request/response DTOs, calls the registered Goja handler through the runtime owner, handles promises by polling, and writes errors if necessary (`pkg/gojahttp/host.go:51-105`).

The runtime owner boundary is important: Goja is not goroutine-safe. HTTP request goroutines must call JavaScript through `h.owner.Call(...)`, not directly.

### goja-site as the closest larger example

The external example at `/home/manuel/code/wesen/2026-05-03--goja-hosting-site` shows the full application shape that xgoja should learn from.

`cmd/goja-site/serve.go` exposes a Glazed `serve` command with flags for bind address, scripts, DB policy, diagnostics, and tracing. It creates a cancellable signal context, constructs `app.NewServer`, and runs it until shutdown (`cmd/goja-site/serve.go:38-69`, `cmd/goja-site/serve.go:72-120`).

`pkg/app/server.go` owns the single-site runtime. It opens SQLite, creates `gojahttp.Host`, registers `express`, UI DSL, Kanban DSL, and database modules, creates one runtime, sets the host runtime owner, loads scripts, and then serves the host through `net/http.Server` (`pkg/app/server.go:32-93`, `pkg/app/server.go:111-150`). `LoadScripts` reads all `.js` files and runs each script on the runtime owner (`pkg/app/server.go:153-173`). `pkg/app/scripts.go` walks script directories and returns sorted `.js` files (`pkg/app/scripts.go:11-70`).

`pkg/app/multi_server.go` shows the multi-site pattern: create one isolated `Server` per configured host, then have an outer `ServeHTTP` dispatch by normalized Host header (`pkg/app/multi_server.go:19-38`, `pkg/app/multi_server.go:63-89`). This is the right mental model for future xgoja `serve-multi`, but it should not be required for the first single-site implementation.

## Gap analysis

### What already works

- The top-level `modules:` list can select the `go-go-goja-http.express` module.
- The HTTP provider exposes `--http-listen` as a module section.
- `run --keep-alive` can serve scripts that register Express routes.
- `commands.jsverbs` can turn `__verb__` declarations into generated commands without choosing among runtime profiles.
- Provider command sets can mount package-owned Glazed commands.

### What is missing

1. **Provider command sets cannot scan configured jsverb sources.** Built-in jsverbs code can call `scanVerbSource(...)`, but `providerapi.CommandSetContext` does not expose those sources or a registry scanner.
2. **Built-in jsverbs commands intentionally close the runtime.** That is correct for normal verbs but incompatible with HTTP serving.
3. **The HTTP provider starts a server as a side effect of `require("express")`.** That is convenient for `run --keep-alive`, but it is too implicit for future multi-site serving where one outer server should dispatch to multiple route hosts.
4. **There is no generated command shape for serving a verb.** Users must know to use `run --keep-alive` with setup scripts, which bypasses the verb repository/discovery model.

## Proposed architecture

### Recommended user model

A site setup verb is an ordinary JavaScript verb that registers routes as side effects. The verb body may still accept fields, read config, mount assets, and initialize state. The command provider keeps the runtime alive after that setup call returns.

```js
__package__({ name: "sites", short: "HTTP sites" })

__verb__("staticSite", {
  short: "Serve the embedded static demo site",
  tags: ["http", "site"]
}, function staticSite() {
  const express = require("express")
  const assets = require("fs:assets")

  const app = express.app()
  app.staticFromAssetsModule("/static", assets, "/app/public")
  app.get("/", (_req, res) => res.redirect("/static/"))
  app.get("/api/health", (_req, res) => res.json({ ok: true }))

  console.log("site routes registered")
})
```

The generated app opts into the command provider:

```yaml
packages:
  - id: go-go-goja-host
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/host
  - id: go-go-goja-http
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/http

modules:
  - package: go-go-goja-host
    name: fs
    as: fs:assets
    config:
      embedded:
        allow: true
        mounts:
          - asset: app-assets
            mount: /app
  - package: go-go-goja-http
    name: express

commands:
  jsverbs:
    enabled: true

commandProviders:
  - id: http-serve
    package: go-go-goja-http
    name: serve
    mount: serve
```

The generated command tree becomes:

```bash
./my-app verbs sites static-site --help
./my-app serve sites static-site --http-listen 127.0.0.1:8787
```

The ordinary `verbs` command remains short-lived. The `serve` command is long-lived.

### High-level runtime flow

```text
User runs: ./my-app serve sites static-site --http-listen 127.0.0.1:8787

Cobra/Glazed parses:
  - verb-owned fields
  - provider module sections, including http.listen

HTTP serve command invoker:
  1. scan configured jsverb source(s)
  2. create the generated runtime with registry.RequireLoader()
  3. initialize selected module runtime capabilities from parsed Glazed values
  4. invoke selected verb once
  5. verb calls require("express") and registers routes
  6. HTTP provider starts or exposes the gojahttp.Host
  7. command waits for signal/context cancellation
  8. runtime close triggers HTTP shutdown and provider closers
```

### Required provider API extension

The HTTP provider command set needs a supported way to discover the same JavaScript verb sources that built-in `commands.jsverbs` uses. It no longer needs to choose a runtime profile; it should use the one generated runtime configuration. Add a small jsverb source accessor to `providerapi.CommandSetContext`.

Proposed API sketch:

```go
// pkg/xgoja/providerapi/commands.go

type JSVerbSourceDescriptor struct {
    ID      string
    Path    string
    Embed   bool
    Package string
    Source  string
}

type JSVerbSourceSet interface {
    ListJSVerbSources() []JSVerbSourceDescriptor
    ScanJSVerbSource(id string) (*jsverbs.Registry, error)
    ScanAllJSVerbSources() ([]*jsverbs.Registry, error)
}

type CommandSetContext struct {
    Context         context.Context
    PackageID       string
    Name            string
    Mount           string
    RuntimeProfile  string // compatibility value, normally "main" in the single-runtime schema
    Config          json.RawMessage
    Host            HostServices
    Providers       *ProviderRegistry
    RuntimeFactory  RuntimeFactory
    SelectedModules []ModuleDescriptor

    // New: lets command providers build commands from configured JS verbs.
    JSVerbs JSVerbSourceSet
}
```

If maintainers want to avoid importing `pkg/jsverbs` from `providerapi`, use a narrower function-based interface in `app` and expose it through a small package that both `app` and `providers/http` can import. The key requirement is the same: command providers must not re-implement local/embed/provider verb source resolution.

### HTTP provider command provider

Add a command set provider in `pkg/xgoja/providers/http.Register`:

```go
func Register(registry *providerapi.ProviderRegistry) error {
    capability := newHTTPCapability()
    return registry.Package(PackageID,
        providerapi.Module{... express ...},
        providerapi.WithPackageCapability(capability),
        providerapi.CommandSetProvider{
            Name:         "serve",
            DefaultMount: "serve",
            Description:  "Serve JavaScript verb-backed HTTP sites",
            NewCommandSet: func(ctx providerapi.CommandSetContext) (*providerapi.CommandSet, error) {
                return newServeCommandSet(ctx, capability)
            },
        },
    )
}
```

The command set should mirror discovered verbs, similar to built-in `commands.jsverbs`, but with a different invoker lifetime:

```go
func newServeCommandSet(ctx providerapi.CommandSetContext, capability *capability) (*providerapi.CommandSet, error) {
    if ctx.JSVerbs == nil {
        return nil, fmt.Errorf("http serve command requires configured jsverb sources")
    }

    sections, err := providerutil.CollectGlazedConfigSections(
        ctx.SelectedModules,
        providerapi.SectionRequest{
            CommandProviderID: ctx.Name,
            RuntimeProfile:    "main",
        },
        nil,
    )
    if err != nil { return nil, err }

    var commands []cmds.Command
    for _, registry := range ctx.JSVerbs.ScanAllJSVerbSources() {
        for _, verb := range registry.Verbs() {
            cmd, err := registry.CommandForVerbWithInvoker(verb,
                func(runCtx context.Context, _ *jsverbs.Registry, verb *jsverbs.VerbSpec, vals *values.Values) (any, error) {
                    return serveVerb(runCtx, ctx, registry, verb, vals)
                })
            if err != nil { return nil, err }
            appendHTTPSections(cmd.Description(), sections)
            commands = append(commands, cmd)
        }
    }
    return &providerapi.CommandSet{Commands: commands}, nil
}
```

The lifetime-aware invoker is the crucial behavior change:

```go
func serveVerb(ctx context.Context, c providerapi.CommandSetContext, registry *jsverbs.Registry, verb *jsverbs.VerbSpec, vals *values.Values) (any, error) {
    rt, err := c.RuntimeFactory.NewRuntimeFromSections(ctx, vals, require.WithLoader(registry.RequireLoader()))
    if err != nil { return nil, err }
    defer func() { _ = rt.Close(context.Background()) }()

    if err := providerutil.InitRuntimeFromSections(ctx, vals, runtimeHandle{rt}, c.SelectedModules); err != nil {
        return nil, err
    }

    if _, err := registry.InvokeInRuntime(ctx, rt, verb, vals); err != nil {
        return nil, err
    }

    fmt.Fprintf(os.Stderr, "xgoja http serve: runtime is alive; press Ctrl-C to stop\n")
    return nil, waitForSignalOrContext(ctx)
}
```

This is intentionally close to `run --keep-alive` in `pkg/xgoja/app/run.go`. The only difference is the startup unit.

### Single-site first, multi-site second

The first version can rely on the existing HTTP provider auto-start behavior: requiring `express` starts the HTTP server using the configured `--http-listen`. This keeps the initial change small and validates the user model quickly.

For multi-site, the HTTP provider needs one additional abstraction: server ownership mode.

```text
Mode auto:
  require("express") starts one net/http.Server for this runtime.
  Good for run --keep-alive and single-site serve.

Mode manual:
  require("express") creates/registers routes on a gojahttp.Host, but does not listen.
  A command provider owns the outer net/http.Server.
  Required for serve-multi.

Mode disabled:
  route host can exist for tests, but no listener starts.
```

Future `serve-multi` should follow the `goja-site` pattern:

```text
serve-multi config
  -> for each site:
       create isolated runtime
       invoke selected site setup verb
       collect that runtime's gojahttp.Host
  -> start one outer net/http.Server
  -> dispatch requests by Host header to the selected site host
```

Pseudocode:

```go
type MultiSite struct {
    Name string
    Host string
    VerbPath string
    Values map[string]any
}

func serveMulti(ctx context.Context, cfg MultiConfig) error {
    sites := map[string]*siteRuntime{}
    for _, site := range cfg.Sites {
        rt, host, err := createRuntimeAndRegisterRoutes(site)
        if err != nil { cleanup(); return err }
        sites[normalizeHost(site.Host)] = &siteRuntime{Runtime: rt, Host: host}
    }

    srv := &http.Server{
        Addr: cfg.Listen,
        Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            site := sites[normalizeHost(r.Host)]
            if site == nil { http.NotFound(w, r); return }
            site.Host.ServeHTTP(w, r)
        }),
    }
    return runUntilSignal(ctx, srv, sites)
}
```

Do not build multi-site first. The single-site command will surface most integration bugs at much lower complexity.

## Decision records

### Decision: Provide `serve` as an HTTP provider command set

- **Context:** The command is HTTP-specific but must participate in xgoja's single runtime module list, module sections, and JavaScript verb discovery.
- **Options considered:** Add `express.serve()` only; add a new built-in `commands.serve`; add an HTTP provider `CommandSetProvider`.
- **Decision:** Add an opt-in `CommandSetProvider{Name: "serve"}` to `go-go-goja-http` and extend command-provider context with jsverb source access.
- **Rationale:** Provider command sets already exist for package-owned commands. The HTTP provider owns `--http-listen`, server lifetime, and Express host behavior. Keeping `serve` in the provider avoids growing xgoja core with HTTP-only command semantics.
- **Consequences:** `providerapi.CommandSetContext` needs a jsverb source accessor. Users must opt in with `commandProviders:`. Future non-HTTP serve-like providers can use the same jsverb-source accessor.
- **Status:** proposed.

### Decision: Site setup verbs register routes as side effects

- **Context:** Express route registration already works through side effects on `gojahttp.Host`.
- **Options considered:** Require verbs to return an `http.Handler`; require verbs to return a declarative route table; keep side-effect registration.
- **Decision:** Preserve side-effect route registration for v1.
- **Rationale:** It matches existing `express.app()` behavior, `goja-site`, and the static-assets tutorial. It keeps the JavaScript API small and avoids introducing a second routing representation.
- **Consequences:** The serve command must invoke the verb exactly once at startup and keep the runtime alive. Route registration errors must fail startup early.
- **Status:** proposed.

### Decision: Reuse jsverbs command generation rather than invent a new site registry

- **Context:** jsverbs already parse metadata, fields, package parents, and function bindings.
- **Options considered:** Add `__site__`; add a new YAML site registry; reuse all verbs and optionally filter by tags.
- **Decision:** Reuse jsverbs and initially expose all configured verbs under `serve`; optionally add filtering by tag or metadata later.
- **Rationale:** Reuse gives users the same command paths and flags under `verbs` and `serve`. It avoids teaching another declaration API.
- **Consequences:** Some non-site verbs may appear under `serve` unless the first implementation adds a filter such as `tags: ["http", "site"]`. If this is noisy, add `commandProviders[].config.tags` in v2.
- **Status:** proposed.

### Decision: Keep `express.serve()` out of the critical path

- **Context:** A JavaScript-level `express.serve()` method is attractive because it is easy to explain, but it hides lifecycle inside JS and does not create generated CLI commands.
- **Options considered:** Implement only `express.serve()`; implement command provider only; implement both.
- **Decision:** Implement command provider first. Consider a small `express.listen()`/`express.serve()` convenience later only as a wrapper for scripts.
- **Rationale:** Command-provider serve can parse CLI/config/env fields, reuse generated help, handle signals in Go, and close runtime resources correctly.
- **Consequences:** `run --keep-alive` remains the script-file path. The first-class verb-serving path is `commandProviders: go-go-goja-http.serve`.
- **Status:** proposed.

### Decision: Postpone true multi-site until server ownership is explicit

- **Context:** The HTTP provider currently starts one server per runtime when `express` is required. Multi-site needs many runtime hosts behind one listener.
- **Options considered:** Start one port per site; share current auto-start host; introduce manual server ownership.
- **Decision:** Ship single-site first; refactor HTTP provider to support manual host/server ownership before `serve-multi`.
- **Rationale:** Reusing auto-start is sufficient for one site. Multi-site without explicit ownership would fight port conflicts and make shutdown hard.
- **Consequences:** The first command can serve different sites one at a time. Serving many sites in one process is a follow-up phase.
- **Status:** proposed.

## Implementation guide

### Phase 1: Expose configured jsverb sources to command providers

Files to start with:

- `pkg/xgoja/providerapi/commands.go`
- `pkg/xgoja/app/command_providers.go`
- `pkg/xgoja/app/root.go`
- `pkg/xgoja/app/runtime_spec.go`

Implementation steps:

1. Add a `JSVerbSourceSet` interface to provider command context.
2. Implement the interface in `pkg/xgoja/app`, using the existing `scanVerbSource(...)` logic.
3. Populate `CommandSetContext.JSVerbs` in `Host.newCommandSet(...)`.
4. Add unit tests that a test command provider can see and scan configured local, embedded, and provider-shipped sources.

Suggested helper shape:

```go
type jsVerbSourceSet struct {
    providers       *providerapi.ProviderRegistry
    embeddedJSVerbs fs.FS
    sources         []JSVerbSourceSpec
}

func (s *jsVerbSourceSet) ListJSVerbSources() []providerapi.JSVerbSourceDescriptor { ... }
func (s *jsVerbSourceSet) ScanJSVerbSource(id string) (*jsverbs.Registry, error) { ... }
func (s *jsVerbSourceSet) ScanAllJSVerbSources() ([]*jsverbs.Registry, error) { ... }
```

Keep the scanning implementation single-sourced. If built-in `buildVerbCommands` and command-provider serve scan differently, generated binaries will behave inconsistently.

### Phase 2: Add `go-go-goja-http.serve` command provider

Files to start with:

- `pkg/xgoja/providers/http/http.go`
- new `pkg/xgoja/providers/http/serve.go`
- new `pkg/xgoja/providers/http/serve_test.go`

Implementation steps:

1. Register `providerapi.CommandSetProvider{Name: "serve", DefaultMount: "serve"}` from `Register`.
2. In `newServeCommandSet`, scan all configured jsverb sources.
3. For each verb, build a command with `registry.CommandForVerbWithInvoker`.
4. Add module-provided Glazed sections, especially the HTTP section.
5. In the invoker, create a runtime, initialize module capabilities, invoke the selected verb, then wait for signal/context cancellation.
6. Close the runtime after wait returns.

The serve invoker must not return immediately after route registration. That is the entire feature.

### Phase 3: Add an xgoja example and smoke test

Create a new example:

```text
examples/xgoja/13-http-serve-jsverbs/
  xgoja.yaml
  verbs/sites.js
  assets/public/index.html
  Makefile
  README.md
```

Example `verbs/sites.js`:

```js
__package__({ name: "sites", short: "Serve demo sites" })

__verb__("static", {
  short: "Serve embedded static assets",
  tags: ["http", "site"]
}, function static() {
  const express = require("express")
  const assets = require("fs:assets")
  const app = express.app()
  app.staticFromAssetsModule("/static", assets, "/app/public")
  app.get("/", (_req, res) => res.redirect("/static/"))
  app.get("/api/health", (_req, res) => res.json({ ok: true }))
})
```

Smoke target:

```make
serve-smoke: build
	log=$$(mktemp); \
	$(BIN) serve sites static --http-listen 127.0.0.1:18787 >$$log 2>&1 & \
	pid=$$!; \
	trap 'kill $$pid >/dev/null 2>&1 || true; wait $$pid >/dev/null 2>&1 || true; rm -f $$log' EXIT; \
	for i in $$(seq 1 50); do \
		curl -fsS http://127.0.0.1:18787/api/health && break; \
		sleep 0.1; \
	done; \
	curl -fsS http://127.0.0.1:18787/static/ | grep -q 'Demo'; \
	kill $$pid >/dev/null 2>&1 || true
```

This test should prove the generated path, not only package-level helpers.

### Phase 4: Refine lifecycle and errors

Add tests and behavior for:

- route registration errors fail before the command starts waiting;
- port conflicts return a useful error instead of only printing from a goroutine;
- Ctrl-C/SIGTERM shuts down the HTTP server and closes the runtime;
- promises returned by setup verbs are awaited before serving is considered ready;
- `--http-enabled=false` produces a clear error for `serve` unless explicitly allowed for dry-run.

The current HTTP provider starts `ListenAndServe` in a goroutine and logs failures asynchronously. For a serve command, startup should be observable. Prefer `net.Listen` first, then `Serve(listener)`:

```go
ln, err := net.Listen("tcp", cfg.Listen)
if err != nil { return err }
server := &http.Server{Handler: host}
go server.Serve(ln)
```

That lets the command fail immediately on bind errors.

### Phase 5: Prepare manual server ownership for multi-site

Refactor the HTTP provider only after single-site is tested.

Add provider config:

```yaml
config:
  serverMode: auto # auto | manual | disabled
```

or public flags:

```text
--http-server-mode auto|manual|disabled
```

Then expose a typed host service so command providers can collect the `gojahttp.Host` for each runtime without starting one listener per runtime.

```go
const HostServiceKey = "go-go-goja-http.host"

type HostService struct {
    Host   *gojahttp.Host
    Listen string
    Mode   ServerMode
}
```

A future `serve-multi` command should create one runtime per site, invoke each site's setup verb, collect each `Host`, and start one outer `net/http.Server` that dispatches by Host header.

## Testing and validation strategy

### Unit tests

Add tests near the code being changed:

- `pkg/xgoja/app`: command-provider context receives jsverb source accessors.
- `pkg/xgoja/providers/http`: `Register` exposes both `express` module and `serve` command provider.
- `pkg/xgoja/providers/http`: serve command set mirrors discovered verb commands.
- `pkg/xgoja/providers/http`: serve invoker keeps runtime open until context cancellation.
- `pkg/xgoja/providers/http`: HTTP section defaults and overrides propagate into the served runtime.

### Generated-binary smoke tests

Generated tests are mandatory because xgoja features can pass package tests but fail in the generated source path. Add a test similar to existing generated-program tests in `cmd/xgoja/internal/generate/generate_test.go`:

```go
func TestGeneratedProgramServesHTTPVerb(t *testing.T) {
    buildSpec := httpServeVerbSpec(t)
    dir, bin := buildGenerated(t, buildSpec)
    cmd := exec.Command(bin, "serve", "sites", "static", "--http-listen", addr)
    startAndProbe(t, cmd, "http://"+addr+"/api/health")
}
```

### Manual validation commands

From the go-go-goja repo:

```bash
GOWORK=off go test ./pkg/xgoja/app ./pkg/xgoja/providers/http ./pkg/jsverbs -count=1
GOWORK=off go test ./cmd/xgoja/internal/generate -run GeneratedProgramServesHTTPVerb -count=1
make -C examples/xgoja/13-http-serve-jsverbs serve-smoke
```

### Regression checks

Run existing HTTP and xgoja tests:

```bash
GOWORK=off go test ./modules/express ./pkg/gojahttp ./pkg/xgoja/providers/http -count=1
GOWORK=off go test ./cmd/xgoja/internal/buildspec ./cmd/xgoja/internal/generate -count=1
```

## Risks and sharp edges

- **Runtime ownership:** HTTP handlers must always call JavaScript through `runtime.Owner.Call`, as `gojahttp.Host` does today.
- **Premature runtime close:** The serve invoker must defer close until after signal wait, not after setup invocation.
- **Bind errors:** Current async server start can hide port conflicts. Serve commands should make bind failures synchronous.
- **Duplicate HTTP sections:** Serve commands will combine verb-owned sections and module sections. Section slugs must remain unique.
- **Source loader consistency:** The same registry loader must be used to invoke the setup verb, otherwise relative `require(...)` calls can fail.
- **Multi-site port conflicts:** The single xgoja.yaml runtime schema does not prevent a serve command from creating multiple runtime instances, but current auto-start mode cannot run many site runtimes on one address. Manual server ownership is required before `serve-multi`.
- **Security:** This system runs trusted JavaScript. Do not describe it as a safe multi-tenant sandbox.

## Alternatives considered

### Alternative A: Add only `express.serve()`

This would let JavaScript write:

```js
const express = require("express")
const app = express.app()
app.get("/", ...)
express.serve({ listen: "127.0.0.1:8787" })
```

It is simple, but it pushes lifecycle and signal handling into JavaScript or a long-pending promise. It also does not create discoverable generated commands. It may still be useful later as a script convenience, but it should not be the main xgoja verb-serving API.

### Alternative B: Add a built-in `commands.serve` to xgoja core

A core command can access `RuntimeSpec.JSVerbs` without provider API changes. The downside is that xgoja core becomes aware of HTTP serving, command flags, and Express assumptions. That is avoidable because command providers already exist.

### Alternative C: Use `run --keep-alive` and document it

This already works for script files and should remain supported. It does not solve generated verb ergonomics. Users would need a separate script entrypoint instead of a generated command per site.

### Alternative D: Invent `__site__`

A separate declaration API could make filtering cleaner, but it would duplicate jsverbs scanning, command generation, sections, and invocation. Reusing `__verb__` is less code and easier for users.

## API reference quick sheet

### Existing JavaScript APIs

```js
const express = require("express")
const app = express.app()

app.get(pattern, handler)
app.post(pattern, handler)
app.put(pattern, handler)
app.patch(pattern, handler)
app.delete(pattern, handler)
app.all(pattern, handler)

app.static(prefix, hostDirectory)
app.staticFromAssetsModule(prefix, assetsModule, root)
```

Handlers receive `(req, res)`. They may return a string, return a UI DSL value, call `res.send`, call `res.html`, call `res.json`, or return a promise.

### Proposed generated command shape

```bash
./my-app serve <verb parents...> <verb-name> [verb fields] --http-listen 127.0.0.1:8787
```

### Proposed buildspec fragment

```yaml
modules:
  - package: go-go-goja-http
    name: express

commands:
  jsverbs:
    enabled: true

commandProviders:
  - id: http-serve
    package: go-go-goja-http
    name: serve
    mount: serve
```

### Proposed provider registration fragment

```go
providerapi.CommandSetProvider{
    Name:         "serve",
    DefaultMount: "serve",
    Description:  "Serve JavaScript verb-backed HTTP sites",
    NewCommandSet: func(ctx providerapi.CommandSetContext) (*providerapi.CommandSet, error) {
        return newServeCommandSet(ctx, capability)
    },
}
```

## File reference map

Read these files in this order when implementing:

1. `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/pkg/xgoja/app/root.go` — generated root and jsverbs command construction.
2. `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/pkg/xgoja/app/run.go` — keep-alive lifecycle model.
3. `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/pkg/xgoja/app/command_providers.go` — provider command set mounting.
4. `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/pkg/xgoja/providerapi/commands.go` — command-provider API extension point.
5. `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/pkg/xgoja/providers/http/http.go` — HTTP provider, module section, server start/shutdown.
6. `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/modules/express/express.go` — JavaScript Express API.
7. `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/pkg/gojahttp/host.go` — request dispatch into Goja callbacks.
8. `/home/manuel/code/wesen/2026-05-03--goja-hosting-site/pkg/app/server.go` — single-site serving reference implementation.
9. `/home/manuel/code/wesen/2026-05-03--goja-hosting-site/pkg/app/multi_server.go` — future multi-site dispatch reference.

## Open questions

1. Should the first `serve` command expose all verbs, or only verbs tagged `http`/`site`?
2. Should `serve` require `--http-enabled=true`, or should it override disabled HTTP with a clear error?
3. Should bind errors be fixed in the HTTP provider before or during the serve command implementation?
4. Should `CommandSetContext.JSVerbs` live in `providerapi`, or should xgoja expose a smaller source-scanning package to avoid coupling provider APIs to `pkg/jsverbs`?
5. How much of `goja-site` observability should move into generic go-go-goja HTTP serving after the first implementation?
