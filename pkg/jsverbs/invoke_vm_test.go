package jsverbs

import (
	"context"
	"testing"
	"time"

	"github.com/dop251/goja"
	noderequire "github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/glazed/pkg/cmds/runner"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/pkg/runtimebridge"
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

function getter() { return { get x() { throw new Error("from a getter"); } }; }
__verb__("getter", {});

async function asyncGetter() { return { get x() { throw new Error("from an async getter"); } }; }
__verb__("asyncGetter", {});

function spinGetter() { return { get x() { for (;;) {} } }; }
__verb__("spinGetter", {});

function seen() { return { value: globalThis.currentValue() }; }
__verb__("seen", {});
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

// Exporting a result runs its getters outside any JavaScript try: what they
// throw, or an interrupt while they run, comes back as the error instead of a
// panic in the host.
func TestInvokeInVMRecoversExportPanics(t *testing.T) {
	registry := plainVMRegistry(t)
	for _, tc := range []struct{ verb, want string }{
		{"verbs num getter", "from a getter"},
		{"verbs num async-getter", "from an async getter"},
	} {
		verb, ok := registry.Verb(tc.verb)
		require.True(t, ok, "verbs: %v", verbPaths(registry))
		vm, req := plainVM(registry)
		var err error
		require.NotPanics(t, func() {
			_, err = registry.InvokeInVM(context.Background(), vm, req, verb, plainVMValues(t, registry, verb, nil))
		})
		var exc *goja.Exception
		require.ErrorAs(t, err, &exc, tc.verb)
		require.ErrorContains(t, err, tc.want)
	}

	verb, _ := registry.Verb("verbs num spin-getter")
	vm, req := plainVM(registry)
	stop := time.AfterFunc(50*time.Millisecond, func() { vm.Interrupt("too long") })
	defer stop.Stop()
	var err error
	require.NotPanics(t, func() {
		_, err = registry.InvokeInVM(context.Background(), vm, req, verb, plainVMValues(t, registry, verb, nil))
	})
	var interrupted *goja.InterruptedError
	require.ErrorAs(t, err, &interrupted)
	require.Equal(t, "too long", interrupted.Value())
}

type ctxKey struct{}

// A VM bound in runtimebridge sees the invocation's context as the current
// owner context; a bare VM keeps its fallback.
func TestInvokeInVMCallContext(t *testing.T) {
	registry := plainVMRegistry(t)
	verb, _ := registry.Verb("verbs num seen")
	ctx := context.WithValue(context.Background(), ctxKey{}, "from the invocation")

	run := func(bind bool) interface{} {
		vm, req := plainVM(registry)
		if bind {
			runtimebridge.Store(vm, runtimebridge.RuntimeServices{LifetimeContext: context.Background()})
			defer runtimebridge.Delete(vm)
		}
		require.NoError(t, vm.Set("currentValue", func() interface{} {
			return runtimebridge.CurrentOwnerContext(vm).Value(ctxKey{})
		}))
		result, err := registry.InvokeInVM(ctx, vm, req, verb, plainVMValues(t, registry, verb, nil))
		require.NoError(t, err)
		return result.(map[string]interface{})["value"]
	}
	require.Equal(t, "from the invocation", run(true))
	require.Nil(t, run(false))
}

func verbPaths(registry *Registry) []string {
	out := []string{}
	for _, v := range registry.Verbs() {
		out = append(out, v.FullPath())
	}
	return out
}
