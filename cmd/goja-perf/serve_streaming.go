package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// runState tracks the state of a background benchmark run for one phase.
type runState struct {
	mu        sync.Mutex
	running   bool
	startedAt time.Time
	lines     []string // accumulated output lines
	done      bool
	exitErr   error
}

// snapshot returns a copy of the current state (safe to read without holding mu).
type runSnapshot struct {
	Running   bool
	StartedAt time.Time
	Elapsed   time.Duration
	Lines     []string
	Done      bool
	ExitErr   error
}

func (rs *runState) snapshot() runSnapshot {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	snap := runSnapshot{
		Running:   rs.running,
		StartedAt: rs.startedAt,
		Done:      rs.done,
		ExitErr:   rs.exitErr,
	}
	if rs.running || rs.done {
		snap.Elapsed = time.Since(rs.startedAt)
	}
	snap.Lines = make([]string, len(rs.lines))
	copy(snap.Lines, rs.lines)
	return snap
}

func (rs *runState) appendLine(line string) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.lines = append(rs.lines, line)
}

func (rs *runState) finish(err error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.done = true
	rs.running = false
	rs.exitErr = err
}

// startBackgroundRun launches the phase command in the background, streaming output line by line.
func (a *perfWebApp) startBackgroundRun(phaseID string) {
	rs := &runState{
		running:   true,
		startedAt: time.Now(),
	}

	a.mu.Lock()
	if a.runs == nil {
		a.runs = make(map[string]*runState)
	}
	a.runs[phaseID] = rs
	a.mu.Unlock()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		cmd, err := buildPhaseRunCommand(ctx, phaseID)
		if err != nil {
			rs.finish(err)
			return
		}
		cmd.Dir = a.repoRoot

		// Create pipes for streaming
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			rs.finish(fmt.Errorf("stdout pipe: %w", err))
			return
		}
		cmd.Stderr = cmd.Stdout // merge stderr into stdout pipe

		if err := cmd.Start(); err != nil {
			rs.finish(fmt.Errorf("start: %w", err))
			return
		}

		// Read lines as they arrive
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 0, 256*1024), 1024*1024)
		for scanner.Scan() {
			rs.appendLine(scanner.Text())
		}

		// Drain any remaining output
		if scanner.Err() != nil && scanner.Err() != io.EOF {
			rs.appendLine(fmt.Sprintf("[scanner error: %v]", scanner.Err()))
		}

		err = cmd.Wait()
		rs.finish(err)
	}()
}

func buildPhaseRunCommand(ctx context.Context, phaseID string) (*exec.Cmd, error) {
	switch phaseID {
	case "phase1":
		return exec.CommandContext(
			ctx,
			"go",
			"run",
			"./cmd/goja-perf",
			"phase1-run",
			"--output-file",
			defaultPhase1OutputFile,
			"--output-dir",
			defaultPhase1TaskOutputDir,
		), nil
	case "phase2":
		return exec.CommandContext(
			ctx,
			"go",
			"run",
			"./cmd/goja-perf",
			"phase2-run",
			"--output-file",
			defaultPhase2OutputFile,
			"--output-dir",
			defaultPhase2TaskOutputDir,
		), nil
	default:
		return nil, fmt.Errorf("%w: %s", os.ErrInvalid, phaseID)
	}
}

// getRunState returns the current run state for a phase (nil if no run has happened).
func (a *perfWebApp) getRunState(phaseID string) *runState {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.runs == nil {
		return nil
	}
	return a.runs[phaseID]
}

// isRunning checks if a phase currently has a run in progress.
func (a *perfWebApp) isRunning(phaseID string) bool {
	rs := a.getRunState(phaseID)
	if rs == nil {
		return false
	}
	rs.mu.Lock()
	defer rs.mu.Unlock()
	return rs.running
}

// progressViewData holds the template data for the streaming progress display.
type progressViewData struct {
	Phase      string
	Elapsed    string
	Lines      []string
	TailLines  []string // last N lines for display
	LineCount  int
	BenchLines []string // lines starting with "Benchmark" (detected results)
	BenchCount int
	Done       bool
	HasError   bool
	ErrorText  string
}

const maxTailLines = 30

func buildProgressView(phaseID string, snap runSnapshot) progressViewData {
	pv := progressViewData{
		Phase:     phaseID,
		Elapsed:   fmtElapsed(snap.Elapsed),
		Lines:     snap.Lines,
		LineCount: len(snap.Lines),
		Done:      snap.Done,
	}

	// Extract benchmark result lines
	for _, line := range snap.Lines {
		if strings.HasPrefix(line, "Benchmark") && strings.Contains(line, "ns/op") {
			pv.BenchLines = append(pv.BenchLines, line)
		}
	}
	pv.BenchCount = len(pv.BenchLines)

	// Tail lines for display
	if len(snap.Lines) > maxTailLines {
		pv.TailLines = snap.Lines[len(snap.Lines)-maxTailLines:]
	} else {
		pv.TailLines = snap.Lines
	}

	if snap.ExitErr != nil {
		pv.HasError = true
		pv.ErrorText = snap.ExitErr.Error()
	}

	return pv
}

