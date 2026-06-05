---
Title: Single runtime profile simplification design and implementation guide
Ticket: GOJA-066
Status: active
Topics:
    - xgoja
    - refactor
    - code-generation
    - config
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/xgoja/internal/buildspec/build_spec.go
      Note: Build-time schema currently has Runtimes map
    - Path: cmd/xgoja/internal/buildspec/validate.go
      Note: Validation currently enforces runtime profiles and command runtime references
    - Path: cmd/xgoja/internal/generate/main.go
      Note: Embedded runtime JSON currently carries Runtimes map
    - Path: pkg/xgoja/app/command_providers.go
      Note: Command providers currently default/validate runtime profile context and selected modules
    - Path: pkg/xgoja/app/factory.go
      Note: RuntimeFactory currently accepts a profile name and constructs modules from Runtimes[profile]
    - Path: pkg/xgoja/app/root.go
      Note: Eval and jsverbs currently select runtime profiles and expose/consume runtime selection
    - Path: pkg/xgoja/app/run.go
      Note: Run command currently exposes --runtime and creates a selected profile runtime
    - Path: pkg/xgoja/app/runtime_spec.go
      Note: Runtime-side DTO currently mirrors multi-profile runtime map
    - Path: pkg/xgoja/app/tui.go
      Note: TUI command currently exposes --runtime and labels UI by profile
    - Path: pkg/xgoja/providerapi/commands.go
      Note: Provider command runtime factory API currently requires profile arguments
ExternalSources: []
Summary: Design and implementation guide for simplifying xgoja from named runtime profiles to one generated runtime module set.
LastUpdated: 2026-06-04T21:00:00-04:00
WhatFor: Use when implementing GOJA-066 or reviewing changes that remove xgoja multi-runtime-profile plumbing.
WhenToUse: Before editing xgoja buildspec/runtime spec fields, generated command runtime selection, provider command context, or xgoja docs/examples.
---


# Single runtime profile simplification design and implementation guide

## Executive summary

GOJA-066 proposes simplifying generated xgoja binaries from a map of named runtime profiles to one runtime module set. Today `xgoja.yaml` declares `runtimes: {name: {modules: [...]}}`, command specs point at those names with `commands.*.runtime`, command providers can point at `runtimeProfile`, and the generated app carries the same map in the embedded JSON runtime spec. In practice, current generated binaries use one runtime profile almost everywhere, usually called `main` or `repl`, while the multiple-profile support adds branching, validation rules, command flags, docs complexity, and tests for behavior we do not want to prioritize right now.

The proposed near-term model is: one generated xgoja app has one module set. All runtime-backed commands (`eval`, `run`, `repl`, `verbs`, and provider-owned command sets) use that same module set. We keep a small internal label such as `main` only where provider APIs currently need a `RuntimeProfile` string for context/provenance, but users should no longer configure or select runtime profiles. This is a simplification and not a permanent claim that multiple profiles are useless; it deliberately removes the feature until there is a concrete use case that justifies reintroducing it with a cleaner API.

## Goals

- Make `xgoja.yaml` easier to read: packages, one `modules` list, commands, jsverbs, help, and assets.
- Remove user-facing runtime profile selection from built-in commands.
- Remove command-level `runtime` settings for `eval`, `run`, `repl`, and `jsverbs`.
- Remove command-provider `runtimeProfile` settings unless we decide to keep a temporary compatibility field with a deprecation error.
- Keep provider module config, public Glazed config sections, internal xgoja config sections, host-service contributions, and lifecycle closers intact.
- Make implementation reviewable by separating schema changes, app runtime factory changes, command changes, docs/examples updates, and tests.

## Non-goals

- Do not remove provider packages, provider modules, command providers, jsverbs, host services, or Glazed config plumbing.
- Do not change the lower-level `pkg/engine` package; that package creates concrete JavaScript runtimes and does not know about xgoja profiles.
- Do not introduce a new configuration framework.
- Do not implement a backwards-compatible multi-profile shim unless the ticket owner explicitly asks for one. The preferred implementation is a clean schema change with clear validation and docs updates.
- Do not make Geppetto or Pinocchio special cases. This is an xgoja schema/runtime simplification.

