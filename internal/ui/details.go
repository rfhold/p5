package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// DetailPanel is a floating panel showing resource details
type DetailPanel struct {
	PanelBase // Embed common panel functionality

	// Current resource being displayed
	resource *ResourceItem

	// Filter state for property keys
	filter FilterState
}

// NewDetailPanel creates a new detail panel component
func NewDetailPanel() *DetailPanel {
	return &DetailPanel{
		filter: NewFilterState(),
	}
}

// SetResource sets the resource to display details for
func (d *DetailPanel) SetResource(resource *ResourceItem) {
	d.resource = resource
	d.ResetScroll()
	// Don't reset filter when changing resources - user might want to keep filtering
}

// FilterActive returns whether the filter is currently active
func (d *DetailPanel) FilterActive() bool {
	return d.filter.Active()
}

// Update handles key events for the detail panel
func (d *DetailPanel) Update(msg tea.KeyMsg) tea.Cmd {
	if !d.Visible() {
		return nil
	}

	// Handle filter activation with "/"
	if key.Matches(msg, Keys.Filter) && !d.filter.Active() {
		d.filter.Activate()
		return nil
	}

	// Forward to filter if active
	if d.filter.Active() {
		cmd, handled := d.filter.Update(msg)
		if handled {
			return cmd
		}
	}

	return nil
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

	// Add filter indicator to header
	if d.filter.Active() || d.filter.Applied() {
		header += DimStyle.Render(" [filtered]")
	}

	// Build unified content
	var content string
	if d.resource == nil {
		content = DimStyle.Render("No resource selected")
	} else {
		content = d.renderUnified()
	}

	// Add filter bar at end of content if active or applied
	if d.filter.Active() || d.filter.Applied() {
		content += "\n\n" + d.filter.View()
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

	// Apply key filter if filter is applied
	if d.filter.Applied() {
		renderer.SetKeyFilter(func(key string) bool {
			return d.filter.Matches(key)
		})
	}

	content := renderer.RenderCombinedProperties(d.resource)
	if d.filter.Applied() && strings.TrimSpace(content) == "" {
		b.WriteString(DimStyle.Render("No matching properties"))
	} else {
		b.WriteString(content)
	}

	return b.String()
}
