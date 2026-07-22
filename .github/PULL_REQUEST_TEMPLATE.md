## Summary

Describe the change, the problem it solves, and the affected packages, commands, generated artifacts, or deployment paths.

## Verification

List the commands, automated tests, manual tests, and environments used to verify the change.

## Security review

Complete the applicable items. Mark an item `N/A` only when the reason is clear from the pull request.

- [ ] The trust boundary and attacker-controlled inputs are identified.
- [ ] New or changed native modules document their host capabilities and safe-mode status.
- [ ] Authentication, authorization, tenant isolation, CSRF, rate limiting, or audit behavior is unchanged, or the change includes negative-path tests.
- [ ] The change does not expose secrets, credentials, session identifiers, authorization headers, personal data, or health data in logs, errors, examples, fixtures, or artifacts.
- [ ] File, command, database, network, parser, and template inputs are bounded and validated.
- [ ] Timeouts, cancellation, cleanup, and dependency-failure behavior are covered where applicable.
- [ ] Dependency, tool, image, and GitHub Action references are intentionally versioned or pinned.
- [ ] Scanner suppressions are narrow, documented, and supported by tests or design rationale.
- [ ] Migration, rollback, backup/restore, and operational effects are documented where applicable.
- [ ] Security, deployment, user, and generated-code documentation is updated.

## Residual risk

State any accepted risk, follow-up issue, deployment assumption, or evidence that is intentionally deferred.
