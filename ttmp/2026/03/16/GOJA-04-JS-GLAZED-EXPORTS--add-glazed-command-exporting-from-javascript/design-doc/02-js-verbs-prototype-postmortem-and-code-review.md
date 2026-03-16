---
Title: JS verbs prototype postmortem and code review
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
    - Path: cmd/jsverbs-example/main.go
      Note: |-
        Example runner bootstrapping and help/logging integration reviewed in detail here
        Example runner bootstrap and logging/help integration reviewed for CLI ergonomics
    - Path: pkg/doc/doc.go
      Note: Shared Glazed help loader used by the example runner
    - Path: pkg/jsverbs/command.go
      Note: |-
        Command-compilation and output-adaptation logic reviewed in detail here
        Command compilation and output adaptation code reviewed for duplicated policy and ergonomics
    - Path: pkg/jsverbs/jsverbs_test.go
      Note: |-
        Test coverage and test-shape analysis for the prototype
        Test coverage reviewed for happy-path strength and failure-path gaps
    - Path: pkg/jsverbs/runtime.go
      Note: |-
        Runtime invocation and overlay-loader implementation reviewed in detail here
        Runtime bridge reviewed for overlay loader
    - Path: pkg/jsverbs/scan.go
      Note: |-
        Static scanner and metadata normalization logic reviewed in detail here
        Static scanner and metadata parsing code reviewed for brittleness and cleanup opportunities
ExternalSources: []
Summary: Detailed postmortem and code review of the jsverbs prototype added since origin/main, covering architecture, strengths, weaknesses, non-idiomatic choices, cleanup priorities, and intern-oriented implementation guidance.
LastUpdated: 2026-03-16T16:24:34.516454875-04:00
WhatFor: Provide an evidence-backed review of the jsverbs prototype that explains both how it works and where it should be cleaned up before being treated as production-quality infrastructure.
WhenToUse: Use when onboarding a new engineer to the jsverbs prototype, reviewing the current implementation against Go and Glazed idioms, or planning the next cleanup/refactor pass.
---


# JS verbs prototype postmortem and code review

## Executive Summary

This document reviews everything added on the current branch since `origin/main`, which at the time of writing is one feature commit:

- `4e4e893` - `Add jsverbs prototype runner and shared docs`

The new code introduces a proof-of-concept subsystem in `pkg/jsverbs` that scans JavaScript files, discovers top-level functions and metadata sentinels, compiles them into Glazed commands, and executes them through a goja runtime with a source-overlay loader. The branch also adds an example runner in `cmd/jsverbs-example`, fixture coverage in `testdata/jsverbs`, shared Glazed help pages under `pkg/doc`, and ticket-local research artifacts under `ttmp/.../GOJA-04-JS-GLAZED-EXPORTS--...`.

The prototype is useful and directionally correct. It proves that the overall product shape is viable:

- static JS discovery can produce a usable command registry,
- command descriptions can be derived from JS metadata and Glazed schemas,
- runtime invocation can bridge Go and JavaScript without hand-written Go commands,
- relative `require()` still works when source is wrapped by an overlay loader.

At the same time, the prototype is still obviously a prototype. The code works, the tests cover the happy path well enough for exploration, and the help/docs are much better than average for a spike, but several parts are not yet production-grade. The major concerns are not deprecated APIs or obvious syntax mistakes. We already cleaned those up. The real problems are structural:

1. the scanner silently ignores malformed metadata,
2. the metadata parser relies on a handwritten JS-object-to-JSON conversion pass,
3. command-compilation logic and runtime argument-binding logic have to stay manually synchronized,
4. the runtime creates a fresh engine factory for every invocation,
5. promise waiting is implemented as polling,
6. the example runner bootstraps command discovery with manual pre-parse argument inspection.

For a new intern, the right mindset is this:

- treat the current implementation as a strong feasibility prototype,
- trust the tested flows,
- do not mistake the current internal shape for the final architecture,
- focus first on making metadata handling stricter and cleaner before adding more features.

## Scope and Review Method

This review covers all branch changes relative to `origin/main`, with primary attention on executable code and the shared help/documentation path. The changed code surface is:

- `cmd/jsverbs-example/main.go`
- `pkg/jsverbs/model.go`
- `pkg/jsverbs/scan.go`
- `pkg/jsverbs/command.go`
- `pkg/jsverbs/runtime.go`
- `pkg/jsverbs/jsverbs_test.go`
- `pkg/doc/08-jsverbs-example-overview.md`
- `pkg/doc/09-jsverbs-example-fixture-format.md`
- `pkg/doc/10-jsverbs-example-developer-guide.md`
- `pkg/doc/11-jsverbs-example-reference.md`
- `testdata/jsverbs/...`

