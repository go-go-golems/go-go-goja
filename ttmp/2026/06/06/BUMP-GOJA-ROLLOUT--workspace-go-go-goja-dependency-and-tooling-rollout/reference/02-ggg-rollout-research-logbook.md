---
Title: GGG Rollout Research Logbook
Ticket: BUMP-GOJA-ROLLOUT
Status: active
Topics:
    - go
    - tooling
    - maintenance
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../../code/wesen/go-go-golems/infra-tooling/README.md
      Note: |-
        Entry point that listed the relevant rollout, release, Codex, and CI resources
        Resource discovery and relevance evaluation
    - Path: ../../../../../../../../../../code/wesen/go-go-golems/infra-tooling/docs/go-go-golems/package-publishing-release-train.md
      Note: |-
        Most useful source for large release train operation
        Evaluated for currency and rollout guidance
    - Path: ../../../../../../../../../../code/wesen/go-go-golems/infra-tooling/docs/go-go-golems/playbooks/pr-readiness-check-scripts.md
      Note: |-
        Most useful source for Codex/PR readiness semantics
        Evaluated for Codex and readiness guidance
    - Path: ../../../../../../../../../../code/wesen/go-go-golems/infra-tooling/pkg/rollout/config.go
      Note: |-
        Source of truth for rollout YAML schema
        Evaluated because docs lacked a full schema
ExternalSources: []
Summary: Research logbook evaluating infra-tooling resources used to build the ggg rollout operations playbook, including what was useful, stale, wrong, and needs updating.
LastUpdated: 2026-06-07T09:15:00-04:00
WhatFor: Track which rollout/ggg resources are useful, stale, misleading, or need follow-up updates after the BUMP-GOJA-ROLLOUT work.
WhenToUse: When updating infra-tooling documentation, running another large rollout, or deciding which docs should be trusted as operator guidance.
---


# GGG Rollout Research Logbook

## Goal

This logbook records the resources consulted while researching how to use `ggg` for large go-go-golems rollouts: repository inventory, rollout configs, CI/CD checks, Codex review triggers, PR readiness watching, release tagging, and post-merge verification.

Each entry records:

- what I was researching,
- what I was looking for in this resource,
- why I chose it,
- how I found it,
- what was useful,
- what was not useful,
- what is out of date or wrong,
- what needs updating.

## Summary Table

| Resource | Useful? | Current? | Main value | Needs update? |
|---|---|---|---|---|
| `infra-tooling/README.md` | Yes | Mostly | Entry-point map to `ggg` and playbooks | Add a concrete `ggg rollout` quickstart |
| `package-publishing-release-train.md` | Very | Mostly | Best end-to-end release train procedure | Add more `ggg rollout` YAML examples |
| `pr-readiness-check-scripts.md` | Very | Current | Best Codex/readiness semantics | Rename or clarify title; it is now about `ggg`, not scripts |
| `glazed-linting-rollout-playbook.md` | Yes | Current for linting | Glazed lint rollout and CI wiring | Cross-link to `ggg rollout validate/push-prs` |
| `logcopter-package-rollout-playbook.md` | Yes | Mostly | Dependency-order and package logger rollout | Add `ggg rollout` examples in the PR phase |
| `ggg --help` and subcommand help | Yes | Current | Actual installed command surface | Help is noisy because Glazed flags dominate |
| `pkg/rollout/config.go` | Very | Current | YAML schema source of truth | Needs user-facing schema docs |
| `pkg/rollout/*.go` source scan | Medium | Current | Confirms behavior not obvious in docs | Docs should expose behavior to avoid reading source |
| `pkg/prready/prready.go` source scan | Medium | Current | Confirms readiness state machine | PR readiness doc already covers most of it |
| Historical scripts under `scripts/go-go-golems/` | Low | Historical | Backward reference only | Mark clearly as deprecated in directory README |

## Entry 1: `infra-tooling/README.md`

### What I was researching

I was looking for the canonical entry point for go-go-golems rollout tooling: where the docs say to start, which commands are current, and whether `ggg` is the intended interface.

