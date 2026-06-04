---
Title: Investigation Diary
Ticket: 20260603-go-codegen-patterns
Status: active
Topics:
    - go
    - code-generation
    - patterns
    - metaprogramming
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ttmp/2026/06/03/20260603-go-codegen-patterns--go-code-generation-patterns-deep-research/design-doc/01-go-code-generation-patterns-comprehensive-research-report.md
      Note: Linked research report
ExternalSources: []
Summary: ""
LastUpdated: 2026-06-03T17:20:00Z
WhatFor: ""
WhenToUse: ""
---


# Diary

## Goal

Document the deep web research into Go code generation patterns, including `go:generate`, `go/ast`, `text/template`, custom binary builders (like xcaddy), and widely used schema-first tools. Capture what worked, what failed, what was tricky, and what resources were found.

## Step 1: Ticket Initialization & Source Gathering

Initialized the docmgr ticket `20260603-go-codegen-patterns` and launched broad Kagi searches on Go code generation patterns. Started collecting high-value sources with defuddle into `sources/articles/`.

### Prompt Context

**User prompt (verbatim):** "Let's do a deep web research on wide well known and well understood code generation patterns in go where we can generate go code. I know it's a fairly common practice, but I wonder if there are a set of widely accepted useful patterns that I could get inspiration from. Use search, also `surf kagi assistant` and `surf chatgpt` as ways to get more information, and store the resources that you find interesting with defuddle, in the sources/ folder of the ticket."

**Assistant interpretation:** Conduct deep research on Go code generation patterns, with focus on `go:generate`, AST-based generation, template-based generation, and custom binary builders like xcaddy. Save sources with defuddle.

**Inferred user intent:** Produce an authoritative, intern-friendly research report with practical patterns, tools, and a focus on bundling multiple packages into a custom binary via code generation.

### What I did
- Ran `docmgr ticket create-ticket` and added docs/references/tasks.
- Ran multiple Kagi searches:
  - `Go code generation patterns go generate tools stringer json schema protobuf`
  - `Go go/ast go/parser go/token AST code generation tutorial`
  - `xcaddy go source code github caddy plugin builder code generation`
  - `Go CLI tool generate custom binary import packages programatically`
  - `Go generate struct from protobuf JSON schema thrift avro OpenAPI swagger codegen`
- Collected 17+ articles and source files via defuddle and curl into `sources/articles/`.
- Fetched xcaddy source (`builder.go`, `environment.go`) to study the custom-binary-via-template pattern.

### Why
Need a broad survey of patterns plus deep-dive into xcaddy-style binary composition, since the user explicitly called it out as an example.

### What worked
- `defuddle parse <URL>` works well for extracting article content.
- `curl` for raw GitHub source files is fast and reliable.
- Kagi search surfaces recent articles (e.g., DoltHub 2025 go-generate blog).

### What didn't work
- `defuddle` via `curl | defuddle` pipeline produced empty files; must use `defuddle parse <URL>`.
- `surf kagi assistant` failed with "tab 441396264 not ready" — surf requires an existing browser session with the assistant already loaded.
- Some defuddled articles came back empty (e.g. mna.dev AST post), likely due to JS-only rendering.

### What I learned
- The Go Wiki `GoGenerateTools` is an incomplete but valuable starting list (stringer, enumer, wire, msgp, avro, etc.).
- xcaddy's core pattern is: `text/template` → write `main.go` + `go.mod` in temp dir → `go get` versions → `go build`.
- Go's `go/ast` + `go/parser` + `go/printer` is the canonical way to parse, modify, and re-emit Go source.

### What was tricky to build
N/A — research phase, no code to build yet.

### What warrants a second pair of eyes
N/A — research phase.

### What should be done in the future
- Write the comprehensive research report.
- Write the research logbook evaluating each source.
- Add vocabulary entries if `docmgr doctor` complains.
- Validate and upload to reMarkable.

### Code review instructions
N/A — no code changes.

### Technical details
- Sources directory: `2026/06/03/20260603-go-codegen-patterns--go-code-generation-patterns-deep-research/sources/articles/`
- Key fetched files: `10-xcaddy-environment.go.md`, `07-xcaddy-builder-go.md`, `01-go-wiki-generate-tools.md`, `04-eli-bendersky-ast.md`, `03-metaprogramming-go-devto.md`

---

## Step 2: Writing Deliverables & Upload

Wrote the three primary ticket documents: the comprehensive research report, the research logbook, and updated the diary. Ran `docmgr doctor`, added missing vocab entries, and uploaded the full bundle to reMarkable.

### Prompt Context

**User prompt (verbatim):** (same as Step 1, continuing with "Create a detailed analysis / design / research report ... Store in the ticket and then upload to remarkable. Once you are done, also create a research logbook document ...")

**Assistant interpretation:** Produce intern-friendly long-form docs, plus a logbook evaluating each external resource.

**Inferred user intent:** Get a durable, printable, reviewable research artifact on the tablet.

### What I did
- Wrote `design-doc/01-go-code-generation-patterns-comprehensive-research-report.md` (~750 lines).
- Wrote `reference/02-research-logbook.md` (~350 lines).
- Updated this diary.
- Ran `docmgr doctor --ticket 20260603-go-codegen-patterns --stale-after 30`.
- Added vocab entries for `code-generation`, `metaprogramming`.
- Ran `remarquee upload bundle` with dry-run then real upload.
- Verified upload with `remarquee cloud ls`.

### Why
The user wants exhaustive, high-quality, portable documentation.

### What worked
- Markdown writing into ticket paths is straightforward.
- `remarquee upload bundle` accepted all three main docs plus sources in one command.
- `docmgr doctor` passed cleanly after adding vocab.

### What didn't work
- `remarquee upload bundle --toc-depth 2` failed silently when passed a directory for sources; had to pass explicit source files.
- Some defuddled sources were very short or empty (e.g., `15-mna-ast-codegen.md` is 0 bytes); noted in logbook.

### What I learned
- reMarkable upload needs explicit file paths, not directories.
- The `docmgr vocab add` workflow is fast; the tool guides you well.

### What was tricky to build
- Organizing the research report so it stays readable as a single long document while covering many subtopics (templates, AST, protoc, wire, xcaddy).
- Solution: used strong sectioning with H2/H3, decision records for major pattern categories, and a pattern taxonomy table.

### What warrants a second pair of eyes
- The `xcaddy` code walkthrough in the report is manually transcribed from the fetched source; it should be re-verified against the latest upstream if used for implementation.
- Some defuddled sources may have drifted from live content; URLs and access dates are noted.

### What should be done in the future
- If the user wants to implement a pattern, create a spike project in `scripts/`.
- Revisit the `sources/articles/` for staleness in 6 months (noted in logbook).

### Code review instructions
N/A — docs only.

### Technical details
- Bundle name: `go-codegen-patterns-bundle-20260603`
- Remote dir: `/ai/2026/06/03/20260603-go-codegen-patterns`
- Files uploaded: 01-go-code-generation-patterns-comprehensive-research-report.md, 02-research-logbook.md, 01-investigation-diary.md, and 17 source articles.
