---
Title: Research logbook
Ticket: XGOJA-JSVERBS-ROOT-001
Status: active
Topics:
    - xgoja
    - jsverbs
    - cli
    - buildspec
    - docsctl
    - release
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../../code/wesen/go-go-golems/infra-tooling/docs/go-go-golems/playbooks/docsctl-docs-publishing-rollout-playbook.md
      Note: |-
        operational docsctl publishing playbook used to determine CI/Vault permissions.
        docsctl rollout playbook reviewed and updated
    - Path: ../../../../../../../../../../code/wesen/terraform/vault/github-actions/envs/k3s/main.tf
      Note: |-
        Terraform Vault publisher-role map updated for goja-text docsctl publishing.
        Vault docsctl publisher role binding reviewed and updated
    - Path: ../../../../../../../goja-text/.github/workflows/publish-docs.yaml
      Note: goja-text separate docsctl publish workflow
    - Path: ../../../../../../../goja-text/.github/workflows/release.yaml
      Note: release workflow reviewed and updated for docsctl publishing.
    - Path: ../../../../../../../goja-text/.goreleaser.yaml
      Note: GoReleaser config reviewed and updated now that cmd/goja-text is a committed generated command module.
    - Path: cmd/xgoja/doc/02-user-guide.md
      Note: |-
        xgoja user-facing jsverbs and path-resolution documentation reviewed and updated during the ticket.
        xgoja user guide reviewed for jsverbs/path semantics
    - Path: cmd/xgoja/doc/03-tutorial-using-xgoja-yaml.md
      Note: xgoja tutorial reviewed and updated with root-mounted jsverbs examples.
    - Path: cmd/xgoja/doc/06-buildspec-reference.md
      Note: buildspec reference reviewed and updated for commands.jsverbs.mount.
ExternalSources:
    - https://goreleaser.com/customization/builds/builders/go/
Summary: Logbook of resources used for root-mounted xgoja jsverbs, committed goja-text generated binary scaffolding, release/docsctl publishing, and Vault CI permissions.
LastUpdated: 2026-06-02T16:15:00-04:00
WhatFor: Use this document to remember which internal docs, workflows, configs, and external references were useful, stale, or need follow-up updates.
WhenToUse: Before continuing XGOJA-JSVERBS-ROOT-001, releasing goja-text, debugging docsctl publishing, or updating xgoja/goja-text release documentation.
---


# Research logbook

## Goal

Track every document and external resource that materially shaped the xgoja root-mounted JavaScript verbs work and the follow-on goja-text release/docs publishing work. The log records why each resource was consulted, how it was found, what was useful, what was stale or wrong, and what should be updated next.

## Context

The ticket began as an xgoja feature request: allow generated binaries to mount JavaScript verb packages directly under the root command instead of under the default `verbs` command. During validation, goja-text became the downstream example binary, gained a committed generated `cmd/goja-text` module, moved to released `go-go-goja`/`xgoja` v0.7.4, and was wired for GoReleaser and docsctl publishing.

This logbook is intentionally operational. It is not a polished design narrative; it is a continuation aid for future release, documentation, and infrastructure work.

## Status summary

| Resource | Useful? | Out of date? | Needs update? | Notes |
|---|---:|---:|---:|---|
| xgoja user/buildspec docs | Yes | Partly, before this ticket | Mostly updated | Root-mounted jsverbs and path semantics were added/corrected. |
| xgoja generator/build code | Yes | No | Maybe | Future docs should mention runtime filesystem path caveat carefully. |
| goja-text README/build docs | Yes | Yes, before committed scaffold | Updated | Build instructions now describe committed generated command module. |
| goja-text release workflow | Yes | Yes | Updated | Enabled docsctl publishing with job-scoped OIDC. |
| goja-text GoReleaser config | Yes | Yes | Updated | It previously skipped builds because no checked-in command existed. |
| infra docsctl playbook | Very useful | Mostly current | Minor addition possible | Could add nested-module export command pattern. |
| Terraform Vault docsctl publisher map | Essential | Missing goja-text | Updated and applied | `docsctl-goja-text-publisher` now exists. |
| GoReleaser online docs search | Somewhat useful | Unknown | No local update needed | Used only to confirm/triage nested module build config. |

## Resource entries

### 1. xgoja user guide

