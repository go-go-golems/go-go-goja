# goja-repl script examples

This folder contains JavaScript scripts intended to be run with `cmd/goja-repl run`.

These used to live under repository-level `testdata/`, but they are examples rather than hidden fixtures. They now live here so users can discover and run them directly.

## Run all smoke examples

From the repository root:

```bash
GOWORK=off go run ./cmd/goja-repl --enable-module yaml run ./examples/goja-repl/scripts/yaml.js
GOWORK=off go run ./cmd/goja-repl --enable-module fs --enable-module exec run ./examples/goja-repl/scripts/hello.js
GOWORK=off go run ./cmd/goja-repl --enable-module database run ./examples/goja-repl/scripts/database.js
```

## What this showcases

- `yaml.js` exercises `require("yaml")`: parse, stringify, validate, and round-trip behavior.
- `hello.js` exercises host-capability modules `require("fs")` and `require("exec")`.
- `database.js` exercises `require("database")` with an in-memory SQLite database.

## Status

All scripts in `scripts/` are expected to run. Use the module flags shown above; several examples require host-capability modules that are not part of safe-mode runtimes.
