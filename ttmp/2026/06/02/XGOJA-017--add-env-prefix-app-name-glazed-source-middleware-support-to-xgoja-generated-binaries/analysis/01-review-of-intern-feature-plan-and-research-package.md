---
Title: Review of Intern Feature Plan and Research Package
Ticket: XGOJA-017
Status: active
Topics:
    - xgoja
    - glazed
    - configuration
    - middleware
    - design
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../glazed/pkg/cmds/sources/update.go
      Note: Review highlights env-prefix normalization behavior and shell-safety issue
    - Path: pkg/xgoja/app/glazed.go
      Note: Review identifies this as the key middleware chokepoint and recommends a narrower MVP
    - Path: ttmp/2026/06/02/XGOJA-017--add-env-prefix-app-name-glazed-source-middleware-support-to-xgoja-generated-binaries/design-doc/01-env-prefix-app-name-glazed-source-middleware-support-for-xgoja-generated-binaries.md
      Note: Primary plan being reviewed
    - Path: ttmp/2026/06/02/XGOJA-017--add-env-prefix-app-name-glazed-source-middleware-support-to-xgoja-generated-binaries/reference/02-research-logbook.md
      Note: Research package being reviewed
ExternalSources: []
Summary: Technical review of the XGOJA-017 design/research package, with coaching notes for the intern who produced it.
LastUpdated: 2026-06-02T12:03:44.384504231-04:00
WhatFor: Use this when reviewing or refining the XGOJA-017 implementation plan before coding.
WhenToUse: Before implementing env-prefix/app-name/config/profile middleware support in xgoja, and as coaching material for future research/design tasks.
---


# Review of Intern Feature Plan and Research Package

## Executive Summary

Your research package is strong. You found the right architectural seam (`pkg/xgoja/app/glazed.go`), traced the build-time and runtime paths, compared xgoja with Glazed and Pinocchio, and produced an implementation plan that is detailed enough for a new engineer to start from. The design document is useful because it explains not just *what* to change, but *why* those files matter.

The main improvement is that you sometimes moved from evidence to proposed API too quickly. The plan correctly identifies the problem, but it should be more cautious about public YAML shape, env-prefix normalization, config/profile semantics, and template complexity. Before coding, the next pass should convert the proposal into a smaller, testable MVP and explicitly validate assumptions with tiny probes.

The most important correction: **do not blindly default `envPrefix` to `strings.ToUpper(appName)` if `appName` may contain hyphens.** Glazed's `updateFromEnv` uppercases the prefix but does not replace `-` with `_` in the prefix. An `appName` of `my-app` would produce env vars like `MY-APP_FIELD`, which are not normal shell variable names. The implementation must normalize or require a separate valid `envPrefix`.

---

## Overall Grade

**Assessment:** Good research, good architecture mapping, good first design. Needs sharper API discipline before implementation.

| Area | Rating | Notes |
|---|---:|---|
| Codebase orientation | Strong | You found the critical flow from YAML spec to generated binary. |
| Evidence gathering | Strong | You read the right xgoja, Glazed, Pinocchio, and Geppetto files. |
| Problem statement | Strong | The missing env/config/profile behavior is clearly described. |
| Proposed API | Medium | Useful starting point, but too broad for first implementation. |
| Risk analysis | Medium | Good risks listed, but missed env-prefix normalization and target-specific behavior details. |
| Implementation sequencing | Good | Phases are logical, but Phase 1 should include assumption probes. |
| Intern-readability | Strong | Clear prose, tables, diagrams, and pseudocode. |
| Restraint / minimality | Medium-low | The plan expands into config, profiles, and arbitrary middlewares at once. |

---

## What You Did Well

### 1. You found the right chokepoint

The best part of the analysis is the identification of `pkg/xgoja/app/glazed.go` as the central chokepoint:

```go
func buildGlazedCobraCommand(command cmds.Command) (*cobra.Command, error) {
    return cli.BuildCobraCommand(command,
        cli.WithParserConfig(cli.CobraParserConfig{
            ShortHelpSections: []string{schema.DefaultSlug},
            MiddlewaresFunc:   cli.CobraCommandDefaultMiddlewares,
        }),
    )
}
```

This is exactly the right starting point. It explains why generated xgoja commands currently only read flags, args, and defaults. It also avoids a common beginner mistake: trying to change every command individually instead of changing the shared builder.

**Keep doing this:** when analyzing a feature, look for the smallest shared abstraction that explains the behavior.

### 2. You traced the full lifecycle

You followed the lifecycle:

```text
xgoja.yaml
  -> buildspec.Spec
  -> generate.WriteAll
  -> main.go.tmpl
  -> app.NewRootCommand / NewHostWithOptions
  -> Host.AttachDefaultCommands
  -> buildGlazedCobraCommand
  -> Glazed middleware parsing
```

That chain is the backbone of the feature. A future implementer can now see which changes are build-time schema changes, which are generated-code changes, and which are runtime host changes.

**Why this matters:** code generation features often fail when people only understand the generated output or only understand the generator. You covered both.

### 3. You compared against the right reference implementation

Pinocchio was the correct comparison point because it has:

- A stable `AppName`.
- A stable `EnvPrefix`.
- A `ConfigPlanBuilder`.
- A custom `MiddlewaresFunc`.
- Profile/bootstrap machinery.

The key file, `pinocchio/pkg/cmds/cobra.go`, is the right canonical reference for the middleware chain. You correctly extracted this pattern:

```go
sources.FromCobra(cmd, fields.WithSource("cobra"))
sources.FromArgs(args, fields.WithSource("arguments"))
sources.FromEnv(cfg.EnvPrefix, fields.WithSource("env"))
sources.FromConfigPlanBuilder(...)
sources.FromDefaults(fields.WithSource(fields.SourceDefaults))
```

### 4. You made precedence explicit

You read `glazed/pkg/cmds/sources/middlewares.go` and explained the reversal behavior. That is important. Without understanding middleware execution order, it is easy to accidentally make config override flags or defaults override env.

The design correctly communicates the intended effective precedence:

```text
Defaults < config files < environment variables < CLI flags
```

### 5. The research logbook is valuable

The logbook is useful because it tells the next person which resources mattered and which did not. This is the kind of artifact that saves hours later.

Especially good entries:

- `pkg/xgoja/app/glazed.go` — identifies the critical chokepoint.
- `glazed/pkg/cli/cobra-parser.go` — records the built-in parser path.
- `pinocchio/pkg/cmds/cobra.go` — records the reference implementation.
- `glazed/pkg/cmds/sources/update.go` — records env parsing mechanics.

---

## What Was Weak or Needs Correction

### 1. The proposed YAML API is too broad for a first implementation

The proposal includes all of these at once:

```yaml
appName: my-app
envPrefix: MY_APP
config:
  enabled: true
  layers: [system, xdg, home, git-root, cwd]
profiles:
  enabled: true
middlewares:
  - source: env
  - source: config
  - source: profiles
```

This is ambitious. It is good as a long-term vision, but risky as a first implementation. The feature request says:

> add env prefix / app name / potentially glazed source middleware support

The word **potentially** matters. It means arbitrary source middleware support is not necessarily required in the first cut.

A safer MVP would be:

```yaml
appName: my-app
envPrefix: MY_APP
config:
  enabled: true
  layers:
    - xdg
    - home
    - cwd
    - explicit
```

Then add profiles and arbitrary middleware after env/config are proven.

**What to do next time:** separate "MVP" from "future extensibility." Do not put them in the same required implementation path.

### 2. The env-prefix default is probably wrong for hyphenated app names

The design says:

> `envPrefix` defaults to `UPPER(appName)` when `appName` is set.

This copies Glazed's built-in behavior from `CobraParserConfig.AppName`, but you should have checked whether the prefix is shell-safe.

In `glazed/pkg/cmds/sources/update.go`, `updateFromEnv` does this:

```go
envKey := strings.ToUpper(strings.ReplaceAll(base, "-", "_"))
if prefix != "" {
    envKey = strings.ToUpper(prefix) + "_" + envKey
}
```

Notice the asymmetry:

- The field/section part replaces `-` with `_`.
- The prefix only gets `strings.ToUpper(prefix)`.
- Therefore `appName: my-app` becomes prefix `MY-APP`, not `MY_APP`.

That produces env names like:

```text
MY-APP_DEFAULT_RUNTIME
```

This is awkward or invalid in normal shell assignment syntax.

**Correct design guidance:**

- `appName` is a human/application identity and may contain hyphens.
- `envPrefix` is an environment-variable namespace and should be validated as `[A-Z][A-Z0-9_]*`.
- If `envPrefix` is omitted, derive it with a normalization function:

```go
func DefaultEnvPrefix(appName string) string {
    s := strings.TrimSpace(appName)
    s = strings.ReplaceAll(s, "-", "_")
    s = strings.ReplaceAll(s, ".", "_")
    s = strings.ToUpper(s)
    s = collapseRepeatedUnderscores(s)
    return strings.Trim(s, "_")
}
```

Do **not** rely on Glazed's current `strings.ToUpper(appName)` if xgoja wants friendly generated binaries.

### 3. You proposed generated closures too early

The design leans toward emitting Go closures directly in `main.go.tmpl`:

```go
middlewaresFunc := func(parsedCommandSections *values.Values, cmd *cobra.Command, args []string) ([]cmd_sources.Middleware, error) {
    // generated source
}
```

That may work, but it is template-heavy. Generated Go templates become hard to maintain when they contain conditional imports, nested closures, and repeated source snippets.

A cleaner alternative is to add runtime helper code in `pkg/xgoja/app`, for example:

```go
func MiddlewaresFromSpec(spec *Spec) cli.CobraMiddlewaresFunc
```

Then generated `main.go` only needs:

```go
root, err := app.NewRootCommand(app.Options{
    Providers: registry,
    SpecJSON: embeddedSpecJSON,
})
```

And `NewRootCommand` can do:

```go
host := NewHostWithOptions(providers, spec, HostOptions{
    MiddlewaresFunc: MiddlewaresFromSpec(spec),
})
```

This keeps policy in tested Go code instead of generated templates.

**What to keep in mind:** prefer templates for structural generation (imports, providers, embeds). Prefer ordinary Go packages for behavioral policy.

### 4. You under-specified the difference between `target.kind: xgoja`, `cobra`, and `adapter`

You noticed the target kinds, but the plan should say more about how middleware policy behaves for each one.

Current generated modes:

1. **`target.kind: xgoja`**
   - xgoja creates the root command.
   - xgoja controls all command building.
   - Middleware support is straightforward.

2. **`target.kind: cobra`**
   - User-provided package creates the root command.
   - xgoja attaches default commands to that root.
   - Middleware support applies only to xgoja-attached commands unless we also modify the target root's own commands.

3. **`target.kind: adapter`**
   - User-provided adapter builds root from `Host`.
   - Middleware support depends on whether adapter uses `host.AttachDefaultCommands` or builds commands manually.

The review question a future implementer must answer is:

> Does `appName/envPrefix/config` apply only to xgoja-provided commands, or also to target-provided commands?

This matters because `target.kind: cobra` may already have its own middleware conventions.

**Better documentation:** add a table in the design doc explaining behavior per target kind.

### 5. Profile support was not investigated deeply enough

You read `glazed/pkg/cmds/sources/profiles.go` and Pinocchio's profile bootstrap, but the proposal should be more cautious.

There are at least three meanings of "profile":

1. **Glazed profile file:** a YAML map of section values.
2. **Geppetto engine profile:** a model/provider configuration profile.
3. **xgoja runtime profile:** a named runtime under `runtimes:`.

xgoja already uses the word "runtime profile" in several places. If you add `profiles:` to xgoja.yaml, readers may confuse it with `runtimes:`.

