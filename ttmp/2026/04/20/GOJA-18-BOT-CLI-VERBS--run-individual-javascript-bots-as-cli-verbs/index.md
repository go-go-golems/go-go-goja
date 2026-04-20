---
Title: Run Individual JavaScript Bots as CLI Verbs
Ticket: GOJA-18-BOT-CLI-VERBS
Status: active
Topics:
    - goja
    - javascript
    - cli
    - cobra
    - glazed
    - bots
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Ticket workspace for designing a stable `go-go-goja bots list|run|help` command surface on top of the existing jsverbs and engine runtime layers.
LastUpdated: 2026-04-20T12:45:00-04:00
WhatFor: Organize the design and research work for running individual JavaScript bots as CLI verbs.
WhenToUse: Start here when orienting to the GOJA-18 ticket.
---

# Run Individual JavaScript Bots as CLI Verbs

## Overview

This ticket defines how `go-go-goja` should expose JavaScript bots as command-line verbs through a stable operator-facing surface:

```text
go-go-goja bots list
go-go-goja bots run <verb>
go-go-goja bots help <verb>
```

The core design direction is to reuse the existing `pkg/jsverbs` scan/describe/invoke pipeline, borrow the strongest repository bootstrap and runtime wrapper ideas from `loupedeck`, and keep the newer sandbox `defineBot(...)` API as a separate runtime concern rather than conflating it with CLI discovery.

## Key documents

- [Design doc](./design-doc/01-bot-cli-verbs-architecture-and-implementation-guide.md)
- [Command/API reference](./reference/01-bot-cli-verbs-command-surface-and-api-reference.md)
- [Diary](./reference/02-diary.md)
- [Tasks](./tasks.md)
- [Changelog](./changelog.md)

## Status

Current status: **active**.

The research and design bundle is complete. The next major step would be to implement the proposed `pkg/botcli` package and root CLI command structure.

## Recommended reading order

1. Read the design doc for the full architecture story.
2. Read the quick reference for the concrete command/API checklist.
3. Read the diary if you need chronological context or delivery evidence.
4. Read `tasks.md` for the proposed implementation phases.

## Structure

- `design-doc/` — long-form architecture and implementation guide
- `reference/` — quick reference and diary
- `playbooks/` — future validation/runbooks if implementation proceeds
- `scripts/` — ticket-local helper scripts if later needed
