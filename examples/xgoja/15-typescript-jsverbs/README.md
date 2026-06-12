# xgoja TypeScript jsverbs example

This example builds a generated xgoja binary that discovers JavaScript verbs from
TypeScript source files. The `sites demo` verb imports a local `message.ts`
helper, registers Express routes, and can be served with hot reload.

The important pieces are in the native `schema: xgoja/v2` spec:

```yaml
schema: xgoja/v2
providers:
  - id: go-go-goja-http
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/http
runtime:
  modules:
    - provider: go-go-goja-http
      name: express
sources:
  - id: local-sites
    kind: jsverbs
    from:
      dir: ./verbs
    language: typescript
    compile:
      mode: runtime
      bundle: true
commands:
  - id: verbs
    type: builtin.jsverbs
    sources: [local-sites]
  - id: http-serve
    type: provider.command-set
    provider: go-go-goja-http
    name: serve
    mount: serve
    sources: [local-sites]
```

`compile.bundle: true` tells xgoja to bundle each TypeScript verb entry before it
is loaded by goja. The `express` runtime module is selected under
`runtime.modules`, so xgoja derives the TypeScript external automatically and
preserves `require("express")` for the Go-backed provider module at runtime.

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
