# Changelog

## 2026-06-16

- Initial workspace created


## 2026-06-16

Created intern-oriented issue #82 design package for replacing the production hand-written Keycloak host with a self-contained xgoja.yaml generated serve host.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST--implement-generated-xgoja-oidc-auth-host-support/design-doc/01-generated-xgoja-oidc-auth-host-design-and-implementation-guide.md — Generated OIDC host design and implementation guide
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST--implement-generated-xgoja-oidc-auth-host-support/reference/01-diary.md — Investigation diary Step 1


## 2026-06-16

Validated the issue #82 ticket and uploaded the design bundle to reMarkable at /ai/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST--implement-generated-xgoja-oidc-auth-host-support/design-doc/01-generated-xgoja-oidc-auth-host-design-and-implementation-guide.md — Primary uploaded design guide
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST--implement-generated-xgoja-oidc-auth-host-support/reference/01-diary.md — Diary records validation and upload evidence


## 2026-06-16

Revised the issue #82 design for a hard cutover: serve owns HTTP server lifecycle, Express only registers routes, and old implicit Express startup users should be migrated rather than wrapped.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST--implement-generated-xgoja-oidc-auth-host-support/design-doc/01-generated-xgoja-oidc-auth-host-design-and-implementation-guide.md — Updated hard-cutover server ownership design
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST--implement-generated-xgoja-oidc-auth-host-support/reference/01-diary.md — Diary Step 3 records the architectural clarification


## 2026-06-16

Uploaded revised v2 design bundle to reMarkable after hard-cutover server ownership clarification.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST--implement-generated-xgoja-oidc-auth-host-support/design-doc/01-generated-xgoja-oidc-auth-host-design-and-implementation-guide.md — Revised v2 design content
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST--implement-generated-xgoja-oidc-auth-host-support/reference/01-diary.md — Upload evidence for v2 bundle


## 2026-06-16

Added detailed issue #82 implementation tasks and recorded the current HTTP/Express listener ownership inventory.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/providers/http/http.go — Current Express side-effect listener path
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/providers/http/serve.go — Current serve command runtime path
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST--implement-generated-xgoja-oidc-auth-host-support/reference/01-diary.md — Step 4 records inventory and next implementation seam
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST--implement-generated-xgoja-oidc-auth-host-support/tasks.md — Detailed implementation task checklist


## 2026-06-16

Moved normal xgoja http serve onto a command-owned listener/server/mux lifecycle and added a regression proving serve starts without Express listen side effects.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/providers/http/serve.go — Normal serve now owns listener/server/top-level mux and graceful shutdown
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/providers/http/serve_test.go — Regression for command-owned serve listener independent of Express listen
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST--implement-generated-xgoja-oidc-auth-host-support/reference/01-diary.md — Step 5 records failing test
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST--implement-generated-xgoja-oidc-auth-host-support/tasks.md — Tasks 6 and 7 completed


## 2026-06-16

Made the Express module pure route registration by removing WithOnUse/capability.start listener side effects; app.listen now reports to use xgoja serve.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/modules/express/auth_builders.go — Planned route handle now only registers into the host
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/modules/express/express.go — Removed Express listener startup hook and made app.listen a migration error
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/providers/http/http.go — HTTP module loader no longer starts listeners
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/providers/http/http_test.go — Updated no-bind regression expectations
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST--implement-generated-xgoja-oidc-auth-host-support/reference/01-diary.md — Step 6 records Express hard-cutover details
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST--implement-generated-xgoja-oidc-auth-host-support/tasks.md — Task 8 completed


## 2026-06-16

Aligned hot reload with the serve-owned top-level handler and shutdown helpers so future native auth routes share one mounting seam.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/providers/http/serve.go — Hot reload now uses buildServeHandler and serveHTTPServer
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST--implement-generated-xgoja-oidc-auth-host-support/reference/01-diary.md — Step 7 records hot reload alignment
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST--implement-generated-xgoja-oidc-auth-host-support/tasks.md — Task 9 completed


## 2026-06-16

Added top-level auth to xgoja/v2 specs and runtime plans, installing a lazy hostauth service factory during generated host construction.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/cmd/xgoja/internal/generate/generate_test.go — Runtime plan auth propagation test
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/cmd/xgoja/internal/generate/templates.go — Runtime plan JSON copies auth config
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/cmd/xgoja/internal/specv2/types.go — Top-level auth schema field
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/app/auth_plan_test.go — Host service factory installation test
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/app/host.go — Generated host installs hostauth service factory
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/app/runtime_plan.go — Runtime plan carries auth config
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST--implement-generated-xgoja-oidc-auth-host-support/reference/01-diary.md — Step 8 records top-level auth planning
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST--implement-generated-xgoja-oidc-auth-host-support/tasks.md — Task 10 completed


