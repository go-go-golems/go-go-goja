---
Title: "jsverbs-example developer guide"
Slug: jsverbs-example-developer-guide
Short: "Intern-friendly guide to how JavaScript verb scanning, command compilation, and runtime execution work in go-go-goja."
Topics:
- goja
- glazed
- developer-guide
- javascript
- commands
- architecture
Commands:
- jsverbs-example
- jsverbs-example list
- jsverbs-example help
Flags:
- --dir
- --log-level
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

This page is the main onramp for a new developer working on the JavaScript-to-Glazed command path in `go-go-goja`.

It explains the system from the outside in: what problem it solves, which files matter, how data flows through the scanner and runtime, where to start reading, how to add a new capability safely, and how to debug failures without guessing.

The most important thing to understand up front is that this subsystem is not “a JavaScript runner with some CLI glue attached.” It is a command-construction pipeline. The JavaScript source is treated first as declarative input that describes commands, and only later as executable code that implements those commands. That distinction explains many of the design choices in the code. We scan source without executing it, convert discovered metadata into Glazed command descriptions, and only after the command line has been parsed do we enter the runtime and invoke a function.

That means a new developer should think in two mental models at the same time. The first model is static: files, functions, sentinels, sections, fields, command paths. The second model is dynamic: runtime factories, `require()` loaders, argument marshalling, promise waiting, and output rendering. Most bugs happen when those two models drift apart. For example, a function may be discovered correctly at scan time but invoked with the wrong argument shape at runtime, or a piece of metadata may be parsed correctly but never applied when the Glazed command description is built.

The hardening pass after the initial prototype tightened exactly those drift-prone seams. Metadata parsing is stricter, scanner diagnostics are explicit, raw and `fs.FS` inputs are supported in addition to directory scanning, and the compile-time/runtime parameter semantics now flow through one shared binding plan. If you read older notes or code-review documents for this subsystem, keep that evolution in mind because some of the rough edges called out there have already been addressed.

## Easy onramp

This section tells you where to start if you are new and want useful progress in the first hour.

### What this subsystem does

The `jsverbs-example` command scans a directory of JavaScript files, discovers top-level functions, turns selected functions into Glazed commands, and executes those functions through `goja`.

The key idea is simple:

- JavaScript authors write ordinary functions.
- Optional metadata sentinels such as `__verb__` and `__section__` refine how those functions should look as CLI commands.
- Go code scans the source tree once, builds command descriptions, and later invokes the right function with parsed Glazed values.

One scope rule is especially important now: `__section__` remains file-local metadata. Cross-file section reuse is implemented on the Go side by registering shared sections on the `Registry` after scanning and before command compilation.

This is valuable because it lets a JavaScript author work in a natural style while still getting the operational benefits of Glazed: schema-driven parsing, grouped flags, help text, table output, alternate renderers, and a conventional Cobra command tree. The author does not need to hand-write Go commands for every function. Instead, they provide a narrow amount of metadata only where the defaults are not good enough.

### The three most important source files

If you only read three files first, read these:

- `pkg/jsverbs/scan.go`
- `pkg/jsverbs/binding.go`
- `pkg/jsverbs/command.go`
- `pkg/jsverbs/runtime.go`

Read them in that order.

Why this order matters:

- `scan.go` tells you what the system believes exists.
- `binding.go` tells you how that discovered metadata becomes one shared command/runtime contract.
- `command.go` tells you how that contract becomes a Glazed command.
- `runtime.go` tells you how execution actually crosses the Go/JS boundary.

### The fastest way to build intuition

Run these commands from the workspace root:

```bash
go test ./go-go-goja/pkg/jsverbs ./go-go-goja/cmd/jsverbs-example
go run ./go-go-goja/cmd/jsverbs-example --dir ./go-go-goja/testdata/jsverbs list
go run ./go-go-goja/cmd/jsverbs-example --dir ./go-go-goja/testdata/jsverbs basics greet Manuel --excited
go run ./go-go-goja/cmd/jsverbs-example --dir ./go-go-goja/testdata/jsverbs basics banner Manuel
```

These four commands show the whole lifecycle:

- tests validate the scan and execution paths,
- `list` shows discovery,
- `greet` shows structured table output,
- `banner` shows plain text writer output.

If you are onboarding somebody new, these commands are worth running live while the source is open in another pane. The `list` command tells you what the scanner believes. The structured command tells you what the Glazed layer built. The text command tells you where the subsystem deliberately stops being table-oriented. That small sequence gives a faster mental picture than reading implementation files cold.

### How to read the fixture tree

