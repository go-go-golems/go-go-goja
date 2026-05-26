package fs

import (
	"context"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/buffer"
	"github.com/go-go-golems/go-go-goja/pkg/runtimebridge"
)

func asyncValue(vm *goja.Runtime, runtimeServices runtimebridge.RuntimeServices, op string, fn func() (any, error)) goja.Value {
	promise, resolve, reject := vm.NewPromise()
	callCtx := runtimebridge.CurrentOwnerContext(vm)
	runtimeCtx := bindingContext(runtimeServices)
	go func() {
		select {
		case <-callCtx.Done():
			return
		case <-runtimeCtx.Done():
			return
		default:
		}

		value, err := fn()
		if err != nil {
			_ = runtimeServices.PostWithCustomContext(callCtx, op+".reject", func(context.Context, *goja.Runtime) {
				_ = reject(fsErrorValue(vm, err))
			})
			return
		}
		_ = runtimeServices.PostWithCustomContext(callCtx, op+".resolve", func(context.Context, *goja.Runtime) {
			if value == nil {
				_ = resolve(goja.Undefined())
				return
			}
			_ = resolve(vm.ToValue(value))
		})
	}()
	return vm.ToValue(promise)
}

func asyncReadFile(vm *goja.Runtime, runtimeServices runtimebridge.RuntimeServices, path string, enc goja.Value) goja.Value {
	promise, resolve, reject := vm.NewPromise()
	callCtx := runtimebridge.CurrentOwnerContext(vm)
	runtimeCtx := bindingContext(runtimeServices)
	go func() {
		select {
		case <-callCtx.Done():
			return
		case <-runtimeCtx.Done():
			return
		default:
		}

		data, err := readFileBytes(path)
		if err != nil {
			_ = runtimeServices.PostWithCustomContext(callCtx, "fs.readFile.reject", func(context.Context, *goja.Runtime) {
				_ = reject(fsErrorValue(vm, err))
			})
			return
		}
		_ = runtimeServices.PostWithCustomContext(callCtx, "fs.readFile.resolve", func(context.Context, *goja.Runtime) {
			_ = resolve(buffer.EncodeBytes(vm, data, enc))
		})
	}()
	return vm.ToValue(promise)
}

func bindingContext(runtimeServices runtimebridge.RuntimeServices) context.Context {
	return runtimeServices.Lifetime()
}

func asyncWriteFile(vm *goja.Runtime, runtimeServices runtimebridge.RuntimeServices, path string, data []byte, mode uint32) goja.Value {
	return asyncValue(vm, runtimeServices, "fs.writeFile", func() (any, error) {
		return nil, writeFileBytes(path, data, fileMode(mode))
	})
}

func asyncExists(vm *goja.Runtime, runtimeServices runtimebridge.RuntimeServices, path string) goja.Value {
	return asyncValue(vm, runtimeServices, "fs.exists", func() (any, error) {
		return existsSync(path), nil
	})
}

func asyncMkdir(vm *goja.Runtime, runtimeServices runtimebridge.RuntimeServices, path string, recursive bool, mode uint32) goja.Value {
	return asyncValue(vm, runtimeServices, "fs.mkdir", func() (any, error) {
		return nil, mkdirSync(path, recursive, fileMode(mode))
	})
}

func asyncReaddir(vm *goja.Runtime, runtimeServices runtimebridge.RuntimeServices, path string) goja.Value {
	return asyncValue(vm, runtimeServices, "fs.readdir", func() (any, error) {
		return readdirSync(path)
	})
}

func asyncStat(vm *goja.Runtime, runtimeServices runtimebridge.RuntimeServices, path string) goja.Value {
	return asyncValue(vm, runtimeServices, "fs.stat", func() (any, error) {
		return statSync(path)
	})
}

func asyncUnlink(vm *goja.Runtime, runtimeServices runtimebridge.RuntimeServices, path string) goja.Value {
	return asyncValue(vm, runtimeServices, "fs.unlink", func() (any, error) {
		return nil, unlinkSync(path)
	})
}

func asyncAppendFile(vm *goja.Runtime, runtimeServices runtimebridge.RuntimeServices, path string, data []byte, mode uint32) goja.Value {
	return asyncValue(vm, runtimeServices, "fs.appendFile", func() (any, error) {
		return nil, appendFileBytes(path, data, fileMode(mode))
	})
}

func asyncRename(vm *goja.Runtime, runtimeServices runtimebridge.RuntimeServices, oldPath, newPath string) goja.Value {
	return asyncValue(vm, runtimeServices, "fs.rename", func() (any, error) {
		return nil, renameSync(oldPath, newPath)
	})
}

func asyncCopyFile(vm *goja.Runtime, runtimeServices runtimebridge.RuntimeServices, src, dst string) goja.Value {
	return asyncValue(vm, runtimeServices, "fs.copyFile", func() (any, error) {
		return nil, copyFileSync(src, dst)
	})
}

func asyncRm(vm *goja.Runtime, runtimeServices runtimebridge.RuntimeServices, path string, recursive, force bool) goja.Value {
	return asyncValue(vm, runtimeServices, "fs.rm", func() (any, error) {
		return nil, rmSync(path, recursive, force)
	})
}
