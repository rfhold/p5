package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SelectorItem is an interface for items that can be displayed in a SelectorDialog
type SelectorItem interface {
	// Label returns the display text for the item
	Label() string
	// IsCurrent returns true if this is the currently selected/active item
	IsCurrent() bool
}

// SelectorDialog is a generic modal dialog for selecting an item from a list
type SelectorDialog[T SelectorItem] struct {
	items   []T
	cursor  int
	width   int
	height  int
	visible bool
	loading bool
	err     error

	// Configuration
	title           string
	loadingText     string
	emptyText       string
	maxVisible      int
	renderItem      func(item T, isCursor bool) string // Optional custom item renderer
	renderExtraInfo func(item T) string                // Optional extra info after item label
}

// NewSelectorDialog creates a new selector dialog with the given title
func NewSelectorDialog[T SelectorItem](title string) *SelectorDialog[T] {
	return &SelectorDialog[T]{
		title:       title,
		loadingText: "Loading...",
		emptyText:   "No items found",
		maxVisible:  10,
	}
}

// SetItems sets the list of available items and positions cursor on current item
func (s *SelectorDialog[T]) SetItems(items []T) {
	s.items = items
	s.loading = false
	s.err = nil
	// Set cursor to current item if found
	for i, item := range items {
		if item.IsCurrent() {
			s.cursor = i
			break
		}
	}
}

// SetLoading sets the loading state
func (s *SelectorDialog[T]) SetLoading(loading bool) {
	s.loading = loading
}

// SetError sets an error state
func (s *SelectorDialog[T]) SetError(err error) {
	s.err = err
	s.loading = false
}

// SetSize sets the dialog dimensions for centering
func (s *SelectorDialog[T]) SetSize(width, height int) {
	s.width = width
	s.height = height
}

// Show shows the selector dialog
func (s *SelectorDialog[T]) Show() {
	s.visible = true
}

// Hide hides the selector dialog
func (s *SelectorDialog[T]) Hide() {
	s.visible = false
}

// Visible returns whether the selector is visible
func (s *SelectorDialog[T]) Visible() bool {
	return s.visible
}

// SetTitle sets the dialog title
func (s *SelectorDialog[T]) SetTitle(title string) {
	s.title = title
}

// SetLoadingText sets the text shown while loading
func (s *SelectorDialog[T]) SetLoadingText(text string) {
	s.loadingText = text
}

// SetEmptyText sets the text shown when no items are available
func (s *SelectorDialog[T]) SetEmptyText(text string) {
	s.emptyText = text
}

// SetMaxVisible sets the maximum number of visible items before scrolling
func (s *SelectorDialog[T]) SetMaxVisible(maxItems int) {
	s.maxVisible = maxItems
}

// SetItemRenderer sets a custom item renderer function
func (s *SelectorDialog[T]) SetItemRenderer(fn func(item T, isCursor bool) string) {
	s.renderItem = fn
}

// SetExtraInfoRenderer sets a function to render extra info after the item label
func (s *SelectorDialog[T]) SetExtraInfoRenderer(fn func(item T) string) {
	s.renderExtraInfo = fn
}

// SelectedItem returns the currently selected item, or nil if none
func (s *SelectorDialog[T]) SelectedItem() *T {
	if len(s.items) == 0 || s.cursor < 0 || s.cursor >= len(s.items) {
		return nil
	}
	return &s.items[s.cursor]
}

// HasItems returns whether any items are available
func (s *SelectorDialog[T]) HasItems() bool {
	return len(s.items) > 0
}

// Update handles key events and returns true if an item was selected
func (s *SelectorDialog[T]) Update(msg tea.KeyMsg) (selected bool, cmd tea.Cmd) {
	if !s.visible {
		return false, nil
	}

	switch {
	case key.Matches(msg, Keys.Up):
		if s.cursor > 0 {
			s.cursor--
		}
	case key.Matches(msg, Keys.Down):
		if s.cursor < len(s.items)-1 {
			s.cursor++
		}
	case key.Matches(msg, Keys.Home):
		s.cursor = 0
	case key.Matches(msg, Keys.End):
		s.cursor = len(s.items) - 1
	case msg.String() == "enter":
		if len(s.items) > 0 {
			s.visible = false
			return true, nil
		}
	case key.Matches(msg, Keys.Escape):
		s.visible = false
		return false, nil
	}

	return false, nil
}

// View renders the selector dialog
func (s *SelectorDialog[T]) View() string {
	titleText := s.title

	var content string
	switch {
	case s.loading:
		content = DimStyle.Render(s.loadingText)
	case s.err != nil:
		content = ErrorStyle.Render(s.err.Error())
	case len(s.items) == 0:
		content = DimStyle.Render(s.emptyText)
	default:
		// Add line count hint to title if scrollable
		if len(s.items) > s.maxVisible {
			// Calculate visible range
			start := max(s.cursor-s.maxVisible/2, 0)
			end := start + s.maxVisible
			if end > len(s.items) {
				end = len(s.items)
				start = end - s.maxVisible
			}
			titleText += DimStyle.Render(fmt.Sprintf(" [%d-%d/%d]", start+1, end, len(s.items)))
		}
		content = s.renderItems()
	}

	title := DialogTitleStyle.Render(titleText)

	// Footer hints
	footer := DimStyle.Render("\n↑/↓ navigate  enter select  esc cancel")

	dialog := DialogStyle.Render(lipgloss.JoinVertical(lipgloss.Left, title, content, footer))

	return lipgloss.Place(s.width, s.height, lipgloss.Center, lipgloss.Center,
		dialog,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(ColorBg),
	)
}

// renderItems renders the scrollable item list
func (s *SelectorDialog[T]) renderItems() string {
	var lines []string

	// Calculate visible range for scrolling
	start := 0
	end := len(s.items)

	if len(s.items) > s.maxVisible {
		// Center cursor in visible range
		start = max(s.cursor-s.maxVisible/2, 0)
		end = start + s.maxVisible
		if end > len(s.items) {
			end = len(s.items)
			start = end - s.maxVisible
		}
	}

	for i := start; i < end; i++ {
		item := s.items[i]
		isCursor := i == s.cursor

		var line string
		if s.renderItem != nil {
			// Use custom renderer
			line = s.renderItem(item, isCursor)
		} else {
			// Default rendering
			line = s.defaultRenderItem(item, isCursor)
		}
		lines = append(lines, line)
	}

	// Add scroll hint at bottom (import modal style)
	if len(s.items) > s.maxVisible {
		canScrollUp := start > 0
		canScrollDown := end < len(s.items)
		hint := RenderScrollHint(canScrollUp, canScrollDown, "  ")
		if hint != "" {
			lines = append(lines, hint)
		}
	}

	return strings.Join(lines, "\n")
}

// defaultRenderItem provides the default item rendering
func (s *SelectorDialog[T]) defaultRenderItem(item T, isCursor bool) string {
	cursor := "  "
	if isCursor {
		cursor = CursorStyle.Render("> ")
	}

	label := item.Label()
	var name string
	switch {
	case item.IsCurrent():
		name = ValueStyle.Render(label) + DimStyle.Render(" (current)")
	case isCursor:
		name = ValueStyle.Render(label)
	default:
		name = DimStyle.Render(label)
	}

	// Add extra info if renderer is set
	extra := ""
	if s.renderExtraInfo != nil {
		extra = s.renderExtraInfo(item)
	}

	return cursor + name + extra
}
