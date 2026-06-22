#!/usr/bin/env bash
set -euo pipefail

EXAMPLE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPO_ROOT="$(cd "${EXAMPLE_DIR}/../../.." && pwd)"
COMPOSE_FILE="${COMPOSE_FILE:-${EXAMPLE_DIR}/docker-compose.yml}"
TINYIDP_ROOT="${TINYIDP_ROOT:-/home/manuel/workspaces/2026-06-20/ui-notebook-package/2026-06-22--mock-oidc-idp}"
LISTEN="${LISTEN:-127.0.0.1:8792}"
BASE_URL="${BASE_URL:-http://${LISTEN}}"
TINYIDP_ADDR="${TINYIDP_ADDR:-127.0.0.1:19092}"
ISSUER="${ISSUER:-http://${TINYIDP_ADDR}}"
CLIENT_ID="${CLIENT_ID:-goja-app}"
POSTGRES_PORT="${POSTGRES_PORT:-15434}"
DSN="${SESSION_DB_DSN:-postgres://goja:goja@127.0.0.1:${POSTGRES_PORT}/goja_auth?sslmode=disable}"
HOST_LOG="${HOST_LOG:-$(mktemp -t goja-tinyidp-host.XXXXXX.log)}"
IDP_LOG="${IDP_LOG:-$(mktemp -t goja-tinyidp-idp.XXXXXX.log)}"
TINYIDP_BIN="${TINYIDP_BIN:-$(mktemp -t tinyidp-bin.XXXXXX)}"
HOST_PID=""
IDP_PID=""

cleanup() {
  local code=$?
  if [[ -n "${HOST_PID}" ]] && kill -0 "${HOST_PID}" >/dev/null 2>&1; then
    kill "${HOST_PID}" >/dev/null 2>&1 || true
    wait "${HOST_PID}" >/dev/null 2>&1 || true
  fi
  if [[ -n "${IDP_PID}" ]] && kill -0 "${IDP_PID}" >/dev/null 2>&1; then
    kill "${IDP_PID}" >/dev/null 2>&1 || true
    wait "${IDP_PID}" >/dev/null 2>&1 || true
  fi
  docker compose -f "${COMPOSE_FILE}" down -v >/dev/null 2>&1 || true
  rm -f "${TINYIDP_BIN}"
  if [[ ${code} -ne 0 ]]; then
    echo "--- host log (${HOST_LOG}) ---" >&2
    tail -200 "${HOST_LOG}" >&2 || true
    echo "--- tinyidp log (${IDP_LOG}) ---" >&2
    tail -200 "${IDP_LOG}" >&2 || true
  fi
}
trap cleanup EXIT

wait_for_url() {
  local label="$1" url="$2" attempts="${3:-120}"
  for _ in $(seq 1 "${attempts}"); do
    if curl -fsS "${url}" >/dev/null 2>&1; then
      echo "ok ${label} ${url}"
      return 0
    fi
    sleep 0.25
  done
  echo "timed out waiting for ${label}: ${url}" >&2
  return 1
}

wait_for_postgres() {
  for _ in $(seq 1 60); do
    if docker compose -f "${COMPOSE_FILE}" exec -T postgres pg_isready -U goja -d goja_auth >/dev/null 2>&1; then
      echo "ok postgres ready"
      return 0
    fi
    sleep 1
  done
  echo "timed out waiting for postgres" >&2
  return 1
}

if [[ ! -d "${TINYIDP_ROOT}" ]]; then
  echo "TINYIDP_ROOT not found: ${TINYIDP_ROOT}" >&2
  exit 1
fi

docker compose -f "${COMPOSE_FILE}" down -v >/dev/null 2>&1 || true
POSTGRES_PORT="${POSTGRES_PORT}" docker compose -f "${COMPOSE_FILE}" up -d postgres
wait_for_postgres

(cd "${TINYIDP_ROOT}" && GOWORK=off go build -o "${TINYIDP_BIN}" ./cmd/tinyidp)
"${TINYIDP_BIN}" serve \
  --addr "${TINYIDP_ADDR}" \
  --issuer "${ISSUER}" \
  --client-id "${CLIENT_ID}" \
  --redirect-uris "${BASE_URL}/auth/callback" \
  >"${IDP_LOG}" 2>&1 &
IDP_PID=$!
wait_for_url tinyidp "${ISSUER}/.well-known/openid-configuration" 120

cd "${REPO_ROOT}"
GOWORK=off go run ./examples/xgoja/19-express-keycloak-auth-host/cmd/host serve \
  --script "${EXAMPLE_DIR}/scripts/server.js" \
  --listen "${LISTEN}" \
  --issuer "${ISSUER}" \
  --client-id "${CLIENT_ID}" \
  --public-base-url "${BASE_URL}" \
  --allow-insecure-http \
  --session-db-dsn "${DSN}" \
  --audit-db-dsn "${DSN}" \
  --app-db-dsn "${DSN}" \
  --capability-db-dsn "${DSN}" \
  >"${HOST_LOG}" 2>&1 &
HOST_PID=$!
wait_for_url host "${BASE_URL}/healthz" 120

python3 "${EXAMPLE_DIR}/scripts/keycloak_smoke.py" \
  --base-url "${BASE_URL}" \
  --username demo

echo "ok express auth host tinyidp smoke"
