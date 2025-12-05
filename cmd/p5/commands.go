package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/rfhold/p5/internal/plugins"
	"github.com/rfhold/p5/internal/pulumi"
	"github.com/rfhold/p5/internal/ui"
)

// checkWorkspace verifies if the current working directory is a Pulumi workspace
func checkWorkspace() tea.Msg {
	return workspaceCheckMsg(pulumi.IsWorkspace(workDir))
}

// continueInit continues initialization after workspace check passes
// First authenticates plugins, then proceeds with normal init
func (m Model) continueInit() tea.Cmd {
	// Authenticate plugins first - they may set env vars needed for Pulumi operations
	return m.authenticatePluginsForInit()
}

// authenticatePluginsForInit authenticates plugins during initialization
// This runs before any Pulumi operations to ensure env vars are set
func (m *Model) authenticatePluginsForInit() tea.Cmd {
	if m.pluginManager == nil {
		// No plugin manager, continue with normal init
		return m.continueInitAfterPlugins()
	}

	return func() tea.Msg {
		// Load and authenticate plugins with minimal context
		// We don't have stack name yet, but plugins can still load from p5.toml
		results, err := m.pluginManager.LoadAndAuthenticate(
			context.Background(),
			workDir,
			"", // program name not known yet
			"", // stack name not known yet
		)
		if err != nil {
			// Plugin errors are non-fatal, continue anyway
			return pluginInitDoneMsg{results: nil, err: err}
		}
		return pluginInitDoneMsg{results: results, err: nil}
	}
}

// continueInitAfterPlugins continues initialization after plugins are authenticated
func (m Model) continueInitAfterPlugins() tea.Cmd {
	// If no stack name specified, we need to check if there's a current stack
	// or show the stack selector
	if stackName == "" {
		// Fetch stacks first to determine if we need to show selector
		return fetchStacksList
	}
	// Stack specified, proceed normally
	cmds := []tea.Cmd{fetchProjectInfo}
	if m.viewMode == ui.ViewPreview {
		cmds = append(cmds, m.initPreview(m.operation))
	} else {
		cmds = append(cmds, m.initLoadStackResources())
	}
	return tea.Batch(cmds...)
}

// initLoadStackResources returns a command to load stack resources (for use in Init)
func (m Model) initLoadStackResources() tea.Cmd {
	return func() tea.Msg {
		resources, err := pulumi.GetStackResources(context.Background(), workDir, stackName)
		if err != nil {
			return errMsg(err)
		}
		return stackResourcesMsg(resources)
	}
}

// initPreview returns a command to start a preview (for use in Init)
func (m Model) initPreview(op pulumi.OperationType) tea.Cmd {
	ch := make(chan pulumi.PreviewEvent)

	// Build options from flags
	opts := pulumi.OperationOptions{
		Targets:  m.resourceList.GetTargetURNs(),
		Replaces: m.resourceList.GetReplaceURNs(),
	}

	// Start preview in background
	go func() {
		switch op {
		case pulumi.OperationUp:
			pulumi.RunUpPreview(context.Background(), workDir, stackName, opts, ch)
		case pulumi.OperationRefresh:
			pulumi.RunRefreshPreview(context.Background(), workDir, stackName, opts, ch)
		case pulumi.OperationDestroy:
			pulumi.RunDestroyPreview(context.Background(), workDir, stackName, opts, ch)
		}
	}()

	return func() tea.Msg {
		return initPreviewMsg{op: op, ch: ch}
	}
}

// loadStackResources fetches stack resources
func (m *Model) loadStackResources() tea.Cmd {
	m.resourceList.SetLoading(true, "Loading stack resources...")
	m.resourceList.SetShowAllOps(true)
	return func() tea.Msg {
		resources, err := pulumi.GetStackResources(context.Background(), workDir, stackName)
		if err != nil {
			return errMsg(err)
		}
		return stackResourcesMsg(resources)
	}
}

// startPreview starts a preview operation
func (m *Model) startPreview(op pulumi.OperationType) tea.Cmd {
	m.viewMode = ui.ViewPreview
	m.operation = op
	m.header.SetViewMode(m.viewMode)
	m.header.SetOperation(m.operation)
	m.details.Hide() // Close details panel when view changes
	m.resourceList.Clear()
	m.resourceList.SetShowAllOps(false) // Hide unchanged resources
	m.resourceList.SetLoading(true, fmt.Sprintf("Running %s preview...", op.String()))

	// Build options from flags
	opts := pulumi.OperationOptions{
		Targets:  m.resourceList.GetTargetURNs(),
		Replaces: m.resourceList.GetReplaceURNs(),
	}

	// Add plugin credentials as env vars
	if m.pluginManager != nil {
		opts.Env = m.pluginManager.GetAllEnv()
	}

	m.previewCh = make(chan pulumi.PreviewEvent)

	// Start preview in background
	go func() {
		switch op {
		case pulumi.OperationUp:
			pulumi.RunUpPreview(context.Background(), workDir, stackName, opts, m.previewCh)
		case pulumi.OperationRefresh:
			pulumi.RunRefreshPreview(context.Background(), workDir, stackName, opts, m.previewCh)
		case pulumi.OperationDestroy:
			pulumi.RunDestroyPreview(context.Background(), workDir, stackName, opts, m.previewCh)
		}
	}()

	return waitForPreviewEvent(m.previewCh)
}

