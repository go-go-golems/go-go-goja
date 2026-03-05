---
Title: 'Design + implementation guide: integrate jsdocex into go-go-goja'
Ticket: GOJA-01-INTEGRATE-JSDOCEX
Status: active
Topics:
    - goja
    - migration
    - architecture
    - tooling
    - analysis
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/cmd/goja-perf/main.go
      Note: Glazed+Cobra command wiring pattern to follow
    - Path: go-go-goja/cmd/goja-perf/serve_command.go
      Note: Example of a server-style command implemented with Glazed flags
    - Path: go-go-goja/pkg/jsparse/treesitter.go
      Note: Tree-sitter binding choice used in go-go-goja (migration constraint)
    - Path: jsdocex/internal/extractor/extractor.go
      Note: Current sentinel extraction rules and heuristics (parity target)
    - Path: jsdocex/internal/model/model.go
      Note: Current JSON model + DocStore indexing semantics
    - Path: jsdocex/internal/server/server.go
      Note: Current HTTP API routes
    - Path: jsdocex/internal/watcher/watcher.go
      Note: Current fsnotify watcher behavior (subdirs + debounce)
ExternalSources: []
Summary: |
    Evidence-based design + implementation guide for migrating the `jsdocex/` JavaScript documentation extractor and web server into `go-go-goja/` as reusable packages and Glazed CLI commands, while preserving the existing HTTP API and UI behavior.
LastUpdated: 2026-03-05T01:19:46.15191912-05:00
WhatFor: |
    Use this as the source of truth for how `jsdocex/` works today, what must remain stable (API/UI/behavior), and how to implement the migration into `go-go-goja/` using the repo’s preferred frameworks (Glazed and existing parsing utilities).
WhenToUse: Use this when implementing GOJA-01-INTEGRATE-JSDOCEX, onboarding a new engineer/intern to the JS doc extraction subsystem, or when extending the doc model to integrate with `pkg/jsparse` and other JS AST tooling.
---


# Design + implementation guide: integrate jsdocex into go-go-goja

## Goal

Move the functionality in `jsdocex/` into `go-go-goja/` in a way that:

- **Extraction becomes reusable**: the doc parsing/extraction lives in an exported Go package (not `internal/`), so it can be reused from CLIs, web servers, tests, and other tooling.
- **HTTP server keeps behavior**: the existing doc browser server + UI continue to work “as is”, with the same endpoints and JSON structure.
- **CLI uses Glazed**: new CLI entrypoints (`extract`, `serve`) use Glazed command definitions and follow the command wiring patterns already used in `go-go-goja`.
- **Future integration path exists**: the design leaves room to later integrate extracted docs with `go-go-goja/pkg/jsparse` (tree-sitter + goja AST analysis) without a major rewrite.

This guide is written for an intern who has never touched this repo before.

## Context

`jsdocex/` is a small Go module in this workspace that:

1) parses JavaScript source using tree-sitter,  
2) extracts documentation metadata from sentinel patterns embedded in JS code,  
3) stores that data in an in-memory index (`DocStore`),  
4) serves a web UI + JSON API + SSE reload events.

The goal of this ticket is to “pull” that system into `go-go-goja/`:

- make it a proper reusable package under `go-go-goja/pkg/`,
- keep the server behavior and HTTP API stable,
- expose the functionality via Glazed commands.

## Quick Reference

### What exists today (evidence)

**Extractor + models (jsdocex)**

- `jsdocex/internal/extractor/extractor.go`:
  - `ParseFile(path)` parses one JS file and extracts a `*model.FileDoc`.
  - It recognizes sentinel calls `__package__`, `__doc__`, `__example__`, and tagged templates `doc\`...\``.
  - It uses a heuristic “JS object literal → JSON-ish string” converter before `json.Unmarshal`.
- `jsdocex/internal/model/model.go`:
  - defines `Package`, `SymbolDoc`, `Example`, `FileDoc`, `DocStore`.
  - `DocStore.AddFile` updates indexes and removes prior docs for the same `FilePath`.

**Server (jsdocex)**

- `jsdocex/internal/server/server.go`:
  - exposes stable routes:
    - `GET /api/store`
    - `GET /api/package/{name}`
    - `GET /api/symbol/{name}` (includes `examples: [...]`)
    - `GET /api/example/{id}`
    - `GET /api/search?q=...`
    - `GET /events` (SSE; sends `reload`)
    - `GET /` (embedded SPA)

