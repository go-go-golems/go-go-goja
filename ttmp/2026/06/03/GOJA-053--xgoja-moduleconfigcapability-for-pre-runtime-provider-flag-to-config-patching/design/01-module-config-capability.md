---
Title: 'ModuleConfigCapability: Analysis, Design, and Implementation Guide'
Ticket: GOJA-053
Status: active
Topics:
    - xgoja
    - provider
    - capability
    - glazed
    - config
    - geppetto
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../geppetto/pkg/js/modules/geppetto/provider/provider.go
      Note: Motivating use case for ModuleConfigCapability
    - Path: pkg/xgoja/app/factory.go
      Note: RuntimeFactory where NewRuntimeFromSections will be added
    - Path: pkg/xgoja/providerapi/capabilities.go
      Note: Core capability interfaces where ModuleConfigCapability will be added
    - Path: pkg/xgoja/providers/http/http.go
      Note: Reference implementation of both ConfigSectionCapability and RuntimeInitializerCapability
    - Path: pkg/xgoja/providerutil/sections.go
      Note: Utility functions where ModuleConfigPatchFromSections will be added
ExternalSources:
    - 'GitHub Issue: https://github.com/go-go-golems/go-go-goja/issues/52'
Summary: Comprehensive analysis of the xgoja capability model and design for ModuleConfigCapability, enabling provider CLI sections to patch module config before Module.New()
LastUpdated: 2026-06-03T08:00:00-04:00
WhatFor: Implementation guide for a new intern to understand the capability system and implement pre-runtime config patching
WhenToUse: Reference when implementing ModuleConfigCapability, understanding xgoja provider architecture, or adding new provider capabilities
---


# ModuleConfigCapability: Analysis, Design, and Implementation Guide

## Executive Summary

go-go-goja's xgoja system lets you generate standalone JavaScript runtime binaries from a YAML spec. Each binary is composed of **provider packages** that contribute native Go modules, CLI flags, and runtime lifecycle hooks through a **capability interface** system.

Today, providers can expose CLI flags via `ConfigSectionCapability` and run post-creation setup via `RuntimeInitializerCapability`, but there is a gap: **parsed CLI/config/env flag values cannot influence how a module is constructed**. The `Module.New(...)` factory receives only the static config from `xgoja.yaml`, while parsed flag values arrive after the runtime already exists.

This document analyzes the entire capability model, explains every moving part an intern needs to understand, evaluates the `ModuleConfigCapability` proposal from issue #52, identifies code that is confusing or could be refactored, and provides a step-by-step implementation guide.

---

## Table of Contents

