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

	// Filter state
	filter      FilterState
	filteredIdx []int // Indices into items that match filter (nil = no filter active)
}

// NewSelectorDialog creates a new selector dialog with the given title
func NewSelectorDialog[T SelectorItem](title string) *SelectorDialog[T] {
	return &SelectorDialog[T]{
		title:       title,
		loadingText: "Loading...",
		emptyText:   "No items found",
		maxVisible:  10,
		filter:      NewFilterState(),
	}
}

// SetItems sets the list of available items and positions cursor on current item
func (s *SelectorDialog[T]) SetItems(items []T) {
	s.items = items
	s.loading = false
	s.err = nil
	s.filteredIdx = nil
	s.filter.Deactivate()
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
	s.filter.Deactivate()
	s.filteredIdx = nil
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
	itemCount := s.effectiveItemCount()
	if itemCount == 0 || s.cursor < 0 || s.cursor >= itemCount {
		return nil
	}
	idx := s.effectiveIndex(s.cursor)
	if idx < 0 || idx >= len(s.items) {
		return nil
	}
	return &s.items[idx]
}

// effectiveItemCount returns the number of items being displayed (filtered or all)
func (s *SelectorDialog[T]) effectiveItemCount() int {
	if s.filteredIdx != nil {
		return len(s.filteredIdx)
	}
	return len(s.items)
}

// effectiveIndex converts a cursor position to the actual item index
func (s *SelectorDialog[T]) effectiveIndex(cursorPos int) int {
	if s.filteredIdx != nil {
		if cursorPos < 0 || cursorPos >= len(s.filteredIdx) {
			return -1
		}
		return s.filteredIdx[cursorPos]
	}
	return cursorPos
}

// rebuildFilteredIndex applies the current filter to build the filtered index
func (s *SelectorDialog[T]) rebuildFilteredIndex() {
	if !s.filter.Applied() {
		s.filteredIdx = nil
		return
	}

	s.filteredIdx = make([]int, 0)
	for i, item := range s.items {
		if s.filter.Matches(item.Label()) {
			s.filteredIdx = append(s.filteredIdx, i)
		}
	}

	// Adjust cursor if it's now outside filtered range
	if len(s.filteredIdx) > 0 && s.cursor >= len(s.filteredIdx) {
		s.cursor = len(s.filteredIdx) - 1
	}
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

	// Handle filter activation with "/"
	if key.Matches(msg, Keys.Filter) && !s.filter.Active() {
		s.filter.Activate()
		s.rebuildFilteredIndex()
		return false, nil
	}

	// Forward to filter if active
	if s.filter.Active() {
		cmd, handled := s.filter.Update(msg)
		if handled {
			s.rebuildFilteredIndex()
			return false, cmd
		}
	}

	itemCount := s.effectiveItemCount()

	switch {
	case key.Matches(msg, Keys.Up):
		if s.cursor > 0 {
			s.cursor--
		}
	case key.Matches(msg, Keys.Down):
		if s.cursor < itemCount-1 {
			s.cursor++
		}
	case key.Matches(msg, Keys.Home):
		s.cursor = 0
	case key.Matches(msg, Keys.End):
		s.cursor = itemCount - 1
	case msg.String() == "enter":
		if itemCount > 0 {
			s.visible = false
			s.filter.Deactivate()
			return true, nil
		}
	case key.Matches(msg, Keys.Escape):
		s.visible = false
		s.filter.Deactivate()
		s.filteredIdx = nil
		return false, nil
	}

	return false, nil
}

// View renders the selector dialog
func (s *SelectorDialog[T]) View() string {
	titleText := s.title
	itemCount := s.effectiveItemCount()

	var content string
	switch {
	case s.loading:
		content = DimStyle.Render(s.loadingText)
	case s.err != nil:
		content = ErrorStyle.Render(s.err.Error())
	case len(s.items) == 0:
		content = DimStyle.Render(s.emptyText)
	case s.filter.Applied() && itemCount == 0:
		content = DimStyle.Render("No matches")
	default:
		// Add line count hint to title if scrollable
		if itemCount > s.maxVisible {
			// Calculate visible range
			start := max(s.cursor-s.maxVisible/2, 0)
			end := start + s.maxVisible
			if end > itemCount {
				end = itemCount
				start = end - s.maxVisible
			}
			if start < 0 {
				start = 0
			}
			titleText += DimStyle.Render(fmt.Sprintf(" [%d-%d/%d]", start+1, end, itemCount))
		}
		content = s.renderItems()
	}

	title := DialogTitleStyle.Render(titleText)

	// Footer with filter bar or hints
	var footer string
	if s.filter.ActiveOrApplied() {
		filterBar := RenderFilterBar(&s.filter, itemCount, len(s.items), s.width)
		footer = "\n" + filterBar + "\n" + DimStyle.Render("↑/↓ navigate  enter select  esc cancel")
	} else {
		footer = DimStyle.Render("\n↑/↓ navigate  / filter  enter select  esc cancel")
	}

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
	itemCount := s.effectiveItemCount()

	// Calculate visible range for scrolling
	start := 0
	end := itemCount

	if itemCount > s.maxVisible {
		// Center cursor in visible range
		start = max(s.cursor-s.maxVisible/2, 0)
		end = start + s.maxVisible
		if end > itemCount {
			end = itemCount
			start = end - s.maxVisible
		}
		if start < 0 {
			start = 0
		}
	}

	for i := start; i < end; i++ {
		idx := s.effectiveIndex(i)
		if idx < 0 || idx >= len(s.items) {
			continue
		}
		item := s.items[idx]
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
	if itemCount > s.maxVisible {
		canScrollUp := start > 0
		canScrollDown := end < itemCount
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
