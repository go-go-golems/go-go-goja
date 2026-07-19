---
Title: Integration implementation diary
Ticket: XGOJA-PR98-INTEGRATION-2026-07-18
Status: active
Topics:
    - auth
    - oidc
    - security
    - testing
    - xgoja
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://pkg/gojahttp/auth/appauth/sqlstore/sqlstore.go
      Note: Transactional issuer identity binding and disablement in merge fdfa96d
    - Path: repo://pkg/gojahttp/auth/oidcauth/oidcauth.go
      Note: Combined durable transactions injected transport observability and strict logout
    - Path: repo://pkg/gojahttp/auth/oidcauth/sqlstore/sqlstore.go
      Note: Durable OIDC transaction store moved to provider-neutral namespace
    - Path: repo://pkg/gojahttp/planned_dispatch.go
      Note: Preserves authenticated actor owner-context propagation
    - Path: repo://pkg/xgoja/hostauth/builder.go
      Note: Combined production hostauth dependency graph
ExternalSources: []
Summary: Chronological implementation and validation record for integrating merged PR 98 with generalized OIDC.
LastUpdated: 2026-07-18T20:00:00-04:00
WhatFor: Preserving merge evidence, failures, decisions, commits, and review instructions.
WhenToUse: When reviewing the integration or resuming work after an interrupted step.
---


# Integration implementation diary

## Goal

This diary records the merge of updated `origin/main`—including PR 98—into `task/prod-tiny-idp`. It explains each implementation checkpoint, the exact failures encountered, the security decisions applied, and how a reviewer can validate the combined system.

## Step 1: Establish the integration project and design

The first step converted an ad-hoc conflict resolution into a documented integration project. I created a new docmgr ticket before merging, fetched the updated remote, mapped both authentication architectures, and wrote the intern-facing design guide. This makes the intended result explicit before conflict markers pressure the implementation toward one parent or the other.

The evidence shows a concentrated merge: the generalized branch and PR 98 both change identity, OIDC handlers, route planning, and host composition. The design therefore fixes the canonical identity key, package naming, logout contract, and service graph before code resolution begins.

### Prompt Context

**User prompt (verbatim):** "ok, it's merged and on origin. Merge origin/main and resolve the conflicts as shown above, by creating a new docmgr ticket, Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.\n\nThen implement stpe by step, committing at appropriate intervals."

**Assistant interpretation:** Create a fresh ticket and teaching-quality design, merge the now-landed PR 98 from `origin/main`, resolve the previously identified semantic conflicts in phased commits, validate the repository, maintain this diary, and publish the docs to reMarkable.

**Inferred user intent:** Recover control over parallel authentication work by producing one coherent, reviewable production architecture rather than a mechanically merged collection of features.

### What I did

- Loaded the ticket research, textbook authoring, diary, docmgr, Git hygiene, and reMarkable delivery instructions.
- Created ticket `XGOJA-PR98-INTEGRATION-2026-07-18` with a design doc, this diary, and eight implementation tasks.
- Fetched `origin/main`; it now ends at `b5f41a1`, the merge of PR 98, including later review-remediation commits.
- Read the current branch and `origin/main` versions of the application identity, OIDC, hostauth builder, and route-plan APIs.
- Wrote the architecture, decisions, pseudocode, diagrams, phases, tests, risks, and file map before starting the merge.

### Why

- The conflicts encode security decisions that Git cannot make.
- The intern guide must explain why the system is shaped this way, not merely list conflict resolutions after the fact.
- A pre-merge design provides review criteria for every later checkpoint.

### What worked

- The worktree was clean before ticket creation.
- `git fetch origin main` completed successfully.
- The shared base remains `6a1a095`, so the previously analyzed conflict model still applies, with additional PR review fixes on `origin/main`.

### What didn't work

- N/A in this step.

### What I learned

- PR 98's final merge includes more than the originally inspected head: generated-host smoke fixes and review findings are part of the remote integration target.
- PR 98 already separates external identities structurally, while the generalized branch correctly treats issuer plus subject as the primary OIDC key. The final model can combine these properties without a compatibility API.

### What was tricky to build

- The documentation has to distinguish browser OIDC, local device credentials, and externally issued OAuth bearers. They share users and authorization but have different credentials, verification paths, and CSRF requirements.
- The guide had to describe the intended post-merge API while citing two pre-merge implementations. It labels decisions explicitly instead of presenting unimplemented behavior as observed fact.

### What warrants a second pair of eyes

- Confirm the accepted identity model matches the desired future account-linking behavior.
- Confirm provider single sign-out should remain deferred and separate from local logout.

### What should be done in the future

- Update the guide's line references and final-state descriptions after implementation, because the merge will move symbols.

### Code review instructions

- Start with the design decisions in the companion design doc.
- Compare `pkg/gojahttp/auth/appauth/appauth.go`, `pkg/gojahttp/auth/oidcauth/oidcauth.go`, and `pkg/xgoja/hostauth/builder.go` across `HEAD` and `origin/main`.
- Verify the ticket tasks with `docmgr task list --ticket XGOJA-PR98-INTEGRATION-2026-07-18`.

### Technical details

