// Example: lazy data structures exposed to JavaScript via goja Proxy.
//
// Run from the repo root:
//
//	go run ./examples/lazy
//
// Builds a runtime whose require("lazy") module wires Go-backed lazy Proxies,
// then runs a small JS script that reads values on demand, enumerates cheaply,
// and shows caching + a million-element lazy array.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/pkg/engine"
)

func main() {
	factory, err := engine.NewRuntimeFactoryBuilder().
		UseModuleMiddleware(engine.MiddlewareOnly("lazy")).
		Build()
	if err != nil {
		log.Fatalf("build factory: %v", err)
	}
	ctx := context.Background()
	rt, err := factory.NewRuntime(engine.WithStartupContext(ctx), engine.WithLifetimeContext(ctx))
	if err != nil {
		log.Fatalf("new runtime: %v", err)
	}
	defer func() { _ = rt.Close(ctx) }()

	const script = `
		const lazy = require("lazy");

		// 1) A "device" where scanning wifis only happens on read.
		let providerCalls = {};
		const device = lazy.map({
			keys: ["name", "wifis", "battery"],
			get: (k) => {
				providerCalls[k] = (providerCalls[k] || 0) + 1;
				if (k === "wifis") return ["scapegoat", "freifunk", "eduroam"];
				if (k === "battery") return 87;
				return "atom-3r";
			}
		});

		// Enumerate WITHOUT triggering the wifi scan:
		const keys = Object.keys(device);
		const enumCalls = (providerCalls.name||0) + (providerCalls.wifis||0) + (providerCalls.battery||0);

		// Now read the expensive one (lazily), twice (cached):
		const n = device.wifis.length;
		const n2 = device.wifis.length;

		// 2) A lazy array of a million squares — element i computed on demand.
		const squares = lazy.array({ length: 1_000_000, get: (i) => i * i });

		JSON.stringify({
			keys, enumCalls,
			wifiCount: n, wifiCalls: providerCalls.wifis||0, cached: n === n2,
			square1000: squares[1000], length: squares.length
		});
	`

	var result string
	if _, err := rt.Owner.Call(ctx, "lazy-demo", func(_ context.Context, vm *goja.Runtime) (any, error) {
		start := time.Now()
		v, runErr := vm.RunString(script)
		fmt.Printf("ran in %s\n", time.Since(start))
		if runErr != nil {
			return nil, runErr
		}
		result = v.String()
		return result, nil
	}); err != nil {
		log.Fatalf("run: %v", err)
	}

	fmt.Println(result)
}
