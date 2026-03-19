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

func TestLoadReportSummaryPrioritizesErrors(t *testing.T) {
	report := LoadReport{
		Candidates: []string{"/plugins/goja-plugin-examples-greeter"},
		Loaded: []LoadedModuleSummary{
			{ModuleName: "plugin:examples:greeter"},
		},
		Errors: []string{"start plugin failed"},
	}

	if got := report.Summary(); got != "plugin load errors: 1; loaded: 1" {
		t.Fatalf("summary = %q", got)
	}

	lines := report.DetailLines()
	if len(lines) == 0 || lines[len(lines)-2] != "Plugin loading errors: 1" {
		t.Fatalf("detail lines = %#v", lines)
	}
}

func TestWrapDiagnosticErrorIncludesBoundedStderr(t *testing.T) {
	buf := newBoundedDiagnosticBuffer(32)
	if _, err := buf.Write([]byte("stderr line one\nstderr line two")); err != nil {
		t.Fatalf("write diagnostics: %v", err)
	}

	err := wrapDiagnosticError(assertErr("boom"), buf)
	if err == nil {
		t.Fatalf("expected wrapped error")
	}
	if got := err.Error(); got != "boom; plugin stderr: stderr line one\nstderr line two" {
		t.Fatalf("wrapped error = %q", got)
	}
}

type testError string

func (e testError) Error() string { return string(e) }

func assertErr(msg string) error { return testError(msg) }
