---
Title: Intern guide to pragmatic xgoja artifact selection
Ticket: XGOJA-ARTIFACT-SELECTION-2026-07-18
Status: active
Topics:
    - xgoja
    - backend
    - testing
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ws://go-go-goja/cmd/xgoja/cmd_build.go
      Note: Build command selection and WriteAllPlan call site
    - Path: ws://go-go-goja/cmd/xgoja/cmd_generate.go
      Note: Generate command selection and package/source/template call sites
    - Path: ws://go-go-goja/cmd/xgoja/doc/17-xgoja-v2-reference.md
      Note: Public artifact contract and current limitation to update
    - Path: ws://go-go-goja/cmd/xgoja/internal/generate/plan.go
      Note: Embedded primary-source union and global asset selection
    - Path: ws://go-go-goja/cmd/xgoja/internal/generate/templates.go
      Note: Independent generator target and runtime metadata derivation
    - Path: ws://go-go-goja/cmd/xgoja/internal/plan/plan.go
      Note: Compiled plan has both Config.Artifacts and Artifacts representations
    - Path: ws://go-go-goja/cmd/xgoja/v2_plan_helpers.go
      Note: Current first-primary selector and primary implementation location
ExternalSources: []
Summary: Evidence-backed design and implementation guide for making xgoja build and generate select command-compatible artifacts without introducing generalized multi-target orchestration.
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: Give a new intern enough architectural, API, file, testing, and operational context to implement and review the artifact-selection fix safely.
WhenToUse: Use before changing xgoja build/generate artifact selection, generated-plan target metadata, or multi-artifact documentation.
---


# Intern guide to pragmatic xgoja artifact selection

## 1. Executive summary

`xgoja` turns a declarative `xgoja/v2` YAML specification into either a generated Go executable workspace or reusable generated Go source. A specification may contain several artifacts, such as a binary, a reusable runtime package, TypeScript declarations, and static web assets. Today, however, both `xgoja build` and `xgoja generate` call the same `targetFromPlan` helper. That helper chooses the first non-support artifact, without considering which command is running.

This creates an order-dependent failure:

- When `binary` is first, `xgoja build` works and `xgoja generate` rejects the binary-derived target.
- When `runtime-package` is first, `xgoja generate` works and `xgoja build` rejects the package-derived target.

There is a second, subtler problem. Command-level selection and generator-level selection are separate. `targetFromPlan` skips `dts` and `embedded-assets`, but generator functions inspect `Config.Artifacts` from the beginning. A support artifact placed first can therefore produce a build that reports selecting the binary while embedding runtime metadata whose target kind is `embedded-assets`.

The proposed fix is intentionally pragmatic:

1. Define which primary artifact types are compatible with `build` and `generate`.
2. Require exactly one compatible primary for the invoked command.
3. Return clear no-match or ambiguity errors containing artifact IDs and types.
4. Create a shallow command-scoped copy of `plan.Plan` containing:
   - the selected primary artifact first, and
   - the existing global `dts` and `embedded-assets` support artifacts.
5. Pass that scoped plan to all generator/rendering calls.
6. Add focused unit, command, and embedding tests.
7. Document the command/artifact matrix and the one-compatible-primary limitation.

The initial design deferred `--artifact`; implementation later added it as a small, local extension after command-compatible scoping existed. `--artifact <id>` selects one compatible primary when a spec intentionally has multiple candidates. The design still does **not** add artifact dependency graphs, arbitrary target closures, or a generalized build orchestrator.

## 2. Problem statement

### 2.1 User-visible requirement

A consumer wants one canonical specification containing both outputs:

```yaml
artifacts:
  - id: binary
    type: binary
    output: dist/my-tool
    sources: [local-verbs, app-help]

  - id: runtime-package
    type: runtime-package
    output: internal/xgojaruntime
    package: xgojaruntime
    sources: [local-verbs, app-help]

  - id: web-assets
    type: embedded-assets
    sources: [webapp]
```

The expected command behavior is uncomplicated:

```bash
xgoja build -f xgoja.yaml
xgoja generate -f xgoja.yaml
```

`build` should choose the sole build-compatible primary (`binary`). `generate` should choose the sole generate-compatible primary (`runtime-package`). The order of those two entries should not matter.

### 2.2 Current failure matrix

The ticket reproduction script is:

```text
ttmp/2026/07/18/XGOJA-ARTIFACT-SELECTION-2026-07-18--pragmatic-command-compatible-artifact-selection-for-xgoja-build-and-generate/scripts/01-reproduce-artifact-order.sh
```

Its captured output is beside it in `01-reproduce-artifact-order.log`.

Observed behavior at repository commit `69b69b6`:

| Artifact order | `xgoja build` | `xgoja generate` |
| --- | --- | --- |
| binary, runtime-package | succeeds | fails with `got "xgoja"` |
| runtime-package, binary | fails with `target.kind package` | succeeds |
| embedded-assets, binary | command selects binary | generated runtime metadata says `kind: embedded-assets` |

