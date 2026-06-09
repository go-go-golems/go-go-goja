# Changelog

## 2026-06-08

- Initial workspace created


## 2026-06-08

Created external Go HTTP host integration implementation guide and diary

### Related Files

- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/ttmp/2026/06/08/XGOJA-EXTERNAL-HTTP-HOST--support-external-go-http-hosts-for-generated-xgoja-runtimes/design-doc/01-external-go-http-host-integration-implementation-guide.md — Detailed intern-facing design and implementation guide
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/ttmp/2026/06/08/XGOJA-EXTERNAL-HTTP-HOST--support-external-go-http-hosts-for-generated-xgoja-runtimes/reference/01-investigation-diary.md — Investigation and delivery diary
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/ttmp/2026/06/08/XGOJA-EXTERNAL-HTTP-HOST--support-external-go-http-hosts-for-generated-xgoja-runtimes/tasks.md — Task checklist for future implementation phases


## 2026-06-08

Uploaded external HTTP host guide bundle to reMarkable

### Related Files

- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/ttmp/2026/06/08/XGOJA-EXTERNAL-HTTP-HOST--support-external-go-http-hosts-for-generated-xgoja-runtimes/design-doc/01-external-go-http-host-integration-implementation-guide.md — Primary uploaded guide
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/ttmp/2026/06/08/XGOJA-EXTERNAL-HTTP-HOST--support-external-go-http-hosts-for-generated-xgoja-runtimes/reference/01-investigation-diary.md — Upload evidence and diary
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/ttmp/2026/06/08/XGOJA-EXTERNAL-HTTP-HOST--support-external-go-http-hosts-for-generated-xgoja-runtimes/tasks.md — Delivery checklist


## 2026-06-08

Step 2: added app HostServices helpers and HostOptions ConfigureServices hook

### Related Files

- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/xgoja/app/assets.go — HostServices set/add helpers
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/xgoja/app/host.go — ConfigureServices hook before runtime factory construction
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/xgoja/app/host_services_test.go — Host service helper and visibility tests
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/ttmp/2026/06/08/XGOJA-EXTERNAL-HTTP-HOST--support-external-go-http-hosts-for-generated-xgoja-runtimes/reference/01-investigation-diary.md — Step 2 diary entry
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/ttmp/2026/06/08/XGOJA-EXTERNAL-HTTP-HOST--support-external-go-http-hosts-for-generated-xgoja-runtimes/tasks.md — Phase 1 tasks checked


## 2026-06-08

Step 3: wired ConfigureServices into generated package and source-fragment bundle templates

### Related Files

- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/cmd/xgoja/internal/generate/generate_test.go — Generated API and smoke coverage
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/cmd/xgoja/internal/generate/templates/bundle_fragment.go.tmpl — Source-fragment bundle service hook
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/cmd/xgoja/internal/generate/templates/runtime_package.go.tmpl — Generated package Options and NewBundle service hook
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/ttmp/2026/06/08/XGOJA-EXTERNAL-HTTP-HOST--support-external-go-http-hosts-for-generated-xgoja-runtimes/reference/01-investigation-diary.md — Step 3 diary entry
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/ttmp/2026/06/08/XGOJA-EXTERNAL-HTTP-HOST--support-external-go-http-hosts-for-generated-xgoja-runtimes/tasks.md — Phase 2 tasks checked


## 2026-06-08

Step 4: added HTTP provider external gojahttp host mode

### Related Files

- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/xgoja/providers/http/http.go — ExternalHostService and listener ownership logic
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/xgoja/providers/http/http_test.go — External host route registration and occupied-port no-listen tests
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/ttmp/2026/06/08/XGOJA-EXTERNAL-HTTP-HOST--support-external-go-http-hosts-for-generated-xgoja-runtimes/reference/01-investigation-diary.md — Step 4 diary entry
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/ttmp/2026/06/08/XGOJA-EXTERNAL-HTTP-HOST--support-external-go-http-hosts-for-generated-xgoja-runtimes/tasks.md — HTTP provider tasks checked


## 2026-06-08

Step 5: added gojahttp route introspection

### Related Files

- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/gojahttp/host.go — Host.Routes delegation
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/gojahttp/route_registry.go — RouteDescriptor and Registry.Routes
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/gojahttp/route_registry_test.go — Copy-safe route descriptor tests
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/xgoja/providers/http/http_test.go — External host route descriptor assertion
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/ttmp/2026/06/08/XGOJA-EXTERNAL-HTTP-HOST--support-external-go-http-hosts-for-generated-xgoja-runtimes/reference/01-investigation-diary.md — Step 5 diary entry
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/ttmp/2026/06/08/XGOJA-EXTERNAL-HTTP-HOST--support-external-go-http-hosts-for-generated-xgoja-runtimes/tasks.md — Route introspection and focused tests checked


## 2026-06-08

Step 6: documented generated package service injection and HTTP external host usage

### Related Files

- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/cmd/xgoja/doc/02-user-guide.md — Generated package ConfigureServices and HTTP external-host mention
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/cmd/xgoja/doc/11-provider-runtime-config-and-host-services.md — Host-supplied services and HTTP provider example
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/examples/xgoja/14-generated-runtime-package/README.md — Generated package example documents ConfigureServices
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/ttmp/2026/06/08/XGOJA-EXTERNAL-HTTP-HOST--support-external-go-http-hosts-for-generated-xgoja-runtimes/design-doc/01-external-go-http-host-integration-implementation-guide.md — Implementation status added
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/ttmp/2026/06/08/XGOJA-EXTERNAL-HTTP-HOST--support-external-go-http-hosts-for-generated-xgoja-runtimes/reference/01-investigation-diary.md — Step 6 diary entry and commit hash backfill


## 2026-06-08

