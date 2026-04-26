package fs

import (
	"fmt"
	"os"
	"reflect"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/buffer"
	"github.com/go-go-golems/go-go-goja/modules"
	"github.com/go-go-golems/go-go-goja/pkg/runtimebridge"
	"github.com/go-go-golems/go-go-goja/pkg/tsgen/spec"
)

type m struct{ name string }

var _ modules.NativeModule = (*m)(nil)
var _ modules.TypeScriptDeclarer = (*m)(nil)

func (m m) Name() string {
	if m.name != "" {
		return m.name
	}
	return "fs"
}

func (m m) TypeScriptModule() *spec.Module {
	return &spec.Module{
		Name: m.Name(),
		RawDTS: []string{
			"interface FileStats {",
			"  name: string;",
			"  size: number;",
			"  mode: number;",
			"  modTime: string;",
			"  isDir: boolean;",
			"  isFile: boolean;",
			"}",
		},
		Functions: []spec.Function{
			{Name: "readFile", Params: []spec.Param{{Name: "path", Type: spec.String()}, {Name: "encoding", Type: spec.Union(spec.String(), spec.Object()), Optional: true}}, Returns: spec.Named("Promise<string | Buffer>")},
			{Name: "writeFile", Params: []spec.Param{{Name: "path", Type: spec.String()}, {Name: "data", Type: spec.Union(spec.String(), spec.Named("Buffer"), spec.Named("Uint8Array"), spec.Named("DataView"))}, {Name: "encoding", Type: spec.Union(spec.String(), spec.Object()), Optional: true}}, Returns: spec.Named("Promise<void>")},
			{Name: "exists", Params: []spec.Param{{Name: "path", Type: spec.String()}}, Returns: spec.Named("Promise<boolean>")},
			{Name: "mkdir", Params: []spec.Param{{Name: "path", Type: spec.String()}, {Name: "options", Type: spec.Object(), Optional: true}}, Returns: spec.Named("Promise<void>")},
			{Name: "readdir", Params: []spec.Param{{Name: "path", Type: spec.String()}}, Returns: spec.Named("Promise<string[]>")},
			{Name: "stat", Params: []spec.Param{{Name: "path", Type: spec.String()}}, Returns: spec.Named("Promise<FileStats>")},
			{Name: "unlink", Params: []spec.Param{{Name: "path", Type: spec.String()}}, Returns: spec.Named("Promise<void>")},
			{Name: "appendFile", Params: []spec.Param{{Name: "path", Type: spec.String()}, {Name: "data", Type: spec.Union(spec.String(), spec.Named("Buffer"), spec.Named("Uint8Array"), spec.Named("DataView"))}, {Name: "encoding", Type: spec.Union(spec.String(), spec.Object()), Optional: true}}, Returns: spec.Named("Promise<void>")},
			{Name: "rename", Params: []spec.Param{{Name: "oldPath", Type: spec.String()}, {Name: "newPath", Type: spec.String()}}, Returns: spec.Named("Promise<void>")},
			{Name: "copyFile", Params: []spec.Param{{Name: "src", Type: spec.String()}, {Name: "dst", Type: spec.String()}}, Returns: spec.Named("Promise<void>")},
			{Name: "rm", Params: []spec.Param{{Name: "path", Type: spec.String()}, {Name: "options", Type: spec.Object(), Optional: true}}, Returns: spec.Named("Promise<void>")},
			{Name: "readFileSync", Params: []spec.Param{{Name: "path", Type: spec.String()}, {Name: "encoding", Type: spec.Union(spec.String(), spec.Object()), Optional: true}}, Returns: spec.Union(spec.String(), spec.Named("Buffer"))},
			{Name: "writeFileSync", Params: []spec.Param{{Name: "path", Type: spec.String()}, {Name: "data", Type: spec.Union(spec.String(), spec.Named("Buffer"), spec.Named("Uint8Array"), spec.Named("DataView"))}, {Name: "encoding", Type: spec.Union(spec.String(), spec.Object()), Optional: true}}, Returns: spec.Void()},
			{Name: "existsSync", Params: []spec.Param{{Name: "path", Type: spec.String()}}, Returns: spec.Boolean()},
			{Name: "mkdirSync", Params: []spec.Param{{Name: "path", Type: spec.String()}, {Name: "options", Type: spec.Object(), Optional: true}}, Returns: spec.Void()},
			{Name: "readdirSync", Params: []spec.Param{{Name: "path", Type: spec.String()}}, Returns: spec.Array(spec.String())},
			{Name: "statSync", Params: []spec.Param{{Name: "path", Type: spec.String()}}, Returns: spec.Named("FileStats")},
			{Name: "unlinkSync", Params: []spec.Param{{Name: "path", Type: spec.String()}}, Returns: spec.Void()},
			{Name: "appendFileSync", Params: []spec.Param{{Name: "path", Type: spec.String()}, {Name: "data", Type: spec.Union(spec.String(), spec.Named("Buffer"), spec.Named("Uint8Array"), spec.Named("DataView"))}, {Name: "encoding", Type: spec.Union(spec.String(), spec.Object()), Optional: true}}, Returns: spec.Void()},
			{Name: "renameSync", Params: []spec.Param{{Name: "oldPath", Type: spec.String()}, {Name: "newPath", Type: spec.String()}}, Returns: spec.Void()},
			{Name: "copyFileSync", Params: []spec.Param{{Name: "src", Type: spec.String()}, {Name: "dst", Type: spec.String()}}, Returns: spec.Void()},
			{Name: "rmSync", Params: []spec.Param{{Name: "path", Type: spec.String()}, {Name: "options", Type: spec.Object(), Optional: true}}, Returns: spec.Void()},
		},
	}
}

