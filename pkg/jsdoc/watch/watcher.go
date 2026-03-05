// Package watch monitors a directory for JS file changes and triggers callbacks.
package watch

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
)

// Event represents a file change event.
type Event struct {
	Path string
	Op   string // "create", "write", "remove", "rename"
}

// Watcher monitors a directory for JS file changes.
type Watcher struct {
	dir      string
	onChange func(Event)
	done     chan struct{}
}

// New creates a new Watcher for the given directory.
func New(dir string, onChange func(Event)) *Watcher {
	return &Watcher{
		dir:      dir,
		onChange: onChange,
		done:     make(chan struct{}),
	}
}

// Start begins watching the directory. Blocks until Stop() is called.
func (w *Watcher) Start() error {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return errors.Wrap(err, "creating watcher")
	}
	defer func() { _ = fw.Close() }()

	// Watch the directory itself.
	if err := fw.Add(w.dir); err != nil {
		return errors.Wrapf(err, "watching %s", w.dir)
	}

	// Also watch any subdirectories.
	_ = filepath.Walk(w.dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() {
			return nil
		}
		_ = fw.Add(path)
		return nil
	})

	// Debounce rapid successive events for the same file.
	debounce := make(map[string]*time.Timer)

	for {
		select {
		case <-w.done:
			return nil

		case event, ok := <-fw.Events:
			if !ok {
				return nil
			}
			if !strings.HasSuffix(event.Name, ".js") {
				continue
			}

			path := event.Name
			op := fsnotifyOpToString(event.Op)

			// Cancel existing debounce timer for this path.
			if t, exists := debounce[path]; exists {
				t.Stop()
			}
			// Debounce 150ms.
			debounce[path] = time.AfterFunc(150*time.Millisecond, func() {
				w.onChange(Event{Path: path, Op: op})
			})

		case err, ok := <-fw.Errors:
			if !ok {
				return nil
			}
			fmt.Fprintf(os.Stderr, "watcher error: %v\n", err)
		}
	}
}

// Stop shuts down the watcher.
func (w *Watcher) Stop() {
	close(w.done)
}

func fsnotifyOpToString(op fsnotify.Op) string {
	switch {
	case op&fsnotify.Create != 0:
		return "create"
	case op&fsnotify.Write != 0:
		return "write"
	case op&fsnotify.Remove != 0:
		return "remove"
	case op&fsnotify.Rename != 0:
		return "rename"
	default:
		return "unknown"
	}
}
