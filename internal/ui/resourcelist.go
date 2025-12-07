package ui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ItemStatus represents execution progress
type ItemStatus int

const (
	StatusNone    ItemStatus = iota // Preview/stack view (not executing)
	StatusPending                   // Queued for execution
	StatusRunning                   // Currently executing
	StatusSuccess                   // Completed successfully
	StatusFailed                    // Failed
)

// ResourceItem is the generic representation of a resource
type ResourceItem struct {
	URN            string
	Type           string
	Name           string
	Op             ResourceOp     // OpSame for stack view, actual op for preview/exec
	Status         ItemStatus     // Execution progress
	Parent         string         // Parent URN for component hierarchy
	Sequence       int            // Event sequence number from Pulumi engine (for ordering)
	Depth          int            // Nesting depth (0 = root)
	IsLast         bool           // True if this is the last child of its parent
	CurrentOp      ResourceOp     // Current step being executed (for replace: create-replacement or delete-replaced)
	Inputs         map[string]any // Resource inputs/args from stack state
	Outputs        map[string]any // Resource outputs from stack state
	OldInputs      map[string]any // Previous inputs (for updates/deletes)
	OldOutputs     map[string]any // Previous outputs (for updates/deletes)
	Provider       string         // Provider reference string (URN::ID format)
	ProviderInputs map[string]any // Provider's configuration inputs
}

// PreviewState represents the current state of the preview (for backwards compatibility)
type PreviewState int

const (
	PreviewLoading PreviewState = iota
	PreviewRunning
	PreviewDone
	PreviewError
)

// PreviewSummary contains counts of each operation type (for backwards compatibility)
type PreviewSummary struct {
	Create  int
	Update  int
	Delete  int
	Replace int
	Total   int
}

// ResourceSummary contains counts of each operation type
type ResourceSummary struct {
	Total   int
	Same    int // Only counted when showAllOps=true
	Create  int
	Update  int
	Delete  int
	Replace int
	Refresh int
}

// ResourceList is the reusable scrollable list component
type ResourceList struct {
	ListBase // Embed common list functionality for loading/error state

	items      []ResourceItem
	visibleIdx []int                    // Indices of visible items (filtered)
	flags      map[string]ResourceFlags // Shared reference from parent

	// Cursor & scrolling
	cursor       int
	scrollOffset int
	visualMode   bool
	visualStart  int

	// Configuration
	showAllOps bool // If false, hide OpSame resources

	// Flash highlight state (for copy feedback)
	flashIdx int  // Index of item to flash (-1 = none, or specific index)
	flashAll bool // Flash all visible items
	flashing bool // Whether flash is active
}

// NewResourceList creates a new ResourceList component
func NewResourceList(flags map[string]ResourceFlags) *ResourceList {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorPrimary)
	r := &ResourceList{
		items:      make([]ResourceItem, 0),
		visibleIdx: make([]int, 0),
		flags:      flags,
		showAllOps: true,
	}
	r.SetSpinner(s)
	return r
}

// SetSize sets the dimensions for the list and ensures cursor is visible
func (r *ResourceList) SetSize(width, height int) {
	r.ListBase.SetSize(width, height)
	r.ensureCursorVisible()
}

// SetShowAllOps sets whether to show all ops or filter out OpSame
func (r *ResourceList) SetShowAllOps(show bool) {
	r.showAllOps = show
	r.rebuildVisibleIndex()
}

// SetItems replaces all items
func (r *ResourceList) SetItems(items []ResourceItem) {
	r.items = organizeItemsAsTree(items)
	r.rebuildVisibleIndex()
	r.cursor = 0
	r.scrollOffset = 0
	r.visualMode = false
	r.SetLoading(false, "")
	r.ClearError()
}

