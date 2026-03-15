---
Title: Obsidian JavaScript API analysis design and implementation guide
Ticket: GOJA-038-OBSIDIAN-JS-API
Status: active
Topics:
    - obsidian
    - goja
    - bobatea
    - repl
    - architecture
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/js-repl/main.go
      Note: |-
        Existing dedicated JS REPL command to use as the base for an Obsidian-specific REPL
        Base command to copy for a dedicated Obsidian REPL
    - Path: engine/factory.go
      Note: FactoryBuilder runtime creation and module registration lifecycle
    - Path: engine/module_specs.go
      Note: Explicit module registration model used by the JS evaluator
    - Path: engine/runtime.go
      Note: Built-in module blank imports and runtime-owned resource lifecycle
    - Path: modules/common.go
      Note: Native module contract and default registry behavior
    - Path: modules/exports.go
      Note: Shared export helper used by native modules
    - Path: modules/obsidian/module.go
      Note: |-
        New native module bridge for require("obsidian")
        Native module bridge for require('obsidian')
    - Path: pkg/obsidian/client.go
      Note: High-level host-language Obsidian service layer
    - Path: pkg/obsidiancli/runner.go
      Note: CLI transport execution boundary
    - Path: pkg/obsidianmd/note_builder.go
      Note: Markdown note construction helpers
    - Path: pkg/repl/adapters/bobatea/javascript.go
      Note: Adapter boundary between Bobatea and go-go-goja evaluator logic
    - Path: pkg/repl/evaluators/javascript/evaluator.go
      Note: REPL evaluation semantics for Promise settlement and top-level await
ExternalSources:
    - https://help.obsidian.md/cli
Summary: Detailed intern-facing architecture, design, and implementation guide for adding and evolving an Obsidian JavaScript API in go-go-goja, including a full section for each implementation phase.
LastUpdated: 2026-03-15T16:45:00-04:00
WhatFor: Explain the system, the layering, the runtime boundaries, and the phased implementation strategy for an Obsidian API and REPL integration in go-go-goja.
WhenToUse: Use when onboarding a new contributor, reviewing the architecture, or continuing implementation of the Obsidian API and REPL work.
---


# Obsidian JavaScript API analysis design and implementation guide

## Executive Summary

This ticket defines how `go-go-goja` should host an Obsidian-oriented JavaScript API that feels ergonomic in scripts and in an interactive REPL, while still respecting the actual boundaries of the existing codebase.

The central design decision is simple:

- Bobatea should remain a generic REPL shell.
- `go-go-goja` should own JavaScript evaluation semantics, Promise behavior, module registration, and runtime safety.
- Obsidian-specific behavior should be built as a layered feature set inside `go-go-goja` and then exposed through a dedicated command.

The implementation is intentionally phased:

1. build CLI transport
2. build markdown helpers
3. build high-level Go service layer
4. expose a native goja module
5. make the REPL evaluator behave well for async APIs
6. add a dedicated Obsidian REPL command
7. prove the system against a real ZK workflow

This document is written for a new intern. It assumes the reader knows basic Go, basic JavaScript, and what a REPL is, but does not assume they already understand `go-go-goja`, goja, or Bobatea.

## What Problem We Are Solving

The source design asks for a friendly JavaScript API that lets a user write code like:

```javascript
const obs = require("obsidian");

await obs.configure({ vault: "My Vault" });
const files = await obs.files({ folder: "ZK/Claims", ext: "md" });
const note = await obs.note("ZK - 2a0 - Systems thinking");
const orphans = await obs.query().inFolder("Inbox").orphans().run();
```

There are several different problems hidden inside that simple example:

- How do we invoke the Obsidian CLI from Go?
- How do we parse and construct Obsidian-flavored Markdown?
- How do we provide higher-level note/query/batch semantics that are nicer than raw subprocess calls?
- How do we expose those operations as a native `require("obsidian")` module in goja?
- How do we make the REPL print the settled value of a Promise rather than `Promise { <pending> }`?
- Where should the Obsidian-specific UX live, and where should the generic REPL/runtime behavior live?

This ticket answers those questions with concrete package boundaries and an implementation order.

## System Overview

At a high level, the system looks like this:

```text
                    +--------------------------------+
                    | Bobatea REPL Shell             |
                    | pkg/repl/model.go              |
                    | - input box                    |
                    | - timeline transcript          |
                    | - completion/help plumbing     |
                    +---------------+----------------+
                                    |
                                    v
                    +--------------------------------+
                    | go-go-goja JS Evaluator        |
                    | pkg/repl/evaluators/javascript |
                    | - RunString / owner-thread use |
                    | - Promise settlement           |
                    | - top-level await strategy     |
                    +---------------+----------------+
                                    |
                                    v
                    +--------------------------------+
                    | goja Native Module             |
                    | modules/obsidian              |
                    | - require("obsidian")         |
                    | - Promise-returning exports    |
                    | - md namespace                 |
                    +---------------+----------------+
                                    |
                                    v
                    +--------------------------------+
                    | High-level Go API              |
                    | pkg/obsidian                  |
                    | - note/query/batch            |
                    | - cache                       |
                    | - ref resolution              |
                    +---------------+----------------+
                                    |
                    +---------------+----------------+
                    |                                |
                    v                                v
          +----------------------+         +----------------------+
          | Markdown Helpers     |         | CLI Transport        |
          | pkg/obsidianmd       |         | pkg/obsidiancli      |
          | - frontmatter        |         | - argv build         |
          | - wikilinks          |         | - subprocess runner  |
          | - tags/tasks         |         | - parsing/errors     |
          +----------------------+         +----------------------+
                                                    |
                                                    v
                                         +----------------------+
                                         | Obsidian CLI binary  |
                                         +----------------------+
```

## What Each Major Repository Part Does

### `engine/*`

This is the explicit runtime composition layer.

Important files:

- `engine/factory.go`
- `engine/module_specs.go`
- `engine/runtime.go`

What these files do:

- build a `require.Registry`
- register native modules into that registry
- create a goja runtime plus Node-ish helpers
- create a runtime owner abstraction for safe owner-thread work
- hold the runtime lifecycle in one place

This matters because the Obsidian module is not a random package. It becomes available to scripts only when the engine composition path enables it.

### `modules/*`

This is where native JS modules live.

Important files:

- `modules/common.go`
- `modules/exports.go`
- `modules/fs/fs.go`
- `modules/exec/exec.go`
- `modules/database/database.go`
- `modules/obsidian/module.go`

What this layer does:

- define `NativeModule`
- register modules in a default registry
- expose Go functions to JavaScript through `module.exports`

This is where `require("obsidian")` belongs.

### `pkg/repl/evaluators/javascript/*`

This is the JavaScript execution behavior for REPL use.

Important file:

- `pkg/repl/evaluators/javascript/evaluator.go`

What it does:

- owns the main `Evaluate` path
- converts results into transcript strings/events
- holds completion/help functionality
- decides whether Promises are settled or just printed as objects

This is the right place for JS-specific async REPL semantics.

### `pkg/repl/adapters/bobatea/*`

Important file:

- `pkg/repl/adapters/bobatea/javascript.go`

This adapter is thin by design. It exists so Bobatea can call into a go-go-goja evaluator using the Bobatea interfaces. It should not start owning JavaScript semantics.

### `bobatea/pkg/repl/*`

Important files in the separate Bobatea repo:

- `pkg/repl/evaluator.go`
- `pkg/repl/model.go`

Bobatea is the shell and transcript UI:

- input
- focus/layout
- timeline rendering
- event display

Bobatea should not learn what a JavaScript Promise is. It should simply render events emitted by the evaluator.

## Architectural Principles

### Principle 1: Put domain logic below the JS bridge

If a behavior is Obsidian-specific but not JavaScript-specific, it belongs in pure Go packages, not inside the goja module loader.

Examples:

- note resolution
- query planning
- batching
- markdown parsing
- cache invalidation

This keeps the module adapter thin and testable.

### Principle 2: Put JS semantics in the evaluator, not in the UI shell

If a behavior is specifically about:

- Promise settlement
- top-level `await`
- how results become transcript strings

then it belongs in `pkg/repl/evaluators/javascript`, not in Bobatea.

### Principle 3: Keep the REPL command thin

The final dedicated Obsidian REPL command should mostly:

- choose the evaluator config
- set prompt/help text
- wire flags into config
- start Bobatea

It should not reimplement the Obsidian API or Promise rules.

### Principle 4: Preserve reviewable commit boundaries

This work is safest when shipped as small slices:

- transport
- markdown
- host API
- native module
- evaluator semantics
- command

Those slices match the real package boundaries in the codebase.

## Current-State Gap Analysis

Before this work, the important gaps were:

1. no Obsidian transport package
2. no Obsidian markdown helper package
3. no high-level note/query/batch package
4. no native `obsidian` module
5. REPL evaluator returned Promise objects rather than settled values
6. no Obsidian-specific REPL entrypoint

Some of these gaps are now implemented in code, but this document still explains the full intended architecture so future contributors can understand why each layer exists.

## API Shape To Preserve

The target JavaScript surface is intentionally friendlier than raw CLI subcommands.

Core API families:

- `configure`
- `version`
- `files`, `read`, `create`, `append`, `prepend`, `move`, `rename`, `delete`
- `note`
- `query`
- `batch`
- `exec`
- `md.*`

Representative examples:

```javascript
const obs = require("obsidian");

await obs.configure({ vault: "My Vault", binaryPath: "obsidian" });

const files = await obs.files({ folder: "ZK/Claims", ext: "md" });
const content = await obs.read("ZK - 2a0 - Systems thinking");

const note = await obs.note("ZK - 2a0 - Systems thinking");
console.log(note.tags);

const rows = await obs.query()
  .inFolder("ZK/Claims")
  .withExtension("md")
  .tagged("software")
  .run();

const md = obs.md.note({
  title: "New Claim",
  wikiTags: ["Software", "Architecture"],
  body: "Architecture is about boundaries.",
  sections: [
    { title: "Brainstorm", body: "- Explore boundary patterns" },
    { title: "Links", body: "- [[2g - Architecture]]" },
  ],
});
```

## Data Flow Walkthrough

Here is a concrete end-to-end example for `await obs.query().inFolder("ZK/Claims").run()`:

```text
JS input in REPL
  -> Evaluator receives code
  -> goja executes native module function
  -> module/obsidian builds or mutates a Query object
  -> Query.Run() calls pkg/obsidian
  -> pkg/obsidian uses pkg/obsidiancli
  -> pkg/obsidiancli executes Obsidian CLI
  -> stdout parsed into paths
  -> pkg/obsidian turns paths into lazy Note objects
  -> module/obsidian converts Notes into JS-friendly maps
  -> evaluator sees a returned Promise
  -> evaluator waits for settlement
  -> Bobatea renders the settled result
```

That flow shows why the work belongs in several layers instead of one giant package.

## Phase-by-Phase Design And Implementation Guide

Each section below explains:

- goal
- why the phase exists
- key packages/files
- data model
- API shape
- pseudocode
- tests
- risks
- implementation notes

## Phase 1: `pkg/obsidiancli`

### Goal

Create a low-level transport package that knows how to invoke the Obsidian CLI predictably.

### Why This Phase Exists

The CLI is the unstable shell boundary:

- process execution
- argument ordering
- stdout parsing
- error translation

If this logic leaks upward, every higher layer becomes harder to test and maintain.

### Key Files

- `pkg/obsidiancli/config.go`
- `pkg/obsidiancli/spec.go`
- `pkg/obsidiancli/args.go`
- `pkg/obsidiancli/parse.go`
- `pkg/obsidiancli/errors.go`
- `pkg/obsidiancli/runner.go`
- `pkg/obsidiancli/*_test.go`

### Design

This package should expose:

- command specs
- deterministic argument serialization
- typed error values
- a `Runner` interface or struct that serializes subprocess access

Important types:

```go
type Config struct {
    BinaryPath string
    Vault      string
    WorkingDir string
    Timeout    time.Duration
    Env        []string
}

type CommandSpec struct {
    Name   string
    Output OutputKind
}

type CallOptions struct {
    Parameters map[string]any
    Flags      []string
    Positional []string
    Vault      string
}
```

### Why Deterministic Args Matter

If parameters are serialized in random map order:

- tests become noisy
- snapshot comparisons become weak
- debugging command mismatches gets harder

So parameter keys should be sorted before formatting.

### Example Pseudocode

```text
BuildArgs(config, spec, call):
  start args = []
  choose vault from call.Vault or config.Vault
  if vault exists:
    append "vault=<vault>"
  append spec.Name
  for key in sorted(call.Parameters):
    format key=value and append
  append sorted unique flags
  append positional args
  return args
```

### Parsing Strategy

