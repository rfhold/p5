package ui

import (
	"strings"
)

// DetailPanel is a floating panel showing resource details
type DetailPanel struct {
	PanelBase // Embed common panel functionality

	// Current resource being displayed
	resource *ResourceItem
}

// NewDetailPanel creates a new detail panel component
func NewDetailPanel() *DetailPanel {
	return &DetailPanel{}
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

	return result.Rendered
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
