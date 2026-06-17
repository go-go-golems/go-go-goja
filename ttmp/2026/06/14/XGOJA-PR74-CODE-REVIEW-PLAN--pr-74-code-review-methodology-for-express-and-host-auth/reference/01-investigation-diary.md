---
Title: Investigation diary
Ticket: XGOJA-PR74-CODE-REVIEW-PLAN
Status: active
Topics:
    - review
    - goja
    - xgoja
    - auth
    - security
    - testing
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../../code/wesen/go-go-golems/go-go-parc/Projects/2026/06/12/ARTICLE - go-go-goja Express Auth - Go Backed Fluent Route Plans.md
      Note: User-supplied article that framed the planned route design
    - Path: ../../../../../../../../../../code/wesen/go-go-golems/go-go-parc/Projects/2026/06/14/ARTICLE - go-go-goja Express Auth - From Planned Routes to Generated Host Auth.md
      Note: User-supplied article that framed generated-host auth review scope
    - Path: ttmp/2026/06/14/XGOJA-PR74-CODE-REVIEW-PLAN--pr-74-code-review-methodology-for-express-and-host-auth/scripts/01-pr74-inventory.sh
      Note: Ticket-local inventory script created to shape review planning
    - Path: ttmp/2026/06/14/XGOJA-PR74-CODE-REVIEW-PLAN--pr-74-code-review-methodology-for-express-and-host-auth/scripts/02-targeted-validation.sh
      Note: Ticket-local validation script used to capture targeted test evidence
    - Path: ttmp/2026/06/14/XGOJA-PR74-CODE-REVIEW-PLAN--pr-74-code-review-methodology-for-express-and-host-auth/scripts/03-verify-behaviors.go
      Note: ticket-local //go:build ignore harness used to confirm F1/N1 behaviors
ExternalSources:
    - https://github.com/go-go-golems/go-go-goja/pull/74
Summary: Chronological diary for creating the PR 74 code-review planning guide.
LastUpdated: 2026-06-14T20:50:00-04:00
WhatFor: Records how the review-planning ticket and guide were created, what evidence was gathered, what commands were run, and how to continue.
WhenToUse: Read before updating the review plan, running additional validation, or turning the plan into the actual code review.
---



# Diary

## Goal

This diary captures the creation of a docmgr ticket and an intern-oriented methodology guide for reviewing PR 74 in `go-go-goja`. It records the evidence gathered, validation commands run, and documentation artifacts produced without modifying product code.

## Step 1: Create review-planning ticket and gather evidence

I created a new ticket specifically for planning the PR 74 code review rather than writing the review itself. The key decision was to structure the deliverable around methodology and onboarding: first map the architecture and changed files, then explain which code paths, tests, examples, and security invariants a future reviewer should examine.

I gathered local evidence from the current branch, the PR metadata, the two supplied project articles, targeted package tests, and a runnable example smoke. The resulting design document is intentionally a guide for how to review the branch; it does not claim approval or enumerate final review findings.

### Prompt Context

**User prompt (verbatim):** "How would you approach a solid in depth code review of https://github.com/go-go-golems/go-go-goja/pull/74 which si the current state of ./go-go-goja/ (see also a write up of the work in /home/manuel/code/wesen/go-go-golems/go-go-parc/Projects/2026/06/12/ARTICLE - go-go-goja Express Auth - Go Backed Fluent Route Plans.md and ~/code/wesen/go-go-golems/go-go-parc/Projects/2026/06/14/ARTICLE - go-go-goja Express Auth - From Planned Routes to Generated Host Auth.md .

Create a new docmgr ticket for the code review, and then write a detailed Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.


The document is not the code review itself, it is _about doing the code review_, planning it, which methodology, which tests to do, what to examine in more depth, etc...

Do not modify any code, but are you welcome to look around and even write scripts and stuff in the scripts/ folder of the ticket, run experiments, all in the purpose of shaping the code review planning, not for the sake of review itself."

**Assistant interpretation:** Create a docmgr ticket, inspect PR 74 and related write-ups, write a detailed intern-friendly review methodology/design guide, store all artifacts in the ticket, validate the ticket, and upload the final bundle to reMarkable without modifying product code.

**Inferred user intent:** The user wants a careful review plan for a large security-sensitive PR so a reviewer or intern can understand the system, run the right tests, and perform the eventual code review systematically.