**Go-go-goja parsing + Glazed patterns**

- `go-go-goja/pkg/jsparse/treesitter.go`:
  - already uses tree-sitter in this repo, but via a different binding than `jsdocex/`.
- `go-go-goja/cmd/goja-perf/main.go` and `go-go-goja/cmd/goja-perf/serve_command.go`:
  - show how Glazed commands are built into Cobra commands in this repo.

### Stable external behavior to preserve

#### JS sentinel contract (what JS authors write)

`__package__({ ... })`

- Declares “package” metadata for the file/module.

`__doc__("name", { ... })` or `__doc__({ name: "...", ... })`

- Declares symbol docs for a named symbol.

`__example__({ ... })`

- Declares example metadata.
- **Note:** current extractor does *not* populate `Example.Body`, even though the model includes it.

`doc\`...\``

- Tagged template literal that contains:
  - a frontmatter block delimited by `---`,
  - followed by Markdown prose.
- Frontmatter keys currently used by samples:
  - `symbol: <name>` attaches prose to a symbol
  - `package: <pkg/name>` attaches prose to a package

Concrete fixtures:
- `jsdocex/samples/01-math.js`
- `jsdocex/samples/02-easing.js`
- `jsdocex/samples/03-vector2.js`

#### HTTP API contract (what UI/client calls)

The migration should keep endpoints and response JSON structure stable (same fields and names).

### Proposed target layout (go-go-goja)

Recommended package split:

```text
go-go-goja/
  pkg/
    jsdoc/
      model/    # exported model + store/index
      extract/  # source -> *model.FileDoc
      watch/    # fsnotify watcher (+ debounce)
      server/   # HTTP handlers + UI + SSE
  cmd/
    goja-jsdoc/ # new glazed-powered binary
      main.go
      extract_command.go
      serve_command.go
```

High-level dependency direction:

```text
cmd/goja-jsdoc
  ├─ pkg/jsdoc/extract  ──> pkg/jsdoc/model
  └─ pkg/jsdoc/server   ──> pkg/jsdoc/model
                          └─ pkg/jsdoc/watch
```

### Tree-sitter binding choice (important)

`go-go-goja` already depends on:

- `github.com/tree-sitter/go-tree-sitter`
- `github.com/tree-sitter/tree-sitter-javascript`

`jsdocex` currently depends on:

- `github.com/smacker/go-tree-sitter`

For the migration, **prefer the `go-go-goja` binding** to avoid carrying two tree-sitter ecosystems long-term.

That means the extractor should be rewritten to the `github.com/tree-sitter/go-tree-sitter` API (not copied verbatim).

#### Porting notes: smacker → go-tree-sitter (practical mapping)

This is the “translation table” an intern will need when rewriting `jsdocex/internal/extractor/extractor.go`:

- **Node kind/type**
  - smacker: `node.Type() string`
  - go-tree-sitter: `node.Kind() string`
- **Child count / child access**
  - smacker: `node.ChildCount() uint32` + `node.Child(i int)`
  - go-tree-sitter: `node.ChildCount() uint` + `node.Child(i uint)`
- **Field access in call expressions**
  - smacker: `node.ChildByFieldName("function")`
  - go-tree-sitter: `node.ChildByFieldName("function")` (same concept; slightly different types)
- **Source slicing**
  - both bindings expose `StartByte()` / `EndByte()`; in go-tree-sitter they are `uint`, so cast to `int` carefully.
- **Row/column positions**
  - both provide `StartPoint/StartPosition` with `.Row` (0-based) and `.Column` (0-based).
  - `jsdocex` currently stores `Line` as `int(row)+1` (1-based for humans).

Minimal parse skeleton using `github.com/tree-sitter/go-tree-sitter`:

```go
import (
	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_javascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
)

func parseJS(src []byte) (*tree_sitter.Tree, *tree_sitter.Parser, *tree_sitter.Language, error) {
	parser := tree_sitter.NewParser()
	lang := tree_sitter.NewLanguage(tree_sitter_javascript.Language())
	if err := parser.SetLanguage(lang); err != nil {
		parser.Close()
		return nil, nil, nil, err
	}
	tree := parser.Parse(src, nil)
	return tree, parser, lang, nil
}

func nodeText(src []byte, n *tree_sitter.Node) string {
	if n == nil {
		return ""
	}
	start := int(n.StartByte())
	end := int(n.EndByte())
	if start < 0 || end < start || end > len(src) {
		return ""
	}
	return string(src[start:end])
}
```

