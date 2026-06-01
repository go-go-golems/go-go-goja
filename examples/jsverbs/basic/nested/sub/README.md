# Nested helper module

This folder contains helper code required by `../with-helper.js`.

## Run

Do not run `helper.js` directly. Run the parent verb instead:

```bash
GOWORK=off go run ./cmd/jsverbs-example --dir ./examples/jsverbs/basic nested with-helper render hi there
```

## What this showcases

The helper is intentionally separate so the parent example can prove relative JavaScript module loading works inside jsverbs.

## Status

Maintained support file for the nested jsverbs example.
