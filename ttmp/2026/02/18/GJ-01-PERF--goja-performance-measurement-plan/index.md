---
Title: Goja Performance Measurement Plan
Ticket: GJ-01-PERF
Status: active
Topics:
    - goja
    - analysis
    - tooling
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/goja-perf/main.go
      Note: New phase-1 command surface
    - Path: perf/goja/README.md
      Note: How to execute and compare benchmark runs
    - Path: perf/goja/bench_test.go
      Note: Dedicated benchmark implementation section for this ticket
    - Path: perf/goja/phase2_bench_test.go
      Note: Phase-2 benchmark implementation
    - Path: ttmp/2026/02/18/GJ-01-PERF--goja-performance-measurement-plan/reference/01-goja-performance-benchmark-plan.md
      Note: Primary implementation plan
    - Path: ttmp/2026/02/18/GJ-01-PERF--goja-performance-measurement-plan/reference/02-diary.md
      Note: Investigation diary
    - Path: ttmp/2026/02/18/GJ-01-PERF--goja-performance-measurement-plan/various/phase1-run-results.yaml
      Note: Ticket artifact for phase-1 execution
    - Path: ttmp/2026/02/18/GJ-01-PERF--goja-performance-measurement-plan/various/phase2-run-results.yaml
      Note: Phase-2 ticket execution artifact
ExternalSources: []
Summary: Ticket workspace for designing and implementing Goja performance benchmarking in go-go-goja.
LastUpdated: 2026-02-18T13:45:00-05:00
WhatFor: Track benchmark design, implementation, and operationalization for Goja performance testing.
WhenToUse: Use as entry point for all GJ-01-PERF artifacts.
---




# Goja Performance Measurement Plan

## Overview

This ticket defines and implements a benchmark framework for measuring Goja performance in `go-go-goja`, with emphasis on VM lifecycle costs, JS loading, and bidirectional Go/JS call overhead.

## Key Links

- Benchmark plan: `reference/01-goja-performance-benchmark-plan.md`
- Investigation diary: `reference/02-diary.md`
- Benchmark code section: `perf/goja/`

## Status

Current status: **active**

## Tasks

See [tasks.md](./tasks.md).

## Changelog

See [changelog.md](./changelog.md).

## Structure

- `reference/`: implementation plan and diary
- `scripts/`: ticket-specific helper scripts if needed
- `various/`: scratch notes and exploratory artifacts
