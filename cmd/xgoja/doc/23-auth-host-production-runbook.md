---
Title: "Auth host production runbook"
Slug: auth-host-production-runbook
Short: "Deploy and validate a Keycloak-backed go-go-goja auth host with GHCR, GitOps, Vault, Postgres, and Argo CD."
Topics:
- xgoja
- gojahttp
- auth
- keycloak
- kubernetes
- gitops
- vault
- postgres
Commands:
- xgoja
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Application
---

This runbook records the production path proven by `goja-auth-host-demo` on `yolo.scapegoat.dev`. It is written for generated-host and host-author users who need a concise deployment checklist from source code to a validated HTTPS login flow.

The current live demo image was originally built from example 19, a hand-composed Keycloak host. Generated `auth.mode=oidc` is now available for `xgoja serve` hosts; use `examples/xgoja/21-generated-host-auth` as the generated-binary template for new work and keep this runbook's platform boundaries for either host shape.

## Live reference deployment

```text
URL:                 https://goja-auth.yolo.scapegoat.dev
Image:               ghcr.io/go-go-golems/go-goja-auth-host:sha-ba77afc
Argo application:    goja-auth-host-demo
Namespace:           goja-auth-host-demo
Keycloak issuer:     https://auth.yolo.scapegoat.dev/realms/goja-auth-host-demo
Keycloak client:     goja-auth-host-demo
Runtime secret:      kv/apps/goja-auth-host-demo/prod/runtime
Image-pull secret:   kv/apps/goja-auth-host-demo/prod/image-pull
```

The public smoke command is:

```bash
python3 examples/xgoja/19-express-keycloak-auth-host/scripts/keycloak_smoke.py \
  --base-url https://goja-auth.yolo.scapegoat.dev \
  --username demo-user \
  --password "$(VAULT_TOKEN=$(cat ~/.vault-token) vault kv get -field=demo_password kv/apps/goja-auth-host-demo/prod/runtime)"
```

Do not paste passwords or client secrets into docs, tickets, or chat. Retrieve them from Vault when needed.

## Deployment boundaries

A production auth host spans six systems:

```text
go-go-goja source
  -> GHCR image
  -> K3s GitOps manifest
  -> Vault runtime and image-pull secrets
  -> Keycloak realm/client
  -> Argo CD sync and Kubernetes runtime
```

Each system has a source of truth. Do not fix a Keycloak redirect mismatch in Kubernetes. Do not fix an image tag mismatch in Vault. Keep each correction in the system that owns the data.

## Source repository checklist

The source repository must provide:

- a generated binary or host binary that can run as a long-lived HTTP server;
- an `xgoja.yaml` or Dockerfile build path that embeds the route script or generated runtime package;
- an image publishing workflow for GHCR;
- a GitOps target entry if automated image PRs are used;
- tests for public URL and redirect URL resolution;
- signal-aware shutdown.

For new generated OIDC hosts, start from:

```text
examples/xgoja/21-generated-host-auth/xgoja.yaml
examples/xgoja/21-generated-host-auth/Makefile
examples/xgoja/21-generated-host-auth/scripts/fake_oidc_provider.py
examples/xgoja/21-generated-host-auth/scripts/compose_smoke.sh
examples/xgoja/21-generated-host-auth/scripts/keycloak_compose_smoke.py
```

For the original live demo image, the hand-composed reference files are:

```text
Dockerfile.auth-host
.github/workflows/publish-auth-host-image.yaml
deploy/gitops-targets.json
examples/xgoja/19-express-keycloak-auth-host/cmd/host/main.go
examples/xgoja/19-express-keycloak-auth-host/cmd/host/main_test.go
```

Build and push the image:

```bash
docker build -f Dockerfile.auth-host \
  -t ghcr.io/go-go-golems/go-goja-auth-host:sha-<short-sha> .
docker push ghcr.io/go-go-golems/go-goja-auth-host:sha-<short-sha>
```

## GitOps checklist

The Kustomize package should contain:

```text
namespace.yaml
serviceaccount.yaml
vault-connection.yaml
vault-auth.yaml
runtime-secret.yaml
image-pull-secret.yaml
postgres-admin-secret.yaml
db-bootstrap-script-configmap.yaml
db-bootstrap-job.yaml
deployment.yaml
service.yaml
ingress.yaml
```

The Argo Application should point at the package path and the intended target revision:

```yaml
source:
  repoURL: https://github.com/wesen/2026-03-27--hetzner-k3s.git
  targetRevision: main
  path: gitops/kustomize/goja-auth-host-demo
```

For pre-merge validation, the live Application can temporarily target a feature branch. Switch it back to `main` after merge.

## Vault checklist

Seed the runtime secret with the database and Keycloak fields:

```text
database
username
password
dsn
public_base_url
keycloak_issuer
keycloak_client_id
keycloak_client_secret
```

Seed the image-pull secret with Docker auth fields:

```text
server
username
password
auth
```

