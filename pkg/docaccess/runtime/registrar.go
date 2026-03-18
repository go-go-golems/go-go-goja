package runtime

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/glazed/pkg/help"
	"github.com/go-go-golems/go-go-goja/engine"
	"github.com/go-go-golems/go-go-goja/modules"
	"github.com/go-go-golems/go-go-goja/pkg/docaccess"
	glazedprovider "github.com/go-go-golems/go-go-goja/pkg/docaccess/glazed"
	jsdocprovider "github.com/go-go-golems/go-go-goja/pkg/docaccess/jsdoc"
	pluginprovider "github.com/go-go-golems/go-go-goja/pkg/docaccess/plugin"
	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/host"
	jsdocmodel "github.com/go-go-golems/go-go-goja/pkg/jsdoc/model"
)

type HelpSource struct {
	ID      string
	Title   string
	Summary string
	System  *help.HelpSystem
}

type JSDocSource struct {
	ID      string
	Title   string
	Summary string
	Store   *jsdocmodel.DocStore
}

type Config struct {
	ModuleName     string
	HelpSources    []HelpSource
	JSDocSources   []JSDocSource
	PluginSourceID string
}

type Registrar struct {
	config Config
}

const RuntimeHubContextKey = "docaccess.hub"

func NewRegistrar(config Config) *Registrar {
	return &Registrar{config: config}
}

func (r *Registrar) ID() string {
	return "docaccess-registrar"
}

func (r *Registrar) RegisterRuntimeModules(ctx *engine.RuntimeModuleContext, reg *require.Registry) error {
	if reg == nil {
		return fmt.Errorf("require registry is nil")
	}

	hub, err := r.buildHub(ctx)
	if err != nil {
		return err
	}
	if ctx != nil {
		ctx.SetValue(RuntimeHubContextKey, hub)
	}

	reg.RegisterNativeModule(r.moduleName(), loader(hub))
	return nil
}

func (r *Registrar) moduleName() string {
	name := strings.TrimSpace(r.config.ModuleName)
	if name == "" {
		return "docs"
	}
	return name
}

func (r *Registrar) buildHub(ctx *engine.RuntimeModuleContext) (*docaccess.Hub, error) {
	hub := docaccess.NewHub()

	helpSources := append([]HelpSource(nil), r.config.HelpSources...)
	sort.Slice(helpSources, func(i, j int) bool { return helpSources[i].ID < helpSources[j].ID })
	for _, source := range helpSources {
		if source.System == nil {
			continue
		}
		provider, err := glazedprovider.NewProvider(source.ID, source.Title, source.Summary, source.System)
		if err != nil {
			return nil, err
		}
		if err := hub.Register(provider); err != nil {
			return nil, err
		}
	}

	jsdocSources := append([]JSDocSource(nil), r.config.JSDocSources...)
	sort.Slice(jsdocSources, func(i, j int) bool { return jsdocSources[i].ID < jsdocSources[j].ID })
	for _, source := range jsdocSources {
		if source.Store == nil {
			continue
		}
		provider, err := jsdocprovider.NewProvider(source.ID, source.Title, source.Summary, source.Store)
		if err != nil {
			return nil, err
		}
		if err := hub.Register(provider); err != nil {
			return nil, err
		}
	}

	if loaded, ok := ctx.Value(host.RuntimeLoadedModulesContextKey); ok {
		modules, ok := loaded.([]host.LoadedModuleInfo)
		if !ok {
			return nil, fmt.Errorf("unexpected type for %q: %T", host.RuntimeLoadedModulesContextKey, loaded)
		}
		provider, err := pluginprovider.NewProvider(r.pluginSourceID(), "Plugin Manifests", "Runtime-scoped plugin metadata", modules)
		if err != nil {
			return nil, err
		}
		if err := hub.Register(provider); err != nil {
			return nil, err
		}
	}

	return hub, nil
}

func (r *Registrar) pluginSourceID() string {
	sourceID := strings.TrimSpace(r.config.PluginSourceID)
	if sourceID == "" {
		return "plugin-manifests"
	}
	return sourceID
}

