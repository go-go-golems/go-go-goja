---
Title: Add fs primitive module and ensure all goja_nodejs modules are require()-able
Ticket: GOJA-053
Status: active
Topics:
    - goja
    - modules
    - fs
    - nodejs-compat
    - goja-nodejs
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: engine/factory.go
      Note: Needs 3 Enable() calls for buffer
    - Path: engine/nodejs_init.go
      Note: goja_nodejs core module registration imports
    - Path: engine/performance.go
      Note: performance.now and console.time implementation
    - Path: engine/runtime.go
      Note: Needs 4 blank imports for goja_nodejs modules
    - Path: modules/common.go
      Note: NativeModule interface definition (reference)
    - Path: modules/crypto/crypto.go
      Note: crypto module implementation
    - Path: modules/fs/fs.go
      Note: Current fs module needing enhancement with 8 new functions
    - Path: modules/fs/fs_async.go
      Note: promise-based fs primitives
    - Path: modules/fs/fs_errors.go
      Note: fs Error object support
    - Path: modules/fs/fs_sync.go
      Note: synchronous fs wrappers
    - Path: modules/fs/fs_test.go
      Note: real runtime fs smoke tests
    - Path: modules/os/os.go
      Note: os module implementation
    - Path: modules/path/path.go
      Note: path module implementation
    - Path: modules/time/time.go
      Note: explicit time module implementation
ExternalSources: []
Summary: ""
LastUpdated: 2026-04-25T07:44:12.286692715-04:00
WhatFor: ""
WhenToUse: ""
---





# Add fs primitive module and ensure all goja_nodejs modules are require()-able

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- goja
- modules
- fs
- nodejs-compat
- goja-nodejs

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
