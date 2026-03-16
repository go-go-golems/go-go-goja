---
Title: Diary
Ticket: GOJA-05-JSVERBS-HARDENING
Status: active
Topics:
    - architecture
    - goja
    - glazed
    - js-bindings
    - tooling
    - refactor
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/pkg/jsverbs/scan.go
      Note: Scanner refactor from text-rewrite metadata parsing to strict AST literal parsing
    - Path: go-go-goja/pkg/jsverbs/binding.go
      Note: Shared binding plan added during the hardening pass
    - Path: go-go-goja/pkg/jsverbs/runtime.go
      Note: Runtime loader moved to in-memory source serving and kept polling with explicit v1 documentation
    - Path: go-go-goja/pkg/jsverbs/jsverbs_test.go
      Note: New failure-path and source-origin coverage added in this ticket
    - Path: go-go-goja/ttmp/2026/03/16/GOJA-05-JSVERBS-HARDENING--harden-jsverbs-scanner-sources-diagnostics-and-binding-plan/design-doc/01-jsverbs-hardening-plan-and-implementation-guide.md
      Note: Primary design and implementation guide for the hardening pass
ExternalSources: []
Summary: Chronological diary for the jsverbs hardening pass that added strict metadata parsing, diagnostics, multi-source scanning, a shared binding plan, and failure-path tests.
LastUpdated: 2026-03-16T16:43:22.758921591-04:00
WhatFor: Preserve the reasoning, commands, and implementation path for the follow-up jsverbs hardening work after the initial prototype ticket.
WhenToUse: Use when reviewing how the jsverbs hardening pass was implemented or continuing similar cleanup work later.
---

# Diary

## Goal

This diary records the follow-up hardening pass for jsverbs after the initial prototype. The goal of the step was not to expand surface area arbitrarily, but to clean the weakest internal seams: metadata parsing, source loading, diagnostics, binding logic, error consistency, and missing failure-path tests.

## Step 1: Create The Follow-Up Ticket And Harden The Package

The follow-up work started from a clear implementation list provided by the user: remove the js-to-json rewrite, support raw strings and `embed.FS`-style scanning, add diagnostics, unify binding logic, keep promise polling but mark it as v1, standardize errors, and add failure-path tests. That was cohesive enough to treat as a single hardening ticket rather than an unstructured patch series.

I created a new ticket first so the implementation, rationale, and later validation evidence would have a clean home separate from the original design/research ticket. After that, I refactored the package around a virtual module-path model so that all source origins could share the same runtime loader and command discovery path.

### Prompt Context

**User prompt (verbatim):**

```text
ok, so there's that to fix,

- then i also want to add adding a raw JS string to the registry, in case we don't load from a dir
- I also want to load from an embed.FS
- scan diagnostics
- get clear schema + runtime binding logic
- keep polling as it is and mark it as a first version
- unify error style
- add error case failure tests

So create a new ticket, add detailed tasks for all of this, is then go through them
```

**Assistant interpretation:** Create a follow-up hardening ticket for jsverbs, turn the requested cleanup items into explicit ticket tasks, implement the scanner/runtime/model refactor in code, validate it with tests and manual runs, and document the result in the new ticket.

**Inferred user intent:** Move the jsverbs package from prototype-grade internals toward a cleaner version-1 implementation without changing the overall product direction.

**Commit (code):** `4e4e893` — `Add jsverbs prototype runner and shared docs`

### What I did

- Created a new ticket:
  - `GOJA-05-JSVERBS-HARDENING`
- Added tasks for:
  - strict AST literal parsing,
  - diagnostics,
  - raw source + `fs.FS` support,
  - shared binding plan,
  - v1 polling note,
  - unified error style,
  - failure-path tests.
- Refactored the package model in `pkg/jsverbs/model.go` to add:
  - `SourceFile`,
  - `Diagnostic`,
  - `ScanError`,
  - in-memory source bytes on `FileSpec`,
  - module-path keyed file lookup on `Registry`.
- Replaced the scanner in `pkg/jsverbs/scan.go` so it now:
  - supports `ScanDir`, `ScanFS`, `ScanSource`, and `ScanSources`,
  - parses metadata literals directly from the tree-sitter AST,
  - records diagnostics instead of silently dropping bad metadata.
- Added `pkg/jsverbs/binding.go` so both command compilation and runtime invocation now consume the same binding plan.
- Updated `pkg/jsverbs/runtime.go` to:
  - load source from the registry rather than from disk,
  - work with virtual module paths for raw and fs-backed sources,
  - keep polling with an explicit version-1 comment.
