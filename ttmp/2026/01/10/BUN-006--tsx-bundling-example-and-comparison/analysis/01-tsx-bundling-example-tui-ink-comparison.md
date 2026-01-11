---
Title: TSX bundling example + tui-ink comparison
Ticket: BUN-006
Status: active
Topics:
    - bun
    - bundling
    - tsx
    - goja
    - analysis
    - docs
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../../code/wesen/corporate-headquarters/vibes/2025/06/22/tui-ink/go-app/main.go
      Note: tui-ink Goja loader contract
    - Path: ../../../../../../../../../../code/wesen/corporate-headquarters/vibes/2025/06/22/tui-ink/js-modules/package.json
      Note: tui-ink bundler scripts and deps
    - Path: ../../../../../../../../../../code/wesen/corporate-headquarters/vibes/2025/06/22/tui-ink/js-modules/webpack.config.js
      Note: tui-ink webpack ES5 + JSX configuration
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-10T22:22:29.760621758-05:00
WhatFor: Document a TSX-to-JS/HTML pipeline for Goja and compare it to tui-ink's approach.
WhenToUse: Use when planning TSX demos or deciding on a bundler/runtime contract.
---


# TSX bundling example + tui-ink comparison

## Overview
This analysis documents a TSX-based example pipeline that compiles components into CommonJS for Goja and renders HTML strings. It also compares the pipeline with the tui-ink project, which uses webpack + Babel to output a global library for Goja consumption.

## Context and goals
TSX support introduces JSX transforms, runtime helpers, and potential SSR output. For Goja, we need an ES5-compatible CommonJS bundle and a predictable export surface (`run()` or `render()`), so the runtime can `require()` a single entrypoint and get a string or HTML.

Goals for the TSX example:
- Keep the output CommonJS (`--format=cjs`) and ES5 (`--target=es5`).
- Render TSX to HTML on the JS side so Go can use the HTML string.
- Provide a minimal, copy/paste-friendly file layout.
- Compare with tui-ink's webpack + Babel approach and note differences.

## Example pipeline: TSX to HTML string (Goja-friendly)
This example uses a tiny custom JSX-to-string runtime so the bundle can remain ES5 without extra transpilation. The output is a CommonJS module that exports `renderHtml()` and can be executed via Goja's `require()`.

Note: the current esbuild version in this repo does not downlevel `const` to ES5, so dependencies that ship modern syntax would require an additional Babel or SWC step. The custom runtime avoids that extra pass.

### Suggested file layout
```
cmd/bun-demo/js/
  src/
    tsx/
      App.tsx
      render.tsx
      entry.tsx
      runtime.ts
  package.json
  tsconfig.json
```

### Implementation in this repo
The example has been implemented under `cmd/bun-demo/js/src/tsx` with the bundler script wired into `cmd/bun-demo/js/package.json`. The compiled output is copied to `cmd/bun-demo/assets/tsx-bundle.cjs` via `make -C cmd/bun-demo js-bundle-tsx` and can be executed using `make -C cmd/bun-demo go-run-bun-tsx`.

### TSX component
`src/tsx/App.tsx`:
```tsx
import { Fragment, jsx } from "./runtime";

export function App(props: { title: string; items: string[] }) {
  return (
    <main>
      <header>
        <h1>{props.title}</h1>
        <p>Rendered from TSX inside Goja.</p>
      </header>
      <section>
        <ul>
          {props.items.map(function (item) {
            return <li key={item}>{item}</li>;
          })}
        </ul>
      </section>
    </main>
  );
}
```

### Render function
`src/tsx/render.tsx`:
```tsx
import { Fragment, jsx, renderToString } from "./runtime";
import { App } from "./App";

export function renderHtml(): string {
  var html = renderToString(
    <App title="Goja TSX Demo" items={["alpha", "beta", "gamma"]} />
  );

  return "<!doctype html>" + html;
}
```

### Bundler scripts (Bun + esbuild)
`package.json` scripts:
```json
{
  "scripts": {
    "build:tsx": "esbuild src/tsx/entry.tsx --bundle --platform=node --format=cjs --target=es5 --jsx=transform --jsx-factory=jsx --jsx-fragment=Fragment --outfile=dist/tsx-bundle.cjs",
    "render:tsx-html": "bun -e \"const m = require('./dist/tsx-bundle.cjs'); console.log(m.renderHtml ? m.renderHtml() : m.run());\""
  }
}
```

