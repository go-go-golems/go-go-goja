#!/usr/bin/env bash
set -euo pipefail
ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"
printf '# Independent static review probes\n\n'
printf 'Date: %s\n' "$(date -Is)"
printf '\n## Auth/security-sensitive grep\n\n'
rg -n 'TODO|FIXME|panic\(|http\.Error|Authorization|Cookie|SameSite|csrf|CSRF|secret|token|password|random|rand\.|nonce|state|pkce|PKCE|Close\(|PingContext|sql\.Open|TODO\(security\)' \
  pkg/gojahttp modules/express pkg/xgoja/hostauth pkg/xgoja/providers/http examples/xgoja \
  -S || true
printf '\n## Public legacy route calls in docs/examples/modules\n\n'
rg -n 'app\.(get|post|put|patch|delete|all)\([^\n]*function|app\.(get|post|put|patch|delete|all)\([^\n]*=>|app\.(get|post|put|patch|delete|all)\([^\n]*handler' \
  pkg/doc examples modules -S || true
printf '\n## Line anchors for core symbols\n\n'
for pattern in \
  'type RoutePlan' \
  'func ValidateRoutePlan' \
  'func \(.*\) buildSecureEnvelope' \
  'func \(.*\) servePlannedRoute' \
  'func \(.*\) RegisterPlanned' \
  'func NewBuilderStore' \
  'func \(.*\) BuildHostAuthServices' \
  'func hostOptionsWithAuth' \
  'func serveVerb' \
  'func serveVerbHotReload'; do
  printf '\n### %s\n' "$pattern"
  rg -n "$pattern" pkg/gojahttp modules/express pkg/xgoja/hostauth pkg/xgoja/providers/http -S || true
done
