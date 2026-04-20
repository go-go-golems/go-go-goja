package replhttp

import (
	"encoding/json"
	"fmt"
	"net/http"

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

	// --- /api/sessions ---
	mux.HandleFunc("GET /api/sessions", func(w http.ResponseWriter, r *http.Request) {
		sessions, err := app.ListSessions(r.Context())
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"sessions": sessions})
	})

	mux.HandleFunc("POST /api/sessions", func(w http.ResponseWriter, r *http.Request) {
		summary, err := app.CreateSession(r.Context())
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"session": summary})
	})

	// --- /api/sessions/{id} ---
	mux.HandleFunc("GET /api/sessions/{id}", func(w http.ResponseWriter, r *http.Request) {
		summary, err := app.Snapshot(r.Context(), r.PathValue("id"))
		if err != nil {
			writeJSONError(w, statusForError(err), err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"session": summary})
	})

	mux.HandleFunc("DELETE /api/sessions/{id}", func(w http.ResponseWriter, r *http.Request) {
		if err := app.DeleteSession(r.Context(), r.PathValue("id")); err != nil {
			writeJSONError(w, statusForError(err), err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
	})

	// --- /api/sessions/{id}/evaluate ---
	mux.HandleFunc("POST /api/sessions/{id}/evaluate", func(w http.ResponseWriter, r *http.Request) {
		var req replsession.EvaluateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONErrorMessage(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		resp, err := app.Evaluate(r.Context(), r.PathValue("id"), req.Source)
		if err != nil {
			writeJSONError(w, statusForError(err), err)
			return
		}
		writeJSON(w, http.StatusOK, resp)
	})

	// --- /api/sessions/{id}/restore ---
	mux.HandleFunc("POST /api/sessions/{id}/restore", func(w http.ResponseWriter, r *http.Request) {
		summary, err := app.Restore(r.Context(), r.PathValue("id"))
		if err != nil {
			writeJSONError(w, statusForError(err), err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"session": summary})
	})

	// --- /api/sessions/{id}/history ---
	mux.HandleFunc("GET /api/sessions/{id}/history", func(w http.ResponseWriter, r *http.Request) {
		history, err := app.History(r.Context(), r.PathValue("id"))
		if err != nil {
			writeJSONError(w, statusForError(err), err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"history": history})
	})

	// --- /api/sessions/{id}/bindings ---
	mux.HandleFunc("GET /api/sessions/{id}/bindings", func(w http.ResponseWriter, r *http.Request) {
		bindings, err := app.Bindings(r.Context(), r.PathValue("id"))
		if err != nil {
			writeJSONError(w, statusForError(err), err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"bindings": bindings})
	})

	// --- /api/sessions/{id}/docs ---
	mux.HandleFunc("GET /api/sessions/{id}/docs", func(w http.ResponseWriter, r *http.Request) {
		docs, err := app.Docs(r.Context(), r.PathValue("id"))
		if err != nil {
			writeJSONError(w, statusForError(err), err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"docs": docs})
	})

	// --- /api/sessions/{id}/export ---
	mux.HandleFunc("GET /api/sessions/{id}/export", func(w http.ResponseWriter, r *http.Request) {
		exported, err := app.Export(r.Context(), r.PathValue("id"))
		if err != nil {
			writeJSONError(w, statusForError(err), err)
			return
		}
		writeJSON(w, http.StatusOK, exported)
	})

	// Recovery middleware: catch panics in handlers and return 500 JSON
	// instead of crashing the connection.
	return recoveryMiddleware(mux), nil
}

func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				writeJSONError(w, http.StatusInternalServerError, fmt.Errorf("internal error: %v", rec))
			}
		}()
		next.ServeHTTP(w, r)
	})
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
