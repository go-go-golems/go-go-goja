# Ségur numérique Vague 1 / Vague 2 applicability and security baseline

**Assessment date:** 2026-07-22  
**Repository:** `go-go-golems/go-go-goja`  
**Assessment type:** source-repository gap analysis, not a conformity assessment

## Executive conclusion

`go-go-goja` is a general-purpose Go/JavaScript runtime, native-module framework, code generator, and HTTP-host toolkit. It is not a Logiciel de Gestion de Cabinet (LGC) and does not implement the minimum LGC product perimeter: patient records, appointments, prescriptions, medical decision support, healthcare document workflows, and exchanges with health services.

Therefore:

- this repository cannot, by itself, be referenced as a Ségur LGC solution;
- this document does not claim Ségur conformity, ANS referencing, HDS compliance, Pro Santé Connect approval, CNDA approval, or any healthcare certification;
- a derived healthcare product may reuse this repository as a technical component, but the complete candidate solution, its publisher, its operating model, and its evidence dossier must satisfy every applicable REM requirement and verification scenario;
- the reusable controls in this repository are relevant mainly to the **Sécurité des SI** portion of Vague 2 and to retained Vague 1 security expectations.

## Official research basis

The authoritative sources must be rechecked for each candidate version and submission date. This assessment used:

- ANS Ségur resources index: <https://esante.gouv.fr/ens/segur-numerique-sante/ressources-segur>
- ANS LGC Vague 2 device page: <https://esante.gouv.fr/ens/segur-numerique-sante/vague-2/dispositif-lgc-couloir-medecin-ville>
- REM LGC Vague 1: <https://esante.gouv.fr/sites/default/files/media_entity/documents/REM-MDV-LGC-Va1.xlsx>
- REM MDV-LGC Vague 2, version 08/06/2026: <https://industriels.esante.gouv.fr/sites/default/files/media/document/REM-MDV-LGC-Va2.xlsx>
- DSR MDV-LGC Vague 2, June 2026: <https://esante.gouv.fr/sites/default/files/media/document/DSR-MDV-LGC-Va2.pdf>
- ANS LGC penetration-test guide: <https://esante.gouv.fr/sites/default/files/media/document/ANS_Guide-utilisation_Test-intrusion_LGC.pdf>

The REM spreadsheet is the requirement-level source of truth. The DSR explains scope, evidence, and the referencing process. This repository document is only an engineering crosswalk.

## Vague 1 and Vague 2 relationship

The Vague 2 LGC profile retains applicable Vague 1 requirements and adds Vague 2 requirements. A product that was not previously referenced must not treat Vague 2 as a security-only delta: it must evaluate the full applicable Vague 1 and Vague 2 perimeter.

For a derived product, maintain a machine-readable traceability matrix with at least:

- REM requirement ID and version;
- applicability decision and rationale;
- implementation component and owner;
- verification scenario;
- automated test, manual test, or external evidence reference;
- result, date, immutable candidate version, and reviewer;
- residual finding and remediation status.

## Vague 2 security themes

The June 2026 DSR groups the security work around a common technical and organizational baseline, including:

- security governance, named responsibilities, and team awareness;
- a secure-development standard applied to new products and new functionality;
- continuous technology and threat monitoring;
- proactive identification and remediation of vulnerabilities, including penetration testing;
- preventive and corrective controls for common critical weaknesses;
- identity, authentication, access, privileged-account, login, and logout controls;
- backup and restore arrangements;
- creation and conservation of security-relevant traces.

The DSR associates these areas with REM identifiers including the `SC.SSI/GEN.*`, `SC.SSI/IAM.*`, and `SC.SSI/IE.*` families. Exact wording, applicability, proof type, and verification steps must be taken from the current REM rather than inferred from this summary.

## Repository crosswalk

Status values:

- **Implemented** - source control contains a reusable control and tests or CI evidence.
- **Added by this baseline** - repository governance/evidence added in the associated change.
- **Partial** - a useful primitive exists, but product or deployment controls remain necessary.
- **External/product obligation** - cannot be satisfied by this component repository.
- **Not applicable to this component** - health-domain feature outside the repository's role.

