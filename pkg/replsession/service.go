package replsession

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/pkg/engine"
	inspectorruntime "github.com/go-go-golems/go-go-goja/pkg/inspector/runtime"
	"github.com/go-go-golems/go-go-goja/pkg/jsparse"
	"github.com/go-go-golems/go-go-goja/pkg/repldb"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

const (
	defaultASTRowLimit         = 512
	defaultCSTRowLimit         = 512
	defaultOwnPropertyLimit    = 20
	defaultPrototypeLevelLimit = 4
	defaultPrototypePropLimit  = 12
)

// ErrSessionNotFound is returned when a requested session ID does not exist.
var ErrSessionNotFound = errors.New("replsession: session not found")

// ErrEvaluationTimeout is returned when one cell exceeds its configured execution deadline.
var ErrEvaluationTimeout = errors.New("replsession: evaluation timed out")

// Service manages persistent REPL sessions and their backing runtimes.
type Service struct {
	mu                 sync.RWMutex
	factory            *engine.RuntimeFactory
	logger             zerolog.Logger
	store              Persistence
	sessions           map[string]*sessionState
	defaultSessionOpts SessionOptions
	lifetimeParent     context.Context
	lifetimeCtx        context.Context
	lifetimeCancel     context.CancelCauseFunc
	phase              ServicePhase
	closeAttempt       chan struct{}
	closeAccum         error
	closeErr           error
	leaseStore         LeasePersistence
	ownerID            string
	now                func() time.Time
	leaseTTL           time.Duration
}

type sessionState struct {
	id          string
	profile     string
	policy      SessionPolicy
	createdAt   time.Time
	runtime     *engine.Runtime
	logger      zerolog.Logger
	nextCellID  int
	cells       []*cellState
	bindings    map[string]*bindingState
	consoleSink []ConsoleEvent
	ignored     map[string]struct{}

	gate        chan struct{}
	stopGate    chan struct{}
	lifecycleMu sync.Mutex
	phase       SessionPhase
	ctx         context.Context
	cancel      context.CancelCauseFunc
	closeErr    error

	health        SessionHealth
	healthCause   error
	pendingCommit *pendingCommit
	inFlightCell  *cellState
	lease         *repldb.SessionLease
}

type cellState struct {
	report   *CellReport
	analysis *jsparse.AnalysisResult
}

type bindingState struct {
	Name            string
	Kind            jsparse.BindingKind
	Origin          string
	DeclaredInCell  int
	LastUpdatedCell int
	DeclaredLine    int
	DeclaredSnippet string
	Static          *BindingStaticView
	Runtime         BindingRuntimeView
}

// Persistence is the durable write surface used by the session service.
type Persistence interface {
	CreateSession(ctx context.Context, record repldb.SessionRecord) error
	DeleteSession(ctx context.Context, sessionID string, deletedAt time.Time) error
	PersistEvaluation(ctx context.Context, record repldb.EvaluationRecord) error
}

// LeasePersistence is the ownership/fencing surface used when a host enables
// multi-process persistent-session coordination.
type LeasePersistence interface {
	AcquireSessionLease(ctx context.Context, sessionID string, ownerID string, now time.Time, ttl time.Duration) (repldb.SessionLease, error)
	RenewSessionLease(ctx context.Context, lease repldb.SessionLease, now time.Time, ttl time.Duration) (repldb.SessionLease, error)
	ReleaseSessionLease(ctx context.Context, lease repldb.SessionLease) error
	PersistEvaluationFenced(ctx context.Context, record repldb.EvaluationRecord, fence repldb.WriteFence, now time.Time) error
}

// Option configures a Service.
type Option func(*Service)

// WithPersistence configures the service to persist sessions and evaluations.
func WithPersistence(store Persistence) Option {
	return func(service *Service) {
		service.store = store
		service.defaultSessionOpts = PersistentSessionOptions()
	}
}

// WithLeaseOwnership enables per-session leases and fenced durable appends.
func WithLeaseOwnership(store LeasePersistence, ownerID string, now func() time.Time, ttl time.Duration) Option {
	return func(service *Service) {
		service.leaseStore = store
		service.ownerID = strings.TrimSpace(ownerID)
		service.now = now
		service.leaseTTL = ttl
	}
}

