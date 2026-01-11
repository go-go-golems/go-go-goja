---
Title: Diary
Ticket: BUN-005
Status: active
Topics:
    - bun
    - bundling
    - goja
    - commonjs
    - docs
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-10T21:53:58.634516305-05:00
WhatFor: ""
WhenToUse: ""
---

# Diary

## Goal

Track the work to add a Model B split-bundle demo for Goja, including Go embed updates and documentation.

## Step 1: Ticket scaffolding and split-bundle plan

I created the ticket workspace for the split bundle demo and captured the planned architecture. This establishes the intended directory layout, build steps, and the minimal Go changes needed to load a runtime module graph.

The analysis doc now documents the split-bundle approach (multiple entrypoints, externalized modules, and a directory embed) so implementation can follow a clear path.

### What I did
- Created the BUN-005 ticket workspace with analysis and diary docs.
- Added tasks for the split demo, Go embed changes, and documentation updates.
- Wrote the initial split-bundle plan in the analysis doc.

### Why
- The work spans JS, Go, and documentation updates, so a ticket ensures consistent tracking.
- Writing the plan first reduces risk of missing path updates in the bundling pipeline.

### What worked
- The ticket workspace and initial analysis were created successfully.

### What didn't work
- N/A

### What I learned
- The split-bundle demo can reuse the existing loader with a broader `//go:embed` scope.

### What was tricky to build
- N/A

### What warrants a second pair of eyes
- N/A

### What should be done in the future
- N/A

### Code review instructions
- Start in `/home/manuel/workspaces/2026-01-10/package-bun-goja-js/go-go-goja/ttmp/2026/01/10/BUN-005--split-bundle-demo-for-goja/analysis/01-split-bundle-demo-plan.md`.
- Confirm the tasks list aligns with the planned implementation steps.

### Technical details
- Ticket created with `docmgr ticket create-ticket --ticket BUN-005 --title "Split bundle demo for Goja" --topics bun,bundling,goja,commonjs,docs`.
