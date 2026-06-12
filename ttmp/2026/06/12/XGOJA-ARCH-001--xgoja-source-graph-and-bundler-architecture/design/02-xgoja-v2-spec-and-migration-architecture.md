---
Title: XGoja v2 spec and migration architecture
Ticket: XGOJA-ARCH-001
Status: active
Topics:
    - goja
    - xgoja
    - typescript
    - tooling
    - developer-experience
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/xgoja/internal/buildspec/build_spec.go
      Note: Current v1 schema to migrate from
    - Path: cmd/xgoja/internal/generate/gomod.go
      Note: Generated Go module rendering should consume v2 workspace plan
    - Path: examples/xgoja/15-typescript-jsverbs/xgoja.yaml
      Note: Concrete v1 TypeScript jsverbs example used for v2 migration sketch
    - Path: go-go-goja/cmd/xgoja/doc/02-user-guide.md
      Note: Existing user-facing v1 docs that need migration documentation.
    - Path: go-go-goja/cmd/xgoja/internal/buildspec/build_spec.go
      Note: Current v1 xgoja.yaml schema that should become migratable legacy input.
    - Path: go-go-goja/cmd/xgoja/internal/generate/gomod.go
      Note: Current generated go.mod rendering should consume v2 workspace and GoModulePlan data.
    - Path: go-go-goja/examples/xgoja/15-typescript-jsverbs/xgoja.yaml
      Note: Recent v1 TypeScript jsverbs example used as a concrete migration target.
    - Path: go-go-goja/pkg/jsverbs/scan.go
      Note: Current jsverb source scanning should migrate behind v2 source graph adapters.
    - Path: go-go-goja/pkg/tsscript/compiler.go
      Note: Current TypeScript compiler facade should become a compiler backend selected by v2 source compile policy.
    - Path: go-go-goja/pkg/xgoja/app/runtime_spec.go
      Note: Current runtime spec that v2 planning should replace or generate from a cleaner model.
    - Path: go-go-goja/pkg/xgoja/providerapi/commands.go
      Note: Command sets and JSVerbSourceSet map to explicit v2 command surfaces and source dependencies.
    - Path: go-go-goja/pkg/xgoja/providerapi/module.go
      Note: Provider modules become first-class provider graph capabilities in v2.
    - Path: go-go-goja/pkg/xgoja/providerapi/verbs.go
      Note: Provider-shipped verb sources become v2 source origins.
    - Path: pkg/jsverbs/scan.go
      Note: Current jsverb scanner to migrate behind v2 source graph adapters
    - Path: pkg/tsscript/compiler.go
      Note: Compiler backend selected by v2 source compile policy
    - Path: pkg/xgoja/app/runtime_spec.go
      Note: Current runtime spec that should be generated from v2 planning
    - Path: pkg/xgoja/providerapi/commands.go
      Note: Command sets and JSVerbSourceSet map to v2 command surfaces
    - Path: pkg/xgoja/providerapi/module.go
      Note: Provider capabilities represented explicitly in v2 providers/runtime modules
ExternalSources:
    - local:01-architecture-reassessment-prompt.md
Summary: A v2 xgoja.yaml architecture that aligns the user-facing spec with source graphs, provider graphs, Go workspace resolution, command surfaces, artifacts, and migration tooling.
LastUpdated: 2026-06-12T12:05:00-04:00
WhatFor: Use when designing the next xgoja config format, migration tooling, and planner implementation without preserving v1 as the long-term internal architecture.
WhenToUse: Before implementing sourcegraph/provider-graph planning, workspace resolution, command surfaces, build artifacts, or migration tooling for existing xgoja users.
---


# XGoja v2 spec and migration architecture

## Executive summary

The first architecture document for `XGOJA-ARCH-001` proposes a source graph, provider graph, import resolver, Go workspace resolver, build plan, and runtime plan. That design can be implemented while preserving the current `xgoja.yaml` shape, but doing so would force the new architecture to infer modern concepts from older names and historical structure. If backwards compatibility is not a hard constraint, the cleaner path is to define a v2 spec that directly names the concepts the planner needs.