- **Resource:** `/home/manuel/workspaces/2026-06-02/goja-text/go-go-goja/cmd/xgoja/doc/02-user-guide.md`
- **What I was researching:** How xgoja currently documents generated commands, JavaScript verb mounting, path resolution, embedded sources, and runtime behavior.
- **What I was looking for in this document in particular:** The user-facing location where `commands.jsverbs.name`, default `verbs` behavior, embedded jsverbs, and root command behavior should be explained.
- **Why I chose it:** It is the primary xgoja user guide and therefore the first place users will look when trying to understand why a generated binary has `verbs ...` versus root-level commands.
- **How I found the resource:** It was already part of the go-go-goja xgoja docs tree and appeared in the current ticket context as one of the important xgoja documentation files.
- **What I found useful:** The document already had sections for buildspec concepts, command modes, and JavaScript verbs. That gave a clear insertion point for root-mounted jsverbs and path-resolution notes.
- **What I didn't find useful:** It did not provide enough precision around build-time spec-relative paths versus runtime filesystem paths, so it was easy to overstate that runtime filesystem paths were spec-relative.
- **What is out of date / what was wrong:** Before correction, my draft documentation incorrectly implied that runtime filesystem `jsverbs[].path` is rewritten or interpreted spec-relative by the generated binary. The generator only rewrites embedded paths; runtime filesystem paths are stored as written.
- **What would need updating:** Keep an eye on any future xgoja generator change that alters runtime path behavior. If runtime filesystem sources ever become spec-relative at runtime, this guide must be updated again.

### 2. xgoja YAML tutorial

- **Resource:** `/home/manuel/workspaces/2026-06-02/goja-text/go-go-goja/cmd/xgoja/doc/03-tutorial-using-xgoja-yaml.md`
- **What I was researching:** How to teach users the practical xgoja buildspec workflow, especially embedded jsverbs and command mounting.
- **What I was looking for in this document in particular:** A tutorial-style example showing `commands.jsverbs.mount: root` and explaining when root mounting is appropriate.
- **Why I chose it:** The feature is a buildspec option, so a tutorial is the best place to explain the workflow and tradeoff rather than only listing schema fields.
- **How I found the resource:** It was in the xgoja docs directory and had already been identified as one of the important docs for xgoja path and buildspec behavior.
- **What I found useful:** The tutorial structure already separated default behavior, embedded source behavior, and decision points. It was easy to add a decision table row for root-mounted jsverbs.
- **What I didn't find useful:** It did not previously call out the self-contained helper-binary use case that motivated goja-text.
- **What is out of date / what was wrong:** Before this work, it still described JavaScript verbs as effectively living under the configured jsverbs command and did not cover direct root mounting.
- **What would need updating:** Add a more complete worked example if root-mounted jsverbs become a common pattern across more packages. Consider a short collision-warning section with built-in commands like `help`, `eval`, `run`, `repl`, and `modules`.

### 3. xgoja buildspec reference

- **Resource:** `/home/manuel/workspaces/2026-06-02/goja-text/go-go-goja/cmd/xgoja/doc/06-buildspec-reference.md`
- **What I was researching:** The authoritative buildspec field reference for command and path semantics.
- **What I was looking for in this document in particular:** Where to document `commands.jsverbs.mount`, accepted values, defaults, and path-resolution rules.
- **Why I chose it:** A new buildspec field should be represented in the reference, not only in prose tutorials.
- **How I found the resource:** It was listed in the existing xgoja documentation set and matched the buildspec validation changes.
- **What I found useful:** It provided concise bullet-style semantics suitable for exact values like `root`, `/`, and `.`.
- **What I didn't find useful:** It was too terse to fully communicate operational caveats; those belong in the user guide/tutorial instead.
- **What is out of date / what was wrong:** It lacked `commands.jsverbs.mount` entirely before this ticket. It also needed correction around runtime filesystem path wording.
- **What would need updating:** If more mount modes are added, this reference should list every accepted value and explicitly state the default container behavior.

### 4. xgoja generator and build command implementation

