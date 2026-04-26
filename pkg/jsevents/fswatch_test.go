package jsevents_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dop251/goja"
	gggengine "github.com/go-go-golems/go-go-goja/engine"
	"github.com/go-go-golems/go-go-goja/pkg/jsevents"
	"github.com/stretchr/testify/require"
)

func TestFSWatchHelperEmitsFileEvents(t *testing.T) {
	dir := t.TempDir()
	rt := newFSWatchRuntime(t, dir)

	_, err := rt.Owner.Call(context.Background(), "jsevents.fswatch.setup", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, err := vm.RunString(fmt.Sprintf(`
			const EventEmitter = require("events");
			globalThis.events = [];
			globalThis.errors = [];
			globalThis.watcher = new EventEmitter();
			watcher.on("event", (ev) => events.push({
				source: ev.source,
				watchPath: ev.watchPath,
				name: ev.name,
				relativeName: ev.relativeName,
				op: ev.op,
				create: ev.create,
				write: ev.write,
				remove: ev.remove,
				rename: ev.rename,
				chmod: ev.chmod,
				recursive: ev.recursive,
				debounced: ev.debounced,
				count: ev.count
			}));
			watcher.on("error", (err) => errors.push(err.message));
			globalThis.conn = fswatch.watch(%q, watcher);
		`, dir))
		return nil, err
	})
	require.NoError(t, err)

	file := filepath.Join(dir, "hello.txt")
	require.NoError(t, os.WriteFile(file, []byte("hello"), 0o644))

	require.Eventually(t, func() bool {
		got := runJS(t, rt, `JSON.stringify(globalThis.events)`)
		return strings.Contains(got, "hello.txt")
	}, 3*time.Second, 20*time.Millisecond)

	errors := runJS(t, rt, `JSON.stringify(globalThis.errors)`)
	require.JSONEq(t, `[]`, errors)

	first := runJS(t, rt, `JSON.stringify(globalThis.events[0])`)
	require.Contains(t, first, `"source":"fsnotify"`)
	require.Contains(t, first, `"watchPath":`)
	require.Contains(t, first, `"name":`)
	require.Contains(t, first, `"relativeName":"hello.txt"`)
	require.Contains(t, first, `"recursive":false`)
	require.Contains(t, first, `"debounced":false`)
	require.Contains(t, first, `"count":1`)
}

func TestFSWatchHelperCloseStopsFutureDelivery(t *testing.T) {
	dir := t.TempDir()
	rt := newFSWatchRuntime(t, dir)

	_, err := rt.Owner.Call(context.Background(), "jsevents.fswatch.close-setup", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, err := vm.RunString(fmt.Sprintf(`
			const EventEmitter = require("events");
			globalThis.events = [];
			const watcher = new EventEmitter();
			watcher.on("event", (ev) => events.push(ev));
			globalThis.conn = fswatch.watch(%q, watcher);
			globalThis.closeResult = conn.close();
		`, dir))
		return nil, err
	})
	require.NoError(t, err)
	require.Equal(t, "true", runJS(t, rt, `String(globalThis.closeResult)`))

	time.Sleep(100 * time.Millisecond)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "after-close.txt"), []byte("ignored"), 0o644))

	require.Never(t, func() bool {
		got := runJS(t, rt, `globalThis.events.length`)
		return got != "0"
	}, 300*time.Millisecond, 25*time.Millisecond)
}

