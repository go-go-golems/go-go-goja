package resultcontent

import (
	"context"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pkg/errors"
)

// DefaultFilePrefix is the file name prefix WriteFiles uses when none is given.
const DefaultFilePrefix = "content"

// WrittenFile describes one content item written by WriteFiles.
type WrittenFile struct {
	// Provider is the 1-based provider number from "[content N: ...]".
	Provider int `json:"provider"`
	// Item is the 1-based position of the item among its provider's items.
	Item     int    `json:"item"`
	Path     string `json:"path"`
	MIMEType string `json:"mimeType,omitempty"`
	Caption  string `json:"caption,omitempty"`
	Bytes    int    `json:"bytes"`
}

var knownExtensions = map[string]string{
	"image/png":                ".png",
	"image/jpeg":               ".jpg",
	"image/gif":                ".gif",
	"image/webp":               ".webp",
	"image/tiff":               ".tif",
	"image/bmp":                ".bmp",
	"image/svg+xml":            ".svg",
	"image/avif":               ".avif",
	"text/plain":               ".txt",
	"text/html":                ".html",
	"text/csv":                 ".csv",
	"text/markdown":            ".md",
	"application/json":         ".json",
	"application/pdf":          ".pdf",
	"application/octet-stream": ".bin",
}

// ExtensionForMIME returns a file extension (with the dot) for a MIME type.
// Common types map to fixed extensions so file names are deterministic;
// unknown types use the system table (first extension in sorted order) and
// fall back to ".bin".
func ExtensionForMIME(mimeType string) string {
	base := strings.ToLower(strings.TrimSpace(mimeType))
	if i := strings.IndexByte(base, ';'); i >= 0 {
		base = strings.TrimSpace(base[:i])
	}
	if ext, ok := knownExtensions[base]; ok {
		return ext
	}
	if exts, err := mime.ExtensionsByType(base); err == nil && len(exts) > 0 {
		sort.Strings(exts)
		return exts[0]
	}
	return ".bin"
}

// WriteFiles writes every delivered item of rendering into dir (created if
// missing) and returns the written files in rendering order. Names are
// deterministic: "<prefix>-<N><ext>" when provider N delivered one item and
// "<prefix>-<N>-<K><ext>" when it delivered several. Existing files with the
// same names are overwritten. An empty prefix selects DefaultFilePrefix.
func WriteFiles(ctx context.Context, rendering Rendering, dir string, prefix string) ([]WrittenFile, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil, errors.New("resultcontent: content directory is empty")
	}
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		prefix = DefaultFilePrefix
	}
	if strings.ContainsAny(prefix, `/\`) {
		return nil, errors.Errorf("resultcontent: file prefix %q must not contain path separators", prefix)
	}
	delivered := map[int]int{}
	for _, ref := range rendering.Refs {
		if ref.Index >= 0 && ref.Index < len(rendering.Content) {
			delivered[ref.Provider]++
		}
	}
	if len(delivered) == 0 {
		return []WrittenFile{}, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, errors.Wrapf(err, "resultcontent: create content directory %s", dir)
	}
	written := make([]WrittenFile, 0, len(rendering.Content))
	seen := map[int]int{}
	for _, ref := range rendering.Refs {
		if ref.Index < 0 || ref.Index >= len(rendering.Content) {
			continue
		}
		if err := ctx.Err(); err != nil {
			return written, err
		}
		item := rendering.Content[ref.Index]
		seen[ref.Provider]++
		name := fmt.Sprintf("%s-%d", prefix, ref.Provider)
		if delivered[ref.Provider] > 1 {
			name = fmt.Sprintf("%s-%d", name, seen[ref.Provider])
		}
		path := filepath.Join(dir, name+ExtensionForMIME(item.MIMEType))
		if err := os.WriteFile(path, item.Data, 0o644); err != nil {
			return written, errors.Wrapf(err, "resultcontent: write %s", path)
		}
		written = append(written, WrittenFile{
			Provider: ref.Provider,
			Item:     seen[ref.Provider],
			Path:     path,
			MIMEType: item.MIMEType,
			Caption:  item.Caption,
			Bytes:    len(item.Data),
		})
	}
	return written, nil
}

// PathsByProvider groups written file paths by provider number.
func PathsByProvider(files []WrittenFile) map[int][]string {
	out := map[int][]string{}
	for _, f := range files {
		out[f.Provider] = append(out[f.Provider], f.Path)
	}
	return out
}

// RenderToDir renders providers and writes the delivered items into dir.
func RenderToDir(ctx context.Context, providers []Provider, limits Limits, dir string, prefix string) (Rendering, []WrittenFile, error) {
	rendering := Render(ctx, providers, limits)
	files, err := WriteFiles(ctx, rendering, dir, prefix)
	return rendering, files, err
}
