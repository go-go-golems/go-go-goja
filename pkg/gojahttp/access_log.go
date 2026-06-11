package gojahttp

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
)

const requestIDHeader = "X-Request-Id"

type accessLogResponseWriter struct {
	http.ResponseWriter
	status      int
	bytes       int64
	wroteHeader bool
}

func newAccessLogResponseWriter(w http.ResponseWriter) *accessLogResponseWriter {
	return &accessLogResponseWriter{ResponseWriter: w, status: http.StatusOK}
}

func (w *accessLogResponseWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.status = status
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *accessLogResponseWriter) Write(p []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(p)
	w.bytes += int64(n)
	return n, err
}

func (w *accessLogResponseWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *accessLogResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	return hijacker.Hijack()
}

func (w *accessLogResponseWriter) ReadFrom(r io.Reader) (int64, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	if readerFrom, ok := w.ResponseWriter.(io.ReaderFrom); ok {
		n, err := readerFrom.ReadFrom(r)
		w.bytes += n
		return n, err
	}
	n, err := io.Copy(w.ResponseWriter, r)
	w.bytes += n
	return n, err
}

func (w *accessLogResponseWriter) Push(target string, opts *http.PushOptions) error {
	pusher, ok := w.ResponseWriter.(http.Pusher)
	if !ok {
		return http.ErrNotSupported
	}
	return pusher.Push(target, opts)
}

func (w *accessLogResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func ensureRequestID(w http.ResponseWriter, r *http.Request) string {
	requestID := r.Header.Get(requestIDHeader)
	if requestID == "" {
		requestID = uuid.NewString()
	}
	w.Header().Set(requestIDHeader, requestID)
	return requestID
}

func requestLogger(r *http.Request, requestID string) zerolog.Logger {
	return zlog.With().
		Str("component", "gojahttp").
		Str("request_id", requestID).
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Str("query", r.URL.RawQuery).
		Str("remote_addr", r.RemoteAddr).
		Str("user_agent", r.UserAgent()).
		Int64("request_content_length", r.ContentLength).
		Logger()
}

func logRequestDone(logger zerolog.Logger, rw *accessLogResponseWriter, started time.Time) {
	event := logger.Info()
	if rw.status >= 500 {
		event = logger.Error()
	} else if rw.status >= 400 {
		event = logger.Warn()
	}
	event.
		Str("event", "http_request_completed").
		Int("status", rw.status).
		Int64("response_bytes", rw.bytes).
		Dur("duration", time.Since(started)).
		Msg("http request completed")
}
