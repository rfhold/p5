package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ErrorModal is a modal dialog for displaying detailed error information
type ErrorModal struct {
	ModalBase // Embedded modal base for common functionality

	// Dialog content
	title   string
	summary string // Brief error summary
	details string // Full error details (scrollable)

	// Viewport for scrollable details
	viewport viewport.Model
}

// NewErrorModal creates a new error modal
func NewErrorModal() *ErrorModal {
	vp := viewport.New(60, 10)
	vp.Style = lipgloss.NewStyle().
		Foreground(ColorText)

	return &ErrorModal{
		viewport: vp,
	}
}

// SetSize sets the dialog dimensions for centering and sizes the viewport
func (m *ErrorModal) SetSize(width, height int) {
	m.ModalBase.SetSize(width, height)

	// Size the viewport to fit within the dialog
	// Account for dialog padding, title, summary, and footer
	dialogWidth := min(width-4, DefaultDialogMaxWidth)
	dialogHeight := min(height-4, DefaultDialogMaxHeight)
	contentWidth := dialogWidth - DialogPaddingAllowance
	contentHeight := dialogHeight - DialogChromeAllowance

	if contentWidth < MinContentWidth {
		contentWidth = MinContentWidth
	}
	if contentHeight < MinContentHeight {
		contentHeight = MinContentHeight
	}

	m.viewport.Width = contentWidth
	m.viewport.Height = contentHeight
}

// Show shows the error modal with the given content
func (m *ErrorModal) Show(title, summary, details string) {
	m.title = title
	m.summary = summary
	m.details = details
	m.ModalBase.Show()

	// Set viewport content
	m.viewport.SetContent(details)
	m.viewport.GotoTop()
}

// Hide is inherited from ModalBase

// Visible is inherited from ModalBase

// Update handles key events
func (m *ErrorModal) Update(msg tea.KeyMsg) (dismissed bool, cmd tea.Cmd) {
	if !m.Visible() {
		return false, nil
	}

	switch {
	case key.Matches(msg, Keys.Escape), msg.String() == "enter", msg.String() == "q":
		m.ModalBase.Hide()
		return true, nil

	case key.Matches(msg, Keys.Up), msg.String() == "k":
		m.viewport.LineUp(1)

	case key.Matches(msg, Keys.Down), msg.String() == "j":
		m.viewport.LineDown(1)

	case key.Matches(msg, Keys.PageUp):
		m.viewport.HalfViewUp()

	case key.Matches(msg, Keys.PageDown):
		m.viewport.HalfViewDown()

	case msg.String() == "g":
		m.viewport.GotoTop()

	case msg.String() == "G":
		m.viewport.GotoBottom()
	}

	return false, nil
}

// View renders the error modal
func (m *ErrorModal) View() string {
	// Title
	titleStyle := DialogTitleStyle.Copy().Foreground(ColorError)
	title := titleStyle.Render(m.title)

	// Summary
	summaryStyle := lipgloss.NewStyle().
		Foreground(ColorText).
		MarginBottom(1)
	summary := summaryStyle.Render(m.summary)

	// Details label
	detailsLabel := DimStyle.Render("Details:")

	// Viewport with border
	viewportStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorDim).
		Padding(0, 1)

	viewportContent := viewportStyle.Render(m.viewport.View())

	// Scroll indicator
	scrollInfo := ""
	if m.viewport.TotalLineCount() > m.viewport.Height {
		percent := int(m.viewport.ScrollPercent() * 100)
		scrollInfo = DimStyle.Render(strings.Repeat(" ", m.viewport.Width-10)) +
			DimStyle.Render("[") +
			ValueStyle.Render(strings.Repeat("j", 1)) +
			DimStyle.Render("/") +
			ValueStyle.Render(strings.Repeat("k", 1)) +
			DimStyle.Render(" scroll ") +
			ValueStyle.Render(fmt.Sprintf("%d", percent)) +
			DimStyle.Render("%]")
	}

	// Footer hints
	footer := DimStyle.Render("\nenter/esc dismiss  j/k scroll  g/G top/bottom")

	// Combine all parts
	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		summary,
		"",
		detailsLabel,
		viewportContent,
		scrollInfo,
		footer,
	)

	errorDialogStyle := DialogStyle.Copy().BorderForeground(ColorError)
	return m.RenderDialogWithStyle(errorDialogStyle, content)
}
