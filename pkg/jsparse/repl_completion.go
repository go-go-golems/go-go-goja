package jsparse

import (
	"regexp"
	"sort"
	"strings"

	"github.com/dop251/goja"
)

var (
	requireAliasPattern = regexp.MustCompile(`(?m)\b(?:const|let|var)\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*=\s*require\(\s*['"]([^'"]+)['"]\s*\)`)
	jsIdentifierPattern = regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_$]*$`)

	nodeModuleCandidates = map[string][]CompletionCandidate{
		"fs": {
			{Label: "readFile", Kind: CandidateMethod, Detail: "fs method"},
			{Label: "writeFile", Kind: CandidateMethod, Detail: "fs method"},
			{Label: "existsSync", Kind: CandidateMethod, Detail: "fs method"},
			{Label: "mkdirSync", Kind: CandidateMethod, Detail: "fs method"},
		},
		"path": {
			{Label: "join", Kind: CandidateMethod, Detail: "path method"},
			{Label: "resolve", Kind: CandidateMethod, Detail: "path method"},
			{Label: "dirname", Kind: CandidateMethod, Detail: "path method"},
			{Label: "basename", Kind: CandidateMethod, Detail: "path method"},
			{Label: "extname", Kind: CandidateMethod, Detail: "path method"},
		},
		"url": {
			{Label: "URL", Kind: CandidateFunction, Detail: "constructor"},
			{Label: "URLSearchParams", Kind: CandidateFunction, Detail: "constructor"},
			{Label: "parse", Kind: CandidateMethod, Detail: "url method"},
		},
	}
)

// ExtractRequireAliases finds top-level aliases like: const fs = require("fs")
// and returns alias -> module name.
func ExtractRequireAliases(input string) map[string]string {
	aliases := make(map[string]string)
	matches := requireAliasPattern.FindAllStringSubmatch(input, -1)
	for _, match := range matches {
		if len(match) != 3 {
			continue
		}
		alias := strings.TrimSpace(match[1])
		moduleName := strings.TrimSpace(match[2])
		if alias == "" || moduleName == "" {
			continue
		}
		aliases[alias] = moduleName
	}
	return aliases
}

// NodeModuleCandidates returns known completions for a Node-style module alias.
func NodeModuleCandidates(moduleName string) []CompletionCandidate {
	candidates, ok := nodeModuleCandidates[moduleName]
	if !ok || len(candidates) == 0 {
		return nil
	}
	out := make([]CompletionCandidate, len(candidates))
	copy(out, candidates)
	return out
}

// FilterCandidatesByPrefix keeps candidates whose labels match the typed prefix.
func FilterCandidatesByPrefix(candidates []CompletionCandidate, partial string) []CompletionCandidate {
	if len(candidates) == 0 {
		return nil
	}
	partial = strings.TrimSpace(partial)
	if partial == "" {
		out := make([]CompletionCandidate, len(candidates))
		copy(out, candidates)
		return out
	}
	prefix := strings.ToLower(partial)
	out := make([]CompletionCandidate, 0, len(candidates))
	for _, c := range candidates {
		if strings.HasPrefix(strings.ToLower(c.Label), prefix) {
			out = append(out, c)
		}
	}
	return out
}

// ExtractTopLevelBindingCandidates extracts top-level binding names from source.
// This complements runtime global-object inspection so REPL lexical declarations
// (const/let) remain discoverable across submissions.
func ExtractTopLevelBindingCandidates(source string) []CompletionCandidate {
	analysis := Analyze("repl-eval.js", source, nil)
	if analysis == nil || analysis.Index == nil || analysis.Index.Resolution == nil {
		return nil
	}
	root := analysis.Index.Resolution.Scopes[analysis.Index.Resolution.RootScopeID]
	if root == nil || len(root.Bindings) == 0 {
		return nil
	}

	out := make([]CompletionCandidate, 0, len(root.Bindings))
	for name, binding := range root.Bindings {
		if !isSimpleIdentifier(name) {
			continue
		}

		var kind CandidateKind
		switch binding.Kind {
		case BindingVar, BindingLet, BindingConst, BindingParameter, BindingCatchParam:
			kind = CandidateVariable
		case BindingFunction:
			kind = CandidateFunction
		case BindingClass:
			kind = CandidateFunction
		default:
			kind = CandidateVariable
		}

		out = append(out, CompletionCandidate{
			Label:  name,
			Kind:   kind,
			Detail: binding.Kind.String(),
		})
	}

	sort.SliceStable(out, func(i, j int) bool {
		li := strings.ToLower(out[i].Label)
		lj := strings.ToLower(out[j].Label)
		if li == lj {
			return out[i].Label < out[j].Label
		}
		return li < lj
	})

	return out
}

// AugmentREPLCandidates merges parser-derived candidates with module and runtime
// candidates using deterministic ranking and dedupe.
//
// Ranking:
//  1. parser/static candidates
//  2. module-alias candidates
//  3. runtime candidates
func AugmentREPLCandidates(
	runtime *goja.Runtime,
	input string,
	ctx CompletionContext,
	staticCandidates []CompletionCandidate,
	runtimeIdentifierHints []CompletionCandidate,
) []CompletionCandidate {
	moduleCandidates := []CompletionCandidate(nil)
	if ctx.Kind == CompletionProperty {
		aliases := ExtractRequireAliases(input)
		if moduleName, ok := aliases[strings.TrimSpace(ctx.BaseExpr)]; ok {
			moduleCandidates = FilterCandidatesByPrefix(NodeModuleCandidates(moduleName), ctx.PartialText)
		}
	}

	runtimeCandidates := []CompletionCandidate(nil)
	switch ctx.Kind {
	case CompletionIdentifier:
		runtimeCandidates = runtimeIdentifierCandidates(runtime, ctx.PartialText, runtimeIdentifierHints)
	case CompletionProperty:
		runtimeCandidates = runtimePropertyCandidates(runtime, ctx.BaseExpr, ctx.PartialText)
	case CompletionArgument, CompletionNone:
		runtimeCandidates = nil
	default:
		runtimeCandidates = nil
	}

	return mergeRankedCompletionCandidates(staticCandidates, moduleCandidates, runtimeCandidates)
}

// DedupeAndSortCandidates removes empty/duplicate labels and returns a
// case-insensitive sorted list.
func DedupeAndSortCandidates(candidates []CompletionCandidate) []CompletionCandidate {
	if len(candidates) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(candidates))
	out := make([]CompletionCandidate, 0, len(candidates))
	for _, c := range candidates {
		if strings.TrimSpace(c.Label) == "" {
			continue
		}
		if _, ok := seen[c.Label]; ok {
			continue
		}
		seen[c.Label] = struct{}{}
		out = append(out, c)
	}

	sort.SliceStable(out, func(i, j int) bool {
		li := strings.ToLower(out[i].Label)
		lj := strings.ToLower(out[j].Label)
		if li == lj {
			return out[i].Label < out[j].Label
		}
		return li < lj
	})
	return out
}

// FindExactCandidate returns the first exact (case-insensitive) candidate label
// match for the provided partial token.
func FindExactCandidate(candidates []CompletionCandidate, partial string) *CompletionCandidate {
	partial = strings.TrimSpace(partial)
	if partial == "" {
		return nil
	}
	for i := range candidates {
		if strings.EqualFold(candidates[i].Label, partial) {
			return &candidates[i]
		}
	}
	return nil
}

func runtimeIdentifierCandidates(runtime *goja.Runtime, partial string, hints []CompletionCandidate) []CompletionCandidate {
	if runtime == nil {
		return nil
	}

	prefix := strings.ToLower(strings.TrimSpace(partial))
	merged := make(map[string]CompletionCandidate)

	for _, key := range safeObjectKeys(runtime.GlobalObject()) {
		if !isSimpleIdentifier(key) {
			continue
		}
		if prefix != "" && !strings.HasPrefix(strings.ToLower(key), prefix) {
			continue
		}
		if candidate, ok := runtimeIdentifierFromName(runtime, key); ok {
			merged[key] = candidate
		}
	}

	for _, hint := range hints {
		name := strings.TrimSpace(hint.Label)
		if !isSimpleIdentifier(name) {
			continue
		}
		if prefix != "" && !strings.HasPrefix(strings.ToLower(name), prefix) {
			continue
		}
		if _, exists := merged[name]; exists {
			continue
		}
		if candidate, ok := runtimeIdentifierFromName(runtime, name); ok {
			if strings.TrimSpace(candidate.Detail) == "" {
				candidate.Detail = "runtime symbol"
			}
			merged[name] = candidate
		}
	}

	return mapToSortedCandidates(merged)
}

func runtimePropertyCandidates(runtime *goja.Runtime, baseExpr, partial string) []CompletionCandidate {
	if runtime == nil {
		return nil
	}

	baseExpr = strings.TrimSpace(baseExpr)
	if !isSimpleIdentifier(baseExpr) {
		return nil
	}

	v := runtime.Get(baseExpr)
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return nil
	}

	obj, ok := safeToObject(runtime, v)
	if !ok {
		return nil
	}

	prefix := strings.ToLower(strings.TrimSpace(partial))
	seen := make(map[string]CompletionCandidate)
	for _, key := range safeObjectKeys(obj) {
		if strings.TrimSpace(key) == "" {
			continue
		}
		if prefix != "" && !strings.HasPrefix(strings.ToLower(key), prefix) {
			continue
		}
		seen[key] = CompletionCandidate{
			Label:  key,
			Kind:   CandidateProperty,
			Detail: "runtime property",
		}
	}

	return mapToSortedCandidates(seen)
}

func runtimeIdentifierFromName(runtime *goja.Runtime, name string) (CompletionCandidate, bool) {
	v := runtime.Get(name)
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return CompletionCandidate{}, false
	}
	if _, ok := goja.AssertFunction(v); ok {
		return CompletionCandidate{
			Label:  name,
			Kind:   CandidateFunction,
			Detail: "runtime function",
		}, true
	}
	return CompletionCandidate{
		Label:  name,
		Kind:   CandidateVariable,
		Detail: "runtime global",
	}, true
}

type rankedCandidate struct {
	candidate CompletionCandidate
	priority  int
}

func mergeRankedCompletionCandidates(
	staticCandidates []CompletionCandidate,
	moduleCandidates []CompletionCandidate,
	runtimeCandidates []CompletionCandidate,
) []CompletionCandidate {
	merged := make(map[string]rankedCandidate)
	addGroup := func(candidates []CompletionCandidate, priority int) {
		for _, c := range candidates {
			label := strings.TrimSpace(c.Label)
			if label == "" {
				continue
			}
			existing, ok := merged[label]
			if !ok || priority < existing.priority {
				merged[label] = rankedCandidate{candidate: c, priority: priority}
			}
		}
	}

	addGroup(staticCandidates, 0)
	addGroup(moduleCandidates, 1)
	addGroup(runtimeCandidates, 2)

	ranked := make([]rankedCandidate, 0, len(merged))
	for _, c := range merged {
		ranked = append(ranked, c)
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].priority != ranked[j].priority {
			return ranked[i].priority < ranked[j].priority
		}
		li := strings.ToLower(ranked[i].candidate.Label)
		lj := strings.ToLower(ranked[j].candidate.Label)
		if li == lj {
			return ranked[i].candidate.Label < ranked[j].candidate.Label
		}
		return li < lj
	})

	out := make([]CompletionCandidate, 0, len(ranked))
	for _, c := range ranked {
		out = append(out, c.candidate)
	}
	return out
}

func mapToSortedCandidates(in map[string]CompletionCandidate) []CompletionCandidate {
	if len(in) == 0 {
		return nil
	}
	out := make([]CompletionCandidate, 0, len(in))
	for _, c := range in {
		out = append(out, c)
	}
	sort.SliceStable(out, func(i, j int) bool {
		li := strings.ToLower(out[i].Label)
		lj := strings.ToLower(out[j].Label)
		if li == lj {
			return out[i].Label < out[j].Label
		}
		return li < lj
	})
	return out
}

func safeToObject(runtime *goja.Runtime, value goja.Value) (*goja.Object, bool) {
	ok := true
	var obj *goja.Object
	defer func() {
		if recover() != nil {
			ok = false
		}
	}()
	obj = value.ToObject(runtime)
	return obj, ok
}

func safeObjectKeys(obj *goja.Object) []string {
	var keys []string
	defer func() {
		if recover() != nil {
			keys = nil
		}
	}()
	if obj == nil {
		return nil
	}
	keys = obj.Keys()
	return keys
}

func isSimpleIdentifier(name string) bool {
	return jsIdentifierPattern.MatchString(name)
}
