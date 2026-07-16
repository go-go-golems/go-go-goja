---
Title: Investigation diary
Ticket: GOJA-068
Status: complete
Topics:
    - goja
    - repl
    - replapi
    - lifecycle
    - persistent-repl
    - http
    - security
    - sqlite
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://cmd/goja-repl/cmd_serve.go
      Note: |-
        Step 12 server timeout and remote-bind deployment safety
        Step 13 server shutdown sequencing
    - Path: repo://cmd/goja-repl/root.go
      Note: Step 13 shared host shutdown contract
    - Path: repo://cmd/goja-repl/shutdown_test.go
      Note: Step 13 ordering and failure regression evidence
    - Path: repo://cmd/goja-repl/tui.go
      Note: Step 13 TUI ownership and error propagation
    - Path: repo://pkg/doc/04-repl-usage.md
      Note: Step 13 generated ErrorResponse client guidance
    - Path: repo://pkg/doc/34-replapi-guide.md
      Note: Step 13 release migration guidance
    - Path: repo://pkg/engine/factory.go
      Note: P2 event-loop startup ordering required for race-free immediate close
    - Path: repo://pkg/replapi/lifecycle.go
      Note: P2 app lifecycle errors, unload, and idempotent close
    - Path: repo://pkg/repldb/lease.go
      Note: P5 durable lease, epoch, release, and fenced append primitives
    - Path: repo://pkg/repldb/migrations.go
      Note: P4 ordered transactional migration and typed version contract
    - Path: repo://pkg/repldb/testdata/repl-v1.sqlite
      Note: P4 immutable real schema-v1 upgrade fixture
    - Path: repo://pkg/replhttp/handler.go
      Note: Step 12 central HTTP hardening implementation
    - Path: repo://pkg/replhttp/proto_handler.go
      Note: Step 12 bounded/versioned request parsing
    - Path: repo://pkg/replhttp/security_test.go
      Note: Step 12 real transport regression coverage
    - Path: repo://pkg/replsession/health.go
      Note: P3 health state, commit error, exact retry, and publication contract
    - Path: repo://pkg/replsession/lifecycle.go
      Note: P2 session gates, lifecycle states, retryable unload/delete/close, and context composition
    - Path: repo://pkg/replsession/ownership.go
      Note: P5 lease acquisition, heartbeat, fencing, and service integration
    - Path: repo://proto/goja/replapi/v1/replapi.proto
      Note: Step 12 generated stable error envelope
    - Path: repo://ttmp/2026/07/15/GOJA-068--harden-replapi-runtime-lifecycle-session-ownership-and-http-safety/scripts/01-probe-http-session-context/main.go
      Note: Reproduces HTTP request cancellation of runtime lifetime
    - Path: repo://ttmp/2026/07/15/GOJA-068--harden-replapi-runtime-lifecycle-session-ownership-and-http-safety/scripts/02-probe-persistent-split-brain/main.go
      Note: Reproduces two-app VM divergence and cell conflict
    - Path: repo://ttmp/2026/07/15/GOJA-068--harden-replapi-runtime-lifecycle-session-ownership-and-http-safety/scripts/03-probe-partial-profile-config/main.go
      Note: Reproduces raw profile and policy mismatch
    - Path: repo://ttmp/2026/07/15/GOJA-068--harden-replapi-runtime-lifecycle-session-ownership-and-http-safety/scripts/04-probe-post-execution-persistence-failure/main.go
      Note: Reproduces durable cell gap after failed append
    - Path: repo://ttmp/2026/07/15/GOJA-068--harden-replapi-runtime-lifecycle-session-ownership-and-http-safety/scripts/05-probe-canceled-waiter/main.go
      Note: Reproduces canceled caller blocked on session mutex
ExternalSources: []
Summary: Chronological record of the GOJA-068 replapi lifecycle, persistence, coordination, configuration, and HTTP hardening investigation.
LastUpdated: 2026-07-15T14:45:00-04:00
WhatFor: Reproduce the audit, understand why the proposed design was selected, and continue implementation without repeating the investigation.
WhenToUse: Read before implementing or reviewing GOJA-068, especially when a test exposes lifecycle or durable-state behavior not obvious from the package API.
---







# Diary

## Goal

This diary records the evidence-first investigation behind GOJA-068: how the ticket was created, which package boundaries were inspected, which defects were reproduced, and how those observations became a phased lifecycle and hardening design. It is intended to let an intern or reviewer continue the work without rediscovering the runtime, session, persistence, and transport contracts.

## Step 1: Create the ticket and establish scope

The first step created a focused ticket rather than mixing implementation hardening into the help-page change that exposed the problems. The ticket scope deliberately joins runtime lifecycle, persistent ownership, SQLite integrity, and HTTP safety because those concerns meet at `replapi.App`; addressing only the HTTP symptom would leave restore and embedded-host paths unsafe.

I also established seven explicit tasks before investigating. This made delivery measurable: architecture mapping, context reproduction, lifecycle API design, persistent ownership, HTTP audit, implementation/test planning, and final validation/upload.

### Prompt Context

**User prompt (verbatim):**

> Ok, create a new ticket to address these issues, and Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.
>
>
> Feel free to investeigate more closely to see if you find other improvements and issues with the replapi part.

**Assistant interpretation:** Create a new docmgr ticket, investigate the REPL API beyond the already known lifecycle limitations, write an intern-oriented technical design and implementation guide, maintain ticket bookkeeping and a diary, validate it, and upload the bundle to reMarkable.

**Inferred user intent:** Turn an informal documentation review into a durable engineering work package that a new contributor can implement safely without needing prior knowledge of Goja runtime ownership or the REPL persistence design.

### What I did

- Loaded the ticket-research, docmgr, diary, Glazed help, and reMarkable workflows.
- Ran `docmgr status --summary-only` and checked vocabulary for `replapi`, `lifecycle`, `persistent-repl`, `http`, `security`, and `sqlite`.
- Confirmed `GOJA-068` did not already exist:

```text
No tickets found.
```

- Created `GOJA-068 — Harden replapi runtime lifecycle, session ownership, and HTTP safety`.
- Created:
  - `design-doc/01-repl-api-lifecycle-ownership-and-http-hardening-implementation-guide.md`
  - `reference/01-investigation-diary.md`
- Added seven ticket tasks before beginning the audit.

### Why

- The work affects several packages and needs an explicit implementation order.
- A ticket keeps the design, evidence scripts, tasks, changelog, and later code relations connected.
- The intern audience requires more than a list of bugs; it requires package orientation, vocabulary, invariants, and review instructions.

### What worked

- Existing vocabulary already contained all required topic slugs.
- `docmgr` created the expected index, tasks, changelog, design-doc, reference, and scripts directories.
- The ticket ID followed the existing sequence after GOJA-067.

### What didn't work

- No setup command failed. The preflight lookup intentionally returned `No tickets found.`, confirming the ID was free.

### What I learned

- The repository already had substantial prior REPL design work under GOJA-20, GOJA-23, GOJA-40, GOJA-41, and GOJA-67. The new ticket should extend those decisions rather than present lifecycle hardening as a greenfield system.
- The current working tree already contained uncommitted help-page work from the preceding request, so ticket work had to preserve and explicitly identify those changes rather than accidentally absorb or overwrite them.

### What was tricky to build

- Scope control was the first sharp edge. Request-context ownership, app close, persistent split brain, and HTTP safety can look like unrelated defects. They are connected by a single ownership question: which component is responsible for a runtime and its durable session at each moment?
- The solution was to define ownership and lifecycle as the ticket center, with HTTP and persistence as adapters that must obey that center.

### What warrants a second pair of eyes

- Confirm that GOJA-068 is the right grouping for lease/fencing and HTTP hardening rather than splitting those into follow-up tickets. The design phases make separation possible if implementation review prefers smaller delivery units.

### What should be done in the future

- Keep task status and changelog synchronized as implementation phases land.
- If the ticket is split, preserve this document as the parent architecture and link child tickets explicitly.

### Code review instructions

- Start at the ticket `index.md` and `tasks.md`.
- Verify the design and diary are the focused subdocuments and that scripts live under the ticket workspace.
- Run `docmgr ticket list --ticket GOJA-068` and `docmgr doc list --ticket GOJA-068`.

### Technical details

Ticket path:

```text
ttmp/2026/07/15/GOJA-068--harden-replapi-runtime-lifecycle-session-ownership-and-http-safety
```

Audit baseline:

```text
cc9f18656f02f42e945806bcb6e3b1d86c0658ad
```

## Step 2: Map the subsystem and reproduce hidden failures

This step moved from API impressions to source-backed behavior. I traced runtime construction, session locking, evaluation ordering, persistence transactions, restore, deletion, HTTP routing, CLI/TUI ownership, and protobuf conversion. I then wrote five small probes to isolate behaviors that existing tests did not cover.

The probes turned suspected limitations into concrete acceptance tests. Three findings are more severe than the original help-page review suggested: durable split brain mutates both VMs before SQLite rejects one, a persistence failure permits later journal gaps, and partial profile configuration can silently disable the expected raw timeout.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Investigate all relevant REPL API layers closely enough to identify additional correctness and safety problems, not only document the two already known limitations.

**Inferred user intent:** Ensure the future implementation ticket fixes root causes and catches adjacent bugs before an intern begins coding.

### What I did

- Read and mapped:
  - `pkg/replapi/app.go` and `config.go`
  - `pkg/replsession/service.go`, `evaluate.go`, `persistence.go`, `policy.go`, `observe.go`, and DTOs
  - `pkg/repldb/store.go`, `schema.go`, `read.go`, and `write.go`
  - `pkg/replhttp/proto_handler.go` and `handler.go`
  - `cmd/goja-repl/root.go`, `cmd_serve.go`, and `tui.go`
  - `pkg/repl/adapters/bobatea/replapi.go`
  - `pkg/engine/options.go`, `factory.go`, and `runtime.go`
  - `pkg/runtimeowner/runner.go`
  - `proto/goja/replapi/v1/replapi.proto`
- Read prior REPL architecture, configurable-profile, persistence-correctness, evaluation-control, and protobuf design documents.
- Counted and categorized the subsystem: the focused source/test/transport set is roughly 8,000 lines, with the evaluation pipeline alone at about 800 lines.
- Added five numbered probe programs under the ticket `scripts/` directory.
- Ran focused tests and the race detector:

```bash
go test -race ./pkg/replapi ./pkg/replsession ./pkg/replhttp ./pkg/repldb ./pkg/repl/adapters/bobatea
```

### Why

- Source locking can be race-free while still violating distributed or durable consistency.
- HTTP request contexts behave differently under a real `net/http` server than under a direct handler call with `httptest.ResponseRecorder`.
- A design for leases or app close is unsafe unless evaluation/persistence ordering and current lock behavior are understood first.

### What worked

All five probes compiled and reproduced the targeted behavior.

**HTTP runtime lifetime:**

```text
session=session-8a973f8b-2559-46ce-a222-bec0b38ddea2 runtime_lifetime_error=context canceled
```

**Persistent split brain:**

```text
appA: cell=2 status=ok error=<nil>
appB: response=nil error=persist cell: write evaluation: persist evaluation: insert evaluation: UNIQUE constraint failed: evaluations.session_id, evaluations.cell_id
durable cell ids: 1 2
```

**Partial raw config:**

```text
profile=raw eval_mode=instrumented timeout_ms=0 static_analysis=false binding_tracking=false
```

**Post-execution persistence failure:**

```text
cell2_response_nil=true cell2_error=persist cell: write evaluation: injected persistence failure
cell3_id=3 cell3_result=3 cell3_error=<nil>
snapshot_cell_count=3 snapshot_error=<nil> persisted_cell_ids=[1 3]
```

**Canceled session waiter:**

```text
snapshot_error=<nil> context_error=context deadline exceeded elapsed=301ms
```

The race suite passed, which establishes a useful baseline:

```text
ok github.com/go-go-golems/go-go-goja/pkg/replapi
ok github.com/go-go-golems/go-go-goja/pkg/replsession
ok github.com/go-go-golems/go-go-goja/pkg/replhttp
ok github.com/go-go-golems/go-go-goja/pkg/repldb
ok github.com/go-go-golems/go-go-goja/pkg/repl/adapters/bobatea
```

### What didn't work

- Existing tests did not expose these logical failures. This was not a command failure, but it invalidated the assumption that a passing race suite means session ownership is safe.
- Direct handler tests were insufficient for the request-lifetime issue because they do not necessarily reproduce `net/http` canceling the server request context after `ServeHTTP` returns. The probe had to use `httptest.NewServer`.

