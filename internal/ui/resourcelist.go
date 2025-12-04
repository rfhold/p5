package ui

import (
	"encoding/json"
	"fmt"
	"strings"

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
	URN        string
	Type       string
	Name       string
	Op         ResourceOp             // OpSame for stack view, actual op for preview/exec
	Status     ItemStatus             // Execution progress
	Parent     string                 // Parent URN for component hierarchy
	Depth      int                    // Nesting depth (0 = root)
	IsLast     bool                   // True if this is the last child of its parent
	CurrentOp  ResourceOp             // Current step being executed (for replace: create-replacement or delete-replaced)
	Inputs     map[string]interface{} // Resource inputs/args from stack state
	Outputs    map[string]interface{} // Resource outputs from stack state
	OldInputs  map[string]interface{} // Previous inputs (for updates/deletes)
	OldOutputs map[string]interface{} // Previous outputs (for updates/deletes)
}

// ResourceFlags tracks selection flags for a resource
type ResourceFlags struct {
	Target  bool // --target flag for update
	Replace bool // --replace flag for update
	Exclude bool // exclude from update (mutually exclusive with Target/Replace)
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
	width      int
	height     int
	ready      bool

	// Loading state (for spinner during data fetch)
	loading    bool
	loadingMsg string
	spinner    spinner.Model

	// Error state
	err error

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
	return &ResourceList{
		items:      make([]ResourceItem, 0),
		visibleIdx: make([]int, 0),
		flags:      flags,
		spinner:    s,
		showAllOps: true,
	}
}

// Spinner returns the spinner model for tick updates
func (r *ResourceList) Spinner() spinner.Model {
	return r.spinner
}

// SetSpinner updates the spinner model
func (r *ResourceList) SetSpinner(s spinner.Model) {
	r.spinner = s
}

// SetSize sets the dimensions for the list
func (r *ResourceList) SetSize(width, height int) {
	r.width = width
	r.height = height
	r.ready = true
	r.ensureCursorVisible()
}

// SetShowAllOps sets whether to show all ops or filter out OpSame
func (r *ResourceList) SetShowAllOps(show bool) {
	r.showAllOps = show
	r.rebuildVisibleIndex()
}

// SetLoading sets the loading state
func (r *ResourceList) SetLoading(loading bool, msg string) {
	r.loading = loading
	r.loadingMsg = msg
}

// SetError sets an error state
func (r *ResourceList) SetError(err error) {
	r.err = err
	r.loading = false
}

// IsLoading returns true if loading
func (r *ResourceList) IsLoading() bool {
	return r.loading
}

// SetItems replaces all items
func (r *ResourceList) SetItems(items []ResourceItem) {
	r.items = organizeItemsAsTree(items)
	r.rebuildVisibleIndex()
	r.cursor = 0
	r.scrollOffset = 0
	r.visualMode = false
	r.loading = false
	r.err = nil
}

// organizeItemsAsTree sorts items into tree order (parent followed by children)
// and sets Depth and IsLast for each item
func organizeItemsAsTree(items []ResourceItem) []ResourceItem {
	if len(items) == 0 {
		return items
	}

	// Build parent -> children map
	childrenOf := make(map[string][]int) // parent URN -> indices of children
	rootIndices := make([]int, 0)

	for i, item := range items {
		if item.Parent == "" {
			rootIndices = append(rootIndices, i)
		} else {
			childrenOf[item.Parent] = append(childrenOf[item.Parent], i)
		}
	}

	// Flatten tree into result slice with depth/isLast info
	result := make([]ResourceItem, 0, len(items))

	var addItem func(idx int, depth int, isLast bool)
	addItem = func(idx int, depth int, isLast bool) {
		item := items[idx]
		item.Depth = depth
		item.IsLast = isLast
		result = append(result, item)

		children := childrenOf[item.URN]
		for i, childIdx := range children {
			isLastChild := i == len(children)-1
			addItem(childIdx, depth+1, isLastChild)
		}
	}

	for i, rootIdx := range rootIndices {
		isLastRoot := i == len(rootIndices)-1
		addItem(rootIdx, 0, isLastRoot)
	}

	return result
}

