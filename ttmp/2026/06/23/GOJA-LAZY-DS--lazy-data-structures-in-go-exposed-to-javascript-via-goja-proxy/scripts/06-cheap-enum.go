package main

import (
	"fmt"

	"github.com/dop251/goja"
)

// runCheapEnum: the key optimisation. GetOwnPropertyDescriptor must NOT call
// the provider — it returns a descriptor with only flags, so Object.keys() /
// for..in enumeration stays cheap. Real values are produced only on [[Get]].
func runCheapEnum() {
	vm := goja.New()

	keys := []string{"name", "wifis", "battery", "uptime"}
	cache := map[string]goja.Value{}
	provider := func(key string) any {
		switch key {
		case "name":
			return "atom-3r"
		case "wifis":
			return []string{"scapegoat", "freifunk", "eduroam"}
		case "battery":
			return 87
		case "uptime":
			return "3d 4h"
		}
		return nil
	}

	target := vm.NewObject()
	proxy := vm.NewProxy(target, &goja.ProxyTrapConfig{
		Get: func(t *goja.Object, p string, r goja.Value) goja.Value {
			return lazyGet(vm, cache, provider, p)
		},
		Has: func(t *goja.Object, p string) bool {
			if keySet(keys, p) {
				return true
			}
			v := t.Get(p)
			return v != nil && !goja.IsUndefined(v)
		},
		// CHEAP descriptor: no Value, no provider call. Just flags.
		GetOwnPropertyDescriptor: func(t *goja.Object, p string) goja.PropertyDescriptor {
			return goja.PropertyDescriptor{
				Enumerable:   goja.FLAG_TRUE,
				Configurable: goja.FLAG_TRUE,
			}
		},
		OwnKeys: func(t *goja.Object) *goja.Object {
			strs := make([]any, len(keys))
			for i, k := range keys {
				strs[i] = k
			}
			return vm.NewArray(strs...)
		},
	})
	_ = vm.Set("device", proxy)

	reset()
	fmt.Println("== cheap: Object.keys(device) ==")
	out, _ := vm.RunString(`JSON.stringify(Object.keys(device))`)
	fmt.Println("  keys:", out.String())
	fmt.Println("  provider called for:", accessLog) // EXPECT [] — enumeration is now free

	reset()
	fmt.Println("== cheap: for..in over device ==")
	_, _ = vm.RunString(`for (var k in device) { /* nothing */ }`)
	fmt.Println("  provider called for:", accessLog) // EXPECT []

	reset()
	fmt.Println("== cheap: read still works ==")
	out, _ = vm.RunString(`device.wifis.length`)
	fmt.Println("  wifis.length:", out.ToNumber().String())
	fmt.Println("  provider called for:", accessLog) // EXPECT [wifis]

	reset()
	fmt.Println("== cheap: JSON.stringify materialises visited values ==")
	out, _ = vm.RunString(`JSON.stringify(device)`)
	fmt.Println("  json:", out.String())
	fmt.Println("  provider called for:", accessLog) // EXPECT all four
}
