---
Title: "Research Logbook"
Ticket: GOJA-053
Status: active
Topics:
    - xgoja
    - provider
    - capability
    - config
    - glazed
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources:
    - "GitHub Issue: https://github.com/go-go-golems/go-go-goja/issues/52"
Summary: "Structured log of every resource consulted during the ModuleConfigCapability investigation, with usefulness assessments and update recommendations"
LastUpdated: 2026-06-03T08:30:00-04:00
WhatFor: "Track which resources were useful, what's out of date, and what needs updating for future researchers"
WhenToUse: "Resume work on GOJA-053, audit resource freshness, or onboard a new researcher to this topic"
---

# Research Logbook

This document records every resource consulted during the GOJA-053 investigation (2026-06-03). Each entry follows a consistent structure so future researchers can assess what's still current and what needs refreshing.

## Legend

| Rating | Meaning |
|---|---|
| 🟢 Current | Resource is up to date, accurately reflects the codebase |
| 🟡 Partially current | Mostly correct but has gaps or minor inaccuracies |
| 🔴 Out of date | Does not reflect current state, needs updating |
| ⚪ Supplemental | Background context, not expected to track code changes |

---

## 1. GitHub Issue #52 — ModuleConfigCapability Proposal

- **URL:** https://github.com/go-go-golems/go-go-goja/issues/52
- **What I was researching:** The problem statement, proposed API, and acceptance criteria for pre-runtime config patching
- **What I was looking for:** The exact capability interface sketch, the proposed `NewRuntimeFromSections` signature, the Geppetto motivating example, the merge semantics, and the test plan
- **Why I chose it:** This is the primary source of truth for the feature request — written by the project author with concrete API sketches and pseudocode
- **How I found it:** Directly referenced in the user's prompt
- **What I found useful:**
  - The "Current xgoja flow" diagram — clearly shows the gap between parsed values and `Module.New()`
  - The `ModuleConfigCapability` interface sketch — directly shaped the design doc's API reference
  - The Geppetto motivating example with concrete CLI usage and config file shape
  - The five acceptance criteria — will form the basis of the test plan
  - The `moduleConfigPatchFromSections` pseudocode — informed the `providerutil` implementation sketch
  - The merge semantics discussion (shallow vs deep merge)
- **What I didn't find useful:**
  - The `CommandSetProvider` integration sketch is incomplete — it mentions updating command providers but doesn't show the full `RuntimeFactory` interface change
  - No discussion of zero-value patching (Glazed defaults overriding YAML `true` with zero-value `false`)
- **What is out of date / wrong:**
  - The pseudocode for `NewRuntimeFromSections` iterates by index (`i < len(descriptors)`) and patches by position — the actual implementation should key patches by module alias, not by index, because `selectedModuleDescriptors` and `runtime.Modules` may not have a 1:1 correspondence
  - The proposal doesn't mention that `providerapi.RuntimeFactory` is an interface used by command providers — extending it is a breaking change
- **What would need updating:**
  - Add a section on the `providerapi.RuntimeFactory` interface change and its impact on command providers
  - Add a section on zero-value patching and pointer-type recommendation
  - Correct the index-based patching pseudocode to alias-based patching
  - Update the "Commands to update" section to also mention the TUI command (it says `repl` but the actual command is `tui`)
- **Rating:** 🟡 Partially current — the core proposal is sound but the pseudocode has implementation issues and several important concerns are missing

---

## 2. `pkg/xgoja/providerapi/capabilities.go` — Capability Interfaces

- **File:** `pkg/xgoja/providerapi/capabilities.go`
- **What I was researching:** The existing capability interface hierarchy, `PackageCapability` marker, `ConfigSectionCapability`, `RuntimeInitializerCapability`, `SectionContext`, `ModuleDescriptor`, `RuntimeHandle`
- **What I was looking for:** The exact interface signatures, how capabilities are attached to packages, the `SectionContext` fields, and the `RuntimeHandle` abstraction
- **Why I chose it:** This is where `ModuleConfigCapability` will be added — must understand the existing patterns to design the new interface consistently
- **How I found it:** Followed from `providerapi` package imports in `providerutil/sections.go`
- **What I found useful:**
  - The `PackageCapability` marker interface pattern — `ModuleConfigCapability` must implement `CapabilityID()`
  - `SectionContext` — shows what context is available during section collection (needed to understand what context `ModuleConfigFromSections` should receive)
  - `ModuleDescriptor` — the bridge type that carries capabilities to the app layer; `ModuleConfigCapability` will appear in `PackageCapabilities`
  - `RuntimeHandle` — the minimal abstraction over `engine.Runtime`; shows the intentional decoupling between `providerapi` and the engine layer
  - `RuntimeCloserRegistry` — relevant for understanding how HTTP provider attaches cleanup
- **What I didn't find useful:**
  - The `capabilityEntry` struct is an internal detail of the `WithPackageCapability` functional option — not directly relevant to the design
- **What is out of date / wrong:**
  - No documentation explaining that capabilities are **package-scoped** — the fact that `PackageCapabilities` are shared across all modules from a package is a critical design decision that's not commented
  - The `SectionContext` struct lacks a `CommandProviderID` doc comment explaining when it's set vs `CommandName`
- **What would need updating:**
  - Add a package-level doc comment explaining the package-scoping of capabilities and the deduplication behavior
  - Add doc comments to `SectionContext` fields explaining which commands/providers set which fields
  - When `ModuleConfigCapability` is added, include a doc comment explaining the pre-runtime vs post-runtime split
- **Rating:** 🟢 Current — accurately reflects the codebase but lacks documentation for key design decisions

---

## 3. `pkg/xgoja/providerapi/module.go` — Module and ModuleContext

- **File:** `pkg/xgoja/providerapi/module.go`
- **What I was researching:** How `Module.New()` receives its config, the `ModuleFactory` signature, `ModuleContext` fields, and `HostServices`
- **What I was looking for:** The `Config json.RawMessage` field in `ModuleContext` — this is the config that `ModuleConfigCapability` patches must be merged into
- **Why I chose it:** The config flow into `Module.New()` is the central mechanism that the new capability must intercept
- **How I found it:** Direct reference from `app/factory.go` where `providerRuntimeModuleSpec` constructs `ModuleContext`
- **What I found useful:**
  - `ModuleContext.Config` is `json.RawMessage` — confirms that config arrives as raw JSON bytes, not a typed struct
  - `ModuleContext` also carries `HostServices` and `RuntimeOwner` — the `ModuleConfigCapability` patch doesn't need to touch these
  - `Module.ConfigSchema` is `json.RawMessage` — providers can declare their config schema but there's no runtime validation of config against schema
- **What I didn't find useful:**
  - `AssetResolver` and `HostServices` interfaces are relevant for host providers but not for the config patching mechanism
- **What is out of date / wrong:**
  - No doc comment on `ModuleContext` explaining the lifecycle (constructed by `providerRuntimeModuleSpec`, consumed by `Module.New`)
  - The `Config` field has no doc comment explaining that it's marshaled from `ModuleInstance.Config` (a `map[string]any`)
- **What would need updating:**
  - Add doc comments to `ModuleContext` and its fields
  - Consider adding a `ConfigSource` field to `ModuleContext` that records where the config came from (YAML only vs YAML + CLI patch) for debugging
- **Rating:** 🟢 Current — accurately reflects the codebase

---

## 4. `pkg/xgoja/providerapi/registry.go` — Provider Registry and Package

