package jsevents_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/dop251/goja"
	gggengine "github.com/go-go-golems/go-go-goja/engine"
	"github.com/go-go-golems/go-go-goja/pkg/jsevents"
	"github.com/stretchr/testify/require"
)

func TestManagerAdoptsJSCreatedEmitterAndEmitsFromGo(t *testing.T) {
	var ref *jsevents.EmitterRef
	rt := newRuntime(t, jsevents.Install())
	manager, ok := jsevents.FromRuntime(rt)
	require.True(t, ok)

	_, err := rt.Owner.Call(context.Background(), "jsevents.test.adopt", func(_ context.Context, vm *goja.Runtime) (any, error) {
		if err := vm.Set("adopt", func(value goja.Value) bool {
			var adoptErr error
			ref, adoptErr = manager.AdoptEmitterOnOwner(value)
			return adoptErr == nil
		}); err != nil {
			return nil, err
		}
		_, err := vm.RunString(`
			const EventEmitter = require("events");
			globalThis.seen = [];
			const emitter = new EventEmitter();
			emitter.on("message", (msg) => seen.push(msg.kind + ":" + msg.value));
			if (!adopt(emitter)) throw new Error("adopt failed");
		`)
		return nil, err
	})
	require.NoError(t, err)
	require.NotNil(t, ref)

	delivered, err := ref.EmitSync(context.Background(), "message", map[string]any{"kind": "go", "value": "payload"})
	require.NoError(t, err)
	require.True(t, delivered)

	got := runJS(t, rt, `JSON.stringify(globalThis.seen)`)
	require.Equal(t, `["go:payload"]`, got)

	require.NoError(t, ref.Close(context.Background()))
	_, err = ref.EmitSync(context.Background(), "message", map[string]any{"kind": "go", "value": "after-close"})
	require.Error(t, err)
}

func TestManagerAsyncEmitReportsListenerErrors(t *testing.T) {
	errCh := make(chan error, 1)
	var ref *jsevents.EmitterRef
	rt := newRuntime(t, jsevents.Install(jsevents.WithErrorHandler(func(err error) {
		errCh <- err
	})))
	manager, ok := jsevents.FromRuntime(rt)
	require.True(t, ok)

	_, err := rt.Owner.Call(context.Background(), "jsevents.test.error-adopt", func(_ context.Context, vm *goja.Runtime) (any, error) {
		if err := vm.Set("adopt", func(value goja.Value) bool {
			var adoptErr error
			ref, adoptErr = manager.AdoptEmitterOnOwner(value)
			return adoptErr == nil
		}); err != nil {
			return nil, err
		}
		_, err := vm.RunString(`
			const EventEmitter = require("events");
			const emitter = new EventEmitter();
			emitter.on("boom", () => { throw new Error("listener failed"); });
			adopt(emitter);
		`)
		return nil, err
	})
	require.NoError(t, err)
	require.NotNil(t, ref)

	require.NoError(t, ref.Emit(context.Background(), "boom"))
	require.Eventually(t, func() bool {
		select {
		case err := <-errCh:
			return strings.Contains(err.Error(), "listener failed")
		default:
			return false
		}
	}, time.Second, 10*time.Millisecond)
}

func newRuntime(t *testing.T, inits ...gggengine.RuntimeInitializer) *gggengine.Runtime {
	t.Helper()
	factory, err := gggengine.NewBuilder().WithRuntimeInitializers(inits...).Build()
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
	str, err := runJSTry(rt, code)
	require.NoError(t, err)
	return str
}

func runJSTry(rt *gggengine.Runtime, code string) (string, error) {
	ret, err := rt.Owner.Call(context.Background(), "jsevents.test.run", func(_ context.Context, vm *goja.Runtime) (any, error) {
		value, err := vm.RunString(code)
		if err != nil {
			return nil, err
		}
		return value.String(), nil
	})
	if err != nil {
		return "", err
	}
	str, ok := ret.(string)
	if !ok {
		return "", fmt.Errorf("expected string result, got %T", ret)
	}
	return str, nil
}
