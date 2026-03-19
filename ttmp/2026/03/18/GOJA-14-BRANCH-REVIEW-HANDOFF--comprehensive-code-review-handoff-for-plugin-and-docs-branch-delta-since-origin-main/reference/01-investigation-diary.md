---
Title: Investigation diary
Ticket: GOJA-14-BRANCH-REVIEW-HANDOFF
Status: active
Topics:
    - goja
    - analysis
    - architecture
    - review
    - plugins
    - repl
    - documentation
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ttmp/2026/03/18/GOJA-14-BRANCH-REVIEW-HANDOFF--comprehensive-code-review-handoff-for-plugin-and-docs-branch-delta-since-origin-main/design-doc/01-comprehensive-branch-review-handoff-guide-for-code-review-since-origin-main.md
      Note: Primary reviewer handoff guide assembled from the gathered evidence
    - Path: ttmp/2026/03/18/GOJA-14-BRANCH-REVIEW-HANDOFF--comprehensive-code-review-handoff-for-plugin-and-docs-branch-delta-since-origin-main/scripts/review-handoff-context.sh
      Note: Helper script that reruns the branch-surface context commands
ExternalSources: []
Summary: Chronological record of how the review handoff was assembled, which evidence was gathered, and how to validate or continue the ticket.
LastUpdated: 2026-03-18T22:32:00-04:00
WhatFor: Preserve the commands, evidence trail, and continuation notes behind the GOJA-14 review handoff bundle.
WhenToUse: Use when continuing the ticket, checking how claims in the review guide were gathered, or repeating the branch review preparation flow.
---


# Investigation diary

## Goal

Capture how the GOJA-14 branch-review handoff was assembled so a later maintainer can verify the evidence trail, rerun the same branch-surface commands, and understand why the final handoff guide is structured the way it is.

## Step 1: Create the review-handoff ticket and map the branch surface

This step created a dedicated ticket for a reviewer-facing handoff rather than trying to overload the feature tickets. The goal was to package the whole branch delta since `origin/main` into one review document that a tired reviewer could follow in dependency order.

The key decision here was to treat GOJA-14 as a synthesis ticket. The feature tickets already contain deep implementation detail, but not a "how to review all of this carefully" guide. GOJA-14 fills that gap.

### Prompt Context

**User prompt (verbatim):**

```text
I need to do my own code review, i'm pretty tired so I want a document that contains everything needed for the review, from context and architecture docs and source and changes and example before and after and hints and remarks on things to pay more attention to, etc...

Create a detailed analysis / design / implementation guide that is very detailed for me as a tired new deveoper to the project that needs to do a careful review, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references.

It should be very clear and detailed. Store in the ticket and then upload to remarkable.
```

**Assistant interpretation:** Create a new docmgr ticket containing a reviewer handoff document for the full branch delta since `origin/main`, then validate and upload the ticket bundle to reMarkable.

**Inferred user intent:** Reduce review fatigue by turning a large, multi-ticket feature branch into one structured guide with context, architecture, concrete files, review order, and validation hints.

**Commit (code):** pending

### What I did

- Ran `docmgr status --summary-only` to confirm the ticket/docmgr root.
- Checked branch and delta anchors:
  - `git rev-parse --abbrev-ref HEAD`
  - `git merge-base origin/main HEAD`
  - `git diff --stat origin/main...HEAD`
  - `git diff --name-only origin/main...HEAD`
  - `git log --oneline --decorate origin/main..HEAD`
- Created ticket `GOJA-14-BRANCH-REVIEW-HANDOFF`.
- Added:
  - one design doc
  - one diary doc
- Inspected the empty scaffold files to confirm what needed replacing.

### Why

- The branch already has feature tickets, but no reviewer-specific handoff artifact.
- The first step was to create a separate synthesis workspace before writing conclusions.

### What worked

- The branch surface commands gave a clean quantitative frame:
  - `118` changed files
  - `18,161` insertions
  - `562` deletions
- The ticket scaffold was created successfully and matched the expected docmgr layout.

### What didn't work

- N/A

### What I learned

