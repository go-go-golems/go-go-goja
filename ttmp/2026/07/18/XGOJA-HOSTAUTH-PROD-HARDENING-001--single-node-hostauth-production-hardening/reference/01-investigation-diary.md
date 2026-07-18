---
Title: Investigation diary
Ticket: XGOJA-HOSTAUTH-PROD-HARDENING-001
Status: active
Topics:
    - architecture
    - auth
    - operations
    - security
    - testing
    - xgoja
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://pkg/gojahttp/auth/programauth/device_handlers.go
      Note: Phase 2 native policy, budgets, lifecycle and management handlers
    - Path: repo://pkg/gojahttp/request_identity.go
      Note: Canonical request identity resolver implemented in Phase 1 commit 3b3b448
    - Path: repo://pkg/gojahttp/request_identity_test.go
      Note: Trusted proxy and context projection regression coverage
    - Path: repo://pkg/xgoja/hostauth/config.go
      Note: |-
        Hostauth configuration vocabulary investigated for this plan
        Production device-policy configuration
    - Path: repo://pkg/xgoja/hostauth/glazed.go
      Note: CLI configuration parity for device policy
    - Path: repo://pkg/xgoja/hostauth/preflight.go
      Note: Validates trusted proxy CIDR policy in commit 30bef69
    - Path: repo://pkg/xgoja/providers/http/serve.go
      Note: |-
        Native handlers are mounted before the application fallback
        Wraps all generated host routes in canonical identity middleware
ExternalSources: []
Summary: Chronological record of the evidence gathering and design work for the minimum credible single-node hostauth hardening implementation.
LastUpdated: 2026-07-18T20:56:00-04:00
WhatFor: Preserve the reasoning, commands, boundaries, and validation approach behind the companion implementation guide.
WhenToUse: Read before resuming ticket implementation or reviewing a change against the intended single-node scope.
---





# Diary

## Goal

Record the investigation that translated the external production review into a
small, evidence-backed `go-go-goja` implementation plan. The companion design
doc is written for the intern who will implement the work; this diary records
how its scope was selected and verified.

## Step 1: Create a focused hardening ticket and map the existing boundary

The ticket was created in the `go-go-goja` documentation workspace, then the
current hostauth, device authorization, rate-limit, audit, readiness, and HTTP
serve code was read before making recommendations. The result is intentionally
a scoped implementation guide rather than a promise to build every long-term
idea named by the review.

The key finding is that PR 98 already establishes durable auth-state
transitions, but the public perimeter still has three concrete gaps: forwarded
client identity is interpreted inconsistently, native device handlers bypass
planned-route limits and accept client-controlled values, and readiness reports
resolved topology rather than live SQL availability. These are implementable
inside `go-go-goja` without expanding the project into HA, a monitoring
product, or an IdP token adapter.

### Prompt Context

**User prompt (verbatim):** "ok, create a new docmgr ticket for go-go-goja to implement the sensible points of the review doc. 

Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Create a new go-go-goja docmgr ticket that turns the highest-value, non-overengineered PR 98 production-review findings into a detailed intern-facing design and implementation guide, then deliver it to reMarkable.

**Inferred user intent:** Give a future implementer a precise, evidence-backed path to harden the single-node hostauth deployment without accidentally expanding scope into unrelated future architecture.

**Commit (code):** N/A — documentation-only ticket creation; no implementation commit exists.

### What I did

- Created ticket `XGOJA-HOSTAUTH-PROD-HARDENING-001` with architecture, auth,
  operations, security, testing, and xgoja topics.
- Created `design-doc/01-intern-implementation-guide-for-single-node-hostauth-hardening.md`
  and this diary.
- Inspected hostauth configuration, preflight, store construction, builder,
  readiness, native device handlers/service, planned-route rate limiting,
  audit IP extraction, request DTO projection, and HTTP serve mux mounting.
- Ran these discovery commands from `/home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja`:

  ~~~bash
  docmgr status --summary-only
  rg -n "Forwarded|X-Forwarded|RemoteAddr|RateLimit|BuildReadinessReport|Readiness|DeviceHandlersConfig|deviceStartRequest|verificationUri|SecurityEvent|MemorySecurityMetrics" pkg/gojahttp pkg/xgoja/hostauth -S
  rg -n "type StoreBundle|sql\.Open|PingContext|apply-schema|single-node|public-base-url|Device" pkg/xgoja/hostauth -S
  rg -n "NativeHandlers|AuthOptions|NewHost|ServeHTTP|Handler" pkg/xgoja -S
  ~~~