// AddItem adds a single item (for streaming)
// If an item with the same URN exists, it updates the existing item
func (r *ResourceList) AddItem(item ResourceItem) {
	r.loading = false

	// First, ensure parent exists (add placeholder if needed)
	if item.Parent != "" {
		r.ensureParentExists(item.Parent)
	}

	// Check if item with same URN already exists
	for i, existing := range r.items {
		if existing.URN == item.URN {
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
			if item.Inputs != nil && !(isDeleteReplaced && r.items[i].Inputs != nil) {
				r.items[i].Inputs = item.Inputs
			}
			// Merge outputs if provided (but not from delete-replaced if we already have them)
			if item.Outputs != nil && !(isDeleteReplaced && r.items[i].Outputs != nil) {
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

// ensureParentExists adds a placeholder parent item if it doesn't exist
// This recursively ensures all ancestors exist
func (r *ResourceList) ensureParentExists(parentURN string) {
	if parentURN == "" {
		return
	}

	// Check if parent already exists
	for _, item := range r.items {
		if item.URN == parentURN {
			return // Parent exists
		}
	}

	// Parent doesn't exist - create a placeholder with OpSame
	// Extract type and name from URN
	parentType := extractResourceType(parentURN)
	parentName := extractResourceName(parentURN)

	// Add the parent placeholder
	// Note: We don't know the grandparent URN from the URN alone,
	// but when the parent's event arrives (if ever), it will update with correct parent
	r.items = append(r.items, ResourceItem{
		URN:    parentURN,
		Type:   parentType,
		Name:   parentName,
		Op:     OpSame,
		Status: StatusNone,
		Parent: "", // Will be updated if parent's event arrives later
	})
}

// extractResourceType gets the resource type from a URN
// URN format: urn:pulumi:stack::project::type::name
func extractResourceType(urn string) string {
	parts := splitURN(urn)
	if len(parts) >= 4 {
		return parts[3]
	}
	return urn
}

// extractResourceName gets the resource name from a URN
// URN format: urn:pulumi:stack::project::type::name
func extractResourceName(urn string) string {
	parts := splitURN(urn)
	if len(parts) >= 5 {
		return parts[4]
	}
	// Fallback: find last :: and return everything after it
	for i := len(urn) - 1; i >= 0; i-- {
		if i > 0 && urn[i-1:i+1] == "::" {
			return urn[i+1:]
		}
	}
	return urn
}

// splitURN splits a URN by :: delimiter
func splitURN(urn string) []string {
	var parts []string
	current := ""
	for i := 0; i < len(urn); i++ {
		if i < len(urn)-1 && urn[i] == ':' && urn[i+1] == ':' {
			parts = append(parts, current)
			current = ""
			i++ // Skip the second ':'
		} else {
			current += string(urn[i])
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

// isReplaceOp returns true for all replace-related operations
func isReplaceOp(op ResourceOp) bool {
	return op == OpReplace || op == OpCreateReplace || op == OpDeleteReplace
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
	r.err = nil
}

// rebuildVisibleIndex applies filters to build the visible index
func (r *ResourceList) rebuildVisibleIndex() {
	r.visibleIdx = make([]int, 0, len(r.items))

	if r.showAllOps {
		// Show everything
		for i := range r.items {
			r.visibleIdx = append(r.visibleIdx, i)
		}
	} else {
		// Build set of URNs that have changes (not OpSame)
		// and URNs that are ancestors of changed items
		visibleURNs := make(map[string]bool)

		// First pass: mark all items with changes
		for _, item := range r.items {
			if item.Op != OpSame {
				visibleURNs[item.URN] = true
			}
		}

		// Second pass: mark all ancestors of changed items
		for _, item := range r.items {
			if item.Op != OpSame && item.Parent != "" {
				r.markAncestorsVisible(item.Parent, visibleURNs)
			}
		}

		// Third pass: add visible items in order
		for i, item := range r.items {
			if visibleURNs[item.URN] {
				r.visibleIdx = append(r.visibleIdx, i)
			}
		}
	}

	// Clamp cursor
	if r.cursor >= len(r.visibleIdx) {
		r.cursor = len(r.visibleIdx) - 1
		if r.cursor < 0 {
			r.cursor = 0
		}
	}
	r.ensureCursorVisible()
}

// markAncestorsVisible recursively marks all ancestors as visible
func (r *ResourceList) markAncestorsVisible(parentURN string, visibleURNs map[string]bool) {
	if parentURN == "" {
		return
	}
	if visibleURNs[parentURN] {
		return // Already marked
	}
	visibleURNs[parentURN] = true

	// Find the parent item and recurse to its parent
	for _, item := range r.items {
		if item.URN == parentURN {
			if item.Parent != "" {
				r.markAncestorsVisible(item.Parent, visibleURNs)
			}
			return
		}
	}
}

// VisualMode returns whether visual selection mode is active
func (r *ResourceList) VisualMode() bool {
	return r.visualMode
}

// visibleHeight returns the number of lines available for resource items
func (r *ResourceList) visibleHeight() int {
	// Account for padding (1 top, 1 bottom)
	h := r.height - 2

	// If content is scrollable, reserve 2 lines for scroll indicators
	if r.isScrollable() {
		h -= 2
	}

	if h < 1 {
		h = 1
	}
	return h
}

// isScrollable returns true if there are more items than can fit without indicators
func (r *ResourceList) isScrollable() bool {
	// Base height without scroll indicators
	baseHeight := r.height - 2
	if baseHeight < 1 {
		baseHeight = 1
	}
	return len(r.visibleIdx) > baseHeight
}

// ensureCursorVisible adjusts scroll offset to keep cursor visible
func (r *ResourceList) ensureCursorVisible() {
	if len(r.visibleIdx) == 0 {
		return
	}

	visible := r.visibleHeight()

	// Scroll up if cursor is above visible area
	if r.cursor < r.scrollOffset {
		r.scrollOffset = r.cursor
	}

	// Scroll down if cursor is below visible area
	if r.cursor >= r.scrollOffset+visible {
		r.scrollOffset = r.cursor - visible + 1
	}

	// Clamp scroll offset
	maxScroll := len(r.visibleIdx) - visible
	if maxScroll < 0 {
		maxScroll = 0
	}
	if r.scrollOffset > maxScroll {
		r.scrollOffset = maxScroll
	}
	if r.scrollOffset < 0 {
		r.scrollOffset = 0
	}
}

// Update handles key events and returns any commands
func (r *ResourceList) Update(msg tea.Msg) tea.Cmd {
	if !r.ready || len(r.visibleIdx) == 0 {
		return nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		// Navigation
		case key.Matches(msg, Keys.Up):
			r.moveCursor(-1)
		case key.Matches(msg, Keys.Down):
			r.moveCursor(1)
		case key.Matches(msg, Keys.PageUp):
			r.moveCursor(-r.visibleHeight())
		case key.Matches(msg, Keys.PageDown):
			r.moveCursor(r.visibleHeight())
		case key.Matches(msg, Keys.Home):
			r.cursor = 0
			r.ensureCursorVisible()
		case key.Matches(msg, Keys.End):
			r.cursor = len(r.visibleIdx) - 1
			r.ensureCursorVisible()

		// Visual mode
		case key.Matches(msg, Keys.VisualMode):
			if !r.visualMode {
				r.visualMode = true
				r.visualStart = r.cursor
			}
		case key.Matches(msg, Keys.Escape):
			r.visualMode = false

		// Flag toggles
		case key.Matches(msg, Keys.ToggleTarget):
			r.toggleFlag("target")
		case key.Matches(msg, Keys.ToggleReplace):
			r.toggleFlag("replace")
		case key.Matches(msg, Keys.ToggleExclude):
			r.toggleFlag("exclude")
		case key.Matches(msg, Keys.ClearFlags):
			r.clearFlags()
		case key.Matches(msg, Keys.ClearAllFlags):
			r.ClearAllFlags()
			r.visualMode = false

		// Copy resource(s) as JSON
		case key.Matches(msg, Keys.CopyResource):
			return r.CopyResourceJSON()
		case key.Matches(msg, Keys.CopyAllResources):
			return r.CopyAllResourcesJSON()
		}
	}

	return nil
}

// moveCursor moves the cursor by delta, clamping to valid range
func (r *ResourceList) moveCursor(delta int) {
	r.cursor += delta
	if r.cursor < 0 {
		r.cursor = 0
	}
	if r.cursor >= len(r.visibleIdx) {
		r.cursor = len(r.visibleIdx) - 1
	}
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

// toggleFlag toggles the specified flag for selected resources
func (r *ResourceList) toggleFlag(flagType string) {
	indices := r.getSelectedIndices()

	for _, idx := range indices {
		if idx < 0 || idx >= len(r.visibleIdx) {
			continue
		}
		item := r.items[r.visibleIdx[idx]]
		urn := item.URN

		flags := r.flags[urn]

		switch flagType {
		case "target":
			// Clear exclude if setting target
			if !flags.Target {
				flags.Exclude = false
			}
			flags.Target = !flags.Target

		case "replace":
			// Replace is allowed on any resource (from stack view or update ops)
			// Skip the stack resource itself since it can't be replaced
			if item.Type == "pulumi:pulumi:Stack" {
				continue
			}
			// Clear exclude if setting replace
			if !flags.Replace {
				flags.Exclude = false
			}
			flags.Replace = !flags.Replace

		case "exclude":
			// Clear target/replace if setting exclude
			if !flags.Exclude {
				flags.Target = false
				flags.Replace = false
			}
			flags.Exclude = !flags.Exclude
		}

		r.flags[urn] = flags
	}

	// Exit visual mode after toggling
	r.visualMode = false
}

// clearFlags clears all flags for selected resources
func (r *ResourceList) clearFlags() {
	indices := r.getSelectedIndices()

	for _, idx := range indices {
		if idx < 0 || idx >= len(r.visibleIdx) {
			continue
		}
		urn := r.items[r.visibleIdx[idx]].URN
		delete(r.flags, urn)
	}

	// Exit visual mode after clearing
	r.visualMode = false
}

// GetTargetURNs returns URNs flagged for --target
func (r *ResourceList) GetTargetURNs() []string {
	var urns []string
	for urn, flags := range r.flags {
		if flags.Target {
			urns = append(urns, urn)
		}
	}
	return urns
}

// GetReplaceURNs returns URNs flagged for --replace
func (r *ResourceList) GetReplaceURNs() []string {
	var urns []string
	for urn, flags := range r.flags {
		if flags.Replace {
			urns = append(urns, urn)
		}
	}
	return urns
}

// GetExcludeURNs returns URNs flagged for exclusion
func (r *ResourceList) GetExcludeURNs() []string {
	var urns []string
	for urn, flags := range r.flags {
		if flags.Exclude {
			urns = append(urns, urn)
		}
	}
	return urns
}

// HasFlags returns true if any resources have flags set
func (r *ResourceList) HasFlags() bool {
	return len(r.flags) > 0
}

// ClearAllFlags clears all flags
func (r *ResourceList) ClearAllFlags() {
	for k := range r.flags {
		delete(r.flags, k)
	}
}

// Summary returns the current summary
func (r *ResourceList) Summary() ResourceSummary {
	summary := ResourceSummary{}
	for _, item := range r.items {
		switch item.Op {
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

// ResourceJSON is the JSON structure for copying a resource
// Matches Pulumi's native output format as closely as possible
type ResourceJSON struct {
	URN     string                 `json:"urn"`
	Type    string                 `json:"type"`
	Inputs  map[string]interface{} `json:"inputs,omitempty"`
	Outputs map[string]interface{} `json:"outputs,omitempty"`
}

// CopyResourceJSON copies the selected resource as JSON to the clipboard
func (r *ResourceList) CopyResourceJSON() tea.Cmd {
	item := r.SelectedItem()
	if item == nil {
		return nil
	}

	// Start flash on current cursor position
	r.flashIdx = r.cursor
	r.flashing = true

	resource := ResourceJSON{
		URN:     item.URN,
		Type:    item.Type,
		Inputs:  item.Inputs,
		Outputs: item.Outputs,
	}

	jsonBytes, err := json.MarshalIndent(resource, "", "  ")
	if err != nil {
		return nil
	}

	return CopyToClipboardWithCountCmd(string(jsonBytes), 1)
}

// CopyAllResourcesJSON copies all visible resources as JSON array to the clipboard
func (r *ResourceList) CopyAllResourcesJSON() tea.Cmd {
	if len(r.visibleIdx) == 0 {
		return nil
	}

	// Flash all visible items
	r.flashAll = true
	r.flashing = true

	// Build JSON array of all visible resources
	resources := make([]ResourceJSON, 0, len(r.visibleIdx))
	for _, idx := range r.visibleIdx {
		item := &r.items[idx]
		resources = append(resources, ResourceJSON{
			URN:     item.URN,
			Type:    item.Type,
			Inputs:  item.Inputs,
			Outputs: item.Outputs,
		})
	}

	jsonBytes, err := json.MarshalIndent(resources, "", "  ")
	if err != nil {
		return nil
	}

	return CopyToClipboardWithCountCmd(string(jsonBytes), len(resources))
}

// VisibleCount returns the number of visible resources
func (r *ResourceList) VisibleCount() int {
	return len(r.visibleIdx)
}

// FlashClearMsg is sent to clear the flash highlight
type FlashClearMsg struct{}

// ClearFlash clears the flash highlight
func (r *ResourceList) ClearFlash() {
	r.flashing = false
	r.flashIdx = -1
	r.flashAll = false
}

// View renders the resource list component
func (r *ResourceList) View() string {
	if r.loading {
		return r.renderLoading()
	}
	if r.err != nil {
		return r.renderError()
	}
	return r.renderItems()
}

func (r *ResourceList) renderLoading() string {
	msg := r.loadingMsg
	if msg == "" {
		msg = "Loading..."
	}
	content := fmt.Sprintf("%s %s", r.spinner.View(), msg)
	return lipgloss.Place(r.width, r.height, lipgloss.Center, lipgloss.Center, content)
}

func (r *ResourceList) renderError() string {
	paddedStyle := lipgloss.NewStyle().Padding(1, 2)
	errMsg := ErrorStyle.Render(fmt.Sprintf("Error: %v", r.err))
	return paddedStyle.Render(errMsg)
}

func (r *ResourceList) renderItems() string {
	if len(r.visibleIdx) == 0 {
		msg := DimStyle.Render("No resources")
		return lipgloss.Place(r.width, r.height, lipgloss.Center, lipgloss.Center, msg)
	}

	var b strings.Builder
	visible := r.visibleHeight()
	endIdx := r.scrollOffset + visible
	if endIdx > len(r.visibleIdx) {
		endIdx = len(r.visibleIdx)
	}

	// Check if content is scrollable at all
	scrollable := r.isScrollable()
	canScrollUp := !r.AtTop()
	canScrollDown := !r.AtBottom()

	// Always reserve space for up arrow when scrollable to prevent layout jumps
	if scrollable {
		if canScrollUp {
			b.WriteString(ScrollIndicatorStyle.Render("  ▲"))
		} else {
			b.WriteString("   ") // Empty space to maintain layout
		}
		b.WriteString("\n")
	}

	// Determine visual selection range
	visualStart, visualEnd := -1, -1
	if r.visualMode {
		visualStart, visualEnd = r.visualStart, r.cursor
		if visualStart > visualEnd {
			visualStart, visualEnd = visualEnd, visualStart
		}
	}

	for i := r.scrollOffset; i < endIdx; i++ {
		itemIdx := r.visibleIdx[i]
		item := r.items[itemIdx]

		isCursor := i == r.cursor
		isSelected := r.visualMode && i >= visualStart && i <= visualEnd
		isFlashing := r.flashing && (r.flashAll || i == r.flashIdx)

		// Build ancestorIsLast by tracing back through parent chain
		ancestorIsLast := r.buildAncestorIsLast(itemIdx)

		line := r.renderItem(item, isCursor, isSelected, isFlashing, ancestorIsLast)
		b.WriteString(line)
		b.WriteString("\n")
	}

	// Always reserve space for down arrow when scrollable to prevent layout jumps
	if scrollable {
		if canScrollDown {
			b.WriteString(ScrollIndicatorStyle.Render("  ▼"))
		} else {
			b.WriteString("   ") // Empty space to maintain layout
		}
		b.WriteString("\n")
	}

	paddedStyle := lipgloss.NewStyle().Padding(1, 2)
	return paddedStyle.Render(b.String())
}

func (r *ResourceList) renderItem(item ResourceItem, isCursor, isSelected, isFlashing bool, ancestorIsLast []bool) string {
	// Op symbol and color
	var symbol string
	var opStyle lipgloss.Style

	switch item.Op {
	case OpCreate:
		symbol = "+"
		opStyle = OpCreateStyle
	case OpUpdate:
		symbol = "~"
		opStyle = OpUpdateStyle
	case OpDelete:
		symbol = "-"
		opStyle = OpDeleteStyle
	case OpReplace, OpCreateReplace, OpDeleteReplace:
		symbol = "+-"
		opStyle = OpReplaceStyle
	case OpRefresh:
		symbol = "↻"
		opStyle = OpRefreshStyle
	case OpSame:
		symbol = " "
		opStyle = DimStyle
	default:
		symbol = " "
		opStyle = DimStyle
	}

	// If selected or flashing, we need to apply background to each styled segment
	dimStyle := DimStyle
	valueStyle := ValueStyle
	cursorStyle := CursorStyle
	targetStyle := FlagTargetStyle
	replaceStyle := FlagReplaceStyle
	excludeStyle := FlagExcludeStyle
	treeStyle := TreeLineStyle

	// Determine background color
	var bg lipgloss.Color
	hasBackground := false
	if isFlashing {
		bg = ColorFlash
		hasBackground = true
	} else if isSelected {
		bg = ColorSelection
		hasBackground = true
	}

	if hasBackground {
		opStyle = opStyle.Background(bg)
		dimStyle = dimStyle.Background(bg)
		valueStyle = valueStyle.Background(bg)
		cursorStyle = cursorStyle.Background(bg)
		targetStyle = targetStyle.Background(bg)
		replaceStyle = replaceStyle.Background(bg)
		excludeStyle = excludeStyle.Background(bg)
		treeStyle = treeStyle.Background(bg)
	}

	// Cursor indicator
	cursor := "  "
	if isCursor {
		cursor = cursorStyle.Render("> ")
	} else if hasBackground {
		// Need to style the spaces too for consistent background
		cursor = lipgloss.NewStyle().Background(bg).Render("  ")
	}

	// Build tree prefix for nested items
	treePrefix := ""
	if item.Depth > 0 {
		var treeParts []string
		// For each ancestor level, draw vertical line or space
		for i := 0; i < item.Depth-1; i++ {
			if i < len(ancestorIsLast) && ancestorIsLast[i] {
				// Ancestor was last child, no vertical line needed
				if hasBackground {
					treeParts = append(treeParts, lipgloss.NewStyle().Background(bg).Render("   "))
				} else {
					treeParts = append(treeParts, "   ")
				}
			} else {
				// Draw vertical line
				treeParts = append(treeParts, treeStyle.Render("│  "))
			}
		}
		// Draw the connector for this item
		if item.IsLast {
			treeParts = append(treeParts, treeStyle.Render("└─ "))
		} else {
			treeParts = append(treeParts, treeStyle.Render("├─ "))
		}
		treePrefix = strings.Join(treeParts, "")
	}

	// Status icon for execution
	statusIcon := r.renderStatusIcon(item.Status, item.Op, item.CurrentOp)
	if statusIcon != "" {
		statusIcon = " " + statusIcon
	}

	// Format: > [tree] [+] type  name  [T][R][E]  status
	opStr := opStyle.Render(fmt.Sprintf("[%s]", symbol))

	// Calculate max width for type to prevent overflow
	// Account for: cursor(2) + tree prefix(3*depth) + op(4) + spacing(3) + name + badges + status
	// Use a reasonable max of 50 chars for type, but adjust based on available width
	maxTypeLen := 50
	if r.width > 0 {
		// Estimate other elements: cursor(2) + op(4) + spaces(3) + name(~20) + badges(~12) + status(~20) + padding(4)
		treePrefixLen := item.Depth * 3
		otherElements := 2 + treePrefixLen + 4 + 3 + len(item.Name) + 12 + 20 + 4
		available := r.width - otherElements
		if available > 20 && available < maxTypeLen {
			maxTypeLen = available
		}
	}
	truncatedType := truncateMiddle(item.Type, maxTypeLen)
	typeStr := dimStyle.Render(truncatedType)
	nameStr := valueStyle.Render(item.Name)

	// Build flag badges
	flags := r.flags[item.URN]
	var badges []string
	if flags.Target {
		badges = append(badges, targetStyle.Render("[T]"))
	}
	if flags.Replace {
		badges = append(badges, replaceStyle.Render("[R]"))
	}
	if flags.Exclude {
		badges = append(badges, excludeStyle.Render("[E]"))
	}
	badgeStr := ""
	if len(badges) > 0 {
		if hasBackground {
			badgeStr = lipgloss.NewStyle().Background(bg).Render("  ") + strings.Join(badges, "")
		} else {
			badgeStr = "  " + strings.Join(badges, "")
		}
	}

	// Build the line with styled separators for background highlight
	var line string
	if hasBackground {
		bgStyle := lipgloss.NewStyle().Background(bg)
		line = fmt.Sprintf("%s%s%s%s%s%s%s%s%s", cursor, treePrefix, opStr, bgStyle.Render(" "), typeStr, bgStyle.Render("  "), nameStr, badgeStr, statusIcon)
	} else {
		line = fmt.Sprintf("%s%s%s %s  %s%s%s", cursor, treePrefix, opStr, typeStr, nameStr, badgeStr, statusIcon)
	}

	return line
}

// buildAncestorIsLast traces back through the parent chain to determine
// which ancestors were the last child of their parent (for tree line drawing)
func (r *ResourceList) buildAncestorIsLast(itemIdx int) []bool {
	item := r.items[itemIdx]
	if item.Depth == 0 {
		return nil
	}

	result := make([]bool, item.Depth-1)

	// Build a URN -> item index map for quick lookup
	urnToIdx := make(map[string]int)
	for i, it := range r.items {
		urnToIdx[it.URN] = i
	}

	// Trace back through parent chain
	currentURN := item.Parent
	for level := item.Depth - 2; level >= 0; level-- {
		if parentIdx, ok := urnToIdx[currentURN]; ok {
			parent := r.items[parentIdx]
			result[level] = parent.IsLast
			currentURN = parent.Parent
		} else {
			break
		}
	}

	return result
}

func (r *ResourceList) renderStatusIcon(status ItemStatus, op ResourceOp, currentOp ResourceOp) string {
	switch status {
	case StatusPending:
		return StatusPendingStyle.Render("pending")
	case StatusRunning:
		// Show what step is currently running with appropriate color
		return r.getRunningStatusText(currentOp)
	case StatusSuccess:
		// Show completed status with appropriate color based on final op
		return r.getCompletedStatusText(op)
	case StatusFailed:
		return StatusFailedStyle.Render("failed")
	default:
		return ""
	}
}

// getRunningStatusText returns a color-coded status text for a running operation
func (r *ResourceList) getRunningStatusText(op ResourceOp) string {
	switch op {
	case OpCreate:
		return OpCreateStyle.Render("creating...")
	case OpUpdate:
		return OpUpdateStyle.Render("updating...")
	case OpDelete:
		return OpDeleteStyle.Render("deleting...")
	case OpReplace:
		return OpReplaceStyle.Render("replacing...")
	case OpCreateReplace:
		return OpCreateStyle.Render("creating replacement...")
	case OpDeleteReplace:
		return OpDeleteStyle.Render("deleting original...")
	case OpRefresh:
		return OpRefreshStyle.Render("refreshing...")
	case OpRead:
		return StatusRunningStyle.Render("reading...")
	default:
		return StatusRunningStyle.Render("running...")
	}
}

// getCompletedStatusText returns a color-coded status text for a completed operation
func (r *ResourceList) getCompletedStatusText(op ResourceOp) string {
	switch op {
	case OpCreate:
		return OpCreateStyle.Render("created")
	case OpUpdate:
		return OpUpdateStyle.Render("updated")
	case OpDelete:
		return OpDeleteStyle.Render("deleted")
	case OpReplace, OpCreateReplace, OpDeleteReplace:
		return OpReplaceStyle.Render("replaced")
	case OpRefresh:
		return OpRefreshStyle.Render("refreshed")
	case OpRead:
		return StatusSuccessStyle.Render("read")
	case OpSame:
		return StatusSuccessStyle.Render("unchanged")
	default:
		return StatusSuccessStyle.Render("done")
	}
}
