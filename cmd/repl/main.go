package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/dop251/goja"
	"github.com/go-go-golems/glazed/pkg/help"
	help_cmd "github.com/go-go-golems/glazed/pkg/help/cmd"
	"github.com/go-go-golems/go-go-goja/engine"
	"github.com/go-go-golems/go-go-goja/pkg/doc"
	docaccessruntime "github.com/go-go-golems/go-go-goja/pkg/docaccess/runtime"
	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/host"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

var appHelpSystem *help.HelpSystem

var rootCmd = &cobra.Command{
	Use:   "repl [script.js]",
	Short: "Interactive JavaScript REPL with native Go modules",
	Long: `A JavaScript runtime environment powered by goja with native Go module support.

Run interactively to evaluate JavaScript expressions with access to all
registered native modules, or provide a script file to execute it once.

The runtime includes Node.js-style require(), console object, and automatic
type conversion between Go and JavaScript values.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If no files were given, just show usage.
		debug, _ := cmd.Flags().GetBool("debug")
		if debug {
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		} else {
			zerolog.SetGlobalLevel(zerolog.ErrorLevel)
		}
		pluginStatus, _ := cmd.Flags().GetBool("plugin-status")
		allowPluginModules, _ := cmd.Flags().GetStringSlice("allow-plugin-module")
		pluginDirs, _ := cmd.Flags().GetStringSlice("plugin-dir")
		pluginSetup := host.NewRuntimeSetup(pluginDirs, allowPluginModules)

		builder := pluginSetup.
			WithBuilder(engine.NewBuilder().
				WithModules(engine.DefaultRegistryModules()))
		if appHelpSystem != nil {
			builder = builder.WithRuntimeModuleRegistrars(docaccessruntime.NewRegistrar(docaccessruntime.Config{
				HelpSources: []docaccessruntime.HelpSource{{
					ID:      "default-help",
					Title:   "Default Help",
					Summary: "Embedded REPL help pages",
					System:  appHelpSystem,
				}},
			}))
		}
		factory, err := builder.Build()
		if err != nil {
			return fmt.Errorf("failed to build engine factory: %v", err)
		}
		rt, err := factory.NewRuntime(context.Background())
		if err != nil {
			return fmt.Errorf("failed to create runtime: %v", err)
		}
		defer func() {
			_ = rt.Close(context.Background())
		}()
		report := pluginSetup.Snapshot()

		if debug {
			log.Printf("engine initialised, args=%v", args)
		}
		if pluginStatus {
			printPluginReport(report)
			return nil
		}
		if summary := pluginStartupSummary(report); summary != "" {
			fmt.Println(summary)
		}

		// If a script path is provided, run it once and exit.
		if len(args) > 0 {
			if _, err := rt.Require.Require(args[0]); err != nil {
				return fmt.Errorf("failed to run script: %v", err)
			}
			return nil
		}

		// Interactive loop.
		return runInteractiveLoop(rt.VM, debug, report)
	},
}

func runInteractiveLoop(vm *goja.Runtime, debug bool, report host.LoadReport) error {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("goja> type JS code (:help for help)")

	for {
		fmt.Print("js> ")
		line, err := reader.ReadString('\n')
		if err != nil {
			if err.Error() == "EOF" {
				fmt.Println()
				return nil
			}
			return fmt.Errorf("reading stdin: %v", err)
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		switch line {
		case ":quit", ":exit":
			return nil
		case ":help":
			fmt.Println("REPL Commands:")
			fmt.Println("  :help    show this help")
			fmt.Println("  :plugins show plugin discovery and load details")
			fmt.Println("  :quit    exit the REPL")
			fmt.Println("\nFor comprehensive documentation, run:")
			fmt.Println("  repl help")
			fmt.Println("  repl help introduction")
			fmt.Println("  repl help creating-modules")
			fmt.Println("  repl help async-patterns")
			fmt.Println("  repl help repl-usage")
			fmt.Println("  repl help bun-bundling-playbook-goja")
			fmt.Println("  repl help typescript-declaration-generator")
			fmt.Println("  repl help goja-docs-module-guide")
			fmt.Println("  repl help goja-plugin-user-guide")
			fmt.Println("  repl help goja-plugin-developer-guide")
			fmt.Println("  repl help plugin-tutorial-build-install")
			fmt.Println("  repl help jsparse-framework-reference")
			fmt.Println("  repl help inspector-example-user-guide")
			fmt.Println("\nOtherwise any line is evaluated as JavaScript.")
			continue
		case ":plugins":
			printPluginReport(report)
			continue
		}

		val, err := vm.RunString(line)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			if debug {
				log.Printf("eval error: %v", err)
			}
			continue
		}

		// Print non-undefined results.
		if val != nil && !goja.IsUndefined(val) {
			fmt.Println(val)
		}
	}
}

func main() {
	// Set up flags
	rootCmd.Flags().Bool("debug", false, "enable verbose debug logs")
	rootCmd.Flags().StringSlice("allow-plugin-module", nil, "allow only the listed plugin module names (for example plugin:examples:greeter)")
	rootCmd.Flags().StringSlice("plugin-dir", nil, fmt.Sprintf("directory containing HashiCorp go-plugin module binaries (defaults to %s/... when omitted)", host.DefaultDiscoveryRoot()))
	rootCmd.Flags().Bool("plugin-status", false, "print plugin discovery/load status and exit")

	// Set up help system
	appHelpSystem = help.NewHelpSystem()
	if err := doc.AddDocToHelpSystem(appHelpSystem); err != nil {
		log.Printf("Warning: failed to load documentation: %v", err)
	}

	// Setup enhanced help system for the complete application
	help_cmd.SetupCobraRootCommand(appHelpSystem, rootCmd)

	// Execute
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func pluginStartupSummary(report host.LoadReport) string {
	if !report.HasActivity() {
		return ""
	}
	return "Plugins: " + report.Summary() + " (type :plugins for details)"
}

func printPluginReport(report host.LoadReport) {
	for _, line := range report.DetailLines() {
		fmt.Println(line)
	}
}
