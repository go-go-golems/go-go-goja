---
Title: xgoja v2-native runtime plan design and implementation guide
Ticket: GOJA-XGOJA-V2-RUNTIME-001
Status: active
Topics:
    - xgoja
    - architecture
    - code-generation
DocType: design-doc
Intent: ""
Owners: []
RelatedFiles:
    - Path: go-go-goja/cmd/xgoja/doc/16-migrating-to-xgoja-v2.md
      Note: migration guide that must be updated during implementation
    - Path: go-go-goja/cmd/xgoja/doc/17-xgoja-v2-reference.md
      Note: v2 reference documentation that must become authoritative for runtime plan semantics
    - Path: go-go-goja/cmd/xgoja/internal/generate/templates.go
      Note: current lossy v2-to-legacy runtime metadata conversion
    - Path: go-go-goja/cmd/xgoja/internal/plan/plan.go
      Note: v2 planner output with provider/source/module/command graphs
    - Path: go-go-goja/cmd/xgoja/internal/specv2/types.go
      Note: v2 YAML schema and command/source/provider model
    - Path: go-go-goja/pkg/xgoja/app/command_providers.go
      Note: current provider command-set runtime attachment
    - Path: go-go-goja/pkg/xgoja/app/runtime_spec.go
      Note: legacy generated runtime DTO to replace
    - Path: go-go-goja/pkg/xgoja/providers/http/serve.go
      Note: HTTP serve provider requiring command-scoped jsverb sources
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# xgoja v2-native runtime plan design and implementation guide

## Executive summary

xgoja already has a native `schema: xgoja/v2` configuration language, a v2 validator, a v2 planner, workspace-aware Go module resolution, provider graph resolution, and source graph resolution. The remaining problem is that generated xgoja binaries still pass through a legacy-shaped runtime metadata DTO before constructing the runtime application. That bridge flattens v2 concepts into older names such as `packages`, `modules`, `commandProviders`, `jsverbs`, `help`, and `assets`.

This ticket proposes a hard cutover: remove the legacy runtime metadata bridge entirely and make generated binaries consume a v2-native runtime plan end-to-end. The target implementation must not keep a legacy execution path, must not generate both legacy and v2 metadata for comparison, and must not add backward-compatibility shims around the old DTO. The public user-facing schema remains `xgoja/v2`; the internal generated runtime plan should finally match that schema.

The immediate symptom that exposed the design debt was a provider command-set example using this v2 configuration:

```yaml
commands:
  - id: serve
    type: provider.command-set
    provider: http
    name: serve
    mount: serve
    sources: [sites]
```

The v2 schema has `commands[].sources`, but the legacy bridge currently drops it while converting to `app.CommandProviderInstanceSpec`. The generated binary therefore creates a top-level `serve` command but does not mount the expected jsverb subcommands or HTTP flags. The correct fix is not another one-field patch to the bridge. The correct fix is to remove the bridge so v2 command, source, provider, module, and artifact concepts survive unchanged until command/runtime construction.

## Problem statement and scope

The problem is architectural drift between the current xgoja front door and the generated runtime back end.

The front door is v2-native. `cmd/xgoja/internal/specv2/types.go` defines `Config` with first-class `providers`, `runtime.modules`, `sources`, `commands`, and `artifacts`. The file explicitly states that the package defines the native xgoja/v2 configuration schema and that the schema is an intent-level planner input for providers, modules, sources, command surfaces, and artifacts (`cmd/xgoja/internal/specv2/types.go:1-25`).

The back end is still legacy-shaped. `pkg/xgoja/app/runtime_spec.go` defines `RuntimeSpec` with fields named `Packages`, `Modules`, `Commands`, `CommandProviders`, `JSVerbs`, `Help`, and `Assets` (`pkg/xgoja/app/runtime_spec.go:12-28`). That DTO is decoded by generated binaries and drives `pkg/xgoja/app.Host` command/runtime construction.

The scope of this ticket is to replace that runtime bridge with a v2-native runtime plan and update every affected layer:

- xgoja generated metadata.
- generated `main.go` and generated runtime-package output.
- runtime application builder.
- source registry APIs.
- provider command-set setup APIs.
- HTTP serve provider integration.
- docs, migration guide, examples, tests, and help pages.

The scope explicitly includes deleting or superseding legacy terminology in generated runtime internals. It does not require changing the public YAML schema to a hypothetical `xgoja/v3`; the intended user-facing schema remains `xgoja/v2`.

## Current-state architecture

### Current high-level flow

The current system has two different models in one pipeline:

```text
xgoja/v2 YAML
  -> specv2.ApplyDefaults / specv2.Validate
  -> plan.Compile
  -> generate.RenderRuntimeSpecJSON   (legacy-shaped bridge)
  -> embedded xgoja.gen.json
  -> pkg/xgoja/app.RuntimeSpec
  -> app.NewHost / app.NewRootCommand / app.RuntimeFactory
```

The cutover target is:

```text
xgoja/v2 YAML
  -> specv2.ApplyDefaults / specv2.Validate
  -> plan.Compile
  -> generated v2 runtime plan
  -> app.NewHostV2 / app.NewRootCommandV2 / app.RuntimeFactoryV2
```

After cutover, there should be no generated `packages/modules/commandProviders/jsverbs` bridge for v2 builds. Generated runtime code should use v2 concepts directly.

### The v2 schema already has the right conceptual model

