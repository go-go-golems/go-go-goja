package replapi

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/go-go-golems/go-go-goja/pkg/repldb"
	"github.com/go-go-golems/go-go-goja/pkg/replsession"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

// App combines the live session kernel with the optional durable session store.
type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now().UTC() }

type App struct {
	config    Config
	service   *replsession.Service
	store     *repldb.Store
	ctx       context.Context
	cancel    context.CancelCauseFunc
	lifecycle appLifecycle
	ownerID   string
}

// OwnerID returns the process-unique lease identity for diagnostics.
func (a *App) OwnerID() string {
	if a == nil {
		return ""
	}
	return a.ownerID
}

// New creates a configurable REPL application facade owned by parent. Request
// contexts passed to individual methods never become runtime lifetime parents.
func New(parent context.Context, factory *engine.RuntimeFactory, logger zerolog.Logger, opts ...Option) (*App, error) {
	config := DefaultConfig()
	for _, opt := range opts {
		if opt != nil {
			opt(&config)
		}
	}
	return NewWithConfig(parent, factory, logger, config)
}

// NewWithConfig creates a configurable REPL application facade owned by parent.
func NewWithConfig(parent context.Context, factory *engine.RuntimeFactory, logger zerolog.Logger, config Config) (*App, error) {
	if factory == nil {
		return nil, errors.New("replapi: factory is nil")
	}

	config, err := normalizeConfig(config)
	if err != nil {
		return nil, err
	}
	if err := validateConfig(config); err != nil {
		return nil, err
	}
	if strings.TrimSpace(config.ownerID) == "" {
		config.ownerID = uuid.NewString()
	}
	if config.Clock == nil {
		config.Clock = systemClock{}
	}

	appCtx, appCancel := context.WithCancelCause(nonNilAppContext(parent))
	serviceOpts := []replsession.Option{
		replsession.WithLifetimeContext(appCtx),
		replsession.WithDefaultSessionOptions(config.SessionOptions),
	}
	if config.Store != nil {
		serviceOpts = append(serviceOpts,
			replsession.WithPersistence(config.Store),
			replsession.WithLeaseOwnership(config.Store, config.ownerID, config.Clock.Now, config.LeaseTTL),
		)
	}

	return &App{
		config:  config,
		service: replsession.NewService(factory, logger, serviceOpts...),
		store:   config.Store,
		ctx:     appCtx,
		cancel:  appCancel,
		ownerID: config.ownerID,
		lifecycle: appLifecycle{
			phase: AppPhaseOpen,
		},
	}, nil
}

// CreateSession creates and publishes a live session using the app defaults.
// ctx controls startup only; the app parent owns the resulting runtime lifetime.
func (a *App) CreateSession(ctx context.Context) (*replsession.SessionSummary, error) {
	return a.CreateSessionWithOptions(ctx, SessionOverrides{})
}

// CreateSessionWithOptions creates a session after validating profile and full-policy overrides.
func (a *App) CreateSessionWithOptions(ctx context.Context, opts SessionOverrides) (*replsession.SessionSummary, error) {
	if err := a.ensureOpen(); err != nil {
		return nil, err
	}
	resolved, err := resolveCreateSessionOptions(a.config, opts)
	if err != nil {
		return nil, err
	}
	if resolved.Policy.PersistenceEnabled() && a.store == nil {
		return nil, errors.New("replapi: persistence requested but no store configured")
	}
	summary, err := a.service.CreateSessionWithOptions(ctx, resolved)
	return summary, a.translateLifecycleError(err)
}

// Evaluate executes one serialized cell. A persistence commit failure returns
// both a populated response and a typed replsession.CommitError; callers must
// retry the retained commit or recover the session, never rerun the source.
func (a *App) Evaluate(ctx context.Context, sessionID string, source string) (*replsession.EvaluateResponse, error) {
	if err := a.ensureOpen(); err != nil {
		return nil, err
	}
	if _, err := a.ensureLiveSession(ctx, sessionID); err != nil {
		return nil, err
	}
	response, err := a.service.Evaluate(ctx, sessionID, source)
	return response, a.translateLifecycleError(err)
}

// Snapshot returns current live state, auto-restoring durable state when configured.
func (a *App) Snapshot(ctx context.Context, sessionID string) (*replsession.SessionSummary, error) {
	if err := a.ensureOpen(); err != nil {
		return nil, err
	}
	return a.ensureLiveSession(ctx, sessionID)
}

