package main

import (
	"context"
	"os"
	"strings"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/plan"
	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/specv2"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
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
		cmds.WithShort("Validate an xgoja/v2 spec and report problems"),
		cmds.WithLong(`
Doctor validates the native xgoja/v2 specification before a full build.

Examples:
  xgoja doctor -f xgoja.yaml
  xgoja doctor -f xgoja.yaml --output json
`),
		cmds.WithFlags(
			fields.New("file", fields.TypeString,
				fields.WithDefault("xgoja.yaml"),
				fields.WithShortFlag("f"),
				fields.WithHelp("Path to the xgoja/v2 specification")),
		),
		cmds.WithSections(sections...),
	)}
}

func (c *doctorCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
	settings := doctorSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return err
	}
	if handled, err := c.runV2Doctor(ctx, settings.File, gp); handled {
		return err
	}
	return v1SpecRejectedError(settings.File)
}

func (c *doctorCommand) runV2Doctor(ctx context.Context, file string, gp middlewares.Processor) (bool, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return false, nil
	}
	kind, _, err := specv2.DetectSchema(data)
	if err != nil || kind != specv2.SchemaKindV2 {
		return false, err
	}
	cfg, err := specv2.LoadFile(file)
	if cfg != nil {
		if addErr := gp.AddRow(ctx, types.NewRow(
			types.MRP("check", "schema"),
			types.MRP("status", "ok"),
			types.MRP("path", "schema"),
			types.MRP("file", file),
			types.MRP("message", cfg.Schema),
		)); addErr != nil {
			return true, addErr
		}
	}
	if err != nil {
		if addErr := gp.AddRow(ctx, types.NewRow(
			types.MRP("check", "specv2"),
			types.MRP("status", "error"),
			types.MRP("path", file),
			types.MRP("file", file),
			types.MRP("message", err.Error()),
		)); addErr != nil {
			return true, addErr
		}
		return true, err
	}
	compiled, err := plan.Compile(plan.Options{Config: *cfg, Providers: syntheticProviderRegistryFromV2(cfg), StartDir: cfg.BaseDir})
	if err != nil {
		if addErr := gp.AddRow(ctx, types.NewRow(
			types.MRP("check", "plan"),
			types.MRP("status", "error"),
			types.MRP("path", file),
			types.MRP("file", file),
			types.MRP("message", err.Error()),
		)); addErr != nil {
			return true, addErr
		}
		return true, err
	}
	if err := emitPlanRows(ctx, gp, file, compiled); err != nil {
		return true, err
	}
	return true, nil
}

func emitPlanRows(ctx context.Context, gp middlewares.Processor, file string, compiled *plan.Plan) error {
	if compiled == nil {
		return nil
	}
	for _, module := range compiled.GoModules.Modules {
		if err := gp.AddRow(ctx, types.NewRow(
			types.MRP("check", "module-resolution"),
			types.MRP("status", "ok"),
			types.MRP("path", module.ModulePath),
			types.MRP("file", file),
			types.MRP("message", string(module.ResolutionSource)),
			types.MRP("module_path", module.ModulePath),
			types.MRP("version", module.Version),
			types.MRP("local_dir", module.LocalDir),
			types.MRP("resolution_kind", string(module.ResolutionKind)),
			types.MRP("resolution_source", string(module.ResolutionSource)),
			types.MRP("required_by", strings.Join(module.RequiredBy, ",")),
		)); err != nil {
			return err
		}
	}
	for _, source := range compiled.Config.Sources {
		if err := gp.AddRow(ctx, types.NewRow(
			types.MRP("check", "source-plan"),
			types.MRP("status", "ok"),
			types.MRP("path", source.ID),
			types.MRP("file", file),
			types.MRP("message", string(source.Kind)),
			types.MRP("source_id", source.ID),
			types.MRP("source_kind", string(source.Kind)),
			types.MRP("source_files", len(compiled.SourceGraph.FilesForSourceSet(source.ID))),
		)); err != nil {
			return err
		}
	}
	return nil
}

func syntheticProviderRegistryFromV2(cfg *specv2.Config) *providerapi.ProviderRegistry {
	registry := providerapi.NewProviderRegistry()
	if cfg == nil {
		return registry
	}
	modulesByProvider := map[string]map[string]bool{}
	commandSetsByProvider := map[string]map[string]bool{}
	verbSourcesByProvider := map[string]map[string]bool{}
	for _, module := range cfg.Runtime.Modules {
		if modulesByProvider[module.Provider] == nil {
			modulesByProvider[module.Provider] = map[string]bool{}
		}
		modulesByProvider[module.Provider][module.Name] = true
	}
	for _, command := range cfg.Commands {
		if command.Type == "provider.command-set" {
			if commandSetsByProvider[command.Provider] == nil {
				commandSetsByProvider[command.Provider] = map[string]bool{}
			}
			commandSetsByProvider[command.Provider][command.Name] = true
		}
	}
	for _, source := range cfg.Sources {
		if source.From.Provider == nil {
			continue
		}
		provider := source.From.Provider.Provider
		if verbSourcesByProvider[provider] == nil {
			verbSourcesByProvider[provider] = map[string]bool{}
		}
		verbSourcesByProvider[provider][source.From.Provider.Source] = true
	}
	for _, provider := range cfg.Providers {
		entries := []providerapi.Entry{}
		for moduleName := range modulesByProvider[provider.ID] {
			name := moduleName
			entries = append(entries, providerapi.Module{Name: name, DefaultAs: name, NewModuleFactory: func(providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
				return func(*goja.Runtime, *goja.Object) {}, nil
			}})
		}
		for commandName := range commandSetsByProvider[provider.ID] {
			name := commandName
			entries = append(entries, providerapi.CommandSetProvider{Name: name, NewCommandSet: func(providerapi.CommandSetContext) (*providerapi.CommandSet, error) {
				return &providerapi.CommandSet{}, nil
			}})
		}
		for sourceName := range verbSourcesByProvider[provider.ID] {
			entries = append(entries, providerapi.VerbSource{Name: sourceName, FS: emptyDirFS{}, Root: "."})
		}
		_ = registry.Package(provider.ID, entries...)
	}
	return registry
}
