package replsession

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/pkg/jsparse"
	"github.com/pkg/errors"
)

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

// Evaluate runs one cell within an existing session.
func (s *Service) Evaluate(ctx context.Context, sessionID string, source string) (*EvaluateResponse, error) {
	state, err := s.getSession(sessionID)
	if err != nil {
		return nil, err
	}

	state.mu.Lock()
	defer state.mu.Unlock()

	// Short-circuit empty or whitespace-only source to avoid panics in
	// jsparse.Resolve (which assumes program.Body is non-empty).
	if strings.TrimSpace(source) == "" {
		state.nextCellID++
		cellID := state.nextCellID
		now := time.Now().UTC()
		cell := &CellReport{
			ID:        cellID,
			CreatedAt: now,
			Source:    source,
			Rewrite: RewriteReport{
				Mode:              "none",
				TransformedSource: source,
				DeclaredNames:     []string{},
				HelperNames:       []string{},
				Operations:        []RewriteStep{},
				Warnings:          []string{"empty source: nothing to evaluate"},
			},
			Execution: ExecutionReport{
				Status:  "empty-source",
				Result:  "undefined",
				Console: []ConsoleEvent{},
			},
			Runtime: RuntimeReport{
				BeforeGlobals:   []GlobalStateView{},
				AfterGlobals:    []GlobalStateView{},
				Diffs:           []GlobalDiffView{},
				NewBindings:     []string{},
				UpdatedBindings: []string{},
				RemovedBindings: []string{},
				LeakedGlobals:   []string{},
				PersistedByWrap: []string{},
			},
			Provenance: []ProvenanceRecord{},
		}
		state.cells = append(state.cells, &cellState{report: cell})
		return &EvaluateResponse{Session: state.buildSummaryLocked(), Cell: cell}, nil
	}

	policy := state.policy // already normalized at session creation/restore
	state.nextCellID++
	cellID := state.nextCellID
	filename := fmt.Sprintf("<repl-cell-%d>", cellID)

	var (
		analysis     *jsparse.AnalysisResult
		staticReport StaticReport
	)

	// Pre-analyze with the pre-wrapped source for top-level await so that
	// static reports (AST/CST) are available, but keep the analysis for
	// rewrite based on the original source.
	if shouldAnalyze(policy) {
		var cstRoot *jsparse.TSNode
		if policy.Observe.StaticAnalysis {
			if parser, parserErr := jsparse.NewTSParser(); parserErr == nil {
				cstRoot = parser.Parse([]byte(source))
				parser.Close()
			}
		}
		analysis = jsparse.Analyze(filename, source, nil)

		// If the original source failed to parse and looks like a top-level
		// await expression, try again with a pre-wrapped version for static
		// analysis only. The rewrite pipeline will handle await separately.
		if analysis != nil && analysis.ParseErr != nil && policy.UsesInstrumentedExecution() && policy.Eval.SupportTopLevelAwait {
			if wrapped, ok := wrapTopLevelAwaitExpression(source); ok {
				wrappedAnalysis := jsparse.Analyze(filename, wrapped, nil)
				if wrappedAnalysis != nil && wrappedAnalysis.ParseErr == nil {
					staticReport = buildStaticReport(wrappedAnalysis, cstRoot, defaultASTRowLimit, defaultCSTRowLimit)
					// Keep the original analysis (with parse error) so the
					// rewrite pipeline sees it and handles the await case.
				}
			}
		}

		if policy.Observe.StaticAnalysis && len(staticReport.Diagnostics) == 0 {
			staticReport = buildStaticReport(analysis, cstRoot, defaultASTRowLimit, defaultCSTRowLimit)
		}
	}

	rewrite := buildRewriteReport(source, analysis, cellID, policy)

	cell := &CellReport{
		ID:         cellID,
		CreatedAt:  time.Now().UTC(),
		Source:     source,
		Static:     staticReport,
		Rewrite:    rewrite,
		Provenance: provenanceForPolicy(policy),
	}

	// When instrumented mode encounters a parse error that looks like a
	// top-level await expression (which goja's parser rejects outside
	// async functions), skip the normal rewrite pipeline and execute
	// the pre-wrapped source directly.
	isTopLevelAwait := false
	if policy.UsesInstrumentedExecution() && analysis != nil && analysis.ParseErr != nil && policy.Eval.SupportTopLevelAwait {
		trimmed := strings.TrimSpace(source)
		isTopLevelAwait = strings.HasPrefix(trimmed, "await ") || strings.HasPrefix(trimmed, "await(")
	}

	if isTopLevelAwait {
		trimmed := strings.TrimSpace(source)
		helperLast := fmt.Sprintf("__ggg_repl_last_%d__", cellID)
		helperBindings := fmt.Sprintf("__ggg_repl_bindings_%d__", cellID)

		rewrite = RewriteReport{
			Mode:              "async-iife-with-binding-capture",
			DeclaredNames:     []string{},
			HelperNames:       []string{helperLast, helperBindings},
			LastHelperName:    helperLast,
			BindingHelperName: helperBindings,
			CapturedLastExpr:  true,
			TransformedSource: buildAwaitIIFEWithCapture(trimmed, helperLast, helperBindings),
			Operations: []RewriteStep{
				{Kind: "wrap", Detail: "wrap cell source in an async IIFE for top-level await"},
				{Kind: "capture-last-expression", Detail: "capture await expression result"},
			},
			FinalExpressionSrc: trimmed,
		}
		cell.Rewrite = rewrite
		return s.evaluateInstrumented(ctx, state, cell, analysis, rewrite)
	}

	if policy.UsesInstrumentedExecution() && (analysis == nil || analysis.ParseErr != nil) {
		cell.Execution = ExecutionReport{
			Status:     "parse-error",
			Error:      firstDiagnosticMessage(staticReport.Diagnostics),
			DurationMS: 0,
			Console:    []ConsoleEvent{},
		}
		cell.Runtime = RuntimeReport{
			BeforeGlobals:   []GlobalStateView{},
			AfterGlobals:    []GlobalStateView{},
			Diffs:           []GlobalDiffView{},
			NewBindings:     []string{},
			UpdatedBindings: []string{},
			RemovedBindings: []string{},
			LeakedGlobals:   []string{},
			PersistedByWrap: []string{},
		}
		state.cells = append(state.cells, &cellState{report: cell, analysis: analysis})
		if err := s.persistCell(ctx, state, cell); err != nil {
			return nil, err
		}
		return &EvaluateResponse{Session: state.buildSummaryLocked(), Cell: cell}, nil
	}

	if policy.UsesInstrumentedExecution() {
		return s.evaluateInstrumented(ctx, state, cell, analysis, rewrite)
	}
	return s.evaluateRaw(ctx, state, cell, analysis, rewrite, policy)
}

