# Nested jsverbs examples

This folder demonstrates relative `require()` from a JavaScript verb file.

## Run

From the repository root:

```bash
GOWORK=off go run ./cmd/jsverbs-example --dir ./examples/jsverbs/basic nested with-helper render hi there
```

Expected output contains `hi:there`.

## What this showcases

`with-helper.js` loads `./sub/helper` relative to the script file. This proves the jsverbs runtime adds the script's directory to module resolution roots before invoking the verb.

## Status

This command is expected to run. If it fails with a module resolution error, check the jsverbs runtime require-root handling.