// WithDefaultSessionOptions configures the default options used by CreateSession.
func WithDefaultSessionOptions(opts SessionOptions) Option {
	return func(service *Service) {
		service.defaultSessionOpts = NormalizeSessionOptions(opts)
	}
}

// WithLifetimeContext sets the parent for service and session-owned resources.
// Create/restore operation contexts remain startup-only contexts.
func WithLifetimeContext(ctx context.Context) Option {
	return func(service *Service) {
		service.lifetimeParent = nonNilContext(ctx)
	}
}

// NewService creates a new session service backed by the supplied runtime factory.
func NewService(factory *engine.RuntimeFactory, logger zerolog.Logger, opts ...Option) *Service {
	if factory == nil {
		panic("replsession: factory is nil")
	}
	service := &Service{
		factory:            factory,
		logger:             logger,
		sessions:           map[string]*sessionState{},
		defaultSessionOpts: InteractiveSessionOptions(),
		lifetimeParent:     context.Background(),
		phase:              ServicePhaseOpen,
		now:                func() time.Time { return time.Now().UTC() },
		leaseTTL:           30 * time.Second,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(service)
		}
	}
	service.lifetimeCtx, service.lifetimeCancel = context.WithCancelCause(service.lifetimeParent)
	return service
}

// CreateSession allocates a fresh runtime using the service defaults and returns its initial summary.
func (s *Service) CreateSession(ctx context.Context) (*SessionSummary, error) {
	return s.CreateSessionWithOptions(ctx, SessionOptions{})
}

// CreateSessionWithOptions allocates a fresh runtime using explicit session options.
// ctx controls startup work only; the runtime lifetime is owned by the service.
func (s *Service) CreateSessionWithOptions(ctx context.Context, opts SessionOptions) (*SessionSummary, error) {
	ctx = nonNilContext(ctx)
	resolved := s.resolveSessionOptions(opts)
	if resolved.Policy.PersistenceEnabled() && s.store == nil {
		return nil, errors.New("create session: persistence enabled but no store configured")
	}

	id := strings.TrimSpace(resolved.ID)
	explicitID := id != ""
	if id == "" {
		id = newDefaultSessionID()
	}

	s.mu.RLock()
	phase := s.phase
	_, exists := s.sessions[id]
	s.mu.RUnlock()
	if phase != ServicePhaseOpen {
		return nil, servicePhaseError(phase)
	}
	if explicitID && exists {
		return nil, errors.Errorf("create session: session %q already exists", id)
	}

	sessionCtx, sessionCancel := context.WithCancelCause(s.lifetimeCtx)
	rt, err := s.factory.NewRuntime(engine.WithStartupContext(ctx), engine.WithLifetimeContext(sessionCtx))
	if err != nil {
		sessionCancel(err)
		return nil, errors.Wrap(err, "create runtime")
	}
	var state *sessionState
	cleanup := func(cause error) {
		sessionCancel(cause)
		cleanupCtx := context.WithoutCancel(ctx)
		if state != nil {
			_ = s.releaseSessionLease(cleanupCtx, state)
		}
		_ = rt.Close(cleanupCtx)
	}

	createdAt := resolved.CreatedAt.UTC()
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	state = &sessionState{
		id:        id,
		profile:   resolved.Profile,
		policy:    NormalizeSessionPolicy(resolved.Policy),
		createdAt: createdAt,
		runtime:   rt,
		logger:    s.logger.With().Str("session", id).Logger(),
		bindings:  map[string]*bindingState{},
		ignored:   map[string]struct{}{},
		gate:      newCapacityOneGate(),
		stopGate:  newCapacityOneGate(),
		phase:     SessionPhaseActive,
		health:    SessionHealthHealthy,
		ctx:       sessionCtx,
		cancel:    sessionCancel,
	}
	if state.policy.Observe.ConsoleCapture {
		if err := state.installConsoleCapture(ctx); err != nil {
			cleanup(err)
			return nil, errors.Wrap(err, "install console capture")
		}
	}
	if state.policy.Observe.JSDocExtraction {
		if err := state.installDocSentinels(ctx); err != nil {
			cleanup(err)
			return nil, errors.Wrap(err, "install doc sentinels")
		}
	}
	if state.policy.Persist.Enabled {
		metadataJSON, err := resolved.SessionMetadataJSON()
		if err != nil {
			cleanup(err)
			return nil, errors.Wrap(err, "persist session metadata")
		}
		if err := s.store.CreateSession(ctx, repldb.SessionRecord{
			SessionID:    id,
			CreatedAt:    state.createdAt,
			UpdatedAt:    state.createdAt,
			EngineKind:   "goja",
			MetadataJSON: metadataJSON,
		}); err != nil {
			cleanup(err)
			return nil, errors.Wrap(err, "persist session")
		}
		lease, err := s.acquireSessionLease(ctx, id, state.policy)
		if err != nil {
			cleanup(err)
			return nil, errors.Wrap(err, "acquire session ownership")
		}
		state.lease = lease
	}

	summary := state.buildSummary(ctx)
	s.mu.Lock()
	if s.phase != ServicePhaseOpen {
		phase = s.phase
		s.mu.Unlock()
		cleanup(servicePhaseError(phase))
		return nil, servicePhaseError(phase)
	}
	if _, exists := s.sessions[id]; exists {
		s.mu.Unlock()
		cleanup(ErrSessionClosing)
		return nil, errors.Errorf("create session: session %q already exists", id)
	}
	s.sessions[id] = state
	s.mu.Unlock()

	return summary, nil
}

