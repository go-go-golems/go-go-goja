package fetch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/buffer"
	"github.com/go-go-golems/go-go-goja/modules"
	"github.com/go-go-golems/go-go-goja/pkg/runtimebridge"
)

type Option func(*Module)

type Module struct {
	name   string
	policy Policy
	client *http.Client
}

var _ modules.NativeModule = (*Module)(nil)
var _ modules.TypeScriptDeclarer = (*Module)(nil)

func New(opts ...Option) *Module {
	m := &Module{policy: Policy{Credentials: CredentialPolicy{AllowEnv: true, AllowFiles: true}}}
	for _, opt := range opts {
		if opt != nil {
			opt(m)
		}
	}
	m.policy = m.policy.normalized()
	if m.client == nil {
		m.client = &http.Client{}
	}
	return m
}

func WithName(name string) Option {
	return func(m *Module) { m.name = strings.TrimSpace(name) }
}

func WithPolicy(policy Policy) Option {
	return func(m *Module) { m.policy = policy.normalized() }
}

func WithHTTPClient(client *http.Client) Option {
	return func(m *Module) { m.client = client }
}

func (m *Module) Name() string {
	if m != nil && m.name != "" {
		return m.name
	}
	return "fetch"
}

func (m *Module) Doc() string {
	return `The fetch module provides guarded outbound HTTP and a fluent authenticated API client for xgoja JavaScript.`
}

func (m *Module) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
	if m == nil {
		m = New()
	}
	module := *m
	module.policy = module.policy.normalized()
	if module.client == nil {
		module.client = &http.Client{}
	}
	runtimeServices, ok := runtimebridge.Lookup(vm)
	if !ok || runtimeServices.Owner == nil {
		panic(vm.NewGoError(fmt.Errorf("fetch module requires runtime services")))
	}
	exports := moduleObj.Get("exports").(*goja.Object)
	store := newBuilderStore()
	modules.SetExport(exports, module.Name(), "fetch", func(call goja.FunctionCall) goja.Value {
		spec, err := module.requestSpecFromFetchCall(vm, call)
		if err != nil {
			return rejectedPromise(vm, err)
		}
		return module.asyncExecute(vm, runtimeServices, spec, expectationResponse)
	})
	modules.SetExport(exports, module.Name(), "client", func() *goja.Object {
		return module.newClientBuilder(vm, runtimeServices, store)
	})
	authObj := vm.NewObject()
	modules.SetExport(authObj, module.Name()+".auth", "none", func() *goja.Object {
		return store.newNoneAuth(vm)
	})
	modules.SetExport(authObj, module.Name()+".auth", "bearer", func() *goja.Object {
		return store.newBearerAuth(vm, module.policy)
	})
	modules.SetExport(exports, module.Name(), "auth", authObj)
}

type requestSpec struct {
	Method      string
	URL         string
	Headers     map[string]string
	Body        []byte
	Timeout     time.Duration
	Expectation expectation
	Credential  credentialSource
}

type responseData struct {
	URL        string
	Status     int
	StatusText string
	Headers    map[string][]string
	Body       []byte
}

type expectation int

const (
	expectationResponse expectation = iota
	expectationJSON
	expectationText
)

func (m Module) requestSpecFromFetchCall(vm *goja.Runtime, call goja.FunctionCall) (requestSpec, error) {
	if len(call.Arguments) == 0 || goja.IsUndefined(call.Argument(0)) || goja.IsNull(call.Argument(0)) {
		return requestSpec{}, fmt.Errorf("fetch.fetch(url, options) requires a URL")
	}
	spec := requestSpec{Method: http.MethodGet, URL: call.Argument(0).String(), Headers: map[string]string{}, Expectation: expectationResponse}
	if len(call.Arguments) < 2 || goja.IsUndefined(call.Argument(1)) || goja.IsNull(call.Argument(1)) {
		return spec, nil
	}
	options := call.Argument(1).ToObject(vm)
	if method := options.Get("method"); method != nil && !goja.IsUndefined(method) && !goja.IsNull(method) {
		spec.Method = method.String()
	}
	if headers := options.Get("headers"); headers != nil && !goja.IsUndefined(headers) && !goja.IsNull(headers) {
		spec.Headers = headersFromValue(vm, headers)
	}
	if timeout := options.Get("timeout"); timeout != nil && !goja.IsUndefined(timeout) && !goja.IsNull(timeout) {
		d, err := time.ParseDuration(timeout.String())
		if err != nil {
			return requestSpec{}, fmt.Errorf("fetch timeout %q: %w", timeout.String(), err)
		}
		spec.Timeout = d
	}
	if jsonValue := options.Get("json"); jsonValue != nil && !goja.IsUndefined(jsonValue) && !goja.IsNull(jsonValue) {
		body, err := json.Marshal(jsonValue.Export())
		if err != nil {
			return requestSpec{}, fmt.Errorf("fetch json body: %w", err)
		}
		spec.Body = body
		setDefaultHeader(spec.Headers, "Content-Type", "application/json")
		setDefaultHeader(spec.Headers, "Accept", "application/json")
		return spec, nil
	}
	if body := options.Get("body"); body != nil && !goja.IsUndefined(body) && !goja.IsNull(body) {
		spec.Body = buffer.DecodeBytes(vm, body, goja.Undefined())
	}
	return spec, nil
}

