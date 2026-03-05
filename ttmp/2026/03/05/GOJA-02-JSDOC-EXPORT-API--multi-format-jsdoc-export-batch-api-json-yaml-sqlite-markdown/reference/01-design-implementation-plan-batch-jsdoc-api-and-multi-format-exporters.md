---
Title: 'Design + implementation plan: batch jsdoc API and multi-format exporters'
Ticket: GOJA-02-JSDOC-EXPORT-API
Status: active
Topics:
    - goja
    - tooling
    - architecture
    - analysis
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/cmd/goja-jsdoc/main.go
      Note: CLI wiring that will gain an export/batch command
    - Path: go-go-goja/pkg/jsdoc/extract/extract.go
      Note: Extractor used by batch builder
    - Path: go-go-goja/pkg/jsdoc/model/model.go
      Note: Base data model that exporters will serialize
    - Path: go-go-goja/pkg/jsdoc/model/store.go
      Note: DocStore structure and indexes
    - Path: go-go-goja/pkg/jsdoc/server/server.go
      Note: Existing browse API; will be extended with batch/export endpoints
ExternalSources: []
Summary: |
    Design/implementation plan to extend go-go-goja’s migrated jsdoc system with batch extraction (multiple inputs) and multi-format export (json, yaml, sqlite, markdown with ToC) for both CLI and HTTP API usage.
LastUpdated: 2026-03-05T04:04:59.991546837-05:00
WhatFor: |
    Provides an intern-friendly architecture map and phased implementation plan for adding exporters and batch extraction, including API contracts and SQLite schema sketches.
WhenToUse: Use when implementing GOJA-02-JSDOC-EXPORT-API or when deciding how to expose jsdoc data as files/streams (CLI) or as an API response (server).
---


# Design + implementation plan: batch jsdoc API and multi-format exporters

## Goal

Add “batch input” and “multi-format output” capabilities to the jsdoc system migrated into `go-go-goja` in GOJA-01.

We want both CLI and HTTP API support for:

- **Inputs**: one or more JavaScript files (and, optionally, directories) and/or in-memory file contents.
- **Outputs**: JSON, YAML, SQLite database, and Markdown documentation (with a Table of Contents).

This reference is written for an intern: it explains the moving parts, proposes concrete APIs, and provides a phased implementation plan with test strategy.

## Context

### Current state (what exists after GOJA-01)

After GOJA-01, go-go-goja contains:

- Data + extraction:
  - `go-go-goja/pkg/jsdoc/model/*`
  - `go-go-goja/pkg/jsdoc/extract/*`
- Server and watcher:
  - `go-go-goja/pkg/jsdoc/server/*` (existing browse endpoints like `/api/store`, `/api/symbol/...`)
  - `go-go-goja/pkg/jsdoc/watch/*`
- CLI:
  - `go-go-goja/cmd/goja-jsdoc/*` (currently “JSON parity” output; primarily for migration parity and debugging).

What is missing (the scope of this ticket):

1) **Batch extraction**: a first-class “build a store from many inputs” API with controllable error handling and input normalization.  
2) **Exporters**: reusable packages that serialize a `DocStore` to:
   - JSON (already easy, but needs standardization),
   - YAML,
   - SQLite,
   - Markdown with ToC.
3) **HTTP API for batch + export**: new endpoints that take one or multiple inputs and return outputs in one of those formats.

### Why this matters

The parity-mode `extract` command is useful for debugging, but real usage often wants:

- “Generate docs for these N files” (batch),
- “Give me a SQLite DB I can query later” (sqlite),
- “Generate Markdown I can read or publish” (markdown+ToC),
- “Provide a web API for on-demand export” (HTTP).

## Quick Reference

### Proposed package layout (additions)

```text
go-go-goja/pkg/jsdoc/
  batch/      # build a DocStore from multiple inputs
  export/     # format selection + convenience helpers
  exportyaml/ # yaml writer helpers
  exportsql/  # sqlite schema + writer
  exportmd/   # markdown generator + ToC
```