### What I learned

- `replsession.Service` uses the operation context for both `WithStartupContext` and `WithLifetimeContext`; this affects create and every restore path, not only the HTTP create route.
- The current per-session mutex serializes complete operations inside one service, but cannot coordinate separate app instances.
- Persistence happens after runtime mutation and in-memory cell append. A write failure is therefore a session health transition, not a normal retryable method error.
- `sync.Mutex` makes caller cancellation ineffective while queued.
- Profile normalization currently treats a profile string and policy preset as separate facts, allowing them to disagree.
- SQLite bootstrap records `currentSchemaVersion` unconditionally and is not sufficient for the lease schema this ticket requires.
- HTTP currently has strict unknown-field protobuf decoding, but no body bound, version check, public protobuf error, or safe internal error redaction.

### What was tricky to build

- The split-brain bug required creating two apps over one store and restoring the same session before either appended the next cell. The underlying cause is not SQLite concurrency itself; it is that ownership is asserted only by an eventual unique cell insert after both VMs have executed.
- The persistence-gap probe needed a stub implementing `replsession.Persistence` so one exact cell write could fail while later writes succeeded. This isolated service ordering from SQLite behavior.
- The canceled-waiter probe used a `WithRuntime` callback as a deterministic lock holder. That avoided relying on JavaScript execution timing and showed the mutex queue issue directly.

### What warrants a second pair of eyes

- Review whether a SQLite lease with fencing is sufficient for all expected deployments or whether some hosts need an external coordinator.
- Review the semantics of a JavaScript cell that executed but failed to commit. External side effects cannot be undone; the proposed degraded state protects future journal consistency but cannot promise transactional JavaScript.
- Review `DeleteSession` overlapping active evaluation. Source inspection shows no session lock in delete; implementation tests should determine the desired grace-versus-interrupt behavior.

### What should be done in the future

- Convert every probe into a deterministic package regression test before behavior changes.
- Keep the standalone probes as investigation evidence until the ticket is complete; archive or retain them after tests supersede them according to ticket policy.
- Add a real v1 SQLite fixture before introducing schema v2.

### Code review instructions

- Run each script with `go run ./ttmp/.../scripts/NN-*.go`.
- Start source review at `pkg/replsession/service.go:126-258`, then follow evaluate persistence at `pkg/replsession/evaluate.go:316-448`.
- Inspect `pkg/repldb/schema.go` for the unique cell invariant and `pkg/replhttp/proto_handler.go:69-87` for request parsing.
- Do not classify the reproduced conflicts as expected user errors; both losing VMs have already advanced.

### Technical details

Probe inventory:

```text
01-probe-http-session-context/main.go
02-probe-persistent-split-brain/main.go
03-probe-partial-profile-config/main.go
04-probe-post-execution-persistence-failure/main.go
05-probe-canceled-waiter/main.go
```

The source audit identified twelve findings, ranked from inherent replay constraints through critical durable-consistency defects. The full evidence table and file references are in the primary design document.

## Step 3: Produce the intern-oriented architecture and implementation guide

This step converted the audit into a concrete design rather than stopping at a bug list. The document begins with terms and package orientation, walks through current create/evaluate/restore/delete/HTTP flows, records probe evidence, defines twelve invariants, proposes APIs and state machines, and ends with phased file-level implementation and testing instructions.

The central design choice is explicit ownership. The app owns the lifetime context, each live session owns one runtime and context-aware operation gate, SQLite owns a per-session lease/fencing epoch, and HTTP requests own only individual operation deadlines. This division resolves the observed failures without pretending replay is VM serialization.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Write a technically precise guide that teaches the subsystem and gives an intern enough API, pseudocode, diagrams, file references, implementation phases, and validation criteria to carry out the work.

**Inferred user intent:** Minimize implementation risk and reviewer burden by making hidden contracts and design decisions explicit before code changes begin.

### What I did

- Replaced the generated design template with a comprehensive implementation guide.
- Added:
  - executive summary, scope, non-goals, terms, and invariants;
  - package and ownership diagrams using portable ASCII;
  - current-state create/evaluate/restore/delete/HTTP flows;
  - exact probe commands and outputs;
  - twelve severity-ranked findings;
  - proposed `replapi`, `replsession`, `repldb`, and HTTP APIs;
  - app/session lifecycle state machines;
  - context-aware operation-gate pseudocode;
  - SQLite lease, renewal, release, and fenced-append algorithms;
  - fail-closed persistence degradation and recovery behavior;
  - schema migration design;
  - configuration normalization repair;
  - protobuf error and HTTP resource-limit design;
  - CLI, TUI, and server shutdown order;
  - eight decision records;
  - eight phased implementation steps;
  - unit, integration, real HTTP, race, failure-injection, migration, and fuzz strategies;
  - intern-specific code review instructions, risks, alternatives, open questions, and references.
- Chose ASCII diagrams instead of Mermaid so the markdown remains readable in Glazed output, terminals, Pandoc, and reMarkable PDF rendering.

### Why

- Lifecycle work is cross-cutting and easy to implement locally while breaking another transport.
- Decision records prevent future contributors from reintroducing request-scoped lifetimes, overloading delete, or treating unique constraints as sufficient ownership.
- A phased plan lets reviews land correctness foundations before leases and HTTP schema changes.

### What worked

- The existing package split supports an incremental design; no wholesale rewrite is required.
- Prior tickets provided useful historical rationale for replay, profiles, SQLite integrity, timeouts, and protobuf boundaries.
- The proposed APIs can be mapped to concrete files and regression tests.

### What didn't work

- The generated design-doc and reference templates were intentionally too sparse for this task and had to be replaced completely.
- A one-page issue summary would not have been enough: the audit showed persistence and lifecycle state machines need explicit algorithms and review invariants.

### What I learned

- The smallest safe persistence repair is not “retry the method.” The service must retain the exact computed record, mark the VM degraded, and reject later source until retry or discard-and-restore.
- Lease/fencing depends on a migration framework; adding `CREATE TABLE IF NOT EXISTS session_leases` directly would repeat the current schema-version weakness.
- An app constructor context is preferable to a background default or compatibility wrapper because compilation forces every host to choose an owner.
- Authentication should remain composable, while body limits, schema validation, error redaction, and server timeouts belong in the transport/command.

### What was tricky to build

- The shutdown design must permit cancellation of active evaluation without making the runtime unreachable if close times out. The proposed state machine keeps an explicit closing state and avoids removing the only pointer before cleanup completes.
- A lease protects the database but cannot roll back external JavaScript effects. The guide separates SQLite fencing guarantees from external idempotency requirements.
- `UnloadSession` semantics are clear for persistent sessions but intentionally remain an open question for in-memory sessions, where unload destroys the only state.

### What warrants a second pair of eyes

- Review the proposed breaking constructor API and whether direct `replsession` users should also require a parent context at construction.
- Review lease TTL/renewal behavior against configurable evaluation timeouts and long restores.
- Review whether commit failure should return a cell report plus typed error or expose diagnostics separately.
- Review public protobuf additions before generated TypeScript changes.

### What should be done in the future

- Implement phase 0 regression tests before changing source behavior.
- Resolve the ten open questions during implementation review and record decisions in the ticket changelog.
- Update the Glazed help page after APIs land; it currently documents the known limitations accurately.

### Code review instructions

- Read the primary guide in order through “Current-State Architecture” before jumping to APIs.
- Check each proposed invariant against the phase acceptance criteria.
- For implementation, review `service.go`, `evaluate.go`, `app.go`, migrations/leases, HTTP, and command shutdown in that order.
- Run all five probes and the focused race suite before starting changes.

### Technical details

Primary guide:

```text
ttmp/2026/07/15/GOJA-068--harden-replapi-runtime-lifecycle-session-ownership-and-http-safety/design-doc/01-repl-api-lifecycle-ownership-and-http-hardening-implementation-guide.md
```

The design proposes these new conceptual operations:

```text
App.Close             close all live runtimes, preserve durable history
App.UnloadSession     close one runtime, preserve durable history
App.RecoverSession    discard degraded/fenced runtime and restore durable head
Store.AcquireLease    establish one active owner and epoch
Store.Persist...Fence append only for valid owner/epoch/expected cell
```

## Step 4: Validate, repair probe packaging, and deliver to reMarkable

The final research step validated both the documentation metadata and the executable evidence. Validation caught a real ticket-packaging problem: five standalone `package main` files in one scripts directory each ran individually, but they could not coexist as one Go package under `go test ./...`. I moved every probe into its own numbered subdirectory, updated all references, reran the probes, and confirmed the ticket passed docmgr doctor.

The reMarkable dry run succeeded. My first real upload command contained a manual path typo and failed before rendering or remote creation; the corrected command then rendered and uploaded the three-document bundle successfully.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Validate that the ticket is internally consistent and executable, then publish the index, implementation guide, and diary as one navigable PDF bundle.

**Inferred user intent:** Receive a trustworthy, portable implementation package rather than markdown that exists only in the repository.

### What I did

- Ran `docmgr doctor --ticket GOJA-068 --stale-after 30`.
- Validated design and diary frontmatter with `docmgr validate frontmatter --suggest-fixes`.
- Checked markdown fence balance and document size.
- Ran `git diff --check`.
- Ran all probes individually and as a package tree.
- Moved each probe from `scripts/NN-name.go` to `scripts/NN-name/main.go`.
- Performed the required reMarkable dry run.
- Uploaded `index.md`, the primary design guide, and this diary as one bundle with ToC depth 2.

### Why

- Ticket-local scripts should not break repository-wide `go test ./...` merely because they share a directory.
- Dry-run and real rendering catch different classes of failures: path/selection mistakes versus Pandoc/PDF/upload errors.
- A bundle keeps orientation, design, and investigation history together for offline review.

### What worked

Documentation validation passed:

```text
## Doctor Report (1 findings)

### GOJA-068

- ✅ All checks passed
```

Frontmatter validation passed for both focused documents. After moving the probes, package validation passed for all five subdirectories, and each probe reproduced its original result.

The upload succeeded:

```text
OK: uploaded GOJA-068 REPL API Lifecycle Hardening.pdf -> /ai/2026/07/15/GOJA-068
```

### What didn't work

The first package-tree test failed because all probes declared `main` and `must` in one package:

```text
# github.com/go-go-golems/go-go-goja/ttmp/2026/07/15/GOJA-068--harden-replapi-runtime-lifecycle-session-ownership-and-http-safety/scripts
ttmp/2026/07/15/GOJA-068--harden-replapi-runtime-lifecycle-session-ownership-and-http-safety/scripts/02-probe-persistent-split-brain.go:15:6: main redeclared in this block
	ttmp/2026/07/15/GOJA-068--harden-replapi-runtime-lifecycle-session-ownership-and-http-safety/scripts/01-probe-http-session-context.go:20:6: other declaration of main
...
FAIL github.com/go-go-golems/go-go-goja/ttmp/2026/07/15/GOJA-068--harden-replapi-runtime-lifecycle-session-ownership-and-http-safety/scripts [build failed]
```

The correction was to make each numbered probe its own Go package directory with `main.go`.

The first real upload used a path missing `design-doc/` and failed before upload:

```text
Error: path not found: /home/manuel/code/wesen/go-go-golems/go-go-goja/ttmp/2026/07/15/GOJA-068--harden-replapi-runtime-lifecycle-session-ownership-and-http-hardening-implementation-guide.md: stat /home/manuel/code/wesen/go-go-golems/go-go-goja/ttmp/2026/07/15/GOJA-068--harden-replapi-runtime-lifecycle-session-ownership-and-http-hardening-implementation-guide.md: no such file or directory
```

I retried with the exact path used by the successful dry run.

### What I learned

- Executable investigation artifacts need package-tree validation, not only one-at-a-time execution.
- Copying the exact dry-run command into the real upload reduces path transcription errors.
- The three-document bundle is large but structurally balanced: the primary guide is 1,711 lines and about 8,745 words; the diary was about 2,774 words before this final step.

### What was tricky to build

- Moving scripts required updating both prose command examples and docmgr-generated `RelatedFiles` frontmatter. Stale paths would have made the ticket pass content review while failing reproducibility.
- The first upload failure produced lengthy CLI help after the concise path error. The root cause was still the single missing `design-doc/` segment, not authentication or rendering.

### What warrants a second pair of eyes

- Review the rendered PDF's long ASCII diagrams and tables for comfortable reMarkable line wrapping during human reading.
- Confirm implementation pull requests keep probe commands and paths current as tests supersede them.

### What should be done in the future

