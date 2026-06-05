package fs

import (
	"bytes"
	"fmt"
	"io"
	iofs "io/fs"
	"mime"
	"net/http"
	"path"
	"sort"
	"strings"

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

// SPAHandlerFromAssetsModule returns an http.Handler backed by a read-only fs
// module instance. It serves existing static files normally and falls back to
// indexFile for missing GET/HEAD routes, which lets client-side routers handle
// deep links such as /pages/demo without producing a server-side 404.
func SPAHandlerFromAssetsModule(vm *goja.Runtime, moduleValue goja.Value, root, indexFile string) (http.Handler, error) {
	backend, ok := readOnlyBackendFromModule(vm, moduleValue)
	if !ok {
		return nil, fmt.Errorf("SPA asset module must be a fs module backed by embedded read-only assets")
	}
	root = cleanVirtualPath(root)
	indexFile = strings.TrimPrefix(strings.TrimSpace(indexFile), "/")
	if indexFile == "" {
		indexFile = "index.html"
	}
	indexPath := path.Join(root, indexFile)
	if _, _, err := backend.stat(indexPath); err != nil {
		return nil, fmt.Errorf("SPA index %q: %w", indexPath, err)
	}
	return &spaHTTPHandler{backend: backend, root: root, indexPath: indexPath, fileServer: http.FileServer(http.FS(&readOnlyHTTPFS{backend: backend, root: root}))}, nil
}

type spaHTTPHandler struct {
	backend    *ReadOnlyFSBackend
	root       string
	indexPath  string
	fileServer http.Handler
}

func (h *spaHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		h.fileServer.ServeHTTP(w, r)
		return
	}
	requestPath := strings.TrimPrefix(r.URL.Path, "/")
	if requestPath == "" {
		requestPath = "."
	}
	virtualPath := h.root
	if requestPath != "." {
		virtualPath = path.Join(h.root, requestPath)
	}
	if _, _, err := h.backend.stat(virtualPath); err == nil {
		h.fileServer.ServeHTTP(w, r)
		return
	}

	data, err := h.backend.ReadFile(h.indexPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if contentType := mime.TypeByExtension(path.Ext(h.indexPath)); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	} else {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	}
	if r.Method == http.MethodHead {
		return
	}
	_, _ = w.Write(data)
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