The current v1 spec uses `packages`, `modules`, `jsverbs`, `commandProviders`, `help`, and `assets`. Those names made sense while xgoja was primarily a generator for a runtime with selected provider packages. The system is now more general. It compiles Go-backed JavaScript runtimes from provider packages, runtime modules, source sets, command surfaces, generated declarations, embedded assets, help pages, TypeScript compilation policy, and Go workspace module resolution.

The v2 spec should therefore expose those concepts directly:

- `providers` describes Go packages that contribute capabilities.
- `runtime.modules` selects Go-backed JavaScript modules by provider and alias.
- `sources` describes disk, workspace, embedded, provider, and virtual source sets.
- `commands` describes user-facing command surfaces and their dependencies.
- `artifacts` describes generated outputs such as binaries, runtime packages, declarations, embedded assets, and help bundles.
- `workspace` describes Go workspace resolution for local provider development.
- `profiles` optionally describe development, CI, and release variations.

The migration strategy should be explicit: v1 becomes legacy input, not the permanent internal model. xgoja should provide `xgoja migrate-spec` to convert v1 files into v2 files, produce warnings for ambiguous cases, and update examples and documentation to v2. Existing packages that use xgoja can migrate when they need new functionality.

## Why v2 changes the architecture

Without a v2 spec, the planner must translate from v1 concepts into new internals every time:

```text
v1 xgoja.yaml
  packages
  modules
  jsverbs
  commandProviders
  assets
  help
        ↓
compatibility inference
        ↓
provider graph + source graph + build plan + runtime plan
```

That translation will be full of historical edge cases. For example, `jsverbs[].embed` currently determines both source origin and generated output behavior. `packages[].replace` handles local provider development but does not express general workspace resolution. `commandProviders` are separate from builtin commands even though both are command surfaces. `typescript.external` duplicates runtime-module alias information that a provider graph should already know.

With v2, the user-facing spec can match the internal graph:

```text
v2 xgoja.yaml
  providers
  workspace
  runtime.modules
  sources
  commands
  artifacts
  profiles
        ↓
provider graph + source graph + build plan + runtime plan
```

This reduces translation. It also makes generated plan debugging easier because the plan can point back to fields that already use the same vocabulary.

## Design goals

The v2 spec should satisfy these goals:

1. **Align user-facing config with planner concepts.** The spec should name providers, runtime modules, sources, command surfaces, artifacts, and workspace resolution directly.
2. **Separate source origin from compilation policy.** A source can come from disk, provider `fs.FS`, embedded generated files, or a workspace directory; compilation can happen at runtime, build time, or not at all.
3. **Separate provider packages from runtime modules.** A Go provider package can contribute many capabilities; selecting a JavaScript module from that package is a runtime decision.
4. **Make command surfaces explicit.** Builtin commands and provider command sets should be represented by the same top-level `commands` list.
5. **Make Go workspace resolution explicit.** Local provider development should use `go.work` without repeated manual replacements.
6. **Support migration over compatibility.** The system should include a migration tool and documentation rather than keeping v1 as the long-term internal shape.
7. **Keep runtime behavior stable.** v2 should not require replacing goja, runtime owner, provider module factories, or jsverbs command metadata.

## Non-goals

- v2 does not introduce npm package management in the first version.
- v2 does not require goja to execute ECMAScript modules directly.
- v2 does not require every source set to be prebundled at build time.
- v2 does not make v1 and v2 equally important forever. v1 should be migrated and eventually retired.
- v2 does not require preserving comments during automated migration. The migration tool should preserve semantics and produce readable output, but exact formatting and comments can be lost.

## Current v1 concepts and their v2 replacements

