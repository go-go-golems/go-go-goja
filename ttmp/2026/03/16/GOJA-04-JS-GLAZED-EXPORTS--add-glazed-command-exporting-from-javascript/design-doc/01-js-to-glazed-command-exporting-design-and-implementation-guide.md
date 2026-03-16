---
Title: JS-to-Glazed command exporting design and implementation guide
Ticket: GOJA-04-JS-GLAZED-EXPORTS
Status: active
Topics:
    - analysis
    - architecture
    - goja
    - glazed
    - js-bindings
    - tooling
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: glazed/pkg/cmds/cmds.go
      Note: Core Glazed command interfaces and descriptions targeted by compilation
    - Path: glazed/pkg/cmds/loaders/loaders.go
      Note: Dynamic command loader interface and recursive directory loading flow
    - Path: go-go-goja/engine/factory.go
      Note: Goja runtime factory lifecycle and require option composition used by JS verb execution
    - Path: go-go-goja/engine/module_specs.go
      Note: Default module registration model and explicit engine composition seam
    - Path: go-go-goja/pkg/jsdoc/extract/extract.go
      Note: Static tree-sitter extraction precedent and helper source for jsverbs discovery
    - Path: go-go-goja/ttmp/2026/03/16/GOJA-04-JS-GLAZED-EXPORTS--add-glazed-command-exporting-from-javascript/scripts/jsverb_overlay_experiment.go
      Note: Ticket-local proof that an overlay loader can capture top-level functions without breaking relative require
    - Path: go-go-goja/ttmp/2026/03/16/GOJA-04-JS-GLAZED-EXPORTS--add-glazed-command-exporting-from-javascript/sources/local/01-goja-js.md
      Note: Imported source proposal that this design doc interprets and refines
ExternalSources:
    - local:01-goja-js.md
Summary: Grounded design guide for adding JS-defined Glazed commands in go-go-goja using static tree-sitter extraction, normalized verb metadata, Glazed command compilation, and runtime overlay invocation.
LastUpdated: 2026-03-16T14:35:00-04:00
WhatFor: Give a new engineer enough context to understand the current go-go-goja and Glazed architecture, evaluate the imported proposal, and implement a new JS-to-Glazed command layer without guessing.
WhenToUse: Use when implementing or reviewing a feature that exposes JavaScript functions as Glazed commands in go-go-goja.
---


# JS-to-Glazed command exporting design and implementation guide

## Executive Summary

The imported source note in `sources/local/01-goja-js.md` is directionally correct: the repository already has the two main ingredients needed for this feature. First, `go-go-goja` already has a static, tree-sitter-backed sentinel extractor in `pkg/jsdoc/extract/extract.go:23-389`. Second, `glazed` already has a command model and registration path that can accept dynamically produced commands through `pkg/cmds/loaders/loaders.go:23-182`, `pkg/cmds/cmds.go:17-379`, and `pkg/cli/cobra.go:410-451`.

My grounded interpretation is that this ticket should create a new `pkg/jsverbs` subsystem inside `go-go-goja`, not extend `pkg/jsdoc` in place and not move the feature into `glazed`. The extraction and runtime concerns belong with the Goja integration layer, while the output target is still ordinary Glazed commands. The right architecture is:

1. statically scan JavaScript for exported command candidates and sentinel metadata,
2. normalize that data into an internal verb registry,
3. compile each normalized verb into an ordinary Glazed command description plus runtime wrapper,
4. invoke the matching JavaScript function through a runtime overlay loader that preserves normal CommonJS behavior.

The main design refinement relative to the imported note is that the implementation should be explicit about the seam boundaries that already exist in this repo. The existing `pkg/jsdoc` package proves out AST walking, literal decoding, prose attachment, and allowed-root enforcement. The existing `engine.Factory` and `require.WithLoader(...)` hooks prove out runtime customization. The existing Glazed interfaces prove out how command registration, section parsing, and structured output should look. The new work is therefore mostly about composition and model design, not about inventing brand new primitives.

## Problem Statement And Scope

The requested outcome is: define commands in JavaScript, discover them without executing arbitrary JS at startup, compile them into normal Glazed commands, and execute the corresponding JavaScript function at runtime through Goja.

The feature needs to satisfy four constraints that are visible in the current codebase:

- Discovery must be static and deterministic.
  Evidence: `pkg/jsdoc/extract/extract.go:38-56` parses source into a tree-sitter AST and walks nodes without executing the file.