- Begin Phase 0 by converting the probes into failing regression tests.
- Keep the remote bundle immutable as the design baseline; upload a newly named or explicitly forced revision only after substantial design changes.

### Code review instructions

- Run `go test ./ttmp/2026/07/15/GOJA-068--harden-replapi-runtime-lifecycle-session-ownership-and-http-safety/scripts/...`.
- Run each `go run ./ttmp/.../scripts/NN-name` command from the primary guide.
- Run `docmgr doctor --ticket GOJA-068 --stale-after 30`.
- Compare the ticket paths included in the reMarkable dry-run output with the repository files.

### Technical details

Remote destination:

```text
/ai/2026/07/15/GOJA-068/GOJA-068 REPL API Lifecycle Hardening.pdf
```

Bundle contents, in order:

1. `index.md`
2. `design-doc/01-repl-api-lifecycle-ownership-and-http-hardening-implementation-guide.md`
3. `reference/01-investigation-diary.md`

## Step 5: Refine implementation phases into a precise task tracker

The original design had eight sensible phase headings, but `tasks.md` represented each phase with only one broad checkbox. This step converted those placeholders into an operational tracker: 81 open implementation tasks, stable phase/task labels, explicit phase gates, dependency milestones, and clear conditionality around multi-process persistence work.

The design now separates mandatory local correctness from conditional ownership coordination and transport/release integration. Each phase states entry criteria, exact work packages, likely files, validation commands, a completion invariant, and a recommended pull-request boundary.

### Prompt Context

**User prompt (verbatim):** "Ok, separate the implementation into phases, and then each phase has a detailed list of task so that we can precisely keep track of where we are at. Refine the design doc and the tasks"

**Assistant interpretation:** Replace broad phase placeholders with granular, independently checkable tasks and refine the design so phase dependencies, necessity, gates, and progress reporting are unambiguous.

**Inferred user intent:** Make future implementation progress objectively trackable across sessions and contributors without relying on vague percentage estimates or rereading the entire design.

### What I did

- Removed the eight one-line implementation placeholder tasks using `docmgr task remove`.
- Added 81 implementation tasks with `docmgr task add`.
- Assigned stable semantic labels from `P0.1` through `P7.10` plus one `P#.GATE` task per phase.
- Reorganized `tasks.md` into:
  - completed research/design;
  - Milestone A, mandatory correctness core;
  - Milestone B, conditional persistent multi-process ownership;
  - Milestone C, transport hardening and release integration.
- Added phase entry/exit prose directly to `tasks.md`.
- Replaced the design's short phased plan with a detailed dependency matrix and phase-by-phase implementation guide.
- Added exact files, task explanations, validation commands, gate invariants, and recommended PR boundaries for every phase.
- Updated `index.md` to summarize the three milestones and the 81-task tracker.

### Why

- A single checkbox such as “add app lifecycle” hides constructor migration, context derivation, operation gates, unload, close, delete serialization, callback migration, and concurrency tests.
- Stable `P#.N` labels allow diary entries, commits, reviews, and handoffs to refer to the same unit of work.
- Explicit gates prevent a phase from appearing complete because most checkboxes are done while its core invariant still fails.
- Conditional classification answers whether all lease/migration work is necessary for a narrower deployment model.

### What worked

- `docmgr task remove` cleanly removed the obsolete broad tasks without affecting seven completed research tasks.
- `docmgr task add` stamped stable internal IDs while semantic labels preserve human ordering.
- `docmgr task list` now reports 81 open implementation tasks and seven completed research/delivery tasks.
- Phase task order matches the refined design's dependency order.

### What didn't work

- No command failed during this refinement.
- The initial flat output from repeated `docmgr task add` was functionally trackable but not sufficiently readable, so `tasks.md` required explicit milestone and phase headings afterward.

### What I learned

- The implementation naturally divides into three milestones rather than eight equally mandatory phases.
- Phases 0–3 are mandatory correctness work.
- Phases 4–5 depend on whether the product supports multiple processes against one persistent database/session; P5.1 is now an explicit deployment-contract decision.
- Phase 6 has a mandatory safety baseline if `serve` remains supported, even if protobuf error-schema refinements are split into a focused follow-up.
- Phase 7 is not cosmetic: shutdown integration and release validation are required to make earlier lifecycle work effective in real hosts.

### What was tricky to build

- The tracker had to be detailed without making every task dependent on the full lease design. The solution was to keep phases sequential within milestones while stating which work is conditional and which Phase 6 items can begin after Phase 2.
- Task IDs generated by docmgr are opaque, so semantic labels such as `P2.5` and `P5.GATE` are included in task text for stable human references.
- A phase gate must represent an invariant, not merely “all subtasks checked.” Each gate now states the behavior that must be demonstrated.

### What warrants a second pair of eyes

- Confirm whether 81 tasks are the right granularity for implementation review or whether specific tasks should be split further only when a PR begins.
- Review P5.1 before Phase 4 starts; selecting database-wide exclusivity may replace several lease-specific subtasks.
- Confirm the proposed constructor break in P2.2 before downstream migration begins.

### What should be done in the future

- At each implementation session, update tasks by semantic label and record checked IDs in the diary/changelog.
- Do not check a phase gate until its validation commands and invariant pass.
- If a design decision changes task applicability, edit/remove affected tasks through docmgr and record the reason rather than silently skipping them.

### Code review instructions

- Start with `tasks.md` and verify phase headings, semantic labels, and gates.
- Compare every task group with the matching phase in the primary design document.
- Run `docmgr task list --ticket GOJA-068` and confirm 81 open / 7 complete before implementation starts.
- Verify `index.md` describes mandatory versus conditional milestones consistently.

### Technical details

Progress should be reported in this format:

```text
Phase: P2 — Runtime lifecycle and cancellation
Completed: P2.1, P2.2, P2.3
In progress: P2.4
Blocked: P2.5 pending operation-gate API review
Gate: P2.GATE open
Validation: go test ./pkg/replapi ./pkg/replsession (pass)
```

Phase gates:

```text
P0.GATE regression evidence
P1.GATE configuration consistency
P2.GATE lifecycle ownership and shutdown
P3.GATE persistence fail-closed invariant
P4.GATE safe schema migration
P5.GATE single writable durable owner
P6.GATE bounded and stable HTTP transport
P7.GATE integrated release validation
```

## Step 6: Build the complete Phase 0 red-test safety net

Phase 0 is now implemented as an opt-in red-test suite. Eight desired-behavior tests convert the five exploratory probes plus profile variants and delete/evaluate characterization into package-level regressions. They use the `replapi_hardening` build tag so the repository's normal CI remains green while later phases have executable failing contracts to turn green one by one.

A ticket-local verification harness runs each red test independently and checks its exact assertion text. This prevents a setup panic, compile failure, or unrelated error from being mistaken for the expected baseline defect.

### Prompt Context

**User prompt (verbatim):** "build all of P0"

**Assistant interpretation:** Complete tasks P0.1 through P0.8 and P0.GATE by adding deterministic regression tests, recording the expected failures, preserving normal CI, and validating the focused race baseline.

**Inferred user intent:** Establish a trustworthy red/green safety net before production lifecycle, configuration, persistence, or ownership behavior changes begin.

### What I did

- Added `pkg/replhttp/hardening_regression_test.go` for real HTTP request/runtime lifetime behavior.
- Added `pkg/replapi/hardening_regression_test.go` for:
  - canceled queue wait;
  - partial raw config;
  - unknown app profile;
  - unknown session profile;
  - second persistent live owner;
  - delete while a session operation is active.
- Added `pkg/replsession/hardening_regression_test.go` with exact-cell persistence failure injection.
- Added `scripts/06-verify-phase-0-red-tests.sh` to verify all eight expected assertion failures.
- Added `reference/02-phase-0-red-test-baseline-and-execution-guide.md` with commands, test/task mapping, expected output, determinism notes, and test-promotion workflow.
- Ran normal full tests, focused race tests, tagged compile-only tests, individual red tests, and the red-baseline harness.

### Why

- Leaving desired-behavior tests untagged would intentionally break every default CI run before fixes exist.
- Using only `t.Skip` would not prove the defects or guard future implementations.
- An opt-in build tag gives Phase 0 a reproducible red suite while preserving a green default branch.
- Exact failure matching ensures each test is red for the defect it names.

### What worked

Normal validation remained green:

```text
go test ./...                                                     PASS
go test -race ./pkg/replapi ./pkg/replsession ./pkg/repldb \
  ./pkg/replhttp ./pkg/repl/adapters/bobatea                     PASS
```

The tagged packages compile when tests are not selected:

```text
go test -tags replapi_hardening \
  ./pkg/replapi ./pkg/replsession ./pkg/replhttp \
  -run '^$' -count=1                                             PASS
```

The harness confirmed the complete expected red set:

```text
EXPECTED RED: pkg/replhttp/TestHardeningHTTPSessionRuntimeOutlivesCreateRequest
EXPECTED RED: pkg/replapi/TestHardeningCanceledWaiterDoesNotExecuteLate
EXPECTED RED: pkg/replapi/TestHardeningPartialRawConfigUsesRawPreset
EXPECTED RED: pkg/replapi/TestHardeningUnknownAppProfileIsRejected
EXPECTED RED: pkg/replapi/TestHardeningUnknownSessionProfileIsRejected
EXPECTED RED: pkg/replapi/TestHardeningPersistentSessionRejectsSecondLiveOwner
EXPECTED RED: pkg/replapi/TestHardeningDeleteWaitsForActiveSessionOperation
EXPECTED RED: pkg/replsession/TestHardeningPersistenceFailureBlocksLaterEvaluation
Phase 0 red baseline confirmed: 8 expected failures.
```

### What didn't work

The direct tagged test command fails by design. Representative failures were:

```text
runtime lifetime was canceled with create request: context canceled
queued snapshot ignored cancellation and completed late with <nil>
expected raw eval mode, got "instrumented"
expected second App to be rejected before publishing a writable runtime
delete completed while a session operation was active: <nil>
expected degraded session to reject cell 3 before execution; got cell=3 status=ok result=3
expected only cell 1 to be durable after failure, got [1 3]
```

The first persistence regression printed an excessively large `%#v` cell report. I narrowed the assertion to cell ID, status, result, and durable ID list so baseline output remains reviewable.

### What I learned

- The five probes map to eight useful contracts because profile validation needs distinct partial, app-level unknown, and session-level unknown cases.
- `httptest.NewServer` is necessary for P0.1; direct `ServeHTTP` does not provide the same request-lifetime boundary.
- Channel barriers make P0.2 and P0.6 deterministic without relying on JavaScript execution duration.
- Phase 0 tests should be promoted out of the tagged files individually when their owning phase fixes them.

### What was tricky to build

- A red suite must fail without destabilizing normal development. The build tag and exact-failure harness provide both properties.
- The delete characterization needed to observe premature completion without deadlocking the test. It records whether delete returns, releases the active callback, drains both goroutines, then reports the failure.
- The persistence test must prove both sides of divergence: cell 3 executes in memory and durable IDs become `[1,3]`.

### What warrants a second pair of eyes

- Review whether the 100 ms HTTP/queue observation guards are acceptable in CI. They are bounded and use deterministic setup, but Phase 2 may expose stronger lifecycle signals that can remove remaining timing windows.
- Confirm the desired delete contract before P2.10: delete may cancel an active operation, but it must not finish closing while that operation still owns the session slot.
- Confirm individual test promotion is included in later implementation PR review checklists.

### What should be done in the future

- Begin Phase 1 with the three profile tests.
- As each defect is fixed, move its regression to the normal suite and remove it from the red-baseline harness.
- P7 must ensure no fixed contract remains hidden only behind `replapi_hardening`.

### Code review instructions

- Read `reference/02-phase-0-red-test-baseline-and-execution-guide.md` first.
- Run `go test ./...` and the focused race suite; both must pass.
- Run `scripts/06-verify-phase-0-red-tests.sh`; it must report exactly eight expected red tests.
- Inspect the three `hardening_regression_test.go` files and verify no production behavior changed.

### Technical details

Phase 0 files:

```text
pkg/replhttp/hardening_regression_test.go
pkg/replapi/hardening_regression_test.go
pkg/replsession/hardening_regression_test.go
ttmp/2026/07/15/GOJA-068--harden-replapi-runtime-lifecycle-session-ownership-and-http-safety/scripts/06-verify-phase-0-red-tests.sh
ttmp/2026/07/15/GOJA-068--harden-replapi-runtime-lifecycle-session-ownership-and-http-safety/reference/02-phase-0-red-test-baseline-and-execution-guide.md
```