## 2026-06-16

Resolved the test import cycle caused by app importing hostauth by moving hostauth lookup tests to external package style.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/hostauth/lookup_test.go — External-package test avoids app<->hostauth test import cycle
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST--implement-generated-xgoja-oidc-auth-host-support/reference/01-diary.md — Step 8 records the import-cycle failure and fix


## 2026-06-16

Built OIDC-capable hostauth services from generated auth config, including Glazed auth-oidc fields, redirect resolution, default OIDC user normalization, and native auth handlers.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/hostauth/builder.go — OIDC native handler construction and default normalizer
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/hostauth/builder_test.go — OIDC service construction and normalizer tests
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/hostauth/config.go — OIDC config and resolved config shape
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/hostauth/glazed.go — auth-oidc Glazed fields for flags/env/config
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/hostauth/glazed_test.go — OIDC Glazed mapping tests
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/hostauth/resolve.go — OIDC URL validation and callback derivation
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/hostauth/resolve_test.go — OIDC config resolution tests
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/hostauth/services.go — NativeHandler service payload
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST--implement-generated-xgoja-oidc-auth-host-support/reference/01-diary.md — Step 9 records hostauth OIDC service construction
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST--implement-generated-xgoja-oidc-auth-host-support/tasks.md — Task 11 completed


## 2026-06-16

Mounted hostauth native handlers in the serve-owned mux before the JavaScript app/hot-reload fallback.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/providers/http/serve.go — buildServeHandler mounts NativeHandlers before fallback
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/providers/http/serve_test.go — Native auth mount ordering regression
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST--implement-generated-xgoja-oidc-auth-host-support/reference/01-diary.md — Step 10 records native handler mounting
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST--implement-generated-xgoja-oidc-auth-host-support/tasks.md — Task 12 completed


## 2026-06-16

Converted example 21 into a self-contained generated OIDC binary example with native /auth/login smoke coverage.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/examples/xgoja/21-generated-host-auth/Makefile — OIDC smoke builds/runs generated binary
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/examples/xgoja/21-generated-host-auth/scripts/fake_oidc_provider.py — Minimal fake OIDC discovery provider for smoke
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/examples/xgoja/21-generated-host-auth/xgoja.yaml — Top-level auth.mode=oidc generated binary fixture
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST--implement-generated-xgoja-oidc-auth-host-support/reference/01-diary.md — Step 11 records generated OIDC example conversion
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST--implement-generated-xgoja-oidc-auth-host-support/tasks.md — Task 13 completed


## 2026-06-16

Recorded OIDC serve behavior coverage across hostauth, HTTP serve, generation, and generated-example smoke tests.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/examples/xgoja/21-generated-host-auth/Makefile — Generated binary OIDC smoke
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/hostauth/builder_test.go — OIDC handler and normalizer coverage
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/hostauth/resolve_test.go — OIDC config resolution coverage
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/providers/http/serve_test.go — Native auth route mount ordering coverage
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST--implement-generated-xgoja-oidc-auth-host-support/reference/01-diary.md — Step 12 records the test coverage matrix
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST--implement-generated-xgoja-oidc-auth-host-support/tasks.md — Task 14 completed


## 2026-06-16

Updated permanent xgoja/goja-repl docs for implemented generated OIDC serve hosts.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/cmd/xgoja/doc/17-xgoja-v2-reference.md — Top-level auth and generated OIDC YAML reference
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/cmd/xgoja/doc/20-hostauth-config-reference.md — OIDC config
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/cmd/xgoja/doc/22-http-serve-command-reference.md — Serve lifecycle and native auth mounting reference
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/cmd/xgoja/doc/23-auth-host-production-runbook.md — Generated OIDC deployment runbook updates
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/doc/32-deploying-an-express-auth-host.md — Deployment tutorial no longer says generated OIDC is unimplemented
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST--implement-generated-xgoja-oidc-auth-host-support/reference/01-diary.md — Step 13 records permanent docs update
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST--implement-generated-xgoja-oidc-auth-host-support/tasks.md — Task 15 completed


## 2026-06-16

