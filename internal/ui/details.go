package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// DetailPanel is a floating panel showing resource details
type DetailPanel struct {
	PanelBase // Embed common panel functionality

	// Position of panel in terminal (for mouse coordinate mapping)
	posX, posY int

	// Current resource being displayed
	resource *ResourceItem

	// Text selection state
	selection *TextSelection

	// Cache of plain text lines for selection extraction
	plainTextLines []string
}

// NewDetailPanel creates a new detail panel component
func NewDetailPanel() *DetailPanel {
	return &DetailPanel{
		selection: NewTextSelection(),
	}
}

// SetPosition sets the position of the panel in terminal coordinates
func (d *DetailPanel) SetPosition(x, y int) {
	d.posX = x
	d.posY = y
	// Update selection bounds - content area is inside border and padding
	// Border: 1, Padding: 1 top/bottom, 2 left/right
	contentX := x + 3               // 1 border + 2 padding
	contentY := y + 2               // 1 border + 1 padding
	contentWidth := d.Width() - 6   // subtract both sides
	contentHeight := d.Height() - 4 // subtract top and bottom
	d.selection.SetBounds(contentX, contentY, contentWidth, contentHeight)
}

// HandleMouseEvent handles mouse events for text selection
// Returns a command if text was copied to clipboard
func (d *DetailPanel) HandleMouseEvent(msg tea.MouseMsg) tea.Cmd {
	if !d.Visible() {
		return nil
	}

	// Check if click is within panel bounds
	inBounds := msg.X >= d.posX && msg.X < d.posX+d.Width() &&
		msg.Y >= d.posY && msg.Y < d.posY+d.Height()

	switch msg.Action {
	case tea.MouseActionPress:
		if msg.Button == tea.MouseButtonLeft && inBounds {
			d.selection.StartSelection(msg.X, msg.Y)
		}

	case tea.MouseActionMotion:
		if d.selection.Active() && msg.Button == tea.MouseButtonLeft {
			d.selection.UpdateSelection(msg.X, msg.Y)
		}

	case tea.MouseActionRelease:
		if msg.Button == tea.MouseButtonLeft && d.selection.Active() {
			d.selection.EndSelection(msg.X, msg.Y)
			// Copy selected text to clipboard
			if d.selection.HasSelection() {
				text := d.getSelectedText()
				if text != "" {
					return CopyToClipboardCmd(text)
				}
			}
		}
	}

	return nil
}

// ClearSelection clears the current text selection
func (d *DetailPanel) ClearSelection() {
	d.selection.Clear()
}

// HasSelection returns true if there is an active text selection
func (d *DetailPanel) HasSelection() bool {
	return d.selection.HasSelection() || d.selection.Active()
}

// getSelectedText extracts the selected text from the panel content
func (d *DetailPanel) getSelectedText() string {
	if len(d.plainTextLines) == 0 {
		return ""
	}
	// The plain text lines start after header (2 lines: header + blank)
	// and are offset by scroll
	return d.selection.ExtractSelectedText(d.plainTextLines, 0)
}

// SetResource sets the resource to display details for
func (d *DetailPanel) SetResource(resource *ResourceItem) {
	d.resource = resource
	d.ResetScroll()
}

// View renders the detail panel
func (d *DetailPanel) View() string {
	if !d.Visible() || d.Width() == 0 || d.Height() == 0 {
		return ""
	}

	// Build header with resource name
	header := "Details"
	if d.resource != nil {
		header = d.resource.Name
	}

	// Build unified content
	var content string
	if d.resource == nil {
		content = DimStyle.Render("No resource selected")
	} else {
		content = d.renderUnified()
	}

	// Use shared helper for common panel rendering
	result := RenderDetailPanel(DetailPanelContent{
		Header:       header,
		Content:      content,
		Width:        d.Width(),
		Height:       d.Height(),
		ScrollOffset: d.ScrollOffset(),
	})

	// Update scroll offset if it was clamped
	if result.NewScrollOffset != d.ScrollOffset() {
		d.SetScrollOffset(result.NewScrollOffset)
	}

	// Cache plain text lines for selection extraction
	// Strip ANSI codes for accurate text extraction
	d.plainTextLines = make([]string, len(result.VisibleLines))
	for i, line := range result.VisibleLines {
		d.plainTextLines[i] = stripAnsi(line)
	}

	// If there's a selection, we need to re-render with highlighting applied
	if d.selection.HasSelection() || d.selection.Active() {
		highlightedLines := make([]string, len(result.VisibleLines))
		for i, line := range result.VisibleLines {
			highlightedLines[i] = d.applySelectionToLine(line, i)
		}
		// Re-render with highlighted content
		highlightedContent := strings.Join(highlightedLines, "\n")
		result = RenderDetailPanel(DetailPanelContent{
			Header:       header,
			Content:      highlightedContent,
			Width:        d.Width(),
			Height:       d.Height(),
			ScrollOffset: 0, // Content is already the visible portion
		})
	}

	return result.Rendered
}

// applySelectionToLine applies selection highlighting to a single line
func (d *DetailPanel) applySelectionToLine(line string, row int) string {
	// Work with runes to handle the line character by character
	runes := []rune(line)
	var result strings.Builder
	col := 0

	for _, r := range runes {
		charWidth := 1
		// Check if this position is selected
		if d.selection.IsPositionSelected(col, row) {
			result.WriteString(TextSelectionStyle.Render(string(r)))
		} else {
			result.WriteRune(r)
		}
		col += charWidth
	}

	return result.String()
}

// stripAnsi removes ANSI escape sequences from a string
func stripAnsi(s string) string {
	var result strings.Builder
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			// Check for end of escape sequence
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || r == '~' {
				inEscape = false
			}
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}

// renderUnified renders a unified view with metadata and combined inputs/outputs diff
func (d *DetailPanel) renderUnified() string {
	if d.resource == nil {
		return DimStyle.Render("No resource selected")
	}

	var b strings.Builder
	maxWidth := d.Width() - 8

	// Compact metadata header
	b.WriteString(DimStyle.Render("Type: "))
	b.WriteString(ValueStyle.Render(d.resource.Type))
	b.WriteString("\n")

	// Operation and status on same line
	b.WriteString(DimStyle.Render("Op: "))
	b.WriteString(RenderOp(d.resource.Op))
	if d.resource.Status != StatusNone {
		b.WriteString("  ")
		b.WriteString(DimStyle.Render("Status: "))
		b.WriteString(RenderStatus(d.resource.Status))
		if d.resource.Status == StatusRunning && d.resource.CurrentOp != "" {
			b.WriteString(" (")
			b.WriteString(RenderOp(d.resource.CurrentOp))
			b.WriteString(")")
		}
	}
	b.WriteString("\n")

	// Combined properties section
	b.WriteString("\n")
	b.WriteString(DimStyle.Render("─── Properties ───"))
	b.WriteString("\n\n")

	// Use the DiffRenderer for property rendering
	renderer := NewDiffRenderer(maxWidth)
	b.WriteString(renderer.RenderCombinedProperties(d.resource))

	return b.String()
}
