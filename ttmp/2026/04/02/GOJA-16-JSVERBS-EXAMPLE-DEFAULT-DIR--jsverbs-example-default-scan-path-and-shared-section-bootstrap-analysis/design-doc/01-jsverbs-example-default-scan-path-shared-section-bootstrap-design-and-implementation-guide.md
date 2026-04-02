---
Title: jsverbs example default scan path shared section bootstrap design and implementation guide
Ticket: GOJA-16-JSVERBS-EXAMPLE-DEFAULT-DIR
Status: active
Topics:
    - goja
    - glazed
    - documentation
    - analysis
    - tooling
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/cmd/jsverbs-example/main.go
      Note: Example runner bootstrap and conditional shared-section registration
    - Path: go-go-goja/pkg/doc/09-jsverbs-example-fixture-format.md
      Note: Documented file-local versus registry-shared section semantics
    - Path: go-go-goja/pkg/jsverbs/binding.go
      Note: Binding-plan validation and unknown-section failure path
    - Path: go-go-goja/pkg/jsverbs/command.go
      Note: Command compilation from binding plan
    - Path: go-go-goja/pkg/jsverbs/jsverbs_test.go
      Note: Existing shared-section behavior tests
    - Path: go-go-goja/pkg/jsverbs/model.go
      Note: Registry and section-resolution semantics
    - Path: go-go-goja/pkg/jsverbs/runtime.go
      Note: Runtime invocation and argument marshaling
    - Path: go-go-goja/pkg/jsverbs/scan.go
      Note: Scanning
    - Path: go-go-goja/testdata/jsverbs-example/registry-shared/issues.js
      Note: Failing registry-shared fixture without local section definition
    - Path: go-go-goja/testdata/jsverbs/basics.js
      Note: Self-contained local-section example that inspired the user hypothesis
ExternalSources: []
Summary: Detailed intern-facing analysis of the jsverbs subsystem and a proposed plan to fix jsverbs-example default scan behavior without changing current section semantics.
LastUpdated: 2026-04-02T08:52:36.435146796-04:00
WhatFor: Explain the jsverbs architecture, document the zero-arg jsverbs-example failure, and propose an intern-friendly implementation plan for fixing the bootstrap/default-directory behavior without changing section semantics accidentally.
WhenToUse: Use when debugging jsverbs discovery/binding/runtime behavior, onboarding a new engineer to the subsystem, or implementing the follow-up fix for jsverbs-example default scan behavior.
---


# jsverbs example default scan path shared section bootstrap design and implementation guide

## Executive Summary

Running `go run ./cmd/jsverbs-example` from the `go-go-goja` repository root currently fails with `testdata/jsverbs-example/registry-shared/issues.js#listIssues references unknown section "filters"`. The failure is real, but the likely cause is not "the system needs to load `basics.js` first."

The evidence in this workspace shows a different root cause:

1. The example command defaults to scanning `"."` when `--dir` is omitted in [`cmd/jsverbs-example/main.go`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/cmd/jsverbs-example/main.go#L96 ).
2. The special host-side shared-section bootstrap only runs when the scanned directory basename is exactly `registry-shared` in [`cmd/jsverbs-example/main.go`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/cmd/jsverbs-example/main.go#L120 ).
3. `pkg/jsverbs` intentionally treats `__section__` as file-local metadata, and registry-level sharing must be registered explicitly through `Registry.AddSharedSection(...)`, as documented in [`pkg/doc/09-jsverbs-example-fixture-format.md`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/pkg/doc/09-jsverbs-example-fixture-format.md#L30 ) and implemented in [`pkg/jsverbs/model.go`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/pkg/jsverbs/model.go#L180 ).

This means loading `testdata/jsverbs/basics.js` first would not solve the current failure under the present design, because the `filters` section declared there is local to that file, not global across the whole scan.

The recommended follow-up is therefore:

1. Fix `jsverbs-example` bootstrap behavior, not `pkg/jsverbs` section semantics.
2. Keep file-local vs registry-shared section scope exactly as currently documented and tested.
3. Make the zero-argument UX deterministic and non-surprising, either by requiring `--dir` explicitly or by choosing a stable default fixture directory such as `./testdata/jsverbs`.

The rest of this document explains how `jsverbs` works, why the failure happens, what not to change accidentally, and how an intern should approach the implementation safely.

## Problem Statement

The user-visible problem is small but architecturally important:

```bash
go run ./cmd/jsverbs-example
```

fails at startup with:

```text
testdata/jsverbs-example/registry-shared/issues.js#listIssues references unknown section "filters"
exit status 1
```

At first glance this looks like a section-loading-order bug. There is another fixture, [`testdata/jsverbs/basics.js`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/testdata/jsverbs/basics.js#L1 ), that defines a `filters` section, so it is tempting to think the solution is "load `basics.js` first."

That hypothesis is inconsistent with the current subsystem contract:

- `__section__` is intentionally file-local metadata.
- cross-file sharing is intentionally host-registered through `Registry.AddSharedSection(...)`.
- `pkg/jsverbs` already has tests covering registry-level shared sections, file-local precedence, and failure when a section is missing.

So the real design problem is:

`jsverbs-example` has a confusing zero-argument bootstrap path that scans a very broad tree but only conditionally registers the special shared section needed by one narrow fixture subtree.

The scope of this ticket is documentation, diagnosis, and implementation guidance for a future code fix. This ticket does not itself change the runtime semantics.

## Scope

In scope:

- explain how `jsverbs` scanning, binding, command generation, and runtime invocation work,
- explain the exact failure mechanism for the zero-arg example run,
- distinguish observed behavior from the user's initial theory,
- propose a concrete follow-up implementation plan,
- preserve the current section-scope model unless a separate design effort explicitly changes it.

Out of scope:

- implementing cross-file `__section__` imports,
- changing `require()` to carry metadata,
- introducing implicit load-order-dependent global section catalogs,
- making backwards-compatibility adapters for old metadata behavior.

## Reproduction And Observed Behavior

### Reproduction that fails

From the repository root:

```bash
go run ./cmd/jsverbs-example
```

Observed output:

```text
testdata/jsverbs-example/registry-shared/issues.js#listIssues references unknown section "filters"
exit status 1
```

### Reproduction that works

The dedicated registry-shared example works when the intended directory is scanned explicitly:

```bash
go run ./cmd/jsverbs-example --dir ./testdata/jsverbs-example/registry-shared list
go run ./cmd/jsverbs-example --dir ./testdata/jsverbs-example/registry-shared \
  issues list-issues go-go-golems/go-go-goja --state closed --labels bug --labels docs
```

Observed behavior:

- `list` reports the two expected verbs from `issues.js` and `summary.js`
- `issues list-issues ...` executes successfully and shows the `filters` section values being passed into JavaScript

This is a critical diagnostic fact. It proves:

- the registry-shared fixture itself is valid,
- `pkg/jsverbs` supports registry-level shared sections correctly,
- the failure is in example-runner bootstrap assumptions, not the core shared-section feature.

## Current-State Architecture

This section is the most important onboarding material for a new engineer. Read it before proposing changes.

### Big-picture pipeline

```text
CLI entrypoint
  cmd/jsverbs-example/main.go
      |
      v
Directory discovery
  discoverDirectory(args)
      |
      v
Static scan
  jsverbs.ScanDir / ScanFS / ScanSource / ScanSources
      |
      v
Registry enrichment
  Registry.AddSharedSection(...)
      |
      v
Binding plan
  buildVerbBindingPlan(...)
      |
      v
Command compilation
  Registry.Commands() -> Glazed command descriptions
      |
      v
Runtime invocation
  Registry.invoke(...) -> goja runtime -> JS function
      |
      v
Output conversion
  rows or text
```

### 1. CLI bootstrap in `cmd/jsverbs-example`

The example runner currently does four core things:

1. pick a directory with `discoverDirectory(...)`,
2. scan it with `jsverbs.ScanDir(dir)`,
3. optionally register a host-side shared section,
4. compile commands and hand them to Cobra/Glazed.

Relevant code:

- directory selection: [`cmd/jsverbs-example/main.go`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/cmd/jsverbs-example/main.go#L21 )
- default `"."` behavior: [`cmd/jsverbs-example/main.go`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/cmd/jsverbs-example/main.go#L96 )
- conditional shared-section registration: [`cmd/jsverbs-example/main.go`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/cmd/jsverbs-example/main.go#L120 )

The critical issue is the mismatch between steps 1 and 3:

- step 1 defaults to scanning the whole current directory,
- step 3 only injects the special `filters` shared section when the scanned directory basename is `registry-shared`.

So a no-arg invocation scans the whole repo tree, finds the registry-shared fixture, but does not perform the matching host registration for it.

### 2. Registry model in `pkg/jsverbs/model.go`

The registry is the in-memory contract that ties scanning, binding, command compilation, and runtime execution together.

Key types:

- `Registry`: global scan result and runtime source catalog
- `FileSpec`: one scanned source file
- `VerbSpec`: one command candidate
- `SectionSpec`: reusable group of fields
- `FieldSpec`: one field definition

Relevant code:

- registry shape: [`pkg/jsverbs/model.go`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/pkg/jsverbs/model.go#L74 )
- shared-section registration: [`pkg/jsverbs/model.go`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/pkg/jsverbs/model.go#L180 )
- section resolution: [`pkg/jsverbs/model.go`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/pkg/jsverbs/model.go#L222 )

The most important semantic rule is `ResolveSection(...)`:

```text
if file-local section exists for slug:
    use file-local section
else if registry shared section exists for slug:
    use registry shared section
else:
    section is missing
```

That rule is deliberate. It means:

- local files can override a shared section intentionally,
- shared sections are injected by the host, not discovered globally across unrelated files,
- section scope is explicit rather than path-order-dependent.

### 3. Static scan in `pkg/jsverbs/scan.go`

The scanner does not execute JavaScript. It parses source with tree-sitter and extracts only a narrow set of constructs.

Entry points:

- [`ScanDir(...)`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/pkg/jsverbs/scan.go#L17 )
- [`ScanFS(...)`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/pkg/jsverbs/scan.go#L76 )
- [`ScanSource(...)`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/pkg/jsverbs/scan.go#L126 )
- [`ScanSources(...)`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/pkg/jsverbs/scan.go#L130 )

Important scan details:

- `ScanDir(...)` recursively walks the provided root: [`pkg/jsverbs/scan.go`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/pkg/jsverbs/scan.go#L36 )
- scanned files are sorted by relative path before verb finalization: [`pkg/jsverbs/scan.go`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/pkg/jsverbs/scan.go#L180 )
- the extractor recognizes `__package__`, `__section__`, `__verb__`, and `doc\`...\`` at top level: [`pkg/jsverbs/scan.go`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/pkg/jsverbs/scan.go#L400 )
- `__section__` metadata is stored on the current `FileSpec`: [`pkg/jsverbs/scan.go`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/pkg/jsverbs/scan.go#L477 )
- `__verb__` metadata is also file-local until later verb finalization: [`pkg/jsverbs/scan.go`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/pkg/jsverbs/scan.go#L500 )

The sort order is real, but it does not create a global section catalog. It only stabilizes processing order. That is the key reason the "load `basics.js` first" hypothesis is not sufficient.

### 4. Binding plan in `pkg/jsverbs/binding.go`

`buildVerbBindingPlan(...)` is the bridge between metadata and runtime semantics.

Relevant code:

- plan construction: [`pkg/jsverbs/binding.go`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/pkg/jsverbs/binding.go#L40 )
- missing-section validation: [`pkg/jsverbs/binding.go`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/pkg/jsverbs/binding.go#L121 )
- binding modes: [`pkg/jsverbs/binding.go`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/pkg/jsverbs/binding.go#L162 )

Conceptually:

```text
for each verb:
    gather referenced sections from:
        - verb.sections / useSections
        - field.section
        - field.bind when bind names a section
    validate each referenced section through registry.ResolveSection(...)
    record final ordered list of referenced sections
    record parameter binding mode for each parameter
```

This is where the startup error comes from. When `issues.js#listIssues` contains:

```js
filters: {
  bind: "filters"
}
```

the binding planner interprets that as "this verb references a section named `filters`." If `ResolveSection(...)` cannot find that slug locally or in registry shared sections, command compilation fails immediately. That is correct behavior.

### 5. Command compilation in `pkg/jsverbs/command.go`

`Registry.Commands()` converts each finalized `VerbSpec` into a Glazed command description.

Relevant code:

- command list assembly: [`pkg/jsverbs/command.go`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/pkg/jsverbs/command.go#L37 )
- command description building: [`pkg/jsverbs/command.go`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/pkg/jsverbs/command.go#L72 )
- section materialization from referenced sections: [`pkg/jsverbs/command.go`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/pkg/jsverbs/command.go#L115 )

The command builder does not invent sections. It can only materialize sections that the binding planner already resolved successfully.

This is another reason the failure is bootstrap-related:

- scan sees `issues.js`,
- binding planner notices `bind: "filters"`,
- no local `__section__("filters", ...)` exists in `issues.js`,
- no shared section has been registered on the registry,
- command generation aborts.

### 6. Runtime invocation in `pkg/jsverbs/runtime.go`

Runtime execution only begins after successful command compilation and argument parsing.

Relevant code:

- runtime invoke entry point: [`pkg/jsverbs/runtime.go`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/pkg/jsverbs/runtime.go#L18 )
- argument building from binding plan: [`pkg/jsverbs/runtime.go`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/pkg/jsverbs/runtime.go#L89 )
- source overlay / sentinel no-ops: [`pkg/jsverbs/runtime.go`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/pkg/jsverbs/runtime.go#L154 )

Important runtime facts for interns:

- sentinels such as `__verb__` and `__section__` matter at scan time,
- at runtime they are installed as no-op globals so loading the module does not crash,
- the runtime uses the same binding plan as command generation,
- `bind: "context"` and `bind: "all"` are runtime argument-shaping rules, not scan rules.

If startup fails with `unknown section`, the runtime has not even started yet. That narrows debugging dramatically.

## How jsverbs Work

This section is a concise tutorial for somebody new to the subsystem.

### Authoring model

A JavaScript author writes:

1. top-level functions,
2. optional `__package__` metadata,
3. optional file-local `__section__` metadata,
4. optional `__verb__` metadata,
5. optional `doc\`...\`` prose.

Example from [`testdata/jsverbs/basics.js`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/testdata/jsverbs/basics.js#L1 ):

```js
__section__("filters", {
  title: "Filters",
  description: "Shared filter flags",
  fields: {
    state: { type: "choice", choices: ["open", "closed"], default: "open" },
    labels: { type: "stringList" }
  }
});

function listIssues(repo, filters, meta) {
  return [{ repo, state: filters.state, rootDir: meta.rootDir }];
}

__verb__("listIssues", {
  sections: ["filters"],
  fields: {
    repo: { argument: true },
    filters: { bind: "filters" },
    meta: { bind: "context" }
  }
});
```

The Go side then turns this into:

- a Cobra/Glazed command path,
- parsed CLI flags and arguments,
- a call into the JS function with correctly marshaled arguments,
- structured or text output.

### Discovery model

The scanner recognizes only stable top-level syntax. It avoids executing code. That makes command discovery deterministic and safe.

### Section model

There are exactly two supported section scopes today:

1. file-local sections declared by `__section__`,
2. registry-level shared sections registered from Go.

There is no third "all scanned files share all sections by slug" scope.

### Binding model

`jsverbs` supports four parameter modes:

- positional field binding,
- named section binding,
- `bind: "all"`,
- `bind: "context"`.

The important point is that a section bind requires the section to exist before command compilation.

### Output model

By default a verb is structured output (`GlazeCommand`). If it sets `output: "text"`, it becomes a writer command instead.

## Why `basics.js` First Does Not Fix The Current Failure

This deserves a dedicated section because it is the most likely misunderstanding for a new engineer.

### The tempting theory

There is a `filters` section in [`testdata/jsverbs/basics.js`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/testdata/jsverbs/basics.js#L1 ), and the failing fixture uses `bind: "filters"` in [`testdata/jsverbs-example/registry-shared/issues.js`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/testdata/jsverbs-example/registry-shared/issues.js#L12 ).

So maybe the scanner just needs to process `basics.js` first.

### Why that is incorrect under current semantics

The docs and code agree that `__section__` is file-local:

- docs: [`pkg/doc/09-jsverbs-example-fixture-format.md`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/pkg/doc/09-jsverbs-example-fixture-format.md#L30 )
- code: file-local sections are stored in `FileSpec.Sections` during scan: [`pkg/jsverbs/scan.go`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/pkg/jsverbs/scan.go#L493 )
- resolution prefers file-local then registry-shared: [`pkg/jsverbs/model.go`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/pkg/jsverbs/model.go#L222 )

So even if `basics.js` is processed before `issues.js`, the `filters` section in `basics.js` remains attached to the `FileSpec` for `basics.js`. It is not promoted to a registry-global shared section automatically.

### What would have to change for "load order" to matter

For load order to matter, the subsystem would need a new behavior such as:

```text
globalScannedSectionCatalog[slug] = first __section__ seen with that slug
```

or:

```text
if a verb in file B references slug X:
    search every scanned file for __section__("X", ...)
```

That is a materially different design. It would:

- contradict existing documentation,
- blur the distinction between file-local and host-shared sections,
- create path-order coupling,
- make section provenance less obvious,
- require new precedence and duplicate-handling rules.

That should only happen in a separate ticket if the team explicitly wants to change the product model.

## Existing Documentation And Test Evidence

The system already contains evidence that the intended model is "host-registered shared sections," not cross-file implicit sharing.

### Docs

The fixture-format doc says:

- `__section__` is file-local
- cross-file reuse should be done by `Registry.AddSharedSection(...)`
- the example runner has a dedicated `registry-shared` demo

Reference: [`pkg/doc/09-jsverbs-example-fixture-format.md`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/pkg/doc/09-jsverbs-example-fixture-format.md#L30 )

### Tests

There are explicit tests for:

- shared sections used successfully: [`pkg/jsverbs/jsverbs_test.go`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/pkg/jsverbs/jsverbs_test.go#L319 )
- local sections overriding shared sections: [`pkg/jsverbs/jsverbs_test.go`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/pkg/jsverbs/jsverbs_test.go#L360 )
- unknown sections still failing when neither catalog contains the slug: [`pkg/jsverbs/jsverbs_test.go`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/pkg/jsverbs/jsverbs_test.go#L443 )

These tests already encode the intended semantics. A future fix should work with them, not around them.

## Gap Analysis

### What the user expected

The user expected `go run ./cmd/jsverbs-example` to be a reasonable out-of-the-box entrypoint.

That expectation is fair. Example binaries should ideally have deterministic default behavior.

### What the code currently does

- scans the whole current directory when `--dir` is omitted,
- includes test fixtures that are only valid under specific host bootstrap conditions,
- applies that host bootstrap only to one basename-sensitive case.

### The gap

The gap is not core `jsverbs` capability. The gap is example-runner UX and bootstrap consistency.

## Proposed Solution

### Recommendation

Do not change section semantics in `pkg/jsverbs` for this ticket.

Instead, change `cmd/jsverbs-example` so the zero-argument path is deterministic and intentional.

### Recommended implementation option

Preferred approach:

1. When no `--dir` is provided, default to `./testdata/jsverbs`.
2. Keep `--dir` override behavior unchanged.
3. Keep `registry-shared` behavior behind explicit `--dir ./testdata/jsverbs-example/registry-shared`.
4. Update help/docs so zero-arg behavior is documented clearly.

Rationale:

- `./testdata/jsverbs` is the main self-contained example tree already used in docs,
- it does not require host-side special registration to be useful,
- it keeps the registry-shared example as an advanced explicit demo,
- it fixes the surprising startup failure without changing `pkg/jsverbs` semantics.

### Alternative acceptable implementation option

If the team wants stronger explicitness:

1. require `--dir` when omitted,
2. print a clear error that also suggests known example directories.

This is slightly less convenient but maximally honest about the command's inputs.

### Not recommended here

Not recommended in this ticket:

- auto-importing sections from unrelated files,
- scanning the whole repo and auto-registering shared sections based on discovered fixture paths,
- making `registerExampleSharedSections(...)` inspect scanned files and infer intent.

Those approaches couple the example runner to incidental repository layout and make the mental model harder.

## Design Decisions

### Decision 1: Preserve file-local vs registry-shared section semantics

Reason:

- docs already teach this model,
- tests already validate it,
- it keeps provenance explicit.

### Decision 2: Treat the failure as example-bootstrap UX debt

Reason:

- explicit `--dir ./testdata/jsverbs-example/registry-shared` already works,
- the failure only appears in the default path,
- changing the CLI bootstrap is much smaller and safer than changing core semantics.

### Decision 3: Prefer deterministic default fixture over repo-root scan

Reason:

- repo-root scan is too broad for an example binary,
- it pulls in fixtures that are not meant to coexist under one bootstrap mode,
- example programs should favor stable onboarding over maximal automatic discovery.

## Pseudocode

### Current behavior

```go
dir := discoverDirectory(os.Args[1:]) // "." when omitted
registry := jsverbs.ScanDir(dir)
if base(dir) == "registry-shared" {
    registry.AddSharedSection(filtersSection)
}
commands := registry.Commands()
```

### Recommended future behavior

```go
dir, dirWasExplicit := discoverDirectory(os.Args[1:])
if !dirWasExplicit {
    dir = "./testdata/jsverbs"
}

registry := jsverbs.ScanDir(dir)

if base(clean(dir)) == "registry-shared" {
    registry.AddSharedSection(filtersSection)
}

commands := registry.Commands()
```

### If the team prefers explicit `--dir`

```go
dir, dirWasExplicit := discoverDirectory(os.Args[1:])
if !dirWasExplicit {
    return error(
        "--dir is required; try ./testdata/jsverbs or " +
        "./testdata/jsverbs-example/registry-shared",
    )
}
```

## Intern Implementation Guide

This section is written as a handoff to a new engineer.

### Step 1: Read the system in order

Read these files in this order:

1. [`cmd/jsverbs-example/main.go`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/cmd/jsverbs-example/main.go )
2. [`pkg/jsverbs/scan.go`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/pkg/jsverbs/scan.go )
3. [`pkg/jsverbs/model.go`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/pkg/jsverbs/model.go )
4. [`pkg/jsverbs/binding.go`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/pkg/jsverbs/binding.go )
5. [`pkg/jsverbs/command.go`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/pkg/jsverbs/command.go )
6. [`pkg/jsverbs/runtime.go`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/pkg/jsverbs/runtime.go )
7. [`pkg/jsverbs/jsverbs_test.go`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/pkg/jsverbs/jsverbs_test.go )
8. [`pkg/doc/09-jsverbs-example-fixture-format.md`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/pkg/doc/09-jsverbs-example-fixture-format.md )

### Step 2: Reproduce both paths

Run:

```bash
go run ./cmd/jsverbs-example
go run ./cmd/jsverbs-example --dir ./testdata/jsverbs list
go run ./cmd/jsverbs-example --dir ./testdata/jsverbs-example/registry-shared list
```

You need to observe both the failing default case and the two valid explicit examples before changing code.

### Step 3: Implement only the bootstrap change

Change only the example-runner directory-selection behavior first.

Do not touch:

- `Registry.ResolveSection(...)`,
- `buildVerbBindingPlan(...)`,
- `registerExampleSharedSections(...)` semantics,
- scan-time `__section__` storage.

Those are core semantics, not UX defaults.

### Step 4: Add tests near the example runner

Add or update tests that validate:

1. zero-arg behavior now succeeds or errors clearly, depending on the chosen design,
2. `--dir ./testdata/jsverbs-example/registry-shared` still works,
3. `pkg/jsverbs` shared-section tests still pass unchanged.

### Step 5: Update docs

If the default behavior changes, update:

- [`pkg/doc/08-jsverbs-example-overview.md`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/pkg/doc/08-jsverbs-example-overview.md )
- [`pkg/doc/10-jsverbs-example-developer-guide.md`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/pkg/doc/10-jsverbs-example-developer-guide.md )
- [`pkg/doc/11-jsverbs-example-reference.md`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/pkg/doc/11-jsverbs-example-reference.md )

### Step 6: Validate manually

Run:

```bash
go test ./pkg/jsverbs ./cmd/jsverbs-example
go run ./cmd/jsverbs-example --dir ./testdata/jsverbs list
go run ./cmd/jsverbs-example --dir ./testdata/jsverbs basics list-issues go-go-golems/go-go-goja --state closed --labels bug
go run ./cmd/jsverbs-example --dir ./testdata/jsverbs-example/registry-shared issues list-issues go-go-golems/go-go-goja --state closed --labels bug
```

## Testing Strategy

The future implementation should be validated at three levels.

### Level 1: Existing package semantics

Keep existing tests green:

- shared sections still work,
- local overrides still win,
- missing sections still fail.

### Level 2: Example-runner UX

Add tests for the chosen zero-arg contract.

Examples:

- `TestDefaultDirectoryUsesSelfContainedFixture`
- `TestExplicitRegistrySharedDirectoryStillRegistersHostSection`
- `TestNoDirectoryPrintsClearUsageError`

### Level 3: Manual smoke tests

Manual CLI runs matter here because the whole point of the ticket is example-runner usability.

## Alternatives Considered

### Alternative A: Make all `__section__` declarations globally visible across scanned files

Rejected for this ticket.

Why:

- contradicts current docs,
- changes semantics broadly,
- introduces ordering and duplicate-resolution complexity,
- harder to explain to users.

### Alternative B: Infer shared sections from fixture path names

Rejected.

Why:

- too magical,
- overfits repository layout,
- not reusable for embedded or in-memory scan modes.

### Alternative C: Keep current behavior and only document it

Rejected.

Why:

- zero-arg startup failure is a bad first-run experience,
- this is an example binary and should be friendlier.

## Risks

### Risk 1: Accidentally changing core section semantics while fixing CLI UX

Mitigation:

- avoid edits in `pkg/jsverbs` unless absolutely necessary,
- keep tests in `pkg/jsverbs/jsverbs_test.go` green,
- compare behavior before and after on the explicit registry-shared fixture.

### Risk 2: Picking the wrong zero-arg default

Mitigation:

- prefer the most documented self-contained fixture tree,
- if uncertainty remains, choose explicit `--dir` instead of magic.

### Risk 3: Docs drift after behavior change

Mitigation:

- update all three jsverbs example docs in the same change,
- include before/after commands in PR description or diary.

## Open Questions

1. Should `jsverbs-example` default to `./testdata/jsverbs`, or should it require `--dir` explicitly?
2. Does the team want a separate future ticket for "cross-file section sharing by scan scope," or is registry-level sharing the permanent model?
3. Should the example runner eventually grow named presets such as `--example basic` and `--example registry-shared` instead of path-based defaults?

## References

- [`cmd/jsverbs-example/main.go`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/cmd/jsverbs-example/main.go )
- [`pkg/jsverbs/model.go`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/pkg/jsverbs/model.go )
- [`pkg/jsverbs/scan.go`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/pkg/jsverbs/scan.go )
- [`pkg/jsverbs/binding.go`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/pkg/jsverbs/binding.go )
- [`pkg/jsverbs/command.go`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/pkg/jsverbs/command.go )
- [`pkg/jsverbs/runtime.go`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/pkg/jsverbs/runtime.go )
- [`pkg/jsverbs/jsverbs_test.go`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/pkg/jsverbs/jsverbs_test.go )
- [`pkg/doc/08-jsverbs-example-overview.md`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/pkg/doc/08-jsverbs-example-overview.md )
- [`pkg/doc/09-jsverbs-example-fixture-format.md`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/pkg/doc/09-jsverbs-example-fixture-format.md )
- [`pkg/doc/10-jsverbs-example-developer-guide.md`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/pkg/doc/10-jsverbs-example-developer-guide.md )
- [`pkg/doc/11-jsverbs-example-reference.md`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/pkg/doc/11-jsverbs-example-reference.md )
- [`testdata/jsverbs/basics.js`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/testdata/jsverbs/basics.js )
- [`testdata/jsverbs-example/registry-shared/issues.js`](/home/manuel/workspaces/2026-04-02/fix-goja-jsverbs/go-go-goja/testdata/jsverbs-example/registry-shared/issues.js )

## Proposed Solution

<!-- Describe the proposed solution in detail -->

## Design Decisions

<!-- Document key design decisions and rationale -->

## Alternatives Considered

<!-- List alternative approaches that were considered and why they were rejected -->

## Implementation Plan

<!-- Outline the steps to implement this design -->

## Open Questions

<!-- List any unresolved questions or concerns -->

## References

<!-- Link to related documents, RFCs, or external resources -->
