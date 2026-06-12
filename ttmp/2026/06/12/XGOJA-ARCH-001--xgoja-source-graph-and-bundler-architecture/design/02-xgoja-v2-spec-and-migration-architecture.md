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
    - Path: go-go-goja/cmd/xgoja/internal/buildspec/build_spec.go
      Note: Current v1 xgoja.yaml schema that should become migratable legacy input.
    - Path: go-go-goja/pkg/xgoja/app/runtime_spec.go
      Note: Current runtime spec that v2 planning should replace or generate from a cleaner model.
    - Path: go-go-goja/pkg/xgoja/providerapi/module.go
      Note: Provider modules become first-class provider graph capabilities in v2.
    - Path: go-go-goja/pkg/xgoja/providerapi/commands.go
      Note: Command sets and JSVerbSourceSet map to explicit v2 command surfaces and source dependencies.
    - Path: go-go-goja/pkg/xgoja/providerapi/verbs.go
      Note: Provider-shipped verb sources become v2 source origins.
    - Path: go-go-goja/pkg/jsverbs/scan.go
      Note: Current jsverb source scanning should migrate behind v2 source graph adapters.
    - Path: go-go-goja/pkg/tsscript/compiler.go
      Note: Current TypeScript compiler facade should become an internal compiler backend for goja-executed source.
    - Path: go-go-goja/cmd/xgoja/internal/generate/gomod.go
      Note: Current generated go.mod rendering should consume v2 workspace and GoModulePlan data.
    - Path: go-go-goja/cmd/xgoja/doc/02-user-guide.md
      Note: Existing user-facing v1 docs that need migration documentation.
    - Path: go-go-goja/examples/xgoja/15-typescript-jsverbs/xgoja.yaml
      Note: Recent v1 TypeScript jsverbs example used as a concrete migration target.
ExternalSources:
    - local:01-architecture-reassessment-prompt.md
Summary: A simplified v2 xgoja.yaml architecture that aligns the user-facing spec with providers, runtime modules, goja-executed sources, command surfaces, artifacts, Go workspace resolution, and migration tooling.
LastUpdated: 2026-06-12T12:35:00-04:00
WhatFor: Use when designing the next xgoja config format, migration tooling, and planner implementation without preserving v1 as the long-term internal architecture.
WhenToUse: Before implementing sourcegraph/provider-graph planning, workspace resolution, command surfaces, build artifacts, or migration tooling for existing xgoja users.
---

# XGoja v2 spec and migration architecture

## Executive summary

This document defines a simplified v2 `xgoja.yaml` architecture. The key simplification is that xgoja only compiles or bundles source that will execute inside the xgoja/goja runtime. Browser applications, frontend bundles, worker bundles, and other non-goja JavaScript artifacts should be built by their own tooling and then included in xgoja as static assets.

That constraint removes a large amount of configuration surface. The v2 spec does not need user-facing fields for esbuild engine selection, browser-vs-node bundler platform, output format, JavaScript target, package manager, install policy, polyfills, CSS loaders, SVG loaders, or frontend build caches. xgoja can own the compiler defaults for goja-executed code. Users should describe intent: which providers are available, which Go-backed runtime modules are selected, which source sets are goja-executed, which commands are exposed, which assets are embedded, and how local Go modules are resolved during development.

The v2 spec should therefore expose these concepts directly:

- `providers` describes Go packages that contribute xgoja capabilities.
- `workspace` describes local Go workspace resolution for provider development.
- `runtime.modules` selects Go-backed JavaScript modules by provider and alias.
- `sources` describes goja-executed source sets and asset/help source sets.
- `commands` describes user-facing command surfaces and their source/module dependencies.
- `artifacts` describes generated outputs such as binaries, declaration files, runtime packages, and embedded assets.

The migration strategy remains the same: v1 is legacy input, v2 is the native planner schema. xgoja should provide `xgoja migrate-spec` to convert existing v1 files into v2 files. Packages that use xgoja can migrate when they need v2 capabilities.

## Core design rule

The central rule is:

> If code runs in goja, xgoja may compile or bundle it. If code runs somewhere else, build it outside xgoja and let xgoja embed or serve the output.

This keeps xgoja opinionated and simple. It also matches the actual runtime boundary. The runtime is goja plus Go-backed CommonJS modules registered by provider packages. xgoja should optimize that path rather than becoming a general frontend bundler.

