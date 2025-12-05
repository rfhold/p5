package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/rfhold/p5/internal/pulumi"
	"github.com/rfhold/p5/internal/ui"
)

// handleMessage handles all non-key, non-window, non-mouse messages
func (m Model) handleMessage(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case projectInfoMsg:
		return m.handleProjectInfo(msg)

	case errMsg:
		return m.handleError(msg)

	case initPreviewMsg:
		return m.handleInitPreview(msg)

	case stackResourcesMsg:
		return m.handleStackResources(msg)

	case previewEventMsg:
		return m.handlePreviewEvent(msg)

	case operationEventMsg:
		return m.handleOperationEvent(msg)

	case stacksListMsg:
		return m.handleStacksList(msg)

	case stackSelectedMsg:
		return m.handleStackSelected(msg)

	case workspacesListMsg:
		return m.handleWorkspacesList(msg)

	case workspaceSelectedMsg:
		return m.handleWorkspaceSelected(msg)

	case importResultMsg:
		return m.handleImportResult(msg)

	case stateDeleteResultMsg:
		return m.handleStateDeleteResult(msg)

	case stackHistoryMsg:
		return m.handleStackHistory(msg)

	case workspaceCheckMsg:
		return m.handleWorkspaceCheck(msg)

	case pluginInitDoneMsg:
		return m.handlePluginInitDone(msg)

	case pluginAuthResultMsg:
		return m.handlePluginAuthResult(msg)

	case pluginAuthErrorMsg:
		return m.handlePluginAuthError(msg)

	case importSuggestionsMsg:
		return m.handleImportSuggestions(msg)

	case importSuggestionsErrMsg:
		// Just stop loading, no suggestions available
		m.importModal.SetSuggestions(nil)
		return m, nil

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

// handleWindowSize handles terminal resize events
func (m Model) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height
	m.header.SetWidth(msg.Width)
	m.help.SetSize(msg.Width, msg.Height)
	m.stackSelector.SetSize(msg.Width, msg.Height)
	m.workspaceSelector.SetSize(msg.Width, msg.Height)
	m.importModal.SetSize(msg.Width, msg.Height)
	m.confirmModal.SetSize(msg.Width, msg.Height)
	m.errorModal.SetSize(msg.Width, msg.Height)
	// Calculate resource list area height
	headerHeight := lipgloss.Height(m.header.View())
	footerHeight := 1 // single line footer
	listHeight := msg.Height - headerHeight - footerHeight - 1
	if listHeight < 1 {
		listHeight = 1
	}
	m.resourceList.SetSize(msg.Width, listHeight)
	// Details panel will be sized when rendered as overlay
	detailsWidth := msg.Width / 2
	m.details.SetSize(detailsWidth, listHeight)
	// Set details panel position (right side of screen, below header)
	m.details.SetPosition(msg.Width-detailsWidth, headerHeight)
	return m, nil
}

// handleMouseEvent handles mouse events for text selection
func (m Model) handleMouseEvent(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// Handle mouse events for text selection in details panel
	if m.details.Visible() {
		if cmd := m.details.HandleMouseEvent(msg); cmd != nil {
			return m, cmd
		}
	}
	return m, nil
}

