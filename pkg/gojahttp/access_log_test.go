package gojahttp

import (
	"bytes"
	"io"
	"net/http"
	"testing"
)

type accessLogTestResponseWriter struct {
	header http.Header
	body   bytes.Buffer
	status int
}

func (w *accessLogTestResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func (w *accessLogTestResponseWriter) Write(p []byte) (int, error) {
	return w.body.Write(p)
}

func (w *accessLogTestResponseWriter) WriteHeader(status int) {
	w.status = status
}

type accessLogTestFlusher struct {
	*accessLogTestResponseWriter
	flushed bool
}

func (w *accessLogTestFlusher) Flush() {
	w.flushed = true
}

type accessLogTestReaderFrom struct {
	*accessLogTestResponseWriter
}

func (w *accessLogTestReaderFrom) ReadFrom(r io.Reader) (int64, error) {
	return w.body.ReadFrom(r)
}

func TestAccessLogResponseWriterDoesNotInventOptionalInterfaces(t *testing.T) {
	_, wrapped := newAccessLogResponseWriter(&accessLogTestResponseWriter{})

	if _, ok := wrapped.(http.Flusher); ok {
		t.Fatalf("wrapped writer unexpectedly implements http.Flusher")
	}
	if _, ok := wrapped.(http.Hijacker); ok {
		t.Fatalf("wrapped writer unexpectedly implements http.Hijacker")
	}
	if _, ok := wrapped.(http.Pusher); ok {
		t.Fatalf("wrapped writer unexpectedly implements http.Pusher")
	}
	if _, ok := wrapped.(io.ReaderFrom); ok {
		t.Fatalf("wrapped writer unexpectedly implements io.ReaderFrom")
	}
}

func TestHostStaticHandlerDoesNotSeeInventedFlusher(t *testing.T) {
	host := NewHost(HostOptions{})
	var sawFlusher bool
	host.RegisterStaticHandler("/", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, sawFlusher = w.(http.Flusher)
		w.WriteHeader(http.StatusNoContent)
	}))

	req, err := http.NewRequest(http.MethodGet, "/events", nil)
	if err != nil {
		t.Fatalf("NewRequest returned error: %v", err)
	}
	host.ServeHTTP(&accessLogTestResponseWriter{}, req)

	if sawFlusher {
		t.Fatalf("static handler saw http.Flusher even though the underlying writer does not support it")
	}
}

func TestAccessLogResponseWriterPreservesFlusherWhenUnderlyingSupportsIt(t *testing.T) {
	underlying := &accessLogTestFlusher{accessLogTestResponseWriter: &accessLogTestResponseWriter{}}
	_, wrapped := newAccessLogResponseWriter(underlying)

	flusher, ok := wrapped.(http.Flusher)
	if !ok {
		t.Fatalf("wrapped writer does not implement http.Flusher")
	}
	flusher.Flush()
	if !underlying.flushed {
		t.Fatalf("Flush was not forwarded to the underlying writer")
	}
}

func TestAccessLogResponseWriterPreservesReaderFromAndCountsBytes(t *testing.T) {
	underlying := &accessLogTestReaderFrom{accessLogTestResponseWriter: &accessLogTestResponseWriter{}}
	logger, wrapped := newAccessLogResponseWriter(underlying)

	readerFrom, ok := wrapped.(io.ReaderFrom)
	if !ok {
		t.Fatalf("wrapped writer does not implement io.ReaderFrom")
	}
	n, err := readerFrom.ReadFrom(bytes.NewBufferString("hello"))
	if err != nil {
		t.Fatalf("ReadFrom returned error: %v", err)
	}
	if n != 5 || logger.bytes != 5 {
		t.Fatalf("ReadFrom bytes = %d, logger bytes = %d; want 5 and 5", n, logger.bytes)
	}
	if logger.status != http.StatusOK || !logger.wroteHeader {
		t.Fatalf("ReadFrom did not record implicit status: status=%d wroteHeader=%v", logger.status, logger.wroteHeader)
	}
}
