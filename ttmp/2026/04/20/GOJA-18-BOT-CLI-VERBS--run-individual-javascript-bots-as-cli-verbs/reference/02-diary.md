---
Title: Diary
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
    - Path: ../../../../../../../loupedeck/cmd/loupedeck/cmds/verbs/command.go
      Note: Was inspected to document the reusable CLI command-wrapper pattern in the diary
    - Path: cmd/go-go-goja/main.go
      Note: Was added during Step 3 as the canonical root CLI binary
    - Path: examples/bots/README.md
      Note: Was added during Step 7 as the real example repository testing guide
    - Path: examples/bots/issues.js
      Note: Was added during Step 7 to exercise sections and context in a real example bot
    - Path: modules/sandbox/runtime.go
      Note: Was inspected to document why sandbox is separate from v1 CLI discovery
    - Path: pkg/botcli/bootstrap.go
      Note: Was added during Step 3 for repository bootstrap and explicit-verb scanning
    - Path: pkg/botcli/command.go
      Note: Was added during Step 3 for the public bots command surface
    - Path: pkg/botcli/command_test.go
      Note: Was added during Step 3 to validate the new bot CLI end-to-end
    - Path: pkg/doc/12-bot-cli-verb-authoring-guide.md
      Note: Was added during Step 5 as the authoring help page for explicit bot verbs
    - Path: pkg/jsverbs/runtime.go
      Note: Was inspected to document invocation behavior in the diary
    - Path: pkg/jsverbs/scan.go
      Note: Was inspected to document the discovery pipeline in the diary
    - Path: testdata/botcli-dupe-a/discord.js
      Note: Was added during Step 5 to validate duplicate repository detection
    - Path: testdata/botcli/discord.js
      Note: Was added during Step 5 as a dedicated bot CLI fixture
ExternalSources: []
Summary: Chronological diary for the GOJA-18 ticket covering investigation of jsverbs, loupedeck verb wiring, and the delivery of the bot CLI verbs design bundle.
LastUpdated: 2026-04-20T12:45:00-04:00
WhatFor: Capture the research and documentation steps behind the bot CLI verbs ticket.
WhenToUse: Read when continuing the ticket, reviewing evidence, or checking how the reMarkable bundle was produced.
---





# Diary

## Goal

Capture how the `GOJA-18-BOT-CLI-VERBS` ticket was created, how the relevant code paths were analyzed, and how the final design bundle for `go-go-goja bots list|run|help` was assembled and delivered.

## Step 1: Create the ticket and map the existing command/runtime evidence

I started by creating a new ticket in the `go-go-goja` repository because the request was not to implement the feature immediately, but to produce a detailed analysis and design guide that a new intern could use as an onboarding and implementation packet. The first job was therefore to identify which existing systems already solve parts of the problem and which pieces are still missing.

The investigation quickly confirmed that the problem is mostly orchestration, not missing runtime capability. `go-go-goja` already has the scanning and execution pipeline in `pkg/jsverbs`, while `loupedeck` already shows how to turn those pieces into a richer CLI. The real design work was to map those pieces onto the exact user-facing surface `go-go-goja bots list`, `go-go-goja bots run <verb>`, and `go-go-goja bots help <verb>`.

### Prompt Context

**User prompt (verbatim):** "this is what i want:

 - go-go-goja bots list                                                                                                                                                                                     
 - go-go-goja bots run <verb>                                                                                                                                                                               
 - go-go-goja bots help <verb>                                                                                                                                                                              
                                                                                                                                                                                                           ---

Create new ticket and reate a detailed analysis / design / implementation guide that is very detailed for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file
  references.
  It should be very clear and detailed. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Create a new docmgr ticket in `go-go-goja`, analyze how a `bots list|run|help` command surface should be built on top of existing `jsverbs` and related code, and deliver the design bundle to reMarkable.

**Inferred user intent:** Produce a reviewer-friendly and intern-friendly implementation guide before writing code, so the future implementation can follow a clear architecture instead of improvising.

### What I did
- Created the ticket `GOJA-18-BOT-CLI-VERBS` in the `go-go-goja` ticket workspace.
- Added three documents:
  - a primary design doc,
  - a quick-reference command/API doc,
  - and a diary.
- Re-read the most relevant `go-go-goja` and `loupedeck` source files with line-oriented evidence collection.
- Confirmed that the current repo has example binaries and utilities but no single canonical `go-go-goja` root CLI binary yet.

