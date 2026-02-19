package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"gopkg.in/yaml.v3"
)

type phase1RunCommand struct {
	*cmds.CommandDescription
}

var _ cmds.BareCommand = (*phase1RunCommand)(nil)

func newPhase1RunCommand() (*phase1RunCommand, error) {
	desc := cmds.NewCommandDescription(
		"phase1-run",
		cmds.WithShort("Run phase-1 benchmark tasks and print YAML report"),
		cmds.WithLong(`Run phase-1 benchmark tasks and emit structured YAML results.

This command uses Glazed only for command/flag definitions.
Output formatting is plain YAML.`),
		cmds.WithFlags(
			fields.New("repo-root", fields.TypeString, fields.WithDefault(defaultRepoRoot), fields.WithHelp("Repository root for command execution")),
			fields.New("bench-package", fields.TypeString, fields.WithDefault(defaultBenchPackage), fields.WithHelp("Go package containing benchmarks")),
			fields.New("count", fields.TypeInteger, fields.WithDefault(defaultPhase1Count), fields.WithHelp("Benchmark sample count")),
			fields.New("benchtime", fields.TypeString, fields.WithDefault(defaultPhase1Benchtime), fields.WithHelp("Go benchmark benchtime")),
			fields.New("benchmem", fields.TypeBool, fields.WithDefault(true), fields.WithHelp("Include allocation metrics")),
			fields.New("timeout", fields.TypeString, fields.WithDefault(defaultPhase1CommandTimeout), fields.WithHelp("Per-go-test timeout")),
			fields.New("output-file", fields.TypeString, fields.WithDefault(defaultPhase1OutputFile), fields.WithHelp("YAML report output path")),
			fields.New("output-dir", fields.TypeString, fields.WithDefault(defaultPhase1TaskOutputDir), fields.WithHelp("Directory for per-task raw command output")),
			fields.New("fail-fast", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Stop executing remaining tasks on first failure")),
		),
	)

	return &phase1RunCommand{CommandDescription: desc}, nil
}

func (c *phase1RunCommand) Run(ctx context.Context, vals *values.Values) error {
	settings := phase1CommandSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return err
	}
	if err := validatePhase1Settings(settings); err != nil {
		return err
	}

	tasks := buildPhase1Tasks(settings)
	if err := os.MkdirAll(settings.OutputDir, 0o755); err != nil {
		return err
	}
	if settings.OutputFile != "" {
		if err := os.MkdirAll(filepath.Dir(settings.OutputFile), 0o755); err != nil {
			return err
		}
	}

	report := phase1RunReport{
		Phase:       "phase-1",
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		RepoRoot:    settings.RepoRoot,
		Plan:        tasks,
		Results:     []phase1TaskResult{},
	}

	for _, task := range tasks {
		result := runPhase1Task(ctx, settings, task)
		report.Results = append(report.Results, result)
		if settings.FailFast && !result.Success {
			break
		}
	}

	report.Summary = summarizePhase1Results(report.Results)

	raw, err := yaml.Marshal(report)
	if err != nil {
		return err
	}

	if settings.OutputFile != "" {
		if err := os.WriteFile(settings.OutputFile, raw, 0o644); err != nil {
			return err
		}
	}

	fmt.Print(string(raw))
	return nil
}

