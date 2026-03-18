---
Title: Diary
Ticket: GOJA-08-HASHICORP-PLUGINS--add-hashicorp-plugin-support-for-runtime-module-registration
Status: active
Topics:
    - goja
    - go
    - js-bindings
    - architecture
    - security
    - tooling
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: engine/factory.go
      Note: |-
        Current factory stores one built require registry and therefore shapes the plugin design
        Factory design constraint discovered during investigation
    - Path: engine/module_specs.go
      Note: Existing static module registration interface inspected for extension points
    - Path: engine/runtime.go
      Note: |-
        Current runtime close logic inspected for cleanup-hook gaps
        Runtime cleanup gap recorded in diary
    - Path: modules/common.go
      Note: |-
        Global default registry semantics inspected for singleton-state implications
        Global registry behavior recorded in diary
    - Path: modules/database/database.go
      Note: Stateful module example used to reason about runtime scoping
    - Path: pkg/repl/evaluators/javascript/evaluator.go
      Note: |-
        Existing owned runtime consumer inspected for future plugin configuration wiring
        Runtime consumer inspected during design
    - Path: pkg/runtimeowner/runner.go
      Note: |-
        Owner-goroutine execution contract inspected for RPC call integration
        Owner-goroutine execution model recorded in diary
    - Path: ttmp/2026/03/18/GOJA-08-HASHICORP-PLUGINS--add-hashicorp-plugin-support-for-runtime-module-registration--add-hashicorp-plugin-support-for-runtime-module-registration/sources/local/Imported goja plugins note.md
      Note: |-
        Imported source that seeded the design analysis
        Imported source tracked in diary context
ExternalSources:
    - local:Imported goja plugins note.md
Summary: Chronological record of how the GOJA-08 design ticket was created, investigated, and written.
LastUpdated: 2026-03-18T09:14:54.586210258-04:00
WhatFor: Capture the investigation steps, decisions, and caveats behind the GOJA-08 plugin-support design ticket.
WhenToUse: Use when reviewing the design ticket or continuing the implementation work later.
---


# Diary

## Goal

Document how the HashiCorp plugin-support design ticket was created, what repository evidence shaped the proposal, and what an implementer should review first.

## Step 1: Create GOJA-08 and turn the imported memo into a repo-grounded plan

This step created the ticket workspace, imported `/tmp/goja-plugins.md`, and then inspected the code paths that actually define runtime composition in `go-go-goja`. The imported note was a useful architectural starting point, but it assumed a cleaner boundary than this repository currently exposes. In particular, the repo still builds one frozen `require.Registry` in `engine.Factory`, uses a package-global module registry, and has no general-purpose runtime cleanup hook beyond `Owner.Shutdown(...)` and `Loop.Stop()`.

The final design therefore did not just restate the imported memo. It translated the memo into the repository’s vocabulary and extension seams: `ModuleSpec`, `RuntimeInitializer`, `runtimeowner.Runner`, CommonJS `require`, and the current REPL/evaluator entrypoints. The key new design choice is to add a runtime-scoped module-registration seam so plugin subprocesses can be discovered, validated, reified, and shut down per runtime instead of leaking through global singletons.

### Prompt Context

**User prompt (verbatim):**

```text
Create a new ticket to add hashicorp plugin support to go-go-goja, so that we can register plugins into a runtime.

Import the file /tmp/goja-plugins.md with docmgr import file --file /tmp/goja-plugins.md --ticket ... and then read it in detail, and transform it into your own analysis and plan. You can run experiments in scripts/ if you want.

Create a detailed analysis / design / implementation guide that is very detailed for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file
  references.
  It should be very clear and detailed. Store in the ticket and the nupload to remarkable.
```

**Assistant interpretation:** Create a new docmgr ticket in `go-go-goja`, import the provided source note, derive a repository-specific design for HashiCorp plugin support, record the work in a diary, and deliver the ticket to reMarkable.

**Inferred user intent:** Produce an implementation-ready architecture guide, not a speculative brainstorm, so a new engineer can add plugin-backed modules to `go-go-goja` without rediscovering the runtime and lifecycle constraints.

**Commit (code):** N/A

### What I did

- Created ticket `GOJA-08-HASHICORP-PLUGINS--add-hashicorp-plugin-support-for-runtime-module-registration`.
- Added a design doc and diary document to the ticket.
- Imported `/tmp/goja-plugins.md` into `sources/local/Imported goja plugins note.md`.
- Inspected the current engine, module registry, runtime owner, REPL, evaluator, and jsverbs runtime usage paths.
- Wrote a new design guide that keeps the imported note’s valid boundary assumptions but changes the repo integration plan.

### Why

- The user asked for a ticketed deliverable, not just an answer in chat.
- The imported memo contained strong ideas, but the implementation plan needed to reflect the current repository structure and constraints.
- The runtime and lifecycle details in this repo are subtle enough that an intern would otherwise make avoidable mistakes.

### What worked

- `docmgr` ticket creation, doc creation, and file import all worked without extra setup.
- The repo had enough existing runtime composition code to anchor the design in concrete files.
- The imported memo aligned with the most important upstream truth: keep `goja.Runtime` in the host and use plugins only for discovery/RPC/process isolation.

### What didn't work

- An early file read targeted the wrong runtimeowner path:

```text
sed: can't read pkg/runtimeowner/runtimeowner.go: No such file or directory
```

- Resolution was straightforward after listing `pkg/runtimeowner/` and reading `runner.go` plus `types.go`.

### What I learned