// Snapshot returns the current session summary.
func (s *Service) Snapshot(ctx context.Context, sessionID string) (*SessionSummary, error) {
	state, err := s.getSession(sessionID)
	if err != nil {
		return nil, err
	}
	op, err := state.beginOperation(ctx)
	if err != nil {
		return nil, err
	}
	defer op.Release()
	return state.buildSummary(op.Context()), nil
}

// WithRuntime runs fn while owning the target session operation gate. The
// runtime must not escape the callback, and fn must not re-enter this service
// for the same session. The callback must honor opCtx so unload/close can stop it.
func (s *Service) WithRuntime(ctx context.Context, sessionID string, fn func(context.Context, *engine.Runtime) error) error {
	if fn == nil {
		return errors.New("with runtime: callback is nil")
	}
	state, err := s.getSession(sessionID)
	if err != nil {
		return err
	}
	op, err := state.beginOperation(ctx)
	if err != nil {
		return err
	}
	defer op.Release()
	if err := state.evaluationHealthError(); err != nil {
		return err
	}
	guardCtx, stopGuard, err := s.startSessionLeaseGuard(op.Context(), state)
	if err != nil {
		return err
	}
	callbackErr := fn(guardCtx, state.runtime)
	if guardErr := stopGuard(); guardErr != nil {
		state.markFenced(guardErr)
		return state.evaluationHealthError()
	}
	return callbackErr
}

// RestoreSession rebuilds a live session by replaying persisted source cells.
// ctx controls load/replay startup work; the restored runtime retains a
// service-owned lifetime context after this method returns.
func (s *Service) RestoreSession(ctx context.Context, opts SessionOptions, history []string) (*SessionSummary, error) {
	return s.restoreSession(ctx, opts, history, nil)
}

// RestoreSessionWithLease restores using ownership acquired before durable
// history was read. On success the live session assumes responsibility for release.
func (s *Service) RestoreSessionWithLease(ctx context.Context, opts SessionOptions, history []string, ownedLease repldb.SessionLease) (*SessionSummary, error) {
	return s.restoreSession(ctx, opts, history, &ownedLease)
}

