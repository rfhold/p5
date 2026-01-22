package ui

// ResourceFlags tracks selection flags for a resource
type ResourceFlags struct {
	Target  bool // --target flag for update
	Replace bool // --replace flag for update
	Exclude bool // exclude from update (mutually exclusive with Target/Replace)
}

// toggleFlag toggles the specified flag for selected resources
func (r *ResourceList) toggleFlag(flagType string) {
	indices := r.getSelectedIndices()
	itemCount := r.effectiveItemCount()

	for _, idx := range indices {
		if idx < 0 || idx >= itemCount {
			continue
		}
		visIdx := r.effectiveIndex(idx)
		if visIdx < 0 || visIdx >= len(r.visibleIdx) {
			continue
		}
		item := r.items[r.visibleIdx[visIdx]]
		urn := item.URN

		flags := r.flags[urn]

		switch flagType {
		case "target":
			// Clear exclude if setting target
			if !flags.Target {
				flags.Exclude = false
			}
			flags.Target = !flags.Target

		case "replace":
			// Replace is allowed on any resource (from stack view or update ops)
			// Skip the stack resource itself since it can't be replaced
			if item.Type == "pulumi:pulumi:Stack" {
				continue
			}
			// Clear exclude if setting replace
			if !flags.Replace {
				flags.Exclude = false
			}
			flags.Replace = !flags.Replace

		case "exclude":
			// Clear target/replace if setting exclude
			if !flags.Exclude {
				flags.Target = false
				flags.Replace = false
			}
			flags.Exclude = !flags.Exclude
		}

		r.flags[urn] = flags
	}

	// Exit visual mode after toggling
	r.visualMode = false
}

// clearFlags clears all flags for selected resources
func (r *ResourceList) clearFlags() {
	indices := r.getSelectedIndices()
	itemCount := r.effectiveItemCount()

	for _, idx := range indices {
		if idx < 0 || idx >= itemCount {
			continue
		}
		visIdx := r.effectiveIndex(idx)
		if visIdx < 0 || visIdx >= len(r.visibleIdx) {
			continue
		}
		urn := r.items[r.visibleIdx[visIdx]].URN
		delete(r.flags, urn)
	}

	// Exit visual mode after clearing
	r.visualMode = false
}

// GetTargetURNs returns URNs flagged for --target
func (r *ResourceList) GetTargetURNs() []string {
	var urns []string
	for urn, flags := range r.flags {
		if flags.Target {
			urns = append(urns, urn)
		}
	}
	return urns
}

// GetReplaceURNs returns URNs flagged for --replace
func (r *ResourceList) GetReplaceURNs() []string {
	var urns []string
	for urn, flags := range r.flags {
		if flags.Replace {
			urns = append(urns, urn)
		}
	}
	return urns
}

// GetExcludeURNs returns URNs flagged for exclusion
func (r *ResourceList) GetExcludeURNs() []string {
	var urns []string
	for urn, flags := range r.flags {
		if flags.Exclude {
			urns = append(urns, urn)
		}
	}
	return urns
}

// HasFlags returns true if any resources have flags set
func (r *ResourceList) HasFlags() bool {
	return len(r.flags) > 0
}

// ClearAllFlags clears all flags
func (r *ResourceList) ClearAllFlags() {
	for k := range r.flags {
		delete(r.flags, k)
	}
}

// SelectedResource represents a selected resource with its URN and name
type SelectedResource struct {
	URN  string
	Name string
	Type string
}

// GetSelectedResourcesForStateDelete returns selected resources that can be deleted from state.
// It excludes the root stack resource (pulumi:pulumi:Stack) as it cannot be deleted.
// Returns the union of discrete selections and visual range, or just the cursor item if neither is active.
func (r *ResourceList) GetSelectedResourcesForStateDelete() []SelectedResource {
	indices := r.getSelectedIndices()
	itemCount := r.effectiveItemCount()
	var resources []SelectedResource

	for _, idx := range indices {
		if idx < 0 || idx >= itemCount {
			continue
		}
		visIdx := r.effectiveIndex(idx)
		if visIdx < 0 || visIdx >= len(r.visibleIdx) {
			continue
		}
		item := r.items[r.visibleIdx[visIdx]]
		// Skip root stack resource - it cannot be deleted from state
		if item.Type == "pulumi:pulumi:Stack" {
			continue
		}
		resources = append(resources, SelectedResource{
			URN:  item.URN,
			Name: item.Name,
			Type: item.Type,
		})
	}

	return resources
}
