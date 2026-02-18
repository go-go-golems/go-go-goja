package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
		ID:             task.ID,
		Command:        task.Command,
		Args:           task.Args,
		OutputFile:     outputFile,
		DurationMS:     duration.Milliseconds(),
		Success:        success,
		ExitCode:       exitCode,
		BenchmarkLines: extractBenchmarkLines(string(output)),
		Error:          errorMessage,
	}

	return result
}

func extractBenchmarkLines(output string) []string {
	lines := []string{}
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Benchmark") || strings.HasPrefix(line, "PASS") || strings.HasPrefix(line, "ok\t") || strings.HasPrefix(line, "FAIL") {
			lines = append(lines, line)
		}
	}
	return lines
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
