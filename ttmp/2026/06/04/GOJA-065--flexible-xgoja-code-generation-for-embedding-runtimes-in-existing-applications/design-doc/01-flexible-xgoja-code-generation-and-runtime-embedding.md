---
Title: Flexible xgoja Code Generation and Runtime Embedding
Ticket: GOJA-065
Status: active
Topics:
    - goja
    - xgoja
    - code-generation
    - application-integration
    - javascript
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../geppetto/pkg/inference/tools/scopedjs/runtime.go
      Note: Scopedjs runtime construction model used as downstream integration reference
    - Path: ../../../../../../../pinocchio/cmd/pinocchio/cmds/js.go
      Note: Pinocchio hand-written JavaScript runtime integration that package generation could simplify
    - Path: cmd/xgoja/cmd_generate.go
      Note: xgoja generate CLI for package source generation
    - Path: cmd/xgoja/doc/13-tutorial-generated-runtime-package.md
      Note: Embedded xgoja help tutorial for generated runtime packages
    - Path: cmd/xgoja/internal/buildspec/build_spec.go
      Note: Buildspec target schema that needs package/template generation fields
    - Path: cmd/xgoja/internal/generate/generate.go
      Note: Current hardcoded generated workspace writer that assumes go.mod/main.go/xgoja.gen.json
    - Path: cmd/xgoja/internal/generate/templates.go
      Note: Current template data construction
    - Path: cmd/xgoja/internal/generate/templates/main.go.tmpl
      Note: Current binary-oriented generated main template and target-mode branch points
    - Path: cmd/xgoja/internal/generate/templates/runtime_package.go.tmpl
      Note: Generated runtime package API template
    - Path: examples/xgoja/14-generated-runtime-package/README.md
      Note: Runnable generated runtime package example
    - Path: pkg/xgoja/app/factory.go
      Note: Reusable runtime factory for xgoja profiles
    - Path: pkg/xgoja/app/host.go
      Note: Reusable xgoja host and command attachment API
ExternalSources: []
Summary: Design and implementation guide for making xgoja generate reusable runtime integration packages, source fragments, and custom-template outputs instead of only finished main.go binaries.
LastUpdated: 2026-06-04T23:26:00-04:00
WhatFor: Use when implementing flexible xgoja code generation for embedding configured runtimes into existing applications such as Pinocchio or Geppetto scopedjs tools.
WhenToUse: Before changing cmd/xgoja/internal/generate, buildspec target modes, pkg/xgoja/app runtime APIs, or downstream integration patterns.
---



# Flexible xgoja Code Generation and Runtime Embedding

## Executive summary

xgoja currently treats `xgoja.yaml` as an input to **build a complete generated Go program**. The generator writes a temporary module with `go.mod`, `main.go`, embedded resource directories, and `xgoja.gen.json`, then compiles a binary. This works well for standalone CLIs, generated Cobra attachment, and adapter-owned roots. It is less useful when an existing application wants to compile xgoja's declarative runtime wiring into its own package and call JavaScript from Go.

The requested use case is an application like Pinocchio or a Geppetto tool host. Pinocchio already has application-specific runtime bootstrapping for JavaScript. Geppetto's `scopedjs` package already models a reusable JavaScript tool runtime with modules, globals, bootstrap files, manifests, evaluation options, and prebuilt/lazy tool registration. The gap is that xgoja cannot yet produce a reusable Go package such as `internal/xgojaruntime` that exposes typed APIs like `NewRuntime`, `NewFactory`, `Eval`, `RunScript`, `AttachCommands`, `EmbeddedSpecJSON`, and `Manifest`. Instead, it emits a `package main` with a fixed `func main()` and compiles it immediately.

This design recommends a staged implementation:

1. **Extract a public runtime bundle API** from `pkg/xgoja/app` so callers can construct providers, embedded filesystems, `app.Host`, and `app.RuntimeFactory` without depending on generated `main.go` details.
2. **Add a new generated target kind `package`** that writes a reusable `xgoja_runtime.gen.go` file into a caller-selected package instead of a full `main.go` program.
3. **Add a new CLI command `xgoja generate`** for source generation without `go build`; keep `xgoja build` for binary builds.
4. **Add optional custom template support** as an advanced mode, with a stable template data contract and generated helper functions so users can customize shape without reimplementing provider imports, embedded resource declarations, or spec rewriting.
5. **Document integration adapters** for Pinocchio/scopedjs that consume the generated package through interfaces, not by making go-go-goja import Geppetto.

The north star is: `xgoja.yaml` should describe runtime composition once, and xgoja should support multiple outputs:

```text
xgoja.yaml
  -> standalone binary       (today)
  -> attach to Cobra root    (today)
  -> adapter root binary     (today)
  -> reusable Go package     (proposed)
  -> source fragments        (proposed)
  -> custom template output  (proposed, advanced)
```

## Problem statement and scope

### User-visible problem

A host application may want xgoja's provider/runtime selection without delegating its whole command tree to xgoja. For example:

- Pinocchio may want to load a runtime profile from `xgoja.yaml`, then run arbitrary scripts from its own `pinocchio js` command.
- A Geppetto tool host may want a scoped JavaScript tool backed by an xgoja-defined runtime profile.
- A web server may want to create one runtime per request, per tenant, or per workspace using the same xgoja runtime spec.
- A TUI may want to evaluate snippets using xgoja modules but keep its own Bubble Tea lifecycle.

Today, the practical choices are awkward:

1. Use `target.kind: adapter` or `target.kind: cobra`, but still build a generated binary.
2. Manually copy the generated `main.go` patterns into the host app.
3. Manually recreate the provider registry and runtime factory in application code.
4. Use command providers, but only when the integration is naturally a generated command.

The user asked specifically for a way to "compile the .go file" or otherwise integrate the xgoja-specified runtime inside an existing app so callers can run JS scripts easily, plus the possibility for callers to provide a custom Go template for generation.

