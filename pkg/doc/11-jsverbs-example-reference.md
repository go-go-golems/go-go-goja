---
Title: "jsverbs-example reference"
Slug: jsverbs-example-reference
Short: "Reference for metadata, inference rules, bindings, output modes, and runtime behavior in the JavaScript verb prototype."
Topics:
- goja
- glazed
- reference
- metadata
- runtime
Commands:
- jsverbs-example
Flags:
- --dir
- --log-level
IsTopLevel: false
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

This page is a compact reference for the JavaScript verb subsystem. Use it after you already understand the main flow and need exact rules instead of a tutorial narrative.

Think of this page as the answer key for the developer guide. The guide explains the shape of the system and why it exists. This page tells you the exact supported inputs and behaviors the current implementation promises. When you change behavior, this is one of the places that should move with the code so future readers can distinguish deliberate design from accidental current behavior.

## Discovery rules

Library entrypoints:

- `ScanDir(root, ...)`
- `ScanFS(fsys, root, ...)`
- `ScanSource(path, source, ...)`
- `ScanSources(files, ...)`

Command-compilation entrypoints:

- `registry.Commands()`
- `registry.CommandsWithInvoker(invoker)`
- `registry.CommandForVerb(verb)`
- `registry.CommandForVerbWithInvoker(verb, invoker)`
- `registry.CommandDescriptionForVerb(verb)`

Runtime entrypoints:

- default internal `registry.invoke(...)` via `Commands()` / `CommandForVerb(...)`
- `registry.InvokeInRuntime(ctx, runtime, verb, parsedValues)` for caller-owned runtimes
- `registry.RequireLoader()` for composing the scanned-source loader into a host runtime

The example runner uses `ScanDir(...)` plus the default `registry.Commands()` path, but the package itself is no longer limited to disk directories or to runtime-owning command wrappers.

Supported function declarations:

- top-level `function name(...) {}`
- top-level `const name = (...) => {}`
- top-level `const name = function(...) {}`

Ignored for command discovery:

- nested functions
- object methods
- functions whose names start with `_`

Public functions are auto-exposed unless an explicit `__verb__` definition overrides them.

This rule gives the subsystem a useful default: simple files can become command trees without much ceremony. At the same time, explicit `__verb__` metadata remains the escape hatch when naming, grouping, or output semantics need to be different from the inferred default.

## Recognized sentinels

- `__package__({...})`
- `__section__("slug", {...})`
- `__verb__("functionName", {...})`
- `doc\`...\``

The sentinels are statically extracted at scan time and installed as runtime no-ops during execution.

That means the same token can matter in two different ways: at scan time it carries meaning for command construction, and at runtime it only needs to exist so requiring the source does not fail. New developers sometimes conflate those roles, so it is worth being explicit that most sentinel behavior is compile-time metadata, not dynamic runtime logic.

Metadata values are parsed as a strict literal subset. Supported metadata values are:

- object literals
- array literals
- quoted strings
- template strings without substitutions
- numbers
- `true`, `false`, `null`

Unsupported metadata values include:

- function calls
- identifiers as values
- template substitutions
- computed keys
- spreads
- other dynamic expressions

## `__package__` fields

Currently used package metadata:

- `name`
- `short`
- `long`
- `parents`
- `tags`

Package metadata mainly affects default command grouping.

In other words, package metadata shapes where commands live in the tree more than how the function itself is called. If you are debugging command hierarchy or unexpectedly nested verbs, package metadata is one of the first places to inspect.

## `__section__` fields

Supported section fields:

- `title`
- `name`
- `description`
- `fields`

Section field definitions support the same field keys used under `__verb__`.

Sections are the main mechanism for sharing schema across multiple verbs in one file. They are especially useful when several commands conceptually operate over the same group of filters or options and you want the command line shape to reflect that shared structure.

Scope rule:

- `__section__` defines file-local sections only.
- Cross-file shared sections are registered from Go with `Registry.AddSharedSection(...)` or `Registry.AddSharedSections(...)`.
- Resolution order is:
  - file-local section with that slug, then
  - registry-level shared section with that slug.

`require()` does not import metadata from another file. It only affects runtime code loading.

## `__verb__` fields

Supported verb-level fields:

- `command`
- `name`
- `short`
- `long`
- `parents`
- `tags`
- `sections`
- `useSections`
- `fields`
- `output`
- `outputMode`
- `mode`

Output aliases:

- `glaze`
- `table`
- `structured`
- `text`
- `raw`
- `writer`
- `plain`

Multiple aliases exist mostly for ergonomics while the subsystem is still evolving. In practice, using one canonical spelling per codebase is easier to read. For now, `output: "text"` and the default structured mode are the clearest choices.

## Field definition keys

Supported field metadata keys:

- `type`
- `help`
- `short`
- `bind`
- `section`
- `default`
- `choices`
- `required`
- `argument`
- `arg`

These keys are intentionally close to Glazed concepts. The subsystem is not inventing a separate CLI schema language; it is translating a small JavaScript-friendly metadata format into existing Glazed schema structures.