- Runtime behavior must fit the explicit runtime lifecycle used by `engine.Factory` and `engine.Runtime`.
  Evidence: `engine/factory.go:90-179` builds an immutable factory and creates runtime instances with `VM`, `Require`, `Loop`, and `Owner`.
- The result must be ordinary Glazed commands, not a parallel command system.
  Evidence: `glazed/pkg/cmds/cmds.go:17-379` defines the command description and `GlazeCommand` interfaces; `glazed/pkg/cli/cobra.go:410-451` adds built commands into the Cobra tree.
- File loading and path handling must respect allowed roots and repository-local module root conventions.
  Evidence: `pkg/jsdoc/extract/scopedfs.go:20-106` already protects static parsing, and `engine/module_roots.go:11-118` already derives module roots from script paths.

In scope for this ticket:

- static JS discovery,
- command metadata sentinels,
- Glazed schema compilation,
- runtime binding and invocation,
- result adaptation into Glazed rows,
- tests, docs, and developer-facing integration guidance.

Out of scope for v1:

- sandboxing untrusted JavaScript,
- streaming Glazed rows directly from JS,
- TypeScript type generation for JS verb metadata,
- class-method command exposure,
- arbitrary comment parsing,
- watch-mode reload behavior.

## Grounded Interpretation Of The Imported Note

The imported note proposes a sibling `pkg/jsverbs` package beside `pkg/jsdoc`, a sentinel family such as `__package__`, `__section__`, and `__verb__`, a two-phase extractor/normalizer pipeline, and runtime invocation through an overlay source loader. After inspecting the repo, I think those are the right large-grain choices.

What the imported note gets right:

- `pkg/jsverbs` should live in `go-go-goja`, not `glazed`.
  Reason: the parsing and runtime seams live in `go-go-goja`; `glazed` should continue to consume ordinary commands.
- Sentinel-based metadata is a better fit than full JSDoc comment parsing.
  Reason: `pkg/jsdoc/extract/extract.go:159-317` already relies on explicit sentinels and `doc` templates rather than free-form comment parsing.
- The architecture should be split into extraction, normalization, compilation, and runtime.
  Reason: the current codebase already benefits from those boundaries. `pkg/jsdoc/model/store.go:3-89` keeps storage/index concerns separate from parsing, and Glazed keeps command description separate from runtime parsing/execution.
- Runtime overlay loading is viable.
  Reason: the ticket-local experiment in `scripts/jsverb_overlay_experiment.go` successfully captured top-level functions through appended source while preserving relative `require("./helper.js")`.

Where the imported note needs repo-specific refinement:

- The note talks about compiling into `cmds.CommandDescription + schema.Section + fields.Definition`, which is correct, but the real integration point is a concrete `cmds.Command` implementation that also satisfies `cmds.GlazeCommand`.
  Evidence: `glazed/pkg/cmds/cmds.go:336-379`.
- The note suggests a clean `LoadCommandsFromFS` integration. In this repo that means implementing `glazed/pkg/cmds/loaders.CommandLoader`.
  Evidence: `glazed/pkg/cmds/loaders/loaders.go:23-29`.
- The note treats section values abstractly. In this repo runtime bindings should work from `values.Values` and `values.SectionValues`.
  Evidence: `glazed/pkg/cmds/values/section-values.go:155-301`.
- The note proposes reusing `__doc__` prose. That is sensible, but the current `pkg/jsdoc` package only models docs, not functions. The new package should therefore reuse helper ideas, not directly reuse the current `FileDoc` model.
  Evidence: `pkg/jsdoc/model/model.go:17-70`.

My conclusion is:

- keep the imported note's overall architecture,
- implement it in a repo-native way using the existing Goja and Glazed seams,
- treat `pkg/jsdoc` as a precedent and helper source, not as the destination package for command logic.

## Current-State Architecture

### 1. Goja runtime composition

`engine.FactoryBuilder` composes runtime settings before build time, validates them, and freezes them into a reusable `Factory` (`engine/factory.go:15-29`, `engine/factory.go:90-132`). `Factory.NewRuntime(...)` then creates a `goja.Runtime`, starts an event loop, creates a `runtimeowner.Runner`, enables the `require` registry, enables console support, and runs runtime initializers (`engine/factory.go:134-179`).

That matters for this ticket because the JS-command runtime should not sidestep this machinery. A generated JS verb command should create a runtime through the factory, not through ad-hoc `goja.New()` calls. That preserves:

- opt-in module registration via `engine.DefaultRegistryModules()` (`engine/module_specs.go:64-82`),
- require customization via `engine.WithRequireOptions(...)` (`engine/options.go:13-20`),
- owned lifecycle via `engine.Runtime` (`engine/runtime.go:21-49`),
- loop ownership and cross-goroutine safety via `pkg/runtimeowner/runner.go:27-225`.

