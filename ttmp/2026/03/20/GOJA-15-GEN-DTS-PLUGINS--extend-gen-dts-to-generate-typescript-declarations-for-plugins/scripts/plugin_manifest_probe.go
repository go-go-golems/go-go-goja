package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/contract"
	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/host"
)

type exportSummary struct {
	Name    string   `json:"name"`
	Kind    string   `json:"kind"`
	Doc     string   `json:"doc,omitempty"`
	Methods []string `json:"methods,omitempty"`
}

type moduleSummary struct {
	Path         string          `json:"path"`
	ModuleName   string          `json:"moduleName"`
	Version      string          `json:"version,omitempty"`
	Doc          string          `json:"doc,omitempty"`
	Capabilities []string        `json:"capabilities,omitempty"`
	Exports      []exportSummary `json:"exports"`
}

func main() {
	var pluginDirs csvFlag
	var allowModules csvFlag
	format := flag.String("format", "both", "Output format: json, dts, both")
	flag.Var(&pluginDirs, "plugin-dir", "Directory containing go-go-goja plugin binaries")
	flag.Var(&allowModules, "allow-plugin-module", "Restrict to module names")
	flag.Parse()

	cfg := host.Config{
		Directories:  host.ResolveDiscoveryDirectories(pluginDirs),
		AllowModules: allowModules,
	}
	paths, err := host.Discover(cfg)
	if err != nil {
		fail(err)
	}
	if len(paths) == 0 {
		fail(fmt.Errorf("no plugins discovered"))
	}

	loaded, err := host.LoadModules(cfg, paths)
	if err != nil {
		fail(err)
	}
	defer func() {
		for _, mod := range loaded {
			mod.Close()
		}
	}()

	summaries := summarize(loaded)
	switch strings.TrimSpace(*format) {
	case "json":
		printJSON(summaries)
	case "dts":
		fmt.Print(renderBestEffortDTS(loaded))
	case "both":
		printJSON(summaries)
		fmt.Println()
		fmt.Print(renderBestEffortDTS(loaded))
	default:
		fail(fmt.Errorf("unsupported --format %q", *format))
	}
}

func summarize(loaded []*host.LoadedModule) []moduleSummary {
	out := make([]moduleSummary, 0, len(loaded))
	for _, mod := range loaded {
		if mod == nil || mod.Manifest == nil {
			continue
		}
		summary := moduleSummary{
			Path:         mod.Path,
			ModuleName:   mod.Manifest.GetModuleName(),
			Version:      mod.Manifest.GetVersion(),
			Doc:          mod.Manifest.GetDoc(),
			Capabilities: append([]string(nil), mod.Manifest.GetCapabilities()...),
			Exports:      make([]exportSummary, 0, len(mod.Manifest.GetExports())),
		}
		for _, exp := range mod.Manifest.GetExports() {
			if exp == nil {
				continue
			}
			item := exportSummary{
				Name: exp.GetName(),
				Kind: exp.GetKind().String(),
				Doc:  exp.GetDoc(),
			}
			for _, method := range exp.GetMethodSpecs() {
				if method == nil {
					continue
				}
				item.Methods = append(item.Methods, method.GetName())
			}
			sort.Strings(item.Methods)
			summary.Exports = append(summary.Exports, item)
		}
		sort.Slice(summary.Exports, func(i, j int) bool {
			return summary.Exports[i].Name < summary.Exports[j].Name
		})
		out = append(out, summary)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ModuleName < out[j].ModuleName
	})
	return out
}

func printJSON(summaries []moduleSummary) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(summaries); err != nil {
		fail(err)
	}
}

func renderBestEffortDTS(loaded []*host.LoadedModule) string {
	var b strings.Builder
	b.WriteString("// Best-effort plugin declarations derived from runtime manifests.\n")
	b.WriteString("// Signatures are unknown because the current plugin manifest does not encode params or return types.\n\n")
	for i, mod := range loaded {
		if mod == nil || mod.Manifest == nil {
			continue
		}
		renderModule(&b, mod.Manifest)
		if i < len(loaded)-1 {
			b.WriteString("\n\n")
		}
	}
	return b.String()
}

func renderModule(b *strings.Builder, manifest *contract.ModuleManifest) {
	fmt.Fprintf(b, "declare module %q {\n", manifest.GetModuleName())
	exports := append([]*contract.ExportSpec(nil), manifest.GetExports()...)
	sort.Slice(exports, func(i, j int) bool {
		return exports[i].GetName() < exports[j].GetName()
	})
	for _, exp := range exports {
		if exp == nil {
			continue
		}
		switch exp.GetKind() {
		case contract.ExportKind_EXPORT_KIND_UNSPECIFIED:
			continue
		case contract.ExportKind_EXPORT_KIND_FUNCTION:
			fmt.Fprintf(b, "  export function %s(...args: unknown[]): unknown;\n", exp.GetName())
		case contract.ExportKind_EXPORT_KIND_OBJECT:
			b.WriteString("  export const ")
			b.WriteString(exp.GetName())
			b.WriteString(": {\n")
			methods := append([]*contract.MethodSpec(nil), exp.GetMethodSpecs()...)
			sort.Slice(methods, func(i, j int) bool {
				return methods[i].GetName() < methods[j].GetName()
			})
			for _, method := range methods {
				if method == nil {
					continue
				}
				fmt.Fprintf(b, "    %s(...args: unknown[]): unknown;\n", method.GetName())
			}
			b.WriteString("  };\n")
		default:
			continue
		}
	}
	b.WriteString("}")
}

type csvFlag []string

func (c *csvFlag) String() string {
	return strings.Join(*c, ",")
}

func (c *csvFlag) Set(value string) error {
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		*c = append(*c, part)
	}
	return nil
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
