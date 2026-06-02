---
Title: Implementation diary
Ticket: XGOJA-JSVERBS-ROOT-001
Status: active
Topics:
  - xgoja
  - jsverbs
  - cli
  - buildspec
DocType: reference
Intent: Chronological implementation diary for root-mounted xgoja JavaScript verbs
---

# Diary

## Goal

Capture the design and implementation process for adding an xgoja buildspec option that mounts JavaScript verb packages directly under the generated root command.

## Step 1: Add root-mounted jsverbs support

The implementation adds an explicit root-mount option for xgoja JavaScript verbs. The default behavior stays unchanged: verbs are still mounted below the configured jsverbs command, usually `verbs`. When a buildspec sets `commands.jsverbs.mount: root`, xgoja scans and builds the same JavaScript verb commands but attaches them directly to the generated root command.

This is useful for self-contained helper binaries where the JavaScript verbs are the primary command surface. A binary can expose `goja-text markdown toc ...` instead of `goja-text verbs markdown toc ...` while still using the same jsverbs scanner, runtime profile, module initialization, and invocation machinery.

### Prompt Context

**User prompt (verbatim):** "ok, create a docmgr ticket in ./go-go-goja/ttmp (with docmgr --root ...) to add said option to register jsverbs to the root verb directly, and implement it, and use it in your example, that way we have a nice self contained helpful binary"

**Assistant interpretation:** Create a docmgr ticket under the go-go-goja `ttmp` root, implement a root-mounted jsverbs option in xgoja, and then use it in the goja-text generated binary spec.

**Inferred user intent:** Make xgoja capable of producing binaries whose JavaScript verbs feel like first-class CLI commands, without an extra `verbs` prefix.

**Commit (code):** pending — implementation not committed yet.

### What I did

- Created ticket `XGOJA-JSVERBS-ROOT-001` under `/home/manuel/workspaces/2026-06-02/goja-text/go-go-goja/ttmp` using `docmgr --root ./ttmp`.
- Wrote the design guide at `design-doc/01-root-mounted-javascript-verbs-design-and-implementation-guide.md`.
- Updated `pkg/xgoja/app/host.go` so `AttachVerbs` checks `commands.jsverbs.mount`.
- Added `commandMount` in `pkg/xgoja/app/root.go`, accepting `root`, `/`, and `.` as root-mount aliases.
- Added buildspec validation in `cmd/xgoja/internal/buildspec/validate.go` for `commands.jsverbs.mount`.
- Added tests in:
  - `pkg/xgoja/app/root_test.go`
  - `cmd/xgoja/internal/buildspec/validate_test.go`
- Updated xgoja docs:
  - `cmd/xgoja/doc/02-user-guide.md`
  - `cmd/xgoja/doc/03-tutorial-using-xgoja-yaml.md`
  - `cmd/xgoja/doc/06-buildspec-reference.md`
- Ran targeted tests successfully:
  - `go test ./cmd/xgoja ./cmd/xgoja/internal/buildspec ./pkg/xgoja/app -count=1`

### Why

The default `verbs` container is safe, but it is not ideal for binaries where bundled JavaScript commands are the main user-facing feature. Root mounting is an explicit opt-in that improves ergonomics without changing the default behavior or the scanner/runtime implementation.

### What worked

- The root mount can reuse existing `buildVerbCommands`; no changes were needed in the JS scanner or invocation path.
- Root-mounted embedded verbs execute correctly in the new runtime test.
- Buildspec validation catches unsupported mount strings.
- The docs now explain both the default container mode and root-mounted mode.

### What didn't work

- My first docmgr command used `docmgr --root .`, which created a misplaced ticket workspace under `go-go-goja/2026/...`. I corrected this by creating the ticket with `docmgr --root ./ttmp` and removing the misplaced workspace.

### What I learned

- `CommandSpec.Mount` already existed, so this feature could be implemented as semantics for the existing field rather than a new buildspec key.
- The root-mount behavior belongs in `Host.AttachVerbs`, not in the scanner. The scanner should continue producing Glazed command descriptions independent of where they are attached.

### What was tricky to build

- The sharp edge is command collision. Root-mounted JavaScript packages can collide with built-in commands such as `help`, `eval`, `run`, `repl`, and `modules`. The feature therefore remains explicit and the docs recommend default container mode when collision avoidance matters.

### What warrants a second pair of eyes

- Whether root-mounted jsverbs should also expose a root-level `sources` helper command or whether omitting it is the right tradeoff.
- Whether additional reserved-name validation should warn when a root-mounted verb package collides with built-in commands.

### What should be done in the future

- Use `commands.jsverbs.mount: root` in `goja-text/cmd/goja-text/xgoja.yaml` and validate the resulting self-contained binary.
- Consider command-collision diagnostics if root mounting becomes common.

### Code review instructions

- Start with `pkg/xgoja/app/host.go`, especially `AttachVerbs`.
- Review `pkg/xgoja/app/root.go` for `commandMount`.
- Review tests in `pkg/xgoja/app/root_test.go` and `cmd/xgoja/internal/buildspec/validate_test.go`.
- Validate with:
  - `go test ./cmd/xgoja ./cmd/xgoja/internal/buildspec ./pkg/xgoja/app -count=1`

### Technical details

Buildspec usage:

```yaml
commands:
  jsverbs:
    enabled: true
    runtime: main
    mount: root
```

Accepted aliases:

```yaml
mount: root
mount: /
mount: .
```