This review also references pre-existing framework seams because those explain whether the new code is idiomatic and where future cleanup should attach:

- `go-go-goja/engine/factory.go`
- `glazed/pkg/cmds/cmds.go`
- `glazed/pkg/cli/cobra.go`
- `glazed/pkg/cmds/logging/init.go`
- `go-go-goja/pkg/doc/doc.go`

The analysis method was:

1. inspect the exact `origin/main...HEAD` diff,
2. read the new code with line anchors,
3. compare the implementation choices against existing Goja and Glazed extension seams,
4. identify correctness, maintainability, and idiomatic-Go concerns,
5. turn that into a cleanup-oriented postmortem for a new engineer.

## Easy Onramp For A New Intern

If you are seeing this subsystem for the first time, start with a very simple mental model:

- `scan.go` answers: "What commands exist in this JS tree?"
- `command.go` answers: "How should those commands look to Glazed and Cobra?"
- `runtime.go` answers: "How do we actually call the JS function once the user ran the command?"
- `main.go` answers: "How does the example binary expose all of that at the CLI?"

Read in exactly that order.

Then run these commands from the workspace root:

```bash
go test ./go-go-goja/pkg/jsverbs ./go-go-goja/cmd/jsverbs-example
go run ./go-go-goja/cmd/jsverbs-example --dir ./go-go-goja/testdata/jsverbs list
go run ./go-go-goja/cmd/jsverbs-example --dir ./go-go-goja/testdata/jsverbs basics greet Manuel --excited
go run ./go-go-goja/cmd/jsverbs-example --dir ./go-go-goja/testdata/jsverbs basics banner Manuel
go run ./go-go-goja/cmd/jsverbs-example --dir ./go-go-goja/testdata/jsverbs help jsverbs-example-developer-guide
```

Those commands show the entire lifecycle:

- `list` shows what the scanner discovered,
- `greet` shows a structured Glazed command,
- `banner` shows text-mode output,
- `help ...` shows how the shared `pkg/doc` docs are wired in.

After that, open these files side by side:

- `pkg/jsverbs/scan.go`
- `pkg/jsverbs/command.go`
- `pkg/jsverbs/runtime.go`
- `testdata/jsverbs/basics.js`
- `pkg/jsverbs/jsverbs_test.go`

That combination is enough to understand almost every branch-specific design choice.

## What The System Is

The subsystem can be viewed as a four-stage pipeline:

```text
JS source files
    |
    v
[scan.go]
Static extraction of:
- top-level functions
- __package__
- __section__
- __verb__
- doc`...`
    |
    v
[model.go]
Normalized in-memory registry:
- FileSpec
- FunctionSpec
- VerbSpec
- SectionSpec
- FieldSpec
    |
    v
[command.go]
Compilation into Glazed commands:
- CommandDescription
- GlazeCommand or WriterCommand
    |
    v
[runtime.go]
Per-invocation goja runtime:
- custom require loader
- overlay prelude/suffix
- argument marshalling
- function call
- result adaptation
```

That architecture is sensible because it respects the difference between discovery and execution.

The static stage answers "what is the command tree?" without running user code.
The runtime stage answers "how do we invoke one chosen function?" only after the CLI has already parsed arguments.

This separation is one of the best parts of the prototype. It matches how Glazed works. A Glazed command needs a schema and description before execution. The JS cannot be treated only as dynamic runtime code because the CLI surface has to exist before the command is invoked.

## Architecture Walkthrough With File References

### 1. Registry model

The central data model lives in `pkg/jsverbs/model.go`.

Important types:

- `Registry`
- `FileSpec`
- `FunctionSpec`
- `VerbSpec`
- `SectionSpec`
- `FieldSpec`
- `ParameterSpec`

Why this matters:

- `Registry` is the stable boundary between scanning and command creation.
- `VerbSpec` is the real "unit of command" in the subsystem.
- `FieldSpec` and `SectionSpec` are the bridge between JS metadata and Glazed schema definitions.

Relevant code:

- `Registry` and `FileSpec`: `pkg/jsverbs/model.go:31-51`
- `VerbSpec`: `pkg/jsverbs/model.go:93-105`
- output mode constants: `pkg/jsverbs/model.go:107-110`

This model is mostly clean. It is simple value-carrying state with a few helper methods. That is idiomatic enough and easy to reason about.

### 2. Static scanner

The scanner lives in `pkg/jsverbs/scan.go`.

The top-level flow is:

1. walk the directory tree,
2. read JS files,
3. parse them with tree-sitter JavaScript,
4. collect top-level function declarations and metadata sentinel calls,
5. normalize those into `FileSpec`,
6. finalize all discovered verbs into the registry.

Relevant code:

