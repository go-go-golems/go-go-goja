package host

import (
	"context"
	"os/exec"
	"path/filepath"
	"runtime"
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
			AllowModules: []string{"plugin:greeter"},
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
