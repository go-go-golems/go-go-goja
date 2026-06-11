# xgoja TypeScript jsverbs example

This example builds a generated xgoja binary that discovers JavaScript verbs from
TypeScript source files. The `sites demo` verb imports a local `message.ts`
helper, registers Express routes, and can be served with hot reload.

The important pieces are in `xgoja.yaml`:

```yaml
jsverbs:
  - id: local-sites
    path: ./verbs
    embed: false
    extensions: [".ts"]
    typescript:
      enabled: true
      bundle: true
      target: es2015
      format: cjs
      platform: neutral
      external:
        - express
```

`typescript.bundle: true` tells xgoja to bundle each TypeScript verb entry before
it is loaded by goja. `external: [express]` preserves `require("express")` so the
Go-backed xgoja provider module supplies it at runtime.

Run the smoke test:

```bash
make -C examples/xgoja/15-typescript-jsverbs smoke
```

Manual run:

```bash
make -C examples/xgoja/15-typescript-jsverbs build
./examples/xgoja/15-typescript-jsverbs/dist/typescript-jsverbs \
  serve sites demo \
  --http-listen 127.0.0.1:18789 \
  --hot-reload \
  --hot-reload-watch-root examples/xgoja/15-typescript-jsverbs/verbs \
  --hot-reload-smoke-path /healthz
```

Open:

- <http://127.0.0.1:18789/>
- <http://127.0.0.1:18789/healthz>
- <http://127.0.0.1:18789/__xgoja/status>

Generate editor declarations for selected xgoja modules:

```bash
xgoja gen-dts -f examples/xgoja/15-typescript-jsverbs/xgoja.yaml \
  --out examples/xgoja/15-typescript-jsverbs/js/types/xgoja-modules.d.ts \
  --strict
```
