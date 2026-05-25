---
Title: XGOJA-013 Diary
Ticket: XGOJA-013
DocType: reference
Topics:
  - xgoja
  - review
  - testing
Summary: Chronological diary for addressing PR 40 code review issues.
---

# Diary

## Step 1: Opened PR review fix ticket

Goal: address Codex PR 40 review comments and proactively inspect nearby xgoja runtime-profile/capability logic for similar subtle bugs.

Known review comments:

1. `pkg/xgoja/app/run.go`: `run` command precomputes selected module descriptors for the default profile at command construction time, but allows `--runtime` override. Runtime creation uses the user-selected profile while initializers still use the default-profile descriptors.
2. `pkg/xgoja/app/module_sections.go`: `selectedModuleDescriptors` attaches package capabilities only to the first module from each package, leaving later descriptors from the same package without capabilities.

## Step 2: Implemented PR review fixes and nearby hardening

### Review issue 1: stale selected descriptors for `--runtime` overrides

The review was correct. `run` built command sections from the configured default runtime and also stored `selectedModules` from that default runtime. At execution time it decoded `settings.Runtime` and created the runtime from that selected profile, but still initialized with the stale default descriptors.

I fixed this in all built-in commands with runtime override flags, not just `run`:

- `eval`
- `run`
- `repl` / TUI

Each command now recomputes `factory.selectedModuleDescriptors(settings.Runtime)` after parsing command values and passes those descriptors to runtime initializers. The command schema is still necessarily built from the configured default runtime; dynamic schemas for arbitrary runtime overrides remain out of scope.

Regression tests added:

- `TestRunCommandRuntimeOverrideInitializesSelectedRuntimeProfile`
- `TestEvalCommandRuntimeOverrideInitializesSelectedRuntimeProfile`

Both use two profiles with the same `fixture` section but different initializer behavior, proving the selected runtime profile initializer runs.

### Review issue 2: package capabilities missing from later modules

The review was also correct that `selectedModuleDescriptors` only attached capabilities to the first selected module from a package. I changed descriptor construction so every selected module descriptor carries that package's capabilities.

Because package capabilities are package-scoped, blindly applying them once per selected module would duplicate sections and initializers when a runtime selects multiple modules from the same package. I therefore moved the single-application guard into `providerutil`:

- `CollectConfigSections` dedupes by `(packageID, capabilityID)`.
- `InitRuntimeFromSections` dedupes by `(packageID, capabilityID)`.

This keeps full descriptors useful for downstream filtering while preserving package-level semantics for full runtime profiles.

Regression tests added:

- `TestRuntimeFactoryAttachesPackageCapabilitiesToEverySelectedModule`
- `TestCollectConfigSectionsDedupesSamePackageCapability`
- `TestInitRuntimeFromSectionsDedupesSamePackageCapability`

### Nearby inspection

Added and ran:

```bash
ttmp/2026/05/25/XGOJA-013--address-pr-40-xgoja-code-review-issues/scripts/01-review-runtime-profile-descriptor-flows.sh
```

Output saved to:

```text
ttmp/2026/05/25/XGOJA-013--address-pr-40-xgoja-code-review-issues/sources/01-review-runtime-profile-descriptor-flows.txt
```

The scan showed the same runtime override pattern in `eval` and `repl`, which is why I fixed those alongside `run`. `jsverbs` does not expose an equivalent runtime override flag in this flow.

### Validation

```bash
go test ./pkg/xgoja/providerutil ./pkg/xgoja/app -count=1
go test ./pkg/xgoja/providerutil ./pkg/xgoja/app ./pkg/xgoja/providers/http -count=1
go test ./pkg/xgoja/... ./cmd/xgoja/... -count=1
```

In `discord-bot`:

```bash
go test ./pkg/xgoja/provider -count=1
go test ./pkg/xgoja/provider ./internal/jsdiscord ./pkg/botcli -count=1
```

All passed.