Completed final validation and uploaded the generated OIDC implementation bundle to reMarkable.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST--implement-generated-xgoja-oidc-auth-host-support/reference/01-diary.md — Step 14 records final validation and reMarkable upload
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST--implement-generated-xgoja-oidc-auth-host-support/tasks.md — Task 16 completed


## 2026-06-16

Added production migration guide and deployment tasks for replacing the live example-19 image with the generated example-21 OIDC host.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST--implement-generated-xgoja-oidc-auth-host-support/design-doc/02-production-migration-to-generated-oidc-image.md — Production migration implementation guide
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST--implement-generated-xgoja-oidc-auth-host-support/reference/01-diary.md — Step 15 records migration planning
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST--implement-generated-xgoja-oidc-auth-host-support/tasks.md — Deployment tasks 17-22 added and task 17 completed


## 2026-06-16

Switched the auth-host source image and workflow to build the generated example-21 OIDC binary, with native /auth/session and closer route parity.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/.github/workflows/publish-auth-host-image.yaml — Publishes generated OIDC image
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/Dockerfile.auth-host — Builds generated OIDC host image
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/examples/xgoja/21-generated-host-auth/verbs/sites.js — Generated demo route parity
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/examples/xgoja/21-generated-host-auth/xgoja.yaml — Generated image spec with timer
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/hostauth/builder.go — Native /auth/session handler
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST--implement-generated-xgoja-oidc-auth-host-support/reference/01-diary.md — Step 16 records source image changes
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST--implement-generated-xgoja-oidc-auth-host-support/tasks.md — Task 18 completed


## 2026-06-16

Validated the generated auth-host Docker image with env-only XGOJA_OIDC_DEMO configuration.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/Dockerfile.auth-host — Generated image validated locally
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST--implement-generated-xgoja-oidc-auth-host-support/reference/01-diary.md — Step 17 records Docker env-only validation
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST--implement-generated-xgoja-oidc-auth-host-support/tasks.md — Task 19 completed


## 2026-06-16

Built and pushed generated OIDC auth-host image ghcr.io/go-go-golems/go-goja-auth-host:sha-f1fa40f.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST--implement-generated-xgoja-oidc-auth-host-support/reference/01-diary.md — Step 18 records GHCR image publication
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST--implement-generated-xgoja-oidc-auth-host-support/tasks.md — Task 20 completed


## 2026-06-16

Updated and pushed K3s GitOps Deployment to run the generated OIDC image with XGOJA_OIDC_DEMO env vars (K3s commit 90dc20c).

### Related Files

- /home/manuel/code/wesen/2026-03-27--hetzner-k3s/gitops/kustomize/goja-auth-host-demo/deployment.yaml — Production Deployment now runs generated OIDC image
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST--implement-generated-xgoja-oidc-auth-host-support/reference/01-diary.md — Step 19 records K3s GitOps migration
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST--implement-generated-xgoja-oidc-auth-host-support/tasks.md — Task 21 completed


## 2026-06-16

Added generated demo invite native handlers so the production replacement can satisfy the existing full smoke contract.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/hostauth/builder.go — Demo invite native handlers
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/hostauth/builder_test.go — Native handler list includes invite endpoints
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST--implement-generated-xgoja-oidc-auth-host-support/reference/01-diary.md — Step 20 records smoke-parity invite handlers


## 2026-06-16

Deployed generated OIDC image sha-81d40b9 to production and passed the full public Keycloak smoke.

### Related Files

- /home/manuel/code/wesen/2026-03-27--hetzner-k3s/gitops/kustomize/goja-auth-host-demo/deployment.yaml — Production Deployment now points at smoke-compatible generated image sha-81d40b9
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST--implement-generated-xgoja-oidc-auth-host-support/design-doc/02-production-migration-to-generated-oidc-image.md — Updated guide to reflect full-smoke-compatible generated replacement
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST--implement-generated-xgoja-oidc-auth-host-support/reference/01-diary.md — Step 21 records final production validation
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST--implement-generated-xgoja-oidc-auth-host-support/tasks.md — Task 22 completed


## 2026-06-16

Added embedded HTML landing dashboard and protected /auth/audit endpoint for the generated OIDC demo.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/examples/xgoja/21-generated-host-auth/verbs/sites.js — Embedded landing dashboard
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/hostauth/builder.go — Protected recent audit endpoint
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/16/XGOJA-ISSUE-82-OIDC-GENERATED-HOST--implement-generated-xgoja-oidc-auth-host-support/reference/01-diary.md — Step 22 records landing/audit work

