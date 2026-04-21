package bobatea

import (
	"context"
	"testing"

	"github.com/dop251/goja"
	bobarepl "github.com/go-go-golems/bobatea/pkg/repl"
)

func TestRuntimeAssistanceRecordDeclarationsFeedsCompletion(t *testing.T) {
	t.Parallel()

	vm := goja.New()
	assist, err := NewRuntimeAssistance(RuntimeAssistanceConfig{Runtime: vm})
	if err != nil {
		t.Fatalf("new runtime assistance: %v", err)
	}

	assist.RecordDeclarations("const answer = 41; function alpha() { return answer; }")

	result, err := assist.CompleteInput(context.Background(), bobarepl.CompletionRequest{
		Input:      "ans",
		CursorByte: len("ans"),
		Reason:     bobarepl.CompletionReasonShortcut,
	})
	if err != nil {
		t.Fatalf("complete input: %v", err)
	}
	if !result.Show {
		t.Fatal("expected completion popup to show")
	}
	found := false
	for _, suggestion := range result.Suggestions {
		if suggestion.Value == "answer" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected answer suggestion, got %#v", result.Suggestions)
	}
}

func TestRuntimeAssistanceProvidesHelp(t *testing.T) {
	t.Parallel()

	vm := goja.New()
	assist, err := NewRuntimeAssistance(RuntimeAssistanceConfig{Runtime: vm})
	if err != nil {
		t.Fatalf("new runtime assistance: %v", err)
	}

	helpBar, err := assist.GetHelpBar(context.Background(), bobarepl.HelpBarRequest{
		Input:      "console.log",
		CursorByte: len("console.log"),
		Reason:     bobarepl.HelpBarReasonDebounce,
	})
	if err != nil {
		t.Fatalf("get help bar: %v", err)
	}
	if !helpBar.Show {
		t.Fatal("expected help bar to show")
	}

	helpDrawer, err := assist.GetHelpDrawer(context.Background(), bobarepl.HelpDrawerRequest{
		Input:      "console.lo",
		CursorByte: len("console.lo"),
		RequestID:  1,
		Trigger:    bobarepl.HelpDrawerTriggerTyping,
	})
	if err != nil {
		t.Fatalf("get help drawer: %v", err)
	}
	if !helpDrawer.Show {
		t.Fatal("expected help drawer to show")
	}
}