A Redux-like package can still be supported later if it is bundled into goja-executed source. For example, a TypeScript script that imports `redux` and then runs inside goja is in scope. A React browser application that uses Redux is out of scope for xgoja compilation; it should be built by Vite, esbuild, pnpm, Bun, or another external tool and then embedded as assets.

## Why this replaces the broader v2 draft

An earlier v2 shape included generic bundle fields such as package manager, platform, format, target, and loaders. Those fields are useful for a general JavaScript bundler, but xgoja does not need to be that tool. In practice, the execution engine for xgoja-managed code is stable:

- Runtime code executes in goja.
- Runtime modules are Go-backed CommonJS modules selected from providers.
- Source-level TypeScript or ESM import syntax is an authoring convenience compiled before execution.
- The safe output profile is owned by xgoja.
- Runtime-module aliases should be externalized automatically.

Users should not configure `engine: esbuild`, `platform: goja-neutral`, `target: es2018`, or `format: cjs` in ordinary specs. Those are implementation defaults for the xgoja runtime profile. If a real need appears later, advanced override fields can be added behind an explicit escape hatch. They should not shape the v2 MVP.

## Design goals

The simplified v2 spec should satisfy these goals:

1. **Expose intent, not compiler mechanics.** Users declare source kind, origin, language, compile timing, and bundling intent. xgoja chooses compiler backend, output format, platform, and target for the goja runtime.
2. **Keep non-goja bundling outside xgoja.** Browser and frontend outputs are external build artifacts. xgoja embeds or serves their output directories as assets.
3. **Separate provider packages from runtime modules.** A Go provider package can contribute many capabilities. Selecting one JavaScript module from that provider is a runtime decision.
4. **Make source sets first-class.** jsverbs, scripts, assets, and help all have origins. Only goja-executed source participates in xgoja compilation/bundling.
5. **Make command surfaces explicit.** Builtin commands and provider command sets use one command model.
6. **Make Go workspace resolution explicit.** Local provider development should work from `go.work` without repeated manual replacements.
7. **Support migration over compatibility.** v1 should be migratable legacy input, not the permanent internal shape.

## Non-goals

- v2 does not make xgoja a general browser bundler.
- v2 does not invoke package managers in the first version.
- v2 does not install npm packages.
- v2 does not expose esbuild platform/format/target settings as ordinary fields.
- v2 does not provide browser or Node polyfills.
- v2 does not require goja to execute ECMAScript modules directly.
- v2 does not keep v1 as an equal long-term config format.

## Current v1 concepts and their v2 replacements

| v1 field | Current meaning | v2 replacement | Reason |
| --- | --- | --- | --- |
| `packages` | Go provider packages imported into generated code. | `providers` | Provider packages contribute modules, commands, sources, assets, help, and declarations. |
| `packages[].replace` | Local Go module replacement. | `workspace` or provider module override | Local development should be centralized and discoverable from `go.work`. |
| `modules` | Runtime module instances selected from providers. | `runtime.modules` | Runtime module selection belongs to runtime composition. |
| `commands.eval/run/repl/jsverbs` | Builtin command enable/name/mount flags. | `commands` entries with `type: builtin.*` | Builtin and provider commands are both command surfaces. |
| `commandProviders` | Provider-owned command sets mounted into the root. | `commands` entries with `type: provider.command-set` | Provider commands can declare dependencies on sources and modules. |
| `jsverbs` | Source roots for JavaScript verb commands. | `sources` plus `commands[].sources` | Source declaration and command mounting are separate concerns. |
| `jsverbs[].typescript` | TypeScript compile options. | `language: typescript` plus `compile` | Language and compile policy should be concise and runtime-profiled. |
| `jsverbs[].embed` | Copy source into generated embed tree. | `compile.mode` and/or artifact embedding | Embedding and compilation are artifact/planning decisions. |
| `help.sources` | Help markdown sources. | `sources` with `kind: help` | Help files are source inputs, not runtime JavaScript. |
| `assets` | Static asset sources. | `sources` with `kind: assets` plus `artifacts` | Assets are external artifacts or static source trees. |
| `target` | Generated binary/package output settings. | `artifacts` | Output shape is an artifact decision. |

## Proposed simplified v2 shape

A v2 spec declares the schema explicitly:

```yaml
schema: xgoja/v2
name: typescript-jsverbs
```

Application identity is optional but explicit:

```yaml
app:
  name: typescript-jsverbs
  envPrefix: TYPESCRIPT_JSVERBS
```

