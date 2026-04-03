package replsession

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/engine"
	inspectoranalysis "github.com/go-go-golems/go-go-goja/pkg/inspector/analysis"
	inspectorcore "github.com/go-go-golems/go-go-goja/pkg/inspector/core"
	inspectorruntime "github.com/go-go-golems/go-go-goja/pkg/inspector/runtime"
	"github.com/go-go-golems/go-go-goja/pkg/jsdoc/extract"
	"github.com/go-go-golems/go-go-goja/pkg/jsparse"
	"github.com/go-go-golems/go-go-goja/pkg/repldb"
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

// Service manages persistent REPL sessions and their backing runtimes.
type Service struct {
	mu       sync.RWMutex
	factory  *engine.Factory
	logger   zerolog.Logger
	store    Persistence
	nextID   uint64
	sessions map[string]*sessionState
}

type sessionState struct {
	id          string
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

type executionOutcome struct {
	Awaited        bool
	LastValue      string
	PersistedNames []string
	HelperError    bool
}

type promiseSnapshot struct {
	State  goja.PromiseState
	Result goja.Value
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
	}
}

// NewService creates a new session service backed by the supplied runtime factory.
func NewService(factory *engine.Factory, logger zerolog.Logger, opts ...Option) *Service {
	if factory == nil {
		panic("replsession: factory is nil")
	}
	service := &Service{
		factory:  factory,
		logger:   logger,
		sessions: map[string]*sessionState{},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(service)
		}
	}
	return service
}

// CreateSession allocates a fresh runtime and returns its initial summary.
func (s *Service) CreateSession(ctx context.Context) (*SessionSummary, error) {
	rt, err := s.factory.NewRuntime(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "create runtime")
	}
	id := fmt.Sprintf("session-%d", atomic.AddUint64(&s.nextID, 1))
	state := &sessionState{
		id:        id,
		createdAt: time.Now().UTC(),
		runtime:   rt,
		logger:    s.logger.With().Str("session", id).Logger(),
		bindings:  map[string]*bindingState{},
		ignored:   map[string]struct{}{},
	}
	if err := state.installConsoleCapture(ctx); err != nil {
		_ = rt.Close(ctx)
		return nil, errors.Wrap(err, "install console capture")
	}
	if err := state.installDocSentinels(ctx); err != nil {
		_ = rt.Close(ctx)
		return nil, errors.Wrap(err, "install doc sentinels")
	}
	if s.store != nil {
		if err := s.store.CreateSession(ctx, repldb.SessionRecord{
			SessionID:  id,
			CreatedAt:  state.createdAt,
			UpdatedAt:  state.createdAt,
			EngineKind: "goja",
		}); err != nil {
			_ = rt.Close(ctx)
			return nil, errors.Wrap(err, "persist session")
		}
	}

	s.mu.Lock()
	s.sessions[id] = state
	s.mu.Unlock()

	return state.buildSummary(ctx), nil
}

