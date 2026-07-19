---
Title: Native tiny-idp resource-server contract and oidcresource design
Ticket: XGOJA-TINYIDP-PROD-AUTH
Status: active
Topics:
    - security
    - oidc
    - auth
    - api-design
    - xgoja
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/provider.go
      Note: |-
        Current UserInfo endpoint and internal Fosite introspection use.
        UserInfo behavior and internal token introspection evidence
    - Path: /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/oidcmeta/discovery.go
      Note: |-
        Current discovery contract; it lacks an introspection endpoint.
        Discovery evidence for missing public introspection endpoint
    - Path: /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/pkg/idpstore/types.go
      Note: Access tokens are opaque and server-side.
    - Path: repo://pkg/gojahttp/auth/programauth/composite.go
      Note: |-
        Current composition point for local session and application-owned bearer authentication.
        Existing local bearer composition boundary
    - Path: repo://pkg/gojahttp/auth_plan.go
      Note: |-
        ResultAuthenticator contract for planned-route authentication.
        Target ResultAuthenticator boundary
    - Path: repo://ttmp/2026/07/15/XGOJA-TINYIDP-PROD-AUTH--production-tiny-idp-integration-and-durable-xgoja-oidc-host-auth/scripts/04-probe-tinyidp-resource-contract.sh
      Note: Credential-free discovery contract probe
ExternalSources: []
Summary: Decision record and implementation design for accepting tiny-idp tokens as resource-server credentials without conflating them with application-owned programauth tokens.
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# Native tiny-idp resource-server contract and `oidcresource` design

## Decision

Do not accept a tiny-idp access token at an xgoja API today. The current
tiny-idp production contract has opaque access tokens and does not advertise a
public OAuth token-introspection endpoint. Its `/userinfo` endpoint is a user
identity endpoint, not a resource-server contract: it returns `sub` and profile
claims after the bearer token has been accepted, but it does not provide the
resource server with an authenticated `client_id`, `aud`, granted `scope`,
expiry, token type, or confirmation (`cnf`) binding.

The correct Phase 5 design is therefore an introspection-based
`oidcresource.Authenticator` that is **disabled until tiny-idp supplies the
matching public contract**. JWT validation is explicitly rejected for the
current provider because the provider documents and stores access tokens as
opaque values. JWKS verifies ID tokens; it cannot verify an opaque bearer
credential.

This is a security boundary, not a missing convenience adapter. Treating
UserInfo success as authorization would let a token issued for an unspecified
client/resource enter an API without audience or scope enforcement.

## Evidence from the current provider

`pkg/idpstore.AccessToken` says that an access token is an opaque server-side
record. `internal/oidcmeta.ProductionDiscovery` publishes authorization, token,
UserInfo, JWKS, and end-session endpoints, but no `introspection_endpoint`.
The `/userinfo` implementation calls Fosite’s internal `IntrospectToken`, then
returns a claims map beginning with `sub`; it deliberately does not emit the
Fosite requester’s client, audience, requested scopes, expiry, or DPoP
confirmation metadata.

```text
current, supported browser path

xgoja callback -- code + PKCE --> tiny-idp /token
xgoja callback -- verified ID token --> local session

current, unsupported API path

external caller -- opaque tiny-idp access token --> xgoja route
                                      ^
                       no issuer-facing validation/audience contract
```

The internal Fosite introspection factory is not a public HTTP API. A generated
host must not share tiny-idp’s database, invoke an internal Go package, or
scrape the UserInfo response to circumvent this boundary.

## Required tiny-idp contract

Tiny-idp should expose RFC 7662-style introspection through discovery:

```json
{
  "introspection_endpoint": "https://idp.example.test/introspect",
  "introspection_endpoint_auth_methods_supported": [
    "client_secret_basic",
    "private_key_jwt"
  ]
}
```

The resource server must authenticate itself. A public caller must not be able
to turn the endpoint into a token-validity oracle. The resource server is a
registered confidential client with an explicit resource/audience policy.

For an active access token, the response must contain at least:

```json
{
  "active": true,
  "iss": "https://idp.example.test",
  "sub": "user-alice-fixed",
  "client_id": "personal-inbox-cli",
  "aud": ["https://inbox.example.test/api"],
  "scope": "openid inbox.read inbox.write",
  "exp": 1780000000,
  "iat": 1779996400,
  "token_type": "Bearer"
}
```

For an inactive, expired, revoked, malformed, or unknown token it returns only
`{"active": false}` with normal cache-control rules. It must never include a
raw token in errors or audit fields. Tiny-idp must define whether `aud` is a
string or list and preserve that shape consistently; this design accepts either
only at decoding, then normalizes to a set.

For DPoP-bound tokens, include `token_type: "DPoP"` and the confirmation
thumbprint:

```json
{"active": true, "token_type": "DPoP", "cnf": {"jkt": "..."}}
```

The resource server, not the introspection endpoint, validates the fresh DPoP
proof bound to the current HTTP method and public URL. Tiny-idp’s existing
UserInfo replay cache cannot protect a separate xgoja API endpoint.

## `oidcresource` package design

The package belongs at `pkg/gojahttp/auth/oidcresource`. It implements
`gojahttp.ResultAuthenticator` and is added only when a host config explicitly
enables the capability. It never replaces `programauth`; composition chooses a
known local bearer prefix first and an external issuer token only under the
configured resource-server path.

```go
type Config struct {
    IssuerURL            string
    IntrospectionURL     string // discovered, then issuer-bound
    ResourceClientID     string
    ResourceClientSecret string
    ExpectedAudience     string
    RequiredScopes       []string
    HTTPClient           *http.Client
    Cache                ActiveTokenCache
    Clock                func() time.Time
    DPoP                 DPoPVerifier // nil rejects DPoP token_type
    SubjectMapper        SubjectMapper
}

type SubjectMapper interface {
    MapOIDCSubject(ctx context.Context, issuer, subject string) (gojahttp.Actor, error)
}
```

The configuration must reject insecure issuer URLs outside development,
missing expected audience, missing confidential-client authentication, an
introspection URL outside the issuer origin/path policy, and a DPoP-tolerant
setting without a DPoP verifier. It must not accept arbitrary issuer URLs from
an HTTP request.

### Authentication algorithm

```pseudocode
authenticate(request, route security spec):
    raw = parse one Authorization header
    reject duplicate, query, or form token transports

    response = cache.lookup(hash(raw))
    if response absent:
        response = POST introspection with resource-client authentication
        cache only active response, bounded by min(exp-now, 60 seconds)

    reject if active is false
    reject if issuer != configured issuer
    reject if expected audience absent
    reject if response expiry is absent or expired
    reject if required scopes are absent

    if token_type == DPoP:
        verify proof HTTP method, canonical public URL, jkt, ath, iat, jti replay
    else if token_type != Bearer:
        reject

    actor = subject mapper(issuer, sub)
    grants = map configured scope-to-grant rules, then intersect route policy
    return AuthResult(actor, external-token method, no CSRF, grants)
```

The cache key is a keyed hash of the raw token. The raw string is never stored,
logged, used as a metric label, or passed to JavaScript. Cache entries are
invalidated on expiry and must be bounded in size. A resource server should use
a short positive cache only; it must not cache inactive results long enough to
create a denial-of-service amplification or cache successful results beyond
revocation latency.

## Authorization mapping

Scopes are not local grants. The host configuration supplies the explicit
mapping, for example:

```yaml
external-resource-auth:
  expected-audience: https://inbox.example.test/api
  scope-grants:
    inbox.read:  [message.read]
    inbox.write: [message.create]
```

The route enforcer still resolves local resources and applies local ownership
and tenant rules. A token that has `inbox.write` cannot create a message for a
different user merely because its IdP subject is valid. Subject mapping must
resolve the same local user created by browser OIDC login and must fail closed
for an unknown or disabled local account.

## Negative test matrix

| Case | Required result |
| --- | --- |
| valid active token, matching audience/scope | authenticated local actor with mapped grants |
| inactive/unknown token | unauthenticated; no validity detail to caller |
| wrong issuer | unauthenticated |
| wrong or absent audience | unauthenticated |
| absent/insufficient scope | authenticated identity may exist, but planned route authorization is denied |
| expired or revoked token | unauthenticated after cache expiry / bounded TTL |
| resource-client authentication failure | fail closed; server error observable without token leakage |
| DPoP token with no/malformed/replayed/wrong-URL proof | unauthenticated |
| duplicate Authorization or query/form transport | invalid request, matching tiny-idp transport hygiene |
| subject for disabled/nonexistent local user | unauthenticated |

The strict tiny-idp fixture must obtain a device token, call an external-token
protected API route, revoke or expire it, and repeat the request. It must run
the wrong-audience/scope and DPoP tests only after tiny-idp implements the
contract above.

## Implementation status and handoff

Tasks 23, 24, and 26 are completed as a design decision: introspection is the
required contract, `oidcresource` has an API/algorithm/test design, and it is
kept separate from application-owned `programauth`. Task 25 remains blocked by
the provider’s missing public introspection/audience/scope/confirmation
contract. Do not add a UserInfo-based authenticator as a shortcut.

The next engineering unit belongs in tiny-idp: add an authenticated discovery
advertised introspection endpoint, contract tests for active/inactive and DPoP
metadata, client/resource registration, audit redaction, and revocation
semantics. Once that lands, return to this ticket to implement the package and
strict cross-project smoke.
