package fs

import (
	"errors"
	"io/fs"
	"os"

	"github.com/dop251/goja"
)

type fsOpError struct {
	err     error
	path    string
	syscall string
}

func (e *fsOpError) Error() string { return e.err.Error() }
func (e *fsOpError) Unwrap() error { return e.err }

func wrapFSError(err error, path, syscall string) error {
	if err == nil {
		return nil
	}
	return &fsOpError{err: err, path: path, syscall: syscall}
}

func fsErrorCode(err error) string {
	switch {
	case errors.Is(err, fs.ErrNotExist), os.IsNotExist(err):
		return "ENOENT"
	case errors.Is(err, fs.ErrPermission), os.IsPermission(err):
		return "EACCES"
	case errors.Is(err, fs.ErrExist):
		return "EEXIST"
	default:
		return "EIO"
	}
}

func fsErrorValue(vm *goja.Runtime, err error) goja.Value {
	if err == nil {
		return goja.Undefined()
	}
	path := ""
	syscall := ""
	var opErr *fsOpError
	if errors.As(err, &opErr) {
		path = opErr.path
		syscall = opErr.syscall
	}
	obj := vm.NewGoError(err)
	_ = obj.Set("code", fsErrorCode(err))
	if path != "" {
		_ = obj.Set("path", path)
	}
	if syscall != "" {
		_ = obj.Set("syscall", syscall)
	}
	return obj
}

func panicFSError(vm *goja.Runtime, err error) {
	if err != nil {
		panic(fsErrorValue(vm, err))
	}
}