Step 7: added blue/green xgoja hot reload manager and polling watcher

### Related Files

- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/cmd/xgoja/doc/02-user-guide.md — Hot reload paragraph for generated package hosts
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/xgoja/hotreload/manager.go — Blue/green reload
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/xgoja/hotreload/manager_test.go — Last-known-good and smoke failure tests
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/xgoja/hotreload/watch.go — Polling file watcher with debounce
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/xgoja/hotreload/watch_test.go — File-change reload test
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/ttmp/2026/06/08/XGOJA-EXTERNAL-HTTP-HOST--support-external-go-http-hosts-for-generated-xgoja-runtimes/design-doc/01-external-go-http-host-integration-implementation-guide.md — Hot reload implementation status
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/ttmp/2026/06/08/XGOJA-EXTERNAL-HTTP-HOST--support-external-go-http-hosts-for-generated-xgoja-runtimes/reference/01-investigation-diary.md — Step 7 diary entry
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/ttmp/2026/06/08/XGOJA-EXTERNAL-HTTP-HOST--support-external-go-http-hosts-for-generated-xgoja-runtimes/tasks.md — RuntimeManager task checked


## 2026-06-08

Step 8: designed opt-in hot reload for generated HTTP serve commands

### Related Files

- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/ttmp/2026/06/08/XGOJA-EXTERNAL-HTTP-HOST--support-external-go-http-hosts-for-generated-xgoja-runtimes/design-doc/02-http-serve-hot-reload-implementation-guide.md — Serve --hot-reload design and phased implementation plan
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/ttmp/2026/06/08/XGOJA-EXTERNAL-HTTP-HOST--support-external-go-http-hosts-for-generated-xgoja-runtimes/reference/01-investigation-diary.md — Step 8 planning diary entry
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/ttmp/2026/06/08/XGOJA-EXTERNAL-HTTP-HOST--support-external-go-http-hosts-for-generated-xgoja-runtimes/tasks.md — Detailed hot reload serve task checklist


## 2026-06-08

Step 9: added per-runtime host service injection to xgoja runtime factory

### Related Files

- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/xgoja/app/factory.go — NewRuntimeFromSectionsWithHostServices and service layering
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/xgoja/app/host_services.go — layeredHostServices implementation
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/xgoja/app/host_services_test.go — Per-runtime service visibility test
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/xgoja/providerapi/commands.go — RuntimeFactoryWithHostServices optional interface
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/ttmp/2026/06/08/XGOJA-EXTERNAL-HTTP-HOST--support-external-go-http-hosts-for-generated-xgoja-runtimes/reference/01-investigation-diary.md — Step 9 diary entry
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/ttmp/2026/06/08/XGOJA-EXTERNAL-HTTP-HOST--support-external-go-http-hosts-for-generated-xgoja-runtimes/tasks.md — Phase 2 tasks checked


## 2026-06-08

Step 10: added HTTP serve hot reload command flags

### Related Files

- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/xgoja/providers/http/serve.go — Hot reload settings section and decode helper
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/xgoja/providers/http/serve_test.go — Serve command schema asserts hot reload fields
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/ttmp/2026/06/08/XGOJA-EXTERNAL-HTTP-HOST--support-external-go-http-hosts-for-generated-xgoja-runtimes/reference/01-investigation-diary.md — Step 10 diary entry
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/ttmp/2026/06/08/XGOJA-EXTERNAL-HTTP-HOST--support-external-go-http-hosts-for-generated-xgoja-runtimes/tasks.md — Phase 3 tasks checked


## 2026-06-08

Step 11: implemented xgoja HTTP serve hot reload execution path

### Related Files

- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/xgoja/providers/http/serve.go — serve --hot-reload manager/server/watch/smoke implementation
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/xgoja/providers/http/serve_test.go — Provider-level hot reload serve/status/reload/last-known-good test
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/ttmp/2026/06/08/XGOJA-EXTERNAL-HTTP-HOST--support-external-go-http-hosts-for-generated-xgoja-runtimes/reference/01-investigation-diary.md — Step 11 diary entry
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/ttmp/2026/06/08/XGOJA-EXTERNAL-HTTP-HOST--support-external-go-http-hosts-for-generated-xgoja-runtimes/tasks.md — Phase 4 tasks checked


## 2026-06-08

Step 12: added generated-binary serve --hot-reload integration coverage

### Related Files

- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/cmd/xgoja/internal/generate/generate_test.go — Generated binary hot reload/status/last-known-good test
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/ttmp/2026/06/08/XGOJA-EXTERNAL-HTTP-HOST--support-external-go-http-hosts-for-generated-xgoja-runtimes/reference/01-investigation-diary.md — Step 12 diary entry
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/ttmp/2026/06/08/XGOJA-EXTERNAL-HTTP-HOST--support-external-go-http-hosts-for-generated-xgoja-runtimes/tasks.md — Phase 5 tasks checked


## 2026-06-08

Step 13: documented final generated serve --hot-reload behavior and validated focused/full tests

### Related Files

- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/cmd/xgoja/doc/02-user-guide.md — User guide hot reload command and semantics
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/cmd/xgoja/doc/12-tutorial-http-serve-jsverbs.md — Detailed tutorial hot reload section
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/examples/xgoja/13-http-serve-jsverbs/README.md — Example README hot reload usage
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/ttmp/2026/06/08/XGOJA-EXTERNAL-HTTP-HOST--support-external-go-http-hosts-for-generated-xgoja-runtimes/reference/01-investigation-diary.md — Step 13 diary entry
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/ttmp/2026/06/08/XGOJA-EXTERNAL-HTTP-HOST--support-external-go-http-hosts-for-generated-xgoja-runtimes/tasks.md — Final validation tasks updated

