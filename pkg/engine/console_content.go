package engine

import (
	"fmt"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/pkg/resultcontent"
)

// installConsoleContentPlaceholders wraps the console printing methods so a
// result content provider passed as an argument prints as its placeholder
// ("[content: look after]") instead of an object dump. Content is not
// rendered or collected; this is display only.
func installConsoleContentPlaceholders(vm *goja.Runtime) error {
	if vm == nil {
		return fmt.Errorf("runtime VM is nil")
	}
	consoleValue := vm.Get("console")
	if consoleValue == nil || goja.IsUndefined(consoleValue) || goja.IsNull(consoleValue) {
		return fmt.Errorf("console global is not installed")
	}
	consoleObj := consoleValue.ToObject(vm)
	for _, name := range []string{"log", "info", "debug", "warn", "error"} {
		original, ok := goja.AssertFunction(consoleObj.Get(name))
		if !ok {
			continue
		}
		if err := consoleObj.Set(name, func(call goja.FunctionCall) goja.Value {
			args := ConsoleArgsWithContentPlaceholders(vm, call.Arguments)
			ret, err := original(call.This, args...)
			if err != nil {
				if exc, ok := err.(*goja.Exception); ok {
					panic(exc.Value())
				}
				panic(vm.NewGoError(err))
			}
			return ret
		}); err != nil {
			return fmt.Errorf("wrap console.%s: %w", name, err)
		}
	}
	return nil
}

// ConsoleArgsWithContentPlaceholders returns args with every top-level result
// content provider replaced by its inline placeholder string. Other arguments
// are returned unchanged (the slice is copied only when something changes).
// It must run on the runtime owner goroutine.
func ConsoleArgsWithContentPlaceholders(vm *goja.Runtime, args []goja.Value) []goja.Value {
	var out []goja.Value
	for i, arg := range args {
		p, ok := resultcontent.ProviderOf(arg)
		if !ok {
			continue
		}
		if out == nil {
			out = append([]goja.Value(nil), args...)
		}
		out[i] = vm.ToValue(resultcontent.InlinePlaceholder(p))
	}
	if out == nil {
		return args
	}
	return out
}
