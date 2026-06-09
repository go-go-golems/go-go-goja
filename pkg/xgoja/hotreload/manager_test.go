package hotreload

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

type fakeRuntime struct {
	closed atomic.Bool
	count  atomic.Int32
}

func (r *fakeRuntime) Close(context.Context) error {
	r.count.Add(1)
	r.closed.Store(true)
	return nil
}

func TestManagerReloadKeepsLastKnownGoodOnFailure(t *testing.T) {
	ctx := context.Background()
	brokenErr := errors.New("broken javascript")
	var desired atomic.Int64
	desired.Store(1)
	var fail atomic.Bool
	var runtimes []*fakeRuntime

	manager := MustNewManager(Options{
		Load: func(_ context.Context, candidate Candidate) (Runtime, error) {
			if fail.Load() {
				return nil, brokenErr
			}
			version := desired.Load()
			candidate.Host.Register("GET", "/version", nil)
			candidate.Host.RegisterStaticHandler("/", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = fmt.Fprintf(w, "version:%d", version)
			}))
			runtime := &fakeRuntime{}
			runtimes = append(runtimes, runtime)
			return runtime, nil
		},
		CloseTimeout: time.Second,
	})

	if _, err := manager.Reload(ctx); err != nil {
		t.Fatalf("initial reload: %v", err)
	}
	assertManagerResponse(t, manager, "version:1")
	status := manager.Status()
	if !status.Ready || status.ActiveVersion != 1 || len(status.Routes) != 1 || status.Routes[0].Pattern != "/version" {
		t.Fatalf("status after initial reload = %#v", status)
	}

	fail.Store(true)
	if _, err := manager.Reload(ctx); !errors.Is(err, brokenErr) {
		t.Fatalf("expected broken reload error, got %v", err)
	}
	assertManagerResponse(t, manager, "version:1")
	if status := manager.Status(); !status.Ready || status.ActiveVersion != 1 || status.LastError != "broken javascript" {
		t.Fatalf("status after failed reload = %#v", status)
	}

	fail.Store(false)
	desired.Store(2)
	if _, err := manager.Reload(ctx); err != nil {
		t.Fatalf("second successful reload: %v", err)
	}
	assertManagerResponse(t, manager, "version:2")
	if status := manager.Status(); !status.Ready || status.ActiveVersion != 3 || status.LastError != "" {
		t.Fatalf("status after second success = %#v", status)
	}
	eventually(t, func() bool { return runtimes[0].closed.Load() })

	if err := manager.Close(ctx); err != nil {
		t.Fatalf("close manager: %v", err)
	}
	eventually(t, func() bool { return runtimes[1].closed.Load() })
}

func TestManagerSmokeFailureClosesCandidate(t *testing.T) {
	candidateRuntime := &fakeRuntime{}
	manager := MustNewManager(Options{
		Load: func(_ context.Context, candidate Candidate) (Runtime, error) {
			candidate.Host.RegisterStaticHandler("/", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("candidate"))
			}))
			return candidateRuntime, nil
		},
		Smoke: func(context.Context, *Snapshot) error {
			return errors.New("smoke failed")
		},
	})

	if _, err := manager.Reload(context.Background()); err == nil || err.Error() != "smoke failed" {
		t.Fatalf("expected smoke error, got %v", err)
	}
	if !candidateRuntime.closed.Load() {
		t.Fatal("expected failed candidate runtime to be closed")
	}
	if status := manager.Status(); status.Ready || status.LastError != "smoke failed" {
		t.Fatalf("status after smoke failure = %#v", status)
	}
}

func assertManagerResponse(t *testing.T, manager *Manager, expected string) {
	t.Helper()
	rr := httptest.NewRecorder()
	manager.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/anything", nil))
	if rr.Code != http.StatusOK || rr.Body.String() != expected {
		t.Fatalf("response code=%d body=%q, want %q", rr.Code, rr.Body.String(), expected)
	}
}

func eventually(t *testing.T, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition not reached")
}
