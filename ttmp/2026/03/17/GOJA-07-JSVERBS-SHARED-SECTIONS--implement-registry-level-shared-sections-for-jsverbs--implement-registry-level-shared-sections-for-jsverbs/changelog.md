# Changelog

## 2026-03-17

- Initial workspace created
- Created a new design ticket for registry-level shared sections in jsverbs.
- Mapped the current file-local section model and wrote a detailed design and implementation guide for the shared-sections feature.
- Validated the ticket with `docmgr doctor` and uploaded the final bundle to reMarkable at `/ai/2026/03/17/GOJA-07-JSVERBS-SHARED-SECTIONS`.

## 2026-03-17

Completed the shared-sections design ticket with an intern-oriented implementation guide, file-by-file plan, and rollout guidance.

### Related Files

- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/ttmp/2026/03/17/GOJA-07-JSVERBS-SHARED-SECTIONS--implement-registry-level-shared-sections-for-jsverbs--implement-registry-level-shared-sections-for-jsverbs/design-doc/01-jsverbs-shared-sections-design-and-implementation-guide.md — Primary design deliverable
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/ttmp/2026/03/17/GOJA-07-JSVERBS-SHARED-SECTIONS--implement-registry-level-shared-sections-for-jsverbs--implement-registry-level-shared-sections-for-jsverbs/reference/01-diary.md — Chronological design diary

## 2026-03-17

Implemented registry-level shared-section storage and resolution helpers (commit 5bf8c28d705230174af7dee39fa9e8e523aa9d70).

### Related Files

- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/jsverbs_test.go — Added focused tests for duplicate registration and local-over-shared precedence
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/model.go — Added shared-section registry fields and helper APIs
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/scan.go — Initialized shared-section storage for all scanned registries


## 2026-03-17

Resolved section lookup through the registry across binding, command generation, and runtime invocation; added integration coverage (commit a7c2897e3e6a3a865c788723fce2104220ae8dba).

### Related Files

- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/binding.go — Made binding-plan validation registry-aware and preserved local-first ordering
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/command.go — Resolved section specs through the registry during Glazed schema generation
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/jsverbs_test.go — Added end-to-end coverage for shared sections and override behavior
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/jsverbs/runtime.go — Used the registry-aware binding plan during invocation


## 2026-03-17

Updated jsverbs help pages for file-local versus registry-level shared sections, validated the ticket, and uploaded the refreshed bundle to reMarkable.

### Related Files

- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/doc/08-jsverbs-example-overview.md — Clarified file-local sections versus registry-level shared sections in the overview
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/doc/09-jsverbs-example-fixture-format.md — Documented the shared-section registration API and local-first precedence
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/doc/10-jsverbs-example-developer-guide.md — Explained registry-level shared sections in the intern guide
- /home/manuel/workspaces/2026-03-17/add-opinionated-apis/go-go-goja/pkg/doc/11-jsverbs-example-reference.md — Added exact scope and precedence rules for section resolution


## 2026-03-17

Implemented registry-level shared sections, updated docs, validated the ticket, and uploaded the refreshed bundle to reMarkable.

