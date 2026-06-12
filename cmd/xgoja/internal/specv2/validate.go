package specv2

import (
	"fmt"
	"strings"

	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/buildspec"
)

type ValidationError = buildspec.ValidationError
type Report = buildspec.Report

func Validate(cfg *Config) *Report {
	report := &Report{}
	if cfg == nil {
		report.AddError("config", "", "Config is nil")
		return report
	}
	if strings.TrimSpace(cfg.Schema) != Schema {
		report.AddError("schema", "schema", fmt.Sprintf("schema must be %q", Schema))
	} else {
		report.AddOK("schema", "schema", Schema)
	}
	if strings.TrimSpace(cfg.Name) == "" {
		report.AddError("name", "name", "name is required")
	} else {
		report.AddOK("name", "name", cfg.Name)
	}
	validateWorkspace(report, cfg.Workspace)
	providers := validateProviders(report, cfg.Providers)
	runtimeAliases := validateRuntime(report, cfg.Runtime, providers)
	sources := validateSources(report, cfg.Sources, providers)
	validateCommands(report, cfg.Commands, providers, sources, runtimeAliases)
	validateArtifacts(report, cfg.Artifacts, sources)
	return report
}

func validateWorkspace(report *Report, workspace WorkspaceSpec) {
	mode := strings.TrimSpace(workspace.Mode)
	switch mode {
	case "off", "auto", "path":
		report.AddOK("workspace-mode", "workspace.mode", mode)
	default:
		report.AddError("workspace-mode", "workspace.mode", fmt.Sprintf("unsupported workspace mode %q", workspace.Mode))
	}
	if mode == "path" && strings.TrimSpace(workspace.File) == "" {
		report.AddError("workspace-file", "workspace.file", "workspace.file is required when workspace.mode is path")
	}
}

func validateProviders(report *Report, providers []ProviderSpec) map[string]ProviderSpec {
	ids := map[string]ProviderSpec{}
	for i, provider := range providers {
		path := fmt.Sprintf("providers[%d]", i)
		id := strings.TrimSpace(provider.ID)
		if id == "" {
			report.AddError("provider-id", path+".id", "provider id is required")
		} else if _, ok := ids[id]; ok {
			report.AddError("provider-id", path+".id", fmt.Sprintf("duplicate provider id %q", id))
		} else {
			ids[id] = provider
			report.AddOK("provider-id", path+".id", id)
		}
		if strings.TrimSpace(provider.Import) == "" {
			report.AddError("provider-import", path+".import", "provider import is required")
		} else {
			report.AddOK("provider-import", path+".import", provider.Import)
		}
		if strings.TrimSpace(provider.Register) == "" {
			report.AddError("provider-register", path+".register", "provider register function is required")
		}
	}
	return ids
}

func validateRuntime(report *Report, runtime RuntimeSpec, providers map[string]ProviderSpec) map[string]RuntimeModuleSpec {
	aliases := map[string]RuntimeModuleSpec{}
	for i, module := range runtime.Modules {
		path := fmt.Sprintf("runtime.modules[%d]", i)
		provider := strings.TrimSpace(module.Provider)
		if provider == "" {
			report.AddError("runtime-module-provider", path+".provider", "runtime module provider is required")
		} else if _, ok := providers[provider]; !ok {
			report.AddError("runtime-module-provider", path+".provider", fmt.Sprintf("unknown provider %q", provider))
		}
		if strings.TrimSpace(module.Name) == "" {
			report.AddError("runtime-module-name", path+".name", "runtime module name is required")
		}
		alias := strings.TrimSpace(module.Alias())
		if alias == "" {
			report.AddError("runtime-module-alias", path+".as", "runtime module alias is required")
		} else if _, ok := aliases[alias]; ok {
			report.AddError("runtime-module-alias", path+".as", fmt.Sprintf("duplicate runtime module alias %q", alias))
		} else {
			aliases[alias] = module
			report.AddOK("runtime-module-alias", path+".as", alias)
		}
	}
	return aliases
}

func validateSources(report *Report, sources []SourceSpec, providers map[string]ProviderSpec) map[string]SourceSpec {
	ids := map[string]SourceSpec{}
	for i, source := range sources {
		path := fmt.Sprintf("sources[%d]", i)
		id := strings.TrimSpace(source.ID)
		if id == "" {
			report.AddError("source-id", path+".id", "source id is required")
		} else if _, ok := ids[id]; ok {
			report.AddError("source-id", path+".id", fmt.Sprintf("duplicate source id %q", id))
		} else {
			ids[id] = source
			report.AddOK("source-id", path+".id", id)
		}
		switch source.Kind {
		case SourceKindJSVerbs, SourceKindScript, SourceKindAssets, SourceKindHelp:
			report.AddOK("source-kind", path+".kind", string(source.Kind))
		default:
			report.AddError("source-kind", path+".kind", fmt.Sprintf("unsupported source kind %q", source.Kind))
		}
		validateSourceFrom(report, path+".from", source.From, providers)
		validateLanguageAndCompile(report, path, source)
	}
	return ids
}

