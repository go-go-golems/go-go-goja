# Tasks

## TODO

### Phase 0: Evidence + parity targets (docs)

- [x] Read `jsdocex/` code and document behavior parity targets (see design guide)
- [x] Design `go-go-goja/pkg/jsdoc` package layout + public API (see design guide)
- [ ] Add a short “Acceptance Criteria” section to the design guide (exact parity checklist)

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
- [ ] Decide and document (parity vs enhancement) for known gaps:
  - [ ] `Example.Body` remains empty for parity (explicitly documented)
  - [ ] frontmatter parser remains simple (explicitly documented)
  - [ ] object-literal JS→JSON conversion remains heuristic (explicitly documented)

**1.3 Watcher**
- [ ] Create `go-go-goja/pkg/jsdoc/watch` ported from `jsdocex/internal/watcher/watcher.go`
- [ ] Preserve behavior:
  - [ ] watch subdirectories
  - [ ] debounce per-path (150ms)
  - [ ] ignore non-`.js`

**1.4 Server**
- [ ] Create `go-go-goja/pkg/jsdoc/server` ported from `jsdocex/internal/server/server.go`
- [ ] Preserve HTTP API contract:
  - [ ] `GET /api/store`
  - [ ] `GET /api/package/{name}`
  - [ ] `GET /api/symbol/{name}` includes `examples: []`
  - [ ] `GET /api/example/{id}`
  - [ ] `GET /api/search?q=...`
  - [ ] `GET /events` SSE sends `reload`
  - [ ] `GET /*` serves embedded UI HTML (copy `jsdocex/internal/server/ui.go` as-is)
- [ ] Preserve change handling + SSE broadcast semantics
- [ ] Add minimal handler tests with `httptest` (store + symbol route)

### Phase 2: Glazed CLI (`cmd/goja-jsdoc`)

- [ ] Create `go-go-goja/cmd/goja-jsdoc`
- [ ] Add Glazed/Cobra wiring like `go-go-goja/cmd/goja-perf/main.go`
- [ ] Implement `extract` command:
  - [ ] `--file` required
  - [ ] default output as rows (symbols/examples/package)
  - [ ] optional `--raw-json` to print `FileDoc` JSON for parity debugging
- [ ] Implement `serve` command:
  - [ ] `--dir`, `--host`, `--port`
  - [ ] initial parse + watcher + server run-until-canceled

### Phase 3: Tests + parity comparison + cutover

- [ ] Add extractor parity tests using `jsdocex/samples/*.js`
  - [ ] golden JSON outputs (optional) or field-level assertions (minimum)
- [ ] Manual parity runbook (documented in a playbook or the design guide):
  - [ ] compare `jsdocex extract` vs `goja-jsdoc extract` on all samples
  - [ ] compare server endpoints for one sample directory
- [ ] Once parity confirmed:
  - [ ] remove `./jsdocex` from the workspace `go.work`
  - [ ] delete or archive `jsdocex/` directory (no compatibility wrapper unless requested)

### Commit checkpoints (recommended)

- [ ] Commit A: ticket docs + task breakdown + diary updates
- [ ] Commit B: `pkg/jsdoc/model` (+ tests)
- [ ] Commit C: `pkg/jsdoc/extract` (+ tests)
- [ ] Commit D: `pkg/jsdoc/watch` + `pkg/jsdoc/server` (+ handler tests)
- [ ] Commit E: `cmd/goja-jsdoc` Glazed commands
- [ ] Commit F: parity tests + `go.work` cutover cleanup
