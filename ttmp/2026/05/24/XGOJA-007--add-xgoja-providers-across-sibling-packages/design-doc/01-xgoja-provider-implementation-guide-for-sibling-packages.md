---
Title: XGoja Provider Implementation Guide For Sibling Packages
Ticket: XGOJA-007
Status: active
Topics:
    - xgoja
    - goja
    - providers
    - workspace-manager
    - geppetto
    - loupedeck
    - go-minitrace
    - goja-git
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/js/modules/geppetto/module.go
      Note: Existing rich geppetto module and Options type
    - Path: go-go-goja/cmd/xgoja/doc/04-providers.md
      Note: Provider authoring reference used as the contract baseline
    - Path: go-go-goja/pkg/xgoja/providerapi/module.go
      Note: Defines ModuleContext and Module contract
    - Path: go-minitrace/cmd/go-minitrace/cmds/query/js_runtime.go
      Note: Existing command-local minitrace module loader
    - Path: goja-git/gitmodule.go
      Note: Existing global git installer and Git API
    - Path: loupedeck/runtime/js/registrar.go
      Note: Existing multi-module runtime registrar
    - Path: workspace-manager/pkg/wsmjs/module/module.go
      Note: Existing wsm Register and Loader shape
ExternalSources: []
Summary: Intern-oriented analysis, design, and implementation guide for adding xgoja providers to geppetto, workspace-manager, goja-git, go-minitrace, and loupedeck.
LastUpdated: 2026-05-24T23:05:00-04:00
WhatFor: Use this as the implementation guide for converting the named sibling Goja bindings into xgoja provider packages.
WhenToUse: Before implementing or reviewing xgoja provider wrappers in geppetto, workspace-manager, goja-git, go-minitrace, or loupedeck.
---


# XGoja Provider Implementation Guide For Sibling Packages

## Executive summary

This ticket defines how to add xgoja provider packages to five sibling repositories in the current workspace:

- `workspace-manager`
- `goja-git`
- `go-minitrace`
- `loupedeck`
- `geppetto`

The goal is not to rewrite those packages' JavaScript APIs. The goal is to expose their existing Goja bindings through the xgoja provider contract so generated xgoja binaries can import provider packages, select modules in `xgoja.yaml`, and execute JavaScript with explicit runtime profiles.

The implementation should proceed in this order:

1. **Add simple provider wrappers first** for packages whose existing code already has a stable module loader or can get one with a small refactor: `workspace-manager` and `goja-git`.
2. **Extract provider-sized public loader factories** where current APIs only register into a runtime directly.
3. **Defer or split host-coupled provider work** for `go-minitrace`, `loupedeck`, and `geppetto` until their host services and config shapes are explicit.
4. **Add generated xgoja smoke examples** for every provider, because a provider is not complete until it works inside a generated binary.

The design follows the existing xgoja provider documentation in `go-go-goja/cmd/xgoja/doc/04-providers.md` and the broader conversion plan in `XGOJA-006`. The most important rule for an intern is this:

> A provider package only makes modules available to a generated binary. A runtime profile still decides which modules a specific command can `require(...)`.

That separation is the safety boundary. Do not create providers that register every possible module globally or hide host capabilities behind implicit defaults.

## Scope and non-scope

### In scope

- Add provider packages or provider adapter subpackages to:
  - `workspace-manager`
  - `goja-git`
  - `go-minitrace`
  - `loupedeck`
  - `geppetto`
- Define provider IDs, module names, config schemas, test strategy, and examples.
- Preserve the existing JavaScript API names where reasonable.
- Provide a roadmap from current code to xgoja-compatible code.
- Explain the system clearly enough for a new intern to implement the work.

### Out of scope for the first implementation pass

- Rewriting the underlying package APIs.
- Making hardware, database, LLM, or Git mutation operations safe by default.
- Automatically exposing all existing runtime modules with no review.
- Moving large app-coupled systems into xgoja without host interfaces.
- Implementing backward-compatibility shims unless the source package already needs them.

## Terms and mental model

### Goja

Goja is the JavaScript VM used by these Go packages. Native Go functionality is usually exposed to JavaScript through one of these shapes:

- a direct global object, such as `rt.Set("git", gitObj)`;
- a CommonJS-style native module registered with `goja_nodejs/require.Registry.RegisterNativeModule`;
- a runtime registrar that receives a richer engine context and registers several modules together.

### xgoja

`xgoja` is the generated-binary system in `go-go-goja`. It reads an `xgoja.yaml` buildspec, generates a Go binary, imports provider packages, registers modules into a provider registry, and creates commands like `eval`, `run`, `repl`, and `jsverbs`.

The generated binary runtime flow is:

```text
xgoja.yaml
  |
  | packages[] import provider packages
  | runtimes[] select provider modules
  v
generated Go main
  |
  | calls provider.Register(registry)
  v
providerapi.Registry
  |
  | RuntimeFactory resolves selected modules
  v
engine runtime + require.Registry
  |
  | JavaScript calls require("alias")
  v
provider module loader installs exports
```

### Provider package

A provider package is a normal Go package that exports a function like this:

```go
func Register(registry *providerapi.Registry) error
```

The provider calls `registry.Package("provider-id", entries...)` and registers one or more `providerapi.Module` entries. A module entry has a public provider module name, a default alias, a description, optional config schema, and a `New` factory that returns a `require.ModuleLoader`.

### Runtime profile

A runtime profile is the part of `xgoja.yaml` that says which provider modules are available to one generated command:

```yaml
runtimes:
  main:
    modules:
      - package: workspace-manager
        name: wsm
        as: wsm
```

If a module is not listed in the active runtime profile, JavaScript cannot require it even if the generated binary imported the provider package.

### Host-coupled module

A host-coupled module needs live host state that cannot be represented as plain JSON config. Examples in this ticket include:

- `go-minitrace` needing an open SQL connection and command metadata.
- `loupedeck/ui` needing a live deck environment and runtime owner callbacks.
- `geppetto` needing runners, profile registries, tool registries, middleware registries, event sinks, and optional persistence.

Host-coupled modules should use `providerapi.ModuleContext.Config` for static JSON and `providerapi.ModuleContext.Host` only for live services.

## Existing xgoja provider contract

The key API files are:

- `go-go-goja/pkg/xgoja/providerapi/module.go`
- `go-go-goja/pkg/xgoja/providerapi/registry.go`
- `go-go-goja/pkg/xgoja/app/spec.go`
- `go-go-goja/pkg/xgoja/app/factory.go`
- `go-go-goja/pkg/xgoja/testprovider/provider.go`
- `go-go-goja/cmd/xgoja/doc/04-providers.md`

### Provider module API

`providerapi.ModuleContext` is defined in `go-go-goja/pkg/xgoja/providerapi/module.go:12` and contains:

- `Context context.Context`
- `Name string`
- `As string`
- `Config json.RawMessage`
- `Host HostServices`

`providerapi.Module` is defined in `go-go-goja/pkg/xgoja/providerapi/module.go:22` and contains:

- `Name string`
- `DefaultAs string`
- `Description string`
- `ConfigSchema json.RawMessage`
- `New ModuleFactory`

The module factory contract is:

```go
type ModuleFactory func(ModuleContext) (require.ModuleLoader, error)
```

The factory runs when xgoja is constructing a runtime profile. This is the right place to:

- decode JSON config;
- reject unsafe defaults;
- construct adapters;
- return a loader for `require(...)`.

### Provider registry API

`Registry.Package` is defined in `go-go-goja/pkg/xgoja/providerapi/registry.go:28`. It creates one provider package namespace. It rejects blank package IDs, duplicate package IDs, nil entries, blank module names, nil module factories, and duplicate module names.

`Registry.ResolveModule` is defined in `go-go-goja/pkg/xgoja/providerapi/registry.go:57`. The runtime factory uses this to resolve `(packageID, moduleName)` from `xgoja.yaml`.

Important invariant:

```text
packages[].id in xgoja.yaml == registry.Package("same-id", ...)
runtimes.<profile>.modules[].package == same provider package ID
runtimes.<profile>.modules[].name == providerapi.Module.Name
require("...") uses modules[].as if present, otherwise modules[].name
```

### Runtime spec API

`go-go-goja/pkg/xgoja/app/spec.go` defines the embedded runtime spec. The most important structs are:

```go
type Spec struct {
    Name     string
    Packages []PackageSpec
    Runtimes map[string]Runtime
    Commands CommandsSpec
    JSVerbs  []JSVerbSourceSpec
}

type ModuleInstance struct {
    Package string
    Name    string
    As      string
    Config  map[string]any
}
```

`ModuleInstance.Alias()` returns `As` when set, otherwise `Name`.

### Runtime factory flow

`go-go-goja/pkg/xgoja/app/factory.go:54` constructs a runtime for a profile. For each module instance it:

1. Resolves the provider module with `providers.ResolveModule(instance.Package, instance.Name)`.
2. Wraps it as an engine runtime module.
3. Calls the provider module factory with `providerapi.ModuleContext`.
4. Registers the returned `require.ModuleLoader` under the selected alias.

The critical adapter in `go-go-goja/pkg/xgoja/app/factory.go:29` passes this context to providers:

```go
loader, err := s.module.New(providerapi.ModuleContext{
    Context: ctx.Context,
    Name:    s.instance.Name,
    As:      s.instance.Alias(),
    Config:  config,
})
```

This means providers should not read the xgoja YAML directly. They only see the normalized module context.

### Provider-shipped JavaScript verbs

`providerapi.VerbSource` lets a provider embed JavaScript commands. The fixture provider in `go-go-goja/pkg/xgoja/testprovider/provider.go:17` registers two modules and a verb source:

```go
return registry.Package("fixture",
    providerapi.Module{...},
    providerapi.Module{...},
    providerapi.VerbSource{Name: "verbs", Root: "verbs", FS: verbsFS},
)
```

The runtime command path resolves provider verb sources through `ResolveVerbSource` in `go-go-goja/pkg/xgoja/app/root.go:192`.

Provider-shipped verbs are optional. For this ticket, use them only when a package already has reusable JavaScript verbs that should travel with the provider. `go-minitrace` uses `jsverbs`, but its current verbs are command/catalog-coupled, so do not immediately expose them as provider-shipped verbs without a separate design step.

## Architecture diagram

```text
                       +-----------------------------+
                       | generated xgoja binary      |
                       |                             |
                       | imports provider packages   |
                       +--------------+--------------+
                                      |
                                      v
+-------------------+     +---------------------------+     +----------------------+
| xgoja.yaml        | --> | providerapi.Registry      | --> | providerapi.Module   |
| packages[]        |     | Package("workspace-manager") |  | Name: "wsm"         |
| runtimes[]        |     | Package("goja-git")      |     | New(ctx) -> Loader   |
+-------------------+     +---------------------------+     +----------+-----------+
                                                                         |
                                                                         v
                                                              +----------------------+
                                                              | require.Registry     |
                                                              | RegisterNativeModule |
                                                              +----------+-----------+
                                                                         |
                                                                         v
                                                              +----------------------+
                                                              | JavaScript VM        |
                                                              | require("wsm")       |
                                                              +----------------------+
```

For host-coupled modules, add one more boundary:

```text
xgoja command or host application
  |
  | creates HostServices implementation
  v
providerapi.ModuleContext.Host
  |
  | typed assertion inside provider
  v
module loader can access live service handles
```

Do not use global variables for this boundary. Pass explicit config and explicit host services.

## Repository-by-repository current-state analysis

This section maps each target repository to its current Goja shape, evidence, provider strategy, and risks.

### 1. workspace-manager

#### Current shape

