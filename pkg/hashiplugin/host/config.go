package host

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/go-hclog"
)

// Config controls plugin discovery and runtime integration.
type Config struct {
	Directories  []string
	Pattern      string
	Namespace    string
	AllowModules []string
	StartTimeout time.Duration
	CallTimeout  time.Duration
	AutoMTLS     bool
	Logger       hclog.Logger
	Report       *ReportCollector
}

// DefaultDiscoveryRoot returns the conventional per-user plugin root.
func DefaultDiscoveryRoot() string {
	homeDir, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(homeDir) == "" {
		return ""
	}
	return filepath.Join(homeDir, ".go-go-goja", "plugins")
}

// DefaultDiscoveryDirectories returns the existing directories under the default plugin root.
func DefaultDiscoveryDirectories() []string {
	root := DefaultDiscoveryRoot()
	if root == "" {
		return nil
	}
	return discoveryDirectoriesUnderRoot(root)
}

// ResolveDiscoveryDirectories returns explicit directories when provided, otherwise the default plugin tree.
func ResolveDiscoveryDirectories(directories []string) []string {
	normalized := normalizeDirectories(directories)
	if len(normalized) > 0 {
		return normalized
	}
	return DefaultDiscoveryDirectories()
}

func (c Config) withDefaults() Config {
	c.AllowModules = normalizeModuleNames(c.AllowModules)
	if strings.TrimSpace(c.Pattern) == "" {
		c.Pattern = "goja-plugin-*"
	}
	if strings.TrimSpace(c.Namespace) == "" {
		c.Namespace = "plugin:"
	}
	if c.StartTimeout <= 0 {
		c.StartTimeout = 10 * time.Second
	}
	if c.CallTimeout <= 0 {
		c.CallTimeout = 5 * time.Second
	}
	if c.Logger == nil {
		c.Logger = hclog.NewNullLogger()
	}
	return c
}

func discoveryDirectoriesUnderRoot(root string) []string {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return nil
	}

	out := make([]string, 0, 8)
	seen := map[string]struct{}{}
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		path = filepath.Clean(path)
		if _, ok := seen[path]; ok {
			return nil
		}
		seen[path] = struct{}{}
		out = append(out, path)
		return nil
	})
	if err != nil {
		return nil
	}
	sort.Strings(out)
	return out
}

func normalizeDirectories(directories []string) []string {
	if len(directories) == 0 {
		return nil
	}
	out := make([]string, 0, len(directories))
	seen := map[string]struct{}{}
	for _, dir := range directories {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		dir = expandHomeDir(dir)
		dir = filepath.Clean(dir)
		if _, ok := seen[dir]; ok {
			continue
		}
		seen[dir] = struct{}{}
		out = append(out, dir)
	}
	sort.Strings(out)
	return out
}

func normalizeModuleNames(names []string) []string {
	if len(names) == 0 {
		return nil
	}
	out := make([]string, 0, len(names))
	seen := map[string]struct{}{}
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func expandHomeDir(dir string) string {
	if dir == "~" {
		if homeDir, err := os.UserHomeDir(); err == nil {
			return homeDir
		}
		return dir
	}
	prefix := "~" + string(os.PathSeparator)
	if strings.HasPrefix(dir, prefix) {
		if homeDir, err := os.UserHomeDir(); err == nil && strings.TrimSpace(homeDir) != "" {
			return filepath.Join(homeDir, dir[len(prefix):])
		}
	}
	return dir
}