## Current model: what an intern needs to understand

### Vocabulary

- **xgoja build spec**: the `xgoja.yaml` file read by `cmd/xgoja`. It is represented by `cmd/xgoja/internal/buildspec.BuildSpec`.
- **embedded runtime spec**: JSON generated from the build spec and embedded into generated `main.go`. It is represented at runtime by `pkg/xgoja/app.RuntimeSpec`.
- **provider package**: Go package compiled into the generated binary and registered with `providerapi.ProviderRegistry`.
- **provider module**: one CommonJS module a provider package can expose through `require("alias")`.
- **module instance**: one selected provider module, plus an optional `as` alias and static `config`.
- **runtime profile**: the current named grouping of module instances. This ticket removes this user-facing concept from xgoja.
- **engine runtime**: the actual Goja runtime produced by `pkg/engine`. This stays; only xgoja's named profile layer changes.

### Build-time schema today

The build-time DTO currently stores runtimes as a map:

- `cmd/xgoja/internal/buildspec/build_spec.go:16-30` defines `BuildSpec`.
- `cmd/xgoja/internal/buildspec/build_spec.go:24` contains `Runtimes map[string]RuntimeSpec`.
- `cmd/xgoja/internal/buildspec/build_spec.go:62-68` defines a runtime profile as only `Modules []ModuleInstanceSpec`.
- `cmd/xgoja/internal/buildspec/build_spec.go:98-103` gives each built-in command a `Runtime string`.
- `cmd/xgoja/internal/buildspec/build_spec.go:109-117` gives command providers `RuntimeProfile string` and an optional module filter.

The important observation is that the runtime profile object has exactly one payload: the selected module instances. The name is mostly a selector key.

Current YAML shape:

```yaml
packages:
  - id: geppetto
    import: github.com/go-go-golems/geppetto/pkg/js/modules/geppetto/provider

runtimes:
  main:
    modules:
      - package: geppetto
        name: geppetto
        as: geppetto

commands:
  jsverbs:
    enabled: true
    runtime: main
    name: verbs
```

Proposed YAML shape:

```yaml
packages:
  - id: geppetto
    import: github.com/go-go-golems/geppetto/pkg/js/modules/geppetto/provider

modules:
  - package: geppetto
    name: geppetto
    as: geppetto

commands:
  jsverbs:
    enabled: true
    name: verbs
```

### Validation today

Validation currently treats runtimes and command runtime references as first-class schema elements:

- `cmd/xgoja/internal/buildspec/validate.go:189-228` validates every runtime profile in the map.
- `cmd/xgoja/internal/buildspec/validate.go:230-235` validates the built-in command runtime selectors.
- `cmd/xgoja/internal/buildspec/validate.go:251-265` requires every enabled built-in command to name a runtime profile.
- `cmd/xgoja/internal/buildspec/validate.go:287-292` validates `commandProviders[].runtimeProfile` when present.

This is the first place to simplify: validate one top-level module list, then stop validating per-command runtime selectors because there should be no per-command runtime selector.

### Embedded runtime schema today

The generated binary does not read YAML directly. `cmd/xgoja/internal/generate` turns the build spec into embedded JSON, then generated `main.go` decodes it into `pkg/xgoja/app.RuntimeSpec`.

Current runtime DTO shape:

- `pkg/xgoja/app/runtime_spec.go:15-28` defines runtime-side `RuntimeSpec`.
- `pkg/xgoja/app/runtime_spec.go:22` has `Runtimes map[string]RuntimeProfileSpec`.
- `pkg/xgoja/app/runtime_spec.go:45-50` defines `RuntimeProfileSpec` as `Modules []ModuleInstanceSpec`.
- `pkg/xgoja/app/runtime_spec.go:77-82` repeats command `Runtime string`.
- `pkg/xgoja/app/runtime_spec.go:87-95` repeats command-provider `RuntimeProfile string`.

After the simplification, the embedded runtime spec should carry `Modules []ModuleInstanceSpec` directly.

