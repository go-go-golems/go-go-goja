---
Title: Generic Symbol Inventory and Glossary
Ticket: GOJA-053
Status: active
Topics:
    - xgoja
    - architecture
    - lifecycle
    - provider
    - capability
    - documentation
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/js/modules/geppetto/module.go
      Note: provider-specific Options and moduleRuntime symbols used as naming examples.
    - Path: geppetto/pkg/js/modules/geppetto/provider/provider.go
      Note: |-
        provider-specific HostServices and StorageHostServices symbols used as naming examples.
        Provider-specific HostServices naming example
    - Path: go-go-goja/cmd/xgoja/internal/buildspec/spec.go
      Note: Build-time xgoja.yaml Spec/Runtime/ModuleInstance symbols.
    - Path: go-go-goja/engine/factory.go
      Note: |-
        engine FactoryBuilder/Factory/runtimebridgeOwner construction symbols.
        engine FactoryBuilder/Factory/runtimebridgeOwner symbol definitions
    - Path: go-go-goja/engine/module_middleware.go
      Note: ModuleSelector and ModuleMiddleware symbols.
    - Path: go-go-goja/engine/module_specs.go
      Note: |-
        RuntimeInitializer, RuntimeContext, NativeModuleSpec, and default module specs.
        RuntimeInitializer/RuntimeContext/NativeModuleSpec symbol definitions
    - Path: go-go-goja/engine/runtime.go
      Note: engine Runtime lifecycle symbol.
    - Path: go-go-goja/engine/runtime_modules.go
      Note: |-
        RuntimeModuleSpec and RuntimeModuleContext symbols.
        RuntimeModuleSpec and RuntimeModuleContext symbol definitions
    - Path: go-go-goja/modules/common.go
      Note: |-
        default native module Registry and NativeModule symbols.
        Default native module Registry/NativeModule definitions
    - Path: go-go-goja/pkg/runtimebridge/runtimebridge.go
      Note: |-
        runtimebridge RuntimeServices and RuntimeOwner bridge symbols.
        RuntimeServices and narrow RuntimeOwner bridge definitions
    - Path: go-go-goja/pkg/runtimeowner/types.go
      Note: |-
        runtimeowner RuntimeOwner, Scheduler, CallFunc/PostFunc, and Options symbols.
        Full RuntimeOwner/Scheduler/Options definitions
    - Path: go-go-goja/pkg/xgoja/app/assets.go
      Note: app HostServices and AssetStore concrete service symbols.
    - Path: go-go-goja/pkg/xgoja/app/factory.go
      Note: |-
        xgoja RuntimeFactory and providerRuntimeModuleSpec symbols.
        xgoja RuntimeFactory and providerRuntimeModuleSpec symbol definitions
    - Path: go-go-goja/pkg/xgoja/app/host.go
      Note: Host and HostOptions generated-app wiring symbols.
    - Path: go-go-goja/pkg/xgoja/app/spec.go
      Note: Runtime embedded app Spec/Runtime/ModuleInstance symbols.
    - Path: go-go-goja/pkg/xgoja/providerapi/capabilities.go
      Note: |-
        SectionContext/ModuleDescriptor/Capability/RuntimeHandle symbols.
        Provider capability/context/descriptor/runtime handle symbol definitions
    - Path: go-go-goja/pkg/xgoja/providerapi/commands.go
      Note: CommandSetProvider, CommandSetContext, and provider RuntimeFactory interface.
    - Path: go-go-goja/pkg/xgoja/providerapi/module.go
      Note: |-
        providerapi Module/ModuleFactory/ModuleContext/HostServices symbols.
        Provider Module/ModuleFactory/ModuleContext/HostServices symbol definitions
    - Path: go-go-goja/pkg/xgoja/providerapi/registry.go
      Note: providerapi Registry and Package catalog symbols.
ExternalSources: []
Summary: Inventory and glossary for generic-sounding Service/Context/Capability/Runtime/Module/Spec/Factory/Registry symbols across xgoja, engine, runtimebridge, runtimeowner, modules, and Geppetto provider layers.
LastUpdated: 2026-06-04T00:00:00Z
WhatFor: Use to disambiguate generic xgoja/go-go-goja symbols and decide which should be renamed, documented, unified, or more clearly separated before GOJA-053 implementation.
WhenToUse: Before modifying ModuleContext, RuntimeModuleContext, RuntimeFactory, provider capabilities, runtimebridge services, or xgoja build/runtime spec types.
---


# Generic Symbol Inventory and Glossary

## Purpose

This document inventories the generic-sounding symbols that make the xgoja stack hard to reason about: `*Service`, `*Context`, `*Capability`, `Runtime*`, `Module*`, `*Spec`, `*Factory`, and `*Registry` names. It records where each symbol is defined, what design pattern it represents, how it differs from similarly named symbols, and whether it should be unified, renamed, or kept separate.

The immediate reason for this document is GOJA-053. The design for pre-`Module.New` config merging depends on understanding exactly which layer owns a module, a runtime, a context, a registry, and a service. The names currently mix several patterns:

- data-transfer specs,
- provider extension capabilities,
- construction factories/builders,
- runtime lifecycle handles,
- dependency/service bundles,
- registries/catalogs/caches,
- CommonJS native module loaders,
- command parsing contexts.

The code is mostly coherent once the layers are separated, but the names do not always make the layer boundaries obvious.

---

## 1. Pattern taxonomy

### 1.1 `Spec`: declarative data shape

A `Spec` in this codebase usually means “a declarative data object,” not a service and not an active runtime. It often has JSON/YAML tags and little behavior. Examples:

- `buildspec.Spec`: build-time `xgoja.yaml` model.
- `app.Spec`: runtime embedded JSON model.
- `engine.NativeModuleSpec`: an active registration instruction, not just data.
- `engine.RuntimeModuleSpec`: an interface for runtime registration behavior.

Important naming problem: the word `Spec` is used both for pure DTOs (`app.Spec`) and executable registration instructions (`RuntimeModuleSpec`). Those are different patterns.

### 1.2 `Context`: either cancellation context or parameter bundle

There are two meanings mixed under `Context`:

1. A real Go `context.Context` for cancellation/deadlines.
2. A parameter bundle passed into a hook/factory/initializer.

Examples:

- `context.Context`: cancellation/deadline object.
- `providerapi.ModuleContext`: parameter bundle for provider module setup, containing a field also named `Context`.
- `engine.RuntimeModuleContext`: parameter bundle for registering native modules in one runtime.
- `engine.RuntimeContext`: parameter bundle for post-require engine runtime initializers.
- `providerapi.SectionContext`: request metadata for asking providers for public command sections.
- `providerapi.CommandSetContext`: parameter bundle for provider command factories.

Important naming problem: `ModuleContext.Context` should almost certainly be renamed to `StartupContext` or `SetupContext` because it is not the runtime lifetime context.

### 1.3 `Capability`: optional provider extension interface

A capability is a provider package extension discovered by type assertion. It is registered into `providerapi.Registry` with `WithPackageCapability` and later applied to selected modules.

Examples:

- `PackageCapability`: marker interface with `CapabilityID()`.
- `ConfigSectionCapability`: current public command section capability; name is ambiguous.
- `RuntimeInitializerCapability`: post-runtime initializer fed parsed command values.

Important naming problem: `ConfigSectionCapability` sounds like internal module config. It currently means “public Glazed sections for generated commands,” so it should be renamed or aliased.

