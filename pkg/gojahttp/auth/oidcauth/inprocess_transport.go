package oidcauth

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"strings"
)

// InProcessIssuerTransport serves back-channel requests for one public OIDC
// issuer directly through an HTTP handler. It is intended for an OIDC provider
// mounted in the same process as its relying party, before the public listener
// is accepting connections.
//
// The transport is deliberately not a general in-process HTTP client. It only
// accepts absolute HTTP(S) URLs on the issuer's exact origin and below the
// issuer's path. All other requests fail closed.
type InProcessIssuerTransport struct {
	scheme     string
	host       string
	pathPrefix string
	handler    http.Handler
}

var _ http.RoundTripper = (*InProcessIssuerTransport)(nil)

// NewInProcessIssuerTransport validates the public issuer URL and binds it to
// handler. issuerURL must not contain userinfo, query parameters, or a fragment.
func NewInProcessIssuerTransport(issuerURL string, handler http.Handler) (*InProcessIssuerTransport, error) {
	if handler == nil {
		return nil, fmt.Errorf("oidcauth: in-process issuer handler is required")
	}
	issuer, err := url.Parse(strings.TrimSpace(issuerURL))
	if err != nil {
		return nil, fmt.Errorf("oidcauth: parse in-process issuer URL: %w", err)
	}
	if issuer.Scheme != "http" && issuer.Scheme != "https" {
		return nil, fmt.Errorf("oidcauth: in-process issuer URL scheme must be http or https")
	}
	if issuer.Host == "" || issuer.User != nil || issuer.RawQuery != "" || issuer.Fragment != "" {
		return nil, fmt.Errorf("oidcauth: in-process issuer URL must be an absolute origin/path without userinfo, query, or fragment")
	}
	prefix := strings.TrimSuffix(path.Clean("/"+strings.TrimPrefix(issuer.EscapedPath(), "/")), "/")
	if prefix == "." {
		prefix = ""
	}
	return &InProcessIssuerTransport{
		scheme:     strings.ToLower(issuer.Scheme),
		host:       strings.ToLower(issuer.Host),
		pathPrefix: prefix,
		handler:    handler,
	}, nil
}

// RoundTrip executes an allowed request synchronously against the bound
// handler and returns a buffered response.
func (t *InProcessIssuerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t == nil || t.handler == nil {
		return nil, fmt.Errorf("oidcauth: in-process issuer transport is not initialized")
	}
	if req == nil || req.URL == nil || !req.URL.IsAbs() {
		return nil, fmt.Errorf("oidcauth: in-process issuer request must use an absolute URL")
	}
	if req.URL.User != nil || strings.ToLower(req.URL.Scheme) != t.scheme || strings.ToLower(req.URL.Host) != t.host {
		return nil, fmt.Errorf("oidcauth: in-process issuer request origin is not allowed")
	}
	escapedPath := req.URL.EscapedPath()
	if t.pathPrefix != "" && escapedPath != t.pathPrefix && !strings.HasPrefix(escapedPath, t.pathPrefix+"/") {
		return nil, fmt.Errorf("oidcauth: in-process issuer request path is outside the issuer")
	}

	recorder := httptest.NewRecorder()
	serverRequest := req.Clone(req.Context())
	serverRequest.RequestURI = req.URL.RequestURI()
	if serverRequest.RemoteAddr == "" {
		// A RoundTripper receives a client-side request, while an http.Handler
		// expects server-side connection metadata. Use an explicit loopback peer
		// for this same-process, exact-origin transport so address resolvers and
		// rate-limit keys retain their normal server contract.
		serverRequest.RemoteAddr = "127.0.0.1:0"
	}
	t.handler.ServeHTTP(recorder, serverRequest)
	response := recorder.Result()
	response.Request = req
	return response, nil
}
