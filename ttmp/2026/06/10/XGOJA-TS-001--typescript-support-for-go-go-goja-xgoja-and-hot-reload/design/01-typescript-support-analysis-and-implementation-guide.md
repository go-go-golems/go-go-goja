---
Title: TypeScript support analysis and implementation guide
Ticket: XGOJA-TS-001
Status: active
Topics:
    - goja
    - xgoja
    - typescript
    - tooling
    - developer-experience
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/cmd/xgoja/internal/buildspec/build_spec.go
      Note: Build-time xgoja YAML schema and JSVerbSourceSpec TypeScript extension point
    - Path: go-go-goja/pkg/jsverbs/runtime.go
      Note: jsverbs require loader and overlay injection path
    - Path: go-go-goja/pkg/jsverbs/scan.go
      Note: JavaScript scanner that needs TypeScript transform input
    - Path: go-go-goja/pkg/xgoja/app/run.go
      Note: xgoja run script execution path that needs .ts entry support
    - Path: go-go-goja/pkg/xgoja/app/runtime_spec.go
      Note: Runtime spec fields embedded in generated xgoja binaries
    - Path: go-go-goja/pkg/xgoja/hotreload/manager.go
      Note: Blue/green reload snapshot manager used by HTTP hot reload
    - Path: go-go-goja/pkg/xgoja/providers/http/serve.go
      Note: HTTP serve command hot reload integration and watch extension defaults
ExternalSources:
    - local:01-goja-typescript-esbuild-note.md
Summary: Design and intern implementation guide for TypeScript execution, xgoja JS verb support, and HTTP hot reload integration.
LastUpdated: 2026-06-10T21:35:00-04:00
WhatFor: Use when implementing TypeScript source support in go-go-goja, generated xgoja runtimes, and hot-reloadable HTTP JS verbs.
WhenToUse: Before changing jsverbs scanning, xgoja YAML schemas, runtime source loaders, or hot reload behavior for .ts/.tsx sources.
---


# TypeScript support for go-go-goja, xgoja, and hot reload

## Executive summary

This ticket is about making TypeScript a first-class authoring format for the existing goja-based scripting system without turning goja into Node.js. The recommended implementation is to compile TypeScript to JavaScript with esbuild's Go API, then continue to execute JavaScript through the existing goja, goja_nodejs `require()`, jsverbs, xgoja provider, and hot reload paths.

The important constraint is that TypeScript support has two different runtime shapes:

- **Static/embedded sources** should be compiled during xgoja generation or build time, then embedded as JavaScript. This keeps production generated binaries small in behavior and avoids runtime compilation surprises.
- **Editable development sources** should be compiled in process at runtime because `xgoja run`, filesystem-backed jsverb sources, and HTTP hot reload need to react to files on disk without rebuilding the Go binary.

The imported source note in `sources/local/01-goja-typescript-esbuild-note.md` is the seed for this design. It points out that `github.com/evanw/esbuild/pkg/api` is usable as a Go library with no Node/npm runtime requirement, that `Transform` is appropriate for one source string, that `Build` is appropriate when imports must be followed, and that esbuild strips types but does not type-check TypeScript.

The proposed implementation adds a small TypeScript compilation package, threads compiler options through xgoja specs and jsverb scan options, and changes jsverb runtime loading so a `.ts` source file can be scanned for command metadata and served to goja as compiled JavaScript. HTTP hot reload then mostly works by composition: it already rescans jsverb sources, creates a candidate runtime, smoke-tests it, and only swaps the candidate into live traffic after success.

## Problem statement

Today the repository has several adjacent but separate capabilities:

- goja runtime creation with Go-backed CommonJS modules.
- xgoja code generation from `xgoja.yaml`.
- jsverbs scanning that turns JavaScript functions into Glazed/Cobra commands.
- generated runtime declaration generation through `.d.ts` files.
- HTTP serve hot reload for jsverb-defined Express routes.

The gap is that these systems treat executable scripts as JavaScript. There is existing TypeScript-related developer experience in the `.d.ts` generation path, but there is not yet an integrated way to author xgoja scripts or jsverbs in `.ts`/`.tsx` and execute them directly.

The implementation should let an intern answer these concrete questions:

1. Where should TypeScript be compiled?
2. How does compiled JavaScript enter goja?
3. How do xgoja-generated binaries discover TypeScript sources?
4. How does hot reload know that a TypeScript edit changed the served runtime?
5. How do editor declarations, runtime module aliases, and esbuild external modules line up?

## Scope and non-goals

### In scope

- Add a shared TypeScript compilation layer based on esbuild's Go API.
- Support `.ts`, `.tsx`, `.mts`, and `.cts` as source extensions where appropriate.
- Support `xgoja run file.ts` for development scripts.
- Support filesystem-backed jsverb sources authored in TypeScript.
- Support HTTP serve hot reload for TypeScript jsverb sources.
- Preserve the existing generated `.d.ts` declaration generation path and explain how it complements TypeScript execution.
- Make implementation decisions explicit and testable.

### Out of scope for the first implementation

- Full TypeScript type checking inside go-go-goja. esbuild strips types; it does not run the TypeScript checker.
- Node.js runtime emulation beyond modules already provided by go-go-goja and goja_nodejs.
- A complete npm package manager or lockfile workflow inside xgoja.
- Bundling arbitrary browser applications. Existing asset embedding can remain separate from script execution.
- Replacing jsverbs' command metadata model.

## Imported TypeScript note distilled

The imported source file gives the central technical direction:

- Add `github.com/evanw/esbuild` and import `github.com/evanw/esbuild/pkg/api`.
- Use `api.Transform` for a single `.ts` source string with no dependency graph.
- Use `api.Build` with `Bundle: true` and `Write: false` for a `.ts` entry point with imports.
- Prefer a conservative output target such as `ES2015` for goja compatibility.
- Do not assume Node built-ins exist in goja; native capabilities should be provided as explicit goja/xgoja modules.
- Run `tsc --noEmit` separately if the project wants type checking.

That model maps cleanly onto this repository: go-go-goja should own transpilation and execution, while users may optionally add `tsc --noEmit` to their project CI if they want static type checking.

## Current-state architecture

### Repository boundaries

The relevant packages and commands are:

| Area | Files | Current responsibility |
| --- | --- | --- |
| xgoja build schema | `cmd/xgoja/internal/buildspec/build_spec.go` | YAML DTOs for `xgoja.yaml`; includes packages, modules, commands, jsverbs, help, and assets. |
| xgoja runtime schema | `pkg/xgoja/app/runtime_spec.go` | Runtime DTO embedded into generated binaries. |
| xgoja run command | `pkg/xgoja/app/run.go` | Runs a script file inside a generated runtime. |
| xgoja jsverb commands | `pkg/xgoja/app/root.go` | Scans configured jsverb sources and mounts Glazed/Cobra commands. |
| jsverbs scanner and loader | `pkg/jsverbs/scan.go`, `pkg/jsverbs/runtime.go` | Finds JS files, extracts command metadata, and provides a `require()` overlay loader. |
| Type declarations | `pkg/xgoja/dtsgen/dtsgen.go`, `pkg/tsgen/*` | Generates `.d.ts` declarations for selected xgoja module aliases. |
| hot reload core | `pkg/xgoja/hotreload/manager.go`, `pkg/xgoja/hotreload/watch.go` | Creates candidate runtimes, swaps successful snapshots, and polls filesystem roots. |
| HTTP provider hot reload | `pkg/xgoja/providers/http/serve.go` | Wires hot reload into HTTP serve commands for jsverb-defined routes. |
| existing TS bundling example | `cmd/bun-demo/js/package.json`, `pkg/doc/bun-goja-bundling-playbook.md` | Demonstrates external esbuild CLI bundling; not currently integrated into xgoja runtime paths. |

### xgoja YAML is build-time input

`cmd/xgoja/internal/buildspec/build_spec.go` defines the full build-time `BuildSpec`. It is explicitly a declarative DTO: comments state that it describes what the generator should build, import, embed, or expose, and that runtime binaries should use a smaller `app.RuntimeSpec` instead. The top-level `BuildSpec` already contains `JSVerbs []JSVerbSourceSpec`, `Assets []AssetSourceSpec`, `Packages`, `Modules`, and `Commands` (`cmd/xgoja/internal/buildspec/build_spec.go:1-31`).

`JSVerbSourceSpec` already has the fields needed to choose files: `ID`, `Path`, `Embed`, provider `Package`/`Source`, `Include`, `Exclude`, and `Extensions` (`cmd/xgoja/internal/buildspec/build_spec.go:118-128`). That makes TypeScript support a schema extension, not a new source kind.

### Generated binaries use a smaller runtime spec

`pkg/xgoja/app/runtime_spec.go` mirrors only runtime-relevant fields. It includes runtime `Modules`, `Commands`, `CommandProviders`, `JSVerbs`, `Help`, and `Assets` (`pkg/xgoja/app/runtime_spec.go:12-28`). Its `JSVerbSourceSpec` also carries `Extensions` (`pkg/xgoja/app/runtime_spec.go:88-98`).

This split matters for TypeScript:

- Build-only fields such as output directories, package versions, and generation policies should live in `buildspec`.
- Runtime compilation policy for non-embedded filesystem sources must be represented in `app.RuntimeSpec`, because generated binaries need it after build time.
- Embedded compiled JavaScript should not need runtime TypeScript fields at all if generation rewrites the embedded source tree to JavaScript.

### xgoja run currently executes CommonJS JavaScript modules

`pkg/xgoja/app/run.go` creates a `run` command that executes a JavaScript file in a generated runtime. It resolves the absolute script path, checks that it exists, derives require module roots from the script directory, creates a runtime, and finally calls `rt.Require.Require(scriptPath)` inside the goja owner loop (`pkg/xgoja/app/run.go:65-113`).

That means `xgoja run` is currently module-loader based, not a raw `vm.RunScript` path. This is useful because it already supports relative `require()` and native module aliases, but it also means TypeScript support must either:

- compile a `.ts` file into a temporary or virtual JavaScript module that the loader can require, or
- bypass `Require()` for `.ts` and run a bundled JavaScript string directly.

The better long-term design is to keep module-loader semantics, because jsverbs and generated xgoja commands already compose around `require.WithLoader(...)`.

### Module roots are script-relative

`pkg/engine/module_roots.go` resolves module roots from a script path. By default it includes the script directory, parent directory, and corresponding `node_modules` folders (`pkg/engine/module_roots.go:1-31`). `RequireOptionWithModuleRootsFromScript` turns those roots into `require.WithGlobalFolders(...)` (`pkg/engine/module_roots.go:82-94`).

This is important for TypeScript projects that have local helpers or `node_modules`. The implementation should not invent a second resolution system unless esbuild needs a `ResolveDir` for bundling.

### goja runtime creation is already centralized

