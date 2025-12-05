package main

import "github.com/rfhold/p5/internal/ui"

// UIState holds all UI component state.
// This groups UI-specific concerns (layout, focus, components) separately
// from pure application state, enabling cleaner separation of concerns.
type UIState struct {
	// Layout dimensions
	Width  int
	Height int

	// Focus management
	Focus ui.FocusStack

	// Current view mode (stack, preview, execute, history)
	ViewMode ui.ViewMode

	// UI Components
	Header            ui.Header
	ResourceList      *ui.ResourceList
	HistoryList       *ui.HistoryList
	Help              *ui.HelpDialog
	Details           *ui.DetailPanel
	HistoryDetails    *ui.HistoryDetailPanel
	StackSelector     *ui.StackSelector
	WorkspaceSelector *ui.WorkspaceSelector
	ImportModal       *ui.ImportModal
	ConfirmModal      *ui.ConfirmModal
	ErrorModal        *ui.ErrorModal
	StackInitModal    *ui.StackInitModal
	Toast             *ui.Toast
}

// NewUIState creates a new UIState with initialized components.
// The flags parameter is shared with AppState for resource flag persistence.
func NewUIState(flags map[string]ui.ResourceFlags) *UIState {
	return &UIState{
		Focus:             ui.NewFocusStack(),
		ViewMode:          ui.ViewStack,
		Header:            ui.NewHeader(),
		ResourceList:      ui.NewResourceList(flags),
		HistoryList:       ui.NewHistoryList(),
		Help:              ui.NewHelpDialog(),
		Details:           ui.NewDetailPanel(),
		HistoryDetails:    ui.NewHistoryDetailPanel(),
		StackSelector:     ui.NewStackSelector(),
		WorkspaceSelector: ui.NewWorkspaceSelector(),
		ImportModal:       ui.NewImportModal(),
		ConfirmModal:      ui.NewConfirmModal(),
		ErrorModal:        ui.NewErrorModal(),
		StackInitModal:    ui.NewStackInitModal(),
		Toast:             ui.NewToast(),
	}
}