func shouldAnalyze(policy SessionPolicy) bool {
	return policy.UsesInstrumentedExecution() || policy.Observe.StaticAnalysis || policy.Observe.BindingTracking
}

func buildRewriteReport(source string, analysis *jsparse.AnalysisResult, cellID int, policy SessionPolicy) RewriteReport {
	if policy.UsesInstrumentedExecution() {
		return buildRewrite(source, analysis, cellID)
	}

	report := RewriteReport{
		Mode:              "raw",
		TransformedSource: source,
		DeclaredNames:     []string{},
		HelperNames:       []string{},
		Operations:        []RewriteStep{},
		Warnings:          []string{},
	}
	if policy.Eval.SupportTopLevelAwait {
		if wrapped, ok := wrapTopLevelAwaitExpression(source); ok {
			report.TransformedSource = wrapped
			report.Warnings = append(report.Warnings, "wrapped top-level await expression in an async IIFE for raw execution")
		}
		report.Operations = append(report.Operations, RewriteStep{
			Kind:   "raw-execution",
			Detail: "execute source directly without declaration capture; top-level await wrapper may be applied at runtime",
		})
	} else {
		report.Operations = append(report.Operations, RewriteStep{
			Kind:   "raw-execution",
			Detail: "execute source directly without source transformation",
		})
	}
	return report
}

