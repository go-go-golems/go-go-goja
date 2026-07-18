package fetch

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/dop251/goja"
)

func responseValue(vm *goja.Runtime, data responseData, expect expectation) (goja.Value, goja.Value) {
	switch expect {
	case expectationResponse:
		return newResponseObject(vm, data), nil
	case expectationJSON:
		if data.Status < 200 || data.Status > 299 {
			return nil, httpErrorValue(vm, data)
		}
		value, err := decodeJSON(vm, data.Body)
		if err != nil {
			return nil, vm.NewGoError(err)
		}
		return value, nil
	case expectationText:
		if data.Status < 200 || data.Status > 299 {
			return nil, httpErrorValue(vm, data)
		}
		return vm.ToValue(string(data.Body)), nil
	default:
		return nil, vm.NewGoError(fmt.Errorf("unsupported fetch response expectation %d", expect))
	}
}

func newResponseObject(vm *goja.Runtime, data responseData) goja.Value {
	obj := vm.NewObject()
	_ = obj.Set("url", data.URL)
	_ = obj.Set("status", data.Status)
	_ = obj.Set("statusText", data.StatusText)
	_ = obj.Set("ok", data.Status >= 200 && data.Status <= 299)
	_ = obj.Set("headers", data.Headers)
	_ = obj.Set("text", func() goja.Value {
		return resolvedPromise(vm, string(data.Body))
	})
	_ = obj.Set("json", func() goja.Value {
		value, err := decodeJSON(vm, data.Body)
		if err != nil {
			return rejectedPromiseValue(vm, vm.NewGoError(err))
		}
		return resolvedPromiseValue(vm, value)
	})
	return obj
}

func decodeJSON(vm *goja.Runtime, body []byte) (goja.Value, error) {
	var decoded any
	if len(body) == 0 {
		decoded = nil
	} else if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, fmt.Errorf("fetch response json: %w", err)
	}
	return vm.ToValue(decoded), nil
}

func httpErrorValue(vm *goja.Runtime, data responseData) goja.Value {
	obj := vm.NewObject()
	_ = obj.Set("name", "HTTPError")
	_ = obj.Set("message", fmt.Sprintf("HTTP %d %s", data.Status, http.StatusText(data.Status)))
	_ = obj.Set("status", data.Status)
	_ = obj.Set("statusText", data.StatusText)
	_ = obj.Set("url", data.URL)
	_ = obj.Set("body", string(data.Body))
	return obj
}

func resolvedPromise(vm *goja.Runtime, value any) goja.Value {
	promise, resolve, _ := vm.NewPromise()
	_ = resolve(value)
	return vm.ToValue(promise)
}

func resolvedPromiseValue(vm *goja.Runtime, value goja.Value) goja.Value {
	promise, resolve, _ := vm.NewPromise()
	_ = resolve(value)
	return vm.ToValue(promise)
}

func rejectedPromise(vm *goja.Runtime, err error) goja.Value {
	return rejectedPromiseValue(vm, vm.NewGoError(err))
}

func rejectedPromiseValue(vm *goja.Runtime, value goja.Value) goja.Value {
	promise, _, reject := vm.NewPromise()
	_ = reject(value)
	return vm.ToValue(promise)
}