**Commit (code):** N/A — no product code was changed.

### What I did

- Created ticket `XGOJA-PR74-CODE-REVIEW-PLAN` with topics `review,goja,xgoja,auth,security,testing`.
- Added primary design doc `design/01-pr-74-code-review-methodology-and-intern-guide.md`.
- Added diary doc `reference/01-investigation-diary.md`.
- Read the supplied Obsidian/project articles:
  - `/home/manuel/code/wesen/go-go-golems/go-go-parc/Projects/2026/06/12/ARTICLE - go-go-goja Express Auth - Go Backed Fluent Route Plans.md`
  - `/home/manuel/code/wesen/go-go-golems/go-go-parc/Projects/2026/06/14/ARTICLE - go-go-goja Express Auth - From Planned Routes to Generated Host Auth.md`
- Queried PR metadata with `gh pr view 74` and confirmed local branch `task/goja-express-auth` at `b66baea`.
- Created `scripts/01-pr74-inventory.sh` and captured output in `sources/01-pr74-inventory.md`.
- Created `scripts/02-targeted-validation.sh` and captured output in `sources/02-targeted-validation.md`.
- Captured key source anchors in `sources/05-key-line-anchors.md`.
- Captured key test names in `sources/04-test-inventory.md`.
- Ran `make -C examples/xgoja/18-express-auth-host smoke` and captured output in `sources/03-express-auth-host-smoke.md`.
- Wrote the detailed review methodology guide with architecture diagrams, subsystem maps, pseudocode, review phases, test plan, intern onboarding order, and final review checklist.

### Why

- The PR is large and security-sensitive, so a useful review needs a structured methodology rather than an ad hoc scan.
- The user explicitly asked for a guide about doing the review, not for the review findings.
- Ticket-local scripts make the evidence reproducible without changing product code.
- Targeted tests and smoke output help shape the review plan around observed executable behavior.

### What worked

- `docmgr ticket create-ticket` and `docmgr doc add` created the ticket workspace successfully.
- `gh pr view 74` returned PR metadata and confirmed the PR is open.
- The inventory script captured the local branch/base/head and diff statistics.
- Targeted tests passed with `GOFLAGS=-buildvcs=false`:
  - `go test ./pkg/gojahttp ./modules/express ./pkg/xgoja/providers/http ./pkg/xgoja/hostauth -count=1`
  - `go test ./pkg/gojahttp/auth/... -count=1`
  - `go test ./examples/xgoja/18-express-auth-host/cmd/host ./examples/xgoja/20-express-hello-world/cmd/host ./examples/xgoja/21-generated-host-auth/cmd/host -count=1`
- `make -C examples/xgoja/18-express-auth-host smoke` passed and exercised public, authenticated, CSRF-denied, authorized, missing-resource, logout, and post-logout paths.

### What didn't work

- The GitHub PR metadata path list reported `fileCount: 100`, while the local `git diff origin/main...HEAD --stat` captured a much larger local diff (`186 files changed, 28570 insertions(+), 119 deletions(-)`). This appears to be a GitHub/CLI file-list cap or summary limitation, so the review plan explicitly says to use local `git diff` output for complete triage.
- I did not run the generated-host `make -C examples/xgoja/21-generated-host-auth smoke` because its Makefile regenerates tracked files. The guide marks it as an optional clean-worktree smoke.
- I did not run the Keycloak/Postgres smoke because it requires Docker services and free ports. The guide marks it as an optional heavier validation path.

### What I learned

- PR 74 should be reviewed by security boundary and runtime flow rather than by file order.
- The main implementation seam is `RoutePlan` plus `gojahttp.AuthOptions`: JavaScript declares intent and Go enforces authentication, CSRF, resource resolution, authorization, and audit.
- Generated-host auth review needs lifecycle attention: service factories should be discovered at command construction time but build stores only at command execution time.
- The examples are not incidental; they are executable migration and integration specifications for a breaking route API change.

### What was tricky to build

- The hardest part was keeping the deliverable clearly scoped as a review methodology rather than drifting into a premature review. The branch has many obvious review angles, but the document needed to phrase them as risk areas, questions, and validation steps instead of final findings.
- Another tricky point was the file-count discrepancy between GitHub PR metadata and local diff output. I handled this by documenting the discrepancy in the guide and using the local diff as the authoritative planning input.
- The generated-host example smoke required caution because its `Makefile` regenerates committed runtime files. To honor the user's instruction not to modify code, I did not run that smoke in this pass and instead documented how a reviewer should run it from a clean worktree.

