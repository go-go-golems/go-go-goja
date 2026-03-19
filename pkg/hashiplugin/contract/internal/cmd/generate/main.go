package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	protocGenGoModule     = "google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11"
	protocGenGoGRPCModule = "google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.6.0"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "hashiplugin contract generate: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	if _, err := exec.LookPath("protoc"); err != nil {
		fmt.Fprintln(os.Stderr, "hashiplugin contract generate: skipping because protoc is not installed")
		return nil
	}

	toolDir, err := os.MkdirTemp("", "goja-protoc-tools-*")
	if err != nil {
		return fmt.Errorf("create temp tool dir: %w", err)
	}
	defer func() {
		if removeErr := os.RemoveAll(toolDir); removeErr != nil {
			fmt.Fprintf(os.Stderr, "hashiplugin contract generate: clean temp tools: %v\n", removeErr)
		}
	}()

	if err := installTool(toolDir, protocGenGoModule); err != nil {
		return err
	}
	if err := installTool(toolDir, protocGenGoGRPCModule); err != nil {
		return err
	}

	cmd := exec.Command(
		"protoc",
		"--proto_path=.",
		"--go_out=.",
		"--go_opt=paths=source_relative",
		"--go-grpc_out=.",
		"--go-grpc_opt=paths=source_relative",
		"jsmodule.proto",
	)
	cmd.Dir = "."
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "PATH="+toolDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run protoc: %w", err)
	}
	return nil
}

func installTool(toolDir, pkg string) error {
	cmd := exec.Command("go", "install", pkg)
	cmd.Dir = "."
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "GOBIN="+filepath.Clean(toolDir))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("install %s: %w", pkg, err)
	}
	return nil
}
