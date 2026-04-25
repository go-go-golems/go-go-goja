package fs_test

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/dop251/goja"
	gggengine "github.com/go-go-golems/go-go-goja/engine"
)

func TestAsyncFsSmoke(t *testing.T) {
	dir := t.TempDir()
	rt := newRuntime(t)
	quotedDir := strconv.Quote(filepath.ToSlash(dir))

	_, err := rt.Owner.Call(context.Background(), "fs.async.setup", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, runErr := vm.RunString(`
			globalThis.__fsSmoke = { done: false, error: "", value: "" };
			(async () => {
				const fs = require("fs");
				const root = ` + quotedDir + `;
				const sub = root + "/sub";
				await fs.mkdir(sub, { recursive: true });
				await fs.writeFile(sub + "/a.txt", "hello");
				await fs.appendFile(sub + "/a.txt", " world");
				const text = await fs.readFile(sub + "/a.txt", "utf8");
				const exists = await fs.exists(sub + "/a.txt");
				const entries = await fs.readdir(sub);
				const stat = await fs.stat(sub + "/a.txt");
				await fs.copyFile(sub + "/a.txt", sub + "/b.txt");
				await fs.rename(sub + "/b.txt", sub + "/c.txt");
				await fs.unlink(sub + "/c.txt");
				globalThis.__fsSmoke = {
					done: true,
					error: "",
					text,
					exists,
					entries,
					isFile: stat.isFile,
					copyRemoved: !(await fs.exists(sub + "/c.txt"))
				};
			})().catch(e => {
				globalThis.__fsSmoke = { done: true, error: String(e), value: "" };
			});
		`)
		return nil, runErr
	})
	if err != nil {
		t.Fatalf("setup async smoke: %v", err)
	}

	requireEventuallyState(t, rt, func(raw string) bool {
		return strings.Contains(raw, `"done":true`)
	})
	state := readState(t, rt)
	if strings.Contains(state, `"error":""`) == false {
		t.Fatalf("async fs failed: %s", state)
	}
	for _, want := range []string{`"text":"hello world"`, `"exists":true`, `"a.txt"`, `"isFile":true`, `"copyRemoved":true`} {
		if !strings.Contains(state, want) {
			t.Fatalf("async state missing %s: %s", want, state)
		}
	}
	content, err := os.ReadFile(filepath.Join(dir, "sub", "a.txt"))
	if err != nil {
		t.Fatalf("read final async file: %v", err)
	}
	if string(content) != "hello world" {
		t.Fatalf("final async content = %q", string(content))
	}
}

func TestAsyncFsBufferSmoke(t *testing.T) {
	dir := t.TempDir()
	rt := newRuntime(t)
	quotedDir := strconv.Quote(filepath.ToSlash(dir))

	_, err := rt.Owner.Call(context.Background(), "fs.async-buffer.setup", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, runErr := vm.RunString(`
			globalThis.__fsSmoke = { done: false, error: "" };
			(async () => {
				const fs = require("fs");
				const root = ` + quotedDir + `;
				const p = root + "/buffer.bin";
				await fs.writeFile(p, Buffer.from([65, 66, 67]));
				await fs.appendFile(p, Buffer.from([68]));
				const b = await fs.readFile(p);
				const s = await fs.readFile(p, { encoding: "utf8" });
				globalThis.__fsSmoke = {
					done: true,
					error: "",
					bufferLength: b.length,
					bufferText: b.toString(),
					encodedText: s
				};
			})().catch(e => {
				globalThis.__fsSmoke = { done: true, error: String(e) };
			});
		`)
		return nil, runErr
	})
	if err != nil {
		t.Fatalf("setup async buffer smoke: %v", err)
	}

	requireEventuallyState(t, rt, func(raw string) bool {
		return strings.Contains(raw, `"done":true`)
	})
	state := readState(t, rt)
	for _, want := range []string{`"error":""`, `"bufferLength":4`, `"bufferText":"ABCD"`, `"encodedText":"ABCD"`} {
		if !strings.Contains(state, want) {
			t.Fatalf("async buffer state missing %s: %s", want, state)
		}
	}
}

