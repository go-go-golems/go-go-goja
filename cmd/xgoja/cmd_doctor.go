package main

import (
	"context"
	"strings"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/buildspec"
)

type doctorCommand struct {
	*cmds.CommandDescription
}

var _ cmds.GlazeCommand = (*doctorCommand)(nil)

type doctorSettings struct {
	File string `glazed:"file"`
}

func newDoctorCommand() *doctorCommand {
	sections, err := commandSections()
	if err != nil {
		panic(err)
	}
	return &doctorCommand{CommandDescription: cmds.NewCommandDescription("doctor",
		cmds.WithShort("Validate an xgoja build spec and report problems"),
		cmds.WithLong(`
Doctor validates the xgoja specification before a full build. Phase 1 emits the
wired command shape; schema validation is added in the buildspec task.

Examples:
  xgoja doctor -f xgoja.yaml
  xgoja doctor -f xgoja.yaml --output json
`),
		cmds.WithFlags(
			fields.New("file", fields.TypeString,
				fields.WithDefault("xgoja.yaml"),
				fields.WithShortFlag("f"),
				fields.WithHelp("Path to the xgoja build specification")),
		),
		cmds.WithSections(sections...),
	)}
}

func (c *doctorCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
	settings := doctorSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return err
	}
	buildSpec, report, err := buildspec.LoadFile(settings.File)
	if report == nil {
		return err
	}
	for _, check := range report.Checks {
		if addErr := gp.AddRow(ctx, types.NewRow(
			types.MRP("check", check.Name),
			types.MRP("status", string(check.Status)),
			types.MRP("path", check.Path),
			types.MRP("file", settings.File),
			types.MRP("message", check.Message),
		)); addErr != nil {
			return addErr
		}
	}
	if err != nil || buildSpec == nil {
		return err
	}
	modulePlan, planErr := goModulePlanForBuildSpec(buildSpec)
	if planErr != nil {
		if addErr := gp.AddRow(ctx, types.NewRow(
			types.MRP("check", "module-resolution"),
			types.MRP("status", "error"),
			types.MRP("path", "workspace"),
			types.MRP("file", settings.File),
			types.MRP("message", planErr.Error()),
		)); addErr != nil {
			return addErr
		}
		return planErr
	}
	if modulePlan != nil {
		for _, module := range modulePlan.Modules {
			if addErr := gp.AddRow(ctx, types.NewRow(
				types.MRP("check", "module-resolution"),
				types.MRP("status", "ok"),
				types.MRP("path", module.ModulePath),
				types.MRP("file", settings.File),
				types.MRP("message", string(module.ResolutionSource)),
				types.MRP("module_path", module.ModulePath),
				types.MRP("version", module.Version),
				types.MRP("local_dir", module.LocalDir),
				types.MRP("resolution_kind", string(module.ResolutionKind)),
				types.MRP("resolution_source", string(module.ResolutionSource)),
				types.MRP("required_by", strings.Join(module.RequiredBy, ",")),
			)); addErr != nil {
				return addErr
			}
		}
	}
	return nil
}
