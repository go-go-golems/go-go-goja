# Glossary

This file records the naming conventions used by `go-go-goja`, especially around `xgoja` code generation, provider registration, and runtime construction. Prefer these names for new public APIs and avoid adding compatibility aliases unless a ticket explicitly requires them.

## Core naming rule

Names should describe the object's role in the lifecycle:

- **Spec**: declarative data; says what should exist.
- **Builder**: mutable construction helper; accumulates settings before freezing.
- **Factory**: reusable object that creates runtime objects.
- **Registrar**: active object that registers something into another object.
- **Initializer**: active object that mutates an already-created runtime.
- **Context**: call-scoped inputs for one setup/registration/initialization operation.
- **Handle**: limited operation handle passed to callbacks.
- **Registry**: active lookup/registration collection.
- **Provider**: package-level API surface that contributes modules, commands, docs, verbs, or capabilities.
- **Source**: declarative origin of external or embedded content.
- **Store**: runtime lookup object that serves previously embedded or loaded content.

If a type performs I/O, registration, scheduling, lifecycle management, or mutation, do not name it `*Spec`.

## `*Spec` pattern

A `*Spec` type is a declarative data structure: it describes what should exist, but it should not itself perform runtime work.

Use `*Spec` for parsed or embedded configuration shapes such as build files, runtime profiles, selected modules, generated command settings, assets, and code-generation inputs. A `*Spec` may have small normalization helpers, but it should not own lifecycle, scheduling, I/O, registration side effects, or mutable runtime state.

Current precise names:

- `buildspec.BuildSpec`: parsed `xgoja.yaml` build model.
- `buildspec.ConfigFileSpec`: build-time generated-command config-file loading settings.
- `buildspec.GoSpec`: build-time Go module/version/replace settings.
- `buildspec.TargetSpec`: build-time output target settings.
- `buildspec.PackageSpec`: build-time provider package import/registration settings.
- `buildspec.RuntimeSpec`: declarative runtime profile in `xgoja.yaml`.
- `buildspec.ModuleInstanceSpec`: declarative selected provider module inside a runtime profile.
- `buildspec.CommandsSpec`: build-time generated command enablement settings.
- `buildspec.CommandSpec`: build-time settings for one built-in generated command.
- `buildspec.CommandProviderInstanceSpec`: declarative selected provider command set in `xgoja.yaml`.
- `buildspec.JSVerbSourceSpec`: build-time JavaScript verb source embedding settings.
- `buildspec.HelpSpec`: build-time help embedding settings.
- `buildspec.HelpSourceSpec`: one build-time help source.
- `buildspec.AssetSourceSpec`: one build-time static asset source.
- `app.RuntimeSpec`: normalized embedded runtime model decoded by generated xgoja binaries.
- `app.ConfigFileSpec`: normalized generated-command config-file loading settings.
- `app.TargetSpec`: normalized generated target settings.
- `app.PackageSpec`: normalized provider package settings.
- `app.RuntimeProfileSpec`: declarative runtime profile in the embedded runtime model.
- `app.ModuleInstanceSpec`: declarative selected provider module in the embedded runtime model.
- `app.CommandsSpec`: normalized generated command enablement settings.
- `app.CommandSpec`: normalized settings for one built-in generated command.
- `app.CommandProviderInstanceSpec`: declarative selected provider command set in the embedded runtime model.
- `app.JSVerbSourceSpec`: normalized JavaScript verb source settings.
- `app.HelpSpec`: normalized help embedding settings.
- `app.HelpSourceSpec`: one normalized help source.
- `app.AssetSourceSpec`: one normalized static asset source.

Rule of thumb: if the type says what to build, select, embed, or configure, `*Spec` is appropriate. If it creates, registers, runs, schedules, resolves, or closes something, it should not be named `*Spec`.

## Engine runtime construction pattern

The `engine` package is the reusable low-level runtime layer. Its names should describe runtime construction phases directly.

Current precise names:

- `engine.RuntimeFactoryBuilder`: mutable builder that composes require options, runtime module registrars, module middleware, and runtime initializers before freezing them.
- `engine.NewRuntimeFactoryBuilder(...)`: constructor for `RuntimeFactoryBuilder`.
- `engine.RuntimeFactory`: immutable runtime creation plan; creates concrete `engine.Runtime` instances.
- `engine.RuntimeFactory.NewRuntime(...)`: creates one owned runtime.
- `engine.Runtime`: concrete VM/event-loop/require runtime with lifecycle.
- `engine.RuntimeOption`: option for one `NewRuntime(...)` call, such as startup/lifetime context.
- `engine.Option`: option for the builder/factory configuration.

Avoid:

- `FactoryBuilder` for engine runtime factories.
- `Factory` for engine runtime factories.
- `NewBuilder` for public engine construction.

## Engine module registration pattern

Module registration is runtime-aware. It happens for a concrete runtime before `require` is enabled.

Current precise names:

- `engine.RuntimeModuleRegistrar`: interface for values that register one or more `require()` modules into a concrete runtime.
- `engine.RuntimeModuleRegistrationContext`: registration-phase context passed to `RuntimeModuleRegistrar.RegisterRuntimeModule`; exposes startup context, VM, event loop, owner, closer registration, and value bag before `require` is enabled.
- `engine.NativeModuleRegistrar`: concrete `RuntimeModuleRegistrar` that registers one native CommonJS module loader.
- `engine.ModuleMiddleware`: builder-time middleware that selects default-registry modules.
- `engine.ModuleSelector`: function used by module middleware to select module names.

Avoid:

- `RuntimeModuleSpec`: registration performs work and is not a declarative DTO.
- `RuntimeModuleContext`: too vague; the precise phase is module registration.
- `NativeModuleSpec`: native module registration performs work and is not a declarative DTO.

## Engine runtime initialization pattern

Runtime initialization happens after the VM and `require` are ready.

Current precise names:

- `engine.RuntimeInitializer`: interface for post-setup runtime initialization hooks.
- `engine.RuntimeInitializationContext`: initialization-phase context passed to `RuntimeInitializer.InitRuntime`; exposes startup context, VM, `require`, event loop, owner, and value bag.

Avoid:

- `RuntimeContext`: too vague; use the phase-specific context name.

## Provider registry pattern

The provider API is the registration layer for xgoja packages. A provider package registers entries into a provider registry.

Current precise names:

- `providerapi.ProviderRegistry`: active registry of provider packages, modules, command sets, help sources, verb sources, and package capabilities.
- `providerapi.NewProviderRegistry()`: constructor for `ProviderRegistry`.
- `providerapi.Entry`: interface implemented by values that can be applied to a `providerapi.Package`.
- `providerapi.Package`: internal registry record for one provider package.
- `providerapi.Module`: provider module definition.
- `providerapi.VerbSource`: provider-owned JavaScript verb source.
- `providerapi.HelpSource`: provider-owned Glazed help source.
- `providerapi.CommandSetProvider`: provider command-set definition.

Avoid:

- `providerapi.Registry`: too generic; collides with `require.Registry` and other registries.
- `providerapi.NewRegistry()`: too generic for public provider APIs.

## Provider module setup pattern

Module setup creates the CommonJS loader for one selected provider module instance.

Current precise names:

- `providerapi.Module.NewModuleFactory`: setup hook on `providerapi.Module`; returns a `require.ModuleLoader` for the selected module instance.
- `providerapi.ModuleSetupContext`: setup-time inputs passed while creating a selected module's CommonJS loader.
- `require.ModuleLoader`: CommonJS loader that populates `module.exports`.
- `providerapi.HostServices`: optional host services passed into provider module setup.
- `providerapi.AssetResolver`: host service for resolving embedded/static assets.

Avoid:

- `providerapi.Module.New`: too generic.
- `providerapi.ModuleContext`: too vague; the precise phase is module setup.
- exported `providerapi.ModuleFactory`: inline `func(providerapi.ModuleSetupContext) (require.ModuleLoader, error)`.

## Provider capability pattern

Capabilities are optional provider package extensions. They are package-scoped today and attached to selected module descriptors for every selected module from that package.

Current precise names:

- `providerapi.PackageCapability`: marker interface for package-level optional capabilities.
- `providerapi.ConfigSectionCapability`: current name for exposing Glazed sections that can be attached to commands. This is intentionally scheduled for a future rename when runtime module configuration is implemented.
- `providerapi.RuntimeInitializerCapability`: capability that initializes an already-created runtime from parsed Glazed values.
- `providerapi.RuntimeInitializerHandle`: limited handle passed to runtime initializer capabilities.
- `providerapi.RuntimeInitializerHandle.EngineRuntime()`: access to the owned `*engine.Runtime`.
- `providerapi.SectionRequest`: request metadata passed when collecting provider configuration sections.
- `providerapi.ModuleDescriptor`: app-facing description of a selected runtime module plus attached package capabilities.

Avoid:

- `RuntimeInitializerHandle.Runtime()`: ambiguous now that the handle exposes the engine runtime, not only a raw Goja VM.
- `RuntimeCloserRegistry`: cleanup registration is available through `handle.EngineRuntime().AddCloser(...)`.
- `SectionContext`: request metadata is not a runtime context.

Planned follow-up:

- Rename `ConfigSectionCapability` when implementing module-specific runtime flags, likely to a public-command/CLI-oriented name such as `CommandLineFlagsSectionCapability`, and introduce a separate internal module config capability if needed.

## Provider command-set pattern

Command-set providers let packages contribute Glazed commands to generated xgoja binaries.

Current precise names:

