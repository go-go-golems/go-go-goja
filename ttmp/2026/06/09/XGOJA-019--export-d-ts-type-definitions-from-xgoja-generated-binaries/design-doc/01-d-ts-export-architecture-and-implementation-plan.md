---
Title: d.ts export architecture and implementation plan
Ticket: XGOJA-019
Status: active
Topics:
    - xgoja
    - typescript
    - modules
    - tooling
    - developer-experience
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/modules/typing.go
      Note: TypeScriptDeclarer interface definition — the contract modules implement to provide d.ts descriptors
    - Path: /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/tsgen/spec/types.go
      Note: spec.Module, spec.Function, spec.TypeRef — the descriptor data model for d.ts generation
    - Path: /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/tsgen/render/dts_renderer.go
      Note: Renders spec.Bundle into a deterministic .d.ts string
    - Path: /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/tsgen/validate/validate.go
      Note: Validates spec.Module and spec.Bundle before rendering
    - Path: /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/cmd/gen-dts/main.go
      Note: Standalone gen-dts tool — iterates ListDefaultModules(), filters, renders
    - Path: /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/cmd/bun-demo/generate.go
      Note: go:generate directive that uses gen-dts for bun-demo
    - Path: /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/cmd/bun-demo/js/src/types/goja-modules.d.ts
      Note: Example generated d.ts output for bun-demo module set
    - Path: /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/xgoja/providerapi/module.go
      Note: providerapi.Module — current provider module struct (no DTS descriptor field)
    - Path: /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/xgoja/providerapi/provider_registry.go
      Note: ProviderRegistry — resolves modules by package+name at build and runtime
    - Path: /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/xgoja/providers/core/core.go
      Note: Core provider — wraps NativeModules via nativeModuleEntry(), loses TypeScriptDeclarer
    - Path: /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/xgoja/providers/host/host.go
      Note: Host provider — wraps fs/exec/database modules, loses TypeScriptDeclarer
    - Path: /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/modules/fs/fs.go
      Note: Example TypeScriptDeclarer implementation with full RawDTS + Functions
    - Path: /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/modules/express/typescript.go
      Note: Express module TypeScriptDeclarer with complex RawDTS (interfaces, types)
    - Path: /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/cmd/xgoja/internal/buildspec/build_spec.go
      Note: BuildSpec schema — the xgoja.yaml data model with Modules, Packages, etc.
    - Path: /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/cmd/xgoja/internal/generate/templates/main.go.tmpl
      Note: Generated main.go template — where embed directives and d.ts wiring would go
ExternalSources: []
Summary: Design for making TypeScript type definitions available to JS developers working with xgoja-generated binaries, both at build time (xgoja gen-dts subcommand) and at runtime (embedded d.ts served via CLI flag or HTTP endpoint).
LastUpdated: 2026-06-09T18:05:00-04:00
WhatFor: Enable JS developers to get accurate TypeScript type definitions for the native modules available in any xgoja-generated binary, without manually running gen-dts against the source repository.
WhenToUse: When implementing d.ts export from xgoja, adding the gen-dts subcommand, or wiring runtime type-definition exposure.
---

# d.ts Export Architecture and Implementation Plan

## Executive Summary

go-go-goja already has a complete d.ts generation pipeline: modules implement `TypeScriptDeclarer`, `pkg/tsgen` provides a spec/render/validate stack, and `cmd/gen-dts` produces `.d.ts` files. However, this pipeline is disconnected from the xgoja build system. The provider layer (`pkg/xgoja/providerapi`) wraps modules without preserving their TypeScript descriptors, xgoja has no `gen-dts` subcommand, and generated binaries cannot emit or serve their type definitions at runtime.

This document describes three complementary changes that close these gaps and make type definitions available to JS developers working against any xgoja-generated binary.

## Problem Statement

When someone uses `xgoja build -f xgoja.yaml` to produce a custom binary (e.g., a club-meetup-site server), JavaScript developers writing code against that binary's runtime have no way to discover the TypeScript types of the available native modules. The types exist in Go source code, but are inaccessible without:

1. Cloning the go-go-goja repository
2. Understanding which modules the xgoja.yaml selects
3. Running `cmd/gen-dts` with the correct module list

This is a developer-experience gap that makes it hard to write type-safe JavaScript against xgoja runtimes.

## Current-State Architecture

### What exists

The d.ts generation pipeline is complete at the module level:

```
modules/typing.go         → TypeScriptDeclarer interface
modules/{fs,events,...}   → 10+ modules implement TypeScriptDeclarer
pkg/tsgen/spec/types.go   → spec.Module, spec.Function, spec.TypeRef data model
pkg/tsgen/render/         → deterministic d.ts string rendering
pkg/tsgen/validate/       → descriptor validation before rendering
cmd/gen-dts/              → standalone tool: ListDefaultModules() → filter → render → file
cmd/bun-demo/generate.go  → go:generate proof-of-concept using gen-dts
```

### What's broken/disconnected

```
pkg/xgoja/providerapi/module.go
  → providerapi.Module has no DTS descriptor field
  → nativeModuleEntry() in core.go/host.go wraps NativeModule but discards TypeScriptDeclarer

cmd/xgoja/
  → No gen-dts subcommand
  → No way to map xgoja.yaml module selections back to TypeScriptDeclarer instances

Generated binary (cmd/xgoja/internal/generate/templates/main.go.tmpl)
  → No embedded d.ts
  → No CLI flag or HTTP endpoint to expose types
```

## Gap Analysis

### Gap A: `providerapi.Module` loses the TypeScript descriptor

