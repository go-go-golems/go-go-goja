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
	"github.com/go-go-golems/go-go-goja/modules/glazehelp"
	"github.com/go-go-golems/go-go-goja/pkg/doc"
	"github.com/spf13/cobra"
)

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

		factory, err := engine.NewBuilder().
			WithModules(engine.DefaultRegistryModules()).
			Build()
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

		if debug {
			log.Printf("engine initialised, args=%v", args)
		}

		// If a script path is provided, run it once and exit.
		if len(args) > 0 {
			if _, err := rt.Require.Require(args[0]); err != nil {
				return fmt.Errorf("failed to run script: %v", err)
			}
			return nil
		}

		// Interactive loop.
		return runInteractiveLoop(rt.VM, debug)
	},
}

func runInteractiveLoop(vm *goja.Runtime, debug bool) error {
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
			fmt.Println("  :quit    exit the REPL")
			fmt.Println("\nFor comprehensive documentation, run:")
			fmt.Println("  repl help")
			fmt.Println("  repl help introduction")
			fmt.Println("  repl help creating-modules")
			fmt.Println("  repl help async-patterns")
			fmt.Println("  repl help repl-usage")
			fmt.Println("  repl help jsparse-framework-reference")
			fmt.Println("  repl help inspector-example-user-guide")
			fmt.Println("\nOtherwise any line is evaluated as JavaScript.")
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

	// Set up help system
	helpSystem := help.NewHelpSystem()
	if err := doc.AddDocToHelpSystem(helpSystem); err != nil {
		log.Printf("Warning: failed to load documentation: %v", err)
	}

	// Register help system for JavaScript access
	glazehelp.Register("default", helpSystem)

	// Setup enhanced help system for the complete application
	help_cmd.SetupCobraRootCommand(helpSystem, rootCmd)

	// Execute
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
