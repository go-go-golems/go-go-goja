---
Title: TypeScript + asset import testing plan
Ticket: BUN-003
Status: active
Topics:
    - bun
    - bundling
    - typescript
    - assets
DocType: playbook
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Real-world testing plan for TS bundling and SVG imports in the Goja demo."
LastUpdated: 2026-01-10T20:41:59-05:00
WhatFor: "Define test coverage for TS, SVG assets, and Goja execution."
WhenToUse: "When validating TypeScript + asset import support."
---

# TypeScript + asset import testing plan

## Purpose
Validate that TypeScript sources and `.svg` imports compile, bundle, and run under the Goja CommonJS runtime.

## Environment assumptions
- `bun` is installed and available in PATH.
- Go toolchain is available for `go run`.
- Working directory: `go-go-goja/`.

## Commands

### 1) Install dependencies
```bash
make js-install
```
Expected: bun completes with no errors and `js/bun.lock` is present.

### 2) Typecheck TypeScript (no emit)
```bash
cd js && bun x tsc --noEmit
```
Expected: exit code 0; no missing module errors for `fs`, `exec`, `database`, or `*.svg`.

### 3) Bundle with esbuild
```bash
make js-bundle
```
Expected: `js/dist/bundle.cjs` created and copied to `cmd/bun-demo/assets/bundle.cjs`.

### 4) Verify externals in bundle
```bash
rg -n "require\(\"fs\"\)" cmd/bun-demo/assets/bundle.cjs
```
Expected: at least one `require("fs")` call.

### 5) Run the Goja demo
```bash
make go-run-bun
```
Expected: prints a value like `YYYY-MM-DD:5:<svg-length>` and exits 0.

## Real-world scenarios

### SVG import variations
- Replace `logo.svg` with a larger SVG (multiple paths) and re-run steps 2-5.
- Add another SVG import (`icon.svg`) and confirm `logo.length` changes and no errors occur.

### TypeScript features
- Add an interface and a small generic helper in `main.ts`, then re-run steps 2-5.
- Add an enum and ensure `tsc` passes and the bundle still runs.

### Regression checks
- Remove the `*.svg` declaration and confirm `tsc` fails (expected).
- Switch esbuild target to `browser` (temporarily) and confirm runtime breaks (expected), then revert.

## Exit criteria
- `tsc --noEmit` passes.
- `make js-bundle` succeeds.
- `make go-run-bun` succeeds and prints a value with the SVG length component.
- Externals (`fs`, `exec`) remain unresolved in the bundle (still using `require`).

## Failure modes
- "Could not resolve" errors indicate missing externals or bad loader settings.
- `TypeError: Object has no member 'writeFileSync'` indicates the bundle was built for `browser` target.
- `Cannot find module '*.svg'` indicates missing TS declaration.
