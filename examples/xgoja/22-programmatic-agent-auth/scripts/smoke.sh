#!/usr/bin/env bash
set -euo pipefail

SERVER_BIN=${1:?usage: smoke.sh /path/to/server-binary /path/to/agent-binary}
AGENT_BIN=${2:?usage: smoke.sh /path/to/server-binary /path/to/agent-binary}
addr="127.0.0.1:$(python3 - <<'PY'
import socket
s=socket.socket()
s.bind(("127.0.0.1", 0))
print(s.getsockname()[1])
s.close()
PY
)"
base_url="http://${addr}"
token_file="$(mktemp)"
log="$(mktemp)"
agent_out="$(mktemp)"
no_token_out="$(mktemp)"
session_out="$(mktemp)"
device_start_out="$(mktemp)"
device_poll_out="$(mktemp)"

cleanup() {
  if [[ -n "${pid:-}" ]]; then
    kill "$pid" >/dev/null 2>&1 || true
    wait "$pid" >/dev/null 2>&1 || true
  fi
  rm -f "$token_file" "$log" "$agent_out" "$no_token_out" "$session_out" "$device_start_out" "$device_poll_out"
}
trap cleanup EXIT

"$SERVER_BIN" serve agentauth server --http-listen "$addr" --token-file "$token_file" >"$log" 2>&1 &
pid=$!

for _ in $(seq 1 100); do
  if curl -fsS "$base_url/healthz" >/dev/null 2>&1; then
    break
  fi
  if ! kill -0 "$pid" >/dev/null 2>&1; then
    cat "$log"
    exit 1
  fi
  sleep 0.1
done

if [[ ! -s "$token_file" ]]; then
  echo "token file was not written: $token_file" >&2
  cat "$log" >&2
  exit 1
fi

grep -q '"value"' "$token_file"
grep -q '"agent"' "$token_file"

curl -fsS -H 'Content-Type: application/json' \
  -d '{"clientName":"smoke-device","tenantId":"o1","actions":["user.self.read"]}' \
  "$base_url/auth/device/start" >"$device_start_out"
grep -q '"device_code"' "$device_start_out"
grep -q '"user_code"' "$device_start_out"
device_code=$(python3 - <<PY
import json
print(json.load(open("$device_start_out"))["device_code"])
PY
)
status=$(curl -sS -H 'Content-Type: application/json' \
  -d "{\"grant_type\":\"urn:ietf:params:oauth:grant-type:device_code\",\"device_code\":\"${device_code}\"}" \
  -o "$device_poll_out" -w '%{http_code}' "$base_url/auth/device/token")
test "$status" = "400"
grep -q '"authorization_pending"' "$device_poll_out"

status=$(curl -sS -o "$no_token_out" -w '%{http_code}' "$base_url/agent/reports/daily")
test "$status" = "401"

"$AGENT_BIN" verbs agentauth call-report --base-url "$base_url" --token-file "$token_file" --report-id daily >"$agent_out"
grep -q '"ok": true' "$agent_out"
grep -q '"reportId": "daily"' "$agent_out"
grep -q '"authMethod": "apiToken"' "$agent_out"
grep -q '"principalKind": "agent"' "$agent_out"
grep -q '"sessionOnlyStatus": 403' "$agent_out"

raw_token=$(python3 - <<PY
import json
print(json.load(open("$token_file"))["token"]["value"])
PY
)
status=$(curl -sS -H "Authorization: Bearer ${raw_token}" -o "$session_out" -w '%{http_code}' "$base_url/session-only")
test "$status" = "403"

echo "programmatic agent auth smoke passed: $base_url"