func provenanceForPolicy(policy SessionPolicy) []ProvenanceRecord {
	if policy.UsesInstrumentedExecution() {
		return []ProvenanceRecord{
			{Section: "static", Source: "jsparse.Analyze + resolver + tree-sitter snapshot", Notes: []string{"top-level bindings come from the root lexical scope", "AST rows come from the indexed node tree", "CST rows come from tree-sitter"}},
			{Section: "rewrite", Source: "async IIFE wrapper with explicit binding capture", Notes: []string{"lexical declarations stay cell-local", "declared names are returned and then mirrored back onto the runtime global object"}},
			{Section: "runtime", Source: "goja runtime snapshots before and after evaluation", Notes: []string{"global diffs come from comparing non-builtin global properties", "binding runtime metadata comes from object/property inspection"}},
		}
	}

	records := []ProvenanceRecord{
		{Section: "execution", Source: "direct goja runtime execution without binding-capture rewrite"},
	}
	if policy.Observe.StaticAnalysis {
		records = append(records, ProvenanceRecord{Section: "static", Source: "optional jsparse static analysis performed without changing execution"})
	}
	if policy.Observe.RuntimeSnapshot || policy.Observe.BindingTracking {
		records = append(records, ProvenanceRecord{Section: "runtime", Source: "optional global snapshots collected around raw execution"})
	}
	return records
}

func (s *Service) evaluateInstrumented(ctx context.Context, state *sessionState, cell *CellReport, analysis *jsparse.AnalysisResult, rewrite RewriteReport) (*EvaluateResponse, error) {
	beforeGlobals, err := state.snapshotGlobals(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "snapshot globals before evaluation")
	}
	state.consoleSink = nil

	start := time.Now()
	outcome, execErr := state.executeWrapped(ctx, rewrite)
	duration := time.Since(start)
	consoleEvents := append([]ConsoleEvent{}, state.consoleSink...)
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
		state.upsertRuntimeDiscoveredBinding(name, cell.ID, afterGlobals[name])
		newBindings = append(newBindings, name)
	}
	for _, name := range updated {
		if _, ok := persistedSet[name]; ok {
			updatedBindings = append(updatedBindings, name)
			continue
		}
		if binding := state.bindings[name]; binding != nil {
			binding.LastUpdatedCell = cell.ID
		} else {
			leakedGlobals = append(leakedGlobals, name)
			state.upsertRuntimeDiscoveredBinding(name, cell.ID, afterGlobals[name])
			newBindings = append(newBindings, name)
		}
	}
	for _, name := range outcome.PersistedNames {
		if state.bindings[name] == nil {
			newBindings = append(newBindings, name)
		}
		state.upsertDeclaredBinding(analysis, name, cell.ID)
	}

	// Append the cell before refreshing runtime details so that cellByID
	// can resolve the declaring cell for function source mapping.
	state.cells = append(state.cells, &cellState{report: cell, analysis: analysis})

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

	if err := s.persistCell(ctx, state, cell); err != nil {
		return nil, err
	}
	return &EvaluateResponse{Session: state.buildSummaryLockedWithGlobals(afterGlobals), Cell: cell}, nil
}