### Why

- The original review correctly identifies a broad production roadmap, but a
  new intern needs an explicit boundary between immediate security work and
  later product/HA work.
- Code-backed claims prevent the guide from treating intended behavior as
  shipped behavior.
- Native paths must be treated separately because they are mounted on the
  `ServeMux` before the JavaScript application fallback.

### What worked

- `docmgr ticket create-ticket` created the ticket and standard workspace at:
  `ttmp/2026/07/18/XGOJA-HOSTAUTH-PROD-HARDENING-001--single-node-hostauth-production-hardening`.
- The source confirms the three central gaps exactly as the review described:
  audit unconditionally selects leftmost XFF, planned routes use RemoteAddr,
  native device routes bypass planned policies, and readiness returns a static
  `ready: true` report.
- Existing `DeviceService` already has a durable denial transition and secure
  hashed-code storage, reducing the new work to an HTTP exposure/redaction
  contract rather than a new device protocol.

### What didn't work

- Running `make lint` at `/home/manuel/workspaces/2026-06-12/goja-express-auth`
  earlier in this session failed with:

  ~~~text
  make: *** No rule to make target 'lint'.  Stop.
  ~~~

  The Go repository is nested at `go-go-goja`; the correct command is
  `cd go-go-goja && make lint`. This was unrelated to ticket documentation but
  confirms that every implementation/validation command must start from the
  repository checkout.

### What I learned

- `single-node` preflight already refuses memory stores and serving-time schema
  application, while retaining a memory limiter specifically because one
  serving process is the supported topology.
- `sql.Open` is lazy; sharing `*sql.DB` handles is already implemented, which
  gives a natural place to deduplicate readiness pings.
- Approval currently intersects stored and browser-supplied grants. The safer,
  simpler native contract is to make the stored device request authoritative
  and remove grant input from approval JSON.

### What was tricky to build

The tricky part was retaining the review's security guarantees without turning
its illustrative APIs into mandatory abstractions. For example, an interface
for request identity and a fully configurable per-endpoint policy object look
general, but the current code has one host and one deployed proxy behavior. The
solution in the guide is one concrete context-propagated resolver, fixed safe
Go budgets, and a minimal application-owned action allowlist. This preserves
the security boundary while avoiding an untested plugin/configuration matrix.

Another sharp edge is that public origin and proxy trust are different facts.
`auth.oidc.public-base-url` controls browser-visible callback construction; it
does not authorize any `X-Forwarded-For` header. The design keeps those fields
separate and validates proxy trust from the TCP peer CIDR.

### What warrants a second pair of eyes

- The exact right-to-left forwarding-chain algorithm must be checked against
  the deployed Traefik header behavior and NetworkPolicy; a unit test alone
  cannot prove ingress topology.
- The change removing `verificationUri` and approval actions from JSON is a
  deliberate compatibility break and needs example/client inventory before
  merging.
- Readiness must neither leak DSNs/errors nor accidentally make liveness fail
  during a transient SQL outage.
- Fixed native budgets need product/operator review before public launch, even
  though their presence is more important than initial numeric tuning.

### What should be done in the future

- Implement the phases in the companion guide, beginning with request identity.
- Record actual Traefik CIDRs and the ingress test result in the deployment
  repository/runbook.
- Revisit HA, metrics export, cleanup scheduling, and an IdP resource-server
  adapter only through separate tickets with concrete requirements.

### Code review instructions

- Start with `pkg/xgoja/hostauth/builder.go`, then follow native handler
  registration into `pkg/xgoja/providers/http/serve.go` and device behavior
  into `pkg/gojahttp/auth/programauth/device_handlers.go`.
- Compare all client-IP consumers: `pkg/gojahttp/ratelimit.go`,
  `pkg/gojahttp/request_response.go`, and `pkg/gojahttp/auth/audit/audit.go`.
