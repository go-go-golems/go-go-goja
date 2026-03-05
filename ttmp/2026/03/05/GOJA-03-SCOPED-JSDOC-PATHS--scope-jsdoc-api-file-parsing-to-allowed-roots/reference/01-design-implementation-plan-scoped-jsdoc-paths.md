---
Title: 'Design + implementation plan: scope jsdoc file parsing to allowed roots'
Ticket: GOJA-03-SCOPED-JSDOC-PATHS
Status: active
Topics:
    - goja
    - tooling
    - security
    - architecture
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/pkg/jsdoc/extract/extract.go
      Note: Current exported file-reading entrypoint that CodeQL flags
    - Path: go-go-goja/pkg/jsdoc/batch/batch.go
      Note: Batch builder that currently delegates file paths to extract.ParseFile
    - Path: go-go-goja/pkg/jsdoc/server/batch_handlers.go
      Note: API request validation and path scoping logic
    - Path: go-go-goja/cmd/goja-jsdoc/export_command.go
      Note: Trusted CLI caller that still benefits from explicit file source boundaries
ExternalSources: []
Summary: |
    Implementation plan for replacing generic path-based jsdoc parsing in untrusted flows
    with scoped filesystem-backed parsing so the API security boundary is encoded in the
    code structure rather than only in call-site validation.
LastUpdated: 2026-03-05T14:55:00-05:00
WhatFor: |
    Provides an intern-friendly plan for addressing the CodeQL uncontrolled path warning
    by introducing an fs.FS-scoped extraction API and refactoring batch/server code to use it.
WhenToUse: |
    Use when implementing or reviewing GOJA-03-SCOPED-JSDOC-PATHS, or when deciding how
    jsdoc extraction should distinguish trusted local file access from API-scoped file access.
---

# Design + implementation plan: scope jsdoc file parsing to allowed roots

## Goal

Refactor the jsdoc extraction path so untrusted API input no longer flows into a generic “read any path from disk” helper.

The desired end state is:

- the HTTP API can only parse files through a filesystem scope rooted at the server’s allowed directory,
- the security boundary is visible in the types and function signatures,
- the reusable parsing code remains clean,
- and CodeQL has a much stronger story than “we validated at the call site, trust us.”

## Problem statement

Today the extractor exposes a convenience function:

```go
func ParseFile(path string) (*model.FileDoc, error)
```

Its implementation is straightforward:

- `os.ReadFile(path)`
- then `ParseSource(path, src)`

That is fine for trusted local callers such as CLI tools or tests, but it creates a structural problem for untrusted inputs:

- the HTTP API accepts user-provided paths,
- validates/scopes them in `pkg/jsdoc/server/batch_handlers.go`,
- but still eventually passes a string path to a generic file reader.

This makes CodeQL report “uncontrolled data used in path expression”, because the sink is an exported file reader that accepts arbitrary path strings.

## Current architecture

### Actual data flow today

```text
HTTP request JSON
   |
   v
server.inputsFromRequest()
   - validate path is relative
   - clean path
   - ensure path resolves under server root
   |
   v
batch.BuildStore()
   |
   v
extract.ParseFile(path)
   |
   v
os.ReadFile(path)
```

### Why this is not ideal

The validation exists, but it lives “next to” the untrusted entrypoint rather than “inside” the abstraction boundary.

That means:

- reviewers must inspect multiple layers to prove safety,
- future callers can accidentally bypass the policy,
- CodeQL sees a path sink exposed through a generic helper,
- and the batch builder cannot express “this file came from a scoped filesystem” as a first-class concept.

## Design decision

Introduce an `fs.FS`-scoped extractor entrypoint and make API-facing code use it.

### Proposed API shape

Keep `ParseSource(name, src)` as the lowest-level reusable parser.

Add:

```go
func ParseFSFile(fsys fs.FS, path string) (*model.FileDoc, error)
```

Optionally keep:

```go
func ParseFile(path string) (*model.FileDoc, error)
```

but treat it as a trusted/local convenience wrapper, not the primary path for untrusted input.

### Why `fs.FS`

`fs.FS` is the right abstraction here because:

- it cleanly expresses “read from this bounded filesystem view,”
- `os.DirFS(root)` gives us an allowlisted subtree for free,
- tests become easier because we can use in-memory or fixture-backed filesystem implementations,
- the extractor package stays reusable and does not need to know about HTTP/server policy.

## Proposed data flow after refactor

```text
HTTP request JSON
   |
   v
server.inputsFromRequest()
   - validate request-level path syntax
   - keep path relative
   |
   v
batch.BuildStoreFromFS(fsys, inputs)
   |
   v
extract.ParseFSFile(fsys, relativePath)
   |
   v
fs.ReadFile(fsys, relativePath)
```

Trusted CLI/local path flow can remain:

```text
CLI args
   |
   v
batch.BuildStore(...)
   |
   v
extract.ParseFile(path)
   |
   v
os.ReadFile(path)
```

This gives us a clear split:

- API: scoped FS
- local CLI/tests: direct filesystem path

## Refactor plan

### 1. Extractor layer

Add a new helper in `pkg/jsdoc/extract/extract.go`:

