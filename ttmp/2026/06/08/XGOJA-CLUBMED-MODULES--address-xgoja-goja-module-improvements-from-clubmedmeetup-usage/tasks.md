# Tasks

## JSVerb source filtering focus

- [x] Create focused JSVerb-only implementation guide.
- [x] Extend JSVerb source specs with include/exclude/extensions fields.
- [x] Extend jsverbs scan options with include/exclude filtering.
- [x] Pass JSVerb filters through generated runtime scan dispatch.
- [x] Add validation for empty JSVerb filter entries.
- [x] Add focused tests for ScanDir/ScanFS filtering, validation, and embedded runtime spec preservation.
- [x] Run focused gofmt and go test validation.
- [x] Fix any validation/test failures.
- [x] Update user-facing JSVerb/xgoja documentation.
- [x] Update diary and changelog with final JSVerb implementation result.

## Deferred from broader guide

- [ ] Implement selected runtime module inventory command in generated xgoja binaries.
- [ ] Add xgoja doctor warning for provider packages without version or replace.
- [ ] Move Express HTTP listener startup out of require("express") module load while preserving normal app autostart.
- [ ] Add filesystem backend capability metadata for host and embedded read-only fs modules.

## Done before refocus

- [x] Create ticket workspace in go-go-goja/ttmp.
- [x] Read ClubMedMeetup second-edition source analysis.
- [x] Write broad intern-facing design and implementation guide.
- [x] Write investigation diary.
