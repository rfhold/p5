package main

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/rfhold/p5/internal/pulumi"
	"github.com/rfhold/p5/internal/ui"
)

// handleKeyPress routes keyboard events to the appropriate handler based on focus stack
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Route to current focus owner - O(1) lookup
	switch m.ui.Focus.Current() {
	case ui.FocusErrorModal:
		return m.updateErrorModal(msg)
	case ui.FocusConfirmModal:
		return m.updateConfirmModal(msg)
	case ui.FocusImportModal:
		return m.updateImportModal(msg)
	case ui.FocusStackInitModal:
		return m.updateStackInitModal(msg)
	case ui.FocusWorkspaceSelector:
		return m.updateWorkspaceSelector(msg)
	case ui.FocusStackSelector:
		return m.updateStackSelector(msg)
	case ui.FocusHelp:
		return m.updateHelp(msg)
	case ui.FocusDetailsPanel:
		return m.updateDetailsPanel(msg)
	case ui.FocusMain:
		return m.updateMain(msg)
	}
	return m, nil
}

// updateErrorModal handles keys when error modal has focus
func (m Model) updateErrorModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	dismissed, cmd := m.ui.ErrorModal.Update(msg)
	if dismissed {
		m.hideErrorModal()
	}
	return m, cmd
}

// updateConfirmModal handles keys when confirm modal has focus
func (m Model) updateConfirmModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	confirmed, cancelled, cmd := m.ui.ConfirmModal.Update(msg)
	if confirmed {
		// Check if this is a pending operation confirmation
		if m.state.PendingOperation != nil {
			op := *m.state.PendingOperation
			m.state.PendingOperation = nil
			m.hideConfirmModal()
			return m, m.startExecution(op)
		}
		// Otherwise it's a state delete confirmation
		return m, m.executeStateDelete()
	}
	if cancelled {
		m.state.PendingOperation = nil
		m.hideConfirmModal()
	}
	return m, cmd
}

// updateImportModal handles keys when import modal has focus
func (m Model) updateImportModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	confirmed, cmd := m.ui.ImportModal.Update(msg)
	if confirmed {
		// User confirmed import, execute it
		return m, m.executeImport()
	}
	// Check if modal was dismissed (ESC pressed)
	if !m.ui.ImportModal.Visible() {
		m.ui.Focus.Remove(ui.FocusImportModal)
	}
	return m, cmd
}

// updateStackInitModal handles keys when stack init modal has focus
func (m Model) updateStackInitModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	action, cmd := m.ui.StackInitModal.Update(msg)
	switch action {
	case ui.StepModalActionConfirm:
		// User completed all configured steps, init the stack
		name := m.ui.StackInitModal.GetStackName()
		provider := m.ui.StackInitModal.GetSecretsProvider()
		passphrase := m.ui.StackInitModal.GetPassphrase()
		return m, m.initStack(name, provider, passphrase)
	case ui.StepModalActionNext:
		// After secrets provider step (step 1 -> step 2), check if we should skip passphrase
		currentStep := m.ui.StackInitModal.CurrentStep()
		if currentStep == 2 && m.ui.StackInitModal.ShouldSkipPassphrase() {
			// Skip passphrase step, init directly
			name := m.ui.StackInitModal.GetStackName()
			provider := m.ui.StackInitModal.GetSecretsProvider()
			return m, m.initStack(name, provider, "")
		}
	case ui.StepModalActionCancel:
		m.hideStackInitModal()
	}
	return m, cmd
}

// updateWorkspaceSelector handles keys when workspace selector has focus
func (m Model) updateWorkspaceSelector(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	selected, cmd := m.ui.WorkspaceSelector.Update(msg)
	if selected {
		// Workspace was selected, update and reload
		selectedWs := m.ui.WorkspaceSelector.SelectedWorkspace()
		if selectedWs != nil {
			return m, selectWorkspace(selectedWs.Path)
		}
	}
	// Check if selector was dismissed (ESC pressed)
	if !m.ui.WorkspaceSelector.Visible() {
		m.ui.Focus.Remove(ui.FocusWorkspaceSelector)
	}
	return m, cmd
}

// updateStackSelector handles keys when stack selector has focus
func (m Model) updateStackSelector(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	selected, cmd := m.ui.StackSelector.Update(msg)
	if selected {
		// Check if "new stack" was selected
		if m.ui.StackSelector.IsNewStackSelected() {
			m.hideStackSelector()
			m.showStackInitModal()
			// Pass auth env from plugins for passphrase detection
			if m.deps != nil && m.deps.PluginProvider != nil {
				m.ui.StackInitModal.SetAuthEnv(m.deps.PluginProvider.GetMergedAuthEnv())
			}
			return m, tea.Batch(m.fetchWhoAmI(), m.fetchStackFiles())
		}
		// Stack was selected, update and reload
		selectedStack := m.ui.StackSelector.SelectedStack()
		if selectedStack != "" {
			return m, m.selectStack(selectedStack)
		}
	}
	// Check if selector was dismissed (ESC pressed)
	if !m.ui.StackSelector.Visible() {
		m.ui.Focus.Remove(ui.FocusStackSelector)
	}
	return m, cmd
}

