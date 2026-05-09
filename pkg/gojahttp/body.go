package gojahttp

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

func parseBody(r *http.Request) (any, string, error) {
	if r.Body == nil {
		return nil, "", nil
	}
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, "", err
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
	if strings.Contains(ct, "application/x-www-form-urlencoded") || strings.Contains(ct, "multipart/form-data") {
		r.Body = io.NopCloser(strings.NewReader(raw))
		if err := r.ParseForm(); err != nil {
			return nil, raw, err
		}
		m := map[string]any{}
		for k, vals := range r.PostForm {
			if len(vals) == 1 {
				m[k] = vals[0]
			} else {
				m[k] = vals
			}
		}
		return m, raw, nil
	}
	return raw, raw, nil
}
