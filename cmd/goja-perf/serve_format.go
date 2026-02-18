package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// fmtNs formats nanoseconds into human-readable duration strings.
func fmtNs(ns float64) string {
	if ns < 0 {
		return "0 ns"
	}
	switch {
	case ns < 1000:
		if ns == math.Trunc(ns) {
			return fmt.Sprintf("%.0f ns", ns)
		}
		return fmt.Sprintf("%.1f ns", ns)
	case ns < 1e6:
		return fmt.Sprintf("%.1f µs", ns/1e3)
	case ns < 1e9:
		return fmt.Sprintf("%.1f ms", ns/1e6)
	default:
		return fmt.Sprintf("%.2f s", ns/1e9)
	}
}

// fmtBytes formats byte counts into human-readable strings.
func fmtBytes(b float64) string {
	if b < 0 {
		return "0 B"
	}
	switch {
	case b < 1024:
		return fmt.Sprintf("%.0f B", b)
	case b < 1024*1024:
		v := b / 1024
		if v >= 10 {
			return fmt.Sprintf("%.0f KB", v)
		}
		return fmt.Sprintf("%.1f KB", v)
	default:
		return fmt.Sprintf("%.1f MB", b/(1024*1024))
	}
}

// fmtCount formats an integer with comma separators.
func fmtCount(n float64) string {
	i := int64(n)
	if i == 0 {
		return "0"
	}
	s := strconv.FormatInt(i, 10)
	if len(s) <= 3 {
		return s
	}
	// Insert commas from the right
	var b strings.Builder
	start := len(s) % 3
	if start == 0 {
		start = 3
	}
	b.WriteString(s[:start])
	for j := start; j < len(s); j += 3 {
		b.WriteByte(',')
		b.WriteString(s[j : j+3])
	}
	return b.String()
}

// shortBench extracts the short sub-case name from a full benchmark name.
// "BenchmarkRuntimeSpawn/GojaNew-8" → "GojaNew"
// "BenchmarkRuntimeSpawn/EngineNew_NoCallLog-8" → "EngineNew_NoCallLog"
func shortBench(name string) string {
	// Take the part after the last "/"
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	} else {
		// No slash — strip "Benchmark" prefix
		name = strings.TrimPrefix(name, "Benchmark")
	}
	// Strip the "-N" GOMAXPROCS suffix
	if i := strings.LastIndex(name, "-"); i >= 0 {
		suffix := name[i+1:]
		if _, err := strconv.Atoi(suffix); err == nil {
			name = name[:i]
		}
	}
	return name
}

// fmtTPS formats nanoseconds-per-op into human-readable throughput (ops/sec).
func fmtTPS(nsPerOp float64) string {
	if nsPerOp <= 0 {
		return "∞ ops/s"
	}
	tps := 1e9 / nsPerOp
	switch {
	case tps >= 1e9:
		return fmt.Sprintf("%.1fG ops/s", tps/1e9)
	case tps >= 1e6:
		return fmt.Sprintf("%.1fM ops/s", tps/1e6)
	case tps >= 1e3:
		return fmt.Sprintf("%.1fK ops/s", tps/1e3)
	case tps >= 1:
		return fmt.Sprintf("%.0f ops/s", tps)
	default:
		return fmt.Sprintf("%.1f ops/s", tps)
	}
}

// fmtDurationMS formats milliseconds into a human-readable duration.
func fmtDurationMS(ms int64) string {
	if ms < 1000 {
		return fmt.Sprintf("%d ms", ms)
	}
	return fmt.Sprintf("%.1f s", float64(ms)/1000)
}

// benchmarkCardData holds pre-processed data for a single benchmark card in the UI.
type benchmarkCardData struct {
	ShortName       string
	Description     string
	NsFormatted     string
	TpsFormatted    string
	NsAvg           float64
	BytesFormatted  string
	AllocsFormatted string
	BarPct          int
	RangeText       string
	RelativeText    string
	IsSlow          bool
}