`workspace-manager` already exposes a CommonJS-style Goja module named `wsm`.

Evidence:

- Module path: `workspace-manager/go.mod` declares `module github.com/go-go-golems/workspace-manager`.
- Module name: `workspace-manager/pkg/wsmjs/module/module.go:17` defines `ModuleName = "wsm"`.
- Options type: `workspace-manager/pkg/wsmjs/module/module.go:23` defines `type Options struct { ManagerOptions service.ManagerOptions }`.
- Current registration: `workspace-manager/pkg/wsmjs/module/module.go:28` defines `Register(reg *require.Registry, opts Options)` and calls `reg.RegisterNativeModule(ModuleName, mod.Loader)`.
- Loader: `workspace-manager/pkg/wsmjs/module/module.go:49` defines `func (m *module) Loader(vm *goja.Runtime, moduleObj *goja.Object)`.
- User-facing API docs: `workspace-manager/pkg/docs/03-js-api-and-runner.md` explains `require("wsm")`, `createManager`, `discover`, `createWorkspace`, and workspace/git methods.

#### Provider classification

- Pattern: existing `Register(reg *require.Registry, opts Options)`.
- Difficulty: low.
- Provider type: config-only provider.
- Host coupling: no live host object required for a first pass.
- Security: medium/high because methods can create workspaces, mutate repositories, run Git operations through workspace service code, and touch filesystem paths.

#### Proposed provider ID and module names

- Provider package ID: `workspace-manager`
- Provider Go package: `github.com/go-go-golems/workspace-manager/pkg/wsmjs/provider`
- Module name: `wsm`
- Default alias: `wsm`

#### Required source refactor

The existing module has a loader method, but it is hidden behind an unexported `module` type. Add one public loader factory in the existing module package:

```go
// package github.com/go-go-golems/workspace-manager/pkg/wsmjs/module
func NewLoader(opts Options) require.ModuleLoader {
    mod := &module{opts: opts}
    return mod.Loader
}

func Register(reg *require.Registry, opts Options) {
    if reg == nil {
        return
    }
    reg.RegisterNativeModule(ModuleName, NewLoader(opts))
}
```

This keeps the existing `Register` API but gives the provider wrapper a direct loader factory.

#### Provider wrapper pseudocode

```go
package provider

import (
    "encoding/json"
    "fmt"

    "github.com/dop251/goja_nodejs/require"
    wsmmodule "github.com/go-go-golems/workspace-manager/pkg/wsmjs/module"
    "github.com/go-go-golems/workspace-manager/pkg/wsmjs/service"
    "github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

const PackageID = "workspace-manager"

type Config struct {
    DefaultJobs int `json:"defaultJobs,omitempty"`
}

func Register(registry *providerapi.Registry) error {
    return registry.Package(PackageID, providerapi.Module{
        Name:        wsmmodule.ModuleName,
        DefaultAs:   wsmmodule.ModuleName,
        Description: "Workspace Manager automation module",
        ConfigSchema: json.RawMessage(`{"type":"object","properties":{"defaultJobs":{"type":"integer","minimum":1}}}`),
        New: func(ctx providerapi.ModuleContext) (require.ModuleLoader, error) {
            opts, err := optionsFromConfig(ctx.Config)
            if err != nil {
                return nil, fmt.Errorf("workspace-manager config: %w", err)
            }
            return wsmmodule.NewLoader(opts), nil
        },
    })
}

func optionsFromConfig(data json.RawMessage) (wsmmodule.Options, error) {
    cfg := Config{}
    if len(data) > 0 && string(data) != "null" {
        if err := json.Unmarshal(data, &cfg); err != nil {
            return wsmmodule.Options{}, err
        }
    }
    if cfg.DefaultJobs < 0 {
        return wsmmodule.Options{}, fmt.Errorf("defaultJobs must be non-negative")
    }
    return wsmmodule.Options{
        ManagerOptions: service.ManagerOptions{DefaultJobs: cfg.DefaultJobs},
    }, nil
}
```

#### Smoke example

Create `examples/xgoja/workspace-manager-provider/` in `workspace-manager` or in a shared examples area. Prefer per-repo examples so each repository can test its provider independently.

```yaml
name: workspace-manager-provider
packages:
  - id: workspace-manager
    import: github.com/go-go-golems/workspace-manager/pkg/wsmjs/provider
runtimes:
  main:
    modules:
      - package: workspace-manager
        name: wsm
        as: wsm
        config:
          defaultJobs: 4
commands:
  run:
    enabled: true
    runtime: main
```

Smoke JavaScript should avoid destructive operations. Use constants and API shape checks first:

```js
const wsm = require("wsm");
if (wsm.version !== "0.2.0") throw new Error("unexpected version");
if (!wsm.consts) throw new Error("missing consts");
if (typeof wsm.createManager !== "function") throw new Error("missing createManager");
console.log("workspace-manager provider ok");
```

#### Intern checklist

- [ ] Add `NewLoader(opts Options) require.ModuleLoader`.
- [ ] Add `pkg/wsmjs/provider/provider.go`.
- [ ] Add provider registration unit test.
- [ ] Add generated xgoja smoke example.
- [ ] Add negative config test for invalid `defaultJobs`.
- [ ] Run package tests and generated smoke.

### 2. goja-git

#### Current shape

`goja-git` exposes Git operations as a global `git` object, not as a CommonJS module.

Evidence:

- Module path: `goja-git/go.mod` declares `module github.com/go-go-golems/goja-git`.
- `goja-git/gitmodule.go:102` defines `InstallGit(rt *goja.Runtime)`.
- `InstallGit` creates a JS object with `open` and `init` and calls `rt.Set("git", gitObj)`.
- Operations include opening repositories, initializing repositories, status, add, commit, log, checkout, diff, branch operations, tags, and filter-repo support.

#### Provider classification

