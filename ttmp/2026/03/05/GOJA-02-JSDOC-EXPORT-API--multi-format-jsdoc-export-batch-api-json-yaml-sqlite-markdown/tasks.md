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
- [ ] Add `export` command (or refactor `extract`) to support:
  - [ ] multiple `--input` flags and/or positional args
  - [ ] `--dir` + `--recursive` + `--glob` (decide which are supported)
  - [ ] `--format json|yaml|sqlite|markdown`
  - [ ] `--output` path (file)
  - [ ] format-specific flags (ToC depth, sqlite options)
- [ ] Decide whether to implement Glazed row output modes here (or explicitly defer)

### Phase 4: HTTP API (`pkg/jsdoc/server`)
- [ ] Add new endpoints without breaking existing routes:
  - [ ] `POST /api/batch/extract`
  - [ ] `POST /api/batch/export`
- [ ] Add input safety constraints (required if path input is supported):
  - [ ] allowed root directory restriction
  - [ ] reject traversal/absolute paths (unless explicitly enabled)
- [ ] Add handler tests:
  - [ ] JSON response
  - [ ] Markdown response
  - [ ] SQLite response headers + non-empty body

### Phase 5: Docs + runbooks
- [ ] Update design doc with final API paths and CLI flags once implemented
- [ ] Add a playbook for manual end-to-end export checks