func (s *Service) restoreSession(ctx context.Context, opts SessionOptions, history []string, providedLease *repldb.SessionLease) (*SessionSummary, error) {
	ctx = nonNilContext(ctx)
	resolved := s.resolveSessionOptions(opts)
	lease := providedLease
	leaseTransferred := false
	defer func() {
		if lease != nil && !leaseTransferred && s.leaseStore != nil {
			_ = s.leaseStore.ReleaseSessionLease(context.WithoutCancel(ctx), *lease)
		}
	}()
	if strings.TrimSpace(resolved.ID) == "" {
		return nil, errors.New("restore session: session id is empty")
	}
	if state, err := s.getSession(resolved.ID); err == nil {
		if lease != nil {
			leaseTransferred = true // existing state has the same app owner token
		}
		op, err := state.beginOperation(ctx)
		if err != nil {
			return nil, err
		}
		defer op.Release()
		return state.buildSummary(op.Context()), nil
	} else if !errors.Is(err, ErrSessionNotFound) {
		return nil, err
	}

	s.mu.RLock()
	phase := s.phase
	s.mu.RUnlock()
	if phase != ServicePhaseOpen {
		return nil, servicePhaseError(phase)
	}

	if lease == nil {
		var err error
		lease, err = s.acquireSessionLease(ctx, resolved.ID, resolved.Policy)
		if err != nil {
			return nil, errors.Wrap(err, "restore session: acquire ownership")
		}
	} else if lease.SessionID != resolved.ID || lease.OwnerID != s.ownerID {
		return nil, errors.New("restore session: provided lease does not match session owner")
	}
	replayCtx := ctx
	stopLeaseGuard := func() error { return nil }
	if lease != nil {
		renewed, err := s.leaseStore.RenewSessionLease(ctx, *lease, s.nowUTC(), s.leaseTTL)
		if err != nil {
			return nil, errors.Wrap(err, "restore session: verify ownership")
		}
		lease = &renewed
		replayCtx, stopLeaseGuard = s.startLeaseGuard(ctx, *lease)
	}
	defer func() { _ = stopLeaseGuard() }()

	replayOpts := resolved
	replayOpts.ID = ""
	replayOpts.Policy.Persist = PersistPolicy{}
	tmpService := NewService(
		s.factory,
		s.logger,
		WithLifetimeContext(s.lifetimeCtx),
		WithDefaultSessionOptions(replayOpts),
	)
	tmpSummary, err := tmpService.CreateSessionWithOptions(replayCtx, replayOpts)
	if err != nil {
		return nil, errors.Wrap(err, "restore session: create replay runtime")
	}
	tmpState, err := tmpService.getSession(tmpSummary.ID)
	if err != nil {
		return nil, errors.Wrap(err, "restore session: get replay state")
	}

	restoreFailed := true
	defer func() {
		if restoreFailed {
			tmpState.cancel(ErrSessionClosing)
			_ = tmpState.runtime.Close(context.WithoutCancel(replayCtx))
		}
	}()

	for idx, source := range history {
		if _, err := tmpService.Evaluate(replayCtx, tmpSummary.ID, source); err != nil {
			return nil, errors.Wrapf(err, "restore session: replay cell %d", idx+1)
		}
	}

	tmpState.id = resolved.ID
	tmpState.profile = resolved.Profile
	tmpState.policy = NormalizeSessionPolicy(resolved.Policy)
	if !resolved.CreatedAt.IsZero() {
		tmpState.createdAt = resolved.CreatedAt.UTC()
	}
	if guardErr := stopLeaseGuard(); guardErr != nil {
		return nil, errors.Wrap(guardErr, "restore session: renew ownership")
	}
	tmpState.logger = s.logger.With().Str("session", resolved.ID).Logger()
	tmpState.lease = lease
	summary := tmpState.buildSummary(ctx)

	s.mu.Lock()
	if s.phase != ServicePhaseOpen {
		phase = s.phase
		s.mu.Unlock()
		return nil, servicePhaseError(phase)
	}
	if existing, ok := s.sessions[resolved.ID]; ok {
		s.mu.Unlock()
		tmpState.cancel(ErrSessionClosing)
		_ = tmpState.runtime.Close(context.WithoutCancel(replayCtx))
		restoreFailed = false
		leaseTransferred = true // same app owner/epoch remains with the published state
		op, err := existing.beginOperation(ctx)
		if err != nil {
			return nil, err
		}
		defer op.Release()
		return existing.buildSummary(op.Context()), nil
	}
	tmpService.mu.Lock()
	delete(tmpService.sessions, tmpSummary.ID)
	tmpService.mu.Unlock()
	s.sessions[resolved.ID] = tmpState
	s.mu.Unlock()

	restoreFailed = false
	leaseTransferred = true
	return summary, nil
}

