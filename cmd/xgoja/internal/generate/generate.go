package generate

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/buildspec"
)

type Options struct {
	XGojaModuleVersion string
	XGojaReplace       string
}

type PackageOptions struct {
	PackageName string
}

func defaultOptions() Options {
	return Options{XGojaModuleVersion: "v0.0.0"}
}

func WriteAll(dir string, buildSpec *buildspec.BuildSpec, opts Options) error {
	if dir == "" {
		return fmt.Errorf("generate directory is required")
	}
	if buildSpec == nil {
		return fmt.Errorf("BuildSpec is nil")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create generate directory %s: %w", dir, err)
	}
	if err := copyEmbeddedJSVerbs(dir, buildSpec); err != nil {
		return err
	}
	if err := copyEmbeddedHelpSources(dir, buildSpec); err != nil {
		return err
	}
	if err := copyEmbeddedAssets(dir, buildSpec); err != nil {
		return err
	}
	files := map[string]string{
		"go.mod":         RenderGoMod(buildSpec, opts),
		"main.go":        RenderMain(buildSpec),
		"xgoja.gen.json": RenderEmbeddedSpec(buildSpec),
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			return fmt.Errorf("write generated %s: %w", name, err)
		}
	}
	return nil
}

func WritePackage(dir string, buildSpec *buildspec.BuildSpec, opts PackageOptions) error {
	if dir == "" {
		return fmt.Errorf("generate directory is required")
	}
	if buildSpec == nil {
		return fmt.Errorf("BuildSpec is nil")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create generate directory %s: %w", dir, err)
	}
	if err := copyEmbeddedJSVerbs(dir, buildSpec); err != nil {
		return err
	}
	if err := copyEmbeddedHelpSources(dir, buildSpec); err != nil {
		return err
	}
	if err := copyEmbeddedAssets(dir, buildSpec); err != nil {
		return err
	}
	packageName := strings.TrimSpace(opts.PackageName)
	if packageName == "" {
		packageName = packageNameFromDir(dir)
	}
	content := RenderPackage(buildSpec, packageName)
	if err := os.WriteFile(filepath.Join(dir, "xgoja_runtime.gen.go"), []byte(content), 0o644); err != nil {
		return fmt.Errorf("write generated package: %w", err)
	}
	return nil
}

func packageNameFromDir(dir string) string {
	base := filepath.Base(filepath.Clean(dir))
	name := sanitizeIdentifier(base)
	if name == "" || name == "internal" {
		return "xgojaruntime"
	}
	return name
}

func copyEmbeddedJSVerbs(dir string, buildSpec *buildspec.BuildSpec) error {
	roots := embeddedJSVerbRoots(buildSpec)
	for i, source := range buildSpec.JSVerbs {
		root := roots[i]
		if root == "" {
			continue
		}
		src, err := resolveSourcePath(buildSpec.BaseDir, source.Path)
		if err != nil {
			return fmt.Errorf("resolve embedded jsverb source %s: %w", source.ID, err)
		}
		dst := filepath.Join(dir, filepath.FromSlash(root))
		if err := copyDir(dst, src); err != nil {
			return fmt.Errorf("copy embedded jsverb source %s: %w", source.ID, err)
		}
	}
	return nil
}

func copyEmbeddedHelpSources(dir string, buildSpec *buildspec.BuildSpec) error {
	roots := embeddedHelpRoots(buildSpec)
	for i, source := range buildSpec.Help.Sources {
		root := roots[i]
		if root == "" {
			continue
		}
		src, err := resolveSourcePath(buildSpec.BaseDir, source.Path)
		if err != nil {
			return fmt.Errorf("resolve embedded help source %s: %w", source.ID, err)
		}
		dst := filepath.Join(dir, filepath.FromSlash(root))
		if err := copyDir(dst, src); err != nil {
			return fmt.Errorf("copy embedded help source %s: %w", source.ID, err)
		}
	}
	return nil
}

func copyEmbeddedAssets(dir string, buildSpec *buildspec.BuildSpec) error {
	roots := embeddedAssetRoots(buildSpec)
	for i, source := range buildSpec.Assets {
		root := roots[i]
		if root == "" {
			continue
		}
		src, err := resolveSourcePath(buildSpec.BaseDir, source.Path)
		if err != nil {
			return fmt.Errorf("resolve embedded asset source %s: %w", source.ID, err)
		}
		dst := filepath.Join(dir, filepath.FromSlash(root))
		if err := copyDirWithOptions(dst, src, copyDirOptions{skipNodeModules: true}); err != nil {
			return fmt.Errorf("copy embedded asset source %s: %w", source.ID, err)
		}
	}
	return nil
}

func resolveSourcePath(baseDir, rawPath string) (string, error) {
	path := strings.TrimSpace(rawPath)
	if path == "" {
		return "", fmt.Errorf("path is empty")
	}
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		path = filepath.Join(home, strings.TrimPrefix(path, "~/"))
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(baseDir, path)
	}
	return filepath.Clean(path), nil
}

type copyDirOptions struct {
	skipDotDirs     bool
	skipNodeModules bool
}

func copyDir(dst, src string) error {
	return copyDirWithOptions(dst, src, copyDirOptions{skipDotDirs: true, skipNodeModules: true})
}

func copyDirWithOptions(dst, src string, opts copyDirOptions) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", src)
	}
	return filepath.WalkDir(src, func(srcPath string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, srcPath)
		if err != nil {
			return err
		}
		if rel == "." {
			return os.MkdirAll(dst, 0o755)
		}
		if d.IsDir() {
			name := d.Name()
			if opts.skipNodeModules && name == "node_modules" {
				return filepath.SkipDir
			}
			if opts.skipDotDirs && strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return os.MkdirAll(filepath.Join(dst, rel), 0o755)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		return copyFile(filepath.Join(dst, rel), srcPath, info.Mode().Perm())
	})
}

func copyFile(dst, src string, perm fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()
	_, err = io.Copy(out, in)
	return err
}