- **File:** `pkg/xgoja/providerapi/registry.go`
- **What I was researching:** How providers register entries (modules, capabilities, etc.), the `Entry` pattern, and how capabilities are stored and resolved
- **What I was looking for:** The `Package.PackageCapabilities` map, the `WithPackageCapability` entry, the `ResolvePackageCapabilities` method, and how capabilities are keyed (by ID string)
- **Why I chose it:** Need to understand the registration mechanism to know where `ModuleConfigCapability` instances will be stored and how they'll be resolved at runtime
- **How I found it:** Referenced from `providerapi/capabilities.go` and from provider `Register()` functions
- **What I found useful:**
  - Capabilities are stored in `map[string]PackageCapability` keyed by `CapabilityID()` — confirms that capability IDs must be unique within a package
  - `ResolvePackageCapabilities` returns `[]PackageCapability` sorted by ID — this is the order capabilities are iterated
  - The `Entry` functional options pattern — `WithPackageCapability` follows the same pattern as module registration
  - `Package.clone()` copies capabilities by reference (not deep copy) — means the same capability instance is shared across clones
- **What I didn't find useful:**
  - The `resolve` methods for `VerbSource`, `HelpSource`, `CommandSetProvider` are not directly relevant to the config patching mechanism
- **What is out of date / wrong:**
  - No doc comment on `Package` explaining the package-scoping of capabilities
  - The `sortedCapabilities()` method is undocumented — the sort order matters for the order in which capabilities are invoked
- **What would need updating:**
  - Add a doc comment on `Package` explaining that all modules from a package share the same capabilities
  - Document the iteration order guarantee for `sortedCapabilities()`
- **Rating:** 🟢 Current — accurately reflects the codebase

---

## 5. `pkg/xgoja/providerapi/commands.go` — CommandSetProvider and RuntimeFactory

- **File:** `pkg/xgoja/providerapi/commands.go`
- **What I was researching:** The `RuntimeFactory` interface that command providers use, the `CommandSetContext` fields, and the `CommandSetProvider` factory signature
- **What I was looking for:** The `RuntimeFactory` interface — this is where `NewRuntimeFromSections` must be added, which is a breaking change for command providers
- **Why I chose it:** Adding `NewRuntimeFromSections` to this interface affects the contract for all command providers
- **How I found it:** Referenced from `app/command_providers.go` and `testprovider/provider.go`
- **What I found useful:**
  - `RuntimeFactory` is a minimal interface with a single method — easy to extend
  - `CommandSetContext` carries `RuntimeFactory`, `SelectedModules`, and `Config` — command providers can create their own runtimes
  - `CommandSetProvider.New()` receives a `CommandSetContext` and returns a `CommandSet` with commands and an optional `ParserConfig`
- **What I didn't find useful:**
  - The `normalizeCommandSetProvider` function is internal validation
- **What is out of date / wrong:**
  - The `RuntimeFactory` interface only has `NewRuntime` — when `NewRuntimeFromSections` is added, all implementors must be updated
  - No doc comment explaining the relationship between `RuntimeFactory` (providerapi interface) and `app.RuntimeFactory` (concrete implementation)
- **What would need updating:**
  - Add `NewRuntimeFromSections` to the interface when the feature is implemented
  - Add a doc comment explaining the `providerapi.RuntimeFactory` vs `app.RuntimeFactory` relationship
  - Update all `RuntimeFactory` implementors (currently only `app.RuntimeFactory`)
- **Rating:** 🟢 Current — but will need updating when `ModuleConfigCapability` is implemented

---

## 6. `engine/factory.go` — Engine Factory and Builder

- **File:** `engine/factory.go`
- **What I was researching:** The low-level engine runtime creation pipeline, `FactoryBuilder` API, `Factory.NewRuntime()` flow, and how `RuntimeModuleSpec` registrations work
- **What I was looking for:** The exact point where `RegisterRuntimeModule` is called (this is where `Module.New()` receives its config), and whether the engine layer needs any changes for the new capability
- **Why I chose it:** Need to understand if the `ModuleConfigCapability` changes are confined to the `app` layer or if they must penetrate the `engine` layer
- **How I found it:** Referenced from `app/factory.go` where `engine.NewBuilder()` is called
- **What I found useful:**
  - **The engine layer does NOT need changes** — the config patching happens in `app/factory.go` before modules are fed into `engine.NewBuilder().WithModules()`. The engine sees already-patched `providerRuntimeModuleSpec` instances
  - `Factory.NewRuntime()` calls `mod.RegisterRuntimeModule(moduleCtx, reg)` for each module spec — this is where `Module.New()` is ultimately invoked
  - Runtime initializers in the engine are separate from provider capabilities — they're `engine.RuntimeInitializer` (engine-scoped), not `providerapi.RuntimeInitializerCapability` (provider-scoped)
  - The builder pattern is clean and immutable after `Build()`
- **What I didn't find useful:**
  - The module middleware pipeline, default registry modules, and process module are not relevant to config patching
- **What is out of date / wrong:**
  - Nothing — this file is stable and correctly implements the builder pattern
- **What would need updating:**
  - No changes needed for `ModuleConfigCapability`
- **Rating:** 🟢 Current

---

## 7. `pkg/xgoja/app/factory.go` — App-Layer RuntimeFactory

- **File:** `pkg/xgoja/app/factory.go`
- **What I was researching:** The concrete `RuntimeFactory` that creates runtimes from the spec, the `providerRuntimeModuleSpec` that wraps provider modules, and the `NewRuntime` method
- **What I was looking for:** The exact code path where `ModuleInstance.Config` is marshaled and passed to `Module.New()` — the **single insertion point** for config patching
- **Why I chose it:** This is the most critical file for the implementation — `NewRuntimeFromSections` will be added here, and `providerRuntimeModuleSpec` must be modified to carry patched config
- **How I found it:** Direct reference from command files (`root.go`, `run.go`, `tui.go`)
- **What I found useful:**
  - **`providerRuntimeModuleSpec.RegisterRuntimeModule()`** at lines 38-49 — this is where `json.Marshal(s.instance.Config)` creates the `json.RawMessage` that becomes `ModuleContext.Config`. This is the choke point.
  - `NewRuntime()` iterates `runtime.Modules`, resolves each from the registry, creates `providerRuntimeModuleSpec`, feeds into engine builder
  - The `services` field (`HostServices`) is passed through to `ModuleContext.Host`
  - The `ID()` method uses `xgoja:` prefix + package.name:alias format
- **What I didn't find useful:**
  - The `NewRuntimeFactory` constructor is straightforward
- **What is out of date / wrong:**
  - No doc comment on `providerRuntimeModuleSpec` explaining its role as the bridge between spec and engine
  - No doc comment on `NewRuntime` explaining the config flow (spec-only, no parsed CLI values)
- **What would need updating:**
  - Add `NewRuntimeFromSections` method
  - Modify the module iteration loop to collect config patches and clone/merge configs
  - Add `cloneMap` and `deepMerge` helper functions
  - Make `NewRuntime` delegate to `NewRuntimeFromSections(ctx, profile, nil, opts...)`
  - Add doc comments explaining the config flow
- **Rating:** 🟢 Current — but will be the primary file modified during implementation

---

## 8. `pkg/xgoja/app/module_sections.go` — Section Collection and Runtime Init

