package jsverbs

import (
	"context"
	"testing"

	"github.com/dop251/goja"
	noderequire "github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/glazed/pkg/cmds/runner"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/stretchr/testify/require"
)

const plainVMVerbs = `
const { double } = require("./lib/math.js");

function twice(n) { return { n, twice: double(n) }; }
__verb__("twice", { fields: { n: { type: "int", argument: true } } });

async function quick(n) { const x = await Promise.resolve(n); return { x }; }
__verb__("quick", { fields: { n: { type: "int", argument: true } } });

async function external(n) { const x = await globalThis.hostWait(); return { x: x + n }; }
__verb__("external", { fields: { n: { type: "int", argument: true } } });

async function fails() { throw new Error("nope"); }
__verb__("fails", {});
`

const plainVMMath = `module.exports = { double: (n) => n * 2 };`

func plainVMRegistry(t *testing.T) *Registry {
	t.Helper()
	registry, err := ScanSources([]SourceFile{
		{Path: "verbs/num.js", Source: []byte(plainVMVerbs)},
		{Path: "verbs/lib/math.js", Source: []byte(plainVMMath)},
	})
	require.NoError(t, err)
	return registry
}

func plainVMValues(t *testing.T, registry *Registry, verb *VerbSpec, fields map[string]interface{}) *values.Values {
	t.Helper()
	desc, err := registry.CommandDescriptionForVerb(verb)
	require.NoError(t, err)
	cmd := &Command{CommandDescription: desc, registry: registry, verb: verb}
	parsed, err := runner.ParseCommandValues(cmd, runner.WithValuesForSections(map[string]map[string]interface{}{"default": fields}))
	require.NoError(t, err)
	return parsed
}

// plainVM is a bare goja runtime with a goja_nodejs require that loads the
// scanned files through the registry, and no engine, loop or owner.
func plainVM(registry *Registry) (*goja.Runtime, RequireFunc) {
	vm := goja.New()
	reg := noderequire.NewRegistry(noderequire.WithLoader(registry.RequireLoader()))
	return vm, reg.Enable(vm).Require
}

func TestInvokeInVMSyncVerbWithRelativeRequire(t *testing.T) {
	registry := plainVMRegistry(t)
	verb, ok := registry.Verb("verbs num twice")
	require.True(t, ok, "verbs: %v", verbPaths(registry))
	vm, req := plainVM(registry)

	result, err := registry.InvokeInVM(context.Background(), vm, req, verb, plainVMValues(t, registry, verb, map[string]interface{}{"n": 21}))
	require.NoError(t, err)
	require.Equal(t, map[string]interface{}{"n": int64(21), "twice": int64(42)}, result)

	// The VM stays usable and the module stays cached for a second call.
	result, err = registry.InvokeInVM(context.Background(), vm, req, verb, plainVMValues(t, registry, verb, map[string]interface{}{"n": 2}))
	require.NoError(t, err)
	require.Equal(t, int64(4), result.(map[string]interface{})["twice"])
}

func TestInvokeInVMPromiseSettledDuringCall(t *testing.T) {
	registry := plainVMRegistry(t)
	verb, _ := registry.Verb("verbs num quick")
	vm, req := plainVM(registry)

	result, err := registry.InvokeInVM(context.Background(), vm, req, verb, plainVMValues(t, registry, verb, map[string]interface{}{"n": 5}))
	require.NoError(t, err)
	require.Equal(t, map[string]interface{}{"x": int64(5)}, result)
}

func TestInvokeInVMPendingPromiseIsReturned(t *testing.T) {
	registry := plainVMRegistry(t)
	verb, _ := registry.Verb("verbs num external")
	vm, req := plainVM(registry)

	var resolve func(interface{}) error
	require.NoError(t, vm.Set("hostWait", func() *goja.Promise {
		p, res, _ := vm.NewPromise()
		resolve = res
		return p
	}))

	result, err := registry.InvokeInVM(context.Background(), vm, req, verb, plainVMValues(t, registry, verb, map[string]interface{}{"n": 1}))
	require.NoError(t, err)
	promise, ok := result.(*goja.Promise)
	require.True(t, ok, "want *goja.Promise, got %T", result)
	require.Equal(t, goja.PromiseStatePending, promise.State())

	// The host settles it later with exclusive use of the VM; goja runs the
	// continuation when the outermost call returns.
	_, err = vm.RunString("0")
	require.NoError(t, err)
	require.NoError(t, resolve(41))
	_, err = vm.RunString("0") // drains the job queue
	require.NoError(t, err)
	require.Equal(t, goja.PromiseStateFulfilled, promise.State())
	require.Equal(t, map[string]interface{}{"x": int64(42)}, promise.Result().Export())
}

func TestInvokeInVMRejectedPromiseIsAnError(t *testing.T) {
	registry := plainVMRegistry(t)
	verb, _ := registry.Verb("verbs num fails")
	vm, req := plainVM(registry)

	_, err := registry.InvokeInVM(context.Background(), vm, req, verb, plainVMValues(t, registry, verb, nil))
	require.ErrorContains(t, err, "promise rejected")
	require.ErrorContains(t, err, "nope")
}

func TestInvokeInVMRejectsMissingArguments(t *testing.T) {
	registry := plainVMRegistry(t)
	verb, _ := registry.Verb("verbs num twice")
	vm, req := plainVM(registry)

	_, err := registry.InvokeInVM(context.Background(), vm, nil, verb, nil)
	require.ErrorContains(t, err, "require function is nil")
	_, err = registry.InvokeInVM(context.Background(), nil, req, verb, nil)
	require.ErrorContains(t, err, "runtime is nil")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = registry.InvokeInVM(ctx, vm, req, verb, nil)
	require.ErrorIs(t, err, context.Canceled)
}

func verbPaths(registry *Registry) []string {
	out := []string{}
	for _, v := range registry.Verbs() {
		out = append(out, v.FullPath())
	}
	return out
}
