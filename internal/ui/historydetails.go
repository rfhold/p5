package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// HistoryDetailPanel is a floating panel showing history update details
type HistoryDetailPanel struct {
	visible bool
	width   int
	height  int

	// Scroll state
	scrollOffset int

	// Current history item being displayed
	item *HistoryItem
}

// NewHistoryDetailPanel creates a new history detail panel component
func NewHistoryDetailPanel() *HistoryDetailPanel {
	return &HistoryDetailPanel{
		visible: false,
	}
}

// Visible returns whether the panel is visible
func (d *HistoryDetailPanel) Visible() bool {
	return d.visible
}

// Toggle toggles the panel visibility
func (d *HistoryDetailPanel) Toggle() {
	d.visible = !d.visible
	d.scrollOffset = 0
}

// Show shows the panel
func (d *HistoryDetailPanel) Show() {
	d.visible = true
	d.scrollOffset = 0
}

// Hide hides the panel
func (d *HistoryDetailPanel) Hide() {
	d.visible = false
}

// SetSize sets the dimensions for the panel
func (d *HistoryDetailPanel) SetSize(width, height int) {
	d.width = width
	d.height = height
}

// SetItem sets the history item to display details for
func (d *HistoryDetailPanel) SetItem(item *HistoryItem) {
	d.item = item
	d.scrollOffset = 0
}

// ScrollUp scrolls the content up
func (d *HistoryDetailPanel) ScrollUp(lines int) {
	d.scrollOffset -= lines
	if d.scrollOffset < 0 {
		d.scrollOffset = 0
	}
}

// ScrollDown scrolls the content down
func (d *HistoryDetailPanel) ScrollDown(lines int) {
	d.scrollOffset += lines
}

// View renders the history detail panel
func (d *HistoryDetailPanel) View() string {
	if !d.visible || d.width == 0 || d.height == 0 {
		return ""
	}

	// Panel takes up the right half of the screen
	panelWidth := d.width
	panelHeight := d.height

	// Build header
	var header string
	if d.item != nil {
		header = LabelStyle.Render(fmt.Sprintf("Update #%d", d.item.Version))
	} else {
		header = LabelStyle.Render("Update Details")
	}

	// Build content
	var content string
	if d.item == nil {
		content = DimStyle.Render("No update selected")
	} else {
		content = d.renderContent()
	}

	// Calculate content height (subtract header, border, padding)
	headerHeight := lipgloss.Height(header)
	contentHeight := panelHeight - headerHeight - 4 // border + padding
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Apply scrolling to content
	contentLines := strings.Split(content, "\n")
	if d.scrollOffset >= len(contentLines) {
		d.scrollOffset = len(contentLines) - 1
		if d.scrollOffset < 0 {
			d.scrollOffset = 0
		}
	}

	// Get visible portion
	endIdx := d.scrollOffset + contentHeight
	if endIdx > len(contentLines) {
		endIdx = len(contentLines)
	}
	visibleLines := contentLines[d.scrollOffset:endIdx]
	visibleContent := strings.Join(visibleLines, "\n")

	// Add scroll indicator if needed
	if len(contentLines) > contentHeight {
		scrollInfo := DimStyle.Render(fmt.Sprintf(" [%d/%d]", d.scrollOffset+1, len(contentLines)))
		header = header + scrollInfo
	}

	// Combine header and content
	body := lipgloss.JoinVertical(lipgloss.Left, header, "", visibleContent)

	// Style the panel with border
	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(1, 2).
		Width(panelWidth - 2).
		Height(panelHeight - 2)

	return panelStyle.Render(body)
}