Key flags:
- `--jsx=transform` compiles JSX into `jsx()` calls.
- `--jsx-factory=jsx` and `--jsx-fragment=Fragment` point to the custom runtime.
- `--format=cjs` produces CommonJS for Goja.
- `--target=es5` ensures Goja compatibility.

### Goja integration sketch
The Go runtime loads the CommonJS bundle and calls `renderHtml()`.

```go
mod, err := req.Require("./assets/tsx-bundle.cjs")
if err != nil {
    log.Fatalf("require tsx bundle: %v", err)
}

exports := mod.ToObject(vm)
renderVal := exports.Get("renderHtml")
render, ok := goja.AssertFunction(renderVal)
if !ok {
    log.Fatalf("renderHtml export is not a function")
}

result, err := render(goja.Undefined())
if err != nil {
    log.Fatalf("renderHtml(): %v", err)
}

fmt.Println(result.Export())
```

This pattern mirrors the existing Goja bundle entrypoints in the bun demo.

## Alternative output: pre-rendered HTML file
If you want a build-time HTML artifact, use the `render:tsx-html` script to emit HTML and redirect it into a file:

```bash
bun run build:tsx
bun run render:tsx-html > dist/index.html
```

This keeps JS bundling and HTML generation separate from the Go runtime. The Go app can still embed and serve `dist/index.html` as a static asset if needed.

In this repo, `make -C cmd/bun-demo js-render-tsx` writes `cmd/bun-demo/assets/tsx.html` from the bundled output.

## Comparison with tui-ink
The tui-ink project uses webpack + Babel to build ES5 JavaScript bundles and then executes them in Goja. It chooses a global variable export instead of CommonJS, which is a different runtime contract.

Key details from tui-ink:
- Webpack entry: `js-modules/src/enhanced-tui.js`.
- Babel handles JSX via `@babel/preset-react`.
- Output is a global variable (`SimpleTUILib`) with `libraryTarget: 'var'`.
- Go reads and executes the bundle with `RunString`, then grabs `SimpleTUILib` from global scope.

Relevant files:
- `/home/manuel/code/wesen/corporate-headquarters/vibes/2025/06/22/tui-ink/js-modules/webpack.config.js`
- `/home/manuel/code/wesen/corporate-headquarters/vibes/2025/06/22/tui-ink/js-modules/package.json`
- `/home/manuel/code/wesen/corporate-headquarters/vibes/2025/06/22/tui-ink/go-app/main.go`

### Contract differences
- **CommonJS require (go-go-goja)**: `require("./bundle.cjs")` returns an exports object. This is explicit and modular, but requires embedding the bundle file and a loader.
- **Global var (tui-ink)**: The bundle registers a global symbol. This simplifies loading (just `RunString`), but makes module boundaries and naming collisions harder to manage.

### Bundler differences
- **Bun + esbuild**: fast, single-file command; supports TS/TSX directly. Easier to keep output CommonJS + ES5 in one step.
- **Webpack + Babel**: more configurable but heavier; handles JSX via Babel and outputs a library target for global injection.

### Compatibility considerations
- Goja prefers ES5. Both approaches target ES5, but webpack's `target: ['web', 'es5']` is explicit and uses Babel for JSX.
- Bun/esbuild requires `--target=es5` and a JSX runtime config (`--jsx=automatic` or `--jsx-factory`).

## Decision guide
Use this decision guide to choose an approach based on runtime constraints.

- Choose **CommonJS bundling** when you want clear module boundaries, multiple entrypoints, or a `require()`-based loader.
- Choose **global library bundling** when you want the simplest loader and are okay with a single global export surface.
- Use **TSX + render-to-string** when you need HTML output for rendering, templating, or static generation.

## Risks and follow-ups
The TSX example depends on JSX runtime choices and SSR libraries, which can pull in ES2015 features if not carefully transpiled. Stick to ES5 targets and avoid polyfill-heavy dependencies.

Follow-ups if needed:
- Add a real TSX demo to the bun workspace once this analysis is approved.
- Define a shared convention for export names (`renderHtml`, `run`, etc.).
- Decide whether to keep HTML generation in JS or pre-generate HTML at build time.
