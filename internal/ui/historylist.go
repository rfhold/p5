package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// HistoryItem represents a single update in the history list
type HistoryItem struct {
	Version         int
	Kind            string // "update", "preview", "refresh", "destroy"
	StartTime       string
	EndTime         string
	Message         string
	Result          string         // "succeeded", "failed", "in-progress"
	ResourceChanges map[string]int // e.g., {"create": 2, "update": 1}
	User            string         // git.author who ran the update
	UserEmail       string         // git.author.email
}

// HistoryList is a scrollable list of stack history updates
type HistoryList struct {
	ListBase // Embed common list functionality for loading/error state

	items []HistoryItem

	// Cursor & scrolling
	cursor       int
	scrollOffset int

	// Filter state
	filter      FilterState
	filteredIdx []int // Indices into items that match filter (nil = no filter active)
}

// NewHistoryList creates a new HistoryList component
func NewHistoryList() *HistoryList {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorPrimary)
	h := &HistoryList{
		items:  make([]HistoryItem, 0),
		filter: NewFilterState(),
	}
	h.SetSpinner(s)
	return h
}

// SetSize sets the dimensions for the list and ensures cursor is visible
func (h *HistoryList) SetSize(width, height int) {
	h.ListBase.SetSize(width, height)
	h.ensureCursorVisible()
}

// SetItems replaces all items
func (h *HistoryList) SetItems(items []HistoryItem) {
	h.items = items
	h.cursor = 0
	h.scrollOffset = 0
	h.filteredIdx = nil
	h.filter.Deactivate()
	h.SetLoading(false, "")
	h.ClearError()
}

// Clear resets the list
func (h *HistoryList) Clear() {
	h.items = make([]HistoryItem, 0)
	h.cursor = 0
	h.scrollOffset = 0
	h.filteredIdx = nil
	h.filter.Deactivate()
	h.ClearError()
}

// effectiveItemCount returns the number of items being displayed (filtered or all)
func (h *HistoryList) effectiveItemCount() int {
	if h.filteredIdx != nil {
		return len(h.filteredIdx)
	}
	return len(h.items)
}

// effectiveIndex converts a cursor position to the actual item index
func (h *HistoryList) effectiveIndex(cursorPos int) int {
	if h.filteredIdx != nil {
		if cursorPos < 0 || cursorPos >= len(h.filteredIdx) {
			return -1
		}
		return h.filteredIdx[cursorPos]
	}
	return cursorPos
}

// visibleHeight returns the number of lines available for items
func (h *HistoryList) visibleHeight() int {
	itemCount := h.effectiveItemCount()
	padding := 2 // 1 top, 1 bottom
	if h.filter.ActiveOrApplied() {
		padding++
	}
	return CalculateVisibleHeight(h.Height(), itemCount, padding)
}

// isScrollable returns true if there are more items than can fit
func (h *HistoryList) isScrollable() bool {
	itemCount := h.effectiveItemCount()
	padding := 2
	if h.filter.ActiveOrApplied() {
		padding++
	}
	return IsScrollable(h.Height(), itemCount, padding)
}

// ensureCursorVisible adjusts scroll offset to keep cursor visible
func (h *HistoryList) ensureCursorVisible() {
	itemCount := h.effectiveItemCount()
	h.scrollOffset = EnsureCursorVisible(h.cursor, h.scrollOffset, itemCount, h.visibleHeight())
}

// rebuildFilteredIndex applies the current filter to build the filtered index
func (h *HistoryList) rebuildFilteredIndex() {
	if !h.filter.Applied() {
		h.filteredIdx = nil
		return
	}

	h.filteredIdx = make([]int, 0)
	for i := range h.items {
		if h.filter.MatchesAny(h.items[i].Kind, h.items[i].Message, h.items[i].User, h.items[i].Result) {
			h.filteredIdx = append(h.filteredIdx, i)
		}
	}

	// Adjust cursor if it's now outside filtered range
	if len(h.filteredIdx) > 0 && h.cursor >= len(h.filteredIdx) {
		h.cursor = len(h.filteredIdx) - 1
	}
}

// Update handles key events
func (h *HistoryList) Update(msg tea.Msg) tea.Cmd {
	if !h.IsReady() || len(h.items) == 0 {
		return nil
	}

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}

	// Handle filter activation with "/"
	if key.Matches(keyMsg, Keys.Filter) && !h.filter.Active() {
		h.filter.Activate()
		h.rebuildFilteredIndex()
		return nil
	}

	// Forward to filter if active
	if h.filter.Active() {
		cmd, handled := h.filter.Update(keyMsg)
		if handled {
			h.rebuildFilteredIndex()
			return cmd
		}
	}

	itemCount := h.effectiveItemCount()

	switch {
	case key.Matches(keyMsg, Keys.Up):
		h.moveCursor(-1)
	case key.Matches(keyMsg, Keys.Down):
		h.moveCursor(1)
	case key.Matches(keyMsg, Keys.PageUp):
		h.moveCursor(-h.visibleHeight())
	case key.Matches(keyMsg, Keys.PageDown):
		h.moveCursor(h.visibleHeight())
	case key.Matches(keyMsg, Keys.Home):
		h.cursor = 0
		h.ensureCursorVisible()
	case key.Matches(keyMsg, Keys.End):
		h.cursor = itemCount - 1
		h.ensureCursorVisible()
	}

	return nil
}

