---
Title: Make jsverbs Commands support caller-provided invokers
Ticket: GOJA-JSVERBS-INVOKER
Status: active
Topics:
    - goja
    - javascript
    - cli
    - documentation
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-04-13/js-loupedeck/go-go-goja/pkg/jsverbs/command.go
      Note: Added the pluggable invoker API for generated jsverbs command wrappers
    - Path: /home/manuel/workspaces/2026-04-13/js-loupedeck/go-go-goja/pkg/jsverbs/runtime.go
      Note: Existing runtime-owning and caller-owned invocation primitives that shaped the design
    - Path: /home/manuel/workspaces/2026-04-13/js-loupedeck/go-go-goja/pkg/jsverbs/jsverbs_test.go
      Note: Regression coverage for the new invoker-aware command generation paths
ExternalSources: []
Summary: Upstream ticket for separating jsverbs command generation from execution policy by adding invoker-aware command wrapper helpers while preserving the existing default path.
LastUpdated: 2026-04-18T12:45:00-04:00
WhatFor: Use this ticket when reviewing or extending the upstream jsverbs command-generation APIs for host applications that need caller-owned runtime execution.
WhenToUse: Open this workspace when implementing or adopting the injected-invoker jsverbs command API.
---

# Make jsverbs Commands support caller-provided invokers

## Overview

This ticket captures and implements a small upstream `pkg/jsverbs` API extension that lets host applications build jsverbs-generated Glazed commands while keeping execution policy under host control.

The implemented result is:

- `Registry.Commands()` remains the simple default path
- `Registry.CommandsWithInvoker(...)` exposes invoker-aware bulk command generation
- `Registry.CommandForVerb(...)` and `Registry.CommandForVerbWithInvoker(...)` expose the same idea for single verbs
- generated wrappers still preserve structured-output versus text-output behavior

## Key Links

- Design doc: `design-doc/01-pluggable-invoker-api-for-jsverbs-commands.md`
- Implementation diary: `reference/01-implementation-diary-for-jsverbs-pluggable-invoker-work.md`
- Tasks: `tasks.md`
- Changelog: `changelog.md`

## Status

Current status: **active**

## Scope

### In scope

- adding a small injected-invoker seam to jsverbs-generated command wrappers
- preserving the current default `Registry.Commands()` behavior
- documenting the new API for host applications
- validating structured-output, text-output, and fallback behavior

### Out of scope

- changing runtime ownership semantics in the default path
- repository discovery or Cobra tree placement for host applications
- downstream adoption work in other repos

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.