| v1 field | Current meaning | v2 replacement | Reason |
| --- | --- | --- | --- |
| `packages` | Go provider packages imported into generated code. | `providers` | Provider packages contribute many capabilities, not only packages. |
| `packages[].replace` | Local module replacement for generated `go.mod`. | `workspace` plus `providers[].module` override | Local development should be centralized and discoverable from `go.work`. |
| `modules` | Runtime module instances selected from providers. | `runtime.modules` | Runtime modules are part of runtime composition, not provider declaration. |
| `commands.eval/run/repl/jsverbs` | Builtin command enable/name/mount flags. | `commands` entries with `type: builtin.*` | Builtin and provider commands should share one command-surface model. |
| `commandProviders` | Provider-owned command sets mounted into the root. | `commands` entries with `type: provider.command-set` | Command surfaces can depend on providers, modules, sources, and host services. |
| `jsverbs` | Source roots for JavaScript verb commands. | `sources` plus `commands[].sources` | Source declaration and command mounting are separate concerns. |
| `jsverbs[].typescript` | TypeScript compile options for a jsverb source. | `sources[].language` and `sources[].compile` | Language and compile policy should apply to any source purpose. |
| `jsverbs[].embed` | Copy source into generated embed tree. | `sources[].origin` plus `artifacts[].embed` or `sources[].compile.mode` | Embedding and compilation are artifact decisions. |
| `help.sources` | Help markdown sources. | `sources` with `purpose: help` or `artifacts` with `type: help` | Help is a source/artifact input. |
| `assets` | Static asset sources. | `sources` with `purpose: asset` and `artifacts` with `type: asset-bundle` | Assets are source sets with output policy. |
| `target` | Generated binary/package/adapter output settings. | `artifacts` | Output shape is an artifact decision. |

## Proposed v2 top-level shape

A v2 file should declare its version explicitly:

```yaml
schema: xgoja/v2
name: typescript-jsverbs
app:
  name: typescript-jsverbs
  envPrefix: TYPESCRIPT_JSVERBS
```

Then it declares Go module and workspace behavior:

```yaml
go:
  module: xgoja.generated/typescript-jsverbs
  version: "1.26"
  tags: []
  ldflags: []
  env: {}

workspace:
  mode: auto        # off | auto | path
  file: ../go.work # used when mode: path
  emit: replace     # replace | go-work
```

Then it declares providers:

```yaml
providers:
  - id: http
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/http
    register: Register
    module:
      version: v0.8.8
      workspace: auto
```

Then runtime composition:

```yaml
runtime:
  modules:
    - provider: http
      name: express
      as: express
      config: {}
```

Then source sets:

```yaml
sources:
  - id: local-sites
    purpose: jsverbs
    from:
      type: dir
      path: ./verbs
    include: ["**/*.ts"]
    exclude: ["**/*.test.ts"]
    language:
      typescript:
        target: es2015
        format: cjs
        platform: neutral
    compile:
      mode: runtime
      bundle: true
      externals:
        runtimeModules: auto
        extra: []
```

Then commands:

```yaml
commands:
  - id: run
    type: builtin.run
    name: run

  - id: verbs
    type: builtin.jsverbs
    name: verbs
    sources: [local-sites]

  - id: serve
    type: provider.command-set
    provider: http
    name: serve
    mount: serve
    sources: [local-sites]
```

Then artifacts:

```yaml
artifacts:
  - id: binary
    type: binary
    output: dist/typescript-jsverbs

  - id: declarations
    type: dts
    output: js/types/xgoja-modules.d.ts
    modules: runtime
    strict: true
```

This layout is longer than v1 for tiny examples, but it is much clearer for real systems. It also gives the planner direct objects to compile.

## Complete v2 example: TypeScript jsverbs

This is the v2 version of `examples/xgoja/15-typescript-jsverbs/xgoja.yaml`.

```yaml
schema: xgoja/v2
name: typescript-jsverbs

app:
  name: typescript-jsverbs

go:
  module: xgoja.generated/typescript-jsverbs
  version: "1.26"

workspace:
  mode: auto
  emit: replace

providers:
  - id: http
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/http
    register: Register
    module:
      version: v0.8.8
      workspace: auto

runtime:
  modules:
    - provider: http
      name: express
      as: express

sources:
  - id: local-sites
    purpose: jsverbs
    from:
      type: dir
      path: ./verbs
    include:
      - "**/*.ts"
    exclude:
      - "**/*.test.ts"
    language:
      typescript:
        target: es2015
        format: cjs
        platform: neutral
    compile:
      mode: runtime
      bundle: true
      externals:
        runtimeModules: auto

commands:
  - id: run
    type: builtin.run
    name: run

  - id: verbs
    type: builtin.jsverbs
    name: verbs
    sources:
      - local-sites

  - id: serve
    type: provider.command-set
    provider: http
    name: serve
    mount: serve
    sources:
      - local-sites

artifacts:
  - id: binary
    type: binary
    output: dist/typescript-jsverbs

  - id: declarations
    type: dts
    output: js/types/xgoja-modules.d.ts
    strict: true
```

