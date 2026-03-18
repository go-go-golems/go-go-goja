# Tasks

## Phase 1: Ticket Setup

- [x] Create GOJA-10 for plugin result normalization.
- [x] Write a compact design note with scope, non-goals, and validation plan.
- [x] Seed the implementation diary.

## Phase 2: Implementation

- [ ] Add recursive result normalization to `pkg/hashiplugin/sdk/convert.go`.
- [ ] Keep `*structpb.Value` as the escape hatch and preserve simple scalar behavior.
- [ ] Normalize common typed slices such as `[]string`, `[]int`, `[]float64`, and `[]bool`.
- [ ] Normalize common `map[string]T` maps and recurse through nested values.
- [ ] Return clear errors for unsupported result shapes.

## Phase 3: Validation

- [ ] Add focused tests for typed slices, typed maps, nested values, and unsupported shapes.
- [ ] Run `go test ./pkg/hashiplugin/sdk -count=1`.
- [ ] Run `go test ./... -count=1`.

## Phase 4: Closeout

- [ ] Update the GOJA-10 diary with commands, failures, and commit hashes.
- [ ] Update `changelog.md` with reviewable slices.
- [ ] Run `docmgr doctor --ticket GOJA-10-PLUGIN-RESULT-NORMALIZATION --stale-after 30`.
- [ ] Upload the refreshed GOJA-10 bundle to reMarkable and verify the remote listing.
