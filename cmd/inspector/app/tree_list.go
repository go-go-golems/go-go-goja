package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
)

type treeListItem struct {
	id          NodeID
	title       string
	description string
}

func (i treeListItem) Title() string       { return i.title }
func (i treeListItem) Description() string { return i.description }
func (i treeListItem) FilterValue() string { return i.title + " " + i.description }

func newTreeListModel() list.Model {
	delegate := list.NewDefaultDelegate()
	delegate.SetHeight(1)
	delegate.ShowDescription = true

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.SetFilteringEnabled(false)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.DisableQuitKeybindings()
	l.KeyMap.Filter.SetEnabled(false)
	l.KeyMap.ClearFilter.SetEnabled(false)
	l.KeyMap.AcceptWhileFiltering.SetEnabled(false)
	l.Styles.NoItems = l.Styles.NoItems.UnsetPadding()
	return l
}

func buildTreeListItem(node *NodeRecord, usageHighlights []NodeID, res *Resolution) treeListItem {
	if node == nil {
		return treeListItem{}
	}

	indent := strings.Repeat("  ", node.Depth)
	expandMarker := " "
	if node.HasChildren() {
		if node.Expanded {
			expandMarker = "▼"
		} else {
			expandMarker = "▶"
		}
	}

	scopeHint := ""
	if res != nil && node.Kind == "Identifier" {
		if res.IsDeclaration(node.ID) {
			if b := res.BindingForNode(node.ID); b != nil {
				scopeHint = fmt.Sprintf(" [%s decl]", b.Kind)
			}
		} else if res.IsReference(node.ID) {
			scopeHint = " [ref]"
		} else if res.IsUnresolved(node.ID) {
			scopeHint = " [global]"
		}
	}

	isUsage := false
	for _, id := range usageHighlights {
		if id == node.ID {
			isUsage = true
			break
		}
	}
	usageHint := ""
	if isUsage {
		usageHint = " ★usage"
	}

	return treeListItem{
		id:          node.ID,
		title:       indent + expandMarker + " " + node.DisplayLabel(),
		description: fmt.Sprintf("[%d..%d]%s%s", node.Start, node.End, scopeHint, usageHint),
	}
}
