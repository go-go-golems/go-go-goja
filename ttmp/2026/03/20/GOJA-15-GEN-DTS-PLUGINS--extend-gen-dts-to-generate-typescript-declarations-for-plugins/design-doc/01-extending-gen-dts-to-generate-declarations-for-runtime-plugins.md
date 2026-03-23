---
Title: Plugin-author-facing TypeScript declaration generation for go-go-goja plugins
Ticket: GOJA-15-GEN-DTS-PLUGINS
Status: active
Topics:
    - goja
    - typescript
    - plugins
    - docs
    - analysis
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/gen-dts/main.go
      Note: Current built-in module generator to reuse for shared tsgen concepts, not the primary plugin-author workflow
    - Path: pkg/hashiplugin/sdk/export.go
      Note: |-
        Function and method option APIs that would carry declaration metadata
        Primary export and method option APIs for plugin-author signatures
    - Path: pkg/hashiplugin/sdk/module.go
      Note: |-
        Plugin module definition and the best place to attach author-owned typing metadata
        Primary module definition API for plugin-author metadata
    - Path: pkg/tsgen/spec/types.go
      Note: |-
        Existing declaration model that plugin generation should reuse where practical
        Declaration model reused by plugin-author generation
    - Path: plugins/examples/greeter/main.go
      Note: Example of the current inline-authoring pattern that should be refactored into a reusable module factory
ExternalSources: []
Summary: Corrected-scope design for making it easy for plugin writers to generate `.d.ts` files from source-owned SDK metadata rather than from discovered plugin binaries.
LastUpdated: 2026-03-20T09:06:20.252610305-04:00
WhatFor: Explain how plugin writers should declare and generate `.d.ts` files directly from plugin source and SDK metadata.
WhenToUse: Read this before adding TypeScript generation support to the plugin SDK or designing plugin-author workflows.
---


# Plugin-author-facing TypeScript declaration generation for go-go-goja plugins

## Executive Summary

The corrected question is not "can the host discover installed plugins and emit declarations for them?" The corrected question is "can a plugin author easily generate a `.d.ts` file for their own plugin from source?"

Yes. That is the right target, and it is a better design target than discovery-driven generation.

The recommended approach is:

1. Keep built-in module generation and plugin-author generation conceptually separate.
2. Move plugin declaration metadata into the plugin SDK authoring path, next to `sdk.Function(...)`, `sdk.Object(...)`, and `sdk.Method(...)`.
3. Give plugin authors a direct generation API, ideally a small Go library such as `pkg/hashiplugin/sdk/dtsgen`, so they can render a `.d.ts` file from `*sdk.Module` without installing or discovering a plugin binary.
4. Recommend a small source layout change for plugin authors: the module definition should live in an importable package or factory function, while `main.go` should only call `sdk.Serve(...)`.
5. Optionally mirror the same metadata into the runtime manifest later, but that should be a secondary benefit, not the primary workflow.

The main reason for this design is ownership. A plugin author should declare the JavaScript surface once, in source, and generate the `.d.ts` from that same source. They should not need to install the plugin somewhere on disk and ask a host-side tool to rediscover it.

## Corrected Problem Statement

The user clarified that the goal is plugin-author ergonomics:

1. a plugin writer should be able to define their plugin API once,
2. generate a `.d.ts` file easily during development or release,
3. keep the declaration close to the plugin source of truth.

This is different from the earlier host-side framing, which asked whether the existing [`cmd/gen-dts`](/home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/cmd/gen-dts/main.go) command should inspect discovered plugin binaries.

That earlier framing is possible as a future adapter, but it is not the best primary workflow because:

1. it makes declaration generation depend on binary discovery,
2. it puts the source of truth on the host side instead of the author side,
3. it is awkward for plugin authors in separate repositories,
4. it is harder to integrate into `go generate` and CI.

So this document intentionally optimizes for source-owned plugin generation.

## Current Architecture

### Part 1: How built-in modules generate declarations today

Built-in modules in `go-go-goja` already follow a good author-owned pattern.

Observed behavior:

1. a built-in module implements `modules.NativeModule` in [modules/common.go](/home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/modules/common.go#L29),
2. it can optionally implement `modules.TypeScriptDeclarer` in [modules/typing.go](/home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/modules/typing.go#L5),
3. it returns a static `*spec.Module`,
4. [`cmd/gen-dts`](/home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/cmd/gen-dts/main.go#L37) validates and renders those descriptors through `pkg/tsgen`.

Important property:

The declaration source of truth lives in code, next to the module implementation. The generator only renders it.

That is the model plugin authors should also get.

### Part 2: How plugin authors define a plugin today

Plugin authors currently build plugins through the SDK.

Observed behavior:

1. [`pkg/hashiplugin/sdk/module.go`](/home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/pkg/hashiplugin/sdk/module.go#L34) builds `*sdk.Module` values.
2. [`pkg/hashiplugin/sdk/export.go`](/home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/pkg/hashiplugin/sdk/export.go#L81) lets authors define function exports, object exports, and object methods.
3. Handler arguments are runtime-decoded through [`sdk.Call`](/home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/pkg/hashiplugin/sdk/call.go#L5).
4. Example plugins such as [`plugins/examples/greeter/main.go`](/home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/plugins/examples/greeter/main.go#L12) define their module inline inside `main()`.

This authoring pattern is good for runtime execution but not ideal for declaration generation, because:

1. the module definition is often buried in `main.go`,
2. there is no explicit typing surface beyond docs and method names,
3. there is no direct generator API for `*sdk.Module`.

### Part 3: What the existing `tsgen` model can already do

The `pkg/tsgen` packages are already a strong foundation.

Observed behavior:

1. [`pkg/tsgen/spec/types.go`](/home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/pkg/tsgen/spec/types.go#L20) defines bundles, modules, functions, params, and types.
2. [`pkg/tsgen/render/dts_renderer.go`](/home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/pkg/tsgen/render/dts_renderer.go#L13) renders deterministic `.d.ts` output.
3. [`pkg/tsgen/validate/validate.go`](/home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/pkg/tsgen/validate/validate.go#L10) validates descriptors before rendering.

Inference:

We do not need a second renderer for plugins. We need a plugin-author-facing way to produce truthful `tsgen` descriptors.

### Part 4: Why discovered-plugin generation is the wrong primary workflow

The earlier analysis inspected the host-side plugin discovery path in [`pkg/hashiplugin/host`](/home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/pkg/hashiplugin/host). That path is useful context, but it should not be the center of this design.

Reasons:

1. It starts from a built binary, not from source.
2. It assumes the plugin is installed somewhere discoverable.
3. It is awkward for third-party plugin authors outside this repository.
4. It couples declaration generation to runtime host concerns.
5. The current manifest shape still lacks accurate signature data, so host-side discovery would not solve the main typing problem anyway.

The host path can remain a future consumer of the same metadata. It should not define the author experience.

## Design Goals

The correct design should satisfy these goals:

1. A plugin author can generate a `.d.ts` file from source without installing the plugin.
2. The source of truth lives next to the plugin author’s export definitions.
3. The workflow is compatible with `go generate` and CI.
4. The declaration model reuses `pkg/tsgen` instead of inventing a parallel renderer.
5. The authoring API is explicit. It should not infer types from handler bodies.
6. The same metadata can optionally flow into runtime manifests later.

Non-goals:

1. making `cmd/gen-dts` discover plugin binaries by default,
2. type inference from `call.String(0)` / `call.Map(0)` style runtime code,
3. solving all host-side plugin introspection in the same change.

## Recommended Design

### Decision

Add plugin-author-facing TypeScript metadata to the SDK, then add a small generation helper that renders a single plugin module from `*sdk.Module`.

This means the core new abstraction should live near `pkg/hashiplugin/sdk`, not near `pkg/hashiplugin/host`.

### Recommended author workflow

The workflow should look like this:

```text
plugin source package
  -> builds *sdk.Module with explicit TS metadata
  -> local generation helper renders .d.ts
  -> plugin main.go serves the same module at runtime
```

Concrete example:

```text
plugins/myplugin/module/module.go
  func NewModule() *sdk.Module

plugins/myplugin/cmd/myplugin/main.go
  func main() { sdk.Serve(module.NewModule()) }

plugins/myplugin/internal/generate_dts/main.go
  func main() { dtsgen.WriteFile("myplugin.d.ts", module.NewModule()) }
```

This is much cleaner than:

```text
build plugin binary
install somewhere
discover from host
load manifest
try to reconstruct declarations
```

### New SDK typing surface

The SDK should grow explicit TypeScript declaration metadata for:

1. function exports,
2. object methods,
3. optional module-level named type declarations or raw DTS blocks later.

The most practical place to attach metadata is the existing export/method option system in [`pkg/hashiplugin/sdk/export.go`](/home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/pkg/hashiplugin/sdk/export.go).

Suggested new internal fields:

1. `exportDefinition.tsSignature`
2. `methodDefinition.tsSignature`
3. optional `moduleDefinition.tsRawDTS` or named declarations if needed later

Suggested author-facing API shape:

```go
sdk.Function(
    "greet",
    greet,
    sdk.ExportDoc("Return a greeting"),
    sdk.TypeScriptSignature(
        sdk.TSParam("name", spec.String(), sdk.TSOptional()),
        sdk.TSReturns(spec.String()),
    ),
)

sdk.Object(
    "store",
    sdk.ObjectDoc("In-memory key/value store"),
    sdk.Method(
        "get",
        store.get,
        sdk.MethodSummary("Get a value"),
        sdk.TypeScriptSignature(
            sdk.TSParam("key", spec.String()),
            sdk.TSReturns(spec.Union(spec.String(), spec.Named("null"))),
        ),
    ),
)
```

I would not expose raw `spec.Function` directly to authors as the primary API because:

1. plugins have object exports as a first-class concept,
2. the SDK should own the mapping from plugin exports to declaration rendering,
3. a plugin-specific signature helper can remain consistent with the existing option style.

### New generator package

Add a small package such as:

1. `pkg/hashiplugin/sdk/dtsgen`

Suggested API:

```go
package dtsgen

type Options struct {
    Header string
}

func Render(mod *sdk.Module, opts Options) (string, error)
func WriteFile(path string, mod *sdk.Module, opts Options) error
```

The package should:

1. inspect the `sdk.Module`,
2. convert SDK typing metadata into `*spec.Module`,
3. call `validate.Module` / `validate.Bundle`,
4. call `render.Bundle`,
5. write the output file if requested.

This library-first design is the easiest thing for plugin authors to embed in a local `go:generate` helper.

### Relationship to `cmd/gen-dts`

`cmd/gen-dts` should remain the built-in module generator.

It may later share internal helpers with plugin author generation, but it should not be the only user-facing entrypoint for plugin authors.

Recommended split:

1. `cmd/gen-dts`: built-in repo module generation
2. `pkg/hashiplugin/sdk/dtsgen`: plugin-author library for one plugin
3. optional future `cmd/gen-plugin-dts`: thin wrapper once the library API stabilizes

This matters because a generic CLI for arbitrary external plugin repositories is much harder to implement cleanly than a library the plugin repo can call directly.

### Recommended module layout for plugin authors

This is a key part of the implementation guide for an intern.

Current pattern in examples:

1. build the module inline inside `main()`
2. immediately call `sdk.Serve(...)`

Recommended pattern:

1. move the module definition into a reusable package function such as `NewModule() *sdk.Module`
2. keep `main.go` tiny and runtime-only
3. let generation code import the reusable module package

Why:

1. generation should not depend on running `main()`
2. generation code needs a stable importable symbol
3. tests can validate the module definition directly

## Detailed Implementation Guide

### Phase 1: Establish the reusable plugin module pattern

Goal: make plugin definitions importable.

Files likely touched in examples:

1. [plugins/examples/greeter/main.go](/home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/plugins/examples/greeter/main.go)
2. [plugins/examples/kv/main.go](/home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/plugins/examples/kv/main.go)
3. [plugins/examples/system-info/main.go](/home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/plugins/examples/system-info/main.go)

Recommended refactor:

1. create `module.go` or `plugin.go` in each example package,
2. move the `sdk.MustModule(...)` call into `NewModule()`,
3. update `main.go` to call `sdk.Serve(NewModule())`.

Pseudocode:

```go
// module.go
package greeter

func NewModule() *sdk.Module {
    return sdk.MustModule(
        "plugin:examples:greeter",
        // ...
    )
}

// cmd/myplugin/main.go
package main

func main() {
    sdk.Serve(greeter.NewModule())
}
```

### Phase 2: Add SDK-side TypeScript signature metadata

Goal: let plugin authors declare signatures where they already declare exports.

Files:

1. [pkg/hashiplugin/sdk/module.go](/home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/pkg/hashiplugin/sdk/module.go)
2. [pkg/hashiplugin/sdk/export.go](/home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/pkg/hashiplugin/sdk/export.go)
3. new file such as `pkg/hashiplugin/sdk/typescript.go`

Suggested data model:

```go
type TSSignature struct {
    Params  []spec.Param
    Returns spec.TypeRef
}
```

For object methods, attach `TSSignature` to `methodDefinition`.

For function exports, attach `TSSignature` to `exportDefinition`.

Validation rules:

1. function export with typing metadata must have a non-empty return type
2. method typing metadata must have named params
3. no duplicate param names
4. `spec.TypeRef` must pass `pkg/tsgen/validate`

### Phase 3: Convert SDK module definitions to `tsgen/spec`

Goal: teach `sdk.Module` how to become a declaration module.

Files:

1. new file such as `pkg/hashiplugin/sdk/to_tsgen.go`
2. [pkg/tsgen/spec/types.go](/home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/pkg/tsgen/spec/types.go) only if small extensions are needed

Conversion rule:

1. function exports become `spec.Function`
2. object exports become generated `RawDTS` lines until or unless `tsgen` grows first-class object exports

Pseudocode:

```go
func ModuleToSpec(mod *sdk.Module) (*spec.Module, error) {
    manifest, err := mod.Manifest(context.Background())
    if err != nil { return nil, err }

    out := &spec.Module{Name: manifest.GetModuleName()}

    for _, exp := range mod.exportDefinitions() {
        switch exp.kind {
        case function:
            out.Functions = append(out.Functions, spec.Function{
                Name: exp.name,
                Params: exp.tsSignature.Params,
                Returns: exp.tsSignature.Returns,
            })
        case object:
            out.RawDTS = append(out.RawDTS, renderObjectExport(exp))
        }
    }
    return out, nil
}
```

Implementation note:

This is one reason to keep the TypeScript metadata in the SDK definitions, not only in the runtime manifest. The SDK definitions already know whether something is a function export or an object export with methods.

### Phase 4: Add the generation helper package

Goal: give authors a direct `WriteFile(...)` helper.

Files:

1. new package `pkg/hashiplugin/sdk/dtsgen`

Responsibilities:

1. accept `*sdk.Module`
2. call `ModuleToSpec(...)`
3. build a `spec.Bundle`
4. render the bundle
5. write the file

Pseudocode:

```go
func WriteFile(path string, mod *sdk.Module, opts Options) error {
    moduleSpec, err := sdk.ModuleToSpec(mod)
    if err != nil { return err }

    bundle := &spec.Bundle{
        HeaderComment: opts.Header,
        Modules: []*spec.Module{moduleSpec},
    }
    if err := validate.Bundle(bundle); err != nil { return err }

    content, err := render.Bundle(bundle)
    if err != nil { return err }
    return os.WriteFile(path, []byte(content), 0o644)
}
```

### Phase 5: Add example author workflow

Goal: make the feature easy to copy.

Files:

1. example plugin package
2. new generator helper program under examples
3. README or plugin authoring docs

Suggested local helper:

```go
package main

import (
    "log"

    "github.com/go-go-golems/go-go-goja/pkg/hashiplugin/sdk/dtsgen"
    "github.com/go-go-golems/go-go-goja/plugins/examples/greeter"
)

func main() {
    if err := dtsgen.WriteFile("greeter.d.ts", greeter.NewModule(), dtsgen.Options{}); err != nil {
        log.Fatal(err)
    }
}
```

Then the example can use:

```go
//go:generate go run ./internal/generate_dts
```

This is not as flashy as a universal CLI, but it is robust, explicit, and works for plugin authors in their own repositories.

### Phase 6: Optionally mirror the same metadata into the runtime manifest

Goal: keep the door open for future host-side introspection.

This phase is optional for the plugin-author workflow, but useful later.

If the repository eventually wants:

1. runtime docs to show parameter and return types,
2. host-side declaration generation for already-installed plugins,
3. richer IDE tooling in host environments,

then the same SDK-side typing metadata can be serialized into the plugin manifest protobuf.

Important ordering:

1. first solve plugin-author generation from source,
2. then decide whether runtime manifest mirroring is worth the extra contract complexity.

## Testing And Validation Strategy

### Unit tests

Add tests for:

1. signature option validation in new SDK typing files,
2. `sdk.Module -> spec.Module` conversion,
3. object export rendering through `RawDTS`,
4. `dtsgen.Render` and `dtsgen.WriteFile`.

Likely locations:

1. [pkg/hashiplugin/sdk/sdk_test.go](/home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/pkg/hashiplugin/sdk/sdk_test.go)
2. new tests under `pkg/hashiplugin/sdk/dtsgen`
3. maybe `pkg/tsgen/render` tests if new edge cases appear

### Integration tests

Recommended integration path:

1. create an example plugin `NewModule()` package
2. call `dtsgen.Render(...)`
3. assert the generated output contains the expected `declare module "plugin:..."` block
4. compare with a golden file if desired

### Commands for an intern

```bash
go test ./pkg/hashiplugin/sdk/... ./pkg/tsgen/... -count=1
go run ./internal/generate_dts
```

If example plugins are updated:

```bash
go generate ./plugins/examples/...
```

## Risks, Tradeoffs, And Alternatives

### Risk 1: duplicate source of truth

If the SDK typing metadata is added poorly, plugin authors may have to describe the same thing twice: once for runtime exports and once for TypeScript exports.

Mitigation:

Attach TypeScript signatures to the existing `sdk.Function` / `sdk.Method` option flow rather than adding a parallel declaration tree.

### Risk 2: overdesigning a universal CLI too early

A universal `gen-plugin-dts --package ... --symbol ...` command sounds convenient, but it is harder than it looks in Go and not required for a good author experience.

Mitigation:

Start with a library-first design plus tiny local helper programs in plugin repos.

### Risk 3: `tsgen/spec` may not model object exports elegantly

Plugins use object exports heavily. The current `spec.Module` is function-oriented.

Mitigation:

Use internal object-to-`RawDTS` rendering first. Only extend `tsgen/spec` if the ergonomics become painful.

### Alternative 1: keep using discovered plugin binaries as the primary source

Rejected as the main workflow.

Why:

1. wrong ownership boundary
2. awkward for third-party authors
3. weak fit for `go generate`
4. still does not solve signature metadata without SDK changes

### Alternative 2: infer types from handler bodies

Rejected.

Why:

1. fragile
2. incomplete
3. implementation-dependent
4. not suitable for a clean author-facing contract

### Alternative 3: separate sidecar YAML or JSON declaration files

Possible, but weaker.

Pros:

1. easy to serialize
2. no SDK API change required initially

Cons:

1. declarations drift from source more easily
2. extra file management burden for plugin authors
3. misses the chance to keep declaration metadata next to export declarations

## Final Recommendation

The corrected implementation target is:

1. plugin authors define explicit TypeScript signatures in the SDK authoring path,
2. plugin authors generate `.d.ts` from `*sdk.Module` directly,
3. the generator is a small SDK-side helper package, not a discovery-based host tool,
4. runtime manifest mirroring is optional follow-on work.

So the best next step is not "teach `cmd/gen-dts` how to discover plugins." The best next step is "teach the SDK how to hold author-owned TS metadata and render one plugin module into a `.d.ts` file."

## References

Core declaration pipeline:

1. [cmd/gen-dts/main.go](/home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/cmd/gen-dts/main.go)
2. [modules/typing.go](/home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/modules/typing.go)
3. [pkg/tsgen/spec/types.go](/home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/pkg/tsgen/spec/types.go)
4. [pkg/tsgen/render/dts_renderer.go](/home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/pkg/tsgen/render/dts_renderer.go)
5. [pkg/tsgen/validate/validate.go](/home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/pkg/tsgen/validate/validate.go)

Plugin authoring files:

1. [pkg/hashiplugin/sdk/module.go](/home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/pkg/hashiplugin/sdk/module.go)
2. [pkg/hashiplugin/sdk/export.go](/home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/pkg/hashiplugin/sdk/export.go)
3. [pkg/hashiplugin/sdk/call.go](/home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/pkg/hashiplugin/sdk/call.go)
4. [pkg/hashiplugin/sdk/sdk_test.go](/home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/pkg/hashiplugin/sdk/sdk_test.go)

Example plugins to refactor toward `NewModule()`:

1. [plugins/examples/greeter/main.go](/home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/plugins/examples/greeter/main.go)
2. [plugins/examples/kv/main.go](/home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/plugins/examples/kv/main.go)
3. [plugins/examples/system-info/main.go](/home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/plugins/examples/system-info/main.go)

Runtime-side context kept only as secondary background:

1. [pkg/hashiplugin/contract/jsmodule.proto](/home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/pkg/hashiplugin/contract/jsmodule.proto)
2. [pkg/hashiplugin/host/client.go](/home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/pkg/hashiplugin/host/client.go)
3. [pkg/hashiplugin/host/reify.go](/home/manuel/workspaces/2026-03-01/generate-js-types/go-go-goja/pkg/hashiplugin/host/reify.go)
