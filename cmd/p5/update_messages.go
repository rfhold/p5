package main

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/rfhold/p5/internal/ui"
)

// handleMessage routes all non-key, non-window, non-mouse messages to appropriate handlers.
// This is a thin router that delegates to domain-specific handlers in:
// - update_init.go: initialization and plugin handlers
// - update_operations.go: preview/execute operation handlers
// - update_selection.go: stack/workspace selection handlers
// - update_ui.go: toast/clipboard/resize handlers
func (m Model) handleMessage(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	// Initialization messages
	case workspaceCheckMsg:
		return m.handleWorkspaceCheck(msg)
	case pluginInitDoneMsg:
		return m.handlePluginInitDone(msg)
	case pluginAuthResultMsg:
		return m.handlePluginAuthResult(msg)
	case pluginAuthErrorMsg:
		return m.handlePluginAuthError(msg)
	case projectInfoMsg:
		return m.handleProjectInfo(msg)
	case errMsg:
		return m.handleError(msg)
	case whoAmIMsg:
		return m.handleWhoAmI(msg)
	case stackFilesMsg:
		return m.handleStackFiles(msg)
	case stackInitResultMsg:
		return m.handleStackInitResult(msg)

	// Operation messages
	case initPreviewMsg:
		return m.handleInitPreview(msg)
	case stackResourcesMsg:
		return m.handleStackResources(msg)
	case previewEventMsg:
		return m.handlePreviewEvent(msg)
	case operationEventMsg:
		return m.handleOperationEvent(msg)
	case importResultMsg:
		return m.handleImportResult(msg)
	case stateDeleteResultMsg:
		return m.handleStateDeleteResult(msg)
	case stackHistoryMsg:
		return m.handleStackHistory(msg)
	case importSuggestionsMsg:
		return m.handleImportSuggestions(msg)
	case importSuggestionsErrMsg:
		return m.handleImportSuggestionsError(msg)
	case openResourceActionMsg:
		return m.handleOpenResourceAction(msg)
	case openResourceErrMsg:
		return m.handleOpenResourceError(msg)
	case openResourceExecDoneMsg:
		return m.handleOpenResourceExecDone(msg)

	// Selection messages
	case stacksListMsg:
		return m.handleStacksList(msg)
	case stackSelectedMsg:
		return m.handleStackSelected(msg)
	case workspacesListMsg:
		return m.handleWorkspacesList(msg)
	case workspaceSelectedMsg:
		return m.handleWorkspaceSelected(msg)

	// UI messages
	case spinner.TickMsg:
		return m.handleSpinnerTick(msg)
	case ui.CopiedToClipboardMsg:
		return m.handleCopiedToClipboard(msg)
	case ui.ToastHideMsg:
		return m.handleToastHide()
	case ui.FlashClearMsg:
		return m.handleFlashClear()
	}

	return m, nil
}
