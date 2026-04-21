package javascript

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/dop251/goja"
	"github.com/go-go-golems/bobatea/pkg/autocomplete"
	"github.com/go-go-golems/bobatea/pkg/repl"
	"github.com/go-go-golems/go-go-goja/pkg/docaccess"
	"github.com/go-go-golems/go-go-goja/pkg/jsparse"
)

// AssistanceConfig wires parser, runtime, and binding metadata access into the
// shared JavaScript completion/help implementation.
type AssistanceConfig struct {
	TSParser     *jsparse.TSParser
	TSMu         *sync.Mutex
	WithRuntime  func(context.Context, func(*goja.Runtime, *docaccess.Hub) error) error
	BindingHints func(context.Context) ([]jsparse.CompletionCandidate, error)
}

// Assistance provides completion/help services that can be shared by the
// classic evaluator and the replapi-backed TUI adapter.
type Assistance struct {
	tsParser     *jsparse.TSParser
	tsMu         *sync.Mutex
	withRuntime  func(context.Context, func(*goja.Runtime, *docaccess.Hub) error) error
	bindingHints func(context.Context) ([]jsparse.CompletionCandidate, error)
}

// NewAssistance constructs the shared JS completion/help provider.
func NewAssistance(config AssistanceConfig) *Assistance {
	return &Assistance{
		tsParser:     config.TSParser,
		tsMu:         config.TSMu,
		withRuntime:  config.WithRuntime,
		bindingHints: config.BindingHints,
	}
}

func (a *Assistance) CompleteInput(ctx context.Context, req repl.CompletionRequest) (repl.CompletionResult, error) {
	if a == nil || a.tsParser == nil {
		return repl.CompletionResult{Show: false}, nil
	}

	input := req.Input
	cursor := clampCursor(req.CursorByte, len(input))
	if strings.TrimSpace(input) == "" {
		return repl.CompletionResult{
			Show:        false,
			ReplaceFrom: cursor,
			ReplaceTo:   cursor,
		}, nil
	}

	root := a.parse(input)
	if root == nil {
		return repl.CompletionResult{Show: false}, nil
	}

	analysis := jsparse.Analyze("repl-input.js", input, nil)
	row, col := byteOffsetToRowCol(input, cursor)
	completionCtx := analysis.CompletionContextAt(root, row, col)
	if completionCtx.Kind == jsparse.CompletionNone {
		return repl.CompletionResult{Show: false}, nil
	}

	aliases := jsparse.ExtractRequireAliases(input)
	candidates := jsparse.ResolveCandidates(completionCtx, analysis.Index, root)
	bindingHints, _ := a.loadBindingHints(ctx)
	if err := a.withRuntimeState(ctx, func(vm *goja.Runtime, docs *docsResolver) error {
		candidates = jsparse.AugmentREPLCandidates(
			vm,
			input,
			completionCtx,
			candidates,
			bindingHints,
		)
		candidates = append(candidates, docCompletionCandidates(docs, completionCtx, aliases)...)
		return nil
	}); err != nil {
		return repl.CompletionResult{}, err
	}

	if len(candidates) == 0 {
		return repl.CompletionResult{
			Show:        false,
			ReplaceFrom: cursor,
			ReplaceTo:   cursor,
		}, nil
	}

	replaceFrom := clampCursor(cursor-len(completionCtx.PartialText), len(input))
	replaceTo := cursor
	if replaceFrom > replaceTo {
		replaceFrom = replaceTo
	}

	suggestions := make([]autocomplete.Suggestion, 0, len(candidates))
	seen := make(map[string]struct{}, len(candidates))
	if err := a.withRuntimeState(ctx, func(_ *goja.Runtime, docs *docsResolver) error {
		for _, candidate := range candidates {
			if candidate.Label == "" {
				continue
			}
			if _, ok := seen[candidate.Label]; ok {
				continue
			}
			seen[candidate.Label] = struct{}{}

			display := candidate.Label
			icon := candidate.Kind.Icon()
			if icon != "" {
				display = icon + " " + display
			}
			if candidate.Detail != "" {
				display += " - " + candidate.Detail
			}
			if entry, ok := resolveDocEntryForCandidate(docs, completionCtx, candidate, aliases); ok {
				if summary := strings.TrimSpace(entry.Summary); summary != "" && !strings.Contains(display, summary) {
					display += " - " + summary
				}
			}

			suggestions = append(suggestions, autocomplete.Suggestion{
				Id:          candidate.Label,
				Value:       candidate.Label,
				DisplayText: display,
			})
		}
		return nil
	}); err != nil {
		return repl.CompletionResult{}, err
	}

	show := len(suggestions) > 0
	if req.Reason == repl.CompletionReasonDebounce {
		if completionCtx.Kind == jsparse.CompletionIdentifier && len(completionCtx.PartialText) == 0 {
			show = false
		}
	}

	return repl.CompletionResult{
		Show:        show,
		Suggestions: suggestions,
		ReplaceFrom: replaceFrom,
		ReplaceTo:   replaceTo,
	}, nil
}

