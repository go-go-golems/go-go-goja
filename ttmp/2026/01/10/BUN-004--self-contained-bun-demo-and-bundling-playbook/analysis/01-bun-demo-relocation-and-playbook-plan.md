---
Title: Bun demo relocation and playbook plan
Ticket: BUN-004
Status: active
Topics:
    - bun
    - bundling
    - docs
    - typescript
    - goja
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-10T21:16:40.5061245-05:00
WhatFor: "Track the plan to relocate the bun demo assets and publish a comprehensive bundling playbook."
WhenToUse: "Use before moving the JS workspace or authoring the bundled Goja playbook."
---

# Bun demo relocation and playbook plan

## Overview
This analysis captures the work required to make the bun demo self-contained under `cmd/bun-demo` and to publish a detailed playbook describing how to bundle TypeScript + assets for Goja. It keeps the file layout, build steps, and documentation scope aligned before we move files.

## Goals
- Relocate the JS workspace to `cmd/bun-demo/js` so the demo is self-contained.
- Add a dedicated `cmd/bun-demo/Makefile` for install, bundle, and run workflows.
- Keep the embedded bundle in `cmd/bun-demo/assets/bundle.cjs`.
- Produce a comprehensive playbook in `pkg/doc` following the Glazed documentation style guide.

## Constraints
- Preserve CommonJS output for Goja's `require` loader.
- Keep native module names (`fs`, `exec`, `database`) external to the bundle.
- Avoid breaking existing Go demo entrypoint (`cmd/bun-demo/main.go`).
- Keep the work reproducible with `make` targets.

## Proposed layout
The demo bundle will live entirely inside `cmd/bun-demo`, with the assets embedded alongside the Go entrypoint.

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

## Build flow changes
The root `Makefile` no longer owns bun targets. The new `cmd/bun-demo/Makefile` will expose:
- `js-install`, `js-typecheck`, `js-bundle`, `js-clean`
- `go-run-bun` (runs the demo after bundling)

## Documentation deliverable
The playbook in `pkg/doc` should:
- Explain the Goja + CommonJS integration model.
- Provide a complete TS + asset bundling pipeline with Bun + esbuild.
- Include examples for `package.json`, `tsconfig.json`, and the demo entrypoint.
- Document how to embed the bundle and invoke `require` in Go.
- Provide a troubleshooting section and validation checklist.

## Risks and validation
Moving the JS workspace requires path updates in build scripts, `.gitignore`, and the demo Makefile. Validation should include running `make -C cmd/bun-demo go-run-bun` and confirming the SVG metrics output still appears.
