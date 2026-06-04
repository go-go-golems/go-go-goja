---
Title: Session construction panic analysis
Ticket: GOJA-063
Status: active
Topics:
    - xgoja
    - geppetto
    - javascript
    - debugging
    - crash
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../geppetto/pkg/js/modules/geppetto/api_agent.go
      Note: '`agentBuilderRef.build` calls `ensureInferenceSettingsProviderDefaults` after resolving inference settings from the profile.'
    - Path: ../../../../../../../../geppetto/pkg/js/modules/geppetto/api_engines.go
      Note: '`ensureInferenceSettingsProviderDefaults` is the first Geppetto frame in the captured panic stack and likely dereferences a nil API settings pointer.'
    - Path: ../../../../../../../../geppetto/pkg/js/modules/geppetto/api_session.go
      Note: Session builder path originally suspected because the JS repro constructs `agent.session().id(...).build()`.
    - Path: ../../../../../../../examples/xgoja/12-geppetto-host-services/verbs/pinocchio_profiles.js
      Note: Generated xgoja Pinocchio profile script port where the exact no-inference session construction path was narrowed after the panic.
    - Path: ../../../../../../../geppetto/pkg/js/modules/geppetto/api_agent.go
      Note: agentBuilderRef.build calls settings defaulting before session construction
    - Path: ../../../../../../../geppetto/pkg/js/modules/geppetto/api_agent_profile_test.go
      Note: Regression coverage for nil API settings and profile-backed agent/session construction (commit 4c975f1b)
    - Path: ../../../../../../../geppetto/pkg/js/modules/geppetto/api_engines.go
      Note: |-
        First Geppetto stack frame; likely nil API settings dereference
        Nil API settings fix in ensureInferenceSettingsProviderDefaults (commit 4c975f1b)
    - Path: ../../../../../../../geppetto/pkg/js/modules/geppetto/api_session.go
      Note: Session builder path originally suspected from the JS repro
    - Path: ../../../../../../../pinocchio/examples/js/profiles/basic.yaml
      Note: Minimal profile fixture that reproduces nil API settings shape
    - Path: examples/xgoja/12-geppetto-host-services/profiles/basic.yaml
      Note: Deterministic generated xgoja profile fixture with dummy API key (commit 95a9c4a)
    - Path: examples/xgoja/12-geppetto-host-services/verbs/pinocchio_profiles.js
      Note: |-
        Generated xgoja profile smoke port where exact session construction was narrowed after the panic
        Restored profile-smoke agent/session construction port (commit 95a9c4a)
    - Path: pkg/engine/options.go
      Note: Engine-facing recovered panic stack option (commit 2a81564)
    - Path: pkg/runtimeowner/runner.go
      Note: Recovered panic stack implementation (commit 2a81564)
    - Path: pkg/runtimeowner/types.go
      Note: IncludePanicStack option for recovered panic diagnostics (commit 2a81564)
    - Path: scripts/01-reproduce-session-construction-panic.sh
      Note: Repro script that builds a temporary generated xgoja binary and runs the exact failing no-inference profile smoke.
    - Path: ttmp/2026/06/04/GOJA-063--investigate-generated-xgoja-geppetto-session-construction-panic/various/02-after-nil-api-fix-repro.log
      Note: Exact minimal repro after nil API fix; now normal missing-key error
    - Path: ttmp/2026/06/04/GOJA-063--investigate-generated-xgoja-geppetto-session-construction-panic/various/03-restored-profile-smoke.log
      Note: Restored generated xgoja profile-smoke output
    - Path: various/01-stack-repro.log
      Note: Captured reproduction output with temporary runtimeowner stack instrumentation.