// Evaluate runs one cell within an existing session.
func (s *Service) Evaluate(ctx context.Context, sessionID string, source string) (*EvaluateResponse, error) {
	state, err := s.getSession(sessionID)
	if err != nil {
		return nil, err
	}

	state.mu.Lock()
	defer state.mu.Unlock()

	state.nextCellID++
	cellID := state.nextCellID
	filename := fmt.Sprintf("<repl-cell-%d>", cellID)

	var cstRoot *jsparse.TSNode
	if parser, parserErr := jsparse.NewTSParser(); parserErr == nil {
		cstRoot = parser.Parse([]byte(source))
		parser.Close()
	}

	analysis := jsparse.Analyze(filename, source, nil)
	staticReport := buildStaticReport(analysis, cstRoot, defaultASTRowLimit, defaultCSTRowLimit)
	rewrite := buildRewrite(source, analysis, cellID)

	cell := &CellReport{
		ID:        cellID,
		CreatedAt: time.Now().UTC(),
		Source:    source,
		Static:    staticReport,
		Rewrite:   rewrite,
		Provenance: []ProvenanceRecord{
			{Section: "static", Source: "jsparse.Analyze + resolver + tree-sitter snapshot", Notes: []string{"top-level bindings come from the root lexical scope", "AST rows come from the indexed node tree", "CST rows come from tree-sitter"}},
			{Section: "rewrite", Source: "async IIFE wrapper with explicit binding capture", Notes: []string{"lexical declarations stay cell-local", "declared names are returned and then mirrored back onto the runtime global object"}},
			{Section: "runtime", Source: "goja runtime snapshots before and after evaluation", Notes: []string{"global diffs come from comparing non-builtin global properties", "binding runtime metadata comes from object/property inspection"}},
		},
	}

	if analysis == nil || analysis.ParseErr != nil {
		cell.Execution = ExecutionReport{
			Status:     "parse-error",
			Error:      firstDiagnosticMessage(staticReport.Diagnostics),
			DurationMS: 0,
		}
		cell.Runtime = RuntimeReport{}
		state.cells = append(state.cells, &cellState{report: cell, analysis: analysis})
		if err := s.persistCell(ctx, state, cell); err != nil {
			return nil, err
		}
		return &EvaluateResponse{Session: state.buildSummaryLocked(), Cell: cell}, nil
	}

	beforeGlobals, err := state.snapshotGlobals(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "snapshot globals before evaluation")
	}
	state.consoleSink = nil

	start := time.Now()
	outcome, execErr := state.executeWrapped(ctx, rewrite)
	duration := time.Since(start)
	consoleEvents := append([]ConsoleEvent(nil), state.consoleSink...)
	state.consoleSink = nil

	afterGlobals, snapErr := state.snapshotGlobals(ctx)
	if snapErr != nil {
		return nil, errors.Wrap(snapErr, "snapshot globals after evaluation")
	}
	diffs, added, updated, removed := diffGlobals(beforeGlobals, afterGlobals, state.bindings)
	persistedSet := make(map[string]struct{}, len(outcome.PersistedNames))
	for _, name := range outcome.PersistedNames {
		persistedSet[name] = struct{}{}
	}

	newBindings := make([]string, 0)
	updatedBindings := make([]string, 0)
	removedBindings := make([]string, 0)
	leakedGlobals := make([]string, 0)
	for _, name := range removed {
		if _, ok := state.bindings[name]; ok {
			delete(state.bindings, name)
			removedBindings = append(removedBindings, name)
		}
	}
	for _, name := range added {
		if _, ok := persistedSet[name]; ok {
			newBindings = append(newBindings, name)
			continue
		}
		leakedGlobals = append(leakedGlobals, name)
		state.upsertRuntimeDiscoveredBinding(name, cellID, afterGlobals[name])
		newBindings = append(newBindings, name)
	}
	for _, name := range updated {
		if _, ok := persistedSet[name]; ok {
			updatedBindings = append(updatedBindings, name)
			continue
		}
		if binding := state.bindings[name]; binding != nil {
			binding.LastUpdatedCell = cellID
		} else {
			leakedGlobals = append(leakedGlobals, name)
			state.upsertRuntimeDiscoveredBinding(name, cellID, afterGlobals[name])
			newBindings = append(newBindings, name)
		}
	}
	for _, name := range outcome.PersistedNames {
		if state.bindings[name] == nil {
			newBindings = append(newBindings, name)
		}
		state.upsertDeclaredBinding(analysis, name, cellID)
	}

	if err := state.refreshBindingRuntimeDetails(ctx); err != nil {
		return nil, errors.Wrap(err, "refresh binding runtime details")
	}

	cell.Execution = ExecutionReport{
		Status:      executionStatus(execErr, outcome.HelperError),
		Result:      outcome.LastValue,
		Error:       errorString(execErr),
		DurationMS:  duration.Milliseconds(),
		Awaited:     outcome.Awaited,
		Console:     consoleEvents,
		HadSideFX:   len(diffs) > 0,
		HelperError: outcome.HelperError,
	}
	cell.Runtime = RuntimeReport{
		BeforeGlobals:    mapGlobalSnapshotViews(beforeGlobals),
		AfterGlobals:     mapGlobalSnapshotViews(afterGlobals),
		Diffs:            diffs,
		NewBindings:      dedupeSortedStrings(newBindings),
		UpdatedBindings:  dedupeSortedStrings(updatedBindings),
		RemovedBindings:  dedupeSortedStrings(removedBindings),
		LeakedGlobals:    dedupeSortedStrings(leakedGlobals),
		PersistedByWrap:  dedupeSortedStrings(outcome.PersistedNames),
		CurrentCellValue: outcome.LastValue,
	}

	state.cells = append(state.cells, &cellState{report: cell, analysis: analysis})
	if err := s.persistCell(ctx, state, cell); err != nil {
		return nil, err
	}
	return &EvaluateResponse{Session: state.buildSummaryLockedWithGlobals(afterGlobals), Cell: cell}, nil
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
	if s.store != nil {
		if err := s.store.DeleteSession(ctx, sessionID, time.Now().UTC()); err != nil {
			return errors.Wrap(err, "persist session deletion")
		}
	}
	return state.runtime.Close(ctx)
}

