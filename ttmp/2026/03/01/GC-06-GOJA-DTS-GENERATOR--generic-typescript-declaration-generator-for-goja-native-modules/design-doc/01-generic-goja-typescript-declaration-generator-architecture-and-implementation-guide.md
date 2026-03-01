---
Title: Generic Goja TypeScript Declaration Generator Architecture and Implementation Guide
Ticket: GC-06-GOJA-DTS-GENERATOR
Status: active
Topics:
    - goja
    - js-bindings
    - modules
    - tooling
    - architecture
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: workspaces/2026-02-22/add-gepa-optimizer/geppetto/cmd/gen-meta/main.go
      Note: Reference generator flow used for migration comparison
    - Path: workspaces/2026-02-22/add-gepa-optimizer/geppetto/pkg/js/modules/geppetto/generate.go
      Note: go:generate wiring reference for future integration
    - Path: workspaces/2026-02-22/add-gepa-optimizer/geppetto/pkg/spec/geppetto_codegen.yaml
      Note: Reference schema/template outputs in current geppetto setup
    - Path: workspaces/2026-02-22/add-gepa-optimizer/go-go-goja/cmd/bun-demo/js/src/types/goja-modules.d.ts
      Note: Existing manual declaration baseline to migrate
    - Path: workspaces/2026-02-22/add-gepa-optimizer/go-go-goja/engine/factory.go
      Note: FactoryBuilder module registration and runtime creation lifecycle
    - Path: workspaces/2026-02-22/add-gepa-optimizer/go-go-goja/engine/module_specs.go
      Note: ModuleSpec registration model and default registry bridge
    - Path: workspaces/2026-02-22/add-gepa-optimizer/go-go-goja/engine/runtime.go
      Note: Blank imports and module init registration behavior
    - Path: workspaces/2026-02-22/add-gepa-optimizer/go-go-goja/modules/common.go
      Note: Defines NativeModule and registry APIs the generator must integrate with
    - Path: workspaces/2026-02-22/add-gepa-optimizer/go-go-goja/modules/exec/exec.go
      Note: Concrete module export surface used for descriptor examples
    - Path: workspaces/2026-02-22/add-gepa-optimizer/go-go-goja/modules/exports.go
      Note: Current export helper; candidate integration point for future typing metadata
    - Path: workspaces/2026-02-22/add-gepa-optimizer/go-go-goja/modules/fs/fs.go
      Note: Concrete module export surface used for descriptor examples
ExternalSources: []
Summary: Detailed intern-facing architecture and implementation guide for adding a reusable TypeScript declaration generator for go-go-goja native modules.
LastUpdated: 2026-03-01T06:14:51.1173263-05:00
WhatFor: Define a non-breaking, reusable descriptor-driven `.d.ts` generation pipeline in go-go-goja and a phased implementation plan.
WhenToUse: Use when implementing or reviewing Goja native modules that need first-class TypeScript bindings.
---


# Generic Goja TypeScript Declaration Generator Architecture and Implementation Guide

## Executive Summary

go-go-goja currently supports runtime registration and composition of native modules, but has no first-class generic TypeScript declaration generation pipeline. Existing `.d.ts` declarations in this repo are manual and localized (for example `cmd/bun-demo/js/src/types/goja-modules.d.ts`), and geppetto has a separate domain-specific YAML/template code generator that is not reusable as a shared Goja facility.

This document defines a concrete architecture for a new descriptor-driven generator in go-go-goja. The design keeps compatibility with existing module registration (`modules.NativeModule` + `init()` registration), introduces an optional TypeScript descriptor interface for modules that opt in, and adds a CLI generator command that renders deterministic `.d.ts` outputs. The plan is intentionally phased so an intern can ship incremental value with validation and golden tests at each phase.

## Problem Statement and Scope

### Problem

The current module system has strong runtime composition, but weak static developer ergonomics:

