---
Title: 'Review and Design: Glazed Section Values as Pre-Runtime xgoja Module Config'
Ticket: GOJA-053
Status: active
Topics:
    - xgoja
    - provider
    - capability
    - glazed
    - config
    - architecture
    - review
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/js/modules/geppetto/provider/provider.go
      Note: |-
        Motivating provider whose config must be available during Module.New.
        Motivating config-time Geppetto provider implementation
    - Path: glazed/pkg/cmds/values/section-values.go
      Note: Glazed Values API and field lookup behavior
    - Path: go-go-goja/pkg/xgoja/app/factory.go
      Note: |-
        Runtime construction choke point where ModuleInstance.Config becomes ModuleContext.Config.
        Runtime construction choke point for pre-Module.New config merging
    - Path: go-go-goja/pkg/xgoja/providerapi/capabilities.go
      Note: |-
        Existing capability contracts and ModuleDescriptor shape.
        Capability contracts reviewed and target API location
    - Path: go-go-goja/pkg/xgoja/providerapi/commands.go
      Note: RuntimeFactory interface exposed to provider-owned command sets.
    - Path: go-go-goja/pkg/xgoja/providerutil/sections.go
      Note: |-
        Shared section collection and runtime-initializer traversal patterns.
        Existing capability traversal pattern and proposed patch helper location
    - Path: pinocchio/cmd/pinocchio/cmds/js.go
      Note: Pinocchio JS command shows turns-dsn/turns-db settings pattern
    - Path: pinocchio/cmd/pinocchio/doc/general/05-js-runner-scripts.md
      Note: Pinocchio documentation for turn-store flags used to simplify Geppetto design
ExternalSources:
    - https://github.com/go-go-golems/go-go-goja/issues/52
Summary: Critical review of the existing GOJA-053 design docs plus an intern-ready design and implementation guide for passing parsed Glazed section values into xgoja module config before Module.New.
LastUpdated: 2026-06-03T00:00:00Z
WhatFor: Use when implementing or reviewing ModuleConfigCapability, NewRuntimeFromSections, Geppetto config flags, or future xgoja plugin/codegen extension points.
WhenToUse: Before changing xgoja provider capabilities, runtime factory APIs, provider-owned command runtime creation, or Glazed config-to-module-config mapping.
---



# Review and Design: Glazed Section Values as Pre-Runtime xgoja Module Config

## Executive summary

The existing GOJA-053 design work correctly identifies the central bug: xgoja can add provider-owned Glazed sections to generated commands, and it can run provider-owned runtime initializers, but parsed command/config/env values arrive too late to influence `providerapi.Module.New`. The runtime factory currently serializes only `ModuleInstance.Config` from `xgoja.yaml` into `providerapi.ModuleContext.Config`; the parsed `*values.Values` object is only used after the runtime has already been built.

The prior design is useful and close to implementable, but it has several review-critical gaps:

- It deduplicates `ModuleConfigCapability` by package/capability, which is correct for section collection and runtime initializers but wrong for per-module config patches. A runtime can contain the same package more than once under different aliases, and each selected module instance needs a chance to receive its own patch.
- It under-specifies zero/default handling. Glazed values include defaults, config-file values, env values, args, and Cobra flag values; a simple `DecodeSectionInto` cannot tell whether `false` means “user explicitly disabled this” or “default false was applied.” The implementation must inspect `fields.FieldValue.Log` sources or provide a helper that does so.
- It alternates between `map[string]any`, `json.RawMessage`, and typed structs without choosing one stable public contract. The revised recommendation is a `ModuleConfigPatch` wrapper: JSON config keys plus optional Glazed provenance.
- It proposes extending `providerapi.RuntimeFactory` directly. That may be acceptable inside this worktree, but it is source-breaking for any external command provider implementation. A safer first release is to add an optional extended interface plus a helper, then promote it when the ecosystem is ready.
- Its plugin/code-generation exploration is directionally useful but not anchored tightly enough to the immediate capability. The right influence from future plugin/codegen targets is: make the new hook pure, typed, serializable, and phase-explicit, but do not introduce a plugin manager or codegen IR to solve this ticket.

My recommended solution is a **pre-runtime module config capability** plus a **runtime factory method that accepts parsed Glazed values**. Keep the generated-code surface unchanged. Add the hook in the app/runtime phase, where the runtime profile has already selected concrete module instances and command parsing has produced `values.Values`.

The minimum implementation sequence is:

1. Add `providerapi.ModuleConfigCapability`.
2. Add `providerutil.ModuleConfigPatchesFromSections`, called once per selected module descriptor, not deduped globally.
3. Add `RuntimeFactory.NewRuntimeFromSections(ctx, profile, vals, opts...)` in `pkg/xgoja/app/factory.go`.
4. Update built-in `eval`, `run`, TUI `repl`, and `jsverbs` paths to call the new factory before running existing runtime initializers.
5. Add a helper for typed patch construction that only emits fields whose last non-default source is `cobra`, `arguments`, `env`, or `config`.
6. Simplify Geppetto config to profile/default-profile plus explicit turn-store flags such as `--turns-dsn` / `--turns-db`; remove the extra `allowRegistryLoad`, `allowNetwork`, `allowTools`, `enableStorage`, and nested `turns` gates from the proposed Geppetto config surface.
7. Add tests for per-alias behavior, zero/default behavior, config precedence, no spec mutation, built-in commands, and provider-owned command sets.

### Follow-up revision from review discussion

After follow-up review, the design should be simplified in two important ways:

1. **Geppetto should not carry extra allow-gate config in this capability.** The xgoja Geppetto module config should not grow `allowRegistryLoad`, `allowNetwork`, `allowTools`, `enableStorage`, or nested `turns`. Turn storage should look like the Pinocchio/Geppetto command-line tools: explicit flags such as `--turns-dsn` and `--turns-db` opt into a turn store. The absence of these flags means no storage.
2. **The return type should be reconsidered.** Raw `values.Values` is not the right return type because it is section/flag-name-oriented, not module-config-key-oriented. But the instinct is correct: we should preserve value provenance. A better public return type is a small `ModuleConfigPatch` wrapper whose keys are module JSON config keys and whose values carry both `Value any` and the original Glazed `FieldValue` log/provenance when available.

---

## 1. Problem statement and vocabulary

### 1.1 What problem are we solving?

An xgoja binary is generated from a spec and provider packages. Each provider package can register Go-backed JavaScript modules and optional capabilities. A generated command such as `eval`, `run`, `repl`, or `verbs ...` can parse Glazed flags into `*values.Values`.

The missing path is:

```text
Glazed command/config/env values
  → provider-owned section values
  → module config patch
  → merged ModuleContext.Config
  → providerapi.Module.New(...)
```

