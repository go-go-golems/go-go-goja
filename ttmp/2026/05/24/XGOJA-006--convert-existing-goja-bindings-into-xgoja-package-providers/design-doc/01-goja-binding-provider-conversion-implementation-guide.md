---
Title: Goja Binding Provider Conversion Implementation Guide
Ticket: XGOJA-006
Status: active
Topics:
    - xgoja
    - goja
    - modules
    - js-bindings
    - architecture
    - testing
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../../code/wesen/go-go-golems/cozodb-goja/pkg/cozoapi/module/cozodb.go
      Note: Example external Loader-style module candidate for provider conversion
    - Path: ../../../../../../../../../../code/wesen/go-go-golems/geppetto/pkg/js/modules/geppetto/module.go
      Note: Example host-coupled high-value provider candidate
    - Path: ../../../../../../../../../../code/wesen/go-go-golems/loupedeck/runtime/js/registrar.go
      Note: Example external runtime registrar candidate for provider conversion
    - Path: cmd/xgoja/doc/04-providers.md
      Note: Bundled xgoja provider authoring help page
    - Path: examples/xgoja/core-provider/xgoja.yaml
      Note: Generated xgoja smoke example for core provider
    - Path: examples/xgoja/host-provider/xgoja.yaml
      Note: Generated xgoja smoke example for guarded host provider
    - Path: modules/common.go
      Note: Existing first-party NativeModule registry pattern to adapt into providers
    - Path: pkg/xgoja/providerapi/module.go
      Note: Module factory and context contract provider wrappers must implement
    - Path: pkg/xgoja/providerapi/registry.go
      Note: Provider package registry contract used by generated xgoja binaries
    - Path: pkg/xgoja/providers/core/core.go
      Note: First-party safe/core provider implementation
    - Path: pkg/xgoja/providers/host/host.go
      Note: Guarded host-capability provider implementation
    - Path: pkg/xgoja/testprovider/provider.go
      Note: Reference provider implementation with modules and provider-shipped jsverbs
ExternalSources: []
Summary: Plan for converting existing go-go-golems Goja bindings into xgoja provider packages with phased implementation, testing, and review guidance.
LastUpdated: 2026-05-24T14:40:34.442824097-04:00
WhatFor: Use this guide when turning existing Goja-facing Go packages into xgoja package providers selectable from xgoja.yaml.
WhenToUse: Before implementing provider wrappers for existing Goja modules, especially packages in sibling go-go-golems repositories.
---




# Goja Binding Provider Conversion Implementation Guide

## Executive Summary

The `xgoja` builder can now generate binaries that import provider packages, register Go-backed Goja modules, select modules by runtime profile, expose generated `repl`, `run`, `tui`, and `modules` commands, and mount JavaScript verbs from runtime, embedded, or provider-shipped sources. The next useful step is to convert existing Goja-binding packages in `~/code/wesen/go-go-golems/` into provider packages that can be consumed by generated xgoja binaries.

This ticket should produce a provider catalog plus implementation adapters for several classes of existing bindings:

1. **Loader-style modules** that already expose `func(vm *goja.Runtime, module *goja.Object)` loaders.
2. **Register-style modules** that already register themselves into a `*require.Registry`.
3. **Runtime-registrar modules** that already use `engine.RuntimeModuleContext` and should preserve owner/context semantics.
4. **Host-coupled modules** that need `providerapi.ModuleContext.Config` or `providerapi.ModuleContext.Host` to receive credentials, clients, stores, service handles, or policy objects.
5. **Internal/app-coupled bindings** that must either move to public packages or receive adapter packages inside the same module.

The implementation should proceed in phases. Start with simple provider wrappers and smoke tests, then move to multi-module provider sets, then address internal/app-coupled packages after their boundaries are explicit.

## Problem Statement

Many sibling repositories already expose functionality to JavaScript through Goja. Today those bindings are usually tied to each repository's own runtime bootstrap, command, or application host. Generated xgoja binaries cannot consume them unless the packages expose the `providerapi.Register(*providerapi.Registry) error` contract expected by `xgoja.yaml`.