- The right unit of documentation here is not another feature design doc. It is a review guide organized by subsystem and risk.
- The existing GOJA-08 through GOJA-13 docs already contain most of the deep architecture rationale; the missing piece is cross-ticket synthesis and reviewer sequencing.

### What was tricky to build

- The branch is broad enough that a naive summary quickly turns into a changelog. The document needed to stay review-oriented rather than just listing work.
- The best way around that was to anchor the guide around review order and architectural seams, not around ticket numbers or raw commit chronology.

### What warrants a second pair of eyes

- Whether the final guide balances "enough context" against "too much prose".
- Whether the chosen review order really is the most efficient for a tired reviewer.

### What should be done in the future

- If future multi-ticket branches follow this pattern, it would be worth templating a reviewer-handoff ticket format in docmgr.

### Code review instructions

- Start with the GOJA-14 design doc.
- Cross-check the branch-surface commands against the helper script once it exists.

### Technical details

Commands run in this step:

```bash
cd /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja
docmgr status --summary-only
git rev-parse --abbrev-ref HEAD
git merge-base origin/main HEAD
git diff --stat origin/main...HEAD
git diff --name-only origin/main...HEAD
git log --oneline --decorate origin/main..HEAD
docmgr ticket create-ticket --ticket GOJA-14-BRANCH-REVIEW-HANDOFF --title "Comprehensive code review handoff for plugin and docs branch delta since origin main" --topics goja,analysis,architecture,review,plugins,repl,documentation
docmgr doc add --ticket GOJA-14-BRANCH-REVIEW-HANDOFF --doc-type design-doc --title "Comprehensive branch review handoff guide for code review since origin main"
docmgr doc add --ticket GOJA-14-BRANCH-REVIEW-HANDOFF --doc-type reference --title "Investigation diary"
```

## Step 2: Gather file-backed evidence and write the reviewer guide

This step turned the ticket scaffold into a real review handoff. The work here was evidence-first: gather line-anchored references from the engine, plugin host, SDK, docs hub, and evaluator, then shape the document around the real architecture rather than memory of the implementation.

The design guide intentionally emphasizes reading order, review heuristics, and review scenarios. That is the main difference between this ticket and the earlier implementation tickets.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Write the detailed handoff itself, with context, architecture, before/after explanations, examples, hints, and file references for careful review.

**Inferred user intent:** Make the review process low-friction enough that the branch can be reviewed carefully even under fatigue.

**Commit (code):** pending

### What I did

- Read and reused earlier architecture material from:
  - GOJA-08
  - GOJA-11
  - GOJA-12
  - GOJA-13
- Gathered line-anchored evidence from key files with `nl -ba` and `sed -n`.
- Rewrote the design doc to include:
  - executive summary
  - scope and review goal
  - review order
  - architecture diagrams
  - before/after system shape
  - subsystem-specific review guidance
  - end-to-end review scenarios
  - commands and checklist
  - references to code and prior tickets
- Expanded the design doc further so it works without local code access:
  - reviewer glossary
  - commit-trail interpretation
  - detailed offline diff walkthroughs
  - curated before/after excerpts from the engine, plugin host, SDK, docs hub, and evaluator
  - explicit review questions to answer per subsystem
  - a file-by-file "why it exists" index
- Rewrote this diary with explicit chronology and copied commands.
- Added a ticket-local helper script to print the main review context.

### Why

- A tired reviewer should not have to discover the correct reading order alone.
- The guide needed to connect the feature tickets into one branch-level story.

### What worked

- GOJA-13 already contained strong architecture findings and made a good backbone for the handoff.
- The line-anchored reads from `engine`, `pkg/hashiplugin`, and `pkg/docaccess` gave enough evidence to make concrete review recommendations instead of generic advice.

### What didn't work

- N/A

### What I learned

- The branch is easiest to understand as a progression from static/global composition toward runtime-scoped composition.
- The single most important "before vs after" shift is the new registrar phase in the engine.

### What was tricky to build

- The guide needed to be detailed without becoming a second implementation spec for every subsystem.
- The solution was to summarize implementation details only insofar as they help the reviewer decide where to look and what to question.