// maybeConfirmExecution checks if confirmation is needed before executing
// Confirmation is needed if the user is not on the preview screen for the requested operation
func (m *Model) maybeConfirmExecution(op pulumi.OperationType) tea.Cmd {
	// If we're on the preview screen for this exact operation, execute directly
	if m.viewMode == ui.ViewPreview && m.operation == op {
		return m.startExecution(op)
	}

	// Otherwise, show confirmation modal
	m.pendingOperation = &op
	m.confirmModal.SetLabels("Cancel", "Execute")
	m.confirmModal.SetKeys("n", "y")
	m.confirmModal.Show(
		fmt.Sprintf("Execute %s", op.String()),
		fmt.Sprintf("Run %s without previewing changes first?", op.String()),
		"This will apply changes to your infrastructure.",
	)
	return nil
}

// startExecution starts an execution operation
func (m *Model) startExecution(op pulumi.OperationType) tea.Cmd {
	m.viewMode = ui.ViewExecute
	m.operation = op
	m.header.SetViewMode(m.viewMode)
	m.header.SetOperation(m.operation)
	m.details.Hide() // Close details panel when view changes

	// Clear the list and show events as they stream in
	m.resourceList.Clear()
	m.resourceList.SetShowAllOps(false)
	m.resourceList.SetLoading(true, fmt.Sprintf("Executing %s...", op.String()))

	// Build options from flags
	opts := pulumi.OperationOptions{
		Targets:  m.resourceList.GetTargetURNs(),
		Replaces: m.resourceList.GetReplaceURNs(),
	}

	// Add plugin credentials as env vars
	if m.pluginManager != nil {
		opts.Env = m.pluginManager.GetAllEnv()
	}

	// Create cancellable context
	m.operationCtx, m.operationCancel = context.WithCancel(context.Background())
	m.operationCh = make(chan pulumi.OperationEvent)

	// Start execution in background
	go func() {
		switch op {
		case pulumi.OperationUp:
			pulumi.RunUp(m.operationCtx, workDir, stackName, opts, m.operationCh)
		case pulumi.OperationRefresh:
			pulumi.RunRefresh(m.operationCtx, workDir, stackName, opts, m.operationCh)
		case pulumi.OperationDestroy:
			pulumi.RunDestroy(m.operationCtx, workDir, stackName, opts, m.operationCh)
		}
	}()

	return waitForOperationEvent(m.operationCh)
}

// switchToStackView switches back to stack view
func (m *Model) switchToStackView() tea.Cmd {
	m.viewMode = ui.ViewStack
	m.header.SetViewMode(m.viewMode)
	m.details.Hide() // Close details panel when view changes
	m.resourceList.Clear()
	m.resourceList.SetShowAllOps(true)
	return m.loadStackResources()
}

// switchToHistoryView switches to history view
func (m *Model) switchToHistoryView() tea.Cmd {
	m.viewMode = ui.ViewHistory
	m.header.SetViewMode(m.viewMode)
	m.details.Hide() // Close resource details panel when switching views
	m.historyList.Clear()
	m.historyList.SetLoading(true, "Loading stack history...")
	return fetchStackHistory
}

// executeStateDelete runs the pulumi state delete command
func (m *Model) executeStateDelete() tea.Cmd {
	urn := m.confirmModal.GetContextURN()

	// Build options with plugin env vars
	opts := pulumi.StateDeleteOptions{}
	if m.pluginManager != nil {
		opts.Env = m.pluginManager.GetAllEnv()
	}

	return func() tea.Msg {
		result, err := pulumi.DeleteFromState(
			context.Background(),
			workDir,
			stackName,
			urn,
			opts,
		)
		if err != nil {
			return stateDeleteResultMsg(&pulumi.StateDeleteResult{
				Success: false,
				Error:   err,
			})
		}
		return stateDeleteResultMsg(result)
	}
}

