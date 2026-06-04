---
Title: Buildspec Spec Rename Diary
Ticket: GOJA-053
Status: active
Topics:
    - xgoja
    - code-generation
    - lifecycle
    - documentation
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/GLOSSARY.md
      Note: |-
        Added top-level glossary defining the *Spec pattern.
        Top-level *Spec pattern definition
        Updated top-level *Spec pattern examples for app DTOs
        Updated BuildSpec and RuntimeSpec glossary examples
    - Path: go-go-goja/cmd/xgoja/cmd_list_modules.go
      Note: CLI helper updated to use RuntimeSpec
    - Path: go-go-goja/cmd/xgoja/internal/buildspec/build_spec.go
      Note: BuildSpec and ConfigFileSpec source
    - Path: go-go-goja/cmd/xgoja/internal/buildspec/spec.go
      Note: |-
        Renamed buildspec Runtime/ModuleInstance/CommandProviderInstance DTO types to explicit *Spec names.
        Buildspec DTO type rename source
        Added build-time spec file documentation
    - Path: go-go-goja/cmd/xgoja/internal/buildspec/validate.go
      Note: Buildspec validation signature updated for RuntimeSpec
    - Path: go-go-goja/cmd/xgoja/internal/generate/main.go
      Note: |-
        Embedded runtime JSON payload updated to use renamed buildspec DTO types
        Embedded runtime JSON emits configFile
    - Path: go-go-goja/pkg/xgoja/app/command_providers.go
      Note: Command provider helpers updated to use CommandProviderInstanceSpec
    - Path: go-go-goja/pkg/xgoja/app/factory.go
      Note: |-
        RuntimeFactory updated to use ModuleInstanceSpec
        RuntimeFactory stores runtimeSpec
    - Path: go-go-goja/pkg/xgoja/app/host.go
      Note: Host field renamed to RuntimeSpec
    - Path: go-go-goja/pkg/xgoja/app/middlewares.go
      Note: Config-file middleware reads RuntimeSpec.ConfigFile
    - Path: go-go-goja/pkg/xgoja/app/runtime_spec.go
      Note: RuntimeSpec
    - Path: go-go-goja/pkg/xgoja/app/spec.go
      Note: Runtime-side embedded DTO type rename source
ExternalSources: []
Summary: 'Diary for the first symbol-name simplification: buildspec runtime/module/command-provider DTOs now use explicit *Spec names.'
LastUpdated: 2026-06-04T00:00:00Z
WhatFor: Use when reviewing the focused commit that renamed buildspec DTO types to make the *Spec pattern explicit.
WhenToUse: Before continuing generic-symbol cleanup or renaming runtime profile DTOs in the app runtime spec.
---




# Diary

## Goal

Capture the first concrete naming simplification from the generic-symbol glossary: make build-time spec DTO types explicitly use the `*Spec` suffix and document the `*Spec` pattern at the repository top level.

## Step 1: Rename Buildspec DTOs to Explicit Spec Types

I renamed the build-time `xgoja.yaml` DTO types in `cmd/xgoja/internal/buildspec/spec.go` so they read as declarative specification data instead of active runtime/module objects. `Runtime` became `RuntimeSpec`, `ModuleInstance` became `ModuleInstanceSpec`, and `CommandProviderInstance` became `CommandProviderInstanceSpec`.

I also added a top-level `GLOSSARY.md` that defines the `*Spec` pattern: a `*Spec` type describes what should be built, selected, embedded, or configured, and should not itself perform lifecycle, scheduling, registration, or runtime work.

### Prompt Context

**User prompt (verbatim):** "ok, so first simplification:

Rename Runtime in buildspec/spec.go to RuntimeSpec, and ModuleInstance to ModuleInstanceSpec.  CommandProviderInstanceSpec as well.

Do that, record diary, commit. 

Also add a GLOSSARY.md in the toplevel that has the definition of the *Spec pattern."

**Assistant interpretation:** Apply the first naming cleanup in buildspec code, add top-level glossary documentation for the `*Spec` pattern, record the change in a diary, validate it, and commit it.

