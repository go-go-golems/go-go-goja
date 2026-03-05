---
Title: Diary
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
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-03-05T14:55:00-05:00
WhatFor: |
    Chronological implementation diary for the scoped jsdoc path refactor, including
    prompts, code changes, validation steps, and follow-up review notes.
WhenToUse: |
    Use when continuing GOJA-03 work or reviewing how the scoped filesystem design was
    implemented step by step.
---

# Diary

## Goal

Keep a detailed chronological record of the GOJA-03 refactor so a new intern can understand why the CodeQL warning exists, how the path-scoping design works, and how to review the resulting implementation.

## Step 1: Create GOJA-03 ticket and write the implementation plan

This step creates a dedicated ticket for the path-scoping refactor and writes the design/implementation plan before changing the code. The goal is to avoid “security by ad hoc patching” and instead make the filesystem trust boundary explicit in the design first.

### Prompt Context

**User prompt (verbatim):** "yes, craete a new docmgr ticket for this, add an implementation plan document and tasks, then go work on the tasks one by one, committing at appropriate intervals, keeping a diary."

**Assistant interpretation:** Create a new docmgr ticket for the CodeQL/path-scoping work, document the plan and task breakdown, then implement the refactor incrementally with commits and diary entries.

**Inferred user intent:** Turn the security discussion into a tracked engineering task with explicit design rationale, reviewable implementation steps, and durable documentation.

**Commit (code):** N/A

### What I did
- Created ticket `GOJA-03-SCOPED-JSDOC-PATHS`.
- Added this diary and a design/implementation plan document.
- Decided the target design should use `fs.FS` scoping for untrusted API paths rather than pushing more policy into the extractor itself.

### Why
- CodeQL is flagging a structural problem, not just a missing string check. The design should fix the abstraction boundary.

### What worked
- Existing GOJA-01 and GOJA-02 ticket structure provided a clear template for the new ticket.

### What didn't work
- N/A

### What I learned
- N/A

### What was tricky to build
- The main design tradeoff is not “can we validate paths?” because we already do; it is “where should the security boundary live so the code structure itself is convincing?”

### What warrants a second pair of eyes
- Whether `ParseFile(path)` should remain as a trusted/local helper or be deprecated immediately.

### What should be done in the future
- Implement the scoped extractor and refactor server/batch to use it.

### Code review instructions
- Start with the design document in `reference/01-design-implementation-plan-scoped-jsdoc-paths.md`.

### Technical details
- N/A

## Step 2: Implement scoped extractor and batch path-parser injection

This step adds the reusable code needed to separate “parse these bytes” from “read from this filesystem path.” The extractor now supports reading from any `fs.FS`, and the batch builder can accept a caller-provided path parser instead of always calling the direct host-path helper.

This is the core architectural change in the ticket because it moves the security boundary into the code structure. The API caller no longer needs to rely on “validated path plus generic file helper” as the only safety story.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Implement the low-level building blocks for scoped parsing before changing the server request flow.

**Inferred user intent:** Fix the CodeQL issue at the abstraction level, not with a shallow input filter.

**Commit (code):** 80f6e1b — "GOJA-03: scope jsdoc API file parsing"

### What I did
- Added `extract.ParseFSFile(fsys fs.FS, path string)` in `pkg/jsdoc/extract/extract.go`.
- Added `pkg/jsdoc/extract/scopedfs.go`:
  - `extract.NewScopedFS(root)`
  - symlink-aware path resolution
  - explicit rejection of absolute paths, traversal, invalid paths, and outside-root targets
- Added extractor tests in `pkg/jsdoc/extract/scopedfs_test.go`:
  - `ParseFSFile` fixture parsing
  - symlink escape rejection
- Extended `pkg/jsdoc/batch/batch.go` with `BatchOptions.ParsePath`.
- Added batch test coverage to prove custom path parsing is actually used.

### Why
- Without this layer, the server would still have to pass validated path strings into a generic host-path reader.

### What worked
- The `fs.FS` split fit naturally with the existing `ParseSource` design.
- Focused tests for `extract` and `batch` passed immediately after the refactor.

