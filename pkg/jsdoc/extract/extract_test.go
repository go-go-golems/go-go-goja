package extract

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSource_Samples(t *testing.T) {
	root := filepath.Join("..", "..", "..", "testdata", "jsdoc")

	t.Run("01-math", func(t *testing.T) {
		path := filepath.Join(root, "01-math.js")
		src, err := os.ReadFile(path)
		require.NoError(t, err)
		fd, err := ParseSource(path, src)
		require.NoError(t, err)
		require.NotNil(t, fd.Package)
		require.Equal(t, "math/core", fd.Package.Name)

		var clampFound bool
		for _, sym := range fd.Symbols {
			if sym.Name == "clamp" {
				clampFound = true
				require.Contains(t, sym.Summary, "Clamps")
				require.Greater(t, sym.Line, 0)
				break
			}
		}
		require.True(t, clampFound, "expected to find symbol clamp")
	})

	t.Run("02-easing_doc_prose_attaches_to_symbol", func(t *testing.T) {
		path := filepath.Join(root, "02-easing.js")
		src, err := os.ReadFile(path)
		require.NoError(t, err)
		fd, err := ParseSource(path, src)
		require.NoError(t, err)
		require.NotNil(t, fd.Package)
		require.Equal(t, "animation/easing", fd.Package.Name)

		var smoothstepProse string
		for _, sym := range fd.Symbols {
			if sym.Name == "smoothstep" {
				smoothstepProse = sym.Prose
				break
			}
		}
		require.NotEmpty(t, smoothstepProse)
		require.Contains(t, smoothstepProse, "smoothstep")
	})

	t.Run("03-vector2_doc_prose_attaches_to_package", func(t *testing.T) {
		path := filepath.Join(root, "03-vector2.js")
		src, err := os.ReadFile(path)
		require.NoError(t, err)
		fd, err := ParseSource(path, src)
		require.NoError(t, err)
		require.NotNil(t, fd.Package)
		require.Equal(t, "math/vector2", fd.Package.Name)
		require.Contains(t, fd.Package.Prose, "2D Vector Mathematics")
	})

	t.Run("04-events_doc_prose_attaches_to_package_and_symbol", func(t *testing.T) {
		path := filepath.Join(root, "04-events.js")
		src, err := os.ReadFile(path)
		require.NoError(t, err)
		fd, err := ParseSource(path, src)
		require.NoError(t, err)
		require.NotNil(t, fd.Package)
		require.Equal(t, "core/events", fd.Package.Name)
		require.Contains(t, fd.Package.Prose, "Event Emitter System")

		var emitterProse string
		for _, sym := range fd.Symbols {
			if sym.Name == "EventEmitter" {
				emitterProse = sym.Prose
				break
			}
		}
		require.NotEmpty(t, emitterProse)
		require.Contains(t, emitterProse, "Wildcard")
	})
}
