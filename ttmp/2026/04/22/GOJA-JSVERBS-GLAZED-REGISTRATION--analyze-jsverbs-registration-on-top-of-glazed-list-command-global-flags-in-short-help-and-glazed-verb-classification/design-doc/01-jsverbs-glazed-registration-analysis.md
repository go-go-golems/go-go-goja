---
Title: Jsverbs Glazed Registration Analysis
Ticket: GOJA-JSVERBS-GLAZED-REGISTRATION
Status: active
Topics:
    - jsverbs
    - glazed
    - cli
    - help-system
    - command-registration
    - short-help
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../glazed/pkg/cmds/schema/cobra_flag_groups.go
      Note: Computes FlagGroupUsage and defines DefaultSlug vs GlobalDefaultSlug
    - Path: ../../../../../../../glazed/pkg/help/cmd/cobra.go
      Note: Help rendering engine with shortHelpSections filtering logic
    - Path: cmd/jsverbs-example/main.go
      Note: Host application that wires jsverbs into Cobra; contains the plain-text list command and ShortHelpSections config
    - Path: pkg/jsverbs/binding.go
      Note: Builds VerbBindingPlan mapping JS parameters to parsed CLI values
    - Path: pkg/jsverbs/command.go
      Note: Builds CommandDescription and selects Command vs WriterCommand based on OutputMode
    - Path: pkg/jsverbs/runtime.go
      Note: Creates Goja runtime
    - Path: pkg/jsverbs/scan.go
      Note: Tree-sitter based scanner that extracts __verb__
ExternalSources: []
Summary: 'Deep-dive analysis of how go-go-goja registers JavaScript verbs (jsverbs) as Glazed commands, with three identified gaps: (1) the `list` discovery command is plain text instead of structured Glazed output, (2) global flags are missing from short help for Glazed-wrapped jsverbs, and (3) clarification on which jsverbs become GlazeCommand vs WriterCommand.'
LastUpdated: 2026-04-22T17:00:00-04:00
WhatFor: Onboarding document for engineers new to the jsverbs + glazed integration. Explains the scan-to-command pipeline, the Glazed command lifecycle, the help-system rendering path, and concrete files to read.
WhenToUse: When you need to modify how jsverbs are discovered, registered, rendered, or when adding new host-provided sections or output modes.
---








# Jsverbs Glazed Registration Analysis

## Executive Summary

`go-go-goja` lets you write CLI commands in JavaScript. It does this by:

1. **Scanning** `.js`/`.cjs` files for top-level functions annotated with `__verb__`, `__section__`, and `__package__` metadata.
2. **Building** a `Registry` of `VerbSpec` objects that describe name, parents, fields, output mode, and doc strings.
3. **Wrapping** each `VerbSpec` in a Glazed `Command` (for structured/table output) or `WriterCommand` (for plain text output).
4. **Registering** those wrapped commands under a Cobra root via `cli.AddCommandsToRootCommand`.

This document explains every step of that pipeline, then identifies three concrete gaps in the current `jsverbs-example` host application:

| # | Gap | Evidence | Severity |
|---|-----|----------|----------|
| 1 | `jsverbs-example list` prints plain text instead of structured Glazed rows | `cmd/jsverbs-example/main.go:57-66` — uses a raw `cobra.Command` with `fmt.Fprintln` | Medium |
| 2 | Global flags (`--dir`, `--log-level`, …) are hidden from short help for Glazed-wrapped jsverbs | `cmd/jsverbs-example/main.go:78-82` — `ShortHelpSections: []string{schema.DefaultSlug}` filters out inherited flag groups | Medium |
| 3 | It is unclear which jsverbs become `GlazeCommand` vs `WriterCommand` | `pkg/jsverbs/command.go:77-93` — decision is based on `verb.OutputMode`, but this is not documented for host authors | Low |

---

## 1. How jsverbs work: the scan-to-command pipeline

### 1.1 Discovery — `pkg/jsverbs/scan.go`

`jsverbs.ScanDir(root)` walks a directory tree and parses every `.js`/`.cjs` file with Tree-sitter. For each file it extracts:

- **Package metadata** from `__package__({ name, short, long, parents, tags })`
- **Section schemas** from `__section__(slug, { title, description, fields })`
- **Verb metadata** from `__verb__(functionName, { short, long, output, parents, fields, sections })`
- **Function signatures** from standard `function` declarations and arrow-function variable declarators
- **Doc templates** from `` doc`...` `` calls with YAML frontmatter

The scanner stores everything in a `Registry`:

```go
// pkg/jsverbs/model.go
 type Registry struct {
     RootDir            string
     Files              []*FileSpec
     Diagnostics        []Diagnostic
     SharedSections     map[string]*SectionSpec
     SharedSectionOrder []string
     verbs              []*VerbSpec
     verbsByKey         map[string]*VerbSpec
     filesByModule      map[string]*FileSpec
     options            ScanOptions
 }
```

**Key file:** `pkg/jsverbs/scan.go` (lines 1–725)  
**Key types:** `Registry`, `FileSpec`, `VerbSpec`, `SectionSpec`, `FieldSpec` in `pkg/jsverbs/model.go`

### 1.2 From verb to binding plan — `pkg/jsverbs/binding.go`

Before a verb can be executed, `buildVerbBindingPlan` reconciles the JavaScript function’s parameters with the verb’s declared fields:

```go
// pkg/jsverbs/binding.go
 type VerbBindingPlan struct {
     Verb               *VerbSpec
     Parameters         []ParameterBinding   // one per JS function param
     ExtraFields        []ExtraFieldBinding  // declared fields with no param counterpart
     ReferencedSections []string             // sections that must exist
 }
```

Each `ParameterBinding` has a `Mode`:

- `BindingModePositional` — the param receives a single parsed flag/argument value.
- `BindingModeSection` — the param receives an entire section as an object (e.g. `bind: "filters"`).
- `BindingModeAll` — the param receives the flat map of all parsed values (e.g. `bind: "all"`).
- `BindingModeContext` — the param receives a host-provided context object with metadata (e.g. `bind: "context"`).

If a JS parameter is an object/array pattern and no `bind` or `type` is provided, the scanner emits an error because Glazed cannot infer a flat CLI flag from a destructured parameter.

**Key file:** `pkg/jsverbs/binding.go` (lines 1–191)

### 1.3 From binding plan to Glazed description — `pkg/jsverbs/command.go`

`Registry.buildDescription` converts a `VerbSpec` + `VerbBindingPlan` into a `cmds.CommandDescription`, which is the Glazed-native representation of a CLI command schema.

```go
// pkg/jsverbs/command.go
 description := cmds.NewCommandDescription(
     verb.Name,
     cmds.WithShort(verb.Short),
     cmds.WithLong(verb.Long),
     cmds.WithParents(verb.Parents...),
     cmds.WithSource("jsverbs:"+verb.SourceRef()),
 )
```

The description’s `Schema` is populated with one or more `schema.SectionImpl` objects:

- The **default section** (`schema.DefaultSlug == "default"`) holds positional arguments.
- **File-local sections** come from `__section__` calls inside the same `.js` file.
- **Shared sections** come from `registry.SharedSections` (registered by the host app, e.g. the example program).
- Fields are built via `buildFieldDefinition`, which maps jsverbs types (`string`, `bool`, `int`, `stringList`, `choice`, …) to Glazed `fields.Type` values.

**Key file:** `pkg/jsverbs/command.go` (lines 83–165)

### 1.4 From description to executable command — `pkg/jsverbs/command.go`

`Registry.CommandForVerbWithInvoker` picks the concrete wrapper type based on `verb.OutputMode`:

```go
// pkg/jsverbs/command.go:77-93
 switch verb.OutputMode {
 case OutputModeGlaze:
     return &Command{
         CommandDescription: description,
         registry:           r,
         verb:               verb,
         invoker:            invoker,
     }, nil
 case OutputModeText:
     return &WriterCommand{
         CommandDescription: description,
         registry:           r,
         verb:               verb,
         invoker:            invoker,
     }, nil
 default:
     return nil, fmt.Errorf("... unsupported output mode %q", verb.OutputMode)
 }
```

