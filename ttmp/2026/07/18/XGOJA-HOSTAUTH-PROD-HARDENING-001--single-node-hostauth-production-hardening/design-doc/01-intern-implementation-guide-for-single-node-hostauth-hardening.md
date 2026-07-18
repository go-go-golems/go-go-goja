---
Title: Intern implementation guide for single-node hostauth hardening
Ticket: XGOJA-HOSTAUTH-PROD-HARDENING-001
Status: active
Topics:
    - architecture
    - auth
    - operations
    - security
    - testing
    - xgoja
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://pkg/gojahttp/auth/audit/audit.go
      Note: Current unsafe forwarded-IP handling to replace
    - Path: repo://pkg/gojahttp/auth/programauth/device_handlers.go
      Note: Current public native device HTTP contract to harden
    - Path: repo://pkg/gojahttp/ratelimit.go
      Note: Existing planned-route limiter and current IP key behavior
    - Path: repo://pkg/xgoja/hostauth/builder.go
      Note: Composes stores, native auth routes, and planned-route auth services
    - Path: repo://pkg/xgoja/hostauth/readiness.go
      Note: Current static readiness report to replace
    - Path: repo://pkg/xgoja/hostauth/stores.go
      Note: Shared SQL handle builder and future health-check ownership
ExternalSources:
    - https://www.rfc-editor.org/rfc/rfc8628
    - https://www.rfc-editor.org/rfc/rfc7009
Summary: Evidence-backed, intern-facing design and phased implementation guide for the minimum credible single-node production hardening of xgoja hostauth.
LastUpdated: 2026-07-18T20:55:00-04:00
WhatFor: Turn the sensible, high-value findings of the production review into small, testable go-go-goja changes without prematurely building HA, a generic metrics system, or an IdP resource-server adapter.
WhenToUse: Before implementing or reviewing hostauth request identity, native device endpoint policy, and SQL-backed readiness for a public single-replica xgoja host behind Traefik.
---


# Intern implementation guide for single-node hostauth hardening

## Executive summary

`hostauth` is optional Go-owned authentication infrastructure for generated
xgoja HTTP hosts. It gives an application browser OIDC login, local sessions,
local users, audit storage, and an application-owned device authorization flow.
It is not tiny-idp, and it is not a generic resource server for every OAuth
bearer token seen by an application.

The repository already has valuable foundations: durable OIDC login
transactions, SQL-backed programmatic token storage, refresh-token rotation,
refresh-family revocation, a single-node preflight profile, and low-cardinality
security events. This ticket deliberately implements only the remaining
minimum safety boundaries that make those foundations credible for one public
process behind a known reverse proxy:

1. derive one trustworthy client address once per request;
2. constrain and rate-limit Go-owned device endpoints before durable work;
3. make device approval inspectable and deniable without trusting browser
   supplied grants; and
4. make readiness probe real SQL dependencies rather than echo configuration.

The design is intentionally smaller than the source review. It does **not**
add high availability, a distributed limiter, a generic exporter framework,
self-service agent-management UI, scheduled cleanup, or tiny-idp token
introspection. Those may be useful later, but they are not prerequisites for
a secure single-node deployment and would obscure the core security work.

## 0. Revision: measured product scope and execution plan

This revision supersedes the earlier use of “minimum” in this document. The
goal is not the smallest patch that closes a vulnerability; it is the complete,
measured capability required by the next product workflows: a multi-user
browser application can authorize, inspect, deny, list, and disconnect its own
application-owned coding agents, and the next follow-up can admit a
single configured tiny-idp issuer through a planned OAuth route.

### Required hostauth delivery

1. Canonical trusted-proxy request identity used by every HTTP consumer.
2. A complete app-owned device lifecycle: action policy, server-owned
   verification URI, request budgets, inspection, approval, denial,
   owner-scoped agent listing, agent disablement, and refresh-family
   revocation.
3. Truthful liveness/readiness, migration and cleanup runbook contracts, and
   redacted audit/security-event integration.
4. A reference-app smoke proving that one user cannot inspect, disable, or
   revoke another user's agents.

This does **not** mean building HA, a distributed limiter, a scheduler,
arbitrary policy plugins, or a monitoring-product abstraction. Those needs
remain unspecified.

### Required OAuth follow-up delivery

