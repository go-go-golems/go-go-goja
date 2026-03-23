package validate_test

import (
	"testing"

	"github.com/go-go-golems/go-go-goja/pkg/tsgen/spec"
	"github.com/go-go-golems/go-go-goja/pkg/tsgen/validate"
)

func TestModuleValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		module  *spec.Module
		wantErr bool
	}{
		{
			name: "valid",
			module: &spec.Module{
				Name: "fs",
				Functions: []spec.Function{
					{
						Name: "readFileSync",
						Params: []spec.Param{
							{Name: "path", Type: spec.String()},
						},
						Returns: spec.String(),
					},
				},
			},
		},
		{
			name: "empty module name",
			module: &spec.Module{
				Name: "   ",
			},
			wantErr: true,
		},
		{
			name: "duplicate function names",
			module: &spec.Module{
				Name: "exec",
				Functions: []spec.Function{
					{Name: "run", Returns: spec.String()},
					{Name: "run", Returns: spec.String()},
				},
			},
			wantErr: true,
		},
		{
			name: "empty parameter name",
			module: &spec.Module{
				Name: "exec",
				Functions: []spec.Function{
					{
						Name: "run",
						Params: []spec.Param{
							{Name: "", Type: spec.String()},
						},
						Returns: spec.String(),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid type tree",
			module: &spec.Module{
				Name: "database",
				Functions: []spec.Function{
					{
						Name:    "query",
						Returns: spec.Array(spec.Union()),
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validate.Module(tt.module)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
		})
	}
}

func TestBundleValidation(t *testing.T) {
	t.Parallel()

	bundle := &spec.Bundle{
		Modules: []*spec.Module{
			{Name: "fs", Functions: []spec.Function{{Name: "readFileSync", Returns: spec.String()}}},
		},
	}

	if err := validate.Bundle(bundle); err != nil {
		t.Fatalf("expected valid bundle, got %v", err)
	}

	if err := validate.Bundle(nil); err == nil {
		t.Fatalf("expected error for nil bundle")
	}
}
