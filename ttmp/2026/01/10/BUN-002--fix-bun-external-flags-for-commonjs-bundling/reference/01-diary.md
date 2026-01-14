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
    - Path: Makefile
      Note: Bun target adjusted in Step 2
    - Path: cmd/bun-demo/assets/bundle.cjs
      Note: Updated bundle output from Step 2
    - Path: js/package.json
      Note: Bun script adjusted in Step 2
    - Path: ttmp/2026/01/10/BUN-002--fix-bun-external-flags-for-commonjs-bundling/analysis/01-bun-external-flag-failure-analysis.md
      Note: Analysis referenced in Step 1
ExternalSources: []
Summary: Implementation diary for fixing bun external flags.
LastUpdated: 2026-01-10T20:37:31-05:00
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

## Step 2: Fix bun externals and target for CommonJS bundling

I corrected the bun build flags to use the proper `--external=` syntax and switched the target to `node` so bun preserves `require("fs")` instead of stubbing it for browsers. After updating the build scripts, the bundle build succeeded and the demo ran correctly through the embedded CommonJS entrypoint.

This step also captured the intermediate failure where `fs` was stubbed under the browser target, which explained why `writeFileSync` was missing at runtime.

**Commit (code):** 52d88d5 — "Build: fix bun externals for goja demo"

### What I did
- Updated `Makefile` and `js/package.json` to use `--external=...` flags and `--target=node`.
- Rebuilt the bundle and copied it into `cmd/bun-demo/assets`.
- Re-ran `make js-bundle` and `make go-run-bun` to validate the fix.
- Checked off tasks 1-3.

### Why
- Bun ignored the esbuild-style external syntax.
- Browser targeting stubs `fs`, which breaks the native module API surface.

### What worked
- `make js-bundle` completed and produced a bundle with `require("fs")`.
- `make go-run-bun` printed a value (`2026-01-10:5`).

### What didn't work
- Before switching to `--target=node`, `make go-run-bun` failed with:
  ```
  run(): TypeError: Object has no member 'writeFileSync' at run (assets/bundle.cjs:5720:19(34))
  ```

### What I learned
- Bun stubs Node-style builtins under the browser target even when externalized.

### What was tricky to build
- Ensuring the generated bundle is kept in sync with the embedded asset file.

### What warrants a second pair of eyes
- Confirm `--target=node` is acceptable for the broader Goja runtime expectations.

### What should be done in the future
- Consider documenting when to use `--target=node` vs `--target=browser` in the main design doc.

### Code review instructions
- Review `/home/manuel/workspaces/2026-01-10/package-bun-goja-js/go-go-goja/Makefile`, `/home/manuel/workspaces/2026-01-10/package-bun-goja-js/go-go-goja/js/package.json`, and `/home/manuel/workspaces/2026-01-10/package-bun-goja-js/go-go-goja/cmd/bun-demo/assets/bundle.cjs`.

### Technical details
- Commands run:
  - `cd js && bun build --target=node --format=cjs --outfile=dist/bundle.cjs src/main.js --external=exec --external=database --external=fs`
  - `make js-bundle`
  - `make go-run-bun`
  - `docmgr task check --ticket BUN-002 --id 1,2,3`

## Step 3: Close the ticket

I closed BUN-002 after confirming the CommonJS demo runs with the updated bundling flags. The ticket is now in review status to reflect completion.

This keeps the docmgr metadata aligned with the fix status.

### What I did
- Closed the ticket with status `review`.

### Why
- All tasks were completed and validation succeeded.

### What worked
- docmgr updated the ticket metadata and changelog.

### What didn't work
- N/A.

### What I learned
- N/A.

### What was tricky to build
- N/A.

### What warrants a second pair of eyes
- Confirm the review status and changelog entry match expectations.

### What should be done in the future
- N/A.

### Code review instructions
- Review `/home/manuel/workspaces/2026-01-10/package-bun-goja-js/go-go-goja/ttmp/2026/01/10/BUN-002--fix-bun-external-flags-for-commonjs-bundling/index.md` and `/home/manuel/workspaces/2026-01-10/package-bun-goja-js/go-go-goja/ttmp/2026/01/10/BUN-002--fix-bun-external-flags-for-commonjs-bundling/changelog.md`.

### Technical details
- Commands run:
  - `docmgr ticket close --ticket BUN-002 --status review --changelog-entry "Closed after fixing bun external flags and validating demo"`
