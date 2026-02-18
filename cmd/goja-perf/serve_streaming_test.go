package main

import (
	"testing"
	"time"
)

func TestFmtElapsed(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{0, "0s"},
		{5 * time.Second, "5s"},
		{59 * time.Second, "59s"},
		{60 * time.Second, "1m 0s"},
		{90 * time.Second, "1m 30s"},
		{5*time.Minute + 12*time.Second, "5m 12s"},
	}
	for _, tt := range tests {
		got := fmtElapsed(tt.d)
		if got != tt.want {
			t.Errorf("fmtElapsed(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestBuildProgressView(t *testing.T) {
	lines := []string{
		"goos: linux",
		"goarch: amd64",
		"pkg: github.com/go-go-golems/go-go-goja/perf/goja",
		"BenchmarkRuntimeSpawn/GojaNew-8         	  270342	       899.5 ns/op	    1760 B/op	       8 allocs/op",
		"BenchmarkRuntimeSpawn/EngineNew_NoCallLog-8 	   12526	     18808 ns/op	   11928 B/op	     140 allocs/op",
		"some other line",
	}

	snap := runSnapshot{
		Running:   true,
		StartedAt: time.Now().Add(-10 * time.Second),
		Elapsed:   10 * time.Second,
		Lines:     lines,
		Done:      false,
	}

	pv := buildProgressView("phase1", snap)

	if pv.Phase != "phase1" {
		t.Errorf("Phase = %q, want %q", pv.Phase, "phase1")
	}
	if pv.LineCount != 6 {
		t.Errorf("LineCount = %d, want 6", pv.LineCount)
	}
	if pv.BenchCount != 2 {
		t.Errorf("BenchCount = %d, want 2", pv.BenchCount)
	}
	if pv.Done {
		t.Error("Done should be false")
	}
	if pv.HasError {
		t.Error("HasError should be false")
	}
	if len(pv.TailLines) != 6 {
		t.Errorf("TailLines len = %d, want 6 (all lines since < maxTailLines)", len(pv.TailLines))
	}
}

func TestBuildProgressView_TailTruncation(t *testing.T) {
	lines := make([]string, 50)
	for i := range lines {
		lines[i] = "line"
	}

	snap := runSnapshot{
		Running: true,
		Lines:   lines,
	}

	pv := buildProgressView("phase2", snap)
	if len(pv.TailLines) != maxTailLines {
		t.Errorf("TailLines len = %d, want %d", len(pv.TailLines), maxTailLines)
	}
}

func TestRunState_AppendAndSnapshot(t *testing.T) {
	rs := &runState{
		running:   true,
		startedAt: time.Now(),
	}

	rs.appendLine("line 1")
	rs.appendLine("line 2")

	snap := rs.snapshot()
	if !snap.Running {
		t.Error("should be running")
	}
	if len(snap.Lines) != 2 {
		t.Errorf("Lines len = %d, want 2", len(snap.Lines))
	}
	if snap.Done {
		t.Error("should not be done")
	}

	rs.finish(nil)
	snap = rs.snapshot()
	if snap.Running {
		t.Error("should not be running after finish")
	}
	if !snap.Done {
		t.Error("should be done after finish")
	}
	if snap.ExitErr != nil {
		t.Errorf("ExitErr = %v, want nil", snap.ExitErr)
	}
}

func TestIsRunning_NoRun(t *testing.T) {
	app := &perfWebApp{
		phases: map[string]phaseConfig{
			"phase1": {ID: "phase1"},
		},
	}
	if app.isRunning("phase1") {
		t.Error("should not be running before any run")
	}
}