The OAuth follow-up is a separate vertical slice, not a change to
`programauth`: typed `express.oauth().issuer().resource().scopes()` plans,
strict enforcement, one tiny-idp RFC 7662 verifier profile, issuer-qualified
local identity, redacted context/audit, and a device-token-to-Express smoke.
It starts without introspection caching; cache design waits for measured load
and an explicit revocation-delay SLO. It does not support mixed credentials,
JWTs, or multi-issuer operations in its first release.

### Ordered work breakdown

| Phase | Deliverable | Ticket task / exit evidence |
| --- | --- | --- |
| 1 | Trusted request identity | Direct and trusted-Traefik tests prove audit and limiter agree. |
| 2 | Device policy and lifecycle | Native handlers enforce budgets/policy; owner-scoped management tests pass. |
| 3 | Operations truthfulness | SQL outage/recovery readiness tests and documented migration/cleanup commands pass. |
| 4 | OAuth plan/API | Builder and enforcer fake-verifier tests reject every malformed/mismatched route before JS. |
| 5 | External identity and verifier | `(issuer, subject)` migration and one configured tiny-idp RFC 7662 profile pass strict tests. |
| 6 | Product proof | App-owned and IdP-owned device flows pass end-to-end smokes through the deployed proxy shape. |

Phases 1–3 are the hostauth release. Phases 4–6 are the explicitly subsequent
OAuth release. A commit completes one coherent invariant and its focused tests;
it must not combine unrelated schema, route syntax, and proxy changes.

## 1. System orientation

### 1.1 What is built here

Goja executes JavaScript in Go. `go-go-goja` adds reusable Go modules,
Express-like HTTP route planning, generated-host tooling, and xgoja provider
composition around it. An xgoja specification selects providers and produces a
specific binary; selecting `hostauth` is therefore an opt-in capability, not a
policy imposed on every Goja user.

The relevant request path is:

~~~text
Internet browser / CLI
        |
        | HTTPS; TLS terminates at Traefik
        v
Traefik ------------------> generated xgoja HTTP server
                                  |
                                  +-- Go-native hostauth paths (/auth/...)
                                  |
                                  +-- planned Express routes enforced in Go
                                  |       before JavaScript runs
                                  v
                              application JavaScript
                                  |
                                  v
                             durable local SQL stores
~~~

`pkg/xgoja/providers/http/serve.go` constructs a standard-library `ServeMux`,
registers `Services.NativeHandlers` before the application fallback, then
mounts the app at `/` (lines 448-476). Consequently, a native endpoint such as
`/auth/device/start` never passes through the planned-route `Enforcer`. This is
the central fact behind the endpoint-budget work.

### 1.2 Identity ownership: do not merge the two credential systems

Tiny-idp is the OpenID Provider. It owns passwords, provider sessions, OIDC
browser authentication, and (separately) its own OAuth tokens. An xgoja host
is an OIDC relying party: it verifies an ID token, maps its issuer/subject to a
**local application user**, and creates a local session cookie.

After an authenticated browser approves a device request, `programauth` creates
an **application agent** and returns application-owned `ggat_` access and
`ggrt_` refresh credentials. They are stored and validated by the application.
They must not be sent to tiny-idp introspection simply because they use the
Bearer scheme.

~~~text
human browser                                      coding agent
     |                                                   |
     | OIDC authorization-code flow + PKCE               | app device code
     v                                                   v
tiny-idp ---> verified ID token ---> local session --> programauth agent
                                                      |
                                                      +--> ggat_ / ggrt_
                                                           validated locally
~~~

The existing handler comment is explicit about this boundary:
`pkg/gojahttp/auth/programauth/device_handlers.go:105-107`. Preserve it in new
APIs, examples, errors, and documentation.

### 1.3 Existing hostauth composition

`hostauth.Config` (`pkg/xgoja/hostauth/config.go:41-54`) is host configuration,
not JavaScript route policy. `ResolveConfig` parses and validates it, and the
service factory builds stores, sessions, authentication, authorization, a
limiter, device services, and native handlers.

`pkg/xgoja/hostauth/builder.go:83-128` is the main composition root:

- `BuildStores` builds stores and shares one `*sql.DB` per equal driver/DSN.
- `BuildSessionManager` owns local browser-session parsing and CSRF validation.
- `programauth.AgentService`, `OAuthTokenService`, and `DeviceService` compose
  the application-owned agent and token lifecycle.