- directory walk and parser setup: `pkg/jsverbs/scan.go:17-133`
- finalize discovered verbs: `pkg/jsverbs/scan.go:136-220`
- top-level extractor dispatch: `pkg/jsverbs/scan.go:255-338`
- sentinel handlers: `pkg/jsverbs/scan.go:340-446`
- parameter parsing: `pkg/jsverbs/scan.go:509-563`
- metadata object conversion: `pkg/jsverbs/scan.go:594-833`

This file is doing the most work and is also the least elegant part of the implementation. That is not surprising. Scanners often start out as a mixture of syntax walking, normalization, and fallback heuristics. But it is the first place that should be cleaned up.

### 3. Command compiler

The command compiler lives in `pkg/jsverbs/command.go`.

Its job is:

1. turn `VerbSpec` into a `cmds.CommandDescription`,
2. build Glazed sections and fields,
3. choose whether a verb becomes a structured `GlazeCommand` or a text `WriterCommand`,
4. adapt JS return values into rows or text.

Relevant code:

- command wrapper types: `pkg/jsverbs/command.go:22-35`
- registry-to-command conversion: `pkg/jsverbs/command.go:37-70`
- schema assembly: `pkg/jsverbs/command.go:72-239`
- field inference and definition building: `pkg/jsverbs/command.go:241-390`
- structured output path: `pkg/jsverbs/command.go:393-408`
- text output path: `pkg/jsverbs/command.go:410-447`
- row conversion helpers: `pkg/jsverbs/command.go:450-513`

This file is conceptually sound, but it currently owns too many policies at once:

- section existence,
- field inference,
- defaults normalization,
- runtime output shape adaptation,
- some cross-checks on parameter binding.

It works, but it is dense. A new contributor can still follow it, but only by reading it carefully from top to bottom.

### 4. Runtime bridge

The runtime bridge lives in `pkg/jsverbs/runtime.go`.

Its job is:

1. create a goja runtime factory,
2. load the JS module through a custom loader,
3. inject a prelude and suffix so sentinel symbols exist and top-level functions are captured,
4. build JS arguments from parsed Glazed values,
5. call the correct function,
6. wait for promise completion if needed.

Relevant code:

- runtime construction and call path: `pkg/jsverbs/runtime.go:21-86`
- argument binding: `pkg/jsverbs/runtime.go:88-155`
- custom loader and overlay injection: `pkg/jsverbs/runtime.go:157-221`
- module resolution: `pkg/jsverbs/runtime.go:223-248`
- promise waiting: `pkg/jsverbs/runtime.go:250-300`

This file proves the feasibility of the runtime approach. It also contains one of the most important design insights of the branch: the static registry does not need to rewrite modules permanently. It only needs a runtime loader that temporarily decorates source.

### 5. Example runner and shared docs

The example runner lives in `cmd/jsverbs-example/main.go`. It loads shared docs from `pkg/doc`.

Relevant code:

- runner bootstrap: `cmd/jsverbs-example/main.go:20-88`
- manual directory discovery: `cmd/jsverbs-example/main.go:91-104`
- shared help loader: `pkg/doc/doc.go:8-12`

The binary is good enough as a manual test harness and demo vehicle. It is not yet the final product shape, but it does its job.

## What Went Well

This prototype has several strong points that should be preserved even if the internal code is refactored.

### The architecture is split along the right seams

The split between scanner, registry, compiler, and runtime is good. That maps well onto the surrounding framework APIs:

- `engine.NewBuilder()` and `Factory.NewRuntime(...)` already separate runtime composition from runtime instantiation in `engine/factory.go:31-179`.
- `cmds.NewCommandDescription(...)` expects schema-first command construction in `glazed/pkg/cmds/cmds.go:221-232`.
- `cli.AddCommandsToRootCommand(...)` and the Glazed Cobra run path expect commands to exist before execution in `glazed/pkg/cli/cobra.go:232-260`.

This is the main reason the prototype feels coherent even where the internals are rough.

### The end-to-end experience is real, not mocked

The tests and example runner exercise the actual stack:

- JS fixtures are scanned from disk,
- Glazed parses their arguments,
- goja executes them,
- helper modules are resolved through `require()`,
- output is rendered through structured rows or text.

That matters. Many spikes stop at an in-memory proof or fake data model. This one crosses the whole boundary.

### Shared help pages were moved to the right place

The docs now live under `pkg/doc`, and `pkg/doc/doc.go:8-12` exposes them through a package-level loader.

That is a good move because:

- the docs describe reusable library behavior, not only one example binary,
- other commands can load the same help bundle,
- the example binary stays lightweight.

### Tests cover the most important happy paths

`pkg/jsverbs/jsverbs_test.go` covers:

- explicit metadata,
- inferred public functions,
- section binding,
- `bind: "all"`,
- `bind: "context"`,
- async promise return,
- rest parameters,
- relative helper imports,
- package metadata grouping,
- text output mode.