Build tag:

```text
replapi_hardening
```

## Step 7: Implement Phase 1 configuration correctness

Phase 1 now makes profile selection authoritative. Unknown profile text returns a typed error, bare explicit configs receive the complete matching preset, duplicate app/session profile labels must agree, structurally invalid policies are rejected, and explicit policy pointers remain complete replacements rather than hidden merges.

The three profile regressions from P0 were moved into the normal test suite and removed from the expected-red harness. The TUI now delegates profile parsing to `replapi.ParseProfile`, so CLI spelling rules and Go API validation share one implementation.

### Prompt Context

**User prompt (verbatim):** "do P1 in one go"

**Assistant interpretation:** Complete P1.1 through P1.7 and P1.GATE in one implementation pass, including production code, promoted regressions, TUI reuse, API comments/help, and full validation.

**Inferred user intent:** Eliminate configuration ambiguity completely before beginning the larger lifecycle refactor.

### What I did

- Added `ErrUnknownProfile`, `UnknownProfileError`, `ErrProfileMismatch`, `ProfileMismatchError`, and `ErrInvalidSessionPolicy`.
- Added public `ParseProfile` and `ValidateProfile` helpers.
- Changed `ConfigForProfile` to return `(Config, error)` instead of silently falling through to persistence.
- Reworked config normalization to:
  - canonicalize profile names;
  - apply a complete preset to bare `Config{Profile: ...}` values;
  - preserve explicit ID/timestamp values;
  - treat nonzero policies as complete replacements;
  - reject conflicting app/session profile labels;
  - reject unknown eval modes and subordinate persistence flags when persistence is disabled.
- Changed per-session option resolution to return typed validation errors.
- Updated `NewWithConfig` and `CreateSessionWithOptions` to propagate normalization/resolution failures.
- Reused `ParseProfile` from the TUI parser.
- Expanded config tests and promoted three P0 regressions into untagged `config_test.go`.
- Removed those three cases from the expected-red harness; five red regressions remain.
- Updated the Glazed `replapi-guide` profile/policy documentation.
- Serialized `cmd/goja-repl` integration tests after the expanded race run exposed their shared Glazed global-logger initialization.

### Why

- Profile names must not be labels disconnected from effective evaluation and timeout behavior.
- `ConfigForProfile("typo")` must not silently become a persistent config.
- Policy override pointers already represent full structs; introducing implicit field merging would make false/zero values ambiguous.
- Central parsing prevents the TUI and Go API from accepting different spellings or producing unrelated validation behavior.

### What worked

The promoted regressions pass normally:

```text
=== RUN   TestHardeningPartialRawConfigUsesRawPreset
--- PASS
=== RUN   TestHardeningUnknownAppProfileIsRejected
--- PASS
=== RUN   TestHardeningUnknownSessionProfileIsRejected
--- PASS
```

The active red harness now reports exactly five unrelated defects:

```text
EXPECTED RED: pkg/replhttp/TestHardeningHTTPSessionRuntimeOutlivesCreateRequest
EXPECTED RED: pkg/replapi/TestHardeningCanceledWaiterDoesNotExecuteLate
EXPECTED RED: pkg/replapi/TestHardeningPersistentSessionRejectsSecondLiveOwner
EXPECTED RED: pkg/replapi/TestHardeningDeleteWaitsForActiveSessionOperation
EXPECTED RED: pkg/replsession/TestHardeningPersistenceFailureBlocksLaterEvaluation
Phase 0 red baseline confirmed: 5 expected failures.
```

`go test ./...`, focused race tests, `go vet ./pkg/replapi ./cmd/goja-repl`, and tagged compile-only validation passed after the test-harness correction.

### What didn't work

An expanded race run that included all `cmd/goja-repl` tests initially failed with a pre-existing process-global logger race:

```text
WARNING: DATA RACE
Write at ... by goroutine ...:
  github.com/go-go-golems/glazed/pkg/cmds/logging.initLoggerFromMergedSettings()
  github.com/go-go-golems/glazed/pkg/cmds/logging.InitLoggerFromCobra()
  github.com/go-go-golems/go-go-goja/cmd/goja-repl.newRootCommand.func1()

Previous write at ... by goroutine ...:
  github.com/go-go-golems/glazed/pkg/cmds/logging.initLoggerFromMergedSettings()
```

The root-command integration tests were marked `t.Parallel()` even though command execution mutates Glazed's global logger. I removed parallel execution from those tests. The command race suite then passed. No production logging behavior changed.

### What I learned

- Bare explicit config and option-built config require different `AutoRestore` handling: a bare config receives the complete preset, while option-built config preserves an explicit `WithAutoRestore(false)` because its session options already carry preset state.
- Duplicate profile labels can be validated strictly while an explicit nonzero policy remains an intentional full override.
- Persist detail flags without `Persist.Enabled` are structurally invalid and should fail before runtime construction.
- P1 reduced the active hardening red set from eight tests to five.

### What was tricky to build

- `Config` contains a plain boolean for `AutoRestore`, so zero cannot distinguish “unset” from “explicit false.” The implementation uses zero `SessionOptions` as the signal for a bare profile config that needs the complete preset. Callers needing explicit changes should start from a preset helper or options.
- `WithProfile` cannot return an error because `Option` is a mutator. Invalid values are retained in config and rejected by `NewWithConfig`, preserving a typed constructor error.
- Full policy replacement must survive normalization: `WithDefaultSessionPolicy(SessionPolicy{})` intentionally becomes instrumented mode with all observation/persistence booleans disabled.

### What warrants a second pair of eyes

- Review the public `ConfigForProfile(Profile) (Config, error)` signature change and release-note requirement.
- Confirm the bare-config `AutoRestore` rule is acceptable until a future tri-state config representation exists.
- Review whether additional policy relationships beyond eval-mode validity and persistence-detail gating should be rejected or remain explicit custom behavior.

### What should be done in the future

- Begin Phase 2 with lifecycle states and app-owned contexts.
- Keep all five remaining hardening cases in the red harness until their owning phases promote them.
- Document the `ConfigForProfile` signature migration in P7 release notes.

### Code review instructions

- Start with `pkg/replapi/config.go`, especially `ParseProfile`, `normalizeConfig`, and `resolveCreateSessionOptions`.
- Review `pkg/replapi/config_test.go` for preset, mismatch, replacement, and promoted-regression coverage.
- Verify `cmd/goja-repl/tui.go` delegates non-empty input to `ParseProfile`.
- Run `go test ./...`, the focused race suite including `./cmd/goja-repl`, and the five-case red harness.

### Technical details

P1 public error checks use standard `errors.Is`/`errors.As`:

```go
config, err := replapi.ConfigForProfile(profile)
if errors.Is(err, replapi.ErrUnknownProfile) {
    // reject user input
}

var mismatch *replapi.ProfileMismatchError
if errors.As(err, &mismatch) {
    // inspect app/session labels
}
```

P1 changed files include:

```text
pkg/replapi/config.go
pkg/replapi/app.go
pkg/replapi/config_test.go
pkg/replapi/hardening_regression_test.go
cmd/goja-repl/tui.go
cmd/goja-repl/root_test.go
pkg/doc/34-replapi-guide.md
scripts/06-verify-phase-0-red-tests.sh
reference/02-phase-0-red-test-baseline-and-execution-guide.md
```

## Step 8: Implement Phase 2 runtime lifecycle and cancellation

Phase 2 introduces explicit owners and state machines for every app and live session. App construction now requires a parent context, create/restore contexts are startup-only, and all session operations pass through a context-aware capacity-one gate. App close, service close, unload, and delete now share one cancellation and serialization protocol.

Shutdown is non-destructive unless the caller explicitly selects delete. A bounded shutdown that cannot acquire an active operation gate leaves the session reachable as `closing`, allowing a later call to retry rather than leaking an unreachable runtime.

### Prompt Context

**User prompt (verbatim):** "do phase 2"

**Assistant interpretation:** Implement P2.1 through P2.11 and P2.GATE as one cohesive lifecycle change, including breaking API migrations, promoted regressions, host integration, documentation, and race validation.

**Inferred user intent:** Establish deterministic runtime ownership and shutdown before implementing persistence degradation and recovery in Phase 3.

### What I did

- Added exported app, service, and session lifecycle phases and typed closing/closed errors.
- Changed `replapi.New` and `NewWithConfig` to require an explicit parent context and migrated every repository Go caller, including ticket/fuzz scripts.
- Added `replsession.WithLifetimeContext` and one child lifetime context per created/restored session.
- Passed operation contexts only to runtime startup and session contexts to `engine.WithLifetimeContext`.
- Replaced per-session blocking `sync.Mutex` operation ownership with a capacity-one channel gate acquired through `select` on context cancellation.
- Composed caller and session cancellation with `context.WithCancelCause` and `context.AfterFunc`; existing evaluation timeout wraps that operation context.
- Changed `WithRuntime` callbacks to `func(context.Context, *engine.Runtime) error` and documented context honoring, non-escape, and non-reentrancy.
- Added non-destructive `Service.UnloadSession` and `App.UnloadSession`.
- Added retryable, concurrent-safe, idempotent `Service.Close` and `App.Close` with `errors.Join` aggregation.
- Serialized unload/delete/close behind active operations; failed gate acquisition retains the closing map entry.
- Integrated bounded app-before-store shutdown into CLI, TUI, and HTTP server command ownership paths.
- Fixed event-loop startup to call `loop.Start()` directly because that API sets running state synchronously and starts its own goroutine.
- Added tests for lifetime separation, parent cancellation, operation-context cancellation, queued cancellation, retryable timeout, unload versus delete, active JavaScript interruption, concurrent/repeated close, aggregated errors, and close hooks exactly once.
- Promoted the HTTP lifetime, canceled waiter, and delete serialization regressions into normal CI. Two red regressions remain for P3 and P5.
- Updated the Glazed help topic, design decisions, phase status, and red-baseline guide.

### Why

- HTTP request completion must never cancel runtime-owned timers, loops, or module resources.
- A canceled queued request must not execute after the caller has gone away.
- Persistent runtime eviction must not soft-delete durable history.
- Close/delete cannot safely race a callback or JavaScript evaluation already using the runtime.
- Explicit parent contexts and breaking constructor migration make every ownership decision compile-visible.

### What worked

All three P2-owned regressions pass without a build tag:

```text
TestHardeningHTTPSessionRuntimeOutlivesCreateRequest       PASS
TestHardeningCanceledWaiterDoesNotExecuteLate             PASS
TestHardeningDeleteWaitsForActiveSessionOperation         PASS
```

The focused race suite passed after the event-loop startup correction:

```text
ok  github.com/go-go-golems/go-go-goja/pkg/engine
ok  github.com/go-go-golems/go-go-goja/pkg/replsession
ok  github.com/go-go-golems/go-go-goja/pkg/replapi
ok  github.com/go-go-golems/go-go-goja/pkg/replhttp
ok  github.com/go-go-golems/go-go-goja/pkg/repl/adapters/bobatea
ok  github.com/go-go-golems/go-go-goja/cmd/goja-repl
```

The red harness now reports only the P3 and P5 contracts:

```text
EXPECTED RED: pkg/replapi/TestHardeningPersistentSessionRejectsSecondLiveOwner
EXPECTED RED: pkg/replsession/TestHardeningPersistenceFailureBlocksLaterEvaluation
Phase 0 red baseline confirmed: 2 expected failures.
```

### What didn't work

The first full repository compile found one ticket probe missed by the constructor migration:

```text
ttmp/2026/07/15/.../scripts/03-probe-partial-profile-config/main.go:18:60: not enough arguments in call to replapi.NewWithConfig
have (*engine.RuntimeFactory, zerolog.Logger, replapi.Config)
want (context.Context, *engine.RuntimeFactory, zerolog.Logger, replapi.Config)
```

Adding its existing `ctx` argument fixed the compile.

The first expanded race run exposed an engine race rather than a session-state race:

```text
WARNING: DATA RACE
Write ... eventloop.(*EventLoop).run()
Previous read ... eventloop.(*EventLoop).Stop()
...
pkg/engine/factory.go:210
pkg/engine/runtime.go:123
```

`NewRuntime` used `go loop.Start()`, but `EventLoop.Start()` already calls `setRunning()` and launches `go loop.run(true)`. An immediate close could reach `Stop()` before the outer goroutine executed `setRunning()`. Replacing it with synchronous `loop.Start()` established running state before runtime publication; the engine and full focused race suites then passed. A direct immediate-close regression test now exercises this path.

One focused non-race run under simultaneous validation load produced:

```text
TestServiceInteractiveSyncRunawayTimeoutKeepsSessionUsable:
expected ok status after timeout, got "timeout"
```

Twenty isolated repetitions passed. The test gave both the intentional infinite loop and the subsequent healthy instrumented evaluation only 20 ms. I widened the two synchronous runaway test budgets to 100 ms; they still validate interruption and post-timeout reuse without depending on a lightly loaded scheduler.

### What I learned

- Goja Node's `EventLoop.Start` is already asynchronous after synchronously setting lifecycle state; wrapping it in `go` is both redundant and unsafe.
- Session cancellation must participate in gate acquisition, not only operation execution, or queued shutdown waiters can remain blocked.
- App close and session close need separate phases: app errors describe host shutdown, while a timed per-session unload remains inspectable as a typed session-closing error.
- Close aggregation must retain terminal close-hook errors across attempts but must not permanently store a caller's transient deadline error.

### What was tricky to build

- Create/restore publication had to be atomic with service phase checks. Runtime startup occurs outside the service map lock, then publication rechecks `ServicePhaseOpen`; a concurrent close either snapshots the published runtime or causes startup cleanup, never misses a newly published runtime.
- Retryable shutdown cannot remove the session before acquiring its operation gate. The implementation marks and cancels first, keeps the pointer in the map on timeout, and removes it only after runtime close has been attempted under the gate.
- Concurrent close callers need one shared attempt without making a failed bounded attempt terminal. An attempt channel lets waiters share completion; a later caller can become the retry owner if the service remains closing.
- Restore builds in a temporary service. The restored session context must still descend from the destination service lifetime, and temporary map ownership must be removed before publication.
- Close errors and incomplete errors have different lifetimes: terminal runtime-hook failures are accumulated for stable idempotent results, while deadline errors are returned only by the incomplete attempt.

### What warrants a second pair of eyes

- Review lock ordering: service map mutex, per-session stop gate, lifecycle mutex, then operation gate. No operation path acquires them in reverse.
- Review the deliberate choice to allow `UnloadSession` for in-memory profiles, where it destroys the only state.
- Review the breaking constructor and `WithRuntime` callback signatures for downstream release notes.
- Confirm host shutdown error propagation can remain deferred until P7; current CLI/TUI/server defers attempt bounded app close before store close but intentionally discard shutdown errors.

### What should be done in the future

- Begin Phase 3 by adding healthy/degraded/fenced state and retaining the exact uncommitted evaluation record.
- P3 should promote `TestHardeningPersistenceFailureBlocksLaterEvaluation`.
- P5 should promote the remaining cross-app ownership regression after the deployment ownership contract is selected.
- P7 must expose app/store shutdown failures from command paths and document both breaking API migrations.

### Code review instructions

- Start with `pkg/replsession/lifecycle.go`, especially `beginOperation`, `stopSessionState`, and `Service.Close`.
- Then inspect runtime creation/publication in `pkg/replsession/service.go` and operation acquisition in `pkg/replsession/evaluate.go`.
- Review `pkg/replapi/lifecycle.go` and constructor wiring in `pkg/replapi/app.go`.
- Verify `pkg/engine/factory.go` calls `loop.Start()` directly.
- Read `pkg/replsession/lifecycle_test.go` and `pkg/replapi/lifecycle_test.go` for concurrency contracts.
- Run `go test ./...`, the focused race suite, and the two-case red harness.

### Technical details

Lifecycle hierarchy:

```text
host parent
  -> app context
     -> service context
        -> session context
           -> merged caller/session operation context
              -> evaluation timeout context
```

Shutdown ordering:

```text
mark closing
  -> cancel session operations
  -> acquire operation gate within shutdown context
  -> close runtime and run close hooks
  -> optionally soft-delete (DeleteSession only)
  -> mark closed and remove live map entry
```

P2 primary files:

```text
pkg/replsession/lifecycle.go
pkg/replsession/service.go
pkg/replsession/evaluate.go
pkg/replsession/lifecycle_test.go
pkg/replapi/lifecycle.go
pkg/replapi/app.go
pkg/replapi/lifecycle_test.go
pkg/engine/factory.go
pkg/engine/factory_test.go
pkg/replhttp/hardening_regression_test.go
cmd/goja-repl/root.go
cmd/goja-repl/tui.go
cmd/goja-repl/cmd_serve.go
pkg/doc/34-replapi-guide.md
```

## Step 9: Implement Phase 3 fail-closed persistence and recovery

Phase 3 makes a post-execution persistence failure an explicit session state instead of an ordinary error that callers can accidentally ignore. A failed append now keeps the exact durable record, returns the executed cell alongside a typed commit error, and rejects every later evaluation until exact retry succeeds or the suspect VM is discarded.

Cell IDs and visible history now advance only after durable append for persistent sessions. Recovery unloads the diverged VM and replays the last committed source head, while exact retry never executes JavaScript a second time.

### Prompt Context

**User prompt (verbatim):** "Do P3 and P4"

**Assistant interpretation:** Complete all Phase 3 persistence state-machine work, close P3.GATE, then implement and validate the Phase 4 migration framework and P4.GATE.

**Inferred user intent:** Finish the mandatory persistence correctness core and prepare SQLite for the ownership schema without combining lease behavior into the migration phase.

### What I did

- Added healthy, degraded, and fenced session health states with typed `SessionHealthError` values.
- Added typed `CommitError`, `ErrCommitFailed`, and `ErrNoPendingCommit` contracts.
- Refactored persistence into exact `EvaluationRecord` construction followed by append.
- Delayed `nextCellID` and committed history publication until append succeeds.
- Retained the exact record and candidate cell on append failure.
- Returned both `EvaluateResponse` and `CommitError` for executed-but-uncommitted cells.
- Rejected later JavaScript before parsing or execution while degraded or fenced.
- Added `Service/App.SessionHealth` and `Service/App.RetryPendingCommit`.
- Added `App.RecoverSession` as non-destructive unload plus restore from durable source.
- Preserved function source mapping with an in-flight cell pointer that is not published as committed history.
- Began persisting empty cells so durable IDs remain contiguous and replay numbering matches live numbering.
- Updated the Bobatea adapter to emit a returned cell result and then its commit error rather than discarding the response.
- Added service tests for exact retry, failed retry, no JavaScript rerun, contiguous IDs, and fenced rejection.
- Added real SQLite-trigger recovery coverage proving uncommitted VM state is discarded.
- Added transaction rollback tests for evaluation insert, child-row, and injected pre-commit failures.
- Promoted the P0 fail-closed persistence regression into normal CI and reduced the red harness to one ownership test.

### Why

- JavaScript and external side effects cannot be rolled back after execution.
- Retrying source could duplicate side effects and further diverge the VM.
- Publishing a failed cell in normal history would falsely claim durable reconstruction can reproduce it.
- Recovery must replay only committed source, not a best-effort reconstruction of the suspect VM.

### What worked

The exact-retry test demonstrates the required property:

```text
cell 1 commits
cell 2 executes and append fails -> degraded
cell 3 is rejected before execution
retry of the retained cell 2 record fails once, then succeeds
VM x remains 2, proving source was not rerun
next evaluation commits as cell 3
persisted IDs are [1 2 3]
```

The real-store recovery test uses a SQLite trigger to reject cell 2. `RecoverSession` restores cell 1, discards the live `x=99` mutation, and permits a replacement contiguous cell 2 after the trigger is removed.

### What didn't work

No Phase 3 implementation failure remained after the first focused test run. The pre-existing red regression passed immediately after health gating and delayed publication were wired together.

The key behavior intentionally changed: empty/whitespace cells are now persisted. Leaving them live-only would create durable ID gaps and replay renumbering even without a failed append, contradicting the Phase 3 contiguous-journal invariant.

### What I learned

- The exact record must include child binding/doc rows before the first write attempt; rebuilding those after failure could observe a different VM.
- Binding function mapping previously depended on appending the cell early. An `inFlightCell` lookup preserves mapping without exposing uncommitted history.
- A useful commit failure API needs both values: the cell explains what executed, while the typed error explains that durability failed.
- Retry and recovery are semantically different: retry accepts the mutated VM and commits its exact record; recovery rejects that VM state and returns to the durable head.

### What was tricky to build

- Much of evaluation mutates bindings before persistence. Phase 3 cannot roll those mutations back, so health gating—not attempted state reversal—is the safety boundary.
- The pending record must remain byte-for-byte equivalent across retries. Retry calls `PersistEvaluation` directly with the retained value and never reruns report building or JavaScript.
- Session snapshots remain available while degraded, but committed `CellCount` and history exclude the pending cell. The returned failed response is the diagnostic surface for that candidate.
- Restore uses persistence-disabled replay, allowing recovery even while a database trigger still rejects the missing next durable cell.

### What warrants a second pair of eyes

- Review the public choice to return a non-nil response and non-nil `CommitError` together.
- Confirm persisting empty cells is the desired compatibility break for contiguous journals.
- Review that health errors intentionally omit the underlying persistence cause from `Error()` while `CommitError.PersistenceCause()` remains available to trusted Go callers.
- Verify Phase 6 maps commit/degraded errors safely without exposing SQL details over HTTP.

### What should be done in the future

- Phase 5 must set fenced health before stale owners can execute and use the same pre-execution rejection path.
- Phase 6 must add stable protobuf/HTTP error codes for commit failure and degraded/fenced sessions.
- P7 release notes must document response-plus-error behavior and empty-cell persistence.

### Code review instructions

- Start at `pkg/replsession/health.go` and `commitEvaluatedCell` in `pkg/replsession/persistence.go`.
- Follow candidate cell construction and publication in `pkg/replsession/evaluate.go`.
- Review `pkg/replsession/health_test.go` for exact-record identity and no-rerun assertions.
- Review `pkg/replapi/recovery_test.go` for the real SQLite recovery path.
- Run `go test ./pkg/replsession ./pkg/replapi ./pkg/repldb -count=1` and the focused race suite.

### Technical details

```text
healthy --append failure--> degraded
   ^                          | exact retry success
   |                          +-------------------->
   +-- unload + restore ------+

fenced -- unload/ownership reacquisition --> future Phase 5 recovery path
```

Primary P3 files:

```text
pkg/replsession/health.go
pkg/replsession/evaluate.go
pkg/replsession/persistence.go
pkg/replsession/health_test.go
pkg/replapi/app.go
pkg/replapi/recovery_test.go
pkg/repldb/write_failure_test.go
```

## Step 10: Implement Phase 4 transactional SQLite migrations

Phase 4 replaces unconditional schema creation and version stamping with ordered migration descriptors and per-version transactions. Empty databases, real v1 databases, repeated opens, and concurrent opens now share the same version-aware path, while newer databases fail closed with a typed error.

The production schema remains version 1. This is deliberate: Phase 4 establishes mechanics but does not consume schema v2 before Phase 5 selects and implements the ownership schema.

### Prompt Context

**User prompt (verbatim):** (same as Step 9)

**Assistant interpretation:** After P3 is green, implement P4.1–P4.8 with a real fixture, transactional rollback, newer-version rejection, concurrent-open safety, and operator documentation.

**Inferred user intent:** Make future lease schema changes trustworthy and recoverable without landing lease behavior prematurely.

### What I did

- Added `CurrentSchemaVersion`, ordered migration descriptors, migration validation, and version lookup.
- Bootstrapped empty databases by applying migration v1 through the normal migration runner.
- Re-read schema version inside an immediate transaction before applying each migration.
- Recorded the target version only after every statement succeeds.
- Added typed `ErrDatabaseTooNew`, `ErrInvalidSchemaVersion`, and `SchemaVersionError`.
- Added `_txlock=immediate` and bounded busy retry around bootstrap/migration/WAL configuration.
- Moved WAL configuration after transactional migration instead of applying it concurrently in every DSN connection setup.
- Added a real 96 KiB `testdata/repl-v1.sqlite` fixture with a session, evaluation, console event, binding version, and binding doc.
- Added fixture provenance and integrity instructions.
- Tested v1 data preservation, second-open idempotence, hypothetical v2 rollback, future-version rejection, and concurrent empty-database opens.
- Documented backup, WAL consistency, upgrade, retry, forward-only, and no-downgrade procedures in the bundled REPL API guide.

### Why

- `CREATE TABLE IF NOT EXISTS` followed by unconditional version replacement can label a partial schema as current.
- Lease/fencing requires durable schema evolution and cannot safely land on the old bootstrap path.
- Concurrent processes must either observe the completed migration or serialize and re-read; they must not both assume an old version.
- Older binaries must reject future schemas rather than silently stamping them back to version 1.