The lack of provider wrappers creates several problems:

- Reusable Goja bindings cannot be composed into a generated binary without writing custom generated-source edits or local bootstrapping code.
- Module names, configuration shape, and runtime ownership assumptions vary from package to package.
- There is no consistent smoke-test pattern proving that a generated xgoja binary can import each provider, select modules in a runtime profile, and execute JavaScript against them.
- Some candidates live under `internal/`, so generated binaries outside the source module cannot import them.
- Some modules expose host operations such as filesystem, shell, credentials, network calls, UI, or device control and need explicit security/configuration review before becoming reusable providers.

## Goals

- Create reusable provider packages for selected existing Goja bindings.
- Keep generated binaries spec-driven: `xgoja.yaml` selects packages, modules, aliases, command runtime profiles, and optional jsverb sources.
- Avoid global implicit module registration for provider packages.
- Preserve runtime owner/context semantics for async modules.
- Provide a repeatable conversion playbook and tests for future packages.
- Document security and configuration boundaries per provider.

## Non-Goals

- Do not force every existing Goja runtime into the xgoja provider model in one pass.
- Do not make app-coupled runtime hosts magically generic without explicit host-service interfaces.
- Do not preserve deprecated `engine` compatibility wrappers or reintroduce global `modules.EnableAll` behavior.
- Do not expose dangerous capabilities by default without config/policy surfaces.

## Current xgoja Provider Contract

A provider package exports a public function, normally named `Register`:

```go
func Register(registry *providerapi.Registry) error
```

That function registers one provider package ID and one or more modules and verb sources:

```go
func Register(registry *providerapi.Registry) error {
    return registry.Package("example",
        providerapi.Module{
            Name:        "hello",
            DefaultAs:   "hello",
            Description: "Example module",
            New: func(ctx providerapi.ModuleContext) (require.ModuleLoader, error) {
                return func(vm *goja.Runtime, module *goja.Object) {
                    exports := module.Get("exports").(*goja.Object)
                    _ = exports.Set("greet", func(name string) string {
                        return "hello " + name
                    })
                }, nil
            },
        },
    )
}
```

Generated xgoja code imports the provider package, calls the configured registration function, embeds the normalized runtime spec, and uses `app.RuntimeFactory.NewRuntime(ctx, profile, ...)` when executing commands. Provider modules should therefore return `require.ModuleLoader` factories and avoid assuming they are installed globally.

## Candidate Inventory

The following packages were found by scanning `~/code/wesen/go-go-golems/` for `goja`, `goja_nodejs`, `require.Registry`, `RegisterNativeModule`, `modules.NativeModule`, and runtime registrar patterns.

### Tier 1: Simple or already aligned

These are the best first-pass conversion targets.