- `engine.Factory` currently stores a single built `*require.Registry`, which makes runtime-scoped module registration harder than the imported memo suggests.
- `modules.DefaultRegistry` stores singleton module receivers, so stateful modules like `DBModule` can leak state expectations across runtimes if the design is not careful.
- `runtimeowner.Runner` is the repo’s critical concurrency guard; any plugin-backed function that touches Goja values must respect that owner context.

### What was tricky to build

- The sharpest design issue was lifecycle ownership. The obvious first draft is “discover plugins in `Build()` and register their loaders into the shared registry.” That is attractive because it matches the existing `ModuleSpec` flow, but it quietly couples plugin subprocess lifetime to the factory rather than to each runtime. In this repo, `Runtime.Close()` only shuts down the runtime owner and loop, so a factory-scoped plugin client design would either leak subprocesses or require an awkward second cleanup system outside the runtime lifecycle. The proposed solution is to add a runtime-scoped registration phase and runtime cleanup hooks.

### What warrants a second pair of eyes

- Whether the factory refactor should rebuild a new `require.Registry` per runtime or introduce a narrower pre-enable extension seam.
- Whether plugin module names should use `plugin:<name>` or `plugin/<name>` as the canonical namespace.
- Whether protobuf codegen should live in-repo or be generated by a ticket-local script before being committed.

### What should be done in the future

- Implement the engine refactor and runtime cleanup hooks first.
- Add the HashiCorp plugin host, manifest contract, and runtime module registrar next.
- Finish with integration tests, example plugins, and entrypoint wiring for the REPL/evaluator.

### Code review instructions

- Start with the design doc in `design-doc/01-hashicorp-plugin-support-for-go-go-goja-architecture-and-implementation-guide.md`.
- Compare its recommendations against `engine/factory.go`, `engine/module_specs.go`, `engine/runtime.go`, `modules/common.go`, and `pkg/runtimeowner/runner.go`.
- Validate that the imported memo was interpreted rather than copied by reading `sources/local/Imported goja plugins note.md` after the design doc.

### Technical details

- Commands run:
  - `docmgr status --summary-only`
  - `docmgr ticket create-ticket --ticket GOJA-08-HASHICORP-PLUGINS--add-hashicorp-plugin-support-for-runtime-module-registration --title "Add HashiCorp plugin support for runtime module registration" --topics goja,go,js-bindings,architecture,security,tooling`
  - `docmgr doc add --ticket GOJA-08-HASHICORP-PLUGINS--add-hashicorp-plugin-support-for-runtime-module-registration --doc-type design-doc --title "HashiCorp plugin support for go-go-goja architecture and implementation guide"`
  - `docmgr doc add --ticket GOJA-08-HASHICORP-PLUGINS--add-hashicorp-plugin-support-for-runtime-module-registration --doc-type reference --title "Diary"`
  - `docmgr import file --file /tmp/goja-plugins.md --ticket GOJA-08-HASHICORP-PLUGINS--add-hashicorp-plugin-support-for-runtime-module-registration --name "Imported goja plugins note"`
  - `rg -n "hashicorp/go-plugin|go-plugin|plugin.NewClient|plugin.Serve|Discover\\(" -S .`
  - `nl -ba engine/factory.go | sed -n '1,220p'`
  - `nl -ba engine/module_specs.go | sed -n '1,220p'`
  - `nl -ba pkg/runtimeowner/runner.go | sed -n '1,260p'`
  - `nl -ba pkg/repl/evaluators/javascript/evaluator.go | sed -n '70,330p'`

## Step 2: Refactor engine lifecycle for runtime-scoped plugin registration

This step implemented the foundational engine changes from the design doc. The factory no longer stores a single built `require.Registry`. Instead, it keeps the build plan and creates a fresh registry inside `NewRuntime()`, which makes runtime-specific module injection possible without sharing synthetic module state across runtime instances.

I also added runtime close hooks and a new runtime-scoped module registrar interface. Those changes are small on the surface, but they close the two lifecycle gaps that would have made plugin support brittle: no pre-enable registration seam and no general-purpose cleanup seam. The new tests prove that the seams behave the way the later plugin host needs them to behave.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Convert the design ticket into implementation work, execute phases in order, keep the task list current, and commit work in reviewable chunks.

**Inferred user intent:** Make concrete code progress while preserving traceability in the ticket.

**Commit (code):** `d50da08` — `engine: add runtime-scoped module registrars`

### What I did

- Added `engine/runtime_modules.go` with `RuntimeModuleRegistrar` and `RuntimeModuleContext`.
- Refactored `engine/factory.go` so `Factory` stores build inputs and rebuilds the `require.Registry` per runtime.
- Added a runtime-registrar phase before `reg.Enable(vm)`.
- Extended `engine.Runtime` with `AddCloser(...)` and reverse-order cleanup execution.
- Added `engine/runtime_modules_test.go` covering per-runtime registration and close-hook behavior.

### Why

- Plugin-backed modules need per-runtime registration state.
- Plugin subprocesses need to shut down through the normal runtime lifecycle.
- The plugin layer should reuse general engine seams rather than own lifecycle logic privately.

### What worked

- The public builder/runtime flow stayed the same for callers.
- The new tests captured the intended lifecycle behavior with a small synthetic registrar.
- Local tool availability is sufficient for the next gRPC-focused phase: `protoc`, `protoc-gen-go`, and `protoc-gen-go-grpc` are installed.

### What didn't work

- The first compile pass left stale references in `engine/factory.go`:

```text
engine/factory.go:170:14: f.registry undefined (type *Factory has no field or method registry)
engine/factory.go:199:9: no new variables on left side of :=
```

- Running tests through the workspace default failed because `go.work` is behind the module minimum:

```text
go: module . listed in go.work file requires go >= 1.26.1, but go.work lists go 1.26; to update it:
	go work use
go: module ../bobatea listed in go.work file requires go >= 1.26.1, but go.work lists go 1.26; to update it:
	go work use
```

- I validated the package slice with:

```text
GOWORK=off go test ./engine/... -count=1
```

### What I learned

- Rebuilding the registry per runtime is a clean way to preserve factory immutability while supporting runtime-scoped module injection.
- Cleanup hooks belong in `engine.Runtime`, not in a plugin package, because the pattern is useful beyond plugins.
- The workspace-level `go.work` file is likely to keep affecting default test commands until it is updated.

### What was tricky to build

- The subtle part was keeping the old construction API stable while changing the ownership model underneath it. The existing code assumes the factory is reusable and immutable. Rebuilding a registry on every `NewRuntime()` call preserves that assumption, but only because the factory now stores module specs and options as immutable configuration rather than prebuilt runtime objects.

### What warrants a second pair of eyes

- Whether `RuntimeModuleContext` should eventually expose the `Runtime` itself.
- Whether `Runtime.AddCloser(...)` should return a typed sentinel error for late registration.
- Whether the `go.work` mismatch should be fixed inside this ticket once broader tests are running.

### What should be done in the future

- Add the `go-plugin` dependency and shared contract layer next.
- Build discovery/validation/reification on top of the new registrar seam.
- Revisit the workspace `go.work` mismatch if it keeps blocking validation.

### Code review instructions

- Review `engine/runtime_modules.go` first.
- Then read `engine/factory.go` and `engine/runtime.go` together.
- Finish with `engine/runtime_modules_test.go`.

### Technical details

- Commands run:
  - `gofmt -w engine/factory.go engine/runtime.go engine/runtime_modules.go engine/runtime_modules_test.go`
  - `go test ./engine/... -count=1`
  - `GOWORK=off go test ./engine/... -count=1`
  - `command -v protoc && command -v protoc-gen-go && command -v protoc-gen-go-grpc`

## Step 3: Add the shared HashiCorp plugin contract and gRPC adapter scaffold

This step added the first real `hashiplugin` package code. I introduced a protobuf contract for manifests and invocation, generated the Go bindings, added a small `contract.JSModule` interface shared by host and plugin implementations, and then wrapped the generated gRPC client/server with a `go-plugin` `GRPCPlugin` adapter under `pkg/hashiplugin/shared`.

The goal of this step was not to implement discovery or runtime registration yet. It was to give the next phase a stable transport surface so the host package can focus on policy and lifecycle instead of also inventing the wire protocol at the same time. The adapter test proves that `go-plugin` can dispense a typed module interface over gRPC using the exact contract this repo will build on.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Continue executing the ticket task list phase by phase, committing coherent slices and keeping the diary current.

**Inferred user intent:** Land the feature incrementally with clear checkpoints instead of one large unreviewable patch.

**Commit (code):** `d474dd9` — `hashiplugin: add shared gRPC contract scaffold`

### What I did

- Added `github.com/hashicorp/go-plugin v1.7.0` to `go.mod`.
- Added `pkg/hashiplugin/contract/jsmodule.proto`.
- Added `pkg/hashiplugin/contract/generate.go` and generated `jsmodule.pb.go` plus `jsmodule_grpc.pb.go`.
- Added `pkg/hashiplugin/contract/contract.go` with the repo-local `JSModule` interface.
- Added `pkg/hashiplugin/shared/plugin.go` with handshake settings, plugin-set helpers, and the `GRPCPlugin` adapter.
- Added `pkg/hashiplugin/shared/plugin_test.go` with a round-trip `plugin.TestPluginGRPCConn(...)` test.

### Why

- The host and plugin subprocess need one typed contract before discovery and runtime registration logic is added.
- The gRPC adapter belongs in a shared package because both the host binary and plugin binaries need the same handshake and plugin-set definitions.
- A narrow adapter test is the fastest way to confirm that the chosen transport shape works before adding more moving parts.

### What worked

- `go-plugin` latest stable version resolved cleanly at `v1.7.0`.
- `protoc`, `protoc-gen-go`, and `protoc-gen-go-grpc` worked with a simple local `go:generate` directive.
- The adapter round-trip test passed:

```text
GOWORK=off go test ./pkg/hashiplugin/... -count=1
?   	github.com/go-go-golems/go-go-goja/pkg/hashiplugin/contract	[no test files]
ok  	github.com/go-go-golems/go-go-goja/pkg/hashiplugin/shared	0.005s
```

### What didn't work

- I initially launched `go test` in parallel with `go generate`, so the test started before the generated files existed and the contract package failed with undefined generated types.
- That was an orchestration mistake rather than a code defect. Rerunning the tests after generation resolved it.

### What I learned

- `plugin.TestPluginGRPCConn(...)` is a convenient way to validate the adapter without launching subprocesses yet.
- The generated service uses `GetManifest(...)`, while the repo-local interface can keep the cleaner `Manifest(...)` method and map between them in the adapter.
- Keeping the generated protobuf package and the handwritten interface in the same `contract` package makes later host code simpler.

### What was tricky to build

- The main judgment call was where to put the boundary between generated code and handwritten code. Putting both the generated message types and the handwritten `JSModule` interface in the same `contract` package makes the public API easier to consume, but it also means the package mixes generated and non-generated files. That tradeoff is worth it here because every future caller wants one import path for the transport contract.

