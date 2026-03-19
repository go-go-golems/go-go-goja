---
Title: Investigation diary
Ticket: GOJA-10-PLUGIN-RESULT-NORMALIZATION
Status: active
Topics:
    - goja
    - go
    - js-bindings
    - tooling
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/hashiplugin/sdk/convert.go
      Note: Main implementation seam changed by GOJA-10
    - Path: pkg/hashiplugin/sdk/sdk_test.go
      Note: Focused tests added for typed slice and map result normalization
    - Path: ttmp/2026/03/18/GOJA-10-PLUGIN-RESULT-NORMALIZATION--normalize-common-go-result-shapes-before-plugin-structpb-encoding/design/01-result-normalization-plan.md
      Note: Compact design note that set the scope for the ticket
ExternalSources: []
Summary: Chronological record of the GOJA-10 plugin result normalization ticket from setup through implementation and closeout.
LastUpdated: 2026-03-18T15:34:38-04:00
WhatFor: Capture the evidence, implementation details, and validation steps for GOJA-10.
WhenToUse: Use when reviewing or extending plugin result normalization behavior.
---

# Diary

## Goal

Record the implementation of GOJA-10, which normalizes common Go plugin result shapes before they are encoded into `structpb.Value`.

## Step 1: Create a narrow ticket around SDK result normalization

The first step was to turn the idea into a tight ticket rather than rolling it into the much broader plugin workstreams. The important decision here was scope discipline: this ticket is about making `sdk.encodeResult(...)` less leaky for common Go values, not about redesigning the transport contract or adding arbitrary serializer magic.

That narrower scope matters because the project already has one proven sharp edge, `[]string` results in the `kv` example, and the right response is to remove that ergonomic footgun without turning the SDK into a reflection-heavy framework. I wrote the plan and tasks around exactly that boundary.

### Prompt Context

**User prompt (verbatim):** "yes, let's do the third, create a separate ticket for it. Add small implementation plan and tasks, then work on it as usual"

**Assistant interpretation:** Create a new ticket for plugin result normalization, document a compact implementation plan, then execute the change in reviewable slices with diary updates and validation.

**Inferred user intent:** Fix the specific result-normalization ergonomics gap without letting the work balloon into a larger schema or transport redesign.

**Commit (code):** N/A yet for setup; the matching ticket-docs commit is recorded separately in git history.

### What I did

- Created GOJA-10 with `docmgr ticket create-ticket`.
- Added a compact design note describing the problem, scope, pseudocode, and validation plan.
- Replaced the placeholder task list with explicit setup, implementation, validation, and closeout phases.
- Seeded this diary.

### Why

- The result-normalization work is small but worth tracking separately from GOJA-08 and GOJA-09.
- A written scope boundary makes it easier to resist pulling in unrelated serialization ideas.

### What worked

- The new ticket skeleton was enough to capture the implementation plan without requiring a full research-style design doc.

### What didn't work

- Nothing failed in this setup step.

### What I learned

- This issue is a good example of a small but real SDK ergonomics ticket: one user-visible problem, one narrow implementation seam, one clear validation strategy.

### What was tricky to build

- The main subtlety was choosing where to stop. It is easy to slide from “normalize common result shapes” into “general arbitrary Go serialization,” which would be a different project.

### What warrants a second pair of eyes

- Whether the ticket should stay explicitly limited to results, or whether arguments should eventually gain the same normalization treatment for symmetry.

### What should be done in the future

- Implement the normalizer in `pkg/hashiplugin/sdk/convert.go`.
- Add tests before broadening scope further.

### Code review instructions

- Read the plan in `design/01-result-normalization-plan.md`.
- Then inspect `pkg/hashiplugin/sdk/convert.go` and `pkg/hashiplugin/sdk/sdk_test.go` to see the current seam this ticket will change.

### Technical details

- Commands run:
  - `docmgr ticket create-ticket --ticket GOJA-10-PLUGIN-RESULT-NORMALIZATION --title "Normalize common Go result shapes before plugin structpb encoding" --topics goja,plugins,hashicorp,protobuf,go`

## Step 2: Normalize common result shapes before `structpb` conversion

The implementation step stayed small, but the code still needed one important design choice: whether to write a narrow ad hoc converter for a few hand-picked slice/map types or make the normalizer recursive enough to handle nested typed collections in one pass. I chose the second option, but still kept the behavior opinionated and bounded. The new logic normalizes common scalar, slice, array, pointer, interface, and `map[string]T` shapes into `structpb`-friendly `[]any` and `map[string]any` trees, and it still fails fast on unsupported types instead of trying to serialize everything in sight.

