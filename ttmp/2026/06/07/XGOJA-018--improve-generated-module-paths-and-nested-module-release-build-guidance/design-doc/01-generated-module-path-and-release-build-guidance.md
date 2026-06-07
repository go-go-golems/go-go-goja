---
Title: Generated Module Path and Release Build Guidance
Ticket: XGOJA-018
Status: active
Topics:
    - xgoja
    - go
    - release
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../goja-bleve/.goreleaser.yaml
      Note: GoReleaser config showing nested module build pattern
    - Path: ../../../../../../../goja-bleve/cmd/goja-bleve/xgoja-vectors.yaml
      Note: Concrete example spec that triggered this issue
    - Path: cmd/xgoja/cmd_build.go
      Note: Owns xgoja build command flow
    - Path: cmd/xgoja/doc/02-user-guide.md
      Note: xgoja.yaml user documentation mentioning go.module default
    - Path: cmd/xgoja/doc/06-buildspec-reference.md
      Note: Quick buildspec reference
    - Path: cmd/xgoja/internal/buildexec/buildexec.go
      Note: Runs go mod tidy and go build without user-facing output context
    - Path: cmd/xgoja/internal/buildspec/build_spec.go
      Note: GoSpec struct with Module field
    - Path: cmd/xgoja/internal/buildspec/load.go
      Note: Default module path in applyDefaults()
    - Path: cmd/xgoja/internal/buildspec/report.go
      Note: Shows buildspec report supports ok/error only
    - Path: cmd/xgoja/internal/generate/generate.go
      Note: WriteAll writes generated files including go.mod
    - Path: cmd/xgoja/internal/generate/gomod.go
      Note: RenderGoMod generates go.mod content
ExternalSources: []
Summary: ""
LastUpdated: 2026-06-07T10:10:51.79087904-04:00
WhatFor: ""
WhenToUse: ""
---









# Generated Module Path and Release Build Guidance

## Executive Summary

This ticket addresses two related pain points surfaced by the `goja-bleve` integration with xgoja: (1) the default generated `go.mod` module path `example.com/generated/<name>` looks like placeholder metadata in a checked-in generated host, and (2) release tooling (GoReleaser, manual builds) needs to understand that generated hosts are nested Go modules and must be built from their own directory. This document proposes a combined solution: allow specs to set the generated module path explicitly (Option A), improve the default path convention when no path is specified (Option B), emit a build guidance note at generation time (Option C), and add docs/examples for release packaging generated hosts (Option D).

**Recommendation:** Implement Options A, B, and C together. Option D is documentation only and should be tracked as a follow-up. Option A gives authors full control, Option B makes the default less confusing, and Option C prevents future release-tooling confusion.

## Problem Statement

When xgoja generates a binary for a project like `goja-bleve`, the generated host lives under `cmd/goja-bleve/` with its own `go.mod`:

```
goja-bleve/
├── .goreleaser.yaml
├── go.mod                    # root module: github.com/go-go-golems/goja-bleve
├── cmd/
│   └── goja-bleve/
│       ├── go.mod             # generated nested module: example.com/generated/goja-bleve-vectors
│       ├── main.go            # generated entrypoint
│       └── xgoja.gen.json     # embedded runtime spec
```

Two problems arise:

1. **Placeholder-looking module path:** The default generated module line is:
   ```
   module example.com/generated/goja-bleve-vectors
   ```
   This reads like boilerplate/template metadata in a checked-in file, even though it is a legitimate generated module path. It leads developers to question whether it should be changed.

2. **Nested module boundary confusion:** GoReleaser configured from the root module with `main: ./cmd/goja-bleve` fails because the nested `cmd/goja-bleve/go.mod` makes it a separate module. The correct config requires:
   ```yaml
   builds:
     - id: goja-bleve-linux
       dir: cmd/goja-bleve
       main: .
   ```
   This works, but developers initially blame the `example.com/generated/...` path instead of the actual cause (nested module boundary).

The root cause is not the module path itself — `example.com/generated/...` is valid and correct — but that:
- The path looks like placeholder metadata, causing unnecessary confusion
- Release tooling documentation doesn't make the nested-module pattern obvious
- There is no opt-in to a more meaningful module path

## Current-State Architecture (Evidence-Based)

### Module path generation

The default module path is set in `cmd/xgoja/internal/buildspec/load.go` in the `applyDefaults()` function:

**File:** `cmd/xgoja/internal/buildspec/load.go`, line ~85

