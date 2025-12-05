package main

import (
	"fmt"
	"log"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/rfhold/p5/internal/pulumi"
	"github.com/rfhold/p5/internal/ui"
)

// Operation handlers - handles preview and execute operations

// transitionOpTo transitions to a new operation state with debug logging.
// This makes operation state changes explicit and traceable.
func (m *Model) transitionOpTo(newState OperationState) {
	if m.state.OpState != newState {
		log.Printf("[Op] %s → %s", m.state.OpState.String(), newState.String())
		m.state.OpState = newState
	}
}

// resetOperation resets the operation state machine to idle.
// Call this when switching away from preview/execute views.
func (m *Model) resetOperation() {
	if m.state.OpState != OpIdle {
		log.Printf("[Op] Reset: %s → Idle", m.state.OpState.String())
		m.state.OpState = OpIdle
	}
	// Clean up cancel function if present
	if m.operationCancel != nil {
		m.operationCancel = nil
	}
}

// cancelOperation requests cancellation of the current operation.
// Returns true if cancellation was initiated.
func (m *Model) cancelOperation() bool {
	if m.state.OpState == OpRunning && m.operationCancel != nil {
		log.Printf("[Op] Cancel requested: Running → Cancelling")
		m.state.OpState = OpCancelling
		m.operationCancel()
		return true
	}
	return false
}

// handleInitPreview handles starting a preview from Init
func (m Model) handleInitPreview(msg initPreviewMsg) (tea.Model, tea.Cmd) {
	// Transition to Running state (Starting was set when preview was initiated)
	m.transitionOpTo(OpRunning)

	// Store the channel and start listening for events
	m.previewCh = msg.ch
	m.ui.ResourceList.SetLoading(true, fmt.Sprintf("Running %s preview...", msg.op.String()))
	return m, waitForPreviewEvent(m.previewCh)
}

// handleStackResources handles loaded stack resources
// State: InitLoadingResources → InitComplete (during init)
func (m Model) handleStackResources(msg stackResourcesMsg) (tea.Model, tea.Cmd) {
	// Use pure function to convert resources
	items := ConvertResourcesToItems(msg)

	m.ui.ResourceList.SetItems(items)
	m.ui.Header.SetSummary(m.ui.ResourceList.Summary(), ui.HeaderDone)
	// Update details panel with current selection
	if m.ui.Details.Visible() {
		m.ui.Details.SetResource(m.ui.ResourceList.SelectedItem())
	}

	// Mark initialization as complete
	if m.state.InitState == InitLoadingResources {
		m.transitionTo(InitComplete)
	}

	return m, nil
}

// handlePreviewEvent handles streaming preview events
// State: InitLoadingResources → InitComplete (when preview completes during init)
// OpState: Running → Complete/Error (when preview finishes)
func (m Model) handlePreviewEvent(msg previewEventMsg) (tea.Model, tea.Cmd) {
	event := pulumi.PreviewEvent(msg)

	// Use pure function to process event and get state changes
	result := ProcessPreviewEvent(event, m.state.OpState, m.state.InitState)

	// Apply state transitions
	if result.NewOpState != m.state.OpState {
		m.transitionOpTo(result.NewOpState)
	}

	// Handle error case
	if result.HasError {
		m.ui.ResourceList.SetError(result.Error)
		m.ui.Header.SetSummary(m.ui.ResourceList.Summary(), ui.HeaderError)
		if result.InitDone {
			m.transitionTo(InitComplete)
		}
		return m, nil
	}

	// Handle done case
	if event.Done {
		m.ui.ResourceList.SetLoading(false, "")
		m.ui.Header.SetSummary(m.ui.ResourceList.Summary(), ui.HeaderDone)
		if result.InitDone {
			m.transitionTo(InitComplete)
		}
		return m, nil
	}

	// Handle step case - add resource item
	if result.Item != nil {
		m.ui.ResourceList.AddItem(*result.Item)
		m.ui.Header.SetSummary(m.ui.ResourceList.Summary(), ui.HeaderRunning)
		// Update details panel with current selection
		if m.ui.Details.Visible() {
			m.ui.Details.SetResource(m.ui.ResourceList.SelectedItem())
		}
	}

	// Continue waiting for more events
	return m, waitForPreviewEvent(m.previewCh)
}

