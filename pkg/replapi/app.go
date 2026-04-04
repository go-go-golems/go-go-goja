package replapi

import (
	"context"
	"sort"
	"time"

	"github.com/go-go-golems/go-go-goja/engine"
	"github.com/go-go-golems/go-go-goja/pkg/repldb"
	"github.com/go-go-golems/go-go-goja/pkg/replsession"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

// App combines the live session kernel with the optional durable session store.
type App struct {
	config  Config
	service *replsession.Service
	store   *repldb.Store
}

// New creates a configurable REPL application facade.
func New(factory *engine.Factory, logger zerolog.Logger, opts ...Option) (*App, error) {
	config := DefaultConfig()
	for _, opt := range opts {
		if opt != nil {
			opt(&config)
		}
	}
	return NewWithConfig(factory, logger, config)
}

// NewWithConfig creates a configurable REPL application facade from an explicit config.
func NewWithConfig(factory *engine.Factory, logger zerolog.Logger, config Config) (*App, error) {
	if factory == nil {
		return nil, errors.New("replapi: factory is nil")
	}

	config = normalizeConfig(config)
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	serviceOpts := []replsession.Option{
		replsession.WithDefaultSessionOptions(config.SessionOptions),
	}
	if config.Store != nil {
		serviceOpts = append(serviceOpts, replsession.WithPersistence(config.Store))
		serviceOpts = append(serviceOpts, replsession.WithDefaultSessionOptions(config.SessionOptions))
	}

	return &App{
		config:  config,
		service: replsession.NewService(factory, logger, serviceOpts...),
		store:   config.Store,
	}, nil
}

func (a *App) CreateSession(ctx context.Context) (*replsession.SessionSummary, error) {
	return a.CreateSessionWithOptions(ctx, SessionOptions{})
}

func (a *App) CreateSessionWithOptions(ctx context.Context, opts SessionOptions) (*replsession.SessionSummary, error) {
	if a == nil {
		return nil, errors.New("replapi: app is nil")
	}
	resolved := resolveSessionOptions(a.config, opts)
	if resolved.Policy.PersistenceEnabled() && a.store == nil {
		return nil, errors.New("replapi: persistence requested but no store configured")
	}
	return a.service.CreateSessionWithOptions(ctx, resolved)
}

func (a *App) Evaluate(ctx context.Context, sessionID string, source string) (*replsession.EvaluateResponse, error) {
	if _, err := a.ensureLiveSession(ctx, sessionID); err != nil {
		return nil, err
	}
	return a.service.Evaluate(ctx, sessionID, source)
}

func (a *App) Snapshot(ctx context.Context, sessionID string) (*replsession.SessionSummary, error) {
	return a.ensureLiveSession(ctx, sessionID)
}

func (a *App) Restore(ctx context.Context, sessionID string) (*replsession.SessionSummary, error) {
	if a.store == nil {
		return nil, errors.New("replapi: restore requires a store")
	}
	record, err := a.store.LoadSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	history, err := a.store.LoadReplaySource(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	restoreOptions := a.restoreOptionsForRecord(record)
	return a.service.RestoreSession(ctx, restoreOptions, history)
}

func (a *App) DeleteSession(ctx context.Context, sessionID string) error {
	err := a.service.DeleteSession(ctx, sessionID)
	if err == nil {
		return nil
	}
	if !errors.Is(err, replsession.ErrSessionNotFound) {
		return err
	}
	if a.store == nil {
		return err
	}
	if _, loadErr := a.store.LoadSession(ctx, sessionID); loadErr != nil {
		return loadErr
	}
	return a.store.DeleteSession(ctx, sessionID, time.Now().UTC())
}

func (a *App) ListSessions(ctx context.Context) ([]repldb.SessionRecord, error) {
	if a.store == nil {
		return nil, errors.New("replapi: list sessions requires a store")
	}
	return a.store.ListSessions(ctx)
}

func (a *App) History(ctx context.Context, sessionID string) ([]repldb.EvaluationRecord, error) {
	if a.store == nil {
		return nil, errors.New("replapi: history requires a store")
	}
	return a.store.LoadEvaluations(ctx, sessionID)
}

func (a *App) Export(ctx context.Context, sessionID string) (*repldb.SessionExport, error) {
	if a.store == nil {
		return nil, errors.New("replapi: export requires a store")
	}
	return a.store.ExportSession(ctx, sessionID)
}

func (a *App) ReplaySource(ctx context.Context, sessionID string) ([]string, error) {
	if a.store == nil {
		return nil, errors.New("replapi: replay source requires a store")
	}
	return a.store.LoadReplaySource(ctx, sessionID)
}

func (a *App) Bindings(ctx context.Context, sessionID string) ([]replsession.BindingView, error) {
	summary, err := a.ensureLiveSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	return append([]replsession.BindingView(nil), summary.Bindings...), nil
}

func (a *App) Docs(ctx context.Context, sessionID string) ([]repldb.BindingDocRecord, error) {
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
// replapi session ownership and auto-restore behavior.
func (a *App) WithRuntime(ctx context.Context, sessionID string, fn func(*engine.Runtime) error) error {
	if a == nil {
		return errors.New("replapi: app is nil")
	}
	if _, err := a.ensureLiveSession(ctx, sessionID); err != nil {
		return err
	}
	return a.service.WithRuntime(ctx, sessionID, fn)
}

func (a *App) ensureLiveSession(ctx context.Context, sessionID string) (*replsession.SessionSummary, error) {
	summary, err := a.service.Snapshot(ctx, sessionID)
	if err == nil {
		return summary, nil
	}
	if !errors.Is(err, replsession.ErrSessionNotFound) {
		return nil, err
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
