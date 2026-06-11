package gojahttp

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/dop251/goja"
)

type Renderer func(*goja.Runtime, goja.Value) (string, error)

type RequestDTO struct {
	Method  string
	URL     string
	Path    string
	Query   map[string]any
	Params  map[string]string
	Headers map[string]string
	Cookies map[string]string
	Session *SessionDTO
	IP      string
	Body    any
	RawBody string
}

func (r *RequestDTO) Map() map[string]any {
	return map[string]any{
		"method":  r.Method,
		"url":     r.URL,
		"path":    r.Path,
		"query":   r.Query,
		"params":  r.Params,
		"headers": r.Headers,
		"cookies": r.Cookies,
		"session": r.Session.Map(),
		"ip":      r.IP,
		"body":    r.Body,
		"rawBody": r.RawBody,
	}
}

func NewRequestDTO(r *http.Request, params map[string]string, session *SessionDTO) (*RequestDTO, error) {
	body, raw, err := parseBody(r)
	if err != nil {
		return nil, err
	}
	query := map[string]any{}
	for k, vals := range r.URL.Query() {
		if len(vals) == 1 {
			query[k] = vals[0]
		} else {
			query[k] = vals
		}
	}
	headers := map[string]string{}
	for k, vals := range r.Header {
		headers[k] = strings.Join(vals, ", ")
	}
	cookies := map[string]string{}
	for _, c := range r.Cookies() {
		cookies[c.Name] = c.Value
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}
	return &RequestDTO{Method: r.Method, URL: r.URL.String(), Path: r.URL.Path, Query: query, Params: params, Headers: headers, Cookies: cookies, Session: session, IP: ip, Body: body, RawBody: raw}, nil
}

type Response struct {
	mu       sync.Mutex
	w        http.ResponseWriter
	renderer Renderer
	status   int
	headers  map[string]string
	sent     bool
}

func NewResponse(w http.ResponseWriter, renderer Renderer) *Response {
	return &Response{w: w, renderer: renderer, status: http.StatusOK, headers: map[string]string{}}
}

func (r *Response) Sent() bool { r.mu.Lock(); defer r.mu.Unlock(); return r.sent }

func (r *Response) JSObject(vm *goja.Runtime) *goja.Object {
	obj := vm.NewObject()
	_ = obj.Set("status", func(code int) *goja.Object { r.setStatus(code); return obj })
	_ = obj.Set("set", func(name, value string) *goja.Object { r.setHeader(name, value); return obj })
	_ = obj.Set("type", func(value string) *goja.Object { r.setHeader("Content-Type", value); return obj })
	_ = obj.Set("json", func(v goja.Value) error { return r.JSON(vm, v) })
	_ = obj.Set("send", func(v goja.Value) error { return r.Send(vm, v) })
	_ = obj.Set("html", func(v goja.Value) error { return r.HTML(vm, v) })
	_ = obj.Set("redirect", func(call goja.FunctionCall) goja.Value {
		if err := r.Redirect(call); err != nil {
			panic(vm.NewGoError(err))
		}
		return goja.Undefined()
	})
	_ = obj.Set("end", func() error { return r.End() })
	return obj
}

func (r *Response) setStatus(code int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.sent {
		r.status = code
	}
}
func (r *Response) setHeader(k, v string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.sent {
		r.headers[k] = v
	}
}

func (r *Response) applyLocked() {
	for k, v := range r.headers {
		r.w.Header().Set(k, v)
	}
	r.w.WriteHeader(r.status)
	r.sent = true
}

func (r *Response) Send(vm *goja.Runtime, v goja.Value) error {
	if goja.IsUndefined(v) || goja.IsNull(v) {
		return r.End()
	}
	if s, ok := v.Export().(string); ok {
		return r.writeString(s)
	}
	return r.JSON(vm, v)
}

func (r *Response) HTML(vm *goja.Runtime, v goja.Value) error {
	if r.renderer == nil {
		return fmt.Errorf("no HTML renderer configured")
	}
	html, err := r.renderer(vm, v)
	if err != nil {
		return err
	}
	r.setHeader("Content-Type", "text/html; charset=utf-8")
	return r.writeString(html)
}

func (r *Response) JSON(vm *goja.Runtime, v goja.Value) error {
	payload, err := json.Marshal(v.Export())
	if err != nil {
		return err
	}
	payload = append(payload, '\n')

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.sent {
		return nil
	}
	r.headers["Content-Type"] = "application/json"
	r.headers["Content-Length"] = strconv.Itoa(len(payload))
	r.applyLocked()
	_, err = r.w.Write(payload)
	return err
}

func (r *Response) writeString(s string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.sent {
		return nil
	}
	if r.headers["Content-Type"] == "" {
		trim := strings.TrimSpace(s)
		if strings.HasPrefix(trim, "<") {
			r.headers["Content-Type"] = "text/html; charset=utf-8"
		} else {
			r.headers["Content-Type"] = "text/plain; charset=utf-8"
		}
	}
	payload := []byte(s)
	r.headers["Content-Length"] = strconv.Itoa(len(payload))
	r.applyLocked()
	_, err := r.w.Write(payload)
	return err
}

func (r *Response) Redirect(call goja.FunctionCall) error {
	status := http.StatusFound
	url := ""
	if len(call.Arguments) == 1 {
		url = call.Argument(0).String()
	}
	if len(call.Arguments) >= 2 {
		status = int(call.Argument(0).ToInteger())
		url = call.Argument(1).String()
	}
	if url == "" {
		return fmt.Errorf("redirect URL is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.sent {
		return nil
	}
	r.w.Header().Set("Location", url)
	r.status = status
	r.applyLocked()
	return nil
}

func (r *Response) End() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.sent {
		return nil
	}
	r.applyLocked()
	return nil
}
