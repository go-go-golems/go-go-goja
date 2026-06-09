package jsverbscli

//glazedclilint:file-ignore legacy jsverbs CLI uses Cobra flags pending Glazed field migration

import (
	"fmt"
	"sort"

	glazedcli "github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/go-go-goja/pkg/jsverbs"
	"github.com/spf13/cobra"

	"github.com/go-go-golems/go-go-goja/pkg/jsverbrepos"
)

type ScannedRepository struct {
	Repository jsverbrepos.Repository
	Registry   *jsverbs.Registry
}

type DiscoveredVerb struct {
	Repository ScannedRepository
	Verb       *jsverbs.VerbSpec
}

type InvokerFactory func(repo ScannedRepository, verb *jsverbs.VerbSpec) jsverbs.VerbInvoker

func NewLazyCommand() *cobra.Command {
	return &cobra.Command{
		Use:                "verbs",
		Short:              "Run repository-scanned JavaScript verbs",
		DisableFlagParsing: true,
		Args:               cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			bootstrap, remainingArgs, err := jsverbrepos.Discover(cmd.Context(), args)
			if err != nil {
				return err
			}
			resolved, err := NewCommand(bootstrap)
			if err != nil {
				return err
			}
			adoptHelpAndOutput(cmd, resolved)
			resolved.SetArgs(remainingArgs)
			return resolved.ExecuteContext(cmd.Context())
		},
	}
}

func NewCommand(bootstrap jsverbrepos.Bootstrap) (*cobra.Command, error) {
	settings := &RuntimeSettings{ReadOnly: true}
	return newCommandWithInvokerFactory(bootstrap, runtimeInvokerFactory(settings), settings)
}

func newCommandWithInvokerFactory(bootstrap jsverbrepos.Bootstrap, invokers InvokerFactory, settings *RuntimeSettings) (*cobra.Command, error) {
	root := &cobra.Command{
		Use:   "verbs",
		Short: "Run repository-scanned JavaScript verbs",
	}
	if settings != nil {
		root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
			if flag := cmd.Flag("db"); flag != nil {
				settings.DBPath = flag.Value.String()
			}
			if flag := cmd.Flag("readonly"); flag != nil {
				settings.ReadOnly = flag.Value.String() != "false"
			}
			if flag := cmd.Flag("allow-writes"); flag != nil {
				settings.AllowWrites = flag.Value.String() == "true"
			}
			return nil
		}
		root.PersistentFlags().StringVar(&settings.DBPath, "db", "", "SQLite database path exposed as require(\"database\") and require(\"db\")")
		root.PersistentFlags().BoolVar(&settings.ReadOnly, "readonly", true, "Open the JavaScript database API in read-only mode")
		root.PersistentFlags().BoolVar(&settings.AllowWrites, "allow-writes", false, "Allow db.exec writes when --readonly=false")
	}

	repositories, err := ScanRepositories(bootstrap)
	if err != nil {
		return nil, err
	}
	discovered, err := CollectDiscoveredVerbs(repositories)
	if err != nil {
		return nil, err
	}
	listCmd, err := newListCommand(discovered)
	if err != nil {
		return nil, err
	}
	root.AddCommand(listCmd)
	commands, err := buildCommands(discovered, invokers)
	if err != nil {
		return nil, err
	}
	if err := glazedcli.AddCommandsToRootCommand(root, commands, nil, glazedcli.WithParserConfig(glazedcli.CobraParserConfig{
		MiddlewaresFunc: glazedcli.CobraCommandDefaultMiddlewares,
	})); err != nil {
		return nil, err
	}
	return root, nil
}

func ScanRepositories(bootstrap jsverbrepos.Bootstrap) ([]ScannedRepository, error) {
	opts := jsverbs.DefaultScanOptions()

	ret := make([]ScannedRepository, 0, len(bootstrap.Repositories))
	for _, repo := range bootstrap.Repositories {
		var (
			registry *jsverbs.Registry
			err      error
		)
		if repo.Embedded {
			registry, err = jsverbs.ScanFS(repo.EmbeddedFS, repo.EmbeddedAt, opts)
		} else {
			registry, err = jsverbs.ScanDir(repo.RootDir, opts)
		}
		if err != nil {
			return nil, fmt.Errorf("scan repository %s: %w", repo.Name, err)
		}
		ret = append(ret, ScannedRepository{Repository: repo, Registry: registry})
	}
	return ret, nil
}

func CollectDiscoveredVerbs(repositories []ScannedRepository) ([]DiscoveredVerb, error) {
	seen := map[string]DiscoveredVerb{}
	ret := []DiscoveredVerb{}
	for _, repo := range repositories {
		if repo.Registry == nil {
			continue
		}
		for _, verb := range repo.Registry.Verbs() {
			key := verb.FullPath()
			candidate := DiscoveredVerb{Repository: repo, Verb: verb}
			if prev, ok := seen[key]; ok {
				return nil, fmt.Errorf("duplicate jsverb path %q from %s and %s", key, discoveredVerbSource(prev), discoveredVerbSource(candidate))
			}
			seen[key] = candidate
			ret = append(ret, candidate)
		}
	}
	sort.Slice(ret, func(i, j int) bool {
		return ret[i].Verb.FullPath() < ret[j].Verb.FullPath()
	})
	return ret, nil
}

func buildCommands(discovered []DiscoveredVerb, invokers InvokerFactory) ([]cmds.Command, error) {
	commands := make([]cmds.Command, 0, len(discovered))
	for _, discoveredVerb := range discovered {
		repo := discoveredVerb.Repository
		verb := discoveredVerb.Verb
		cmd, err := repo.Registry.CommandForVerbWithInvoker(verb, invokers(repo, verb))
		if err != nil {
			return nil, err
		}
		commands = append(commands, cmd)
	}
	return commands, nil
}

func discoveredVerbSource(verb DiscoveredVerb) string {
	if verb.Verb == nil || verb.Verb.File == nil {
		return verb.Repository.Repository.Name
	}
	if verb.Verb.File.AbsPath != "" {
		return fmt.Sprintf("%s (%s)", verb.Repository.Repository.Name, verb.Verb.File.AbsPath)
	}
	return fmt.Sprintf("%s (%s)", verb.Repository.Repository.Name, verb.Verb.File.RelPath)
}

func describeRepository(repo ScannedRepository) string {
	if repo.Repository.Embedded {
		return fmt.Sprintf("%s:%s", repo.Repository.Name, repo.Repository.EmbeddedAt)
	}
	return repo.Repository.RootDir
}

func adoptHelpAndOutput(source *cobra.Command, target *cobra.Command) {
	if source == nil || target == nil {
		return
	}
	target.SetOut(source.OutOrStdout())
	target.SetErr(source.ErrOrStderr())
	root := source.Root()
	if root == nil {
		return
	}
	target.SetHelpFunc(root.HelpFunc())
	if usageFunc := root.UsageFunc(); usageFunc != nil {
		target.SetUsageFunc(usageFunc)
	}
	target.SetHelpTemplate(root.HelpTemplate())
	target.SetUsageTemplate(root.UsageTemplate())
}