That kept the ticket aligned with its original scope. The SDK is still not a general serializer; it just no longer forces plugin authors to manually widen values like `[]string` or nested `map[string][]int` by hand before returning them.

### Prompt Context

**User prompt (verbatim):** "yes, let's do the third, create a separate ticket for it. Add small implementation plan and tasks, then work on it as usual"

**Assistant interpretation:** After ticket setup, implement the actual result-normalization change, test it, and record the work in reviewable slices.

**Inferred user intent:** Remove the concrete `structpb` shape leak from the SDK while preserving the existing transport model and keeping the fix operationally simple.

**Commit (code):** `11db436` - `hashiplugin: normalize common plugin result shapes`

### What I did

- Updated `pkg/hashiplugin/sdk/convert.go` so `encodeResult(...)` first normalizes returned Go values before calling `structpb.NewValue(...)`.
- Preserved `*structpb.Value` as an escape hatch for callers that want exact protobuf control.
- Added recursive normalization for typed slices, arrays, pointers, interfaces, and `map[string]T` maps using bounded reflection.
- Kept clear errors for unsupported shapes such as non-string map keys and function values.
- Added focused tests in `pkg/hashiplugin/sdk/sdk_test.go` for typed slices, typed maps, nested values, and unsupported shapes.

### Why

- The current API was forcing plugin authors to think in protobuf container shapes even when returning ordinary Go values.
- The specific bug already surfaced in the example catalog, where `[]string` had to be widened manually to `[]any`.
- A recursive normalizer fixes the real ergonomics problem without changing the host/plugin contract.

### What worked

- The reflection-backed normalizer handled nested typed slices and `map[string]T` values cleanly.
- The existing `encodeResult(...)` seam was exactly the right insertion point; no host-side changes were needed.
- The tests covered the main ergonomic cases and the expected rejection behavior.

### What didn't work

- The first implementation failed the repository lint/test path because the `reflect.Kind` switch in `normalizeResultValue(...)` was not exhaustive enough for the configured linter.
- After fixing the missing switch branches, the file still needed an explicit trailing return to satisfy the compiler's control-flow analysis.

### What I learned

- This repo's lint configuration is strict enough that reflection helpers need explicit grouped cases for unsupported kinds, not just a generic `default`.
- The bounded-reflection approach is a good middle ground here: it fixes nested typed collections without committing the SDK to arbitrary struct serialization.

### What was tricky to build

- The tricky part was avoiding both extremes:
  - too narrow, which would just re-create the same problem for another typed slice later
  - too broad, which would accidentally turn the SDK into a reflection-driven serializer with implicit policies
- Keeping `map[string]T` support while rejecting non-string map keys was an important policy boundary.

### What warrants a second pair of eyes

- Whether the current normalization boundary is the right one for v1, especially the choice to reject structs rather than JSON-round-tripping them.
- Whether argument normalization should ever mirror result normalization, or whether results should stay the only convenience layer.

### What should be done in the future

- Expose this behavior clearly in the SDK docs so plugin authors know common typed slices and maps are now accepted.
- Decide later whether there should be a separate opt-in path for struct serialization or richer schema-based results.

### Code review instructions

- Read `pkg/hashiplugin/sdk/convert.go` first to understand the recursive normalizer and the `encodeResult(...)` entrypoint.
- Then read the new tests in `pkg/hashiplugin/sdk/sdk_test.go`.
- Compare the behavior against `plugins/examples/kv/main.go` to see the ergonomic issue this change removes.

### Technical details

- Commands run:
  - `gofmt -w pkg/hashiplugin/sdk/convert.go pkg/hashiplugin/sdk/sdk_test.go`
  - `go test ./pkg/hashiplugin/sdk -count=1`
  - `go test ./pkg/hashiplugin/sdk ./pkg/hashiplugin/host ./pkg/hashiplugin/shared -count=1`
  - `go test ./... -count=1`
- Notable failure during implementation:
  - `go test ./pkg/hashiplugin/sdk -count=1`
  - Failure summary: the `exhaustive` linter reported missing `reflect.Kind` cases in `pkg/hashiplugin/sdk/convert.go`, and the compiler later required an explicit trailing `return` after the switch.
- Final touched code files:
  - `pkg/hashiplugin/sdk/convert.go`
  - `pkg/hashiplugin/sdk/sdk_test.go`

## Step 3: Repair ticket metadata, validate, and publish the bundle

