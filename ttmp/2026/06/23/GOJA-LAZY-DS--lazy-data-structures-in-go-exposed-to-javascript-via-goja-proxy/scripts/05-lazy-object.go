package main

import (
	"fmt"
	"strconv"

	"github.com/dop251/goja"
)

// runLazyPropertyObject — a map whose keys are unknown until queried, e.g. a
// sparse / dynamic namespace like environment variables or a key/value store.
// ownKeys enumerates the live backing map; get returns values on demand.
func runLazyPropertyObject() {
	vm := goja.New()

	// Backing store: a Go map that could be filled lazily on first access.
	backing := map[string]string{
		"HOME":     "/home/manuel",
		"EDITOR":   "nvim",
		"SHELL":    "/bin/zsh",
		"LANGUAGE": "en",
	}

	cache := map[string]goja.Value{}
	get := func(prop string) goja.Value {
		if v, ok := cache[prop]; ok {
			return v
		}
		accessLog = append(accessLog, prop)
		raw, ok := backing[prop]
		if !ok {
			return goja.Undefined()
		}
		v := vm.ToValue(raw)
		cache[prop] = v
		return v
	}

	// snapshot keys for enumeration; in a real lazy store you'd page/query.
	keys := func() []string {
		out := make([]string, 0, len(backing))
		for k := range backing {
			out = append(out, k)
		}
		return out
	}

	target := vm.NewObject()
	proxy := vm.NewProxy(target, &goja.ProxyTrapConfig{
		Get:                      func(t *goja.Object, p string, r goja.Value) goja.Value { return get(p) },
		Has:                      func(t *goja.Object, p string) bool { _, ok := backing[p]; return ok },
		GetOwnPropertyDescriptor: func(t *goja.Object, p string) goja.PropertyDescriptor { return dataProp(get(p)) },
		OwnKeys: func(t *goja.Object) *goja.Object {
			ks := keys()
			strs := make([]any, len(ks))
			for i, k := range ks {
				strs[i] = k
			}
			return vm.NewArray(strs...)
		},
	})
	_ = vm.Set("env", proxy)
	_ = vm.Set("num", func(s string) int { n, _ := strconv.Atoi(s); return n })

	reset()
	fmt.Println("== A: env.EDITOR only ==")
	out, _ := vm.RunString(`env.EDITOR`)
	fmt.Println("  result:", out.String())
	fmt.Println("  provider called for:", accessLog) // expect [EDITOR]

	reset()
	fmt.Println("== B: Object.keys(env) then count ==")
	out, _ = vm.RunString(`Object.keys(env).length`)
	fmt.Println("  key count:", out.ToNumber().String())
	fmt.Println("  provider called for:", accessLog) // expect [] (enumeration is cheap)

	reset()
	fmt.Println("== C: 'HOME' in env ==")
	out, _ = vm.RunString(`'HOME' in env && !('NOPE' in env)`)
	fmt.Println("  result:", out.ToBoolean())
	fmt.Println("  provider called for:", accessLog) // expect [] (Has doesn't materialise)
}