func (s *Service) evaluateRaw(ctx context.Context, state *sessionState, cell *CellReport, analysis *jsparse.AnalysisResult, rewrite RewriteReport, policy SessionPolicy) (*EvaluateResponse, error) {
	var (
		beforeGlobals map[string]GlobalStateView
		err           error
	)
	observeRuntime := policy.Observe.RuntimeSnapshot || policy.Observe.BindingTracking
	if observeRuntime {
		beforeGlobals, err = state.snapshotGlobals(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "snapshot globals before evaluation")
		}
	}

	state.consoleSink = nil
	start := time.Now()
	outcome, execErr := state.executeRaw(ctx, rewrite.TransformedSource, policy)
	duration := time.Since(start)
	consoleEvents := append([]ConsoleEvent{}, state.consoleSink...)
	state.consoleSink = nil

	var afterGlobals map[string]GlobalStateView
	if observeRuntime {
		afterGlobals, err = state.snapshotGlobals(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "snapshot globals after evaluation")
		}
	}

	diffs := []GlobalDiffView{}
	newBindings := []string{}
	updatedBindings := []string{}
	removedBindings := []string{}
	if observeRuntime {
		var added []string
		diffs, added, updatedBindings, removedBindings = diffGlobals(beforeGlobals, afterGlobals, state.bindings)
		if policy.Observe.BindingTracking {
			newBindings = append(newBindings, added...)
			for _, name := range removedBindings {
				delete(state.bindings, name)
			}
			for _, name := range added {
				state.upsertRuntimeDiscoveredBinding(name, cell.ID, afterGlobals[name])
			}
			for _, name := range updatedBindings {
				if binding := state.bindings[name]; binding != nil {
					binding.LastUpdatedCell = cell.ID
				} else {
					state.upsertRuntimeDiscoveredBinding(name, cell.ID, afterGlobals[name])
					newBindings = append(newBindings, name)
				}
			}
			if err := state.refreshBindingRuntimeDetails(ctx); err != nil {
				return nil, errors.Wrap(err, "refresh binding runtime details")
			}
		}
	}

	cell.Execution = ExecutionReport{
		Status:     executionStatus(execErr, false),
		Result:     outcome.LastValue,
		Error:      errorString(execErr),
		DurationMS: duration.Milliseconds(),
		Awaited:    outcome.Awaited,
		Console:    consoleEvents,
		HadSideFX:  len(diffs) > 0,
	}
	cell.Runtime = RuntimeReport{
		CurrentCellValue: outcome.LastValue,
		BeforeGlobals:    []GlobalStateView{},
		AfterGlobals:     []GlobalStateView{},
		Diffs:            []GlobalDiffView{},
		NewBindings:      []string{},
		UpdatedBindings:  []string{},
		RemovedBindings:  []string{},
		LeakedGlobals:    []string{},
		PersistedByWrap:  []string{},
	}
	if policy.Observe.RuntimeSnapshot {
		cell.Runtime.BeforeGlobals = mapGlobalSnapshotViews(beforeGlobals)
		cell.Runtime.AfterGlobals = mapGlobalSnapshotViews(afterGlobals)
		cell.Runtime.Diffs = diffs
	}
	if policy.Observe.BindingTracking {
		cell.Runtime.NewBindings = dedupeSortedStrings(newBindings)
		cell.Runtime.UpdatedBindings = dedupeSortedStrings(updatedBindings)
		cell.Runtime.RemovedBindings = dedupeSortedStrings(removedBindings)
	}

	state.cells = append(state.cells, &cellState{report: cell, analysis: analysis})
	if err := s.persistCell(ctx, state, cell); err != nil {
		return nil, err
	}
	if observeRuntime {
		return &EvaluateResponse{Session: state.buildSummaryLockedWithGlobals(afterGlobals), Cell: cell}, nil
	}
	return &EvaluateResponse{Session: state.buildSummaryLocked(), Cell: cell}, nil
}

func wrapTopLevelAwaitExpression(source string) (string, bool) {
	trimmed := strings.TrimSpace(source)
	if strings.HasPrefix(trimmed, "await ") || strings.HasPrefix(trimmed, "await(") {
		return "(async () => { return " + trimmed + "; })()", true
	}
	return source, false
}

// buildAwaitIIFEWithCapture constructs an async IIFE that wraps a top-level
// await expression and captures the result in the helper object format
// expected by persistWrappedReturn.
func buildAwaitIIFEWithCapture(awaitSource string, helperLast string, helperBindings string) string {
	var builder strings.Builder
	builder.WriteString("(async function () {\n")
	builder.WriteString("  let ")
	builder.WriteString(helperLast)
	builder.WriteString(";\n")
	builder.WriteString(helperLast)
	builder.WriteString(" = (")
	builder.WriteString(awaitSource)
	builder.WriteString(");\n")
	builder.WriteString("  return {\n")
	builder.WriteString("    ")
	fmt.Fprintf(&builder, "%q", helperBindings)
	builder.WriteString(": {},\n")
	builder.WriteString("    ")
	fmt.Fprintf(&builder, "%q", helperLast)
	builder.WriteString(": (typeof ")
	builder.WriteString(helperLast)
	builder.WriteString(" === \"undefined\" ? undefined : ")
	builder.WriteString(helperLast)
	builder.WriteString(")\n")
	builder.WriteString("  };\n")
	builder.WriteString("})()")
	return builder.String()
}