func fmtElapsed(d time.Duration) string {
	d = d.Truncate(time.Second)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	m := int(d.Minutes())
	s := int(d.Seconds()) - m*60
	return fmt.Sprintf("%dm %ds", m, s)
}

// handleRunStreaming is the new POST /api/run/{phase} handler: starts background run, returns progress fragment.
func (a *perfWebApp) handleRunStreaming(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	phaseID := strings.TrimPrefix(r.URL.Path, "/api/run/")
	_, ok := a.phases[phaseID]
	if !ok {
		http.Error(w, "unknown phase", http.StatusNotFound)
		return
	}

	// Guard: only one run per phase at a time
	if a.isRunning(phaseID) {
		a.renderProgressFragment(w, phaseID)
		return
	}

	a.startBackgroundRun(phaseID)
	// Small delay so the first poll has something to show
	time.Sleep(100 * time.Millisecond)
	a.renderProgressFragment(w, phaseID)
}

// handleRunStatus is the GET /api/run-status/{phase} handler: returns current progress or final report.
func (a *perfWebApp) handleRunStatus(w http.ResponseWriter, r *http.Request) {
	phaseID := strings.TrimPrefix(r.URL.Path, "/api/run-status/")
	cfg, ok := a.phases[phaseID]
	if !ok {
		http.Error(w, "unknown phase", http.StatusNotFound)
		return
	}

	rs := a.getRunState(phaseID)
	if rs == nil {
		// No run ever started — show the report
		a.renderReportForPhase(w, phaseID)
		return
	}

	snap := rs.snapshot()
	if snap.Done {
		// Run finished — show the final report (with any errors)
		view := reportViewData{Phase: cfg.ID, Title: cfg.Title, ReportPath: cfg.OutputFile}
		report, err := a.loadReport(cfg)
		if err != nil {
			view.HasError = true
			if snap.ExitErr != nil {
				view.Error = fmt.Sprintf("Run failed: %v", snap.ExitErr)
			} else {
				view.Error = fmt.Sprintf("Run completed but report could not be loaded: %v", err)
			}
			view.RunOutput = strings.Join(snap.Lines, "\n")
			a.renderFragment(w, view)
			return
		}
		view.HasReport = true
		view.UpdatedAt = report.GeneratedAt
		view.Summary = report.Summary
		view.Results = report.Results
		view.Tasks = prepareTasks(report.Results)
		if snap.ExitErr != nil {
			view.RunOutput = strings.Join(snap.Lines, "\n")
		}
		a.renderFragment(w, view)
		return
	}

	// Still running — show progress with polling
	a.renderProgressFragment(w, phaseID)
}

func (a *perfWebApp) renderProgressFragment(w http.ResponseWriter, phaseID string) {
	rs := a.getRunState(phaseID)
	if rs == nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<div class="text-muted py-4 text-center">No run in progress.</div>`))
		return
	}

	snap := rs.snapshot()
	pv := buildProgressView(phaseID, snap)

	tmpl := template.Must(template.New("progress").Parse(progressTemplate))
	buf := &bytes.Buffer{}
	if err := tmpl.Execute(buf, pv); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(buf.Bytes())
}

const progressTemplate = `<div id="run-progress"
     hx-get="/api/run-status/{{.Phase}}"
     hx-trigger="every 1s"
     hx-swap="outerHTML">

  <div class="d-flex align-items-center gap-3 mb-3 py-2">
    <span class="loading-spinner"></span>
    <strong>Running benchmarks&#x2026;</strong>
    <span class="text-muted small">{{.Elapsed}} elapsed</span>
    {{if gt .BenchCount 0}}
      <span class="badge bg-primary">{{.BenchCount}} result{{if ne .BenchCount 1}}s{{end}} so far</span>
    {{end}}
    <span class="text-muted small">{{.LineCount}} output lines</span>
  </div>

  {{if gt .BenchCount 0}}
  <div class="mb-3">
    <div class="small fw-semibold mb-1">&#x2713; Benchmark results detected:</div>
    <div style="max-height:200px;overflow:auto;background:#f8f9fa;border:1px solid #dee2e6;border-radius:6px;padding:8px;">
      {{range .BenchLines}}
        <div class="small" style="font-family:monospace;white-space:pre;color:#198754;">{{.}}</div>
      {{end}}
    </div>
  </div>
  {{end}}

  <div>
    <div class="small fw-semibold mb-1">Live output (last {{len .TailLines}} lines):</div>
    <pre class="small mb-0" style="max-height:300px;overflow:auto;background:#1e1e1e;color:#d4d4d4;border-radius:6px;padding:12px;font-size:0.8rem;">{{range .TailLines}}{{.}}
{{end}}</pre>
  </div>

  {{if .HasError}}
    <div class="alert alert-danger mt-2 small">{{.ErrorText}}</div>
  {{end}}
</div>`