`specv2.Config` already preserves the core concepts we want runtime code to see:

- `Providers []ProviderSpec` (`cmd/xgoja/internal/specv2/types.go:19`).
- `Runtime RuntimeSpec` with `Modules []RuntimeModuleSpec` (`cmd/xgoja/internal/specv2/types.go:20,73-82`).
- `Sources []SourceSpec` (`cmd/xgoja/internal/specv2/types.go:21,100-109`).
- `Commands []CommandSurfaceSpec` (`cmd/xgoja/internal/specv2/types.go:22,138-148`).
- `Artifacts []ArtifactSpec` (`cmd/xgoja/internal/specv2/types.go:23,150-159`).

The v2 command model already includes the information that provider command sets need:

```go
type CommandSurfaceSpec struct {
    ID       string
    Type     string
    Name     string
    Mount    string
    Provider string
    Sources  []string
    Modules  []string
    Config   map[string]any
    Lazy     bool
}
```

The important fields for the HTTP serve use case are `Type`, `Provider`, `Name`, `Mount`, and `Sources`. The current bridge loses `Sources`; a v2-native runtime builder must not.

### Defaults already make v2 practical for users

`specv2.ApplyDefaults` defaults missing schema to `xgoja/v2`, missing app name from `name`, Go version to `1.26`, Go module to `xgoja.generated/<name>`, and workspace mode to `auto` (`cmd/xgoja/internal/specv2/defaults.go:8-36`). This means examples should not need explicit `replace:` entries when a parent `go.work` covers local modules. The v2 design goal should preserve that user experience.

An idiomatic v2 example should look like this:

```yaml
schema: xgoja/v2
name: goja-chatdemo-server
workspace:
  mode: auto
providers:
  - id: http
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/http
  - id: sessionstream
    import: github.com/go-go-golems/sessionstream/pkg/js/modules/sessionstream/provider
runtime:
  modules:
    - provider: http
      name: express
    - provider: sessionstream
      name: sessionstream
sources:
  - id: sites
    kind: jsverbs
    from:
      dir: ./verbs
commands:
  - id: serve
    type: provider.command-set
    provider: http
    name: serve
    mount: serve
    sources: [sites]
```

No `provider.module.replace` should be needed when `workspace.mode: auto` resolves the modules from `go.work`.

### The v2 planner already builds the needed graphs

`plan.Plan` already stores v2 `Config`, Go module plan, provider graph, source graph, command plans, artifact plans, and runtime aliases (`cmd/xgoja/internal/plan/plan.go:18-29`). `plan.Compile` validates the spec, builds provider graph, resolves Go modules, builds source graph, resolves imports, and returns command/artifact plans (`cmd/xgoja/internal/plan/plan.go:46-82`).

The planner is not the main problem. The main problem begins after the planner, when code generation reshapes the v2 plan into legacy runtime metadata.

### The generator currently converts v2 into legacy runtime fields

The generator currently populates `runtimeSpec.Packages`, `runtimeSpec.Modules`, `runtimeSpec.Commands`, `runtimeSpec.CommandProviders`, `runtimeSpec.JSVerbs`, `runtimeSpec.Assets`, and help sources from v2 config (`cmd/xgoja/internal/generate/templates.go:360-405`). This conversion is the bridge that must be removed.

The clearest lossy point is `applyPlanRuntimeCommand`:

```go
func applyPlanRuntimeCommand(commands *app.CommandsSpec, providers *[]app.CommandProviderInstanceSpec, command specv2.CommandSurfaceSpec) {
    spec := app.CommandSpec{Enabled: true, Name: command.Name, Mount: command.Mount}
    switch command.Type {
    case "builtin.eval":
        commands.Eval = spec
    case "builtin.run":
        commands.Run = spec
    case "builtin.repl":
        commands.Repl = spec
    case "builtin.jsverbs":
        commands.JSVerbs = spec
    case "provider.command-set":
        *providers = append(*providers, app.CommandProviderInstanceSpec{
            ID: command.ID,
            Package: command.Provider,
            Name: command.Name,
            Mount: command.Mount,
            Modules: append([]string(nil), command.Modules...),
            Config: command.Config,
            Lazy: command.Lazy,
        })
    }
}
```

The original command has `Sources []string` (`cmd/xgoja/internal/specv2/types.go:144`), but the legacy `CommandProviderInstanceSpec` has no `Sources` field (`pkg/xgoja/app/runtime_spec.go:79-87`), and this conversion cannot carry the source binding. That is the observed bug. More importantly, it demonstrates that the legacy DTO is not expressive enough for v2.

### The runtime app builder consumes the legacy DTO everywhere

The current `pkg/xgoja/app` runtime builder takes `*RuntimeSpec`. `Host` stores `RuntimeSpec`, embedded JS verbs, help, and assets (`pkg/xgoja/app/host.go`, evidence search lines show `RuntimeSpec` used throughout). `NewRootCommand` decodes `SpecJSON` into `RuntimeSpec` and constructs commands from it. Built-in jsverb commands scan `runtimeSpec.JSVerbs`; command providers receive a source set built from `runtimeSpec.JSVerbs`.

This means a proper cutover cannot be completed only inside `cmd/xgoja/internal/generate`. The application runtime package must also stop depending on the legacy DTO.

### Provider command sets need v2 command-scoped sources