### 2. JS static extraction precedent

The strongest precedent is `pkg/jsdoc/extract/extract.go`. It already shows how this repo parses JavaScript source with tree-sitter, walks the top-level AST, identifies sentinel calls, attaches line numbers, and decodes simple object literals into Go models (`pkg/jsdoc/extract/extract.go:38-56`, `pkg/jsdoc/extract/extract.go:96-205`, `pkg/jsdoc/extract/extract.go:271-389`).

Key reusable ideas from that package:

- parser construction and AST walking,
- sentinel detection via `call_expression`,
- lightweight JS-object-to-JSON literal decoding,
- `doc` tagged-template frontmatter parsing,
- 1-based source line preservation,
- allowed-root enforcement via `ScopedFS`.

Key things that are too `jsdoc`-specific to reuse directly:

- `FileDoc`, `Package`, `SymbolDoc`, and `Example` as the main output model,
- `DocStore` indexing rules,
- the assumption that only documentation sentinels matter and function declarations do not.

This is why a sibling package is preferable to extending `pkg/jsdoc` in place.

### 3. Glazed command plumbing

Glazed already has the exact runtime types this feature should target:

- `cmds.CommandDescription` carries `Name`, `Short`, `Long`, `Schema`, `Parents`, and `Source` (`glazed/pkg/cmds/cmds.go:17-35`).
- `schema.SectionImpl` groups field definitions and knows how to register them on Cobra (`glazed/pkg/cmds/schema/section-impl.go:10-127`, `glazed/pkg/cmds/schema/section-impl.go:216-258`).
- `fields.Definition` already supports names, types, help text, defaults, choices, requiredness, arguments, and short flags (`glazed/pkg/cmds/fields/definitions.go:16-84`).
- The legal Glazed type strings are already enumerated in `glazed/pkg/cmds/fields/field-type.go:7-47`.
- Parsed values arrive at runtime as `values.Values`, grouped by section slug (`glazed/pkg/cmds/values/section-values.go:155-301`).
- `cli.BuildCobraCommandFromCommand(...)` and `cli.AddCommandsToRootCommand(...)` already wire dynamic commands into the Cobra tree (`glazed/pkg/cli/cobra.go:379-451`).

This is a major simplification. The JS-command layer does not need to invent flag parsing, help layout, or CLI registration. It only needs to compile into the existing target types.

### 4. Existing CLI precedent inside go-go-goja

The `cmd/goja-jsdoc` command is the best local precedent for "new reusable subsystem plus thin Glazed CLI wrapper." Its `extract` command builds a `cmds.CommandDescription`, decodes values through the default section, and delegates into a package-level implementation (`cmd/goja-jsdoc/extract_command.go:19-95`).

That suggests a good shape for future user-facing JS-command tooling:

- keep the reusable logic in `pkg/jsverbs/...`,
- keep any CLI integration thin,
- avoid embedding the business logic directly in Cobra command setup.

## Design Goals

The design should optimize for these goals:

1. Preserve static discovery.
   Startup should not execute user JS just to discover commands.
2. Preserve Glazed normality.
   Generated commands should look like normal Glazed commands to the rest of the system.
3. Preserve CommonJS behavior.
   Relative `require()` inside command source files should keep working.
4. Reuse the current runtime lifecycle.
   Command execution should go through `engine.Factory`, `Runtime`, and `runtimeowner`.
5. Keep metadata near the JS code.
   Sentinel metadata should live in the same file as the functions it describes.
6. Fail loudly on collisions and unsupported syntax.
   Discovery should be deterministic and explicit.

## Proposed Package Layout

I recommend the following package layout inside `go-go-goja/pkg`:

```text
pkg/
  jsverbs/
    model/
      raw.go
      model.go
      diagnostics.go
      registry.go

    extract/
      extract.go
      functions.go
      literals.go
      templates.go

    normalize/
      normalize.go
      merge_jsdoc.go

    compile/
      glazed.go
      bindings.go
      help.go

    runtime/
      command.go
      loader.go
      invoke.go
      marshal.go
      adapt.go
      context.go

    loaders/
      loader.go
```

Reasoning for this split:

- `model/` keeps extraction output and normalized runtime contracts distinct.
- `extract/` remains static and syntax-oriented.
- `normalize/` is where semantic rules, defaults, inheritance, and collision checks should live.
- `compile/` is explicitly Glazed-facing.
- `runtime/` is explicitly Goja-facing.
- `loaders/` is the Glazed `CommandLoader` adapter.

