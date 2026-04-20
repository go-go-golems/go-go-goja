package replsession

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/engine"
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
	factory            *engine.Factory
	logger             zerolog.Logger
	store              Persistence
	sessions           map[string]*sessionState
	defaultSessionOpts SessionOptions
}

type sessionState struct {
	id          string
	profile     string
	policy      SessionPolicy
	createdAt   time.Time
	runtime     *engine.Runtime
	logger      zerolog.Logger
	mu          sync.Mutex
	nextCellID  int
	cells       []*cellState
	bindings    map[string]*bindingState
	consoleSink []ConsoleEvent
	ignored     map[string]struct{}
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

// Option configures a Service.
type Option func(*Service)

// WithPersistence configures the service to persist sessions and evaluations.
func WithPersistence(store Persistence) Option {
	return func(service *Service) {
		service.store = store
		service.defaultSessionOpts = PersistentSessionOptions()
	}
}

// WithDefaultSessionOptions configures the default options used by CreateSession.
func WithDefaultSessionOptions(opts SessionOptions) Option {
	return func(service *Service) {
		service.defaultSessionOpts = NormalizeSessionOptions(opts)
	}
}

// NewService creates a new session service backed by the supplied runtime factory.
func NewService(factory *engine.Factory, logger zerolog.Logger, opts ...Option) *Service {
	if factory == nil {
		panic("replsession: factory is nil")
	}
	service := &Service{
		factory:            factory,
		logger:             logger,
		sessions:           map[string]*sessionState{},
		defaultSessionOpts: InteractiveSessionOptions(),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(service)
		}
	}
	return service
}

// CreateSession allocates a fresh runtime using the service defaults and returns its initial summary.
func (s *Service) CreateSession(ctx context.Context) (*SessionSummary, error) {
	return s.CreateSessionWithOptions(ctx, SessionOptions{})
}

// CreateSessionWithOptions allocates a fresh runtime using explicit session options.
func (s *Service) CreateSessionWithOptions(ctx context.Context, opts SessionOptions) (*SessionSummary, error) {
	resolved := s.resolveSessionOptions(opts)
	if resolved.Policy.PersistenceEnabled() && s.store == nil {
		return nil, errors.New("create session: persistence enabled but no store configured")
	}

	id := strings.TrimSpace(resolved.ID)
	explicitID := id != ""
	if id == "" {
		id = newDefaultSessionID()
	}
	if explicitID {
		s.mu.RLock()
		_, exists := s.sessions[id]
		s.mu.RUnlock()
		if exists {
			return nil, errors.Errorf("create session: session %q already exists", id)
		}
	}

	rt, err := s.factory.NewRuntime(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "create runtime")
	}
	createdAt := resolved.CreatedAt.UTC()
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	state := &sessionState{
		id:        id,
		profile:   resolved.Profile,
		policy:    NormalizeSessionPolicy(resolved.Policy),
		createdAt: createdAt,
		runtime:   rt,
		logger:    s.logger.With().Str("session", id).Logger(),
		bindings:  map[string]*bindingState{},
		ignored:   map[string]struct{}{},
	}
	if state.policy.Observe.ConsoleCapture {
		if err := state.installConsoleCapture(ctx); err != nil {
			_ = rt.Close(ctx)
			return nil, errors.Wrap(err, "install console capture")
		}
	}
	if state.policy.Observe.JSDocExtraction {
		if err := state.installDocSentinels(ctx); err != nil {
			_ = rt.Close(ctx)
			return nil, errors.Wrap(err, "install doc sentinels")
		}
	}
	if state.policy.Persist.Enabled {
		metadataJSON, err := resolved.SessionMetadataJSON()
		if err != nil {
			_ = rt.Close(ctx)
			return nil, errors.Wrap(err, "persist session metadata")
		}
		if err := s.store.CreateSession(ctx, repldb.SessionRecord{
			SessionID:    id,
			CreatedAt:    state.createdAt,
			UpdatedAt:    state.createdAt,
			EngineKind:   "goja",
			MetadataJSON: metadataJSON,
		}); err != nil {
			_ = rt.Close(ctx)
			return nil, errors.Wrap(err, "persist session")
		}
	}

	s.mu.Lock()
	if _, exists := s.sessions[id]; exists {
		s.mu.Unlock()
		_ = rt.Close(ctx)
		return nil, errors.Errorf("create session: session %q already exists", id)
	}
	s.sessions[id] = state
	s.mu.Unlock()

	return state.buildSummary(ctx), nil
}

