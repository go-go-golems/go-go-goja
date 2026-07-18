---
Title: Investigation diary
Ticket: XGOJA-ARTIFACT-SELECTION-2026-07-18
Status: active
Topics:
    - xgoja
    - backend
    - testing
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ws://go-go-goja/cmd/xgoja/root_test.go
      Note: Existing command test patterns and future regression test location
    - Path: ws://go-go-goja/ttmp/2026/07/18/XGOJA-ARTIFACT-SELECTION-2026-07-18--pragmatic-command-compatible-artifact-selection-for-xgoja-build-and-generate/scripts/01-reproduce-artifact-order.log
      Note: Captured baseline failures and inconsistent runtime target metadata
    - Path: ws://go-go-goja/ttmp/2026/07/18/XGOJA-ARTIFACT-SELECTION-2026-07-18--pragmatic-command-compatible-artifact-selection-for-xgoja-build-and-generate/scripts/01-reproduce-artifact-order.sh
      Note: Self-contained artifact-order and support-first reproduction
ExternalSources: []
Summary: Chronological evidence and design record for the pragmatic xgoja command-compatible artifact-selection ticket.
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: Preserve reproduction commands, observed failures, reasoning, and implementation review instructions.
WhenToUse: Read before implementing or reviewing XGOJA-ARTIFACT-SELECTION-2026-07-18.
---


# Diary

## Goal

Investigate and design the smallest production-quality fix that lets one xgoja/v2 specification contain a binary, a runtime package, and support artifacts while `xgoja build` and `xgoja generate` select the correct output independently of YAML order.

## Step 1: Reproduce artifact-order failures and write the intern implementation guide

I created the requested docmgr ticket in the isolated `task/improve-xgoja` workspace, mapped the CLI-to-planner-to-generator path, reproduced the order failures against commit `69b69b6`, and wrote the implementation guide. The design deliberately avoids a public artifact-selection feature or dependency framework.

The investigation found one additional concrete defect that changes the minimum safe fix: command selection and generated runtime metadata can disagree even when `build` succeeds. Therefore the design uses a shallow scoped plan containing the selected primary plus global support artifacts, rather than changing only the command's local output selection.

### Prompt Context

**User prompt (verbatim):** "Create a new docmgr ticket in /home/manuel/workspaces/2026-07-18/improve-xgoja/go-go-goja/ttmp (use docmgr --root ...) and Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Create a standalone xgoja ticket and an evidence-backed, intern-oriented design package for the pragmatic artifact-selection fix, validate it with docmgr, and deliver it as a reMarkable bundle.

**Inferred user intent:** Hand implementation to a new engineer without requiring them to rediscover xgoja's planner/generator architecture, while preserving the agreed boundary against feature and architecture astronauting.

### What I did

- Created ticket `XGOJA-ARTIFACT-SELECTION-2026-07-18` with `docmgr --root /home/manuel/workspaces/2026-07-18/improve-xgoja/go-go-goja/ttmp`.
- Added a design doc and investigation diary.
- Read repository and parent `AGENT.md` guidance.
- Inspected:
  - `cmd/xgoja/root.go`
  - `cmd/xgoja/v2_bridge.go`
  - `cmd/xgoja/v2_plan_helpers.go`
  - `cmd/xgoja/cmd_build.go`
  - `cmd/xgoja/cmd_generate.go`
  - `cmd/xgoja/internal/specv2/types.go`
  - `cmd/xgoja/internal/specv2/validate.go`
  - `cmd/xgoja/internal/plan/plan.go`
  - `cmd/xgoja/internal/generate/plan.go`
  - `cmd/xgoja/internal/generate/templates.go`
  - existing root/generator tests and v2 documentation.
- Ran the focused baseline:

  ```bash
  go test ./cmd/xgoja/... -count=1
  ```

  All focused packages passed.
- Created and ran:

  ```bash
  ttmp/2026/07/18/XGOJA-ARTIFACT-SELECTION-2026-07-18--pragmatic-command-compatible-artifact-selection-for-xgoja-build-and-generate/scripts/01-reproduce-artifact-order.sh
  ```

- Captured output in `scripts/01-reproduce-artifact-order.log`.
- Wrote the detailed design and implementation guide with architecture diagrams, API sketches, pseudocode, decision records, phased implementation, tests, risks, and file references.

### Why