// handleOperationEvent handles streaming execution events
// OpState: Starting → Running → Complete/Error
func (m Model) handleOperationEvent(msg operationEventMsg) (tea.Model, tea.Cmd) {
	event := pulumi.OperationEvent(msg)

	// Use pure function to process event and get state changes
	result := ProcessOperationEvent(event, m.state.OpState)

	// Apply state transitions
	if result.NewOpState != m.state.OpState {
		m.transitionOpTo(result.NewOpState)
	}

	// Handle error case
	if result.HasError {
		m.ui.ResourceList.SetError(result.Error)
		m.ui.Header.SetSummary(m.ui.ResourceList.Summary(), ui.HeaderError)
		m.operationCancel = nil // Clear so escape can navigate back
		return m, nil
	}

	// Handle done case
	if result.Done {
		m.ui.ResourceList.SetLoading(false, "")
		m.ui.Header.SetSummary(m.ui.ResourceList.Summary(), ui.HeaderDone)
		m.operationCancel = nil // Clear so escape can navigate back
		return m, nil
	}

	// Handle item case - add/update resource item
	if result.Item != nil {
		m.ui.ResourceList.AddItem(*result.Item)
		m.ui.Header.SetSummary(m.ui.ResourceList.Summary(), ui.HeaderRunning)
		// Update details panel with current selection
		if m.ui.Details.Visible() {
			m.ui.Details.SetResource(m.ui.ResourceList.SelectedItem())
		}
	}

	// Continue waiting for more events
	return m, waitForOperationEvent(m.operationCh)
}

// handleImportResult handles import command result
func (m Model) handleImportResult(msg importResultMsg) (tea.Model, tea.Cmd) {
	m.hideImportModal() // Always hide import modal when result comes in
	if msg == nil {
		m.showErrorModal(
			"Import Failed",
			"Unknown error occurred during import",
			"No additional details available",
		)
		return m, nil
	}
	if msg.Success {
		// Import succeeded, show success and refresh the preview
		cmds := []tea.Cmd{
			m.ui.Toast.Show(fmt.Sprintf("Imported %s successfully", m.ui.ImportModal.GetResourceName())),
			m.startPreview(m.state.Operation), // Re-run preview to show updated state
		}
		return m, tea.Batch(cmds...)
	}
	// Import failed, show error modal with full details
	summary := fmt.Sprintf("Failed to import '%s' (%s)",
		m.ui.ImportModal.GetResourceName(),
		m.ui.ImportModal.GetResourceType())
	details := msg.Output
	if details == "" && msg.Error != nil {
		details = msg.Error.Error()
	}
	m.showErrorModal("Import Failed", summary, details)
	return m, nil
}

// handleStateDeleteResult handles state delete command result
func (m Model) handleStateDeleteResult(msg stateDeleteResultMsg) (tea.Model, tea.Cmd) {
	resourceName := m.ui.ConfirmModal.GetContextName()
	m.hideConfirmModal()
	if msg == nil {
		return m, m.ui.Toast.Show("Delete from state failed: unknown error")
	}
	if msg.Success {
		// State delete succeeded, show success and refresh the stack view
		cmds := []tea.Cmd{
			m.ui.Toast.Show(fmt.Sprintf("Removed '%s' from state", resourceName)),
			m.loadStackResources(), // Reload to show updated state
		}
		return m, tea.Batch(cmds...)
	}
	// State delete failed, show error
	errMsg := "Delete from state failed"
	if msg.Error != nil {
		errMsg = msg.Error.Error()
	}
	return m, m.ui.Toast.Show(errMsg)
}

// handleStackHistory handles loaded stack history
func (m Model) handleStackHistory(msg stackHistoryMsg) (tea.Model, tea.Cmd) {
	// Use pure function to convert history
	items := ConvertHistoryToItems(msg)

	m.ui.HistoryList.SetItems(items)
	m.ui.Header.SetSummary(ui.ResourceSummary{Total: len(items)}, ui.HeaderDone)
	return m, nil
}

// handleImportSuggestions handles import suggestions from plugins
func (m Model) handleImportSuggestions(msg importSuggestionsMsg) (tea.Model, tea.Cmd) {
	// Use pure function to convert suggestions
	suggestions := ConvertImportSuggestions(msg)
	m.ui.ImportModal.SetSuggestions(suggestions)
	return m, nil
}

// handleImportSuggestionsError handles import suggestions error
func (m Model) handleImportSuggestionsError(msg importSuggestionsErrMsg) (tea.Model, tea.Cmd) {
	// Just stop loading, no suggestions available
	m.ui.ImportModal.SetSuggestions(nil)
	return m, nil
}