- Pattern: global object installer.
- Difficulty: medium because it needs conversion to a `require.ModuleLoader`.
- Provider type: host-capability provider.
- Host coupling: no live host services needed for a first pass, but it touches filesystem and Git repositories.
- Security: high because it can mutate repositories and write filtered repositories.

#### Proposed provider ID and module names

- Provider package ID: `goja-git`
- Provider Go package: `github.com/go-go-golems/goja-git/provider` or `github.com/go-go-golems/goja-git/gitprovider`
- Module name: `git`
- Default alias: `git`

#### Required source refactor

The current `GitModule` stores a runtime pointer and installs methods on a global object. Convert this into a loader while preserving `InstallGit` for the existing CLI.

Pseudocode:

```go
func NewGitObject(rt *goja.Runtime) *goja.Object {
    m := &GitModule{rt: rt}
    gitObj := rt.NewObject()
    _ = gitObj.Set("open", m.Open)
    _ = gitObj.Set("init", m.Init)
    return gitObj
}

func NewLoader() require.ModuleLoader {
    return func(rt *goja.Runtime, moduleObj *goja.Object) {
        exports := moduleObj.Get("exports").(*goja.Object)
        gitObj := NewGitObject(rt)
        _ = exports.Set("open", gitObj.Get("open"))
        _ = exports.Set("init", gitObj.Get("init"))
    }
}

func InstallGit(rt *goja.Runtime) {
    _ = rt.Set("git", NewGitObject(rt))
}
```

The provider then returns `NewLoader()`.

#### Safety config

Because `git.init`, `repo.add`, `repo.commit`, `repo.checkout`, `repo.branch.create`, `repo.tag.create`, and `repo.filterRepo` can modify disk, add a guard config before enabling the provider in generated binaries.

Suggested config:

```go
type Config struct {
    AllowWrite bool     `json:"allowWrite"`
    AllowedRoots []string `json:"allowedRoots,omitempty"`
}
```

The existing API currently receives paths at call time. A strict root sandbox would require path checks in every path-taking method. That is more work but safer. For the first provider, choose one of these options explicitly:

1. **Acknowledgement gate only:** require `allowWrite: true` but do not sandbox paths. This is easy but must be documented loudly.
2. **Root guard:** require every `Dir` and `OutDir` to be under `allowedRoots`. This requires adding path validation to `GitModule` and `RepoHandle`.

Recommended for intern implementation: start with acknowledgement gate plus docs, then add root guard in a second PR if the provider will be used outside trusted local automation.

#### Provider wrapper pseudocode

```go
const PackageID = "goja-git"
const ModuleName = "git"

type Config struct {
    AllowWrite bool `json:"allowWrite"`
}

func Register(registry *providerapi.Registry) error {
    return registry.Package(PackageID, providerapi.Module{
        Name:        ModuleName,
        DefaultAs:   ModuleName,
        Description: "Git operations backed by go-git",
        ConfigSchema: gitConfigSchema,
        New: func(ctx providerapi.ModuleContext) (require.ModuleLoader, error) {
            cfg, err := decodeConfig(ctx.Config)
            if err != nil { return nil, err }
            if !cfg.AllowWrite {
                return nil, fmt.Errorf("goja-git requires config.allowWrite=true")
            }
            return NewLoader(), nil
        },
    })
}
```

#### Smoke example

Use a temporary directory, not a checked-in repository. The smoke script should create a repository, check status, and avoid committing unless author config is deterministic.

```js
const git = require("git");
const repo = git.init({ Dir: tmpdir, DefaultBranch: "main" });
const status = repo.status();
if (!status) throw new Error("missing status");
console.log("goja-git provider ok");
```

The example Makefile can create `tmpdir` before running the generated binary.

#### Intern checklist

- [ ] Refactor `InstallGit` around `NewGitObject(rt)`.
- [ ] Add `NewLoader()`.
- [ ] Add provider package with `allowWrite` guard.
- [ ] Add provider registry test.
- [ ] Add generated smoke that creates a temp repo.
- [ ] Add documentation warning about repository mutations.

### 3. go-minitrace

#### Current shape

`go-minitrace` currently creates a command-local JavaScript runtime for query commands and registers a `minitrace` module inline.

Evidence:

- Module path: `go-minitrace/go.mod` declares `module github.com/go-go-golems/go-minitrace`.
- `go-minitrace/cmd/go-minitrace/cmds/query/js_runtime.go:24` defines `RunJSCommandIntoProcessor(...)`.
- That function builds a `go-go-goja/engine` runtime and registers a native module named `minitrace` in `NativeModuleSpec` at `go-minitrace/cmd/go-minitrace/cmds/query/js_runtime.go:65`.
- `minitraceModuleLoader` is defined at `go-minitrace/cmd/go-minitrace/cmds/query/js_runtime.go:151`.
- The loader exports `query`, `queryOne`, `tableName`, `runtime`, and SQL helper functions.
- Query execution calls `queryengine.ValidateReadOnlyQuery(sqlText)` before running SQL.

#### Provider classification

- Pattern: command-local loader with live SQL connection.
- Difficulty: high.
- Provider type: host-coupled provider.
- Host coupling: requires `*sql.Conn`, command metadata, runtime settings, and context.
- Security: medium/high because it reads local trace databases. It validates read-only SQL, which is a helpful guard.

#### Design decision

Do not expose the current command-local loader directly from `cmd/...`. Move or copy the reusable module surface into a public package first.

Recommended package layout:

```text
go-minitrace/
  pkg/minitracejs/
    module.go        # ModuleName, Config, HostServices, NewLoader
    query.go         # read-only query helpers if reusable
    provider/
      provider.go    # Register(*providerapi.Registry)
```

#### Host services contract

The provider cannot open an arbitrary database safely from config by default. Define a host contract that target-mode or command integration can pass through `ModuleContext.Host`:

