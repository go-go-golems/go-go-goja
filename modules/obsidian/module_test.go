package obsidianmod_test

import (
	"context"
	"testing"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/engine"
	obsidianmod "github.com/go-go-golems/go-go-goja/modules/obsidian"
	obsidianpkg "github.com/go-go-golems/go-go-goja/pkg/obsidian"
	"github.com/go-go-golems/go-go-goja/pkg/obsidiancli"
	"github.com/stretchr/testify/require"
)

type fakeRunner struct {
	t        *testing.T
	handlers map[string]func(call obsidiancli.CallOptions) (obsidiancli.Result, error)
}

func (f *fakeRunner) Run(_ context.Context, spec obsidiancli.CommandSpec, call obsidiancli.CallOptions) (obsidiancli.Result, error) {
	handler, ok := f.handlers[spec.Name]
	require.Truef(f.t, ok, "unexpected command %s", spec.Name)
	return handler(call)
}

func newTestRuntime(t *testing.T, module *obsidianmod.Module) *engine.Runtime {
	t.Helper()

	factory, err := engine.NewBuilder().
		WithModules(engine.NativeModuleSpec{
			ModuleID:   "native:obsidian-test",
			ModuleName: module.Name(),
			Loader:     module.Loader,
		}).
		Build()
	require.NoError(t, err)

	rt, err := factory.NewRuntime(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, rt.Close(context.Background()))
	})
	return rt
}

func runScript(t *testing.T, rt *engine.Runtime, script string) {
	t.Helper()
	_, err := rt.Owner.Call(context.Background(), "test.run", func(_ context.Context, vm *goja.Runtime) (any, error) {
		return vm.RunString(script)
	})
	require.NoError(t, err)
}

func waitForValue(t *testing.T, rt *engine.Runtime, key string) any {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		value, err := rt.Owner.Call(context.Background(), "test.poll", func(_ context.Context, vm *goja.Runtime) (any, error) {
			done := vm.Get("done")
			if goja.IsUndefined(done) || !done.ToBoolean() {
				return nil, nil
			}
			return vm.Get(key).Export(), nil
		})
		require.NoError(t, err)
		if value != nil {
			return value
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", key)
	return nil
}

func TestModuleResolvesPromise(t *testing.T) {
	seenVaults := []string{}
	runner := &fakeRunner{
		t: t,
		handlers: map[string]func(call obsidiancli.CallOptions) (obsidiancli.Result, error){
			obsidiancli.SpecVersion.Name: func(call obsidiancli.CallOptions) (obsidiancli.Result, error) {
				require.Nil(t, call.Parameters)
				return obsidiancli.Result{Parsed: "1.12.4", Stdout: "1.12.4"}, nil
			},
		},
	}

	module := obsidianmod.New(obsidianmod.Options{
		NewRunner: func(cfg obsidiancli.Config) obsidianpkg.Runner {
			seenVaults = append(seenVaults, cfg.Vault)
			return runner
		},
	})
	rt := newTestRuntime(t, module)

	runScript(t, rt, `
		const obs = require("obsidian");
		obs.configure({ vault: "Test Vault" });
		globalThis.done = false;
		globalThis.value = null;
		obs.version().then(v => {
			globalThis.value = v;
			globalThis.done = true;
		}, err => {
			globalThis.value = String(err);
			globalThis.done = true;
		});
	`)

	value := waitForValue(t, rt, "value")
	require.Equal(t, "1.12.4", value)
	require.Contains(t, seenVaults, "Test Vault")
}

func TestModuleRejectsPromise(t *testing.T) {
	runner := &fakeRunner{
		t: t,
		handlers: map[string]func(call obsidiancli.CallOptions) (obsidiancli.Result, error){
			obsidiancli.SpecFilesList.Name: func(call obsidiancli.CallOptions) (obsidiancli.Result, error) {
				return obsidiancli.Result{Parsed: []string{}}, nil
			},
		},
	}

	module := obsidianmod.New(obsidianmod.Options{
		NewRunner: func(obsidiancli.Config) obsidianpkg.Runner { return runner },
	})
	rt := newTestRuntime(t, module)

	runScript(t, rt, `
		const obs = require("obsidian");
		globalThis.done = false;
		globalThis.value = null;
		obs.read("missing note").then(v => {
			globalThis.value = v;
			globalThis.done = true;
		}, err => {
			globalThis.value = String(err);
			globalThis.done = true;
		});
	`)

	value := waitForValue(t, rt, "value")
	require.Contains(t, value.(string), "not found")
}

func TestModuleQueryBuilderChainsAndRuns(t *testing.T) {
	runner := &fakeRunner{
		t: t,
		handlers: map[string]func(call obsidiancli.CallOptions) (obsidiancli.Result, error){
			obsidiancli.SpecFilesList.Name: func(call obsidiancli.CallOptions) (obsidiancli.Result, error) {
				require.Equal(t, "ZK/Claims", call.Parameters["folder"])
				require.Equal(t, "md", call.Parameters["ext"])
				return obsidiancli.Result{
					Parsed: []string{"ZK/Claims/Systems.md", "ZK/Claims/Architecture.md"},
				}, nil
			},
			obsidiancli.SpecFileRead.Name: func(call obsidiancli.CallOptions) (obsidiancli.Result, error) {
				path := call.Parameters["path"].(string)
				switch path {
				case "ZK/Claims/Systems.md":
					content := "# Systems\n\n#software\n"
					return obsidiancli.Result{Parsed: content, Stdout: content}, nil
				case "ZK/Claims/Architecture.md":
					content := "# Architecture\n\n#design\n"
					return obsidiancli.Result{Parsed: content, Stdout: content}, nil
				default:
					t.Fatalf("unexpected path: %s", path)
					return obsidiancli.Result{}, nil
				}
			},
		},
	}

	module := obsidianmod.New(obsidianmod.Options{
		NewRunner: func(obsidiancli.Config) obsidianpkg.Runner { return runner },
	})
	rt := newTestRuntime(t, module)

	runScript(t, rt, `
		const obs = require("obsidian");
		globalThis.done = false;
		globalThis.value = null;
		obs.query()
		  .inFolder("ZK/Claims")
		  .withExtension("md")
		  .tagged("software")
		  .run()
		  .then(rows => {
			globalThis.value = rows.map(r => r.path).join(",");
			globalThis.done = true;
		  }, err => {
			globalThis.value = String(err);
			globalThis.done = true;
		  });
	`)

	value := waitForValue(t, rt, "value")
	require.Equal(t, "ZK/Claims/Systems.md", value)
}
