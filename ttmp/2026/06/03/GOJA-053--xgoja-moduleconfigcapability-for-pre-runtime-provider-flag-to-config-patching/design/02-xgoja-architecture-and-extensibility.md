---
Title: "xgoja Architecture Analysis: Refactoring Toward Pluggable Extensibility"
Ticket: GOJA-053
Status: active
Topics:
    - xgoja
    - architecture
    - refactoring
    - plugin-system
    - code-generation
DocType: design
Intent: long-term
WhatFor: "Understand the full xgoja pipeline and identify refactoring targets for extensibility"
LastUpdated: 2026-06-03
---

# xgoja Architecture Analysis: Extensibility, Plugin System, and Code Generation

This document provides a comprehensive, intern-friendly reference for the xgoja build pipeline, its code generation strategy, and opportunities for refactoring toward a pluggable, scriptable architecture.

---

## 1. Architecture Overview

The xgoja system has two distinct phases: **build time** (code generation) and **runtime** (running the generated binary).

### 1.1 The Build-Time Pipeline

```
                    xgoja.yaml (spec)
                        │
                        ▼
                  ┌──────────────────┐
                    │   xgoja build   │
                    │  (cmd/xgoja)    │
                    └───────┬──────────┘
                            │
         ┌────────────────────┼─────────────────────┐
         ▼                  ▼                    ▼
   cmd/xgoja/internal/   Generated      go.mod + .go files
   generate/template.go   (main.go)
                              │
                              ▼
                     `go build -o <output>`
```

The `xgoja build` command:

1. **Loads the spec** from `xgoja.yaml` via `internal/buildspec.Load()`.
2. **Generates** a complete Go program (`main.go`, `go.mod`, `go.sum`).
3. **Compiles** the generated program with `go build`, producing a self-contained binary.

The `xgoja` CLI itself never runs JavaScript — it only **generates and compiles a Go program that, when run, creates a JavaScript runtime.

### 1.2 Runtime Pipeline (what the generated binary does)

When a user runs the generated binary:

```
main() →
  provider packages Register() → providerapi.Registry
  ↓
  app.Host → attach commands →
    cobra root → eval / run / repl / jsverbs
  ↓
RuntimeFactory.NewRuntime() → engine.Factory → NewRuntime → JS interpreter alive
```

### 1.3 The Generated main.go

The template `cmd/xgoja/internal/generate/templates/main.go.tmpl` produces a file like:

```go
package main

import (
    "encoding/json"
    "fmt"
    "os"

    "github.com/go-go-golems/go-go-goja/pkg/xgoja/app"
    "github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
    core "github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/core"
    geppetto "github.com/go-go-golems/geppetto/pkg/js/modules/geppetto/provider"
    // ... more imports
)

const embeddedSpecJSON = `{"name":"myapp",...}

func main() {
    registry := providerapi.NewRegistry()
    geppetto.Register(registry)
    // ... more provider registrations
    spec, _ := decodeSpec()
    host := app.NewHostWithOptions(registry, spec, opts...)
    root := app.NewRootCommand(...)
    root.Execute()
}
```

### 1.4 Key Data Structures

| Structure | Purpose | File |
|---|---|---|
| `buildspec.Spec` | The xgoja.yaml contents deserialized | `cmd/xgoja/internal/buildspec/spec.go` |
| `providerapi.Package` | Runtime provider package with modules, capabilities, verb sources | `pkg/xgoja/providerapi/registry.go` |
| `ModuleDescriptor` | Per-module-instance metadata including capabilities | `pkg/xgoja/app/module_sections.go` |
| `RuntimeFactory` | Creates JS runtimes with given profile selection | `pkg/xgoja/app/factory.go` |

## 2. The Build System in Detail

### 2.1 xgoja.yaml → Spec

The `xgoja build` command:

1. Reads `xgoja.yaml`, parses it into `buildspec.Spec`:
   - `Name`, `AppName`, `EnvPrefix`, `Go.Version`
   - `Packages []PackageSpec` — each with `Import`, `Register` (the Go function that registers the package)
   - `Runtimes` — maps runtime profile names to their module lists
   - `Commands` (eval, run, repl, jsverbs) with `Enabled`, `Runtime`, `Name`
   - `CommandProviders`, `JSVerbs`, `Help`, `Assets`

2. Calls `generate.WriteAll(workdir, spec, opts)`.

### 2.2 Code Generation

The code generator (`cmd/xgoja/internal/generate`) has:

| Component | What it produces |
|---|---|
| `RenderMain()` | `main.go` using `main.go.tmpl` |
| `RenderGoMod()` | `go.mod` file |
| `RenderMain()` | The `func main()` that wires up the registry |

The template renders a complete `main.go` that:
- Imports all provider packages (the `Register()` function of each provider is called)
- Embeds the spec as JSON constant
- Creates `app.Host`, builds a `root` cobra command, and executes it

### 2.3 The Spec as the Single Source of Truth

The `Spec` (YAML) controls everything:

```yaml
name: myapp
target: { kind: "binary" }
packages:
  - id: core
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/core
    register: Register
  - id: geppetto
    import: github.com/go-go-golems/geppetto/pkg/js/modules/geppetto/provider
    register: Register