```text
current HEAD: 49d1d8e
origin/main:  b5f41a1
merge base:   6a1a095
```

## Step 2: Merge and resolve application identity persistence

The merge produced the five conflicts predicted by the design. I selected the issuer-scoped application identity implementation as the baseline, then added PR 98's external identity binding and local-user disablement APIs. The memory and SQL stores now implement one canonical `ByExternalIdentity(issuer, subject)` lookup; the old Keycloak and transitional OIDC lookup names were migrated directly rather than retained as adapters.

The SQL store required special care because a fixture user is also a valid externally authenticated user in contract tests. `AddUser` now inserts the user and, when issuer identity is present, inserts its external binding in the same database transaction.

### Prompt Context

**User prompt (verbatim):** (see Step 1; resumed after the debugging stop with “continue”)

**Assistant interpretation:** Continue from the narrow SQL identity blocker and complete the same ticketed integration.

**Inferred user intent:** Finish the integration without discarding the prior analysis or weakening the chosen identity model.

**Commit (code):** `fdfa96d` — "Merge origin/main and integrate generalized OIDC"

### What I did

- Merged `origin/main` and recorded the five conflicts.
- Defined `ByExternalIdentity`, `BindExternalIdentity`, issuer-aware `UpsertFromOIDC`, and `DisableUser` on `UserStore`.
- Migrated test and contract callers away from `ByOIDCIdentity`.
- Added memory-store external identity indexing and disablement.
- Added SQL external identity lookup/binding and user disablement.
- Made SQL `AddUser` transactional across the user and identity rows.

### Why

- OIDC subjects are unique only inside an issuer.
- A local user ID must remain stable when external identities are linked.
- Tests and bootstrap operations must produce the same relational invariants as first login.

### What worked

- Memory tests passed after the API migration.
- Final focused command passed:

```text
ok github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/appauth
ok github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/appauth/sqlstore
```

### What didn't work

The first sandboxed test attempt could not access the normal Go cache:

```text
open /home/manuel/.cache/go-build/...: read-only file system
```

The first real compile showed the deliberate API migration was incomplete:

```text
h.Users.ByOIDCIdentity undefined (type appauth.UserStore has no field or method ByOIDCIdentity)
```

After migrating callers, SQL tests exposed a missing binding:

```text
TestSQLiteStoreContract/users_by_id_sub_and_oidc_upsert
store_contract.go:50: by sub: not found
```

The initial upsert fix bound the deterministic candidate ID even when an existing identity row caused the user upsert to retain a different fixture ID. The corrected implementation first reads the actual conflict-resolved user, then binds that user ID. The remaining fixture failure showed that `AddUser` itself also needed to establish the identity relation.

### What I learned

- A relational identity invariant must hold for production login, fixtures, migrations, and tests. Enforcing it only in `UpsertFromOIDC` leaves bootstrap users incomplete.
- Conflict-resolution upserts must use the row selected after the conflict, not assume the candidate primary key won.

### What was tricky to build

- The users table still contains issuer/subject profile columns while the external identity table is the canonical resolver. Both writes must agree during the migration period, but only the binding table drives external resolution.
- SQLite may not surface a useful foreign-key error in every test configuration, so the observed symptom was a later joined lookup returning not found.

### What warrants a second pair of eyes

- Review transaction behavior for PostgreSQL as well as SQLite.
- Review whether future manual account linking should clear or retain the user's original issuer profile columns.

### What should be done in the future

- A schema migration for already deployed databases should backfill `auth_app_external_identities` from non-empty issuer/subject user columns before switching resolvers.

### Code review instructions

- Start at `appauth.UserStore`, then follow memory and SQL implementations.
- Run `go test ./pkg/gojahttp/auth/appauth ./pkg/gojahttp/auth/appauth/sqlstore`.

### Technical details

```text
external identity uniqueness: (issuer, subject)
external identity target:     local user ID
disabled users:               resolve as not found
```

## Step 3: Port durable OIDC and compose production hostauth

This step retained PR 98's production surface while removing its provider-specific namespace. The durable transaction SQL store moved from `keycloakauth/sqlstore` to `oidcauth/sqlstore`. The OIDC handler now combines its transaction store, audit sink, security observer, injected HTTP client, issuer claims, public-client token exchange behavior, and strict logout contract.

The host builder keeps PR 98's trusted proxy resolver, dependency-aware readiness, device request/approval/denial, refresh and revocation, agent and user disablement, external OAuth verifier, maintenance service, and security metrics. It additionally passes the in-process OIDC client through that graph and registers no GET logout route.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Integrate both parents' production features without reintroducing Keycloak naming or unsafe logout behavior.

**Inferred user intent:** Obtain one coherent production authentication host rather than parallel browser and agent implementations.

**Commit (code):** `fdfa96d` — "Merge origin/main and integrate generalized OIDC"

### What I did