### In scope

This design covers:

- current xgoja generation architecture;
- current runtime APIs that can be reused;
- integration needs from Pinocchio and Geppetto `scopedjs`;
- proposed target modes and CLI changes;
- public API sketches for generated runtime packages;
- custom template design;
- implementation phases and tests.

### Out of scope for the first implementation

The first implementation should not try to solve every downstream runtime policy. In particular, it should not immediately implement:

- a Geppetto dependency inside go-go-goja;
- all possible scopedjs adapters;
- a full plugin system for arbitrary template functions;
- runtime loading of Go providers from YAML without compile-time imports;
- dynamic Go plugins.

The reliable extension boundary remains source generation plus normal Go compilation.

## Current-state architecture

### Build-time schema

`cmd/xgoja/internal/buildspec/build_spec.go` defines `BuildSpec`, the YAML schema consumed by xgoja. The important fields are `Target`, `Packages`, `Runtimes`, `Commands`, `CommandProviders`, `JSVerbs`, `Help`, and `Assets` (`cmd/xgoja/internal/buildspec/build_spec.go:16-30`). `TargetSpec` currently has `kind`, `import`, `version`, `root`, and `output` (`cmd/xgoja/internal/buildspec/build_spec.go:46-52`).

The schema is already declarative. Runtime profiles list provider module instances, and each module instance is `(package, name, as, config)` rather than Go code (`cmd/xgoja/internal/buildspec/build_spec.go:62-76`). This is exactly the right foundation for reusable runtime generation: the runtime composition is already data.

### Target validation

The validator currently accepts only three target kinds: `xgoja`, `adapter`, and `cobra` (`cmd/xgoja/internal/buildspec/validate.go:125-132`). It requires `target.output` for all target kinds, `target.import` for `adapter` and `cobra`, and `target.root` for `cobra` (`cmd/xgoja/internal/buildspec/validate.go:133-150`).

This is where new target kinds such as `package`, `source`, or `template` would first need validation support.

### Generated workspace

`cmd/xgoja/internal/generate/generate.go` writes the generated build workspace. `WriteAll` creates the output directory, copies embedded jsverbs, help sources, and assets, and then writes exactly three generated files: `go.mod`, `main.go`, and `xgoja.gen.json` (`cmd/xgoja/internal/generate/generate.go:23-52`).

That hardcoded file map is a central limitation. It assumes the product is a temporary module that will be built as a binary. A reusable package target needs a more general "render outputs" pipeline, where output file names and package names can vary.

### Embedded resource rewriting

`RenderEmbeddedSpec` converts build-time paths into runtime paths. Embedded jsverbs become `xgoja_embed/jsverbs/<id>`, embedded help becomes `xgoja_embed/help/<id>`, and embedded assets become `xgoja_embed/assets/<id>` (`cmd/xgoja/internal/generate/main.go:59-91`, `cmd/xgoja/internal/generate/main.go:130-207`).

This path rewriting is valuable and should be reused by all future generation modes. A package target should not invent different embedded root conventions.

### Current main template

`cmd/xgoja/internal/generate/templates/main.go.tmpl` is currently the only Go source template. It always emits `package main`, imports `app` and `providerapi`, imports provider packages, embeds optional resources, registers providers, and then either:

- constructs a standalone xgoja root with `app.NewRootCommand`,
- imports a target Cobra root and attaches xgoja commands, or
- imports an adapter and calls `target.Build(context.Background(), host)`.

The key template lines are:

- provider registry setup at `templates/main.go.tmpl:40-44`;
- adapter mode at `templates/main.go.tmpl:45-49`;
- cobra mode at `templates/main.go.tmpl:50-54`;
- pure xgoja mode at `templates/main.go.tmpl:55-58`;
- `decodeSpec()` helper at `templates/main.go.tmpl:65-69`.

This is effective for binaries, but it hides reusable pieces inside `func main()`.

### Template data construction

`cmd/xgoja/internal/generate/templates.go` builds `mainTemplateData`. It calculates provider import aliases, embedded resource booleans, target kind, target import, root construction text, and host construction text (`cmd/xgoja/internal/generate/templates.go:17-31`, `cmd/xgoja/internal/generate/templates.go:55-107`). It formats the rendered template with `go/format` (`cmd/xgoja/internal/generate/templates.go:39-52`).

Two things matter for flexible codegen:

1. There is already a useful data model for rendering.
2. Some fields are pre-rendered Go snippets (`HostConstruction`, `RootConstruction`), which is convenient but less stable for custom templates than structured fields.

### Runtime host and factory APIs

At runtime, `pkg/xgoja/app.Host` holds the provider registry, runtime spec, runtime factory, embedded filesystems, host services, output writer, and Cobra middleware configuration (`pkg/xgoja/app/host.go:12-22`). `NewHostWithOptions` builds the asset store and `RuntimeFactory` (`pkg/xgoja/app/host.go:36-52`). `AttachDefaultCommands` installs xgoja commands onto a Cobra root (`pkg/xgoja/app/host.go:55-76`).

`pkg/xgoja/app.RuntimeFactory` is the real reusable runtime construction API. It resolves a profile, maps provider modules to runtime module registrars, applies config sections and host service contributions, builds an engine runtime factory, and creates an `engine.Runtime` (`pkg/xgoja/app/factory.go:17-21`, `pkg/xgoja/app/factory.go:62-140`).

This means a host app does not need generated commands to create a runtime. It needs an easy way to compile and call:

```go
registry := providerapi.NewProviderRegistry()
provider.Register(registry)
runtimeSpec := decode embedded JSON
host := app.NewHostWithOptions(registry, runtimeSpec, embedded FS options)
rt, err := host.Factory.NewRuntime(ctx, "main")
```

The current generated `main.go` already does most of this, but only inside a private executable.

