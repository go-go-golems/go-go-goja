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
    - Path: cmd/xgoja/internal/buildspec/build_spec.go
      Note: Primary schema implementation target
    - Path: cmd/xgoja/internal/generate/templates.go
      Note: Primary generator implementation target
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
