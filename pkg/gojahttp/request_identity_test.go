package gojahttp_test

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
)

func TestTrustedProxyResolver(t *testing.T) {
	trusted := netip.MustParsePrefix("10.42.0.0/16")
	for _, test := range []struct {
		name       string
		mode       gojahttp.ProxyMode
		peer       string
		xff        string
		wantClient string
		wantProxy  bool
		wantErr    bool
	}{
		{name: "direct ignores forged header", mode: gojahttp.ProxyModeDirect, peer: "192.0.2.10:1234", xff: "198.51.100.7", wantClient: "192.0.2.10"},
		{name: "trusted peer uses client", mode: gojahttp.ProxyModeTrustedForwarded, peer: "10.42.0.3:1234", xff: "198.51.100.7", wantClient: "198.51.100.7", wantProxy: true},
		{name: "trusted chain walks right to left", mode: gojahttp.ProxyModeTrustedForwarded, peer: "10.42.0.3:1234", xff: "198.51.100.7, 10.42.0.8", wantClient: "198.51.100.7", wantProxy: true},
		{name: "untrusted peer ignores header", mode: gojahttp.ProxyModeTrustedForwarded, peer: "192.0.2.10:1234", xff: "198.51.100.7", wantClient: "192.0.2.10"},
		{name: "trusted malformed chain fails", mode: gojahttp.ProxyModeTrustedForwarded, peer: "10.42.0.3:1234", xff: "not-an-ip", wantErr: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
			req.RemoteAddr = test.peer
			req.Header.Set("X-Forwarded-For", test.xff)
			identity, err := (gojahttp.TrustedProxyResolver{Mode: test.mode, TrustedPrefixes: []netip.Prefix{trusted}}).Resolve(req)
			if test.wantErr {
				if err == nil {
					t.Fatal("Resolve error = nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			if got := identity.ClientIP.String(); got != test.wantClient {
				t.Fatalf("ClientIP = %q, want %q", got, test.wantClient)
			}
			if identity.ViaProxy != test.wantProxy {
				t.Fatalf("ViaProxy = %v, want %v", identity.ViaProxy, test.wantProxy)
			}
		})
	}
}

func TestRequestIdentityMiddlewareProjectsClientIP(t *testing.T) {
	resolver := gojahttp.TrustedProxyResolver{Mode: gojahttp.ProxyModeTrustedForwarded, TrustedPrefixes: []netip.Prefix{netip.MustParsePrefix("10.42.0.0/16")}}
	handler := gojahttp.RequestIdentityMiddleware(resolver, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := gojahttp.RequestClientIP(r); got != "198.51.100.9" {
			t.Fatalf("RequestClientIP = %q", got)
		}
		request, err := gojahttp.NewRequestDTO(r, nil, nil)
		if err != nil {
			t.Fatalf("NewRequestDTO: %v", err)
		}
		if request.IP != "198.51.100.9" {
			t.Fatalf("RequestDTO.IP = %q", request.IP)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
	req.RemoteAddr = "10.42.0.2:1234"
	req.Header.Set("X-Forwarded-For", "198.51.100.9")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d", recorder.Code)
	}
}
