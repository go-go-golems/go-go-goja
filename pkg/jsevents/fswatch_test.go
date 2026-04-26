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
				op: ev.op,
				create: ev.create,
				write: ev.write,
				remove: ev.remove,
				rename: ev.rename,
				chmod: ev.chmod
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
	require.Contains(t, err.Error(), "recursive watches are not supported")
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
	require.Contains(t, err.Error(), "fswatch: watch")
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
	allowedRoot, err := filepath.Abs(allowedRoot)
	require.NoError(t, err)
	return newRuntime(t,
		jsevents.Install(),
		jsevents.FSWatchHelper(jsevents.FSWatchOptions{
			AllowPath: func(path string) bool {
				path, err := filepath.Abs(path)
				if err != nil {
					return false
				}
				rel, err := filepath.Rel(allowedRoot, path)
				return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
			},
		}),
	)
}
