---
Title: Keycloak OIDC session authorization host notes
Ticket: XGOJA-HOST-AUTH
Status: active
Topics:
    - goja
    - http
    - security
    - xgoja
    - keycloak
    - oidc
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources:
    - /tmp/auth2.md
Summary: Imported source notes for an opinionated Go host-side user/auth system using Keycloak OIDC, server-side sessions, app-owned authorization, capabilities, and audit.
LastUpdated: 2026-06-12T17:00:00-04:00
WhatFor: Source material for planning the robust production and dev/demo host auth packages behind gojahttp planned routes.
WhenToUse: Read when implementing login/logout, sessions, Keycloak integration, user normalization, authorization, capabilities, and audit for go-go-goja hosts.
---

Use **Keycloak as your IdP**. Do **not** write your own unless identity itself is your product. Building login, MFA, recovery, token issuance, key rotation, federation, and audit correctly is a large security surface.

For a Go web app, I would separate the problem into four layers:

## 1. Authentication: use OIDC, not “custom login”

Learn:

* **OAuth 2.0 vs OpenID Connect**: OAuth is delegated access; OIDC is authentication on top of OAuth. OIDC lets your app verify who the user is. ([OpenID Foundation][1])
* **Authorization Code Flow + PKCE**: this is the flow you want for browser login. Keycloak documents Authorization Code flow as the web-app-oriented OIDC flow that returns an authorization code, then your app exchanges it for tokens. ([Keycloak][2])
* **Avoid Implicit Flow and Resource Owner Password Credentials**. RFC 9700 says clients should use code flow instead of implicit flow, and password grant “MUST NOT” be used. ([IETF Datatracker][3]) ([IETF Datatracker][3])

For Go, look at:

* `golang.org/x/oauth2`
* `github.com/coreos/go-oidc/v3/oidc`, which implements OIDC client logic for Go and integrates with `x/oauth2`. ([GitHub][4])

Recommended browser architecture:

```text
Browser
  -> your Go app
      -> redirects to Keycloak
      <- callback with authorization code
      -> Go app exchanges code for tokens
      -> Go app creates its own server-side session
Browser
  <- receives only your app session cookie
```

Do **not** store Keycloak access tokens in `localStorage`.

## 2. Session management: keep app sessions separate from IdP tokens

For a normal server-rendered Go app or BFF-style app, use an **opaque server-side session cookie** for the browser. Keep Keycloak tokens server-side, attached to the session if you need them.

Use:

* `Secure`
* `HttpOnly`
* `SameSite=Lax` or `Strict`
* short idle timeout
* absolute timeout
* session ID regeneration after login / privilege elevation
* server-side invalidation on logout

OWASP recommends cookies for session ID exchange, CSPRNG-generated meaningless session IDs, server-side session state, HTTPS, `Secure`, session renewal after privilege changes, idle timeouts, and absolute timeouts. ([OWASP Cheat Sheet Series][5])

In Go, `alexedwards/scs` is a solid package to inspect; it supports server-side stores, token regeneration, idle timeout, absolute timeout, and common backends like PostgreSQL and Redis. ([Go Packages][6])

A good default:

```text
session cookie: __Host-app
store: Redis or Postgres
idle timeout: 30–60 minutes
absolute timeout: 8–24 hours
remember-me: separate, longer-lived, revocable token
```

## 3. Authorization: do not confuse roles with permissions

This is the part people usually get wrong.

Use Keycloak for:

```text
identity
login
MFA
groups
coarse roles
account lifecycle
federation
```

Use your app for:

```text
object ownership
tenant membership
document/project/resource permissions
business rules
workflow states
billing/account limits
per-resource sharing
```

Do not rely on “user has role admin/editor/viewer” for everything. Role checks are often too coarse and cause horizontal privilege bugs.

Model authorization like this:

```go
Authorize(ctx, subject, action, resource, context) Decision
```

Example:

```text
subject: user:123
action: project.delete
resource: project:456
context: tenant=acme, mfa_age=3m, ip_country=DE
```

Then enforce it **server-side on every request**. OWASP explicitly recommends deny-by-default, least privilege, validating permissions on every request, and not relying on client-side authorization checks. ([OWASP Cheat Sheet Series][7]) ([OWASP Cheat Sheet Series][7]) ([OWASP Cheat Sheet Series][7])

Start simple:

```text
users
tenants
memberships: user_id, tenant_id, role
resources: tenant_id, owner_id, ...
permission checks in Go code
```

Then graduate to a policy engine when the rules become hard to audit.

Options:

* **Casbin**: good embedded Go authorization library; supports ACL, RBAC, ABAC, ReBAC, and custom models. ([casbin.org][8])
* **OpenFGA**: good if you need Google-Drive-like sharing, inherited access, groups, folders, teams, parent-child resources, etc. It uses relationship-based authorization. ([openfga.dev][9])
* **OPA/Rego**: good for centralized policy-as-code across services, infra, gateways, and APIs. ([Open Policy Agent][10])
* **Cedar**: worth studying for readable application authorization policies; it is a language/spec for evaluating permission policies. ([cedarpolicy.com][11])

My bias: for a Go monolith, start with **explicit Go authorization functions + tests**. Add Casbin/OpenFGA/OPA only when the authorization model is complex enough to justify the machinery.

