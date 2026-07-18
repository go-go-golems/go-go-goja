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
    - Path: ws://go-go-goja/cmd/xgoja/internal/specv2/defaults.go
      Note: Central ArtifactSpec.Type normalization before default output derivation
    - Path: ws://go-go-goja/cmd/xgoja/root_test.go
      Note: |-
        Existing command test patterns and future regression test location
        End-to-end order, target metadata, source-isolation, and shared-asset regressions
    - Path: ws://go-go-goja/cmd/xgoja/v2_plan_helpers_test.go
      Note: Unit coverage for compatibility, ambiguity/no-match diagnostics, scoping, and non-mutation
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

## Step 3: Implement command-compatible selection and scoped generation

I replaced the order-dependent first-primary helper with command-aware selection, then passed a scoped plan to both build and generate. The implementation selects exactly one primary compatible with the invoked command, preserves support artifacts, and makes the command's selected target consistent with generator metadata and embedded sources.

The change was committed in two focused code commits: `7caaee6` contains the selector, plan scoping, caller integration, and helper tests; `4003433` contains command-level regression tests and public documentation. No new public selection flag or artifact dependency model was added.

### Prompt Context

**User prompt (verbatim):** "Implement"

**Assistant interpretation:** Implement the accepted pragmatic xgoja artifact-selection design in the local `task/improve-xgoja` workspace.

**Inferred user intent:** Replace the duplicate-spec workaround with deterministic, well-tested build/generate behavior without feature astronauting.

### What I did

- Replaced `targetFromPlan` in `cmd/xgoja/v2_plan_helpers.go` with:
  - `selectPlanTarget(compiled, command)`;
  - explicit build/generate compatibility predicates;
  - no-compatible and ambiguous-primary errors naming YAML artifact IDs/types;
  - `scopePlanToPrimary`, which shallow-copies `plan.Plan` and its config, puts the selected primary first, retains `dts`/`embedded-assets`, and removes unselected primaries.
- Updated `cmd/xgoja/cmd_build.go` and `cmd/xgoja/cmd_generate.go` to rebind their plan to the scoped copy before every generator path, including `--template-data`.
- Added helper tests in `cmd/xgoja/v2_plan_helpers_test.go` for selection, support retention, error diagnostics, and non-mutation of the original plan.
- Added command regressions in `cmd/xgoja/root_test.go` for binary/package in both YAML orders and for JS/help source isolation with shared assets.
- Updated `cmd/xgoja/doc/17-xgoja-v2-reference.md` with the command/artifact matrix and ambiguity behavior.

### Why

- Command-local target selection alone would leave `internal/generate` deriving target metadata and embedded sources from the full artifact list.
- Scoped plans reuse existing generator APIs while guaranteeing one selected primary across command output, runtime metadata, and copied source roots.

### What worked

- `go test ./cmd/xgoja -run 'Test(BuildAndGenerate|SelectPlanTarget)' -count=1` passed.
- `go test ./cmd/xgoja/... -count=1` passed.
- `go test ./... -count=1` passed.
- `go vet ./...` passed.
- Both commits' pre-commit hooks passed the repository's `go generate`, full test, golangci-lint, and Glazed vet checks.
- Regression tests prove that binary/package order no longer matters; build receives `kind: xgoja`, generate receives `kind: package`, each receives only its selected JS source, and both receive global assets.

### What didn't work

- The first focused test after deleting the obsolete helper failed because the two command callers still referenced it:

  ```text
  cmd/xgoja/cmd_build.go:90:12: undefined: targetFromPlan
  cmd/xgoja/cmd_generate.go:92:12: undefined: targetFromPlan
  ```

  Fix: wire both callers to `selectPlanTarget` in the planned core integration.
- An initial command-test helper had an unused local `dir`; removed it.
- The first follow-up commit attempt failed lint with:

  ```text
  cmd/xgoja/root_test.go:659:1: named return "specPath" with type "string" found (nonamedreturns)
  ```

  Removing named returns initially left local path variables undeclared; the resumed fix declared them explicitly. Focused tests and `./.bin/golangci-lint run ./cmd/xgoja/...` then passed before retrying the commit.

### What I learned

- A shallow scoped plan is enough because generator functions already consume `plan.Plan`; changing their public signatures was unnecessary.
- Updating both `Plan.Config.Artifacts` and `Plan.Artifacts` is mandatory: the generator reads the former while selection reads the latter.
- Keeping support artifacts after the selected primary preserves existing global asset behavior while preventing selected-source leakage.