### 1.4 `Service(s)`: dependency bundle or host/runtime bridge

Services are objects that expose capabilities owned by a host or runtime:

- `providerapi.HostServices`: provider-facing host services interface.
- `app.HostServices`: concrete generated-app implementation of provider-facing host services.
- `runtimebridge.RuntimeServices`: services stored per Goja VM for native module code.
- Geppetto provider `HostServices`: provider-specific host requirements.

Important naming problem: several packages define `HostServices`. They are different interfaces/bundles at different layers.

### 1.5 `Factory` / `Builder`: construction pattern

Factories/builders construct complex objects:

- `app.RuntimeFactory`: xgoja profile-to-engine-runtime adapter.
- `providerapi.RuntimeFactory`: provider-facing interface for command providers.
- `engine.FactoryBuilder`: mutable composition builder.
- `engine.Factory`: immutable low-level runtime factory.
- `providerapi.ModuleFactory`: provider module setup function returning `require.ModuleLoader`.
- `providerapi.CommandSetProviderFactory`: provider command bundle setup function.

Important naming problem: `app.RuntimeFactory`, `providerapi.RuntimeFactory`, and `engine.Factory` are easy to conflate, but they serve different layers.

### 1.6 `Registry`: catalog or cache

Registries differ sharply:

- `providerapi.Registry`: xgoja provider metadata catalog.
- `modules.Registry`: global/default native module catalog.
- `require.Registry`: goja_nodejs per-runtime CommonJS loader registry/cache.
- `runtimebridge` internal maps: VM-to-runtime-services lookup, not named registry but registry-like.

Important naming problem: `Registry` does not say whether it is build metadata, native module defaults, or per-runtime CommonJS resolution.

### 1.7 `Runtime`: either a concrete VM or a profile DTO

`Runtime` is overloaded:

- `engine.Runtime`: concrete owned VM/event-loop/require/runtime-owner instance.
- `app.Runtime`: declarative runtime profile containing selected module instances.
- `buildspec.Runtime`: build-time declarative runtime profile.
- `moduleRuntime` in Geppetto: provider-specific per-VM module state.

Important naming problem: `app.Runtime` and `buildspec.Runtime` are not runtimes; they are runtime profiles.

### 1.8 `Module`: provider module, CommonJS module, runtime module spec, or module instance

`Module` also spans multiple layers:

- `providerapi.Module`: provider metadata plus factory.
- `goja_nodejs/require.ModuleLoader`: CommonJS loader that populates `module.exports`.
- `engine.RuntimeModuleSpec`: instruction to register native loaders into one runtime.
- `modules.NativeModule`: default native module implementation interface.
- `app.ModuleInstance` / `buildspec.ModuleInstance`: selected provider module in a runtime profile.

Important naming problem: `Module.New` builds a loader; `ModuleLoader` runs when JS requires the module. These are separate phases.

---

## 2. Source map by layer

```text
Build/codegen layer
  cmd/xgoja/internal/buildspec/*
  cmd/xgoja/internal/generate/*

Generated app layer
  pkg/xgoja/app/*

Provider API layer
  pkg/xgoja/providerapi/*
  pkg/xgoja/providerutil/*

Engine/runtime layer
  engine/*

Bridge/owner layer
  pkg/runtimebridge/*
  pkg/runtimeowner/*

Default native module layer
  modules/*

Provider-specific example layer
  geppetto/pkg/js/modules/geppetto/*
```

Layer boundaries matter. Many confusing names are acceptable inside their own layer but become confusing when a design document mentions several layers at once.

---

## 3. Spec and DTO inventory

### 3.1 `buildspec.Spec`

- **Defined in:** `go-go-goja/cmd/xgoja/internal/buildspec/spec.go`
- **Pattern:** build-time DTO / YAML schema.
- **Represents:** the raw `xgoja.yaml` build specification after YAML parsing and defaults.
- **Important fields:** `Go`, `Target`, `Packages`, `Runtimes`, `Commands`, `CommandProviders`, `JSVerbs`, `Help`, `Assets`, `BaseDir`.
- **Differs from:** `app.Spec`, which is runtime embedded JSON and omits build-only fields.
- **Keep separate?** Yes. Build-time data includes Go module/build concerns that generated binaries do not need.
- **Recommended naming/documentation:** In docs, call it `BuildSpec` even if the Go type remains `buildspec.Spec`.

### 3.2 `app.Spec`

- **Defined in:** `go-go-goja/pkg/xgoja/app/spec.go`
- **Pattern:** runtime DTO / embedded JSON schema.
- **Represents:** normalized runtime spec decoded by the generated target program.
- **Important fields:** `Name`, `AppName`, `EnvPrefix`, `Config`, `Target`, `Packages`, `Runtimes`, `Commands`, `CommandProviders`, `JSVerbs`, `Help`, `Assets`.
- **Differs from:** `buildspec.Spec`; it is not the source YAML and does not include Go build details.
- **Keep separate?** Yes, but documentation should consistently call it `RuntimeSpec` or `EmbeddedSpec`.
- **Could be renamed?** `app.RuntimeSpec` would be clearer than `app.Spec`, but package name already scopes it.

### 3.3 `buildspec.Runtime` and `app.Runtime`

- **Defined in:**
  - `go-go-goja/cmd/xgoja/internal/buildspec/spec.go`
  - `go-go-goja/pkg/xgoja/app/spec.go`
- **Pattern:** runtime profile DTO, not a concrete runtime.
- **Represents:** named runtime profile under `runtimes:` with selected modules.
- **Differs from:** `engine.Runtime`, the concrete VM/event-loop instance.
- **Keep separate?** They should remain distinct from `engine.Runtime`.
- **Recommended rename:** `RuntimeProfile` in both buildspec and app packages would be much clearer.

### 3.4 `buildspec.ModuleInstance` and `app.ModuleInstance`

- **Defined in:**
  - `go-go-goja/cmd/xgoja/internal/buildspec/spec.go`
  - `go-go-goja/pkg/xgoja/app/spec.go`
- **Pattern:** selected module DTO.
- **Represents:** a selected provider module in a runtime profile: `Package`, `Name`, optional `As`, and static `Config`.
- **Differs from:**
  - `providerapi.Module`: provider's module definition.
  - `engine.RuntimeModuleSpec`: runtime registration instruction.
  - `require.ModuleLoader`: CommonJS loader.
- **Keep separate?** Yes.
- **GOJA-053 relevance:** `Config map[string]any` is currently marshaled directly into `providerapi.ModuleContext.Config` before `Module.New`. GOJA-053 wants to parse/merge this before `Module.New`.
- **Recommended documentation:** Add comments explaining the conversion chain: `ModuleInstance` → `providerapi.Module` + `providerRuntimeModuleSpec` → `ModuleContext` → `require.ModuleLoader`.

### 3.5 `ConfigSpec`, `TargetSpec`, `PackageSpec`, `CommandsSpec`, `CommandSpec`, `CommandProviderInstance`, `JSVerbSourceSpec`, `HelpSpec`, `HelpSourceSpec`, `AssetSourceSpec`

