# Basic jsverbs example tree

This is the default example tree for `cmd/jsverbs-example`.

## Run

From the repository root:

```bash
GOWORK=off go run ./cmd/jsverbs-example --dir ./examples/jsverbs/basic list
GOWORK=off go run ./cmd/jsverbs-example --dir ./examples/jsverbs/basic basics greet Manuel --excited
GOWORK=off go run ./cmd/jsverbs-example --dir ./examples/jsverbs/basic basics banner Manuel
GOWORK=off go run ./cmd/jsverbs-example --dir ./examples/jsverbs/basic basics summarize --owner go-go-golems --repo go-go-goja
GOWORK=off go run ./cmd/jsverbs-example --dir ./examples/jsverbs/basic advanced numbers add 2 3
GOWORK=off go run ./cmd/jsverbs-example --dir ./examples/jsverbs/basic events event-timeline evt --count 2
GOWORK=off go run ./cmd/jsverbs-example --dir ./examples/jsverbs/basic nested with-helper render hi there
GOWORK=off go run ./cmd/jsverbs-example --dir ./examples/jsverbs/basic meta pkg-demo ping
```

## What this showcases

- `basics.js` — verb metadata, arguments, flags, writer output, and shared sections.
- `advanced/numbers.js` — nested command paths, arithmetic, async return values, and repeated values.
- `events.js` — native EventEmitter usage from JavaScript verbs.
- `nested/with-helper.js` — relative `require()` from a verb file.
- `packaged.js` — package metadata changing the command path.
- `fswatch.js` — connected EventEmitter helper fixture.

## Status

Most commands are directly runnable with `cmd/jsverbs-example`. `fswatch.js` is not a normal default-CLI example; it requires a runtime initializer that installs `fswatch`, and is validated by Go tests.
