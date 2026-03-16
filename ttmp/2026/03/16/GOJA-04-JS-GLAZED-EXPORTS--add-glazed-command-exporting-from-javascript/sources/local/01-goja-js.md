---
Title: Imported source note: goja-js
Ticket: GOJA-04-JS-GLAZED-EXPORTS
Status: active
Topics:
    - analysis
    - architecture
    - goja
    - glazed
    - js-bindings
    - tooling
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources:
    - local:/tmp/goja-js.md
Summary: Imported source note proposing a jsdocex-style static metadata system that compiles JavaScript functions into ordinary Glazed commands.
LastUpdated: 2026-03-16T14:52:00-04:00
WhatFor: Preserve the original proposal text inside the ticket so the grounded design guide can reference and refine it.
WhenToUse: Use when reviewing the original unedited proposal that motivated GOJA-04.
---

JSDoc itself is comment-block based, but your repo already has a tree-sitter sentinel extractor (`pkg/jsdoc/extract`) rather than a free-form comment parser. Goja is a pure-Go ECMAScript 5.1+ runtime, and Glazed already gives you a rich command model with fields, sections, and multiple output modes. So the right cut here is not “teach JSDoc new tricks.” It is “build a sibling jsdocex-style static metadata system for CLI verbs, then compile that into ordinary Glazed commands.” ([JSDoc][1])

The key idea is simple: every public top-level JS function becomes a candidate verb; optional sentinel metadata next to the function refines the command path, flags, sections, help text, and binding rules; then the runner turns that into `cmds.CommandDescription + schema.Section + fields.Definition`, which means the rest of the Glazed machinery stays boring and reliable instead of mutating into a many-headed parser goblin.

## Core proposal

Add a new subsystem, `pkg/jsverbs`, parallel to `pkg/jsdoc`.

It should do four things:

1. Statically scan JS files without executing them.
2. Discover public top-level functions and CLI metadata sentinels.
3. Normalize that into a `VerbRegistry`.
4. Compile each discovered verb into a regular Glazed command that invokes the matching JS function at runtime.

The most important design decision is this:

**Use explicit sentinel calls and tagged templates, not comment parsing.**

That preserves the jsdocex feel, keeps extraction deterministic, and lets the metadata be shaped like Glazed’s actual schema model instead of squeezing CLI structure through comment tags.

## User-facing authoring model

I would use this sentinel family:

* `__package__(...)` for file/module-level metadata
* `__section__(...)` for reusable flag sections
* `__verb__(...)` for per-function CLI metadata
* `doc\`...`` for long-form prose/help text

Runtime should inject no-op implementations for these symbols so the same file can be executed normally.

### Minimal example

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
    min:   { type: "float", positional: true, help: "Lower bound" },
    max:   { type: "float", positional: true, help: "Upper bound" }
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

That yields:

* `math clamp <value> <min> <max>`
* `math lerp --a ... --b ... --t ...`

`clamp` is customized. `lerp` is auto-exposed from the signature.

### Shared section example

```js
__package__({
  name: "github",
  title: "GitHub tools",
  defaultSections: ["auth"]
});

__section__("auth", {
  title: "Authentication",
  prefix: "gh",
  flags: [
    { name: "token", type: "string", required: true, help: "GitHub token" },
    { name: "base-url", type: "string", default: "https://api.github.com", help: "API base URL" }
  ]
});

__verb__("listIssues", {
  name: "list-issues",
  short: "List issues for a repository",
  useSections: ["auth"],
  params: {
    repo:  { positional: true, help: "owner/repo" },
    state: { type: "choice", choices: ["open", "closed", "all"], default: "open" },
    auth:  { bind: "section", section: "auth" },
    ctx:   { bind: "context" }
  }
});