- Validate the eventual implementation with:

  ~~~bash
  cd /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja
  go test ./pkg/gojahttp ./pkg/gojahttp/auth/programauth ./pkg/xgoja/hostauth ./pkg/xgoja/providers/http -count=1
  go test -race ./pkg/gojahttp ./pkg/gojahttp/auth/programauth/... ./pkg/xgoja/hostauth -count=1
  go test ./... && go build ./... && make lint
  ~~~

### Technical details

- Current native device paths are mounted in
  `pkg/xgoja/hostauth/builder.go:136-151` and are registered by
  `pkg/xgoja/providers/http/serve.go:448-476` before the app fallback.
- Current audit takes the first XFF value in
  `pkg/gojahttp/auth/audit/audit.go:358-366`; current limiter IP keys use
  `RequestDTO.IP`/`RemoteAddr` in `pkg/gojahttp/ratelimit.go:363-377`.
- Current readiness unconditionally builds `{ready:true}` in
  `pkg/xgoja/hostauth/readiness.go:22-35`.
- The complete design, API sketches, diagrams, phased plan, test matrix, and
  source-review reference are in the companion design document.

## Step 2: Validate the ticket and publish the review bundle to reMarkable

The completed design guide and diary were validated with docmgr and bundled as
one PDF with a table of contents. The bundle was uploaded only after the
docmgr doctor report passed cleanly and the mandated dry-run described the
correct two source documents and remote destination.

This delivery contains the design and the reasoning record together, so an
implementer reading it on reMarkable can understand both the intended contract
and the evidence that led to its constrained scope.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Validate the new ticket’s documentation and upload the completed deliverables to reMarkable.

**Inferred user intent:** Make the implementation guide reviewable away from the repository while preserving ticket bookkeeping and quality checks.

**Commit (code):** N/A — documentation-only ticket creation; no implementation commit exists.

### What I did

- Validated both document frontmatters with `docmgr validate frontmatter`.
- Added the missing `operations` topic to the repository vocabulary after
  `docmgr doctor` identified it as unknown.
- Ran `docmgr doctor --ticket XGOJA-HOSTAUTH-PROD-HARDENING-001 --stale-after 30`
  successfully.
- Verified `remarquee status` and cloud account access.
- Ran the bundle upload dry-run, then uploaded the design guide and diary to:
  `/ai/2026/07/18/XGOJA-HOSTAUTH-PROD-HARDENING-001`.
- Verified the remote directory contains `Single-node hostauth production hardening`.

### Why

- A clean doctor report ensures the ticket is searchable and uses the
  repository’s controlled vocabulary.
- The dry-run prevents publishing the wrong documents or remote path.
- Bundling the guide and diary makes the design review self-contained.

### What worked

- Both frontmatters validated successfully.
- After adding `topics/operations`, doctor reported: `✅ All checks passed`.
- `remarquee upload bundle --dry-run` showed both expected files and the
  expected destination.
- The real upload returned:

  ~~~text
  OK: uploaded Single-node hostauth production hardening.pdf -> /ai/2026/07/18/XGOJA-HOSTAUTH-PROD-HARDENING-001
  ~~~

### What didn't work

- The first doctor run emitted three unknown-topic warnings for `operations`
  (the ticket index, design doc, and diary) because the repository vocabulary
  did not yet define it. The exact remediation was:

  ~~~bash
  docmgr vocab add --category topics --slug operations --description "Production operations, health checks, deployment readiness, and maintenance guidance."
  ~~~

  Rerunning doctor passed cleanly.

### What I learned

- Ticket creation propagates its topic list into generated index/doc
  frontmatter, so a missing vocabulary entry affects all generated ticket
  documents rather than only the primary design doc.
- `remarquee upload bundle` is suitable for this deliverable because it
  preserves both documents in one review artifact with a table of contents.

### What was tricky to build

The main delivery risk was not PDF generation but provenance: a technical guide
without its diary can look authoritative while hiding important scope decisions.
Bundling both documents after a dry-run preserves the implementation contract,
the evidence trail, and validation results in one artifact.

### What warrants a second pair of eyes

- Confirm that the selected scope—request identity, native device boundary,
  and SQL readiness—matches the intended first production release before code
  work begins.
- The remote artifact is a snapshot; source-of-truth updates must continue in
  the docmgr ticket and be re-uploaded when implementation changes decisions.

### What should be done in the future

- Begin Phase 1 only after reviewers accept the design’s proxy trust contract.
- Re-upload an updated bundle after each major implemented phase or material
  design change.

