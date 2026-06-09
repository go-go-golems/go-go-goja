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

## Step 2: Phase 1 Provider Metadata and dtsgen Library

This step implemented the corrected design's first layer: provider modules can now carry TypeScript declaration descriptors, and a reusable `pkg/xgoja/dtsgen` package can render declarations for the exact module set selected by an `app.RuntimeSpec`.

The implementation deliberately keeps declaration generation out of `cmd/xgoja` for now. The reusable library is the stable seam needed by generated package APIs, generated binary `types` commands, and the future sidecar-backed `xgoja gen-dts` command.

### Prompt Context

**User prompt (verbatim):** "continue"

**Assistant interpretation:** Continue from the corrected XGOJA-019 design by starting implementation, keeping the diary updated, and committing at sensible boundaries.

**Inferred user intent:** Move from design into code, beginning with the corrected Phase 1 plan rather than the earlier flawed plan.

**Commit (code):** pending at time of diary update — "xgoja: add provider TypeScript declaration metadata"

### What I did

- Added `TypeScript *spec.Module` to `providerapi.Module`.
- Wired first-party descriptors through:
  - `pkg/xgoja/providers/core/core.go`
  - `pkg/xgoja/providers/host/host.go`
  - `pkg/xgoja/providers/http/http.go`
- Added `pkg/xgoja/dtsgen` with:
  - `RenderRuntimeSpec`
  - `BundleRuntimeSpec`
  - strict/non-strict missing descriptor behavior
  - alias normalization (`as`, then provider default alias, then module name)
  - deep-copy descriptor handling so alias rewriting does not mutate provider-owned descriptors
  - duplicate require-alias detection
- Added tests for dtsgen behavior and provider descriptor propagation.
- Ran targeted validation:
  - `go test ./pkg/xgoja/providerapi ./pkg/xgoja/providers/core ./pkg/xgoja/providers/host ./pkg/xgoja/providers/http ./pkg/xgoja/dtsgen -count=1`

### Why

The corrected design depends on provider metadata being available at the provider layer, not just on legacy `modules.NativeModule` implementations. Without this, generated binaries and sidecar generators cannot ask the selected provider modules for their declarations.

The dtsgen package also prevents command-specific duplication. It can be reused by generated package mode, generated binary commands, and future `xgoja gen-dts` sidecars.

### What worked

- The provider metadata field was a small, low-friction addition because `pkg/tsgen/spec` is a pure descriptor package.
- Core provider modules that wrap `modules.NativeModule` can preserve existing `TypeScriptDeclarer` descriptors through a small helper.
- Host provider modules can attach descriptors even when their runtime behavior is guarded/customized (`fs`, `exec`, `database`).
- HTTP provider can attach the existing express TypeScript descriptor via `express.NewRegistrar(nil).TypeScriptModule()`.
- dtsgen can render aliased declarations without mutating the original provider descriptor.

### What didn't work

The first targeted test run failed:

```text
--- FAIL: TestRegisterCoreProvider (0.00s)
    core_test.go:27: expected core module "timer" to carry TypeScript descriptor
FAIL
FAIL	github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/core	0.019s
```

The cause was an incorrect test assumption. `timer` is registered by the core provider, but it does not currently implement `modules.TypeScriptDeclarer`. I corrected the test to assert descriptors only for the core modules that actually provide them, leaving `timer` as an intentionally untyped module for non-strict rendering.

### What I learned

- Not every core provider module is typed yet. Strict mode is therefore useful immediately: it lets users decide whether missing declarations are acceptable.
- The provider metadata design supports progressive typing: modules can be selected and used even if their `TypeScript` field is nil.
- Alias handling must happen at selected-module instance time, not provider-module definition time.

### What was tricky to build

The main tricky part was avoiding accidental descriptor mutation. Provider descriptors are shared metadata. If runtime spec aliases `fs` as `fs:assets`, the rendered descriptor must be renamed to `fs:assets`, but the provider's original `fs` descriptor must remain named `fs` for other selections and future calls. I solved this by deep-copying `spec.Module`, including functions, parameters, type references, unions, arrays, and object fields before rewriting `Name`.

The second tricky part was host `exec`: the host provider implements guarded exec behavior directly rather than using the legacy unguarded `modules/exec` loader. The descriptor is still the same shape, so the provider imports the exec module for registration side effects and reads its descriptor from the default module registry. This should be reviewed because it couples descriptor discovery to a side-effect import.

### What warrants a second pair of eyes

- The new dependency from `providerapi` to `pkg/tsgen/spec`.
- The host provider's `exec` descriptor source: importing `modules/exec` for descriptor registration is pragmatic but may be less explicit than manually constructing the descriptor or exporting a constructor from the exec module.
- Whether `timer` should get a TypeScript descriptor now, or remain a test case for missing descriptor behavior.
- The duplicate-alias error policy: it is strict and early, which seems correct because duplicate `declare module` blocks for the same require name would be ambiguous.

### What should be done in the future

- Add a TypeScript descriptor for `timer` if it is part of the stable JS-facing core API.
- Consider making descriptor helper functions available for first-party modules whose runtime provider implementation is custom but API shape is shared.
- Move on to Phase 2: generated package/binary declaration exposure using `pkg/xgoja/dtsgen`.

### Code review instructions

- Start with `pkg/xgoja/providerapi/module.go` to see the metadata contract.
- Then review `pkg/xgoja/dtsgen/dtsgen.go`, especially `BundleRuntimeSpec`, `requireName`, and `cloneTypeRef`.
- Review provider wiring in:
  - `pkg/xgoja/providers/core/core.go`
  - `pkg/xgoja/providers/host/host.go`
  - `pkg/xgoja/providers/http/http.go`
- Validate with:
  - `go test ./pkg/xgoja/providerapi ./pkg/xgoja/providers/core ./pkg/xgoja/providers/host ./pkg/xgoja/providers/http ./pkg/xgoja/dtsgen -count=1`

### Technical details

The successful validation command was:

```bash
go test ./pkg/xgoja/providerapi ./pkg/xgoja/providers/core ./pkg/xgoja/providers/host ./pkg/xgoja/providers/http ./pkg/xgoja/dtsgen -count=1
```

The successful output was:

```text
ok  	github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi	0.019s
ok  	github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/core	0.015s
ok  	github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/host	0.078s
ok  	github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/http	0.349s
ok  	github.com/go-go-golems/go-go-goja/pkg/xgoja/dtsgen	0.036s
```
