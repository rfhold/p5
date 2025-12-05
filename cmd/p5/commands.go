package main

import (
	"context"
	"encoding/json"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/rfhold/p5/internal/plugins"
	"github.com/rfhold/p5/internal/pulumi"
	"github.com/rfhold/p5/internal/ui"
)

// checkWorkspace returns a command to verify if the working directory is a Pulumi workspace
func (m *Model) checkWorkspace() tea.Cmd {
	workDir := m.ctx.WorkDir
	workspaceReader := m.deps.WorkspaceReader
	return func() tea.Msg {
		return workspaceCheckMsg(workspaceReader.IsWorkspace(workDir))
	}
}

// authenticatePluginsForInit authenticates plugins during initialization
// This runs before any Pulumi operations to ensure env vars are set.
// Returns pluginInitDoneMsg which is handled by the init state machine.
func (m *Model) authenticatePluginsForInit() tea.Cmd {
	if m.deps == nil || m.deps.PluginProvider == nil {
		// No plugin provider, return empty result to continue init flow
		return func() tea.Msg {
			return pluginInitDoneMsg{results: nil, err: nil}
		}
	}

	workDir := m.ctx.WorkDir
	pluginProvider := m.deps.PluginProvider
	return func() tea.Msg {
		// Load and authenticate plugins with minimal context
		// We don't have stack name yet, but plugins can still load from p5.toml
		results, err := pluginProvider.Initialize(
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

// authenticatePluginsForWorkspace authenticates plugins after a workspace is selected
// This reuses the same pluginInitDoneMsg flow as initial authentication, ensuring
// that env vars are set before any Pulumi operations (like fetching stacks)
func (m *Model) authenticatePluginsForWorkspace() tea.Cmd {
	// Reuse the same authentication flow - it will:
	// 1. Load p5.toml from the new workDir
	// 2. Authenticate plugins
	// 3. Return pluginInitDoneMsg
	// 4. handlePluginInitDone in the state machine handles the rest
	return m.authenticatePluginsForInit()
}

// initLoadStackResources returns a command to load stack resources (for use in Init)
func (m Model) initLoadStackResources() tea.Cmd {
	workDir := m.ctx.WorkDir
	stackName := m.ctx.StackName
	stackReader := m.deps.StackReader
	return func() tea.Msg {
		resources, err := stackReader.GetResources(context.Background(), workDir, stackName)
		if err != nil {
			return errMsg(err)
		}
		return stackResourcesMsg(resources)
	}
}

// initPreview returns a command to start a preview (for use in Init)
func (m Model) initPreview(op pulumi.OperationType) tea.Cmd {
	// Build options from flags
	opts := pulumi.OperationOptions{
		Targets:  m.ui.ResourceList.GetTargetURNs(),
		Replaces: m.ui.ResourceList.GetReplaceURNs(),
	}

	workDir := m.ctx.WorkDir
	stackName := m.ctx.StackName
	stackOperator := m.deps.StackOperator

	// Use injected StackOperator - it owns the channel and returns receive-only
	ch := stackOperator.Preview(context.Background(), workDir, stackName, op, opts)

	return func() tea.Msg {
		return initPreviewMsg{op: op, ch: ch}
	}
}

// loadStackResources fetches stack resources
func (m *Model) loadStackResources() tea.Cmd {
	m.ui.ResourceList.SetLoading(true, "Loading stack resources...")
	m.ui.ResourceList.SetShowAllOps(true)
	workDir := m.ctx.WorkDir
	stackName := m.ctx.StackName
	stackReader := m.deps.StackReader
	return func() tea.Msg {
		resources, err := stackReader.GetResources(context.Background(), workDir, stackName)
		if err != nil {
			return errMsg(err)
		}
		return stackResourcesMsg(resources)
	}
}

// startPreview starts a preview operation
func (m *Model) startPreview(op pulumi.OperationType) tea.Cmd {
	// Transition operation state
	m.transitionOpTo(OpStarting)

	m.ui.ViewMode = ui.ViewPreview
	m.state.Operation = op
	m.ui.Header.SetViewMode(m.ui.ViewMode)
	m.ui.Header.SetOperation(m.state.Operation)
	m.ui.Details.Hide() // Close details panel when view changes
	m.ui.ResourceList.Clear()
	m.ui.ResourceList.SetShowAllOps(false) // Hide unchanged resources
	m.ui.ResourceList.SetLoading(true, fmt.Sprintf("Running %s preview...", op.String()))

	// Build options from flags
	opts := pulumi.OperationOptions{
		Targets:  m.ui.ResourceList.GetTargetURNs(),
		Replaces: m.ui.ResourceList.GetReplaceURNs(),
	}

	// Add plugin credentials as env vars
	if m.deps != nil && m.deps.PluginProvider != nil {
		opts.Env = m.deps.PluginProvider.GetAllEnv()
	}

	workDir := m.ctx.WorkDir
	stackName := m.ctx.StackName

	// Use injected StackOperator - it owns the channel and returns receive-only
	m.previewCh = m.deps.StackOperator.Preview(context.Background(), workDir, stackName, op, opts)

	return waitForPreviewEvent(m.previewCh)
}

// maybeConfirmExecution checks if confirmation is needed before executing
// Confirmation is needed if the user is not on the preview screen for the requested operation
func (m *Model) maybeConfirmExecution(op pulumi.OperationType) tea.Cmd {
	// If we're on the preview screen for this exact operation, execute directly
	if m.ui.ViewMode == ui.ViewPreview && m.state.Operation == op {
		return m.startExecution(op)
	}

	// Otherwise, show confirmation modal
	m.state.PendingOperation = &op
	m.ui.ConfirmModal.SetLabels("Cancel", "Execute")
	m.ui.ConfirmModal.SetKeys("n", "y")
	m.ui.ConfirmModal.Show(
		fmt.Sprintf("Execute %s", op.String()),
		fmt.Sprintf("Run %s without previewing changes first?", op.String()),
		"This will apply changes to your infrastructure.",
	)
	m.showConfirmModal()
	return nil
}

// startExecution starts an execution operation
func (m *Model) startExecution(op pulumi.OperationType) tea.Cmd {
	// Transition operation state
	m.transitionOpTo(OpStarting)

	m.ui.ViewMode = ui.ViewExecute
	m.state.Operation = op
	m.ui.Header.SetViewMode(m.ui.ViewMode)
	m.ui.Header.SetOperation(m.state.Operation)
	m.ui.Details.Hide() // Close details panel when view changes

	// Clear the list and show events as they stream in
	m.ui.ResourceList.Clear()
	m.ui.ResourceList.SetShowAllOps(false)
	m.ui.ResourceList.SetLoading(true, fmt.Sprintf("Executing %s...", op.String()))

	// Build options from flags
	opts := pulumi.OperationOptions{
		Targets:  m.ui.ResourceList.GetTargetURNs(),
		Replaces: m.ui.ResourceList.GetReplaceURNs(),
	}

	// Add plugin credentials as env vars
	if m.deps != nil && m.deps.PluginProvider != nil {
		opts.Env = m.deps.PluginProvider.GetAllEnv()
	}

	// Create cancellable context
	m.operationCtx, m.operationCancel = context.WithCancel(context.Background())

	workDir := m.ctx.WorkDir
	stackName := m.ctx.StackName
	stackOperator := m.deps.StackOperator

	// Use injected StackOperator - it owns the channel and returns receive-only
	switch op {
	case pulumi.OperationUp:
		m.operationCh = stackOperator.Up(m.operationCtx, workDir, stackName, opts)
	case pulumi.OperationRefresh:
		m.operationCh = stackOperator.Refresh(m.operationCtx, workDir, stackName, opts)
	case pulumi.OperationDestroy:
		m.operationCh = stackOperator.Destroy(m.operationCtx, workDir, stackName, opts)
	}

	return waitForOperationEvent(m.operationCh)
}

// switchToStackView switches back to stack view
func (m *Model) switchToStackView() tea.Cmd {
	// Reset operation state when leaving preview/execute views
	m.resetOperation()

	m.ui.ViewMode = ui.ViewStack
	m.ui.Header.SetViewMode(m.ui.ViewMode)
	m.ui.Details.Hide() // Close details panel when view changes
	m.ui.ResourceList.Clear()
	m.ui.ResourceList.SetShowAllOps(true)
	return m.loadStackResources()
}

// switchToHistoryView switches to history view
func (m *Model) switchToHistoryView() tea.Cmd {
	m.ui.ViewMode = ui.ViewHistory
	m.ui.Header.SetViewMode(m.ui.ViewMode)
	m.ui.Details.Hide() // Close resource details panel when switching views
	m.ui.HistoryList.Clear()
	m.ui.HistoryList.SetLoading(true, "Loading stack history...")
	return m.fetchStackHistory()
}

// executeStateDelete runs the pulumi state delete command
func (m *Model) executeStateDelete() tea.Cmd {
	urn := m.ui.ConfirmModal.GetContextURN()

	// Build options with plugin env vars
	opts := pulumi.StateDeleteOptions{}
	if m.deps != nil && m.deps.PluginProvider != nil {
		opts.Env = m.deps.PluginProvider.GetAllEnv()
	}

	workDir := m.ctx.WorkDir
	stackName := m.ctx.StackName
	resourceImporter := m.deps.ResourceImporter

	return func() tea.Msg {
		result, err := resourceImporter.StateDelete(
			context.Background(),
			workDir,
			stackName,
			urn,
			opts,
		)
		if err != nil {
			return stateDeleteResultMsg(&pulumi.CommandResult{
				Success: false,
				Error:   err,
			})
		}
		return stateDeleteResultMsg(result)
	}
}

// executeImport runs the pulumi import command
func (m *Model) executeImport() tea.Cmd {
	resourceType := m.ui.ImportModal.GetResourceType()
	resourceName := m.ui.ImportModal.GetResourceName()
	importID := m.ui.ImportModal.GetImportID()
	parentURN := m.ui.ImportModal.GetParentURN()

	// Build import options with plugin env vars
	opts := pulumi.ImportOptions{}
	if m.deps != nil && m.deps.PluginProvider != nil {
		opts.Env = m.deps.PluginProvider.GetAllEnv()
	}

	workDir := m.ctx.WorkDir
	stackName := m.ctx.StackName
	resourceImporter := m.deps.ResourceImporter

	return func() tea.Msg {
		result, err := resourceImporter.Import(
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
			return importResultMsg(&pulumi.CommandResult{
				Success: false,
				Error:   err,
			})
		}
		return importResultMsg(result)
	}
}

// fetchStackHistory returns a command to load the stack history
func (m *Model) fetchStackHistory() tea.Cmd {
	workDir := m.ctx.WorkDir
	stackName := m.ctx.StackName
	stackReader := m.deps.StackReader
	return func() tea.Msg {
		history, err := stackReader.GetHistory(context.Background(), workDir, stackName, pulumi.DefaultHistoryPageSize, pulumi.DefaultHistoryPage)
		if err != nil {
			return errMsg(err)
		}
		return stackHistoryMsg(history)
	}
}

// fetchImportSuggestions queries plugins for import suggestions
func (m *Model) fetchImportSuggestions(resourceType, resourceName, resourceURN, parentURN string, inputs map[string]interface{}) tea.Cmd {
	if m.deps == nil || m.deps.PluginProvider == nil {
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

		suggestions, err := m.deps.PluginProvider.GetImportSuggestions(context.Background(), req)
		if err != nil {
			return importSuggestionsErrMsg(err)
		}
		return importSuggestionsMsg(suggestions)
	}
}

// authenticatePlugins triggers plugin authentication for the current workspace/stack
func (m *Model) authenticatePlugins() tea.Cmd {
	if m.deps == nil || m.deps.PluginProvider == nil {
		return nil
	}

	workDir := m.ctx.WorkDir
	stackName := m.ctx.StackName
	pluginProvider := m.deps.PluginProvider
	workspaceReader := m.deps.WorkspaceReader
	return func() tea.Msg {
		// Get project info for the program name
		info, err := workspaceReader.GetProjectInfo(context.Background(), workDir, stackName)
		if err != nil {
			return pluginAuthErrorMsg(err)
		}

		results, err := pluginProvider.Initialize(
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
func waitForPreviewEvent(ch <-chan pulumi.PreviewEvent) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-ch
		if !ok {
			return previewEventMsg{Done: true}
		}
		return previewEventMsg(event)
	}
}

// waitForOperationEvent waits for the next operation event
func waitForOperationEvent(ch <-chan pulumi.OperationEvent) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-ch
		if !ok {
			return operationEventMsg{Done: true}
		}
		return operationEventMsg(event)
	}
}

// fetchProjectInfo returns a command to load project info using Pulumi Automation API
func (m *Model) fetchProjectInfo() tea.Cmd {
	workDir := m.ctx.WorkDir
	stackName := m.ctx.StackName
	workspaceReader := m.deps.WorkspaceReader
	return func() tea.Msg {
		info, err := workspaceReader.GetProjectInfo(context.Background(), workDir, stackName)
		if err != nil {
			return errMsg(err)
		}
		return projectInfoMsg(info)
	}
}

// fetchStacksList returns a command to load the list of available stacks
func (m *Model) fetchStacksList() tea.Cmd {
	workDir := m.ctx.WorkDir
	stackReader := m.deps.StackReader
	return func() tea.Msg {
		stacks, err := stackReader.GetStacks(context.Background(), workDir)
		if err != nil {
			return errMsg(err)
		}
		return stacksListMsg(stacks)
	}
}

// selectStack sets the current stack and reloads data
func (m *Model) selectStack(name string) tea.Cmd {
	workDir := m.ctx.WorkDir
	stackReader := m.deps.StackReader
	return func() tea.Msg {
		err := stackReader.SelectStack(context.Background(), workDir, name)
		if err != nil {
			return errMsg(err)
		}
		return stackSelectedMsg(name)
	}
}

// fetchWorkspacesList returns a command to search for Pulumi workspaces in the current directory tree
func (m *Model) fetchWorkspacesList() tea.Cmd {
	cwd := m.ctx.Cwd
	workDir := m.ctx.WorkDir
	workspaceReader := m.deps.WorkspaceReader
	return func() tea.Msg {
		workspaces, err := workspaceReader.FindWorkspaces(cwd, workDir)
		if err != nil {
			return errMsg(err)
		}
		return workspacesListMsg(workspaces)
	}
}

// selectWorkspace changes the current workspace directory
func selectWorkspace(path string) tea.Cmd {
	return func() tea.Msg {
		return workspaceSelectedMsg(path)
	}
}

// fetchWhoAmI returns a command to get backend connection info
func (m *Model) fetchWhoAmI() tea.Cmd {
	workDir := m.ctx.WorkDir
	workspaceReader := m.deps.WorkspaceReader
	return func() tea.Msg {
		info, err := workspaceReader.GetWhoAmI(context.Background(), workDir)
		if err != nil {
			// Non-fatal - return empty info
			return whoAmIMsg(&pulumi.WhoAmIInfo{})
		}
		return whoAmIMsg(info)
	}
}

// fetchStackFiles returns a command to list stack config files in workspace
func (m *Model) fetchStackFiles() tea.Cmd {
	workDir := m.ctx.WorkDir
	workspaceReader := m.deps.WorkspaceReader
	return func() tea.Msg {
		files, err := workspaceReader.ListStackFiles(workDir)
		if err != nil {
			// Non-fatal - return empty list
			return stackFilesMsg(nil)
		}
		return stackFilesMsg(files)
	}
}

// initStack creates a new stack
func (m *Model) initStack(name, secretsProvider, passphrase string) tea.Cmd {
	workDir := m.ctx.WorkDir
	stackInitializer := m.deps.StackInitializer
	// Add plugin credentials as env vars
	var pluginEnv map[string]string
	if m.deps != nil && m.deps.PluginProvider != nil {
		pluginEnv = m.deps.PluginProvider.GetAllEnv()
	}
	return func() tea.Msg {
		opts := pulumi.InitStackOptions{
			SecretsProvider: secretsProvider,
			Passphrase:      passphrase,
			Env:             pluginEnv,
		}

		err := stackInitializer.InitStack(context.Background(), workDir, name, opts)
		if err != nil {
			return stackInitResultMsg{StackName: name, Error: err}
		}
		return stackInitResultMsg{StackName: name, Error: nil}
	}
}
