package jsevents

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/dop251/goja"
	"github.com/fsnotify/fsnotify"
	"github.com/go-go-golems/go-go-goja/engine"
)

// FSWatchOptions configures the opt-in JavaScript helper installed by
// FSWatchHelper.
type FSWatchOptions struct {
	// GlobalName is the JavaScript global object name. Defaults to "fswatch".
	GlobalName string
	// Root optionally restricts watched paths to a subtree. Relative JavaScript
	// paths are resolved against this root. Absolute paths must still remain
	// inside the root.
	Root string
	// AllowPath decides whether the normalized path may be watched. If nil, all
	// normalized paths are allowed.
	AllowPath func(path string) bool
}

type fsWatchHelper struct {
	opts FSWatchOptions
}

// FSWatchHelper installs a JS-callable helper object with watch(path, emitter,
// options?). It does not create any filesystem watchers until JavaScript calls
// watch.
func FSWatchHelper(opts FSWatchOptions) engine.RuntimeInitializer {
	return &fsWatchHelper{opts: opts}
}

func (h *fsWatchHelper) ID() string { return "jsevents.fswatch-helper" }

func (h *fsWatchHelper) InitRuntime(ctx *engine.RuntimeContext) error {
	if ctx == nil || ctx.VM == nil {
		return fmt.Errorf("jsevents fswatch: incomplete runtime context")
	}
	managerValue, ok := ctx.Value(RuntimeValueKey)
	if !ok {
		return fmt.Errorf("jsevents fswatch: manager is not installed; add jsevents.Install() before FSWatchHelper")
	}
	manager, ok := managerValue.(*Manager)
	if !ok || manager == nil {
		return fmt.Errorf("jsevents fswatch: invalid manager value")
	}

	globalName := h.opts.GlobalName
	if globalName == "" {
		globalName = "fswatch"
	}

	obj := ctx.VM.NewObject()
	if err := obj.Set("watch", func(call goja.FunctionCall) goja.Value {
		path, err := normalizeWatchPath(call.Argument(0), h.opts)
		if err != nil {
			panic(ctx.VM.NewGoError(err))
		}
		if err := validateFSWatchOptions(ctx.VM, call.Argument(2)); err != nil {
			panic(ctx.VM.NewGoError(err))
		}

		ref, err := manager.AdoptEmitterOnOwner(call.Argument(1))
		if err != nil {
			panic(ctx.VM.NewGoError(err))
		}

		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			_ = ref.Close(context.Background())
			panic(ctx.VM.NewGoError(fmt.Errorf("fswatch: create watcher: %w", err)))
		}
		if err := watcher.Add(path); err != nil {
			_ = watcher.Close()
			_ = ref.Close(context.Background())
			panic(ctx.VM.NewGoError(fmt.Errorf("fswatch: watch %q: %w", path, err)))
		}

		watchCtx, cancel := context.WithCancel(ctx.Context)
		ref.SetCancel(cancel)
		go runFSWatcher(watchCtx, ref, watcher, path)

		return fsWatchConnectionObject(ctx.VM, ref, path)
	}); err != nil {
		return err
	}

	return ctx.VM.Set(globalName, obj)
}

func normalizeWatchPath(value goja.Value, opts FSWatchOptions) (string, error) {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return "", fmt.Errorf("fswatch: path is required")
	}
	raw := strings.TrimSpace(value.String())
	if raw == "" {
		return "", fmt.Errorf("fswatch: path is empty")
	}

	if opts.Root != "" {
		root, err := filepath.Abs(opts.Root)
		if err != nil {
			return "", fmt.Errorf("fswatch: resolve root: %w", err)
		}
		root = filepath.Clean(root)

		candidate := raw
		if filepath.IsAbs(candidate) {
			candidate = filepath.Clean(candidate)
		} else {
			candidate = filepath.Join(root, candidate)
		}
		candidate, err = filepath.Abs(candidate)
		if err != nil {
			return "", fmt.Errorf("fswatch: resolve path: %w", err)
		}
		candidate = filepath.Clean(candidate)

		rel, err := filepath.Rel(root, candidate)
		if err != nil {
			return "", fmt.Errorf("fswatch: compare path to root: %w", err)
		}
		if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("fswatch: path %q escapes root %q", raw, root)
		}
		raw = candidate
	} else {
		raw = filepath.Clean(raw)
	}

	if opts.AllowPath != nil && !opts.AllowPath(raw) {
		return "", fmt.Errorf("fswatch: path %q is not allowed", raw)
	}
	return raw, nil
}

func validateFSWatchOptions(vm *goja.Runtime, value goja.Value) error {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return nil
	}
	obj := value.ToObject(vm)
	recursive := obj.Get("recursive")
	if recursive != nil && !goja.IsUndefined(recursive) && !goja.IsNull(recursive) && recursive.ToBoolean() {
		return fmt.Errorf("fswatch: recursive watches are not supported")
	}
	return nil
}

func fsWatchConnectionObject(vm *goja.Runtime, ref *EmitterRef, path string) *goja.Object {
	obj := vm.NewObject()
	_ = obj.Set("id", ref.ID())
	_ = obj.Set("path", path)
	_ = obj.Set("close", func() bool {
		return ref.Close(context.Background()) == nil
	})
	return obj
}

func runFSWatcher(ctx context.Context, ref *EmitterRef, watcher *fsnotify.Watcher, path string) {
	defer func() { _ = watcher.Close() }()

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-watcher.Events:
			if !ok {
				_ = ref.Emit(context.Background(), "close")
				_ = ref.Close(context.Background())
				return
			}
			_ = ref.Emit(ctx, "event", fsWatchEventPayload(path, event))
		case err, ok := <-watcher.Errors:
			if !ok {
				_ = ref.Emit(context.Background(), "close")
				_ = ref.Close(context.Background())
				return
			}
			_ = emitFSWatchError(ctx, ref, path, err)
		}
	}
}

func fsWatchEventPayload(watchPath string, event fsnotify.Event) map[string]any {
	return map[string]any{
		"source":    "fsnotify",
		"watchPath": watchPath,
		"name":      event.Name,
		"op":        event.Op.String(),
		"create":    event.Has(fsnotify.Create),
		"write":     event.Has(fsnotify.Write),
		"remove":    event.Has(fsnotify.Remove),
		"rename":    event.Has(fsnotify.Rename),
		"chmod":     event.Has(fsnotify.Chmod),
	}
}

func emitFSWatchError(ctx context.Context, ref *EmitterRef, path string, err error) error {
	return ref.Emit(ctx, "error", map[string]any{
		"source":  "fsnotify",
		"path":    path,
		"message": err.Error(),
	})
}