1. Runtime module registration is explicit and composable (`modules.NativeModule`, `engine.ModuleSpec`, `FactoryBuilder.WithModules`) but disconnected from static type output.
2. TypeScript declarations are manually maintained in at least one place (`cmd/bun-demo/js/src/types/goja-modules.d.ts`), so API drift is likely.
3. There is no common contract in go-go-goja for module authors to declare JS-facing types once and generate `.d.ts` automatically.
4. geppetto solves a related problem with its own generator (`cmd/gen-meta` + `geppetto_codegen.yaml`) but this is geppetto-specific and not a go-go-goja primitive.

### In Scope

1. New generic descriptor model for JS/TS-exposed module APIs.
2. Non-breaking optional module interface for declaring TypeScript descriptors.
3. CLI generator under go-go-goja to render `.d.ts` files.
4. Validation rules, deterministic rendering, and test strategy.
5. Migration guidance for built-in modules (`fs`, `exec`, `database`) and bun-demo typing file.

### Out of Scope

1. Full automatic Go AST inference of JS API contracts.
2. Rewriting geppetto generator in this ticket.
3. Runtime enforcement of TS contracts inside VM execution.
4. Cross-repo automatic publishing pipeline.

## Current-State Architecture (Evidence)

### Module registration and runtime wiring

1. `modules.NativeModule` requires `Name()`, `Doc()`, and `Loader(...)` and has no typing descriptor surface (`modules/common.go:30-34`).
2. Modules self-register via `modules.Register(...)` (`modules/common.go:88-91`), and all registered modules are enabled through `modules.EnableAll(...)` (`modules/common.go:98-103`).
3. Engine composition accepts module specs and registers them into `require.Registry` during `Build()` (`engine/factory.go:119-124`).
4. Built-in modules are blank imported in `engine/runtime.go` to trigger `init()` registration (`engine/runtime.go:12-19`).

### Existing module authoring shape

1. `modules/fs/fs.go` exports `readFileSync` and `writeFileSync` in `Loader` (`modules/fs/fs.go:31-44`).
2. `modules/exec/exec.go` exports `run` in `Loader` (`modules/exec/exec.go:22-31`).
3. Export helper currently only writes runtime values (`modules/exports.go:7-12`), with no type metadata capture.

### Existing TypeScript declaration state

1. `cmd/bun-demo/js/src/types/goja-modules.d.ts` contains manual declarations for `fs`, `exec`, and `database` (`.../goja-modules.d.ts:1-15`).
2. This file is not tied to module registration and can drift.

### Related external pattern in this workspace (geppetto)

1. geppetto uses a dedicated generator command with sectioned outputs (`geppetto/cmd/gen-meta/main.go:137-187`).
2. Output/template paths are schema-driven (`geppetto/pkg/spec/geppetto_codegen.yaml:5-15`).
3. Generation is wired through `go:generate` in package-local `generate.go` files (`geppetto/pkg/js/modules/geppetto/generate.go:3-4`, `geppetto/pkg/turns/generate.go:3-4`).

Inference from evidence: go-go-goja has a robust runtime module system but no equivalent generic static type-generation path.

## Gap Analysis

1. No single source of truth for module TypeScript contracts.
2. No generic internal DSL for typed declaration authoring.
3. No command to collect module contracts and render module-level or bundle-level `.d.ts` output.
4. No CI check preventing drift between module runtime API and TS declarations.

## Proposed Architecture

### Design Goals

1. Preserve existing runtime API and module authoring patterns.
2. Add typings support as an opt-in path so migration can be incremental.
3. Keep descriptor definitions in Go (typed, refactor-friendly).
4. Keep output deterministic and testable via golden files.
5. Allow escape hatches for advanced TS constructs.

### High-Level Design

Introduce four components:

1. `pkg/tsgen/spec`: typed descriptor DSL (module declarations, functions, params, type refs).
2. `modules/typing.go`: optional interface for modules to expose descriptors.
3. `cmd/gen-dts`: CLI that discovers modules, validates descriptors, renders `.d.ts` output.
4. `pkg/tsgen/render` and `pkg/tsgen/validate`: rendering and schema validation.

