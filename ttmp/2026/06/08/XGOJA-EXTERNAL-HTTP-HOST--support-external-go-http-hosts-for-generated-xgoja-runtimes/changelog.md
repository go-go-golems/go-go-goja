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