### Generated runtime construction today

`pkg/xgoja/app.RuntimeFactory` turns a selected runtime profile into a concrete `engine.Runtime`:

- `pkg/xgoja/app/factory.go:70-72` exposes `NewRuntime(ctx, profile, ...)`.
- `pkg/xgoja/app/factory.go:74-82` exposes `NewRuntimeFromSections(ctx, profile, vals, ...)` and looks up `runtimeSpec.Runtimes[profile]`.
- `pkg/xgoja/app/factory.go:86-114` collects host services, resolves modules, maps config, and builds module registrars for the selected profile.
- `pkg/xgoja/app/factory.go:119-127` forwards `--debug-panic-stack` into the engine builder.

The profile parameter should disappear from xgoja-facing runtime factory methods. The factory already owns one `RuntimeSpec`; after this ticket, that spec should have exactly one module list.

Proposed API sketch:

```go
type RuntimeFactory struct {
    providers   *providerapi.ProviderRegistry
    runtimeSpec *RuntimeSpec
    services    providerapi.HostServices
}

func (f *RuntimeFactory) NewRuntime(ctx context.Context, opts ...require.Option) (*JSRuntime, error) {
    return f.NewRuntimeFromSections(ctx, nil, opts...)
}

func (f *RuntimeFactory) NewRuntimeFromSections(
    ctx context.Context,
    vals *values.Values,
    opts ...require.Option,
) (*JSRuntime, error) {
    modules := f.runtimeSpec.Modules
    descriptors, err := f.selectedModuleDescriptors()
    // host services, config mapping, engine builder remain the same
}
```

### Built-in generated commands today

Built-in runtime-backed commands all carry profile selection:

- `pkg/xgoja/app/root.go:65-68` defines `evalSettings` with `Runtime string`.
- `pkg/xgoja/app/root.go:70-92` creates an `eval --runtime` flag.
- `pkg/xgoja/app/root.go:105-117` reads the selected runtime and creates a runtime from that profile.
- `pkg/xgoja/app/run.go:28-31` defines `runSettings` with `Runtime string`.
- `pkg/xgoja/app/run.go:36-58` creates a `run --runtime` flag.
- `pkg/xgoja/app/tui.go:35-38` defines `tuiSettings` with `Runtime string`.
- `pkg/xgoja/app/tui.go:42-58` creates a `repl --runtime` flag.
- `pkg/xgoja/app/root.go:233-275` builds jsverb commands for one selected profile and passes that profile into runtime creation.
- `pkg/xgoja/app/root.go:316-329` has `firstRuntime`, which chooses a fallback profile from the map.

After the simplification:

- remove `runtime` from `evalSettings`, `runSettings`, and `tuiSettings`;
- remove all `--runtime` flags from generated built-in commands;
- replace `firstRuntime` and `commandRuntime` with a constant internal context label or remove them entirely;
- have all commands call `factory.NewRuntimeFromSections(ctx, vals, ...)`.

### Command providers today

Provider-owned command sets also receive runtime-profile context:

- `pkg/xgoja/app/command_providers.go:59-79` builds `providerapi.CommandSetContext` with `RuntimeProfile` and `SelectedModules`.
- `pkg/xgoja/app/command_providers.go:89-95` defaults an empty command-provider profile to `firstRuntime`.
- `pkg/xgoja/app/command_providers.go:97-108` selects modules for that profile and optionally filters them by `commandProviders[].modules`.

There are two separate concepts here:

1. the selected runtime module set, and
2. optional module filtering for a command provider.

The first becomes global. The second can stay if command providers need to know only a subset of selected modules. The recommended near-term change is to keep `CommandProviderInstanceSpec.Modules []string` but remove `RuntimeProfile`. `SelectedModules` should be computed from the one global module set, optionally filtered by `Modules`.

## Proposed model

### New build-time schema

Use one top-level module list:

