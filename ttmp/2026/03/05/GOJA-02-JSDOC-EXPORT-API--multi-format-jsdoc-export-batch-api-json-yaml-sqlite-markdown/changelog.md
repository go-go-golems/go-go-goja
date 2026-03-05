# Changelog

## 2026-03-05

- Initial workspace created
- Created GOJA-02 ticket workspace with initial design/implementation plan, tasks, and diary.
- Added `pkg/jsdoc/batch` to build a `DocStore` from multiple inputs (commit 6987c36).
- Added `pkg/jsdoc/export` dispatcher and exporters for JSON/YAML/Markdown/SQLite (commit 57899b0).
- Added `goja-jsdoc export` command (batch input + multi-format output) (commit 229566f).
- Added HTTP endpoints `POST /api/batch/extract` and `POST /api/batch/export` with path safety and handler tests (commit 3d02600).
- Updated GOJA-02 design doc with implemented CLI/API details and added an E2E runbook playbook.

### Related Files

- /home/manuel/workspaces/2026-03-05/add-jsdocex/go-go-goja/ttmp/2026/03/05/GOJA-02-JSDOC-EXPORT-API--multi-format-jsdoc-export-batch-api-json-yaml-sqlite-markdown/reference/01-design-implementation-plan-batch-jsdoc-api-and-multi-format-exporters.md — Primary design doc
- /home/manuel/workspaces/2026-03-05/add-jsdocex/go-go-goja/pkg/jsdoc/batch/batch.go — Batch store builder used by CLI/API export
- /home/manuel/workspaces/2026-03-05/add-jsdocex/go-go-goja/pkg/jsdoc/export/export.go — Format dispatcher and JSON/YAML exporters
- /home/manuel/workspaces/2026-03-05/add-jsdocex/go-go-goja/pkg/jsdoc/exportmd/exportmd.go — Markdown generator + deterministic ToC
- /home/manuel/workspaces/2026-03-05/add-jsdocex/go-go-goja/pkg/jsdoc/exportsq/exportsq.go — SQLite schema + transactional export
- /home/manuel/workspaces/2026-03-05/add-jsdocex/go-go-goja/cmd/goja-jsdoc/export_command.go — CLI entry point for batch export
- /home/manuel/workspaces/2026-03-05/add-jsdocex/go-go-goja/pkg/jsdoc/server/batch_handlers.go — HTTP handlers for batch extraction/export
- /home/manuel/workspaces/2026-03-05/add-jsdocex/go-go-goja/ttmp/2026/03/05/GOJA-02-JSDOC-EXPORT-API--multi-format-jsdoc-export-batch-api-json-yaml-sqlite-markdown/playbooks/01-e2e-export-runbook.md — Manual E2E validation runbook
