package gojahttp

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// RequestIdentity is the normalized network identity selected by the host.
// PeerIP is the direct TCP peer; ClientIP is the address used for security
// decisions. Forwarding headers never become trusted without a configured
// proxy policy.
type RequestIdentity struct {
	PeerIP   netip.Addr
	ClientIP netip.Addr
	ViaProxy bool
}

type requestIdentityKey struct{}

// WithRequestIdentity attaches a normalized immutable identity to a request.
func WithRequestIdentity(r *http.Request, identity RequestIdentity) *http.Request {
	if r == nil {
		return nil
	}
	return r.WithContext(context.WithValue(r.Context(), requestIdentityKey{}, identity))
}

// RequestIdentityFromContext returns the host-selected identity, if available.
func RequestIdentityFromContext(ctx context.Context) (RequestIdentity, bool) {
	identity, ok := ctx.Value(requestIdentityKey{}).(RequestIdentity)
	return identity, ok
}

// ProxyMode controls whether forwarding headers are ignored or accepted only
// from direct peers in TrustedPrefixes.
type ProxyMode string

const (
	ProxyModeDirect           ProxyMode = "direct"
	ProxyModeTrustedForwarded ProxyMode = "trusted-forwarded"
)

// TrustedProxyResolver resolves client addresses for one concrete host policy.
// It supports X-Forwarded-For because that is the deployed Traefik contract.
type TrustedProxyResolver struct {
	Mode             ProxyMode
	TrustedPrefixes  []netip.Prefix
	MaxForwardedHops int
}

// Resolve selects a request identity. In direct mode and for untrusted peers,
// forwarding headers are ignored. A malformed chain from a trusted proxy is a
// host/proxy contract failure and returns an error rather than trusting a
// partial value.
func (r TrustedProxyResolver) Resolve(req *http.Request) (RequestIdentity, error) {
	if req == nil {
		return RequestIdentity{}, fmt.Errorf("request identity: request is required")
	}
	peer, err := parseRemoteIP(req.RemoteAddr)
	if err != nil {
		return RequestIdentity{}, fmt.Errorf("request identity: %w", err)
	}
	identity := RequestIdentity{PeerIP: peer, ClientIP: peer}
	if r.Mode == "" || r.Mode == ProxyModeDirect {
		return identity, nil
	}
	if r.Mode != ProxyModeTrustedForwarded {
		return RequestIdentity{}, fmt.Errorf("request identity: unsupported proxy mode %q", r.Mode)
	}
	if !r.trusted(peer) {
		return identity, nil
	}
	value := strings.TrimSpace(req.Header.Get("X-Forwarded-For"))
	if value == "" {
		return identity, nil
	}
	hops, err := parseForwardedFor(value, r.maxHops())
	if err != nil {
		return RequestIdentity{}, fmt.Errorf("request identity: invalid X-Forwarded-For from trusted peer: %w", err)
	}
	for i := len(hops) - 1; i >= 0; i-- {
		if !r.trusted(hops[i]) {
			identity.ClientIP = hops[i]
			identity.ViaProxy = true
			return identity, nil
		}
	}
	identity.ViaProxy = true
	return identity, nil
}

func (r TrustedProxyResolver) maxHops() int {
	if r.MaxForwardedHops > 0 {
		return r.MaxForwardedHops
	}
	return 16
}

func (r TrustedProxyResolver) trusted(addr netip.Addr) bool {
	for _, prefix := range r.TrustedPrefixes {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

func parseRemoteIP(remoteAddr string) (netip.Addr, error) {
	remoteAddr = strings.TrimSpace(remoteAddr)
	if remoteAddr == "" {
		return netip.Addr{}, fmt.Errorf("missing RemoteAddr")
	}
	host, _, err := net.SplitHostPort(remoteAddr)
	if err == nil {
		remoteAddr = host
	}
	addr, err := netip.ParseAddr(remoteAddr)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("parse RemoteAddr %q: %w", remoteAddr, err)
	}
	return addr.Unmap(), nil
}

func parseForwardedFor(value string, maxHops int) ([]netip.Addr, error) {
	if len(value) > 4096 {
		return nil, fmt.Errorf("header exceeds 4096 bytes")
	}
	parts := strings.Split(value, ",")
	if len(parts) == 0 || len(parts) > maxHops {
		return nil, fmt.Errorf("header has %d hops; maximum is %d", len(parts), maxHops)
	}
	hops := make([]netip.Addr, 0, len(parts))
	for _, part := range parts {
		addr, err := netip.ParseAddr(strings.TrimSpace(part))
		if err != nil {
			return nil, fmt.Errorf("invalid hop %q: %w", part, err)
		}
		hops = append(hops, addr.Unmap())
	}
	return hops, nil
}

// RequestIdentityMiddleware resolves identity once before invoking next.
func RequestIdentityMiddleware(resolver TrustedProxyResolver, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		identity, err := resolver.Resolve(req)
		if err != nil {
			http.Error(w, "invalid forwarding headers", http.StatusBadRequest)
			return
		}
		next.ServeHTTP(w, WithRequestIdentity(req, identity))
	})
}

// RequestClientIP returns the canonical client IP when present, otherwise a
// conservative RemoteAddr-derived value for hosts which have not installed a
// request identity policy yet.
func RequestClientIP(req *http.Request) string {
	if req == nil {
		return "unknown"
	}
	if identity, ok := RequestIdentityFromContext(req.Context()); ok && identity.ClientIP.IsValid() {
		return identity.ClientIP.String()
	}
	addr, err := parseRemoteIP(req.RemoteAddr)
	if err == nil {
		return addr.String()
	}
	return "unknown"
}
