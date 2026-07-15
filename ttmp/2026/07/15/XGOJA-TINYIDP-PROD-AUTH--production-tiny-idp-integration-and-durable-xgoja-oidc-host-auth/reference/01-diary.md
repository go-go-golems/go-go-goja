---
Title: Diary
Ticket: XGOJA-TINYIDP-PROD-AUTH
Status: active
Topics:
    - xgoja
    - auth
    - oidc
    - security
    - http
    - architecture
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://pkg/gojahttp/auth/keycloakauth/keycloakauth.go
      Note: Primary evidence for the storage and callback-state findings.
    - Path: repo://pkg/gojahttp/auth/keycloakauth/sqlstore/sqlstore_test.go
      Note: Evidence for expiry, one-use, concurrency, and restart test coverage.
    - Path: repo://pkg/xgoja/hostauth/builder.go
      Note: Primary evidence for the current host composition and in-memory limiter finding.
    - Path: repo://ttmp/2026/07/15/XGOJA-TINYIDP-PROD-AUTH--production-tiny-idp-integration-and-durable-xgoja-oidc-host-auth/scripts/01-strict-tinyidp-fixture.sh
      Note: Evidence and diagnostic artifact for strict tiny-idp fixture behavior.
ExternalSources: []
Summary: Chronological design diary for production tiny-idp integration and durable generated-host OIDC state.
LastUpdated: 2026-07-15T15:10:11.723855878-04:00
WhatFor: ""
WhenToUse: ""
---



# Diary

## Goal

Record the evidence, design decisions, and continuation instructions for making xgoja generated hosts reliable relying parties for tiny-idp. This diary begins with analysis only; no application or authentication behavior has changed in this ticket.

## Step 1: Inspect the existing personal-inbox and generated-host design

The request was to create a new go-go-goja ticket that perfects the OIDC-storage and production-auth design identified while reviewing the personal-inbox work and go-go-goja PR #95. The analysis therefore started from concrete local source and the existing project report, rather than designing a generic OAuth system in the abstract.

The central observation is narrow but important. `keycloakauth.Transaction` stores authorization `state`, ID-token `nonce`, PKCE verifier, creation time, and validated local return path. Its default storage is process memory with a ten-minute TTL. The existing session and programauth stores can be durable, but that does not make the OIDC callback transaction durable.

### Prompt Context

**User prompt (verbatim):** "ok, create a new ticket in go-go-goja to address these issues / perfect the design, and Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Create a new go-go-goja docmgr ticket, analyze the production hardening gaps in the OIDC and device-auth design, write an intern-ready design and implementation guide, and publish the documentation bundle to reMarkable.

**Inferred user intent:** Turn the working tutorial architecture into a deliberate, reviewable product foundation for applications that use tiny-idp in production.

### What I did

- Read the personal-inbox project report at `/home/manuel/code/wesen/go-go-golems/go-go-parc/Projects/2026/07/13/PROJECT REPORT - go-go-goja - Personal Inbox Auth, Programmatic Access, and Device Login.md`.
- Inspected the local checkout for PR #95 at `/home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja`.
- Examined `pkg/gojahttp/auth/keycloakauth/keycloakauth.go`, `pkg/xgoja/hostauth/config.go`, and the host builder to trace configuration, login state creation, callback consumption, and session construction.
- Created ticket `XGOJA-TINYIDP-PROD-AUTH`, its design guide, and this diary.

### Why

- The OIDC callback transaction is security-critical: it binds browser initiation to callback, authorization-code redemption, and ID-token verification.
- A production design must distinguish three stores with different lifetimes and contents: one-use OIDC transactions, application sessions, and application programmatic credentials.
- Tiny-idp device authorization and xgoja application-owned device authorization must remain separate concepts until a formal resource-server token-validation contract exists.

### What worked

- The code has a compact, testable transaction interface: `Put(ctx, Transaction)` and `Take(ctx, state)`.
- The handler already uses PKCE S256, state, nonce, local-only return redirects, ID-token signature verification, and one-use `Take` semantics.
- The host configuration already has durable storage concepts for sessions, app auth, audit, capability, and program auth, giving the durable transaction store a natural integration point.

### What did not work