The major improvement is that `express` does not need to be repeated as `typescript.external`. The planner can derive it from `runtime.modules` because `compile.externals.runtimeModules: auto` tells the source compiler to preserve selected runtime module aliases.

## v2 schema details

### `schema`

```yaml
schema: xgoja/v2
```

The schema field should be required. It lets xgoja select the correct parser, validator, migration behavior, and documentation links. If omitted, xgoja can treat the file as v1 during the migration period and emit a warning.

### `go`

```yaml
go:
  module: example.com/generated/app
  version: "1.26"
  tags: []
  ldflags: []
  env: {}
  imports:
    - import: github.com/mattn/go-sqlite3
      alias: _
      module: github.com/mattn/go-sqlite3
      version: v1.14.32
```

The `go` section describes generated Go module behavior. It replaces v1 `go.module`, `go.version`, `go.tags`, `go.ldflags`, `go.env`, and `go.imports` with mostly the same semantics. The difference is that module resolution is handled through the workspace/module plan instead of scattered replacement fields.

### `workspace`

```yaml
workspace:
  mode: auto
  file: ../go.work
  emit: replace
  include:
    - github.com/go-go-golems/go-go-goja
  exclude: []
```

`workspace` controls local Go module resolution. It is build-time only. It should not appear in generated runtime specs.

Recommended defaults:

- `mode: auto` for local commands run from source.
- `emit: replace` for the first implementation.
- `include: []` means apply workspace resolution to required modules that are found in the workspace.
- `exclude: []` means no module is explicitly excluded.

Release-oriented CI can use:

```yaml
workspace:
  mode: off
```

or a profile override.

### `providers`

```yaml
providers:
  - id: http
    import: github.com/go-go-golems/go-goja-http/pkg/xgoja/provider
    register: Register
    module:
      path: github.com/go-go-golems/go-goja-http
      version: v0.3.0
      workspace: auto
```

`providers` replaces v1 `packages`. The new name is more accurate. A provider can contribute runtime modules, commands, help, assets, and sources. The `module` subsection describes Go module resolution and can override inferred module path or version.

Fields:

| Field | Meaning |
| --- | --- |
| `id` | Stable provider ID used by runtime modules and commands. |
| `import` | Go import path for generated code. |
| `register` | Provider registry function. Defaults to `Register` if omitted. |
| `module.path` | Optional Go module path override. If omitted, infer from import path. |
| `module.version` | Version to require when no local workspace/replace is used. |
| `module.workspace` | `auto`, `off`, or a future explicit workspace reference. |

### `runtime.modules`

```yaml
runtime:
  modules:
    - provider: http
      name: express
      as: express
      config: {}
```

This replaces v1 `modules`. Runtime modules are selected capabilities from providers. Their aliases become valid imports for JavaScript/TypeScript source bundling.

The provider graph should validate:

- provider exists;
- module exists in provider registry;
- aliases are unique;
- config matches provider schema if validation is available;
- TypeScript descriptors exist when declaration generation runs in strict mode.

### `sources`

`sources` is the most important v2 addition. It gives the source graph its user-facing input.

```yaml
sources:
  - id: local-sites
    purpose: jsverbs
    from:
      type: dir
      path: ./verbs
    include: ["**/*.ts"]
    exclude: ["**/*.test.ts"]
    language:
      typescript:
        target: es2015
    compile:
      mode: runtime
      bundle: true
```

Supported source origins:

```yaml
from:
  type: dir
  path: ./verbs
```

```yaml
from:
  type: provider
  provider: http
  source: builtins
```

```yaml
from:
  type: workspace-dir
  module: github.com/acme/app
  path: ./verbs
```

```yaml
from:
  type: embedded
  id: local-sites
```

The first implementation only needs `dir` and `provider`. `embedded` can be an internal planned origin rather than a user-authored input. `workspace-dir` becomes useful once Go workspace resolution can map module paths to directories.

Source purposes:

| Purpose | Meaning |
| --- | --- |
| `jsverbs` | Source files scanned for jsverb command metadata and loaded into goja. |
| `script` | Direct runnable scripts or entrypoints. |
| `asset` | Static files copied or embedded as assets. |
| `help` | Markdown help files loaded into the help system. |
| `declarations` | Type declaration inputs or generated declaration artifacts. |

