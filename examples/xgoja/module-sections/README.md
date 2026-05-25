# module-sections xgoja example

This example demonstrates module-provided Glazed sections on built-in generated commands.

The fixture provider exposes a `fixture` section with a `--fixture-value` flag and a runtime initializer that writes the parsed value to `globalThis.fixtureValue`.

The smoke test proves all built-in non-interactive paths:

- `eval` exposes `--fixture-value` and initializes the runtime before evaluating the source string.
- `run` exposes `--fixture-value` and initializes the runtime before loading `scripts/check-fixture.js`.
- `verbs tools check-fixture` exposes the same flag and initializes the runtime before invoking an embedded JavaScript verb.

Run:

```bash
make -C examples/xgoja/module-sections smoke
```
