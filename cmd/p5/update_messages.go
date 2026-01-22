package main

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/rfhold/p5/internal/ui"
)

// handleMessage routes all non-key, non-window, non-mouse messages to appropriate handlers.
func (m Model) handleMessage(msg tea.Msg) (tea.Model, tea.Cmd) {
	if model, cmd, handled := m.handleInitMessages(msg); handled {
		return model, cmd
	}
	if model, cmd, handled := m.handleOperationMessages(msg); handled {
		return model, cmd
	}
	if model, cmd, handled := m.handleSelectionMessages(msg); handled {
		return model, cmd
	}
	if model, cmd, handled := m.handleUIMessages(msg); handled {
		return model, cmd
	}
	return m, nil
}

func (m Model) handleInitMessages(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case workspaceCheckMsg:
		model, cmd := m.handleWorkspaceCheck(msg)
		return model, cmd, true
	case pluginInitDoneMsg:
		model, cmd := m.handlePluginInitDone(msg)
		return model, cmd, true
	case pluginAuthResultMsg:
		model, cmd := m.handlePluginAuthResult(msg)
		return model, cmd, true
	case pluginAuthErrorMsg:
		model, cmd := m.handlePluginAuthError(msg)
		return model, cmd, true
	case authCompleteMsg:
		model, cmd := m.handleAuthComplete(msg)
		return model, cmd, true
	case projectInfoMsg:
		model, cmd := m.handleProjectInfo(msg)
		return model, cmd, true
	case errMsg: //nolint:staticcheck // SA4020: type aliases to error are dispatched by explicit cast at call site
		model, cmd := m.handleError(msg)
		return model, cmd, true
	case whoAmIMsg:
		model, cmd := m.handleWhoAmI(msg)
		return model, cmd, true
	case stackFilesMsg:
		model, cmd := m.handleStackFiles(msg)
		return model, cmd, true
	case stackInitResultMsg:
		model, cmd := m.handleStackInitResult(msg)
		return model, cmd, true
	}
	return m, nil, false
}

func (m Model) handleOperationMessages(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case initPreviewMsg:
		model, cmd := m.handleInitPreview(msg)
		return model, cmd, true
	case stackResourcesMsg:
		model, cmd := m.handleStackResources(msg)
		return model, cmd, true
	case previewEventMsg:
		model, cmd := m.handlePreviewEvent(msg)
		return model, cmd, true
	case operationEventMsg:
		model, cmd := m.handleOperationEvent(msg)
		return model, cmd, true
	case importResultMsg:
		model, cmd := m.handleImportResult(msg)
		return model, cmd, true
	case stateDeleteResultMsg:
		model, cmd := m.handleStateDeleteResult(msg)
		return model, cmd, true
	case bulkStateDeleteResultMsg:
		model, cmd := m.handleBulkStateDeleteResult(msg)
		return model, cmd, true
	case protectResultMsg:
		model, cmd := m.handleProtectResult(msg)
		return model, cmd, true
	case stackHistoryMsg:
		model, cmd := m.handleStackHistory(msg)
		return model, cmd, true
	case importSuggestionsMsg:
		model, cmd := m.handleImportSuggestions(msg)
		return model, cmd, true
	case importSuggestionsErrMsg:
		model, cmd := m.handleImportSuggestionsError(msg)
		return model, cmd, true
	case openResourceActionMsg:
		model, cmd := m.handleOpenResourceAction(msg)
		return model, cmd, true
	case openResourceErrMsg: //nolint:staticcheck // SA4020: type aliases to error are dispatched by explicit cast at call site
		model, cmd := m.handleOpenResourceError(msg)
		return model, cmd, true
	case openResourceExecDoneMsg:
		model, cmd := m.handleOpenResourceExecDone(msg)
		return model, cmd, true
	}
	return m, nil, false
}

func (m Model) handleSelectionMessages(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case stacksListMsg:
		model, cmd := m.handleStacksList(msg)
		return model, cmd, true
	case stackSelectedMsg:
		model, cmd := m.handleStackSelected(msg)
		return model, cmd, true
	case workspacesListMsg:
		model, cmd := m.handleWorkspacesList(msg)
		return model, cmd, true
	case workspaceSelectedMsg:
		model, cmd := m.handleWorkspaceSelected(msg)
		return model, cmd, true
	}
	return m, nil, false
}

func (m Model) handleUIMessages(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		model, cmd := m.handleSpinnerTick(msg)
		return model, cmd, true
	case ui.CopiedToClipboardMsg:
		model, cmd := m.handleCopiedToClipboard(msg)
		return model, cmd, true
	case ui.ToastHideMsg:
		model, cmd := m.handleToastHide()
		return model, cmd, true
	case ui.FlashClearMsg:
		model, cmd := m.handleFlashClear()
		return model, cmd, true
	}
	return m, nil, false
}
