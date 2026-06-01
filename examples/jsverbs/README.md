# jsverbs examples

This folder contains JavaScript verb trees for `cmd/jsverbs-example`.

## Run

From the repository root:

```bash
GOWORK=off go run ./cmd/jsverbs-example --dir ./examples/jsverbs/basic list
GOWORK=off go run ./cmd/jsverbs-example --dir ./examples/jsverbs/basic basics greet Manuel --excited
GOWORK=off go run ./cmd/jsverbs-example --dir ./examples/jsverbs/basic nested with-helper render hi there
GOWORK=off go run ./cmd/jsverbs-example --dir ./examples/jsverbs/registry-shared list
GOWORK=off go run ./cmd/jsverbs-example --dir ./examples/jsverbs/registry-shared summary summarize-filters --state closed --labels bug --labels docs
```

`cmd/jsverbs-example` also defaults to `examples/jsverbs/basic` when `--dir` is omitted.

## Folders

- `basic/` — the primary jsverbs learning and test fixture tree.
- `registry-shared/` — demonstrates host-registered shared sections used by multiple files.

## Status

The listed commands are expected to work. `basic/fswatch.js` is a specialized connected-EventEmitter fixture: it scans and appears in `list`, but running it requires a runtime that installs the `fswatch` helper. It is covered by Go tests rather than the default CLI smoke command.
