# Tasks

## Phase 1: Provider metadata + reusable dtsgen library

- [ ] 1.1 Add TypeScript declaration metadata to `providerapi.Module` (`TypeScript *spec.Module` or `DTSDescriptor *spec.Module`)
- [ ] 1.2 Wire first-party core provider descriptors through `pkg/xgoja/providers/core/core.go`
- [ ] 1.3 Wire first-party host provider descriptors through `pkg/xgoja/providers/host/host.go`
- [ ] 1.4 Inspect/wire HTTP provider descriptors if it exposes modules that should be typed
- [ ] 1.5 Create `pkg/xgoja/dtsgen` for runtime-spec + provider-registry → d.ts rendering
- [ ] 1.6 Implement alias normalization with deep-copy, avoiding mutation of provider descriptors
- [ ] 1.7 Implement strict/non-strict missing descriptor behavior
- [ ] 1.8 Unit-test descriptor propagation, alias renaming, strict behavior, validation errors, and duplicate aliases

## Phase 2: Generated package/binary type exposure

- [ ] 2.1 Add generated package APIs: `TypeScriptDeclarations() (string, error)` and/or `WriteTypeScriptDeclarations(io.Writer) error`
- [ ] 2.2 Add a generated/default `types` cobra command instead of a global `--emit-types` flag
- [ ] 2.3 Attach `types` command to default xgoja roots and generated package helpers where appropriate
- [ ] 2.4 Test generated package declaration output
- [ ] 2.5 Test generated binary `types` command output
- [ ] 2.6 Test aliases in `xgoja.yaml` appear as declaration module names

## Phase 3: Sidecar-backed `xgoja gen-dts`

- [ ] 3.1 Add `cmd/xgoja/cmd_gen_dts.go`
- [ ] 3.2 Generate a temporary sidecar Go module that imports provider packages from `xgoja.yaml`
- [ ] 3.3 Reuse xgoja go.mod requirement/replacement logic for provider packages and `--xgoja-replace`
- [ ] 3.4 Run the sidecar with `go run .` and capture rendered d.ts output
- [ ] 3.5 Support `--out`, `--check`, `--strict`, `--work-dir`, and `--keep-work`
- [ ] 3.6 Test first-party provider specs, check-mode mismatch, and keep-work inspection

## Phase 4: Future HTTP serving (separate ticket)

- [ ] 4.1 Decide whether generated web hosts should serve declarations at `/xgoja/types.d.ts`
- [ ] 4.2 Define route conflict and security behavior before implementation
