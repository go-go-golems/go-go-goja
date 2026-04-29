package timermod_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/dop251/goja"
	gggengine "github.com/go-go-golems/go-go-goja/engine"
	"github.com/stretchr/testify/require"
)

func TestTimerSleepResolves(t *testing.T) {
	rt := newDefaultRuntime(t)

	_, err := rt.Owner.Call(context.Background(), "timer.sleep.resolve.setup", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, err := vm.RunString(`
			globalThis.timerState = { done: false, error: "" };
			const timer = require("timer");
			timer.sleep(20)
				.then(() => {
					globalThis.timerState.done = true;
				})
				.catch((err) => {
					globalThis.timerState.error = String(err);
				});
		`)
		return nil, err
	})
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		state, stateErr := readTimerState(t, rt)
		require.NoError(t, stateErr)
		return state.Done && state.Error == ""
	}, time.Second, 10*time.Millisecond)
}

func TestTimerSleepRejectsNegativeDuration(t *testing.T) {
	rt := newDefaultRuntime(t)

	_, err := rt.Owner.Call(context.Background(), "timer.sleep.reject.setup", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, err := vm.RunString(`
			globalThis.timerState = { done: false, error: "" };
			const timer = require("timer");
			timer.sleep(-1)
				.then(() => {
					globalThis.timerState.done = true;
				})
				.catch((err) => {
					globalThis.timerState.error = String(err);
				});
		`)
		return nil, err
	})
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		state, stateErr := readTimerState(t, rt)
		require.NoError(t, stateErr)
		return state.Error != ""
	}, time.Second, 10*time.Millisecond)

	state, err := readTimerState(t, rt)
	require.NoError(t, err)
	require.False(t, state.Done)
	require.Contains(t, state.Error, "duration must be >= 0")
}

func TestDefaultRuntimeCanRequireTimerModule(t *testing.T) {
	rt := newDefaultRuntime(t)

	ret, err := rt.Owner.Call(context.Background(), "timer.require", func(_ context.Context, vm *goja.Runtime) (any, error) {
		value, err := vm.RunString(`
			const timer = require("timer");
			typeof timer.sleep;
		`)
		if err != nil {
			return nil, err
		}
		return value.Export(), nil
	})
	require.NoError(t, err)
	require.Equal(t, "function", ret)
}

type timerState struct {
	Done  bool
	Error string
}

func newDefaultRuntime(t *testing.T) *gggengine.Runtime {
	t.Helper()

	factory, err := gggengine.NewBuilder().
		UseModuleMiddleware(gggengine.MiddlewareSafe()).
		Build()
	require.NoError(t, err)

	rt, err := factory.NewRuntime(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, rt.Close(context.Background()))
	})

	return rt
}

func readTimerState(t *testing.T, rt *gggengine.Runtime) (timerState, error) {
	t.Helper()

	value, err := rt.Owner.Call(context.Background(), "timer.state.read", func(_ context.Context, vm *goja.Runtime) (any, error) {
		val, runErr := vm.RunString(`JSON.stringify(globalThis.timerState || { done: false, error: "" })`)
		if runErr != nil {
			return nil, runErr
		}
		return val.Export(), nil
	})
	if err != nil {
		return timerState{}, err
	}

	switch raw := value.(type) {
	case string:
		state := timerState{}
		if raw == "" {
			return state, nil
		}
		if err := json.Unmarshal([]byte(raw), &state); err != nil {
			return timerState{}, err
		}
		return state, nil
	default:
		return timerState{}, nil
	}
}
