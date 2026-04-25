package fs

import (
	"context"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/pkg/runtimebridge"
)

func asyncValue(vm *goja.Runtime, bindings runtimebridge.Bindings, op string, fn func() (any, error)) goja.Value {
	promise, resolve, reject := vm.NewPromise()
	go func() {
		select {
		case <-bindings.Context.Done():
			return
		default:
		}

		value, err := fn()
		if err != nil {
			_ = bindings.Owner.Post(bindings.Context, op+".reject", func(context.Context, *goja.Runtime) {
				_ = reject(vm.ToValue(err.Error()))
			})
			return
		}
		_ = bindings.Owner.Post(bindings.Context, op+".resolve", func(context.Context, *goja.Runtime) {
			if value == nil {
				_ = resolve(goja.Undefined())
				return
			}
			_ = resolve(vm.ToValue(value))
		})
	}()
	return vm.ToValue(promise)
}

func asyncReadFile(vm *goja.Runtime, bindings runtimebridge.Bindings, path string) goja.Value {
	return asyncValue(vm, bindings, "fs.readFile", func() (any, error) {
		return readFileSync(path)
	})
}

func asyncWriteFile(vm *goja.Runtime, bindings runtimebridge.Bindings, path, data string) goja.Value {
	return asyncValue(vm, bindings, "fs.writeFile", func() (any, error) {
		return nil, writeFileSync(path, data)
	})
}

func asyncExists(vm *goja.Runtime, bindings runtimebridge.Bindings, path string) goja.Value {
	return asyncValue(vm, bindings, "fs.exists", func() (any, error) {
		return existsSync(path), nil
	})
}

func asyncMkdir(vm *goja.Runtime, bindings runtimebridge.Bindings, path string, recursive bool, mode uint32) goja.Value {
	return asyncValue(vm, bindings, "fs.mkdir", func() (any, error) {
		return nil, mkdirSync(path, recursive, fileMode(mode))
	})
}

func asyncReaddir(vm *goja.Runtime, bindings runtimebridge.Bindings, path string) goja.Value {
	return asyncValue(vm, bindings, "fs.readdir", func() (any, error) {
		return readdirSync(path)
	})
}

func asyncStat(vm *goja.Runtime, bindings runtimebridge.Bindings, path string) goja.Value {
	return asyncValue(vm, bindings, "fs.stat", func() (any, error) {
		return statSync(path)
	})
}

func asyncUnlink(vm *goja.Runtime, bindings runtimebridge.Bindings, path string) goja.Value {
	return asyncValue(vm, bindings, "fs.unlink", func() (any, error) {
		return nil, unlinkSync(path)
	})
}

func asyncAppendFile(vm *goja.Runtime, bindings runtimebridge.Bindings, path, data string) goja.Value {
	return asyncValue(vm, bindings, "fs.appendFile", func() (any, error) {
		return nil, appendFileSync(path, data)
	})
}

func asyncRename(vm *goja.Runtime, bindings runtimebridge.Bindings, oldPath, newPath string) goja.Value {
	return asyncValue(vm, bindings, "fs.rename", func() (any, error) {
		return nil, renameSync(oldPath, newPath)
	})
}

func asyncCopyFile(vm *goja.Runtime, bindings runtimebridge.Bindings, src, dst string) goja.Value {
	return asyncValue(vm, bindings, "fs.copyFile", func() (any, error) {
		return nil, copyFileSync(src, dst)
	})
}