Today the actual path is:

```text
xgoja.yaml ModuleInstance.Config
  → json.Marshal(instance.Config)
  → providerapi.ModuleContext.Config
  → providerapi.Module.New(...)

Glazed values.Values
  → RuntimeInitializerCapability.InitRuntimeFromSections(...)
  → too late for Module.New
```

This matters for Geppetto. The Geppetto provider decodes config and constructs module options inside `Module.New` (`geppetto/pkg/js/modules/geppetto/provider/provider.go:82-101`). Its profile registries and default profile are config-time inputs (`provider.go:17-25`, `provider.go:154-180`), not post-runtime side effects. The revised design keeps the Geppetto surface narrow: profile registry/default-profile settings plus explicit turn-store flags such as `turns-dsn` / `turns-db`; it does not keep the earlier allow-gate booleans or nested `turns` config object.

### 1.2 Key terms

- **Provider package**: A Go package that registers xgoja modules, capabilities, verb sources, help sources, and command set providers into `providerapi.Registry`.
- **Module**: A `require()`-loadable JS module backed by a `providerapi.ModuleFactory`.
- **Runtime profile**: A named list of module instances from the xgoja spec.
- **Module instance**: One entry in `runtimes.<profile>.modules[]`. It has a package, module name, optional alias, and static config.
- **Capability**: Optional provider behavior registered at package scope.
- **Glazed section**: A named group of command fields/flags parsed into `values.Values`.
- **Pre-runtime config hook**: The new phase proposed here; it runs after command parsing and before `Module.New`.
- **Runtime initializer**: Existing post-runtime phase; it runs after the JS runtime and provider modules have been created.

---

## 2. Current system map with file evidence

### 2.1 Provider capability contracts

`providerapi/capabilities.go` currently defines the capability model:

- `SectionContext` carries command/runtime/module metadata (`capabilities.go:13-23`).
- `ModuleDescriptor` describes a selected module instance and carries package capabilities (`capabilities.go:25-33`).
- `PackageCapability` is the marker interface (`capabilities.go:35-38`).
- `ConfigSectionCapability` declares Glazed sections (`capabilities.go:40-45`).
- `RuntimeInitializerCapability` mutates or initializes an already-created runtime (`capabilities.go:60-66`).
- `WithPackageCapability` documents that capabilities are package-scoped and attached to every selected module from that package (`capabilities.go:72-76`).

This gives us the right extension point family. We should add a sibling phase rather than overloading runtime initializers.

### 2.2 Module config contract

`providerapi/module.go` defines the `ModuleFactory` and `ModuleContext`:

```go
type ModuleFactory func(ModuleContext) (require.ModuleLoader, error)

type ModuleContext struct {
    Context      context.Context
    Name         string
    As           string
    Config       json.RawMessage
    Host         HostServices
    RuntimeOwner runtimeowner.RuntimeOwner
}
```

The important field is `Config json.RawMessage` (`module.go:14-21`). Providers already understand this config as JSON. Geppetto’s provider decodes it with `decodeConfig(ctx.Config)` before constructing `geppettomodule.Options` (`provider.go:82-101`). Therefore the pre-runtime hook should produce a JSON-shaped patch that can be merged into this same config path.

### 2.3 Runtime construction choke point

`pkg/xgoja/app/factory.go` is the single place where spec config becomes module factory config:

```go
config, err := json.Marshal(s.instance.Config)
loader, err := s.module.New(providerapi.ModuleContext{
    Context:      ctx.Context,
    Name:         s.instance.Name,
    As:           s.instance.Alias(),
    Config:       config,
    Host:         s.services,
    RuntimeOwner: ctx.Owner,
})
```

This is at `factory.go:31-50`. The static `ModuleInstance.Config` is defined as `map[string]any` in `app/spec.go:37-42`. The factory’s `NewRuntime` loops over `runtime.Modules`, resolves provider modules, wraps them as `providerRuntimeModuleSpec`, and delegates to the lower-level engine builder (`factory.go:62-90`).

This file should get the new `NewRuntimeFromSections` method. The lower-level engine does not need to know about Glazed sections.

### 2.4 Section collection and runtime initializer traversal

`pkg/xgoja/providerutil/sections.go` is the existing helper for capability traversal:

- `CollectConfigSections` iterates selected module descriptors, type-asserts `ConfigSectionCapability`, enriches `SectionContext`, deduplicates by package/capability, and rejects duplicate section slugs (`sections.go:13-49`).
- `InitRuntimeFromSections` does similar traversal for `RuntimeInitializerCapability`, with a runtime handle and the parsed values (`sections.go:74-99`).

These helpers are the model for the new helper, with one crucial difference: config patching must be **per descriptor**, not globally deduplicated.

### 2.5 Built-in command flow

The built-in commands already collect module sections during construction and parse values during execution:

- `eval` collects sections for the default profile (`root.go:70-96`), decodes settings, resolves selected modules, then calls `factory.NewRuntime` and only afterwards `initRuntimeFromSections` (`root.go:105-136`).
- `run` follows the same pattern, adding script module roots before `factory.NewRuntime` (`run.go:72-118`).
- TUI `repl` creates a long-lived runtime in `newXGojaTUIEvaluator`, then initializes it from sections (`tui.go:151-161`).
- `jsverbs` creates a runtime inside the verb invoker with an extra JS verb loader, then initializes from sections (`root.go:251-262`).

All four call sites must switch to `NewRuntimeFromSections` while preserving their existing `require.Option` arguments and post-runtime initializer pass.

### 2.6 Provider-owned commands

Provider-owned commands receive `providerapi.CommandSetContext`, which includes:

```go
RuntimeFactory  RuntimeFactory
SelectedModules []ModuleDescriptor
```

This is defined in `providerapi/commands.go:19-39` and populated by `Host.newCommandSet` in `app/command_providers.go:59-79`. The xgoja docs already instruct provider commands to collect selected module sections and then call `RuntimeFactory.NewRuntime` followed by `providerutil.InitRuntimeFromSections` (`cmd/xgoja/doc/05-tutorial-providing-commands.md:64-100`).

The new design must not forget provider-owned commands. They are not built-in commands, but they are a first-class expansion point.

### 2.7 Glazed values and source logs

`glazed/pkg/cmds/values/section-values.go` shows that `values.Values` is a map of section slug to `SectionValues` (`section-values.go:155-177`) and that `DecodeSectionInto` decodes a section into a struct (`section-values.go:246-260`). `GetField(slug, key)` returns a `*fields.FieldValue` (`section-values.go:280-285`).

