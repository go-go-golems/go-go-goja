---
Title: Code Review Bug Analysis
Ticket: GOJA-002-FIX-INSPECTOR-CR-ISSUES
Status: active
Topics:
    - goja
    - analysis
    - tooling
    - bugfix
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/cmd/inspector/app/model.go
      Note: Drawer go-to-definition and usage highlight safety paths
    - Path: go-go-goja/cmd/inspector/app/model_test.go
      Note: Inspector model regression coverage additions
    - Path: go-go-goja/pkg/jsparse/resolve.go
      Note: Scope resolver for for-in/of and function/arrow parameter defaults
    - Path: go-go-goja/pkg/jsparse/resolve_test.go
      Note: Resolver regression tests for all reviewed issues
ExternalSources: []
Summary: Root-cause analysis of code-review findings and closely related resolver stability issues discovered during targeted audit.
LastUpdated: 2026-02-12T19:08:00-05:00
WhatFor: Provide implementation-ready bug breakdown and acceptance criteria before patching code.
WhenToUse: Use while implementing or reviewing fixes for GOJA-002 code-review findings.
---

# Code Review Bug Analysis

## Goal

Analyze and fix code-review findings in inspector/jsparse while proactively finding adjacent issues in the same execution paths.

## Reported Findings (from review comments)

1. `cmd/inspector/app/model.go`: panic in drawer go-to-definition when lookup returns no binding (`binding == nil` then dereferenced).
2. `cmd/inspector/app/model.go`: panic in drawer highlight-usages when unresolved identifier path dereferences `binding.AllUsages()` on nil.
3. `pkg/jsparse/resolve.go`: `for-in` / `for-of` resolves source/body only; misses assignment target resolution for expression form (`for (x in obj)` / `for (x of arr)`).
4. `pkg/jsparse/resolve.go`: function parameter default initializers are resolved after body declaration collection, allowing invalid links to body-local hoisted names.

## Additional Similar Issues Found

### A. `for (var x in/of ...)` declarations are not collected

Current declaration collection handles `ForDeclaration` (`let`/`const`) but skips `ForIntoVar` (`var`). This can leave loop variable bindings absent from scope graph and downstream features.

### B. Arrow-function parameter default initializers are never resolved

`resolveArrowFunction` binds params and resolves body, but does not resolve `param.Initializer`. This causes missing references and unresolved tracking gaps for defaults like `(a = ext) => ...`.

## Reproduction Summary

Static inspection and targeted probes showed:
- drawer `ctrl+d` / `ctrl+g` paths can dereference nil `binding` in unresolved-name scenarios.
- `ForInStatement` / `ForOfStatement` handling currently omits `Into` resolution.
- function default initializer resolution order allows incorrect body-local binding capture.
- arrow defaults are not resolved at all.

## Root Cause Analysis

1. Missing nil guard in drawer lookup code:
   - code scans all scopes, assigns first binding match, then unconditionally dereferences.
2. Same pattern in highlight-usages:
   - unresolved lookup can return nil, then `binding.AllUsages()` panics.
3. Resolver control-flow omission in for-in/of:
   - `resolveStatement` handles only `Source` and `Body`.
   - declaration collector handles only `ForDeclaration`, missing `ForIntoVar`.
4. Resolver ordering bug in function literals:
   - collects body declarations before evaluating default initializers.
   - defaults should resolve in function-parameter environment before body-hoisted locals.
5. Missing coverage in arrow function parameter defaults:
   - no default-initializer resolve pass exists in `resolveArrowFunction`.

## Fix Strategy

1. Inspector safety:
   - add explicit `binding == nil` guards in drawer go-to-definition/highlight paths.
   - unresolved symbol action should no-op safely (or clear highlight if applicable).
2. For-in/of resolver correctness:
   - resolve `Into` target when `Into` is `ForIntoExpression`.
   - collect and bind declarations for `ForIntoVar`.
3. Function/arrow default semantics:
   - resolve defaults after parameter bindings but before body declaration collection.
   - add same behavior for arrow functions.
4. Tests:
   - inspector model regression tests for unresolved drawer actions.
   - resolver tests for for-in/of target coverage, for-var loop binding, function default scoping, and arrow default resolution.

## Acceptance Criteria

- No panic on unresolved drawer identifiers for `ctrl+d` or `ctrl+g`.
- `for (x in/of ...)` target identifiers are recorded as references when appropriate.
- `for (var x in/of ...)` loop variable is correctly bound/resolved.
- `function f(a = b) { var b = ... }` does not resolve default `b` to body-local declaration.
- Arrow defaults (`(a = b) => ...`) resolve identifiers through enclosing scope and are tracked.
- Tests and lint pass.

## Validation Plan

1. `GOWORK=off go test ./cmd/inspector/... -count=1`
2. `GOWORK=off go build ./cmd/inspector`
3. `GOWORK=off go test ./pkg/jsparse -count=1`
4. `GOWORK=off go test ./... -count=1`
5. `make lint`
