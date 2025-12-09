package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// DiffType represents the type of change for a value
type DiffType int

const (
	DiffUnchanged DiffType = iota
	DiffAdded
	DiffRemoved
	DiffModified
)

// DiffRenderer handles rendering of property diffs
type DiffRenderer struct {
	maxWidth  int
	keyFilter func(key string) bool // Optional filter function for property keys
}

// NewDiffRenderer creates a new diff renderer with the specified max width
func NewDiffRenderer(maxWidth int) *DiffRenderer {
	return &DiffRenderer{maxWidth: maxWidth}
}

// SetKeyFilter sets a filter function for property keys
// Only keys where filter(key) returns true will be displayed
func (r *DiffRenderer) SetKeyFilter(filter func(key string) bool) {
	r.keyFilter = filter
}

// ClearKeyFilter removes the key filter
func (r *DiffRenderer) ClearKeyFilter() {
	r.keyFilter = nil
}

// shouldShowKey returns true if the key should be displayed based on filter
func (r *DiffRenderer) shouldShowKey(key string) bool {
	if r.keyFilter == nil {
		return true
	}
	return r.keyFilter(key)
}

type diffState struct {
	oldInputs, newInputs   map[string]any
	oldOutputs, newOutputs map[string]any
}

func getDiffStateForOperation(resource *ResourceItem) diffState {
	switch resource.Op {
	case OpCreate:
		return diffState{
			oldInputs: nil, newInputs: resource.Inputs,
			oldOutputs: nil, newOutputs: resource.Outputs,
		}
	case OpDelete:
		return getDiffStateForDelete(resource)
	case OpUpdate, OpReplace, OpCreateReplace, OpDeleteReplace:
		return diffState{
			oldInputs: resource.OldInputs, newInputs: resource.Inputs,
			oldOutputs: resource.OldOutputs, newOutputs: resource.Outputs,
		}
	case OpSame, OpRefresh:
		return getDiffStateForSameRefresh(resource)
	default:
		return diffState{
			oldInputs: resource.Inputs, newInputs: resource.Inputs,
			oldOutputs: resource.Outputs, newOutputs: resource.Outputs,
		}
	}
}

func getDiffStateForDelete(resource *ResourceItem) diffState {
	oldInputs := resource.OldInputs
	if oldInputs == nil {
		oldInputs = resource.Inputs
	}
	oldOutputs := resource.OldOutputs
	if oldOutputs == nil {
		oldOutputs = resource.Outputs
	}
	return diffState{oldInputs: oldInputs, newInputs: nil, oldOutputs: oldOutputs, newOutputs: nil}
}

func getDiffStateForSameRefresh(resource *ResourceItem) diffState {
	inputs := resource.Inputs
	if inputs == nil {
		inputs = resource.OldInputs
	}
	oldOutputs := resource.OldOutputs
	newOutputs := resource.Outputs
	if oldOutputs == nil && newOutputs == nil {
		oldOutputs = resource.Outputs
		newOutputs = resource.Outputs
	}
	return diffState{oldInputs: inputs, newInputs: inputs, oldOutputs: oldOutputs, newOutputs: newOutputs}
}

func collectKeys(maps ...map[string]any) map[string]bool {
	keys := make(map[string]bool)
	for _, m := range maps {
		for k := range m {
			keys[k] = true
		}
	}
	return keys
}

// RenderCombinedProperties renders inputs and outputs in a single unified diff view
// Properties that exist in both inputs and outputs are shown once (from inputs for diff)
// Output-only properties (computed values like id, arn) are shown separately at the end
func (r *DiffRenderer) RenderCombinedProperties(resource *ResourceItem) string {
	state := getDiffStateForOperation(resource)

	if state.oldInputs == nil && state.newInputs == nil && state.oldOutputs == nil && state.newOutputs == nil {
		return DimStyle.Render("No properties available")
	}

	inputKeys := collectKeys(state.oldInputs, state.newInputs)
	outputKeys := collectKeys(state.oldOutputs, state.newOutputs)

	var b strings.Builder

	if len(inputKeys) > 0 {
		b.WriteString(r.renderDiffMap(state.oldInputs, state.newInputs, 0))
	}

	b.WriteString(r.renderOutputOnlyProperties(state, inputKeys, outputKeys))

	result := b.String()
	if result == "" {
		return DimStyle.Render("No properties available")
	}
	return result
}

func (r *DiffRenderer) renderOutputOnlyProperties(state diffState, inputKeys, outputKeys map[string]bool) string {
	var outputOnlyKeys []string
	for k := range outputKeys {
		if !inputKeys[k] {
			outputOnlyKeys = append(outputOnlyKeys, k)
		}
	}
	sortStrings(outputOnlyKeys)

	if len(outputOnlyKeys) == 0 {
		return ""
	}

	var b strings.Builder
	if len(inputKeys) > 0 {
		b.WriteString("\n")
	}
	b.WriteString(DimStyle.Render("── Computed ──"))
	b.WriteString("\n")

	oldOutputOnly := make(map[string]any)
	newOutputOnly := make(map[string]any)
	for _, k := range outputOnlyKeys {
		if v, ok := state.oldOutputs[k]; ok {
			oldOutputOnly[k] = v
		}
		if v, ok := state.newOutputs[k]; ok {
			newOutputOnly[k] = v
		}
	}

	b.WriteString(r.renderDiffMap(oldOutputOnly, newOutputOnly, 0))
	return b.String()
}

