package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ImportSuggestion represents a single import suggestion from a plugin
type ImportSuggestion struct {
	ID          string
	Label       string
	Description string
	PluginName  string
}

// ImportModal is a modal dialog for importing a resource
type ImportModal struct {
	ModalBase // Embedded modal base for common functionality

	// Resource being imported
	resourceType string
	resourceName string
	resourceURN  string
	parentURN    string // Parent URN for component hierarchy

	// Text input for import ID
	input textinput.Model

	// Suggestions from plugins
	suggestions        []ImportSuggestion
	selectedIdx        int
	loadingSuggestions bool
	showSuggestions    bool

	// State
	err error

	// Filter state for suggestions
	filter      FilterState
	filteredIdx []int // Indices into suggestions that match filter (nil = no filter active)
}

// NewImportModal creates a new import modal
func NewImportModal() *ImportModal {
	ti := textinput.New()
	ti.Placeholder = "Enter import ID..."
	ti.CharLimit = 256
	ti.Width = DefaultInputWidth

	return &ImportModal{
		input:  ti,
		filter: NewFilterState(),
	}
}

// SetSize is inherited from ModalBase

// Show shows the import modal for the given resource
func (m *ImportModal) Show(resourceType, resourceName, resourceURN, parentURN string) {
	m.resourceType = resourceType
	m.resourceName = resourceName
	m.resourceURN = resourceURN
	m.parentURN = parentURN
	m.ModalBase.Show()
	m.err = nil
	m.input.SetValue("")
	m.input.Focus()
	m.suggestions = nil
	m.selectedIdx = 0
	m.loadingSuggestions = true
	m.showSuggestions = false
}

// SetSuggestions sets the import suggestions from plugins
func (m *ImportModal) SetSuggestions(suggestions []ImportSuggestion) {
	m.suggestions = suggestions
	m.loadingSuggestions = false
	m.showSuggestions = len(suggestions) > 0
	m.selectedIdx = 0
	m.filteredIdx = nil
	m.filter.Deactivate()
}

// SetLoadingSuggestions sets the loading state
func (m *ImportModal) SetLoadingSuggestions(loading bool) {
	m.loadingSuggestions = loading
}

// Hide hides the import modal
func (m *ImportModal) Hide() {
	m.ModalBase.Hide()
	m.input.Blur()
	m.filter.Deactivate()
	m.filteredIdx = nil
}

// Visible is inherited from ModalBase

// SetError sets an error to display
func (m *ImportModal) SetError(err error) {
	m.err = err
}

// GetImportID returns the entered import ID
func (m *ImportModal) GetImportID() string {
	return strings.TrimSpace(m.input.Value())
}

// GetResourceURN returns the URN of the resource being imported
func (m *ImportModal) GetResourceURN() string {
	return m.resourceURN
}

// GetResourceType returns the type of the resource being imported
func (m *ImportModal) GetResourceType() string {
	return m.resourceType
}

// GetResourceName returns the name of the resource being imported
func (m *ImportModal) GetResourceName() string {
	return m.resourceName
}

// GetParentURN returns the parent URN for the resource being imported
func (m *ImportModal) GetParentURN() string {
	return m.parentURN
}

// maxVisibleSuggestions is the max number of suggestions shown at once
const maxVisibleSuggestions = 8

// effectiveSuggestionCount returns the number of suggestions being displayed (filtered or all)
func (m *ImportModal) effectiveSuggestionCount() int {
	if m.filteredIdx != nil {
		return len(m.filteredIdx)
	}
	return len(m.suggestions)
}

// effectiveSuggestionIndex converts a cursor position to the actual suggestion index
func (m *ImportModal) effectiveSuggestionIndex(cursorPos int) int {
	if m.filteredIdx != nil {
		if cursorPos < 0 || cursorPos >= len(m.filteredIdx) {
			return -1
		}
		return m.filteredIdx[cursorPos]
	}
	return cursorPos
}

// rebuildFilteredIndex applies the current filter to build the filtered index
func (m *ImportModal) rebuildFilteredIndex() {
	if !m.filter.Applied() {
		m.filteredIdx = nil
		return
	}

	m.filteredIdx = make([]int, 0)
	for i, s := range m.suggestions {
		if m.filter.MatchesAny(s.Label, s.Description) {
			m.filteredIdx = append(m.filteredIdx, i)
		}
	}

	// Adjust cursor if it's now outside filtered range
	if len(m.filteredIdx) > 0 && m.selectedIdx >= len(m.filteredIdx) {
		m.selectedIdx = len(m.filteredIdx) - 1
	}
}