func validateSourceFrom(report *Report, path string, from SourceFromSpec, providers map[string]ProviderSpec) {
	count := 0
	if strings.TrimSpace(from.Dir) != "" {
		count++
	}
	if from.Provider != nil {
		count++
		if strings.TrimSpace(from.Provider.Provider) == "" {
			report.AddError("source-provider", path+".provider.provider", "provider source provider is required")
		} else if _, ok := providers[from.Provider.Provider]; !ok {
			report.AddError("source-provider", path+".provider.provider", fmt.Sprintf("unknown provider %q", from.Provider.Provider))
		}
		if strings.TrimSpace(from.Provider.Source) == "" {
			report.AddError("source-provider-source", path+".provider.source", "provider source name is required")
		}
	}
	if from.Workspace != nil {
		count++
		if strings.TrimSpace(from.Workspace.Module) == "" {
			report.AddError("source-workspace-module", path+".workspace.module", "workspace module is required")
		}
		if strings.TrimSpace(from.Workspace.Path) == "" {
			report.AddError("source-workspace-path", path+".workspace.path", "workspace path is required")
		}
	}
	if count != 1 {
		report.AddError("source-from", path, "exactly one source origin is required")
	}
}

func validateLanguageAndCompile(report *Report, path string, source SourceSpec) {
	language := strings.TrimSpace(source.Language)
	if language != "" {
		switch language {
		case "javascript", "typescript":
			report.AddOK("source-language", path+".language", language)
		default:
			report.AddError("source-language", path+".language", fmt.Sprintf("unsupported language %q", language))
		}
	}
	if source.Compile == nil {
		return
	}
	switch strings.TrimSpace(source.Compile.Mode) {
	case "runtime", "build-time", "preserve":
		report.AddOK("source-compile-mode", path+".compile.mode", source.Compile.Mode)
	default:
		report.AddError("source-compile-mode", path+".compile.mode", fmt.Sprintf("unsupported compile mode %q", source.Compile.Mode))
	}
	if source.Compile.Check != nil && len(source.Compile.Check.Command) == 0 {
		report.AddError("source-compile-check", path+".compile.check.command", "compile check command cannot be empty when check is set")
	}
}

func validateCommands(report *Report, commands []CommandSurfaceSpec, providers map[string]ProviderSpec, sources map[string]SourceSpec, runtimeAliases map[string]RuntimeModuleSpec) {
	ids := map[string]CommandSurfaceSpec{}
	for i, command := range commands {
		path := fmt.Sprintf("commands[%d]", i)
		id := strings.TrimSpace(command.ID)
		if id == "" {
			report.AddError("command-id", path+".id", "command id is required")
		} else if _, ok := ids[id]; ok {
			report.AddError("command-id", path+".id", fmt.Sprintf("duplicate command id %q", id))
		} else {
			ids[id] = command
			report.AddOK("command-id", path+".id", id)
		}
		switch strings.TrimSpace(command.Type) {
		case "builtin.eval", "builtin.run", "builtin.repl", "builtin.jsverbs", "provider.command-set":
			report.AddOK("command-type", path+".type", command.Type)
		default:
			report.AddError("command-type", path+".type", fmt.Sprintf("unsupported command type %q", command.Type))
		}
		if command.Type == "provider.command-set" {
			if strings.TrimSpace(command.Provider) == "" {
				report.AddError("command-provider", path+".provider", "provider command set requires provider")
			} else if _, ok := providers[command.Provider]; !ok {
				report.AddError("command-provider", path+".provider", fmt.Sprintf("unknown provider %q", command.Provider))
			}
			if strings.TrimSpace(command.Name) == "" {
				report.AddError("command-name", path+".name", "provider command set requires name")
			}
		}
		for j, sourceID := range command.Sources {
			if _, ok := sources[sourceID]; !ok {
				report.AddError("command-source", fmt.Sprintf("%s.sources[%d]", path, j), fmt.Sprintf("unknown source %q", sourceID))
			}
		}
		for j, alias := range command.Modules {
			if _, ok := runtimeAliases[alias]; !ok {
				report.AddError("command-module", fmt.Sprintf("%s.modules[%d]", path, j), fmt.Sprintf("unknown runtime module alias %q", alias))
			}
		}
	}
}

func validateArtifacts(report *Report, artifacts []ArtifactSpec, sources map[string]SourceSpec) {
	ids := map[string]ArtifactSpec{}
	for i, artifact := range artifacts {
		path := fmt.Sprintf("artifacts[%d]", i)
		id := strings.TrimSpace(artifact.ID)
		if id == "" {
			report.AddError("artifact-id", path+".id", "artifact id is required")
		} else if _, ok := ids[id]; ok {
			report.AddError("artifact-id", path+".id", fmt.Sprintf("duplicate artifact id %q", id))
		} else {
			ids[id] = artifact
			report.AddOK("artifact-id", path+".id", id)
		}
		switch strings.TrimSpace(artifact.Type) {
		case "binary", "runtime-package", "dts", "embedded-assets", "adapter", "cobra", "source", "template":
			report.AddOK("artifact-type", path+".type", artifact.Type)
		default:
			report.AddError("artifact-type", path+".type", fmt.Sprintf("unsupported artifact type %q", artifact.Type))
		}
		for j, sourceID := range artifact.Sources {
			if _, ok := sources[sourceID]; !ok {
				report.AddError("artifact-source", fmt.Sprintf("%s.sources[%d]", path, j), fmt.Sprintf("unknown source %q", sourceID))
			}
		}
	}
}
