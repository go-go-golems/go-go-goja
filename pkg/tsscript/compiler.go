package tsscript

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
)

// TransformSource transpiles one source string without following imports.
func TransformSource(src Source, opts Options) (*Artifact, error) {
	path := sourcePath(src)
	loader := LoaderForPath(path)
	result := api.Transform(string(src.Contents), api.TransformOptions{
		LogLevel:    defaultLogLevel(opts.LogLevel),
		Sourcemap:   opts.Sourcemap,
		SourceRoot:  opts.SourceRoot,
		Target:      defaultTarget(opts.Target),
		Platform:    defaultPlatform(opts.Platform),
		Format:      defaultTransformFormat(opts.Format),
		JSX:         opts.JSX,
		Define:      cloneStringMap(opts.Define),
		TsconfigRaw: opts.TsconfigRaw,
		Sourcefile:  path,
		Loader:      loader,
	})
	if len(result.Errors) > 0 {
		return nil, errorFromMessages("typescript transform", result.Errors)
	}
	return &Artifact{
		Path:       path,
		Code:       append([]byte(nil), result.Code...),
		SourceMap:  append([]byte(nil), result.Map...),
		Warnings:   diagnosticsFromMessages(result.Warnings),
		LoaderUsed: loader,
	}, nil
}

// BundleEntry bundles an entry point and its dependency graph into one output file.
func BundleEntry(entryPath string, opts Options) (*Artifact, error) {
	entryPath = strings.TrimSpace(entryPath)
	if entryPath == "" {
		return nil, fmt.Errorf("typescript bundle entry path is required")
	}
	absWorkingDir := ""
	if abs, err := filepath.Abs(filepath.Dir(entryPath)); err == nil {
		absWorkingDir = abs
	}
	result := api.Build(api.BuildOptions{
		LogLevel:      defaultLogLevel(opts.LogLevel),
		EntryPoints:   []string{entryPath},
		Bundle:        true,
		Write:         false,
		Sourcemap:     opts.Sourcemap,
		SourceRoot:    opts.SourceRoot,
		Target:        defaultTarget(opts.Target),
		Platform:      defaultPlatform(opts.Platform),
		Format:        defaultBundleFormat(opts.Format),
		External:      append([]string(nil), opts.External...),
		Define:        cloneStringMap(opts.Define),
		Tsconfig:      opts.Tsconfig,
		AbsWorkingDir: absWorkingDir,
		Loader:        defaultLoaders(),
	})
	return artifactFromBuildResult("typescript bundle", entryPath, result)
}

// BundleVirtualEntry bundles source provided in memory and resolves relative
// imports from Source.ResolveDir.
func BundleVirtualEntry(src Source, opts Options) (*Artifact, error) {
	path := sourcePath(src)
	loader := LoaderForPath(path)
	result := api.Build(api.BuildOptions{
		LogLevel:   defaultLogLevel(opts.LogLevel),
		Stdin:      &api.StdinOptions{Contents: string(src.Contents), ResolveDir: src.ResolveDir, Sourcefile: path, Loader: loader},
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
	})
	artifact, err := artifactFromBuildResult("typescript bundle", path, result)
	if artifact != nil {
		artifact.LoaderUsed = loader
	}
	return artifact, err
}

func artifactFromBuildResult(op string, path string, result api.BuildResult) (*Artifact, error) {
	if len(result.Errors) > 0 {
		return nil, errorFromMessages(op, result.Errors)
	}
	if len(result.OutputFiles) == 0 {
		return nil, fmt.Errorf("%s failed: no output files", op)
	}
	artifact := &Artifact{Path: path, Warnings: diagnosticsFromMessages(result.Warnings), Bundled: true}
	for _, out := range result.OutputFiles {
		if strings.HasSuffix(out.Path, ".map") {
			artifact.SourceMap = append([]byte(nil), out.Contents...)
			continue
		}
		if len(artifact.Code) == 0 {
			artifact.Code = append([]byte(nil), out.Contents...)
		}
	}
	if len(artifact.Code) == 0 {
		artifact.Code = append([]byte(nil), result.OutputFiles[0].Contents...)
	}
	return artifact, nil
}

func defaultLoaders() map[string]api.Loader {
	return map[string]api.Loader{
		".js":   api.LoaderJS,
		".cjs":  api.LoaderJS,
		".mjs":  api.LoaderJS,
		".jsx":  api.LoaderJSX,
		".ts":   api.LoaderTS,
		".tsx":  api.LoaderTSX,
		".mts":  api.LoaderTS,
		".cts":  api.LoaderTS,
		".json": api.LoaderJSON,
	}
}

func sourcePath(src Source) string {
	if strings.TrimSpace(src.Path) != "" {
		return strings.TrimSpace(src.Path)
	}
	if strings.TrimSpace(src.AbsPath) != "" {
		return strings.TrimSpace(src.AbsPath)
	}
	return "stdin.ts"
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