`glazed/pkg/cmds/fields/field-value.go` stores `FieldValue.Log []ParseStep` (`field-value.go:11-17`). `ParseStep` includes `Source` (`parse.go:21-27`). Glazed’s Cobra parser marks sources as `cobra`, `arguments`, `env`, `config`, and `defaults` (`cobra-parser.go:148-183`).

This is essential. A safe config patch helper should not blindly emit decoded zero values. It should inspect field logs and include only values whose effective source is not `defaults`, unless the provider explicitly asks to include defaults.

---

## 3. Review of the existing GOJA-053 design documents

### 3.1 What is good

The first design document is a strong starting point.

- It correctly explains the high-level gap: parsed values are available only after runtime creation (`design/01-module-config-capability.md:450-459`).
- It identifies the correct insertion point: merge config before `Module.New` (`design/01-module-config-capability.md:481-484`).
- It finds the important files: `providerapi/capabilities.go`, `app/factory.go`, `providerutil/sections.go`, built-in command files, and Geppetto provider code.
- It distinguishes `ModuleConfigCapability` from `RuntimeInitializerCapability`, which keeps lifecycle phases conceptually clean (`design/01-module-config-capability.md:463-517`).
- It recognizes that package-scoped capabilities are surprising and that `ModuleDescriptor` is needed for per-module decisions (`design/01-module-config-capability.md:523-530`).
- It includes a practical implementation plan and a useful test list (`design/01-module-config-capability.md:820-1158`).
- It raises the right type-safety concern about `map[string]any` patches (`design/01-module-config-capability.md:703-814`).

The second architecture document is useful as context for future evolution.

- It separates build-time code generation from runtime execution (`design/02-xgoja-architecture-and-extensibility.md:23-67`).
- It notices that xgoja is currently statically extensible through Go provider packages and could later grow plugin or alternative codegen targets (`design/02-xgoja-architecture-and-extensibility.md:300-310`).
- It encourages a spec-to-model-to-renderer mental model, which is helpful for future codegen targets.

### 3.2 What is not so good

#### 3.2.1 The proposed patch collector deduplicates the wrong phase

The design proposes `ModuleConfigPatchFromSections` that creates `applied := map[string]struct{}{}` outside the descriptor loop and deduplicates by `(packageID, capabilityID)` (`design/01-module-config-capability.md:852-869`). That is wrong for config patching.

Deduplication is correct for `CollectConfigSections`: a package’s section should appear once in one command. Deduplication is also correct for runtime initializers: a package-level initializer should not start the same HTTP server twice for one runtime. But a config patch is attached to a specific selected module instance. If a runtime contains two `geppetto` instances with different aliases, the provider capability must run for each descriptor.

The collector should call every `ModuleConfigCapability` for every selected descriptor, and then merge the returned patch into that descriptor’s config. The capability can decide to return nil for non-target modules.

#### 3.2.2 The zero-value/default problem is named but not solved

The design notes that zero-value patching can override YAML config accidentally (`design/01-module-config-capability.md:1169-1178` in the source document), and it sketches a `cleanPatch(patch, dst)` call (`design/01-module-config-capability.md:1102-1104`). But it never specifies how `cleanPatch` can distinguish defaults from explicit values.

Glazed already records parse-source history in `FieldValue.Log`. The implementation should use that instead of trying to infer intent from Go zero values. This is especially important for booleans:

```yaml
# xgoja.yaml for some provider with an enable/disable boolean
config:
  enabled: true
```

If a Glazed field has default `false`, decoding into `Enabled bool` and blindly marshaling the struct can overwrite `enabled: true` with `false` even though the user did not pass a flag.

#### 3.2.3 It underplays command-provider compatibility

The design recommends adding `NewRuntimeFromSections` directly to `providerapi.RuntimeFactory` (`design/01-module-config-capability.md:1017-1026`). That is clean for app internals, but it is source-breaking for external implementations. The current interface has one method (`commands.go:22-24`). Any adapter or test double implementing it must change if the method is added.

A better first step is:

```go
type RuntimeFactoryWithSections interface {
    RuntimeFactory
    NewRuntimeFromSections(ctx context.Context, profile string, vals *values.Values, opts ...require.Option) (*engine.Runtime, error)
}
```

Then provide `providerutil.NewRuntimeFromSections(...)` that type-asserts the extended interface. Built-in app code can call the concrete method directly. Provider-owned commands can use the helper and get a useful error if their runtime factory cannot honor module config capabilities.

#### 3.2.4 It treats `json.RawMessage` as cleaner than it really is

The design suggests returning `json.RawMessage` as a better API (`design/01-module-config-capability.md:647-666`, `728-782`). That is a good instinct for provider-local typing, but the app still must merge JSON objects. If the public hook returns bytes, app code must unmarshal them, validate that the patch is an object, deep-merge it, and marshal again. Raw `values.Values` has the opposite problem: it preserves provenance but remains in section/field-key space instead of module-config-key space.

The better solution is a middle path: typed provider helpers that produce a `ModuleConfigPatch` with JSON config keys, JSON-compatible values, and copied Glazed provenance for fields that came from `values.Values`.

#### 3.2.5 The Geppetto sketch carried too much config surface

The prior sketch tried to account for the current Geppetto provider config shape, including `allowRegistryLoad`, `allowNetwork`, `allowTools`, `enableStorage`, and nested `turns` (`geppetto/provider/provider.go:17-34`). Follow-up review makes that unnecessary and undesirable for this capability. The better direction is to simplify the Geppetto provider config itself: keep profile registry/default-profile inputs, and expose turn storage with concrete command-style flags such as `--turns-dsn` and `--turns-db`, matching Pinocchio's JS runner documentation (`pinocchio/cmd/pinocchio/doc/general/05-js-runner-scripts.md:60-148`) and command settings (`pinocchio/cmd/pinocchio/cmds/js.go:53-104`).

That simplification removes most boolean-clearing complexity from the Geppetto case. The generic capability still needs source-aware default handling because other providers may legitimately have booleans, but Geppetto should not use this feature as a dumping ground for broad safety gates.

#### 3.2.6 The plugin/codegen exploration is too detached from the immediate API

The architecture document discusses subprocess plugins, WASM plugins, JS config, and alternate targets (`design/02-xgoja-architecture-and-extensibility.md:300-310` and later sections). These are plausible future directions, but the document does not explain how the current `ModuleConfigCapability` design should change because of them.

The actionable influence is:

- Keep the hook input/output serializable so it can cross process/WASM boundaries later.
- Keep lifecycle phases explicit so a plugin protocol can advertise `configSections`, `moduleConfigPatch`, and `runtimeInitializer` separately.
- Avoid requiring code generation changes for this feature; that keeps future library/adapter targets able to reuse the same runtime API.

### 3.3 What they missed

