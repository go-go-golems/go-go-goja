package engine

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/dop251/goja_nodejs/require"
)

// ModuleRootsOptions controls how module roots are derived from a script path.
type ModuleRootsOptions struct {
	IncludeScriptDir   bool
	IncludeParentDir   bool
	IncludeNodeModules bool
	ExtraFolders       []string
}

// DefaultModuleRootsOptions returns the standard layered roots used by script
// runners in this repository.
func DefaultModuleRootsOptions() ModuleRootsOptions {
	return ModuleRootsOptions{
		IncludeScriptDir:   true,
		IncludeParentDir:   true,
		IncludeNodeModules: true,
	}
}

// ResolveModuleRootsFromScript derives global module folders from script path
// conventions and returns a deduplicated ordered list.
func ResolveModuleRootsFromScript(scriptPath string, opts ModuleRootsOptions) ([]string, error) {
	if strings.TrimSpace(scriptPath) == "" {
		return nil, fmt.Errorf("script path is empty")
	}

	absScript, err := filepath.Abs(scriptPath)
	if err != nil {
		return nil, fmt.Errorf("resolve absolute script path: %w", err)
	}
	scriptDir := filepath.Dir(absScript)
	parentDir := filepath.Dir(scriptDir)

	candidates := make([]string, 0, 6+len(opts.ExtraFolders))
	if opts.IncludeScriptDir {
		candidates = append(candidates, scriptDir)
	}
	if opts.IncludeParentDir {
		candidates = append(candidates, parentDir)
	}
	if opts.IncludeNodeModules {
		if opts.IncludeScriptDir {
			candidates = append(candidates, filepath.Join(scriptDir, "node_modules"))
		}
		if opts.IncludeParentDir {
			candidates = append(candidates, filepath.Join(parentDir, "node_modules"))
		}
	}
	candidates = append(candidates, opts.ExtraFolders...)

	out := make([]string, 0, len(candidates))
	seen := map[string]struct{}{}
	for _, folder := range candidates {
		folder = strings.TrimSpace(folder)
		if folder == "" {
			continue
		}
		cleaned := filepath.Clean(folder)
		if !filepath.IsAbs(cleaned) {
			abs, err := filepath.Abs(cleaned)
			if err != nil {
				return nil, fmt.Errorf("resolve absolute module root %q: %w", cleaned, err)
			}
			cleaned = abs
		}
		if _, ok := seen[cleaned]; ok {
			continue
		}
		seen[cleaned] = struct{}{}
		out = append(out, cleaned)
	}

	return out, nil
}

// RequireOptionWithModuleRootsFromScript converts resolved module roots into a
// require option for WithGlobalFolders.
func RequireOptionWithModuleRootsFromScript(scriptPath string, opts ModuleRootsOptions) (require.Option, error) {
	roots, err := ResolveModuleRootsFromScript(scriptPath, opts)
	if err != nil {
		return nil, err
	}
	if len(roots) == 0 {
		return nil, nil
	}
	return require.WithGlobalFolders(roots...), nil
}

// WithModuleRootsFromScript appends WithGlobalFolders() based on script path.
// Invalid paths are ignored at builder construction time; callers that need
// strict error handling should use ResolveModuleRootsFromScript directly.
func WithModuleRootsFromScript(scriptPath string, opts ModuleRootsOptions) Option {
	return func(s *builderSettings) {
		requireOpt, err := RequireOptionWithModuleRootsFromScript(scriptPath, opts)
		if err != nil || requireOpt == nil {
			return
		}
		s.requireOptions = append(s.requireOptions, requireOpt)
	}
}
