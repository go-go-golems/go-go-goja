---
Title: Bun Bundling Playbook for Goja
Slug: bun-bundling-playbook-goja
Short: End-to-end guide for bundling TypeScript + assets with Bun and running them in Goja.
Topics:
- bun
- bundling
- goja
- typescript
- commonjs
- assets
- go
IsTemplate: false
IsTopLevel: true
ShowPerDefault: true
SectionType: GeneralTopic
---

# Bun Bundling Playbook for Goja

## Overview
This playbook shows how to manage npm dependencies with Bun, bundle TypeScript and assets into a CommonJS bundle, and execute the result inside Goja using the Go-provided `require` loader. It is written for a Go developer who needs a repeatable pipeline that turns a modern JS/TS project into a single embedded artifact.

## Architecture and flow
The bundling pipeline is a simple, reproducible assembly line: Bun installs dependencies, esbuild bundles the TS entrypoint into a single CommonJS file, Go embeds that file, and Goja `require()` executes it at runtime. This keeps the runtime loader focused on CommonJS and avoids shipping a full Node runtime.

High-level flow:
- Author TS/JS sources in `cmd/bun-demo/js/src`.
- Run `bun install` to lock and install npm deps.
- Bundle with esbuild into `cmd/bun-demo/js/dist/bundle.cjs`.
- Copy the bundle to `cmd/bun-demo/assets/bundle.cjs` for embedding.
- Load the bundle via `require` inside Goja.

## Packaging models (A vs B)
Bundling for Goja usually fits into one of two models. The demo uses Model A because it keeps the runtime loader simple and uses the existing CommonJS `require` you already provide.

**Model A: Single CommonJS bundle (recommended)**
- One entrypoint (`bundle.cjs`) embeds all npm-managed code.
- Only native or host-provided modules stay external (`fs`, `exec`, `database`).

**Model B: Split bundles + runtime module graph (optional)**
- Multiple bundles or unbundled files are shipped and loaded via `require` at runtime.
- Often paired with a custom resolver or multiple embedded module roots.

Pros/cons matrix:

| Model | Pros | Cons |
| --- | --- | --- |
| A: Single bundle | Simplest runtime loader; single embedded artifact; easy to deploy | Larger bundle; rebuild required for any change; harder to tree-shake at runtime |
| B: Split bundles | Smaller updates per module; can lazily load modules; easier to share code across bundles | Requires more complex loader; higher runtime I/O; more moving parts to embed |

## CommonJS affordances in Goja
CommonJS is a good fit for Goja because `require()` and `module.exports` are already part of the Goja NodeJS compatibility layer. Your bundled output should be CommonJS so that Goja can execute it without an ESM loader.

CommonJS patterns you can rely on:
- `require()` resolves modules through the loader you provide.
- `module.exports` and `exports` control what the bundle exposes.
- Modules are cached after the first load, so repeated `require()` calls reuse the same instance.

Example module export:
```js
function run() {
  return "hello from bundle";
}

module.exports = { run };
```

Example consumer inside Goja:
```go
vm, req := engine.NewWithOptions(require.WithLoader(embeddedSourceLoader))
mod, err := req.Require("./assets/bundle.cjs")
if err != nil {
    log.Fatalf("require bundle: %v", err)
}

exports := mod.ToObject(vm)
run, ok := goja.AssertFunction(exports.Get("run"))
if !ok {
    log.Fatalf("bundle export 'run' is not a function")
}
```

For repeated runtime creation (worker pools, high-throughput request handlers),
use a reusable factory to reduce setup overhead:

```go
factory := engine.NewFactory(
    engine.WithRequireOptions(require.WithLoader(embeddedSourceLoader)),
)

vm, req := factory.NewRuntime()
mod, err := req.Require("./assets/bundle.cjs")
if err != nil {
    log.Fatalf("require bundle: %v", err)
}
_ = vm
_ = mod
```

## Demo layout
A self-contained demo keeps the JS workspace, bundle, and Go entrypoint in one directory so it is easy to copy or vendor. This is the layout used by `cmd/bun-demo`:

```
cmd/bun-demo/
  Makefile
  main.go
  assets/
    bundle.cjs
  js/
    package.json
    bun.lock
    tsconfig.json
    src/
      main.ts
      assets/
        logo.svg
      types/
        goja-modules.d.ts
```

