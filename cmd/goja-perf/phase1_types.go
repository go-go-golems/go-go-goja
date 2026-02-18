package main

import (
	"fmt"
	"strconv"
)

const (
	defaultRepoRoot             = "."
	defaultBenchPackage         = "./perf/goja"
	defaultPhase1Count          = 3
	defaultPhase1Benchtime      = "200ms"
	defaultPhase1OutputFile     = "ttmp/2026/02/18/GJ-01-PERF--goja-performance-measurement-plan/various/phase1-run-results.yaml"
	defaultPhase1TaskDefsFile   = "ttmp/2026/02/18/GJ-01-PERF--goja-performance-measurement-plan/various/phase1-task-definitions.yaml"
	defaultPhase1TaskOutputDir  = "ttmp/2026/02/18/GJ-01-PERF--goja-performance-measurement-plan/various/phase1-task-output"
	defaultPhase1CommandTimeout = "15m"
)

type phase1Task struct {
	ID          string                `yaml:"id"`
	Title       string                `yaml:"title"`
	Description string                `yaml:"description"`
	Command     string                `yaml:"command"`
	Args        []string              `yaml:"args"`
	Flags       map[string]string     `yaml:"flags"`
	Benchmarks  []benchmarkDefinition `yaml:"benchmarks"`
}

type phase1Plan struct {
	Phase       string       `yaml:"phase"`
	GeneratedAt string       `yaml:"generated_at"`
	Tasks       []phase1Task `yaml:"tasks"`
}

type phase1RunSummary struct {
	TotalTasks       int    `yaml:"total_tasks"`
	SuccessfulTasks  int    `yaml:"successful_tasks"`
	FailedTasks      int    `yaml:"failed_tasks"`
	TotalDurationMS  int64  `yaml:"total_duration_ms"`
	TotalDurationSec string `yaml:"total_duration_seconds"`
}

type phase1TaskResult struct {
	ID                   string                `yaml:"id"`
	TaskTitle            string                `yaml:"task_title"`
	TaskDescription      string                `yaml:"task_description"`
	BenchmarkDefinitions []benchmarkDefinition `yaml:"benchmark_definitions"`
	Command              string                `yaml:"command"`
	Args                 []string              `yaml:"args"`
	OutputFile           string                `yaml:"output_file,omitempty"`
	DurationMS           int64                 `yaml:"duration_ms"`
	Success              bool                  `yaml:"success"`
	ExitCode             int                   `yaml:"exit_code"`
	Samples              []benchmarkSample     `yaml:"samples"`
	Summaries            []benchmarkSummary    `yaml:"summaries"`
	Error                string                `yaml:"error,omitempty"`
}

type phase1RunReport struct {
	Phase       string             `yaml:"phase"`
	GeneratedAt string             `yaml:"generated_at"`
	RepoRoot    string             `yaml:"repo_root"`
	Plan        []phase1Task       `yaml:"plan"`
	Results     []phase1TaskResult `yaml:"results"`
	Summary     phase1RunSummary   `yaml:"summary"`
}

type phase1CommandSettings struct {
	RepoRoot   string `glazed:"repo-root"`
	BenchPkg   string `glazed:"bench-package"`
	Count      int    `glazed:"count"`
	Benchtime  string `glazed:"benchtime"`
	BenchMem   bool   `glazed:"benchmem"`
	Timeout    string `glazed:"timeout"`
	OutputFile string `glazed:"output-file"`
	OutputDir  string `glazed:"output-dir"`
	FailFast   bool   `glazed:"fail-fast"`
}

type benchmarkDefinition struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

type benchmarkSample struct {
	Benchmark   string             `yaml:"benchmark"`
	Description string             `yaml:"description,omitempty"`
	Iterations  int64              `yaml:"iterations"`
	Metrics     map[string]float64 `yaml:"metrics"`
	RawLine     string             `yaml:"raw_line"`
}

type benchmarkSummary struct {
	Benchmark   string                `yaml:"benchmark"`
	Description string                `yaml:"description,omitempty"`
	Runs        int                   `yaml:"runs"`
	Metrics     []benchmarkMetricStat `yaml:"metrics"`
}