### What I was looking for in this document in particular

I wanted a top-level map of the repository: which documents cover release trains, PR readiness, Codex, CI/CD, logcopter, Glazed linting, and reusable workflows.

### Why I chose it

A repository README should answer “where do I start?” before diving into narrower playbooks or source code.

### How I found the resource itself

I searched the infra-tooling repository for `ggg`, `rollout`, `codex`, `CI`, `PR`, and related terms. The README appeared as the repository-level overview and explicitly mentioned `ggg`.

### What I found useful

- It states that the repo contains an installed `ggg` CLI for GitHub Actions, Codex PR readiness, batch watching, Codex comments, and release tagging.
- It lists the recommended reuse points:
  - `package-publishing-release-train.md`
  - `logcopter-package-rollout-playbook.md`
  - `pr-readiness-check-scripts.md`
  - `glazed-linting-rollout-playbook.md`
  - installed `ggg` commands.
- It clarifies that historical scripts exist but `ggg` is the preferred current interface.

### What I didn't find useful

- It does not contain a concrete command sequence for a multi-repo rollout.
- It does not show a sample `rollout.yaml`.
- It does not explain how `ggg rollout` relates to `ggg batch ready` and `ggg run status`.

### What is out of date / what was wrong

Nothing materially wrong. The README is accurate as an overview, but it under-describes the newer `ggg rollout` subcommands.

### What would need updating

Add a short “Large rollout quickstart” section:

```bash
ggg rollout inventory ...
ggg rollout init ...
ggg rollout validate rollout.yaml
ggg rollout push-prs rollout.yaml --yes
ggg pr codex-trigger --file prs.yaml --wait-for-auto 30s
ggg batch ready prs.yaml --watch --until actionable
ggg batch actions actions.yaml --watch
```

## Entry 2: `docs/go-go-golems/package-publishing-release-train.md`

### What I was researching

I was researching the correct operator flow for package publishing and dependency release trains: dependency order, downstream bumping, PR creation, Codex review, CI/CD checking, merging, and tagging.

### What I was looking for in this document in particular

I wanted the high-level policy and sequence: what must happen before merging, how to avoid workspace leakage, how to trigger Codex, how to watch PRs in batch, and how to verify post-merge CI.

### Why I chose it

The README named this as the primary starting point for go-go-golems package publishing and release trains. The title directly matches the rollout problem.

### How I found the resource itself

The README listed it under “Current Recommended Reuse Points.” It also appeared in repository search for `rollout`, `ggg`, and `Codex`.

### What I found useful

- The release train principle: follow dependency order and do not merge downstream until upstream is merged and published.
- The explicit `GOWORK=off` guidance for downstream validation.
- Early downstream PR guidance: open PRs early for feedback, but do not merge until upstream tags are available.
- Exact `ggg` commands for:
  - `ggg pr codex-trigger --file ... --wait-for-auto 30s`
  - `ggg batch ready ... --watch --until actionable`
  - `ggg pr ready --findings`
  - `ggg run status`
  - `ggg batch actions`
  - `ggg release preflight`
  - `ggg release tag-patch`
  - `ggg release watch --verify-docs`
- Readiness criteria for mergeability, checks, and Codex satisfaction.
- Merge policy: use merge commits, not squash.
- Common gotchas such as GitHub token workflow scope and standard library govulncheck failures.

### What I didn't find useful

- It is broad and release-train oriented, so it does not provide a full `ggg rollout init` / `rollout.yaml` schema example.
- It does not show how to use `ggg rollout push-prs` to generate the PR YAML consumed by `ggg batch ready`.
- It does not distinguish strongly between manual per-repo commits and `ggg rollout branch --commit`.

### What is out of date / what was wrong

No major wrong content found. The document is current for the `ggg pr`, `ggg batch`, `ggg run`, and `ggg release` commands. It is incomplete for the newer `ggg rollout` workflow.

### What would need updating

Add a section titled “Using `ggg rollout` for the repository set” with:

- `ggg rollout inventory`,
- `ggg rollout init`,
- sample `rollout.yaml`,
- `ggg rollout validate`,
- `ggg rollout branch`,
- `ggg rollout push-prs`,
- `ggg rollout status`,
- `ggg rollout report`.

## Entry 3: `docs/go-go-golems/playbooks/pr-readiness-check-scripts.md`

### What I was researching

I was researching how `ggg` decides whether a PR is ready, how Codex review state is interpreted, what batch watch exit codes mean, and how to inspect Codex comments.

### What I was looking for in this document in particular

I needed exact semantics for:

- satisfied Codex review,
- stale vs current-head feedback,
- `EYES` reactions,
- failed checks,
- merge conflicts,
- batch watch stop modes and exit codes.

### Why I chose it

The README listed it as the PR readiness reference, and it has related files pointing at `ggg pr ready`, `ggg pr codex-trigger`, `ggg batch ready`, and `pkg/prready`.

### How I found the resource itself

It appeared in the README and in `rg` search output for `codex`, `PR`, and `ready`.

### What I found useful

- It explicitly says new workflows should use installed `ggg`, not the old scripts.
- It gives the command overview for:
  - `ggg pr ready`,
  - `ggg pr codex-trigger`,
  - `ggg pr codex-comments`,
  - `ggg pr watch`,
  - `ggg batch ready`.
- It documents PR list YAML format.
- It explains Codex trigger safety behavior.
- It documents batch watch `--until actionable`, `--until all-ready`, `--until terminal`, and `--until first-ready`.
- It lists exit codes `0` through `6` and their meanings.
- It explains readiness states such as `waiting_checks`, `waiting_codex`, `failed_checks`, `merge_conflict`, and `codex_feedback`.

### What I didn't find useful

- The title includes “scripts,” which is misleading because the body says the scripts are historical and `ggg` is current.
- It focuses on PR readiness only; it does not cover `ggg rollout` config, branch, validate, push-prs, or post-merge release tagging.

### What is out of date / what was wrong

The content is current, but the title is stale. It should not be called “PR readiness check scripts” if operators are expected to use `ggg`.

### What would need updating

Rename or alias the document to something like:

```text
PR readiness with ggg
```

Add a short cross-link to the release train and rollout config playbooks.

## Entry 4: `docs/go-go-golems/glazed-linting-rollout-playbook.md`

### What I was researching

I was researching the linting side of large rollouts: how Glazed CLI linting should be added, validated, wired into Makefiles/hooks/CI, and treated during dependency bumps.

### What I was looking for in this document in particular

I wanted the standard Makefile targets, validation commands, GitHub Actions guidance, and policy for suppressions/allow paths.

### Why I chose it

This rollout included Glazed linting in some repositories, and the infra-tooling README lists this playbook as a recommended reuse point.

### How I found the resource itself

It was listed in the README and had already been read earlier in the rollout as part of the lint/logcopter guidance.

### What I found useful

- It documents the `glazed-lint` vettool package.
- It provides Makefile snippets for `GLAZED_LINT_BIN`, `GLAZED_VERSION`, `glazed-lint-build`, and `glazed-lint`.
- It says to run Glazed lint after dependency bumps so the analyzer comes from the published Glazed version.
- It warns against broad allow-paths and requires reasons for suppressions.
- It says to open PRs and wait for CI/Codex readiness because linter wiring affects hooks and release behavior.

### What I didn't find useful

- It does not directly show how to batch-roll this out with `ggg rollout`.
- It does not show how to put Glazed lint validation into a rollout YAML.

### What is out of date / what was wrong

No wrong content found. It appears current for Glazed linting.

### What would need updating

Add a small section:

```yaml
validation:
  commands:
    - name: test
      run: GOWORK=off go test ./...
    - name: glazed-lint
      run: make glazed-lint
```

and show `ggg rollout validate` / `ggg rollout push-prs` usage for lint rollouts.

## Entry 5: `docs/go-go-golems/playbooks/logcopter-package-rollout-playbook.md`

### What I was researching