- `BuildAuthOptions` supplies planned routes with Go-owned authenticators,
  audit, authorization, and rate limiting.
- `BuildNativeHandlers` mounts readiness and device endpoints independently of
  the planned-route pipeline.

A useful learning exercise before editing is to trace `BuildNativeHandlers`
(lines 134-189), then `DeviceHandlers` (`pkg/gojahttp/auth/programauth/device_handlers.go`), then `DeviceService`
(`pkg/gojahttp/auth/programauth/device.go`).

## 2. Evidence-backed current state and gaps

### 2.1 Single-node preflight is already correctly strict

The `single-node` profile rejects memory-backed stores, runtime schema changes,
insecure session cookies, non-OIDC mode, and non-memory limiter drivers
(`pkg/xgoja/hostauth/preflight.go:29-56`). Its contract is one serving process;
the in-memory fixed-window limiter is therefore acceptable for this deployment
only. Do not change the profile to imply HA.

The profile also requires migrations before startup. This ticket does not build
a migration framework: each SQL store already owns an `ApplySchema` path, but
production must run migration tooling outside the serving process.

### 2.2 Client identity disagrees across subsystems

The planned-route limiter obtains `RateLimitKeyIP` from `RequestDTO.IP` or
`RemoteAddr` (`pkg/gojahttp/ratelimit.go:314-377`). `NewRequestDTO` itself
splits only `RemoteAddr` (`pkg/gojahttp/request_response.go:68-73`). In
contrast, audit's `clientIP` blindly trusts the leftmost `X-Forwarded-For`
value (`pkg/gojahttp/auth/audit/audit.go:358-366`).

This produces two unsafe cases:

| Incoming path | Audit identity now | Rate-limit identity now | Problem |
|---|---|---|---|
| Direct request with forged `X-Forwarded-For` | Caller-selected | Direct peer | Spoofable audit attribution |
| Real request through Traefik | Original caller | Traefik peer | All clients share one limiter bucket |

There is no existing proxy trust configuration. The public OIDC base URL solves
an entirely separate problem: it lets callbacks and Secure cookies use the
browser-visible HTTPS origin. It does not establish that a forwarding header
is trustworthy.

### 2.3 Device endpoints have protocol mechanics but no host policy perimeter

`BuildNativeHandlers` currently mounts five unauthenticated/native paths
(lines 140-151): start, poll/token, refresh, revoke, and approve. The handlers
accept raw client actions and, for start, a caller-provided `verificationUri`
(lines 46-59 and 242-248). `DeviceService` only uses its configured
`VerificationURI` when the supplied value is empty
(`pkg/gojahttp/auth/programauth/device.go:138-187`). A public client can thus
make the host advertise an attacker URL in a valid-looking device response.

The underlying service has good protocol building blocks:

- device and user codes are random and only hashes are persisted;
- device polling increases the per-device interval and returns `slow_down`;
- approval intersects requested grants with supplied grants, so it cannot
  broaden the request (`device.go:194-249`);
- a store transition denies a device request, and polling maps denial to
  `access_denied` (`device.go:293-305` and `device_handlers.go:324-330`);
- SQL pair stores can atomically issue/rotate access and refresh token pairs.

However, RFC 8628 `slow_down` only protects a known device authorization. It
does not limit durable device creation, invalid-code guessing, or refresh
floods. Existing planned-route rate limits cannot cover native handlers.

### 2.4 Readiness is a static topology report

`BuildReadinessReport` unconditionally sets `Ready: true`
(`pkg/xgoja/hostauth/readiness.go:22-35`). `BuildNativeHandlers` closes over
that static value at construction. `BuildStores` calls `sql.Open`, which creates
a handle but does not prove the backing service is reachable
(`pkg/xgoja/hostauth/stores.go:57-66,258-273`).

The current response safely omits DSNs and OIDC secrets, which must remain
true, but it answers “was configuration resolved?” rather than “can the host
serve an auth request now?” Kubernetes readiness needs the latter.

## 3. Scope and non-goals

### In scope

- A concrete, host-level trusted-proxy policy and canonical request identity.
- Native device action validation, server-owned verification URI, and bounded
  per-endpoint request budgets using the existing limiter contract.