// SessionHealth returns whether the live session is healthy, degraded, or fenced.
// SessionHealth reports whether the live VM is healthy, degraded, or fenced.
func (a *App) SessionHealth(ctx context.Context, sessionID string) (replsession.SessionHealth, error) {
	if err := a.ensureOpen(); err != nil {
		return "", err
	}
	health, err := a.service.SessionHealth(ctx, sessionID)
	return health, a.translateLifecycleError(err)
}

// RetryPendingCommit retries the exact retained durable record without rerunning JavaScript.
// RetryPendingCommit persists the exact retained record without rerunning JavaScript.
func (a *App) RetryPendingCommit(ctx context.Context, sessionID string) (*replsession.EvaluateResponse, error) {
	if err := a.ensureOpen(); err != nil {
		return nil, err
	}
	response, err := a.service.RetryPendingCommit(ctx, sessionID)
	return response, a.translateLifecycleError(err)
}

// Restore acquires durable ownership and reconstructs a fresh VM by source replay.
func (a *App) Restore(ctx context.Context, sessionID string) (*replsession.SessionSummary, error) {
	if err := a.ensureOpen(); err != nil {
		return nil, err
	}
	if a.store == nil {
		return nil, errors.New("replapi: restore requires a store")
	}
	lease, err := a.store.AcquireSessionLease(ctx, sessionID, a.ownerID, a.config.Clock.Now(), a.config.LeaseTTL)
	if err != nil {
		return nil, err
	}
	releaseOnReadFailure := func() { _ = a.store.ReleaseSessionLease(context.WithoutCancel(ctx), lease) }
	record, err := a.store.LoadSession(ctx, sessionID)
	if err != nil {
		releaseOnReadFailure()
		return nil, err
	}
	history, err := a.store.LoadReplaySource(ctx, sessionID)
	if err != nil {
		releaseOnReadFailure()
		return nil, err
	}
	restoreOptions := a.restoreOptionsForRecord(record)
	summary, err := a.service.RestoreSessionWithLease(ctx, restoreOptions, history, lease)
	return summary, a.translateLifecycleError(err)
}

// RecoverSession discards a suspect live VM and restores the last durable head.
// Source that executed but failed to commit is not replayed.
// RecoverSession discards a degraded or fenced VM and restores the durable head.
func (a *App) RecoverSession(ctx context.Context, sessionID string) (*replsession.SessionSummary, error) {
	if err := a.ensureOpen(); err != nil {
		return nil, err
	}
	if a.store == nil {
		return nil, errors.New("replapi: recover requires a store")
	}
	if err := a.service.UnloadSession(ctx, sessionID); err != nil && !errors.Is(err, replsession.ErrSessionNotFound) {
		return nil, a.translateLifecycleError(err)
	}
	return a.Restore(ctx, sessionID)
}

// DeleteSession closes live state, releases ownership, and soft-deletes durable history.
// Use UnloadSession when durable history must remain visible.
func (a *App) DeleteSession(ctx context.Context, sessionID string) error {
	if err := a.ensureOpen(); err != nil {
		return err
	}
	err := a.service.DeleteSession(ctx, sessionID)
	if err == nil {
		return nil
	}
	if !errors.Is(err, replsession.ErrSessionNotFound) {
		return a.translateLifecycleError(err)
	}
	if a.store == nil {
		return err
	}
	if _, loadErr := a.store.LoadSession(ctx, sessionID); loadErr != nil {
		return loadErr
	}
	lease, leaseErr := a.store.AcquireSessionLease(ctx, sessionID, a.ownerID, a.config.Clock.Now(), a.config.LeaseTTL)
	if leaseErr != nil {
		return leaseErr
	}
	deleteErr := a.store.DeleteSession(ctx, sessionID, time.Now().UTC())
	releaseErr := a.store.ReleaseSessionLease(ctx, lease)
	if deleteErr != nil {
		if releaseErr != nil {
			return errors.Wrapf(deleteErr, "delete durable session (lease release also failed: %v)", releaseErr)
		}
		return deleteErr
	}
	return releaseErr
}

