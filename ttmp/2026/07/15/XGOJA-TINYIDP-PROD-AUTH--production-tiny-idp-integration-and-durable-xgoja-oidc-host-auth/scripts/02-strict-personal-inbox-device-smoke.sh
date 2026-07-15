#!/usr/bin/env bash
# Exercise the existing Step 08 xgoja inbox against the strict tiny-idp
# fixture. The fixture owns TLS tiny-idp provisioning; this script owns the
# generated host, its disposable application/auth databases, and the browser+
# CLI-style device-flow isolation assertion.
set -euo pipefail

if [[ $# -ne 2 ]]; then
  echo "usage: $0 <personal-inbox-binary> <personal-inbox-example-dir>" >&2
  exit 2
fi

bin=$1
example_dir=$2
app_addr=${TINYIDP_APP_ADDR:-127.0.0.1:19797}
app_base_url=${TINYIDP_APP_BASE_URL:-http://$app_addr}
db=$(mktemp -u /tmp/xgoja-strict-inbox-app-XXXXXX.sqlite)
authdb=$(mktemp -u /tmp/xgoja-strict-inbox-auth-XXXXXX.sqlite)
log=$(mktemp)
app_pid=

cleanup() {
  status=$?
  if [[ -n "$app_pid" ]]; then
    kill "$app_pid" >/dev/null 2>&1 || true
    wait "$app_pid" >/dev/null 2>&1 || true
  fi
  rm -f "$db" "$authdb" "$log"
  return "$status"
}
trap cleanup EXIT

"$bin" serve inbox server --http-listen "$app_addr" --db "$db" \
  --auth-oidc-issuer-url "$TINYIDP_ISSUER" \
  --auth-oidc-client-id "$TINYIDP_CLIENT_ID" \
  --auth-oidc-public-base-url "$app_base_url" \
  --auth-session-cookie-allow-insecure-http=true \
  --auth-default-store-driver sqlite \
  --auth-default-store-dsn "$authdb" \
  --auth-default-store-apply-schema=true \
  >"$log" 2>&1 &
app_pid=$!

for _ in $(seq 1 120); do
  if curl --fail --silent --show-error "$app_base_url/healthz" >/dev/null; then
    break
  fi
  if ! kill -0 "$app_pid" >/dev/null 2>&1; then
    cat "$log" >&2
    exit 1
  fi
  sleep 0.25
done
curl --fail --silent --show-error "$app_base_url/auth/readyz" | grep -q '"ready":true'

cli_device=$("$bin" verbs inboxctl device-start --base-url "$app_base_url")
cli_device_code=$(python3 -c 'import json,sys; print(json.load(sys.stdin)["device_code"])' <<<"$cli_device")
if "$bin" verbs inboxctl device-token --base-url "$app_base_url" --device-code "$cli_device_code" >/dev/null 2>&1; then
  echo "CLI device poll unexpectedly succeeded before browser approval" >&2
  exit 1
fi

python3 "$example_dir/../06-browser-login-keycloak/scripts/tinyidp_device_capture_isolation_smoke.py" \
  --base-url "$app_base_url" \
  --ca-file "$SSL_CERT_FILE" \
  --alice-login "$TINYIDP_LOGIN" \
  --alice-password "$TINYIDP_PASSWORD" \
  --alice-email "$TINYIDP_EMAIL" \
  --bob-login "$TINYIDP_BOB_LOGIN" \
  --bob-password "$TINYIDP_BOB_PASSWORD" \
  --bob-email "$TINYIDP_BOB_EMAIL"

script_dir=$(cd "$(dirname "$0")" && pwd)
PLAYWRIGHT_CHROMIUM_EXECUTABLE=${PLAYWRIGHT_CHROMIUM_EXECUTABLE:-/usr/bin/chromium-browser} \
pnpm --dir "$script_dir" exec playwright test "$script_dir/03-personal-inbox-ui.spec.js" --reporter=line

echo "ok strict tinyidp personal-inbox browser and device-flow smoke"