**Location:** `pkg/xgoja/providerapi/module.go`, `pkg/xgoja/providers/core/core.go:46`, `pkg/xgoja/providers/host/host.go`

When `core.Register()` and `host.Register()` wrap `NativeModule` instances into `providerapi.Module` structs, they extract only `Name()`, `Doc()`, and `Loader`. The `TypeScriptDeclarer` interface assertion is never checked, and the `*spec.Module` descriptor is never passed through.

```go
// core.go:46 — current state
func nativeModuleEntry(mod modules.NativeModule) providerapi.Module {
    return providerapi.Module{
        Name:        mod.Name(),
        DefaultAs:   mod.Name(),
        Description: mod.Doc(),
        NewModuleFactory: func(providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
            return mod.Loader, nil
        },
    }
}
```

**Fix:** Add a `DTSDescriptor *spec.Module` field to `providerapi.Module`, and check the `TypeScriptDeclarer` interface in `nativeModuleEntry()`.

### Gap B: No `xgoja gen-dts` subcommand

**Location:** `cmd/xgoja/` (new file needed)

`cmd/gen-dts` is a standalone binary that uses `modules.ListDefaultModules()` — the global registry. It cannot know which modules an xgoja.yaml selects because that information lives in the `ProviderRegistry`, not the global module registry.

The xgoja build pipeline already resolves modules: `buildspec.LoadFile()` parses the yaml, the `ProviderRegistry` maps package+module to `providerapi.Module`. Once Gap A is fixed, each resolved `providerapi.Module` will carry its `DTSDescriptor`.

**Fix:** Add a new `genDtsCommand` in `cmd/xgoja/cmd_gen_dts.go` that:
1. Loads the build spec
2. Resolves each selected module from the provider registry
3. Collects non-nil `DTSDescriptor` entries into a `spec.Bundle`
4. Renders and writes the `.d.ts` file

### Gap C: No runtime d.ts exposure

**Location:** Generated binary template (`cmd/xgoja/internal/generate/templates/main.go.tmpl`)

Even after Gap A and B are fixed, the generated binary has no mechanism to expose types at runtime. A JS developer working against a deployed binary still can't get the types.

**Options for runtime exposure:**

| Option | Pros | Cons |
|--------|------|------|
| **CLI flag** `--emit-types` | Simple, no HTTP dependency | Requires running the binary separately |
| **HTTP endpoint** `GET /xgoja/types.d.ts` | Available during dev, IDE can fetch | Only works if binary runs a web server |
| **Embedded file** via `go:embed` in generated code | Always available, can be extracted | Adds ~5-10KB to binary size |
| **go:generate** directive in generated code | Build-time only, zero runtime cost | Requires Go toolchain at generation site |

**Recommended approach:** Start with the `xgoja gen-dts` subcommand (Gap B) as the primary surface. Add an `--embed-dts` flag to `xgoja build` that embeds the generated d.ts as a string constant and exposes it via `--emit-types` CLI flag on the generated binary. The HTTP endpoint is a future enhancement that naturally follows once the binary can emit types.

## Proposed Solution

### Phase 1: Wire DTS through the provider layer

**Files to change:**

1. **`pkg/xgoja/providerapi/module.go`** — Add `DTSDescriptor` field:

```go
type Module struct {
    Name             string
    DefaultAs        string
    Description      string
    ConfigSchema     json.RawMessage
    DTSDescriptor    *spec.Module             // NEW
    NewModuleFactory func(ModuleSetupContext) (require.ModuleLoader, error)
}
```

2. **`pkg/xgoja/providers/core/core.go`** — Extract descriptor in `nativeModuleEntry()`:

```go
func nativeModuleEntry(mod modules.NativeModule) providerapi.Module {
    entry := providerapi.Module{
        Name:        mod.Name(),
        DefaultAs:   mod.Name(),
        Description: mod.Doc(),
        NewModuleFactory: func(providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
            return mod.Loader, nil
        },
    }
    if td, ok := mod.(modules.TypeScriptDeclarer); ok {
        entry.DTSDescriptor = td.TypeScriptModule()
    }
    return entry
}
```

3. **`pkg/xgoja/providers/host/host.go`** — Same pattern for `fsModule()`, `execModule()`, `databaseModule()`. These create modules from `fsmod.New()`, `dbm.New()` etc., so they need to call `TypeScriptModule()` on the underlying module type.

### Phase 2: Add `xgoja gen-dts` subcommand

**New file:** `cmd/xgoja/cmd_gen_dts.go`

```
xgoja gen-dts -f xgoja.yaml --out types.d.ts [--strict] [--check]
```

Implementation:
1. Load build spec via `buildspec.LoadFile()`
2. Initialize `ProviderRegistry`, call all `Register()` functions from provider packages
3. For each `ModuleInstanceSpec` in the build spec, resolve `providerapi.Module` from registry
4. Collect non-nil `DTSDescriptor` entries into `spec.Bundle`
5. Render via `render.Bundle()` and write to `--out`
6. Support `--check` mode for CI (same as `cmd/gen-dts`)

### Phase 3: Embed d.ts in generated binaries

**Files to change:**

1. **`cmd/xgoja/internal/buildspec/build_spec.go`** — Add `emitTypes` field to `BuildSpec` (optional, defaults to false).

2. **`cmd/xgoja/internal/generate/templates/main.go.tmpl`** — When `emitTypes` is true, add:
   - `const embeddedDTS = "..."` string constant with the rendered d.ts
   - A `--emit-types` flag on the generated binary that writes the d.ts to stdout or a file