func (s *Service) persistCell(ctx context.Context, state *sessionState, cell *CellReport) error {
	if s.store == nil {
		return nil
	}
	if state == nil || cell == nil {
		return errors.New("persist cell: state or cell is nil")
	}

	resultJSON, err := json.Marshal(cell)
	if err != nil {
		return errors.Wrap(err, "persist cell: marshal cell report")
	}
	analysisJSON, err := json.Marshal(cell.Static)
	if err != nil {
		return errors.Wrap(err, "persist cell: marshal static report")
	}
	globalsBeforeJSON, err := json.Marshal(cell.Runtime.BeforeGlobals)
	if err != nil {
		return errors.Wrap(err, "persist cell: marshal globals before")
	}
	globalsAfterJSON, err := json.Marshal(cell.Runtime.AfterGlobals)
	if err != nil {
		return errors.Wrap(err, "persist cell: marshal globals after")
	}

	consoleEvents := make([]repldb.ConsoleEventRecord, 0, len(cell.Execution.Console))
	for idx, event := range cell.Execution.Console {
		consoleEvents = append(consoleEvents, repldb.ConsoleEventRecord{
			Stream: event.Kind,
			Seq:    idx + 1,
			Text:   event.Message,
		})
	}
	bindingVersions, bindingDocs, err := s.bindingPersistenceRecords(ctx, state, cell)
	if err != nil {
		return err
	}

	if err := s.store.PersistEvaluation(ctx, repldb.EvaluationRecord{
		SessionID:         state.id,
		CellID:            cell.ID,
		CreatedAt:         cell.CreatedAt,
		RawSource:         cell.Source,
		RewrittenSource:   cell.Rewrite.TransformedSource,
		OK:                cell.Execution.Status == "ok",
		ResultJSON:        resultJSON,
		ErrorText:         cell.Execution.Error,
		AnalysisJSON:      analysisJSON,
		GlobalsBeforeJSON: globalsBeforeJSON,
		GlobalsAfterJSON:  globalsAfterJSON,
		ConsoleEvents:     consoleEvents,
		BindingVersions:   bindingVersions,
		BindingDocs:       bindingDocs,
	}); err != nil {
		return errors.Wrap(err, "persist cell: write evaluation")
	}

	return nil
}

func (s *Service) bindingPersistenceRecords(ctx context.Context, state *sessionState, cell *CellReport) ([]repldb.BindingVersionRecord, []repldb.BindingDocRecord, error) {
	docRecords, docDigests, err := extractBindingDocs(cell)
	if err != nil {
		return nil, nil, err
	}

	changedNames := append([]string(nil), cell.Runtime.NewBindings...)
	changedNames = append(changedNames, cell.Runtime.UpdatedBindings...)
	exportSnapshots, err := state.snapshotBindingExports(ctx, changedNames)
	if err != nil {
		return nil, nil, errors.Wrap(err, "persist cell: snapshot binding exports")
	}

	versionRecords := make([]repldb.BindingVersionRecord, 0, len(changedNames)+len(cell.Runtime.RemovedBindings))
	for _, name := range dedupeSortedStrings(cell.Runtime.NewBindings) {
		record, ok, err := state.bindingVersionRecord(name, cell.ID, cell.CreatedAt, "insert", exportSnapshots[name], docDigests[name])
		if err != nil {
			return nil, nil, err
		}
		if ok {
			versionRecords = append(versionRecords, record)
		}
	}
	for _, name := range dedupeSortedStrings(cell.Runtime.UpdatedBindings) {
		record, ok, err := state.bindingVersionRecord(name, cell.ID, cell.CreatedAt, "update", exportSnapshots[name], docDigests[name])
		if err != nil {
			return nil, nil, err
		}
		if ok {
			versionRecords = append(versionRecords, record)
		}
	}
	for _, name := range dedupeSortedStrings(cell.Runtime.RemovedBindings) {
		record, ok, err := bindingRemovalRecord(cell, name, docDigests[name])
		if err != nil {
			return nil, nil, err
		}
		if ok {
			versionRecords = append(versionRecords, record)
		}
	}

	return versionRecords, docRecords, nil
}

type bindingExportSnapshot struct {
	ExportKind string
	ExportJSON string
}

func (s *sessionState) snapshotBindingExports(ctx context.Context, names []string) (map[string]bindingExportSnapshot, error) {
	names = dedupeSortedStrings(names)
	if len(names) == 0 {
		return map[string]bindingExportSnapshot{}, nil
	}

	ret, err := s.runtime.Owner.Call(ctx, "replsession.snapshot-binding-exports", func(_ context.Context, vm *goja.Runtime) (any, error) {
		out := make(map[string]bindingExportSnapshot, len(names))
		for _, name := range names {
			out[name] = classifyBindingExport(vm.Get(name), vm)
		}
		return out, nil
	})
	if err != nil {
		return nil, err
	}
	snapshots, ok := ret.(map[string]bindingExportSnapshot)
	if !ok {
		return nil, fmt.Errorf("unexpected binding export snapshot type %T", ret)
	}
	return snapshots, nil
}

