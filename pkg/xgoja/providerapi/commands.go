package providerapi

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/go-go-golems/go-go-goja/pkg/jsverbs"
)

// RuntimeFactory creates xgoja runtimes from the generated binary's single
// selected module set. Command set providers use it when they own
// domain-specific commands but still want those commands to run JavaScript with
// xgoja-selected modules.
type RuntimeFactory interface {
	NewRuntime(ctx context.Context, opts ...require.Option) (*engine.Runtime, error)
	NewRuntimeFromSections(ctx context.Context, vals *values.Values, opts ...require.Option) (*engine.Runtime, error)
}

type RuntimeFactoryWithHostServices interface {
	RuntimeFactory
	NewRuntimeFromSectionsWithHostServices(ctx context.Context, vals *values.Values, hostServices HostServices, opts ...require.Option) (*engine.Runtime, error)
}

// JSVerbSourceDescriptor describes one configured JavaScript verb source in the
// generated binary's runtime spec.
type JSVerbSourceDescriptor struct {
	ID         string
	Path       string
	Embed      bool
	Package    string
	Source     string
	Include    []string
	Exclude    []string
	Extensions []string
	TypeScript *TypeScriptDescriptor
}

// TypeScriptDescriptor describes optional TypeScript compilation settings for a
// JavaScript verb source.
type TypeScriptDescriptor struct {
	Enabled      bool
	Bundle       bool
	Target       string
	Format       string
	Platform     string
	Tsconfig     string
	Sourcemap    string
	External     []string
	Define       map[string]string
	CheckCommand []string
}

// JSVerbSourceSet lets command providers discover and scan the JavaScript verb
// sources configured for the generated binary. Providers should use this instead
// of reimplementing local, embedded, and provider-shipped source resolution.
type JSVerbSourceSet interface {
	ListJSVerbSources() []JSVerbSourceDescriptor
	ScanJSVerbSource(id string) (*jsverbs.Registry, error)
	ScanAllJSVerbSources() ([]*jsverbs.Registry, error)
}

// CommandSetContext is passed to command set providers when generated xgoja
// attaches custom commands.
type CommandSetContext struct {
	Context         context.Context
	PackageID       string
	Name            string
	Mount           string
	Config          json.RawMessage
	Host            HostServices
	Providers       *ProviderRegistry
	RuntimeFactory  RuntimeFactory
	SelectedModules []ModuleDescriptor
	Sources         SourceRegistry
	JSVerbs         JSVerbSourceSet
}

// CommandSetProvider registers a package-owned command factory.
type CommandSetProvider struct {
	Name          string
	DefaultMount  string
	Description   string
	ConfigSchema  json.RawMessage
	NewCommandSet func(CommandSetContext) (*CommandSet, error)
}

// CommandSet is the Glazed command bundle returned by a provider.
type CommandSet struct {
	Commands     []cmds.Command
	ParserConfig *cli.CobraParserConfig
}

func (p CommandSetProvider) applyToPackage(pkg *Package) error {
	return pkg.addCommandSetProvider(p)
}

func normalizeCommandSetProvider(provider CommandSetProvider) (CommandSetProvider, error) {
	name := strings.TrimSpace(provider.Name)
	if name == "" {
		return CommandSetProvider{}, fmt.Errorf("command set provider name is required")
	}
	if provider.NewCommandSet == nil {
		return CommandSetProvider{}, fmt.Errorf("command set provider %q factory is required", name)
	}
	provider.Name = name
	provider.DefaultMount = strings.TrimSpace(provider.DefaultMount)
	provider.Description = strings.TrimSpace(provider.Description)
	return provider, nil
}
