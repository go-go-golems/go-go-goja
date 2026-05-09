package jsverbrepos

import (
	"context"
	"embed"
	"encoding/csv"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	EnvVar             = "GOJA_VERB_REPOSITORIES"
	RepositoryFlag     = "repository"
	VerbRepositoryFlag = "verb-repository"

	LocalConfigFileName         = ".goja-verbs.yml"
	LocalOverrideConfigFileName = ".goja-verbs.override.yml"
)

//go:embed builtin/*.js
var builtinScripts embed.FS

type Bootstrap struct {
	Repositories []Repository
}

type Repository struct {
	Name       string
	Source     string
	SourceRef  string
	RootDir    string
	EmbeddedFS fs.FS
	Embedded   bool
	EmbeddedAt string
}

type appConfig struct {
	Verbs verbsConfig `yaml:"verbs"`
}

type verbsConfig struct {
	Repositories []repositorySpec `yaml:"repositories"`
}

type repositorySpec struct {
	Name    string `yaml:"name,omitempty"`
	Path    string `yaml:"path"`
	Enabled *bool  `yaml:"enabled,omitempty"`
}

func Discover(ctx context.Context, args []string) (Bootstrap, []string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return Bootstrap{}, nil, fmt.Errorf("resolve cwd: %w", err)
	}
	return DiscoverFrom(ctx, cwd, args)
}

func DiscoverFrom(ctx context.Context, cwd string, args []string) (Bootstrap, []string, error) {
	cliRepos, remainingArgs, err := RepositoriesFromArgs(args, cwd)
	if err != nil {
		return Bootstrap{}, nil, err
	}
	bootstrap, err := discoverFromSources(ctx, cwd, cliRepos)
	if err != nil {
		return Bootstrap{}, nil, err
	}
	return bootstrap, remainingArgs, nil
}

func discoverFromSources(ctx context.Context, cwd string, cliRepos []Repository) (Bootstrap, error) {
	_ = ctx
	repositories := []Repository{BuiltinRepository()}
	seen := map[string]struct{}{RepositoryIdentity(repositories[0]): {}}

	configRepos, err := LoadConfigRepositories(cwd)
	if err != nil {
		return Bootstrap{}, err
	}
	for _, repo := range configRepos {
		appendRepository(&repositories, seen, repo)
	}

	envRepos, err := RepositoriesFromEnv(cwd)
	if err != nil {
		return Bootstrap{}, err
	}
	for _, repo := range envRepos {
		appendRepository(&repositories, seen, repo)
	}
	for _, repo := range cliRepos {
		appendRepository(&repositories, seen, repo)
	}

	return Bootstrap{Repositories: repositories}, nil
}

func BuiltinRepository() Repository {
	return Repository{
		Name:       "builtin",
		Source:     "embedded",
		SourceRef:  "builtin scripts",
		EmbeddedFS: builtinScripts,
		Embedded:   true,
		EmbeddedAt: "builtin",
	}
}

func appendRepository(repositories *[]Repository, seen map[string]struct{}, repo Repository) {
	identity := RepositoryIdentity(repo)
	if _, ok := seen[identity]; ok {
		return
	}
	seen[identity] = struct{}{}
	*repositories = append(*repositories, repo)
}

func RepositoryIdentity(repo Repository) string {
	if repo.Embedded {
		return "embedded:" + repo.Name + ":" + repo.EmbeddedAt
	}
	return "path:" + filepath.Clean(repo.RootDir)
}

func LoadConfigRepositories(cwd string) ([]Repository, error) {
	paths := configPaths(cwd)
	ret := []Repository{}
	for _, path := range paths {
		repos, err := loadRepositoriesFromConfigFile(path)
		if err != nil {
			return nil, err
		}
		ret = append(ret, repos...)
	}
	return ret, nil
}

func configPaths(cwd string) []string {
	candidates := []string{}
	if root := findGitRoot(cwd); root != "" {
		candidates = append(candidates,
			filepath.Join(root, LocalConfigFileName),
			filepath.Join(root, LocalOverrideConfigFileName),
		)
	}
	candidates = append(candidates,
		filepath.Join(cwd, LocalConfigFileName),
		filepath.Join(cwd, LocalOverrideConfigFileName),
	)

	seen := map[string]struct{}{}
	ret := []string{}
	for _, path := range candidates {
		path = filepath.Clean(path)
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			ret = append(ret, path)
		}
	}
	return ret
}

