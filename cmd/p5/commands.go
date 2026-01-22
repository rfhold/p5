package main

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pkg/browser"

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
	appCtx := m.appCtx
	return func() tea.Msg {
		// Load and authenticate plugins with minimal context
		// We don't have stack name yet, but plugins can still load from p5.toml
		results, err := pluginProvider.Initialize(
			appCtx,
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
	appCtx := m.appCtx
	opts := pulumi.ReadOptions{Env: m.deps.Env}
	return func() tea.Msg {
		resources, err := stackReader.GetResources(appCtx, workDir, stackName, opts)
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
		Excludes: m.ui.ResourceList.GetExcludeURNs(),
	}

	// Merge base env with plugin env
	opts.Env = mergeEnvMaps(m.deps.Env, m.deps.PluginProvider.GetAllEnv())

	workDir := m.ctx.WorkDir
	stackName := m.ctx.StackName
	stackOperator := m.deps.StackOperator
	appCtx := m.appCtx

	// Use injected StackOperator - it owns the channel and returns receive-only
	ch := stackOperator.Preview(appCtx, workDir, stackName, op, opts)

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
	appCtx := m.appCtx
	opts := pulumi.ReadOptions{Env: m.deps.Env}
	return func() tea.Msg {
		resources, err := stackReader.GetResources(appCtx, workDir, stackName, opts)
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
		Excludes: m.ui.ResourceList.GetExcludeURNs(),
	}

	// Merge base env with plugin credentials
	opts.Env = mergeEnvMaps(m.deps.Env, m.deps.PluginProvider.GetAllEnv())

	workDir := m.ctx.WorkDir
	stackName := m.ctx.StackName

	// Use injected StackOperator - it owns the channel and returns receive-only
	// Create a child context for preview so it can be cancelled independently
	previewCtx, previewCancel := context.WithCancel(m.appCtx)
	m.previewCancel = previewCancel
	m.previewCh = m.deps.StackOperator.Preview(previewCtx, workDir, stackName, op, opts)

	return waitForPreviewEvent(m.previewCh)
}

// maybeConfirmExecution checks if confirmation is needed before executing
// Confirmation is needed if the user is not on the preview screen for the requested operation
func (m *Model) maybeConfirmExecution(op pulumi.OperationType) tea.Cmd {
	// Don't start execution if an operation is already running (prevents race with preview)
	if m.state.OpState.IsActive() {
		return nil
	}
	// If we're on the preview screen for this exact operation, execute directly
	if m.ui.ViewMode == ui.ViewPreview && m.state.Operation == op {
		return m.startExecution(op)
	}

	// Otherwise, show confirmation modal
	m.state.PendingOperation = &op
	m.ui.ConfirmModal.SetLabels("Cancel", "Execute")
	m.ui.ConfirmModal.SetKeys("n", "y")
	m.ui.ConfirmModal.Show(
		"Execute "+op.String(),
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
		Excludes: m.ui.ResourceList.GetExcludeURNs(),
	}

	// Merge base env with plugin credentials
	opts.Env = mergeEnvMaps(m.deps.Env, m.deps.PluginProvider.GetAllEnv())

	// Create cancellable context as child of app context
	m.operationCtx, m.operationCancel = context.WithCancel(m.appCtx)

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
	appCtx := m.appCtx

	return func() tea.Msg {
		result, err := resourceImporter.StateDelete(
			appCtx,
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

// executeBulkStateDelete runs pulumi state delete for multiple resources
// It processes each resource sequentially and reports partial failures
func (m *Model) executeBulkStateDelete() tea.Cmd {
	resources := m.ui.ConfirmModal.GetBulkResources()

	// Build options with plugin env vars
	opts := pulumi.StateDeleteOptions{}
	if m.deps != nil && m.deps.PluginProvider != nil {
		opts.Env = m.deps.PluginProvider.GetAllEnv()
	}

	workDir := m.ctx.WorkDir
	stackName := m.ctx.StackName
	resourceImporter := m.deps.ResourceImporter
	appCtx := m.appCtx

	return func() tea.Msg {
		var succeeded, failed int
		var errors []string

		for _, res := range resources {
			result, err := resourceImporter.StateDelete(
				appCtx,
				workDir,
				stackName,
				res.URN,
				opts,
			)
			if err != nil {
				failed++
				errors = append(errors, fmt.Sprintf("%s: %v", res.Name, err))
				continue
			}
			if result.Success {
				succeeded++
			} else {
				failed++
				errMsg := "unknown error"
				if result.Error != nil {
					errMsg = result.Error.Error()
				}
				errors = append(errors, fmt.Sprintf("%s: %s", res.Name, errMsg))
			}
		}

		return bulkStateDeleteResultMsg{
			Succeeded: succeeded,
			Failed:    failed,
			Errors:    errors,
		}
	}
}

// executeProtect runs the pulumi state protect or unprotect command
func (m *Model) executeProtect(urn, name string, protect bool) tea.Cmd {
	// Build options with plugin env vars
	opts := pulumi.StateProtectOptions{}
	if m.deps != nil && m.deps.PluginProvider != nil {
		opts.Env = m.deps.PluginProvider.GetAllEnv()
	}

	workDir := m.ctx.WorkDir
	stackName := m.ctx.StackName
	resourceImporter := m.deps.ResourceImporter
	appCtx := m.appCtx

	return func() tea.Msg {
		var result *pulumi.CommandResult
		var err error

		if protect {
			result, err = resourceImporter.Protect(appCtx, workDir, stackName, urn, opts)
		} else {
			result, err = resourceImporter.Unprotect(appCtx, workDir, stackName, urn, opts)
		}

		if err != nil {
			return protectResultMsg{
				Result:    &pulumi.CommandResult{Success: false, Error: err},
				Protected: protect,
				URN:       urn,
				Name:      name,
			}
		}
		return protectResultMsg{
			Result:    result,
			Protected: protect,
			URN:       urn,
			Name:      name,
		}
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
	appCtx := m.appCtx

	return func() tea.Msg {
		result, err := resourceImporter.Import(
			appCtx,
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
	appCtx := m.appCtx
	opts := pulumi.ReadOptions{Env: m.deps.Env}
	return func() tea.Msg {
		history, err := stackReader.GetHistory(appCtx, workDir, stackName, pulumi.DefaultHistoryPageSize, pulumi.DefaultHistoryPage, opts)
		if err != nil {
			return errMsg(err)
		}
		return stackHistoryMsg(history)
	}
}

// fetchImportSuggestions queries plugins for import suggestions
func (m *Model) fetchImportSuggestions(resourceType, resourceName, resourceURN, parentURN, providerURN string, inputs, providerInputs map[string]any) tea.Cmd {
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

	// Convert provider inputs to string map for proto
	providerInputStrings := make(map[string]string)
	for k, v := range providerInputs {
		switch val := v.(type) {
		case string:
			providerInputStrings[k] = val
		default:
			// For non-string values, JSON serialize them
			if b, err := json.Marshal(val); err == nil {
				providerInputStrings[k] = string(b)
			}
		}
	}

	appCtx := m.appCtx
	pluginProvider := m.deps.PluginProvider
	return func() tea.Msg {
		req := &plugins.ImportSuggestionsRequest{
			ResourceType:   resourceType,
			ResourceName:   resourceName,
			ResourceUrn:    resourceURN,
			ParentUrn:      parentURN,
			Inputs:         inputStrings,
			ProviderUrn:    providerURN,
			ProviderInputs: providerInputStrings,
		}

		suggestions, err := pluginProvider.GetImportSuggestions(appCtx, req)
		if err != nil {
			return importSuggestionsErrMsg(err)
		}
		return importSuggestionsMsg(suggestions)
	}
}

// authenticatePluginsWithLock sets the busy lock, queues an operation, and runs auth.
// When auth completes (success or error), the lock is released and pending ops execute.
func (m *Model) authenticatePluginsWithLock(pendingOp PendingOperation) tea.Cmd {
	// Set busy lock before starting auth
	m.state.SetBusy("auth")
	m.state.QueueOperation(pendingOp)

	if m.deps == nil || m.deps.PluginProvider == nil {
		// No plugin provider - return completion immediately to release lock
		return func() tea.Msg {
			return authCompleteMsg{results: nil, err: nil}
		}
	}

	workDir := m.ctx.WorkDir
	stackName := m.ctx.StackName
	pluginProvider := m.deps.PluginProvider
	workspaceReader := m.deps.WorkspaceReader
	appCtx := m.appCtx
	opts := pulumi.ReadOptions{Env: m.deps.Env}

	return func() tea.Msg {
		// Get project info for the program name
		info, err := workspaceReader.GetProjectInfo(appCtx, workDir, stackName, opts)
		if err != nil {
			return authCompleteMsg{results: nil, err: err}
		}

		results, err := pluginProvider.Initialize(
			appCtx,
			workDir,
			info.ProgramName,
			info.StackName,
		)
		return authCompleteMsg{results: results, err: err}
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
	appCtx := m.appCtx
	opts := pulumi.ReadOptions{Env: m.deps.Env}
	return func() tea.Msg {
		info, err := workspaceReader.GetProjectInfo(appCtx, workDir, stackName, opts)
		if err != nil {
			return errMsg(err)
		}
		return projectInfoMsg(info)
	}
}

// fetchStacksList returns a command to load the list of available stacks from both backend and config files
func (m *Model) fetchStacksList() tea.Cmd {
	workDir := m.ctx.WorkDir
	stackReader := m.deps.StackReader
	workspaceReader := m.deps.WorkspaceReader
	appCtx := m.appCtx
	opts := pulumi.ReadOptions{Env: m.deps.Env}
	return func() tea.Msg {
		// Get backend stacks (non-fatal if fails - we can still show file-based stacks)
		stacks, _ := stackReader.GetStacks(appCtx, workDir, opts)

		// Also get stack files (non-fatal if fails)
		files, _ := workspaceReader.ListStackFiles(workDir)

		return stacksListMsg{
			Stacks: stacks,
			Files:  files,
		}
	}
}

// selectStack returns a command that triggers stack selection.
// This does NOT call Pulumi's SelectStack API because:
// 1. Plugin auth needs to happen first to get correct env vars
// 2. Operations explicitly pass the stack name with proper env vars
// The stackSelectedMsg handler will trigger auth and load resources.
func (m *Model) selectStack(name string) tea.Cmd {
	return func() tea.Msg {
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
	appCtx := m.appCtx
	opts := pulumi.ReadOptions{Env: m.deps.Env}
	return func() tea.Msg {
		info, err := workspaceReader.GetWhoAmI(appCtx, workDir, opts)
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
	appCtx := m.appCtx
	// Merge base env with plugin credentials
	var pluginEnv map[string]string
	if m.deps != nil && m.deps.PluginProvider != nil {
		pluginEnv = m.deps.PluginProvider.GetAllEnv()
	}
	env := mergeEnvMaps(m.deps.Env, pluginEnv)
	return func() tea.Msg {
		opts := pulumi.InitStackOptions{
			SecretsProvider: secretsProvider,
			Passphrase:      passphrase,
			Env:             env,
		}

		err := stackInitializer.InitStack(appCtx, workDir, name, opts)
		if err != nil {
			return stackInitResultMsg{StackName: name, Error: err}
		}
		return stackInitResultMsg{StackName: name, Error: nil}
	}
}

// fetchOpenResourceAction queries plugins for an action to open the resource
func (m *Model) fetchOpenResourceAction(resourceType, resourceName, resourceURN, providerURN string, inputs, outputs, providerInputs map[string]any) tea.Cmd {
	if m.deps == nil || m.deps.PluginProvider == nil {
		return func() tea.Msg {
			return openResourceActionMsg{Response: nil, PluginName: ""}
		}
	}

	// Convert inputs to string map for proto
	inputStrings := make(map[string]string)
	for k, v := range inputs {
		switch val := v.(type) {
		case string:
			inputStrings[k] = val
		default:
			if b, err := json.Marshal(val); err == nil {
				inputStrings[k] = string(b)
			}
		}
	}

	// Convert outputs to string map for proto
	outputStrings := make(map[string]string)
	for k, v := range outputs {
		switch val := v.(type) {
		case string:
			outputStrings[k] = val
		default:
			if b, err := json.Marshal(val); err == nil {
				outputStrings[k] = string(b)
			}
		}
	}

	// Convert provider inputs to string map for proto
	providerInputStrings := make(map[string]string)
	for k, v := range providerInputs {
		switch val := v.(type) {
		case string:
			providerInputStrings[k] = val
		default:
			if b, err := json.Marshal(val); err == nil {
				providerInputStrings[k] = string(b)
			}
		}
	}

	appCtx := m.appCtx
	pluginProvider := m.deps.PluginProvider
	return func() tea.Msg {
		req := &plugins.OpenResourceRequest{
			ResourceType:   resourceType,
			ResourceName:   resourceName,
			ResourceUrn:    resourceURN,
			ProviderUrn:    providerURN,
			ProviderInputs: providerInputStrings,
			Inputs:         inputStrings,
			Outputs:        outputStrings,
		}

		resp, pluginName, err := pluginProvider.OpenResource(appCtx, req)
		if err != nil {
			return openResourceErrMsg(err)
		}
		return openResourceActionMsg{Response: resp, PluginName: pluginName}
	}
}

// openInBrowser opens a URL in the default browser
func openInBrowser(url string) tea.Cmd {
	return func() tea.Msg {
		if err := browser.OpenURL(url); err != nil {
			return openResourceErrMsg(fmt.Errorf("failed to open browser: %w", err))
		}
		return nil
	}
}

// openWithExec launches an alternate screen program using tea.ExecProcess
func openWithExec(command string, args []string, env map[string]string) tea.Cmd {
	cmd := exec.Command(command, args...)

	// Set environment variables
	if len(env) > 0 {
		cmd.Env = append(cmd.Environ(), mapToEnvSlice(env)...)
	}

	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return openResourceExecDoneMsg{Error: err}
	})
}

// mapToEnvSlice converts a map to a slice of KEY=VALUE strings
func mapToEnvSlice(m map[string]string) []string {
	result := make([]string, 0, len(m))
	for k, v := range m {
		result = append(result, k+"="+v)
	}
	return result
}

// mergeEnvMaps merges multiple env maps, with later maps taking precedence
func mergeEnvMaps(envMaps ...map[string]string) map[string]string {
	result := make(map[string]string)
	for _, m := range envMaps {
		maps.Copy(result, m)
	}
	return result
}

// CanOpenResource checks if a resource can be opened (requires plugins)
func CanOpenResource(viewMode ui.ViewMode, item *ui.ResourceItem, hasResourceOpeners bool) bool {
	// Only works in stack view with selected resource and active resource opener plugins
	if viewMode != ui.ViewStack && viewMode != ui.ViewPreview {
		return false
	}
	if item == nil {
		return false
	}
	// Don't allow opening the root stack resource
	if item.Type == "pulumi:pulumi:Stack" {
		return false
	}
	return hasResourceOpeners
}