| Repository / package | Current shape | Candidate provider package ID | Module names | Notes |
| --- | --- | --- | --- | --- |
| `go-go-goja/modules/fs` | `modules.NativeModule` loader | `go-go-goja-core` or `nodejs-primitives` | `fs`, `node:fs` | Existing module; provider wrapper can adapt loaders directly. Security review required for filesystem scope. |
| `go-go-goja/modules/os` | `modules.NativeModule` loader | `go-go-goja-core` or `nodejs-primitives` | `os`, `node:os` | Consider whether this belongs in data-only vs host-capability provider sets. |
| `go-go-goja/modules/path` | `modules.NativeModule` loader | `go-go-goja-core` or `nodejs-primitives` | `path`, `node:path` | Low-risk. Good smoke-test module. |
| `go-go-goja/modules/yaml` | `modules.NativeModule` loader | `go-go-goja-core` | `yaml` | Low-risk data transform module. |
| `go-go-goja/modules/crypto` | `modules.NativeModule` loader | `go-go-goja-core` | `crypto`, `node:crypto` | Mostly safe, but document API scope. |
| `go-go-goja/modules/time` | `modules.NativeModule` loader | `go-go-goja-core` | `time` | Low-risk. |
| `go-go-goja/modules/timer` | `modules.NativeModule` loader | `go-go-goja-core` | `timer` | Async behavior should be tested with runtime owner/event loop. |
| `go-go-goja/modules/events` | `modules.NativeModule` loader | `go-go-goja-core` | `events`, `node:events` | Important base dependency for event-oriented modules. |
| `go-go-goja/modules/database` | `modules.NativeModule` loader with options | `go-go-goja-data` | `database`, `db` | Needs config schema for DSN/driver/permissions. |
| `go-go-goja/modules/exec` | `modules.NativeModule` loader | `go-go-goja-host` | `exec` | Dangerous by default; require explicit provider/config docs. |
| `go-go-goja/modules/uidsl` | `RegisterRuntimeModules` | `go-go-goja-ui` | `ui.dsl`, `ui` | Already runtime-aware. |
| `go-go-goja/modules/express` | `RegisterRuntimeModules` | `go-go-goja-web` | likely `express` | Needs runtime lifecycle/server semantics review. |
| `go-go-goja/pkg/docaccess/runtime` | `RegisterRuntimeModules` | `go-go-goja-docaccess` | doc module name from registrar config | Good host/runtime context candidate. |
| `cozodb-goja/pkg/cozoapi/module` | `Module` with `Loader` | `cozodb` | `cozodb` | Strong first external repo target. Config should select in-memory/path-backed DB. |
| `smailnail/pkg/js/modules/smailnail` | `modules.NativeModule` | `smailnail` | `smailnail` | Good if mail credentials/config are explicit. |
| `workspace-manager/pkg/wsmjs/module` | `Register(reg, opts)` | `workspace-manager` | package constant `ModuleName` | Needs config-to-options adapter. |
| `pinocchio/pkg/js/modules/pinocchio` | `Register(reg, opts)` | `pinocchio` | package constant `ModuleName` | Needs config-to-options adapter. |
| `devctl/pkg/logjs` | Goja logging module | `devctl-logjs` | likely logging module | Verify exported API and module name. |
| `goja-git` | `GitModule` | `goja-git` | likely `git` | Good small provider candidate after API inspection. |

### Tier 2: Multi-module provider sets

These packages have several modules and should be converted as provider sets rather than one-off wrappers.

| Repository / package | Current shape | Candidate provider package ID | Module set | Notes |
| --- | --- | --- | --- | --- |
| `loupedeck/runtime/js` | Existing `Registrar.RegisterRuntimeModules` plus per-module `Register` functions | `loupedeck` | `easing`, `ui`, `gfx`, `anim`, `present`, `state`, `metrics`, `scene_metrics` | Very good target. Preserve runtime context and host environment bridge. |
| `goja-github-actions/pkg/modules/*` | Several module packages | `github-actions` | `core`, `github`, `exec`, `io`, `ui`, `workflows` | Needs config for GitHub token/context and security policy around `exec`/IO. |
| `geppetto/pkg/js/modules/geppetto` | Rich module with options and runtime bridge | `geppetto` | `geppetto` plus sub-APIs | High value; requires host services/config for engines, profiles, tools, sessions. |
| `openai-app-server/pkg/js` | App runtime registers modules | `openai-app-server` or split providers | `codex`, `approval`, `rpc`, `ui`, `clock` | App-coupled; define host interfaces before provider wrapper. |
| `esp32-s3-m5/zigctl/pkg/jsruntime/zigctlmod` | `modules.NativeModule` | `zigctl` | `zigctl` | Device/network capability; config must describe connection settings and safety. |

### Tier 3: Internal or app-coupled packages

These require package movement, public adapter packages, or same-module generated binaries.