- **File:** `pkg/xgoja/app/module_sections.go`
- **What I was researching:** How sections are collected from capabilities, how `initRuntimeFromSections` works, the `runtimeHandle` adapter, and the `selectedModuleDescriptors` method
- **What I was looking for:** The existing `initRuntimeFromSections` implementation (post-runtime), the `runtimeHandle` adapter, and where `ModuleConfigPatchFromSections` should be called relative to these existing functions
- **Why I chose it:** This is where `NewRuntimeFromSections` will call both the config patching (new, pre-runtime) and the runtime initialization (existing, post-runtime)
- **How I found it:** Direct reference from `root.go`, `run.go`, `tui.go`
- **What I found useful:**
  - `selectedModuleDescriptors()` — the exact method that resolves capabilities for a profile; the config patching must use the same descriptors
  - `initRuntimeFromSections()` — delegates to `providerutil.InitRuntimeFromSections()`, confirming that the pre-runtime patching should happen before this call
  - `runtimeHandle` adapter — implements `providerapi.RuntimeHandle` and `RuntimeCloserRegistry`; the new code won't need this adapter since it runs before the runtime exists
  - `addSectionsToCommandDescription()` — the existing section-attachment logic
- **What I didn't find useful:**
  - `appendSectionsToCommandDescription` internal helper
- **What is out of date / wrong:**
  - `runtimeHandle` is defined here instead of its own file — makes it harder to find
  - No doc comment on `initRuntimeFromSections` explaining the lifecycle position (post-runtime only)
- **What would need updating:**
  - Move `runtimeHandle` to a dedicated file (e.g., `handle.go`)
  - Add doc comments distinguishing pre-runtime vs post-runtime capability phases
  - No functional changes needed — the new `ModuleConfigPatchFromSections` call will happen in `factory.go`, not here
- **Rating:** 🟢 Current

---

## 9. `pkg/xgoja/app/spec.go` — Spec, Runtime, ModuleInstance

- **File:** `pkg/xgoja/app/spec.go`
- **What I was researching:** The spec data model — `ModuleInstance.Config`, `Runtime.Modules`, `CommandProviderInstance`, and how the spec JSON is structured
- **What I was looking for:** The `ModuleInstance.Config map[string]any` field — this is the config that gets patched, and its type (`map[string]any`) has implications for merge semantics
- **Why I chose it:** The spec defines the shape of the static config that `ModuleConfigCapability` patches will override
- **How I found it:** Referenced from `factory.go` and from the user's prompt mentioning the spec
- **What I found useful:**
  - `ModuleInstance.Config` is `map[string]any` — means patching is map-to-map merging, not typed struct merging
  - `ModuleInstance.Alias()` — confirms that the alias (not the module name) is the `require()` registration key, and should be the key for the patch map
  - `CommandProviderInstance` also has `Config map[string]any` — command providers could also benefit from config patching in the future
  - `CommandsSpec` — shows which commands are enabled and which runtime profile they use
- **What I didn't find useful:**
  - `ConfigSpec`, `TargetSpec`, `AssetSourceSpec` — not related to the config patching mechanism
- **What is out of date / wrong:**
  - No doc comment on `ModuleInstance.Config` explaining that it's marshaled to `json.RawMessage` before reaching `Module.New()`
  - No validation of `Config` against `Module.ConfigSchema` — the schema is purely informational
- **What would need updating:**
  - Add a doc comment on `Config` explaining the marshaling flow and the precedence model (YAML < CLI patch)
  - Consider adding config validation against `ConfigSchema` in a future iteration
- **Rating:** 🟢 Current

---

## 10. `pkg/xgoja/app/root.go` — Eval Command and Verb Commands

- **File:** `pkg/xgoja/app/root.go`
- **What I was researching:** The `evalCommand` implementation, the `buildVerbCommands` function, and how they create runtimes and call initializers
- **What I was looking for:** The exact code path where `factory.NewRuntime()` is called and where `initRuntimeFromSections()` is called after — these are the two calls that must be updated to use `NewRuntimeFromSections`
- **Why I chose it:** Two of the four built-in command paths (eval and jsverbs) are defined here
- **How I found it:** Direct file in the `app` package
- **What I found useful:**
  - `evalSourceWithInitializers()` — clear pattern: `factory.NewRuntime()` then `initRuntimeFromSections()`. The update is mechanical: change `NewRuntime()` to `NewRuntimeFromSections(ctx, profile, vals)`
  - `buildVerbCommands()` invoker callback — same pattern inside the verb invoker. Important: the verb invoker creates a fresh runtime per invocation, so each invocation gets its own patched config
  - `firstRuntime()` — utility for determining the default runtime profile
  - `commandName()`, `commandMount()` — naming utilities
- **What I didn't find useful:**
  - The `modulesCommand` and `scanVerbSource` functions are not affected by the change
  - The `newVerbsCommand` is a wrapper that delegates to `buildVerbCommands`
- **What is out of date / wrong:**
  - Nothing — this file accurately implements the current flow
- **What would need updating:**
  - Change `factory.NewRuntime(ctx, profile)` → `factory.NewRuntimeFromSections(ctx, profile, vals)` in `evalSourceWithInitializers()`
  - Change `factory.NewRuntime(ctx, profile, require.WithLoader(...))` → `factory.NewRuntimeFromSections(ctx, profile, parsedValues, require.WithLoader(...))` in `buildVerbCommands()` invoker
  - Update the `providerapi.RuntimeFactory` usage in the invoker
- **Rating:** 🟢 Current — will need updating during implementation

---

## 11. `pkg/xgoja/app/run.go` — Run Command

- **File:** `pkg/xgoja/app/run.go`
- **What I was researching:** The `runCommand` implementation, specifically `runScriptFileWithInitializers()`
- **What I was looking for:** Same pattern as eval — where `NewRuntime()` and `initRuntimeFromSections()` are called
- **Why I chose it:** The run command is the third built-in command path that needs updating
- **How I found it:** Direct file in the `app` package
- **What I found useful:**
  - `runScriptFileWithInitializers()` — follows the exact same pattern as `evalSourceWithInitializers()`: `factory.NewRuntime()` then `initRuntimeFromSections()`
  - The `requireOpt` from `engine.RequireOptionWithModuleRootsFromScript()` — this is passed to `NewRuntime` and must also be passed to `NewRuntimeFromSections`
  - The `keepAlive` flag and `waitForKeepAlive()` — post-runtime behavior, not affected by config patching
- **What I didn't find useful:**
  - `waitForKeepAlive()` and `commandRuntime()` — not related to config patching
- **What is out of date / wrong:**
  - Nothing
- **What would need updating:**
  - Change `factory.NewRuntime(ctx, profile, requireOpt)` → `factory.NewRuntimeFromSections(ctx, profile, vals, requireOpt)` in `runScriptFileWithInitializers()`
- **Rating:** 🟢 Current

---

## 12. `pkg/xgoja/app/tui.go` — TUI REPL Command

- **File:** `pkg/xgoja/app/tui.go`
- **What I was researching:** The `tuiCommand` and `newXGojaTUIEvaluator()` — how the TUI REPL creates its runtime
- **What I was looking for:** Where `factory.NewRuntime()` is called in the TUI path
- **Why I chose it:** The TUI is the fourth built-in command path; it's unique because the runtime persists for the entire session
- **How I found it:** Direct file in the `app` package
- **What I found useful:**
  - `newXGojaTUIEvaluator()` — calls `factory.NewRuntime(ctx, profile)` without `requireOpt` (no module roots from script path)
  - The runtime is created once and wrapped in a `jsadapter.JavaScriptEvaluator` — the patched config will affect the entire session
  - The `initRuntimeFromSections` call is in the same function
- **What I didn't find useful:**
  - The Bubbletea TUI setup, event bus, program options — not related to config patching
  - `newQuietInMemoryBus()` — utility for the TUI framework
- **What is out of date / wrong:**
  - Nothing
- **What would need updating:**
  - Change `factory.NewRuntime(ctx, profile)` → `factory.NewRuntimeFromSections(ctx, profile, vals)` in `newXGojaTUIEvaluator()`
