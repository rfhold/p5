package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/rfhold/p5/internal/ui"
)

// transitionTo moves the init state machine to a new state with logging
func (m *Model) transitionTo(newState InitState) {
	oldState := m.state.InitState
	m.state.InitState = newState
	m.deps.Logger.Debug("init state transition",
		"from", oldState.String(),
		"to", newState.String())
}

// startPluginAuth kicks off plugin authentication.
func (m *Model) startPluginAuth() tea.Cmd {
	return m.authenticatePluginsForInit()
}

// handleWorkspaceCheck handles the result of checking if we're in a valid workspace.
func (m Model) handleWorkspaceCheck(msg workspaceCheckMsg) (tea.Model, tea.Cmd) {
	if msg {
		m.transitionTo(InitLoadingPlugins)
		return m, m.startPluginAuth()
	}
	m.showWorkspaceSelector()
	return m, m.fetchWorkspacesList()
}

// handlePluginInitDone handles completion of initial plugin authentication.
func (m Model) handlePluginInitDone(msg pluginInitDoneMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if m.deps != nil && m.deps.PluginProvider != nil {
		m.deps.PluginProvider.ApplyEnvToProcess()
	}

	if msg.err != nil {
		cmds = append(cmds, m.ui.Toast.Show(fmt.Sprintf("Plugin error: %v", msg.err)))
	} else if len(msg.results) > 0 {
		summary := SummarizePluginAuthResults(msg.results)
		if len(summary.AuthenticatedPlugins) > 0 {
			cmds = append(cmds, m.ui.Toast.Show("Authenticated: "+strings.Join(summary.AuthenticatedPlugins, ", ")))
		}
	}

	if m.ctx.StackName == "" {
		m.transitionTo(InitLoadingStacks)
		cmds = append(cmds, m.fetchStacksList())
	} else {
		m.transitionTo(InitLoadingResources)
		if m.deps != nil && m.deps.PluginProvider != nil {
			m.deps.PluginProvider.InvalidateAllCredentials()
		}
		cmds = append(cmds, m.fetchProjectInfo(), m.authenticatePlugins())
		if m.ui.ViewMode == ui.ViewPreview {
			cmds = append(cmds, m.initPreview(m.state.Operation))
		} else {
			cmds = append(cmds, m.initLoadStackResources())
		}
	}

	return m, tea.Batch(cmds...)
}

// handlePluginAuthResult handles completion of plugin re-authentication.
func (m Model) handlePluginAuthResult(msg pluginAuthResultMsg) (tea.Model, tea.Cmd) {
	if m.deps != nil && m.deps.PluginProvider != nil {
		m.deps.PluginProvider.ApplyEnvToProcess()
	}

	summary := SummarizePluginAuthResults(msg)

	if summary.HasErrors {
		return m, m.ui.Toast.Show("Plugin auth failed: " + strings.Join(summary.ErrorMessages, "; "))
	}
	if len(summary.AuthenticatedPlugins) > 0 {
		return m, m.ui.Toast.Show("Authenticated: " + strings.Join(summary.AuthenticatedPlugins, ", "))
	}
	return m, nil
}

// handlePluginAuthError handles plugin system errors
func (m Model) handlePluginAuthError(msg pluginAuthErrorMsg) (tea.Model, tea.Cmd) {
	return m, m.ui.Toast.Show(fmt.Sprintf("Plugin error: %v", error(msg)))
}

// handleProjectInfo handles project info loaded from Pulumi
func (m Model) handleProjectInfo(msg projectInfoMsg) (tea.Model, tea.Cmd) { //nolint:unparam // Bubble Tea handler signature
	m.ui.Header.SetData(&ui.HeaderData{
		ProgramName: msg.ProgramName,
		StackName:   msg.StackName,
		Runtime:     msg.Runtime,
	})
	return m, nil
}

// handleError handles general errors.
func (m Model) handleError(msg errMsg) (tea.Model, tea.Cmd) { //nolint:unparam // Bubble Tea handler signature
	m.deps.Logger.Error("init error",
		"state", m.state.InitState.String(),
		"error", error(msg))
	m.ui.Header.SetError(msg)
	m.ui.ResourceList.SetError(msg)
	m.state.Err = msg

	if m.state.InitState != InitComplete {
		m.transitionTo(InitComplete)
	}

	return m, nil
}

// handleWhoAmI handles backend connection info for stack init
func (m Model) handleWhoAmI(msg whoAmIMsg) (tea.Model, tea.Cmd) { //nolint:unparam // Bubble Tea handler signature
	if msg != nil {
		m.ui.StackInitModal.SetBackendInfo(msg.User, msg.URL)
	}
	return m, nil
}

// handleStackFiles handles stack config files list for stack init
func (m Model) handleStackFiles(msg stackFilesMsg) (tea.Model, tea.Cmd) { //nolint:unparam // Bubble Tea handler signature
	m.ui.StackInitModal.SetStackFiles(msg)
	return m, nil
}

// handleStackInitResult handles result of stack creation.
func (m Model) handleStackInitResult(msg stackInitResultMsg) (tea.Model, tea.Cmd) {
	if msg.Error != nil {
		m.ui.StackInitModal.SetError(msg.Error)
		return m, nil
	}
	m.hideStackInitModal()
	m.ctx.StackName = msg.StackName

	// Transition to loading resources
	if m.state.InitState == InitSelectingStack {
		m.transitionTo(InitLoadingResources)
	}

	cmds := []tea.Cmd{
		m.ui.Toast.Show(fmt.Sprintf("Created stack '%s'", msg.StackName)),
		m.fetchProjectInfo(),
	}
	if m.ui.ViewMode == ui.ViewPreview {
		cmds = append(cmds, m.initPreview(m.state.Operation))
	} else {
		cmds = append(cmds, m.loadStackResources())
	}
	return m, tea.Batch(cmds...)
}
