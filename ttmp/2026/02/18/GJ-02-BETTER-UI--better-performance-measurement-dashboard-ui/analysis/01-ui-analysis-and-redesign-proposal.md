---
title: "UI Analysis and Redesign Proposal"
doc_type: analysis
status: active
intent: long-term
ticket: GJ-02-BETTER-UI
topics:
  - performance
  - frontend
  - ui
created: 2026-02-18
updated: 2026-02-18
---

# Goja Performance Dashboard — UI Analysis and Redesign

## 1. Current State Analysis

### 1.1 Architecture

The current dashboard is a single-file Go HTTP server (`serve_command.go`) serving:
- **`indexTemplate`**: A Bootstrap 5 + HTMX page with two side-by-side cards
- **`fragmentTemplate`**: An HTMX fragment returned from `/api/report/{phase}` and `/api/run/{phase}`

### 1.2 Current UI Problems

**Problem 1: Flat table dumps everything into one undifferentiated wall**

The `fragmentTemplate` renders every benchmark summary as a flat `<table>` with
columns `Benchmark | What it does | Runs | Metric | Avg | Min | Max`. Each metric
(ns/op, B/op, allocs/op) gets its own row, so a single benchmark expands into 3
rows. With 7 benchmarks × 3 metrics = 21 rows in one dense table per task, and
3 tasks visible at once, the user sees 63+ rows of numbers with no visual
hierarchy.

**Problem 2: Benchmark names are long and cryptic**

Names like `BenchmarkRuntimeSpawn/EngineNew_WithCallLog-8` occupy a full table
cell, pushing the data columns off-screen on smaller viewports.

**Problem 3: No comparative context**

Raw numbers like `ns/op: 232808` mean nothing without context. The user cannot
see which benchmarks are fast vs slow relative to each other, nor which represent
meaningful overhead.

**Problem 4: Repeated "What it does" column wastes space**

