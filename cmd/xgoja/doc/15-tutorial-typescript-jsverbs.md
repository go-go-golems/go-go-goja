---
Title: "Tutorial: TypeScript JavaScript verbs"
Slug: tutorial-typescript-jsverbs
Short: "Author xgoja JavaScript verbs in TypeScript and serve them with hot reload."
Topics:
- xgoja
- tutorial
- typescript
- jsverbs
- hot-reload
Commands:
- xgoja build
- xgoja doctor
- xgoja gen-dts
Flags:
- --hot-reload
- --hot-reload-watch-root
- --hot-reload-smoke-path
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

xgoja can scan JavaScript verb sources written in TypeScript. At scan time,
xgoja transpiles the TypeScript source to JavaScript so the existing jsverbs
metadata extractor can discover packages, verbs, sections, and function names.
At runtime, xgoja compiles the original TypeScript source plus the jsverbs
capture overlay before goja loads the module.

This lets you write files such as `verbs/sites.ts` while keeping the same
Go-backed provider modules and generated command model.

## 1. Configure a TypeScript jsverb source

Add a `typescript` block to a jsverb source in `xgoja.yaml`:

```yaml
jsverbs:
  - id: local-sites
    path: ./verbs
    embed: false
    extensions: [".ts"]
    include: ["**/*.ts"]
    exclude: ["**/*.test.ts"]
    typescript:
      enabled: true
      bundle: true
      target: es2015
      format: cjs
      platform: neutral
      external:
        - express
```

Use `bundle: true` when a verb imports local TypeScript helpers. Add xgoja native
module names such as `express`, `fs:assets`, or `path` to `external` so esbuild
preserves those `require()` calls for goja's runtime module loader.

## 2. Write TypeScript verbs

```ts
import { message } from "./message"

__package__({ name: "sites", short: "TypeScript demo sites" })
__verb__("demo", { name: "demo", output: "text" })

function demo(): void {
  const express = require("express")
  const app = express.app()
  const version = 1

  app.get("/", (_req: unknown, res: any) => {
    res.send(message("xgoja", version))
  })

  app.get("/healthz", (_req: unknown, res: any) => {
    res.json({ ok: true, site: "typescript-demo", version })
  })
}
```

The runtime is still goja, not Node.js. Import local helpers freely when bundling
is enabled, but keep Go-backed xgoja modules external.

## 3. Generate editor declarations

Use the existing declaration workflow for selected xgoja modules:

```bash
xgoja gen-dts -f xgoja.yaml --out js/types/xgoja-modules.d.ts --strict
```

A minimal `tsconfig.json` can include generated declarations and any local
ambient globals used by jsverbs:

```json
{
  "compilerOptions": {
    "target": "ES2015",
    "module": "CommonJS",
    "moduleResolution": "Bundler",
    "strict": true,
    "types": [],
    "typeRoots": ["./js/types", "./node_modules/@types"],
    "skipLibCheck": true
  },
  "include": ["verbs/**/*.ts", "js/types/**/*.d.ts"]
}
```

xgoja uses esbuild for runtime compilation. esbuild strips TypeScript types but
does not type-check. Run `tsc --noEmit` separately in project CI if you need full
static type checking.

## 4. Build and serve with hot reload

```bash
xgoja doctor -f xgoja.yaml
xgoja build -f xgoja.yaml --output dist/typescript-jsverbs
./dist/typescript-jsverbs serve sites demo \
  --http-listen 127.0.0.1:18789 \
  --hot-reload \
  --hot-reload-watch-root ./verbs \
  --hot-reload-smoke-path /healthz
```

When any configured jsverb source has TypeScript enabled, HTTP hot reload also
watches `.ts`, `.tsx`, `.mts`, and `.cts` files. A successful edit creates a new
candidate runtime, smoke-tests it if configured, and swaps it live. A broken edit
leaves the last-known-good runtime active and records the error in the hot reload
status endpoint.

## 5. Try the example

The repository includes a complete example:

```bash
make -C examples/xgoja/15-typescript-jsverbs smoke
```

The smoke test builds a generated binary, runs a TypeScript entry file through
`xgoja run`, serves a TypeScript jsverb site with hot reload, edits the source
from version 1 to version 2, and verifies the reloaded HTTP response.
