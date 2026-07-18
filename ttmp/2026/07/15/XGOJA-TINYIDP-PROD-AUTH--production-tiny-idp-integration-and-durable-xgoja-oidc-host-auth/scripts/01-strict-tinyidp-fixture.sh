#!/usr/bin/env bash
# Start a disposable strict tiny-idp issuer and run a generated-host test command
# against it. The supplied command receives TINYIDP_ISSUER, TINYIDP_CLIENT_ID,
# TINYIDP_APP_BASE_URL, and SSL_CERT_FILE. It must start or exercise the xgoja
# host itself and register the callback/post-logout URLs shown below.
set -euo pipefail

if [[ $# -eq 0 ]]; then
  echo "usage: $0 <generated-host test command> [args...]" >&2
  exit 2
fi

TINYIDP_ROOT=${TINYIDP_ROOT:?set TINYIDP_ROOT to the strict tiny-idp checkout}
TINYIDP_ADDR=${TINYIDP_ADDR:-127.0.0.1:19443}
TINYIDP_APP_BASE_URL=${TINYIDP_APP_BASE_URL:-http://127.0.0.1:19797}
TINYIDP_CLIENT_ID=${TINYIDP_CLIENT_ID:-xgoja-oidc-fixture}
TINYIDP_LOGIN=${TINYIDP_LOGIN:-alice}
TINYIDP_PASSWORD=${TINYIDP_PASSWORD:-alice-password-2026}
TINYIDP_EMAIL=${TINYIDP_EMAIL:-alice@example.test}
TINYIDP_BOB_LOGIN=${TINYIDP_BOB_LOGIN:-bob}
TINYIDP_BOB_PASSWORD=${TINYIDP_BOB_PASSWORD:-bob-password-2026}
TINYIDP_BOB_EMAIL=${TINYIDP_BOB_EMAIL:-bob@example.test}
TINYIDP_ISSUER=${TINYIDP_ISSUER:-https://127.0.0.1:19443}

for program in go openssl curl; do
  command -v "$program" >/dev/null || { echo "required program not found: $program" >&2; exit 1; }
done
[[ -d "$TINYIDP_ROOT" ]] || { echo "TINYIDP_ROOT not found: $TINYIDP_ROOT" >&2; exit 1; }

workdir=$(mktemp -d)
tinyidp_bin="$workdir/tinyidp"
idp_pid=
cleanup() {
  local status=$?
  if [[ "${KEEP_FIXTURE_DIR:-}" == "1" ]]; then
    echo "preserved strict tiny-idp fixture directory: $workdir (pid $idp_pid)" >&2
    return "$status"
  fi
  if [[ -n "$idp_pid" ]]; then
    kill "$idp_pid" >/dev/null 2>&1 || true
    wait "$idp_pid" >/dev/null 2>&1 || true
  fi
  rm -rf "$workdir"
  return "$status"
}
trap cleanup EXIT

openssl req -x509 -newkey rsa:2048 -nodes -days 1 \
  -keyout "$workdir/tls.key" \
  -out "$workdir/tls.crt" \
  -subj '/CN=localhost' \
  -addext 'subjectAltName=DNS:localhost,IP:127.0.0.1' \
  >/dev/null 2>&1
printf '0123456789abcdef0123456789abcdef' >"$workdir/token-secret"
chmod 0600 "$workdir/token-secret"

(cd "$TINYIDP_ROOT" && go build -o "$tinyidp_bin" ./cmd/tinyidp)
"$tinyidp_bin" admin --db "$workdir/tinyidp.db" init --generate-signing-key --kid fixture-rsa-1 >/dev/null
"$tinyidp_bin" admin --db "$workdir/tinyidp.db" client create \
  --id "$TINYIDP_CLIENT_ID" \
  --public \
  --redirect-uri "$TINYIDP_APP_BASE_URL/auth/callback" \
  --post-logout-redirect-uri "$TINYIDP_APP_BASE_URL/" \
  --grant-type authorization_code \
  --scope openid --scope profile --scope email \
  --require-pkce \
  >/dev/null
printf '%s\n' "$TINYIDP_PASSWORD" | "$tinyidp_bin" admin --db "$workdir/tinyidp.db" user create \
  --login "$TINYIDP_LOGIN" \
  --email "$TINYIDP_EMAIL" \
  --email-verified \
  --name 'Fixture Alice' \
  --password-from-stdin \
  >/dev/null
printf '%s\n' "$TINYIDP_BOB_PASSWORD" | "$tinyidp_bin" admin --db "$workdir/tinyidp.db" user create \
  --login "$TINYIDP_BOB_LOGIN" \
  --email "$TINYIDP_BOB_EMAIL" \
  --email-verified \
  --name 'Fixture Bob' \
  --password-from-stdin \
  >/dev/null

"$tinyidp_bin" serve-production \
  --addr "$TINYIDP_ADDR" \
  --issuer "$TINYIDP_ISSUER" \
  --db "$workdir/tinyidp.db" \
  --audit-path "$workdir/audit.jsonl" \
  --token-secret-file "$workdir/token-secret" \
  --tls-cert "$workdir/tls.crt" \
  --tls-key "$workdir/tls.key" \
  >"$workdir/tinyidp.log" 2>&1 &
idp_pid=$!

for _ in $(seq 1 80); do
  if curl --fail --silent --show-error --cacert "$workdir/tls.crt" "$TINYIDP_ISSUER/.well-known/openid-configuration" >/dev/null; then
    break
  fi
  if ! kill -0 "$idp_pid" >/dev/null 2>&1; then
    cat "$workdir/tinyidp.log" >&2
    exit 1
  fi
  sleep 0.25
done
if ! curl --fail --silent --show-error --cacert "$workdir/tls.crt" "$TINYIDP_ISSUER/.well-known/openid-configuration" >/dev/null; then
  cat "$workdir/tinyidp.log" >&2
  exit 1
fi

export TINYIDP_ISSUER TINYIDP_CLIENT_ID TINYIDP_APP_BASE_URL TINYIDP_LOGIN TINYIDP_PASSWORD TINYIDP_EMAIL
export TINYIDP_BOB_LOGIN TINYIDP_BOB_PASSWORD TINYIDP_BOB_EMAIL
export SSL_CERT_FILE="$workdir/tls.crt"
"$@"
