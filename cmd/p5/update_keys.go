package main

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/rfhold/p5/internal/pulumi"
	"github.com/rfhold/p5/internal/ui"
)

// handleKeyPress handles all keyboard events
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle error modal first if visible
	if m.errorModal.Visible() {
		dismissed, cmd := m.errorModal.Update(msg)
		if dismissed {
			m.errorModal.Hide()
		}
		return m, cmd
	}

	// Handle confirm modal first if visible
	if m.confirmModal.Visible() {
		confirmed, cancelled, cmd := m.confirmModal.Update(msg)
		if confirmed {
			// Check if this is a pending operation confirmation
			if m.pendingOperation != nil {
				op := *m.pendingOperation
				m.pendingOperation = nil
				m.confirmModal.Hide()
				return m, m.startExecution(op)
			}
			// Otherwise it's a state delete confirmation
			return m, m.executeStateDelete()
		}
		if cancelled {
			m.pendingOperation = nil
			m.confirmModal.Hide()
		}
		return m, cmd
	}

	// Handle import modal if visible
	if m.importModal.Visible() {
		confirmed, cmd := m.importModal.Update(msg)
		if confirmed {
			// User confirmed import, execute it
			return m, m.executeImport()
		}
		return m, cmd
	}

	// Handle workspace selector first if visible
	if m.workspaceSelector.Visible() {
		selected, cmd := m.workspaceSelector.Update(msg)
		if selected {
			// Workspace was selected, update and reload
			selectedWs := m.workspaceSelector.SelectedWorkspace()
			if selectedWs != nil {
				return m, selectWorkspace(selectedWs.Path)
			}
		}
		return m, cmd
	}

	// Handle stack selector if visible
	if m.stackSelector.Visible() {
		selected, cmd := m.stackSelector.Update(msg)
		if selected {
			// Stack was selected, update and reload
			selectedStack := m.stackSelector.SelectedStack()
			if selectedStack != "" {
				return m, selectStack(selectedStack)
			}
		}
		return m, cmd
	}

	// Handle help toggle first
	if key.Matches(msg, ui.Keys.Help) {
		m.showHelp = !m.showHelp
		return m, nil
	}

	// If help is showing, handle scrolling or close on other keys
	if m.showHelp {
		// Allow scrolling keys
		if key.Matches(msg, ui.Keys.Up) || key.Matches(msg, ui.Keys.Down) ||
			key.Matches(msg, ui.Keys.PageUp) || key.Matches(msg, ui.Keys.PageDown) {
			m.help.Update(msg)
			return m, nil
		}
		// Esc or q closes help
		if key.Matches(msg, ui.Keys.Escape) || key.Matches(msg, ui.Keys.Quit) {
			m.showHelp = false
			return m, nil
		}
		// Any other key is ignored while help is open
		return m, nil
	}

	// Escape handling
	if key.Matches(msg, ui.Keys.Escape) {
		return m.handleEscape()
	}

	// Quit
	if key.Matches(msg, ui.Keys.Quit) {
		m.quitting = true
		return m, tea.Quit
	}

	// Details panel toggle
	if key.Matches(msg, ui.Keys.ToggleDetails) {
		return m.handleToggleDetails()
	}

	// Stack selector toggle
	if key.Matches(msg, ui.Keys.SelectStack) {
		m.stackSelector.SetLoading(true)
		m.stackSelector.Show()
		return m, fetchStacksList
	}

	// Workspace selector toggle
	if key.Matches(msg, ui.Keys.SelectWorkspace) {
		m.workspaceSelector.SetLoading(true)
		m.workspaceSelector.Show()
		return m, fetchWorkspacesList
	}

	// History view toggle
	if key.Matches(msg, ui.Keys.ViewHistory) {
		return m, m.switchToHistoryView()
	}

	// Import resource (only in preview view for create operations)
	if key.Matches(msg, ui.Keys.Import) {
		if m.viewMode == ui.ViewPreview {
			item := m.resourceList.SelectedItem()
			if item != nil && item.Op == ui.OpCreate {
				m.importModal.Show(item.Type, item.Name, item.URN, item.Parent)
				// Fetch import suggestions from plugins
				return m, m.fetchImportSuggestions(item.Type, item.Name, item.URN, item.Parent, item.Inputs)
			}
		}
	}

	// Delete from state (only in stack view, not for pulumi:pulumi:Stack)
	if key.Matches(msg, ui.Keys.DeleteFromState) {
		if m.viewMode == ui.ViewStack {
			item := m.resourceList.SelectedItem()
			if item != nil && item.Type != "pulumi:pulumi:Stack" {
				m.confirmModal.SetLabels("Cancel", "Delete")
				m.confirmModal.ShowWithContext(
					"Delete from State",
					fmt.Sprintf("Remove '%s' from Pulumi state?\n\nType: %s", item.Name, item.Type),
					"This will NOT delete the actual resource.\nThe resource will become unmanaged by Pulumi.",
					item.URN,
					item.Name,
					item.Type,
				)
				return m, nil
			}
		}
	}

	// Preview operations (lowercase u/r/d)
	if key.Matches(msg, ui.Keys.PreviewUp) {
		return m, m.startPreview(pulumi.OperationUp)
	}
	if key.Matches(msg, ui.Keys.PreviewRefresh) {
		return m, m.startPreview(pulumi.OperationRefresh)
	}
	if key.Matches(msg, ui.Keys.PreviewDestroy) {
		return m, m.startPreview(pulumi.OperationDestroy)
	}

	// Execute operations (ctrl+u/r/d)
	if key.Matches(msg, ui.Keys.ExecuteUp) {
		return m, m.maybeConfirmExecution(pulumi.OperationUp)
	}
	if key.Matches(msg, ui.Keys.ExecuteRefresh) {
		return m, m.maybeConfirmExecution(pulumi.OperationRefresh)
	}
	if key.Matches(msg, ui.Keys.ExecuteDestroy) {
		return m, m.maybeConfirmExecution(pulumi.OperationDestroy)
	}

	// Forward keys to appropriate list for cursor/selection handling
	return m.handleListNavigation(msg)
}