Every row in a benchmark group repeats the identical description text ("Compare
runtime creation costs...") because metrics are flattened into separate rows.

**Problem 5: Phase cards are side-by-side but scroll independently**

On a typical screen, each card is 50% width, making the tables extremely narrow.
The `table-responsive` wrapper creates horizontal scrollbars inside each card.

**Problem 6: No visual feedback during long-running benchmarks**

Clicking "Run" blocks with no progress indicator. Phase runs take 10-30 seconds.

### 1.3 Current Layout (ASCII Screenshot)

```
┌──────────────────────────────────────────────────────────────────────────────────┐
│  Goja Performance Dashboard                                                      │
│  Run phase benchmarks and inspect YAML-backed reports.                            │
├──────────────────────────────┬───────────────────────────────────────────────────┤
│  Phase 1          [Run][Ref] │  Phase 2                            [Run][Ref]    │
│  report: ...phase1-run-r...  │  report: ...phase2-run-r...                      │
│  updated: 2026-02-18T14:... │  updated: 2026-02-18T14:...                      │
│  ┌─────────────────────────┐ │  ┌──────────────────────────────────────────────┐ │
│  │ Total tasks      3      │ │  │ Total tasks      3                          │ │
│  │ Successful       3      │ │  │ Successful       3                          │ │
│  │ Failed           0      │ │  │ Failed           0                          │ │
│  │ Duration (ms)  30242    │ │  │ Duration (ms)  27191                        │ │
│  └─────────────────────────┘ │  └──────────────────────────────────────────────┘ │
│                              │                                                   │
│  ┌─ Runtime Lifecycle ────┐  │  ┌─ Value Conversion ──────────────────────────┐  │
│  │ What this task measures │  │  │ What this task measures                     │  │
│  │ ┌──────┬────────────┐  │  │  │ ┌──────┬──────────────────────────────────┐ │  │
│  │ │Bench │ Description│  │  │  │ │Bench │ Description                      │ │  │
│  │ ├──────┼────────────┤  │  │  │ ├──────┼──────────────────────────────────┤ │  │
│  │ │BmkRu │ Compare... │  │  │  │ │BmkVa │ Measure conversion overhead...  │ │  │
│  │ │BmkRu │ Measure... │  │  │  │ └──────┴──────────────────────────────────┘ │  │
│  │ │BmkRu │ Measure... │  │  │  │                                             │  │
│  │ └──────┴────────────┘  │  │  │ Structured results                          │  │
│  │                        │  │  │ ┌──────┬─────┬────┬──────┬─────┬────┬────┐  │  │
│  │ Structured results     │  │  │ │Bench │What │Runs│Metric│ Avg │Min │Max │  │  │
│  │ ┌──────┬──┬──┬──┬──┬──│  │  │ ├──────┼─────┼────┼──────┼─────┼────┼────┤  │  │
│  │ │Bench │Wh│Rn│Me│Av│Mn│  │  │ │BmkVa │Meas │ 3  │ns/op │4.05 │3.5 │5.0│  │  │
│  │ ├──────┼──┼──┼──┼──┼──│  │  │ │BmkVa │Meas │ 3  │B/op  │ 0   │ 0  │ 0 │  │  │
│  │ │BmkRu │Co│ 3│ns│98│89│  │  │ │BmkVa │Meas │ 3  │al/op │ 0   │ 0  │ 0 │  │  │
│  │ │BmkRu │Co│ 3│B/│17│17│  │  │ │ ... 30+ more rows ...                  │  │  │
│  │ │BmkRu │Co│ 3│al│ 8│ 8│  │  │ └──────┴─────┴────┴──────┴─────┴────┴────┘  │  │
│  │ │ ... 18+ more rows .. │  │  │                                             │  │
│  │ └──────┴──┴──┴──┴──┴──│  │  └─────────────────────────────────────────────┘  │
│  └────────────────────────┘  │                                                   │
│                              │  ┌─ Payload Sweep ─────────────────────────────┐  │
│  ┌─ Loading and Require ──┐  │  │ ... (same pattern, very long) ...           │  │
│  │ ... (same dense wall)  │  │  └─────────────────────────────────────────────┘  │
│  └────────────────────────┘  │                                                   │
│                              │  ┌─ GC Sensitivity ────────────────────────────┐  │
│  ┌─ Go/JS Boundary Calls ┐  │  │ ... (same pattern) ...                      │  │
│  │ ... (same dense wall)  │  │  └─────────────────────────────────────────────┘  │
│  └────────────────────────┘  │                                                   │
└──────────────────────────────┴───────────────────────────────────────────────────┘
```

**Key issues visible in the ASCII:**
- Tables are truncated/scrolling horizontally in narrow 50% columns
- "What it does" column repeats the same text on every metric row
- No visual distinction between fast and slow benchmarks
- No grouping — benchmark sub-cases are interleaved with metrics

---

## 2. Redesign Proposal

### 2.1 Design Principles

1. **Task-first navigation** — Phase tabs at top, tasks as collapsible accordion sections
2. **Benchmark cards** — Each benchmark sub-case gets its own compact card with all 3 metrics side-by-side (not 3 separate rows)
3. **Smart formatting** — Format nanoseconds into human-readable units (ns, µs, ms); format bytes with K/M suffixes
4. **Relative bars** — Show proportional bar charts within a task group to give visual comparison
5. **Full-width layout** — Use the full viewport width instead of splitting into cramped 50% columns
6. **Loading state** — Add a spinner/indicator during benchmark runs
7. **Color coding** — Green/yellow/red indicators for relative performance within a benchmark family

### 2.2 Proposed Layout (ASCII Screenshot)

```
┌──────────────────────────────────────────────────────────────────────────────────┐
│  ⚡ Goja Performance Dashboard                                                   │
│                                                                                  │
│  ┌─────────────┐ ┌─────────────┐                                                │
│  │ ● Phase 1   │ │   Phase 2   │                            ⟳ Refresh  ▶ Run    │
│  └─────────────┘ └─────────────┘                                                │
│                                                                                  │
│  Run: 2026-02-18 14:28 UTC  ·  3/3 tasks passed  ·  30.2s total                │
│                                                                                  │
│  ┌──────────────────────────────────────────────────────────────────────────────┐│
│  │ ▼ Runtime Lifecycle                                              8.2s  ✓    ││
│  │   Measure VM spawn and spawn+execute/reuse behavior.                        ││
│  │                                                                              ││
│  │   ┌ GojaNew ──────────────────────────────────────────────────────────────┐  ││
│  │   │  981 ns/op  ██░░░░░░░░░░░░░░░░░░░   1.8 KB/op    8 allocs/op        │  ││
│  │   │  range: 900 ns – 1.1 µs                                              │  ││
│  │   └──────────────────────────────────────────────────────────────────────┘  ││
│  │                                                                              ││
│  │   ┌ EngineNew_NoCallLog ──────────────────────────────────────────────────┐  ││
│  │   │   20 µs/op  █████████░░░░░░░░░░░░  11.9 KB/op  140 allocs/op        │  ││
│  │   │  range: 18.8 µs – 21.4 µs                  21× slower than GojaNew  │  ││
│  │   └──────────────────────────────────────────────────────────────────────┘  ││
│  │                                                                              ││
│  │   ┌ EngineNew_WithCallLog ────────────────────────────────────────────────┐  ││
│  │   │  233 µs/op  ████████████████████░  17.7 KB/op  321 allocs/op        │  ││
│  │   │  range: 211 µs – 253 µs                    237× slower than GojaNew  │  ││
│  │   └──────────────────────────────────────────────────────────────────────┘  ││
│  │                                                                              ││
│  │   ┌ RunString_FreshRuntime ───────────────────────────────────────────────┐  ││
│  │   │   38 µs/op  ██████████░░░░░░░░░░░  14.3 KB/op  189 allocs/op        │  ││
│  │   │  range: 35.6 µs – 41.0 µs                                            │  ││
│  │   └──────────────────────────────────────────────────────────────────────┘  ││
│  │                                                                              ││
│  │   ┌ RunProgram_FreshRuntime ──────────────────────────────────────────────┐  ││
│  │   │   31 µs/op  ████████░░░░░░░░░░░░░  12.2 KB/op  146 allocs/op        │  ││
│  │   │  range: 29.5 µs – 32.4 µs                   18% faster than String  │  ││
│  │   └──────────────────────────────────────────────────────────────────────┘  ││
│  │                                                                              ││
│  │   ┌ RunString_ReusedRuntime ──────────────────────────────────────────────┐  ││
│  │   │  6.1 µs/op  ███░░░░░░░░░░░░░░░░░░   3.3 KB/op   41 allocs/op        │  ││
│  │   │  range: 5.8 µs – 6.3 µs                                              │  ││
│  │   └──────────────────────────────────────────────────────────────────────┘  ││
│  │                                                                              ││
│  │   ┌ RunProgram_ReusedRuntime ─────────────────────────────────────────────┐  ││
│  │   │  155 ns/op  █░░░░░░░░░░░░░░░░░░░░     32 B/op    1 allocs/op        │  ││
│  │   │  range: 118 ns – 201 ns                          ⚡ fastest in group  │  ││
│  │   └──────────────────────────────────────────────────────────────────────┘  ││
│  └──────────────────────────────────────────────────────────────────────────────┘│
│                                                                                  │
│  ┌──────────────────────────────────────────────────────────────────────────────┐│
│  │ ▶ Loading and Require                                           14.3s  ✓    ││
│  │   (collapsed — click to expand)                                              ││
│  └──────────────────────────────────────────────────────────────────────────────┘│
│                                                                                  │
│  ┌──────────────────────────────────────────────────────────────────────────────┐│
│  │ ▶ Go/JS Boundary Calls                                          7.8s  ✓    ││
│  │   (collapsed — click to expand)                                              ││
│  └──────────────────────────────────────────────────────────────────────────────┘│
└──────────────────────────────────────────────────────────────────────────────────┘
```

### 2.3 Comparison Table View (Alternative Compact Mode)

For users who want a denser view, a toggleable compact table mode per task:

```
┌──────────────────────────────────────────────────────────────────────────────────┐
│ ▼ Runtime Lifecycle                         [Cards ◉] [Table ○]    8.2s  ✓     │
│                                                                                  │
│  Benchmark                     ns/op      B/op   allocs   vs baseline            │
│  ─────────────────────────────────────────────────────────────────────────────── │
│  GojaNew                          981    1,760        8   baseline               │
│  EngineNew_NoCallLog           20,450   11,928      140   ████████░░  21×        │
│  EngineNew_WithCallLog        232,808   17,689      321   ██████████ 237×        │
│  RunString_FreshRuntime        38,227   14,320      189   ████████░░  39×        │
│  RunProgram_FreshRuntime       31,149   12,216      146   ███████░░░  32×        │
│  RunString_ReusedRuntime        6,067    3,328       41   ██░░░░░░░░   6×        │
│  RunProgram_ReusedRuntime         155       32        1   ░░░░░░░░░░  0.2×       │
└──────────────────────────────────────────────────────────────────────────────────┘
```

### 2.4 Error and Running States

```
Running state:                            Error state:
┌──────────────────────────────┐          ┌──────────────────────────────┐
│ ▼ Runtime Lifecycle          │          │ ▼ Value Conversion           │
│   ◐ Running benchmarks...   │          │   ✗ Run failed (exit 1)      │
│   ░░░░░░░░░░░░░░░░░░░░░░░  │          │                              │
│   Started 12s ago            │          │   ▸ Show command output      │
└──────────────────────────────┘          │     go test ./perf/goja ...  │
                                          │     --- FAIL: ...            │
                                          └──────────────────────────────┘
```

---

## 3. Implementation Plan

### 3.1 Template Functions Needed

| Function | Purpose |
|----------|---------|
| `fmtNs` | Format nanoseconds: `<1000` → `"981 ns"`, `<1e6` → `"20.5 µs"`, `<1e9` → `"232 ms"` |
| `fmtBytes` | Format bytes: `<1024` → `"32 B"`, `<1M` → `"11.9 KB"`, else `"1.2 MB"` |
| `fmtCount` | Format with commas: `232808` → `"232,808"` |
| `shortBench` | Strip `Benchmark` prefix and `-N` suffix: `BenchmarkRuntimeSpawn/GojaNew-8` → `GojaNew` |
| `pctBar` | Generate CSS bar width percentage relative to max in group |
| `relLabel` | "21× slower" or "fastest in group" relative label |

### 3.2 Changes to `serve_command.go`

1. **Replace `indexTemplate`** with new full-width, tabbed layout using Bootstrap 5 tabs
2. **Replace `fragmentTemplate`** with accordion-based task cards
3. **Add template functions** for smart formatting
4. **Add CSS** for benchmark cards, proportional bars, and status indicators
5. **Add loading indicator** via HTMX `hx-indicator`
6. **Pre-process data** in `renderFragment` to compute relative metrics and bar widths

### 3.3 Data Pre-processing (Go side)

Add a `prepareViewData` function that:
1. Groups summaries by benchmark family (strip sub-case suffix)
2. Finds max ns/op within each task for bar scaling
3. Computes relative multiplier vs the fastest benchmark
4. Formats all values using the template functions above
5. Strips the `Benchmark` prefix and `-N` suffix from names

### 3.4 New Go Types for View

```go
type benchmarkCard struct {
    ShortName    string   // "GojaNew"
    NsFormatted  string   // "981 ns"
    BytesFormatted string // "1.8 KB"
    AllocsFormatted string // "8"
    BarPct       int      // 0-100 for CSS width
    RangeText    string   // "range: 900 ns – 1.1 µs"
    RelativeText string   // "21× slower than GojaNew" or "⚡ fastest"
    IsError      bool
}

type taskView struct {
    ID          string
    Title       string
    Description string
    DurationFormatted string
    Success     bool
    Cards       []benchmarkCard
}
```

### 3.5 File Changes

| File | Change |
|------|--------|
| `cmd/goja-perf/serve_command.go` | Complete rewrite of templates and addition of template functions + data preparation |
| `cmd/goja-perf/serve_format.go` | New file — formatting utilities (`fmtNs`, `fmtBytes`, `shortBench`, etc.) |
| `cmd/goja-perf/serve_format_test.go` | New file — unit tests for formatting |

---

## 4. Key Design Decisions

### 4.1 Full-width with tabs (not side-by-side cards)

**Rationale**: The data is too wide for 50% columns. Phase 1 and Phase 2 are
never compared side-by-side, they are independent measurement runs. Tabs
eliminate the column-cramming problem entirely.

### 4.2 One card per benchmark sub-case (not one row per metric)

**Rationale**: Each benchmark sub-case (e.g., `GojaNew`, `EngineNew_NoCallLog`)
is a coherent unit. Showing ns/op, B/op, and allocs/op on one line gives the
reader all three dimensions at a glance instead of scanning 3 separate rows.

### 4.3 Proportional bars within task groups

**Rationale**: A bar chart instantly answers "which is slowest?" without reading
numbers. The max ns/op in the task group becomes 100%, everything else is
proportional. This is the single biggest readability improvement.

### 4.4 Smart unit formatting

**Rationale**: `232808` is unreadable. `233 µs` is instant. Similarly `17689 B`
→ `17.3 KB`. The human eye parses units much faster than counting digits.

### 4.5 Collapsible accordion tasks

**Rationale**: When all tasks are expanded, the page is still long. Accordion
lets the user focus on one task at a time, and collapsed tasks show just the
headline: name, duration, pass/fail status.

---

## 5. Implementation — Complete Rewritten Templates

Below are the exact templates that will replace the existing ones.

### 5.1 Index Template (complete)

```html
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Goja Perf Dashboard</title>
  <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.3/dist/css/bootstrap.min.css"
        rel="stylesheet">
  <script src="https://unpkg.com/htmx.org@1.9.12"></script>
  <style>
    body { background: #f8f9fa; }
    .bench-card {
      border: 1px solid #dee2e6; border-radius: 8px; padding: 12px 16px;
      margin-bottom: 8px; background: #fff;
      transition: box-shadow .15s;
    }
    .bench-card:hover { box-shadow: 0 2px 8px rgba(0,0,0,.08); }
    .bench-name { font-weight: 600; font-size: 0.95rem; color: #212529; }
    .bench-metrics { display: flex; gap: 24px; align-items: center;
                     margin-top: 4px; flex-wrap: wrap; }
    .bench-metric { font-family: 'SF Mono', 'Cascadia Code', monospace;
                    font-size: 0.9rem; white-space: nowrap; }
    .bench-metric .value { font-weight: 700; color: #0d6efd; }
    .bench-metric .unit { color: #6c757d; font-weight: 400; }
    .bench-bar-wrap { flex: 1; min-width: 120px; max-width: 300px; }
    .bench-bar { height: 8px; border-radius: 4px; background: #e9ecef; }
    .bench-bar-fill { height: 100%; border-radius: 4px;
                      background: linear-gradient(90deg, #0d6efd, #6610f2); }
    .bench-range { font-size: 0.78rem; color: #6c757d; margin-top: 2px; }
    .bench-relative { font-size: 0.78rem; color: #198754; font-weight: 500; }
    .bench-relative.slow { color: #dc3545; }
    .task-header { cursor: pointer; user-select: none; }
    .task-header:hover { background: #f1f3f5; }
    .task-toggle { font-size: 0.75rem; color: #6c757d; }
    .status-badge { font-size: 0.75rem; padding: 2px 8px; border-radius: 4px; }
    .status-ok { background: #d1e7dd; color: #0f5132; }
    .status-fail { background: #f8d7da; color: #842029; }
    .loading-spinner { display: inline-block; width: 16px; height: 16px;
      border: 2px solid #0d6efd; border-right-color: transparent;
      border-radius: 50%; animation: spin .6s linear infinite; }
    @keyframes spin { to { transform: rotate(360deg); } }
    .phase-tab { border: none; background: none; padding: 8px 20px;
      font-weight: 500; color: #6c757d; border-bottom: 3px solid transparent;
      cursor: pointer; font-size: 1rem; }
    .phase-tab.active { color: #0d6efd; border-bottom-color: #0d6efd; }
    .phase-tab:hover { color: #0d6efd; }
    .summary-bar { display: flex; gap: 20px; padding: 8px 0;
      font-size: 0.88rem; color: #495057; flex-wrap: wrap; }
    .summary-item strong { color: #212529; }
  </style>
</head>
<body>
  <div class="container-fluid" style="max-width: 960px; margin: 0 auto;">
    <div class="py-4">
      <h1 class="mb-1" style="font-size: 1.6rem;">⚡ Goja Performance Dashboard</h1>

      <div class="d-flex align-items-center gap-2 mt-3 mb-3 border-bottom">
        <button class="phase-tab active" id="tab-phase1"
                onclick="switchPhase('phase1')">Phase 1</button>
        <button class="phase-tab" id="tab-phase2"
                onclick="switchPhase('phase2')">Phase 2</button>
        <div class="ms-auto d-flex gap-2 pb-2">
          <button class="btn btn-sm btn-outline-secondary" id="btn-refresh"
                  onclick="refreshPhase()">⟳ Refresh</button>
          <button class="btn btn-sm btn-primary" id="btn-run"
                  onclick="runPhase()">▶ Run</button>
        </div>
      </div>

      <div id="report-content"
           hx-get="/api/report/phase1"
           hx-trigger="load"
           hx-swap="innerHTML">
        <div class="text-muted py-4 text-center">Loading...</div>
      </div>
    </div>
  </div>

  <script>
    let currentPhase = 'phase1';
    function switchPhase(phase) {
      currentPhase = phase;
      document.querySelectorAll('.phase-tab').forEach(t => t.classList.remove('active'));
      document.getElementById('tab-' + phase).classList.add('active');
      htmx.ajax('GET', '/api/report/' + phase, '#report-content');
    }
    function refreshPhase() {
      htmx.ajax('GET', '/api/report/' + currentPhase, '#report-content');
    }
    function runPhase() {
      const btn = document.getElementById('btn-run');
      btn.disabled = true;
      btn.innerHTML = '<span class="loading-spinner"></span> Running...';
      htmx.ajax('POST', '/api/run/' + currentPhase, {target: '#report-content'}).then(() => {
        btn.disabled = false;
        btn.innerHTML = '▶ Run';
      });
    }
    function toggleTask(id) {
      const el = document.getElementById('task-body-' + id);
      const arrow = document.getElementById('task-arrow-' + id);
      if (el.style.display === 'none') {
        el.style.display = 'block';
        arrow.textContent = '▼';
      } else {
        el.style.display = 'none';
        arrow.textContent = '▶';
      }
    }
  </script>
</body>
</html>
```

### 5.2 Fragment Template (complete)

```html
<div>
  {{if .HasError}}
    <div class="alert alert-warning">
      <strong>{{.Error}}</strong>
      {{if .RunOutput}}
      <details class="mt-2"><summary>Command Output</summary>
        <pre class="mt-2 mb-0 small" style="max-height:300px;overflow:auto;">{{.RunOutput}}</pre>
      </details>
      {{end}}
    </div>
  {{end}}

  {{if .HasReport}}
    <div class="summary-bar">
      <span>📅 <strong>{{.UpdatedAt}}</strong></span>
      <span>✓ <strong>{{.Summary.SuccessfulTasks}}</strong>/{{.Summary.TotalTasks}} passed</span>
      {{if gt .Summary.FailedTasks 0}}
        <span style="color:#dc3545;">✗ <strong>{{.Summary.FailedTasks}}</strong> failed</span>
      {{end}}
      <span>⏱ <strong>{{fmtDuration .Summary.TotalDurationMS}}</strong></span>
    </div>

    {{range $i, $task := .Tasks}}
      <div class="border rounded mb-3 overflow-hidden">
        <div class="task-header d-flex align-items-center px-3 py-2"
             onclick="toggleTask('{{$task.ID}}')">
          <span id="task-arrow-{{$task.ID}}" class="task-toggle me-2">{{if eq $i 0}}▼{{else}}▶{{end}}</span>
          <strong class="me-2">{{$task.Title}}</strong>
          <span class="text-muted small me-auto">{{$task.Description}}</span>
          <span class="text-muted small me-2">{{fmtDuration $task.DurationMS}}</span>
          <span class="status-badge {{if $task.Success}}status-ok{{else}}status-fail{{end}}">
            {{if $task.Success}}✓{{else}}✗{{end}}
          </span>
        </div>
        <div id="task-body-{{$task.ID}}" class="px-3 pb-3"
             style="{{if ne $i 0}}display:none;{{end}}">
          {{if $task.Cards}}
            {{range $task.Cards}}
              <div class="bench-card">
                <div class="d-flex align-items-center justify-content-between">
                  <span class="bench-name">{{.ShortName}}</span>
                  {{if .RelativeText}}
                    <span class="bench-relative {{if .IsSlow}}slow{{end}}">{{.RelativeText}}</span>
                  {{end}}
                </div>
                <div class="bench-metrics">
                  <span class="bench-metric"><span class="value">{{.NsFormatted}}</span></span>
                  <span class="bench-metric"><span class="value">{{.BytesFormatted}}</span><span class="unit">/op</span></span>
                  <span class="bench-metric"><span class="value">{{.AllocsFormatted}}</span><span class="unit"> allocs</span></span>
                  <div class="bench-bar-wrap">
                    <div class="bench-bar"><div class="bench-bar-fill" style="width:{{.BarPct}}%"></div></div>
                  </div>
                </div>
                {{if .RangeText}}
                  <div class="bench-range">{{.RangeText}}</div>
                {{end}}
              </div>
            {{end}}
          {{else}}
            <div class="text-muted small py-2">No benchmark results parsed.</div>
          {{end}}
        </div>
      </div>
    {{end}}
  {{else if not .HasError}}
    <div class="text-muted text-center py-4">No report yet. Click ▶ Run to start benchmarks.</div>
  {{end}}
</div>
```

---

## 6. Formatting Utilities (Go Code)

```go
// serve_format.go

func fmtNs(ns float64) string {
    switch {
    case ns < 1000:
        return fmt.Sprintf("%.0f ns", ns)
    case ns < 1e6:
        return fmt.Sprintf("%.1f µs", ns/1e3)
    case ns < 1e9:
        return fmt.Sprintf("%.1f ms", ns/1e6)
    default:
        return fmt.Sprintf("%.2f s", ns/1e9)
    }
}

func fmtBytes(b float64) string {
    switch {
    case b < 1024:
        return fmt.Sprintf("%.0f B", b)
    case b < 1024*1024:
        return fmt.Sprintf("%.1f KB", b/1024)
    default:
        return fmt.Sprintf("%.1f MB", b/(1024*1024))
    }
}

func shortBench(name string) string {
    // "BenchmarkRuntimeSpawn/GojaNew-8" → "GojaNew"
    if i := strings.LastIndex(name, "/"); i >= 0 {
        name = name[i+1:]
    }
    if i := strings.LastIndex(name, "-"); i >= 0 {
        if _, err := strconv.Atoi(name[i+1:]); err == nil {
            name = name[:i]
        }
    }
    return name
}

func fmtDuration(ms int64) string {
    if ms < 1000 {
        return fmt.Sprintf("%d ms", ms)
    }
    return fmt.Sprintf("%.1f s", float64(ms)/1000)
}
```

---

## 7. Before / After Comparison

| Aspect | Before | After |
|--------|--------|-------|
| Layout | Two cramped 50% columns | Full-width, tabbed phases |
| Benchmark display | 3 rows per benchmark (one per metric) | 1 card per benchmark (all metrics inline) |
| Number formatting | Raw (`232808`, `17689`) | Human (`233 µs`, `17.3 KB`) |
| Visual comparison | None — scan raw numbers | Proportional bars + relative labels |
| Task navigation | All expanded, scroll forever | Accordion — first expanded, rest collapsed |
| Run feedback | No indicator | Spinner + "Running..." button state |
| Benchmark names | Full qualified (`BenchmarkRuntimeSpawn/GojaNew-8`) | Short (`GojaNew`) |
| Space efficiency | ~21 table rows per task × 7 columns | ~7 compact cards per task |

---

## 8. Risk Assessment

| Risk | Mitigation |
|------|-----------|
| Template complexity increases | Extract templates to separate `.go` constants, add tests |
| Bar scaling with outliers (e.g., 0.4 ns GoDirect vs 233 µs WithCallLog) | Use log scale or cap minimum bar at 1% |
| Browser compatibility of CSS gradient bars | Bootstrap 5 already targets modern browsers |
| Loss of raw data visibility | Add expandable "Raw samples" section per task (collapsed by default) |