### Why
- The user explicitly asked for a detailed ticket deliverable rather than direct implementation.
- A new intern would need both architecture orientation and concrete file references.
- The command surface requested by the user is close to, but not identical to, the existing `jsverbs-example` and `loupedeck verbs` flows, so the differences had to be documented carefully.

### What worked
- `docmgr` was already initialized in the repository and ticket creation worked immediately.
- The previously explored `jsverbs` and `loupedeck` files lined up very cleanly with the requested feature.
- The separation between `jsverbs` and sandbox runtime concerns became a useful central design point.

### What didn't work
- Nothing materially blocked this step.
- One source of confusion during analysis was the temptation to treat sandbox `defineBot(...)` objects as if they were already scanner-visible CLI verbs. The code does not support that today, so I documented that boundary explicitly instead of papering over it.

### What I learned
- The missing feature is mostly a command orchestration layer, not a new runtime subsystem.
- `loupedeck` is the strongest reference for repository scanning, deduplication, and runtime-backed verb execution.
- The user-requested `bots run <verb>` shape is importantly different from `loupedeck verbs <verb>` and deserves its own adapter design.

### What was tricky to build
- The trickiest design point was preserving the exact requested UX while still reusing the strongest existing code patterns. The natural `loupedeck` pattern is “dynamic verb subcommands,” but the user wants static action commands (`list`, `run`, `help`). I handled that by designing a single-verb lazy handoff flow for `run` and `help` rather than recommending fully dynamic subcommands.

### What warrants a second pair of eyes
- Whether the project wants to introduce a canonical `cmd/go-go-goja` root binary immediately or stage the feature as another example binary first.
- Whether v1 should require explicit `__verb__` annotations only, or allow the current permissive `IncludePublicFunctions` behavior.

### What should be done in the future
- Implement the `pkg/botcli` orchestration package described in the design doc.
- Decide on the root binary strategy before coding Phase 1.

### Code review instructions
- Start with the design doc and then cross-check the cited files in `pkg/jsverbs` and `loupedeck/cmds/verbs`.
- Focus on whether the proposed `bots run <verb>` flow genuinely reuses `CommandDescriptionForVerb(...)` and `InvokeInRuntime(...)` rather than duplicating them.

### Technical details
- Ticket creation command used:
  - `docmgr ticket create-ticket --ticket GOJA-18-BOT-CLI-VERBS --title "Run Individual JavaScript Bots as CLI Verbs" --topics goja,javascript,cli,cobra,glazed,bots`
- Document creation commands used:
  - `docmgr doc add --ticket GOJA-18-BOT-CLI-VERBS --doc-type design-doc --title "Bot CLI verbs architecture and implementation guide"`
  - `docmgr doc add --ticket GOJA-18-BOT-CLI-VERBS --doc-type reference --title "Bot CLI verbs command surface and API reference"`
  - `docmgr doc add --ticket GOJA-18-BOT-CLI-VERBS --doc-type reference --title "Diary"`

## Step 2: Write the design bundle and publish it to reMarkable

Once the evidence was gathered, I wrote the design doc as the primary handoff document and the reference doc as the quick command/API sheet. I also updated the tasks, index, and changelog so the ticket behaves like a complete docmgr workspace rather than a loose markdown dump.

After the docs were in place, I validated the ticket with `docmgr doctor`, performed a reMarkable bundle dry-run, uploaded the final bundle, and verified that the remote path contains the document. That last step matters because the user explicitly asked for ticket storage plus reMarkable delivery.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Finish the documentation package, validate it, and upload the bundle to reMarkable.

**Inferred user intent:** Leave behind a durable, reviewable artifact that can be read on-device and used later during implementation.

### What I did
- Wrote the primary design doc with:
  - executive summary,
  - current-state mapping,
  - gap analysis,
  - proposed architecture,
  - pseudocode,
  - diagrams,
  - phased implementation plan,
  - risks and open questions,
  - file references.
- Wrote the quick-reference command/API doc.
- Updated `index.md`, `tasks.md`, and `changelog.md`.
- Related the key source files to the focused docs using `docmgr doc relate`.
- Ran `docmgr doctor --ticket GOJA-18-BOT-CLI-VERBS --stale-after 30`.
- Ran `remarquee status` and `remarquee cloud account --non-interactive`.
- Performed a bundle dry-run, then uploaded the ticket bundle, then verified the remote folder listing.

