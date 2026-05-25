---
Title: XGOJA-014 implementation diary
Ticket: XGOJA-014
Status: active
Topics:
  - xgoja
  - command-providers
  - diary
DocType: reference
Intent: diary
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Chronological implementation diary for adding command providers to go-minitrace, loupedeck, and css-visual-diff."
LastUpdated: 2026-05-25T20:05:00-04:00
WhatFor: "Preserve investigation context, commands run, decisions, errors, and validation outcomes."
WhenToUse: "Read before resuming XGOJA-014 work or reviewing the implementation."
---

# XGOJA-014 implementation diary

## 2026-05-25 20:05 — Ticket creation and initial analysis

The user asked for command provider code in three sibling repositories: `go-minitrace`, `loupedeck`, and `css-visual-diff`. They also noted that `go-go-goja@0.5.0` has been published, which means the downstream repos should depend on the released command-provider API instead of workspace-local replaces.

I created ticket `XGOJA-014 — Add xgoja command providers to go-minitrace, loupedeck, and css-visual-diff` under the `go-go-goja/ttmp` doc root. I inspected each package:

- `go-minitrace` already has `pkg/minitracejs/provider` for the `minitrace` module and reusable Glazed catalog commands in `cmd/go-minitrace/cmds/query`.
- `loupedeck` already has `runtime/js/provider` for safe `easing`/`gfx` modules plus Glazed `run` and annotated verb commands under `cmd/loupedeck/cmds`.
- `css-visual-diff` has no public xgoja provider yet; it has internal runtime-module registration through `internal/cssvisualdiff/jsapi` and jsverb command discovery through `internal/cssvisualdiff/verbcli`.

Wrote three package-specific implementation guides:

1. `design/01-go-minitrace-command-provider-guide.md`
2. `design/02-loupedeck-command-provider-guide.md`
3. `design/03-css-visual-diff-command-provider-guide.md`

Initial implementation strategy:

- Start with `go-minitrace` because it has the smallest surface: expose catalog query commands and construction-only tests.
- Then `loupedeck`: expose `run` plus annotated verb commands without opening hardware in tests.
- Then `css-visual-diff`: add a real public xgoja provider and command provider; this is the largest step because internal runtime-module code must become loader-friendly.
