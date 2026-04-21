---
Title: Diary
Ticket: GOJA-23-CONFIGURABLE-REPLAPI
Status: active
Topics:
    - persistent-repl
    - architecture
    - repl
    - refactor
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/replapi/app.go
      Note: Step 2 configurable replapi construction and restore behavior
    - Path: pkg/replapi/config.go
      Note: Step 2 profile and config surface
    - Path: pkg/replapi/app_test.go
      Note: Step 2 profile and override tests
    - Path: pkg/replsession/policy.go
      Note: Step 2 session policy definitions
    - Path: pkg/replsession/service.go
      Note: Step 2 policy-driven evaluation and persistence gating
    - Path: pkg/replsession/service_policy_test.go
      Note: Step 2 raw and interactive service coverage
    - Path: cmd/goja-repl/root.go
      Note: Step 2 persistent profile adoption
    - Path: cmd/repl/main.go
      Note: Step 3 interactive profile adoption in the line REPL
ExternalSources: []
Summary: Chronological implementation diary for the configurable replapi profile and policy refactor.
LastUpdated: 2026-04-03T19:40:11.382694834-04:00
WhatFor: Use this diary to review the implementation sequence, tradeoffs, and validation evidence for the configurable replapi refactor.
WhenToUse: Use when handing off GOJA-23 work or checking why a specific config/profile decision was made.
---

# Diary

## Goal

Record the implementation of the configurable `replapi` refactor that introduces profile-based policy control for raw, interactive, and persistent REPL usage.

## Step 1: Ticket Plan and Refactor Boundary

The first step was to define the scope tightly enough that the refactor improves the API instead of scattering one-off flags across constructors. The current code already has the durable machinery, but it exposes that machinery as the only `replapi` behavior. That works for `goja-repl`, but it is the wrong default shape for the traditional REPLs and for any caller that just wants execution plus a little REPL ergonomics.

The resulting plan reframes the problem as policy selection. `replapi` should stay opinionated, but the opinion should be chosen explicitly through profiles like `raw`, `interactive`, and `persistent`, with targeted overrides rather than hidden all-or-nothing behavior.

### Prompt Context

**User prompt (verbatim):** "Alright, create new tasks to do this refactor, and then task by task, implement it, commit as you go, keep a diary"

**Assistant interpretation:** Create a new ticket for the configurable replapi refactor, write the implementation plan and tasks first, then implement the refactor in documented code slices with commits and diary updates.

**Inferred user intent:** Drive the API redesign in the same disciplined ticket-oriented workflow as the earlier REPL tickets, with enough documentation for review and continuation.

### What I did

- Re-read the current boundaries in:
  - `pkg/replapi/app.go`
  - `pkg/replsession/service.go`
  - `cmd/goja-repl/root.go`
  - `cmd/js-repl/main.go`
- Created ticket `GOJA-23-CONFIGURABLE-REPLAPI`.
- Added the implementation plan and diary docs.
- Replaced the placeholder task list with a phased execution plan for config, policy-driven evaluation, adoption, tests, and commit bookkeeping.

### Why

- The refactor needs an explicit target shape before code changes begin.
- The main risk is accidental API sprawl, so the design must be written down first.

### What worked

- The prior tickets already established a good naming and ticket structure, so the new ticket could follow the same pattern cleanly.
- The current codebase is already factored enough that the key seam is visible: `replapi` construction and `replsession` evaluation policy.

### What didn't work

- The generated ticket/docs were placeholders only and had to be fully rewritten before they were useful.

### What I learned

- The essential design problem is not "how do we expose more knobs" but "how do we expose coherent policy bundles without forcing all consumers into persistence mode."

### What was tricky to build

- The main tricky part at the planning stage was keeping the proposal concrete enough for implementation while avoiding premature overdesign around every possible future profile.

### What warrants a second pair of eyes

- The exact default semantics for the `interactive` profile, especially how much instrumentation it should enable by default.

### What should be done in the future

- Implement the refactor in narrow slices so the transition from implicit behavior to explicit policy is easy to review.

### Code review instructions

- Start with the GOJA-23 design doc and task list.
- Then compare the current `replapi` constructor with the planned profile-based API.

### Technical details

Commands used during this step:

```bash
docmgr ticket create-ticket --ticket GOJA-23-CONFIGURABLE-REPLAPI --title "Configurable replapi profiles and policies" --topics persistent-repl,architecture,api,refactor
docmgr doc add --ticket GOJA-23-CONFIGURABLE-REPLAPI --doc-type design-doc --title "Configurable replapi profiles and policies implementation plan"
docmgr doc add --ticket GOJA-23-CONFIGURABLE-REPLAPI --doc-type reference --title "Diary"
sed -n '1,220p' pkg/replapi/app.go
sed -n '1,420p' pkg/replsession/service.go
sed -n '1,220p' cmd/js-repl/main.go
sed -n '1,240p' cmd/goja-repl/root.go
```

## Step 2: Profile-Based API and Policy-Driven Session Kernel

The first code slice converted the refactor from a sketch into a usable API. `replapi` now has explicit profiles and config helpers, while `replsession` now carries per-session policy so evaluation can run in either raw or instrumented mode. The important part is not the enum names; it is that the behavior is now explicit and durable sessions remember their own profile/policy through `metadata_json`.

This slice also kept the persistent path intact. `goja-repl` now chooses the persistent profile explicitly instead of getting the old behavior by accident, and the tests cover the key combinations: persistent restore, raw without a store, and per-session override from a persistent app down to a raw session.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Implement the configurable replapi surface first, then refactor replsession and the persistent CLI/server adoption onto that new policy model.

**Inferred user intent:** Replace the hard-wired persistent-only replapi with a coherent API that can scale from direct Goja execution to the full persistent REPL stack without exposing an incoherent flag matrix.

**Commit (code):** `de8a47d` — `Refactor replapi around profiles and session policies`

### What I did

- Added `pkg/replsession/policy.go` with:
  - `EvalMode`
  - `EvalPolicy`
  - `ObservePolicy`
  - `PersistPolicy`
  - `SessionPolicy`
  - `SessionOptions`
  - profile helpers for raw, interactive, and persistent behavior
- Extended `pkg/replsession/types.go` so session summaries expose:
  - `profile`
  - `policy`
- Refactored `pkg/replsession/service.go` to:
  - create sessions with explicit session options,
  - persist profile/policy into `sessions.metadata_json`,
  - restore sessions using persisted metadata,
  - split evaluation into raw and instrumented paths,
  - gate console capture, doc sentinels, runtime snapshots, binding tracking, and durable writes by policy.
- Added `pkg/replapi/config.go` with:
  - `Profile`
  - `Config`
  - `SessionOptions`
  - `WithProfile`
  - `WithStore`
  - `WithAutoRestore`
  - `WithDefaultSessionOptions`
  - `WithDefaultSessionPolicy`
  - `RawConfig`
  - `InteractiveConfig`
  - `PersistentConfig`
- Replaced `pkg/replapi/app.go` with a config-driven constructor and per-session overrides.
- Updated `cmd/goja-repl/root.go` so the durable CLI opts into `ProfilePersistent` explicitly.
- Added focused tests in:
  - `pkg/replapi/app_test.go`
  - `pkg/replsession/service_policy_test.go`
  - `pkg/replhttp/handler_test.go`
- Ran focused validation first:
  - `go test ./pkg/replsession ./pkg/replapi ./pkg/replhttp ./cmd/goja-repl`
- Then ran full validation before the commit:
  - `golangci-lint run -v`
  - `go test ./...`

### Why

- The old `replapi.New(...)` shape forced persistence and restore on every caller.
- The evaluation pipeline in `replsession` needed an explicit raw path so callers worried about transformation side effects are not forced through the rewrite machinery.
- Durable sessions need to remember how they were created so restore does not silently drift when the app default changes.

### What worked

- Reusing the existing `sessions.metadata_json` field avoided a schema migration while still preserving session profile/policy across restore.
- The policy split made the code more coherent than expected: execution, observation, and persistence now have obvious gates.
- The persistent profile preserved existing `goja-repl` behavior while making that behavior explicit in code.

### What didn't work

- I initially ran `go fmt ./go-go-goja/pkg/replsession ./go-go-goja/pkg/replapi ./go-go-goja/pkg/replhttp ./go-go-goja/cmd/goja-repl` from `/home/manuel/workspaces/2026-04-03/js-repl-smailnail`, which failed with:

```text
go: go.mod file not found in current directory or any parent directory; see 'go help modules'
```

- I reran formatting from `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja`, which resolved it cleanly.

### What I learned

- The existing persistent design was already close to a policy-driven shape; the missing part was making those policies explicit and durable.
- A config object plus per-session override is enough flexibility here. More constructor booleans would have been worse than the original design.

### What was tricky to build

