package testprovider

import (
	"context"
	"embed"
	"fmt"
	"io"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/go-go-golems/go-go-goja/pkg/runtimebridge"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

//go:embed verbs/*.js
var verbsFS embed.FS

func Register(registry *providerapi.Registry) error {
	return registry.Package("fixture",
		providerapi.Module{
			Name:        "hello",
			DefaultAs:   "hello",
			Description: "Fixture module used by xgoja tests",
			New: func(providerapi.ModuleContext) (require.ModuleLoader, error) {
				return func(vm *goja.Runtime, module *goja.Object) {
					exports := module.Get("exports").(*goja.Object)
					_ = exports.Set("greet", func(name string) string { return "hello " + name })
				}, nil
			},
		},
		providerapi.Module{
			Name:        "owner-check",
			DefaultAs:   "owner-check",
			Description: "Fixture module that requires xgoja runtime services",
			New: func(providerapi.ModuleContext) (require.ModuleLoader, error) {
				return func(vm *goja.Runtime, module *goja.Object) {
					exports := module.Get("exports").(*goja.Object)
					_ = exports.Set("hasOwner", func() bool {
						runtimeServices, ok := runtimebridge.Lookup(vm)
						return ok && runtimeServices.Owner != nil && runtimeServices.Loop != nil
					})
					_ = exports.Set("pingAsync", func() goja.Value {
						promise, resolve, reject := vm.NewPromise()
						runtimeServices, ok := runtimebridge.Lookup(vm)
						if !ok || runtimeServices.Owner == nil {
							_ = reject(vm.ToValue("missing runtime services"))
							return vm.ToValue(promise)
						}
						callCtx := runtimebridge.CurrentOwnerContext(vm)
						go func() {
							if err := runtimeServices.PostWithCustomContext(callCtx, "owner-check.ping", func(context.Context, *goja.Runtime) {
								_ = resolve("pong")
							}); err != nil {
								_ = runtimeServices.PostWithCustomContext(context.Background(), "owner-check.ping.reject", func(context.Context, *goja.Runtime) {
									_ = reject(vm.ToValue(fmt.Sprintf("post failed: %v", err)))
								})
							}
						}()
						return vm.ToValue(promise)
					})
				}, nil
			},
		},
		providerapi.WithPackageCapability(FixtureCapability{}),
		providerapi.CommandSetProvider{
			Name:         "tools",
			DefaultMount: "fixture",
			Description:  "Fixture Glazed commands used by xgoja tests and examples",
			New:          NewFixtureCommandSet,
		},
		providerapi.VerbSource{Name: "verbs", Root: "verbs", FS: verbsFS},
	)
}

type FixtureSettings struct {
	Value string `glazed:"value"`
}

type FixtureCapability struct{}

func (FixtureCapability) CapabilityID() string { return "fixture.settings" }

func (FixtureCapability) ConfigSections(providerapi.SectionContext) ([]schema.Section, error) {
	section, err := FixtureSection()
	if err != nil {
		return nil, err
	}
	return []schema.Section{section}, nil
}

func (FixtureCapability) InitRuntimeFromSections(_ context.Context, vals *values.Values, handle providerapi.RuntimeHandle) error {
	var settings FixtureSettings
	if err := vals.DecodeSectionInto("fixture", &settings); err != nil {
		return err
	}
	if handle.Runtime() == nil {
		return fmt.Errorf("runtime handle has no goja runtime")
	}
	return handle.Runtime().Set("fixtureValue", settings.Value)
}

func FixtureSection() (schema.Section, error) {
	return schema.NewSection(
		"fixture",
		"Fixture settings",
		schema.WithPrefix("fixture-"),
		schema.WithFields(fields.New("value", fields.TypeString, fields.WithDefault(""))),
	)
}

type commandSettings struct {
	Message string `glazed:"message"`
}

func NewFixtureCommandSet(ctx providerapi.CommandSetContext) (*providerapi.CommandSet, error) {
	sections, err := sectionsFromSelectedModules(ctx)
	if err != nil {
		return nil, err
	}
	commands := []cmds.Command{
		&fixtureBareCommand{CommandDescription: fixtureDescription("bare", "Run a fixture bare command", sections)},
		&fixtureWriterCommand{CommandDescription: fixtureDescription("write", "Run a fixture writer command", sections)},
		&fixtureGlazeCommand{CommandDescription: fixtureDescription("rows", "Run a fixture glaze command", sections)},
	}
	return &providerapi.CommandSet{Commands: commands}, nil
}

func sectionsFromSelectedModules(ctx providerapi.CommandSetContext) ([]schema.Section, error) {
	sections := []schema.Section{}
	seen := map[string]struct{}{}
	for _, module := range ctx.SelectedModules {
		for _, capability := range module.PackageCapabilities {
			sectionCapability, ok := capability.(providerapi.ConfigSectionCapability)
			if !ok {
				continue
			}
			moduleSections, err := sectionCapability.ConfigSections(providerapi.SectionContext{
				CommandProviderID: ctx.Name,
				RuntimeProfile:    "",
				PackageID:         module.PackageID,
				ModuleID:          module.ModuleID,
			})
			if err != nil {
				return nil, err
			}
			for _, section := range moduleSections {
				if _, ok := seen[section.GetSlug()]; ok {
					continue
				}
				seen[section.GetSlug()] = struct{}{}
				sections = append(sections, section)
			}
		}
	}
	return sections, nil
}

func fixtureDescription(name, short string, sections []schema.Section) *cmds.CommandDescription {
	options := []cmds.CommandDescriptionOption{
		cmds.WithShort(short),
		cmds.WithFlags(fields.New("message", fields.TypeString, fields.WithDefault("hello"))),
	}
	if len(sections) > 0 {
		options = append(options, cmds.WithSections(sections...))
	}
	return cmds.NewCommandDescription(name, options...)
}

func decodeFixtureCommand(vals *values.Values) (commandSettings, FixtureSettings, error) {
	var command commandSettings
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &command); err != nil {
		return commandSettings{}, FixtureSettings{}, err
	}
	var fixture FixtureSettings
	if err := vals.DecodeSectionInto("fixture", &fixture); err != nil {
		return commandSettings{}, FixtureSettings{}, err
	}
	return command, fixture, nil
}

type fixtureBareCommand struct{ *cmds.CommandDescription }

var _ cmds.BareCommand = (*fixtureBareCommand)(nil)

func (c *fixtureBareCommand) Run(_ context.Context, vals *values.Values) error {
	command, fixture, err := decodeFixtureCommand(vals)
	if err != nil {
		return err
	}
	if command.Message == "" || fixture.Value == "" {
		return fmt.Errorf("message and fixture value are required")
	}
	return nil
}

type fixtureWriterCommand struct{ *cmds.CommandDescription }

var _ cmds.WriterCommand = (*fixtureWriterCommand)(nil)

func (c *fixtureWriterCommand) RunIntoWriter(_ context.Context, vals *values.Values, w io.Writer) error {
	command, fixture, err := decodeFixtureCommand(vals)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "%s:%s\n", command.Message, fixture.Value)
	return err
}

type fixtureGlazeCommand struct{ *cmds.CommandDescription }

var _ cmds.GlazeCommand = (*fixtureGlazeCommand)(nil)

func (c *fixtureGlazeCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
	command, fixture, err := decodeFixtureCommand(vals)
	if err != nil {
		return err
	}
	return gp.AddRow(ctx, types.NewRow(
		types.MRP("message", command.Message),
		types.MRP("fixture", fixture.Value),
	))
}
