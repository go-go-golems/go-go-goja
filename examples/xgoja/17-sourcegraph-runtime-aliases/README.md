# xgoja sourcegraph runtime aliases coverage

This example is a regression guard for xgoja source graph import resolution. It intentionally combines several source and runtime features in one generated binary:

- JavaScript jsverbs with a local `require("./helper.js")` import.
- TypeScript jsverbs with a local `import { format } from "./format"` import.
- Literal runtime aliases with colons, especially `require("fs:assets")`.
- Embedded assets served through the host `fs` provider alias `fs:assets`.
- Express HTTP serving through the `go-go-goja-http` provider command set.
- Builtin `verbs` command execution and provider-backed `serve` command execution.
- A generated DTS artifact.

The important regression is that static source scanning must accept selected runtime aliases exactly as declared, including aliases such as `fs:assets`. The runtime already supports this alias; the source graph must not reject it as an unknown bare specifier.

## Run

```bash
make -C examples/xgoja/17-sourcegraph-runtime-aliases smoke
```

The smoke target validates the xgoja spec, builds the generated binary, runs a TypeScript jsverb that reads embedded config through `require("fs:assets")`, serves an Express site from a JavaScript jsverb, checks embedded static assets, and proves the built binary can run away from the original source tree.

## Manual HTTP run

```bash
make -C examples/xgoja/17-sourcegraph-runtime-aliases build
examples/xgoja/17-sourcegraph-runtime-aliases/dist/sourcegraph-runtime-aliases \
  serve imports serve \
  --http-listen 127.0.0.1:18790
```

Then open:

- <http://127.0.0.1:18790/>
- <http://127.0.0.1:18790/api/config>
- <http://127.0.0.1:18790/static/app.js>

## Key xgoja fragment

```yaml
runtime:
  modules:
    - provider: go-go-goja-host
      name: fs
      as: fs:assets
      config:
        embedded:
          allow: true
          mounts:
            - asset: web-assets
              mount: /app
    - provider: go-go-goja-http
      name: express
sources:
  - id: js-site
    kind: jsverbs
    from:
      dir: ./verbs-js
  - id: ts-tools
    kind: jsverbs
    from:
      dir: ./verbs-ts
    language: typescript
    compile:
      mode: runtime
      bundle: true
  - id: web-assets
    kind: assets
    from:
      dir: ./assets
```

Both `verbs-js/site.js` and `verbs-ts/tools.ts` use literal `require("fs:assets")`.