Example confusion:

```yaml
runtimes:
  prod:
    modules: [...]

profiles:
  enabled: true
  defaultProfile: prod
```

Does `prod` mean a Glazed config profile or an xgoja runtime profile? They are different concepts.

**Recommendation:** defer profile support or name it more explicitly:

```yaml
parameterProfiles:
  enabled: true
```

or:

```yaml
glazedProfiles:
  enabled: true
```

### 6. The config file structure needs a clearer contract

The design says config files can use Glazed's standard section map format, but it should give explicit examples tied to xgoja commands.

For example, if the command is `eval` and its settings struct is:

```go
type evalSettings struct {
    Source  string `glazed:"source"`
    Runtime string `glazed:"runtime"`
}
```

What config file sets the default runtime?

Possibility A:

```yaml
default:
  runtime: main
```

Possibility B:

```yaml
eval:
  runtime: main
```

Possibility C:

```yaml
command:
  runtime: main
```

The answer depends on section slugs and command descriptions. The design should point to the exact section slug for built-in commands and provider sections.

**Next research task:** inspect generated command descriptions and confirm the section slugs for `eval`, `run`, `repl`, `modules`, JS verbs, and provider command sections.

### 7. You did not propose a concrete first test fixture

The plan says to add integration tests, but it should define a minimal test fixture that exercises the feature without depending on complex providers.

A good fixture would be:

```yaml
name: env-config-fixture
appName: env-config-fixture
envPrefix: XGOJA_FIXTURE
config:
  enabled: true
  layers: [cwd, explicit]

target:
  kind: xgoja
  output: dist/env-config-fixture
packages:
  - id: fixture
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/testprovider
runtimes:
  main:
    modules:
      - package: fixture
        name: hello
        as: hello
  alternate:
    modules:
      - package: fixture
        name: hello
        as: hello
commands:
  eval:
    enabled: true
    runtime: main
```

Then test only the `eval.runtime` setting:

```bash
# defaults to main
./dist/env-config-fixture eval 'require("hello").greet("x")'

# config overrides default
cat > config.yaml <<'EOF'
default:
  runtime: alternate
EOF
./dist/env-config-fixture eval '...'

# env overrides config
XGOJA_FIXTURE_RUNTIME=main ./dist/env-config-fixture eval '...'

# flag overrides env
XGOJA_FIXTURE_RUNTIME=alternate ./dist/env-config-fixture eval --runtime main '...'
```

This may reveal the exact section slug issue mentioned above.

### 8. You should have looked at validation tests earlier

The design says Phase 1 should update `validate.go`, `validate_test.go`, and `load_test.go`, but the research did not actually inspect those tests.

That is a gap. Before implementing a schema change, you should inspect existing validation tests to understand:

- Error message style.
- Table-driven test patterns.
- Existing validation helpers.
- How strict the validator is about unknown or future fields.

**What you should have looked at:**

- `cmd/xgoja/internal/buildspec/validate.go`
- `cmd/xgoja/internal/buildspec/validate_test.go`
- `cmd/xgoja/internal/buildspec/load.go`
- `cmd/xgoja/internal/buildspec/load_test.go`

You read `spec.go`, but not the validator implementation deeply enough.

---

## What You Should Have Known or Checked

### 1. Public YAML shape is an API

Once `xgoja.yaml` supports a field, users will write it in repositories. Renaming it later becomes a migration problem. Therefore, public YAML fields require more caution than internal Go types.

Before proposing top-level fields, ask:

- Is the name clear to a user who does not know Glazed internals?
- Will it conflict with existing or future xgoja concepts?
- Can it be validated cleanly?
- Does it compose with provider-defined commands?
- Is it stable enough to document?

### 2. "App name" and "env prefix" are not the same thing

They overlap but have different constraints:

| Concept | Purpose | Example | Valid characters |
|---|---|---|---|
| `name` | xgoja binary/spec name | `discord-bot` | user-facing, can contain hyphen |
| `appName` | config/logging application identity | `discord-bot` | usually path-ish/name-ish |
| `envPrefix` | environment variable namespace | `DISCORD_BOT` | shell-safe uppercase identifier |

