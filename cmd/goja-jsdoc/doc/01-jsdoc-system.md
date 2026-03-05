---
Title: "goja-jsdoc: JSDoc extraction, exports, and batch API"
Slug: goja-jsdoc-jsdoc-system
Short: "Intern-friendly guide to how goja-jsdoc extracts documentation from JavaScript, builds a DocStore, and exports it via CLI and HTTP."
Topics:
- goja
- jsdoc
- documentation
- export
- http-api
Commands:
- goja-jsdoc
- goja-jsdoc extract
- goja-jsdoc export
- goja-jsdoc serve
Flags:
- --file
- --input
- --dir
- --recursive
- --format
- --shape
- --pretty
- --output-file
- --toc-depth
- --continue-on-error
- --host
- --port
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Application
---

This page explains the “jsdoc system” inside `go-go-goja` and how it is exposed via the `goja-jsdoc` CLI and HTTP server.

It is written for a new intern: it focuses on the “what”, “how”, and “why”, and includes system diagrams, pseudocode, API contracts, and concrete file references.

## What problem does this solve?

Many JavaScript projects contain “documentation metadata” that is not in traditional JSDoc comments. Instead, they embed documentation as structured data near the code. The `goja-jsdoc` tool extracts this metadata into a uniform Go data model (`DocStore`) that you can:

- browse in a web UI,
- query via a JSON API,
- export to durable formats (SQLite, Markdown),
- or consume programmatically from other Go code.

## Architecture at a glance

### Core data flow (CLI and HTTP reuse the same core)

```text
          +-------------------+
inputs -> | pkg/jsdoc/batch   | -> DocStore -> +----------------------------+
          | - validate        |                | pkg/jsdoc/export           |
          | - parse via       |                | - json/yaml/markdown/sqlite|
          |   pkg/jsdoc/extract|               +----------------------------+
          +-------------------+
                      |
                      +-> errors (optional; ContinueOnError)
```

### “Where does each piece live?” (key files)

The system is deliberately split into reusable packages:

- Parsing/extraction:
  - `pkg/jsdoc/extract/extract.go` — tree-sitter-based JavaScript parsing + sentinel extraction.
- Model:
  - `pkg/jsdoc/model/model.go` — types (`Package`, `SymbolDoc`, `Example`, `FileDoc`).
  - `pkg/jsdoc/model/store.go` — aggregate store + indexes (`BySymbol`, `ByPackage`, …).
- Batch builder:
  - `pkg/jsdoc/batch/batch.go` — build a store from multiple inputs (paths and/or inline content).
- Exporters:
  - `pkg/jsdoc/export/export.go` — format dispatcher, JSON/YAML export.
  - `pkg/jsdoc/exportmd/exportmd.go` — deterministic single-file Markdown + ToC.
  - `pkg/jsdoc/exportsq/exportsq.go` — SQLite schema + transactional writer.
- Server:
  - `pkg/jsdoc/server/server.go` — web UI + browse API + SSE live reload.
  - `pkg/jsdoc/server/batch_handlers.go` — batch endpoints (`/api/batch/*`) + path safety.
- CLI:
  - `cmd/goja-jsdoc/main.go` — Cobra root command + help integration.
  - `cmd/goja-jsdoc/extract_command.go` — single-file JSON extractor (parity/debug).
  - `cmd/goja-jsdoc/export_command.go` — batch export command (json/yaml/md/sqlite).
  - `cmd/goja-jsdoc/serve_command.go` — server runner (watch + HTTP).

## How docs are written in JavaScript (sentinel patterns)

The extractor looks for specific “sentinel” calls and tagged templates. These are *runtime-noops* in your JS project (they can be defined as identity functions), but they carry structured documentation metadata.

### `__package__(...)` (package-level metadata)

```js
__package__({
  name: "math",
  title: "Math utilities",
  description: "Functions for interpolation and easing."
})
```

### `__doc__(...)` (symbol-level metadata)

There are two forms:

```js
__doc__("smoothstep", {
  summary: "Smooth interpolation from 0..1",
  concepts: ["interpolation"],
  tags: ["math"],
})
```

Or “name inside object”:

```js
__doc__({
  name: "smoothstep",
  summary: "Smooth interpolation from 0..1",
})
```

### `__example__(...)` (example metadata)

```js
__example__({
  id: "ex-smoothstep-basic",
  title: "Basic smoothstep usage",
  symbols: ["smoothstep"],
})
```

### `doc` tagged template (long-form prose)

The extractor also supports Markdown prose blocks using a `doc` tagged template:

