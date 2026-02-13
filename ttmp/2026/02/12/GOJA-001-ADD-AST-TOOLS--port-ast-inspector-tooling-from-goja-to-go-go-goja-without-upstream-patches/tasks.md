# Tasks

## TODO

- [x] Create reusable `pkg/jsparse` package in `go-go-goja`
- [x] Move reusable analysis files into `pkg/jsparse` (`index`, `noderecord`, `resolve`, `treesitter`, `completion`)
- [x] Move reusable tests into `pkg/jsparse` and ensure they are UI-independent
- [x] Create command-local inspector app package under `cmd/inspector/app` for TUI/editor code (`model`, `drawer`)
- [x] Port `cmd/goja-inspector/main.go` into `go-go-goja/cmd/inspector/main.go`
- [x] Rewrite command and app imports to consume `pkg/jsparse` + `cmd/inspector/app`
- [x] Add a reusable analysis facade API in `pkg/jsparse` for non-inspector tooling use cases
- [x] Add tree-sitter dependencies to `go-go-goja/go.mod` and run `go mod tidy`
- [x] Run focused tests: `go test ./pkg/jsparse -count=1`
- [x] Run focused tests: `go test ./cmd/inspector/... -count=1`
- [x] Run focused build: `go build ./cmd/inspector`
- [x] Validate key UX features manually (source/tree sync, go-to-def, highlight usages, drawer completion)
- [x] Decide whether to keep standalone command only or integrate with `cmd/repl` as follow-up
- [x] Ensure CI/test strategy avoids unrelated bun-demo false negatives during inspector migration

## DONE

- [x] Created ticket workspace `GOJA-001-ADD-AST-TOOLS`
- [x] Created `Diary` and `Porting Analysis` docs
- [x] Completed scope inventory of inspector files/commits/dependencies in `goja`
- [x] Validated inspector build/tests baseline in `goja`
- [x] Validated external portability with `GOWORK=off` smoke module
- [x] Authored implementation-ready migration analysis and phased plan
- [x] Revised plan to require split architecture (`pkg/jsparse` reusable framework + command-local inspector example app)
- [x] Cleaned vocabulary/doctor warnings for this ticket
- [x] Uploaded bundled ticket analysis to reMarkable and verified remote presence
- [x] Add glazed help user-guide entry for cmd/inspector as example consumer of pkg/jsparse
- [x] Write detailed reference documentation for pkg/jsparse in go-go-goja/pkg/doc
- [x] Validate new help entries are discoverable via repl help slugs and document usage
