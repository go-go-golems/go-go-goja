package host

import (
	"context"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/engine"
)

func TestRegistrarRegistersPluginModuleIntoRuntime(t *testing.T) {
	binDir := t.TempDir()
	buildTestPlugin(t, filepath.Join(binDir, "goja-plugin-echo"), "./plugins/testplugin/echo")

	factory, err := engine.NewBuilder().
		WithRuntimeModuleRegistrars(NewRegistrar(Config{
			Directories: []string{binDir},
		})).
		Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}

	rt, err := factory.NewRuntime(context.Background())
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}

	mod, err := rt.Require.Require("plugin:echo")
	if err != nil {
		t.Fatalf("require plugin module: %v", err)
	}

	obj := mod.ToObject(rt.VM)
	ping := assertFunction(t, obj.Get("ping"))
	pingResp, err := ping(goja.Undefined(), rt.VM.ToValue("hello"))
	if err != nil {
		t.Fatalf("call ping: %v", err)
	}
	if got := pingResp.String(); got != "hello" {
		t.Fatalf("ping result = %q, want hello", got)
	}

	mathObj := obj.Get("math").ToObject(rt.VM)
	add := assertFunction(t, mathObj.Get("add"))
	addResp, err := add(goja.Undefined(), rt.VM.ToValue(2), rt.VM.ToValue(3))
	if err != nil {
		t.Fatalf("call math.add: %v", err)
	}
	if got := addResp.ToInteger(); got != 5 {
		t.Fatalf("math.add = %d, want 5", got)
	}

	pidFn := assertFunction(t, obj.Get("pid"))
	pidResp, err := pidFn(goja.Undefined())
	if err != nil {
		t.Fatalf("call pid: %v", err)
	}
	pid := int(pidResp.ToInteger())

	if err := rt.Close(context.Background()); err != nil {
		t.Fatalf("close runtime: %v", err)
	}
	waitForProcessExit(t, pid)
}

func TestRegistrarLoadsSDKAuthoredExamplePlugin(t *testing.T) {
	binDir := t.TempDir()
	buildTestPlugin(t, filepath.Join(binDir, "goja-plugin-examples-greeter"), "./plugins/examples/greeter")

	factory, err := engine.NewBuilder().
		WithRuntimeModuleRegistrars(NewRegistrar(Config{
			Directories: []string{binDir},
		})).
		Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}

	rt, err := factory.NewRuntime(context.Background())
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	defer func() {
		if err := rt.Close(context.Background()); err != nil {
			t.Fatalf("close runtime: %v", err)
		}
	}()

	mod, err := rt.Require.Require("plugin:examples:greeter")
	if err != nil {
		t.Fatalf("require plugin module: %v", err)
	}

	obj := mod.ToObject(rt.VM)
	greet := assertFunction(t, obj.Get("greet"))
	greetResp, err := greet(goja.Undefined(), rt.VM.ToValue("Manuel"))
	if err != nil {
		t.Fatalf("call greet: %v", err)
	}
	if got := greetResp.String(); got != "hello, Manuel" {
		t.Fatalf("greet result = %q, want hello, Manuel", got)
	}

	stringsObj := obj.Get("strings").ToObject(rt.VM)
	upper := assertFunction(t, stringsObj.Get("upper"))
	upperResp, err := upper(goja.Undefined(), rt.VM.ToValue("hello"))
	if err != nil {
		t.Fatalf("call strings.upper: %v", err)
	}
	if got := upperResp.String(); got != "HELLO" {
		t.Fatalf("strings.upper = %q, want HELLO", got)
	}

	metaObj := obj.Get("meta").ToObject(rt.VM)
	pidFn := assertFunction(t, metaObj.Get("pid"))
	pidResp, err := pidFn(goja.Undefined())
	if err != nil {
		t.Fatalf("call meta.pid: %v", err)
	}
	if got := pidResp.ToInteger(); got <= 0 {
		t.Fatalf("meta.pid = %d, want positive pid", got)
	}
}