func (a *Assistance) GetHelpBar(ctx context.Context, req repl.HelpBarRequest) (repl.HelpBarPayload, error) {
	if a == nil || a.tsParser == nil {
		return repl.HelpBarPayload{Show: false}, nil
	}

	input := req.Input
	if strings.TrimSpace(input) == "" {
		return repl.HelpBarPayload{Show: false}, nil
	}
	cursor := clampCursor(req.CursorByte, len(input))

	root := a.parse(input)
	if root == nil {
		return repl.HelpBarPayload{Show: false}, nil
	}

	analysis := jsparse.Analyze("repl-input.js", input, nil)
	row, col := byteOffsetToRowCol(input, cursor)
	completionCtx := analysis.CompletionContextAt(root, row, col)

	token, _, _ := tokenAtCursor(input, cursor)
	token = strings.TrimSpace(token)
	if token == "" && completionCtx.Kind == jsparse.CompletionNone {
		return repl.HelpBarPayload{Show: false}, nil
	}
	if req.Reason == repl.HelpBarReasonDebounce && completionCtx.Kind == jsparse.CompletionIdentifier && len(strings.TrimSpace(completionCtx.PartialText)) < 2 {
		return repl.HelpBarPayload{Show: false}, nil
	}

	aliases := jsparse.ExtractRequireAliases(input)
	candidates := []jsparse.CompletionCandidate{}
	if completionCtx.Kind != jsparse.CompletionNone {
		candidates = jsparse.ResolveCandidates(completionCtx, analysis.Index, root)
		if completionCtx.Kind == jsparse.CompletionProperty {
			if moduleName, ok := aliases[completionCtx.BaseExpr]; ok {
				candidates = append(candidates, jsparse.FilterCandidatesByPrefix(jsparse.NodeModuleCandidates(moduleName), completionCtx.PartialText)...)
			}
		}
	}

	bindingHints, _ := a.loadBindingHints(ctx)
	var payload repl.HelpBarPayload
	err := a.withRuntimeState(ctx, func(vm *goja.Runtime, docs *docsResolver) error {
		candidates = jsparse.AugmentREPLCandidates(vm, input, completionCtx, candidates, bindingHints)
		candidates = append(candidates, docCompletionCandidates(docs, completionCtx, aliases)...)
		candidates = jsparse.DedupeAndSortCandidates(candidates)

		if p, ok := helpBarFromContext(vm, docs, completionCtx, candidates, aliases); ok {
			payload = p
			return nil
		}

		payload = helpBarFromTokenFallback(vm, docs, token, aliases)
		return nil
	})
	if err != nil {
		return repl.HelpBarPayload{}, err
	}
	return payload, nil
}

