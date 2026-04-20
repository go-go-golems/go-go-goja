package botcli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	root := &cobra.Command{
		Use:   "bots",
		Short: "List, run, and inspect JavaScript bot verbs",
	}
	root.PersistentFlags().StringArray(BotRepositoryFlag, nil, "JavaScript repository directory to scan for bot verbs (repeatable)")
	root.AddCommand(newListCommand())
	root.AddCommand(newRunCommand())
	root.AddCommand(newHelpCommand())
	return root
}

func newListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List discovered bot verbs",
		RunE: func(cmd *cobra.Command, _ []string) error {
			bootstrap, err := DiscoverBootstrapFromCommand(cmd)
			if err != nil {
				return err
			}
			repositories, err := ScanRepositories(bootstrap)
			if err != nil {
				return err
			}
			bots, err := CollectDiscoveredBots(repositories)
			if err != nil {
				return err
			}
			lines := make([]string, 0, len(bots))
			for _, bot := range bots {
				lines = append(lines, fmt.Sprintf("%s\t%s", bot.FullPath(), bot.SourceLabel()))
			}
			sort.Strings(lines)
			if len(lines) == 0 {
				return nil
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), strings.Join(lines, "\n"))
			return err
		},
	}
}

func newRunCommand() *cobra.Command {
	return &cobra.Command{
		Use:                "run <verb>",
		Short:              "Run one discovered bot verb",
		DisableFlagParsing: true,
		Args:               cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			bootstrap, remainingArgs, err := DiscoverBootstrapFromCommandAndArgs(cmd, args)
			if err != nil {
				return err
			}
			repositories, err := ScanRepositories(bootstrap)
			if err != nil {
				return err
			}
			bots, err := CollectDiscoveredBots(repositories)
			if err != nil {
				return err
			}
			bot, verbArgs, err := ResolveBotFromArgs(remainingArgs, bots)
			if err != nil {
				return err
			}
			runtimeCommand, err := BuildRuntimeCommand(bot)
			if err != nil {
				return err
			}
			adoptIO(cmd, runtimeCommand)
			setRuntimeCommandUse(runtimeCommand, "go-go-goja bots run", bot.FullPath())
			runtimeCommand.SetArgs(verbArgs)
			return runtimeCommand.ExecuteContext(cmd.Context())
		},
	}
}

func newHelpCommand() *cobra.Command {
	return &cobra.Command{
		Use:                "help <verb>",
		Short:              "Show help for one discovered bot verb",
		DisableFlagParsing: true,
		Args:               cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			bootstrap, remainingArgs, err := DiscoverBootstrapFromCommandAndArgs(cmd, args)
			if err != nil {
				return err
			}
			repositories, err := ScanRepositories(bootstrap)
			if err != nil {
				return err
			}
			bots, err := CollectDiscoveredBots(repositories)
			if err != nil {
				return err
			}
			selector := strings.TrimSpace(strings.Join(remainingArgs, " "))
			if selector == "" {
				return fmt.Errorf("bot selector is empty")
			}
			bot, err := ResolveBot(selector, bots)
			if err != nil {
				return err
			}
			runtimeCommand, err := BuildRuntimeCommand(bot)
			if err != nil {
				return err
			}
			adoptIO(cmd, runtimeCommand)
			setRuntimeCommandUse(runtimeCommand, "go-go-goja bots run", bot.FullPath())
			return runtimeCommand.Help()
		},
	}
}