// renderDiffMap renders a diff between two maps, showing added/removed/changed values
func (r *DiffRenderer) renderDiffMap(oldMap, newMap map[string]any, indent int) string {
	var b strings.Builder
	indentStr := strings.Repeat("  ", indent)

	// Collect all keys from both maps
	allKeys := make(map[string]bool)
	for k := range oldMap {
		allKeys[k] = true
	}
	for k := range newMap {
		allKeys[k] = true
	}

	// Sort keys for consistent output, excluding internal __ prefixed keys
	keys := make([]string, 0, len(allKeys))
	for k := range allKeys {
		if !strings.HasPrefix(k, "__") {
			keys = append(keys, k)
		}
	}
	sortStrings(keys)

	for _, key := range keys {
		// Apply filter at root level (indent == 0)
		if indent == 0 && !r.shouldShowKey(key) {
			continue
		}

		oldVal, hasOld := getMapValue(oldMap, key)
		newVal, hasNew := getMapValue(newMap, key)

		switch {
		case !hasOld && hasNew:
			// Added
			b.WriteString(r.renderDiffValue(key, nil, newVal, DiffAdded, indentStr, indent))
		case hasOld && !hasNew:
			// Removed
			b.WriteString(r.renderDiffValue(key, oldVal, nil, DiffRemoved, indentStr, indent))
		case hasOld && hasNew:
			// Both exist - check if changed
			if valuesEqual(oldVal, newVal) {
				// Unchanged - show dimmed
				b.WriteString(r.renderDiffValue(key, oldVal, newVal, DiffUnchanged, indentStr, indent))
			} else {
				// Modified
				b.WriteString(r.renderDiffValue(key, oldVal, newVal, DiffModified, indentStr, indent))
			}
		}
	}

	return b.String()
}

// renderStyledValue renders a value with consistent styling for add/remove/unchanged operations
func (r *DiffRenderer) renderStyledValue(b *strings.Builder, key string, val any, style lipgloss.Style, prefix, indentStr string, indent int) {
	if valMap, isMap := val.(map[string]any); isMap && len(valMap) > 0 {
		b.WriteString(style.Render(indentStr + prefix + " "))
		b.WriteString(style.Render(key + ":"))
		b.WriteString("\n")
		b.WriteString(r.renderObjectExpanded(valMap, style, prefix, indent+1))
	} else if valArr, isArr := val.([]any); isArr && len(valArr) > 0 {
		b.WriteString(style.Render(indentStr + prefix + " "))
		b.WriteString(style.Render(key + ":"))
		b.WriteString("\n")
		b.WriteString(r.renderArrayExpanded(valArr, style, prefix, indent+1))
	} else {
		b.WriteString(style.Render(indentStr + prefix + " "))
		b.WriteString(style.Render(key + ": "))
		b.WriteString(formatDiffValue(val, style, r.maxWidth, indent))
		b.WriteString("\n")
	}
}

// renderDiffValue renders a single key-value pair with appropriate diff styling
func (r *DiffRenderer) renderDiffValue(key string, oldVal, newVal any, diffType DiffType, indentStr string, indent int) string {
	var b strings.Builder

	switch diffType {
	case DiffAdded:
		r.renderStyledValue(&b, key, newVal, OpCreateStyle, "+", indentStr, indent)

	case DiffRemoved:
		r.renderStyledValue(&b, key, oldVal, OpDeleteStyle, "-", indentStr, indent)

	case DiffModified:
		// Check if both are maps - if so, recurse
		oldMap, oldIsMap := oldVal.(map[string]any)
		newMap, newIsMap := newVal.(map[string]any)

		if oldIsMap && newIsMap {
			// Recurse into nested maps
			b.WriteString(OpUpdateStyle.Render(indentStr + "~ "))
			b.WriteString(OpUpdateStyle.Render(key + ":"))
			b.WriteString("\n")
			b.WriteString(r.renderDiffMap(oldMap, newMap, indent+1))
		} else {
			// Check if both are arrays - if so, show element-level diff
			oldArr, oldIsArr := oldVal.([]any)
			newArr, newIsArr := newVal.([]any)

			if oldIsArr && newIsArr {
				// Show array diff with element-level changes
				b.WriteString(OpUpdateStyle.Render(indentStr + "~ "))
				b.WriteString(OpUpdateStyle.Render(key + ":"))
				b.WriteString("\n")
				b.WriteString(r.renderArrayDiff(oldArr, newArr, indent+1))
			} else {
				// Show inline diff with ~ prefix and > separator
				b.WriteString(OpUpdateStyle.Render(indentStr + "~ "))
				b.WriteString(OpUpdateStyle.Render(key + ": "))
				b.WriteString(formatDiffValue(oldVal, OpDeleteStyle, r.maxWidth, indent))
				b.WriteString(OpUpdateStyle.Render(" > "))
				b.WriteString(formatDiffValue(newVal, OpCreateStyle, r.maxWidth, indent))
				b.WriteString("\n")
			}
		}

	case DiffUnchanged:
		r.renderStyledValue(&b, key, newVal, DimStyle, " ", indentStr, indent)
	}

	return b.String()
}