func (s *sessionState) executeRaw(ctx context.Context, source string, policy SessionPolicy) (executionOutcome, error) {
	outcome := executionOutcome{}
	execCtx, cancel := evaluationContext(ctx, policy)
	defer cancel()

	value, err := s.runString(execCtx, source)
	if err != nil {
		return outcome, err
	}
	if value == nil {
		outcome.LastValue = "undefined"
		return outcome, nil
	}
	if promise, ok := value.Export().(*goja.Promise); ok {
		if policy.Eval.SupportTopLevelAwait {
			outcome.Awaited = true
			value, err = s.waitPromise(execCtx, promise)
			if err != nil {
				return outcome, err
			}
		} else {
			outcome.LastValue = promisePreview(promise)
			return outcome, nil
		}
	}
	outcome.LastValue = gojaValuePreview(value, s.runtime.VM)
	if outcome.LastValue == "" && (value == nil || goja.IsUndefined(value) || goja.IsNull(value)) {
		outcome.LastValue = "undefined"
	}
	return outcome, nil
}

func (s *sessionState) executeWrapped(ctx context.Context, rewrite RewriteReport) (executionOutcome, error) {
	outcome := executionOutcome{}
	execCtx, cancel := evaluationContext(ctx, s.policy)
	defer cancel()

	value, err := s.runString(execCtx, rewrite.TransformedSource)
	if err != nil {
		return outcome, err
	}
	if value == nil {
		return outcome, nil
	}
	if promise, ok := value.Export().(*goja.Promise); ok {
		outcome.Awaited = true
		value, err = s.waitPromise(execCtx, promise)
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
	if ctx == nil {
		ctx = context.Background()
	}
	if err := evaluationContextError(ctx); err != nil {
		return nil, err
	}

	stopInterrupt := make(chan struct{})
	interrupted := make(chan bool, 1)
	go func() {
		wasInterrupted := false
		select {
		case <-ctx.Done():
			wasInterrupted = true
			cause := evaluationContextError(ctx)
			if cause == nil {
				cause = context.Canceled
			}
			s.runtime.VM.Interrupt(cause)
		case <-stopInterrupt:
		}
		interrupted <- wasInterrupted
	}()

	ret, err := s.runtime.Owner.Call(context.WithoutCancel(ctx), "replsession.run-string", func(_ context.Context, vm *goja.Runtime) (any, error) {
		return vm.RunString(source)
	})
	close(stopInterrupt)
	if <-interrupted {
		s.runtime.VM.ClearInterrupt()
	}
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
		select {
		case <-ctx.Done():
			return nil, evaluationContextError(ctx)
		default:
		}

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
			select {
			case <-ctx.Done():
				return nil, evaluationContextError(ctx)
			case <-time.After(5 * time.Millisecond):
			}
			continue
		case goja.PromiseStateRejected:
			return nil, fmt.Errorf("promise rejected: %s", rejectionMessage(snapshot.Result, s.runtime.VM))
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

func promisePreview(promise *goja.Promise) string {
	switch promise.State() {
	case goja.PromiseStatePending:
		return "Promise { <pending> }"
	case goja.PromiseStateRejected:
		return "Promise { <rejected> }"
	case goja.PromiseStateFulfilled:
		return "Promise { <fulfilled> }"
	default:
		return "Promise { <pending> }"
	}
}

// rejectionMessage extracts a human-readable error message from a promise
// rejection value. When the value is a JS Error object it reads .message
// and .stack; otherwise it falls back to a general preview.
func rejectionMessage(value goja.Value, vm *goja.Runtime) string {
	if value == nil || goja.IsUndefined(value) {
		return "undefined"
	}
	obj, ok := value.(*goja.Object)
	if !ok {
		return gojaValuePreview(value, vm)
	}
	// Check for .message (Error objects)
	msgVal := obj.Get("message")
	if msgVal != nil && !goja.IsUndefined(msgVal) {
		msg := msgVal.String()
		if msg != "" {
			// Also try to get the constructor name for context
			ctor := obj.Get("constructor")
			if ctorObj, ok := ctor.(*goja.Object); ok {
				nameVal := ctorObj.Get("name")
				if nameVal != nil && !goja.IsUndefined(nameVal) {
					name := nameVal.String()
					if name != "" && name != "Error" {
						return name + ": " + msg
					}
				}
			}
			return msg
		}
	}
	// Fallback: try .toString()
	toStringVal := obj.Get("toString")
	if toStringVal != nil && !goja.IsUndefined(toStringVal) {
		if fn, ok := goja.AssertFunction(toStringVal); ok {
			result, err := fn(obj)
			if err == nil && result != nil && !goja.IsUndefined(result) {
				s := result.String()
				if s != "[object Object]" {
					return s
				}
			}
		}
	}
	return gojaValuePreview(value, vm)
}