You can collapse into fewer packages if needed, but keep at least:

- one batch builder (`batch`),
- one format dispatcher (`export`),
- separate implementation files for sqlite and markdown (they’re the “big” ones).

### Data flow diagram (batch + export)

```text
inputs (paths or inline content)
        |
        v
  pkg/jsdoc/batch
   - resolves inputs
   - parses with pkg/jsdoc/extract
   - builds pkg/jsdoc/model.DocStore
        |
        v
  pkg/jsdoc/export
   - selects format
   - writes to io.Writer or files
        |
        +--> json/yaml (text)
        +--> markdown (text + ToC)
        +--> sqlite (file or stream)
```

### CLI proposal (extend `goja-jsdoc`)

Add an explicit “batch + export” command:

- `goja-jsdoc export --input file1.js --input file2.js --format json`
- `goja-jsdoc export --dir ./src --recursive --format sqlite --output out.db`
- `goja-jsdoc export --glob 'src/**/*.js' --format markdown --output docs.md --toc-depth 3`

### HTTP API proposal (new endpoints; keep existing routes stable)

Add new endpoints under a new namespace so existing doc-browser endpoints remain unchanged:

1) `POST /api/batch/extract`
- Input specs in request (paths and/or inline content).
- Response is `DocStore` (JSON by default; optionally YAML/Markdown).

2) `POST /api/batch/export`
- Input specs + format + options.
- Response body is formatted content (JSON/YAML/Markdown) or a SQLite file stream.

Security note: If the server accepts paths, it must restrict reads to a configured root directory to avoid path traversal.

## Detailed design

### 1) Batch inputs (paths vs inline)

Define an input model that works for both CLI and HTTP:

```go
type InputFile struct {
	// One of these must be set:
	Path    string // server reads from disk (restricted)
	Content []byte // server uses provided content

	// Optional: friendly name for reporting and output naming.
	DisplayName string
}

type BatchOptions struct {
	ContinueOnError bool
}

type BatchError struct {
	Input InputFile
	Err   error
}

type BatchResult struct {
	Store  *model.DocStore
	Errors []BatchError // filled when ContinueOnError=true
}
```

Key decisions:

- CLI will mostly supply `Path`.
- HTTP can support both `Path` and `Content`.
- If `ContinueOnError=false`, fail fast (simplest).
- If `ContinueOnError=true`, return partial store + per-input errors.

### 2) Export formats

Implement exporters as pure functions writing to `io.Writer`:

```go
type Format string
const (
	FormatJSON     Format = "json"
	FormatYAML     Format = "yaml"
	FormatSQLite   Format = "sqlite"
	FormatMarkdown Format = "markdown"
)

type ExportOptions struct {
	Format Format

	// Markdown
	TOCDepth int

	// SQLite
	SQLitePragma map[string]string
}

func Export(ctx context.Context, store *model.DocStore, w io.Writer, opts ExportOptions) error
```

#### JSON

Standardize:
- pretty vs compact,
- whether to export full `DocStore` (with indexes) vs `[]FileDoc`.

Recommendation:
- default to full `DocStore` for API responses,
- allow `--shape files|store` for CLI.

#### YAML

Mirror the chosen JSON shape using `gopkg.in/yaml.v3`.

#### SQLite

Suggested normalized schema (starter version):

```sql
CREATE TABLE packages (
  name TEXT PRIMARY KEY,
  title TEXT,
  category TEXT,
  guide TEXT,
  version TEXT,
  description TEXT,
  prose TEXT,
  source_file TEXT
);

CREATE TABLE symbols (
  name TEXT PRIMARY KEY,
  summary TEXT,
  docpage TEXT,
  prose TEXT,
  source_file TEXT,
  line INTEGER
);

CREATE TABLE symbol_tags (
  symbol_name TEXT,
  tag TEXT,
  PRIMARY KEY(symbol_name, tag)
);

CREATE TABLE symbol_concepts (
  symbol_name TEXT,
  concept TEXT,
  PRIMARY KEY(symbol_name, concept)
);

CREATE TABLE examples (
  id TEXT PRIMARY KEY,
  title TEXT,
  docpage TEXT,
  body TEXT,
  source_file TEXT,
  line INTEGER
);

CREATE TABLE example_symbols (
  example_id TEXT,
  symbol_name TEXT,
  PRIMARY KEY(example_id, symbol_name)
);
```

