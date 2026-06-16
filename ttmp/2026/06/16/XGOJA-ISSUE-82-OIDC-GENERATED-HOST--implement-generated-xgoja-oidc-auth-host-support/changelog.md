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