### What was tricky to build

- The generator has separate target and source paths, so a fix that only changed `cmd_build.go`/`cmd_generate.go` would have produced correct command output with incorrect generated metadata. Scoping the artifact lists before calling generators aligned those independent paths without introducing a new architecture.

### What warrants a second pair of eyes

- Confirm the intentional no-compatible-primary behavior for v2 specs that declare only support artifacts.
- Confirm `adapter` and `cobra` should remain build-compatible primary types; their existing generated-main branches support this classification.

### What should be done in the future

- Add `--artifact` only when a real configuration needs multiple compatible primaries; current ambiguity errors deliberately make that need visible.
- Do not add target-specific asset dependency modeling until a real consumer needs non-global support assets.

### Code review instructions

- Start at `cmd/xgoja/v2_plan_helpers.go` and verify compatibility, errors, and non-mutating plan scope.
- Verify `cmd_build.go` and `cmd_generate.go` pass the scoped plan through all generator branches.
- Read `v2_plan_helpers_test.go` and the two new root tests for policy and end-to-end regressions.
- Verify documentation matches the test contract in `cmd/xgoja/doc/17-xgoja-v2-reference.md`.

### Technical details

- Code commits: `7caaee6fb803a69785d00b8c36918c0b9d9cdfb0`, `400343331a188ed372d647ecb959d659134dc3d2`.
- Selected build primary types: `binary`, `adapter`, `cobra`.
- Selected generate primary types: `runtime-package`, `source`, `template`.
- Retained support types: `dts`, `embedded-assets`.

## Step 4: Address review — normalize whitespace-padded artifact types

A review correctly identified that `specv2.validateArtifacts` accepts whitespace around `artifact.type`, while the new compatibility checks initially compared raw strings. That would reject an accepted `type: " binary "` configuration and regress the previous selector's trimming behavior.

I normalized artifact types at the selection boundary and in the scoped copies sent to generators. The latter matters because generator code compares artifact types directly; merely trimming candidate checks would select a binary but still leave downstream rendering with a whitespace-padded type.

### Prompt Context