I was researching package logger/logcopter rollout mechanics and how those interact with release trains and downstream dependency bumps.

### What I was looking for in this document in particular

I wanted the step-by-step logcopter adoption sequence and any notes on dependency order, generated files, and Codex/CI handling.

### Why I chose it

The README listed it as the logcopter package rollout guide, and logcopter/package logger baselines were part of the broader rollout context.

### How I found the resource itself

It was listed in the README and appeared in search results for `rollout`, `Codex`, and `dependency`.

### What I found useful

- It explains the target shape for logcopter-enabled repos.
- It gives the dependency-order loop:
  1. merge and release upstream,
  2. bump downstream,
  3. validate with `GOWORK=off`,
  4. open PR,
  5. wait for Actions and Codex.
- It explains non-mutating `make logcopter-check` vs mutating `go generate ./...`.
- It gives practical guidance on logger conversion and generated files.

### What I didn't find useful

- It is focused on logcopter mechanics, so it is not a general `ggg` rollout operations guide.
- It still describes the PR phase mostly conceptually rather than showing `ggg rollout` commands.

### What is out of date / what was wrong

No wrong content found. It is accurate for logcopter, but should be cross-linked to the newer `ggg` operations flow.

### What would need updating

Add examples for:

```bash
ggg rollout validate rollout.yaml
ggg rollout push-prs rollout.yaml --yes
ggg pr codex-trigger --file prs.yaml --wait-for-auto 30s
ggg batch ready prs.yaml --watch --until actionable
```

## Entry 6: Installed `ggg` CLI help

### What I was researching

I was researching the actual installed command surface, especially whether the docs match the CLI on disk.

### What I was looking for in this resource in particular

I wanted to confirm available commands and flags for:

- `ggg rollout`,
- `ggg pr`,
- `ggg batch`,
- `ggg run`,
- `ggg release`.

### Why I chose it

CLI help is the closest runtime reference for what the operator can actually invoke.

### How I found the resource itself

The README said `ggg` should already be installed. I ran `command -v ggg` and then `ggg --help` plus subcommand help.

### What I found useful

- Confirmed installed binary path: `/home/manuel/.local/bin/ggg`.
- Confirmed top-level commands:
  - `batch`,
  - `pr`,
  - `release`,
  - `rollout`,
  - `run`.
- Confirmed `ggg rollout` subcommands:
  - `branch`,
  - `docsctl`,
  - `init`,
  - `inventory`,
  - `plan`,
  - `push-prs`,
  - `report`,
  - `status`,
  - `validate`.
- Confirmed `ggg batch actions` exists for CI/CD monitoring.
- Confirmed `ggg pr codex-trigger`, `ggg pr codex-comments`, `ggg pr ready`, and `ggg pr watch` exist.

### What I didn't find useful

- Help output is very noisy because Glazed structured-output flags dominate each command.
- `ggg rollout init --help` lists all flags but does not show a concise example.
- `ggg rollout validate --help` does not explain log file naming or result semantics in prose.

### What is out of date / what was wrong

No runtime mismatch found for commands used in the playbook. The issue is presentation, not correctness.

### What would need updating

Add concise examples to each `ggg rollout` subcommand help page, or add a `ggg rollout quickstart` help topic.

## Entry 7: Generated sample from `ggg rollout init`

### What I was researching

I wanted to see the exact YAML structure produced by `ggg rollout init` rather than inferring it from docs.

### What I was looking for in this resource in particular

I needed the concrete field names for rollout config: selection, validation, PR output, readiness, release.

### Why I chose it

The package release-train doc did not contain a rollout YAML schema, and CLI help did not show an example.

### How I found the resource itself

I ran a temporary command:

```bash
ggg rollout init \
  --id TEST \
  --name 'Test rollout' \
  --workspace /tmp/ws \
  --branch task/test \
  --include repo-a,repo-b \
  --require-module github.com/go-go-golems/go-go-goja \
  --validation 'GOWORK=off go test ./...' \
  --write-to /tmp/rollout-test.yaml \
  --output yaml
```