- **Defined in:** both `cmd/xgoja/internal/buildspec/spec.go` and `pkg/xgoja/app/spec.go`.
- **Pattern:** declarative DTOs.
- **Represent:** slices/sections of buildspec/runtime spec.
- **Differs from:** active factories/providers/runtime objects.
- **Keep separate?** Yes, but docs should group them under “spec DTOs” so they are not confused with active services.
- **Potential cleanup:**
  - `CommandProviderInstance` is a runtime-selected provider command set, not the provider definition (`providerapi.CommandSetProvider`). The name is reasonable.
  - `ConfigSpec` means generated command config-file support, not module config. This should be documented because GOJA-053 uses “config” in several ways.

### 3.6 `generate.Options`

- **Defined in:** `go-go-goja/cmd/xgoja/internal/generate/generate.go`
- **Pattern:** codegen options.
- **Represents:** inputs controlling generated `go.mod`, such as xgoja module version and local replacement.
- **Differs from:** `engine.Options`, `runtimeowner.Options`, `geppetto.Options`, and command settings.
- **Keep separate?** Yes.
- **Recommended docs:** Mention this is build generator config, not runtime config.

---

## 4. Provider API inventory

### 4.1 `providerapi.Registry`

- **Defined in:** `go-go-goja/pkg/xgoja/providerapi/registry.go`
- **Pattern:** metadata catalog.
- **Represents:** provider packages compiled into a generated target: modules, capabilities, command providers, help sources, jsverb sources.
- **Differs from:**
  - `require.Registry`: per-runtime CommonJS/native module registry.
  - `modules.Registry`: global default module catalog.
- **Keep separate?** Yes.
- **Recommended documentation:** Always call this “provider registry” in prose, never just “registry.”

### 4.2 `providerapi.Package`

- **Defined in:** `go-go-goja/pkg/xgoja/providerapi/registry.go`
- **Pattern:** provider package catalog entry.
- **Represents:** one provider package ID and its registered entries.
- **Differs from:** Go package/import path; it is runtime metadata keyed by provider ID.
- **Keep separate?** Yes.
- **Potential cleanup:** Minimal; comments are adequate.

### 4.3 `providerapi.Entry`

- **Defined in:** `go-go-goja/pkg/xgoja/providerapi/registry.go`
- **Pattern:** registration command object.
- **Represents:** something that can be applied to a provider package during `registry.Package(...)`.
- **Differs from:** provider capabilities and modules themselves; it is a registration wrapper pattern.
- **Keep separate?** Yes, internal API convenience.

### 4.4 `providerapi.Module`

- **Defined in:** `go-go-goja/pkg/xgoja/providerapi/module.go`
- **Pattern:** provider module definition / metadata plus factory.
- **Represents:** a module that can be selected in `xgoja.yaml` and later exposed through `require(alias)`.
- **Important fields:** `Name`, `DefaultAs`, `Description`, `ConfigSchema`, `New`.
- **Differs from:**
  - `app.ModuleInstance`: selection/config instance.
  - `require.ModuleLoader`: actual CommonJS module loader.
  - `engine.RuntimeModuleSpec`: engine registration instruction.
- **Keep separate?** Yes.
- **Potential cleanup:** Add comments emphasizing `New` runs during runtime construction; the returned loader runs when JS requires the alias.

### 4.5 `providerapi.ModuleFactory`

- **Defined in:** `go-go-goja/pkg/xgoja/providerapi/module.go`
- **Pattern:** factory function.
- **Signature:** `func(ModuleContext) (require.ModuleLoader, error)`.
- **Represents:** provider setup logic that converts module config/host services into a CommonJS loader.
- **Differs from:** `require.ModuleLoader`, which has signature `func(*goja.Runtime, *goja.Object)` and populates `module.exports`.
- **Keep separate?** Yes.
- **Recommended documentation:** “ModuleFactory builds/configures the loader; ModuleLoader populates exports.”

### 4.6 `providerapi.ModuleContext`

- **Defined in:** `go-go-goja/pkg/xgoja/providerapi/module.go`
- **Pattern:** provider-facing setup context / parameter bundle.
- **Represents:** data passed to `providerapi.Module.New`.
- **Fields:** `Context`, `Name`, `As`, `Config`, `Host`, `RuntimeOwner`.
- **Differs from:**
  - `engine.RuntimeModuleContext`: richer engine setup context.
  - `engine.RuntimeContext`: post-require engine initializer context.
  - `runtimebridge.RuntimeServices`: VM lookup services used after runtime exists.
- **Keep separate?** Yes, but rename fields.
- **High-priority cleanup:** Rename `Context` to `StartupContext` or `SetupContext`. Current name hides that it is the runtime construction context, not runtime lifetime context.
- **GOJA-053 relevance:** `Config` is where merged static+command-derived config should arrive before `Module.New`.

### 4.7 `providerapi.AssetResolver` and `providerapi.HostServices`

- **Defined in:** `go-go-goja/pkg/xgoja/providerapi/module.go`
- **Pattern:** provider-facing service interface.
- **Represents:** host-provided services available to module factories.
- **Differs from:** `app.HostServices`, the concrete implementation; provider-specific `HostServices` such as Geppetto's interface; `runtimebridge.RuntimeServices`.
- **Keep separate?** Yes.
- **Potential cleanup:** Rename to `ProviderHostServices` if package name is not enough in docs.

### 4.8 `providerapi.SectionContext`

- **Defined in:** `go-go-goja/pkg/xgoja/providerapi/capabilities.go`
- **Pattern:** request metadata context, not cancellation context.
- **Represents:** why xgoja is asking a provider for public Glazed sections.
- **Fields:** `CommandName`, `CommandProviderID`, `RuntimeProfile`, `PackageID`, `ModuleID`.
- **Differs from:** `context.Context` and runtime setup contexts.
- **Keep separate?** Yes.
- **Potential rename:** `SectionRequest` or `CommandSectionContext` would be clearer.

### 4.9 `providerapi.ModuleDescriptor`

- **Defined in:** `go-go-goja/pkg/xgoja/providerapi/capabilities.go`
- **Pattern:** selected-module descriptor / metadata aggregate.
- **Represents:** resolved provider module plus selected alias and package capabilities.
- **Differs from:** `ModuleInstance` raw DTO; `providerapi.Module` definition.
- **Keep separate?** Yes.
- **Potential rename:** `SelectedModuleDescriptor` would better reflect that it is derived from a selected runtime profile.

### 4.10 `providerapi.PackageCapability`

- **Defined in:** `go-go-goja/pkg/xgoja/providerapi/capabilities.go`
- **Pattern:** marker interface for optional provider extension.
- **Represents:** a package-scoped optional extension registered with `WithPackageCapability`.
- **Differs from:** module definitions; command providers.
- **Keep separate?** Yes.
- **Potential rename:** `ProviderCapability` may be simpler, but `PackageCapability` accurately says package-scoped.

### 4.11 `providerapi.ConfigSectionCapability`

- **Defined in:** `go-go-goja/pkg/xgoja/providerapi/capabilities.go`
- **Pattern:** optional provider extension interface.
- **Represents today:** public Glazed sections attached to generated commands and package-owned command providers; these sections feed flags/config/env/default parsing.
- **Differs from:** the proposed GOJA-053 internal module config schema capability.
- **Keep separate?** Yes, from internal module config schema.
- **High-priority cleanup:** Rename. Candidate names:
  - `CommandLineFlagsSectionCapability` — matches user preference but underplays config/env sources.
  - `CommandInputSectionCapability` — broader and more accurate.
  - `PublicCommandSectionCapability` — emphasizes user-facing command surface.