async function listIssues(repo, state = "open", auth, ctx) {
  // auth = { token, baseUrl }
  // ctx contains runner metadata + resolved section values
  return [];
}
```

That yields a verb roughly like:

```bash
runner github list-issues openai/openai --state open --gh-token $TOKEN
```

## Sentinel schemas

Use JSON-ish object literals only. No computed keys, spreads, function values, interpolations, or other jazz-hands.

```ts
type PackageDef = {
  name?: string;              // slash-separated package path, default = file stem
  title?: string;
  description?: string;
  defaultSections?: string[]; // shared sections auto-attached to all verbs in file
  hidden?: boolean;           // hide package/group from auto-attach helpers
};

type SectionDef = {
  name: string;               // unique within file/package
  title?: string;
  description?: string;
  prefix?: string;            // Glazed section prefix
  flags: FlagDef[];
};

type FlagDef = {
  name: string;               // CLI field name, e.g. "base-url"
  short?: string;             // short flag
  type?: string;              // existing glazed field type strings
  help?: string;
  default?: any;
  choices?: string[];
  required?: boolean;
  positional?: boolean;       // compile as argument instead of flag
};

type ParamOverride = {
  name?: string;              // flag name override
  section?: string;           // section slug; default = "default"
  short?: string;
  type?: string;
  help?: string;
  default?: any;
  choices?: string[];
  required?: boolean;
  positional?: boolean;
  bind?: "flag" | "section" | "all" | "context";
};

type VerbDef = {
  name?: string;              // CLI leaf name; default = kebab-case(functionName)
  short?: string;
  aliases?: string[];
  hidden?: boolean;
  useSections?: string[];     // attach shared sections
  sections?: SectionDef[];    // inline, verb-local sections
  params?: Record<string, ParamOverride>;
};
```

## Public function discovery rules

The extractor should recognize these top-level forms:

* `function foo(...) {}`
* `async function foo(...) {}`
* `const foo = (...) => {}`
* `const foo = async (...) => {}`
* `const foo = function(...) {}`
* `export function foo(...) {}` if present in source

It should **not** expose:

* nested functions
* class methods
* anonymous expressions
* names starting with `_` unless explicitly decorated with `__verb__`

That last rule is important. Without it, helper functions become accidental commands and the CLI turns into a raccoon nest.

## Parameter inference

Every public top-level function becomes a candidate verb even with no `__verb__` metadata.

Default inference rules:

* parameter name → field name using kebab-case
* no JS default → required field
* `= true/false` → `bool`
* `= 1` → `int`
* `= 1.5` → `float`
* `= "x"` → `string`
* `= ["a", "b"]` → `stringList`
* `= [1, 2]` → `intList`
* `= [1.5, 2.5]` → `floatList`
* unknown / unsupported default literal → field type falls back to `string`

Destructured params are where dragons sleep. For v1:

* destructured params are **not auto-exposed**
* if a function uses destructuring, the user must bind that parameter as `bind: "all"` or `bind: "section"`

Example:

```js
function good(name, verbose = false) {}
function alsoGood(auth, ctx) {} // if bound via metadata
function nope({ name, age }) {} // error unless explicitly bound
```

## Binding model

This is the most important runtime contract.

Each JS function parameter gets a binding plan.

### `bind: "flag"` (default)

The parameter receives one scalar value from one Glazed field.

Example:

```js
function echo(text, upper = false) {}
```

becomes:

* `text` ← `--text` or positional argument if marked so
* `upper` ← `--upper`

### `bind: "section"`

The parameter receives one section object.

Example:

```js
__section__("auth", { ... });

__verb__("fetchData", {
  useSections: ["auth"],
  params: {
    auth: { bind: "section", section: "auth" }
  }
});