`pkg/engine/factory.go` builds runtimes from a frozen runtime composition. `Build()` validates modules and runtime initializers, applies module middleware, and returns a `RuntimeFactory` (`pkg/engine/factory.go:122-180`). `NewRuntime()` constructs a goja VM, starts a goja_nodejs event loop, creates a `RuntimeOwner`, registers runtime modules, enables `require`, console, buffer, URL, and then runs runtime initializers (`pkg/engine/factory.go:190-280`).

TypeScript support should not alter this lifecycle. It should compile source before `require` loads it, then let the current runtime factory register modules exactly as before.

### jsverbs scanning assumes JavaScript source

The jsverbs package defaults to scanning `.js` and `.cjs` (`pkg/jsverbs/model.go:44-52`). `ScanDir` walks the root, filters by include/exclude and extension, reads files, and collects source inputs (`pkg/jsverbs/scan.go:13-79`). `ScanFS` does the same for an `fs.FS` (`pkg/jsverbs/scan.go:82-130`). `scanInput` uses the tree-sitter JavaScript grammar (`tree_sitter_javascript.Language()`) and parses the source as JavaScript (`pkg/jsverbs/scan.go:290-305`).

At runtime, `Registry.RequireLoader()` exposes a loader backed by `sourceLoader` (`pkg/jsverbs/runtime.go:40-45`). `InvokeInRuntime()` requires the verb's module path and then looks up a function in `globalThis.__glazedVerbRegistry` (`pkg/jsverbs/runtime.go:48-88`). `sourceLoader()` injects an overlay into the original source before returning it to goja (`pkg/jsverbs/runtime.go:159-164`).

This is the main TypeScript seam. A `.ts` file cannot be passed directly to the current tree-sitter JavaScript grammar or to goja. The scanner and loader need a transform/bundle hook that turns TypeScript into JavaScript while preserving jsverb metadata and the overlay capture behavior.

### xgoja already generates TypeScript declarations

Provider modules can carry TypeScript descriptors. `providerapi.Module` has a `TypeScript *spec.Module` field (`pkg/xgoja/providerapi/module.go:31-41`). `pkg/tsgen/spec/types.go` defines a simple declaration AST for `declare module "name"`, functions, params, and type refs (`pkg/tsgen/spec/types.go:1-48`).

`pkg/xgoja/dtsgen/dtsgen.go` renders declarations for the selected xgoja module instances. It resolves provider modules, applies each runtime alias as the declaration module name, validates descriptors, reports missing descriptors in non-strict mode, and fails in strict mode (`pkg/xgoja/dtsgen/dtsgen.go:22-83`).

The xgoja docs already teach users to run `xgoja gen-dts`, point `tsconfig.json` at the generated file, use `--strict`, and use generated binaries' `types` command (`cmd/xgoja/doc/14-tutorial-typescript-declarations.md:1-120`).

This means runtime TypeScript support should reuse the existing declaration story instead of generating a separate one.

### hot reload is already blue/green

The hot reload manager creates candidate versions and only swaps them into service after load and optional smoke test success. `Reload()` creates a new candidate host, calls the configured `Load` function, records routes, runs `Smoke` if configured, swaps the active pointer, and retires the old runtime asynchronously (`pkg/xgoja/hotreload/manager.go:62-111`). `ServeHTTP` delegates to the active snapshot's host or returns 503 when no runtime is ready (`pkg/xgoja/hotreload/manager.go:113-120`).

The watch loop polls roots, filters by extension, debounces changes, and calls `Reload()` (`pkg/xgoja/hotreload/watch.go:20-75`). By default it ignores `.bin`, `.git`, `dist`, and `node_modules` (`pkg/xgoja/hotreload/watch.go:13-18`).

The HTTP provider composes this manager. In `serveVerbHotReload`, it creates a manager whose `Load` function rescans jsverb sources, creates a new runtime with host services, initializes selected modules, invokes the active verb to register routes, and returns the candidate runtime (`pkg/xgoja/providers/http/serve.go:126-181`). It starts the watcher, triggers an initial reload, registers a JSON status endpoint, and serves the manager behind one Go HTTP listener (`pkg/xgoja/providers/http/serve.go:190-225`). The command exposes `--hot-reload-watch-ext`, whose current default is `.js`, `.json`, `.md`, `.yaml`, and `.yml` (`pkg/xgoja/providers/http/serve.go:378-392`).

TypeScript hot reload therefore needs only two structural changes:

- the watcher must include `.ts`, `.tsx`, `.mts`, and `.cts` when TypeScript support is enabled;
- the rescan/load path must compile TypeScript before parser and goja execution.

## Gap analysis

### Gap 1: No shared TypeScript compiler abstraction

The repository has esbuild usage in a Bun demo through JavaScript tooling, but `go.mod` does not currently depend on `github.com/evanw/esbuild`. There is no reusable Go package that converts TypeScript source to JavaScript for goja.

### Gap 2: jsverbs can filter `.ts`, but cannot safely parse or execute it

`JSVerbSourceSpec.Extensions` can already include `.ts`, but the scanner still uses tree-sitter JavaScript and the runtime loader still returns the original source with an overlay. That will fail for TypeScript syntax such as parameter types, interfaces, type-only imports, `as` casts, and TSX.

### Gap 3: embedded and filesystem sources need different compilation timing

For `embed: true`, xgoja generation copies source directories into generated Go output. The generated binary can then scan embedded files. If embedded TypeScript is copied unchanged, the generated binary must ship esbuild and compile at runtime. That is useful for development but not ideal for production.