Exact errors:

```text
Error: xgoja generate supports target.kind package, source, or template; got "xgoja"
```

```text
Error: target.kind package is source generation only; use xgoja generate -f ...
```

These errors mention `target.kind`, but `target` is not a top-level field in the v2 YAML schema. It is derived from an artifact. A user cannot fix the error by adding `target:` to the file.

### 2.3 Why this matters beyond output selection

Selecting only the output path is not enough. The artifact order also influences:

- generated `main.go` target behavior,
- `RuntimePlan.Target.Kind`,
- `RuntimePlan.Target.Output`,
- which JS/help sources are copied into `xgoja_embed`,
- which artifacts appear in embedded runtime metadata,
- package and custom-template generation.

A fix that changes only the command's local `target` variable can still generate internally inconsistent output.

## 3. Scope and non-goals

### 3.1 In scope

- Command-compatible primary artifact classification.
- Exactly-one-compatible-primary selection.
- Clear diagnostics for no compatible primary and ambiguous compatible primaries.
- A shallow, command-scoped plan copy.
- Preservation of existing global `dts` and `embedded-assets` support artifacts.
- Correct selected target metadata and selected-primary source embedding.
- Focused tests at helper, CLI command, and generated-output levels.
- A concise documentation update.
- Preservation of existing single-primary behavior.

### 3.2 Explicit non-goals

- No dependency graph between artifacts.
- No `depends-on`, `for`, or target-specific support-artifact schema.
- No command that builds every binary or generates every package.
- No parallel artifact execution.
- No redesign of `specv2.ArtifactSpec`.
- No changes to provider resolution, source-graph resolution, runtime modules, host services, or authentication.
- No redesign of `NewBundle`, `AttachDefaultCommands`, or generated runtime-package APIs.
- No attempt to recover or preserve undocumented first-artifact behavior when multiple compatible primaries exist; ambiguity becomes an actionable error.

### 3.3 Pragmatic quality rule

Add a safeguard when it is local, easy to test, and closes a demonstrated failure. Do not introduce an abstraction whose value depends on hypothetical future artifact graphs.

The shallow scoped plan qualifies because the current reproduction already demonstrates contradictory command and generated-runtime target selection. A dependency model does not qualify because no current consumer requires target-specific asset ownership.

## 4. System orientation for a new intern

### 4.1 What xgoja is

Goja is a JavaScript runtime implemented in Go. `xgoja` is the repository's generator and CLI layer around that runtime. It lets an application author describe:

- Go provider packages,
- JavaScript-visible runtime modules,
- local or provider-owned source sets,
- generated command surfaces,
- generated output artifacts.

The CLI then validates the spec, resolves providers and sources, generates Go code and embedded files, and optionally invokes the Go toolchain.

### 4.2 Main directories involved

| Path | Responsibility |
| --- | --- |
| `cmd/xgoja/main.go` | Process entrypoint. Builds the root command and executes it. |
| `cmd/xgoja/root.go` | Registers Glazed-backed `build`, `generate`, `gen-dts`, `doctor`, and related commands on Cobra. |
| `cmd/xgoja/v2_bridge.go` | Loads only native v2 specs and compiles them into `plan.Plan`. |
| `cmd/xgoja/v2_plan_helpers.go` | Current first-artifact target derivation; primary implementation location for this ticket. |
| `cmd/xgoja/cmd_build.go` | Build command: generates a temporary Go workspace and invokes `go mod tidy` / `go build`. |
| `cmd/xgoja/cmd_generate.go` | Generate command: emits runtime package, source fragments, or custom-template output. |
| `cmd/xgoja/internal/specv2/` | YAML schema, defaults, validation, rendering, and v1 migration. |
| `cmd/xgoja/internal/plan/` | Compiles validated config into provider, source, command, artifact, and workspace plans. |
| `cmd/xgoja/internal/generate/` | Copies embedded files and renders generated Go/runtime-plan files. |
| `cmd/xgoja/root_test.go` | End-to-end command tests through the Cobra/Glazed root. |
| `cmd/xgoja/internal/generate/generate_test.go` | Generator rendering and embedded-file tests. |
| `cmd/xgoja/doc/17-xgoja-v2-reference.md` | Main v2 schema and artifact reference. |
| `examples/xgoja/14-generated-runtime-package/` | Runnable single-runtime-package example and generated host API reference. |

### 4.3 CLI composition

`cmd/xgoja/root.go:17-65` constructs the Cobra root. The actual `build` and `generate` commands implement Glazed `cmds.BareCommand` and are adapted into Cobra commands.

```text
main.go
  |
  v
newRootCommand(stdout)
  |
  +-- newBuildCommand
  +-- newGenerateCommand
  +-- newGenDTSCommand
  +-- newDoctorCommand
  +-- ...
  |
  v
Cobra Execute
```

Settings are decoded from Glazed values inside each command's `Run` method. This ticket does not need to change the root command or settings schema.