```go
type BuildSpec struct {
    Name             string
    AppName          string
    EnvPrefix        string
    ConfigFile       *ConfigFileSpec
    Go               GoSpec
    Target           TargetSpec
    Packages         []PackageSpec
    Modules          []ModuleInstanceSpec `yaml:"modules"`
    Commands         CommandsSpec
    CommandProviders []CommandProviderInstanceSpec
    JSVerbs          []JSVerbSourceSpec
    Help             HelpSpec
    Assets           []AssetSourceSpec
    BaseDir          string
}
```

Command specs lose `Runtime`:

```go
type CommandSpec struct {
    Enabled bool   `yaml:"enabled" json:"enabled"`
    Name    string `yaml:"name" json:"name,omitempty"`
    Mount   string `yaml:"mount" json:"mount,omitempty"`
}
```

Command providers lose `RuntimeProfile`:

```go
type CommandProviderInstanceSpec struct {
    ID      string
    Package string
    Name    string
    Mount   string
    Modules []string
    Config  map[string]any
    Lazy    bool
}
```

### New runtime-side schema

Mirror the same simplification in `pkg/xgoja/app`:

```go
type RuntimeSpec struct {
    Name             string
    AppName          string
    EnvPrefix        string
    ConfigFile       *ConfigFileSpec
    Target           TargetSpec
    Packages         []PackageSpec
    Modules          []ModuleInstanceSpec `json:"modules,omitempty"`
    Commands         CommandsSpec
    CommandProviders []CommandProviderInstanceSpec
    JSVerbs          []JSVerbSourceSpec
    Help             HelpSpec
    Assets           []AssetSourceSpec
}
```

`RuntimeProfileSpec` can be deleted unless a temporary migration helper needs it.

### Internal compatibility label

Provider APIs currently include `RuntimeProfile` in `providerapi.SectionRequest`, `providerapi.HostServiceContributionRequest`, and `providerapi.CommandSetContext`. Removing that field immediately is possible, but it is a larger provider API break. The simpler implementation is to keep the field for now and always pass a constant:

```go
const defaultRuntimeProfile = "main"
```

That gives provider capabilities a stable context string while removing user-facing selection. Later cleanup can rename this field to something more accurate, such as `RuntimeID` or remove it from provider-facing APIs if it proves unused.

### Before/after architecture diagram

Current architecture:

```text
xgoja.yaml
  packages[]
  runtimes[name].modules[]
  commands.eval.runtime --+
  commands.run.runtime ---+--> selected profile name
  commands.repl.runtime --+
  commands.jsverbs.runtime
           |
           v
embedded RuntimeSpec.Runtimes map
           |
           v
RuntimeFactory.NewRuntimeFromSections(ctx, profile, vals)
           |
           v
engine.Runtime with modules from Runtimes[profile]
```

Proposed architecture:

```text
xgoja.yaml
  packages[]
  modules[]
  commands.* without runtime selectors
           |
           v
embedded RuntimeSpec.Modules
           |
           v
RuntimeFactory.NewRuntimeFromSections(ctx, vals)
           |
           v
engine.Runtime with the one generated module set
```

### Runtime construction pseudocode

```text
NewRuntimeFromSections(ctx, vals, opts...):
  assert factory, provider registry, and runtime spec are present

  descriptors = selectedModuleDescriptors()

  runtimeServices = hostServicesForRuntime(ctx, "main", vals, descriptors)
  ensure contribution closers are closed if construction fails

  for instance in runtimeSpec.Modules:
    module = providers.ResolveModule(instance.Package, instance.Name)
    config = configForModuleInstance(ctx, "main", instance, descriptor, vals)
    registrar = providerRuntimeModuleRegistrar(instance, module, config, runtimeServices.services)
    append registrar

  if runtimeServices has closers:
    prepend hostServiceCloserRegistrar

  includeStack = includeRecoveredPanicStack(vals)
  builder = engine.NewRuntimeFactoryBuilder(
    WithImplicitDefaultRegistryModules(false),
    WithDataOnlyDefaultRegistryModules(false),
    WithRecoveredPanicStack(includeStack),
  ).WithModules(registrars...)

  if require options were passed:
    builder.WithRequireOptions(opts...)

  return builder.Build().NewRuntime(startup/lifetime ctx)
```

## Implementation plan