// handleSpinnerTick handles spinner animation ticks
func (m Model) handleSpinnerTick(msg spinner.TickMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	if m.header.IsLoading() {
		s, cmd := m.header.Spinner().Update(msg)
		m.header.SetSpinner(s)
		cmds = append(cmds, cmd)
	}
	if m.resourceList.IsLoading() {
		s, cmd := m.resourceList.Spinner().Update(msg)
		m.resourceList.SetSpinner(s)
		cmds = append(cmds, cmd)
	}
	if m.historyList.IsLoading() {
		s, cmd := m.historyList.Spinner().Update(msg)
		m.historyList.SetSpinner(s)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

// Data message handlers

func (m Model) handleProjectInfo(msg projectInfoMsg) (tea.Model, tea.Cmd) {
	m.header.SetData(&ui.HeaderData{
		ProgramName: msg.ProgramName,
		StackName:   msg.StackName,
		Runtime:     msg.Runtime,
	})
	return m, nil
}

func (m Model) handleError(msg errMsg) (tea.Model, tea.Cmd) {
	m.header.SetError(msg)
	m.resourceList.SetError(msg)
	m.err = msg
	return m, nil
}

func (m Model) handleInitPreview(msg initPreviewMsg) (tea.Model, tea.Cmd) {
	// Store the channel and start listening for events
	m.previewCh = msg.ch
	m.resourceList.SetLoading(true, fmt.Sprintf("Running %s preview...", msg.op.String()))
	return m, waitForPreviewEvent(m.previewCh)
}

func (m Model) handleStackResources(msg stackResourcesMsg) (tea.Model, tea.Cmd) {
	// Convert to ResourceItems
	items := make([]ui.ResourceItem, 0, len(msg))
	for _, r := range msg {
		items = append(items, ui.ResourceItem{
			URN:     r.URN,
			Type:    r.Type,
			Name:    r.Name,
			Op:      pulumi.OpSame, // Stack view shows existing resources
			Status:  ui.StatusNone,
			Parent:  r.Parent,
			Inputs:  r.Inputs,
			Outputs: r.Outputs,
		})
	}
	m.resourceList.SetItems(items)
	m.header.SetSummary(m.resourceList.Summary(), ui.HeaderDone)
	// Update details panel with current selection
	if m.details.Visible() {
		m.details.SetResource(m.resourceList.SelectedItem())
	}
	return m, nil
}

// Event streaming message handlers

func (m Model) handlePreviewEvent(msg previewEventMsg) (tea.Model, tea.Cmd) {
	event := pulumi.PreviewEvent(msg)
	if event.Error != nil {
		m.resourceList.SetError(event.Error)
		m.header.SetSummary(m.resourceList.Summary(), ui.HeaderError)
		return m, nil
	}
	if event.Done {
		m.resourceList.SetLoading(false, "")
		m.header.SetSummary(m.resourceList.Summary(), ui.HeaderDone)
		return m, nil
	}
	if event.Step != nil {
		// New state inputs/outputs (for create/update)
		inputs := event.Step.Inputs
		outputs := event.Step.Outputs
		// Old state (for updates/deletes) - used for diff view
		var oldInputs, oldOutputs map[string]interface{}
		if event.Step.Old != nil {
			oldInputs = event.Step.Old.Inputs
			oldOutputs = event.Step.Old.Outputs
			// For delete ops, use old as current since new doesn't exist
			if inputs == nil {
				inputs = event.Step.Old.Inputs
			}
			if outputs == nil {
				outputs = event.Step.Old.Outputs
			}
		}
		m.resourceList.AddItem(ui.ResourceItem{
			URN:        event.Step.URN,
			Type:       event.Step.Type,
			Name:       event.Step.Name,
			Op:         event.Step.Op,
			Status:     ui.StatusNone,
			Parent:     event.Step.Parent,
			Inputs:     inputs,
			Outputs:    outputs,
			OldInputs:  oldInputs,
			OldOutputs: oldOutputs,
		})
		m.header.SetSummary(m.resourceList.Summary(), ui.HeaderRunning)
		// Update details panel with current selection
		if m.details.Visible() {
			m.details.SetResource(m.resourceList.SelectedItem())
		}
	}
	// Continue waiting for more events
	return m, waitForPreviewEvent(m.previewCh)
}

func (m Model) handleOperationEvent(msg operationEventMsg) (tea.Model, tea.Cmd) {
	event := pulumi.OperationEvent(msg)
	if event.Error != nil {
		m.resourceList.SetError(event.Error)
		m.header.SetSummary(m.resourceList.Summary(), ui.HeaderError)
		m.operationCancel = nil // Clear so escape can navigate back
		return m, nil
	}
	if event.Done {
		m.resourceList.SetLoading(false, "")
		m.header.SetSummary(m.resourceList.Summary(), ui.HeaderDone)
		m.operationCancel = nil // Clear so escape can navigate back
		return m, nil
	}
	// Add items as events stream in
	if event.URN != "" {
		var status ui.ItemStatus
		switch event.Status {
		case pulumi.StepPending:
			status = ui.StatusPending
		case pulumi.StepRunning:
			status = ui.StatusRunning
		case pulumi.StepSuccess:
			status = ui.StatusSuccess
		case pulumi.StepFailed:
			status = ui.StatusFailed
		}
		// Add or update the item - AddItem handles both cases
		m.resourceList.AddItem(ui.ResourceItem{
			URN:        event.URN,
			Type:       event.Type,
			Name:       event.Name,
			Op:         event.Op,
			Parent:     event.Parent,
			Status:     status,
			Inputs:     event.Inputs,
			Outputs:    event.Outputs,
			OldInputs:  event.OldInputs,
			OldOutputs: event.OldOutputs,
		})
		m.header.SetSummary(m.resourceList.Summary(), ui.HeaderRunning)
		// Update details panel with current selection
		if m.details.Visible() {
			m.details.SetResource(m.resourceList.SelectedItem())
		}
	}
	// Continue waiting for more events
	return m, waitForOperationEvent(m.operationCh)
}

// Stack/workspace selection handlers

func (m Model) handleStacksList(msg stacksListMsg) (tea.Model, tea.Cmd) {
	// Convert to UI stack items
	items := make([]ui.StackItem, 0, len(msg))
	var hasCurrentStack bool
	for _, s := range msg {
		items = append(items, ui.StackItem{
			Name:    s.Name,
			Current: s.Current,
		})
		if s.Current {
			hasCurrentStack = true
		}
	}
	m.stackSelector.SetStacks(items)

	// If stack selector is not visible (initial load), check if we need to show it
	if !m.stackSelector.Visible() {
		if !hasCurrentStack && len(items) > 0 {
			// No current stack, show selector
			m.stackSelector.Show()
			return m, nil
		} else if hasCurrentStack || len(items) == 0 {
			// Has current stack or no stacks at all, proceed with normal load
			cmds := []tea.Cmd{fetchProjectInfo}
			if m.viewMode == ui.ViewPreview {
				cmds = append(cmds, m.initPreview(m.operation))
			} else {
				cmds = append(cmds, m.initLoadStackResources())
			}
			return m, tea.Batch(cmds...)
		}
	}
	return m, nil
}

func (m Model) handleStackSelected(msg stackSelectedMsg) (tea.Model, tea.Cmd) {
	// Stack was selected, update the global and reload everything
	stackName = string(msg)
	m.details.Hide() // Close details panel when stack changes
	m.resourceList.Clear()
	// Invalidate credentials based on plugin refresh triggers
	if m.pluginManager != nil {
		// Use the merged config for checking refresh triggers
		mergedConfig := m.pluginManager.GetMergedConfig()
		m.pluginManager.InvalidateCredentialsForContext(workDir, stackName, "", mergedConfig)
	}
	cmds := []tea.Cmd{fetchProjectInfo, m.authenticatePlugins()}
	if m.viewMode == ui.ViewPreview {
		cmds = append(cmds, m.initPreview(m.operation))
	} else {
		cmds = append(cmds, m.loadStackResources())
	}
	return m, tea.Batch(cmds...)
}

func (m Model) handleWorkspacesList(msg workspacesListMsg) (tea.Model, tea.Cmd) {
	// Convert to UI workspace items
	items := make([]ui.WorkspaceItem, 0, len(msg))
	cwd, _ := os.Getwd()
	for _, w := range msg {
		// Compute relative path from current working directory
		relPath := w.Path
		if cwd != "" {
			if rel, err := filepath.Rel(cwd, w.Path); err == nil {
				relPath = rel
			}
		}
		items = append(items, ui.WorkspaceItem{
			Path:         w.Path,
			RelativePath: relPath,
			Name:         w.Name,
			Current:      w.Current,
		})
	}
	m.workspaceSelector.SetWorkspaces(items)
	return m, nil
}

func (m Model) handleWorkspaceSelected(msg workspaceSelectedMsg) (tea.Model, tea.Cmd) {
	// Workspace was selected, update the global workDir and reload everything
	workDir = string(msg)
	stackName = ""   // Reset stack selection for new workspace
	m.details.Hide() // Close details panel when workspace changes
	m.resourceList.Clear()
	// Invalidate credentials based on plugin refresh triggers
	if m.pluginManager != nil {
		// Use the merged config for checking refresh triggers
		mergedConfig := m.pluginManager.GetMergedConfig()
		m.pluginManager.InvalidateCredentialsForContext(workDir, stackName, "", mergedConfig)
	}
	// Fetch stacks for the new workspace (auth will happen after stack selection)
	return m, tea.Batch(fetchProjectInfo, fetchStacksList)
}

// Import suggestion handlers

func (m Model) handleImportSuggestions(msg importSuggestionsMsg) (tea.Model, tea.Cmd) {
	suggestions := make([]ui.ImportSuggestion, 0, len(msg))
	for _, s := range msg {
		suggestions = append(suggestions, ui.ImportSuggestion{
			ID:          s.Suggestion.Id,
			Label:       s.Suggestion.Label,
			Description: s.Suggestion.Description,
			PluginName:  s.PluginName,
		})
	}
	m.importModal.SetSuggestions(suggestions)
	return m, nil
}

// Operation result handlers

func (m Model) handleImportResult(msg importResultMsg) (tea.Model, tea.Cmd) {
	if msg == nil {
		m.errorModal.Show(
			"Import Failed",
			"Unknown error occurred during import",
			"No additional details available",
		)
		return m, nil
	}
	if msg.Success {
		// Import succeeded, show success and refresh the preview
		cmds := []tea.Cmd{
			m.toast.Show(fmt.Sprintf("Imported %s successfully", m.importModal.GetResourceName())),
			m.startPreview(m.operation), // Re-run preview to show updated state
		}
		return m, tea.Batch(cmds...)
	}
	// Import failed, show error modal with full details
	summary := fmt.Sprintf("Failed to import '%s' (%s)",
		m.importModal.GetResourceName(),
		m.importModal.GetResourceType())
	details := msg.Output
	if details == "" && msg.Error != nil {
		details = msg.Error.Error()
	}
	m.errorModal.Show("Import Failed", summary, details)
	return m, nil
}

func (m Model) handleStateDeleteResult(msg stateDeleteResultMsg) (tea.Model, tea.Cmd) {
	if msg == nil {
		return m, m.toast.Show("Delete from state failed: unknown error")
	}
	resourceName := m.confirmModal.GetContextName()
	m.confirmModal.Hide()
	if msg.Success {
		// State delete succeeded, show success and refresh the stack view
		cmds := []tea.Cmd{
			m.toast.Show(fmt.Sprintf("Removed '%s' from state", resourceName)),
			m.loadStackResources(), // Reload to show updated state
		}
		return m, tea.Batch(cmds...)
	}
	// State delete failed, show error
	errMsg := "Delete from state failed"
	if msg.Error != nil {
		errMsg = msg.Error.Error()
	}
	return m, m.toast.Show(errMsg)
}

func (m Model) handleStackHistory(msg stackHistoryMsg) (tea.Model, tea.Cmd) {
	// Convert to UI history items
	items := make([]ui.HistoryItem, 0, len(msg))
	for i, h := range msg {
		version := h.Version
		// Pulumi local backend doesn't track version numbers, so use index
		// History is returned newest-first, so index 0 = most recent
		if version == 0 {
			version = len(msg) - i
		}
		items = append(items, ui.HistoryItem{
			Version:         version,
			Kind:            h.Kind,
			StartTime:       h.StartTime,
			EndTime:         h.EndTime,
			Message:         h.Message,
			Result:          h.Result,
			ResourceChanges: h.ResourceChanges,
			User:            h.User,
			UserEmail:       h.UserEmail,
		})
	}
	m.historyList.SetItems(items)
	m.header.SetSummary(ui.ResourceSummary{Total: len(items)}, ui.HeaderDone)
	return m, nil
}

// Initialization handlers

func (m Model) handleWorkspaceCheck(msg workspaceCheckMsg) (tea.Model, tea.Cmd) {
	if msg {
		// We're in a valid workspace, continue with normal initialization
		return m, m.continueInit()
	}
	// Not in a workspace, show the workspace selector
	m.workspaceSelector.SetLoading(true)
	m.workspaceSelector.Show()
	return m, fetchWorkspacesList
}

func (m Model) handlePluginInitDone(msg pluginInitDoneMsg) (tea.Model, tea.Cmd) {
	// Initial plugin authentication completed, continue with normal init
	var cmds []tea.Cmd

	// Apply plugin env vars to the process environment so Pulumi operations inherit them
	if m.pluginManager != nil {
		m.pluginManager.ApplyEnvToProcess()
	}

	cmds = append(cmds, m.continueInitAfterPlugins())

	// Show toast for plugin results
	if msg.err != nil {
		cmds = append(cmds, m.toast.Show(fmt.Sprintf("Plugin error: %v", msg.err)))
	} else if len(msg.results) > 0 {
		var pluginNames []string
		for _, r := range msg.results {
			if r.Credentials != nil && len(r.Credentials.Env) > 0 {
				pluginNames = append(pluginNames, r.PluginName)
			}
		}
		if len(pluginNames) > 0 {
			cmds = append(cmds, m.toast.Show(fmt.Sprintf("Authenticated: %s", strings.Join(pluginNames, ", "))))
		}
	}
	return m, tea.Batch(cmds...)
}

func (m Model) handlePluginAuthResult(msg pluginAuthResultMsg) (tea.Model, tea.Cmd) {
	// Plugin authentication completed (for re-auth after stack/workspace change)
	// Apply env vars to process environment
	if m.pluginManager != nil {
		m.pluginManager.ApplyEnvToProcess()
	}

	var hasErrors bool
	var errorMsgs []string
	for _, result := range msg {
		if result.Error != nil {
			hasErrors = true
			errorMsgs = append(errorMsgs, fmt.Sprintf("%s: %v", result.PluginName, result.Error))
		}
	}
	if hasErrors {
		// Show error toast but don't block - credentials are optional
		return m, m.toast.Show(fmt.Sprintf("Plugin auth failed: %s", strings.Join(errorMsgs, "; ")))
	}
	// Show success if we have plugins that authenticated
	if len(msg) > 0 {
		var pluginNames []string
		for _, r := range msg {
			if r.Credentials != nil && len(r.Credentials.Env) > 0 {
				pluginNames = append(pluginNames, r.PluginName)
			}
		}
		if len(pluginNames) > 0 {
			return m, m.toast.Show(fmt.Sprintf("Authenticated: %s", strings.Join(pluginNames, ", ")))
		}
	}
	return m, nil
}

func (m Model) handlePluginAuthError(msg pluginAuthErrorMsg) (tea.Model, tea.Cmd) {
	// Plugin system error - show but don't block
	return m, m.toast.Show(fmt.Sprintf("Plugin error: %v", error(msg)))
}

// UI message handlers

func (m Model) handleCopiedToClipboard(msg ui.CopiedToClipboardMsg) (tea.Model, tea.Cmd) {
	var toastMsg string
	var cmds []tea.Cmd

	if msg.Count == 1 {
		// Single resource - show name and flash
		item := m.resourceList.SelectedItem()
		if item != nil {
			toastMsg = fmt.Sprintf("Copied %s", item.Name)
		} else {
			toastMsg = "Copied resource"
		}
	} else if msg.Count > 1 {
		// Multiple resources - show count
		toastMsg = fmt.Sprintf("Copied %d resources", msg.Count)
	} else {
		// Text copy (from details panel)
		toastMsg = "Copied to clipboard"
	}

	// Flash clear after short duration (for both single and all)
	if msg.Count >= 1 {
		cmds = append(cmds, tea.Tick(ui.FlashDuration, func(time.Time) tea.Msg {
			return ui.FlashClearMsg{}
		}))
	}

	cmds = append(cmds, m.toast.Show(toastMsg))
	return m, tea.Batch(cmds...)
}

func (m Model) handleToastHide() (tea.Model, tea.Cmd) {
	m.toast.Hide()
	return m, nil
}

func (m Model) handleFlashClear() (tea.Model, tea.Cmd) {
	m.resourceList.ClearFlash()
	return m, nil
}