### Current adapter and cobra modes

The user guide documents existing target modes. Pure `xgoja` creates a standalone root. `cobra` imports an existing target package and calls a root constructor. `adapter` imports a package with `Build(context.Context, *app.Host) (*cobra.Command, error)` (`cmd/xgoja/doc/02-user-guide.md:260-292`).

The test adapter shows the pattern: create an application root, add app-specific commands, call `host.AttachDefaultCommands(root)`, and return the root (`pkg/xgoja/testadapter/adapter.go:11-25`). The Cobra fixture creates a root with a target-owned command, and generated xgoja attaches its commands later (`pkg/xgoja/testcobra/root.go:9-20`). Generated tests verify both target modes build and execute (`cmd/xgoja/internal/generate/generate_test.go:583-592`).

These modes partially solve application integration, but only at the CLI-root level and still through a generated binary. They do not produce a library package that an existing app can import.

## Downstream integration evidence: Pinocchio and Geppetto scopedjs

### Geppetto scopedjs model

`geppetto/pkg/inference/tools/scopedjs` is a reusable JavaScript tool environment. Its schema defines:

- `EnvironmentSpec[Scope, Meta]` with runtime label, tool definition, default eval options, manifest description, and a `Configure` callback (`geppetto/pkg/inference/tools/scopedjs/schema.go:47-53`);
- `BuildResult` with `Runtime`, `Executor`, `Meta`, `Manifest`, and `Cleanup` (`geppetto/pkg/inference/tools/scopedjs/schema.go:55-61`);
- `Builder`, which records modules, native modules, globals, initializers, bootstrap entries, and a manifest (`geppetto/pkg/inference/tools/scopedjs/schema.go:127-134`).

`BuildRuntime` uses the builder callback to construct an engine runtime factory, create a runtime, load bootstrap code, and return a `BuildResult` with a `RuntimeExecutor` and cleanup function (`geppetto/pkg/inference/tools/scopedjs/runtime.go:50-94`). `RunEval` evaluates user code through the runtime owner, captures console output, handles promises, applies timeouts, and returns structured `EvalOutput` (`geppetto/pkg/inference/tools/scopedjs/eval.go:27-69`, `geppetto/pkg/inference/tools/scopedjs/eval.go:116-172`).

`RegisterPrebuilt` and `NewLazyRegistrar` show two useful runtime ownership models: reuse one prebuilt runtime, or build a fresh runtime per tool call (`geppetto/pkg/inference/tools/scopedjs/tool.go:11-39`, `geppetto/pkg/inference/tools/scopedjs/tool.go:41-88`).

This maps directly to xgoja integration. xgoja can provide the configured runtime factory; scopedjs can provide the tool schema, console capture, timeout behavior, and registry integration.

### Pinocchio scopedjs example

`pinocchio/cmd/examples/scopedjs-tui-demo/environment.go` shows a concrete application-specific scoped JS environment. It defines custom modules (`webserver`, `obsidian`), a scope object with workspace data, a manifest, and a `configureDemoRuntime` callback that adds native modules, globals, and bootstrap helpers (`pinocchio/cmd/examples/scopedjs-tui-demo/environment.go:40-63`, `pinocchio/cmd/examples/scopedjs-tui-demo/environment.go:65-96`, `pinocchio/cmd/examples/scopedjs-tui-demo/environment.go:98-145`, `pinocchio/cmd/examples/scopedjs-tui-demo/environment.go:145-195`).

This example demonstrates what xgoja should not erase: host applications still need to bind scope-specific globals and modules. Flexible xgoja codegen should provide the base runtime profile and module registry, then let applications add per-scope bindings around it.

### Pinocchio `js` command

Pinocchio's `cmd/pinocchio/cmds/js.go` manually builds a JavaScript runtime for its `pinocchio js` command. It resolves profile/bootstrap config, tool registries, middleware registries, turn stores, and then calls `newPinocchioJSRuntime` (`pinocchio/cmd/pinocchio/cmds/js.go:141-220`). That runtime constructor creates goja require options, builds a go-go-goja runtime factory, creates a runtime, manually registers Geppetto and Pinocchio modules, enables require, and installs console/helpers (`pinocchio/cmd/pinocchio/cmds/js.go:285-369`).

This is exactly the class of code xgoja package generation could simplify. A Pinocchio runtime package generated from `xgoja.yaml` could own provider registration, selected modules, embedded assets, and base runtime construction, while Pinocchio keeps app-specific profile resolution, turn stores, and console behavior.

## Gap analysis

### What already exists

- Declarative runtime profiles in `xgoja.yaml`.
- Provider package registration and module selection.
- Runtime construction APIs in `pkg/xgoja/app`.
- Embedded resource copying and spec path rewriting.
- Generated target modes for pure xgoja, Cobra attachment, and adapter-owned roots.
- A single `main.go` template with useful data preparation.
- Downstream runtime tool models in scopedjs that can consume a `*engine.Runtime`.

### What is missing

1. **No generated package target.** xgoja cannot emit `package myruntime` with exported runtime helpers.
2. **No source-generation command.** `xgoja build` always generates a temporary module and then compiles, unless using dry-run/keep-work as a workaround.
3. **No public codegen package.** The generator lives under `cmd/xgoja/internal/generate`, so other Go code cannot call it.
4. **No stable custom template contract.** The only template is internal and tailored to `package main`.
5. **No first-class runtime bundle abstraction.** `app.Host` is close, but generated code still hand-wires provider registration and embedded FS globals.
6. **No documented scopedjs adapter pattern.** Integrators must manually bridge xgoja runtime factories into scopedjs prebuilt/lazy runtime execution.

## Design goals

A good solution should:

- preserve existing `xgoja build` behavior;
- make generated library integration easy without copy/pasting generated `main.go`;
- keep provider imports compile-time explicit;
- support embedded jsverbs/help/assets in packages as well as binaries;
- let host apps control lifecycle, concurrency, and evaluation policy;
- provide stable APIs before exposing custom templates;
- allow advanced users to replace only the outer template, not low-level path rewriting and provider alias logic;
- be testable through generated source, not only unit tests.

## Proposed architecture

### Conceptual model

Split xgoja output into three layers:

```text
Layer 1: Runtime bundle data
  - provider imports and registration
  - embedded runtime spec JSON
  - embedded jsverbs/help/assets filesystems
  - helpers to build app.Host and app.RuntimeFactory

Layer 2: Integration API
  - NewHost
  - NewRuntimeFactory
  - NewRuntime
  - Eval / RunScript helpers
  - AttachDefaultCommands
  - optional jsverb source scanning helpers

Layer 3: Product shell
  - standalone main.go
  - existing cobra/adapter binary wrappers
  - host-owned package import
  - custom template output
```

Current xgoja combines all three in `main.go`. The proposed change makes layer 1 and 2 available as generated package output, while keeping layer 3 for binaries.

### New target kind: `package`

Add a target mode that emits a reusable Go package instead of a binary:

```yaml
target:
  kind: package
  package: xgojaruntime
  output: ./internal/xgojaruntime
```

Generated output:

```text
internal/xgojaruntime/
  xgoja_runtime.gen.go
  xgoja_embed/jsverbs/...
  xgoja_embed/help/...
  xgoja_embed/assets/...
```

Unlike `xgoja build`, package generation should not create a standalone `go.mod`. It writes files into an existing module. Provider imports are regular imports in the generated `.go` file, so the host module's `go.mod` must already require or be able to resolve those packages.

### Generated package API sketch

A generated package should expose a small, boring API. Example generated file:

```go
// Code generated by xgoja; DO NOT EDIT.
package xgojaruntime

import (
    "context"
    "encoding/json"
    "io"
    "io/fs"

    "github.com/dop251/goja_nodejs/require"
    "github.com/go-go-golems/glazed/pkg/cmds/values"
    "github.com/go-go-golems/go-go-goja/pkg/engine"
    "github.com/go-go-golems/go-go-goja/pkg/xgoja/app"
    "github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

const EmbeddedSpecJSON = `...`

type Options struct {
    Out             io.Writer
    MiddlewaresFunc cli.CobraMiddlewaresFunc
    HostServices    providerapi.HostServices // optional override/extension
}

type Bundle struct {
    Providers   *providerapi.ProviderRegistry
    RuntimeSpec *app.RuntimeSpec
    Host        *app.Host
}

func NewBundle(opts Options) (*Bundle, error) {
    registry := providerapi.NewProviderRegistry()
    if err := RegisterProviders(registry); err != nil { return nil, err }
    spec, err := DecodeSpec()
    if err != nil { return nil, err }
    host := app.NewHostWithOptions(registry, spec, app.HostOptions{
        EmbeddedJSVerbs: embeddedJSVerbsOrNil(),
        EmbeddedHelp:    embeddedHelpOrNil(),
        EmbeddedAssets:  embeddedAssetsOrNil(),
        Out:             opts.Out,
        MiddlewaresFunc: opts.MiddlewaresFunc,
    })
    return &Bundle{Providers: registry, RuntimeSpec: spec, Host: host}, nil
}

func RegisterProviders(registry *providerapi.ProviderRegistry) error {
    if registry == nil { return fmt.Errorf("provider registry is nil") }
    if err := core.Register(registry); err != nil { return err }
    if err := geppetto.Register(registry); err != nil { return err }
    return nil
}

func DecodeSpec() (*app.RuntimeSpec, error) {
    spec := &app.RuntimeSpec{}
    if err := json.Unmarshal([]byte(EmbeddedSpecJSON), spec); err != nil { return nil, err }
    return spec, nil
}

func (b *Bundle) NewRuntime(ctx context.Context, profile string, vals *values.Values, opts ...require.Option) (*engine.Runtime, error) {
    return b.Host.Factory.NewRuntimeFromSections(ctx, profile, vals, opts...)
}

func (b *Bundle) AttachDefaultCommands(root *cobra.Command) {
    b.Host.AttachDefaultCommands(root)
}
```

The first version does not need every helper. The minimum useful API is:

- `EmbeddedSpecJSON`;
- `DecodeSpec()`;
- `RegisterProviders(registry)`;
- `NewBundle(opts)`;
- `(*Bundle).NewRuntime(...)`;
- `(*Bundle).AttachDefaultCommands(root)`.

### New CLI command: `xgoja generate`

Add a source-generation command separate from `build`:

```bash
xgoja generate -f xgoja.yaml --output ./internal/xgojaruntime --package xgojaruntime
```

Responsibilities:

1. Load and validate `xgoja.yaml`.
2. Copy embedded jsverbs/help/assets into the output package directory.
3. Render one or more `.gen.go` files.
4. Optionally run `gofmt`.
5. Do **not** run `go mod tidy` or `go build` unless explicitly requested.

Possible flags:

```text
--file, -f          xgoja.yaml path
--output, -o        output directory or file depending on target kind
--package           generated Go package name override
--template          custom Go template path
--template-data     print JSON template data and exit
--clean             remove previously generated xgoja_embed and *.gen.go files first
--dry-run           validate and describe planned outputs
```

`xgoja build` can keep using the same renderer internally, but should continue to produce a temporary module and binary.

### Source fragments mode

Some applications may not want a full generated package. They may want fragments:

```yaml
target:
  kind: source
  package: jsruntime
  output: ./internal/jsruntime
  files:
    providers: providers.gen.go
    spec: spec.gen.go
    assets: embed.gen.go
```

This is lower priority than `package`, but worth designing because it answers the "not just finished main.go" request in a composable way. Fragment outputs could be:

- `spec.gen.go` — `EmbeddedSpecJSON`, `DecodeSpec`;
- `providers.gen.go` — provider imports and `RegisterProviders`;
- `embed.gen.go` — `//go:embed` resource variables;
- `bundle.gen.go` — `NewBundle` and helpers.

A package target can simply be the default bundle of these fragments in one file.

### Custom template support

Custom templates are useful, but they are dangerous if introduced before a stable data contract. The implementation should support them after, or alongside, generated package helpers.

Recommended buildspec shape:

```yaml
target:
  kind: template
  package: myruntime
  output: ./internal/myruntime/xgoja_custom.gen.go
  template: ./xgoja.runtime.go.tmpl
```

CLI override:

```bash
xgoja generate -f xgoja.yaml \
  --template ./templates/runtime.go.tmpl \
  --output ./internal/jsruntime/runtime.gen.go \
  --package jsruntime
```

Template data should be structured, not just pre-rendered snippets:

```go
type TemplateData struct {
    PackageName string
    SpecJSON string
    Providers []ProviderImport
    Embedded EmbeddedData
    Target TargetData
    RuntimeProfiles []RuntimeProfileData
    GeneratedBy string
}

type ProviderImport struct {
    ID string
    Alias string
    ImportPath string
    Register string
}

type EmbeddedData struct {
    HasJSVerbs bool
    JSVerbsVar string
    JSVerbsPattern string
    HasHelp bool
    HelpVar string
    HelpPattern string
    HasAssets bool
    AssetsVar string
    AssetsPattern string
}
```

Expose helper functions:

```text
quote        -> strconv.Quote
rawString    -> safe raw string literal
goIdent      -> sanitized Go identifier
json         -> JSON encode
```

Rules:

- xgoja still owns embedded resource copying and runtime spec rewriting.
- custom templates only control Go source shape.
- generated output is always passed through `go/format`.
- template rendering errors should include the template path and unformatted output when formatting fails.

### Public codegen package

Because `cmd/xgoja/internal/generate` is internal, existing applications cannot call it. Add a public-ish package:

```text
pkg/xgoja/codegen
```

Public API sketch:

```go
type OutputKind string
const (
    OutputMain    OutputKind = "main"
    OutputPackage OutputKind = "package"
    OutputTemplate OutputKind = "template"
)

type Options struct {
    Kind OutputKind
    PackageName string
    OutputDir string
    OutputFile string
    TemplatePath string
    XGojaModuleVersion string
    XGojaReplace string
    Clean bool
}

type Plan struct {
    Files []GeneratedFile
    EmbeddedCopies []CopyPlan
}

func Plan(ctx context.Context, spec *buildspec.BuildSpec, opts Options) (*Plan, error)
func Write(ctx context.Context, spec *buildspec.BuildSpec, opts Options) (*Plan, error)
func RenderTemplate(spec *buildspec.BuildSpec, opts Options) ([]byte, error)
```

The existing internal generator can move here, or `cmd/xgoja/internal/generate` can become a thin wrapper. Be careful: `buildspec` is currently also internal. If `codegen` is public, the buildspec schema may need to move to `pkg/xgoja/buildspec` or a public `pkg/xgoja/spec` package. A staged migration can first add `pkg/xgoja/codegen` that accepts `*app.RuntimeSpec` plus a provider list, then later publicize buildspec. However, for generation from YAML, a public buildspec package is cleaner.

## Integration patterns

### Pattern A: Existing app imports generated runtime package

Generated package:

```go
// internal/xgojaruntime/xgoja_runtime.gen.go
bundle, err := xgojaruntime.NewBundle(xgojaruntime.Options{Out: os.Stdout})
rt, err := bundle.NewRuntime(ctx, "main", values.New())
```

Pinocchio command:

```go
func runScript(ctx context.Context, scriptPath string, vals *values.Values) error {
    bundle, err := xgojaruntime.NewBundle(xgojaruntime.Options{})
    if err != nil { return err }

    requireOpt, err := engine.RequireOptionWithModuleRootsFromScript(scriptPath, engine.DefaultModuleRootsOptions())
    if err != nil { return err }

    rt, err := bundle.NewRuntime(ctx, "pinocchio", vals, requireOpt)
    if err != nil { return err }
    defer rt.Close(context.Background())

    return RunScriptFile(ctx, rt, scriptPath)
}
```

### Pattern B: Host adds xgoja commands to its own Cobra root

```go
func NewRootCommand() *cobra.Command {
    root := &cobra.Command{Use: "pinocchio"}
    bundle, err := xgojaruntime.NewBundle(xgojaruntime.Options{})
    if err != nil {
        root.PersistentPreRunE = func(*cobra.Command, []string) error { return err }
        return root
    }
    bundle.AttachDefaultCommands(root)
    root.AddCommand(NewPinocchioCommands()...)
    return root
}
```

This is similar to existing `target.kind: cobra`, but the app owns the final build and import direction.

### Pattern C: Geppetto scopedjs prebuilt runtime

Do not make go-go-goja import Geppetto. Instead, let a Pinocchio or Geppetto-side adapter consume the generated package:

```go
func BuildXGojaScopedJS(ctx context.Context, vals *values.Values) (*scopedjs.BuildResult[Meta], error) {
    bundle, err := xgojaruntime.NewBundle(xgojaruntime.Options{})
    if err != nil { return nil, err }

    rt, err := bundle.NewRuntime(ctx, "main", vals)
    if err != nil { return nil, err }

    manifest := scopedjs.EnvironmentManifest{
        Modules: xgojaManifestModules(bundle.RuntimeSpec),
        Globals: []scopedjs.GlobalDoc{{Name: "input", Type: "object"}},
    }

    return &scopedjs.BuildResult[Meta]{
        Runtime: rt,
        Executor: scopedjs.NewRuntimeExecutor(rt),
        Manifest: manifest,
        Cleanup: func() error { return rt.Close(context.Background()) },
    }, nil
}
```

