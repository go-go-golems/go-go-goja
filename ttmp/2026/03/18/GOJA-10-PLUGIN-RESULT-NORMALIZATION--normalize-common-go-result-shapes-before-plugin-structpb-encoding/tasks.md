# Tasks

## Phase 1: Ticket Setup

- [x] Create GOJA-10 for plugin result normalization.
- [x] Write a compact design note with scope, non-goals, and validation plan.
- [x] Seed the implementation diary.

## Phase 2: Implementation

- [x] Add recursive result normalization to `pkg/hashiplugin/sdk/convert.go`.
- [x] Keep `*structpb.Value` as the escape hatch and preserve simple scalar behavior.
- [x] Normalize common typed slices such as `[]string`, `[]int`, `[]float64`, and `[]bool`.
- [x] Normalize common `map[string]T` maps and recurse through nested values.
- [x] Return clear errors for unsupported result shapes.

## Phase 3: Validation

- [x] Add focused tests for typed slices, typed maps, nested values, and unsupported shapes.
- [x] Run `go test ./pkg/hashiplugin/sdk -count=1`.
- [x] Run `go test ./... -count=1`.

## Phase 4: Closeout

- [x] Update the GOJA-10 diary with commands, failures, and commit hashes.
- [x] Update `changelog.md` with reviewable slices.
- [x] Run `docmgr doctor --ticket GOJA-10-PLUGIN-RESULT-NORMALIZATION --stale-after 30`.
- [x] Upload the refreshed GOJA-10 bundle to reMarkable and verify the remote listing.