- **Rating:** 🟢 Current

---

## 13. `pkg/xgoja/app/host.go` — Host and Command Attachment

- **File:** `pkg/xgoja/app/host.go`
- **What I was researching:** The `Host` type that wires the provider registry, spec, factory, and commands together
- **What I was looking for:** How commands are attached and whether the `Host` needs changes for `NewRuntimeFromSections`
- **Why I chose it:** The `Host` is the top-level orchestrator — need to verify it doesn't need changes
- **How I found it:** Referenced from `NewRootCommand` in `root.go`
- **What I found useful:**
  - `NewHostWithOptions()` creates the `RuntimeFactory` and `HostServices` — no changes needed here
  - `AttachDefaultCommands()` dispatches to individual attach methods — no changes needed
  - The `Factory` field is a `*RuntimeFactory` — this is what commands use to create runtimes
- **What I didn't find useful:**
  - The individual `Attach*` methods are thin wrappers that create commands and add them to the Cobra root
- **What is out of date / wrong:**
  - Nothing — the `Host` doesn't need changes; the command implementations themselves will be updated
- **What would need updating:**
  - No changes needed in this file
- **Rating:** 🟢 Current

---

## 14. `pkg/xgoja/app/command_providers.go` — Command Provider Attachment

- **File:** `pkg/xgoja/app/command_providers.go`
- **What I was researching:** How command providers are attached and how they create command sets, specifically the `newCommandSet` method and the `selectedModulesForCommandProvider` helper
- **What I was looking for:** Whether command providers create runtimes (they do, via `ctx.RuntimeFactory`), and whether their runtime creation path needs `NewRuntimeFromSections`
- **Why I chose it:** Command providers are the most complex runtime creation path — they may need to pass parsed values to their runtimes
- **How I found it:** Referenced from `host.go` → `AttachCommandProviders()`
- **What I found useful:**
  - `newCommandSet()` creates a `CommandSetContext` with `RuntimeFactory` and `SelectedModules` — command providers can call `ctx.RuntimeFactory.NewRuntime()` to create runtimes
  - The `selectedModulesForCommandProvider` can filter modules by the instance's `Modules` list
  - `applyMountToCommands` — mounting logic, not related to config patching
- **What I didn't find useful:**
  - The `mountedCommand*` types are thin wrappers for command mounting
- **What is out of date / wrong:**
  - Command providers currently have no way to pass parsed `values.Values` to `RuntimeFactory.NewRuntime()` — they create runtimes without section values
  - When `NewRuntimeFromSections` is added to `providerapi.RuntimeFactory`, command providers can start using it
- **What would need updating:**
  - Command providers that create runtimes should be updated to use `NewRuntimeFromSections` when they have parsed values available
  - The `CommandSetContext` may need a `Values` field in the future for command providers that want to pass parsed values to runtime creation
- **Rating:** 🟢 Current — but command provider runtime creation is a gap that `ModuleConfigCapability` partially addresses

---

## 15. `pkg/xgoja/app/middlewares.go` — Glazed Middleware Chain

- **File:** `pkg/xgoja/app/middlewares.go`
- **What I was researching:** How Glazed middlewares are configured for xgoja commands, specifically the precedence order (defaults < config < env < args < cobra flags)
- **What I was looking for:** The precedence model — this determines how `ModuleConfigCapability` patches override YAML config values
- **Why I chose it:** The config precedence directly affects the merge semantics of `ModuleConfigCapability`
- **How I found it:** Referenced from `host.go` → `MiddlewaresFunc`
- **What I found useful:**
  - The middleware chain is ordered from highest to lowest precedence: Cobra flags → args → env → config → defaults
  - `EffectiveEnvPrefix()` derives the env namespace from `appName` or `envPrefix` in the spec
  - `buildConfigPlan()` configures config file layering (system, XDG, home, git-root, cwd, explicit)
  - The `DefaultEnvPrefix()` function handles hyphenated app names correctly
- **What I didn't find useful:**
  - The `DefaultEnvPrefix` implementation details are not directly relevant to config patching
- **What is out of date / wrong:**
  - Nothing
- **What would need updating:**
  - No changes needed — the existing middleware chain is exactly what `ModuleConfigCapability` needs. The `values.Values` produced by this chain will be passed to `ModuleConfigFromSections()`
- **Rating:** 🟢 Current

---

## 16. `pkg/xgoja/app/assets.go` — AssetStore and HostServices

- **File:** `pkg/xgoja/app/assets.go`
- **What I was researching:** The `HostServices` implementation and `AssetResolver` — what services are available to modules
- **What I was looking for:** Whether `HostServices` needs to be extended for `ModuleConfigCapability`
- **Why I chose it:** Need to understand what's available in `ModuleContext.Host` for modules that need host services during construction
- **How I found it:** Referenced from `factory.go` where `HostServices` is passed to `providerRuntimeModuleSpec`
- **What I found useful:**
  - `HostServices` is a simple struct with only an `Assets *AssetStore` field
  - `AssetStore` resolves embedded asset filesystems by ID — used by `host` provider for embedded FS mounts
  - Geppetto extends `HostServices` with its own interface (`geppetto.HostServices`) — this is the pattern for provider-specific host services
- **What I didn't find useful:**
  - The asset resolution logic is not directly relevant to config patching
- **What is out of date / wrong:**
  - Nothing
- **What would need updating:**
  - No changes needed
- **Rating:** 🟢 Current

---

## 17. `pkg/xgoja/app/glazed.go` — Glazed Command Builder

- **File:** `pkg/xgoja/app/glazed.go`
- **What I was researching:** How Glazed Cobra commands are built from `cmds.Command` descriptions
- **What I was looking for:** Whether the Glazed command builder needs changes for `ModuleConfigCapability`
- **Why I chose it:** The command builder determines how sections become flags and how parsed values flow back to `Run()`
- **How I found it:** Referenced from all command constructors
- **What I found useful:**
  - `buildGlazedCobraCommand()` wraps `cli.BuildCobraCommand()` with a standard parser config
  - The `CobraParserConfig` sets `ShortHelpSections` and `MiddlewaresFunc`
  - This is where the Glazed plumbing turns `schema.Section` objects into Cobra flags
- **What I didn't find useful:**
  - The `commandErrorStub` utility
- **What is out of date / wrong:**
  - Nothing
- **What would need updating:**
  - No changes needed — the existing Glazed integration correctly produces `values.Values` that `ModuleConfigCapability` can consume
- **Rating:** 🟢 Current

---

## 18. `pkg/xgoja/app/framework.go` — Root Framework Installation

- **File:** `pkg/xgoja/app/framework.go`
- **What I was researching:** How the root command framework (logging, help system) is installed
- **What I was looking for:** Whether the framework installation affects the config patching flow
- **Why I chose it:** Need to verify no interactions
- **How I found it:** Referenced from `host.go`
- **What I found useful:**
  - `installRootFramework()` adds logging and help to the root command — unrelated to config patching
  - `loadConfiguredHelpSources()` loads help from provider packages — unrelated
- **What I didn't find useful:**
  - Most of this file is unrelated to the feature
- **What is out of date / wrong:**
  - Nothing
- **What would need updating:**
  - None
- **Rating:** 🟢 Current

---

## 19. `pkg/xgoja/providerutil/sections.go` — Provider Utility Functions