Do not collapse them without normalization.

### 3. Generated code should be boring

Generated code should be easy to inspect, but not full of complex logic. Ideally:

- The template imports providers and embeds files.
- Runtime packages implement behavior.
- Tests cover behavior in normal Go code.

If the template starts generating complex `ConfigPlanBuilder` code, debugging becomes harder.

### 4. Target modes change ownership

In generated-code systems, always ask: who owns the root command?

- xgoja-owned root: xgoja can enforce middleware behavior.
- target-owned root: xgoja can only control commands it attaches.
- adapter-owned root: behavior depends on adapter code.

This affects user expectations and documentation.

### 5. Config semantics need concrete examples

Do not describe config support only abstractly. Always show exact files:

```yaml
# ~/.config/my-app/config.yaml
section-slug:
  field-name: value
```

Then show the corresponding CLI flag and env var:

```bash
my-app command --field-name value
MY_APP_SECTION_FIELD_NAME=value my-app command
```

If you cannot write the example confidently, the design is not yet precise enough.

---

## What They Should Have Looked At

This list is not a criticism of the work already done; it is the next research checklist before coding.

### 1. `cmd/xgoja/internal/buildspec/validate.go`

Reason: The new fields require validation, and the existing validator likely has conventions for error aggregation and reporting.

Questions to answer:

- How are validation errors represented?
- Does validation stop early or collect all issues?
- How are unknown enum values reported?
- Where should `envPrefix` validation live?

### 2. `cmd/xgoja/internal/buildspec/validate_test.go`

Reason: Tests show style and edge cases.

Questions to answer:

- Are tests table-driven?
- Do they assert exact error strings?
- Are there helper specs you can extend?

### 3. `cmd/xgoja/internal/buildspec/load.go` and `load_test.go`

Reason: New YAML fields must round-trip through load and report paths.

Questions to answer:

- Is unknown YAML allowed or rejected?
- Is `BaseDir` set during loading?
- Does `LoadFile` normalize paths or leave raw YAML values?

### 4. `pkg/xgoja/app/command_providers.go`

Reason: Provider commands may have their own sections and could be most affected by env/config support.

Questions to answer:

- Do provider commands use `buildGlazedCobraCommand`?
- Are command sections modified with runtime module sections?
- Does middleware config need to be passed through provider command builders?

### 5. `pkg/xgoja/app/run.go` and `tui.go`

Reason: Built-in commands may have command-specific section slugs or additional runtime/module sections.

Questions to answer:

- Which section slug controls `runtime`, `keep-alive`, etc.?
- How do module-provided sections get attached?
- Are there sensitive fields that env/config should parse?

### 6. Glazed tests for env/config parsing

Useful files to inspect:

- `glazed/pkg/cli/cobra_parser_config_test.go`
- `glazed/pkg/cmds/sources/config_files_test.go`
- `glazed/pkg/cmds/sources/custom-profiles_test.go`

Reason: These tests define the real expected behavior more precisely than prose docs.

Questions to answer:

- How exactly is `AppName` transformed into env vars?
- Are hyphenated app names tested?
- How does config precedence behave in assertions?

---

## Recommended Revision to the Feature Plan

Before implementation, revise the plan into three tiers.

### Tier 1: MVP — app name + env prefix only

Goal: Let generated binaries read command fields from environment variables.

YAML:

```yaml
appName: my-app
envPrefix: MY_APP
```

Behavior:

- If `envPrefix` is set, generated commands use `sources.FromEnv(envPrefix)`.
- If `envPrefix` is omitted but `appName` is set, derive shell-safe prefix from `appName`.
- CLI flags still override env.
- Existing specs behave exactly as before.

Implementation:

- Add fields to buildspec and runtime spec.
- Add middleware factory to `Host`.
- Prefer `app.MiddlewaresFromSpec(spec)` over generated closures.
- Add tests for env precedence.