```js
doc`
## Notes

This symbol is commonly used for animation curves.
`
```

Prose gets attached to the closest package/symbol context the extractor can infer (see `pkg/jsdoc/extract/extract.go` for the exact behavior).

## Extraction pipeline (what happens under the hood)

### Step-by-step

1) Read source bytes from a file (or use inline bytes).  
2) Parse JavaScript into an AST via tree-sitter.  
3) Walk the AST and detect sentinel call patterns.  
4) Convert simple JS object literals into JSON-ish strings (heuristic), then `json.Unmarshal` into Go structs.  
5) Produce a `FileDoc` for each input file.  
6) Merge `FileDoc`s into an aggregate `DocStore` with indexes.

### Pseudocode (single file)

```pseudo
function ParseFile(path):
  src = readFile(path)
  tree = treeSitterParseJavaScript(src)
  fileDoc = new FileDoc(filePath=path)

  for each topLevelNode in tree.root.children:
    if node is __package__ call:
      fileDoc.package = parsePackageObject(node.args)
    if node is __doc__ call:
      fileDoc.symbols.append(parseSymbolDoc(node.args, node.location))
    if node is __example__ call:
      fileDoc.examples.append(parseExample(node.args, node.location))
    if node is doc`...` template:
      attachProseToNearestContext(fileDoc, templateText)

  return fileDoc
```

### Pseudocode (batch build)

```pseudo
function BuildStore(inputs, continueOnError):
  store = NewDocStore()
  errors = []

  for each input in inputs:
    try:
      if input.path:
        fd = ParseFile(input.path)
      else if input.content:
        fd = ParseSource(input.displayName, input.content)
      else:
        raise "invalid input"

      store.AddFile(fd)
    catch err:
      if !continueOnError: raise err
      errors.append({inputSummary, errString})

  return {store, errors}
```

## The data model you export and browse

The canonical “in-memory representation” is `DocStore` (`pkg/jsdoc/model/store.go`).

Conceptually:

```text
DocStore
  - Files: []*FileDoc               (the raw per-file docs)
  - ByPackage: map[name]*Package    (index)
  - BySymbol:  map[name]*SymbolDoc  (index)
  - ByExample: map[id]*Example      (index)
  - ByConcept: map[concept][]symbol (concept index)
```

You can think of `Files` as the source of truth, and the `By*` maps as convenience indexes built when `AddFile` is called.

## Export formats (what “export” means)

`pkg/jsdoc/export.Export(ctx, store, writer, opts)` chooses an output format and writes to an `io.Writer`.

Supported formats:

- `json`:
  - either full store (`--shape store`) or just `[]FileDoc` (`--shape files`)
  - optional pretty-print (`--pretty`)
- `yaml`:
  - mirrors JSON shape (store or files)
- `markdown`:
  - deterministic single-file document
  - deterministic ToC derived from the generated headings (`--toc-depth`)
- `sqlite`:
  - normalized starter schema (packages/symbols/examples + join tables)
  - transactional inserts + a few indexes

## CLI usage

### `goja-jsdoc extract` (single file, JSON; primarily parity/debug)

Use this when you want to inspect extraction output for one file.

```bash
goja-jsdoc extract --file path/to/file.js --pretty
goja-jsdoc extract --file path/to/file.js --pretty --output-file /tmp/doc.json
```

### `goja-jsdoc export` (batch build + export)

Use this when you want a durable artifact for multiple files.

#### Export explicit files (positional args and/or `--input`)

```bash
goja-jsdoc export a.js b.js --format json --shape store --pretty
goja-jsdoc export --input a.js --input b.js --format yaml --shape files
goja-jsdoc export a.js --format markdown --toc-depth 3 --output-file docs.md
goja-jsdoc export a.js b.js --format sqlite --output-file docs.sqlite
```

#### Export a whole directory

```bash
goja-jsdoc export --dir ./src --format markdown --output-file docs.md
goja-jsdoc export --dir ./src --recursive --format sqlite --output-file docs.sqlite
```

#### Error handling

By default, the command fails on the first unreadable/unparseable file. If you want partial output:

```bash
goja-jsdoc export --dir ./src --recursive --continue-on-error --format json --pretty
```

### `goja-jsdoc serve` (web UI + JSON API + SSE reload)

This starts:

- the web UI (single-page app),
- browse APIs (`/api/store`, `/api/symbol/...`, …),
- batch APIs (`/api/batch/*`),
- and an SSE stream on `/events` that tells the UI to reload when `.js` files change.