### Why
- The user asked for a detailed intern-friendly deliverable, not only a short answer in chat.
- The ticket should be continuation-friendly for later implementation work.
- Uploading the bundle keeps the off-screen review artifact synchronized with the repo-local ticket workspace.

### What worked
- The doc structure naturally supported one long-form design doc plus one quick-reference document.
- `docmgr doctor` validated the ticket cleanly.
- The reMarkable bundle upload and remote verification both succeeded.

### What didn't work
- Nothing failed in a consequential way during the delivery step.
- The only caution was making sure the design doc stayed focused on the exact requested command surface instead of drifting into broader `loupedeck` parity work.

### What I learned
- The best design story here is not “invent a new bot system,” but “package the existing jsverbs system into a stable operator-facing CLI.”
- Keeping the ticket docs narrow and file-backed makes the final bundle much easier to hand off.

### What was tricky to build
- The tricky part was balancing detail with clarity. The user wanted a very detailed guide for a new intern, which meant the doc had to explain the engine, `jsverbs`, `loupedeck`, and sandbox boundaries without turning into a random code tour. I handled that by centering the document around the requested commands and only including system parts that materially affect those commands.

### What warrants a second pair of eyes
- Whether the recommended v1 bootstrap scope (`--bot-repository` first, richer config later) matches project expectations.
- Whether JSON-first output for `glaze` mode is acceptable in v1 or whether the implementation should integrate more deeply with full Glazed renderers.

### What should be done in the future
- Implement the command package described in the design doc.
- Add fixtures and integration tests once the package exists.
- If needed, write a follow-up design note for sandbox-to-jsverbs wrappers.

### Code review instructions
- Review the design doc first, then the quick-reference doc, then the ticket `tasks.md`.
- Re-run `docmgr doctor` if any frontmatter or relations change.
- Re-run the bundle upload if the design doc is revised later.

### Technical details
- Validation command used:
  - `docmgr doctor --ticket GOJA-18-BOT-CLI-VERBS --stale-after 30`
- Upload commands used:
  - `remarquee upload bundle --dry-run --name "GOJA-18 Bot CLI Verbs" --remote-dir "/ai/2026/04/20/GOJA-18-BOT-CLI-VERBS" <ticket directory>`
  - `remarquee upload bundle --name "GOJA-18 Bot CLI Verbs" --remote-dir "/ai/2026/04/20/GOJA-18-BOT-CLI-VERBS" <ticket directory>`
- Remote verification command used:
  - `remarquee cloud ls /ai/2026/04/20/GOJA-18-BOT-CLI-VERBS --long --non-interactive`

## Step 3: Implement the first working `bots list|run|help` command surface

After the design bundle was committed, I started implementing the feature in the repository itself, beginning with the command shell and the reusable orchestration package. I chose the canonical `cmd/go-go-goja` binary path from the design doc and put the feature-specific logic in a new `pkg/botcli` package so the lower-level `pkg/jsverbs` APIs stayed generic.

The implementation deliberately followed the architecture from the ticket: repository bootstrap and duplicate detection live in `pkg/botcli`, verb resolution stays selector-driven, and runtime invocation reuses `jsverbs.CommandDescriptionForVerb(...)` plus `registry.InvokeInRuntime(...)`. I also kept the v1 discovery policy strict by scanning with `IncludePublicFunctions = false`, which means only explicit `__verb__` metadata becomes a bot command.

### Prompt Context

**User prompt (verbatim):** "add tasks. then work on them one by one, committing at appropriate intervals, keeping a diary as you go."

**Assistant interpretation:** Turn the design ticket into active implementation work, complete the command-line feature incrementally, and checkpoint the work with focused commits and diary updates.

**Inferred user intent:** Move from design into disciplined implementation without losing documentation quality or reviewability.

**Commit (code):** 869b954 — "Add bot CLI verbs command surface"

### What I did
- Added `cmd/go-go-goja/main.go` as a new canonical root CLI binary for this repo.
- Added `pkg/botcli/model.go` with repository and discovered-bot types.
- Added `pkg/botcli/bootstrap.go` for:
  - `--bot-repository` handling,
  - raw-arg filtering for lazy `run`/`help`,
  - path normalization,
  - directory scanning,
  - duplicate full-path detection.
