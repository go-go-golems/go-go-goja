package javascript

import (
	"context"
	"errors"
	"strings"

	ggjengine "github.com/go-go-golems/go-go-goja/engine"
	"github.com/go-go-golems/go-go-goja/pkg/docaccess"
	pluginprovider "github.com/go-go-golems/go-go-goja/pkg/docaccess/plugin"
	docaccessruntime "github.com/go-go-golems/go-go-goja/pkg/docaccess/runtime"
	"github.com/go-go-golems/go-go-goja/pkg/jsparse"
)

// docsResolver keeps the evaluator-side lookup seam narrow even though the
// runtime stores the raw docs hub.
type docsResolver struct {
	hub             *docaccess.Hub
	pluginSourceIDs []string
	moduleEntries   map[string]docaccess.Entry
	exportEntries   map[string]map[string]docaccess.Entry
	methodEntries   map[string]map[string]map[string]docaccess.Entry
}

func newDocsResolver(runtime *ggjengine.Runtime) *docsResolver {
	return newDocsResolverFromHub(docHubFromRuntime(runtime))
}

func docHubFromRuntime(runtime *ggjengine.Runtime) *docaccess.Hub {
	if runtime == nil {
		return nil
	}
	value, ok := runtime.Value(docaccessruntime.RuntimeHubContextKey)
	if !ok {
		return nil
	}
	hub, ok := value.(*docaccess.Hub)
	if !ok || hub == nil {
		return nil
	}
	return hub
}

// DocHubFromRuntime exposes the runtime docs hub for cross-package assistance
// adapters without leaking the private resolver type.
func DocHubFromRuntime(runtime *ggjengine.Runtime) *docaccess.Hub {
	return docHubFromRuntime(runtime)
}

func newDocsResolverFromHub(hub *docaccess.Hub) *docsResolver {
	if hub == nil {
		return nil
	}
	sourceIDs := make([]string, 0, 1)
	for _, descriptor := range hub.Sources() {
		if descriptor.Kind == docaccess.SourceKindPlugin {
			sourceIDs = append(sourceIDs, descriptor.ID)
		}
	}
	if len(sourceIDs) == 0 {
		return nil
	}

	return &docsResolver{
		hub:             hub,
		pluginSourceIDs: sourceIDs,
		moduleEntries:   indexModuleEntries(hub, sourceIDs),
		exportEntries:   indexExportEntries(hub, sourceIDs),
		methodEntries:   indexMethodEntries(hub, sourceIDs),
	}
}

func (r *docsResolver) ResolveIdentifier(name string, aliases map[string]string) (*docaccess.Entry, bool) {
	if r == nil {
		return nil, false
	}
	moduleName, ok := resolvePluginModuleName(strings.TrimSpace(name), aliases)
	if !ok {
		return nil, false
	}
	return r.findEntry(pluginprovider.EntryKindPluginModule, moduleName)
}

func (r *docsResolver) ResolveProperty(baseExpr, property string, aliases map[string]string) (*docaccess.Entry, bool) {
	if r == nil {
		return nil, false
	}
	parts := splitPropertyPath(baseExpr)
	if len(parts) == 0 || len(parts) > 2 {
		return nil, false
	}

	moduleName, ok := resolvePluginModuleName(parts[0], aliases)
	if !ok {
		return nil, false
	}

	property = strings.TrimSpace(property)
	if len(parts) == 1 {
		if property == "" {
			return r.findEntry(pluginprovider.EntryKindPluginModule, moduleName)
		}
		return r.findEntry(pluginprovider.EntryKindPluginExport, moduleName+"/"+property)
	}

	exportName := parts[1]
	if property == "" {
		return r.findEntry(pluginprovider.EntryKindPluginExport, moduleName+"/"+exportName)
	}
	return r.findEntry(pluginprovider.EntryKindPluginMethod, moduleName+"/"+exportName+"."+property)
}

