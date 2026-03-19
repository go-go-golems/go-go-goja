package javascript

import (
	"context"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/go-go-golems/bobatea/pkg/repl"
	"github.com/go-go-golems/go-go-goja/pkg/docaccess"
	pluginprovider "github.com/go-go-golems/go-go-goja/pkg/docaccess/plugin"
	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/contract"
	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/host"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocsResolver_ResolvePluginEntries(t *testing.T) {
	provider, err := pluginprovider.NewProvider(
		pluginprovider.DefaultSourceID,
		"Plugin Manifests",
		"Runtime-scoped plugin metadata",
		[]host.LoadedModuleInfo{{
			Path: "/tmp/plugins/goja-plugin-examples-kv",
			Manifest: &contract.ModuleManifest{
				ModuleName:   "plugin:examples:kv",
				Version:      "v1",
				Doc:          "Example stateful plugin with object methods",
				Capabilities: []string{"examples", "stateful"},
				Exports: []*contract.ExportSpec{{
					Name: "store",
					Kind: contract.ExportKind_EXPORT_KIND_OBJECT,
					Doc:  "In-memory key/value store scoped to the plugin process",
					MethodSpecs: []*contract.MethodSpec{{
						Name: "get",
						Doc:  "Get a key, returning null if it is absent",
						Tags: []string{"lookup"},
					}},
				}},
			},
		}},
	)
	require.NoError(t, err)

	hub := docaccess.NewHub()
	require.NoError(t, hub.Register(provider))

	resolver := &docsResolver{
		hub:             hub,
		pluginSourceIDs: []string{pluginprovider.DefaultSourceID},
	}
	aliases := map[string]string{"kv": "plugin:examples:kv"}

	moduleEntry, ok := resolver.ResolveIdentifier("kv", aliases)
	require.True(t, ok)
	assert.Equal(t, "plugin:examples:kv", moduleEntry.Title)
	assert.Equal(t, "Example stateful plugin with object methods", moduleEntry.Summary)

	exportEntry, ok := resolver.ResolveProperty("kv", "store", aliases)
	require.True(t, ok)
	assert.Equal(t, "store", exportEntry.Title)
	assert.Equal(t, "In-memory key/value store scoped to the plugin process", exportEntry.Summary)

	methodEntry, ok := resolver.ResolveProperty("kv.store", "get", aliases)
	require.True(t, ok)
	assert.Equal(t, "store.get", methodEntry.Title)
	assert.Equal(t, "Get a key, returning null if it is absent", methodEntry.Summary)

	_, ok = resolver.ResolveProperty("Math", "max", aliases)
	assert.False(t, ok)
}

func TestEvaluator_PluginDocsDriveCompletionsAndHelp(t *testing.T) {
	binDir := t.TempDir()
	buildEvaluatorTestPlugin(t, filepath.Join(binDir, "goja-plugin-examples-kv"), "./plugins/examples/kv")

	evaluator, err := New(Config{
		EnableModules:     true,
		EnableConsoleLog:  true,
		EnableNodeModules: true,
		PluginDirectories: []string{binDir},
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, evaluator.Close())
	})

	ctx := context.Background()
	input := "const kv = require(\"plugin:examples:kv\");\nkv.store.g"

	completions, err := evaluator.CompleteInput(ctx, repl.CompletionRequest{
		Input:      input,
		CursorByte: len(input),
		Reason:     repl.CompletionReasonShortcut,
		Shortcut:   "tab",
	})
	require.NoError(t, err)
	require.True(t, completions.Show)
	assert.True(t, hasSuggestion(completions, "get"))
	assert.Contains(t, suggestionDisplay(completions, "get"), "Get a key, returning null if it is absent")

	helpBar, err := evaluator.GetHelpBar(ctx, repl.HelpBarRequest{
		Input:      input,
		CursorByte: len(input),
		Reason:     repl.HelpBarReasonManual,
	})
	require.NoError(t, err)
	require.True(t, helpBar.Show)
	assert.Equal(t, "docs", helpBar.Kind)
	assert.Contains(t, helpBar.Text, "store.get")
	assert.Contains(t, helpBar.Text, "Get a key, returning null if it is absent")

	helpDrawer, err := evaluator.GetHelpDrawer(ctx, repl.HelpDrawerRequest{
		Input:      input,
		CursorByte: len(input),
		RequestID:  12,
		Trigger:    repl.HelpDrawerTriggerManualRefresh,
	})
	require.NoError(t, err)
	require.True(t, helpDrawer.Show)
	assert.Equal(t, "store.get", helpDrawer.Title)
	assert.Contains(t, helpDrawer.Subtitle, "Plugin Method")
	assert.Contains(t, helpDrawer.Markdown, "### Documentation")
	assert.Contains(t, helpDrawer.Markdown, "Get a key, returning null if it is absent")
	assert.Contains(t, helpDrawer.Markdown, "### Related Docs")
}

func buildEvaluatorTestPlugin(t *testing.T, outputPath, packagePath string) {
	t.Helper()

	repoRoot := evaluatorTestRepoRoot(t)
	cmd := exec.Command("go", "build", "-o", outputPath, packagePath)
	cmd.Dir = repoRoot
	cmd.Env = append([]string{"GOWORK=off"}, cmd.Environ()...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build plugin %s: %v\n%s", packagePath, err, string(out))
	}
}

func evaluatorTestRepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("resolve caller")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", ".."))
}

func suggestionDisplay(result repl.CompletionResult, label string) string {
	for _, suggestion := range result.Suggestions {
		if suggestion.Value == label {
			return suggestion.DisplayText
		}
	}
	return ""
}