- **Recommendation:** Use `CommandLineFlagsSectionCapability` if CLI flags are the primary mental model, but document: “also parsed from config files/env/defaults by Glazed.”

### 4.12 `providerapi.RuntimeHandle`

- **Defined in:** `go-go-goja/pkg/xgoja/providerapi/capabilities.go`
- **Pattern:** narrow provider-facing runtime handle.
- **Represents:** minimal operations available to provider runtime initializers after a runtime exists.
- **Methods:** `Runtime() *goja.Runtime`, `Close(context.Context) error`.
- **Differs from:** `engine.Runtime`, which exposes VM, Require, Loop, Owner, Values, context, closers.
- **Keep separate?** Yes.
- **Potential rename:** `RuntimeInitializerHandle` would clarify where it is used.

### 4.13 `providerapi.RuntimeCloserRegistry`

- **Defined in:** `go-go-goja/pkg/xgoja/providerapi/capabilities.go`
- **Pattern:** optional extension interface on runtime handles.
- **Represents:** ability to attach cleanup hooks to the underlying runtime.
- **Differs from:** `RuntimeHandle`; it is optional and implemented when closers are supported.
- **Keep separate?** Yes.
- **Potential cleanup:** Good name; document that providers should type-assert this when starting resources.

### 4.14 `providerapi.RuntimeInitializerCapability`

- **Defined in:** `go-go-goja/pkg/xgoja/providerapi/capabilities.go`
- **Pattern:** optional provider extension interface / post-runtime initializer.
- **Represents:** hook that configures an already-created runtime from parsed public command values.
- **Differs from:**
  - `engine.RuntimeInitializer`: engine-level hook run inside `engine.Factory.NewRuntime` after `require` setup.
  - proposed pre-`Module.New` config mapping capability.
- **Keep separate?** Yes, but rename/document phase.
- **Potential rename:** `PostRuntimeInitializerCapability` or `RuntimeCommandValuesInitializerCapability`.
- **GOJA-053 relevance:** This is too late for config needed by `Module.New`.

### 4.15 `providerapi.CommandSetProviderFactory`, `CommandSetProvider`, `CommandSetContext`, `CommandSet`

- **Defined in:** `go-go-goja/pkg/xgoja/providerapi/commands.go`
- **Pattern:** provider-owned command factory and result bundle.
- **Represents:** provider packages can contribute their own Glazed command sets to generated binaries.
- **Differs from:** built-in xgoja commands such as `run`, `eval`, `repl`, and `verbs`.
- **Keep separate?** Yes.
- **Potential cleanup:** `CommandSetContext.Context` likely has the same generic-name problem as `ModuleContext.Context`; document as setup/command construction context.

### 4.16 `providerapi.RuntimeFactory` interface

- **Defined in:** `go-go-goja/pkg/xgoja/providerapi/commands.go`
- **Pattern:** provider-facing factory interface.
- **Represents:** way for provider-owned command sets to create xgoja runtimes from named profiles.
- **Differs from:**
  - `app.RuntimeFactory`: concrete implementation.
  - `engine.Factory`: low-level VM factory.
- **Keep separate?** Yes.
- **GOJA-053 impact:** May need a value-aware method if provider command sets should get pre-`Module.New` config merging from parsed values.
- **Potential rename:** `XGojaRuntimeFactory` or `RuntimeProfileFactory`.

---

## 5. Generated app layer inventory

### 5.1 `app.Options`

- **Defined in:** `go-go-goja/pkg/xgoja/app/root.go`
- **Pattern:** root command constructor options.
- **Represents:** inputs to `NewRootCommand`: providers, embedded spec JSON, output writer, embedded filesystems, parser middleware override.
- **Differs from:** build generator options, runtime owner options, Geppetto module options.
- **Keep separate?** Yes.
- **Potential rename:** `RootCommandOptions` would be clearer than generic `Options`.

### 5.2 `app.Host`

- **Defined in:** `go-go-goja/pkg/xgoja/app/host.go`
- **Pattern:** generated app wiring object.
- **Represents:** provider registry, runtime spec, runtime factory, embedded file systems, host services, output, and parser middleware function.
- **Differs from:** provider host services interface; it is not a Goja runtime host.
- **Keep separate?** Yes.
- **Potential documentation:** “Host is generated CLI host/wiring, not VM host.”

### 5.3 `app.HostOptions`

- **Defined in:** `go-go-goja/pkg/xgoja/app/host.go`
- **Pattern:** generated app host constructor options.
- **Represents:** optional embedded FSs, output writer, parser middleware function.
- **Differs from:** provider host services and runtime services.
- **Keep separate?** Yes.

### 5.4 `app.HostServices`

- **Defined in:** `go-go-goja/pkg/xgoja/app/assets.go`
- **Pattern:** concrete service bundle implementing provider-facing `providerapi.HostServices`.
- **Represents:** generated app services exposed to provider module factories.
- **Currently contains:** `Assets *AssetStore`.
- **Differs from:**
  - `providerapi.HostServices`: interface.
  - Geppetto provider `HostServices`: provider-specific required host interface.
  - `runtimebridge.RuntimeServices`: VM runtime services.
- **Keep separate?** Yes.
- **Potential rename:** `DefaultHostServices` or `GeneratedHostServices`.

### 5.5 `app.AssetStore`

- **Defined in:** `go-go-goja/pkg/xgoja/app/assets.go`
- **Pattern:** concrete service implementation.
- **Represents:** embedded asset lookup by asset ID.
- **Differs from:** `providerapi.AssetResolver`, which is the interface.
- **Keep separate?** Yes.

### 5.6 `app.RuntimeFactory`

- **Defined in:** `go-go-goja/pkg/xgoja/app/factory.go`
- **Pattern:** xgoja profile-to-runtime adapter.
- **Represents:** concrete factory that uses `app.Spec` and `providerapi.Registry` to create `engine.Runtime` instances for named runtime profiles.
- **Differs from:**
  - `engine.Factory`: low-level immutable VM construction factory.
  - `providerapi.RuntimeFactory`: interface exposed to provider command sets.
- **Keep separate?** Yes.
- **Potential rename:** `RuntimeProfileFactory` or `XGojaRuntimeFactory`.
- **GOJA-053 relevance:** This is where parsed command values should enter pre-`Module.New` config merging.

### 5.7 `app.JSRuntime`

- **Defined in:** `go-go-goja/pkg/xgoja/app/factory.go`
- **Pattern:** type alias.
- **Definition:** `type JSRuntime = engine.Runtime`.
- **Represents:** app-layer name for concrete engine runtime.
- **Differs from:** `app.Runtime` DTO profile.
- **Keep?** Maybe.
- **Potential cleanup:** This alias can add confusion because `app.Runtime` already exists as DTO. Prefer using `*engine.Runtime` in docs or rename app DTO to `RuntimeProfile`.

### 5.8 `providerRuntimeModuleSpec` (unexported)

- **Defined in:** `go-go-goja/pkg/xgoja/app/factory.go`
- **Pattern:** adapter object implementing `engine.RuntimeModuleSpec`.
- **Represents:** selected provider module instance converted into an engine runtime registration instruction.
- **Differs from:**
  - `providerapi.Module`: provider definition.
  - `app.ModuleInstance`: selected DTO.
  - `engine.NativeModuleSpec`: simple direct loader spec.
