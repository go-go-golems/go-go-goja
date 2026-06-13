package app

import (
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/go-go-golems/glazed/pkg/cmds/logging"
	"github.com/go-go-golems/glazed/pkg/help"
	help_cmd "github.com/go-go-golems/glazed/pkg/help/cmd"
	xgojadoc "github.com/go-go-golems/go-go-goja/pkg/xgoja/doc"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
	"github.com/spf13/cobra"
)

const rootFrameworkInstalledAnnotation = "xgoja/root-framework-installed"

type frameworkOptions struct {
	Providers      *providerapi.ProviderRegistry
	SourceRegistry *SourceRegistry
	EmbeddedHelp   fs.FS
}

func installRootFramework(root *cobra.Command, runtimePlan *RuntimePlan, opts frameworkOptions) error {
	if root == nil {
		return fmt.Errorf("root command is nil")
	}
	if root.Annotations == nil {
		root.Annotations = map[string]string{}
	}
	if root.Annotations[rootFrameworkInstalledAnnotation] == "true" {
		return nil
	}
	appName := "xgoja"
	if runtimePlan != nil {
		if strings.TrimSpace(runtimePlan.AppName()) != "" {
			appName = strings.TrimSpace(runtimePlan.AppName())
		} else if strings.TrimSpace(runtimePlan.Name) != "" {
			appName = strings.TrimSpace(runtimePlan.Name)
		}
	}
	if err := logging.AddLoggingSectionToRootCommand(root, appName); err != nil {
		return err
	}
	chainPersistentPreRun(root, func(cmd *cobra.Command, args []string) error {
		return logging.InitLoggerFromCobra(cmd)
	})
	helpSystem := help.NewHelpSystem()
	if err := xgojadoc.AddDocToHelpSystem(helpSystem); err != nil {
		return fmt.Errorf("load generated xgoja help docs: %w", err)
	}
	if err := loadConfiguredHelpSources(helpSystem, runtimePlan, opts); err != nil {
		return err
	}
	help_cmd.SetupCobraRootCommand(helpSystem, root)
	root.Annotations[rootFrameworkInstalledAnnotation] = "true"
	return nil
}

func loadConfiguredHelpSources(helpSystem *help.HelpSystem, runtimePlan *RuntimePlan, opts frameworkOptions) error {
	helpSources := runtimePlan.sourcesByKind(SourceKindHelp)
	if opts.SourceRegistry != nil {
		helpSources = sourcePlansFromDescriptors(opts.SourceRegistry.ListSourcesByKind(providerapi.RuntimeSourceKindHelp))
	}
	if helpSystem == nil || runtimePlan == nil || len(helpSources) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	for _, source := range helpSources {
		id := strings.TrimSpace(source.ID)
		if id == "" {
			return fmt.Errorf("help source id is required")
		}
		if _, ok := seen[id]; ok {
			return fmt.Errorf("duplicate help source %q", id)
		}
		seen[id] = struct{}{}

		hasProvider := strings.TrimSpace(source.ProviderID()) != "" || strings.TrimSpace(source.Source) != ""
		if hasProvider {
			if opts.Providers == nil {
				return fmt.Errorf("load help source %s: providers registry is required", id)
			}
			providerSource, ok := opts.Providers.ResolveHelpSource(source.ProviderID(), source.Source)
			if !ok {
				return fmt.Errorf("load help source %s: unknown provider help source %s.%s", id, source.ProviderID(), source.Source)
			}
			if err := helpSystem.LoadSectionsFromFS(providerSource.FS, providerSource.Root); err != nil {
				return fmt.Errorf("load provider help source %s (%s.%s): %w", id, source.ProviderID(), source.Source, err)
			}
			continue
		}

		path := strings.TrimSpace(source.Path)
		if path == "" {
			return fmt.Errorf("load help source %s: path or provider source is required", id)
		}
		if source.Embed {
			if opts.EmbeddedHelp == nil {
				return fmt.Errorf("load help source %s: embedded help filesystem is not configured", id)
			}
			if err := helpSystem.LoadSectionsFromFS(opts.EmbeddedHelp, path); err != nil {
				return fmt.Errorf("load embedded help source %s: %w", id, err)
			}
			continue
		}
		if err := helpSystem.LoadSectionsFromFS(os.DirFS(path), "."); err != nil {
			return fmt.Errorf("load filesystem help source %s: %w", id, err)
		}
	}
	return nil
}

func chainPersistentPreRun(root *cobra.Command, next func(*cobra.Command, []string) error) {
	existing := root.PersistentPreRunE
	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if existing != nil {
			if err := existing(cmd, args); err != nil {
				return err
			}
		}
		return next(cmd, args)
	}
}