- **Resources:**
  - `/home/manuel/workspaces/2026-06-02/goja-text/go-go-goja/cmd/xgoja/cmd_build.go`
  - `/home/manuel/workspaces/2026-06-02/goja-text/go-go-goja/cmd/xgoja/internal/generate/generate.go`
  - `/home/manuel/workspaces/2026-06-02/goja-text/go-go-goja/cmd/xgoja/internal/generate/main.go`
  - `/home/manuel/workspaces/2026-06-02/goja-text/go-go-goja/cmd/xgoja/internal/generate/gomod.go`
- **What I was researching:** How xgoja writes generated workspaces, how `--work-dir`, `--dry-run`, and `--xgoja-replace` behave, and what generated files can be committed under `cmd/goja-text`.
- **What I was looking for in this document/code in particular:** Whether `xgoja build --work-dir . --dry-run` writes a complete generated module without compiling, and how module replacements and embedded paths are generated.
- **Why I chose it:** The user asked to keep generated `main.go` and scaffolding committed and reproducible through `go tool` + `go generate`; the generator code is the source of truth for that workflow.
- **How I found the resource:** I inspected xgoja’s build command after deciding that `cmd/goja-text` should be a checked-in generated command module rather than only a temporary build output.
- **What I found useful:** `cmd_build.go` confirmed that `generate.WriteAll` runs before the dry-run exit, so `--dry-run --work-dir .` can refresh checked-in generated files without compiling. `gomod.go` clarified how local package replaces are resolved.
- **What I didn't find useful:** The generator is optimized for temporary workspaces, not for a polished committed nested-module workflow. That required a small `internal/postgenerate` normalizer in goja-text.
- **What is out of date / what was wrong:** No implementation issue was found, but the docs around runtime filesystem paths were easy to misread until `RenderEmbeddedSpec` was inspected.
- **What would need updating:** xgoja could eventually grow a first-class `xgoja generate` command for committed scaffolds, avoiding the slightly surprising `build --dry-run --work-dir .` pattern.

### 5. go-go-goja release workflow

- **Resource:** `/home/manuel/workspaces/2026-06-02/goja-text/go-go-goja/.github/workflows/release.yaml`
- **What I was researching:** The established release workflow shape for GoReleaser split builds and docsctl publishing in a go-go-golems package.
- **What I was looking for in this document in particular:** The working `publish-docs` job structure, docsctl reusable workflow inputs, and where docs publishing sits relative to GoReleaser merge.
- **Why I chose it:** go-go-goja already publishes docsctl docs, so it is a close model for goja-text.
- **How I found the resource:** The user explicitly asked to look at `go-go-goja/.github/workflows/`.
- **What I found useful:** The job uses `go-go-golems/infra-tooling/.github/workflows/publish-docsctl.yml@main`, passes package/version/export command, and runs after `goreleaser-merge`.
- **What I didn't find useful:** The export command was specific to `./cmd/goja-repl` and did not cover nested modules.
- **What is out of date / what was wrong:** No obvious issue for go-go-goja itself. It still used `goreleaser-action@v6`; Glazed uses v7, but goja-text already had v6 in its generated workflow.
- **What would need updating:** If repository policy standardizes on `goreleaser-action@v7`, go-go-goja and goja-text workflows should be aligned later.

### 6. Glazed release workflow and GoReleaser config

- **Resources:**
  - `/home/manuel/workspaces/2026-06-02/goja-text/glazed/.github/workflows/release.yaml`
  - `/home/manuel/workspaces/2026-06-02/goja-text/glazed/.goreleaser.yaml`
- **What I was researching:** A mature example of docsctl publishing, GoReleaser binary builds, Homebrew formula config, nfpm packages, and Fury publisher filtering.
- **What I was looking for in this document in particular:** The exact docsctl workflow shape and release packaging patterns to copy into goja-text now that it has a committed generated CLI.
- **Why I chose it:** Glazed is the canonical source for docsctl and GoReleaser publishing patterns in this ecosystem.
- **How I found the resource:** The user suggested that relevant docs or examples might be in `glazed/`, so I searched its workflows and release config.
- **What I found useful:** The docsctl `publish-docs` job used job-level inputs and verified publish. The GoReleaser config had useful examples for `brews`, `nfpms`, checksum signing, and a safer Fury publisher that skips non-package artifacts.
- **What I didn't find useful:** Glazed builds from a normal root module command (`./cmd/glaze`), so it did not answer the nested module question directly.
- **What is out of date / what was wrong:** No direct issue found. It did show `brews` is now reported by GoReleaser as being phased out in favor of `homebrew_casks`; the ecosystem still uses `brews` today.
- **What would need updating:** At some point, package release configs should migrate away from deprecated `brews` if GoReleaser requires it. For now, it is a warning, not a blocker.

