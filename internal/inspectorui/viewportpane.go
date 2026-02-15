package inspectorui

import "github.com/charmbracelet/bubbles/viewport"

// EnsureRowVisible keeps the selected row within viewport bounds.
// It only adjusts Y offset and does not alter content.
func EnsureRowVisible(vp *viewport.Model, selected, totalRows, viewportHeight int) {
	if vp == nil {
		return
	}
	if viewportHeight < 1 {
		viewportHeight = 1
	}
	vp.Height = viewportHeight

	if totalRows <= 0 {
		vp.YOffset = 0
		return
	}
	if selected < 0 {
		selected = 0
	}
	if selected >= totalRows {
		selected = totalRows - 1
	}

	if selected < vp.YOffset {
		vp.YOffset = selected
	}
	if selected >= vp.YOffset+vp.Height {
		vp.YOffset = selected - vp.Height + 1
	}

	if vp.YOffset < 0 {
		vp.YOffset = 0
	}
	maxOffset := totalRows - vp.Height
	if maxOffset < 0 {
		maxOffset = 0
	}
	if vp.YOffset > maxOffset {
		vp.YOffset = maxOffset
	}
}