// renderArrayDiff renders a diff between two arrays showing element-level changes
func (r *DiffRenderer) renderArrayDiff(oldArr, newArr []any, indent int) string {
	var b strings.Builder
	indentStr := strings.Repeat("  ", indent)

	maxLen := max(len(newArr), len(oldArr))

	for i := range maxLen {
		hasOld := i < len(oldArr)
		hasNew := i < len(newArr)

		switch {
		case hasOld && hasNew:
			oldVal := oldArr[i]
			newVal := newArr[i]

			if valuesEqual(oldVal, newVal) {
				// Unchanged element
				b.WriteString(DimStyle.Render(fmt.Sprintf("%s  [%d]: ", indentStr, i)))
				b.WriteString(formatDiffValue(oldVal, DimStyle, r.maxWidth, indent+1))
				b.WriteString("\n")
			} else {
				// Check if both are maps - recurse
				oldMap, oldIsMap := oldVal.(map[string]any)
				newMap, newIsMap := newVal.(map[string]any)

				if oldIsMap && newIsMap {
					b.WriteString(OpUpdateStyle.Render(fmt.Sprintf("%s~ [%d]:", indentStr, i)))
					b.WriteString("\n")
					b.WriteString(r.renderDiffMap(oldMap, newMap, indent+2))
				} else {
					// Modified element - show inline with ~ and > separator
					b.WriteString(OpUpdateStyle.Render(fmt.Sprintf("%s~ [%d]: ", indentStr, i)))
					b.WriteString(formatDiffValue(oldVal, OpDeleteStyle, r.maxWidth, indent+1))
					b.WriteString(OpUpdateStyle.Render(" > "))
					b.WriteString(formatDiffValue(newVal, OpCreateStyle, r.maxWidth, indent+1))
					b.WriteString("\n")
				}
			}
		case hasOld:
			// Removed element (old array was longer)
			b.WriteString(OpDeleteStyle.Render(fmt.Sprintf("%s- [%d]: ", indentStr, i)))
			b.WriteString(formatDiffValue(oldArr[i], OpDeleteStyle, r.maxWidth, indent+1))
			b.WriteString("\n")
		case hasNew:
			// Added element (new array is longer)
			b.WriteString(OpCreateStyle.Render(fmt.Sprintf("%s+ [%d]: ", indentStr, i)))
			b.WriteString(formatDiffValue(newArr[i], OpCreateStyle, r.maxWidth, indent+1))
			b.WriteString("\n")
		}
	}

	return b.String()
}

// renderObjectExpanded renders all keys of an object with consistent styling
func (r *DiffRenderer) renderObjectExpanded(obj map[string]any, style lipgloss.Style, prefix string, indent int) string {
	var b strings.Builder
	indentStr := strings.Repeat("  ", indent)

	// Collect and sort keys, excluding internal __ prefixed keys
	keys := make([]string, 0, len(obj))
	for k := range obj {
		if !strings.HasPrefix(k, "__") {
			keys = append(keys, k)
		}
	}
	sortStrings(keys)

	for _, key := range keys {
		r.renderStyledValue(&b, key, obj[key], style, prefix, indentStr, indent)
	}

	return b.String()
}

// renderArrayExpanded renders all elements of an array with consistent styling
func (r *DiffRenderer) renderArrayExpanded(arr []any, style lipgloss.Style, prefix string, indent int) string {
	var b strings.Builder
	indentStr := strings.Repeat("  ", indent)

	for i, val := range arr {
		// Check if value is a nested map
		if nestedMap, isMap := val.(map[string]any); isMap && len(nestedMap) > 0 {
			b.WriteString(style.Render(fmt.Sprintf("%s%s [%d]:", indentStr, prefix, i)))
			b.WriteString("\n")
			b.WriteString(r.renderObjectExpanded(nestedMap, style, prefix, indent+1))
		} else if nestedArr, isArr := val.([]any); isArr && len(nestedArr) > 0 {
			b.WriteString(style.Render(fmt.Sprintf("%s%s [%d]:", indentStr, prefix, i)))
			b.WriteString("\n")
			b.WriteString(r.renderArrayExpanded(nestedArr, style, prefix, indent+1))
		} else {
			b.WriteString(style.Render(fmt.Sprintf("%s%s [%d]: ", indentStr, prefix, i)))
			b.WriteString(formatDiffValue(val, style, r.maxWidth, indent))
			b.WriteString("\n")
		}
	}

	return b.String()
}