## Field type mapping

Supported field types currently map to Glazed like this:

- `string` -> string field
- `bool` or `boolean` -> bool field
- `int` or `integer` -> integer field
- `float` or `number` -> float field
- `stringList`, `list`, `[]string` -> string-list field
- `choice` -> choice field
- `choiceList` -> choice-list field

If `choices` is set and no type is provided, the system treats the field as `choice`.

This is another example of the system preferring useful defaults over forcing constant ceremony. When the metadata already expresses a closed set of valid values, treating the field as a choice is usually what the author intended anyway.

## Inference rules

If a parameter has no explicit field metadata:

- identifier parameter -> inferred string field
- rest parameter -> inferred string-list argument

Requiredness rule:

- arguments are required by default unless a default value is supplied

That requiredness rule is one of the most visible user-facing defaults in the subsystem. It is there to make the generated CLI behave like a serious command, not like an unconstrained JavaScript function call.

Parameters using object or array patterns are not treated as normal inferred positional values. They require an explicit `bind`, because the package now prefers one explicit binding contract over guesswork.

## Binding modes

No bind:

- use a normal field or inferred field value

`bind: "all"`:

- pass one object containing every resolved field value

`bind: "context"`:

- pass one object containing:
  - `verb`
  - `function`
  - `module`
  - `sourceFile`
  - `rootDir`
  - `values`
  - `sections`

`bind: "<section>"`:

- pass one object with values from that named section

Bindings are where a lot of expressiveness comes from. They let the generated CLI remain idiomatic on the Go side while still letting JavaScript authors think in terms of cohesive objects instead of only flat parameter lists.

Internally, bindings are resolved through a shared binding plan that is consumed by both command compilation and runtime invocation. The user-visible implication is simple: invalid binds should now fail consistently instead of becoming mismatched behavior between those two phases.

## Output modes

`glaze` mode:

- command compiles as `GlazeCommand`
- objects become rows
- arrays become many rows
- primitives become `{value: ...}`

`text` mode:

- command compiles as `WriterCommand`
- strings are written directly
- `[]byte` is written directly
- all other values fall back to pretty JSON

That JSON fallback is a guardrail rather than a recommendation. If a text verb consistently returns structured objects, it is usually a sign that the command should probably be a structured verb instead.

## Promise handling

If a JS function returns a `Promise`, the runtime polls promise state through the runtime owner and waits until the promise is fulfilled or rejected.

Behavior:

- fulfilled -> exported value continues through normal output handling
- rejected -> command returns an error
- pending with canceled context -> command stops with context error

Promise support matters because it allows the runtime path to remain usable for more realistic JavaScript helpers instead of only synchronous utility functions. At the same time, the waiting logic is intentionally simple and explicit so developers can reason about how long-running or failing async code surfaces back into the CLI.

The current polling implementation is deliberately retained as version-1 behavior. It is documented as simple and temporary rather than as the final async design.

## Relative module loading

Execution uses a source loader that serves the source already stored in the registry and injects an overlay around it. The overlay captures top-level functions while preserving normal relative `require()` behavior.

Resolution currently tries:

- the registry-backed module path for scanned files
- relative lookups resolved by the underlying `require()` resolver against that scanned module path

This means relative helpers still work for directory scans, `fs.FS` scans, and raw in-memory source sets, as long as the referenced helper module is part of the scanned source set.

## Diagnostics

The scanner records diagnostics on the registry and, by default, fails with `ScanError` when error diagnostics are present.

Practical implications:

- malformed metadata is not silently dropped anymore
- callers can inspect `registry.Diagnostics`
- callers can inspect `registry.ErrorDiagnostics()`
- `ScanOptions.FailOnErrorDiagnostics` controls whether error diagnostics become a returned scan error

This is one of the most important behavioral changes from the original prototype because it turns missing-command debugging into an explicit scanner contract.

## Testing pattern

Preferred testing structure:

- use `testdata/jsverbs` for fixtures
- use `ScanDir(...)`, `ScanFS(...)`, or `ScanSource(...)` depending on what source origin you want to validate
- use `registry.Commands()` for default runtime-owning command compilation, or `registry.CommandsWithInvoker(...)` when you need to verify host-owned execution hooks
- execute structured verbs through a capture processor
- execute text verbs through a `strings.Builder`

That split in test style mirrors the split in command interfaces. If you add a new output behavior and the tests no longer read naturally, it is worth checking whether the command abstraction itself is still clean.

## Help integration

The example runner loads the shared `pkg/doc` help pages into its help system.

That means:

- adding a new `*.md` file in `pkg/doc` makes it reusable for other commands
- `jsverbs-example` automatically benefits from the shared doc package
- no extra registration code is needed unless the docs move to a different package

This is a good pattern to preserve because it keeps reusable developer documentation in a package-level location instead of hiding it under one command. A new contributor should be able to discover implementation and help content close to the reusable library code, not only inside one example binary.

## See Also

- `glaze help jsverbs-example-overview`
- `glaze help jsverbs-example-developer-guide`
- `glaze help jsverbs-example-fixture-format`