The HTTP provider serve command is the best concrete example. `newServeCommandSet` requires `ctx.JSVerbs` and calls `ctx.JSVerbs.ScanAllJSVerbSources()` (`pkg/xgoja/providers/http/serve.go:44-65`). It then creates a command for each jsverb (`pkg/xgoja/providers/http/serve.go:68-90`).

That is the right behavior, but the source set passed into the command should be the command's declared sources, not every runtime jsverb source and not an accidentally empty global list. In v2 terms:

```yaml
commands:
  - id: serve
    type: provider.command-set
    provider: http
    name: serve
    sources: [sites]
```

The provider command-set setup request should receive exactly the resolved source sets for `sites`.

## Gap analysis

### Gap 1: v2 command sources are not preserved

Evidence:

- V2 command has `Sources []string` (`cmd/xgoja/internal/specv2/types.go:144`).
- Legacy command-provider DTO does not (`pkg/xgoja/app/runtime_spec.go:79-87`).
- Conversion omits sources (`cmd/xgoja/internal/generate/templates.go:425-438`).

Impact:

- A generated `serve` command may exist without its jsverb subcommands.
- HTTP provider command flags may not appear because no jsverb commands were mounted.
- Users correctly writing v2 YAML see behavior that contradicts the v2 docs.

### Gap 2: Source model is split by legacy buckets

V2 has one `sources` list with `kind`, `from`, `include`, `exclude`, `extensions`, `language`, and `compile`. Runtime metadata splits sources into `jsverbs`, `help`, and `assets` fields (`pkg/xgoja/app/runtime_spec.go:25-27`). This makes each consumer handle a slightly different shape and forces conversion functions like `runtimeJSVerbSourceFromPlan`, `runtimeHelpSourceFromPlan`, and `runtimeAssetSourceFromPlan` (`cmd/xgoja/internal/generate/templates.go:441-480`).

Impact:

- Source-related behavior is duplicated.
- Command-scoped source selection is harder than it should be.
- Adding a new source kind requires another legacy bucket or special case.

### Gap 3: Provider terminology is inconsistent

V2 uses `providers`. The legacy runtime DTO uses `packages`. V2 runtime modules refer to `provider`; legacy module instances refer to `package` (`pkg/xgoja/app/runtime_spec.go:41-54`). This matters because docs, generated JSON, runtime code, and errors use different terms for the same thing.

Impact:

- New contributors must learn two vocabularies.
- Generated debugging artifacts are harder to map back to YAML.
- Migration docs must explain internal legacy names that should not exist.

### Gap 4: Built-in commands are hardcoded as separate fields

V2 commands are a list. Legacy `CommandsSpec` has fields for `Eval`, `Run`, `Repl`, and `JSVerbs` (`pkg/xgoja/app/runtime_spec.go:63-68`). This prevents a single runtime command loop from handling all command surfaces uniformly.

Impact:

- Built-in command handling and provider command handling diverge.
- Adding a new built-in command requires a DTO field rather than another command plan entry.
- Command ordering, mounting, and source/module binding are harder to reason about.

### Gap 5: Generated-runtime-package support inherits legacy assumptions

Generated runtime packages currently expose an API over generated runtime metadata and host construction. That path must also cut over. Otherwise a binary generated by `xgoja build` could be v2-native while a `runtime-package` artifact still embeds/loads legacy metadata.

Impact:

- Two generated artifact types would behave differently.
- Docs would remain ambiguous.
- The legacy DTO would survive through the runtime-package path.

## Proposed architecture

### Design goals

1. **Hard cutover:** remove the legacy runtime metadata path. Do not keep both runtime plans. Do not compare generated legacy and v2 plans in production code.
2. **V2 terminology end-to-end:** generated runtime data uses `providers`, `runtime.modules`, `sources`, `commands`, and `artifacts` concepts.
3. **Command-scoped dependencies:** commands receive their declared sources/modules/config directly.
4. **One source registry:** jsverbs, help, assets, and future source kinds are resolved through one source registry keyed by v2 source ID and kind.
5. **Workspace-first examples:** examples use `workspace.mode: auto` and avoid explicit `replace:` when a parent `go.work` already covers local modules.
6. **Docs updated in same PR:** no implementation is complete until the migration guide, v2 reference, user guide, tutorials, and examples reflect the cutover.
7. **No legacy left:** old runtime DTOs, generated JSON names, and docs referring to runtime `packages`/`commandProviders` as active internals must be removed or explicitly marked historical in archival notes only.

### New package-level structure

The implementation can keep package names stable if desired, but the conceptual split should be:

```text
cmd/xgoja/internal/specv2     YAML schema, defaults, validation
cmd/xgoja/internal/plan       v2 compile-time graph plan
cmd/xgoja/internal/generate   v2 code generation only
pkg/xgoja/app                 v2 runtime app builder
pkg/xgoja/providerapi         v2 provider/module/command APIs
pkg/xgoja/sourcegraph         source scanning/resolution primitives
```

Do not create `appv2` as a permanent parallel package unless it is immediately renamed into `app` before merge. A hard cutover means the public internal package should end in the v2-native state.

### V2 runtime plan DTO

Introduce a runtime plan that mirrors the v2 model but contains only runtime-safe data. It should not include build-only fields such as provider Go import paths, module versions, replace directives, or artifact output paths unrelated to runtime execution.

API sketch:

```go
package app

type RuntimePlan struct {
    Schema     string          `json:"schema"`
    Name       string          `json:"name"`
    App        AppPlan         `json:"app,omitempty"`
    Runtime    RuntimeSection  `json:"runtime,omitempty"`
    Sources    []SourcePlan    `json:"sources,omitempty"`
    Commands   []CommandPlan   `json:"commands,omitempty"`
    Assets     []SourcePlan    `json:"-"` // derived through SourceRegistry, not a top-level legacy field
    Help       []SourcePlan    `json:"-"` // derived through SourceRegistry, not a top-level legacy field
}

type AppPlan struct {
    Name       string          `json:"name,omitempty"`
    EnvPrefix  string          `json:"envPrefix,omitempty"`
    ConfigFile *ConfigFilePlan `json:"configFile,omitempty"`
}

type RuntimeSection struct {
    Modules []RuntimeModulePlan `json:"modules,omitempty"`
}

type RuntimeModulePlan struct {
    Provider string          `json:"provider"`
    Name     string          `json:"name"`
    As       string          `json:"as,omitempty"`
    Config   json.RawMessage `json:"config,omitempty"`
}

type SourcePlan struct {
    ID         string              `json:"id"`
    Kind       string              `json:"kind"`
    Origin     SourceOriginPlan    `json:"origin"`
    Include    []string            `json:"include,omitempty"`
    Exclude    []string            `json:"exclude,omitempty"`
    Extensions []string            `json:"extensions,omitempty"`
    Language   string              `json:"language,omitempty"`
    Compile    *CompilePlan        `json:"compile,omitempty"`
}

type SourceOriginPlan struct {
    Kind     string `json:"kind"` // embedded, disk, provider, workspace-resolved
    Path     string `json:"path,omitempty"`
    Provider string `json:"provider,omitempty"`
    Source   string `json:"source,omitempty"`
    Root     string `json:"root,omitempty"`
}

type CommandPlan struct {
    ID       string          `json:"id"`
    Type     string          `json:"type"`
    Name     string          `json:"name,omitempty"`
    Mount    string          `json:"mount,omitempty"`
    Provider string          `json:"provider,omitempty"`
    Sources  []string        `json:"sources,omitempty"`
    Modules  []string        `json:"modules,omitempty"`
    Config   json.RawMessage `json:"config,omitempty"`
    Lazy     bool            `json:"lazy,omitempty"`
}
```

The exact Go type names can change, but the invariant must not: the runtime plan keeps commands as commands and sources as sources.

### Source registry API

Runtime code should expose one source registry to command builders, help loaders, asset modules, and jsverb commands.

API sketch:

```go
type SourceRegistry interface {
    Source(id string) (SourceHandle, bool)
    Sources(ids []string, kinds ...SourceKind) ([]SourceHandle, error)
    SourcesByKind(kind SourceKind) []SourceHandle
}

type SourceHandle struct {
    ID         string
    Kind       SourceKind
    Origin     SourceOrigin
    Include    []string
    Exclude    []string
    Extensions []string
    Language   string
    Compile    CompileOptions
}
```

For JS verbs, keep an adapter that implements the existing provider-facing `JSVerbSourceSet` while internally selecting from the new source registry:

```go
func (r *sourceRegistry) JSVerbSourceSet(ids []string) (providerapi.JSVerbSourceSet, error) {
    sources, err := r.Sources(ids, SourceKindJSVerbs)
    if err != nil {
        return nil, err
    }
    return newJSVerbSourceSetFromHandles(r.providers, r.embedded, sources), nil
}
```

### Provider command-set setup API

Provider command sets need to receive a v2 command-scoped setup request.

Current HTTP provider code expects `providerapi.CommandSetContext` with `JSVerbs`, `RuntimeFactory`, and selected modules. The new context should make the command's declared dependencies explicit.

API sketch:

```go
type CommandSetContext struct {
    Context        context.Context
    ID             string
    Provider       string
    Name           string
    Mount          string
    Config         json.RawMessage
    RuntimeFactory RuntimeFactory
    Host           HostServices

    SelectedModules []ModuleDescriptor

    // Source access is scoped to the command's sources field.
    Sources SourceRegistry
    JSVerbs JSVerbSourceSet // convenience view over Sources filtered to jsverbs
}
```

For a command:

```yaml
commands:
  - id: serve
    type: provider.command-set
    provider: http
    name: serve
    sources: [sites]
```

`CommandSetContext.JSVerbs` should scan only `sites`. It should not scan unrelated jsverb sources and should never be empty because of a conversion loss.

### Runtime host construction

The runtime app builder should construct root commands from the `[]CommandPlan` list.

Pseudocode:

```go
func (h *Host) AttachCommands(root *cobra.Command) error {
    for _, cmd := range h.Plan.Commands {
        switch cmd.Type {
        case "builtin.eval":
            rootOrMount(root, cmd).AddCommand(newEvalCommand(h.Factory, h.Plan, cmd))
        case "builtin.run":
            rootOrMount(root, cmd).AddCommand(newRunCommand(h.Factory, h.Plan, cmd))
        case "builtin.repl":
            rootOrMount(root, cmd).AddCommand(newReplCommand(h.Factory, h.Plan, cmd))
        case "builtin.jsverbs":
            sourceSet := h.Sources.JSVerbSourceSet(cmd.Sources)
            rootOrMount(root, cmd).AddCommand(newJSVerbCommands(h.Factory, sourceSet, cmd))
        case "provider.command-set":
            provider := h.Providers.ResolveCommandSetProvider(cmd.Provider, cmd.Name)
            sourceView := h.Sources.CommandScoped(cmd.Sources)
            selectedModules := h.selectModules(cmd.Modules)
            commandSet := provider.NewCommandSet(providerapi.CommandSetContext{
                ID: cmd.ID,
                Provider: cmd.Provider,
                Name: cmd.Name,
                Mount: cmd.Mount,
                Config: cmd.Config,
                RuntimeFactory: h.Factory,
                Host: h.Services,
                Sources: sourceView,
                JSVerbs: sourceView.JSVerbs(),
                SelectedModules: selectedModules,
            })
            rootOrMount(root, cmd).AddCommand(commandSet.Commands...)
        default:
            return fmt.Errorf("unsupported command type %q", cmd.Type)
        }
    }
}
```

The key difference is that there is no `CommandsSpec` and no `CommandProviders` list. There is one command loop.

### Generated files after cutover

Generated builds should embed something named after the v2 model, not `xgoja.gen.json` if that name implies the old runtime DTO.

Recommended generated files:

```text
main.go
xgoja.v2.runtime.json
xgoja_embed/jsverbs/<source-id>/...
xgoja_embed/help/<source-id>/...
xgoja_embed/assets/<source-id>/...
```

Generated `main.go` sketch:

```go
package main

import (
    "embed"
    "os"

    "github.com/go-go-golems/go-go-goja/pkg/xgoja/app"
    "github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
    httpprovider "github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/http"
)

//go:embed xgoja.v2.runtime.json
var runtimePlanJSON []byte

//go:embed xgoja_embed/jsverbs/**
var embeddedJSVerbs embed.FS

func main() {
    providers := providerapi.NewProviderRegistry()
    if err := httpprovider.Register(providers); err != nil {
        panic(err)
    }
    root, err := app.NewRootCommand(app.Options{
        Providers: providers,
        PlanJSON: runtimePlanJSON,
        EmbeddedSources: app.EmbeddedSources{
            JSVerbs: embeddedJSVerbs,
        },
        Out: os.Stdout,
    })
    if err != nil {
        panic(err)
    }
    if err := root.Execute(); err != nil {
        os.Exit(1)
    }
}
```

The names are illustrative. The requirement is that the generated plan is v2-native and the generated host calls a v2-native app constructor.

### Generated runtime package after cutover

Runtime-package artifacts should use the same `RuntimePlan` and app builder as generated binaries. Do not leave runtime-package artifacts on the old DTO.

API sketch:

```go
package xgojaruntime

type Bundle struct {
    providers *providerapi.ProviderRegistry
    plan      *app.RuntimePlan
    embedded  app.EmbeddedSources
    options   Options
}

func NewBundle(opts Options) (*Bundle, error) {
    providers := providerapi.NewProviderRegistry()
    registerGeneratedProviders(providers)
    plan, err := app.DecodeRuntimePlan(runtimePlanJSON)
    if err != nil { return nil, err }
    return &Bundle{providers: providers, plan: plan, embedded: generatedEmbeddedSources(), options: opts}, nil
}

func (b *Bundle) NewRuntime(ctx context.Context) (*engine.Runtime, error) {
    host := app.NewHostWithOptions(b.providers, b.plan, app.HostOptions{EmbeddedSources: b.embedded})
    return host.NewRuntime(ctx)
}
```

## Required hard-cutover removals

The implementation must remove or replace these active runtime concepts:

- `pkg/xgoja/app.RuntimeSpec` as the generated binary runtime DTO.
- `PackageSpec` as active runtime terminology; use provider terminology.
- `ModuleInstanceSpec.Package`; use `Provider`.
- `CommandsSpec` as separate `Eval/Run/Repl/JSVerbs` fields.
- `CommandProviderInstanceSpec` as a separate command-provider bucket.
- `JSVerbSourceSpec`, `HelpSourceSpec`, and `AssetSourceSpec` as independent top-level runtime buckets when used only to avoid a unified source registry.
- `RenderRuntimeSpecJSON` or equivalent functions that reshape v2 plans into legacy runtime metadata.
- Generated `xgoja.gen.json` if it stores the legacy shape.

If any of these names remain, they must either be:

1. deleted,
2. renamed/reworked into v2-native types, or
3. moved into test fixtures explicitly labeled as historical migration inputs.

No production code should need to understand the legacy runtime DTO after the cutover.

## Documentation update requirements

The implementation is not done until documentation is updated. The PR must update these documents:

1. `cmd/xgoja/doc/02-user-guide.md`
   - Explain the v2-native runtime flow.
   - Show provider command sets with `sources` and `modules`.
   - Remove language implying that v2 specs are converted into legacy runtime metadata.

2. `cmd/xgoja/doc/16-migrating-to-xgoja-v2.md`
   - Add a section explaining the hard runtime cutover.
   - State that generated binaries now preserve v2 concepts end-to-end.
   - Explain that `workspace.mode: auto` is preferred over explicit `replace` when a parent `go.work` covers local modules.
   - Add a troubleshooting section for provider command-set `sources` and runtime module aliases.

3. `cmd/xgoja/doc/17-xgoja-v2-reference.md`
   - Make the runtime plan model authoritative.
   - Document command-scoped `sources` for `provider.command-set`.
   - Clarify that `mount` is a CLI mount, not an HTTP mount.
   - Document source kinds through the unified source registry model.

