package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/rfhold/p5/internal/plugins"
	"github.com/rfhold/p5/internal/pulumi"
	"github.com/rfhold/p5/internal/ui"
)

var workDir string
var stackName string
var startView string

// Messages
type projectInfoMsg *pulumi.ProjectInfo
type errMsg error
type previewEventMsg pulumi.PreviewEvent
type operationEventMsg pulumi.OperationEvent
type stackResourcesMsg []pulumi.ResourceInfo
type stacksListMsg []pulumi.StackInfo
type stackSelectedMsg string
type workspacesListMsg []pulumi.WorkspaceInfo
type workspaceSelectedMsg string
type workspaceCheckMsg bool // true if current dir is a valid workspace
type stackHistoryMsg []pulumi.UpdateSummary

// Plugin-related messages
type pluginAuthResultMsg []plugins.AuthenticateResult
type pluginAuthErrorMsg error

// Model is the main application model
type Model struct {
	// Shared persistent state
	flags map[string]ui.ResourceFlags // Persists across all views

	// Current view state
	viewMode  ui.ViewMode
	operation pulumi.OperationType

	// Components
	header            ui.Header
	resourceList      *ui.ResourceList
	historyList       *ui.HistoryList
	help              *ui.HelpDialog
	details           *ui.DetailPanel
	historyDetails    *ui.HistoryDetailPanel
	stackSelector     *ui.StackSelector
	workspaceSelector *ui.WorkspaceSelector
	toast             *ui.Toast
	showHelp          bool

	// Plugin system
	pluginManager *plugins.Manager

	// Error state
	err      error
	quitting bool

	// Dimensions
	width  int
	height int

	// Event channels
	previewCh   chan pulumi.PreviewEvent
	operationCh chan pulumi.OperationEvent

	// Operation context for cancellation
	operationCtx    context.Context
	operationCancel context.CancelFunc
}

func initialModel() Model {
	flags := make(map[string]ui.ResourceFlags)

	// Initialize plugin manager with launch directory for p5.toml discovery
	pluginMgr, err := plugins.NewManager(workDir)
	if err != nil {
		// Log but don't fail - plugins are optional
		fmt.Fprintf(os.Stderr, "Warning: failed to initialize plugin manager: %v\n", err)
	}

	m := Model{
		flags:             flags,
		viewMode:          ui.ViewStack,
		header:            ui.NewHeader(),
		resourceList:      ui.NewResourceList(flags),
		historyList:       ui.NewHistoryList(),
		help:              ui.NewHelpDialog(),
		details:           ui.NewDetailPanel(),
		historyDetails:    ui.NewHistoryDetailPanel(),
		stackSelector:     ui.NewStackSelector(),
		toast:             ui.NewToast(),
		workspaceSelector: ui.NewWorkspaceSelector(),
		pluginManager:     pluginMgr,
	}

	// Set initial view based on command argument
	switch startView {
	case "up":
		m.viewMode = ui.ViewPreview
		m.operation = pulumi.OperationUp
		m.resourceList.SetShowAllOps(false)
	case "refresh":
		m.viewMode = ui.ViewPreview
		m.operation = pulumi.OperationRefresh
		m.resourceList.SetShowAllOps(false)
	case "destroy":
		m.viewMode = ui.ViewPreview
		m.operation = pulumi.OperationDestroy
		m.resourceList.SetShowAllOps(false)
	}

	m.header.SetViewMode(m.viewMode)
	m.header.SetOperation(m.operation)

	return m
}

// Init starts the initial data fetch
func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		m.header.Spinner().Tick,
		m.resourceList.Spinner().Tick,
		m.historyList.Spinner().Tick,
	}

	// First check if we're in a valid Pulumi workspace
	cmds = append(cmds, checkWorkspace)

	return tea.Batch(cmds...)
}

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