### What warrants a second pair of eyes

- Whether the manifest schema should include more explicit fields for docs, capability flags, or future dynamic-object support now or later.
- Whether we want a stricter naming scheme for the magic cookie constants before plugin binaries are published.
- Whether a dedicated `proto/` subdirectory would be preferable if more services are added later.

### What should be done in the future

- Implement the host package on top of this scaffold next.
- Add manifest validation rules before any discovered plugin can be registered.
- Add real subprocess-backed integration tests after host loading exists.

### Code review instructions

- Start with `pkg/hashiplugin/contract/jsmodule.proto`.
- Then read `pkg/hashiplugin/shared/plugin.go`.
- Finish with `pkg/hashiplugin/shared/plugin_test.go` to see the intended usage.

### Technical details

- Commands run:
  - `GOWORK=off go list -m -versions github.com/hashicorp/go-plugin`
  - `GOWORK=off go get github.com/hashicorp/go-plugin@v1.7.0`
  - `GOWORK=off go doc github.com/hashicorp/go-plugin`
  - `GOWORK=off go doc github.com/hashicorp/go-plugin.ClientConfig`
  - `GOWORK=off go doc github.com/hashicorp/go-plugin.GRPCPlugin`
  - `PATH="$(go env GOPATH)/bin:$PATH" GOWORK=off go generate ./pkg/hashiplugin/contract`
  - `gofmt -w pkg/hashiplugin/contract/contract.go pkg/hashiplugin/shared/plugin.go pkg/hashiplugin/shared/plugin_test.go`
  - `GOWORK=off go test ./pkg/hashiplugin/... -count=1`

## Step 4: Implement host loading, integration tests, and REPL wiring

This step turned the scaffold into a working feature. I added the `pkg/hashiplugin/host` package that discovers binaries, launches plugin clients, reads and validates manifests, reifies plugin exports into CommonJS native modules, and registers a cleanup hook so plugin subprocesses are killed when the runtime closes. I also added two tiny test plugins, one valid and one intentionally invalid, and used them in integration tests that build the plugin binaries on the fly.

I then wired plugin directory support into real runtime creation paths. The simple Cobra REPL now accepts `--plugin-dir`, and the reusable JavaScript evaluator config gained a `PluginDirectories` field so other runtime entrypoints can opt in without duplicating registrar wiring. At this point the feature is functionally present: a runtime can load `plugin:echo`, call exported functions and object methods, and tear down the plugin process on close.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Continue implementing the ticket tasks until the designed plugin support becomes a working runtime feature, while keeping the ticket docs up to date.

**Inferred user intent:** Reach a usable end-to-end plugin path, not just internal library scaffolding.

**Commit (code):** `9a463cf` — `hashiplugin: load plugin modules into runtimes`

### What I did

- Added `pkg/hashiplugin/host/config.go`, `discover.go`, `validate.go`, `client.go`, `reify.go`, and `registrar.go`.
- Added `pkg/hashiplugin/testplugin/echo/main.go` and `pkg/hashiplugin/testplugin/invalid/main.go`.
- Added `pkg/hashiplugin/host/registrar_test.go` with end-to-end tests that build plugin binaries and verify loading, validation failure, and subprocess cleanup.
- Wired plugin-directory support into `pkg/repl/evaluators/javascript.Config`.
- Wired `--plugin-dir` into `cmd/repl`.

### Why

- The host package is where the design becomes real: discovery, trust boundary checks, subprocess lifecycle, and JS export reification all live there.
- Example plugins make the tests meaningful and give future reviewers a concrete authoring template.
- Wiring at least one entrypoint was part of the task list and proves the feature can be used outside tests.

### What worked

- The valid `plugin:echo` example loaded into a real runtime and returned expected values from `ping`, `math.add`, and `pid`.
- The invalid plugin fixture failed runtime creation with the expected namespace validation error.
- The runtime cleanup test confirmed that the plugin subprocess exits after `Runtime.Close()`.
- The evaluator package still passed after adding `PluginDirectories`.

### What didn't work

- Nothing in this phase failed after the initial host code landed. The main debugging effort had already been burned down in the earlier engine and contract phases.

### What I learned

- Keeping manifest validation separate from client startup and module reification makes the host package much easier to reason about.
- Building test plugin binaries inside the integration tests is a practical way to exercise real subprocess behavior without committing built artifacts.
- The engine seam added in Step 2 is exactly the right boundary for this feature; the host registrar fit into it with very little friction.

### What was tricky to build

- The sharpest part was deciding how much policy to implement in v1. It would be easy to over-design the host config with checksum maps, richer capability gates, or more elaborate naming policies before the core loading path existed. I kept the first host version strict but small: namespace enforcement, duplicate/shape validation, executable filtering, gRPC-only transport, and per-runtime lifecycle. That gives a stable base without locking the API into a half-finished security story.

## Step 5: Validate the repo-wide change set and refresh the ticket bundle

This step closed the ticket loop after the implementation work landed. I reran the full repository test suite, reran `docmgr doctor` for GOJA-08, updated the ticket bookkeeping with the actual implementation commit hashes, and prepared the refreshed document bundle for upload back to reMarkable. The purpose here was not to add new feature code, but to make the ticket trustworthy as a continuation point for the next engineer.

The important outcome is that the implementation and the documentation now describe the same state. The task list is fully checked off, the diary records the reviewable commit slices, and the final validation step was executed against the repository after the host/runtime integration landed.