- The prior recommendation to add only command-compatible scanning was incomplete: generator target and source logic independently reads the original artifact list.
- An intern needs to understand both `Plan.Artifacts` and `Plan.Config.Artifacts`; changing only one produces internally inconsistent behavior.
- Reproduction and line-anchored code evidence prevent the design from drifting into speculative architecture.

### What worked

- The focused baseline passed:

  ```text
  ok github.com/go-go-golems/go-go-goja/cmd/xgoja
  ok github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/generate
  ok github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/plan
  ok github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/specv2
  ```

- The reproduction cleanly demonstrated all four binary/package order outcomes.
- A support-first case demonstrated the second selector mismatch:

  ```text
  === support artifact first: build command selects binary ===
  xgoja dry run ok: ... target=xgoja ...

  === support artifact first: generated runtime metadata still uses support artifact ===
    "target": {
      "kind": "embedded-assets",
      "output": ".../binary"
    },
  ```

- Existing generator APIs can consume a shallow scoped plan; no public generator API redesign is needed.
- Current examples contain binary/support combinations and a runtime-package-only example, but no checked-in binary+runtime-package example. This limits compatibility risk.

### What didn't work

- An exploratory shell loop used `rg -c` under command substitution and assumed it always printed `0`. For files without a match, the variable was empty, producing repeated errors:

  ```text
  /bin/bash: line 39: [: : integer expression expected
  ```

  The useful matched output still appeared, but the command was not robust. The later search used `rg -c ... || true` and did not rely on the first loop for conclusions.
- The first version of the reproduction script calculated the repository root with one too many `..` components. I corrected:

  ```text
  ../../../../../../..  ->  ../../../../../..
  ```

  before running it. No evidence was recorded from the incorrect version.
- Current `generate` behavior fails for binary-first order exactly as expected:

  ```text
  Error: xgoja generate supports target.kind package, source, or template; got "xgoja"
  exit status 1
  ```

- Current `build` behavior fails for runtime-package-first order exactly as expected:

  ```text
  Error: target.kind package is source generation only; use xgoja generate -f ...
  exit status 1
  ```

### What I learned

- `targetFromPlan` uses `Plan.Artifacts`, while generator rendering uses `Plan.Config.Artifacts`. They are parallel representations populated by `plan.Compile`.
- `targetFromPlan` skips `dts` and `embedded-assets`; `targetDataFromPlanArtifacts` does not. This is the cause of support-first target metadata corruption.
- Generator embedding unions JS/help sources across all primary artifact types and assets across all `embedded-assets` artifacts.
- A shallow plan copy that filters unselected primaries and retains support artifacts is small enough for the pragmatic scope and directly fixes both metadata and source-selection inconsistencies.
- `adapter` and `cobra` belong to build-compatible primaries because `main.go.tmpl` contains dedicated host-integration branches for them. `runtime-package`, `source`, and `template` belong to generate.
- A no-ambiguity rule gives deterministic behavior without prematurely adding `--artifact`.

### What was tricky to build

- **Keeping the plan internally synchronized:** `Plan.Config` is a value, but it contains slices. A safe shallow scope operation must copy `Plan`, copy `Config`, allocate new artifact slices, and update both config-level and compiled artifact slices. Mutating or reusing the original slice backing arrays could alter the caller's plan.
- **Drawing the pragmatic boundary:** reordering only the selected primary is simpler but still leaves source leakage from unselected primaries. Filtering to selected primary plus global support artifacts is only slightly more code and closes an observed class of mismatch. Adding target-specific support dependencies would cross into unsupported architecture.
- **Separating command compatibility from internal target kind:** user errors should name YAML artifact types and IDs, while existing rendering still uses mapped kinds (`binary` to `xgoja`, `runtime-package` to `package`).

### What warrants a second pair of eyes

- Confirm that `adapter` and `cobra` should remain build-compatible and are not intended for `generate` despite the broad wording in the artifact documentation.
- Review slice copying carefully to ensure the original plan and config are not mutated.
- Confirm desired behavior for a v2 spec with no primary artifacts. The design prefers a clear no-compatible-primary error rather than the current implicit binary fallback.
- Verify that retaining every global `dts` and `embedded-assets` artifact is the intended support behavior for this phase.

### What should be done in the future

- Add `--artifact` only when a real consumer needs multiple compatible primary artifacts in one spec.
- Model target-specific support artifacts only when global assets become insufficient.
- Consider consolidating command and generator target derivation in a larger refactor only if further divergence appears; it is not needed for this ticket.