func (a *Assistance) GetHelpDrawer(ctx context.Context, req repl.HelpDrawerRequest) (repl.HelpDrawerDocument, error) {
	select {
	case <-ctx.Done():
		return repl.HelpDrawerDocument{}, ctx.Err()
	default:
	}

	doc := repl.HelpDrawerDocument{
		Show:       true,
		Title:      "JavaScript Context",
		Subtitle:   describeHelpDrawerTrigger(req.Trigger),
		Markdown:   "Start typing JavaScript to inspect contextual symbol help.",
		VersionTag: fmt.Sprintf("request-%d", req.RequestID),
	}
	if strings.TrimSpace(req.Input) == "" {
		return doc, nil
	}

	if a == nil || a.tsParser == nil {
		doc.Diagnostics = []string{"jsparse parser not available"}
		return doc, nil
	}

	input := req.Input
	cursor := clampCursor(req.CursorByte, len(input))

	root := a.parse(input)
	if root == nil {
		doc.Diagnostics = []string{"failed to parse input"}
		return doc, nil
	}

	analysis := jsparse.Analyze("repl-input.js", input, nil)
	row, col := byteOffsetToRowCol(input, cursor)
	completionCtx := analysis.CompletionContextAt(root, row, col)

	token, _, _ := tokenAtCursor(input, cursor)
	token = strings.TrimSpace(token)
	aliases := jsparse.ExtractRequireAliases(input)
	bindingHints, _ := a.loadBindingHints(ctx)

	err := a.withRuntimeState(ctx, func(vm *goja.Runtime, docs *docsResolver) error {
		candidates := []jsparse.CompletionCandidate{}
		if completionCtx.Kind != jsparse.CompletionNone {
			candidates = jsparse.ResolveCandidates(completionCtx, analysis.Index, root)
		}
		candidates = jsparse.AugmentREPLCandidates(vm, input, completionCtx, candidates, bindingHints)
		candidates = append(candidates, docCompletionCandidates(docs, completionCtx, aliases)...)
		candidates = jsparse.DedupeAndSortCandidates(candidates)

		entry, entryOK := resolveDocEntryFromContext(docs, completionCtx, candidates, aliases)
		payload, ok := helpBarFromContext(vm, docs, completionCtx, candidates, aliases)
		if !ok {
			if !entryOK {
				entry, entryOK = resolveDocEntryFromToken(docs, token, aliases)
			}
			payload = helpBarFromTokenFallback(vm, docs, token, aliases)
		}

		if payload.Show {
			doc.Title = payload.Text
		}
		if entryOK {
			doc.Title = entry.Title
		}

		doc.Subtitle = fmt.Sprintf(
			"%s | kind: %s | cursor: %d",
			describeHelpDrawerTrigger(req.Trigger),
			describeCompletionKind(completionCtx.Kind),
			cursor,
		)
		if entryOK {
			doc.Subtitle = fmt.Sprintf(
				"%s | %s | source: %s | cursor: %d",
				describeHelpDrawerTrigger(req.Trigger),
				entry.KindLabel,
				entry.Ref.SourceID,
				cursor,
			)
		}

		var md strings.Builder
		if strings.TrimSpace(input) == "" {
			md.WriteString("Type a symbol such as `console.lo` or `Math.ma` to inspect context.\n")
		} else {
			md.WriteString("```javascript\n")
			md.WriteString(clipForDrawer(input, 320))
			md.WriteString("\n```\n")
		}

		if payload.Show {
			md.WriteString("\n### Symbol\n")
			md.WriteString("- ")
			md.WriteString(payload.Text)
			md.WriteString("\n")
		}
		if entryOK {
			md.WriteString("\n### Documentation\n")
			if summary := strings.TrimSpace(entry.Summary); summary != "" {
				md.WriteString(summary)
				md.WriteString("\n")
			}
			if body := strings.TrimSpace(entry.Body); body != "" && body != strings.TrimSpace(entry.Summary) {
				md.WriteString("\n")
				md.WriteString(body)
				md.WriteString("\n")
			}
			md.WriteString("\n### Doc Metadata\n")
			md.WriteString("- Source: `")
			md.WriteString(entry.Ref.SourceID)
			md.WriteString("`\n")
			md.WriteString("- Kind: ")
			md.WriteString(entry.KindLabel)
			md.WriteString("\n")
			if entry.Path != "" {
				md.WriteString("- Path: `")
				md.WriteString(entry.Path)
				md.WriteString("`\n")
			}
			if len(entry.Tags) > 0 {
				md.WriteString("- Tags: `")
				md.WriteString(strings.Join(entry.Tags, "`, `"))
				md.WriteString("`\n")
			}
			if len(entry.Related) > 0 {
				md.WriteString("\n### Related Docs\n")
				limit := min(6, len(entry.Related))
				for i := 0; i < limit; i++ {
					ref := entry.Related[i]
					md.WriteString("- `")
					md.WriteString(ref.ID)
					md.WriteString("`")
					if ref.Kind != "" {
						md.WriteString(" (")
						md.WriteString(ref.Kind)
						md.WriteString(")")
					}
					md.WriteString("\n")
				}
				if len(entry.Related) > limit {
					md.WriteString("- ...")
					fmt.Fprintf(&md, " %d more", len(entry.Related)-limit)
					md.WriteString("\n")
				}
			}
		}

		if completionCtx.Kind == jsparse.CompletionProperty {
			base := strings.TrimSpace(completionCtx.BaseExpr)
			if base != "" {
				md.WriteString("\n### Property Context\n")
				md.WriteString("- Base expression: `")
				md.WriteString(base)
				md.WriteString("`\n")
				if partial := strings.TrimSpace(completionCtx.PartialText); partial != "" {
					md.WriteString("- Typed prefix: `")
					md.WriteString(partial)
					md.WriteString("`\n")
				}
			}
		}

		if len(candidates) > 0 {
			md.WriteString("\n### Completion Candidates\n")
			limit := min(8, len(candidates))
			for i := 0; i < limit; i++ {
				candidate := candidates[i]
				md.WriteString("- `")
				md.WriteString(candidate.Label)
				md.WriteString("`")
				if detail := normalizeCandidateDetail(candidate.Detail); detail != "symbol" {
					md.WriteString(" - ")
					md.WriteString(detail)
				}
				md.WriteString("\n")
			}
			if len(candidates) > limit {
				md.WriteString("- ...")
				fmt.Fprintf(&md, " %d more", len(candidates)-limit)
				md.WriteString("\n")
			}
		}

		if len(aliases) > 0 {
			md.WriteString("\n### require() Aliases\n")
			keys := make([]string, 0, len(aliases))
			for alias := range aliases {
				keys = append(keys, alias)
			}
			sort.Strings(keys)
			for _, alias := range keys {
				md.WriteString("- `")
				md.WriteString(alias)
				md.WriteString("` -> `")
				md.WriteString(aliases[alias])
				md.WriteString("`\n")
			}
		}

		doc.Markdown = strings.TrimSpace(md.String())
		return nil
	})
	if err != nil {
		return repl.HelpDrawerDocument{}, err
	}

	return doc, nil
}

