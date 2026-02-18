package calllog

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/dop251/goja"
)

func countRows(t *testing.T, path string) int {
	t.Helper()
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("open sqlite %s: %v", path, err)
	}
	defer func() { _ = db.Close() }()

	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM call_log").Scan(&n); err != nil {
		t.Fatalf("count rows %s: %v", path, err)
	}
	return n
}

func TestRuntimeLoggerOverridesDefault(t *testing.T) {
	defaultPath := filepath.Join(t.TempDir(), "default.sqlite")
	if err := Configure(defaultPath); err != nil {
		t.Fatalf("configure default logger: %v", err)
	}
	t.Cleanup(Disable)

	vm := goja.New()
	runtimePath := filepath.Join(t.TempDir(), "runtime.sqlite")
	runtimeLogger, err := New(runtimePath)
	if err != nil {
		t.Fatalf("create runtime logger: %v", err)
	}
	BindOwnedRuntimeLogger(vm, runtimeLogger)
	t.Cleanup(func() { ReleaseRuntimeLogger(vm) })

	wrapped := WrapGoFunction(vm, "bench", "noop", func() int {
		return 42
	})
	_ = wrapped(goja.FunctionCall{This: goja.Undefined()})

	if got := countRows(t, runtimePath); got != 1 {
		t.Fatalf("runtime logger rows = %d, want 1", got)
	}
	if got := countRows(t, defaultPath); got != 0 {
		t.Fatalf("default logger rows = %d, want 0", got)
	}
}

func TestDisableRuntimeLoggerSuppressesDefault(t *testing.T) {
	defaultPath := filepath.Join(t.TempDir(), "default.sqlite")
	if err := Configure(defaultPath); err != nil {
		t.Fatalf("configure default logger: %v", err)
	}
	t.Cleanup(Disable)

	vm := goja.New()
	DisableRuntimeLogger(vm)
	t.Cleanup(func() { ReleaseRuntimeLogger(vm) })

	wrapped := WrapGoFunction(vm, "bench", "noop", func() int {
		return 42
	})
	_ = wrapped(goja.FunctionCall{This: goja.Undefined()})

	if got := countRows(t, defaultPath); got != 0 {
		t.Fatalf("default logger rows = %d, want 0", got)
	}
}

func TestReleaseRuntimeLoggerFallsBackToDefault(t *testing.T) {
	defaultPath := filepath.Join(t.TempDir(), "default.sqlite")
	if err := Configure(defaultPath); err != nil {
		t.Fatalf("configure default logger: %v", err)
	}
	t.Cleanup(Disable)

	vm := goja.New()
	runtimePath := filepath.Join(t.TempDir(), "runtime.sqlite")
	runtimeLogger, err := New(runtimePath)
	if err != nil {
		t.Fatalf("create runtime logger: %v", err)
	}
	BindOwnedRuntimeLogger(vm, runtimeLogger)

	wrapped := WrapGoFunction(vm, "bench", "noop", func() int {
		return 42
	})
	_ = wrapped(goja.FunctionCall{This: goja.Undefined()})

	ReleaseRuntimeLogger(vm)
	_ = wrapped(goja.FunctionCall{This: goja.Undefined()})

	if got := countRows(t, runtimePath); got != 1 {
		t.Fatalf("runtime logger rows = %d, want 1", got)
	}
	if got := countRows(t, defaultPath); got != 1 {
		t.Fatalf("default logger rows = %d, want 1", got)
	}
}
