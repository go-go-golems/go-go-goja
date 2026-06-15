package gojahttp

import (
	"context"
	"fmt"
	"net/http"
)

func (h *Host) servePlannedHTTP(w http.ResponseWriter, r *http.Request, route Route, req *RequestDTO, loggingWriter *accessLogResponseWriter) {
	envelope, status, err := h.buildSecureEnvelope(r.Context(), r, req, route.Plan)
	if err != nil {
		h.recordAudit(r.Context(), r, req, route.Plan, envelope, "denied", status, err)
		h.writePlannedHTTPError(w, loggingWriter, status, err)
		return
	}
	h.recordAudit(r.Context(), r, req, route.Plan, envelope, "allowed", 0, nil)
	if route.HTTPHandler == nil {
		err := fmt.Errorf("planned HTTP route %s %s has nil handler", route.Method, route.Pattern)
		h.recordAudit(r.Context(), r, req, route.Plan, envelope, "failed", http.StatusInternalServerError, err)
		h.writePlannedHTTPError(w, loggingWriter, http.StatusInternalServerError, err)
		return
	}
	if err := route.HTTPHandler(r.Context(), envelope.SecureContext, w, r); err != nil {
		h.recordAudit(r.Context(), r, req, route.Plan, envelope, "failed", http.StatusInternalServerError, err)
		h.writePlannedHTTPError(w, loggingWriter, http.StatusInternalServerError, err)
		return
	}
	h.recordAudit(r.Context(), r, req, route.Plan, envelope, "completed", loggingWriter.status, nil)
}

func (h *Host) writePlannedHTTPError(w http.ResponseWriter, loggingWriter *accessLogResponseWriter, status int, err error) {
	if loggingWriter != nil && loggingWriter.wroteHeader {
		return
	}
	if status == 0 {
		status = http.StatusInternalServerError
	}
	message := http.StatusText(status)
	if h.dev && err != nil && status >= 500 {
		message = err.Error()
	}
	http.Error(w, message, status)
}

// PlannedHandlerFunc adapts a function to PlannedHTTPHandler.
type PlannedHandlerFunc func(context.Context, *SecureContext, http.ResponseWriter, *http.Request) error

func (f PlannedHandlerFunc) HandlePlannedHTTP(ctx context.Context, sec *SecureContext, w http.ResponseWriter, r *http.Request) error {
	return f(ctx, sec, w, r)
}