- Session + CSRF protected device request inspection and terminal denial.
- Dynamic, bounded SQL dependency readiness and a separate cheap liveness
  endpoint.
- Unit, concurrency, race-focused, and reverse-proxy integration coverage.
- Updates to the reference application/configuration documentation needed to
  exercise the changed native contract.

### Explicitly deferred

- Multiple replicas, distributed limiting, and a `production-ha` profile.
- A generic `RequestIdentityResolver` plugin interface; one concrete resolver
  is enough until a second implementation exists.
- A generic metrics exporter or vendor-specific monitoring implementation. The
  existing `SecurityEventObserver` remains the integration seam; memory is an
  acceptable default until an application/operator selects an exporter.
- Full user-facing agent inventory, rename, token-family history, and
  disconnect UI. Those need product design beyond the shared host boundary.
- Cleanup schedulers and migration orchestration. Document operator procedures
  separately; do not embed schema mutation into a serving process.
- Accepting tiny-idp-issued OAuth tokens in `programauth`, or building the
  future tiny-idp introspection adapter.

## 4. Proposed architecture

### 4.1 Canonical request identity

Add a minimal proxy configuration in `pkg/xgoja/hostauth/config.go`, resolve
and validate it in `resolve.go`/`preflight.go`, and compile it to a concrete
resolver. Suggested configuration:

~~~go
type ProxyMode string
const (
    ProxyModeDirect ProxyMode = "direct"
    ProxyModeTrustedForwarded ProxyMode = "trusted-forwarded"
)

type ProxyConfig struct {
    Mode         ProxyMode `yaml:"mode" json:"mode"`
    TrustedCIDRs []string  `yaml:"trusted-cidrs" json:"trusted-cidrs"`
}
~~~

`direct` ignores forwarding headers. `trusted-forwarded` only considers a
forwarding chain after parsing the TCP peer from `RemoteAddr` and establishing
that peer is inside a configured trusted network. No header alone creates
trust. Empty/malformed CIDRs are configuration errors; a trusted-forwarded
mode with no ranges is also a configuration error.

Put generic parsing/context code in `pkg/gojahttp`, for example
`request_identity.go`; put deployment YAML parsing only in `hostauth`. Store a
small immutable value in the request context:

~~~go
type RequestIdentity struct {
    PeerIP   netip.Addr
    ClientIP netip.Addr
    ViaProxy bool
}

func RequestIdentityFromContext(ctx context.Context) (RequestIdentity, bool)
~~~

The server wrapper must run around the **whole** HTTP mux. This is important:
it means native handlers and planned JavaScript routes receive the same
request context. Add wrapping in `buildServeHandler` in
`pkg/xgoja/providers/http/serve.go`, using the resolver exposed from
`hostauth.Services`; do not bolt separate parsing into each handler.

For the first version, support only `X-Forwarded-For`, as that is the current
Traefik assumption. Reject malformed/oversized chains with `400` when they
come from a trusted peer; ignore any forwarded header from an untrusted peer.
Cap header bytes and hop count before splitting it. Start walking at the right,
skipping trusted proxy addresses, and return the first untrusted address. This
prevents a client-provided leftmost value from overriding a chain that Traefik
appended to.

~~~text
resolve(r):
  peer = parse RemoteAddr as IP; reject malformed peer
  if mode == direct:
      return {peer, peer, false}
  if peer not in trusted CIDRs:
      return {peer, peer, false}       # headers are attacker input
  hops = parse bounded X-Forwarded-For
  if hops malformed: return error      # trusted proxy contract broken
  for hop in hops from right to left:
      if hop not in trusted CIDRs: return {peer, hop, true}
  return {peer, peer, true}            # all chain entries were trusted
~~~

All consumers must use `RequestIdentity.ClientIP`: `NewRequestDTO` sets `IP`
from it, rate-limit `requestIP` prefers it, audit hashes it, access logs include
redacted/appropriate peer and client fields, and native budgets read it. Do not
project raw header values to JavaScript.

### Decision: one context value, not package-local header parsing

- **Context:** audit and limiting already disagree about the client address.
- **Options considered:** fix audit only; fix limiter only; each package parses
  forwarding headers; resolve once at the server boundary.
- **Decision:** resolve once and propagate an immutable context value.
- **Rationale:** it gives every security record and limiter bucket a single,
  reviewable definition of “client.”
