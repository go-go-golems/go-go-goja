package extract

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseFSFile_Samples(t *testing.T) {
	root := filepath.Join("..", "..", "..", "testdata", "jsdoc")

	fd, err := ParseFSFile(os.DirFS(root), "01-math.js")
	require.NoError(t, err)
	require.NotNil(t, fd.Package)
	require.Equal(t, "math/core", fd.Package.Name)
	require.Equal(t, "01-math.js", fd.FilePath)
}

func TestScopedFS_RejectsSymlinkEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink permissions are environment-dependent on Windows")
	}

	root := t.TempDir()
	outsideDir := t.TempDir()
	outsideFile := filepath.Join(outsideDir, "outside.js")
	require.NoError(t, os.WriteFile(outsideFile, []byte(`__doc__({"name":"outside"})`), 0o644))

	linkPath := filepath.Join(root, "linked")
	require.NoError(t, os.Symlink(outsideDir, linkPath))

	scopedFS, err := NewScopedFS(root)
	require.NoError(t, err)

	_, err = ParseFSFile(scopedFS, "linked/outside.js")
	require.Error(t, err)
	require.ErrorContains(t, err, "outside allowed root")
}