func (s *Service) resolveSessionOptions(opts SessionOptions) SessionOptions {
	base := NormalizeSessionOptions(s.defaultSessionOpts)
	if strings.TrimSpace(base.Profile) == "" {
		base = NormalizeSessionOptions(InteractiveSessionOptions())
	}

	if strings.TrimSpace(opts.ID) != "" {
		base.ID = strings.TrimSpace(opts.ID)
	}
	if !opts.CreatedAt.IsZero() {
		base.CreatedAt = opts.CreatedAt.UTC()
	}
	if strings.TrimSpace(opts.Profile) != "" {
		base.Profile = strings.TrimSpace(opts.Profile)
	}
	if !opts.Policy.IsZero() {
		base.Policy = NormalizeSessionPolicy(opts.Policy)
	}
	if base.CreatedAt.IsZero() {
		base.CreatedAt = time.Now().UTC()
	}
	return base
}

func newDefaultSessionID() string {
	return "session-" + uuid.NewString()
}

func (s *Service) getSession(sessionID string) (*sessionState, error) {
	s.mu.RLock()
	state, ok := s.sessions[sessionID]
	phase := s.phase
	s.mu.RUnlock()
	if ok {
		return state, nil
	}
	if phase != ServicePhaseOpen {
		return nil, servicePhaseError(phase)
	}
	return nil, ErrSessionNotFound
}

func (s *sessionState) installConsoleCapture(ctx context.Context) error {
	_, err := s.runtime.Owner.Call(ctx, "replsession.install-console", func(_ context.Context, vm *goja.Runtime) (any, error) {
		consoleObj := vm.NewObject()
		setMethod := func(name string, kind string) error {
			return consoleObj.Set(name, func(call goja.FunctionCall) goja.Value {
				s.consoleSink = append(s.consoleSink, ConsoleEvent{Kind: kind, Message: formatConsoleMessage(call.Arguments, vm)})
				return goja.Undefined()
			})
		}
		for _, item := range []struct {
			Name string
			Kind string
		}{
			{Name: "log", Kind: "log"},
			{Name: "info", Kind: "info"},
			{Name: "debug", Kind: "debug"},
			{Name: "warn", Kind: "warn"},
			{Name: "error", Kind: "error"},
			{Name: "table", Kind: "table"},
		} {
			if setErr := setMethod(item.Name, item.Kind); setErr != nil {
				return nil, setErr
			}
		}
		if err := vm.Set("console", consoleObj); err != nil {
			return nil, err
		}
		return nil, nil
	})
	return err
}

func (s *sessionState) installDocSentinels(ctx context.Context) error {
	for _, name := range []string{"__doc__", "__package__", "__example__", "doc"} {
		s.ignored[name] = struct{}{}
	}

	_, err := s.runtime.Owner.Call(ctx, "replsession.install-doc-sentinels", func(_ context.Context, vm *goja.Runtime) (any, error) {
		noOp := func(goja.FunctionCall) goja.Value { return goja.Undefined() }
		for _, name := range []string{"__doc__", "__package__", "__example__"} {
			if err := vm.Set(name, noOp); err != nil {
				return nil, err
			}
		}
		if err := vm.Set("doc", func(goja.FunctionCall) goja.Value { return goja.Undefined() }); err != nil {
			return nil, err
		}
		return nil, nil
	})
	return err
}

func formatConsoleMessage(args []goja.Value, vm *goja.Runtime) string {
	parts := make([]string, 0, len(args))
	for _, arg := range args {
		parts = append(parts, inspectorruntime.ValuePreview(arg, vm, 120))
	}
	return strings.Join(parts, " ")
}

func executionStatus(err error, helperError bool) string {
	if errors.Is(err, ErrEvaluationTimeout) || errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	if err != nil {
		return "runtime-error"
	}
	if helperError {
		return "helper-error"
	}
	return "ok"
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func evaluationContext(ctx context.Context, policy SessionPolicy) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	timeout := policy.Eval.Timeout()
	if timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeoutCause(ctx, timeout, errors.Wrapf(ErrEvaluationTimeout, "evaluation exceeded %s", timeout))
}

func evaluationContextError(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	if cause := context.Cause(ctx); cause != nil {
		return cause
	}
	return ctx.Err()
}

func firstDiagnosticMessage(diagnostics []DiagnosticView) string {
	if len(diagnostics) == 0 {
		return "parse error"
	}
	return diagnostics[0].Message
}

func dedupeSortedStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	if len(out) == 0 {
		return []string{}
	}
	return out
}
