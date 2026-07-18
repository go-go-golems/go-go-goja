#!/usr/bin/env bash
set -euo pipefail

# Inspect, without sending a token, whether an issuer advertises the contract
# required by the Phase 5 oidcresource design. This is a research probe, not a
# login or authorization test.
usage() {
  echo "usage: $0 [--require-introspection] ISSUER_URL" >&2
  exit 64
}

require_introspection=false
if [[ "${1:-}" == "--require-introspection" ]]; then
  require_introspection=true
  shift
fi
[[ $# -eq 1 ]] || usage

issuer="${1%/}"
document="$(curl --fail --silent --show-error "${issuer}/.well-known/openid-configuration")"

printf '%s\n' "$document" | jq '{issuer, userinfo_endpoint, introspection_endpoint, introspection_endpoint_auth_methods_supported, grant_types_supported, scopes_supported}'

endpoint="$(printf '%s\n' "$document" | jq -r '.introspection_endpoint // empty')"
if [[ -z "$endpoint" ]]; then
  echo "Phase 5 contract not available: discovery does not advertise introspection_endpoint." >&2
  if [[ "$require_introspection" == true ]]; then
    exit 1
  fi
  exit 0
fi

echo "Phase 5 introspection endpoint advertised: $endpoint" >&2