// renderContent renders the main content for the history item
func (d *HistoryDetailPanel) renderContent() string {
	if d.item == nil {
		return DimStyle.Render("No update selected")
	}

	var b strings.Builder

	// Operation kind with color
	b.WriteString(DimStyle.Render("Kind: "))
	b.WriteString(d.renderKind(d.item.Kind))
	b.WriteString("\n")

	// Result status
	b.WriteString(DimStyle.Render("Result: "))
	b.WriteString(d.renderResult(d.item.Result))
	b.WriteString("\n")

	// User who ran the update
	if d.item.User != "" {
		b.WriteString(DimStyle.Render("User: "))
		userStr := d.item.User
		if d.item.UserEmail != "" {
			userStr += " <" + d.item.UserEmail + ">"
		}
		b.WriteString(ValueStyle.Render(userStr))
		b.WriteString("\n")
	}

	// Start time
	b.WriteString(DimStyle.Render("Started: "))
	b.WriteString(ValueStyle.Render(d.formatTime(d.item.StartTime)))
	b.WriteString("\n")

	// End time (if available)
	if d.item.EndTime != "" {
		b.WriteString(DimStyle.Render("Ended: "))
		b.WriteString(ValueStyle.Render(d.formatTime(d.item.EndTime)))
		b.WriteString("\n")

		// Duration
		if duration := d.calculateDuration(d.item.StartTime, d.item.EndTime); duration != "" {
			b.WriteString(DimStyle.Render("Duration: "))
			b.WriteString(ValueStyle.Render(duration))
			b.WriteString("\n")
		}
	}

	// Message
	if d.item.Message != "" {
		b.WriteString("\n")
		b.WriteString(DimStyle.Render("Message:"))
		b.WriteString("\n")
		b.WriteString(ValueStyle.Render(d.item.Message))
		b.WriteString("\n")
	}

	// Resource changes section
	b.WriteString("\n")
	b.WriteString(DimStyle.Render("─── Resource Changes ───"))
	b.WriteString("\n\n")

	if d.item.ResourceChanges == nil || len(d.item.ResourceChanges) == 0 {
		b.WriteString(DimStyle.Render("No resource information available"))
	} else {
		d.renderResourceChanges(&b)
	}

	return b.String()
}

// renderResourceChanges renders detailed resource change information
func (d *HistoryDetailPanel) renderResourceChanges(b *strings.Builder) {
	changes := d.item.ResourceChanges

	// Display counts in a structured way
	create := changes["create"]
	update := changes["update"]
	del := changes["delete"]
	replace := changes["replace"]
	same := changes["same"]

	if create > 0 {
		b.WriteString(OpCreateStyle.Render(fmt.Sprintf("  + %d created", create)))
		b.WriteString("\n")
	}
	if update > 0 {
		b.WriteString(OpUpdateStyle.Render(fmt.Sprintf("  ~ %d updated", update)))
		b.WriteString("\n")
	}
	if replace > 0 {
		b.WriteString(OpReplaceStyle.Render(fmt.Sprintf("  ± %d replaced", replace)))
		b.WriteString("\n")
	}
	if del > 0 {
		b.WriteString(OpDeleteStyle.Render(fmt.Sprintf("  - %d deleted", del)))
		b.WriteString("\n")
	}
	if same > 0 {
		b.WriteString(DimStyle.Render(fmt.Sprintf("  = %d unchanged", same)))
		b.WriteString("\n")
	}

	// Total
	total := create + update + del + replace + same
	b.WriteString("\n")
	b.WriteString(DimStyle.Render(fmt.Sprintf("Total: %d resources", total)))
}

func (d *HistoryDetailPanel) renderKind(kind string) string {
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

func (d *HistoryDetailPanel) renderResult(result string) string {
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

func (d *HistoryDetailPanel) formatTime(timeStr string) string {
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return timeStr
	}
	return t.Format("2006-01-02 15:04:05")
}

func (d *HistoryDetailPanel) calculateDuration(startStr, endStr string) string {
	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		return ""
	}
	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		return ""
	}

	duration := end.Sub(start)

	if duration < time.Second {
		return fmt.Sprintf("%dms", duration.Milliseconds())
	}
	if duration < time.Minute {
		return fmt.Sprintf("%.1fs", duration.Seconds())
	}
	if duration < time.Hour {
		mins := int(duration.Minutes())
		secs := int(duration.Seconds()) % 60
		return fmt.Sprintf("%dm %ds", mins, secs)
	}

	hours := int(duration.Hours())
	mins := int(duration.Minutes()) % 60
	return fmt.Sprintf("%dh %dm", hours, mins)
}
