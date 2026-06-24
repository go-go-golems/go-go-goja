package main

import (
	"fmt"

	"github.com/dop251/goja"
)

// runInfiniteArray — an array-like object that synthesises values at any
// non-negative integer index. Demonstrates GetIdx/HasIdx/ownKeys interplay and
// the Proxy invariant that ownKeys must agree with the target.
func runInfiniteArray() {
	vm := goja.New()

	// The "length" is reported as a large number; indices are computed on demand.
	const reportLen = 1000000

	target := vm.NewArray() // an Array target (so .length, index invariants work)
	cache := map[int]goja.Value{}
	provider := func(i int) any {
		return fmt.Sprintf("item-%d", i*i) // e.g. lazy squares
	}

	proxy := vm.NewProxy(target, &goja.ProxyTrapConfig{
		GetIdx: func(target *goja.Object, i int, receiver goja.Value) goja.Value {
			if i < 0 {
				return goja.Undefined()
			}
			if v, ok := cache[i]; ok {
				return v
			}
			accessLog = append(accessLog, fmt.Sprintf("[%d]", i))
			v := vm.ToValue(provider(i))
			cache[i] = v
			return v
		},
		HasIdx: func(target *goja.Object, i int) bool { return i >= 0 && i < reportLen },
		// length is read as a normal string property.
		Get: func(target *goja.Object, property string, receiver goja.Value) goja.Value {
			if property == "length" {
				return vm.ToValue(reportLen)
			}
			return goja.Undefined()
		},
	})
	_ = vm.Set("infinite", proxy)

	reset()
	fmt.Println("== A: index [5] and [3] on demand ==")
	out, _ := vm.RunString(`[infinite[5], infinite[3], infinite[5]].join("|")`)
	fmt.Println("  result:", out.String())           // item-25|item-9|item-25 (cached)
	fmt.Println("  provider called for:", accessLog) // expect [5] then [3]

	reset()
	fmt.Println("== B: spreading first 4 entries ==")
	out, _ = vm.RunString(`JSON.stringify([infinite[0],infinite[1],infinite[2]])`)
	fmt.Println("  result:", out.String())
	fmt.Println("  provider called for:", accessLog)
}
