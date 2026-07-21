---
Title: Deploying an Express auth host
Slug: deploying-an-express-auth-host
Short: Deploy a Keycloak-backed go-go-goja Express auth host with GHCR, GitOps, Vault, Postgres, and Argo CD.
Topics:
- http
- express
- auth
- keycloak
- deployment
- kubernetes
- gitops
- vault
- postgres
Commands:
- goja-repl
- xgoja
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

This tutorial explains the deployment path proven by `goja-auth-host-demo` on `yolo.scapegoat.dev`. It starts with an Express planned-auth host in `go-go-goja`, builds a container image, deploys it through K3s GitOps, reads secrets from Vault, authenticates through Keycloak, persists auth state in PostgreSQL, and validates the public browser flow.

The current live deployment image was built from `examples/xgoja/19-express-keycloak-auth-host`, a hand-composed Keycloak host that proved the production platform. Generated `auth.mode=oidc` is now available for `xgoja serve`; use `examples/xgoja/21-generated-host-auth` as the generated-binary starting point for new hosts while reusing the same Keycloak, Vault, Postgres, GitOps, and public smoke boundaries described here.

## What is deployed

```text
Public URL:          https://goja-auth.yolo.scapegoat.dev
Image:               ghcr.io/go-go-golems/go-go-goja-auth-host:sha-ba77afc
Argo app:            goja-auth-host-demo
Namespace:           goja-auth-host-demo
Keycloak issuer:     https://auth.yolo.scapegoat.dev/realms/goja-auth-host-demo
Runtime secret:      kv/apps/goja-auth-host-demo/prod/runtime
Postgres database:   goja_auth_host_demo
```

The host serves planned Express routes. Public routes such as `/healthz` work without login. Protected routes such as `/me` require the app session created by the Keycloak callback. Unsafe routes require CSRF. App-owned resources and memberships are stored in PostgreSQL.

## The deployment model

A working deployment requires these systems to agree:

```text
source repo
  -> GHCR image
  -> GitOps deployment manifest
  -> Vault runtime and pull secrets
  -> Keycloak realm/client
  -> PostgreSQL database and role
  -> Argo CD Application
  -> HTTPS Ingress
```

The app will not work if any one of these contracts drifts. The image can build successfully with the wrong Kubernetes args. The Pod can start with the wrong Keycloak redirect URI. Keycloak can authenticate the user while the app session cookie is rejected because the public URL and cookie settings are wrong.

## Step 1: expose operator-facing host settings

A deployable auth host needs explicit configuration for the values that differ between local Docker Compose and the cluster.

Generated OIDC hosts expose these through the HTTP provider's Glazed-backed `serve` command:

```text
--http-listen
--auth-mode
--auth-default-store-driver
--auth-default-store-dsn
--auth-default-store-apply-schema
--auth-oidc-issuer-url
--auth-oidc-client-id
--auth-oidc-client-secret
--auth-oidc-public-base-url
--auth-oidc-redirect-url
--auth-oidc-after-login-url
--auth-oidc-after-logout-url
--auth-session-cookie-allow-insecure-http
```

The original example 19 host exposes a hand-composed equivalent through its own Glazed `serve` command:

```text
--listen                  LISTEN_ADDR
--script                  SCRIPT_PATH
--issuer                  KEYCLOAK_ISSUER
--client-id               KEYCLOAK_CLIENT_ID
--client-secret           KEYCLOAK_CLIENT_SECRET
--public-base-url          PUBLIC_BASE_URL
--redirect-url             KEYCLOAK_REDIRECT_URL
--after-login-url          AFTER_LOGIN_URL
--after-logout-url         AFTER_LOGOUT_URL
--allow-insecure-http      ALLOW_INSECURE_HTTP
--session-db-dsn           SESSION_DB_DSN
--audit-db-dsn             AUDIT_DB_DSN
--app-db-dsn               APPAUTH_DB_DSN
--capability-db-dsn        CAPABILITY_DB_DSN
```

`public-base-url` is required in production. It is the browser-visible origin behind ingress. Do not derive callback URLs from `--listen`.

For local generated-host smokes:

```bash
make -C examples/xgoja/21-generated-host-auth smoke
make -C examples/xgoja/21-generated-host-auth compose-smoke
```