For a first implementation pass, that is decent coverage.

## Primary Findings

This section is the core code review. Findings are ordered from highest leverage / highest risk to lower-severity cleanup items.

### Finding 1: Metadata parse failures are silently ignored

Severity: high

Where to look:

- `pkg/jsverbs/scan.go:340-420`
- `pkg/jsverbs/scan.go:484-488`

Problem:

The scanner swallows metadata parsing errors for `__package__`, `__section__`, and `__verb__`. If the metadata object cannot be converted or unmarshaled, the handler simply returns. That means the command may disappear, partially degrade, or fall back to inferred behavior without telling the author what happened.

Example:

```go
data := map[string]interface{}{}
if err := e.unmarshalObject(objectNode, &data); err != nil {
    return
}
```

Why it matters:

- Silent fallback is hostile to users and maintainers.
- A typo in metadata can turn into a missing or malformed command with no explanation.
- The more metadata features we add, the worse this failure mode becomes.

Why it is non-idiomatic:

In Go, especially in infrastructure code, silent error suppression is usually only acceptable for truly optional best-effort behavior. This metadata is not optional in the same sense. It defines the command contract. Silent failure here creates hidden state transitions.

Recommended cleanup:

- Collect scanner diagnostics rather than dropping them.
- Either:
  - fail the entire scan on invalid metadata, or
  - record structured warnings and expose them in `Registry`.

Preferred shape:

```text
ScanDir
  -> Registry
     -> Files
     -> Verbs
     -> Diagnostics []Diagnostic
```

Pseudocode:

```go
type Diagnostic struct {
    Severity string
    File     string
    Symbol   string
    Message  string
}

func (e *extractor) handleVerb(argsNode *tree_sitter.Node) {
    name, objectNode := e.namedObjectArgs(argsNode)
    if objectNode == nil {
        e.warn("verb sentinel requires object argument")
        return
    }

    data := map[string]interface{}{}
    if err := e.unmarshalObject(objectNode, &data); err != nil {
        e.errorf("invalid __verb__ metadata for %s: %v", name, err)
        return
    }

    ...
}
```

Intern guidance:

- If you make only one cleanup, make it this one first.
- Strict errors will save far more engineering time than adding one more metadata feature.

### Finding 2: The scanner uses a handwritten JS-object-to-JSON converter

Severity: high

Where to look:

- `pkg/jsverbs/scan.go:594-700`
- `pkg/jsverbs/scan.go:703-833`

Problem:

The metadata parser extracts raw source text for JS object literals, then runs it through `convertJSToJSON(...)`, then unmarshals JSON into Go maps.

This is clever and pragmatic, but also brittle. It is not a real JavaScript value parser. It relies on source-text rewriting rules that will eventually diverge from actual JavaScript grammar.

Why it matters:

- It can mis-handle edge cases such as template expressions, regex literals, computed properties, spread, nested expressions, or more complicated strings.
- It makes metadata support feel larger than it really is. Every new JS syntax shape becomes a parser-maintenance problem.
- It encourages "just one more special case" growth.

Why it is inelegant:

The code already has tree-sitter parse trees. Converting syntax back into text and then reparsing that text with a handmade converter is a classic smell. It means the code has access to a structured syntax tree but is not fully using it.

A new contributor reading this code will immediately ask: "If we already parsed the AST, why are we reparsing object literals through string munging?"

That is the right question.

What the prototype got right:

- It kept the behavior local to metadata parsing.
- It made the happy path work quickly.
- It avoided premature overengineering.

But the next pass should improve this area.

Cleanup options:

1. Best medium-term option:
   Walk object and array AST nodes directly and convert them to Go values.

2. Acceptable short-term option:
   Keep the current approach, but reject unsupported syntax loudly and document the allowed metadata subset explicitly.

Preferred rule:

- metadata should be static and literal-only,
- dynamic expressions inside `__verb__` or `__section__` should be errors, not heuristically accepted.

Pseudocode:

```go
func parseLiteralNode(node *tree_sitter.Node) (interface{}, error) {
    switch node.Kind() {
    case "object":
        return parseObject(node)
    case "array":
        return parseArray(node)
    case "string", "template_string":
        return parseString(node)
    case "true":
        return true, nil
    case "false":
        return false, nil
    case "number":
        return parseNumber(node)
    case "null":
        return nil, nil
    default:
        return nil, fmt.Errorf("unsupported non-literal metadata node: %s", node.Kind())
    }
}
```

### Finding 3: Schema construction and runtime argument binding are manually duplicated

Severity: high

Where to look:

- schema logic: `pkg/jsverbs/command.go:110-226`
- runtime binding logic: `pkg/jsverbs/runtime.go:106-153`

