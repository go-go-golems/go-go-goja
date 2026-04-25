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
	return os.ReadFile(path)
}

func writeFileBytes(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644)
}

func existsSync(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func mkdirSync(path string, recursive bool, mode os.FileMode) error {
	if recursive {
		return os.MkdirAll(path, mode)
	}
	return os.Mkdir(path, mode)
}

func readdirSync(path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
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
		return nil, err
	}
	return statMap(info), nil
}

func unlinkSync(path string) error {
	return os.Remove(path)
}

func appendFileBytes(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)
	return err
}

func renameSync(oldPath, newPath string) error {
	return os.Rename(oldPath, newPath)
}

func copyFileSync(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}
