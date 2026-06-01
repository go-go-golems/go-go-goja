package fs

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"sort"
	"strings"
)

var errReadOnlyFS = errors.New("read-only file system")

type FSMount struct {
	FS    fs.FS
	Root  string
	Mount string
}

type ReadOnlyFSBackend struct {
	mounts []FSMount
}

func NewReadOnlyFSBackend(mounts ...FSMount) *ReadOnlyFSBackend {
	out := &ReadOnlyFSBackend{mounts: make([]FSMount, 0, len(mounts))}
	for _, mount := range mounts {
		if mount.FS == nil {
			continue
		}
		mount.Root = cleanFSRoot(mount.Root)
		mount.Mount = cleanVirtualMount(mount.Mount)
		if mount.Mount == "" {
			continue
		}
		out.mounts = append(out.mounts, mount)
	}
	sort.SliceStable(out.mounts, func(i, j int) bool {
		return len(out.mounts[i].Mount) > len(out.mounts[j].Mount)
	})
	return out
}

func (b *ReadOnlyFSBackend) ReadFile(p string) ([]byte, error) {
	fsys, subpath, ok := b.resolve(p)
	if !ok {
		return nil, wrapFSError(fs.ErrNotExist, p, "open")
	}
	data, err := fs.ReadFile(fsys, subpath)
	return data, wrapFSError(err, p, "open")
}

func (b *ReadOnlyFSBackend) WriteFile(p string, data []byte, mode os.FileMode) error {
	_ = data
	_ = mode
	return b.mutationError(p, "open")
}

func (b *ReadOnlyFSBackend) Exists(p string) bool {
	_, _, err := b.stat(p)
	return err == nil
}

func (b *ReadOnlyFSBackend) Mkdir(p string, recursive bool, mode os.FileMode) error {
	_ = recursive
	_ = mode
	return b.mutationError(p, "mkdir")
}

func (b *ReadOnlyFSBackend) ReadDir(p string) ([]string, error) {
	fsys, subpath, ok := b.resolve(p)
	if !ok {
		return nil, wrapFSError(fs.ErrNotExist, p, "scandir")
	}
	entries, err := fs.ReadDir(fsys, subpath)
	if err != nil {
		return nil, wrapFSError(err, p, "scandir")
	}
	names := make([]string, len(entries))
	for i, entry := range entries {
		names[i] = entry.Name()
	}
	return names, nil
}

func (b *ReadOnlyFSBackend) Stat(p string) (fileStats, error) {
	info, _, err := b.stat(p)
	if err != nil {
		return nil, wrapFSError(err, p, "stat")
	}
	return statMap(info), nil
}

func (b *ReadOnlyFSBackend) Remove(p string) error {
	return b.mutationError(p, "unlink")
}

func (b *ReadOnlyFSBackend) AppendFile(p string, data []byte, mode os.FileMode) error {
	_ = data
	_ = mode
	return b.mutationError(p, "open")
}

func (b *ReadOnlyFSBackend) Rename(oldPath, newPath string) error {
	if b.isMountedPath(oldPath) || b.isMountedPath(newPath) {
		return wrapFSError(errReadOnlyFS, oldPath, "rename")
	}
	return wrapFSError(fs.ErrNotExist, oldPath, "rename")
}

func (b *ReadOnlyFSBackend) CopyFile(src, dst string) error {
	if b.isMountedPath(dst) {
		return wrapFSError(errReadOnlyFS, dst, "open")
	}
	if b.isMountedPath(src) {
		return wrapFSError(errReadOnlyFS, dst, "open")
	}
	return wrapFSError(fs.ErrNotExist, dst, "open")
}

func (b *ReadOnlyFSBackend) RemoveAll(p string) error {
	return b.mutationError(p, "rm")
}

func (b *ReadOnlyFSBackend) stat(p string) (fs.FileInfo, string, error) {
	fsys, subpath, ok := b.resolve(p)
	if !ok {
		return nil, "", fs.ErrNotExist
	}
	info, err := fs.Stat(fsys, subpath)
	return info, subpath, err
}

func (b *ReadOnlyFSBackend) mutationError(p string, syscall string) error {
	if b.isMountedPath(p) {
		return wrapFSError(errReadOnlyFS, p, syscall)
	}
	return wrapFSError(fs.ErrNotExist, p, syscall)
}

func (b *ReadOnlyFSBackend) resolve(p string) (fs.FS, string, bool) {
	clean := cleanVirtualPath(p)
	for _, mount := range b.mounts {
		if clean == mount.Mount || strings.HasPrefix(clean, mount.Mount+"/") {
			rel := strings.TrimPrefix(clean, mount.Mount)
			rel = strings.TrimPrefix(rel, "/")
			if rel == "" {
				rel = "."
			}
			return mount.FS, path.Join(mount.Root, rel), true
		}
	}
	return nil, "", false
}

func (b *ReadOnlyFSBackend) isMountedPath(p string) bool {
	_, _, ok := b.resolve(p)
	return ok
}

func cleanVirtualPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return "/"
	}
	return path.Clean("/" + strings.TrimPrefix(p, "/"))
}

func cleanVirtualMount(p string) string {
	clean := cleanVirtualPath(p)
	if clean == "/" {
		return ""
	}
	return clean
}

func cleanFSRoot(root string) string {
	root = strings.TrimSpace(strings.TrimPrefix(root, "/"))
	if root == "" || root == "." {
		return "."
	}
	return path.Clean(root)
}