// AddItem adds a single item (for streaming)
// If an item with the same URN exists, it updates the existing item
func (r *ResourceList) AddItem(item ResourceItem) {
	r.SetLoading(false, "")

	// First, ensure parent exists (add placeholder if needed)
	if item.Parent != "" {
		r.ensureParentExists(item.Parent)
	}

	// Check if item with same URN already exists
	for i := range r.items {
		if r.items[i].URN != item.URN {
			continue
		}
		// Update existing item - keep the most significant op
		// Replace-related ops should consolidate to OpReplace
		if isReplaceOp(item.Op) {
			r.items[i].Op = OpReplace
			// Track the current step being executed (create-replacement or delete-replaced)
			r.items[i].CurrentOp = item.Op
		} else if item.Op != OpSame {
			r.items[i].Op = item.Op
			r.items[i].CurrentOp = item.Op
		}
		// Update parent if set
		if item.Parent != "" {
			r.items[i].Parent = item.Parent
		}
		// Update status if set
		if item.Status != StatusNone {
			r.items[i].Status = item.Status
		}
		// For delete-replaced ops, don't overwrite inputs/outputs since they
		// contain OLD values (we want to preserve NEW values from create-replacement)
		isDeleteReplaced := item.Op == OpDeleteReplace
		// Merge inputs if provided (but not from delete-replaced if we already have them)
		if item.Inputs != nil && (!isDeleteReplaced || r.items[i].Inputs == nil) {
			r.items[i].Inputs = item.Inputs
		}
		// Merge outputs if provided (but not from delete-replaced if we already have them)
		if item.Outputs != nil && (!isDeleteReplaced || r.items[i].Outputs == nil) {
			r.items[i].Outputs = item.Outputs
		}
		// Only set old inputs/outputs on first event for this resource
		// (subsequent events for same resource shouldn't overwrite)
		if item.OldInputs != nil && r.items[i].OldInputs == nil {
			r.items[i].OldInputs = item.OldInputs
		}
		if item.OldOutputs != nil && r.items[i].OldOutputs == nil {
			r.items[i].OldOutputs = item.OldOutputs
		}
		// Reorganize as tree and rebuild visible index
		r.items = organizeItemsAsTree(r.items)
		r.rebuildVisibleIndex()
		return
	}

	// New item - add it
	// Consolidate replace ops to single OpReplace but track current step
	if isReplaceOp(item.Op) {
		item.CurrentOp = item.Op
		item.Op = OpReplace
	} else {
		item.CurrentOp = item.Op
	}
	r.items = append(r.items, item)

	// Reorganize as tree and rebuild visible index
	r.items = organizeItemsAsTree(r.items)
	r.rebuildVisibleIndex()
}

// UpdateItemStatus updates the status of an item by URN
func (r *ResourceList) UpdateItemStatus(urn string, status ItemStatus) {
	for i := range r.items {
		if r.items[i].URN == urn {
			r.items[i].Status = status
			return
		}
	}
}

// Clear resets the list for a new view
func (r *ResourceList) Clear() {
	r.items = make([]ResourceItem, 0)
	r.visibleIdx = make([]int, 0)
	r.cursor = 0
	r.scrollOffset = 0
	r.visualMode = false
	r.ClearError()
}

// VisualMode returns whether visual selection mode is active
func (r *ResourceList) VisualMode() bool {
	return r.visualMode
}

// visibleHeight returns the number of lines available for resource items
func (r *ResourceList) visibleHeight() int {
	return CalculateVisibleHeight(r.Height(), len(r.visibleIdx), 2) // 2 = padding (1 top, 1 bottom)
}

// isScrollable returns true if there are more items than can fit without indicators
func (r *ResourceList) isScrollable() bool {
	return IsScrollable(r.Height(), len(r.visibleIdx), 2)
}

// ensureCursorVisible adjusts scroll offset to keep cursor visible
func (r *ResourceList) ensureCursorVisible() {
	r.scrollOffset = EnsureCursorVisible(r.cursor, r.scrollOffset, len(r.visibleIdx), r.visibleHeight())
}

// Update handles key events and returns any commands
func (r *ResourceList) Update(msg tea.Msg) tea.Cmd {
	// Handle ClearAllFlags even when list is empty (e.g., preview with no changes)
	if msg, ok := msg.(tea.KeyMsg); ok && key.Matches(msg, Keys.ClearAllFlags) {
		r.ClearAllFlags()
		r.visualMode = false
		return nil
	}

	if !r.IsReady() || len(r.visibleIdx) == 0 {
		return nil
	}

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}

	if r.handleNavigationKeys(keyMsg) {
		return nil
	}
	if r.handleSelectionKeys(keyMsg) {
		return nil
	}
	return r.handleCopyKeys(keyMsg)
}