### `language`

```yaml
language:
  typescript:
    target: es2015
    format: cjs
    platform: neutral
    tsconfig: ./tsconfig.json
    sourcemap: none
    define: {}
```

`language` describes syntax and compiler options. It does not decide when compilation happens. That belongs to `compile`.

For plain JavaScript:

```yaml
language:
  javascript: {}
```

If omitted, xgoja can infer language from extensions, but explicit language is better for non-trivial source sets.

### `compile`

```yaml
compile:
  mode: runtime # runtime | build-time | preserve
  bundle: true
  check:
    command: ["tsc", "--noEmit"]
  externals:
    runtimeModules: auto
    extra:
      - node:crypto
```

`compile` separates policy from language.

| Field | Meaning |
| --- | --- |
| `mode` | `runtime`, `build-time`, or `preserve`. |
| `bundle` | Whether local imports should be bundled. |
| `check.command` | Optional external type-check command. |
| `externals.runtimeModules` | `auto` means selected runtime module aliases are preserved. |
| `externals.extra` | Additional explicit externals. |

This replaces the overloaded v1 `typescript.bundle` and `typescript.external` behavior.

### `commands`

Commands become a list of command surfaces.

Builtin run:

```yaml
commands:
  - id: run
    type: builtin.run
    name: run
```

Builtin jsverbs:

```yaml
commands:
  - id: verbs
    type: builtin.jsverbs
    name: verbs
    sources: [local-sites]
```

Provider command set:

```yaml
commands:
  - id: serve
    type: provider.command-set
    provider: http
    name: serve
    mount: serve
    sources: [local-sites]
    config: {}
```

This shape lets command surfaces declare dependencies explicitly. A command can depend on source sets, runtime modules, assets, host services, or provider command factories. The plan can validate these dependencies before generation.

### `artifacts`

Artifacts describe generated outputs.

```yaml
artifacts:
  - id: binary
    type: binary
    output: dist/app

  - id: runtime-package
    type: runtime-package
    package: internal/xgojaruntime
    output: ./internal/xgojaruntime

  - id: declarations
    type: dts
    output: js/types/xgoja-modules.d.ts
    modules: runtime
    strict: true

  - id: embedded-assets
    type: embed
    sources: [web-assets]
```

This replaces v1 `target` for output shape. It also gives the planner a place to represent multiple outputs from one spec.

### `profiles`

Profiles are optional but useful because v2 deliberately separates development and release behavior.

```yaml
profiles:
  dev:
    workspace:
      mode: auto
    sources:
      local-sites:
        compile:
          mode: runtime
  release:
    workspace:
      mode: off
    sources:
      local-sites:
        compile:
          mode: build-time
```

A future CLI can select profiles:

```bash
xgoja build -f xgoja.yaml --profile release
xgoja serve -f xgoja.yaml --profile dev
```

Profiles are not required for v2. They should be designed early enough that schema choices do not block them.

## Planner mapping

The planner should load v2 into a `ConfigV2` DTO, validate it, and compile it into graphs and plans.

```go
type ConfigV2 struct {
    Schema    string
    Name      string
    App       AppSpec
    Go        GoSpec
    Workspace WorkspaceSpec
    Providers []ProviderSpec
    Runtime   RuntimeSpec
    Sources   []SourceSpec
    Commands  []CommandSurfaceSpec
    Artifacts []ArtifactSpec
    Profiles  map[string]ProfileOverlay
}
```

Planner flow:

```go
func CompileV2(cfg ConfigV2, opts CompileOptions) (*Plan, error) {
    cfg = ApplyProfile(cfg, opts.Profile)
    if err := ValidateConfigV2(cfg); err != nil { return nil, err }

    goModules, err := workspace.Resolve(cfg.Go, cfg.Workspace, cfg.Providers)
    if err != nil { return nil, err }

    providers, err := ResolveProviderGraph(cfg.Providers, cfg.Runtime.Modules, goModules)
    if err != nil { return nil, err }

    sourceSets, err := ResolveSourceSets(cfg.Sources, providers, goModules)
    if err != nil { return nil, err }

    graph, err := sourcegraph.Build(sourceSets)
    if err != nil { return nil, err }

    resolver := sourcegraph.NewResolver(graph, providers.RuntimeModuleAliases(), cfg.ExternalPolicy())
    if err := resolver.ResolveAll(); err != nil { return nil, err }

    commands, err := ResolveCommandSurfaces(cfg.Commands, providers, graph)
    if err != nil { return nil, err }

    artifacts, err := ResolveArtifacts(cfg.Artifacts, cfg, providers, graph, commands)
    if err != nil { return nil, err }

    return &Plan{GoModules: goModules, Providers: providers, SourceGraph: graph, Commands: commands, Artifacts: artifacts}, nil
}
```

