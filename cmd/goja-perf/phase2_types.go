package main

import "strconv"

const (
	defaultPhase2Count          = 3
	defaultPhase2Benchtime      = "250ms"
	defaultPhase2OutputFile     = "ttmp/2026/02/18/GJ-01-PERF--goja-performance-measurement-plan/various/phase2-run-results.yaml"
	defaultPhase2TaskDefsFile   = "ttmp/2026/02/18/GJ-01-PERF--goja-performance-measurement-plan/various/phase2-task-definitions.yaml"
	defaultPhase2TaskOutputDir  = "ttmp/2026/02/18/GJ-01-PERF--goja-performance-measurement-plan/various/phase2-task-output"
	defaultPhase2CommandTimeout = "20m"
)

type phase2Task = phase1Task

type phase2Plan = phase1Plan

type phase2TaskResult = phase1TaskResult

type phase2RunReport = phase1RunReport

type phase2CommandSettings = phase1CommandSettings

func buildPhase2Tasks(settings phase2CommandSettings) []phase2Task {
	return []phase2Task{
		newTaskFromExpression(settings, "p2-value-conversion", "Value Conversion", "Measure Runtime.ToValue / Value.Export / ExportTo conversion overhead.", "^BenchmarkValueConversion$"),
		newTaskFromExpression(settings, "p2-payload-sweep", "Payload Sweep", "Measure boundary-call behavior across tiny/medium/large payloads.", "^BenchmarkPayloadSweep$"),
		newTaskFromExpression(settings, "p2-gc-sensitivity", "GC Sensitivity", "Measure runtime spawn/reuse behavior under allocation-heavy scripts.", "^BenchmarkGCSensitivity$"),
	}
}

func newTaskFromExpression(settings phase2CommandSettings, id, title, description, benchExpr string) phase2Task {
	return phase2Task{
		ID:          id,
		Title:       title,
		Description: description,
		Command:     "go",
		Args:        makeGoTestArgs(settings, benchExpr),
		Flags:       makeGoTestFlags(settings, benchExpr),
	}
}

func makeGoTestArgs(settings phase2CommandSettings, benchExpr string) []string {
	args := []string{"test", settings.BenchPkg, "-run", "^$", "-bench", benchExpr, "-count", strconv.Itoa(settings.Count), "-benchtime", settings.Benchtime}
	if settings.BenchMem {
		args = append(args, "-benchmem")
	}
	if settings.Timeout != "" {
		args = append(args, "-timeout", settings.Timeout)
	}
	return args
}

func makeGoTestFlags(settings phase2CommandSettings, benchExpr string) map[string]string {
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

func validatePhase2Settings(settings phase2CommandSettings) error {
	return validatePhase1Settings(settings)
}
