package replhttp

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/go-go-golems/go-go-goja/pkg/replapi"
	replapiv1 "github.com/go-go-golems/go-go-goja/pkg/replapi/pb/proto/goja/replapi/v1"
	"github.com/go-go-golems/go-go-goja/pkg/replapi/pbconv"
	"google.golang.org/protobuf/proto"
)

// NewHandler returns the bounded protobuf-JSON REPL HTTP transport.
// Authentication and authorization are intentionally supplied by outer middleware.
func NewHandler(app *replapi.App, opts ...HandlerOption) (http.Handler, error) {
	if app == nil {
		return nil, fmt.Errorf("replhttp: app is nil")
	}
	config := DefaultHandlerConfig()
	for _, opt := range opts {
		if opt != nil {
			opt(&config)
		}
	}
	if config.MaxRequestBodyBytes <= 0 {
		return nil, fmt.Errorf("replhttp: maximum request body must be positive")
	}
	if config.MaxSourceBytes <= 0 || int64(config.MaxSourceBytes) > config.MaxRequestBodyBytes {
		return nil, fmt.Errorf("replhttp: maximum source size must be positive and no larger than request body limit")
	}
	if config.RequestID == nil {
		config.RequestID = defaultRequestID
	}
	h := &handlerRuntime{config: config}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/sessions", func(w http.ResponseWriter, r *http.Request) {
		sessions, err := app.ListSessions(r.Context())
		if err != nil {
			h.writeError(w, r, err)
			return
		}
		resp, err := pbconv.ListSessionsResponseToProto(sessions)
		if err != nil {
			h.writeError(w, r, err)
			return
		}
		h.writeProto(w, r, http.StatusOK, resp)
	})

	mux.HandleFunc("POST /api/sessions", func(w http.ResponseWriter, r *http.Request) {
		summary, err := app.CreateSession(r.Context())
		if err != nil {
			h.writeError(w, r, err)
			return
		}
		h.writeProto(w, r, http.StatusCreated, &replapiv1.CreateSessionResponse{SchemaVersion: pbconv.SchemaVersion, Session: pbconv.SessionSummaryToProto(summary)})
	})

	mux.HandleFunc("GET /api/sessions/{id}", func(w http.ResponseWriter, r *http.Request) {
		summary, err := app.Snapshot(r.Context(), r.PathValue("id"))
		if err != nil {
			h.writeError(w, r, err)
			return
		}
		h.writeProto(w, r, http.StatusOK, &replapiv1.GetSessionResponse{SchemaVersion: pbconv.SchemaVersion, Session: pbconv.SessionSummaryToProto(summary)})
	})

	mux.HandleFunc("DELETE /api/sessions/{id}", func(w http.ResponseWriter, r *http.Request) {
		if err := app.DeleteSession(r.Context(), r.PathValue("id")); err != nil {
			h.writeError(w, r, err)
			return
		}
		h.writeProto(w, r, http.StatusOK, &replapiv1.DeleteSessionResponse{SchemaVersion: pbconv.SchemaVersion, Deleted: true})
	})

	mux.HandleFunc("POST /api/sessions/{id}/evaluate", func(w http.ResponseWriter, r *http.Request) {
		if err := validateJSONContentType(r.Header.Get("Content-Type")); err != nil {
			h.writeError(w, r, err)
			return
		}
		if r.ContentLength > config.MaxRequestBodyBytes {
			h.writeError(w, r, newCodedError(http.StatusRequestEntityTooLarge, "request_too_large", "request body is too large", ErrRequestTooLarge))
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, config.MaxRequestBodyBytes)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				h.writeError(w, r, newCodedError(http.StatusRequestEntityTooLarge, "request_too_large", "request body is too large", ErrRequestTooLarge))
				return
			}
			h.writeError(w, r, newCodedError(http.StatusBadRequest, "invalid_argument", "invalid request body", fmt.Errorf("%w: read body: %v", ErrInvalidRequest, err)))
			return
		}
		req, err := pbconv.UnmarshalEvaluateRequestJSON(body)
		if err != nil {
			h.writeError(w, r, newCodedError(http.StatusBadRequest, "invalid_argument", "invalid protobuf JSON body", fmt.Errorf("%w: %v", ErrInvalidRequest, err)))
			return
		}
		if req.GetSchemaVersion() != pbconv.SchemaVersion {
			h.writeError(w, r, newCodedError(http.StatusBadRequest, "unsupported_schema_version", "unsupported schema version", ErrUnsupportedVersion))
			return
		}
		if len([]byte(req.GetSource())) > config.MaxSourceBytes {
			h.writeError(w, r, newCodedError(http.StatusRequestEntityTooLarge, "source_too_large", "JavaScript source is too large", ErrSourceTooLarge))
			return
		}
		internalReq := pbconv.EvaluateRequestFromProto(req)
		resp, err := app.Evaluate(r.Context(), r.PathValue("id"), internalReq.Source)
		if err != nil {
			h.writeError(w, r, err)
			return
		}
		h.writeProto(w, r, http.StatusOK, pbconv.EvaluateResponseToProto(resp))
	})

	mux.HandleFunc("POST /api/sessions/{id}/restore", func(w http.ResponseWriter, r *http.Request) {
		summary, err := app.Restore(r.Context(), r.PathValue("id"))
		if err != nil {
			h.writeError(w, r, err)
			return
		}
		h.writeProto(w, r, http.StatusOK, &replapiv1.RestoreSessionResponse{SchemaVersion: pbconv.SchemaVersion, Session: pbconv.SessionSummaryToProto(summary)})
	})

	mux.HandleFunc("GET /api/sessions/{id}/history", func(w http.ResponseWriter, r *http.Request) {
		history, err := app.History(r.Context(), r.PathValue("id"))
		if err != nil {
			h.writeError(w, r, err)
			return
		}
		converted, err := pbconv.EvaluationRecordsToProto(history)
		if err != nil {
			h.writeError(w, r, err)
			return
		}
		h.writeProto(w, r, http.StatusOK, &replapiv1.HistoryResponse{SchemaVersion: pbconv.SchemaVersion, History: converted})
	})

	mux.HandleFunc("GET /api/sessions/{id}/bindings", func(w http.ResponseWriter, r *http.Request) {
		bindings, err := app.Bindings(r.Context(), r.PathValue("id"))
		if err != nil {
			h.writeError(w, r, err)
			return
		}
		h.writeProto(w, r, http.StatusOK, &replapiv1.BindingsResponse{SchemaVersion: pbconv.SchemaVersion, Bindings: pbconv.BindingViewsToProto(bindings)})
	})

	mux.HandleFunc("GET /api/sessions/{id}/docs", func(w http.ResponseWriter, r *http.Request) {
		docs, err := app.Docs(r.Context(), r.PathValue("id"))
		if err != nil {
			h.writeError(w, r, err)
			return
		}
		converted, err := pbconv.BindingDocRecordsToProto(docs)
		if err != nil {
			h.writeError(w, r, err)
			return
		}
		h.writeProto(w, r, http.StatusOK, &replapiv1.DocsResponse{SchemaVersion: pbconv.SchemaVersion, Docs: converted})
	})

	mux.HandleFunc("GET /api/sessions/{id}/export", func(w http.ResponseWriter, r *http.Request) {
		exported, err := app.Export(r.Context(), r.PathValue("id"))
		if err != nil {
			h.writeError(w, r, err)
			return
		}
		converted, err := pbconv.SessionExportToProto(exported)
		if err != nil {
			h.writeError(w, r, err)
			return
		}
		h.writeProto(w, r, http.StatusOK, &replapiv1.ExportSessionResponse{SchemaVersion: pbconv.SchemaVersion, SessionExport: converted})
	})

	return h.middleware(mux), nil
}

func validateJSONContentType(value string) error {
	mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(value))
	if err != nil || !strings.EqualFold(mediaType, "application/json") {
		return newCodedError(http.StatusBadRequest, "invalid_content_type", "Content-Type must be application/json", ErrInvalidContentType)
	}
	return nil
}

func (h *handlerRuntime) writeProto(w http.ResponseWriter, r *http.Request, status int, payload proto.Message) {
	if payload == nil {
		h.writeError(w, r, fmt.Errorf("nil protobuf response"))
		return
	}
	body, err := pbconv.MarshalJSON(payload)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	// #nosec G705 -- body is emitted by protojson, served as application/json,
	// and protected by X-Content-Type-Options: nosniff.
	_, _ = w.Write(body)
}
