package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	glazedcli "github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
	"github.com/spf13/cobra"
)

func (h *Host) AttachCommandProviders(root *cobra.Command) {
	if root == nil || h == nil || h.Spec == nil || h.Providers == nil {
		return
	}
	for _, instance := range h.Spec.CommandProviders {
		provider, ok := h.Providers.ResolveCommandSetProvider(instance.Package, instance.Name)
		mount := strings.TrimSpace(instance.Mount)
		if mount == "" {
			mount = provider.DefaultMount
		}
		if !ok {
			root.AddCommand(commandErrorStub(commandProviderUse(instance, mount), "Attach custom xgoja command provider", fmt.Errorf("unknown command provider %s.%s", instance.Package, instance.Name)))
			continue
		}
		set, err := h.newCommandSet(instance, provider, mount)
		if err != nil {
			root.AddCommand(commandErrorStub(commandProviderUse(instance, mount), "Attach custom xgoja command provider", err))
			continue
		}
		commands := applyMountToCommands(set.Commands, mount)
		parserConfig := glazedcli.CobraParserConfig{
			ShortHelpSections: []string{schema.DefaultSlug},
			MiddlewaresFunc:   glazedcli.CobraCommandDefaultMiddlewares,
		}
		if set.ParserConfig != nil {
			parserConfig = *set.ParserConfig
		}
		if err := glazedcli.AddCommandsToRootCommand(root, commands, nil, glazedcli.WithParserConfig(parserConfig)); err != nil {
			root.AddCommand(commandErrorStub(commandProviderUse(instance, mount), "Attach custom xgoja command provider", err))
		}
	}
}

func (h *Host) newCommandSet(instance CommandProviderInstance, provider providerapi.CommandSetProvider, mount string) (*providerapi.CommandSet, error) {
	config, err := json.Marshal(instance.Config)
	if err != nil {
		return nil, fmt.Errorf("marshal command provider config %s: %w", instance.ID, err)
	}
	profile := h.runtimeProfileForCommandProvider(instance)
	selected, err := h.selectedModulesForCommandProvider(instance)
	if err != nil {
		return nil, err
	}
	set, err := provider.New(providerapi.CommandSetContext{
		Context:         context.Background(),
		PackageID:       instance.Package,
		Name:            instance.Name,
		Mount:           mount,
		RuntimeProfile:  profile,
		Config:          config,
		Providers:       h.Providers,
		RuntimeFactory:  h.Factory,
		SelectedModules: selected,
	})
	if err != nil {
		return nil, fmt.Errorf("create command set %s.%s: %w", instance.Package, instance.Name, err)
	}
	if set == nil {
		return nil, fmt.Errorf("command provider %s.%s returned nil command set", instance.Package, instance.Name)
	}
	return set, nil
}

func (h *Host) runtimeProfileForCommandProvider(instance CommandProviderInstance) string {
	profile := strings.TrimSpace(instance.RuntimeProfile)
	if profile == "" {
		profile = firstRuntime(h.Spec)
	}
	return profile
}

func (h *Host) selectedModulesForCommandProvider(instance CommandProviderInstance) ([]providerapi.ModuleDescriptor, error) {
	profile := h.runtimeProfileForCommandProvider(instance)
	if profile == "" {
		return nil, nil
	}
	descriptors, err := h.Factory.selectedModuleDescriptors(profile)
	if err != nil {
		return nil, err
	}
	if len(instance.Modules) == 0 {
		return descriptors, nil
	}
	wanted := map[string]struct{}{}
	for _, module := range instance.Modules {
		module = strings.TrimSpace(module)
		if module != "" {
			wanted[module] = struct{}{}
		}
	}
	filtered := make([]providerapi.ModuleDescriptor, 0, len(descriptors))
	for _, descriptor := range descriptors {
		if _, ok := wanted[descriptor.PackageID+"."+descriptor.ModuleID]; ok {
			filtered = append(filtered, descriptor)
			continue
		}
		if _, ok := wanted[descriptor.As]; ok {
			filtered = append(filtered, descriptor)
		}
	}
	return filtered, nil
}

func applyMountToCommands(commands []cmds.Command, mount string) []cmds.Command {
	mount = strings.TrimSpace(mount)
	if mount == "" {
		return commands
	}
	mounted := make([]cmds.Command, 0, len(commands))
	for _, command := range commands {
		mounted = append(mounted, commandWithMount(command, mount))
	}
	return mounted
}

func commandWithMount(command cmds.Command, mount string) cmds.Command {
	if command == nil || command.Description() == nil {
		return command
	}
	desc := command.Description().Clone(true)
	if len(desc.Parents) == 0 || desc.Parents[0] != mount {
		desc.Parents = append([]string{mount}, desc.Parents...)
	}
	base := mountedCommandBase{command: command, description: desc}
	if _, ok := command.(cmds.GlazeCommand); ok {
		return mountedGlazeCommand{mountedCommandBase: base}
	}
	if _, ok := command.(cmds.WriterCommand); ok {
		return mountedWriterCommand{mountedCommandBase: base}
	}
	if _, ok := command.(cmds.BareCommand); ok {
		return mountedBareCommand{mountedCommandBase: base}
	}
	return mountedCommand{mountedCommandBase: base}
}

type mountedCommandBase struct {
	command     cmds.Command
	description *cmds.CommandDescription
}

func (c mountedCommandBase) Description() *cmds.CommandDescription {
	return c.description
}

func (c mountedCommandBase) ToYAML(w io.Writer) error {
	return c.description.ToYAML(w)
}

type mountedCommand struct {
	mountedCommandBase
}

type mountedBareCommand struct {
	mountedCommandBase
}

func (c mountedBareCommand) Run(ctx context.Context, vals *values.Values) error {
	return c.command.(cmds.BareCommand).Run(ctx, vals)
}

type mountedWriterCommand struct {
	mountedCommandBase
}

func (c mountedWriterCommand) RunIntoWriter(ctx context.Context, vals *values.Values, w io.Writer) error {
	return c.command.(cmds.WriterCommand).RunIntoWriter(ctx, vals, w)
}

type mountedGlazeCommand struct {
	mountedCommandBase
}

func (c mountedGlazeCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
	return c.command.(cmds.GlazeCommand).RunIntoGlazeProcessor(ctx, vals, gp)
}

func commandProviderUse(instance CommandProviderInstance, mount string) string {
	if strings.TrimSpace(mount) != "" {
		return strings.TrimSpace(mount)
	}
	if strings.TrimSpace(instance.Name) != "" {
		return strings.TrimSpace(instance.Name)
	}
	if strings.TrimSpace(instance.ID) != "" {
		return strings.TrimSpace(instance.ID)
	}
	return "command-provider"
}
