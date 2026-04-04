---
Title: Diary
Ticket: GOJA-24-REPL-TUI-UNIFICATION
Status: active
Topics:
    - repl
    - tui
    - cli
    - architecture
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/goja-repl/root.go
      Note: Primary evidence for unified-binary direction
    - Path: cmd/js-repl/main.go
      Note: Primary evidence for current TUI bootstrap
    - Path: pkg/repl/evaluators/javascript/evaluator.go
      Note: Primary evidence for evaluator responsibility split
    - Path: pkg/replapi/app.go
      Note: Primary evidence for shared session API adoption
ExternalSources: []
Summary: ""
LastUpdated: 2026-04-04T15:32:08.343140703-04:00
WhatFor: ""
WhenToUse: ""
---


# Diary

## Goal

Capture the analysis and design work for migrating the Bubble Tea `js-repl` onto the shared `replapi` session core and unifying the user-facing TUI under `goja-repl tui`.

## Step 1: Analyze the split REPL architecture and define the unification plan

This step established the new ticket, mapped the current REPL surfaces, and turned the user's high-level product direction into a concrete migration design. The key outcome was confirming that the hard part is not the Bubble Tea UI itself; it is separating execution/session ownership from the old monolithic evaluator without losing completion/help behavior.

The analysis also made the binary-level decision explicit. `goja-repl` already has the right long-term shape as the one command root, so the migration guide centers on adding `goja-repl tui` rather than extending the standalone `cmd/js-repl` command further.

### Prompt Context

**User prompt (verbatim):** "create a new ticket to do 1. + 2. (one binary goje-repl tui) 3. ok 4. ok 

reate a detailed analysis / design / implementation guide that is very detailed for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file
  references.
  It should be very clear and detailed. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Create a new docmgr ticket for migrating `js-repl` to the shared REPL core and merging it into `goja-repl tui`, then write a detailed evidence-backed design and implementation guide and publish it to reMarkable.

**Inferred user intent:** Turn the next REPL phase into an actionable engineering ticket that a new contributor can pick up, while keeping the ticket chain and documentation quality consistent with the earlier REPL work.

**Commit (code):** N/A

### What I did
- Read the `ticket-research-docmgr-remarkable` skill and its writing/checklist references.
- Inspected existing REPL tickets `GOJA-20` through `GOJA-23` to keep the new ticket aligned with the current architectural direction.
- Created the new ticket workspace:
  - `GOJA-24-REPL-TUI-UNIFICATION`
- Added:
  - the primary design document
  - the diary document
- Collected evidence from:
  - `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/cmd/js-repl/main.go`
  - `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/cmd/goja-repl/root.go`
  - `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/cmd/repl/main.go`
  - `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/repl/adapters/bobatea/javascript.go`
  - `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/repl/evaluators/javascript/evaluator.go`
  - `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replapi/app.go`
  - `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replapi/config.go`
- Wrote the primary design guide with:
  - current-state analysis
  - gap analysis
  - proposed architecture
  - command/API sketches
  - diagrams
  - phased implementation plan
  - testing strategy
  - risks and alternatives

### Why
- The new ticket needed to be grounded in current code, not just in prior design discussions.
- The most important design choice was whether to merge at the binary level now or later; the evidence showed that `goja-repl` is already the natural root command and that keeping `cmd/js-repl` separate would only preserve duplication.
- A new intern will need a document that explains not just "what to code", but also "where the seams are" and "what not to redesign in this ticket".

### What worked
- The current codebase already exposes a clean contrast between the old evaluator path and the new `replapi` path, which made the migration narrative straightforward.
- The adapter boundary in `pkg/repl/adapters/bobatea` is a good leverage point for the migration.
- The earlier tickets already narrowed the design space enough that this ticket could focus on integration instead of re-litigating the core architecture.

### What didn't work
- My first attempt to inspect `/pkg/repl/adapters/bobatea/evaluator.go` failed because the file does not exist; the adapter file is actually `/pkg/repl/adapters/bobatea/javascript.go`.

Exact command and error:

```bash
sed -n '1,260p' /home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/repl/adapters/bobatea/evaluator.go
```

```text
sed: can't read /home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/repl/adapters/bobatea/evaluator.go: No such file or directory
```

### What I learned
- The old evaluator path is even more monolithic than it first appears: it owns runtime construction, evaluation, promise handling, console setup, completion, help bar, help drawer, and docs resolver state.
- The binary merge is not the hard part. The hard part is preserving TUI assistance behavior while moving execution ownership onto `replapi`.
- `cmd/repl` already demonstrates the minimal execution migration pattern, which is a useful sanity check for the larger TUI migration.

### What was tricky to build
- The main design challenge was avoiding two opposite mistakes:
  - making the migration sound too small, as if the old evaluator could simply be swapped out without touching assistance logic
  - making the migration sound too large, as if the Bubble Tea UI needed a full rewrite
- The right framing was to separate the backend into two roles:
  - session execution
  - editor assistance
- Once those roles were named explicitly, the design became much more precise and teachable.

### What warrants a second pair of eyes
- The proposed assistance extraction boundary. That is the area most likely to hide unexpected coupling.
- The `goja-repl tui` startup contract, especially around `interactive` versus `persistent` default behavior.
- Whether to delete `cmd/js-repl` immediately or keep a one-step wrapper very briefly for developer ergonomics.

### What should be done in the future
- Implement the ticket in phases exactly as described in the design doc.
- After the migration lands, follow with cleanup/hardening:
  - delete obsolete prototype paths
  - strengthen adapter/TUI regression tests

### Code review instructions
- Start with the design doc:
  - `design-doc/01-js-repl-migration-to-replapi-and-goja-repl-tui-unification-guide.md`
- Then review the evidence anchors in the code:
  - `cmd/js-repl/main.go`
  - `pkg/repl/adapters/bobatea/javascript.go`
  - `pkg/repl/evaluators/javascript/evaluator.go`
  - `cmd/goja-repl/root.go`
  - `pkg/replapi/app.go`
- Validate the ticket bookkeeping:
  - `index.md`
  - `tasks.md`
  - `changelog.md`

### Technical details
- Ticket creation:

```bash
docmgr ticket create-ticket --ticket GOJA-24-REPL-TUI-UNIFICATION --title "Migrate js-repl onto replapi and unify under goja-repl tui" --topics repl,tui,cli,architecture
```

- Document creation:

```bash
docmgr doc add --ticket GOJA-24-REPL-TUI-UNIFICATION --doc-type design-doc --title "js-repl migration to replapi and goja-repl tui unification guide"
docmgr doc add --ticket GOJA-24-REPL-TUI-UNIFICATION --doc-type reference --title "Diary"
```

## Context

Use this diary alongside the main design doc while implementing the ticket. The design doc explains the target state and migration plan. The diary explains how the plan was derived and where the sharp edges are likely to be.

## Quick Reference

- Ticket root:
  - `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/ttmp/2026/04/04/GOJA-24-REPL-TUI-UNIFICATION--migrate-js-repl-onto-replapi-and-unify-under-goja-repl-tui`
- Primary design doc:
  - `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/ttmp/2026/04/04/GOJA-24-REPL-TUI-UNIFICATION--migrate-js-repl-onto-replapi-and-unify-under-goja-repl-tui/design-doc/01-js-repl-migration-to-replapi-and-goja-repl-tui-unification-guide.md`

## Usage Examples

- Before implementation:
  - read the executive summary, current-state analysis, and implementation phases
- During implementation:
  - update this diary as each migration phase lands
- During review:
  - use the code-review instructions in Step 1 to trace evidence and validate scope

## Related

- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/ttmp/2026/04/03/GOJA-22-PERSISTENT-REPL-CLI-SERVER--persistent-repl-cli-and-json-server-surfaces/design-doc/01-cli-and-json-server-implementation-plan.md`
- `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/ttmp/2026/04/03/GOJA-23-CONFIGURABLE-REPLAPI--configurable-replapi-profiles-and-policies/design-doc/01-configurable-replapi-profiles-and-policies-implementation-plan.md`