```go
if strings.TrimSpace(buildSpec.Go.Module) == "" {
    buildSpec.Go.Module = "example.com/generated/" + sanitizeModulePathPart(buildSpec.Name)
}
```

The `sanitizeModulePathPart()` function (same file) lowercases the name and replaces non-alphanumeric characters with dashes. For `name: goja-bleve-vectors`, this produces `example.com/generated/goja-bleve-vectors`.

### go.mod rendering

The generated `go.mod` content is produced by `cmd/xgoja/internal/generate/gomod.go` in `RenderGoMod()`:

**File:** `cmd/xgoja/internal/generate/gomod.go`, line ~18

```go
fmt.Fprintf(&b, "module %s\n\n", buildSpec.Go.Module)
```

This reads `buildSpec.Go.Module` directly. The field comes from the `go: module` field in the xgoja.yaml spec, which maps to `buildspec.GoSpec.Module`.

### BuildSpec schema

**File:** `cmd/xgoja/internal/buildspec/build_spec.go`

```go
type GoSpec struct {
    Version string            `yaml:"version" json:"version"`
    Module  string            `yaml:"module" json:"module"`
    Tags    []string          `yaml:"tags" json:"tags,omitempty"`
    LDFlags []string          `yaml:"ldflags" json:"ldflags,omitempty"`
    Env     map[string]string `yaml:"env" json:"env,omitempty"`
    Imports []GoImportSpec    `yaml:"imports" json:"imports,omitempty"`
}
```

The `Module` field is already supported in the YAML schema (see `goja-bleve/cmd/goja-bleve/xgoja-vectors.yaml` does not set it, so it falls through to the default).

### File generation flow

**File:** `cmd/xgoja/internal/generate/generate.go`, `WriteAll()` function

```go
files := map[string]string{
    "go.mod":         RenderGoMod(buildSpec, opts),
    "main.go":        RenderMain(buildSpec),
    "xgoja.gen.json": RenderEmbeddedSpec(buildSpec),
}
```

The generated files land in the temp work directory, which becomes the generated module root. When the host is a subdirectory of the spec (like `cmd/goja-bleve/`), the go.mod is written there as a nested module.

### Existing documentation

**File:** `cmd/xgoja/doc/02-user-guide.md`

The user guide mentions `go.module` defaults to `example.com/generated/<name>` in the "Top-level fields" section:

> `go` controls the generated module. `go.version` defaults to `1.26`, and `go.module` defaults to `example.com/generated/<name>`.

But there is no mention of the nested-module build pattern in release tooling, no GoReleaser examples for generated hosts, and no note emitted after generation.

### goja-bleve concrete example

**Spec:** `goja-bleve/cmd/goja-bleve/xgoja-vectors.yaml`

```yaml
name: goja-bleve-vectors
# No go.module specified → defaults to example.com/generated/goja-bleve-vectors
```

**Generated go.mod:**
```
module example.com/generated/goja-bleve-vectors
go 1.26.4
```

**GoReleaser config (corrected):**
```yaml
builds:
  - id: goja-bleve-linux
    dir: cmd/goja-bleve
    main: .
```

## Takeover Review Notes (2026-06-07)

A second pass over the code found several corrections to the initial plan:

1. **`go.module` is already supported.** The issue's Option A does not require schema or generation code. The only required work is documentation/examples and tests that prove explicit values are preserved.
2. **`buildspec.Report` has no warning status today.** The initial idea to "emit a validation warning" is not a small additive change; it would require expanding the report model beyond `ok`/`error`. Do not add warnings for this ticket unless the scope explicitly changes.
3. **`xgoja build` writes a temporary module, not the checked-in release module.** `cmd_build.go` calls `generate.WriteAll(workDir, ...)` and then runs `go mod tidy` / `go build .` in that `workDir`. GoReleaser builds like `dir: cmd/goja-bleve, main: .` apply when a project has checked in or otherwise materialized the generated host as a nested module under the repository.
4. **The build guidance hook belongs in `cmd/xgoja/cmd_build.go`, not `internal/buildexec`.** `buildexec` intentionally only runs `go mod tidy` and `go build`; it has no output writer, no `BuildSpec`, and no user-facing command context.
5. **Release guidance should not assume xgoja can infer the GoReleaser `dir` reliably.** The command can print the generated workspace path and a generic nested-module snippet. The docs should carry the authoritative GoReleaser example.

These corrections are reflected in the updated implementation plan below.

## Gap Analysis

### What's missing today

1. **Explicit generated module path is under-documented:** Authors can already set `go.module`, but the docs do not make this prominent for checked-in generated hosts. The gap is discoverability, not schema support.

