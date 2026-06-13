package tsscript

import (
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
)

// BundleVirtualEntryFS bundles an in-memory entry source and resolves relative
// imports from root. It is intended for embedded/provider sources loaded from
// fs.FS, where there is no useful on-disk ResolveDir for esbuild.
func BundleVirtualEntryFS(root fs.FS, src Source, opts Options) (*Artifact, error) {
	if root == nil {
		return nil, fmt.Errorf("typescript fs bundle root is required")
	}
	entryPath := cleanVirtualPath(sourcePath(src))
	loader := LoaderForPath(entryPath)
	plugin := fsResolverPlugin(root)
	result := api.Build(api.BuildOptions{
		LogLevel:   defaultLogLevel(opts.LogLevel),
		Stdin:      &api.StdinOptions{Contents: string(src.Contents), ResolveDir: "/" + path.Dir(entryPath), Sourcefile: entryPath, Loader: loader},
		Bundle:     true,
		Write:      false,
		Sourcemap:  opts.Sourcemap,
		SourceRoot: opts.SourceRoot,
		Target:     defaultTarget(opts.Target),
		Platform:   defaultPlatform(opts.Platform),
		Format:     defaultBundleFormat(opts.Format),
		External:   append([]string(nil), opts.External...),
		Define:     cloneStringMap(opts.Define),
		Tsconfig:   opts.Tsconfig,
		Loader:     defaultLoaders(),
		Plugins:    []api.Plugin{plugin},
	})
	artifact, err := artifactFromBuildResult("typescript fs bundle", entryPath, result)
	if artifact != nil {
		artifact.LoaderUsed = loader
	}
	return artifact, err
}

func fsResolverPlugin(root fs.FS) api.Plugin {
	return api.Plugin{
		Name: "xgoja-fs-resolver",
		Setup: func(build api.PluginBuild) {
			build.OnResolve(api.OnResolveOptions{Filter: `^\.`}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
				baseDir := cleanVirtualPath(args.ResolveDir)
				candidate := cleanVirtualPath(path.Join(baseDir, args.Path))
				resolved, ok := resolveFSPath(root, candidate)
				if !ok {
					return api.OnResolveResult{}, fmt.Errorf("resolve %q from %q: no matching fs source", args.Path, baseDir)
				}
				return api.OnResolveResult{Path: "/" + resolved, Namespace: "xgoja-fs"}, nil
			})
			build.OnLoad(api.OnLoadOptions{Filter: `.*`, Namespace: "xgoja-fs"}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
				p := cleanVirtualPath(args.Path)
				data, err := fs.ReadFile(root, p)
				if err != nil {
					return api.OnLoadResult{}, err
				}
				contents := string(data)
				loader := LoaderForPath(p)
				return api.OnLoadResult{Contents: &contents, Loader: loader, ResolveDir: "/" + path.Dir(p)}, nil
			})
		},
	}
}

func resolveFSPath(root fs.FS, candidate string) (string, bool) {
	candidate = cleanVirtualPath(candidate)
	candidates := []string{candidate}
	for _, ext := range []string{".ts", ".tsx", ".mts", ".cts", ".js", ".jsx", ".mjs", ".cjs", ".json"} {
		candidates = append(candidates, candidate+ext)
	}
	for _, index := range []string{"index.ts", "index.tsx", "index.js", "index.jsx"} {
		candidates = append(candidates, path.Join(candidate, index))
	}
	for _, p := range candidates {
		if p == "." || strings.HasPrefix(p, "../") || p == ".." {
			continue
		}
		if _, err := fs.Stat(root, p); err == nil {
			return p, true
		}
	}
	return "", false
}

func cleanVirtualPath(p string) string {
	p = strings.TrimSpace(strings.ReplaceAll(p, "\\", "/"))
	p = strings.TrimPrefix(p, "/")
	if p == "" {
		return "."
	}
	return path.Clean(p)
}