// taskViewData holds pre-processed data for a single task section.
type taskViewData struct {
	ID          string
	Title       string
	Description string
	DurationMS  int64
	Success     bool
	Cards       []benchmarkCardData
}

// prepareTasks converts raw phase1TaskResult data into taskViewData for template rendering.
func prepareTasks(results []phase1TaskResult) []taskViewData {
	tasks := make([]taskViewData, 0, len(results))
	for _, r := range results {
		tv := taskViewData{
			ID:          r.ID,
			Title:       r.TaskTitle,
			Description: r.TaskDescription,
			DurationMS:  r.DurationMS,
			Success:     r.Success,
		}

		// Build cards from summaries
		cards := make([]benchmarkCardData, 0, len(r.Summaries))
		for _, s := range r.Summaries {
			card := benchmarkCardData{
				ShortName:   shortBench(s.Benchmark),
				Description: s.Description,
			}

			// Extract metrics
			var nsAvg, nsMin, nsMax float64
			var bytesAvg, allocsAvg float64
			for _, m := range s.Metrics {
				switch m.Metric {
				case "ns/op":
					nsAvg = m.Avg
					nsMin = m.Min
					nsMax = m.Max
				case "B/op":
					bytesAvg = m.Avg
				case "allocs/op":
					allocsAvg = m.Avg
				}
			}

			card.NsAvg = nsAvg
			card.NsFormatted = fmtNs(nsAvg)
			card.TpsFormatted = fmtTPS(nsAvg)
			card.BytesFormatted = fmtBytes(bytesAvg)
			card.AllocsFormatted = fmtCount(allocsAvg)

			// Range text (only if min != max)
			if nsMin != nsMax {
				card.RangeText = fmt.Sprintf("range: %s – %s", fmtNs(nsMin), fmtNs(nsMax))
			}

			cards = append(cards, card)
		}

		// Compute bar percentages and relative labels
		if len(cards) > 0 {
			// Find min and max ns/op for scaling
			maxNs := 0.0
			minNs := math.MaxFloat64
			minIdx := 0
			for i, c := range cards {
				if c.NsAvg > maxNs {
					maxNs = c.NsAvg
				}
				if c.NsAvg < minNs {
					minNs = c.NsAvg
					minIdx = i
				}
			}

			for i := range cards {
				if maxNs > 0 {
					// Use log scale to handle extreme ranges (e.g., 0.4 ns vs 233 µs)
					if minNs > 0 && maxNs/minNs > 100 {
						// Log scale
						logMin := math.Log10(minNs)
						logMax := math.Log10(maxNs)
						logVal := math.Log10(math.Max(cards[i].NsAvg, minNs))
						if logMax > logMin {
							cards[i].BarPct = int(((logVal - logMin) / (logMax - logMin)) * 100)
						} else {
							cards[i].BarPct = 100
						}
					} else {
						cards[i].BarPct = int((cards[i].NsAvg / maxNs) * 100)
					}
					// Minimum 2% for visibility
					if cards[i].BarPct < 2 {
						cards[i].BarPct = 2
					}
				}

				// Relative text
				if len(cards) > 1 {
					if i == minIdx {
						cards[i].RelativeText = "⚡ fastest"
						cards[i].IsSlow = false
					} else if minNs > 0 {
						ratio := cards[i].NsAvg / minNs
						if ratio >= 1000 {
							cards[i].RelativeText = fmt.Sprintf("%.0f× slower", ratio)
						} else if ratio >= 10 {
							cards[i].RelativeText = fmt.Sprintf("%.0f× slower", ratio)
						} else if ratio >= 1.1 {
							cards[i].RelativeText = fmt.Sprintf("%.1f× slower", ratio)
						}
						cards[i].IsSlow = ratio >= 5
					}
				}
			}
		}

		tv.Cards = cards
		tasks = append(tasks, tv)
	}
	return tasks
}
