---
Title: Investigation diary
Ticket: GOJA-063
Status: active
Topics:
    - xgoja
    - geppetto
    - javascript
    - debugging
    - crash
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ttmp/2026/06/04/GOJA-063--investigate-generated-xgoja-geppetto-session-construction-panic/analysis/01-session-construction-panic-analysis.md
      Note: Living analysis document created for GOJA-063.
    - Path: ttmp/2026/06/04/GOJA-063--investigate-generated-xgoja-geppetto-session-construction-panic/scripts/01-reproduce-session-construction-panic.sh
      Note: |-
        Repro script for the generated xgoja no-inference Geppetto session construction panic.
        Generated xgoja crash repro script
    - Path: ttmp/2026/06/04/GOJA-063--investigate-generated-xgoja-geppetto-session-construction-panic/various/01-stack-repro.log
      Note: |-
        Captured stack trace from temporary runtimeowner panic-recovery instrumentation.
        Captured stack trace from temporary runtimeowner instrumentation
ExternalSources: []
Summary: Chronological diary for GOJA-063, the investigation of a generated xgoja Geppetto agent/session construction panic.
LastUpdated: 2026-06-04T18:45:00-04:00
WhatFor: Use to continue the crash investigation with exact commands, findings, and open questions.
WhenToUse: Before changing Geppetto agent/session/profile-resolution code or generated xgoja Pinocchio profile ports.
---


# Diary

## Goal

This diary records the investigation of a generated xgoja crash observed while porting Pinocchio's no-inference Geppetto profile smoke into an xgoja jsverb. It captures the ticket setup, reproduction, stack collection, current root-cause hypothesis, and next steps.

## Step 1: Create GOJA-063 and Capture the First Stack Trace

The investigation started by turning the observed crash into its own ticket so the work does not get buried inside GOJA-053. I created a new docmgr workspace, added an analysis document, added this diary, and wrote a reproducible generated xgoja script that rebuilds the failing scenario from scratch.

The first investigation pass changed the suspected failure site. The user-visible failure appeared when the JavaScript smoke constructed a session, but the captured Go stack shows the first Geppetto frame is `ensureInferenceSettingsProviderDefaults` during `agentBuilderRef.build`. The current hypothesis is that a minimal profile with chat settings but no API settings leaves `InferenceSettings.API` nil, and the defaulting helper dereferences it.

### Prompt Context

**User prompt (verbatim):** "create a new ticket to inestigate the crash, and add a detailed diary as you work, as well as an analysis document that you can flesh out as you make progress."

**Assistant interpretation:** Create a new docmgr ticket dedicated to the generated xgoja Geppetto crash, initialize a detailed diary and analysis document, and begin the investigation by reproducing and characterizing the crash.

**Inferred user intent:** Keep the crash investigation separate, continuation-friendly, and evidence-based, with enough documentation for later debugging and fix work.

### What I did

- Created ticket `GOJA-063`:
  - `ttmp/2026/06/04/GOJA-063--investigate-generated-xgoja-geppetto-session-construction-panic`
- Created analysis document:
  - `analysis/01-session-construction-panic-analysis.md`
- Created diary document:
  - `reference/01-investigation-diary.md`
- Replaced the placeholder task list with concrete investigation/fix tasks.
- Added repro script:
  - `scripts/01-reproduce-session-construction-panic.sh`
- Ran the repro script:
  - `cd go-go-goja && ttmp/2026/06/04/GOJA-063--investigate-generated-xgoja-geppetto-session-construction-panic/scripts/01-reproduce-session-construction-panic.sh`
- Captured the expected crash:
  - `Error: runtimeowner jsverbs.invoke: runtime call panicked: runtime error: invalid memory address or nil pointer dereference`
  - `repro_exit_status=1`
- Temporarily instrumented `pkg/runtimeowner/runner.go` to include `debug.Stack()` in recovered panic errors, ran the repro again, and restored the file.
- Stored the captured stack output in:
  - `various/01-stack-repro.log`
- Updated the analysis document with the stack excerpt, current hypothesis, open questions, and proposed next steps.

### Why

The previous GOJA-053 diary only recorded the panic as an issue discovered while porting Pinocchio scripts. A separate ticket makes it possible to investigate the crash without mixing it into the host-services feature work. The repro script is important because it avoids relying on the currently narrowed committed example and instead recreates the exact failing no-inference path.

### What worked