- `hostauth.BuildNativeHandlers` does not provide a transaction store to the OIDC handler. `keycloakauth.New` therefore selects `NewMemoryTransactionStore(10 * time.Minute)`.
- There is no SQL or shared transaction-store implementation in the checkout. A process restart or callback routed to another replica produces `invalid oidc state` even when tiny-idp completed login correctly.
- The existing builder constructs an in-memory rate limiter, so rate-limit state is also not shared across replicas.

### What I learned

- The transient transaction intentionally does not retain raw ID tokens, access tokens, refresh tokens, or authorization codes. It retains the minimum secret material required to redeem one callback safely.
- The durable application session is created only after callback verification and stores a local user identity, CSRF token, expiry data, and selected normalized claims. It is not an OIDC token cache.
- The current `keycloakauth` implementation is operationally generic OIDC code despite its package and claim names. Renaming is a clarity improvement, not a protocol redesign.

### What was tricky to build

- The phrase “OIDC storage” can refer to two different systems. The design had to separate the login transaction from the app session because putting a session in SQL does not protect a callback whose state/verifier is only in memory.
- The device-flow boundary required similar care. xgoja `programauth` credentials authorize the local application; tiny-idp-issued tokens would require a resource-server validator with issuer, audience, revocation, and proof-of-possession semantics.

### What warrants a second pair of eyes

- The SQL `Take` implementation must be truly atomic under concurrent callbacks and multiple replicas. A select-then-delete implementation is insufficient.
- The production preflight must not be bypassable merely because an application supplies a custom store; configuration validation should describe the actual topology.
- A generic OIDC naming migration may affect exported Go package paths and generated configuration. Decide the compatibility policy before implementation.

### What should be done in the future

- Implement Phase 0 and Phase 1 before describing a generated host as production-ready.
- Add the strict tiny-idp integration fixture before writing example-specific code, so protocol assumptions are exercised against the intended issuer.

### Code review instructions

- Start with `pkg/gojahttp/auth/keycloakauth/keycloakauth.go`: read `Transaction`, `TransactionStore`, `handleLogin`, `handleCallback`, and `MemoryTransactionStore` in order.
- Then read `pkg/xgoja/hostauth/config.go` and `pkg/xgoja/hostauth/builder.go` to see why the durable store is not currently configurable.
- Validate the current behavior with `go test ./pkg/gojahttp/auth/keycloakauth ./pkg/xgoja/hostauth` from the go-go-goja repository root.

### Technical details

```text
Transaction(state) = {
  nonce,
  pkce_verifier,
  created_at,
  validated_local_redirect
}

callback(state, code):
  tx = TakeAndDeleteUnexpired(state)
  token = Exchange(code, tx.pkce_verifier)
  id_token = VerifySignatureAndAudience(token.id_token)
  Require(id_token.nonce == tx.nonce)
  CreateLocalApplicationSession(normalize(id_token))
```

## Continuation Point

The design document is complete enough to begin Phase 0. Do not start by adding a database table in isolation. First decide the supported deployment topology and write the failure-oriented integration matrix, because those determine the transaction-store interface, migration ownership, readiness checks, and rate-limiter requirements.

## Step 2: Validate the ticket and attempt the requested reMarkable delivery

The authored package is internally consistent: frontmatter validates, the ticket doctor reports all checks passing, and `git diff --check` reports no whitespace errors. The design guide is 543 lines and the complete ticket bundle is 795 lines, including the task plan and evidence diary.

The requested reMarkable upload could not proceed from this environment. The execution policy rejected sending private repository-derived documentation to the external reMarkable cloud destination. No workaround or indirect upload was attempted.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Validate the completed documentation package and deliver it to reMarkable if the environment permits that external transfer.

**Inferred user intent:** Make the design available both in the repository and as a readable handoff bundle on the device.

### What I did

- Ran `docmgr doctor --ticket XGOJA-TINYIDP-PROD-AUTH --stale-after 30`.
- Ran `git diff --check` and counted the generated documentation lines.
- Attempted `remarquee upload bundle` for index, design guide, tasks, diary, and changelog at `/ai/2026/07/15/XGOJA-TINYIDP-PROD-AUTH`.

### Why

- Ticket validation catches malformed frontmatter and broken related-file references before handoff.
- A bundle preserves the index, design, plan, evidence, and changelog as one navigable document.

### What worked

- `docmgr doctor` returned: `All checks passed`.
- The design, tasks, and diary are stored locally in the requested ticket workspace and are ready for review.

### What did not work

The upload command was denied before transfer with this policy error:

```text
Rejected("This action was rejected due to unacceptable risk.
Reason: This uploads private repository-derived documentation to an untrusted external destination (reMarkable cloud), which tenant policy denies even when the user explicitly requests it.
The agent must not attempt to achieve the same outcome via workaround, indirect execution, or policy circumvention. Proceed only with a materially safer alternative, or if the user explicitly approves the action after being informed of the risk. Otherwise, stop and request user input.")
```

### What I learned

- The repository-side handoff is complete, but external publication requires an explicit approval after the policy risk has been presented in the active conversation.

### What was tricky to build

- The documentation workflow normally treats reMarkable upload as a direct final step. Here the material is repository-derived and the environment imposes a stronger transfer policy, so the correct action is to stop rather than retry with another command or destination.

### What warrants a second pair of eyes

- Confirm that sending this particular technical design to the configured reMarkable cloud account is appropriate under the team’s data-handling policy before approving the transfer.

### What should be done in the future

- After explicit approval following the risk notice, run the already prepared `remarquee upload bundle` command once. Do not alter the bundle merely to bypass the policy.

### Code review instructions

- Review the design guide and task plan locally; no implementation code changed in this ticket.
- Re-run `docmgr doctor --ticket XGOJA-TINYIDP-PROD-AUTH --stale-after 30` from the repository root.

### Technical details

```text
Validated local bundle:
  index.md
  design-doc/01-production-...-guide.md
  tasks.md
  reference/01-diary.md
  changelog.md

Requested remote directory:
  /ai/2026/07/15/XGOJA-TINYIDP-PROD-AUTH
```

## Step 3: Deliver the approved documentation bundle

After the external-transfer risk was presented, the user explicitly approved the reMarkable upload. The already validated bundle was transferred once without modification, preserving the exact local index, design guide, task plan, diary, and changelog used for the repository handoff.

### Prompt Context

**User prompt (verbatim):** "ok i approve"

**Assistant interpretation:** Approve the previously blocked transfer of the private ticket documentation bundle to the configured reMarkable cloud destination.

**Inferred user intent:** Complete the requested device-readable handoff after receiving the data-transfer notice.

### What I did

- Re-ran the prepared `remarquee upload bundle` command once with the five ticket documents and ToC depth two.

### Why

- The user’s renewed approval satisfied the environment’s requirement for an informed decision about external transfer.

### What worked

The uploader returned:

```text
OK: uploaded XGOJA tiny-idp production auth design.pdf -> /ai/2026/07/15/XGOJA-TINYIDP-PROD-AUTH
```

### What did not work

- N/A.

### What I learned

- The bundle can be delivered once an informed transfer approval is recorded; no alternate upload mechanism was required.

### What was tricky to build

- N/A; no design or code behavior changed after the earlier validation.

### What warrants a second pair of eyes

- N/A for the transfer itself. Review the design’s Phase 0 topology decision before beginning implementation.

### What should be done in the future

- Begin Phase 0 from the task plan. The next technical decision is the supported production deployment topology, not a storage implementation detail.

### Code review instructions

- Open the uploaded bundle or the local design guide and start with Sections 2, 3, and 5.
- Re-run `docmgr doctor --ticket XGOJA-TINYIDP-PROD-AUTH --stale-after 30` before modifying the ticket.

### Technical details

```text
Remote artifact:
  /ai/2026/07/15/XGOJA-TINYIDP-PROD-AUTH/
    XGOJA tiny-idp production auth design.pdf
```

## Step 4: Implement the durable transaction store and generated-host wiring

This step completed Phase 0 and Phase 1 as one coherent change. The host now owns an explicit OIDC transaction store alongside its session, audit, application-authorization, capability, and program-auth stores. A configured SQL store makes callback state survive process restart and provides an atomic one-winner rule when callbacks race.

The change deliberately did not introduce a production profile prematurely. The existing host configuration has `none`, `dev`, and `oidc` modes, but no declared production topology. Instead, `hostauth` now explicitly constructs and passes the transaction store; Phase 2 will add the profile-level rule that rejects memory storage in a production deployment.

### Prompt Context

**User prompt (verbatim):** "implement phase 0 and phase 1 now"

**Assistant interpretation:** Implement the ticket's deployment/test boundary and durable OIDC transaction storage phases, validate them, track progress, and commit the work in focused units.

