---
Title: Pluggable invoker API for jsverbs Commands
Ticket: GOJA-JSVERBS-INVOKER
Status: active
Topics:
    - goja
    - javascript
    - cli
    - documentation
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-04-13/js-loupedeck/go-go-goja/pkg/jsverbs/command.go
      Note: Command wrapper generation and injected invoker hook
    - Path: /home/manuel/workspaces/2026-04-13/js-loupedeck/go-go-goja/pkg/jsverbs/runtime.go
      Note: Existing runtime-owning invoke path and caller-owned InvokeInRuntime API
    - Path: /home/manuel/workspaces/2026-04-13/js-loupedeck/go-go-goja/pkg/jsverbs/jsverbs_test.go
      Note: Regression coverage for injected invokers and default fallback behavior
    - Path: /home/manuel/workspaces/2026-04-13/js-loupedeck/go-go-goja/pkg/doc/10-jsverbs-example-developer-guide.md
      Note: Developer-facing explanation of default versus host-owned execution
    - Path: /home/manuel/workspaces/2026-04-13/js-loupedeck/go-go-goja/pkg/doc/11-jsverbs-example-reference.md
      Note: Reference-level API surface for scanning, command generation, and runtime ownership
ExternalSources: []
Summary: Minimal upstream design for letting jsverbs-generated commands use caller-provided execution strategies while preserving the existing runtime-owning default path.
LastUpdated: 2026-04-18T12:45:00-04:00
WhatFor: Explain and document the smallest jsverbs API extension needed for host applications that want generated commands but need to keep ownership of runtime/session execution.
WhenToUse: Read before changing pkg/jsverbs command generation, adding more host-integration APIs, or wiring jsverbs into applications with long-lived runtimes or sessions.
---

# Pluggable invoker API for jsverbs Commands

## Executive Summary

`pkg/jsverbs` already had the hard parts needed by host applications with special runtime ownership requirements: `CommandDescriptionForVerb(...)`, `InvokeInRuntime(...)`, and `RequireLoader()`. The missing convenience layer was command generation that could reuse the normal Glazed command wrappers without hardwiring execution to `registry.invoke(...)`.

This ticket adds a small upstream API for that gap:

- `Registry.CommandsWithInvoker(...)`
- `Registry.CommandForVerb(...)`
- `Registry.CommandForVerbWithInvoker(...)`
- `VerbInvoker`

The existing `Registry.Commands()` behavior remains the default and simply delegates to `CommandsWithInvoker(nil)`. That keeps the simple path simple, while giving host applications a clean hook for caller-owned runtime execution.

## Problem Statement

Before this change, `Registry.Commands()` did two jobs at once:

1. compile scanned `VerbSpec` values into Glazed command wrappers
2. decide that those wrappers must execute through `registry.invoke(...)`

That coupling was fine for small tools like `cmd/jsverbs-example`, but it created friction for host applications that need generated command schemas while keeping runtime or session ownership on the host side.

Typical examples include:

- hardware-backed runtimes
- long-lived UI/event-loop runtimes
- application-specific session state
- runtimes that must remain alive after the initial command returns

Those applications could already bypass `Commands()` and manually compose `CommandDescriptionForVerb(...)` with custom execution code, but that duplicated wrapper glue that `pkg/jsverbs` should be able to provide generically.

## Proposed Solution

Add a minimal injected-invoker abstraction in `pkg/jsverbs/command.go`.

### New API surface

```go
type VerbInvoker func(ctx context.Context, registry *Registry, verb *VerbSpec, parsedValues *values.Values) (interface{}, error)

func (r *Registry) CommandsWithInvoker(invoker VerbInvoker) ([]cmds.Command, error)
func (r *Registry) CommandForVerb(verb *VerbSpec) (cmds.Command, error)
func (r *Registry) CommandForVerbWithInvoker(verb *VerbSpec, invoker VerbInvoker) (cmds.Command, error)
```

### Default behavior

The old default behavior remains intact:

```go
func (r *Registry) Commands() ([]cmds.Command, error) {
    return r.CommandsWithInvoker(nil)
}
```

When the invoker is `nil`, generated commands still execute through the existing runtime-owning `registry.invoke(...)` path.

### Custom behavior

When the invoker is non-nil, the generated `Command` or `WriterCommand` wrapper calls the injected invoker instead of calling `registry.invoke(...)` directly.

That means a host application can:

1. scan files into a `Registry`
2. build normal jsverbs-generated commands
3. parse values through the generated Glazed schema
4. execute the selected verb inside caller-owned runtime/session logic

The invoker still receives the `Registry`, `VerbSpec`, and parsed values, so a host can delegate to `InvokeInRuntime(...)` if that is the right execution model.

## Design Decisions

### Decision 1: Keep `Commands()` as the stable default

**Why:** existing tools already use `Commands()` successfully. The default runtime-owning path is still the right convenience API for `jsverbs-example` and similar small tools.

### Decision 2: Add explicit injected-invoker variants instead of a broad options struct

**Why:** the missing concept is narrow. A function callback is enough to separate command generation from execution policy without introducing a larger configuration framework.

### Decision 3: Support both bulk and single-verb command construction

**Why:** host applications sometimes want to build a whole command tree, and sometimes want just one wrapper for a selected verb. Both are useful and cheap to expose once the invoker is factored out.

### Decision 4: Keep output-mode handling inside `pkg/jsverbs`

**Why:** structured-output and text-output wrappers should still be generated centrally. Host applications should override execution policy, not rewrite wrapper selection and result rendering logic.

## Alternatives Considered

### Alternative A: Do nothing and keep manual host wrappers outside `pkg/jsverbs`

Pros:

- no upstream code change

Cons:

- forces host applications to duplicate wrapper glue
- leaves an obvious reuse seam unexposed
- makes future host integrations needlessly inconsistent

Rejected.

### Alternative B: Change `Registry.Commands()` to always require an invoker

Pros:

- one API path only

Cons:

- breaks existing callers
- makes simple use cases more verbose
- adds migration noise for little gain

Rejected.

### Alternative C: Introduce a large command-builder/options abstraction

Pros:

- potentially extensible for future options

Cons:

- more complexity than the current problem requires
- risks turning `pkg/jsverbs` into a bigger framework than needed

Rejected for now.

## Implementation Plan

1. Add `VerbInvoker` and optional invoker storage on the generated command wrapper types.
2. Refactor `Registry.Commands()` to delegate to `CommandsWithInvoker(nil)`.
3. Add `CommandForVerb(...)` and `CommandForVerbWithInvoker(...)`.
4. Update `Command` and `WriterCommand` execution methods to use the injected invoker when present.
5. Add regression tests for:
   - structured-output custom invoker path
   - text-output custom invoker path
   - nil-invoker fallback to the old behavior
6. Update jsverbs package docs to explain the new host-owned execution hook.

## Open Questions

There are no blocking open questions for the upstream API added in this ticket.

Future work, if needed, should stay narrow:

- better helper docs showing `InvokeInRuntime(...)` inside a host-owned invoker
- downstream adoption in callers that currently hand-roll command wrappers

## References

- `pkg/jsverbs/command.go`
- `pkg/jsverbs/runtime.go`
- `pkg/jsverbs/jsverbs_test.go`
- `pkg/doc/10-jsverbs-example-developer-guide.md`
- `pkg/doc/11-jsverbs-example-reference.md`