### 4.4 V2 loading and compilation

`cmd/xgoja/v2_bridge.go:11-32` performs the loading pipeline:

```text
YAML file
  -> DetectSchema
  -> specv2.LoadFile
       -> parse
       -> defaults
       -> validation
  -> plan.Compile
       -> provider graph
       -> Go workspace plan
       -> source graph
       -> command plans
       -> artifact plans
  -> *plan.Plan
```

`plan.Plan` is defined at `cmd/xgoja/internal/plan/plan.go:21-29`:

```go
type Plan struct {
    Config         specv2.Config
    GoModules      *workspace.Plan
    ProviderGraph  *providergraph.Graph
    SourceGraph    *sourcegraph.Graph
    Commands       []CommandPlan
    Artifacts      []ArtifactPlan
    RuntimeAliases []string
}
```

Two artifact representations matter:

- `Plan.Config.Artifacts` is consumed by generator rendering and embedded-source logic.
- `Plan.Artifacts` is consumed by `targetFromPlan`.

A scoped copy must update **both** slices or later stages can disagree.

### 4.5 Artifact schema

`specv2.ArtifactSpec` is defined at `cmd/xgoja/internal/specv2/types.go:153-163`:

```go
type ArtifactSpec struct {
    ID       string
    Type     string
    Output   string
    Package  string
    Import   string
    Root     string
    Template string
    Sources  []string
    Strict   bool
}
```

Validation accepts these types at `cmd/xgoja/internal/specv2/validate.go:218-242`:

- `binary`
- `runtime-package`
- `dts`
- `embedded-assets`
- `adapter`
- `cobra`
- `source`
- `template`

For this design, types are grouped by command role:

| Role | Artifact types | Consuming command |
| --- | --- | --- |
| Build primary | `binary`, `adapter`, `cobra` | `xgoja build` |
| Generate primary | `runtime-package`, `source`, `template` | `xgoja generate` |
| Support | `dts`, `embedded-assets` | Retained beside the selected primary; `gen-dts` separately owns DTS output |

The mapping between artifact type and existing internal target kind is:

| Artifact type | Internal target kind |
| --- | --- |
| `binary` | `xgoja` |
| `runtime-package` | `package` |
| all other primary types | unchanged |

### 4.6 Build flow

The important portion of `cmd/xgoja/cmd_build.go` is at lines 81-126:

```text
loadV2Plan
  -> targetFromPlan
  -> reject source-generation kinds
  -> choose output
  -> generate.WriteAllPlan
  -> optional dry-run return
  -> go mod tidy
  -> go build
```

`generate.WriteAllPlan` writes:

- copied embedded JS verbs,
- copied help sources,
- copied static assets,
- `go.mod`,
- `main.go`,
- `xgoja.runtime.json`.

The command currently passes the original compiled plan to `WriteAllPlan`. Therefore any command-compatible selection fix must also change the plan supplied to that call.

### 4.7 Generate flow

The important portion of `cmd/xgoja/cmd_generate.go` is at lines 82-164:

```text
loadV2Plan
  -> targetFromPlan
  -> require package/source/template kind
  -> resolve output/package/template overrides
  -> template-data OR dry-run OR clean
  -> WritePackagePlan / WriteSourceFragmentsPlan / WriteCustomTemplatePlan
```

All rendering paths must receive the same scoped plan, including `--template-data`. Otherwise dry runs or template data can describe a different target than actual generated files.

### 4.8 Generator target derivation

`cmd/xgoja/internal/generate/templates.go` independently derives target information:

- `targetDataFromPlanArtifacts`, lines 287-300
- `RenderRuntimePlanJSONFromPlan`, lines 343-385
- `targetOutputFromPlanArtifacts`, lines 388-395

The current `targetDataFromPlanArtifacts` does not skip support artifacts. This is why a leading `embedded-assets` artifact can become `RuntimePlan.Target.Kind` even though command-level selection chose a binary.

### 4.9 Embedded source selection

`cmd/xgoja/internal/generate/plan.go:199-227` currently unions sources across every executable-style primary:

```text
binary sources ---------+
runtime-package sources +--> copied JS/help source IDs
source/template sources -+

all embedded-assets sources --> copied static asset IDs
```

If both a binary and runtime package are present with different source lists, generating either output currently embeds their union. The scoped plan solves this without adding new generator APIs:

```text
selected primary sources --> copied JS/help source IDs
retained embedded-assets --> copied static asset IDs
unselected primaries ----> absent from scoped plan
```

## 5. Root cause analysis

### 5.1 First non-support artifact wins

`cmd/xgoja/v2_plan_helpers.go:16-35` loops over artifacts, skips empty/`dts`/`embedded-assets`, maps two types, and immediately returns. It has no command context.

Pseudocode of current behavior:

```text
for artifact in plan.artifacts:
    if artifact is support:
        continue
    return map_to_target(artifact)
return default_binary_target
```

Both build and generate call this same function.

