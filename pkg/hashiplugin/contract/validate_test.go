package contract

import (
	"strings"
	"testing"
)

func TestValidateManifestAcceptsValidModule(t *testing.T) {
	manifest := &ModuleManifest{
		ModuleName: "plugin:examples:greeter",
		Exports: []*ExportSpec{
			{Name: "greet", Kind: ExportKind_EXPORT_KIND_FUNCTION},
			{Name: "strings", Kind: ExportKind_EXPORT_KIND_OBJECT, MethodSpecs: []*MethodSpec{{Name: "upper"}, {Name: "lower"}}},
		},
	}

	err := ValidateManifest(manifest, ManifestValidationOptions{
		NamespacePrefix: "plugin:",
		AllowModules:    []string{"plugin:examples:greeter"},
	})
	if err != nil {
		t.Fatalf("validate manifest: %v", err)
	}
}

func TestValidateManifestRejectsInvalidShapes(t *testing.T) {
	testCases := []struct {
		name     string
		manifest *ModuleManifest
		opts     ManifestValidationOptions
		wantErr  string
	}{
		{
			name:     "nil manifest",
			manifest: nil,
			wantErr:  "manifest is nil",
		},
		{
			name: "wrong namespace",
			manifest: &ModuleManifest{
				ModuleName: "example:greeter",
			},
			opts:    ManifestValidationOptions{NamespacePrefix: "plugin:"},
			wantErr: "must use namespace",
		},
		{
			name: "not allowlisted",
			manifest: &ModuleManifest{
				ModuleName: "plugin:examples:greeter",
			},
			opts: ManifestValidationOptions{
				AllowModules: []string{"plugin:examples:clock"},
			},
			wantErr: "not in the allowlist",
		},
		{
			name: "duplicate exports",
			manifest: &ModuleManifest{
				ModuleName: "plugin:examples:greeter",
				Exports: []*ExportSpec{
					{Name: "greet", Kind: ExportKind_EXPORT_KIND_FUNCTION},
					{Name: "greet", Kind: ExportKind_EXPORT_KIND_FUNCTION},
				},
			},
			wantErr: "duplicate export",
		},
		{
			name: "object without methods",
			manifest: &ModuleManifest{
				ModuleName: "plugin:examples:greeter",
				Exports: []*ExportSpec{
					{Name: "strings", Kind: ExportKind_EXPORT_KIND_OBJECT},
				},
			},
			wantErr: "must define methods",
		},
		{
			name: "duplicate methods",
			manifest: &ModuleManifest{
				ModuleName: "plugin:examples:greeter",
				Exports: []*ExportSpec{
					{Name: "strings", Kind: ExportKind_EXPORT_KIND_OBJECT, MethodSpecs: []*MethodSpec{{Name: "upper"}, {Name: "upper"}}},
				},
			},
			wantErr: "duplicate method",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateManifest(tc.manifest, tc.opts)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}
