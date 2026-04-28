package yamlmod_test

import (
	"context"
	"testing"

	"github.com/dop251/goja"
	gggengine "github.com/go-go-golems/go-go-goja/engine"
	"github.com/stretchr/testify/require"
)

func TestYamlParseSimple(t *testing.T) {
	rt := newDefaultRuntime(t)

	ret, err := rt.Owner.Call(context.Background(), "yaml.parse.simple", func(_ context.Context, vm *goja.Runtime) (any, error) {
		val, runErr := vm.RunString(`
			const yaml = require("yaml");
			yaml.parse("hello: world");
		`)
		if runErr != nil {
			return nil, runErr
		}
		return val.Export(), nil
	})
	require.NoError(t, err)
	require.Equal(t, map[string]any{"hello": "world"}, ret)
}

func TestYamlParseNested(t *testing.T) {
	rt := newDefaultRuntime(t)

	ret, err := rt.Owner.Call(context.Background(), "yaml.parse.nested", func(_ context.Context, vm *goja.Runtime) (any, error) {
		val, runErr := vm.RunString(`
			const yaml = require("yaml");
			yaml.parse("a:\n  b: 1\n  c:\n    - x\n    - y");
		`)
		if runErr != nil {
			return nil, runErr
		}
		return val.Export(), nil
	})
	require.NoError(t, err)
	require.Equal(t, map[string]any{
		"a": map[string]any{
			"b": 1,
			"c": []any{"x", "y"},
		},
	}, ret)
}

func TestYamlParseInvalid(t *testing.T) {
	rt := newDefaultRuntime(t)

	_, err := rt.Owner.Call(context.Background(), "yaml.parse.invalid", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, runErr := vm.RunString(`
			const yaml = require("yaml");
			yaml.parse("[bad");
		`)
		return nil, runErr
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "yaml.parse")
}

func TestYamlStringifySimple(t *testing.T) {
	rt := newDefaultRuntime(t)

	ret, err := rt.Owner.Call(context.Background(), "yaml.stringify.simple", func(_ context.Context, vm *goja.Runtime) (any, error) {
		val, runErr := vm.RunString(`
			const yaml = require("yaml");
			yaml.stringify({ hello: "world", count: 42 });
		`)
		if runErr != nil {
			return nil, runErr
		}
		return val.Export(), nil
	})
	require.NoError(t, err)
	s, ok := ret.(string)
	require.True(t, ok)
	require.Contains(t, s, "hello: world")
	require.Contains(t, s, "count: 42")
}

func TestYamlStringifyWithIndent(t *testing.T) {
	rt := newDefaultRuntime(t)

	ret, err := rt.Owner.Call(context.Background(), "yaml.stringify.indent", func(_ context.Context, vm *goja.Runtime) (any, error) {
		val, runErr := vm.RunString(`
			const yaml = require("yaml");
			yaml.stringify({ a: { b: 1 } }, { indent: 4 });
		`)
		if runErr != nil {
			return nil, runErr
		}
		return val.Export(), nil
	})
	require.NoError(t, err)
	s, ok := ret.(string)
	require.True(t, ok)
	// With indent 4, nested map should have 4 spaces
	require.Contains(t, s, "    b: 1")
}

