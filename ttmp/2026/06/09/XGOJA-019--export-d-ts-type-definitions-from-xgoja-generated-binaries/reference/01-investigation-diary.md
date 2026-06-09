---
Title: Investigation diary
Ticket: XGOJA-019
Status: active
Topics:
    - xgoja
    - typescript
    - modules
    - tooling
    - developer-experience
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/modules/typing.go
      Note: TypeScriptDeclarer interface
    - Path: /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/tsgen/spec/types.go
      Note: spec.Module data model
    - Path: /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/tsgen/render/dts_renderer.go
      Note: d.ts renderer
    - Path: /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/cmd/gen-dts/main.go
      Note: Standalone gen-dts tool
    - Path: /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/xgoja/providerapi/module.go
      Note: providerapi.Module — gap A location
    - Path: /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/xgoja/providers/core/core.go
      Note: Core provider — discards TypeScriptDeclarer
    - Path: /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/xgoja/providers/host/host.go
      Note: Host provider — discards TypeScriptDeclarer
ExternalSources: []
Summary: Investigation of the existing d.ts generation pipeline and identification of three architectural gaps that prevent xgoja-generated binaries from exposing TypeScript type definitions.
LastUpdated: 2026-06-09
WhatFor: Record the investigation findings and architectural analysis for XGOJA-019.
WhenToUse: Read this before starting implementation work on XGOJA-019 to understand what was discovered and why the proposed design looks the way it does.
---

# Investigation Diary

## Goal

Investigate how d.ts type definitions can be exported from xgoja-generated binaries so that JS developers writing code against those binaries have accurate TypeScript type information.

## Step 1: Architecture Investigation

The user asked to analyze the xgoja setup and determine how d.ts files could be exposed both from xgoja itself and from generated binaries. I investigated the full d.ts generation pipeline and identified three architectural gaps.

### Prompt Context

**User prompt (verbatim):** "Create a new docmgr ticket in go-go-goja/ using `docmgr --root go-go-goja/ttmp ...` about exporting d.ts files using xgoja. Analyze xgoja setup and how we can expose the d.ts files when not just with xgoja but also in the generated binary so that people writing js using the resulting system have type definitions. That might actually be already possible. Investigate first before creating the ticket, and let me know what you find before moving on."

**Assistant interpretation:** Investigate the full d.ts generation pipeline in go-go-goja, assess what already works for xgoja, identify gaps, and create a docmgr ticket with the findings.

**Inferred user intent:** Understand whether d.ts export from xgoja binaries is already possible or what work remains, to plan an implementation that gives JS developers type-safe access to native modules.

### What I did

- Traced the full d.ts pipeline: `TypeScriptDeclarer` interface → `spec.Module` descriptors → `pkg/tsgen/render` → `cmd/gen-dts`
- Found 10+ modules that implement `TypeScriptDeclarer` (fs, node:fs, exec, database, events, crypto, path, os, yaml, express, uidsl)
- Examined `cmd/bun-demo/generate.go` as a working proof-of-concept using `go:generate`
- Traced the xgoja provider layer: `providerapi.Module` → `providers/core/core.go` → `providers/host/host.go`
- Found that `nativeModuleEntry()` wraps `NativeModule` but never checks for `TypeScriptDeclarer`
- Examined the generated binary template (`main.go.tmpl`) — no d.ts embedding
- Reviewed prior tickets GC-06 and GOJA-15 for historical context

### Why

The user needs to understand the gap between what exists (a standalone gen-dts tool using the global registry) and what's needed (d.ts export driven by xgoja.yaml module selections, available from the generated binary).

### What worked

- The existing `pkg/tsgen` stack is clean and well-designed — spec/render/validate separation means it can be reused directly
- `cmd/gen-dts/main.go` already does 90% of the work; the missing 10% is wiring it through the provider registry
- The `spec.Bundle` abstraction naturally fits the xgoja use case (collect descriptors from selected modules)

### What didn't work

- The provider layer loses the TypeScript descriptor during wrapping. `nativeModuleEntry()` only extracts Name/Doc/Loader.
- xgoja has no subcommand for d.ts generation — can't map xgoja.yaml → types
- No runtime exposure mechanism exists in generated binaries

### What I learned

- The `bun-demo` `go:generate` pattern already proves the pipeline works end-to-end
- The provider architecture was designed before `TypeScriptDeclarer` was added to xgoja scope, so the gap is architectural timing, not a design flaw
- `providerapi.Module` is the right place to carry the descriptor — it's the single point of truth for module metadata

### What was tricky to build

- Tracing the full module instantiation path through xgoja was non-trivial: `xgoja.yaml` → `buildspec.LoadFile()` → `ProviderRegistry` → `providerapi.Module.NewModuleFactory` → `require.ModuleLoader`. The descriptor needs to be available *before* `NewModuleFactory` is called (at generation time, not runtime).

### What warrants a second pair of eyes

- The decision to add `DTSDescriptor *spec.Module` to `providerapi.Module` — this creates a dependency from `providerapi` on `pkg/tsgen/spec`. The spec package is a pure data model (no side effects), so this should be safe, but worth confirming.
- Module aliases (`as:` in xgoja.yaml) — the d.ts should probably use the alias as the module name since that's what `require()` sees, but this changes the rendering semantics.

### What should be done in the future

- HTTP endpoint for d.ts serving (Phase 4 in the design doc)
- Config-dependent type narrowing (e.g., fs module exposes different types when in embedded mode)
- Support for third-party provider packages that implement their own `TypeScriptDeclarer`-equivalent

### Code review instructions

- Start with `modules/typing.go` (the interface) and `pkg/xgoja/providerapi/module.go` (where the field needs to go)
- Then `pkg/xgoja/providers/core/core.go:46` and `pkg/xgoja/providers/host/host.go` (where the descriptor is lost)
- Compare `cmd/gen-dts/main.go` (existing standalone) with the proposed `cmd/xgoja/cmd_gen_dts.go` (new xgoja subcommand)
- Verify `pkg/tsgen/spec/types.go` is indeed a pure data model with no non-deterministic imports

### Technical details

**Modules implementing TypeScriptDeclarer (10+):**
```
modules/fs/fs.go
modules/events/events.go
modules/time/time.go
modules/path/path.go
modules/express/typescript.go
modules/exec/exec.go
modules/os/os.go
modules/yaml/yaml.go
modules/crypto/crypto.go
modules/uidsl/typescript.go
modules/database/database.go
```

**Current gen-dts usage pattern:**
```go
// cmd/bun-demo/generate.go
//go:generate go run ../gen-dts --out ./js/src/types/goja-modules.d.ts --module fs,node:fs,exec,database,events,node:events,crypto,node:crypto,path,node:path,os,node:os,yaml --strict
```

**Example d.ts output** (from `cmd/bun-demo/js/src/types/goja-modules.d.ts`):
```typescript
// Code generated by go-go-goja/cmd/gen-dts. DO NOT EDIT.

declare module "fs" {
  export function readFile(path: string, encoding?: string | {  }): Promise<string | Buffer>;
  export function writeFile(path: string, data: string | Buffer | Uint8Array | DataView, encoding?: string | {  }): Promise<void>;
  // ... 20+ more functions
}
```
