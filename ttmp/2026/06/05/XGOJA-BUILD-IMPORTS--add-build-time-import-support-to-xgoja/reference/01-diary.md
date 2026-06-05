---
Title: Diary
Ticket: XGOJA-BUILD-IMPORTS
Status: active
Topics:
    - goja
    - xgoja
    - go
    - javascript
    - database
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/xgoja/doc/06-buildspec-reference.md
      Note: Documented go.imports and SQL driver examples (commit fedde8d)
    - Path: cmd/xgoja/internal/buildspec/build_spec.go
      Note: |-
        Primary schema implementation target
        Added GoImportSpec and GoSpec.Imports (commit fedde8d)
    - Path: cmd/xgoja/internal/buildspec/validate.go
      Note: Added go.imports validation (commit fedde8d)
    - Path: cmd/xgoja/internal/generate/gomod.go
      Note: Added versioned extra import go.mod requirements (commit fedde8d)
    - Path: cmd/xgoja/internal/generate/templates.go
      Note: |-
        Primary generator implementation target
        Added ExtraImports template data and deduplication (commit fedde8d)
    - Path: ttmp/2026/06/05/XGOJA-BUILD-IMPORTS--add-build-time-import-support-to-xgoja/design-doc/01-xgoja-build-time-imports-design-and-implementation-guide.md
      Note: Design written in Step 1
ExternalSources: []
Summary: Chronological implementation diary for adding explicit build-time Go import support to xgoja.
LastUpdated: 2026-06-05T14:25:00-04:00
WhatFor: Track design, implementation, validation, and review guidance for xgoja build-time import support.
WhenToUse: Read before resuming or reviewing the XGOJA-BUILD-IMPORTS ticket.
---



# Diary

## Goal

This diary records the work to add explicit build-time import support to `xgoja.yaml`. The immediate motivating example is SQL drivers for the goja database module, but the design is intentionally generic: generated Go source should be able to include additional imports that exist for side effects, registration, or generated-code integration.

## Step 1: Ticket setup and design

This step created the ticket workspace and wrote the intern-oriented design guide before changing code. The design explains the difference between runtime database configuration and compile-time driver registration, then maps that distinction onto the xgoja buildspec, validation, generator template, and `go.mod` rendering layers.

The outcome is a concrete implementation plan for `go.imports`: decode it from YAML, validate import paths and aliases, render imports into generated source, add versioned entries to generated `go.mod`, document the feature, and test all generated target shapes.

### Prompt Context

**User prompt (verbatim):** "Create a new ticket for the imports tweaking in xgoja as discussed before. Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.


Keep a diary. Then add tasks to the ticket and implement them one by one, keeping a diary, and committing at appropriate intervals"

**Assistant interpretation:** Create a new go-go-goja docmgr ticket for xgoja build-time import support, write and upload a detailed intern-facing design, then implement the feature step by step with diary updates and commits.

**Inferred user intent:** The user wants xgoja generated binaries to support optional compile-time imports, especially SQL driver blank imports needed by database module configurations.

**Commit (code):** N/A — documentation setup step before implementation.

### What I did

- Created ticket `XGOJA-BUILD-IMPORTS` under the go-go-goja `ttmp` root.
- Created design doc `design-doc/01-xgoja-build-time-imports-design-and-implementation-guide.md`.
- Created this diary document.
- Inspected the existing buildspec schema, validation, template data, generated source templates, generated `go.mod` rendering, and buildspec docs.

### Why

- SQL drivers register themselves through Go package initialization, so runtime `driverName` config alone is not enough.
- xgoja currently imports providers and targets, but has no generic mechanism for side-effect imports.
- A design-first pass makes it clear which layer owns each responsibility.

### What worked

- `GoSpec` is the natural place for the feature because it already owns generated module settings such as Go version, module path, tags, and ldflags.
- Existing template data patterns for provider imports provide a straightforward model for extra imports.
- `RenderGoMod` already has deterministic require sorting and module path heuristics that can be reused.

### What didn't work

