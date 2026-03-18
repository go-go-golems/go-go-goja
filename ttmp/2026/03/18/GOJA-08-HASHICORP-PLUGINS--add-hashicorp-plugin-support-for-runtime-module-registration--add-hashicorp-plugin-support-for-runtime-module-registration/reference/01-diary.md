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

**Commit (code):** pending

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
