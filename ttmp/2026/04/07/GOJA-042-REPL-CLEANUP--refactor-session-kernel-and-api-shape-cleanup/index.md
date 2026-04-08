---
Title: Refactor REPL session kernel and API-shape cleanup
Ticket: GOJA-042-REPL-CLEANUP
Status: active
Topics:
    - goja
    - go
    - repl
    - refactor
    - architecture
    - documentation
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Detailed implementation guide for the cleanup/refactor follow-up after correctness fixes land, focused on service.go structure, API naming, and legacy evaluator consolidation planning."
LastUpdated: 2026-04-07T10:00:00-04:00
WhatFor: "Track the cleanup/refactor PR and provide an intern-oriented implementation guide."
WhenToUse: "Use when implementing or reviewing GOJA-042."
---

# Refactor REPL session kernel and API-shape cleanup

## Overview

This ticket is the cleanup follow-up after correctness fixes land. It is intentionally lower-priority than the persistence and evaluation-control tickets.

Main themes:

- split `pkg/replsession/service.go` by responsibility
- reduce API-shape confusion around duplicate `SessionOptions`
- decide what to do with the older evaluator path still used through the Bobatea adapter

The main design guide lives in `design-doc/01-repl-cleanup-analysis-design-and-implementation-guide.md`.