### Code review instructions

- Review the guide’s Scope and non-goals, Proposed architecture, and
  File-by-file implementation plan before reviewing implementation patches.
- Use the Phase-specific focused test commands in the guide and retain their
  output in future diary entries.

### Technical details

Commands used for delivery:

~~~bash
cd /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja
docmgr doctor --ticket XGOJA-HOSTAUTH-PROD-HARDENING-001 --stale-after 30
remarquee upload bundle --dry-run <design-doc> <diary> \
  --name "Single-node hostauth production hardening" \
  --remote-dir "/ai/2026/07/18/XGOJA-HOSTAUTH-PROD-HARDENING-001" --toc-depth 2
remarquee upload bundle <design-doc> <diary> \
  --name "Single-node hostauth production hardening" \
  --remote-dir "/ai/2026/07/18/XGOJA-HOSTAUTH-PROD-HARDENING-001" --toc-depth 2
remarquee cloud ls /ai/2026/07/18/XGOJA-HOSTAUTH-PROD-HARDENING-001 --long --non-interactive
~~~

## Step 3: Establish canonical request-identity primitives

The plan was revised from a minimum-patch framing to a measured product-slice
framing, then the first Phase 1 code invariant was implemented. Go HTTP code
can now resolve a direct peer or a trusted `X-Forwarded-For` chain once,
attach the result to request context, and make request DTOs, rate-limit keys,
audit IP hashes, and access logs consume the same canonical client value.

This commit intentionally establishes the generic primitive and migrates its
consumers before generated-host configuration/wiring is added. The next Phase
1 commit must expose validated hostauth proxy configuration and wrap the
complete generated serve mux, including native handlers, in this middleware.
Until then, callers that do not install the middleware retain conservative
direct-peer behavior.

### Prompt Context

**User prompt (verbatim):** "Update design, then implement, use a detailed list of phases and tasks, commit at appropriate intervals, keep a detailed diary as you work."

**Assistant interpretation:** Revise the ticket to reflect the agreed measured product scope, then begin implementation in focused commits while documenting each step, command, and incomplete boundary.

**Inferred user intent:** Produce reviewable, resumable engineering work rather than a broad uncommitted change set or speculative architecture.

**Commit (code):** 3b3b448e0212fc8a7a6a4aad27b0332655d122cf — "gojahttp: add canonical request identity primitives"

### What I did

- Added `pkg/gojahttp/request_identity.go` with `RequestIdentity`, context
  helpers, `TrustedProxyResolver`, bounded `X-Forwarded-For` parsing, and HTTP
  middleware.
- Added resolver tests for direct forged headers, trusted and multi-hop proxy
  chains, untrusted peers, malformed trusted chains, and DTO context
  projection.
- Changed `NewRequestDTO`, planned-route IP rate-limit keys, audit IP hashing,
  and access logging to use `RequestClientIP`.
- Added the revised product-slice phase table and two additional ticket tasks;
  committed the documentation plan as
  `28346461f8e001799cc02f689ebf556aec6b8712`.
- Ran:

  ~~~bash
  gofmt -w pkg/gojahttp/request_identity.go pkg/gojahttp/request_identity_test.go pkg/gojahttp/request_response.go pkg/gojahttp/ratelimit.go pkg/gojahttp/auth/audit/audit.go pkg/gojahttp/access_log.go
  go test ./pkg/gojahttp ./pkg/gojahttp/auth/audit -count=1
  git commit -m "gojahttp: add canonical request identity primitives"
  ~~~

### Why

- Audit previously trusted leftmost XFF while planned rate limits used
  `RemoteAddr`; one canonical value is required before either system can be
  trusted behind Traefik.
- Keeping proxy parsing in `gojahttp` avoids a hostauth-only implementation
  that native or non-hostauth HTTP users could not share.
- Separating primitive/consumer migration from host configuration keeps the
  security invariant reviewable and makes the remaining deployment wiring
  explicit.

### What worked

- Focused tests passed:

  ~~~text
  ok github.com/go-go-golems/go-go-goja/pkg/gojahttp
  ok github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/audit
  ~~~

- The repository pre-commit hook completed lint and the full generated test
  suite successfully. It ran `golangci-lint`, `go vet` with `glazed-lint`,
  `go generate ./...`, and `go test ./...`.

