package ui

import (
	"fmt"
	"strings"
	"time"

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
	items []HistoryItem

	// Cursor & scrolling
	cursor       int
	scrollOffset int

	// Configuration
	width  int
	height int
	ready  bool

	// Loading state
	loading    bool
	loadingMsg string
	spinner    spinner.Model

	// Error state
	err error
}

// NewHistoryList creates a new HistoryList component
func NewHistoryList() *HistoryList {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorPrimary)
	return &HistoryList{
		items:   make([]HistoryItem, 0),
		spinner: s,
	}
}

// Spinner returns the spinner model for tick updates
func (h *HistoryList) Spinner() spinner.Model {
	return h.spinner
}

// SetSpinner updates the spinner model
func (h *HistoryList) SetSpinner(s spinner.Model) {
	h.spinner = s
}

// SetSize sets the dimensions for the list
func (h *HistoryList) SetSize(width, height int) {
	h.width = width
	h.height = height
	h.ready = true
	h.ensureCursorVisible()
}

// SetLoading sets the loading state
func (h *HistoryList) SetLoading(loading bool, msg string) {
	h.loading = loading
	h.loadingMsg = msg
}

// SetError sets an error state
func (h *HistoryList) SetError(err error) {
	h.err = err
	h.loading = false
}

// IsLoading returns true if loading
func (h *HistoryList) IsLoading() bool {
	return h.loading
}

// SetItems replaces all items
func (h *HistoryList) SetItems(items []HistoryItem) {
	h.items = items
	h.cursor = 0
	h.scrollOffset = 0
	h.loading = false
	h.err = nil
}

// Clear resets the list
func (h *HistoryList) Clear() {
	h.items = make([]HistoryItem, 0)
	h.cursor = 0
	h.scrollOffset = 0
	h.err = nil
}

// visibleHeight returns the number of lines available for items
func (h *HistoryList) visibleHeight() int {
	// Account for padding (1 top, 1 bottom)
	height := h.height - 2

	// If content is scrollable, reserve 2 lines for scroll indicators
	if h.isScrollable() {
		height -= 2
	}

	if height < 1 {
		height = 1
	}
	return height
}

// isScrollable returns true if there are more items than can fit
func (h *HistoryList) isScrollable() bool {
	baseHeight := h.height - 2
	if baseHeight < 1 {
		baseHeight = 1
	}
	return len(h.items) > baseHeight
}

// ensureCursorVisible adjusts scroll offset to keep cursor visible
func (h *HistoryList) ensureCursorVisible() {
	if len(h.items) == 0 {
		return
	}

	visible := h.visibleHeight()

	// Scroll up if cursor is above visible area
	if h.cursor < h.scrollOffset {
		h.scrollOffset = h.cursor
	}

	// Scroll down if cursor is below visible area
	if h.cursor >= h.scrollOffset+visible {
		h.scrollOffset = h.cursor - visible + 1
	}

	// Clamp scroll offset
	maxScroll := len(h.items) - visible
	if maxScroll < 0 {
		maxScroll = 0
	}
	if h.scrollOffset > maxScroll {
		h.scrollOffset = maxScroll
	}
	if h.scrollOffset < 0 {
		h.scrollOffset = 0
	}
}

// Update handles key events
func (h *HistoryList) Update(msg tea.Msg) tea.Cmd {
	if !h.ready || len(h.items) == 0 {
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
	h.cursor += delta
	if h.cursor < 0 {
		h.cursor = 0
	}
	if h.cursor >= len(h.items) {
		h.cursor = len(h.items) - 1
	}
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
	if h.loading {
		return h.renderLoading()
	}
	if h.err != nil {
		return h.renderError()
	}
	return h.renderItems()
}

func (h *HistoryList) renderLoading() string {
	msg := h.loadingMsg
	if msg == "" {
		msg = "Loading..."
	}
	content := fmt.Sprintf("%s %s", h.spinner.View(), msg)
	return lipgloss.Place(h.width, h.height, lipgloss.Center, lipgloss.Center, content)
}

func (h *HistoryList) renderError() string {
	paddedStyle := lipgloss.NewStyle().Padding(1, 2)
	errMsg := ErrorStyle.Render(fmt.Sprintf("Error: %v", h.err))
	return paddedStyle.Render(errMsg)
}

func (h *HistoryList) renderItems() string {
	if len(h.items) == 0 {
		msg := DimStyle.Render("No history")
		return lipgloss.Place(h.width, h.height, lipgloss.Center, lipgloss.Center, msg)
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
		if canScrollUp {
			b.WriteString(ScrollIndicatorStyle.Render("  ▲"))
		} else {
			b.WriteString("   ")
		}
		b.WriteString("\n")
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
		if canScrollDown {
			b.WriteString(ScrollIndicatorStyle.Render("  ▼"))
		} else {
			b.WriteString("   ")
		}
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
	kindStr := h.renderKind(item.Kind)

	// Result status
	resultStr := h.renderResult(item.Result)

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

func (h *HistoryList) renderKind(kind string) string {
	// Match the kind to operation colors
	switch kind {
	case "update":
		return OpCreateStyle.Render("update")
	case "refresh":
		return OpRefreshStyle.Render("refresh")
	case "destroy":
		return OpDeleteStyle.Render("destroy")
	case "preview":
		return DimStyle.Render("preview")
	default:
		return DimStyle.Render(kind)
	}
}

func (h *HistoryList) renderResult(result string) string {
	switch result {
	case "succeeded":
		return StatusSuccessStyle.Render("succeeded")
	case "failed":
		return StatusFailedStyle.Render("failed")
	case "in-progress":
		return StatusRunningStyle.Render("in-progress")
	default:
		return DimStyle.Render(result)
	}
}

func (h *HistoryList) formatTime(timeStr string) string {
	// Try to parse the time string (Pulumi uses RFC3339)
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		// Fall back to original string, truncated
		if len(timeStr) > 16 {
			return DimStyle.Render(timeStr[:16])
		}
		return DimStyle.Render(timeStr)
	}

	// Format as "2006-01-02 15:04"
	return DimStyle.Render(t.Format("2006-01-02 15:04"))
}

func (h *HistoryList) renderChanges(changes map[string]int) string {
	if changes == nil || len(changes) == 0 {
		return DimStyle.Render("no changes")
	}

	var parts []string

	create := changes["create"]
	update := changes["update"]
	del := changes["delete"]
	replace := changes["replace"]
	same := changes["same"]

	if create > 0 {
		parts = append(parts, OpCreateStyle.Render(fmt.Sprintf("+%d", create)))
	}
	if update > 0 {
		parts = append(parts, OpUpdateStyle.Render(fmt.Sprintf("~%d", update)))
	}
	if replace > 0 {
		parts = append(parts, OpReplaceStyle.Render(fmt.Sprintf("±%d", replace)))
	}
	if del > 0 {
		parts = append(parts, OpDeleteStyle.Render(fmt.Sprintf("-%d", del)))
	}

	if len(parts) == 0 {
		// Only "same" resources
		if same > 0 {
			return DimStyle.Render(fmt.Sprintf("%d unchanged", same))
		}
		return DimStyle.Render("no changes")
	}

	return strings.Join(parts, " ")
}