### Tier 2: Config files

Goal: Let generated binaries read section maps from config files.

YAML:

```yaml
config:
  enabled: true
  layers: [xdg, home, cwd, explicit]
  fileName: config.yaml
```

Behavior:

- Config values load below env and flags.
- Missing discovered files are skipped.
- Explicit config file errors if missing.
- Config file format is documented with real xgoja command examples.

Implementation:

- Add `ConfigSpec`.
- Add `ConfigPlanFromSpec` helper.
- Add tests for config precedence.

### Tier 3: Profiles and arbitrary source middleware

Goal: Advanced users can load named profiles or opt into additional sources.

YAML:

```yaml
glazedProfiles:
  enabled: true
  appName: my-app
  defaultProfile: default
```

Arbitrary middleware should not be added until there is a concrete use case. It is harder to design safely because middleware order is behavior.

---

## Coaching Notes for Next Time

### Start with the smallest useful behavior

When a task says "potentially support X," do not turn X into a first-class requirement immediately. Write:

- MVP
- Follow-up
- Future exploration

This keeps the plan realistic.

### Validate copied behavior before adopting it

You copied Glazed's `strings.ToUpper(appName)` behavior as a proposed default. The copy was understandable, but it missed shell-safety. When copying a pattern, ask:

- Does the source project have the same constraints?
- Is the source behavior ideal or just historical?
- Are there known edge cases?

### Prefer helper functions over generated snippets

Generated code is powerful, but every conditional branch in a template increases maintenance cost. If behavior can live in normal Go code, put it there.

Bad first instinct:

```gotemplate
{{ if .Config.Enabled }}
// emit many lines of config plan Go code
{{ end }}
```

Better first instinct:

```go
middlewaresFunc := app.MiddlewaresFromSpec(spec)
```

### Always include one concrete config example

A design about config parsing is incomplete until it shows:

- YAML config file
- equivalent CLI flag
- equivalent env var
- resulting parsed value

### Explicitly mark unresolved semantics

If you are not sure whether config applies to target-owned Cobra commands, write that uncertainty as a decision point. Do not let it hide inside implementation phases.

---

## Specific Review Comments on the Existing Documents

### Design doc

**Good:**

- Excellent architecture trace.
- Strong gap analysis table.
- Useful decision records.
- Clear file-level implementation plan.

**Needs improvement:**

- Split MVP vs future work.
- Fix env-prefix derivation.
- Add target-kind behavior table.
- Add concrete config file examples.
- Prefer helper package/function over generated closures.

### Diary

**Good:**

- Captures commands and research sequence.
- Records what was tricky.
- Good final handoff summary.

**Needs improvement:**

- It states some upload and doctor results as if they were known before they happened in earlier draft sections. Keep diary entries strictly chronological and update them after the fact.
- Add exact error for the first failed `remarquee upload bundle` with spaces in bundle name, then note the successful simpler name.

### Research logbook

**Good:**

- Very useful list of resources.
- Clearly identifies what each file contributed.
- Summary table is helpful.

**Needs improvement:**

- Add validator/test files after they are read.
- Mark `glazed/pkg/cmds/sources/update.go` as more important because it reveals the env-prefix normalization issue.
- Avoid saying "no external docs needed" too strongly; source code was sufficient for this pass, but public docs may still matter for user-facing terminology.

---

## Final Recommendation

Do not start coding the full original proposal. Start with a narrower implementation spike:

1. Add `appName` and `envPrefix` to both spec structs.
2. Add `MiddlewaresFromSpec(spec)` in `pkg/xgoja/app`.
3. Wire `HostOptions.MiddlewaresFunc` through built-in commands.
4. Add one env precedence test.
5. Only then add config file support.

This approach will prove the architecture without committing to too much YAML surface area. Once env support works end-to-end, config support will be much safer to add because the Host/middleware plumbing will already exist.

The intern's work is a good foundation. The next step is to sharpen it into a smaller, stricter, more easily testable implementation plan.