- **File:** `pkg/xgoja/providerutil/sections.go`
- **What I was researching:** The existing `CollectConfigSections()` and `InitRuntimeFromSections()` functions — the pattern that `ModuleConfigPatchFromSections()` will follow
- **What I was looking for:** The deduplication mechanism (`packageCapabilityKey`), the iteration pattern, the error wrapping convention, and the function signatures
- **Why I chose it:** `ModuleConfigPatchFromSections()` must follow the same patterns and conventions
- **How I found it:** Referenced from `app/module_sections.go`
- **What I found useful:**
  - `CollectConfigSections()` — the exact iteration pattern: for each descriptor, for each capability, type-assert, dedupe by `(packageID, capabilityID)`, call, wrap errors
  - `InitRuntimeFromSections()` — same pattern but for runtime initializers
  - `packageCapabilityKey()` — private deduplication key function; the new function will reuse it
  - `AppendUniqueSections()` — duplicate slug detection; not needed for config patching
- **What I didn't find useful:**
  - `AppendUniqueSections()` — only relevant for section collection, not for config patching
- **What is out of date / wrong:**
  - Nothing
- **What would need updating:**
  - Add `ModuleConfigPatchFromSections()` following the same iteration and error-wrapping pattern
  - Optionally add `PatchFromSection()` helper for dual-tagged struct convenience
- **Rating:** 🟢 Current

---

## 20. `pkg/xgoja/providerutil/sections_test.go` — Provider Utility Tests

- **File:** `pkg/xgoja/providerutil/sections_test.go`
- **What I was researching:** The existing test patterns for `CollectConfigSections` and `InitRuntimeFromSections`
- **What I was looking for:** The test fixture types (`sectionCapability`, `runtimeInitCapability`, `fakeRuntimeHandle`), the deduplication test pattern, and the error-wrapping test pattern
- **Why I chose it:** New tests for `ModuleConfigPatchFromSections` should follow the same patterns
- **How I found it:** Adjacent to `sections.go`
- **What I found useful:**
  - The `fakeRuntimeHandle` — a minimal `RuntimeHandle` implementation for testing; won't be needed for `ModuleConfigPatchFromSections` tests (which are pre-runtime)
  - Deduplication test (`TestCollectConfigSectionsDedupesSamePackageCapability`) — the same deduplication should be tested for config patching
  - Error-wrapping test (`TestInitRuntimeFromSectionsWrapsErrors`) — same pattern for config patching errors
  - The fixture capability types follow a consistent pattern: struct with fields, `CapabilityID()`, and the interface method
- **What I didn't find useful:**
  - The nil section and empty slug tests are specific to section collection
- **What is out of date / wrong:**
  - Nothing
- **What would need updating:**
  - Add tests for `ModuleConfigPatchFromSections` using the same fixture patterns
- **Rating:** 🟢 Current

---

## 21. `pkg/xgoja/providers/core/core.go` — Core Provider

- **File:** `pkg/xgoja/providers/core/core.go`
- **What I was researching:** The simplest provider implementation — modules only, no capabilities
- **What I was looking for:** A minimal provider registration example to contrast with providers that have capabilities
- **Why I chose it:** Understanding the simplest case helps explain the capability system to an intern
- **How I found it:** Listed in the provider packages directory
- **What I found useful:**
  - Shows the minimal `Register()` pattern: `registry.Package(PackageID, entries...)` with only `Module` entries
  - Uses `modules.GetModule()` and `nativeModuleEntry()` — the bridge between the `modules` package's `NativeModule` and `providerapi.Module`
  - No capabilities, no config, no sections — the simplest possible provider
- **What I didn't find useful:**
  - Not much — this file is very small and focused
- **What is out of date / wrong:**
  - Nothing
- **What would need updating:**
  - None — this provider doesn't need `ModuleConfigCapability` since it has no config
- **Rating:** 🟢 Current

---

## 22. `pkg/xgoja/providers/host/host.go` — Host Provider (Guarded Modules)

- **File:** `pkg/xgoja/providers/host/host.go`
- **What I was researching:** The host provider — the most complex existing provider with guarded modules, `json.RawMessage` config schemas, and the `decodeConfig` pattern
- **What I was looking for:** How providers decode config from `ModuleContext.Config`, the `GuardConfig` pattern, and whether the host provider could benefit from `ModuleConfigCapability` (e.g., `--host-fs-allow` CLI flag)
- **Why I chose it:** The host provider demonstrates the config decoding pattern that `ModuleConfigCapability` aims to make CLI-configurable
- **How I found it:** Listed in the provider packages directory
- **What I found useful:**
  - The `decodeConfig()` function — the exact pattern used by every provider: `json.Unmarshal(data, &cfg)`, with nil/empty guards
  - `FSConfig`, `ExecConfig`, `DatabaseConfig` — typed config structs with `json:` tags; these would need `glazed:` tag counterparts for CLI exposure
  - `ConfigSchema` as `json.RawMessage` — providers embed JSON Schema for documentation but there's no runtime validation
  - The guard pattern (`allow: true` required in config) — this is exactly the kind of flag that could be exposed as a CLI flag via `ModuleConfigCapability`
  - `embeddedBackendFromConfig()` — shows how config drives host service resolution
- **What I didn't find useful:**
  - The `processModuleLoader` is an internal implementation detail
- **What is out of date / wrong:**
  - The `decodeConfig` function is duplicated across providers — should be a shared helper
  - No doc comments explaining the guard config pattern
- **What would need updating:**
  - In the future, the host provider could add `ConfigSectionCapability` + `ModuleConfigCapability` to expose guard flags as CLI flags (e.g., `--host-fs-allow`, `--host-exec-allow`)
  - Extract `decodeConfig` into a shared `providerutil` helper
- **Rating:** 🟢 Current

---

## 23. `pkg/xgoja/providers/http/http.go` — HTTP Provider (Reference Implementation)

- **File:** `pkg/xgoja/providers/http/http.go`
- **What I was researching:** The HTTP provider — the only existing provider that implements both `ConfigSectionCapability` and `RuntimeInitializerCapability`
- **What I was looking for:** The full capability implementation pattern: how sections are declared, how values are decoded, how runtime initialization works, and how the capability maintains per-runtime state
- **Why I chose it:** This is the **reference implementation** for the two existing capabilities — any new capability should follow this pattern
- **How I found it:** Listed in the provider packages directory; known to be the most complete capability implementation
- **What I found useful:**
  - The `capability` struct with `entries map[*goja.Runtime]*runtimeEntry` — per-runtime state management
  - `ConfigSections()` — declares the `http` section with `enabled` and `listen` fields, using `schema.NewSection` with `schema.WithPrefix("http-")`
  - `InitRuntimeFromSections()` — decodes the `http` section into a `settings` struct using `vals.DecodeSectionInto("http", &cfg)`, then applies settings to the per-runtime entry
  - The pattern of `section → decode → apply` is exactly what `ModuleConfigCapability` will do, except the "apply" step produces a config patch instead of directly modifying runtime state
  - The `settings` struct uses `glazed:` tags for Glazed decoding
  - `NewExpressLoader()` — shows how the module loader accesses capability state
- **What I didn't find useful:**
  - The server startup/shutdown logic is specific to HTTP and not relevant to the capability pattern
- **What is out of date / wrong:**
  - The `settings` struct doesn't have `json:` tags — if we wanted to add `ModuleConfigCapability`, we'd need dual-tagged structs or a separate JSON-keyed struct
  - The `InitRuntimeFromSections` doesn't receive `SectionContext` — it can't distinguish which command triggered the initialization
- **What would need updating:**
  - The HTTP provider could add `ModuleConfigCapability` to allow CLI flags to override the HTTP listen address and enabled state before module construction
  - The `settings` struct would need dual tags (`glazed:` + `json:`)