- `providerapi.CommandSetProvider`: provider command-set definition.
- `providerapi.CommandSetProvider.NewCommandSet`: hook that constructs a provider-owned command set.
- `providerapi.CommandSetContext`: inputs passed while constructing a command set.
- `providerapi.CommandSet`: bundle of Glazed commands plus optional parser config.
- `providerapi.RuntimeFactory`: xgoja-facing interface for creating runtimes from named runtime profiles; command providers use it when their commands need JavaScript execution.

Avoid:

- `CommandSetProvider.New`: too generic.
- exported `CommandSetProviderFactory`: inline `func(providerapi.CommandSetContext) (*providerapi.CommandSet, error)`.

## xgoja app runtime pattern

The `pkg/xgoja/app` package adapts embedded specs and provider registry entries into runtime-profile execution.

Current precise names:

- `app.RuntimeFactory`: xgoja runtime-profile factory that creates concrete runtimes from named runtime profiles.
- `app.NewRuntimeFactory(...)`: constructor for `app.RuntimeFactory`.
- `app.JSRuntime`: alias for `engine.Runtime`.
- `app.Host`: generated-binary host that owns providers, runtime spec, assets, help, and command mounting.
- `app.HostOptions`: options for constructing `Host`.
- `app.Options`: options for constructing the generated root command.
- `app.AssetStore`: runtime lookup store for embedded asset sources.
- `app.HostServices`: app implementation of provider host services.

Important distinction:

- `engine.RuntimeFactory` creates low-level engine runtimes from an immutable engine plan.
- `app.RuntimeFactory` creates xgoja runtimes by selecting a named `app.RuntimeProfileSpec`, adapting provider modules into engine registrars, and applying xgoja runtime options.

## Source and store pattern

Use `*Source` for a declared origin of content and `*Store` for runtime lookup/service objects.

Current precise names:

- `providerapi.VerbSource`: provider-declared JavaScript verb source.
- `providerapi.HelpSource`: provider-declared help source.
- `buildspec.JSVerbSourceSpec` / `app.JSVerbSourceSpec`: declarative verb source configuration.
- `buildspec.HelpSourceSpec` / `app.HelpSourceSpec`: declarative help source configuration.
- `buildspec.AssetSourceSpec` / `app.AssetSourceSpec`: declarative static asset source configuration.
- `app.AssetStore`: runtime asset lookup object.

## Request, context, options, and settings pattern

Use these suffixes consistently:

- `*Request`: metadata describing why information is being requested; should not imply lifecycle ownership. Example: `providerapi.SectionRequest`.
- `*Context`: call-scoped operational inputs for setup, registration, initialization, or command construction. Examples: `providerapi.ModuleSetupContext`, `providerapi.CommandSetContext`, `engine.RuntimeModuleRegistrationContext`, `engine.RuntimeInitializationContext`.
- `*Options`: optional constructor or call options, usually explicit API input. Examples: `app.HostOptions`, `app.Options`, `engine.RuntimeOption`.
- `*Settings`: decoded command/user values, often internal to a command implementation. Examples: app command settings structs such as eval/run/TUI settings.

## Current external YAML/JSON field names

Some fields are external schema names and should not be renamed just because a Go DTO was renamed.

Current important names:

- `CommandSpec.Runtime`: external generated-command runtime profile selector. Do not rename this field to `RuntimeSpec`.
- Provider/module instance `config`: internal provider/module config payload. Keep this serialized field as `config`.
- Top-level generated-command config-file settings: `configFile`.

## Quick replacement table

| Old | Current precise name |
| --- | --- |
| `engine.NewBuilder(...)` | `engine.NewRuntimeFactoryBuilder(...)` |
| `engine.FactoryBuilder` | `engine.RuntimeFactoryBuilder` |
| `engine.Factory` | `engine.RuntimeFactory` |
| `engine.RuntimeModuleSpec` | `engine.RuntimeModuleRegistrar` |
| `engine.RuntimeModuleContext` | `engine.RuntimeModuleRegistrationContext` |
| `engine.RuntimeContext` | `engine.RuntimeInitializationContext` |
| `engine.NativeModuleSpec` | `engine.NativeModuleRegistrar` |
| `providerapi.Registry` | `providerapi.ProviderRegistry` |
| `providerapi.NewRegistry()` | `providerapi.NewProviderRegistry()` |
| `providerapi.SectionContext` | `providerapi.SectionRequest` |
| `providerapi.ModuleContext` | `providerapi.ModuleSetupContext` |
| `providerapi.Module.New` | `providerapi.Module.NewModuleFactory` |
| `providerapi.RuntimeHandle` | `providerapi.RuntimeInitializerHandle` |
| `RuntimeInitializerHandle.Runtime()` | `RuntimeInitializerHandle.EngineRuntime()` |
| `providerapi.CommandSetProvider.New` | `providerapi.CommandSetProvider.NewCommandSet` |
| `buildspec.Spec` | `buildspec.BuildSpec` |
| `app.Spec` | `app.RuntimeSpec` |
| `app.Runtime` | `app.RuntimeSpec` |