function fetchData(auth) {}
```

At runtime:

```js
auth = {
  token: "...",
  baseUrl: "..."
}
```

I would materialize object keys in `camelCase`, with the original field names available only in raw metadata if needed. CLI flags can stay kebab-case; JS objects should not.

### `bind: "all"`

The parameter receives all resolved section values.

```js
function main(all) {}
```

At runtime:

```js
all = {
  default: { ... },
  auth: { ... },
  glazed: { ... },   // if included
  command: { ... }   // if included
}
```

### `bind: "context"`

The parameter receives runner context, not user flags.

Suggested shape:

```js
{
  verb: {
    name: "list-issues",
    functionName: "listIssues",
    sourceFile: "github/issues.js",
    path: ["github", "list-issues"]
  },
  sections: { ... },      // same grouped values as bind:"all"
  cwd: "/current/dir"
}
```

Keep this small in v1. Do not dump the full process environment in there by default unless you enjoy accidental footguns.

## Command path strategy

When a directory is provisioned, every discovered verb gets a stable command path.

I would compute it like this:

1. Start with relative directory parents from the provisioned root.
2. Add package parents from `__package__.name`, split by `/`.
3. If no package name was declared, use the file stem as the package segment.
4. Use `__verb__.name` or kebab-case of the function name as the leaf command.

Examples:

* `scripts/math.js` with `function clamp()` → `math clamp`
* `scripts/github/issues.js` with `__package__({name:"github"})` and `function listIssues()` → `github list-issues`
* `scripts/admin/users/list.js` with no package override and `function run()` → `admin users list run`

Collision rules:

* duplicate full command path = hard error
* duplicate shared section name within file/package = hard error
* duplicate local section names within a verb = hard error

You want discovery to fail loudly and early, not quietly choose one command and leave future-you in a swamp.

## Help and documentation merge

This system should play nicely with the existing jsdoc-style metadata instead of pretending it lives on a different planet.

Recommended merge order:

* command short help:

  1. `__verb__.short`
  2. matching `__doc__.summary`
  3. humanized function name

* flag help:

  1. `__verb__.params[param].help`
  2. matching `__doc__.params[].description`
  3. empty string

* long help:

  1. `doc` template with `verb: <functionName>`
  2. matching jsdoc symbol prose
  3. package prose as preamble

That lets users keep existing symbol docs and only add CLI-specific bits where needed.

## Extraction architecture

I would split this into two phases.

### Phase A: raw extraction

Produce a raw file model:

```go
type RawFileSpec struct {
    FilePath       string
    Package        *RawPackage
    SharedSections []*RawSection
    VerbOverlays   []*RawVerb
    Functions      []*FunctionDecl
    ProseBlocks    []*RawDocBlock
}
```

With:

```go
type FunctionDecl struct {
    Name       string
    Async      bool
    Params     []*FunctionParam
    Kind       FunctionKind
    SourceFile string
    Line       int
}
```

This phase only parses syntax and literals. It does not decide command paths yet.

### Phase B: normalization

Normalize raw data into a runtime/compiler model:

```go
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
```

`BindingSpec` is the contract between extraction and execution:

```go
type BindingSpec struct {
    ParamName   string
    Kind        BindingKind
    SectionSlug string
    FieldName   string
    Required    bool
}
```

This split matters. It keeps the tree-sitter pass dumb and deterministic, and it moves all semantic rules—collisions, inheritance, default sections, help merge, binding rules—into a pure normalization layer that is easy to test.

## Reuse existing repo patterns

I would not invent a parallel loader framework.

Glazed already has `pkg/cmds/loaders.CommandLoader` and a recursive directory-loading flow. Use that.

The JS loader should implement the same interface and return **multiple commands per file**.

That means a host program can provision a directory roughly like this:

```go
factory, err := engine.NewBuilder().
    WithModules(engine.DefaultRegistryModules()).
    Build()
if err != nil {
    return err
}

loader := jsverbsloaders.NewJSVerbLoader(
    factory,
    jsverbsloaders.WithRoot(rootDir),
)

commands, err := loaders.LoadCommandsFromFS(
    os.DirFS(rootDir),
    ".",
    "js",
    loader,
    nil,
    nil,
)
if err != nil {
    return err
}

