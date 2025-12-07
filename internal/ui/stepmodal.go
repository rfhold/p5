package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// StepModalAction represents an action taken by the user in a step modal
type StepModalAction int

const (
	StepModalActionNone    StepModalAction = iota
	StepModalActionNext                    // Move to next step
	StepModalActionPrev                    // Move to previous step
	StepModalActionConfirm                 // Confirm final step
	StepModalActionCancel                  // Cancel the modal
)

// InfoLine represents a key-value pair displayed in the modal header
type InfoLine struct {
	Label string
	Value string
}

// StepSuggestion represents a selectable suggestion item
type StepSuggestion struct {
	ID          string
	Label       string
	Description string
	Source      string // e.g., plugin name or "from Pulumi.dev.yaml"
	Warning     string // Per-item warning (e.g., "has existing encryption")
}

// StepModalStep defines the configuration for a single step in the modal
type StepModalStep struct {
	Title            string
	InfoLines        []InfoLine       // Read-only info display at top
	Suggestions      []StepSuggestion // Selectable list items
	InputLabel       string           // Label for text input
	InputPlaceholder string
	Warning          string // Warning message (shown in yellow)
	FooterHints      string // Custom footer hints
	PasswordMode     bool   // Mask input like a password
}

// StepModal is a multi-step modal dialog with navigation support
type StepModal struct {
	ModalBase

	title       string
	steps       []StepModalStep
	currentStep int

	input           textinput.Model
	selectedIdx     int
	showSuggestions bool
	scrollOffset    int

	// Results collected from each step
	results map[int]string // step index -> selected/entered value

	// Error state
	err error
}

// NewStepModal creates a new step modal
func NewStepModal(title string) *StepModal {
	ti := textinput.New()
	ti.CharLimit = 256
	ti.Width = DefaultInputWidth

	return &StepModal{
		title:   title,
		input:   ti,
		results: make(map[int]string),
	}
}

// SetSteps configures the modal steps
func (m *StepModal) SetSteps(steps []StepModalStep) {
	m.steps = steps
	m.currentStep = 0
	m.results = make(map[int]string)
	m.updateInputForCurrentStep()
}

// SetStepSuggestions sets suggestions for a specific step
func (m *StepModal) SetStepSuggestions(step int, suggestions []StepSuggestion) {
	if step >= 0 && step < len(m.steps) {
		m.steps[step].Suggestions = suggestions
		if step == m.currentStep {
			m.selectedIdx = 0
			m.scrollOffset = 0
			m.showSuggestions = len(suggestions) > 0
		}
	}
}

// SetStepWarning sets a warning for a specific step
func (m *StepModal) SetStepWarning(step int, warning string) {
	if step >= 0 && step < len(m.steps) {
		m.steps[step].Warning = warning
	}
}

// SetStepInfoLines sets info lines for a specific step
func (m *StepModal) SetStepInfoLines(step int, lines []InfoLine) {
	if step >= 0 && step < len(m.steps) {
		m.steps[step].InfoLines = lines
	}
}

// CurrentStep returns the current step index
func (m *StepModal) CurrentStep() int {
	return m.currentStep
}

// IsLastStep returns true if on the final step
func (m *StepModal) IsLastStep() bool {
	return m.currentStep >= len(m.steps)-1
}

// GetResult returns the collected result for a specific step
func (m *StepModal) GetResult(step int) string {
	return m.results[step]
}

// SetResult sets a result for a specific step (useful for pre-populating)
func (m *StepModal) SetResult(step int, value string) {
	m.results[step] = value
}

// SetError sets an error to display
func (m *StepModal) SetError(err error) {
	m.err = err
}

// ClearError clears any displayed error
func (m *StepModal) ClearError() {
	m.err = nil
}

// Show shows the modal and resets to first step
func (m *StepModal) Show() {
	m.ModalBase.Show()
	m.currentStep = 0
	m.results = make(map[int]string)
	m.err = nil
	m.updateInputForCurrentStep()
}

// NextStep advances to the next step, saving the current result
func (m *StepModal) NextStep() bool {
	m.saveCurrentResult()

	if m.currentStep < len(m.steps)-1 {
		m.currentStep++
		m.updateInputForCurrentStep()
		return true
	}
	return false
}

// PrevStep goes back to the previous step
func (m *StepModal) PrevStep() bool {
	if m.currentStep > 0 {
		m.currentStep--
		m.updateInputForCurrentStep()
		return true
	}
	return false
}

// saveCurrentResult saves the current input/selection to results
func (m *StepModal) saveCurrentResult() {
	if m.currentStep < 0 || m.currentStep >= len(m.steps) {
		return
	}

	step := m.steps[m.currentStep]
	value := strings.TrimSpace(m.input.Value())

	// If we have suggestions and one is selected, use it
	if len(step.Suggestions) > 0 && m.showSuggestions && value == "" {
		if m.selectedIdx >= 0 && m.selectedIdx < len(step.Suggestions) {
			value = step.Suggestions[m.selectedIdx].ID
		}
	}

	m.results[m.currentStep] = value
}

