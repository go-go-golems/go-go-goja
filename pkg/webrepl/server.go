package webrepl

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"

	"github.com/go-go-golems/go-go-goja/pkg/replsession"
	"github.com/pkg/errors"
)

//go:embed static/*
var staticFS embed.FS

// NewHandler returns the HTTP handler exposing the web UI and JSON API.
func NewHandler(service *replsession.Service) (http.Handler, error) {
	if service == nil {
		return nil, errors.New("webrepl: service is nil")
	}
	staticSub, err := fs.Sub(staticFS, "static")
	if err != nil {
		return nil, errors.Wrap(err, "sub static fs")
	}

	mux := http.NewServeMux()
	fileServer := http.FileServer(http.FS(staticSub))
	mux.Handle("/static/", http.StripPrefix("/static/", fileServer))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && r.URL.Path != "/index.html" {
			http.NotFound(w, r)
			return
		}
		data, readErr := fs.ReadFile(staticSub, "index.html")
		if readErr != nil {
			http.Error(w, readErr.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(data)
	})
	mux.HandleFunc("/api/sessions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		summary, createErr := service.CreateSession(r.Context())
		if createErr != nil {
			writeJSONError(w, http.StatusInternalServerError, createErr.Error())
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"session": summary})
	})
	mux.HandleFunc("/api/sessions/", func(w http.ResponseWriter, r *http.Request) {
		trimmed := strings.TrimPrefix(r.URL.Path, "/api/sessions/")
		parts := strings.Split(strings.Trim(trimmed, "/"), "/")
		if len(parts) == 0 || parts[0] == "" {
			writeJSONError(w, http.StatusNotFound, "session id missing")
			return
		}
		sessionID := parts[0]
		if len(parts) == 1 {
			switch r.Method {
			case http.MethodGet:
				summary, snapErr := service.Snapshot(r.Context(), sessionID)
				if snapErr != nil {
					status := http.StatusInternalServerError
					if errors.Is(snapErr, replsession.ErrSessionNotFound) {
						status = http.StatusNotFound
					}
					writeJSONError(w, status, snapErr.Error())
					return
				}
				writeJSON(w, http.StatusOK, map[string]any{"session": summary})
			case http.MethodDelete:
				if delErr := service.DeleteSession(r.Context(), sessionID); delErr != nil {
					status := http.StatusInternalServerError
					if errors.Is(delErr, replsession.ErrSessionNotFound) {
						status = http.StatusNotFound
					}
					writeJSONError(w, status, delErr.Error())
					return
				}
				writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
			default:
				writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			}
			return
		}
		if len(parts) == 2 && parts[1] == "evaluate" {
			if r.Method != http.MethodPost {
				writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
				return
			}
			var req replsession.EvaluateRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
				return
			}
			resp, evalErr := service.Evaluate(r.Context(), sessionID, req.Source)
			if evalErr != nil {
				status := http.StatusInternalServerError
				if errors.Is(evalErr, replsession.ErrSessionNotFound) {
					status = http.StatusNotFound
				}
				writeJSONError(w, status, evalErr.Error())
				return
			}
			writeJSON(w, http.StatusOK, resp)
			return
		}
		writeJSONError(w, http.StatusNotFound, "route not found")
	})
	return mux, nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"error": message})
}
