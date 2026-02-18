package engine

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"

	"github.com/go-go-golems/go-go-goja/modules"
	"github.com/go-go-golems/go-go-goja/pkg/calllog"
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

func TestOpenWithRequireOptions(t *testing.T) {
	loader := func(path string) ([]byte, error) {
		trimmed := strings.TrimPrefix(path, "./")
		if trimmed == "entry.js" {
			return []byte("module.exports = { ok: 42 };"), nil
		}
		return nil, require.ModuleFileDoesNotExistError
	}

	vm, req := Open(WithRequireOptions(require.WithLoader(loader)))
	val, err := req.Require("./entry.js")
	if err != nil {
		t.Fatalf("require entry.js: %v", err)
	}

	obj := val.ToObject(vm)
	if got := obj.Get("ok").ToInteger(); got != 42 {
		t.Fatalf("ok = %d, want 42", got)
	}
}

func TestOpenDefaultDisablesRuntimeCallLog(t *testing.T) {
	defaultPath := filepath.Join(t.TempDir(), "default.sqlite")
	if err := calllog.Configure(defaultPath); err != nil {
		t.Fatalf("configure default logger: %v", err)
	}
	t.Cleanup(calllog.Disable)

	vm, _ := Open()
	exports := vm.NewObject()
	modules.SetExport(vm, exports, "bench", "noop", func() int {
		return 42
	})
	fn, ok := goja.AssertFunction(exports.Get("noop"))
	if !ok {
		t.Fatal("noop is not callable")
	}

	if _, err := fn(goja.Undefined()); err != nil {
		t.Fatalf("invoke noop: %v", err)
	}

	if got := countRows(t, defaultPath); got != 0 {
		t.Fatalf("default logger rows = %d, want 0", got)
	}
}

func TestOpenWithCallLogOptionUsesRuntimeLogger(t *testing.T) {
	defaultPath := filepath.Join(t.TempDir(), "default.sqlite")
	if err := calllog.Configure(defaultPath); err != nil {
		t.Fatalf("configure default logger: %v", err)
	}
	t.Cleanup(calllog.Disable)

	runtimePath := filepath.Join(t.TempDir(), "runtime.sqlite")
	vm, _ := Open(WithCallLog(runtimePath))
	defer calllog.ReleaseRuntimeLogger(vm)

	exports := vm.NewObject()
	modules.SetExport(vm, exports, "bench", "noop", func() int {
		return 42
	})
	fn, ok := goja.AssertFunction(exports.Get("noop"))
	if !ok {
		t.Fatal("noop is not callable")
	}

	if _, err := fn(goja.Undefined()); err != nil {
		t.Fatalf("invoke noop: %v", err)
	}

	if got := countRows(t, runtimePath); got != 1 {
		t.Fatalf("runtime logger rows = %d, want 1", got)
	}
	if got := countRows(t, defaultPath); got != 0 {
		t.Fatalf("default logger rows = %d, want 0", got)
	}
}
