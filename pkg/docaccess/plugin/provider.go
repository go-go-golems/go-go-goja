package plugin

import (
	"context"
	"strings"

	"github.com/go-go-golems/go-go-goja/pkg/docaccess"
	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/contract"
	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/host"
)

const (
	DefaultSourceID       = "plugin-manifests"
	EntryKindPluginModule = "plugin-module"
	EntryKindPluginExport = "plugin-export"
	EntryKindPluginMethod = "plugin-method"
)

type Provider struct {
	sourceID string
	title    string
	summary  string
	modules  []host.LoadedModuleInfo
}

func NewProvider(sourceID, title, summary string, modules []host.LoadedModuleInfo) (*Provider, error) {
	if sourceID == "" {
		sourceID = DefaultSourceID
	}
	if title == "" {
		title = "Plugin Manifests"
	}
	return &Provider{
		sourceID: sourceID,
		title:    title,
		summary:  summary,
		modules:  append([]host.LoadedModuleInfo(nil), modules...),
	}, nil
}

func (p *Provider) Descriptor() docaccess.SourceDescriptor {
	return docaccess.SourceDescriptor{
		ID:            p.sourceID,
		Kind:          docaccess.SourceKindPlugin,
		Title:         p.title,
		Summary:       p.summary,
		RuntimeScoped: true,
	}
}

func (p *Provider) List(ctx context.Context) ([]docaccess.EntryRef, error) {
	entries, err := p.Search(ctx, docaccess.Query{})
	if err != nil {
		return nil, err
	}
	out := make([]docaccess.EntryRef, 0, len(entries))
	for _, entry := range entries {
		out = append(out, entry.Ref)
	}
	return out, nil
}

func (p *Provider) Get(_ context.Context, ref docaccess.EntryRef) (*docaccess.Entry, error) {
	if ref.SourceID != p.sourceID {
		return nil, docaccess.ErrEntryNotFound
	}

	switch ref.Kind {
	case EntryKindPluginModule:
		info, ok := p.findModule(ref.ID)
		if !ok {
			return nil, docaccess.ErrEntryNotFound
		}
		entry := moduleEntry(p.sourceID, info)
		return &entry, nil
	case EntryKindPluginExport:
		moduleName, exportName, ok := parseExportID(ref.ID)
		if !ok {
			return nil, docaccess.ErrEntryNotFound
		}
		info, exp, ok := p.findExport(moduleName, exportName)
		if !ok {
			return nil, docaccess.ErrEntryNotFound
		}
		entry := exportEntry(p.sourceID, info, exp)
		return &entry, nil
	case EntryKindPluginMethod:
		moduleName, exportName, methodName, ok := parseMethodID(ref.ID)
		if !ok {
			return nil, docaccess.ErrEntryNotFound
		}
		info, exp, method, ok := p.findMethod(moduleName, exportName, methodName)
		if !ok {
			return nil, docaccess.ErrEntryNotFound
		}
		entry := methodEntry(p.sourceID, info, exp, method)
		return &entry, nil
	default:
		return nil, docaccess.ErrEntryNotFound
	}
}

func (p *Provider) Search(_ context.Context, _ docaccess.Query) ([]docaccess.Entry, error) {
	out := make([]docaccess.Entry, 0, len(p.modules)*4)
	for _, info := range p.modules {
		if info.Manifest == nil {
			continue
		}
		out = append(out, moduleEntry(p.sourceID, info))
		for _, exp := range info.Manifest.GetExports() {
			if exp == nil {
				continue
			}
			out = append(out, exportEntry(p.sourceID, info, exp))
			for _, method := range exp.GetMethodSpecs() {
				if method == nil {
					continue
				}
				out = append(out, methodEntry(p.sourceID, info, exp, method))
			}
		}
	}
	return out, nil
}

func (p *Provider) findModule(moduleName string) (host.LoadedModuleInfo, bool) {
	for _, info := range p.modules {
		if info.Manifest != nil && info.Manifest.GetModuleName() == moduleName {
			return info, true
		}
	}
	return host.LoadedModuleInfo{}, false
}

func (p *Provider) findExport(moduleName, exportName string) (host.LoadedModuleInfo, *contract.ExportSpec, bool) {
	info, ok := p.findModule(moduleName)
	if !ok {
		return host.LoadedModuleInfo{}, nil, false
	}
	for _, exp := range info.Manifest.GetExports() {
		if exp != nil && exp.GetName() == exportName {
			return info, exp, true
		}
	}
	return host.LoadedModuleInfo{}, nil, false
}

func (p *Provider) findMethod(moduleName, exportName, methodName string) (host.LoadedModuleInfo, *contract.ExportSpec, *contract.MethodSpec, bool) {
	info, exp, ok := p.findExport(moduleName, exportName)
	if !ok {
		return host.LoadedModuleInfo{}, nil, nil, false
	}
	for _, method := range exp.GetMethodSpecs() {
		if method != nil && method.GetName() == methodName {
			return info, exp, method, true
		}
	}
	return host.LoadedModuleInfo{}, nil, nil, false
}

