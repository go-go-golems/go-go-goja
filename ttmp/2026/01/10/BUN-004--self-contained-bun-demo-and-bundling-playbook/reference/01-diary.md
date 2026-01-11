---
Title: Diary
Ticket: BUN-004
Status: active
Topics:
    - bun
    - bundling
    - docs
    - typescript
    - goja
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: .gitignore
      Note: Ignore paths for relocated JS workspace
    - Path: Makefile
      Note: Removed bun demo targets
    - Path: cmd/bun-demo/Makefile
      Note: Local demo build targets
    - Path: cmd/bun-demo/js/package.json
      Note: Bun build config now co-located with demo
    - Path: pkg/doc/bun-goja-bundling-playbook.md
      Note: Bundling playbook document
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-10T21:16:45.122940517-05:00
WhatFor: ""
WhenToUse: ""
---



# Diary

## Goal

Track the work to relocate the bun demo workspace under `cmd/bun-demo` and produce a detailed bundling playbook.

## Step 1: Ticket scaffolding and initial plan

I set up a new ticket workspace for the demo relocation and documentation work, then documented the intended layout and build flow. This gives us a place to track tasks, diary entries, and follow-up changes while we restructure the demo.

The initial analysis notes capture the new directory layout and the Makefile move so we can execute the file moves with fewer surprises.

### What I did
- Created the BUN-004 ticket workspace, analysis doc, and diary.
- Added tasks for moving the JS workspace, updating Makefiles, refreshing the bundle, and writing the playbook.
- Wrote the initial analysis describing the target layout and validation steps.

### Why
- The work spans multiple files and documentation deliverables, so a dedicated ticket keeps the changes organized.
- A pre-move analysis helps avoid missed path updates when relocating the JS workspace.

### What worked
- The ticket workspace, analysis doc, and task list were created successfully.
- The analysis doc now captures the target file layout and build flow.

### What didn't work
- N/A

### What I learned
- The new ticket index and analysis frontmatter are a good place to record the target layout for the relocation.

### What was tricky to build
- N/A

### What warrants a second pair of eyes
- N/A

### What should be done in the future
- N/A

### Code review instructions
- Start in `/home/manuel/workspaces/2026-01-10/package-bun-goja-js/go-go-goja/ttmp/2026/01/10/BUN-004--self-contained-bun-demo-and-bundling-playbook/index.md` and `/home/manuel/workspaces/2026-01-10/package-bun-goja-js/go-go-goja/ttmp/2026/01/10/BUN-004--self-contained-bun-demo-and-bundling-playbook/analysis/01-bun-demo-relocation-and-playbook-plan.md`.
- Confirm the tasks in `/home/manuel/workspaces/2026-01-10/package-bun-goja-js/go-go-goja/ttmp/2026/01/10/BUN-004--self-contained-bun-demo-and-bundling-playbook/tasks.md` match the intended scope.

### Technical details
- Ticket created with `docmgr ticket create-ticket --ticket BUN-004 --title "Self-contained bun demo and bundling playbook" --topics bun,bundling,docs,typescript,goja`.

## Step 2: Relocate bun demo workspace and Makefile

I moved the bun demo JS workspace under `cmd/bun-demo`, added a local Makefile for the demo workflow, and updated the ignore rules and root Makefile. This consolidates the demo into a single directory and keeps bundling steps tied to the demo itself.

I also rebuilt and ran the demo from the new location to confirm the bundled output still loads in Goja and prints the expected SVG metrics.

**Commit (code):** 255f4f3 — "Move bun demo workspace under cmd/bun-demo"

### What I did
- Moved the `js/` workspace to `cmd/bun-demo/js`.
- Added `cmd/bun-demo/Makefile` with bun install/build/run targets.
- Removed bun targets from the root `Makefile` and updated `.gitignore`.
- Ran `make -C cmd/bun-demo go-run-bun` to rebuild and validate output.
- Recorded commit metadata in a follow-up commit (`6e7e983`) for `.git-commit-message.yaml`.

### Why
- Keeping the demo assets and tooling under `cmd/bun-demo` makes the sample self-contained and easier to share.
- The local Makefile reflects the actual demo workflow without relying on root-level targets.

### What worked
- `make -C cmd/bun-demo go-run-bun` rebuilt the bundle and printed the expected SVG metrics string.
- The Goja demo continued to load the embedded CommonJS bundle after the move.

### What didn't work
- N/A

### What I learned
- The bundling flow remains intact after moving the workspace as long as the asset copy stays inside `cmd/bun-demo`.

### What was tricky to build
- Ensuring the root Makefile no longer referenced the moved JS workspace while keeping the demo targets intact.

### What warrants a second pair of eyes
- Confirm the new Makefile layout aligns with the expected repository-wide tooling conventions.

### What should be done in the future
- N/A

### Code review instructions
- Start in `/home/manuel/workspaces/2026-01-10/package-bun-goja-js/go-go-goja/cmd/bun-demo/Makefile` and `/home/manuel/workspaces/2026-01-10/package-bun-goja-js/go-go-goja/cmd/bun-demo/js/package.json`.
- Verify the root `Makefile` no longer contains bun demo targets and `.gitignore` now points at `cmd/bun-demo/js`.
- Run `make -C cmd/bun-demo go-run-bun` to validate the demo output.

### Technical details
- Build command: `make -C cmd/bun-demo go-run-bun`.

## Step 3: Author the bundling playbook

I wrote the full Bun + TypeScript + asset bundling playbook in `pkg/doc` so users have a single, detailed reference for building CommonJS bundles for Goja. The document walks through the architecture, file layout, build commands, and the Go loader integration.

The playbook now documents the recommended Makefile workflow, CommonJS considerations, and a troubleshooting checklist tailored to the Goja runtime.

**Commit (code):** 768e9c3 — "Docs: add bun bundling playbook"

### What I did
- Authored the playbook under `pkg/doc/bun-goja-bundling-playbook.md`.
- Included step-by-step setup, code snippets, and validation instructions aligned with the demo layout.

### Why
- The project needed a durable, copy/paste-friendly guide that captures the bundling pipeline in one place.

### What worked
- The playbook covers the full pipeline from Bun install to Goja execution.

### What didn't work
- N/A

### What I learned
- Keeping CommonJS output and ES5 targets front-and-center makes the documentation clearer for Goja users.

### What was tricky to build
- Balancing detailed examples with concise, runnable snippets in a single document.

### What warrants a second pair of eyes
- Confirm the playbook matches current demo paths and Makefile targets after the relocation.

### What should be done in the future
- N/A

### Code review instructions
- Start in `/home/manuel/workspaces/2026-01-10/package-bun-goja-js/go-go-goja/pkg/doc/bun-goja-bundling-playbook.md`.
- Verify the file layout and commands match `cmd/bun-demo/Makefile` and `cmd/bun-demo/js`.

### Technical details
- Document path: `/home/manuel/workspaces/2026-01-10/package-bun-goja-js/go-go-goja/pkg/doc/bun-goja-bundling-playbook.md`.