// handleEscape handles escape key presses based on current state
func (m Model) handleEscape() (tea.Model, tea.Cmd) {
	// Clear text selection first if active (details panel)
	if m.details.HasSelection() {
		m.details.ClearSelection()
		return m, nil
	}
	// Cancel visual mode
	if m.resourceList.VisualMode() {
		cmd := m.resourceList.Update(tea.KeyMsg{Type: tea.KeyEscape})
		return m, cmd
	}
	// Close details panel if visible
	if m.viewMode == ui.ViewHistory && m.historyDetails.Visible() {
		m.historyDetails.Hide()
		return m, nil
	}
	if m.details.Visible() {
		m.details.Hide()
		return m, nil
	}
	// Cancel running operation
	if m.viewMode == ui.ViewExecute && m.operationCancel != nil {
		m.operationCancel()
		return m, nil
	}
	// Go back to stack view from preview, history, or completed execution
	if m.viewMode == ui.ViewPreview || m.viewMode == ui.ViewExecute || m.viewMode == ui.ViewHistory {
		return m, m.switchToStackView()
	}
	return m, nil
}

// handleToggleDetails toggles the appropriate details panel based on view mode
func (m Model) handleToggleDetails() (tea.Model, tea.Cmd) {
	if m.viewMode == ui.ViewHistory {
		// Toggle history details panel
		m.historyDetails.Toggle()
		if m.historyDetails.Visible() {
			m.historyDetails.SetItem(m.historyList.SelectedItem())
		}
	} else {
		// Toggle resource details panel
		m.details.Toggle()
		if m.details.Visible() {
			m.details.SetResource(m.resourceList.SelectedItem())
		}
	}
	return m, nil
}

// handleListNavigation forwards keys to the appropriate list component
func (m Model) handleListNavigation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.viewMode == ui.ViewHistory {
		// Handle scrolling in history details panel
		if m.historyDetails.Visible() {
			if m.handleDetailsPanelScroll(msg, m.historyDetails) {
				return m, nil
			}
		}
		cmd := m.historyList.Update(msg)
		// Update history details panel with newly selected item
		if m.historyDetails.Visible() {
			m.historyDetails.SetItem(m.historyList.SelectedItem())
		}
		return m, cmd
	}

	// Handle scrolling in details panel
	if m.details.Visible() {
		if m.handleDetailsPanelScroll(msg, m.details) {
			return m, nil
		}
	}

	cmd := m.resourceList.Update(msg)
	// Update details panel with newly selected resource
	if m.details.Visible() {
		m.details.SetResource(m.resourceList.SelectedItem())
	}
	return m, cmd
}

// scrollablePanel is an interface for panels that support scrolling
type scrollablePanel interface {
	ScrollUp(lines int)
	ScrollDown(lines int)
	SetScrollOffset(offset int)
	ScrollOffset() int
}

// handleDetailsPanelScroll handles scroll keys for detail panels
// Returns true if the key was handled (consumed for scrolling)
func (m Model) handleDetailsPanelScroll(msg tea.KeyMsg, panel scrollablePanel) bool {
	switch {
	case key.Matches(msg, ui.Keys.Up):
		panel.ScrollUp(1)
		return true
	case key.Matches(msg, ui.Keys.Down):
		panel.ScrollDown(1)
		return true
	case key.Matches(msg, ui.Keys.PageUp):
		panel.ScrollUp(10)
		return true
	case key.Matches(msg, ui.Keys.PageDown):
		panel.ScrollDown(10)
		return true
	case key.Matches(msg, ui.Keys.Home):
		panel.SetScrollOffset(0)
		return true
	case key.Matches(msg, ui.Keys.End):
		// Set to a large value - the render will clamp it
		panel.SetScrollOffset(9999)
		return true
	}
	return false
}
