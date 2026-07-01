package replhttp

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-go-golems/go-go-goja/pkg/repldb"
	"github.com/go-go-golems/go-go-goja/pkg/replsession"
	"github.com/pkg/errors"
)

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

// isSessionNotFound checks both sentinel error values since repldb and
// replsession cannot share a single sentinel (import cycle).
func isSessionNotFound(err error) bool {
	return errors.Is(err, replsession.ErrSessionNotFound) || errors.Is(err, repldb.ErrSessionNotFound)
}

func statusForError(err error) int {
	switch {
	case isSessionNotFound(err):
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