- `Command` implements `cmds.GlazeCommand` → its `RunIntoGlazeProcessor` feeds rows into Glazed’s table/middleware pipeline.
- `WriterCommand` implements `cmds.WriterCommand` → its `RunIntoWriter` writes plain text directly to `os.Stdout`.

The default output mode is `OutputModeGlaze` unless the JS author sets `output: "text"` in `__verb__` metadata.

### 1.5 Registration with Cobra — `cmd/jsverbs-example/main.go`

The host application (`jsverbs-example`) wires everything together:

```go
// cmd/jsverbs-example/main.go:44-82
 registry, err := jsverbs.ScanDir(dir)
 commands, err := registry.Commands()

 root := &cobra.Command{...}

 // 1. Plain Cobra "list" subcommand (NOT a Glazed command)
 root.AddCommand(&cobra.Command{
     Use:   "list",
     Short: "List discovered JS verbs",
     Run: func(cmd *cobra.Command, args []string) {
         // ... plain text fmt.Fprintln
     },
 })

 // 2. Glazed-wrapped jsverbs
 if err := cli.AddCommandsToRootCommand(
     root,
     commands,
     nil,
     cli.WithParserConfig(cli.CobraParserConfig{
         ShortHelpSections: []string{schema.DefaultSlug},
         MiddlewaresFunc:   cli.CobraCommandDefaultMiddlewares,
     }),
 ); err != nil { ... }
```

`cli.AddCommandsToRootCommand` (from `glazed/pkg/cli/cobra.go`) does three things for each `cmds.Command`:

1. **Builds** a `*cobra.Command` via `BuildCobraCommandFromCommand`.
2. **Creates** a `CobraParser` that maps Glazed `schema.Section` objects to Cobra flags/persistent flags.
3. **Attaches** the parser to the cobra command so that at runtime flags are parsed back into `*values.Values` and passed to the jsverb.

If a verb has `Parents: ["basics"]`, Glazed’s `findOrCreateParentCommand` creates a `basics` cobra intermediate command and hangs the leaf under it.

**Key file:** `cmd/jsverbs-example/main.go`  
**Key Glazed file:** `glazed/pkg/cli/cobra.go` (`AddCommandsToRootCommand`, `BuildCobraCommandFromCommand`)

---

## 2. How the Glazed help system renders short vs long help

Glazed replaces Cobra’s default help renderer with a markdown-based system in `pkg/help/cmd/cobra.go`. When a user runs `--help`, the renderer builds `FlagGroupUsage` objects from:

- **Local flags** (flags added directly to the command)
- **Inherited flags** (persistent flags from parent commands)

Flag groups are determined by Cobra annotations of the form `glazed:flag-group:<id>:<name>`. Flags that belong to no group end up in:

- Local → `schema.DefaultSlug` (displayed as **Flags** or **Arguments**)
- Inherited → `schema.GlobalDefaultSlug` (displayed as **Global flags**)

### 2.1 Short-help filtering

In `pkg/help/cmd/cobra.go:231-249`, if `cmd.Annotations["shortHelpSections"]` is set and `--long-help` is NOT used, the renderer discards every flag group whose slug is not in the annotation list.

In `jsverbs-example`, the parser config sets:

```go
ShortHelpSections: []string{schema.DefaultSlug},  // == "default"
```

This means:

- **Local** default-section flags are shown in short help ✅
- **Inherited** global-default flags are hidden in short help ❌
- The `--long-help` flag itself is special-cased: `ComputeCommandFlagGroupUsage` moves it from inherited to local default so it always appears.

**Why this matters:** `jsverbs-example` adds `--dir`, `--log-level`, `--log-format`, and all logstash flags as **Cobra persistent flags** via `logging.AddLoggingSectionToRootCommand`. Because they are added as raw Cobra flags (not through Glazed sections), they land in `GlobalDefaultSlug` and are filtered out of short help.

---

## 3. Gap analysis