This is consistent with the way the repo already separates extraction, model, and consumer-specific behavior in `pkg/jsdoc`.

## Authoring Model

### Sentinel family

I agree with the imported note that the command system should use explicit sentinels rather than comment parsing.

Recommended sentinel set:

- `__package__(...)`
  File-level metadata and default command grouping.
- `__section__(...)`
  Shared or reusable flag sections.
- `__verb__(...)`
  Per-function command metadata.
- `doc\`...\``
  Long-form prose and frontmatter for command help.

This is additive to the existing `jsdoc` style. A future implementation may optionally merge in `__doc__` summaries and parameter descriptions, but command extraction should not require `__doc__`.

### Minimal authoring example

```js
__package__({
  name: "math",
  title: "Math tools",
  description: "Small math utilities exposed as CLI verbs."
});

__verb__("clamp", {
  short: "Clamp a number to a range",
  params: {
    value: { type: "float", positional: true, help: "Input value" },
    min: { type: "float", positional: true, help: "Lower bound" },
    max: { type: "float", positional: true, help: "Upper bound" }
  }
});

doc`
---
verb: clamp
---

Clamp a value to the inclusive range [min, max].
`;

function clamp(value, min, max) {
  return Math.min(Math.max(value, min), max);
}

function lerp(a, b, t = 0.5) {
  return a + (b - a) * t;
}
```

### Public function discovery rules

For v1 I recommend discovering these top-level forms:

- `function foo(...) {}`
- `async function foo(...) {}`
- `const foo = (...) => {}`
- `const foo = async (...) => {}`
- `const foo = function (...) {}`
- exported variants of the same forms if they appear in source

Do not auto-expose:

- nested functions,
- class methods,
- anonymous expressions,
- names starting with `_`.

Inference based on current Glazed loader behavior:

Because `LoadCommandsFromFS` recursively scans directories (`glazed/pkg/cmds/loaders/loaders.go:109-182`), accidental command exposure would be confusing. I therefore recommend one loader option for strict mode:

```go
type LoaderOptions struct {
    RequireVerbSentinel bool
}
```

Default behavior can remain auto-expose-public-functions, but hosts should be able to opt into strict sentinel-only behavior.

## Core Data Model

### Raw extraction model

```go
type RawFileSpec struct {
    FilePath       string
    Package        *RawPackage
    SharedSections []*RawSection
    VerbOverlays   []*RawVerb
    Functions      []*FunctionDecl
    DocBlocks      []*RawDocBlock
    Diagnostics    []Diagnostic
}

type FunctionDecl struct {
    Name       string
    Async      bool
    Kind       FunctionKind
    Params     []*FunctionParam
    SourceFile string
    Line       int
}

type FunctionParam struct {
    Name          string
    IsDestructured bool
    DefaultLiteral any
}
```

### Normalized model

```go
type Registry struct {
    Verbs       []*VerbSpec
    ByPath      map[string]*VerbSpec
    Diagnostics []Diagnostic
}

type VerbSpec struct {
    FunctionName string
    CommandName  string
    Parents      []string
    Short        string
    Long         string
    Hidden       bool
    Sections     []*SectionSpec
    Bindings     []*BindingSpec
    SourceFile   string
    Line         int
}

type SectionSpec struct {
    Slug        string
    Name        string
    Description string
    Prefix      string
    Fields      []*FieldSpec
}

type FieldSpec struct {
    Name       string
    Type       string
    Help       string
    Default    any
    Choices    []string
    Required   bool
    ShortFlag  string
    Positional bool
}

type BindingSpec struct {
    ParamName   string
    Kind        BindingKind // flag | section | all | context
    SectionSlug string
    FieldName   string
}
```

Why this split matters:

- extraction can stay deterministic and syntax-only,
- normalization can accumulate semantic diagnostics,
- compilation can focus only on Glazed targets,
- runtime can rely on a stable binding contract.

## Discovery, Normalization, Compilation, Runtime

### High-level architecture diagram

```text
JavaScript source files
        |
        v
+---------------------+
| jsverbs/extract     |
| - tree-sitter walk  |
| - function capture  |
| - sentinel capture  |
| - doc block capture |
+---------------------+
        |
        v
+---------------------+
| jsverbs/normalize   |
| - defaults          |
| - path rules        |
| - collision checks  |
| - binding plan      |
+---------------------+
        |
        +-----------------------------+
        |                             |
        v                             v
+---------------------+      +----------------------+
| jsverbs/compile     |      | jsverbs/runtime      |
| -> Glazed command   |      | -> call JS function  |
| descriptions        |      | -> adapt results     |
+---------------------+      +----------------------+
        |
        v
+---------------------+
| Glazed/Cobra tree   |
+---------------------+
```