func runPhase1Task(ctx context.Context, settings phase1CommandSettings, task phase1Task) phase1TaskResult {
	started := time.Now()
	cmd := exec.CommandContext(ctx, task.Command, task.Args...)
	cmd.Dir = settings.RepoRoot
	output, err := cmd.CombinedOutput()
	duration := time.Since(started)

	outputFile := filepath.Join(settings.OutputDir, task.ID+".txt")
	writeErr := os.WriteFile(outputFile, output, 0o644)

	exitCode := 0
	success := err == nil
	errorMessage := ""
	if err != nil {
		success = false
		errorMessage = err.Error()
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}
	if writeErr != nil {
		success = false
		if errorMessage == "" {
			errorMessage = writeErr.Error()
		} else {
			errorMessage = errorMessage + "; output write error: " + writeErr.Error()
		}
	}

	result := phase1TaskResult{
		ID:                   task.ID,
		TaskTitle:            task.Title,
		TaskDescription:      task.Description,
		BenchmarkDefinitions: task.Benchmarks,
		Command:              task.Command,
		Args:                 task.Args,
		OutputFile:           outputFile,
		DurationMS:           duration.Milliseconds(),
		Success:              success,
		ExitCode:             exitCode,
		Samples:              parseBenchmarkSamples(string(output), task.Benchmarks),
		Error:                errorMessage,
	}
	result.Summaries = summarizeBenchmarkSamples(result.Samples)

	return result
}

func parseBenchmarkSamples(output string, definitions []benchmarkDefinition) []benchmarkSample {
	samples := []benchmarkSample{}
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "Benchmark") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		iterations, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil {
			continue
		}

		metrics := map[string]float64{}
		for i := 2; i+1 < len(fields); i += 2 {
			value, err := strconv.ParseFloat(fields[i], 64)
			if err != nil {
				continue
			}
			metric := fields[i+1]
			metrics[metric] = value
		}

		samples = append(samples, benchmarkSample{
			Benchmark:   fields[0],
			Description: lookupBenchmarkDescription(fields[0], definitions),
			Iterations:  iterations,
			Metrics:     metrics,
			RawLine:     line,
		})
	}
	return samples
}

func summarizeBenchmarkSamples(samples []benchmarkSample) []benchmarkSummary {
	byName := map[string][]benchmarkSample{}
	for _, sample := range samples {
		byName[sample.Benchmark] = append(byName[sample.Benchmark], sample)
	}

	names := make([]string, 0, len(byName))
	for name := range byName {
		names = append(names, name)
	}
	sort.Strings(names)

	out := make([]benchmarkSummary, 0, len(names))
	for _, name := range names {
		group := byName[name]
		metricsByName := map[string][]float64{}
		for _, sample := range group {
			for metric, value := range sample.Metrics {
				metricsByName[metric] = append(metricsByName[metric], value)
			}
		}

		metricNames := make([]string, 0, len(metricsByName))
		for metric := range metricsByName {
			metricNames = append(metricNames, metric)
		}
		sort.Strings(metricNames)

		metricStats := make([]benchmarkMetricStat, 0, len(metricNames))
		for _, metric := range metricNames {
			values := metricsByName[metric]
			if len(values) == 0 {
				continue
			}
			minValue := values[0]
			maxValue := values[0]
			sum := 0.0
			for _, value := range values {
				if value < minValue {
					minValue = value
				}
				if value > maxValue {
					maxValue = value
				}
				sum += value
			}
			metricStats = append(metricStats, benchmarkMetricStat{
				Metric: metric,
				Avg:    sum / float64(len(values)),
				Min:    minValue,
				Max:    maxValue,
			})
		}

		out = append(out, benchmarkSummary{
			Benchmark:   name,
			Description: group[0].Description,
			Runs:        len(group),
			Metrics:     metricStats,
		})
	}

	return out
}

func lookupBenchmarkDescription(benchmarkName string, defs []benchmarkDefinition) string {
	for _, def := range defs {
		if benchmarkName == def.Name || strings.HasPrefix(benchmarkName, def.Name+"/") {
			return def.Description
		}
	}
	return ""
}

func summarizePhase1Results(results []phase1TaskResult) phase1RunSummary {
	summary := phase1RunSummary{TotalTasks: len(results)}
	for _, result := range results {
		summary.TotalDurationMS += result.DurationMS
		if result.Success {
			summary.SuccessfulTasks++
		} else {
			summary.FailedTasks++
		}
	}
	summary.TotalDurationSec = fmt.Sprintf("%.3f", float64(summary.TotalDurationMS)/1000.0)
	return summary
}