The CLI will not always return the same shape. So parse by declared `OutputKind`:

- raw string
- JSON
- line list
- key/value pairs

### Error Model

The transport layer should classify at least:

- binary not found
- command failure
- parse error
- not found
- ambiguous reference
- unsupported version

These errors should be Go-first errors. Later layers decide how to expose them to JS.

### Tests

Required tests:

- arg serialization ordering
- vault precedence
- flag normalization
- JSON parsing
- line list parsing
- command execution with fake executor
- binary-not-found classification

### Risks

- CLI output may vary across versions
- line-based parsing can be brittle
- some commands may need richer structured parsing later

### Intern Implementation Checklist

- Add specs first
- Add formatting/parsing helpers second
- Add runner with injected executor third
- Add fake-executor tests before using a real subprocess

## Phase 2: `pkg/obsidianmd`

### Goal

Provide pure Go helpers for Obsidian-flavored Markdown operations.

### Why This Phase Exists

The Obsidian API design is not only about file I/O. It also expects markdown-aware helpers:

- frontmatter parsing
- wikilink extraction
- heading extraction
- tag extraction
- task extraction
- note construction

This logic should not live in the CLI transport layer or the JS module adapter.

### Key Files

- `pkg/obsidianmd/frontmatter.go`
- `pkg/obsidianmd/wikilinks.go`
- `pkg/obsidianmd/headings.go`
- `pkg/obsidianmd/tags.go`
- `pkg/obsidianmd/tasks.go`
- `pkg/obsidianmd/note_builder.go`
- `pkg/obsidianmd/obsidianmd_test.go`

### Design

This package should be pure and easy to unit test.

Core operations:

- `ParseDocument(raw string) (Document, error)`
- `ExtractWikilinks(body string) []string`
- `ExtractHeadings(body string) []Heading`
- `ExtractTags(body string) []string`
- `ExtractTasks(body string) []Task`
- `BuildNote(template NoteTemplate) (string, error)`

### Document Model

```go
type Document struct {
    Frontmatter map[string]any
    Body        string
}
```

### Why This Package Is Separate

Because these helpers are reused by:

- the Go service layer
- the JS `md` namespace
- future scripts or commands

If they are embedded in `pkg/obsidian` or `modules/obsidian`, reuse becomes awkward.

### Example Pseudocode

```text
ParseDocument(raw):
  if raw does not start with frontmatter fence:
    return {Body: raw}
  scan for closing fence
  decode YAML section into map
  return {Frontmatter: map, Body: remainder}
```

### Note Builder Design

The builder should let callers create consistent ZK-style notes.

Example:

```go
type NoteTemplate struct {
    Title    string
    WikiTags []string
    Body     string
    Sections []NoteSection
}
```

This is intentionally a builder, not a parser. It standardizes output shape.

### Tests

Required tests:

- no-frontmatter case
- YAML-frontmatter case
- wikilinks with aliases and heading fragments
- headings with correct levels and lines
- tags excluding heading syntax
- checkbox task parsing
- note-building ordering for ZK notes

### Risks

- tag extraction can overmatch
- wikilink parsing can under-handle advanced cases
- future vault conventions may want different section ordering

### Intern Implementation Checklist

- keep this package dependency-light
- favor simple deterministic parsers
- test plain text edge cases
- avoid mixing process execution concerns into markdown helpers

## Phase 3: `pkg/obsidian`

### Goal

Create the high-level host-language API that makes the raw CLI transport feel like an actual Obsidian domain API.

### Why This Phase Exists

Users do not want to think in terms of:

- command specs
- stdout parsers
- manual file list filtering

They want to think in terms of:

- note references
- notes
- queries
- batches

That is the job of `pkg/obsidian`.

### Key Files

- `pkg/obsidian/types.go`
- `pkg/obsidian/cache.go`
- `pkg/obsidian/client.go`
- `pkg/obsidian/note.go`
- `pkg/obsidian/query.go`
- `pkg/obsidian/batch.go`
- `pkg/obsidian/client_test.go`

### Core Objects

```go
type Client struct { ... }
type Note struct { ... }
type Query struct { ... }
type Cache struct { ... }
```

### Responsibilities

`Client`:

- version
- files
- search
- read
- create/update/delete style operations
- note resolution
- cache invalidation

`Note`:

- lazy content loading
- derived metadata
- frontmatter
- wikilinks
- tags
- tasks