### What warrants a second pair of eyes

- Whether the hot-spot sections are the right level of suspicion.
- Whether the end-to-end scenarios are sufficient or if another scenario is needed around plugin load failure and diagnostics.

### What should be done in the future

- If new major branch slices are added, GOJA-14 should be refreshed rather than scattering more branch-level review guidance across later tickets.

### Code review instructions

- Start with:
  - the GOJA-14 design doc
  - `engine/factory.go`
  - `pkg/hashiplugin/host/registrar.go`
  - `pkg/docaccess/runtime/registrar.go`
  - `pkg/repl/evaluators/javascript/evaluator.go`
- Validate by running:
  - `go test ./... -count=1`
  - `make install-modules`
  - `go run ./cmd/repl`
  - `go run ./cmd/js-repl`

### Technical details

Representative evidence-gathering commands:

```bash
cd /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja
nl -ba engine/runtime_modules.go | sed -n '1,120p'
nl -ba engine/factory.go | sed -n '1,260p'
nl -ba engine/runtime.go | sed -n '1,160p'
nl -ba engine/module_specs.go | sed -n '1,140p'
nl -ba pkg/hashiplugin/host/registrar.go | sed -n '1,220p'
nl -ba pkg/hashiplugin/host/client.go | sed -n '1,220p'
nl -ba pkg/hashiplugin/host/reify.go | sed -n '1,220p'
nl -ba pkg/hashiplugin/host/report.go | sed -n '1,260p'
nl -ba pkg/hashiplugin/sdk/export.go | sed -n '1,240p'
nl -ba pkg/hashiplugin/sdk/module.go | sed -n '1,240p'
nl -ba pkg/hashiplugin/sdk/convert.go | sed -n '1,240p'
nl -ba pkg/hashiplugin/contract/jsmodule.proto | sed -n '1,220p'
```

## Step 3: Validate the ticket and prepare bundle upload

This step is for hygiene and delivery. After the docs and helper script are in place, the ticket should be related to the main code files, checked by `docmgr doctor`, and uploaded as a reMarkable bundle with a dry-run first.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Finish the ticket as a real deliverable, not just a local markdown draft.

**Inferred user intent:** Make the review handoff easy to consume from both the repo and reMarkable.

**Commit (code):** pending

### What I did

- Related the design doc to the engine, plugin host, SDK, docs registrar, and evaluator files that anchor the review.
- Related this diary to the helper script and primary design doc.
- Added missing topic vocabulary entries:
  - `documentation`
  - `plugins`
  - `review`
- Ran `docmgr doctor --ticket GOJA-14-BRANCH-REVIEW-HANDOFF --stale-after 30` until it passed cleanly.
- Ran a bundle dry-run with `remarquee upload bundle --dry-run ...`.
- Fixed a pandoc/PDF rendering problem by replacing the giant inline verbatim prompt in this diary with a fenced `text` block containing only the substantive user request.
- Re-ran the dry-run, uploaded the final bundle, and verified the remote listing at `/ai/2026/03/18/GOJA-14-BRANCH-REVIEW-HANDOFF`.

### Why

- The user explicitly asked to store the work in the ticket and upload it to reMarkable.

### What worked

- `docmgr doc relate` updated the review guide and diary metadata cleanly.
- `docmgr doctor` passed after adding the missing topic vocabulary and keeping the ticket files internally consistent.
- The second bundle attempt succeeded once the diary prompt block was made PDF-friendly.
- The remote listing confirmed the uploaded PDF:
  - `GOJA-14 Branch review handoff`

### What didn't work

- The first `remarquee upload bundle` attempt failed during pandoc PDF generation.
- Exact failure:

```text
Error: pandoc failed: Error producing PDF.
! Undefined control sequence.
l.3709 ...workspaces/2026-03-18/add-goja-plugins\n
```

- Root cause: the diary contained the entire multi-line prompt as one giant inline quoted string with escaped sequences, which rendered poorly for LaTeX/PDF generation.

### What I learned

