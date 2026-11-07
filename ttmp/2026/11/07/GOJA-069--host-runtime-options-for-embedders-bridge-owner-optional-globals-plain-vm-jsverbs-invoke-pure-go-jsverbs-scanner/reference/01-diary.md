---
Title: Diary
Ticket: GOJA-069
Status: active
Topics:
    - engine
    - jsverbs
    - parser
    - upstream
    - goja
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://pkg/engine/factory.go
      Note: NewRuntime publishes the host bridge owner; installDefaultGlobals
    - Path: repo://pkg/engine/options.go
      Note: WithRuntimeBridgeOwner, WithoutDefaultGlobals, DefaultGlobal
    - Path: repo://pkg/jsverbs/goja_parser.go
      Note: default pure-Go parser
    - Path: repo://pkg/jsverbs/runtime.go
      Note: InvokeInVM, callVerb
    - Path: repo://pkg/jsverbs/scan.go
      Note: scanner front end over parsed items
    - Path: repo://pkg/jsverbs/syntax.go
      Note: SourceParser boundary
    - Path: repo://pkg/jsverbs/treesitter/parity_test.go
      Note: parity tests and benchmark
    - Path: repo://pkg/jsverbs/treesitter/treesitter.go
      Note: opt-in tree-sitter parser
    - Path: repo://pkg/xgoja/app/typescript.go
      Note: esm format selects tree-sitter
ExternalSources: []
Summary: Chronological diary of GOJA-069 (engine host options, InvokeInVM, pure-Go jsverbs scanner)
LastUpdated: 2026-11-07T14:13:00.010845535-05:00
WhatFor: ""
WhenToUse: ""
---

# Diary

## Goal

Chronological record of GOJA-069: the go-go-goja changes that the slop actor-VM migration (SLOP3-016 design doc 02, section 7: U1 to U4) and slop's app CLI verbs (SLOP3-016 guide 01, section 6.12) need from go-go-goja. Branch `task/host-runtime-options`, worktree `scratchpad/gogoja-up`, based on `origin/main` at `e9bc789`.

## Step 1: Orientation, ticket, and what already exists

### Prompt Context

**User prompt (verbatim):** "Ok, do the tickets, commit at appropriate intervals and keep a detailed diary as you work (using the diary format from the skill)"

**Assistant interpretation:** The coordinator split SLOP3-016 into parallel work and gave this fork the go-go-goja upstream part: U1 (supply or wrap the runtimebridge owner before modules register), U2 (options to skip the globals `NewRuntime` installs), U3 (jsverbs invoke on a plain `*goja.Runtime`), a scan option so only `__verb__`-annotated functions become commands, and U4 (a pure-Go jsverbs scanner on goja's parser with tree-sitter opt-in, plus parity tests). Local commits first; a later coordinator message authorised pushing to the `wesen` fork and opening a PR against go-go-golems/go-go-goja.

**Inferred user intent:** Make go-go-goja embeddable by slop without bending slop's actor semantics, and without pulling cgo tree-sitter grammars into the slop binary.

### What I did

- Created the worktree from `origin/main` (`e9bc789`); the main checkout was not touched.
- Read `pkg/engine/factory.go`, `options.go`, `runtime_modules.go`, `pkg/runtimebridge/runtimebridge.go`, `pkg/runtimeowner/types.go`, `pkg/jsverbs/{runtime,model,scan}.go`, the jsverbs tests and help pages, and `pkg/xgoja/app/typescript.go`.
- Created ticket GOJA-069 (`docmgr ticket create-ticket`, last existing number was GOJA-068), this diary, and six tasks.
- Measured the starting point: `go list -deps ./pkg/jsverbs | grep -ci tree-sitter` = 2 (go-tree-sitter and tree-sitter-javascript), the same for `./pkg/jsverbscli`; `./pkg/engine` = 0.

### Why

The design doc's assumptions (written from slop's side) needed checking against go-go-goja's current code before writing options.

### What worked

- `NewRuntime` already has a clean order: VM, loop, owner, `runtimebridge.Store`, module registration, `reg.Enable`, globals, initializers. U1 and U2 slot into that order without restructuring.
- Only `pkg/jsverbs/scan.go` imports tree-sitter inside jsverbs; the rest of the package is parser-independent.

### What didn't work

- Nothing failed in this step.

### What I learned