### 5.2 Command validation happens too late

The command selects an arbitrary primary first and only afterward asks whether the selected kind is compatible. It never searches for another compatible primary.

```text
select first primary
  |
  +-- build: reject package/source/template
  |
  +-- generate: reject anything except package/source/template
```

The command should instead classify first and select among compatible candidates.

### 5.3 Generator reselects from the original list

Even if command selection were corrected, generator helpers receive the complete original artifact list and derive their own first target and source union. This violates a useful invariant:

> One command execution must have one selected primary artifact, and every downstream rendering decision must see that same primary.

The scoped-plan copy establishes that invariant without changing generator function signatures.

## 6. Proposed design

### 6.1 API shape

Keep the implementation in `cmd/xgoja/v2_plan_helpers.go`. It is command glue, not a reusable domain package.

Suggested private types:

```go
type artifactCommand string

const (
    artifactCommandBuild    artifactCommand = "build"
    artifactCommandGenerate artifactCommand = "generate"
)

type selectedPlanTarget struct {
    ID      string
    Type    string
    Kind    string
    Output  string
    Package string
    Template string
    Plan    *plan.Plan
}
```

Suggested entrypoint:

```go
func selectPlanTarget(
    compiled *plan.Plan,
    command artifactCommand,
) (*selectedPlanTarget, error)
```

Returning the scoped plan together with target metadata makes it harder for callers to accidentally select correctly and then generate from the original plan.

### 6.2 Compatibility predicates

Keep these as small explicit switches rather than a registry or plugin mechanism:

```go
func isCompatiblePrimary(command artifactCommand, artifactType string) bool {
    switch command {
    case artifactCommandBuild:
        return artifactType == "binary" ||
            artifactType == "adapter" ||
            artifactType == "cobra"
    case artifactCommandGenerate:
        return artifactType == "runtime-package" ||
            artifactType == "source" ||
            artifactType == "template"
    default:
        return false
    }
}

func isSupportArtifact(artifactType string) bool {
    return artifactType == "dts" || artifactType == "embedded-assets"
}
```

Do not infer compatibility from today's rejection conditions. An allow-list documents intended behavior and makes tests obvious.

### 6.3 Selection algorithm

```text
function selectPlanTarget(plan, command):
    if plan is nil:
        return error("xgoja <command>: plan is nil")

    candidates = []
    allArtifacts = []

    for artifact in plan.Artifacts:
        record artifact ID/type in allArtifacts
        if compatible(command, artifact.type):
            append artifact to candidates

    if candidates is empty:
        return error containing command and all artifact IDs/types

    if candidates has more than one item:
        return error containing command and candidate IDs/types

    selected = candidates[0]
    scoped = scopePlan(plan, selected.ID)

    return target metadata + scoped plan
```

Use artifact ID to correlate `Plan.Artifacts` and `Config.Artifacts`; IDs are validated as unique by `specv2.Validate`.

### 6.4 Scoped-plan algorithm

The copy should be shallow except for artifact slices:

```text
function scopePlan(original, selectedID):
    clone = shallow copy of original
    clone.Config = shallow copy of original.Config

    selectedConfigArtifact = find selectedID in original.Config.Artifacts
    selectedPlanArtifact   = find selectedID in original.Artifacts

    clone.Config.Artifacts = [selectedConfigArtifact]
    clone.Artifacts        = [selectedPlanArtifact]

    for artifact in original order:
        if artifact is dts or embedded-assets:
            append to corresponding clone slice

    return &clone
```

Resulting shape:

```text
Original plan
  artifacts: [runtime-package, assets, binary, dts]
                     |
             select for build
                     v
Build-scoped plan
  artifacts: [binary, assets, dts]

Original plan
                     |
             select for generate
                     v
Generate-scoped plan
  artifacts: [runtime-package, assets, dts]
```

The selected primary is always first. This preserves the existing generator's first-primary assumptions while removing unselected primary source leakage.

### 6.5 Diagnostic contract

No compatible primary:

```text
xgoja build found no compatible primary artifact; build accepts binary, adapter, or cobra; configured artifacts: runtime-package (runtime-package), web-assets (embedded-assets)
```

Ambiguous compatible primaries:

```text
xgoja generate requires exactly one compatible primary artifact; found 2: runtime (runtime-package), custom (template)
```

Properties of good errors:

- Name the invoked command.
- Use YAML-facing artifact type names, not only derived `target.kind` names.
- Include IDs so the user can find the entries.
- State accepted types.
- Do not suggest adding a nonexistent `target:` field.
- Suggest `--artifact <id>` only when more than one primary is compatible with the invoked command.
- Do not imply that one invocation builds or generates every artifact.

### 6.6 Command integration

Build:

```go
target, scopedPlan, err := selectPlanTarget(compiledPlan, artifactCommandBuild, settings.Artifact)
if err != nil {
    return err
}
compiledPlan = scopedPlan

// Existing output/work-dir/build logic follows.
generate.WriteAllPlan(workDir, compiledPlan, opts)
```

