//go:build ignore
// +build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

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
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
	if err != nil {
		fatalf("connect dagger: %v", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			log.Printf("close dagger client: %v", err)
		}
	}()

	src := client.Host().Directory(projectDir)
	cache := client.CacheVolume(npmCacheVolumeName)
	ctr := client.Container().
		From(nodeImage).
		WithMountedDirectory("/src", src).
		WithMountedCache("/root/.npm", cache).
		WithWorkdir("/src/js").
		WithEnvVariable("CI", "1").
		WithExec([]string{"npm", "install", "--no-audit", "--no-fund"}).
		WithExec([]string{"npm", "run", "build:split"})

	dist := ctr.Directory("/src/js/dist-split")
	if _, err := dist.Export(ctx, outDir); err != nil {
		fatalf("export dist: %v", err)
	}

	log.Printf("exported split assets to %s", outDir)
}

func fatalf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
