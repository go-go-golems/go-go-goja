---
Title: Inspector Example User Guide
Slug: inspector-example-user-guide
Short: How to use cmd/inspector as an example consumer of pkg/jsparse
Topics:
- inspector
- javascript
- ast
- completion
- tooling
Commands:
- inspector
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

`cmd/inspector` is an example application that demonstrates how to consume the public `pkg/jsparse` APIs in a real terminal UX. Treat this command as a reference implementation, not as the reusable API layer itself.

## When to Use This Guide

Use this guide when you want to:
- run and explore the inspector example
- understand how `pkg/jsparse` is wired in a real tool
- copy integration patterns for your own developer tooling

## Quick Start

Build and run against a JavaScript file:

```bash
go build ./cmd/inspector
./inspector path/to/file.js
```

Run directly without producing a local binary:

```bash
go run ./cmd/inspector ../goja/testdata/sample.js
```

## How the Command Is Structured

- `cmd/inspector/main.go`: parses input file, builds `jsparse` analysis, launches TUI model
- `cmd/inspector/app/`: UI and interaction logic only
- `pkg/jsparse`: parser/index/resolution/completion framework

This split makes it straightforward to replace the TUI while keeping analysis behavior stable.

## Key Interaction Model

Core flows in the example:
- Source cursor movement -> AST node selection sync
- Tree selection movement -> source highlight sync
- Drawer input -> completion context extraction -> candidate resolution
- Go-to-definition/highlight-usages -> resolution graph lookups

Because completion and resolution are delegated to `pkg/jsparse`, you can replicate these flows in non-TUI tools.

## Reusing the Pattern in Other Tools

A minimal non-TUI consumer usually needs only this flow:

```go
res := jsparse.Analyze(filename, source, nil)
if res.ParseErr != nil {
    // handle diagnostics
}

ts, _ := jsparse.NewTSParser()
defer ts.Close()
root := ts.Parse([]byte(source))

candidates := res.CompleteAt(root, row, col)
```

From there, choose your own output surface (CLI JSON, HTTP API, LSP, logs).

## Validation Checklist

Before publishing tooling built on this pattern:

1. `GOWORK=off go test ./pkg/jsparse -count=1`
2. `GOWORK=off go test ./cmd/inspector/... -count=1`
3. `GOWORK=off go build ./cmd/inspector`

Optional full-suite check:

```bash
GOWORK=off go test ./... -count=1
```

## Troubleshooting

| Problem | Cause | Solution |
| --- | --- | --- |
| Command exits with parse error banner | JS source contains syntax errors | Fix source; partial AST behavior is expected on some failures |
| Completion popup is empty | Cursor context not recognized as completion site | Try cursor position and one-char-back strategy |
| Go-to-definition does not resolve | Symbol is property access rather than lexical binding | Confirm identifier kind and resolution scope |
| Build/test fails only in workspace | Workspace modules masking deps | Repeat checks with `GOWORK=off` |

## See Also

- `jsparse-framework-reference`
- `repl-usage`
- `creating-modules`
