package bobatea

import (
	"context"
	"testing"

	bobarepl "github.com/go-go-golems/bobatea/pkg/repl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJavaScriptEvaluatorAdapter_Defaults(t *testing.T) {
	adapter, err := NewJavaScriptEvaluatorWithDefaults()
	require.NoError(t, err)
	require.NotNil(t, adapter)

	assert.Equal(t, "JavaScript", adapter.GetName())
	assert.Equal(t, "js>", adapter.GetPrompt())
	assert.True(t, adapter.SupportsMultiline())
	assert.Equal(t, ".js", adapter.GetFileExtension())
}

func TestJavaScriptEvaluatorAdapter_EvaluateStream(t *testing.T) {
	adapter, err := NewJavaScriptEvaluatorWithDefaults()
	require.NoError(t, err)

	var events []bobarepl.Event
	err = adapter.EvaluateStream(context.Background(), "2 + 3", func(ev bobarepl.Event) {
		events = append(events, ev)
	})
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, bobarepl.EventResultMarkdown, events[0].Kind)
	assert.Equal(t, "5", events[0].Props["markdown"])
}

func TestJavaScriptEvaluatorAdapter_CapabilityDelegation(t *testing.T) {
	adapter, err := NewJavaScriptEvaluatorWithDefaults()
	require.NoError(t, err)

	completion, err := adapter.CompleteInput(context.Background(), bobarepl.CompletionRequest{
		Input:      "console.lo",
		CursorByte: len("console.lo"),
		Reason:     bobarepl.CompletionReasonShortcut,
	})
	require.NoError(t, err)
	assert.True(t, completion.Show)

	helpBar, err := adapter.GetHelpBar(context.Background(), bobarepl.HelpBarRequest{
		Input:      "console.log",
		CursorByte: len("console.log"),
		Reason:     bobarepl.HelpBarReasonDebounce,
	})
	require.NoError(t, err)
	assert.True(t, helpBar.Show)

	helpDrawer, err := adapter.GetHelpDrawer(context.Background(), bobarepl.HelpDrawerRequest{
		Input:      "console.lo",
		CursorByte: len("console.lo"),
		RequestID:  1,
		Trigger:    bobarepl.HelpDrawerTriggerTyping,
	})
	require.NoError(t, err)
	assert.True(t, helpDrawer.Show)
}
