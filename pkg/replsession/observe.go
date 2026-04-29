package replsession

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/dop251/goja"
	inspectoranalysis "github.com/go-go-golems/go-go-goja/pkg/inspector/analysis"
	inspectorcore "github.com/go-go-golems/go-go-goja/pkg/inspector/core"
	inspectorruntime "github.com/go-go-golems/go-go-goja/pkg/inspector/runtime"
	"github.com/go-go-golems/go-go-goja/pkg/jsparse"
)

type persistResult struct {
	Persisted     []string
	LastValue     string
	LastValueJSON string
	HelperError   bool
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
			if binding != nil {
				if binding.Kind == jsparse.BindingFunction {
					// Build FunctionMapping from static analysis data since the
					// runtime function was compiled from the IIFE wrapper, not
					// the original source, so MapFunctionToSource offsets don't
					// match. Use the declared line and parameters we already
					// captured during static analysis.
					if c := s.cellByID(binding.DeclaredInCell); c != nil && c.analysis != nil {
						view.FunctionMapping = staticFunctionMapping(binding, c.analysis)
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

func (s *sessionState) buildSummary(ctx context.Context) *SessionSummary {
	if !s.policy.Observe.RuntimeSnapshot && !s.policy.Observe.BindingTracking {
		return s.buildSummaryLocked()
	}
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
		Profile:      s.profile,
		Policy:       s.policy, // already normalized at session creation/restore
		CreatedAt:    s.createdAt,
		CellCount:    len(s.cells),
		BindingCount: len(bindings),
		Bindings:     bindings,
		History:      history,
		Provenance:   provenanceForSummary(),
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

// staticFunctionMapping builds a FunctionMappingView from static analysis
// data rather than trying to match the runtime function's bytecode offsets,
// which don't align because the runtime compiled from the IIFE wrapper.
func staticFunctionMapping(binding *bindingState, analysis *jsparse.AnalysisResult) *FunctionMappingView {
	if binding == nil {
		return nil
	}
	// Try to find the function node in the AST for position info
	startLine := binding.DeclaredLine
	endLine := 0
	nodeID := 0
	if analysis != nil && analysis.Resolution != nil && analysis.Resolution.RootScopeID >= 0 {
		root := analysis.Resolution.Scopes[analysis.Resolution.RootScopeID]
		if root != nil {
			if b, ok := root.Bindings[binding.Name]; ok && b != nil && b.DeclNodeID >= 0 {
				nodeID = int(b.DeclNodeID)
				if analysis.Index != nil && analysis.Index.Nodes[b.DeclNodeID] != nil {
					node := analysis.Index.Nodes[b.DeclNodeID]
					endLine = node.EndLine
				}
			}
		}
	}
	return &FunctionMappingView{
		Name:      binding.Name,
		StartLine: startLine,
		EndLine:   endLine,
		NodeID:    nodeID,
	}
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
		Provenance:      provenanceForBinding(),
	}
}

func provenanceForSummary() []ProvenanceRecord {
	return []ProvenanceRecord{
		{Section: "session.bindings", Source: "aggregated persistent bindings stored across cells"},
		{Section: "session.history", Source: "evaluation reports recorded after each submitted cell"},
		{Section: "session.globals", Source: "current non-builtin goja global object snapshot"},
	}
}

func provenanceForBinding() []ProvenanceRecord {
	return []ProvenanceRecord{
		{Section: "binding.static", Source: "root-scope binding extraction from the declaring cell"},
		{Section: "binding.runtime", Source: "current runtime value inspection from goja"},
	}
}
