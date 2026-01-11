---
Title: Package Goja research
Ticket: BUN-001
Status: active
Topics:
    - goja
    - bun
    - bundling
    - javascript
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../../../../tmp/package-goja-research.md
      Note: Imported source text
ExternalSources:
    - https://pkg.go.dev/github.com/dop251/goja
    - https://github.com/grafana/grafana-foundation-sdk/issues/560
    - https://grafana.com/docs/k6/latest/using-k6/modules/
    - https://github.com/dop251/goja_nodejs
    - https://github.com/tliron/commonjs-goja
    - https://developers.cloudflare.com/workers/wrangler/bundling/
    - https://github.com/bellard/quickjs/issues/191
Summary: Research notes on bundling npm-managed JS for Goja and common pipeline constraints.
LastUpdated: 2026-01-10T18:46:33-05:00
WhatFor: Document constraints and bundling models for running JS in Goja.
WhenToUse: When selecting bundlers or packaging options for Goja.
---


# Package Goja research

## Goal
Capture research on bundling npm-managed JS for Goja runtimes, including constraints, workable models, and pipeline examples.

## Context
- Goja is ES5.1, so bundle output must avoid ESM-only syntax.
- Goja does not provide a native module loader; bundling or a CommonJS shim is required.

## Quick Reference
Source note: imported from `/tmp/package-goja-research.md` on 2026-01-10.

### Overview
You can absolutely use **npm/pnpm/bun as the dependency/locking layer**, and then treat a **bundler as the "compiler" that adapts Node-ish packages into something your Goja host can execute**. This is a pretty common pattern for "JS, but not Node" runtimes.

The key constraint: **Goja is ES5.1(+)** ([Go Packages][1]) and (in practice) you should assume **no native ESM module loader** ("import/export") unless you build/provide one; people run into this with Goja-based runtimes ([GitHub][2]). So you typically **bundle/transpile**.

### Two workable models

#### Model A (most common): "bundle to one file" (no module loader in Goja)

1. Use npm/pnpm/bun to install deps (lockfile, workspaces, versioning).
2. Use a bundler to resolve `node_modules`, inline what's needed, and emit a **single JS file** (IIFE/UMD).
3. Your Go code loads and runs that one file in Goja.

This is exactly how **Grafana k6** (Go + Goja) recommends using npm dependencies: k6 itself doesn't resolve Node modules, so you bundle them (they even provide a Rollup example repo). ([Grafana Labs][3])

#### Model B: CommonJS runtime in Goja + bundle-to-CJS (or even unbundled)

If you want `require()` semantics at runtime, use a compatibility layer like:

- `dop251/goja_nodejs/require` ([GitHub][4])
- or other CommonJS loaders built for Goja ([GitHub][5])

Then your bundler can emit **CommonJS**, or you can load a small set of files and let `require()` resolve them (still tricky once you hit modern `exports` rules, conditional exports, etc.).

In practice: Model A is simpler and more reproducible.

### A concrete pipeline that works well with Goja

#### 1) Install deps normally

Use whichever you like:

- `pnpm` (great for workspaces)
- `npm`
- `bun install`

#### 2) Bundle with esbuild (example)

Bundle to an IIFE so Goja just executes it:

```bash
# example
pnpm add -D esbuild
pnpm esbuild src/main.ts \
  --bundle \
  --format=iife \
  --platform=browser \
  --target=es5 \
  --outfile=dist/app.bundle.js
```

Why `--platform=browser`? It nudges resolution toward "no Node built-ins" variants (where libraries ship them), similar to how restricted runtimes (Workers, etc.) lean on bundling + non-Node entrypoints. Cloudflare Workers' toolchain does this by default: Wrangler bundles with esbuild and supports importing npm modules, even though Workers is not Node. ([Cloudflare Docs][6])

#### 3) Run in Goja

Then in Go you just read `dist/app.bundle.js` and `runtime.RunString(...)`.

If instead you want CommonJS at runtime:

```bash
pnpm esbuild src/main.ts --bundle --format=cjs --platform=browser --target=es5 --outfile=dist/app.cjs
```

...and enable `require()` via goja_nodejs. ([GitHub][4])

### What breaks (and how people usually handle it)

- **Node built-ins** (`fs`, `net`, `child_process`) and **native addons** won't work. You either:
  - avoid those deps,
  - pick packages that ship browser/universal builds,
  - or provide explicit shims (sometimes as "empty" modules) during bundling.
- **Dynamic `require()` / dynamic import paths** can defeat bundlers.
- Some packages assume globals (`process`, `Buffer`) -- you can shim these at bundle time or in the Goja global scope.

A good mental model is: you're doing the same thing as other "restricted JS runtimes":

- **k6 (Goja)**: bundle npm deps externally ([Grafana Labs][3])
- **Cloudflare Workers**: npm deps -> bundled artifact uploaded to runtime ([Cloudflare Docs][6])
- **QuickJS embedding**: people also solve "TS + npm deps but not Node" by bundling to a single file with Rollup/esbuild/etc. ([GitHub][7])

### Practical recommendation for your Goja VM

If your goal is "applications" (scripts/plugins) that run inside your API surface:

- Standardize on **"bundle-first"** (Model A).
- Publish plugins as npm packages if you want (versioning, semver, lockfiles), but at install/build time produce a **runtime artifact** (`dist/bundle.js`) that your Go app actually loads.
- Keep your runtime API stable and expose it either as `globalThis.host` or as a "virtual module" that the bundler marks as external (so it doesn't try to inline it).

If you tell me:

1. do you want plugin authors to write **ESM** (`import`) or **CommonJS** (`require`)?
2. do you want **multiple files** loaded at runtime, or is "single bundle per plugin" fine?

...I can sketch the cleanest packaging layout (`package.json` fields, build scripts, and what your Go side should expect).

## Usage Examples

```bash
# Install dependencies
bun install

# Bundle for Goja (example; see design doc for the bun-first pipeline)
# bun build --target=browser --format=iife --outfile=dist/bundle.js src/main.ts
```

## Related
- [Bun bundling design + analysis](../design/01-bun-bundling-design-analysis.md)

[1]: https://pkg.go.dev/github.com/dop251/goja "goja package - github.com/dop251/goja"
[2]: https://github.com/grafana/grafana-foundation-sdk/issues/560 "[Feature]: Support CommonJS modules - Issue #560"
[3]: https://grafana.com/docs/k6/latest/using-k6/modules/ "Modules - Grafana k6 documentation"
[4]: https://github.com/dop251/goja_nodejs "Nodejs compatibility library for Goja"
[5]: https://github.com/tliron/commonjs-goja "CommonJS support for Goja JavaScript engine for Go"
[6]: https://developers.cloudflare.com/workers/wrangler/bundling/ "Bundling - Workers"
[7]: https://github.com/bellard/quickjs/issues/191 "Recommendation to transpile and bundle typescript for QuickJS"
