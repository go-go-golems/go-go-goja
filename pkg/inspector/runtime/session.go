// Package runtime provides helpers for introspecting goja runtime values
// in the context of the Smalltalk-style inspector.
package runtime

import (
	"fmt"
	"strings"

	"github.com/dop251/goja"
)

// Session owns a goja.Runtime and provides load/eval/inspect helpers.
type Session struct {
	VM *goja.Runtime
}

// NewSession creates a new runtime session with a fresh VM.
func NewSession() *Session {
	return &Session{VM: goja.New()}
}

// Load executes source code in the VM.
func (s *Session) Load(source string) error {
	_, err := s.VM.RunString(source)
	return err
}

// Eval evaluates an expression and returns the result value.
func (s *Session) Eval(expr string) (goja.Value, error) {
	return s.VM.RunString(expr)
}

// EvalResult holds the outcome of an eval operation.
type EvalResult struct {
	Expression string
	Value      goja.Value
	Error      error
	ErrorStack string // parsed exception stack if available
}

// EvalWithCapture evaluates an expression and captures result or error details.
func (s *Session) EvalWithCapture(expr string) EvalResult {
	val, err := s.VM.RunString(expr)
	result := EvalResult{
		Expression: expr,
		Value:      val,
		Error:      err,
	}
	if err != nil {
		if ex, ok := err.(*goja.Exception); ok {
			result.ErrorStack = ex.String()
		}
	}
	return result
}

// GlobalValue returns the value of a global variable by name.
func (s *Session) GlobalValue(name string) goja.Value {
	return s.VM.Get(name)
}

// ValuePreview returns a short string representation of a value.
func ValuePreview(val goja.Value, vm *goja.Runtime, maxLen int) string {
	if val == nil || goja.IsUndefined(val) {
		return "undefined"
	}
	if goja.IsNull(val) {
		return "null"
	}

	exportVal := val.Export()
	switch v := exportVal.(type) {
	case string:
		s := fmt.Sprintf("%q", v)
		if len(s) > maxLen {
			return s[:maxLen-1] + "…\""
		}
		return s
	case bool:
		return fmt.Sprintf("%v", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case float64:
		return fmt.Sprintf("%g", v)
	}

	// Check if it's a function
	if _, ok := goja.AssertFunction(val); ok {
		obj := val.ToObject(vm)
		name := ""
		if n := obj.Get("name"); n != nil && !goja.IsUndefined(n) {
			name = n.String()
		}
		if name != "" {
			return fmt.Sprintf("ƒ %s()", name)
		}
		return "ƒ ()"
	}

	// Object
	if obj, ok := val.(*goja.Object); ok {
		keys := obj.Keys()
		if len(keys) == 0 {
			return "{}"
		}
		preview := "{" + strings.Join(keys[:minInt(len(keys), 3)], ", ")
		if len(keys) > 3 {
			preview += ", …"
		}
		preview += "}"
		if len(preview) > maxLen {
			return preview[:maxLen-1] + "…"
		}
		return preview
	}

	s := val.String()
	if len(s) > maxLen {
		return s[:maxLen-1] + "…"
	}
	return s
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
