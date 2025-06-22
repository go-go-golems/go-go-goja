package goja_test

import (
    "bytes"
    "os/exec"
    "strings"
    "testing"
)

func TestHelloScript(t *testing.T) {
    cmd := exec.Command("go", "run", "./cmd/repl", "testdata/hello.js")
    // Ensure we run from module root where cmd/repl exists.
    cmd.Dir = "./" // project root already.

    var out bytes.Buffer
    cmd.Stdout = &out
    cmd.Stderr = &out

    if err := cmd.Run(); err != nil {
        t.Fatalf("runner failed: %v\noutput:\n%s", err, out.String())
    }

    if !strings.Contains(out.String(), "OK") {
        t.Fatalf("expected output to contain 'OK', got:\n%s", out.String())
    }
} 