| Repository / package | Constraint | Recommended path |
| --- | --- | --- |
| `css-visual-diff/internal/cssvisualdiff/jsapi` | `internal/` package; exposes `css-visual-diff` | Move reusable module surface to `pkg/cssvisualdiff/jsapi` or add a public provider package in the same module. |
| `css-visual-diff/internal/cssvisualdiff/dsl` | `internal/`; exposes `diff` and `report` | Same as above; define report/diff host dependencies explicitly. |
| `discord-bot/internal/jsdiscord` | `internal/` and bot-host-coupled | Do not convert first. Extract stable public UI/Discord binding package only after host-service interfaces are clear. |
| `plz-confirm/internal/scriptengine` | `internal/`; runtime engine not module provider | Convert only if there is a public JS API worth exposing. |
| `scraper/pkg/js/runtime` | Runtime/executor package, not a standalone module | Identify specific modules first; avoid wrapping whole runtime. |
| `js-analyzer/pkg/jsanalyzer/runtime` | Analyzer runtime support | Treat as tooling/runtime support unless a `require()` API emerges. |
| `go-minitrace/cmd/go-minitrace/cmds/query/js_runtime.go` | Command-local JS runtime | Not a provider target unless query helpers are factored into public module package. |

## Provider Adapter Patterns

### Pattern A: Existing `modules.NativeModule`

Use this for `go-go-goja/modules/*`, `smailnail`, and similar modules.

```go
func Register(registry *providerapi.Registry) error {
    return registry.Package("my-package",
        providerapi.Module{
            Name:        mod.Name(),
            DefaultAs:   mod.Name(),
            Description: "...",
            New: func(ctx providerapi.ModuleContext) (require.ModuleLoader, error) {
                return mod.Loader, nil
            },
        },
    )
}
```

If module constructors accept options, decode `ctx.Config`:

```go
var cfg Config
if len(ctx.Config) > 0 {
    if err := json.Unmarshal(ctx.Config, &cfg); err != nil {
        return nil, fmt.Errorf("decode config: %w", err)
    }
}
mod := New(WithConfig(cfg))
return mod.Loader, nil
```

### Pattern B: Existing `Register(reg *require.Registry, opts Options)`

Use this for packages such as `pinocchio` and `workspace-manager`.

Refactor the source package to expose a loader factory without forcing registration:

```go
func NewLoader(opts Options) require.ModuleLoader {
    mod := &module{opts: opts}
    return mod.Loader
}
```

Then provider wrapper:

```go
providerapi.Module{
    Name:      ModuleName,
    DefaultAs: ModuleName,
    New: func(ctx providerapi.ModuleContext) (require.ModuleLoader, error) {
        opts, err := optionsFromConfig(ctx.Config)
        if err != nil {
            return nil, err
        }
        return NewLoader(opts), nil
    },
}
```

### Pattern C: Existing `RegisterRuntimeModules(ctx, reg)`

Use this for modules that already require runtime context, owner bindings, runtime-loaded module metadata, or host lifecycle information.

If the registrar can be constructed from config:

```go
providerapi.Module{
    Name:      "ui.dsl",
    DefaultAs: "ui",
    New: func(ctx providerapi.ModuleContext) (require.ModuleLoader, error) {
        registrar := NewRegistrar(optionsFrom(ctx))
        // Prefer exposing one loader factory per require name if possible.
        return registrar.LoaderFor("ui.dsl"), nil
    },
}
```

If the existing registrar only registers many modules at once, add a thin provider adapter package that exposes one `providerapi.Module` per registered module name and reuses the same underlying loader constructors. Avoid invoking broad registrar registration from every module factory because that obscures runtime profile policy.

### Pattern D: Provider-shipped JS verbs

If the package contains JS verbs, embed and register them:

```go
//go:embed verbs/*.js
var verbsFS embed.FS

func Register(registry *providerapi.Registry) error {
    return registry.Package("my-package",
        providerapi.Module{...},
        providerapi.VerbSource{
            Name:        "verbs",
            Description: "Bundled JavaScript verbs",
            Root:        "verbs",
            FS:          verbsFS,
        },
    )
}
```

Then generated specs can mount them:

```yaml
jsverbs:
  - id: provider-defaults
    package: my-package
    source: verbs
```

### Pattern E: Host-coupled modules

Use `providerapi.ModuleContext.Config` for JSON-serializable static configuration and `providerapi.ModuleContext.Host` for live host services. The generated pure xgoja binary should be able to use config-only modules. Target-mode integrations may pass richer host services.

