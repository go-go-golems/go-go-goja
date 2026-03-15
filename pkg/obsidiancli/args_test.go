package obsidiancli

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildArgsOrdersParametersAndInjectsVault(t *testing.T) {
	args, err := BuildArgs(Config{Vault: "Main"}, SpecFileRead, CallOptions{
		Parameters: map[string]any{
			"path":   "Inbox/test.md",
			"format": "json",
			"tags":   []string{"foo", "bar"},
		},
		Flags:      []string{"verbose", "dry-run", "verbose"},
		Positional: []string{"tail"},
	})
	require.NoError(t, err)
	require.Equal(t, []string{
		"vault=Main",
		"file:read",
		"format=json",
		"path=Inbox/test.md",
		"tags=foo,bar",
		"dry-run",
		"verbose",
		"tail",
	}, args)
}

func TestBuildArgsSupportsCallVaultOverride(t *testing.T) {
	args, err := BuildArgs(Config{Vault: "Main"}, SpecVersion, CallOptions{
		Vault: "Work",
	})
	require.NoError(t, err)
	require.Equal(t, []string{"vault=Work", "version"}, args)
}

func TestBuildArgsRejectsEmptyCommandName(t *testing.T) {
	_, err := BuildArgs(DefaultConfig(), CommandSpec{}, CallOptions{})
	require.Error(t, err)
}
