# Changelog

## 2026-03-05

- Initial workspace created


## 2026-03-05

Added intern-focused design/implementation guide with current-state analysis, target package layout, API parity checklist, and Glazed CLI plan.

### Related Files

- /home/manuel/workspaces/2026-03-05/add-jsdocex/go-go-goja/ttmp/2026/03/05/GOJA-01-INTEGRATE-JSDOCEX--integrate-jsdocex-extraction-server-into-go-go-goja/reference/01-design-implementation-guide-integrate-jsdocex-into-go-go-goja.md — Primary deliverable for migration work


## 2026-03-05

Uploaded ticket bundle to reMarkable under /ai/2026/03/05/GOJA-01-INTEGRATE-JSDOCEX (see PDFs named 'GOJA-01 Integrate jsdocex into go-go-goja').

### Related Files

- /home/manuel/workspaces/2026-03-05/add-jsdocex/go-go-goja/ttmp/2026/03/05/GOJA-01-INTEGRATE-JSDOCEX--integrate-jsdocex-extraction-server-into-go-go-goja/reference/01-design-implementation-guide-integrate-jsdocex-into-go-go-goja.md — Bundle contents


## 2026-03-05

Step 3: Ported jsdoc model + DocStore into go-go-goja (commit 80eefd1).

### Related Files

- /home/manuel/workspaces/2026-03-05/add-jsdocex/go-go-goja/pkg/jsdoc/model/model.go — Exported jsdoc model types
- /home/manuel/workspaces/2026-03-05/add-jsdocex/go-go-goja/pkg/jsdoc/model/store.go — DocStore and indexing semantics
- /home/manuel/workspaces/2026-03-05/add-jsdocex/go-go-goja/pkg/jsdoc/model/store_test.go — Unit tests for overwrite/removal semantics


## 2026-03-05

Step 4: Ported jsdocex extractor into pkg/jsdoc/extract using go-tree-sitter (commit 510dbde).

### Related Files

- /home/manuel/workspaces/2026-03-05/add-jsdocex/go-go-goja/pkg/jsdoc/extract/extract.go — Extractor implementation (sentinels + doc)


## 2026-03-05

Step 5: Ported watcher and HTTP doc server into pkg/jsdoc/watch and pkg/jsdoc/server (commit 7d0e00c).

### Related Files

- /home/manuel/workspaces/2026-03-05/add-jsdocex/go-go-goja/pkg/jsdoc/server/server.go — HTTP API + SSE + UI handler
- /home/manuel/workspaces/2026-03-05/add-jsdocex/go-go-goja/pkg/jsdoc/server/server_test.go — Minimal handler tests
- /home/manuel/workspaces/2026-03-05/add-jsdocex/go-go-goja/pkg/jsdoc/server/ui.go — Embedded UI (copied from jsdocex)
- /home/manuel/workspaces/2026-03-05/add-jsdocex/go-go-goja/pkg/jsdoc/watch/watcher.go — FS watcher with debounce