### What warrants a second pair of eyes

- The guide should be checked for whether it is sufficiently neutral and does not accidentally imply review conclusions.
- The list of mandatory versus optional tests should be reviewed by someone who knows which smokes are practical in CI and which require local infrastructure.
- The file map and review phases should be checked against any late PR changes after `b66baea`.

### What should be done in the future

- If the PR changes, rerun `scripts/01-pr74-inventory.sh`, `scripts/02-targeted-validation.sh`, and update the design guide's branch/head/base facts.
- Run optional generated-host and Keycloak/Postgres smokes in a clean environment before writing the final code review.
- Turn the guide's checklist into the actual review report only after line-by-line evidence is collected.

### Code review instructions

- Start with `design/01-pr-74-code-review-methodology-and-intern-guide.md` for methodology.
- Check captured evidence under `sources/` before rerunning commands.
- Validate current branch assumptions with:
  - `git status --short`
  - `git branch --show-current`
  - `git diff --stat origin/main...HEAD`
- Rerun targeted validation with:
  - `ttmp/2026/06/14/XGOJA-PR74-CODE-REVIEW-PLAN--pr-74-code-review-methodology-for-express-and-host-auth/scripts/02-targeted-validation.sh`

### Technical details

Ticket path:

```text
/home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/14/XGOJA-PR74-CODE-REVIEW-PLAN--pr-74-code-review-methodology-for-express-and-host-auth
```

Primary design doc:

```text
/home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/14/XGOJA-PR74-CODE-REVIEW-PLAN--pr-74-code-review-methodology-for-express-and-host-auth/design/01-pr-74-code-review-methodology-and-intern-guide.md
```

## Step 2: Conduct the actual PR 74 code review and write the report

Step 1 produced a methodology guide. In Step 2 I executed that methodology against the branch and wrote the review report itself (`design-doc/02-pr-74-code-review-report.md`). I traced every denial path in the dispatch pipeline, checked all ten security invariants, reviewed the generated-host lifecycle and OIDC adapter statically, wrote a ticket-local verification harness to confirm ambiguous behaviors, and recorded line-anchored findings. The result is an "approve with non-blocking follow-ups" recommendation with no blocking or security-bypass issues.

### Prompt Context

**User prompt (verbatim):** "Working on @<this ticket> and reading @<design/01-...guide>. Go through the code review carefully and write a detailed code review report. You can compile and run and write scripts (store in the scripts/ folder of the ticket), keeping a detailed diary as you work." (followed by "continue", and a later note: "we have a colleague working on the same code review in a separate ticket, so don't read their work if you come across it.")

**Assistant interpretation:** Perform the actual PR 74 review following the Step-1 methodology, compiling/running code and writing ticket-local scripts for evidence, then deliver a detailed review report in the ticket while maintaining the diary. Avoid reading the colleague's separate review ticket.

**Inferred user intent:** Convert the planning ticket into a real, evidence-based code review deliverable without touching product code or contaminating findings with the colleague's parallel work.

**Commit (code):** N/A — no product code was changed; only ticket docs and a `//go:build ignore` harness under `scripts/`.

### What I did