2. **Default path looks like placeholder:** `example.com/generated/...` is syntactically valid, but conveys documentation-example semantics and can look accidental in checked-in generated `go.mod` files.

3. **`xgoja build` output is too terse for module-boundary debugging:** The command currently prints only `generated build workspace: <workDir>` after `generate.WriteAll()`. It does not show the generated module path, that `workDir` is the module root for the build, or how this maps to manual inspection with `--keep-work`.

4. **Release packaging docs are missing:** The docs do not explain that a generated host with its own `go.mod` must be built as a nested module (`dir: <generated-module-dir>`, `main: .`) rather than as a subpackage of the parent module (`main: ./cmd/...`).

### What exists today

1. **The `go.module` field already works:** It is part of `buildspec.GoSpec` and is consumed directly by `RenderGoMod()`.
2. **Explicit values are preserved:** `applyDefaults()` only fills `buildSpec.Go.Module` when the value is empty.
3. **`xgoja build` already identifies the generated workspace:** `cmd_build.go` prints `generated build workspace: <workDir>` immediately after `generate.WriteAll()`.
4. **There is no warning mechanism:** `buildspec.Report` currently supports only `StatusOK` and `StatusError`.

## Proposed Solution

### Option A: Document explicit generated module paths (docs/tests only)

**What changes:** No schema or rendering changes are required. Add docs showing:

```yaml
go:
  module: github.com/go-go-golems/goja-bleve/cmd/goja-bleve
```

Also add or tighten tests that prove explicit `go.module` values survive defaulting and render into `go.mod` unchanged.

**Do not implement:** a validation warning for missing `go.module`. The current report model has no warning status; adding one is larger than this ticket and is not required by Issue #61.

### Option B: Improve the default generated module path

**What changes in `cmd/xgoja/internal/buildspec/load.go`:**

Replace the current default:

```go
// Current
buildSpec.Go.Module = "example.com/generated/" + sanitizeModulePathPart(buildSpec.Name)
```

With a less placeholder-looking convention:

```go
// Proposed
buildSpec.Go.Module = "xgoja.generated/" + sanitizeModulePathPart(buildSpec.Name)
```

This produces paths like `xgoja.generated/goja-bleve-vectors` which are:

- clearly internal/generated;
- shorter than `example.com/generated/...`;
- not tied to the `example.com` documentation domain;
- syntactically valid as a Go module path.

**Important caveat:** the generated module path itself is not how dependencies are resolved. It names the generated host module. Dependencies still come from `require` and `replace` entries emitted by `RenderGoMod()`.

### Option C: Emit clearer xgoja build workspace guidance

**What changes in `cmd/xgoja/cmd_build.go`:**

Extend the existing post-generation line:

```go
_, _ = fmt.Fprintf(c.out, "generated build workspace: %s\n", workDir)
```

into a small build guidance block that includes:

- generated workspace path (`workDir`);
- generated module path (`buildSpec.Go.Module`);
- whether the workspace is temporary or user-supplied;
- a manual inspection hint: pass `--keep-work` when the workspace is temporary;
- the command shape xgoja itself runs: `go mod tidy` and `go build .` inside `workDir`.

Example output:

```text
generated build workspace: /tmp/xgoja-build-1234
generated module: xgoja.generated/goja-bleve-vectors
xgoja builds from the generated module root:
  cd /tmp/xgoja-build-1234 && go mod tidy && go build .
use --keep-work to inspect generated go.mod/main.go after the build
```

For release packaging, print only a compact pointer, not a guessed config:

```text
release note: if you check this generated host into a repository as a nested Go module, configure GoReleaser with dir: <generated-module-dir> and main: .
```

**Do not put this in `internal/buildexec`:** that package is intentionally command-execution-only and has no stdout writer or user-facing command context.

### Option D: Add docs/examples for release packaging

**What changes in `cmd/xgoja/doc/02-user-guide.md` and `06-buildspec-reference.md`:**

Add a section "Release packaging generated hosts" with:

- a GoReleaser example using `dir:` and `main: .`;
- an explanation that `main: ./cmd/generated-app` only works when `cmd/generated-app` is a package inside the parent module, not when it has its own `go.mod`;
- a note that `xgoja build` uses a generated build workspace, while GoReleaser usually builds a checked-in generated host directory;
- the goja-bleve pattern as the concrete example.

## Decision Records

### DR-1: Use `xgoja.generated/` as the new default prefix (Option B)

