Yes. The viable design is: keep **goja fully owned by the host process**, and use **HashiCorp `go-plugin` only for discovery, process isolation, version negotiation, and RPC**. `go-plugin` launches plugin subprocesses and talks to them over `net/rpc` or gRPC, while `goja.Runtime` is single-goroutine and its object values are not transferable across runtimes. In practice, that means a plugin should **not** try to return a real `goja.Object` or `goja.Value`; it should return a **manifest** plus callable RPC methods, and the host should reify those into JS functions/objects inside its own runtime. ([GitHub][1])

The cleanest shape is:

1. **Discovery**
   Scan a plugin directory for binaries. `go-plugin` has `Discover(glob, dir)`, but it is intentionally basic: it uses a glob and currently assumes any matching file is a plugin. So you should layer your own checks on top: executable bit, filename convention, allowlist, checksum, manifest sanity. ([Go Packages][2])

2. **RPC contract**
   Each plugin binary serves one logical service such as `js_module`. That service exposes:

   * `Manifest() -> ModuleManifest`
   * `Invoke(exportName, methodName?, args) -> result`
   * optionally `GetProperty` / `SetProperty` for dynamic objects

   The manifest describes what the host should register into goja:

   * module name
   * exported functions
   * exported objects and their methods
   * version/capabilities

3. **Host-side registration into goja**
   The host receives the manifest and creates:

   * plain JS functions backed by Go closures that call `Invoke(...)`
   * plain JS objects whose methods are backed by those same closures
   * or, for truly dynamic objects, a `goja.DynamicObject` via `Runtime.NewDynamicObject()`

4. **Marshalling boundary**
   Only exchange host-language-neutral data across RPC: `null`, bool, string, numbers, byte slices, arrays, maps, maybe JSON/CBOR/protobuf `Struct`. On the goja side, arguments can be converted with `Value.Export()` / `ExportTo()`, and results can be reintroduced with `Runtime.ToValue()`. ([Go Packages][3])

5. **Security/versioning**
   The handshake cookie is only a UX check, not a security boundary. For actual hardening, use `SecureConfig` to verify plugin checksums before execution and `AutoMTLS` for transport authentication. If you use gRPC, explicitly opt in with `AllowedProtocols`, because the client defaults to accepting only `net/rpc` for legacy reasons. `VersionedPlugins` is the right place to negotiate contract versions. ([GitHub][4])

## Recommended design

Use **gRPC**, not `net/rpc`, unless every plugin is Go-only and you want the smallest possible prototype. `go-plugin` supports both, but gRPC gives you an explicit schema and better future-proofing. The library’s gRPC plugin interface is `GRPCServer(...)` / `GRPCClient(...)`, and gRPC-based plugins are the path HashiCorp uses for broader language interoperability. ([Go Packages][2])

### Shared contract

Define a host/plugin interface like this:

```go
type JSModule interface {
	Manifest(ctx context.Context) (*ModuleManifest, error)
	Invoke(ctx context.Context, req *InvokeRequest) (*InvokeResponse, error)
}

type ModuleManifest struct {
	Name    string
	Version string
	Exports []ExportDesc
}

type ExportDesc struct {
	Name    string   // "slugify" or "math"
	Kind    string   // "function" or "object"
	Methods []string // only for Kind=="object"
}
```

For gRPC, map that to protobuf. Keep payloads boring. Do not try to serialize closures or `goja.Value`.

### `go-plugin` wrapper

This follows the same pattern as HashiCorp’s gRPC example: a shared `PluginMap`, a handshake config, and a plugin type that implements `GRPCServer` and `GRPCClient`. ([GitHub][5])

```go
// shared/plugin.go
type JSModulePlugin struct {
	plugin.NetRPCUnsupportedPlugin
	Impl JSModule
}

func (p *JSModulePlugin) GRPCServer(b *plugin.GRPCBroker, s *grpc.Server) error {
	pb.RegisterJSModuleServer(s, &grpcServer{impl: p.Impl})
	return nil
}

func (p *JSModulePlugin) GRPCClient(
	ctx context.Context,
	b *plugin.GRPCBroker,
	conn *grpc.ClientConn,
) (interface{}, error) {
	return &grpcClient{client: pb.NewJSModuleClient(conn)}, nil
}

var Handshake = plugin.HandshakeConfig{
	ProtocolVersion: 1,
	MagicCookieKey:   "MYAPP_PLUGIN",
	MagicCookieValue: "js-module",
}

var PluginMap = map[string]plugin.Plugin{
	"js_module": &JSModulePlugin{},
}
```

### Plugin binary

Each plugin is just a subprocess that calls `plugin.Serve(...)` in `main()`. That is the intended server-side entrypoint. ([Go Packages][2])

```go
func main() {
	impl := &MyPlugin{} // implements JSModule

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: Handshake,
		VersionedPlugins: map[int]plugin.PluginSet{
			1: PluginMap,
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
```

### Host discovery and loading

```go
func discoverAndLoad(dir string) ([]LoadedModule, error) {
	paths, err := plugin.Discover("myapp-plugin-*", dir)
	if err != nil {
		return nil, err
	}

	var out []LoadedModule
	for _, path := range paths {
		client := plugin.NewClient(&plugin.ClientConfig{
			Cmd: exec.Command(path),
			HandshakeConfig: Handshake,
			VersionedPlugins: map[int]plugin.PluginSet{
				1: PluginMap,
			},
			AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
			AutoMTLS:         true,
			// SecureConfig: checksum config here
		})

		rpcClient, err := client.Client()
		if err != nil {
			client.Kill()
			continue
		}

		raw, err := rpcClient.Dispense("js_module")
		if err != nil {
			client.Kill()
			continue
		}

		mod := raw.(JSModule)
		manifest, err := mod.Manifest(context.Background())
		if err != nil {
			client.Kill()
			continue
		}

		out = append(out, LoadedModule{
			Client:   client,
			Module:   mod,
			Manifest: manifest,
		})
	}

	return out, nil
}
```

