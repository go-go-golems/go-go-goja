package jsverbrepos

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRepositoriesFromArgsParsesLeadingRepositoryFlags(t *testing.T) {
	dir := t.TempDir()
	repoA := mkdir(t, dir, "repo-a")
	repoB := mkdir(t, dir, "repo-b")

	repos, remaining, err := RepositoriesFromArgs([]string{
		"--repository", repoA,
		"--verb-repository=" + repoB,
		"examples", "builtin", "hello", "--name", "Manuel",
	}, dir)
	if err != nil {
		t.Fatalf("RepositoriesFromArgs() error = %v", err)
	}
	if len(repos) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(repos))
	}
	if repos[0].RootDir != repoA || repos[1].RootDir != repoB {
		t.Fatalf("unexpected repo roots: %#v", repos)
	}
	wantRemaining := []string{"examples", "builtin", "hello", "--name", "Manuel"}
	if !equalStrings(remaining, wantRemaining) {
		t.Fatalf("remaining = %#v, want %#v", remaining, wantRemaining)
	}
}

func TestRepositoriesFromArgsStopsAtFirstNonRepositoryArg(t *testing.T) {
	dir := t.TempDir()
	repoA := mkdir(t, dir, "repo-a")

	repos, remaining, err := RepositoriesFromArgs([]string{
		"--repository", repoA,
		"examples", "--repository", repoA,
	}, dir)
	if err != nil {
		t.Fatalf("RepositoriesFromArgs() error = %v", err)
	}
	if len(repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(repos))
	}
	wantRemaining := []string{"examples", "--repository", repoA}
	if !equalStrings(remaining, wantRemaining) {
		t.Fatalf("remaining = %#v, want %#v", remaining, wantRemaining)
	}
}

func TestDiscoverFromIncludesBuiltinConfigEnvAndCLIWithDedupe(t *testing.T) {
	t.Setenv(EnvVar, "")
	dir := t.TempDir()
	repoConfig := mkdir(t, dir, "config-repo")
	repoEnv := mkdir(t, dir, "env-repo")
	repoCLI := mkdir(t, dir, "cli-repo")
	writeFile(t, filepath.Join(dir, LocalConfigFileName), "verbs:\n  repositories:\n    - name: configured\n      path: ./config-repo\n    - name: disabled\n      path: ./config-repo\n      enabled: false\n")
	t.Setenv(EnvVar, repoEnv+string(os.PathListSeparator)+repoConfig)

	bootstrap, remaining, err := DiscoverFrom(context.Background(), dir, []string{"--repository", repoCLI, "list"})
	if err != nil {
		t.Fatalf("DiscoverFrom() error = %v", err)
	}
	if !equalStrings(remaining, []string{"list"}) {
		t.Fatalf("remaining = %#v", remaining)
	}
	if len(bootstrap.Repositories) != 4 {
		t.Fatalf("expected builtin + config + env + cli after dedupe, got %d: %#v", len(bootstrap.Repositories), bootstrap.Repositories)
	}
	if bootstrap.Repositories[0].Name != "builtin" || !bootstrap.Repositories[0].Embedded {
		t.Fatalf("first repository should be embedded builtin: %#v", bootstrap.Repositories[0])
	}
	if bootstrap.Repositories[1].Name != "configured" || bootstrap.Repositories[1].RootDir != repoConfig {
		t.Fatalf("second repository should be config repo: %#v", bootstrap.Repositories[1])
	}
	if bootstrap.Repositories[2].RootDir != repoEnv {
		t.Fatalf("third repository should be env repo: %#v", bootstrap.Repositories[2])
	}
	if bootstrap.Repositories[3].RootDir != repoCLI {
		t.Fatalf("fourth repository should be cli repo: %#v", bootstrap.Repositories[3])
	}
}

func TestNormalizeFilesystemRepositoryPathExpandsRelativeAndHome(t *testing.T) {
	dir := t.TempDir()
	repo := mkdir(t, dir, "repo")

	normalized, err := NormalizeFilesystemRepositoryPath("./repo", dir)
	if err != nil {
		t.Fatalf("NormalizeFilesystemRepositoryPath(relative) error = %v", err)
	}
	if normalized != repo {
		t.Fatalf("relative normalized = %q, want %q", normalized, repo)
	}

	home := t.TempDir()
	t.Setenv("HOME", home)
	homeRepo := mkdir(t, home, "home-repo")
	normalized, err = NormalizeFilesystemRepositoryPath("~/home-repo", dir)
	if err != nil {
		t.Fatalf("NormalizeFilesystemRepositoryPath(home) error = %v", err)
	}
	if normalized != homeRepo {
		t.Fatalf("home normalized = %q, want %q", normalized, homeRepo)
	}
}

func mkdir(t *testing.T, base string, name string) string {
	t.Helper()
	path := filepath.Join(base, name)
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	return path
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