// updateInputForCurrentStep resets input state for the current step
func (m *StepModal) updateInputForCurrentStep() {
	if m.currentStep < 0 || m.currentStep >= len(m.steps) {
		return
	}

	step := m.steps[m.currentStep]

	// Configure input for current step
	m.input.Placeholder = step.InputPlaceholder
	if step.PasswordMode {
		m.input.EchoMode = textinput.EchoPassword
	} else {
		m.input.EchoMode = textinput.EchoNormal
	}

	// Restore previous value if we have one, otherwise clear
	if prev, ok := m.results[m.currentStep]; ok {
		m.input.SetValue(prev)
	} else {
		m.input.SetValue("")
	}

	m.input.Focus()
	m.selectedIdx = 0
	m.scrollOffset = 0
	m.showSuggestions = len(step.Suggestions) > 0
	m.err = nil
}

// maxVisibleStepSuggestions is the max number of suggestions shown at once
const maxVisibleStepSuggestions = 6

// ensureSelectedVisible adjusts scroll offset to keep the selected suggestion visible
func (m *StepModal) ensureSelectedVisible() {
	if m.currentStep < 0 || m.currentStep >= len(m.steps) {
		return
	}
	suggestions := m.steps[m.currentStep].Suggestions
	if len(suggestions) <= maxVisibleStepSuggestions {
		return
	}

	if m.selectedIdx < m.scrollOffset {
		m.scrollOffset = m.selectedIdx
	}
	if m.selectedIdx >= m.scrollOffset+maxVisibleStepSuggestions {
		m.scrollOffset = m.selectedIdx - maxVisibleStepSuggestions + 1
	}
}

func (m *StepModal) handleEnterKey(step StepModalStep) StepModalAction {
	m.saveCurrentResult()
	if m.results[m.currentStep] == "" && len(step.Suggestions) > 0 && m.showSuggestions {
		if m.selectedIdx >= 0 && m.selectedIdx < len(step.Suggestions) {
			m.results[m.currentStep] = step.Suggestions[m.selectedIdx].ID
		}
	}

	if m.results[m.currentStep] == "" {
		return StepModalActionNone
	}

	if m.IsLastStep() {
		return StepModalActionConfirm
	}
	m.NextStep()
	return StepModalActionNext
}

func (m *StepModal) handleNavigationKey(step StepModalStep, direction int) {
	if len(step.Suggestions) == 0 || !m.showSuggestions {
		return
	}
	m.selectedIdx += direction
	if m.selectedIdx < 0 {
		m.selectedIdx = len(step.Suggestions) - 1
	} else if m.selectedIdx >= len(step.Suggestions) {
		m.selectedIdx = 0
	}
	m.ensureSelectedVisible()
}

func (m *StepModal) handleEscapeKey() StepModalAction {
	if m.showSuggestions && m.input.Value() != "" {
		m.showSuggestions = false
		return StepModalActionNone
	}
	m.Hide()
	return StepModalActionCancel
}

func (m *StepModal) handleTextInput(msg tea.KeyMsg, step StepModalStep) tea.Cmd {
	var inputCmd tea.Cmd
	m.input, inputCmd = m.input.Update(msg)
	if len(step.Suggestions) > 0 && m.input.Value() == "" {
		m.showSuggestions = true
	}
	return inputCmd
}

// Update handles key events and returns the action taken
func (m *StepModal) Update(msg tea.KeyMsg) (StepModalAction, tea.Cmd) {
	if !m.Visible() || m.currentStep < 0 || m.currentStep >= len(m.steps) {
		return StepModalActionNone, nil
	}

	step := m.steps[m.currentStep]

	switch msg.String() {
	case "enter":
		return m.handleEnterKey(step), nil
	case "backspace":
		if m.input.Value() == "" && m.currentStep > 0 {
			m.PrevStep()
			return StepModalActionPrev, nil
		}
	case "up":
		m.handleNavigationKey(step, -1)
		return StepModalActionNone, nil
	case "down":
		m.handleNavigationKey(step, 1)
		return StepModalActionNone, nil
	case "tab":
		if len(step.Suggestions) > 0 {
			m.showSuggestions = !m.showSuggestions
		}
		return StepModalActionNone, nil
	}

	if key.Matches(msg, Keys.Escape) {
		return m.handleEscapeKey(), nil
	}

	return StepModalActionNone, m.handleTextInput(msg, step)
}

