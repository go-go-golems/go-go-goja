---
Title: Investigation Diary
Ticket: GOJA-050
Status: active
Topics:
    - fuzzing
    - replapi
    - testing
    - security
    - goja
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Chronological investigation diary for GOJA-050 fuzzing analysis"
LastUpdated: 2026-04-20
WhatFor: "Track the step-by-step investigation of the go-go-goja replapi fuzzing surface"
WhenToUse: "When continuing fuzzing work or reviewing the investigation process"
---

# Investigation Diary

## Goal

Track the step-by-step investigation of go-go-goja's replapi layer for fuzzing opportunities, recording what was explored, what worked, what didn't, and what was discovered.

---

## Step 1: Codebase Mapping

We mapped the full directory structure of the go-go-goja repository at `/home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja`. The repository contains 256 Go source files (excluding `ttmp/`).

### Prompt Context

**User prompt (verbatim):** "Create a new docmgr ticket and analyze how we can create a fuzzer for the go-go-goja package, and in particular the replapi part of it."

**Assistant interpretation:** Analyze the replapi package and its dependencies to design a comprehensive fuzzing strategy.

**Inferred user intent:** Produce a detailed, intern-ready design document for fuzzing the replapi layer, with working proof-of-concept experiments.

### What I did
- Listed all Go source files in the repository (256 files).
- Read the key files: `pkg/replapi/app.go`, `pkg/replapi/config.go`, `pkg/replapi/app_test.go`, `pkg/replsession/service.go`, `pkg/replsession/types.go`, `pkg/replsession/evaluate.go`, `pkg/replsession/policy.go`, `pkg/replsession/rewrite.go`, `pkg/replsession/observe.go`, `pkg/replsession/persistence.go`, `engine/factory.go`, `engine/runtime.go`, `pkg/runtimeowner/runner.go`, `pkg/runtimeowner/types.go`, `pkg/runtimebridge/runtimebridge.go`, `pkg/repldb/types.go`, `pkg/repldb/store.go`, `pkg/repl/adapters/bobatea/replapi.go`, `pkg/repl/adapters/bobatea/replapi_test.go`.
- Mapped the architecture into 8 layers.

### Why
Understanding the full data flow from user input to VM execution is essential for identifying every fuzzable surface.

### What worked
- The codebase is well-structured with clear separation of concerns.
- Tests in `pkg/replapi/app_test.go` and `pkg/repl/adapters/bobatea/replapi_test.go` provide excellent examples of how to set up test infrastructure (factory creation, store setup, session lifecycle).

### What didn't work
- N/A (pure investigation step).

### What I learned
- The three profiles (raw, interactive, persistent) exercise very different code paths.
- The `buildRewrite` function in `rewrite.go` performs fragile string slicing by AST indices — this is the highest-risk area.
- The `runtimeowner.Runner` uses goroutine-based scheduling with channel-based request/response — potential race condition surface.

### What was tricky to build
- N/A (investigation only).

### What warrants a second pair of eyes
- The `runtimeValueKind` function in `observe.go` has a `default` case that catches several Go types as "unknown" — should these be handled explicitly?

### What should be done in the future
- Audit `runtimeValueKind` for completeness against all goja value types.

### Code review instructions
- Start with `pkg/replsession/evaluate.go` and trace both paths (raw and instrumented).
- Key symbols: `Evaluate`, `evaluateInstrumented`, `evaluateRaw`, `buildRewrite`.

### Technical details
- 256 Go source files in the repository.
- ~2,000 lines in the core evaluation pipeline (replapi + replsession).

---

## Step 2: Experiment 01 — Basic Corpus Runner

We created a standalone Go program that exercises the raw evaluation path with 37 hand-picked JavaScript inputs.

### What I did
- Created `scripts/01-basic-replapi-fuzz/main.go`.
- Ran it against a raw-profile app.

### What worked
- All 37 inputs passed without panics.
- Raw mode handles edge cases well (empty strings, unicode, type coercion).

### What didn't work
- N/A (all inputs processed cleanly).

### What I learned
- Creating a factory + app + session takes ~1ms — acceptable for fuzz harnesses.
- The raw path is very robust for common edge cases.

---

## Step 3: Experiment 02 — Native Go Fuzz

We created a `go test -fuzz` harness for the raw evaluation path.

### What I did
- Created `scripts/02-native-go-fuzz/fuzz_test.go` with three fuzz targets: `FuzzEvaluateRaw`, `FuzzEvaluateInstrumented`, `FuzzSessionLifecycle`.
- Ran `FuzzEvaluateRaw` for 12 seconds.

### What worked
- 33,043 executions in 12 seconds (~2,750 exec/sec average).
- 63 new interesting inputs discovered (coverage expansion).
- No panics found in raw mode.

### What didn't work
- Initial timeout at 60 seconds (the fuzz run took longer than expected). Fixed by reducing fuzztime.

---

## Step 4: Experiment 03 — Rewrite Pipeline Fuzz 🐛 BUG FOUND

We created a fuzz target specifically for the instrumented (interactive) evaluation path.

### What I did
- Created `scripts/03-rewrite-pipeline-fuzz/fuzz_test.go`.
- Ran `FuzzRewritePipeline`.

### What worked
- **The fuzzer immediately found a panic!** Empty string input causes `index out of range [0] with length 0` in `buildRewrite`.

### What didn't work
- The crash was found in 0.23 seconds — the empty string was in the seed corpus as an edge case.

### What I learned
- The `finalExpressionStatement` function doesn't guard against empty `result.Program.Body`.
- This validates that even a minimal seed corpus can find real bugs.

### What was tricky to build
- The panic occurs deep inside `buildRewrite` → `finalExpressionStatement`, which accesses `result.Program.Body[len(result.Program.Body)-1]` without checking length.

---

## Step 5: Experiment 04 — Persistence Fuzz

We created a fuzz target for persistence round-trips (create → persist → restore → continue).

### What I did
- Created `scripts/04-persistence-fuzz/fuzz_test.go`.
- Ran `FuzzPersistenceRestore` for 10 seconds.

### What worked
- 1,938 executions, no crashes.
- Persistence and restore are robust for fuzz-derived inputs.

---

## Step 6: Documentation

We wrote the comprehensive design document with:
- Architecture overview with ASCII diagrams
- Attack surface analysis (7 targets)
- Seed corpus design (10 categories)
- Phased implementation plan (4 phases)
- Code walkthrough for interns (10 files)
- Experiment results

### What I did
- Wrote the design doc section by section (one edit per section to avoid timeout).
- Related all key source files via docmgr.

### What worked
- Section-by-section approach avoided the timeout issues.
- The document is comprehensive (~1,000+ lines).

### What should be done in the future
- Upload to reMarkable for offline reading.
- Create the actual `./fuzz/` package with production harnesses.