runtimes:
  main:
    modules:
      - package: core
        name: goja-core
      - package: geppetto
        name: geppetto
```

Each entry in `runtimes.<name>.modules` becomes a `ModuleInstance` in the generated binary's runtime factory.

## 3. Runtime Architecture: How the Generated Binary Works

### 3.1 Provider Registration

When the generated `main()` runs:

1. It calls each provider's `Register(registry)` function.
2. Each `Register` function does `registry.Package(PackageID, entries...)` with modules, capabilities, verb sources, etc.
3. The `providerapi.Registry` now holds all packages, their modules, and capabilities.

### 3.2 Host Initialization

```go
host, err := app.NewHostWithOptions(registry, spec, hostOptions)
```

The `Host` object ties the spec, registry, and factory together. When a user runs a command (e.g., `mybinary eval "1+1"`), the following flow executes:

1. `eval` command runs
2. `factory.NewRuntime(ctx, profile, ...)` creates a JS runtime with all the selected modules
3. If sections are present, `initRuntimeFromSections` runs, calling all `RuntimeInitializerCapability.InitRuntimeFromSections()`
4. The JS expression is evaluated.

### 3.2 How Modules Get Loaded (The `providerRuntimeModuleSpec` Bridge)

```go
// pkg/xgoja/app/factory.go
type providerRuntimeModuleSpec struct {
    instance  ModuleInstance
    module    providerapi.Module
    services HostServices
}

func (s providerRuntimeModuleSpec) RegisterRuntimeModule(ctx *engine.RuntimeModuleContext, reg *require.Registry) error {
    config, _ := json.Marshal(s.instance.Config)
    ctx := providerapi.ModuleContext{
        Name:    s.instance.Alias(),
        As:      s.instance.Alias(),
        Config:  json.RawMessage(config),
        Host:    s.services,
    }
    loader, err := s.module.New(ctx)
    // ...
    reg.RegisterNativeModule(s.instance.Alias(), loader)
}
```

This is the critical bridge:
- `instance.Config` (from `xgoja.yaml`) becomes `ModuleContext.Config`, and
- `Module.New(ModuleContext)` is called, producing the actual JS-land module loader.

## 4. The Capability Model

Capabilities are the extensibility points. Currently, there are three capability types:

| Capability | Purpose | Phase |
|---|---|---|
| `ConfigSectionCapability` | Declares Glazed CLI flags | Build-time & runtime |
| `RuntimeInitializerCapability` | Post-creation hooks | Post-runtime |
| (proposeded) `ModuleConfigCapability` | Patch config before `Module.New()` | Pre-runtime |

Currently all capabilities are **package-scoped** — when you register a package, its capabilities apply to every module in the package, regardless of alias.

## 5. Refactoring Toward Core Abstractions

The system has organically grown into a single-path pipeline: spec → codegen → binary.  To make it pluggable and scriptable, we can identify the following primitives:

### 5.1 Core Primitives

We propose the following set of core abstractions:

```
┌─────────────────────────────────────────────────────────┐
│                 Provider Package                     │
│  ┌───────────┐ ┌──────────┐ ┌───────────┐      │
│  │  Module   │ │Capabilities│ │VerbSource│           │
│  └───────────┘ └──────────┘ └─────────┘           │
│     ▲ A ProviderPackage groups these together       │
└─────────────────────────────────────────────────────────┘
```

- **ProviderPackage**: An atomic collection of Modules + Capabilities.
  Current `providerapi.Package` already approximates this. Formalize it.

- **ModuleSpec**: replaces `providerRuntimeModuleSpec`. A ModuleSpec is
  - A reference (import path, register function)
  - An optional static config
  - Resolved at build time
  - Instanced at runtime via `Module.New(ctx)`

- **Capability**: A generic extension point. Instead of specific interfaces (ConfigSectionCapability, RuntimeInitializerCapability, etc.), define a generic `Capability` interface that can be `Attach()`-ed at runtime. Each capability has a unique ID and a set of lifecycle hooks.

```go
type Capability interface {
    CapabilityID() string
}
```

- **Host**: an inversion-of-control container that holds:
  1. Registry
  2. Spec
  3. Factory

Instead of the `app.Host` being an opaque bag of services, it becomes a typed container:

```go
type Host interface {
    Registry() *providerapi.Registry
    Spec() *app.Spec
    Services() ServiceLocator
}
```

### 5.2 Separation of Concerns

```
Spec File
   │
   ▼