Recommended shape:

```go
type Config struct {
    RootDir string `json:"rootDir,omitempty"`
    TokenEnv string `json:"tokenEnv,omitempty"`
    AllowExec bool `json:"allowExec,omitempty"`
}

type HostServices interface {
    Logger() *zerolog.Logger
    Client(name string) (any, bool)
}
```

Rules:

- Do not read credentials directly from global environment unless config explicitly names an env var.
- Do not perform filesystem or process operations unless config enables them.
- Return clear provider construction errors before runtime execution where possible.
- Keep module loaders deterministic for a given config.

## Implementation Phases

### Phase 1: Inventory and classification

Tasks:

1. Re-run and save a reproducible inventory script under the ticket `scripts/` directory.
2. For each candidate, record:
   - repository path,
   - Go module path,
   - importable package path,
   - current Goja registration pattern,
   - module names,
   - config/options type,
   - whether it uses `internal/`,
   - whether it needs runtime owner/event loop/context,
   - security level.
3. Produce a provider conversion table with statuses:
   - `ready-loader`,
   - `ready-register-adapter`,
   - `ready-runtime-registrar-adapter`,
   - `needs-public-package`,
   - `needs-host-interface`,
   - `defer`.

Deliverables:

- Updated inventory section in this guide or a separate reference doc.
- Script in `ttmp/.../scripts/01-inventory-goja-bindings.*`.

### Phase 2: Core adapter helpers and conventions

Tasks:

1. Decide where provider wrappers live:
   - inside each source repo, or
   - in `go-go-goja/pkg/xgoja/providers/...` for first-party modules only.
2. Define naming conventions:
   - provider package IDs,
   - module names,
   - default aliases,
   - config schema naming.
3. Add helper functions if needed:
   - config decode helper,
   - `modules.NativeModule` to `providerapi.Module` adapter,
   - multi-alias module helper for `fs`/`node:fs` style pairs.
4. Add documentation for provider authoring in xgoja builder docs.

Deliverables:

- Provider adapter helper package if justified.
- Design note explaining wrapper location and naming decisions.

### Phase 3: First-pass simple providers

Start with packages that need minimal refactor and have low-risk smoke tests.

Suggested order:

1. `go-go-goja/modules/path`, `yaml`, `time`, `crypto`, `events`, `timer` as a core provider set.
2. `cozodb-goja/pkg/cozoapi/module` as an external repo provider.
3. `workspace-manager/pkg/wsmjs/module` or `pinocchio/pkg/js/modules/pinocchio` as `Register(reg, opts)` adapters.
4. `goja-git` and `devctl/pkg/logjs` after confirming public API shape.

Tasks per provider:

1. Add or refactor public loader factory if needed.
2. Add `providerapi.Register` wrapper.
3. Add config struct and config schema when options exist.
4. Add a minimal generated xgoja smoke example:

```yaml
packages:
  - id: provider-id
    import: module/path/providerpkg

runtimes:
  main:
    modules:
      - package: provider-id
        name: module-name
        as: module-name

commands:
  run:
    enabled: true
    runtime: main
```

5. Add a `run.js` script that requires the module and exercises one deterministic function.
6. Validate with generated build and run.

Deliverables:

- One provider wrapper per package.
- One xgoja example or generated-build test per provider.

### Phase 4: Multi-module provider sets

Convert packages with several related modules.

Suggested order:

1. `loupedeck/runtime/js` provider set.
2. `goja-github-actions/pkg/modules/*` provider set.
3. `geppetto/pkg/js/modules/geppetto` provider.
4. `esp32-s3-m5/zigctl/pkg/jsruntime/zigctlmod` provider if device config can be made safe.

Tasks per provider set:

1. Extract module loader factories where modules currently only expose registry registration.
2. Register each module as a separate `providerapi.Module` so runtime profiles can select them individually.
3. Add config structs for shared provider settings.
4. Add per-module or provider-level docs describing available `require()` names.
5. Add smoke tests for at least one low-risk function per module.
6. For async modules, verify runtime owner/event-loop behavior through generated `run` or jsverb invocation.