// pluginInitDoneMsg is sent when initial plugin auth completes
type pluginInitDoneMsg struct {
	results []plugins.AuthenticateResult
	err     error
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

// initPreviewMsg is sent to start a preview from Init
type initPreviewMsg struct {
	op pulumi.OperationType
	ch chan pulumi.PreviewEvent
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

// startExecution starts an execution operation
func (m *Model) startExecution(op pulumi.OperationType) tea.Cmd {
	m.viewMode = ui.ViewExecute
	m.operation = op
	m.header.SetViewMode(m.viewMode)
	m.header.SetOperation(m.operation)

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
	m.resourceList.Clear()
	m.resourceList.SetShowAllOps(true)
	return m.loadStackResources()
}

// switchToHistoryView switches to history view
func (m *Model) switchToHistoryView() tea.Cmd {
	m.viewMode = ui.ViewHistory
	m.header.SetViewMode(m.viewMode)
	m.historyList.Clear()
	m.historyList.SetLoading(true, "Loading stack history...")
	return fetchStackHistory
}

// fetchStackHistory loads the stack history
func fetchStackHistory() tea.Msg {
	history, err := pulumi.GetStackHistory(context.Background(), workDir, stackName, 50, 1)
	if err != nil {
		return errMsg(err)
	}
	return stackHistoryMsg(history)
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

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.header.SetWidth(msg.Width)
		m.help.SetSize(msg.Width, msg.Height)
		m.stackSelector.SetSize(msg.Width, msg.Height)
		m.workspaceSelector.SetSize(msg.Width, msg.Height)
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

	case tea.MouseMsg:
		// Handle mouse events for text selection in details panel
		if m.details.Visible() {
			if cmd := m.details.HandleMouseEvent(msg); cmd != nil {
				return m, cmd
			}
		}
		return m, nil

	case ui.CopiedToClipboardMsg:
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

	case ui.ToastHideMsg:
		m.toast.Hide()
		return m, nil

	case ui.FlashClearMsg:
		m.resourceList.ClearFlash()
		return m, nil

	case tea.KeyMsg:
		// Handle workspace selector first if visible
		if m.workspaceSelector.Visible() {
			selected, cmd := m.workspaceSelector.Update(msg)
			if selected {
				// Workspace was selected, update and reload
				selectedWs := m.workspaceSelector.SelectedWorkspace()
				if selectedWs != nil {
					return m, selectWorkspace(selectedWs.Path)
				}
			}
			return m, cmd
		}

		// Handle stack selector if visible
		if m.stackSelector.Visible() {
			selected, cmd := m.stackSelector.Update(msg)
			if selected {
				// Stack was selected, update and reload
				selectedStack := m.stackSelector.SelectedStack()
				if selectedStack != "" {
					return m, selectStack(selectedStack)
				}
			}
			return m, cmd
		}

		// Handle help toggle first
		if key.Matches(msg, ui.Keys.Help) {
			m.showHelp = !m.showHelp
			return m, nil
		}

		// If help is showing, handle scrolling or close on other keys
		if m.showHelp {
			// Allow scrolling keys
			if key.Matches(msg, ui.Keys.Up) || key.Matches(msg, ui.Keys.Down) ||
				key.Matches(msg, ui.Keys.PageUp) || key.Matches(msg, ui.Keys.PageDown) {
				m.help.Update(msg)
				return m, nil
			}
			// Esc or q closes help
			if key.Matches(msg, ui.Keys.Escape) || key.Matches(msg, ui.Keys.Quit) {
				m.showHelp = false
				return m, nil
			}
			// Any other key is ignored while help is open
			return m, nil
		}

		// Escape handling
		if key.Matches(msg, ui.Keys.Escape) {
			// Clear text selection first if active (details panel)
			if m.details.HasSelection() {
				m.details.ClearSelection()
				return m, nil
			}
			// Cancel visual mode
			if m.resourceList.VisualMode() {
				cmd := m.resourceList.Update(msg)
				return m, cmd
			}
			// Cancel running operation
			if m.viewMode == ui.ViewExecute && m.operationCancel != nil {
				m.operationCancel()
				return m, nil
			}
			// Go back to stack view from preview, history, or completed execution
			if m.viewMode == ui.ViewPreview || m.viewMode == ui.ViewExecute || m.viewMode == ui.ViewHistory {
				return m, m.switchToStackView()
			}
			return m, nil
		}

		// Quit
		if key.Matches(msg, ui.Keys.Quit) {
			m.quitting = true
			return m, tea.Quit
		}

		// Details panel toggle
		if key.Matches(msg, ui.Keys.ToggleDetails) {
			if m.viewMode == ui.ViewHistory {
				// Toggle history details panel
				m.historyDetails.Toggle()
				if m.historyDetails.Visible() {
					m.historyDetails.SetItem(m.historyList.SelectedItem())
				}
			} else {
				// Toggle resource details panel
				m.details.Toggle()
				if m.details.Visible() {
					m.details.SetResource(m.resourceList.SelectedItem())
				}
			}
			return m, nil
		}

		// Stack selector toggle
		if key.Matches(msg, ui.Keys.SelectStack) {
			m.stackSelector.SetLoading(true)
			m.stackSelector.Show()
			return m, fetchStacksList
		}

		// Workspace selector toggle
		if key.Matches(msg, ui.Keys.SelectWorkspace) {
			m.workspaceSelector.SetLoading(true)
			m.workspaceSelector.Show()
			return m, fetchWorkspacesList
		}

		// History view toggle
		if key.Matches(msg, ui.Keys.ViewHistory) {
			return m, m.switchToHistoryView()
		}

		// Preview operations (lowercase u/r/d)
		if key.Matches(msg, ui.Keys.PreviewUp) {
			return m, m.startPreview(pulumi.OperationUp)
		}
		if key.Matches(msg, ui.Keys.PreviewRefresh) {
			return m, m.startPreview(pulumi.OperationRefresh)
		}
		if key.Matches(msg, ui.Keys.PreviewDestroy) {
			return m, m.startPreview(pulumi.OperationDestroy)
		}

		// Execute operations (ctrl+u/r/d)
		if key.Matches(msg, ui.Keys.ExecuteUp) {
			return m, m.startExecution(pulumi.OperationUp)
		}
		if key.Matches(msg, ui.Keys.ExecuteRefresh) {
			return m, m.startExecution(pulumi.OperationRefresh)
		}
		if key.Matches(msg, ui.Keys.ExecuteDestroy) {
			return m, m.startExecution(pulumi.OperationDestroy)
		}

		// Forward keys to appropriate list for cursor/selection handling
		if m.viewMode == ui.ViewHistory {
			cmd := m.historyList.Update(msg)
			// Update history details panel with newly selected item
			if m.historyDetails.Visible() {
				m.historyDetails.SetItem(m.historyList.SelectedItem())
			}
			return m, cmd
		}
		cmd := m.resourceList.Update(msg)
		// Update details panel with newly selected resource
		if m.details.Visible() {
			m.details.SetResource(m.resourceList.SelectedItem())
		}
		return m, cmd

	case projectInfoMsg:
		m.header.SetData(&ui.HeaderData{
			ProgramName: msg.ProgramName,
			StackName:   msg.StackName,
			Runtime:     msg.Runtime,
		})
		return m, nil

	case errMsg:
		m.header.SetError(msg)
		m.resourceList.SetError(msg)
		m.err = msg
		return m, nil

	case initPreviewMsg:
		// Store the channel and start listening for events
		m.previewCh = msg.ch
		m.resourceList.SetLoading(true, fmt.Sprintf("Running %s preview...", msg.op.String()))
		return m, waitForPreviewEvent(m.previewCh)

	case stackResourcesMsg:
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

	case previewEventMsg:
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

	case operationEventMsg:
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

	case stacksListMsg:
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

	case stackSelectedMsg:
		// Stack was selected, update the global and reload everything
		stackName = string(msg)
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

	case workspacesListMsg:
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

	case workspaceSelectedMsg:
		// Workspace was selected, update the global workDir and reload everything
		workDir = string(msg)
		stackName = "" // Reset stack selection for new workspace
		m.resourceList.Clear()
		// Invalidate credentials based on plugin refresh triggers
		if m.pluginManager != nil {
			// Use the merged config for checking refresh triggers
			mergedConfig := m.pluginManager.GetMergedConfig()
			m.pluginManager.InvalidateCredentialsForContext(workDir, stackName, "", mergedConfig)
		}
		// Fetch stacks for the new workspace (auth will happen after stack selection)
		return m, tea.Batch(fetchProjectInfo, fetchStacksList)

	case stackHistoryMsg:
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

	case workspaceCheckMsg:
		if msg {
			// We're in a valid workspace, continue with normal initialization
			return m, m.continueInit()
		}
		// Not in a workspace, show the workspace selector
		m.workspaceSelector.SetLoading(true)
		m.workspaceSelector.Show()
		return m, fetchWorkspacesList

	case pluginInitDoneMsg:
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

	case pluginAuthResultMsg:
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

	case pluginAuthErrorMsg:
		// Plugin system error - show but don't block
		return m, m.toast.Show(fmt.Sprintf("Plugin error: %v", error(msg)))

	case spinner.TickMsg:
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

	return m, nil
}

// View renders the UI
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	// Build header
	header := m.header.View()

	// Build footer with keybind hints
	footer := m.renderFooter()

	// Calculate available height for main content
	headerHeight := lipgloss.Height(header)
	footerHeight := lipgloss.Height(footer)
	mainHeight := m.height - headerHeight - footerHeight - 1

	if mainHeight < 1 {
		mainHeight = 1
	}

	// Build main content area
	var mainContent string
	if m.viewMode == ui.ViewHistory {
		m.historyList.SetSize(m.width, mainHeight)
		mainContent = m.historyList.View()
	} else {
		mainContent = m.resourceList.View()
	}
	mainArea := lipgloss.NewStyle().
		Height(mainHeight).
		Width(m.width).
		Render(mainContent)

	fullView := lipgloss.JoinVertical(lipgloss.Left, header, mainArea, footer)

	// Overlay details panel on right half if visible (resource or history)
	if m.viewMode == ui.ViewHistory && m.historyDetails.Visible() {
		detailsWidth := m.width / 2
		m.historyDetails.SetSize(detailsWidth, mainHeight)
		detailsView := m.historyDetails.View()

		// Place the details panel on the right side
		fullView = placeOverlay(m.width/2, headerHeight, detailsView, fullView)
	} else if m.details.Visible() {
		detailsWidth := m.width / 2
		m.details.SetSize(detailsWidth, mainHeight)
		detailsView := m.details.View()

		// Place the details panel on the right side
		fullView = placeOverlay(m.width/2, headerHeight, detailsView, fullView)
	}

	// Overlay help dialog if showing
	if m.showHelp {
		fullView = m.help.View()
	}

	// Overlay stack selector if showing
	if m.stackSelector.Visible() {
		fullView = m.stackSelector.View()
	}

	// Overlay workspace selector if showing
	if m.workspaceSelector.Visible() {
		fullView = m.workspaceSelector.View()
	}

	// Overlay toast notification if showing
	if m.toast.Visible() {
		toastView := m.toast.View(m.width)
		// Place toast near the bottom, above the footer
		footerHeight := 1
		toastY := m.height - footerHeight - 2
		if toastY < 0 {
			toastY = 0
		}
		fullView = placeOverlay(0, toastY, toastView, fullView)
	}

	return fullView
}

// renderFooter renders the bottom footer with keybind hints
func (m Model) renderFooter() string {
	var leftParts []string
	var rightParts []string

	// Show visual mode indicator
	if m.resourceList.VisualMode() {
		leftParts = append(leftParts, ui.LabelStyle.Render("VISUAL"))
	}

	// Show flag counts if any
	if m.resourceList.HasFlags() {
		targets := len(m.resourceList.GetTargetURNs())
		replaces := len(m.resourceList.GetReplaceURNs())
		excludes := len(m.resourceList.GetExcludeURNs())

		var flagParts []string
		if targets > 0 {
			flagParts = append(flagParts, ui.FlagTargetStyle.Render(fmt.Sprintf("T:%d", targets)))
		}
		if replaces > 0 {
			flagParts = append(flagParts, ui.FlagReplaceStyle.Render(fmt.Sprintf("R:%d", replaces)))
		}
		if excludes > 0 {
			flagParts = append(flagParts, ui.FlagExcludeStyle.Render(fmt.Sprintf("E:%d", excludes)))
		}
		if len(flagParts) > 0 {
			leftParts = append(leftParts, strings.Join(flagParts, " "))
		}
	}

	// Keybind hints on the right - context sensitive
	if m.resourceList.VisualMode() {
		rightParts = append(rightParts, ui.DimStyle.Render("T target"))
		rightParts = append(rightParts, ui.DimStyle.Render("R replace"))
		rightParts = append(rightParts, ui.DimStyle.Render("E exclude"))
		rightParts = append(rightParts, ui.DimStyle.Render("esc cancel"))
	} else {
		// Show operation hints based on view
		switch m.viewMode {
		case ui.ViewStack:
			rightParts = append(rightParts, ui.DimStyle.Render("u up"))
			rightParts = append(rightParts, ui.DimStyle.Render("r refresh"))
			rightParts = append(rightParts, ui.DimStyle.Render("d destroy"))
		case ui.ViewPreview:
			rightParts = append(rightParts, ui.DimStyle.Render("ctrl+u execute"))
			rightParts = append(rightParts, ui.DimStyle.Render("esc back"))
		case ui.ViewExecute:
			rightParts = append(rightParts, ui.DimStyle.Render("esc cancel"))
		case ui.ViewHistory:
			rightParts = append(rightParts, ui.DimStyle.Render("esc back"))
		}
		rightParts = append(rightParts, ui.DimStyle.Render("v select"))
		rightParts = append(rightParts, ui.DimStyle.Render("D details"))
		rightParts = append(rightParts, ui.DimStyle.Render("s stack"))
		rightParts = append(rightParts, ui.DimStyle.Render("w workspace"))
		rightParts = append(rightParts, ui.DimStyle.Render("h history"))
		rightParts = append(rightParts, ui.DimStyle.Render("? help"))
		rightParts = append(rightParts, ui.DimStyle.Render("q quit"))
	}

	left := joinWithSeparator(leftParts, "  ")
	right := joinWithSeparator(rightParts, "  ")

	// Calculate padding between left and right
	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	padding := m.width - leftWidth - rightWidth - 2 // -2 for margins
	if padding < 1 {
		padding = 1
	}

	return " " + left + strings.Repeat(" ", padding) + right + " "
}

// joinWithSeparator joins strings with a separator
func joinWithSeparator(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += sep + parts[i]
	}
	return result
}

