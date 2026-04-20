---
Title: Bot CLI verbs command surface and API reference
Ticket: GOJA-18-BOT-CLI-VERBS
Status: active
Topics:
    - goja
    - javascript
    - cli
    - cobra
    - glazed
    - bots
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../loupedeck/cmd/loupedeck/cmds/run/session.go
      Note: Shows caller-owned runtime invocation for a selected verb
    - Path: ../../../../../../../loupedeck/pkg/scriptmeta/scriptmeta.go
      Note: Shows useful selector and target-resolution patterns for follow-up implementation
    - Path: cmd/go-go-goja/main.go
      Note: Exposes the new bots command from the root binary
    - Path: cmd/jsverbs-example/main.go
      Note: Smallest example of scanning and exposing jsverbs through a CLI
    - Path: pkg/botcli/command_test.go
      Note: Proves list
    - Path: pkg/botcli/resolve.go
      Note: Defines the selector resolution rules used by bots run and bots help
    - Path: pkg/jsverbs/model.go
      Note: Defines Registry and VerbSpec fields referenced by the new command surface
    - Path: pkg/jsverbs/runtime.go
      Note: Defines the runtime invocation contract reused by bots run
ExternalSources: []
Summary: Quick-reference companion to the design doc for the proposed `go-go-goja bots list|run|help` surface, including command contracts, API sketches, file map, and implementation checkpoints.
LastUpdated: 2026-04-20T12:45:00-04:00
WhatFor: Give reviewers and implementers a concise copy/paste-ready reference for the bot CLI verb design.
WhenToUse: Use while implementing or reviewing the new bot CLI package and its Cobra/Glazed/jsverbs integration.
---



# Bot CLI verbs command surface and API reference

## Goal

Provide a quick-reference document for the proposed bot CLI surface so an engineer can answer four questions quickly:

1. What commands are we building?
2. What existing APIs should we reuse?
3. Which files matter most?
4. What should the implementation sequence look like?

## Context

The design doc is the long-form explanation. This reference is the short-form checklist and contract sheet.

The important context is:

- `pkg/jsverbs` already scans, describes, and runs JavaScript verbs.
- `loupedeck` proves those APIs can power a real CLI.
- the sandbox module is runtime-oriented and should not be used as the primary scanner target in v1.

## Quick reference

### User-facing command contract

```text
go-go-goja bots list [--bot-repository DIR...]
go-go-goja bots run <verb> [verb flags and args...]
go-go-goja bots help <verb>
```

### Behavioral contract

| Command | Input | Output | Notes |
|---|---|---|---|
| `bots list` | repository flags | sorted list of discovered verbs | show full path + source ref |
| `bots run <verb>` | selector + verb-specific args | structured JSON or text | schema comes from `CommandDescriptionForVerb(...)` |
| `bots help <verb>` | selector | Cobra/Glazed help text | must reuse the same description as `run` |

### Core reusable APIs

#### Scan and registry

- `jsverbs.ScanDir(root, opts...)` — `pkg/jsverbs/scan.go:17-74`
- `jsverbs.ScanFS(...)` — `pkg/jsverbs/scan.go:76-124`
- `(*Registry).Verbs()` — `pkg/jsverbs/model.go:159-163`
- `(*Registry).Verb(fullPath)` — `pkg/jsverbs/model.go:165-168`

#### Description building

- `(*Registry).CommandDescriptionForVerb(verb)` — `pkg/jsverbs/command.go:61-65`
- `(*Registry).CommandForVerbWithInvoker(...)` — `pkg/jsverbs/command.go:73-100`

#### Runtime invocation

- `(*Registry).InvokeInRuntime(ctx, runtime, verb, values)` — `pkg/jsverbs/runtime.go:44-107`
- `(*Registry).RequireLoader()` — `pkg/jsverbs/runtime.go:38-42`

#### Engine composition

- `engine.NewBuilder(...)` — `engine/factory.go:35-46`
- `(*FactoryBuilder).WithRequireOptions(...)` — `engine/factory.go:57-62`
- `(*FactoryBuilder).WithModules(...)` — `engine/factory.go:64-69`
- `(*Factory).NewRuntime(ctx)` — `engine/factory.go:152-230`

### Reference architecture diagram

```text
bots command
  |
  +--> bootstrap repositories
  |
  +--> scan via jsverbs.ScanDir
  |
  +--> resolve one VerbSpec
  |
  +--> build CommandDescriptionForVerb
  |
  +--> parse verb args with Glazed/Cobra parser
  |
  +--> build engine runtime
  |
  +--> registry.InvokeInRuntime
  |
  +--> print result
```

### Recommended new package layout

```text
cmd/go-go-goja/main.go
pkg/botcli/bootstrap.go
pkg/botcli/model.go
pkg/botcli/list_command.go
pkg/botcli/run_command.go
pkg/botcli/help_command.go
pkg/botcli/command_helpers.go
```

### Minimal data model sketch

```go
type Repository struct {
    Name      string
    RootDir   string
    Source    string
    SourceRef string
}

type Bootstrap struct {
    Repositories []Repository
}

type ScannedRepository struct {
    Repository Repository
    Registry   *jsverbs.Registry
}

type DiscoveredBot struct {
    Repository ScannedRepository
    Verb       *jsverbs.VerbSpec
}
```

### Minimal helper sketch