**Inferred user intent:** Start converting the generic-symbol glossary into concrete code cleanup while keeping the change focused and reviewable.

**Commit (code):** cc46401040a073fd8f0207ed0dbb3207a53430f5 — "Clarify buildspec spec DTO names"

### What I did

- Renamed buildspec DTO types:
  - `Runtime` → `RuntimeSpec`
  - `ModuleInstance` → `ModuleInstanceSpec`
  - `CommandProviderInstance` → `CommandProviderInstanceSpec`
- Updated buildspec validation and generate package references.
- Preserved `CommandSpec.Runtime` as a field name because it is a YAML/JSON field for selecting a runtime profile, not the `Runtime` type being renamed.
- Added top-level `GLOSSARY.md` with the `*Spec` pattern definition and examples.
- Ran targeted tests:
  - `go test ./cmd/xgoja/internal/buildspec ./cmd/xgoja/internal/generate -count=1`
  - `go test ./cmd/xgoja/... -count=1`

### Why

The old buildspec names made declarative runtime profile data look like active runtime/module objects. The `*Spec` suffix is a cheap, focused way to encode the pattern boundary in code: these types describe build-time inputs, while active runtime behavior remains in `engine.Runtime`, `app.RuntimeFactory`, provider modules, and CommonJS loaders.

### What worked

- The type rename was localized to buildspec and code generation references.
- Existing YAML/JSON tags remain unchanged, so the external `xgoja.yaml` shape is preserved.
- `go test ./cmd/xgoja/... -count=1` passed.

### What didn't work

- My first broad textual replacement also renamed the `CommandSpec.Runtime` field to `RuntimeSpec`. I corrected that immediately because the user requested type renames, not a YAML/JSON field rename.
- I initially ran `go test ./cmd/xgoja/internal/buildspec ./cmd/xgoja/internal/generate -count=1` from the workspace root, which failed because the Go module root is `go-go-goja`:
  - `stat /home/manuel/workspaces/2026-06-03/goja-runtime-flags/cmd/xgoja/internal/buildspec: directory not found`
  - `stat /home/manuel/workspaces/2026-06-03/goja-runtime-flags/cmd/xgoja/internal/generate: directory not found`
- Re-running from `go-go-goja` passed.

### What I learned

- The buildspec rename is mechanically simple, but broad search/replace is unsafe because `Runtime` is both a type name and a field name in `CommandSpec`.
- The top-level glossary helps document the intended naming pattern before further renames touch broader app/runtime layers.

### What was tricky to build

The sharp edge was distinguishing type names from schema field names. `RuntimeSpec` is the right name for the build-time runtime profile DTO, but the `CommandSpec.Runtime` field should remain `runtime` because it is the external YAML/JSON field users already write and because it denotes the selected runtime profile value.

### What warrants a second pair of eyes

- Confirm that only build-time `buildspec` DTOs should be renamed in this first step; `app.Runtime`, `app.ModuleInstance`, and `app.CommandProviderInstance` remain unchanged for now.
- Confirm whether the top-level glossary should stay concise or become the landing page for the larger GOJA-053 naming glossary.

### What should be done in the future

- Consider a follow-up rename for `app.Runtime` to `RuntimeProfile` or `RuntimeSpec` after deciding whether runtime embedded spec types should also carry `*Spec` names.
- Consider adding type aliases only if downstream packages outside this module break; no aliases were added in this focused internal rename.

### Code review instructions

- Start with `cmd/xgoja/internal/buildspec/spec.go` and verify the renamed DTO types and unchanged YAML/JSON tags.
- Review `cmd/xgoja/internal/generate/main.go` and `generate_test.go` to confirm generated JSON shape is unchanged.
- Review `GLOSSARY.md` to confirm the `*Spec` pattern definition matches the code direction.
- Validate with: `cd go-go-goja && go test ./cmd/xgoja/... -count=1`.

### Technical details

Commands run:

```bash
cd go-go-goja && go test ./cmd/xgoja/internal/buildspec ./cmd/xgoja/internal/generate -count=1
cd go-go-goja && go test ./cmd/xgoja/... -count=1
```

## Step 2: Rename Runtime-Side App DTOs to Spec Types

