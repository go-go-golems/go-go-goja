package main

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"gopkg.in/yaml.v3"
)

type serveCommand struct {
	*cmds.CommandDescription
}

var _ cmds.BareCommand = (*serveCommand)(nil)

type serveSettings struct {
	RepoRoot string `glazed:"repo-root"`
	Host     string `glazed:"host"`
	Port     int    `glazed:"port"`
}

type phaseConfig struct {
	ID         string
	Title      string
	OutputFile string
	OutputDir  string
}

type reportPageData struct {
	Host string
	Port int
}

type reportViewData struct {
	Phase      string
	Title      string
	ReportPath string
	UpdatedAt  string
	HasReport  bool
	HasError   bool
	Error      string
	Summary    phase1RunSummary
	Results    []phase1TaskResult
	Tasks      []taskViewData
	RunOutput  string
}

func newServeCommand() (*serveCommand, error) {
	desc := cmds.NewCommandDescription(
		"serve",
		cmds.WithShort("Run a local web UI for perf runs and report browsing"),
		cmds.WithLong(`Start a small local web app so you can:
- trigger phase-1 and phase-2 benchmark runs
- inspect generated YAML summaries
- inspect structured benchmark metrics rendered as tables

This command uses Glazed for command/flag definitions only.`),
		cmds.WithFlags(
			fields.New("repo-root", fields.TypeString, fields.WithDefault("."), fields.WithHelp("Repository root where goja-perf commands run")),
			fields.New("host", fields.TypeString, fields.WithDefault("127.0.0.1"), fields.WithHelp("HTTP bind host")),
			fields.New("port", fields.TypeInteger, fields.WithDefault(8090), fields.WithHelp("HTTP bind port")),
		),
	)
	return &serveCommand{CommandDescription: desc}, nil
}

func (c *serveCommand) Run(ctx context.Context, vals *values.Values) error {
	settings := serveSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return err
	}
	if settings.Host == "" {
		settings.Host = "127.0.0.1"
	}
	if settings.Port <= 0 {
		settings.Port = 8090
	}
	if settings.RepoRoot == "" {
		settings.RepoRoot = "."
	}

	root, err := filepath.Abs(settings.RepoRoot)
	if err != nil {
		return err
	}

	app := &perfWebApp{
		repoRoot: root,
		phases: map[string]phaseConfig{
			"phase1": {
				ID:         "phase1",
				Title:      "Phase 1",
				OutputFile: defaultPhase1OutputFile,
				OutputDir:  defaultPhase1TaskOutputDir,
			},
			"phase2": {
				ID:         "phase2",
				Title:      "Phase 2",
				OutputFile: defaultPhase2OutputFile,
				OutputDir:  defaultPhase2TaskOutputDir,
			},
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", app.handleIndex)
	mux.HandleFunc("/api/report/", app.handleReport)
	mux.HandleFunc("/api/run/", app.handleRun)

	addr := fmt.Sprintf("%s:%d", settings.Host, settings.Port)
	fmt.Printf("goja-perf web UI: http://%s\n", addr)
	fmt.Printf("repo root: %s\n", root)

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	runCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	return runServerUntilCanceled(runCtx, srv)
}

type perfWebApp struct {
	repoRoot string
	phases   map[string]phaseConfig
	mu       sync.Mutex
}

func (a *perfWebApp) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	pageTmpl := template.Must(template.New("page").Parse(indexTemplate))
	_ = pageTmpl.Execute(w, reportPageData{})
}

func (a *perfWebApp) handleReport(w http.ResponseWriter, r *http.Request) {
	phaseID := strings.TrimPrefix(r.URL.Path, "/api/report/")
	cfg, ok := a.phases[phaseID]
	if !ok {
		http.Error(w, "unknown phase", http.StatusNotFound)
		return
	}

	view := reportViewData{Phase: cfg.ID, Title: cfg.Title, ReportPath: cfg.OutputFile}
	report, err := a.loadReport(cfg)
	if err != nil {
		view.HasError = true
		view.Error = fmt.Sprintf("Report not available yet: %v", err)
		a.renderFragment(w, view)
		return
	}
	view.HasReport = true
	view.UpdatedAt = report.GeneratedAt
	view.Summary = report.Summary
	view.Results = report.Results
	view.Tasks = prepareTasks(report.Results)
	a.renderFragment(w, view)
}