The first command validates generated build wiring with a fake discovery endpoint. The second command reuses the local Keycloak/Postgres Docker Compose stack, drives a real login, seeds demo appauth rows, and exercises the JavaScript-owned audit and capability routes.

For local Docker Compose with the original example 19 host:

```bash
go run ./examples/xgoja/19-express-keycloak-auth-host/cmd/host serve \
  --listen 127.0.0.1:8790 \
  --public-base-url http://127.0.0.1:8790 \
  --allow-insecure-http
```

For Kubernetes:

```text
PUBLIC_BASE_URL=https://goja-auth.yolo.scapegoat.dev
ALLOW_INSECURE_HTTP=false
```

The derived redirect URL is:

```text
https://goja-auth.yolo.scapegoat.dev/auth/callback
```

## Step 2: build and publish the image

`Dockerfile.auth-host` builds the example host and copies the route script into the runtime image:

```dockerfile
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/goja-auth-host ./examples/xgoja/19-express-keycloak-auth-host/cmd/host
COPY --from=builder /out/goja-auth-host /app/goja-auth-host
COPY examples/xgoja/19-express-keycloak-auth-host/scripts/server.js /app/server.js
ENTRYPOINT ["/app/goja-auth-host", "serve"]
CMD ["--listen", ":8080", "--script", "/app/server.js"]
```

Build and push:

```bash
docker build -f Dockerfile.auth-host \
  -t ghcr.io/go-go-golems/go-go-goja-auth-host:sha-<short-sha> .
docker push ghcr.io/go-go-golems/go-go-goja-auth-host:sha-<short-sha>
```

The workflow `.github/workflows/publish-auth-host-image.yaml` performs the same build in CI and can open a GitOps PR through `deploy/gitops-targets.json`.

## Step 3: create the GitOps package

The K3s package is a normal Kustomize directory:

```text
gitops/kustomize/goja-auth-host-demo/
  namespace.yaml
  serviceaccount.yaml
  db-bootstrap-serviceaccount.yaml
  vault-connection.yaml
  vault-auth.yaml
  db-bootstrap-vault-auth.yaml
  runtime-secret.yaml
  image-pull-secret.yaml
  postgres-admin-secret.yaml
  db-bootstrap-script-configmap.yaml
  db-bootstrap-job.yaml
  deployment.yaml
  service.yaml
  ingress.yaml
```

The Deployment uses the pushed image and reads runtime settings from the VSO-rendered Secret:

```yaml
image: ghcr.io/go-go-golems/go-go-goja-auth-host:sha-ba77afc
args:
  - --listen
  - :8080
  - --script
  - /app/server.js
env:
  - name: KEYCLOAK_ISSUER
    valueFrom:
      secretKeyRef:
        name: goja-auth-host-demo-runtime
        key: keycloak_issuer
```

The `args` field intentionally does not include `serve`. The image ENTRYPOINT already includes the subcommand.

## Step 4: seed Vault

Create the runtime secret:

```bash
export VAULT_ADDR=https://vault.yolo.scapegoat.dev
export VAULT_TOKEN="$(cat ~/.vault-token)"

GOJA_AUTH_HOST_DEMO_KEYCLOAK_ISSUER=https://auth.yolo.scapegoat.dev/realms/goja-auth-host-demo \
GOJA_AUTH_HOST_DEMO_KEYCLOAK_CLIENT_ID=goja-auth-host-demo \
GOJA_AUTH_HOST_DEMO_KEYCLOAK_CLIENT_SECRET=<client-secret> \
./scripts/bootstrap-goja-auth-host-demo-runtime-secrets.sh
```

Create the GHCR pull secret:

```bash
GITHUB_DEPLOY_PAT="$(gh auth token)" GITHUB_DEPLOY_USERNAME=wesen \
  ./scripts/bootstrap-goja-auth-host-demo-image-pull-secret.sh
```

The runtime secret contains the database DSN and Keycloak fields. The image-pull secret contains Docker auth fields and is transformed by VSO into a `kubernetes.io/dockerconfigjson` Secret.

## Step 5: provision Keycloak

The live demo currently uses manual `kcadm.sh` provisioning:

```text
Realm:        goja-auth-host-demo
Client:       goja-auth-host-demo
Access type:  confidential
Redirect URI: https://goja-auth.yolo.scapegoat.dev/auth/callback
Web origin:   https://goja-auth.yolo.scapegoat.dev
```

For a long-lived deployment, move this state into the Terraform Keycloak repository. The client secret must be copied into Vault after creation.

## Step 6: bootstrap PostgreSQL

The GitOps package includes an Argo sync-hook Job that creates or updates the database role and database. The app runtime Secret supplies:

```text
database = goja_auth_host_demo
username = goja_auth_host_demo_app
password = generated
```

The bootstrap Job reads the shared Postgres admin secret from Vault and runs `psql`. After that, the app starts and each auth store applies its own schema.

## Step 7: sync Argo CD

Apply or merge the Argo Application:

```bash
kubectl apply -f gitops/applications/goja-auth-host-demo.yaml
kubectl -n argocd annotate application goja-auth-host-demo argocd.argoproj.io/refresh=hard --overwrite
```

For pre-merge validation, the live Application may temporarily target a feature branch. After merge, switch it back to `main`.

Check status:

```bash
kubectl -n argocd get application goja-auth-host-demo
kubectl -n goja-auth-host-demo get pods,svc,ingress,certificate,vaultauth,vaultstaticsecret
```

The expected state is `Synced Healthy`, one running app Pod, a ready TLS certificate, and VSO resources with `SYNCED=True`, `HEALTHY=True`, and `READY=True`.

## Step 8: validate the public auth flow

Start with health:

```bash
curl -fsS https://goja-auth.yolo.scapegoat.dev/healthz
```

Check the login redirect with GET:

```bash
curl -sS -D - -o /dev/null https://goja-auth.yolo.scapegoat.dev/auth/login
```

Run the full smoke test:

```bash
python3 examples/xgoja/19-express-keycloak-auth-host/scripts/keycloak_smoke.py \
  --base-url https://goja-auth.yolo.scapegoat.dev \
  --username demo-user \
  --password "$(VAULT_TOKEN=$(cat ~/.vault-token) vault kv get -field=demo_password kv/apps/goja-auth-host-demo/prod/runtime)"
```

A passing smoke proves more than liveness. It proves public routes, unauthenticated denial, Keycloak login, session creation, CSRF enforcement, authorization, capability token behavior, logout, and post-logout denial.

## Troubleshooting

| Problem | Cause | Solution |
| --- | --- | --- |
| Pod exits with `Too many arguments` | The Deployment passed `serve` even though the image ENTRYPOINT already includes it. | Remove `serve` from Kubernetes args. |
| Vault bootstrap says `VAULT_TOKEN required` | Operator shell has no token. | Export `VAULT_TOKEN=$(cat ~/.vault-token)` or log in with OIDC. |
| Image pull secret bootstrap says `GITHUB_DEPLOY_PAT` is missing | No GHCR credential was provided. | Export a token with package read access. |
| Keycloak callback fails | Redirect URI or client secret mismatch. | Compare Keycloak client config, Vault secret, and `PUBLIC_BASE_URL`. |
| `curl -I /auth/login` returns 405 | HEAD is not the login method. | Use GET for redirect checks. |
| Smoke hangs after success | Server may not handle SIGTERM. | Add signal-aware `http.Server.Shutdown`. |

## Migrating from the original live image

The deployed service is functional, but the original image is still the hand-composed example-19 architecture. To migrate it to the generated architecture, build a generated binary from an `xgoja.yaml` like `examples/xgoja/21-generated-host-auth/xgoja.yaml`, keep the same HTTPS `public-base-url`, Keycloak client, Vault secret fields, and Postgres DSN, then rerun the public smoke test. Keep replicas at 1 until OIDC transaction storage is durable across pods.

## See also

- `goja-repl help auth-module-guide`
- `goja-repl help express-auth-user-guide`
- `goja-repl help express-auth-examples`
- `xgoja help generated-auth-javascript-apis`
- `xgoja help express-auth-host-integration-guide`
- `xgoja help hostauth-config-reference`
- `xgoja help auth-host-production-runbook`
