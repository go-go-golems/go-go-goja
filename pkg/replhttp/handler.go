package replhttp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/go-go-golems/go-go-goja/pkg/replapi"
	replapiv1 "github.com/go-go-golems/go-go-goja/pkg/replapi/pb/proto/goja/replapi/v1"
	"github.com/go-go-golems/go-go-goja/pkg/replapi/pbconv"
	"github.com/go-go-golems/go-go-goja/pkg/repldb"
	"github.com/go-go-golems/go-go-goja/pkg/replsession"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

const (
	DefaultMaxRequestBodyBytes int64 = 1 << 20 // 1 MiB
	DefaultMaxSourceBytes            = 256 << 10
	RequestIDHeader                  = "X-Request-ID"
)

var (
	ErrRequestTooLarge    = errors.New("replhttp: request body too large")
	ErrSourceTooLarge     = errors.New("replhttp: JavaScript source too large")
	ErrUnsupportedVersion = errors.New("replhttp: unsupported schema version")
	ErrInvalidContentType = errors.New("replhttp: Content-Type must be application/json")
	ErrInvalidRequest     = errors.New("replhttp: invalid request")
)

// HandlerConfig controls bounded parsing and internal diagnostics. Authentication
// and authorization remain the responsibility of outer middleware.
type HandlerConfig struct {
	MaxRequestBodyBytes  int64
	MaxSourceBytes       int
	ExposeInternalErrors bool
	Logger               zerolog.Logger
	RequestID            func(*http.Request) string
}

// HandlerOption configures the protobuf-JSON HTTP handler.
type HandlerOption func(*HandlerConfig)

// DefaultHandlerConfig returns bounded, redacted transport defaults.
func DefaultHandlerConfig() HandlerConfig {
	return HandlerConfig{
		MaxRequestBodyBytes: DefaultMaxRequestBodyBytes,
		MaxSourceBytes:      DefaultMaxSourceBytes,
		Logger:              zerolog.Nop(),
		RequestID:           defaultRequestID,
	}
}

// WithMaxRequestBodyBytes sets the evaluate request-body limit in bytes.
func WithMaxRequestBodyBytes(limit int64) HandlerOption {
	return func(config *HandlerConfig) { config.MaxRequestBodyBytes = limit }
}

// WithMaxSourceBytes sets the decoded UTF-8 JavaScript source limit in bytes.
func WithMaxSourceBytes(limit int) HandlerOption {
	return func(config *HandlerConfig) { config.MaxSourceBytes = limit }
}

// WithHandlerLogger sets the server-side diagnostic logger.
func WithHandlerLogger(logger zerolog.Logger) HandlerOption {
	return func(config *HandlerConfig) { config.Logger = logger }
}

// WithExposeInternalErrors is intended only for trusted local debugging.
func WithExposeInternalErrors(expose bool) HandlerOption {
	return func(config *HandlerConfig) { config.ExposeInternalErrors = expose }
}

// WithRequestIDGenerator supplies request IDs before validation and propagation.
func WithRequestIDGenerator(generator func(*http.Request) string) HandlerOption {
	return func(config *HandlerConfig) { config.RequestID = generator }
}

type handlerRuntime struct {
	config HandlerConfig
}

type requestIDContextKey struct{}

type codedError struct {
	status  int
	code    string
	message string
	cause   error
}

