#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../../.." && pwd)"
cd "$repo_root"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
mkdir -p "$tmp/site-a" "$tmp/site-b" "$tmp/build"

cat > "$tmp/site-a/a.js" <<'JS'
__package__({ name: "sitea" })
__verb__("serve", { name: "serve", output: "text", short: "Serve A" })
function serve() { return "a" }
JS

cat > "$tmp/site-b/b.js" <<'JS'
__package__({ name: "siteb" })
__verb__("serve", { name: "serve", output: "text", short: "Serve B" })
function serve() { return "b" }
JS

cat > "$tmp/xgoja.yaml" <<YAML
schema: xgoja/v2
name: source-loss-repro
go:
  module: xgoja.generated/source-loss-repro
  version: "1.26"
workspace:
  mode: auto
providers:
  - id: http
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/http
runtime:
  modules:
    - provider: http
      name: express
sources:
  - id: site-a
    kind: jsverbs
    from:
      dir: $tmp/site-a
    language: javascript
  - id: site-b
    kind: jsverbs
    from:
      dir: $tmp/site-b
    language: javascript
commands:
  - id: serve
    type: provider.command-set
    provider: http
    name: serve
    mount: serve
    sources: [site-a]
artifacts:
  - id: binary
    type: binary
    output: $tmp/source-loss-repro
    sources: [site-a, site-b]
YAML

GOWORK=off go run ./cmd/xgoja build -f "$tmp/xgoja.yaml" --work-dir "$tmp/build" --xgoja-replace "$repo_root" --keep-work >/tmp/xgoja-source-loss-build.log

python3 - "$tmp/build/xgoja.gen.json" <<'PY'
import json, sys
p = sys.argv[1]
with open(p) as f:
    data = json.load(f)
providers = data.get("commandProviders", [])
jsverbs = data.get("jsverbs", [])
if not providers:
    raise SystemExit("repro failed: generated metadata has no commandProviders")
provider = providers[0]
print("generated command provider metadata:", json.dumps(provider, sort_keys=True))
print("generated top-level jsverb sources:", json.dumps(jsverbs, sort_keys=True))
if "sources" in provider:
    raise SystemExit("unexpected: command provider already preserves sources")
ids = sorted(s.get("id") for s in jsverbs)
if ids != ["site-a", "site-b"]:
    raise SystemExit(f"unexpected jsverb source ids: {ids}")
print("OK: reproduced source loss: command sources [site-a] were dropped, while both jsverb sources remain global")
PY
