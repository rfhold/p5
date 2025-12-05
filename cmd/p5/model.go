package main

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/rfhold/p5/internal/plugins"
	"github.com/rfhold/p5/internal/pulumi"
	"github.com/rfhold/p5/internal/ui"
)

// Model is the main application model
type Model struct {
	// Shared persistent state
	flags map[string]ui.ResourceFlags // Persists across all views

	// Current view state
	viewMode  ui.ViewMode
	operation pulumi.OperationType

	// Pending operation confirmation
	pendingOperation *pulumi.OperationType // Operation awaiting confirmation

	// Components
	header            ui.Header
	resourceList      *ui.ResourceList
	historyList       *ui.HistoryList
	help              *ui.HelpDialog
	details           *ui.DetailPanel
	historyDetails    *ui.HistoryDetailPanel
	stackSelector     *ui.StackSelector
	workspaceSelector *ui.WorkspaceSelector
	importModal       *ui.ImportModal
	confirmModal      *ui.ConfirmModal
	errorModal        *ui.ErrorModal
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
		importModal:       ui.NewImportModal(),
		confirmModal:      ui.NewConfirmModal(),
		errorModal:        ui.NewErrorModal(),
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
