//go:build ignore
// +build ignore

package main

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"dagger.io/dagger"
)

const (
	nodeImage          = "node:20.18.1"
	npmCacheVolumeName = "bun-demo-npm-cache"
)

func main() {
	projectDir, err := os.Getwd()
	if err != nil {
		fatalf("get working dir: %v", err)
	}

	jsDir := filepath.Join(projectDir, "js")
	if _, err := os.Stat(filepath.Join(jsDir, "package.json")); err != nil {
		fatalf("js workspace missing at %s: %v", jsDir, err)
	}

	outDir := filepath.Join(projectDir, "assets-split")
	if err := os.RemoveAll(outDir); err != nil {
		fatalf("remove %s: %v", outDir, err)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fatalf("mkdir %s: %v", outDir, err)
	}

	ctx := context.Background()

	var daggerErr error
	if useDagger() {
		daggerErr = buildWithDagger(ctx, projectDir, outDir)
		if daggerErr == nil {
			log.Printf("exported split assets to %s (dagger)", outDir)
			return
		}
		log.Printf("dagger build failed, falling back to local npm build: %v", daggerErr)
	} else {
		log.Printf("skipping dagger build (BUN_DEMO_GENERATE_NO_DAGGER set)")
	}

	if err := buildWithLocalNPM(jsDir, outDir); err != nil {
		if daggerErr != nil {
			fatalf("dagger build failed: %v; local npm fallback failed: %v", daggerErr, err)
		}
		fatalf("local npm build failed: %v", err)
	}
	log.Printf("exported split assets to %s (local npm fallback)", outDir)
}

func useDagger() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("BUN_DEMO_GENERATE_NO_DAGGER")))
	return v != "1" && v != "true" && v != "yes"
}

func buildWithDagger(ctx context.Context, projectDir, outDir string) error {
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
	if err != nil {
		return fmt.Errorf("connect dagger: %w", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			log.Printf("close dagger client: %v", err)
		}
	}()

	src := client.Host().Directory(projectDir)
	cache := client.CacheVolume(npmCacheVolumeName)
	jsDir := filepath.Join(projectDir, "js")
	installArgs := npmInstallArgs(jsDir)
	image := nodeImage
	if v := strings.TrimSpace(os.Getenv("BUN_DEMO_BUILDER_IMAGE")); v != "" {
		image = v
	}
	ctr := client.Container().
		From(image).
		WithMountedDirectory("/src", src).
		WithMountedCache("/root/.npm", cache).
		WithWorkdir("/src/js").
		WithEnvVariable("CI", "1").
		WithExec(installArgs).
		WithExec([]string{"npm", "run", "build:split"})

	dist := ctr.Directory("/src/js/dist-split")
	if _, err := dist.Export(ctx, outDir); err != nil {
		return fmt.Errorf("export dist: %w", err)
	}
	return nil
}

func buildWithLocalNPM(jsDir, outDir string) error {
	if _, err := exec.LookPath("npm"); err != nil {
		return fmt.Errorf("npm is required for local fallback but was not found in PATH: %w", err)
	}

	installArgs := npmInstallArgs(jsDir)
	if err := runCommand(jsDir, installArgs[0], installArgs[1:]...); err != nil {
		return err
	}
	if err := runCommand(jsDir, "npm", "run", "build:split"); err != nil {
		return err
	}

	distDir := filepath.Join(jsDir, "dist-split")
	if _, err := os.Stat(distDir); err != nil {
		return fmt.Errorf("expected local build output at %s: %w", distDir, err)
	}
	if err := copyDirContents(distDir, outDir); err != nil {
		return fmt.Errorf("copy %s to %s: %w", distDir, outDir, err)
	}
	return nil
}

func npmInstallArgs(jsDir string) []string {
	lockPath := filepath.Join(jsDir, "package-lock.json")
	if _, err := os.Stat(lockPath); err == nil {
		return []string{"npm", "ci", "--no-audit", "--no-fund"}
	}
	// Keep generator side-effect free for repos that don't track package-lock.json.
	return []string{"npm", "install", "--no-audit", "--no-fund", "--package-lock=false"}
}

func runCommand(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "CI=1")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("run %s %s in %s: %w\n%s", name, strings.Join(args, " "), dir, err, strings.TrimSpace(string(output)))
	}
	return nil
}

func copyDirContents(srcDir, dstDir string) error {
	return filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		dstPath := filepath.Join(dstDir, rel)
		if d.IsDir() {
			return os.MkdirAll(dstPath, 0o755)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		return copyFile(path, dstPath, info.Mode().Perm())
	})
}

func copyFile(srcPath, dstPath string, perm fs.FileMode) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return err
	}
	dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return err
	}
	return nil
}

func fatalf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