Generate:

```go
target, scopedPlan, err := selectPlanTarget(compiledPlan, artifactCommandGenerate, settings.Artifact)
if err != nil {
    return err
}
compiledPlan = scopedPlan

// Every branch uses compiledPlan, including template-data.
generate.TemplateDataJSONFromPlan(compiledPlan, packageName)
generate.WritePackagePlan(output, compiledPlan, opts)
```

Delete the old post-selection kind rejection blocks once compatibility is enforced by selection. Keep output/package/template validation because those validate selected-artifact fields and CLI overrides.

### 6.7 Explicit `--artifact` selection (implementation amendment)

The initial design deferred an explicit selector. It was subsequently added because command-compatible scoping already makes its behavior local and deterministic: the flag chooses one compatible primary, while the existing scoped-plan rule still retains global support artifacts and excludes unselected primary JS/help sources.

`--artifact` does not build or generate multiple outputs, add artifact dependencies, or assign target-specific assets. It only resolves an otherwise actionable ambiguity. Unknown or incompatible IDs fail with a command-specific error naming accepted artifact types.

## 7. Architecture diagrams

### 7.1 Current flow

```text
                         +------------------+
                         | xgoja/v2 YAML    |
                         +---------+--------+
                                   |
                                   v
                         +------------------+
                         | loadV2Plan       |
                         | spec + compile   |
                         +---------+--------+
                                   |
                                   v
                         +------------------+
                         | plan.Plan        |
                         | all artifacts    |
                         +---------+--------+
                                   |
                         v         v
                  +--------+       +----------+
                  | build  |       | generate |
                  +---+----+       +----+-----+
                      |                 |
                      +--------+--------+
                               v
                    first non-support artifact
                               |
                   +-----------+-----------+
                   | incompatible? reject  |
                   +-----------+-----------+
                               |
                               v
                    generator sees original
                    all-artifact plan again
```

### 7.2 Proposed flow

```text
                         +------------------+
                         | xgoja/v2 YAML    |
                         +---------+--------+
                                   |
                                   v
                         +------------------+
                         | plan.Plan        |
                         | all artifacts    |
                         +---------+--------+
                                   |
                         +---------+---------+
                         | command context   |
                         | build / generate  |
                         +---------+---------+
                                   |
                                   v
                    +-----------------------------+
                    | compatible candidate scan   |
                    | require exactly one primary |
                    +--------------+--------------+
                                   |
                                   v
                    +-----------------------------+
                    | shallow scoped plan         |
                    | selected primary + supports |
                    +--------------+--------------+
                                   |
                                   v
                    +-----------------------------+
                    | existing generator APIs     |
                    | consistent target + sources |
                    +-----------------------------+
```

### 7.3 Source embedding after scoping

```text
binary.sources --------------------+
                                    | build-scoped plan
embedded-assets.sources ------------+----> generated binary embeds
runtime-package.sources -- removed -+

runtime-package.sources ------------+
                                    | generate-scoped plan
embedded-assets.sources ------------+----> generated package embeds
binary.sources ----------- removed -+
```

## 8. Design decisions

### Decision: Select by command compatibility

- **Context:** One spec needs a binary and runtime package, while both commands currently choose the same first primary.
- **Options considered:** Preserve first-primary behavior; reorder consumer YAML; use separate specs; scan for command-compatible artifacts.
- **Decision:** Scan for artifact types compatible with the invoked command.
- **Rationale:** It directly models user intent and removes order dependence with a small local change.
- **Consequences:** Specs with one compatible primary per command work in either order. Multiple compatible primaries become errors.
- **Status:** accepted

### Decision: Require exactly one compatible primary

- **Context:** Silently choosing the first of two binaries or two generate targets remains order-dependent.
- **Options considered:** First compatible wins; add `--artifact`; generate all; reject ambiguity.
- **Decision:** Reject zero or multiple compatible primaries with IDs/types in the error.
- **Rationale:** Deterministic, safe, and small. It avoids publishing an underdesigned selection API.
- **Consequences:** Existing ambiguous specs must be simplified. A future ticket may add explicit selection when a concrete consumer exists.
- **Status:** accepted

### Decision: Pass a shallow scoped plan to generators

- **Context:** Generator target and source logic re-reads the complete artifact list, producing contradictory metadata and source unions.
- **Options considered:** Change every generator API to accept a target; mutate the original plan; reorder only; shallow-copy and filter.
- **Decision:** Shallow-copy `plan.Plan` and `Config`, retain selected primary plus global support artifacts, and update both artifact slices.
- **Rationale:** It establishes one-primary consistency using existing generator APIs and avoids mutation shared across dry-run/template/render paths.
- **Consequences:** Generated runtime metadata no longer lists unselected primaries. Source embedding is scoped automatically. Global support artifacts remain global.
- **Status:** accepted

### Decision: Keep support artifacts global

