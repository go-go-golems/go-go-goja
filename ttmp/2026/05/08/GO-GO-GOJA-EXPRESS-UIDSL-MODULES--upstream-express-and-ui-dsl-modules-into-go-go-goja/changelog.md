# Changelog

## 2026-05-08

- Initial workspace created


## 2026-05-08

Created initial design guide and task plan for upstreaming Express-style HTTP hosting and ui.dsl into go-go-goja.

### Related Files

- /home/manuel/code/wesen/2026-05-07--db-browser/ttmp/2026/05/08/GO-GO-GOJA-EXPRESS-UIDSL-MODULES--upstream-express-and-ui-dsl-modules-into-go-go-goja/design-doc/01-go-go-goja-express-and-ui-dsl-module-upstreaming-design-guide.md — Initial upstreaming analysis and design guide
- /home/manuel/code/wesen/2026-05-07--db-browser/ttmp/2026/05/08/GO-GO-GOJA-EXPRESS-UIDSL-MODULES--upstream-express-and-ui-dsl-modules-into-go-go-goja/tasks.md — Implementation task sequence


## 2026-05-08

Upstreamed db-browser/goja-hosting-site HTTP hosting and ui.dsl into go-go-goja, migrated both downstream apps to the new packages, deleted local copied packages, added docs/type descriptors, and validated go-go-goja plus downstream tests.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-express-goja/2026-05-03--goja-hosting-site/pkg/app/server.go — goja-hosting-site now consumes upstream gojahttp/express/uidsl
- /home/manuel/workspaces/2026-05-08/extract-express-goja/2026-05-07--db-browser/internal/app/server.go — db-browser now consumes upstream gojahttp/express/uidsl
- /home/manuel/workspaces/2026-05-08/extract-express-goja/go-go-goja/modules/express — Runtime-scoped express module registrar
- /home/manuel/workspaces/2026-05-08/extract-express-goja/go-go-goja/modules/uidsl — Rich UI DSL module moved from db-browser
- /home/manuel/workspaces/2026-05-08/extract-express-goja/go-go-goja/pkg/gojahttp — Reusable renderer-neutral HTTP host copied from db-browser internal/web


## 2026-05-08

Ran targeted golangci-lint for pkg/gojahttp, modules/express, and modules/uidsl; fixed errcheck/staticcheck/unused findings in the moved code.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-express-goja/go-go-goja/modules/express — Cleaned lint findings in Express integration tests
- /home/manuel/workspaces/2026-05-08/extract-express-goja/go-go-goja/modules/uidsl — Cleaned lint findings in moved UI DSL tests/components
- /home/manuel/workspaces/2026-05-08/extract-express-goja/go-go-goja/pkg/gojahttp — Cleaned lint findings in gojahttp integration tests


## 2026-05-08

Added shell merge design and detailed task plan using Option A (retire db-browser) and Option C (support both dbguard and simple DB policies).

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-express-goja/go-go-goja/ttmp/2026/05/08/GO-GO-GOJA-EXPRESS-UIDSL-MODULES--upstream-express-and-ui-dsl-modules-into-go-go-goja/design-doc/02-merge-db-browser-and-goja-hosting-site-web-shells.md — Shell merge design
- /home/manuel/workspaces/2026-05-08/extract-express-goja/go-go-goja/ttmp/2026/05/08/GO-GO-GOJA-EXPRESS-UIDSL-MODULES--upstream-express-and-ui-dsl-modules-into-go-go-goja/tasks.md — Detailed task plan

