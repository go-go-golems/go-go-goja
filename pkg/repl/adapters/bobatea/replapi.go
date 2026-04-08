package bobatea

import (
	"context"
	"strings"

	"github.com/dop251/goja"
	bobarepl "github.com/go-go-golems/bobatea/pkg/repl"
	"github.com/go-go-golems/go-go-goja/engine"
	"github.com/go-go-golems/go-go-goja/pkg/docaccess"
	"github.com/go-go-golems/go-go-goja/pkg/jsparse"
	js "github.com/go-go-golems/go-go-goja/pkg/repl/evaluators/javascript"
	"github.com/go-go-golems/go-go-goja/pkg/replapi"
	"github.com/go-go-golems/go-go-goja/pkg/replsession"
	"github.com/pkg/errors"
)

// REPLAPIAdapter adapts replapi-backed session execution to the Bobatea
// evaluator contract used by the Bubble Tea REPL model.
//
// This adapter is the bridge between the Bobatea/TUI evaluator surface and the
// replapi/replsession kernel. It exists on purpose; it is not dead compatibility
// code.
type REPLAPIAdapter struct {
	app       *replapi.App
	sessionID string
	assist    *js.Assistance
}

// NewREPLAPIAdapter creates a Bobatea evaluator backed by one replapi session.
func NewREPLAPIAdapter(app *replapi.App, sessionID string) (*REPLAPIAdapter, error) {
	if app == nil {
		return nil, errors.New("replapi adapter: app is nil")
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, errors.New("replapi adapter: session id is empty")
	}
	tsParser, err := jsparse.NewTSParser()
	if err != nil {
		return nil, errors.Wrap(err, "replapi adapter: create TypeScript parser")
	}
	return &REPLAPIAdapter{
		app:       app,
		sessionID: sessionID,
		assist: js.NewAssistance(js.AssistanceConfig{
			TSParser: tsParser,
			WithRuntime: func(ctx context.Context, fn func(*goja.Runtime, *docaccess.Hub) error) error {
				return app.WithRuntime(ctx, sessionID, func(runtime *engine.Runtime) error {
					return fn(runtime.VM, js.DocHubFromRuntime(runtime))
				})
			},
			BindingHints: func(ctx context.Context) ([]jsparse.CompletionCandidate, error) {
				bindings, err := app.Bindings(ctx, sessionID)
				if err != nil {
					return nil, nil
				}
				return bindingHintCandidates(bindings), nil
			},
		}),
	}, nil
}

// App returns the underlying replapi app for advanced integrations.
func (a *REPLAPIAdapter) App() *replapi.App {
	if a == nil {
		return nil
	}
	return a.app
}

// SessionID returns the current target session.
func (a *REPLAPIAdapter) SessionID() string {
	if a == nil {
		return ""
	}
	return a.sessionID
}

func (a *REPLAPIAdapter) EvaluateStream(ctx context.Context, code string, emit func(bobarepl.Event)) error {
	if a == nil || a.app == nil {
		return errors.New("replapi adapter: app is nil")
	}
	resp, err := a.app.Evaluate(ctx, a.sessionID, code)
	if err != nil {
		emit(bobarepl.Event{
			Kind:  bobarepl.EventStderr,
			Props: map[string]any{"text": err.Error(), "append": err.Error(), "is_error": true},
		})
		return nil
	}
	if resp == nil || resp.Cell == nil {
		return nil
	}
	for _, event := range resp.Cell.Execution.Console {
		props := map[string]any{
			"text":   event.Message,
			"append": event.Message,
		}
		switch strings.ToLower(strings.TrimSpace(event.Kind)) {
		case "error", "warn":
			props["is_error"] = true
			emit(bobarepl.Event{Kind: bobarepl.EventStderr, Props: props})
		default:
			emit(bobarepl.Event{Kind: bobarepl.EventStdout, Props: props})
		}
	}
	if resp.Cell.Execution.Error != "" {
		emit(bobarepl.Event{
			Kind:  bobarepl.EventStderr,
			Props: map[string]any{"text": resp.Cell.Execution.Error, "append": resp.Cell.Execution.Error, "is_error": true},
		})
		return nil
	}
	if resp.Cell.Execution.Result != "" {
		emit(bobarepl.Event{
			Kind:  bobarepl.EventResultMarkdown,
			Props: map[string]any{"markdown": resp.Cell.Execution.Result},
		})
	}
	return nil
}

func (a *REPLAPIAdapter) GetPrompt() string {
	return "js>"
}

func (a *REPLAPIAdapter) GetName() string {
	return "JavaScript"
}

func (a *REPLAPIAdapter) SupportsMultiline() bool {
	return true
}

func (a *REPLAPIAdapter) GetFileExtension() string {
	return ".js"
}

func (a *REPLAPIAdapter) CompleteInput(ctx context.Context, req bobarepl.CompletionRequest) (bobarepl.CompletionResult, error) {
	return a.assist.CompleteInput(ctx, req)
}

func (a *REPLAPIAdapter) GetHelpBar(ctx context.Context, req bobarepl.HelpBarRequest) (bobarepl.HelpBarPayload, error) {
	return a.assist.GetHelpBar(ctx, req)
}

func (a *REPLAPIAdapter) GetHelpDrawer(ctx context.Context, req bobarepl.HelpDrawerRequest) (bobarepl.HelpDrawerDocument, error) {
	return a.assist.GetHelpDrawer(ctx, req)
}

// Close releases adapter resources. The adapter does not own the underlying app.
func (a *REPLAPIAdapter) Close() error {
	return nil
}

var _ bobarepl.Evaluator = (*REPLAPIAdapter)(nil)
var _ bobarepl.InputCompleter = (*REPLAPIAdapter)(nil)
var _ bobarepl.HelpBarProvider = (*REPLAPIAdapter)(nil)
var _ bobarepl.HelpDrawerProvider = (*REPLAPIAdapter)(nil)

func bindingHintCandidates(bindings []replsession.BindingView) []jsparse.CompletionCandidate {
	if len(bindings) == 0 {
		return nil
	}
	out := make([]jsparse.CompletionCandidate, 0, len(bindings))
	for _, binding := range bindings {
		name := strings.TrimSpace(binding.Name)
		if name == "" {
			continue
		}
		kind := jsparse.CandidateVariable
		switch strings.ToLower(strings.TrimSpace(binding.Kind)) {
		case "function", "class":
			kind = jsparse.CandidateFunction
		}
		detail := strings.TrimSpace(binding.Kind)
		if detail == "" {
			detail = "session binding"
		}
		out = append(out, jsparse.CompletionCandidate{
			Label:  name,
			Kind:   kind,
			Detail: detail,
		})
	}
	return out
}