func classifyBindingExport(value goja.Value, vm *goja.Runtime) bindingExportSnapshot {
	if value == nil || goja.IsUndefined(value) {
		return stringExportSnapshot("undefined")
	}
	if goja.IsNull(value) {
		return bindingExportSnapshot{ExportKind: "json", ExportJSON: "null"}
	}
	if _, ok := goja.AssertFunction(value); ok {
		return stringExportSnapshot(inspectorruntime.ValuePreview(value, vm, 120))
	}

	exported := value.Export()
	bytes, err := json.Marshal(exported)
	if err == nil {
		return bindingExportSnapshot{ExportKind: "json", ExportJSON: string(bytes)}
	}
	return stringExportSnapshot(inspectorruntime.ValuePreview(value, vm, 120))
}

func stringExportSnapshot(preview string) bindingExportSnapshot {
	bytes, err := json.Marshal(preview)
	if err != nil {
		return bindingExportSnapshot{ExportKind: "none", ExportJSON: "null"}
	}
	return bindingExportSnapshot{ExportKind: "string", ExportJSON: string(bytes)}
}

func (s *sessionState) bindingVersionRecord(name string, cellID int, createdAt time.Time, action string, exportSnapshot bindingExportSnapshot, docDigest string) (repldb.BindingVersionRecord, bool, error) {
	binding := s.bindings[name]
	if binding == nil || s.isIgnoredGlobal(name) {
		return repldb.BindingVersionRecord{}, false, nil
	}

	summaryJSON, err := json.Marshal(bindingViewFromState(binding))
	if err != nil {
		return repldb.BindingVersionRecord{}, false, errors.Wrap(err, "persist cell: marshal binding summary")
	}

	return repldb.BindingVersionRecord{
		Name:         name,
		CreatedAt:    createdAt,
		CellID:       cellID,
		Action:       action,
		RuntimeType:  binding.Runtime.ValueKind,
		DisplayValue: binding.Runtime.Preview,
		SummaryJSON:  summaryJSON,
		ExportKind:   defaultBindingExportKind(exportSnapshot.ExportKind),
		ExportJSON:   json.RawMessage(defaultBindingExportJSON(exportSnapshot.ExportJSON)),
		DocDigest:    docDigest,
	}, true, nil
}

func bindingRemovalRecord(cell *CellReport, name string, docDigest string) (repldb.BindingVersionRecord, bool, error) {
	if cell == nil {
		return repldb.BindingVersionRecord{}, false, nil
	}
	for _, diff := range cell.Runtime.Diffs {
		if diff.Name != name || diff.Change != "removed" {
			continue
		}
		summaryJSON, err := json.Marshal(diff)
		if err != nil {
			return repldb.BindingVersionRecord{}, false, errors.Wrap(err, "persist cell: marshal removal summary")
		}
		return repldb.BindingVersionRecord{
			Name:         name,
			CreatedAt:    cell.CreatedAt,
			CellID:       cell.ID,
			Action:       "remove",
			RuntimeType:  diff.BeforeKind,
			DisplayValue: diff.Before,
			SummaryJSON:  summaryJSON,
			ExportKind:   "none",
			ExportJSON:   json.RawMessage(`null`),
			DocDigest:    docDigest,
		}, true, nil
	}
	return repldb.BindingVersionRecord{}, false, nil
}

func extractBindingDocs(cell *CellReport) ([]repldb.BindingDocRecord, map[string]string, error) {
	if cell == nil || cell.Execution.Status == "parse-error" {
		return nil, map[string]string{}, nil
	}

	fileDoc, err := extract.ParseSource(fmt.Sprintf("<repl-cell-%d>", cell.ID), []byte(cell.Source))
	if err != nil {
		return nil, nil, errors.Wrap(err, "persist cell: extract jsdocex docs")
	}

	docRecords := make([]repldb.BindingDocRecord, 0, len(fileDoc.Symbols))
	docPayloads := map[string][]string{}
	for _, symbol := range fileDoc.Symbols {
		if symbol == nil || strings.TrimSpace(symbol.Name) == "" {
			continue
		}
		normalizedJSON, err := json.Marshal(symbol)
		if err != nil {
			return nil, nil, errors.Wrap(err, "persist cell: marshal symbol doc")
		}
		name := strings.TrimSpace(symbol.Name)
		docRecords = append(docRecords, repldb.BindingDocRecord{
			SymbolName:     name,
			CellID:         cell.ID,
			SourceKind:     "jsdocex",
			RawDoc:         string(normalizedJSON),
			NormalizedJSON: normalizedJSON,
		})
		docPayloads[name] = append(docPayloads[name], string(normalizedJSON))
	}

	digests := make(map[string]string, len(docPayloads))
	for name, payloads := range docPayloads {
		sort.Strings(payloads)
		h := sha256.New()
		for _, payload := range payloads {
			_, _ = h.Write([]byte(payload))
			_, _ = h.Write([]byte{'\n'})
		}
		digests[name] = hex.EncodeToString(h.Sum(nil))
	}

	return docRecords, digests, nil
}

