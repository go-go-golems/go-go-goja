---
Title: "Research Logbook for Architecture Analysis"
Ticket: GOJA-053
Status: active
Topics:
    - xgoja
    - architecture
    - plugin
    - codegen
DocType: reference
Intent: long-term
WhatFor: "Track resources consulted for the xgoja architecture analysis"
LastUpdated: 2026-06-03
---

# Research Logbook: xgoja Architecture Analysis

## 1. `cmd/xgoja/main.go` — xgoja CLI entry point

- **What I was researching:** How the xgoja CLI works, what commands it exposes
- **Looking for:** The build subcommand, how it loads the spec, and how it triggers code generation
- **Why I chose it:** Need to understand the full build pipeline from spec → code → binary
- **How I found it:** Direct file in the cmd/xgoja directory
- **Found useful:** Entry point just calls `newRootCommand()` which wires up `build`, `list`, `inspect`, `doctor` subcommands. The `build` command (`cmd_build.go`) loads the spec, calls `generate.WriteAll()`, then `go build`.
- **Not useful:** None — this was the key starting point
- **What is wrong / out of date:** Nothing
- **Needs updating:** Nothing

**Rating:** 🟢 Current

---

## 2. `cmd/xgoja/internal/generate/main.go` — Code Generation Engine

- **What I was researching:** How the code generation actually works
- **Looking for:** The template rendering, how spec data flows into generated code
- **Why I chose it:** This is the core of the code generation pipeline
- **How I found it:** Direct file in the cmd/xgoja/ tree
- **Useful:**
  - `RenderMain()` applies the `main.go.tmpl` template with data from the Spec
  - `RenderEmbeddedSpec()` serializes the Spec to JSON and embeds it as a constant
  - Import aliases are computed from package IDs to avoid conflicts
  - `RenderGoMod()` generates the `go.mod` file
- **Not useful:** The template file itself is heavily tied to the single-binary output
- **Out of date / wrong:** The template only supports one output format (standalone binary). No way to target a library or WASM.
- **Needs updating:** Template should be pluggable — a user who wants a library target instead of a binary would need to write their own template.

**Rating:** 🟢 Current but limited

---

## 3. `cmd/xgoja/internal/buildspec/spec.go` — The Spec Model

- **Researching:** The data model that xgoja.yaml deserializes into
- **Looking for:** The Spec, Runtime, ModuleInstance, PackageSpec types
- **Why:** The spec drives the entire pipeline
- **How I found it:** Referenced from generate/main.go
- **Useful:**
  - `Spec` struct has `Name`, `Packages`, `Runtimes`, `Commands`, `CommandProviders`, `JSVerbs`, `Help`, `Assets`
  - `Runtime` contains `[]ModuleInstance`
  - `ModuleInstance` has Package, Name, Alias, Config
  - `PackageSpec` has `Import` (Go import path) and `Register` (the registration function name)
- **Not useful:** The `Target` struct (TargetSpec) has fields `Kind`, `Import`, `Version`, `Output`, `Root` — but only `binary` kind is really exercised
- **Out of date / wrong:** Target.Kind only handles "binary" and "adapter" well; other targets aren't implemented
- **Needs updating:** Consider formalizing Target as an interface so plugins can contribute new output targets

**Rating:** 🟢 Current

---

## 4. `cmd/xgoja/internal/generate/templates/main.go.tmpl` — Code Generation Template

- **What I was researching:** The actual template that produces `main()`
- **Looking for:** How providers get wired into the generated binary
- **Why I chose it:** This is the single most important file — it defines what the user's compiled binary actually does at runtime
- **How I found:** Referenced from generate/main.go
- **Found useful:**
  - Provider packages are imported by their computed alias
  - `must({alias}.Register(registry))` is called for each
  - If embedded, the spec JSON is a constant, otherwise read from file at startup
  - `app.NewRootCommand()` creates the Host and Cobra root command
- **Not useful:** Template only generates standalone binary — no library mode
- **Out of date/wrong:** No template extensibility — you'd have to modify the template itself
- **Needs updating:** Support alternative targets (library, WASM, plugin)

**Rating:** 🟡 Partially current — needs extension for multi-target

---

## 5. `pkg/xgoja/app/factory.go` — Runtime Factory

