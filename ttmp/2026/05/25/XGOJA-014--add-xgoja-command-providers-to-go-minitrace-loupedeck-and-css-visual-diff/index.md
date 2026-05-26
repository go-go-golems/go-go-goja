---
Title: Add xgoja command providers to go-minitrace, loupedeck, and css-visual-diff
Ticket: XGOJA-014
Status: active
Topics:
    - xgoja
    - providers
    - command-registration
    - goja
    - documentation
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-05-24/add-js-providers/css-visual-diff/internal/cssvisualdiff/dsl/registrar.go
      Note: Runtime-module and loader refactor for css-visual-diff diff/report modules
    - Path: /home/manuel/workspaces/2026-05-24/add-js-providers/css-visual-diff/internal/cssvisualdiff/jsapi/module.go
      Note: Existing css-visual-diff JS API installer that must become xgoja-loader-friendly
    - Path: /home/manuel/workspaces/2026-05-24/add-js-providers/css-visual-diff/internal/cssvisualdiff/verbcli/command.go
      Note: Existing css-visual-diff verb command builder to reuse from command provider
    - Path: /home/manuel/workspaces/2026-05-24/add-js-providers/css-visual-diff/pkg/xgoja/provider/provider.go
      Note: Public css-visual-diff xgoja module and command provider
    - Path: /home/manuel/workspaces/2026-05-24/add-js-providers/go-minitrace/cmd/go-minitrace/cmds/query/catalog_commands.go
      Note: Reusable catalog-to-Glazed command helper for go-minitrace command provider
    - Path: /home/manuel/workspaces/2026-05-24/add-js-providers/go-minitrace/pkg/minitracejs/provider/provider.go
      Note: Existing go-minitrace xgoja package provider to receive the queries command provider
    - Path: /home/manuel/workspaces/2026-05-24/add-js-providers/loupedeck/cmd/loupedeck/cmds/verbs/command.go
      Note: Exported non-Cobra Loupedeck verb command helper
    - Path: /home/manuel/workspaces/2026-05-24/add-js-providers/loupedeck/runtime/js/provider/provider.go
      Note: Existing loupedeck xgoja provider to receive the scenes command provider
ExternalSources: []
Summary: Track implementation of xgoja command providers for go-minitrace, loupedeck, and css-visual-diff.
LastUpdated: 2026-05-25T20:05:00-04:00
WhatFor: Coordinate cross-repository command-provider implementation and validation.
WhenToUse: Use when reviewing or resuming XGOJA-014 work.
---



# Add xgoja command providers to go-minitrace, loupedeck, and css-visual-diff

## Overview

Add xgoja `CommandSetProvider` support to three sibling packages. `go-minitrace` should expose its repository-backed query catalog; `loupedeck` should expose hardware scene/verb commands; `css-visual-diff` should gain a first public xgoja provider plus workflow verb command provider. The downstream packages should use the published `go-go-goja v0.5.0` provider API rather than local replaces.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- xgoja
- providers
- command-registration
- goja
- documentation

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
