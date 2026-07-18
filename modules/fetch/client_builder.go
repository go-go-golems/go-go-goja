package fetch

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/buffer"
	"github.com/go-go-golems/go-go-goja/pkg/runtimebridge"
)

type clientState struct {
	baseURL     string
	timeout     time.Duration
	headers     map[string]string
	credential  credentialSource
	expectation expectation
}

func (m Module) newClientBuilder(vm *goja.Runtime, runtimeServices runtimebridge.RuntimeServices, store *builderStore) *goja.Object {
	state := &clientState{headers: map[string]string{}, expectation: expectationResponse}
	obj := vm.NewObject()
	_ = obj.Set("baseUrl", func(raw string) *goja.Object {
		state.baseURL = strings.TrimRight(strings.TrimSpace(raw), "/")
		return obj
	})
	_ = obj.Set("timeout", func(raw string) goja.Value {
		d, err := time.ParseDuration(strings.TrimSpace(raw))
		if err != nil {
			panic(vm.NewGoError(fmt.Errorf("fetch.client().timeout(%q): %w", raw, err)))
		}
		state.timeout = d
		return obj
	})
	_ = obj.Set("header", func(name, value string) *goja.Object {
		state.headers[strings.TrimSpace(name)] = value
		return obj
	})
	_ = obj.Set("auth", func(value goja.Value) *goja.Object {
		credential, err := store.credential(vm, value)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		state.credential = credential
		return obj
	})
	_ = obj.Set("acceptJson", func() *goja.Object {
		setDefaultHeader(state.headers, "Accept", "application/json")
		return obj
	})
	_ = obj.Set("expectJson", func() *goja.Object {
		state.expectation = expectationJSON
		setDefaultHeader(state.headers, "Accept", "application/json")
		return obj
	})
	_ = obj.Set("expectText", func() *goja.Object { state.expectation = expectationText; return obj })
	_ = obj.Set("expectResponse", func() *goja.Object { state.expectation = expectationResponse; return obj })
	for _, method := range []string{"get", "post", "put", "patch", "delete"} {
		method := method
		_ = obj.Set(method, func(path string) *goja.Object {
			return m.newRequestBuilder(vm, runtimeServices, state, strings.ToUpper(method), path)
		})
	}
	_ = obj.Set("request", func(method, path string) *goja.Object {
		return m.newRequestBuilder(vm, runtimeServices, state, method, path)
	})
	return obj
}

func (m Module) newRequestBuilder(vm *goja.Runtime, runtimeServices runtimebridge.RuntimeServices, client *clientState, method, path string) *goja.Object {
	spec := requestSpec{Method: method, Headers: cloneStringMap(client.headers), Timeout: client.timeout, Expectation: client.expectation, Credential: client.credential}
	resolved, err := resolveURL(client.baseURL, path)
	if err != nil {
		// Defer the error to run() so construction stays chainable.
		resolved = ""
	}
	spec.URL = resolved
	constructionErr := err
	obj := vm.NewObject()
	_ = obj.Set("query", func(name string, value goja.Value) *goja.Object {
		if constructionErr != nil {
			return obj
		}
		next, err := addQuery(spec.URL, name, value.String())
		if err != nil {
			constructionErr = err
			return obj
		}
		spec.URL = next
		return obj
	})
	_ = obj.Set("header", func(name, value string) *goja.Object {
		spec.Headers[strings.TrimSpace(name)] = value
		return obj
	})
	_ = obj.Set("json", func(value goja.Value) *goja.Object {
		body, err := json.Marshal(value.Export())
		if err != nil {
			constructionErr = err
			return obj
		}
		spec.Body = body
		setDefaultHeader(spec.Headers, "Content-Type", "application/json")
		setDefaultHeader(spec.Headers, "Accept", "application/json")
		return obj
	})
	_ = obj.Set("body", func(value goja.Value) *goja.Object {
		spec.Body = buffer.DecodeBytes(vm, value, goja.Undefined())
		return obj
	})
	_ = obj.Set("expectJson", func() *goja.Object { spec.Expectation = expectationJSON; return obj })
	_ = obj.Set("expectText", func() *goja.Object { spec.Expectation = expectationText; return obj })
	_ = obj.Set("expectResponse", func() *goja.Object { spec.Expectation = expectationResponse; return obj })
	_ = obj.Set("run", func() goja.Value {
		if constructionErr != nil {
			return rejectedPromise(vm, constructionErr)
		}
		return m.asyncExecute(vm, runtimeServices, spec, spec.Expectation)
	})
	return obj
}

func cloneStringMap(in map[string]string) map[string]string {
	out := map[string]string{}
	for key, value := range in {
		out[key] = value
	}
	return out
}

func addQuery(raw, name, value string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	query := u.Query()
	query.Set(strings.TrimSpace(name), value)
	u.RawQuery = query.Encode()
	return u.String(), nil
}