- **Consequences:** the HTTP serve layer must wrap its mux, and all consumers
  must be migrated in one change; ingress behavior needs an integration test.
- **Status:** proposed.

### 4.2 Minimal native device policy

Add a focused `DeviceConfig` under `hostauth.Config`, rather than exposing a
large policy DSL. The application must provide the vocabulary it allows;
security defaults remain owned by Go.

~~~go
type DeviceConfig struct {
    AllowedActions  []string `yaml:"allowed-actions" json:"allowed-actions"`
    VerificationPath string  `yaml:"verification-path" json:"verification-path"`
}
~~~

Require a non-empty, normalized allowlist in the `single-node` profile. Reject
an empty action, duplicates, or a requested action not in the allowlist with
`invalid_scope` **before** `StartDeviceAuthorization` writes a row. Preserve
exact requested actions in the device record. The host does not know whether
`inbox.capture` is meaningful; an application does.

Build verification URI from the configured public origin and a fixed,
slash-normalized relative path. Reject `verificationUri` as an unknown JSON
field by removing it from `deviceStartRequest`; do not silently honor it. In
development, a relative configured URI may be retained only if existing
examples require it. In `single-node`, require an absolute HTTPS result from
the known public base URL.

Use the existing `gojahttp.RateLimiter`, not a second limiter interface. Create
small internal `RateLimitSpec` values with stable policy names, one fixed safe
set of limits, and an IP key resolved from request identity. This avoids an
unnecessary configuration matrix while ensuring every public endpoint is
bounded.

| Endpoint | Example policy | Key | Order |
|---|---|---|---|
| `POST /auth/device/start` | `auth.device.start` | client IP | before JSON decode/persistence |
| `POST /auth/device/token` | `auth.device.poll` | client IP | before code lookup |
| `POST /auth/device/refresh` | `auth.device.refresh` | client IP | before token lookup |
| `POST /auth/device/revoke` | `auth.device.revoke` | client IP | before token lookup |
| inspect/approve/deny | `auth.device.approval` | client IP, then session actor where useful | before code lookup |

Write `429` plus a rounded-up `Retry-After` when the limiter denies. Do not
replace RFC 8628's protocol-level `slow_down`: it remains the response for a
valid device code polled sooner than its stored interval.

A narrow helper in `programauth/device_handlers.go` can call the limiter with a
native `RateLimitRequest` and avoid depending on private planned-route types.
If the existing request type cannot be used cleanly without fake route plans,
add a small, shared `CheckNamedRateLimit(ctx, limiter, policy, key, spec)`
helper in `gojahttp`; do not fabricate a JavaScript route just to reuse it.

### 4.3 Inspection, approval, and denial

The service already has `DenyDeviceAuthorization`, but no HTTP handler exposes
it. It also has only private lookups. Add a redacted inspection service method
and three native handlers:

~~~http
POST /auth/device/request
Cookie: app_session=...
X-CSRF-Token: ...

{"user_code":"ABCD-EFGH-IJKL"}

200 {"clientName":"inbox-cli","requestedActions":["user.self.read"],
     "expiresIn":418,"status":"pending"}

POST /auth/device/approve
Cookie: app_session=...
X-CSRF-Token: ...

{"user_code":"ABCD-EFGH-IJKL","agentName":"my laptop"}

POST /auth/device/deny
Cookie: app_session=...
X-CSRF-Token: ...

{"user_code":"ABCD-EFGH-IJKL"}
~~~

All three require the same `SessionFromRequest` and `VerifyCSRF` sequence as
the current approval handler (`device_handlers.go:170-185`). Never return a
device code/hash, token family, subject ID, or arbitrary stored record. The
inspection response must not echo the user code either. It may return client
name, stored actions, expiry, and terminal/pending state.

Simplify approval: the browser supplies a user code and optional agent display
name, but not actions or tenant. The pending record is authoritative and
`DeviceService.ApproveDeviceAuthorization` receives its stored grant set. This
makes the UI contract clearer than the existing intersection behavior while
preserving its service-side safety invariant. The service remains the final
boundary; a future UI cannot gain authority by hiding fields.

### Decision: fixed Go defaults, small application policy