### 3.1 Gap 1 — `list` should be a Glazed command with structured output

**Current behavior**

```go
// cmd/jsverbs-example/main.go:57-66
root.AddCommand(&cobra.Command{
    Use:   "list",
    Short: "List discovered JS verbs",
    Run: func(cmd *cobra.Command, args []string) {
        paths := make([]string, 0, len(registry.Verbs()))
        for _, verb := range registry.Verbs() {
            paths = append(paths, fmt.Sprintf("%s\t%s", verb.FullPath(), verb.SourceRef()))
        }
        sort.Strings(paths)
        fmt.Fprintln(cmd.OutOrStdout(), strings.Join(paths, "\n"))
    },
})
```

This outputs tab-separated plain text. It cannot be piped through `jq`, filtered with `--fields`, or rendered as JSON/YAML/CSV via Glazed’s output pipeline.

**Expected behavior**

`list` should be a `cmds.GlazeCommand` that emits rows with at minimum the columns:

- `path` — the verb’s full command path (e.g. `basics echo`)
- `source` — the source ref (e.g. `basics.js#echo`)
- `output_mode` — `glaze` or `text`
- `parents` — parent path slice

Because it is a host-provided command (not a scanned jsverb), it should be built by hand using `cmds.NewCommandDescription` + `fields.New`, then passed to `cli.AddCommandsToRootCommand` alongside the scanned verbs.

**Pseudocode for the fix**

```go
listDesc := cmds.NewCommandDescription(
    "list",
    cmds.WithShort("List discovered JS verbs"),
    cmds.WithLong("Emit all discovered jsverbs as a structured table."),
)
// No arguments needed; just flags for filtering if desired.

listCmd := &listGlazeCommand{
    CommandDescription: listDesc,
    registry:           registry,
}

// Pass it into AddCommandsToRootCommand as the first element
allCommands := append([]cmds.Command{listCmd}, commands...)
cli.AddCommandsToRootCommand(root, allCommands, nil, opts...)
```

Where `listGlazeCommand` implements:

```go
func (c *listGlazeCommand) RunIntoGlazeProcessor(
    ctx context.Context,
    parsedValues *values.Values,
    gp middlewares.Processor,
) error {
    for _, verb := range c.registry.Verbs() {
        row := types.NewRow(
            types.MRP("path", verb.FullPath()),
            types.MRP("source", verb.SourceRef()),
            types.MRP("output_mode", verb.OutputMode),
            types.MRP("parents", strings.Join(verb.Parents, " ")),
        )
        if err := gp.AddRow(ctx, row); err != nil {
            return err
        }
    }
    return nil
}
```

**Reference example:** `glazed/pkg/cli/cobra.go` already contains many examples of `cmds.GlazeCommand` implementations in the test suite.

---

### 3.2 Gap 2 — global flags missing from short help for Glazed jsverbs

**Current behavior**

Running `jsverbs-example basics echo --help` shows:

```
## Arguments:
  -h, --help    help for echo
 --long-help    Show long help
```

The `--dir`, `--log-level`, and other root persistent flags are hidden until `--help --long-help` is used.

By contrast, `jsverbs-example list --help` (a plain Cobra command) shows:

```
## Flags:
  -h, --help    help for list
 --long-help    Show long help

## Global flags:
  -d, --dir    Directory scanned ...
  --log-level  Log level ...
  ...
```

**Root cause**

`jsverbs-example` passes:

```go
cli.WithParserConfig(cli.CobraParserConfig{
    ShortHelpSections: []string{schema.DefaultSlug},
    ...
})
```

The `shortHelpSections` annotation is `"default"`. When the help renderer filters flag groups, it keeps only groups whose slug is `"default"`. The inherited flags live in `"global-default"`, so they are discarded.

**Why `list` works:** `list` is a plain `cobra.Command`. It does not use Glazed’s parser, so it has no `shortHelpSections` annotation. Cobra’s native template (embedded in `cobra-short-help.tmpl`) renders `InheritedFlags` unconditionally.

**Fix options**