Implementation tips:
- use a transaction,
- prepared statements,
- create indexes where useful,
- if exporting via HTTP, write to a temp file then stream (simplest).

#### Markdown (+ ToC)

Start with single-file output:

- deterministic ToC generated from known section list (no markdown parsing needed),
- sections for Packages, Symbols, Examples,
- anchor-friendly headings (even if only “best effort”).

ToC depth:
- `--toc-depth` controls whether ToC includes only packages, or also symbols/examples.

## HTTP API contract (detailed)

### Endpoint: `POST /api/batch/export`

Request (JSON):

```json
{
  "inputs": [
    { "path": "src/a.js" },
    { "path": "src/b.js" }
  ],
  "format": "sqlite",
  "options": {
    "tocDepth": 2
  }
}
```

Response content types:
- `json` → `application/json`
- `yaml` → `application/yaml` (or `text/yaml`)
- `markdown` → `text/markdown`
- `sqlite` → `application/octet-stream` with `Content-Disposition: attachment; filename="docs.sqlite"`

Security constraints (required if `path` is supported):
- normalize with `filepath.Clean`,
- reject paths outside an allowed root (e.g., the server’s watched dir),
- optionally disallow absolute paths by default.

## Implementation plan (phased)

### Phase 1: Batch builder
- Add `pkg/jsdoc/batch`:
  - resolve inputs (paths -> bytes)
  - parse with `pkg/jsdoc/extract`
  - aggregate into `DocStore`
- Add tests with a couple of small fixtures.

### Phase 2: Exporters
- Add `pkg/jsdoc/export` dispatcher + format enums.
- Implement:
  - JSON exporter (standardize options)
  - YAML exporter
  - Markdown exporter (single-file + ToC)
  - SQLite exporter (schema + writer)
- Add tests:
  - JSON/YAML: decode back and spot-check fields
  - Markdown: contains key headings/ToC entries
  - SQLite: open DB and query row counts + sample symbol

### Phase 3: CLI integration
- Add `goja-jsdoc export` (or upgrade `extract`) to support:
  - multiple `--input` flags and/or positional args
  - `--dir` + `--recursive`
  - `--format` and format-specific flags
  - `--output` to write files
- Use Glazed for flags and help.

### Phase 4: HTTP API integration
- Extend `pkg/jsdoc/server` with new routes (keep old routes stable):
  - `/api/batch/extract`
  - `/api/batch/export`
- Add `httptest` coverage:
  - JSON export response
  - Markdown export response
  - SQLite response headers + basic size sanity check

## Usage Examples

### CLI (proposed)

```bash
go run ./go-go-goja/cmd/goja-jsdoc export --input a.js --input b.js --format yaml
go run ./go-go-goja/cmd/goja-jsdoc export --dir ./src --recursive --format sqlite --output docs.db
go run ./go-go-goja/cmd/goja-jsdoc export --dir ./src --format markdown --output docs.md --toc-depth 3
```

### HTTP API (proposed)

```bash
curl -sS -X POST http://127.0.0.1:8080/api/batch/export \
  -H 'Content-Type: application/json' \
  -d '{"inputs":[{"path":"src/a.js"},{"path":"src/b.js"}],"format":"markdown","options":{"tocDepth":2}}'
```

## Related

- Ticket index: `../index.md`
- Tasks: `../tasks.md`
- Diary: `02-diary.md`
- Predecessor ticket: GOJA-01-INTEGRATE-JSDOCEX (migration)
