package plan

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/specv2"
	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/workspace"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providergraph"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/sourcegraph"
)

// Plan is the v2 compiler output consumed by doctor/build/gen-dts cutover code.
// It keeps the individual subgraphs intact so early callers can adopt the plan
// incrementally while sharing one validation and resolution pass.
type Plan struct {
	Config         specv2.Config
	GoModules      *workspace.Plan
	ProviderGraph  *providergraph.Graph
	SourceGraph    *sourcegraph.Graph
	Commands       []CommandPlan
	Artifacts      []ArtifactPlan
	RuntimeAliases []string
}

type CommandPlan struct {
	Spec specv2.CommandSurfaceSpec
}

type ArtifactPlan struct {
	Spec specv2.ArtifactSpec
}

type Options struct {
	Config     specv2.Config
	Providers  *providerapi.ProviderRegistry
	StartDir   string
	CLIReplace map[string]string
}

func Compile(opts Options) (*Plan, error) {
	cfg := opts.Config
	if cfg.BaseDir == "" {
		cfg.BaseDir = opts.StartDir
	}
	if cfg.BaseDir == "" {
		cfg.BaseDir = "."
	}
	if report := specv2.Validate(&cfg); report.HasErrors() {
		return nil, fmt.Errorf("invalid xgoja/v2 spec")
	}
	providerGraph, err := buildProviderGraph(opts.Providers, cfg)
	if err != nil {
		return nil, err
	}
	goModules, err := buildGoModulePlan(cfg, opts.StartDir, opts.CLIReplace)
	if err != nil {
		return nil, err
	}
	sourceGraph, err := buildSourceGraph(cfg, opts.Providers, goModules, providerGraph.RuntimeModuleAliases())
	if err != nil {
		return nil, err
	}
	if err := sourceGraph.ResolveImports(func(file sourcegraph.File) ([]byte, error) {
		return readSourceGraphFile(file)
	}); err != nil {
		return nil, err
	}
	return &Plan{
		Config:         cfg,
		GoModules:      goModules,
		ProviderGraph:  providerGraph,
		SourceGraph:    sourceGraph,
		Commands:       commandPlans(cfg.Commands),
		Artifacts:      artifactPlans(cfg.Artifacts),
		RuntimeAliases: providerGraph.RuntimeModuleAliases(),
	}, nil
}

func buildProviderGraph(registry *providerapi.ProviderRegistry, cfg specv2.Config) (*providergraph.Graph, error) {
	providers := make([]providergraph.ProviderSelection, 0, len(cfg.Providers))
	for _, provider := range cfg.Providers {
		providers = append(providers, providergraph.ProviderSelection{ID: provider.ID})
	}
	modules := make([]providergraph.RuntimeModuleSelection, 0, len(cfg.Runtime.Modules))
	for _, module := range cfg.Runtime.Modules {
		modules = append(modules, providergraph.RuntimeModuleSelection{Provider: module.Provider, Name: module.Name, As: module.As})
	}
	commandSets := []providergraph.CommandSetSelection{}
	for _, command := range cfg.Commands {
		if strings.TrimSpace(command.Type) != "provider.command-set" {
			continue
		}
		commandSets = append(commandSets, providergraph.CommandSetSelection{ID: command.ID, Provider: command.Provider, Name: command.Name})
	}
	graph, err := providergraph.Build(registry, providergraph.Options{Providers: providers, Modules: modules, CommandSets: commandSets})
	if err != nil {
		return nil, fmt.Errorf("build provider graph: %w", err)
	}
	return graph, nil
}

func buildGoModulePlan(cfg specv2.Config, startDir string, cliReplace map[string]string) (*workspace.Plan, error) {
	requirements := []workspace.Requirement{}
	for _, provider := range cfg.Providers {
		requirements = append(requirements, workspace.Requirement{
			ModulePath:      modulePathFromImport(provider.Import),
			Version:         provider.Module.Version,
			ExplicitReplace: provider.Module.Replace,
			RequiredBy:      []string{"provider:" + provider.ID},
		})
	}
	for _, goImport := range cfg.Go.Imports {
		modulePath := goImport.Module
		if strings.TrimSpace(modulePath) == "" {
			modulePath = modulePathFromImport(goImport.Import)
		}
		requirements = append(requirements, workspace.Requirement{
			ModulePath: modulePath,
			Version:    goImport.Version,
			RequiredBy: []string{"go.import:" + goImport.Import},
		})
	}
	ws, err := workspace.Resolve(requirements, workspace.Options{
		Spec:       workspace.Spec{Mode: workspace.Mode(cfg.Workspace.Mode), File: cfg.Workspace.File},
		StartDir:   startDir,
		CLIReplace: cliReplace,
	})
	if err != nil {
		return nil, fmt.Errorf("resolve Go workspace: %w", err)
	}
	return ws, nil
}

