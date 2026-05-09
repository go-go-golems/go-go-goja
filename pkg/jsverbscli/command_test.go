package jsverbscli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-go-golems/go-go-goja/pkg/jsverbrepos"
)

func TestScanRepositoriesDiscoversBuiltinVerb(t *testing.T) {
	repos, err := ScanRepositories(jsverbrepos.Bootstrap{Repositories: []jsverbrepos.Repository{jsverbrepos.BuiltinRepository()}})
	if err != nil {
		t.Fatalf("ScanRepositories() error = %v", err)
	}
	discovered, err := CollectDiscoveredVerbs(repos)
	if err != nil {
		t.Fatalf("CollectDiscoveredVerbs() error = %v", err)
	}
	if len(discovered) != 4 {
		t.Fatalf("expected four built-in verbs, got %d", len(discovered))
	}
	paths := []string{}
	for _, item := range discovered {
		paths = append(paths, item.Verb.FullPath())
	}
	if !contains(paths, "examples builtin hello") || !contains(paths, "examples builtin yaml-keys") || !contains(paths, "examples builtin tables") || !contains(paths, "examples builtin render-sample-table") {
		t.Fatalf("builtin verb paths = %#v", paths)
	}
}

func TestCollectDiscoveredVerbsRejectsDuplicatePaths(t *testing.T) {
	dir := t.TempDir()
	repoA := writeRepo(t, dir, "a")
	repoB := writeRepo(t, dir, "b")
	repos, err := ScanRepositories(jsverbrepos.Bootstrap{Repositories: []jsverbrepos.Repository{
		{Name: "a", Source: "test", RootDir: repoA},
		{Name: "b", Source: "test", RootDir: repoB},
	}})
	if err != nil {
		t.Fatalf("ScanRepositories() error = %v", err)
	}
	_, err = CollectDiscoveredVerbs(repos)
	if err == nil {
		t.Fatalf("expected duplicate error")
	}
	if !strings.Contains(err.Error(), "duplicate jsverb path") {
		t.Fatalf("unexpected duplicate error: %v", err)
	}
}

func TestLazyCommandListsBuiltinVerb(t *testing.T) {
	cmd := NewLazyCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"list", "--output", "json"})
	stdout := captureStdout(t, func() {
		if err := cmd.ExecuteContext(context.Background()); err != nil {
			t.Fatalf("ExecuteContext() error = %v\noutput:\n%s", err, out.String())
		}
	})
	if !strings.Contains(stdout, `"path": "examples builtin hello"`) {
		t.Fatalf("list JSON output did not contain builtin verb: %q", stdout)
	}
	if !strings.Contains(stdout, `"repository": "builtin"`) {
		t.Fatalf("list JSON output did not contain repository: %q", stdout)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()
	fn()
	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	return buf.String()
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func writeRepo(t *testing.T, base string, name string) string {
	t.Helper()
	dir := filepath.Join(base, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	content := `__package__({ name: "dupe", parents: ["examples"] });
function hello() { return { ok: true }; }
__verb__("hello", { short: "duplicate" });
`
	if err := os.WriteFile(filepath.Join(dir, "dupe.js"), []byte(content), 0o644); err != nil {
		t.Fatalf("write dupe.js: %v", err)
	}
	return dir
}
