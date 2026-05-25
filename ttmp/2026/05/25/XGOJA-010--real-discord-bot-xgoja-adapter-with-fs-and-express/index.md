---
Title: Real discord-bot xgoja adapter with fs and express
Ticket: XGOJA-010
Status: complete
Topics:
    - xgoja
    - goja
    - providers
    - fs
    - architecture
    - command-registration
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: discord-bot/examples/xgoja/discord-bot-provider/README.md
      Note: Generated example and manual Discord test instructions
    - Path: discord-bot/internal/jsdiscord/host.go
      Note: Host-managed bot runs can use xgoja-provided runtimes
    - Path: discord-bot/pkg/botcli/command_root.go
      Note: Public Glazed command construction for provider mounting
    - Path: discord-bot/pkg/xgoja/provider/provider.go
      Note: Real xgoja provider for discord-bot modules and bots command provider
    - Path: go-go-goja/pkg/xgoja/providerapi/commands.go
      Note: CommandSetContext runtime profile field for provider adapters
ExternalSources: []
Summary: ""
LastUpdated: 2026-05-25T18:56:30.05668838-04:00
WhatFor: ""
WhenToUse: ""
---



# Real discord-bot xgoja adapter with fs and express

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
- discord
- bot
- fs
- express
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