4. `cmd/xgoja/doc/11-provider-runtime-config-and-host-services.md`
   - Update provider author guidance for the new `CommandSetContext`.
   - Explain how command providers receive command-scoped sources and selected modules.
   - Keep host-service guidance current.

5. `cmd/xgoja/doc/15-tutorial-protobuf-builder-provider.md`
   - Ensure provider examples use v2-native terminology and workspace resolution.

6. `examples/xgoja/*/README.md` and `examples/xgoja/*/xgoja.yaml`
   - Update examples to remove unnecessary explicit replaces where `workspace.mode: auto` works.
   - Ensure provider command-set examples work with `sources`.
   - Update the HTTP serve example command syntax to match the new runtime behavior.

7. Any generated-runtime-package documentation and examples.
   - Replace references to legacy embedded runtime spec with the v2 runtime plan.

Docs must be validated through existing help/doc tests. The migration guide is a first-class deliverable, not an optional follow-up.

## Decision records

### Decision: hard cutover instead of dual runtime paths

- **Context:** The legacy bridge is already causing lossy conversion from v2 commands to runtime command-provider setup. A dual path would preserve ambiguity and require future contributors to understand both models.
- **Options considered:** Patch the bridge; generate both legacy and v2 plans temporarily; hard cutover to v2-native runtime plan.
- **Decision:** Perform a hard cutover. Generated binaries and runtime packages should use only the v2-native runtime plan.
- **Rationale:** The user explicitly requested no legacy leftovers. The codebase already has a v2 planner; keeping the old DTO would continue to invite mismatches.
- **Consequences:** The implementation must update more tests/examples/docs in one PR. It reduces migration ambiguity and makes failures easier to diagnose.
- **Status:** accepted.

### Decision: preserve v2 command list at runtime

- **Context:** Legacy `CommandsSpec` splits built-ins into fields and provider commands into a separate list, losing common command semantics.
- **Options considered:** Add missing fields to `CommandProviderInstanceSpec`; keep built-ins as special fields; represent every command as a `CommandPlan`.
- **Decision:** Use one runtime `[]CommandPlan` list.
- **Rationale:** This matches the YAML and lets all commands share `id`, `type`, `name`, `mount`, `sources`, `modules`, `config`, and `lazy` behavior.
- **Consequences:** Command installation becomes a switch over `CommandPlan.Type`; tests should assert built-ins and providers both use the same command path.
- **Status:** accepted.

### Decision: unified source registry

- **Context:** V2 sources have one schema with `kind`; legacy runtime metadata has separate source buckets.
- **Options considered:** Keep separate `JSVerbs`, `Help`, and `Assets`; add per-kind slices under a new DTO; use one source registry with kind filters.
- **Decision:** Use one source registry and derive kind-specific views from it.
- **Rationale:** Command providers need command-scoped source selection, and future source kinds should not require new top-level runtime fields.
- **Consequences:** Source scanning adapters must be rewritten, but command/help/asset behavior becomes more regular.
- **Status:** accepted.

### Decision: provider command contexts receive resolved command-scoped sources

- **Context:** HTTP serve must scan only the jsverb sources attached to the serve command.
- **Options considered:** Let providers scan all runtime sources; add source IDs only; pass resolved source handles/views.
- **Decision:** Pass a source registry view scoped to `commands[].sources`, with a JS verb convenience adapter.
- **Rationale:** Providers should not duplicate source ID resolution or accidentally scan unrelated sources.
- **Consequences:** Provider API changes are required. HTTP serve tests must assert source scoping.
- **Status:** accepted.

### Decision: docs and migration guide are part of definition of done

- **Context:** This refactor changes internal architecture and the mental model for provider authors and users.
- **Options considered:** Update docs later; update only v2 reference; update all relevant docs in the same implementation.
- **Decision:** Update all relevant docs and examples in the same PR.
- **Rationale:** Leaving docs behind would preserve the old terminology and make the hard cutover incomplete.
- **Consequences:** The implementation PR is larger, but the feature is reviewable as a complete migration.
- **Status:** accepted.

## Implementation plan

### Phase 1: Inventory and failing tests

1. Add tests that demonstrate current breakage:
   - A v2 spec with `provider.command-set` and `sources: [sites]` must generate a binary whose `serve` command exposes the jsverb commands.
   - The generated runtime metadata must contain `commands[].sources` in v2-native form.
   - No generated `commandProviders` field should exist.
2. Add grep-style test or CI check for active legacy runtime markers in generated output:
   - `"packages"` as provider list.
   - `"commandProviders"`.
   - top-level `"jsverbs"` instead of `sources`.
   - `xgoja.gen.json` if it contains legacy shape.
3. Run the existing xgoja example suite to establish the failing baseline.

### Phase 2: Define v2 runtime plan types

1. Replace `pkg/xgoja/app.RuntimeSpec` with `RuntimePlan` or rename/rework it in place.
2. Replace `PackageSpec` with `ProviderPlan` if provider identity is still needed at runtime.
3. Replace `ModuleInstanceSpec.Package` with `RuntimeModulePlan.Provider`.
4. Replace `CommandsSpec` and `CommandProviderInstanceSpec` with `[]CommandPlan`.
5. Replace per-kind top-level source specs with `[]SourcePlan` plus `SourceRegistry`.

