package botcli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/go-go-golems/go-go-goja/pkg/jsverbs"
)

func DiscoverBootstrapFromCommand(cmd *cobra.Command) (Bootstrap, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return Bootstrap{}, fmt.Errorf("resolve cwd: %w", err)
	}
	repositories, err := repositoriesFromCommand(cmd, cwd)
	if err != nil {
		return Bootstrap{}, err
	}
	return bootstrapFromRepositories(repositories)
}

func DiscoverBootstrapFromCommandAndArgs(cmd *cobra.Command, args []string) (Bootstrap, []string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return Bootstrap{}, nil, fmt.Errorf("resolve cwd: %w", err)
	}
	fromCommand, err := repositoriesFromCommand(cmd, cwd)
	if err != nil {
		return Bootstrap{}, nil, err
	}
	fromArgs, remainingArgs, err := repositoriesFromArgs(args, cwd)
	if err != nil {
		return Bootstrap{}, nil, err
	}
	repositories := append(append([]Repository{}, fromCommand...), fromArgs...)
	bootstrap, err := bootstrapFromRepositories(repositories)
	if err != nil {
		return Bootstrap{}, nil, err
	}
	return bootstrap, remainingArgs, nil
}

func bootstrapFromRepositories(repositories []Repository) (Bootstrap, error) {
	if len(repositories) == 0 {
		return Bootstrap{}, fmt.Errorf("at least one --%s must be provided", BotRepositoryFlag)
	}
	seen := map[string]struct{}{}
	ret := make([]Repository, 0, len(repositories))
	for _, repo := range repositories {
		identity := filepath.Clean(repo.RootDir)
		if _, ok := seen[identity]; ok {
			continue
		}
		seen[identity] = struct{}{}
		ret = append(ret, repo)
	}
	sort.Slice(ret, func(i, j int) bool {
		return ret[i].RootDir < ret[j].RootDir
	})
	return Bootstrap{Repositories: ret}, nil
}

func repositoriesFromCommand(cmd *cobra.Command, cwd string) ([]Repository, error) {
	if cmd == nil {
		return nil, nil
	}
	paths := []string{}
	if cmd.Flags().Lookup(BotRepositoryFlag) != nil {
		flagPaths, err := cmd.Flags().GetStringArray(BotRepositoryFlag)
		if err != nil {
			return nil, err
		}
		paths = append(paths, flagPaths...)
	}
	if cmd.InheritedFlags().Lookup(BotRepositoryFlag) != nil {
		flagPaths, err := cmd.InheritedFlags().GetStringArray(BotRepositoryFlag)
		if err != nil {
			return nil, err
		}
		paths = append(paths, flagPaths...)
	}
	return repositoriesFromCLIPaths(paths, cwd)
}

func repositoriesFromArgs(args []string, cwd string) ([]Repository, []string, error) {
	paths := []string{}
	remaining := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--"+BotRepositoryFlag:
			if i+1 >= len(args) {
				return nil, nil, fmt.Errorf("missing value for --%s", BotRepositoryFlag)
			}
			paths = append(paths, args[i+1])
			i++
		case strings.HasPrefix(arg, "--"+BotRepositoryFlag+"="):
			paths = append(paths, strings.TrimPrefix(arg, "--"+BotRepositoryFlag+"="))
		default:
			remaining = append(remaining, arg)
		}
	}
	repositories, err := repositoriesFromCLIPaths(paths, cwd)
	if err != nil {
		return nil, nil, err
	}
	return repositories, remaining, nil
}

func repositoriesFromCLIPaths(paths []string, cwd string) ([]Repository, error) {
	ret := []Repository{}
	for _, raw := range paths {
		normalized, err := normalizeFilesystemRepositoryPath(raw, cwd)
		if err != nil {
			return nil, fmt.Errorf("CLI repository %q: %w", raw, err)
		}
		ret = append(ret, Repository{
			Name:      filepath.Base(normalized),
			Source:    "cli",
			SourceRef: "--" + BotRepositoryFlag,
			RootDir:   normalized,
		})
	}
	return ret, nil
}

func normalizeFilesystemRepositoryPath(path string, baseDir string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("repository path is empty")
	}
	if strings.HasPrefix(path, "~/") || path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		path = filepath.Join(home, strings.TrimPrefix(path, "~/"))
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(baseDir, path)
	}
	path = filepath.Clean(path)
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s is not a directory", path)
	}
	return path, nil
}

func ScanRepositories(bootstrap Bootstrap) ([]ScannedRepository, error) {
	opts := jsverbs.DefaultScanOptions()
	opts.IncludePublicFunctions = false

	ret := make([]ScannedRepository, 0, len(bootstrap.Repositories))
	for _, repo := range bootstrap.Repositories {
		registry, err := jsverbs.ScanDir(repo.RootDir, opts)
		if err != nil {
			return nil, fmt.Errorf("scan repository %s: %w", repo.Name, err)
		}
		ret = append(ret, ScannedRepository{Repository: repo, Registry: registry})
	}
	return ret, nil
}

func CollectDiscoveredBots(repositories []ScannedRepository) ([]DiscoveredBot, error) {
	seen := map[string]DiscoveredBot{}
	ret := []DiscoveredBot{}
	for _, repo := range repositories {
		for _, verb := range repo.Registry.Verbs() {
			candidate := DiscoveredBot{Repository: repo, Verb: verb}
			key := candidate.FullPath()
			if prev, ok := seen[key]; ok {
				return nil, fmt.Errorf("duplicate bot path %q from %s and %s", key, prev.SourceRef(), candidate.SourceRef())
			}
			seen[key] = candidate
			ret = append(ret, candidate)
		}
	}
	sort.Slice(ret, func(i, j int) bool {
		return ret[i].FullPath() < ret[j].FullPath()
	})
	return ret, nil
}
