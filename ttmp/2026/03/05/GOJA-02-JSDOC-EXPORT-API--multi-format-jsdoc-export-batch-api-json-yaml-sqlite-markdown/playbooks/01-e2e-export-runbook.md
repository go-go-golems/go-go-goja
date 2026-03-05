---
Title: "E2E Runbook: GOJA-02 batch extract/export (CLI + HTTP)"
Ticket: GOJA-02-JSDOC-EXPORT-API
Status: active
Topics:
  - goja
  - tooling
  - playbook
DocType: playbook
Intent: short-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: >
  Manual, copy/paste-ready runbook to validate GOJA-02 batch extraction and
  multi-format exports via both the goja-jsdoc CLI and the server’s /api/batch/*
  endpoints.
LastUpdated: 2026-03-05T00:00:00-05:00
---

# E2E Runbook: GOJA-02 batch extract/export (CLI + HTTP)

This playbook is a manual checklist to validate:

- CLI: `goja-jsdoc export`
- HTTP: `POST /api/batch/extract` and `POST /api/batch/export`

It is designed for quick “does this still work?” checks after refactors.

## Preconditions

- You can run Go commands in the workspace that contains `go-go-goja/`.
- You have `curl` installed.
- Optional: `jq` installed (for JSON readability).

## 1) CLI export checks

Use the migrated fixtures from GOJA-01:

```bash
cd /home/manuel/workspaces/2026-03-05/add-jsdocex/go-go-goja
ls testdata/jsdoc/*.js
```

### JSON export (store shape)

```bash
go run ./cmd/goja-jsdoc export testdata/jsdoc/basic_symbol.js --format json --shape store --pretty | jq .
```

### YAML export (files shape)

```bash
go run ./cmd/goja-jsdoc export testdata/jsdoc/basic_symbol.js --format yaml --shape files
```

### Markdown export

```bash
rm -f /tmp/jsdoc-export.md
go run ./cmd/goja-jsdoc export testdata/jsdoc/basic_symbol.js --format markdown --toc-depth 3 --output-file /tmp/jsdoc-export.md
wc -c /tmp/jsdoc-export.md
```

### SQLite export

```bash
rm -f /tmp/jsdoc-export.sqlite
go run ./cmd/goja-jsdoc export testdata/jsdoc/basic_symbol.js --format sqlite --output-file /tmp/jsdoc-export.sqlite
ls -lh /tmp/jsdoc-export.sqlite
```

## 2) Server export checks

Start the server against a directory. For path safety, the batch API only allows **relative** paths that resolve under this directory.

```bash
cd /home/manuel/workspaces/2026-03-05/add-jsdocex/go-go-goja
go run ./cmd/goja-jsdoc serve --dir ./testdata/jsdoc --host 127.0.0.1 --port 8090
```

### Batch extract (JSON)

```bash
curl -sS -X POST http://127.0.0.1:8090/api/batch/extract \
  -H 'Content-Type: application/json' \
  -d '{"inputs":[{"path":"basic_symbol.js"}]}' | jq '.store.by_symbol | keys'
```

### Batch export (Markdown)

```bash
curl -sS -X POST http://127.0.0.1:8090/api/batch/export \
  -H 'Content-Type: application/json' \
  -d '{"inputs":[{"path":"basic_symbol.js"}],"format":"markdown","options":{"tocDepth":2}}' \
  > /tmp/jsdoc-export-api.md
wc -c /tmp/jsdoc-export-api.md
```

### Batch export (SQLite)

```bash
curl -sS -D /tmp/jsdoc-export-api.headers -X POST http://127.0.0.1:8090/api/batch/export \
  -H 'Content-Type: application/json' \
  -d '{"inputs":[{"path":"basic_symbol.js"}],"format":"sqlite"}' \
  > /tmp/jsdoc-export-api.sqlite
grep -i '^content-type\\|^content-disposition\\|^x-jsdoc-error-count' /tmp/jsdoc-export-api.headers || true
ls -lh /tmp/jsdoc-export-api.sqlite
```

### Batch export with inline content (no filesystem)

```bash
curl -sS -X POST http://127.0.0.1:8090/api/batch/export \
  -H 'Content-Type: application/json' \
  -d '{"inputs":[{"displayName":"inline.js","content":"__doc__({\\\"name\\\":\\\"fn\\\",\\\"summary\\\":\\\"hello\\\"})"}],"format":"json","options":{"pretty":true}}' | jq .
```

## 3) Negative checks (safety)

Traversal should be rejected:

```bash
curl -sS -o /tmp/jsdoc-export-api.err -w '%{http_code}\\n' -X POST http://127.0.0.1:8090/api/batch/extract \
  -H 'Content-Type: application/json' \
  -d '{"inputs":[{"path":"../secrets.js"}]}'
cat /tmp/jsdoc-export-api.err
```