For `embed: false`, hot reload and `xgoja run` must compile from disk at runtime because the source changes without rebuilding the generated binary.

### Gap 4: local imports need bundling, not only transform

`api.Transform` is sufficient for isolated files, but jsverb files and `xgoja run` scripts often import helpers. If compiled output contains `require("./helper")`, goja_nodejs resolution may not try `.ts` extensions by default, and helper files may not be available as JavaScript. Bundling the entry file with esbuild avoids this mismatch for runtime execution.

### Gap 5: module aliases must be externalized consistently

xgoja native modules are exposed through `require("alias")`, where alias comes from `modules[].as` or the provider module name. esbuild must not try to resolve those aliases from `node_modules`. It should mark selected xgoja module aliases as external so the compiled output still calls goja's `require("alias")` at runtime.

## Proposed architecture

### High-level flow

```text
TypeScript authoring
  |
  |  .ts/.tsx source files + optional generated xgoja-modules.d.ts
  v
TypeScript compiler facade (new pkg/typescript or pkg/tsscript)
  |
  |-- Transform for scanner metadata and simple eval/run strings
  |-- Build+Bundle for executable entry modules and imports
  v
JavaScript artifact
  |
  |-- xgoja run: execute compiled entry in a generated runtime
  |-- jsverbs: scan metadata, compile executable loader source, inject overlay
  |-- HTTP hot reload: rescan/recompile candidate, smoke test, swap
  v
goja + goja_nodejs require + xgoja provider modules
```

### New package: `pkg/typescript` or `pkg/tsscript`

Create one small package that owns esbuild integration. The package name should avoid colliding with Go's historical `go/types` mental model; `pkg/tsscript` is explicit and short.

Recommended files:

```text
pkg/tsscript/
  options.go        // public options and defaults
  compiler.go       // TransformSource, BundleEntry, BundleVirtualEntry
  diagnostics.go    // esbuild message formatting
  extensions.go     // IsTypeScriptExtension, LoaderForPath
  compiler_test.go  // isolated unit tests
```

Proposed public API:

```go
package tsscript

import "github.com/evanw/esbuild/pkg/api"

type Format string

const (
    FormatIIFE Format = "iife"
    FormatCJS  Format = "cjs"
)

type Options struct {
    Enabled     bool
    Target      api.Target
    Format      api.Format
    Platform    api.Platform
    Bundle      bool
    Sourcemap   api.SourceMap
    JSX         api.JSX
    External    []string
    Define      map[string]string
    Tsconfig    string
    SourceRoot  string
    Loader      api.Loader
}

type Source struct {
    Path       string // user-facing path for diagnostics
    AbsPath    string // optional filesystem path
    ResolveDir string // base directory for relative imports
    Contents   []byte
}

type Artifact struct {
    Path        string
    Code        []byte
    SourceMap   []byte
    Warnings    []Diagnostic
    LoaderUsed  api.Loader
    Bundled     bool
}

func DefaultOptions() Options
func IsTypeScriptPath(path string) bool
func LoaderForPath(path string) api.Loader
func TransformSource(src Source, opts Options) (*Artifact, error)
func BundleVirtualEntry(src Source, opts Options) (*Artifact, error)
func BundleEntry(entryPath string, opts Options) (*Artifact, error)
```

Implementation rules:

- `TransformSource` should use `api.Transform`.
- `BundleEntry` and `BundleVirtualEntry` should use `api.Build` with `Bundle: true` and `Write: false`.
- Default `Target` should be `api.ES2015` until tests prove a newer target is safe.
- `External` should include xgoja selected module aliases, plus user-configured externals.
- Return all esbuild errors in one Go error, with file, line, column, and text.
- Do not run `tsc`; document that users can run `tsc --noEmit` separately.

### xgoja schema extension

Add a reusable TypeScript config block to both build-time and runtime specs.

Build-time schema sketch:

```go
type TypeScriptSpec struct {
    Enabled       bool              `yaml:"enabled" json:"enabled"`
    Bundle        bool              `yaml:"bundle" json:"bundle"`
    Target        string            `yaml:"target,omitempty" json:"target,omitempty"`
    Format        string            `yaml:"format,omitempty" json:"format,omitempty"`
    Platform      string            `yaml:"platform,omitempty" json:"platform,omitempty"`
    Tsconfig      string            `yaml:"tsconfig,omitempty" json:"tsconfig,omitempty"`
    SourceMap     string            `yaml:"sourcemap,omitempty" json:"sourcemap,omitempty"`
    External      []string          `yaml:"external,omitempty" json:"external,omitempty"`
    Define        map[string]string `yaml:"define,omitempty" json:"define,omitempty"`
    CheckCommand  []string          `yaml:"checkCommand,omitempty" json:"checkCommand,omitempty"`
}

type JSVerbSourceSpec struct {
    // existing fields...
    TypeScript *TypeScriptSpec `yaml:"typescript,omitempty" json:"typescript,omitempty"`
}
```

Runtime schema should include the same fields only when runtime compilation is needed. If an embedded source is precompiled during generation, its runtime `TypeScript` field can be omitted or disabled in the emitted runtime spec.

YAML example for filesystem-backed development jsverbs:

```yaml
jsverbs:
  - id: app
    path: verbs
    embed: false
    extensions: [".ts", ".tsx", ".js"]
    include: ["**/*.{ts,tsx,js}"]
    exclude: ["**/*.test.ts"]
    typescript:
      enabled: true
      bundle: true
      target: es2015
      format: cjs
      platform: neutral
      tsconfig: tsconfig.json
      external:
        - express
        - fs:assets
```

