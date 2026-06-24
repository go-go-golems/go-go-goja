---
Title: Lazy data structures in Go exposed to JavaScript via goja Proxy
Ticket: GOJA-LAZY-DS
Status: closed
Topics:
    - goja
    - js-bindings
    - javascript
    - go
    - architecture
    - patterns
    - modules
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: modules/common.go
      Note: NativeModule/Registry/Loader — how a reusable 'lazy' module would register
    - Path: modules/lazy/lazy.go
      Note: core Map/Array/Record + flagOnlyDescriptor
    - Path: modules/lazy/lazy_test.go
      Note: enumeration-free + descriptor tradeoff regression tests
    - Path: modules/lazy/module.go
      Note: require('lazy') NativeModule + builders
    - Path: ttmp/2026/06/23/GOJA-LAZY-DS--lazy-data-structures-in-go-exposed-to-javascript-via-goja-proxy/design-doc/01-lazy-data-structures-over-goja-proxy-design-and-implementation-guide.md
      Note: primary intern-facing design guide
    - Path: ttmp/2026/06/23/GOJA-LAZY-DS--lazy-data-structures-in-go-exposed-to-javascript-via-goja-proxy/reference/01-investigation-diary.md
      Note: chronological investigation diary
    - Path: ttmp/2026/06/23/GOJA-LAZY-DS--lazy-data-structures-in-go-exposed-to-javascript-via-goja-proxy/scripts/03-lazy-map.go
      Note: 'evidence: naive descriptor materialises on enumeration'
    - Path: ttmp/2026/06/23/GOJA-LAZY-DS--lazy-data-structures-in-go-exposed-to-javascript-via-goja-proxy/scripts/06-cheap-enum.go
      Note: 'evidence: flag-only descriptor makes Object.keys free'
    - Path: ttmp/2026/06/23/GOJA-LAZY-DS--lazy-data-structures-in-go-exposed-to-javascript-via-goja-proxy/scripts/outputs.txt
      Note: captured verbatim run output of experiments 01-04
ExternalSources: []
Summary: ""
LastUpdated: 2026-06-23T21:36:21.634448553-04:00
WhatFor: ""
WhenToUse: ""
---































# Lazy data structures in Go exposed to JavaScript via goja Proxy

## Overview

Investigate and design how to expose lazily-materialised Go data structures
(on-demand maps, infinite arrays, dynamic key/value namespaces) to JavaScript
as ordinary objects/maps through goja's Proxy mechanism. The deliverables are
a runnable set of proof experiments, an intern-grade design & implementation
guide, and an investigation diary. A future phase builds a reusable Go `lazy`
package plus a `require("lazy")` JS module.

The central technical finding: goja's Go-facing `ProxyTrapConfig` lets you put
arbitrary Go logic behind every object operation, and — crucially — a
*flag-only* `GetOwnPropertyDescriptor` keeps `Object.keys()` / `for...in`
enumeration cheap (no value materialisation), while values are produced lazily
on the `Get` trap.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **in-progress** — analysis, design doc, diary, and experiments
complete; reMarkable upload pending.

## Topics

- goja
- js-bindings
- javascript
- go
- architecture
- patterns
- modules

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- **design-doc/01-...md** — the primary intern-grade design & implementation guide
- **reference/01-investigation-diary.md** — chronological investigation diary
- **scripts/** — runnable proof programs (00-go.mod … 06-cheap-enum.go) + outputs.txt
- playbooks/ - Command sequences and test procedures
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
