# Changelog

## 2026-06-17

- Initial workspace created


## 2026-06-17

Created Issue 85 ticket, wrote intern-oriented JavaScript auth DB/audit access design guide, and recorded investigation diary.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/17/XGOJA-ISSUE-85-JS-AUTH-DB-AUDIT--issue-85-javascript-auth-db-and-audit-access-design/design-doc/01-javascript-auth-db-and-audit-access-design-and-implementation-guide.md — Primary design deliverable
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/17/XGOJA-ISSUE-85-JS-AUTH-DB-AUDIT--issue-85-javascript-auth-db-and-audit-access-design/reference/01-investigation-diary.md — Chronological investigation diary


## 2026-06-17

Validated Issue 85 ticket, added audit/database vocabulary topics, and uploaded the design bundle to reMarkable.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/17/XGOJA-ISSUE-85-JS-AUTH-DB-AUDIT--issue-85-javascript-auth-db-and-audit-access-design/reference/01-investigation-diary.md — Step 2 records validation and reMarkable upload
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/vocabulary.yaml — Added audit and database topic vocabulary entries


## 2026-06-17

Step 3: implemented bounded audit QueryStore for memory and SQL stores (commit a0a2eeb).

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth/audit/audit.go — Query contract and memory query implementation
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth/audit/sqlstore/sqlstore.go — SQL query implementation


## 2026-06-17

Step 4: implemented guarded JavaScript auth audit module exposing require("auth").audit.query (commit 53156f5).

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/providers/hostauth/hostauth.go — Auth provider module implementation
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/providers/hostauth/hostauth_test.go — Provider and runtime tests


## 2026-06-17

Step 5: wired example 21 to JS-owned /orgs/:orgId/audit route using auth.audit.query and audit.read authorization (commit b7f85cc).

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/examples/xgoja/21-generated-host-auth/verbs/sites.js — JS-owned audit route
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth/appauth/appauth.go — audit.read authorization action


## 2026-06-17

Step 6: added reusable auth core cleanup and self-contained demo design doc.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/17/XGOJA-ISSUE-85-JS-AUTH-DB-AUDIT--issue-85-javascript-auth-db-and-audit-access-design/design-doc/02-reusable-auth-core-interface-cleanup-and-demo-design.md — Design for auth core cleanup and richer demo


## 2026-06-17

Step 7: uploaded reusable auth core cleanup design to reMarkable.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/17/XGOJA-ISSUE-85-JS-AUTH-DB-AUDIT--issue-85-javascript-auth-db-and-audit-access-design/design-doc/02-reusable-auth-core-interface-cleanup-and-demo-design.md — Uploaded to reMarkable as XGOJA Issue 85 Reusable Auth Core Cleanup Design.pdf
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/17/XGOJA-ISSUE-85-JS-AUTH-DB-AUDIT--issue-85-javascript-auth-db-and-audit-access-design/reference/01-investigation-diary.md — Step 7 records upload result