- **Researching:** How the generated binary creates JS runtimes at runtime
- **Looking for:** How the factory assembles modules from spec
- **Why I chose it:** This is the bridge between code generation (static) and runtime (dynamic)
- **Found useful:**
  - `providerRuntimeModuleSpec` is the glue between provider registrations and the engine
  - `NewRuntime()` builds the `engine.Factory`, adds all `providerRuntimeModuleSpec` entries, and creates a runtime
  - The `NewRuntimeFromSections` addition (GOJA-052) would go here
- **Not useful:** Nothing — the file is essential.
- **Out of date/wrong:** Nothing
- **Needs updating:** Add `NewRuntimeFromSections()` as designed in GOJA-052

**Rating:** 🟢 Current

---

## 6. `pkg/xgoja/providerapi/registry.go` — Provider Registry

- **Researching:** The actual provider registration system
- **Looking for:** How Register() builds the internal Package map
- **Why I chose it:** Provider packages call `registry.Package()` with their entries — I needed to trace this chain
- **Found useful:**
  - `Package` struct has `Modules`, `PackageCapabilities`, `VerbSources`, `HelpSources`, `CommandSetProviders`
  - Entry deduplication ensures the same package can't be registered twice
  - `ResolveModule()` resolves `(packageID, moduleName) → Module`
- **Not useful:** Nothing
- **Out of date/wrong:** Nothing
- **Needs updating:** None

**Rating:** 🟢 Current

---

## 7. `devctl` Plugin System (go-go-golems/devctl)

- **Researching:** Plugin architecture from devctl for comparison
- **Looking for:** A proven plugin/host model that could be adapted for xgoja
- **Why I chose it:** devctl already has a working plugin model with handshake protocol
- **Found useful:**
  - Plugin = subprocess, communicates over JSON-RPC over stdio
  - Handshake protocol defines ops the plugin supports
  - `engine.Pipeline` sends config mutations, launch plans, etc. to each plugin in sequence
  - Each plugin can `MutateConfig`, `LaunchPlan`, etc.
- **Not useful:** Nothing specific, but the scope of devctl plugins (dev services) is very different from xgoja's needs
- **Out of date / wrong:** Nothing wrong, but devctl's plugin model is built for a CLI multi-service orchestrator, not an extensible JS runtime. The concepts are similar but the details differ.
- **Needs updating:** Could benefit from a more general plugin spec (gRPC, WASM, or out-of-process)

**Rating:** 🟢 Current (for what it is)

---

## 8. `go-go-golems/devctl/pkg/runtime/factory.go`

- **Researching:** The `Factory` and `Start()`/`Close()` pattern for subprocess plugins
- **Found useful:** `Factory` returns `Client` instances that talk to plugin processes
- **Not useful:** Process management is specific to devctl's needs

**Rating:** 🟢 Current

---

## 9. `engine/factory.go` — Engine-Level Runtime

- **Researching:** How the engine-level Factory works
- **Found useful:**
  - The engine `Factory` builds a `*engine.Runtime` from a set of `RuntimeModuleSpec`s
  - `NewRuntime()` creates the VM, adds all registered modules, and starts the event loop
  - Runtime initializer hooks run after runtime is created

- **Not useful:** The engine layer doesn't know about providers — it's the lowest layer
- **Rating:** 🟢

---

## 10. `cmd/xgoja/cmd_build.go` — Build Command

- **Researching:** How the build command works end-to-end
- **Found useful:**
  - Loads spec → validates → generates code in temp dir → runs `go build`
  - The `xgoja build` command does not call `go build` itself — it generates files and invokes the Go toolchain
- **Not useful:** Nothing
- **Rating:** 🟢

---

## Summary

| # | Resource | Rating | Key Takeaway |
|---|---|---|---|
| 1 | xgoja CLI entry | 🟢 | Simple, subcommands: build, list, doctor, inspect |
| 2 | Code generation engine | 🟢 | Template-based, extensible by replacing template |
| 3 | BuildSpec spec model | 🟢 | Maps 1:1 to xgoja.yaml |
| 4 | RuntimeFactory (app/factory.go) | 🟢 | Central to all runtime creation |
| 5 | Provider Registry | 🟢 | Package-level registration |
| 6 | devctl plugin system | 🟢 | Proven pattern, needs adaptation for JS runtime context |
| 7 | Engine Factory | 🟢 | Low-level runtime construction |
| 8 | Build command | 🟢 | Generates + compiles in temp dir |
