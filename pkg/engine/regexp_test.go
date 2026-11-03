package engine

import (
	"context"
	"testing"

	"github.com/dop251/goja"
)

// Unicode property escapes (\p{..}, ES2018) under the u flag. goja before
// e1fc788 (dop251/goja#745) read \p as a literal "p", so /[\p{L}]+/u matched
// only the characters p { L } and no error was raised.
func TestRegExpUnicodePropertyEscapes(t *testing.T) {
	factory, err := NewRuntimeFactoryBuilder().Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}
	rt, err := factory.NewRuntime(WithStartupContext(context.Background()), WithLifetimeContext(context.Background()))
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	defer func() { _ = rt.Close(context.Background()) }()

	tests := []struct {
		name string
		code string
		want string
	}{
		{name: "letters and numbers", code: `JSON.stringify("Grüße, x² und 42".match(/[\p{L}\p{N}]+/gu))`, want: `["Grüße","x²","und","42"]`},
		{name: "negated class", code: `"a-b_c".replace(/[^\p{L}]+/gu, " ")`, want: "a b c"},
		{name: "not a literal p", code: `/^\p{L}$/u.test("p{L}") ? "literal" : "property"`, want: "property"},
		{name: "unknown property is a SyntaxError", code: `try { new RegExp("\\p{NotAProperty}", "u"); "accepted" } catch (e) { e.name }`, want: "SyntaxError"},
		{name: "without u flag p stays an identity escape", code: `JSON.stringify("pp".match(/\p/g))`, want: `["p","p"]`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ret, err := rt.Owner.Call(context.Background(), "regexp-unicode-property-escapes", func(_ context.Context, vm *goja.Runtime) (any, error) {
				value, runErr := vm.RunString(tc.code)
				if runErr != nil {
					return nil, runErr
				}
				return value.String(), nil
			})
			if err != nil {
				t.Fatalf("run %s: %v", tc.code, err)
			}
			if ret != tc.want {
				t.Fatalf("%s = %v, want %s", tc.code, ret, tc.want)
			}
		})
	}
}
