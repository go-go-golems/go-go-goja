package app

import (
	"fmt"

	"github.com/go-go-golems/glazed/pkg/cmds/logging"
	"github.com/go-go-golems/glazed/pkg/help"
	help_cmd "github.com/go-go-golems/glazed/pkg/help/cmd"
	xgojadoc "github.com/go-go-golems/go-go-goja/pkg/xgoja/doc"
	"github.com/spf13/cobra"
)

const rootFrameworkInstalledAnnotation = "xgoja/root-framework-installed"

func installRootFramework(root *cobra.Command, spec *Spec) error {
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
	if spec != nil && spec.Name != "" {
		appName = spec.Name
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
	help_cmd.SetupCobraRootCommand(helpSystem, root)
	root.Annotations[rootFrameworkInstalledAnnotation] = "true"
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
