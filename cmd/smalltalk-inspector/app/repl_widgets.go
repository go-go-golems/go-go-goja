package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/go-go-golems/bobatea/pkg/overlay"
	"github.com/go-go-golems/bobatea/pkg/tui/widgets/contextbar"
	"github.com/go-go-golems/bobatea/pkg/tui/widgets/contextpanel"
	"github.com/go-go-golems/bobatea/pkg/tui/widgets/suggest"
	jsadapter "github.com/go-go-golems/go-go-goja/pkg/repl/adapters/bobatea"
	jsrepl "github.com/go-go-golems/go-go-goja/pkg/repl/evaluators/javascript"
)

type replSuggestProvider struct {
	evaluator interface {
		CompleteInput(context.Context, suggest.Request) (suggest.Result, error)
	}
}

func (p replSuggestProvider) CompleteInput(ctx context.Context, req suggest.Request) (suggest.Result, error) {
	if p.evaluator == nil {
		return suggest.Result{}, nil
	}
	return p.evaluator.CompleteInput(ctx, req)
}

type replContextBarProvider struct {
	evaluator interface {
		GetHelpBar(context.Context, contextbar.Request) (contextbar.Payload, error)
	}
}

func (p replContextBarProvider) GetContextBar(ctx context.Context, req contextbar.Request) (contextbar.Payload, error) {
	if p.evaluator == nil {
		return contextbar.Payload{}, nil
	}
	return p.evaluator.GetHelpBar(ctx, req)
}

type replContextPanelProvider struct {
	evaluator interface {
		GetHelpDrawer(context.Context, contextpanel.Request) (contextpanel.Document, error)
	}
}

func (p replContextPanelProvider) GetContextPanel(ctx context.Context, req contextpanel.Request) (contextpanel.Document, error) {
	if p.evaluator == nil {
		return contextpanel.Document{}, nil
	}
	return p.evaluator.GetHelpDrawer(ctx, req)
}

type replInputBufferAdapter struct {
	input *textinput.Model
}

func (a replInputBufferAdapter) Value() string {
	if a.input == nil {
		return ""
	}
	return a.input.Value()
}

func (a replInputBufferAdapter) CursorByte() int {
	if a.input == nil {
		return 0
	}
	return a.input.Position()
}

func (a replInputBufferAdapter) SetValue(value string) {
	if a.input == nil {
		return
	}
	a.input.SetValue(value)
}

func (a replInputBufferAdapter) SetCursorByte(cursor int) {
	if a.input == nil {
		return
	}
	a.input.SetCursor(cursor)
}

func (m *Model) setupReplWidgetsForRuntime() {
	m.replAssist = nil
	m.replSuggestWidget = nil
	m.replContextBarWidget = nil
	m.replContextPanelWidget = nil

	if m.rtSession == nil || m.rtSession.VM == nil {
		return
	}

	cfg := jsrepl.DefaultConfig()
	cfg.Runtime = m.rtSession.VM
	cfg.EnableConsoleLog = false

	evaluator, err := jsadapter.NewJavaScriptEvaluator(cfg)
	if err != nil {
		m.statusMsg = fmt.Sprintf("%s | ⚠ REPL assist disabled: %v", strings.TrimSpace(m.statusMsg), err)
		return
	}

	m.replAssist = evaluator
	m.replSuggestWidget = suggest.New(replSuggestProvider{evaluator: evaluator}, suggest.Config{
		Debounce:       120 * time.Millisecond,
		RequestTimeout: 400 * time.Millisecond,
		MaxVisible:     8,
		PageSize:       8,
		MaxWidth:       56,
		MaxHeight:      12,
		MinWidth:       24,
		Margin:         1,
		OffsetX:        0,
		OffsetY:        0,
		NoBorder:       false,
		Placement:      suggest.PlacementAuto,
		HorizontalGrow: suggest.HorizontalGrowRight,
	})
	m.replContextBarWidget = contextbar.New(replContextBarProvider{evaluator: evaluator}, 120*time.Millisecond, 300*time.Millisecond)
	m.replContextPanelWidget = contextpanel.New(replContextPanelProvider{evaluator: evaluator}, contextpanel.Config{
		Debounce:           140 * time.Millisecond,
		RequestTimeout:     500 * time.Millisecond,
		PrefetchWhenHidden: false,
		Dock:               contextpanel.DockAboveRepl,
		WidthPercent:       52,
		HeightPercent:      46,
		Margin:             1,
	})
}

