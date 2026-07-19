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

func newAccessLogResponseWriter(w http.ResponseWriter) (*accessLogResponseWriter, http.ResponseWriter) {
	logger := &accessLogResponseWriter{ResponseWriter: w, status: http.StatusOK}
	return logger, accessLogOptionalInterfaces(logger, w)
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

type accessLogFlusher struct{ *accessLogResponseWriter }

type accessLogHijacker struct{ *accessLogResponseWriter }

type accessLogReaderFrom struct{ *accessLogResponseWriter }

type accessLogPusher struct{ *accessLogResponseWriter }

func (w accessLogFlusher) Flush() {
	w.ResponseWriter.(http.Flusher).Flush()
}

func (w accessLogHijacker) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.ResponseWriter.(http.Hijacker).Hijack()
}

func (w accessLogReaderFrom) ReadFrom(r io.Reader) (int64, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.(io.ReaderFrom).ReadFrom(r)
	w.bytes += n
	return n, err
}

func (w accessLogPusher) Push(target string, opts *http.PushOptions) error {
	return w.ResponseWriter.(http.Pusher).Push(target, opts)
}

func accessLogOptionalInterfaces(logger *accessLogResponseWriter, underlying http.ResponseWriter) http.ResponseWriter {
	_, flusher := underlying.(http.Flusher)
	_, hijacker := underlying.(http.Hijacker)
	_, readerFrom := underlying.(io.ReaderFrom)
	_, pusher := underlying.(http.Pusher)

	switch {
	case flusher && hijacker && readerFrom && pusher:
		return struct {
			*accessLogResponseWriter
			accessLogFlusher
			accessLogHijacker
			accessLogReaderFrom
			accessLogPusher
		}{logger, accessLogFlusher{logger}, accessLogHijacker{logger}, accessLogReaderFrom{logger}, accessLogPusher{logger}}
	case flusher && hijacker && readerFrom:
		return struct {
			*accessLogResponseWriter
			accessLogFlusher
			accessLogHijacker
			accessLogReaderFrom
		}{logger, accessLogFlusher{logger}, accessLogHijacker{logger}, accessLogReaderFrom{logger}}
	case flusher && hijacker && pusher:
		return struct {
			*accessLogResponseWriter
			accessLogFlusher
			accessLogHijacker
			accessLogPusher
		}{logger, accessLogFlusher{logger}, accessLogHijacker{logger}, accessLogPusher{logger}}
	case flusher && readerFrom && pusher:
		return struct {
			*accessLogResponseWriter
			accessLogFlusher
			accessLogReaderFrom
			accessLogPusher
		}{logger, accessLogFlusher{logger}, accessLogReaderFrom{logger}, accessLogPusher{logger}}
	case hijacker && readerFrom && pusher:
		return struct {
			*accessLogResponseWriter
			accessLogHijacker
			accessLogReaderFrom
			accessLogPusher
		}{logger, accessLogHijacker{logger}, accessLogReaderFrom{logger}, accessLogPusher{logger}}
	case flusher && hijacker:
		return struct {
			*accessLogResponseWriter
			accessLogFlusher
			accessLogHijacker
		}{logger, accessLogFlusher{logger}, accessLogHijacker{logger}}
	case flusher && readerFrom:
		return struct {
			*accessLogResponseWriter
			accessLogFlusher
			accessLogReaderFrom
		}{logger, accessLogFlusher{logger}, accessLogReaderFrom{logger}}
	case flusher && pusher:
		return struct {
			*accessLogResponseWriter
			accessLogFlusher
			accessLogPusher
		}{logger, accessLogFlusher{logger}, accessLogPusher{logger}}
	case hijacker && readerFrom:
		return struct {
			*accessLogResponseWriter
			accessLogHijacker
			accessLogReaderFrom
		}{logger, accessLogHijacker{logger}, accessLogReaderFrom{logger}}
	case hijacker && pusher:
		return struct {
			*accessLogResponseWriter
			accessLogHijacker
			accessLogPusher
		}{logger, accessLogHijacker{logger}, accessLogPusher{logger}}
	case readerFrom && pusher:
		return struct {
			*accessLogResponseWriter
			accessLogReaderFrom
			accessLogPusher
		}{logger, accessLogReaderFrom{logger}, accessLogPusher{logger}}
	case flusher:
		return struct {
			*accessLogResponseWriter
			accessLogFlusher
		}{logger, accessLogFlusher{logger}}
	case hijacker:
		return struct {
			*accessLogResponseWriter
			accessLogHijacker
		}{logger, accessLogHijacker{logger}}
	case readerFrom:
		return struct {
			*accessLogResponseWriter
			accessLogReaderFrom
		}{logger, accessLogReaderFrom{logger}}
	case pusher:
		return struct {
			*accessLogResponseWriter
			accessLogPusher
		}{logger, accessLogPusher{logger}}
	default:
		return logger
	}
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
		Str("client_ip", RequestClientIP(r)).
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