### Phase A: extraction

Implementation notes:

- Reuse the tree-sitter parser setup from `pkg/jsdoc/extract/extract.go:38-56`.
- Reuse the literal-decoding helper style from `pkg/jsdoc/extract/extract.go:385-519`, but move any reusable pieces into `pkg/jsverbs/extract/literals.go`.
- Extend the AST walker to capture function declarations and variable declarators in addition to sentinel calls.

Pseudocode:

```pseudo
function ExtractFile(path, src):
  tree = parseJavaScript(src)
  raw = new RawFileSpec(path)

  for node in root.children:
    switch node.kind:
      case call_expression:
        maybeCapturePackage(node)
        maybeCaptureSection(node)
        maybeCaptureVerb(node)
        maybeCaptureDocTemplate(node)
      case function_declaration:
        captureFunction(node)
      case lexical_declaration:
        captureVariableFunctionForms(node)
      case export_statement:
        recurseIntoExport(node)

  return raw
```

### Phase B: normalization

Normalization should:

1. compute command paths,
2. infer default field definitions from JS parameters,
3. merge explicit overrides,
4. attach shared sections,
5. merge help text and prose,
6. validate collisions and unsupported shapes,
7. produce a final `Registry`.

Important normalization rules:

- command path = discovered parents from directory + optional package path + verb leaf,
- default field name = kebab-case parameter name,
- no JS default literal means required field,
- destructured parameters are invalid unless explicitly bound as `section` or `all`,
- duplicate command paths are hard errors,
- duplicate short flags are hard errors,
- positional arguments must stay in the default section.

### Phase C: Glazed compilation

Compilation should be intentionally boring:

- `VerbSpec.CommandName` -> `cmds.CommandDescription.Name`
- `VerbSpec.Parents` -> `cmds.WithParents(...)`
- `VerbSpec.Short` and `Long` -> help text
- `SectionSpec` -> `schema.NewSection(...)`
- `FieldSpec` -> `fields.New(...)`
- positional fields -> `fields.WithIsArgument(true)`

Compilation pseudocode:

```pseudo
function CompileVerb(verbSpec):
  sections = []
  for each sectionSpec in verbSpec.sections:
    section = schema.NewSection(sectionSpec.slug, sectionSpec.name, ...)
    for each fieldSpec in sectionSpec.fields:
      def = fields.New(fieldSpec.name, fieldSpec.type, ...)
      if fieldSpec.positional:
        def.IsArgument = true
      section.AddFields(def)
    sections.append(section)

  desc = cmds.NewCommandDescription(
    verbSpec.commandName,
    WithShort(verbSpec.short),
    WithLong(verbSpec.long),
    WithParents(verbSpec.parents...),
    WithSections(sections...),
    WithSource("jsverbs/" + verbSpec.sourceFile),
  )

  return JSVerbCommand{CommandDescription: desc, Verb: verbSpec}
```

### Phase D: runtime invocation

Runtime should construct a real `cmds.GlazeCommand` implementation. That command should:

1. create a fresh runtime from `engine.Factory`,
2. install sentinel no-ops,
3. require the target file through an overlay source loader,
4. fetch the target function from a global registry inserted by the overlay,
5. convert parsed Glazed values into JS call arguments,
6. call the JS function,
7. adapt the result into Glazed rows.

Runtime diagram:

```text
parsed Glazed values
        |
        v
+------------------------+
| JSVerbCommand.Run...   |
| create runtime         |
+------------------------+
        |
        v
+------------------------+
| require.WithLoader     |
| overlay source         |
| - sentinel no-ops      |
| - original source      |
| - registry appendix    |
+------------------------+
        |
        v
+------------------------+
| globalThis registry    |
| function lookup        |
+------------------------+
        |
        v
+------------------------+
| JS invocation          |
| result adaptation      |
+------------------------+
        |
        v
Glazed rows
```

## Overlay Loader Design

### Why the overlay is needed

The imported note is correct that CommonJS top-level functions are not automatically exported. Requiring a file normally only gives `module.exports`, not arbitrary top-level declarations.

The overlay strategy is the cleanest repo-native answer because:

- `engine.WithRequireOptions(...)` already accepts `require.WithLoader(...)` (`engine/options.go:13-20`),
- `cmd/bun-demo/main.go:21-58` already proves that custom source loaders are a supported pattern here,
- the overlay preserves CommonJS semantics instead of rewriting `module.exports`.

