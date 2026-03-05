# Parity Runbook: jsdocex vs goja-jsdoc

This playbook is a manual checklist to validate that the migrated implementation in `go-go-goja` behaves like the original `jsdocex` tool for:

- extraction (`extract`)
- web server UI + JSON API + SSE reload (`serve`)

## Preconditions

- You are in the workspace that contains both modules:
  - `jsdocex/`
  - `go-go-goja/`
- You have Go installed and can run `go run ...`.

## 1) Extract parity (JSON diff)

Run extraction on each fixture and diff the JSON output.

```bash
# From workspace root
set -euo pipefail

rm -rf /tmp/jsdoc-parity
mkdir -p /tmp/jsdoc-parity/jsdocex /tmp/jsdoc-parity/goja-jsdoc

for f in jsdocex/samples/*.js; do
  base="$(basename "$f")"
  go run ./jsdocex/cmd/jsdocex extract "$f" > "/tmp/jsdoc-parity/jsdocex/$base.json"
  go run ./go-go-goja/cmd/goja-jsdoc extract --file "$f" --pretty > "/tmp/jsdoc-parity/goja-jsdoc/$base.json"
  diff -u "/tmp/jsdoc-parity/jsdocex/$base.json" "/tmp/jsdoc-parity/goja-jsdoc/$base.json" || true
done
```

Notes:
- Some differences may be acceptable if they are purely path normalization (relative vs absolute) or ordering artifacts; document any diffs and decide if they matter.

## 2) Server parity (API routes)

Run both servers pointed at the same directory and compare key endpoints.

### Start jsdocex server

```bash
go run ./jsdocex/cmd/jsdocex serve ./jsdocex/samples 8081
```

### Start goja-jsdoc server

```bash
go run ./go-go-goja/cmd/goja-jsdoc serve --dir ./jsdocex/samples --host 127.0.0.1 --port 8082
```

### Compare endpoints

```bash
curl -sS http://127.0.0.1:8081/api/store | jq . > /tmp/jsdoc-parity-store-jsdocex.json
curl -sS http://127.0.0.1:8082/api/store | jq . > /tmp/jsdoc-parity-store-goja.json
diff -u /tmp/jsdoc-parity-store-jsdocex.json /tmp/jsdoc-parity-store-goja.json || true

curl -sS http://127.0.0.1:8081/api/symbol/smoothstep | jq . > /tmp/jsdoc-parity-sym-jsdocex.json
curl -sS http://127.0.0.1:8082/api/symbol/smoothstep | jq . > /tmp/jsdoc-parity-sym-goja.json
diff -u /tmp/jsdoc-parity-sym-jsdocex.json /tmp/jsdoc-parity-sym-goja.json || true
```

## 3) SSE live reload (manual)

1. Open the UI for each server in a browser:
   - jsdocex: `http://127.0.0.1:8081/`
   - goja-jsdoc: `http://127.0.0.1:8082/`
2. Edit a sample file (e.g., add a new `__doc__` block) and save:
   - verify each UI shows a reload badge and updates after reload.

