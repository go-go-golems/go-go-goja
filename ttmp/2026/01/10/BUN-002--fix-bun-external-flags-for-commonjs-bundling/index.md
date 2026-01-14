---
Title: Fix bun external flags for CommonJS bundling
Ticket: BUN-002
Status: complete
Topics:
    - bun
    - bundling
    - build
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Fix bun build external flag syntax so native module requires are not resolved by the bundler.
LastUpdated: 2026-01-14T16:03:35.242959739-05:00
WhatFor: Track the fix for bun external flags in the CommonJS demo pipeline.
WhenToUse: When reviewing ticket status or related docs.
---



# Fix bun external flags for CommonJS bundling

## Overview
`bun build` fails to treat native module specifiers (`exec`, `fs`, `database`) as external due to incorrect flag syntax, blocking the CommonJS demo pipeline.

## Key Links
- [Bun external flag failure analysis](./analysis/01-bun-external-flag-failure-analysis.md)
- [Diary](./reference/01-diary.md)

## Status
Current status: **active**

## Tasks
See [tasks.md](./tasks.md).

## Changelog
See [changelog.md](./changelog.md).
