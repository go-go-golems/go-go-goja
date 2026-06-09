package hotreload

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestWatchReloadsAfterFileChange(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "routes.js")
	if err := os.WriteFile(path, []byte("version 1"), 0o644); err != nil {
		t.Fatalf("write initial file: %v", err)
	}

	var loadCount atomic.Int64
	manager := MustNewManager(Options{Load: func(_ context.Context, candidate Candidate) (Runtime, error) {
		version := loadCount.Add(1)
		candidate.Host.RegisterStaticHandler("/", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = fmt.Fprintf(w, "version:%d", version)
		}))
		return &fakeRuntime{}, nil
	}})
	if _, err := manager.Reload(context.Background()); err != nil {
		t.Fatalf("initial reload: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	reloaded := make(chan *Snapshot, 1)
	errs := make(chan error, 1)
	go func() {
		err := manager.Watch(ctx, WatchOptions{
			Roots:        []string{dir},
			Extensions:   []string{".js"},
			PollInterval: 10 * time.Millisecond,
			Debounce:     10 * time.Millisecond,
			OnReload: func(snapshot *Snapshot) {
				reloaded <- snapshot
			},
			OnError: func(err error) {
				errs <- err
			},
		})
		if err != nil && err != context.Canceled {
			errs <- err
		}
	}()

	// Give Watch enough time to take its initial snapshot before mutating the file.
	time.Sleep(50 * time.Millisecond)
	if err := os.WriteFile(path, []byte("version 2 changed"), 0o644); err != nil {
		t.Fatalf("write changed file: %v", err)
	}

	select {
	case snapshot := <-reloaded:
		if snapshot.Version != 2 {
			t.Fatalf("snapshot version = %d, want 2", snapshot.Version)
		}
	case err := <-errs:
		t.Fatalf("watch error: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for reload")
	}
	assertManagerResponse(t, manager, "version:2")
}
