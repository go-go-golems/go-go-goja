package botcli

import (
	"bytes"
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

func fixtureDir(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	return filepath.Join(root, "testdata", "jsverbs")
}