The key point is that v2 config maps directly to planner inputs. There is no need to infer source sets from `jsverbs`, command surfaces from `commands` plus `commandProviders`, or workspace behavior from scattered replace fields.

## Migration tooling

The migration tool is what makes dropping long-term backwards compatibility practical.

### CLI commands

Recommended commands:

```bash
xgoja migrate-spec -f xgoja.yaml --out xgoja.v2.yaml
xgoja migrate-spec -f xgoja.yaml --in-place --backup
xgoja migrate-spec -f xgoja.yaml --check
```

Options:

| Option | Meaning |
| --- | --- |
| `--from v1` | Explicit source schema. Defaults to auto-detect. |
| `--to v2` | Target schema. Defaults to latest. |
| `--profile dev` | Emit dev-oriented defaults such as workspace auto and runtime compilation. |
| `--profile release` | Emit release-oriented defaults such as workspace off and build-time compilation where safe. |
| `--in-place` | Replace input file. |
| `--backup` | Write `xgoja.yaml.bak` before in-place migration. |
| `--check` | Report whether migration would change the file. |
| `--format` | Reformat migrated YAML without semantic changes. |

### Migration mapping

Pseudocode:

```go
func MigrateV1ToV2(v1 buildspec.BuildSpec) (ConfigV2, []MigrationWarning) {
    v2 := ConfigV2{
        Schema: "xgoja/v2",
        Name:   v1.Name,
        App: AppSpec{Name: firstNonEmpty(v1.AppName, v1.Name), EnvPrefix: v1.EnvPrefix},
        Go: migrateGo(v1.Go),
        Providers: migratePackages(v1.Packages),
        Runtime: RuntimeSpec{Modules: migrateModules(v1.Modules)},
        Sources: migrateSources(v1.JSVerbs, v1.Help, v1.Assets),
        Commands: migrateCommands(v1.Commands, v1.CommandProviders, v1.JSVerbs),
        Artifacts: migrateTargetAndGeneratedOutputs(v1.Target),
    }
    return v2, warnings
}
```

Mapping details:

```text
v1 packages[]           -> v2 providers[]
v1 packages[].replace   -> v2 providers[].module.replace or workspace override warning
v1 modules[]            -> v2 runtime.modules[]
v1 jsverbs[]            -> v2 sources[purpose=jsverbs][]
v1 commands.jsverbs     -> v2 commands[type=builtin.jsverbs]
v1 commandProviders[]   -> v2 commands[type=provider.command-set][]
v1 help.sources[]       -> v2 sources[purpose=help][] or artifacts[type=help][]
v1 assets[]             -> v2 sources[purpose=asset][] and artifacts[type=asset-bundle][]
v1 target               -> v2 artifacts[]
v1 go.imports[]         -> v2 go.imports[]
```

### Migration warnings

The migration tool should warn, not silently guess, when v1 does not map cleanly.

Examples:

```text
warning: jsverbs[local].embed=true migrated to source compile.mode=runtime and artifact embedding; review whether release profile should use build-time compilation
```

```text
warning: packages[http].replace migrated as explicit provider module replacement; consider workspace.mode=auto if this path comes from go.work
```

```text
warning: commandProviders[serve].modules is set; v2 command surfaces should express runtime module dependencies explicitly
```

```text
warning: target.kind=adapter migrated to artifact type adapter; verify target root function and host attachment behavior
```

### Refactoring tooling beyond YAML

Some migrations may need source or Makefile updates. Provide optional tooling:

```bash
xgoja migrate-examples --dir examples/xgoja --write
xgoja migrate-docs --rewrite-command-examples
```

The first useful refactoring tool is probably not source rewriting. It is workspace replacement cleanup:

```bash
xgoja workspace suggest -f xgoja.yaml
```

Output:

```text
packages[http].replace points to ../go-go-goja-provider-http
This module is present in ../go.work as github.com/acme/go-go-goja-provider-http.
Suggested v2 replacement: remove provider module replace and use workspace.mode=auto.
```

## Compatibility policy

The project does not need indefinite backwards compatibility. The policy can be:

1. v1 remains loadable only for `migrate-spec` and perhaps for one transitional release.
2. v2 becomes the planner's native schema.
3. Docs and examples move to v2 first.
4. Runtime internals do not carry v1 compatibility branches; v1 is converted to v2 before planning.
5. After downstream packages migrate, v1 execution support can be removed.

This keeps the architecture clean. Compatibility exists at the CLI boundary through migration tooling, not throughout the planner and runtime implementation.

## Decision records

### Decision: Make v2 the native planner schema

- **Context:** The source graph architecture is cleaner than the v1 spec. Preserving v1 as the native shape would force repeated compatibility inference.
- **Options considered:** Keep v1 indefinitely; build planner around v1 and add hidden graph concepts; introduce v2 and migrate v1 at the boundary.
- **Decision:** Define v2 as the native planner schema. Treat v1 as legacy input that is converted by migration tooling.
- **Rationale:** v2 aligns user-facing configuration with provider graph, source graph, workspace resolution, command surfaces, and artifacts.
- **Consequences:** Existing users need migration, but migration can be automated and documented. Internals become simpler and easier to extend.
- **Status:** proposed

### Decision: Use command surfaces instead of separate builtin/provider command sections

- **Context:** v1 separates builtin command toggles from provider command sets, but both become mounted commands on the generated root.
- **Options considered:** Keep separate sections; unify commands into a list; infer command surfaces from sources.
- **Decision:** Use a top-level `commands` list with `type` values such as `builtin.run`, `builtin.jsverbs`, and `provider.command-set`.
- **Rationale:** Commands can declare dependencies on providers, modules, sources, and host services in one shape.
- **Consequences:** v2 command config is more explicit. Migration from v1 is mechanical.
- **Status:** proposed

### Decision: Use `sources` for jsverbs, help, assets, and future source kinds

- **Context:** v1 has separate `jsverbs`, `help.sources`, and `assets` sections, each with similar origin/embed/provider concepts.
- **Options considered:** Keep separate sections; unify all inputs under `sources`; use artifacts only.
- **Decision:** Use `sources` for source-set declaration and `artifacts` for output policy.
- **Rationale:** Source origin, include/exclude, language, and compile policy are reusable concepts.
- **Consequences:** Some simple asset/help examples become more verbose, but graph planning becomes consistent.
- **Status:** proposed

### Decision: Provide migration tooling instead of permanent internal compatibility

- **Context:** The user explicitly stated that backwards compatibility is not required if migration documentation and tooling are good.
- **Options considered:** Maintain v1 and v2 forever; remove v1 immediately; provide migration tooling and retire v1 after packages migrate.
- **Decision:** Provide `xgoja migrate-spec`, migration documentation, and a transitional period. Keep the planner v2-native.
- **Rationale:** This protects engineering velocity while giving users a safe path.
- **Consequences:** Migration tooling becomes part of the release criteria for v2.
- **Status:** proposed

## Implementation plan

### Phase 1: Define v2 DTOs and validator

Add a new internal package:

```text
cmd/xgoja/internal/specv2/
  types.go
  load.go
  validate.go
  defaults.go
  migrate_v1.go
  render.go
```

Do not replace v1 loading yet. Add tests for parsing, defaulting, validation, and rendering.

### Phase 2: Implement v1-to-v2 migration

Implement `MigrateV1ToV2`. It should use current `buildspec.BuildSpec` as input and output `specv2.Config` plus warnings.

Add golden tests:

```text
testdata/migrate/simple-v1.yaml
testdata/migrate/simple-v2.golden.yaml
testdata/migrate/typescript-jsverbs-v1.yaml
testdata/migrate/typescript-jsverbs-v2.golden.yaml
```

### Phase 3: Add `xgoja migrate-spec`

Add the CLI command with `--out`, `--in-place`, `--backup`, and `--check`.

The command should print warnings and a short next-step message:

```text
migrated xgoja.yaml -> xgoja.v2.yaml
warnings: 2
run: xgoja doctor -f xgoja.v2.yaml
```

