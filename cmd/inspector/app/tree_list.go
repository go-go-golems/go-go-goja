package app

import (
	"github.com/charmbracelet/bubbles/list"
	inspectortree "github.com/go-go-golems/go-go-goja/pkg/inspector/tree"
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
	row := inspectortree.BuildRow(node, usageHighlights, res)
	return treeListItem{
		id:          row.NodeID,
		title:       row.Title,
		description: row.Description,
	}
}