- **Rating:** 🟢 Current — the best reference for implementing `ModuleConfigCapability`

---

## 24. `pkg/xgoja/testprovider/provider.go` — Test Fixture Provider

- **File:** `pkg/xgoja/testprovider/provider.go`
- **What I was researching:** The test fixture provider — implements both `ConfigSectionCapability` and `RuntimeInitializerCapability` for testing
- **What I was looking for:** How test fixtures implement capabilities, how `FixtureCapability` works end-to-end, and the `sectionsFromSelectedModules` helper used by command providers
- **Why I chose it:** This is the test fixture that will be extended to implement `ModuleConfigCapability` for testing
- **How I found it:** Referenced from test files
- **What I found useful:**
  - `FixtureCapability` — clean example of implementing both `ConfigSectionCapability` and `RuntimeInitializerCapability`
  - `FixtureSection()` — reusable section factory function
  - `InitRuntimeFromSections()` — decodes values and sets a global (`fixtureValue`) on the goja runtime
  - `sectionsFromSelectedModules()` — demonstrates how command providers collect sections from selected modules' capabilities
  - `NewFixtureCommandSet()` — shows the full command provider pattern
  - `decodeFixtureCommand()` — demonstrates decoding both default-section and named-section values
- **What I didn't find useful:**
  - The `fixtureGlazeCommand`, `fixtureWriterCommand` implementations are specific to test scenarios
- **What is out of date / wrong:**
  - Nothing — this is a test fixture that correctly implements the current capability interfaces
- **What would need updating:**
  - Add a `ModuleConfigCapability` implementation to the fixture for testing the pre-runtime config patching path
  - The test fixture's `Module.New` factory should record the received `ModuleContext.Config` so tests can verify the patch was applied
- **Rating:** 🟢 Current

---

## 25. `geppetto/pkg/js/modules/geppetto/provider/provider.go` — Geppetto Provider (Motivating Case)

- **File:** `geppetto/pkg/js/modules/geppetto/provider/provider.go`
- **What I was researching:** The Geppetto provider — the concrete motivating use case for `ModuleConfigCapability`
- **What I was looking for:** How Geppetto decodes config, applies config to `geppettomodule.Options`, and where CLI flags would need to be injected
- **Why I chose it:** This is the real-world provider that needs the feature — understanding its config flow is essential for validating the design
- **How I found it:** Referenced in the GitHub issue #52
- **What I found useful:**
  - `Config` struct with `json:` tags — `ProfileRegistries`, `DefaultProfile`, `AllowRegistryLoad`, `AllowNetwork`, `AllowTools`, `EnableStorage`, `Turns`
  - `decodeConfig()` — handles the `profileRegistries` polymorphism (string vs array) using a `configAlias` pattern
  - `applyConfigRegistryOptions()` — where profile registries are loaded and chained; this runs inside `Module.New()` and is where CLI flags would need to flow
  - `applyConfigStorageOptions()` — where storage is configured; also runs inside `Module.New()`
  - `HostServices` and `StorageHostServices` — Geppetto's own host service interfaces, separate from xgoja's
  - The `configSchema` JSON Schema — comprehensive but purely informational
- **What I didn't find useful:**
  - The `decodeSourceEntries` helper for parsing profile registry source entries is an implementation detail
- **What is out of date / wrong:**
  - Geppetto has no `ConfigSectionCapability` or `ModuleConfigCapability` — all configuration is YAML-only
  - The `Config` struct only has `json:` tags — no `glazed:` tags for CLI exposure
- **What would need updating:**
  - Add `ConfigSectionCapability` to declare the `geppetto` section with CLI flags
  - Add `ModuleConfigCapability` to convert parsed CLI values into a config patch
  - Either add `glazed:` tags to the existing `Config` struct or create a separate Glazed-section struct
- **Rating:** 🟢 Current — accurately reflects the current state (no CLI flags)

---

## 26. `pkg/xgoja/app/module_sections_test.go` — Section Test Fixtures

- **File:** `pkg/xgoja/app/module_sections_test.go`
- **What I was researching:** The existing test fixture patterns for section collection and runtime initialization tests
- **What I was looking for:** The `sectionCapability`, `runtimeInitCapability`, `noopSectionModule`, and `newSectionTestFactory` fixtures that will be extended for `ModuleConfigCapability` tests
- **Why I chose it:** New tests must follow the same fixture patterns and factory construction
- **How I found it:** Direct test file in the `app` package
- **What I found useful:**
  - `newSectionTestFactory()` — creates a minimal `RuntimeFactory` with a single `fixture` package containing a `mod` module; this is the standard test setup
  - `noopSectionModule` — a module factory that does nothing; used as the module entry in test packages
  - `sectionCapability` — a minimal `ConfigSectionCapability` with configurable ID and slug
  - `runtimeInitCapability` — a minimal `RuntimeInitializerCapability` with a configurable callback function
  - `sectionSlugs()` — utility for extracting section slugs
  - The deduplication test pattern and the error-wrapping test pattern
- **What I didn't find useful:**
  - Nothing — all fixtures are directly relevant
- **What is out of date / wrong:**
  - Nothing
- **What would need updating:**
  - Add a `moduleConfigCapability` fixture type with a configurable callback
  - Add tests for `NewRuntimeFromSections` that verify config patching
- **Rating:** 🟢 Current

---

## 27. `pkg/xgoja/app/eval_module_sections_test.go` — Eval Command Section Tests

- **File:** `pkg/xgoja/app/eval_module_sections_test.go`
- **What I was researching:** The eval command section tests — how the eval command integrates with capabilities
- **What I was looking for:** The end-to-end test pattern: construct command → set args → execute → verify behavior
- **Why I chose it:** The eval test pattern will be replicated for `NewRuntimeFromSections` tests
- **How I found it:** Direct test file in the `app` package
- **What I found useful:**
  - `TestEvalCommandInitializesRuntimeFromModuleSections` — the exact end-to-end pattern: `buildGlazedCobraCommand` → `cmd.SetArgs` → `cmd.ExecuteContext` → verify output
  - `TestEvalCommandRuntimeOverrideInitializesSelectedRuntimeProfile` — tests runtime profile override
  - The test verifies that `fixtureValue` was set on the global scope — confirming that `InitRuntimeFromSections` ran
- **What I didn't find useful:**
  - The section-presence test (`TestEvalCommandIncludesRuntimeProfileModuleSections`) is for section collection, not for config patching
- **What is out of date / wrong:**
  - Nothing
- **What would need updating:**
  - Add a test that verifies `Module.New()` receives patched config when `NewRuntimeFromSections` is used
  - The test module factory would need to record `ModuleContext.Config` for later assertion
- **Rating:** 🟢 Current

---

## 28. `pkg/xgoja/app/run_module_sections_test.go` — Run Command Section Tests

- **File:** `pkg/xgoja/app/run_module_sections_test.go`
- **What I was researching:** The run command section tests and the `runFixtureCapability` pattern
- **What I was looking for:** The `runFixtureCapability` and `prefixedRunFixtureCapability` types, the `newRuntimeOverrideFactory` helper, and the `setFixtureValue` utility
- **Why I chose it:** The run command tests show how to test multi-runtime-profile scenarios and prefixed fixture values
- **How I found it:** Direct test file in the `app` package
- **What I found useful:**
  - `newRuntimeOverrideFactory()` — creates a factory with two packages (`defaultpkg`, `overridepkg`) with different prefixed capabilities; useful pattern for testing profile-dependent config patching
  - `prefixedRunFixtureCapability` — shows how to parameterize test capabilities
  - `setFixtureValue()` — utility for setting a global on the runtime from parsed values