┌─────────────┐
│  CodeGen    │──→ generated main.go → `go build`
└─────┬───────┘
      │
      ▼
┌──────────────┐
│  Plugin     │ ← Shared object, WASM, or JS-based
│  Manager    │
└─────┬──────┘
      │
      ▼
┌──────────────┐
│  Engine     │ ← Runtime: creates VM, registers modules, runs code
└─────────────┘
```

### 5.3 Plugin System

Currently, the only way to extend xgoja is to write a Go package with a `Register()` function, and include it at build time. This is static — the generated binary's capabilities are fixed at compile time.

To support runtime extensibility, we can draw from the devctl plugin model:

**devctl Plugin Architecture (for reference):**
- Each plugin is a subprocess that communicates over JSON-RPC over stdin/stdout.
- The host performs a "handshake" with a versioned protocol, then issues requests.
- Each plugin declares capabilities (ops it supports), which the host routes to it.

Adapting this for xgoja:

```
┌─────────────┐         ┌──────────┐
│  xgoja      │   RPC    │  Plugin  │
│  runtime    │◄──────►│ Process  │
│             │◄───────│          │
└─────────────┘         └──────────┘
```

A plugin would be a Go plugin (`.so` on Linux, compiled with `go build -buildmode=plugin`) or a JSON-RPC/JSON-over-stdio subprocess.

### 5.4 Scripting xgoja with Itself

The most radical extensibility option: **the xgoja generator is itself a xgoja runtime.** If the code generator ran inside a JavaScript runtime, we could script generation using JS. Consider a world where:

```js
// xgoja.config.js
export default {
  providers: ['core', 'geppetto'],
  runtimes: { ... },
  onBuild(spec) { /* transform spec before codegen */ },
}
```

This would let users write build logic in JS. This is essentially "xgoja bootstrapping itself" — xgoja could interpret the config as JS and generate code accordingly.

In the far future, `xgoja.yaml` could be a `.js` config file:
```js
export default {
  name: "myapp",
  packages: {
    "core": { import: "...", register: "Register" },
  },
  runtimes: { ... }
}
```

The JS runtime for this could be the same engine xgoja is built with — goja.

### 5.5 Alternative Code Generation Targets

Today xgoja generates a single Go `main()` with all imports baked in. But the pipeline is:

```
xgoja.yaml → xgoja build → generated code → go build → binary
```

If we separate "spec → generated code" from "generated code → binary," the spec could target different outputs:

| Target | Output | Use Case |
|---|---|---|
| Standalone binary | Current behavior | Standard distribution |
| Go library package | `package myapp` with exported `NewRuntime()` | Embedding xgoja in a larger Go application |
| WASM module | Compiled via TinyGo or wasm-pack | Browser-side JS engine |
| Docker image | Multi-stage Dockerfile + Go build | Cloud deployment |

Each of these targets shares the spec-to-model step but differs in "rendering."

To achieve this, the code generator should be refactored into:

1. **Spec → IR**: Parse the YAML spec into an intermediate representation
2. **IR → Code**: Render the IR with different templates

```
xgoja.yaml
   ↓
[ Spec ]
   ↓
[  IR  ] ← This is the key abstraction. Currently missing.
   ↓
[ Code Generation Backend ]
   ├── Go binary (main.go + go build)
   ├── Go library package
   ├── Docker image (Dockerfile)
   └── WASM component
```

## 6. Plugin System Design

### 6.1 Devctl-Style Plugin System

The devctl system uses a subprocess-based plugin protocol:
1. The host starts the plugin as a child process.
2. The plugin sends a JSON handshake with supported capabilities.
3. The host and plugin communicate via JSON-RPC over stdin/stdout.

For xgoja, a similar model would work:

```go
// PluginHost manages plugin processes
type PluginHost struct {
    plugins []Plugin
}