YAML example for embedded production jsverbs:

```yaml
jsverbs:
  - id: app
    path: verbs
    embed: true
    extensions: [".ts"]
    typescript:
      enabled: true
      bundle: true
      target: es2015
      format: cjs
```

For embedded production mode, generation should compile `verbs/**/*.ts` into generated JavaScript under `xgoja_embed/jsverbs/<id>/...` and emit runtime extensions as `.js`. The generated binary should not need runtime esbuild for these sources.

### jsverbs scanner extension

Add a source transform hook to `jsverbs.ScanOptions`.

```go
type ScanSourceTransform func(SourceFile) (SourceFile, error)

type ScanOptions struct {
    Extensions             []string
    FailOnErrorDiagnostics bool
    Include                []string
    Exclude                []string
    Transform              ScanSourceTransform
    RuntimeLoader          SourceRuntimeLoader // optional, see below
}
```

A transform lets `ScanDir`, `ScanFS`, and `ScanSources` keep their filesystem behavior but normalize source before tree-sitter sees it.

Recommended data model change:

```go
type FileSpec struct {
    AbsPath          string
    RelPath          string
    ModulePath       string
    Source           []byte // executable source returned by loader, or original when no transform
    OriginalSource   []byte // optional original TS source for diagnostics
    SourceLanguage   string // "javascript" or "typescript"
    ExecutableSource []byte // compiled JS if known
    // existing fields...
}
```

The scanner should parse JavaScript only. For TypeScript files, it should parse the transformed JavaScript produced by `tsscript.TransformSource` or by a dedicated jsverb compile adapter. This avoids introducing a second tree-sitter grammar in the first implementation.

### jsverbs runtime loader extension

The current loader appends the overlay to the source and returns it. For TypeScript with local imports, the better order is:

1. Build overlay text from extracted function names.
2. Append overlay text to the original TypeScript source.
3. Bundle the virtual entry with esbuild.
4. Mark xgoja native module aliases as external.
5. Return the compiled JavaScript to goja_nodejs.

Pseudocode:

```go
func (r *Registry) sourceLoader(modulePath string) ([]byte, error) {
    file := r.filesByModule[modulePath]
    if file == nil {
        return nil, require.ModuleFileDoesNotExistError
    }

    overlay := r.overlayForFile(modulePath, file)

    if file.SourceLanguage != "typescript" {
        return append(file.Source, overlay...), nil
    }

    sourceWithOverlay := append(file.OriginalSource, []byte("\n")...)
    sourceWithOverlay = append(sourceWithOverlay, []byte(overlay)...)

    artifact, err := r.compiler.BundleVirtualEntry(tsscript.Source{
        Path:       file.RelPath,
        AbsPath:    file.AbsPath,
        ResolveDir: filepath.Dir(file.AbsPath),
        Contents:   sourceWithOverlay,
    }, r.compilerOptionsFor(file))
    if err != nil {
        return nil, err
    }
    return artifact.Code, nil
}
```

Why append overlay before bundling? Because esbuild wraps modules in closures when it bundles. If the overlay is appended after bundling, the top-level verb function may no longer be in scope. Appending before bundling lets esbuild compile the function and overlay together.

### xgoja run extension

`xgoja run` should detect TypeScript entry files by extension.

Pseudocode:

```go
func runScriptFileWithInitializers(...) error {
    // existing setup: abs path, require roots, runtime creation

    if !tsscript.IsTypeScriptPath(scriptPath) {
        _, err = rt.Owner.Call(ctx, "xgoja.run", func(_ context.Context, vm *goja.Runtime) (any, error) {
            return rt.Require.Require(scriptPath)
        })
        return err
    }

    aliases := aliasesFromSelectedModules(selectedModules)
    artifact, err := tsscript.BundleEntry(scriptPath, tsscript.Options{
        Enabled:  true,
        Bundle:   true,
        Format:   api.FormatIIFE,
        Target:   api.ES2015,
        Platform: api.PlatformNeutral,
        External: aliases,
    })
    if err != nil {
        return fmt.Errorf("compile TypeScript %s: %w", scriptPath, err)
    }

    _, err = rt.Owner.Call(ctx, "xgoja.run.ts", func(_ context.Context, vm *goja.Runtime) (any, error) {
        return vm.RunScript(scriptPath, string(artifact.Code))
    })
    return err
}
```

For initial support, use `FormatIIFE` for `xgoja run` because the command does not need exports. If tests show that external `require("alias")` calls need CommonJS wrappers, switch to a virtual require loader with `FormatCJS`.

### HTTP hot reload extension

The HTTP hot reload provider should not learn how to compile TypeScript directly. It should pass TypeScript-aware jsverb scan options to `commandCtx.JSVerbs`, then keep the current manager flow.

Changes needed:

- `serveHotReloadSection()` should include TypeScript extensions in the default `hot-reload-watch-ext` when any selected jsverb source enables TypeScript.
- `defaultServeHotReloadWatchRoots()` already returns non-embedded filesystem source paths; keep that behavior.
- `resolveServeHotReloadVerb()` already rescans all jsverb sources on each reload. If the source set compiles TypeScript during scan/load, hot reload inherits the behavior.

Pseudocode:

```go
func defaultWatchExtsForSources(sources JSVerbSourceSet) []string {
    exts := []string{".js", ".json", ".md", ".yaml", ".yml"}
    if sourcesHasTypeScript(sources) {
        exts = append(exts, ".ts", ".tsx", ".mts", ".cts")
    }
    return unique(exts)
}
```