I applied the same `*Spec` naming pattern to the runtime-side embedded schema in `pkg/xgoja/app/spec.go`. `Runtime` became `RuntimeSpec`, `ModuleInstance` became `ModuleInstanceSpec`, and `CommandProviderInstance` became `CommandProviderInstanceSpec` so the generated app layer is explicit that these are decoded runtime-spec DTOs, not concrete runtime/module/provider objects.

I also added file-level documentation to both build-time and runtime-side spec files. The comments explain that `buildspec.Spec` is the `xgoja.yaml` input schema and that `app.Spec` is the smaller embedded JSON schema generated from it, with build-only concerns stripped out.

### Prompt Context

**User prompt (verbatim):** "ok, rename Rntime to RuntimeSpec, ModuleInstance to ModuleInstanceSpec, CommandProviderInstanceSpec to clearly show they are DTOs, in app/spec.go as well."

**Assistant interpretation:** Continue the DTO naming cleanup by applying the `*Spec` suffix to app runtime spec types as well.

**Inferred user intent:** Make both build-time and embedded runtime spec layers consistently communicate that these types are declarative data transfer objects.

**Commit (code):** af97b7a60e3c6d9a2c9817e5d400d95a64b9603e — "Clarify app runtime spec DTO names"

### What I did

- Renamed app runtime spec DTO types:
  - `app.Runtime` → `app.RuntimeSpec`
  - `app.ModuleInstance` → `app.ModuleInstanceSpec`
  - `app.CommandProviderInstance` → `app.CommandProviderInstanceSpec`
- Updated app runtime factory, command provider helpers, tests, and provider tests that referenced the renamed types.
- Added file-level comments to:
  - `cmd/xgoja/internal/buildspec/spec.go`
  - `pkg/xgoja/app/spec.go`
- Added top-level examples for `app.*Spec` DTOs in `GLOSSARY.md`.
- Ran targeted tests:
  - `go test ./pkg/xgoja/... ./cmd/xgoja/... -count=1`

### Why

After renaming build-time DTOs, leaving the runtime embedded DTOs as `Runtime`, `ModuleInstance`, and `CommandProviderInstance` preserved the same ambiguity one layer later. Applying `*Spec` consistently makes the buildspec-to-app-spec boundary easier to explain and keeps concrete runtime concepts reserved for `engine.Runtime` and runtime factories.

### What worked

- The rename was mechanical and all targeted xgoja tests passed.
- External JSON shape remains unchanged because struct tags are unchanged.
- The new comments document why the apparent duplication between `buildspec/spec.go` and `app/spec.go` exists.

### What didn't work

- N/A.

### What I learned

- The app runtime layer had the same DTO naming problem as buildspec, but the fix is straightforward when the schema tags are preserved.
- File-level comments are important here because otherwise `buildspec.Spec` and `app.Spec` look like accidental duplication.

### What was tricky to build

The main invariant was preserving external JSON/YAML field names while renaming Go types only. `CommandSpec.Runtime` remains unchanged because it is a field representing the selected runtime profile name, not the runtime-profile DTO type.

### What warrants a second pair of eyes

- Confirm that `app.RuntimeSpec` is the preferred name over `app.RuntimeProfileSpec`.
- Confirm whether `app.Spec` itself should eventually be renamed to `RuntimeSpec` or `EmbeddedSpec`, or whether package qualification is enough.

### What should be done in the future

- Update longer GOJA-053 design documents to use the new `app.RuntimeSpec` and `buildspec.RuntimeSpec` names when they are next edited.
- Consider renaming `app.JSRuntime` alias or removing it from docs if it still causes confusion with spec DTOs.

### Code review instructions

- Start with `pkg/xgoja/app/spec.go` to verify renamed runtime DTO types and comments.
- Check `pkg/xgoja/app/factory.go` and `pkg/xgoja/app/command_providers.go` for updated parameter types.
- Validate with: `cd go-go-goja && go test ./pkg/xgoja/... ./cmd/xgoja/... -count=1`.

### Technical details

Command run:

```bash
cd go-go-goja && go test ./pkg/xgoja/... ./cmd/xgoja/... -count=1
```

## Step 3: Make Top-Level Build and Runtime Specs Explicit

I renamed the top-level DTOs so package qualification is no longer the only signal for whether a document is build-time or runtime-time. `buildspec.Spec` is now `buildspec.BuildSpec`, and `app.Spec` is now `app.RuntimeSpec`; the per-profile runtime selection type was renamed to `app.RuntimeProfileSpec` to avoid colliding with the new top-level runtime document name.

I also renamed the source files to match the new top-level type names and renamed owner fields that previously used the generic `Spec` label. `Host.Spec` is now `Host.RuntimeSpec`, and the runtime factory stores `runtimeSpec` rather than a generic `spec` field.

### Prompt Context

**User prompt (verbatim):** "rename app/spec.go Spec to RuntimeSpec, and buildspec/spec.go Spec to BuildSpec (even if it echose the package name). Rename the field namesfr omSpec to Runtime/BuildSpec as well. The files accordingly as well to runtime_spec and build_spec"

**Assistant interpretation:** Rename the remaining generic top-level spec DTOs and matching file/field names so they explicitly describe build-time versus runtime-time schema roles.

**Inferred user intent:** Remove the last ambiguous `Spec` type/field names from xgoja's generated app and buildspec layers.

**Commit (code):** a50de5d3d1d3b2c2b7b95d42efca9bd06bda0263 — "Clarify top-level build and runtime specs"

### What I did

- Renamed `cmd/xgoja/internal/buildspec/spec.go` to `build_spec.go`.
- Renamed `pkg/xgoja/app/spec.go` to `runtime_spec.go`.
- Renamed `buildspec.Spec` to `buildspec.BuildSpec` across load, validate, generate, CLI, and tests.
- Renamed `app.Spec` to `app.RuntimeSpec` across app code, generated templates, host tests, and generated-output tests.
- Renamed the embedded runtime-profile DTO to `app.RuntimeProfileSpec` so the top-level runtime document can own the `RuntimeSpec` name.
- Renamed app owner fields from generic `Spec`/`spec` to `RuntimeSpec`/`runtimeSpec` where they store the top-level runtime document.

### Why

The prior cleanup made many DTOs explicit but still left the most important top-level documents as `Spec`. That forced readers to rely on package names and local variable context. Naming the top-level types `BuildSpec` and `RuntimeSpec` makes the build-time-to-runtime-time conversion path visible in signatures.

### What worked

- The rename stayed mechanical after introducing `RuntimeProfileSpec` for nested runtime profiles.
- Targeted xgoja tests passed after fixing references to the renamed runtime factory field.

### What didn't work

- The first mechanical rename temporarily produced build errors because `RuntimeFactory` had been renamed from `spec` to `runtimeSpec`, while helper code and tests still referenced `factory.spec` / `f.spec`.
- Exact failure:
  - `pkg/xgoja/app/module_sections.go:16:41: f.spec undefined (type *RuntimeFactory has no field or method spec)`
  - `pkg/xgoja/app/eval_module_sections_test.go:13:41: factory.spec undefined (type *RuntimeFactory has no field or method spec)`

### What I learned

- `RuntimeSpec` is better reserved for the top-level embedded runtime document; the nested named runtime entries are clearer as `RuntimeProfileSpec`.
- Field renames are more error-prone than type renames because they affect tests that intentionally inspect private fields inside the package.

### What was tricky to build

The sharp edge was avoiding a name collision between the newly requested `app.RuntimeSpec` top-level document and the existing `app.RuntimeSpec` runtime-profile DTO from Step 2. I resolved that by introducing `RuntimeProfileSpec` for the nested `runtimes` map values and updating external references such as `map[string]app.RuntimeProfileSpec`.

### What warrants a second pair of eyes

- Confirm that `RuntimeProfileSpec` is the right name for entries in the `runtimes` map.
- Confirm whether field names such as `RuntimeFactory.runtimeSpec` and `Host.RuntimeSpec` are the desired public/private spelling.

### What should be done in the future

