# Security policy

## Scope

`go-go-goja` is a Go library and toolchain for embedding JavaScript, exposing native Go modules, and building HTTP hosts. It is not a security boundary by itself and it is not a complete healthcare application.

Some modules intentionally provide powerful host capabilities, including file-system access, command execution, database access, networking, and access to host configuration. Code that executes untrusted or tenant-supplied JavaScript must use an explicit allowlist or `engine.MiddlewareSafe()` and must not expose `exec`, `fs`, `database`, `process.env`, or equivalent host capabilities unless the deployment threat model permits them.

The Ségur numérique applicability assessment and evidence checklist are documented in [`docs/compliance/segur-v1-v2.md`](docs/compliance/segur-v1-v2.md). That document does not claim ANS certification or Ségur referencing.

## Supported versions

Security fixes are developed against the default branch and the latest published release. Older releases may receive fixes when practical, but no long-term support period is implied unless a release announcement states otherwise.

Users should reproduce reported issues against the latest release or the current default branch before reporting them.

## Reporting a vulnerability

Do not open a public issue for a suspected vulnerability.

1. Use GitHub private vulnerability reporting when the repository Security tab exposes **Report a vulnerability**.
2. If private reporting is unavailable, contact a repository maintainer privately through the contact information on their GitHub profile or through an established private organization channel.
3. Include only the minimum sensitive information needed to reproduce the issue. Do not include production credentials, patient data, access tokens, session cookies, private keys, or unredacted logs.

A useful report contains:

- affected commit, release, package, command, or generated-host configuration;
- deployment assumptions and whether JavaScript input is trusted;
- reproduction steps or a minimal proof of concept;
- expected and observed behavior;
- impact, required privileges, and reachable data or capabilities;
- suggested mitigation, when known;
- whether the issue is already public or under an embargo.

## Triage and disclosure

Maintainers will validate the report, determine affected versions, and coordinate remediation and disclosure with the reporter. Response times depend on severity, reproducibility, and maintainer availability; this policy does not establish a contractual service-level agreement.

When a vulnerability is confirmed, the remediation process should produce:

- a private reproduction and regression test;
- a minimal fix that preserves the intended trust boundary;
- a review of adjacent call paths and generated code;
- updated documentation or deployment guidance where configuration contributed to the issue;
- a release or advisory describing affected versions and mitigations without exposing secrets.

## Security expectations for contributors

Contributions must follow [`docs/security/secure-development.md`](docs/security/secure-development.md). In particular:

- security decisions that can be enforced by the Go host must not be delegated to JavaScript;
- new native modules must document their host capabilities and be excluded from safe-mode defaults unless they are data-only;
- authentication, authorization, CSRF, rate limiting, and audit enforcement must fail closed;
- secrets and bearer credentials must not be placed in logs, audit attributes, errors, tests, examples, fixtures, or generated artifacts;
- third-party tools and GitHub Actions must use immutable or versioned references rather than mutable development branches;
- scanner suppressions require a narrow, documented justification and a regression test when applicable.

## Deployment responsibilities

Operators remain responsible for the security of the complete deployed system, including TLS termination, trusted-proxy configuration, identity-provider policy, secret management, database access, backups, restore tests, log retention, monitoring, incident response, vulnerability remediation, and penetration testing.

A deployment that processes French health data must separately assess all applicable legal, regulatory, contractual, interoperability, hosting, and Ségur requirements. The presence of security features in this repository is not evidence that a derived product is compliant, certified, HDS-hosted, or referenced by the Agence du Numérique en Santé.
