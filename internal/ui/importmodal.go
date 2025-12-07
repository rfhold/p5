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
}

// NewImportModal creates a new import modal
func NewImportModal() *ImportModal {
	ti := textinput.New()
	ti.Placeholder = "Enter import ID..."
	ti.CharLimit = 256
	ti.Width = DefaultInputWidth

	return &ImportModal{
		input: ti,
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
}

// SetLoadingSuggestions sets the loading state
func (m *ImportModal) SetLoadingSuggestions(loading bool) {
	m.loadingSuggestions = loading
}

// Hide hides the import modal
func (m *ImportModal) Hide() {
	m.ModalBase.Hide()
	m.input.Blur()
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

// ensureSelectedVisible adjusts scroll offset to keep the selected suggestion visible
func (m *ImportModal) ensureSelectedVisible() {
	if len(m.suggestions) <= maxVisibleSuggestions {
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
	if len(m.suggestions) > 0 && m.showSuggestions {
		m.input.SetValue(m.suggestions[m.selectedIdx].ID)
		m.showSuggestions = false
		return false
	}
	if m.GetImportID() != "" {
		m.ModalBase.Hide()
		m.input.Blur()
		return true
	}
	return false
}

func (m *ImportModal) handleNavigationKey(direction, pageSize int) {
	if len(m.suggestions) == 0 || !m.showSuggestions {
		return
	}
	m.selectedIdx += direction * pageSize
	m.selectedIdx = m.clampSelectedIndex(pageSize)
	m.ensureSelectedVisible()
}

func (m *ImportModal) clampSelectedIndex(pageSize int) int {
	wrapAround := pageSize == 1
	if m.selectedIdx < 0 {
		if wrapAround {
			return len(m.suggestions) - 1
		}
		return 0
	}
	if m.selectedIdx >= len(m.suggestions) {
		if wrapAround {
			return 0
		}
		return len(m.suggestions) - 1
	}
	return m.selectedIdx
}

func (m *ImportModal) handleEscapeKey() {
	if m.showSuggestions {
		m.showSuggestions = false
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
	content.WriteString(LabelStyle.Render("Suggestions"))
	switch {
	case m.loadingSuggestions:
		content.WriteString("\n")
		content.WriteString(DimStyle.Render("  Loading..."))
	case len(m.suggestions) == 0:
		content.WriteString("\n")
		content.WriteString(DimStyle.Render("  No suggestions available"))
	default:
		// Calculate visible suggestions based on scroll
		totalSuggestions := len(m.suggestions)

		// Clamp scroll offset
		maxOffset := max(totalSuggestions-maxVisibleSuggestions, 0)
		scrollOffset := m.ScrollOffset()
		if scrollOffset > maxOffset {
			scrollOffset = maxOffset
			m.SetScrollOffset(scrollOffset)
		}
		if scrollOffset < 0 {
			scrollOffset = 0
			m.SetScrollOffset(scrollOffset)
		}

		// Add scroll indicator to header if needed
		if totalSuggestions > maxVisibleSuggestions {
			endIdx := min(scrollOffset+maxVisibleSuggestions, totalSuggestions)
			content.WriteString(DimStyle.Render(fmt.Sprintf(" [%d-%d/%d]", scrollOffset+1, endIdx, totalSuggestions)))
		}
		content.WriteString("\n")

		// Render visible suggestions
		endIdx := min(scrollOffset+maxVisibleSuggestions, totalSuggestions)
		for i := scrollOffset; i < endIdx; i++ {
			s := m.suggestions[i]
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

		// Scroll hints
		if totalSuggestions > maxVisibleSuggestions {
			hint := RenderScrollHint(scrollOffset > 0, scrollOffset < maxOffset, "  ")
			if hint != "" {
				content.WriteString(hint)
				content.WriteString("\n")
			}
		}
	}
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