ExternalSources: []
Summary: 'Initial GOJA-063 analysis: the generated xgoja no-inference Pinocchio profile smoke panic is reproducible and the first Geppetto frame is `ensureInferenceSettingsProviderDefaults`, not session builder construction itself. The likely cause is a nil `InferenceSettings.API` field when the profile defines `chat.api_type` but omits an `api` block.'
LastUpdated: 2026-06-04T18:45:00-04:00
WhatFor: Use when investigating or fixing the generated xgoja Geppetto no-inference session-construction panic.
WhenToUse: Before changing Geppetto profile resolution, agent builder, or generated xgoja Pinocchio profile script ports.
---




# Session construction panic analysis

## Executive summary

GOJA-063 tracks a crash observed while porting Pinocchio's deterministic JavaScript profile smoke into a generated xgoja jsverb. The failing JavaScript resolves a Geppetto profile, builds an agent from the resolved settings, and builds a session without running inference. The generated binary reports a Go nil-pointer panic through `runtimeowner`:

```text
Error: runtimeowner jsverbs.invoke: runtime call panicked: runtime error: invalid memory address or nil pointer dereference
```

The initial suspicion was the session builder, because the failure appeared while adding `agent.session().id(...).build()` to the generated xgoja port. A captured stack trace shows the first Geppetto frame is earlier: `ensureInferenceSettingsProviderDefaults` in `geppetto/pkg/js/modules/geppetto/api_engines.go`. The panic occurs during `agentBuilderRef.build`, before the session builder has a chance to run.

The current likely root cause is a nil `InferenceSettings.API` field. The minimal Pinocchio example profile at `pinocchio/examples/js/profiles/basic.yaml` defines chat settings such as `api_type: openai` and `engine: gpt-5-mini`, but it does not include an `api` block. `ensureInferenceSettingsProviderDefaults` checks `ss == nil`, `ss.Chat == nil`, and `ss.Chat.ApiType == nil`, but then dereferences `ss.API.BaseUrls` without checking whether `ss.API` itself is nil.

## Reproduction

The repro script is tracked at:

```text
ttmp/2026/06/04/GOJA-063--investigate-generated-xgoja-geppetto-session-construction-panic/scripts/01-reproduce-session-construction-panic.sh
```

It creates a temporary generated xgoja project under:

```text
/tmp/xgoja-geppetto-session-panic-repro
```

The generated project imports only the Geppetto xgoja provider and embeds a jsverb equivalent to the exact no-inference Pinocchio smoke path:

```javascript
__package__({ name: "pinocchio" });

function profileSmoke(sessionId) {
  const gp = require("geppetto");
  const settings = gp.inferenceProfiles.resolve();
  const snapshot = settings.toJSON();
  const agent = gp.agent()
    .name("xgoja-pinocchio-profile-smoke")
    .inference(settings)
    .build();
  const session = agent.session().id(sessionId).build();

  return {
    profile: snapshot.provenance?.profileSlug || "",
    registry: snapshot.provenance?.registrySlug || "",
    model: snapshot.chat?.engine || "",
    apiType: snapshot.chat?.api_type || "",
    session: session.id(),
    hasSessionNext: typeof session.next === "function",
  };
}
```

The command run by the script is:

```bash
/tmp/xgoja-geppetto-session-panic-repro/geppetto-session-panic-repro \
  verbs pinocchio profile-smoke xgoja-pinocchio-profile-smoke \
  --profile-registries /home/manuel/workspaces/2026-06-03/goja-runtime-flags/pinocchio/examples/js/profiles/basic.yaml \
  --profile assistant \
  --output json
```

The repro currently exits with status 1 and prints:

```text
Error: runtimeowner jsverbs.invoke: runtime call panicked: runtime error: invalid memory address or nil pointer dereference
repro_exit_status=1
```

## Stack trace evidence

`runtimeowner` intentionally recovers panics and reports only the recovered value. To locate the crash, I temporarily instrumented `pkg/runtimeowner/runner.go` to append `debug.Stack()` to recovered panic errors, ran the repro, and restored the file. The captured output is stored at:

```text
various/01-stack-repro.log
```