- **Context:** device endpoints need limits and application-specific grants,
  but a fully configurable endpoint-policy object is difficult to document and
  tune safely for first deployment.
- **Options considered:** no native limits; all limits/actions in JavaScript;
  expose every duration/budget in YAML; fixed Go limits plus allowlisted
  actions/path.
- **Decision:** use fixed conservative Go limits, with only action vocabulary
  and verification path supplied by host configuration.
- **Rationale:** actions are application semantics; abuse limits are a shared
  transport-security baseline.
- **Consequences:** applications needing materially different capacity must
  propose a measured configuration extension rather than silently weaken the
  defaults.
- **Status:** proposed.

### 4.4 Dependency-aware readiness

Keep three concerns distinct:

| Endpoint/signal | Meaning | SQL outage behavior |
|---|---|---|
| `/healthz` | process can answer HTTP | still 200 |
| `/auth/readyz` | hostauth can safely accept new auth traffic | 503 |
| startup/preflight logs | configured topology is supported | not a live probe |

Add a private `[]DependencyHealth` to `StoreBundle` rather than adding `Ping`
to every domain store interface. `storeBuilder` already deduplicates SQL
handles by `(driver, DSN)` in `dbs`; append exactly one health dependency when
a new DB is opened. Memory stores add none. This preserves domain interfaces
and avoids six pings when logical stores share one database.

~~~go
type DependencyHealth interface {
    Name() string                 // safe label: "sqlite" or "programauth-db"
    CheckHealth(context.Context) error
}

type sqlHealth struct { name string; db *sql.DB }
func (h sqlHealth) CheckHealth(ctx context.Context) error { return h.db.PingContext(ctx) }
~~~

Construct `readinessHandler` with the resolved safe topology and a health
checker, not a precomputed `Ready: true` report. Give the entire operation a
short timeout (two seconds is a reasonable initial default). Check unique
dependencies concurrently, collect only component names/statuses, return 200
only if all required checks pass, and otherwise return 503.

~~~text
readyz(request):
  ctx = timeout(request.Context, 2 seconds)
  run each unique SQL PingContext concurrently
  gather safe {name, healthy} results
  if every required dependency passed: write 200 {ready:true,...}
  else: write 503 {ready:false,...}
~~~

Never emit a DSN, raw driver error, issuer URL, token, or client secret. A
server-side log may include a wrapped error only if its logging/redaction
policy is known safe.

### Decision: health lives on the store bundle

- **Context:** readiness needs live SQL evidence, but the existing store
  interfaces model domain operations rather than infrastructure.
- **Options considered:** add `Ping` to every store interface; reopen DSNs per
  readiness request; track health once in `StoreBundle` from shared `*sql.DB`s.
- **Decision:** retain optional health capabilities in `StoreBundle`.
- **Rationale:** it deduplicates database checks and avoids contaminating
  memory/domain test doubles with transport concerns.
- **Consequences:** builders must preserve health entries beside closers, and
  readiness tests should cover shared and distinct handles.
- **Status:** proposed.

## 5. End-to-end flows

### 5.1 Public device start

~~~text
CLI -> Traefik -> request-identity middleware -> native start handler
       |                                              |
       |                                              +-- IP budget (429 if exhausted)
       |                                              +-- strict JSON decode
       |                                              +-- allowlist validation
       |                                              +-- configured verification URI
       |                                              +-- persist hashed codes/grants
       v                                              v
CLI <- device_code + user_code + trusted verification URI
~~~

No handler should log either raw code. Audit and security events use bounded
outcomes such as `issued`, `invalid_scope`, or `rate_limited`, never a
credential or client-provided action as a metric label.

### 5.2 Browser decision

~~~text
browser local session -> POST request (CSRF) -> pending redacted view
browser local session -> POST approve (CSRF) -> agent created with stored grants
browser local session -> POST deny    (CSRF) -> terminal denied transition
CLI poll -> token pair once OR access_denied after denial
~~~

The service/store transition, not UI behavior, protects terminal state and
one-use consumption. Retain existing concurrent-redemption test coverage and
add handler-level tests for the new contract.

### 5.3 Readiness during outage

~~~text
Kubernetes probe --> /auth/readyz --> bounded PingContext on each unique DB
                                      |
                       success -------+------> 200 ready:true
                       timeout/error -+------> 503 ready:false

