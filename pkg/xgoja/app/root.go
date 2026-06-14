package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	glazedcli "github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/go-go-golems/go-go-goja/pkg/jsverbs"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/sourcegraph"
	"github.com/spf13/cobra"
)

type Options struct {
	Providers       *providerapi.ProviderRegistry
	RuntimePlanJSON string
	Out             io.Writer
	EmbeddedJSVerbs fs.FS
	EmbeddedHelp    fs.FS
	EmbeddedAssets  fs.FS
	MiddlewaresFunc glazedcli.CobraMiddlewaresFunc
}

func NewRootCommand(opts Options) (*cobra.Command, error) {
	if opts.Providers == nil {
		return nil, fmt.Errorf("providers registry is required")
	}
	runtimePlan := &RuntimePlan{}
	if err := json.Unmarshal([]byte(opts.RuntimePlanJSON), runtimePlan); err != nil {
		return nil, fmt.Errorf("decode embedded xgoja runtime plan: %w", err)
	}
	host := NewHostWithOptions(opts.Providers, runtimePlan, HostOptions{EmbeddedJSVerbs: opts.EmbeddedJSVerbs, EmbeddedHelp: opts.EmbeddedHelp, EmbeddedAssets: opts.EmbeddedAssets, Out: opts.Out, MiddlewaresFunc: opts.MiddlewaresFunc})
	root := &cobra.Command{
		Use:   runtimePlan.Name,
		Short: "Generated xgoja binary",
	}
	if opts.Out != nil {
		root.SetOut(opts.Out)
	}
	host.AttachDefaultCommands(root)
	return root, nil
}

type evalCommand struct {
	*cmds.CommandDescription
	factory    *RuntimeFactory
	out        io.Writer
	sectionErr error
}

var _ cmds.BareCommand = (*evalCommand)(nil)

type evalSettings struct {
	Source string `glazed:"source"`
}

func newEvalCommand(factory *RuntimeFactory, runtimePlan *RuntimePlan, out io.Writer) cmds.Command {
	moduleSections, _, sectionErr := factory.sectionsForRuntime("eval")
	options := []cmds.CommandDescriptionOption{
		cmds.WithShort("Evaluate JavaScript in a generated xgoja runtime"),
		cmds.WithLong(`
Evaluate executes a JavaScript source string in a fresh xgoja runtime and prints
non-null, non-undefined results.

The generated runtime controls which provider modules are available through
require(). Provider modules may add Glazed sections; those sections are parsed
before evaluation and runtime initializers run before the JavaScript source.
`),
		cmds.WithArguments(
			fields.New("source", fields.TypeString,
				fields.WithRequired(true),
				fields.WithHelp("JavaScript source to evaluate")),
		),
	}
	if sectionErr == nil && len(moduleSections) > 0 {
		options = append(options, cmds.WithSections(moduleSections...))
	}
	command, _ := runtimePlan.commandByType("builtin.eval")
	return &evalCommand{
		CommandDescription: cmds.NewCommandDescription(commandName(command, "eval"), options...),
		factory:            factory,
		out:                out,
		sectionErr:         sectionErr,
	}
}

func (c *evalCommand) Run(ctx context.Context, vals *values.Values) error {
	if c.sectionErr != nil {
		return c.sectionErr
	}
	settings := evalSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return err
	}
	selectedModules, err := c.factory.selectedModuleDescriptors()
	if err != nil {
		return err
	}
	return evalSourceWithInitializers(ctx, c.factory, settings.Source, vals, selectedModules, c.out)
}

func evalSourceWithInitializers(ctx context.Context, factory *RuntimeFactory, source string, vals *values.Values, selectedModules []providerapi.ModuleDescriptor, out io.Writer) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if factory == nil {
		return fmt.Errorf("runtime factory is required")
	}
	rt, err := factory.NewRuntimeFromSections(ctx, vals)
	if err != nil {
		return err
	}
	defer func() { _ = rt.Close(context.Background()) }()
	if vals != nil && len(selectedModules) > 0 {
		if err := initRuntimeFromSections(ctx, vals, rt, selectedModules); err != nil {
			return err
		}
	}
	ret, err := rt.Owner.Call(ctx, "xgoja.eval", func(_ context.Context, vm *goja.Runtime) (any, error) {
		value, err := vm.RunString(source)
		if err != nil {
			return nil, err
		}
		if value != nil && !goja.IsUndefined(value) && !goja.IsNull(value) {
			return value.Export(), nil
		}
		return nil, nil
	})
	if err != nil {
		return err
	}
	if ret != nil {
		if out == nil {
			out = io.Discard
		}
		fmt.Fprintln(out, ret)
	}
	return nil
}

