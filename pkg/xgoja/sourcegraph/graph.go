package sourcegraph

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

type SourceKind string

const (
	SourceKindJSVerbs SourceKind = "jsverbs"
	SourceKindScript  SourceKind = "script"
	SourceKindAssets  SourceKind = "assets"
	SourceKindHelp    SourceKind = "help"
)

type OriginKind string

const (
	OriginDisk     OriginKind = "disk"
	OriginProvider OriginKind = "provider"
	OriginEmbedded OriginKind = "embedded"
)

type Origin struct {
	Kind     OriginKind
	Dir      string
	FS       fs.FS
	Root     string
	Provider string
	Source   string
}

type SourceSet struct {
	ID         string
	Kind       SourceKind
	Origin     Origin
	Include    []string
	Exclude    []string
	Extensions []string
	Language   string
}

type File struct {
	SourceSetID string
	Kind        SourceKind
	Origin      Origin
	OriginKind  OriginKind
	Path        string
	AbsPath     string
}

type ImportKind string

const (
	ImportLocal   ImportKind = "local"
	ImportRuntime ImportKind = "runtime"
	ImportUnknown ImportKind = "unknown"
)

type ImportResolution struct {
	From       string
	Specifier  string
	Kind       ImportKind
	TargetPath string
	Alias      string
}

type Graph struct {
	files   map[string]File
	bySet   map[string][]File
	aliases map[string]bool
	imports map[string][]ImportResolution
}

type Options struct {
	RuntimeModuleAliases []string
}

func Build(sources []SourceSet, opts Options) (*Graph, error) {
	graph := &Graph{files: map[string]File{}, bySet: map[string][]File{}, aliases: map[string]bool{}, imports: map[string][]ImportResolution{}}
	for _, alias := range opts.RuntimeModuleAliases {
		alias = strings.TrimSpace(alias)
		if alias != "" {
			graph.aliases[alias] = true
		}
	}
	for _, source := range sources {
		files, err := discoverSourceSet(source)
		if err != nil {
			return nil, err
		}
		for _, file := range files {
			key := fileKey(file)
			if _, exists := graph.files[key]; exists {
				return nil, fmt.Errorf("duplicate source file %q", key)
			}
			graph.files[key] = file
			graph.bySet[file.SourceSetID] = append(graph.bySet[file.SourceSetID], file)
		}
	}
	for id := range graph.bySet {
		sort.Slice(graph.bySet[id], func(i, j int) bool { return graph.bySet[id][i].Path < graph.bySet[id][j].Path })
	}
	return graph, nil
}

func (g *Graph) Files() []File {
	if g == nil {
		return nil
	}
	keys := make([]string, 0, len(g.files))
	for key := range g.files {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]File, 0, len(keys))
	for _, key := range keys {
		out = append(out, g.files[key])
	}
	return out
}

func (g *Graph) FilesForSourceSet(id string) []File {
	if g == nil {
		return nil
	}
	return append([]File(nil), g.bySet[id]...)
}

func (g *Graph) ResolveImports(readFile func(File) ([]byte, error)) error {
	if g == nil {
		return nil
	}
	for _, file := range g.Files() {
		if !isExecutableKind(file.Kind) || !isJSLike(file.Path) {
			continue
		}
		data, err := readFile(file)
		if err != nil {
			return err
		}
		resolutions, err := g.resolveFileImports(file, string(data))
		if err != nil {
			return err
		}
		g.imports[fileKey(file)] = resolutions
	}
	return nil
}

func (g *Graph) ImportResolutions(file File) []ImportResolution {
	if g == nil {
		return nil
	}
	return append([]ImportResolution(nil), g.imports[fileKey(file)]...)
}

func (g *Graph) resolveFileImports(file File, contents string) ([]ImportResolution, error) {
	imports, err := parseImports(file.Path, []byte(contents))
	if err != nil {
		return nil, err
	}
	out := make([]ImportResolution, 0, len(imports))
	for _, imp := range imports {
		if imp.Dynamic {
			return nil, fmt.Errorf("%s contains dynamic non-literal %s import", file.Path, imp.Kind)
		}
		specifier := imp.Specifier
		if strings.HasPrefix(specifier, ".") {
			target, err := g.resolveLocal(file, specifier)
			if err != nil {
				return nil, err
			}
			out = append(out, ImportResolution{From: file.Path, Specifier: specifier, Kind: ImportLocal, TargetPath: target.Path})
			continue
		}
		if g.aliases[specifier] {
			out = append(out, ImportResolution{From: file.Path, Specifier: specifier, Kind: ImportRuntime, Alias: specifier})
			continue
		}
		return nil, fmt.Errorf("%s imports unknown bare specifier %q", file.Path, specifier)
	}
	return out, nil
}

