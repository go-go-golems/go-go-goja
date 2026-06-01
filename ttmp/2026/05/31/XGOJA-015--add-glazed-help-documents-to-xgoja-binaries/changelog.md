# Changelog

## 2026-05-31

- Initial workspace created


## 2026-05-31

Created research/design package for bundling Glazed help documents into generated xgoja binaries, including provider HelpSource API sketch, help.sources buildspec design, generation/app wiring plan, test strategy, and diary.

### Related Files

- /home/manuel/workspaces/2026-05-31/xgoja-docs/go-go-goja/ttmp/2026/05/31/XGOJA-015--add-glazed-help-documents-to-xgoja-binaries/design-doc/01-glazed-help-documents-for-xgoja-binaries-implementation-guide.md — Primary XGOJA-015 design deliverable
- /home/manuel/workspaces/2026-05-31/xgoja-docs/go-go-goja/ttmp/2026/05/31/XGOJA-015--add-glazed-help-documents-to-xgoja-binaries/reference/01-investigation-diary.md — Chronological investigation diary


## 2026-05-31

Recorded validation and reMarkable delivery evidence in the investigation diary.

### Related Files

- /home/manuel/workspaces/2026-05-31/xgoja-docs/go-go-goja/ttmp/2026/05/31/XGOJA-015--add-glazed-help-documents-to-xgoja-binaries/reference/01-investigation-diary.md — Diary now includes docmgr doctor and reMarkable upload evidence


## 2026-05-31

Implemented Glazed help document support for xgoja binaries across providerapi, buildspec validation, generator embedding, generated root loading, Loupedeck provider docs registration, and xgoja help docs. go-go-goja commits: 13d30d0, e006c2c, 1dec2ba, 4c646d9, 2c6b112; loupedeck commit: b5825fa.

### Related Files

- /home/manuel/workspaces/2026-05-31/xgoja-docs/go-go-goja/pkg/xgoja/app/framework.go — Loads configured help sources
- /home/manuel/workspaces/2026-05-31/xgoja-docs/go-go-goja/pkg/xgoja/providerapi/help.go — New HelpSource API
- /home/manuel/workspaces/2026-05-31/xgoja-docs/loupedeck/runtime/js/provider/provider.go — Registers loupedeck.runtime-api help source


## 2026-05-31

Finalized implementation diary/tasks after docmgr validation and reMarkable upload to /ai/2026/05/31/XGOJA-015/XGOJA-015 Glazed help docs implementation final.

### Related Files

- /home/manuel/workspaces/2026-05-31/xgoja-docs/go-go-goja/ttmp/2026/05/31/XGOJA-015--add-glazed-help-documents-to-xgoja-binaries/reference/01-investigation-diary.md — Final validation and upload evidence
- /home/manuel/workspaces/2026-05-31/xgoja-docs/go-go-goja/ttmp/2026/05/31/XGOJA-015--add-glazed-help-documents-to-xgoja-binaries/tasks.md — Implementation checklist status


## 2026-05-31

Added examples/xgoja/09-provider-shipped-help-docs and manually smoke-tested a generated binary that loads Loupedeck provider-shipped help docs via help loupedeck-js-api-reference and help loupedeck-js-first-live-script.

### Related Files

- /home/manuel/workspaces/2026-05-31/xgoja-docs/go-go-goja/examples/xgoja/09-provider-shipped-help-docs/Makefile — Manual smoke test target
- /home/manuel/workspaces/2026-05-31/xgoja-docs/go-go-goja/examples/xgoja/09-provider-shipped-help-docs/xgoja.yaml — Reference buildspec for provider-shipped help docs

