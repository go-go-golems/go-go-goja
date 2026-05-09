---
Title: Upstream Express and ui.dsl modules into go-go-goja
Ticket: GO-GO-GOJA-EXPRESS-UIDSL-MODULES
Status: active
Topics:
    - goja
    - ui-dsl
    - web-ui
    - documentation
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../2026-05-03--goja-hosting-site/cmd/goja-site/main.go
      Note: goja-site verbs command registration
    - Path: modules/express
      Note: Runtime-scoped Express-style Goja module
    - Path: modules/uidsl
      Note: Reusable rich UI DSL module
    - Path: pkg/gojahttp
      Note: Upstream renderer-neutral HTTP host
    - Path: pkg/jsverbrepos
      Note: Reusable jsverbs repository discovery extracted from db-browser
    - Path: pkg/jsverbscli
      Note: Reusable jsverbs CLI shell extracted from db-browser
    - Path: ttmp/2026/05/08/GO-GO-GOJA-EXPRESS-UIDSL-MODULES--upstream-express-and-ui-dsl-modules-into-go-go-goja/design-doc/02-merge-db-browser-and-goja-hosting-site-web-shells.md
      Note: Design for retiring db-browser into the goja-site shell
    - Path: ttmp/2026/05/08/GO-GO-GOJA-EXPRESS-UIDSL-MODULES--upstream-express-and-ui-dsl-modules-into-go-go-goja/log/01-implementation-diary.md
      Note: Running diary for extraction and shell merge work
    - Path: ttmp/2026/05/08/GO-GO-GOJA-EXPRESS-UIDSL-MODULES--upstream-express-and-ui-dsl-modules-into-go-go-goja/tasks.md
      Note: Detailed task plan including shell merge phases
ExternalSources: []
Summary: ""
LastUpdated: 2026-05-08T14:29:03.028112443-04:00
WhatFor: ""
WhenToUse: ""
---







# Upstream Express and ui.dsl modules into go-go-goja

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- goja
- ui-dsl
- web-ui
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
