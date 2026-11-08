# Changelog

## 2026-11-07

- Initial workspace created

## 2026-11-07

Engine host options (WithRuntimeBridgeOwner, WithoutDefaultGlobals), jsverbs InvokeInVM, pure-Go scanner with tree-sitter opt-in (pkg/jsverbs/treesitter) and parity tests; xgoja esm TypeScript scans with tree-sitter; docs. Commits 775e4f1 2d2a533 6fc7937 7b668e9 2805dc7 48145f7.

## 2026-11-07

Review fixes for #107: InvokeInVM recovers JavaScript panics (export getters, interrupts) and makes ctx the runtimebridge call context on bound VMs; the goja parser reads a file as the CommonJS function body (top-level return). Commits 3992113 363f343.