## Step-by-step setup
Each step focuses on a single moving part: the JS workspace, the TypeScript config, the bundler, and the Go loader. Keep each file minimal and focused so the workflow is easy to reason about.

### 1) Initialize the Bun workspace
The Bun workspace manages npm dependencies and scripts. You can create the workspace inside `cmd/bun-demo/js` and install common libraries.

```bash
cd cmd/bun-demo/js
bun init
bun add dayjs lodash
bun add -d typescript esbuild @types/lodash
```

### 2) Configure TypeScript and ambient types
TypeScript should target ES5 so Goja can execute the output. Add ambient type declarations for Goja native modules and asset imports.

`cmd/bun-demo/js/tsconfig.json`:
```json
{
  "compilerOptions": {
    "target": "ES5",
    "module": "ESNext",
    "moduleResolution": "bundler",
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "forceConsistentCasingInFileNames": true,
    "baseUrl": "."
  },
  "include": ["src/**/*"]
}
```

`cmd/bun-demo/js/src/types/goja-modules.d.ts`:
```ts
declare module "fs";
declare module "exec";
declare module "database";

declare module "*.svg" {
  const content: string;
  export default content;
}
```

### 3) Write the TypeScript entrypoint
The entrypoint can use modern TS syntax, import assets, and export a single `run` function for Goja to invoke.

`cmd/bun-demo/js/src/main.ts`:
```ts
import dayjs from "dayjs";
import _ from "lodash";
import logoSvg from "./assets/logo.svg";

function countTags(svg: string): number {
  return (svg.match(/</g) || []).length;
}

export function run(): string {
  var items = [1, 2, 3, 4];
  var sum = _.sum(items);
  var svgTags = countTags(logoSvg);

  return [
    "date=" + dayjs().format("YYYY-MM-DD"),
    "sum=" + sum,
    "svgLen=" + logoSvg.length,
    "svgTags=" + svgTags,
  ].join(" ");
}
```

### 4) Configure the bundler
Bun manages dependencies, but esbuild performs the bundling so you can control the output format and asset loaders. The key flags are `--format=cjs` (CommonJS), `--target=es5`, and `--loader:.svg=text`.

`cmd/bun-demo/js/package.json`:
```json
{
  "name": "goja-bun-demo",
  "private": true,
  "type": "commonjs",
  "scripts": {
    "build": "esbuild src/main.ts --bundle --platform=node --format=cjs --target=es5 --loader:.svg=text --outfile=dist/bundle.cjs --external:fs --external:exec --external:database",
    "typecheck": "tsc --noEmit"
  },
  "dependencies": {
    "dayjs": "^1.11.10",
    "lodash": "^4.17.21"
  },
  "devDependencies": {
    "@types/lodash": "^4.14.202",
    "esbuild": "^0.25.0",
    "typescript": "^5.4.5"
  }
}
```

### 5) Add a demo Makefile
The demo Makefile ties the steps together and keeps the bundling commands co-located with the demo.

`cmd/bun-demo/Makefile`:
```makefile
JS_DIR=js
BUN_ASSET_DIR=assets
BUN_ASSET=$(BUN_ASSET_DIR)/bundle.cjs

js-install:
	cd $(JS_DIR) && bun install

js-typecheck: js-install
	cd $(JS_DIR) && bun run typecheck

js-bundle: js-install
	cd $(JS_DIR) && bun run build
	mkdir -p $(BUN_ASSET_DIR)
	cp $(JS_DIR)/dist/bundle.cjs $(BUN_ASSET)

go-run-bun: js-bundle
	go run .
```

### 6) Embed and load the bundle in Go
Go embeds the bundle and provides a loader for Goja's CommonJS `require`. The loader maps bundle paths to embedded content and errors out if the file does not exist.

`cmd/bun-demo/main.go` (excerpt):
```go
//go:embed assets/bundle.cjs
var bundleFS embed.FS

func embeddedSourceLoader(path string) ([]byte, error) {
    cleaned := strings.TrimPrefix(path, "./")
    cleaned = strings.TrimPrefix(cleaned, "/")

    data, err := bundleFS.ReadFile(cleaned)
    if err == nil {
        return data, nil
    }
    if errors.Is(err, fs.ErrNotExist) {
        return nil, require.ModuleFileDoesNotExistError
    }
    return nil, err
}

vm, req := engine.NewWithOptions(require.WithLoader(embeddedSourceLoader))
mod, err := req.Require("./assets/bundle.cjs")
```

