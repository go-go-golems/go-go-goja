package botcli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/dop251/goja_nodejs/require"
	glazedcli "github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/spf13/cobra"

	"github.com/go-go-golems/go-go-goja/engine"
	"github.com/go-go-golems/go-go-goja/pkg/jsverbs"
)

func BuildRuntimeCommand(bot DiscoveredBot) (*cobra.Command, error) {
	if err := bot.Validate(); err != nil {
		return nil, err
	}

	description, err := bot.Repository.Registry.CommandDescriptionForVerb(bot.Verb)
	if err != nil {
		return nil, err
	}

	cobraCommand := glazedcli.NewCobraCommandFromCommandDescription(description)
	cobraCommand.Annotations = map[string]string{}
	parser, err := glazedcli.NewCobraParserFromSections(description.Schema, &glazedcli.CobraParserConfig{
		SkipCommandSettingsSection: true,
	})
	if err != nil {
		return nil, err
	}
	if err := parser.AddToCobraCommand(cobraCommand); err != nil {
		return nil, err
	}

	cobraCommand.RunE = func(cmd *cobra.Command, args []string) error {
		parsedValues, err := parser.Parse(cmd, args)
		if err != nil {
			return err
		}
		ctx, cancel := context.WithCancel(cmd.Context())
		defer cancel()
		ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
		defer stop()

		result, err := InvokeBot(ctx, bot, parsedValues)
		if err != nil {
			if err == context.Canceled {
				return nil
			}
			return err
		}
		return PrintResult(cmd.OutOrStdout(), bot.Verb.OutputMode, result)
	}

	return cobraCommand, nil
}

func InvokeBot(ctx context.Context, bot DiscoveredBot, parsedValues *values.Values) (interface{}, error) {
	if err := bot.Validate(); err != nil {
		return nil, err
	}
	factory, err := engine.NewBuilder(bot.Repository.runtimeOptions()...).
		WithModules(engine.DefaultRegistryModules()).
		Build()
	if err != nil {
		return nil, err
	}
	runtime, err := factory.NewRuntime(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = runtime.Close(context.Background())
	}()
	return bot.Repository.Registry.InvokeInRuntime(ctx, runtime, bot.Verb, parsedValues)
}

func (r ScannedRepository) runtimeOptions() []engine.Option {
	options := []engine.Option{engine.WithRequireOptions(require.WithLoader(r.Registry.RequireLoader()))}
	folders := []string{r.Repository.RootDir, filepath.Join(r.Repository.RootDir, "node_modules")}
	parent := filepath.Dir(r.Repository.RootDir)
	if parent != r.Repository.RootDir {
		folders = append(folders, parent, filepath.Join(parent, "node_modules"))
	}
	return append(options, engine.WithRequireOptions(require.WithGlobalFolders(folders...)))
}

func PrintResult(w io.Writer, outputMode string, result interface{}) error {
	if w == nil || result == nil {
		return nil
	}
	switch outputMode {
	case jsverbs.OutputModeText:
		switch value := result.(type) {
		case nil:
			return nil
		case string:
			_, err := io.WriteString(w, value)
			if err != nil {
				return err
			}
			if !strings.HasSuffix(value, "\n") {
				_, err = io.WriteString(w, "\n")
			}
			return err
		case []byte:
			if _, err := w.Write(value); err != nil {
				return err
			}
			if len(value) == 0 || value[len(value)-1] != '\n' {
				_, err := io.WriteString(w, "\n")
				return err
			}
			return nil
		default:
			_, err := fmt.Fprintf(w, "%v\n", result)
			return err
		}
	default:
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(result)
	}
}

func adoptIO(source *cobra.Command, target *cobra.Command) {
	if source == nil || target == nil {
		return
	}
	target.SetOut(source.OutOrStdout())
	target.SetErr(source.ErrOrStderr())
}

func setRuntimeCommandUse(command *cobra.Command, prefix, selector string) {
	if command == nil {
		return
	}
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return
	}
	command.Use = strings.TrimSpace(prefix + " " + selector)
	command.Args = cobra.ArbitraryArgs
}
