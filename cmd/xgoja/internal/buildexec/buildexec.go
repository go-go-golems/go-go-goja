package buildexec

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sort"
)

type Result struct {
	Command string
	Output  string
}

func GoModTidy(ctx context.Context, dir string) (Result, error) {
	return run(ctx, dir, nil, "go", "mod", "tidy")
}

func GoBuild(ctx context.Context, dir string, output string, tags []string, ldflags []string, env map[string]string) (Result, error) {
	args := []string{"build", "-o", output}
	if len(tags) > 0 {
		args = append(args, "-tags", joinSpace(tags))
	}
	if len(ldflags) > 0 {
		args = append(args, "-ldflags", joinSpace(ldflags))
	}
	args = append(args, ".")
	return run(ctx, dir, env, "go", args...)
}

func run(ctx context.Context, dir string, env map[string]string, name string, args ...string) (Result, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), sortedEnv(env)...)
	}
	out, err := cmd.CombinedOutput()
	result := Result{Command: commandString(env, name, args), Output: string(out)}
	if err != nil {
		return result, fmt.Errorf("%s failed: %w\n%s", result.Command, err, result.Output)
	}
	return result, nil
}

func commandString(env map[string]string, name string, args []string) string {
	cmd := name + " " + joinSpace(args)
	if len(env) == 0 {
		return cmd
	}
	return joinSpace(sortedEnv(env)) + " " + cmd
}

func sortedEnv(env map[string]string) []string {
	keys := make([]string, 0, len(env))
	for key := range env {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, key+"="+env[key])
	}
	return out
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