- Added `pkg/botcli/resolve.go` for bot selector resolution from either explicit full paths or unique shorter selectors.
- Added `pkg/botcli/runtime.go` for:
  - building single-verb Cobra commands from `CommandDescriptionForVerb(...)`,
  - creating runtimes with the scanned-source loader plus filesystem module roots,
  - invoking selected verbs,
  - printing structured JSON or text output.
- Added `pkg/botcli/command.go` for the public `bots list`, `bots run`, and `bots help` subcommands.
- Added `pkg/botcli/command_test.go` covering:
  - list output,
  - structured verb execution,
  - text verb execution,
  - async Promise settlement,
  - relative `require()` support,
  - help output.
- Ran manual smoke commands and full repo tests with `GOWORK=off go test ./...`.

### Why
- The user asked to work through the implementation tasks rather than stopping at design.
- The `pkg/botcli` package keeps the UX-specific logic separate from the reusable `jsverbs` package.
- A dedicated root binary makes the new surface easier to discover and easier to extend later.

### What worked
- The `loupedeck` pattern translated cleanly into a smaller `go-go-goja`-specific command layer.
- Reusing `CommandDescriptionForVerb(...)` kept `run` and `help` aligned.
- The existing async settlement work in `jsverbs` and sandbox made the async bot CLI test pass without extra promise-specific CLI logic.
- The full repository test suite passed with `GOWORK=off go test ./...` after the new package landed.

### What didn't work
- The first test expectation incorrectly assumed the `packaged.js` fixture would still show up in `bots list`. That fixture relies on public-function inference, but the new CLI deliberately disables `IncludePublicFunctions`, so the test had to be updated to reflect the explicit-verb-only policy.
- The initial help test also assumed positional argument help text would be rendered more verbosely than Cobra currently prints for this generated command, so I adjusted the assertion to match the actual help output.
- I hit one compile error while wiring runtime options:
  - `pkg/botcli/runtime.go:72:40: bot.runtimeOptions undefined (type DiscoveredBot has no field or method runtimeOptions)`
  - The fix was to call the helper on `bot.Repository` instead.

### What I learned
- The static `bots run <verb>` UX works well when selector resolution is implemented as “longest matching argument prefix.”
- Supporting both parsed inherited flags and manually filtered raw args makes lazy commands much easier to use.
- Explicit `__verb__`-only discovery is a good default for a production-shaped CLI because it avoids accidental command exposure.

### What was tricky to build
- The trickiest part was balancing lazy command dispatch with verb-specific flag parsing. `bots run` and `bots help` need to defer verb parsing until after the selected verb is known, so they cannot rely on normal Cobra flag registration at the outer command layer. I handled that by manually stripping repository flags from the raw argument stream, resolving the bot selector from the remaining args, then building an ephemeral verb-specific Cobra command and parser.

### What warrants a second pair of eyes
- Whether the exact selector-matching rules should stay as implemented or be narrowed to exact full paths only.
- Whether the v1 explicit-verb-only policy should be surfaced more prominently in user-facing docs/help text.
- Whether `bots run` should eventually integrate deeper with Glazed table renderers rather than using JSON-first structured output.

### What should be done in the future
- Add dedicated bot-specific fixture trees under `testdata/` instead of only reusing `testdata/jsverbs`.
- Add multi-repository and duplicate-selector tests.
- Add user-facing authoring docs or a help page explaining how to write bot scripts with `__verb__`.

### Code review instructions
- Start with `pkg/botcli/command.go`, then read `bootstrap.go`, `resolve.go`, and `runtime.go` in that order.
- Review `cmd/go-go-goja/main.go` to see how the new root command is exposed.
- Run these manual smoke commands:
  - `GOWORK=off go run ./cmd/go-go-goja bots list --bot-repository ./testdata/jsverbs`
  - `GOWORK=off go run ./cmd/go-go-goja bots run basics greet --bot-repository ./testdata/jsverbs Manuel --excited`
  - `GOWORK=off go run ./cmd/go-go-goja bots help basics greet --bot-repository ./testdata/jsverbs`
- Re-run the full suite with `GOWORK=off go test ./...`.

### Technical details
- Manual validation commands used:
  - `GOWORK=off go run ./cmd/go-go-goja bots list --bot-repository ./testdata/jsverbs`
  - `GOWORK=off go run ./cmd/go-go-goja bots run basics greet --bot-repository ./testdata/jsverbs Manuel --excited`
  - `GOWORK=off go run ./cmd/go-go-goja bots help basics greet --bot-repository ./testdata/jsverbs`