func TestYamlStringifyUnknownOption(t *testing.T) {
	rt := newDefaultRuntime(t)

	_, err := rt.Owner.Call(context.Background(), "yaml.stringify.unknown", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, runErr := vm.RunString(`
			const yaml = require("yaml");
			yaml.stringify({}, { foo: 1 });
		`)
		return nil, runErr
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown option")
}

func TestYamlValidateValid(t *testing.T) {
	rt := newDefaultRuntime(t)

	ret, err := rt.Owner.Call(context.Background(), "yaml.validate.valid", func(_ context.Context, vm *goja.Runtime) (any, error) {
		val, runErr := vm.RunString(`
			const yaml = require("yaml");
			yaml.validate("hello: world");
		`)
		if runErr != nil {
			return nil, runErr
		}
		return val.Export(), nil
	})
	require.NoError(t, err)
	m, ok := ret.(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, m["valid"])
}

func TestYamlValidateInvalid(t *testing.T) {
	rt := newDefaultRuntime(t)

	ret, err := rt.Owner.Call(context.Background(), "yaml.validate.invalid", func(_ context.Context, vm *goja.Runtime) (any, error) {
		val, runErr := vm.RunString(`
			const yaml = require("yaml");
			yaml.validate("[bad");
		`)
		if runErr != nil {
			return nil, runErr
		}
		return val.Export(), nil
	})
	require.NoError(t, err)
	m, ok := ret.(map[string]any)
	require.True(t, ok)
	require.Equal(t, false, m["valid"])
	require.NotNil(t, m["errors"])
	switch e := m["errors"].(type) {
	case []any:
		require.GreaterOrEqual(t, len(e), 1)
	case []string:
		require.GreaterOrEqual(t, len(e), 1)
	default:
		t.Fatalf("unexpected errors type %T", m["errors"])
	}
}

func TestYamlRoundTrip(t *testing.T) {
	rt := newDefaultRuntime(t)

	ret, err := rt.Owner.Call(context.Background(), "yaml.roundtrip", func(_ context.Context, vm *goja.Runtime) (any, error) {
		val, runErr := vm.RunString(`
			const yaml = require("yaml");
			const original = { name: "test", items: [1, 2, 3], nested: { ok: true } };
			const serialized = yaml.stringify(original);
			const parsed = yaml.parse(serialized);
			JSON.stringify(parsed);
		`)
		if runErr != nil {
			return nil, runErr
		}
		return val.Export(), nil
	})
	require.NoError(t, err)
	s, ok := ret.(string)
	require.True(t, ok)
	require.JSONEq(t, `{"name":"test","items":[1,2,3],"nested":{"ok":true}}`, s)
}

func TestDefaultRuntimeCanRequireYamlModule(t *testing.T) {
	rt := newDefaultRuntime(t)

	ret, err := rt.Owner.Call(context.Background(), "yaml.require", func(_ context.Context, vm *goja.Runtime) (any, error) {
		value, err := vm.RunString(`
			const yaml = require("yaml");
			typeof yaml.parse + ":" + typeof yaml.stringify + ":" + typeof yaml.validate;
		`)
		if err != nil {
			return nil, err
		}
		return value.Export(), nil
	})
	require.NoError(t, err)
	require.Equal(t, "function:function:function", ret)
}

func TestYamlParseMultiDocumentReturnsFirst(t *testing.T) {
	rt := newDefaultRuntime(t)

	ret, err := rt.Owner.Call(context.Background(), "yaml.parse.multi", func(_ context.Context, vm *goja.Runtime) (any, error) {
		val, runErr := vm.RunString(`
			const yaml = require("yaml");
			yaml.parse("a: 1\n---\nb: 2");
		`)
		if runErr != nil {
			return nil, runErr
		}
		return val.Export(), nil
	})
	require.NoError(t, err)
	require.Equal(t, map[string]any{"a": 1}, ret)
}

func TestYamlValidateMultiDocument(t *testing.T) {
	rt := newDefaultRuntime(t)

	ret, err := rt.Owner.Call(context.Background(), "yaml.validate.multi", func(_ context.Context, vm *goja.Runtime) (any, error) {
		val, runErr := vm.RunString(`
			const yaml = require("yaml");
			yaml.validate("a: 1\n---\nb: 2");
		`)
		if runErr != nil {
			return nil, runErr
		}
		return val.Export(), nil
	})
	require.NoError(t, err)
	m, ok := ret.(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, m["valid"])
}

func TestYamlStringifyNegativeIndent(t *testing.T) {
	rt := newDefaultRuntime(t)

	_, err := rt.Owner.Call(context.Background(), "yaml.stringify.negindent", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, runErr := vm.RunString(`
			const yaml = require("yaml");
			yaml.stringify({}, { indent: -1 });
		`)
		return nil, runErr
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "indent must be >= 0")
}

func newDefaultRuntime(t *testing.T) *gggengine.Runtime {
	t.Helper()

	factory, err := gggengine.NewBuilder().
		WithModules(gggengine.DefaultRegistryModules()).
		Build()
	require.NoError(t, err)

	rt, err := factory.NewRuntime(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, rt.Close(context.Background()))
	})

	return rt
}