### Code review instructions

- Start with the design doc's sections 4-6 for system architecture and the proposed helper contract.
- Review implementation in this order:
  1. `cmd/xgoja/v2_plan_helpers.go`
  2. `cmd/xgoja/v2_plan_helpers_test.go`
  3. `cmd/xgoja/cmd_build.go`
  4. `cmd/xgoja/cmd_generate.go`
  5. `cmd/xgoja/root_test.go`
  6. `cmd/xgoja/doc/17-xgoja-v2-reference.md`
- Validate with:

  ```bash
  go test ./cmd/xgoja/... -count=1
  go test ./... -count=1
  go vet ./...
  ```

- Rerun the ticket reproduction and verify both commands succeed in either binary/package order, while ambiguity/no-match cases fail clearly.

### Technical details

- Repository: `/home/manuel/workspaces/2026-07-18/improve-xgoja/go-go-goja`
- Branch: `task/improve-xgoja`
- Baseline commit: `69b69b6`
- Ticket root: `/home/manuel/workspaces/2026-07-18/improve-xgoja/go-go-goja/ttmp`
- Ticket: `XGOJA-ARTIFACT-SELECTION-2026-07-18`
- Current helper: `cmd/xgoja/v2_plan_helpers.go:16-35`
- Build call site: `cmd/xgoja/cmd_build.go:86-125`
- Generate call site: `cmd/xgoja/cmd_generate.go:88-159`
- Generator target derivation: `cmd/xgoja/internal/generate/templates.go:287-300`
- Runtime metadata target: `cmd/xgoja/internal/generate/templates.go:343-395`
- Source unions: `cmd/xgoja/internal/generate/plan.go:199-227`

## Step 2: Validate and deliver the design bundle

I validated the ticket metadata and relationships, ran docmgr doctor successfully, previewed the reMarkable bundle, and uploaded the overview, intern guide, and diary as one PDF with a depth-two table of contents.

The upload command reported success directly, so no additional cloud listing was needed.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Complete ticket bookkeeping and deliver the validated guide to the requested reMarkable destination.

**Inferred user intent:** Make the implementation guide immediately available for offline review and intern handoff.

### What I did

- Validated frontmatter on the design and diary documents.
- Ran:

  ```bash
  docmgr --root /home/manuel/workspaces/2026-07-18/improve-xgoja/go-go-goja/ttmp \
    doctor --ticket XGOJA-ARTIFACT-SELECTION-2026-07-18 --stale-after 30
  ```

- Ran a `remarquee upload bundle --dry-run` with the ticket index, design doc, and diary.
- Uploaded the same bundle as `XGOJA Artifact Selection Intern Guide.pdf`.

### Why

- The dry run catches document ordering, destination, and rendering-command errors before upload.
- A single PDF with a table of contents is easier to review than separate documents.

### What worked

- Both frontmatter validations passed.
- `docmgr doctor` reported `All checks passed`.
- Dry run selected the expected three Markdown documents and destination.
- Actual upload returned:

  ```text
  OK: uploaded XGOJA Artifact Selection Intern Guide.pdf -> /ai/2026/07/18/XGOJA-ARTIFACT-SELECTION-2026-07-18
  ```

### What didn't work

- N/A.

### What I learned

- The 1,124-line design document and supporting diary render successfully as one remarquee bundle without changes to Markdown diagrams or code fences.

### What was tricky to build

- The bundle needed enough context for a new intern without including tasks/changelog noise. The selected order is overview, implementation guide, then investigation evidence.

### What warrants a second pair of eyes

- Review the scoped-plan non-mutation pseudocode and the explicit decision that a spec with no compatible primary should error.

### What should be done in the future

- Upload a revised bundle only after implementation changes materially alter the accepted design; avoid overwriting annotated copies casually.

### Code review instructions

- Open the reMarkable bundle and begin with the design's executive summary, then sections 4-6 and 9-10.
- Use the repository ticket as the authoritative mutable copy.

### Technical details

- Bundle: `XGOJA Artifact Selection Intern Guide.pdf`
- Destination: `/ai/2026/07/18/XGOJA-ARTIFACT-SELECTION-2026-07-18`
- ToC depth: 2
- Documents: `index.md`, design doc, investigation diary.