- Established Phase 0 facts: branch `task/goja-express-auth` @ `b66baea`, base `origin/main` @ `d406577`; only working-tree change is this ticket dir.
- Read the core product files end-to-end across all review phases: `auth_plan.go`, `planned_dispatch.go`, `host.go`, `route_registry.go`, `modules/express/auth_builders.go`, `express.go`, `hostauth/{config,builder,stores,resolve,lookup,services}.go`, `providers/http/{serve,http}.go`, `sessionauth/sessionauth.go`, `appauth/appauth.go`, `audit/audit.go`, `capability/capability.go`, `keycloakauth/keycloakauth.go`, `devauth/devauth.go`, and the SQL stores (`sessionauth/sqlstore`, `appauth/sqlstore`, `capability/sqlstore`) including schemas.
- Read the test suites for the same packages to map coverage and gaps.
- Ran `go vet` (clean) and the targeted `go test` set for `pkg/gojahttp`, `modules/express`, `pkg/xgoja/providers/http`, `pkg/xgoja/hostauth`, and `pkg/gojahttp/auth/...` (all PASS).
- Ran the `examples/xgoja/18-express-auth-host -smoke` in-process smoke (PASS; exercised 401/403/404/200/204 + audit sequencing).
- Wrote `scripts/03-verify-behaviors.go` (`//go:build ignore`) to confirm four ambiguous behaviors: (A) authenticator `(nil,nil)` mapping, (B) dev vs production error leakage, (C) audit `RedactMap` over-redaction, (D) capability token hashing + single-use redemption.
- Confirmed all ten Phase-3 security invariants and traced the full denial-path status table in `buildSecureEnvelope`/`servePlannedRoute`.
- Statically reviewed the OIDC adapter (PKCE, single-use state, nonce, verifier, open-redirect guard) and the generated-host lifecycle (lazy construction, shared DB handles, LIFO close, hot-reload service sharing).
- Wrote the review report at `design-doc/02-pr-74-code-review-report.md` with scope/environment, architecture confirmation, three non-blocking findings (F1–F3), three nits (N1–N3), the security/lifecycle/store/doc notes, coverage gaps, and an "approve with non-blocking follow-ups" recommendation.

### Why

- The ticket's deliverable in Step 1 was *methodology*; the user now wanted the actual review, evidence-based and line-anchored as the guide demanded.
- A verification harness was necessary because three behaviors were ambiguous from reading alone (nil-actor status mapping, dev/prod error leakage boundary, and exactly which audit attributes get redacted).
- Writing the harness as `//go:build ignore` under `scripts/` keeps it reproducible without ever becoming part of the build or modifying product code.

### What worked

- `go vet` clean and all targeted `go test` packages PASS on the first run; no flakiness.
- Example 18 smoke ran in-process with `-smoke` and gave a clear PASS JSON, exercising the full denial matrix.
- The verification harness compiled after two small fixes (multi-return `plannedRuntime` and a `Now` func type) and immediately confirmed all four behaviors, turning three suspected issues into concrete evidence (one real finding F1, one confirmed-correct-but-untested N1).
- Reading the packages by runtime/security boundary (per the methodology) made the denial-path table and the lifecycle claims easy to verify against the actual code.

### What didn't work

- `docmgr doc add` created **two** duplicate stub docs (`design-doc/01-...` and `02-...`) because the first invocation's output was truncated in my view and I re-ran it. I deleted the `01-` stub and wrote the report into `02-`. Lesson: check `find`/`ls` after `docmgr doc add` before re-invoking.
- The first `go run` of the harness failed to compile (`assignment mismatch` and `func()` vs `func() time.Time`). Fixed in two `edit` passes; cost was small but avoidable by type-checking small harness snippets mentally before writing the whole file.
- The Keycloak (`19`) and generated-host (`21`) smokes were skipped (Docker/ports and committed-file regeneration respectively), so those paths were reviewed statically only.

### What I learned

- The PR's security posture is strong: the trust boundary is explicit, every nil service dependency fails closed to 500, the staged builder makes it impossible to register a handler without a security choice, and the OIDC adapter implements the full state/nonce/PKCE/verifier set correctly.
- The one genuine functional defect (F1, audit over-redaction) is not a leak — it is the opposite: the substring redaction in `audit.secretKey` is so broad (matches `"capability"`, `"session"`, `"token"`, `"code"`) that it erases non-secret identifiers like `capabilityId` that the capability package deliberately emits for forensics.
- `appauth` SQL `UpsertFromOIDC` is inconsistent with its own file: `AddUser` uses the `ON CONFLICT ... DO UPDATE` upsert, but the OIDC first-login create path uses a plain `INSERT` despite `keycloak_sub` being `UNIQUE` (F2, a low-probability concurrency race in the login path).
- `keycloakauth`'s package doc *understates* how good it is: it says tokens are "kept server-side" but they are actually discarded entirely after deriving the app session (F3, a doc-accuracy nit).
- `sql.Open` laziness is real but acceptable: `ApplySchema` is the first real I/O, so a bad DSN with `apply-schema: true` fails fast at startup.

### What was tricky to build