- The sharpest edge was restore. Temporary replay must not durably write while rebuilding the live runtime, but the restored live session must still end up with its original persistent policy attached. The implementation solves that by replaying with persistence disabled and then transferring the rebuilt state back onto the final session identity and policy.

### What warrants a second pair of eyes

- Whether the raw-mode promise behavior is the right default when top-level await support is disabled.
- Whether the current profile set is complete enough, or if a future “forensics” or “analysis-heavy interactive” profile is worth introducing.

### What should be done in the future

- Migrate the Bubble Tea `cmd/js-repl` path onto the new policy-aware session kernel in its own ticket instead of stretching this refactor further.

### Code review instructions

- Start with:
  - `pkg/replapi/config.go`
  - `pkg/replapi/app.go`
  - `pkg/replsession/policy.go`
  - `pkg/replsession/service.go`
- Then validate the important profile paths with:
  - `go test ./pkg/replsession ./pkg/replapi ./pkg/replhttp ./cmd/goja-repl`
- Then review the explicit persistent adoption in:
  - `cmd/goja-repl/root.go`

### Technical details

Focused commands:

```bash
go test ./pkg/replsession ./pkg/replapi ./pkg/replhttp ./cmd/goja-repl
go fmt ./pkg/replsession ./pkg/replapi ./pkg/replhttp ./cmd/goja-repl
golangci-lint run -v
go test ./...
```

## Step 3: Adopt the Interactive Profile in the Traditional Line REPL

After the core API refactor landed, I used the simpler line-based REPL as the first non-persistent downstream consumer. This is a better proving ground than the Bubble Tea TUI because it exercises real multi-cell interactive behavior without entangling completion and help-drawer concerns. The result is that `cmd/repl` no longer drives `goja.Runtime` directly in interactive mode; it now goes through `replapi` with the interactive profile.

That adoption is intentionally narrow. File-based script execution still uses the direct runtime path, while the interactive loop now creates one interactive profile session and evaluates each submitted line through the shared session kernel.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Prove the new profile API against at least one traditional non-persistent REPL consumer, not just the new persistent CLI.

**Inferred user intent:** Ensure the new API is actually ergonomic for non-persistent interactive use, instead of only working well for the store-backed server path.

**Commit (code):** `d4fa0b5` — `Adopt interactive replapi profile in cmd/repl`

### What I did

- Updated `cmd/repl/main.go` so interactive mode now:
  - constructs `replapi.New(..., replapi.WithProfile(replapi.ProfileInteractive))`,
  - creates one session for the REPL loop,
  - routes each entered line through `app.Evaluate(...)`,
  - prints captured console output and final cell results from the shared response shape.
- Left script-file execution on the direct runtime path so this slice only changes interactive behavior.
- Ran targeted validation:
  - `go test ./cmd/repl ./pkg/replapi ./pkg/replsession`
- Pre-commit on the command commit also reran:
  - `golangci-lint run -v`
  - `go generate ./...`
  - `go test ./...`

### Why

- `cmd/repl` is the cheapest real consumer of the new interactive profile.
- Moving one concrete interactive client onto `replapi` proves the API is usable beyond the persistent server/CLI path.

### What worked

- The integration was small because `replapi` now exposes the right level of abstraction: create a session, evaluate code, render the returned execution fields.
- The line REPL benefits immediately from the shared interactive semantics without introducing persistence requirements.

### What didn't work

- Nothing substantial failed in this slice once the core refactor had landed.

### What I learned

- `cmd/repl` is a good intermediate adoption point before tackling the Bubble Tea TUI, which has more UI-specific coupling around completion/help behavior.

### What was tricky to build

- The main subtlety was keeping script-file execution untouched while only switching the interactive loop to the new session API. That avoids conflating "module/script execution" semantics with "interactive session" semantics in one patch.

### What warrants a second pair of eyes

- Whether console output in `cmd/repl` should eventually include stream labels for `warn`/`error`, or whether the current plain message rendering is sufficient.

### What should be done in the future

- Move `cmd/js-repl` onto `replapi` in a separate ticket, likely by splitting execution/session ownership from completion/help providers.

### Code review instructions

- Start with `cmd/repl/main.go`.
- Compare the old direct `vm.RunString(...)` loop with the new `replapi`-backed loop.
- Validate with:
  - `go test ./cmd/repl ./pkg/replapi ./pkg/replsession`

### Technical details

Focused commands:

```bash
go fmt ./cmd/repl
go test ./cmd/repl ./pkg/replapi ./pkg/replsession
```

## Related

- `../design-doc/01-configurable-replapi-profiles-and-policies-implementation-plan.md`
