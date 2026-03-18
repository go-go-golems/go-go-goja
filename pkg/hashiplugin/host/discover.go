package host

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hashicorp/go-plugin"
)

// Discover returns candidate plugin binaries after basic path/executable checks.
func Discover(cfg Config) ([]string, error) {
	cfg = cfg.withDefaults()
	if len(cfg.Directories) == 0 {
		return nil, nil
	}

	seen := map[string]struct{}{}
	out := make([]string, 0)
	for _, dir := range cfg.Directories {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		absDir, err := filepath.Abs(dir)
		if err != nil {
			return nil, fmt.Errorf("resolve plugin directory %q: %w", dir, err)
		}
		paths, err := plugin.Discover(cfg.Pattern, absDir)
		if err != nil {
			return nil, fmt.Errorf("discover plugins in %q: %w", absDir, err)
		}
		for _, path := range paths {
			if err := validateDiscoveredBinary(path); err != nil {
				return nil, err
			}
			if _, ok := seen[path]; ok {
				continue
			}
			seen[path] = struct{}{}
			out = append(out, path)
		}
	}

	sort.Strings(out)
	return out, nil
}

func validateDiscoveredBinary(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat plugin candidate %q: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("plugin candidate %q is not a regular file", path)
	}
	if info.Mode().Perm()&0o111 == 0 {
		return fmt.Errorf("plugin candidate %q is not executable", path)
	}
	return nil
}