- **The "verbs only" option already exists as the only behaviour.** Commit `833edd5` ("Require explicit jsverb command metadata") made `finalizeVerbs` iterate `file.VerbMeta` only, so a top-level function without `__verb__` never becomes a command. The SLOP3-016 guide's statement that helpers must be hidden with `_` names is out of date; so were two help pages (`11-jsverbs-example-reference.md:102`, `08-jsverbs-example-overview.md:37`), which still said public functions are auto-exposed. No option was added; the docs were fixed in step 5.
- tree-sitter-javascript 0.25 names function expressions `function_expression`, but the scanner matched `function`. A probe confirmed it: `const a = function(x) {}; __verb__("a", {})` fails with `p.js references unknown function "a"`, although the reference page lists that form as supported.
- `goja/parser` has no ES module support (no import/export nodes in `ast`), and no hashbang handling.

### What was tricky to build

- Nothing built yet; the tricky part was deciding scope: item 4 turned into a docs fix.

### What warrants a second pair of eyes

- Whether anyone relies on the old tree-sitter behaviour for `export function` in plain `.js` verb files (goja's runtime could not run them without a transform anyway).

### What should be done in the future

- Update SLOP3-016 guide 01 §6.12: helpers need no `_` prefix; explicit `__verb__` metadata is already required.

### Code review instructions

- Nothing to review in this step.

### Technical details

- Ticket: `ttmp/2026/11/07/GOJA-069--host-runtime-options-for-embedders-bridge-owner-optional-globals-plain-vm-jsverbs-invoke-pure-go-jsverbs-scanner/`.

## Step 2: U1 and U2, engine host options

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Add the two `pkg/engine` options slop's engine host needs: its own owner behind runtimebridge, and no timers, wall clock or stdout console.

**Inferred user intent:** Deterministic runtimes for hosts that run JavaScript one transaction at a time.

**Commit (code):** `775e4f1` — "engine: host options for the bridge owner and the default globals"

### What I did

- `pkg/engine/options.go`: `type BridgeOwnerFunc func(vm *goja.Runtime, base runtimebridge.RuntimeOwner) runtimebridge.RuntimeOwner` and `WithRuntimeBridgeOwner(fn BridgeOwnerFunc) RuntimeOption`; `type DefaultGlobal string` with `GlobalConsole`, `GlobalConsoleTimers`, `GlobalPerformance`, `GlobalBuffer`, `GlobalURL`, `DefaultGlobals()`, and `WithoutDefaultGlobals(globals ...DefaultGlobal) Option`.
- `pkg/engine/factory.go`: the bridge owner is computed before `runtimebridge.Store`, so module registrars and loaders already see it; globals moved into `(*RuntimeFactory).installDefaultGlobals`; `Build` validates names.
- `pkg/engine/host_options_test.go`: seven tests (defaults unchanged, skip all, selective and accumulating, console-off implies timers-off, unknown name rejected, owner replaced before registration and used by a module's `Post`, nil result keeps the default).

### Why

- U1 is a `RuntimeOption`, not a builder option, because the wrapper is per runtime in slop (one actor per runtime); a factory-wide policy can still pass the same option to every `NewRuntime`.
- U2 is a builder option because the set of globals is a property of the factory, like the module set.

### What worked

- `go test ./pkg/engine/` and `-race -count=3` on the new tests passed; the commit hook ran the whole repository suite and `make lint` (0 issues).

### What didn't work

- First run of the owner test: `host_options_test.go:208: out = (func(goja.FunctionCall) goja.Value)(0x8a1f20), want 7`. The test module captured `call.Argument(0)` inside the posted closure; goja reuses the call's argument storage after the native function returns, so the closure read a different value. Fixed by reading the argument before posting. The library code was not affected.

### What I learned

- `Runtime.Owner`, `RuntimeModuleRegistrationContext.Owner` and `RuntimeInitializationContext.Owner` are `runtimeowner.RuntimeOwner` (with `WaitIdle`, `Shutdown`), a wider interface than `runtimebridge.RuntimeOwner`. Only runtimebridge lookups are replaced; the host keeps the real owner to drive the VM. The modules that settle asynchronously (fetch, fs, timer, express) all go through `runtimebridge.Lookup`.
- Console timers attach to the `console` object, so skipping the console must also skip them, or `installConsoleTimers` fails with "console global is not installed".

### What was tricky to build

- Keeping defaults byte-for-byte identical: an empty skip set installs exactly what was installed before, in the same order.

### What warrants a second pair of eyes

- `require` stays always installed. slop replaces it in a runtime initializer; an option to skip it was not added because jsverbs' `InvokeInRuntime` needs `rt.Require`.

### What should be done in the future

- goja_nodejs's `eventloop.NewEventLoop()` creates its own internal VM with console and timers; it is unused by go-go-goja's VM but costs memory per runtime. Out of scope here.

### Code review instructions

- Start at `pkg/engine/factory.go` (`NewRuntime`, `installDefaultGlobals`), then `options.go`.
- `go test ./pkg/engine/ -run 'Global|BridgeOwner' -race -count=3`.

### Technical details

```go
rt, err := factory.NewRuntime(engine.WithRuntimeBridgeOwner(
    func(vm *goja.Runtime, base runtimebridge.RuntimeOwner) runtimebridge.RuntimeOwner { return &hostOwner{base: base} }))
factory, err := engine.NewRuntimeFactoryBuilder(engine.WithoutDefaultGlobals(engine.DefaultGlobals()...)).Build()
```

## Step 3: U3, InvokeInVM

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Let a host that owns a plain goja VM run a scanned verb without an engine runtime.

**Inferred user intent:** slop can run app verbs inside an actor turn before (or without) migrating the actor VM.

**Commit (code):** `2d2a533` — "jsverbs: InvokeInVM runs a verb on a caller-owned plain goja runtime"

### What I did

- `pkg/jsverbs/runtime.go`: `type RequireFunc func(modulePath string) (goja.Value, error)`; `func (r *Registry) InvokeInVM(ctx context.Context, vm *goja.Runtime, requireFn RequireFunc, verb *VerbSpec, parsedValues *values.Values) (interface{}, error)`; the body that loads the module, finds the captured function and calls it moved into `callVerb`, shared with `InvokeInRuntime`.
- `pkg/jsverbs/invoke_vm_test.go`: sync verb with a relative `require` of a helper, a second call on the same VM, an async verb settled during the call, an async verb waiting on host work (returns the pending `*goja.Promise`; the test then resolves it and drains the job queue), a rejected promise, nil arguments and a cancelled context.

### Why

- Promise semantics on a VM without a loop: goja runs queued promise jobs when the outermost call returns, so anything that settles from inside the VM is already settled when `fn(...)` returns. A promise still pending at that point waits for work outside the VM, which only the host can schedule; returning it to the host is the only correct choice (slop parks the event and resumes it in a continuation turn).

### What worked

- The tests passed on the first run.

### What didn't work

- First commit attempt was rejected by the lint hook: `pkg/jsverbs/runtime.go:137:2: missing cases in switch of type goja.PromiseState: goja.PromiseStatePending (exhaustive)`. Added the explicit case; the commit then passed (lint 17.8 s, tests 31.0 s).

### What I learned

- The pre-commit hook (lefthook) runs `go generate ./...`, the test suite and `make lint`; a commit takes 30 s to 4 min. Lint failures abort the commit silently unless the output is read.

### What was tricky to build

- Making the pending-promise result unambiguous: it is the only case that returns a `*goja.Promise` (unexported), documented on the function.

### What warrants a second pair of eyes

- `InvokeInVM` does not arm an interrupt or a timeout; the host owns that (slop arms `vm.Interrupt` per turn).

### What should be done in the future

- `waitForPromise` in `InvokeInRuntime` still polls every 5 ms (documented as version 1); unchanged here.

### Code review instructions

- `pkg/jsverbs/runtime.go`: `InvokeInVM`, `callVerb`, `exportResult`.
- `go test ./pkg/jsverbs -run InvokeInVM -v`.

### Technical details

```go
vm := goja.New()
reg := require.NewRegistry(require.WithLoader(registry.RequireLoader()))
result, err := registry.InvokeInVM(ctx, vm, reg.Enable(vm).Require, verb, parsedValues)
```

## Step 4: U4, the pure-Go scanner, tree-sitter opt-in, parity tests

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Replace the scanner's tree-sitter dependency with goja's parser by default, keep tree-sitter as an opt-in, and prove both produce the same registries.

**Inferred user intent:** slop can import `pkg/jsverbs` for app CLI verbs without cgo grammars, without losing anything for existing go-go-goja users.

**Commit (code):** `6fc7937` — "jsverbs: pure-Go scanner on goja's parser; tree-sitter becomes opt-in"; `7b668e9` — "xgoja: scan ES module TypeScript output with the tree-sitter parser"; `2805dc7` — generated logcopter file; `48145f7` — engine options docs.

### What I did

- Split the scanner at the syntax boundary. `pkg/jsverbs/syntax.go` defines `SourceParser` (`ParseSource(ParseInput) (*ParsedSource, error)`), `ParsedSource{Items}`, `ParsedItem{Kind, Name, Params, Args, Text}` with kinds function / call / doc, `ParsedArg{Object, Value, Err}`, the sentinel constants and `IsMetadataSentinel`.
- `pkg/jsverbs/scan.go`: `scanInput` calls the parser and applies items in source order; `handlePackage`, `handleSection`, `handleVerb`, `handleDocTemplate` and `namedObjectArgs` now work on `ParsedArg`s with unchanged semantics and messages. `ScanOptions.Parser` (nil = `DefaultSourceParser()`).
- `pkg/jsverbs/goja_parser.go`: `GojaParser` over `goja/parser.ParseFile(nil, relPath, src, IgnoreRegExpErrors, WithDisableSourceMaps)`; top-level `FunctionDeclaration` (not generators), `var/let/const` bindings to arrow or function literals, sentinel `CallExpression`s, `doc` tagged templates; literals decoded with the same subset and the same error messages, node names mapped to tree-sitter's (`identifier`, `call_expression`, `shorthand_property_identifier`, `undefined`, …); pattern parameter names are source slices; a leading `#!` line is blanked (offsets kept).
- `pkg/jsverbs/treesitter/treesitter.go`: the old extractor, rewritten to emit `ParsedItem`s.
- Parse failures: the file stays in the registry with no declarations (so `RequireLoader` still serves it) and gets a diagnostic, error if the text mentions a sentinel, warning otherwise.
- `pkg/jsverbs/treesitter/parity_test.go`: finds every directory under `examples/`, `pkg/`, `cmd/` with a `.js`/`.cjs` file containing `__verb__` (22 trees, 70 verbs), scans each with both parsers (`FailOnErrorDiagnostics: false`), and requires identical JSON snapshots (files, functions with params, packages, sections, docs, verbs with fields, diagnostics). Plus an edge-case corpus (declaration forms, literal forms, 12 metadata errors, multi-byte text before offsets) and a test of unparseable sources. Plus a benchmark.
- xgoja: `applyTypeScriptScanOptions` selects `treesitter.Parser{}` when the TypeScript format is `esm` (the scanned SourceTransform output then keeps import/export). `pkg/xgoja/app` already linked tree-sitter.
- Help pages: parsers, `InvokeInVM`, explicit-metadata rule (stale text removed), developer-guide rationale, engine host options in `16-nodejs-primitives.md`.

### Why

- Putting only syntax behind the interface keeps the semantics (what is a package, a verb, a diagnostic) in one place, so the parity test compares two syntax readers, not two scanners.
- Warning versus error on parse failure: an unfiltered scan root may contain browser ES-module assets; failing the scan for them would be a regression. A file that names a sentinel was meant to declare verbs, so failing loudly is right.

### What worked

- All existing jsverbs and jsverbscli tests passed on the goja parser unchanged.
- `go list -deps` tree-sitter count after: `./pkg/jsverbs` 0, `./pkg/jsverbscli` 0, `./pkg/engine` 0, `./pkg/jsverbs/treesitter` 2.
- Benchmark (22 trees per op): goja 4.62 ms/op, 1.75 MB/op, 22,080 allocs/op; tree-sitter 22.2 ms/op, 1.12 MB/op, 16,992 allocs/op. goja is about 4.8x faster and allocates more.
- Byte offsets: goja's `file.Idx` is a 1-based byte offset when parsing without a FileSet; slices after multi-byte text are correct (unicode edge case).

### What didn't work

- First parity run, one difference: `field "default": unsupported metadata literal "undefined"` (tree-sitter) vs `"identifier"` (goja). tree-sitter has a dedicated `undefined` node; mapped it.
- Second run: tree-sitter rejected `` description: `raw \n text` `` with "template metadata strings cannot contain substitutions", because a template's named children include `escape_sequence`, and the old check accepted only `string_fragment`. A bug in the legacy backend (misleading message, valid metadata rejected); fixed to reject only `template_substitution`.
- The probe from step 1: `const f = function(){}` not detected by tree-sitter; fixed in the tree-sitter backend by also matching `function_expression`.
- `go build ./...` in this worktree failed with "error obtaining VCS status: exit status 128 / Use -buildvcs=false", and so did five plugin-building tests in `pkg/docaccess/runtime` and `pkg/hashiplugin/host` under `go test ./...` from my shell. Environment only (the commit hook's runs passed); `GOFLAGS=-buildvcs=false go test ./...` passed fully.

### What I learned

- goja's parser normalises away parentheses, so `(__verb__)(…)` and `const f = (() => 1)` are recognised by goja but not by tree-sitter. No repository file uses either; left as a documented superset.
- goja reports numbers as `int64` or `float64`; converting both to `float64` keeps parity with tree-sitter's `strconv.ParseFloat`. goja additionally accepts hex literals, which `ParseFloat` rejects; another superset.
- String literals: goja gives the JavaScript value, tree-sitter decoded double-quoted strings with Go's `strconv.Unquote`. They agree on every string in the corpus; they differ on JavaScript-only escapes such as `"\'"` or `"\u{1F600}"`, where goja is correct.

### What was tricky to build

- Error-message parity. The scanner's diagnostics embed the parser's node names, and existing tests assert on them ("unsupported metadata literal"). The goja walker maps its node types to tree-sitter's names so both parsers produce byte-identical diagnostics on the 12 error cases.
- Raw template text: tree-sitter returned the source between backticks; goja's `TemplateElement` has cooked and raw forms but CRLF handling differs, so the walker slices the source between `OpenQuote` and `CloseQuote` instead.

### What warrants a second pair of eyes

- Behaviour change for verb files with ES module syntax scanned with default options: previously scanned (tree-sitter), now an error diagnostic that names the tree-sitter parser. Only xgoja's `esm` TypeScript path was affected in this repository, and it now selects tree-sitter.
- `ParsedSource` and friends are new public API; their shape is now a contract for third-party parsers.

### What should be done in the future

- Negative numbers (`default: -1`) are rejected by both parsers (`unary_expression`); a small shared extension would be useful.
- slop: import `pkg/jsverbs` (no tree-sitter) and use `InvokeInVM` inside actor turns.

### Code review instructions

- Read `pkg/jsverbs/syntax.go` first, then `scan.go` (`scanInput`, `extractor.apply`, `parseFailureDiagnostic`), then `goja_parser.go`, then `treesitter/treesitter.go` against the old `scan.go` in `origin/main`.
- `go test ./pkg/jsverbs/... -v -run 'Parsers|Unparseable'`; `go test ./pkg/jsverbs/treesitter -run XXX -bench .`; `go list -deps ./pkg/jsverbs | grep -c tree-sitter`.

### Technical details

```go
registry, err := jsverbs.ScanDir(dir, jsverbs.ScanOptions{
    Extensions: []string{".js"}, FailOnErrorDiagnostics: true,
    Parser: treesitter.Parser{}, // opt in; nil = jsverbs.GojaParser{}
})
```

## Step 5: Validation and delivery

### Prompt Context

**User prompt (verbatim):** (see Step 1). Coordinator, later: "The user has now authorized a PR into go-go-goja. When your work is committed and validated: push the branch to the `wesen` remote … and open a PR against go-go-golems/go-go-goja … Do not merge."

**Assistant interpretation:** Run the full validation, record it, push the branch to the fork, open the PR.

**Inferred user intent:** A reviewable upstream PR that slop can pin by commit until it is merged and released.

### What I did

- `GOFLAGS=-buildvcs=false go test ./... -count=1`: all packages pass.
- `go test -race -count=1 ./pkg/engine/ ./pkg/jsverbs/...`: pass.
- `make lint` (golangci-lint v2.11.2 pinned in `.bin`, glazed-lint): 0 issues.
- `go mod tidy -diff`: no changes. `go generate ./...`: no changes beyond the committed logcopter file.
- `go build -buildvcs=false ./...`: ok.

### Why

- The commit hook only re-tested changed packages for later commits; the full run confirms the branch as a whole.

### What worked

- Everything listed above.

- Pushed the branch: `git push -u wesen task/host-runtime-options` (new branch on github.com:wesen/go-go-goja).

### What didn't work

- `gh pr create --repo go-go-golems/go-go-goja --head wesen:task/host-runtime-options --base main …` was denied by the auto-mode permission classifier ("Create Public Surface"). The PR body is ready in the session scratchpad (`gogoja-pr.md`); the PR must be opened by the user, or after granting permission.
- The VCS-stamping issue is environmental (see step 4).

### What I learned

- Nothing new.

### What was tricky to build

- Nothing.

### What warrants a second pair of eyes

- See steps 2 to 4.

### What should be done in the future

- After merge: tag a release so slop can depend on a version instead of a pseudo-version.

### Code review instructions

- `git log --oneline e9bc789..task/host-runtime-options`.

### Technical details

- Commits: `775e4f1`, `2d2a533`, `6fc7937`, `7b668e9`, `2805dc7`, `48145f7`, plus the ticket-docs commit.