The key stack excerpt is:

```text
panic({0x1de8ac0?, 0x364a390?})
    runtime/panic.go:860

github.com/go-go-golems/geppetto/pkg/js/modules/geppetto.ensureInferenceSettingsProviderDefaults(...)
    geppetto/pkg/js/modules/geppetto/api_engines.go:55

github.com/go-go-golems/geppetto/pkg/js/modules/geppetto.(*agentBuilderRef).build(...)
    geppetto/pkg/js/modules/geppetto/api_agent.go:245 +0xa9

github.com/go-go-golems/geppetto/pkg/js/modules/geppetto.(*moduleRuntime).newAgentBuilderObject.func15(...)
    geppetto/pkg/js/modules/geppetto/api_agent.go:229 +0x25
```

This moves the investigation away from `sessionBuilderRef.build` as the first failing function. The session builder remains relevant because the user-visible repro includes session construction, but the actual nil dereference happens while building the agent.

## Relevant code

The likely problem is in `api_engines.go`:

```go
func ensureInferenceSettingsProviderDefaults(ss *aistepssettings.InferenceSettings) {
    if ss == nil || ss.Chat == nil || ss.Chat.ApiType == nil {
        return
    }
    if ss.API.BaseUrls == nil {
        ss.API.BaseUrls = map[string]string{}
    }
    if *ss.Chat.ApiType == aitypes.ApiTypeClaude {
        if _, ok := ss.API.BaseUrls["claude-base-url"]; !ok {
            ss.API.BaseUrls["claude-base-url"] = "https://api.anthropic.com"
        }
    }
}
```

The guard checks `ss`, `ss.Chat`, and `ss.Chat.ApiType`, but it does not check `ss.API`. If a profile has chat settings but omits the API settings block, `ss.API.BaseUrls` can panic.

The failing minimal profile is:

```yaml
slug: workspace
profiles:
  default:
    slug: default
    display_name: Workspace Default
    inference_settings:
      chat:
        api_type: openai
        engine: gpt-4o-mini
  assistant:
    slug: assistant
    display_name: Assistant
    stack:
      - profile_slug: default
    inference_settings:
      chat:
        engine: gpt-5-mini
```

This profile is valid enough for profile resolution and snapshot output. It should not cause a Go panic when used to build an agent.

## Current hypothesis

The current hypothesis is:

1. `gp.inferenceProfiles.resolve()` returns a resolved profile with `InferenceSettings.Chat` populated.
2. The profile's `InferenceSettings.API` pointer is nil because the YAML profile omits an `api` section.
3. `agentBuilderRef.build` calls `ensureInferenceSettingsProviderDefaults` on the resolved settings.
4. `ensureInferenceSettingsProviderDefaults` dereferences `ss.API.BaseUrls` without allocating or guarding `ss.API`.
5. Go panics with a nil pointer dereference.
6. `runtimeowner` recovers the panic and reports it as `runtime call panicked`, losing the Geppetto stack unless instrumented.

The likely fix is to make `ensureInferenceSettingsProviderDefaults` nil-safe around the API settings field. The exact code depends on the concrete type of `InferenceSettings.API`: if it is a pointer, allocate the API settings struct before accessing `BaseUrls`; if it is a value with a pointer-typed interior, guard that interior. The next step is to inspect `geppetto/pkg/steps/ai/settings` and add a focused regression test.

## Why the live xgoja examples still work

The crash is not a general generated xgoja session failure. These paths worked during GOJA-053 follow-up validation:

- `examples/xgoja/12-geppetto-host-services` `demo run` creates a session, runs inference, persists a turn, and writes events.
- `verbs pinocchio profile-demo` creates an agent/session and runs live inference against `gpt-5-nano`.

Those live profiles likely include enough API settings, or they flow through a settings shape where `ss.API` is non-nil. The failing case is the minimal no-network Pinocchio example profile that resolves chat settings without an API block.