The code change was already done, but the closeout still needed the ticket to satisfy the repo's `docmgr` expectations. The first `docmgr doctor` run caught a real documentation hygiene problem: the GOJA-10 design note and diary had been created as plain markdown without YAML frontmatter, and the generated ticket index still carried topic values outside the configured vocabulary. I fixed those metadata issues rather than ignoring them, reran `doctor`, and then uploaded the final bundle to reMarkable.

This is the normal closeout pattern worth preserving for later tickets: code first, then make the ticket actually reflect the code and pass the local doc checks before publishing.

### Prompt Context

**User prompt (verbatim):** "yes, let's do the third, create a separate ticket for it. Add small implementation plan and tasks, then work on it as usual"

**Assistant interpretation:** Finish GOJA-10 end-to-end, including the usual task/diary/changelog updates, validation, and reMarkable upload.

**Inferred user intent:** Treat this as a normal tracked ticket, not just a code-only patch.

**Commit (docs):** `e556f05` - `docs: record GOJA-10 normalization slice`

### What I did

- Added proper docmgr frontmatter to `design/01-result-normalization-plan.md` and `reference/01-diary.md`.
- Trimmed the generated `index.md` topics to the repository vocabulary.
- Ran `docmgr doctor --ticket GOJA-10-PLUGIN-RESULT-NORMALIZATION --stale-after 30` until it passed.
- Dry-ran the bundle upload, then uploaded the final GOJA-10 PDF to reMarkable.
- Verified the remote listing under `/ai/2026/03/18/GOJA-10-PLUGIN-RESULT-NORMALIZATION`.

### Why

- Ticket docs are part of the deliverable in this repository, not an optional side artifact.
- The doctor failures were legitimate and would have left the ticket in a lower-quality state than the code itself.

### What worked

- `docmgr doctor` gave actionable file-level failures.
- The bundle upload path worked cleanly once the ticket metadata was fixed.

### What didn't work

- The first `docmgr doctor` run did not pass because two docs lacked frontmatter and the ticket index topics were too broad for the configured vocabulary.

### What I learned

- Even short ticket docs should start with valid frontmatter if they are going to live in `ttmp/...` and participate in docmgr validation.
- The generated `index.md` can still need hand cleanup when the default topics do not match the repository vocabulary.

### What was tricky to build

- The only subtle part was deciding whether to broaden the vocabulary or narrow the ticket topics. For this ticket, narrowing the topics was the right choice because the implementation is fundamentally about `goja`, Go SDK ergonomics, and tooling rather than a separate taxonomy expansion.

### What warrants a second pair of eyes

- Nothing significant remains in the ticket mechanics; the remaining review focus should stay on the actual normalization behavior in the code.

### What should be done in the future

- If more tickets keep wanting topic values like `plugins` or `protobuf`, it may be worth expanding the repo vocabulary instead of trimming them manually each time.

### Code review instructions

- Confirm `docmgr doctor` is clean for GOJA-10.
- Open the ticket workspace and verify the diary, tasks, and changelog match commits `e5fed78`, `11db436`, and `e556f05`.
- Optionally inspect the uploaded reMarkable folder listing for `GOJA-10 Plugin result normalization`.

### Technical details

- Commands run:
  - `docmgr doctor --ticket GOJA-10-PLUGIN-RESULT-NORMALIZATION --stale-after 30`
  - `remarquee status`
  - `remarquee cloud account --non-interactive`
  - `remarquee upload bundle ttmp/2026/03/18/GOJA-10-PLUGIN-RESULT-NORMALIZATION--normalize-common-go-result-shapes-before-plugin-structpb-encoding --dry-run --name "GOJA-10 Plugin result normalization" --remote-dir "/ai/2026/03/18/GOJA-10-PLUGIN-RESULT-NORMALIZATION" --toc-depth 2`
  - `remarquee upload bundle ttmp/2026/03/18/GOJA-10-PLUGIN-RESULT-NORMALIZATION--normalize-common-go-result-shapes-before-plugin-structpb-encoding --force --name "GOJA-10 Plugin result normalization" --remote-dir "/ai/2026/03/18/GOJA-10-PLUGIN-RESULT-NORMALIZATION" --toc-depth 2`
  - `remarquee cloud ls /ai/2026/03/18/GOJA-10-PLUGIN-RESULT-NORMALIZATION --long --non-interactive`
- Final validation result:
  - `docmgr doctor`: all checks passed
  - reMarkable listing: `[f] GOJA-10 Plugin result normalization`