- N/A in this step. No code was changed yet.

### What I learned

- Provider-driven automatic imports would require a larger build-time contribution mechanism because providers are imported by generated code, not loaded by the generator.
- A simple explicit `go.imports` list solves the current database driver problem without adding provider introspection.

### What was tricky to build

- The tricky part is explaining why this belongs in buildspec `go.imports` rather than `modules[].config`. Runtime module config decides which driver name to pass to `sql.Open`; generated Go imports decide which packages are linked and initialized before that runtime call can work.

### What warrants a second pair of eyes

- Whether `go.imports[].module` should be optional with a heuristic default or required whenever `version` is set.
- Whether `replace` support should be added immediately or deferred.
- Whether source-fragment generation should emit extra imports in `providers.gen.go` or a separate generated file.

### What should be done in the future

- Upload the design/doc bundle to reMarkable after docmgr relations and doctor pass.
- Implement schema, validation, generation, docs, and tests in separate commits where practical.

### Code review instructions

- Start with `cmd/xgoja/internal/buildspec/build_spec.go` and `validate.go` for schema and validation.
- Then inspect `cmd/xgoja/internal/generate/templates.go`, templates, and `gomod.go` for generation behavior.
- Validate docs with `docmgr --root /home/manuel/workspaces/2026-05-27/rag-evaluation-system/go-go-goja/ttmp doctor --ticket XGOJA-BUILD-IMPORTS --stale-after 30`.

### Technical details

The proposed YAML shape is:

```yaml
go:
  imports:
    - import: github.com/lib/pq
      alias: _
      version: v1.10.9
```

Generated source should contain:

```go
import _ "github.com/lib/pq"
```

Generated `go.mod` should contain:

```go
require github.com/lib/pq v1.10.9
```

## Step 2: Implement explicit build-time imports

This step implemented the `go.imports` feature across the xgoja build pipeline. The buildspec schema now accepts extra Go imports, validation checks import paths and aliases, generated source templates render those imports, and generated `go.mod` includes versioned import modules when the YAML asks for them.

The feature is intentionally explicit. A user who wants Postgres support in a generated binary can add a blank import for `github.com/lib/pq`, while the database module's runtime config still controls the `driverName` and DSN passed to `database/sql`.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Continue from the design and implement the build-time import feature step by step, with tests, documentation, diary updates, and commits.

**Inferred user intent:** Make xgoja generated binaries able to compile optional side-effect imports such as SQL drivers directly from `xgoja.yaml`.

**Commit (code):** `fedde8dbe3aac23aaeb7290696c99cb3be039ecd` — "feat: add xgoja build-time imports"

### What I did

- Extended `cmd/xgoja/internal/buildspec/build_spec.go`:
  - added `GoImportSpec`;
  - added `GoSpec.Imports []GoImportSpec`.
- Extended `cmd/xgoja/internal/buildspec/validate.go`:
  - validates `go.imports[].import` is present and does not contain whitespace/quotes;
  - validates aliases are empty, `_`, `.`, or valid Go identifiers;
  - rejects duplicate non-blank/non-dot aliases;
  - rejects duplicate alias+import entries.
- Extended generator template data in `cmd/xgoja/internal/generate/templates.go`:
  - added `extraImport` and `ExtraImports`;
  - deduplicates extra imports by alias+path;
  - precomputes an alias prefix so templates can render both blank and normal imports cleanly.
- Updated generated source templates:
  - `templates/main.go.tmpl`;
  - `templates/runtime_package.go.tmpl`;
  - `templates/providers_fragment.go.tmpl`.
- Updated generated `go.mod` rendering in `cmd/xgoja/internal/generate/gomod.go`:
  - adds versioned `go.imports` entries to `require`;
  - uses `go.imports[].module` when present;
  - otherwise falls back to the existing module-root heuristic.
- Updated buildspec docs in `cmd/xgoja/doc/06-buildspec-reference.md` with SQL driver examples.
- Added tests for YAML loading, validation, rendered main/package/source-fragment imports, custom template data visibility, and `go.mod` requirements.