| Option | Change | Tradeoff |
|--------|--------|----------|
| A | Add `"global-default"` to `ShortHelpSections` | Simple, but shows ALL global flags in short help for every jsverb |
| B | Convert root persistent flags into a Glazed section with a known slug, then add that slug to `ShortHelpSections` | More idiomatic; keeps flag grouping consistent |
| C | Remove `ShortHelpSections` restriction entirely | Short help becomes as long as long help; defeats the purpose |

**Recommended fix: Option B**

Instead of adding raw Cobra persistent flags in `main.go`, register a proper Glazed section (e.g. `"global"`) and pass its slug into `ShortHelpSections`:

```go
// cmd/jsverbs-example/main.go (conceptual)
globalSection, _ := schema.NewSection("global", "Global flags",
    schema.WithDescription("Directory and logging options"),
    schema.WithFields(
        fields.New("dir", fields.TypeString, fields.WithHelp("Directory scanned before command registration")),
        // ... log-level, log-format, etc.
    ),
)

// Attach the section to every jsverb command description before registration
for _, cmd := range commands {
    cmd.Description().Schema.Set("global", globalSection)
}

// Then include "global" in short help
cli.AddCommandsToRootCommand(root, commands, nil,
    cli.WithParserConfig(cli.CobraParserConfig{
        ShortHelpSections: []string{schema.DefaultSlug, "global"},
    }),
)
```

**Caveat:** `logging.AddLoggingSectionToRootCommand` currently adds raw Cobra persistent flags because of a historical limitation (see comment in `glazed/pkg/cmds/logging/section.go:107-112`). Whoever fixes this should either:

1. Upgrade `AddLoggingSectionToRootCommand` to use Glazed sections properly, or
2. Create a lightweight wrapper that registers the same flags through a Glazed section and then mirrors them as Cobra persistent flags for compatibility.

---

### 3.3 Gap 3 — not all jsverbs should be glazed verbs (clarification)

**Current behavior (already correct)**

As shown in §1.4, `CommandForVerbWithInvoker` already branches on `verb.OutputMode`:

- `OutputModeGlaze` → `Command` (`cmds.GlazeCommand`) → gets all Glazed output flags (table format, CSV, JSON, jq, fields, filters, sort, etc.)
- `OutputModeText` → `WriterCommand` (`cmds.WriterCommand`) → gets only general command options (`--config-file`, `--print-yaml`, etc.)

**Evidence from `--long-help`**

- `basics echo` (glaze mode) shows **Glazed output format flags**, **Glazed fields and filters flags**, **Glazed jq flags**, etc.
- `basics banner` (text mode) shows only **General purpose command options** and **Global flags**.

**Conclusion:** The system already respects `output: "text"` in `__verb__` metadata. The gap is **documentation**, not code. Interns should know that:

- `output: "glaze"` (default) → structured rows, supports `--output json`, `--fields`, `--jq`, etc.
- `output: "text"` → plain string output, no table pipeline.

---

## 4. Phased implementation plan

### Phase 1 — Convert `list` to a Glazed command

1. Create `listCommand` type in `cmd/jsverbs-example/main.go` (or a new `pkg/jsverbs/commands.go` if reusable).
2. Implement `cmds.GlazeCommand` for it.
3. Remove the raw `cobra.Command` registration.
4. Insert `listCommand` at the front of the slice passed to `cli.AddCommandsToRootCommand`.
5. Add a simple test in `cmd/jsverbs-example/main_test.go` that runs `list --output json` and asserts valid JSON array output.

### Phase 2 — Fix short-help visibility of global flags

1. Decide whether to upgrade `logging.AddLoggingSectionToRootCommand` or add a host-specific wrapper.
2. Create a Glazed section for global flags (dir + logging).
3. Attach that section to every command description before registration.
4. Update `ShortHelpSections` to include the global section slug.
5. Verify short help for both `list` and a glazed verb like `echo` now shows global flags.

### Phase 3 — Documentation and hardening

