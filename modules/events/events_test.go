package events_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/dop251/goja"
	gggengine "github.com/go-go-golems/go-go-goja/engine"
	eventsmodule "github.com/go-go-golems/go-go-goja/modules/events"
	"github.com/stretchr/testify/require"
)

func TestEventsModuleExportsGoNativeConstructor(t *testing.T) {
	rt := newRuntime(t)

	got := runJS(t, rt, `
		const EventEmitter = require("events");
		const ee = new EventEmitter();
		JSON.stringify({
			functionExport: typeof EventEmitter,
			sameNamed: EventEmitter === require("events").EventEmitter,
			nodeNamed: typeof require("node:events").EventEmitter,
			hasOn: typeof ee.on,
			hasEmit: typeof ee.emit,
			instance: ee instanceof EventEmitter
		});
	`)
	require.JSONEq(t, `{"functionExport":"function","sameNamed":true,"nodeNamed":"function","hasOn":"function","hasEmit":"function","instance":true}`, got)
}

func TestEventEmitterSynchronousEmitOnceAndRemove(t *testing.T) {
	rt := newRuntime(t)

	got := runJS(t, rt, `
		const EventEmitter = require("events");
		const ee = new EventEmitter();
		const calls = [];
		function persistent(v) { calls.push("on:" + v); }
		function once(v) { calls.push("once:" + v); }
		ee.on("x", persistent);
		ee.once("x", once);
		const first = ee.emit("x", 1);
		const second = ee.emit("x", 2);
		const countBeforeOff = ee.listenerCount("x");
		ee.off("x", persistent);
		const third = ee.emit("x", 3);
		JSON.stringify({ first, second, third, calls, countBeforeOff, countAfterOff: ee.listenerCount("x") });
	`)
	require.JSONEq(t, `{"first":true,"second":true,"third":false,"calls":["on:1","once:1","on:2"],"countBeforeOff":1,"countAfterOff":0}`, got)
}

func TestEventEmitterListenerIntrospectionAndMetaEvents(t *testing.T) {
	rt := newRuntime(t)

	got := runJS(t, rt, `
		const EventEmitter = require("events");
		const ee = new EventEmitter();
		const meta = [];
		function a() {}
		function b() {}
		ee.on("newListener", (name, fn) => meta.push("new:" + name + ":" + (typeof fn)));
		ee.on("removeListener", (name, fn) => meta.push("remove:" + name + ":" + (fn === b)));
		ee.on("alpha", a);
		ee.once("beta", b);
		const listenersBetaIsOriginal = ee.listeners("beta")[0] === b;
		const rawBetaIsOriginal = ee.rawListeners("beta")[0] === b;
		ee.removeListener("beta", b);
		JSON.stringify({
			eventNames: ee.eventNames(),
			listenersBetaIsOriginal,
			rawBetaIsOriginal,
			meta
		});
	`)
	require.JSONEq(t, `{"eventNames":["alpha","newListener","removeListener"],"listenersBetaIsOriginal":true,"rawBetaIsOriginal":true,"meta":["new:removeListener:function","new:alpha:function","new:beta:function","remove:beta:true"]}`, got)
}

func TestUnhandledErrorEventThrows(t *testing.T) {
	rt := newRuntime(t)

	_, err := runJSValue(t, rt, `
		const EventEmitter = require("events");
		new EventEmitter().emit("error", new Error("boom"));
	`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "boom")
}

func TestEventEmitterEmitWithoutNameThrowsTypeError(t *testing.T) {
	rt := newRuntime(t)

	_, err := runJSValue(t, rt, `
		const EventEmitter = require("events");
		new EventEmitter().emit();
	`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "event name is required")
}