## Architecture and data flow (explained)

### Data flow diagram

```text
 JS source files on disk
        |
        v
  [Extractor]
    - parse JS with tree-sitter
    - find sentinel calls and doc`...`
    - build FileDoc
        |
        v
  [DocStore]
    - aggregate FileDoc objects
    - build indexes: by symbol, package, example, concept
        |
        v
  [HTTP server]
    - serve UI HTML
    - serve JSON API from DocStore
    - SSE "reload" events on file change
```

### Conceptual boundaries (what goes where)

- `pkg/jsdoc/model`: “dumb” data (structs), plus store/index update logic.
- `pkg/jsdoc/extract`: parsing and extraction (turn bytes into those structs).
- `pkg/jsdoc/server`: adapters that serve docs to humans and clients (HTTP + UI + SSE).
- `pkg/jsdoc/watch`: OS integration (fsnotify), no parsing logic.
- `cmd/goja-jsdoc`: user-facing CLI, mostly wiring and options.

Keeping these boundaries makes the system testable and reusable.

## Detailed “what to build” design

### `pkg/jsdoc/model`

Port (nearly 1:1) from `jsdocex/internal/model/model.go`, preserving JSON tags and field names.

Suggested files:

- `go-go-goja/pkg/jsdoc/model/model.go`: structs (`Package`, `SymbolDoc`, `Example`, `FileDoc`).
- `go-go-goja/pkg/jsdoc/model/store.go`: store + indexing logic (`DocStore`, `NewDocStore`, `AddFile`).

Intern tip: **do not rename JSON fields** unless you are willing to update the UI and any clients.

### `pkg/jsdoc/extract`

Public API (suggested):

```go
// ParseFile reads a file, parses it, and returns docs.
func ParseFile(path string) (*model.FileDoc, error)

// ParseSource parses already-loaded bytes (useful for tests).
func ParseSource(path string, src []byte) (*model.FileDoc, error)

// ParseDir parses a directory of .js files (parity note: current jsdocex ParseDir is non-recursive).
func ParseDir(dir string) ([]*model.FileDoc, error)
```

Extraction rules to preserve:

- recognize sentinel calls:
  - `__package__( {object literal} )`
  - `__doc__( "name", {object literal} )` or `__doc__( {object literal with name} )`
  - `__example__( {object literal} )`
- recognize `doc\`...\`` and attach prose:
  - if frontmatter includes `symbol: X`: attach to the symbol named `X`
  - if frontmatter includes `package: P`: attach to the package
  - if neither: attach to the last symbol (only if it doesn’t already have prose)

Implementation sketch (language-agnostic pseudocode):

```text
function parseSource(path, src):
  tree = treesitterParse(src)
  fd = new FileDoc(file_path = path)

  for each top-level node in tree.root.children:
    processNode(node, fd)

  return fd

function processNode(node, fd):
  if node.kind == "expression_statement":
     processNode(node.child(0), fd)
     return

  if node.kind == "call_expression":
     fnName = text(childByField(node, "function"))
     if fnName == "__package__": fd.package = parsePackage(node)
     if fnName == "__doc__":     fd.symbols += parseSymbolDoc(node)
     if fnName == "__example__": fd.examples += parseExample(node)
     if fnName == "doc":         attachDocTemplate(node, fd)
     return

  if node.kind in {class_declaration, export_statement, variable_declaration, ...}:
     for child in node.children:
        processNode(child, fd)
```

#### Object-literal parsing: parity first, then harden

Current jsdocex approach:
- converts JS object literal text to “JSON-ish” by heuristics (quotes keys, converts quotes, strips comments, trailing commas),
- then uses `encoding/json` to unmarshal into structs.

Migration plan:

1) for parity: keep the heuristic approach (ported),
2) add tests around it using the sample files,
3) after parity: consider replacing with a real parse (goja AST) if needed.

#### Frontmatter parsing: parity first, then YAML

Current jsdocex approach:
- splits frontmatter vs prose using `---` lines,
- parses only `key: value` lines.

After migration (optional improvement):
- use `gopkg.in/yaml.v3` to parse full YAML frontmatter,
- or use `github.com/adrg/frontmatter` (already in `go-go-goja` dependency graph).

