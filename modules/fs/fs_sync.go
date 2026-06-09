package fs

import (
	"errors"
	"io/fs"
	"os"
	"time"
)

type fileStats map[string]any

type OSBackend struct{}

func (OSBackend) FSCapabilities() Capabilities {
	return Capabilities{Backend: "host", Read: true, Write: true}
}

func statMap(info os.FileInfo) fileStats {
	return fileStats{
		"name":    info.Name(),
		"size":    info.Size(),
		"mode":    int64(info.Mode()),
		"modTime": info.ModTime().Format(time.RFC3339),
		"isDir":   info.IsDir(),
		"isFile":  info.Mode().IsRegular(),
	}
}

func (OSBackend) ReadFile(path string) ([]byte, error) {
	b, err := os.ReadFile(path)
	return b, wrapFSError(err, path, "open")
}

func (OSBackend) WriteFile(path string, data []byte, mode os.FileMode) error {
	return wrapFSError(os.WriteFile(path, data, mode), path, "open")
}

func (OSBackend) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (OSBackend) Mkdir(path string, recursive bool, mode os.FileMode) error {
	if recursive {
		return wrapFSError(os.MkdirAll(path, mode), path, "mkdir")
	}
	return wrapFSError(os.Mkdir(path, mode), path, "mkdir")
}

func (OSBackend) ReadDir(path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, wrapFSError(err, path, "scandir")
	}
	names := make([]string, len(entries))
	for i, entry := range entries {
		names[i] = entry.Name()
	}
	return names, nil
}

func (OSBackend) Stat(path string) (fileStats, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, wrapFSError(err, path, "stat")
	}
	return statMap(info), nil
}

func (OSBackend) Remove(path string) error {
	return wrapFSError(os.Remove(path), path, "unlink")
}

func (OSBackend) AppendFile(path string, data []byte, mode os.FileMode) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, mode)
	if err != nil {
		return wrapFSError(err, path, "open")
	}
	defer func() { _ = f.Close() }()
	_, err = f.Write(data)
	return wrapFSError(err, path, "write")
}

func (OSBackend) Rename(oldPath, newPath string) error {
	return wrapFSError(os.Rename(oldPath, newPath), oldPath, "rename")
}

func (b OSBackend) CopyFile(src, dst string) error {
	data, err := b.ReadFile(src)
	if err != nil {
		return err
	}
	return b.WriteFile(dst, data, 0o644)
}

func (OSBackend) RemoveAll(path string) error {
	return wrapFSError(os.RemoveAll(path), path, "rm")
}

func isNotExist(err error) bool {
	return errors.Is(err, fs.ErrNotExist) || os.IsNotExist(err)
}
