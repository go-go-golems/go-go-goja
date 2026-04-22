---
Title: Investigation diary
Ticket: GOJA-JSVERBS-GLAZED-REGISTRATION
Status: active
Topics:
    - jsverbs
    - glazed
    - cli
    - help-system
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Chronological log of the investigation into jsverbs registration on top of glazed."
LastUpdated: 2026-04-22T17:05:00-04:00
WhatFor: "Track what was tried, what worked, what failed, and what to do next."
WhenToUse: "When resuming work on this ticket or when a new engineer needs to understand the investigation path."
---

# Investigation diary

## 2026-04-22 — Initial deep-dive

### What was asked

Create a docmgr ticket to analyze how go-go-goja registers jsverbs on top of glazed, focusing on three issues:

1. `jsverbs-examples list` should be a glazed command with structured output.
2. `jsverbs-example basics echo` and others should show global flags in short help.
3. Clarify whether all jsverbs become glazed verbs (they should not).

### What worked

1. **Built the example** to observe actual behavior:
   ```bash
   cd /home/manuel/code/wesen/go-go-golems/go-go-goja
   go build -o /tmp/jsverbs-example ./cmd/jsverbs-example
   ```

2. **Confirmed Gap 1** — `list` is plain text:
   ```bash
   /tmp/jsverbs-example list
   # Output: tab-separated strings, no --output json support
   ```

3. **Confirmed Gap 2** — global flags hidden in short help for glazed verbs:
   ```bash
   /tmp/jsverbs-example basics echo --help
   # Shows only "Arguments" section (help + long-help)
   # --dir, --log-level hidden

   /tmp/jsverbs-example basics echo --help --long-help
   # Shows "Global flags" section with --dir, --log-level, etc.
   ```

4. **Confirmed Gap 3 is already handled correctly** — `banner` (text mode) does NOT get glazed output flags, while `echo` (glaze mode) DOES. The branch is in `pkg/jsverbs/command.go:77-93`.

5. **Located the root cause of Gap 2** in `glazed/pkg/help/cmd/cobra.go:231-249`:
   - `shortHelpSections` annotation filters flag groups.
   - `jsverbs-example` sets `ShortHelpSections: []string{schema.DefaultSlug}` (="default").
   - Inherited flags live in `GlobalDefaultSlug` (="global-default"), so they are discarded.
   - The plain `list` command has no `shortHelpSections` annotation, so Cobra renders inherited flags natively.

6. **Understood why global flags are raw Cobra persistent flags**:
   - `logging.AddLoggingSectionToRootCommand` in `glazed/pkg/cmds/logging/section.go` adds flags manually to `rootCmd.PersistentFlags()`.
   - There is a commented-out TODO showing the "proper" Glazed-section approach.

### What was tricky

- The help system has two rendering paths:
  1. **Glazed path** (for commands built via `cli.AddCommandsToRootCommand`) — uses `cobra-short-help.tmpl` + `FlagGroupUsage` + `shortHelpSections` filtering.
  2. **Plain Cobra path** (for raw `cobra.Command` like `list`) — uses Cobra’s native `InheritedFlags` rendering.
  This duality makes it easy to see inconsistent behavior between `list` and `echo`.

- `schema.DefaultSlug` is `"default"` while the inherited bucket is `"global-default"`. The naming similarity makes it easy to assume they are the same or related.

### Commands run

```bash
# Build and inspect help
cd /home/manuel/code/wesen/go-go-golems/go-go-goja
go build -o /tmp/jsverbs-example ./cmd/jsverbs-example
/tmp/jsverbs-example list --help
/tmp/jsverbs-example basics echo --help
/tmp/jsverbs-example basics echo --help --long-help
/tmp/jsverbs-example basics banner --help --long-help

# Search for key patterns in glazed
rg -n "shortHelpSections" /home/manuel/code/wesen/go-go-golems/glazed/
rg -n "DefaultSlug\|GlobalDefaultSlug" /home/manuel/code/wesen/go-go-golems/glazed/pkg/cmds/schema/
```

## 2026-04-22 — Implementation

### What changed

1. **Converted `list` to a glazed `GlazeCommand`** in `cmd/jsverbs-example/main.go`:
   - Added `listCommand` struct embedding `*cmds.CommandDescription` and implementing `RunIntoGlazeProcessor`.
   - Emits rows with `path`, `source`, `output_mode` columns.
   - Removed the old raw `cobra.Command` with `fmt.Fprintln`/`sort.Strings`.
   - Prepended `listCmd` to the commands slice passed to `cli.AddCommandsToRootCommand`.

2. **Fixed short-help global flag visibility** in the same file:
   - Changed `ShortHelpSections: []string{schema.DefaultSlug}` to `ShortHelpSections: []string{schema.DefaultSlug, schema.GlobalDefaultSlug}`.
   - This makes `--dir`, `--log-level`, and all other inherited persistent flags visible in short help for every glazed command (including the new `list`).

### Verification

```bash
cd /home/manuel/code/wesen/go-go-golems/go-go-goja
go build -o /tmp/jsverbs-example ./cmd/jsverbs-example

# Structured JSON output
/tmp/jsverbs-example list --output json
# → valid JSON array with {"path":"...","source":"...","output_mode":"..."}

# CSV output
/tmp/jsverbs-example list --output csv
# → header + rows

# Short help now shows global flags
/tmp/jsverbs-example list --help
# → shows "Global flags" section with --dir, --log-level, etc.

# Existing tests pass
go test ./cmd/jsverbs-example/...
# → ok
```

### What to do next

1. Update `README.md` or add `docs/jsverbs-output-modes.md` to document `output: "glaze"` vs `output: "text"`.
2. Potentially add filtering flags to `list` (e.g. `--output-mode text` to show only text verbs) if desired.
3. Consider upstreaming a proper Glazed-section based global flag registration in `glazed/pkg/cmds/logging/section.go` (replacing the raw Cobra persistent flags).
