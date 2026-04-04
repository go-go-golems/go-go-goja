package bobatea

import (
	"context"
	"strings"

	bobarepl "github.com/go-go-golems/bobatea/pkg/repl"
	"github.com/go-go-golems/go-go-goja/pkg/replapi"
	"github.com/pkg/errors"
)

// REPLAPIAdapter adapts replapi-backed session execution to the Bobatea
// evaluator contract used by the Bubble Tea REPL model.
type REPLAPIAdapter struct {
	app       *replapi.App
	sessionID string
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
	return &REPLAPIAdapter{
		app:       app,
		sessionID: sessionID,
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

// Close releases adapter resources. The adapter does not own the underlying app.
func (a *REPLAPIAdapter) Close() error {
	return nil
}

var _ bobarepl.Evaluator = (*REPLAPIAdapter)(nil)
