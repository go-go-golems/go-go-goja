package host

import (
	"testing"

	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/contract"
)

func TestReportCollectorSnapshot(t *testing.T) {
	reporter := NewReportCollector([]string{"/plugins"})
	reporter.SetCandidates([]string{"/plugins/goja-plugin-examples-greeter"})
	reporter.AddLoaded(&LoadedModule{
		Path: "/plugins/goja-plugin-examples-greeter",
		Manifest: &contract.ModuleManifest{
			ModuleName: "plugin:examples:greeter",
			Version:    "v1",
			Exports: []*contract.ExportSpec{
				{Name: "greet", Kind: contract.ExportKind_EXPORT_KIND_FUNCTION},
				{Name: "strings", Kind: contract.ExportKind_EXPORT_KIND_OBJECT, MethodSpecs: []*contract.MethodSpec{{Name: "upper"}, {Name: "lower"}}},
			},
		},
	})

	snapshot := reporter.Snapshot()
	if len(snapshot.Directories) != 1 || snapshot.Directories[0] != "/plugins" {
		t.Fatalf("directories = %#v", snapshot.Directories)
	}
	if len(snapshot.Candidates) != 1 || snapshot.Candidates[0] != "/plugins/goja-plugin-examples-greeter" {
		t.Fatalf("candidates = %#v", snapshot.Candidates)
	}
	if len(snapshot.Loaded) != 1 {
		t.Fatalf("loaded = %#v", snapshot.Loaded)
	}
	if got := snapshot.Loaded[0].ModuleName; got != "plugin:examples:greeter" {
		t.Fatalf("module name = %q", got)
	}
	if got := snapshot.Loaded[0].Exports; len(got) != 3 || got[0] != "greet" || got[1] != "strings.upper" || got[2] != "strings.lower" {
		t.Fatalf("exports = %#v", got)
	}
}

func TestLoadReportSummary(t *testing.T) {
	report := LoadReport{
		Loaded: []LoadedModuleSummary{
			{ModuleName: "plugin:examples:greeter"},
			{ModuleName: "plugin:echo"},
		},
	}
	if got := report.Summary(); got != "plugins loaded: plugin:examples:greeter, plugin:echo" {
		t.Fatalf("summary = %q", got)
	}
}