### What didn't work

- There was no code failure. The pre-commit hook started Dagger to regenerate
  Bun demo assets and printed registry transport retries, but it completed
  successfully and the commit was created.
- The generated host has not yet installed a `TrustedProxyResolver`; therefore
  this commit alone does not cause a production Traefik request to use XFF.
  That wiring is deliberately recorded as the immediate next change, not
  treated as complete.

### What I learned

- `RequestDTO.IP` is the correct bridge from native `http.Request` identity to
  planned-route limits because the limiter already prefers it.
- Moving audit to `gojahttp.RequestClientIP` removes duplicate forwarding-header
  parsing rather than merely making the two parsers behave similarly.
- The pre-commit contract is stronger than the focused test command and is a
  useful commit boundary for this repository.

### What was tricky to build

The right-to-left rule is essential. A trusted proxy appends its observation to
an existing header chain, so selecting the leftmost value would still allow a
caller-supplied address to dominate. The resolver parses a bounded chain,
starts at the right, skips configured trusted proxy addresses, and chooses the
first untrusted hop. Malformed data from a trusted proxy fails closed; data
from an untrusted direct peer is ignored entirely.

### What warrants a second pair of eyes

- The resolver currently supports the deployed `X-Forwarded-For` contract,
  not the standardized `Forwarded` header. Confirm Traefik configuration
  before adding formats.
- The final hostauth configuration must validate CIDRs and wrap the outer
  ServeMux, rather than only the JavaScript host, so native auth handlers
  receive the identity too.
- Access-log privacy policy should decide whether recording canonical
  `client_ip` in clear text is acceptable for the target deployment.

### What should be done in the future

- Complete Phase 1 hostauth proxy configuration and generated-server middleware
  installation before checking its ticket task complete.
- Continue Phase 2 device lifecycle only after the identity middleware covers
  both native and planned routes.

### Code review instructions

- Start at `pkg/gojahttp/request_identity.go`; review direct mode, trusted peer
  detection, chain bounds, and the right-to-left walk.
- Confirm every migrated consumer calls `RequestClientIP` rather than parsing
  XFF or `RemoteAddr` itself.
- Validate with `go test ./pkg/gojahttp ./pkg/gojahttp/auth/audit -count=1`,
  then use `make lint` and `go test ./...` from the repository root.

### Technical details

The canonical fallback deliberately returns `unknown` rather than a malformed
`RemoteAddr` string. This avoids allowing malformed transport data into a
rate-limit key or audit hash. `TrustedProxyResolver.Resolve` requires a valid
peer IP; its middleware returns HTTP 400 for malformed trusted forwarded data.

## Step 4: Wire the proxy policy around generated hosts

The generic resolver is now reachable from generated hostauth configuration and
wraps the outer generated HTTP mux. This completes the phase-one propagation
path: native handlers and the JavaScript application receive the same resolved
request context rather than only a planned-route-specific view.

### Prompt Context

**User prompt (verbatim):** "phase 1"

**Assistant interpretation:** Complete the outstanding Phase 1 generated-host proxy configuration and middleware wiring.

**Inferred user intent:** Make the request-identity foundation operational for a host deployed behind trusted Traefik, not merely available as a library primitive.

**Commit (code):** 30bef695142eb652f59bd34c43ada40cf2167c7d — "hostauth: wire trusted proxy identity"

### What I did

- Added `auth.proxy.mode` and `auth.proxy.trusted-cidrs` to hostauth config and
  resolved CIDR prefixes with reject-on-invalid, duplicate, or contradictory
  configuration behavior.
- Added the resolved `gojahttp.TrustedProxyResolver` to `hostauth.Services`.
- Wrapped the complete `ServeMux` in `RequestIdentityMiddleware` in
  `pkg/xgoja/providers/http/serve.go`, after mounting native routes and the
  application fallback.
- Added hostauth resolution coverage for valid trusted forwarding and invalid
  direct/empty/malformed configurations.
- Ran focused package tests and committed; the repository hook subsequently
  passed lint, vet, generation, and `go test ./...`.

### Why

- Native `/auth/*` endpoints and planned JavaScript routes must agree on the
  client identity; wrapping only `gojahttp.Host` would leave native handlers
  outside the security boundary.