func loader(hub *docaccess.Hub) require.ModuleLoader {
	return func(vm *goja.Runtime, moduleObj *goja.Object) {
		exports := moduleObj.Get("exports").(*goja.Object)

		modules.SetExport(exports, "docs", "sources", func() any {
			return sourceDescriptorsToMaps(hub.Sources())
		})
		modules.SetExport(exports, "docs", "search", func(input map[string]any) (any, error) {
			entries, err := hub.Search(context.Background(), queryFromMap(input))
			if err != nil {
				return nil, err
			}
			return entriesToMaps(entries), nil
		})
		modules.SetExport(exports, "docs", "get", func(input map[string]any) (any, error) {
			entry, err := hub.Get(context.Background(), refFromMap(input))
			if err != nil {
				if errors.Is(err, docaccess.ErrEntryNotFound) {
					return nil, nil
				}
				return nil, err
			}
			return entryToMap(*entry), nil
		})
		modules.SetExport(exports, "docs", "byID", func(sourceID, kind, id string) (any, error) {
			entry, err := hub.FindByID(sourceID, kind, id)
			if err != nil {
				if errors.Is(err, docaccess.ErrEntryNotFound) {
					return nil, nil
				}
				return nil, err
			}
			return entryToMap(*entry), nil
		})
		modules.SetExport(exports, "docs", "bySlug", func(sourceID, slug string) (any, error) {
			entry, err := hub.FindByID(sourceID, glazedprovider.EntryKindHelpSection, slug)
			if err != nil {
				if errors.Is(err, docaccess.ErrEntryNotFound) {
					return nil, nil
				}
				return nil, err
			}
			return entryToMap(*entry), nil
		})
		modules.SetExport(exports, "docs", "bySymbol", func(sourceID, symbol string) (any, error) {
			entry, err := hub.FindByID(sourceID, jsdocprovider.EntryKindSymbol, symbol)
			if err != nil {
				if errors.Is(err, docaccess.ErrEntryNotFound) {
					return nil, nil
				}
				return nil, err
			}
			return entryToMap(*entry), nil
		})
	}
}

func queryFromMap(input map[string]any) docaccess.Query {
	if len(input) == 0 {
		return docaccess.Query{}
	}

	query := docaccess.Query{
		Text:      stringValue(input["text"]),
		SourceIDs: stringSliceValue(input["sourceIds"]),
		Kinds:     stringSliceValue(input["kinds"]),
		Topics:    stringSliceValue(input["topics"]),
		Tags:      stringSliceValue(input["tags"]),
	}
	if limit, ok := intValue(input["limit"]); ok {
		query.Limit = limit
	}
	return query
}

func refFromMap(input map[string]any) docaccess.EntryRef {
	return docaccess.EntryRef{
		SourceID: stringValue(input["sourceId"]),
		Kind:     stringValue(input["kind"]),
		ID:       stringValue(input["id"]),
	}
}

func sourceDescriptorsToMaps(sources []docaccess.SourceDescriptor) []map[string]any {
	out := make([]map[string]any, 0, len(sources))
	for _, source := range sources {
		out = append(out, map[string]any{
			"id":            source.ID,
			"kind":          string(source.Kind),
			"title":         source.Title,
			"summary":       source.Summary,
			"runtimeScoped": source.RuntimeScoped,
			"metadata":      source.Metadata,
		})
	}
	return out
}

func entriesToMaps(entries []docaccess.Entry) []map[string]any {
	out := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		out = append(out, entryToMap(entry))
	}
	return out
}

func entryToMap(entry docaccess.Entry) map[string]any {
	related := make([]map[string]any, 0, len(entry.Related))
	for _, ref := range entry.Related {
		related = append(related, refToMap(ref))
	}

	return map[string]any{
		"ref":       refToMap(entry.Ref),
		"title":     entry.Title,
		"summary":   entry.Summary,
		"body":      entry.Body,
		"topics":    append([]string(nil), entry.Topics...),
		"tags":      append([]string(nil), entry.Tags...),
		"path":      entry.Path,
		"kindLabel": entry.KindLabel,
		"related":   related,
		"metadata":  entry.Metadata,
	}
}

func refToMap(ref docaccess.EntryRef) map[string]any {
	return map[string]any{
		"sourceId": ref.SourceID,
		"kind":     ref.Kind,
		"id":       ref.ID,
	}
}

func stringValue(value any) string {
	s, _ := value.(string)
	return strings.TrimSpace(s)
}

func intValue(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	default:
		return 0, false
	}
}

func stringSliceValue(value any) []string {
	switch v := value.(type) {
	case nil:
		return nil
	case []string:
		return append([]string(nil), v...)
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, strings.TrimSpace(s))
			}
		}
		return out
	default:
		return nil
	}
}
