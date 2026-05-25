package buildspec

import "testing"

func TestValidateCommandProvidersAcceptsKnownPackageAndRuntime(t *testing.T) {
	spec := validSpec()
	spec.CommandProviders = []CommandProviderInstance{{
		ID:             "fixture-tools",
		Package:        "fixture",
		Name:           "tools",
		RuntimeProfile: "main",
	}}

	report := Validate(spec)
	if report.HasErrors() {
		t.Fatalf("expected no validation errors, got %#v", report.Checks)
	}
	assertCheck(t, report, StatusOK, "command-provider-runtime", "commandProviders[0].runtimeProfile")
	assertCheck(t, report, StatusOK, "command-providers", "commandProviders")
}

func TestValidateCommandProvidersRejectsInvalidEntries(t *testing.T) {
	spec := validSpec()
	spec.CommandProviders = []CommandProviderInstance{
		{ID: "dup", Package: "missing", Name: "tools", RuntimeProfile: "missing"},
		{ID: "dup", Package: "fixture"},
	}

	report := Validate(spec)
	if !report.HasErrors() {
		t.Fatalf("expected validation errors, got %#v", report.Checks)
	}
	assertCheck(t, report, StatusError, "command-provider-package", "commandProviders[0].package")
	assertCheck(t, report, StatusError, "command-provider-runtime", "commandProviders[0].runtimeProfile")
	assertCheck(t, report, StatusError, "command-provider-id", "commandProviders[1].id")
	assertCheck(t, report, StatusError, "command-provider-name", "commandProviders[1].name")
}

func validSpec() *Spec {
	return &Spec{
		Name: "fixture",
		Target: TargetSpec{
			Kind:   "xgoja",
			Output: "dist/fixture",
		},
		Packages: []PackageSpec{{
			ID:     "fixture",
			Import: "github.com/go-go-golems/go-go-goja/pkg/xgoja/testprovider",
		}},
		Runtimes: map[string]Runtime{
			"main": {
				Modules: []ModuleInstance{{
					Package: "fixture",
					Name:    "hello",
					As:      "hello",
				}},
			},
		},
		Commands: CommandsSpec{
			Run: CommandSpec{Enabled: false},
		},
	}
}

func assertCheck(t *testing.T, report *Report, status Status, name string, path string) {
	t.Helper()
	for _, check := range report.Checks {
		if check.Status == status && check.Name == name && check.Path == path {
			return
		}
	}
	t.Fatalf("missing %s check %s at %s in %#v", status, name, path, report.Checks)
}
