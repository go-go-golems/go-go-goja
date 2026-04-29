package bobatea

import (
	"context"
	"strings"
	"sync"

	"github.com/dop251/goja"
	bobarepl "github.com/go-go-golems/bobatea/pkg/repl"
	"github.com/go-go-golems/go-go-goja/pkg/docaccess"
	"github.com/go-go-golems/go-go-goja/pkg/jsparse"
	js "github.com/go-go-golems/go-go-goja/pkg/repl/evaluators/javascript"
	"github.com/pkg/errors"
)

// RuntimeAssistance provides Bobatea completion/help capabilities for callers
// that already own a goja runtime and only need editor assistance.
type RuntimeAssistance struct {
	runtime        *goja.Runtime
	docHub         *docaccess.Hub
	assist         *js.Assistance
	tsMu           sync.Mutex
	declaredMu     sync.RWMutex
	declaredByName map[string]jsparse.CompletionCandidate
}

// RuntimeAssistanceConfig configures assistance against an existing runtime.
type RuntimeAssistanceConfig struct {
	Runtime *goja.Runtime
	DocHub  *docaccess.Hub
}

// NewRuntimeAssistance creates a Bobatea assistance adapter backed by an
// already-owned runtime. It does not evaluate code or own the runtime.
func NewRuntimeAssistance(config RuntimeAssistanceConfig) (*RuntimeAssistance, error) {
	if config.Runtime == nil {
		return nil, errors.New("runtime assistance: runtime is nil")
	}
	tsParser, err := jsparse.NewTSParser()
	if err != nil {
		return nil, errors.Wrap(err, "runtime assistance: create TypeScript parser")
	}
	ret := &RuntimeAssistance{
		runtime:        config.Runtime,
		docHub:         config.DocHub,
		declaredByName: map[string]jsparse.CompletionCandidate{},
	}
	ret.assist = js.NewAssistance(js.AssistanceConfig{
		TSParser: tsParser,
		TSMu:     &ret.tsMu,
		WithRuntime: func(ctx context.Context, fn func(*goja.Runtime, *docaccess.Hub) error) error {
			return fn(ret.runtime, ret.docHub)
		},
		BindingHints: func(context.Context) ([]jsparse.CompletionCandidate, error) {
			return ret.bindingHints(), nil
		},
	})
	return ret, nil
}

// RecordDeclarations updates identifier hints from source text without
// evaluating the code.
func (a *RuntimeAssistance) RecordDeclarations(code string) {
	if a == nil {
		return
	}
	candidates := jsparse.ExtractTopLevelBindingCandidates(code)
	if len(candidates) == 0 {
		return
	}
	a.declaredMu.Lock()
	defer a.declaredMu.Unlock()
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate.Label) == "" {
			continue
		}
		a.declaredByName[candidate.Label] = candidate
	}
}

func (a *RuntimeAssistance) CompleteInput(ctx context.Context, req bobarepl.CompletionRequest) (bobarepl.CompletionResult, error) {
	return a.assist.CompleteInput(ctx, req)
}

func (a *RuntimeAssistance) GetHelpBar(ctx context.Context, req bobarepl.HelpBarRequest) (bobarepl.HelpBarPayload, error) {
	return a.assist.GetHelpBar(ctx, req)
}

func (a *RuntimeAssistance) GetHelpDrawer(ctx context.Context, req bobarepl.HelpDrawerRequest) (bobarepl.HelpDrawerDocument, error) {
	return a.assist.GetHelpDrawer(ctx, req)
}

// Close releases adapter-owned resources. It does not close the runtime.
func (a *RuntimeAssistance) Close() error {
	return nil
}

func (a *RuntimeAssistance) bindingHints() []jsparse.CompletionCandidate {
	if a == nil {
		return nil
	}
	a.declaredMu.RLock()
	defer a.declaredMu.RUnlock()
	if len(a.declaredByName) == 0 {
		return nil
	}
	ret := make([]jsparse.CompletionCandidate, 0, len(a.declaredByName))
	for _, candidate := range a.declaredByName {
		ret = append(ret, candidate)
	}
	return ret
}

var _ bobarepl.InputCompleter = (*RuntimeAssistance)(nil)
var _ bobarepl.HelpBarProvider = (*RuntimeAssistance)(nil)
var _ bobarepl.HelpDrawerProvider = (*RuntimeAssistance)(nil)