1. [Problem Statement](#1-problem-statement)
2. [Current Architecture Deep Dive](#2-current-architecture-deep-dive)
   - 2.1 [The Big Picture: How an xgoja Binary is Born](#21-the-big-picture-how-an-xgoja-binary-is-born)
   - 2.2 [Provider Registry and Package Model](#22-provider-registry-and-package-model)
   - 2.3 [The Capability Interface Hierarchy](#23-the-capability-interface-hierarchy)
   - 2.4 [Runtime Factory and Module Instantiation](#24-runtime-factory-and-module-instantiation)
   - 2.5 [The Glazed Integration: Sections, Flags, and Values](#25-the-glazed-integration-sections-flags-and-values)
   - 2.6 [Built-in Command Flow (eval, run, repl, jsverbs)](#26-built-in-command-flow-eval-run-repl-jsverbs)
   - 2.7 [Command Providers](#27-command-providers)
3. [Gap Analysis](#3-gap-analysis)
4. [The Proposal: ModuleConfigCapability](#4-the-proposal-moduleconfigcapability)
5. [What I Found Confusing](#5-what-i-found-confusing)
6. [Refactoring Opportunities](#6-refactoring-opportunities)
7. [Typing vs. Config HashMaps](#7-typing-vs-config-hashmaps)
8. [Implementation Plan](#8-implementation-plan)
9. [Testing Strategy](#9-testing-strategy)
10. [Risks and Open Questions](#10-risks-and-open-questions)
11. [API Reference](#11-api-reference)
12. [File Reference](#12-file-reference)

---

## 1. Problem Statement

The concrete motivating case is **Geppetto** — the AI inference engine provider. Geppetto needs to expose CLI/config flags like `--geppetto-profile-registries` and `--geppetto-default-profile` that configure how the `geppetto` module is instantiated. These values must be available during `Module.New(...)`, because that is where profile registry loading and `geppettomodule.Options` construction happen.

Currently, the only way to pass these values is through the `xgoja.yaml` static config. A post-runtime `RuntimeInitializerCapability` is too late — the module factory has already run with whatever config was in the YAML file.

**What we want:**

```
CLI flags / env vars / config file values
    ↓ (parsed by Glazed)
    ↓ (patched into ModuleInstance.Config BEFORE Module.New)
Module.New(ModuleContext{Config: merged-config})
    ↓
RuntimeInitializerCapability (for post-creation side effects)
```

**What we have today:**

```
CLI flags / env vars / config file values
    ↓ (parsed by Glazed, but only used AFTER runtime exists)
Module.New(ModuleContext{Config: xgoja.yaml-config-only})
    ↓
RuntimeInitializerCapability (receives parsed values, too late for factory config)
```

---

## 2. Current Architecture Deep Dive

This section is written for someone who has never seen the codebase. Every subsystem is explained with prose, diagrams, file references, and key type signatures.

### 2.1 The Big Picture: How an xgoja Binary is Born

An xgoja binary is a generated Go program that embeds a **spec** (JSON/YAML) describing which provider packages, runtime profiles, commands, and assets to include. The generation flow is:

```
xgoja.yaml (spec file)
    ↓ go generate / code generation
main.go (generated, calls app.NewRootCommand)
    ↓ go build
my-xgoja-binary (standalone CLI)
```

The generated `main.go` looks approximately like:

```go
func main() {
    registry := providerapi.NewRegistry()
    core.Register(registry)       // registers "go-go-goja-core" package
    host.Register(registry)        // registers "go-go-goja-host" package
    http.Register(registry)        // registers "go-go-goja-http" package
    geppetto.Register(registry)    // registers "geppetto" package
    // ... more providers

    specJSON := `{ ... embedded spec ... }`
    root, err := app.NewRootCommand(app.Options{
        Providers:       registry,
        SpecJSON:        specJSON,
        EmbeddedJSVerbs: embeddedFS,
        EmbeddedHelp:    helpFS,
        EmbeddedAssets:  assetsFS,
    })
    // ...
    root.Execute()
}
```

Key points:

- Every provider's `Register()` function populates the `providerapi.Registry` with **entries**: modules, capabilities, verb sources, help sources, and command set providers.
- The embedded spec JSON tells the `app.Host` which packages are active, which runtime profiles exist, and which modules go into each profile.
- The `Host` constructs a `RuntimeFactory` and attaches commands to a Cobra root command.

### 2.2 Provider Registry and Package Model

The provider registry is a simple, flat namespace keyed by **package ID**. Each package contains:

| Entry Type | Purpose | Storage |
|---|---|---|
| `Module` | A `require()`-loadable native module | `pkg.Modules map[string]Module` |
| `PackageCapability` | An optional behavior extension (sections, init, etc.) | `pkg.PackageCapabilities map[string]PackageCapability` |
| `VerbSource` | A filesystem of JavaScript verb definitions | `pkg.VerbSources map[string]VerbSource` |
| `HelpSource` | A filesystem of Glazed help markdown | `pkg.HelpSources map[string]HelpSource` |
| `CommandSetProvider` | A factory for package-owned CLI commands | `pkg.CommandSetProviders map[string]CommandSetProvider` |

**File:** `pkg/xgoja/providerapi/registry.go`

```go
type Package struct {
    ID                  string
    Modules             map[string]Module
    VerbSources         map[string]VerbSource
    HelpSources         map[string]HelpSource
    PackageCapabilities map[string]PackageCapability
    CommandSetProviders map[string]CommandSetProvider
}
```

Entries are registered via the **functional options** pattern. Each entry type implements the `Entry` interface:

```go
type Entry interface {
    applyToPackage(*Package) error
}
```

So `Module`, `VerbSource`, `HelpSource`, `PackageCapability` wrappers, and `CommandSetProvider` all implement `applyToPackage`. The `Registry.Package(id, entries...)` function creates a new package and applies all entries in order.

**Capabilities** are registered via `WithPackageCapability(capability)`:

```go
func WithPackageCapability(capability PackageCapability) Entry {
    return capabilityEntry{capability: capability}
}
```

Important: capabilities are **package-scoped**, not module-scoped. If a package has two modules (e.g., `fs` and `node:fs`), the same capabilities are attached to both when the app layer resolves module descriptors. This is a design choice that avoids N×M capability-to-module wiring but can be confusing.

### 2.3 The Capability Interface Hierarchy

Capabilities are defined in `pkg/xgoja/providerapi/capabilities.go`. The hierarchy is:

```
PackageCapability (marker interface)
    ├── ConfigSectionCapability
    │     ConfigSections(SectionContext) → []schema.Section
    │
    └── RuntimeInitializerCapability
          InitRuntimeFromSections(ctx, vals, RuntimeHandle) → error
```

**`PackageCapability`** is the common marker:

```go
type PackageCapability interface {
    CapabilityID() string
}
```

**`ConfigSectionCapability`** lets a provider declare Glazed sections (groups of flags) that should appear on built-in commands or command provider commands:

```go
type ConfigSectionCapability interface {
    PackageCapability
    ConfigSections(SectionContext) ([]schema.Section, error)
}
```

When the app builds a command, it calls `factory.sectionsForRuntimeProfile()` which iterates over every selected module's package capabilities, type-asserts each to `ConfigSectionCapability`, and collects all sections. These sections become Cobra flags via Glazed.

**`RuntimeInitializerCapability`** lets a provider run code after the runtime has been created:

```go
type RuntimeInitializerCapability interface {
    PackageCapability
    InitRuntimeFromSections(context.Context, *values.Values, RuntimeHandle) error
}
```

The `RuntimeHandle` is a minimal abstraction over the concrete `engine.Runtime`:

```go
type RuntimeHandle interface {
    Runtime() *goja.Runtime
    Close(context.Context) error
}
```

There's also `RuntimeCloserRegistry` for attaching cleanup hooks:

```go
type RuntimeCloserRegistry interface {
    AddCloser(func(context.Context) error) error
}
```

**The `ModuleDescriptor` bridges providers and the app layer:**

```go
type ModuleDescriptor struct {
    PackageID           string
    ModuleID            string
    As                  string
    Module              Module
    PackageCapabilities []PackageCapability
}
```

When the `RuntimeFactory` resolves a runtime profile, it produces one `ModuleDescriptor` per module instance. Every descriptor carries the **full set of package capabilities** from the module's owning package. This means all capabilities from a package are visible on every module from that package, which is how `providerutil.CollectConfigSections` and `providerutil.InitRuntimeFromSections` find and invoke capabilities.

### 2.4 Runtime Factory and Module Instantiation

The `app.RuntimeFactory` is the bridge between the spec and the engine. It lives in `pkg/xgoja/app/factory.go`.

```go
type RuntimeFactory struct {
    providers *providerapi.Registry
    spec      *Spec
    services  providerapi.HostServices
}
```

**`NewRuntime(ctx, profile, opts...)`** does the following:

1. Look up the runtime profile in `spec.Runtimes`.
2. For each `ModuleInstance` in the profile's `Modules` list:
   - Resolve the `Module` from the provider registry.
   - Create a `providerRuntimeModuleSpec` that wraps the instance config.
3. Feed all module specs into `engine.NewBuilder().WithModules(modules...).Build()`.
4. The engine builder creates a `Factory` and calls `Factory.NewRuntime()`.

**The critical path** is in `providerRuntimeModuleSpec.RegisterRuntimeModule()`:

```go
func (s providerRuntimeModuleSpec) RegisterRuntimeModule(ctx *engine.RuntimeModuleContext, reg *require.Registry) error {
    config, err := json.Marshal(s.instance.Config)  // ← ONLY xgoja.yaml config!
    if err != nil {
        return fmt.Errorf("marshal config for %s.%s: %w", s.instance.Package, s.instance.Name, err)
    }
    loader, err := s.module.New(providerapi.ModuleContext{
        Context:      ctx.Context,
        Name:         s.instance.Name,
        As:           s.instance.Alias(),
        Config:       config,        // ← No parsed CLI values here!
        Host:         s.services,
        RuntimeOwner: ctx.Owner,
    })
    // ...
    reg.RegisterNativeModule(s.instance.Alias(), loader)
    return nil
}
```

This is the **single choke point** where `Module.New` receives its config. The config comes exclusively from `s.instance.Config`, which is the `map[string]any` from the YAML spec. No parsed CLI flags or config file values are available here.

### 2.5 The Glazed Integration: Sections, Flags, and Values

Glazed is the CLI framework that powers all xgoja commands. The integration has three stages:

**Stage 1: Section Collection (command construction time)**

```go
moduleSections, _, sectionErr := factory.sectionsForRuntimeProfile("run", profile)
// → collects ConfigSectionCapability sections from all selected modules
// → these become schema.Section objects with fields
```

**Stage 2: Flag Parsing (command execution time)**

Glazed's `CobraParserConfig` handles the plumbing. When a command runs:

1. Cobra parses flags.
2. Glazed middlewares layer in values from: defaults → config file → env → args → Cobra flags.
3. The result is a `*values.Values` object, which is an ordered map of section slugs to `SectionValues`.

**Stage 3: Value Consumption (post-runtime)**

After `factory.NewRuntime()` returns a runtime, the command calls:

```go
initRuntimeFromSections(ctx, vals, rt, selectedModules)
// → iterates over RuntimeInitializerCapability implementations
// → each one receives the parsed vals and can configure the runtime
```

The `values.Values` structure:

```go
type Values struct {
    *orderedmap.OrderedMap[string, *SectionValues]
}

// DecodeSectionInto decodes a named section's field values into a struct:
func (p *Values) DecodeSectionInto(sectionKey string, dst interface{}) error
```

This is how providers extract typed data from parsed flags:

```go
var settings FixtureSettings  // struct with `glazed:"field-name"` tags
vals.DecodeSectionInto("fixture", &settings)
```

### 2.6 Built-in Command Flow (eval, run, repl, jsverbs)

All four built-in commands follow the same pattern. Here's the `run` command as a representative example:

**Construction time:**

```go
func newRunCommand(factory *RuntimeFactory, spec *Spec) cmds.Command {
    profile := commandRuntime(spec.Commands.Run, firstRuntime(spec))
    moduleSections, _, sectionErr := factory.sectionsForRuntimeProfile("run", profile)
    // ... create CommandDescription with moduleSections ...
}
```

**Execution time:**

```go
func (c *runCommand) Run(ctx context.Context, vals *values.Values) error {
    settings := runSettings{}
    vals.DecodeSectionInto(schema.DefaultSlug, &settings)  // file, runtime, keep-alive

    selectedModules, _ := c.factory.selectedModuleDescriptors(settings.Runtime)

    // Step 1: Create runtime (NO parsed flag values influence this!)
    rt, _ := factory.NewRuntime(ctx, profile, requireOpt)

    // Step 2: Post-runtime initialization (parsed flag values available here)
    initRuntimeFromSections(ctx, vals, rt, selectedModules)

    // Step 3: Execute the script
    rt.Require.Require(scriptPath)
}
```

**The timing problem is clear:** between Step 1 and Step 2, `Module.New()` has already run with only the YAML config. Step 2 can only do post-hoc patching (setting globals, starting servers, etc.), not change how the module was constructed.

**For jsverbs**, the pattern is the same but happens inside a verb invoker callback:

```go
func(...) (interface{}, error) {
    rt, _ := factory.NewRuntime(ctx, profile, require.WithLoader(registry.RequireLoader()))
    initRuntimeFromSections(ctx, parsedValues, rt, selectedModules)
    return registry.InvokeInRuntime(ctx, rt, verb, parsedValues)
}
```

**For the TUI REPL**, the runtime is created once and kept alive for the session:

```go
func newXGojaTUIEvaluator(...) (*xgojaTUIEvaluator, error) {
    rt, _ := factory.NewRuntime(ctx, profile)
    initRuntimeFromSections(ctx, vals, rt, selectedModules)
    // ... wrap in JavaScriptEvaluator ...
}
```

### 2.7 Command Providers

Command providers are the most complex path. A `CommandSetProvider` creates its own Glazed commands that may also need xgoja runtimes. The key difference: command providers receive a `CommandSetContext` that includes:

```go
type CommandSetContext struct {
    Context         context.Context
    PackageID       string
    Name            string
    Mount           string
    RuntimeProfile  string
    Config          json.RawMessage
    Host            HostServices
    Providers       *Registry
    RuntimeFactory  RuntimeFactory    // ← can create runtimes
    SelectedModules []ModuleDescriptor
}
```

The command provider can call `ctx.RuntimeFactory.NewRuntime()` to create a runtime for its own commands. Currently, it does NOT pass parsed values to this factory — it only gets the YAML-config-driven behavior.

The `testprovider` package demonstrates the full pattern:

```go
func NewFixtureCommandSet(ctx providerapi.CommandSetContext) (*providerapi.CommandSet, error) {
    sections, _ := sectionsFromSelectedModules(ctx)  // collects sections from capabilities
    commands := []cmds.Command{
        &fixtureBareCommand{CommandDescription: fixtureDescription("bare", "...", sections)},
        // ...
    }
    return &providerapi.CommandSet{Commands: commands}, nil
}
```

When the bare command runs, it has access to parsed `vals` but can only use `RuntimeInitializerCapability` after the fact.

---

## 3. Gap Analysis

| Need | Current Support | Gap |
|---|---|---|
| Declare CLI flags that influence module construction | `ConfigSectionCapability` declares flags but values are only available post-runtime | Values can't reach `Module.New()` |
| Configure modules from config files / env vars | Glazed middlewares parse config/env but feed into `RuntimeInitializerCapability` only | No pre-runtime config patching |
| Pass runtime profile CLI flags to module factory | Not possible — `NewRuntime` has no `values.Values` parameter | No API surface |
| Keep existing post-runtime hooks working | `RuntimeInitializerCapability` works as-is | No gap, but must coexist |

**The fundamental gap:** there is no mechanism to merge parsed Glazed values into `ModuleInstance.Config` before `Module.New()` is called.

---

## 4. The Proposal: ModuleConfigCapability

Issue #52 proposes a new capability interface:

```go
type ModuleConfigCapability interface {
    PackageCapability

    ModuleConfigFromSections(
        ctx context.Context,
        vals *values.Values,
        descriptor ModuleDescriptor,
    ) (map[string]any, error)
}
```

**Semantics:**

1. `ConfigSectionCapability` still declares which sections/flags exist.
2. `ModuleConfigCapability` decodes the parsed `values.Values` and returns a config **patch** (a `map[string]any`).
3. The app layer merges the patch into `ModuleInstance.Config` **before** calling `Module.New()`.
4. `RuntimeInitializerCapability` remains for post-runtime lifecycle work.

**Proposed new factory method:**

```go
func (f *RuntimeFactory) NewRuntimeFromSections(
    ctx context.Context,
    profile string,
    vals *values.Values,
    opts ...require.Option,
) (*JSRuntime, error)
```

The existing `NewRuntime` becomes a thin wrapper:

```go
func (f *RuntimeFactory) NewRuntime(ctx context.Context, profile string, opts ...require.Option) (*JSRuntime, error) {
    return f.NewRuntimeFromSections(ctx, profile, nil, opts...)
}
```

**My evaluation of the proposal:**

The proposal is **sound and well-scoped**. It correctly identifies the gap, proposes a minimal new interface, and preserves backward compatibility. However, I have several observations and refinements:

1. **The `map[string]any` return type is a design smell** — see Section 7 for a detailed discussion. Returning untyped maps forces every consumer to do type assertions and loses compile-time safety. A `json.RawMessage` return or a generic mechanism would be better for the long term.

2. **The capability is well-named** — "ModuleConfig" clearly conveys "this patches module configuration" and sits alongside the existing naming pattern.

3. **Package-scoping is correct for now** — capabilities are already package-scoped, and the `ModuleConfigFromSections` method receives a `ModuleDescriptor` so the capability can decide whether to patch a specific module instance or all instances from the package.

4. **The merge-before-construct approach is the right one** — patching a copy of `ModuleInstance.Config` before passing it to `Module.New` avoids mutating the shared spec and preserves the existing config precedence model.

5. **The `NewRuntimeFromSections` naming is clear** — it signals that parsed section values are involved and that this is a richer version of `NewRuntime`.

---

## 5. What I Found Confusing

### 5.1 Package-Scoped Capabilities vs. Module-Scoped Behavior

Capabilities are registered at the package level but applied to every module from that package. This means:

- If `go-go-goja-host` registers a `ConfigSectionCapability`, it shows up on the `ModuleDescriptor` for **every** host module (`fs`, `node:fs`, `exec`, `database`, `db`).
- The `providerutil` helpers deduplicate by `(packageID, capabilityID)` key, so the capability's `ConfigSections()` and `InitRuntimeFromSections()` are called only once per package, not once per module.

This is correct behavior but **not documented in code** and can be surprising. A comment in `providerapi/capabilities.go` or `providerutil/sections.go` would help.

### 5.2 The Dual `decodeConfig` Pattern

There are two completely separate config decoding paths:

1. **`providerapi.ModuleContext.Config`** — a `json.RawMessage` from the YAML spec, decoded inside `Module.New()` by each provider's own `decodeConfig()` function (e.g., `host.decodeConfig`, `provider.decodeConfig`).

2. **`values.Values`** — parsed from Glazed sections, decoded by `vals.DecodeSectionInto()` with `glazed:` struct tags.

These two paths use **different struct tags** (`json:` vs `glazed:`), **different decoding libraries**, and **different merge semantics**. This means a provider author must maintain two parallel config struct definitions that describe the same conceptual configuration. The `ModuleConfigCapability` proposal doesn't eliminate this duplication — it adds a third struct (the "patch" struct with `glazed:` tags that maps to `json:` keys).

### 5.3 `ModuleInstance.Config` is `map[string]any`, Not Typed

In `app/spec.go`:

```go
type ModuleInstance struct {
    Package string         `json:"package"`
    Name    string         `json:"name"`
    As      string         `json:"as,omitempty"`
    Config  map[string]any `json:"config,omitempty"`
}
```

This config is marshaled to `json.RawMessage` and then unmarshaled by the provider's `decodeConfig`. The intermediate `map[string]any` means:
- Nested objects are `map[string]any` (not typed).
- Arrays are `[]any` (not typed).
- There's no schema validation at the app layer; the provider must validate its own config.

The `ModuleConfigCapability` proposal returns `map[string]any` too, which compounds the issue — merging two untyped maps requires deep-merge logic that is fragile and error-prone.

### 5.4 The Engine Factory vs. The App Factory

There are two "factory" types:

- **`engine.Factory`** — creates low-level runtimes from explicit `RuntimeModuleSpec` lists. Lives in `engine/factory.go`.
- **`app.RuntimeFactory`** — creates runtimes from a spec + provider registry. Delegates to `engine.Factory` internally. Lives in `pkg/xgoja/app/factory.go`.

The naming overlap is confusing. The `app.RuntimeFactory` is really a "spec-driven runtime factory" or "xgoja runtime factory." The `engine.Factory` is the "bare engine factory." The relationship is:

```
app.RuntimeFactory.NewRuntime()
    → resolves spec to ModuleInstances
    → creates providerRuntimeModuleSpec for each
    → feeds into engine.NewBuilder().WithModules(specs...).Build()
    → calls engine.Factory.NewRuntime()
```

### 5.5 The `runtimeHandle` Adapter

In `pkg/xgoja/app/module_sections.go`, there's a private adapter:

```go
type runtimeHandle struct {
    rt *JSRuntime
}
```

This implements `providerapi.RuntimeHandle` by delegating to `engine.Runtime`. The adapter exists because `providerapi` intentionally does not depend on the concrete `engine.Runtime` type. This is good architecture but the thin wrapper is easy to miss — it's defined in a file called `module_sections.go` rather than a dedicated adapter file.

### 5.6 The `SectionContext` Fields

`SectionContext` has five fields:

```go
type SectionContext struct {
    CommandName       string
    CommandProviderID string
    RuntimeProfile    string
    PackageID         string
    ModuleID          string
}
```

Built-in commands set `CommandName` and `RuntimeProfile`. Command providers set `CommandProviderID` and optionally `RuntimeProfile`. The `PackageID` and `ModuleID` are set per-descriptor during iteration. This context is passed to `ConfigSections()` but **not** to `InitRuntimeFromSections()` — the runtime initializer only gets `(ctx, vals, handle)`. If a runtime initializer needs to know which command invoked it, it has to look at the values or use some other side channel.

---

## 6. Refactoring Opportunities

### 6.1 Extract a `ConfigPatch` Helper into `providerutil`

The proposal sketches `moduleConfigPatchFromSections()` as a free function. This should live in `pkg/xgoja/providerutil/` alongside the existing `CollectConfigSections()` and `InitRuntimeFromSections()`. The existing file `providerutil/sections.go` is the natural home.

### 6.2 Unify Config Decoding: Bridge `json:` and `glazed:` Tags

Currently, providers maintain two config structs:

```go
// YAML config struct (decoded from ModuleContext.Config)
type Config struct {
    ProfileRegistries []string `json:"profileRegistries"`
    DefaultProfile    string   `json:"defaultProfile"`
}

// Glazed section struct (decoded from values.Values)
type SectionConfig struct {
    ProfileRegistries []string `glazed:"profile-registries"`
    DefaultProfile    string   `glazed:"default-profile"`
}
```

A possible refactoring: introduce a single config struct with both tags, and a helper that decodes from either source:

```go
type Config struct {
    ProfileRegistries []string `json:"profileRegistries" glazed:"profile-registries"`
    DefaultProfile    string   `json:"defaultProfile" glazed:"default-profile"`
}

func DecodeConfigFromValues(vals *values.Values, section string, dst any) error { ... }
func DecodeConfigFromJSON(data json.RawMessage, dst any) error { ... }
```

This would reduce the maintenance burden of keeping two structs in sync. However, Glazed's `DecodeSectionInto` uses `glazed:` tags for snake_case CLI names, while JSON uses camelCase — the two naming conventions may not always align cleanly.

### 6.3 Replace `map[string]any` with `json.RawMessage` in Config Patch

The proposal returns `map[string]any` from `ModuleConfigFromSections`. A better approach:

```go
type ModuleConfigCapability interface {
    PackageCapability
    ModuleConfigFromSections(
        ctx context.Context,
        vals *values.Values,
        descriptor ModuleDescriptor,
    ) (json.RawMessage, error)
}
```

Why:
- `json.RawMessage` is opaque bytes — the merger doesn't need to understand the structure.
- Deep merge of `json.RawMessage` can be done via `json.Merge` or a simple recursive merge on unmarshaled maps.
- The provider's `Module.New()` already knows how to decode `json.RawMessage` (it does it today from `ModuleContext.Config`).
- No need to expose an untyped map that every consumer must type-assert.

### 6.4 Rename `app.RuntimeFactory` to `app.SpecRuntimeFactory`

To reduce confusion with `engine.Factory`, rename to `SpecRuntimeFactory` or `XGojaRuntimeFactory`. This is a cosmetic change but would help readability.

### 6.5 Move `runtimeHandle` to Its Own File

The `runtimeHandle` adapter in `module_sections.go` should be in a dedicated file like `handle.go` or `runtime_adapter.go`.

### 6.6 Consider a `ModuleCapability` Distinct from `PackageCapability`

Capabilities are currently package-scoped. For `ModuleConfigCapability`, the capability receives a `ModuleDescriptor` so it can decide per-module, but the registration is still at the package level. If in the future providers need per-module capabilities (e.g., the `fs` module has a different config capability than `exec`), the current model would need an extension. This is not needed for the current proposal but is worth noting as a future direction.

### 6.7 Deduplicate the `decodeConfig` Pattern

Four different packages have nearly identical `decodeConfig` functions:

- `pkg/xgoja/providers/host/host.go:222`
- `geppetto/pkg/js/modules/geppetto/provider/provider.go`
- Various test helpers

A shared helper could be:

```go
func DecodeRawConfig(data json.RawMessage, dst any) error {
    if len(data) == 0 || string(data) == "null" {
        return nil
    }
    return json.Unmarshal(data, dst)
}
```

This could live in `providerapi` or `providerutil`.

---

## 7. Typing vs. Config HashMaps

### The Current State

The xgoja system uses **three** different config representations:

| Representation | Type | Used By | Tags |
|---|---|---|---|
| YAML static config | `map[string]any` (in Spec) → `json.RawMessage` (in ModuleContext) | `Module.New()` | `json:` |
| Glazed parsed values | `*values.Values` | `RuntimeInitializerCapability`, command `Run()` | `glazed:` |
| Config patch (proposed) | `map[string]any` | `ModuleConfigFromSections()` | None (manual map construction) |

Each step involves lossy type conversions: typed struct → `map[string]any` → `json.RawMessage` → typed struct.

### Why `map[string]any` is Problematic

1. **No compile-time validation.** A typo in a map key is a runtime error (or silent nil), not a compile error.

2. **Nested structures are fragile.** Deep merge of `map[string]any` must handle `[]any` vs `map[string]any` vs scalars, with no schema to guide it.

3. **The patch must map `glazed:` key names to `json:` key names.** In the Geppetto example:
   - CLI: `--geppetto-profile-registries` → `glazed:"profile-registries"`
   - Config JSON: `{"profileRegistries": [...]}` → `json:"profileRegistries"`
   - The `ModuleConfigFromSections` method must manually construct the `json:`-keyed map from the `glazed:`-decoded struct.

### A Better Approach: Struct-Based Patching

Instead of returning `map[string]any`, the capability could return `json.RawMessage` derived from a typed struct:

```go
type ModuleConfigCapability interface {
    PackageCapability
    ModuleConfigFromSections(
        ctx context.Context,
        vals *values.Values,
        descriptor ModuleDescriptor,
    ) (json.RawMessage, error)
}
```

Provider implementation:

```go
func (capability) ModuleConfigFromSections(ctx context.Context, vals *values.Values, desc providerapi.ModuleDescriptor) (json.RawMessage, error) {
    var cfg struct {
        ProfileRegistries []string `glazed:"profile-registries"`
        DefaultProfile    string   `glazed:"default-profile"`
        AllowRegistryLoad bool     `glazed:"allow-registry-load"`
        AllowNetwork      bool     `glazed:"allow-network"`
    }
    if vals == nil {
        return nil, nil
    }
    if err := vals.DecodeSectionInto("geppetto", &cfg); err != nil {
        return nil, err
    }
    // Map to the JSON-keyed struct that Module.New expects:
    patch := struct {
        ProfileRegistries []string `json:"profileRegistries,omitempty"`
        DefaultProfile    string   `json:"defaultProfile,omitempty"`
        AllowRegistryLoad bool     `json:"allowRegistryLoad,omitempty"`
        AllowNetwork      bool     `json:"allowNetwork,omitempty"`
    }{
        ProfileRegistries: cfg.ProfileRegistries,
        DefaultProfile:    cfg.DefaultProfile,
        AllowRegistryLoad: cfg.AllowRegistryLoad,
        AllowNetwork:      cfg.AllowNetwork,
    }
    return json.Marshal(patch)
}
```

**Advantages:**
- `json.RawMessage` merges cleanly with `ModuleInstance.Config` via a simple deep-merge on unmarshaled maps.
- No untyped `map[string]any` construction.
- The patch is validated by `json.Marshal` — missing fields are simply omitted (`omitempty`).

**Disadvantages:**
- Still requires two structs (glazed-tagged and json-tagged) or a single struct with both tag sets.
- Slightly more verbose than a map literal.

### An Even Better Approach: Dual-Tagged Struct with a Helper

If we accept the dual-tag pattern, a helper could automate the "decode from glazed, re-encode as json" dance:

```go
// PatchFromSection decodes a Glazed section into a struct (using glazed: tags),
// then re-marshals it as JSON (using json: tags) for config patching.
func PatchFromSection(vals *values.Values, section string, dst any) (json.RawMessage, error) {
    if err := vals.DecodeSectionInto(section, dst); err != nil {
        return nil, err
    }
    return json.Marshal(dst)
}
```

Used with a dual-tagged struct:

```go
type geppettoPatch struct {
    ProfileRegistries []string `glazed:"profile-registries" json:"profileRegistries,omitempty"`
    DefaultProfile    string   `glazed:"default-profile" json:"defaultProfile,omitempty"`
    AllowRegistryLoad bool     `glazed:"allow-registry-load" json:"allowRegistryLoad,omitempty"`
    AllowNetwork      bool     `glazed:"allow-network" json:"allowNetwork,omitempty"`
}
```

One decode + one encode = automatic key mapping with compile-time type safety.

### Recommendation

For the initial implementation, I recommend **accepting the proposal's `map[string]any` return type** to keep the change small and reviewable, but **adding a `PatchFromSection` helper** in `providerutil` that makes it easy to produce the map from a dual-tagged struct. In a follow-up, the return type can be changed to `json.RawMessage` if the team agrees.

---

## 8. Implementation Plan

### Phase 1: Core Capability Interface and Factory Extension

**Step 1.1: Add `ModuleConfigCapability` to `providerapi/capabilities.go`**

```go
// ModuleConfigCapability lets a provider decode parsed Glazed section values
// and produce a config patch that is merged into ModuleInstance.Config before
// Module.New() is called. This is the pre-runtime counterpart to
// RuntimeInitializerCapability.
type ModuleConfigCapability interface {
    PackageCapability

    ModuleConfigFromSections(
        ctx context.Context,
        vals *values.Values,
        descriptor ModuleDescriptor,
    ) (map[string]any, error)
}
```

**Step 1.2: Add `ModuleConfigPatchFromSections` to `providerutil/sections.go`**

```go
// ModuleConfigPatchFromSections collects config patches from all
// ModuleConfigCapability implementations attached to the selected module
// descriptors. Patches are deep-merged into a single map.
func ModuleConfigPatchFromSections(
    ctx context.Context,
    vals *values.Values,
    descriptors []providerapi.ModuleDescriptor,
) (map[string]map[string]any, error) {
    // Returns a map from module alias → patch, so each module gets its own patch
    patches := map[string]map[string]any{}
    applied := map[string]struct{}{}

    for _, descriptor := range descriptors {
        alias := descriptor.As
        patch := map[string]any{}

        for _, capability := range descriptor.PackageCapabilities {
            configCap, ok := capability.(providerapi.ModuleConfigCapability)
            if !ok {
                continue
            }
            id := capability.CapabilityID()
            key := packageCapabilityKey(descriptor.PackageID, id)
            if _, ok := applied[key]; ok {
                continue
            }
            applied[key] = struct{}{}

            partial, err := configCap.ModuleConfigFromSections(ctx, vals, descriptor)
            if err != nil {
                return nil, fmt.Errorf(
                    "module config from sections for %s.%s capability %s: %w",
                    descriptor.PackageID, descriptor.ModuleID, id, err,
                )
            }
            deepMerge(patch, partial)
        }

        if len(patch) > 0 {
            patches[alias] = patch
        }
    }
    return patches, nil
}
```

Note: This returns `map[string]map[string]any` (alias → patch) rather than a single flat map, because each module instance may receive a different patch.

**Step 1.3: Add `NewRuntimeFromSections` to `app.RuntimeFactory`**

```go
func (f *RuntimeFactory) NewRuntimeFromSections(
    ctx context.Context,
    profile string,
    vals *values.Values,
    opts ...require.Option,
) (*JSRuntime, error) {
    if f == nil || f.providers == nil || f.spec == nil {
        return nil, fmt.Errorf("xgoja runtime factory is not initialized")
    }
    runtime, ok := f.spec.Runtimes[profile]
    if !ok {
        return nil, fmt.Errorf("unknown runtime profile %q", profile)
    }

    // Resolve module descriptors for config patching
    descriptors, err := f.selectedModuleDescriptors(profile)
    if err != nil {
        return nil, err
    }

    // Collect config patches from ModuleConfigCapability implementations
    patches := map[string]map[string]any{}
    if vals != nil {
        patches, err = providerutil.ModuleConfigPatchFromSections(ctx, vals, descriptors)
        if err != nil {
            return nil, err
        }
    }

    modules := make([]engine.RuntimeModuleSpec, 0, len(runtime.Modules))
    for _, instance := range runtime.Modules {
        module, ok := f.providers.ResolveModule(instance.Package, instance.Name)
        if !ok {
            return nil, fmt.Errorf("runtime %s references unknown provider module %s.%s",
                profile, instance.Package, instance.Name)
        }

        // Clone config and merge patch
        config := cloneMap(instance.Config)
        if patch, ok := patches[instance.Alias()]; ok {
            deepMerge(config, patch)
        }

        patched := instance
        patched.Config = config
        modules = append(modules, providerRuntimeModuleSpec{
            instance: patched,
            module:   module,
            services: f.services,
        })
    }

    builder := engine.NewBuilder(
        engine.WithImplicitDefaultRegistryModules(false),
        engine.WithDataOnlyDefaultRegistryModules(false),
    ).WithModules(modules...)
    if len(opts) > 0 {
        builder = builder.WithRequireOptions(opts...)
    }
    runtimeFactory, err := builder.Build()
    if err != nil {
        return nil, err
    }
    return runtimeFactory.NewRuntime(
        engine.WithStartupContext(ctx),
        engine.WithLifetimeContext(ctx),
    )
}

func (f *RuntimeFactory) NewRuntime(ctx context.Context, profile string, opts ...require.Option) (*JSRuntime, error) {
    return f.NewRuntimeFromSections(ctx, profile, nil, opts...)
}

func cloneMap(m map[string]any) map[string]any {
    if m == nil {
        return map[string]any{}
    }
    out := make(map[string]any, len(m))
    for k, v := range m {
        out[k] = v
    }
    return out
}

func deepMerge(dst, src map[string]any) {
    for k, v := range src {
        if existing, ok := dst[k]; ok {
            if existingMap, ok := existing.(map[string]any); ok {
                if srcMap, ok := v.(map[string]any); ok {
                    deepMerge(existingMap, srcMap)
                    continue
                }
            }
        }
        dst[k] = v
    }
}
```

### Phase 2: Update Built-in Commands

**Step 2.1: Update `evalSourceWithInitializers`**

```go
func evalSourceWithInitializers(ctx context.Context, factory *RuntimeFactory, profile string, source string, vals *values.Values, selectedModules []providerapi.ModuleDescriptor, out io.Writer) error {
    // ...
    rt, err := factory.NewRuntimeFromSections(ctx, profile, vals)  // ← was NewRuntime
    // ...
    if vals != nil && len(selectedModules) > 0 {
        if err := initRuntimeFromSections(ctx, vals, rt, selectedModules); err != nil {
            return err
        }
    }
    // ...
}
```

**Step 2.2: Update `runScriptFileWithInitializers`** — same pattern.

**Step 2.3: Update `newXGojaTUIEvaluator`** — same pattern.

**Step 2.4: Update `buildVerbCommands` invoker** — same pattern.

**Step 2.5: Update `providerapi.RuntimeFactory` interface** — add `NewRuntimeFromSections`:

```go
type RuntimeFactory interface {
    NewRuntime(ctx context.Context, profile string, opts ...require.Option) (*engine.Runtime, error)
    NewRuntimeFromSections(ctx context.Context, profile string, vals *values.Values, opts ...require.Option) (*engine.Runtime, error)
}
```

**Step 2.6: Update command provider paths** — command providers that create runtimes should also use `NewRuntimeFromSections`.

### Phase 3: Geppetto Provider Implementation

```go
type moduleConfigCapability struct{}

func (moduleConfigCapability) CapabilityID() string { return "geppetto.module-config" }

func (moduleConfigCapability) ModuleConfigFromSections(
    ctx context.Context,
    vals *values.Values,
    descriptor providerapi.ModuleDescriptor,
) (map[string]any, error) {
    var cfg struct {
        ProfileRegistries []string `glazed:"profile-registries"`
        DefaultProfile    string   `glazed:"default-profile"`
        AllowRegistryLoad bool     `glazed:"allow-registry-load"`
        AllowNetwork      bool     `glazed:"allow-network"`
    }
    if vals == nil {
        return nil, nil
    }
    if err := vals.DecodeSectionInto("geppetto", &cfg); err != nil {
        return nil, err
    }
    patch := map[string]any{}
    if len(cfg.ProfileRegistries) > 0 {
        patch["profileRegistries"] = cfg.ProfileRegistries
        patch["allowRegistryLoad"] = cfg.AllowRegistryLoad
    }
    if cfg.DefaultProfile != "" {
        patch["defaultProfile"] = cfg.DefaultProfile
    }
    if cfg.AllowNetwork {
        patch["allowNetwork"] = true
    }
    return patch, nil
}
```

And register it alongside the existing `ConfigSectionCapability`:

```go
func Register(registry *providerapi.Registry) error {
    return registry.Package(PackageID,
        providerapi.Module{ /* ... */ },
        providerapi.WithPackageCapability(geppettoConfigSectionCapability{}),
        providerapi.WithPackageCapability(moduleConfigCapability{}),
    )
}
```

### Phase 4: Helper Utilities

Add to `providerutil/sections.go`:

```go
// PatchFromSection decodes a Glazed section into a struct (using glazed: tags),
// then re-marshals it as JSON (using json: tags) for config patching.
// The struct must have both glazed: and json: tags.
func PatchFromSection(vals *values.Values, section string, dst any) (map[string]any, error) {
    if vals == nil {
        return nil, nil
    }
    if err := vals.DecodeSectionInto(section, dst); err != nil {
        return nil, err
    }
    data, err := json.Marshal(dst)
    if err != nil {
        return nil, err
    }
    var patch map[string]any
    if err := json.Unmarshal(data, &patch); err != nil {
        return nil, err
    }
    // Remove zero-value entries to avoid overriding YAML config with defaults
    cleanPatch(patch, dst)
    return patch, nil
}
```

---

## 9. Testing Strategy

### 9.1 Unit Tests for `providerutil.ModuleConfigPatchFromSections`

- Test with a single descriptor + single `ModuleConfigCapability`.
- Test with multiple descriptors from the same package (deduplication).
- Test with nil vals → empty patches.
- Test with a capability that returns an error → wrapped error.

### 9.2 Unit Tests for `app.RuntimeFactory.NewRuntimeFromSections`

- **Factory merges module config patch before Module.New**:
  - Fixture provider exposes `ConfigSectionCapability` + `ModuleConfigCapability`.
  - Module factory records `ctx.Config`.
  - `NewRuntimeFromSections(ctx, profile, vals)` should see merged config.

- **Existing NewRuntime remains unchanged**:
  - No vals means only xgoja.yaml config is used.

- **Pre-runtime and post-runtime capabilities both run in order**:
  - Pre-runtime patch influences module factory config.
  - Runtime initializer sees same parsed values after runtime creation.

### 9.3 Integration Tests for Built-in Commands

- **Eval command passes parsed values into NewRuntimeFromSections**:
  - Command field shows up in module factory config before evaluation.

- **Run command passes parsed values into NewRuntimeFromSections**:
  - Script can verify that its module received the patched config.

- **TUI command passes parsed values into NewRuntimeFromSections**:
  - TUI evaluator receives merged config.

- **jsverbs invoker passes parsed values into NewRuntimeFromSections**:
  - Verb execution sees patched module config.

### 9.4 Config Precedence Tests

- **xgoja.yaml default overridden by parsed section values**:
  - YAML sets `allowNetwork: false`, CLI sets `--geppetto-allow-network` → patch wins.
  - YAML sets nested object, patch sets leaf → deep merge preserves non-conflicting keys.

- **Patch does not mutate spec**:
  - After creating two runtimes with different patches, the original spec is unchanged.

### 9.5 Existing Test Regression

- All existing `module_sections_test.go`, `eval_module_sections_test.go`, `run_module_sections_test.go`, `jsverbs_module_sections_test.go`, and `tui_module_sections_test.go` must continue to pass unchanged.

---

## 10. Risks and Open Questions

### 10.1 Deep Merge Semantics

The exact behavior of `deepMerge(dst, src)` for arrays is undefined. If both YAML config and a patch provide a `profileRegistries` array, should the patch **replace** the array or **append** to it? The issue says "shallow merge may be sufficient for the first implementation" but also notes "deep merge is better for nested config objects."

**Recommendation:** For v1, use **replace semantics** for arrays (patch wins entirely) and recursive merge for maps. Document this clearly.

### 10.2 Zero-Value Patching

If a Glazed flag has a default value (e.g., `fields.WithDefault(false)`), and the user doesn't set it, the decoded struct will have the zero value. If we merge this into the YAML config, it could **override** a YAML-set `true` with the zero-value `false`.

**Mitigation:** The `ModuleConfigFromSections` implementation should only include fields in the patch that were **explicitly set** by the user. This requires either:
- Checking if the field was present in the parsed values (not just decoded as zero).
- Using pointer types in the decode struct (e.g., `*bool`) so nil means "not set."

### 10.3 Multiple Capabilities on One Package

If a package has both a `ConfigSectionCapability` and a `ModuleConfigCapability`, they must agree on the section slug. If the section capability declares a section named "geppetto" and the config capability tries to decode section "geppetto", they're coupled by convention. This is fine for a single provider but could be confusing if capabilities are composed from different sources.

### 10.4 Command Provider Runtime Creation

Command providers currently receive a `RuntimeFactory` interface. Adding `NewRuntimeFromSections` to this interface is a breaking change for existing command providers. We need to decide:

- **Option A:** Add `NewRuntimeFromSections` to the interface (breaking change, but there are few implementors).
- **Option B:** Create a new `RuntimeFactoryWithSections` interface that extends `RuntimeFactory`, and type-assert in command providers that want to use it.

**Recommendation:** Option A — there are very few command provider implementations, and the benefit of a clean interface outweighs the migration cost.

### 10.5 `providerapi.RuntimeFactory` vs `app.RuntimeFactory`

The `providerapi.RuntimeFactory` interface (in `commands.go`) is what command providers see. The `app.RuntimeFactory` concrete type implements it. When we add `NewRuntimeFromSections` to the interface, we must update all implementations.

---

## 11. API Reference

### New Types and Interfaces

#### `ModuleConfigCapability`

```go
// In pkg/xgoja/providerapi/capabilities.go

type ModuleConfigCapability interface {
    PackageCapability

    // ModuleConfigFromSections decodes parsed Glazed section values and returns
    // a config patch that will be merged into the module's ModuleInstance.Config
    // before Module.New() is called.
    //
    // The returned map uses JSON key names (as expected by the provider's
    // decodeConfig function), not Glazed flag names.
    //
    // Return nil or an empty map if no patching is needed (e.g., vals is nil
    // or the relevant section has no values).
    ModuleConfigFromSections(
        ctx context.Context,
        vals *values.Values,
        descriptor ModuleDescriptor,
    ) (map[string]any, error)
}
```

#### `RuntimeFactory.NewRuntimeFromSections`

```go
// In pkg/xgoja/app/factory.go

func (f *RuntimeFactory) NewRuntimeFromSections(
    ctx context.Context,
    profile string,
    vals *values.Values,
    opts ...require.Option,
) (*JSRuntime, error)

// NewRuntime is unchanged in signature; implementation delegates to NewRuntimeFromSections.
func (f *RuntimeFactory) NewRuntime(
    ctx context.Context,
    profile string,
    opts ...require.Option,
) (*JSRuntime, error)
```

#### `providerapi.RuntimeFactory` Interface Extension

```go
// In pkg/xgoja/providerapi/commands.go

type RuntimeFactory interface {
    NewRuntime(ctx context.Context, profile string, opts ...require.Option) (*engine.Runtime, error)
    NewRuntimeFromSections(ctx context.Context, profile string, vals *values.Values, opts ...require.Option) (*engine.Runtime, error)
}
```

#### `providerutil.ModuleConfigPatchFromSections`

```go
// In pkg/xgoja/providerutil/sections.go

func ModuleConfigPatchFromSections(
    ctx context.Context,
    vals *values.Values,
    descriptors []providerapi.ModuleDescriptor,
) (map[string]map[string]any, error)
// Returns alias → patch map
```

### New Helper (Optional, Phase 4)

```go
// In pkg/xgoja/providerutil/sections.go

func PatchFromSection(vals *values.Values, section string, dst any) (map[string]any, error)
// Decodes a Glazed section into dst (glazed: tags), marshals to JSON (json: tags),
// and returns the result as a map[string]any config patch.
```

---

## 12. File Reference

### Core Capability System

| File | Purpose |
|---|---|
| `pkg/xgoja/providerapi/capabilities.go` | `PackageCapability`, `ConfigSectionCapability`, `RuntimeInitializerCapability`, `SectionContext`, `ModuleDescriptor`, `RuntimeHandle` — **add `ModuleConfigCapability` here** |
| `pkg/xgoja/providerapi/module.go` | `Module`, `ModuleFactory`, `ModuleContext`, `HostServices` — the factory receives `Config json.RawMessage` |
| `pkg/xgoja/providerapi/registry.go` | `Registry`, `Package`, `Entry`, `WithPackageCapability` — package registration and resolution |
| `pkg/xgoja/providerapi/commands.go` | `CommandSetProvider`, `RuntimeFactory` interface, `CommandSetContext` — **extend `RuntimeFactory` interface here** |
| `pkg/xgoja/providerapi/verbs.go` | `VerbSource` |
| `pkg/xgoja/providerapi/help.go` | `HelpSource` |

### App Layer (Runtime Construction and Commands)

| File | Purpose |
|---|---|
| `pkg/xgoja/app/factory.go` | `RuntimeFactory`, `providerRuntimeModuleSpec`, `NewRuntime()` — **add `NewRuntimeFromSections()`, `cloneMap()`, `deepMerge()` here** |
| `pkg/xgoja/app/module_sections.go` | `selectedModuleDescriptors()`, `sectionsForRuntimeProfile()`, `initRuntimeFromSections()`, `runtimeHandle` adapter |
| `pkg/xgoja/app/spec.go` | `Spec`, `Runtime`, `ModuleInstance` (has `Config map[string]any`) |
| `pkg/xgoja/app/root.go` | `evalCommand`, `newVerbsCommand`, `buildVerbCommands` — **update to use `NewRuntimeFromSections`** |
| `pkg/xgoja/app/run.go` | `runCommand`, `runScriptFileWithInitializers` — **update to use `NewRuntimeFromSections`** |
| `pkg/xgoja/app/tui.go` | `tuiCommand`, `newXGojaTUIEvaluator` — **update to use `NewRuntimeFromSections`** |
| `pkg/xgoja/app/host.go` | `Host`, `HostServices`, `AttachDefaultCommands` |
| `pkg/xgoja/app/command_providers.go` | `AttachCommandProviders`, `newCommandSet` — **update command providers to use `NewRuntimeFromSections`** |
| `pkg/xgoja/app/glazed.go` | `buildGlazedCobraCommand` |
| `pkg/xgoja/app/middlewares.go` | `MiddlewaresFromSpec`, `buildConfigPlan` |
| `pkg/xgoja/app/assets.go` | `AssetStore`, `HostServices` |

### Provider Utilities

| File | Purpose |
|---|---|
| `pkg/xgoja/providerutil/sections.go` | `CollectConfigSections()`, `InitRuntimeFromSections()`, `AppendUniqueSections()` — **add `ModuleConfigPatchFromSections()` here** |

### Engine Layer

| File | Purpose |
|---|---|
| `engine/factory.go` | `Factory`, `FactoryBuilder`, `NewRuntime()` — unchanged |
| `engine/module_specs.go` | `RuntimeModuleSpec`, `RuntimeInitializer`, `RuntimeContext` — unchanged |
| `engine/runtime_modules.go` | `RuntimeModuleContext` — unchanged |

### Existing Providers (Reference)

| File | Purpose |
|---|---|
| `pkg/xgoja/providers/core/core.go` | Simple provider — modules only, no capabilities |
| `pkg/xgoja/providers/host/host.go` | Guarded modules with `json.RawMessage` config schemas and `decodeConfig()` — **future `ModuleConfigCapability` candidate for CLI guard flags** |
| `pkg/xgoja/providers/http/http.go` | `ConfigSectionCapability` + `RuntimeInitializerCapability` — reference implementation for both capabilities |
| `pkg/xgoja/testprovider/provider.go` | `FixtureCapability` implementing both `ConfigSectionCapability` and `RuntimeInitializerCapability` — reference for tests |

### Geppetto (Motivating Use Case)

| File | Purpose |
|---|---|
| `geppetto/pkg/js/modules/geppetto/provider/provider.go` | `Config` struct with `json:` tags, `decodeConfig()`, `applyConfigRegistryOptions()`, `applyConfigStorageOptions()` — where `ModuleConfigCapability` will be added |

### Test Files

| File | Purpose |
|---|---|
| `pkg/xgoja/app/module_sections_test.go` | Existing tests for section collection and runtime initializer — **add `ModuleConfigCapability` tests here** |
| `pkg/xgoja/app/eval_module_sections_test.go` | Eval command section tests |
| `pkg/xgoja/app/run_module_sections_test.go` | Run command section tests |
| `pkg/xgoja/app/jsverbs_module_sections_test.go` | jsverbs section tests |
| `pkg/xgoja/app/tui_module_sections_test.go` | TUI section tests |
| `pkg/xgoja/providerutil/sections_test.go` | Provider utility tests |

---

## Appendix A: Flow Diagrams

### Current Flow

```
┌─────────────────────────────────────────────────────────┐
│                  Command Construction                     │
│                                                          │
│  factory.sectionsForRuntimeProfile()                     │
│    → ConfigSectionCapability.ConfigSections()             │
│    → []schema.Section attached to command description    │
└──────────────────────┬──────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────┐
│                  Command Execution                       │
│                                                          │
│  Glazed parses flags → *values.Values                    │
│       │                                                  │
│       ▼                                                  │
│  factory.NewRuntime(ctx, profile)                        │
│    → ModuleInstance.Config from xgoja.yaml ONLY          │
│    → providerRuntimeModuleSpec.RegisterRuntimeModule()   │
│      → json.Marshal(instance.Config)                     │
│      → Module.New(ModuleContext{Config: marshaled})      │
│       │                                                  │
│       ▼                                                  │
│  initRuntimeFromSections(ctx, vals, rt, modules)         │
│    → RuntimeInitializerCapability.InitRuntimeFromSections │
│       │                                                  │
│       ▼                                                  │
│  Execute eval / run / repl / jsverb                      │
└─────────────────────────────────────────────────────────┘
```

### Proposed Flow

```
┌─────────────────────────────────────────────────────────┐
│                  Command Construction                     │
│                                                          │
│  factory.sectionsForRuntimeProfile()                     │
│    → ConfigSectionCapability.ConfigSections()             │
│    → []schema.Section attached to command description    │
└──────────────────────┬──────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────┐
│                  Command Execution                       │
│                                                          │
│  Glazed parses flags → *values.Values                    │
│       │                                                  │
│       ▼                                                  │
│  factory.NewRuntimeFromSections(ctx, profile, vals)      │
│    → providerutil.ModuleConfigPatchFromSections()        │
│      → ModuleConfigCapability.ModuleConfigFromSections() │
│      → map[string]map[string]any (alias → patch)        │
│    → cloneMap(instance.Config)                           │
│    → deepMerge(config, patches[alias])                   │
│    → providerRuntimeModuleSpec with patched config       │
│      → Module.New(ModuleContext{Config: MERGED})         │
│       │                                                  │
│       ▼                                                  │
│  initRuntimeFromSections(ctx, vals, rt, modules)         │
│    → RuntimeInitializerCapability.InitRuntimeFromSections │
│       │                                                  │
│       ▼                                                  │
│  Execute eval / run / repl / jsverb                      │
└─────────────────────────────────────────────────────────┘
```

### Capability Timeline

```
                    ConfigSectionCapability
                    ┌──────────────────┐
                    │ ConfigSections()  │ → Declares what flags exist
                    └──────────────────┘
                              │
                              ▼  (Glazed parses flags into values.Values)
                              │
              ModuleConfigCapability (NEW)
              ┌────────────────────────────┐
              │ ModuleConfigFromSections()  │ → Converts parsed values to config patch
              └────────────────────────────┘
                              │
                              ▼  (App merges patch into ModuleInstance.Config)
                              │
                    Module.New(ModuleContext{Config: merged})
                              │
                              ▼  (Runtime now exists)
                              │
              RuntimeInitializerCapability
              ┌────────────────────────────┐
              │ InitRuntimeFromSections()   │ → Post-creation runtime setup
              └────────────────────────────┘
```

---

## Appendix B: Config Precedence Model

From lowest to highest precedence:

```
1. Module.New() code defaults         (hardcoded in Go)
2. xgoja.yaml module config           (static, from Spec)
3. Config file values                 (Glazed config middleware)
4. Environment variable values        (Glazed env middleware)
5. CLI flag values                    (Glazed cobra middleware)
```

The `ModuleConfigCapability` patch (which encodes config file / env / CLI values) overrides the xgoja.yaml static config. This is the desired behavior: runtime-supplied values should win over build-time defaults.

---

## Appendix C: Glossary

| Term | Definition |
|---|---|
| **xgoja** | The code generation and runtime framework that produces standalone JavaScript runtime binaries |
| **Spec** | The YAML/JSON configuration that defines an xgoja binary's packages, runtimes, commands, and assets |
| **Provider** | A Go package that registers modules, capabilities, and other entries into the provider registry |
| **Package** | A named group of entries in the provider registry (e.g., `go-go-goja-host`, `geppetto`) |
| **Module** | A single `require()`-loadable JavaScript module backed by a Go `ModuleFactory` |
| **Capability** | An optional behavior extension on a provider package (sections, init, config) |
| **Runtime Profile** | A named selection of modules in the spec that defines a specific runtime configuration |
| **ModuleInstance** | A spec entry that selects a module from a package with optional alias and config |
| **Glazed** | The CLI framework used by xgoja for command definitions, flag parsing, and output formatting |
| **Section** | A Glazed concept: a named group of flags/fields that can be attached to a command |
| **Values** | The parsed result of Glazed flag/argument/config/env parsing, organized by section |
