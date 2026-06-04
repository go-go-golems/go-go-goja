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
    - Path: go-go-goja/cmd/xgoja/cmd_list_modules.go
      Note: CLI helper updated to use RuntimeSpec
    - Path: go-go-goja/cmd/xgoja/internal/buildspec/spec.go
      Note: |-
        Renamed buildspec Runtime/ModuleInstance/CommandProviderInstance DTO types to explicit *Spec names.
        Buildspec DTO type rename source
        Added build-time spec file documentation
    - Path: go-go-goja/cmd/xgoja/internal/buildspec/validate.go
      Note: Buildspec validation signature updated for RuntimeSpec
    - Path: go-go-goja/cmd/xgoja/internal/generate/main.go
      Note: Embedded runtime JSON payload updated to use renamed buildspec DTO types
    - Path: go-go-goja/pkg/xgoja/app/command_providers.go
      Note: Command provider helpers updated to use CommandProviderInstanceSpec
    - Path: go-go-goja/pkg/xgoja/app/factory.go
      Note: RuntimeFactory updated to use ModuleInstanceSpec
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

**Commit (code):** ddc12d26dcd84f68c186b29753cbb45c228e97f5 — "Clarify app runtime spec DTO names"

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
