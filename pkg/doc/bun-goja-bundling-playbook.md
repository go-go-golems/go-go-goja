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

The `cmd/bun-demo` runtime path now also supports HashiCorp plugin-backed modules through the same `--plugin-dir` and `--allow-plugin-module` flags exposed by the REPLs. That means a bundled CommonJS app can `require("plugin:...")` as long as the runtime entrypoint opts into plugin discovery.

## Architecture and flow
The bundling pipeline is a simple, reproducible assembly line: Bun installs dependencies, esbuild bundles the TS entrypoint into a single CommonJS file, Go embeds that file, and Goja `require()` executes it at runtime. This keeps the runtime loader focused on CommonJS and avoids shipping a full Node runtime.

High-level flow:
- Author TS/JS sources in `cmd/bun-demo/js/src`.
- Run `make -C cmd/bun-demo js-install` to install deps via the demo pipeline.
- Bundle with esbuild into `cmd/bun-demo/js/dist/bundle.cjs`.
- Generate native module declarations with `go generate ./cmd/bun-demo`.
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
factory, err := engine.NewBuilder().
    WithRequireOptions(require.WithLoader(embeddedSourceLoader)).
    WithModules(engine.DefaultRegistryModules()).
    Build()
if err != nil {
    log.Fatalf("build factory: %v", err)
}

rt, err := factory.NewRuntime(context.Background())
if err != nil {
    log.Fatalf("new runtime: %v", err)
}
defer rt.Close(context.Background())

mod, err := rt.Require.Require("./assets/bundle.cjs")
if err != nil {
    log.Fatalf("require bundle: %v", err)
}

exports := mod.ToObject(rt.VM)
run, ok := goja.AssertFunction(exports.Get("run"))
if !ok {
    log.Fatalf("bundle export 'run' is not a function")
}
```

For repeated runtime creation (worker pools, high-throughput request handlers),
use a reusable factory to reduce setup overhead:

```go
factory, err := engine.NewBuilder().
    WithRequireOptions(require.WithLoader(embeddedSourceLoader)).
    WithModules(engine.DefaultRegistryModules()).
    Build()
if err != nil {
    log.Fatalf("build factory: %v", err)
}

rt, err := factory.NewRuntime(context.Background())
if err != nil {
    log.Fatalf("new runtime: %v", err)
}
defer rt.Close(context.Background())

mod, err := rt.Require.Require("./assets/bundle.cjs")
if err != nil {
    log.Fatalf("require bundle: %v", err)
}
_ = mod
```

## Demo layout
A self-contained demo keeps the JS workspace, bundle, and Go entrypoint in one directory so it is easy to copy or vendor. This is the layout used by `cmd/bun-demo`:

```

## Plugin-backed runtime extensions

If your bundled code wants to call plugin-backed modules in addition to built-in native modules, build the plugin into the default plugin tree and start the demo with the same flags used by the REPLs.

Example:

```bash
mkdir -p ~/.go-go-goja/plugins/examples
go build -o ~/.go-go-goja/plugins/examples/goja-plugin-examples-greeter ./plugins/examples/greeter
go run ./cmd/bun-demo --plugin-dir ~/.go-go-goja/plugins/examples
```

Inside the bundle, the JavaScript side can then call:

```javascript
const greeter = require("plugin:examples:greeter")
```

If you want to constrain the runtime to a known plugin module name, add:

```bash
--allow-plugin-module plugin:examples:greeter
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
        assets.d.ts
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

### 2) Configure TypeScript and declaration files
TypeScript should target ES5 so Goja can execute the output. Native module declarations are generated by `cmd/gen-dts`, while asset declarations remain hand-authored.

`cmd/bun-demo/js/tsconfig.json`:
```json
{
  "compilerOptions": {
    "target": "ES5",
    "module": "ESNext",
    "moduleResolution": "Bundler",
    "jsx": "react",
    "jsxFactory": "jsx",
    "jsxFragmentFactory": "Fragment",
    "noEmit": true,
    "strict": true,
    "skipLibCheck": true,
    "resolveJsonModule": true
  },
  "include": ["src/**/*"]
}
```

`cmd/bun-demo/js/src/types/goja-modules.d.ts`:
```ts
// Code generated by go-go-goja/cmd/gen-dts. DO NOT EDIT.

declare module "database" {
  export function close(): void;
  export function configure(driverName: string, dataSourceName: string): void;
  export function exec(query: string, ...args: unknown[]): unknown;
  export function query(query: string, ...args: unknown[]): unknown;
}