- `ParseFSFile(fsys fs.FS, path string)`

Behavior:

- reads file bytes via `fs.ReadFile`,
- rejects empty path,
- passes the same `path` string to `ParseSource` so `FileDoc.FilePath` stays meaningful.

Do not move path-policy checks into the extractor itself beyond basic input sanity:

- no “must be relative to root” logic here,
- no server-specific configuration,
- no transport/security assumptions.

The extractor should stay a parser, not a policy engine.

### 2. Batch layer

The batch builder currently models “path or content” only. Extend it so callers can choose a file-reading strategy.

Recommended direction:

```go
type FileParser interface {
    ParsePath(path string) (*model.FileDoc, error)
    ParseContent(name string, content []byte) (*model.FileDoc, error)
}
```

or, more simply:

```go
type PathParser func(path string) (*model.FileDoc, error)
```

with `ParseSource` used directly for inline content.

Pragmatic choice for this repo: use a function option or an extra builder variant rather than adding a heavy interface.

Example:

```go
type BatchOptions struct {
    ContinueOnError bool
    ParsePath       func(path string) (*model.FileDoc, error)
}
```

Defaults:

- if `ParsePath == nil`, use `extract.ParseFile`

Scoped API caller:

- set `ParsePath` to a closure around `extract.ParseFSFile(os.DirFS(root), path)`

This keeps the batch package small and avoids over-abstracting.

### 3. Server layer

In `pkg/jsdoc/server/batch_handlers.go`:

- keep request validation,
- but stop converting accepted paths into absolute host paths,
- instead keep them as cleaned relative paths,
- create `fsys := os.DirFS(s.dir)`,
- call batch builder with a `ParsePath` override that uses `extract.ParseFSFile(fsys, path)`.

Important detail:

- `resolvePath` should probably become “normalize relative path” rather than “return absolute path”.

That change is desirable because absolute paths are precisely what blur the scoped boundary.

### 4. CLI layer

The CLI is a trusted/local caller. It can continue to use direct path parsing.

However, it will probably need minor adjustment if batch options change.

We should keep CLI semantics intact:

- positional files,
- `--input`,
- `--dir` / `--recursive`,
- direct host filesystem access.

### 5. Tests

Add/adjust tests in three places:

- extractor:
  - `ParseFSFile` reads fixtures correctly
- batch:
  - custom `ParsePath` hook is used
  - inline content still works
- server:
  - batch API still succeeds for valid relative paths
  - traversal/absolute paths still fail
  - path returned to parser is relative, not absolute host path

## Pseudocode

### Extractor

```pseudo
function ParseFSFile(fsys, path):
  if path is empty:
    error
  src = fs.ReadFile(fsys, path)
  return ParseSource(path, src)
```

### Batch

```pseudo
function BuildStore(inputs, opts):
  parsePath = opts.ParsePath or extract.ParseFile

  for input in inputs:
    if input has content:
      fd = extract.ParseSource(bestName(input), input.content)
    else:
      fd = parsePath(input.path)
```

### Server

```pseudo
function inputsFromRequest(requestInputs):
  for each requestInput:
    if content:
      keep as inline input
    else:
      path = cleanRelativePath(requestInput.path)
      ensure not absolute
      ensure not traversal
      keep relative path only

function handleBatchExport():
  fsys = os.DirFS(serverRoot)
  batch.BuildStore(inputs, ParsePath = func(path):
    return extract.ParseFSFile(fsys, path))
```

## Tradeoffs

### Benefits

- Stronger security boundary in code structure
- Better static-analysis story
- Cleaner separation between parsing and access policy
- Easier testing of scoped filesystem behavior

### Costs

- Slightly more plumbing in batch/server
- Need to update some tests and docs
- `ParseFile` remains a potentially flagged helper unless we clearly narrow its role or deprecate it

## Open design choice

We have two reasonable end states:

1. Keep `ParseFile(path)` for trusted/local callers, but ensure API paths never use it.
2. Deprecate `ParseFile(path)` entirely and force all file reading through either:
   - `ParseFSFile(fsys, path)`, or
   - explicit caller-side `os.ReadFile + ParseSource`.

Recommendation:

- implement option 1 in this ticket,
- document `ParseFile` as trusted/local convenience,
- revisit full deprecation only if CodeQL still reports the exported sink as problematic after dataflow changes.

## Implementation phases

### Phase 1: Design the scoped path contract

- add ticket docs
- define the target API and data flow
- clarify absolute-vs-relative path expectations for server callers

### Phase 2: Add extractor + batch support

- add `ParseFSFile`
- add batch path-parser injection
- test the new path-reading strategy

### Phase 3: Refactor server to use scoped filesystem parsing

- keep validated relative paths
- use `os.DirFS(s.dir)` in batch handlers
- verify no absolute host path leaks into extractor path reads

### Phase 4: Validate and document

- run focused tests
- update diary/changelog/tasks
- run `docmgr doctor`

## Review checklist

- Does any API path still flow into `os.ReadFile` directly?
- Are server request paths kept relative after validation?
- Does CLI behavior remain unchanged?
- Do tests cover both valid relative paths and rejected traversal cases?
- Is the parsing package still reusable outside HTTP?

