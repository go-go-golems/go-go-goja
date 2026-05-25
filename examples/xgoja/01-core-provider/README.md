# xgoja core provider example

This example builds a generated xgoja binary using the first-party safe/core provider package at `pkg/xgoja/providers/core`.

The runtime selects data-oriented modules only:

- `path`
- `yaml`
- `crypto`

Run:

```bash
make smoke
```

The smoke target validates the spec, lists selected modules, builds the generated binary, runs one `eval` expression, and executes `scripts/core-smoke.js` with the generated `run` command.