| Ségur-related control area | Repository evidence | Status | Remaining work for a candidate healthcare product |
| --- | --- | --- | --- |
| Security roles and governance (`SC.SSI/GEN.01` family) | `SECURITY.md`; this crosswalk; repository ownership and review history | Added by this baseline | Name accountable product-security, DPO/privacy, operations, incident, and regulatory owners; retain approval records. |
| Security awareness (`SC.SSI/GEN.02 BIS` family) | `docs/security/secure-development.md`; PR security checklist | Added by this baseline | Conduct and record role-specific training; include healthcare-data, identity, incident, and operational topics. |
| Secure-development guide (`SC.SSI/GEN.11`, `SC.SSI/GEN.20` family) | Secure-development standard; host-owned route enforcement; safe module selection; tests, lint, CodeQL, gosec | Implemented / added | Apply the standard through the complete product SDLC, generated code, client applications, integrations, and infrastructure. |
| Technology and threat watch (`SC.SSI/GEN.03 BIS` family) | Scheduled CodeQL, `govulncheck`, `gosec`, dependency review, secret scanning; dependency-update PRs | Partial | Assign an owner, define review frequency and severity targets, monitor CERT Santé/ANSSI/vendor sources, and retain remediation decisions. |
| Vulnerability management and proactive patching | `govulncheck`, gosec, CodeQL, dependency review; version-pinned scanner installation | Partial | Define remediation SLAs, asset/version inventory, supported-version policy, emergency patch process, exception register, and customer notification process. |
| Penetration testing | Testable HTTP/auth surfaces and deployment runbook | External/product obligation | Commission the ANS-required test against the complete candidate solution using the current ANS form and guide. The guide requires an organization qualified PASSI, while clarifying that the engagement itself need not be a formal PASSI audit. Remediate and retest findings. |
| Least privilege and runtime capability control | `engine.MiddlewareSafe()`, module allowlist/exclusion middleware, opt-in `process.env`, explicit documentation of privileged modules | Implemented | Enforce the selected capability policy in the product; add OS/container/network isolation and tenant-specific threat analysis. |
| Authentication and session security | Server-side opaque sessions, secure cookie defaults, idle/absolute expiry, CSRF, OIDC state/nonce/PKCE, revocation, recent-MFA support | Implemented / partial | Integrate the required healthcare identity services, including Pro Santé Connect when applicable; define privileged-account policy and operational access reviews. |
| Authorization and tenant isolation | Host-owned route plans, resource resolution, authorization actions, negative-path tests | Implemented / partial | Provide product-specific policy, exhaustive cross-tenant tests, administrative separation, and evidence for every sensitive operation. |
| Login/logout hardening | OIDC callback verification, CSRF-protected POST logout, session revocation and cookie clearing | Implemented / partial | Validate the complete UI and identity-provider configuration, account lifecycle, lockout/rate limiting, privileged sessions, and user messaging. |
| Audit trace generation | `pkg/gojahttp/auth/audit`, structured records, secret-key redaction, bounded queries, durable store interfaces | Implemented / partial | Define events, retention, integrity, access controls, export, monitoring, clock synchronization, legal basis, and deletion rules for the product. |
| Backup and restore | `cmd/xgoja/doc/23-auth-host-production-runbook.md` includes backup, isolated restore rehearsal, migration, and rollback guidance | Partial | Implement scheduled encrypted backups, off-site protection, recovery objectives, periodic restore evidence, key recovery, and healthcare-data obligations. |
| Secure deployment and operations | Single-node production profile, trusted-proxy configuration, durable stores, SQL readiness, external secret management guidance | Partial | Produce the candidate architecture, hardening baseline, HDS analysis where applicable, monitoring, incident response, business continuity, capacity, and supplier controls. |
| Software supply-chain evidence | Go module manifests, lockfiles, CI scanners, release checksums/signing mechanisms in release tooling | Partial | Generate and retain an SBOM for each candidate artifact, provenance, artifact digest, signature verification, dependency exception register, and reproducible evidence bundle. |
| Vulnerability disclosure and incident intake | `SECURITY.md` | Added by this baseline | Operate a private intake channel, incident classification, CERT Santé/authority notification analysis, customer communication, exercises, and post-incident review. |

## CI changes in this baseline

The associated repository change applies the following low-risk controls:

- explicit read-only default GitHub token permissions for test and security workflows;
- only the CodeQL job receives `security-events: write`;
- job timeouts to bound runner and token exposure;
- immutable pinning of the TruffleHog action;
- released-version pinning of `govulncheck` and `gosec` rather than build-time `@latest` resolution;
- public vulnerability-reporting and secure-development guidance;
- a pull-request security review checklist.

These changes improve evidence quality and software-supply-chain hygiene. They do not replace artifact signing, SBOM generation, branch protection, protected environments, independent review, or penetration testing.

## Known repository gaps

The following items should be resolved or explicitly accepted before using a derived host for health data:

1. **Scanner exclusions are broad.** The gosec workflow currently excludes several rules globally, including command execution and filesystem rules. Some exclusions reflect deliberate runtime capabilities, but each should be narrowed to paths or inline justifications and reviewed for new call sites.
2. **Not every GitHub Action is immutable.** Existing actions generally use major-version tags; the mutable TruffleHog `main` reference is fixed by this baseline, but a complete supply-chain program would pin every third-party action to a reviewed commit and automate controlled updates.
3. **No repository-enforced SBOM/provenance gate.** Release tooling has signing-related configuration, but the repository does not yet expose a single verified SBOM/provenance policy for all artifacts.
4. **Security ownership is not named in source control.** This baseline defines roles but does not appoint individuals or an organizational security function.
5. **Audit retention is deployment-specific.** The code normalizes and stores records but intentionally does not impose a legal/operational retention duration.
6. **The JavaScript runtime is not a hostile-code sandbox.** Safe module selection reduces exposed capabilities but does not provide process, kernel, network, CPU, or memory isolation.
7. **No Ségur penetration-test evidence exists for this repository.** A test must target the complete, configured candidate product and its actual architecture.
8. **The repository contains demos and development modes.** In-memory stores, insecure HTTP options, permissive module configurations, and demonstration identity setups must not be carried into production merely because they are present in examples.

## Health-domain requirements outside this repository

For an LGC candidate, the following Vague 1/Vague 2 areas are product obligations and are not implemented merely by adopting this repository:

- patient-record, appointment, prescription, clinical decision-support, dashboard, document, and professional-exchange functions;
- Identité Nationale de Santé (INS) identity lifecycle and qualification;
- DMP / Mon espace santé consultation and alimentation;
- MSSanté integration;
- Pro Santé Connect and applicable Espace de Confiance flows;
- healthcare document formats, including applicable CDA profiles and terminology/value sets;
- e-prescription and CNDA homologation requirements;
- consent, medical workflow, user-interface, accessibility, and ergonomic scenarios;
- production of Ségur indicators;
- healthcare hosting, privacy, data-protection, retention, interoperability, and contractual obligations;
- every retained Vague 1 requirement and every applicable Vague 2 profile.

A product architecture should map these obligations to dedicated components. Do not represent generic OIDC, generic audit logging, or generic document handling as equivalent to the named French health services.

## Candidate release evidence checklist

Before asserting compliance for a derived product, create an evidence bundle tied to the exact candidate version:

### Governance and scope

- [ ] accountable publisher and security roles named;
- [ ] current REM version frozen and archived;
- [ ] complete Vague 1/Vague 2 applicability matrix approved;
- [ ] architecture, data flows, trust boundaries, suppliers, and hosting documented;
- [ ] inventory of web, mobile, desktop, API, proxy, and background components complete.

### Engineering evidence

- [ ] immutable source commit and artifact digests;
- [ ] SBOM for every shipped component;
- [ ] build provenance and signature-verification records;
- [ ] unit, integration, end-to-end, negative authorization, migration, and recovery results;
- [ ] CodeQL, `govulncheck`, gosec, secret-scan, dependency-review, and container/infrastructure scan results;
- [ ] scanner suppressions and accepted risks reviewed;
- [ ] threat model and secure-design review current;
- [ ] no production secrets or health data in source, logs, fixtures, or artifacts.

### Identity, access, and audit

- [ ] user and privileged-account lifecycle tested;
- [ ] authentication, logout, timeout, revocation, MFA, and failure behavior tested;
- [ ] authorization and cross-tenant isolation tested;
- [ ] audit event catalogue, redaction, integrity, access, export, monitoring, and retention approved;
- [ ] trusted-proxy and client-address behavior verified in the deployed topology.

### Operations and resilience

- [ ] hardened production configuration reviewed;
- [ ] secrets and keys stored, rotated, backed up, and recoverable;
- [ ] database migrations and rollback constraints recorded;
- [ ] backup coverage, encryption, checksum, and isolated restore rehearsal evidenced;
- [ ] liveness/readiness, monitoring, alerting, capacity, cleanup, and incident runbooks tested;
- [ ] vulnerability monitoring and remediation targets operating.

### External and regulatory evidence

- [ ] ANS verification scenarios completed for every applicable REM requirement;
- [ ] required service approvals and homologations obtained;
- [ ] ANS penetration-test form completed, signed, current, and tied to the candidate major version;
- [ ] all high-severity disqualifying findings corrected and required counter-audits completed;
- [ ] privacy, healthcare hosting, data-processing, supplier, and contractual reviews complete;
- [ ] final dossier independently reviewed before submission.

## Maintenance rule

Review this document when any of the following changes:

- ANS publishes a new REM, DSR, penetration-test guide, or regulatory text;
- the repository adds a new privileged native module or authentication mechanism;
- production deployment topology or identity provider changes;
- a vulnerability, penetration-test finding, or incident changes the threat model;
- a derived product begins a formal Ségur referencing process.

Do not copy this crosswalk into a conformity dossier without validating every statement against the exact product version and current official source documents.