### What worked

- The real v1 fixture passes `PRAGMA integrity_check` and remains byte-level schema v1 input across repeated opens.
- An injected migration that creates a table and then executes invalid SQL leaves neither the table nor a version change.
- A database manually marked version 999 returns `ErrDatabaseTooNew` with `Found=999` and `Supported=1`.
- Fifty repetitions of the final eight-opener concurrent bootstrap test passed.
- Full repository tests and the focused P3/P4 race suite passed.

### What didn't work

The first repeated concurrent-open run intermittently failed before migration:

```text
concurrent open: bootstrap repl db: enable foreign keys: database is locked
```

The explicit bootstrap foreign-key pragma was redundant because `_foreign_keys=on` is already in the DSN. Removing that write moved serialization to the actual migration, but repeated runs then exposed:

```text
concurrent open: bootstrap repl db: apply migrations:
read repl db schema version: inspect metadata table: database is locked
```

`_journal_mode=WAL` in the DSN made every newly opened connection attempt a journal-mode write before migration coordination. I removed that DSN setting, applied migrations under immediate transactions, configured WAL afterward, and wrapped only bootstrap lock errors in a bounded retry with context cancellation. Fifty repeated concurrent-open runs then passed.

### What I learned

- SQLite busy timeout does not make every schema/journal pragma wait reliably; startup must explicitly retry lock-class errors.
- WAL mode is persistent database state, so it can be configured after migration rather than on every connection DSN.
- Migration transactions must re-read the current version after acquiring their write lock because another opener may have advanced it while they waited.
- A migration framework can be proven without inventing a fake production schema v2: tests can inject a hypothetical failing v2 descriptor while production remains v1.

### What was tricky to build

- Empty databases have no metadata table, so version lookup must distinguish that legitimate version-0 state from a metadata table whose required version row is missing or malformed.
- Concurrent migration safety required coordination at both layers: SQLite immediate transactions for version/DDL atomicity and bounded retry for lock errors raised before or around transaction acquisition.
- The real fixture must not be regenerated from future schemas. Its README establishes append-only fixture history so later migrations retain genuine old inputs.
- Newer-version errors must survive bootstrap wrapping and database close so callers can still use `errors.Is` and `errors.As`.

### What warrants a second pair of eyes

- Review the five-second bootstrap busy-retry ceiling and exponential delay bounds.
- Confirm schema v2 should remain reserved for the Phase 5 ownership migration.
- Review whether future migrations need checksums in addition to ordered integer versions; P4 intentionally does not add them.
- Verify operator documentation is sufficient for WAL-safe backup and forward-only deployment.

### What should be done in the future

- P5.1 must finalize the persistent deployment ownership contract before adding the next descriptor.
- If per-session leases are selected, add them as migration v2 without modifying the v1 fixture.
- Future schema versions should add new immutable fixtures and migration tests rather than replacing `repl-v1.sqlite`.

### Code review instructions

- Start with `pkg/repldb/migrations.go`, then inspect `Store.bootstrap` and `sqliteDSN` in `pkg/repldb/store.go`.
- Review `pkg/repldb/migrations_test.go` and the immutable fixture README.
- Run `sqlite3 pkg/repldb/testdata/repl-v1.sqlite 'PRAGMA integrity_check;'`.
- Run `go test ./pkg/repldb -count=1` and `go test -race ./pkg/repldb -count=1`.
- Stress concurrent startup with `go test ./pkg/repldb -run '^TestConcurrentOpenBootstrapsOneConsistentSchema$' -count=50`.

### Technical details

```text
read version
  -> reject if newer than binary
  -> begin immediate transaction for next version
  -> re-read version under write lock
  -> execute ordered statements
  -> write schema_version
  -> commit
  -> configure persistent WAL mode
```

Primary P4 files:

```text
pkg/repldb/migrations.go
pkg/repldb/store.go
pkg/repldb/schema.go
pkg/repldb/migrations_test.go
pkg/repldb/testdata/repl-v1.sqlite
pkg/repldb/testdata/README.md
pkg/doc/34-replapi-guide.md
```

## Step 11: Implement Phase 5 per-session ownership and fencing

Phase 5 selects and implements the multi-process deployment contract: unrelated persistent sessions may be owned concurrently, but one session has exactly one writable app owner. SQLite schema v2 stores an owner ID, fencing epoch, expiry, and update timestamp; every durable append validates that token and the expected next cell in the append transaction.

Each app now receives an unforgeable random owner ID. Create and restore acquire before runtime publication, evaluation and runtime callbacks renew before use, long operations heartbeat at one-third TTL, and unload/delete/app close expire the lease before the store closes. Expired takeover increments the epoch, and stale apps become fenced before later JavaScript.

### Prompt Context

**User prompt (verbatim):** "do phase 5"

**Assistant interpretation:** Complete P5.1 through P5.10 and P5.GATE, including the deployment decision, schema v2, lease primitives, app/service integration, fencing, heartbeat, release, takeover/recovery tests, and promotion of the final red regression.

**Inferred user intent:** Eliminate persistent-session split brain so concurrent processes cannot mutate independent VMs for the same durable journal.

### What I did

- Accepted the per-session lease contract and rejected database-wide exclusive ownership.
- Added schema-v2 `session_leases` plus expiry index through the Phase 4 migration runner.
- Added typed `ErrSessionOwned`, `ErrLeaseLost`, `ErrWriteConflict`, and metadata-bearing errors.
- Implemented atomic absent/same-owner/expired/conflicting acquire behavior.
- Implemented renewal, safe stale release, explicit expiry, and monotonic epochs across takeover/release.
- Added fenced append validating owner, epoch, expiry, record cell, and durable next cell in one transaction.
- Generated one random owner ID per `App`; exposed it read-only through `OwnerID()`.
- Added configurable lease TTL and injectable clock; owner-ID injection remains package-private for tests.
- Added optional lease ownership to `replsession.Service`, enabled by every persistent `replapi.App`.
- Acquired before create/restore publication and before App restore reads durable history.
- Added heartbeat guards around evaluation, `WithRuntime`, and complete replay.
- Marked stale owners fenced on failed renewal before JavaScript begins.
- Integrated fenced persistence with Phase 3 exact retry and commit-error handling.
- Released leases during unload, delete, app/service close, recovery, and failed startup.
- Protected store-only delete by acquiring ownership before soft deletion.
- Added store tests for simultaneous acquisition, renewal, expiry, takeover, stale epoch, release epoch, expected-cell conflict, and fenced append.
- Added app tests for distinct default owner IDs, active-owner rejection, fake-clock takeover, stale-VM fencing, ownership recovery, heartbeat during long evaluation/replay, delete safety, and sequential process use.
- Promoted the final P0 ownership regression; the expected-red harness is now empty.
- Updated help, design decisions, migration fixture notes, and operator semantics.

### Why

- A uniqueness error after JavaScript executes detects conflict too late.
- Process-local locks cannot coordinate CLI invocations or separate servers.
- Database-wide exclusion would unnecessarily block unrelated sessions.
- Epoch fencing prevents a paused old process from writing after expiry and takeover.
- Expected-cell fencing prevents stale or gapped durable heads even when owner metadata matches.

### What worked

- Eight simultaneous owners contending for an absent lease produce exactly one success and seven typed ownership conflicts.
- Fake-clock takeover advances owner A epoch 1 to owner B epoch 2; owner A is fenced before `x=99` executes.
- After owner B closes, owner A discards its fenced VM, reacquires the released lease at a later epoch, restores durable cell 1, and continues at cell 2.
- A 240 ms evaluation succeeds under a 90 ms TTL because heartbeat renews during JavaScript.
- A 350 ms replay retains ownership under a 90 ms TTL; a third app attempting takeover after two TTLs still receives `ErrSessionOwned`.
- Fenced append tests reject expired leases, stale epochs, and wrong durable heads while preserving contiguous `[1,2]` history.
- The last tagged red ownership test passed and was promoted into normal CI.

### What didn't work

The first focused run correctly broke old tests that simulated a new process without shutting down the previous app:

```text
TestAppRestoresPersistedSessionAndContinuesEvaluation:
restore snapshot: restore session: acquire ownership:
repldb: session has another active owner
```

Those tests now call `App.Close` before constructing/restoring through the next app. This is the intended sequential CLI contract, not a compatibility workaround.

The first full repository run found the same stale test assumption in fuzz persistence setup:

```text
TestFuzzPersistenceBasicRoundTrip:
restore snapshot: repldb: session has another active owner
```

The fuzz tests closed SQLite directly while leaving the app/lease alive. They now close app first and store second in every seed/restore phase.

An API review found that a public `WithOwnerID` option would let two apps deliberately impersonate one owner and bypass the split-brain defense. I replaced it with package-private test injection; production IDs are always generated and only readable through `App.OwnerID()`.

### What I learned

- Lease identity must be per app instance, random, and non-configurable; stable/reused identities weaken epoch protection.
- Release should expire the row rather than delete it so a different next owner receives a monotonically larger epoch.
- Restore must acquire before reading replay history, then verify/renew again before replay to close the read-to-publication race.
- Caller cancellation during renewal must not fence a healthy session; only an ownership verification failure does.
- Heartbeat needs to cover `WithRuntime` because that escape hatch can mutate the VM just like Evaluate.

### What was tricky to build

- App restore and service restore cross package boundaries. The app acquires before durable reads, then transfers the exact lease to `RestoreSessionWithLease`; service cleanup releases it unless a live state assumes ownership.
- Two concurrent restores in one app can share owner/epoch. The losing temporary runtime must not expire the lease now owned by the winning published state.
- Heartbeat cancellation has two meanings: normal caller/session cancellation and actual renewal failure. The guard records/fences only the latter.
- Phase 3 pending commits retain their original write fence. Exact retry renews the current lease but refuses retry if ownership/epoch was lost.
- A store-only delete path previously bypassed the live service. It now acquires the lease before soft deletion so a non-owner cannot delete another process's session.

### What warrants a second pair of eyes

- Review the restore lease-transfer paths, especially concurrent same-app restore and all early returns.
- Review the 30-second default TTL and one-third heartbeat interval for production workloads.
- Confirm expiring rather than deleting released lease rows is acceptable operationally; rows are bounded one-per-session and preserve epochs.
- Review whether lease heartbeat database errors should always fence conservatively or distinguish clearly transient connection failures in a later operational policy.
- Verify Phase 6 maps ownership/fencing errors without exposing owner IDs unless explicitly intended.

### What should be done in the future

- Phase 6 must map `ErrSessionOwned`, `ErrLeaseLost`, `ErrWriteConflict`, and `ErrSessionFenced` to stable safe transport errors.
- External side effects still require host-specific idempotency keys or fencing tokens; SQLite fencing protects only the journal.
- P7 should exercise command shutdown errors rather than silently discarding deferred lease-release failures.

### Code review instructions

- Start with `pkg/repldb/lease.go` and schema-v2 migration in `pkg/repldb/migrations.go`.
- Review `pkg/replsession/ownership.go`, then lease integration in `service.go`, `evaluate.go`, `persistence.go`, and `lifecycle.go`.
- Review App pre-read acquisition and store-only delete in `pkg/replapi/app.go`.
- Read `pkg/repldb/lease_test.go` and `pkg/replapi/ownership_test.go` for all ownership invariants.
- Run `go test ./...`, focused race tests, and the zero-red harness.

### Technical details

```text
no lease -> owner A epoch 1
owner A renew -> owner A epoch 1, later expiry
expired -> owner B epoch 2
owner A append/renew -> fenced
owner B release -> row expired at epoch 2
owner C acquire -> epoch 3
```

Fenced append transaction:

```text
BEGIN IMMEDIATE
verify owner_id + epoch
verify lease_until > now
verify MAX(cell_id)+1 == expected_cell_id == record.cell_id
insert evaluation and all child rows
update session timestamp
COMMIT
```

Primary P5 files:

```text
pkg/repldb/lease.go
pkg/repldb/lease_test.go
pkg/repldb/migrations.go
pkg/replsession/ownership.go
pkg/replsession/service.go
pkg/replsession/evaluate.go
pkg/replsession/persistence.go
pkg/replsession/lifecycle.go
pkg/replapi/app.go
pkg/replapi/config.go
pkg/replapi/ownership_test.go
pkg/doc/34-replapi-guide.md
```

## Step 12: Harden the HTTP and protobuf transport

Phase 6 turned the HTTP adapter from an unbounded, string-error wrapper into a bounded and versioned protobuf-JSON transport. Evaluate requests now have explicit content, body, source, and schema contracts; failures have stable generated envelopes; and internal diagnostics stay in server logs by default.