```go
type HostServices interface {
    Conn() *sql.Conn
    RuntimeSettings() RuntimeSettings
    CommandName() string
}

type RuntimeSettings struct {
    TableName string
    DBPath string
    ArchiveGlob []string
    PersistLoaded bool
}
```

Provider module factory:

```go
New: func(ctx providerapi.ModuleContext) (require.ModuleLoader, error) {
    host, ok := ctx.Host.(minitracejs.HostServices)
    if !ok || host == nil {
        return nil, fmt.Errorf("minitrace provider requires minitracejs.HostServices")
    }
    return minitracejs.NewLoader(ctx.Context, host.Conn(), host.CommandName(), host.RuntimeSettings()), nil
}
```

For pure generated xgoja binaries without host services, add a config-only mode only if it can safely open a read-only DuckDB connection:

```go
type Config struct {
    DBPath string `json:"dbPath,omitempty"`
    TableName string `json:"tableName,omitempty"`
    ReadOnly bool `json:"readOnly"`
}
```

Require `readOnly: true` and reject write-capable or missing mode. This is a later phase, not the first provider wrapper.

#### Provider module API

- Provider package ID: `go-minitrace`
- Module name: `minitrace`
- Default alias: `minitrace`
- First pass availability: host-mode only.

#### Smoke strategy

A generated standalone xgoja smoke is difficult until config-only DB opening exists. Use two tiers:

1. Unit test for `provider.Register` and host type assertion error.
2. Integration test in `go-minitrace` command code that constructs a real host services implementation and invokes JavaScript through xgoja runtime factory.

Smoke JavaScript:

```js
const mt = require("minitrace");
const one = mt.queryOne(`select 1 as ok`);
if (one.ok !== 1) throw new Error("queryOne failed");
if (!mt.runtime.tableName) throw new Error("missing runtime metadata");
```

#### Intern checklist

- [ ] Extract `minitraceModuleLoader` from `cmd/.../query/js_runtime.go` into `pkg/minitracejs`.
- [ ] Keep command behavior unchanged by calling the new loader package.
- [ ] Define `HostServices` and a provider wrapper.
- [ ] Add tests for missing host services and happy-path host services.
- [ ] Only add config-only DB opening after read-only connection policy is designed.

### 4. loupedeck

#### Current shape

`loupedeck` has a rich runtime registrar that installs several modules and stores an environment in runtime-scoped bindings.

Evidence:

- Module path: `loupedeck/go.mod` declares `module github.com/go-go-golems/loupedeck`.
- `loupedeck/runtime/js/registrar.go:32` defines `RegisterRuntimeModules(ctx *engine.RuntimeModuleContext, reg *require.Registry) error`.
- The registrar stores the environment in the VM, sets context value `environment`, installs metadata sentinel functions, registers cleanup, then registers module packages.
- Module names include:
  - `loupedeck/runtime/js/module_ui/module.go:18`: `loupedeck/ui`
  - `loupedeck/runtime/js/module_gfx/module.go:11`: `loupedeck/gfx`
  - `loupedeck/runtime/js/module_easing/module.go:9`: `loupedeck/easing`
  - `loupedeck/runtime/js/module_anim/module.go:17`: `loupedeck/anim`
  - `loupedeck/runtime/js/module_state/module.go:15`: `loupedeck/state`
  - `loupedeck/runtime/js/module_present/module.go:15`: `loupedeck/present`
  - `loupedeck/runtime/js/module_metrics/module.go:8`: `loupedeck/metrics`
  - `loupedeck/runtime/js/module_scene_metrics/module.go:8`: `loupedeck/scene-metrics`
- `loupedeck/pkg/jsmetrics/jsmetrics.go:22` registers metrics modules under a prefix.
- `loupedeck/runtime/js/module_ui/module.go:18` and `module_present` require runtime owner bindings and environment bindings.

#### Provider classification

- Pattern: multi-module runtime registrar.
- Difficulty: high.
- Provider type: mixed provider set.
- Host coupling:
  - `gfx` and `easing` are mostly standalone.
  - `state` and `anim` depend on runtime owner semantics.
  - `ui`, `present`, `metrics`, and `scene-metrics` depend on environment bindings.
- Security: hardware/device control and UI event callbacks require explicit host services.

#### Split the provider set

Do not create one provider module that implicitly registers everything. Split modules by host dependency:

1. `loupedeck-core` or package ID `loupedeck`
   - `loupedeck/easing`
   - `loupedeck/gfx`
2. `loupedeck-runtime`
   - `loupedeck/state`
   - `loupedeck/anim`
3. `loupedeck-host`
   - `loupedeck/ui`
   - `loupedeck/present`
   - `loupedeck/metrics`
   - `loupedeck/scene-metrics`

However, `providerapi.Registry.Package` registers one package ID per call. The code can still place all modules in one Go provider package, but use explicit module entries and require explicit runtime profile selection. The package ID can remain `loupedeck` if the module-level config/host requirements are clear.

Recommended first pass:

- Provider package ID: `loupedeck`
- Provider package: `github.com/go-go-golems/loupedeck/runtime/js/provider`
- Register only `loupedeck/easing` and `loupedeck/gfx` first.
- Add host-coupled modules in later phases.

#### Required source refactor

Each module currently exposes only `Register(registry *require.Registry)`, which hides the actual loader. Add `Loader()` functions to each module package.

Example for `module_easing`:

```go
func Loader() require.ModuleLoader {
    return func(runtime *goja.Runtime, module *goja.Object) {
        exports := module.Get("exports").(*goja.Object)
        // existing body from RegisterNativeModule
    }
}

func Register(registry *require.Registry) {
    registry.RegisterNativeModule(ModuleName, Loader())
}
```

For modules that need environment setup, loader factories should be explicit about requirements:

