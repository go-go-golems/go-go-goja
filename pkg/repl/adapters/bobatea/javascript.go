package bobatea

import (
	"context"

	bobarepl "github.com/go-go-golems/bobatea/pkg/repl"
	js "github.com/go-go-golems/go-go-goja/pkg/repl/evaluators/javascript"
)

// JavaScriptEvaluator adapts the go-go-goja-owned JS evaluator to Bobatea REPL contracts.
type JavaScriptEvaluator struct {
	core *js.Evaluator
}

// NewJavaScriptEvaluator creates a Bobatea REPL adapter backed by the go-go-goja JS evaluator.
func NewJavaScriptEvaluator(config js.Config) (*JavaScriptEvaluator, error) {
	core, err := js.New(config)
	if err != nil {
		return nil, err
	}
	return &JavaScriptEvaluator{core: core}, nil
}

// NewJavaScriptEvaluatorWithDefaults creates an adapter with default JS evaluator configuration.
func NewJavaScriptEvaluatorWithDefaults() (*JavaScriptEvaluator, error) {
	return NewJavaScriptEvaluator(js.DefaultConfig())
}

// Core returns the underlying go-go-goja evaluator for advanced integrations.
func (e *JavaScriptEvaluator) Core() *js.Evaluator {
	return e.core
}

func (e *JavaScriptEvaluator) EvaluateStream(ctx context.Context, code string, emit func(bobarepl.Event)) error {
	return e.core.EvaluateStream(ctx, code, emit)
}

func (e *JavaScriptEvaluator) GetPrompt() string {
	return e.core.GetPrompt()
}

func (e *JavaScriptEvaluator) GetName() string {
	return e.core.GetName()
}

func (e *JavaScriptEvaluator) SupportsMultiline() bool {
	return e.core.SupportsMultiline()
}

func (e *JavaScriptEvaluator) GetFileExtension() string {
	return e.core.GetFileExtension()
}

func (e *JavaScriptEvaluator) CompleteInput(ctx context.Context, req bobarepl.CompletionRequest) (bobarepl.CompletionResult, error) {
	return e.core.CompleteInput(ctx, req)
}

func (e *JavaScriptEvaluator) GetHelpBar(ctx context.Context, req bobarepl.HelpBarRequest) (bobarepl.HelpBarPayload, error) {
	return e.core.GetHelpBar(ctx, req)
}

func (e *JavaScriptEvaluator) GetHelpDrawer(ctx context.Context, req bobarepl.HelpDrawerRequest) (bobarepl.HelpDrawerDocument, error) {
	return e.core.GetHelpDrawer(ctx, req)
}

var _ bobarepl.Evaluator = (*JavaScriptEvaluator)(nil)
var _ bobarepl.InputCompleter = (*JavaScriptEvaluator)(nil)
var _ bobarepl.HelpBarProvider = (*JavaScriptEvaluator)(nil)
var _ bobarepl.HelpDrawerProvider = (*JavaScriptEvaluator)(nil)
