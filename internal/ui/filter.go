package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// FilterState manages filter state for list components
type FilterState struct {
	active bool
	input  textinput.Model
}

// NewFilterState creates a new filter state
func NewFilterState() FilterState {
	ti := textinput.New()
	ti.Prompt = "/"
	ti.CharLimit = 100
	ti.Width = 30
	ti.PromptStyle = CursorStyle
	ti.TextStyle = ValueStyle
	return FilterState{input: ti}
}

// Active returns whether filter input mode is active (user is typing)
func (f *FilterState) Active() bool {
	return f.active
}

// Applied returns whether a filter is currently applied (has text)
func (f *FilterState) Applied() bool {
	return f.input.Value() != ""
}

// ActiveOrApplied returns true if the filter is either being typed in or has text applied
func (f *FilterState) ActiveOrApplied() bool {
	return f.active || f.input.Value() != ""
}

// Text returns the current filter text
func (f *FilterState) Text() string {
	return f.input.Value()
}

// Activate enters filter mode, resetting any previous filter text
func (f *FilterState) Activate() {
	f.active = true
	f.input.SetValue("")
	f.input.Focus()
}

// Deactivate exits filter mode but keeps filter text applied
func (f *FilterState) Deactivate() {
	f.active = false
	f.input.Blur()
}

// Clear clears the filter text but stays in filter mode
func (f *FilterState) Clear() {
	f.input.SetValue("")
}

// Update handles key events when filter is active
// Returns: (tea.Cmd, handled bool)
func (f *FilterState) Update(msg tea.KeyMsg) (tea.Cmd, bool) {
	if !f.active {
		return nil, false
	}

	// Handle escape - exit filter mode, keep filter applied
	if msg.Type == tea.KeyEscape {
		f.Deactivate()
		return nil, true
	}

	// Handle enter - exit filter mode keeping filter applied
	if msg.Type == tea.KeyEnter {
		f.Deactivate()
		return nil, true
	}

	// Forward other keys to text input
	var cmd tea.Cmd
	f.input, cmd = f.input.Update(msg)
	return cmd, true
}

// Matches returns true if the given text matches the filter (case-insensitive)
func (f *FilterState) Matches(text string) bool {
	if f.input.Value() == "" {
		return true
	}
	return strings.Contains(strings.ToLower(text), strings.ToLower(f.input.Value()))
}

// MatchesAny returns true if any of the given texts match the filter (case-insensitive)
func (f *FilterState) MatchesAny(texts ...string) bool {
	if f.input.Value() == "" {
		return true
	}
	filter := strings.ToLower(f.input.Value())
	for _, text := range texts {
		if strings.Contains(strings.ToLower(text), filter) {
			return true
		}
	}
	return false
}

// View returns the filter input view
func (f *FilterState) View() string {
	if f.active {
		return f.input.View()
	}
	// When not active but has text, show static filter display
	if f.input.Value() != "" {
		return DimStyle.Render("/") + ValueStyle.Render(f.input.Value())
	}
	return ""
}

// RenderFilterBar renders a filter bar with match count
func RenderFilterBar(filter *FilterState, matchCount, totalCount, width int) string {
	if !filter.ActiveOrApplied() {
		return ""
	}

	// Build: /search... (3/10)
	input := filter.View()
	countStr := DimStyle.Render(fmt.Sprintf(" (%d/%d)", matchCount, totalCount))

	return input + countStr
}