```go
func Loader() require.ModuleLoader {
    return func(runtime *goja.Runtime, module *goja.Object) {
        bindings, ok := runtimebridge.Lookup(runtime)
        if !ok || bindings.Owner == nil {
            panic(runtime.NewGoError(fmt.Errorf("ui module requires runtime owner bindings")))
        }
        env, ok := envpkg.Lookup(runtime)
        if !ok || env == nil {
            panic(runtime.NewGoError(fmt.Errorf("ui module requires environment bindings")))
        }
        ...
    }
}
```

#### Host services contract

For environment-bound modules, the provider should have one shared host contract:

```go
type HostServices interface {
    Environment() *env.LoupeDeckEnvironment
}
```

The provider must store this environment into the runtime before the module is loaded. That is tricky because `providerapi.Module.New` returns a loader, while environment storage happens against the VM inside the loader. The loader can store the environment at the beginning of each module load:

```go
New: func(ctx providerapi.ModuleContext) (require.ModuleLoader, error) {
    host, ok := ctx.Host.(HostServices)
    if !ok || host.Environment() == nil {
        return nil, fmt.Errorf("loupedeck/ui requires loupedeck provider host services")
    }
    env := host.Environment()
    return func(vm *goja.Runtime, moduleObj *goja.Object) {
        envpkg.Store(vm, env)
        module_ui.Loader()(vm, moduleObj)
    }, nil
}
```

If multiple modules store the same environment, make storage idempotent and add cleanup through runtime owner/engine closer only if xgoja exposes a closer hook for provider modules. If no closer hook is available at provider level, document that environment lifetime is owned by the host application.

#### Provider wrapper pseudocode for safe modules

```go
func Register(registry *providerapi.Registry) error {
    return registry.Package("loupedeck",
        providerapi.Module{
            Name: "loupedeck/easing",
            DefaultAs: "loupedeck/easing",
            Description: "Easing functions for animation curves",
            New: func(providerapi.ModuleContext) (require.ModuleLoader, error) {
                return module_easing.Loader(), nil
            },
        },
        providerapi.Module{
            Name: "loupedeck/gfx",
            DefaultAs: "loupedeck/gfx",
            Description: "Offscreen drawing surface and font helpers",
            New: func(providerapi.ModuleContext) (require.ModuleLoader, error) {
                return module_gfx.Loader(), nil
            },
        },
    )
}
```

#### Smoke example

```js
const easing = require("loupedeck/easing");
const gfx = require("loupedeck/gfx");
if (typeof easing.linear !== "function") throw new Error("missing easing.linear");
const s = gfx.surface(64, 32);
if (!s) throw new Error("missing surface");
console.log("loupedeck provider ok");
```

Do not require `loupedeck/ui` in a standalone generated binary smoke unless a fake host environment is provided.

#### Intern checklist

- [ ] Add `Loader()` to `module_easing` and `module_gfx` first.
- [ ] Add `runtime/js/provider/provider.go` registering those safe modules.
- [ ] Add tests that `Register` resolves those modules.
- [ ] Add generated smoke for `easing` and `gfx`.
- [ ] Design host services before exposing `ui`, `present`, `metrics`, or `scene-metrics`.
- [ ] Verify owner-thread behavior before exposing `anim` and `state`.

### 5. geppetto

#### Current shape

`geppetto` exposes a rich `require("geppetto")` module with options for runners, tools, middleware, profile registries, event sinks, persistence, and logging.

Evidence:

- Module path: `geppetto/go.mod` declares `module github.com/go-go-golems/geppetto`.
- `geppetto/pkg/js/modules/geppetto/module.go:25` defines `ModuleName = "geppetto"`.
- `geppetto/pkg/js/modules/geppetto/module.go:34` defines `type Options struct` with many host service fields.
- `geppetto/pkg/js/modules/geppetto/module.go:52` defines `Register(reg *require.Registry, opts Options)`.
- `geppetto/pkg/js/modules/geppetto/module.go:131` defines the loader.
- `installExports` exports `createBuilder`, `createSession`, `runInference`, turns helpers, engine helpers, profile helpers, and runner helpers.

#### Provider classification

- Pattern: existing `Register(reg *require.Registry, opts Options)` with heavy options.
- Difficulty: high.
- Provider type: host-coupled plus optional config-only helper provider.
- Host coupling: significant. Many useful operations need runtime runner, tool registry, profile registry, middleware factories, event sinks, and optional persistence.
- Security: high because it can connect to LLM providers, use credentials, run tools, and persist turns.

#### Split implementation into two phases

Do not try to expose the entire geppetto runtime surface from static config in the first pass. Split into:

1. **Static/helper provider** for operations that do not require external engines or live runners.
2. **Host-services provider** for full inference/session/runner operations.

If the module cannot be meaningfully split internally yet, still expose only a host-services provider first and require the host to supply `Options`.

#### Required source refactor

Like `workspace-manager`, add a public loader factory:

```go
func NewLoader(opts Options) require.ModuleLoader {
    mod := &module{opts: opts}
    return mod.Loader
}

func Register(reg *require.Registry, opts Options) {
    if reg == nil {
        return
    }
    reg.RegisterNativeModule(ModuleName, NewLoader(opts))
}
```

#### Host services contract

Avoid putting the entire `Options` type into `providerapi.ModuleContext.Host` as an untyped `any` without a contract. Define a small interface in the provider package:

```go
type HostServices interface {
    GeppettoOptions(ctx context.Context, cfg Config) (geppettomodule.Options, error)
}

type Config struct {
    Profile string `json:"profile,omitempty"`
    Registry string `json:"registry,omitempty"`
    AllowNetwork bool `json:"allowNetwork"`
    AllowTools bool `json:"allowTools"`
}
```

Provider factory:

```go
New: func(ctx providerapi.ModuleContext) (require.ModuleLoader, error) {
    cfg, err := decodeConfig(ctx.Config)
    if err != nil { return nil, err }
    if !cfg.AllowNetwork {
        // Either reject full provider use or install only local helper APIs.
        return nil, fmt.Errorf("geppetto provider requires config.allowNetwork=true for inference APIs")
    }
    host, ok := ctx.Host.(HostServices)
    if !ok || host == nil {
        return nil, fmt.Errorf("geppetto provider requires geppetto HostServices")
    }
    opts, err := host.GeppettoOptions(ctx.Context, cfg)
    if err != nil { return nil, err }
    return geppettomodule.NewLoader(opts), nil
}
```

#### Config policy

Minimum config fields:

- `allowNetwork`: required for engine/provider calls.
- `allowTools`: required if Go tool registry is exposed.
- `profile`: optional default engine profile name.
- `registry`: optional registry selector.
- `persist`: optional flag or policy object if persistence is available.

Do not read API keys directly from global environment in the provider. If an implementation must use environment variables, config should name the variable:

```yaml
config:
  allowNetwork: true
  apiKeyEnv: OPENAI_API_KEY
```

Then the host service decides whether to honor it.

#### Smoke strategy

Use an offline engine first. The geppetto module appears to expose `engines.echo` in `installExports`, so use that if it does not require network or profiles.

```js
const geppetto = require("geppetto");
if (geppetto.version !== "0.1.0") throw new Error("unexpected version");
if (typeof geppetto.turns.newTurn !== "function") throw new Error("missing turns API");
if (typeof geppetto.engines.echo !== "function") throw new Error("missing echo engine");
console.log("geppetto provider basic API ok");
```

If even this requires full options, write the first test as a provider registry and loader construction test with fake host services.

#### Intern checklist

- [ ] Add `NewLoader(opts Options)` to `pkg/js/modules/geppetto`.
- [ ] Define `pkg/js/modules/geppetto/provider` with config and host service interface.
- [ ] Add fake host services tests.
- [ ] Add offline smoke if `turns` and `engines.echo` work without network.
- [ ] Add docs for network, credentials, tool execution, and persistence risks.

## Provider classification table

| Repository | Current shape | Provider ID | Module(s) | First-pass status | Risk level |
| --- | --- | --- | --- | --- | --- |
| `workspace-manager` | `Register(reg, opts)` native module | `workspace-manager` | `wsm` | Ready after `NewLoader` refactor | Medium/high filesystem and Git operations |
| `goja-git` | Global `InstallGit(rt)` object | `goja-git` | `git` | Needs `NewLoader` extraction | High repository mutation |
| `go-minitrace` | Command-local loader with SQL conn | `go-minitrace` | `minitrace` | Needs public package and host services | Medium/high database access |
| `loupedeck` | Runtime registrar plus multiple modules | `loupedeck` | `loupedeck/easing`, `gfx`, `state`, `anim`, `ui`, `present`, `metrics`, `scene-metrics` | Start with safe modules, host services later | High hardware/UI callbacks |
| `geppetto` | `Register(reg, opts)` native module with rich options | `geppetto` | `geppetto` | Needs `NewLoader` and host services | High network/tools/credentials |

## Common implementation pattern

Every provider implementation should follow this file layout where possible:

```text
<repo>/
  pkg/.../module/          # existing JS module implementation
    module.go              # ModuleName, Options, Register, NewLoader
  pkg/.../provider/
    provider.go            # providerapi.Register wrapper
    provider_test.go       # registry and config tests
  examples/xgoja/<name>/
    xgoja.yaml
    scripts/smoke.js
    Makefile
```

If a repository already has a better internal layout, keep the provider close to the existing Goja module. The provider should be importable from a generated binary, so do not put it under `internal/` unless the generated binary is built inside the same module.

### Minimal provider test

```go
func TestRegisterProvider(t *testing.T) {
    registry := providerapi.NewRegistry()
    if err := provider.Register(registry); err != nil {
        t.Fatalf("register provider: %v", err)
    }
    if _, ok := registry.ResolveModule(provider.PackageID, module.ModuleName); !ok {
        t.Fatalf("missing module %s.%s", provider.PackageID, module.ModuleName)
    }
}
```

### Minimal generated smoke commands

```bash
xgoja doctor -f examples/xgoja/<provider>/xgoja.yaml
xgoja list-modules -f examples/xgoja/<provider>/xgoja.yaml
xgoja build -f examples/xgoja/<provider>/xgoja.yaml --xgoja-replace /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja --keep-work
./examples/xgoja/<provider>/dist/<provider> run examples/xgoja/<provider>/scripts/smoke.js
```

When working from this multi-module workspace, use `replace` directives or `--xgoja-replace` so generated builds resolve local checkouts instead of stale released versions.

## Configuration and security rules

### Default-deny rules

Provider modules should fail closed. If a module can mutate host state, run tools, access a device, call a network API, or open a database, require config that acknowledges the capability.

Good:

```yaml
config:
  allowWrite: true
```

Bad:

```yaml
config: {}
```

for a module that can write files or mutate repositories.

### Capability matrix

| Capability | Packages | Required guard |
| --- | --- | --- |
| Pure JavaScript/CPU helpers | `loupedeck/easing`, parts of `geppetto.turns` | No special guard beyond tests |
| Filesystem read/write | `workspace-manager`, `goja-git`, `loupedeck/gfx` font loading | Config acknowledgement and optional allowed roots |
| Git mutation | `workspace-manager`, `goja-git` | `allowWrite` and temp-dir smoke tests |
| Database read | `go-minitrace` | Read-only query validation and explicit DB/host services |
| Hardware/UI | `loupedeck/ui`, `present` | Host services interface, fake-host tests |
| Network/LLM | `geppetto` | `allowNetwork`, credential policy, host services |
| Tool execution | `geppetto`, possibly `workspace-manager` through Git | `allowTools` or explicit tool registry from host |
| Async owner callbacks | `loupedeck/anim`, `state`, `ui`, `present`, `geppetto` runner APIs | Generated xgoja/engine tests, never raw `goja.New()` only |

### Config decoding helper

