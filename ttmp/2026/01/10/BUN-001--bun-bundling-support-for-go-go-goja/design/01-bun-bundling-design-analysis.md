---
Title: Bun bundling design + analysis
Ticket: BUN-001
Status: active
Topics:
    - goja
    - bun
    - bundling
    - javascript
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../../../../tmp/package-goja-research.md
      Note: Original research notes imported into reference doc
    - Path: go-go-goja/engine/runtime.go
      Note: Existing CommonJS runtime setup and require() integration
    - Path: go-go-goja/modules/common.go
      Note: Native module registry used by require
    - Path: go-go-goja/ttmp/2026/01/10/BUN-001--bun-bundling-support-for-go-go-goja/reference/01-package-goja-research.md
      Note: Research summary used for design decisions
ExternalSources: []
Summary: Design and analysis for bundling npm-managed JS with bun and running it in Goja.
LastUpdated: 2026-01-10T19:24:59-05:00
WhatFor: Define bundling model, build pipeline, and test project layout for Goja.
WhenToUse: When implementing bun-based packaging for go-go-goja or reviewing the proposed architecture.
---



# Bun bundling design + analysis

## Goals
- Provide a repeatable pipeline to install npm dependencies with bun and bundle/transpile for Goja.
- Produce a single JS artifact suitable for go:embed.
- Define a small test project and Makefile flow for validation.

## Non-goals
- Build a new Node.js compatibility layer or module loader (we rely on goja_nodejs/require).
- Support unbounded dynamic import/require outside controlled module roots.
- Support native addons or Node built-ins.

## Constraints and assumptions
- Goja is ES5.1 and does not provide a native ESM module loader.
- Bundled output must avoid Node built-ins and native addons.
- Bundling happens at build time; runtime should be offline and deterministic.

## Packaging model (CommonJS-first)
- Primary: Model B (CommonJS) since go-go-goja already enables `require()` via goja_nodejs.
- Optional fallback: Model A (IIFE) when a single self-contained bundle is preferred.

## Current go-go-goja CommonJS runtime
- `engine.New()` creates a goja runtime and enables `require()` via goja_nodejs/require.
- Native modules are registered through `modules.EnableAll(reg)` (current modules: `fs`, `exec`, `database`).
- `engine.New()` returns `*require.RequireModule` so Go can call `req.Require(path)` for an entrypoint.
- The default source loader reads from the host filesystem, so `require("./dist/bundle.cjs")` works when files are on disk.

Implication: bundling should output CommonJS and keep native module names external so `require("fs")` and friends resolve to Go-provided modules.

## CommonJS affordances (Model B)

### What it gives you as a developer
- Runtime `require()` for conditional or feature-flagged loading without rebuilding bundles.
- Multi-file codebases that keep module boundaries for easier debugging and targeted edits.
- A familiar Node-like module system (`module.exports`, `exports`) for teams and existing code.
- Optional plugin loading by name/path at runtime (as long as resolution is controlled).

### Examples

**Split modules with `module.exports`:**
```js
// math.js
module.exports = {
  sum(a, b) {
    return a + b;
  },
};

// main.js
const math = require("./math");
globalThis.run = () => math.sum(2, 3);
```

**Conditional loading at runtime:**
```js
let impl;
if (globalThis.host && globalThis.host.fastMode) {
  impl = require("./fast");
} else {
  impl = require("./safe");
}
```

**Simple plugin loader (runtime-resolved):**
```js
function loadPlugin(name) {
  return require("./plugins/" + name);
}
```

### Tradeoffs vs Model A
- Requires a CommonJS loader and module resolution rules in Goja.
- More runtime surface area (file loading, path handling, caching).
- Still must avoid Node built-ins and keep ES5 syntax.
- Dynamic `require()` can be harder to audit and bundle statically.

## Architecture overview
- JS workspace under `js/` with `package.json`, `bun.lockb`, and `src/`.
- Entry point at `src/main.ts` (or `src/main.js`).
- Bundled output in `dist/bundle.cjs`.
- Go app loads the bundle via `require()` (filesystem or embed-backed loader).
- Native modules (`fs`, `exec`, `database`) remain external and are resolved by go-go-goja's registry.

## Build pipeline (Makefile)
- `make js-install`: run `bun install` in `js/`.
- `make js-bundle`: run `bun build --target=browser --format=cjs --outfile=dist/bundle.cjs src/main.ts --external:fs --external:exec --external:database`.
- `make js-transpile`: optional downlevel pass if bun output is not ES5 (ex: `bun x esbuild dist/bundle.cjs --target=es5 --format=cjs --outfile=dist/bundle.es5.cjs`).
- `make go-build`: ensure bundle exists, then `go build ./...`.
- `make go-run`: build bundle and run the demo.
- `make clean`: remove `js/dist/`.

Note: bun does not expose a language-level `--target` flag in `bun build`. If Goja rejects the bundle, add the explicit downlevel step.

## Test project design

### Packages
- Use browser-safe, common packages: `lodash` and `dayjs`.
- Avoid packages that depend on `fs`, `net`, `crypto`, or native addons.

### JS layout
```
js/
  package.json
  bun.lockb
  src/
    main.ts
  dist/
    bundle.cjs
```

### Example behavior
- `main.ts` uses CommonJS `require()` to import npm packages and native modules.
- Uses dayjs to format a timestamp and lodash to compute a summary.
- Exports a single entry function as `module.exports = { run }`.

Example entry (CJS-style):
```js
const dayjs = require("dayjs");
const { sumBy } = require("lodash");
const fs = require("fs");
const exec = require("exec");

function run() {
  const items = [{ n: 2 }, { n: 3 }];
  const total = sumBy(items, "n");
  const stamp = dayjs().format("YYYY-MM-DD");
  fs.writeFileSync("/tmp/goja-bun.txt", stamp + ":" + total);
  const out = exec.run("cat", ["/tmp/goja-bun.txt"]).trim();
  return out;
}

module.exports = { run };
```

### Go layout
```
cmd/goja-bun-demo/
  main.go
internal/runtime/
  runtime.go
```

- `main.go` embeds `js/dist/bundle.cjs` or reads it from disk.
- Use `engine.New()` and `req.Require("./js/dist/bundle.cjs")` to load the CommonJS entrypoint.
- Call `run()` from the returned `module.exports` and verify output.

Embedding option: use `require.WithLoader` to read from `embed.FS` instead of disk, so `require()` can load embedded modules.

## Validation plan
- Run `make js-install` and `make js-bundle`.
- Run `go test ./...` and a demo run that logs output.
- Confirm Goja executes the bundle without syntax errors.

## Open questions and risks
- Confirm bun bundle output is ES5-compatible for Goja; otherwise add downlevel step.
- Verify `--target=browser` resolves browser entrypoints and avoids Node built-ins.
- Decide whether to shim `process`/`Buffer` in bundler or in runtime.
- Define a safe module resolution policy for embedded assets (path normalization, allowed roots).

## Decision log
- Use CommonJS bundling by default to align with go-go-goja's `require()` runtime.
- Keep IIFE bundling as a fallback for single-file use cases.
- Use bun for dependency management and bundling; add explicit ES5 downlevel step if required.

## References
- [Package Goja research](../reference/01-package-goja-research.md)
