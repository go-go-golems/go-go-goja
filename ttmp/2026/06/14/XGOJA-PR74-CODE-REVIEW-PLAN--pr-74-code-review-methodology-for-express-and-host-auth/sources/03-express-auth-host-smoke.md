---
Title: Express Auth Host Smoke Output
Ticket: XGOJA-PR74-CODE-REVIEW-PLAN
Status: active
Topics:
    - review
    - goja
    - xgoja
    - auth
    - security
    - testing
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Captured output from the hand-written Express auth host smoke test."
LastUpdated: 2026-06-14T20:55:00-04:00
WhatFor: "Evidence captured while planning the PR 74 code review."
WhenToUse: "Use as supporting evidence for the PR 74 review methodology guide."
---

make: Entering directory '/home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/examples/xgoja/18-express-auth-host'
cd /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja && GOWORK=off go run ./examples/xgoja/18-express-auth-host/cmd/host --script /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/examples/xgoja/18-express-auth-host/scripts/server.js --smoke
ok public health            200
ok me before login          401
ok bad login                401
ok login                    200
ok async return             200
ok async send               200
ok session after login      200
ok me after login           200
ok project missing csrf     403
ok project update           200
ok project missing          404
ok logout                   204
ok me after logout          401
{"auditEvents":14,"status":"PASS"}
make: Leaving directory '/home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/examples/xgoja/18-express-auth-host'
