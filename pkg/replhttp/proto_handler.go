package replhttp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-go-golems/go-go-goja/pkg/replapi"
	replapiv1 "github.com/go-go-golems/go-go-goja/pkg/replapi/pb/proto/goja/replapi/v1"
	"github.com/go-go-golems/go-go-goja/pkg/replapi/pbconv"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

// NewHandler returns the protobuf-JSON persistent REPL HTTP handler.
//
// The REPL API now uses the generated goja.replapi.v1 protobuf messages
// directly on /api routes. The previous hand-written encoding/json response
// envelopes were removed before public adoption to keep the server transport
// surface small and schema-first.
func NewHandler(app *replapi.App) (http.Handler, error) {
	if app == nil {
		return nil, errors.New("replhttp: app is nil")
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/sessions", func(w http.ResponseWriter, r *http.Request) {
		sessions, err := app.ListSessions(r.Context())
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err)
			return
		}
		resp, err := pbconv.ListSessionsResponseToProto(sessions)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err)
			return
		}
		writeProtoJSON(w, http.StatusOK, resp)
	})

	mux.HandleFunc("POST /api/sessions", func(w http.ResponseWriter, r *http.Request) {
		summary, err := app.CreateSession(r.Context())
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err)
			return
		}
		writeProtoJSON(w, http.StatusCreated, &replapiv1.CreateSessionResponse{SchemaVersion: pbconv.SchemaVersion, Session: pbconv.SessionSummaryToProto(summary)})
	})

	mux.HandleFunc("GET /api/sessions/{id}", func(w http.ResponseWriter, r *http.Request) {
		summary, err := app.Snapshot(r.Context(), r.PathValue("id"))
		if err != nil {
			writeJSONError(w, statusForError(err), err)
			return
		}
		writeProtoJSON(w, http.StatusOK, &replapiv1.GetSessionResponse{SchemaVersion: pbconv.SchemaVersion, Session: pbconv.SessionSummaryToProto(summary)})
	})

	mux.HandleFunc("DELETE /api/sessions/{id}", func(w http.ResponseWriter, r *http.Request) {
		if err := app.DeleteSession(r.Context(), r.PathValue("id")); err != nil {
			writeJSONError(w, statusForError(err), err)
			return
		}
		writeProtoJSON(w, http.StatusOK, &replapiv1.DeleteSessionResponse{SchemaVersion: pbconv.SchemaVersion, Deleted: true})
	})

	mux.HandleFunc("POST /api/sessions/{id}/evaluate", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeJSONErrorMessage(w, http.StatusBadRequest, "read request body")
			return
		}
		req, err := pbconv.UnmarshalEvaluateRequestJSON(body)
		if err != nil {
			writeJSONErrorMessage(w, http.StatusBadRequest, "invalid protobuf JSON body: "+err.Error())
			return
		}
		internalReq := pbconv.EvaluateRequestFromProto(req)
		resp, err := app.Evaluate(r.Context(), r.PathValue("id"), internalReq.Source)
		if err != nil {
			writeJSONError(w, statusForError(err), err)
			return
		}
		writeProtoJSON(w, http.StatusOK, pbconv.EvaluateResponseToProto(resp))
	})

	mux.HandleFunc("POST /api/sessions/{id}/restore", func(w http.ResponseWriter, r *http.Request) {
		summary, err := app.Restore(r.Context(), r.PathValue("id"))
		if err != nil {
			writeJSONError(w, statusForError(err), err)
			return
		}
		writeProtoJSON(w, http.StatusOK, &replapiv1.RestoreSessionResponse{SchemaVersion: pbconv.SchemaVersion, Session: pbconv.SessionSummaryToProto(summary)})
	})

	mux.HandleFunc("GET /api/sessions/{id}/history", func(w http.ResponseWriter, r *http.Request) {
		history, err := app.History(r.Context(), r.PathValue("id"))
		if err != nil {
			writeJSONError(w, statusForError(err), err)
			return
		}
		converted, err := pbconv.EvaluationRecordsToProto(history)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err)
			return
		}
		writeProtoJSON(w, http.StatusOK, &replapiv1.HistoryResponse{SchemaVersion: pbconv.SchemaVersion, History: converted})
	})

	mux.HandleFunc("GET /api/sessions/{id}/bindings", func(w http.ResponseWriter, r *http.Request) {
		bindings, err := app.Bindings(r.Context(), r.PathValue("id"))
		if err != nil {
			writeJSONError(w, statusForError(err), err)
			return
		}
		writeProtoJSON(w, http.StatusOK, &replapiv1.BindingsResponse{SchemaVersion: pbconv.SchemaVersion, Bindings: pbconv.BindingViewsToProto(bindings)})
	})

	mux.HandleFunc("GET /api/sessions/{id}/docs", func(w http.ResponseWriter, r *http.Request) {
		docs, err := app.Docs(r.Context(), r.PathValue("id"))
		if err != nil {
			writeJSONError(w, statusForError(err), err)
			return
		}
		converted, err := pbconv.BindingDocRecordsToProto(docs)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err)
			return
		}
		writeProtoJSON(w, http.StatusOK, &replapiv1.DocsResponse{SchemaVersion: pbconv.SchemaVersion, Docs: converted})
	})

	mux.HandleFunc("GET /api/sessions/{id}/export", func(w http.ResponseWriter, r *http.Request) {
		exported, err := app.Export(r.Context(), r.PathValue("id"))
		if err != nil {
			writeJSONError(w, statusForError(err), err)
			return
		}
		converted, err := pbconv.SessionExportToProto(exported)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err)
			return
		}
		writeProtoJSON(w, http.StatusOK, &replapiv1.ExportSessionResponse{SchemaVersion: pbconv.SchemaVersion, SessionExport: converted})
	})

	return recoveryMiddleware(mux), nil
}

func writeProtoJSON(w http.ResponseWriter, status int, payload proto.Message) {
	if payload == nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Errorf("nil protobuf response"))
		return
	}
	body, err := pbconv.MarshalJSON(payload)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(json.RawMessage(body))
}
