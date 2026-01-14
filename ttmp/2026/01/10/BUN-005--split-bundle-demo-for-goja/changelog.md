# Changelog

## 2026-01-10

- Initial workspace created


## 2026-01-10

Scaffolded ticket, tasks, and initial split-bundle plan.

### Related Files

- /home/manuel/workspaces/2026-01-10/package-bun-goja-js/go-go-goja/ttmp/2026/01/10/BUN-005--split-bundle-demo-for-goja/analysis/01-split-bundle-demo-plan.md — Split demo plan
- /home/manuel/workspaces/2026-01-10/package-bun-goja-js/go-go-goja/ttmp/2026/01/10/BUN-005--split-bundle-demo-for-goja/index.md — Ticket overview
- /home/manuel/workspaces/2026-01-10/package-bun-goja-js/go-go-goja/ttmp/2026/01/10/BUN-005--split-bundle-demo-for-goja/reference/01-diary.md — Diary step 1
- /home/manuel/workspaces/2026-01-10/package-bun-goja-js/go-go-goja/ttmp/2026/01/10/BUN-005--split-bundle-demo-for-goja/tasks.md — Initial tasks


## 2026-01-10

Implemented split bundle demo with new JS entrypoints, Makefile targets, and Go embed updates.

### Related Files

- /home/manuel/workspaces/2026-01-10/package-bun-goja-js/go-go-goja/cmd/bun-demo/Makefile — Split bundle targets
- /home/manuel/workspaces/2026-01-10/package-bun-goja-js/go-go-goja/cmd/bun-demo/assets-split/app.js — Embedded split output
- /home/manuel/workspaces/2026-01-10/package-bun-goja-js/go-go-goja/cmd/bun-demo/js/src/split/app.ts — Split entrypoint
- /home/manuel/workspaces/2026-01-10/package-bun-goja-js/go-go-goja/cmd/bun-demo/js/src/split/modules/metrics.ts — Split module bundle
- /home/manuel/workspaces/2026-01-10/package-bun-goja-js/go-go-goja/cmd/bun-demo/main.go — Embed assets-split
- /home/manuel/workspaces/2026-01-10/package-bun-goja-js/go-go-goja/ttmp/2026/01/10/BUN-005--split-bundle-demo-for-goja/reference/01-diary.md — Diary step 2


## 2026-01-10

Documented the Model B split-bundle workflow in the bundling playbook.

### Related Files

- /home/manuel/workspaces/2026-01-10/package-bun-goja-js/go-go-goja/pkg/doc/bun-goja-bundling-playbook.md — Model B playbook update
- /home/manuel/workspaces/2026-01-10/package-bun-goja-js/go-go-goja/ttmp/2026/01/10/BUN-005--split-bundle-demo-for-goja/reference/01-diary.md — Diary step 3
- /home/manuel/workspaces/2026-01-10/package-bun-goja-js/go-go-goja/ttmp/2026/01/10/BUN-005--split-bundle-demo-for-goja/tasks.md — Checked playbook task


## 2026-01-14

Ticket closed


## 2026-01-14

Replace bun-driven Makefile steps with a Dagger pipeline and update the plan/diary.

### Related Files

- /home/manuel/workspaces/2026-01-10/package-bun-goja-js/go-go-goja/cmd/bun-demo/Makefile — Makefile targets now invoke Dagger
- /home/manuel/workspaces/2026-01-10/package-bun-goja-js/go-go-goja/cmd/bun-demo/dagger/main.go — Dagger pipeline implementation
- /home/manuel/workspaces/2026-01-10/package-bun-goja-js/go-go-goja/cmd/bun-demo/js/package.json — Node-based render script
- /home/manuel/workspaces/2026-01-10/package-bun-goja-js/go-go-goja/go.mod — Dagger SDK dependency
- /home/manuel/workspaces/2026-01-10/package-bun-goja-js/go-go-goja/ttmp/2026/01/10/BUN-005--split-bundle-demo-for-goja/analysis/01-split-bundle-demo-plan.md — Dagger migration plan
- /home/manuel/workspaces/2026-01-10/package-bun-goja-js/go-go-goja/ttmp/2026/01/10/BUN-005--split-bundle-demo-for-goja/reference/01-diary.md — Diary steps 4-5