func TestFSWatchHelperRejectsDisallowedPath(t *testing.T) {
	rt := newRuntime(t,
		jsevents.Install(),
		jsevents.FSWatchHelper(jsevents.FSWatchOptions{
			AllowPath: func(string) bool { return false },
		}),
	)

	_, err := rt.Owner.Call(context.Background(), "jsevents.fswatch.disallowed", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, err := vm.RunString(`
			const EventEmitter = require("events");
			fswatch.watch("/tmp", new EventEmitter());
		`)
		return nil, err
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not allowed")
}

func TestFSWatchHelperRejectsRootEscape(t *testing.T) {
	dir := t.TempDir()
	rt := newRuntime(t,
		jsevents.Install(),
		jsevents.FSWatchHelper(jsevents.FSWatchOptions{Root: dir}),
	)

	_, err := rt.Owner.Call(context.Background(), "jsevents.fswatch.root-escape", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, err := vm.RunString(`
			const EventEmitter = require("events");
			fswatch.watch("../outside", new EventEmitter());
		`)
		return nil, err
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "escapes root")
}

func TestFSWatchHelperRejectsInvalidEmitter(t *testing.T) {
	dir := t.TempDir()
	rt := newFSWatchRuntime(t, dir)

	_, err := rt.Owner.Call(context.Background(), "jsevents.fswatch.invalid-emitter", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, err := vm.RunString(fmt.Sprintf(`fswatch.watch(%q, {});`, dir))
		return nil, err
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "EventEmitter")
}

func TestFSWatchHelperRejectsUnsupportedRecursiveOption(t *testing.T) {
	dir := t.TempDir()
	rt := newFSWatchRuntime(t, dir)

	_, err := rt.Owner.Call(context.Background(), "jsevents.fswatch.recursive", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, err := vm.RunString(fmt.Sprintf(`
			const EventEmitter = require("events");
			fswatch.watch(%q, new EventEmitter(), { recursive: true });
		`, dir))
		return nil, err
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "recursive watches are not allowed")
}

func TestFSWatchHelperThrowsWhenAddFails(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing")
	rt := newFSWatchRuntime(t, dir)

	_, err := rt.Owner.Call(context.Background(), "jsevents.fswatch.add-fails", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, err := vm.RunString(fmt.Sprintf(`
			const EventEmitter = require("events");
			fswatch.watch(%q, new EventEmitter());
		`, missing))
		return nil, err
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "fswatch: stat")
}

func TestFSWatchHelperRecursiveWatchesExistingNestedDirectories(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "src", "components")
	require.NoError(t, os.MkdirAll(nested, 0o755))
	rt := newRecursiveFSWatchRuntime(t, dir)

	_, err := rt.Owner.Call(context.Background(), "jsevents.fswatch.recursive-existing", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, err := vm.RunString(fmt.Sprintf(`
			const EventEmitter = require("events");
			globalThis.events = [];
			const watcher = new EventEmitter();
			watcher.on("event", (ev) => events.push({ relativeName: ev.relativeName, recursive: ev.recursive }));
			globalThis.conn = fswatch.watch(%q, watcher, { recursive: true });
		`, dir))
		return nil, err
	})
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(filepath.Join(nested, "view.js"), []byte("export default 1"), 0o644))
	require.Eventually(t, func() bool {
		got := runJS(t, rt, `JSON.stringify(globalThis.events)`)
		return strings.Contains(got, "src/components/view.js") && strings.Contains(got, `"recursive":true`)
	}, 3*time.Second, 20*time.Millisecond)
}

func TestFSWatchHelperRecursiveAddsNewDirectories(t *testing.T) {
	dir := t.TempDir()
	rt := newRecursiveFSWatchRuntime(t, dir)

	_, err := rt.Owner.Call(context.Background(), "jsevents.fswatch.recursive-new", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, err := vm.RunString(fmt.Sprintf(`
			const EventEmitter = require("events");
			globalThis.events = [];
			const watcher = new EventEmitter();
			watcher.on("event", (ev) => events.push(ev.relativeName));
			globalThis.conn = fswatch.watch(%q, watcher, { recursive: true });
		`, dir))
		return nil, err
	})
	require.NoError(t, err)

	newDir := filepath.Join(dir, "generated")
	require.NoError(t, os.MkdirAll(newDir, 0o755))
	require.Eventually(t, func() bool {
		_ = os.WriteFile(filepath.Join(newDir, "later.js"), []byte("export default 2"), 0o644)
		got := runJS(t, rt, `JSON.stringify(globalThis.events)`)
		return strings.Contains(got, "generated/later.js")
	}, 3*time.Second, 50*time.Millisecond)
}