- **Keep separate?** Yes.
- **Potential rename:** `providerModuleRegistrationSpec` would better describe its role.
- **GOJA-053 relevance:** Its `RegisterRuntimeModule` method currently marshals `ModuleInstance.Config` and calls `module.New`.

### 5.9 `runtimeHandle` (unexported app type)

- **Defined in:** `go-go-goja/pkg/xgoja/app/module_sections.go`
- **Pattern:** adapter / narrowed handle.
- **Represents:** wraps `*engine.Runtime` to implement provider-facing `providerapi.RuntimeHandle` and `RuntimeCloserRegistry`.
- **Differs from:** `engine.Runtime` itself.
- **Keep separate?** Yes, because it enforces provider API narrowness.
- **Potential rename:** `providerRuntimeHandle`.

---

## 6. Engine layer inventory

### 6.1 `engine.FactoryBuilder`

- **Defined in:** `go-go-goja/engine/factory.go`
- **Pattern:** mutable builder.
- **Represents:** construction-time composition of module specs, module middleware, runtime initializers, and require options before freezing into `engine.Factory`.
- **Differs from:** `engine.Factory`, which is immutable and creates runtimes.
- **Keep separate?** Yes.
- **Potential documentation:** Emphasize builder is not per runtime; it is a plan builder.

### 6.2 `engine.Factory`

- **Defined in:** `go-go-goja/engine/factory.go`
- **Pattern:** immutable low-level runtime factory.
- **Represents:** frozen engine composition plan that can create `engine.Runtime` instances.
- **Differs from:** `app.RuntimeFactory` profile adapter.
- **Keep separate?** Yes.
- **Potential rename:** `RuntimeEngineFactory` would reduce confusion but is a larger API change.

### 6.3 `engine.Runtime`

- **Defined in:** `go-go-goja/engine/runtime.go`
- **Pattern:** concrete owned runtime/lifecycle object.
- **Represents:** one Goja VM with event loop, `require`, runtime owner, values, lifetime context, and closers.
- **Differs from:** `app.Runtime` / `buildspec.Runtime` runtime profile DTOs.
- **Keep separate?** Yes.
- **Potential cleanup:** Rename spec DTOs instead of this; this is the true runtime.

### 6.4 `engine.RuntimeModuleSpec`

- **Defined in:** `go-go-goja/engine/runtime_modules.go`
- **Pattern:** registration command interface.
- **Represents:** something that can register one or more native modules into a concrete runtime's `require.Registry`.
- **Methods:** `ID()`, `RegisterRuntimeModule(*RuntimeModuleContext, *require.Registry) error`.
- **Differs from:** provider modules and module instances.
- **Keep separate?** Yes.
- **Potential rename:** `NativeModuleRegistration` or `RuntimeModuleRegistrar` would be more descriptive than `Spec`.

### 6.5 `engine.RuntimeModuleContext`

- **Defined in:** `go-go-goja/engine/runtime_modules.go`
- **Pattern:** engine setup parameter bundle.
- **Represents:** runtime-scoped objects available while registering native modules before `require` is enabled.
- **Fields:** startup `Context`, `VM`, `Loop`, `Owner`, `AddCloser`, `Values`.
- **Differs from:**
  - `providerapi.ModuleContext`: narrowed provider-facing setup context.
  - `engine.RuntimeContext`: initializer context after `require` exists.
- **Keep separate?** Yes.
- **Potential cleanup:** Rename `Context` field to `StartupContext`, matching `engine.WithStartupContext`.

### 6.6 `engine.RuntimeInitializer`

- **Defined in:** `go-go-goja/engine/module_specs.go`
- **Pattern:** engine-level post-require initialization hook.
- **Represents:** hook run inside `engine.Factory.NewRuntime` after `require` and globals are installed.
- **Differs from:** `providerapi.RuntimeInitializerCapability`, which is xgoja provider-level and run after `app.RuntimeFactory.NewRuntime` returns.
- **Keep separate?** Yes, but document phase differences.
- **Potential rename:** `EngineRuntimeInitializer` if exported across layers.

### 6.7 `engine.RuntimeContext`

- **Defined in:** `go-go-goja/engine/module_specs.go`
- **Pattern:** engine initializer parameter bundle.
- **Represents:** objects passed to `engine.RuntimeInitializer.InitRuntime`.
- **Fields:** startup `Context`, `VM`, `Require`, `Loop`, `Owner`, `Values`.
- **Differs from:** `RuntimeModuleContext`, which is before `require` is enabled and includes `AddCloser` but not `Require`.
- **Keep separate?** Yes.
- **Potential cleanup:** Document “RuntimeModuleContext is for registering modules; RuntimeContext is for post-require initializers.”

### 6.8 `engine.NativeModuleSpec`

- **Defined in:** `go-go-goja/engine/module_specs.go`
- **Pattern:** simple runtime module registration spec.
- **Represents:** direct `ModuleName` + `require.ModuleLoader` pair registered into the require registry.
- **Differs from:** `providerRuntimeModuleSpec`, which constructs the loader from provider config first.
- **Keep separate?** Yes.
- **Potential rename:** Good enough, though `NativeModuleRegistrationSpec` is more precise.

### 6.9 `namedDefaultRegistryModulesSpec`, `processModuleSpec`, `processEnvInitializer` (unexported)

- **Defined in:** `go-go-goja/engine/module_specs.go`
- **Pattern:** internal runtime module specs and initializer implementations.
- **Represents:** default-registry native modules and `process` module/global installation.
- **Differs from:** provider API modules.
- **Keep separate?** Yes, internal implementations.

### 6.10 `engine.ModuleSelector` and `engine.ModuleMiddleware`

- **Defined in:** `go-go-goja/engine/module_middleware.go`
- **Pattern:** functional middleware / selector pipeline.
- **Represents:** default-registry module selection transformations.
- **Differs from:** provider package capabilities; it operates on built-in/default module names.
- **Keep separate?** Yes.
- **Potential cleanup:** Docs should call this “default module selection middleware,” not provider middleware.

### 6.11 `engine.ModuleRootsOptions`

- **Defined in:** `go-go-goja/engine/module_roots.go`
- **Pattern:** options struct for module path resolution.
- **Represents:** how to derive require global folders from a script path.
- **Differs from:** runtime factory options and provider module options.
- **Keep separate?** Yes.
- **Potential rename:** Good enough because name is specific.

### 6.12 `runtimeOptions` and `builderSettings` (unexported)

- **Defined in:** `go-go-goja/engine/options.go`
- **Pattern:** internal settings structs.
- **Represent:** normalized runtime options and builder settings.
- **Differs from:** exported `Options` in other packages.
- **Keep separate?** Yes.
- **Potential cleanup:** None; unexported names are scoped.

### 6.13 `runtimebridgeOwner` (unexported)

- **Defined in:** `go-go-goja/engine/factory.go`
- **Pattern:** adapter.
- **Represents:** wraps `runtimeowner.RuntimeOwner` to satisfy `runtimebridge.RuntimeOwner` without exposing full owner interface.
- **Differs from:** concrete runtime owner implementation.
- **Keep separate?** Yes, this is a good boundary adapter.

---

## 7. Runtime bridge and owner inventory

### 7.1 `runtimebridge.RuntimeServices`