Pseudocode for type migration:

```go
// Before
type RuntimeSpec struct {
    Packages []PackageSpec
    Modules []ModuleInstanceSpec
    Commands CommandsSpec
    CommandProviders []CommandProviderInstanceSpec
    JSVerbs []JSVerbSourceSpec
    Help HelpSpec
    Assets []AssetSourceSpec
}

// After
type RuntimePlan struct {
    Schema string
    Name string
    App AppPlan
    Runtime RuntimeSection
    Sources []SourcePlan
    Commands []CommandPlan
}
```

### Phase 3: Rewrite generator output

1. Replace `RenderRuntimeSpecJSON` with `RenderRuntimePlanJSON`.
2. Generate `xgoja.v2.runtime.json` or an equivalently named v2-native artifact.
3. Remove `applyPlanRuntimeCommand`, `runtimeJSVerbSourceFromPlan`, `runtimeHelpSourceFromPlan`, and `runtimeAssetSourceFromPlan` if their only purpose is legacy reshaping.
4. Generate runtime source origins from `plan.SourceGraph` rather than reconstructing sources from raw config alone.
5. Ensure generated `go.mod` still uses workspace resolution from `plan.GoModules`; `RenderGoModPlan` already resolves workspace replacements from `plannedGoModules` (`cmd/xgoja/internal/generate/gomod.go:55-80`).

### Phase 4: Rewrite runtime app builder

1. Change `NewRootCommand` to decode `RuntimePlan`.
2. Change `NewHost` / `NewHostWithOptions` to store `*RuntimePlan`.
3. Change `RuntimeFactory` module setup to read `plan.Runtime.Modules`.
4. Replace built-in command attachment logic with a loop over `plan.Commands`.
5. Replace old jsverb command builder with a command-scoped source lookup.
6. Replace help loading and asset store setup to query sources by kind.

Pseudocode:

```go
func NewRootCommand(opts Options) (*cobra.Command, error) {
    plan, err := DecodeRuntimePlan(opts.PlanJSON)
    if err != nil { return nil, err }
    host := NewHostWithOptions(opts.Providers, plan, HostOptions{EmbeddedSources: opts.EmbeddedSources})
    root := newRoot(plan.App.Name)
    if err := host.InstallFramework(root); err != nil { return nil, err }
    if err := host.AttachCommands(root); err != nil { return nil, err }
    return root, nil
}
```

### Phase 5: Update provider API and providers

1. Update `providerapi.CommandSetContext` to include command ID, provider, command config, scoped source registry, and selected modules.
2. Update `pkg/xgoja/providers/http/serve.go` so `newServeCommandSet` uses the command-scoped JS verb source set.
3. Ensure hot reload uses the same scoped source set for re-scan and watch-root calculation.
4. Update tests in `pkg/xgoja/providers/http` and `pkg/xgoja/app`.

Important HTTP serve assertions:

- `ctx.JSVerbs == nil` remains an error for serve.
- `ctx.JSVerbs.ScanAllJSVerbSources()` sees only command-attached sources.
- A serve command with no `sources` should fail clearly unless provider-shipped default sources are explicitly supported.

### Phase 6: Runtime package artifacts

1. Update runtime-package generation to embed the v2 runtime plan.
2. Update generated `NewBundle` and `NewRuntime` to use the v2 app builder.
3. Update `examples/xgoja/14-generated-runtime-package` and its tests/docs.
4. Remove any legacy runtime DTO references from generated package output.

### Phase 7: Documentation and examples

1. Update all docs listed in “Documentation update requirements”.
2. Update examples to use idiomatic v2.
3. Ensure `examples/xgoja/13-http-serve-jsverbs` passes with provider command-set `sources`.
4. Finish `sessionstream/examples/goja-chatdemo-server` once the v2 command path works.
5. Update migration guide to say there is no legacy runtime bridge.

### Phase 8: Removal sweep

Run repository searches and remove active legacy terms from implementation paths:

```bash
rg -n "CommandProviderInstanceSpec|CommandsSpec|PackageSpec|JSVerbSourceSpec|xgoja.gen.json|commandProviders|packages" cmd/xgoja pkg/xgoja examples/xgoja
```

Allowed matches after cutover:

- Historical changelog or ticket docs.
- Tests explicitly verifying old input migration if such tests are still useful.
- User-facing docs only when describing old versions in the migration guide.

Everything else should be renamed or removed.

## Testing strategy

### Unit tests

- `cmd/xgoja/internal/specv2`: v2 schema defaults and validation still pass.
- `cmd/xgoja/internal/plan`: command plans preserve `sources`, `modules`, `config`, and `lazy`.
- `cmd/xgoja/internal/generate`: generated runtime plan JSON contains `commands` list and `sources` list; it does not contain `commandProviders` or top-level `jsverbs`.
- `pkg/xgoja/app`: root command construction from v2 plan attaches built-ins and provider command sets.
- `pkg/xgoja/providers/http`: HTTP serve command receives command-scoped jsverb sources.

### Golden tests

Add or update golden output for generated `main.go` and runtime plan JSON. The golden should show:

```json
{
  "schema": "xgoja/runtime/v2",
  "commands": [
    {
      "id": "serve",
      "type": "provider.command-set",
      "provider": "http",
      "name": "serve",
      "mount": "serve",
      "sources": ["sites"]
    }
  ],
  "sources": [
    {
      "id": "sites",
      "kind": "jsverbs"
    }
  ]
}
```

