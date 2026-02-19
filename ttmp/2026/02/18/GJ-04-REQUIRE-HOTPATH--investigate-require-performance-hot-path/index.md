---
Title: Investigate require() performance hot path
Ticket: GJ-04-REQUIRE-HOTPATH
Status: active
Topics:
    - analysis
    - goja
    - tooling
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: perf/goja/bench_test.go
      Note: Existing require benchmarks and runtime spawn benchmarks to profile
    - Path: cmd/goja-perf/phase1_types.go
      Note: Task definitions include BenchmarkRequireLoading execution
    - Path: cmd/goja-perf/phase1_run_command.go
      Note: Run pipeline for collecting benchmark output and summaries
    - Path: engine/runtime.go
      Note: Open/New runtime path potentially paying require/console setup costs
    - Path: ttmp/2026/02/18/GC-04-ENGINE-FACTORY--implement-enginefactory-for-reusable-runtime-setup/various/profiles/runtime_spawn_profile_summary.yaml
      Note: Prior profiling evidence implicating require/native module setup
ExternalSources: []
Summary: Investigation ticket to validate whether require/module bootstrap dominates runtime spawn costs and to identify optimizations.
LastUpdated: 2026-02-18T17:00:00-05:00
WhatFor: Organize hypothesis-driven profiling and evidence capture around require overhead.
WhenToUse: Use when diagnosing runtime initialization hotspots and planning require-path optimizations.
---

# Investigate require() performance hot path

## Overview

This ticket investigates whether `require` and native module bootstrap are the
dominant contributors to runtime initialization cost in `go-go-goja`. It tracks
hypotheses, profiling evidence, and next optimization options.

## Key Links

- Investigation plan: `reference/01-require-performance-investigation-plan.md`
- Investigation diary: `reference/02-diary.md`
- Tasks: `tasks.md`
- Changelog: `changelog.md`

## Status

Current status: **active**

## Topics

- analysis
- goja
- tooling

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
