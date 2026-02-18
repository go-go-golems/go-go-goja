package main

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestRunServerUntilCanceled_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	srv := &http.Server{
		Addr:              "127.0.0.1:0",
		Handler:           http.NewServeMux(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	if err := runServerUntilCanceled(ctx, srv); err != nil {
		t.Fatalf("expected nil on context cancellation, got %v", err)
	}
}

func TestRunServerUntilCanceled_InvalidAddr(t *testing.T) {
	srv := &http.Server{
		Addr:              "127.0.0.1:-1",
		Handler:           http.NewServeMux(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	if err := runServerUntilCanceled(context.Background(), srv); err == nil {
		t.Fatal("expected listen error for invalid address")
	}
}
