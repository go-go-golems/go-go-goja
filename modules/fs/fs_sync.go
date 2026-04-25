package fs

import (
	"os"
	"time"
)

type fileStats map[string]any

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

func readFileBytes(path string) ([]byte, error) {
	b, err := os.ReadFile(path)
	return b, wrapFSError(err, path, "open")
}

func writeFileBytes(path string, data []byte, mode os.FileMode) error {
	return wrapFSError(os.WriteFile(path, data, mode), path, "open")
}

func existsSync(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func mkdirSync(path string, recursive bool, mode os.FileMode) error {
	if recursive {
		return wrapFSError(os.MkdirAll(path, mode), path, "mkdir")
	}
	return wrapFSError(os.Mkdir(path, mode), path, "mkdir")
}

func readdirSync(path string) ([]string, error) {
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

func statSync(path string) (fileStats, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, wrapFSError(err, path, "stat")
	}
	return statMap(info), nil
}

func unlinkSync(path string) error {
	return wrapFSError(os.Remove(path), path, "unlink")
}

func appendFileBytes(path string, data []byte, mode os.FileMode) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, mode)
	if err != nil {
		return wrapFSError(err, path, "open")
	}
	defer f.Close()
	_, err = f.Write(data)
	return wrapFSError(err, path, "write")
}

func renameSync(oldPath, newPath string) error {
	return wrapFSError(os.Rename(oldPath, newPath), oldPath, "rename")
}

func copyFileSync(src, dst string) error {
	data, err := readFileBytes(src)
	if err != nil {
		return err
	}
	return writeFileBytes(dst, data, 0o644)
}

func rmSync(path string, recursive, force bool) error {
	var err error
	if recursive {
		err = os.RemoveAll(path)
	} else {
		err = os.Remove(path)
	}
	if force && os.IsNotExist(err) {
		return nil
	}
	return wrapFSError(err, path, "rm")
}
