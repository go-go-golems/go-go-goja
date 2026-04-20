package botcli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestListCommandOutputsDiscoveredBots(t *testing.T) {
	root := NewCommand()
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stdout)
	root.SetArgs([]string{"list", "--bot-repository", fixtureDir(t)})

	err := root.Execute()
	require.NoError(t, err)
	output := stdout.String()
	require.Contains(t, output, "basics greet\tbasics.js")
	require.NotContains(t, output, "meta pkg-demo ping\tpackaged.js")
}

func TestRunCommandExecutesStructuredBot(t *testing.T) {
	root := NewCommand()
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stdout)
	root.SetArgs([]string{"run", "basics", "greet", "--bot-repository", fixtureDir(t), "Manuel", "--excited"})

	err := root.Execute()
	require.NoError(t, err)
	output := stdout.String()
	require.Contains(t, output, "\"greeting\": \"Hello, Manuel!\"")
}

func TestRunCommandExecutesTextBot(t *testing.T) {
	root := NewCommand()
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stdout)
	root.SetArgs([]string{"run", "basics", "banner", "--bot-repository", fixtureDir(t), "Manuel"})

	err := root.Execute()
	require.NoError(t, err)
	require.Equal(t, "=== Manuel ===\n", stdout.String())
}

func TestRunCommandSettlesAsyncBot(t *testing.T) {
	root := NewCommand()
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stdout)
	root.SetArgs([]string{"run", "advanced", "numbers", "multiply", "--bot-repository", fixtureDir(t), "6", "7"})

	err := root.Execute()
	require.NoError(t, err)
	output := stdout.String()
	require.Contains(t, output, "\"product\": 42")
}

func TestRunCommandSupportsRelativeRequire(t *testing.T) {
	root := NewCommand()
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stdout)
	root.SetArgs([]string{"run", "nested", "with-helper", "render", "--bot-repository", fixtureDir(t), "hi", "there"})

	err := root.Execute()
	require.NoError(t, err)
	output := stdout.String()
	require.Contains(t, output, "\"value\": \"hi:there\"")
}

func TestHelpCommandShowsVerbFlags(t *testing.T) {
	root := NewCommand()
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stdout)
	root.SetArgs([]string{"help", "basics", "greet", "--bot-repository", fixtureDir(t)})

	err := root.Execute()
	require.NoError(t, err)
	output := stdout.String()
	require.Contains(t, output, "go-go-goja bots run basics greet")
	require.Contains(t, output, "--excited")
	require.Contains(t, output, "Greets one person")
}

func TestListCommandSupportsMultipleRepositories(t *testing.T) {
	root := NewCommand()
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stdout)
	root.SetArgs([]string{"list", "--bot-repository", fixtureDir(t), "--bot-repository", botFixtureDir(t)})

	err := root.Execute()
	require.NoError(t, err)
	output := stdout.String()
	require.Contains(t, output, "basics greet\tbasics.js")
	require.Contains(t, output, "discord greet\tdiscord.js")
}

func TestListCommandAllowsEmptyRepository(t *testing.T) {
	emptyDir := t.TempDir()
	root := NewCommand()
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stdout)
	root.SetArgs([]string{"list", "--bot-repository", emptyDir})

	err := root.Execute()
	require.NoError(t, err)
	require.Equal(t, "", stdout.String())
}

func TestListCommandRejectsDuplicateBotsAcrossRepositories(t *testing.T) {
	root := NewCommand()
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stdout)
	root.SetArgs([]string{"list", "--bot-repository", duplicateFixtureDir(t, "a"), "--bot-repository", duplicateFixtureDir(t, "b")})

	err := root.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "duplicate bot path")
	require.Contains(t, err.Error(), "discord greet")
}

func TestDedicatedBotFixtureRunsAndSupportsRelativeRequire(t *testing.T) {
	root := NewCommand()
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stdout)
	root.SetArgs([]string{"run", "nested", "relay", "relay", "--bot-repository", botFixtureDir(t), "hi", "there"})

	err := root.Execute()
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "\"value\": \"hi:there\"")
}

func TestExamplesRepositoryListsAllCorePaths(t *testing.T) {
	root := NewCommand()
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stdout)
	root.SetArgs([]string{"list", "--bot-repository", examplesFixtureDir(t)})

	err := root.Execute()
	require.NoError(t, err)
	output := stdout.String()
	require.Contains(t, output, "discord greet\tdiscord.js")
	require.Contains(t, output, "discord banner\tdiscord.js")
	require.Contains(t, output, "math multiply\tmath/index.js")
	require.Contains(t, output, "nested relay\tnested/index.js")
	require.Contains(t, output, "issues list\tissues.js")
	require.Contains(t, output, "meta ops status\tadmin.js")
	require.Contains(t, output, "all-values echo-all\tall-values.js")
}

