# Changelog

## 2026-03-16

- Initial workspace created


## 2026-03-16

Implemented the jsverbs hardening pass: strict AST metadata parsing, diagnostics, raw/fs source support, shared binding planning, standardized errors, and failure-path tests.

### Related Files

- /home/manuel/workspaces/2026-03-16/add-glazed-js-layer/go-go-goja/pkg/jsverbs/binding.go — New shared binding plan introduced
- /home/manuel/workspaces/2026-03-16/add-glazed-js-layer/go-go-goja/pkg/jsverbs/jsverbs_test.go — Expanded success and failure coverage
- /home/manuel/workspaces/2026-03-16/add-glazed-js-layer/go-go-goja/pkg/jsverbs/runtime.go — Runtime loader refactored to in-memory module serving with v1 polling note
- /home/manuel/workspaces/2026-03-16/add-glazed-js-layer/go-go-goja/pkg/jsverbs/scan.go — Scanner rewritten around strict literal parsing and diagnostics