- **What I didn't find useful:**
  - The section-presence test
- **What is out of date / wrong:**
  - Nothing
- **What would need updating:**
  - Add tests for `NewRuntimeFromSections` with the run command
- **Rating:** 🟢 Current

---

## 29. `pkg/xgoja/app/jsverbs_module_sections_test.go` — JSVerbs Section Tests

- **File:** `pkg/xgoja/app/jsverbs_module_sections_test.go`
- **What I was researching:** The jsverbs section tests and the embedded verb filesystem test pattern
- **What I was looking for:** How to test that jsverb commands receive sections and that initializers run inside verb invocations
- **Why I chose it:** The jsverbs path is the most complex runtime creation path (one runtime per verb invocation)
- **How I found it:** Direct test file in the `app` package
- **What I found useful:**
  - `jsverbsSectionFS()` — creates an in-memory `fstest.MapFS` with a test verb JavaScript file; good pattern for embedded filesystem testing
  - `TestJSVerbsInitializeRuntimeFromModuleSections` — full end-to-end test that creates a root command with embedded jsverbs and verifies the initializer ran
  - The test verb checks `globalThis.fixtureValue` — confirming that `InitRuntimeFromSections` ran before the verb
- **What I didn't find useful:**
  - The section-presence test
- **What is out of date / wrong:**
  - Nothing
- **What would need updating:**
  - Add tests for `NewRuntimeFromSections` with jsverbs
- **Rating:** 🟢 Current

---

## 30. `pkg/xgoja/app/tui_module_sections_test.go` — TUI Section Tests

- **File:** `pkg/xgoja/app/tui_module_sections_test.go`
- **What I was researching:** The TUI section tests
- **What I was looking for:** How to test the TUI evaluator initialization with capabilities
- **Why I chose it:** The TUI is the fourth built-in command path
- **How I found it:** Direct test file in the `app` package
- **What I found useful:**
  - `TestNewXGojaTUIEvaluatorInitializesRuntimeFromModuleSections` — tests the evaluator constructor directly (without running the full TUI), which is the right level of testing for config patching
  - Uses `runtimeInitCapability` with a `called` flag — simpler than the eval/run tests which verify global scope values
- **What I didn't find useful:**
  - The section-presence test
- **What is out of date / wrong:**
  - Nothing
- **What would need updating:**
  - Add tests for `NewRuntimeFromSections` with the TUI evaluator
- **Rating:** 🟢 Current

---

## 31. `glazed/pkg/cmds/values/section-values.go` — Values and DecodeSectionInto

- **File:** `glazed/pkg/cmds/values/section-values.go`
- **What I was researching:** The `values.Values` type and the `DecodeSectionInto` method that providers use to extract typed config from parsed flags
- **What I was looking for:** The `Values` struct definition, `DecodeSectionInto` signature and behavior, and `SectionValues` structure
- **Why I chose it:** `ModuleConfigCapability.ModuleConfigFromSections` receives `*values.Values` — must understand how to extract data from it
- **How I found it:** Referenced from `providerutil/sections.go` and from provider implementations
- **What I found useful:**
  - `Values` is an `orderedmap.OrderedMap[string, *SectionValues]` — sections are keyed by slug
  - `DecodeSectionInto(sectionKey, &dst)` — decodes a named section's field values into a struct using `glazed:` tags; special-cases `DefaultSlug`
  - `SectionValues` contains `Section` (metadata) and `Fields` (parsed field values)
  - `Values.ForEach()` — iterates sections in order
- **What I didn't find useful:**
  - The `SectionValuesOption` and `WithFieldValue` are for constructing values, not consuming them
- **What is out of date / wrong:**
  - Nothing
- **What would need updating:**
  - No changes needed — this API is what `ModuleConfigCapability` implementations will use
- **Rating:** 🟢 Current

---

## 32. `glazed/pkg/cmds/schema/section-impl.go` — Section Implementation

- **File:** `glazed/pkg/cmds/schema/section-impl.go`
- **What I was researching:** How `schema.Section` is implemented and how sections are constructed with `NewSection`
- **What I was looking for:** The `SectionImpl` struct, `NewSection()` constructor, and `WithPrefix` / `WithFields` options
- **Why I chose it:** `ConfigSectionCapability.ConfigSections()` returns `[]schema.Section` — providers use `NewSection()` to declare sections
- **How I found it:** Referenced from `http.go` and `testprovider/provider.go`
- **What I found useful:**
  - `SectionImpl` has `Name`, `Slug`, `Description`, `Prefix`, `Definitions` — the prefix is used for CLI flag namespacing (e.g., `http-` → `--http-enabled`)
  - `NewSection(slug, name, ...options)` — the standard constructor
  - `WithPrefix(prefix)` — sets the flag prefix
  - `WithFields(fields.New(...))` — adds field definitions
- **What I didn't find useful:**
  - The YAML unmarshaling is not relevant to the capability design
- **What is out of date / wrong:**
  - Nothing
- **What would need updating:**
  - No changes needed
- **Rating:** 🟢 Current

---

## 33. `glazed/pkg/cmds/values/serialize_parsed.go` — Values Serialization

- **File:** `glazed/pkg/cmds/values/serialize_parsed.go`
- **What I was researching:** How `Values` are serialized and the `SerializableValues` structure
- **What I was looking for:** The ordered map structure and how sections are keyed
- **Why I chose it:** Understanding the internal structure helps reason about how `DecodeSectionInto` works
- **How I found it:** Adjacent to `section-values.go`
- **What I found useful:**
  - `SerializableValues.Sections` is an `orderedmap.OrderedMap` — confirms the ordering guarantee
  - `ToSerializableValues()` — shows the conversion from internal to serializable representation
- **What I didn't find useful:**
  - Serialization is not needed for `ModuleConfigCapability` — we consume values, not serialize them
- **What is out of date / wrong:**
  - Nothing
- **What would need updating:**
  - None
- **Rating:** 🟢 Current

---

## 34. `engine/module_specs.go` — Runtime Module Specs and Initializers

- **File:** `engine/module_specs.go`
- **What I was researching:** The `RuntimeModuleSpec` interface, `RuntimeInitializer` interface, and `RuntimeContext` — the engine-layer counterparts to the provider-layer capabilities
- **What I was looking for:** Whether the engine layer has its own module/config system that interacts with the provider layer
- **Why I chose it:** Need to understand the boundary between engine and provider APIs
- **How I found it:** Referenced from `engine/factory.go`
- **What I found useful:**
  - `RuntimeModuleSpec` — the engine's module registration interface; `providerRuntimeModuleSpec` implements this
  - `RuntimeInitializer` — engine-level initializers (separate from provider capabilities); these run inside `Factory.NewRuntime()`
  - `NativeModuleSpec` — simpler module spec for modules that don't need runtime context
  - The `RuntimeModuleContext` has `SetValue/Value` methods for inter-module communication via the value bag
  - `defaultRegistryModuleAliases` — maps short names to Node.js-style aliases (`fs` → `node:fs`)
- **What I didn't find useful:**
  - The `processModuleLoader` and `ProcessEnv` are not related to config patching
- **What is out of date / wrong:**
  - Nothing
- **What would need updating:**
  - No changes needed for the engine layer
- **Rating:** 🟢 Current

---

## 35. `engine/runtime_modules.go` — RuntimeModuleContext

