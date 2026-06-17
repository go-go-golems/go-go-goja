# Changelog

## 2026-06-12

- Initial workspace created


## 2026-06-12

Create initial Keycloak production hardening and MFA flow planning ticket with phased design and tasks

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/12/XGOJA-AUTH-KEYCLOAK-MFA--production-keycloak-hardening-and-mfa-flows/design/01-keycloak-production-hardening-and-mfa-implementation-plan.md — Initial Keycloak/MFA design
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/12/XGOJA-AUTH-KEYCLOAK-MFA--production-keycloak-hardening-and-mfa-flows/tasks.md — Initial phased task list


## 2026-06-16

Updated design doc to v2: temporary deployment should build from examples/xgoja/19 instead of promoting to cmd/goja-auth-host, mandate Postgres-backed runtime stores, add required deploy flags (public base/redirect URL, secure cookie toggle, DSNs), specify DB/user bootstrap via Vault-backed Postgres Job, and document demo-apps Argo registration including AppProject namespace allowlist.

### Related Files

- /home/manuel/code/wesen/2026-03-27--hetzner-k3s/gitops/projects/demo-apps.yaml — Demo AppProject namespace allowlist must include goja-auth-host-demo
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/examples/xgoja/19-express-keycloak-auth-host/cmd/host/main.go — Example host to patch and deploy temporarily


## 2026-06-16

Updated v2 design to require Glazed for the temporary example host command instead of raw flag parsing: public-base-url is a first-class Glazed setting used to derive redirect-url, redirect-url remains an advanced override, and allow-insecure-http gates localhost-only HTTP. Added glazed-lint as a tracked go.mod tool dependency and changed Makefile to install/use the project-local .bin/glazed-lint from go.mod so make lint/pre-commit/pre-push enforce Glazed command rules.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/Makefile — glazed-lint now installs from go.mod tool dependency into .bin and is already part of make lint
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/examples/xgoja/19-express-keycloak-auth-host/cmd/host/main.go — planned conversion from raw flag parsing to Glazed command/settings
- /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/go.mod — tracks github.com/go-go-golems/glazed/cmd/tools/glazed-lint as a Go tool