Write Kubernetes auth roles for the app service account and the DB bootstrap service account. The app role reads runtime and image-pull secrets. The bootstrap role reads the shared Postgres admin secret and the app runtime secret.

## Keycloak checklist

The Keycloak client must agree with the host's public URL.

```text
Realm:        goja-auth-host-demo
Client:       goja-auth-host-demo
Access type:  confidential
Redirect URI: https://goja-auth.yolo.scapegoat.dev/auth/callback
Web origin:   https://goja-auth.yolo.scapegoat.dev
```

Manual `kcadm.sh` provisioning is acceptable for a short-lived demo. Terraform is the preferred state for long-lived environments.

## Kubernetes command contract

Check the Dockerfile ENTRYPOINT before writing Deployment args.

For generated `xgoja serve` binaries, the runtime command is normally:

```bash
/app/generated-oidc-host-auth serve sites demo \
  --http-listen :8080 \
  --auth-mode oidc \
  --auth-default-store-driver postgres \
  --auth-default-store-dsn "$DATABASE_URL" \
  --auth-default-store-apply-schema \
  --auth-oidc-issuer-url "$KEYCLOAK_ISSUER" \
  --auth-oidc-client-id "$KEYCLOAK_CLIENT_ID" \
  --auth-oidc-client-secret "$KEYCLOAK_CLIENT_SECRET" \
  --auth-oidc-public-base-url "$PUBLIC_BASE_URL"
```

If the Dockerfile ENTRYPOINT already includes `serve`, Kubernetes args must pass only the remaining command path/flags. The original live image contains:

```dockerfile
ENTRYPOINT ["/app/goja-auth-host", "serve"]
CMD ["--listen", ":8080", "--script", "/app/server.js"]
```

The Kubernetes Deployment for that image therefore passes flags only:

```yaml
args:
  - --listen
  - :8080
  - --script
  - /app/server.js
```

Passing `serve` again crashes the process with `Too many arguments`.

## Runtime validation

Check Argo and Kubernetes first:

```bash
kubectl -n argocd get application goja-auth-host-demo
kubectl -n goja-auth-host-demo get pods,svc,ingress,certificate,vaultauth,vaultstaticsecret
```

Check public health:

```bash
curl -fsS https://goja-auth.yolo.scapegoat.dev/healthz
```

Check login redirect with GET, not HEAD:

```bash
curl -sS -D - -o /dev/null https://goja-auth.yolo.scapegoat.dev/auth/login
```

Run the full smoke script after TLS and Keycloak are ready. Before publishing a generated image, also run the local generated-host checks:

```bash
make -C examples/xgoja/21-generated-host-auth smoke
make -C examples/xgoja/21-generated-host-auth compose-smoke
```

The compose smoke validates real Keycloak login, Postgres-backed auth stores, JavaScript-owned audit and invite routes, and `409 Conflict` behavior for reused single-use capability tokens.

## Troubleshooting

| Problem | Cause | Solution |
| --- | --- | --- |
| `VAULT_TOKEN required` | Operator shell has no Vault token. | Export `VAULT_TOKEN=$(cat ~/.vault-token)` or run `vault login -method=oidc role=operators`. |
| `GITHUB_DEPLOY_PAT` missing | Image-pull bootstrap needs registry credentials. | Export a token with GHCR package read access before seeding the pull secret. |
| Pod crashes with `Too many arguments` | ENTRYPOINT already includes `serve`; Deployment args include it again. | Remove `serve` from Kubernetes args. |
| Generated pod fails during OIDC discovery | Issuer URL is wrong or Keycloak is unreachable from the pod. | Check `--auth-oidc-issuer-url`, DNS, network policy, and Keycloak readiness. |
| Generated pod rejects callback URL as insecure | Public base URL is HTTP outside localhost. | Use HTTPS ingress and `--auth-oidc-public-base-url https://...`; only use insecure HTTP for localhost smoke tests. |
| Argo shows old revision after a fix | A previous sync operation is still running or cached. | Hard-refresh the app and, if needed, clear the operation before resync. |
| `/auth/login` returns 405 in curl | `curl -I` sent HEAD. | Use GET for login redirect checks. |
| Login redirects to Keycloak but callback fails | Redirect URI mismatch or wrong client secret. | Compare `PUBLIC_BASE_URL`, Keycloak client redirect URI, and Vault client secret. |

## Migration criteria

The original example-19 image can be replaced when the generated binary uses durable stores, the same Keycloak realm/client, the same Vault-secret contract, the same HTTPS `public-base-url`, and the public smoke test passes. Keep replicas at 1 until the OIDC transaction store is durable across pods.

## See also

- `xgoja help generated-auth-javascript-apis`
- `xgoja help express-auth-host-integration-guide`
- `xgoja help hostauth-config-reference`
- `xgoja help auth-stores-reference`
- `xgoja help http-serve-command-reference`
- `goja-repl help auth-module-guide`
- `goja-repl help deploying-an-express-auth-host`
