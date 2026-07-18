# Changelog

## 2026-07-18

- Initial workspace created


## 2026-07-18

Created ticket, reproduced order-dependent build/generate failures and support-first runtime metadata mismatch, mapped planner/generator architecture, and wrote the pragmatic intern implementation guide

### Related Files

- /home/manuel/workspaces/2026-07-18/improve-xgoja/go-go-goja/cmd/xgoja/internal/generate/templates.go — Independent generator target derivation requiring a scoped plan
- /home/manuel/workspaces/2026-07-18/improve-xgoja/go-go-goja/cmd/xgoja/v2_plan_helpers.go — First-artifact selection root cause
- /home/manuel/workspaces/2026-07-18/improve-xgoja/go-go-goja/ttmp/2026/07/18/XGOJA-ARTIFACT-SELECTION-2026-07-18--pragmatic-command-compatible-artifact-selection-for-xgoja-build-and-generate/scripts/01-reproduce-artifact-order.log — Captured reproducible evidence


## 2026-07-18

Validated the ticket with docmgr doctor and uploaded the overview, intern guide, and diary as the XGOJA Artifact Selection Intern Guide reMarkable bundle

### Related Files

- /home/manuel/workspaces/2026-07-18/improve-xgoja/go-go-goja/ttmp/2026/07/18/XGOJA-ARTIFACT-SELECTION-2026-07-18--pragmatic-command-compatible-artifact-selection-for-xgoja-build-and-generate/design-doc/01-intern-guide-to-xgoja-artifact-selection.md — Primary document delivered to reMarkable
- /home/manuel/workspaces/2026-07-18/improve-xgoja/go-go-goja/ttmp/2026/07/18/XGOJA-ARTIFACT-SELECTION-2026-07-18--pragmatic-command-compatible-artifact-selection-for-xgoja-build-and-generate/reference/01-investigation-diary.md — Validation and upload evidence


## 2026-07-18

Implemented command-compatible primary selection and non-mutating scoped plans; build/generate now work with binary/runtime-package in either order, reject ambiguity clearly, scope JS/help sources, retain assets, and pass full validation

### Related Files

- /home/manuel/workspaces/2026-07-18/improve-xgoja/go-go-goja/cmd/xgoja/doc/17-xgoja-v2-reference.md — Documented public command/artifact behavior (commit 4003433)
- /home/manuel/workspaces/2026-07-18/improve-xgoja/go-go-goja/cmd/xgoja/root_test.go — Order and embedded-source regression coverage (commit 4003433)
- /home/manuel/workspaces/2026-07-18/improve-xgoja/go-go-goja/cmd/xgoja/v2_plan_helpers.go — Core selection and scoped-plan implementation (commit 7caaee6)


## 2026-07-18

Ticket closed


## 2026-07-18

Addressed review regression: normalize whitespace-padded artifact types during compatibility/support classification and in scoped generator plans; added non-mutation regression coverage (commit 30d0d88)

### Related Files

- /home/manuel/workspaces/2026-07-18/improve-xgoja/go-go-goja/cmd/xgoja/v2_plan_helpers.go — Whitespace normalization at selection and scoped generation boundaries
- /home/manuel/workspaces/2026-07-18/improve-xgoja/go-go-goja/cmd/xgoja/v2_plan_helpers_test.go — Regression coverage for padded primary/support artifact types