1. **Per-alias/per-instance semantics.** Same package, same module, different aliases must be handled.
2. **Effective-source detection.** Defaults must not accidentally override static config.
3. **Config clearing semantics.** Can a CLI/config value clear a YAML list? Can it set a boolean to false? The design needs a policy.
4. **Nil and missing section behavior.** `DecodeSectionInto` returns an error when a non-default section is absent (`glazed/values/section-values.go:256-260`). The capability/helper should treat absent sections as no patch when appropriate.
5. **Command-provider migration.** Provider-owned commands need a safe pattern, not just built-in command changes.
6. **Documentation updates.** `cmd/xgoja/doc/04-tutorial-providing-package-and-modules.md` and `cmd/xgoja/doc/05-tutorial-providing-commands.md` need updates, otherwise new providers will keep using only runtime initializers.
7. **Scope discipline for provider config.** Geppetto should avoid broad allow-gate booleans in this hook. Runtime flags should map to concrete user intent, such as profile selection or turn-store DSNs, rather than weakening static policy implicitly.
8. **Compile-checkable pseudocode.** Several snippets are good conceptually but would not compile as written or omit required imports/helpers. Intern-facing implementation guides need code that is closer to pasteable.
9. **Doc quality issues in architecture doc.** It contains syntax/formatting errors such as an unterminated bold phrase at `design/02...md:53`, malformed code snippets (`design/02...md:88`), duplicate heading numbering (`3.2` twice), and typo `proposeded` (`design/02...md:224`).

### 3.4 Resources they should have read more closely

- `pkg/xgoja/providerapi/capabilities.go` — especially the package-scoped capability comment and `SectionContext` shape.
- `pkg/xgoja/providerutil/sections.go` — to understand where deduplication is appropriate and where it is not.
- `pkg/xgoja/app/factory.go` — the concrete config serialization point.
- `pkg/xgoja/app/root.go`, `run.go`, `tui.go` — all built-in runtime creation paths.
- `pkg/xgoja/app/command_providers.go` and `cmd/xgoja/doc/05-tutorial-providing-commands.md` — provider-owned command path.
- `glazed/pkg/cmds/values/section-values.go` and `glazed/pkg/cmds/fields/field-value.go` — especially `GetField` and `FieldValue.Log`.
- `glazed/pkg/cli/cobra-parser.go` — source precedence and source labels.
- `geppetto/pkg/js/modules/geppetto/provider/provider.go` and `provider_test.go` — actual config schema, validation gates, and failure modes.
- `cmd/xgoja/doc/04-tutorial-providing-package-and-modules.md` — current provider author guidance that will need to be amended.
- The GOJA-053 architecture exploration — not as something to implement now, but as a constraint to keep the hook pure and serializable.

### 3.5 What they should do better next time

- Separate “observed code behavior” from “proposed future behavior.”
- Use line references for important claims, especially when reviewing another design.
- Treat command providers as first-class; they are not an edge case.
- When a design mentions a helper (`cleanPatch`, `deepMerge`, `PatchFromSection`), specify its exact contract and test cases.
- For booleans and defaults, always ask: “How do we know the user explicitly set this?”
- Make pseudocode preserve invariants from the current code: require options, runtime owner/lifetime contexts, host services, aliases, and nil behavior.
- If discussing plugins or alternate codegen targets, tie each future direction back to concrete API constraints in the current ticket.

---

## 4. Proposed design

### 4.1 Design goals

The solution should satisfy these goals:

1. **Correct phase:** Run after Glazed values exist and before `Module.New`.
2. **Provider-local typing:** Let providers decode their own section values and produce JSON-keyed config patches.
3. **Per-instance behavior:** Apply patches to selected module instances, not globally to packages.
4. **No spec mutation:** Each runtime creation gets a cloned config map.
5. **Safe defaults:** Do not let Glazed defaults overwrite static config unless explicitly requested.
6. **Backward compatibility:** Existing providers and callers of `NewRuntime` keep working.
7. **Command-provider support:** Provider-owned commands can opt into the same pre-runtime path.
8. **Future extensibility:** The hook is pure and serializable enough for future plugin/codegen targets.

### 4.2 Lifecycle diagram

```text
Command construction phase
──────────────────────────
selected runtime profile
  → selected module descriptors
  → ConfigSectionCapability.ConfigSections(SectionContext)
  → Glazed command schema with provider flags

Command execution phase
───────────────────────
Cobra/Glazed parse args/env/config/defaults
  → *values.Values
  → RuntimeFactory.NewRuntimeFromSections(ctx, profile, vals, opts...)
      → selected module descriptors
      → for each descriptor:
          ModuleConfigCapability.ModuleConfigFromSections(...)
          clone static ModuleInstance.Config
          merge patch into clone
      → engine builder
      → providerRuntimeModuleSpec.RegisterRuntimeModule(...)
      → Module.New(ModuleContext{Config: mergedConfig})
  → RuntimeInitializerCapability.InitRuntimeFromSections(...)
  → eval/run/repl/jsverb/provider command body
```

### 4.3 Public capability interface

A stronger long-lived API uses a request struct and a config-patch wrapper:

```go
type ModuleConfigRequest struct {
    SectionContext SectionContext
    Descriptor     ModuleDescriptor
}

type ModuleConfigPatch struct {
    Fields map[string]ModuleConfigValue // JSON config key → value + provenance
}

type ModuleConfigValue struct {
    Value any
    Log   []fields.ParseStep // optional Glazed provenance, when value came from a field
}

type ModuleConfigCapability interface {
    PackageCapability

    ModuleConfigFromSections(
        ctx context.Context,
        vals *values.Values,
        req ModuleConfigRequest,
    ) (*ModuleConfigPatch, error)
}
```

Why this is better than returning raw `values.Values`:

- `values.Values` is organized by Glazed section slug and Glazed field names, such as `geppetto.profile-registries`; module config needs JSON config keys such as `profileRegistries`.
- `values.Values` carries whole command state, including unrelated sections. The return value should be scoped to one module instance.
- `values.Values` carries useful history, but that history can be copied into `ModuleConfigValue.Log` while still returning a module-config-shaped patch.
- The app/factory layer still needs a JSON-compatible object to merge into `ModuleInstance.Config`; a patch wrapper can expose `JSONMap()` for that final merge.

Generics are useful for provider-local helpers, but not for the registry-facing capability interface. Provider packages are heterogeneous, and the registry stores capabilities behind `PackageCapability`; a generic `ModuleConfigCapability[T]` would not type-assert cleanly across providers. Good generic helpers are still possible:

```go
func PatchFromSection[T any](vals *values.Values, section string, mapper FieldMapper[T]) (*ModuleConfigPatch, error)
```

The minimal issue API returning `map[string]any` still works, but if implementing from scratch now, prefer `*ModuleConfigPatch` so provenance is not lost.