### 7. infra-tooling docsctl publishing playbook

- **Resource:** `/home/manuel/code/wesen/go-go-golems/infra-tooling/docs/go-go-golems/playbooks/docsctl-docs-publishing-rollout-playbook.md`
- **What I was researching:** The operational process and permission model for docsctl publishing to `docs.yolo.scapegoat.dev`.
- **What I was looking for in this document in particular:** Required GitHub permissions, Vault role names, Terraform workspace path, bound claims, validation commands, and release verification steps.
- **Why I chose it:** The user suspected this playbook existed in `infra-tooling`; it is the most authoritative operational source for docsctl rollout.
- **How I found the resource:** I searched `/home/manuel/code/wesen/go-go-golems/infra-tooling` for `docsctl`, `publish`, `release`, and `playbook`.
- **What I found useful:** It clearly states that docsctl publishing requires job-scoped `id-token: write`, a reusable workflow call, package version equal to the Git tag, and matching Vault roles named `docsctl-<package>-publisher`.
- **What I didn't find useful:** The local validation command assumes a normal root-module command path: `GOWORK=off go run ./cmd/<package> ...`. goja-text now uses a nested module at `cmd/goja-text`, so the export command had to become `cd cmd/goja-text && GOWORK=off go run . ...`.
- **What is out of date / what was wrong:** The playbook is current for normal command layouts but incomplete for committed generated xgoja nested modules.
- **What would need updating:** Add a nested-module example under the help export section, including `mkdir -p .docsctl && cd cmd/<package> && GOWORK=off go run . help export --format sqlite --output-path ../../.docsctl/help.sqlite`.

### 8. reusable docsctl GitHub Actions workflow

- **Resource:** `/home/manuel/code/wesen/go-go-golems/infra-tooling/.github/workflows/publish-docsctl.yml`
- **What I was researching:** What the reusable publish workflow actually requires and how it authenticates to Vault and docs-registry.
- **What I was looking for in this document in particular:** Inputs, permissions, Vault login settings, JWT minting path, and `docsctl publish` invocation.
- **Why I chose it:** The caller workflow in goja-text delegates all docs publishing to this reusable workflow; permissions must match what this workflow expects.
- **How I found the resource:** It is referenced by go-go-goja, Glazed, and the infra-tooling playbook.
- **What I found useful:** It confirmed that the reusable workflow itself declares `contents: read` and `id-token: write`, computes default role names, logs into Vault using GitHub OIDC, mints `identity/oidc/token/<role>`, then runs `docsctl publish` with a token file.
- **What I didn't find useful:** It cannot by itself solve caller-side Vault claim binding; that still requires Terraform roles for each package.
- **What is out of date / what was wrong:** No issue found. It matched the playbook.
- **What would need updating:** If package workflows commonly use nested modules, consider documenting that `export_command` is evaluated from the caller repository root and may include `cd cmd/<package>`.

### 9. Terraform Vault GitHub Actions configuration

- **Resource:** `/home/manuel/code/wesen/terraform/vault/github-actions/envs/k3s/main.tf`
- **What I was researching:** How docsctl publisher permissions are represented in Vault/Terraform and how to add goja-text.
- **What I was looking for in this document in particular:** Existing `local.docsctl_publishers`, role naming, policy capabilities, bound GitHub OIDC claims, and the correct place to add `goja-text`.
- **Why I chose it:** The docsctl release job cannot publish until Vault trusts the exact repository/workflow/tag claims and allows minting a package-specific docs-registry JWT.
- **How I found the resource:** The infra-tooling playbook named this Terraform workspace explicitly.
- **What I found useful:** The `docsctl_publishers` map made the change small: add `package_name`, `repository`, numeric `repository_id`, and `workflow_ref`; Terraform creates the OIDC role, policy, and JWT auth role.
- **What I didn't find useful:** The plan also contained unrelated pending Pinocchio BSR resources, so it was not immediately obvious whether a full apply would be safe. I first targeted only goja-text, then later applied the unrelated BSR resources after the user approved.
- **What is out of date / what was wrong:** `goja-text` was missing from `docsctl_publishers`. There was also existing unapplied Pinocchio BSR configuration drift.
- **What would need updating:** Commit history should preserve the Terraform change. Future docsctl packages need the same map entry and a clean post-apply plan.

