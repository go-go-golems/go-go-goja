package engine

import (
	"fmt"
	"time"

	"github.com/dop251/goja"
)

func installPerformanceGlobals(vm *goja.Runtime) error {
	if vm == nil {
		return fmt.Errorf("runtime VM is nil")
	}
	start := time.Now()
	performance := vm.NewObject()
	if err := performance.Set("now", func() float64 {
		return float64(time.Since(start).Nanoseconds()) / 1e6
	}); err != nil {
		return fmt.Errorf("set performance.now: %w", err)
	}
	if err := vm.Set("performance", performance); err != nil {
		return fmt.Errorf("set performance global: %w", err)
	}
	return nil
}

func installConsoleTimers(vm *goja.Runtime) error {
	if vm == nil {
		return fmt.Errorf("runtime VM is nil")
	}
	consoleValue := vm.Get("console")
	if consoleValue == nil || goja.IsUndefined(consoleValue) || goja.IsNull(consoleValue) {
		return fmt.Errorf("console global is not installed")
	}
	consoleObj := consoleValue.ToObject(vm)
	timers := map[string]time.Time{}
	label := func(call goja.FunctionCall) string {
		if len(call.Arguments) == 0 || goja.IsUndefined(call.Argument(0)) {
			return "default"
		}
		return call.Argument(0).String()
	}
	log := func(message string) {
		if fn, ok := goja.AssertFunction(consoleObj.Get("log")); ok {
			_, _ = fn(consoleObj, vm.ToValue(message))
		}
	}
	if err := consoleObj.Set("time", func(call goja.FunctionCall) goja.Value {
		timers[label(call)] = time.Now()
		return goja.Undefined()
	}); err != nil {
		return fmt.Errorf("set console.time: %w", err)
	}
	if err := consoleObj.Set("timeLog", func(call goja.FunctionCall) goja.Value {
		l := label(call)
		if started, ok := timers[l]; ok {
			log(fmt.Sprintf("%s: %.3fms", l, float64(time.Since(started).Nanoseconds())/1e6))
		}
		return goja.Undefined()
	}); err != nil {
		return fmt.Errorf("set console.timeLog: %w", err)
	}
	if err := consoleObj.Set("timeEnd", func(call goja.FunctionCall) goja.Value {
		l := label(call)
		if started, ok := timers[l]; ok {
			log(fmt.Sprintf("%s: %.3fms", l, float64(time.Since(started).Nanoseconds())/1e6))
			delete(timers, l)
		}
		return goja.Undefined()
	}); err != nil {
		return fmt.Errorf("set console.timeEnd: %w", err)
	}
	return nil
}
