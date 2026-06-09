package hotreload

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
)

const defaultCloseTimeout = 5 * time.Second

type Runtime interface {
	Close(context.Context) error
}

type LoadFunc func(context.Context, Candidate) (Runtime, error)

type SmokeFunc func(context.Context, *Snapshot) error

type Options struct {
	HostOptions  gojahttp.HostOptions
	Load         LoadFunc
	Smoke        SmokeFunc
	CloseGrace   time.Duration
	CloseTimeout time.Duration
	Now          func() time.Time
}

type Candidate struct {
	Version int64
	Host    *gojahttp.Host
}

type Snapshot struct {
	Version  int64
	Host     *gojahttp.Host
	Runtime  Runtime
	Routes   []gojahttp.RouteDescriptor
	LoadedAt time.Time
}

type Status struct {
	Ready                  bool                       `json:"ready"`
	ActiveVersion          int64                      `json:"activeVersion"`
	LastReloadAt           time.Time                  `json:"lastReloadAt,omitempty"`
	LastSuccessfulReloadAt time.Time                  `json:"lastSuccessfulReloadAt,omitempty"`
	LastError              string                     `json:"lastError,omitempty"`
	Routes                 []gojahttp.RouteDescriptor `json:"routes,omitempty"`
}

type Manager struct {
	opts Options

	active      atomic.Pointer[Snapshot]
	nextVersion atomic.Int64
	reloadMu    sync.Mutex

	statusMu sync.RWMutex
	status   Status
}

func NewManager(opts Options) (*Manager, error) {
	if opts.Load == nil {
		return nil, fmt.Errorf("hotreload manager requires Load function")
	}
	if opts.CloseTimeout < 0 {
		return nil, fmt.Errorf("close timeout must not be negative")
	}
	if opts.CloseGrace < 0 {
		return nil, fmt.Errorf("close grace must not be negative")
	}
	if opts.CloseTimeout == 0 {
		opts.CloseTimeout = defaultCloseTimeout
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	return &Manager{opts: opts}, nil
}

func MustNewManager(opts Options) *Manager {
	m, err := NewManager(opts)
	if err != nil {
		panic(err)
	}
	return m
}

func (m *Manager) Reload(ctx context.Context) (*Snapshot, error) {
	if m == nil {
		return nil, fmt.Errorf("hotreload manager is nil")
	}
	m.reloadMu.Lock()
	defer m.reloadMu.Unlock()

	version := m.nextVersion.Add(1)
	candidate := Candidate{Version: version, Host: gojahttp.NewHost(m.opts.HostOptions)}
	m.recordAttempt()

	runtime, err := m.opts.Load(ctx, candidate)
	if err != nil {
		m.recordFailure(err)
		return nil, err
	}
	if runtime == nil {
		err := fmt.Errorf("hotreload load returned nil runtime")
		m.recordFailure(err)
		return nil, err
	}

	snapshot := &Snapshot{
		Version:  version,
		Host:     candidate.Host,
		Runtime:  runtime,
		Routes:   cloneRoutes(candidate.Host.Routes()),
		LoadedAt: m.opts.Now(),
	}
	if m.opts.Smoke != nil {
		if err := m.opts.Smoke(ctx, snapshot); err != nil {
			_ = m.closeSnapshot(ctx, snapshot)
			m.recordFailure(err)
			return nil, err
		}
	}

	old := m.active.Swap(snapshot)
	m.recordSuccess(snapshot)
	if old != nil {
		m.closeRetired(old)
	}
	return snapshot, nil
}

func (m *Manager) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	snapshot := m.Active()
	if snapshot == nil || snapshot.Host == nil {
		http.Error(w, "runtime not ready", http.StatusServiceUnavailable)
		return
	}
	snapshot.Host.ServeHTTP(w, r)
}

func (m *Manager) Active() *Snapshot {
	if m == nil {
		return nil
	}
	return m.active.Load()
}

func (m *Manager) Status() Status {
	if m == nil {
		return Status{}
	}
	m.statusMu.RLock()
	defer m.statusMu.RUnlock()
	status := m.status
	status.Routes = cloneRoutes(status.Routes)
	return status
}

func (m *Manager) Close(ctx context.Context) error {
	if m == nil {
		return nil
	}
	snapshot := m.active.Swap(nil)
	m.statusMu.Lock()
	m.status.Ready = false
	m.status.Routes = nil
	m.statusMu.Unlock()
	return m.closeSnapshot(ctx, snapshot)
}

func (m *Manager) closeRetired(snapshot *Snapshot) {
	if snapshot == nil {
		return
	}
	go func() {
		if m.opts.CloseGrace > 0 {
			time.Sleep(m.opts.CloseGrace)
		}
		_ = m.closeSnapshot(context.Background(), snapshot)
	}()
}

func (m *Manager) closeSnapshot(ctx context.Context, snapshot *Snapshot) error {
	if snapshot == nil || snapshot.Runtime == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if m.opts.CloseTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, m.opts.CloseTimeout)
		defer cancel()
	}
	return snapshot.Runtime.Close(ctx)
}

func (m *Manager) recordAttempt() {
	m.statusMu.Lock()
	defer m.statusMu.Unlock()
	m.status.LastReloadAt = m.opts.Now()
}

func (m *Manager) recordFailure(err error) {
	m.statusMu.Lock()
	defer m.statusMu.Unlock()
	if err != nil {
		m.status.LastError = err.Error()
	}
}

func (m *Manager) recordSuccess(snapshot *Snapshot) {
	m.statusMu.Lock()
	defer m.statusMu.Unlock()
	m.status.Ready = true
	m.status.ActiveVersion = snapshot.Version
	m.status.LastSuccessfulReloadAt = snapshot.LoadedAt
	m.status.LastError = ""
	m.status.Routes = cloneRoutes(snapshot.Routes)
}

func cloneRoutes(routes []gojahttp.RouteDescriptor) []gojahttp.RouteDescriptor {
	if len(routes) == 0 {
		return nil
	}
	out := make([]gojahttp.RouteDescriptor, len(routes))
	copy(out, routes)
	return out
}
