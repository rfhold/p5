package main

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/rfhold/p5/internal/ui"
)

// Selection handlers - handles stack and workspace selection

// handleStacksList handles the loaded list of stacks
// During initialization (InitLoadingStacks state), this determines whether to:
// - Show stack init modal (no stacks exist)
// - Show stack selector (stacks exist but none selected)
// - Proceed to loading resources (stack already selected)
func (m Model) handleStacksList(msg stacksListMsg) (tea.Model, tea.Cmd) {
	// Convert to UI stack items using pure function
	result := ConvertStacksToItems(msg)
	items := result.Items
	currentStackName := result.CurrentStackName
	m.ui.StackSelector.SetStacks(items)

	// Determine action using pure function
	action := DetermineStackInitAction(m.state.InitState, len(items), currentStackName)

	switch action {
	case StackInitActionShowInit:
		// No stacks at all - show stack init modal
		m.transitionTo(InitSelectingStack)
		m.showStackInitModal()
		// Pass auth env from plugins for passphrase detection
		if m.deps != nil && m.deps.PluginProvider != nil {
			m.ui.StackInitModal.SetAuthEnv(m.deps.PluginProvider.GetMergedAuthEnv())
		}
		return m, tea.Batch(m.fetchWhoAmI(), m.fetchStackFiles())

	case StackInitActionShowSelector:
		// No current stack, show selector
		m.transitionTo(InitSelectingStack)
		m.showStackSelector()
		m.ui.StackSelector.SetLoading(false) // Already loaded
		return m, nil

	case StackInitActionProceed:
		// Has current stack - proceed to loading resources
		m.ctx.StackName = currentStackName
		m.transitionTo(InitLoadingResources)
		// Re-authenticate plugins with stack name to load stack-level config
		if m.deps != nil && m.deps.PluginProvider != nil {
			m.deps.PluginProvider.InvalidateAllCredentials()
		}
		cmds := []tea.Cmd{m.fetchProjectInfo(), m.authenticatePlugins()}
		if m.ui.ViewMode == ui.ViewPreview {
			cmds = append(cmds, m.initPreview(m.state.Operation))
		} else {
			cmds = append(cmds, m.initLoadStackResources())
		}
		return m, tea.Batch(cmds...)
	}

	return m, nil
}

// handleStackSelected handles a stack being selected
// State: InitSelectingStack → InitLoadingResources (during init)
// Also handles runtime stack switching (when initState is InitComplete)
func (m Model) handleStackSelected(msg stackSelectedMsg) (tea.Model, tea.Cmd) {
	// Stack was selected, update the context and reload everything
	m.ctx.StackName = string(msg)
	m.hideDetailsPanel() // Close details panel when stack changes
	m.hideStackSelector()
	m.ui.ResourceList.Clear()

	// Transition to loading resources if we were in stack selection during init
	if m.state.InitState == InitSelectingStack {
		m.transitionTo(InitLoadingResources)
	}

	// Invalidate credentials based on plugin refresh triggers
	if m.deps != nil && m.deps.PluginProvider != nil {
		// Use the merged config for checking refresh triggers
		mergedConfig := m.deps.PluginProvider.GetMergedConfig()
		m.deps.PluginProvider.InvalidateCredentialsForContext(m.ctx.WorkDir, m.ctx.StackName, "", mergedConfig)
	}
	cmds := []tea.Cmd{m.fetchProjectInfo(), m.authenticatePlugins()}
	if m.ui.ViewMode == ui.ViewPreview {
		cmds = append(cmds, m.initPreview(m.state.Operation))
	} else {
		cmds = append(cmds, m.loadStackResources())
	}
	return m, tea.Batch(cmds...)
}

// handleWorkspacesList handles the loaded list of workspaces
func (m Model) handleWorkspacesList(msg workspacesListMsg) (tea.Model, tea.Cmd) {
	// Convert to UI workspace items using pure function
	items := ConvertWorkspacesToItems(msg, m.ctx.Cwd)
	m.ui.WorkspaceSelector.SetWorkspaces(items)
	return m, nil
}

// handleWorkspaceSelected handles a workspace being selected
// This restarts the init state machine from InitLoadingPlugins for the new workspace
func (m Model) handleWorkspaceSelected(msg workspaceSelectedMsg) (tea.Model, tea.Cmd) {
	// Workspace was selected, update the context and reload everything
	m.ctx.WorkDir = string(msg)
	m.ctx.StackName = "" // Reset stack selection for new workspace
	m.hideDetailsPanel() // Close details panel when workspace changes
	m.hideWorkspaceSelector()
	m.ui.ResourceList.Clear()

	// Restart init flow from plugin loading
	m.transitionTo(InitLoadingPlugins)

	// Invalidate credentials based on plugin refresh triggers
	if m.deps != nil && m.deps.PluginProvider != nil {
		// Use the merged config for checking refresh triggers
		mergedConfig := m.deps.PluginProvider.GetMergedConfig()
		m.deps.PluginProvider.InvalidateCredentialsForContext(m.ctx.WorkDir, m.ctx.StackName, "", mergedConfig)
	}
	// Re-authenticate plugins for the new workspace (picks up p5.toml from new workDir)
	// This must complete BEFORE fetching stacks/project info, because plugins may set
	// required env vars like PULUMI_BACKEND_URL. Use authenticatePluginsForWorkspace
	// which returns pluginInitDoneMsg, triggering handlePluginInitDone flow
	return m, m.authenticatePluginsForWorkspace()
}