- Explicit CIDRs prevent the dangerous configuration where an XFF header alone
  creates trust.

### What worked

- `go test ./pkg/xgoja/hostauth ./pkg/xgoja/providers/http ./pkg/gojahttp -count=1`
  passed before commit.
- The pre-commit hook passed its full lint/generate/test workflow.

### What didn't work

- No implementation failure occurred. The generated CLI/Glazed flat settings
  do not yet expose proxy fields; direct YAML/host configuration works, but
  command-flag parity remains a small Phase 1 documentation/config follow-up.

### What I learned

- `buildServeHandler` is the correct wrapping point because it owns the mux
  that registers both `Services.NativeHandlers` and the application fallback.

### What was tricky to build

The ordering matters: the middleware must wrap the mux after all routes are
registered, not be applied only to the fallback application handler. The final
shape preserves native-route precedence while adding the same request context
to every branch.

### What warrants a second pair of eyes

- Confirm the configured production Traefik CIDR and NetworkPolicy in the
  deployment repository; code cannot infer a safe cluster network range.
- Add Glazed proxy fields before declaring command-line configuration complete.

### What should be done in the future

- Add Glazed/config-reference parity and an ingress-level Traefik test, then
  mark Phase 1 complete.
- Begin Phase 2 only after the real proxy topology is verified.

### Code review instructions

- Review `resolveProxyConfig`, then verify that `buildServeHandler` wraps the
  whole mux with `authServices.RequestIdentity`.
- Re-run the focused test command above and `make lint` from the repository
  root.

### Technical details

`trusted-forwarded` requires at least one CIDR; `direct` rejects CIDRs. A
trusted peer with a malformed XFF chain receives a 400 rather than a guessed
identity, while an untrusted peer's XFF is ignored.

## Step 5: Start Phase 2 with inspectable, denyable device requests

Phase 2 began by exposing the existing durable denial state transition through
session- and CSRF-protected native endpoints. A browser can now inspect a
redacted pending device request and deny it; later polling reports
`access_denied` without exposing either code.

### Prompt Context

**User prompt (verbatim):** "do it, phase 2"

**Assistant interpretation:** Begin the complete application-owned device lifecycle hardening phase.

**Inferred user intent:** Make the coding-agent authorization flow safe and understandable for a browser user, beginning with informed approval and refusal.

**Commit (code):** 831887c3aa81d7596744028e7bad34cfcad1a3a6 — "hostauth: add device request inspection and denial"

### What I did

- Added a redacted `PendingDeviceRequestView` and `InspectDeviceAuthorization`.
- Added `POST /auth/device/request` and `POST /auth/device/deny`, both guarded
  by local session authentication and CSRF.
- Mounted both routes in hostauth and added a lifecycle test proving inspection
  leaks neither device nor user code and denial makes polling return
  `access_denied`.
- Ran focused package tests; the commit hook passed lint, generation, and the
  complete test suite.

### Why

- A user cannot make an informed authorization decision without seeing the
  stored client name/actions, and must be able to refuse a request terminally.
- User codes belong in POST bodies, not URLs, to avoid query/history leakage.

### What worked

`go test ./pkg/gojahttp/auth/programauth ./pkg/xgoja/hostauth -count=1` and
the full pre-commit test/lint workflow passed.

### What didn't work

No code failure occurred. Action allowlists, server-owned verification URIs,
native endpoint budgets, and owner-scoped agent management are still Phase 2
work; this commit deliberately does not claim the phase complete.

### What I learned

The service already had correct denial persistence and poll semantics; the
missing boundary was redacted, session-protected HTTP exposure.

### What was tricky to build

Inspection needs the entered user code to locate a record but must not echo it
or expose the device credential/token state. A distinct pending view prevents
accidentally serializing `DeviceAuthorizationView` fields meant for internal
workflow responses.

### What warrants a second pair of eyes

Review that new handler errors are generic and that session/CSRF validation is
identical to existing approval semantics.

### What should be done in the future

Continue Phase 2 with configured action policy, trusted verification URI,
native budgets, and owner-scoped agent operations.

### Code review instructions

Review `device.go` pending projection, `device_handlers.go` session/CSRF helper,
and the lifecycle regression test; run the focused command above.

### Technical details

The inspection response contains client name, stored requested actions, expiry,
and terminal/pending status only. It contains no user code, device code, token
hash, token family, or subject identity.