### What I found useful

The generated file showed the actual schema:

```yaml
id: TEST
name: Test rollout
workspace: /tmp/ws
branch: task/test
base: origin/main
commit_message: Apply rollout changes
selection:
  require_go_mod_contains:
    - github.com/go-go-golems/go-go-goja
  include:
    - repo-a
    - repo-b
  exclude: []
validation:
  commands:
    - name: validation
      run: GOWORK=off go test ./...
  continue_on_error: true
  log_dir: .ggg-rollout-logs
pull_request:
  title: Apply rollout changes
  body_file: ""
  output_prs: ""
  no_verify_push: false
readiness:
  trigger_codex: false
  watch: false
release:
  mode: ""
  require_manual_approval: false
```

### What I didn't find useful

The generated config uses a generic validation command name (`validation`) when multiple validation commands may be clearer as `build`, `test`, `lint`, etc.

### What is out of date / what was wrong

Nothing wrong. This was generated from the installed CLI.

### What would need updating

The generated schema should be copied into infra-tooling docs so operators do not need to run a sample command or read source.

## Entry 8: `pkg/rollout/config.go`

### What I was researching

I was researching the rollout YAML schema and defaults at the source-code level.

### What I was looking for in this file in particular

I wanted exact field names, types, default behavior, and target resolution semantics.

### Why I chose it

The docs and CLI help did not include a full schema reference.

### How I found the resource itself

Search results for `pkg/rollout` and `ggg rollout` led to the rollout package. `config.go` was the obvious schema file.

### What I found useful

- It defines `Config` with:
  - `ID`,
  - `Name`,
  - `Workspace`,
  - `Branch`,
  - `Base`,
  - `CommitMessage`,
  - `Selection`,
  - `Validation`,
  - `PullRequest`,
  - `Readiness`,
  - `Release`.
- It defines selection by explicit include/exclude or by `require_go_mod_contains` inventory.
- It confirms default `base` is `origin/main`.
- It confirms target resolution uses explicit includes first; otherwise inventory is based on module requirements.

### What I didn't find useful

- Source code is not operator documentation.
- It does not explain how fields should be used in real rollouts.

### What is out of date / what was wrong

No code issue found.

### What would need updating

Add a user-facing schema reference to the docs with comments and examples.

## Entry 9: `pkg/rollout/pushprs.go`, `status.go`, and `report.go`

### What I was researching

I wanted to understand how `ggg rollout push-prs`, `status`, and `report` tie together with the PR list consumed by `ggg batch ready`.

### What I was looking for in these files in particular

I wanted to confirm:

- whether PR URLs are written to a file,
- what format that file uses,
- whether status consumes readiness reports,
- what `report` includes.

### Why I chose them

The help text lists the commands, but the docs did not explain the data handoff between rollout PR creation and batch readiness.

### How I found the resource itself

`rg` search for `OutputPRs`, `PushPRs`, and `rollout status` led to these files.

### What I found useful

- `pushprs.go` writes a YAML file containing `prs:` when `pull_request.output_prs` is configured.
- `status.go` reads readiness reports keyed by repository when PR output exists.
- `report.go` includes the PR output file content and tells the operator to run `ggg batch ready`.

### What I didn't find useful

- Reading source was necessary to understand behavior that should be in docs.
- The report guidance is useful but terse.

### What is out of date / what was wrong

No source issue found.

### What would need updating

Document the file handoff explicitly:

```text
ggg rollout push-prs -> pull_request.output_prs -> ggg batch ready / ggg pr codex-trigger --file
```

## Entry 10: `pkg/prready/prready.go`

### What I was researching

I wanted to confirm how `ggg` classifies PR readiness internally and what blocks a merge.

### What I was looking for in this file in particular

I wanted the underlying state machine: failed checks, no Codex, waiting Codex, Codex feedback, stale feedback, merge conflicts, terminal states.

### Why I chose it

The PR readiness doc referenced `pkg/prready/prready.go` as the shared readiness implementation.

### How I found the resource itself