// ensureSelectedVisible adjusts scroll offset to keep the selected suggestion visible
func (m *ImportModal) ensureSelectedVisible() {
	suggestionCount := m.effectiveSuggestionCount()
	if suggestionCount <= maxVisibleSuggestions {
		return // No scrolling needed
	}

	scrollOffset := m.ScrollOffset()

	// If selected is above visible area, scroll up
	if m.selectedIdx < scrollOffset {
		m.SetScrollOffset(m.selectedIdx)
		return
	}

	// If selected is below visible area, scroll down
	if m.selectedIdx >= scrollOffset+maxVisibleSuggestions {
		m.SetScrollOffset(m.selectedIdx - maxVisibleSuggestions + 1)
	}
}

func (m *ImportModal) handleEnterKey() (confirmed bool) {
	suggestionCount := m.effectiveSuggestionCount()
	if suggestionCount > 0 && m.showSuggestions {
		idx := m.effectiveSuggestionIndex(m.selectedIdx)
		if idx >= 0 && idx < len(m.suggestions) {
			m.input.SetValue(m.suggestions[idx].ID)
		}
		m.showSuggestions = false
		m.filter.Deactivate()
		m.filteredIdx = nil
		return false
	}
	if m.GetImportID() != "" {
		m.ModalBase.Hide()
		m.input.Blur()
		m.filter.Deactivate()
		return true
	}
	return false
}

func (m *ImportModal) handleNavigationKey(direction, pageSize int) {
	suggestionCount := m.effectiveSuggestionCount()
	if suggestionCount == 0 || !m.showSuggestions {
		return
	}
	m.selectedIdx += direction * pageSize
	m.selectedIdx = m.clampSelectedIndex(pageSize)
	m.ensureSelectedVisible()
}

func (m *ImportModal) clampSelectedIndex(pageSize int) int {
	suggestionCount := m.effectiveSuggestionCount()
	wrapAround := pageSize == 1
	if m.selectedIdx < 0 {
		if wrapAround {
			return suggestionCount - 1
		}
		return 0
	}
	if m.selectedIdx >= suggestionCount {
		if wrapAround {
			return 0
		}
		return suggestionCount - 1
	}
	return m.selectedIdx
}

func (m *ImportModal) handleEscapeKey() {
	// If filter is active, exit filter mode but keep filter applied
	if m.filter.Active() {
		m.filter.Deactivate()
		return
	}
	if m.showSuggestions {
		m.showSuggestions = false
		m.filteredIdx = nil
		return
	}
	m.ModalBase.Hide()
	m.input.Blur()
}

// Update handles key events and returns true if import was confirmed, false if cancelled
func (m *ImportModal) Update(msg tea.KeyMsg) (confirmed bool, cmd tea.Cmd) {
	if !m.Visible() {
		return false, nil
	}

	// Handle filter activation with "/" when suggestions are showing
	if key.Matches(msg, Keys.Filter) && m.showSuggestions && !m.filter.Active() {
		m.filter.Activate()
		m.rebuildFilteredIndex()
		return false, nil
	}

	// Forward to filter if active
	if m.filter.Active() {
		cmd, handled := m.filter.Update(msg)
		if handled {
			m.rebuildFilteredIndex()
			return false, cmd
		}
	}

	switch msg.String() {
	case "enter":
		return m.handleEnterKey(), nil
	case "up":
		m.handleNavigationKey(-1, 1)
		return false, nil
	case "down":
		m.handleNavigationKey(1, 1)
		return false, nil
	case "pgup":
		m.handleNavigationKey(-1, 8)
		return false, nil
	case "pgdown":
		m.handleNavigationKey(1, 8)
		return false, nil
	case "tab":
		if len(m.suggestions) > 0 {
			m.showSuggestions = !m.showSuggestions
			if !m.showSuggestions {
				m.filter.Deactivate()
				m.filteredIdx = nil
			}
		}
		return false, nil
	}

	if key.Matches(msg, Keys.Escape) {
		m.handleEscapeKey()
		return false, nil
	}

	m.input, cmd = m.input.Update(msg)
	return false, cmd
}

// renderSuggestionsSection renders the suggestions list with scrolling and filtering
func (m *ImportModal) renderSuggestionsSection(content *strings.Builder) {
	content.WriteString(LabelStyle.Render("Suggestions"))

	if m.loadingSuggestions {
		content.WriteString("\n")
		content.WriteString(DimStyle.Render("  Loading..."))
		return
	}

	if len(m.suggestions) == 0 {
		content.WriteString("\n")
		content.WriteString(DimStyle.Render("  No suggestions available"))
		return
	}

	suggestionCount := m.effectiveSuggestionCount()
	if m.filter.Applied() && suggestionCount == 0 {
		m.renderFilterNoMatches(content)
		return
	}

	m.renderSuggestionsList(content, suggestionCount)
}

