---
Title: TypeScript and asset import bundling analysis
Ticket: BUN-003
Status: active
Topics:
    - bun
    - bundling
    - typescript
    - assets
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: Makefile
      Note: Bundling targets for TS + SVG
    - Path: js/package.json
      Note: esbuild build and typecheck scripts
    - Path: js/src/assets/logo.svg
      Note: SVG asset import example
    - Path: js/src/main.ts
      Note: TypeScript demo entry
    - Path: js/src/types/goja-modules.d.ts
      Note: Module typings for Goja modules
    - Path: js/tsconfig.json
      Note: TS compiler settings
ExternalSources: []
Summary: Plan for TypeScript support and SVG asset imports in the Goja bundling pipeline.
LastUpdated: 2026-01-10T20:47:50-05:00
WhatFor: Define the bundler configuration and file layout for TS + asset imports.
WhenToUse: When implementing or reviewing TS and asset import support for the demo pipeline.
---


# TypeScript and asset import bundling analysis

## Goal
Add TypeScript support and allow importing asset types (ex: `.svg`) in the demo bundling pipeline, while keeping the CommonJS output compatible with goja_nodejs/require.

## Requirements
- Bundle TypeScript entrypoints with minimal friction (no manual pre-transpile step).
- Support importing `.svg` as a string (for runtime use or embedding).
- Keep CommonJS output and externalize native Go modules (`fs`, `exec`, `database`).
- Preserve Goja compatibility (ES5 target).

## Current state
- The demo uses `js/src/main.js` with CommonJS requires.
- Bundling is done via `bun build` to CJS and copies into `cmd/bun-demo/assets/bundle.cjs`.
- Bun does not expose a loader flag for assets, which is required for `.svg` imports.

## Proposed approach
Use `esbuild` as the bundler (invoked via bun) to gain loader support and native TypeScript handling.

- `esbuild` can bundle TypeScript directly (`.ts` input).
- `--loader:.svg=text` returns SVG file contents as a string.
- `--platform=node` preserves `require("fs")` and aligns with the CommonJS runtime.
- `--target=es5` keeps Goja compatibility.

### Build command (Makefile / package.json)
```
esbuild src/main.ts \
  --bundle \
  --platform=node \
  --format=cjs \
  --target=es5 \
  --loader:.svg=text \
  --outfile=dist/bundle.cjs \
  --external:fs \
  --external:exec \
  --external:database
```

## TypeScript ergonomics
- Add `tsconfig.json` with strict type checking and bundler-style module resolution.
- Provide local `.d.ts` modules for `fs`, `exec`, `database`, and `*.svg`.
- Keep `type` as `commonjs` in `package.json` so runtime expectations stay consistent.

## Asset import format
- `.svg` imports return a string (raw XML). This can be hashed, embedded into output, or passed to host APIs.
- Use a small sample SVG in `js/src/assets/logo.svg` to validate the pipeline.

## Risks and mitigations
- **Large assets**: `.svg` imported as text increases bundle size. Mitigation: keep small demo assets and document limitations.
- **Type drift**: the Goja module APIs are not typed. Mitigation: maintain minimal `.d.ts` declarations.
- **Compatibility**: esbuild target might need tuning if packages emit syntax beyond ES5. Mitigation: enforce `--target=es5` and validate Goja execution.
- **ES5 downleveling**: esbuild may reject `const` when targeting ES5. Mitigation: prefer `var` in demo sources or adjust the target.

## Validation plan (summary)
- `bun run build` produces a CJS bundle.
- `rg 'require\("fs"\)' dist/bundle.cjs` to confirm externals remain.
- `make go-run-bun` prints a value that includes `logo.length` (proof SVG import worked).
- `bun x tsc --noEmit` passes for the demo sources.

## Related files
- `js/src/main.ts`
- `js/src/assets/logo.svg`
- `js/src/types/*.d.ts`
- `js/tsconfig.json`
- `js/package.json`
- `Makefile`
