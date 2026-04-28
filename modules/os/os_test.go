package osmod_test

import (
	"context"
	"strings"
	"testing"

	"github.com/dop251/goja"
	gggengine "github.com/go-go-golems/go-go-goja/engine"
)

func TestOSModuleSmoke(t *testing.T) {
	factory, err := gggengine.NewBuilder().WithModules(gggengine.DefaultRegistryModules()).Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}
	rt, err := factory.NewRuntime(context.Background())
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	defer func() { _ = rt.Close(context.Background()) }()
	ret, err := rt.Owner.Call(context.Background(), "os.smoke", func(_ context.Context, vm *goja.Runtime) (any, error) {
		v, err := vm.RunString(`
			const os = require("os");
			JSON.stringify({
				home: typeof os.homedir(), tmp: typeof os.tmpdir(), platform: os.platform(),
				arch: typeof os.arch(), hostname: typeof os.hostname(), cpus: os.cpus().length > 0,
				eol: typeof os.EOL
			});
		`)
		if err != nil {
			return nil, err
		}
		return v.String(), nil
	})
	if err != nil {
		t.Fatalf("run os smoke: %v", err)
	}
	s := ret.(string)
	for _, want := range []string{`"home":"string"`, `"tmp":"string"`, `"arch":"string"`, `"hostname":"string"`, `"cpus":true`, `"eol":"string"`} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %s in %s", want, s)
		}
	}
}
