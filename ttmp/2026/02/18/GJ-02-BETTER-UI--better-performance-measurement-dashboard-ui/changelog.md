# Changelog

## 2026-02-18

- Initial workspace created


## 2026-02-18

Created UI analysis document with current state critique, ASCII screenshots of before/after layouts, and complete redesign proposal with tabbed phases, benchmark cards, proportional bars, smart unit formatting, and accordion navigation

### Related Files

- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/ttmp/2026/02/18/GJ-02-BETTER-UI--better-performance-measurement-dashboard-ui/analysis/01-ui-analysis-and-redesign-proposal.md — Analysis document


## 2026-02-18

Implemented redesigned UI: new serve_format.go with fmtNs/fmtBytes/shortBench/prepareTasks, rewrote indexTemplate and fragmentTemplate in serve_command.go with tabbed phase navigation, accordion tasks, benchmark cards with proportional bars, and loading spinner

### Related Files

- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/cmd/goja-perf/serve_command.go — Rewritten templates and updated rendering
- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/cmd/goja-perf/serve_format.go — New formatting utilities and data preparation
- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/cmd/goja-perf/serve_format_test.go — Unit tests for formatting


## 2026-02-18

Implemented realtime streaming feedback during benchmark runs: background run tracking per phase, 1-second polling progress display with elapsed time, benchmark result count, live output tail, and automatic transition to final report on completion. Concurrent runs guarded.

### Related Files

- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/cmd/goja-perf/serve_streaming.go — Background run state
- /home/manuel/workspaces/2026-02-18/goja-performance/go-go-goja/cmd/goja-perf/serve_streaming_test.go — Tests for elapsed formatting