`Query`:

- native filter planning
- post-filter planning
- fluent chain behavior

`Batch`:

- sequential default execution
- per-note results

### Important Design Decision: Split Native Filters From Post-Filters

Some filters can be pushed to the CLI:

- folder
- extension
- search term
- limit

Some filters need note content:

- tags in body
- derived metadata
- more complex note inspections

So the query planner should do:

```text
nativePaths = fetch best candidate path list from CLI
filtered = apply post-filters by loading notes only when needed
return note objects for remaining paths
```

This is efficient and conceptually clean.

### Reference Resolution

The API design wants friendly note references like:

```javascript
await obs.read("ZK - 2a0 - Systems thinking")
```

That means `Client.Read()` needs a reference resolution step:

1. if ref already looks like a path, use it directly
2. otherwise list markdown files
3. match by basename/title
4. return not-found or ambiguous errors when needed

### Example Pseudocode

```text
resolveReference(ref):
  if ref looks like a path:
    return ref
  files = Files(ext="md")
  matches = all files matching basename/title
  if matches == 0: not found
  if matches > 1: ambiguous
  return matches[0]
```

### Cache Design

Start small:

- cache note contents by resolved path
- invalidate on write operations
- clear all when config changes if necessary

This is enough for a first version and keeps behavior obvious.

### Tests

Required tests:

- read resolves friendly note names
- read uses cache on second fetch
- note derives tags/wikilinks/frontmatter
- query mixes native and post-filters
- batch returns per-note results

### Risks

- reference resolution can become expensive in large vaults
- cache coherence becomes harder when many write paths exist
- query post-filtering can get expensive for big vaults

### Intern Implementation Checklist

- implement client methods before the query builder
- build fake-runner tests rather than integration tests first
- make zero-value query behavior sane
- keep conversion logic out of the JS module layer

## Phase 4: `modules/obsidian`

### Goal

Expose the host-language Obsidian API to JavaScript through `require("obsidian")`.

### Why This Phase Exists

Without this phase, all previous work is only usable from Go. The design requires a JavaScript-facing API.

### Key Files

- `modules/obsidian/module.go`
- `modules/obsidian/module_test.go`
- `engine/runtime.go`
- `modules/common.go`
- `modules/exports.go`

### What This Layer Should Do

This layer should:

- decode JS options
- call `pkg/obsidian`
- convert Go results to JS-friendly values
- return Promises where the JS contract expects async operations

This layer should not:

- reimplement note/query logic
- manually parse markdown
- directly own subprocess policies

### Export Surface

Required exports:

- `configure`
- `version`
- `files`
- `read`
- `create`
- `append`
- `prepend`
- `move`
- `rename`
- `delete`
- `note`
- `query`
- `batch`
- `exec`
- `md`

### `md` Namespace

The `md` namespace is a clean place to export pure markdown helpers:

- `parseFrontmatter`
- `wikilinks`
- `headings`
- `tags`
- `tasks`
- `note`

### Query Builder in JS

The JS query object should be stateful and chainable:

```javascript
obs.query()
  .inFolder("ZK/Claims")
  .withExtension("md")
  .tagged("software")
  .run()
```

This maps naturally onto the Go `Query` object. The JS object should just mutate the underlying Go query and return itself.

### Async Design

The user-facing contract should be Promise-based. That means exports like `read`, `files`, `note`, and `query().run()` should return Promises.

There is one important runtime subtlety:

- native modules only receive `*goja.Runtime` and `module.exports` in `Loader`
- they do not automatically receive the full engine runtime context
- so access to a `runtimeowner.Runner` is not implicit in the default module contract

That leads to the design compromise:

- if an owner runner is injected, use it for Promise settlement on the owner thread
- otherwise do synchronous settlement on the VM thread for current blocking operations

That compromise keeps the JS shape correct while avoiding unsafe VM usage.

### Module Loader Diagram

```text
Loader(vm, moduleObj)
  -> exports := moduleObj.exports
  -> state := ensure runtime-specific state
  -> set exported functions on exports
  -> each export decodes args -> calls pkg/obsidian -> returns Promise/value
```

### Tests

Required tests:

- fulfilled Promise result
- rejected Promise result
- fluent query chaining
- module availability through engine runtime creation

### Risks