**User prompt (verbatim):** "Address code review
cmd/xgoja/v2_plan_helpers.go
Comment on lines +63 to +68
func isCompatiblePrimary(command artifactCommand, artifactType string) bool {
    switch command {
    case artifactCommandBuild:
        return artifactType == \"binary\" || artifactType == \"adapter\" || artifactType == \"cobra\"
    case artifactCommandGenerate:
        return artifactType == \"runtime-package\" || artifactType == \"source\" || artifactType == \"template\"
@chatgpt-codex-connector
chatgpt-codex-connector Bot
3 minutes ago

P2 Badge Normalize artifact types before compatibility checks

When an artifact type contains surrounding whitespace (for example, YAML type: \" binary \"), specv2.validateArtifacts accepts it because validation switches on strings.TrimSpace(artifact.Type), but this exact comparison finds no build candidate and rejects the otherwise valid plan. The previous targetFromPlan also trimmed the type before selecting it, so this introduces a regression for accepted specs; trim or normalize artifact types before compatibility and support classification.

Useful? React with 👍 / 👎."

**Assistant interpretation:** Correct the artifact-type whitespace regression identified in code review and add regression coverage, without widening the feature scope.

**Inferred user intent:** Preserve compatibility with accepted v2 YAML while keeping selected plans safe for existing generator code.

### What I did

- Added `normalizedArtifactType`, used by primary compatibility and support-artifact classification.
- Normalized `ArtifactSpec.Type` and `ArtifactPlan.Spec.Type` only in the shallow scoped copies, preserving the caller's original plan unchanged.
- Normalized target metadata and diagnostic type formatting.
- Added `TestSelectPlanTargetNormalizesArtifactTypesInScopedPlan` for whitespace-padded binary, runtime-package, and embedded-assets types.
- Committed the fix as `30d0d88`.

### Why

- Validation's `strings.TrimSpace` contract should match selection semantics.
- Generator helpers perform direct artifact-type comparisons, so normalized candidates alone would not make the generated runtime plan or embedding behavior correct.

### What worked

- Focused selection test and xgoja golangci-lint passed before commit.
- The pre-commit hook passed repository generation, full tests, lint, and Glazed vet.

### What didn't work

- N/A.

### What I learned

- Normalization belongs at the boundary where validation-normalized semantics meet raw config values. Scoped copies provide a safe normalization point because they are already intended for downstream generation and do not mutate the original compiled plan.

### What was tricky to build

- The review specifically mentioned compatibility checks, but direct comparisons also occur in downstream generator logic. Normalizing scoped config and compiled artifact representations prevents a partial fix where selection succeeds but runtime metadata remains malformed.

### What warrants a second pair of eyes

- Confirm that preserving raw whitespace in the original plan while normalizing only generation-scoped copies is preferred over normalizing all spec fields at load time. This matches the narrow review fix and avoids unrelated schema behavior changes.

### What should be done in the future

- Consider normalizing artifact types centrally in `specv2.ApplyDefaults` only as part of a broader normalization policy with tests for all schema fields; it is not needed for this targeted correction.

### Code review instructions

- Review `normalizedArtifactType`, `normalizedArtifactSpec`, and `normalizedArtifactPlan` in `cmd/xgoja/v2_plan_helpers.go`.
- Confirm the new test proves selection and support retention while asserting original config non-mutation.

### Technical details

- Review-fix commit: `30d0d88a1ac707d8b86808b6c98391417268f0b5`.
- Validation: focused `TestSelectPlanTarget`, `./.bin/golangci-lint run ./cmd/xgoja/...`, and successful full pre-commit test/lint hooks.

## Step 5: Add explicit artifact selection and central artifact-type normalization

The user chose to include two previously deferred but low-risk improvements: explicit `--artifact <id>` selection and normalization of artifact types at v2 defaulting time. The existing scoped-plan implementation made the flag a contained extension: it selects one compatible primary and reuses the same metadata/source-isolation path rather than introducing output orchestration.

The implementation is committed as `a6f83de` and replaces the design guide's prior recommendation to defer the flag.

### Prompt Context

**User prompt (verbatim):** "do 1. and 3."

**Assistant interpretation:** Implement the previously listed `--artifact` flag and central `ArtifactSpec.Type` normalization in `specv2.ApplyDefaults`.

**Inferred user intent:** Improve the practical multi-artifact UX now that the small supporting architecture is in place, without adding dependency graphs or build-all behavior.

### What I did

- Added `--artifact` fields to both build and generate Glazed command settings and help text.
- Extended `selectPlanTarget` with an optional artifact ID:
  - no ID retains automatic exactly-one-compatible selection;
  - an ID selects one compatible primary even if others exist;
  - unknown IDs and incompatible IDs return command-specific errors.
- Updated ambiguity diagnostics to recommend `--artifact <id>`.
- Trimmed every `ArtifactSpec.Type` in `specv2.ApplyDefaults` before binary default-output logic.
- Added unit tests for explicit selection, unknown/incompatible IDs, and central type normalization.
- Updated command integration tests to pass `--artifact binary` and `--artifact runtime`.
- Updated the v2 reference and the intern guide to reflect the new public contract.

### Why

- `--artifact` resolves legitimate multiple-binary or multiple-generation-target specs without changing the scoped-plan model.
- Central normalization aligns defaults, validation, selection, and downstream generator comparisons for all normal v2 load paths.

### What worked

- Focused `go test ./cmd/xgoja/... -count=1` passed.
- `./.bin/golangci-lint run ./cmd/xgoja/...` passed.
- Commit hook passed `go generate ./...`, `go test ./...`, golangci-lint, and Glazed vet.

### What didn't work

- The first help-text revision put Markdown backticks inside Go's raw string literal, causing:

  ```text
  cmd/xgoja/cmd_generate.go:42:22: syntax error: unexpected name runtime in argument list
  ```

  Fix: use plain text in the raw CLI help string; Markdown backticks remain appropriate in Markdown docs.

### What I learned

- The scoped plan converts explicit selection from an architectural feature into a local policy choice: once a primary is selected, existing source/asset handling already follows it.
- Central `ApplyDefaults` normalization makes the later defensive scoped-copy normalization redundant for normal loaded specs, but retaining it protects direct manually constructed plans used by tests/internal callers.

### What was tricky to build

- `--artifact` must not bypass compatibility checks. Allowing a runtime-package ID on `build`, for example, would recreate the original wrong-command failure under a different path. The helper therefore validates the selected artifact against the command's allow-list before scoping.

### What warrants a second pair of eyes

- Review exact artifact-ID matching semantics. The flag trims its input; artifact IDs otherwise retain existing schema behavior. This is intentionally narrower than normalizing all artifact fields.

### What should be done in the future

- Continue to defer dependency graphs, target-specific support assets, and multi-output orchestration; `--artifact` selects one output per invocation.

### Code review instructions

- Start with `selectPlanTarget`'s explicit-ID branch and verify unknown/incompatible errors.
- Verify build/generate decode and pass `settings.Artifact`.
- Verify `specv2.ApplyDefaults` trims type before checking `binary` default output.
- Run the focused tests and inspect documentation examples.

### Technical details

- Code commit: `a6f83de3bc0808ba871462bd88b11b19b0426683`.
- New CLI forms: `xgoja build --artifact <id>` and `xgoja generate --artifact <id>`.
- Non-goals remain: dependency graphs, target-specific assets, and multiple outputs per invocation.

## Step 6: Address review — normalize artifact IDs for `--artifact`

A second review correctly found that `validateArtifacts` trims artifact IDs for validation while the initial `--artifact` implementation trimmed the flag but compared it with the raw stored ID. A valid `id: " release "` therefore could not be selected as `--artifact release` in an ambiguous spec.

I made artifact-ID normalization central for normal v2 loading and defensive for direct plans. This gives the CLI, validation, selection, scoped generation, and diagnostics one identifier representation without expanding the feature surface.

### Prompt Context

**User prompt (verbatim):** "Address
cmd/xgoja/v2_plan_helpers.go
Comment on lines +32 to +36
    artifactID = strings.TrimSpace(artifactID)
    var selected plan.ArtifactPlan
    if artifactID != \"\" {
        var ok bool
        selected, ok = artifactPlanByID(compiled.Artifacts, artifactID)
@chatgpt-codex-connector
chatgpt-codex-connector Bot
now

P2 Badge Normalize artifact IDs before matching --artifact

When a valid spec uses an artifact ID with surrounding whitespace (for example, id: \" release \"), validation accepts it because validateArtifacts trims IDs, but this code trims the flag value and then compares it exactly with the unnormalized stored ID. Consequently --artifact release is reported as unknown—and passing the padded value cannot work because it is trimmed here—so an ambiguous spec containing that artifact cannot be resolved explicitly. Normalize artifact IDs during defaulting or compare their trimmed forms consistently.

Useful? React with 👍 / 👎."

**Assistant interpretation:** Align artifact-ID defaulting and explicit selection with validation's existing whitespace semantics, then prove the flag handles a padded declared ID.

**Inferred user intent:** Remove the remaining accepted-spec regression from `--artifact` without adding unrelated identifier redesign.

### What I did

- Added `cfg.Artifacts[i].ID = strings.TrimSpace(cfg.Artifacts[i].ID)` in `specv2.ApplyDefaults`.
- Added `normalizedArtifactID` and used it when matching IDs, scoping artifacts, rendering diagnostics, and copying artifact specs.
- Added an explicit-selection regression with `ID: " binary "` and `--artifact " binary "`; the selected/scoped ID becomes `binary`, while the manually constructed original plan remains unchanged.
- Extended the existing defaults test to assert normalized ID, type, and binary output.
- Committed code/tests as `2820fad`.

### Why

- The explicit selector must use the same normalized identifier representation as v2 validation.
- Defensive helper normalization keeps direct internal test plans correct even before they pass through `ApplyDefaults`.

### What worked

- Focused `go test ./cmd/xgoja/... -count=1` passed.
- Focused xgoja lint passed.
- The full pre-commit hook passed generation, all tests, lint, and Glazed vet.

### What didn't work

- N/A.

### What I learned

- The artifact-ID and artifact-type fixes share a pattern: central defaulting establishes correct normal behavior, while scoped helper normalization guards manually assembled `plan.Plan` values without mutating them.

### What was tricky to build

- The selected `ArtifactPlan` can originate with a raw padded ID, so scoping must pass the normalized ID into lookup; normalizing lookup comparisons alone is insufficient if the next lookup receives the unnormalized selected ID.

### What warrants a second pair of eyes

- Confirm that normalizing artifact IDs in defaults is consistent with the existing validation behavior and other normalized identifier fields such as provider/source IDs.

### What should be done in the future

- N/A.

### Code review instructions

- Verify `ApplyDefaults` trims artifact IDs before deriving binary defaults.
- Verify explicit selection, config lookup, plan lookup, scoped copies, target metadata, and diagnostics use normalized IDs.
- Run the padded-ID explicit-selection regression.

### Technical details

- Code commit: `2820fadd440b79018266ac038df79cc7d04130f1`.
- Validation: focused xgoja tests/lint plus successful full pre-commit hook.