## Running and validating
The quickest validation is running the demo Makefile target and inspecting the output. The output should include the SVG length and tag count, which proves the asset import survived bundling.

```bash
make -C cmd/bun-demo go-run-bun
```

Expected output (example):
```
date=2026-01-10 sum=5 svgLen=191 svgTags=4
```

## Model B: split bundles in practice
The split-bundle workflow demonstrates a runtime module graph: `app.js` is bundled, but it still `require()`s `modules/metrics.js` at runtime. This keeps modules decoupled and makes it easier to ship independent bundles while still relying on CommonJS execution in Goja.

The demo uses a separate entrypoint and output directory:
- `cmd/bun-demo/js/src/split/app.ts` is the entrypoint.
- `cmd/bun-demo/js/src/split/modules/metrics.ts` is bundled separately and loaded via `require`.
- Outputs are copied into `cmd/bun-demo/assets-split/`.

Build and run the split demo:
```bash
make -C cmd/bun-demo go-run-bun-split
```

Expected output (example):
```
mode=split date=2026-01-10 svgLen=191 svgTags=4 svgCsum=13804
```

## Extending for real apps
Once the pipeline works for the demo, you can scale it to larger projects by treating the JS workspace like any other app repository. The key is to keep the output CommonJS and keep host-provided modules external.

Recommended practices:
- Keep one bundle per logical plugin or entrypoint.
- Add more asset loaders via esbuild flags (`--loader:.json=text`, `--loader:.txt=text`, etc.).
- Use path aliases in `tsconfig.json` only if your bundler supports them.
- Add more `--external:` flags for any modules provided by Go or injected by the host.

## Troubleshooting
Most issues come down to module format, target level, or missing loaders. Use these fixes when bundling or runtime errors appear.

Common fixes:
- **"Unexpected token 'export'"**: Ensure `--format=cjs` is set and `"type": "commonjs"` is in `package.json`.
- **"SyntaxError: Unexpected identifier"**: Ensure `--target=es5` and avoid ES2015+ runtime features without transpilation.
- **"Cannot find module 'fs'"**: Add `--external:fs` (and other native modules) to the build script.
- **SVG import returns empty string**: Verify `--loader:.svg=text` and that the asset is under `src/`.

## API reference
This section summarizes the integration points between Goja and the bundled output. Use these as the stable contract for your bundling pipeline.

**Bundle entrypoint**
- Exports: `run()` function on `module.exports` or `exports`.
- Format: CommonJS (`--format=cjs`).
- Target: ES5 (`--target=es5`).

**Goja loader**
- `engine.NewWithOptions(require.WithLoader(loader))` configures Goja to load embedded sources.
- `engine.NewFactory(engine.WithRequireOptions(require.WithLoader(loader)))` prebuilds runtime bootstrap state for repeated runtime creation.
- `require.ModuleFileDoesNotExistError` signals to Goja that the module path is missing.

**Makefile targets**
- `js-install`: install npm dependencies using Bun.
- `js-typecheck`: run `tsc --noEmit` for TS validation.
- `js-bundle`: produce `dist/bundle.cjs` and copy to `assets/bundle.cjs`.
- `js-bundle-split`: produce `dist-split` outputs and copy to `assets-split`.
- `go-run-bun`: build the bundle and run the Go demo.
- `go-run-bun-split`: build the split outputs and run the split demo entrypoint.

## Testing checklist
Use this checklist before shipping a new bundle or updating dependencies.

- Run `make -C cmd/bun-demo js-typecheck` to ensure TS types are clean.
- Run `make -C cmd/bun-demo js-bundle` and confirm `assets/bundle.cjs` updates.
- Run `make -C cmd/bun-demo go-run-bun` and confirm SVG metrics output.
- Run `make -C cmd/bun-demo go-run-bun-split` and confirm the split demo output.
- Review `cmd/bun-demo/js/package.json` to ensure native modules remain external.