### Phase 1: schema and validation

1. Update `cmd/xgoja/internal/buildspec/build_spec.go`:
   - add `Modules []ModuleInstanceSpec` to `BuildSpec`;
   - remove `Runtimes map[string]RuntimeSpec`;
   - remove `RuntimeSpec` if no longer needed;
   - remove `CommandSpec.Runtime`;
   - remove `CommandProviderInstanceSpec.RuntimeProfile`.
2. Update `cmd/xgoja/internal/buildspec/load.go` if unsupported/deprecated-field detection is desired.
3. Replace `validateRuntimes` with `validateModules`:
   - require at least one top-level module;
   - validate package id, module name, alias non-empty, and alias uniqueness;
   - use paths such as `modules[0].package` instead of `runtimes.main.modules[0].package`.
4. Replace `validateCommandRuntime` with a command-enabled check that no longer requires a runtime selector.
5. Update command-provider validation so `runtimeProfile` is not accepted or is reported as deprecated/unsupported.

Recommended validation behavior for a clean break:

```text
if YAML contains top-level `runtimes`:
  error: runtimes are no longer supported; move the one runtime's modules to top-level modules

if command has `runtime`:
  error: commands.*.runtime is no longer supported; all commands use top-level modules

if commandProvider has `runtimeProfile`:
  error: commandProviders[].runtimeProfile is no longer supported; all command providers use top-level modules
```

Strict validation is better than silently choosing one profile because the goal is to remove ambiguity.

### Phase 2: code generation

1. Update `cmd/xgoja/internal/generate/main.go`:
   - embedded payload should include `Modules` instead of `Runtimes`;
   - generated JSON should not include command `runtime` or command provider `runtimeProfile`.
2. Update generated tests in `cmd/xgoja/internal/generate/generate_test.go`.
3. Confirm `cmd/xgoja/internal/generate/templates/main.go.tmpl` does not need changes. It decodes the embedded spec and delegates to `app`; the schema change should be handled by DTOs and renderer input.

### Phase 3: app runtime DTOs and factory

1. Update `pkg/xgoja/app/runtime_spec.go`:
   - replace `Runtimes map[string]RuntimeProfileSpec` with `Modules []ModuleInstanceSpec`;
   - delete `RuntimeProfileSpec` if unused;
   - remove `CommandSpec.Runtime`;
   - remove `CommandProviderInstanceSpec.RuntimeProfile`.
2. Update `pkg/xgoja/app/module_sections.go`:
   - change `selectedModuleDescriptors(profile string)` to `selectedModuleDescriptors()`;
   - change `sectionsForRuntimeProfile(commandName, profile string)` to `sectionsForRuntime(commandName string)`;
   - keep `providerapi.SectionRequest.RuntimeProfile = "main"` internally for now.
3. Update `pkg/xgoja/app/factory.go`:
   - change `NewRuntime(ctx, profile, opts...)` to `NewRuntime(ctx, opts...)`;
   - change `NewRuntimeFromSections(ctx, profile, vals, opts...)` to `NewRuntimeFromSections(ctx, vals, opts...)`;
   - iterate over `f.runtimeSpec.Modules` instead of `runtime.Modules`;
   - use `defaultRuntimeProfile` only as provider context.
4. Update host-service and config mapping helpers to receive the constant context label instead of user-selected profile names.

### Phase 4: built-in commands

1. `eval`:
   - remove `Runtime string` from `evalSettings`;
   - remove the `runtime` flag;
   - call `selectedModuleDescriptors()`;
   - call `evalSourceWithInitializers(ctx, factory, source, vals, selectedModules, out)`.
2. `run`:
   - remove `Runtime string` from `runSettings`;
   - remove the `runtime` flag;
   - keep `keep-alive`;
   - call `factory.NewRuntimeFromSections(ctx, vals, requireOpt)`.
3. `repl`/TUI:
   - remove `Runtime string` from `tuiSettings`;
   - remove the `runtime` flag;
   - change UI title from `Runtime %s` to something like `Generated xgoja runtime`.