Generated Go module behavior:

```yaml
go:
  module: xgoja.generated/typescript-jsverbs
  version: "1.26"
```

Local Go workspace behavior:

```yaml
workspace:
  mode: auto # off | auto | path
```

Provider packages:

```yaml
providers:
  - id: http
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/http
```

Runtime modules:

```yaml
runtime:
  modules:
    - provider: http
      name: express
      as: express
```

Source sets:

```yaml
sources:
  - id: sites
    kind: jsverbs
    from:
      dir: ./verbs
    language: typescript
    compile:
      mode: runtime
      bundle: true
```

Command surfaces:

```yaml
commands:
  - id: run
    type: builtin.run

  - id: verbs
    type: builtin.jsverbs
    sources: [sites]

  - id: serve
    type: provider.command-set
    provider: http
    name: serve
    mount: serve
    sources: [sites]
```

Generated artifacts:

```yaml
artifacts:
  - id: binary
    type: binary
    output: dist/typescript-jsverbs

  - id: declarations
    type: dts
    output: js/types/xgoja-modules.d.ts
    strict: true
```

This is the normal v2 shape. It has no low-level bundler settings.

## Complete v2 example: TypeScript jsverbs

This is the simplified v2 version of `examples/xgoja/15-typescript-jsverbs/xgoja.yaml`.

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

providers:
  - id: http
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/http

runtime:
  modules:
    - provider: http
      name: express
      as: express

sources:
  - id: sites
    kind: jsverbs
    from:
      dir: ./verbs
    include:
      - "**/*.ts"
    exclude:
      - "**/*.test.ts"
    language: typescript
    compile:
      mode: runtime
      bundle: true

commands:
  - id: run
    type: builtin.run

  - id: verbs
    type: builtin.jsverbs
    sources:
      - sites

  - id: serve
    type: provider.command-set
    provider: http
    name: serve
    mount: serve
    sources:
      - sites

artifacts:
  - id: binary
    type: binary
    output: dist/typescript-jsverbs

  - id: declarations
    type: dts
    output: js/types/xgoja-modules.d.ts
    strict: true
```

The planner derives these internal decisions:

- `sites` is a goja-executed jsverb source set.
- TypeScript is compiled with xgoja's goja runtime compiler profile.
- `bundle: true` follows local imports.
- Selected runtime module aliases, including `express`, are preserved as runtime imports.
- The generated binary imports the `http` provider package.
- Workspace mode can derive local Go module replacements from `go.work`.
- Declaration generation uses the selected runtime module aliases.

The user does not repeat `external: [express]`. The provider graph and runtime module plan already know that `express` is a runtime module alias.

## External frontend bundles as assets

If a project has a frontend that uses Redux, React, CSS, SVG loaders, browser polyfills, and a package manager, that build stays outside xgoja. The v2 spec sees only the output directory.

External build:

```bash
cd web
pnpm install
pnpm build
```

v2 xgoja spec:

```yaml
sources:
  - id: web-dist
    kind: assets
    from:
      dir: ./web/dist

artifacts:
  - id: web-assets
    type: embedded-assets
    sources: [web-dist]
```

A provider command such as HTTP serve can then use the embedded assets through existing asset resolver/provider mechanisms. xgoja does not need to know how `web/dist` was built.

This rule keeps frontend concerns out of xgoja while still allowing xgoja to package the final files into generated binaries.

## Future package bundling for goja-executed code

The simplified v2 spec still allows future package bundling for goja-executed code. The schema does not need package-manager fields now because package resolution can be inferred from source directories when implemented later.

Future example:

```ts
import { createStore } from "redux"
import { message } from "./message"

const store = createStore(reducer)
```

The same v2 source can remain:

```yaml
sources:
  - id: state-script
    kind: script
    from:
      dir: ./scripts
    language: typescript
    compile:
      mode: build-time
      bundle: true
```

Future resolver behavior:

```text
./message -> local source dependency
redux     -> package dependency bundled into goja-executed artifact
express   -> runtime module alias preserved for goja require()
```

This can be added later without changing the basic v2 fields. If package-root discovery becomes ambiguous, xgoja can add one optional field at that time:

```yaml
packageRoot: ./js
```

Do not add it until the implementation needs it. The current schema can reserve the semantic meaning of `bundle: true`: bundle dependencies for goja-executed source. Initially that means local dependencies; later it can include package dependencies.

## v2 schema details

### `schema`

```yaml
schema: xgoja/v2
```

The schema field is required. If omitted during the migration period, xgoja can assume v1 and print a warning.

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

The `go` section describes generated Go module behavior. It keeps the useful v1 fields but removes the need to scatter replacement information across provider entries when workspace discovery can do it.

### `workspace`

```yaml
workspace:
  mode: auto
  file: ../go.work
