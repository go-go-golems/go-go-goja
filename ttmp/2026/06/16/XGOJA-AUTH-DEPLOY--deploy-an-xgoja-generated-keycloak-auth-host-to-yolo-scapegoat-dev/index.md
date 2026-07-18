---
Title: Deploy an xgoja-generated Keycloak auth host to yolo.scapegoat.dev
Ticket: XGOJA-AUTH-DEPLOY
Status: complete
Topics:
    - goja
    - keycloak
    - oidc
    - deployment
    - kubernetes
    - gitops
    - vault
    - security
    - backend
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: design/01-deploy-xgoja-keycloak-auth-host-to-yolo.md
      Note: Primary intern-facing analysis/design/implementation guide
    - Path: reference/01-investigation-diary.md
      Note: Chronological investigation and decision diary
    - Path: examples/xgoja/19-express-keycloak-auth-host/cmd/host/main.go
      Note: Production-oriented Keycloak/OIDC host to promote into cmd/goja-auth-host
    - Path: examples/xgoja/21-generated-host-auth/xgoja.yaml
      Note: Generated-runtime-package seam (xgoja generate) template
    - Path: pkg/gojahttp/auth/keycloakauth/keycloakauth.go
      Note: keycloakauth.Config + OIDC login/callback/logout handlers
    - Path: pkg/xgoja/hostauth/config.go
      Note: hostauth.Config host-owned auth infra knobs (Mode, Session, Stores)
    - Path: deploy/gitops-targets.json
      Note: Existing target metadata (essay); append goja-auth-host-prod target
    - Path: .github/workflows/publish-auth-host-image.yaml
      Note: Existing GHCR + open_gitops_pr.py pipeline to extend
ExternalSources:
    - /home/manuel/code/wesen/2026-03-27--hetzner-k3s/gitops/kustomize/go-go-host/
    - /home/manuel/code/wesen/2026-03-27--hetzner-k3s/gitops/applications/go-go-host.yaml
    - /home/manuel/code/wesen/2026-03-27--hetzner-k3s/docs/app-runtime-secrets-and-identity-provisioning-playbook.md
    - /home/manuel/code/wesen/terraform/keycloak/apps/go-go-host/envs/k3s-beta/
    - /home/manuel/code/wesen/terraform/docs/shared-keycloak-platform-playbook.md
    - /home/manuel/code/wesen/go-go-golems/infra-tooling/docs/platform/source-repo-to-gitops-pr.md
Summary: Productionize an xgoja-generated Keycloak auth host (from examples/xgoja/19 + 21) and deploy it to the yolo.scapegoat.dev K3s cluster via the shared GHCR + Argo GitOps + Vault + Keycloak pipeline. In-repo work only; cluster/terraform/vault changes are specified and approval-gated.
LastUpdated: 2026-06-18T17:34:37.869034328-04:00
WhatFor: Use when onboarding an engineer to deploying go-go-goja auth hosts to the cluster, or when implementing cmd/goja-auth-host and its release pipeline.
WhenToUse: Before promoting the example host to a production binary, before adding the Dockerfile target / GitOps target, and before requesting the out-of-repo cluster/terraform/vault changes.
---


# Deploy an xgoja-generated Keycloak auth host to yolo.scapegoat.dev

## Overview

This ticket productionizes the existing xgoja Keycloak auth example
(`examples/xgoja/19-express-keycloak-auth-host`, with the generated seam from
`examples/xgoja/21-generated-host-auth`) and ships it to the
`yolo.scapegoat.dev` K3s cluster using our real shared Keycloak, Vault,
PostgreSQL, and Argo CD GitOps pipeline.

The deliverable is an intern-grade analysis/design/implementation guide plus an
investigation diary. Scope is strictly limited to `./go-go-goja/`; every change
required in the cluster repo, terraform repo, or Vault is specified verbatim in
the design doc and is **approval-gated** (not implemented here).

## Key docs

- **Design / implementation guide** (start here):
  [design/01-deploy-xgoja-keycloak-auth-host-to-yolo.md](./design/01-deploy-xgoja-keycloak-auth-host-to-yolo.md)
- **Investigation diary**: [reference/01-investigation-diary.md](./reference/01-investigation-diary.md)

## Scope boundary (important)

- **In scope (this repo):** `cmd/goja-auth-host`, `Dockerfile.auth-host`,
  `.github/workflows/publish-auth-host-image.yaml`, `deploy/gitops-targets.json`, docs.
- **Out of scope / approval-gated:** cluster repo Kustomize + Argo Application
  + Vault policies/roles; terraform Keycloak realm/client; Vault secret
  seeding. See design doc §10–§12.

## Related/sibling tickets (avoid overlap)

- `XGOJA-AUTH-STORES` — durable store schemas/migrations.
- `XGOJA-AUTH-KEYCLOAK-MFA` — Keycloak hardening, OIDC transaction store, MFA, `hostauth.ModeOIDC`.
- `XGOJA-AUTH-PROD-DOCS` — production hardening prose + policy-adapter evaluation.

## Status

Current status: **active** (design complete, implementation pending operator approval)

## Topics

- goja, keycloak, oidc, deployment, kubernetes, gitops, vault, security, backend

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries (diary)
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