### Prompt Context

**User prompt (verbatim):**

```text
alright, create a detailed set of tasks in the ticket, and then work on them one by one, checking things off as you go, committing at appropriate intervals, and keeping a diary.
```

**Assistant interpretation:** Finish the remaining ticket tasks, keep the implementation broken into reviewable commits, and record the final verification/publication work in the ticket itself.

**Inferred user intent:** Leave behind a clean, auditable ticket that matches the code now in the branch.

**Commit (code):** `8239796` — `docs: update GOJA-08 implementation diary`

### What I did

- Ran `go test ./... -count=1` after the plugin host integration commit.
- Ran `docmgr doctor --ticket GOJA-08-HASHICORP-PLUGINS--add-hashicorp-plugin-support-for-runtime-module-registration --stale-after 30`.
- Updated `tasks.md` so the implementation sequence is fully checked off.
- Updated this diary with concrete commit hashes for the three implementation slices.
- Added a closeout entry to `changelog.md`.
- Prepared the refreshed ticket bundle for reMarkable publication.

### Why

- The user explicitly asked for task-by-task execution, checked-off progress, commits at reasonable intervals, and a maintained diary.
- Without a closeout step, the ticket would still reflect the earlier research-only state instead of the implemented branch state.
- Re-running repository-wide validation after the final feature commit is the fastest way to catch accidental regressions outside the focused package tests.

### What worked

- The full repository test suite passed after the plugin changes landed.
- `docmgr doctor` reported that all GOJA-08 checks passed.
- The earlier task breakdown translated cleanly into three code commits plus this final documentation/closeout pass.

### What didn't work

- There were no new technical failures in this step. The main issue that surfaced during closeout was process-related: the earlier failed host commit had been caused by stale staged content, which is now reflected accurately in the diary and resolved in the final commit history.

### What I learned

- Keeping the ticket diary current during implementation makes the final closeout substantially easier because the major design decisions are already captured while they are fresh.
- The runtime/plugin work is broad enough that repo-wide validation is worth doing even after focused package tests have already passed; it confirms there are no hidden compile or lint regressions in neighboring entrypoints.

### What was tricky to build

- The tricky part here was sequencing the final docs update and publication so the uploaded bundle reflects the real branch state rather than a slightly earlier snapshot. That is why the ticket bookkeeping is updated before the final upload instead of after it.

### What warrants a second pair of eyes

- Whether the closeout bundle should eventually include a shorter operator-facing plugin authoring quickstart in addition to the architecture guide.
- Whether future tickets should split the research/design phase and the implementation phase into separate bundled documents for easier consumption on reMarkable.

### What should be done in the future

- Add a dedicated plugin authoring example or template package once the first external plugin consumer exists.
- Decide whether plugin trust policy should grow beyond namespace/shape validation into checksums or explicit allowlists by default.
- Consider wiring plugin directories into more runtime entrypoints if the feature proves broadly useful.

### Code review instructions

- Read the three implementation commits in order: engine seam, shared transport, then host/runtime integration.
- Use this diary to map each commit to the underlying design intent.
- Cross-check the final task list in `tasks.md` if you want the high-level execution sequence.

### Technical details

- Commands run:
  - `go test ./... -count=1`
  - `docmgr doctor --ticket GOJA-08-HASHICORP-PLUGINS--add-hashicorp-plugin-support-for-runtime-module-registration --stale-after 30`
  - `git log --oneline -3`
  - `remarquee status`
  - `remarquee cloud account --non-interactive`
  - `remarquee upload bundle --dry-run ttmp/2026/03/18/GOJA-08-HASHICORP-PLUGINS--add-hashicorp-plugin-support-for-runtime-module-registration--add-hashicorp-plugin-support-for-runtime-module-registration/index.md ttmp/2026/03/18/GOJA-08-HASHICORP-PLUGINS--add-hashicorp-plugin-support-for-runtime-module-registration--add-hashicorp-plugin-support-for-runtime-module-registration/design-doc/01-hashicorp-plugin-support-for-go-go-goja-architecture-and-implementation-guide.md ttmp/2026/03/18/GOJA-08-HASHICORP-PLUGINS--add-hashicorp-plugin-support-for-runtime-module-registration--add-hashicorp-plugin-support-for-runtime-module-registration/reference/01-diary.md ttmp/2026/03/18/GOJA-08-HASHICORP-PLUGINS--add-hashicorp-plugin-support-for-runtime-module-registration--add-hashicorp-plugin-support-for-runtime-module-registration/tasks.md ttmp/2026/03/18/GOJA-08-HASHICORP-PLUGINS--add-hashicorp-plugin-support-for-runtime-module-registration--add-hashicorp-plugin-support-for-runtime-module-registration/changelog.md --name "GOJA-08 HashiCorp plugin support" --remote-dir "/ai/2026/03/18/GOJA-08-HASHICORP-PLUGINS" --toc-depth 2`

## Step 6: Add a user-facing example plugin path and retarget the docs

This step started the productization pass after the core runtime work was already complete. The immediate problem was that the docs were still telling users to build from `plugins/testplugin/...`, which is fine for integration tests but not a clean onboarding story for plugin authors. I split the responsibilities by adding a new `plugins/examples/greeter` sample plus a small `plugins/examples/README.md`, while keeping `plugins/testplugin/...` as the deterministic fixture path used by host integration tests.

That split matters because examples and test fixtures optimize for different things. The example should be readable, copyable, and slightly more narrative. The test fixture should stay tiny, stable, and shaped around assertions. Once those two concerns are separated, the help docs can teach from the example path without forcing future authors to reverse-engineer why a “sample” package is named like a test.