func (a *App) ListSessions(ctx context.Context) ([]repldb.SessionRecord, error) {
	if err := a.ensureOpen(); err != nil {
		return nil, err
	}
	if a.store == nil {
		return nil, errors.New("replapi: list sessions requires a store")
	}
	return a.store.ListSessions(ctx)
}

func (a *App) History(ctx context.Context, sessionID string) ([]repldb.EvaluationRecord, error) {
	if err := a.ensureOpen(); err != nil {
		return nil, err
	}
	if a.store == nil {
		return nil, errors.New("replapi: history requires a store")
	}
	return a.store.LoadEvaluations(ctx, sessionID)
}

func (a *App) Export(ctx context.Context, sessionID string) (*repldb.SessionExport, error) {
	if err := a.ensureOpen(); err != nil {
		return nil, err
	}
	if a.store == nil {
		return nil, errors.New("replapi: export requires a store")
	}
	return a.store.ExportSession(ctx, sessionID)
}

func (a *App) ReplaySource(ctx context.Context, sessionID string) ([]string, error) {
	if err := a.ensureOpen(); err != nil {
		return nil, err
	}
	if a.store == nil {
		return nil, errors.New("replapi: replay source requires a store")
	}
	return a.store.LoadReplaySource(ctx, sessionID)
}

func (a *App) Bindings(ctx context.Context, sessionID string) ([]replsession.BindingView, error) {
	if err := a.ensureOpen(); err != nil {
		return nil, err
	}
	summary, err := a.ensureLiveSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	return append([]replsession.BindingView(nil), summary.Bindings...), nil
}

func (a *App) Docs(ctx context.Context, sessionID string) ([]repldb.BindingDocRecord, error) {
	if err := a.ensureOpen(); err != nil {
		return nil, err
	}
	if a.store == nil {
		return nil, errors.New("replapi: docs requires a store")
	}
	exported, err := a.store.ExportSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	docs := []repldb.BindingDocRecord{}
	for _, evaluation := range exported.Evaluations {
		docs = append(docs, evaluation.BindingDocs...)
	}
	sort.SliceStable(docs, func(i, j int) bool {
		if docs[i].SymbolName != docs[j].SymbolName {
			return docs[i].SymbolName < docs[j].SymbolName
		}
		return docs[i].CellID < docs[j].CellID
	})
	return docs, nil
}

// WithRuntime runs fn against the live runtime for one session while preserving
// replapi session ownership and auto-restore behavior. The runtime must not
// escape fn, fn must not re-enter the same session, and fn must honor opCtx so
// unload and app shutdown can cancel active work.
// WithRuntime runs fn while holding the session operation gate. The callback
// context is canceled by caller cancellation, unload, or app shutdown. Callers
// must not retain the runtime or re-enter the same session from fn.
func (a *App) WithRuntime(ctx context.Context, sessionID string, fn func(context.Context, *engine.Runtime) error) error {
	if err := a.ensureOpen(); err != nil {
		return err
	}
	if _, err := a.ensureLiveSession(ctx, sessionID); err != nil {
		return err
	}
	return a.translateLifecycleError(a.service.WithRuntime(ctx, sessionID, fn))
}

func (a *App) ensureLiveSession(ctx context.Context, sessionID string) (*replsession.SessionSummary, error) {
	summary, err := a.service.Snapshot(ctx, sessionID)
	if err == nil {
		return summary, nil
	}
	if !errors.Is(err, replsession.ErrSessionNotFound) {
		return nil, a.translateLifecycleError(err)
	}
	if !a.config.AutoRestore || a.store == nil {
		return nil, err
	}
	return a.Restore(ctx, sessionID)
}

func (a *App) restoreOptionsForRecord(record repldb.SessionRecord) replsession.SessionOptions {
	options := replsession.NormalizeSessionOptions(a.config.SessionOptions)
	options.ID = record.SessionID
	options.CreatedAt = record.CreatedAt

	if metadataOpts, ok, err := replsession.SessionOptionsFromMetadata(record.MetadataJSON); err == nil && ok {
		metadataOpts.ID = record.SessionID
		metadataOpts.CreatedAt = record.CreatedAt
		return replsession.NormalizeSessionOptions(metadataOpts)
	}

	if options.Profile == "" {
		options = replsession.PersistentSessionOptions()
		options.ID = record.SessionID
		options.CreatedAt = record.CreatedAt
	}
	return replsession.NormalizeSessionOptions(options)
}
