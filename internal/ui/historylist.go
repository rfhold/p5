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
}

// NewHistoryList creates a new HistoryList component
func NewHistoryList() *HistoryList {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorPrimary)
	h := &HistoryList{
		items: make([]HistoryItem, 0),
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
	h.SetLoading(false, "")
	h.ClearError()
}

// Clear resets the list
func (h *HistoryList) Clear() {
	h.items = make([]HistoryItem, 0)
	h.cursor = 0
	h.scrollOffset = 0
	h.ClearError()
}

// visibleHeight returns the number of lines available for items
func (h *HistoryList) visibleHeight() int {
	return CalculateVisibleHeight(h.Height(), len(h.items), 2) // 2 = padding (1 top, 1 bottom)
}

// isScrollable returns true if there are more items than can fit
func (h *HistoryList) isScrollable() bool {
	return IsScrollable(h.Height(), len(h.items), 2)
}

// ensureCursorVisible adjusts scroll offset to keep cursor visible
func (h *HistoryList) ensureCursorVisible() {
	h.scrollOffset = EnsureCursorVisible(h.cursor, h.scrollOffset, len(h.items), h.visibleHeight())
}

// Update handles key events
func (h *HistoryList) Update(msg tea.Msg) tea.Cmd {
	if !h.IsReady() || len(h.items) == 0 {
		return nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, Keys.Up):
			h.moveCursor(-1)
		case key.Matches(msg, Keys.Down):
			h.moveCursor(1)
		case key.Matches(msg, Keys.PageUp):
			h.moveCursor(-h.visibleHeight())
		case key.Matches(msg, Keys.PageDown):
			h.moveCursor(h.visibleHeight())
		case key.Matches(msg, Keys.Home):
			h.cursor = 0
			h.ensureCursorVisible()
		case key.Matches(msg, Keys.End):
			h.cursor = len(h.items) - 1
			h.ensureCursorVisible()
		}
	}

	return nil
}

// moveCursor moves the cursor by delta, clamping to valid range
func (h *HistoryList) moveCursor(delta int) {
	h.cursor = MoveCursor(h.cursor, delta, len(h.items))
	h.ensureCursorVisible()
}

// SelectedItem returns the currently selected item, or nil if none
func (h *HistoryList) SelectedItem() *HistoryItem {
	if len(h.items) == 0 || h.cursor < 0 || h.cursor >= len(h.items) {
		return nil
	}
	return &h.items[h.cursor]
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
	return h.scrollOffset >= len(h.items)-h.visibleHeight()
}

// View renders the history list
func (h *HistoryList) View() string {
	if rendered, handled := h.RenderLoadingState(); handled {
		return rendered
	}
	return h.renderItems()
}

func (h *HistoryList) renderItems() string {
	if len(h.items) == 0 {
		return RenderCenteredMessage("No history", h.Width(), h.Height())
	}

	var b strings.Builder
	visible := h.visibleHeight()
	endIdx := h.scrollOffset + visible
	if endIdx > len(h.items) {
		endIdx = len(h.items)
	}

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
		item := h.items[i]
		isCursor := i == h.cursor
		line := h.renderItem(item, isCursor)
		b.WriteString(line)
		b.WriteString("\n")
	}

	// Down arrow indicator
	if scrollable {
		b.WriteString(RenderScrollDownIndicator(canScrollDown))
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
		userStr = DimStyle.Render(fmt.Sprintf("by %s", item.User))
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
