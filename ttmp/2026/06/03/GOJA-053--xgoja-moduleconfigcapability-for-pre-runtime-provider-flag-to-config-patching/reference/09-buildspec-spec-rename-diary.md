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
    - Path: go-go-goja/cmd/xgoja/cmd_list_modules.go
      Note: CLI helper updated to use RuntimeSpec
    - Path: go-go-goja/cmd/xgoja/internal/buildspec/spec.go
      Note: |-
        Renamed buildspec Runtime/ModuleInstance/CommandProviderInstance DTO types to explicit *Spec names.
        Buildspec DTO type rename source
    - Path: go-go-goja/cmd/xgoja/internal/buildspec/validate.go
      Note: Buildspec validation signature updated for RuntimeSpec
    - Path: go-go-goja/cmd/xgoja/internal/generate/main.go
      Note: Embedded runtime JSON payload updated to use renamed buildspec DTO types
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