- **File:** `engine/runtime_modules.go`
- **What I was researching:** The `RuntimeModuleContext` struct passed to module specs during registration
- **What I was looking for:** What context is available at module registration time (this is where `Module.New()` is called)
- **Why I chose it:** Understanding what `Module.New()` has access to helps design the config patching mechanism
- **How I found it:** Referenced from `engine/factory.go`
- **What I found useful:**
  - `RuntimeModuleContext` provides `Context`, `VM`, `Loop`, `Owner`, `AddCloser`, `Values` — these are the runtime-scoped objects
  - The `Values` map is a simple `map[string]any` — separate from `values.Values` (Glazed) and from `ModuleInstance.Config`
  - `SetValue/Value` methods — inter-module communication mechanism
- **What I didn't find useful:**
  - Nothing beyond the context understanding
- **What is out of date / wrong:**
  - Nothing
- **What would need updating:**
  - None
- **Rating:** 🟢 Current

---

## 36. `README.md` — Project README

- **File:** `README.md`
- **What I was researching:** The project overview, folder layout, and quick start
- **What I was looking for:** A high-level description of the project architecture, the canonical API flow, and the folder layout
- **Why I chose it:** General orientation before diving into code
- **How I found it:** Project root
- **What I found useful:**
  - The canonical API flow: `engine.NewBuilder() → Build() → Factory.NewRuntime(...)` — confirms the engine-level composition model
  - The folder layout — shows where modules, engine, and examples live
  - The `--log-level debug` tip for seeing module registration
- **What I didn't find useful:**
  - The quick start examples are for the standalone REPL, not for the xgoja generated binary flow
  - Doesn't describe the provider/capability system at all
- **What is out of date / wrong:**
  - **Does not mention the xgoja provider system, capability model, or generated binary flow** — the README describes the low-level engine API but not the high-level provider/app layer that sits on top
  - **Does not mention `pkg/xgoja/`** — the entire generated binary infrastructure is undocumented in the README
  - The folder layout doesn't include `pkg/xgoja/` or the provider packages
- **What would need updating:**
  - Add a section describing the xgoja generated binary architecture
  - Add the `pkg/xgoja/` directory to the folder layout
  - Describe the provider registry, capability model, and spec-driven runtime creation
  - Add a link to the xgoja documentation (if one exists)
- **Rating:** 🔴 Out of date — missing critical sections about the xgoja provider/app layer

---

## 37. `AGENT.md` — Agent Guidelines

- **File:** `AGENT.md`
- **What I was researching:** Agent coding guidelines and project conventions
- **What I was looking for:** Build commands, test commands, code style guidelines
- **Why I chose it:** General orientation for the project conventions
- **How I found it:** Project root
- **What I found useful:**
  - Build: `go build ./...`, Test: `go test ./...`, Single test: `go test ./pkg/path/to/package -run TestName`
  - Code style: interfaces with `var _ Interface = &Foo{}`, cobra for CLI, `context` arguments, `pkg/errors` for wrapping
- **What I didn't find useful:**
  - Web guidelines and debugging guidelines are not relevant to this investigation
- **What is out of date / wrong:**
  - Nothing directly wrong, but it doesn't mention the xgoja provider conventions (register, capabilities, etc.)
- **What would need updating:**
  - Consider adding a section on xgoja provider authoring conventions
- **Rating:** ⚪ Supplemental

---

## 38. `engine/options.go` — Engine Options

- **File:** `engine/options.go`
- **What I was researching:** Engine builder options
- **What I was looking for:** The `WithImplicitDefaultRegistryModules` and `WithDataOnlyDefaultRegistryModules` options used by the app factory
- **Why I chose it:** The app factory sets both to `false` — need to understand what this means
- **How I found it:** Referenced from `app/factory.go`
- **What I found useful:**
  - Confirmed that the app layer disables implicit default modules (because providers explicitly register what they need)
- **What I didn't find useful:**
  - The other options are not directly relevant
- **What is out of date / wrong:**
  - Nothing
- **What would need updating:**
  - None
- **Rating:** 🟢 Current

---

## Summary Table

| # | Resource | Rating | Needs Update? |
|---|---|---|---|
| 1 | GitHub Issue #52 | 🟡 | Add missing concerns (zero-value patching, interface change, index-based pseudocode) |
| 2 | `providerapi/capabilities.go` | 🟢 | Add package-scoping docs, field doc comments |
| 3 | `providerapi/module.go` | 🟢 | Add lifecycle docs, `Config` field comment |
| 4 | `providerapi/registry.go` | 🟢 | Add package-scoping docs, sort order docs |
| 5 | `providerapi/commands.go` | 🟢 | Will need `NewRuntimeFromSections` addition; add factory relationship docs |
| 6 | `engine/factory.go` | 🟢 | No changes needed |
| 7 | `app/factory.go` | 🟢 | Primary implementation target — add `NewRuntimeFromSections`, `cloneMap`, `deepMerge` |
| 8 | `app/module_sections.go` | 🟢 | Move `runtimeHandle` to own file; add lifecycle docs |
| 9 | `app/spec.go` | 🟢 | Add `Config` field doc about marshaling and precedence |
| 10 | `app/root.go` | 🟢 | Update to use `NewRuntimeFromSections` |
| 11 | `app/run.go` | 🟢 | Update to use `NewRuntimeFromSections` |
| 12 | `app/tui.go` | 🟢 | Update to use `NewRuntimeFromSections` |
| 13 | `app/host.go` | 🟢 | No changes needed |
| 14 | `app/command_providers.go` | 🟢 | Command providers should eventually use `NewRuntimeFromSections` |
| 15 | `app/middlewares.go` | 🟢 | No changes needed |
| 16 | `app/assets.go` | 🟢 | No changes needed |
| 17 | `app/glazed.go` | 🟢 | No changes needed |
| 18 | `app/framework.go` | 🟢 | No changes needed |
| 19 | `providerutil/sections.go` | 🟢 | Add `ModuleConfigPatchFromSections`, optionally `PatchFromSection` |
| 20 | `providerutil/sections_test.go` | 🟢 | Add `ModuleConfigPatchFromSections` tests |
| 21 | `providers/core/core.go` | 🟢 | No changes needed |
| 22 | `providers/host/host.go` | 🟢 | Extract `decodeConfig`; future `ModuleConfigCapability` candidate |
| 23 | `providers/http/http.go` | 🟢 | Best reference for capability implementation |
| 24 | `testprovider/provider.go` | 🟢 | Add `ModuleConfigCapability` fixture |
| 25 | `geppetto/provider/provider.go` | 🟢 | Motivating case — add `ConfigSectionCapability` + `ModuleConfigCapability` |
| 26 | `app/module_sections_test.go` | 🟢 | Add `ModuleConfigCapability` fixtures and tests |
| 27 | `app/eval_module_sections_test.go` | 🟢 | Add `NewRuntimeFromSections` tests |
| 28 | `app/run_module_sections_test.go` | 🟢 | Add `NewRuntimeFromSections` tests |
| 29 | `app/jsverbs_module_sections_test.go` | 🟢 | Add `NewRuntimeFromSections` tests |
| 30 | `app/tui_module_sections_test.go` | 🟢 | Add `NewRuntimeFromSections` tests |
| 31 | `glazed/.../section-values.go` | 🟢 | No changes needed |
| 32 | `glazed/.../section-impl.go` | 🟢 | No changes needed |
| 33 | `glazed/.../serialize_parsed.go` | 🟢 | No changes needed |
| 34 | `engine/module_specs.go` | 🟢 | No changes needed |
| 35 | `engine/runtime_modules.go` | 🟢 | No changes needed |
| 36 | `README.md` | 🔴 | Missing xgoja provider/app layer documentation entirely |
| 37 | `AGENT.md` | ⚪ | Consider adding xgoja provider conventions |
| 38 | `engine/options.go` | 🟢 | No changes needed |