```

`workspace` is build-time only. It does not enter generated runtime specs.

Supported modes:

| Mode | Meaning |
| --- | --- |
| `off` | Ignore `go.work`; use explicit versions/replacements only. |
| `auto` | Search upward from the spec directory for `go.work`. |
| `path` | Use `workspace.file` exactly. |

The first implementation should derive generated `replace` directives from workspace modules. A generated temporary `go.work` mode can be added later if needed.

### `providers`

```yaml
providers:
  - id: http
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/http
    register: Register
    module:
      version: v0.8.8
```

`register` defaults to `Register`. `module.version` is used when no workspace/local replacement applies. The provider module path can be inferred from the import path using the current `providerModulePath` behavior, with an override added only if needed.

### `runtime.modules`

```yaml
runtime:
  modules:
    - provider: http
      name: express
      as: express
      config: {}
```

Runtime module aliases become valid imports for goja-executed source. The source compiler should preserve them automatically when bundling.

### `sources`

A source set declares origin, kind, language, filters, and compile policy.

```yaml
sources:
  - id: sites
    kind: jsverbs
    from:
      dir: ./verbs
    include: ["**/*.ts"]
    exclude: ["**/*.test.ts"]
    language: typescript
    compile:
      mode: runtime
      bundle: true
```

Supported `kind` values for the MVP:

| Kind | Meaning | xgoja compiles? |
| --- | --- | --- |
| `jsverbs` | goja-executed command source scanned by jsverbs. | Yes, if `language` requires it. |
| `script` | goja-executed script source. | Yes, if used by an artifact/command. |
| `assets` | static files to embed or serve. | No. |
| `help` | markdown help files. | No. |

Supported `from` values for the MVP:

```yaml
from:
  dir: ./verbs
```

```yaml
from:
  provider:
    provider: http
    source: builtins
```

Workspace-dir origins can be added later after Go workspace resolution exists:

```yaml
from:
  workspace:
    module: github.com/acme/app
    path: ./verbs
```

### `language`

Use a string for the common case:

```yaml
language: javascript
language: typescript
```

If advanced TypeScript type-checking is needed, it belongs under `compile.check`, not a large language block:

```yaml
compile:
  check:
    command: ["tsc", "--noEmit"]
```

Do not expose TypeScript target/format/platform in the normal schema. xgoja owns those defaults for goja-executed code.

### `compile`

```yaml
compile:
  mode: runtime # runtime | build-time | preserve
  bundle: true
  check:
    command: ["tsc", "--noEmit"]
```

Fields:

| Field | Meaning |
| --- | --- |
| `mode` | `runtime`, `build-time`, or `preserve`. |
| `bundle` | Whether dependencies should be bundled for goja-executed source. |
| `check.command` | Optional external validation command. |

There is no ordinary `externals` field in the MVP. Runtime module aliases are external automatically. Extra externals can be added later if there is a concrete need.

### `commands`

Commands are explicit command surfaces.

```yaml
commands:
  - id: run
    type: builtin.run

  - id: verbs
    type: builtin.jsverbs
    sources: [sites]

  - id: serve
    type: provider.command-set
    provider: http
    name: serve
    mount: serve
    sources: [sites]
```

This replaces separate v1 builtin command toggles and `commandProviders` entries.

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
    strict: true

  - id: web-assets
    type: embedded-assets
    sources: [web-dist]
```

Do not add generic `js-bundle` artifact support for browser or node bundles. If xgoja needs a build-time goja script bundle later, it can be represented as a goja-executed `script` source with `compile.mode: build-time` and a binary/runtime artifact that consumes it.

### `profiles`

Profiles are optional and can be deferred. They are useful for workspace and compile mode differences:

```yaml
profiles:
  dev:
    workspace:
      mode: auto
  release:
    workspace:
      mode: off
```

Do not block v2 MVP on profiles.

## Planner mapping

