# Changelog

## 2026-03-05

- Initial workspace created
- Created GOJA-02 ticket workspace with initial design/implementation plan, tasks, and diary.
- Added `pkg/jsdoc/batch` to build a `DocStore` from multiple inputs (commit 6987c36).
- Added `pkg/jsdoc/export` dispatcher and exporters for JSON/YAML/Markdown/SQLite (commit 57899b0).

### Related Files

- /home/manuel/workspaces/2026-03-05/add-jsdocex/go-go-goja/ttmp/2026/03/05/GOJA-02-JSDOC-EXPORT-API--multi-format-jsdoc-export-batch-api-json-yaml-sqlite-markdown/reference/01-design-implementation-plan-batch-jsdoc-api-and-multi-format-exporters.md — Primary design doc
- /home/manuel/workspaces/2026-03-05/add-jsdocex/go-go-goja/pkg/jsdoc/batch/batch.go — Batch store builder used by CLI/API export
- /home/manuel/workspaces/2026-03-05/add-jsdocex/go-go-goja/pkg/jsdoc/export/export.go — Format dispatcher and JSON/YAML exporters
- /home/manuel/workspaces/2026-03-05/add-jsdocex/go-go-goja/pkg/jsdoc/exportmd/exportmd.go — Markdown generator + deterministic ToC
- /home/manuel/workspaces/2026-03-05/add-jsdocex/go-go-goja/pkg/jsdoc/exportsq/exportsq.go — SQLite schema + transactional export