func (r *docsResolver) ResolveToken(token string, aliases map[string]string) (*docaccess.Entry, bool) {
	token = strings.Trim(strings.TrimSpace(token), ".")
	if token == "" {
		return nil, false
	}
	parts := splitPropertyPath(token)
	switch len(parts) {
	case 0:
		return nil, false
	case 1:
		return r.ResolveIdentifier(parts[0], aliases)
	default:
		return r.ResolveProperty(strings.Join(parts[:len(parts)-1], "."), parts[len(parts)-1], aliases)
	}
}

func (r *docsResolver) findEntry(kind, id string) (*docaccess.Entry, bool) {
	if r == nil || r.hub == nil || strings.TrimSpace(kind) == "" || strings.TrimSpace(id) == "" {
		return nil, false
	}
	if entry, ok := r.indexedEntry(kind, id); ok {
		return entry, true
	}
	for _, sourceID := range r.pluginSourceIDs {
		entry, err := r.hub.FindByID(sourceID, kind, id)
		if err == nil && entry != nil {
			return entry, true
		}
		if err != nil && !errors.Is(err, docaccess.ErrEntryNotFound) {
			return nil, false
		}
	}
	return nil, false
}

func (r *docsResolver) CompletionCandidates(ctx jsparse.CompletionContext, aliases map[string]string) []jsparse.CompletionCandidate {
	if r == nil || ctx.Kind != jsparse.CompletionProperty {
		return nil
	}

	parts := splitPropertyPath(ctx.BaseExpr)
	if len(parts) == 0 || len(parts) > 2 {
		return nil
	}

	moduleName, ok := resolvePluginModuleName(parts[0], aliases)
	if !ok {
		return nil
	}

	switch len(parts) {
	case 1:
		exports := r.exportEntries[moduleName]
		if len(exports) == 0 {
			return nil
		}
		out := make([]jsparse.CompletionCandidate, 0, len(exports))
		for name, entry := range exports {
			if !matchesCandidatePrefix(name, ctx.PartialText) {
				continue
			}
			candidateKind := jsparse.CandidateProperty
			if kind, _ := entry.Metadata["kind"].(string); strings.Contains(strings.ToLower(kind), "function") {
				candidateKind = jsparse.CandidateFunction
			}
			out = append(out, jsparse.CompletionCandidate{
				Label:  name,
				Kind:   candidateKind,
				Detail: "plugin export",
			})
		}
		return out
	case 2:
		methods := r.methodEntries[moduleName][parts[1]]
		if len(methods) == 0 {
			return nil
		}
		out := make([]jsparse.CompletionCandidate, 0, len(methods))
		for name := range methods {
			if !matchesCandidatePrefix(name, ctx.PartialText) {
				continue
			}
			out = append(out, jsparse.CompletionCandidate{
				Label:  name,
				Kind:   jsparse.CandidateMethod,
				Detail: "plugin method",
			})
		}
		return out
	default:
		return nil
	}
}

func (r *docsResolver) indexedEntry(kind, id string) (*docaccess.Entry, bool) {
	switch kind {
	case pluginprovider.EntryKindPluginModule:
		entry, ok := r.moduleEntries[id]
		if !ok {
			return nil, false
		}
		return entryPtr(entry), true
	case pluginprovider.EntryKindPluginExport:
		moduleName, exportName, ok := parseExportKey(id)
		if !ok {
			return nil, false
		}
		entry, ok := r.exportEntries[moduleName][exportName]
		if !ok {
			return nil, false
		}
		return entryPtr(entry), true
	case pluginprovider.EntryKindPluginMethod:
		moduleName, exportName, methodName, ok := parseMethodKey(id)
		if !ok {
			return nil, false
		}
		entry, ok := r.methodEntries[moduleName][exportName][methodName]
		if !ok {
			return nil, false
		}
		return entryPtr(entry), true
	default:
		return nil, false
	}
}