### Non-breaking module extension strategy

Keep `modules.NativeModule` unchanged. Add an optional secondary interface:

```go
package modules

import "github.com/go-go-golems/go-go-goja/pkg/tsgen/spec"

type TypeScriptDeclarer interface {
    TypeScriptModule() *spec.Module
}
```

Generator behavior:

1. Module implements `NativeModule` only: included in runtime, skipped for TS unless `--strict` requires descriptor.
2. Module implements both `NativeModule` and `TypeScriptDeclarer`: included in `.d.ts` generation.

Why this is chosen:

1. No breaking change to all existing modules.
2. Easy incremental adoption module-by-module.
3. Avoids forcing placeholder descriptors in early phases.

## Descriptor DSL (Go Types)

Create minimal but expressive spec model.

```go
package spec

type Bundle struct {
    HeaderComment string
    Modules       []*Module
}

type Module struct {
    Name        string        // require("fs") module id
    Description string
    Consts      []ConstGroup
    Types       []TypeDecl
    Functions   []Function
    Namespaces  []Namespace
    RawDTS      []string      // escape hatch
}

type Function struct {
    Name        string
    Description string
    Params      []Param
    Returns     TypeRef
    Throws      []TypeRef
}

type Param struct {
    Name        string
    Type        TypeRef
    Optional    bool
    Variadic    bool
    Description string
}

type TypeRef struct {
    Kind      TypeKind        // String, Number, Boolean, Any, Void, Unknown, Never, ...
    Name      string          // for Named, Literal
    Items     *TypeRef        // for Array
    Union     []TypeRef       // for Union
    Fields    []Field         // for Object
    Signature *FunctionSig    // for Function
}
```

Supporting declarations:

1. `TypeDecl` with `InterfaceDecl` and `TypeAliasDecl` variants.
2. `Namespace` for nested exported groups.
3. `ConstGroup` for readonly const object patterns.

### DSL conventions

1. JS-facing names are lowerCamelCase for functions and options unless ecosystem conventions require otherwise.
2. Module names must match `Name()` from `NativeModule`.
3. Prefer explicit return types over `any`.
4. Use `RawDTS` only for constructs the model cannot yet represent.

## Generator Command Design

### CLI

Proposed command path: `cmd/gen-dts/main.go`.

Proposed flags:

1. `--out <path>` required, output file path.
2. `--module <name1,name2,...>` optional filter.
3. `--strict` fail if selected modules lack descriptors.
4. `--header "..."` optional header comment override.
5. `--format` currently only `dts` (future-proof).
6. `--check` verify generated output matches file content (CI mode).

### Module discovery

Generator can discover modules through `modules.DefaultRegistry` after imports initialize module packages.

Required helper additions:

1. `func (r *Registry) ListModules() []NativeModule` returns a copy.
2. `func ListDefaultModules() []NativeModule` convenience wrapper.

Pseudocode:

```go
mods := modules.ListDefaultModules()
mods = filterByFlag(mods, moduleFilter)

for _, m := range mods {
    td, ok := m.(modules.TypeScriptDeclarer)
    if !ok {
        if strict {
            return errorf("module %s has no TypeScript descriptor", m.Name())
        }
        continue
    }

    decl := td.TypeScriptModule()
    if err := validate.Module(decl); err != nil {
        return errorf("module %s invalid descriptor: %w", m.Name(), err)
    }

    bundle.Modules = append(bundle.Modules, decl)
}

sortByModuleName(bundle.Modules)
text := render.Bundle(bundle)
writeOrCheck(outPath, text, checkMode)
```

### Rendering

Renderer rules:

1. Stable ordering:
   - modules by name,
   - functions by name,
   - type declarations by name.
2. Deterministic whitespace and quote style.
3. Output pattern:

```ts
declare module "fs" {
  export function readFileSync(path: string): string;
}
```

4. Optional grouped output for namespace exports.
5. Top-level banner: `// Code generated by go-go-goja/cmd/gen-dts. DO NOT EDIT.`