The test fixture directory at `testdata/jsverbs` is the shortest path to understanding supported behavior.

Use it as a map:

- `basics.js` shows public functions, explicit verb metadata, file-local sections, `bind: "all"`, `bind: "context"`, structured output, and text output.
- `advanced/numbers.js` shows integer arguments, async results, and rest parameters.
- `nested/with-helper.js` and `nested/sub/helper.js` show relative `require()` behavior.
- `packaged.js` shows `__package__` metadata and automatic exposure of public functions.

The fixture tree is still disk-based because it is meant to be browsed by humans and exercised by the example runner. The package itself is now broader than that. It can also scan a generic `fs.FS` or a raw list of in-memory source files. That matters when you move from “example runner” thinking to “library API” thinking. The fixture tree teaches the metadata format and runtime behavior; it does not define the full list of supported source origins anymore.

When you are unsure how a feature should behave, start by adding or changing a fixture first and then make the implementation satisfy it.

This is not just a testing convenience. It is a design discipline. The fixture tree acts as the most concrete form of subsystem documentation because it shows what JavaScript authors are actually expected to write. A fixture is harder to misread than a prose sentence and easier to stabilize than an informal code comment. In practice, the safest development loop is: write or update a fixture, make the scanner see it, make the compiler shape it, and then make the runtime execute it.

## System at a glance

The system has five stages:

```text
JavaScript files
    |
    v
scan.go
  discover functions + sentinels
    |
    v
binding.go
  resolve parameter + field + section binding plan
    |
    v
command.go
  build Glazed command descriptions
    |
    v
runtime.go
  invoke selected JS function in goja
    |
    v
CLI output
  structured rows or plain text
```

Each stage has one job:

- scanning answers “what functions and metadata exist?”
- binding planning answers “what is the one agreed contract between schema and invocation?”
- command building answers “what CLI shape should each function have?”
- runtime invocation answers “how do we call the function correctly?”
- output handling answers “should this become rows or plain text?”

It is useful to treat these as hard boundaries. Discovery should not depend on runtime execution. Runtime invocation should not have to rediscover command metadata. Output rendering should not mutate the meaning of the command itself. When the code respects those boundaries, changes are localized and reasoning stays tractable. When those boundaries blur, the subsystem becomes fragile because behavior starts depending on hidden coupling between phases.

## File map

This section explains which files exist so you do not waste time searching.

### Core implementation

- `pkg/jsverbs/model.go`
  This defines the in-memory model: registry, file specs, verb specs, parameter specs, field specs, diagnostics, source-file inputs, output modes, and registry-level shared-section storage.
- `pkg/jsverbs/scan.go`
  This walks input sources, parses JavaScript with tree-sitter, extracts functions and sentinel metadata, records diagnostics, and finalizes discovered verbs.
- `pkg/jsverbs/binding.go`
  This resolves one shared binding plan so schema generation and runtime invocation do not each invent their own interpretation of parameter semantics.
- `pkg/jsverbs/command.go`
  This turns `VerbSpec` values into real Glazed commands. It decides whether a verb becomes a `GlazeCommand` or a `WriterCommand`.
- `pkg/jsverbs/runtime.go`
  This builds a goja runtime, injects a registry overlay, serves in-memory module source through the runtime loader, maps parsed Glazed values back into JS arguments, waits on promises, and returns results to Go.

These files are intentionally small enough to read in one sitting. That is a good property to preserve. If you add a feature and it feels like it needs to spread arbitrary logic across all files at once, stop and ask whether you are really adding a new concept or only patching over a mismatch. Good additions usually fit cleanly into one stage and then require only a narrow thread through the others.

### Example CLI

- `cmd/jsverbs-example/main.go`
  This is the runnable entrypoint. It scans a directory, registers commands, sets up Glazed logging, and loads the help system.
- `pkg/doc/*.md`
  These are shared Glazed help pages for `go-go-goja`, including the `jsverbs` guide and reference.

The example CLI is not throwaway scaffolding. It is the fastest manual integration harness in the repo for this functionality. If a change only passes package tests but feels awkward in `jsverbs-example`, the design is probably not finished.

### Fixtures and tests

- `testdata/jsverbs/*`
  Example JavaScript trees used for automated validation and interactive experiments.
- `pkg/jsverbs/jsverbs_test.go`
  End-to-end tests for discovery and execution behavior.

## Discovery model

This section explains what the scanner accepts and why it is intentionally conservative.

### Supported function shapes

The scanner currently discovers:

- top-level `function foo(...) { ... }`
- top-level `const foo = (...) => { ... }`
- top-level `const foo = function(...) { ... }`

The scanner does not try to discover every valid JavaScript callable shape. It stays narrow because command registration must be stable and predictable.

That means these are not the main path:

- object methods,
- nested functions,
- dynamically assigned exports,
- runtime-generated functions.

This conservatism is deliberate. Command trees need to be explainable to humans. If discovery were too clever, a developer would no longer be able to predict which functions become commands without mentally executing the file. That would make debugging and code review worse. A slightly narrower discovery model is usually better than a broader but surprising one.

### Supported sentinels

The scanner currently understands:

- `__package__({...})`
- `__section__("slug", {...})`
- `__verb__("functionName", {...})`
- `doc\`...\``

The key design decision is that these are treated as source metadata, not runtime APIs. The scanner reads them statically; the runtime later installs no-op definitions so requiring the source does not crash.

That separation is important for security and correctness. We do not want discovery to execute arbitrary module top-level code just to learn what a command looks like. We also do not want JavaScript authors to think of sentinels as runtime hooks with side effects. They are effectively annotations embedded in real code.

The shared-section feature follows that same rule. `require()` still does not import metadata. If a host application wants one reusable section catalog across many files, it scans first and then calls `Registry.AddSharedSection(...)` or `Registry.AddSharedSections(...)`. During command compilation and runtime binding, the registry resolves sections local-first and shared-second.

The scanner now enforces that the metadata attached to these sentinels is a strict literal subset. This is worth emphasizing because it changes how you should think about authoring metadata. The supported style is “declarative configuration written in JavaScript syntax,” not “arbitrary code that eventually produces an object.”

### Why tree-sitter is used

The scanner uses tree-sitter because the system needs source-aware extraction without executing arbitrary JavaScript during discovery.

That gives three important properties:

- discovery happens without running user code,
- function names and parameter lists come from syntax rather than brittle regexes,
- metadata sentinels can be read from source in a deterministic way.

Tree-sitter is not free; it adds a parser dependency and a source-model layer that a new developer must learn a bit. But the tradeoff is worth it here because the subsystem needs syntax awareness. Once you accept that requirement, a proper parser is much cheaper than inventing a homegrown partial parser through string operations and then slowly discovering edge cases one by one.

That last sentence is no longer hypothetical. The subsystem originally used a JS-to-JSON rewrite shortcut for metadata objects. The hardening pass removed it and replaced it with AST literal parsing precisely because source-text rewriting was the wrong long-term abstraction once tree-sitter was already in the stack.

## Source origins

This section explains a change that is easy to miss if you only use `jsverbs-example`.

The library now supports several input shapes:

- `ScanDir(...)`
- `ScanFS(...)`
- `ScanSource(...)`
- `ScanSources(...)`

All of them normalize into the same internal `FileSpec` and runtime module-path model. That means a future consumer can:

- scan a real directory,
- scan an `embed.FS`,
- synthesize command files in memory,
- and still reuse the same runtime loader and command compiler.

The runtime no longer depends on going back to disk after scanning. It loads source from the registry itself. That is a subtle but important shift because it makes the scanner the source of truth for module content, not just a discovery pass that happens before a second, unrelated disk read.

## Command compilation

This section explains how a discovered verb becomes a usable CLI command.

This stage is where the subsystem starts feeling like “real Glazed” instead of an extraction experiment. By the time a `VerbSpec` reaches `command.go`, the problem is no longer “what did the JavaScript file contain?” but “what command interface should users see?” That is a different problem, and it is why `command.go` deserves to stay focused on schema construction and command semantics.

### Structured output verbs

Most verbs default to structured output. They compile into a Glazed command and their result is normalized like this:

- object -> one row
- array of objects -> many rows
- primitive -> one row with `value`
- `null` or `undefined` -> no rows

This is the right default for data-oriented commands because users immediately get table, JSON, CSV, and other Glazed formatting behavior.

This default also keeps the subsystem aligned with the broader Glazed philosophy: commands should prefer structured data when they naturally produce structured data. If the result is conceptually a record or list of records, preserving that shape gives users far more leverage than flattening it into ad hoc strings.

### Text output verbs

Some commands should act like classic CLI tools and print plain output directly. For those, the verb metadata can declare:

```js
__verb__("banner", {
  output: "text"
})
```

That changes compilation from `GlazeCommand` to `WriterCommand`.

The runtime invocation is the same. Only the Go-side result handling changes:

- strings are written directly,
- `[]byte` is written directly,
- other values are JSON-rendered as a fallback.