func (a *Assistance) parse(input string) *jsparse.TSNode {
	if a == nil || a.tsParser == nil {
		return nil
	}
	if a.tsMu != nil {
		a.tsMu.Lock()
		defer a.tsMu.Unlock()
	}
	return a.tsParser.Parse([]byte(input))
}

func (a *Assistance) withRuntimeState(ctx context.Context, fn func(*goja.Runtime, *docsResolver) error) error {
	if a == nil || a.withRuntime == nil {
		return fn(nil, nil)
	}
	return a.withRuntime(ctx, func(vm *goja.Runtime, hub *docaccess.Hub) error {
		return fn(vm, newDocsResolverFromHub(hub))
	})
}

func (a *Assistance) loadBindingHints(ctx context.Context) ([]jsparse.CompletionCandidate, error) {
	if a == nil || a.bindingHints == nil {
		return nil, nil
	}
	return a.bindingHints(ctx)
}

func helpBarFromContext(
	vm *goja.Runtime,
	docs *docsResolver,
	ctx jsparse.CompletionContext,
	candidates []jsparse.CompletionCandidate,
	aliases map[string]string,
) (repl.HelpBarPayload, bool) {
	if entry, ok := resolveDocEntryFromContext(docs, ctx, candidates, aliases); ok {
		return makeHelpBarPayload(docEntrySummary(entry), "docs"), true
	}

	switch ctx.Kind {
	case jsparse.CompletionProperty:
		base := strings.TrimSpace(ctx.BaseExpr)
		if base == "" {
			return repl.HelpBarPayload{}, false
		}
		if strings.TrimSpace(ctx.PartialText) == "" {
			if txt, ok := helpBarSignatureFor(base, "", aliases); ok {
				return makeHelpBarPayload(txt, "signature"), true
			}
		}
		exact := jsparse.FindExactCandidate(candidates, ctx.PartialText)
		if exact != nil {
			if txt, ok := helpBarSignatureFor(base, exact.Label, aliases); ok {
				return makeHelpBarPayload(txt, "signature"), true
			}
			return makeHelpBarPayload(fmt.Sprintf("%s.%s - %s", base, exact.Label, normalizeCandidateDetail(exact.Detail)), "info"), true
		}
		if len(candidates) > 0 {
			c := candidates[0]
			if txt, ok := helpBarSignatureFor(base, c.Label, aliases); ok {
				return makeHelpBarPayload(txt, "signature"), true
			}
			return makeHelpBarPayload(fmt.Sprintf("%s.%s - %s", base, c.Label, normalizeCandidateDetail(c.Detail)), "info"), true
		}
	case jsparse.CompletionIdentifier:
		exact := jsparse.FindExactCandidate(candidates, ctx.PartialText)
		if exact != nil {
			if txt, ok := helpBarSignatureFor(exact.Label, "", nil); ok {
				return makeHelpBarPayload(txt, "signature"), true
			}
			if txt, ok := runtimeHelpForIdentifier(vm, exact.Label); ok {
				return makeHelpBarPayload(txt, "runtime"), true
			}
			return makeHelpBarPayload(fmt.Sprintf("%s - %s", exact.Label, normalizeCandidateDetail(exact.Detail)), "info"), true
		}
		if len(candidates) > 0 {
			c := candidates[0]
			if txt, ok := helpBarSignatureFor(c.Label, "", nil); ok {
				return makeHelpBarPayload(txt, "signature"), true
			}
			if txt, ok := runtimeHelpForIdentifier(vm, c.Label); ok {
				return makeHelpBarPayload(txt, "runtime"), true
			}
			return makeHelpBarPayload(fmt.Sprintf("%s - %s", c.Label, normalizeCandidateDetail(c.Detail)), "info"), true
		}
	case jsparse.CompletionNone, jsparse.CompletionArgument:
		return repl.HelpBarPayload{}, false
	}

	return repl.HelpBarPayload{}, false
}