func (a *perfWebApp) handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	phaseID := strings.TrimPrefix(r.URL.Path, "/api/run/")
	cfg, ok := a.phases[phaseID]
	if !ok {
		http.Error(w, "unknown phase", http.StatusNotFound)
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	view := reportViewData{Phase: cfg.ID, Title: cfg.Title, ReportPath: cfg.OutputFile}
	out, runErr := a.runPhaseCommand(r.Context(), cfg)
	if runErr != nil {
		view.HasError = true
		view.Error = fmt.Sprintf("Run failed: %v", runErr)
		view.RunOutput = out
		a.renderFragment(w, view)
		return
	}

	report, err := a.loadReport(cfg)
	if err != nil {
		view.HasError = true
		view.Error = fmt.Sprintf("Run succeeded, but report could not be loaded: %v", err)
		view.RunOutput = out
		a.renderFragment(w, view)
		return
	}

	view.HasReport = true
	view.UpdatedAt = report.GeneratedAt
	view.Summary = report.Summary
	view.Results = report.Results
	view.Tasks = prepareTasks(report.Results)
	view.RunOutput = out
	a.renderFragment(w, view)
}

func (a *perfWebApp) runPhaseCommand(ctx context.Context, cfg phaseConfig) (string, error) {
	args := []string{"run", "./cmd/goja-perf", cfg.ID + "-run", "--output-file", cfg.OutputFile, "--output-dir", cfg.OutputDir}
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = a.repoRoot
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (a *perfWebApp) loadReport(cfg phaseConfig) (*phase1RunReport, error) {
	path := filepath.Join(a.repoRoot, cfg.OutputFile)
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	report := &phase1RunReport{}
	if err := yaml.Unmarshal(content, report); err != nil {
		return nil, err
	}
	return report, nil
}

func (a *perfWebApp) renderFragment(w http.ResponseWriter, data reportViewData) {
	tmpl := template.Must(template.New("fragment").Funcs(template.FuncMap{
		"fmtFloat": func(v float64) string {
			return strconv.FormatFloat(v, 'f', 3, 64)
		},
		"fmtDuration": fmtDurationMS,
	}).Parse(fragmentTemplate))
	buf := &bytes.Buffer{}
	if err := tmpl.Execute(buf, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(buf.Bytes())
}

const indexTemplate = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Goja Perf Dashboard</title>
  <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.3/dist/css/bootstrap.min.css" rel="stylesheet">
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
    .bench-metric { font-family: 'SF Mono', 'Cascadia Code', 'Consolas', monospace;
                    font-size: 0.9rem; white-space: nowrap; }
    .bench-metric .value { font-weight: 700; color: #0d6efd; }
    .bench-metric .unit { color: #6c757d; font-weight: 400; }
    .bench-metric.tps .value { color: #198754; }
    .bench-bar-wrap { flex: 1; min-width: 120px; max-width: 300px; }
    .bench-bar { height: 8px; border-radius: 4px; background: #e9ecef; }
    .bench-bar-fill { height: 100%; border-radius: 4px;
                      background: linear-gradient(90deg, #0d6efd, #6610f2); }
    .bench-range { font-size: 0.78rem; color: #6c757d; margin-top: 2px; }
    .bench-relative { font-size: 0.78rem; color: #198754; font-weight: 500; }
    .bench-relative.slow { color: #dc3545; }
    .task-header { cursor: pointer; user-select: none; border-radius: 6px; }
    .task-header:hover { background: #f1f3f5; }
    .task-toggle { font-size: 0.75rem; color: #6c757d; }
    .status-badge { font-size: 0.75rem; padding: 2px 8px; border-radius: 4px; }
    .status-ok { background: #d1e7dd; color: #0f5132; }
    .status-fail { background: #f8d7da; color: #842029; }
    .loading-spinner { display: inline-block; width: 16px; height: 16px;
      border: 2px solid #0d6efd; border-right-color: transparent;
      border-radius: 50%; animation: spin .6s linear infinite;
      vertical-align: middle; }
    @keyframes spin { to { transform: rotate(360deg); } }
    .phase-tab { border: none; background: none; padding: 8px 20px;
      font-weight: 500; color: #6c757d; border-bottom: 3px solid transparent;
      cursor: pointer; font-size: 1rem; }
    .phase-tab.active { color: #0d6efd; border-bottom-color: #0d6efd; }
    .phase-tab:hover { color: #0d6efd; }
    .summary-bar { display: flex; gap: 20px; padding: 8px 0;
      font-size: 0.88rem; color: #495057; flex-wrap: wrap; }
    .summary-bar strong { color: #212529; }
  </style>
</head>
<body>
  <div class="container-fluid" style="max-width: 960px; margin: 0 auto;">
    <div class="py-4">
      <h1 class="mb-1" style="font-size: 1.6rem;">&#9889; Goja Performance Dashboard</h1>

      <div class="d-flex align-items-center gap-2 mt-3 mb-3 border-bottom">
        <button class="phase-tab active" id="tab-phase1"
                onclick="switchPhase('phase1')">Phase 1</button>
        <button class="phase-tab" id="tab-phase2"
                onclick="switchPhase('phase2')">Phase 2</button>
        <div class="ms-auto d-flex gap-2 pb-2">
          <button class="btn btn-sm btn-outline-secondary" id="btn-refresh"
                  onclick="refreshPhase()">&#x27F3; Refresh</button>
          <button class="btn btn-sm btn-primary" id="btn-run"
                  onclick="runPhase()">&#x25B6; Run</button>
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
      document.querySelectorAll('.phase-tab').forEach(function(t) { t.classList.remove('active'); });
      document.getElementById('tab-' + phase).classList.add('active');
      htmx.ajax('GET', '/api/report/' + phase, '#report-content');
    }
    function refreshPhase() {
      htmx.ajax('GET', '/api/report/' + currentPhase, '#report-content');
    }
    function runPhase() {
      var btn = document.getElementById('btn-run');
      btn.disabled = true;
      btn.innerHTML = '<span class="loading-spinner"></span> Running\u2026';
      htmx.ajax('POST', '/api/run/' + currentPhase, {target: '#report-content'}).then(function() {
        btn.disabled = false;
        btn.innerHTML = '\u25B6 Run';
      });
    }
    function toggleTask(id) {
      var el = document.getElementById('task-body-' + id);
      var arrow = document.getElementById('task-arrow-' + id);
      if (el.style.display === 'none') {
        el.style.display = 'block';
        arrow.textContent = '\u25BC';
      } else {
        el.style.display = 'none';
        arrow.textContent = '\u25B6';
      }
    }
  </script>
</body>
</html>`

const fragmentTemplate = `<div>
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
      <span>&#x1F4C5; <strong>{{.UpdatedAt}}</strong></span>
      <span>&#x2713; <strong>{{.Summary.SuccessfulTasks}}</strong>/{{.Summary.TotalTasks}} passed</span>
      {{if gt .Summary.FailedTasks 0}}
        <span style="color:#dc3545;">&#x2717; <strong>{{.Summary.FailedTasks}}</strong> failed</span>
      {{end}}
      <span>&#x23F1; <strong>{{fmtDuration .Summary.TotalDurationMS}}</strong></span>
    </div>

    {{range $i, $task := .Tasks}}
      <div class="border rounded mb-3 overflow-hidden">
        <div class="task-header d-flex align-items-center px-3 py-2"
             onclick="toggleTask('{{$task.ID}}')">
          <span id="task-arrow-{{$task.ID}}" class="task-toggle me-2">{{if eq $i 0}}&#x25BC;{{else}}&#x25B6;{{end}}</span>
          <strong class="me-2">{{$task.Title}}</strong>
          <span class="text-muted small me-auto">{{$task.Description}}</span>
          <span class="text-muted small me-2">{{fmtDuration $task.DurationMS}}</span>
          <span class="status-badge {{if $task.Success}}status-ok{{else}}status-fail{{end}}">
            {{if $task.Success}}&#x2713;{{else}}&#x2717;{{end}}
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
                {{if .Description}}
                  <div class="small text-muted" style="margin-top:2px;">{{.Description}}</div>
                {{end}}
                <div class="bench-metrics">
                  <span class="bench-metric"><span class="value">{{.NsFormatted}}</span></span>
                  <span class="bench-metric tps"><span class="value">{{.TpsFormatted}}</span></span>
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
    <div class="text-muted text-center py-4">No report yet. Click &#x25B6; Run to start benchmarks.</div>
  {{end}}
</div>`
