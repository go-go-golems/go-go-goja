# xgoja multiple runtimes example

This example shows one generated xgoja binary with two runtime profiles:

- `safe` includes only safe/core modules: `path` and `yaml`.
- `host` includes `path` plus the guarded host `fs` module with `config.allow: true`.

The command mapping is intentionally different per command:

- `eval` uses the `safe` runtime.
- `run` uses the `host` runtime.
- `repl` uses the `safe` runtime.

Run:

```bash
make smoke
```

The smoke target proves three things:

1. `eval` can require `path` from the safe runtime.
2. `eval` cannot require `fs`, because `fs` is not selected by the safe runtime.
3. `run scripts/host-run.js` can require `fs`, because `run` is bound to the host runtime.

This is the key xgoja composition model: provider packages are compiled into the binary, but each command invocation receives only the modules selected by that command's runtime profile.
