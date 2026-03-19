package host

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDiscoveryDirectoriesUnderRoot(t *testing.T) {
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, "alpha", "nested"))
	mustMkdirAll(t, filepath.Join(root, "beta"))

	got := discoveryDirectoriesUnderRoot(root)
	want := []string{
		filepath.Clean(root),
		filepath.Join(root, "alpha"),
		filepath.Join(root, "alpha", "nested"),
		filepath.Join(root, "beta"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("discoveryDirectoriesUnderRoot() = %#v, want %#v", got, want)
	}
}

func TestResolveDiscoveryDirectoriesPrefersExplicitDirectories(t *testing.T) {
	homeDir := setTestHomeDir(t)

	got := ResolveDiscoveryDirectories([]string{"~/plugins-b", "~/plugins-a", "~/plugins-a"})
	want := []string{
		filepath.Join(homeDir, "plugins-a"),
		filepath.Join(homeDir, "plugins-b"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ResolveDiscoveryDirectories() = %#v, want %#v", got, want)
	}
}

func TestResolveDiscoveryDirectoriesFallsBackToDefaultRoot(t *testing.T) {
	homeDir := setTestHomeDir(t)
	root := filepath.Join(homeDir, ".go-go-goja", "plugins")
	mustMkdirAll(t, filepath.Join(root, "team", "alpha"))

	got := ResolveDiscoveryDirectories(nil)
	want := []string{
		filepath.Clean(root),
		filepath.Join(root, "team"),
		filepath.Join(root, "team", "alpha"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ResolveDiscoveryDirectories(nil) = %#v, want %#v", got, want)
	}
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

func setTestHomeDir(t *testing.T) string {
	t.Helper()
	homeDir := filepath.Join(t.TempDir(), "home")
	t.Setenv("HOME", homeDir)
	return homeDir
}
