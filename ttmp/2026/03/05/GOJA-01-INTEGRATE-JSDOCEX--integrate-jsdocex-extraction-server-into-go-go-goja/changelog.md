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


## 2026-03-05

Step 6: Added cmd/goja-jsdoc Glazed CLI with extract and serve commands (commit 692e148).

### Related Files

- /home/manuel/workspaces/2026-03-05/add-jsdocex/go-go-goja/cmd/goja-jsdoc/extract_command.go — Extract command (JSON parity output)
- /home/manuel/workspaces/2026-03-05/add-jsdocex/go-go-goja/cmd/goja-jsdoc/main.go — Cobra wiring for jsdoc CLI
- /home/manuel/workspaces/2026-03-05/add-jsdocex/go-go-goja/cmd/goja-jsdoc/serve_command.go — Serve command (web UI + API + watcher)


## 2026-03-05

Step 7: Added extractor parity tests and copied jsdocex samples into go-go-goja/testdata (commit 3795939).

### Related Files

- /home/manuel/workspaces/2026-03-05/add-jsdocex/go-go-goja/pkg/jsdoc/extract/extract_test.go — Parity tests for sentinel + prose attachment
- /home/manuel/workspaces/2026-03-05/add-jsdocex/go-go-goja/testdata/jsdoc/ — JS fixture files copied from jsdocex


## 2026-03-05

Step 8: Added parity runbook playbook for manual jsdocex vs goja-jsdoc comparison (commit c7ddb6e).

### Related Files

- /home/manuel/workspaces/2026-03-05/add-jsdocex/go-go-goja/ttmp/2026/03/05/GOJA-01-INTEGRATE-JSDOCEX--integrate-jsdocex-extraction-server-into-go-go-goja/playbooks/01-parity-runbook.md — Manual parity checklist


## 2026-03-05

Step 9: Executed parity runbook; extract output matched and server endpoints matched after JSON normalization (see playbook).

### Related Files

- /home/manuel/workspaces/2026-03-05/add-jsdocex/go-go-goja/ttmp/2026/03/05/GOJA-01-INTEGRATE-JSDOCEX--integrate-jsdocex-extraction-server-into-go-go-goja/playbooks/01-parity-runbook.md — Commands used for parity run


## 2026-03-05

Step 10: Removed jsdocex module from workspace go.work after parity checks; destructive deletion deferred pending confirmation.

### Related Files

- /home/manuel/workspaces/2026-03-05/add-jsdocex/go.work — Workspace module wiring


## 2026-03-05

Step 11: Updated design doc with acceptance criteria and explicit parity decisions (commit 6a0eb6c).

### Related Files

- /home/manuel/workspaces/2026-03-05/add-jsdocex/go-go-goja/ttmp/2026/03/05/GOJA-01-INTEGRATE-JSDOCEX--integrate-jsdocex-extraction-server-into-go-go-goja/reference/01-design-implementation-guide-integrate-jsdocex-into-go-go-goja.md — Acceptance criteria + parity decisions


## 2026-03-05

Step 12: Fixed docmgr validation by adding YAML frontmatter to parity playbook; doctor now passes clean.

### Related Files

- /home/manuel/workspaces/2026-03-05/add-jsdocex/go-go-goja/ttmp/2026/03/05/GOJA-01-INTEGRATE-JSDOCEX--integrate-jsdocex-extraction-server-into-go-go-goja/playbooks/01-parity-runbook.md — Added required frontmatter

