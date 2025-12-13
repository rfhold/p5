package main

import (
	"github.com/rfhold/p5/internal/pulumi"
	"github.com/rfhold/p5/internal/ui"
)

// PendingOperation represents an operation queued while the app is busy
type PendingOperation struct {
	Type string // Operation type: "preview", "load_resources", "init_load_resources", etc.
	Data any    // Optional data needed for the operation
}

// PendingProtectAction represents a protect/unprotect action awaiting confirmation
type PendingProtectAction struct {
	URN     string
	Name    string
	Protect bool // true = protect, false = unprotect
}

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

	// Pending protect action (awaiting confirmation)
	PendingProtectAction *PendingProtectAction

	// Resource flags (persists across all views)
	// Maps URN to flags for each resource
	Flags map[string]ui.ResourceFlags

	// Error state
	Err error

	// BusyLock is the reason we're busy (empty string means not busy)
	BusyLock string
	// PendingOps are operations queued to run when the busy lock is released
	PendingOps []PendingOperation
}

// NewAppState creates initial application state with default values
func NewAppState() *AppState {
	return &AppState{
		InitState: InitCheckingWorkspace,
		OpState:   OpIdle,
		Flags:     make(map[string]ui.ResourceFlags),
	}
}

// SetBusy sets the busy lock with a reason
func (s *AppState) SetBusy(reason string) {
	s.BusyLock = reason
}

// ClearBusy clears the busy lock and returns any pending operations
func (s *AppState) ClearBusy() []PendingOperation {
	ops := s.PendingOps
	s.BusyLock = ""
	s.PendingOps = nil
	return ops
}

// IsBusy returns true if the app is busy
func (s *AppState) IsBusy() bool {
	return s.BusyLock != ""
}

// QueueOperation adds an operation to the pending queue
func (s *AppState) QueueOperation(op PendingOperation) {
	s.PendingOps = append(s.PendingOps, op)
}