1. Add a paragraph to `README.md` or a new `docs/jsverbs-output-modes.md` explaining `output: "glaze"` vs `output: "text"`.
2. Add a test that asserts `banner` (text mode) does NOT have `--output` flag, while `echo` (glaze mode) DOES.
3. Run `docmgr doctor` on any related tickets and update changelog.

---

## 5. Key files to read (onboarding checklist)

| File | Why read it |
|------|-------------|
| `pkg/jsverbs/scan.go` | Understand how JS source is parsed into `VerbSpec` / `SectionSpec` / `FieldSpec` |
| `pkg/jsverbs/model.go` | Type definitions for the registry and all spec objects |
| `pkg/jsverbs/binding.go` | How JS parameters map to parsed CLI values (binding modes) |
| `pkg/jsverbs/command.go` | How `VerbSpec` → `CommandDescription` → `Command` / `WriterCommand` |
| `pkg/jsverbs/runtime.go` | How the Goja runtime is created, how `__glazedVerbRegistry` overlay works, how promises are awaited |
| `cmd/jsverbs-example/main.go` | The host application that scans, registers shared sections, and wires into Cobra |
| `glazed/pkg/cli/cobra.go` | `AddCommandsToRootCommand`, `BuildCobraCommandFromCommand`, parent-command creation |
| `glazed/pkg/cli/cobra-parser.go` | `CobraParser`, `CobraParserConfig`, `ShortHelpSections`, middleware chain |
| `glazed/pkg/help/cmd/cobra.go` | Help rendering, `shortHelpSections` filtering logic, `FlagGroupUsage` |
| `glazed/pkg/cmds/schema/cobra_flag_groups.go` | How Cobra flags are grouped into slugs (`DefaultSlug`, `GlobalDefaultSlug`) |
| `glazed/pkg/cmds/logging/section.go` | How logging flags are defined and why they are currently raw Cobra persistent flags |

---

## 6. Testing and validation strategy

1. **Build** the example: `go build ./cmd/jsverbs-example`
2. **Short help check**: Run `./jsverbs-example basics echo --help` and confirm `--dir` and `--log-level` appear.
3. **Long help check**: Run `./jsverbs-example basics echo --help --long-help` and confirm Glazed output sections still appear.
4. **List structured check**: Run `./jsverbs-example list --output json` and confirm valid JSON array with objects containing `path`, `source`, `output_mode`.
5. **Output mode check**: Run `./jsverbs-example basics banner --help --long-help` and confirm NO `--output` or `--fields` flags. Run `./jsverbs-example basics echo --help --long-help` and confirm `--output` and `--fields` ARE present.
6. **Unit tests**: `go test ./pkg/jsverbs/...` and `go test ./cmd/jsverbs-example/...`

---

## 7. Risks, alternatives, and open questions

| Risk | Mitigation |
|------|------------|
| Changing `ShortHelpSections` might make short help too verbose for heavily-flagged commands | Only add the global section slug, not every glazed section |
| `logging.AddLoggingSectionToRootCommand` is used by many other go-go-golems repos | Any change must be backward-compatible or version-gated |
| Re-implementing `list` as Glazed adds dependency on `types.Row` and `middlewares.Processor` | The dependency already exists because `pkg/jsverbs` imports them |

**Open questions**

1. Should `list` support filtering flags (e.g. `--output-mode text` to show only text verbs)?
2. Should the global section be reusable across other host applications, or is it specific to `jsverbs-example`?
3. Is there a plan to make `AddLoggingSectionToRootCommand` use Glazed sections natively? (See TODO comment in `glazed/pkg/cmds/logging/section.go`)

---

## 8. References

- `go-go-goja` repo: `/home/manuel/code/wesen/go-go-golems/go-go-goja`
- `glazed` repo: `/home/manuel/code/wesen/go-go-golems/glazed`
- Relevant prior tickets in `go-go-goja/ttmp`:
  - `GOJA-05-JSVERBS-HARDENING`
  - `GOJA-07-JSVERBS-SHARED-SECTIONS`
  - `GOJA-16-JSVERBS-EXAMPLE-DEFAULT-DIR`
  - `GOJA-JSVERBS-INVOKER`