func TestExamplesRepositorySupportsRealSmokeCommands(t *testing.T) {
	t.Run("structured", func(t *testing.T) {
		root := NewCommand()
		var stdout bytes.Buffer
		root.SetOut(&stdout)
		root.SetErr(&stdout)
		root.SetArgs([]string{"run", "discord", "greet", "--bot-repository", examplesFixtureDir(t), "Manuel", "--excited"})
		require.NoError(t, root.Execute())
		require.Contains(t, stdout.String(), "\"greeting\": \"Hello, Manuel!\"")
	})

	t.Run("text", func(t *testing.T) {
		root := NewCommand()
		var stdout bytes.Buffer
		root.SetOut(&stdout)
		root.SetErr(&stdout)
		root.SetArgs([]string{"run", "discord", "banner", "--bot-repository", examplesFixtureDir(t), "Manuel"})
		require.NoError(t, root.Execute())
		require.Equal(t, "*** Manuel ***\n", stdout.String())
	})

	t.Run("async", func(t *testing.T) {
		root := NewCommand()
		var stdout bytes.Buffer
		root.SetOut(&stdout)
		root.SetErr(&stdout)
		root.SetArgs([]string{"run", "math", "multiply", "--bot-repository", examplesFixtureDir(t), "6", "7"})
		require.NoError(t, root.Execute())
		require.Contains(t, stdout.String(), "\"product\": 42")
	})

	t.Run("nested require", func(t *testing.T) {
		root := NewCommand()
		var stdout bytes.Buffer
		root.SetOut(&stdout)
		root.SetErr(&stdout)
		root.SetArgs([]string{"run", "nested", "relay", "--bot-repository", examplesFixtureDir(t), "hi", "there"})
		require.NoError(t, root.Execute())
		require.Contains(t, stdout.String(), "\"value\": \"hi:there\"")
	})

	t.Run("sections and context", func(t *testing.T) {
		root := NewCommand()
		var stdout bytes.Buffer
		root.SetOut(&stdout)
		root.SetErr(&stdout)
		root.SetArgs([]string{"run", "issues", "list", "--bot-repository", examplesFixtureDir(t), "acme/repo", "--state", "closed", "--labels", "bug", "--labels", "docs"})
		require.NoError(t, root.Execute())
		out := stdout.String()
		require.Contains(t, out, "\"repo\": \"acme/repo\"")
		require.Contains(t, out, "\"state\": \"closed\"")
		require.Contains(t, out, "\"labelCount\": 2")
	})

	t.Run("package metadata", func(t *testing.T) {
		root := NewCommand()
		var stdout bytes.Buffer
		root.SetOut(&stdout)
		root.SetErr(&stdout)
		root.SetArgs([]string{"run", "meta", "ops", "status", "--bot-repository", examplesFixtureDir(t)})
		require.NoError(t, root.Execute())
		require.Contains(t, stdout.String(), "\"scope\": \"meta ops\"")
	})

	t.Run("bind all", func(t *testing.T) {
		root := NewCommand()
		var stdout bytes.Buffer
		root.SetOut(&stdout)
		root.SetErr(&stdout)
		root.SetArgs([]string{"run", "all-values", "echo-all", "--bot-repository", examplesFixtureDir(t), "--repo", "acme/demo", "--dryRun", "--names", "one", "--names", "two"})
		require.NoError(t, root.Execute())
		out := stdout.String()
		require.Contains(t, out, "\"repo\": \"acme/demo\"")
		require.Contains(t, out, "\"dryRun\": true")
		require.Contains(t, out, "\"count\": 2")
	})
}

func TestExamplesRepositoryHelpShowsRealFlags(t *testing.T) {
	root := NewCommand()
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stdout)
	root.SetArgs([]string{"help", "issues", "list", "--bot-repository", examplesFixtureDir(t)})

	err := root.Execute()
	require.NoError(t, err)
	output := stdout.String()
	require.Contains(t, output, "go-go-goja bots run issues list")
	require.Contains(t, output, "--state")
	require.Contains(t, output, "--labels")
}

func fixtureDir(t *testing.T) string {
	t.Helper()
	root := repoRoot(t)
	return filepath.Join(root, "testdata", "jsverbs")
}

func botFixtureDir(t *testing.T) string {
	t.Helper()
	root := repoRoot(t)
	return filepath.Join(root, "testdata", "botcli")
}

func duplicateFixtureDir(t *testing.T, suffix string) string {
	t.Helper()
	root := repoRoot(t)
	return filepath.Join(root, "testdata", "botcli-dupe-"+suffix)
}

func examplesFixtureDir(t *testing.T) string {
	t.Helper()
	root := repoRoot(t)
	return filepath.Join(root, "examples", "bots")
}

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	_, err = os.Stat(root)
	require.NoError(t, err)
	return root
}
