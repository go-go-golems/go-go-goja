package cryptomod_test

import (
	"context"
	"strings"
	"testing"

	"github.com/dop251/goja"
	gggengine "github.com/go-go-golems/go-go-goja/engine"
)

func TestCryptoModuleSmoke(t *testing.T) {
	factory, err := gggengine.NewBuilder().UseModuleMiddleware(gggengine.MiddlewareSafe()).Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}
	rt, err := factory.NewRuntime(context.Background())
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}
	defer func() { _ = rt.Close(context.Background()) }()
	ret, err := rt.Owner.Call(context.Background(), "crypto.smoke", func(_ context.Context, vm *goja.Runtime) (any, error) {
		v, err := vm.RunString(`
			const crypto = require("crypto");
			const id = crypto.randomUUID();
			const rb = crypto.randomBytes(8);
			const hex = crypto.createHash("sha256").update("abc").digest("hex");
			const raw = crypto.createHash("sha1").update(Buffer.from("abc")).digest();
			JSON.stringify({ uuid: id.length, bytes: rb.length, hex, rawLen: raw.length });
		`)
		if err != nil {
			return nil, err
		}
		return v.String(), nil
	})
	if err != nil {
		t.Fatalf("run crypto smoke: %v", err)
	}
	s := ret.(string)
	for _, want := range []string{`"uuid":36`, `"bytes":8`, `"hex":"ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"`, `"rawLen":20`} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %s in %s", want, s)
		}
	}
}
