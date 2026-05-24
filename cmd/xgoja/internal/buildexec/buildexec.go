package buildexec

import (
	"context"
	"fmt"
	"os/exec"
)

type Result struct {
	Command string
	Output  string
}

func GoModTidy(ctx context.Context, dir string) (Result, error) {
	return run(ctx, dir, "go", "mod", "tidy")
}

func GoBuild(ctx context.Context, dir string, output string, tags []string, ldflags []string) (Result, error) {
	args := []string{"build", "-o", output}
	if len(tags) > 0 {
		args = append(args, "-tags", joinSpace(tags))
	}
	if len(ldflags) > 0 {
		args = append(args, "-ldflags", joinSpace(ldflags))
	}
	args = append(args, ".")
	return run(ctx, dir, "go", args...)
}

func run(ctx context.Context, dir string, name string, args ...string) (Result, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	result := Result{Command: name + " " + joinSpace(args), Output: string(out)}
	if err != nil {
		return result, fmt.Errorf("%s failed: %w\n%s", result.Command, err, result.Output)
	}
	return result, nil
}

func joinSpace(items []string) string {
	if len(items) == 0 {
		return ""
	}
	out := items[0]
	for _, item := range items[1:] {
		out += " " + item
	}
	return out
}
