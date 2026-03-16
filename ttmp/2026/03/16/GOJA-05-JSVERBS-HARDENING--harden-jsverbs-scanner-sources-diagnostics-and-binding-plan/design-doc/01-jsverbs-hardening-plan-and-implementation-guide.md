---
Title: jsverbs hardening plan and implementation guide
Ticket: GOJA-05-JSVERBS-HARDENING
Status: active
Topics:
    - architecture
    - goja
    - glazed
    - js-bindings
    - tooling
    - refactor
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/jsverbs/binding.go
      Note: |-
        Shared binding plan used by both schema generation and runtime invocation
        Shared schema/runtime binding contract introduced in this ticket
    - Path: pkg/jsverbs/command.go
      Note: |-
        Command compilation now consumes the shared binding plan
        Command compilation updated to consume binding plan
    - Path: pkg/jsverbs/jsverbs_test.go
      Note: |-
        Added raw-source, fs-backed, and failure-path coverage
        New raw-source
    - Path: pkg/jsverbs/model.go
      Note: |-
        New source and diagnostic model for the hardened jsverbs package
        Expanded registry model with diagnostics and source-file abstractions
    - Path: pkg/jsverbs/runtime.go
      Note: |-
        Runtime loader now serves in-memory sources and documents polling as v1 behavior
        Runtime source loader moved to in-memory registry-backed module serving
    - Path: pkg/jsverbs/scan.go
      Note: |-
        Strict AST literal parsing, scan diagnostics, and multi-source scanning entrypoints
        Strict AST literal parser and multi-source scanning entrypoints
ExternalSources: []
Summary: Hardening pass for jsverbs that replaced js-to-json rewriting with strict AST literal parsing, added diagnostics and multi-source scanning, unified binding logic, standardized errors, and expanded failure-path tests.
LastUpdated: 2026-03-16T16:43:22.752967662-04:00
WhatFor: Document the follow-up hardening work that turns the first jsverbs prototype into a stricter and more reusable package-level implementation.
WhenToUse: Use when reviewing or extending the hardened jsverbs package, especially around scanner input sources, diagnostics, or command/runtime binding behavior.
---


# jsverbs hardening plan and implementation guide

## Executive Summary

This ticket is the first cleanup pass after the initial jsverbs prototype. The original spike proved the concept, but it still had several prototype-level weaknesses:

- metadata parsing relied on rewriting JS object text into fake JSON,
- malformed metadata was silently dropped,
- the package only scanned directories on disk,
- schema generation and runtime argument binding had parallel hand-maintained logic,
- error style was inconsistent,
- failure-path tests were thin.

This hardening pass addresses those issues directly. The result is still recognizably the same subsystem, but it is a cleaner subsystem:

- metadata is now parsed from the tree-sitter AST rather than through `jsObjectToJSON`,
- scanner diagnostics are recorded and surfaced as `ScanError`,
- jsverbs can now scan from disk directories, generic `fs.FS` trees, and raw in-memory source strings,
- runtime loading no longer depends on disk reads after scanning,
- compile-time schema generation and runtime invocation now share one binding plan,
- promise polling is intentionally kept as version-1 behavior and clearly marked that way,
- new tests cover raw-source inputs, fs-backed inputs, and several error cases.

The package is still a v1 prototype, but it is now much clearer about what is static data, what is runtime behavior, and where errors should appear.

## Problem Statement

The first jsverbs prototype had the right architecture direction but the wrong internal pressure points. The most important problem was not that the package lacked features. It was that the parts of the system closest to the user contract were also the least strict.

That showed up in several places:

1. scanner metadata parsing was text-based and heuristic,
2. invalid metadata could silently disappear,
3. the runtime loader still assumed disk-backed files,
4. `command.go` and `runtime.go` each encoded their own interpretation of parameter bindings,
5. test coverage emphasized the success path more than the failure path.

For a package that generates command surfaces from source code, that is the wrong tradeoff. The scanner and binding logic need to be the most explicit parts of the implementation, not the loosest.

## What Changed

### 1. The package now supports multiple source origins

Before:

- `ScanDir(root string, ...)`

Now:

- `ScanDir(root string, ...)`
- `ScanFS(fsys fs.FS, root string, ...)`
- `ScanSource(path string, source string, ...)`
- `ScanSources(files []SourceFile, ...)`

Relevant files:

- `pkg/jsverbs/model.go`
- `pkg/jsverbs/scan.go`

The key design choice here is that all of these inputs normalize into the same internal model:

```text
Source origin
  -> sourceInput
  -> FileSpec{
       RelPath,
       ModulePath,
       Source,
       ...
     }
```

That means runtime loading no longer needs to care whether code originally came from the host filesystem, an embedded filesystem, or a raw string in memory.

### 2. The runtime loader now serves in-memory scanned sources

Before, `runtime.go` loaded source from disk again using real file paths.

Now, the registry stores scanned source bytes on each `FileSpec`, and the runtime loader resolves modules from `Registry.filesByModule`.

That gives the subsystem a clean virtual-module story:

- `ModulePath` is the canonical runtime path,
- `Source` is the canonical runtime source,
- `AbsPath` is only extra metadata when a source really came from disk.

This is what makes `ScanFS` and `ScanSource` practical instead of superficial.

### 3. Metadata parsing now walks AST literals directly

The old path was:

```text
tree-sitter object node
  -> raw source text
  -> jsObjectToJSON(...)
  -> json.Unmarshal(...)
  -> map[string]any
```

The new path is:

```text
tree-sitter literal node
  -> parseLiteralNode(...)
     -> parseObjectLiteral(...)
     -> parseArrayLiteral(...)
     -> decode string / number / bool / null
  -> map[string]any / []any / scalar
```

Supported metadata literals are intentionally narrow:

- objects
- arrays
- quoted strings
- template strings without substitutions
- numbers
- `true`, `false`, `null`

Unsupported metadata shapes now fail explicitly:

- calls
- identifiers as values
- spreads
- computed keys
- template substitutions
- other dynamic expressions

That is the right boundary. Metadata should be static data, not code.

### 4. The scanner now records diagnostics

The registry now carries `Diagnostics []Diagnostic`, and scan failures surface through `ScanError`.

That gives the package a more honest contract:

- invalid metadata is no longer silently ignored,
- callers can inspect diagnostics,
- strict callers still get an error by default.

This is especially important for future CLIs or editors that may want to show scan feedback without immediately crashing.

### 5. Schema generation and runtime binding now share one plan

The new file `pkg/jsverbs/binding.go` introduces a shared binding plan.

That plan resolves:

- parameter -> field,
- parameter -> section,
- parameter -> bind mode,
- extra verb fields that exist only in the schema,
- which shared sections are required.

`command.go` uses the plan to build Glazed sections and fields.
`runtime.go` uses the same plan to build JS call arguments.

That removes the previous "parallel interpretations" problem where compile-time and runtime had to stay synchronized by convention.

### 6. Promise polling remains, but is now explicitly marked as v1

The implementation still polls promises. That was requested and is reasonable for now.

The important change is not behavioral. It is communicative. `waitForPromise(...)` is now documented as intentionally simple version-1 behavior so future readers do not mistake it for the final async bridge design.

### 7. Error style is now standardized

The new package code now uses standard `fmt.Errorf(... %w ...)` wrapping rather than mixing in `github.com/pkg/errors`.

That makes the package more internally consistent and more in line with normal modern Go style.

## Architecture Diagram

```text
ScanDir / ScanFS / ScanSource / ScanSources
    |
    v
[scan.go]
- walk inputs
- parse with tree-sitter
- parse strict metadata literals
- record diagnostics
    |
    v
[model.go]
Registry + FileSpec + VerbSpec + Diagnostic
    |
    v
[binding.go]
VerbBindingPlan
- parameter bindings
- shared sections
- extra fields
    |
    +-----------------------------+
    |                             |
    v                             v
[command.go]                 [runtime.go]
build Glazed schema          build JS arguments
and command wrappers         and execute function
```

## API Reference

### Scanning APIs

#### `ScanDir`

Use when the JS lives on disk in a normal directory tree.

```go
registry, err := jsverbs.ScanDir("./testdata/jsverbs")
```

#### `ScanFS`

Use when the JS is packaged in an `embed.FS` or any other `fs.FS`.

```go
registry, err := jsverbs.ScanFS(embeddedFiles, ".", jsverbs.DefaultScanOptions())
```

#### `ScanSource`

Use when you only have one JS source string.

```go
registry, err := jsverbs.ScanSource("inline.js", sourceText)
```

#### `ScanSources`

Use when you want to construct a virtual module tree from in-memory files.

```go
registry, err := jsverbs.ScanSources([]jsverbs.SourceFile{
    {Path: "entry.js", Source: []byte(entry)},
    {Path: "lib/helper.js", Source: []byte(helper)},
})
```

### Diagnostics

The package now exposes:

- `Diagnostic`
- `DiagnosticSeverity`
- `ScanError`
- `Registry.ErrorDiagnostics()`