## File and Package Layout

Proposed additions in go-go-goja:

1. `pkg/tsgen/spec/types.go`
2. `pkg/tsgen/spec/helpers.go`
3. `pkg/tsgen/validate/validate.go`
4. `pkg/tsgen/render/dts_renderer.go`
5. `pkg/tsgen/render/dts_renderer_test.go`
6. `modules/typing.go`
7. `cmd/gen-dts/main.go`
8. `cmd/gen-dts/main_test.go`
9. `cmd/bun-demo/js/src/types/goja-modules.d.ts` (generated target)
10. Optional: `modules/fs/typescript.go`, `modules/exec/typescript.go`, `modules/database/typescript.go`

## Implementation Phases

### Phase 0: Scaffolding and Contracts

1. Add `modules.TypeScriptDeclarer` interface.
2. Add module listing helpers in registry.
3. Add `pkg/tsgen/spec` with core types.

Acceptance criteria:

1. Builds clean with no behavior changes.
2. No module code must change yet.

### Phase 1: Validator + Renderer MVP

1. Implement validation for required fields:
   - module name non-empty,
   - unique function names per module,
   - param names non-empty,
   - valid `TypeRef` trees.
2. Implement `.d.ts` renderer for:
   - primitive types,
   - arrays,
   - unions,
   - named refs,
   - module-level functions.

Acceptance criteria:

1. Golden tests pass.
2. Invalid descriptors return clear error messages with path context.

### Phase 2: Generator CLI + CI Check Mode

1. Implement `cmd/gen-dts` with `--out`, `--module`, `--strict`, `--check`.
2. Add dry-run/check mode for CI drift detection.
3. Add docs in README and `pkg/doc/`.

Acceptance criteria:

1. `go run ./cmd/gen-dts --out /tmp/goja.d.ts` generates file from registered modules.
2. `--check` exits non-zero on drift.

### Phase 3: Migrate Built-In Modules

1. Add descriptors for `fs`, `exec`, and `database`.
2. Generate `cmd/bun-demo/js/src/types/goja-modules.d.ts` from command.
3. Mark file as generated and remove manual edits.

Acceptance criteria:

1. Generated file includes current exports from these modules.
2. Module tests + generator tests pass.

### Phase 4: Hardening and Escape Hatches

1. Add `RawDTS` append support.
2. Add namespace/type alias/interface support.
3. Add stricter validation toggles and lint checks.

Acceptance criteria:

1. Complex modules can express needed types without changing renderer internals every time.
2. Drift detection integrated in CI path (or make target).

## Example Descriptor Snippets

### fs module descriptor

```go
func (m) TypeScriptModule() *spec.Module {
    return &spec.Module{
        Name: "fs",
        Functions: []spec.Function{
            {
                Name: "readFileSync",
                Params: []spec.Param{{Name: "path", Type: spec.String()}},
                Returns: spec.String(),
            },
            {
                Name: "writeFileSync",
                Params: []spec.Param{
                    {Name: "path", Type: spec.String()},
                    {Name: "data", Type: spec.String()},
                },
                Returns: spec.Void(),
            },
        },
    }
}
```

### generated output

```ts
declare module "fs" {
  export function readFileSync(path: string): string;
  export function writeFileSync(path: string, data: string): void;
}
```

## Testing and Validation Strategy

### Unit tests

1. `pkg/tsgen/validate` table tests for malformed descriptors.
2. `pkg/tsgen/render` golden tests for deterministic output.
3. `cmd/gen-dts` CLI tests:
   - strict mode pass/fail,
   - module filter behavior,
   - check mode behavior.

### Integration tests

1. Build a runtime with `engine.DefaultRegistryModules()` and ensure generated modules match expected module names.
2. Compare generated output with committed declaration file in bun-demo.

### CI integration

Suggested target:

```make
check-dts:
	go run ./cmd/gen-dts --out cmd/bun-demo/js/src/types/goja-modules.d.ts --check --strict
```

