---
Title: xgoja HTTP Express provider and Discord outbound runtime API
Ticket: XGOJA-011
Status: active
Topics:
    - xgoja
    - goja
    - providers
    - fs
    - architecture
    - command-registration
    - goja-nodejs
    - modules
    - runtime
    - web-ui
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: discord-bot/internal/jsdiscord/runtime.go
      Note: Discord module needs top-level outbound runtime API
    - Path: discord-bot/pkg/xgoja/provider/provider.go
      Note: Discord command provider needs selected module section aggregation and runtime initialization
    - Path: go-go-goja/modules/express/express.go
      Note: Existing Express module that needs xgoja provider/lifecycle integration
    - Path: go-go-goja/pkg/gojahttp/host.go
      Note: HTTP host used by Express route registration and server serving
    - Path: go-go-goja/pkg/xgoja/providerapi/capabilities.go
      Note: Provider capability interfaces for module sections and runtime initializers
ExternalSources: []
Summary: ""
LastUpdated: 2026-05-25T13:00:13.764139612-04:00
WhatFor: ""
WhenToUse: ""
---


# xgoja HTTP Express provider and Discord outbound runtime API

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- xgoja
- goja
- providers
- fs
- architecture
- command-registration
- goja-nodejs
- modules
- runtime
- web-ui

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
