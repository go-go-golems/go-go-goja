package gojahttp

import (
	"context"
	"net/http"
)

func (h *Host) servePlannedHTTP(w http.ResponseWriter, r *http.Request, route Route, req *RequestDTO, loggingWriter *accessLogResponseWriter) {
	h.enforcer.servePlannedHTTP(w, r, route.Plan, req, route.HTTPHandler, loggingWriter)
}

// PlannedHandlerFunc adapts a function to PlannedHTTPHandler.
type PlannedHandlerFunc func(context.Context, *SecureContext, http.ResponseWriter, *http.Request) error

func (f PlannedHandlerFunc) HandlePlannedHTTP(ctx context.Context, sec *SecureContext, w http.ResponseWriter, r *http.Request) error {
	return f(ctx, sec, w, r)
}