type Plugin interface {
    Init(ctx context.Context, host Host) error
    Capabilities() []string
    Call(ctx context.Context, method string, args any) (any, error)
}
```

The host sends a handshake request, the plugin responds with its capabilities (which "ops" it supports), and the host routes calls.

### 6.2 Plugin API Surface

A plugin should be able to:
- Register additional modules (like a provider package does today)
- Provide capabilities at the package level
- Hook into the spec → IR step (to allow a plugin to add/modify the spec)

This could work as a gRPC or JSON-RPC protocol. For an initial implementation, a simpler approach:

**Phase 1: WASM Plugins** — Compile plugins as WASM modules, loaded by the xgoja runtime at startup. The host provides a thin ABI for IO, and the WASM module can call back into the host for runtime configuration.

**Phase 2: Out-of-Process Plugins** — For use-cases that need system access or native dependencies, plugins run as subprocesses (like devctl).

## 7. Full System Map

```
                  ┌──────────────────┐
                  │  xgoja.yaml     │
                  └───────┬────────┘
                          │
                          ▼
              ┌───────────────────────┐
              │  xgoja build       │
              │  (cmd/xgoja)       │
              └─────────┬───────────┘
                        │
          ┌─────────────┼───────────────┐
          ▼             ▼              ▼
   ┌──────────┐  ┌─────────────┐  ┌──────────┐
   │main.go   │  │ go.mod     │  │ go.sum   │
   └──────────┘  └─────────────┘  └──────────┘
        │
        ▼
   go build → binary
```

After compilation the generated binary, when run:

```
mybinary [eval|run|repl|verbs] [args]
       │
       └──> RuntimeFactory.NewRuntime(profile)
                   │
                   ┌─────────────┐
                   │ engine      │
                   │ .NewRuntime │
                   └─────┬───────┘
                         │
           ┌───────────┴─────────────┐
           ▼                      ▼
  ┌──────────────┐     ┌─────────────────┐
  │ JS runtime   │     │  Module.New() │
  │  (goja VM)   │◄────│  with merged  │
  └──────────────┘     │  config      │
                       └───────────────┘
```

## 8. Refactoring Roadmap

Based on the analysis above, I recommend these staged refactors:

**Phase 1: Core Abstractions**
- Extract `Spec → IR → Codegen` as a distinct step
- Define `IRSpec` (intermediate representation)
- Make the template rendering pluggable

**Phase 2: Pluggable Backends**
- Add output templates for "library" mode (Go package) and WASM target
- Pull the Go source template out of `cmd/xgoja/internal/generate/` into a registered-template system

**Phase 3: Plugin Runtime**
- Add the plugin host from §6 above
- Support both WASM and out-of-process plugins

**Phase 3: Scripted Config**
- Allow `.js` xgoja config
- Use goja to interpret the build config
- Build: `xgoja build -f xgoja.yaml` → evaluates the config, builds IR, generates Go code.

**Phase 4: Alternative Code Generation**
- CLI target: produce `main.go` for `go build`
- Library target: produce a `go:generate`-compatible Go package
- Docker image build target

## 9. Research Log

| Resource | Freshness | Key Finding |
|---|---|---|
| `cmd/xgoja/internal/buildspec/spec.go` | 🟢 Current | Typed representation of xgoja.yaml |
| `cmd/xgoja/internal/generate/main.go` | 🟢 Current | Entry point for code generation; `RenderMain()`, `RenderEmbeddedSpec()` |
| `cmd/xgoja/internal/generate/templates/main.go.tmpl` | 🟢 | Go template for generated main.go |
| `pkg/xgoja/app/factory.go` | 🟢 | RuntimeFactory: the core of runtime creation |
| `pkg/xgoja/providerapi/*.go` | 🟢 | Provider registration and capabilities |
| `pkg/xgoja/app/host.go` | 🟢 | Wires providers, spec, and factory together |

## 10. Appendix: File Reference

| File | Purpose |
|---|---|
| `cmd/xgoja/main.go` | CLI entry |
| `cmd/xgoja/cmd_build.go` | `xgoja build` subcommand |
| `cmd/xgoja/internal/buildspec/spec.go` | Spec types (Spec, PackageSpec, Runtime, ModuleInstance) |
| `cmd/xgoja/internal/generate/main.go` | Code generation engine |
| `cmd/xgoja/internal/generate/templates/main.go.tmpl` | Template for generated main |
| `pkg/xgoja/app/factory.go` | RuntimeFactory and runtime creation |
| `pkg/xgoja/app/spec.go` | App-level Spec struct |
| `pkg/xgoja/providerapi/*.go` | Registry, Package, Module, Capabilities |

---

*End of analysis document.*