func (m m) Doc() string {
	return `
The fs module provides promise-based and synchronous file system helpers.

Async functions return Promises: readFile, writeFile, exists, mkdir, readdir,
stat, unlink, appendFile, rename, copyFile. readFile returns a Buffer by default
and a string when an encoding is provided.

Sync functions block the JavaScript runtime: readFileSync, writeFileSync,
existsSync, mkdirSync, readdirSync, statSync, unlinkSync, appendFileSync,
renameSync, copyFileSync. writeFile/writeFileSync and appendFile/appendFileSync
accept strings, Buffers, TypedArrays, and DataViews.
`
}

func (mod m) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
	exports := moduleObj.Get("exports").(*goja.Object)
	bindings, ok := runtimebridge.Lookup(vm)
	if !ok || bindings.Owner == nil {
		panic(vm.NewGoError(fmt.Errorf("fs module requires runtime owner bindings")))
	}

	modules.SetExport(exports, mod.Name(), "readFile", func(call goja.FunctionCall) goja.Value {
		path := call.Argument(0).String()
		enc := encodingOption(vm, call.Argument(1))
		return asyncReadFile(vm, bindings, path, enc)
	})
	modules.SetExport(exports, mod.Name(), "writeFile", func(call goja.FunctionCall) goja.Value {
		path := call.Argument(0).String()
		enc, mode := writeOptions(vm, call.Argument(2))
		data := buffer.DecodeBytes(vm, call.Argument(1), enc)
		return asyncWriteFile(vm, bindings, path, data, mode)
	})
	modules.SetExport(exports, mod.Name(), "exists", func(path string) goja.Value {
		return asyncExists(vm, bindings, path)
	})
	modules.SetExport(exports, mod.Name(), "mkdir", func(call goja.FunctionCall) goja.Value {
		path := call.Argument(0).String()
		recursive, mode := mkdirOptions(vm, call.Argument(1))
		return asyncMkdir(vm, bindings, path, recursive, mode)
	})
	modules.SetExport(exports, mod.Name(), "readdir", func(path string) goja.Value {
		return asyncReaddir(vm, bindings, path)
	})
	modules.SetExport(exports, mod.Name(), "stat", func(path string) goja.Value {
		return asyncStat(vm, bindings, path)
	})
	modules.SetExport(exports, mod.Name(), "unlink", func(path string) goja.Value {
		return asyncUnlink(vm, bindings, path)
	})
	modules.SetExport(exports, mod.Name(), "appendFile", func(call goja.FunctionCall) goja.Value {
		path := call.Argument(0).String()
		enc, mode := writeOptions(vm, call.Argument(2))
		data := buffer.DecodeBytes(vm, call.Argument(1), enc)
		return asyncAppendFile(vm, bindings, path, data, mode)
	})
	modules.SetExport(exports, mod.Name(), "rename", func(oldPath, newPath string) goja.Value {
		return asyncRename(vm, bindings, oldPath, newPath)
	})
	modules.SetExport(exports, mod.Name(), "copyFile", func(src, dst string) goja.Value {
		return asyncCopyFile(vm, bindings, src, dst)
	})
	modules.SetExport(exports, mod.Name(), "rm", func(call goja.FunctionCall) goja.Value {
		recursive, force := rmOptions(vm, call.Argument(1))
		return asyncRm(vm, bindings, call.Argument(0).String(), recursive, force)
	})

	modules.SetExport(exports, mod.Name(), "readFileSync", func(call goja.FunctionCall) goja.Value {
		data, err := readFileBytes(call.Argument(0).String())
		panicFSError(vm, err)
		return buffer.EncodeBytes(vm, data, encodingOption(vm, call.Argument(1)))
	})
	modules.SetExport(exports, mod.Name(), "writeFileSync", func(call goja.FunctionCall) goja.Value {
		path := call.Argument(0).String()
		enc, mode := writeOptions(vm, call.Argument(2))
		data := buffer.DecodeBytes(vm, call.Argument(1), enc)
		panicFSError(vm, writeFileBytes(path, data, fileMode(mode)))
		return goja.Undefined()
	})
	modules.SetExport(exports, mod.Name(), "existsSync", existsSync)
	modules.SetExport(exports, mod.Name(), "mkdirSync", func(call goja.FunctionCall) goja.Value {
		path := call.Argument(0).String()
		recursive, mode := mkdirOptions(vm, call.Argument(1))
		panicFSError(vm, mkdirSync(path, recursive, fileMode(mode)))
		return goja.Undefined()
	})
	modules.SetExport(exports, mod.Name(), "readdirSync", func(path string) []string {
		ret, err := readdirSync(path)
		panicFSError(vm, err)
		return ret
	})
	modules.SetExport(exports, mod.Name(), "statSync", func(path string) fileStats {
		ret, err := statSync(path)
		panicFSError(vm, err)
		return ret
	})
	modules.SetExport(exports, mod.Name(), "unlinkSync", func(path string) {
		panicFSError(vm, unlinkSync(path))
	})
	modules.SetExport(exports, mod.Name(), "appendFileSync", func(call goja.FunctionCall) goja.Value {
		path := call.Argument(0).String()
		enc, mode := writeOptions(vm, call.Argument(2))
		data := buffer.DecodeBytes(vm, call.Argument(1), enc)
		panicFSError(vm, appendFileBytes(path, data, fileMode(mode)))
		return goja.Undefined()
	})
	modules.SetExport(exports, mod.Name(), "renameSync", func(oldPath, newPath string) {
		panicFSError(vm, renameSync(oldPath, newPath))
	})
	modules.SetExport(exports, mod.Name(), "copyFileSync", func(src, dst string) {
		panicFSError(vm, copyFileSync(src, dst))
	})
	modules.SetExport(exports, mod.Name(), "rmSync", func(call goja.FunctionCall) goja.Value {
		recursive, force := rmOptions(vm, call.Argument(1))
		panicFSError(vm, rmSync(call.Argument(0).String(), recursive, force))
		return goja.Undefined()
	})
}