### Experiment result

I validated the overlay idea with `ttmp/.../scripts/jsverb_overlay_experiment.go`.

Observed result:

```json
{
  "hiddenType": "func(goja.FunctionCall) goja.Value",
  "listIssues": "repo:openai/openai",
  "moduleExports": {
    "exported": true
  },
  "registryKeys": [
    "listIssues",
    "hidden"
  ]
}
```

What this proves:

- appended code in the same module scope can still reference top-level function and `const` declarations,
- relative `require("./helper.js")` continues to work,
- the overlay can register functions without mutating `module.exports`.

Operational note:

- Running `go run` from the repo root initially failed because the top-level `go.work` file advertises lower Go versions than some modules require.
- The experiment ran successfully with `GOWORK=off`.

That exact failure should be preserved in the diary because it affects future ticket-local experiments.

## Binding Model

Each JS parameter should compile into one of four runtime binding modes.

### `bind: "flag"`

Default mode. One Glazed field becomes one JS argument.

Example:

```js
function echo(text, upper = false) {}
```

Runtime:

- `text` <- string or positional argument,
- `upper` <- bool flag.

### `bind: "section"`

One JS parameter receives an object built from one Glazed section.

Example:

```js
function listIssues(repo, auth) {}
```

Where:

```js
auth = {
  token: "...",
  baseUrl: "https://api.github.com"
}
```

Recommendation:

- JS object keys should be camelCase,
- CLI flags should remain kebab-case,
- normalization should own this name conversion.

### `bind: "all"`

One JS parameter receives a grouped object of all resolved sections.

Suggested shape:

```js
{
  default: { ... },
  auth: { ... },
  glazed: { ... },
  commandSettings: { ... }
}
```

### `bind: "context"`

One JS parameter receives runtime metadata rather than user-provided flags.

Suggested shape:

```js
{
  verb: {
    name: "list-issues",
    functionName: "listIssues",
    sourceFile: "github/issues.js",
    path: ["github", "list-issues"]
  },
  sections: { ... },
  cwd: "/current/dir"
}
```

Keep this intentionally small in v1.

## Help Text And JSDoc Merging

I recommend a layered help strategy rather than making `jsverbs` depend on `jsdoc` authoring everywhere.

Merge order:

1. `__verb__.short`
2. matching `__doc__.summary` if available
3. humanized function name

For flag help:

1. explicit `__verb__.params[param].help`
2. matching `__doc__.params[].description`
3. empty string

For long help:

1. `doc` block with `verb: <name>` frontmatter
2. matching `__doc__.prose`
3. package prose

This gives a nice migration path:

- command authors can stay fully command-centric,
- existing jsdocex-style docs can enrich help without becoming mandatory.

## Security And Allowed Roots

This feature must make an explicit trust-boundary statement: it is not a sandbox for untrusted JavaScript.

Static discovery should reuse the defensive posture already present in `pkg/jsdoc/extract/scopedfs.go:20-106`:

- reject absolute paths,
- reject traversal,
- reject symlink escapes,
- require all parsed files to stay inside the configured root.

Runtime execution should mirror that intent:

- only load files under an explicitly provisioned root,
- only expose native modules that the host registered on the engine factory,
- never imply that "Goja means safe."

## API Sketch

### Top-level package API

```go
package jsverbs

type Loader struct {
    factory *engine.Factory
    rootDir string
    opts    LoaderOptions
}

func NewLoader(factory *engine.Factory, rootDir string, opts ...LoaderOption) (*Loader, error)

func (l *Loader) Discover(ctx context.Context) (*model.Registry, error)
```

### Glazed loader adapter

```go
package loaders

type JSVerbLoader struct {
    Factory *engine.Factory
    RootDir string
    Opts    Options
}

func (l *JSVerbLoader) LoadCommands(
    f fs.FS,
    entryName string,
    options []cmds.CommandDescriptionOption,
    aliasOptions []alias.Option,
) ([]cmds.Command, error)

func (l *JSVerbLoader) IsFileSupported(f fs.FS, fileName string) bool
```

### Runtime command wrapper

```go
type JSVerbCommand struct {
    *cmds.CommandDescription
    Verb    *model.VerbSpec
    Factory *engine.Factory
    RootDir string
}

func (c *JSVerbCommand) RunIntoGlazeProcessor(
    ctx context.Context,
    parsedValues *values.Values,
    gp middlewares.Processor,
) error
```

## Detailed Implementation Plan