Problem:

The same conceptual policy exists in two different forms:

- `command.go` decides which fields and sections exist and when a parameter requires a bind.
- `runtime.go` decides how those same binds and sections become JS arguments.

That means the subsystem depends on two independently-written interpretations of the same command model staying synchronized.

Why it matters:

- Drift bugs will be subtle.
- One side may accept a parameter shape that the other side cannot invoke correctly.
- Adding new parameter kinds or bind modes becomes risky.

This is the classic prototype risk: the model is conceptually singular, but operationally duplicated.

Example of the problem shape:

```text
VerbSpec + FieldSpec
    |
    +--> buildDescription() chooses sections/fields/requirements
    |
    +--> buildArguments() chooses argument objects and binds
```

A better shape is:

```text
VerbSpec
    |
    v
BindingPlan
    |
    +--> schema builder uses it
    +--> runtime binder uses it
```

Recommended cleanup:

Introduce an explicit intermediate plan, for example:

- `ParameterBindingPlan`
- `VerbExecutionPlan`
- `ResolvedFieldPlan`

This plan would contain:

- source parameter name,
- parameter kind,
- resolved Glazed field name,
- section slug,
- bind mode,
- argument/index order,
- whether the value is positional, section-bound, context-bound, or all-values-bound.

Pseudocode:

```go
type BoundParameter struct {
    ParamName   string
    ParamKind   ParameterKind
    FieldName   string
    SectionSlug string
    BindMode    string // positional, section, all, context
    Rest        bool
}

type VerbPlan struct {
    Verb       *VerbSpec
    Sections   []*schema.SectionImpl
    Parameters []BoundParameter
}
```

Then:

- `buildDescription()` consumes `VerbPlan.Sections`
- `buildArguments()` consumes `VerbPlan.Parameters`

That would make future behavior changes much safer.

### Finding 4: A new engine factory is built on every invocation

Severity: medium-high

Where to look:

- `pkg/jsverbs/runtime.go:21-30`
- comparison seam: `go-go-goja/engine/factory.go:31-179`

Problem:

Each command invocation does:

1. `engine.NewBuilder()`
2. `WithRequireOptions(...)`
3. `WithModules(engine.DefaultRegistryModules())`
4. `Build()`
5. `NewRuntime(ctx)`

That means both the require registry and module registration plan are rebuilt for every command execution.

Why it matters:

- unnecessary overhead,
- unnecessary allocation churn,
- harder testing and configuration,
- harder future extension if callers want custom modules or runtime initializers.

Why it is non-idiomatic relative to the existing engine API:

`engine/factory.go` is already designed around an immutable built `Factory` that can create many runtimes. The prototype is rebuilding the composition instead of reusing it.

This is exactly the kind of thing an initial spike does, and exactly the kind of thing we should clean before hardening.

Better shape:

- `Registry` or a higher-level runner should own a prepared `Factory`.
- `invoke()` should only ask for `factory.NewRuntime(ctx)`.

Pseudocode:

```go
type RuntimeProvider interface {
    NewRuntime(ctx context.Context) (*engine.Runtime, error)
}

type Registry struct {
    RootDir string
    ...
    runtimeFactory *engine.Factory
}

func (r *Registry) prepareRuntimeFactory() error {
    factory, err := engine.NewBuilder().
        WithRequireOptions(require.WithLoader(r.sourceLoader)).
        WithModules(engine.DefaultRegistryModules()...).
        Build()
    if err != nil {
        return err
    }
    r.runtimeFactory = factory
    return nil
}
```

That would improve performance and make dependency injection easier.

### Finding 5: Promise completion is implemented as polling with `time.Sleep`

Severity: medium

Where to look:

- `pkg/jsverbs/runtime.go:250-280`

Problem:

`waitForPromise(...)` loops, checks promise state via `runtime.Owner.Call(...)`, and sleeps for `5 * time.Millisecond` while pending.

This is functional, but inelegant and inefficient. It is a polling bridge, not an event-driven bridge.

Why it matters:

- polling adds latency and jitter,
- frequent command calls can stack unnecessary wakeups,
- the pattern is harder to reason about when async behavior gets more complex.

Why it is acceptable for a prototype:

- it is straightforward,
- it keeps the logic local,
- it is easy to debug,
- it handles context cancellation.

But it should still be called what it is: a temporary bridge.

Future cleanup:

- investigate whether the event loop or promise job queue can expose a less polling-heavy synchronization mechanism,
- or explicitly label this as "prototype polling" in code comments so nobody mistakes it for final architecture.

At minimum, if polling remains, add a comment that explains the reason and tradeoff.

### Finding 6: The example runner uses manual pre-parse discovery of `--dir`

Severity: medium

