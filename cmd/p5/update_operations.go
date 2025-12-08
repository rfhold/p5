package main

import (
	"fmt"
	"maps"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/rfhold/p5/internal/plugins/proto"
	"github.com/rfhold/p5/internal/pulumi"
	"github.com/rfhold/p5/internal/ui"
)

// transitionOpTo transitions to a new operation state with debug logging.
func (m *Model) transitionOpTo(newState OperationState) {
	if m.state.OpState != newState {
		m.deps.Logger.Debug("operation state transition",
			"from", m.state.OpState.String(),
			"to", newState.String())
		m.state.OpState = newState
	}
}

// resetOperation resets the operation state machine to idle.
func (m *Model) resetOperation() {
	if m.state.OpState != OpIdle {
		m.deps.Logger.Debug("operation reset",
			"from", m.state.OpState.String(),
			"to", "Idle")
		m.state.OpState = OpIdle
	}
	if m.operationCancel != nil {
		m.operationCancel = nil
	}
}

// cancelOperation requests cancellation of the current operation.
// Returns true if cancellation was initiated.
func (m *Model) cancelOperation() bool {
	if m.state.OpState == OpRunning && m.operationCancel != nil {
		m.deps.Logger.Debug("operation cancel requested",
			"from", "Running",
			"to", "Cancelling")
		m.state.OpState = OpCancelling
		m.operationCancel()
		return true
	}
	return false
}

// handleInitPreview handles starting a preview from Init
func (m Model) handleInitPreview(msg initPreviewMsg) (tea.Model, tea.Cmd) {
	m.transitionOpTo(OpRunning)
	m.previewCh = msg.ch
	m.ui.ResourceList.SetLoading(true, fmt.Sprintf("Running %s preview...", msg.op.String()))
	return m, waitForPreviewEvent(m.previewCh)
}

// handleStackResources handles loaded stack resources.
func (m Model) handleStackResources(msg stackResourcesMsg) (tea.Model, tea.Cmd) { //nolint:unparam // Bubble Tea handler signature
	items := ConvertResourcesToItems(msg)

	m.ui.ResourceList.SetItems(items)
	m.ui.Header.SetSummary(m.ui.ResourceList.Summary(), ui.HeaderDone)
	if m.ui.Details.Visible() {
		m.ui.Details.SetResource(m.ui.ResourceList.SelectedItem())
	}

	if m.state.InitState == InitLoadingResources {
		m.transitionTo(InitComplete)
	}

	return m, nil
}

// handlePreviewEvent handles streaming preview events.
func (m Model) handlePreviewEvent(msg previewEventMsg) (tea.Model, tea.Cmd) {
	event := pulumi.PreviewEvent(msg)
	result := ProcessPreviewEvent(event, m.state.OpState, m.state.InitState)

	if result.NewOpState != m.state.OpState {
		m.transitionOpTo(result.NewOpState)
	}

	// Handle error case
	if result.HasError {
		m.ui.ResourceList.SetError(result.Error)
		m.ui.Header.SetSummary(m.ui.ResourceList.Summary(), ui.HeaderError)
		m.previewCancel = nil
		if result.InitDone {
			m.transitionTo(InitComplete)
		}
		return m, nil
	}

	if event.Done {
		m.ui.ResourceList.SetLoading(false, "")
		m.ui.Header.SetSummary(m.ui.ResourceList.Summary(), ui.HeaderDone)
		m.previewCancel = nil
		if result.InitDone {
			m.transitionTo(InitComplete)
		}
		return m, nil
	}

	if result.Item != nil {
		m.ui.ResourceList.AddItem(*result.Item)
		m.ui.Header.SetSummary(m.ui.ResourceList.Summary(), ui.HeaderRunning)
		if m.ui.Details.Visible() {
			m.ui.Details.SetResource(m.ui.ResourceList.SelectedItem())
		}
	}

	return m, waitForPreviewEvent(m.previewCh)
}

// handleOperationEvent handles streaming execution events.
func (m Model) handleOperationEvent(msg operationEventMsg) (tea.Model, tea.Cmd) {
	event := pulumi.OperationEvent(msg)
	result := ProcessOperationEvent(event, m.state.OpState)

	if result.NewOpState != m.state.OpState {
		m.transitionOpTo(result.NewOpState)
	}

	if result.HasError {
		m.ui.ResourceList.SetError(result.Error)
		m.ui.Header.SetSummary(m.ui.ResourceList.Summary(), ui.HeaderError)
		m.operationCancel = nil
		return m, nil
	}

	if result.Done {
		m.ui.ResourceList.SetLoading(false, "")
		m.ui.Header.SetSummary(m.ui.ResourceList.Summary(), ui.HeaderDone)
		m.operationCancel = nil
		return m, nil
	}

	if result.Item != nil {
		m.ui.ResourceList.AddItem(*result.Item)
		m.ui.Header.SetSummary(m.ui.ResourceList.Summary(), ui.HeaderRunning)
		if m.ui.Details.Visible() {
			m.ui.Details.SetResource(m.ui.ResourceList.SelectedItem())
		}
	}

	return m, waitForOperationEvent(m.operationCh)
}

