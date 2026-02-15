package inspectorui

// EnsureSelectionVisible adjusts a list scroll offset so the selected row stays visible.
func EnsureSelectionVisible(scroll *int, selected, totalRows, viewportHeight int) {
	if scroll == nil {
		return
	}
	if viewportHeight < 1 {
		viewportHeight = 1
	}
	if totalRows <= 0 {
		*scroll = 0
		return
	}
	if selected < 0 {
		selected = 0
	}
	if selected >= totalRows {
		selected = totalRows - 1
	}
	if *scroll < 0 {
		*scroll = 0
	}

	if selected < *scroll {
		*scroll = selected
	}
	if selected >= *scroll+viewportHeight {
		*scroll = selected - viewportHeight + 1
	}

	maxScroll := totalRows - viewportHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if *scroll > maxScroll {
		*scroll = maxScroll
	}
}

// VisibleRange returns the half-open [start,end) window for rendering rows.
func VisibleRange(scroll, totalRows, viewportHeight int) (int, int) {
	if viewportHeight < 1 {
		viewportHeight = 1
	}
	if totalRows <= 0 {
		return 0, 0
	}
	if scroll < 0 {
		scroll = 0
	}
	maxScroll := totalRows - viewportHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if scroll > maxScroll {
		scroll = maxScroll
	}

	start := scroll
	end := start + viewportHeight
	if end > totalRows {
		end = totalRows
	}
	return start, end
}