// placeOverlay places an overlay string at the specified x,y position on the background
func placeOverlay(x, y int, overlay, background string) string {
	bgLines := strings.Split(background, "\n")
	overlayLines := strings.Split(overlay, "\n")

	for i, overlayLine := range overlayLines {
		bgIdx := y + i
		if bgIdx < 0 || bgIdx >= len(bgLines) {
			continue
		}

		bgLine := bgLines[bgIdx]

		// Truncate background line to x visual width and append overlay
		truncatedBg := truncateToWidth(bgLine, x)
		// Pad if needed
		currentWidth := lipgloss.Width(truncatedBg)
		if currentWidth < x {
			truncatedBg += strings.Repeat(" ", x-currentWidth)
		}

		bgLines[bgIdx] = truncatedBg + overlayLine
	}

	return strings.Join(bgLines, "\n")
}

// truncateToWidth truncates a string (which may contain ANSI codes) to the given visual width
func truncateToWidth(s string, width int) string {
	if width <= 0 {
		return ""
	}

	// Use lipgloss to handle ANSI-aware truncation
	style := lipgloss.NewStyle().MaxWidth(width)
	return style.Render(s)
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

func main() {
	flag.StringVar(&workDir, "C", "", "Run as if p5 was started in `path`")
	flag.StringVar(&workDir, "cwd", "", "Run as if p5 was started in `path`")
	flag.StringVar(&stackName, "s", "", "Select the Pulumi `stack` to use")
	flag.StringVar(&stackName, "stack", "", "Select the Pulumi `stack` to use")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: p5 [flags] [command]\n\n")
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "  up        Start with up preview\n")
		fmt.Fprintf(os.Stderr, "  refresh   Start with refresh preview\n")
		fmt.Fprintf(os.Stderr, "  destroy   Start with destroy preview\n")
		fmt.Fprintf(os.Stderr, "\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	// Get command from positional argument
	args := flag.Args()
	if len(args) > 0 {
		startView = args[0]
	} else {
		startView = "stack"
	}

	// Default to current directory if not specified
	if workDir == "" {
		var err error
		workDir, err = os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	p := tea.NewProgram(initialModel(), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