// Snapshot returns the current session summary.
func (s *Service) Snapshot(ctx context.Context, sessionID string) (*SessionSummary, error) {
	state, err := s.getSession(sessionID)
	if err != nil {
		return nil, err
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	return state.buildSummary(ctx), nil
}

// WithRuntime runs fn while holding the target session lock, allowing callers
// to inspect the live runtime without bypassing session ownership rules.
func (s *Service) WithRuntime(ctx context.Context, sessionID string, fn func(*engine.Runtime) error) error {
	_ = ctx
	if fn == nil {
		return errors.New("with runtime: callback is nil")
	}
	state, err := s.getSession(sessionID)
	if err != nil {
		return err
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	return fn(state.runtime)
}

// DeleteSession shuts down a session and removes it from the service.
func (s *Service) DeleteSession(ctx context.Context, sessionID string) error {
	s.mu.Lock()
	state, ok := s.sessions[sessionID]
	if ok {
		delete(s.sessions, sessionID)
	}
	s.mu.Unlock()
	if !ok {
		return ErrSessionNotFound
	}

	var persistErr error
	if state.policy.Persist.Enabled {
		persistErr = s.store.DeleteSession(ctx, sessionID, time.Now().UTC())
	}
	closeErr := state.runtime.Close(ctx)

	if persistErr != nil {
		if closeErr != nil {
			return errors.Wrapf(persistErr, "persist session deletion (runtime close also failed: %v)", closeErr)
		}
		return errors.Wrap(persistErr, "persist session deletion")
	}
	return closeErr
}

// RestoreSession rebuilds a live session by replaying persisted source cells.
func (s *Service) RestoreSession(ctx context.Context, opts SessionOptions, history []string) (*SessionSummary, error) {
	resolved := s.resolveSessionOptions(opts)
	if strings.TrimSpace(resolved.ID) == "" {
		return nil, errors.New("restore session: session id is empty")
	}
	if state, err := s.getSession(resolved.ID); err == nil {
		state.mu.Lock()
		defer state.mu.Unlock()
		return state.buildSummary(ctx), nil
	}

	replayOpts := resolved
	replayOpts.ID = ""
	replayOpts.Policy.Persist = PersistPolicy{}
	tmpService := NewService(s.factory, s.logger, WithDefaultSessionOptions(replayOpts))
	tmpSummary, err := tmpService.CreateSessionWithOptions(ctx, replayOpts)
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
			_ = tmpState.runtime.Close(ctx)
		}
	}()

	for idx, source := range history {
		if _, err := tmpService.Evaluate(ctx, tmpSummary.ID, source); err != nil {
			return nil, errors.Wrapf(err, "restore session: replay cell %d", idx+1)
		}
	}

	tmpState.mu.Lock()
	tmpState.id = resolved.ID
	tmpState.profile = resolved.Profile
	tmpState.policy = NormalizeSessionPolicy(resolved.Policy)
	if !resolved.CreatedAt.IsZero() {
		tmpState.createdAt = resolved.CreatedAt.UTC()
	}
	tmpState.logger = s.logger.With().Str("session", resolved.ID).Logger()
	tmpState.mu.Unlock()

	s.mu.Lock()
	if existing, ok := s.sessions[resolved.ID]; ok {
		s.mu.Unlock()
		_ = tmpState.runtime.Close(ctx)
		existing.mu.Lock()
		defer existing.mu.Unlock()
		return existing.buildSummary(ctx), nil
	}
	delete(tmpService.sessions, tmpSummary.ID)
	s.sessions[resolved.ID] = tmpState
	s.mu.Unlock()

	restoreFailed = false
	tmpState.mu.Lock()
	defer tmpState.mu.Unlock()
	return tmpState.buildSummary(ctx), nil
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
	return NormalizeSessionOptions(base)
}

func newDefaultSessionID() string {
	return "session-" + uuid.NewString()
}

func (s *Service) getSession(sessionID string) (*sessionState, error) {
	s.mu.RLock()
	state, ok := s.sessions[sessionID]
	s.mu.RUnlock()
	if !ok {
		return nil, ErrSessionNotFound
	}
	return state, nil
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
	timeout := NormalizeSessionPolicy(policy).Eval.Timeout()
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
