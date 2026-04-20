---
Title: Scriptable JS Sandbox API and Runtime Module Surface
Ticket: GOJA-17-JS-SANDBOX-API
Status: active
Topics:
    - goja
    - js-bindings
    - architecture
    - documentation
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ttmp/2026/04/20/GOJA-17-JS-SANDBOX-API--scriptable-js-sandbox-api-and-runtime-module-surface/design-doc/01-js-sandbox-host-api-and-runtime-architecture.md
      Note: Primary sandbox host architecture and implementation guide
    - Path: ttmp/2026/04/20/GOJA-17-JS-SANDBOX-API--scriptable-js-sandbox-api-and-runtime-module-surface/reference/01-js-sandbox-api-reference-and-example-bots.md
      Note: Compact API reference and example bot scripts
    - Path: ttmp/2026/04/20/GOJA-17-JS-SANDBOX-API--scriptable-js-sandbox-api-and-runtime-module-surface/reference/02-diary.md
      Note: Chronological work log for the sandbox API ticket
ExternalSources: []
Summary: Ticket workspace for a runtime-scoped JS sandbox API that makes go-go-goja hosts scriptable.
LastUpdated: 2026-04-20T11:06:13.752344184-04:00
WhatFor: Track the design guide, API reference, diary, and implementation follow-ups for the JS sandbox host API.
WhenToUse: Use as the landing page for the sandbox API ticket.
---


# Scriptable JS Sandbox API and Runtime Module Surface

## Overview

This ticket captures the design for a CommonJS-based JS sandbox API on top of the existing go-go-goja runtime. The goal is to let a host application load a JS script, expose a small capability-based API, and let that script register commands and event handlers while using an in-memory store.

The key design choice is to keep the sandbox API runtime-scoped and host-owned. The existing `pkg/jsverbs` package remains the precedent for JS-to-Glazed command scanning, but the sandbox API itself is a separate host capability layer.

## Key Links

- [JS Sandbox Host API and Runtime Architecture](./design-doc/01-js-sandbox-host-api-and-runtime-architecture.md)
- [JS Sandbox API Reference and Example Bots](./reference/01-js-sandbox-api-reference-and-example-bots.md)
- [Diary](./reference/02-diary.md)

## Status

Current status: **active**

## Topics

- goja
- js-bindings
- architecture
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