Where to look:

- `cmd/jsverbs-example/main.go:20-27`
- `cmd/jsverbs-example/main.go:91-104`

Problem:

The example runner scans the target directory before Cobra parses flags, so it manually inspects `os.Args` to find `--dir`. That is why `discoverDirectory(...)` exists.

This works, but it is not very elegant:

- it duplicates a small amount of flag-parsing behavior,
- the command tree is fixed before Cobra owns the CLI state,
- changing the scanning root is a bootstrap concern rather than a normal parsed configuration path.

Why it happened:

This is a real constraint. The command tree must exist before Cobra can execute subcommands, so the scan root has to be known before command registration.

That means the code is not "wrong", but it is awkward.

Possible improvements:

1. Keep it, but comment clearly that this is a bootstrap phase and intentionally pre-Cobra.
2. Introduce a tiny bootstrap command that builds the actual command tree after parsing only root-level bootstrap flags.
3. Separate the example runner into:
   - bootstrap options,
   - registry builder,
   - root command factory.

For an example binary, option 1 is acceptable. For a production binary, option 2 or 3 would be cleaner.

### Finding 7: Error style and helper style are inconsistent

Severity: medium-low

Where to look:

- `pkg/jsverbs/scan.go:12`
- `pkg/jsverbs/scan.go:25`, `92`, `96`, `105`
- rest of new package uses mostly `fmt.Errorf`

Problem:

`scan.go` mixes `github.com/pkg/errors` wrapping with the rest of the package's `fmt.Errorf(... %w ...)` style.

That is not a correctness bug, but it is an avoidable inconsistency in new code. In a small new package, style drift is a signal that the code was assembled incrementally rather than shaped as one coherent unit.

Recommendation:

- standardize on `fmt.Errorf(... %w ...)` unless the repository strongly prefers `pkg/errors` in newly-written code.

This is low priority compared with the scanner and binding issues, but worth cleaning when touching the file.

### Finding 8: The scanner creates a fresh parser per file

Severity: low

Where to look:

- `pkg/jsverbs/scan.go:101-112`

Problem:

`scanFile(...)` constructs a new tree-sitter parser and language setup for every file.

This is simpler than managing parser reuse, and for a small fixture tree it is fine, but it is not ideal for larger trees.

Why it is probably okay for now:

- the subsystem is still exploratory,
- scan sizes are likely small,
- clarity is better than premature pooling.

Future cleanup:

- if scan volume grows, reuse a parser per scan run or hide parser setup inside a dedicated scanner object.

### Finding 9: The test suite is good on happy paths but weak on failure modes

Severity: low-to-medium

Where to look:

- `pkg/jsverbs/jsverbs_test.go:15-212`

What is covered well:

- happy-path discovery,
- successful structured commands,
- successful writer commands,
- section binds,
- async returns,
- helper imports.

What is missing:

- malformed `__verb__` metadata,
- invalid field types,
- duplicate command paths,
- unsupported metadata shapes,
- failure diagnostics,
- promise rejection behavior,
- invalid `bind` references,
- object/array pattern parameters without binds,
- runtime loader failure messages.

This matters because the weakest part of the implementation is metadata strictness, and the tests do not yet pin that behavior down.

Recommended additions:

- a `testdata/jsverbs-invalid/` tree,
- scanner-diagnostic tests,
- command-compilation failure tests,
- promise rejection test,
- invalid bind test.

## Findings Summary Table

| Area | Current state | Why it is a problem | Recommended next move |
| --- | --- | --- | --- |
| Metadata errors | Silent drop | Hidden behavior changes | Add diagnostics or fail-fast |
| Metadata parsing | Handwritten JS-to-JSON conversion | Brittle and hard to extend | Parse AST literals directly |
| Schema vs runtime | Duplicated policies | Drift risk | Introduce binding plan |
| Runtime factory | Built per invocation | Wasteful and harder to configure | Cache factory or inject provider |
| Promises | Polling loop | Inefficient, temporary shape | Replace or document as prototype |
| Runner bootstrap | Manual `os.Args` scan | Awkward CLI bootstrap | Keep with comment or split bootstrap |
| Error style | Mixed wrapping styles | Inconsistent package style | Standardize |
| Parser lifecycle | New parser per file | Extra overhead | Reuse later if needed |
| Tests | Happy-path focused | Weak contract around failures | Add failure fixtures |

## Deprecated APIs And Non-Idiomatic Choices

This section exists because the user explicitly asked for special attention to deprecated and non-idiomatic code.

### Deprecated APIs

Good news:

- the obvious deprecated call (`strings.Title`) was already removed and replaced with `golang.org/x/text/cases`,
- the logging setup uses `logging.InitLoggerFromCobra(...)`, which is the correct current direction according to `glazed/pkg/cmds/logging/init.go:148-220`,
- this branch does not currently introduce known deprecated APIs in the new jsverbs path.