type modulesCommand struct {
	*cmds.CommandDescription
	providers *providerapi.ProviderRegistry
}

type selectedModulesCommand struct {
	*cmds.CommandDescription
	runtimePlan *RuntimePlan
}

var _ cmds.GlazeCommand = (*modulesCommand)(nil)
var _ cmds.GlazeCommand = (*selectedModulesCommand)(nil)

func newModulesCommand(providers *providerapi.ProviderRegistry, runtimePlan *RuntimePlan) cmds.Command {
	_ = runtimePlan
	return &modulesCommand{
		CommandDescription: cmds.NewCommandDescription("modules",
			cmds.WithShort("List provider modules compiled into this generated binary"),
			cmds.WithLong("List provider modules compiled into this generated xgoja binary. This is a provider catalog, not the selected require() aliases for this runtime. Use selected-modules for runtime aliases."),
		),
		providers: providers,
	}
}

func newSelectedModulesCommand(runtimePlan *RuntimePlan) cmds.Command {
	return &selectedModulesCommand{
		CommandDescription: cmds.NewCommandDescription("selected-modules",
			cmds.WithShort("List require() modules selected for this generated runtime"),
			cmds.WithLong("List the provider modules selected into this generated xgoja runtime, including the actual CommonJS require() alias and static module config."),
		),
		runtimePlan: runtimePlan,
	}
}

