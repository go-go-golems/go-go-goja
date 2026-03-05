package extract

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

var (
	ErrEmptyPath          = errors.New("path is empty")
	ErrAbsolutePath       = errors.New("absolute paths are not allowed")
	ErrInvalidPath        = errors.New("path is invalid")
	ErrPathTraversal      = errors.New("path traversal is not allowed")
	ErrOutsideAllowedRoot = errors.New("path is outside allowed root")
)

// ScopedFS is an fs.FS rooted at a real filesystem directory that rejects
// absolute paths, traversal, and symlink escapes outside the configured root.
type ScopedFS struct {
	root     string
	realRoot string
}

var _ fs.FS = (*ScopedFS)(nil)
var _ fs.ReadFileFS = (*ScopedFS)(nil)

// NewScopedFS creates a filesystem view rooted at root.
func NewScopedFS(root string) (*ScopedFS, error) {
	if root == "" {
		return nil, errors.New("root is empty")
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, errors.Wrap(err, "abs root")
	}
	realRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		return nil, errors.Wrap(err, "eval root symlinks")
	}

	return &ScopedFS{
		root:     absRoot,
		realRoot: realRoot,
	}, nil
}

func (s *ScopedFS) Open(name string) (fs.File, error) {
	resolved, err := s.resolve(name)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(resolved)
	if err != nil {
		return nil, errors.Wrap(err, "open scoped file")
	}
	return f, nil
}

func (s *ScopedFS) ReadFile(name string) ([]byte, error) {
	resolved, err := s.resolve(name)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(resolved)
	if err != nil {
		return nil, errors.Wrap(err, "read scoped file")
	}
	return b, nil
}

func (s *ScopedFS) resolve(name string) (string, error) {
	if name == "" {
		return "", ErrEmptyPath
	}
	if filepath.IsAbs(name) {
		return "", ErrAbsolutePath
	}

	clean := filepath.Clean(name)
	if clean == "." || clean == string(filepath.Separator) {
		return "", ErrInvalidPath
	}
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", ErrPathTraversal
	}

	full := filepath.Join(s.root, clean)
	resolved, err := filepath.EvalSymlinks(full)
	if err != nil {
		return "", errors.Wrap(err, "eval path symlinks")
	}
	resolvedAbs, err := filepath.Abs(resolved)
	if err != nil {
		return "", errors.Wrap(err, "abs resolved path")
	}

	sep := string(os.PathSeparator)
	if resolvedAbs != s.realRoot && !strings.HasPrefix(resolvedAbs, s.realRoot+sep) {
		return "", ErrOutsideAllowedRoot
	}

	return resolvedAbs, nil
}