## How to register into goja

For fixed exports, create ordinary goja globals or module objects. `Runtime.Set()` sets a global after converting via `ToValue()`. ([Go Packages][3])

```go
func registerModule(vm *goja.Runtime, mod JSModule, m *ModuleManifest) error {
	moduleObj := vm.NewObject()

	for _, exp := range m.Exports {
		exp := exp

		switch exp.Kind {
		case "function":
			fnName := exp.Name
			fn := func(call goja.FunctionCall) goja.Value {
				args := make([]any, len(call.Arguments))
				for i, a := range call.Arguments {
					args[i] = a.Export()
				}

				resp, err := mod.Invoke(context.Background(), &InvokeRequest{
					ExportName: fnName,
					Args:       args,
				})
				if err != nil {
					panic(vm.NewGoError(err))
				}

				return vm.ToValue(resp.Result)
			}

			if err := moduleObj.Set(fnName, fn); err != nil {
				return err
			}

		case "object":
			obj := vm.NewObject()

			for _, method := range exp.Methods {
				method := method
				fn := func(call goja.FunctionCall) goja.Value {
					args := make([]any, len(call.Arguments))
					for i, a := range call.Arguments {
						args[i] = a.Export()
					}

					resp, err := mod.Invoke(context.Background(), &InvokeRequest{
						ExportName: exp.Name,
						MethodName: method,
						Args:       args,
					})
					if err != nil {
						panic(vm.NewGoError(err))
					}

					return vm.ToValue(resp.Result)
				}

				if err := obj.Set(method, fn); err != nil {
					return err
				}
			}

			if err := moduleObj.Set(exp.Name, obj); err != nil {
				return err
			}
		}
	}

	return vm.Set(m.Name, moduleObj)
}
```

That gives JS code a natural surface:

```javascript
plugin_math.add(2, 3)
plugin_fs.readFile("/tmp/x")
```

## When to use `DynamicObject`

Use `Runtime.NewDynamicObject()` only if the plugin’s object shape is not known until runtime or can change after load. goja’s `DynamicObject` lets the host implement `Get/Set/Has/Delete/Keys`, and `NewDynamicObject()` wraps that handler as a JS object. But the object’s properties are always writable/enumerable/configurable, and the object cannot be made non-extensible, so it is best used as a proxy surface, not as a strict capability model. ([Go Packages][3])

A good pattern is:

* manifest says `"kind":"dynamic-object"`
* host creates `vm.NewDynamicObject(...)`
* `Get("method")` returns a Go closure that calls back into the plugin

That is the closest analogue to a “remote JS object”.

## Important constraints

The two constraints that matter most are:

* **All goja runtime access must stay on the VM’s owning goroutine.** `goja.Runtime` is not goroutine-safe. If a plugin call blocks, that is acceptable for a synchronous API, but anything async must resolve back onto the VM goroutine. ([Go Packages][3])
* **Do not ship JS values over RPC.** Export to ordinary Go values on the host boundary, send those, then re-create JS values with `ToValue()` on the way back. ([Go Packages][3])

Because of that, I would treat plugins as **remote capability providers**, not as code that directly mutates the runtime.

## A better API than “plugin registers itself”

Do not let plugins call “register into runtime” directly. Make the flow:

* host discovers plugin binaries
* host loads plugin
* host asks for manifest
* host decides whether to allow it
* host reifies allowed exports into goja

That preserves runtime ownership, makes auditing possible, and keeps plugin crashes isolated in subprocesses, which is a core benefit of `go-plugin`’s subprocess/RPC model. ([GitHub][1])

## Practical v1 scope

For a first implementation, limit plugins to:

* top-level functions
* object methods
* JSON-like args/results
* synchronous calls

Do **not** support, initially:

* passing JS callbacks into plugins
* returning function-valued results
* mutating global scope from plugin side
* streaming/event subscriptions

If you later need callbacks or streams, `go-plugin`’s gRPC broker exists specifically to broker extra connections and complex data flows, but that is a second-phase feature. ([Go Packages][2])

Net: **yes, this architecture fits well**. The key is to think of each plugin as a **manifest + RPC service**, and of the host as the **only entity allowed to create or mutate goja objects/functions**. That gives you dynamic discovery, safe process isolation, versioning, and a predictable JS API surface.

I can turn this into a minimal working host/plugin prototype next.

[1]: https://github.com/hashicorp/go-plugin "https://github.com/hashicorp/go-plugin"
[2]: https://pkg.go.dev/github.com/hashicorp/go-plugin "https://pkg.go.dev/github.com/hashicorp/go-plugin"
[3]: https://pkg.go.dev/github.com/dop251/goja "https://pkg.go.dev/github.com/dop251/goja"
[4]: https://github.com/hashicorp/go-plugin/blob/master/examples/basic/main.go "https://github.com/hashicorp/go-plugin/blob/master/examples/basic/main.go"
[5]: https://raw.githubusercontent.com/hashicorp/go-plugin/main/examples/grpc/shared/interface.go "https://raw.githubusercontent.com/hashicorp/go-plugin/main/examples/grpc/shared/interface.go"

