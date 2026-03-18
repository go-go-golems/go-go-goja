---
Title: jsverbs shared sections design and implementation guide
Ticket: GOJA-07-JSVERBS-SHARED-SECTIONS--implement-registry-level-shared-sections-for-jsverbs
Status: active
Topics:
    - go
    - glazed
    - js-bindings
    - architecture
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/jsverbs-example/main.go
      Note: Example runner and future integration point for host-provided shared sections
    - Path: pkg/doc/09-jsverbs-example-fixture-format.md
      Note: User-facing metadata contract that will need updated shared-section documentation
    - Path: pkg/jsverbs/binding.go
      Note: Current file-local section validation and binding-plan construction
    - Path: pkg/jsverbs/command.go
      Note: Current Glazed schema generation path that consumes referenced sections
    - Path: pkg/jsverbs/jsverbs_test.go
      Note: Existing tests that should gain shared-section coverage
    - Path: pkg/jsverbs/model.go
      Note: |-
        Current Registry, FileSpec, SectionSpec, and VerbSpec structures that must evolve for shared sections
        Current Registry
    - Path: pkg/jsverbs/scan.go
      Note: Current scan-time ownership of sections and file-local verb finalization
    - Path: ttmp/2026/03/17/GOJA-06-JSVERBS-DB-SHARED-LIBS--investigate-db-flags-shared-libs-and-bundling-for-jsverbs--investigate-db-flags-shared-libs-and-bundling-for-jsverbs/design-doc/01-jsverbs-db-flags-shared-libraries-and-bundling-investigation-guide.md
      Note: Prior investigation that established the current limitation and motivated this ticket
ExternalSources: []
Summary: Detailed intern-oriented design and implementation guide for adding registry-level shared sections to jsverbs so multiple scripts can reuse a shared section catalog while keeping metadata scanning strict and predictable.
LastUpdated: 2026-03-17T14:00:25.446945137-04:00
WhatFor: Explain how jsverbs works today, why cross-file shared sections fail, and how to implement registry-level shared sections safely and incrementally.
WhenToUse: Use when implementing or reviewing the shared-sections feature, onboarding a new engineer to jsverbs internals, or planning follow-on runner APIs built on top of shared sections.
---


# jsverbs shared sections design and implementation guide

## Executive Summary

`pkg/jsverbs` already has the right abstraction pieces for shared sections, but they are currently wired together in a file-local way. Sections are collected into `FileSpec.Sections` during scanning, verbs are finalized against functions in that same file, and the binding plan validates referenced section slugs only against `verb.File.Sections`. That design keeps metadata deterministic, but it prevents a common and useful pattern: one shared section catalog for flags like `db`, `auth`, `profile`, or `output`, reused across many jsverb files.

This ticket proposes a registry-level shared-section layer that sits above file-local sections without replacing them. The key idea is simple:

1. keep file-local sections exactly as they work today,
2. add `Registry.SharedSections` and `Registry.SharedSectionOrder`,
3. resolve section references against local sections first and shared sections second,
4. keep `require()` for runtime code reuse, but do not make metadata depend on runtime execution.

The recommended design preserves the current strict scanning model, keeps existing jsverbs compatible, and gives Go runners a clean place to inject shared CLI schemas. For an intern, the most important conceptual split is this: `require()` is a runtime code-loading tool, but shared sections must be a scan-time schema feature.

## Problem Statement

Today, this JavaScript structure looks natural but does not work:

```js
// common.js
__section__("db", {
  fields: {
    db: { default: ":memory:" }
  }
});

exports.connect = function(dbConfig) {
  const db = require("database");
  db.configure("sqlite3", dbConfig.db);
  return db;
};
```

```js
// users.js
const common = require("./common");

function listUsers(prefix, dbConfig) {
  const db = common.connect(dbConfig);
  return db.query("SELECT ...", prefix + "%");
}

__verb__("listUsers", {
  sections: ["db"],
  fields: {
    prefix: { argument: true },
    dbConfig: { bind: "db" }
  }
});
```

The runtime helper part works, but the shared section part fails. The verb in `users.js` does not see the `db` section from `common.js`, because the current binding-plan logic validates referenced sections against the file-local map only.

The feature we want is a system where a shared section provider can define reusable flag sets and many scripts can consume them without copying the same `__section__` block into every file. The new design must satisfy four constraints:

1. It must preserve strict metadata scanning.
2. It must not require evaluating JavaScript at scan time.
3. It must preserve backward compatibility for existing file-local sections.
4. It must be easy for a Go-based runner to opt into shared sections.

## Proposed Solution

Add a registry-level shared-section catalog and make section resolution two-tiered:

```text
lookup(slug):
  1. check verb.File.Sections
  2. if not found, check registry.SharedSections
  3. if still not found, return the current unknown-section error
```

The public API should remain small and explicit. I recommend the following shape:

```go
type Registry struct {
    RootDir            string
    Files              []*FileSpec
    Diagnostics        []Diagnostic
    SharedSections     map[string]*SectionSpec
    SharedSectionOrder []string

    verbs         []*VerbSpec
    verbsByKey    map[string]*VerbSpec
    filesByModule map[string]*FileSpec
    options       ScanOptions
}

func (r *Registry) AddSharedSection(section *SectionSpec) error
func (r *Registry) AddSharedSections(sections ...*SectionSpec) error
func (r *Registry) ResolveSection(verb *VerbSpec, slug string) (*SectionSpec, bool)
func (r *Registry) SectionSource(verb *VerbSpec, slug string) string
```

The main code-path change is that binding-plan construction and command description generation must stop assuming that every section comes from `verb.File.Sections`.

### Core flow after the change

```text
Scan files
  -> file-local sections stored on each FileSpec
Construct registry
  -> host optionally adds shared sections
Build commands
  -> binding plan resolves local-first, shared-second
Generate schema
  -> section fields copied from whichever catalog provided the slug
Invoke verb
  -> runtime behavior unchanged
```

### Why this is the right layer

`require()` cannot solve this cleanly because:

- `require()` runs too late,
- `__section__` is part of the static metadata contract,
- metadata is intentionally documented as declarative and strict in [pkg/doc/09-jsverbs-example-fixture-format.md](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/doc/09-jsverbs-example-fixture-format.md#L53).

So section sharing must be implemented where schema is built, not where JS code executes.

## Current System Map

### Registry and file model

The current `Registry` stores scanned files, diagnostics, verbs, and module lookup maps, but no concept of shared schema in [pkg/jsverbs/model.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/model.go#L74-L82).

Each `FileSpec` owns:

- `Functions`
- `SectionOrder`
- `Sections`
- `VerbMeta`

in [pkg/jsverbs/model.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/model.go#L84-L96).

That ownership model is the root reason cross-file sections fail: there is no registry-global section namespace.

### Scan-time section extraction

During scanning, `handleSection` parses `__section__` metadata and writes the resulting `SectionSpec` only into `e.file.Sections` and `e.file.SectionOrder` in [pkg/jsverbs/scan.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/scan.go#L472-L509).

There is no second write to a registry-level catalog. The scanner therefore preserves file-local section ownership by construction.

### Verb finalization

When `finalizeVerb` runs, it associates the verb with a `FileSpec` and copies function parameters from that file in [pkg/jsverbs/scan.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/scan.go#L289-L329).

This part does not need to change for shared sections. A verb should still belong to one source file. The section feature should expand section resolution, not verb ownership.

### Binding-plan validation

`buildVerbBindingPlan` is where the feature currently fails. After collecting referenced section slugs, it validates every slug against `verb.File.Sections` only in [pkg/jsverbs/binding.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/binding.go#L117-L129).

This is the conceptual bottleneck:

```go
if _, ok := verb.File.Sections[slug]; !ok {
    return nil, fmt.Errorf("%s references unknown section %q", verb.SourceRef(), slug)
}
```

### Command generation

`buildDescription` iterates over `plan.ReferencedSections` and then pulls every referenced section from `verb.File.Sections[slug]` in [pkg/jsverbs/command.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/command.go#L115-L133).

So even if validation were changed, command generation would still need to be updated to fetch the actual section spec from local-or-shared resolution.

## Design Decisions

### Decision 1: Local sections remain first-class and keep precedence

If both a file-local section and a shared section have slug `db`, the file-local section wins.

Why:

- the file author is closest to the concrete command contract,
- existing files continue to behave exactly as before,
- local override is easier to reason about than a hidden merge.

### Decision 2: Shared sections are injected from Go, not discovered from `require()`

Why:

- keeps scan-time metadata deterministic,
- avoids evaluating arbitrary code during discovery,
- matches the repo’s preference for explicit composition.

### Decision 3: One resolver must be used everywhere

The same local-first/shared-second rule must be reused in:

- binding-plan validation,
- section ordering,
- command generation,
- diagnostics.

Why:

- prevents schema/runtime drift,
- gives one place to reason about section source.

### Decision 4: Do not auto-promote file-local sections into a registry-global pool

Why:

- hidden global promotion would create surprising coupling,
- helper files could accidentally influence unrelated commands,
- duplicate slug handling would become much more confusing.

## Alternatives Considered

### Alternative A: Use `require()` to import metadata

Rejected because:

- `require()` executes at runtime after command schema generation,
- metadata would become dynamic,
- it would violate the current strict metadata model.

### Alternative B: Auto-promote every `__section__` into a registry-global namespace

Rejected because:

- unrelated files would start affecting each other implicitly,
- duplicate slugs become harder to interpret,
- the boundary between local intent and global schema disappears.

### Alternative C: Put all shared sections in a completely separate runner-only schema type

Partially workable, but unnecessarily duplicative.

Using the existing `SectionSpec` and `FieldSpec` keeps the model unified. The better move is to let runners inject shared sections using the same structure the scanner already emits.

## Implementation Plan

### 1. Extend `Registry` in `pkg/jsverbs/model.go`

Current registry state is in [pkg/jsverbs/model.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/model.go#L74-L82). Add:

```go
SharedSections     map[string]*SectionSpec
SharedSectionOrder []string
```

Also add helpers such as:

```go
func (r *Registry) AddSharedSection(section *SectionSpec) error
func (r *Registry) AddSharedSections(sections ...*SectionSpec) error
func (r *Registry) ResolveSection(verb *VerbSpec, slug string) (*SectionSpec, bool)
```

Behavior rules:

- reject `nil`,
- reject empty slugs,
- clone sections before storing,
- reject duplicate shared slugs,
- do not mutate file-local sections.

Suggested pseudocode:

```go
func (r *Registry) AddSharedSection(section *SectionSpec) error {
    if section == nil {
        return fmt.Errorf("shared section is nil")
    }
    slug := cleanCommandWord(section.Slug)
    if slug == "" {
        return fmt.Errorf("shared section slug is empty")
    }
    if r.SharedSections == nil {
        r.SharedSections = map[string]*SectionSpec{}
    }
    if _, exists := r.SharedSections[slug]; exists {
        return fmt.Errorf("duplicate shared section %q", slug)
    }
    cloned := section.Clone()
    cloned.Slug = slug
    r.SharedSections[slug] = cloned
    r.SharedSectionOrder = append(r.SharedSectionOrder, slug)
    return nil
}
```

### 2. Initialize new fields in `pkg/jsverbs/scan.go`

The registry is constructed in [pkg/jsverbs/scan.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/scan.go#L159-L167). Initialize:

```go
SharedSections:     map[string]*SectionSpec{},
SharedSectionOrder: []string{},
```

Do not change `handleSection` in [pkg/jsverbs/scan.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/scan.go#L472-L509). Sections found in JS should remain file-local unless a runner deliberately registers them as shared.

### 3. Change binding-plan construction in `pkg/jsverbs/binding.go`

Current code validates sections against `verb.File.Sections` only in [pkg/jsverbs/binding.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/binding.go#L117-L129).

Change:

```go
func buildVerbBindingPlan(verb *VerbSpec) (*VerbBindingPlan, error)
```

to:

```go
func buildVerbBindingPlan(r *Registry, verb *VerbSpec) (*VerbBindingPlan, error)
```

Then validate with:

```go
for slug := range referencedSections {
    if _, ok := r.ResolveSection(verb, slug); !ok {
        return nil, fmt.Errorf("%s references unknown section %q", verb.SourceRef(), slug)
    }
}
```

Section ordering should be updated to:

1. include referenced file-local sections in `verb.File.SectionOrder`,
2. then include referenced shared sections in `r.SharedSectionOrder`.

### 4. Change schema generation in `pkg/jsverbs/command.go`

`buildDescription` currently pulls section specs from `verb.File.Sections[slug]` in [pkg/jsverbs/command.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/command.go#L115-L133).

Replace that with:

```go
spec, ok := r.ResolveSection(verb, slug)
if !ok {
    return nil, fmt.Errorf("%s references unknown section %q", verb.SourceRef(), slug)
}
```

That keeps validation and command compilation aligned.

### 5. Update runtime-side binding-plan calls in `pkg/jsverbs/runtime.go`

No runtime semantics need to change, but the binding plan should still be built through the registry-aware function:

```go
plan, err := buildVerbBindingPlan(r, verb)
```

This keeps argument binding consistent with the new section resolver.

### 6. Add tests in `pkg/jsverbs/jsverbs_test.go`

Existing tests already cover:

- virtual-file scanning in [pkg/jsverbs/jsverbs_test.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/jsverbs_test.go#L162-L197),
- unknown bound section failure in [pkg/jsverbs/jsverbs_test.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/jsverbs_test.go#L216-L233).

Add new tests for:

```go
func TestCommandsUseRegistrySharedSection(t *testing.T)
func TestLocalSectionOverridesRegistrySharedSection(t *testing.T)
func TestAddSharedSectionRejectsDuplicateSlug(t *testing.T)
func TestSharedSectionsWorkWithScanFS(t *testing.T)
func TestUnknownSectionStillFailsWhenAbsentFromBothCatalogs(t *testing.T)
```

Test expectations:

- a command may bind to `db` without declaring `__section__("db", ...)` locally if the registry has a shared `db` section,
- a local `db` section overrides the shared one,
- duplicate shared-section registration returns an error,
- ordering is stable.

### 7. Update user-facing docs in `pkg/doc/09-jsverbs-example-fixture-format.md`

The current doc says `__section__` defines reusable sections in [pkg/doc/09-jsverbs-example-fixture-format.md](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/doc/09-jsverbs-example-fixture-format.md#L30-L33), but it does not clarify scope.

Update it to say:

- `__section__` defines file-local sections,
- Go runners may now inject registry-level shared sections,
- `require()` still does not import metadata.

## Implementation Guide For A New Intern

### Mental model

Before touching code, keep these invariants in mind:

1. A verb always belongs to one source file.
2. A section slug can come from either the verb file or the registry shared catalog.
3. The same resolution rule must be used by both validation and command generation.
4. Runtime invocation should not care where the section came from.

### Suggested sequence

1. Read [pkg/jsverbs/model.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/model.go#L74-L150) to understand the current types.
2. Read [pkg/jsverbs/scan.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/scan.go#L207-L242) and [pkg/jsverbs/scan.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/scan.go#L472-L550) to see where files and sections are built.
3. Read [pkg/jsverbs/binding.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/binding.go#L40-L129) and mark the places that assume file-local section ownership.
4. Read [pkg/jsverbs/command.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/command.go#L72-L177) and note where section specs are fetched.
5. Change the data model first.
6. Add resolver helpers second.
7. Update binding-plan logic third.
8. Update command/runtime callers fourth.
9. Add tests before changing docs.
10. Update docs after tests pass.

### Suggested pseudocode patch order

```text
model.go
  add SharedSections + AddSharedSection + ResolveSection

scan.go
  initialize new registry fields

binding.go
  change buildVerbBindingPlan signature
  validate sections through ResolveSection
  compute ordered referenced sections from local + shared order

command.go
  fetch section specs through ResolveSection

runtime.go
  pass registry into buildVerbBindingPlan

jsverbs_test.go
  add success and precedence tests

pkg/doc
  document the new contract
```

## Testing Strategy

### Unit tests

Required unit-test coverage:

- `AddSharedSection` success,
- duplicate slug rejection,
- empty slug rejection,
- shared section used by command generation,
- local override of shared section,
- unknown section still fails when absent from both catalogs.

### Integration-style tests

Use `ScanSource` or `ScanFS` to keep tests small and focused. A good pattern is:

```go
registry, err := ScanFS(fstest.MapFS{ ... }, ".")
require.NoError(t, err)

require.NoError(t, registry.AddSharedSection(...))

commands, err := registry.Commands()
require.NoError(t, err)
```

### Documentation validation

Once docs are updated:

- ensure examples use the real API shape,
- verify there is no implication that `require()` imports sections,
- keep wording aligned with the fixture-format reference.

## Rollout Plan

### Phase 1

- implement the core API and tests,
- no runner changes yet.

### Phase 2

- update `cmd/jsverbs-example` or a new example runner to inject a shared `db` section,
- add a small example showing one shared section plus one shared runtime helper.

### Phase 3

- consider whether higher-level runner configuration helpers are worth adding.

## Open Questions

- Should local-over-shared precedence be fixed, or should there also be a strict duplicate-rejection mode later?
- Should the public API expose section-source introspection for debugging and tests?
- Should `cmd/jsverbs-example` be updated in the same PR or as a follow-up once the core package is stable?

## References

- [pkg/jsverbs/model.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/model.go)
- [pkg/jsverbs/scan.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/scan.go)
- [pkg/jsverbs/binding.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/binding.go)
- [pkg/jsverbs/command.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/command.go)
- [pkg/jsverbs/jsverbs_test.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/jsverbs_test.go)
- [pkg/doc/09-jsverbs-example-fixture-format.md](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/doc/09-jsverbs-example-fixture-format.md)
- [cmd/jsverbs-example/main.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/cmd/jsverbs-example/main.go)
- [GOJA-06 prior investigation](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/ttmp/2026/03/17/GOJA-06-JSVERBS-DB-SHARED-LIBS--investigate-db-flags-shared-libs-and-bundling-for-jsverbs--investigate-db-flags-shared-libs-and-bundling-for-jsverbs/design-doc/01-jsverbs-db-flags-shared-libraries-and-bundling-investigation-guide.md)