### Type declarations and editor setup

Runtime TypeScript support does not replace `xgoja gen-dts`. The intern should treat declarations as the editor-facing contract and esbuild as the runtime compiler.

Recommended project layout:

```text
my-xgoja-project/
  xgoja.yaml
  verbs/
    sites.ts
    helpers.ts
  js/
    types/
      xgoja-modules.d.ts
  tsconfig.json
```

Recommended `tsconfig.json`:

```json
{
  "compilerOptions": {
    "target": "ES2015",
    "module": "CommonJS",
    "moduleResolution": "Bundler",
    "strict": true,
    "types": [],
    "typeRoots": ["./js/types", "./node_modules/@types"],
    "skipLibCheck": true
  },
  "include": ["verbs/**/*.ts", "js/types/**/*.d.ts"]
}
```

Recommended commands:

```bash
xgoja gen-dts -f xgoja.yaml --out js/types/xgoja-modules.d.ts --strict
tsc --noEmit
xgoja build -f xgoja.yaml
./dist/my-runtime serve sites demo --hot-reload --hot-reload-smoke-path /healthz
```

## Decision records

### Decision: Use esbuild Go API as the compiler backend

- **Context:** The system needs TypeScript-to-JavaScript compilation inside Go binaries and generated xgoja programs.
- **Options considered:** Shell out to `esbuild`; require Node/npm; use the esbuild Go API; implement TypeScript parsing manually.
- **Decision:** Use `github.com/evanw/esbuild/pkg/api` behind a small repository-owned facade.
- **Rationale:** The imported note confirms the Go API needs no Node/npm runtime and provides both `Transform` and `Build`. It also cleanly separates transpilation from type checking.
- **Consequences:** Generated binaries that compile TypeScript at runtime will include esbuild. Production embedded sources should preferably be precompiled during xgoja generation.
- **Status:** proposed.

### Decision: Compile TypeScript before jsverbs parsing

- **Context:** jsverbs currently uses the tree-sitter JavaScript grammar, not TypeScript.
- **Options considered:** Add tree-sitter TypeScript; compile TypeScript to JavaScript before parsing; parse TypeScript with a separate custom parser.
- **Decision:** Compile to JavaScript before the existing scanner parses.
- **Rationale:** This preserves the existing metadata extraction and avoids expanding the parser surface in the first implementation.
- **Consequences:** Scanner diagnostics should report original source paths, but line/column mapping may be less precise until source maps are threaded through diagnostics.
- **Status:** proposed.

### Decision: Bundle TypeScript jsverb runtime entries, not just transform them

- **Context:** TypeScript source commonly imports helper files. goja_nodejs may not resolve `.ts` helpers the same way esbuild does.
- **Options considered:** Per-file transform only; custom require loader for `.ts` extension resolution; bundle each entry; precompile a whole source tree.
- **Decision:** Bundle each executable jsverb entry when TypeScript is enabled, with native xgoja module aliases marked as external.
- **Rationale:** Bundling makes local helper imports deterministic while preserving `require("alias")` calls for Go-backed xgoja modules.
- **Consequences:** The overlay must be appended before bundling so captured functions remain in scope.
- **Status:** proposed.

### Decision: Keep type checking outside the runtime path

- **Context:** esbuild strips types but does not type-check TypeScript.
- **Options considered:** Run `tsc --noEmit` inside xgoja; run `tsc` only in CI/user projects; ignore type checking entirely.
- **Decision:** xgoja should expose optional documentation/hooks for `tsc --noEmit`, but runtime execution should not depend on type checking.
- **Rationale:** Runtime reload should be fast and local. Type checking can be slow, project-specific, and sensitive to npm package setup.
- **Consequences:** A TypeScript file with type errors but valid erasable syntax may still run. Users who want stricter guarantees should add `tsc --noEmit` to CI or a pre-build script.
- **Status:** proposed.

### Decision: Precompile embedded TypeScript during generation when possible

- **Context:** Embedded xgoja sources are copied into generated output at build time.
- **Options considered:** Embed `.ts` and compile in the generated binary; compile to `.js` during generation; require users to precompile manually.
- **Decision:** Add generation-time compilation for embedded TypeScript sources.
- **Rationale:** It keeps production runtime behavior simple and avoids shipping esbuild where it is unnecessary.
- **Consequences:** Runtime specs need to accurately reflect generated `.js` paths/extensions, and diagnostics should still point users back to original `.ts` paths.
- **Status:** proposed.

## Implementation guide for a new intern

### Phase 0: Add a minimal failing test matrix

Start with tests before changing behavior. The smallest useful matrix is:

1. `pkg/tsscript`: compile a string with `type User = ...` to runnable JavaScript.
2. `pkg/tsscript`: bundle an entry file that imports `./helper.ts`.
3. `pkg/jsverbs`: scan a TypeScript verb file with a typed parameter and discover the verb.
4. `pkg/jsverbs`: invoke that TypeScript verb in a runtime with a native module marked external.
5. `pkg/xgoja/app`: run a `.ts` file through `xgoja run` or the underlying helper.
6. `pkg/xgoja/providers/http`: hot reload a `.ts` site verb and keep the old version after a broken edit.

### Phase 1: Add the compiler facade

Files to create:

- `/home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/pkg/tsscript/options.go`
- `/home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/pkg/tsscript/compiler.go`
- `/home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/pkg/tsscript/diagnostics.go`
- `/home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/pkg/tsscript/compiler_test.go`