func TestEventEmitterPreservesSymbolEventNames(t *testing.T) {
	rt := newRuntime(t)

	got := runJS(t, rt, `
		const EventEmitter = require("events");
		const ee = new EventEmitter();
		const s1 = Symbol("same");
		const s2 = Symbol("same");
		const calls = [];
		ee.on(s1, () => calls.push("s1"));
		const emittedWrong = ee.emit(s2);
		const emittedRight = ee.emit(s1);
		const names = ee.eventNames();
		JSON.stringify({
			emittedWrong,
			emittedRight,
			calls,
			count1: ee.listenerCount(s1),
			count2: ee.listenerCount(s2),
			eventNameIsSymbol: names.length === 1 && names[0] === s1
		});
	`)
	require.JSONEq(t, `{"emittedWrong":false,"emittedRight":true,"calls":["s1"],"count1":1,"count2":0,"eventNameIsSymbol":true}`, got)
}

func TestGoCanAdoptJSCreatedEmitterAndEmitToIt(t *testing.T) {
	rt := newRuntime(t)
	var adopted *eventsmodule.EventEmitter

	_, err := rt.Owner.Call(context.Background(), "events.install-adopt", func(_ context.Context, vm *goja.Runtime) (any, error) {
		if err := vm.Set("adoptEmitter", func(value goja.Value) bool {
			emitter, _, ok := eventsmodule.FromValue(value)
			if ok {
				adopted = emitter
			}
			return ok
		}); err != nil {
			return nil, err
		}
		_, err := vm.RunString(`
			const EventEmitter = require("events");
			globalThis.seen = [];
			globalThis.ee = new EventEmitter();
			globalThis.ee.on("go", value => globalThis.seen.push("js:" + value));
			adoptEmitter(globalThis.ee);
		`)
		return nil, err
	})
	require.NoError(t, err)
	require.NotNil(t, adopted)

	_, err = rt.Owner.Call(context.Background(), "events.emit-from-go", func(_ context.Context, vm *goja.Runtime) (any, error) {
		if err := adopted.AddGoListener("fromJS", func(call goja.FunctionCall) goja.Value {
			arg := call.Argument(0).String()
			seen := vm.Get("seen").ToObject(vm)
			push, ok := goja.AssertFunction(seen.Get("push"))
			require.True(t, ok)
			_, callErr := push(seen, vm.ToValue("go:"+arg))
			require.NoError(t, callErr)
			return goja.Undefined()
		}); err != nil {
			return nil, err
		}
		_, err := adopted.Emit("go", vm.ToValue("payload"))
		if err != nil {
			return nil, err
		}
		_, err = vm.RunString(`globalThis.ee.emit("fromJS", "callback")`)
		return nil, err
	})
	require.NoError(t, err)

	got := runJS(t, rt, `JSON.stringify(globalThis.seen)`)
	require.Equal(t, `["js:payload","go:callback"]`, got)
}

func TestEventsModuleIsEnabledByDefault(t *testing.T) {
	rt := newRuntime(t)
	got := runJS(t, rt, `
		function canRequire(name) {
			try { return typeof require(name).EventEmitter === "function"; }
			catch (e) { return false; }
		}
		JSON.stringify({ events: canRequire("events"), nodeEvents: canRequire("node:events") });
	`)
	require.JSONEq(t, `{"events":true,"nodeEvents":true}`, got)
}

func newRuntime(t *testing.T) *gggengine.Runtime {
	t.Helper()
	factory, err := gggengine.NewBuilder().Build()
	require.NoError(t, err)
	rt, err := factory.NewRuntime(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, rt.Close(context.Background()))
	})
	return rt
}

func runJS(t *testing.T, rt *gggengine.Runtime, code string) string {
	t.Helper()
	value, err := runJSValue(t, rt, code)
	require.NoError(t, err)
	str, ok := value.(string)
	require.Truef(t, ok, "expected string result, got %T", value)
	if !json.Valid([]byte(str)) && len(str) > 0 && str[0] == '{' {
		t.Fatalf("invalid JSON returned: %s", str)
	}
	return str
}

func runJSValue(t *testing.T, rt *gggengine.Runtime, code string) (any, error) {
	t.Helper()
	return rt.Owner.Call(context.Background(), "events.test.run", func(_ context.Context, vm *goja.Runtime) (any, error) {
		value, err := vm.RunString(code)
		if err != nil {
			return nil, err
		}
		return value.Export(), nil
	})
}