func TestFSWatchHelperGlobIncludeExcludeFiltersEvents(t *testing.T) {
	dir := t.TempDir()
	nodeModules := filepath.Join(dir, "node_modules")
	require.NoError(t, os.MkdirAll(nodeModules, 0o755))
	rt := newRecursiveFSWatchRuntime(t, dir)

	_, err := rt.Owner.Call(context.Background(), "jsevents.fswatch.glob", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, err := vm.RunString(fmt.Sprintf(`
			const EventEmitter = require("events");
			globalThis.events = [];
			const watcher = new EventEmitter();
			watcher.on("event", (ev) => events.push(ev.relativeName));
			globalThis.conn = fswatch.watch(%q, watcher, {
				recursive: true,
				include: ["**/*.js"],
				exclude: ["**/node_modules/**"]
			});
		`, dir))
		return nil, err
	})
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "allowed.js"), []byte("ok"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ignored.txt"), []byte("ignored"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(nodeModules, "ignored.js"), []byte("ignored"), 0o644))

	require.Eventually(t, func() bool {
		got := runJS(t, rt, `JSON.stringify(globalThis.events)`)
		return strings.Contains(got, "allowed.js")
	}, 3*time.Second, 20*time.Millisecond)
	got := runJS(t, rt, `JSON.stringify(globalThis.events)`)
	require.NotContains(t, got, "ignored.txt")
	require.NotContains(t, got, "node_modules/ignored.js")
}

func TestFSWatchHelperGlobFiltersWatchedFileByBasename(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "watched.js")
	require.NoError(t, os.WriteFile(file, []byte("initial"), 0o644))
	rt := newFSWatchRuntime(t, dir)

	_, err := rt.Owner.Call(context.Background(), "jsevents.fswatch.file-glob", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, err := vm.RunString(fmt.Sprintf(`
			const EventEmitter = require("events");
			globalThis.events = [];
			const watcher = new EventEmitter();
			watcher.on("event", (ev) => events.push(ev.relativeName));
			globalThis.conn = fswatch.watch(%q, watcher, { include: ["**/*.js"] });
		`, file))
		return nil, err
	})
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(file, []byte("updated"), 0o644))
	require.Eventually(t, func() bool {
		got := runJS(t, rt, `JSON.stringify(globalThis.events)`)
		return strings.Contains(got, "watched.js")
	}, 3*time.Second, 20*time.Millisecond)
}

func TestFSWatchHelperRejectsInvalidGlobOptions(t *testing.T) {
	dir := t.TempDir()
	rt := newRecursiveFSWatchRuntime(t, dir)

	_, err := rt.Owner.Call(context.Background(), "jsevents.fswatch.bad-glob", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, err := vm.RunString(fmt.Sprintf(`
			const EventEmitter = require("events");
			fswatch.watch(%q, new EventEmitter(), { include: ["["] });
		`, dir))
		return nil, err
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid include glob")
}

func TestFSWatchHelperDebouncesEvents(t *testing.T) {
	dir := t.TempDir()
	rt := newRuntime(t,
		jsevents.Install(),
		jsevents.FSWatchHelper(jsevents.FSWatchOptions{MaxDebounce: time.Second}),
	)

	_, err := rt.Owner.Call(context.Background(), "jsevents.fswatch.debounce", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, err := vm.RunString(fmt.Sprintf(`
			const EventEmitter = require("events");
			globalThis.events = [];
			const watcher = new EventEmitter();
			watcher.on("event", (ev) => events.push({ relativeName: ev.relativeName, debounced: ev.debounced, count: ev.count }));
			globalThis.conn = fswatch.watch(%q, watcher, { debounceMs: 100 });
		`, dir))
		return nil, err
	})
	require.NoError(t, err)

	file := filepath.Join(dir, "debounced.txt")
	for i := 0; i < 5; i++ {
		require.NoError(t, os.WriteFile(file, []byte(fmt.Sprintf("%d", i)), 0o644))
	}

	require.Eventually(t, func() bool {
		got := runJS(t, rt, `JSON.stringify(globalThis.events)`)
		return strings.Contains(got, "debounced.txt") && strings.Contains(got, `"debounced":true`)
	}, 3*time.Second, 20*time.Millisecond)

	countAfterFlush := runJS(t, rt, `globalThis.events.filter(ev => ev.relativeName === "debounced.txt").length`)
	time.Sleep(200 * time.Millisecond)
	countAfterStability := runJS(t, rt, `globalThis.events.filter(ev => ev.relativeName === "debounced.txt").length`)
	require.Equal(t, countAfterFlush, countAfterStability)
}