Commands:

```bash
cd /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja
go get github.com/evanw/esbuild
go test ./pkg/tsscript -count=1
```

Implementation sketch:

```go
func TransformSource(src Source, opts Options) (*Artifact, error) {
    result := api.Transform(string(src.Contents), api.TransformOptions{
        Loader:     LoaderForPath(src.Path),
        Sourcefile: src.Path,
        Target:     targetOrDefault(opts.Target),
        Format:     formatOrDefault(opts.Format, api.FormatIIFE),
        Sourcemap:  opts.Sourcemap,
        TsconfigRaw: tsconfigRawIfNeeded(opts),
    })
    if len(result.Errors) > 0 {
        return nil, formatMessages("typescript transform", result.Errors)
    }
    return &Artifact{Path: src.Path, Code: result.Code, Warnings: convertWarnings(result.Warnings)}, nil
}
```

```go
func BundleVirtualEntry(src Source, opts Options) (*Artifact, error) {
    result := api.Build(api.BuildOptions{
        Stdin: &api.StdinOptions{
            Contents:   string(src.Contents),
            Sourcefile: src.Path,
            ResolveDir: src.ResolveDir,
            Loader:     LoaderForPath(src.Path),
        },
        Bundle:   true,
        Write:    false,
        Format:   formatOrDefault(opts.Format, api.FormatCommonJS),
        Target:   targetOrDefault(opts.Target),
        Platform: platformOrDefault(opts.Platform),
        External: append([]string(nil), opts.External...),
        Sourcemap: opts.Sourcemap,
    })
    if len(result.Errors) > 0 {
        return nil, formatMessages("typescript bundle", result.Errors)
    }
    if len(result.OutputFiles) == 0 {
        return nil, fmt.Errorf("typescript bundle produced no output")
    }
    return &Artifact{Path: src.Path, Code: result.OutputFiles[0].Contents, Bundled: true}, nil
}
```

### Phase 2: Thread TypeScript options through buildspec and runtime spec

Files to edit:

- `cmd/xgoja/internal/buildspec/build_spec.go`
- `cmd/xgoja/internal/buildspec/load.go`
- `cmd/xgoja/internal/buildspec/validate.go`
- `pkg/xgoja/app/runtime_spec.go`
- tests under `cmd/xgoja/internal/buildspec/*_test.go`

Work items:

- Add `TypeScriptSpec` to build and runtime DTOs.
- Default `typescript.enabled` to true when a source has TypeScript extensions? Prefer explicit `enabled: true` for the first implementation to avoid surprising users.
- Validate target/format/platform strings.
- Validate `tsconfig` path for embedded/generation-time compilation.
- Validate empty external entries.

### Phase 3: Add TypeScript-aware jsverb scan/load

Files to edit:

- `pkg/jsverbs/model.go`
- `pkg/jsverbs/scan.go`
- `pkg/jsverbs/runtime.go`
- `pkg/xgoja/app/root.go`
- `pkg/xgoja/app/jsverb_sources.go`

Work items:

- Add a scan transform hook.
- Preserve original source and executable source separately.
- Build xgoja module alias externals from selected modules.
- Append overlay before bundling for TypeScript runtime loader.
- Add tests for typed function params, local helper imports, and native `require("alias")` calls.

Important invariant:

> The function names extracted during scan must match the function names captured by the runtime overlay after compilation.

Test this explicitly with a function named `demo` and a `__verb__("demo", ...)` declaration.

### Phase 4: Implement `xgoja run file.ts`

Files to edit:

- `pkg/xgoja/app/run.go`
- `pkg/xgoja/app/run_module_sections_test.go` or a new `run_typescript_test.go`

Work items:

- Detect `.ts`, `.tsx`, `.mts`, `.cts` after path resolution.
- Compile with esbuild before execution.
- Include selected xgoja module aliases as externals.
- Prefer IIFE execution first; switch to a loader-backed CommonJS execution if tests require CommonJS semantics.

### Phase 5: Update HTTP hot reload defaults

Files to edit:

- `pkg/xgoja/providers/http/serve.go`
- `pkg/xgoja/providers/http/serve_test.go`

Work items:

- Include `.ts`, `.tsx`, `.mts`, `.cts` in watch extensions when TypeScript jsverb sources are active.
- Add a hot reload test based on `TestServeVerbHotReloadServesStatusAndReloadsChangedSource`, but write `sites.ts` instead of `sites.js`.
- Ensure a broken TypeScript edit records `LastError` and does not change `ActiveVersion`.

### Phase 6: Generation-time compilation for embedded jsverbs

Files to edit:

- `cmd/xgoja/internal/generate/generate.go`
- `cmd/xgoja/internal/generate/templates.go`
- `cmd/xgoja/internal/generate/generate_test.go`

Work items:

- In `copyEmbeddedJSVerbs`, detect TypeScript-enabled sources.
- Compile TypeScript entries to generated `.js` files under the existing `xgoja_embed/jsverbs/<id>/...` root.
- Preserve non-TypeScript JavaScript files unchanged.
- Rewrite emitted runtime source extensions to `.js` for precompiled sources.
- Add tests for generated output paths and runtime spec JSON.

### Phase 7: Documentation and examples

Files to add or update:

- `cmd/xgoja/doc/15-tutorial-typescript-jsverbs.md`
- `examples/xgoja/15-typescript-jsverbs/README.md`
- `examples/xgoja/15-typescript-jsverbs/xgoja.yaml`
- `examples/xgoja/15-typescript-jsverbs/verbs/site.ts`
- `examples/xgoja/15-typescript-jsverbs/tsconfig.json`

Example TypeScript verb:

```ts
__package__({ name: "sites" })
__verb__("demo", { name: "demo", output: "text" })

import { message } from "./message"

export function demo(): void {
  const express = require("express")
  const app = express.app()
  app.get("/", (_req: unknown, res: any) => res.send(message("xgoja")))
  app.get("/healthz", (_req: unknown, res: any) => res.json({ ok: true }))
}
```

## Testing strategy

### Unit tests

- `pkg/tsscript`: transform, bundle, diagnostics, loader selection, externals.
- `pkg/jsverbs`: scan transformed TypeScript, invoke compiled TypeScript, preserve overlay capture.
- `cmd/xgoja/internal/buildspec`: validation for `typescript` fields.
- `pkg/xgoja/dtsgen`: no behavior change expected; existing tests should continue passing.

### Integration tests

- Build a generated xgoja binary with filesystem-backed TypeScript jsverbs.
- Run `types` and verify declarations still match selected module aliases.
- Serve a TypeScript HTTP jsverb with `--hot-reload`.
- Modify a helper `.ts` file and verify a reload occurs.
- Introduce a syntax error and verify the status endpoint reports `LastError` while the old version stays active.

### Commands for local validation

```bash
cd /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja
go test ./pkg/tsscript ./pkg/jsverbs ./pkg/xgoja/app ./pkg/xgoja/providers/http ./cmd/xgoja/internal/buildspec ./cmd/xgoja/internal/generate -count=1
go test ./... -count=1
```

For example project validation:

```bash
make -C examples/xgoja/15-typescript-jsverbs build
./examples/xgoja/15-typescript-jsverbs/dist/typescript-jsverbs types > /tmp/xgoja-modules.d.ts
./examples/xgoja/15-typescript-jsverbs/dist/typescript-jsverbs serve sites demo \
  --http-listen 127.0.0.1:8787 \
  --hot-reload \
  --hot-reload-smoke-path /healthz \
  --hot-reload-watch-root examples/xgoja/15-typescript-jsverbs/verbs
```

Then smoke-test:

```bash
curl -fsS http://127.0.0.1:8787/healthz
curl -fsS http://127.0.0.1:8787/__xgoja/status
```

## Risks and mitigations

| Risk | Why it matters | Mitigation |
| --- | --- | --- |
| Source map diagnostics are confusing | esbuild errors may point at generated virtual sources. | Always set `Sourcefile`, `ResolveDir`, and include original file path in wrapped errors. Add source maps later if needed. |
| Bundling hides functions from overlay | esbuild wraps modules when bundling. | Append overlay before bundling and test capture explicitly. |
| Native module aliases are bundled incorrectly | esbuild may try to resolve `require("express")` from npm instead of leaving it for xgoja. | Mark selected xgoja aliases as `External`. Add tests. |
| Runtime esbuild increases binary size | Generated binaries that compile `.ts` at runtime include esbuild. | Precompile embedded sources at generation time. Runtime compile only for filesystem/hot reload mode. |
| TypeScript type errors still run | esbuild strips types without checking. | Document and example `tsc --noEmit`; optional `checkCommand` can be future work. |
| TSX support needs JSX settings | `.tsx` requires JSX mode decisions. | Default to esbuild's transform behavior; expose JSX options only when first real TSX example requires it. |

## Open questions

1. Should `typescript.enabled` be inferred from `.ts` extensions, or always explicit? Recommendation: explicit for the first release.
2. Should `xgoja run file.ts` use `FormatIIFE` or `FormatCommonJS`? Recommendation: start with IIFE, then use tests with external `require()` to decide.
3. Should embedded TypeScript compilation happen by default or behind a separate `xgoja generate --compile-typescript` flag? Recommendation: default when `typescript.enabled: true` and `embed: true`.
4. Should jsverbs eventually parse TypeScript directly with tree-sitter-typescript for better line mapping? Recommendation: not in the first implementation.

## File reference map

- `/home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/internal/buildspec/build_spec.go`: build-time xgoja YAML schema and `JSVerbSourceSpec`.
- `/home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/pkg/xgoja/app/runtime_spec.go`: runtime spec embedded into generated binaries.
- `/home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/pkg/xgoja/app/run.go`: `xgoja run` script execution path.
- `/home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/pkg/xgoja/app/root.go`: jsverb source scanning and command mounting.
- `/home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/pkg/jsverbs/scan.go`: jsverb filesystem scanning and JavaScript parsing.
- `/home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/pkg/jsverbs/runtime.go`: jsverb runtime loader and overlay injection.
- `/home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/pkg/xgoja/dtsgen/dtsgen.go`: existing declaration generation for selected xgoja modules.
- `/home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/pkg/xgoja/hotreload/manager.go`: blue/green hot reload manager.
- `/home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/pkg/xgoja/hotreload/watch.go`: polling watch loop and extension filters.
- `/home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/pkg/xgoja/providers/http/serve.go`: HTTP provider command and hot reload integration.
- `/home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/doc/14-tutorial-typescript-declarations.md`: current user-facing declaration workflow.
- `/home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/ttmp/2026/06/10/XGOJA-TS-001--typescript-support-for-go-go-goja-xgoja-and-hot-reload/sources/local/01-goja-typescript-esbuild-note.md`: imported esbuild/goja TypeScript source note.