- **Context:** The schema has no relationship from `embedded-assets` or `dts` to a primary artifact.
- **Options considered:** Drop supports; infer by order; add dependency fields; retain all supports.
- **Decision:** Retain all `dts` and `embedded-assets` artifacts in the scoped plan.
- **Rationale:** This matches current semantics and supports the real binary/runtime-package consumer, which shares web assets.
- **Consequences:** Different primaries cannot yet own different asset sets. That is an explicit limitation, not silently invented behavior.
- **Status:** accepted

### Decision: Add scoped `--artifact` selection

- **Context:** Explicit selection resolves a real ambiguity without requiring multi-output orchestration because selected-plan scoping already controls metadata and embedded sources.
- **Options considered:** Continue rejecting ambiguity; add an explicit selector; generate all candidates.
- **Decision:** Add `--artifact <id>` to `build` and `generate`.
- **Rationale:** It is a small, testable CLI extension that selects one command-compatible primary and retains existing scoped support-artifact semantics.
- **Consequences:** Multiple compatible primaries are now usable one-at-a-time. Artifact dependency graphs, target-specific assets, and build-all/generate-all remain out of scope.
- **Status:** accepted

## 9. Implementation guide

### Phase 1: Add helper-level selection

Primary file:

```text
cmd/xgoja/v2_plan_helpers.go
```

Steps:

1. Replace `targetFromPlan` with command-aware selection.
2. Add ID and original artifact type to target metadata.
3. Add small compatibility/support predicates.
4. Add deterministic artifact formatting for errors.
5. Add shallow plan scoping.
6. Avoid modifying `specv2`, `plan.Compile`, or generator public APIs.

Recommended new test file:

```text
cmd/xgoja/v2_plan_helpers_test.go
```

Helper tests should construct `plan.Plan` directly. They should not parse YAML or invoke Cobra.

### Phase 2: Integrate build and generate

Files:

```text
cmd/xgoja/cmd_build.go
cmd/xgoja/cmd_generate.go
```

Build changes:

- Call selection with build mode after `loadV2Plan`.
- Replace `compiledPlan` with the scoped copy.
- Remove the old generate-kind rejection.
- Continue honoring CLI `--output` over the selected artifact output.
- Ensure dry-run reports selected kind/output.

Generate changes:

- Call selection with generate mode.
- Replace `compiledPlan` with the scoped copy before template-data/dry-run/clean/render branches.
- Remove the old general kind rejection; keep the kind switch as an internal exhaustiveness check.
- Continue honoring CLI overrides.

### Phase 3: Add command-level regressions

Primary file:

```text
cmd/xgoja/root_test.go
```

Prefer small YAML helpers that accept an artifact slice. Command tests should exercise the real Glazed/Cobra path using `newRootCommand` and `root.SetArgs`.

Required matrix:

| Test | Expected result |
| --- | --- |
| build with binary before package | success; target binary |
| build with package before binary | success; target binary |
| generate with binary before package | success; target package |
| generate with package before binary | success; target package |
| support artifact before binary | generated runtime target is `xgoja`, not support type |
| two binaries | build ambiguity error containing both IDs/types |
| runtime-package plus template | generate ambiguity error containing both IDs/types |
| package only, invoke build | no-compatible error |
| binary only, invoke generate | no-compatible error |

### Phase 4: Test scoped embedding

Use generator or command integration tests with two local source directories:

```yaml
sources:
  - id: binary-verbs
    kind: jsverbs
    from: { dir: ./binary-verbs }

  - id: package-verbs
    kind: jsverbs
    from: { dir: ./package-verbs }

  - id: webapp
    kind: assets
    from: { dir: ./assets }

artifacts:
  - id: binary
    type: binary
    sources: [binary-verbs]

  - id: runtime-package
    type: runtime-package
    sources: [package-verbs]

  - id: assets
    type: embedded-assets
    sources: [webapp]
```

Assertions:

- Build workspace contains `binary-verbs`, not `package-verbs`.
- Generated package contains `package-verbs`, not `binary-verbs`.
- Both contain `webapp` assets.
- Embedded runtime target kind/output matches the selected primary.

This test is the main reason to scope rather than merely reorder.

### Phase 5: Documentation

Update:

```text
cmd/xgoja/doc/17-xgoja-v2-reference.md
```

Add a compact command matrix:

| Command | Compatible primary artifacts | Supporting artifacts |
| --- | --- | --- |
| `xgoja build` | `binary`, `adapter`, `cobra` | `dts`, `embedded-assets` |
| `xgoja generate` | `runtime-package`, `source`, `template` | `dts`, `embedded-assets` |
| `xgoja gen-dts` | first `dts` artifact, existing behavior | N/A |

State:

- Artifact order does not choose between build and generate primaries.
- Each command requires exactly one compatible primary.
- Multiple compatible primaries are not orchestrated and produce an error.
- `sources` on the selected primary control embedded JS/help.
- `embedded-assets` remain global support artifacts.
- Runtime-package hosts that need standalone verbs/help must list those source IDs.