**Context:** The current `example.com/generated/` prefix looks like boilerplate. We need a new default that is clearly generated but doesn't use the `example.com` documentation/example namespace.

**Options considered:**
1. `xgoja.generated/` — short, unambiguous, self-describing
2. `generated.xgoja/` — same concept, different ordering
3. `xgoja.host/` — focuses on "host binary" concept
4. Keep `example.com/generated/` — no behavior change

**Decision:** Option 1 (`xgoja.generated/`). It is concise, self-describing, and keeps the generated-host name out of `example.com`.

**Consequences:**
- Specs without explicit `go.module` will get a different generated `module` line.
- Checked-in generated `go.mod` files may show a cosmetic diff after regeneration.
- Existing specs that set `go.module` explicitly are unaffected.

### DR-2: Put build guidance in `cmd_build.go` stdout, not in `buildexec`

**Context:** Developers need clearer output when debugging generated build workspaces and nested module release failures.

**Options considered:**
1. Extend `cmd_build.go` output after `generate.WriteAll()`.
2. Add output to `internal/buildexec`.
3. Write a persistent generated `BUILD-GUIDANCE.md` file.

**Decision:** Option 1. `cmd_build.go` already owns the command output writer and has access to the settings, `workDir`, and `BuildSpec`.

**Consequences:**
- No extra generated files.
- `buildexec` remains a small command runner.
- The output can accurately describe xgoja's own build workspace without guessing repository release layout.

## Implementation Plan

### Phase 1: Improve default module path (Option B)

**Files to modify:**
1. `cmd/xgoja/internal/buildspec/load.go` — change the default in `applyDefaults()`
2. `cmd/xgoja/internal/generate/generate_test.go` — update expectations only where tests rely on defaulted module paths; leave tests with explicit module values alone unless the test's fixture deliberately models the old default
3. `cmd/xgoja/doc/02-user-guide.md` — update the documented default
4. `cmd/xgoja/doc/06-buildspec-reference.md` — update examples/default notes where relevant
5. `cmd/xgoja/doc/03-tutorial-using-xgoja-yaml.md` — update example module path if it is meant to show the default convention

**Steps:**
1. Change the default string in `applyDefaults()` from `example.com/generated/` to `xgoja.generated/`.
2. Add or update a buildspec-defaulting test that loads a spec without `go.module` and asserts `xgoja.generated/<sanitized-name>`.
3. Keep a preservation test for explicit `go.module` values.
4. Update docs and examples that teach the default path.

### Phase 2: Build workspace guidance (Option C)

**Files to modify:**
1. `cmd/xgoja/cmd_build.go` — extend the existing `generated build workspace` output
2. `cmd/xgoja/root_test.go` or command-level build tests — update expected output snippets

**Steps:**
1. After `generate.WriteAll()` completes, print the generated module path and clarify that xgoja runs `go mod tidy` and `go build .` inside `workDir`.
2. If `settings.WorkDir == ""` and `settings.KeepWork == false`, print a `--keep-work` inspection hint.
3. Add a release note that says nested checked-in generated hosts should use GoReleaser `dir:` + `main: .`, without trying to infer the exact `dir` value.
4. Do not add this output to `buildexec.GoModTidy` or `buildexec.GoBuild`.

### Phase 3: Documentation for release packaging (Option D)

**Files to modify:**
1. `cmd/xgoja/doc/02-user-guide.md` — add "Release packaging generated hosts" section
2. `cmd/xgoja/doc/06-buildspec-reference.md` — add compact GoReleaser snippet

**Steps:**
1. Explain the difference between a package inside the parent module and a nested generated module with its own `go.mod`.
2. Include a complete GoReleaser example using `dir: cmd/my-app` and `main: .`.
3. Explain why `main: ./cmd/my-app` fails when `cmd/my-app/go.mod` exists.
4. Add a troubleshooting row for `main module does not contain package .../cmd/...`.

### Phase 4: Explicit module path documentation (Option A — docs/tests only)

**Files to modify:**
1. `cmd/xgoja/doc/06-buildspec-reference.md` — document `go.module` with a checked-in host example
2. `cmd/xgoja/doc/02-user-guide.md` — explain when to set `go.module`
3. `cmd/xgoja/internal/buildspec` tests if a preservation test does not already exist

**Steps:**
1. Add a note about `go.module` with an example showing:
   ```yaml
   go:
     module: github.com/go-go-golems/goja-bleve/cmd/goja-bleve
   ```
2. Explain that this is most useful when generated host files are checked in or released as a nested module.
3. Test that explicit `go.module` survives `applyDefaults()` unchanged and renders into `go.mod` unchanged.