Run in CI before merge.

## Migration Guidance for geppetto Consumers

This ticket does not migrate geppetto, but the new go-go-goja generator should be designed so geppetto can adopt the pattern incrementally.

Recommended migration approach:

1. Keep geppetto `cmd/gen-meta` for geppetto-specific const/key schema generation.
2. For Goja module API declarations, add adapter descriptors conforming to go-go-goja `spec.Module`.
3. Optionally add a small bridge generator in geppetto that imports go-go-goja descriptors and emits `declare module "geppetto"` blocks.

Why hybrid first:

1. geppetto schema covers more than module API typing.
2. Reduces migration risk and keeps existing generated artifacts stable.

## Risks and Mitigations

1. Descriptor drift from actual `Loader` exports.
   - Mitigation: add module-level tests asserting exported key names and descriptor key names match.
2. DSL scope creep.
   - Mitigation: keep core types small; use `RawDTS` for edge cases.
3. Breaking module authors with required interface changes.
   - Mitigation: optional `TypeScriptDeclarer` interface only.
4. Registry initialization ambiguity for generator command.
   - Mitigation: explicit blank imports in `cmd/gen-dts` and clear docs.

## Alternatives Considered

### Alternative A: Keep YAML/template approach like geppetto

Pros:

1. Familiar if already using schema files.

Cons:

1. Maintains a separate text schema away from module code.
2. Lower refactor safety for Go-first module authors.

Decision: reject as primary approach for go-go-goja; retain as possible adapter layer only.

### Alternative B: AST/comment-based auto inference

Pros:

1. Potentially less manual descriptor work.

Cons:

1. Hard to infer dynamic Goja patterns and JS semantics correctly.
2. More brittle and significantly larger implementation.

Decision: reject for initial delivery.

### Alternative C: Extend `modules.SetExport` to capture runtime type metadata

Pros:

1. Co-locates export and typing intent.

Cons:

1. Requires wrapping every export and building reflective function analysis.
2. Harder to represent rich TS constructs.

Decision: not for initial milestone; revisit later if desired.

## Intern Handoff Checklist

1. Create `pkg/tsgen/spec` types and helpers.
2. Add `modules.TypeScriptDeclarer` and registry listing helpers.
3. Implement validator + renderer with golden tests.
4. Implement `cmd/gen-dts` command and CLI tests.
5. Add descriptors for `fs`, `exec`, `database`.
6. Convert bun-demo `.d.ts` to generated file.
7. Add `--check` workflow and Makefile target.
8. Update docs with authoring instructions and examples.
9. Run full test suite and capture outputs in diary.

## Open Questions

1. Should generator emit one bundled `.d.ts` file or one file per module by default?
2. Should strict mode require descriptors for all registered modules, or only selected modules?
3. Is `RawDTS` sufficient as an escape hatch, or do we need an explicit AST node for mapped/conditional types now?
4. Should generated declarations include `Doc()` text as JSDoc automatically?

## References

### go-go-goja codebase

1. `go-go-goja/modules/common.go` (NativeModule interface and registry lifecycle)
2. `go-go-goja/modules/exports.go` (export helper)
3. `go-go-goja/modules/fs/fs.go` (concrete module export pattern)
4. `go-go-goja/modules/exec/exec.go` (concrete module export pattern)
5. `go-go-goja/engine/module_specs.go` (module spec registration model)
6. `go-go-goja/engine/factory.go` (factory Build/NewRuntime lifecycle)
7. `go-go-goja/engine/runtime.go` (blank-import module init strategy)
8. `go-go-goja/cmd/bun-demo/js/src/types/goja-modules.d.ts` (manual declaration baseline)

### related generator pattern

1. `geppetto/cmd/gen-meta/main.go`
2. `geppetto/pkg/spec/geppetto_codegen.yaml`
3. `geppetto/pkg/js/modules/geppetto/generate.go`
4. `geppetto/pkg/turns/generate.go`