4. `jsverbs`:
   - build module sections from the one module set;
   - use the same one runtime in every verb invocation;
   - update error/source strings to avoid mentioning runtime profiles.
5. Delete `firstRuntime` and `commandRuntime` if no longer used.

### Phase 5: command providers

1. Update runtime provider factory interface in `pkg/xgoja/providerapi/commands.go`:
   - prefer `NewRuntime(ctx, opts...)` and `NewRuntimeFromSections(ctx, vals, opts...)`;
   - consider keeping old names only if a compatibility shim is explicitly requested.
2. Update `CommandSetContext`:
   - remove or deprecate `RuntimeProfile`;
   - keep `SelectedModules` because it is still useful.
3. Update `pkg/xgoja/app/command_providers.go`:
   - delete `runtimeProfileForCommandProvider`;
   - select modules from the global module set;
   - apply `commandProviders[].modules` as a filter if present.

### Phase 6: docs, examples, tests

Update every example `xgoja.yaml`:

```text
runtimes:
  main:
    modules:
      - package: p
        name: m
commands:
  eval:
    enabled: true
    runtime: main
```

becomes:

```text
modules:
  - package: p
    name: m
commands:
  eval:
    enabled: true
```

Remove or rewrite `examples/xgoja/03-multiple-runtimes`. Suggested replacement:

```text
examples/xgoja/03-single-runtime-modules
```

Use it to show that one generated binary can include multiple modules in one runtime, not multiple named runtimes.

Docs requiring updates include at least:

- `cmd/xgoja/doc/01-overview.md`
- `cmd/xgoja/doc/02-user-guide.md`
- `cmd/xgoja/doc/03-tutorial-using-xgoja-yaml.md`
- `cmd/xgoja/doc/04-tutorial-providing-package-and-modules.md`
- `cmd/xgoja/doc/05-tutorial-providing-commands.md`
- `cmd/xgoja/doc/06-buildspec-reference.md`
- `cmd/xgoja/doc/08-playbook-adding-xgoja-support.md`
- `cmd/xgoja/doc/09-tutorial-static-assets-http-server.md`
- `cmd/xgoja/doc/11-provider-runtime-config-and-host-services.md`
- `examples/xgoja/**/xgoja.yaml`

Tests requiring updates include at least:

- `cmd/xgoja/internal/buildspec/load_test.go`
- `cmd/xgoja/internal/buildspec/validate_test.go`
- `cmd/xgoja/internal/generate/generate_test.go`
- `pkg/xgoja/app/*module_sections_test.go`
- `pkg/xgoja/app/command_providers_test.go`
- `pkg/xgoja/app/root_test.go`
- `pkg/xgoja/app/run_module_sections_test.go`
- `pkg/xgoja/app/tui_module_sections_test.go`
- `pkg/xgoja/app/jsverbs_module_sections_test.go`
- `pkg/xgoja/providers/host/host_test.go`

## Decision records

### Decision 1: one top-level module list, not a map with one required key

Status: proposed.

Options considered:

- Keep `runtimes.main.modules` but validate that the map has exactly one key.
- Move modules to top-level `modules`.
- Keep multiple profiles but hide the `--runtime` flags.

Decision: move modules to top-level `modules`.

Rationale: a one-entry map still teaches users a profile abstraction that this ticket wants to remove. Top-level `modules` is simpler and directly names the thing users configure: the modules compiled into the generated runtime.

Consequences: this is a breaking schema change for existing xgoja examples and generated build specs. The implementation should update docs and examples in the same branch.

### Decision 2: remove command-level runtime selectors

Status: proposed.

Decision: remove `commands.*.runtime` from schema and generated command flags.

Rationale: if there is one module set, command-level runtime selectors are redundant and misleading. Leaving the fields in place would preserve the old mental model.

Consequences: command validation becomes simpler, and built-in commands no longer need `runtime` in their settings structs.

### Decision 3: keep a temporary internal provider context label

Status: proposed.

Decision: keep passing `RuntimeProfile: "main"` to provider capability request structs for now, even though users cannot select a profile.