- Focused package test command used:
  - `GOWORK=off go test ./pkg/botcli ./cmd/go-go-goja`
- Full validation command used:
  - `GOWORK=off go test ./...`
- Exact compile error encountered during implementation:
  - `pkg/botcli/runtime.go:72:40: bot.runtimeOptions undefined (type DiscoveredBot has no field or method runtimeOptions)`

## Step 4: Refresh the ticket bundle after implementation so the PDF matches the repo

After the command surface and tests were committed, I refreshed the reMarkable bundle so the ticket PDF would include the implementation progress, updated tasks, and diary notes. This is the delivery hygiene step that keeps the off-screen artifact synchronized with the repository state.

### Prompt Context

**User prompt (verbatim):** (same as Step 3)

**Assistant interpretation:** Keep the ticket diary and deliverable current as implementation progresses.

**Inferred user intent:** Make sure the implementation work remains documented and reviewable, not just coded.

### What I did
- Uploaded the updated ticket workspace again with `remarquee upload bundle --force ...`.
- Verified the remote folder after upload.

### Why
- The original bundle only reflected the design stage.
- The implementation added code and diary entries that should be visible in the reMarkable PDF.

### What worked
- The forced bundle upload succeeded on the first try.
- The remote folder still contains the expected `GOJA-18 Bot CLI Verbs` document.

### What didn't work
- Nothing new failed in this refresh step.

### What I learned
- It is worth treating the ticket directory as the single source of truth and re-uploading after each meaningful implementation milestone.

### What was tricky to build
- Nothing was technically tricky here; the only requirement was remembering to refresh the bundle after the code and diary changed.

### What warrants a second pair of eyes
- Whether future tickets should always re-upload after each implementation milestone or only once at ticket closeout.

### What should be done in the future
- Re-run the bundle upload again if the remaining follow-up tasks change the docs substantially.

### Code review instructions
- Compare the refreshed PDF on reMarkable with the latest files in the ticket workspace if you want to validate the handoff artifact.

### Technical details
- Upload command used:
  - `remarquee upload bundle --force --name "GOJA-18 Bot CLI Verbs" --remote-dir "/ai/2026/04/20/GOJA-18-BOT-CLI-VERBS" <ticket directory>`
- Verification command used:
  - `remarquee cloud ls /ai/2026/04/20/GOJA-18-BOT-CLI-VERBS --long --non-interactive`

## Step 5: Add bot-specific fixtures, edge-case tests, and an authoring help page

With the first working command surface in place, I finished the remaining immediate tasks by creating dedicated fixture trees for the bot CLI, extending the tests to cover empty and duplicate repository behavior, and adding a help page that explains how bot authors should write scripts for the new command surface. This step matters because it turns the feature from “working in the happy path” into something that is easier to maintain and teach.

The new fixture tree also makes the bot CLI less dependent on the older generic `testdata/jsverbs` examples. That is useful for future contributors because the tests now show the exact authoring style expected by `go-go-goja bots ...` rather than only borrowing from the broader jsverbs prototype fixtures.

### Prompt Context

**User prompt (verbatim):** (same as Step 3)

**Assistant interpretation:** Continue working through the remaining implementation tasks instead of stopping after the first successful command surface.

**Inferred user intent:** Finish the most obvious validation and documentation follow-through so the feature feels complete, not merely demonstrated.

**Commit (code):** eb5a944 — "Add bot CLI fixtures and authoring guide"

### What I did
- Added a dedicated `testdata/botcli/` fixture tree with explicit bot verbs:
  - `discord.js`
  - `async.js`
  - `nested/relay.js`
  - `nested/sub/helper.js`
- Added duplicate-path fixtures under:
  - `testdata/botcli-dupe-a/discord.js`
  - `testdata/botcli-dupe-b/discord.js`
- Extended `pkg/botcli/command_test.go` with tests for:
  - multiple repositories,
  - empty repositories,
  - duplicate bot path rejection,
  - dedicated bot fixture relative `require()` behavior.
- Added `pkg/doc/12-bot-cli-verb-authoring-guide.md` to explain how to write bot scripts with explicit `__verb__(...)` metadata.
- Re-ran `GOWORK=off go test ./pkg/botcli ./cmd/go-go-goja` and `GOWORK=off go test ./...`.

