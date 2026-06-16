package main

import "testing"

func TestResolveRedirectURLFromPublicBaseURL(t *testing.T) {
	got, err := resolveRedirectURL(serveSettings{PublicBaseURL: "https://goja-auth.yolo.scapegoat.dev/"})
	if err != nil {
		t.Fatalf("resolve redirect URL: %v", err)
	}
	want := "https://goja-auth.yolo.scapegoat.dev/auth/callback"
	if got != want {
		t.Fatalf("redirect URL = %q, want %q", got, want)
	}
}

func TestResolveRedirectURLOverride(t *testing.T) {
	got, err := resolveRedirectURL(serveSettings{
		PublicBaseURL: "https://ignored.example.com",
		RedirectURL:   "https://goja-auth.yolo.scapegoat.dev/custom/callback",
	})
	if err != nil {
		t.Fatalf("resolve redirect URL: %v", err)
	}
	want := "https://goja-auth.yolo.scapegoat.dev/custom/callback"
	if got != want {
		t.Fatalf("redirect URL = %q, want %q", got, want)
	}
}

func TestResolveRedirectURLRejectsHTTPUnlessLocalInsecure(t *testing.T) {
	_, err := resolveRedirectURL(serveSettings{PublicBaseURL: "http://goja-auth.yolo.scapegoat.dev"})
	if err == nil {
		t.Fatal("expected public HTTP URL to be rejected")
	}
}

func TestResolveRedirectURLAllowsLocalHTTPWhenInsecure(t *testing.T) {
	got, err := resolveRedirectURL(serveSettings{PublicBaseURL: "http://127.0.0.1:8790", AllowInsecureHTTP: true})
	if err != nil {
		t.Fatalf("resolve redirect URL: %v", err)
	}
	want := "http://127.0.0.1:8790/auth/callback"
	if got != want {
		t.Fatalf("redirect URL = %q, want %q", got, want)
	}
}

func TestResolveRedirectURLRequiresPublicBaseOrOverride(t *testing.T) {
	_, err := resolveRedirectURL(serveSettings{})
	if err == nil {
		t.Fatal("expected missing public-base-url/redirect-url error")
	}
}