func (m *Model) recordReplAssistDeclarations(expr string) {
	if m.replAssist == nil || m.replAssist.Core() == nil {
		return
	}
	m.replAssist.Core().RecordDeclarations(expr)
}

func (m *Model) scheduleReplWidgetDebounce(prevValue string, prevCursor int) tea.Cmd {
	if m.replAssist == nil {
		return nil
	}

	value := m.replInput.Value()
	cursor := m.replInput.Position()

	var cmds []tea.Cmd
	if m.replSuggestWidget != nil {
		if cmd := m.replSuggestWidget.OnBufferChanged(prevValue, prevCursor, value, cursor); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	if m.replContextBarWidget != nil {
		if cmd := m.replContextBarWidget.OnBufferChanged(prevValue, prevCursor, value, cursor); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	if m.replContextPanelWidget != nil {
		if cmd := m.replContextPanelWidget.OnBufferChanged(prevValue, prevCursor, value, cursor); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

func (m *Model) handleReplSuggestDebounce(msg suggest.DebounceMsg) tea.Cmd {
	if m.replSuggestWidget == nil {
		return nil
	}
	return m.replSuggestWidget.HandleDebounce(context.Background(), msg, m.replInput.Value(), m.replInput.Position())
}

func (m *Model) handleReplSuggestResult(msg suggest.ResultMsg) tea.Cmd {
	if m.replSuggestWidget == nil {
		return nil
	}
	m.replSuggestWidget.HandleResult(msg)
	return nil
}

func (m *Model) handleReplContextBarDebounce(msg contextbar.DebounceMsg) tea.Cmd {
	if m.replContextBarWidget == nil {
		return nil
	}
	return m.replContextBarWidget.HandleDebounce(context.Background(), msg, m.replInput.Value(), m.replInput.Position())
}

func (m *Model) handleReplContextBarResult(msg contextbar.ResultMsg) tea.Cmd {
	if m.replContextBarWidget == nil {
		return nil
	}
	m.replContextBarWidget.HandleResult(msg)
	return nil
}

func (m *Model) handleReplContextPanelDebounce(msg contextpanel.DebounceMsg) tea.Cmd {
	if m.replContextPanelWidget == nil {
		return nil
	}
	return m.replContextPanelWidget.HandleDebounce(context.Background(), msg, m.replInput.Value(), m.replInput.Position())
}

func (m *Model) handleReplContextPanelResult(msg contextpanel.ResultMsg) tea.Cmd {
	if m.replContextPanelWidget == nil {
		return nil
	}
	m.replContextPanelWidget.HandleResult(msg)
	return nil
}

func (m *Model) triggerReplCompletionShortcut(msg tea.KeyMsg) tea.Cmd {
	if m.replSuggestWidget == nil {
		return nil
	}
	if !key.Matches(msg, m.keyMap.CompletionTrigger) {
		return nil
	}
	return m.replSuggestWidget.TriggerShortcut(context.Background(), m.replInput.Value(), m.replInput.Position(), msg.String())
}

func (m *Model) handleReplSuggestionNavigation(msg tea.KeyMsg) bool {
	if m.replSuggestWidget == nil || !m.replSuggestWidget.Visible() {
		return false
	}
	buf := replInputBufferAdapter{input: &m.replInput}

	switch {
	case key.Matches(msg, m.keyMap.CompletionCancel):
		return m.replSuggestWidget.HandleNavigation(suggest.ActionCancel, buf)
	case key.Matches(msg, m.keyMap.CompletionPrev):
		return m.replSuggestWidget.HandleNavigation(suggest.ActionPrev, buf)
	case key.Matches(msg, m.keyMap.CompletionNext):
		return m.replSuggestWidget.HandleNavigation(suggest.ActionNext, buf)
	case key.Matches(msg, m.keyMap.CompletionPageUp):
		return m.replSuggestWidget.HandleNavigation(suggest.ActionPageUp, buf)
	case key.Matches(msg, m.keyMap.CompletionPageDown):
		return m.replSuggestWidget.HandleNavigation(suggest.ActionPageDown, buf)
	case key.Matches(msg, m.keyMap.CompletionAccept):
		return m.replSuggestWidget.HandleNavigation(suggest.ActionAccept, buf)
	default:
		return false
	}
}

func (m *Model) handleReplHelpDrawerKeys(msg tea.KeyMsg) (bool, tea.Cmd) {
	if m.replContextPanelWidget == nil {
		return false, nil
	}

	switch {
	case key.Matches(msg, m.keyMap.HelpDrawerToggle):
		return true, m.replContextPanelWidget.Toggle(context.Background(), m.replInput.Value(), m.replInput.Position())
	case m.replContextPanelWidget.Visible() && key.Matches(msg, m.keyMap.HelpDrawerRefresh):
		return true, m.replContextPanelWidget.RequestNow(context.Background(), m.replInput.Value(), m.replInput.Position(), contextpanel.TriggerManualRefresh)
	case m.replContextPanelWidget.Visible() && key.Matches(msg, m.keyMap.HelpDrawerPin):
		m.replContextPanelWidget.TogglePin()
		return true, nil
	default:
		return false, nil
	}
}

func (m Model) renderReplContextBar() string {
	if m.replContextBarWidget == nil {
		return ""
	}
	return m.replContextBarWidget.Render(func(severity string, text string) string {
		switch severity {
		case "error":
			return styleReplError.Render(text)
		case "warning":
			return styleSeparator.Render(text)
		default:
			return styleEmptyHint.Render(text)
		}
	})
}

func (m Model) applyReplWidgetOverlays(base string) string {
	if m.width <= 0 || m.height <= 0 {
		return base
	}

	headerHeight := 1
	timelineHeight := m.contentHeight() + 1
	out := base

	if m.replContextPanelWidget != nil && m.replContextPanelWidget.Visible() {
		layout, ok := m.replContextPanelWidget.ComputeOverlayLayout(m.width, m.height, headerHeight, timelineHeight)
		if ok {
			panel := m.replContextPanelWidget.RenderPanel(layout, contextpanel.RenderOptions{
				ToggleBinding:  "alt+h",
				RefreshBinding: "ctrl+r",
				PinBinding:     "ctrl+g",
				FooterRenderer: func(s string) string { return styleHelp.Render(s) },
			})
			if panel != "" {
				out = overlay.PlaceOverlay(layout.PanelX, layout.PanelY, panel, out, false)
			}
		}
	}

	if m.focus == FocusRepl && m.replSuggestWidget != nil && m.replSuggestWidget.Visible() {
		popupStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorMuted)

		layout, ok := m.replSuggestWidget.ComputeOverlayLayout(
			m.width,
			m.height,
			headerHeight,
			timelineHeight,
			m.replInput.Prompt,
			m.replInput.Value(),
			m.replInput.Position(),
			popupStyle,
		)
		if ok {
			popup := m.replSuggestWidget.RenderPopup(suggest.Styles{
				Item:     lipgloss.NewStyle().Foreground(colorBright),
				Selected: lipgloss.NewStyle().Foreground(colorBright).Background(colorPaneFocusBg).Bold(true),
				Popup:    popupStyle,
			}, layout)
			if popup != "" {
				out = overlay.PlaceOverlay(layout.PopupX, layout.PopupY, popup, out, false)
			}
		}
	}

	return out
}