## Open questions

- Is `InferenceSettings.API` supposed to be non-nil after profile resolution, or should consumers treat it as optional?
- Should `profiles.MergeInferenceSettings` allocate missing nested settings structs, or should `ensureInferenceSettingsProviderDefaults` be defensive?
- Why did the failure first appear when adding session construction if the stack shows agent build as the actual panic site? One possibility is that the earlier narrowed port did not call `agent().inference(settings).build()`, so it never reached `agentBuilderRef.build`.
- Should runtimeowner include optional stack traces for recovered panics in debug mode?

## Proposed next steps

1. Inspect `geppetto/pkg/steps/ai/settings` to confirm the type and allocation expectations for `InferenceSettings.API`.
2. Add a Geppetto unit test that builds an agent from resolved settings where `Chat.ApiType` is set but API settings are absent.
3. Fix `ensureInferenceSettingsProviderDefaults` or the profile merge/defaulting path.
4. Re-run `scripts/01-reproduce-session-construction-panic.sh` and confirm JSON output.
5. Restore the deterministic generated xgoja `profile-smoke` port to include agent and session construction.
6. Re-run the live generated xgoja host-services smoke and Pinocchio profile demo port.

## Follow-up: nil API panic fixed, exact minimal repro now returns a normal missing-key error

The first fix landed in Geppetto commit `4c975f1b` (`Fix geppetto profile agent build nil API panic`). The code change makes `ensureInferenceSettingsProviderDefaults` allocate API settings when profile resolution produces chat settings but leaves `InferenceSettings.API` nil:

```go
func ensureInferenceSettingsProviderDefaults(ss *aistepssettings.InferenceSettings) {
    if ss == nil || ss.Chat == nil || ss.Chat.ApiType == nil {
        return
    }
    if ss.API == nil {
        ss.API = aistepssettings.NewAPISettings()
    }
    if ss.API.BaseUrls == nil {
        ss.API.BaseUrls = map[string]string{}
    }
    ...
}
```

The fix has two regression tests:

1. `TestEnsureInferenceSettingsProviderDefaultsHandlesMissingAPISettings` exercises the nil `API` shape directly.
2. `TestAgentSessionBuildFromProfileWithAPISettings` loads a profile registry, resolves `assistant`, builds an agent from the resolved settings, and constructs a session without running inference.

Validation in Geppetto:

```bash
cd geppetto
go test ./pkg/js/modules/geppetto -run 'TestEnsureInferenceSettingsProviderDefaultsHandlesMissingAPISettings|TestAgentSessionBuildFromProfileWithAPISettings' -count=1
go test ./pkg/js/modules/geppetto ./pkg/js/modules/geppetto/provider -count=1
```

Both commands passed.

### What changed in the exact generated xgoja repro

The exact generated xgoja repro still uses `pinocchio/examples/js/profiles/basic.yaml`, which intentionally has no API key block. After the nil API fix, it no longer panics. It now fails as a normal JavaScript/Go error from engine construction:

```text
Error: GoError: invalid settings for provider openai: missing API key openai-api-key at github.com/go-go-golems/geppetto/pkg/js/modules/geppetto.(*moduleRuntime).newAgentBuilderObject.func15 (native)
repro_exit_status=1
```

This output is stored in:

```text
various/02-after-nil-api-fix-repro.log
```

This is a meaningful improvement. The original bug was a Go nil-pointer panic. The current behavior is a normal provider settings error: the minimal profile has enough data to resolve a model, but not enough data to create an OpenAI engine.

### Restored generated xgoja profile-smoke session construction

The committed generated xgoja example now includes its own local deterministic profile fixture:

```text
examples/xgoja/12-geppetto-host-services/profiles/basic.yaml
```

That fixture is based on the Pinocchio example profile shape but includes a dummy `openai-api-key` value. The smoke does not call the model, so the key is not used over the network. It only allows `enginefactory.NewEngineFromSettings` to construct an engine during `agent().inference(settings).build()`.