### 10. goja-text release workflow

- **Resource:** `/home/manuel/workspaces/2026-06-02/goja-text/goja-text/.github/workflows/release.yaml`
- **What I was researching:** Whether goja-text already had docsctl publishing and whether the generated template needed enabling.
- **What I was looking for in this document in particular:** Disabled docsctl template job, placeholder package names, permissions, and release job dependencies.
- **Why I chose it:** The user asked whether docsctl should now be published because `cmd/goja-text/main.go` is committed.
- **How I found the resource:** I searched goja-text workflows for `docsctl`, `publish-docs`, `release`, and `goreleaser`.
- **What I found useful:** The workflow already contained a disabled docsctl template job with the right reusable workflow reference and input names.
- **What I didn't find useful:** It used placeholder `XXX` values and the normal `go run ./cmd/XXX` export pattern, which fails for goja-text because `cmd/goja-text` is a nested module.
- **What is out of date / what was wrong:** The disabled template was stale now that goja-text has a real generated command module. It also suggested enabling workflow-level OIDC in comments, whereas the current playbook prefers job-scoped OIDC permissions.
- **What would need updating:** Already updated for `goja-text`, job-scoped `id-token: write`, and the nested-module export command. If CI fails at tag time, inspect decoded claims in the publish job and compare them to Terraform `workflow_ref`/`job_workflow_ref`.

### 11. goja-text GoReleaser configuration

- **Resource:** `/home/manuel/workspaces/2026-06-02/goja-text/goja-text/.goreleaser.yaml`
- **What I was researching:** Whether goja-text should still skip release builds now that it has a committed generated CLI.
- **What I was looking for in this document in particular:** Existing skip settings, build definitions, package publishing, and whether `cmd/goja-text` can be built by GoReleaser.
- **Why I chose it:** GoReleaser must produce the binary before docs publishing runs after `goreleaser-merge`.
- **How I found the resource:** It was in goja-text root and showed up when searching release workflow/config files.
- **What I found useful:** The comments clearly explained the old assumption: goja-text was a library/provider repo and GoReleaser should skip builds because there was no checked-in command. That made the required update obvious.
- **What I didn't find useful:** The old config had no examples for packaging or Homebrew/Fury because it skipped builds.
- **What is out of date / what was wrong:** The assumption is now wrong: `cmd/goja-text/main.go` and its nested module are committed. GoReleaser should build and package `goja-text`.
- **What would need updating:** Already updated to build from `dir: cmd/goja-text`. Future improvement: confirm full Linux arm64 build in CI; local full snapshot requires `aarch64-linux-gnu-gcc`, which CI installs.

### 12. goja-text README and generated command docs

- **Resource:** `/home/manuel/workspaces/2026-06-02/goja-text/goja-text/README.md`
- **What I was researching:** How the project tells users to build and run the generated xgoja binary.
- **What I was looking for in this document in particular:** Build instructions that still referenced `go run ../go-go-goja/cmd/xgoja build` with an absolute `--xgoja-replace`.
- **Why I chose it:** The user requested a committed generated scaffold and `go tool` + `go generate` workflow; user-facing instructions needed to match.
- **How I found the resource:** README is the main project guide and was already being updated alongside command-local jsverbs.
- **What I found useful:** It already described module purpose, built-in help, demos, and root-mounted commands.
- **What I didn't find useful:** The old build section assumed a temporary generated workspace and a local go-go-goja checkout.
- **What is out of date / what was wrong:** The build instructions were stale once xgoja v0.7.4 was released and once `cmd/goja-text` became a committed nested module.
- **What would need updating:** Consider adding a short release/docs publishing section after the next successful tag proves docsctl publication end-to-end.

### 13. GoReleaser online documentation/search result

