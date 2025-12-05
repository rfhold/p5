package ui

import (
	"fmt"
	"strings"
)

// HistoryDetailPanel is a floating panel showing history update details
type HistoryDetailPanel struct {
	PanelBase // Embed common panel functionality

	// Current history item being displayed
	item *HistoryItem
}

// NewHistoryDetailPanel creates a new history detail panel component
func NewHistoryDetailPanel() *HistoryDetailPanel {
	return &HistoryDetailPanel{}
}

// SetItem sets the history item to display details for
func (d *HistoryDetailPanel) SetItem(item *HistoryItem) {
	d.item = item
	d.ResetScroll()
}

// View renders the history detail panel
func (d *HistoryDetailPanel) View() string {
	if !d.Visible() || d.Width() == 0 || d.Height() == 0 {
		return ""
	}

	// Build header
	header := "Update Details"
	if d.item != nil {
		header = fmt.Sprintf("Update #%d", d.item.Version)
	}

	// Build content
	var content string
	if d.item == nil {
		content = DimStyle.Render("No update selected")
	} else {
		content = d.renderContent()
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

	return result.Rendered
}

// renderContent renders the main content for the history item
func (d *HistoryDetailPanel) renderContent() string {
	if d.item == nil {
		return DimStyle.Render("No update selected")
	}

	var b strings.Builder

	// Operation kind with color
	b.WriteString(DimStyle.Render("Kind: "))
	b.WriteString(RenderHistoryKind(d.item.Kind))
	b.WriteString("\n")

	// Result status
	b.WriteString(DimStyle.Render("Result: "))
	b.WriteString(RenderHistoryResult(d.item.Result))
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
	b.WriteString(RenderResourceChanges(changes, ResourceChangesExpanded))
	b.WriteString("\n")

	// Total
	total := changes["create"] + changes["update"] + changes["delete"] + changes["replace"] + changes["same"]
	b.WriteString("\n")
	b.WriteString(DimStyle.Render(fmt.Sprintf("Total: %d resources", total)))
}

// renderKind and renderResult are now shared functions in styles.go:
// - RenderHistoryKind(kind string) string
// - RenderHistoryResult(result string) string

func (d *HistoryDetailPanel) formatTime(timeStr string) string {
	return FormatTime(timeStr, "2006-01-02 15:04:05")
}

func (d *HistoryDetailPanel) calculateDuration(startStr, endStr string) string {
	return CalculateDuration(startStr, endStr)
}