func defaultBindingExportKind(kind string) string {
	if strings.TrimSpace(kind) == "" {
		return "none"
	}
	return kind
}

func defaultBindingExportJSON(value string) string {
	if strings.TrimSpace(value) == "" {
		return "null"
	}
	return value
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

func (s *sessionState) executeWrapped(ctx context.Context, rewrite RewriteReport) (executionOutcome, error) {
	outcome := executionOutcome{}
	value, err := s.runString(ctx, rewrite.TransformedSource)
	if err != nil {
		return outcome, err
	}
	if value == nil {
		return outcome, nil
	}
	if promise, ok := value.Export().(*goja.Promise); ok {
		outcome.Awaited = true
		value, err = s.waitPromise(ctx, promise)
		if err != nil {
			return outcome, err
		}
	}
	persisted, lastValue, helperError, err := s.persistWrappedReturn(ctx, value, rewrite.BindingHelperName, rewrite.LastHelperName)
	if err != nil {
		return outcome, err
	}
	outcome.PersistedNames = persisted
	outcome.LastValue = lastValue
	outcome.HelperError = helperError
	return outcome, nil
}

func (s *sessionState) runString(ctx context.Context, source string) (goja.Value, error) {
	ret, err := s.runtime.Owner.Call(ctx, "replsession.run-string", func(_ context.Context, vm *goja.Runtime) (any, error) {
		return vm.RunString(source)
	})
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return nil, nil
	}
	value, ok := ret.(goja.Value)
	if !ok {
		return nil, fmt.Errorf("unexpected evaluation result type %T", ret)
	}
	return value, nil
}