### Phase 1: extraction package

Files:

- `pkg/jsverbs/model/raw.go`
- `pkg/jsverbs/extract/extract.go`
- `pkg/jsverbs/extract/functions.go`
- `pkg/jsverbs/extract/literals.go`
- `pkg/jsverbs/extract/templates.go`

Tasks:

1. Copy the parser/bootstrap approach from `pkg/jsdoc/extract/extract.go`.
2. Add function-declaration discovery.
3. Add `__section__` and `__verb__` literal parsing.
4. Preserve line numbers and source file paths.
5. Add extraction diagnostics instead of silent drops where possible.

Tests:

- top-level function discovery,
- arrow and function-expression discovery,
- underscore-private exclusion,
- sentinel extraction,
- doc block attachment,
- line-number correctness.

### Phase 2: normalization package

Files:

- `pkg/jsverbs/model/model.go`
- `pkg/jsverbs/model/diagnostics.go`
- `pkg/jsverbs/model/registry.go`
- `pkg/jsverbs/normalize/normalize.go`
- `pkg/jsverbs/normalize/merge_jsdoc.go`

Tasks:

1. Build command paths from directory and package context.
2. Infer fields from function parameters.
3. Attach shared sections.
4. Compile binding plans.
5. Detect collisions and unsupported cases.

Tests:

- duplicate command path detection,
- duplicate short flags,
- default type inference,
- destructuring rejection,
- `bind: section`, `bind: all`, `bind: context`,
- package-path overrides.

### Phase 3: Glazed compilation

Files:

- `pkg/jsverbs/compile/glazed.go`
- `pkg/jsverbs/compile/bindings.go`
- `pkg/jsverbs/compile/help.go`

Tasks:

1. Map section specs to `schema.Section`.
2. Map fields to `fields.Definition`.
3. Construct `cmds.CommandDescription`.
4. Preserve `Parents` and `Source`.
5. Generate stable short and long help text.

Tests:

- field type compilation,
- positional argument ordering,
- prefixed section flags,
- help-merge precedence.

### Phase 4: runtime package

Files:

- `pkg/jsverbs/runtime/command.go`
- `pkg/jsverbs/runtime/loader.go`
- `pkg/jsverbs/runtime/invoke.go`
- `pkg/jsverbs/runtime/marshal.go`
- `pkg/jsverbs/runtime/adapt.go`
- `pkg/jsverbs/runtime/context.go`

Tasks:

1. Build overlay loader around `require.WithLoader(...)`.
2. Install sentinel no-ops.
3. Register discovered top-level functions in a runtime-global registry.
4. Resolve the selected function by file and name.
5. Marshal section values into JS arguments.
6. Adapt sync and async results into rows.

Tests:

- relative require survives overlay,
- top-level function capture,
- object-param section binding,
- all-sections binding,
- context binding,
- primitive/object/list result adaptation,
- promise resolution if async support is enabled.

### Phase 5: Glazed loader integration

Files:

- `pkg/jsverbs/loaders/loader.go`
- optionally a small example command under `cmd/`

Tasks:

1. Implement `glazed/pkg/cmds/loaders.CommandLoader`.
2. Return multiple commands per JS file when necessary.
3. Integrate with `loaders.LoadCommandsFromFS(...)`.
4. Document how host apps attach the loader.

Minimal host example:

```go
factory, err := engine.NewBuilder().
    WithModules(engine.DefaultRegistryModules()).
    Build()
if err != nil {
    return err
}

loader := jsverbsloaders.NewJSVerbLoader(factory, rootDir)

commands, err := loaders.LoadCommandsFromFS(
    os.DirFS(rootDir),
    ".",
    "jsverbs",
    loader,
    nil,
    nil,
)
if err != nil {
    return err
}

return cli.AddCommandsToRootCommand(rootCmd, commands, nil)
```

## Test And Validation Strategy

Unit tests:

- extractor tests using small inline JS snippets,
- normalizer tests using raw fixtures,
- compiler tests over normalized specs,
- runtime loader and invocation tests using in-memory source maps,
- result adapter tests for primitives, objects, arrays, and nulls.

Integration tests:

- one directory with multiple JS verbs,
- nested package paths,
- duplicate path failure,
- host command registration through `LoadCommandsFromFS(...)`,
- smoke test executing a generated Glazed command and asserting rows.

Manual validation:

1. Create a sample JS directory with `__package__`, `__section__`, `__verb__`, and plain top-level functions.
2. Discover and register commands into a small Cobra root.
3. Run:
   - positional-argument command,
   - prefixed-section command,
   - async-return command,
   - malformed-file failure case.