// renderFilterNoMatches renders the "No matches" state when filter has no results
func (m *ImportModal) renderFilterNoMatches(content *strings.Builder) {
	content.WriteString("\n")
	content.WriteString(DimStyle.Render("  No matches"))
	content.WriteString("\n")
	content.WriteString(RenderFilterBar(&m.filter, 0, len(m.suggestions), m.Width()))
}

// renderSuggestionsList renders the scrollable list of suggestions
func (m *ImportModal) renderSuggestionsList(content *strings.Builder, suggestionCount int) {
	scrollOffset := m.clampScrollOffset(suggestionCount)

	// Add scroll indicator to header if needed
	if suggestionCount > maxVisibleSuggestions {
		endIdx := min(scrollOffset+maxVisibleSuggestions, suggestionCount)
		content.WriteString(DimStyle.Render(fmt.Sprintf(" [%d-%d/%d]", scrollOffset+1, endIdx, suggestionCount)))
	}
	content.WriteString("\n")

	// Render visible suggestions
	endIdx := min(scrollOffset+maxVisibleSuggestions, suggestionCount)
	for i := scrollOffset; i < endIdx; i++ {
		m.renderSuggestionItem(content, i)
	}

	// Scroll hints
	if suggestionCount > maxVisibleSuggestions {
		maxOffset := max(suggestionCount-maxVisibleSuggestions, 0)
		if hint := RenderScrollHint(scrollOffset > 0, scrollOffset < maxOffset, "  "); hint != "" {
			content.WriteString(hint)
			content.WriteString("\n")
		}
	}

	// Add filter bar if active or applied
	if m.filter.Active() || m.filter.Applied() {
		content.WriteString(RenderFilterBar(&m.filter, suggestionCount, len(m.suggestions), m.Width()))
		content.WriteString("\n")
	}
}

// clampScrollOffset ensures scroll offset is within valid bounds and returns the clamped value
func (m *ImportModal) clampScrollOffset(suggestionCount int) int {
	maxOffset := max(suggestionCount-maxVisibleSuggestions, 0)
	scrollOffset := m.ScrollOffset()
	if scrollOffset > maxOffset {
		scrollOffset = maxOffset
		m.SetScrollOffset(scrollOffset)
	}
	if scrollOffset < 0 {
		scrollOffset = 0
		m.SetScrollOffset(scrollOffset)
	}
	return scrollOffset
}

// renderSuggestionItem renders a single suggestion item
func (m *ImportModal) renderSuggestionItem(content *strings.Builder, i int) {
	idx := m.effectiveSuggestionIndex(i)
	if idx < 0 || idx >= len(m.suggestions) {
		return
	}
	s := m.suggestions[idx]
	if i == m.selectedIdx && m.showSuggestions {
		content.WriteString(ValueStyle.Render("> " + s.Label))
	} else {
		content.WriteString(DimStyle.Render("  " + s.Label))
	}
	if s.Description != "" {
		content.WriteString(DimStyle.Render(" - " + s.Description))
	}
	if s.PluginName != "" {
		content.WriteString(DimStyle.Render(" [" + s.PluginName + "]"))
	}
	content.WriteString("\n")
}

// View renders the import modal
func (m *ImportModal) View() string {
	title := DialogTitleStyle.Render("Import Resource")

	var content strings.Builder

	// Resource info (always visible, not scrolled)
	content.WriteString(DimStyle.Render("Type: "))
	content.WriteString(ValueStyle.Render(m.resourceType))
	content.WriteString("\n")

	content.WriteString(DimStyle.Render("Name: "))
	content.WriteString(ValueStyle.Render(m.resourceName))
	content.WriteString("\n\n")

	// Suggestions section with scrolling
	m.renderSuggestionsSection(&content)
	content.WriteString("\n")

	// Import ID input (always visible, not scrolled)
	content.WriteString(LabelStyle.Render("Import ID"))
	content.WriteString("\n")
	content.WriteString(m.input.View())

	// Error if any
	if m.err != nil {
		content.WriteString("\n\n")
		content.WriteString(ErrorStyle.Render(m.err.Error()))
	}

	// Footer hints
	footer := DimStyle.Render("\ntab suggestions  enter select/confirm  esc cancel")

	dialog := DialogStyle.Render(lipgloss.JoinVertical(lipgloss.Left, title, content.String(), footer))
	return m.CenterDialog(dialog)
}
