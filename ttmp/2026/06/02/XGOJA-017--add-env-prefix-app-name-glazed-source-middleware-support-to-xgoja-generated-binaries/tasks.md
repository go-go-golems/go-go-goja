# Tasks

## Phase 0: Revise plan according to review

- [x] Add a narrowed implementation plan to `tasks.md` that separates MVP env support from config/profile follow-ups.
- [x] Treat `appName` and `envPrefix` as different concepts: app identity vs shell-safe env namespace.
- [x] Prefer runtime helper functions over generated Go closure snippets for behavior.

## Phase 1: MVP — app name and env prefix support

- [x] Add `appName` and `envPrefix` to the build-time xgoja YAML spec.
- [x] Add `appName` and `envPrefix` to the runtime embedded xgoja spec.
- [x] Add shell-safe env-prefix derivation and validation helpers.
- [x] Wire a Glazed middleware factory from the runtime spec instead of hardcoding `CobraCommandDefaultMiddlewares`.
- [x] Propagate the middleware factory through `HostOptions`, `Options`, built-in commands, JS verb commands, and command-provider commands.
- [x] Use `appName` for root logging/help framework identity, falling back to `name`.
- [x] Add focused tests for env-prefix derivation, env precedence, and existing default behavior.
- [x] Update xgoja buildspec docs with the MVP fields and examples.

## Phase 2: Config-file support

- [x] Read existing buildspec load/validation tests before adding config schema.
- [x] Add `config` schema only after Phase 1 is passing.
- [x] Implement config plan construction in normal Go helper code, not template snippets.
- [x] Add concrete config examples showing section slug, CLI flag, env var, and resulting value.
- [x] Add integration tests for config < env < CLI precedence.

## Phase 3: Profiles and advanced source middleware exploration

- [x] Re-evaluate naming to avoid confusion with xgoja runtime profiles.
- [x] Decide whether profile support should be `glazedProfiles`, `parameterProfiles`, or deferred.
- [x] Inspect Glazed profile tests before proposing public YAML.
- [x] Do not add arbitrary `middlewares:` YAML until there is a concrete use case.

**Decision:** Profile support is deferred. Xgoja already uses the word "profile" for runtime profiles (`runtimes:`). Adding Glazed-style parameter profiles would create naming collisions. The current env + config support covers the 90% use case. If a user needs profiles, they can use Glazed's `--config` flag with multiple config files. Arbitrary `middlewares:` YAML is deferred until a concrete use case emerges.

## Phase 4: Review and release hardening

- [x] Build all existing `examples/xgoja/*` specs and confirm backward-compatible behavior.
- [x] Add a new minimal env/config example after Phase 2.
- [x] Update diary and changelog after each completed phase.
- [x] Fix generator `RenderEmbeddedSpec` to include new fields in embedded JSON.
- [x] Verify end-to-end precedence with actual generated binary.
