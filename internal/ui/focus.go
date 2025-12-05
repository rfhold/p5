package ui

// FocusLayer represents what component currently owns keyboard input
type FocusLayer int

const (
	FocusMain              FocusLayer = iota // Normal app interaction (resource list, history list)
	FocusDetailsPanel                        // Details panel is open and capturing scroll keys
	FocusHelp                                // Help dialog open
	FocusStackSelector                       // Stack selector modal
	FocusWorkspaceSelector                   // Workspace selector modal
	FocusImportModal                         // Import modal
	FocusStackInitModal                      // Stack creation modal
	FocusConfirmModal                        // Confirmation dialog
	FocusErrorModal                          // Error dialog (highest priority)
)

// String returns a human-readable name for the focus layer
func (f FocusLayer) String() string {
	switch f {
	case FocusMain:
		return "Main"
	case FocusDetailsPanel:
		return "DetailsPanel"
	case FocusHelp:
		return "Help"
	case FocusStackSelector:
		return "StackSelector"
	case FocusWorkspaceSelector:
		return "WorkspaceSelector"
	case FocusImportModal:
		return "ImportModal"
	case FocusStackInitModal:
		return "StackInitModal"
	case FocusConfirmModal:
		return "ConfirmModal"
	case FocusErrorModal:
		return "ErrorModal"
	default:
		return "Unknown"
	}
}

// FocusStack manages the stack of focus layers.
// The stack always has at least one element (FocusMain at the bottom).
type FocusStack struct {
	stack []FocusLayer
}

// NewFocusStack creates a new focus stack with FocusMain as the base layer
func NewFocusStack() FocusStack {
	return FocusStack{
		stack: []FocusLayer{FocusMain},
	}
}

// Push adds a new focus layer to the top of the stack.
// Does nothing if the layer is already at the top (prevents duplicate pushes).
func (f *FocusStack) Push(layer FocusLayer) {
	if len(f.stack) > 0 && f.stack[len(f.stack)-1] == layer {
		return // Already at top, no-op
	}
	f.stack = append(f.stack, layer)
}

// Pop removes and returns the top focus layer.
// Returns FocusMain if only the base layer remains (never pops below FocusMain).
func (f *FocusStack) Pop() FocusLayer {
	if len(f.stack) <= 1 {
		return FocusMain
	}
	top := f.stack[len(f.stack)-1]
	f.stack = f.stack[:len(f.stack)-1]
	return top
}

// Current returns the current focus layer (top of stack)
func (f *FocusStack) Current() FocusLayer {
	if len(f.stack) == 0 {
		return FocusMain
	}
	return f.stack[len(f.stack)-1]
}

// Clear resets the stack to just the base FocusMain layer
func (f *FocusStack) Clear() {
	f.stack = []FocusLayer{FocusMain}
}

// Has returns true if the given layer is anywhere in the stack
func (f *FocusStack) Has(layer FocusLayer) bool {
	for _, l := range f.stack {
		if l == layer {
			return true
		}
	}
	return false
}

// Remove removes a specific layer from anywhere in the stack.
// This is useful when a modal is hidden by some external event.
func (f *FocusStack) Remove(layer FocusLayer) {
	if layer == FocusMain {
		return // Never remove the base layer
	}
	newStack := make([]FocusLayer, 0, len(f.stack))
	for _, l := range f.stack {
		if l != layer {
			newStack = append(newStack, l)
		}
	}
	if len(newStack) == 0 {
		newStack = []FocusLayer{FocusMain}
	}
	f.stack = newStack
}

// Depth returns the number of layers in the stack (including FocusMain)
func (f *FocusStack) Depth() int {
	return len(f.stack)
}
