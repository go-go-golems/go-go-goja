package generate

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/workspace"
)

type Options struct {
	XGojaModuleVersion string
	XGojaReplace       string
	GoModules          *workspace.Plan
}

type PackageOptions struct {
	PackageName string
}

type TemplateOptions struct {
	PackageName  string
	TemplatePath string
}

func CleanGenerated(dir string) error {
	if strings.TrimSpace(dir) == "" {
		return fmt.Errorf("generate directory is required")
	}
	for _, name := range []string{
		"xgoja_runtime.gen.go",
		"runtime_plan.gen.go",
		"providers.gen.go",
		"bundle.gen.go",
		"embed.gen.go",
		"xgoja_embed",
	} {
		path := filepath.Join(dir, name)
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("remove generated %s: %w", path, err)
		}
	}
	return nil
}

func CleanGeneratedFile(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("generated file path is required")
	}
	base := filepath.Base(path)
	if !strings.HasSuffix(base, ".gen.go") {
		return fmt.Errorf("refusing to clean non-generated file %s", path)
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove generated file %s: %w", path, err)
	}
	return nil
}

func defaultOptions() Options {
	return Options{XGojaModuleVersion: "v0.0.0"}
}

func InferPackageNameFromDir(dir string) string {
	base := filepath.Base(filepath.Clean(dir))
	name := sanitizeIdentifier(base)
	if name == "" || name == "internal" {
		return "xgojaruntime"
	}
	return name
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