Then register it:

```go
handle, err := BuildXGojaScopedJS(ctx, vals)
if err != nil { return err }
return scopedjs.RegisterPrebuilt(registry, spec, handle, scopedjs.EvalOptionOverrides{})
```

### Pattern D: Scoped runtime with host-provided globals

The generated xgoja runtime package should allow host applications to add runtime options or post-create initialization:

```go
rt, err := bundle.NewRuntime(ctx, "main", vals)
if err != nil { return err }
_, err = rt.Owner.Call(ctx, "pinocchio.scope", func(_ context.Context, vm *goja.Runtime) (any, error) {
    if err := vm.Set("workspaceRoot", scope.Root); err != nil { return nil, err }
    if err := vm.Set("db", buildDBGlobal(scope)); err != nil { return nil, err }
    return nil, nil
})
```

A more formal v2 API could accept `[]engine.RuntimeInitializer` as an option, but the first version can document owner-call initialization.

## Decision records

### Decision: Add `package` generation before custom templates

- **Context:** Users want flexible integration and custom templates, but custom templates are hard to support safely without a stable data model.
- **Options considered:** custom template first; package target first; only document adapter mode.
- **Decision:** Implement `target.kind: package` first, then custom templates on the same template data contract.
- **Rationale:** A generated package covers most real integrations and defines the helper functions custom templates can reuse.
- **Consequences:** The first milestone is less flexible than arbitrary templates but much easier to test and document.
- **Status:** proposed.

### Decision: Keep provider registration compile-time explicit

- **Context:** xgoja provider packages are normal Go imports. Dynamic Go plugin loading is brittle and not required.
- **Options considered:** runtime plugin loading; generated imports; reflection over imported packages.
- **Decision:** Continue generating explicit imports and `RegisterProviders` calls.
- **Rationale:** This matches current xgoja design, keeps builds reproducible, and works with static binaries.
- **Consequences:** Host modules must keep `go.mod` dependencies for provider imports.
- **Status:** proposed.

### Decision: Generate reusable runtime APIs instead of only command providers

- **Context:** Command providers solve generated CLI extension, but Pinocchio/scopedjs need to call JavaScript inside their own lifecycle.
- **Options considered:** force everything through command providers; expose generated package APIs; require manual runtime construction.
- **Decision:** Expose generated package APIs for `NewBundle`, `NewRuntime`, and `AttachDefaultCommands`.
- **Rationale:** The same runtime profile can then serve CLIs, tools, TUIs, web handlers, and tests.
- **Consequences:** The generated API surface must be versioned and kept stable.
- **Status:** proposed.

### Decision: Do not add a direct Geppetto dependency to go-go-goja

- **Context:** scopedjs is a motivating example, but go-go-goja should remain lower-level.
- **Options considered:** add `pkg/xgoja/scopedjs` importing Geppetto; provide generic runtime package; document adapter on Geppetto/Pinocchio side.
- **Decision:** Keep core generic and put scopedjs adapters downstream or in a separate bridge module.
- **Rationale:** Avoid dependency direction issues and keep xgoja usable without Geppetto.
- **Consequences:** The design must provide enough generic hooks for downstream adapters.
- **Status:** proposed.

### Decision: Add `xgoja generate` rather than overloading `build --dry-run --keep-work`

- **Context:** Existing `build` can leave a generated workspace, but that is a debugging workaround.
- **Options considered:** document `--keep-work`; add `generate`; add `build --emit-source`.
- **Decision:** Add `xgoja generate` for durable source generation.
- **Rationale:** Source generation has different outputs, cleanup rules, and success criteria than binary building.
- **Consequences:** `build` and `generate` should share a renderer/planner to avoid divergence.
- **Status:** proposed.

## Implementation guide

### Phase 1: Refactor generation into a plan/render pipeline

Files to start with:

- `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/cmd/xgoja/internal/generate/generate.go`
- `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/cmd/xgoja/internal/generate/templates.go`
- `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/cmd/xgoja/internal/generate/main.go`

Current `WriteAll` writes a hardcoded map of files. Replace that with a `Plan`:

```go
type GeneratedFile struct {
    RelPath string
    Content []byte
    Mode fs.FileMode
}

type CopyPlan struct {
    Source string
    DestRel string
    Options CopyOptions
}

type Plan struct {
    Files []GeneratedFile
    Copies []CopyPlan
}
```

Then implement:

```go
func PlanMainModule(spec *buildspec.BuildSpec, opts Options) (*Plan, error)
func PlanPackage(spec *buildspec.BuildSpec, opts PackageOptions) (*Plan, error)
func WritePlan(dir string, plan *Plan) error
```

Keep `xgoja build` using `PlanMainModule` so existing behavior stays stable.

### Phase 2: Add package-name and package-target schema

Update `TargetSpec`:

```go
type TargetSpec struct {
    Kind     string `yaml:"kind" json:"kind"`
    Import   string `yaml:"import" json:"import,omitempty"`
    Version  string `yaml:"version" json:"version,omitempty"`
    Root     string `yaml:"root" json:"root,omitempty"`
    Output   string `yaml:"output" json:"output"`
    Package  string `yaml:"package" json:"package,omitempty"`
    Template string `yaml:"template" json:"template,omitempty"`
}
```

Validation rules:

- `xgoja`, `cobra`, `adapter`: preserve existing rules.
- `package`: require `target.output`; default `target.package` from output directory basename if omitted.
- `template`: require `target.output` and `target.template`; package optional if template does not use it.

### Phase 3: Add runtime package template

Create:

```text
cmd/xgoja/internal/generate/templates/runtime_package.go.tmpl
```

The template should:

- use `package {{ .PackageName }}`;
- declare `EmbeddedSpecJSON`;
- declare embed FS variables conditionally;
- expose `RegisterProviders`, `DecodeSpec`, `NewBundle`, and `Bundle.NewRuntime`;
- avoid `func main()`;
- avoid `os.Exit` and `must` helpers;
- return errors instead of printing.

Test with string assertions and compile-generated tests.

### Phase 4: Add `xgoja generate`

Implement `cmd/xgoja/cmd_generate.go` similar to `cmd_build.go`, but without `go mod tidy` or `go build`.

Important behaviors:

- `--dry-run` prints planned files and copied directories.
- `--clean` only removes generated xgoja outputs, not arbitrary directories.
- output paths for embedded assets are deterministic.
- generated `.go` files are gofmt'd.

### Phase 5: Add custom template support

Add a renderer that can parse either embedded templates or a user path:

```go
func renderCustomTemplate(path string, data TemplateData) ([]byte, error) {
    tmpl, err := template.New(filepath.Base(path)).Funcs(templateFuncs()).ParseFiles(path)
    if err != nil { return nil, err }
    var b bytes.Buffer
    if err := tmpl.Execute(&b, data); err != nil { return nil, err }
    return format.Source(b.Bytes())
}
```

Also add:

```bash
xgoja generate -f xgoja.yaml --template-data
```

This prints the JSON form of the template data so users can debug templates without reading Go internals.

### Phase 6: Add integration examples

Add examples:

```text
examples/xgoja/14-generated-runtime-package/
  xgoja.yaml
  internal/xgojaruntime/         # generated by make generate
  cmd/hostapp/main.go            # imports generated package
  scripts/hello.js
  Makefile

examples/xgoja/15-generated-scopedjs-bridge/
  xgoja.yaml
  internal/xgojaruntime/
  cmd/scoped-tool/main.go        # builds scopedjs tool from generated package
  Makefile
```

The scopedjs example can live in a repo that already depends on Geppetto, or be documented as an integration sketch if dependency direction is undesirable.

## Testing strategy

### Unit tests

Add tests for:

- target validation for `package` and `template`;
- package-name defaulting/sanitization;
- package template includes provider registration and embedded FS variables;
- package template does not contain `func main()` or `os.Exit`;
- custom template rendering and formatting errors;
- plan output paths for embedded jsverbs/help/assets.

### Generated compile tests

Extend `cmd/xgoja/internal/generate/generate_test.go` with:

```go
func TestGeneratedPackageTargetBuildsAndCreatesRuntime(t *testing.T) {
    spec := buildableSpec("package", "", "")
    spec.Target.Output = filepath.Join(dir, "internal", "xgojaruntime")
    spec.Target.Package = "xgojaruntime"

    // render package into a temp host module
    // write a small cmd/host/main.go that imports it
    // go run ./cmd/host and assert require("hello").greet("intern") works
}
```

Also test custom templates with a minimal template that emits a package exposing `DecodeSpec`.

### Downstream-style smoke tests

A useful smoke test should imitate Pinocchio/scopedjs without importing Pinocchio:

```go
bundle, err := xgojaruntime.NewBundle(xgojaruntime.Options{})
rt, err := bundle.NewRuntime(ctx, "main", values.New())
executor := scopedjs.NewRuntimeExecutor(rt)
out, err := executor.RunEval(ctx, scopedjs.EvalInput{Code: `const h = require("hello"); return h.greet("intern");`}, scopedjs.DefaultEvalOptions())
```

If adding a real scopedjs example is too expensive due to module dependencies, keep this as documentation and add a lightweight local executor test using `rt.Owner.Call`.

## Migration and compatibility

Existing buildspecs must continue to work unchanged. New features should be additive:

- `target.kind: xgoja` remains default.
- `xgoja build` remains the binary-building command.
- `target.kind: cobra` and `adapter` remain supported.
- Generated `main.go` remains marked `DO NOT EDIT`.
- Package generation uses a new output path and should not overwrite existing application files unless they are known generated files.

The first package-generation implementation can be marked experimental in docs. Once downstream examples prove stable API shape, treat generated package APIs as compatibility commitments.

## Risks and mitigations

| Risk | Why it matters | Mitigation |
| --- | --- | --- |
| Generated package API becomes too large | Hard to maintain compatibility | Start with minimal `NewBundle`, `NewRuntime`, `AttachDefaultCommands` |
| Custom templates depend on unstable internals | Breaks users silently | Publish `TemplateData` JSON contract and `--template-data` |
| Package generation overwrites host files | Dangerous in existing apps | Only write `*.gen.go` and `xgoja_embed`; add `--clean` guardrails |
| Publicizing internal buildspec creates API burden | Internal packages can change freely today | Introduce `pkg/xgoja/codegen` carefully or keep CLI-only first |
| Geppetto bridge creates dependency cycle | go-go-goja should be lower-level | Put scopedjs bridge downstream or in separate module |
| Runtime values/config sections are CLI-shaped | Non-CLI callers may not have `values.Values` | Allow nil values and document helper constructors for defaults/config |

## Alternatives considered

### Alternative A: Use existing adapter mode only

Adapter mode lets a generated binary call `Build(context.Context, *app.Host)` and gives the target package access to xgoja host. It is useful for generated binaries, but it does not let an existing app import a generated runtime package. It also keeps xgoja in charge of the final executable.

### Alternative B: Tell callers to copy generated main.go

This works once, but it is brittle. Callers inherit internal template details, embedded path conventions, provider aliases, and future xgoja changes manually. This is exactly what generated packages should avoid.

### Alternative C: Only add custom templates

Custom templates are flexible, but without a stable helper layer every user must understand provider registration, embedded FS variables, spec rewriting, and runtime factory creation. A standard package target should be the default path; custom templates should be advanced.

### Alternative D: Runtime YAML loading without code generation

A pure runtime loader could parse `xgoja.yaml`, but it cannot import provider packages listed in YAML. xgoja's core design relies on compile-time imports. Runtime YAML loading is useful only after providers are already registered by Go code.

