package http

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/go-go-golems/go-go-goja/pkg/jsverbs"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerutil"
)

func newServeCommandSet(ctx providerapi.CommandSetContext) (*providerapi.CommandSet, error) {
	if ctx.JSVerbs == nil {
		return nil, fmt.Errorf("http serve command requires configured jsverb sources")
	}
	if ctx.RuntimeFactory == nil {
		return nil, fmt.Errorf("http serve command requires runtime factory")
	}

	sections, err := providerutil.CollectGlazedConfigSections(ctx.SelectedModules, providerapi.SectionRequest{
		CommandProviderID: ctx.Name,
	}, nil)
	if err != nil {
		return nil, err
	}

	registries, err := ctx.JSVerbs.ScanAllJSVerbSources()
	if err != nil {
		return nil, err
	}
	commands := make([]cmds.Command, 0)
	for _, registry := range registries {
		if registry == nil {
			continue
		}
		for _, verb := range registry.Verbs() {
			verb := verb
			registry := registry
			cmd, err := registry.CommandForVerbWithInvoker(verb, func(runCtx context.Context, _ *jsverbs.Registry, verb *jsverbs.VerbSpec, parsedValues *values.Values) (interface{}, error) {
				return serveVerb(runCtx, ctx, registry, verb, parsedValues)
			})
			if err != nil {
				return nil, err
			}
			if len(sections) > 0 {
				if err := addSectionsToServeCommand(cmd.Description(), sections, "http serve runtime"); err != nil {
					return nil, err
				}
			}
			commands = append(commands, cmd)
		}
	}
	return &providerapi.CommandSet{Commands: commands}, nil
}

func serveVerb(ctx context.Context, commandCtx providerapi.CommandSetContext, registry *jsverbs.Registry, verb *jsverbs.VerbSpec, parsedValues *values.Values) (interface{}, error) {
	if registry == nil {
		return nil, fmt.Errorf("jsverb registry is nil")
	}
	if verb == nil {
		return nil, fmt.Errorf("jsverb is nil")
	}
	rt, err := commandCtx.RuntimeFactory.NewRuntimeFromSections(ctx, parsedValues, require.WithLoader(registry.RequireLoader()))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rt.Close(context.Background()) }()

	if len(commandCtx.SelectedModules) > 0 {
		if err := providerutil.InitRuntimeFromSections(ctx, parsedValues, runtimeHandle{rt: rt}, commandCtx.SelectedModules); err != nil {
			return nil, err
		}
	}
	if _, err := registry.InvokeInRuntime(ctx, rt, verb, parsedValues); err != nil {
		return nil, err
	}

	fmt.Fprintln(os.Stderr, "xgoja http serve: runtime is alive; press Ctrl-C to stop")
	return nil, waitForServeShutdown(ctx)
}

func waitForServeShutdown(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	signalCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-signalCtx.Done()
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

func addSectionsToServeCommand(desc *cmds.CommandDescription, sections []schema.Section, source string) error {
	if desc == nil {
		return fmt.Errorf("command description is nil")
	}
	seen := map[string]string{}
	if desc.Schema != nil {
		desc.Schema.ForEach(func(slug string, _ schema.Section) {
			seen[slug] = "command schema"
		})
	}
	collected := []schema.Section{}
	if err := providerutil.AppendUniqueSections(&collected, seen, sections, source); err != nil {
		return err
	}
	desc.SetSections(collected...)
	return nil
}

type runtimeHandle struct {
	rt *engine.Runtime
}

func (h runtimeHandle) EngineRuntime() *engine.Runtime { return h.rt }

func (h runtimeHandle) Close(ctx context.Context) error {
	if h.rt == nil {
		return nil
	}
	return h.rt.Close(ctx)
}
