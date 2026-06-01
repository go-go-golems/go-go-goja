# Advanced jsverbs examples

This folder demonstrates nested command paths and slightly richer argument handling.

## Run

From the repository root:

```bash
GOWORK=off go run ./cmd/jsverbs-example --dir ./examples/jsverbs/basic advanced numbers add 2 3
GOWORK=off go run ./cmd/jsverbs-example --dir ./examples/jsverbs/basic advanced numbers multiply 2 3
GOWORK=off go run ./cmd/jsverbs-example --dir ./examples/jsverbs/basic advanced numbers list-names alice bob charlie
```

## What this showcases

`numbers.js` contributes commands under `advanced numbers ...`, including synchronous output, async output, and rest/repeated argument behavior.

## Status

These commands are expected to run as part of manual jsverbs smoke checks.
