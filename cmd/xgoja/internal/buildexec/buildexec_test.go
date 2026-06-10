package buildexec

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

func TestRunSeparatesStdoutAndStderr(t *testing.T) {
	t.Parallel()

	result, err := run(context.Background(), t.TempDir(), nil, "sh", "-c", "printf dts-output; printf diagnostic >&2")
	if err != nil {
		t.Fatalf("run command: %v", err)
	}
	if result.Stdout != "dts-output" {
		t.Fatalf("stdout = %q, want dts-output", result.Stdout)
	}
	if result.Stderr != "diagnostic" {
		t.Fatalf("stderr = %q, want diagnostic", result.Stderr)
	}
	if result.Output != "dts-outputdiagnostic" {
		t.Fatalf("combined output = %q", result.Output)
	}
}

func TestSortedEnvAndCommandString(t *testing.T) {
	env := map[string]string{"CGO_LDFLAGS": "-L/usr/local/lib -lfaiss", "A": "b"}
	got := sortedEnv(env)
	want := []string{"A=b", "CGO_LDFLAGS=-L/usr/local/lib -lfaiss"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sortedEnv = %#v, want %#v", got, want)
	}
	cmd := commandString(env, "go", []string{"build", "."})
	if !strings.HasPrefix(cmd, "A=b CGO_LDFLAGS=-L/usr/local/lib -lfaiss go build .") {
		t.Fatalf("commandString = %q", cmd)
	}
}