Replace the current line saying the first binary-style artifact controls build with the new exactly-one-compatible-primary rule.

### Phase 6: Validation

Run from the repository root:

```bash
gofmt -w cmd/xgoja/v2_plan_helpers.go \
  cmd/xgoja/v2_plan_helpers_test.go \
  cmd/xgoja/cmd_build.go \
  cmd/xgoja/cmd_generate.go \
  cmd/xgoja/root_test.go

go test ./cmd/xgoja/... -count=1
go test ./... -count=1
go vet ./...
```

Then run the ticket reproduction script. After the fix, both build and generate cases should succeed regardless of binary/package order. Update the script's labels and expected statuses as part of implementation, or add a separate verification script so the original failure evidence remains preserved.

## 10. Testing strategy in detail

### 10.1 Unit tests

Target the pure selection/scoping helper. Unit tests should verify:

- artifact compatibility table,
- mapping from YAML type to internal kind,
- selected primary is first,
- support order is preserved,
- unselected primaries are removed,
- original plan is not mutated,
- `Config.Artifacts` and `Artifacts` remain synchronized,
- errors include stable IDs/types.

### 10.2 Command tests

Command tests prove settings decoding, plan loading, and command wiring use the helper correctly. Use dry-run where possible to avoid invoking `go mod tidy` or compilation.

Note: build dry-run still writes the generated workspace before returning. This makes `xgoja.runtime.json` and `main.go` available for assertions.

Generate dry-run returns before writing files. For generated package assertions, invoke generate normally with a fixture provider or a provider-free valid plan.

### 10.3 Generator tests

Most generator functions do not need modification. Add tests only if helper/command tests cannot clearly prove source scoping. Avoid moving command-selection policy into `internal/generate`; generators should consume the already-scoped plan.

### 10.4 Full regression suite

The baseline before implementation is green:

```text
go test ./cmd/xgoja/... -count=1
```

Observed baseline result:

```text
ok github.com/go-go-golems/go-go-goja/cmd/xgoja
ok github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/generate
ok github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/plan
ok github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/specv2
...
```

Run the full repository suite after focused tests because generated runtime metadata is shared by examples and provider integrations.

## 11. Risks and mitigations

### Risk: Existing ambiguous specs begin failing

This is intentional. Current behavior silently depends on order and only one primary is actually generated. The error must identify candidates clearly. Search repository examples before merging; current examples contain no specification with both a binary and runtime package, and no known example contains multiple primaries compatible with the same command.

### Risk: Scoped plan loses required support artifacts

Mitigation: retain all `dts` and `embedded-assets` artifacts and test embedded web assets. Do not filter providers, runtime modules, commands, sources, or Go workspace data.

### Risk: Only one artifact representation is scoped

`plan.Plan` contains both `Config.Artifacts` and `Artifacts`. Tests must assert both. Prefer one helper that constructs both slices from the same selected ID.

### Risk: Template-data uses the wrong plan

`--template-data` returns before file generation. Rebind `compiledPlan` to the scoped copy immediately after selection, before any command branch.

### Risk: Error wording leaks internal target kinds

Use YAML artifact types in selection errors. Internal kind remains useful in dry-run output but should not be the only diagnostic.

### Risk: The change grows into an orchestration framework

Stop if implementation proposes artifact dependency fields, an execution DAG, parallel output loops, or a public selector API. Those are separate designs requiring real consumers.

## 12. Alternatives considered

### Keep two near-identical YAML files

Rejected. It works operationally but duplicates providers, modules, sources, commands, and application settings. Drift is likely and already produced a runtime package with no embedded application files in the motivating consumer.

### Require users to reorder artifacts before each command

Rejected. A canonical declarative spec should not need mutation based on the command being run.

### First compatible artifact wins

Rejected. It fixes binary/package coexistence but leaves two binaries or package+template order-dependent. Exactly-one checking is a small, valuable safeguard.

### Add `--artifact` now

Implemented after the scoped-plan foundation landed. The flag chooses one compatible primary; it does not imply multi-output orchestration or target-specific support-artifact ownership.

### Generate every compatible artifact

Rejected. `build` and `generate` currently accept one output override and have singular lifecycle/reporting semantics. Multi-output orchestration is a different feature.

### Change all generator APIs to take a selected artifact

Rejected for this ticket. A scoped plan works with existing APIs and keeps target/source decisions consistent. Explicit generator parameters may be worthwhile in a larger future refactor.

### Add artifact dependency metadata

Rejected. The current consumer shares global static assets. There is no demonstrated need for target-specific support closure.

## 13. Intern implementation checklist

Before editing:

- [ ] Run `go test ./cmd/xgoja/... -count=1`.
- [ ] Run the reproduction script and read its log.
- [ ] Read `v2_plan_helpers.go`, both command `Run` methods, and generator artifact helpers.
- [ ] Confirm the branch is `task/improve-xgoja` and preserve unrelated changes.

While implementing:

