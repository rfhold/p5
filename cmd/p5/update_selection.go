package main

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/rfhold/p5/internal/ui"
)

// handleStacksList handles the loaded list of stacks
// During initialization (InitLoadingStacks state), this determines whether to:
// - Show stack init modal (no stacks exist)
// - Show stack selector (stacks exist but none selected)
// - Proceed to loading resources (stack already selected)
func (m Model) handleStacksList(msg stacksListMsg) (tea.Model, tea.Cmd) {
	result := ConvertStacksToItems(msg)
	items := result.Items
	currentStackName := result.CurrentStackName
	m.ui.StackSelector.SetStacks(items)

	action := DetermineStackInitAction(m.state.InitState, len(items), currentStackName)

	switch action {
	case StackInitActionShowInit:
		m.transitionTo(InitSelectingStack)
		m.showStackInitModal()
		// Pass auth env from plugins for passphrase detection
		if m.deps != nil && m.deps.PluginProvider != nil {
			m.ui.StackInitModal.SetAuthEnv(m.deps.PluginProvider.GetMergedAuthEnv())
		}
		return m, tea.Batch(m.fetchWhoAmI(), m.fetchStackFiles())

	case StackInitActionShowSelector:
		m.transitionTo(InitSelectingStack)
		m.showStackSelector()
		m.ui.StackSelector.SetLoading(false) // Already loaded
		return m, nil

	case StackInitActionProceed:
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
	m.ctx.StackName = string(msg)
	m.hideDetailsPanel() // Close details panel when stack changes
	m.hideStackSelector()
	m.ui.ResourceList.Clear()

	if m.state.InitState == InitSelectingStack {
		m.transitionTo(InitLoadingResources)
	}

	if m.deps != nil && m.deps.PluginProvider != nil {
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
	items := ConvertWorkspacesToItems(msg, m.ctx.Cwd)
	m.ui.WorkspaceSelector.SetWorkspaces(items)
	return m, nil
}

// handleWorkspaceSelected handles a workspace being selected.
// This restarts the init state machine from InitLoadingPlugins for the new workspace.
func (m Model) handleWorkspaceSelected(msg workspaceSelectedMsg) (tea.Model, tea.Cmd) {
	m.ctx.WorkDir = string(msg)
	m.ctx.StackName = ""
	m.hideDetailsPanel()
	m.hideWorkspaceSelector()
	m.ui.ResourceList.Clear()

	m.transitionTo(InitLoadingPlugins)

	if m.deps != nil && m.deps.PluginProvider != nil {
		mergedConfig := m.deps.PluginProvider.GetMergedConfig()
		m.deps.PluginProvider.InvalidateCredentialsForContext(m.ctx.WorkDir, m.ctx.StackName, "", mergedConfig)
	}
	return m, m.authenticatePluginsForWorkspace()
}
