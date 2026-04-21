---
Title: Fix REPL persistence correctness and SQLite integrity
Ticket: GOJA-040-PERSISTENCE-CORRECTNESS
Status: active
Topics:
    - goja
    - go
    - sqlite
    - repl
    - architecture
    - documentation
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Detailed implementation guide for fixing deleted-session semantics, durable session ID generation, and SQLite connection integrity in the REPL persistence layer."
LastUpdated: 2026-04-07T10:00:00-04:00
WhatFor: "Track the persistence-correctness PR and provide an intern-oriented implementation guide."
WhenToUse: "Use when implementing or reviewing the persistence correctness PR."
---

# Fix REPL persistence correctness and SQLite integrity

## Overview

This ticket covers the highest-priority persistence fixes for the REPL stack:

- deleted sessions must disappear from normal reads
- durable session IDs must be unique across separate processes
- SQLite integrity settings must apply consistently across pooled connections

The main design guide lives in `design-doc/01-persistence-correctness-analysis-design-and-implementation-guide.md`.

## Documents

- `design-doc/01-persistence-correctness-analysis-design-and-implementation-guide.md`
- `reference/01-investigation-diary.md`
- `tasks.md`
- `changelog.md`
