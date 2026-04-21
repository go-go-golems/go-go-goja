---
Title: Configurable replapi profiles and policies
Ticket: GOJA-23-CONFIGURABLE-REPLAPI
Status: active
Topics:
    - persistent-repl
    - architecture
    - repl
    - refactor
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Implementation ticket for converting replapi into a profile-based API with explicit session policy and for adopting that API in the persistent CLI and the traditional line REPL.
LastUpdated: 2026-04-03T19:40:11.173430794-04:00
WhatFor: Use this ticket when implementing or reviewing the refactor that turns replapi into an opinionated but configurable API spanning raw execution, interactive REPL use, and fully persistent auditable sessions.
WhenToUse: Use when working on replapi configuration, replsession evaluation policy, session creation options, or the adoption path for cmd/js-repl and cmd/goja-repl.
---

# Configurable replapi profiles and policies

## Overview

This ticket refactors `replapi` from a single hard-wired persistent behavior into a configurable API with clear profiles and feature policies. The goal is to let callers choose a mode that ranges from almost-straight Goja execution up to the full persistent REPL stack with replay restore, binding tracking, JSDoc extraction, and durable history.

The design target is opinionated defaults without hidden behavior. Callers should be able to say "raw", "interactive", or "persistent" and get a coherent preset, while still overriding specific policies when necessary.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- persistent-repl
- architecture
- repl
- refactor

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
