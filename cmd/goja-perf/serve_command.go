package main

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
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

func (c *serveCommand) Run(_ context.Context, vals *values.Values) error {
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

	return srv.ListenAndServe()
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
</head>
<body class="bg-light">
  <div class="container py-4">
    <h1 class="mb-3">Goja Performance Dashboard</h1>
    <p class="text-muted">Run phase benchmarks and inspect YAML-backed reports.</p>

    <div class="row g-3">
      <div class="col-lg-6">
        <div class="card shadow-sm">
          <div class="card-header d-flex justify-content-between align-items-center">
            <strong>Phase 1</strong>
            <div class="btn-group">
              <button class="btn btn-sm btn-primary"
                      hx-post="/api/run/phase1"
                      hx-target="#phase1-report"
                      hx-swap="innerHTML">Run</button>
              <button class="btn btn-sm btn-outline-secondary"
                      hx-get="/api/report/phase1"
                      hx-target="#phase1-report"
                      hx-swap="innerHTML">Refresh</button>
            </div>
          </div>
          <div id="phase1-report" class="card-body"
               hx-get="/api/report/phase1"
               hx-trigger="load"
               hx-swap="innerHTML">
            Loading...
          </div>
        </div>
      </div>

      <div class="col-lg-6">
        <div class="card shadow-sm">
          <div class="card-header d-flex justify-content-between align-items-center">
            <strong>Phase 2</strong>
            <div class="btn-group">
              <button class="btn btn-sm btn-primary"
                      hx-post="/api/run/phase2"
                      hx-target="#phase2-report"
                      hx-swap="innerHTML">Run</button>
              <button class="btn btn-sm btn-outline-secondary"
                      hx-get="/api/report/phase2"
                      hx-target="#phase2-report"
                      hx-swap="innerHTML">Refresh</button>
            </div>
          </div>
          <div id="phase2-report" class="card-body"
               hx-get="/api/report/phase2"
               hx-trigger="load"
               hx-swap="innerHTML">
            Loading...
          </div>
        </div>
      </div>
    </div>
  </div>
</body>
</html>`

const fragmentTemplate = `<div>
  <div class="small text-muted mb-2">report: {{.ReportPath}}</div>
  {{if .HasError}}
    <div class="alert alert-warning">
      <div><strong>{{.Error}}</strong></div>
      {{if .RunOutput}}
      <details class="mt-2"><summary>Command Output</summary><pre class="mt-2 mb-0 small">{{.RunOutput}}</pre></details>
      {{end}}
    </div>
  {{end}}

  {{if .HasReport}}
    <div class="mb-2 small text-muted">updated: {{.UpdatedAt}}</div>
    <ul class="list-group mb-3">
      <li class="list-group-item d-flex justify-content-between"><span>Total tasks</span><strong>{{.Summary.TotalTasks}}</strong></li>
      <li class="list-group-item d-flex justify-content-between"><span>Successful</span><strong>{{.Summary.SuccessfulTasks}}</strong></li>
      <li class="list-group-item d-flex justify-content-between"><span>Failed</span><strong>{{.Summary.FailedTasks}}</strong></li>
      <li class="list-group-item d-flex justify-content-between"><span>Duration (ms)</span><strong>{{.Summary.TotalDurationMS}}</strong></li>
    </ul>

    {{range .Results}}
      <div class="border rounded p-2 mb-3 bg-white">
        <div class="d-flex justify-content-between align-items-center">
          <strong>{{.TaskTitle}}</strong>
          <span class="badge {{if .Success}}text-bg-success{{else}}text-bg-danger{{end}}">{{if .Success}}success{{else}}failed{{end}}</span>
        </div>
        <div class="small text-muted">{{.ID}} • {{.DurationMS}} ms • exit={{.ExitCode}}</div>
        <p class="small mt-2 mb-2">{{.TaskDescription}}</p>

        {{if .BenchmarkDefinitions}}
        <div class="small fw-semibold mt-2">What this task measures</div>
        <div class="table-responsive">
          <table class="table table-sm table-bordered align-middle mb-2">
            <thead class="table-light"><tr><th>Benchmark</th><th>Description</th></tr></thead>
            <tbody>
              {{range .BenchmarkDefinitions}}
              <tr><td><code>{{.Name}}</code></td><td>{{.Description}}</td></tr>
              {{end}}
            </tbody>
          </table>
        </div>
        {{end}}

        {{if .Summaries}}
        <div class="small fw-semibold mt-2">Structured results</div>
        <div class="table-responsive">
          <table class="table table-sm table-striped table-bordered align-middle mb-0">
            <thead class="table-light">
              <tr>
                <th>Benchmark</th>
                <th>What it does</th>
                <th>Runs</th>
                <th>Metric</th>
                <th>Avg</th>
                <th>Min</th>
                <th>Max</th>
              </tr>
            </thead>
            <tbody>
              {{range .Summaries}}
                {{$bench := .Benchmark}}
                {{$description := .Description}}
                {{$runs := .Runs}}
                {{range .Metrics}}
                  <tr>
                    <td><code>{{$bench}}</code></td>
                    <td>{{$description}}</td>
                    <td>{{$runs}}</td>
                    <td><code>{{.Metric}}</code></td>
                    <td>{{fmtFloat .Avg}}</td>
                    <td>{{fmtFloat .Min}}</td>
                    <td>{{fmtFloat .Max}}</td>
                  </tr>
                {{end}}
              {{end}}
            </tbody>
          </table>
        </div>
        {{else}}
        <div class="small text-muted">No structured benchmark rows parsed.</div>
        {{end}}
      </div>
    {{end}}
  {{else if not .HasError}}
    <div class="text-muted">No report yet. Click Run.</div>
  {{end}}
</div>`
