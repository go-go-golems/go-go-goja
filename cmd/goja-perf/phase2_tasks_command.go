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

type phase2TasksCommand struct {
	*cmds.CommandDescription
}

var _ cmds.BareCommand = (*phase2TasksCommand)(nil)

func newPhase2TasksCommand() (*phase2TasksCommand, error) {
	desc := cmds.NewCommandDescription(
		"phase2-tasks",
		cmds.WithShort("Print phase-2 benchmark command definitions as YAML"),
		cmds.WithLong(`Emit phase-2 benchmark task definitions as YAML.

This command uses Glazed for command/flag definitions only.
Result formatting is plain YAML (not Glazed structured output).`),
		cmds.WithFlags(
			fields.New("repo-root", fields.TypeString, fields.WithDefault(defaultRepoRoot), fields.WithHelp("Repository root for command execution")),
			fields.New("bench-package", fields.TypeString, fields.WithDefault(defaultBenchPackage), fields.WithHelp("Go package containing benchmarks")),
			fields.New("count", fields.TypeInteger, fields.WithDefault(defaultPhase2Count), fields.WithHelp("Benchmark sample count")),
			fields.New("benchtime", fields.TypeString, fields.WithDefault(defaultPhase2Benchtime), fields.WithHelp("Go benchmark benchtime")),
			fields.New("benchmem", fields.TypeBool, fields.WithDefault(true), fields.WithHelp("Include allocation metrics")),
			fields.New("timeout", fields.TypeString, fields.WithDefault(defaultPhase2CommandTimeout), fields.WithHelp("Per-go-test timeout")),
			fields.New("output-file", fields.TypeString, fields.WithDefault(defaultPhase2TaskDefsFile), fields.WithHelp("Optional YAML output file path")),
		),
	)

	return &phase2TasksCommand{CommandDescription: desc}, nil
}

func (c *phase2TasksCommand) Run(_ context.Context, vals *values.Values) error {
	settings := phase2CommandSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return err
	}
	if err := validatePhase2Settings(settings); err != nil {
		return err
	}

	plan := phase2Plan{
		Phase:       "phase-2",
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Tasks:       buildPhase2Tasks(settings),
	}

	raw, err := yaml.Marshal(plan)
	if err != nil {
		return err
	}

	if settings.OutputFile != "" {
		if err := os.MkdirAll(filepath.Dir(settings.OutputFile), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(settings.OutputFile, raw, 0o644); err != nil {
			return err
		}
	}

	fmt.Print(string(raw))
	return nil
}