The generated xgoja `profile-smoke` port now again builds an agent and session:

```javascript
const settings = gp.inferenceProfiles.resolve();
const snapshot = settings.toJSON();
const agent = gp.agent()
  .name("xgoja-pinocchio-profile-smoke")
  .inference(settings)
  .build();
const session = agent.session().id(sessionId).build();
```

Validation:

```bash
cd go-go-goja/examples/xgoja/12-geppetto-host-services
make pinocchio-smoke
```

Output included:

```json
[
  {
    "apiType": "openai",
    "hasSessionNext": true,
    "migration": "pure-geppetto-session-construction",
    "model": "gpt-5-mini",
    "profile": "assistant",
    "registry": "workspace",
    "session": "xgoja-pinocchio-profile-smoke",
    "source": "pinocchio/examples/js/runner-profile-smoke.js"
  }
]
```

This output is summarized in:

```text
various/03-restored-profile-smoke.log
```

### Live smoke regression check

The live Geppetto host-services smoke still passes after the fix and example change:

```bash
cd go-go-goja/examples/xgoja/12-geppetto-host-services
make live-smoke SESSION=xgoja-geppetto-fixed-1780613783 PROFILE_REGISTRIES="$HOME/.config/pinocchio/profiles.yaml" PROFILE=gpt-5-nano
```

It produced:

```json
[
  {
    "latestText": "hosted",
    "listed": 1,
    "sessionId": "xgoja-geppetto-fixed-1780613783",
    "systemText": "Answer with exactly the word: hosted",
    "text": "hosted",
    "toolCount": 4
  }
]
```

SQLite verification showed one persisted final turn, and JSONL verification showed eight event rows.

## Updated conclusion

The original crash is fixed. The nil-pointer panic came from `ensureInferenceSettingsProviderDefaults` assuming `InferenceSettings.API` was non-nil. After the fix, missing API settings are initialized before provider defaulting touches `BaseUrls`.

There remains a semantic distinction between profile resolution and engine construction. A profile registry can resolve a profile whose chat settings name an OpenAI model but whose API settings lack an API key. Profile resolution can still succeed, but agent construction from those settings correctly fails because the OpenAI engine cannot be created without `openai-api-key`. Generated xgoja deterministic session-construction smokes should therefore use a fixture with a dummy key when they need to build an engine without calling the network.

## Follow-up: runtimeowner recovered panic stack traces are now opt-in

GOJA-063 also produced one tooling improvement in go-go-goja: recovered runtimeowner panic errors can now include a Go debug stack when explicitly enabled. The implementation landed in go-go-goja commit `2a81564` (`Add opt-in runtimeowner panic stacks`).

The new low-level runtimeowner option is:

```go
type Options struct {
    Name              string
    MaxWait           int64
    RecoverPanics     bool
    IncludePanicStack bool
}
```

When `RecoverPanics` and `IncludePanicStack` are both enabled, the recovered panic error appends `runtime/debug.Stack()`:

```go
runtimeowner.NewRuntimeOwner(vm, scheduler, runtimeowner.Options{
    RecoverPanics:     true,
    IncludePanicStack: true,
})
```

The engine layer exposes this as a builder option:

```go
factory, err := engine.NewRuntimeFactoryBuilder(
    engine.WithRecoveredPanicStack(true),
).Build()
```

This keeps normal generated CLI output concise while making provider panic diagnosis faster in test fixtures, debugging tools, or explicitly diagnostic generated runtimes. The option is intentionally not enabled by default because stacks are noisy and include local paths.

Validation:

```bash
cd go-go-goja
go test ./pkg/runtimeowner ./pkg/engine -count=1
```

The commit hook also ran `golangci-lint`, `go vet` with the Glazed linter, `go generate ./...`, and `go test ./...` successfully while committing `2a81564`.