- **Defined in:** `go-go-goja/pkg/runtimebridge/runtimebridge.go`
- **Pattern:** VM-associated service bundle.
- **Represents:** runtime lifetime context, event loop, and owner scheduler discoverable from a `*goja.Runtime`.
- **Differs from:** provider host services; this is runtime-owned and VM-indexed.
- **Keep separate?** Yes.
- **Potential documentation:** Add package-level note: runtimebridge is a service lookup/context propagation bridge, not a module registry.

### 7.2 `runtimebridge.RuntimeOwner`

- **Defined in:** `go-go-goja/pkg/runtimebridge/runtimebridge.go`
- **Pattern:** narrow interface to avoid import cycles.
- **Represents:** subset of owner scheduling services: `Call` and `Post`.
- **Differs from:** `runtimeowner.RuntimeOwner`, which also has `WaitIdle`, `Shutdown`, `IsClosed`.
- **Keep separate?** Yes, because it avoids a package cycle and narrows exposure.
- **Potential rename:** `OwnerScheduler` would distinguish it from full `runtimeowner.RuntimeOwner`.

### 7.3 `runtimebridge.linkedContext` (unexported)

- **Defined in:** `go-go-goja/pkg/runtimebridge/runtimebridge.go`
- **Pattern:** internal cancellation helper.
- **Represents:** caller context linked to runtime lifetime context.
- **Keep separate?** Yes, internal.

### 7.4 `runtimeowner.RuntimeOwner`

- **Defined in:** `go-go-goja/pkg/runtimeowner/types.go`
- **Pattern:** owner-thread scheduler/lifecycle interface.
- **Represents:** safe serialized execution against a Goja runtime: `Call`, `Post`, `WaitIdle`, `Shutdown`, `IsClosed`.
- **Differs from:** runtimebridge's narrower `RuntimeOwner` and providerapi's `ModuleContext.RuntimeOwner` field.
- **Keep separate?** Yes.
- **Potential documentation:** Call it “owner-thread runtime scheduler” in docs.

### 7.5 `runtimeowner.Scheduler`, `CallFunc`, `PostFunc`, `Options`, and `runtimeOwner`

- **Defined in:**
  - `go-go-goja/pkg/runtimeowner/types.go`
  - `go-go-goja/pkg/runtimeowner/runner.go`
- **Pattern:** scheduling abstraction, callback types, configuration, concrete implementation.
- **Represents:** implementation details for owner-thread execution.
- **Differs from:** runtimebridge services and provider contexts.
- **Keep separate?** Yes.
- **Potential cleanup:** `Options` could become `RuntimeOwnerOptions` for external clarity, but package name scopes it.

---

## 8. Default native module layer inventory

### 8.1 `modules.NativeModule`

- **Defined in:** `go-go-goja/modules/common.go`
- **Pattern:** native module implementation interface.
- **Represents:** built-in/default native module with name, docs, and loader.
- **Differs from:** `providerapi.Module`, which is provider metadata and factory; `require.ModuleLoader`, which is just the loader function.
- **Keep separate?** Yes.
- **Potential cleanup:** Docs should call this “default-registry native module,” not provider module.

### 8.2 `modules.Registry`

- **Defined in:** `go-go-goja/modules/common.go`
- **Pattern:** native module catalog.
- **Represents:** collection of built-in/default native modules that can be enabled into a `require.Registry`.
- **Differs from:**
  - `providerapi.Registry`: provider package metadata catalog.
  - `require.Registry`: per-runtime CommonJS registry.
- **Keep separate?** Yes.
- **Potential cleanup:** In docs, call it `modules.DefaultRegistry` or “default native module registry.”

### 8.3 `require.ModuleLoader` and `require.Registry` (goja_nodejs)

- **Defined in:** Go module cache, package `github.com/dop251/goja_nodejs/require`.
- **Pattern:** CommonJS module loader and per-runtime registry/cache.
- **Represents:**
  - `ModuleLoader`: `func(*goja.Runtime, *goja.Object)` that populates `module.exports`.
  - `Registry`: cache/resolver/loader registry enabled onto a concrete VM.
- **Differs from:** all xgoja/provider registries.
- **Keep separate?** Yes, third-party API.
- **Documentation need:** xgoja docs should explicitly say `providerapi.Module.New` returns a `require.ModuleLoader`, and that loader is called when JS `require(alias)` resolves the native module.

---

## 9. Provider-specific Geppetto symbols used as examples

### 9.1 `geppetto.Options`

- **Defined in:** `geppetto/pkg/js/modules/geppetto/module.go`
- **Pattern:** provider module behavior options.
- **Represents:** concrete configuration for Geppetto's native module loader and per-runtime module behavior.
- **Differs from:** xgoja config DTOs and engine options.
- **Keep separate?** Yes.
- **GOJA-053 relevance:** `providerapi.Module.New` builds these options from `ModuleContext.Config` and host services.

### 9.2 `geppetto.moduleRuntime` (unexported)

- **Defined in:** `geppetto/pkg/js/modules/geppetto/module.go`
- **Pattern:** provider-specific per-VM module state.
- **Represents:** state created when Geppetto module loader installs exports into a Goja runtime.
- **Differs from:** `engine.Runtime` and xgoja runtime profiles.
- **Keep separate?** Yes.
- **Potential documentation:** In provider docs, call these “module runtime state” to distinguish from engine runtime.

### 9.3 `geppetto/provider.HostServices` and `StorageHostServices`

- **Defined in:** `geppetto/pkg/js/modules/geppetto/provider/provider.go`
- **Pattern:** provider-specific host service contract.
- **Represents:** extra methods Geppetto requires from xgoja host services.
- **Differs from:** generic `providerapi.HostServices`, which only promises asset resolution.
- **Keep separate?** Yes.
- **Potential cleanup:** Rename locally to `GeppettoHostServices` and `GeppettoStorageHostServices` or keep package qualification in docs.

### 9.4 `geppetto.StorageOptions`, `TurnStore`, `TurnStoreQuery`, `TurnStoreSnapshot`

- **Defined in:** `geppetto/pkg/js/modules/geppetto/api_turn_store.go`
- **Pattern:** provider domain service/data contracts.
- **Represents:** host-backed turn storage exposed through Geppetto JS API.
- **Differs from:** xgoja runtime services and host services; these are Geppetto domain concepts.
- **Keep separate?** Yes.
- **GOJA-053 relevance:** turn-store settings should likely be simplified and mapped from public flags/config into Geppetto module config before `Module.New`.

---

## 10. Confusing clusters and recommendations

### 10.1 The `Runtime` cluster

| Symbol | Actual meaning | Recommendation |
|---|---|---|
| `engine.Runtime` | Concrete VM/event-loop/require/owner lifecycle object | Keep name. This is the true runtime. |
| `app.Runtime` | Runtime profile DTO from embedded spec | Rename/docs as `RuntimeProfile`. |
| `buildspec.Runtime` | Runtime profile DTO from YAML | Rename/docs as `RuntimeProfile`. |
| `app.JSRuntime` | Alias to `engine.Runtime` | Consider removing alias or avoid in docs. |
| `moduleRuntime` in Geppetto | Provider-specific per-VM module state | Keep unexported; document in provider context only. |

Main recommendation: rename DTO runtimes, not concrete `engine.Runtime`.

### 10.2 The `Factory` cluster

