---
Title: Implementation guide
Ticket: xgoja-jsverbs-loader-only-sources
Status: active
Topics:
    - xgoja
    - jsverbs
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - path: /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/cmd/xgoja/internal/buildspec/build_spec.go
      note: Buildspec JSVerbSourceSpec must not grow compatibility options
    - path: /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/xgoja/app/runtime_spec.go
      note: Runtime JSVerbSourceSpec must stay option-free
    - path: /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/xgoja/app/root.go
      note: Maps source filters to jsverbs.ScanOptions
    - path: /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/jsverbs/model.go
      note: ScanOptions no longer exposes public-function discovery
    - path: /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/jsverbs/scan.go
      note: Explicit __verb__ metadata is now the only command discovery path
    - path: /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/xgoja/providers/http/serve.go
      note: HTTP serve consumes registry verbs and registry require loader
ExternalSources: []
Summary: "Clean-break implementation guide for making jsverbs/xgoja command discovery explicit: only __verb__ metadata creates commands, while scanned helper files remain require-loader visible."
LastUpdated: 2026-06-09T01:20:00-04:00
WhatFor: "Implementation plan for removing implicit public-function command discovery from jsverbs and xgoja."
WhenToUse: "Use when modifying jsverbs scanning, xgoja jsverb sources, or HTTP serve command-provider module loading."
---

# Implementation guide

## Executive Summary

`jsverbs` now uses a clean explicit contract: a JavaScript function becomes a command only when the file declares matching `__verb__()` metadata. Top-level public functions are no longer auto-promoted into commands.

This fixes xgoja HTTP serve applications that include normal CommonJS helper files. The scanner can still include `server.js` and `lib/**/*.js` so `Registry.RequireLoader()` can resolve `require("./server.js")`, but helper functions in those files remain helpers unless they have `__verb__()` metadata.

## Problem Statement

Before this change, `jsverbs.ScanOptions` had `IncludePublicFunctions`, defaulted it to `true`, and `scan.go` created implicit commands for every public top-level function. That was convenient for small CLI sketches but unsafe for applications with helper modules.

The failure mode was:

1. `site.js` exposes `__verb__("start", ...)` and calls `require("./server.js")`.
2. If only `site.js` is scanned, `serve site start` exists but `require("./server.js")` fails because the registry loader does not know `server.js`.
3. If `server.js` and `lib/**/*.js` are scanned, `require()` can resolve them, but helper functions become unintended commands.

## Proposed Solution

Remove implicit public-function discovery entirely.

The only command declaration model is now:

```js
function start(options) {
  require("./server.js");
}

__verb__("start", {
  name: "start",
  short: "Start the site",
  fields: {
    options: { bind: "all" }
  }
});
```

The corresponding xgoja source can include helper files without extra options:

```yaml
jsverbs:
  - id: site
    path: .
    include:
      - site.js
      - server.js
      - lib/**/*.js
```

## Design Decisions

### Decision 1: No compatibility option

Status: accepted.

The user explicitly requested a clean break: remove the option and backwards-compatibility behavior. Therefore there is no `includePublicFunctions` field and no scanner flag for legacy public-function discovery.

### Decision 2: Keep helper files loader-visible

Status: accepted.

Scanned files are still registered in the `Registry.RequireLoader()` map. The only removed behavior is implicit command creation. This keeps xgoja HTTP serve apps ergonomic without creating a separate loader-only source model yet.

### Decision 3: Update fixtures to declare commands explicitly

Status: accepted.

Any test fixture or example that intends to expose a command must include `__verb__()`. This makes examples match the new contract.

## Implementation Plan

1. Delete `ScanOptions.IncludePublicFunctions` from `pkg/jsverbs/model.go`.
2. Delete the implicit public-function loop in `pkg/jsverbs/scan.go::finalizeVerbs`.
3. Remove the transient `includePublicFunctions` field from xgoja buildspec/runtime/provider descriptor structs.
4. Remove xgoja scan-option wiring for the deleted field.
5. Add/adjust tests:
   - explicit commands still work,
   - helper files remain available through `Registry.RequireLoader()`,
   - HTTP serve can require a helper module without exposing helper commands.
6. Update docs to say `__verb__()` metadata is required.
7. Validate targeted packages and then the ClubMed minitrace-viz runtime path.
8. Commit code and docs at a coherent interval.

## Validation

Passing targeted validation:

```bash
go test ./pkg/jsverbs ./pkg/xgoja/app ./cmd/xgoja/internal/... ./pkg/xgoja/providers/http
```

## Open Questions

- Whether to add a future `loaderInclude`/`verbInclude` split for more precise source modeling. This is not required for the clean-break fix.

## References

- `/home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/jsverbs/model.go`
- `/home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/jsverbs/scan.go`
- `/home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/xgoja/app/root.go`
- `/home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/xgoja/providers/http/serve_test.go`
