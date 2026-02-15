package inspectorui

import "testing"

func TestEnsureSelectionVisible(t *testing.T) {
	t.Run("resets scroll for empty lists", func(t *testing.T) {
		scroll := 7
		EnsureSelectionVisible(&scroll, 3, 0, 5)
		if scroll != 0 {
			t.Fatalf("expected scroll=0, got %d", scroll)
		}
	})

	t.Run("moves scroll up when selection is above window", func(t *testing.T) {
		scroll := 5
		EnsureSelectionVisible(&scroll, 2, 20, 4)
		if scroll != 2 {
			t.Fatalf("expected scroll=2, got %d", scroll)
		}
	})

	t.Run("moves scroll down when selection is below window", func(t *testing.T) {
		scroll := 2
		EnsureSelectionVisible(&scroll, 9, 20, 4)
		if scroll != 6 {
			t.Fatalf("expected scroll=6, got %d", scroll)
		}
	})

	t.Run("clamps scroll to max", func(t *testing.T) {
		scroll := 100
		EnsureSelectionVisible(&scroll, 99, 10, 3)
		if scroll != 7 {
			t.Fatalf("expected scroll=7, got %d", scroll)
		}
	})
}

func TestVisibleRange(t *testing.T) {
	tests := []struct {
		name           string
		scroll         int
		totalRows      int
		viewportHeight int
		wantStart      int
		wantEnd        int
	}{
		{
			name:           "empty",
			scroll:         0,
			totalRows:      0,
			viewportHeight: 5,
			wantStart:      0,
			wantEnd:        0,
		},
		{
			name:           "simple window",
			scroll:         3,
			totalRows:      20,
			viewportHeight: 5,
			wantStart:      3,
			wantEnd:        8,
		},
		{
			name:           "clamps negative scroll",
			scroll:         -5,
			totalRows:      7,
			viewportHeight: 3,
			wantStart:      0,
			wantEnd:        3,
		},
		{
			name:           "clamps past end",
			scroll:         50,
			totalRows:      7,
			viewportHeight: 3,
			wantStart:      4,
			wantEnd:        7,
		},
		{
			name:           "viewport taller than rows",
			scroll:         2,
			totalRows:      3,
			viewportHeight: 10,
			wantStart:      0,
			wantEnd:        3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := VisibleRange(tt.scroll, tt.totalRows, tt.viewportHeight)
			if start != tt.wantStart || end != tt.wantEnd {
				t.Fatalf("got (%d,%d), want (%d,%d)", start, end, tt.wantStart, tt.wantEnd)
			}
		})
	}
}
