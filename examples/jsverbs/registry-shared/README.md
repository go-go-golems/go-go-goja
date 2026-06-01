# Registry-shared jsverbs example

This folder demonstrates shared sections registered by the Go host instead of declared in each JavaScript file.

## Run

From the repository root:

```bash
GOWORK=off go run ./cmd/jsverbs-example --dir ./examples/jsverbs/registry-shared list
GOWORK=off go run ./cmd/jsverbs-example --dir ./examples/jsverbs/registry-shared issues list-issues go-go-golems/go-go-goja --state closed --labels bug --labels docs
GOWORK=off go run ./cmd/jsverbs-example --dir ./examples/jsverbs/registry-shared summary summarize-filters --state closed --labels bug --labels docs
```

## What this showcases

The JavaScript files bind fields to a `filters` section without declaring that section locally. `cmd/jsverbs-example` detects this directory and registers the shared `filters` section from Go before command compilation.

## Status

These commands are expected to run. This folder is intentionally separate from `basic/` so the host-provided-section behavior is explicit and deterministic.