// moveCursor moves the cursor by delta, clamping to valid range
func (h *HistoryList) moveCursor(delta int) {
	itemCount := h.effectiveItemCount()
	h.cursor = MoveCursor(h.cursor, delta, itemCount)
	h.ensureCursorVisible()
}

// SelectedItem returns the currently selected item, or nil if none
func (h *HistoryList) SelectedItem() *HistoryItem {
	itemCount := h.effectiveItemCount()
	if itemCount == 0 || h.cursor < 0 || h.cursor >= itemCount {
		return nil
	}
	idx := h.effectiveIndex(h.cursor)
	if idx < 0 || idx >= len(h.items) {
		return nil
	}
	return &h.items[idx]
}

// TotalItems returns the total number of items
func (h *HistoryList) TotalItems() int {
	return len(h.items)
}

// AtTop returns true if scrolled to top
func (h *HistoryList) AtTop() bool {
	return h.scrollOffset == 0
}

// AtBottom returns true if scrolled to bottom
func (h *HistoryList) AtBottom() bool {
	itemCount := h.effectiveItemCount()
	return h.scrollOffset >= itemCount-h.visibleHeight()
}

// View renders the history list
func (h *HistoryList) View() string {
	if rendered, handled := h.RenderLoadingState(); handled {
		return rendered
	}
	return h.renderItems()
}

func (h *HistoryList) renderItems() string {
	itemCount := h.effectiveItemCount()

	// Handle filter with no matches
	if h.filter.Applied() && itemCount == 0 {
		var b strings.Builder
		b.WriteString(DimStyle.Render("No matches"))
		b.WriteString("\n\n")
		b.WriteString(RenderFilterBar(&h.filter, 0, len(h.items), h.Width()))
		paddedStyle := lipgloss.NewStyle().Padding(1, 2)
		return paddedStyle.Render(b.String())
	}

	if len(h.items) == 0 {
		return RenderCenteredMessage("No history", h.Width(), h.Height())
	}

	var b strings.Builder
	visible := h.visibleHeight()
	endIdx := min(h.scrollOffset+visible, itemCount)

	// Check if content is scrollable
	scrollable := h.isScrollable()
	canScrollUp := !h.AtTop()
	canScrollDown := !h.AtBottom()

	// Up arrow indicator
	if scrollable {
		b.WriteString(RenderScrollUpIndicator(canScrollUp))
	}

	// Render items
	for i := h.scrollOffset; i < endIdx; i++ {
		idx := h.effectiveIndex(i)
		if idx < 0 || idx >= len(h.items) {
			continue
		}
		item := h.items[idx]
		isCursor := i == h.cursor
		line := h.renderItem(item, isCursor)
		b.WriteString(line)
		b.WriteString("\n")
	}

	// Down arrow indicator
	if scrollable {
		b.WriteString(RenderScrollDownIndicator(canScrollDown))
	}

	// Add filter bar at bottom when active or applied
	if h.filter.ActiveOrApplied() {
		filterBar := RenderFilterBar(&h.filter, itemCount, len(h.items), h.Width())
		b.WriteString(filterBar)
		b.WriteString("\n")
	}

	paddedStyle := lipgloss.NewStyle().Padding(1, 2)
	return paddedStyle.Render(b.String())
}

func (h *HistoryList) renderItem(item HistoryItem, isCursor bool) string {
	// Cursor indicator
	cursor := "  "
	if isCursor {
		cursor = CursorStyle.Render("> ")
	}

	// Version
	versionStr := DimStyle.Render(fmt.Sprintf("#%d", item.Version))

	// Kind with appropriate color
	kindStr := RenderHistoryKind(item.Kind)

	// Result status
	resultStr := RenderHistoryResult(item.Result)

	// Timestamp - try to format it nicely
	timeStr := h.formatTime(item.StartTime)

	// Resource changes summary
	changesStr := h.renderChanges(item.ResourceChanges)

	// User (short form)
	userStr := ""
	if item.User != "" {
		userStr = DimStyle.Render("by " + item.User)
	}

	// Message (truncated)
	msgStr := ""
	if item.Message != "" {
		msg := item.Message
		if len(msg) > 30 {
			msg = msg[:27] + "..."
		}
		msgStr = DimStyle.Render(fmt.Sprintf(" %q", msg))
	}

	// Format: > #1  update  succeeded  2024-01-15 10:30  +2 ~1 -0  by user  "commit message"
	line := fmt.Sprintf("%s%s  %s  %s  %s  %s",
		cursor,
		versionStr,
		kindStr,
		resultStr,
		timeStr,
		changesStr,
	)
	if userStr != "" {
		line += "  " + userStr
	}
	if msgStr != "" {
		line += msgStr
	}

	return line
}

// renderKind and renderResult are now shared functions in styles.go.

func (h *HistoryList) formatTime(timeStr string) string {
	return FormatTimeStyled(timeStr, "2006-01-02 15:04", 16, DimStyle)
}

func (h *HistoryList) renderChanges(changes map[string]int) string {
	return RenderResourceChanges(changes, ResourceChangesCompact)
}

// FilterActive returns whether the filter is currently active (typing) or applied (has text)
func (h *HistoryList) FilterActive() bool {
	return h.filter.ActiveOrApplied()
}

// FilterInputActive returns true if the filter is actively receiving input (user is typing)
func (h *HistoryList) FilterInputActive() bool {
	return h.filter.Active()
}