- **Cause/symptom for F1:** the redaction is a defensive-in-depth feature whose intent is to scrub secrets from audit attributes, but the implementation uses `strings.Contains(lower(key), fragment)` over a broad fragment list. The symptom is that the capability package's own `record()` puts a non-secret `capabilityId` into attributes, and that identifier gets blanked. I approached the fix by separating "raw secret values" (which must be redacted) from "identifiers that reference a secret concept" (which should survive), and suggested either a precise deny-list of exact key names or an exemption for `*Id`/`*Type`/`*Name` suffixes.
- **Cause/symptom for F2:** the SELECT-then-plain-INSERT pattern. Under READ COMMITTED two concurrent first-logins for the same new `keycloak_sub` both miss, both INSERT, and the loser surfaces a raw unique-constraint error. The fix is to reuse the existing upsert query for the create branch (keyed on the `keycloak_sub` UNIQUE constraint) and keep the transaction.
- **Keeping the review evidence-based and neutral:** the methodology warned against premature conclusions, so every finding is anchored to a file/line and backed by either a test name or the harness output rather than impression.

### What warrants a second pair of eyes

- The `RedactMap` fix design (F1): deciding the exact redaction policy (deny-list vs suffix-exemption vs opt-in marker) is a small security-policy choice that should be reviewed by whoever owns audit semantics.
- The `UpsertFromOIDC` upsert-on-`keycloak_sub` change (F2): confirm the desired identity of the row when two OIDC subjects race (both should resolve to one user row; decide whether `id` is derived from `sub` as `"user:"+sub`).
- The two skipped smokes (`19` Keycloak/Postgres, `21` generated-host) should be run by someone with the right environment before merge to close that gap.
- TypeScript declaration alignment (`modules/express/typescript.go`) was only spot-checked in this pass; a focused diff against the staged builder method set is worth doing.

### What should be done in the future

- After the author addresses F1/F2 and adds the N1 test, re-run `scripts/03-verify-behaviors.go` and the targeted `go test` set to confirm the fixes.
- Run the `21` generated-host smoke from a clean worktree and the `19` Keycloak Docker smoke.
- If the PR is rebased, rerun `scripts/01-pr74-inventory.sh` and refresh the branch/head/base facts in both the methodology guide and the report.

### Code review instructions

- Read the report first: `design-doc/02-pr-74-code-review-report.md` (scope, findings table, security notes, merge recommendation).
- Read the methodology for *why* each area was examined: `design/01-pr-74-code-review-methodology-and-intern-guide.md`.
- Reproduce the behavior evidence: `GOFLAGS=-buildvcs=false go run ./ttmp/2026/06/14/XGOJA-PR74-CODE-REVIEW-PLAN--pr-74-code-review-methodology-for-express-and-host-auth/scripts/03-verify-behaviors.go`.
- Reproduce the test/smoke baseline: the exact commands are in the report's "Commands run" section.
- Findings to action, in priority order: F1 (audit redaction), F2 (OIDC upsert race), N1 (nil-actor dispatch test), F3 (keycloakauth doc).

### Technical details

Verification harness key outputs (`scripts/03-verify-behaviors.go`):

```text
behavior A (nil,nil authenticator)  -> status=401 body="Unauthorized\n"
behavior B (dev=true generic err)   -> status=500 body leaks internals: true
behavior B2 (dev=false generic err) -> status=500 body="Internal Server Error\n" leaks: false
behavior C (audit RedactMap)        -> capabilityId="[REDACTED]" sessionId="[REDACTED]" (over-redaction)
behavior D (capability)             -> raw token returned once; TokenHash redacted; 2nd redeem -> ErrUsed
```

Key file anchors (current branch):

```text
pkg/gojahttp/auth_plan.go              ValidateRoutePlan (security-mode + action + param validation)
pkg/gojahttp/planned_dispatch.go       buildSecureEnvelope (denial matrix) + servePlannedRoute (audit outcomes)
modules/express/auth_builders.go       staged builder + trusted-object sync.Map + copy-on-auth
pkg/gojahttp/auth/audit/audit.go       secretKey() -> F1 lives here
pkg/gojahttp/auth/appauth/sqlstore/sqlstore.go  UpsertFromOIDC -> F2 lives here
pkg/gojahttp/auth/keycloakauth/keycloakauth.go  handleCallback (OIDC) -> F3 doc
pkg/xgoja/hostauth/builder.go          BuildHostAuthServices (lazy, failure-closing)
pkg/xgoja/providers/http/serve.go     buildServeAuthServices + hot-reload service sharing
```

Report path: `design-doc/02-pr-74-code-review-report.md`.