- `docmgr ticket create-ticket` created a clean GOJA-063 workspace.
- The repro script reliably rebuilds a generated xgoja binary and reproduces the panic.
- Temporary stack instrumentation identified the first Geppetto frame:
  - `geppetto/pkg/js/modules/geppetto/api_engines.go:55`
  - `ensureInferenceSettingsProviderDefaults(...)`
- The stack shows the panic happens during:
  - `agentBuilderRef.build`
  - not first inside `sessionBuilderRef.build`.

### What didn't work

- `runtimeowner` recovered the panic and reported only the recovered panic value by default, so the first repro did not include a Go stack.
- My first attempt to instrument `runtimeowner` imported `runtime/debug` without actually replacing the recovered-panic error line, producing:
  - `pkg/runtimeowner/runner.go:49:3: "runtime/debug" (untyped string constant) is not used`
- The second instrumentation attempt also missed the replacement and produced:
  - `pkg/runtimeowner/runner.go:8:2: "runtime/debug" imported and not used`
- A third, exact string replacement succeeded and produced the stack trace.

### What I learned

- The crash is reproducible outside the committed narrowed example.
- The failing path is not primarily a session builder failure, even though the original symptom appeared when session construction was added.
- `ensureInferenceSettingsProviderDefaults` checks `ss`, `ss.Chat`, and `ss.Chat.ApiType`, but appears to dereference `ss.API.BaseUrls` without guarding `ss.API`.
- The minimal Pinocchio profile fixture defines chat settings but no API block, which likely explains why live profiles work while the no-network smoke panics.

### What was tricky to build

The tricky part was getting a useful stack trace without permanently changing runtimeowner behavior. The runtime owner intentionally recovers panics so generated commands report an error instead of crashing the process, but that makes the stack unavailable. I temporarily patched the recovered-panic error to include `debug.Stack()`, ran the repro, copied the resulting log into the ticket, and restored the source file before continuing.

The other tricky point is naming the failure accurately. The repro JavaScript includes `agent.session().id(...).build()`, but the stack points to `agent().inference(settings).build()`. The ticket title preserves the user-visible symptom, while the analysis document records that the current root-cause hypothesis is agent/profile settings defaulting.

### What warrants a second pair of eyes

- Confirm whether `InferenceSettings.API` is allowed to be nil after profile resolution.
- Confirm whether the correct fix belongs in `ensureInferenceSettingsProviderDefaults` or earlier in profile merge/defaulting.
- Review whether runtimeowner should optionally include stack traces for recovered panics in debug mode.

### What should be done in the future

- Add a focused Geppetto regression test for the nil API settings case.
- Fix the nil guard/allocation around `ensureInferenceSettingsProviderDefaults`.
- Re-run the generated xgoja repro script and confirm it returns JSON.
- Restore the committed `profile-smoke` example to include agent/session construction once the crash is fixed.

### Code review instructions

- Start with `analysis/01-session-construction-panic-analysis.md` for the current hypothesis and stack excerpt.
- Run `scripts/01-reproduce-session-construction-panic.sh` to reproduce the crash.
- Inspect Geppetto files:
  - `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/geppetto/pkg/js/modules/geppetto/api_engines.go`
  - `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/geppetto/pkg/js/modules/geppetto/api_agent.go`
  - `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/geppetto/pkg/js/modules/geppetto/api_session.go`
- After a fix, validate by rerunning the repro script and the generated xgoja Geppetto example smokes.

### Technical details

Repro command:

```bash
cd /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja
ttmp/2026/06/04/GOJA-063--investigate-generated-xgoja-geppetto-session-construction-panic/scripts/01-reproduce-session-construction-panic.sh
```

Key stack excerpt:

```text
github.com/go-go-golems/geppetto/pkg/js/modules/geppetto.ensureInferenceSettingsProviderDefaults(...)
    /home/manuel/workspaces/2026-06-03/goja-runtime-flags/geppetto/pkg/js/modules/geppetto/api_engines.go:55

github.com/go-go-golems/geppetto/pkg/js/modules/geppetto.(*agentBuilderRef).build(...)
    /home/manuel/workspaces/2026-06-03/goja-runtime-flags/geppetto/pkg/js/modules/geppetto/api_agent.go:245 +0xa9
```

Likely failing code shape:

```go
func ensureInferenceSettingsProviderDefaults(ss *aistepssettings.InferenceSettings) {
    if ss == nil || ss.Chat == nil || ss.Chat.ApiType == nil {
        return
    }
    if ss.API.BaseUrls == nil { // likely nil ss.API dereference
        ss.API.BaseUrls = map[string]string{}
    }
    ...
}
```