func (m Module) execute(ctx context.Context, spec requestSpec) (responseData, error) {
	policy := m.policy.normalized()
	u, err := policy.CheckURL(spec.URL)
	if err != nil {
		return responseData{}, err
	}
	method := strings.ToUpper(strings.TrimSpace(spec.Method))
	if method == "" {
		method = http.MethodGet
	}
	timeout := spec.Timeout
	if timeout <= 0 {
		timeout = policy.Timeout
	}
	reqCtx := ctx
	var cancel context.CancelFunc
	if timeout > 0 {
		reqCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	req, err := http.NewRequestWithContext(reqCtx, method, u.String(), bytes.NewReader(spec.Body))
	if err != nil {
		return responseData{}, err
	}
	for name, value := range spec.Headers {
		if strings.TrimSpace(name) == "" {
			continue
		}
		req.Header.Set(name, value)
	}
	if spec.Credential != nil {
		if err := spec.Credential.apply(reqCtx, req); err != nil {
			return responseData{}, err
		}
	}
	client := m.client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return responseData{}, redactError(err)
	}
	defer resp.Body.Close()
	limit := policy.MaxResponseBytes
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return responseData{}, err
	}
	if int64(len(body)) > limit {
		return responseData{}, fmt.Errorf("fetch response body exceeds configured limit of %d bytes", limit)
	}
	return responseData{URL: resp.Request.URL.String(), Status: resp.StatusCode, StatusText: resp.Status, Headers: cloneHeaders(resp.Header), Body: body}, nil
}

func (m Module) asyncExecute(vm *goja.Runtime, runtimeServices runtimebridge.RuntimeServices, spec requestSpec, expect expectation) goja.Value {
	promise, resolve, reject := vm.NewPromise()
	callCtx := runtimebridge.CurrentOwnerContext(vm)
	runtimeCtx := runtimeServices.Lifetime()
	go func() {
		select {
		case <-callCtx.Done():
			return
		case <-runtimeCtx.Done():
			return
		default:
		}
		data, err := m.execute(callCtx, spec)
		if err != nil {
			_ = runtimeServices.PostWithCustomContext(callCtx, "fetch.reject", func(context.Context, *goja.Runtime) {
				_ = reject(vm.NewGoError(err))
			})
			return
		}
		_ = runtimeServices.PostWithCustomContext(callCtx, "fetch.resolve", func(context.Context, *goja.Runtime) {
			value, valueErr := responseValue(vm, data, expect)
			if valueErr != nil {
				_ = reject(valueErr)
				return
			}
			_ = resolve(value)
		})
	}()
	return vm.ToValue(promise)
}

func headersFromValue(vm *goja.Runtime, value goja.Value) map[string]string {
	out := map[string]string{}
	obj := value.ToObject(vm)
	for _, key := range obj.Keys() {
		v := obj.Get(key)
		if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
			continue
		}
		out[key] = v.String()
	}
	return out
}

func setDefaultHeader(headers map[string]string, name, value string) {
	for key := range headers {
		if strings.EqualFold(key, name) {
			return
		}
	}
	headers[name] = value
}

func cloneHeaders(headers http.Header) map[string][]string {
	out := map[string][]string{}
	for key, values := range headers {
		out[key] = append([]string(nil), values...)
	}
	return out
}

func resolveURL(baseURL, path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("request path is required")
	}
	if parsed, err := url.Parse(path); err == nil && parsed.IsAbs() {
		return parsed.String(), nil
	}
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return "", fmt.Errorf("client baseUrl is required for relative request paths")
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("client baseUrl: %w", err)
	}
	if !strings.HasSuffix(base.Path, "/") {
		base.Path += "/"
	}
	rel, err := url.Parse(strings.TrimPrefix(path, "/"))
	if err != nil {
		return "", err
	}
	return base.ResolveReference(rel).String(), nil
}

func redactError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("fetch request failed: %w", err)
}

func init() {
	modules.Register(New())
}