The ready-made server now applies complete connection limits and refuses a non-loopback listener unless the operator passes an explicit acknowledgement. Real HTTP tests exercise cancellation and resource limits over sockets rather than relying only on direct handler calls.

### Prompt Context

**User prompt (verbatim):** "continue the ongoing implementation work."

**Assistant interpretation:** Continue GOJA-068 at Phase 6 and complete the HTTP/protobuf hardening gate as one cohesive phase.

**Inferred user intent:** Finish the next tracked phase without weakening the lifecycle and ownership guarantees already completed, keep generated clients synchronized, and leave a fully validated handoff for Phase 7.

### What I did

- Added `HandlerConfig`, secure nonzero defaults, handler options, strict JSON media-type/schema validation, byte-counted source limits, and `http.MaxBytesReader` before `io.ReadAll`.
- Added generated protobuf `ErrorResponse`, regenerated Go and TypeScript bindings, added the TypeScript error fixture/decode test, and updated package usage documentation.
- Centralized typed status/code mapping for not-found, ownership, fencing, degraded, commit, lifecycle, cancellation, malformed, oversized, and unexpected errors.
- Added request-ID generation/propagation, `nosniff`, `no-store`, panic recovery, server-side structured diagnostics, and default client redaction.
- Added server read/header/write/idle/max-header limits and `--allow-remote`; loopback remains the default and safe-mode/module allowlisting is documented for intentional remote deployment.
- Added real HTTP tests for streaming body limits, source limits, media type, exact schema version, unknown fields, request/security headers, cancellation recovery, ownership conflict, app shutdown, status mapping, and database-error redaction.
- Updated the implementation guide and the public Glazed REPL API guide with the implemented limits, error table, middleware boundary, and corrected runtime-context behavior.

### Why

- An execution endpoint must reject oversized or incompatible input before expensive JavaScript parsing or full-body allocation.
- Stable protobuf error payloads let Go and TypeScript clients branch on codes without string matching.
- Ownership, persistence, SQL, panic, and plugin internals are useful diagnostics but unsafe remote response content.
- Binding a host-capable JavaScript runtime beyond loopback must be a deliberate operator action, not an incidental address change.

### What worked

- `http.MaxBytesReader` reliably returned 413 for a chunked request with no trusted `Content-Length`.
- The real cancellation test interrupted an infinite JavaScript loop at the client deadline and confirmed the same session accepted a later evaluation.
- A second app restoring a leased session produced the stable 409 `session_owned` envelope without exposing the owner ID.
- Closing SQLite under a live app produced a generic 500 `internal` response while the configured Zerolog sink retained `database is closed` diagnostics.
- Full Go tests, focused race tests, Go vet, Buf lint, all TypeScript checks/build/package smoke tests, and the zero-red harness passed.

### What didn't work

- The first generation command used the configured registry credential and failed exactly:

  ```text
  $ buf lint && buf generate
  Failure: your Buf API token for buf.build is invalid. Run "buf registry login" using a valid Buf API token. For details, visit https://buf.build/docs/bsr/authentication
  ```

  I used an isolated temporary `HOME` to fetch public remote plugins anonymously, without modifying the user's credential configuration.

- Running unscoped `buf generate` generated unrelated repository protos under `pkg/replapi/pb/examples`, `pkg/replapi/pb/pkg`, and matching TypeScript directories. I removed that noise and regenerated only `--path proto/goja/replapi/v1/replapi.proto`.
- The first TypeScript check failed because dependencies were not installed:

  ```text
  sh: 1: tsc: not found
  WARN  Local package.json exists, but node_modules missing, did you mean to install?
  ```

  `pnpm install --frozen-lockfile` restored the locked workspace dependencies, after which typecheck, decode tests, build, pack smoke, and clean-consumer smoke passed.
- The first serve-help assertion expected an unwrapped phrase, but Glazed wrapped `built-in` across lines. The test now checks the semantic markers `--allow-remote` and `authentication` rather than terminal line wrapping.
- The first `go vet ./...` found that the test helper returned a generated protobuf value containing a mutex:

  ```text
  pkg/replhttp/security_test.go:331:9: return copies lock value: github.com/go-go-golems/go-go-goja/pkg/replapi/pb/proto/goja/replapi/v1.ErrorResponse contains google.golang.org/protobuf/runtime/protoimpl.MessageState contains sync.Mutex
  ```

  The helper now returns `*ErrorResponse`; vet then passed.

### What I learned

- Body-limit tests need chunked/unknown-length requests; a normal `strings.Reader` only proves the `Content-Length` fast path.
- Generated message placement affects descriptor indices and diff size. Appending `ErrorResponse` after existing messages preserved all prior message indices and produced a focused additive binding diff.
- Request IDs need validation after any custom generator, not only when accepting the inbound header, because response headers must reject control characters and unreasonable lengths.
- Client cancellation is both a transport concern and a session-liveness concern; proving only that the request fails would miss a poisoned gate/runtime.

### What was tricky to build

- `replsession.CommitError` intentionally carries a populated evaluation response and an error. HTTP cannot simultaneously return the normal success protobuf and `ErrorResponse`, so transport maps the commit failure to 503 `persistence_unavailable`; direct Go callers retain the richer response-plus-error contract.
- Error mapping had to inspect wrapped sentinels in the correct order. Commit failure is distinct from later degraded-session rejection, while lease loss, write conflict, degraded, and fenced state all converge on the safe 409 `session_not_writable` contract.
- The recovery middleware must log panic value and stack server-side but construct the same redacted protobuf envelope used by ordinary internal errors. Security headers and request IDs are installed before dispatch so they also cover recovery and ServeMux responses.
- Buf generation was reproducible only after avoiding the invalid local credential and scoping generation to the public transport proto; both details matter for future release validation.

### What warrants a second pair of eyes

- Review whether `ErrCommitFailed` should remain 503 or be 500; 503 currently communicates temporary persistence unavailability while preserving a stable distinct code.
- Review the default 1 MiB body, 256 KiB source, and 30-second write timeout against production evaluation policies before deployment-specific overrides.
- Confirm the accepted request-ID character/length policy is compatible with the tracing middleware used by downstream hosts.
- Review whether future non-evaluate request bodies need the same content/body contract; current body parsing exists only on the evaluate route because other routes have no request payload.

### What should be done in the future

- Complete Phase 7 host/release integration, migration notes, global validation, ticket bookkeeping, and refreshed reMarkable delivery.
- Hosts exposing this handler remotely must add authentication, authorization, TLS, tenant controls, and rate limiting outside `replhttp`; `--allow-remote` is only an acknowledgement.

### Code review instructions

- Start at `pkg/replhttp/handler.go` for defaults, middleware, redaction, and typed mappings, then read `pkg/replhttp/proto_handler.go` for bounded decoding and route behavior.
- Review `proto/goja/replapi/v1/replapi.proto` beside generated Go/TypeScript bindings and `error_response.golden.json`.
- Read `pkg/replhttp/security_test.go` for transport invariants and `cmd/goja-repl/cmd_serve.go` plus its tests for deployment defaults.
- Validate with `go test ./...`, the focused race command, `go vet ./...`, isolated-home `buf lint`, and all `replapi-types` scripts.

### Technical details

```text
400 invalid_argument | invalid_content_type | unsupported_schema_version
404 session_not_found
409 session_owned | session_not_writable
413 request_too_large | source_too_large
500 internal
503 persistence_unavailable | service_shutting_down | service_unavailable
```

Implemented default resource envelope:

```text
request body: 1 MiB
JavaScript source: 256 KiB
read header: 5s
read: 15s
write: 30s
idle: 60s
max headers: 1 MiB
```

Primary P6 files:

```text
pkg/replhttp/handler.go
pkg/replhttp/proto_handler.go
pkg/replhttp/security_test.go
proto/goja/replapi/v1/replapi.proto
pkg/replapi/pb/proto/goja/replapi/v1/replapi.pb.go
web/packages/replapi-types/src/generated/proto/goja/replapi/v1/replapi_pb.ts
web/packages/replapi-types/src/testdata/error_response.golden.json
cmd/goja-repl/cmd_serve.go
cmd/goja-repl/cmd_serve_test.go
pkg/doc/34-replapi-guide.md
```

## Step 13: Complete host integration and release validation

Phase 7 closed the gap between package-level lifecycle guarantees and the actual command hosts. CLI helpers, the TUI, and the HTTP server now return shutdown failures instead of silently discarding them, while a shared helper enforces app-before-store ordering and a regression test proves the released lease can be reacquired.

The release-facing API surface now explains every breaking ownership change and the protobuf transport migration. All repository callers, generated clients, documentation, ticket artifacts, and validation commands agree on the final behavior.

### Prompt Context

**User prompt (verbatim):** "continue"

**Assistant interpretation:** Continue from the completed Phase 6 gate and finish the remaining Phase 7 integration, documentation, validation, bookkeeping, and delivery work.

**Inferred user intent:** Bring GOJA-068 to a fully reviewed and distributable state rather than stopping with package implementation complete but host shutdown and release guidance unfinished.

### What I did

- Audited every `replapi.New`, `NewWithConfig`, and `WithRuntime` caller; all constructors now supply an intentional parent and all callbacks use the context-bearing signature.
- Added `closeAppAndStore`, which uses a bounded app close, releases runtimes/leases before SQLite, aggregates close failures, and is shared by command, TUI, and server ownership paths.
- Changed `runWithApp`, TUI cleanup, and serve cleanup to preserve both operation errors and shutdown errors rather than dropping deferred failures.
- Confirmed server cleanup occurs only after `Server.Shutdown` and `errgroup.Wait`, and confirmed the TUI adapter closes before its non-owned app.
- Made Bobatea assistance check the operation context before and after direct VM access.
- Added shutdown tests proving orderly lease release enables immediate restore by a second app and premature store closure produces actionable app-close diagnostics.
- Added public method/handler option comments, a breaking-API migration table, shutdown example, HTTP client migration notes, and refreshed REPL usage/type-decoding guidance.
- Updated an older fuzzing design example that still showed the obsolete constructor and store-before-app shutdown pattern.
- Ran the complete Go, race, vet, Glazed, Buf, TypeScript, packaging, red-harness, and documentation validation matrix.

### Why

- Correct package lifecycle primitives are insufficient if production hosts ignore their errors or close SQLite first.
- Breaking constructor and callback signatures need explicit migration guidance so downstream users make ownership decisions instead of substituting accidental background lifetimes.
- Generated transport clients and public help must describe stable `ErrorResponse` behavior and remote-security boundaries consistently.
- The final gate requires reproducible release evidence, not only focused package tests.

### What worked

- The new shutdown test closed app A and its store, then app B immediately restored the same durable session, proving lease release happened before SQLite closure.
- Closing SQLite prematurely caused `closeAppAndStore` to return contextual `close repl app` and `database is closed` diagnostics instead of swallowing the failure.
- The full repository test suite and focused race suite passed on the first Phase 7 validation run.
- Go vet, Glazed lint, Buf lint/scoped generation, TypeScript typecheck/tests/build/package/consumer smoke, `git diff --check`, and the zero-red harness all passed.

### What didn't work

N/A. No Phase 7 implementation or validation command failed.

### What I learned

- Deferred cleanup is safe for ordering but inadequate for release-quality commands when errors are discarded; named returns plus `errors.Join` preserve both primary and shutdown failures.
- The adapter's downstream assistance callback cannot accept a context, but checking `opCtx` immediately before and after synchronous VM access still prevents stale successful completion across shutdown.
- Historical design examples can remain operationally dangerous after compile-visible API changes; repository-wide migration searches must include Markdown and ticket documents, not only Go call sites.

### What was tricky to build

- TUI and serve methods have many early returns. A named return and one deferred `errors.Join` cleanup keeps all paths safe without duplicating shutdown code, while defer registration order still closes the adapter before its non-owned app.
- Server ownership has two cancellation roots: OS signals cancel the server run context without canceling the app parent, while caller cancellation may cancel both. The explicit sequence still stops/waits HTTP before final app/store cleanup for normal serve shutdown and safely tolerates already-canceled runtime contexts.
- Updating old examples required distinguishing intentionally historical defect descriptions from executable-looking guidance. Only unsafe current/pseudocode usage was migrated; evidence quotations remain intact.

### What warrants a second pair of eyes

- Review whether command shutdown should use a configurable timeout rather than the shared five-second constant.
- Review server behavior when a caller cancels the app parent before `Server.Shutdown`; operations cancel early by design, although resource closure still remains ordered.
- Confirm downstream assistance APIs cannot be evolved to receive the operation context directly, which would allow long-running completion providers to stop mid-callback.