4. Confirm help text, command paths, and row output.

## Risks, Alternatives, And Open Questions

### Risks

- Accidental command exposure if auto-discovery is too permissive.
- Complexity creep if `jsverbs` tries to subsume every `jsdoc` behavior instead of staying command-focused.
- Binding ambiguity for destructured parameters and nested objects.
- Runtime/debug confusion if overlay-generated source is not easy to inspect in errors.

### Recommended mitigations

- provide strict sentinel-only mode,
- accumulate file-local diagnostics before failing,
- include source file and line in every normalized verb,
- keep the overlay source builder small and testable,
- write the generated overlaid source to temp files in debug mode when needed.

### Alternatives considered

#### 1. Sidecar YAML or JSON command descriptors

Rejected because metadata drift is likely and the repo already favors code-local metadata.

#### 2. Extending `pkg/jsdoc` directly

Rejected because documentation and command exposure are related but distinct concerns. `pkg/jsdoc` should remain focused on documentation extraction and export.

#### 3. Runtime-only introspection

Rejected because it would require executing JS during discovery, which conflicts with the repository's existing static-extraction precedent and makes failure modes less deterministic.

### Open questions

1. Should v1 auto-expose public top-level functions by default, or require `__verb__` unless explicitly opted out?
2. Should `__section__` be file-scoped only in v1, or support package-wide reuse across files?
3. Should async support await native promises immediately in v1, or land after sync commands are stable?
4. Should `jsverbs` merge with `jsdoc` only when a separate `DocStore` is available, or directly parse `__doc__` in the same pass?

## File-Level Guidance For A New Intern

If you are implementing this ticket from scratch, start in this order:

1. Read `go-go-goja/pkg/jsdoc/extract/extract.go`.
   Learn how the repo already uses tree-sitter and simple JS-literal decoding.
2. Read `go-go-goja/engine/factory.go`, `engine/runtime.go`, and `pkg/runtimeowner/runner.go`.
   Learn how runtimes are constructed and why you should not bypass the factory.
3. Read `glazed/pkg/cmds/cmds.go`, `glazed/pkg/cmds/schema/section-impl.go`, `glazed/pkg/cmds/fields/definitions.go`, and `glazed/pkg/cmds/values/section-values.go`.
   Learn the command/section/field/value model you are compiling into.
4. Read `glazed/pkg/cmds/loaders/loaders.go` and `glazed/pkg/cli/cobra.go`.
   Learn how dynamically discovered commands get attached to Cobra.
5. Run `ttmp/.../scripts/jsverb_overlay_experiment.go`.
   Learn why the overlay loader approach works before you write production code.

## References

- Imported proposal:
  - `ttmp/2026/03/16/GOJA-04-JS-GLAZED-EXPORTS--add-glazed-command-exporting-from-javascript/sources/local/01-goja-js.md`
- Ticket-local experiment:
  - `ttmp/2026/03/16/GOJA-04-JS-GLAZED-EXPORTS--add-glazed-command-exporting-from-javascript/scripts/jsverb_overlay_experiment.go`
- Goja runtime composition:
  - `engine/factory.go:15-179`
  - `engine/runtime.go:21-49`
  - `engine/module_specs.go:14-82`
  - `engine/module_roots.go:11-118`
  - `pkg/runtimeowner/runner.go:27-225`
- JS extraction precedent:
  - `pkg/jsdoc/extract/extract.go:23-584`
  - `pkg/jsdoc/extract/scopedfs.go:20-106`
  - `pkg/jsdoc/model/model.go:17-70`
  - `pkg/jsdoc/model/store.go:3-89`
  - `cmd/goja-jsdoc/extract_command.go:19-95`
  - `cmd/goja-jsdoc/doc/01-jsdoc-system.md:35-259`
- Glazed command target types:
  - `../glazed/pkg/cmds/cmds.go:17-379`
  - `../glazed/pkg/cmds/loaders/loaders.go:23-182`
  - `../glazed/pkg/cmds/schema/section-impl.go:10-279`
  - `../glazed/pkg/cmds/fields/definitions.go:16-84`
  - `../glazed/pkg/cmds/fields/field-type.go:7-47`
  - `../glazed/pkg/cmds/fields/cobra.go:54-280`
  - `../glazed/pkg/cmds/values/section-values.go:155-301`
  - `../glazed/pkg/cli/cobra-parser.go:91-260`
  - `../glazed/pkg/cli/cobra.go:379-451`
  - `../glazed/pkg/cmds/runner/run.go:39-92`