### Why
- Dedicated fixtures make the bot CLI contract clearer than reusing only the generic jsverbs examples.
- Duplicate and empty repository cases are exactly the kinds of operational edges that should be tested before the feature is treated as stable.
- The authoring guide closes the loop for future users by documenting the explicit-verb policy chosen for v1.

### What worked
- The duplicate fixture setup made it easy to verify that conflicting full bot paths fail loudly.
- The new authoring guide fit naturally into the existing embedded help-doc system because `pkg/doc` already loads all markdown help pages.
- The full repository test suite still passed after the additional fixtures and tests landed.

### What didn't work
- Nothing substantial failed in this step.
- The only small adjustment was making sure the dedicated nested fixture used the expected full path (`nested relay relay`) that comes from jsverbs’ directory/file/verb path rules.

### What I learned
- A dedicated fixture tree is more than test data; it becomes executable documentation for future contributors.
- The v1 explicit `__verb__` policy is much easier to explain once there is a focused help page that says so plainly.

### What was tricky to build
- The slightly tricky part was remembering how `jsverbs` derives full paths from both directories and file names. For nested fixtures, the full command path includes the directory component, the file-derived parent, and the verb name, so the test selectors had to match that exact structure.

### What warrants a second pair of eyes
- Whether the authoring guide should eventually be split into both a help page and a README section.
- Whether the duplicate-repository error should include repository names in addition to the source refs already shown.

### What should be done in the future
- Consider adding shell-completion tests once the root CLI surface stabilizes further.
- Consider adding env/config repository discovery if users want parity with the richer `loupedeck` bootstrap flow.

### Code review instructions
- Read `pkg/botcli/command_test.go` to see the new validation coverage.
- Inspect `testdata/botcli/` and `pkg/doc/12-bot-cli-verb-authoring-guide.md` together to understand the intended authoring contract.
- Re-run `GOWORK=off go test ./...` from the repo root.

### Technical details
- Focused validation command used:
  - `GOWORK=off go test ./pkg/botcli ./cmd/go-go-goja`
- Full validation command used:
  - `GOWORK=off go test ./...`
- New authoring help page:
  - `pkg/doc/12-bot-cli-verb-authoring-guide.md`

## Step 6: Refresh the bundle again after the fixture/docs follow-up

Because the fixture additions, extra tests, and authoring guide changed the ticket content after the prior upload, I refreshed the reMarkable bundle one more time so the PDF stayed aligned with the latest implementation and diary state.

### Prompt Context

**User prompt (verbatim):** (same as Step 3)

**Assistant interpretation:** Keep the ticket artifact synchronized as the remaining follow-up tasks land.

**Inferred user intent:** Preserve a current deliverable, not just current source files.

### What I did
- Re-uploaded the ticket bundle after the fixture/tests/help-page follow-up.
- Verified the remote reMarkable folder again.

### Why
- The ticket changed materially after the previous refresh.
- The reMarkable PDF should include the dedicated fixture and authoring-guide work too.

### What worked
- The forced upload succeeded.
- The verification listing still shows the expected document in the target folder.

### What didn't work
- Nothing new failed in this refresh step.

### What I learned
- The ticket bundle really is easiest to maintain when it is refreshed after each meaningful documentation milestone.

### What was tricky to build
- Nothing technical here; this was just a disciplined refresh step.

### What warrants a second pair of eyes
- Whether the project wants a stricter ticket-closeout checklist for bundle refreshes.

### What should be done in the future
- Re-run the bundle upload again if the open follow-up questions materially change the docs.

### Code review instructions
- Compare the refreshed PDF with the final state of the ticket workspace if you want to verify the artifact pipeline end to end.

### Technical details
- Upload command used:
  - `remarquee upload bundle --force --name "GOJA-18 Bot CLI Verbs" --remote-dir "/ai/2026/04/20/GOJA-18-BOT-CLI-VERBS" <ticket directory>`
- Verification command used:
  - `remarquee cloud ls /ai/2026/04/20/GOJA-18-BOT-CLI-VERBS --long --non-interactive`

## Step 7: Add a real example bot repository and a smoke-test playbook

The earlier fixture work made the bot CLI testable, but the user asked for something more realistic: a whole set of example bots that can be run manually to exercise the feature set end to end. To support that, I created a real example repository under `examples/bots` and wrote a playbook document with exact commands that a human can copy and run.

This step is important because it turns the feature into something you can actually demo and validate interactively. Instead of pointing people only at low-level fixtures or unit tests, the repo now contains a discoverable bot repository with examples for structured output, text output, async commands, nested `require()`, bound sections, package metadata, and `bind: all`.