func TestSyncFsSmoke(t *testing.T) {
	dir := t.TempDir()
	rt := newRuntime(t)
	quotedDir := strconv.Quote(filepath.ToSlash(dir))

	ret, err := rt.Owner.Call(context.Background(), "fs.sync", func(_ context.Context, vm *goja.Runtime) (any, error) {
		value, runErr := vm.RunString(`
			const fs = require("fs");
			const root = ` + quotedDir + `;
			const sub = root + "/sync";
			fs.mkdirSync(sub, { recursive: true });
			fs.writeFileSync(sub + "/a.txt", "sync");
			fs.appendFileSync(sub + "/a.txt", " data");
			const text = fs.readFileSync(sub + "/a.txt", "utf8");
			const entries = fs.readdirSync(sub);
			const stat = fs.statSync(sub + "/a.txt");
			fs.copyFileSync(sub + "/a.txt", sub + "/b.txt");
			fs.renameSync(sub + "/b.txt", sub + "/c.txt");
			fs.unlinkSync(sub + "/c.txt");
			JSON.stringify({ text, exists: fs.existsSync(sub + "/a.txt"), entries, isFile: stat.isFile, removed: !fs.existsSync(sub + "/c.txt") });
		`)
		if runErr != nil {
			return nil, runErr
		}
		return value.String(), nil
	})
	if err != nil {
		t.Fatalf("run sync smoke: %v", err)
	}
	state, ok := ret.(string)
	if !ok {
		t.Fatalf("sync smoke result type %T", ret)
	}
	for _, want := range []string{`"text":"sync data"`, `"exists":true`, `"a.txt"`, `"isFile":true`, `"removed":true`} {
		if !strings.Contains(state, want) {
			t.Fatalf("sync state missing %s: %s", want, state)
		}
	}
}

func TestSyncFsBufferSmoke(t *testing.T) {
	dir := t.TempDir()
	rt := newRuntime(t)
	quotedDir := strconv.Quote(filepath.ToSlash(dir))

	ret, err := rt.Owner.Call(context.Background(), "fs.sync-buffer", func(_ context.Context, vm *goja.Runtime) (any, error) {
		value, runErr := vm.RunString(`
			const fs = require("fs");
			const root = ` + quotedDir + `;
			const p = root + "/sync-buffer.bin";
			fs.writeFileSync(p, Buffer.from([88, 89]));
			fs.appendFileSync(p, Buffer.from([90]));
			const b = fs.readFileSync(p);
			const s = fs.readFileSync(p, "utf8");
			JSON.stringify({ bufferLength: b.length, bufferText: b.toString(), encodedText: s });
		`)
		if runErr != nil {
			return nil, runErr
		}
		return value.String(), nil
	})
	if err != nil {
		t.Fatalf("run sync buffer smoke: %v", err)
	}
	state, ok := ret.(string)
	if !ok {
		t.Fatalf("sync buffer result type %T", ret)
	}
	for _, want := range []string{`"bufferLength":3`, `"bufferText":"XYZ"`, `"encodedText":"XYZ"`} {
		if !strings.Contains(state, want) {
			t.Fatalf("sync buffer state missing %s: %s", want, state)
		}
	}
}

func newRuntime(t *testing.T) *gggengine.Runtime {
	t.Helper()
	factory, err := gggengine.NewBuilder().WithModules(gggengine.DefaultRegistryModules()).Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}
	rt, err := factory.NewRuntime(context.Background())
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	t.Cleanup(func() { _ = rt.Close(context.Background()) })
	return rt
}

func requireEventuallyState(t *testing.T, rt *gggengine.Runtime, pred func(string) bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		state := readState(t, rt)
		if pred(state) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("condition not met; state=%s", readState(t, rt))
}

func readState(t *testing.T, rt *gggengine.Runtime) string {
	t.Helper()
	ret, err := rt.Owner.Call(context.Background(), "fs.state", func(_ context.Context, vm *goja.Runtime) (any, error) {
		value, runErr := vm.RunString(`JSON.stringify(globalThis.__fsSmoke || { done: false, error: "", value: "" })`)
		if runErr != nil {
			return nil, runErr
		}
		return value.String(), nil
	})
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	state, ok := ret.(string)
	if !ok {
		t.Fatalf("state type %T", ret)
	}
	return state
}