- Update broader GOJA-053 docs and xgoja user docs to mention `BuildSpec`, `RuntimeSpec`, and `RuntimeProfileSpec` when those docs are next edited.

### Code review instructions

- Start with `cmd/xgoja/internal/buildspec/build_spec.go` and `pkg/xgoja/app/runtime_spec.go`.
- Review `cmd/xgoja/internal/generate/main.go` and `templates/main.go.tmpl` to confirm generated binaries decode `app.RuntimeSpec`.
- Validate with: `cd go-go-goja && go test ./pkg/xgoja/... ./cmd/xgoja/... -count=1`.

### Technical details

Commands run:

```bash
cd go-go-goja && go test ./pkg/xgoja/... ./cmd/xgoja/... -count=1
```

## Step 4: Rename Config File Spec Fields

I renamed the top-level generated-command config-file DTO from generic `Config`/`ConfigSpec` to `ConfigFile`/`ConfigFileSpec` in both build-time and runtime spec files. This separates Glazed config-file discovery settings from module/provider `Config` maps, which still represent provider-specific static module config.

The serialized top-level field also moved from `config` to `configFile` in the build YAML and embedded runtime JSON shapes. Module instance and command provider `config` maps intentionally keep their existing tags because they are provider configuration payloads, not generated-command config-file discovery settings.

### Prompt Context

**User prompt (verbatim):** "rename Config (ConfigSpec) into ConfigFile and ConfigFileSpec into both files as well."

**Assistant interpretation:** Rename only the top-level config-file settings DTO and field in both build and app runtime spec models, while preserving provider/module config maps.

**Inferred user intent:** Remove ambiguity between config-file loading settings and provider module configuration.

**Commit (code):** a50de5d3d1d3b2c2b7b95d42efca9bd06bda0263 — "Clarify top-level build and runtime specs"

### What I did

- Renamed `ConfigSpec` to `ConfigFileSpec` in `buildspec` and `app` runtime specs.
- Renamed top-level `Config` fields to `ConfigFile` in `BuildSpec`, `RuntimeSpec`, generated embedded payloads, middleware setup, load/defaulting, validation, and tests.
- Changed top-level serialized tags from `config` to `configFile`.
- Preserved module instance and command provider `Config` maps and their `config` tags.
- Ran targeted tests:
  - `go test ./pkg/xgoja/... ./cmd/xgoja/... -count=1`

### Why

`Config` was overloaded: it could mean generated-command config-file discovery, provider module static config, or command provider config. `ConfigFile` is more precise for the Glazed config-file loading settings.

### What worked

- Targeted xgoja tests passed after the rename.
- The provider/module config maps remained separate and still serialize as `config`.

### What didn't work

- The first mechanical tag replacement accidentally changed module instance and command provider JSON tags from `config` to `configFile`. I reverted those tags so only the top-level config-file settings moved to `configFile`.

### What I learned

- The same word `config` appears in three distinct contexts in these files; broad replacements need follow-up inspection of struct tags.

### What was tricky to build

The main invariant was preserving provider configuration payload names while changing only generated-command config-file settings. I verified this by searching for remaining `ConfigSpec`, top-level `.Config`, and accidental `json:"configFile"` tags on module/provider config maps.

### What warrants a second pair of eyes

- Confirm that changing the external xgoja.yaml key from `config` to `configFile` is acceptable and should be documented as a deliberate schema change.
- Confirm whether validation check paths should also move from `config.*` to `configFile.*`.

### What should be done in the future

- Update xgoja user documentation and examples that still use top-level `config:` if this schema rename is intended to ship.

### Code review instructions

- Start with `cmd/xgoja/internal/buildspec/build_spec.go` and `pkg/xgoja/app/runtime_spec.go` and compare top-level `ConfigFile` with module/provider `Config` maps.
- Review `pkg/xgoja/app/middlewares.go` for runtime use of `RuntimeSpec.ConfigFile`.
- Validate with: `cd go-go-goja && go test ./pkg/xgoja/... ./cmd/xgoja/... -count=1`.

### Technical details

Command run:

```bash
cd go-go-goja && go test ./pkg/xgoja/... ./cmd/xgoja/... -count=1
```
