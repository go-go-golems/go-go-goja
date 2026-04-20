package replhttp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-go-golems/go-go-goja/pkg/replapi"
	"github.com/go-go-golems/go-go-goja/pkg/repldb"
	"github.com/go-go-golems/go-go-goja/pkg/replsession"
	"github.com/pkg/errors"
)

// NewHandler returns a JSON-only persistent REPL HTTP handler.
func NewHandler(app *replapi.App) (http.Handler, error) {
	if app == nil {
		return nil, errors.New("replhttp: app is nil")
	}

	mux := http.NewServeMux()

	// Recovery middleware: catch panics in handlers and return 500 JSON
	// instead of crashing the connection.
	recoverWrapper := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					writeJSONError(w, http.StatusInternalServerError, fmt.Errorf("internal error: %v", rec))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
	mux.HandleFunc("/api/sessions", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			sessions, err := app.ListSessions(r.Context())
			if err != nil {
				writeJSONError(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"sessions": sessions})
		case http.MethodPost:
			summary, err := app.CreateSession(r.Context())
			if err != nil {
				writeJSONError(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusCreated, map[string]any{"session": summary})
		default:
			writeJSONErrorMessage(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})
	mux.HandleFunc("/api/sessions/", func(w http.ResponseWriter, r *http.Request) {
		trimmed := strings.TrimPrefix(r.URL.Path, "/api/sessions/")
		parts := strings.Split(strings.Trim(trimmed, "/"), "/")
		if len(parts) == 0 || parts[0] == "" {
			writeJSONErrorMessage(w, http.StatusNotFound, "session id missing")
			return
		}
		sessionID := parts[0]
		if len(parts) == 1 {
			switch r.Method {
			case http.MethodGet:
				summary, err := app.Snapshot(r.Context(), sessionID)
				if err != nil {
					writeJSONError(w, statusForError(err), err)
					return
				}
				writeJSON(w, http.StatusOK, map[string]any{"session": summary})
			case http.MethodDelete:
				if err := app.DeleteSession(r.Context(), sessionID); err != nil {
					writeJSONError(w, statusForError(err), err)
					return
				}
				writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
			default:
				writeJSONErrorMessage(w, http.StatusMethodNotAllowed, "method not allowed")
			}
			return
		}

		if len(parts) != 2 {
			writeJSONErrorMessage(w, http.StatusNotFound, "route not found")
			return
		}

		switch parts[1] {
		case "evaluate":
			if r.Method != http.MethodPost {
				writeJSONErrorMessage(w, http.StatusMethodNotAllowed, "method not allowed")
				return
			}
			var req replsession.EvaluateRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeJSONErrorMessage(w, http.StatusBadRequest, "invalid JSON body")
				return
			}
			resp, err := app.Evaluate(r.Context(), sessionID, req.Source)
			if err != nil {
				writeJSONError(w, statusForError(err), err)
				return
			}
			writeJSON(w, http.StatusOK, resp)
		case "restore":
			if r.Method != http.MethodPost {
				writeJSONErrorMessage(w, http.StatusMethodNotAllowed, "method not allowed")
				return
			}
			summary, err := app.Restore(r.Context(), sessionID)
			if err != nil {
				writeJSONError(w, statusForError(err), err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"session": summary})
		case "history":
			if r.Method != http.MethodGet {
				writeJSONErrorMessage(w, http.StatusMethodNotAllowed, "method not allowed")
				return
			}
			history, err := app.History(r.Context(), sessionID)
			if err != nil {
				writeJSONError(w, statusForError(err), err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"history": history})
		case "bindings":
			if r.Method != http.MethodGet {
				writeJSONErrorMessage(w, http.StatusMethodNotAllowed, "method not allowed")
				return
			}
			bindings, err := app.Bindings(r.Context(), sessionID)
			if err != nil {
				writeJSONError(w, statusForError(err), err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"bindings": bindings})
		case "docs":
			if r.Method != http.MethodGet {
				writeJSONErrorMessage(w, http.StatusMethodNotAllowed, "method not allowed")
				return
			}
			docs, err := app.Docs(r.Context(), sessionID)
			if err != nil {
				writeJSONError(w, statusForError(err), err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"docs": docs})
		case "export":
			if r.Method != http.MethodGet {
				writeJSONErrorMessage(w, http.StatusMethodNotAllowed, "method not allowed")
				return
			}
			exported, err := app.Export(r.Context(), sessionID)
			if err != nil {
				writeJSONError(w, statusForError(err), err)
				return
			}
			writeJSON(w, http.StatusOK, exported)
		default:
			writeJSONErrorMessage(w, http.StatusNotFound, "route not found")
		}
	})
	return recoverWrapper(mux), nil
}

func statusForError(err error) int {
	switch {
	case errors.Is(err, replsession.ErrSessionNotFound), errors.Is(err, repldb.ErrSessionNotFound):
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeJSONError(w http.ResponseWriter, status int, err error) {
	writeJSONErrorMessage(w, status, err.Error())
}

func writeJSONErrorMessage(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"error": message})
}
