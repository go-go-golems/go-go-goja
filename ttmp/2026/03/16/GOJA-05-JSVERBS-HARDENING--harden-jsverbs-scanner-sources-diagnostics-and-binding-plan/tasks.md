# Tasks

## TODO

- [x] Replace js-to-json source rewriting with strict AST literal parsing for metadata objects and arrays
- [x] Add scan diagnostics and surface invalid metadata instead of silently dropping it
- [x] Unify new jsverbs package error style around standard fmt.Errorf wrapping
- [x] Keep promise polling as v1 behavior but document it clearly in code and docs
- [x] Extract a shared binding plan so schema generation and runtime argument binding follow one contract
- [x] Add failure-path tests for malformed metadata, invalid binds, and scanner/runtime error cases
- [x] Add jsverbs source inputs for raw JS strings and embed.FS-backed scanning/loading