### 4.4 What `SectionContext` and `ModuleDescriptor` mean

`SectionContext` answers “why are we asking for provider fields?” It is command metadata: built-in command name, provider-owned command ID, runtime profile, package ID, and module ID. It lets a provider tailor section definitions or config patching to the invocation context without guessing from global state.

`ModuleDescriptor` answers “which concrete module instance are we configuring?” It contains the selected package ID, module ID, alias (`As`), resolved provider module, and package capabilities. In the pre-runtime config phase, this is the unit of scoping. A runtime profile can select the same provider package multiple times under different aliases, so the config patch is not “for Geppetto globally”; it is for this selected `geppetto` module instance.

Conceptually, `ModuleConfigFromSections` merges two value spaces:

```text
static module values from xgoja.yaml
    +
external command/config/env values from Glazed
    ↓ scoped by runtime profile + module descriptor
merged module config for this Module.New call
```

So yes: it is “merging value with values,” but the tricky part is that the two sides live in different namespaces. Static module config uses JSON config keys (`profileRegistries`, `turnsDSN`), while Glazed uses section/field keys (`geppetto.profile-registries`, `geppetto.turns-dsn`) plus source history. The capability is the adapter that maps external section values into the module config namespace.

### 4.5 Patch semantics

A module config patch is a JSON-object-shaped patch with optional provenance:

```go
&providerapi.ModuleConfigPatch{Fields: map[string]providerapi.ModuleConfigValue{
    "profileRegistries": {Value: []string{"./profiles.yaml"}, Log: profileRegistriesLog},
    "defaultProfile":    {Value: "assistant", Log: defaultProfileLog},
    "turnsDSN":          {Value: "sqlite:///tmp/turns.db", Log: turnsDSNLog},
}}
```

Rules:

- Keys are **JSON config keys**, not Glazed field names.
- `nil` or an empty patch means “no patch.”
- The merge layer converts the patch to a plain JSON-compatible map for merging.
- Patches are merged into a clone of the static `ModuleInstance.Config`.
- Object values are recursively merged.
- Scalars and arrays replace the old value.
- The implementation must not mutate the original spec config map or nested maps.
- A capability is called for each selected descriptor. It may return nil for descriptors it does not own or should not patch.

### 4.6 Default and explicit-value semantics

By default, a helper should emit only values whose effective source is not `defaults`.

Effective source can be computed from `FieldValue.Log`:

```go
func EffectiveSource(fv *fields.FieldValue) string {
    if fv == nil || len(fv.Log) == 0 {
        return ""
    }
    return fv.Log[len(fv.Log)-1].Source
}

func WasProvided(fv *fields.FieldValue) bool {
    switch EffectiveSource(fv) {
    case "cobra", "arguments", "env", "config":
        return true
    default:
        return false
    }
}
```

This lets a provider-specific `--enabled=false` or config-file `enabled: false` explicitly clear a YAML `enabled: true`, while a schema default `false` does not. For Geppetto specifically, the revised design avoids the earlier allow-gate booleans and mostly uses string/list fields such as profile registries and turn-store DSNs.

### 4.7 Provider helper API

Add a helper package surface in `providerutil` so provider authors do not hand-write source checks and maps:

```go
type PatchBuilder struct {
    vals    *values.Values
    section string
    patch   *providerapi.ModuleConfigPatch
}

func NewPatchBuilder(vals *values.Values, section string) *PatchBuilder

func (b *PatchBuilder) SetIfProvided(fieldName string, jsonKey string) error
func (b *PatchBuilder) SetStringIfProvided(fieldName string, jsonKey string) error
func (b *PatchBuilder) SetStringListIfProvided(fieldName string, jsonKey string) error
func (b *PatchBuilder) SetBoolIfProvided(fieldName string, jsonKey string) error
func (b *PatchBuilder) Patch() *providerapi.ModuleConfigPatch
```

Pseudocode:

```go
func (b *PatchBuilder) SetIfProvided(fieldName, jsonKey string) error {
    if b == nil || b.vals == nil {
        return nil
    }
    fv, ok := b.vals.GetField(b.section, fieldName)
    if !ok || !WasProvided(fv) {
        return nil
    }
    v, err := fv.GetInterfaceValue()
    if err != nil {
        return fmt.Errorf("read %s.%s: %w", b.section, fieldName, err)
    }
    b.patch.Fields[jsonKey] = providerapi.ModuleConfigValue{Value: v, Log: append([]fields.ParseStep(nil), fv.Log...)}
    return nil
}
```

This helper is intentionally boring. It keeps provider code explicit, preserves source semantics, and avoids reflection-heavy “magic” while still preventing typo-prone map assembly.

### 4.8 Runtime factory API

Add a concrete method to `app.RuntimeFactory`:

```go
func (f *RuntimeFactory) NewRuntimeFromSections(
    ctx context.Context,
    profile string,
    vals *values.Values,
    opts ...require.Option,
) (*JSRuntime, error)
```

Keep the current method as a wrapper:

```go
func (f *RuntimeFactory) NewRuntime(ctx context.Context, profile string, opts ...require.Option) (*JSRuntime, error) {
    return f.NewRuntimeFromSections(ctx, profile, nil, opts...)
}
```

For provider-owned commands, add an optional interface:

```go
type RuntimeFactoryWithSections interface {
    RuntimeFactory
    NewRuntimeFromSections(ctx context.Context, profile string, vals *values.Values, opts ...require.Option) (*engine.Runtime, error)
}
```

And a helper:

```go
func NewRuntimeFromSections(
    ctx context.Context,
    factory providerapi.RuntimeFactory,
    profile string,
    vals *values.Values,
    opts ...require.Option,
) (*engine.Runtime, error) {
    if factory == nil {
        return nil, fmt.Errorf("runtime factory is nil")
    }
    if withSections, ok := factory.(providerapi.RuntimeFactoryWithSections); ok {
        return withSections.NewRuntimeFromSections(ctx, profile, vals, opts...)
    }
    if vals == nil {
        return factory.NewRuntime(ctx, profile, opts...)
    }
    return nil, fmt.Errorf("runtime factory does not support pre-runtime section config")
}
```

If the team decides source compatibility does not matter, it can add the method directly to `RuntimeFactory`. I would not make that the first release unless all downstream adapters are in the same change set.

---

## 5. Implementation guide for an intern

### Phase 0: Read and trace before coding

Start with these files in this order:

1. `pkg/xgoja/providerapi/capabilities.go`
2. `pkg/xgoja/providerapi/module.go`
3. `pkg/xgoja/providerapi/commands.go`
4. `pkg/xgoja/app/spec.go`
5. `pkg/xgoja/app/factory.go`
6. `pkg/xgoja/app/module_sections.go`
7. `pkg/xgoja/providerutil/sections.go`
8. `pkg/xgoja/app/root.go`
9. `pkg/xgoja/app/run.go`
10. `pkg/xgoja/app/tui.go`
11. `pkg/xgoja/app/command_providers.go`
12. `geppetto/pkg/js/modules/geppetto/provider/provider.go`
13. `glazed/pkg/cmds/values/section-values.go`
14. `glazed/pkg/cmds/fields/field-value.go`
15. `glazed/pkg/cli/cobra-parser.go`

You should be able to answer these before editing:

- Where does `ModuleInstance.Config` become `ModuleContext.Config`?
- Which commands call `RuntimeFactory.NewRuntime`?
- How do Glazed values store field source history?
- Why is `RuntimeInitializerCapability` too late for Geppetto profile registry config?

### Phase 1: Add capability API

In `pkg/xgoja/providerapi/capabilities.go`, add:

```go
// ModuleConfigRequest identifies why config patching is happening and which
// selected module instance is being patched.
type ModuleConfigRequest struct {
    SectionContext SectionContext
    Descriptor     ModuleDescriptor
}

// ModuleConfigPatch is a JSON-config-keyed patch plus optional provenance.
type ModuleConfigPatch struct {
    Fields map[string]ModuleConfigValue
}

type ModuleConfigValue struct {
    Value any
    Log   []fields.ParseStep
}

// ModuleConfigCapability lets a provider convert parsed Glazed section values
// into a JSON-object-shaped config patch before Module.New is called.
type ModuleConfigCapability interface {
    PackageCapability

    ModuleConfigFromSections(
        context.Context,
        *values.Values,
        ModuleConfigRequest,
    ) (*ModuleConfigPatch, error)
}
```

Define `ModuleConfigRequest` next to `SectionContext` and `ModuleDescriptor`, because it is the pre-runtime sibling of section collection context.

### Phase 2: Add patch collection helper

In `pkg/xgoja/providerutil/sections.go`, add a helper that returns patches by module alias.

Important: do not use the same global dedupe pattern as `CollectConfigSections`.

Pseudocode:

```go
func ModuleConfigPatchesFromSections(
    ctx context.Context,
    vals *values.Values,
    descriptors []providerapi.ModuleDescriptor,
) (map[string]*providerapi.ModuleConfigPatch, error) {
    patches := map[string]*providerapi.ModuleConfigPatch{}

    if vals == nil {
        return patches, nil
    }

    for _, descriptor := range descriptors {
        merged := providerapi.NewModuleConfigPatch()

        for _, capability := range descriptor.PackageCapabilities {
            configCap, ok := capability.(providerapi.ModuleConfigCapability)
            if !ok {
                continue
            }

            partial, err := configCap.ModuleConfigFromSections(ctx, vals, providerapi.ModuleConfigRequest{Descriptor: descriptor})
            if err != nil {
                return nil, fmt.Errorf(
                    "module config from sections for %s.%s as %s capability %s: %w",
                    descriptor.PackageID,
                    descriptor.ModuleID,
                    descriptor.As,
                    capability.CapabilityID(),
                    err,
                )
            }
            merged.Merge(partial)
        }

        if !merged.Empty() {
            patches[descriptor.As] = merged
        }
    }

    return patches, nil
}
```

Caveat: alias alone can collide if the spec allows duplicate aliases. Today aliases are used for `require()` registration, so duplicate aliases should already be invalid or fail at runtime. If validation does not reject duplicates, add a test and consider keying by module index instead of alias.

### Phase 3: Add safe merge/clone helpers

A shallow `cloneMap` is not enough if nested maps are merged. Use a JSON-compatible deep clone.

Pseudocode:

```go
func cloneJSONMap(in map[string]any) map[string]any {
    if in == nil {
        return map[string]any{}
    }
    out := make(map[string]any, len(in))
    for k, v := range in {
        out[k] = cloneJSONValue(v)
    }
    return out
}

func cloneJSONValue(v any) any {
    switch x := v.(type) {
    case map[string]any:
        return cloneJSONMap(x)
    case []any:
        out := make([]any, len(x))
        for i, item := range x {
            out[i] = cloneJSONValue(item)
        }
        return out
    default:
        return x
    }
}

func DeepMergeJSONObjects(dst, src map[string]any) {
    for k, v := range src {
        srcMap, srcIsMap := v.(map[string]any)
        dstMap, dstIsMap := dst[k].(map[string]any)
        if srcIsMap && dstIsMap {
            DeepMergeJSONObjects(dstMap, srcMap)
            continue
        }
        dst[k] = cloneJSONValue(v)
    }
}
```

Policy:

- maps merge recursively;
- arrays replace;
- scalars replace;
- `null` replaces if the provider intentionally emits it.

### Phase 4: Add `NewRuntimeFromSections`

Refactor `app.RuntimeFactory.NewRuntime` so the code path is shared.