Deliverables:

- Multi-module provider wrappers.
- Provider docs listing module names, aliases, config, and risks.

### Phase 5: Internal and app-coupled bindings

Handle packages that cannot be imported directly or need live host services.

Tasks:

1. For `css-visual-diff`, decide whether to:
   - move reusable JS API packages from `internal/` to `pkg/`, or
   - generate xgoja binaries inside the same module only.
2. For `openai-app-server`, define host-service interfaces for `codex`, `approval`, `rpc`, `ui`, and `clock`.
3. For `discord-bot`, decide whether UI/Discord builders are general enough to extract from `internal/jsdiscord`.
4. For `plz-confirm`, identify whether the script engine contains reusable module APIs or just runtime execution machinery.
5. For `scraper`, identify specific JS modules rather than wrapping the whole executor.
6. For each extracted module, repeat Phase 3 or Phase 4 provider tasks.

Deliverables:

- Public package move plan or adapter strategy per internal package.
- Host-service interface sketches for app-coupled providers.

### Phase 6: Security, configuration, and API review

Tasks:

1. Classify each provider capability:
   - pure/data-only,
   - filesystem read,
   - filesystem write,
   - process execution,
   - network/API access,
   - credentials/secrets,
   - hardware/device control,
   - long-running server/listener.
2. Require config for dangerous capabilities.
3. Document default-deny behavior where appropriate.
4. Ensure config parsing fails closed.
5. Review async modules for owner-thread correctness.
6. Review generated examples to avoid embedding secrets or local-only absolute paths.

Deliverables:

- Provider security matrix.
- Review checklist for provider PRs.

### Phase 7: Validation and closure

Tasks:

1. Run focused tests for modified packages.
2. Run generated xgoja smoke tests for each provider example.
3. Run examples with `GOWORK=off` where the workspace goja mismatch still applies.
4. Run `docmgr doctor --ticket XGOJA-006 --stale-after 30`.
5. Update the diary and changelog after each implementation tranche.
6. Close the ticket only after provider docs, smoke tests, and security matrix are complete.

Deliverables:

- Passing validation logs recorded in diary.
- Closed ticket with changelog entry.

## Detailed Task Checklist

### Planning

- [ ] Save inventory script under `scripts/01-inventory-goja-bindings.*`.
- [ ] Add provider classification table to docs.
- [ ] Decide first implementation tranche.
- [ ] Relate all modified provider files with `docmgr doc relate`.

### Adapter conventions

- [ ] Define provider ID naming convention.
- [ ] Define module alias convention.
- [ ] Define config-schema convention.
- [ ] Decide whether to add provider helper functions in `pkg/xgoja/providerapi` or a separate package.

### Simple providers

- [ ] Add first-party core provider for safe `go-go-goja/modules/*` modules.
- [ ] Add host-capability provider for `fs`, `exec`, `database` only with explicit config/security docs.
- [ ] Add `cozodb-goja` provider wrapper.
- [ ] Add `workspace-manager` provider wrapper.
- [ ] Add `pinocchio` provider wrapper.
- [ ] Add `smailnail` provider wrapper.
- [ ] Add `goja-git` provider wrapper.
- [ ] Add `devctl-logjs` provider wrapper.

### Multi-module providers

- [ ] Add `loupedeck` provider set.
- [ ] Add `goja-github-actions` provider set.
- [ ] Add `geppetto` provider.
- [ ] Add `zigctl` provider if safe config is available.

### Internal/app-coupled providers

- [ ] Plan public extraction for `css-visual-diff` JS APIs.
- [ ] Plan host-service interfaces for `openai-app-server` JS modules.
- [ ] Decide whether to extract `discord-bot/internal/jsdiscord` APIs.
- [ ] Decide whether `plz-confirm/internal/scriptengine` should expose a module provider.
- [ ] Identify provider-sized pieces in `scraper/pkg/js/runtime`.

### Tests and examples