### Why

- SQL drivers must be linked into the generated Go binary before `database/sql` can open a configured driver name.
- Provider packages are imported by generated code, not dynamically inspected by the xgoja generator, so an explicit buildspec field is the smallest reliable solution.
- Extra imports need to be available in all generated target modes, not only `target.kind: xgoja`.

### What worked

- Focused tests passed:
  - `go test ./cmd/xgoja/internal/buildspec ./cmd/xgoja/internal/generate -count=1`
  - `go test ./cmd/xgoja/internal/buildspec ./cmd/xgoja/internal/generate ./cmd/xgoja ./pkg/xgoja/app ./pkg/xgoja/providers/host -count=1`
- Pre-commit hook passed lint and full tests before creating commit `fedde8dbe3aac23aaeb7290696c99cb3be039ecd`.

### What didn't work

- No implementation blocker occurred in this step.
- The pre-commit hook ran `go generate ./...`; Dagger emitted an HTTP HEAD error internally while resolving/building the bun demo container path, but it recovered through cached/container execution and completed successfully. Unlike the earlier database transaction commit, no local fallback failure path was needed here.

### What I learned

- The existing generator already cleanly separates main binary generation, package generation, and source-fragment generation; the same `packageTemplateData` can carry `ExtraImports` through all non-main modes.
- Custom template users need access to `ExtraImports`, but xgoja cannot force a custom template to render them. The feature exposes the data; custom template correctness remains the template author's responsibility.
- The existing `providerModulePath` heuristic is good enough for default module paths, but `go.imports[].module` is still needed for subpackage imports whose module path differs from the import path.

### What was tricky to build

- Rendering import aliases without malformed Go source required a precomputed `AliasPrefix`. If the template printed `{{ .Alias }} "{{ .Import }}"` directly, empty aliases could produce leading spacing that is usually okay but harder to reason about. `AliasPrefix` makes blank imports (`_ `), dot imports (`. `), named imports (`hooks `), and normal imports (`"path"`) explicit.
- Duplicate validation had to treat blank imports differently from named imports. Multiple `_` aliases are valid in a Go import block as long as they point to different packages, but duplicate named aliases such as `hooks` should be rejected.

### What warrants a second pair of eyes

- Whether the import path validation should be stricter. Current validation rejects empty paths and whitespace/quotes, but it does not fully parse Go import path syntax.
- Whether duplicate blank imports should be a validation error or silently deduplicated. Current behavior rejects duplicate alias+import entries and deduplicates in generation as a fallback.
- Whether source-fragment mode should put extra imports in `providers.gen.go` or a dedicated `imports.gen.go`. The current implementation uses `providers.gen.go` because it is always generated and already owns provider imports.

### What should be done in the future

- Consider `go.imports[].replace` if local driver/provider development needs it.
- Consider provider-shipped static manifests for automatic import suggestions, but keep explicit `go.imports` as the simple baseline.
- Consider adding a runnable example under `examples/xgoja` that configures a non-SQLite driver if a lightweight driver can be tested reliably.

### Code review instructions

- Start with `cmd/xgoja/internal/buildspec/build_spec.go` and `cmd/xgoja/internal/buildspec/validate.go` to review the public YAML contract.
- Review `cmd/xgoja/internal/generate/templates.go` for deduplication and alias prefix construction.
- Review the three templates to ensure extra imports appear in every generated mode.
- Review `cmd/xgoja/internal/generate/gomod.go` to verify versioned imports affect generated `go.mod` only when a version is provided.
- Validate with `go test ./cmd/xgoja/internal/buildspec ./cmd/xgoja/internal/generate -count=1` and the full pre-commit hook or `go test ./... -count=1`.

### Technical details

Example YAML:

```yaml
go:
  imports:
    - import: github.com/lib/pq
      alias: _
      version: v1.10.9
```

Generated import:

```go
import (
    _ "github.com/lib/pq"
)
```

Generated `go.mod` requirement:

```go
require github.com/lib/pq v1.10.9
```