- JS/Go conversion drift
- hidden coupling between module state and runtime lifecycle
- future truly async background operations need explicit owner-thread wiring

### Intern Implementation Checklist

- keep module functions thin
- test through a real runtime, not only unit helpers
- prefer explicit option structs to loose exported maps when possible
- do not move domain logic from `pkg/obsidian` into `module.go`

## Phase 5: Evaluator And REPL Runtime Behavior

### Goal

Make the JavaScript REPL behave naturally for Promise-based APIs and top-level `await`.

### Why This Phase Exists

Without this phase, a perfectly good Promise-based module still feels bad in the REPL.

Bad REPL behavior looks like:

- user types `obs.version()`
- transcript shows a raw Promise object

Good REPL behavior looks like:

- user types `await obs.version()`
- transcript shows `1.12.4`

### Key Files

- `pkg/repl/evaluators/javascript/evaluator.go`
- `pkg/repl/evaluators/javascript/evaluator_test.go`
- `pkg/repl/adapters/bobatea/javascript.go`
- Bobatea reference: `bobatea/pkg/repl/evaluator.go`

### Where This Logic Belongs

It belongs in `go-go-goja`, not Bobatea.

Why:

- Bobatea only knows how to ask an evaluator to emit events
- Promise semantics are JavaScript runtime semantics
- top-level `await` is a JavaScript evaluation concern

### Minimum Useful Support

You do not need full JavaScript language transformation on day one.

The narrow, high-value behavior is:

1. if the returned evaluation result is a Promise, wait for it and render the settled value
2. if user input starts with expression-style `await ...`, rewrite it into an async IIFE

Example rewrite:

```text
await obs.files()
```

becomes:

```javascript
(async () => { return await obs.files(); })()
```

This preserves the existing REPL model while supporting the most important user workflow.

### Promise Settlement Strategy

When the evaluator owns an engine runtime with a runtime owner, it should inspect Promise state on the owner thread.

Pseudocode:

```text
Evaluate(code):
  maybe rewrite await-expression input
  result = run code
  if result is Promise:
    loop:
      inspect promise state on owner thread
      if pending: sleep briefly and retry
      if rejected: return error
      if fulfilled: stringify result
  else:
    stringify result normally
```

### Why Not Put This In Bobatea

Because Bobatea would then need to know:

- what a Promise is
- how to inspect JS runtime state
- how to rewrite JS input

That would be a layering mistake.

### Tests

Required tests:

- fulfilled Promise becomes transcript output
- rejected Promise becomes evaluator error
- expression-style top-level await works
- adapter to Bobatea still functions

### Risks

- full statement/declaration top-level await is harder than expression-only support
- waiting indefinitely on unresolved Promises can hang the REPL
- cancellation and timeout semantics need care

### Intern Implementation Checklist

- start with returned Promise settlement
- add expression-style `await`
- do not try to parse the full JS grammar for v1
- keep the change localized to evaluator code

## Phase 6: Dedicated Obsidian REPL Command

### Goal

Provide a command tailored to the Obsidian workflow rather than a generic JS REPL.

### Why This Phase Exists

Generic `cmd/js-repl` is useful for development, but a user-focused Obsidian tool should:

- provide Obsidian-aware help text
- accept vault/binary flags
- preconfigure the runtime with the Obsidian module
- feel like a product, not just a runtime demo

### Key Files

- `cmd/js-repl/main.go`
- new `cmd/obsidian-js-repl/main.go`
- `pkg/repl/adapters/bobatea/javascript.go`
- `pkg/repl/evaluators/javascript/evaluator.go`

### Command Design

The new command should:

- reuse existing Bobatea REPL shell setup
- reuse existing JS evaluator
- pass config for:
  - vault
  - binary path
  - working directory
  - timeout
- change placeholder/help text to show Obsidian examples

### Suggested UX

Prompt and placeholder examples:

```text
obs>
Placeholder: await obs.files({ folder: "ZK/Claims", ext: "md" })
```

### Suggested Startup Help Snippet

```javascript
const obs = require("obsidian");

await obs.configure({ vault: "My Vault" });
await obs.version();
await obs.files({ folder: "Inbox" });
```

### Pseudocode

```text
main():
  parse flags
  create JS evaluator config
  inject obsidian module defaults/config
  create Bobatea JS adapter
  create Bobatea REPL model with prompt/help settings
  run app
```