### What didn't work
- The initial design idea of using `os.DirFS` directly is not strong enough against symlink escapes inside the allowed root, so the implementation had to tighten into a custom scoped filesystem wrapper.

### What I learned
- The right boundary is not just “relative path only”; it is “real resolved path remains inside the intended root.”

### What was tricky to build
- Preserving extractor reusability while still making the API path flow structurally safe required two separate abstractions:
  - generic `ParseFSFile`
  - security-aware `ScopedFS`

### What warrants a second pair of eyes
- Whether we should deprecate `ParseFile(path)` entirely in a follow-up instead of keeping it as a trusted/local convenience.

### What should be done in the future
- Refactor the server to use the new scoped path parser and close the API dataflow loop.

### Code review instructions
- Review:
  - `pkg/jsdoc/extract/extract.go`
  - `pkg/jsdoc/extract/scopedfs.go`
  - `pkg/jsdoc/batch/batch.go`

### Technical details
- `ScopedFS` uses `filepath.EvalSymlinks` on both the root and the candidate file path before enforcing containment.

## Step 3: Refactor the server and fix the review findings

This step changes the HTTP API flow so accepted request paths remain relative and are parsed through the scoped filesystem instead of being converted into absolute host paths. It also folds in two follow-up issues raised during review:

- reject symlink escapes instead of relying on lexical prefix checks,
- reject mixed `path` + `content` inputs as bad requests.

### Prompt Context

**User prompt (verbatim):** "Also handle these as parts of the ticket."

**Assistant interpretation:** Include the two server review findings in the same GOJA-03 implementation and treat them as first-class tasks rather than follow-up notes.

**Inferred user intent:** Close the ticket with the actual security-sensitive edge cases handled, not just the initial structural refactor.

**Commit (code):** 80f6e1b — "GOJA-03: scope jsdoc API file parsing"

### What I did
- Updated `pkg/jsdoc/server/batch_handlers.go`:
  - request path normalization now returns cleaned relative paths
  - batch handlers create a scoped path parser using `extract.NewScopedFS(s.dir)`
  - mixed `path` + `content` request items now return `400 Bad Request`
  - path-policy violations from scoped reads are mapped to bad-request responses
- Extended `pkg/jsdoc/server/batch_handlers_test.go`:
  - store file path remains relative (`a.js`)
  - mixed `path` + `content` is rejected
  - symlink escape via `linked/outside.js` is rejected
- Ran focused tests:
  - `go test ./pkg/jsdoc/extract ./pkg/jsdoc/batch ./pkg/jsdoc/server -count=1`
- Pre-commit hooks also passed on the code commit:
  - `golangci-lint`
  - `go generate ./...`
  - `go test ./...`

### Why
- The server was the only untrusted path entrypoint, so this is where the dataflow fix actually matters.

### What worked
- Keeping request paths relative produces cleaner `FileDoc.FilePath` values and a better separation between client input and host filesystem layout.
- The review findings fit naturally into the same server refactor instead of requiring a second ticket.

### What didn't work
- N/A

### What I learned
- Treating mixed `path` + `content` as a transport validation failure instead of a build failure makes the API behavior much cleaner and avoids false “500” responses for malformed input.

### What was tricky to build
- Error classification: once path checks happen inside the scoped filesystem read path, the handler still needs to surface those failures as request errors instead of generic internal errors.

### What warrants a second pair of eyes
- Whether missing files requested through the batch API should remain `500` or should also be treated as a request error class; the current change only maps explicit path-policy failures to `400`.

### What should be done in the future
- Decide whether missing/unreadable API-requested files should become `400` in a follow-up API cleanup.

### Code review instructions
- Review:
  - `pkg/jsdoc/server/batch_handlers.go`
  - `pkg/jsdoc/server/batch_handlers_test.go`
- Validate:
  - `go test ./pkg/jsdoc/extract ./pkg/jsdoc/batch ./pkg/jsdoc/server -count=1`

### Technical details
- The symlink escape fix specifically addresses the case where a directory inside the allowed root is a symlink to content outside the root.
