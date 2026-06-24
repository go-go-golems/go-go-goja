package main

import (
	"fmt"
	"time"

	"github.com/dop251/goja"
)

// lazyMap: keys known up front, values produced on demand by provider().
type lazyMap struct {
	keys     []string
	provider func(key string) any
	cache    map[string]goja.Value
}

// runLazyMap — a "device" object where reading .wifis triggers a Go scan.
func runLazyMap() {
	vm := goja.New()

	device := &lazyMap{
		keys: []string{"name", "wifis", "battery", "uptime"},
		provider: func(key string) any {
			switch key {
			case "name":
				return "atom-3r"
			case "wifis":
				time.Sleep(50 * time.Millisecond) // stand-in for an expensive scan
				return []string{"scapegoat", "freifunk", "eduroam"}
			case "battery":
				return 87
			case "uptime":
				return "3d 4h"
			}
			return nil
		},
		cache: map[string]goja.Value{},
	}

	_ = vm.Set("device", newLazyMapProxy(vm, device))

	// Scenario A: read a single cheap property
	reset()
	fmt.Println("== A: device.name only ==")
	out, _ := vm.RunString(`device.name`)
	fmt.Println("  result:", out.String())
	fmt.Println("  provider called for:", accessLog) // expect [name]; wifis NOT scanned

	// Scenario B: read wifis (expensive)
	reset()
	fmt.Println("== B: device.wifis (expensive) ==")
	out, _ = vm.RunString(`device.wifis.length + " networks: " + device.wifis.join(",")`)
	fmt.Println("  result:", out.String())
	fmt.Println("  provider called for:", accessLog) // expect [wifis]

	// Scenario C: enumeration must NOT materialise values
	reset()
	fmt.Println("== C: Object.keys(device) ==")
	out, _ = vm.RunString(`JSON.stringify(Object.keys(device))`)
	fmt.Println("  keys:", out.String())
	fmt.Println("  provider called for:", accessLog) // expect []

	// Scenario D: JSON.stringify materialises the values it visits
	reset()
	fmt.Println("== D: JSON.stringify(device) ==")
	out, _ = vm.RunString(`JSON.stringify(device)`)
	fmt.Println("  json:", out.String())
	fmt.Println("  provider called for:", accessLog) // expect all four

	// Scenario E: caching — second read must NOT call provider again
	reset()
	fmt.Println("== E: caching ==")
	_, _ = vm.RunString(`device.wifis`)
	_, _ = vm.RunString(`device.wifis`)
	fmt.Println("  provider called for:", accessLog) // expect [wifis] once
}

// newLazyMapProxy wraps a lazyMap in a Proxy with a fresh empty target. The
// empty, extensible target keeps the Proxy invariants trivial to satisfy.
func newLazyMapProxy(vm *goja.Runtime, lm *lazyMap) goja.Proxy {
	target := vm.NewObject()
	cache := lm.cache
	return vm.NewProxy(target, &goja.ProxyTrapConfig{
		Get: func(target *goja.Object, property string, receiver goja.Value) goja.Value {
			return lazyGet(vm, cache, lm.provider, property)
		},
		Has: func(target *goja.Object, property string) bool {
			if keySet(lm.keys, property) {
				return true
			}
			// Fall through to the target's prototype chain (inherited props).
			v := target.Get(property)
			return v != nil && !goja.IsUndefined(v)
		},
		GetOwnPropertyDescriptor: func(target *goja.Object, prop string) goja.PropertyDescriptor {
			return dataProp(lazyGet(vm, cache, lm.provider, prop))
		},
		OwnKeys: func(target *goja.Object) *goja.Object {
			strs := make([]any, len(lm.keys))
			for i, k := range lm.keys {
				strs[i] = k
			}
			return vm.NewArray(strs...)
		},
	})
}