The PR readiness document lists this file in `RelatedFiles`, and repository search for `CodexFeedback` / `waiting_codex` led directly to it.

### What I found useful

- It confirms readiness states and terminal behavior.
- It confirms Codex review comments and substantive bodies are treated as blockers.
- It confirms stale Codex feedback is detected by reviewed commit markers.
- It confirms checks must exist and complete successfully/skipped/neutral.

### What I didn't find useful

- Most operators should not need to read this; the PR readiness document already explains it well.

### What is out of date / what was wrong

No issue found.

### What would need updating

Keep the PR readiness doc in sync with any changes to `pkg/prready` state names or exit codes.

## Entry 11: Historical scripts under `scripts/go-go-golems/`

### What I was researching

I wanted to know whether older PR readiness scripts are still expected in operator workflows.

### What I was looking for in this resource in particular

I was checking whether commands like `00-pr-ready-check.sh` and `05-batch-pr-ready.sh` should still be used or whether they are superseded.

### Why I chose it

The README and PR readiness doc mention historical scripts, and the user asked for guidance on “ggg specifically,” so it was important to identify what not to use.

### How I found the resource itself

The PR readiness document lists the historical scripts by name, and `find` showed them under `scripts/go-go-golems/`.

### What I found useful

- The names are useful for understanding the evolution of the workflow.
- They may still help diagnose behavior if someone remembers the old scripts.

### What I didn't find useful

- They are not the right interface for new work.
- They split behavior across multiple scripts that `ggg` now centralizes.

### What is out of date / what was wrong

The scripts are historical. The current docs explicitly say not to add new playbook examples that call them.

### What would need updating

Add a small README in `scripts/go-go-golems/` saying:

```text
Historical reference only. Use ggg pr ready, ggg batch ready, ggg pr codex-trigger, and ggg run status for new workflows.
```

## Entry 12: `docs/go-go-golems/playbooks/docsctl-docs-publishing-rollout-playbook.md` and docsctl mentions

### What I was researching

I briefly checked docsctl references because release-train docs mention docs publishing verification with `ggg release watch --verify-docs`.

### What I was looking for in this document in particular

I wanted to know whether docsctl publishing affects generic dependency-rollout PR readiness or only tag/release verification.

### Why I chose it

The package release train document lists docsctl as a cross-cutting rollout check.

### How I found the resource itself

The `find` listing of docs and the package release train document pointed at docsctl publishing docs.

### What I found useful

- It clarifies that docsctl is relevant when a repo has tag-triggered docs publishing or release workflows.
- It reinforces that release verification can go beyond Go module tags.

### What I didn't find useful

- It is not central to the go-go-goja dependency bump rollout because the immediate work focused on code/API migrations rather than docs publishing.

### What is out of date / what was wrong

Not evaluated deeply enough to make a correctness claim.

### What would need updating

Cross-link docsctl rollout docs from the package release-train document is already partially present; add a note to the `ggg` rollout playbook that docsctl validation is optional and repo-specific.

## Recommended Documentation Updates

1. **Add a `ggg rollout` quickstart to `infra-tooling/README.md`.**
2. **Add a `rollout.yaml` schema section to `package-publishing-release-train.md`.**
3. **Rename or alias `pr-readiness-check-scripts.md` to “PR readiness with ggg.”**
4. **Add `ggg rollout` examples to the Glazed lint and logcopter playbooks.**
5. **Add a deprecation README to `scripts/go-go-golems/`.**
6. **Add concise examples to `ggg rollout --help` or provide a `ggg rollout quickstart` help topic.**

## Quick Resource Ranking for Future Operators

Start here, in order:

1. `docs/go-go-golems/package-publishing-release-train.md`
2. `docs/go-go-golems/playbooks/pr-readiness-check-scripts.md`
3. `ggg --help` and specific subcommand help
4. `pkg/rollout/config.go` only if YAML schema behavior is unclear
5. Specialized playbooks (`glazed-linting`, `logcopter`, `docsctl`) depending on the rollout type

Avoid starting with historical scripts unless debugging legacy behavior.
