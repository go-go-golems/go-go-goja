package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestIsLoopbackListenAddress(t *testing.T) {
	tests := []struct {
		addr string
		want bool
	}{
		{addr: "127.0.0.1:3090", want: true},
		{addr: "[::1]:3090", want: true},
		{addr: "localhost:3090", want: true},
		{addr: "0.0.0.0:3090", want: false},
		{addr: "[::]:3090", want: false},
		{addr: ":3090", want: false},
	}
	for _, test := range tests {
		t.Run(test.addr, func(t *testing.T) {
			got, err := isLoopbackListenAddress(test.addr)
			if err != nil {
				t.Fatalf("classify address: %v", err)
			}
			if got != test.want {
				t.Fatalf("isLoopbackListenAddress(%q)=%v, want %v", test.addr, got, test.want)
			}
		})
	}
	if _, err := isLoopbackListenAddress("not-an-address"); err == nil {
		t.Fatal("expected malformed listen address to fail")
	}
}

func TestValidateServeListenAddressRequiresRemoteAcknowledgement(t *testing.T) {
	if err := validateServeListenAddress("0.0.0.0:3090", false); err == nil || !strings.Contains(err.Error(), "--allow-remote") {
		t.Fatalf("expected explicit remote acknowledgement error, got %v", err)
	}
	if err := validateServeListenAddress("0.0.0.0:3090", true); err != nil {
		t.Fatalf("allow acknowledged remote listener: %v", err)
	}
	if err := validateServeListenAddress("127.0.0.1:3090", false); err != nil {
		t.Fatalf("allow loopback by default: %v", err)
	}
}

func TestServeHelpDocumentsRemoteAcknowledgement(t *testing.T) {
	out := &bytes.Buffer{}
	root, err := newRootCommand(out)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	root.SetArgs([]string{"serve", "--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute serve help: %v", err)
	}
	if rendered := out.String(); !strings.Contains(rendered, "--allow-remote") || !strings.Contains(rendered, "authentication") {
		t.Fatalf("serve help does not explain remote acknowledgement: %s", rendered)
	}
}