- Moved the durable transaction SQL package under `oidcauth/sqlstore` and updated stores, services, tests, and maintenance types.
- Combined `TransactionStore`, audit, security events, and `HTTPClient` in `oidcauth.Config`.
- Restored injected-client use for discovery, token exchange, and JWKS verification.
- Restored issuer on `OIDCClaims` and passed issuer plus subject to user normalization.
- Preserved public PKCE client `AuthStyleInParams` behavior.
- Kept POST-only CSRF logout and revocation failure propagation.
- Combined host builder options and native endpoint construction.
- Updated host tests to reject GET logout and assert issuer-scoped session claims.

### Why

- The production durability and operations work is protocol-generic and belongs under OIDC, not Keycloak.
- Same-process Tiny-IDP deployment requires a fail-closed injected transport for discovery and token/JWKS traffic.
- Local logout and provider single sign-out have different security contracts.

### What worked

- Focused OIDC, transaction-store, and hostauth tests passed.
- Broader route, device, OAuth, Express, and provider tests passed.
- Searches found no legacy Go package or method names.

### What didn't work

PR 98's session-store failure test initially reached CSRF rejection before its intended revoke failure:

```text
TestLogoutReportsSessionStoreFailure: logout status=403, want 500
```

The test now creates a valid session and supplies its CSRF token, then asserts the injected store failure is reported.

The first merge commit attempt failed lint because three provider-logout helpers became unused after GET logout was intentionally removed:

```text
pkg/gojahttp/auth/oidcauth/oidcauth.go:340:20: func (*Handlers).logoutRedirectURL is unused
pkg/gojahttp/auth/oidcauth/oidcauth.go:360:6: func absoluteRedirectURL is unused
pkg/gojahttp/auth/oidcauth/oidcauth.go:384:6: func localRedirectPath is unused
```

I removed the deferred provider-logout implementation instead of suppressing lint or retaining dead code.

### What I learned

- Removing a route is not complete until provider metadata parsing and redirect helpers are removed too.
- A failure-path test must first satisfy every earlier security gate; otherwise it asserts the wrong layer.

### What was tricky to build

- PR 98 arrived as a rename conflict: Git placed changes from `keycloakauth.go` into the existing `oidcauth.go`. The correct resolution required both a namespace move and semantic combination.
- The same HTTP client must flow through discovery, OAuth exchange, and verifier key retrieval. Passing it to only one call creates an integration that works locally but dials unexpectedly later.

### What warrants a second pair of eyes

- Review the exact device endpoint policy and rate-limit wiring in `BuildNativeHandlers`.
- Review audit payloads to ensure no credential material is recorded.
- Confirm removing provider single sign-out metadata is the intended current product behavior.

### What should be done in the future

- If provider single sign-out is required, design a separately named CSRF-protected flow with allow-listed post-logout redirects.

### Code review instructions

- Read `oidcauth.Config`, `New`, `handleCallback`, and `handleLogout` in sequence.
- Read `BuildHostAuthServices` through the final `Services` literal, then inspect `BuildNativeHandlers`.
- Run the focused auth commands listed in the design guide.

### Technical details

```text
browser logout: POST + session CSRF -> revoke -> clear cookie -> 204
OIDC state:     durable one-time TransactionStore
OIDC transport: injected client for discovery, exchange, and JWKS
```

## Step 4: Validate and commit the integrated repository

The completed merge passed isolated packages, cross-layer authentication packages, the entire repository test suite, and the full build. The commit hook then independently ran lint, generation, and the repository tests. This gave both targeted evidence for the security seams and broad evidence that PR 98's generated-host and operational changes remain compatible.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Validate proportionally to the security and repository-wide merge risk, then create a reviewable checkpoint.

**Inferred user intent:** Make the merge trustworthy enough to continue production deployment work.

**Commit (code):** `fdfa96d` — "Merge origin/main and integrate generalized OIDC"

### What I did

- Ran focused identity, OIDC, transaction, hostauth, route, device, OAuth, Express, and provider tests.
- Ran `go test ./...` successfully.
- Ran `go build ./...` successfully.
- Ran the pre-commit hook twice; the second run passed lint, generation, and tests.
- Created merge commit `fdfa96d`.

### Why

- Security integration requires targeted tests that make failures local and understandable.
- A large merged PR also requires repository-wide validation for generated code and unrelated consumers.

### What worked

- All focused and full validation passed.
- Final lint reported zero issues.
- Git recorded all five conflict resolutions.

### What didn't work

- The first commit hook failed on the unused provider-logout helpers recorded in Step 3. Its test half still passed.

### What I learned

- The repository's pre-commit hook is a valuable independent check because it includes generation and Dagger-backed frontend assets in addition to Go tests.

### What was tricky to build

- Full validation output is large; the useful evidence is the command exit status plus the specific focused suites and lint result, not copying every cached package line.

### What warrants a second pair of eyes

- Review the generated diff boundary to ensure the hook did not leave uncommitted generated artifacts. The post-commit worktree was clean.

### What should be done in the future

- Run deployment-specific Tiny-IDP and k3s smoke tests when the deployment manifests consume this commit.

### Code review instructions

- Review merge commit `fdfa96d` with `git show --cc fdfa96d`.
- Reproduce with `go test ./...` and `go build ./...`.

### Technical details

```text
design checkpoint: db007ad
merge checkpoint:  fdfa96d
origin/main merged: b5f41a1
```
