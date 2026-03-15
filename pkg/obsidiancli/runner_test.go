package obsidiancli

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type fakeExecutor struct {
	mu          sync.Mutex
	invocations []Invocation
	result      ExecResult
	err         error
	active      int
	maxActive   int
	blockCh     chan struct{}
}

func (f *fakeExecutor) Run(_ context.Context, inv Invocation) (ExecResult, error) {
	f.mu.Lock()
	f.invocations = append(f.invocations, inv)
	f.active++
	if f.active > f.maxActive {
		f.maxActive = f.active
	}
	f.mu.Unlock()

	if f.blockCh != nil {
		<-f.blockCh
	}

	f.mu.Lock()
	f.active--
	f.mu.Unlock()
	return f.result, f.err
}

func TestRunnerParsesCommandOutput(t *testing.T) {
	exec := &fakeExecutor{
		result: ExecResult{Stdout: "a.md\nb.md\n", ExitCode: 0},
	}
	runner := NewRunner(Config{BinaryPath: "obsidian-cli", Vault: "Main", WorkingDir: "/vault"}, exec)

	result, err := runner.Run(context.Background(), SpecFilesList, CallOptions{})
	require.NoError(t, err)
	require.Equal(t, []string{"a.md", "b.md"}, result.Parsed)
	require.Len(t, exec.invocations, 1)
	require.Equal(t, "obsidian-cli", exec.invocations[0].Binary)
	require.Equal(t, "/vault", exec.invocations[0].Dir)
	require.Equal(t, []string{"vault=Main", "files:list"}, exec.invocations[0].Args)
}

func TestRunnerWrapsParseErrors(t *testing.T) {
	exec := &fakeExecutor{
		result: ExecResult{Stdout: "broken", ExitCode: 0},
	}
	runner := NewRunner(DefaultConfig(), exec)

	_, err := runner.Run(context.Background(), CommandSpec{Name: "vault:info", Output: OutputJSON}, CallOptions{})
	require.Error(t, err)
	var parseErr *ParseError
	require.ErrorAs(t, err, &parseErr)
}

func TestRunnerSerializesConcurrentInvocations(t *testing.T) {
	exec := &fakeExecutor{
		result:  ExecResult{Stdout: "ok", ExitCode: 0},
		blockCh: make(chan struct{}),
	}
	runner := NewRunner(DefaultConfig(), exec)

	errCh := make(chan error, 2)
	go func() {
		_, err := runner.Run(context.Background(), SpecVersion, CallOptions{})
		errCh <- err
	}()

	started := make(chan struct{})
	go func() {
		close(started)
		_, err := runner.Run(context.Background(), SpecVersion, CallOptions{})
		errCh <- err
	}()
	<-started

	time.Sleep(50 * time.Millisecond)
	close(exec.blockCh)

	require.NoError(t, <-errCh)
	require.NoError(t, <-errCh)
	require.Equal(t, 1, exec.maxActive, fmt.Sprintf("expected serialized execution, got maxActive=%d", exec.maxActive))
}