### What should be done in the future

- Split and commit the currently uncommitted Phase 0–7 working tree into reviewable commits or a carefully structured pull request.
- Downstream consumers must apply the migration table and regenerate clients before upgrading.
- Production remote deployments still need host-selected authentication, authorization, TLS, rate limiting, auditing, and external-effect idempotency.

### Code review instructions

- Start with `cmd/goja-repl/root.go:closeAppAndStore`, then inspect its use from `runWithApp`, `tui.go`, and `cmd_serve.go`.
- Review `cmd/goja-repl/shutdown_test.go` for ordering and error-propagation evidence.
- Review the Bobatea `WithRuntime` bridge and the API comments in `pkg/replapi/app.go` and `pkg/replapi/config.go`.
- Read the migration and HTTP sections in `pkg/doc/34-replapi-guide.md` and `pkg/doc/04-repl-usage.md`.
- Re-run full tests, focused race, vet, Glazed lint, scoped Buf generation, TypeScript scripts, docmgr doctor, and the zero-red harness.

### Technical details

Host shutdown contract:

```text
stop accepting HTTP / stop UI and bus
wait in-flight handlers or UI workers
App.Close(5-second bounded context)
Store.Close()
return errors.Join(operation error, app close error, store close error)
```

Final breaking API summary:

```go
replapi.New(appCtx, factory, logger, opts...)
replapi.NewWithConfig(appCtx, factory, logger, config)
ConfigForProfile(profile) (Config, error)
app.WithRuntime(ctx, sessionID, func(opCtx context.Context, runtime *engine.Runtime) error)
```

Primary P7 files:

```text
cmd/goja-repl/root.go
cmd/goja-repl/tui.go
cmd/goja-repl/cmd_serve.go
cmd/goja-repl/shutdown_test.go
pkg/repl/adapters/bobatea/replapi.go
pkg/replapi/app.go
pkg/replapi/config.go
pkg/doc/34-replapi-guide.md
pkg/doc/04-repl-usage.md
```

## Step 14: Prepare the review branch and pull request

I created a dedicated review branch, staged the complete GOJA-068 change set, inspected the staged file list and diff, and committed the implementation only after the repository pre-commit checks passed. The hook exposed lint rules that the earlier vet/Glazed validation did not cover, so I repaired those findings rather than bypassing the hook.

The resulting implementation commit contains the complete lifecycle, persistence, ownership, migration, HTTP, generated-client, test, and ticket documentation work. Build artifacts and dependency directories remained ignored and were not staged.

### Prompt Context

**User prompt (verbatim):** "Open a PR"

**Assistant interpretation:** Create a focused Git branch, commit the completed GOJA-068 working tree with repository hooks enabled, push it, and open a GitHub pull request against `main`.

**Inferred user intent:** Move the fully validated ticket into the normal review and CI workflow with a useful PR summary and no accidental local artifacts.

**Commit (code):** 2f39f309112387823c10cfefb7796fcd16fd43b3 — "Harden replapi lifecycle, persistence, and HTTP safety"

### What I did

- Created branch `task/goja-068-replapi-hardening` from current `origin/main`.
- Reviewed status, remotes, ignored build output, staged paths, staged statistics, and secret-like text before committing.
- Kept `node_modules/` and generated package `dist/` output ignored.
- Ran `git diff --cached --check` and committed through the configured Lefthook pre-commit pipeline.
- Fixed all hook findings and reran focused tests, GolangCI-Lint, the complete hook lint suite, code generation, and full Go tests.

### Why

- The ticket spans tightly coupled public API and storage invariants, so one atomic implementation commit prevents intermediate commits from exposing uncompilable constructor or schema states.
- Running the real repository hook is required because it includes lint policies beyond the earlier standalone vet and Glazed checks.

### What worked

- The final commit passed GolangCI-Lint, Glazed lint, generated-code checks, and `go test ./...` through Lefthook.
- The working tree was clean immediately after the implementation commit.
- The immutable SQLite fixture, generated protobuf bindings, and ticket documents were intentionally included; ignored dependency/build outputs were not.

### What didn't work

- The first commit attempt was rejected by the pre-commit lint hook with seven findings:

```text
ttmp/.../scripts/01-probe-http-session-context/main.go:24:20: Error return value of `os.RemoveAll` is not checked (errcheck)
ttmp/.../scripts/01-probe-http-session-context/main.go:28:19: Error return value of `store.Close` is not checked (errcheck)
ttmp/.../scripts/02-probe-persistent-split-brain/main.go:19:20: Error return value of `os.RemoveAll` is not checked (errcheck)
ttmp/.../scripts/02-probe-persistent-split-brain/main.go:23:19: Error return value of `store.Close` is not checked (errcheck)
pkg/replsession/health.go:86:2: missing cases in switch of type replsession.SessionHealth: replsession.SessionHealthHealthy (exhaustive)
pkg/replsession/evaluate.go:30:1: named return "response" with type "*EvaluateResponse" found (nonamedreturns)
pkg/replsession/lifecycle.go:221:1: named return "terminal" with type "bool" found (nonamedreturns)
```

I wrapped probe cleanup calls with explicit ignored assignments, added app cleanup, made the health switch exhaustive, extracted evaluation work behind a local result/deferred-guard pattern without named returns, and replaced lifecycle named returns with local values. The next hook run passed.

### What I learned

- Standalone `go vet` and Glazed lint do not replace the repository's configured GolangCI-Lint policy.
- Lease-guard cleanup can preserve panic-safe deferred behavior without named returns by assigning local result variables inside a closure.
- Ticket probe programs are compiled and linted as part of the repository and must follow production cleanup hygiene.

### What was tricky to build

- Removing named returns from `Evaluate` could have accidentally stopped lease heartbeat cleanup on panic or changed the special response-plus-`CommitError` behavior. A small inner closure retains deferred guard shutdown while the public function returns unnamed values.
- The pre-commit hook runs code generation, including a Dagger-backed frontend build, so generated output had to be checked after the hook to confirm no tracked noise remained.

### What warrants a second pair of eyes

- Review `Service.Evaluate` around `evaluateWithLease` to confirm guard errors still preserve commit-error responses and fence all other failures exactly as before.
- Confirm the single atomic implementation commit is acceptable for review despite the ticket's breadth; the phased diary and design guide provide the detailed navigation path.

### What should be done in the future

- Monitor GitHub CI and address review findings on the same branch.

### Code review instructions

- Follow the phase-by-phase path in the implementation guide and Steps 6–13 of this diary.
- Start code review with configuration/lifecycle, then persistence/migrations/ownership, then HTTP/host integration.
- Verify the PR checks match the successful local pre-commit hook.

### Technical details

```text
branch: task/goja-068-replapi-hardening
base: origin/main at cc9f186
implementation commit: 2f39f309112387823c10cfefb7796fcd16fd43b3
pre-commit: GolangCI-Lint + Glazed lint + go generate + go test ./...
```

## Step 15: Address PR review and security build failures

I reproduced the two failed security jobs locally and implemented the reviewer's stale-owner deletion fix at the storage transaction boundary. Durable soft deletion now has the same owner/epoch/expiry fencing guarantee as evaluation append, and both live-session and store-only app deletion paths use it.

I also upgraded the pinned Go toolchain to the patched standard library and documented the narrowly justified Gosec suppression for protobuf JSON output. New store- and app-level regressions cover the exact takeover/delete scenario from review.

### Prompt Context

**User prompt (verbatim):** "address the review comments and the build errors"

**Assistant interpretation:** Inspect PR #96 review threads and failed GitHub jobs, implement the requested correctness fix, reproduce security failures locally, and push validated corrections.

**Inferred user intent:** Make the PR review-ready and green without dismissing the ownership race or weakening security checks.

**Commit (code):** d8f92378cc64b9ffbdfc1d67dfe71e155955aa4b — "Fence session deletion and fix security checks"

### What I did

- Read the Codex P1 inline review and both failed GitHub Actions logs.
- Added `repldb.Store.DeleteSessionFenced`, which verifies lease row, owner ID, epoch, and non-expiry in the same transaction as the soft delete.
- Extended `LeasePersistence` and routed live persistent deletion through lease renewal plus fenced deletion before lease release/runtime close.
- Routed App's store-only deletion fallback through the same fenced primitive.
- Added store-level stale-token/current-owner delete tests and an app-level expired-takeover test proving the stale live owner cannot soft-delete the new owner's session.
- Upgraded `toolchain go1.26.4` to `go1.26.5` for GO-2026-5856 and GO-2026-4970.
- Replaced the unnecessary `json.RawMessage` conversion at HTTP response write time and added a scoped `#nosec G705` rationale because `protojson` output is served as `application/json` with `nosniff`.
- Ran focused tests/race tests, GolangCI-Lint, the CI-equivalent Gosec command, govulncheck, and the complete pre-commit pipeline.

### Why

- Renewing before delete alone leaves a race between lease verification and the unfenced SQL update; the ownership check and soft delete must share one transaction.
- A stale VM may remain resident after TTL expiry, so lifecycle serialization inside one process cannot substitute for the durable epoch fence.
- The vulnerability job failures were standard-library issues fixed by the next Go patch release, not dependency call-path changes.
- Protobuf JSON may contain user-derived strings, but it is encoded JSON rather than HTML and is explicitly delivered with non-sniffable JSON headers; a localized suppression records that trust boundary.

### What worked

- `TestDeleteSessionFencedRejectsStaleOwnerAfterTakeover` proves stale lease A cannot delete after lease B takes epoch 2, while B can delete.
- `TestStaleLiveOwnerCannotDeleteSessionAfterLeaseTakeover` reproduces the review scenario through two Apps and confirms app B remains live after app A's rejected delete.
- `govulncheck ./...` under Go 1.26.5 reports zero affecting vulnerabilities.
- The CI-equivalent Gosec invocation reports zero issues and recognizes one additional justified suppression.
- Focused race tests and the full pre-commit lint/generation/test pipeline passed.

### What didn't work

GitHub's original jobs failed exactly because:

```text
Go Vulnerability Check:
GO-2026-5856 found in crypto/tls@go1.26.4; fixed in go1.26.5
GO-2026-4970 found in os@go1.26.4; fixed in go1.26.5

GoSec Security Scan:
pkg/replhttp/proto_handler.go:207 - G705 (CWE-79): XSS via taint analysis
_, _ = w.Write(json.RawMessage(body))
```

No local command failed while implementing the fixes.

### What I learned

- Ownership fencing must cover destructive metadata changes, not only append operations.
- Expiring a lease row on release is harmless for stale tokens only when every durable mutation validates that token transactionally.
- The workflow's `go-version-file: go.mod` honors the toolchain directive, so patching the directive is sufficient to move security jobs to Go 1.26.5.

### What was tricky to build

- A separate renew followed by ordinary deletion would still be vulnerable. `DeleteSessionFenced` therefore rechecks the current durable token and expiry inside the deletion transaction.
- Stale deletion should dispose the stale runtime while preserving the durable session. The service returns a fenced error, performs only a stale no-op lease release, closes/removes the old VM, and leaves the current owner's durable/live state intact.
- Gosec's taint rule cannot infer `protojson` encoding and response headers, so changing from `json.RawMessage` to `[]byte` alone would not establish safety; the suppression needed an explicit encoding/content-type rationale.

### What warrants a second pair of eyes

- Review transaction behavior in `DeleteSessionFenced`, especially lease validation before the update and stale-owner error typing.
- Confirm that closing/removing the stale VM after rejected deletion is the desired fail-closed lifecycle behavior.
- Review the G705 suppression alongside middleware header installation and `pbconv.MarshalJSON`.

### What should be done in the future

- Monitor rerun GitHub checks and respond to any further review findings.
- Keep the pinned Go toolchain current as future standard-library advisories land.

### Code review instructions

- Start with `pkg/repldb/lease.go:DeleteSessionFenced` and its store regression.
- Follow the call through `pkg/replsession/lifecycle.go` and `pkg/replapi/app.go`.
- Read `TestStaleLiveOwnerCannotDeleteSessionAfterLeaseTakeover` for the reported end-to-end race.
- Verify with focused race tests, govulncheck, the workflow Gosec command, and the repository pre-commit hook.

### Technical details

```text
owner A epoch 1 expires
owner B acquires epoch 2
owner A DeleteSession
  -> renewal/token check fails
  -> no sessions.deleted_at update
  -> stale VM closes and is evicted
owner B remains live and durable session remains visible
```