return cli.AddCommandsToRootCommand(rootCmd, commands, nil)
```

That is the cleanest part of this design: the JS side is dynamic, but the host still deals in ordinary `[]cmds.Command`.

## Compiling to Glazed

Each normalized `VerbSpec` compiles into:

* one `cmds.CommandDescription`
* zero or more `schema.Section`
* one concrete `JSVerbCommand` implementing `cmds.GlazeCommand`

Compilation rules:

* `VerbSpec.CommandName` → `CommandDescription.Name`
* `VerbSpec.Parents` → `CommandDescription.Parents`
* `Short/Long` → help text
* each `SectionSpec` → `schema.NewSection(...)`
* each flag → `fields.New(...)`
* `positional: true` compiles to `IsArgument`
* shared sections are attached before local sections
* default scalar params go into `schema.DefaultSlug`

Do **not** invent a second field type system. The JS metadata `type` should accept the existing Glazed field type strings directly.

## Runtime invocation design

Each generated command is backed by a `JSVerbCommand`:

```go
type JSVerbCommand struct {
    *cmds.CommandDescription
    Verb    *model.VerbSpec
    Factory *engine.Factory
    RootDir string
}
```

It implements `RunIntoGlazeProcessor`.

### Invocation flow

1. Create a fresh runtime from the factory.
2. Install sentinel no-ops.
3. Require the target script through an overlay loader.
4. Resolve the selected top-level function.
5. Build JS call args from the binding plan.
6. Call the function.
7. Convert the result to rows.

### Why an overlay loader

This bit is crucial.

Top-level functions in a CommonJS module are not automatically exported. So the runner needs a way to call them **without requiring the user to write `module.exports = { ... }`**.

The clean answer is:

* load the target file through a custom `require.WithLoader(...)`
* for command source files, return:

```text
[preamble with sentinel no-ops]
[original source]
[appendix that registers discovered top-level functions in a runtime-global registry]
```

Example appendix:

```js
globalThis.__glazedVerbRegistry = globalThis.__glazedVerbRegistry || {};
globalThis.__glazedVerbRegistry["github/issues.js"] = {
  listIssues: typeof listIssues === "function" ? listIssues : undefined
};
```

Then after `require("./github/issues.js")`, Go reads:

```go
globalThis.__glazedVerbRegistry["github/issues.js"]["listIssues"]
```

This keeps relative `require("./other-file")` working, because the file still executes as a real module. It also avoids mutating `module.exports`, which gets weird fast if a script reassigns it to something exotic.

## Result adaptation

All generated commands should implement `cmds.GlazeCommand`.

Default result adapter:

* `undefined` / `null` → no rows
* plain object → one row
* array of plain objects → one row per object
* array of primitives → one row per value, using `value`
* primitive → one row with field `value`

Examples:

```js
return { id: 1, name: "alice" }         // 1 row
return [{id:1}, {id:2}]                 // 2 rows
return "hello"                          // 1 row: { value: "hello" }
return 42                               // 1 row: { value: 42 }
```

For v1, that is enough. Later, add a native `glazed` JS module for streaming rows directly if needed.

## Async support

Support both sync and async functions.

Runtime behavior:

* if return value is not a Promise, adapt directly
* if it is a Promise, await settlement through the existing event-loop/owned-runtime machinery, then adapt the resolved value

That gives you `async function` support without changing the command model.

## Validation and diagnostics

Discovery should return structured diagnostics with file and line numbers.

Reject these with clear errors:

* `__verb__("foo")` but no matching top-level function `foo`
* `params.limit` override when function has no `limit` parameter
* `bind:"section"` without `section`
* `useSections:["auth"]` when `auth` is not defined
* destructured param without explicit binding
* unsupported metadata literal form
* duplicate command path
* duplicate short flag within one command
* positional flag inside a non-default section if you do not want to support that in v1

I would make normalization accumulate all file-local diagnostics before failing, so the user gets one useful report instead of a single whack-a-mole error.

## Security model

Important plain statement: this is **not** a sandbox for untrusted code.

Good practices:

* discovery never executes JS
* scanning is limited to an allowed root
* path traversal is rejected
* command execution only loads scripts under the provisioned root
* native modules exposed to JS are explicit and whitelisted by the engine factory

Bad fantasy to avoid:

* “it’s in Goja so it must be safe”
* no, the JS can still do whatever the host runtime and native modules allow

So the design should borrow the allowed-root pattern already used in the jsdoc extractor, but it should still document that executing third-party scripts is a trust boundary.

## Proposed package layout

```text
pkg/
  jsverbs/
    model/
      raw.go          // RawFileSpec, RawPackage, RawVerb, RawSection
      model.go        // VerbSpec, SectionSpec, BindingSpec, Registry
      store.go        // indexes, collision checks

    extract/
      extract.go      // tree-sitter extraction
      functions.go    // top-level function discovery
      literals.go     // JS literal -> JSON-ish decoding
      templates.go    // doc frontmatter parsing

    normalize/
      normalize.go    // raw -> normalized registry
      merge_docs.go   // optional merge with jsdoc summaries/prose

    compile/
      glazed.go       // VerbSpec -> CommandDescription + Sections
      bindings.go     // Binding plan compilation

    runtime/
      command.go      // JSVerbCommand
      loader.go       // overlay source loader
      invoke.go       // call selected function
      marshal.go      // glazed values -> JS args
      adapt.go        // JS return -> rows
      context.go      // runner context object

    loaders/
      loader.go       // implements glazed loaders.CommandLoader
      attach.go       // optional root attachment helper