### Prompt Context

**User prompt (verbatim):**

```text
Ok, add tasks for all of this to the ticket, and then work on them task by task.
```

**Assistant interpretation:** Turn the earlier “what’s next” list into explicit ticket tasks and then execute those tasks one by one, starting with the most concrete user-facing improvement.

**Inferred user intent:** Move from raw feature-complete plumbing to a more polished, teachable plugin subsystem.

**Commit (code):** pending

### What I did

- Added the next-phase task list to `tasks.md`.
- Added `plugins/examples/README.md`.
- Added `plugins/examples/greeter/main.go` as a copyable sample plugin.
- Updated the user guide and tutorial to build the example plugin instead of the test fixture.
- Updated the developer guide to distinguish `plugins/examples/...` from `plugins/testplugin/...`.

### Why

- The docs should teach from a path that exists for users rather than a path that exists primarily for tests.
- A copyable plugin example shortens the path from “I understand the architecture” to “I can build one myself.”
- The productization pass needed a clean first slice before adding visibility, policy, and more entrypoint wiring.

### What worked

- The new example plugin compiled cleanly.
- The updated help pages rendered without frontmatter or embedding issues.
- Keeping the integration-test fixture packages untouched avoided unnecessary churn in the runtime tests.

### What didn't work

- The existing docs still had several references to the test fixture path, so the first patch pass needed a wider sweep than expected.

### What I learned

- The docs were already detailed enough that moving from fixture-path examples to real example-path examples had a large user-facing effect with relatively little code.
- Keeping the example plugin separate from the test fixture makes later documentation and README work much easier, because the example can evolve for clarity without destabilizing tests.

### What was tricky to build

- The tricky part was deciding whether to replace the existing fixture package entirely or introduce a second path. Replacing it would have blurred the line between authoring guidance and test stability again. Adding a second path costs a bit more maintenance but keeps the intent of each package obvious.

### What warrants a second pair of eyes

- Whether the example plugin should eventually expose one async-style method as well, so authors see that shape early.
- Whether `plugins/examples` should grow into a small catalog or stay intentionally tiny with just one canonical example.

### What should be done in the future

- Add discovery visibility in the REPLs next.
- Add allowlisting/trust-policy knobs after the visibility work.
- Refresh the ticket bundle once the productization pass is complete.

### Code review instructions

- Start with `plugins/examples/greeter/main.go`.
- Then read `plugins/examples/README.md`.
- Finish by skimming the three plugin help pages to confirm the teaching path now points at the example package.

### Technical details

- Commands run:
  - `find plugins -maxdepth 3 -type f | sort`
  - `gofmt -w plugins/examples/greeter/main.go`
  - `go test ./plugins/examples/... ./pkg/doc -count=1`
  - `go run ./cmd/repl help goja-plugin-user-guide | sed -n '1,120p'`
  - `go run ./cmd/repl help plugin-tutorial-build-install | sed -n '1,120p'`

## Step 7: Add plugin discovery visibility to the REPL surfaces

This step focused on operability rather than transport or runtime semantics. The plugin system was already functional, but once default plugin discovery was enabled it became harder for a user to answer basic questions such as “what directories were scanned?”, “what module names actually loaded?”, and “why am I not seeing any plugin modules?” I addressed that by adding a small host-side report collector and surfacing it through both REPL entrypoints.

The line REPL now gives the best interactive experience: it prints a short startup summary when plugin directories are configured, supports `:plugins` inside the loop, and supports `--plugin-status` for a non-interactive report. The Bobatea `js-repl` does not have a command palette command for this yet, but it does now expose the same one-shot `--plugin-status` path and shows a short plugin summary in the initial placeholder text. That keeps the diagnostics story consistent without having to design a full TUI diagnostics panel yet.

### Prompt Context

**User prompt (verbatim):**

```text
Ok, add tasks for all of this to the ticket, and then work on them task by task.
```

**Assistant interpretation:** Continue the productization backlog in small, reviewable slices, prioritizing visibility and usability after the example plugin work.

**Inferred user intent:** Make the plugin system inspectable from the user-facing tools, not just from code and tests.

**Commit (code):** pending

### What I did

- Added `pkg/hashiplugin/host/report.go` with a report collector and CLI-friendly summaries.
- Updated the registrar to record discovered candidates, loaded modules, and startup errors.
- Added `:plugins` and `--plugin-status` to `cmd/repl`.
- Added `--plugin-status` and startup summary support to `cmd/js-repl`.
- Extended the JavaScript evaluator config so the TUI path can pass the host report collector down to the registrar.
- Updated the help pages to document the new diagnostics surface.

### Why

- Default discovery is convenient, but it increases the need for visibility.
- Users need a way to confirm which modules were actually loaded without reading code or turning on unrelated debug logs.
- A small reporting seam in the host package is reusable for later policy work and future entrypoints.

### What worked

- The report collector fit cleanly into the existing registrar without changing the transport contract.
- `repl --plugin-status` and `js-repl --plugin-status` both work without building a second discovery/load path.
- The line REPL gained a useful `:plugins` command with minimal complexity.

### What didn't work

- The first `repl --plugin-status` pass printed unrelated native-module debug logs before the report because the command had never pinned the global `zerolog` level. I fixed that by explicitly setting `zerolog.ErrorLevel` unless `--debug` is enabled.

### What I learned