func helpBarFromTokenFallback(vm *goja.Runtime, docs *docsResolver, token string, aliases map[string]string) repl.HelpBarPayload {
	if token == "" {
		return repl.HelpBarPayload{Show: false}
	}
	token = strings.Trim(token, ".")
	if token == "" {
		return repl.HelpBarPayload{Show: false}
	}
	if entry, ok := resolveDocEntryFromToken(docs, token, aliases); ok {
		return makeHelpBarPayload(docEntrySummary(entry), "docs")
	}
	if txt, ok := helpBarSymbolSignatures[token]; ok {
		return makeHelpBarPayload(txt, "signature")
	}
	if txt, ok := runtimeHelpForIdentifier(vm, token); ok {
		return makeHelpBarPayload(txt, "runtime")
	}
	return repl.HelpBarPayload{Show: false}
}

func helpBarSignatureFor(base, property string, aliases map[string]string) (string, bool) {
	candidates := make([]string, 0, 4)

	if property == "" {
		if aliases != nil {
			if moduleName, ok := aliases[base]; ok {
				candidates = append(candidates, moduleName)
			}
		}
		candidates = append(candidates, base)
	} else {
		if aliases != nil {
			if moduleName, ok := aliases[base]; ok {
				candidates = append(candidates, moduleName+"."+property)
			}
		}
		candidates = append(candidates, base+"."+property)
	}

	for _, key := range candidates {
		if txt, ok := helpBarSymbolSignatures[key]; ok {
			return txt, true
		}
	}
	return "", false
}