- Added tests in `pkg/jsverbs/jsverbs_test.go` for:
  - raw JS string scanning,
  - `fs.FS` scanning,
  - invalid metadata diagnostics,
  - invalid bound sections,
  - unsupported object-pattern parameters without binds.
- Validated with:
  - `go test ./go-go-goja/pkg/jsverbs`
  - `go test ./go-go-goja/cmd/jsverbs-example`
  - `go run ./go-go-goja/cmd/jsverbs-example --dir ./go-go-goja/testdata/jsverbs list`
  - `go run ./go-go-goja/cmd/jsverbs-example --dir ./go-go-goja/testdata/jsverbs basics list-issues go-go-golems/go-go-goja --state closed --labels bug --labels docs`

### Why

- The old scanner and runtime had the right broad architecture but the wrong kinds of shortcuts in the places closest to the user contract.
- Supporting raw sources and `fs.FS` cleanly required a source model change anyway, so it made sense to fix diagnostics and runtime loading around the same underlying abstraction.
- Binding-plan cleanup was worth doing in the same pass because scanner strictness without compile/runtime alignment would still leave a major drift risk in place.

### What worked

- The virtual module-path model simplified more than one problem at once:
  - runtime loading,
  - raw source support,
  - fs-backed support,
  - relative `require()` behavior.
- The direct AST literal parser was smaller and easier to reason about than the previous text-to-JSON rewrite.
- The shared binding plan made the compile/runtime contract much easier to explain.
- The failure tests caught the exact behavior that the user wanted tightened.

### What didn't work

- My first refactor attempt changed `engine.DefaultRegistryModules()` to a variadic form in `runtime.go`, but the actual API returns a single composite `ModuleSpec`. The initial test run failed with:

```text
cannot use engine.DefaultRegistryModules() (value of interface type engine.ModuleSpec) as []engine.ModuleSpec value in argument to ...WithModules
```

- I fixed that by restoring the non-variadic call.

### What I learned

- The best way to add raw string and `embed.FS` support was not a special-case code path. It was making the registry itself the source of truth for runtime source bytes.
- A strict literal parser is less code than a permissive text-rewrite system once the package already has an AST.
- The shared binding plan is the real center of gravity for future jsverbs changes. It is the place where command schema and runtime invocation should keep meeting.

### What was tricky to build

- The main tricky part was making the source model generic enough for all three input origins without breaking the relative `require()` behavior already proven by the first prototype.
- The key insight was to use canonical module paths like `/nested/entry.js` as the runtime identity, while keeping `AbsPath` only as optional source metadata for disk-backed inputs.
- That preserved the old runtime behavior while decoupling runtime loading from the host filesystem.

### What warrants a second pair of eyes

- The exact literal subset supported by the new metadata parser.
- Whether the current `ScanError` / diagnostics balance is the right API for callers that may want non-fatal warnings later.
- Whether a public incremental `Registry.AddSource(...)` API is still worth adding even though `ScanSource` and `ScanSources` now cover the main non-directory use cases.

### What should be done in the future

- Consider updating shared jsverbs help docs under `pkg/doc` so they explicitly document `ScanFS`, `ScanSource`, diagnostics, and the stricter metadata rules.
- If async command complexity grows, revisit the polling bridge.
- Consider a richer diagnostics API if editors or interactive tools want warning-level feedback without failing the scan.

### Code review instructions

- Start with:
  - `pkg/jsverbs/model.go`
  - `pkg/jsverbs/scan.go`
  - `pkg/jsverbs/binding.go`
  - `pkg/jsverbs/runtime.go`
- Then confirm the new behaviors through:
  - `pkg/jsverbs/jsverbs_test.go`
- Finally, verify the example runner still works with:
  - `go run ./go-go-goja/cmd/jsverbs-example --dir ./go-go-goja/testdata/jsverbs list`

### Technical details

- New scanning entrypoints:

```go
ScanDir(root string, opts ...ScanOptions)
ScanFS(fsys fs.FS, root string, opts ...ScanOptions)
ScanSource(path string, source string, opts ...ScanOptions)
ScanSources(files []SourceFile, opts ...ScanOptions)
```

- New diagnostic types:

```go
type Diagnostic struct { ... }
type ScanError struct { Diagnostics []Diagnostic }
```

- New shared binding-plan file:

```text
pkg/jsverbs/binding.go
```
