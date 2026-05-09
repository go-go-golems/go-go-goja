package gojahttp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	maxRequestBodyBytes      = 64 << 20
	multipartFormMemoryLimit = 32 << 20
)

func parseBody(r *http.Request) (any, string, error) {
	if r.Body == nil {
		return nil, "", nil
	}
	limited := &io.LimitedReader{R: r.Body, N: maxRequestBodyBytes + 1}
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, "", err
	}
	if int64(len(data)) > maxRequestBodyBytes {
		return nil, "", fmt.Errorf("request body exceeds %d bytes", maxRequestBodyBytes)
	}
	raw := string(data)
	ct := strings.ToLower(r.Header.Get("Content-Type"))
	if len(data) == 0 {
		return nil, raw, nil
	}
	if strings.Contains(ct, "application/json") {
		var v any
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, raw, err
		}
		return v, raw, nil
	}
	if strings.Contains(ct, "application/x-www-form-urlencoded") {
		r.Body = io.NopCloser(strings.NewReader(raw))
		if err := r.ParseForm(); err != nil {
			return nil, raw, err
		}
		return postFormMap(r), raw, nil
	}
	if strings.Contains(ct, "multipart/form-data") {
		r.Body = io.NopCloser(strings.NewReader(raw))
		// Request body is capped above and ParseMultipartForm uses a bounded in-memory budget.
		if err := r.ParseMultipartForm(multipartFormMemoryLimit); err != nil { // #nosec G120
			return nil, raw, err
		}
		return postFormMap(r), raw, nil
	}
	return raw, raw, nil
}

func postFormMap(r *http.Request) map[string]any {
	m := map[string]any{}
	for k, vals := range r.PostForm {
		if len(vals) == 1 {
			m[k] = vals[0]
		} else {
			m[k] = vals
		}
	}
	return m
}
