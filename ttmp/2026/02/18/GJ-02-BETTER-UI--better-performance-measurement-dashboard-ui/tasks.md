# Tasks

## TODO

- [x] Add tasks here

- [x] Add server-side run state tracking (per-phase runState with output buffer, elapsed time, task progress)
- [x] Add SSE endpoint GET /api/run-stream/{phase} that streams HTML fragments as benchmark output arrives
- [x] Update POST /api/run/{phase} to start background run and return SSE-connected fragment instead of blocking
- [x] Update index template to load htmx-sse extension and add CSS for streaming output display
- [x] Add progress display showing: current task name, elapsed time, output lines, and task N/M counter
- [ ] Guard against concurrent runs (only one run per phase at a time, show already-running state)