// handleImportResult handles import command result
func (m Model) handleImportResult(msg importResultMsg) (tea.Model, tea.Cmd) {
	m.hideImportModal()
	if msg == nil {
		m.showErrorModal(
			"Import Failed",
			"Unknown error occurred during import",
			"No additional details available",
		)
		return m, nil
	}
	if msg.Success {
		cmds := []tea.Cmd{
			m.ui.Toast.Show(fmt.Sprintf("Imported %s successfully", m.ui.ImportModal.GetResourceName())),
			m.startPreview(m.state.Operation),
		}
		return m, tea.Batch(cmds...)
	}
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
		cmds := []tea.Cmd{
			m.ui.Toast.Show(fmt.Sprintf("Removed '%s' from state", resourceName)),
			m.loadStackResources(),
		}
		return m, tea.Batch(cmds...)
	}
	errMsg := "Delete from state failed"
	if msg.Error != nil {
		errMsg = msg.Error.Error()
	}
	return m, m.ui.Toast.Show(errMsg)
}

// handleStackHistory handles loaded stack history
func (m Model) handleStackHistory(msg stackHistoryMsg) (tea.Model, tea.Cmd) { //nolint:unparam // Bubble Tea handler signature
	items := ConvertHistoryToItems(msg)

	m.ui.HistoryList.SetItems(items)
	m.ui.Header.SetSummary(ui.ResourceSummary{Total: len(items)}, ui.HeaderDone)
	return m, nil
}

// handleImportSuggestions handles import suggestions from plugins
func (m Model) handleImportSuggestions(msg importSuggestionsMsg) (tea.Model, tea.Cmd) { //nolint:unparam // Bubble Tea handler signature
	suggestions := ConvertImportSuggestions(msg)
	m.ui.ImportModal.SetSuggestions(suggestions)
	return m, nil
}

// handleImportSuggestionsError handles import suggestions error
func (m Model) handleImportSuggestionsError(_ importSuggestionsErrMsg) (tea.Model, tea.Cmd) { //nolint:unparam // Bubble Tea handler signature
	m.ui.ImportModal.SetSuggestions(nil)
	return m, nil
}

// handleOpenResourceAction handles the response from plugin open resource query
func (m Model) handleOpenResourceAction(msg openResourceActionMsg) (tea.Model, tea.Cmd) {
	resp := msg.Response
	if resp == nil {
		// No plugin could open this resource
		return m, m.ui.Toast.Show("No plugin can open this resource type")
	}

	if !resp.CanOpen {
		return m, m.ui.Toast.Show("Resource type not supported for opening")
	}

	if resp.Error != "" {
		return m, m.ui.Toast.Show("Open resource failed: " + resp.Error)
	}

	action := resp.Action
	if action == nil {
		return m, m.ui.Toast.Show("Plugin returned no action")
	}

	switch action.Type {
	case proto.OpenActionType_OPEN_ACTION_TYPE_BROWSER:
		return m, tea.Batch(
			m.ui.Toast.Show("Opening in browser..."),
			openInBrowser(action.Url),
		)
	case proto.OpenActionType_OPEN_ACTION_TYPE_EXEC:
		// Convert proto env map to Go map
		env := make(map[string]string)
		maps.Copy(env, action.Env)
		return m, openWithExec(action.Command, action.Args, env)
	default:
		return m, m.ui.Toast.Show("Unknown open action type")
	}
}

// handleOpenResourceError handles errors from plugin open resource query
func (m Model) handleOpenResourceError(msg openResourceErrMsg) (tea.Model, tea.Cmd) {
	return m, m.ui.Toast.Show("Open resource failed: " + error(msg).Error())
}

// handleOpenResourceExecDone handles completion of an exec-based open action
func (m Model) handleOpenResourceExecDone(msg openResourceExecDoneMsg) (tea.Model, tea.Cmd) {
	if msg.Error != nil {
		return m, m.ui.Toast.Show("Program exited with error: " + msg.Error.Error())
	}
	return m, nil
}
