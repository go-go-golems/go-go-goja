package main

import (
	"testing"
)

func TestFmtNs(t *testing.T) {
	tests := []struct {
		input float64
		want  string
	}{
		{0, "0 ns"},
		{0.4, "0.4 ns"},
		{155, "155 ns"},
		{981.2, "981.2 ns"},
		{1830, "1.8 µs"},
		{20450, "20.4 µs"},
		{232808, "232.8 µs"},
		{1037406, "1.0 ms"},
		{38226.67, "38.2 µs"},
		{1e9, "1.00 s"},
		{2.5e9, "2.50 s"},
		{-5, "0 ns"},
	}
	for _, tt := range tests {
		got := fmtNs(tt.input)
		if got != tt.want {
			t.Errorf("fmtNs(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFmtBytes(t *testing.T) {
	tests := []struct {
		input float64
		want  string
	}{
		{0, "0 B"},
		{32, "32 B"},
		{160, "160 B"},
		{1760, "1.7 KB"},
		{3328, "3.2 KB"},
		{11928, "12 KB"},
		{14320, "14 KB"},
		{33728, "33 KB"},
		{495050, "483 KB"},
		{1048576, "1.0 MB"},
		{-1, "0 B"},
	}
	for _, tt := range tests {
		got := fmtBytes(tt.input)
		if got != tt.want {
			t.Errorf("fmtBytes(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFmtCount(t *testing.T) {
	tests := []struct {
		input float64
		want  string
	}{
		{0, "0"},
		{8, "8"},
		{41, "41"},
		{140, "140"},
		{1000, "1,000"},
		{12345, "12,345"},
		{1234567, "1,234,567"},
	}
	for _, tt := range tests {
		got := fmtCount(tt.input)
		if got != tt.want {
			t.Errorf("fmtCount(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestShortBench(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"BenchmarkRuntimeSpawn/GojaNew-8", "GojaNew"},
		{"BenchmarkRuntimeSpawn/EngineNew_NoCallLog-8", "EngineNew_NoCallLog"},
		{"BenchmarkRuntimeSpawn/EngineNew_WithCallLog-8", "EngineNew_WithCallLog"},
		{"BenchmarkRuntimeReuse/RunProgram_ReusedRuntime-8", "RunProgram_ReusedRuntime"},
		{"BenchmarkJSCallingGo/GoDirect-8", "GoDirect"},
		{"BenchmarkValueConversion/ToValue_Primitive_Int-8", "ToValue_Primitive_Int"},
		{"BenchmarkSimple", "Simple"},
		{"BenchmarkNoSlash-16", "NoSlash"},
	}
	for _, tt := range tests {
		got := shortBench(tt.input)
		if got != tt.want {
			t.Errorf("shortBench(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFmtTPS(t *testing.T) {
	tests := []struct {
		nsPerOp float64
		want    string
	}{
		{0, "∞ ops/s"},
		{-1, "∞ ops/s"},
		{0.4, "2.5G ops/s"},    // GoDirect: ~2.5 billion ops/s
		{155, "6.5M ops/s"},    // RunProgram_ReusedRuntime
		{981, "1.0M ops/s"},    // GojaNew
		{6067, "164.8K ops/s"}, // RunString_ReusedRuntime
		{20450, "48.9K ops/s"}, // EngineNew_NoCallLog
		{232808, "4.3K ops/s"}, // EngineNew_WithCallLog
		{1037406, "964 ops/s"}, // Compile_medium
		{1e9, "1 ops/s"},       // 1 second per op
		{2e9, "0.5 ops/s"},     // 2 seconds per op
	}
	for _, tt := range tests {
		got := fmtTPS(tt.nsPerOp)
		if got != tt.want {
			t.Errorf("fmtTPS(%v) = %q, want %q", tt.nsPerOp, got, tt.want)
		}
	}
}

func TestFmtDurationMS(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0 ms"},
		{500, "500 ms"},
		{999, "999 ms"},
		{1000, "1.0 s"},
		{8172, "8.2 s"},
		{30242, "30.2 s"},
	}
	for _, tt := range tests {
		got := fmtDurationMS(tt.input)
		if got != tt.want {
			t.Errorf("fmtDurationMS(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestPrepareTasks(t *testing.T) {
	results := []phase1TaskResult{
		{
			ID:              "test-task",
			TaskTitle:       "Test Task",
			TaskDescription: "Test description",
			DurationMS:      5000,
			Success:         true,
			Summaries: []benchmarkSummary{
				{
					Benchmark: "BenchmarkTest/Fast-8",
					Runs:      3,
					Metrics: []benchmarkMetricStat{
						{Metric: "ns/op", Avg: 100, Min: 90, Max: 110},
						{Metric: "B/op", Avg: 32, Min: 32, Max: 32},
						{Metric: "allocs/op", Avg: 1, Min: 1, Max: 1},
					},
				},
				{
					Benchmark: "BenchmarkTest/Slow-8",
					Runs:      3,
					Metrics: []benchmarkMetricStat{
						{Metric: "ns/op", Avg: 50000, Min: 48000, Max: 52000},
						{Metric: "B/op", Avg: 4096, Min: 4096, Max: 4096},
						{Metric: "allocs/op", Avg: 40, Min: 40, Max: 40},
					},
				},
			},
		},
	}

	tasks := prepareTasks(results)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}

	task := tasks[0]
	if task.Title != "Test Task" {
		t.Errorf("task title = %q, want %q", task.Title, "Test Task")
	}
	if len(task.Cards) != 2 {
		t.Fatalf("expected 2 cards, got %d", len(task.Cards))
	}

	fast := task.Cards[0]
	if fast.ShortName != "Fast" {
		t.Errorf("card[0].ShortName = %q, want %q", fast.ShortName, "Fast")
	}
	if fast.NsFormatted != "100 ns" {
		t.Errorf("card[0].NsFormatted = %q, want %q", fast.NsFormatted, "100 ns")
	}
	if fast.RelativeText != "⚡ fastest" {
		t.Errorf("card[0].RelativeText = %q, want %q", fast.RelativeText, "⚡ fastest")
	}

	slow := task.Cards[1]
	if slow.ShortName != "Slow" {
		t.Errorf("card[1].ShortName = %q, want %q", slow.ShortName, "Slow")
	}
	if slow.IsSlow != true {
		t.Error("card[1].IsSlow should be true for 500× ratio")
	}
	if slow.BarPct < fast.BarPct {
		t.Errorf("slow bar (%d) should be >= fast bar (%d)", slow.BarPct, fast.BarPct)
	}
}
