package fs_test

import (
	"context"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/dop251/goja"
	gggengine "github.com/go-go-golems/go-go-goja/engine"
	fsmod "github.com/go-go-golems/go-go-goja/modules/fs"
)

func TestReadOnlyEmbeddedFsSync(t *testing.T) {
	rt := newEmbeddedRuntime(t)
	ret, err := rt.Owner.Call(context.Background(), "fs.embedded.sync", func(_ context.Context, vm *goja.Runtime) (any, error) {
		value, runErr := vm.RunString(`
			const fs = require("fs:assets");
			let writeCode = "";
			let writePath = "";
			let missingCode = "";
			try { fs.writeFileSync("/app/config/default.json", "nope", "utf8"); } catch (e) { writeCode = e.code; writePath = e.path; }
			try { fs.readFileSync("/app/missing.txt", "utf8"); } catch (e) { missingCode = e.code + ":" + e.syscall; }
			const text = fs.readFileSync("/app/config/default.json", "utf8");
			const entries = fs.readdirSync("/app/config");
			const stat = fs.statSync("/app/config/default.json");
			JSON.stringify({ text, entries, exists: fs.existsSync("/app/config/default.json"), missing: fs.existsSync("/app/missing.txt"), isFile: stat.isFile, writeCode, writePath, missingCode });
		`)
		if runErr != nil {
			return nil, runErr
		}
		return value.String(), nil
	})
	if err != nil {
		t.Fatalf("run embedded sync: %v", err)
	}
	state := ret.(string)
	for _, want := range []string{`"text":"{\"ok\":true}"`, `"default.json"`, `"exists":true`, `"missing":false`, `"isFile":true`, `"writeCode":"EROFS"`, `"writePath":"/app/config/default.json"`, `"missingCode":"ENOENT:open"`} {
		if !strings.Contains(state, want) {
			t.Fatalf("embedded sync state missing %s: %s", want, state)
		}
	}
}

func TestReadOnlyEmbeddedFsAsync(t *testing.T) {
	rt := newEmbeddedRuntime(t)
	_, err := rt.Owner.Call(context.Background(), "fs.embedded.async.setup", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, runErr := vm.RunString(`
			globalThis.__fsSmoke = { done: false, error: "" };
			(async () => {
				const fs = require("fs:assets");
				let writeCode = "";
				try { await fs.writeFile("/app/config/default.json", "nope", "utf8"); } catch (e) { writeCode = e.code; }
				const text = await fs.readFile("/app/config/default.json", "utf8");
				const entries = await fs.readdir("/app/config");
				const stat = await fs.stat("/app/config/default.json");
				globalThis.__fsSmoke = { done: true, error: "", text, entries, isFile: stat.isFile, writeCode };
			})().catch(e => { globalThis.__fsSmoke = { done: true, error: String(e) }; });
		`)
		return nil, runErr
	})
	if err != nil {
		t.Fatalf("setup embedded async: %v", err)
	}
	requireEventuallyState(t, rt, func(raw string) bool { return strings.Contains(raw, `"done":true`) })
	state := readState(t, rt)
	for _, want := range []string{`"error":""`, `"text":"{\"ok\":true}"`, `"default.json"`, `"isFile":true`, `"writeCode":"EROFS"`} {
		if !strings.Contains(state, want) {
			t.Fatalf("embedded async state missing %s: %s", want, state)
		}
	}
}

func newEmbeddedRuntime(t *testing.T) *gggengine.Runtime {
	t.Helper()
	assetFS := fstest.MapFS{
		"xgoja_embed/assets/app/config/default.json": &fstest.MapFile{Data: []byte(`{"ok":true}`)},
	}
	mod := fsmod.New(
		fsmod.WithName("fs:assets"),
		fsmod.WithBackend(fsmod.NewReadOnlyFSBackend(fsmod.FSMount{FS: assetFS, Root: "xgoja_embed/assets/app", Mount: "/app"})),
	)
	factory, err := gggengine.NewBuilder(
		gggengine.WithImplicitDefaultRegistryModules(false),
		gggengine.WithDataOnlyDefaultRegistryModules(false),
	).WithModules(gggengine.NativeModuleSpec{ModuleName: "fs:assets", Loader: mod.Loader}).Build()
	if err != nil {
		t.Fatalf("build embedded factory: %v", err)
	}
	rt, err := factory.NewRuntime(gggengine.WithStartupContext(context.Background()), gggengine.WithLifetimeContext(context.Background()))
	if err != nil {
		t.Fatalf("new embedded runtime: %v", err)
	}
	t.Cleanup(func() { _ = rt.Close(context.Background()) })
	return rt
}
