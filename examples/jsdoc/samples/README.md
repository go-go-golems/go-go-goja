# JSDoc sample files

These JavaScript files are source inputs for JSDoc extraction.

## Run one export

```bash
GOWORK=off go run ./cmd/goja-jsdoc export ./examples/jsdoc/samples/01-math.js --format markdown --toc-depth 3
```

## Files

- `01-math.js` — documented math functions.
- `02-easing.js` — easing helper API documentation.
- `03-vector2.js` — small vector type/function sample.
- `04-events.js` — event-related API sample.

## Status

These files are active fixtures for extractor tests. They should remain small, deterministic, and free of runtime side effects.