func (e *codedError) Error() string {
	if e == nil || e.cause == nil {
		return "replhttp: request failed"
	}
	return e.cause.Error()
}
func (e *codedError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

func newCodedError(status int, code string, message string, cause error) error {
	return &codedError{status: status, code: code, message: message, cause: cause}
}

func (h *handlerRuntime) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := strings.TrimSpace(h.config.RequestID(r))
		if !validRequestID(requestID) {
			requestID = uuid.NewString()
		}
		w.Header().Set(RequestIDHeader, requestID)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		r = r.WithContext(context.WithValue(r.Context(), requestIDContextKey{}, requestID))
		defer func() {
			if recovered := recover(); recovered != nil {
				err := fmt.Errorf("panic: %v", recovered)
				h.config.Logger.Error().Err(err).Bytes("stack", debug.Stack()).Str("request_id", requestID).Str("method", r.Method).Str("path", r.URL.Path).Msg("repl HTTP panic")
				h.writeError(w, r, newCodedError(http.StatusInternalServerError, "internal", "internal server error", err))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func requestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDContextKey{}).(string)
	return requestID
}

func defaultRequestID(r *http.Request) string {
	if r != nil {
		return r.Header.Get(RequestIDHeader)
	}
	return ""
}

func validRequestID(candidate string) bool {
	if candidate == "" || len(candidate) > 128 {
		return false
	}
	for _, char := range candidate {
		if char < 0x21 || char > 0x7e {
			return false
		}
	}
	return true
}

func (h *handlerRuntime) writeError(w http.ResponseWriter, r *http.Request, err error) {
	mapped := mapTransportError(err)
	message := mapped.message
	if h.config.ExposeInternalErrors && mapped.status >= http.StatusInternalServerError && err != nil {
		message = err.Error()
	}
	requestID := requestIDFromContext(r.Context())
	if mapped.status >= http.StatusInternalServerError {
		h.config.Logger.Error().Err(err).Str("request_id", requestID).Str("method", r.Method).Str("path", r.URL.Path).Str("code", mapped.code).Msg("repl HTTP request failed")
	} else {
		h.config.Logger.Warn().Err(err).Str("request_id", requestID).Str("method", r.Method).Str("path", r.URL.Path).Str("code", mapped.code).Msg("repl HTTP request rejected")
	}
	response := &replapiv1.ErrorResponse{
		SchemaVersion: pbconv.SchemaVersion,
		Code:          mapped.code,
		Message:       message,
		RequestId:     requestID,
	}
	body, marshalErr := pbconv.MarshalJSON(response)
	if marshalErr != nil {
		h.config.Logger.Error().Err(marshalErr).Str("request_id", requestID).Msg("marshal repl error response")
		body = []byte(`{"schemaVersion":1,"code":"internal","message":"internal server error"}`)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(mapped.status)
	_, _ = w.Write(body)
}

func mapTransportError(err error) *codedError {
	var explicit *codedError
	if errors.As(err, &explicit) {
		return explicit
	}
	switch {
	case isSessionNotFound(err):
		return &codedError{status: http.StatusNotFound, code: "session_not_found", message: "session not found", cause: err}
	case errors.Is(err, repldb.ErrSessionOwned):
		return &codedError{status: http.StatusConflict, code: "session_owned", message: "session is owned by another app", cause: err}
	case errors.Is(err, repldb.ErrLeaseLost), errors.Is(err, repldb.ErrWriteConflict), errors.Is(err, replsession.ErrSessionDegraded), errors.Is(err, replsession.ErrSessionFenced):
		return &codedError{status: http.StatusConflict, code: "session_not_writable", message: "session is not writable", cause: err}
	case errors.Is(err, replsession.ErrCommitFailed):
		return &codedError{status: http.StatusServiceUnavailable, code: "persistence_unavailable", message: "evaluation executed but could not be committed", cause: err}
	case errors.Is(err, replapi.ErrAppClosing), errors.Is(err, replapi.ErrAppClosed), errors.Is(err, replsession.ErrServiceClosing), errors.Is(err, replsession.ErrServiceClosed), errors.Is(err, replsession.ErrSessionClosing), errors.Is(err, replsession.ErrSessionClosed):
		return &codedError{status: http.StatusServiceUnavailable, code: "service_shutting_down", message: "service is shutting down", cause: err}
	case errors.Is(err, replapi.ErrUnknownProfile), errors.Is(err, replapi.ErrProfileMismatch), errors.Is(err, replapi.ErrInvalidSessionPolicy):
		return &codedError{status: http.StatusBadRequest, code: "invalid_argument", message: "invalid request", cause: err}
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return &codedError{status: http.StatusServiceUnavailable, code: "service_unavailable", message: "service unavailable", cause: err}
	default:
		return &codedError{status: http.StatusInternalServerError, code: "internal", message: "internal server error", cause: err}
	}
}

// isSessionNotFound checks both sentinel values since repldb and replsession
// cannot share one sentinel without an import cycle.
func isSessionNotFound(err error) bool {
	return errors.Is(err, replsession.ErrSessionNotFound) || errors.Is(err, repldb.ErrSessionNotFound)
}