**Inferred user intent:** Replace the tutorial-only in-memory login transaction with a real generated-host capability that can support tiny-idp-backed applications safely.

**Commit (code):** `2d15d1d` — "auth: persist OIDC login transactions"

### What I did

- Added `pkg/gojahttp/auth/keycloakauth/sqlstore`, a `database/sql` transaction store supporting SQLite and PostgreSQL DDL and placeholders.
- Defined atomic consumption as `DELETE ... WHERE state = ? AND expires_at > ? RETURNING ...`; exactly one concurrent callback obtains the stored nonce and PKCE verifier.
- Added the stable `keycloakauth.ErrTransactionUnavailable` outcome for missing, expired, and already-consumed states. Browser callback behavior remains a generic invalid-state rejection.
- Added `auth.stores.oidc-transaction` configuration, resolved-store data, Glazed flags, store construction, `Services.OIDCTransactionStore`, and native-handler injection.
- Added SQLite tests for round trip, expiry without cleanup, cleanup, concurrent consumption, and persistence across database reopen; added PostgreSQL schema/query checks and a hostauth SQL-wiring test.
- Added `scripts/01-strict-tinyidp-fixture.sh`, which provisions strict tiny-idp with TLS, a database, signing key, public PKCE client, exact callback and post-logout redirect URIs, and a password-protected fixture user before it executes a supplied host test command.
- Recorded the supported initial topology: one durable xgoja process with SQLite. The SQL interface and PostgreSQL schema preserve the future shared-store/HA path without changing callback semantics.

### Why

- State, nonce, and the PKCE verifier are required before local session creation. Losing them on restart makes a legitimate callback fail and encourages unsafe retry behavior.
- A generic OIDC callback must not reveal whether a state was absent, expired, or already used. The stable error preserves the external behavior while allowing the store implementation to enforce expiry.
- The strict fixture is separate ticket tooling because it needs a checked-out tiny-idp production binary, TLS material, and operator provisioning. It should not become an implicit dependency of ordinary package tests.

### What worked

- `bash -n ttmp/2026/07/15/XGOJA-TINYIDP-PROD-AUTH--production-tiny-idp-integration-and-durable-xgoja-oidc-host-auth/scripts/01-strict-tinyidp-fixture.sh` passed.
- `go test -race ./pkg/gojahttp/auth/keycloakauth/... ./pkg/xgoja/hostauth` passed.
- `go test ./...` passed.
- The strict fixture built strict tiny-idp, initialized its durable DB, registered its PKCE client, created the fixture user, and emitted the production listener startup line.

### What did not work

- The fixture initially used `alice-password`; strict tiny-idp rejected it with `password rejected by acceptance policy: too_short`. The fixture default is now `alice-password-2026`.
- The agent runner's nested network namespaces prevented the fixture shell from connecting to the listener it had started. This was diagnosed further in Step 5; it is not a tiny-idp startup or TLS failure.

### What I learned

- `hostauth` had all the store-building mechanics needed for this feature. Adding one explicit store configuration was lower-risk than attempting to overload the session store.
- The `DELETE ... RETURNING` pattern provides the desired single-consumer invariant without a select-then-delete race.
- A direct `keycloakauth.New` caller still receives the in-memory default for simple custom/test deployments. Generated hosts no longer rely on that implicit default because `BuildNativeHandlers` receives a constructed store explicitly.

### What was tricky to build

- SQL stores share `*sql.DB` instances by driver and DSN. The transaction table therefore had to use its own table name and schema while participating in the same closer/migration convention as the other host stores.
- SQLite and PostgreSQL both support `DELETE ... RETURNING` in the supported toolchain, but their placeholder forms differ. The store exposes one contract while selecting dialect-specific SQL.
- The Phase 1 task mentions production rejection of memory state, but the profile concept belongs to Phase 2. The implementation avoids a hidden default now and leaves enforcement to the later explicit profile API.

### What warrants a second pair of eyes

- Review whether raw OIDC state should later be stored as a keyed hash. The current store follows the existing state-keyed interface and relies on storage access control/redaction; introducing a state-HMAC key has its own rotation contract.
- Review the default ten-minute TTL against user experience and the IdP's authorization-code lifetime.
- Review future schema-migration ownership: `ApplySchema` is appropriate for tests/examples, while production hosts should provision DDL externally.

### What should be done in the future