So the real critique is not "deprecated API usage". It is "prototype-grade internal architecture".

### Non-idiomatic patterns worth calling out

The main non-idiomatic choices are:

- silently swallowing parse errors for contract-defining metadata,
- using manual text rewriting where AST-based literal parsing is available,
- duplicating command semantics across compile-time and runtime code paths,
- rebuilding immutable factories per invocation,
- polling promises with `time.Sleep`,
- pre-parsing `os.Args` to bootstrap the command tree.

None of those are embarrassing in a spike. But they should be treated as debt, not as patterns to cargo-cult into future packages.

## Intern-Friendly Data Flow

This is the simplest way to understand execution.

### Static phase

```text
ScanDir(root)
  -> WalkDir(root)
  -> read file
  -> tree-sitter parse
  -> find:
       function greet(name, excited)
       __verb__("greet", {...})
       __section__("filters", {...})
       doc`...`
  -> build FileSpec
  -> finalize VerbSpec
```

### Command-registration phase

```text
Registry.Commands()
  -> for each VerbSpec
      -> buildDescription(verb)
      -> choose:
           OutputModeGlaze -> Command
           OutputModeText  -> WriterCommand
```

### Runtime phase

```text
Run command
  -> Glazed parses CLI args into values.Values
  -> jsverbs.invoke(...)
  -> buildArguments(parsedValues, verb, rootDir)
  -> require(module)
  -> overlay captures top-level functions
  -> call selected function
  -> if Promise: wait
  -> adapt result to rows or text
```

### Concrete example

For `testdata/jsverbs/basics.js`:

```text
JS:
  function listIssues(repo, filters, meta) { ... }
  __verb__("listIssues", {
    sections: ["filters"],
    fields: {
      repo: { argument: true },
      filters: { bind: "filters" },
      meta: { bind: "context" }
    }
  })

CLI:
  jsverbs-example basics list-issues my/repo --state closed --labels bug

Go path:
  ScanDir -> VerbSpec
  Commands -> CommandDescription with default + filters sections
  ParseCommandValues -> values.Values
  buildArguments -> ["my/repo", {"state":"closed","labels":["bug"]}, contextMap]
  goja function call
  rowsFromResult -> Glazed rows
```

That is the whole subsystem in one example.

## Where The Current Design Matches Existing Framework Idioms

The following decisions are good because they align with existing framework seams rather than inventing entirely separate infrastructure.

### It compiles to ordinary Glazed commands

This is exactly right. `glazed/pkg/cmds/cmds.go:17-35` defines `CommandDescription` as the command-registration contract. The prototype uses that instead of creating a parallel command abstraction.

That means:

- Glazed help and schemas work,
- Glazed output modes work,
- Cobra integration works,
- future Glazed middlewares can still apply.

### It uses the go-go-goja engine builder instead of bypassing it

The runtime bridge uses `engine.NewBuilder()` and `Factory.NewRuntime(...)` from `engine/factory.go:31-179`.

That is the correct seam. Even though the current implementation rebuilds the factory too often, it still integrates through the intended extension point:

- custom require options,
- default native modules,
- future runtime initializers.

### It reuses the shared help system pattern

The docs live in `pkg/doc`, and `pkg/doc/doc.go:8-12` exposes them through `LoadSectionsFromFS(...)`. That matches the general Glazed help-loading pattern.

This is a good example of a prototype that managed to land docs in a reusable place instead of burying them inside the example binary.

## Where The Current Design Does Not Yet Feel Production-Ready

This section is intentionally blunt. These are the areas that still feel like a successful branch-local experiment rather than a hardened subsystem.

### The scanner feels permissive in the wrong places

The scanner is strict about some structural facts, like duplicate command paths in `pkg/jsverbs/scan.go:214-220`, but permissive about the more user-visible failure mode: invalid metadata input.

That is backwards. Users can recover from explicit, well-scoped errors. They struggle with silent behavior changes.

### The model contract is under-specified

There is an implicit contract among:

- `ParameterSpec`,
- `FieldSpec`,
- `buildDescription(...)`,
- `buildArguments(...)`.

That contract exists in code, but not in one explicit data structure. That makes review and extension harder than it should be.

### Runtime composition is not injected

The current code bakes in:

- default native modules,
- overlay loader wiring,
- factory construction strategy.

That is okay for an example, but not yet ideal for a reusable package. Eventually this should be injectable or at least configurable.

## Cleanup Roadmap

If we want to harden this subsystem without rewriting it from scratch, this is the order I would recommend.

### Phase 1: Make metadata handling explicit and strict

Goals:

- no silent metadata failures,
- no unsupported dynamic metadata shapes slipping through,
- clear error messages.

Changes:

- add scanner diagnostics,
- fail fast on invalid `__verb__`, `__section__`, `__package__`,
- document the supported literal subset,
- add tests for failure cases.

### Phase 2: Extract a shared binding plan

Goals:

- remove duplicated policy,
- make compile-time and runtime semantics provably aligned.

Changes:

- introduce `VerbPlan` or `BindingPlan`,
- derive sections and runtime argument mapping from the same plan,
- centralize bind validation there.

### Phase 3: Harden runtime composition

Goals:

- reduce per-call overhead,
- improve configurability,
- make runtime behavior more testable.

Changes:

- prepare and reuse an `engine.Factory`,
- optionally inject runtime factory or modules,
- keep overlay loader logic but isolate it behind a small runtime adapter object.

### Phase 4: Replace or clearly isolate polling-based promise waiting

Goals:

- remove temporary-feeling async glue,
- improve clarity around cancellation and promise completion.

Changes:

- investigate event-driven alternatives,
- or keep polling but isolate/document it as temporary.

### Phase 5: Refine the example runner

Goals:

- cleaner bootstrap,
- easier reuse by other binaries,
- less command-tree setup awkwardness.

Changes:

- split bootstrap config from registry build,
- maybe expose a `NewRootCommand(registry *Registry)` helper,
- comment the pre-parse `--dir` bootstrap constraint clearly if it remains.

## Suggested Refactor Shape

One possible future package layout:

```text
pkg/jsverbs/
  model.go
  scan.go
  metadata_parse.go
  diagnostics.go
  plan.go
  command_build.go
  runtime_invoke.go
  runtime_loader.go
  registry.go
  jsverbs_test.go
```

What this buys us:

- scanner concerns become smaller,
- metadata parsing becomes independently testable,
- binding-plan logic has one home,
- runtime loader and runtime invocation stop competing for space in one file.

This does not need to happen immediately. But if the package grows, the current file sizes already justify this direction:

- `scan.go`: 833 lines
- `command.go`: 513 lines
- `runtime.go`: 300 lines

That is manageable today, but clearly trending toward "too much per file".

## What A New Intern Should Change First

If you are the new engineer assigned to this package, do not start by adding more metadata features.

Start here:

1. add scanner diagnostics and fail-fast behavior,
2. add failure-path tests,
3. extract binding-plan logic,
4. only then add new parameter shapes or metadata.

That order matters because otherwise every new feature will deepen the weakest part of the implementation.

## Review Checklist For The Next Iteration

Use this checklist before calling the subsystem "ready for broader adoption":

- invalid metadata fails loudly and points to the right file/symbol,
- supported metadata syntax is explicit and tested,
- one binding plan feeds both schema generation and runtime invocation,
- runtime factory construction is reused or injected,
- promise waiting is documented or improved,
- example runner bootstrap behavior is clearly explained,
- failure fixtures exist alongside happy-path fixtures,
- help docs stay in `pkg/doc` and describe the real current contract.

## Final Assessment

The jsverbs branch is a successful feasibility spike with unusually good documentation for a spike. It proves the product direction and gives the team a real end-to-end artifact to discuss.

The main lesson from the postmortem is not that the design direction was wrong. It is that the next cleanup pass should target contract clarity, not more surface area. The architecture is good enough to continue. The internals are just still wearing prototype clothes.

If we harden metadata parsing, unify binding policy, and stop rebuilding runtime composition on every call, this can become a strong reusable subsystem rather than a one-off example.

## References

- `go-go-goja/pkg/jsverbs/model.go`
- `go-go-goja/pkg/jsverbs/scan.go`
- `go-go-goja/pkg/jsverbs/command.go`
- `go-go-goja/pkg/jsverbs/runtime.go`
- `go-go-goja/pkg/jsverbs/jsverbs_test.go`
- `go-go-goja/cmd/jsverbs-example/main.go`
- `go-go-goja/pkg/doc/doc.go`
- `go-go-goja/testdata/jsverbs/basics.js`
- `go-go-goja/testdata/jsverbs/advanced/numbers.js`
- `go-go-goja/engine/factory.go`
- `glazed/pkg/cmds/cmds.go`
- `glazed/pkg/cli/cobra.go`
- `glazed/pkg/cmds/logging/init.go`
- `go-go-goja/ttmp/2026/03/16/GOJA-04-JS-GLAZED-EXPORTS--add-glazed-command-exporting-from-javascript/reference/01-diary.md`
- `go-go-goja/ttmp/2026/03/16/GOJA-04-JS-GLAZED-EXPORTS--add-glazed-command-exporting-from-javascript/design-doc/01-js-to-glazed-command-exporting-design-and-implementation-guide.md`
