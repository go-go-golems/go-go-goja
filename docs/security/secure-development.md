# Secure development standard

This document defines the repository's minimum secure-development rules. It is intended to support routine engineering and evidence collection; it is not a substitute for a product-specific threat model, independent security assessment, or regulatory review.

## 1. Define the trust boundary before coding

Every feature must identify:

- who controls the JavaScript, HTTP request, configuration, database content, and generated artifacts;
- which host resources can be reached;
- which identities and tenants are involved;
- what must fail closed when a dependency is unavailable;
- which security events and audit records are required.

`goja` is an execution engine, not an isolation boundary for hostile code. A runtime that exposes native modules can reach the privileges of its host process.

For untrusted or tenant-supplied JavaScript:

- start with `engine.MiddlewareSafe()` or an explicit `engine.MiddlewareOnly(...)` allowlist;
- do not expose `exec`, `fs`, `database`, `process.env`, unrestricted HTTP clients, or plugin loading without a documented capability model;
- use a dedicated operating-system identity, constrained filesystem, network policy, resource limits, and process supervision;
- assume that any value returned to JavaScript can be copied or exfiltrated.

A new native module must document its capability class:

- **data-only**: transforms supplied values without accessing host resources;
- **bounded host access**: accesses an explicitly scoped resource;
- **privileged host access**: executes commands, reads arbitrary files or environment variables, opens arbitrary connections, or modifies host state.

Only data-only modules are candidates for safe-mode defaults.

## 2. Keep security enforcement in Go

Authentication, authorization, CSRF protection, rate limiting, resource resolution, and audit dispatch must be enforced by the Go host before a JavaScript handler runs.

Planned HTTP routes must:

- declare `.public()` or an authenticated mode explicitly;
- require a host-owned authorization action for authenticated routes;
- resolve tenant and resource identifiers through typed host-owned logic;
- require CSRF protection for state-changing browser-session requests;
- emit a stable audit event for security-sensitive operations;
- reject missing security services rather than silently disabling enforcement.

Do not trust JavaScript to assert that a request is authenticated, that a user owns a resource, or that a client-supplied tenant identifier is authorized.

## 3. Authentication and session rules

Browser sessions must use server-side state and opaque, unpredictable identifiers. Production cookies must be `Secure`, `HttpOnly`, use an appropriate `SameSite` policy, and have explicit idle and absolute expiry.

OIDC browser login must validate issuer, audience/client ID, signature, expiry, state, nonce, and PKCE. Identity mapping must use the stable issuer/subject pair; email addresses and display names are attributes, not durable identity keys.

Logout and other state-changing session operations must require CSRF verification. Session rotation, revocation, expiry, and recent-MFA requirements must have negative tests.

Programmatic credentials must be scoped, revocable, and excluded from request context, audit attributes, error messages, and logs in raw form.

## 4. Input, output, and resource handling

- Validate sizes, counts, formats, enumerations, and time ranges at the host boundary.
- Use structured arguments rather than shell command construction.
- Use parameterized database queries and bounded result sets.
- Normalize and validate paths before file access; prefer an application-owned root.
- Set request, database, subprocess, and shutdown deadlines.
- Propagate `context.Context` and stop background work when the runtime or request is closed.
- Avoid reflecting internal errors, stack traces, SQL text, filesystem paths, or identity-provider responses to clients.
- Treat generated code, templates, schemas, and embedded assets as security-relevant source.

## 5. Secrets and privacy

Never commit or log:

- passwords, private keys, client secrets, bearer tokens, refresh tokens, capability tokens, device codes, session IDs, cookies, authorization headers, or unredacted connection strings;
- production personal data or health data;
- full identity-provider responses when a minimal normalized claim set is sufficient.

Audit data must be minimized. Use stable identifiers only when operationally necessary, hash network identifiers when appropriate, and define retention and access control at the deployment level. Redaction must recurse into nested maps and lists.

Examples and tests must use synthetic data and non-working credentials.

## 6. Dependencies and build-chain controls

- Review changes to direct and transitive dependencies.
- Run `govulncheck` for reachable Go vulnerabilities and `gosec` for source-level issues.
- Run CodeQL and secret scanning on pull requests and on a schedule.
- Pin installed security tools to an intentional released version.
- Pin GitHub Actions to an immutable commit when practical; otherwise use a reviewed version tag, never a mutable development branch such as `main`.
- Keep lockfiles and generated dependency metadata under review.
- Record exceptions for scanner suppressions close to the affected code. Repository-wide exclusions must have an owner and a removal plan.

A green scanner result is evidence for a specific commit, not a general assertion that the system is secure.

## 7. Testing requirements

Security-sensitive changes require tests covering both acceptance and rejection paths. Depending on the feature, include:

- unauthenticated, wrong-tenant, wrong-resource, and insufficient-scope requests;
- expired, revoked, replayed, malformed, and rotated credentials;
- CSRF failures and unsafe HTTP-method handling;
- proxy-header spoofing and trusted-proxy boundaries;
- rate-limit bypass attempts and concurrency;
- oversized or malformed input;
- cancellation, timeout, partial failure, and dependency outage behavior;
- audit redaction and absence of secrets;
- backup/restore and migration compatibility for persistent security state;
- fuzzing for parsers, protocol adapters, and serialization boundaries.

Tests that prove an authorization invariant should state the invariant in the test name.

## 8. Review and release evidence

A security-relevant pull request should identify:

- the changed trust boundary;
- threat scenarios considered;
- controls added or changed;
- test and scanner evidence;
- migration, rollback, and operational effects;
- documentation updates;
- accepted residual risk.

For a release used in a regulated or health-data environment, retain an evidence bundle tied to the immutable commit or artifact digest:

- dependency inventory or SBOM;
- test, lint, CodeQL, `govulncheck`, `gosec`, and secret-scan results;
- reviewed scanner exceptions;
- build provenance, checksums, and signatures where available;
- configuration baseline and deployment architecture;
- migration and rollback records;
- backup and restore-test records;
- penetration-test report and remediation register;
- incident-response and vulnerability-disclosure procedures.

## 9. Operational baseline for generated auth hosts

Production generated hosts must use the repository's production preflight and runbook guidance. In particular:

- use durable stores for sessions, audit records, credentials, and OIDC transactions;
- apply schema migrations outside the serving process;
- enable secure cookies and HTTPS;
- trust forwarding headers only from measured proxy source ranges;
- separate liveness from SQL-backed readiness;
- manage secrets outside source control;
- test database backup, restore, and rollback;
- configure audit retention, monitoring, alerting, cleanup, and incident response.

See `cmd/xgoja/doc/23-auth-host-production-runbook.md` for the current deployment runbook.