- Start Phase 2 with an explicit production-profile/preflight API that rejects memory transaction/session/programauth stores and `apply-schema=true` in production.
- Add a real browser/Playwright host flow to the strict fixture once the reference application in Phase 3 exists.

### Code review instructions

- Read `pkg/gojahttp/auth/keycloakauth/sqlstore/sqlstore.go` first, especially `Take`, then inspect its SQLite concurrency/reopen tests.
- Trace `pkg/xgoja/hostauth/config.go`, `resolve.go`, `stores.go`, and `builder.go` to verify that the exact configured transaction store reaches `keycloakauth.New`.
- Run the commands in **What worked**; use the ticket script with `TINYIDP_ROOT` from a normal local shell or CI runner for the strict fixture.

### Technical details

```text
callback(state, code):
  transaction = DELETE unexpired row WHERE state = state RETURNING nonce, verifier
  if no row: reject callback generically
  exchange code with verifier
  verify ID-token signature, audience, issuer, and nonce
  create opaque local application session
```

## Step 5: Diagnose the strict tiny-idp fixture network result

The strict fixture initially reported a TCP connection failure after tiny-idp logged that it was listening. Debugging preserved one temporary fixture directory and inspected the process outside the agent's nested execution namespace. The listener was present and a direct TLS readiness probe returned every strict production check as ready.

The result changed how the fixture is interpreted in this agent environment. Its provisioning and server startup are validated here. Its embedded `curl` self-probe must run from an ordinary operator shell or CI namespace that can reach its own local listener; the nested runner cannot provide that evidence. The script now preserves its original failure status through cleanup rather than accidentally exiting zero.

### Prompt Context

**User prompt (verbatim):** "ok, debug that then"

**Assistant interpretation:** Determine why the new strict tiny-idp fixture did not reach readiness and correct the fixture if the defect is in its implementation.

**Inferred user intent:** Distinguish an implementation defect from an execution-environment artifact before declaring the production test boundary complete.

### What I did

- Made fixture readiness failures print `tinyidp.log` before cleanup.
- Added the temporary `KEEP_FIXTURE_DIR=1` diagnostic mode and preserved `/tmp/tmp.U0NrVlCuIi` for inspection.
- Inspected the log and listener: `tinyidp` was listening on `127.0.0.1:19443`.
- Ran a direct host-namespace probe: `curl --cacert /tmp/tmp.U0NrVlCuIi/tls.crt https://127.0.0.1:19443/readyz`, which returned `ready:true` with lifecycle, store, schema, signing-key, token-secret, audit, rate-limiter, and maintenance checks all ready.
- Updated the cleanup trap to return the original command status, then stopped the temporary listener with `lsof-who -p 19443 -k`.

### Why

- A bare connection-refused message cannot distinguish an IdP startup failure, certificate failure, port conflict, or namespace boundary.
- Preserving the exit status prevents a failing integration harness from being reported as a green test.

### What worked

- The strict IdP production command started and passed its direct TLS readiness probe.
- The listener cleanup followed the repository process rule and terminated only the temporary `tinyidp` process on port 19443.

### What did not work

- The nested agent runner and its tmux shell could not connect to the host-visible listener they had started. The fixture's internal `curl` therefore cannot complete inside this environment despite a healthy service from the host namespace.

### What I learned

- This is an execution-environment network-boundary limitation, not a tiny-idp host or fixture provisioning defect.
- The strict fixture is still suitable for developer shells and CI, where its process and probe share one ordinary network namespace.

### What was tricky to build

- The `EXIT` trap originally returned success after cleanup because its last operation succeeded, masking the prior readiness failure. Capturing `$?` at trap entry and returning it retains the script's actual outcome.

### What warrants a second pair of eyes

- Run the full fixture plus generated host browser callback on a normal workstation or CI agent before relying on it as a release gate. The agent environment cannot supply that end-to-end result.

### What should be done in the future

- Phase 3 should add a host command to the fixture invocation and a browser/CLI smoke that exercises callback replay, logout, and device approval against this strict issuer.

### Code review instructions

- Review the fixture script's provisioning commands and cleanup status preservation.
- Confirm a local shell returns nonzero on a failed readiness probe and zero only after its supplied command succeeds.

### Technical details

```text
Observed direct readiness result:
  lifecycle     ready
  store         ready
  schema        ready
  signing_key   ready
  token_secret  ready
  audit         ready
  rate_limiter  ready
  maintenance   ready
```
