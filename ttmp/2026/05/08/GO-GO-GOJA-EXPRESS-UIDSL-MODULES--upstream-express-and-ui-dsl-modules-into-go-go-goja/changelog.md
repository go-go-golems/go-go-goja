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


## 2026-05-08

Committed baseline extraction and downstream migrations: go-go-goja 229fd920786ae83dc96bac1732bf80eda4c68307, goja-hosting-site dda6fa41cee1048b7e54087f535eed99432c1cbc, db-browser 4e3009f8ee68e119d31aa08f451644ace896fbee.

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-express-goja/go-go-goja/ttmp/2026/05/08/GO-GO-GOJA-EXPRESS-UIDSL-MODULES--upstream-express-and-ui-dsl-modules-into-go-go-goja/log/01-implementation-diary.md — Baseline commit hashes and setup diary


## 2026-05-08

T12: extracted db-browser verb repository discovery into go-go-goja/pkg/jsverbrepos with neutral GOJA_VERB_REPOSITORIES and .goja-verbs.yml names (commit d4aec1568fa1278d2a612780f2c68de27c46a7db).

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-express-goja/go-go-goja/pkg/jsverbrepos — Reusable jsverbs repository discovery package
- /home/manuel/workspaces/2026-05-08/extract-express-goja/go-go-goja/ttmp/2026/05/08/GO-GO-GOJA-EXPRESS-UIDSL-MODULES--upstream-express-and-ui-dsl-modules-into-go-go-goja/log/01-implementation-diary.md — T12 implementation diary


## 2026-05-08

T13: extracted db-browser jsverbs CLI/runtime shell into go-go-goja/pkg/jsverbscli, wired it to pkg/jsverbrepos, and replaced deprecated module registration (commit d84a38f177a6e5e9cf375b14d1d7fb1f90fc4ae9).

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-express-goja/go-go-goja/pkg/jsverbscli — Reusable jsverbs CLI and runtime invocation package
- /home/manuel/workspaces/2026-05-08/extract-express-goja/go-go-goja/ttmp/2026/05/08/GO-GO-GOJA-EXPRESS-UIDSL-MODULES--upstream-express-and-ui-dsl-modules-into-go-go-goja/log/01-implementation-diary.md — T13 implementation diary


## 2026-05-08

T14: added goja-site verbs by wiring jsverbscli.NewLazyCommand into goja-site and validating built-in hello/yaml/renderSampleTable/tables verbs (goja-hosting-site commit d62fa16c71d2f6567bca53915888910247667d3a).

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-express-goja/2026-05-03--goja-hosting-site/cmd/goja-site/main.go — goja-site verbs command integration
- /home/manuel/workspaces/2026-05-08/extract-express-goja/go-go-goja/ttmp/2026/05/08/GO-GO-GOJA-EXPRESS-UIDSL-MODULES--upstream-express-and-ui-dsl-modules-into-go-go-goja/log/01-implementation-diary.md — T14 implementation diary


## 2026-05-08

T15: generalized goja-site script loading to repeatable script directories, updated multi-site config to scripts lists, and added script discovery tests (goja-hosting-site commit 67eff77dfa48f5fe4d521fef9cfd5aae99414051).

### Related Files

- /home/manuel/workspaces/2026-05-08/extract-express-goja/2026-05-03--goja-hosting-site/cmd/goja-site/serve.go — Repeatable --scripts flag
- /home/manuel/workspaces/2026-05-08/extract-express-goja/2026-05-03--goja-hosting-site/pkg/app/scripts.go — Multi-directory script discovery
- /home/manuel/workspaces/2026-05-08/extract-express-goja/go-go-goja/ttmp/2026/05/08/GO-GO-GOJA-EXPRESS-UIDSL-MODULES--upstream-express-and-ui-dsl-modules-into-go-go-goja/log/01-implementation-diary.md — T15 implementation diary

