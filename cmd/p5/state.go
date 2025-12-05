package main

import (
	"github.com/rfhold/p5/internal/pulumi"
	"github.com/rfhold/p5/internal/ui"
)

// AppState holds pure application state (no UI components).
// This can be serialized, compared, and tested independently of UI concerns.
// The separation enables easier unit testing of business logic.
type AppState struct {
	// Initialization state machine
	InitState InitState

	// Operation state machine
	OpState   OperationState
	Operation pulumi.OperationType

	// Pending operation confirmation (operation awaiting user confirm)
	PendingOperation *pulumi.OperationType

	// Resource flags (persists across all views)
	// Maps URN to flags for each resource
	Flags map[string]ui.ResourceFlags

	// Error state
	Err error
}

// NewAppState creates initial application state with default values
func NewAppState() *AppState {
	return &AppState{
		InitState: InitCheckingWorkspace,
		OpState:   OpIdle,
		Flags:     make(map[string]ui.ResourceFlags),
	}
}