## File reference map

Read these files in order before implementing:

1. `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/cmd/xgoja/internal/buildspec/build_spec.go` — build-time schema and target fields.
2. `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/cmd/xgoja/internal/buildspec/validate.go` — target-kind validation.
3. `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/cmd/xgoja/internal/generate/generate.go` — current hardcoded workspace writer.
4. `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/cmd/xgoja/internal/generate/main.go` — embedded runtime spec rendering and path rewriting.
5. `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/cmd/xgoja/internal/generate/templates.go` — current template data and rendering.
6. `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/cmd/xgoja/internal/generate/templates/main.go.tmpl` — current binary-oriented template.
7. `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/pkg/xgoja/app/host.go` — reusable host object and command attachment.
8. `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/pkg/xgoja/app/factory.go` — reusable runtime factory.
9. `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/geppetto/pkg/inference/tools/scopedjs/runtime.go` — scopedjs runtime construction model.
10. `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/geppetto/pkg/inference/tools/scopedjs/eval.go` — scopedjs evaluation/executor semantics.
11. `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/pinocchio/cmd/pinocchio/cmds/js.go` — current hand-written Pinocchio JavaScript runtime integration.
12. `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/pinocchio/cmd/examples/scopedjs-tui-demo/environment.go` — scoped runtime with host-specific modules/globals.

## Open questions

1. Should `target.kind: package` be generated by `xgoja generate` only, or should `xgoja build` also support building a host app that imports it?
2. Should buildspec and generate packages become public Go APIs, or should source generation remain CLI-only initially?
3. What is the minimum stable generated package API before v1 compatibility expectations apply?
4. Should generated packages expose manifest data derived from provider descriptors, or leave manifests to downstream tools like scopedjs?
5. Should package generation support multiple generated files from the start or begin with one `xgoja_runtime.gen.go`?
6. Should custom templates be allowed from `xgoja.yaml`, CLI flags, or both?

## Implementation update: first package-generation milestone

GOJA-065 now has a first implementation milestone for generated runtime packages. The implementation intentionally covers the most useful subset of the original design: `target.kind: package`, `xgoja generate`, a standard generated runtime package API, generated-package compile tests, a runnable repository example, and embedded xgoja help documentation. Custom templates and source-fragment mode remain future work.

### Implemented command shape

The new CLI command is:

```bash
xgoja generate -f xgoja.yaml \
  --output ./internal/xgojaruntime \
  --package xgojaruntime
```

The command currently accepts only `target.kind: package`. It validates the buildspec, writes generated source into the selected output directory, and exits. It does not create a temporary module, does not write `go.mod`, does not write `main.go`, does not run `go mod tidy`, and does not compile a binary. This keeps source generation separate from binary building.

### Implemented buildspec shape

`TargetSpec` now includes package-generation fields:

```go
type TargetSpec struct {
    Kind     string `yaml:"kind" json:"kind"`
    Import   string `yaml:"import" json:"import,omitempty"`
    Version  string `yaml:"version" json:"version,omitempty"`
    Root     string `yaml:"root" json:"root,omitempty"`
    Output   string `yaml:"output" json:"output"`
    Package  string `yaml:"package" json:"package,omitempty"`
    Template string `yaml:"template" json:"template,omitempty"`
}
```

Validation now accepts `package` as a target kind and validates `target.package` when present. `template` is present in the schema for forward compatibility with the design, but the first implementation does not yet enable `target.kind: template`.

### Implemented generated package API

The generated file is `xgoja_runtime.gen.go`. Its standard API is:

```go
const EmbeddedSpecJSON = `...`

type Options struct {
    Out             io.Writer
    MiddlewaresFunc cli.CobraMiddlewaresFunc
}

type Bundle struct {
    Providers   *providerapi.ProviderRegistry
    RuntimeSpec *app.RuntimeSpec
    Host        *app.Host
}

func RegisterProviders(registry *providerapi.ProviderRegistry) error
func DecodeSpec() (*app.RuntimeSpec, error)
func NewBundle(opts Options) (*Bundle, error)
func (b *Bundle) NewRuntime(ctx context.Context, opts ...require.Option) (*engine.Runtime, error)
func (b *Bundle) NewRuntimeFromSections(ctx context.Context, vals *values.Values, opts ...require.Option) (*engine.Runtime, error)
func (b *Bundle) AttachDefaultCommands(root *cobra.Command)
```

This is enough for a host application to import the generated package, create a runtime, run JavaScript through `rt.Owner.Call`, or attach generated xgoja commands to its own Cobra root.

### Implemented example

The new example is:

```text
examples/xgoja/14-generated-runtime-package
```

It contains an `xgoja.yaml` with `target.kind: package`, a host application under `cmd/host`, a generated `internal/xgojaruntime/xgoja_runtime.gen.go`, and a Makefile smoke target. The host application imports the generated package and evaluates:

```js
require("hello").greet("package host")
```

Expected output:

```text
hello package host
```

### Implemented tests

The implementation adds tests for:

- package target validation;
- invalid package name validation;
- package template API rendering;
- package writer behavior, including not writing `go.mod`, `main.go`, or `xgoja.gen.json`;
- generated-package compile/smoke behavior through a temporary host module;
- root command wiring for `xgoja generate`.

### Remaining future work

The original design's broader ideas are still useful but intentionally deferred:

1. Custom templates with a stable JSON template-data contract.
2. Fragmented generation into separate `spec.gen.go`, `providers.gen.go`, `embed.gen.go`, and `bundle.gen.go` files.
3. A public `pkg/xgoja/codegen` API outside `cmd/xgoja/internal`.
4. A downstream Pinocchio or Geppetto scopedjs adapter example.
5. Optional clean/dry-run plan details beyond the current concise dry-run message.