Each provider can define a small helper:

```go
func decodeConfig(data json.RawMessage, out any) error {
    if len(data) == 0 || string(data) == "null" {
        return nil
    }
    return json.Unmarshal(data, out)
}
```

Validate after decode. Do not rely on JavaScript runtime errors for invalid provider config.

## Implementation phases

### Phase 1: workspace-manager provider

This is the best first implementation because it already has a native module shape.

Deliverables:

- `pkg/wsmjs/module.NewLoader`.
- `pkg/wsmjs/provider.Register`.
- Provider unit tests.
- Generated xgoja smoke example.
- Documentation update in `pkg/docs/03-js-api-and-runner.md` or a new provider doc.

Validation:

```bash
cd workspace-manager
go test ./pkg/wsmjs/... -count=1
make -C examples/xgoja/workspace-manager-provider smoke
```

### Phase 2: goja-git provider

Deliverables:

- `NewGitObject(rt)` and `NewLoader()`.
- Existing CLI remains on `InstallGit(rt)`.
- Provider with `allowWrite` config.
- Temp-repo generated smoke.

Validation:

```bash
cd goja-git
go test ./... -count=1
make -C examples/xgoja/goja-git-provider smoke
```

### Phase 3: loupedeck safe modules

Deliverables:

- `Loader()` functions for `module_easing` and `module_gfx`.
- `runtime/js/provider.Register` for safe modules.
- Smoke example requiring only safe modules.

Validation:

```bash
cd loupedeck
go test ./runtime/js/... ./pkg/jsmetrics/... -count=1
make -C examples/xgoja/loupedeck-provider smoke
```

### Phase 4: geppetto host-services provider

Deliverables:

- `NewLoader(opts Options)`.
- Provider config and `HostServices` interface.
- Fake-host unit tests.
- Offline smoke if possible.

Validation:

```bash
cd geppetto
go test ./pkg/js/modules/geppetto/... -count=1
make -C examples/xgoja/geppetto-provider smoke
```

### Phase 5: go-minitrace host-services provider

Deliverables:

- Extract `minitraceModuleLoader` into a public package.
- Provider that requires host services.
- Integration test with a test SQL connection.
- Optional config-only read-only mode as a later improvement.

Validation:

```bash
cd go-minitrace
go test ./pkg/minitracejs/... ./cmd/go-minitrace/cmds/query/... -count=1
```

### Phase 6: host-coupled loupedeck modules

Deliverables:

- Host services interface for environment.
- Provider entries for `ui`, `present`, `metrics`, `scene-metrics`.
- Fake environment tests.
- Hardware integration test remains manual or separately gated.

## Review checklist

Use this checklist for every provider PR:

- [ ] Provider package exports `Register(*providerapi.Registry) error`.
- [ ] Provider ID is a constant and matches examples.
- [ ] Every module has explicit `Name`, `DefaultAs`, and `Description`.
- [ ] Dangerous modules require explicit config.
- [ ] Config decode errors happen during runtime construction, not after random JS execution.
- [ ] Existing non-xgoja registration APIs keep working.
- [ ] Tests call `providerapi.NewRegistry()` and `ResolveModule`.
- [ ] Generated xgoja smoke proves `require(...)` works.
- [ ] Async modules are tested with owner/runtime semantics, not only raw `goja.New()`.
- [ ] Docs list module name, alias, config fields, and risk level.

## Open questions

- Should `loupedeck` use one provider package ID with many explicitly selected modules, or separate provider IDs for `loupedeck-core`, `loupedeck-runtime`, and `loupedeck-host`?
- Should `goja-git` implement path root sandboxing before the provider is considered complete, or is an explicit `allowWrite` acknowledgement acceptable for trusted local use?
- Should `go-minitrace` support config-only read-only database opening, or should it remain host-services-only?
- Should `geppetto` split a helper-only provider from the full inference provider, or is one host-services provider simpler for now?
- Does `providerapi.ModuleContext.Host` need typed examples in `go-go-goja` before interns implement host-coupled providers?

## References

### Existing xgoja provider docs and APIs

- `go-go-goja/cmd/xgoja/doc/04-providers.md`
- `go-go-goja/ttmp/2026/05/24/XGOJA-006--convert-existing-goja-bindings-into-xgoja-package-providers/design-doc/01-goja-binding-provider-conversion-implementation-guide.md`
- `go-go-goja/pkg/xgoja/providerapi/module.go`
- `go-go-goja/pkg/xgoja/providerapi/registry.go`
- `go-go-goja/pkg/xgoja/app/spec.go`
- `go-go-goja/pkg/xgoja/app/factory.go`
- `go-go-goja/pkg/xgoja/testprovider/provider.go`
- `go-go-goja/pkg/xgoja/providers/core/core.go`
- `go-go-goja/pkg/xgoja/providers/host/host.go`

### Target package files

- `workspace-manager/pkg/wsmjs/module/module.go`
- `workspace-manager/pkg/docs/03-js-api-and-runner.md`
- `goja-git/gitmodule.go`
- `go-minitrace/cmd/go-minitrace/cmds/query/js_runtime.go`
- `loupedeck/runtime/js/registrar.go`
- `loupedeck/runtime/js/module_easing/module.go`
- `loupedeck/runtime/js/module_gfx/module.go`
- `loupedeck/runtime/js/module_ui/module.go`
- `loupedeck/runtime/js/module_present/module.go`
- `loupedeck/pkg/jsmetrics/jsmetrics.go`
- `geppetto/pkg/js/modules/geppetto/module.go`

### Ticket-local evidence

- `ttmp/2026/05/24/XGOJA-007--add-xgoja-providers-across-sibling-packages/scripts/01-inventory-target-goja-bindings.sh`
- `ttmp/2026/05/24/XGOJA-007--add-xgoja-providers-across-sibling-packages/sources/01-inventory-target-goja-bindings.txt`