Rationale: provider APIs already accept `RuntimeProfile`, and some providers may log or branch on it. Removing it is possible but broadens the change. Passing a constant keeps the refactor focused on xgoja user-facing simplification.

Consequences: code still has one misleading field name in providerapi. A later cleanup can rename or remove it after this schema simplification lands.

## Risks and review points

- **Breaking existing examples:** all xgoja examples currently use `runtimes`. Update them in the same change and run generation/build smokes.
- **Hidden provider assumptions:** providers may use `RuntimeProfile` in `SectionRequest` or `CommandSetContext`. Keep a constant initially and search downstream providers before removing it.
- **Config-file migration:** generated xgoja config-file examples may contain command `runtime` fields. Docs should say these fields are obsolete.
- **Command-provider module filtering:** make sure `commandProviders[].modules` continues to filter selected modules if that feature is still useful.
- **Test churn:** many tests build `RuntimeSpec{Runtimes: ...}` inline. Expect a wide but mechanical update.

## Validation checklist

Run these after implementation:

```bash
cd go-go-goja
go test ./cmd/xgoja/internal/buildspec ./cmd/xgoja/internal/generate ./pkg/xgoja/app -count=1
go test ./cmd/xgoja/... ./pkg/xgoja/... -count=1
go test ./... -count=1
```

Run example generation/build checks for at least:

```bash
cd go-go-goja/examples/xgoja/01-core-provider && make clean build smoke
cd go-go-goja/examples/xgoja/07-embedded-jsverbs && make clean build smoke
cd go-go-goja/examples/xgoja/12-geppetto-host-services && make clean build pinocchio-smoke
```

If a `make smoke` target does not exist for an example, run the example's documented build and invocation commands from its README.

## Suggested migration message for users

```text
xgoja no longer supports named runtime profiles. Move the module list from
runtimes.<name>.modules to top-level modules, then remove commands.*.runtime and
commandProviders[].runtimeProfile. All generated commands now use the single
module set.
```

## File reference index

| Area | Files | Why they matter |
|---|---|---|
| Build-time YAML DTOs | `cmd/xgoja/internal/buildspec/build_spec.go` | Owns `BuildSpec`, `Runtimes`, `CommandSpec.Runtime`, and `CommandProviderInstanceSpec.RuntimeProfile`. |
| Buildspec loading/defaults | `cmd/xgoja/internal/buildspec/load.go` | Reads YAML and applies defaults before validation. |
| Buildspec validation | `cmd/xgoja/internal/buildspec/validate.go` | Validates runtime profile maps and command runtime references today. |
| Embedded JSON generation | `cmd/xgoja/internal/generate/main.go` | Converts build-time spec to runtime JSON; must switch from `Runtimes` to `Modules`. |
| Generated main template | `cmd/xgoja/internal/generate/templates/main.go.tmpl` | Probably unchanged; it delegates to `app.NewRootCommand`/`Host`. |
| Runtime DTOs | `pkg/xgoja/app/runtime_spec.go` | Runtime-side schema currently mirrors multi-profile shape. |
| Runtime factory | `pkg/xgoja/app/factory.go` | Creates concrete engine runtimes from a selected profile. |
| Module sections | `pkg/xgoja/app/module_sections.go` | Collects provider Glazed sections for selected modules. |
| Built-in commands | `pkg/xgoja/app/root.go`, `run.go`, `tui.go` | Define `--runtime` flags and pass profile names into factory calls. |
| Command providers | `pkg/xgoja/app/command_providers.go`, `pkg/xgoja/providerapi/commands.go` | Carry runtime profile context for provider-owned commands. |
| Docs/examples | `cmd/xgoja/doc/*.md`, `examples/xgoja/**/xgoja.yaml` | Need to teach the simplified schema. |

## Final recommendation

Implement this as a clean breaking xgoja schema refactor, not as a partial compatibility layer. The old feature can be reintroduced later if a real user story needs multiple isolated module sets in one generated binary. For now, the simpler one-runtime model matches current usage, makes provider config lifecycle work easier to understand, and removes a misleading distinction between xgoja runtime profiles and actual `engine.Runtime` instances.