- Review handoff docs intended for PDF delivery need stricter markdown hygiene than repo-local notes.
- For reMarkable delivery, fenced blocks are much safer than giant inline quoted strings when preserving prompt text.
- The longer, more narrative prose materially improves the usefulness of the packet when code is not open nearby.

### What was tricky to build

- The hardest part was balancing three competing goals:
  - enough technical detail to support a real review,
  - enough embedded diffs and excerpts to work offline,
  - enough prose continuity that the document remains readable rather than feeling like a stitched-together changelog.
- The bundle generation failure also forced a second kind of care: not just good engineering content, but markdown that survives PDF tooling.

### What warrants a second pair of eyes

- Whether the final packet is now sufficiently self-contained for a true offline review.
- Whether the selected diff excerpts are the highest-value ones, or if one more appendix would still help.

### What should be done in the future

- N/A

### Code review instructions

- Verify:
  - `docmgr doctor --ticket GOJA-14-BRANCH-REVIEW-HANDOFF --stale-after 30`
  - `remarquee cloud ls /ai/2026/03/18/GOJA-14-BRANCH-REVIEW-HANDOFF --long --non-interactive`
- Read the design doc first, then use the helper script if you want to recreate the branch-surface view locally.

### Technical details

Planned commands:

```bash
cd /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja
docmgr doc relate --doc /abs/path/to/design-doc.md --file-note "/abs/path/to/file.go:reason"
docmgr doctor --ticket GOJA-14-BRANCH-REVIEW-HANDOFF --stale-after 30
remarquee upload bundle --dry-run ...
remarquee upload bundle ...
remarquee cloud ls /ai/2026/03/18/GOJA-14-BRANCH-REVIEW-HANDOFF --long --non-interactive
```

Commands actually run:

```bash
cd /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja
docmgr doc relate --doc /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/ttmp/2026/03/18/GOJA-14-BRANCH-REVIEW-HANDOFF--comprehensive-code-review-handoff-for-plugin-and-docs-branch-delta-since-origin-main/design-doc/01-comprehensive-branch-review-handoff-guide-for-code-review-since-origin-main.md --file-note "/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/engine/factory.go:Core runtime composition entrypoint and registrar phase" ...
docmgr doc relate --doc /home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/ttmp/2026/03/18/GOJA-14-BRANCH-REVIEW-HANDOFF--comprehensive-code-review-handoff-for-plugin-and-docs-branch-delta-since-origin-main/reference/01-investigation-diary.md --file-note "/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/ttmp/2026/03/18/GOJA-14-BRANCH-REVIEW-HANDOFF--comprehensive-code-review-handoff-for-plugin-and-docs-branch-delta-since-origin-main/scripts/review-handoff-context.sh:Helper script that reruns the branch-surface context commands" ...
docmgr vocab add --category topics --slug documentation --description "Documentation systems, help content, and doc-access surfaces."
docmgr vocab add --category topics --slug plugins --description "Runtime plugin loading, plugin authoring, and plugin integration work."
docmgr vocab add --category topics --slug review --description "Review-oriented analysis, cleanup, and branch handoff documentation."
docmgr doctor --ticket GOJA-14-BRANCH-REVIEW-HANDOFF --stale-after 30
remarquee upload bundle --dry-run ...
remarquee upload bundle ...
remarquee cloud ls /ai/2026/03/18/GOJA-14-BRANCH-REVIEW-HANDOFF --long --non-interactive
```

## Related

- [GOJA-14 design guide](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/ttmp/2026/03/18/GOJA-14-BRANCH-REVIEW-HANDOFF--comprehensive-code-review-handoff-for-plugin-and-docs-branch-delta-since-origin-main/design-doc/01-comprehensive-branch-review-handoff-guide-for-code-review-since-origin-main.md)
- [GOJA-13 architecture review](/home/manuel/workspaces/2026-03-18/add-goja-plugins/go-go-goja/ttmp/2026/03/18/GOJA-13-ORIGIN-MAIN-ARCHITECTURE-REVIEW--architecture-and-code-review-of-goja-plugin-work-since-origin-main/design-doc/01-origin-main-review-report-for-plugin-and-documentation-architecture.md)
