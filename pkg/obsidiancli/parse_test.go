package obsidiancli

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseOutputJSON(t *testing.T) {
	parsed, err := ParseOutput(OutputJSON, `{"name":"Main","count":2}`)
	require.NoError(t, err)
	data, ok := parsed.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "Main", data["name"])
	require.Equal(t, float64(2), data["count"])
}

func TestParseLineList(t *testing.T) {
	parsed, err := ParseOutput(OutputLineList, "a.md\n\nb.md\n")
	require.NoError(t, err)
	require.Equal(t, []string{"a.md", "b.md"}, parsed)
}

func TestParseKeyValue(t *testing.T) {
	parsed, err := ParseOutput(OutputKeyValue, "name=Main\npath: /vault\n")
	require.NoError(t, err)
	require.Equal(t, map[string]string{
		"name": "Main",
		"path": "/vault",
	}, parsed)
}

func TestParseKeyValueRejectsInvalidLine(t *testing.T) {
	_, err := ParseOutput(OutputKeyValue, "name=Main\nbroken\n")
	require.Error(t, err)
}
