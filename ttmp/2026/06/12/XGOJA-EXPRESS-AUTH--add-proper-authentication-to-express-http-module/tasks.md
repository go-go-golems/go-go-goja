# Tasks

## Completed research and delivery

- [x] Create ticket workspace and import preliminary auth API source
- [x] Analyze current Express/gojahttp/xgoja HTTP provider design
- [x] Write MVP authentication API design and implementation guide
- [x] Upload ticket documentation bundle to reMarkable
- [x] Write Express-style middleware and router auth alternative design
- [x] Upload updated ticket bundle with Express-style middleware design

## Phase 0 — Finalize implementation direction and task breakdown

- [x] Record the selected direction: Go-backed fluent staged route builders as the primary API
- [x] Update ticket tasks with detailed implementation phases
- [x] Update diary with the implementation kickoff and rationale
- [x] Commit the ticket/task planning update

## Phase 1 — RoutePlan model and host auth interfaces

- [x] Add `pkg/gojahttp` route-plan types: `RoutePlan`, `SecuritySpec`, `ResourceSpec`, `ValueSource`, `Actor`, `ResourceRef`
- [x] Add host auth service interfaces: `Authenticator`, `ResourceResolver`, `Authorizer`, optional future placeholders for body/CSRF/audit
- [x] Add sentinel auth errors and HTTP status mapping helpers for unauthenticated, forbidden, and not-found cases
- [x] Extend `HostOptions` with `Auth AuthOptions`
- [x] Extend `Route` with optional `Plan *RoutePlan`
- [x] Add `Registry.AddPlanned` and `Host.RegisterPlanned`
- [x] Add route-plan validation for method/path/security mode/action/resource parameter references
- [x] Add unit tests for planned route registration and validation
- [x] Commit Phase 1

## Phase 2 — Planned route dispatch and secure context

- [x] Add planned-route dispatch branch in `Host.ServeHTTP`
- [x] Implement actor authentication before handler invocation
- [x] Implement resource resolution from typed value sources such as `idFromParam` and `tenantFromParam`
- [x] Implement authorization using host-provided `Authorizer`
- [x] Build Go-owned secure JS context with `ctx.actor`, `ctx.request`, `ctx.body`, `ctx.params`, `ctx.resource(name)`, and `ctx.resources`
- [x] Preserve existing return-value and promise handling behavior for planned handlers
- [x] Add host-level integration tests for public, authenticated, unauthorized, missing resource, and resource success paths
- [x] Commit Phase 2

## Phase 3 — Express Go-backed fluent builders

- [x] Expose `app.route(method, pattern)` as a staged builder
- [x] Expose Go-backed `express.user()` auth spec builder
- [x] Expose Go-backed `express.resource(type)` resource spec builder
- [x] Implement strict runtime type validation so `.auth(...)` and `.resource(...)` reject plain JS objects/maps
- [x] Implement staged objects so `.handle(...)` is unavailable until `.public()` or `.auth(...).allow(...)`
- [x] Register compiled plans through `Host.RegisterPlanned`
- [x] Add aliases only when useful (`idFromParam` primary, `fromParam` compatibility alias; `tenantFromParam` primary, `withinTenantParam` compatibility alias)
- [x] Add Express integration tests for public route, auth route, resource route, and invalid spec object errors
- [x] Commit Phase 3

## Phase 4 — TypeScript declarations and user docs

- [x] Update `modules/express/typescript.go` with staged builder, auth spec, resource spec, actor/resource context, and planned handler types
- [x] Update `pkg/doc/18-express-module.md` with secure planned route examples and compatibility notes
- [x] Add troubleshooting notes for common registration-time errors
- [x] Run targeted docs/type generation tests if available
- [x] Commit Phase 4

## Phase 5 — Validation, examples, and provider integration

- [x] Add/adjust xgoja HTTP provider tests to ensure generated runtimes can use planned public routes
- [x] Add an example script demonstrating public, self, and resource-bound routes
- [x] Run `go test ./pkg/gojahttp ./modules/express ./pkg/xgoja/providers/http -count=1`
- [x] Run broader test subset if targeted tests pass
- [x] Update diary with final validation, commands, failures, and review instructions
- [x] Update changelog and mark completed implementation tasks
- [x] Commit final docs/test updates

## Phase 6 — xgoja/v2 runtime-plan HTTP migration

- [x] Read `cmd/xgoja/doc/16-migrating-to-xgoja-v2.md` and assess the HTTP-specific migration rules
- [x] Merge the xgoja/v2 runtime-plan cutover from PR #76 into the Express auth branch
- [x] Replace remaining HTTP provider test references to removed `app.RuntimeSpec` / `app.ModuleInstanceSpec` with `app.RuntimePlan` / `app.RuntimeModulePlan`
- [x] Verify the HTTP provider uses command-scoped v2 `SourceRegistry` for `serve` command-set jsverb discovery
- [x] Validate `pkg/xgoja/providers/http` tests after the RuntimeSpec removal
- [ ] Re-run xgoja TypeScript HTTP smoke against the v2 spec after the merge commit lands
- [ ] Re-run Express auth host and Keycloak auth host smokes after the v2 merge commit lands
- [ ] Update PR handoff with the v2 migration commit and validation results

## Future / out of MVP

- [ ] Add `.body(...)` with a Go-owned schema registry and validator
- [x] Add `.csrf()` for unsafe cookie-authenticated browser routes
- [x] Add `.audit(...)` for Go-owned structured audit emission
- [x] Add strict host mode to reject legacy raw routes in production
- [ ] Consider Express-style middleware/router support after the planned-route auth core is stable
- [x] Hard-cut Express verb helpers over to staged planned route builders
- [x] Update tests, examples, and docs for planned verb helper migration
- [x] Add dedicated Express auth user guide help entry
- [x] Add Express planned auth migration tutorial help entry
- [x] Add runnable Express auth host example with smoke coverage