That fallback is intentionally pragmatic rather than pure. In a perfect world, text verbs would always return strings. In practice, developers experiment, and having a JSON fallback keeps a text-output verb from becoming unusable the moment it returns an object during refactoring. It is better to see slightly ugly output than no output at all, especially while a feature is still moving.

### Field inference and required arguments

If a parameter has no explicit field metadata, the system infers a simple default:

- identifier parameter -> string field
- rest parameter -> string-list argument

Positional arguments are treated as required unless a default value is declared. This avoids the earlier bad UX where missing arguments could lead to silent `undefined` results.

This rule is worth remembering because it is the point where JavaScript permissiveness meets CLI strictness. JavaScript is happy to call a function with fewer arguments. A CLI should usually not be. The subsystem therefore chooses the CLI interpretation by default: if something is a positional argument, users should be told when it is missing.

The current implementation now reaches that behavior through a shared binding plan rather than through duplicated logic in `command.go` and `runtime.go`. That matters for maintainability. If you change how a parameter should behave, the safest place to start is the binding-plan layer rather than editing compile-time and runtime code separately.

## Runtime model

This section explains what happens when the user actually runs a discovered verb.

The runtime layer is where most “this feels magical” reactions come from, because it has to preserve normal module semantics while still injecting enough bookkeeping for Go to call the right function. The easiest way to stay oriented is to remember that the runtime does not invent a second module system. It rides on top of the existing `require()` path and only adds a thin overlay.

### Why the source overlay exists

The runtime does not call a function by parsing the file again. Instead, it uses the normal `require()` pipeline and injects a small overlay around the source.

That overlay does two things:

- defines no-op sentinel functions such as `__verb__`,
- records top-level discovered functions into `globalThis.__glazedVerbRegistry`.

This matters because it preserves normal module behavior, especially relative `require()` paths, while still letting Go find the function object later.

This is the key design choice that made the prototype practical. A runtime that evaluated stitched-together snippets or bypassed `require()` entirely would quickly break real JavaScript modules. By overlaying the actual source instead, the subsystem can keep relative imports, helper modules, and familiar CommonJS behavior intact.

The overlay still exists in the hardened implementation, but the backing source no longer has to come from the host filesystem. The loader now serves source from the registry’s scanned files, which is what enables raw-source and `fs.FS` use cases while preserving the same relative-module behavior.

### Runtime sequence

The rough runtime sequence is:

```text
build engine factory
  -> create goja runtime
  -> require the target JS file
  -> look up function in __glazedVerbRegistry
  -> convert parsed Glazed values into JS arguments
  -> call function
  -> await Promise if needed
  -> normalize result for Glazed rows or writer output
```

### Parameter binding modes

Bindings control how one JS parameter is populated.

- no `bind`
  The parameter is fed from a normal field or inferred argument.
- `bind: "filters"`
  The parameter receives the named section as an object.
- `bind: "all"`
  The parameter receives every resolved field value as one object.
- `bind: "context"`
  The parameter receives execution metadata such as command path, module path, root directory, full value map, and section map.

These bindings are the bridge between Glazed’s section-based model and JavaScript’s natural object-parameter style.

Bindings are also one of the most important readability tools available to the JavaScript author. If a function really wants a cohesive object like `filters` or a runtime context object, encoding that directly is clearer than pretending every value should arrive as a flat string parameter. A good binding makes both the JS source and the Go-side command model easier to understand.

From the Go side, bindings are now resolved once into a `VerbBindingPlan`. You do not need to memorize the exact struct fields to work on the package, but you should remember the architectural rule: schema generation and runtime invocation should read from the same resolved binding contract. If you find yourself teaching those two phases the same rule separately, you are probably regressing the design.

## Diagnostics and failure handling

This section covers a behavior change that improves day-to-day development substantially.

Malformed metadata is no longer silently dropped. The scanner now records diagnostics on the registry and, by default, returns a `ScanError` when error-level diagnostics are present.

In practice, that means:

- invalid `__verb__` metadata now fails loudly,
- unsupported dynamic expressions inside metadata are rejected explicitly,
- callers can inspect diagnostics instead of guessing why a verb disappeared.

This makes the package stricter, but in a way that saves engineering time. For generated command trees, explicit scanner failures are much easier to debug than missing commands with no explanation.

## Output choice guide

This section helps a new developer choose between structured and text output without overthinking it.

Choose structured output when:

- the result is a record or list of records,
- users may want JSON/CSV/table formatting,
- the command behaves like data retrieval or listing.

Choose text output when:

- the command should print a banner, snippet, script, or prose block,
- table formatting would make the result worse,
- the command should behave like a classic writer command.

If you are unsure, start with structured output. It is easier to downshift one verb to `output: "text"` than to recover Glazed structure later.

That recommendation is partly about user value and partly about future flexibility. Structured output preserves information. Text output usually throws structure away. Once structure is gone, adding a better formatter later is harder because the command contract has already collapsed into prose.

## How to add a new capability safely

This section gives the preferred workflow for extending the subsystem.

### If you want to support a new metadata field

1. Add or update a fixture in `testdata/jsverbs`.
2. Extend the model in `model.go` if needed.
3. Parse the metadata in `scan.go`.
4. If it changes parameter semantics, express it through `binding.go`.
5. Apply the metadata in `command.go` or `runtime.go`.
6. Add assertions in `pkg/jsverbs/jsverbs_test.go`.
7. Run the package tests and one real `go run` command.

This sequence matters because it forces the new behavior to exist in all three forms that matter: source syntax, in-memory model, and user-facing command behavior. Skipping one of those tends to create half-finished features that technically parse but never become useful commands, or commands that work only in tests but are not documented for actual users.

### If you want to support a new parameter shape

Be careful here. Parameter extraction is where convenience can easily become ambiguity.

Preferred order:

1. define the exact supported syntax,
2. add a fixture,
3. decide whether the new shape is inferable or must require explicit `bind`,
4. update `extractParameters` and the shared binding plan together,
5. verify both structured and writer verbs still work.

Parameter support looks deceptively local, but it is one of the few places where scanner, compiler, and runtime all have to agree. That is why this kind of change deserves extra care. A parameter shape that is easy to extract but hard to marshal, or easy to marshal but hard to explain to users, is usually not worth supporting.

### If you want to support a new output mode

Treat output mode as a command-compilation concern first, not a runtime concern.

Ask:

- does this verb still use Glazed parsing?
- does it need Glazed formatting?
- does it need a new command interface or only a new renderer?

In many cases, a new output style should probably be implemented as a formatter on top of structured rows, not as a brand new command type.

This is a good place to raise the technical bar. New output modes should earn their complexity. A subsystem with too many command kinds becomes hard to reason about because each one drags new assumptions into parsing, help text, and execution. If a formatter can solve the problem, prefer that. If it cannot, document clearly why a new command interface is justified.

## Common debugging path

This section explains how to debug without jumping randomly between files.

The main habit to build here is phase isolation. Ask first whether the issue is in discovery, compilation, runtime invocation, or output rendering. Once you answer that, the search space collapses dramatically. Most time is wasted not on hard bugs, but on debugging the wrong phase.

When a verb is missing:

1. run `jsverbs-example --dir ... list`
2. if it is absent, debug `scan.go`
3. if it is present but malformed, debug `command.go`

When a verb is listed but crashes:

1. run with `--log-level debug`
2. inspect the fixture file and relative imports
3. debug `runtime.go`

When a verb runs but output looks wrong:

1. determine whether it is a structured or text verb
2. for structured verbs, inspect row normalization in `command.go`
3. for text verbs, inspect text rendering fallback in `command.go`

## Maintenance checklist

Use this checklist before considering a change complete:

- fixture added or updated
- automated test added or updated
- help docs updated if the developer-facing behavior changed
- `go test ./go-go-goja/pkg/jsverbs ./go-go-goja/cmd/jsverbs-example`
- at least one manual `go run` validation for the changed path

The manual run is not optional ceremony. This subsystem is deeply user-facing: command names, help text, grouped flags, runtime behavior, and output shape all show up at the CLI boundary. Package tests are necessary, but they are not a substitute for looking at the actual command once.

## Troubleshooting

| Problem | Cause | Solution |
| --- | --- | --- |
| A function should be exposed but is not listed | It is not top-level, starts with `_`, or is not one of the supported declaration forms | Move it to top level or add a supported declaration form |
| Relative `require()` works in Node but not here | The helper file is outside the scanned tree or the path resolution assumption does not match the overlay loader | Keep helpers inside the scanned tree and verify the resolved module path |
| A parameter gets `null` or empty values unexpectedly | The parameter was inferred differently than you expected or needed a section bind | Add explicit field metadata and, if needed, a `bind` |
| A text verb prints JSON instead of raw text | The JS function returned a non-string value in `output: "text"` mode | Return a string or intentionally rely on the JSON fallback |

## See Also

- `glaze help jsverbs-example-overview`
- `glaze help jsverbs-example-fixture-format`
- `glaze help jsverbs-example-reference`
