package host

import (
	"fmt"
	"strings"
	"sync"

	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/contract"
)

// LoadReport is a user-facing snapshot of plugin discovery and loading.
type LoadReport struct {
	Directories []string
	Candidates  []string
	Loaded      []LoadedModuleSummary
	Error       string
}

// LoadedModuleSummary records the loaded manifest in a CLI-friendly form.
type LoadedModuleSummary struct {
	Path       string
	ModuleName string
	Version    string
	Exports    []string
}

// ReportCollector records plugin discovery/load information during runtime setup.
type ReportCollector struct {
	mu     sync.Mutex
	report LoadReport
}

func NewReportCollector(directories []string) *ReportCollector {
	return &ReportCollector{
		report: LoadReport{
			Directories: append([]string(nil), directories...),
		},
	}
}

func (r *ReportCollector) SetCandidates(paths []string) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.report.Candidates = append([]string(nil), paths...)
}

func (r *ReportCollector) AddLoaded(mod *LoadedModule) {
	if r == nil || mod == nil || mod.Manifest == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.report.Loaded = append(r.report.Loaded, LoadedModuleSummary{
		Path:       mod.Path,
		ModuleName: mod.Manifest.GetModuleName(),
		Version:    mod.Manifest.GetVersion(),
		Exports:    summarizeExports(mod.Manifest),
	})
}

func (r *ReportCollector) SetError(err error) {
	if r == nil || err == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.report.Error = err.Error()
}

func (r *ReportCollector) Snapshot() LoadReport {
	if r == nil {
		return LoadReport{}
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	out := LoadReport{
		Directories: append([]string(nil), r.report.Directories...),
		Candidates:  append([]string(nil), r.report.Candidates...),
		Error:       r.report.Error,
	}
	if len(r.report.Loaded) > 0 {
		out.Loaded = make([]LoadedModuleSummary, 0, len(r.report.Loaded))
		for _, loaded := range r.report.Loaded {
			out.Loaded = append(out.Loaded, LoadedModuleSummary{
				Path:       loaded.Path,
				ModuleName: loaded.ModuleName,
				Version:    loaded.Version,
				Exports:    append([]string(nil), loaded.Exports...),
			})
		}
	}
	return out
}

func (r LoadReport) Summary() string {
	switch {
	case len(r.Loaded) > 0:
		names := make([]string, 0, len(r.Loaded))
		for _, loaded := range r.Loaded {
			names = append(names, loaded.ModuleName)
		}
		return fmt.Sprintf("plugins loaded: %s", strings.Join(names, ", "))
	case len(r.Candidates) > 0:
		return fmt.Sprintf("plugin candidates found: %d, but no modules loaded", len(r.Candidates))
	case len(r.Directories) > 0:
		return fmt.Sprintf("no plugins found under %s", strings.Join(r.Directories, ", "))
	default:
		return "no plugin directories configured"
	}
}

func (r LoadReport) DetailLines() []string {
	lines := make([]string, 0, 8)
	if len(r.Directories) == 0 {
		lines = append(lines, "Plugin discovery directories: none")
	} else {
		lines = append(lines, "Plugin discovery directories:")
		for _, dir := range r.Directories {
			lines = append(lines, "  - "+dir)
		}
	}

	lines = append(lines, fmt.Sprintf("Plugin candidates discovered: %d", len(r.Candidates)))
	for _, candidate := range r.Candidates {
		lines = append(lines, "  - "+candidate)
	}

	lines = append(lines, fmt.Sprintf("Plugin modules loaded: %d", len(r.Loaded)))
	for _, loaded := range r.Loaded {
		line := fmt.Sprintf("  - %s", loaded.ModuleName)
		if strings.TrimSpace(loaded.Version) != "" {
			line += " (" + loaded.Version + ")"
		}
		if strings.TrimSpace(loaded.Path) != "" {
			line += " <- " + loaded.Path
		}
		lines = append(lines, line)
		if len(loaded.Exports) > 0 {
			lines = append(lines, "    exports: "+strings.Join(loaded.Exports, ", "))
		}
	}

	if strings.TrimSpace(r.Error) != "" {
		lines = append(lines, "Plugin loading error: "+r.Error)
	}

	return lines
}

func summarizeExports(manifest *contract.ModuleManifest) []string {
	exports := manifest.GetExports()
	out := make([]string, 0, len(exports))
	for _, exp := range exports {
		if exp == nil {
			continue
		}
		name := strings.TrimSpace(exp.GetName())
		if name == "" {
			continue
		}
		if len(exp.GetMethodSpecs()) == 0 {
			out = append(out, name)
			continue
		}
		for _, method := range exp.GetMethodSpecs() {
			methodName := strings.TrimSpace(method.GetName())
			if methodName == "" {
				continue
			}
			out = append(out, name+"."+methodName)
		}
	}
	return out
}