func resolvePluginModuleName(root string, aliases map[string]string) (string, bool) {
	root = strings.TrimSpace(root)
	if root == "" {
		return "", false
	}
	if aliases != nil {
		if moduleName, ok := aliases[root]; ok && strings.HasPrefix(moduleName, "plugin:") {
			return moduleName, true
		}
	}
	if strings.HasPrefix(root, "plugin:") {
		return root, true
	}
	return "", false
}

func splitPropertyPath(input string) []string {
	rawParts := strings.Split(strings.TrimSpace(input), ".")
	out := make([]string, 0, len(rawParts))
	for _, part := range rawParts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}

func indexModuleEntries(hub *docaccess.Hub, sourceIDs []string) map[string]docaccess.Entry {
	entries := searchPluginEntries(hub, sourceIDs)
	out := map[string]docaccess.Entry{}
	for _, entry := range entries {
		if entry.Ref.Kind == pluginprovider.EntryKindPluginModule {
			out[entry.Ref.ID] = entry
		}
	}
	return out
}

func indexExportEntries(hub *docaccess.Hub, sourceIDs []string) map[string]map[string]docaccess.Entry {
	entries := searchPluginEntries(hub, sourceIDs)
	out := map[string]map[string]docaccess.Entry{}
	for _, entry := range entries {
		if entry.Ref.Kind != pluginprovider.EntryKindPluginExport {
			continue
		}
		moduleName, _ := entry.Metadata["moduleName"].(string)
		exportName, _ := entry.Metadata["exportName"].(string)
		if moduleName == "" || exportName == "" {
			continue
		}
		if out[moduleName] == nil {
			out[moduleName] = map[string]docaccess.Entry{}
		}
		out[moduleName][exportName] = entry
	}
	return out
}

func indexMethodEntries(hub *docaccess.Hub, sourceIDs []string) map[string]map[string]map[string]docaccess.Entry {
	entries := searchPluginEntries(hub, sourceIDs)
	out := map[string]map[string]map[string]docaccess.Entry{}
	for _, entry := range entries {
		if entry.Ref.Kind != pluginprovider.EntryKindPluginMethod {
			continue
		}
		moduleName, _ := entry.Metadata["moduleName"].(string)
		exportName, _ := entry.Metadata["exportName"].(string)
		methodName, _ := entry.Metadata["methodName"].(string)
		if moduleName == "" || exportName == "" || methodName == "" {
			continue
		}
		if out[moduleName] == nil {
			out[moduleName] = map[string]map[string]docaccess.Entry{}
		}
		if out[moduleName][exportName] == nil {
			out[moduleName][exportName] = map[string]docaccess.Entry{}
		}
		out[moduleName][exportName][methodName] = entry
	}
	return out
}

func searchPluginEntries(hub *docaccess.Hub, sourceIDs []string) []docaccess.Entry {
	if hub == nil || len(sourceIDs) == 0 {
		return nil
	}
	entries, err := hub.Search(context.Background(), docaccess.Query{SourceIDs: sourceIDs})
	if err != nil {
		return nil
	}
	return entries
}

func parseExportKey(id string) (string, string, bool) {
	index := strings.LastIndex(id, "/")
	if index <= 0 || index == len(id)-1 {
		return "", "", false
	}
	return id[:index], id[index+1:], true
}

func parseMethodKey(id string) (string, string, string, bool) {
	index := strings.LastIndex(id, ".")
	if index <= 0 || index == len(id)-1 {
		return "", "", "", false
	}
	moduleName, exportName, ok := parseExportKey(id[:index])
	if !ok {
		return "", "", "", false
	}
	return moduleName, exportName, id[index+1:], true
}

func matchesCandidatePrefix(name, partial string) bool {
	if strings.TrimSpace(partial) == "" {
		return true
	}
	return strings.HasPrefix(strings.ToLower(name), strings.ToLower(strings.TrimSpace(partial)))
}

func entryPtr(entry docaccess.Entry) *docaccess.Entry {
	entryCopy := entry
	return &entryCopy
}