func (c *modulesCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
	_ = vals
	if c.providers == nil {
		return fmt.Errorf("providers registry is required")
	}
	for _, pkg := range c.providers.Packages() {
		names := make([]string, 0, len(pkg.Modules))
		for name := range pkg.Modules {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			providerRef := fmt.Sprintf("%s.%s", pkg.ID, name)
			if err := gp.AddRow(ctx, types.NewRow(
				types.MRP("package", pkg.ID),
				types.MRP("module", name),
				types.MRP("provider_ref", providerRef),
			)); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *selectedModulesCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
	_ = vals
	if c.runtimePlan == nil {
		return fmt.Errorf("runtime plan is required")
	}
	for _, mod := range c.runtimePlan.runtimeModules() {
		config := "{}"
		if len(mod.Config) > 0 {
			data, err := json.Marshal(mod.Config)
			if err != nil {
				return fmt.Errorf("marshal config for %s.%s: %w", mod.ProviderID(), mod.Name, err)
			}
			config = string(data)
		}
		if err := gp.AddRow(ctx, types.NewRow(
			types.MRP("package", mod.ProviderID()),
			types.MRP("module", mod.Name),
			types.MRP("alias", mod.Alias()),
			types.MRP("provider_ref", fmt.Sprintf("%s.%s", mod.ProviderID(), mod.Name)),
			types.MRP("config", config),
		)); err != nil {
			return err
		}
	}
	return nil
}

func newVerbsCommand(sourceRegistry *SourceRegistry, factory *RuntimeFactory, runtimePlan *RuntimePlan, jsverbsCommand CommandPlan, middlewaresFunc glazedcli.CobraMiddlewaresFunc) *cobra.Command {
	root := &cobra.Command{
		Use:   commandName(jsverbsCommand, "verbs"),
		Short: "Run configured JavaScript verb commands",
	}
	mounted, err := buildVerbCommands(sourceRegistry, factory, runtimePlan)
	if err != nil {
		root.RunE = func(cmd *cobra.Command, args []string) error { return err }
		return root
	}
	list := &cobra.Command{
		Use:   "sources",
		Short: "List configured JavaScript verb sources",
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, source := range sourceRegistry.ListSourcesByKind(providerapi.RuntimeSourceKindJSVerbs) {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\n", source.ID)
			}
			return nil
		},
	}
	root.AddCommand(list)
	if middlewaresFunc == nil {
		middlewaresFunc = glazedcli.CobraCommandDefaultMiddlewares
	}
	if err := glazedcli.AddCommandsToRootCommand(root, mounted, nil, glazedcli.WithParserConfig(glazedcli.CobraParserConfig{
		MiddlewaresFunc: middlewaresFunc,
	})); err != nil {
		root.RunE = func(cmd *cobra.Command, args []string) error { return err }
	}
	return root
}

func buildVerbCommands(sourceRegistry *SourceRegistry, factory *RuntimeFactory, runtimePlan *RuntimePlan) ([]cmds.Command, error) {
	moduleSections, selectedModules, err := factory.sectionsForRuntime("jsverbs")
	if err != nil {
		return nil, err
	}
	commands := []cmds.Command{}
	if sourceRegistry == nil {
		return nil, fmt.Errorf("source registry is required")
	}
	jsverbSources := sourceRegistry.JSVerbs()
	for _, source := range jsverbSources.ListJSVerbSources() {
		registry, err := sourceRegistry.scanJSVerbSource(source.ID, sourceGraphRuntimeAliases(moduleAliases(selectedModules)))
		if err != nil {
			return nil, err
		}
		if registry == nil {
			continue
		}
		for _, verb := range registry.Verbs() {
			verb := verb
			registry := registry
			cmd, err := registry.CommandForVerbWithInvoker(verb, func(ctx context.Context, _ *jsverbs.Registry, verb *jsverbs.VerbSpec, parsedValues *values.Values) (interface{}, error) {
				rt, err := factory.NewRuntimeFromSections(ctx, parsedValues, require.WithLoader(registry.RequireLoader()))
				if err != nil {
					return nil, err
				}
				defer func() { _ = rt.Close(context.Background()) }()
				if len(selectedModules) > 0 {
					if err := initRuntimeFromSections(ctx, parsedValues, rt, selectedModules); err != nil {
						return nil, err
					}
				}
				return registry.InvokeInRuntime(ctx, rt, verb, parsedValues)
			})
			if err != nil {
				return nil, err
			}
			if len(moduleSections) > 0 {
				if err := addSectionsToCommandDescription(cmd.Description(), moduleSections, "jsverbs runtime"); err != nil {
					return nil, err
				}
			}
			commands = append(commands, cmd)
		}
	}
	return commands, nil
}

func scanVerbSource(providers *providerapi.ProviderRegistry, embeddedJSVerbs fs.FS, source SourcePlan, runtimeAliases []string) (*jsverbs.Registry, error) {
	scanOptions := jsVerbScanOptions(source, runtimeAliases)
	sourceSet, err := sourceGraphSourceSet(providers, embeddedJSVerbs, source)
	if err != nil {
		return nil, err
	}
	if sourceSet == nil {
		return nil, nil
	}
	graph, err := sourcegraph.Build([]sourcegraph.SourceSet{*sourceSet}, sourcegraph.Options{RuntimeModuleAliases: runtimeAliases})
	if err != nil {
		return nil, fmt.Errorf("build source graph for jsverb source %s: %w", source.ID, err)
	}
	if err := graph.ResolveImports(readSourceGraphFile); err != nil {
		return nil, fmt.Errorf("resolve imports for jsverb source %s: %w", source.ID, err)
	}
	files, err := jsverbSourceFilesFromGraph(graph, source.ID)
	if err != nil {
		return nil, fmt.Errorf("read jsverb source %s: %w", source.ID, err)
	}
	registry, err := jsverbs.ScanSources(files, scanOptions)
	if err != nil {
		return nil, fmt.Errorf("scan jsverb source %s: %w", source.ID, err)
	}
	return registry, nil
}

func sourceGraphSourceSet(providers *providerapi.ProviderRegistry, embeddedJSVerbs fs.FS, source SourcePlan) (*sourcegraph.SourceSet, error) {
	set := sourcegraph.SourceSet{
		ID:         source.ID,
		Kind:       sourcegraph.SourceKindJSVerbs,
		Include:    append([]string(nil), source.Include...),
		Exclude:    append([]string(nil), source.Exclude...),
		Extensions: sourceGraphExtensions(source),
	}
	if source.ProviderID() != "" || source.Source != "" {
		if providers == nil {
			return nil, fmt.Errorf("scan jsverb source %s: providers registry is required", source.ID)
		}
		providerSource, ok := providers.ResolveVerbSource(source.ProviderID(), source.Source)
		if !ok {
			return nil, fmt.Errorf("scan jsverb source %s: unknown provider verb source %s.%s", source.ID, source.ProviderID(), source.Source)
		}
		if providerSource.FS == nil {
			return nil, fmt.Errorf("scan jsverb source %s: provider verb source %s.%s has no filesystem", source.ID, source.ProviderID(), source.Source)
		}
		set.Origin = sourcegraph.Origin{Kind: sourcegraph.OriginProvider, FS: providerSource.FS, Root: providerSource.Root, Provider: source.ProviderID(), Source: source.Source}
		return &set, nil
	}
	if source.Path == "" {
		return nil, nil
	}
	if source.Embed {
		if embeddedJSVerbs == nil {
			return nil, fmt.Errorf("scan jsverb source %s: embedded jsverbs filesystem is not configured", source.ID)
		}
		set.Origin = sourcegraph.Origin{Kind: sourcegraph.OriginEmbedded, FS: embeddedJSVerbs, Root: source.Path}
		return &set, nil
	}
	set.Origin = sourcegraph.Origin{Kind: sourcegraph.OriginDisk, Dir: source.Path}
	return &set, nil
}

func sourceGraphRuntimeAliases(selectedAliases []string) []string {
	return appendUniqueStrings(nil, selectedAliases...)
}

func allProviderRuntimeAliases(providers *providerapi.ProviderRegistry) []string {
	if providers == nil {
		return nil
	}
	aliases := []string{}
	for _, pkg := range providers.Packages() {
		for name, module := range pkg.Modules {
			aliases = append(aliases, name)
			if module.DefaultAs != "" {
				aliases = append(aliases, module.DefaultAs)
			}
		}
	}
	return appendUniqueStrings(nil, aliases...)
}

func sourceGraphExtensions(source SourcePlan) []string {
	if len(source.Extensions) > 0 {
		return append([]string(nil), source.Extensions...)
	}
	options := jsverbs.DefaultScanOptions()
	extensions := append([]string(nil), options.Extensions...)
	if source.TypeScript != nil {
		extensions = appendUniqueStrings(extensions, ".ts", ".tsx", ".mts", ".cts")
	}
	return extensions
}

func jsverbSourceFilesFromGraph(graph *sourcegraph.Graph, sourceSetID string) ([]jsverbs.SourceFile, error) {
	files := graph.FilesForSourceSet(sourceSetID)
	out := make([]jsverbs.SourceFile, 0, len(files))
	for _, file := range files {
		data, err := readSourceGraphFile(file)
		if err != nil {
			return nil, err
		}
		rootFS, err := sourceGraphRootFS(file.Origin)
		if err != nil {
			return nil, err
		}
		out = append(out, jsverbs.SourceFile{
			Path:       file.Path,
			AbsPath:    file.AbsPath,
			ResolveDir: sourceGraphResolveDir(file),
			RootFS:     rootFS,
			Source:     data,
		})
	}
	return out, nil
}

func readSourceGraphFile(file sourcegraph.File) ([]byte, error) {
	if file.AbsPath != "" {
		return os.ReadFile(file.AbsPath)
	}
	root := strings.Trim(strings.TrimSpace(file.Origin.Root), "/")
	if root == "" || root == "." {
		return fs.ReadFile(file.Origin.FS, file.Path)
	}
	return fs.ReadFile(file.Origin.FS, filepath.ToSlash(filepath.Join(root, file.Path)))
}

func sourceGraphRootFS(origin sourcegraph.Origin) (fs.FS, error) {
	if origin.FS == nil {
		return nil, nil
	}
	root := strings.Trim(strings.TrimSpace(origin.Root), "/")
	if root == "" || root == "." {
		return origin.FS, nil
	}
	return fs.Sub(origin.FS, root)
}

func sourceGraphResolveDir(file sourcegraph.File) string {
	if file.AbsPath == "" {
		return ""
	}
	return filepath.Dir(file.AbsPath)
}

func jsVerbScanOptions(source SourcePlan, runtimeAliases []string) jsverbs.ScanOptions {
	options := jsverbs.DefaultScanOptions()
	if len(source.Extensions) > 0 {
		options.Extensions = append([]string(nil), source.Extensions...)
	}
	options.Include = append([]string(nil), source.Include...)
	options.Exclude = append([]string(nil), source.Exclude...)
	applyTypeScriptScanOptions(source, &options, runtimeAliases)
	return options
}

func commandName(command CommandPlan, fallback string) string {
	if command.Name != "" {
		return command.Name
	}
	return fallback
}

func commandMount(command CommandPlan) string {
	switch strings.ToLower(strings.TrimSpace(command.Mount)) {
	case "root", "/", ".":
		return "root"
	default:
		return ""
	}
}