func TestRegistrarLoadsStatefulKVExamplePlugin(t *testing.T) {
	binDir := t.TempDir()
	buildTestPlugin(t, filepath.Join(binDir, "goja-plugin-examples-kv"), "./plugins/examples/kv")

	factory, err := engine.NewBuilder().
		WithRuntimeModuleRegistrars(NewRegistrar(Config{
			Directories: []string{binDir},
		})).
		Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}

	rt, err := factory.NewRuntime(context.Background())
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	defer func() {
		if err := rt.Close(context.Background()); err != nil {
			t.Fatalf("close runtime: %v", err)
		}
	}()

	mod, err := rt.Require.Require("plugin:examples:kv")
	if err != nil {
		t.Fatalf("require plugin module: %v", err)
	}

	store := mod.ToObject(rt.VM).Get("store").ToObject(rt.VM)
	setFn := assertFunction(t, store.Get("set"))
	getFn := assertFunction(t, store.Get("get"))
	keysFn := assertFunction(t, store.Get("keys"))
	sizeFn := assertFunction(t, store.Get("size"))
	clearFn := assertFunction(t, store.Get("clear"))

	if _, err := setFn(goja.Undefined(), rt.VM.ToValue("name"), rt.VM.ToValue("Manuel")); err != nil {
		t.Fatalf("call store.set(name): %v", err)
	}
	if _, err := setFn(goja.Undefined(), rt.VM.ToValue("role"), rt.VM.ToValue("admin")); err != nil {
		t.Fatalf("call store.set(role): %v", err)
	}

	getResp, err := getFn(goja.Undefined(), rt.VM.ToValue("name"))
	if err != nil {
		t.Fatalf("call store.get: %v", err)
	}
	if got := getResp.String(); got != "Manuel" {
		t.Fatalf("store.get(name) = %q, want Manuel", got)
	}

	sizeResp, err := sizeFn(goja.Undefined())
	if err != nil {
		t.Fatalf("call store.size: %v", err)
	}
	if got := sizeResp.ToInteger(); got != 2 {
		t.Fatalf("store.size = %d, want 2", got)
	}

	keysResp, err := keysFn(goja.Undefined())
	if err != nil {
		t.Fatalf("call store.keys: %v", err)
	}
	exported, ok := keysResp.Export().([]any)
	if !ok {
		t.Fatalf("store.keys export type = %T, want []any", keysResp.Export())
	}
	keys := make([]string, 0, len(exported))
	for _, value := range exported {
		s, ok := value.(string)
		if !ok {
			t.Fatalf("store.keys element type = %T, want string", value)
		}
		keys = append(keys, s)
	}
	if !slices.Equal(keys, []string{"name", "role"}) {
		t.Fatalf("store.keys = %#v, want [name role]", keys)
	}

	if _, err := clearFn(goja.Undefined()); err != nil {
		t.Fatalf("call store.clear: %v", err)
	}
	sizeResp, err = sizeFn(goja.Undefined())
	if err != nil {
		t.Fatalf("call store.size after clear: %v", err)
	}
	if got := sizeResp.ToInteger(); got != 0 {
		t.Fatalf("store.size after clear = %d, want 0", got)
	}
}

func TestRegistrarSurfacesPluginHandlerErrors(t *testing.T) {
	binDir := t.TempDir()
	buildTestPlugin(t, filepath.Join(binDir, "goja-plugin-examples-failing"), "./plugins/examples/failing")

	factory, err := engine.NewBuilder().
		WithRuntimeModuleRegistrars(NewRegistrar(Config{
			Directories: []string{binDir},
		})).
		Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}

	rt, err := factory.NewRuntime(context.Background())
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	defer func() {
		if err := rt.Close(context.Background()); err != nil {
			t.Fatalf("close runtime: %v", err)
		}
	}()

	mod, err := rt.Require.Require("plugin:examples:failing")
	if err != nil {
		t.Fatalf("require plugin module: %v", err)
	}

	obj := mod.ToObject(rt.VM)
	alwaysFn := assertFunction(t, obj.Get("always"))
	if _, err := alwaysFn(goja.Undefined(), rt.VM.ToValue("boom")); err == nil {
		t.Fatalf("expected always() to fail")
	} else if !strings.Contains(err.Error(), "always failed: boom") {
		t.Fatalf("always() error = %v, want reason text", err)
	}

	checks := obj.Get("checks").ToObject(rt.VM)
	requirePositive := assertFunction(t, checks.Get("requirePositive"))
	if _, err := requirePositive(goja.Undefined(), rt.VM.ToValue(-1)); err == nil {
		t.Fatalf("expected requirePositive() to fail")
	} else if !strings.Contains(err.Error(), "value must be positive") {
		t.Fatalf("requirePositive() error = %v, want validation text", err)
	}
}

func TestRegistrarRejectsInvalidManifest(t *testing.T) {
	binDir := t.TempDir()
	buildTestPlugin(t, filepath.Join(binDir, "goja-plugin-invalid"), "./plugins/testplugin/invalid")

	factory, err := engine.NewBuilder().
		WithRuntimeModuleRegistrars(NewRegistrar(Config{
			Directories: []string{binDir},
		})).
		Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}

	_, err = factory.NewRuntime(context.Background())
	if err == nil {
		t.Fatalf("expected invalid plugin manifest error")
	}
	if !strings.Contains(err.Error(), "must use namespace") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRegistrarRejectsModuleOutsideAllowlist(t *testing.T) {
	binDir := t.TempDir()
	buildTestPlugin(t, filepath.Join(binDir, "goja-plugin-echo"), "./plugins/testplugin/echo")

	factory, err := engine.NewBuilder().
		WithRuntimeModuleRegistrars(NewRegistrar(Config{
			Directories:  []string{binDir},
			AllowModules: []string{"plugin:examples:greeter"},
		})).
		Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}

	_, err = factory.NewRuntime(context.Background())
	if err == nil {
		t.Fatalf("expected allowlist error")
	}
	if !strings.Contains(err.Error(), "allowlist") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func buildTestPlugin(t *testing.T, outputPath, packagePath string) {
	t.Helper()

	repoRoot := repoRoot(t)
	cmd := exec.Command("go", "build", "-o", outputPath, packagePath)
	cmd.Dir = repoRoot
	cmd.Env = append([]string{"GOWORK=off"}, cmd.Environ()...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build plugin %s: %v\n%s", packagePath, err, string(out))
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("resolve caller")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

func assertFunction(t *testing.T, value goja.Value) goja.Callable {
	t.Helper()
	fn, ok := goja.AssertFunction(value)
	if !ok {
		t.Fatalf("value %v is not a function", value)
	}
	return fn
}

func waitForProcessExit(t *testing.T, pid int) {
	t.Helper()
	if runtime.GOOS == "windows" {
		return
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if err := syscall.Kill(pid, 0); err != nil {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("plugin process %d still alive after runtime close", pid)
}