- A small runtime setup report gives most of the operational value users need without having to build a broader observability subsystem.
- The right seam for visibility was the host registrar, not the CLI layer, because that keeps the report tied to the real load path instead of a duplicate probe path.

### What was tricky to build

- The tricky design point was avoiding a second startup path that would rediscover and reload plugins just for reporting. That would have been easier to write, but it would also have been less trustworthy and would have introduced extra plugin process churn. Threading a report collector through the existing registrar kept the behavior honest.

### What warrants a second pair of eyes

- Whether `js-repl` should eventually gain a first-class in-UI plugin diagnostics view instead of relying on `--plugin-status` plus the placeholder summary.
- Whether the report should eventually include more manifest detail such as versioned capability flags once the contract grows.

### What should be done in the future

- Add allowlisting/trust-policy knobs next.
- Wire plugin configuration into one more runtime consumer after the policy surface settles.
- Refresh and republish the ticket bundle once the whole productization pass is done.

### Code review instructions

- Read `pkg/hashiplugin/host/report.go` first.
- Then read `cmd/repl/main.go` and `cmd/js-repl/main.go`.
- Finish with `pkg/repl/evaluators/javascript/evaluator.go` to see how the report collector reaches the TUI path.

### Technical details

- Commands run:
  - `sed -n '1,260p' pkg/hashiplugin/host/client.go`
  - `sed -n '1,260p' pkg/hashiplugin/host/registrar.go`
  - `gofmt -w cmd/repl/main.go cmd/js-repl/main.go pkg/repl/evaluators/javascript/evaluator.go pkg/hashiplugin/host/report.go pkg/hashiplugin/host/report_test.go pkg/hashiplugin/host/registrar.go`
  - `go test ./cmd/repl ./cmd/js-repl ./pkg/hashiplugin/host ./pkg/repl/evaluators/javascript -count=1`
  - `go run ./cmd/repl --plugin-status | sed -n '1,120p'`
  - `go run ./cmd/js-repl --plugin-status | sed -n '1,120p'`

## Step 8: Add allowlist trust-policy knobs to the supported entrypoints

This step took an existing internal validation feature and made it usable. The host config already had `AllowModules`, but until now there was no straightforward way for a user or runtime consumer to ask for “only these plugin module names are allowed in this run.” I wired that allowlist through the two REPL entrypoints and the shared JavaScript evaluator config, then added an integration test proving that a plugin is rejected when it is outside the requested allowlist.

This is still a modest trust-policy surface. It is not checksums, signatures, or provenance. But it is an important operational guardrail because it lets a caller constrain module names explicitly even when discovery is happening over a broader directory tree. That is the right next step before stronger policy mechanisms, because it is simple, visible, and easy to reason about.

### Prompt Context

**User prompt (verbatim):**

```text
Ok, add tasks for all of this to the ticket, and then work on them task by task.
```

**Assistant interpretation:** Continue the productization pass, moving from visibility into lightweight trust controls that are concrete enough to use immediately.

**Inferred user intent:** Make the plugin system safer to operate without over-designing the trust model in one jump.

**Commit (code):** pending

### What I did

- Added `--allow-plugin-module` to `cmd/repl`.
- Added `--allow-plugin-module` to `cmd/js-repl`.
- Added `PluginAllowModules` to the shared JavaScript evaluator config.
- Normalized allowlist values in `host.Config.withDefaults()`.
- Added an integration test that rejects `plugin:echo` when only `plugin:greeter` is allowed.
- Updated the help docs to describe the allowlist flag and the resulting startup error shape.

### Why

- The existing allowlist check had no user-facing control surface.
- Module-name allowlisting is a practical first trust-policy control because it is explicit and low-complexity.
- Wiring the flag through the shared evaluator path keeps the policy surface consistent across the two REPL entrypoints.

### What worked

- The flag mapped cleanly onto the existing `ValidateManifest(...)` allowlist logic.
- The integration test gave direct proof that the entrypoint-facing configuration now matters in the runtime load path.
- The help output for both REPL binaries now surfaces the allowlist flag.

### What didn't work

- Nothing failed technically in this step once the wiring path was chosen. The main risk was missing one of the config layers, which is why I validated both command help surfaces and the host integration path explicitly.

### What I learned

- Small policy features become much more valuable once they are available uniformly at both the direct engine-builder path and the higher-level evaluator path.
- The existing host config design was already good enough for this feature; the missing work was mostly propagation and documentation.

### What was tricky to build

- The subtle part was resisting the urge to add a larger policy system immediately. A single explicit allowlist flag is easy to teach and review. That keeps the system moving while leaving room for stronger future controls.

### What warrants a second pair of eyes

- Whether `--allow-plugin-module` should eventually accept globbing or remain exact-match only.
- Whether the allowlist should be reflected directly in the plugin status report so operators can see both “what loaded” and “what was permitted.”

### What should be done in the future

- Wire plugin configuration into one additional runtime consumer next.
- Refresh and republish the ticket bundle after the productization pass is complete.

### Code review instructions

- Start with `cmd/repl/main.go` and `cmd/js-repl/main.go`.
- Then read `pkg/repl/evaluators/javascript/evaluator.go`.
- Finish with `pkg/hashiplugin/host/registrar_test.go` to verify the allowlist behavior end to end.

### Technical details