- **Resource:** Kagi search result pointing to `https://goreleaser.com/customization/builds/builders/go/`
- **What I was researching:** Whether GoReleaser supports building a Go binary from a nested module directory.
- **What I was looking for in this document in particular:** Confirmation of build options such as `dir` and `main` for nested module builds.
- **Why I chose it:** goja-text’s committed generated command lives in `cmd/goja-text` with its own `go.mod`, so normal root-module `main: ./cmd/goja-text` semantics are not sufficient.
- **How I found the resource:** I ran a Kagi search for `GoReleaser builds dir option main nested module`.
- **What I found useful:** The search result confirmed the relevant GoReleaser build customization area to check. The actual validation came from local `goreleaser check` and snapshot builds.
- **What I didn't find useful:** The search result itself was broad and did not directly solve the goja-text config. Local GoReleaser validation was more decisive.
- **What is out of date / what was wrong:** Not assessed deeply; this was a discovery pointer rather than a fully read reference.
- **What would need updating:** No project docs need to point to this unless we create a general nested-module GoReleaser playbook.

### 14. goja-text committed generated command module

- **Resources:**
  - `/home/manuel/workspaces/2026-06-02/goja-text/goja-text/cmd/goja-text/generate.go`
  - `/home/manuel/workspaces/2026-06-02/goja-text/goja-text/cmd/goja-text/go.mod`
  - `/home/manuel/workspaces/2026-06-02/goja-text/goja-text/cmd/goja-text/main.go`
  - `/home/manuel/workspaces/2026-06-02/goja-text/goja-text/cmd/goja-text/xgoja.yaml`
- **What I was researching:** How the generated command module should be regenerated, built, and exported for docsctl.
- **What I was looking for in this document/code in particular:** Whether the nested module can run `go tool xgoja`, regenerate files in place, build with released xgoja v0.7.4, and export Glazed help.
- **Why I chose it:** This is the actual binary package used by GoReleaser and docsctl.
- **How I found the resource:** It was created during the ticket after the user asked to commit generated xgoja scaffolding under `cmd/goja-text`.
- **What I found useful:** Keeping its own `go.mod` makes `GOWORK=off go run . help export ...` reliable in CI. The generated binary successfully exported an SQLite help database with 8 sections and 8 slugs.
- **What I didn't find useful:** Because it is a nested module, normal root commands like `GOWORK=off go run ./cmd/goja-text ...` fail with `main module ... does not contain package ...`. The export command must `cd cmd/goja-text` first.
- **What is out of date / what was wrong:** The initial generation directive used `--xgoja-replace ../../../go-go-goja`, which became unnecessary once xgoja v0.7.4 was published. That replace was removed.
- **What would need updating:** If goja-text itself is published and the generated command should consume the released module instead of `replace github.com/go-go-golems/goja-text => ../..`, revisit the nested module replace policy.

## Follow-up update checklist

- [ ] Add a nested-module export command example to the infra-tooling docsctl rollout playbook.
- [ ] After the first goja-text tag release, verify `https://docs.yolo.scapegoat.dev/api/packages` includes `goja-text` with the tag version.
- [ ] Watch the release workflow’s `Publish docs` job and compare decoded JWT claims if Vault login fails.
- [ ] Consider adding a first-class xgoja `generate` command for committed generated modules.
- [ ] Decide whether to migrate release configs away from GoReleaser `brews` before the deprecation becomes a hard failure.

## Validation evidence

Useful commands and outcomes observed during this investigation:

```bash
# xgoja feature tests
go test ./cmd/xgoja ./cmd/xgoja/internal/buildspec ./pkg/xgoja/app -count=1

# goja-text full validation after generated command and release wiring
make check
GOWORK=off goreleaser check
GOWORK=off goreleaser release --snapshot --clean --skip=publish,sign --single-target

# docsctl local export for nested generated module
mkdir -p .docsctl
cd cmd/goja-text
GOWORK=off go run . help export --format sqlite --output-path ../../.docsctl/help.sqlite
cd ../..
docsctl validate --file .docsctl/help.sqlite --package goja-text --version v0.0.0-local

# Terraform permission validation
cd /home/manuel/code/wesen/terraform
source .envrc
cd vault/github-actions/envs/k3s
terraform plan
terraform apply -auto-approve
terraform plan
```

Final Terraform state after applying goja-text docsctl and Pinocchio BSR resources reported:

```text
No changes. Your infrastructure matches the configuration.
```

## Related

- Design guide: `../design-doc/01-root-mounted-javascript-verbs-design-and-implementation-guide.md`
- Diary: `01-implementation-diary.md`
- Ticket index: `../index.md`
