---
Title: Implementation Diary
Ticket: GJ-03-FLAMEGRAPH-UI
Status: active
Topics:
  - ui
  - tooling
  - goja
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
Summary: Step-by-step diary of flamegraph UI implementation.
LastUpdated: 2026-02-18T10:30:00-05:00
WhatFor: Track implementation progress, decisions, and tricky bits.
WhenToUse: Reference during code review or future maintenance.
---

# GJ-03 Flamegraph UI — Implementation Diary

## Entry 1: Initial implementation (2026-02-18 ~10:25 EST)

### What I read
- The plan doc at `reference/01-flamegraph-web-ui-plan.md` specifies:
  - Profile manifest YAML schema with artifacts + comparisons
  - Backend endpoints: `/api/profiles/{phase}`, `/api/profile-artifact/{phase}/{id}`
  - Security: manifest-only lookup, path traversal prevention, regular-file check
  - UI: Benchmarks/Profiles sub-nav toggle, comparison cards, artifact table
- Existing GC-04 profile summary YAML at `various/profiles/runtime_spawn_profile_summary.yaml`
  shows the real data shape. 6 SVGs already exist (~170-180KB each).

### What I did
1. Created `serve_profiles.go` with:
   - **Data model**: `profileManifest`, `profileArtifact`, `profileComparison` structs
   - **Manifest loading**: `loadProfileManifest()` from YAML
   - **Artifact lookup**: `findArtifact()` by ID
   - **Safe file serving**: `safeResolvePath()` with path traversal prevention,
     regular file check, repo root prefix enforcement
   - **MIME detection**: `detectMime()` with explicit mappings for svg/pprof/txt/yaml
   - **HTTP handlers**: `handleProfiles` (HTML fragment), `handleProfileArtifact`
     (file serving with optional `?download=1`)
   - **View data construction**: `buildProfilesViewData()` resolving comparison
     artifact references into view structs
   - **HTML template**: `profilesFragmentTemplate` with comparison cards (baseline/
     candidate/diff) and artifact table with View/Download actions

2. Created `serve_profiles_test.go` with 7 tests:
   - `TestLoadProfileManifest` — YAML parsing round-trip
   - `TestFindArtifact` — lookup by ID, missing ID
   - `TestSafeResolvePath` — valid path, traversal attack, empty, nonexistent, directory
   - `TestDetectMime` — explicit mime, extension fallback
   - `TestBuildProfilesViewData_NoManifest` — nil manifest
   - `TestBuildProfilesViewData_WithManifest` — URLs, benchmark shortening, comparison resolution
   - `TestKindLabel` — display labels for artifact kinds

3. Updated `serve_command.go`:
   - Wired `/api/profiles/` and `/api/profile-artifact/` routes
   - Added Benchmarks/Profiles sub-nav toggle in index template
   - Added profile-card and sub-nav CSS styles
   - Updated JS: `switchSubNav()`, `loadCurrentView()`, `runPhase()` resets to benchmarks

4. Created `profiles.yaml` manifest with 6 real artifacts from GC-04 and 1 comparison

### What worked
- Clean separation: all profile logic in one file, no changes to existing types
- `safeResolvePath` catches all traversal variants in tests
- Manifest-only artifact lookup means raw paths never come from query params

### What was tricky
- The plan's YAML schema is slightly different from the existing GC-04 summary YAML.
  I went with the plan's schema (normalized artifact list + comparisons) since it's
  designed for the UI. The existing GC-04 YAML could be converted to this format.

### How to verify
1. `go test ./cmd/goja-perf/ -v` — all 21 tests pass
2. `go run ./cmd/goja-perf serve` then open http://127.0.0.1:8090
3. Click "Profiles" tab — should show comparison card + artifact table
4. Click "View" on any SVG — should open flamegraph in new tab

## Entry 2: Validation and commit (2026-02-18 ~16:45 EST)

### What happened
- First commit attempt caught by errcheck linter: `f.Close()` error not checked.
  Fixed with `defer func() { _ = f.Close() }()`.
- Created real `profiles.yaml` manifest in the GJ-01 phase output directory
  pointing to the 6 existing GC-04 SVG flamegraphs.
- All 7 tasks (F1-F7) checked off.

### Files created
- `cmd/goja-perf/serve_profiles.go` — 340 lines: types, handlers, template
- `cmd/goja-perf/serve_profiles_test.go` — 190 lines: 7 test functions
- `ttmp/.../GJ-01-PERF/.../profiles.yaml` — manifest with 6 artifacts, 1 comparison
- This diary

### Files modified
- `cmd/goja-perf/serve_command.go` — added routes, sub-nav toggle, CSS, JS

### Test count: 21 total (7 format + 2 shutdown + 5 streaming + 7 profiles)

### Commit: 235d077
