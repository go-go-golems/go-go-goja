package generate

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/buildspec"
)

type Options struct {
	XGojaModuleVersion string
	XGojaReplace       string
}

func defaultOptions() Options {
	return Options{XGojaModuleVersion: "v0.0.0"}
}

func WriteAll(dir string, spec *buildspec.Spec, opts Options) error {
	if dir == "" {
		return fmt.Errorf("generate directory is required")
	}
	if spec == nil {
		return fmt.Errorf("spec is nil")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create generate directory %s: %w", dir, err)
	}
	files := map[string]string{
		"go.mod":         RenderGoMod(spec, opts),
		"main.go":        RenderMain(spec),
		"xgoja.gen.json": RenderEmbeddedSpec(spec),
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			return fmt.Errorf("write generated %s: %w", name, err)
		}
	}
	return nil
}