## Testing and Validation Strategy

### Unit tests

1. **Defaulting test:** Verify that loading/defaulting a spec without `go.module` sets `buildSpec.Go.Module` to `xgoja.generated/<sanitized-name>`.

2. **Explicit preservation test:** Verify that a spec with `go.module: github.com/acme/project/cmd/app` is not overwritten by defaults.

3. **RenderGoMod test:** Verify that `RenderGoMod()` writes the module path present in `buildSpec.Go.Module`; do not conflate this with defaulting unless the fixture actually passed through `buildspec.LoadFile()`.

4. **Command output test:** Update `cmd/xgoja/root_test.go` or another command-level test so it still expects `generated build workspace` and also checks the new module/build guidance text.

### Integration tests

5. **xgoja build smoke:** Run `xgoja build` on a representative spec with `--keep-work`, inspect the generated `go.mod` in the kept workspace, and confirm the output guidance is printed.

6. **goja-bleve validation:** For the concrete motivating repository, run the existing vector smoke target when the local environment supports FAISS/CGO. If that environment is not available, validate only docs/config shape and record the limitation.

7. **GoReleaser validation:** Prefer `goreleaser release --snapshot --clean --single-target` in `goja-bleve` when toolchain and CGO cross-compiler dependencies are installed. Treat this as a heavier integration check, not a mandatory unit-test gate.

### Doc validation

8. Run `docmgr validate frontmatter --doc path/to/doc.md --suggest-fixes` on updated docs and `docmgr doctor --ticket XGOJA-018` for the ticket.

## Risks, Alternatives, and Open Questions

### Risks

1. **Visible generated `go.mod` diffs:** Projects that check in generated `go.mod` files with `example.com/generated/...` will see the path change on next regeneration unless they set `go.module` explicitly.

2. **Unclear default-module semantics:** `xgoja.generated/...` is intentionally non-resolvable. That is fine for a main/generated host module, but docs should explain that dependencies come from `require`/`replace`, not from resolving the module's own path.

3. **No warning status in validation:** Adding warnings would touch the validation/reporting model and CLI output. This ticket should not expand into that unless explicitly requested.

4. **GoReleaser validation may be environment-dependent:** The goja-bleve vector build requires CGO/FAISS/cross-compiler details, so full release smoke may not be possible on every development machine.

### Alternatives not chosen

1. **Deriving the module path from `target.output`:** Rejected because `target.output` is resolved relative to the shell's CWD, not the spec directory. This makes it unreliable as a source for module paths.

2. **Adding a new `go.moduleTemplate` field:** Rejected as unnecessary complexity. The existing `go.module` field is sufficient.

3. **Adding validation warnings for missing `go.module`:** Rejected for this ticket because `buildspec.Report` has only `ok` and `error` statuses.

4. **Putting guidance in `internal/buildexec`:** Rejected because `buildexec` has no output writer and should remain a command runner.

### Open questions

1. Is `xgoja.generated/` the final preferred prefix, or should the project choose a real domain it controls (for example `go-go-golems.dev/xgoja/generated/...`)? **Recommendation:** Keep `xgoja.generated/` unless maintainers prefer a project-owned domain for aesthetics.

2. Should the release guidance note print for every `xgoja build`, or only when the spec directory itself contains a `go.mod` suggesting a checked-in generated host? **Recommendation:** Always print a short workspace note; make the GoReleaser note compact and generic so it is not misleading.

3. Should there be a dedicated command or doc for materializing a standalone generated host into a repository directory? **Recommendation:** Out of scope for Issue #61; document the existing nested-module pattern first.

## References

- **Issue #61:** [xgoja: make generated module paths and nested-module release builds clearer](https://github.com/go-go-golems/go-go-goja/issues/61)
- **Key files:**
  - `cmd/xgoja/internal/buildspec/load.go` — `applyDefaults()` (default module path)
  - `cmd/xgoja/internal/buildspec/build_spec.go` — `GoSpec` struct
  - `cmd/xgoja/internal/generate/gomod.go` — `RenderGoMod()` (go.mod output)
  - `cmd/xgoja/internal/generate/generate.go` — `WriteAll()` (file generation)
  - `cmd/xgoja/doc/02-user-guide.md` — xgoja.yaml documentation
  - `cmd/xgoja/doc/06-buildspec-reference.md` — buildspec quick reference
- **Concrete example:** `goja-bleve/cmd/goja-bleve/xgoja-vectors.yaml`, `goja-bleve/.goreleaser.yaml`