Kubernetes liveness --> /healthz --------------------> 200 while process runs
~~~

An IdP outage is not a hard readiness dependency in this ticket. Existing
local sessions and application access tokens do not need an IdP network call;
login is already dependent on issuer discovery at construction. Treating the
IdP as a separate degraded signal is a later product decision.

## 6. File-by-file implementation plan

### Phase 0 — Read and freeze contracts

1. Read `config.go`, `resolve.go`, `preflight.go`, `stores.go`, `builder.go`,
   and `services.go` in `pkg/xgoja/hostauth/`.
2. Read `device.go`, `device_handlers.go`, and `oauth_token.go` in
   `pkg/gojahttp/auth/programauth/`.
3. Read `ratelimit.go`, `request_response.go`, `access_log.go`, and
   `auth/audit/audit.go` in `pkg/gojahttp/`.
4. Capture existing behavior with tests before changing public JSON fields.

Do not edit tiny-idp or cluster manifests in this repository. Deployment
manifests must configure a real Traefik source CIDR and prevent direct pod
access, but those changes belong to the GitOps repository.

### Phase 1 — Request identity

1. Add config/resolution validation and a compiled CIDR representation.
2. Add generic identity context helpers and a concrete trusted-forwarded
   resolver in `gojahttp`.
3. Expose resolver/middleware capability through `hostauth.Services`.
4. Wrap the entire mux in `buildServeHandler`.
5. Migrate `NewRequestDTO`, limiter, audit, and access log consumers.
6. Add tests for direct, forged forwarded, trusted single-hop, trusted
   multi-hop, untrusted peer, malformed header, IPv4, and IPv6 cases.

**Exit criterion:** an untrusted peer cannot choose audit or limiter identity,
and a Traefik request produces matching audit and limiter client values.

### Phase 2 — Device boundary

1. Add and resolve `DeviceConfig`; update Glazed fields and config reference.
2. Remove `verificationUri` and approval action/tenant fields from public JSON.
3. Validate the action allowlist before persistence and derive the verification
   URI from resolved public configuration.
4. Add a small native-limit helper that reuses `gojahttp.RateLimiter`; make
   native handlers return 429/`Retry-After` before credential lookup.
5. Add request inspection and denial service/handlers; mount them.
6. Update inbox Step 08 UI/client/smoke to inspect then approve or deny.

**Exit criterion:** only declared local actions can be issued, no client picks
the verification origin, every native path has a budget, and denial is
session/CSRF-protected and terminal.

### Phase 3 — Readiness

1. Record one safe `sqlHealth` entry per newly opened shared DB handle.
2. Make `StoreBundle` own health checks along with closers.
3. Replace static readiness closure with dynamic bounded checks and safe result
   objects.
4. Mount `/healthz`; preserve or document `/auth/readyz` as the readiness URL.
5. Test initial outage, runtime failure, recovery, timeout, shared-handle
   deduplication, and secret-free response bodies.

**Exit criterion:** required SQL failure turns readiness to 503 and recovery
turns it back to 200 without a process restart; liveness remains responsive.

### Phase 4 — Integration and release evidence

1. Update the hostauth config help and reference example with `single-node`,
   proxy mode/CIDRs, durable stores, and pre-applied migrations.
2. Add a reverse-proxy integration test or a minimal Traefik-shaped test
   harness that proves actual header order assumed by Phase 1.
3. Run focused package, race, full test, build, and lint commands.
4. Add exact commands/results to the diary and deployment runbook.

## 7. Testing and validation

### Focused tests

~~~bash
go test ./pkg/gojahttp -count=1
go test ./pkg/gojahttp/auth/audit -count=1
go test ./pkg/gojahttp/auth/programauth -count=1
go test ./pkg/gojahttp/auth/programauth/sqlstore -count=1
go test ./pkg/xgoja/hostauth -count=1
go test ./pkg/xgoja/providers/http -count=1

go test -race ./pkg/gojahttp ./pkg/gojahttp/auth/programauth/... ./pkg/xgoja/hostauth -count=1

go fmt ./...
go test ./...
go build ./...
golangci-lint run -v
~~~

### Required negative and concurrency cases

