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

// App combines the live session kernel with the durable session store.
type App struct {
	service *replsession.Service
	store   *repldb.Store
}

// New creates a restore-aware persistent REPL application façade.
func New(factory *engine.Factory, store *repldb.Store, logger zerolog.Logger) *App {
	if factory == nil {
		panic("replapi: factory is nil")
	}
	if store == nil {
		panic("replapi: store is nil")
	}
	return &App{
		service: replsession.NewService(factory, logger, replsession.WithPersistence(store)),
		store:   store,
	}
}

func (a *App) CreateSession(ctx context.Context) (*replsession.SessionSummary, error) {
	return a.service.CreateSession(ctx)
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
	record, err := a.store.LoadSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	history, err := a.store.LoadReplaySource(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	return a.service.RestoreSession(ctx, sessionID, record.CreatedAt, history)
}

func (a *App) DeleteSession(ctx context.Context, sessionID string) error {
	err := a.service.DeleteSession(ctx, sessionID)
	if err == nil {
		return nil
	}
	if !errors.Is(err, replsession.ErrSessionNotFound) {
		return err
	}
	if _, loadErr := a.store.LoadSession(ctx, sessionID); loadErr != nil {
		return loadErr
	}
	return a.store.DeleteSession(ctx, sessionID, time.Now().UTC())
}

func (a *App) ListSessions(ctx context.Context) ([]repldb.SessionRecord, error) {
	return a.store.ListSessions(ctx)
}

func (a *App) History(ctx context.Context, sessionID string) ([]repldb.EvaluationRecord, error) {
	return a.store.LoadEvaluations(ctx, sessionID)
}

func (a *App) Export(ctx context.Context, sessionID string) (*repldb.SessionExport, error) {
	return a.store.ExportSession(ctx, sessionID)
}

func (a *App) ReplaySource(ctx context.Context, sessionID string) ([]string, error) {
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

func (a *App) ensureLiveSession(ctx context.Context, sessionID string) (*replsession.SessionSummary, error) {
	summary, err := a.service.Snapshot(ctx, sessionID)
	if err == nil {
		return summary, nil
	}
	if !errors.Is(err, replsession.ErrSessionNotFound) {
		return nil, err
	}
	return a.Restore(ctx, sessionID)
}
