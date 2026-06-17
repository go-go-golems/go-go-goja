---
Title: PR 74 Targeted Validation Output
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
Summary: "Captured output from targeted package validation tests."
LastUpdated: 2026-06-14T20:55:00-04:00
WhatFor: "Evidence captured while planning the PR 74 code review."
WhenToUse: "Use as supporting evidence for the PR 74 review methodology guide."
---

# PR 74 Targeted Validation

repo: /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja
go: go version go1.26.1 linux/amd64
GOFLAGS: -buildvcs=false


## go test ./pkg/gojahttp ./modules/express ./pkg/xgoja/providers/http ./pkg/xgoja/hostauth -count=1
ok  	github.com/go-go-golems/go-go-goja/pkg/gojahttp	0.022s
ok  	github.com/go-go-golems/go-go-goja/modules/express	0.043s
ok  	github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/http	0.499s
ok  	github.com/go-go-golems/go-go-goja/pkg/xgoja/hostauth	0.030s

## go test ./pkg/gojahttp/auth/... -count=1
ok  	github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/appauth	0.021s
ok  	github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/appauth/sqlstore	0.010s
ok  	github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/audit	0.006s
ok  	github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/audit/sqlstore	0.011s
ok  	github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/capability	0.006s
ok  	github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/capability/sqlstore	0.009s
ok  	github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/devauth	0.004s
?   	github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/internal/appauthtest	[no test files]
?   	github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/internal/audittest	[no test files]
?   	github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/internal/capabilitytest	[no test files]
?   	github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/internal/sessionauthtest	[no test files]
ok  	github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/keycloakauth	0.297s
ok  	github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/sessionauth	0.005s
ok  	github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/sessionauth/sqlstore	0.007s

## go test ./examples/xgoja/18-express-auth-host/cmd/host ./examples/xgoja/20-express-hello-world/cmd/host ./examples/xgoja/21-generated-host-auth/cmd/host -count=1
?   	github.com/go-go-golems/go-go-goja/examples/xgoja/18-express-auth-host/cmd/host	[no test files]
?   	github.com/go-go-golems/go-go-goja/examples/xgoja/20-express-hello-world/cmd/host	[no test files]
?   	github.com/go-go-golems/go-go-goja/examples/xgoja/21-generated-host-auth/cmd/host	[no test files]

## examples/xgoja/21-generated-host-auth make targets
.PHONY
smoke
doctor
generate
memory-smoke
sqlite-smoke
clean
