package ui

import (
	"encoding/json"

	tea "github.com/charmbracelet/bubbletea"
)

// ResourceJSON is the JSON structure for copying a resource
// Matches Pulumi's native output format as closely as possible
type ResourceJSON struct {
	URN     string                 `json:"urn"`
	Type    string                 `json:"type"`
	Inputs  map[string]interface{} `json:"inputs,omitempty"`
	Outputs map[string]interface{} `json:"outputs,omitempty"`
}

// CopyResourceJSON copies the selected resource as JSON to the clipboard
func (r *ResourceList) CopyResourceJSON() tea.Cmd {
	item := r.SelectedItem()
	if item == nil {
		return nil
	}

	// Start flash on current cursor position
	r.flashIdx = r.cursor
	r.flashing = true

	resource := ResourceJSON{
		URN:     item.URN,
		Type:    item.Type,
		Inputs:  item.Inputs,
		Outputs: item.Outputs,
	}

	jsonBytes, err := json.MarshalIndent(resource, "", "  ")
	if err != nil {
		return nil
	}

	return CopyToClipboardWithCountCmd(string(jsonBytes), 1)
}

// CopyAllResourcesJSON copies all visible resources as JSON array to the clipboard
func (r *ResourceList) CopyAllResourcesJSON() tea.Cmd {
	if len(r.visibleIdx) == 0 {
		return nil
	}

	// Flash all visible items
	r.flashAll = true
	r.flashing = true

	// Build JSON array of all visible resources
	resources := make([]ResourceJSON, 0, len(r.visibleIdx))
	for _, idx := range r.visibleIdx {
		item := &r.items[idx]
		resources = append(resources, ResourceJSON{
			URN:     item.URN,
			Type:    item.Type,
			Inputs:  item.Inputs,
			Outputs: item.Outputs,
		})
	}

	jsonBytes, err := json.MarshalIndent(resources, "", "  ")
	if err != nil {
		return nil
	}

	return CopyToClipboardWithCountCmd(string(jsonBytes), len(resources))
}

// VisibleCount returns the number of visible resources
func (r *ResourceList) VisibleCount() int {
	return len(r.visibleIdx)
}