### Phase 4: Build planner from v2

Implement planner APIs around `specv2.Config`, not v1. During transition, v1 command paths can convert to v2 before calling the planner.

```go
func CompileV2(cfg specv2.Config, opts CompileOptions) (*plan.Plan, error)
```

### Phase 5: Update examples and docs

Migrate the examples that matter first:

1. TypeScript jsverbs example.
2. HTTP serve jsverbs example.
3. Generated runtime package example.
4. Provider examples.

Add migration documentation:

```text
cmd/xgoja/doc/16-migrating-to-xgoja-v2.md
cmd/xgoja/doc/17-xgoja-v2-reference.md
```

### Phase 6: Switch default docs and generated examples to v2

Once examples pass, docs should teach v2 as the primary format. v1 docs can remain under migration/reference until removed.

### Phase 7: Retire v1 runtime support

After downstream packages migrate, remove v1 execution support. Keep `migrate-spec` able to read v1 for longer if maintaining the parser is cheap.

## Testing strategy

### Schema tests

- Parse minimal v2 spec.
- Parse full TypeScript jsverbs v2 spec.
- Reject duplicate provider IDs.
- Reject duplicate runtime module aliases.
- Reject command references to missing sources.
- Reject source references to missing providers.
- Reject artifact references to missing source IDs.

### Migration tests

- v1 packages become v2 providers.
- v1 modules become v2 runtime modules.
- v1 jsverbs become v2 sources and builtin jsverbs command dependencies.
- v1 command providers become v2 provider command surfaces.
- v1 TypeScript externals become `compile.externals.extra`, with warning if they duplicate runtime modules.
- v1 replace paths become provider module local overrides or workspace migration warnings.

### Planner tests

- v2 TypeScript jsverbs compile to the same runtime behavior as v1.
- v2 workspace auto can derive generated replaces from `go.work`.
- v2 declaration artifact uses runtime module aliases.
- v2 command surfaces mount expected Cobra commands.

### End-to-end tests

- `xgoja migrate-spec` on existing examples.
- `xgoja doctor` on migrated examples.
- `xgoja build` on migrated examples.
- `make -C examples/xgoja/15-typescript-jsverbs smoke` after migrating the example.

## Documentation plan

Migration documentation should include:

1. A high-level explanation of why v2 exists.
2. A field mapping table from v1 to v2.
3. Before/after examples.
4. `xgoja migrate-spec` usage.
5. Workspace migration guidance.
6. Common warnings and how to resolve them.
7. Release validation guidance with workspace disabled.

A concise migration page is more important than preserving v1 behavior indefinitely. The documentation should be explicit that v2 is the native architecture going forward.

## Near-term recommendation

Update the main architecture roadmap:

1. Finish `XGOJA-TS-002` because it fixes a real review issue and teaches the system how to preserve `fs.FS` source origin metadata.
2. Implement `specv2` DTOs and `migrate-spec` before building too much planner logic.
3. Build the source graph and provider graph against v2, not v1.
4. Migrate examples to v2 as soon as planner coverage is sufficient.
5. Keep v1 as migration input, not as a permanent internal architecture.

## References

- `cmd/xgoja/internal/buildspec/build_spec.go` — current v1 schema.
- `pkg/xgoja/app/runtime_spec.go` — current generated runtime DTO.
- `pkg/xgoja/providerapi/module.go` — provider module capabilities.
- `pkg/xgoja/providerapi/commands.go` — provider command sets and JSVerbSourceSet.
- `pkg/xgoja/providerapi/verbs.go` — provider-shipped source roots.
- `pkg/jsverbs/scan.go` — current source scanning behavior.
- `pkg/tsscript/compiler.go` — current TypeScript compiler backend.
- `cmd/xgoja/internal/generate/gomod.go` — current generated Go module rendering.
- `cmd/xgoja/doc/02-user-guide.md` — current v1 user guide that needs migration/update.
- `examples/xgoja/15-typescript-jsverbs/xgoja.yaml` — concrete v1 TypeScript example to migrate.
- `ttmp/2026/06/12/XGOJA-ARCH-001--xgoja-source-graph-and-bundler-architecture/design/01-xgoja-source-graph-and-bundler-architecture.md` — parent source graph architecture.
