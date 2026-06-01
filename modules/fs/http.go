package fs

import (
	"bytes"
	"fmt"
	"io"
	iofs "io/fs"
	"net/http"
	"path"
	"sort"

	"github.com/dop251/goja"
)

const backendExportKey = "__go_go_goja_fs_backend"

// StaticHandlerFromAssetsModule returns an http.Handler backed by a read-only fs
// module instance. The moduleValue is expected to be the object returned from
// require("fs:assets") or another fs module configured with a ReadOnlyFSBackend.
func StaticHandlerFromAssetsModule(vm *goja.Runtime, moduleValue goja.Value, root string) (http.Handler, error) {
	backend, ok := readOnlyBackendFromModule(vm, moduleValue)
	if !ok {
		return nil, fmt.Errorf("static asset module must be a fs module backed by embedded read-only assets")
	}
	return http.FileServer(http.FS(&readOnlyHTTPFS{backend: backend, root: cleanVirtualPath(root)})), nil
}

func readOnlyBackendFromModule(vm *goja.Runtime, moduleValue goja.Value) (*ReadOnlyFSBackend, bool) {
	if vm == nil || moduleValue == nil || goja.IsUndefined(moduleValue) || goja.IsNull(moduleValue) {
		return nil, false
	}
	value := moduleValue.ToObject(vm).Get(backendExportKey)
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return nil, false
	}
	backend, ok := value.Export().(*ReadOnlyFSBackend)
	return backend, ok
}

type readOnlyHTTPFS struct {
	backend *ReadOnlyFSBackend
	root    string
}

func (f *readOnlyHTTPFS) Open(name string) (iofs.File, error) {
	if name == "" {
		name = "."
	}
	if !iofs.ValidPath(name) {
		return nil, &iofs.PathError{Op: "open", Path: name, Err: iofs.ErrInvalid}
	}
	virtualPath := f.root
	if name != "." {
		virtualPath = path.Join(f.root, name)
	}
	info, _, err := f.backend.stat(virtualPath)
	if err != nil {
		return nil, &iofs.PathError{Op: "open", Path: name, Err: err}
	}
	if info.IsDir() {
		return &readOnlyHTTPDir{fsys: f, virtualPath: virtualPath, info: info}, nil
	}
	data, err := f.backend.ReadFile(virtualPath)
	if err != nil {
		return nil, err
	}
	return &readOnlyHTTPFile{Reader: bytes.NewReader(data), info: info}, nil
}

type readOnlyHTTPFile struct {
	*bytes.Reader
	info iofs.FileInfo
}

func (f *readOnlyHTTPFile) Close() error { return nil }
func (f *readOnlyHTTPFile) Stat() (iofs.FileInfo, error) {
	return f.info, nil
}

type readOnlyHTTPDir struct {
	fsys        *readOnlyHTTPFS
	virtualPath string
	info        iofs.FileInfo
	offset      int
}

func (d *readOnlyHTTPDir) Close() error { return nil }
func (d *readOnlyHTTPDir) Read([]byte) (int, error) {
	return 0, io.EOF
}
func (d *readOnlyHTTPDir) Stat() (iofs.FileInfo, error) {
	return d.info, nil
}
func (d *readOnlyHTTPDir) ReadDir(n int) ([]iofs.DirEntry, error) {
	names, err := d.fsys.backend.ReadDir(d.virtualPath)
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	if d.offset >= len(names) && n > 0 {
		return nil, io.EOF
	}
	end := len(names)
	if n > 0 && d.offset+n < end {
		end = d.offset + n
	}
	entries := make([]iofs.DirEntry, 0, end-d.offset)
	for _, name := range names[d.offset:end] {
		child := path.Join(d.virtualPath, name)
		info, _, err := d.fsys.backend.stat(child)
		if err != nil {
			return nil, err
		}
		entries = append(entries, iofs.FileInfoToDirEntry(info))
	}
	d.offset = end
	return entries, nil
}