declare module "exec" {
  export function run(cmd: string, args: string[]): string;
}

declare module "fs" {
  export function readFileSync(path: string): string;
  export function writeFileSync(path: string, data: string): void;
}
```

`cmd/bun-demo/js/src/types/assets.d.ts`:
```ts
declare module "*.svg" {
  const content: string;
  export default content;
}
```

Regenerate declarations after changing native module descriptors:
```bash
cd go-go-goja
go generate ./cmd/bun-demo
git diff --exit-code cmd/bun-demo/js/src/types/goja-modules.d.ts
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
DAGGER?=dagger
DAGGER_RUN=$(DAGGER) run --
DAGGER_PIPELINE=$(DAGGER_RUN) go run ./dagger --project-dir .

js-install:
	$(DAGGER_PIPELINE) deps

js-typecheck:
	$(DAGGER_PIPELINE) typecheck

js-bundle:
	$(DAGGER_PIPELINE) bundle

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

factory, err := engine.NewBuilder().
    WithRequireOptions(require.WithLoader(embeddedSourceLoader)).
    WithModules(engine.DefaultRegistryModules()).
    Build()
if err != nil {
    log.Fatalf("build engine factory: %v", err)
}

rt, err := factory.NewRuntime(context.Background())
if err != nil {
    log.Fatalf("create runtime: %v", err)
}
defer rt.Close(context.Background())

mod, err := rt.Require.Require("./assets/bundle.cjs")
```

## Running and validating
The quickest validation is running declaration generation and then the demo targets. This confirms both static types and runtime bundle behavior.

```bash
go generate ./cmd/bun-demo
git diff --exit-code cmd/bun-demo/js/src/types/goja-modules.d.ts
```

Then run the demo:

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
Most issues come down to module format, transpilation target, declaration drift, or missing loaders.

| Problem | Cause | Solution |
| --- | --- | --- |
| `Unexpected token 'export'` | Bundle not emitted as CommonJS | Ensure `--format=cjs` and `"type": "commonjs"` |
| `SyntaxError: Unexpected identifier` | Output targets unsupported runtime syntax | Ensure `--target=es5` and avoid unsupported runtime features |
| `Cannot find module 'fs'` | Native modules were bundled or unresolved at runtime | Keep native modules external (`--external:fs`, `--external:exec`, `--external:database`) |
| Declaration diff appears after `go generate ./cmd/bun-demo` | Generated declaration file drifted | Review and commit the generated `goja-modules.d.ts` diff |
| SVG import type errors | Asset ambient declaration missing | Keep `declare module "*.svg"` in `assets.d.ts` |

## API reference
This section summarizes the integration points between Goja and the bundled output. Use these as the stable contract for your bundling pipeline.

**Bundle entrypoint**
- Exports: `run()` function on `module.exports` or `exports`.
- Format: CommonJS (`--format=cjs`).
- Target: ES5 (`--target=es5`).

**Goja loader**
- `engine.NewBuilder().WithRequireOptions(require.WithLoader(loader)).WithModules(engine.DefaultRegistryModules()).Build()` configures runtime creation.
- `factory.NewRuntime(context.Background())` creates an owned runtime that should be closed explicitly.
- `require.ModuleFileDoesNotExistError` signals to Goja that the module path is missing.

**Makefile targets**
- `js-install`: install npm dependencies via the demo Dagger pipeline.
- `js-typecheck`: run TypeScript checks in the demo pipeline.
- `js-bundle`: produce `dist/bundle.cjs` and copy to `assets/bundle.cjs`.
- `js-bundle-split`: produce `dist-split` outputs and copy to `assets-split`.
- `go-run-bun`: build the bundle and run the Go demo.
- `go-run-bun-split`: build the split outputs and run the split demo entrypoint.
Native module declarations are generated by `go generate ./cmd/bun-demo`; there are no Makefile wrappers for this workflow.

## Testing checklist
Use this checklist before shipping a new bundle or updating dependencies.

- Run `make -C cmd/bun-demo js-typecheck` to ensure TS types are clean.
- Run `go generate ./cmd/bun-demo` and review/commit any `goja-modules.d.ts` diff to ensure native module declarations are up to date.
- Run `make -C cmd/bun-demo js-bundle` and confirm `assets/bundle.cjs` updates.
- Run `make -C cmd/bun-demo go-run-bun` and confirm SVG metrics output.
- Run `make -C cmd/bun-demo go-run-bun-split` and confirm the split demo output.
- Review `cmd/bun-demo/js/package.json` to ensure native modules remain external.
