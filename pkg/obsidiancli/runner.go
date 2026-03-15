package obsidiancli

import (
	"bytes"
	"context"
	stderrors "errors"
	"io"
	osexec "os/exec"
	"sync"

	"github.com/pkg/errors"
)

// Invocation captures one concrete subprocess execution.
type Invocation struct {
	Binary string
	Args   []string
	Dir    string
	Env    []string
}

// ExecResult contains raw subprocess outputs.
type ExecResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// Result contains raw and parsed command output.
type Result struct {
	Stdout   string
	Stderr   string
	Parsed   any
	ExitCode int
}

// Executor runs a subprocess invocation.
type Executor interface {
	Run(ctx context.Context, inv Invocation) (ExecResult, error)
}

// Runner serializes access to the Obsidian CLI for one configured client.
type Runner struct {
	cfg      Config
	executor Executor

	mu sync.Mutex
}

// NewRunner creates a new serialized Obsidian CLI runner.
func NewRunner(cfg Config, executor Executor) *Runner {
	defaults := DefaultConfig()
	if cfg.BinaryPath == "" {
		cfg.BinaryPath = defaults.BinaryPath
	}
	if executor == nil {
		executor = osExecutor{}
	}
	return &Runner{
		cfg:      cfg,
		executor: executor,
	}
}

// Run executes one command and parses stdout according to the spec.
func (r *Runner) Run(ctx context.Context, spec CommandSpec, call CallOptions) (Result, error) {
	if r == nil {
		return Result{}, errors.New("obsidiancli: runner is nil")
	}
	args, err := BuildArgs(r.cfg, spec, call)
	if err != nil {
		return Result{}, err
	}

	inv := Invocation{
		Binary: r.cfg.BinaryPath,
		Args:   args,
		Dir:    r.cfg.WorkingDir,
		Env:    append([]string(nil), r.cfg.Env...),
	}
	runCtx := ctx
	var cancel context.CancelFunc
	if runCtx == nil {
		runCtx = context.Background()
	}
	if r.cfg.Timeout > 0 {
		if _, ok := runCtx.Deadline(); !ok {
			runCtx, cancel = context.WithTimeout(runCtx, r.cfg.Timeout)
			defer cancel()
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	execResult, err := r.executor.Run(runCtx, inv)
	if err != nil {
		if isBinaryPathError(err) {
			return Result{}, &BinaryNotFoundError{Path: inv.Binary}
		}
		return Result{}, &CommandError{
			Spec:     spec,
			Args:     append([]string(nil), args...),
			Stdout:   execResult.Stdout,
			Stderr:   execResult.Stderr,
			ExitCode: execResult.ExitCode,
			Err:      err,
		}
	}

	parsed, err := ParseOutput(spec.Output, execResult.Stdout)
	if err != nil {
		return Result{}, &ParseError{
			Spec:   spec,
			Stdout: execResult.Stdout,
			Err:    err,
		}
	}

	return Result{
		Stdout:   execResult.Stdout,
		Stderr:   execResult.Stderr,
		Parsed:   parsed,
		ExitCode: execResult.ExitCode,
	}, nil
}

type osExecutor struct{}

func (osExecutor) Run(ctx context.Context, inv Invocation) (ExecResult, error) {
	cmd := osexec.CommandContext(ctx, inv.Binary, inv.Args...)
	cmd.Dir = inv.Dir
	if len(inv.Env) > 0 {
		cmd.Env = append(cmd.Env, inv.Env...)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	ret := ExecResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	if cmd.ProcessState != nil {
		ret.ExitCode = cmd.ProcessState.ExitCode()
	}
	if err != nil {
		return ret, err
	}
	return ret, nil
}

func isBinaryPathError(err error) bool {
	if err == nil {
		return false
	}
	var execErr *osexec.Error
	if stderrors.As(err, &execErr) && stderrors.Is(execErr.Err, osexec.ErrNotFound) {
		return true
	}
	var exitErr *osexec.ExitError
	if stderrors.As(err, &exitErr) {
		return false
	}
	return stderrors.Is(err, osexec.ErrNotFound) || stderrors.Is(err, io.EOF)
}