### Tests

Required validation:

- command starts
- flags propagate into evaluator/module config
- example commands work in smoke tests or manual validation steps

### Risks

- too much app logic in `main.go`
- config duplication between command and module
- trying to do product UX before core runtime behavior is stable

### Intern Implementation Checklist

- clone `cmd/js-repl/main.go` structure first
- keep the first version small
- prefer smoke validation plus clear manual testing instructions

## Phase 7: Local ZK Workflow Proof Of Concept

### Goal

Demonstrate that the API is not just elegant in theory; it should support a real note-management workflow.

### Why This Phase Exists

Architecture work becomes credible when it can replace or mirror a real workflow.

The motivating workflow here is a Zettelkasten filing/classification flow:

- inspect vault tree/index
- read inbox material
- classify notes
- create or update target notes

### Inputs To Study

External but relevant source files:

- `DESIGN-obsidian-js-api.md`
- `PROJ - ZK Tool.md`
- `scripts/zk_create.py`
- `scripts/build_tree_index.py`

These are not part of `go-go-goja`, but they define the practical consumer story.

### Proof-Of-Concept Options

Option A: JS script using the new REPL/module

- easiest to validate the user-facing API
- good for demos and documentation

Option B: Go command using `pkg/obsidian`

- useful for service-layer validation
- less representative of the target JS ergonomics

Recommended order:

1. first JS proof-of-concept
2. optional Go wrapper or helper command later

### Example Pseudocode

```javascript
const obs = require("obsidian");

await obs.configure({ vault: "My Vault" });

for (const note of await obs.query().inFolder("Inbox").run()) {
  const classification = classify(note.content);
  const body = obs.md.note({
    title: classification.title,
    wikiTags: classification.tags,
    body: classification.summary,
    sections: [
      { title: "Links", body: classification.links },
      { title: "Logs", body: `[[2026-03-15]]\n- Created` },
    ],
  });
  await obs.create(classification.filename, {
    folder: classification.folder,
    content: body,
  });
}
```

### Validation Criteria

The proof-of-concept is successful if it proves:

- friendly note reads work
- query chaining works
- markdown note generation works
- create/update paths work
- the REPL can support exploratory execution

### Risks

- local vault conventions may differ from the abstract design
- note naming rules may expose ambiguity edge cases
- throughput/performance may matter more on larger vaults

### Intern Implementation Checklist

- start with a narrow happy path
- log every mismatch between real notes and parser assumptions
- treat proof-of-concept findings as input for API refinement

## Recommended File-Level Implementation Order

```text
1. pkg/obsidiancli/*
2. pkg/obsidianmd/*
3. pkg/obsidian/*
4. modules/obsidian/*
5. pkg/repl/evaluators/javascript/*
6. cmd/obsidian-js-repl/main.go
7. proof-of-concept script/command
```

This order is important because each phase depends on the layer below it.

## Testing Strategy Across All Phases

### Unit tests

Use for:

- transport
- markdown helpers
- client/query/note

### Runtime integration tests

Use for:

- native module loading
- Promise fulfillment/rejection
- query chaining through `require("obsidian")`

### Smoke/manual tests

Use for:

- dedicated REPL command
- real vault interactions
- proof-of-concept workflow

## Open Design Questions

These are the main questions that still deserve explicit review:

1. Should `NativeModule.Loader` eventually have a richer runtime-scoped dependency path for owner-thread async module support?
2. Should query evaluation support streaming/iterator-style results for very large vaults?
3. Should full top-level `await` statement support be added, or is expression-style support enough for the intended REPL?
4. How much caching is acceptable before correctness becomes harder to reason about?

## Final Recommendation

The architecture should remain layered exactly as described above.

The most important long-term rule is:

- keep Bobatea generic
- keep JS semantics in `go-go-goja`
- keep Obsidian domain behavior below the JS bridge

If future contributors preserve that rule, the system will stay understandable and reviewable. If they break it, the code will quickly become hard to test and hard to reason about.

For an intern joining this project, the best reading order is:

1. `engine/factory.go`
2. `modules/common.go`
3. `pkg/obsidiancli/*`
4. `pkg/obsidianmd/*`
5. `pkg/obsidian/*`
6. `modules/obsidian/module.go`
7. `pkg/repl/evaluators/javascript/evaluator.go`
8. `cmd/js-repl/main.go`
