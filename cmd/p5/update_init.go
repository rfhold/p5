package main

import (
	"fmt"
	"log"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/rfhold/p5/internal/ui"
)

// Initialization handlers - handles app startup and workspace/plugin initialization flow
// This file implements an explicit state machine for the initialization flow:
//
// InitCheckingWorkspace → InitLoadingPlugins → InitLoadingStacks → InitSelectingStack → InitLoadingResources → InitComplete
//                    ↓                                                    ↓
//              [show workspace selector]                           [show stack selector/init modal]

// transitionTo moves the init state machine to a new state with logging
func (m *Model) transitionTo(newState InitState) {
	oldState := m.state.InitState
	m.state.InitState = newState
	log.Printf("[init] %s → %s", oldState, newState)
}

// startPluginAuth kicks off plugin authentication
// Called when transitioning to InitLoadingPlugins
func (m *Model) startPluginAuth() tea.Cmd {
	return m.authenticatePluginsForInit()
}

// handleWorkspaceCheck handles the result of checking if we're in a valid workspace
// State: InitCheckingWorkspace → InitLoadingPlugins (if valid) or shows workspace selector
func (m Model) handleWorkspaceCheck(msg workspaceCheckMsg) (tea.Model, tea.Cmd) {
	if msg {
		// We're in a valid workspace, transition to loading plugins
		m.transitionTo(InitLoadingPlugins)
		return m, m.startPluginAuth()
	}
	// Not in a workspace, show the workspace selector
	// Stay in InitCheckingWorkspace until workspace is selected
	m.showWorkspaceSelector()
	return m, m.fetchWorkspacesList()
}

// handlePluginInitDone handles completion of initial plugin authentication
// State: InitLoadingPlugins → InitLoadingStacks (or InitLoadingResources if stack specified)
func (m Model) handlePluginInitDone(msg pluginInitDoneMsg) (tea.Model, tea.Cmd) {
	// Initial plugin authentication completed
	var cmds []tea.Cmd

	// Apply plugin env vars to the process environment so Pulumi operations inherit them
	if m.deps != nil && m.deps.PluginProvider != nil {
		m.deps.PluginProvider.ApplyEnvToProcess()
	}

	// Show toast for plugin results (non-blocking notification)
	if msg.err != nil {
		cmds = append(cmds, m.ui.Toast.Show(fmt.Sprintf("Plugin error: %v", msg.err)))
	} else if len(msg.results) > 0 {
		summary := SummarizePluginAuthResults(msg.results)
		if len(summary.AuthenticatedPlugins) > 0 {
			cmds = append(cmds, m.ui.Toast.Show(fmt.Sprintf("Authenticated: %s", strings.Join(summary.AuthenticatedPlugins, ", "))))
		}
	}

	// Determine next state based on whether stack was specified
	if m.ctx.StackName == "" {
		// No stack specified, need to load stacks and potentially show selector
		m.transitionTo(InitLoadingStacks)
		cmds = append(cmds, m.fetchStacksList())
	} else {
		// Stack was specified on command line, go directly to loading resources
		m.transitionTo(InitLoadingResources)
		// Re-authenticate plugins with stack name to load stack-level config
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

// handlePluginAuthResult handles completion of plugin re-authentication (after stack/workspace change)
func (m Model) handlePluginAuthResult(msg pluginAuthResultMsg) (tea.Model, tea.Cmd) {
	// Plugin authentication completed (for re-auth after stack/workspace change)
	// Apply env vars to process environment
	if m.deps != nil && m.deps.PluginProvider != nil {
		m.deps.PluginProvider.ApplyEnvToProcess()
	}

	// Use pure function to summarize auth results
	summary := SummarizePluginAuthResults(msg)

	if summary.HasErrors {
		// Show error toast but don't block - credentials are optional
		return m, m.ui.Toast.Show(fmt.Sprintf("Plugin auth failed: %s", strings.Join(summary.ErrorMessages, "; ")))
	}
	// Show success if we have plugins that authenticated
	if len(summary.AuthenticatedPlugins) > 0 {
		return m, m.ui.Toast.Show(fmt.Sprintf("Authenticated: %s", strings.Join(summary.AuthenticatedPlugins, ", ")))
	}
	return m, nil
}

// handlePluginAuthError handles plugin system errors
func (m Model) handlePluginAuthError(msg pluginAuthErrorMsg) (tea.Model, tea.Cmd) {
	// Plugin system error - show but don't block
	return m, m.ui.Toast.Show(fmt.Sprintf("Plugin error: %v", error(msg)))
}

// handleProjectInfo handles project info loaded from Pulumi
func (m Model) handleProjectInfo(msg projectInfoMsg) (tea.Model, tea.Cmd) {
	m.ui.Header.SetData(&ui.HeaderData{
		ProgramName: msg.ProgramName,
		StackName:   msg.StackName,
		Runtime:     msg.Runtime,
	})
	return m, nil
}

// handleError handles general errors
// During initialization, errors may occur at various states. We log the state
// for debugging but don't block the user from trying again.
func (m Model) handleError(msg errMsg) (tea.Model, tea.Cmd) {
	log.Printf("[init] error in state %s: %v", m.state.InitState, msg)
	m.ui.Header.SetError(msg)
	m.ui.ResourceList.SetError(msg)
	m.state.Err = msg

	// If we errored during init, mark as complete so user can interact
	// They can retry by switching stacks/workspaces or restarting
	if m.state.InitState != InitComplete {
		m.transitionTo(InitComplete)
	}

	return m, nil
}

// handleWhoAmI handles backend connection info for stack init
func (m Model) handleWhoAmI(msg whoAmIMsg) (tea.Model, tea.Cmd) {
	if msg != nil {
		m.ui.StackInitModal.SetBackendInfo(msg.User, msg.URL)
	}
	return m, nil
}

// handleStackFiles handles stack config files list for stack init
func (m Model) handleStackFiles(msg stackFilesMsg) (tea.Model, tea.Cmd) {
	m.ui.StackInitModal.SetStackFiles(msg)
	return m, nil
}

// handleStackInitResult handles result of stack creation
// State: InitSelectingStack → InitLoadingResources (on success)
func (m Model) handleStackInitResult(msg stackInitResultMsg) (tea.Model, tea.Cmd) {
	if msg.Error != nil {
		// Stay in InitSelectingStack, show error in modal
		m.ui.StackInitModal.SetError(msg.Error)
		return m, nil
	}
	// Stack created successfully, hide modal and transition to loading resources
	m.hideStackInitModal()
	m.ctx.StackName = msg.StackName

	// Transition to loading resources
	if m.state.InitState == InitSelectingStack {
		m.transitionTo(InitLoadingResources)
	}

	// Reload everything for the new stack
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
