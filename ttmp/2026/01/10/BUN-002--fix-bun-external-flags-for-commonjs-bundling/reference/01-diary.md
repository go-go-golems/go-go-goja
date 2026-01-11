---
Title: Diary
Ticket: BUN-002
Status: active
Topics:
    - bun
    - bundling
    - build
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ttmp/2026/01/10/BUN-002--fix-bun-external-flags-for-commonjs-bundling/analysis/01-bun-external-flag-failure-analysis.md
      Note: Analysis referenced in Step 1
ExternalSources: []
Summary: Implementation diary for fixing bun external flags.
LastUpdated: 2026-01-10T20:30:17-05:00
WhatFor: Track investigation and fixes for bun bundling failures.
WhenToUse: When reviewing or continuing the ticket.
---


# Diary

## Goal
Track the investigation and fix for bun build failing to externalize native module specifiers.

## Step 1: Reproduce failure and open BUN-002

I ran the requested bundling commands and reproduced a `bun build` failure where the native `exec` module could not be resolved. Since this blocks the CommonJS demo pipeline, I opened a new ticket (BUN-002) to track the fix, wrote the initial analysis, and added tasks to guide the work.

This step captures the exact error output and establishes the plan to correct the bundler flag syntax in both the Makefile and package.json.

### What I did
- Ran `make js-bundle` and `make go-run-bun` to reproduce the failure.
- Created ticket BUN-002 and added an analysis doc and diary.
- Added tasks covering the flag fix, validation runs, and documentation updates.

### Why
- The demo pipeline is blocked until bun treats native modules as externals.
- A dedicated ticket keeps the fix scoped and traceable.

### What worked
- The failure reproduced consistently, which confirms the issue is in the current bundling flags.

### What didn't work
- `make js-bundle` failed with:
  ```
  cd js && bun build --target=browser --format=cjs --outfile=dist/bundle.cjs src/main.js --external:fs --external:exec --external:database
  4 | var exec = require("exec");
                         ^
  error: Could not resolve: "exec". Maybe you need to "bun install"?
      at /home/manuel/workspaces/2026-01-10/package-bun-goja-js/go-go-goja/js/src/main.js:4:20
  make: *** [Makefile:15: js-bundle] Error 1
  ```
- `make go-run-bun` failed for the same reason (js-bundle dependency).

### What I learned
- The current `--external:exec` syntax is not being honored by bun.

### What was tricky to build
- N/A.

### What warrants a second pair of eyes
- Confirm the correct bun external flag syntax before changing build scripts.

### What should be done in the future
- If bun changes its CLI, verify that the external flag syntax remains valid.

### Code review instructions
- Review `/home/manuel/workspaces/2026-01-10/package-bun-goja-js/go-go-goja/ttmp/2026/01/10/BUN-002--fix-bun-external-flags-for-commonjs-bundling/analysis/01-bun-external-flag-failure-analysis.md` for the error reproduction and fix plan.
- Check `js/package.json` and `Makefile` when the fix lands.

### Technical details
- Commands run:
  - `make js-bundle`
  - `make go-run-bun`
  - `docmgr ticket create-ticket --ticket BUN-002 --title "Fix bun external flags for CommonJS bundling" --topics bun,bundling,build`
  - `docmgr doc add --ticket BUN-002 --doc-type analysis --title "Bun external flag failure analysis"`
  - `docmgr doc add --ticket BUN-002 --doc-type reference --title "Diary"`
  - `docmgr task add --ticket BUN-002 --text "Diagnose bun --external flag behavior and update Makefile/package.json to use correct syntax"`
  - `docmgr task add --ticket BUN-002 --text "Re-run make js-bundle and make go-run-bun to confirm fix"`
  - `docmgr task add --ticket BUN-002 --text "Update analysis + diary and link related files"`