| Symbol | Actual meaning | Recommendation |
|---|---|---|
| `engine.FactoryBuilder` | Mutable engine composition builder | Keep. |
| `engine.Factory` | Immutable concrete runtime factory | Keep, but document as engine-level. |
| `app.RuntimeFactory` | xgoja runtime-profile adapter | Rename/docs as `RuntimeProfileFactory` or `XGojaRuntimeFactory`. |
| `providerapi.RuntimeFactory` | Provider-facing interface to create xgoja runtimes | Rename/docs as `RuntimeProfileFactory` interface. |
| `providerapi.ModuleFactory` | Builds a CommonJS `ModuleLoader` | Keep, but document with loader distinction. |
| `CommandSetProviderFactory` | Builds provider-owned command sets | Keep. |

Main recommendation: call `app.RuntimeFactory` “xgoja runtime profile factory” in docs.

### 10.3 The `Context` cluster

| Symbol | Actual meaning | Recommendation |
|---|---|---|
| `context.Context` | Cancellation/deadline/tracing context | Keep. |
| `ModuleContext.Context` | Startup/setup context for `Module.New` | Rename to `StartupContext`/`SetupContext`. |
| `RuntimeModuleContext.Context` | Startup/setup context for native module registration | Rename field to `StartupContext` or document. |
| `RuntimeContext.Context` | Startup/setup context for engine runtime initializers | Rename field to `StartupContext` or document. |
| `SectionContext` | Public section request metadata | Rename to `SectionRequest` or `CommandSectionContext`. |
| `CommandSetContext.Context` | Context passed during provider command set construction | Document phase; maybe `SetupContext`. |
| `runtimebridge.CurrentOwnerContext` | Dynamic current owner-call context | Keep; name is descriptive. |

Main recommendation: reserve bare field name `Context` only when the struct is primarily a context wrapper, otherwise use `StartupContext`, `LifetimeContext`, or `RequestContext`.

### 10.4 The `Capability` cluster

| Symbol | Actual meaning | Recommendation |
|---|---|---|
| `PackageCapability` | Package-scoped optional provider extension | Keep. |
| `ConfigSectionCapability` | Public command sections for flags/config/env/defaults | Rename to `CommandLineFlagsSectionCapability` or `PublicCommandSectionCapability`. |
| `RuntimeInitializerCapability` | Post-runtime provider initializer from parsed values | Rename/docs as post-runtime initializer. |
| Proposed `ModuleConfigSectionCapability` | Internal module config schema | Add separately; do not merge with command flags capability. |
| Proposed mapping capability/helper | Public command values → internal config SectionValues | Add separately or helper; do not overload existing initializer. |

Main recommendation: capabilities should be named by **phase** and **surface**.

### 10.5 The `Services` cluster

| Symbol | Actual meaning | Recommendation |
|---|---|---|
| `providerapi.HostServices` | Generic host services interface for provider modules | Keep, maybe document as generic. |
| `app.HostServices` | Concrete generated-app implementation | Rename/docs as `GeneratedHostServices`. |
| `runtimebridge.RuntimeServices` | VM-associated runtime services | Keep; document as bridge services. |
| `geppetto/provider.HostServices` | Geppetto-specific host contract | Rename/docs as `GeppettoHostServices`. |
| `StorageHostServices` | Geppetto-specific storage host contract | Rename/docs as `GeppettoStorageHostServices`. |

Main recommendation: avoid unqualified `HostServices` in cross-package design docs; always prefix with package/layer.

### 10.6 The `Registry` cluster

| Symbol | Actual meaning | Recommendation |
|---|---|---|
| `providerapi.Registry` | Provider metadata catalog | Call “provider registry.” |
| `modules.Registry` | Default native module catalog | Call “default native module registry.” |
| `require.Registry` | CommonJS/native module resolver/cache for a VM | Call “goja_nodejs require registry.” |
| runtimebridge service map | VM-to-services lookup | Do not call registry in docs. |

Main recommendation: never say just “runtime registry” or “module registry” without qualifying the package/layer.

### 10.7 The `Module` cluster

| Symbol | Actual meaning | Recommendation |
|---|---|---|
| `providerapi.Module` | Provider-defined selectable module | Keep; document phase. |
| `providerapi.ModuleFactory` | Builds/configures require loader | Keep; document. |
| `require.ModuleLoader` | Populates `module.exports` when required | Use explicit term “CommonJS loader.” |
| `engine.RuntimeModuleSpec` | Runtime module registration instruction | Rename/docs as registrar/registration. |
| `engine.NativeModuleSpec` | Simple name+loader runtime registration | Keep/docs. |
| `modules.NativeModule` | Built-in/default native module interface | Keep/docs as default native module. |
| `app.ModuleInstance` | Selected module instance in a runtime profile | Keep or docs as selected provider module. |

Main recommendation: design docs should spell out the chain:

```text
ModuleInstance (selected DTO)
  → providerapi.Module (provider definition)
  → ModuleFactory / Module.New (setup)
  → require.ModuleLoader (CommonJS loader)
  → module.exports (JS object)
```

---

## 11. Proposed cleanup backlog

### High priority for GOJA-053 clarity

1. Rename or alias `ConfigSectionCapability`.
   - Preferred user-requested name: `CommandLineFlagsSectionCapability`.
   - Doc comment must say it also feeds config/env/default parsing.
2. Add a new internal module config capability, separate from public command sections.
   - Candidate: `ModuleConfigSectionCapability`.
3. Rename or document `ModuleContext.Context` as setup/startup context.
   - Candidate field: `StartupContext context.Context`.
4. Rename or document runtime profile DTOs.
   - Candidate: `RuntimeProfile` for `app.Runtime` and `buildspec.Runtime`.

### Medium priority

5. Rename/document `RuntimeInitializerCapability` as post-runtime.
   - Candidate: `PostRuntimeInitializerCapability`.
6. Rename/document `app.RuntimeFactory` as profile adapter.
   - Candidate: `RuntimeProfileFactory`.
7. Add comments distinguishing `RuntimeModuleContext` and `RuntimeContext`.
8. Add docs explaining `providerapi.Registry` vs `require.Registry` vs `modules.Registry`.

### Lower priority

9. Rename provider-specific `HostServices` interfaces to include provider name where exported.
10. Consider more descriptive names for `engine.RuntimeModuleSpec` such as `RuntimeModuleRegistrar`.
11. Avoid `Options` as exported type name in cross-package docs unless package-qualified.

---

## 12. GOJA-053 naming recommendations

For GOJA-053, use these names in new design/API work:

```go
// Existing capability, renamed for clarity.
type CommandLineFlagsSectionCapability interface {
    PackageCapability
    CommandLineFlagsSections(SectionContext) ([]schema.Section, error)
}
```

If this exact method name is too CLI-specific, use:

```go
type PublicCommandSectionCapability interface {
    PublicCommandSections(SectionContext) ([]schema.Section, error)
}
```

For internal module config:

```go
type ModuleConfigSectionCapability interface {
    PackageCapability
    ModuleConfigSection(SectionContext, ModuleDescriptor) (schema.Section, error)
}
```

For mapping public values to internal config values:

```go
type ModuleConfigValuesCapability interface {
    PackageCapability
    ModuleConfigValuesFromSections(
        context.Context,
        ModuleConfigValuesRequest,
    ) (*values.SectionValues, error)
}
```

For setup context clarity:

```go
type ModuleContext struct {
    StartupContext context.Context
    Name           string
    As             string
    Config         json.RawMessage
    Host           HostServices
    RuntimeOwner   runtimeowner.RuntimeOwner
}
```

For docs, use these phrases consistently:

- “provider registry” for `providerapi.Registry`.
- “goja_nodejs require registry” for `require.Registry`.
- “default native module registry” for `modules.Registry`.
- “runtime profile” for `app.Runtime` / `buildspec.Runtime`.
- “concrete runtime” for `engine.Runtime`.
- “module setup factory” for `providerapi.ModuleFactory`.
- “CommonJS module loader” for `require.ModuleLoader`.
- “runtime module registrar” for `engine.RuntimeModuleSpec`.
- “public command section” for current `ConfigSectionCapability` behavior.
- “internal module config section” for GOJA-053 config parsing.

---

## 13. Compact glossary

| Symbol | Defined in | Pattern | Plain-English meaning | Rename/docs? |
|---|---|---|---|---|
| `buildspec.Spec` | `cmd/xgoja/internal/buildspec/spec.go` | DTO | Parsed `xgoja.yaml` build model | Docs: BuildSpec |
| `app.Spec` | `pkg/xgoja/app/spec.go` | DTO | Embedded runtime spec | Docs: RuntimeSpec |
| `buildspec.Runtime` | `cmd/xgoja/internal/buildspec/spec.go` | DTO | Runtime profile in YAML | Rename/docs: RuntimeProfile |
| `app.Runtime` | `pkg/xgoja/app/spec.go` | DTO | Runtime profile in embedded spec | Rename/docs: RuntimeProfile |
| `engine.Runtime` | `engine/runtime.go` | Lifecycle object | Concrete Goja VM runtime | Keep |
| `ModuleInstance` | buildspec/app spec files | DTO | Selected provider module in a profile | Keep, document |
| `providerapi.Module` | `providerapi/module.go` | Provider definition | Selectable provider module metadata/factory | Keep |
| `providerapi.ModuleFactory` | `providerapi/module.go` | Factory | Builds a `require.ModuleLoader` | Keep, document |
| `require.ModuleLoader` | goja_nodejs `require/module.go` | Loader callback | Populates `module.exports` | Docs: CommonJS loader |
| `engine.RuntimeModuleSpec` | `engine/runtime_modules.go` | Registrar interface | Registers loaders into one runtime | Docs/rename: RuntimeModuleRegistrar |
| `engine.NativeModuleSpec` | `engine/module_specs.go` | Registrar DTO | Direct name+loader registration | Keep |
| `modules.NativeModule` | `modules/common.go` | Native module interface | Built-in/default module implementation | Keep, qualify |
| `providerapi.Registry` | `providerapi/registry.go` | Catalog | Provider metadata registry | Docs: provider registry |
| `modules.Registry` | `modules/common.go` | Catalog | Default native module registry | Qualify |
| `require.Registry` | goja_nodejs | Resolver/cache | Per-runtime CommonJS registry | Qualify |
| `app.RuntimeFactory` | `app/factory.go` | Adapter factory | Creates concrete runtime from profile | Docs/rename: RuntimeProfileFactory |
| `engine.FactoryBuilder` | `engine/factory.go` | Builder | Builds engine runtime factory plan | Keep |
| `engine.Factory` | `engine/factory.go` | Factory | Creates concrete engine runtimes | Qualify |
| `providerapi.RuntimeFactory` | `providerapi/commands.go` | Interface | Provider command runtime factory API | Docs/rename |
| `ModuleContext` | `providerapi/module.go` | Setup context bundle | Data passed to `Module.New` | Rename `Context` field |
| `RuntimeModuleContext` | `engine/runtime_modules.go` | Setup context bundle | Data passed to module registrars | Document phase |
| `RuntimeContext` | `engine/module_specs.go` | Init context bundle | Data passed to engine runtime initializers | Document phase |
| `SectionContext` | `providerapi/capabilities.go` | Request metadata | Why public sections are requested | Rename/docs |
| `CommandSetContext` | `providerapi/commands.go` | Factory context bundle | Data passed to command set providers | Document phase |
| `PackageCapability` | `providerapi/capabilities.go` | Marker interface | Optional provider package extension | Keep |
| `ConfigSectionCapability` | `providerapi/capabilities.go` | Capability | Public command parse sections | Rename high priority |
| `RuntimeInitializerCapability` | `providerapi/capabilities.go` | Capability | Post-runtime provider initializer | Rename/docs phase |
| `RuntimeHandle` | `providerapi/capabilities.go` | Narrow handle | Provider initializer handle to runtime | Rename/docs |
| `RuntimeCloserRegistry` | `providerapi/capabilities.go` | Optional handle extension | Add cleanup hooks | Keep |
| `providerapi.HostServices` | `providerapi/module.go` | Service interface | Generic host services for providers | Qualify |
| `app.HostServices` | `app/assets.go` | Concrete service bundle | Generated app service implementation | Docs: GeneratedHostServices |
| `runtimebridge.RuntimeServices` | `runtimebridge/runtimebridge.go` | Service bridge | VM-associated runtime services | Keep, document |
| `runtimeowner.RuntimeOwner` | `runtimeowner/types.go` | Scheduler/lifecycle interface | Owner-thread VM execution gateway | Keep, qualify |
| `runtimebridge.RuntimeOwner` | `runtimebridge/runtimebridge.go` | Narrow scheduler interface | `Call`/`Post` subset for bridge | Rename/docs: OwnerScheduler |
| `app.Host` | `app/host.go` | Wiring object | Generated app command/runtime wiring | Document |
| `app.Options` | `app/root.go` | Constructor options | New root command options | Rename/docs: RootCommandOptions |
| `HostOptions` | `app/host.go` | Constructor options | New host options | Keep |
| `ModuleRootsOptions` | `engine/module_roots.go` | Options | Script module root resolution | Keep |
| `runtimeowner.Options` | `runtimeowner/types.go` | Options | Owner scheduler options | Qualify |
| `geppetto.Options` | Geppetto module | Provider options | Geppetto native module behavior | Qualify |
| `geppetto HostServices` | Geppetto provider | Provider-specific service interface | Geppetto host contract | Rename/docs provider-specific |

---

## 14. Final recommendation

Do not try to unify all of these symbols into one grand framework. The system legitimately has multiple layers. The improvement should be **clearer names at layer boundaries**, not flattening.

Keep separate:

- provider registry vs require registry vs default native module registry;
- app runtime profile factory vs engine runtime factory;
- provider module factory vs CommonJS module loader;
- engine runtime module registration context vs provider module setup context;
- public command section capability vs internal module config section capability;
- runtime lifetime services vs host services.

Unify or rename:

- `app.Runtime` / `buildspec.Runtime` → `RuntimeProfile` in docs or code.
- `ConfigSectionCapability` → command/public-section capability.
- `ModuleContext.Context` → `StartupContext` / `SetupContext`.
- `RuntimeInitializerCapability` → post-runtime initializer capability.
- `app.RuntimeFactory` / `providerapi.RuntimeFactory` docs → runtime profile factory.

The biggest GOJA-053 design rule is: name APIs by **phase** and **surface**.

- Phase: build-time, startup/setup, pre-`Module.New`, module registration, post-runtime initialization, JS execution, runtime lifetime, shutdown.
- Surface: public command flags/config/env, internal module config, host services, runtime services, provider metadata, CommonJS module exports.