## 4. Capabilities: use them for narrow, intentional delegation

A capability is not just a permission name. It is closer to: “possession of this unguessable token grants this specific authority.”

Good uses:

```text
password reset link
email verification link
invite link
share link
one-time file download
API token scoped to one integration
temporary upload URL
```

Capability tokens should be:

```text
random or signed
scoped to one action/resource
time-limited
revocable if possible
audited
single-use where appropriate
never logged
never placed in Referer-leaking contexts
```

Do not replace your whole permission system with bearer capability links unless the application is intentionally capability-based.

## Concrete implementation plan

1. **Keep Keycloak.**
   Configure one OIDC client for your Go app. Use Authorization Code Flow with PKCE. Disable Direct Access Grants / password grant unless you have a very specific machine-only reason.

2. **Use Go as a backend-for-frontend.**
   Browser gets your app cookie, not Keycloak tokens. Your Go app validates OIDC callback, stores user identity in a server-side session, and optionally stores refresh/access tokens server-side.

3. **Normalize the Keycloak user into your DB.**

   ```text
   app_user:
     id
     keycloak_sub
     email
     display_name
     created_at
     disabled_at
   ```

   Treat `sub` as the stable external identity key, not email.

4. **Create a central authz package.**

   ```go
   type Action string

   const (
       ProjectRead   Action = "project.read"
       ProjectUpdate Action = "project.update"
       ProjectDelete Action = "project.delete"
   )

   type Authorizer interface {
       Can(ctx context.Context, user User, action Action, resource Resource) (bool, error)
   }
   ```

5. **Make every protected handler follow this shape.**

   ```text
   authenticate session
   load user
   load resource from DB
   authorize(user, action, resource)
   execute business operation
   audit important writes
   ```

6. **Fail closed.**
   Missing user, missing tenant, missing policy, missing resource ownership, or policy engine error should normally be deny.

7. **Write negative authorization tests.**
   Test that user A cannot read/update/delete user B’s resource, even if they guess the ID.

8. **Audit security events.**

   Log:

   ```text
   login success/failure
   logout
   session revoked
   MFA changes
   role/membership changes
   permission denied for sensitive actions
   admin actions
   API token creation/use/revocation
   ```

   Do not log session IDs, access tokens, refresh tokens, authorization codes, or capability tokens.

## What to learn, in order

1. **HTTP cookies and CSRF**
2. **OAuth 2.0 roles and flows**
3. **OpenID Connect Core**
4. **Keycloak realms, clients, scopes, roles, groups, protocol mappers**
5. **Session fixation, session rotation, idle/absolute timeout**
6. **RBAC vs ABAC vs ReBAC**
7. **OWASP ASVS sections on authentication, session management, and access control**
8. **Policy testing**
9. **Audit logging and incident response**
10. **Optional: OpenFGA / Zanzibar-style authorization**

Keycloak Authorization Services can model fine-grained authorization with resources, scopes, policies, and permissions, but I would not start there unless you specifically want authorization decisions externalized into Keycloak. Keycloak documents that enabling Authorization Services turns an OIDC client into a resource server with resources, authorization scopes, policies, and permissions. ([Keycloak][12])

## Practical default stack

```text
IdP: Keycloak
Protocol: OIDC Authorization Code + PKCE
Go OIDC: coreos/go-oidc + x/oauth2
Browser session: server-side session via scs
Session store: Redis or Postgres
Authz v1: explicit Go policy functions
Authz v2: Casbin if rules become generic
Authz v3: OpenFGA if permissions become relationship-heavy
Security reference: OWASP ASVS + OWASP Cheat Sheets
```

The main design rule: **Keycloak authenticates the user; your app authorizes the action on the specific resource.**

[1]: https://openid.net/specs/openid-connect-core-1_0.html?utm_source=chatgpt.com "OpenID Connect Core 1.0 incorporating errata set 2"
[2]: https://www.keycloak.org/securing-apps/oidc-layers "Securing applications and services with OpenID Connect - Keycloak"
[3]: https://datatracker.ietf.org/doc/rfc9700/ "
            
        RFC 9700 - Best Current Practice for OAuth 2.0 Security

        "
[4]: https://github.com/coreos/go-oidc?utm_source=chatgpt.com "coreos/go-oidc: A Go OpenID Connect client."
[5]: https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html "Session Management - OWASP Cheat Sheet Series"
[6]: https://pkg.go.dev/github.com/alexedwards/scs/v2?utm_source=chatgpt.com "scs package - github.com/alexedwards/scs/v2"
[7]: https://cheatsheetseries.owasp.org/cheatsheets/Authorization_Cheat_Sheet.html "Authorization - OWASP Cheat Sheet Series"
[8]: https://casbin.org/?utm_source=chatgpt.com "Apache Casbin · An authorization library | Apache Casbin ..."
[9]: https://openfga.dev/docs/authorization-concepts?utm_source=chatgpt.com "Fine-Grained Authorization, ReBAC, ABAC & Zanzibar ..."
[10]: https://openpolicyagent.org/docs?utm_source=chatgpt.com "Open Policy Agent (OPA)"
[11]: https://cedarpolicy.com/?utm_source=chatgpt.com "Cedar Language"
[12]: https://www.keycloak.org/docs/latest/authorization_services/index.html "Authorization Services Guide"