```

If you want to avoid premature abstraction, skip a shared `pkg/jssentinel` package initially. Start with `pkg/jsverbs/extract`, copy the few helpers you need from `pkg/jsdoc/extract`, and factor later once tests prove the stable seams.

## Implementation plan

### Step 1: model + extraction

Build raw extractor first.

Tests:

* simple top-level function discovery
* arrow/function-expression discovery
* `_private` exclusion
* `__package__`, `__section__`, `__verb__` literal parsing
* `doc` block attachment
* line number/source file capture

### Step 2: normalization

Convert raw data into final `VerbSpec`.

Tests:

* package path defaults
* file-stem fallback
* default section creation from params
* section reuse
* param override merge
* collision detection
* destructuring validation

### Step 3: Glazed compilation

Compile into `CommandDescription` and sections.

Tests:

* flag names/types/defaults
* positional args
* section prefixes
* short flag collisions
* help text precedence

### Step 4: runtime invocation

Implement overlay loader + function lookup + binding + result adapter.

Tests:

* call top-level declaration
* call arrow function
* section-bound object param
* `bind:"all"`
* `bind:"context"`
* relative `require()` still works
* async function returns Promise
* row adaptation

### Step 5: directory loader

Implement `loaders.CommandLoader`.

Tests:

* one file with multiple verbs
* recursive directory scan
* duplicate paths across files
* source metadata on commands

### Step 6: optional polish

* merge with `__doc__` for help text
* package/group command help
* watch mode / hot reload for long-lived hosts
* native `glazed` JS module for streaming rows

## Design choices I would explicitly reject

### 1. Sidecar YAML for JS commands

It works, but it drifts. Metadata next to code is the whole point here.

### 2. Full JSDoc comment parsing

That is a much bigger parser problem for much less payoff. You already have a sentinel-based extraction style. Use it.

### 3. Runtime-only introspection

Do not execute every script at startup just to discover commands. That is slower, less predictable, and much harder to secure or debug.

## Bottom line

The clean design is:

* **static sentinel extraction**
* **auto-discovery of public top-level functions**
* **normalization into a command registry**
* **compilation into ordinary Glazed commands**
* **runtime invocation through an overlay loader that exposes top-level functions without requiring explicit exports**

That gives you the jsdocex authoring feel, preserves locality of metadata, fits the Glazed command model, and avoids turning startup into a dynamic-code séance.

[1]: https://jsdoc.app/ "https://jsdoc.app/"
