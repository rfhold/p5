package ui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// WorkspaceItem represents a workspace in the selector
type WorkspaceItem struct {
	Path         string
	RelativePath string // Path relative to current working directory
	Name         string
	Current      bool
}

// Label implements SelectorItem
func (w WorkspaceItem) Label() string {
	return w.Name
}

// IsCurrent implements SelectorItem
func (w WorkspaceItem) IsCurrent() bool {
	return w.Current
}

// WorkspaceSelector is a modal dialog for selecting a workspace
type WorkspaceSelector struct {
	*SelectorDialog[WorkspaceItem]
}

// NewWorkspaceSelector creates a new workspace selector
func NewWorkspaceSelector() *WorkspaceSelector {
	dialog := NewSelectorDialog[WorkspaceItem]("Select Workspace")
	dialog.SetLoadingText("Searching for Pulumi projects...")
	dialog.SetEmptyText("No Pulumi projects found")

	// Custom extra info renderer to show path after name
	dialog.SetExtraInfoRenderer(func(item WorkspaceItem) string {
		if item.Current {
			return "" // Don't show path for current item (already shows "(current)")
		}
		displayPath := item.RelativePath
		if displayPath == "" {
			displayPath = item.Path
		}
		return DimStyle.Render(" " + displayPath)
	})

	return &WorkspaceSelector{
		SelectorDialog: dialog,
	}
}

// SetWorkspaces sets the list of available workspaces
func (s *WorkspaceSelector) SetWorkspaces(workspaces []WorkspaceItem) {
	s.SetItems(workspaces)
}

// SelectedWorkspace returns the currently selected workspace
func (s *WorkspaceSelector) SelectedWorkspace() *WorkspaceItem {
	return s.SelectedItem()
}

// HasWorkspaces returns whether any workspaces are available
func (s *WorkspaceSelector) HasWorkspaces() bool {
	return s.HasItems()
}

// Update handles key events and returns true if a workspace was selected
func (s *WorkspaceSelector) Update(msg tea.KeyMsg) (selected bool, cmd tea.Cmd) {
	return s.SelectorDialog.Update(msg)
}

// View renders the workspace selector dialog
func (s *WorkspaceSelector) View() string {
	return s.SelectorDialog.View()
}
