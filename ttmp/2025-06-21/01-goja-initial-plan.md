Below is a practical repo blueprint that lets you drop any number of Go-implemented “native” modules into a single place and expose them to JavaScript running inside **goja** with the *goja\_nodejs* `require()` system.

---

## 1 · Folder layout

```
go-go-goja/
├── cmd/
│   └── repl/            # tiny example runner (loads a script or REPL)
├── engine/              # helpers to spin-up a goja runtime
│   └── runtime.go
├── modules/             # ❶ every native module lives here
│   ├── common.go        # registry plumbing (see §3)
│   ├── fs/
│   │   └── fs.go
│   └── exec/
│       └── exec.go
├── go.mod
└── README.md
```

*You can add more modules later by creating another sub-folder under `modules/` and a single `*.go` file.*

---

## 2 · Baseline dependencies

```go
require (
    github.com/dop251/goja             v0.0.0-…   // JS VM
    github.com/dop251/goja_nodejs      v0.0.0-…   // Node-style require()
)
```

`goja_nodejs/require` gives you:

* `type ModuleLoader func(*goja.Runtime, *goja.Object)` – the exact signature each module must implement
* `(*Registry).RegisterNativeModule(name, loader)` – calls this once per module
  ([pkg.go.dev][1])

---

## 3 · One-time registry plumbing (`modules/common.go`)

```go
package modules

import (
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
)

// The interface every sub-module satisfies
type NativeModule interface {
	Name() string
	Loader(*goja.Runtime, *goja.Object)
}

// internal slice automatically filled by init()s
var all []NativeModule

// Called by each module's init()
func register(m NativeModule) { all = append(all, m) }

// EnableAll wires every registered module into a Registry.
func EnableAll(reg *require.Registry) {
	for _, m := range all {
		reg.RegisterNativeModule(m.Name(), m.Loader)
	}
}
```

---

## 4 · Example module #1 – `fs`

```go
// modules/fs/fs.go
package fs

import (
	"os"

	"github.com/dop251/goja"
	"your.repo/jsmods/modules"
)

type m struct{}

func (m) Name() string { return "fs" }

func (m) Loader(vm *goja.Runtime, module *goja.Object) {
	exports := module.Get("exports").(*goja.Object)

	exports.Set("readFileSync", func(path string) (string, error) {
		b, err := os.ReadFile(path)
		return string(b), err
	})

	exports.Set("writeFileSync", func(path, data string) error {
		return os.WriteFile(path, []byte(data), 0o644)
	})
}

func init() { modules.Register(&m{}) }
```

---

## 5 · Example module #2 – `exec`

```go
// modules/exec/exec.go
package execmod

import (
	"os/exec"

	"github.com/dop251/goja"
	"your.repo/jsmods/modules"
)

type m struct{}

func (m) Name() string { return "exec" }

func (m) Loader(vm *goja.Runtime, module *goja.Object) {
	exports := module.Get("exports").(*goja.Object)

	exports.Set("run", func(cmd string, args []string) (string, error) {
		out, err := exec.Command(cmd, args...).CombinedOutput()
		return string(out), err
	})
}

func init() { modules.Register(&m{}) }
```

---

## 6 · Runtime helper (`engine/runtime.go`)

```go
package engine

import (
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"

	"your.repo/jsmods/modules"
)

func New() (*goja.Runtime, *require.RequireModule) {
	vm      := goja.New()
	reg     := require.NewRegistry()
	modules.EnableAll(reg)          // <- wires every native module
	reqMod  := reg.Enable(vm)        // installs require() in JS
	return vm, reqMod
}
```

---

## 7 · Tiny runner (`cmd/repl/main.go`)

```go
package main

import (
	"fmt"
	"log"
	"os"

	"your.repo/jsmods/engine"
)

func main() {
	vm, req := engine.New()

	if len(os.Args) > 1 {          // run a file
		if _, err := req.Require(os.Args[1]); err != nil {
			log.Fatal(err)
		}
		return
	}

	// quick REPL
	_, _ = vm.RunString(`console.log("native modules:", Object.keys(require.cache));`)
}
```

---

## 8 · Using the modules from JS

```js
const fs   = require("fs");
const exec = require("exec");

fs.writeFileSync("hello.txt", "Hi from Go!");
console.log(fs.readFileSync("hello.txt"));

const out = exec.run("echo", ["yo"]);
console.log(out);
```

Everything is discoverable through plain Node-style `require()` because each Go module was registered via `RegisterNativeModule()` — the mechanism officially recommended in *goja\_nodejs*. ([pkg.go.dev][1])

---

### How to extend

1. `mkdir modules/<newname>`
2. Implement `Name()` and `Loader()` as above
3. Call `modules.Register(&yourMod{})` in `init()`
4. Go modules are auto-wired at runtime through `modules.EnableAll()` – no extra bookkeeping needed.

This structure keeps native bindings self-contained, encourages small packages, and scales as your catalogue of Go-backed JS modules grows.

[1]: https://pkg.go.dev/github.com/dop251/goja_nodejs/require "require package - github.com/dop251/goja_nodejs/require - Go Packages"