func encodingOption(vm *goja.Runtime, value goja.Value) goja.Value {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return goja.Undefined()
	}
	if value.ExportType().Kind() == reflect.String {
		return value
	}
	obj := value.ToObject(vm)
	if enc := obj.Get("encoding"); enc != nil && !goja.IsUndefined(enc) && !goja.IsNull(enc) {
		return enc
	}
	return goja.Undefined()
}

func writeOptions(vm *goja.Runtime, value goja.Value) (goja.Value, uint32) {
	mode := uint32(0o644)
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return goja.Undefined(), mode
	}
	if value.ExportType().Kind() == reflect.String {
		return value, mode
	}
	obj := value.ToObject(vm)
	if m := obj.Get("mode"); m != nil && !goja.IsUndefined(m) && !goja.IsNull(m) {
		mode = uint32(m.ToInteger())
	}
	return encodingOption(vm, value), mode
}

func rmOptions(vm *goja.Runtime, value goja.Value) (bool, bool) {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return false, false
	}
	obj := value.ToObject(vm)
	recursive := false
	force := false
	if r := obj.Get("recursive"); r != nil && !goja.IsUndefined(r) {
		recursive = r.ToBoolean()
	}
	if f := obj.Get("force"); f != nil && !goja.IsUndefined(f) {
		force = f.ToBoolean()
	}
	return recursive, force
}

func mkdirOptions(vm *goja.Runtime, value goja.Value) (bool, uint32) {
	recursive := false
	mode := uint32(0o755)
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return recursive, mode
	}
	obj := value.ToObject(vm)
	if r := obj.Get("recursive"); r != nil && !goja.IsUndefined(r) {
		recursive = r.ToBoolean()
	}
	if m := obj.Get("mode"); m != nil && !goja.IsUndefined(m) {
		mode = uint32(m.ToInteger())
	}
	return recursive, mode
}

func fileMode(mode uint32) os.FileMode {
	return os.FileMode(mode)
}

func init() {
	modules.Register(&m{name: "fs"})
	modules.Register(&m{name: "node:fs"})
}
