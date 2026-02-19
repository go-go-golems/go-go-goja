package engine

import (
	"strings"
	"testing"

	"github.com/dop251/goja_nodejs/require"
)

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