- Commands run:
  - `sed -n '1,220p' pkg/hashiplugin/host/validate.go`
  - `sed -n '1,140p' pkg/repl/evaluators/javascript/evaluator.go`
  - `gofmt -w pkg/hashiplugin/host/config.go pkg/hashiplugin/host/registrar_test.go pkg/repl/evaluators/javascript/evaluator.go cmd/repl/main.go cmd/js-repl/main.go`
  - `go test ./cmd/repl ./cmd/js-repl ./pkg/hashiplugin/host ./pkg/repl/evaluators/javascript -count=1`
  - `go run ./cmd/repl --help | sed -n '1,120p'`
  - `go run ./cmd/js-repl --help | sed -n '1,120p'`

## Step 9: Wire plugins into one additional runtime consumer

This step closed the last implementation item in the productization backlog by wiring plugin support into `cmd/bun-demo`. That command was a good target because it already builds an engine runtime directly and loads a bundled CommonJS entrypoint through `require()`. Once it gained the same `--plugin-dir` and `--allow-plugin-module` controls as the REPLs, the plugin system stopped being “a REPL feature” and became a more general runtime composition capability.

This was a deliberately smaller choice than pushing plugins into the Smalltalk inspector runtime path immediately. `bun-demo` uses the same engine/factory seam as the REPLs, so it validates the portability of the plugin design without dragging in a second custom runtime model. That keeps the architectural signal clean: the plugin seam works anywhere the repository uses the owned engine runtime composition path.

### Prompt Context

**User prompt (verbatim):**

```text
Ok, add tasks for all of this to the ticket, and then work on them task by task.
```

**Assistant interpretation:** Finish the remaining concrete implementation task before the final docs/validation/upload refresh.

**Inferred user intent:** Prove the plugin system composes into another real consumer, not just the REPL binaries.

**Commit (code):** pending

### What I did

- Added `--plugin-dir` and `--allow-plugin-module` support to `cmd/bun-demo`.
- Reused the existing host registrar rather than inventing a bundle-specific plugin path.
- Updated the Bun bundling playbook to show plugin-backed runtime extensions.
- Updated the developer guide to list `cmd/bun-demo` as another wired entrypoint.

### Why

- The user explicitly asked to work through the full follow-up list task by task.
- An extra consumer proves the runtime/plugin architecture is reusable.
- `cmd/bun-demo` is a clean engine-runtime consumer with fewer unrelated concerns than the inspector stack.

### What worked

- The Bun demo already used the owned engine-builder path, so the plugin registrar fit cleanly.
- The command help now shows the plugin flags exactly as expected.
- The bundling doc now has a place to explain plugin-backed extensions in a non-REPL runtime.

### What didn't work

- Nothing failed technically in this step. The main design choice was selecting the right consumer to wire first.

### What I learned

- The plugin runtime seam is broad enough to support both interactive and embedded-bundle consumers with minimal extra code.
- `cmd/bun-demo` is a useful proving ground for runtime composition changes because it is simple but not trivial.

### What was tricky to build

- The tricky part was choosing a second consumer that validates the design without turning the task into a separate UI integration project. `bun-demo` hit that balance well.

### What warrants a second pair of eyes

- Whether `bun-demo` should eventually grow a small example bundle that actually requires `plugin:greeter` so the docs have a matching runnable artifact.
- Whether the next consumer after this should be `smalltalk-inspector` or another direct engine-runtime tool.

### What should be done in the future

- Refresh the ticket docs, validation results, and reMarkable bundle now that the productization backlog is implemented.

### Code review instructions

- Start with `cmd/bun-demo/main.go`.
- Then read the new plugin section in `pkg/doc/bun-goja-bundling-playbook.md`.
- Finish with `pkg/doc/13-plugin-developer-guide.md` to confirm the architecture notes now reflect the extra consumer.

### Technical details

- Commands run:
  - `sed -n '1,220p' cmd/bun-demo/main.go`
  - `gofmt -w cmd/bun-demo/main.go`
  - `go test ./cmd/bun-demo ./pkg/doc -count=1`
  - `go run ./cmd/bun-demo --help | sed -n '1,120p'`

### What warrants a second pair of eyes

- Whether `Config` should default `AutoMTLS` to true unconditionally or expose a more explicit enable/disable option.
- Whether `plugin:...` names are definitely safe across all intended module-resolution paths, or whether `plugin/...` would be a lower-risk namespace.
- Whether the example plugin packages should move under `testdata/` once the feature stabilizes.

### What should be done in the future

- Run the full repo validation pass and refresh the ticket bundle on reMarkable.
- Decide whether to expose plugin directory flags in additional entrypoints such as the Bobatea JS REPL.
- Add stronger checksum/allowlist policy once the runtime-facing path is stable.

### Code review instructions

- Start with `pkg/hashiplugin/host/registrar.go`.
- Then read `client.go`, `validate.go`, and `reify.go` as one flow.
- Review `pkg/hashiplugin/host/registrar_test.go` and the two `testplugin` packages together.
- Finish with the small entrypoint changes in `pkg/repl/evaluators/javascript/evaluator.go` and `cmd/repl/main.go`.

### Technical details

- Commands run:
  - `GOWORK=off go doc github.com/hashicorp/go-plugin.Client.Kill`
  - `GOWORK=off go doc github.com/hashicorp/go-plugin.Client.Exited`
  - `gofmt -w pkg/hashiplugin/host/*.go pkg/hashiplugin/testplugin/echo/main.go pkg/hashiplugin/testplugin/invalid/main.go`
  - `GOWORK=off go test ./pkg/hashiplugin/... -count=1`
  - `gofmt -w pkg/repl/evaluators/javascript/evaluator.go cmd/repl/main.go`
  - `GOWORK=off go test ./pkg/hashiplugin/... ./pkg/repl/evaluators/javascript -count=1`
