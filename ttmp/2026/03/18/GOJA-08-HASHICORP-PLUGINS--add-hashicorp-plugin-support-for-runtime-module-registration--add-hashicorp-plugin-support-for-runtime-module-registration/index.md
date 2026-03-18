---
Title: Add HashiCorp plugin support for runtime module registration
Ticket: GOJA-08-HASHICORP-PLUGINS--add-hashicorp-plugin-support-for-runtime-module-registration
Status: active
Topics:
    - goja
    - go
    - js-bindings
    - architecture
    - security
    - tooling
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: engine/factory.go
      Note: Current immutable factory build plan and runtime creation lifecycle
    - Path: engine/module_specs.go
      Note: Existing module registration seam that plugin support must extend
    - Path: engine/runtime.go
      Note: Current runtime close behavior and missing generic cleanup hooks
    - Path: modules/common.go
      Note: Native module contract and default global registry behavior
    - Path: modules/database/database.go
      Note: Example of stateful module receiver that highlights singleton-lifecycle risks
    - Path: pkg/runtimeowner/runner.go
      Note: Runtime owner goroutine contract that all plugin-backed JS calls must respect
    - Path: pkg/repl/evaluators/javascript/evaluator.go
      Note: Existing owned-runtime consumer that will need plugin configuration wiring
    - Path: ttmp/2026/03/18/GOJA-08-HASHICORP-PLUGINS--add-hashicorp-plugin-support-for-runtime-module-registration--add-hashicorp-plugin-support-for-runtime-module-registration/sources/local/Imported goja plugins note.md
      Note: Imported source memo that this ticket interprets and refines
ExternalSources:
    - local:Imported goja plugins note.md
Summary: Ticket workspace for a repo-grounded design of HashiCorp go-plugin support in go-go-goja, with an intern-facing implementation guide and imported source analysis.
LastUpdated: 2026-03-18T09:14:54.589316697-04:00
WhatFor: Explain how to add runtime-registered plugin modules to go-go-goja without violating goja runtime ownership, module lifecycle, or trust-boundary constraints.
WhenToUse: Use when implementing or reviewing HashiCorp plugin-based module loading in go-go-goja.
---


# Add HashiCorp plugin support for runtime module registration

## Overview

This ticket turns the imported plugin memo into a repo-specific plan for `go-go-goja`. The core conclusion is that the memo is directionally right about the trust boundary: the host process must remain the sole owner of `goja.Runtime`, while plugin subprocesses should be treated as RPC-backed capability providers. The repo, however, has some important local constraints that the imported note does not address on its own: `engine.Factory` currently freezes a single `require.Registry`, `modules.DefaultRegistry` stores singleton module receivers, and `Runtime.Close` only shuts down the runtime owner and event loop.

The primary deliverable is a detailed implementation guide for a new engineer. It explains the current runtime/module system first, then proposes the smallest coherent set of refactors needed to support runtime-scoped plugin discovery, manifest validation, native-module reification, and cleanup.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field
- **Primary analysis**: [design-doc/01-hashicorp-plugin-support-for-go-go-goja-architecture-and-implementation-guide.md](./design-doc/01-hashicorp-plugin-support-for-go-go-goja-architecture-and-implementation-guide.md)
- **Diary**: [reference/01-diary.md](./reference/01-diary.md)

## Status

Current status: **active**

## Topics

- goja
- go
- js-bindings
- architecture
- security
- tooling

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