It should not show:

```json
{
  "commandProviders": [],
  "jsverbs": [],
  "packages": []
}
```

### Example smoke tests

Required examples:

```bash
make -C examples/xgoja/13-http-serve-jsverbs smoke
make -C examples/xgoja/14-generated-runtime-package smoke
make -C examples/xgoja/15-protobuf-builder-provider smoke
```

After the sessionstream example is committed:

```bash
make -C ../sessionstream/examples/goja-chatdemo-server smoke
```

### Full validation

Run:

```bash
go test ./cmd/xgoja/... ./pkg/xgoja/... ./pkg/gojahttp ./modules/express -count=1
go test ./... -count=1
```

Also run docs/help validation tests that load xgoja help pages.

## Risks and open questions

### Risk: large PR surface

A hard cutover touches generation, runtime app, provider APIs, docs, and examples. The mitigation is to phase internally but merge as one coherent cutover PR with strong tests.

### Risk: generated runtime packages may have hidden users

Runtime packages are a public-ish integration surface. The cutover should keep the high-level generated package API stable where possible (`NewBundle`, `NewRuntime`), while replacing internals.

### Risk: provider API changes affect all providers

Built-in providers and examples must be updated in the same PR. Custom external providers may need migration guidance. Since this is internal project work and the user requested a hard cutover, do not keep old provider command-set APIs as compatibility shims.

### Open question: runtime plan JSON schema name

Options:

- `xgoja/runtime/v2`
- `xgoja/v2-runtime`
- no separate schema marker, relying on generated code version

Recommendation: include `schema: xgoja/runtime/v2` in generated runtime JSON so tests can fail clearly if old JSON is embedded.

### Open question: naming of app package types

Options:

- Rename `RuntimeSpec` to `RuntimePlan`.
- Keep `RuntimeSpec` name but change its shape completely.

Recommendation: use `RuntimePlan` because it distinguishes v2 runtime data from the historical DTO and communicates that the data is already planned/resolved.

## Implementation checklist

- [ ] Add failing tests for provider command-set `sources`.
- [ ] Define v2 runtime plan types.
- [ ] Replace generated runtime metadata with v2 runtime plan JSON.
- [ ] Rewrite app host/root/factory to consume v2 runtime plan.
- [ ] Replace runtime source buckets with source registry.
- [ ] Update provider command-set context to receive scoped sources.
- [ ] Update HTTP serve provider.
- [ ] Update generated runtime-package path.
- [ ] Remove active legacy DTO/types/conversion functions.
- [ ] Update `cmd/xgoja/doc/02-user-guide.md`.
- [ ] Update `cmd/xgoja/doc/16-migrating-to-xgoja-v2.md`.
- [ ] Update `cmd/xgoja/doc/17-xgoja-v2-reference.md`.
- [ ] Update provider author docs and tutorials.
- [ ] Update examples and READMEs.
- [ ] Add grep/test guard against legacy generated metadata returning.
- [ ] Run full test and example smoke suite.

## File reference map

Start reading here:

- `cmd/xgoja/internal/specv2/types.go`: v2 user-facing schema.
- `cmd/xgoja/internal/specv2/defaults.go`: v2 defaults, including `workspace.mode: auto`.
- `cmd/xgoja/internal/plan/plan.go`: planner output and provider/source/module graph resolution.
- `cmd/xgoja/internal/generate/templates.go`: current lossy conversion into legacy runtime metadata; primary deletion target.
- `cmd/xgoja/internal/generate/gomod.go`: Go module/workspace replacement generation; mostly reusable.
- `pkg/xgoja/app/runtime_spec.go`: current legacy runtime DTO; primary replacement target.
- `pkg/xgoja/app/host.go`: root command attachment and command-provider mounting.
- `pkg/xgoja/app/root.go`: generated root command construction and built-in commands.
- `pkg/xgoja/app/command_providers.go`: current provider command-set construction.
- `pkg/xgoja/app/jsverb_sources.go`: current JS verb source set adapter.
- `pkg/xgoja/providerapi/module.go`: module setup API.
- `pkg/xgoja/providerapi/capabilities.go`: host services and runtime initializer capabilities.
- `pkg/xgoja/providers/http/serve.go`: provider command-set consumer that exposed the source-scoping bug.
- `cmd/xgoja/doc/16-migrating-to-xgoja-v2.md`: must be updated for the hard runtime cutover.
- `cmd/xgoja/doc/17-xgoja-v2-reference.md`: must become authoritative for runtime plan semantics.

## Appendix: mental model for interns

Think of xgoja as a compiler and runtime host generator.

At compile time, xgoja reads a YAML file that says:

- Which Go provider packages are available.
- Which provider modules JavaScript can import.
- Which local/provider/workspace source files exist.
- Which CLI commands the generated binary should expose.
- Which artifacts should be built.

At runtime, the generated binary should not care about Go module replacements or YAML parsing. It should care about:

- Which modules are selected.
- Which commands exist.
- Which source sets those commands can read.
- Which embedded files are available.
- Which provider command/module factories should be called.

The legacy bridge mixed those concerns by reshaping v2 concepts into older runtime fields. The v2-native cutover keeps the compile-time and runtime data models aligned: a v2 command remains a command, a v2 source remains a source, and a provider remains a provider all the way through the generated binary.