func TestFSWatchHelperCloseStopsPendingDebounce(t *testing.T) {
	dir := t.TempDir()
	rt := newRuntime(t,
		jsevents.Install(),
		jsevents.FSWatchHelper(jsevents.FSWatchOptions{MaxDebounce: time.Second}),
	)

	_, err := rt.Owner.Call(context.Background(), "jsevents.fswatch.debounce-close", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, err := vm.RunString(fmt.Sprintf(`
			const EventEmitter = require("events");
			globalThis.events = [];
			const watcher = new EventEmitter();
			watcher.on("event", (ev) => events.push(ev.relativeName));
			globalThis.conn = fswatch.watch(%q, watcher, { debounceMs: 500 });
		`, dir))
		return nil, err
	})
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "pending.txt"), []byte("pending"), 0o644))
	time.Sleep(50 * time.Millisecond)
	_ = runJS(t, rt, `String(globalThis.conn.close())`)
	time.Sleep(650 * time.Millisecond)
	require.Equal(t, "[]", runJS(t, rt, `JSON.stringify(globalThis.events)`))
}

func TestFSWatchHelperRejectsDebounceAboveHostLimit(t *testing.T) {
	dir := t.TempDir()
	rt := newRuntime(t,
		jsevents.Install(),
		jsevents.FSWatchHelper(jsevents.FSWatchOptions{MaxDebounce: 25 * time.Millisecond}),
	)

	_, err := rt.Owner.Call(context.Background(), "jsevents.fswatch.debounce-limit", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, err := vm.RunString(fmt.Sprintf(`
			const EventEmitter = require("events");
			fswatch.watch(%q, new EventEmitter(), { debounceMs: 100 });
		`, dir))
		return nil, err
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "exceeds maximum")
}

func TestFSWatchHelperRequiresManager(t *testing.T) {
	factory, err := gggengine.NewBuilder().
		WithRuntimeInitializers(jsevents.FSWatchHelper(jsevents.FSWatchOptions{})).
		Build()
	require.NoError(t, err)
	_, err = factory.NewRuntime(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "manager is not installed")
}

func newFSWatchRuntime(t *testing.T, allowedRoot string) *gggengine.Runtime {
	t.Helper()
	return newFSWatchRuntimeWithOptions(t, allowedRoot, jsevents.FSWatchOptions{})
}

func newRecursiveFSWatchRuntime(t *testing.T, allowedRoot string) *gggengine.Runtime {
	t.Helper()
	return newFSWatchRuntimeWithOptions(t, allowedRoot, jsevents.FSWatchOptions{
		AllowRecursive: true,
		MaxDebounce:    time.Second,
	})
}

func newFSWatchRuntimeWithOptions(t *testing.T, allowedRoot string, opts jsevents.FSWatchOptions) *gggengine.Runtime {
	t.Helper()
	allowedRoot, err := filepath.Abs(allowedRoot)
	require.NoError(t, err)
	opts.AllowPath = func(path string) bool {
		path, err := filepath.Abs(path)
		if err != nil {
			return false
		}
		rel, err := filepath.Rel(allowedRoot, path)
		return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
	}
	return newRuntime(t,
		jsevents.Install(),
		jsevents.FSWatchHelper(opts),
	)
}