func (g *Graph) resolveLocal(from File, specifier string) (File, error) {
	base := path.Clean(path.Join(path.Dir(from.Path), specifier))
	candidates := []string{base}
	for _, ext := range []string{".ts", ".tsx", ".mts", ".cts", ".js", ".jsx", ".mjs", ".cjs"} {
		candidates = append(candidates, base+ext)
	}
	for _, index := range []string{"index.ts", "index.tsx", "index.js", "index.jsx"} {
		candidates = append(candidates, path.Join(base, index))
	}
	for _, candidate := range candidates {
		if strings.HasPrefix(candidate, "../") || candidate == ".." {
			return File{}, fmt.Errorf("%s imports %q outside source root", from.Path, specifier)
		}
		key := from.SourceSetID + "\x00" + candidate
		if target, ok := g.files[key]; ok {
			return target, nil
		}
	}
	return File{}, fmt.Errorf("%s imports %q but no matching source file was found", from.Path, specifier)
}

func discoverSourceSet(source SourceSet) ([]File, error) {
	id := strings.TrimSpace(source.ID)
	if id == "" {
		return nil, fmt.Errorf("source set id is required")
	}
	switch source.Origin.Kind {
	case OriginDisk:
		return discoverDisk(source)
	case OriginProvider, OriginEmbedded:
		return discoverFS(source)
	default:
		return nil, fmt.Errorf("unsupported source origin %q", source.Origin.Kind)
	}
}

func discoverDisk(source SourceSet) ([]File, error) {
	root := strings.TrimSpace(source.Origin.Dir)
	if root == "" {
		return nil, fmt.Errorf("source set %s disk dir is required", source.ID)
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	out := []File{}
	err = filepath.WalkDir(rootAbs, func(absPath string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		rel, err := filepath.Rel(rootAbs, absPath)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if !includePath(rel, source) {
			return nil
		}
		out = append(out, File{SourceSetID: source.ID, Kind: source.Kind, Origin: source.Origin, OriginKind: OriginDisk, Path: rel, AbsPath: absPath})
		return nil
	})
	return out, err
}

func discoverFS(source SourceSet) ([]File, error) {
	if source.Origin.FS == nil {
		return nil, fmt.Errorf("source set %s fs is required", source.ID)
	}
	root := strings.Trim(strings.TrimSpace(source.Origin.Root), "/")
	if root == "" {
		root = "."
	}
	out := []File{}
	err := fs.WalkDir(source.Origin.FS, root, func(fsPath string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		rel := fsPath
		if root != "." {
			var ok bool
			rel, ok = strings.CutPrefix(fsPath, root+"/")
			if !ok {
				return nil
			}
		}
		if !includePath(rel, source) {
			return nil
		}
		out = append(out, File{SourceSetID: source.ID, Kind: source.Kind, Origin: source.Origin, OriginKind: source.Origin.Kind, Path: rel})
		return nil
	})
	return out, err
}

func includePath(rel string, source SourceSet) bool {
	rel = path.Clean(filepath.ToSlash(rel))
	if len(source.Extensions) > 0 {
		ok := false
		for _, ext := range source.Extensions {
			if strings.EqualFold(path.Ext(rel), strings.TrimSpace(ext)) {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	if len(source.Include) > 0 {
		matched := false
		for _, pattern := range source.Include {
			if ok, _ := doublestar.Match(pattern, rel); ok {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	for _, pattern := range source.Exclude {
		if ok, _ := doublestar.Match(pattern, rel); ok {
			return false
		}
	}
	return true
}

func fileKey(file File) string { return file.SourceSetID + "\x00" + file.Path }

func isExecutableKind(kind SourceKind) bool {
	return kind == SourceKindJSVerbs || kind == SourceKindScript
}
func isJSLike(p string) bool {
	switch strings.ToLower(path.Ext(p)) {
	case ".js", ".jsx", ".mjs", ".cjs", ".ts", ".tsx", ".mts", ".cts":
		return true
	default:
		return false
	}
}
