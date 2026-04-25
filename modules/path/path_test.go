package pathmod_test

import (
	"context"
	"strings"
	"testing"

	"github.com/dop251/goja"
	gggengine "github.com/go-go-golems/go-go-goja/engine"
)

func TestPathModuleSmoke(t *testing.T) {
	factory, err := gggengine.NewBuilder().WithModules(gggengine.DefaultRegistryModules()).Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}
	rt, err := factory.NewRuntime(context.Background())
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	defer func() { _ = rt.Close(context.Background()) }()
	ret, err := rt.Owner.Call(context.Background(), "path.smoke", func(_ context.Context, vm *goja.Runtime) (any, error) {
		v, err := vm.RunString(`
			const path = require("path");
			JSON.stringify({
				join: path.join("a", "b", "c.txt"),
				dir: path.dirname("/tmp/a/b.txt"),
				base: path.basename("/tmp/a/b.txt"),
				ext: path.extname("/tmp/a/b.txt"),
				abs: path.isAbsolute("/tmp"),
				rel: path.relative("/tmp/a", "/tmp/a/b/c"),
				sep: typeof path.separator,
				delim: typeof path.delimiter,
				resolved: path.isAbsolute(path.resolve("."))
			});
		`)
		if err != nil {
			return nil, err
		}
		return v.String(), nil
	})
	if err != nil {
		t.Fatalf("run path smoke: %v", err)
	}
	s := ret.(string)
	for _, want := range []string{`c.txt`, `"base":"b.txt"`, `"ext":".txt"`, `"abs":true`, `"sep":"string"`, `"delim":"string"`, `"resolved":true`} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %s in %s", want, s)
		}
	}
}