The planner should load v2 into a concise DTO:

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

    resolver := sourcegraph.NewResolver(graph, providers.RuntimeModuleAliases())
    if err := resolver.ResolveAll(); err != nil { return nil, err }

    commands, err := ResolveCommandSurfaces(cfg.Commands, providers, graph)
    if err != nil { return nil, err }

    artifacts, err := ResolveArtifacts(cfg.Artifacts, cfg, providers, graph, commands)
    if err != nil { return nil, err }

    return &Plan{GoModules: goModules, Providers: providers, SourceGraph: graph, Commands: commands, Artifacts: artifacts}, nil
}
```

The planner should apply xgoja's compiler profile internally. The spec does not need to carry esbuild settings.

## Migration tooling

The migration tool makes strict backwards compatibility unnecessary.

### CLI commands

```bash
xgoja migrate-spec -f xgoja.yaml --out xgoja.v2.yaml
xgoja migrate-spec -f xgoja.yaml --in-place --backup
xgoja migrate-spec -f xgoja.yaml --check
```

### Migration mapping

```text
v1 packages[]           -> v2 providers[]
v1 packages[].replace   -> v2 provider module local override or workspace warning
v1 modules[]            -> v2 runtime.modules[]
v1 jsverbs[]            -> v2 sources[kind=jsverbs][]
v1 commands.jsverbs     -> v2 commands[type=builtin.jsverbs]
v1 commandProviders[]   -> v2 commands[type=provider.command-set][]
v1 help.sources[]       -> v2 sources[kind=help][]
v1 assets[]             -> v2 sources[kind=assets][] and artifacts[type=embedded-assets][]
v1 target               -> v2 artifacts[]
v1 go.imports[]         -> v2 go.imports[]
```

### Migration warnings

The migration tool should emit practical warnings:

```text
warning: packages[http].replace migrated as a local provider module override; if this path is in go.work, prefer workspace.mode=auto
```

```text
warning: jsverbs[local].typescript.external included runtime module alias "express"; v2 derives runtime module externals automatically
```

```text
warning: jsverbs[local].embed=true migrated to source compile.mode=runtime; review whether release builds should use build-time compilation
```

## Compatibility policy

The project does not need indefinite backwards compatibility.

Recommended policy:

1. v1 remains loadable for `migrate-spec`.
2. v2 becomes the native planner schema.
3. Docs and examples move to v2.
4. Runtime internals do not carry v1 compatibility branches; v1 is converted to v2 before planning.
5. After downstream packages migrate, v1 execution support can be removed.

Compatibility exists at the spec boundary through migration tooling, not throughout the planner and runtime implementation.

## Decision records

### Decision: Make v2 the native planner schema

- **Context:** The source graph architecture is cleaner than the v1 spec. Preserving v1 as the native shape would force repeated compatibility inference.
- **Options considered:** Keep v1 indefinitely; build planner around v1 and add hidden graph concepts; introduce v2 and migrate v1 at the boundary.
- **Decision:** Define v2 as the native planner schema. Treat v1 as legacy input converted by migration tooling.
- **Rationale:** v2 aligns user-facing configuration with provider graph, source graph, workspace resolution, command surfaces, and artifacts.
- **Consequences:** Existing users need migration, but migration can be automated and documented. Internals become simpler and easier to extend.
- **Status:** proposed

### Decision: Keep xgoja bundling scoped to goja-executed source

- **Context:** A broader v2 design could include browser bundling, package managers, loaders, and polyfills.
- **Options considered:** Make xgoja a general JS bundler; expose esbuild options in v2; keep xgoja focused on goja-executed source and embed external bundle outputs as assets.
- **Decision:** xgoja compiles/bundles only source that runs in goja. Non-goja bundles are built externally and included as assets.
- **Rationale:** The execution target is stable and xgoja's value is Go-backed JavaScript runtime composition, not frontend build orchestration.
- **Consequences:** The v2 schema is much smaller. Future goja-runtime package bundling remains possible under `compile.bundle: true`.
- **Status:** proposed

### Decision: Hide compiler backend settings in normal config

- **Context:** Fields such as `engine`, `platform`, `target`, and `format` are implementation details for goja-executed source.
- **Options considered:** Expose all esbuild settings; expose an advanced escape hatch only; hide them entirely in the MVP.
- **Decision:** Hide them in the MVP. xgoja owns the goja runtime compiler profile.
- **Rationale:** Users should express intent. Runtime code always targets xgoja/goja, so ordinary specs do not need backend settings.
- **Consequences:** Fewer knobs. If a real need appears, an explicit advanced section can be added later.
- **Status:** proposed

### Decision: Use command surfaces instead of separate builtin/provider command sections

- **Context:** v1 separates builtin command toggles from provider command sets, but both become mounted commands on the generated root.
- **Options considered:** Keep separate sections; unify commands into a list; infer command surfaces from sources.
- **Decision:** Use a top-level `commands` list with `type` values such as `builtin.run`, `builtin.jsverbs`, and `provider.command-set`.
- **Rationale:** Commands can declare dependencies on providers, modules, sources, and host services in one shape.
- **Consequences:** v2 command config is more explicit. Migration from v1 is mechanical.
- **Status:** proposed

### Decision: Provide migration tooling instead of permanent internal compatibility

- **Context:** Backwards compatibility is not required if migration documentation and tooling are good.
- **Options considered:** Maintain v1 and v2 forever; remove v1 immediately; provide migration tooling and retire v1 after packages migrate.
- **Decision:** Provide `xgoja migrate-spec`, migration documentation, and a transitional period. Keep the planner v2-native.
- **Rationale:** This protects engineering velocity while giving users a safe path.
- **Consequences:** Migration tooling becomes part of the release criteria for v2.
- **Status:** proposed

## Implementation plan

### Phase 1: Define simplified v2 DTOs and validator

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

The DTO should not include broad bundler fields. It should include concise `language` and `compile` fields for goja-executed source.

### Phase 2: Implement v1-to-v2 migration

Implement `MigrateV1ToV2`. Add golden tests for simple specs and the TypeScript jsverbs example.

### Phase 3: Add `xgoja migrate-spec`

Add the CLI command with `--out`, `--in-place`, `--backup`, and `--check`.

### Phase 4: Build planner from v2

Implement planner APIs around `specv2.Config`, not v1. During transition, v1 command paths can convert to v2 before calling the planner.

### Phase 5: Update examples and docs

Migrate examples in this order:

1. TypeScript jsverbs example.
2. HTTP serve jsverbs example.
3. Generated runtime package example.
4. Provider examples.

Add:

```text
cmd/xgoja/doc/16-migrating-to-xgoja-v2.md
cmd/xgoja/doc/17-xgoja-v2-reference.md
```

### Phase 6: Retire v1 runtime support after migration

Keep v1 readable by `migrate-spec` if cheap, but do not keep v1 as an internal planner shape.

## Testing strategy

### Schema tests

- Parse minimal v2 spec.
- Parse simplified TypeScript jsverbs v2 spec.
- Reject duplicate provider IDs.
- Reject duplicate runtime module aliases.
- Reject command references to missing sources.
- Reject source references to missing providers.
- Reject artifact references to missing source IDs.
- Reject unsupported broad bundler fields in MVP, with clear diagnostics.

### Migration tests

- v1 packages become v2 providers.
- v1 modules become v2 runtime modules.
- v1 jsverbs become v2 sources and builtin jsverbs command dependencies.
- v1 command providers become v2 provider command surfaces.
- v1 TypeScript externals that duplicate runtime modules are removed or warned about.
- v1 assets become v2 asset sources and embedded asset artifacts.

### Planner tests

- v2 TypeScript jsverbs compile to the same runtime behavior as v1.
- v2 workspace auto derives generated replacements from `go.work`.
- v2 declaration artifact uses runtime module aliases.
- v2 command surfaces mount expected Cobra commands.
- External frontend build outputs are embedded as assets without xgoja attempting to compile them.

### End-to-end tests

- `xgoja migrate-spec` on existing examples.
- `xgoja doctor` on migrated examples.
- `xgoja build` on migrated examples.
- `make -C examples/xgoja/15-typescript-jsverbs smoke` after migrating the example.

## Documentation plan

Migration documentation should include:

1. Why v2 exists.
2. The core rule: xgoja only compiles goja-executed source.
3. A field mapping table from v1 to v2.
4. Before/after examples.
5. `xgoja migrate-spec` usage.
6. Workspace migration guidance.
7. How to embed externally-built frontend assets.
8. Common warnings and how to resolve them.
9. Release validation guidance with workspace disabled.

## Near-term recommendation

Update the architecture roadmap:

1. Finish `XGOJA-TS-002` because it fixes a real review issue and teaches the system how to preserve `fs.FS` source origin metadata.
2. Implement simplified `specv2` DTOs and `migrate-spec` before building too much planner logic.
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