### `pkg/jsdoc/watch`

Port from `jsdocex/internal/watcher/watcher.go`.

Two small “go-go-goja style” improvements you may choose:

1) accept `context.Context` in `Start(ctx)` rather than `Stop()` + internal `done` channel,  
2) surface events as a typed enum instead of a string (`create|write|remove|rename`).

Do not change behavior during parity work:
- watch subdirs,
- debounce 150ms,
- filter `.js`.

### `pkg/jsdoc/server`

Port from `jsdocex/internal/server/server.go`, keeping:

- the same routes,
- SSE behavior,
- UI HTML (copy `jsdocex/internal/server/ui.go` as-is),
- store update semantics on change:
  - delete-on-remove/rename via an “empty FileDoc” update,
  - reparse on write/create,
  - broadcast `reload`.

Refactor recommendation:

Expose a handler rather than hard-wiring `ListenAndServe`:

```go
type Server struct { ... }
func (s *Server) Handler() http.Handler
func (s *Server) Run(ctx context.Context) error
```

That makes it easier to:
- run from a Glazed command,
- test with `httptest`,
- embed in future tools.

### CLI: `cmd/goja-jsdoc` (Glazed commands)

Use the same Glazed+Cobra wiring style as `go-go-goja/cmd/goja-perf`.

Suggested commands:

1) `extract`
   - Glazed output (rows) for symbols/examples/package, plus an option for raw JSON.

2) `serve`
   - a side-effecting command, implemented as a Glazed command (flags) but run like `cmds.BareCommand`.

Suggested flags:

- `extract`: `--file <path>`
- `serve`: `--dir <path> --host <host> --port <port>`

Intern tip: start from `go-go-goja/cmd/goja-perf/serve_command.go` because it already shows a “Glazed for flags, but run a server” pattern.

## API contract details (for parity testing)

Keep these routes and shapes stable:

- `GET /api/store` returns the full store, including the indexes (`by_symbol`, `by_package`, ...).
- `GET /api/symbol/{name}` returns `SymbolDoc` plus `examples: []Example`.
- `GET /api/search?q=...` returns `{symbols, examples, packages}` with arrays (possibly null/empty).
- `GET /events` streams SSE and sends `reload` on changes.

If you change field names in the Go structs, the UI will break. Keep JSON tags stable.

## Implementation plan (intern checklist)

### Phase 1: “Move without changing behavior”

1) Create package skeletons in `go-go-goja/pkg/jsdoc/*`.
2) Port model structs + store.
3) Port extractor (rewrite to the `go-tree-sitter` binding).
4) Port watcher and server.
5) Add a new binary `goja-jsdoc` with Glazed commands.

### Phase 2: “Parity validation”

1) Compare extractor output:
   - run old `jsdocex extract` on sample files,
   - run new `goja-jsdoc extract`,
   - diff the JSON.
2) Compare server output:
   - run old `jsdocex serve`,
   - run new `goja-jsdoc serve`,
   - hit endpoints and confirm UI works.

### Phase 3: “Cutover”

1) Remove `./jsdocex` from `go.work` once parity is achieved.
2) Delete or archive `jsdocex/` directory (do not keep compatibility wrappers unless asked).

## Known gaps / explicit decisions to make

1) `Example.Body` is currently always empty.
   - Decide whether to keep it empty for parity, or implement extraction.
2) `ParseDir` currently parses only the direct directory (non-recursive).
   - Decide whether to preserve that behavior or intentionally change it.
3) Object literal parsing and YAML frontmatter parsing are “best-effort”.
   - For parity, keep behavior.
   - For robustness, add tests and consider upgrading parsers after cutover.

## Usage Examples

### Today (existing jsdocex)

```bash
go run ./jsdocex/cmd/jsdocex extract ./jsdocex/samples/01-math.js
go run ./jsdocex/cmd/jsdocex serve ./jsdocex/samples 8080
```

### After migration (expected go-go-goja CLI)

```bash
go run ./go-go-goja/cmd/goja-jsdoc extract --file ./jsdocex/samples/01-math.js --output json
go run ./go-go-goja/cmd/goja-jsdoc serve --dir ./jsdocex/samples --host 127.0.0.1 --port 8080
```

## Related

- Ticket index: `../index.md`
- Tasks: `../tasks.md`
- Diary: `02-diary.md`
