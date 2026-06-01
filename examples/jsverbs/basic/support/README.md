# jsverbs support modules

This folder contains helper modules loaded by the basic jsverbs examples.

## Run

Do not run these files directly. Run a verb that imports them, for example:

```bash
GOWORK=off go run ./cmd/jsverbs-example --dir ./examples/jsverbs/basic basics list-issues go-go-golems/go-go-goja --state closed --labels bug --labels docs
```

## What this showcases

Support files let examples demonstrate script-local module roots and ordinary JavaScript helper reuse without exposing every helper as a command.

## Status

Maintained support files for `examples/jsverbs/basic`.
