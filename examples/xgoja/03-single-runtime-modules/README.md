# xgoja single runtime modules example

This example shows one generated xgoja binary with one top-level `modules` list. The generated runtime includes both safe core modules (`path`, `yaml`) and an explicitly configured host `fs` module.

Run:

```bash
make smoke
```

The smoke target proves three things:

1. `eval` can require `path`.
2. `eval` can also require `fs` because all runtime-backed commands use the same generated module set.
3. `run scripts/host-run.js` can require `fs` and write the configured host output file.

This is the current xgoja composition model: provider packages are compiled into the binary, and the top-level `modules` list defines the JavaScript `require()` surface for all generated runtime-backed commands.
