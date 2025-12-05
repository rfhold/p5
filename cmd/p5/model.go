package main

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/rfhold/p5/internal/pulumi"
	"github.com/rfhold/p5/internal/ui"
)

// InitState tracks progress through the initialization state machine.
// This makes the initialization flow explicit and traceable.
type InitState int

const (
	// InitCheckingWorkspace - checking if we're in a valid Pulumi workspace
	InitCheckingWorkspace InitState = iota
	// InitLoadingPlugins - loading and authenticating plugins
	InitLoadingPlugins
	// InitLoadingStacks - fetching available stacks
	InitLoadingStacks
	// InitSelectingStack - user must select or create a stack
	InitSelectingStack
	// InitLoadingResources - loading stack resources or starting preview
	InitLoadingResources
	// InitComplete - initialization is done, app is ready
	InitComplete
)

// String returns a human-readable name for the init state
func (s InitState) String() string {
	switch s {
	case InitCheckingWorkspace:
		return "CheckingWorkspace"
	case InitLoadingPlugins:
		return "LoadingPlugins"
	case InitLoadingStacks:
		return "LoadingStacks"
	case InitSelectingStack:
		return "SelectingStack"
	case InitLoadingResources:
		return "LoadingResources"
	case InitComplete:
		return "Complete"
	default:
		return "Unknown"
	}
}

// OperationState tracks the lifecycle of preview and execute operations.
// This makes operation handling explicit and easier to reason about.
type OperationState int

const (
	// OpIdle - no operation is running
	OpIdle OperationState = iota
	// OpStarting - operation is being initialized
	OpStarting
	// OpRunning - operation is in progress, receiving events
	OpRunning
	// OpCancelling - cancel was requested, waiting for operation to stop
	OpCancelling
	// OpComplete - operation finished successfully
	OpComplete
	// OpError - operation finished with an error
	OpError
)

// String returns a human-readable name for the operation state
func (s OperationState) String() string {
	switch s {
	case OpIdle:
		return "Idle"
	case OpStarting:
		return "Starting"
	case OpRunning:
		return "Running"
	case OpCancelling:
		return "Cancelling"
	case OpComplete:
		return "Complete"
	case OpError:
		return "Error"
	default:
		return "Unknown"
	}
}

// IsActive returns true if the operation is currently running or starting
func (s OperationState) IsActive() bool {
	return s == OpStarting || s == OpRunning || s == OpCancelling
}

// AppContext holds application-level configuration that was previously stored in globals.
// This improves testability and makes data flow explicit.
type AppContext struct {
	Cwd       string // Current working directory (where app was launched from)
	WorkDir   string // Working directory (Pulumi project root)
	StackName string // Currently selected stack name
	StartView string // Initial view mode ("stack", "up", "refresh", "destroy")
}

// Model is the main application model.
// It coordinates between pure application state (AppState), UI state (UIState),
// and async operations. This separation enables easier testing.
type Model struct {
	// App context (configuration, replaces globals)
	ctx AppContext

	// Injected dependencies (enables testing)
	deps *Dependencies

	// Pure application state (testable without UI)
	state *AppState

	// UI component state (layout, focus, components)
	ui *UIState

	// Control flags
	quitting bool

	// Event channels for async operations (receive-only from StackOperator)
	previewCh   <-chan pulumi.PreviewEvent
	operationCh <-chan pulumi.OperationEvent

	// Operation context for cancellation
	operationCtx    context.Context
	operationCancel context.CancelFunc
}

func initialModel(ctx AppContext, deps *Dependencies) Model {
	// Create shared state
	state := NewAppState()

	// Create UI state with shared flags reference
	uiState := NewUIState(state.Flags)

	m := Model{
		ctx:   ctx,
		deps:  deps,
		state: state,
		ui:    uiState,
	}

	// Set initial view based on command argument
	switch ctx.StartView {
	case "up":
		m.ui.ViewMode = ui.ViewPreview
		m.state.Operation = pulumi.OperationUp
		m.ui.ResourceList.SetShowAllOps(false)
	case "refresh":
		m.ui.ViewMode = ui.ViewPreview
		m.state.Operation = pulumi.OperationRefresh
		m.ui.ResourceList.SetShowAllOps(false)
	case "destroy":
		m.ui.ViewMode = ui.ViewPreview
		m.state.Operation = pulumi.OperationDestroy
		m.ui.ResourceList.SetShowAllOps(false)
	}

	m.ui.Header.SetViewMode(m.ui.ViewMode)
	m.ui.Header.SetOperation(m.state.Operation)

	return m
}

// Init starts the initial data fetch
func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		m.ui.Header.Spinner().Tick,
		m.ui.ResourceList.Spinner().Tick,
		m.ui.HistoryList.Spinner().Tick,
	}

	// First check if we're in a valid Pulumi workspace
	cmds = append(cmds, m.checkWorkspace())

	return tea.Batch(cmds...)
}
