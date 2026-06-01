# JSDoc extraction examples

This folder contains JavaScript files with JSDoc comments for the `goja-jsdoc` extractor and related tests.

## Run

From the repository root, export one sample:

```bash
GOWORK=off go run ./cmd/goja-jsdoc export ./examples/jsdoc/samples/01-math.js --format json --shape store --pretty
```

Export all samples by pointing tools at the directory:

```bash
GOWORK=off go run ./cmd/goja-jsdoc serve --dir ./examples/jsdoc/samples --host 127.0.0.1 --port 8090
```

Stop the server with Ctrl-C when done.

## What this showcases

- Function and symbol extraction from documented JavaScript.
- Multiple sample domains: math helpers, easing functions, vector utilities, and events.
- Fixture coverage for scoped filesystem extraction tests.

## Status

The samples are maintained extractor fixtures and examples. They are not meant to be executed as application scripts; they are inputs to JSDoc tooling.
