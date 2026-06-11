# Changelog

## 2026-06-10

- Initial workspace created


## 2026-06-10

Created TypeScript support research ticket, imported /tmp/goja-ts.md, wrote intern-oriented design guide and diary, and updated tasks/index.

### Related Files

- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/ttmp/2026/06/10/XGOJA-TS-001--typescript-support-for-go-go-goja-xgoja-and-hot-reload/design/01-typescript-support-analysis-and-implementation-guide.md — Primary design deliverable
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/ttmp/2026/06/10/XGOJA-TS-001--typescript-support-for-go-go-goja-xgoja-and-hot-reload/reference/01-investigation-diary.md — Chronological investigation diary


## 2026-06-10

Validated ticket hygiene, fixed imported source frontmatter/prefix, and uploaded the documentation bundle to reMarkable at /ai/2026/06/10/XGOJA-TS-001.

### Related Files

- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/ttmp/2026/06/10/XGOJA-TS-001--typescript-support-for-go-go-goja-xgoja-and-hot-reload/reference/01-investigation-diary.md — Records validation failure
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/ttmp/2026/06/10/XGOJA-TS-001--typescript-support-for-go-go-goja-xgoja-and-hot-reload/sources/local/01-goja-typescript-esbuild-note.md — Imported source note normalized for docmgr validation


## 2026-06-10

Step 5: added pkg/tsscript esbuild compiler facade and tests (commit 9f8c8be).

### Related Files

- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/pkg/tsscript/compiler.go — Compiler facade
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/pkg/tsscript/compiler_test.go — Compiler facade tests


## 2026-06-10

Step 6: added TypeScript configuration schema/defaulting/validation and provider descriptors (commit d2b9d58).

### Related Files

- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/cmd/xgoja/internal/buildspec/build_spec.go — Build-time TypeScript schema
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/pkg/xgoja/app/runtime_spec.go — Runtime TypeScript schema


## 2026-06-10

Step 7: wired TypeScript scan/runtime transforms into jsverbs and xgoja app, with a bundled TypeScript jsverb invocation test (commit 5fc1baa).

### Related Files

- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/pkg/jsverbs/runtime.go — Runtime transform and overlay handling
- /home/manuel/workspaces/2026-06-10/goja-xgoja-ts-support/go-go-goja/pkg/xgoja/app/typescript.go — TypeScript adapter