```bash
goja-jsdoc serve --dir ./src --host 127.0.0.1 --port 8080
```

Open: `http://127.0.0.1:8080/`

## HTTP API reference

The server has two “namespaces”:

1) Browse APIs (read from the server’s live store; used by the UI)  
2) Batch APIs (build a store from request inputs and export on-demand)

### Browse API (existing; GET)

- `GET /api/store` → full store JSON
- `GET /api/package/{name}` → one package
- `GET /api/symbol/{name}` → one symbol (includes related examples)
- `GET /api/example/{id}` → one example
- `GET /api/search?q=...` → symbols/examples/packages that match

### Batch API: `POST /api/batch/extract`

Builds a store from the request inputs and returns a JSON `BatchResult`:

```json
{
  "store": { "...DocStore..." },
  "errors": [
    { "input": { "path": "missing.js" }, "error": "reading ..." }
  ]
}
```

Example (path input):

```bash
curl -sS -X POST http://127.0.0.1:8080/api/batch/extract \
  -H 'Content-Type: application/json' \
  -d '{"inputs":[{"path":"file.js"}]}' | jq .
```

Example (inline content input):

```bash
curl -sS -X POST http://127.0.0.1:8080/api/batch/extract \
  -H 'Content-Type: application/json' \
  -d '{"inputs":[{"displayName":"inline.js","content":"__doc__({\"name\":\"fn\"})"}]}' | jq .
```

### Batch API: `POST /api/batch/export`

Builds a store from inputs and returns an export artifact.

Request shape:

```json
{
  "inputs": [
    { "path": "file.js" }
  ],
  "format": "markdown",
  "options": {
    "shape": "store",
    "pretty": true,
    "indent": "  ",
    "tocDepth": 3,
    "continueOnError": false
  }
}
```

Response content types:

- `json` → `application/json`
- `yaml` → `application/yaml`
- `markdown` → `text/markdown; charset=utf-8`
- `sqlite` → `application/octet-stream` + `Content-Disposition: attachment; filename="docs.sqlite"`

Example (markdown):

```bash
curl -sS -X POST http://127.0.0.1:8080/api/batch/export \
  -H 'Content-Type: application/json' \
  -d '{"inputs":[{"path":"file.js"}],"format":"markdown","options":{"tocDepth":2}}'
```

### Server-side path safety (important)

If you send `path` inputs to the server, they are treated as **relative paths under the server root directory** (the `--dir` you started the server with).

The server rejects:

- absolute paths,
- traversal paths like `../secrets.js`,
- paths that resolve outside the allowed root.

If you need to export content that is not on the server filesystem, use `content` inputs.

## Validation and “how do I know it works?”

Fast checks:

```bash
go test ./pkg/jsdoc/batch ./pkg/jsdoc/export ./pkg/jsdoc/exportsq ./pkg/jsdoc/server -count=1
go test ./cmd/goja-jsdoc -count=1
```

Manual end-to-end checks:

- See `ttmp/.../playbooks/01-e2e-export-runbook.md` (it includes copy/paste CLI and curl commands).

## Troubleshooting

| Problem | Likely cause | Solution |
|---|---|---|
| `goja-jsdoc help ...` shows nothing | Docs not loaded into help system | Ensure `cmd/goja-jsdoc/main.go` loads docs and calls `help_cmd.SetupCobraRootCommand` |
| `POST /api/batch/export` returns 400 `unknown format` | `format` string not one of `json|yaml|markdown|sqlite` | Fix request format value |
| `POST /api/batch/extract` returns 400 about traversal | You passed `../...` or absolute path | Use relative paths under the server `--dir`, or send `content` |
| SQLite output seems empty | You exported zero inputs or inputs failed | Check request inputs; if using `continueOnError`, inspect `X-JSDoc-Error-Count` |
| Markdown ToC order changes between runs | Inputs were in different order / map iteration | Use the exporter’s deterministic output; if you see nondeterminism, file a bug with a reproducer |

## See Also

- GOJA-02 runbook (manual checks): `ttmp/2026/03/05/GOJA-02-JSDOC-EXPORT-API--multi-format-jsdoc-export-batch-api-json-yaml-sqlite-markdown/playbooks/01-e2e-export-runbook.md`
- Extractor and sentinel rules: `pkg/jsdoc/extract/extract.go`
- Data model: `pkg/jsdoc/model/model.go`, `pkg/jsdoc/model/store.go`
- Exporters: `pkg/jsdoc/export/export.go`, `pkg/jsdoc/exportmd/exportmd.go`, `pkg/jsdoc/exportsq/exportsq.go`
