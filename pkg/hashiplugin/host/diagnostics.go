package host

import (
	"fmt"
	"strings"
	"sync"
)

const maxDiagnosticBytes = 4096

type boundedDiagnosticBuffer struct {
	mu      sync.Mutex
	maxSize int
	data    []byte
}

func newBoundedDiagnosticBuffer(maxSize int) *boundedDiagnosticBuffer {
	if maxSize <= 0 {
		maxSize = maxDiagnosticBytes
	}
	return &boundedDiagnosticBuffer{maxSize: maxSize}
}

func (b *boundedDiagnosticBuffer) Write(p []byte) (int, error) {
	if b == nil || len(p) == 0 {
		return len(p), nil
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if len(p) >= b.maxSize {
		b.data = append(b.data[:0], p[len(p)-b.maxSize:]...)
		return len(p), nil
	}

	total := len(b.data) + len(p)
	if total > b.maxSize {
		drop := total - b.maxSize
		b.data = append([]byte(nil), b.data[drop:]...)
	}
	b.data = append(b.data, p...)
	return len(p), nil
}

func (b *boundedDiagnosticBuffer) String() string {
	if b == nil {
		return ""
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return strings.TrimSpace(string(b.data))
}

func wrapDiagnosticError(err error, diag *boundedDiagnosticBuffer) error {
	if err == nil {
		return nil
	}
	if diag == nil {
		return err
	}
	stderr := diag.String()
	if stderr == "" {
		return err
	}
	return fmt.Errorf("%w; plugin stderr: %s", err, stderr)
}