3. **`cmd/xgoja/internal/generate/generate.go`** — During `WriteAll()`, if `emitTypes` is enabled, run the same d.ts rendering pipeline and include the output in the generated source.

### Phase 4 (future): HTTP endpoint

When the generated binary runs an HTTP server (via the express provider or xgoja's HTTP host support), automatically serve the embedded d.ts at a well-known path like `/xgoja/types.d.ts`. This requires coordination with the HTTP host provider and is a natural follow-up to Phase 3.

## Decision Records

### DR-1: DTSDescriptor on providerapi.Module vs separate registry

**Context:** The TypeScript descriptor needs to flow from provider packages to the xgoja CLI and generated binary.

**Options:**
- A: Add `DTSDescriptor *spec.Module` to `providerapi.Module`
- B: Create a separate `TypeScriptRegistry` that modules opt into

**Decision:** Option A.

**Rationale:** The descriptor is an inherent property of the module. A separate registry would require every provider to register in two places and would be error-prone. The `DTSDescriptor` field is optional (nil for modules without types) and does not affect the runtime module loading path.

**Consequences:** `pkg/xgoja/providerapi` takes a dependency on `pkg/tsgen/spec`. This is acceptable because `spec` is a pure data model with no side effects.

### DR-2: Runtime exposure strategy

**Context:** JS developers need to get type definitions from the running binary.

**Options:**
- A: CLI flag `--emit-types` only
- B: HTTP endpoint only
- C: Both, with CLI as primary

**Decision:** Option C, phased — CLI first (Phase 3), HTTP later (Phase 4).

**Rationale:** The CLI flag works universally (no HTTP dependency). The HTTP endpoint is more convenient for web-based workflows but requires the binary to actually run a server. Phasing lets us ship value immediately.

## Implementation Plan

### Phase 1: Wire DTS through provider layer

| Step | File | Change |
|------|------|--------|
| 1.1 | `pkg/xgoja/providerapi/module.go` | Add `DTSDescriptor *spec.Module` field |
| 1.2 | `pkg/xgoja/providers/core/core.go` | Extract descriptor via TypeScriptDeclarer assertion |
| 1.3 | `pkg/xgoja/providers/host/host.go` | Extract descriptors for fs/exec/database modules |
| 1.4 | Tests | Verify provider packages carry descriptors through |

### Phase 2: Add `xgoja gen-dts` subcommand

| Step | File | Change |
|------|------|--------|
| 2.1 | `cmd/xgoja/cmd_gen_dts.go` | New command, reuses tsgen/render |
| 2.2 | `cmd/xgoja/root.go` | Wire into command tree |
| 2.3 | Tests | Integration test: xgoja.yaml → .d.ts output |

### Phase 3: Embed d.ts in generated binaries

| Step | File | Change |
|------|------|--------|
| 3.1 | `cmd/xgoja/internal/buildspec/build_spec.go` | Add `emitTypes` config |
| 3.2 | `cmd/xgoja/internal/generate/generate.go` | Render d.ts during WriteAll |
| 3.3 | `cmd/xgoja/internal/generate/templates/main.go.tmpl` | Embed d.ts constant + --emit-types flag |
| 3.4 | Tests | Build a binary, verify --emit-types output matches gen-dts |

### Phase 4: HTTP endpoint (future)

| Step | File | Change |
|------|------|--------|
| 4.1 | `pkg/xgoja/providers/http/http.go` | Serve embedded d.ts at /xgoja/types.d.ts |

## Testing and Validation

1. **Unit tests:** Verify `nativeModuleEntry()` preserves `DTSDescriptor` for modules that implement `TypeScriptDeclarer`
2. **Integration test:** `xgoja gen-dts -f testdata/xgoja.yaml --out /tmp/test.d.ts` and compare against golden file
3. **Build test:** `xgoja build -f testdata/xgoja.yaml --output /tmp/test-binary && /tmp/test-binary --emit-types` and compare output against `xgoja gen-dts` output
4. **CI check:** `xgoja gen-dts -f xgoja.yaml --out types.d.ts --check` to catch descriptor drift

## Risks and Open Questions

1. **Third-party provider modules** may not implement `TypeScriptDeclarer`. The `--strict` flag should make this an error; without it, silently skip.
2. **Module aliases** (`as:` in xgoja.yaml) — should the d.ts use the alias or the original module name? Probably the alias, since that's what `require()` uses.
3. **Config-dependent types** — some modules might expose different APIs based on config (e.g., fs with embedded vs host backend). For now, emit the full descriptor regardless of config. This is a refinement for the future.
4. **Binary size** — the embedded d.ts string is typically 2-10KB. Negligible.

## References

### Prior tickets
- `GC-06-GOJA-DTS-GENERATOR` (ttmp/2026/03/01/) — Original d.ts generator design and implementation
- `GOJA-15-GEN-DTS-PLUGINS` (ttmp/2026/03/20/) — Extending gen-dts for plugin modules

### Key source files
- `modules/typing.go` — TypeScriptDeclarer interface
- `pkg/tsgen/spec/types.go` — Descriptor data model
- `pkg/tsgen/render/dts_renderer.go` — d.ts rendering
- `cmd/gen-dts/main.go` — Standalone gen-dts tool
- `cmd/bun-demo/generate.go` — go:generate usage example
- `pkg/xgoja/providerapi/module.go` — Provider module struct (needs DTSDescriptor)
- `pkg/xgoja/providers/core/core.go` — Core provider wrapping
- `pkg/xgoja/providers/host/host.go` — Host provider wrapping

---

# Design Review and Corrected Redesign

## Why this review exists

The initial design above got the broad user-facing goal right, but it did not fully respect the most important constraint in xgoja: **the xgoja CLI is a precompiled tool, while provider packages named in `xgoja.yaml` are imported only by generated code**. That distinction changes the design materially. A normal `xgoja gen-dts -f xgoja.yaml` command cannot simply call `Register()` on arbitrary provider packages listed in the build spec unless those provider packages are already linked into the `xgoja` binary.

This section reviews what was good, what was weak, what was confusing, what should be dropped, and what the corrected design should be.

## What was good in the original design

### 1. It correctly identified that the existing d.ts pipeline is real and reusable

The original analysis correctly found the established pipeline:

- `modules.TypeScriptDeclarer` in `modules/typing.go`
- `pkg/tsgen/spec` as the declaration descriptor model
- `pkg/tsgen/render` as the deterministic `.d.ts` renderer
- `pkg/tsgen/validate` as the descriptor validation layer
- `cmd/gen-dts` as a working command-line proof that descriptors can become `.d.ts` output
- `cmd/bun-demo/generate.go` as a real `go:generate` consumer

That part is solid. It means the problem is not “invent a TypeScript declaration generator.” The problem is “connect an existing declaration generator to xgoja’s provider/generation model.”

That is the right framing.

### 2. It correctly identified metadata loss at the provider boundary

The original design correctly noticed that `pkg/xgoja/providerapi.Module` currently carries runtime and documentation metadata, but no TypeScript declaration metadata:

```go
type Module struct {
    Name             string
    DefaultAs        string
    Description      string
    ConfigSchema     json.RawMessage
    NewModuleFactory func(ModuleSetupContext) (require.ModuleLoader, error)
}
```

It also correctly identified that `pkg/xgoja/providers/core/core.go` wraps `modules.NativeModule` values in `providerapi.Module` via `nativeModuleEntry()`, but discards the optional `modules.TypeScriptDeclarer` interface. That is a real gap.

### 3. It understood that aliases matter

The original design flagged the `as:` alias question in `xgoja.yaml`. That is important. TypeScript declarations should describe what JavaScript authors actually import with `require()`, not the implementation module’s canonical name.

If an xgoja spec says:

```yaml
modules:
  - package: go-go-goja-host
    name: fs
    as: fs:assets
```

then the generated declaration probably needs:

```ts
declare module "fs:assets" { ... }
```

not only:

```ts
declare module "fs" { ... }
```

This cannot be an afterthought. It belongs in the descriptor normalization layer.

### 4. It separated build-time generation and runtime exposure

The original design separated:

1. generate a `.d.ts` file from a spec, and
2. expose the `.d.ts` from the generated binary.

That separation is useful. It lets us support editor workflows where the declaration file is checked into a JS project, and also runtime workflows where a generated binary can print or serve the type definitions it was built with.

### 5. It proposed tests at the right seams

The original test ideas were broadly good:

- provider-level tests for descriptor propagation
- build-spec-to-d.ts integration tests
- generated-binary tests for runtime emission
- check mode for CI drift detection

Those are the right classes of tests. The details need adjustment, but the testing seams are correct.

## What was not so good

### 1. The biggest mistake: assuming `xgoja` can import arbitrary providers from `xgoja.yaml`

The original design said:

> Initialize `ProviderRegistry`, call all `Register()` functions from provider packages.

That is not how the compiled `xgoja` CLI works for third-party providers.

In `cmd/xgoja/internal/generate/templates/main.go.tmpl`, provider packages are imported by the **generated program**:

```go
{{- range .ProviderImports }}
    {{ .Alias }} "{{ .Import }}"
{{- end }}
```

Then the generated program calls:

```go
registry := providerapi.NewProviderRegistry()
{{- range .ProviderImports }}
    must({{ .Alias }}.{{ .Register }}(registry))
{{- end }}
```

The compiled `xgoja` CLI itself does not import arbitrary provider packages from user specs. It only writes Go source that imports them. This is crucial. A precompiled Go binary cannot dynamically import and call Go functions by import path and function name. It would need to generate, compile, and execute a helper program, or require providers to emit static metadata, or rely on some external manifest format.

So the earlier `xgoja gen-dts -f xgoja.yaml` design is only valid for provider packages already linked into `xgoja`, such as first-party providers if xgoja explicitly imports them. It is not valid for arbitrary third-party providers.

This is the main correction.

### 2. It conflated module descriptors with provider descriptors

`modules.TypeScriptDeclarer` belongs to the older/native module layer:

```go
type TypeScriptDeclarer interface {
    TypeScriptModule() *spec.Module
}
```

But xgoja provider modules are not necessarily `modules.NativeModule`. They are `providerapi.Module` values whose runtime loader is produced by:

```go
NewModuleFactory func(ModuleSetupContext) (require.ModuleLoader, error)
```

A provider can define a module without ever constructing a `modules.NativeModule` value. That means “assert `TypeScriptDeclarer` on the native module” is a good migration path for first-party wrappers, but it is not a general provider API.

The provider-level API needs its own explicit declaration metadata contract. It can reuse `pkg/tsgen/spec.Module`, but it should not require providers to expose a `modules.NativeModule`.

### 3. It did not distinguish three different outputs

The original design used “gen-dts” loosely for three different products:

1. a checked-out source tree output: `types/goja-modules.d.ts`
2. generated Go source that embeds a declaration string
3. a generated binary runtime command/flag that prints the embedded declaration

Those are related but distinct. They have different lifecycle constraints:

- Source-tree generation can happen before build and can write files.
- Generated Go source can embed constants but must be deterministic and gofmt-safe.
- Runtime emission must be represented as an application command/flag without accidentally interfering with cobra/glazed command parsing.

A sharper design should name these as separate surfaces.

### 4. It hand-waved generated binary CLI integration

The original design said “add a `--emit-types` flag on the generated binary.” That sounds easy, but the generated binary already delegates root command construction to `app.NewRootCommand(...)` or to a target adapter/cobra root.

There are at least three generated-target shapes:

- default xgoja root command (`target.kind` empty / generated root)
- adapter target (`target.kind: adapter`)
- cobra target (`target.kind: cobra`)
- package/source/template generation modes that do not produce a binary directly

A top-level `--emit-types` flag has to work across these shapes, or the design must explicitly scope it to the generated default root command only. The earlier design did not specify this.

The safer first implementation is likely a **generated command** named something like `types` or `xgoja-types`, mounted by `app.AttachDefaultCommands`, rather than a global flag. A command avoids early flag parsing problems and fits the cobra command tree.

### 5. It proposed an HTTP endpoint too early

The HTTP endpoint idea is useful eventually, but it does not belong in the first implementation. It crosses into HTTP host/provider behavior, route mounting, conflict handling, security expectations, and server lifecycle.

The first implementation should focus on deterministic generation and a CLI extraction surface. HTTP serving can be a separate ticket once the d.ts payload exists in the runtime bundle.

### 6. It did not specify how descriptors are cloned or renamed

If a descriptor is reused for aliases, the implementation must not mutate the original descriptor in-place. For example, if one provider module descriptor is named `fs` and the xgoja spec selects it twice as `fs:host` and `fs:assets`, then generating declarations requires two module blocks with different names but otherwise similar content.

A robust implementation needs a helper like:

```go
func DescriptorForAlias(desc *spec.Module, alias string) (*spec.Module, error)
```

which deep-copies the descriptor and replaces only `Name`.

The original design implied this behavior but did not say how to avoid accidental shared mutation.

### 7. It initially did not acknowledge the nested doc root mistake

The ticket was originally created under:

```text
go-go-goja/go-go-goja/ttmp/...
```

because `docmgr --root go-go-goja/ttmp` was run from inside the `go-go-goja` repository directory. The intended root was:

```text
go-go-goja/ttmp/...
```

from the parent workspace, or simply `ttmp` from inside the repository. This was later corrected by moving the ticket into the repository's real `ttmp/` root. The lesson remains relevant: before creating ticket docs, confirm whether the current working directory is the parent workspace or the repository root.

## What was confusing

### 1. “Using xgoja” was underspecified

There are multiple meanings:

- using the `xgoja` CLI to generate type files
- using xgoja generated source/package mode to expose types
- using the generated xgoja binary to print/serve types
- using the xgoja provider system to carry type descriptors

The first design mixed these together. The improved design should give each one a named surface:

1. **xgoja source-time declaration generation**
2. **generated runtime declaration embedding**
3. **generated binary declaration extraction**
4. **provider declaration metadata contract**

### 2. “Provider registry” sounded available everywhere

The original text made the `ProviderRegistry` sound like something the `xgoja` CLI can always populate. In reality there are two provider registries:

- the registry inside the generated program, populated by generated imports and function calls
- any registry inside the xgoja CLI, which can only contain packages already imported into xgoja itself

This distinction should have been explicit.

### 3. “Strict” semantics were not fully specified

Strict mode can mean several things:

- fail if a selected module has no descriptor
- fail only for selected modules from packages that advertise descriptor support
- fail if descriptor validation fails
- fail if an alias cannot be rendered
- fail if a provider package cannot be loaded by the sidecar generator

The original design used `--strict` but did not define these cases.

## What could be done better

### 1. Use a sidecar generator for arbitrary third-party providers

Because the xgoja CLI cannot dynamically import provider packages, the correct general solution is to generate a temporary Go program that imports the provider packages from `xgoja.yaml`, registers them, resolves selected modules, renders declarations, and writes them out.

This mirrors how `xgoja build` already works:

1. read `xgoja.yaml`
2. generate temporary Go module/source
3. import selected provider packages
4. run Go tooling in that generated workspace

For d.ts generation, the sidecar can be much smaller than a full binary build:

```go
package main

import (
    "fmt"
    "os"

    "github.com/go-go-golems/go-go-goja/pkg/xgoja/dtsgen"
    "github.com/go-go-golems/go-goja/pkg/xgoja/providerapi"
    p0 "provider/import/path"
)

func main() {
    registry := providerapi.NewProviderRegistry()
    must(p0.Register(registry))

    out, err := dtsgen.RenderFromRuntimeSpec(registry, embeddedRuntimeSpecJSON, dtsgen.Options{Strict: true})
    must(err)
    fmt.Print(out)
}
```

Then `xgoja gen-dts` can generate and run this sidecar with `go run`, using the same module requirement/replacement logic as `xgoja build`.

### 2. Put declaration rendering in a reusable package, not inside `cmd/xgoja`

The code that turns an xgoja runtime spec plus provider registry into a d.ts bundle should live in a library package, not only in a CLI command.

A good package shape would be:

```text
pkg/xgoja/dtsgen/
  descriptors.go      // collect selected module descriptors from provider registry + runtime spec
  render.go           // render bundle using pkg/tsgen/render
  options.go          // strict, includeHeader, includeMissingComments, alias behavior
  descriptors_test.go
```

This lets all surfaces reuse one implementation:

- `xgoja gen-dts`
- generated binary `types` command
- generated package/source mode exported API
- tests
- future HTTP endpoint

### 3. Make provider type metadata explicit and provider-native

Instead of only relying on `modules.TypeScriptDeclarer`, add a provider-native field or interface:

```go
type Module struct {
    Name             string
    DefaultAs        string
    Description      string
    ConfigSchema     json.RawMessage
    TypeScript       *spec.Module
    NewModuleFactory func(ModuleSetupContext) (require.ModuleLoader, error)
}
```

Name choice matters. `DTSDescriptor` is precise but implementation-flavored. `TypeScript` or `TypeScriptModule` might be clearer to provider authors. A field like `TypeScript *spec.Module` says: this is the module’s TypeScript declaration descriptor.

For first-party `modules.NativeModule` wrappers, add a helper:

```go
func TypeScriptDescriptorFromNativeModule(mod modules.NativeModule) *spec.Module {
    declarer, ok := mod.(modules.TypeScriptDeclarer)
    if !ok {
        return nil
    }
    return declarer.TypeScriptModule()
}
```

That keeps native-module compatibility without making native modules the only declaration path.

### 4. Treat aliasing as a first-class transformation

The d.ts generator should operate on selected module instances, not raw module definitions.

Input should effectively be:

```go
type SelectedModuleDescriptor struct {
    Package string
    Name    string
    Alias   string
    Module  providerapi.Module
}
```

Rendering should use the alias if present, otherwise `DefaultAs`, otherwise `Name`.

Recommended rule:

1. `ModuleInstanceSpec.As` if non-empty
2. `providerapi.Module.DefaultAs` if non-empty
3. `ModuleInstanceSpec.Name`

Then deep-copy the descriptor and set `descriptor.Name = requireName`.

### 5. Prefer a generated `types` command over a global `--emit-types` flag

Instead of injecting a top-level flag, add a generated command under the xgoja command tree:

```text
<binary> types                 # print d.ts to stdout
<binary> types --out types.d.ts
<binary> types --check types.d.ts
<binary> types --format dts    # future-proof, default dts
```

This is easier to integrate with cobra and glazed. It also matches the conceptual model: type definitions are a command, not a runtime execution mode.

If a flag is still desired later, it can be a convenience wrapper.

### 6. Support generated package/source mode directly

For `target.kind: package` and `target.kind: source`, the generated package should expose APIs such as:

```go
func TypeScriptDeclarations() string
func WriteTypeScriptDeclarations(w io.Writer) error
```

That is more useful than only generating a binary command. It lets embedding applications decide whether to:

- expose an HTTP endpoint
- write a file during their own build
- add their own cobra command
- feed declarations into an IDE/dev server

## What could be dropped

### 1. Drop HTTP endpoint from this ticket’s implementation scope

Keep it as a future follow-up. Do not implement it in phases 1-3. It is not needed to solve the immediate type-definition availability problem.

### 2. Drop `emitTypes` as a build-spec field for the first pass

A build spec field like:

```yaml
emitTypes: true
```

is probably the wrong first surface. It bakes one output policy into the spec. Instead:

- generated binaries should always have enough embedded metadata to print their declarations if requested, or
- the generator should have an explicit CLI flag such as `xgoja build --include-types-command`

Even better: if the declaration payload is small and deterministic, always include the `types` command in generated xgoja roots. Avoid an extra configuration knob unless binary size or compatibility proves it necessary.

### 3. Drop the idea that `cmd/gen-dts` should be the central implementation

`cmd/gen-dts` should remain a compatibility/simple-source tool for the global module registry. The xgoja-aware implementation should live in `pkg/xgoja/dtsgen` and be used by new xgoja surfaces.

`cmd/gen-dts` can later be refactored to call common rendering helpers, but it should not become the place where xgoja build-spec logic lives.

## What the original designers should have known

### 1. Go cannot import arbitrary packages at runtime by string import path

This is the key Go constraint. A compiled Go binary can only call code linked into it at compile time. `xgoja.yaml` import paths are used to generate Go source, not to dynamically load Go packages into the existing xgoja process.

That one fact invalidates the naive `xgoja gen-dts` implementation.

### 2. xgoja is a code generator, not only a runtime configurator

The xgoja build spec is not merely runtime config. It describes code to generate:

- Go module name
- imports
- provider registration calls
- target root construction
- embedded assets/help/jsverbs

Any feature that needs third-party provider code must either be generated into the output program or executed through a generated sidecar.

### 3. Provider modules are not native modules

Some provider modules wrap `modules.NativeModule`, but the provider abstraction is broader. The TypeScript metadata contract needs to belong to the provider abstraction, not only to native modules.

### 4. Generated package mode matters

xgoja does not only build binaries. It also generates reusable runtime packages/source fragments/templates. A design that only talks about generated binaries misses important users.

### 5. Type declarations describe the selected runtime, not the implementation catalog

A `.d.ts` file for an xgoja runtime should describe the exact modules selected by the build spec, including aliases. It should not merely dump every descriptor known to a package or global registry.

## What they should know next time

### 1. Start from execution boundaries

Before designing a feature in a code generator, ask:

- Which code is running inside the generator binary?
- Which code is only present in generated output?
- Which data crosses the boundary as JSON/source/text?
- Which imports exist in which compiled binary?

This prevents impossible designs.

### 2. Draw the dataflow before naming phases

For this problem, the corrected dataflow is:

```text
xgoja.yaml
  → buildspec.BuildSpec
  → generated helper or generated binary source
  → provider package imports become real Go imports
  → providerapi.ProviderRegistry is populated
  → selected modules are resolved
  → providerapi.Module.TypeScript descriptors are collected
  → descriptors are alias-normalized and validated
  → spec.Bundle is rendered into .d.ts
  → output file / generated command / package API
```

The first design jumped from `xgoja.yaml` to `ProviderRegistry` too quickly.

### 3. Prefer reusable library seams over command-specific logic

If a feature is needed by a CLI command, generated binary, and generated package mode, put the core logic in a package. Commands should parse flags and call library code.

### 4. Decide first implementation scope ruthlessly

The first implementation should not include HTTP serving. The smallest useful loop is:

1. provider module carries TypeScript descriptor
2. sidecar `xgoja gen-dts` emits a d.ts file for a spec
3. generated root exposes a `types` command
4. generated package exposes `TypeScriptDeclarations()`

That is enough to satisfy the user goal.

## Corrected Redesign

## Corrected goal

Make TypeScript declarations available for the exact runtime described by an xgoja build spec, including provider packages, selected modules, aliases, and generated package/binary use cases.

The solution must work for:

1. first-party providers compiled into go-go-goja,
2. third-party providers imported by generated xgoja code,
3. source/package/template generation modes,
4. generated binaries used by JS authors.

## Corrected architecture

### Layer 1: Provider declaration metadata

Add declaration metadata to the provider module model:

```go
// pkg/xgoja/providerapi/module.go

type Module struct {
    Name             string
    DefaultAs        string
    Description      string
    ConfigSchema     json.RawMessage
    TypeScript       *spec.Module
    NewModuleFactory func(ModuleSetupContext) (require.ModuleLoader, error)
}
```

Use `TypeScript` rather than `DTSDescriptor` unless the project strongly prefers the latter. The declaration model is TypeScript-specific but output-format-neutral enough to be rendered to `.d.ts`.

For first-party native module wrappers, add a helper:

```go
func nativeModuleTypeScript(mod modules.NativeModule) *spec.Module {
    declarer, ok := mod.(modules.TypeScriptDeclarer)
    if !ok {
        return nil
    }
    return declarer.TypeScriptModule()
}
```

Then:

```go
func nativeModuleEntry(mod modules.NativeModule) providerapi.Module {
    return providerapi.Module{
        Name:             mod.Name(),
        DefaultAs:        mod.Name(),
        Description:      mod.Doc(),
        TypeScript:       nativeModuleTypeScript(mod),
        NewModuleFactory: func(providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
            return mod.Loader, nil
        },
    }
}
```

For manually-authored provider modules (`host`, `http`, future third-party providers), authors can set `TypeScript` directly without implementing `modules.NativeModule`.

### Layer 2: xgoja d.ts library package

Create:

```text
pkg/xgoja/dtsgen/
```

Recommended API:

```go
type Options struct {
    Strict bool
    Header string
}

type MissingDescriptor struct {
    Package string
    Name    string
    Alias   string
}

type Result struct {
    DTS     string
    Missing []MissingDescriptor
}

func RenderRuntimeSpec(registry *providerapi.ProviderRegistry, runtimeSpec *app.RuntimeSpec, opts Options) (*Result, error)
func BundleRuntimeSpec(registry *providerapi.ProviderRegistry, runtimeSpec *app.RuntimeSpec, opts Options) (*spec.Bundle, []MissingDescriptor, error)
```

Responsibilities:

1. iterate selected modules from `app.RuntimeSpec` (or the generated runtime spec equivalent)
2. resolve each provider module through `ProviderRegistry.ResolveModule(packageID, moduleName)`
3. obtain `providerapi.Module.TypeScript`
4. deep-copy descriptor
5. rename descriptor to selected require name (`as`, `DefaultAs`, `name`)
6. validate each descriptor and the bundle
7. render via `pkg/tsgen/render.Bundle`

Strict behavior:

- if selected module is missing from provider registry: error
- if selected module has nil `TypeScript` and `Strict` is true: error with package/name/alias
- if selected module has nil `TypeScript` and `Strict` is false: record in `Result.Missing`, skip it
- descriptor validation failures always error

### Layer 3: Sidecar-backed `xgoja gen-dts`

Add a command:

```text
xgoja gen-dts -f xgoja.yaml --out types.d.ts [--strict] [--check] [--work-dir DIR] [--keep-work] [--xgoja-replace PATH]
```

But implement it as a generated sidecar, not as direct provider loading.

Flow:

1. Load and validate `xgoja.yaml`.
2. Generate a temporary Go module similar to `xgoja build`.
3. Generate a small `main.go` that imports the provider packages from the build spec and registers them.
4. Embed or write the runtime spec JSON.
5. Call `pkg/xgoja/dtsgen.RenderRuntimeSpec(...)`.
6. Print to stdout or write to a sidecar output file.
7. The parent `xgoja gen-dts` command copies/compares output for `--out`/`--check`.

This works for third-party provider packages because `go run` compiles a helper program with exactly those imports.

Implementation reuse:

- Reuse `RenderGoMod()` for module requirements/replaces.
- Add a new generator function like `RenderDTSGenMain(buildSpec)` or a template under `cmd/xgoja/internal/generate/templates/dtsgen_main.go.tmpl`.
- Use the same alias generation logic as normal generated main/package templates.

### Layer 4: Generated source/package APIs

For generated package/source mode, expose declaration APIs directly:

```go
func TypeScriptDeclarations() (string, error)
func WriteTypeScriptDeclarations(w io.Writer) error
```

or, if generation has already rendered the declaration string:

```go
const EmbeddedTypeScriptDeclarations = `...`

func TypeScriptDeclarations() string {
    return EmbeddedTypeScriptDeclarations
}
```

There are two implementation options:

#### Option A: render declarations at generated-code runtime

The generated package registers providers, decodes spec JSON, then calls `dtsgen.RenderRuntimeSpec()`.

Pros:
- No large generated string unless requested
- Always uses same code path
- Easier to test

Cons:
- Requires descriptors and renderer linked into generated package
- Type declaration output computed at runtime

#### Option B: render declarations during code generation and embed as constant

Pros:
- Generated binary/package can print instantly
- Output is frozen at generation time
- No dtsgen traversal needed at runtime

Cons:
- For arbitrary providers, xgoja generator still needs the sidecar path to render the string
- Adds generated source size

Recommended first implementation: **Option A** for generated package/source mode. It avoids sidecar complexity inside `xgoja generate` and uses the fact that generated packages already import provider packages and can register providers at runtime.

### Layer 5: Generated binary command

Add a generated command, not a global flag:

```text
<binary> types
<binary> types --out types.d.ts
<binary> types --check types.d.ts
```

Where to attach:

- For default generated xgoja roots: attach in `app.NewRootCommand` or the generated main after root construction.
- For generated package mode: expose `NewTypesCommand()` or let embedding apps call `TypeScriptDeclarations()`.
- For `target.kind: cobra`: the host can attach default xgoja commands already; include `types` among those default commands if the host uses `AttachDefaultCommands`.
- For `target.kind: adapter`: require the adapter host integration to decide whether to expose the command, or expose it on the host object.

The command should call the same `dtsgen.RenderRuntimeSpec()` path using the generated program’s real provider registry and embedded runtime spec.

### Layer 6: Optional source-tree output

`xgoja gen-dts` writes files for JS/TS projects:

```bash
xgoja gen-dts -f xgoja.yaml --out js/types/xgoja-modules.d.ts --strict
```

This is the primary developer workflow for editor support.

Generated binary command is the fallback/distribution workflow:

```bash
my-generated-binary types --out js/types/xgoja-modules.d.ts
```

## Corrected implementation phases

### Phase 1: Provider metadata + dtsgen library

1. Add `TypeScript *spec.Module` (or `DTSDescriptor *spec.Module`) to `providerapi.Module`.
2. Wire first-party providers:
   - `pkg/xgoja/providers/core/core.go`
   - `pkg/xgoja/providers/host/host.go`
   - `pkg/xgoja/providers/http/http.go` if it provides express/http modules with descriptors
3. Create `pkg/xgoja/dtsgen`.
4. Implement descriptor collection, alias normalization, validation, and rendering.
5. Unit tests:
   - nil descriptors skipped/non-strict
   - nil descriptors fail/strict
   - alias renames descriptor without mutating original
   - duplicate aliases fail clearly

### Phase 2: Generated package/binary exposure

1. Add dtsgen-backed functions to generated package template:
   - `TypeScriptDeclarations() (string, error)`
   - optionally `WriteTypeScriptDeclarations(io.Writer) error`
2. Add a generated/default `types` cobra command.
3. Attach `types` command to default xgoja roots and generated package helper APIs.
4. Tests:
   - generated package can return declarations
   - generated binary `types` command prints expected module declarations
   - alias in xgoja.yaml appears in declaration module name

### Phase 3: Sidecar-backed `xgoja gen-dts`

1. Add `cmd/xgoja/cmd_gen_dts.go`.
2. Generate a temporary d.ts sidecar Go program using provider imports from the build spec.
3. Reuse go.mod rendering/replacement logic.
4. Run `go run .` in the sidecar workspace.
5. Implement `--out`, `--check`, `--strict`, `--work-dir`, `--keep-work`, `--xgoja-replace`.
6. Tests:
   - first-party provider spec produces declarations
   - `--check` fails on mismatch
   - `--keep-work` leaves inspectable sidecar source

### Phase 4: Future HTTP serving (separate ticket)

If still desired after phases 1-3, add HTTP serving as an explicit integration point. Do not couple it to core d.ts generation.

## Corrected acceptance criteria

A good implementation is done when:

1. Provider modules can carry TypeScript descriptors without needing to be `modules.NativeModule`.
2. First-party xgoja providers preserve descriptors for wrapped native modules.
3. `pkg/xgoja/dtsgen` can render the exact selected runtime module set, including aliases.
4. Generated packages can return their own declarations programmatically.
5. Generated binaries expose a `types` command.
6. `xgoja gen-dts` works for arbitrary third-party provider packages by using a generated sidecar, not by pretending to dynamically import Go packages.
7. `--strict` and `--check` semantics are documented and tested.
8. HTTP serving is not required for the first implementation.

## Replacement summary

The original design should be treated as a useful discovery document, not as an implementation blueprint. Keep these parts:

- Existing d.ts pipeline inventory
- Provider metadata gap
- Alias concern
- Need for source-time and runtime surfaces
- Test categories

Replace these parts:

- direct `xgoja gen-dts` provider loading → **sidecar-generated provider loading**
- global `--emit-types` flag → **generated `types` command + package API**
- build-spec `emitTypes` field → **default generated capability or explicit generator flag**
- HTTP endpoint in implementation phases → **future separate ticket**

The corrected design respects xgoja’s actual architecture: provider imports become real only in generated Go code, so any declaration system for arbitrary providers must either run inside generated code or compile a generated helper.