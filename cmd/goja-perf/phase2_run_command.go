package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"gopkg.in/yaml.v3"
)

type phase2RunCommand struct {
	*cmds.CommandDescription
}

var _ cmds.BareCommand = (*phase2RunCommand)(nil)

func newPhase2RunCommand() (*phase2RunCommand, error) {
	desc := cmds.NewCommandDescription(
		"phase2-run",
		cmds.WithShort("Run phase-2 benchmark tasks and print YAML report"),
		cmds.WithLong(`Run phase-2 benchmark tasks and emit structured YAML results.

This command uses Glazed only for command/flag definitions.
Output formatting is plain YAML.`),
		cmds.WithFlags(
			fields.New("repo-root", fields.TypeString, fields.WithDefault(defaultRepoRoot), fields.WithHelp("Repository root for command execution")),
			fields.New("bench-package", fields.TypeString, fields.WithDefault(defaultBenchPackage), fields.WithHelp("Go package containing benchmarks")),
			fields.New("count", fields.TypeInteger, fields.WithDefault(defaultPhase2Count), fields.WithHelp("Benchmark sample count")),
			fields.New("benchtime", fields.TypeString, fields.WithDefault(defaultPhase2Benchtime), fields.WithHelp("Go benchmark benchtime")),
			fields.New("benchmem", fields.TypeBool, fields.WithDefault(true), fields.WithHelp("Include allocation metrics")),
			fields.New("timeout", fields.TypeString, fields.WithDefault(defaultPhase2CommandTimeout), fields.WithHelp("Per-go-test timeout")),
			fields.New("output-file", fields.TypeString, fields.WithDefault(defaultPhase2OutputFile), fields.WithHelp("YAML report output path")),
			fields.New("output-dir", fields.TypeString, fields.WithDefault(defaultPhase2TaskOutputDir), fields.WithHelp("Directory for per-task raw command output")),
			fields.New("fail-fast", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Stop executing remaining tasks on first failure")),
		),
	)

	return &phase2RunCommand{CommandDescription: desc}, nil
}

func (c *phase2RunCommand) Run(ctx context.Context, vals *values.Values) error {
	settings := phase2CommandSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return err
	}
	if err := validatePhase2Settings(settings); err != nil {
		return err
	}

	tasks := buildPhase2Tasks(settings)
	if err := os.MkdirAll(settings.OutputDir, 0o755); err != nil {
		return err
	}
	if settings.OutputFile != "" {
		if err := os.MkdirAll(filepath.Dir(settings.OutputFile), 0o755); err != nil {
			return err
		}
	}

	report := phase2RunReport{
		Phase:       "phase-2",
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		RepoRoot:    settings.RepoRoot,
		Plan:        tasks,
		Results:     []phase2TaskResult{},
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