That means callers can choose between:

- fail-fast usage,
- or tooling/editor usage that inspects diagnostics.

### Binding Plan

The new shared types are:

- `BindingMode`
- `ParameterBinding`
- `ExtraFieldBinding`
- `VerbBindingPlan`

These are intentionally internal package mechanics right now, but they are the conceptual center of the hardening pass.

## Intern Onramp

If you are new to the hardened version of jsverbs, read these files in order:

1. `pkg/jsverbs/model.go`
2. `pkg/jsverbs/scan.go`
3. `pkg/jsverbs/binding.go`
4. `pkg/jsverbs/command.go`
5. `pkg/jsverbs/runtime.go`
6. `pkg/jsverbs/jsverbs_test.go`

Why this order works:

- `model.go` tells you what data the package believes in,
- `scan.go` tells you how that data is discovered,
- `binding.go` tells you how parameters become a single shared execution/schema contract,
- `command.go` and `runtime.go` then become much easier to understand because they are consumers of that plan rather than inventors of parallel rules.

## Design Decisions

### Decision 1: Use a virtual module path as the runtime identity

Rationale:

- it works for disk, `embed.FS`, and raw strings,
- it preserves relative `require()` semantics,
- it separates runtime identity from host filesystem identity.

Tradeoff:

- extra normalization logic is required up front,
- but the runtime becomes much cleaner afterward.

### Decision 2: Keep metadata literal support intentionally small

Rationale:

- metadata should be declarative,
- strict parsing keeps the scanner maintainable,
- explicit failure is better than heuristic acceptance.

Tradeoff:

- some dynamic JS patterns are rejected,
- but that is a deliberate boundary, not a missing feature.

### Decision 3: Keep promise polling for now

Rationale:

- the user explicitly wanted it kept,
- it is simple and easy to debug,
- it avoids a larger async bridge design detour in this ticket.

Tradeoff:

- it is not the final async story,
- so the code now says that clearly.

### Decision 4: Prefer one binding plan over repeated helper logic

Rationale:

- compile-time schema behavior and runtime argument behavior must not drift,
- one plan is easier to test than two coupled functions.

Tradeoff:

- introduces one extra internal concept,
- but removes a larger maintenance risk.

## Alternatives Considered

### Alternative: keep js-to-json and just add diagnostics

Rejected because it would preserve the most brittle part of the old scanner while only improving the error surface.

### Alternative: add `embed.FS` support by extracting embedded files to disk first

Rejected because it would keep the runtime tied to disk paths and would not solve the raw-string source case cleanly.

### Alternative: only factor out a few shared helpers instead of a binding plan

Rejected because helper extraction would reduce duplication but would not create one explicit contract for parameter binding semantics.

## Testing Strategy

The hardened package now validates:

- existing fixture discovery and command execution,
- raw-string scanning via `ScanSource` / `ScanSources`,
- generic `fs.FS` scanning via `ScanFS`,
- invalid metadata diagnostics,
- invalid bind references,
- unsupported object-pattern parameters without binds.

That testing mix is important because it covers both:

- new feature paths,
- and the exact failure surfaces that used to be vague or silent.

## Remaining Gaps

This ticket deliberately does not do everything.

Still open or future-worthy:

- a public incremental `Registry.AddSource(...)` API if callers want to mutate an existing registry instead of building from `ScanSources`,
- richer warning-level diagnostics beyond hard errors,
- package docs/help pages that describe the new source APIs explicitly,
- a less polling-heavy async bridge in a future version.

## Implementation Status

The requested items from the ticket have been implemented:

- strict AST literal parsing instead of js-to-json rewriting,
- scan diagnostics,
- raw source support,
- `fs.FS` support suitable for `embed.FS`,
- shared schema/runtime binding plan,
- explicit v1 polling comment,
- unified `fmt.Errorf` error style,
- failure-path tests.

## References

- `go-go-goja/pkg/jsverbs/model.go`
- `go-go-goja/pkg/jsverbs/scan.go`
- `go-go-goja/pkg/jsverbs/binding.go`
- `go-go-goja/pkg/jsverbs/command.go`
- `go-go-goja/pkg/jsverbs/runtime.go`
- `go-go-goja/pkg/jsverbs/jsverbs_test.go`
- `go-go-goja/cmd/jsverbs-example/main.go`
- `go-go-goja/ttmp/2026/03/16/GOJA-04-JS-GLAZED-EXPORTS--add-glazed-command-exporting-from-javascript/design-doc/02-js-verbs-prototype-postmortem-and-code-review.md`