// View renders the step modal
func (m *StepModal) View() string {
	if m.currentStep < 0 || m.currentStep >= len(m.steps) {
		return ""
	}

	step := m.steps[m.currentStep]

	// Title with step indicator
	stepIndicator := ""
	if len(m.steps) > 1 {
		stepIndicator = DimStyle.Render(fmt.Sprintf(" (%d/%d)", m.currentStep+1, len(m.steps)))
	}
	title := DialogTitleStyle.Render(m.title) + stepIndicator

	var content strings.Builder

	// Step title
	if step.Title != "" {
		content.WriteString(LabelStyle.Render(step.Title))
		content.WriteString("\n\n")
	}

	// Info lines
	if len(step.InfoLines) > 0 {
		for _, info := range step.InfoLines {
			content.WriteString(DimStyle.Render(info.Label + ": "))
			content.WriteString(ValueStyle.Render(info.Value))
			content.WriteString("\n")
		}
		content.WriteString("\n")
	}

	// Warning (if any)
	if step.Warning != "" {
		warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#e0af68")) // yellow/orange
		content.WriteString(warningStyle.Render("! " + step.Warning))
		content.WriteString("\n\n")
	}

	// Suggestions section
	if len(step.Suggestions) > 0 {
		m.renderSuggestionsSection(&content, step.Suggestions)
	}

	// Input section
	if step.InputLabel != "" {
		content.WriteString(LabelStyle.Render(step.InputLabel))
		content.WriteString("\n")
	}
	content.WriteString(m.input.View())

	// Error if any
	if m.err != nil {
		content.WriteString("\n\n")
		content.WriteString(ErrorStyle.Render(m.err.Error()))
	}

	// Footer hints
	footer := m.buildFooterHints(step)

	dialog := DialogStyle.Render(lipgloss.JoinVertical(lipgloss.Left, title, content.String(), footer))
	return m.CenterDialog(dialog)
}

func (m *StepModal) buildFooterHints(step StepModalStep) string {
	if step.FooterHints != "" {
		return DimStyle.Render("\n" + step.FooterHints)
	}

	var hints []string
	if len(step.Suggestions) > 0 {
		hints = append(hints, "tab suggestions")
	}
	if m.IsLastStep() {
		hints = append(hints, "enter confirm")
	} else {
		hints = append(hints, "enter next")
	}
	if m.currentStep > 0 {
		hints = append(hints, "backspace back")
	}
	hints = append(hints, "esc cancel")
	return DimStyle.Render("\n" + strings.Join(hints, "  "))
}

func (m *StepModal) renderSuggestionsSection(content *strings.Builder, suggestions []StepSuggestion) {
	totalSuggestions := len(suggestions)
	maxOffset := max(totalSuggestions-maxVisibleStepSuggestions, 0)

	m.clampScrollOffset(maxOffset)

	if totalSuggestions > maxVisibleStepSuggestions {
		endIdx := min(m.scrollOffset+maxVisibleStepSuggestions, totalSuggestions)
		content.WriteString(DimStyle.Render(fmt.Sprintf("[%d-%d/%d]", m.scrollOffset+1, endIdx, totalSuggestions)))
		content.WriteString("\n")
	}

	m.renderVisibleSuggestions(content, suggestions)

	if totalSuggestions > maxVisibleStepSuggestions {
		if hint := RenderScrollHint(m.scrollOffset > 0, m.scrollOffset < maxOffset, "  "); hint != "" {
			content.WriteString(hint)
			content.WriteString("\n")
		}
	}
	content.WriteString("\n")
}

func (m *StepModal) clampScrollOffset(maxOffset int) {
	if m.scrollOffset > maxOffset {
		m.scrollOffset = maxOffset
	}
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
}

func (m *StepModal) renderVisibleSuggestions(content *strings.Builder, suggestions []StepSuggestion) {
	endIdx := min(m.scrollOffset+maxVisibleStepSuggestions, len(suggestions))
	warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#e0af68"))

	for i := m.scrollOffset; i < endIdx; i++ {
		s := suggestions[i]
		m.renderSuggestionLine(content, s, i, warningStyle)
	}
}

func (m *StepModal) renderSuggestionLine(content *strings.Builder, s StepSuggestion, idx int, warningStyle lipgloss.Style) {
	if idx == m.selectedIdx && m.showSuggestions {
		content.WriteString(ValueStyle.Render("> " + s.Label))
	} else {
		content.WriteString(DimStyle.Render("  " + s.Label))
	}
	if s.Description != "" {
		content.WriteString(DimStyle.Render(" - " + s.Description))
	}
	if s.Source != "" {
		content.WriteString(DimStyle.Render(" [" + s.Source + "]"))
	}
	if s.Warning != "" {
		content.WriteString(warningStyle.Render(" !" + s.Warning))
	}
	content.WriteString("\n")
}