// updateHelp handles keys when help dialog has focus
func (m Model) updateHelp(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Allow scrolling keys
	if key.Matches(msg, ui.Keys.Up) || key.Matches(msg, ui.Keys.Down) ||
		key.Matches(msg, ui.Keys.PageUp) || key.Matches(msg, ui.Keys.PageDown) {
		m.ui.Help.Update(msg)
		return m, nil
	}
	// Esc, q, or ? closes help
	if key.Matches(msg, ui.Keys.Escape) || key.Matches(msg, ui.Keys.Quit) || key.Matches(msg, ui.Keys.Help) {
		m.hideHelp()
		return m, nil
	}
	// Any other key is ignored while help is open
	return m, nil
}

// updateDetailsPanel handles keys when details panel has focus
func (m Model) updateDetailsPanel(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Get the appropriate panel based on view mode
	var panel scrollablePanel
	if m.ui.ViewMode == ui.ViewHistory {
		panel = m.ui.HistoryDetails
	} else {
		panel = m.ui.Details
	}

	// Handle scroll keys
	switch {
	case key.Matches(msg, ui.Keys.Up):
		panel.ScrollUp(1)
		return m, nil
	case key.Matches(msg, ui.Keys.Down):
		panel.ScrollDown(1)
		return m, nil
	case key.Matches(msg, ui.Keys.PageUp):
		panel.ScrollUp(10)
		return m, nil
	case key.Matches(msg, ui.Keys.PageDown):
		panel.ScrollDown(10)
		return m, nil
	case key.Matches(msg, ui.Keys.Home):
		panel.SetScrollOffset(0)
		return m, nil
	case key.Matches(msg, ui.Keys.End):
		// Set to a large value - the render will clamp it
		panel.SetScrollOffset(9999)
		return m, nil
	case key.Matches(msg, ui.Keys.Escape), key.Matches(msg, ui.Keys.ToggleDetails):
		// Close details panel
		m.hideDetailsPanel()
		return m, nil
	case key.Matches(msg, ui.Keys.Help):
		// Help can open on top of details
		m.showHelp()
		return m, nil
	}

	// Other keys close the panel and fall through to main
	m.hideDetailsPanel()
	return m.updateMain(msg)
}

// updateMain handles keys when no modal is active
func (m Model) updateMain(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Help toggle
	if key.Matches(msg, ui.Keys.Help) {
		m.showHelp()
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
		m.toggleDetailsPanel()
		return m, nil
	}

	// Stack selector toggle
	if key.Matches(msg, ui.Keys.SelectStack) {
		m.showStackSelector()
		return m, m.fetchStacksList()
	}

	// Workspace selector toggle
	if key.Matches(msg, ui.Keys.SelectWorkspace) {
		m.showWorkspaceSelector()
		return m, m.fetchWorkspacesList()
	}

	// History view toggle
	if key.Matches(msg, ui.Keys.ViewHistory) {
		return m, m.switchToHistoryView()
	}

	// Import resource (only in preview view for create operations)
	if key.Matches(msg, ui.Keys.Import) {
		item := m.ui.ResourceList.SelectedItem()
		if CanImportResource(m.ui.ViewMode, item) {
			m.showImportModal(item.Type, item.Name, item.URN, item.Parent)
			// Fetch import suggestions from plugins
			return m, m.fetchImportSuggestions(item.Type, item.Name, item.URN, item.Parent, item.Provider, item.Inputs, item.ProviderInputs)
		}
	}

	// Delete from state (only in stack view, not for pulumi:pulumi:Stack)
	if key.Matches(msg, ui.Keys.DeleteFromState) {
		item := m.ui.ResourceList.SelectedItem()
		if CanDeleteFromState(m.ui.ViewMode, item) {
			m.ui.ConfirmModal.SetLabels("Cancel", "Delete")
			m.ui.ConfirmModal.ShowWithContext(
				"Delete from State",
				fmt.Sprintf("Remove '%s' from Pulumi state?\n\nType: %s", item.Name, item.Type),
				"This will NOT delete the actual resource.\nThe resource will become unmanaged by Pulumi.",
				item.URN,
				item.Name,
				item.Type,
			)
			m.showConfirmModal()
			return m, nil
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
	// Determine action using pure function
	action := DetermineEscapeAction(m.ui.ViewMode, m.state.OpState, m.ui.ResourceList.VisualMode())

	switch action {
	case EscapeActionExitVisualMode:
		cmd := m.ui.ResourceList.Update(tea.KeyMsg{Type: tea.KeyEscape})
		return m, cmd
	case EscapeActionCancelOp:
		m.cancelOperation()
		return m, nil
	case EscapeActionNavigateBack:
		return m, m.switchToStackView()
	}

	return m, nil
}

// handleListNavigation forwards keys to the appropriate list component
func (m Model) handleListNavigation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.ui.ViewMode == ui.ViewHistory {
		cmd := m.ui.HistoryList.Update(msg)
		// Update history details panel with newly selected item if visible
		if m.ui.Focus.Has(ui.FocusDetailsPanel) {
			m.ui.HistoryDetails.SetItem(m.ui.HistoryList.SelectedItem())
		}
		return m, cmd
	}

	cmd := m.ui.ResourceList.Update(msg)
	// Update details panel with newly selected resource if visible
	if m.ui.Focus.Has(ui.FocusDetailsPanel) {
		m.ui.Details.SetResource(m.ui.ResourceList.SelectedItem())
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
