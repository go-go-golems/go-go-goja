package fs

import (
	"context"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/buffer"
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
				_ = reject(fsErrorValue(vm, err))
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

func asyncReadFile(vm *goja.Runtime, bindings runtimebridge.Bindings, path string, enc goja.Value) goja.Value {
	promise, resolve, reject := vm.NewPromise()
	go func() {
		select {
		case <-bindings.Context.Done():
			return
		default:
		}

		data, err := readFileBytes(path)
		if err != nil {
			_ = bindings.Owner.Post(bindings.Context, "fs.readFile.reject", func(context.Context, *goja.Runtime) {
				_ = reject(fsErrorValue(vm, err))
			})
			return
		}
		_ = bindings.Owner.Post(bindings.Context, "fs.readFile.resolve", func(context.Context, *goja.Runtime) {
			_ = resolve(buffer.EncodeBytes(vm, data, enc))
		})
	}()
	return vm.ToValue(promise)
}

func asyncWriteFile(vm *goja.Runtime, bindings runtimebridge.Bindings, path string, data []byte, mode uint32) goja.Value {
	return asyncValue(vm, bindings, "fs.writeFile", func() (any, error) {
		return nil, writeFileBytes(path, data, fileMode(mode))
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

func asyncAppendFile(vm *goja.Runtime, bindings runtimebridge.Bindings, path string, data []byte, mode uint32) goja.Value {
	return asyncValue(vm, bindings, "fs.appendFile", func() (any, error) {
		return nil, appendFileBytes(path, data, fileMode(mode))
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

func asyncRm(vm *goja.Runtime, bindings runtimebridge.Bindings, path string, recursive, force bool) goja.Value {
	return asyncValue(vm, bindings, "fs.rm", func() (any, error) {
		return nil, rmSync(path, recursive, force)
	})
}