## Step 6: Complete the implemented Phase 2 device boundary

The remaining Phase 2 slices were completed as focused commits: device-start
grant constraints, native request budgets, single-node policy configuration,
and owner-scoped agent listing/disablement. The work keeps policy in Go-owned
host configuration and keeps the ownership predicate in the service layer,
rather than trusting a browser UI.

### Prompt Context

**User prompt (verbatim):** "You are still stopping after the commit. DON'T. Do phase 2 in full, with all its commits and diary updates."

**Assistant interpretation:** Continue through the remaining agreed Phase 2 implementation and record the whole sequence before reporting completion.

**Inferred user intent:** Receive a complete, auditable lifecycle hardening increment rather than repeated partial progress messages.

**Commit (code):** 2c87e64d03d23d5cfd5ee7e94463c29accf73fd5 — "programauth: constrain device start grants"; 318e6d4b1d6ec605ad68db1c0b296bfb05451bde — "hostauth: rate limit native device endpoints"; 5bab59a52c463e9ae7f08503956a5555da6b201f — "programauth: enforce owner-scoped agent disablement"; 3412610812054a723c0bd201d8f90c0c76c8b7ef — "hostauth: require production device policy"; a34076882bc78ee7d3accff5e24efc5a7f28a8c4 — "hostauth: expose owner-scoped agent management"; 83133e5fdcaebed28d8eb096c7d5064c38536dda — "hostauth: expose device policy command settings"

### What I did

- Required non-empty device actions, optional allowlist membership, and an
  optional maximum action count before device persistence.
- Removed client control of `verificationUri`; host policy supplies it.
- Applied IP-keyed fixed-window budgets to start, polling, refresh, revoke,
  inspect, deny, and approval, returning 429 with `Retry-After`.
- Added `auth.device` config plus Glazed flags and made single-node preflight
  require configured actions and a verification URI.
- Added owner-scoped agent service methods and native `GET /auth/agents` plus
  CSRF-protected `POST /auth/agents/disable` routes.
- Added lifecycle and ownership regression coverage; every code commit passed
  the repository pre-commit lint, generation, vet, and full test workflow.

### Why

These pieces together provide the current product’s agent lifecycle: a user
can inspect/deny a pending authorization, receive only approved local actions,
and list or disable only their own agents. They also bound the public native
work before code/token persistence and remove client-selected browser origins.

### What worked

All focused tests and every full pre-commit workflow passed. The new
single-node configuration fails closed when device policy is absent.

### What didn't work

Refresh-family **self-service listing/revocation** is not implemented because
current store interfaces do not expose an owner-scoped family query. The
existing credential-presented `/auth/device/revoke` endpoint remains available.
Adding a browser management view safely requires a separate store/query design;
it must not be faked by accepting arbitrary family IDs from a browser.

### What I learned

The existing programauth domain already contained durable device denial,
agent-disable semantics, and refresh-family revocation. The central work was
placing the correct HTTP/configuration boundaries around those primitives.

### What was tricky to build

Native endpoints do not traverse planned-route enforcement, so they need their
own small use of the shared limiter. The implementation uses canonical client
identity and stable low-cardinality policy names rather than manufacturing fake
route plans or allowing raw codes into limiter keys.

### What warrants a second pair of eyes

- Budget values are conservative defaults and should be tuned against observed
  usage before public launch.
- `verification-uri` should be constrained to the configured public HTTPS
  origin in a follow-up rather than merely supplied by trusted configuration.
- Management handlers should gain HTTP-level cross-owner tests in addition to
  the service-level ownership test.

### What should be done in the future

- Design owner-scoped refresh-family storage/query APIs and then add browser
  revocation UI/routes.
- Add real Traefik ingress proof, maintenance/cleanup commands, and Phase 3
  dependency readiness.

### Code review instructions

Review `DeviceEndpointPolicy`, `allowRequest`, `DeviceConfig` preflight, and
`DisableOwnedAgent` together. Confirm no handler takes an owner ID, raw device
code in a URL, or client-selected verification URI.

### Technical details

The production config shape is:

~~~yaml
auth:
  device:
    allowed-actions: [user.self.read]
    max-actions: 1
    verification-uri: https://app.example.test/auth/device
~~~