- [ ] Add generated-build tests for provider wrapper packages.
- [ ] Add runnable `examples/xgoja/providers/...` examples or per-repo examples.
- [ ] Add `run.js` smoke scripts for deterministic APIs.
- [ ] Add jsverb examples for packages with provider-shipped verbs.
- [ ] Add CI-friendly Makefile targets.

### Docs

- [ ] Update xgoja provider authoring docs.
- [ ] Add provider catalog reference doc.
- [ ] Add provider security matrix.
- [ ] Update README/examples with provider usage.

## Validation Commands

Use focused package tests while developing:

```bash
GOWORK=off go test ./pkg/xgoja/providerapi ./pkg/xgoja/app ./cmd/xgoja/internal/generate ./cmd/xgoja/internal/buildspec ./cmd/xgoja -count=1
```

Use generated example smokes for each provider:

```bash
make -C examples/xgoja/providers/<provider-name> smoke
```

Use docmgr hygiene before closing:

```bash
docmgr doctor --ticket XGOJA-006 --stale-after 30
```

## Review Checklist

For each provider PR, reviewers should check:

- The provider has a stable public `Register(*providerapi.Registry) error` function.
- Each module can be selected independently by runtime profile.
- Config decoding is explicit and fails closed.
- Dangerous capabilities are disabled unless config enables them.
- Async callbacks and promises use runtime owner/event-loop APIs correctly.
- The provider does not install unrelated global modules.
- Generated xgoja smoke tests prove `require()` works in a generated binary.
- Docs list module names, aliases, config fields, and security implications.

## Open Questions

- Should first-party `go-go-goja/modules/*` expose one aggregate provider package or multiple capability providers such as `core`, `host`, `database`, and `web`?
- Should provider config schemas be free-form JSON examples for now, or should `providerapi` grow a stronger schema type?
- Should generated xgoja support provider-level default runtime profiles, or should every runtime profile remain fully explicit?
- Should app-coupled providers use `ModuleContext.Host` immediately, or should they wait for target-mode host-service examples?

## Implementation Note: First-Party Core and Host Providers

The first provider tranche implements two first-party provider packages inside `go-go-goja`:

- `pkg/xgoja/providers/core` with provider package ID `go-go-goja-core`.
- `pkg/xgoja/providers/host` with provider package ID `go-go-goja-host`.

### Core provider

The core provider exposes data-oriented modules from `go-go-goja/modules/*`:

- `path`
- `node:path`
- `yaml`
- `crypto`
- `node:crypto`
- `time`
- `timer`
- `events`
- `node:events`

The provider adapts the existing `modules.NativeModule` registry. It imports the relevant module packages for their `init()` registration, looks up each module by name, and exposes it as a `providerapi.Module`. This preserves the existing loader implementations while making each module selectable by an xgoja runtime profile.

### Host provider

The host provider exposes host-capability modules separately from the core provider:

- `fs`
- `node:fs`
- `exec`
- `database`
- `db`

The host provider intentionally requires explicit config for dangerous capabilities:

```yaml
runtimes:
  main:
    modules:
      - package: go-go-goja-host
        name: fs
        as: fs
        config:
          allow: true
      - package: go-go-goja-host
        name: exec
        as: exec
        config:
          allow: true
          allowedCommands:
            - echo
      - package: go-go-goja-host
        name: database
        as: database
        config:
          allowConfigure: true
```

Security semantics:

- `fs` and `node:fs` require `config.allow: true`. This is an explicit acknowledgement gate; it is not a path sandbox.
- `exec` requires `config.allow: true`. If `allowedCommands` is non-empty, exact command names must appear in the allow-list.
- `database` and `db` disable JavaScript `configure()` by default. Set `config.allowConfigure: true` to let JavaScript open driver/data-source pairs such as sqlite3 in-memory databases.

### Examples

Two generated-binary smoke examples exercise these providers:

- `examples/xgoja/core-provider`
- `examples/xgoja/host-provider`

Run them with:

```bash
make -C examples/xgoja/core-provider smoke
make -C examples/xgoja/host-provider smoke
```
