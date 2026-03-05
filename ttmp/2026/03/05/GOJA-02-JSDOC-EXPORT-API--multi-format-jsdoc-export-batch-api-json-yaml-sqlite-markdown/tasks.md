# Tasks

## TODO

- [x] Review GOJA-01 baseline packages (`pkg/jsdoc/*`) and confirm extension points

### Phase 1: Batch builder (`pkg/jsdoc/batch`)
- [x] Define `InputFile`, `BatchOptions`, `BatchResult` types
- [x] Implement batch store builder:
  - [x] supports multiple inputs (paths)
  - [x] optional support for inline content (for HTTP usage)
  - [x] `ContinueOnError` behavior with per-input error capture
- [x] Add unit tests (small inline fixtures)

### Phase 2: Exporters (`pkg/jsdoc/export*`)
- [x] Add format enums and an `Export(ctx, store, writer, opts)` dispatcher
- [x] Implement JSON export (choose store vs files shapes)
- [x] Implement YAML export (mirror JSON shape)
- [x] Implement Markdown export:
  - [x] single-file output
  - [x] deterministic ToC generation
  - [x] `--toc-depth` option
- [x] Implement SQLite export:
  - [x] define schema
  - [x] transactional inserts + indexes
  - [x] tests that open DB and query counts/fields

### Phase 3: CLI (`cmd/goja-jsdoc`)
- [x] Add `export` command (or refactor `extract`) to support:
  - [x] multiple `--input` flags and/or positional args
  - [ ] `--dir` + `--recursive` + `--glob` (decide which are supported)
  - [x] `--format json|yaml|sqlite|markdown`
  - [x] `--output` path (file)
  - [x] format-specific flags (ToC depth, sqlite options)
- [ ] Decide whether to implement Glazed row output modes here (or explicitly defer)

### Phase 4: HTTP API (`pkg/jsdoc/server`)
- [x] Add new endpoints without breaking existing routes:
  - [x] `POST /api/batch/extract`
  - [x] `POST /api/batch/export`
- [x] Add input safety constraints (required if path input is supported):
  - [x] allowed root directory restriction
  - [x] reject traversal/absolute paths (unless explicitly enabled)
- [x] Add handler tests:
  - [x] JSON response
  - [x] Markdown response
  - [x] SQLite response headers + non-empty body

### Phase 5: Docs + runbooks
- [x] Update design doc with final API paths and CLI flags once implemented
- [x] Add a playbook for manual end-to-end export checks
