# Examples

This directory contains runnable examples and sample inputs for go-go-goja commands and libraries.

## Folders

- `goja-repl/` — standalone JavaScript scripts for `cmd/goja-repl run`.
- `gojahttp/` — Go-native HTTP examples, including planned auth without JavaScript Express.
- `inspector/` — JavaScript input for the inspector TUI example.
- `jsdoc/` — JavaScript files with JSDoc comments used by extractor tests and manual export commands.
- `jsverbs/` — JavaScript verb examples for `cmd/jsverbs-example`.
- `xgoja/` — generated xgoja binary examples with numbered smoke-testable projects.

## Quick smoke checks

From the repository root:

```bash
GOWORK=off go run ./cmd/goja-repl --enable-module yaml run ./examples/goja-repl/scripts/yaml.js
GOWORK=off go run ./cmd/jsverbs-example --dir ./examples/jsverbs/basic list
GOWORK=off go run ./cmd/jsverbs-example --dir ./examples/jsverbs/registry-shared list
make -C examples/gojahttp/01-planned-auth smoke
```

See each subdirectory README for the command-specific examples and caveats.
