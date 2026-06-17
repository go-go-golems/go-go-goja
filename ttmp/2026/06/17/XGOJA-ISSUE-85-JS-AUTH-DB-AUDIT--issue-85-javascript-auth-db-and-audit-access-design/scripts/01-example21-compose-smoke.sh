#!/usr/bin/env bash
set -euo pipefail

EXAMPLE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPO_ROOT="$(cd "${EXAMPLE_DIR}/../../.." && pwd)"
COMPOSE_FILE="${COMPOSE_FILE:-${REPO_ROOT}/examples/xgoja/19-express-keycloak-auth-host/docker-compose.yml}"
LISTEN="${LISTEN:-127.0.0.1:8790}"
BASE_URL="${BASE_URL:-http://${LISTEN}}"
KEYCLOAK_PORT="${KEYCLOAK_PORT:-18080}"
POSTGRES_PORT="${POSTGRES_PORT:-15432}"
ISSUER="${ISSUER:-http://127.0.0.1:${KEYCLOAK_PORT}/realms/goja-demo}"
CLIENT_ID="${CLIENT_ID:-goja-app}"
DSN="${AUTH_DB_DSN:-postgres://goja:goja@127.0.0.1:${POSTGRES_PORT}/goja_auth?sslmode=disable}"
BIN="${BIN:-${EXAMPLE_DIR}/dist/generated-oidc-host-auth}"
HOST_LOG="${HOST_LOG:-$(mktemp -t generated-keycloak-host.XXXXXX.log)}"
HOST_PID=""

cleanup() {
  local code=$?
  echo "cleanup: start (exit=${code})" >&2
  if [[ -n "${HOST_PID}" ]] && kill -0 "${HOST_PID}" >/dev/null 2>&1; then
    echo "cleanup: stopping host pid ${HOST_PID}" >&2
    kill "${HOST_PID}" >/dev/null 2>&1 || true
    for _ in $(seq 1 20); do
      if ! kill -0 "${HOST_PID}" >/dev/null 2>&1; then
        break
      fi
      sleep 0.25
    done
    if kill -0 "${HOST_PID}" >/dev/null 2>&1; then
      kill -KILL "${HOST_PID}" >/dev/null 2>&1 || true
    fi
    wait "${HOST_PID}" >/dev/null 2>&1 || true
  fi
  if [[ "${KEEP_KEYCLOAK:-0}" != "1" ]]; then
    echo "cleanup: docker compose down -v" >&2
    docker compose -f "${COMPOSE_FILE}" down -v >/dev/null 2>&1 || true
  fi
  if [[ ${code} -ne 0 ]]; then
    echo "--- host log (${HOST_LOG}) ---" >&2
    tail -200 "${HOST_LOG}" >&2 || true
  else
    echo "host log: ${HOST_LOG}"
  fi
  echo "cleanup: done" >&2
}
trap cleanup EXIT

wait_for_url() {
  local label="$1"
  local url="$2"
  local attempts="${3:-60}"
  for _ in $(seq 1 "${attempts}"); do
    if curl -fsS "${url}" >/dev/null 2>&1; then
      echo "ok ${label} ${url}"
      return 0
    fi
    sleep 2
  done
  echo "timed out waiting for ${label}: ${url}" >&2
  return 1
}

wait_for_postgres() {
  local attempts="${1:-60}"
  for _ in $(seq 1 "${attempts}"); do
    if docker compose -f "${COMPOSE_FILE}" exec -T postgres pg_isready -U goja -d goja_auth >/dev/null 2>&1; then
      echo "ok postgres ready"
      return 0
    fi
    sleep 2
  done
  echo "timed out waiting for postgres" >&2
  return 1
}

wait_for_host() {
  local attempts="${1:-60}"
  for _ in $(seq 1 "${attempts}"); do
    if ! kill -0 "${HOST_PID}" >/dev/null 2>&1; then
      echo "host exited before becoming healthy" >&2
      return 1
    fi
    if curl -fsS "${BASE_URL}/healthz" >/dev/null 2>&1; then
      echo "ok host health ${BASE_URL}/healthz"
      return 0
    fi
    sleep 2
  done
  echo "timed out waiting for host health" >&2
  return 1
}

if [[ "${SKIP_KEYCLOAK_UP:-0}" != "1" ]]; then
  docker compose -f "${COMPOSE_FILE}" down -v >/dev/null 2>&1 || true
  KEYCLOAK_PORT="${KEYCLOAK_PORT}" POSTGRES_PORT="${POSTGRES_PORT}" docker compose -f "${COMPOSE_FILE}" up -d
fi
wait_for_postgres 60
wait_for_url "keycloak discovery" "${ISSUER}/.well-known/openid-configuration" 90

if [[ "${SKIP_BUILD:-0}" != "1" ]]; then
  make -C "${EXAMPLE_DIR}" build
fi

"${BIN}" serve sites demo \
  --http-listen "${LISTEN}" \
  --auth-oidc-issuer-url "${ISSUER}" \
  --auth-oidc-client-id "${CLIENT_ID}" \
  --auth-oidc-public-base-url "${BASE_URL}" \
  --auth-session-cookie-allow-insecure-http=true \
  --auth-default-store-driver postgres \
  --auth-default-store-dsn "${DSN}" \
  --auth-default-store-apply-schema=true \
  --auth-session-store-driver postgres \
  --auth-session-store-dsn "${DSN}" \
  --auth-session-store-apply-schema=true \
  --auth-audit-store-driver postgres \
  --auth-audit-store-dsn "${DSN}" \
  --auth-audit-store-apply-schema=true \
  --auth-appauth-store-driver postgres \
  --auth-appauth-store-dsn "${DSN}" \
  --auth-appauth-store-apply-schema=true \
  --auth-capability-store-driver postgres \
  --auth-capability-store-dsn "${DSN}" \
  --auth-capability-store-apply-schema=true \
  >"${HOST_LOG}" 2>&1 &
HOST_PID=$!

wait_for_host 60
python3 "${EXAMPLE_DIR}/scripts/keycloak_compose_smoke.py" \
  --repo-root "${REPO_ROOT}" \
  --compose-file "${COMPOSE_FILE}" \
  --base-url "${BASE_URL}"