func runtimeHelpForIdentifier(vm *goja.Runtime, name string) (string, bool) {
	if vm == nil || name == "" || strings.Contains(name, ".") {
		return "", false
	}
	v := vm.Get(name)
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return "", false
	}
	if _, ok := goja.AssertFunction(v); ok {
		obj := v.ToObject(vm)
		displayName := name
		if n := obj.Get("name"); n != nil && !goja.IsUndefined(n) {
			if s := strings.TrimSpace(n.String()); s != "" {
				displayName = s
			}
		}
		if l := obj.Get("length"); l != nil && !goja.IsUndefined(l) {
			switch vv := l.Export().(type) {
			case int64:
				return fmt.Sprintf("%s(...): function (arity %d)", displayName, vv), true
			case int32:
				return fmt.Sprintf("%s(...): function (arity %d)", displayName, vv), true
			case int:
				return fmt.Sprintf("%s(...): function (arity %d)", displayName, vv), true
			case float64:
				return fmt.Sprintf("%s(...): function (arity %d)", displayName, int64(vv)), true
			}
		}
		return fmt.Sprintf("%s(...): function", displayName), true
	}
	obj := v.ToObject(vm)
	className := strings.ToLower(strings.TrimSpace(obj.ClassName()))
	if className == "" {
		className = "value"
	}
	return fmt.Sprintf("%s: %s", name, className), true
}

func resolveDocEntryFromContext(
	docs *docsResolver,
	ctx jsparse.CompletionContext,
	candidates []jsparse.CompletionCandidate,
	aliases map[string]string,
) (*docaccess.Entry, bool) {
	if docs == nil {
		return nil, false
	}
	switch ctx.Kind {
	case jsparse.CompletionProperty:
		base := strings.TrimSpace(ctx.BaseExpr)
		if base == "" {
			return nil, false
		}
		if strings.TrimSpace(ctx.PartialText) == "" {
			return docs.ResolveProperty(base, "", aliases)
		}
		if exact := jsparse.FindExactCandidate(candidates, ctx.PartialText); exact != nil {
			return docs.ResolveProperty(base, exact.Label, aliases)
		}
		if len(candidates) > 0 {
			return docs.ResolveProperty(base, candidates[0].Label, aliases)
		}
	case jsparse.CompletionIdentifier:
		if exact := jsparse.FindExactCandidate(candidates, ctx.PartialText); exact != nil {
			return docs.ResolveIdentifier(exact.Label, aliases)
		}
		if len(candidates) > 0 {
			return docs.ResolveIdentifier(candidates[0].Label, aliases)
		}
	case jsparse.CompletionNone, jsparse.CompletionArgument:
		return nil, false
	}
	return nil, false
}

func resolveDocEntryFromToken(docs *docsResolver, token string, aliases map[string]string) (*docaccess.Entry, bool) {
	if docs == nil {
		return nil, false
	}
	return docs.ResolveToken(token, aliases)
}

func resolveDocEntryForCandidate(
	docs *docsResolver,
	ctx jsparse.CompletionContext,
	candidate jsparse.CompletionCandidate,
	aliases map[string]string,
) (*docaccess.Entry, bool) {
	if docs == nil {
		return nil, false
	}
	switch ctx.Kind {
	case jsparse.CompletionProperty:
		return docs.ResolveProperty(ctx.BaseExpr, candidate.Label, aliases)
	case jsparse.CompletionIdentifier:
		return docs.ResolveIdentifier(candidate.Label, aliases)
	case jsparse.CompletionNone, jsparse.CompletionArgument:
		return nil, false
	default:
		return nil, false
	}
}

func docCompletionCandidates(
	docs *docsResolver,
	ctx jsparse.CompletionContext,
	aliases map[string]string,
) []jsparse.CompletionCandidate {
	if docs == nil {
		return nil
	}
	return docs.CompletionCandidates(ctx, aliases)
}
