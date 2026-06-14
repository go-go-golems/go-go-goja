# Changelog

## 2026-06-14

- Initial workspace created


## 2026-06-14

Created sourcegraph parser ticket, preserved Goja/esbuild/tree-sitter experiments, and confirmed tree-sitter TypeScript/TSX support is a strong implementation path.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/ttmp/2026/06/14/GOJA-XGOJA-SOURCEGRAPH-PARSER-001--replace-xgoja-sourcegraph-regex-import-scanning-with-parser-backed-runtime-alias-resolution/design-doc/01-parser-backed-xgoja-sourcegraph-import-resolution-design.md — Initial design and parser comparison
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/ttmp/2026/06/14/GOJA-XGOJA-SOURCEGRAPH-PARSER-001--replace-xgoja-sourcegraph-regex-import-scanning-with-parser-backed-runtime-alias-resolution/scripts/04-treesitter-typescript-experiment.sh — TypeScript/TSX tree-sitter evidence


## 2026-06-14

Implemented tree-sitter-backed sourcegraph import parsing and RuntimePlan alias propagation; focused tests and sourcegraph-runtime-aliases smoke pass.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/pkg/xgoja/app/source_registry.go — Carries runtime aliases into jsverb source scans
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/pkg/xgoja/sourcegraph/imports.go — New parser-backed collector