### Prompt Context

**User prompt (verbatim):** "How can I test a bot? can you make a whole set of bot examples so we can now test sall the functionality for real"

**Assistant interpretation:** Create a realistic example repository plus concrete testing instructions so the bot CLI can be exercised manually and end to end.

**Inferred user intent:** Move from implementation and unit tests to hands-on validation with believable example bots.

### What I did
- Added a real example repository under `examples/bots` with these example scripts:
  - `discord.js`
  - `math/index.js`
  - `nested/index.js`
  - `nested/sub/helper.js`
  - `issues.js`
  - `admin.js`
  - `all-values.js`
  - `README.md`
- Extended `pkg/botcli/command_test.go` with example-repository smoke tests that cover:
  - listing all example paths,
  - structured output,
  - text output,
  - async Promise settlement,
  - relative `require()`,
  - bound sections and context,
  - package metadata,
  - `bind: all`,
  - help output.
- Added a ticket playbook:
  - `playbook/01-real-bot-example-repository-smoke-tests.md`
- Ran focused tests, manual smoke commands, and the full repository test suite.

### Why
- The user asked how to test a bot in practice, not only how the architecture works.
- A real example repository is easier to understand than minimal internal fixtures.
- A playbook makes repeated manual validation much faster for future review and regression checking.

### What worked
- The example repository exercised the bot CLI exactly the way the user wanted: by running actual `go-go-goja bots ...` commands against a realistic directory tree.
- The examples surfaced and validated several important behaviors:
  - async settlement,
  - relative `require()`,
  - file-local sections,
  - `bind: context`,
  - `bind: all`,
  - package metadata-driven command paths.
- The manual smoke commands and `GOWORK=off go test ./...` both passed after the examples were added.

### What didn't work
- One of the example smoke tests initially failed because I guessed the flag would be exposed as `--dry-run`, but the generated Glazed/Cobra flag name for that field is `--dryRun`.
- The exact error was:
  - `unknown flag: --dry-run`
- I updated the test and example README to use the actual generated flag name.

### What I learned
- Real examples are very good at exposing tiny UX mismatches that unit-style fixtures can miss.
- The bot CLI is already strong enough to support a meaningful example repository without additional framework work.

### What was tricky to build
- The slightly tricky part was choosing examples that each exercised a distinct feature without making the repository feel random. I solved that by giving each file one job: Discord-style verbs for basic output modes, math verbs for async and list args, nested verbs for relative imports, issues for sections/context, admin for package metadata, and all-values for `bind: all`.

### What warrants a second pair of eyes
- Whether the `--dryRun` flag naming is acceptable as-is or whether future work should normalize camelCase field names into kebab-case flags more aggressively.
- Whether the example repository should eventually move into a top-level `examples/` convention used more broadly across the repo.

### What should be done in the future
- Add shell-completion checks or snapshot help tests against the example repository if the CLI surface evolves further.
- Consider adding one example that wraps sandbox functionality behind an explicit `__verb__` if that integration becomes a priority.

### Code review instructions
- Start with `examples/bots/README.md` and run the listed commands manually.
- Then read `pkg/botcli/command_test.go` to see how the example repository is used in automated coverage.
- Finally read the smoke-test playbook in the ticket workspace.

### Technical details
- Manual smoke commands used:
  - `GOWORK=off go run ./cmd/go-go-goja bots list --bot-repository ./examples/bots`
  - `GOWORK=off go run ./cmd/go-go-goja bots run issues list --bot-repository ./examples/bots acme/repo --state closed --labels bug --labels docs`
  - `GOWORK=off go run ./cmd/go-go-goja bots run all-values echo-all --bot-repository ./examples/bots --repo acme/demo --dryRun --names one --names two`
  - `GOWORK=off go run ./cmd/go-go-goja bots help discord greet --bot-repository ./examples/bots`
- Focused test command used:
  - `GOWORK=off go test ./pkg/botcli ./cmd/go-go-goja`
- Full validation command used:
  - `GOWORK=off go test ./...`
- Exact CLI mismatch encountered while validating examples:
  - `unknown flag: --dry-run`

## Related

- `../design-doc/01-bot-cli-verbs-architecture-and-implementation-guide.md`
- `01-bot-cli-verbs-command-surface-and-api-reference.md`