func (s *sessionState) waitPromise(ctx context.Context, promise *goja.Promise) (goja.Value, error) {
	for {
		ret, err := s.runtime.Owner.Call(ctx, "replsession.promise-state", func(_ context.Context, vm *goja.Runtime) (any, error) {
			return promiseSnapshot{State: promise.State(), Result: promise.Result()}, nil
		})
		if err != nil {
			return nil, err
		}
		snapshot, ok := ret.(promiseSnapshot)
		if !ok {
			return nil, fmt.Errorf("unexpected promise snapshot type %T", ret)
		}
		switch snapshot.State {
		case goja.PromiseStatePending:
			time.Sleep(5 * time.Millisecond)
			continue
		case goja.PromiseStateRejected:
			return nil, fmt.Errorf("promise rejected: %s", gojaValuePreview(snapshot.Result, s.runtime.VM))
		case goja.PromiseStateFulfilled:
			return snapshot.Result, nil
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
}

func (s *sessionState) persistWrappedReturn(ctx context.Context, value goja.Value, bindingsKey string, lastKey string) ([]string, string, bool, error) {
	ret, err := s.runtime.Owner.Call(ctx, "replsession.persist-return", func(_ context.Context, vm *goja.Runtime) (any, error) {
		if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
			return persistResult{HelperError: true, LastValue: "undefined"}, nil
		}
		obj := value.ToObject(vm)
		bindingsValue := obj.Get(bindingsKey)
		lastValue := obj.Get(lastKey)
		if bindingsValue == nil || goja.IsUndefined(bindingsValue) || goja.IsNull(bindingsValue) {
			return persistResult{HelperError: true, LastValue: gojaValuePreview(lastValue, vm)}, nil
		}
		bindingsObj := bindingsValue.ToObject(vm)
		names := bindingsObj.Keys()
		sort.Strings(names)
		for _, name := range names {
			if setErr := vm.Set(name, bindingsObj.Get(name)); setErr != nil {
				return nil, setErr
			}
		}
		return persistResult{Persisted: names, LastValue: gojaValuePreview(lastValue, vm)}, nil
	})
	if err != nil {
		return nil, "", false, err
	}
	result, ok := ret.(persistResult)
	if !ok {
		return nil, "", false, fmt.Errorf("unexpected persist result type %T", ret)
	}
	return result.Persisted, result.LastValue, result.HelperError, nil
}

type persistResult struct {
	Persisted   []string
	LastValue   string
	HelperError bool
}

func (s *sessionState) snapshotGlobals(ctx context.Context) (map[string]GlobalStateView, error) {
	ret, err := s.runtime.Owner.Call(ctx, "replsession.snapshot-globals", func(_ context.Context, vm *goja.Runtime) (any, error) {
		out := map[string]GlobalStateView{}
		global := vm.GlobalObject()
		for _, key := range global.Keys() {
			if inspectorruntime.IsBuiltinGlobal(key) || s.isIgnoredGlobal(key) {
				continue
			}
			val := global.Get(key)
			out[key] = globalStateFromValue(key, val, vm)
		}
		return out, nil
	})
	if err != nil {
		return nil, err
	}
	globals, ok := ret.(map[string]GlobalStateView)
	if !ok {
		return nil, fmt.Errorf("unexpected global snapshot type %T", ret)
	}
	return globals, nil
}

func (s *sessionState) isIgnoredGlobal(name string) bool {
	if s == nil {
		return false
	}
	_, ok := s.ignored[name]
	return ok
}

func globalStateFromValue(name string, value goja.Value, vm *goja.Runtime) GlobalStateView {
	view := GlobalStateView{
		Name:    name,
		Kind:    runtimeValueKind(value),
		Preview: gojaValuePreview(value, vm),
	}
	if obj, ok := value.(*goja.Object); ok {
		view.Identity = fmt.Sprintf("%p", obj)
		view.PropertyCount = len(obj.Keys())
	} else {
		view.Identity = view.Preview
	}
	return view
}

func diffGlobals(before map[string]GlobalStateView, after map[string]GlobalStateView, bindings map[string]*bindingState) ([]GlobalDiffView, []string, []string, []string) {
	diffs := make([]GlobalDiffView, 0)
	added := make([]string, 0)
	updated := make([]string, 0)
	removed := make([]string, 0)
	seen := map[string]struct{}{}
	for name, beforeState := range before {
		seen[name] = struct{}{}
		afterState, ok := after[name]
		if !ok {
			removed = append(removed, name)
			diffs = append(diffs, GlobalDiffView{
				Name:         name,
				Change:       "removed",
				Before:       beforeState.Preview,
				BeforeKind:   beforeState.Kind,
				SessionBound: bindings[name] != nil,
			})
			continue
		}
		if beforeState.Preview != afterState.Preview || beforeState.Kind != afterState.Kind || beforeState.Identity != afterState.Identity || beforeState.PropertyCount != afterState.PropertyCount {
			updated = append(updated, name)
			diffs = append(diffs, GlobalDiffView{
				Name:         name,
				Change:       "updated",
				Before:       beforeState.Preview,
				After:        afterState.Preview,
				BeforeKind:   beforeState.Kind,
				AfterKind:    afterState.Kind,
				SessionBound: bindings[name] != nil,
			})
		}
	}
	for name, afterState := range after {
		if _, ok := seen[name]; ok {
			continue
		}
		added = append(added, name)
		diffs = append(diffs, GlobalDiffView{
			Name:         name,
			Change:       "added",
			After:        afterState.Preview,
			AfterKind:    afterState.Kind,
			SessionBound: bindings[name] != nil,
		})
	}
	sort.Slice(diffs, func(i, j int) bool { return diffs[i].Name < diffs[j].Name })
	sort.Strings(added)
	sort.Strings(updated)
	sort.Strings(removed)
	return diffs, added, updated, removed
}

func mapGlobalSnapshotViews(globals map[string]GlobalStateView) []GlobalStateView {
	out := make([]GlobalStateView, 0, len(globals))
	for _, state := range globals {
		out = append(out, state)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *sessionState) upsertRuntimeDiscoveredBinding(name string, cellID int, state GlobalStateView) {
	kind := jsparse.BindingVar
	if state.Kind == "function" {
		kind = jsparse.BindingFunction
	}
	binding := s.bindings[name]
	if binding == nil {
		binding = &bindingState{Name: name, Kind: kind, Origin: "runtime-global-diff"}
		s.bindings[name] = binding
	}
	binding.Kind = kind
	binding.LastUpdatedCell = cellID
	if binding.DeclaredInCell == 0 {
		binding.DeclaredSnippet = "runtime-discovered global"
	}
}

func (s *sessionState) upsertDeclaredBinding(result *jsparse.AnalysisResult, name string, cellID int) {
	session := inspectoranalysis.NewSessionFromResult(result)
	globals := session.Globals()
	var extends string
	var kind jsparse.BindingKind
	for _, g := range globals {
		if g.Name == name {
			extends = g.Extends
			kind = g.Kind
			break
		}
	}
	binding := s.bindings[name]
	if binding == nil {
		binding = &bindingState{Name: name}
		s.bindings[name] = binding
	}
	binding.Kind = kind
	binding.Origin = "declared-top-level"
	binding.DeclaredInCell = cellID
	binding.LastUpdatedCell = cellID
	binding.DeclaredLine, _ = session.BindingDeclLine(name)
	binding.DeclaredSnippet = declarationSnippet(result, name)
	binding.Static = &BindingStaticView{
		References: bindingReferences(result, name),
		Extends:    extends,
	}
	for _, member := range session.FunctionMembers(name) {
		if member.Kind == "param" {
			binding.Static.Parameters = append(binding.Static.Parameters, member.Name)
		}
	}
	for _, member := range session.ClassMembers(name) {
		binding.Static.Members = append(binding.Static.Members, memberView(member))
	}
}

func memberView(member inspectorcore.Member) MemberView {
	return MemberView{
		Name:      member.Name,
		Kind:      member.Kind,
		Preview:   member.Preview,
		Inherited: member.Inherited,
		Source:    member.Source,
	}
}

func (s *sessionState) refreshBindingRuntimeDetails(ctx context.Context) error {
	ret, err := s.runtime.Owner.Call(ctx, "replsession.refresh-binding-runtime", func(_ context.Context, vm *goja.Runtime) (any, error) {
		out := map[string]BindingRuntimeView{}
		for name, binding := range s.bindings {
			val := vm.Get(name)
			view := BindingRuntimeView{
				ValueKind: runtimeValueKind(val),
				Preview:   gojaValuePreview(val, vm),
			}
			if obj, ok := val.(*goja.Object); ok {
				view.OwnProperties = ownPropertiesView(obj, vm)
				view.PrototypeChain = prototypeChainView(obj, vm)
			}
			if binding != nil && binding.Kind == jsparse.BindingFunction {
				if cell := s.cellByID(binding.DeclaredInCell); cell != nil && cell.analysis != nil {
					if mapping := inspectorruntime.MapFunctionToSource(val, vm, cell.analysis); mapping != nil {
						view.FunctionMapping = &FunctionMappingView{
							Name:      mapping.Name,
							ClassName: mapping.ClassName,
							StartLine: mapping.StartLine,
							StartCol:  mapping.StartCol,
							EndLine:   mapping.EndLine,
							EndCol:    mapping.EndCol,
							NodeID:    int(mapping.NodeID),
						}
					}
				}
			}
			out[name] = view
		}
		return out, nil
	})
	if err != nil {
		return err
	}
	views, ok := ret.(map[string]BindingRuntimeView)
	if !ok {
		return fmt.Errorf("unexpected binding runtime refresh type %T", ret)
	}
	for name, view := range views {
		if binding := s.bindings[name]; binding != nil {
			binding.Runtime = view
		}
	}
	return nil
}

func ownPropertiesView(obj *goja.Object, vm *goja.Runtime) []PropertyView {
	props := inspectorruntime.InspectObject(obj, vm)
	out := make([]PropertyView, 0, minInt(len(props), defaultOwnPropertyLimit))
	for i, prop := range props {
		if i >= defaultOwnPropertyLimit {
			break
		}
		view := PropertyView{
			Name:     prop.Name,
			Kind:     prop.Kind,
			Preview:  prop.Preview,
			IsSymbol: prop.IsSymbol,
		}
		if !prop.IsSymbol {
			if d, err := inspectorruntime.GetDescriptor(obj, vm, prop.Name); err == nil && d != nil {
				view.Descriptor = &DescriptorView{
					Writable:     d.Writable,
					Enumerable:   d.Enumerable,
					Configurable: d.Configurable,
					HasGetter:    d.HasGetter,
					HasSetter:    d.HasSetter,
				}
			}
		}
		out = append(out, view)
	}
	return out
}

func prototypeChainView(obj *goja.Object, vm *goja.Runtime) []PrototypeLevelView {
	out := make([]PrototypeLevelView, 0)
	for level, proto := 0, obj.Prototype(); proto != nil && level < defaultPrototypeLevelLimit; level, proto = level+1, proto.Prototype() {
		props := ownPropertiesView(proto, vm)
		if len(props) > defaultPrototypePropLimit {
			props = props[:defaultPrototypePropLimit]
		}
		out = append(out, PrototypeLevelView{
			Name:       prototypeName(proto),
			Properties: props,
		})
	}
	return out
}

func prototypeName(obj *goja.Object) string {
	if obj == nil {
		return "<nil>"
	}
	ctor := obj.Get("constructor")
	if ctor == nil || goja.IsUndefined(ctor) {
		return "<anonymous>"
	}
	ctorObj, ok := ctor.(*goja.Object)
	if !ok {
		return "<anonymous>"
	}
	name := ctorObj.Get("name")
	if name == nil || goja.IsUndefined(name) {
		return "<anonymous>"
	}
	if s := name.String(); s != "" {
		return s
	}
	return "<anonymous>"
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func runtimeValueKind(value goja.Value) string {
	if value == nil || goja.IsUndefined(value) {
		return "undefined"
	}
	if goja.IsNull(value) {
		return "null"
	}
	if _, ok := goja.AssertFunction(value); ok {
		return "function"
	}
	switch value.Export().(type) {
	case string:
		return "string"
	case bool:
		return "boolean"
	case int64, int32, int, float64, float32:
		return "number"
	default:
		if _, ok := value.(*goja.Object); ok {
			return "object"
		}
		return "unknown"
	}
}

func gojaValuePreview(value goja.Value, vm *goja.Runtime) string {
	return inspectorruntime.ValuePreview(value, vm, 120)
}

func executionStatus(err error, helperError bool) string {
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

func firstDiagnosticMessage(diagnostics []DiagnosticView) string {
	if len(diagnostics) == 0 {
		return "parse error"
	}
	return diagnostics[0].Message
}

func dedupeSortedStrings(values []string) []string {
	if len(values) == 0 {
		return nil
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
	return out
}

func (s *sessionState) buildSummary(ctx context.Context) *SessionSummary {
	globals, err := s.snapshotGlobals(ctx)
	if err != nil {
		s.logger.Debug().Err(err).Msg("failed to snapshot globals for summary")
		return s.buildSummaryLocked()
	}
	return s.buildSummaryLockedWithGlobals(globals)
}

func (s *sessionState) buildSummaryLocked() *SessionSummary {
	return s.buildSummaryLockedWithGlobals(nil)
}

func (s *sessionState) buildSummaryLockedWithGlobals(globals map[string]GlobalStateView) *SessionSummary {
	bindings := make([]BindingView, 0, len(s.bindings))
	for _, binding := range s.bindings {
		if binding == nil {
			continue
		}
		bindings = append(bindings, bindingViewFromState(binding))
	}
	sort.Slice(bindings, func(i, j int) bool {
		if bindings[i].Kind != bindings[j].Kind {
			return bindings[i].Kind < bindings[j].Kind
		}
		return bindings[i].Name < bindings[j].Name
	})

	history := make([]HistoryEntry, 0, len(s.cells))
	for _, cell := range s.cells {
		if cell == nil || cell.report == nil {
			continue
		}
		history = append(history, HistoryEntry{
			CellID:        cell.report.ID,
			CreatedAt:     cell.report.CreatedAt,
			SourcePreview: trimForDisplay(strings.ReplaceAll(strings.TrimSpace(cell.report.Source), "\n", " ⏎ "), 100),
			ResultPreview: trimForDisplay(cell.report.Execution.Result, 100),
			Status:        cell.report.Execution.Status,
		})
	}

	summary := &SessionSummary{
		ID:           s.id,
		CreatedAt:    s.createdAt,
		CellCount:    len(s.cells),
		BindingCount: len(bindings),
		Bindings:     bindings,
		History:      history,
		Provenance: []ProvenanceRecord{
			{Section: "session.bindings", Source: "aggregated persistent bindings stored across cells"},
			{Section: "session.history", Source: "evaluation reports recorded after each submitted cell"},
			{Section: "session.globals", Source: "current non-builtin goja global object snapshot"},
		},
	}
	if globals != nil {
		summary.CurrentGlobals = mapGlobalSnapshotViews(globals)
	}
	return summary
}

func (s *sessionState) cellByID(cellID int) *cellState {
	for _, cell := range s.cells {
		if cell != nil && cell.report != nil && cell.report.ID == cellID {
			return cell
		}
	}
	return nil
}

func bindingViewFromState(binding *bindingState) BindingView {
	return BindingView{
		Name:            binding.Name,
		Kind:            binding.Kind.String(),
		Origin:          binding.Origin,
		DeclaredInCell:  binding.DeclaredInCell,
		LastUpdatedCell: binding.LastUpdatedCell,
		DeclaredLine:    binding.DeclaredLine,
		DeclaredSnippet: binding.DeclaredSnippet,
		Static:          binding.Static,
		Runtime:         binding.Runtime,
		Provenance: []ProvenanceRecord{
			{Section: "binding.static", Source: "root-scope binding extraction from the declaring cell"},
			{Section: "binding.runtime", Source: "current runtime value inspection from goja"},
		},
	}
}