type benchmarkMetricStat struct {
	Metric string  `yaml:"metric"`
	Avg    float64 `yaml:"avg"`
	Min    float64 `yaml:"min"`
	Max    float64 `yaml:"max"`
}

func buildPhase1Tasks(settings phase1CommandSettings) []phase1Task {
	makeArgs := func(benchExpr string) []string {
		args := []string{"test", settings.BenchPkg, "-run", "^$", "-bench", benchExpr, "-count", strconv.Itoa(settings.Count), "-benchtime", settings.Benchtime}
		if settings.BenchMem {
			args = append(args, "-benchmem")
		}
		if settings.Timeout != "" {
			args = append(args, "-timeout", settings.Timeout)
		}
		return args
	}

	makeFlags := func(benchExpr string) map[string]string {
		flags := map[string]string{
			"bench-package": settings.BenchPkg,
			"run":           "^$",
			"bench":         benchExpr,
			"count":         strconv.Itoa(settings.Count),
			"benchtime":     settings.Benchtime,
			"timeout":       settings.Timeout,
		}
		if settings.BenchMem {
			flags["benchmem"] = "true"
		} else {
			flags["benchmem"] = "false"
		}
		return flags
	}

	return []phase1Task{
		{
			ID:          "p1-runtime-lifecycle",
			Title:       "Runtime Lifecycle",
			Description: "Measure VM spawn and spawn+execute/reuse behavior.",
			Command:     "go",
			Args:        makeArgs("^(BenchmarkRuntimeSpawn|BenchmarkRuntimeSpawnAndExecute|BenchmarkRuntimeReuse)$"),
			Flags:       makeFlags("^(BenchmarkRuntimeSpawn|BenchmarkRuntimeSpawnAndExecute|BenchmarkRuntimeReuse)$"),
			Benchmarks: []benchmarkDefinition{
				{Name: "BenchmarkRuntimeSpawn", Description: "Compare runtime creation costs, including calllog-enabled and calllog-disabled modes."},
				{Name: "BenchmarkRuntimeSpawnAndExecute", Description: "Measure cost of creating a runtime and immediately executing one script/program."},
				{Name: "BenchmarkRuntimeReuse", Description: "Measure repeated execution on a reused runtime for RunString vs precompiled RunProgram."},
			},
		},
		{
			ID:          "p1-loading-require",
			Title:       "Loading and Require",
			Description: "Measure JS compile/load cost and require cold/warm behavior.",
			Command:     "go",
			Args:        makeArgs("^(BenchmarkJSLoading|BenchmarkRequireLoading)$"),
			Flags:       makeFlags("^(BenchmarkJSLoading|BenchmarkRequireLoading)$"),
			Benchmarks: []benchmarkDefinition{
				{Name: "BenchmarkJSLoading", Description: "Measure compile/run costs across small, medium, and large scripts."},
				{Name: "BenchmarkRequireLoading", Description: "Measure module loading overhead in cold runtime and warm cached runtime paths."},
			},
		},
		{
			ID:          "p1-boundary-calls",
			Title:       "Go/JS Boundary Calls",
			Description: "Measure JS->Go and Go->JS call overhead and logging mode deltas.",
			Command:     "go",
			Args:        makeArgs("^(BenchmarkJSCallingGo|BenchmarkGoCallingJS)$"),
			Flags:       makeFlags("^(BenchmarkJSCallingGo|BenchmarkGoCallingJS)$"),
			Benchmarks: []benchmarkDefinition{
				{Name: "BenchmarkJSCallingGo", Description: "Compare direct Go baseline against calls crossing JS->Go boundary."},
				{Name: "BenchmarkGoCallingJS", Description: "Compare direct Go baseline against Go->JS calls with/without calllog wrappers."},
			},
		},
	}
}

func validatePhase1Settings(settings phase1CommandSettings) error {
	if settings.Count < 1 {
		return fmt.Errorf("count must be >= 1")
	}
	if settings.BenchPkg == "" {
		return fmt.Errorf("bench-package must not be empty")
	}
	if settings.Benchtime == "" {
		return fmt.Errorf("benchtime must not be empty")
	}
	if settings.RepoRoot == "" {
		return fmt.Errorf("repo-root must not be empty")
	}
	return nil
}