func findGitRoot(cwd string) string {
	dir := filepath.Clean(cwd)
	for {
		if info, err := os.Stat(filepath.Join(dir, ".git")); err == nil && info.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func loadRepositoriesFromConfigFile(path string) ([]Repository, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read app config %s: %w", path, err)
	}
	cfg := &appConfig{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse app config %s: %w", path, err)
	}
	baseDir := filepath.Dir(path)
	ret := []Repository{}
	for _, spec := range cfg.Verbs.Repositories {
		if spec.Enabled != nil && !*spec.Enabled {
			continue
		}
		normalized, err := NormalizeFilesystemRepositoryPath(spec.Path, baseDir)
		if err != nil {
			return nil, fmt.Errorf("config repository %q in %s: %w", spec.Path, path, err)
		}
		name := strings.TrimSpace(spec.Name)
		if name == "" {
			name = filepath.Base(normalized)
		}
		ret = append(ret, Repository{Name: name, Source: "config", SourceRef: path, RootDir: normalized})
	}
	return ret, nil
}

func RepositoriesFromEnv(cwd string) ([]Repository, error) {
	value := strings.TrimSpace(os.Getenv(EnvVar))
	if value == "" {
		return nil, nil
	}
	return repositoriesFromPathList(filepath.SplitList(value), cwd, "env", EnvVar)
}

func RepositoriesFromArgs(args []string, cwd string) ([]Repository, []string, error) {
	paths := []string{}
	remainingStart := 0
	for remainingStart < len(args) {
		arg := args[remainingStart]
		switch {
		case arg == "--":
			remainingStart++
			goto done
		case arg == "--"+RepositoryFlag || arg == "--"+VerbRepositoryFlag:
			if remainingStart+1 >= len(args) {
				return nil, nil, fmt.Errorf("%s requires a value", arg)
			}
			paths = appendCSVPaths(paths, args[remainingStart+1])
			remainingStart += 2
		case strings.HasPrefix(arg, "--"+RepositoryFlag+"="):
			paths = appendCSVPaths(paths, strings.TrimPrefix(arg, "--"+RepositoryFlag+"="))
			remainingStart++
		case strings.HasPrefix(arg, "--"+VerbRepositoryFlag+"="):
			paths = appendCSVPaths(paths, strings.TrimPrefix(arg, "--"+VerbRepositoryFlag+"="))
			remainingStart++
		default:
			goto done
		}
	}

done:
	repos, err := repositoriesFromPathList(paths, cwd, "cli", "--"+RepositoryFlag)
	if err != nil {
		return nil, nil, err
	}
	return repos, append([]string{}, args[remainingStart:]...), nil
}

func appendCSVPaths(paths []string, value string) []string {
	reader := csv.NewReader(strings.NewReader(value))
	fields, err := reader.Read()
	if err != nil || len(fields) == 0 {
		return append(paths, value)
	}
	return append(paths, fields...)
}

func repositoriesFromPathList(paths []string, cwd string, source string, sourceRef string) ([]Repository, error) {
	ret := []Repository{}
	for _, raw := range paths {
		normalized, err := NormalizeFilesystemRepositoryPath(raw, cwd)
		if err != nil {
			return nil, fmt.Errorf("%s repository %q: %w", source, raw, err)
		}
		ret = append(ret, Repository{Name: filepath.Base(normalized), Source: source, SourceRef: sourceRef, RootDir: normalized})
	}
	return ret, nil
}

func NormalizeFilesystemRepositoryPath(path string, baseDir string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("repository path is empty")
	}
	if path == "~" || strings.HasPrefix(path, "~/") {
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

func RepositoryNames(repos []Repository) []string {
	names := make([]string, 0, len(repos))
	for _, repo := range repos {
		names = append(names, repo.Name)
	}
	sort.Strings(names)
	return names
}
