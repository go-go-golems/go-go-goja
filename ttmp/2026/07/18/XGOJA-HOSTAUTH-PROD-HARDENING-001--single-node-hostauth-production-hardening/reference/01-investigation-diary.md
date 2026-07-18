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
    - Path: repo://pkg/xgoja/hostauth/config.go
      Note: Hostauth configuration vocabulary investigated for this plan
    - Path: repo://pkg/xgoja/providers/http/serve.go
      Note: Native handlers are mounted before the application fallback
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
