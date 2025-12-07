package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// HelpItem represents a single help entry
type HelpItem struct {
	Key  string
	Desc string
}

// HelpDialog renders a help overlay
type HelpDialog struct {
	items    []HelpItem
	width    int
	height   int
	viewport viewport.Model
	ready    bool
}

// NewHelpDialog creates a new help dialog
func NewHelpDialog() *HelpDialog {
	return &HelpDialog{
		items: []HelpItem{
			// Navigation
			{Key: "", Desc: "Navigation"},
			{Key: "↑/k", Desc: "Move up"},
			{Key: "↓/j", Desc: "Move down"},
			{Key: "pgup", Desc: "Page up"},
			{Key: "pgdn", Desc: "Page down"},
			{Key: "g", Desc: "Go to top"},
			{Key: "G", Desc: "Go to bottom"},
			{Key: "", Desc: ""},

			// Selection
			{Key: "", Desc: "Selection"},
			{Key: "v", Desc: "Visual select mode"},
			{Key: "T", Desc: "Toggle target flag"},
			{Key: "R", Desc: "Toggle replace flag"},
			{Key: "E", Desc: "Toggle exclude flag"},
			{Key: "c", Desc: "Clear flags on selection"},
			{Key: "esc", Desc: "Cancel selection / back"},
			{Key: "", Desc: ""},

			// Operations
			{Key: "", Desc: "Operations"},
			{Key: "u", Desc: "Preview up"},
			{Key: "r", Desc: "Preview refresh"},
			{Key: "d", Desc: "Preview destroy"},
			{Key: "ctrl+u", Desc: "Execute up"},
			{Key: "ctrl+r", Desc: "Execute refresh"},
			{Key: "ctrl+d", Desc: "Execute destroy"},
			{Key: "I", Desc: "Import resource (in preview)"},
			{Key: "", Desc: ""},

			// General
			{Key: "", Desc: "General"},
			{Key: "s", Desc: "Select stack"},
			{Key: "w", Desc: "Select workspace"},
			{Key: "h", Desc: "View stack history"},
			{Key: "D", Desc: "Toggle details panel"},
			{Key: "?", Desc: "Toggle help"},
			{Key: "q", Desc: "Quit"},
		},
	}
}

// SetSize sets the dialog dimensions for centering
func (h *HelpDialog) SetSize(width, height int) {
	h.width = width
	h.height = height

	// Build content first to measure it
	content := h.buildContent()
	contentLines := strings.Count(content, "\n") + 1

	// Calculate how much space the dialog chrome takes: border, padding, title, scroll indicator, and screen margins.
	dialogChrome := 11

	// Maximum viewport height that will fit on screen
	maxVpHeight := height - dialogChrome
	if maxVpHeight < 3 {
		maxVpHeight = 3
	}

	// Viewport height is the smaller of content height or max available
	vpHeight := contentLines
	if vpHeight > maxVpHeight {
		vpHeight = maxVpHeight
	}

	// Initialize or update viewport
	if !h.ready {
		h.viewport = viewport.New(40, vpHeight)
		h.viewport.SetContent(content)
		h.ready = true
	} else {
		h.viewport.Width = 40
		h.viewport.Height = vpHeight
		h.viewport.SetContent(content)
	}
}

// buildContent builds the help content string
func (h *HelpDialog) buildContent() string {
	var lines []string
	for _, item := range h.items {
		if item.Key == "" && item.Desc != "" {
			// Section header
			lines = append(lines, "")
			lines = append(lines, LabelStyle.Render(item.Desc))
		} else if item.Key == "" {
			// Empty line
			lines = append(lines, "")
		} else {
			lines = append(lines, fmt.Sprintf("  %s  %s",
				ValueStyle.Render(fmt.Sprintf("%8s", item.Key)),
				DimStyle.Render(item.Desc)))
		}
	}

	// Remove leading empty line if present
	if len(lines) > 0 && lines[0] == "" {
		lines = lines[1:]
	}

	return strings.Join(lines, "\n")
}

// Update handles key events for scrolling
func (h *HelpDialog) Update(msg tea.KeyMsg) {
	if !h.ready {
		return
	}
	h.viewport, _ = h.viewport.Update(msg)
}

// GotoTop scrolls to the top of the help content
func (h *HelpDialog) GotoTop() {
	if h.ready {
		h.viewport.SetYOffset(0)
	}
}

// GotoBottom scrolls to the bottom of the help content
func (h *HelpDialog) GotoBottom() {
	if h.ready {
		maxOffset := h.viewport.TotalLineCount() - h.viewport.Height
		if maxOffset < 0 {
			maxOffset = 0
		}
		h.viewport.SetYOffset(maxOffset)
	}
}

// View renders the help dialog centered on screen
func (h *HelpDialog) View() string {
	titleText := "Keyboard Shortcuts"

	var content string
	if h.ready {
		// Check if content is scrollable
		isScrollable := h.viewport.TotalLineCount() > h.viewport.Height
		canScrollUp := h.viewport.YOffset > 0
		canScrollDown := h.viewport.YOffset < h.viewport.TotalLineCount()-h.viewport.Height

		// Add line count hint to title if scrollable
		if isScrollable {
			startLine := h.viewport.YOffset + 1
			endLine := h.viewport.YOffset + h.viewport.Height
			if endLine > h.viewport.TotalLineCount() {
				endLine = h.viewport.TotalLineCount()
			}
			titleText += DimStyle.Render(fmt.Sprintf(" [%d-%d/%d]", startLine, endLine, h.viewport.TotalLineCount()))
		}

		// Build content with scroll hint at bottom only
		var parts []string
		parts = append(parts, h.viewport.View())

		// Add scroll hint at bottom (import modal style)
		if isScrollable {
			hint := RenderScrollHint(canScrollUp, canScrollDown, "      ")
			if hint != "" {
				parts = append(parts, hint)
			}
		}

		content = strings.Join(parts, "\n")
	} else {
		content = h.buildContent()
	}

	title := DialogTitleStyle.Render(titleText)
	dialog := DialogStyle.Render(lipgloss.JoinVertical(lipgloss.Left, title, content))

	return lipgloss.Place(h.width, h.height, lipgloss.Center, lipgloss.Center,
		dialog,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(ColorBg),
	)
}
