package replsession

import (
	"context"
	"fmt"
	"reflect"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/pkg/resultcontent"
	"github.com/pkg/errors"
)

// ResultProcessor turns the settled value of one cell into a text rendering
// and the content providers to render for transports.
type ResultProcessor interface {
	// Process runs on the runtime owner goroutine with the settled value. It
	// must not await, call "then", or block on work that needs the owner.
	Process(ctx context.Context, vm *goja.Runtime, value goja.Value) (ProcessedResult, error)
}

// ProcessedResult is the outcome of a ResultProcessor.
type ProcessedResult struct {
	// Text replaces the default result preview when non-empty.
	Text string
	// Providers are rendered after Process returns, off the owner goroutine.
	Providers []resultcontent.Provider
}

// BeforeEvalHook may be implemented by a ResultProcessor to run code on the
// owner goroutine before each cell executes (for example to reset a per-cell
// content queue). An error fails the cell with that error.
type BeforeEvalHook interface {
	BeforeEval(ctx context.Context, vm *goja.Runtime) error
}

// FailureProcessor may be implemented by a ResultProcessor to contribute text
// and providers for a cell that threw or timed out (for example images queued
// before the error). It runs on the owner goroutine.
type FailureProcessor interface {
	ProcessFailure(ctx context.Context, vm *goja.Runtime, evalErr error) (ProcessedResult, error)
}

// DefaultResultProcessor keeps the default preview and extracts content
// providers with resultcontent.Extract. When providers are found, the text
// becomes the placeholder rendering of the whole value.
type DefaultResultProcessor struct {
	Limits resultcontent.Limits
	// FullText always uses the resultcontent rendering of the whole value as
	// the result text instead of the 120-character inspector preview. Agent
	// transports (MCP) want this; the web UI keeps the preview.
	FullText bool
}

var _ ResultProcessor = DefaultResultProcessor{}

// Process implements ResultProcessor.
func (p DefaultResultProcessor) Process(_ context.Context, vm *goja.Runtime, value goja.Value) (ProcessedResult, error) {
	extraction, err := resultcontent.Extract(vm, value, p.Limits)
	if err != nil {
		return ProcessedResult{}, err
	}
	if len(extraction.Providers) == 0 && !p.FullText {
		return ProcessedResult{}, nil
	}
	return ProcessedResult{Text: extraction.Text, Providers: extraction.Providers}, nil
}

// WithResultProcessor installs the processor used for every session of the
// service. A nil processor selects DefaultResultProcessor.
func WithResultProcessor(processor ResultProcessor) Option {
	return func(service *Service) {
		service.resultProcessor = processor
	}
}

// WithResultContentLimits sets the limits for content extraction (when the
// default processor is used) and rendering.
func WithResultContentLimits(limits resultcontent.Limits) Option {
	return func(service *Service) {
		service.contentLimits = limits.Normalize()
	}
}

// withoutContentRendering keeps processing (text, hooks) but skips calling
// providers; used for restore replay, whose responses are discarded.
func withoutContentRendering() Option {
	return func(service *Service) {
		service.skipContentRendering = true
	}
}

type processOutcome struct {
	Text      string
	Providers []resultcontent.Provider
	Err       string
}

// processValue runs the processor on the owner goroutine (the caller must be
// inside an owner call).
func (s *sessionState) processValue(ctx context.Context, vm *goja.Runtime, value goja.Value) processOutcome {
	if s.resultProcessor == nil {
		return processOutcome{}
	}
	result, err := safeProcess(func() (ProcessedResult, error) { return s.resultProcessor.Process(ctx, vm, value) })
	if err != nil {
		return processOutcome{Err: err.Error()}
	}
	return processOutcome{Text: result.Text, Providers: result.Providers}
}

func (s *sessionState) processFailure(ctx context.Context, evalErr error) processOutcome {
	failureProcessor, ok := s.resultProcessor.(FailureProcessor)
	if !ok || evalErr == nil {
		return processOutcome{}
	}
	ret, err := s.runtime.Owner.Call(context.WithoutCancel(ctx), "replsession.process-failure", func(_ context.Context, vm *goja.Runtime) (any, error) {
		result, err := safeProcess(func() (ProcessedResult, error) { return failureProcessor.ProcessFailure(ctx, vm, evalErr) })
		if err != nil {
			return processOutcome{Err: err.Error()}, nil
		}
		return processOutcome{Text: result.Text, Providers: result.Providers}, nil
	})
	if err != nil {
		return processOutcome{Err: err.Error()}
	}
	outcome, _ := ret.(processOutcome)
	return outcome
}

func (s *sessionState) beforeEval(ctx context.Context) error {
	hook, ok := s.resultProcessor.(BeforeEvalHook)
	if !ok {
		return nil
	}
	_, err := s.runtime.Owner.Call(ctx, "replsession.before-eval", func(_ context.Context, vm *goja.Runtime) (any, error) {
		return nil, hook.BeforeEval(ctx, vm)
	})
	if err != nil {
		return errors.Wrap(err, "before-eval hook")
	}
	return nil
}

func safeProcess(fn func() (ProcessedResult, error)) (ProcessedResult, error) {
	var (
		result ProcessedResult
		err    error
	)
	func() {
		defer func() {
			if r := recover(); r != nil {
				result, err = ProcessedResult{}, fmt.Errorf("result processor panicked: %v", r)
			}
		}()
		result, err = fn()
	}()
	return result, err
}

// renderContent renders providers off the owner goroutine.
func (s *Service) renderContent(ctx context.Context, providers []resultcontent.Provider) resultcontent.Rendering {
	if len(providers) == 0 {
		return resultcontent.Rendering{}
	}
	if s.skipContentRendering {
		return resultcontent.Rendering{}
	}
	return resultcontent.Render(ctx, providers, s.contentLimits)
}

// providerPlaceholders maps comparable providers to their 1-based number so the
// JSON envelope can name them the same way the text rendering does.
type providerPlaceholders map[resultcontent.Provider]int

func newProviderPlaceholders(providers []resultcontent.Provider) providerPlaceholders {
	out := providerPlaceholders{}
	for i, p := range providers {
		if p == nil || !reflect.TypeOf(p).Comparable() {
			continue
		}
		if _, ok := out[p]; !ok {
			out[p] = i + 1
		}
	}
	return out
}

func (m providerPlaceholders) placeholder(p resultcontent.Provider) string {
	if p != nil && reflect.TypeOf(p).Comparable() {
		if n, ok := m[p]; ok {
			return resultcontent.Placeholder(n, p)
		}
	}
	if desc := resultcontent.Describe(p); desc != "" {
		return "[content: " + desc + "]"
	}
	return "[content]"
}

// resultEnvelopeJSONWithProviders serializes the result like resultEnvelopeJSON
// but replaces every provider with its placeholder string so content bytes
// that a Go value might expose never reach the JSON (or the store).
func resultEnvelopeJSONWithProviders(vm *goja.Runtime, value goja.Value, providers []resultcontent.Provider) string {
	if len(providers) == 0 {
		return resultEnvelopeJSON(vm, value)
	}
	if p, ok := resultcontent.ProviderOf(value); ok {
		value = vm.ToValue(newProviderPlaceholders(providers).placeholder(p))
	}
	placeholders := newProviderPlaceholders(providers)
	replacer := vm.ToValue(func(call goja.FunctionCall) goja.Value {
		v := call.Argument(1)
		if p, ok := resultcontent.ProviderOf(v); ok {
			return vm.ToValue(placeholders.placeholder(p))
		}
		return v
	})
	return resultEnvelopeJSONWithReplacer(vm, value, replacer)
}