func (r *ResourceList) handleNavigationKeys(keyMsg tea.KeyMsg) bool {
	switch {
	case key.Matches(keyMsg, Keys.Up):
		r.moveCursor(-1)
	case key.Matches(keyMsg, Keys.Down):
		r.moveCursor(1)
	case key.Matches(keyMsg, Keys.PageUp):
		r.moveCursor(-r.visibleHeight())
	case key.Matches(keyMsg, Keys.PageDown):
		r.moveCursor(r.visibleHeight())
	case key.Matches(keyMsg, Keys.Home):
		r.cursor = 0
		r.ensureCursorVisible()
	case key.Matches(keyMsg, Keys.End):
		r.cursor = len(r.visibleIdx) - 1
		r.ensureCursorVisible()
	default:
		return false
	}
	return true
}

func (r *ResourceList) handleSelectionKeys(keyMsg tea.KeyMsg) bool {
	switch {
	case key.Matches(keyMsg, Keys.VisualMode):
		if !r.visualMode {
			r.visualMode = true
			r.visualStart = r.cursor
		}
	case key.Matches(keyMsg, Keys.Escape):
		r.visualMode = false
	case key.Matches(keyMsg, Keys.ToggleTarget):
		r.toggleFlag("target")
	case key.Matches(keyMsg, Keys.ToggleReplace):
		r.toggleFlag("replace")
	case key.Matches(keyMsg, Keys.ToggleExclude):
		r.toggleFlag("exclude")
	case key.Matches(keyMsg, Keys.ClearFlags):
		r.clearFlags()
	case key.Matches(keyMsg, Keys.ClearAllFlags):
		r.ClearAllFlags()
		r.visualMode = false
	default:
		return false
	}
	return true
}

func (r *ResourceList) handleCopyKeys(keyMsg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(keyMsg, Keys.CopyResource):
		return r.CopyResourceJSON()
	case key.Matches(keyMsg, Keys.CopyAllResources):
		return r.CopyAllResourcesJSON()
	}
	return nil
}

// moveCursor moves the cursor by delta, clamping to valid range
func (r *ResourceList) moveCursor(delta int) {
	r.cursor = MoveCursor(r.cursor, delta, len(r.visibleIdx))
	r.ensureCursorVisible()
}

// getSelectedIndices returns the indices of selected items (cursor or visual range)
func (r *ResourceList) getSelectedIndices() []int {
	if !r.visualMode {
		return []int{r.cursor}
	}

	start, end := r.visualStart, r.cursor
	if start > end {
		start, end = end, start
	}

	indices := make([]int, 0, end-start+1)
	for i := start; i <= end; i++ {
		indices = append(indices, i)
	}
	return indices
}

// Summary returns the current summary
func (r *ResourceList) Summary() ResourceSummary {
	summary := ResourceSummary{}
	for i := range r.items {
		switch r.items[i].Op {
		case OpSame:
			summary.Same++
		case OpCreate:
			summary.Create++
		case OpUpdate:
			summary.Update++
		case OpDelete:
			summary.Delete++
		case OpReplace, OpCreateReplace, OpDeleteReplace:
			summary.Replace++
		case OpRefresh:
			summary.Refresh++
		}
		summary.Total++
	}
	return summary
}

// ScrollPercent returns the current scroll percentage (0-100)
func (r *ResourceList) ScrollPercent() float64 {
	if len(r.visibleIdx) <= r.visibleHeight() {
		return 100
	}
	maxScroll := len(r.visibleIdx) - r.visibleHeight()
	return float64(r.scrollOffset) / float64(maxScroll) * 100
}

// AtTop returns true if scrolled to top
func (r *ResourceList) AtTop() bool {
	return r.scrollOffset == 0
}

// AtBottom returns true if scrolled to bottom
func (r *ResourceList) AtBottom() bool {
	return r.scrollOffset >= len(r.visibleIdx)-r.visibleHeight()
}

// TotalLines returns the total number of visible lines
func (r *ResourceList) TotalLines() int {
	return len(r.visibleIdx)
}

// VisibleLines returns the number of lines that fit on screen
func (r *ResourceList) VisibleLines() int {
	return r.visibleHeight()
}

// SelectedItem returns a pointer to the currently selected item, or nil if none
func (r *ResourceList) SelectedItem() *ResourceItem {
	if len(r.visibleIdx) == 0 || r.cursor < 0 || r.cursor >= len(r.visibleIdx) {
		return nil
	}
	itemIdx := r.visibleIdx[r.cursor]
	if itemIdx < 0 || itemIdx >= len(r.items) {
		return nil
	}
	return &r.items[itemIdx]
}
