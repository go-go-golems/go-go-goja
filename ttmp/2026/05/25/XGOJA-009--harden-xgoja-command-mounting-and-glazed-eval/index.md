---
Title: Harden xgoja command mounting and Glazed eval
Ticket: XGOJA-009
Status: complete
Topics:
    - xgoja
    - goja
    - providers
    - command-registration
    - architecture
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/examples/xgoja/module-sections/Makefile
      Note: Smokes eval with module-provided fixture section
    - Path: go-go-goja/pkg/xgoja/app/command_providers.go
      Note: Clones command descriptions before applying command-provider mount parents
    - Path: go-go-goja/pkg/xgoja/app/host.go
      Note: Threads generated root output writer into eval command
    - Path: go-go-goja/pkg/xgoja/app/root.go
      Note: Converted eval to Glazed command with module sections and runtime initializers
ExternalSources: []
Summary: ""
LastUpdated: 2026-05-25T11:45:14.617944845-04:00
WhatFor: ""
WhenToUse: ""
---



# Harden xgoja command mounting and Glazed eval

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
- command-registration
- architecture

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