func moduleEntry(sourceID string, info host.LoadedModuleInfo) docaccess.Entry {
	manifest := info.Manifest
	related := make([]docaccess.EntryRef, 0, len(manifest.GetExports()))
	for _, exp := range manifest.GetExports() {
		if exp == nil {
			continue
		}
		related = append(related, docaccess.EntryRef{
			SourceID: sourceID,
			Kind:     EntryKindPluginExport,
			ID:       exportID(manifest.GetModuleName(), exp.GetName()),
		})
	}

	return docaccess.Entry{
		Ref:       docaccess.EntryRef{SourceID: sourceID, Kind: EntryKindPluginModule, ID: manifest.GetModuleName()},
		Title:     manifest.GetModuleName(),
		Summary:   summaryFrom(manifest.GetDoc()),
		Body:      manifest.GetDoc(),
		Path:      info.Path,
		KindLabel: "Plugin Module",
		Related:   related,
		Metadata: map[string]any{
			"moduleName":    manifest.GetModuleName(),
			"version":       manifest.GetVersion(),
			"capabilities":  append([]string(nil), manifest.GetCapabilities()...),
			"exportCount":   len(manifest.GetExports()),
			"runtimeScoped": true,
		},
	}
}

func exportEntry(sourceID string, info host.LoadedModuleInfo, exp *contract.ExportSpec) docaccess.Entry {
	related := []docaccess.EntryRef{{
		SourceID: sourceID,
		Kind:     EntryKindPluginModule,
		ID:       info.Manifest.GetModuleName(),
	}}
	for _, method := range exp.GetMethodSpecs() {
		if method == nil {
			continue
		}
		related = append(related, docaccess.EntryRef{
			SourceID: sourceID,
			Kind:     EntryKindPluginMethod,
			ID:       methodID(info.Manifest.GetModuleName(), exp.GetName(), method.GetName()),
		})
	}

	return docaccess.Entry{
		Ref:       docaccess.EntryRef{SourceID: sourceID, Kind: EntryKindPluginExport, ID: exportID(info.Manifest.GetModuleName(), exp.GetName())},
		Title:     exp.GetName(),
		Summary:   summaryFrom(exp.GetDoc()),
		Body:      exp.GetDoc(),
		Path:      info.Path,
		KindLabel: "Plugin Export",
		Related:   related,
		Metadata: map[string]any{
			"moduleName":  info.Manifest.GetModuleName(),
			"exportName":  exp.GetName(),
			"kind":        exp.GetKind().String(),
			"methodCount": len(exp.GetMethodSpecs()),
		},
	}
}

func methodEntry(sourceID string, info host.LoadedModuleInfo, exp *contract.ExportSpec, method *contract.MethodSpec) docaccess.Entry {
	related := []docaccess.EntryRef{
		{SourceID: sourceID, Kind: EntryKindPluginModule, ID: info.Manifest.GetModuleName()},
		{SourceID: sourceID, Kind: EntryKindPluginExport, ID: exportID(info.Manifest.GetModuleName(), exp.GetName())},
	}

	return docaccess.Entry{
		Ref:       docaccess.EntryRef{SourceID: sourceID, Kind: EntryKindPluginMethod, ID: methodID(info.Manifest.GetModuleName(), exp.GetName(), method.GetName())},
		Title:     exp.GetName() + "." + method.GetName(),
		Summary:   summaryFrom(method.GetSummary(), method.GetDoc()),
		Body:      method.GetDoc(),
		Tags:      append([]string(nil), method.GetTags()...),
		Path:      info.Path,
		KindLabel: "Plugin Method",
		Related:   related,
		Metadata: map[string]any{
			"moduleName": info.Manifest.GetModuleName(),
			"exportName": exp.GetName(),
			"methodName": method.GetName(),
		},
	}
}

func exportID(moduleName, exportName string) string {
	return moduleName + "/" + exportName
}

func methodID(moduleName, exportName, methodName string) string {
	return exportID(moduleName, exportName) + "." + methodName
}

func parseExportID(id string) (string, string, bool) {
	index := strings.LastIndex(id, "/")
	if index <= 0 || index == len(id)-1 {
		return "", "", false
	}
	return id[:index], id[index+1:], true
}

func parseMethodID(id string) (string, string, string, bool) {
	slash := strings.LastIndex(id, "/")
	dot := strings.LastIndex(id, ".")
	if slash <= 0 || dot <= slash+1 || dot == len(id)-1 {
		return "", "", "", false
	}
	return id[:slash], id[slash+1 : dot], id[dot+1:], true
}

func summaryFrom(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			if idx := strings.Index(value, "\n\n"); idx >= 0 {
				return strings.TrimSpace(value[:idx])
			}
			if idx := strings.IndexByte(value, '\n'); idx >= 0 {
				return strings.TrimSpace(value[:idx])
			}
			return value
		}
	}
	return ""
}