```go
func DiscoverBootstrapFromCommand(cmd *cobra.Command) (Bootstrap, error)
func ScanRepositories(bootstrap Bootstrap) ([]ScannedRepository, error)
func CollectDiscoveredBots(repos []ScannedRepository) ([]DiscoveredBot, error)
func ResolveBot(selector string, discovered []DiscoveredBot) (DiscoveredBot, error)
func PrintVerbResult(w io.Writer, outputMode string, result any) error
```

### Command implementation sketch

#### `bots list`

```go
func runList(cmd *cobra.Command, args []string) error {
    bootstrap, _ := DiscoverBootstrapFromCommand(cmd)
    repos, _ := ScanRepositories(bootstrap)
    bots, _ := CollectDiscoveredBots(repos)
    for _, bot := range bots {
        fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", bot.Verb.FullPath(), bot.Verb.SourceRef())
    }
    return nil
}
```

#### `bots run <verb>`

```go
func runSelectedBot(cmd *cobra.Command, selector string, rawArgs []string) error {
    bot, _ := resolveFromCommand(cmd, selector)
    desc, _ := bot.Repository.Registry.CommandDescriptionForVerb(bot.Verb)
    parser, _ := glazedcli.NewCobraParserFromSections(desc.Schema, &glazedcli.CobraParserConfig{
        SkipCommandSettingsSection: true,
    })
    parsed, _ := parser.Parse(cmd, rawArgs)
    rt, _ := newRuntimeForBot(cmd.Context(), bot)
    defer rt.Close(context.Background())
    result, _ := bot.Repository.Registry.InvokeInRuntime(cmd.Context(), rt, bot.Verb, parsed)
    return PrintVerbResult(cmd.OutOrStdout(), bot.Verb.OutputMode, result)
}
```

#### `bots help <verb>`

```go
func showSelectedBotHelp(cmd *cobra.Command, selector string) error {
    bot, _ := resolveFromCommand(cmd, selector)
    desc, _ := bot.Repository.Registry.CommandDescriptionForVerb(bot.Verb)
    verbCmd := glazedcli.NewCobraCommandFromCommandDescription(desc)
    parser, _ := glazedcli.NewCobraParserFromSections(desc.Schema, cfg)
    _ = parser.AddToCobraCommand(verbCmd)
    verbCmd.SetOut(cmd.OutOrStdout())
    return verbCmd.Help()
}
```

## Usage examples

### Example 1: list available bots

```bash
go run ./cmd/go-go-goja bots list --bot-repository ./testdata/jsverbs
```

Expected shape:

```text
basics banner          basics.js
basics echo            basics.js
basics greet           basics.js
basics listIssues      basics.js
```

### Example 2: run one bot verb

```bash
go run ./cmd/go-go-goja bots run basics greet Manuel --bot-repository ./testdata/jsverbs --excited
```

Expected shape:

```json
{
  "greeting": "Hello, Manuel!"
}
```

### Example 3: show help for one bot verb

```bash
go run ./cmd/go-go-goja bots help basics greet --bot-repository ./testdata/jsverbs
```

Expected shape:

```text
Usage:
  go-go-goja bots run basics greet <name> [flags]

Flags:
  -e, --excited   Add excitement
```

## File map

### Most important `go-go-goja` files

- `cmd/jsverbs-example/main.go`
  - smallest example of scanning + mounting commands
- `pkg/jsverbs/scan.go`
  - source discovery
- `pkg/jsverbs/model.go`
  - registry and verb metadata
- `pkg/jsverbs/command.go`
  - schema and command description generation
- `pkg/jsverbs/runtime.go`
  - runtime invocation and Promise handling
- `engine/factory.go`
  - runtime builder and ownership model
- `engine/runtime_modules.go`
  - runtime-scoped registrar interface
- `modules/sandbox/runtime.go`
  - contrast case: runtime bot API, not CLI scanning
- `pkg/sandbox/registrar.go`
  - host-side sandbox registration

### Most important `loupedeck` files

- `cmd/loupedeck/main.go`
  - root command wiring for `verbs`
- `cmd/loupedeck/cmds/verbs/bootstrap.go`
  - repository discovery and dedupe
- `cmd/loupedeck/cmds/verbs/command.go`
  - dynamic command wrappers and result printing
- `pkg/scriptmeta/scriptmeta.go`
  - target and selector resolution patterns
- `cmd/loupedeck/cmds/run/session.go`
  - runtime invocation inside a caller-owned session

## Review checklist

### Architecture

- [ ] Does the new command layer remain thin and orchestration-only?
- [ ] Does it reuse `pkg/jsverbs` rather than duplicating it?
- [ ] Does it keep sandbox discovery out of v1?

### Command semantics

- [ ] Does `bots list` sort and label results clearly?
- [ ] Does `bots run` resolve exactly one selector and parse verb-specific flags?
- [ ] Does `bots help` use the same description source as `bots run`?

### Runtime correctness

- [ ] Are `require` overlay and module roots configured together?
- [ ] Does Promise-returning JavaScript still settle correctly?
- [ ] Are structured and text output both handled?

### Errors

- [ ] Duplicate full paths fail loudly.
- [ ] Missing selectors fail clearly.
- [ ] Ambiguous selectors explain the valid choices.

## Implementation order

1. Add root command shell.
2. Add bootstrap/repository scanning package.
3. Implement `bots list`.
4. Implement selector resolution.
5. Implement `bots run`.
6. Implement `bots help`.
7. Add fixtures and tests.
8. Add docs/examples.

## Related

- `../design-doc/01-bot-cli-verbs-architecture-and-implementation-guide.md`
- `02-diary.md`