| Area | Case | Expected result |
|---|---|---|
| Proxy | Direct peer sends forged XFF | peer remains client IP |
| Proxy | Trusted peer supplies valid chain | resolver selects right-to-left first untrusted client |
| Proxy | Untrusted peer supplies chain | headers ignored |
| Proxy | Trusted malformed/too-long chain | documented fail-closed error, no fallback trust |
| Device | Empty/unknown/duplicate action | `invalid_scope`, no persisted device record |
| Device | Client provides `verificationUri` | strict decoder rejects it |
| Device | Start flood | 429 before a durable row is created |
| Device | Fast valid poll | RFC 8628 `slow_down`, not transport 429 unless outer budget is exhausted |
| Device | Unknown code flood | 429 without an existence oracle |
| Device | Inspection | redacted response has no device code/hash/token data |
| Device | Denial then poll | terminal `access_denied` |
| Device | Two redemptions | exactly one token pair |
| Readiness | SQL unreachable | 503 within deadline, response has no DSN |
| Readiness | SQL recovers | returns 200 without restart |
| Readiness | SQL unavailable | `/healthz` remains 200 |

### Manual deployment proof

A unit test cannot prove an ingress controller's precise forwarding behavior.
In the cluster repository, deploy a single replica with a NetworkPolicy that
allows ingress only from Traefik, configure its actual source CIDR, and send
requests through Traefik. Compare the expected IP in an audit record with the
rate-limit bucket/test observation. Also restart the host between OIDC login
and callback, run device approval/refresh/revoke, stop SQL, and confirm probe
transitions.

## 8. Risks, compatibility, and review focus

- **Forwarded-header format is deployment-specific.** Keep `Forwarded` header
  support out of the first patch unless the cluster proves it is needed. Test
  the exactly deployed Traefik configuration.
- **Changing JSON is a deliberate breaking change.** Removing caller-selected
  verification URI and approval grants requires updating tutorial clients and
  any consumers. Prefer a clear release note over a compatibility branch that
  keeps the unsafe field alive.
- **Limiter is still process-local.** Limits protect the declared one-process
  topology only; never claim distributed abuse resistance.
- **Database ping is not a transaction proof.** It is the correct readiness
  baseline, but it cannot prove every future query/migration will succeed.
- **CSRF must remain checked before state transition.** Review new handlers for
  exact reuse of the existing session manager, not a copied cookie parser.
- **No secrets in observability.** Review JSON responses, audit metadata,
  security event labels, errors, and access logs after each new endpoint.

## 9. References

### Repository files

- `pkg/xgoja/providers/http/serve.go:448-476` — native handler mux order.
- `pkg/xgoja/hostauth/config.go:41-145` — host configuration vocabulary.
- `pkg/xgoja/hostauth/resolve.go:28-71` — config resolution flow.
- `pkg/xgoja/hostauth/preflight.go:29-56` — single-node guardrails.
- `pkg/xgoja/hostauth/builder.go:83-189` — service/native-handler composition.
- `pkg/xgoja/hostauth/stores.go:37-104,258-273` — shared SQL handle ownership.
- `pkg/xgoja/hostauth/readiness.go:5-40` — current static readiness behavior.
- `pkg/gojahttp/ratelimit.go:240-377` — planned-route limits and IP key.
- `pkg/gojahttp/request_response.go:68-73` — current DTO IP projection.
- `pkg/gojahttp/auth/audit/audit.go:358-371` — current unsafe XFF handling.
- `pkg/gojahttp/auth/programauth/device_handlers.go:39-215` — native HTTP
  contract and session/CSRF sequence.
- `pkg/gojahttp/auth/programauth/device.go:138-305` — device state machine.
- `pkg/gojahttp/auth/programauth/oauth_token.go:1-250` — rotating token model.
- `examples/xgoja/23-personal-knowledge-inbox/08-device-authorization/` —
  reference app and client/UI contract to update.

### Standards

- [RFC 8628: OAuth 2.0 Device Authorization Grant](https://www.rfc-editor.org/rfc/rfc8628)
- [RFC 7009: OAuth 2.0 Token Revocation](https://www.rfc-editor.org/rfc/rfc7009)

### Source review

- `/home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/18/TINYIDP-PROD-XGOJA-REVIEW-001--production-tiny-idp-review-for-multi-user-xgoja-applications-and-coding-agents/design-doc/03-pr-98-production-hardening-implementation-guide-for-xgoja-hostauth.md`