- [ ] Keep command policy in `cmd/xgoja`, not `internal/generate`.
- [ ] Use explicit allow-lists for compatibility.
- [ ] Require exactly one compatible primary.
- [ ] Include IDs/types in errors.
- [ ] Shallow-copy; do not mutate the original plan.
- [ ] Update both artifact slices.
- [ ] Retain global support artifacts.
- [ ] Pass the scoped plan through every branch.

Before review:

- [ ] Run focused helper tests.
- [ ] Run command tests.
- [ ] Run all `cmd/xgoja/...` tests.
- [ ] Run full repository tests and vet.
- [ ] Verify build/generate in both orders.
- [ ] Verify support-first runtime target metadata.
- [ ] Verify source isolation and shared assets.
- [ ] Update the v2 reference.
- [ ] Keep the implementation local; no new public feature surface.

## 14. Review guide

Review in this order:

1. `cmd/xgoja/v2_plan_helpers.go`: compatibility, candidate counting, error contract, and shallow-copy correctness.
2. `cmd/xgoja/v2_plan_helpers_test.go`: policy matrix and non-mutation tests.
3. `cmd/xgoja/cmd_build.go`: selected/scoped plan reaches `WriteAllPlan`.
4. `cmd/xgoja/cmd_generate.go`: selected/scoped plan reaches template-data and all write modes.
5. `cmd/xgoja/root_test.go`: real command regressions.
6. `cmd/xgoja/doc/17-xgoja-v2-reference.md`: public behavior matches tests.

The central review invariant is:

> For one command invocation, command diagnostics, output selection, generated target metadata, artifact metadata, and embedded primary sources must all refer to the same selected primary artifact.

## 15. API and file references

### CLI and loading

- `cmd/xgoja/main.go`
- `cmd/xgoja/root.go:17-65` — root construction and command registration.
- `cmd/xgoja/v2_bridge.go:11-32` — v2 file loading and plan compilation.
- `cmd/xgoja/cmd_build.go:81-126` — build selection and generation path.
- `cmd/xgoja/cmd_generate.go:82-164` — generate selection and rendering path.

### Schema and plan

- `cmd/xgoja/internal/specv2/types.go:153-163` — `ArtifactSpec` API.
- `cmd/xgoja/internal/specv2/validate.go:218-242` — accepted types, IDs, and source references.
- `cmd/xgoja/internal/specv2/defaults.go:48-52` — default binary output.
- `cmd/xgoja/internal/plan/plan.go:18-37` — compiled plan and artifact wrappers.
- `cmd/xgoja/internal/plan/plan.go:46-82` — compilation pipeline.

### Existing defect

- `cmd/xgoja/v2_plan_helpers.go:9-35` — singular first-primary target selection.
- `cmd/xgoja/internal/generate/templates.go:165-190` — main template target derivation.
- `cmd/xgoja/internal/generate/templates.go:287-315` — independent target and embedded-path derivation.
- `cmd/xgoja/internal/generate/templates.go:343-395` — embedded runtime target/artifact metadata.
- `cmd/xgoja/internal/generate/plan.go:14-76` — binary/package generation entrypoints.
- `cmd/xgoja/internal/generate/plan.go:199-227` — executable-source and asset-source union.

### Tests and documentation

- `cmd/xgoja/root_test.go:52-220` — current build/generate command tests.
- `cmd/xgoja/internal/generate/generate_test.go:50-153` — runtime metadata and embedded-file tests.
- `cmd/xgoja/doc/17-xgoja-v2-reference.md:376-405` — artifact schema documentation.
- `cmd/xgoja/doc/17-xgoja-v2-reference.md:615-625` — current multi-artifact limitation.
- `examples/xgoja/14-generated-runtime-package/README.md` — generated runtime-package API.
- `examples/xgoja/14-generated-runtime-package/xgoja.yaml` — single-package example.

## 16. Open questions

No open question blocks implementation. Confirm these details during coding:

1. `adapter` and `cobra` are intentionally build-compatible because the generated main template has dedicated branches for them at `templates/main.go.tmpl:48-58`.
2. `dts` remains globally retained in the scoped plan even though `gen-dts` is its actual output command; this preserves metadata and current support-artifact semantics.
3. If a repository fixture without any primary artifacts exists outside `examples/xgoja`, decide whether to produce the new no-compatible-primary error or retain implicit binary behavior. The preferred design is a clear error because generation requires an explicit artifact contract.

## 17. Definition of done

The ticket is complete when:

- One spec containing one binary, one runtime package, and support artifacts works with both build and generate in either order.
- No command silently chooses among multiple compatible primaries.
- Errors name artifact IDs and types.
- Generated target metadata matches the command-selected primary.
- Unselected-primary JS/help sources are not embedded.
- Global static assets remain embedded.
- Existing single-primary examples and tests pass.
- The public v2 reference documents the actual behavior.
- `--artifact` selects a compatible primary; no dependency graph or multi-output orchestration was introduced.
