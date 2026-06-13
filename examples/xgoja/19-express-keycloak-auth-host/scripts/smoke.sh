#!/usr/bin/env bash
set -euo pipefail

EXAMPLE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPO_ROOT="$(cd "${EXAMPLE_DIR}/../../.." && pwd)"
LISTEN="${LISTEN:-127.0.0.1:8790}"
BASE_URL="${BASE_URL:-http://${LISTEN}}"
KEYCLOAK_PORT="${KEYCLOAK_PORT:-18080}"
POSTGRES_PORT="${POSTGRES_PORT:-15432}"
ISSUER="${ISSUER:-http://127.0.0.1:${KEYCLOAK_PORT}/realms/goja-demo}"
SESSION_DB_DSN="${SESSION_DB_DSN:-postgres://goja:goja@127.0.0.1:${POSTGRES_PORT}/goja_auth?sslmode=disable}"
AUDIT_DB_DSN="${AUDIT_DB_DSN:-${SESSION_DB_DSN}}"
SCRIPT="${SCRIPT:-${EXAMPLE_DIR}/scripts/server.js}"
HOST_LOG="${HOST_LOG:-$(mktemp -t goja-keycloak-host.XXXXXX.log)}"
HOST_BIN="${HOST_BIN:-$(mktemp -t goja-keycloak-host.XXXXXX)}"
HOST_PID=""

cleanup() {
  local code=$?
  if [[ -n "${HOST_PID}" ]] && kill -0 "${HOST_PID}" >/dev/null 2>&1; then
    kill "${HOST_PID}" >/dev/null 2>&1 || true
    wait "${HOST_PID}" >/dev/null 2>&1 || true
  fi
  rm -f "${HOST_BIN}" >/dev/null 2>&1 || true
  if [[ "${KEEP_KEYCLOAK:-0}" != "1" ]]; then
    docker compose -f "${EXAMPLE_DIR}/docker-compose.yml" down -v >/dev/null 2>&1 || true
  fi
  if [[ ${code} -ne 0 ]]; then
    echo "--- host log (${HOST_LOG}) ---" >&2
    tail -200 "${HOST_LOG}" >&2 || true
  else
    echo "host log: ${HOST_LOG}"
  fi
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

wait_for_host_url() {
  local label="$1"
  local url="$2"
  local attempts="${3:-60}"
  for _ in $(seq 1 "${attempts}"); do
    if ! kill -0 "${HOST_PID}" >/dev/null 2>&1; then
      echo "host exited before ${label} became ready" >&2
      return 1
    fi
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
    if docker compose -f "${EXAMPLE_DIR}/docker-compose.yml" exec -T postgres pg_isready -U goja -d goja_auth >/dev/null 2>&1; then
      echo "ok postgres ready"
      return 0
    fi
    sleep 2
  done
  echo "timed out waiting for postgres" >&2
  return 1
}

if [[ "${SKIP_KEYCLOAK_UP:-0}" != "1" ]]; then
  docker compose -f "${EXAMPLE_DIR}/docker-compose.yml" down -v >/dev/null 2>&1 || true
  KEYCLOAK_PORT="${KEYCLOAK_PORT}" POSTGRES_PORT="${POSTGRES_PORT}" docker compose -f "${EXAMPLE_DIR}/docker-compose.yml" up -d
fi
wait_for_postgres 60
wait_for_url "keycloak discovery" "${ISSUER}/.well-known/openid-configuration" 90

(
  cd "${REPO_ROOT}"
  GOWORK=off go build -o "${HOST_BIN}" ./examples/xgoja/19-express-keycloak-auth-host/cmd/host
)
"${HOST_BIN}" \
  --script "${SCRIPT}" \
  --listen "${LISTEN}" \
  --issuer "${ISSUER}" \
  --session-db-dsn "${SESSION_DB_DSN}" \
  --audit-db-dsn "${AUDIT_DB_DSN}" >"${HOST_LOG}" 2>&1 &
HOST_PID=$!

wait_for_host_url "host health" "${BASE_URL}/healthz" 60
python3 "${EXAMPLE_DIR}/scripts/keycloak_smoke.py" --base-url "${BASE_URL}"

audit_count="$(docker compose -f "${EXAMPLE_DIR}/docker-compose.yml" exec -T postgres psql -U goja -d goja_auth -tAc "SELECT count(*) FROM auth_audit_records WHERE event IN ('health.checked', 'user.self.read', 'project.updated')")"
audit_count="${audit_count//[[:space:]]/}"
if [[ -z "${audit_count}" || "${audit_count}" -lt 5 ]]; then
  echo "expected persisted audit records, got count=${audit_count:-<empty>}" >&2
  exit 1
fi
echo "ok persisted audit records ${audit_count}"
