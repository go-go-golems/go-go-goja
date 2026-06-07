package buildexec

import (
	"reflect"
	"strings"
	"testing"
)

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
