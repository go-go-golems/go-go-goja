# Tasks

## TODO

### Phase 0: Evidence + parity targets (docs)

- [x] Read `jsdocex/` code and document behavior parity targets (see design guide)
- [x] Design `go-go-goja/pkg/jsdoc` package layout + public API (see design guide)
- [x] Add a short “Acceptance Criteria” section to the design guide (exact parity checklist)

### Phase 1: New reusable packages in `go-go-goja/pkg/jsdoc`

**1.1 Model + store**
- [x] Create `go-go-goja/pkg/jsdoc/model` and port structs from `jsdocex/internal/model/model.go`
- [x] Port `DocStore` (or rename to `Store`, but keep JSON stable) including `AddFile` remove/replace semantics
- [x] Add unit tests for `DocStore.AddFile` overwrite/removal semantics

**1.2 Extractor**
- [x] Create `go-go-goja/pkg/jsdoc/extract`
- [x] Implement `ParseFile`, `ParseSource`, `ParseDir` with current semantics
  - [x] keep `ParseDir` non-recursive (parity)
  - [x] preserve `Line` as 1-based line number for `__doc__` and `__example__` nodes
- [x] Re-implement tree-walk using `github.com/tree-sitter/go-tree-sitter` (do not carry smacker binding in go-go-goja)
- [x] Preserve sentinel + template behavior:
  - [x] `__package__({ ... })`
  - [x] `__doc__("name", { ... })` and `__doc__({name: "...", ...})`
  - [x] `__example__({ ... })`
  - [x] `doc\`...\`` attaches prose by `symbol:` / `package:` frontmatter
- [x] Decide and document (parity vs enhancement) for known gaps:
  - [x] `Example.Body` remains empty for parity (explicitly documented)
  - [x] frontmatter parser remains simple (explicitly documented)
  - [x] object-literal JS→JSON conversion remains heuristic (explicitly documented)

**1.3 Watcher**
- [x] Create `go-go-goja/pkg/jsdoc/watch` ported from `jsdocex/internal/watcher/watcher.go`
- [x] Preserve behavior:
  - [x] watch subdirectories
  - [x] debounce per-path (150ms)
  - [x] ignore non-`.js`

**1.4 Server**
- [x] Create `go-go-goja/pkg/jsdoc/server` ported from `jsdocex/internal/server/server.go`
- [x] Preserve HTTP API contract:
  - [x] `GET /api/store`
  - [x] `GET /api/package/{name}`
  - [x] `GET /api/symbol/{name}` includes `examples: []`
  - [x] `GET /api/example/{id}`
  - [x] `GET /api/search?q=...`
  - [x] `GET /events` SSE sends `reload`
  - [x] `GET /*` serves embedded UI HTML (copy `jsdocex/internal/server/ui.go` as-is)
- [x] Preserve change handling + SSE broadcast semantics
- [x] Add minimal handler tests with `httptest` (store + symbol route)

### Phase 2: Glazed CLI (`cmd/goja-jsdoc`)

- [x] Create `go-go-goja/cmd/goja-jsdoc`
- [x] Add Glazed/Cobra wiring like `go-go-goja/cmd/goja-perf/main.go`
- [x] Implement `extract` command (JSON parity mode):
  - [x] `--file` required
  - [x] JSON output with `--pretty` and optional `--output-file`
  - [ ] (defer) Glazed row output modes (handled in follow-up ticket about output formats)
- [x] Implement `serve` command:
  - [x] `--dir`, `--host`, `--port`
  - [x] initial parse + watcher + server run-until-canceled

### Phase 3: Tests + parity comparison + cutover

- [x] Add extractor parity tests using `jsdocex/samples/*.js` (copied into `go-go-goja/testdata/jsdoc`)
  - [x] field-level assertions (minimum)
  - [ ] golden JSON outputs (optional)
- [x] Manual parity runbook (documented in a playbook or the design guide):
  - [x] write runbook doc (`playbooks/01-parity-runbook.md`)
  - [x] compare `jsdocex extract` vs `goja-jsdoc extract` on all samples
  - [x] compare server endpoints for one sample directory
- [ ] Once parity confirmed:
  - [x] remove `./jsdocex` from the workspace `go.work`
  - [ ] delete or archive `jsdocex/` directory (destructive; confirm before doing)

### Commit checkpoints (recommended)

- [x] Commit A: ticket docs + task breakdown + diary updates
- [x] Commit B: `pkg/jsdoc/model` (+ tests)
- [x] Commit C: `pkg/jsdoc/extract` (+ tests)
- [x] Commit D: `pkg/jsdoc/watch` + `pkg/jsdoc/server` (+ handler tests)
- [x] Commit E: `cmd/goja-jsdoc` Glazed commands
- [x] Commit F: parity tests + `go.work` cutover cleanup