// executeImport runs the pulumi import command
func (m *Model) executeImport() tea.Cmd {
	resourceType := m.importModal.GetResourceType()
	resourceName := m.importModal.GetResourceName()
	importID := m.importModal.GetImportID()
	parentURN := m.importModal.GetParentURN()

	// Build import options with plugin env vars
	opts := pulumi.ImportOptions{}
	if m.pluginManager != nil {
		opts.Env = m.pluginManager.GetAllEnv()
	}

	return func() tea.Msg {
		result, err := pulumi.ImportResource(
			context.Background(),
			workDir,
			stackName,
			resourceType,
			resourceName,
			importID,
			parentURN,
			opts,
		)
		if err != nil {
			return importResultMsg(&pulumi.ImportResult{
				Success: false,
				Error:   err,
			})
		}
		return importResultMsg(result)
	}
}

// fetchStackHistory loads the stack history
func fetchStackHistory() tea.Msg {
	history, err := pulumi.GetStackHistory(context.Background(), workDir, stackName, 50, 1)
	if err != nil {
		return errMsg(err)
	}
	return stackHistoryMsg(history)
}

// fetchImportSuggestions queries plugins for import suggestions
func (m *Model) fetchImportSuggestions(resourceType, resourceName, resourceURN, parentURN string, inputs map[string]interface{}) tea.Cmd {
	if m.pluginManager == nil {
		return func() tea.Msg {
			return importSuggestionsMsg(nil)
		}
	}

	// Convert inputs to string map for proto
	inputStrings := make(map[string]string)
	for k, v := range inputs {
		switch val := v.(type) {
		case string:
			inputStrings[k] = val
		default:
			// For non-string values, JSON serialize them
			if b, err := json.Marshal(val); err == nil {
				inputStrings[k] = string(b)
			}
		}
	}

	return func() tea.Msg {
		req := &plugins.ImportSuggestionsRequest{
			ResourceType: resourceType,
			ResourceName: resourceName,
			ResourceUrn:  resourceURN,
			ParentUrn:    parentURN,
			Inputs:       inputStrings,
		}

		suggestions, err := m.pluginManager.GetImportSuggestions(context.Background(), req)
		if err != nil {
			return importSuggestionsErrMsg(err)
		}
		return importSuggestionsMsg(suggestions)
	}
}

// authenticatePlugins triggers plugin authentication for the current workspace/stack
func (m *Model) authenticatePlugins() tea.Cmd {
	if m.pluginManager == nil {
		return nil
	}

	return func() tea.Msg {
		// Get project info for the program name
		info, err := pulumi.FetchProjectInfo(context.Background(), workDir, stackName)
		if err != nil {
			return pluginAuthErrorMsg(err)
		}

		results, err := m.pluginManager.LoadAndAuthenticate(
			context.Background(),
			workDir,
			info.ProgramName,
			info.StackName,
		)
		if err != nil {
			return pluginAuthErrorMsg(err)
		}

		return pluginAuthResultMsg(results)
	}
}

// waitForPreviewEvent waits for the next preview event
func waitForPreviewEvent(ch chan pulumi.PreviewEvent) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-ch
		if !ok {
			return previewEventMsg{Done: true}
		}
		return previewEventMsg(event)
	}
}

// waitForOperationEvent waits for the next operation event
func waitForOperationEvent(ch chan pulumi.OperationEvent) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-ch
		if !ok {
			return operationEventMsg{Done: true}
		}
		return operationEventMsg(event)
	}
}

// fetchProjectInfo loads project info using Pulumi Automation API
func fetchProjectInfo() tea.Msg {
	info, err := pulumi.FetchProjectInfo(context.Background(), workDir, stackName)
	if err != nil {
		return errMsg(err)
	}
	return projectInfoMsg(info)
}

// fetchStacksList loads the list of available stacks
func fetchStacksList() tea.Msg {
	stacks, err := pulumi.ListStacks(context.Background(), workDir)
	if err != nil {
		return errMsg(err)
	}
	return stacksListMsg(stacks)
}

// selectStack sets the current stack and reloads data
func selectStack(name string) tea.Cmd {
	return func() tea.Msg {
		err := pulumi.SelectStack(context.Background(), workDir, name)
		if err != nil {
			return errMsg(err)
		}
		return stackSelectedMsg(name)
	}
}

// fetchWorkspacesList searches for Pulumi workspaces in the current directory tree
func fetchWorkspacesList() tea.Msg {
	// Search from current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return errMsg(err)
	}
	workspaces, err := pulumi.FindWorkspaces(cwd, workDir)
	if err != nil {
		return errMsg(err)
	}
	return workspacesListMsg(workspaces)
}

// selectWorkspace changes the current workspace directory
func selectWorkspace(path string) tea.Cmd {
	return func() tea.Msg {
		return workspaceSelectedMsg(path)
	}
}