func buildSourceGraph(cfg specv2.Config, registry *providerapi.ProviderRegistry, goModules *workspace.Plan, runtimeAliases []string) (*sourcegraph.Graph, error) {
	sources := make([]sourcegraph.SourceSet, 0, len(cfg.Sources))
	for _, source := range cfg.Sources {
		set, err := sourceSetFromSpec(cfg, registry, goModules, source)
		if err != nil {
			return nil, err
		}
		sources = append(sources, set)
	}
	graph, err := sourcegraph.Build(sources, sourcegraph.Options{RuntimeModuleAliases: runtimeAliases})
	if err != nil {
		return nil, fmt.Errorf("build source graph: %w", err)
	}
	return graph, nil
}

func modulePathFromImport(importPath string) string {
	importPath = strings.Trim(path.Clean(strings.TrimSpace(importPath)), "/")
	if importPath == "." {
		return ""
	}
	for _, marker := range []string{"/pkg/", "/cmd/", "/internal/"} {
		if idx := strings.Index(importPath, marker); idx >= 0 {
			return importPath[:idx]
		}
	}
	if strings.HasSuffix(importPath, "/xgoja") {
		return path.Dir(importPath)
	}
	return importPath
}

func sourceSetFromSpec(cfg specv2.Config, registry *providerapi.ProviderRegistry, goModules *workspace.Plan, source specv2.SourceSpec) (sourcegraph.SourceSet, error) {
	set := sourcegraph.SourceSet{
		ID:         source.ID,
		Kind:       sourceKind(source.Kind),
		Include:    append([]string(nil), source.Include...),
		Exclude:    append([]string(nil), source.Exclude...),
		Extensions: append([]string(nil), source.Extensions...),
		Language:   source.Language,
	}
	if source.From.Provider != nil {
		providerSource, ok := registry.ResolveVerbSource(source.From.Provider.Provider, source.From.Provider.Source)
		if !ok {
			return set, fmt.Errorf("source %s references unknown provider verb source %s.%s", source.ID, source.From.Provider.Provider, source.From.Provider.Source)
		}
		set.Origin = sourcegraph.Origin{Kind: sourcegraph.OriginProvider, FS: providerSource.FS, Root: providerSource.Root, Provider: source.From.Provider.Provider, Source: source.From.Provider.Source}
		return set, nil
	}
	if source.From.Workspace != nil {
		module, ok := findWorkspaceModule(goModules, source.From.Workspace.Module)
		if !ok || module.LocalDir == "" {
			return set, fmt.Errorf("source %s references unresolved workspace module %q", source.ID, source.From.Workspace.Module)
		}
		set.Origin = sourcegraph.Origin{Kind: sourcegraph.OriginDisk, Dir: filepath.Join(module.LocalDir, source.From.Workspace.Path)}
		return set, nil
	}
	if strings.TrimSpace(source.From.Dir) == "" {
		return set, fmt.Errorf("source %s has no source origin", source.ID)
	}
	set.Origin = sourcegraph.Origin{Kind: sourcegraph.OriginDisk, Dir: absMaybe(cfg.BaseDir, source.From.Dir)}
	return set, nil
}

func sourceKind(kind specv2.SourceKind) sourcegraph.SourceKind {
	switch kind {
	case specv2.SourceKindJSVerbs:
		return sourcegraph.SourceKindJSVerbs
	case specv2.SourceKindScript:
		return sourcegraph.SourceKindScript
	case specv2.SourceKindAssets:
		return sourcegraph.SourceKindAssets
	case specv2.SourceKindHelp:
		return sourcegraph.SourceKindHelp
	default:
		return sourcegraph.SourceKind(kind)
	}
}

func findWorkspaceModule(plan *workspace.Plan, modulePath string) (workspace.GoModulePlan, bool) {
	if plan == nil {
		return workspace.GoModulePlan{}, false
	}
	for _, module := range plan.Modules {
		if module.ModulePath == modulePath {
			return module, true
		}
	}
	return workspace.GoModulePlan{}, false
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

func commandPlans(commands []specv2.CommandSurfaceSpec) []CommandPlan {
	out := make([]CommandPlan, 0, len(commands))
	for _, command := range commands {
		out = append(out, CommandPlan{Spec: command})
	}
	return out
}

func artifactPlans(artifacts []specv2.ArtifactSpec) []ArtifactPlan {
	out := make([]ArtifactPlan, 0, len(artifacts))
	for _, artifact := range artifacts {
		out = append(out, ArtifactPlan{Spec: artifact})
	}
	return out
}

func absMaybe(baseDir, path string) string {
	path = strings.TrimSpace(path)
	if path == "" || filepath.IsAbs(path) || strings.TrimSpace(baseDir) == "" {
		return filepath.Clean(path)
	}
	return filepath.Clean(filepath.Join(baseDir, path))
}
