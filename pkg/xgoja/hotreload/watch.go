package hotreload

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var defaultIgnoredDirs = map[string]struct{}{
	".bin":         {},
	".git":         {},
	"dist":         {},
	"node_modules": {},
}

type WatchOptions struct {
	Roots        []string
	Extensions   []string
	IgnoreDirs   []string
	PollInterval time.Duration
	Debounce     time.Duration
	OnReload     func(*Snapshot)
	OnError      func(error)
}

type fileState struct {
	ModTime time.Time
	Size    int64
}

func (m *Manager) Watch(ctx context.Context, opts WatchOptions) error {
	if m == nil {
		return fmt.Errorf("hotreload manager is nil")
	}
	if len(opts.Roots) == 0 {
		return fmt.Errorf("watch roots are required")
	}
	if opts.PollInterval <= 0 {
		opts.PollInterval = 500 * time.Millisecond
	}
	if opts.Debounce < 0 {
		return fmt.Errorf("watch debounce must not be negative")
	}

	state, err := scanWatchRoots(opts)
	if err != nil {
		return err
	}
	ticker := time.NewTicker(opts.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			next, err := scanWatchRoots(opts)
			if err != nil {
				reportWatchError(opts, err)
				continue
			}
			if !watchStateChanged(state, next) {
				continue
			}
			state = next
			if opts.Debounce > 0 {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(opts.Debounce):
				}
			}
			snapshot, err := m.Reload(ctx)
			if err != nil {
				reportWatchError(opts, err)
				continue
			}
			if opts.OnReload != nil {
				opts.OnReload(snapshot)
			}
		}
	}
}

func scanWatchRoots(opts WatchOptions) (map[string]fileState, error) {
	state := map[string]fileState{}
	extensions := normalizedExtensions(opts.Extensions)
	ignoredDirs := normalizedIgnoredDirs(opts.IgnoreDirs)
	for _, root := range opts.Roots {
		root = strings.TrimSpace(root)
		if root == "" {
			return nil, fmt.Errorf("watch root is empty")
		}
		if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if _, ok := ignoredDirs[d.Name()]; ok && path != root {
					return filepath.SkipDir
				}
				return nil
			}
			if !d.Type().IsRegular() {
				return nil
			}
			if len(extensions) > 0 {
				if _, ok := extensions[strings.ToLower(filepath.Ext(path))]; !ok {
					return nil
				}
			}
			info, err := d.Info()
			if err != nil {
				return err
			}
			abs, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			state[abs] = fileState{ModTime: info.ModTime(), Size: info.Size()}
			return nil
		}); err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("watch root %q does not exist", root)
			}
			return nil, err
		}
	}
	return state, nil
}

func normalizedExtensions(values []string) map[string]struct{} {
	if len(values) == 0 {
		return nil
	}
	out := map[string]struct{}{}
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		if !strings.HasPrefix(value, ".") {
			value = "." + value
		}
		out[value] = struct{}{}
	}
	return out
}

func normalizedIgnoredDirs(values []string) map[string]struct{} {
	out := map[string]struct{}{}
	if values == nil {
		for value := range defaultIgnoredDirs {
			out[value] = struct{}{}
		}
		return out
	}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out[value] = struct{}{}
		}
	}
	return out
}

func watchStateChanged(a, b map[string]fileState) bool {
	if len(a) != len(b) {
		return true
	}
	for path, av := range a {
		bv, ok := b[path]
		if !ok {
			return true
		}
		if av.Size != bv.Size || !av.ModTime.Equal(bv.ModTime) {
			return true
		}
	}
	return false
}

func reportWatchError(opts WatchOptions, err error) {
	if opts.OnError != nil && err != nil {
		opts.OnError(err)
	}
}
