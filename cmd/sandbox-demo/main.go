package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/go-go-golems/glazed/pkg/cmds/logging"
	"github.com/spf13/cobra"

	"github.com/go-go-golems/go-go-goja/engine"
	sandbox "github.com/go-go-golems/go-go-goja/pkg/sandbox"
)

func main() {
	var (
		scriptPath  string
		commandName string
		eventName   string
		argsJSON    string
	)

	root := &cobra.Command{
		Use:   "sandbox-demo",
		Short: "Smoke test the go-go-goja sandbox module",
		RunE: func(cmd *cobra.Command, args []string) error {
			if scriptPath == "" {
				return fmt.Errorf("--script is required")
			}
			return runDemo(cmd.Context(), scriptPath, commandName, eventName, argsJSON)
		},
	}
	root.PersistentFlags().StringVar(&scriptPath, "script", "", "Path to a CommonJS bot script")
	root.Flags().StringVar(&commandName, "command", "", "Command name to dispatch")
	root.Flags().StringVar(&eventName, "event", "", "Event name to dispatch")
	root.Flags().StringVar(&argsJSON, "args", "{}", "JSON object passed as ctx.args")
	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		return logging.InitLoggerFromCobra(cmd)
	}
	if err := logging.AddLoggingSectionToRootCommand(root, "sandbox-demo"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func runDemo(ctx context.Context, scriptPath, commandName, eventName, argsJSON string) error {
	absScript, err := filepath.Abs(scriptPath)
	if err != nil {
		return fmt.Errorf("resolve script path: %w", err)
	}

	factory, err := engine.NewBuilder(
		engine.WithModuleRootsFromScript(absScript, engine.DefaultModuleRootsOptions()),
	).
		WithModules(engine.DefaultRegistryModules()).
		WithRuntimeModuleRegistrars(sandbox.NewRegistrar(sandbox.Config{})).
		Build()
	if err != nil {
		return fmt.Errorf("build engine factory: %w", err)
	}

	rt, err := factory.NewRuntime(ctx)
	if err != nil {
		return fmt.Errorf("create runtime: %w", err)
	}
	defer func() { _ = rt.Close(ctx) }()

	value, err := rt.Require.Require(absScript)
	if err != nil {
		return fmt.Errorf("load script: %w", err)
	}

	handle, err := sandbox.CompileBot(rt.VM, value)
	if err != nil {
		return fmt.Errorf("compile bot: %w", err)
	}

	if commandName == "" && eventName == "" {
		desc, err := handle.Describe(ctx)
		if err != nil {
			return fmt.Errorf("describe bot: %w", err)
		}
		payload, _ := json.MarshalIndent(desc, "", "  ")
		fmt.Println(string(payload))
		return nil
	}

	request := sandbox.DispatchRequest{Name: commandName}
	if eventName != "" {
		request.Name = eventName
	}
	request.Args = map[string]any{}
	if argsJSON != "" {
		if err := json.Unmarshal([]byte(argsJSON), &request.Args); err != nil {
			return fmt.Errorf("parse --args json: %w", err)
		}
	}

	var (
		replies []string
		mu      sync.Mutex
	)
	request.Reply = func(_ context.Context, value any) error {
		mu.Lock()
		defer mu.Unlock()
		replies = append(replies, fmt.Sprint(value))
		return nil
	}

	if commandName != "" {
		result, err := handle.DispatchCommand(ctx, request)
		if err != nil {
			return fmt.Errorf("dispatch command: %w", err)
		}
		fmt.Printf("result: %v\n", result)
	} else {
		result, err := handle.DispatchEvent(ctx, request)
		if err != nil {
			return fmt.Errorf("dispatch event: %w", err)
		}
		fmt.Printf("event result: %v\n", result)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(replies) > 0 {
		fmt.Printf("replies: %v\n", replies)
	}
	return nil
}