Pseudocode:

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

    descriptors, err := f.selectedModuleDescriptors(profile)
    if err != nil {
        return nil, err
    }

    patches, err := providerutil.ModuleConfigPatchesFromSections(ctx, vals, descriptors)
    if err != nil {
        return nil, err
    }

    modules := make([]engine.RuntimeModuleSpec, 0, len(runtime.Modules))
    for _, instance := range runtime.Modules {
        module, ok := f.providers.ResolveModule(instance.Package, instance.Name)
        if !ok {
            return nil, fmt.Errorf("runtime %s references unknown provider module %s.%s", profile, instance.Package, instance.Name)
        }

        config := cloneJSONMap(instance.Config)
        if patch, ok := patches[instance.Alias()]; ok {
            DeepMergeJSONObjects(config, patch.JSONMap())
        }

        patched := instance
        patched.Config = config
        modules = append(modules, providerRuntimeModuleSpec{
            instance: patched,
            module:   module,
            services: f.services,
        })
    }

    return f.newRuntimeWithModuleSpecs(ctx, modules, opts...)
}
```

Extract the existing builder code into a private helper:

```go
func (f *RuntimeFactory) newRuntimeWithModuleSpecs(
    ctx context.Context,
    modules []engine.RuntimeModuleSpec,
    opts ...require.Option,
) (*JSRuntime, error) {
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
```

### Phase 5: Update built-in commands

Change only the runtime creation call in each path:

- `evalSourceWithInitializers`: `factory.NewRuntimeFromSections(ctx, profile, vals)`.
- `runScriptFileWithInitializers`: `factory.NewRuntimeFromSections(ctx, profile, vals, requireOpt)`.
- `newXGojaTUIEvaluator`: `factory.NewRuntimeFromSections(ctx, profile, vals)`.
- `buildVerbCommands` invoker: `factory.NewRuntimeFromSections(ctx, profile, parsedValues, require.WithLoader(registry.RequireLoader()))`.

Do not remove `initRuntimeFromSections`. The phases are complementary:

```text
ModuleConfigCapability      → before Module.New
RuntimeInitializerCapability → after runtime exists
```

### Phase 6: Update provider-owned command guidance

Add the optional extended runtime factory interface or a `providerutil` helper. Then update `cmd/xgoja/doc/05-tutorial-providing-commands.md`:

```go
runtime, err := providerutil.NewRuntimeFromSections(
    ctx,
    c.RuntimeFactory,
    c.RuntimeProfile,
    vals,
)
```

If the command does not parse provider sections, it can keep using `NewRuntime`.

### Phase 7: Implement Geppetto sections and patching

Add a Geppetto capability in `geppetto/pkg/js/modules/geppetto/provider/provider.go` or a small sibling file.

Suggested fields for v1:

- `profile-registries` → `profileRegistries`
- `default-profile` → `defaultProfile`
- `turns-dsn` → `turnsDSN`
- `turns-db` → `turnsDB`

Do **not** add `allow-registry-load`, `allow-network`, `allow-tools`, `enable-storage`, or nested `turns` config for this capability. If turn storage is requested, the presence of `turns-dsn` or `turns-db` is the opt-in, matching Pinocchio's `js` command flags.

Geppetto capability pseudocode:

```go
type capability struct{}

func (capability) CapabilityID() string { return "geppetto.config" }

func (capability) ConfigSections(providerapi.SectionContext) ([]schema.Section, error) {
    return []schema.Section{schema.NewSection(...)}
}

func (capability) ModuleConfigFromSections(
    ctx context.Context,
    vals *values.Values,
    req providerapi.ModuleConfigRequest,
) (*providerapi.ModuleConfigPatch, error) {
    descriptor := req.Descriptor
    if descriptor.PackageID != PackageID || descriptor.ModuleID != geppettomodule.ModuleName {
        return nil, nil
    }

    b := providerutil.NewPatchBuilder(vals, "geppetto")
    if err := b.SetStringListIfProvided("profile-registries", "profileRegistries"); err != nil { return nil, err }
    if err := b.SetStringIfProvided("default-profile", "defaultProfile"); err != nil { return nil, err }
    if err := b.SetStringIfProvided("turns-dsn", "turnsDSN"); err != nil { return nil, err }
    if err := b.SetStringIfProvided("turns-db", "turnsDB"); err != nil { return nil, err }
    return b.Patch(), nil
}
```

Register one capability that implements both `ConfigSectionCapability` and `ModuleConfigCapability`:

```go
func Register(registry *providerapi.Registry) error {
    cap := capability{}
    return registry.Package(PackageID,
        providerapi.Module{...},
        providerapi.WithPackageCapability(cap),
    )
}
```

### Phase 8: Update docs

Update at least:

- `cmd/xgoja/doc/04-tutorial-providing-package-and-modules.md`
- `cmd/xgoja/doc/05-tutorial-providing-commands.md`
- potentially `cmd/xgoja/doc/06-buildspec-reference.md` if config precedence is documented there

Add a provider extension-point table row:

| Need | Use | Phase |
|---|---|---|
| Configure `Module.New` from CLI/config/env values | `ModuleConfigCapability` | after Glazed parse, before runtime construction |

---

## 6. Test plan

### 6.1 Providerutil tests

Add tests in `pkg/xgoja/providerutil/sections_test.go`:

1. Single descriptor + one config capability returns alias patch.
2. Two descriptors from same package get two calls and two patches.
3. Capability returning nil is ignored.
4. Capability errors are wrapped with package/module/alias/capability ID.
5. Deep merge keeps unrelated nested keys and replaces arrays.
6. Patch builder omits default-source values.
7. Patch builder includes config/env/cobra-source values, including explicit `false`.

### 6.2 Runtime factory tests

Add tests in `pkg/xgoja/app/module_sections_test.go` or a new `module_config_sections_test.go`:

1. `NewRuntimeFromSections` passes patched config to `Module.New`.
2. `NewRuntime` still passes only static config.
3. Two runtime creations with different values do not mutate `f.spec`.
4. Two aliases from the same package receive independent patches.
5. Unknown runtime/profile errors remain unchanged.

### 6.3 Built-in command integration tests

Update or add tests next to existing module-section tests:

- `eval_module_sections_test.go`: module factory config sees `--fixture-value` before eval.
- `run_module_sections_test.go`: script can require a module whose loader was constructed from patched config.
- `jsverbs_module_sections_test.go`: verb invocation passes `parsedValues` into pre-runtime patching.
- `tui_module_sections_test.go`: evaluator runtime sees patch.

### 6.4 Geppetto tests

Add tests in `geppetto/pkg/js/modules/geppetto/provider/provider_test.go`:

1. CLI/config field patch populates `profileRegistries` and `defaultProfile` before `decodeConfig`.
2. `turns-dsn` and `turns-db` patch the simplified Geppetto config before `Module.New`.
3. Absence of turn-store flags leaves storage disabled without requiring an `enableStorage` gate.
4. String and string-list registry values behave consistently with `decodeSourceEntries`.

---

## 7. Decision records

### Decision: Add a pre-runtime capability instead of extending RuntimeInitializerCapability

- **Context:** Runtime initializers run after modules have already been created.
- **Options considered:** Reuse `RuntimeInitializerCapability`; add a new lifecycle hook; pass `values.Values` directly into every `Module.New` call.
- **Decision:** Add `ModuleConfigCapability`.
- **Rationale:** It preserves the distinction between module construction and post-runtime setup.
- **Consequences:** Providers may implement two capabilities when they need both pre-runtime config and post-runtime side effects.
- **Status:** proposed.

### Decision: Apply config patches per selected descriptor

- **Context:** Capabilities are package-scoped, but module config is instance-scoped.
- **Options considered:** Deduplicate by package/capability; call per descriptor; call per package and broadcast patch.
- **Decision:** Call per selected descriptor.
- **Rationale:** A runtime can contain multiple instances/aliases from the same package, and each `Module.New` receives its own config.
- **Consequences:** Provider capability methods must be side-effect-light and idempotent.
- **Status:** proposed.

### Decision: Use a ModuleConfigPatch wrapper instead of raw values.Values

- **Context:** `ModuleInstance.Config` is JSON-object-shaped, while Glazed `values.Values` is section/field-shaped and carries parse history.
- **Options considered:** Return `map[string]any`; return `json.RawMessage`; return raw `*values.Values`; return a custom `ModuleConfigPatch` with value provenance; use a generic capability interface.
- **Decision:** Prefer `ModuleConfigPatch` (`JSON key → value + optional FieldValue log`) for the capability return type. Keep generics for provider-local helpers, not the registry-facing interface.
- **Rationale:** This keeps the returned patch scoped to one module instance and in module-config key space while preserving useful Glazed provenance.
- **Consequences:** The merge layer needs a `JSONMap()` conversion, and docs must be clear that raw command `values.Values` is input/provenance, not the module-config return representation.
- **Status:** proposed.

### Decision: Simplify Geppetto config to profile and turn-store inputs

- **Context:** The earlier Geppetto sketch included broad gates and nested storage config.
- **Options considered:** Keep `allowRegistryLoad`/`allowNetwork`/`allowTools`/`enableStorage`/nested `turns`; move turn storage to explicit flags; keep only profile registry/default-profile.
- **Decision:** Remove the broad gates from this capability and represent storage with concrete flags such as `turns-dsn` / `turns-db`.
- **Rationale:** Pinocchio already uses this shape, and presence of a DSN/DB is a clear opt-in without extra booleans.
- **Consequences:** The Geppetto provider config and tests should be simplified before or alongside implementing the xgoja capability.
- **Status:** proposed.

### Decision: Use Glazed field logs to avoid accidental default overrides

- **Context:** Decoding a section into a struct erases whether a value came from a default or user/config/env.
- **Options considered:** Include all decoded values; omit Go zero values; use pointer fields; inspect `FieldValue.Log` sources.
- **Decision:** Inspect `FieldValue.Log` and emit only non-default effective sources by default.
- **Rationale:** It supports explicit `false` while avoiding accidental overrides.
- **Consequences:** Tests must lock down source labels and default behavior.
- **Status:** proposed.

### Decision: Keep code generation unchanged for this ticket

- **Context:** Future explorations include plugins, JS config, and alternative codegen targets.
- **Options considered:** Change generated templates; introduce spec-to-IR; implement a plugin manager; add runtime API only.
- **Decision:** Add runtime API only.
- **Rationale:** The generated binary already embeds the app runtime stack; no template changes are needed to pass parsed values into runtime construction.
- **Consequences:** Future codegen/library/plugin targets can reuse the same runtime API.
- **Status:** proposed.

---

## 8. Future plugin/codegen influence

The plugin and codegen explorations should influence this design, but not expand this ticket.

### 8.1 Plugin architecture influence

If xgoja later supports WASM or subprocess plugins, a plugin may need to declare:

```json
{
  "capabilities": [
    "configSections",
    "moduleConfigPatch",
    "runtimeInitializer"
  ]
}
```

The current design is compatible if:

- `ModuleConfigCapability` is pure: input is context metadata + values, output is a JSON-object patch.
- The patch is serializable.
- The hook does not require direct access to `*goja.Runtime`.
- The hook phase is explicit and separate from runtime initialization.

### 8.2 Alternative codegen target influence

A future library target might export:

```go
func NewRuntimeFromSections(ctx context.Context, profile string, vals *values.Values, opts ...require.Option) (*engine.Runtime, error)
```

A future adapter target might receive an `app.Host` and call the same method. Because this ticket changes the app runtime layer instead of generated templates, these targets can share the implementation.

### 8.3 Spec-to-IR influence

If a future generator introduces an IR, module config patching should not be represented as build-time IR. It is runtime input. The IR can describe which packages and modules exist, but parsed CLI/config/env values remain command-execution data.

---

## 9. Risks and mitigations

| Risk | Mitigation |
|---|---|
| Glazed defaults overwrite YAML config | Use field source logs; emit only non-default values by default. |
| Same package appears twice and only first alias gets patched | Do not globally dedupe `ModuleConfigCapability`; call per descriptor. |
| Provider authors hand-write typo-prone maps | Provide `PatchBuilder` and docs with examples. |
| External command-provider runtime factories break | Add optional extended interface first, or coordinate a breaking change explicitly. |
| Deep merge mutates static nested maps | Deep-clone config and patch values before merge. |
| Security flags become too easy to override | Document precedence; require explicit allow flags for risky Geppetto behavior; test gates. |
| Command providers continue using old `NewRuntime` | Update docs and provide a helper; add tests using fixture command provider. |

---

## 10. Implementation checklist

- [ ] Add `ModuleConfigCapability` API.
- [ ] Add optional `RuntimeFactoryWithSections` API or decide on direct interface extension.
- [ ] Add `ModuleConfigPatchesFromSections` without global dedupe.
- [ ] Add JSON-compatible deep clone and deep merge helpers.
- [ ] Add source-aware `PatchBuilder` helpers.
- [ ] Add `app.RuntimeFactory.NewRuntimeFromSections`.
- [ ] Update `NewRuntime` wrapper.
- [ ] Update eval/run/TUI/jsverbs call sites.
- [ ] Update provider-owned command docs/helper.
- [ ] Add providerutil tests.
- [ ] Add app runtime factory tests.
- [ ] Add built-in command tests.
- [ ] Add Geppetto provider capability and tests.
- [ ] Update xgoja provider author docs.
- [ ] Run `go test ./pkg/xgoja/...` in go-go-goja and Geppetto provider tests.

---

## 11. Reference map

### go-go-goja

- `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/pkg/xgoja/providerapi/capabilities.go`
- `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/pkg/xgoja/providerapi/module.go`
- `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/pkg/xgoja/providerapi/commands.go`
- `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/pkg/xgoja/app/factory.go`
- `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/pkg/xgoja/app/module_sections.go`
- `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/pkg/xgoja/app/root.go`
- `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/pkg/xgoja/app/run.go`
- `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/pkg/xgoja/app/tui.go`
- `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/pkg/xgoja/app/command_providers.go`
- `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/pkg/xgoja/providerutil/sections.go`
- `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/cmd/xgoja/doc/04-tutorial-providing-package-and-modules.md`
- `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/cmd/xgoja/doc/05-tutorial-providing-commands.md`

### geppetto

- `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/geppetto/pkg/js/modules/geppetto/provider/provider.go`
- `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/geppetto/pkg/js/modules/geppetto/provider/provider_test.go`

### glazed

- `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/glazed/pkg/cmds/values/section-values.go`
- `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/glazed/pkg/cmds/fields/field-value.go`
- `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/glazed/pkg/cmds/fields/parse.go`
- `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/glazed/pkg/cli/cobra-parser.go`

### Existing GOJA-053 docs reviewed

- `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/03/GOJA-053--xgoja-moduleconfigcapability-for-pre-runtime-provider-flag-to-config-patching/design/01-module-config-capability.md`
- `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/03/GOJA-053--xgoja-moduleconfigcapability-for-pre-runtime-provider-flag-to-config-patching/design/02-xgoja-architecture-and-extensibility.md`
