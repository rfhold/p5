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
	maxWidth int
}

// NewDiffRenderer creates a new diff renderer with the specified max width
func NewDiffRenderer(maxWidth int) *DiffRenderer {
	return &DiffRenderer{maxWidth: maxWidth}
}

// RenderCombinedProperties renders inputs and outputs in a single unified diff view
// Properties that exist in both inputs and outputs are shown once (from inputs for diff)
// Output-only properties (computed values like id, arn) are shown separately at the end
func (r *DiffRenderer) RenderCombinedProperties(resource *ResourceItem) string {
	var b strings.Builder

	// Determine old and new states based on operation
	var oldInputs, newInputs, oldOutputs, newOutputs map[string]interface{}

	switch resource.Op {
	case OpCreate:
		// Create: no old state, new inputs and outputs
		oldInputs = nil
		newInputs = resource.Inputs
		oldOutputs = nil
		newOutputs = resource.Outputs

	case OpDelete:
		// Delete: old state being removed
		oldInputs = resource.OldInputs
		if oldInputs == nil {
			oldInputs = resource.Inputs
		}
		newInputs = nil
		oldOutputs = resource.OldOutputs
		if oldOutputs == nil {
			oldOutputs = resource.Outputs
		}
		newOutputs = nil

	case OpUpdate, OpReplace, OpCreateReplace, OpDeleteReplace:
		// Update/Replace: diff between old and new
		oldInputs = resource.OldInputs
		newInputs = resource.Inputs
		oldOutputs = resource.OldOutputs
		newOutputs = resource.Outputs

	case OpSame, OpRefresh:
		// Same/Refresh: inputs don't change, but outputs might (e.g., stack outputs)
		// We diff OldOutputs vs Outputs to show any changes in computed values
		inputs := resource.Inputs
		if inputs == nil {
			inputs = resource.OldInputs
		}
		oldInputs = inputs
		newInputs = inputs
		// For outputs, use old vs new to detect changes (e.g., timestamps in stack outputs)
		oldOutputs = resource.OldOutputs
		newOutputs = resource.Outputs
		// Fallback if both are nil
		if oldOutputs == nil && newOutputs == nil {
			oldOutputs = resource.Outputs
			newOutputs = resource.Outputs
		}

	default:
		// Unknown: show current state
		oldInputs = resource.Inputs
		newInputs = resource.Inputs
		oldOutputs = resource.Outputs
		newOutputs = resource.Outputs
	}

	// If no data at all
	if oldInputs == nil && newInputs == nil && oldOutputs == nil && newOutputs == nil {
		return DimStyle.Render("No properties available")
	}

	// Find which keys are input-only, output-only, or both
	inputKeys := make(map[string]bool)
	outputKeys := make(map[string]bool)

	for k := range oldInputs {
		inputKeys[k] = true
	}
	for k := range newInputs {
		inputKeys[k] = true
	}
	for k := range oldOutputs {
		outputKeys[k] = true
	}
	for k := range newOutputs {
		outputKeys[k] = true
	}

	// Render inputs (these are the user-specified values)
	if len(inputKeys) > 0 {
		b.WriteString(r.renderDiffMap(oldInputs, newInputs, 0))
	}

	// Find output-only keys (computed values not in inputs)
	var outputOnlyKeys []string
	for k := range outputKeys {
		if !inputKeys[k] {
			outputOnlyKeys = append(outputOnlyKeys, k)
		}
	}
	sortStrings(outputOnlyKeys)

	// Render output-only properties with a label
	if len(outputOnlyKeys) > 0 {
		if len(inputKeys) > 0 {
			b.WriteString("\n")
		}
		b.WriteString(DimStyle.Render("── Computed ──"))
		b.WriteString("\n")

		// Build maps with only output-only keys
		oldOutputOnly := make(map[string]interface{})
		newOutputOnly := make(map[string]interface{})
		for _, k := range outputOnlyKeys {
			if v, ok := oldOutputs[k]; ok {
				oldOutputOnly[k] = v
			}
			if v, ok := newOutputs[k]; ok {
				newOutputOnly[k] = v
			}
		}

		b.WriteString(r.renderDiffMap(oldOutputOnly, newOutputOnly, 0))
	}

	result := b.String()
	if result == "" {
		return DimStyle.Render("No properties available")
	}
	return result
}

// renderDiffMap renders a diff between two maps, showing added/removed/changed values
func (r *DiffRenderer) renderDiffMap(oldMap, newMap map[string]interface{}, indent int) string {
	var b strings.Builder
	indentStr := strings.Repeat("  ", indent)

	// Collect all keys from both maps
	allKeys := make(map[string]bool)
	if oldMap != nil {
		for k := range oldMap {
			allKeys[k] = true
		}
	}
	if newMap != nil {
		for k := range newMap {
			allKeys[k] = true
		}
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
		oldVal, hasOld := getMapValue(oldMap, key)
		newVal, hasNew := getMapValue(newMap, key)

		if !hasOld && hasNew {
			// Added
			b.WriteString(r.renderDiffValue(key, nil, newVal, DiffAdded, indentStr, indent))
		} else if hasOld && !hasNew {
			// Removed
			b.WriteString(r.renderDiffValue(key, oldVal, nil, DiffRemoved, indentStr, indent))
		} else if hasOld && hasNew {
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

// renderDiffValue renders a single key-value pair with appropriate diff styling
func (r *DiffRenderer) renderDiffValue(key string, oldVal, newVal interface{}, diffType DiffType, indentStr string, indent int) string {
	var b strings.Builder

	switch diffType {
	case DiffAdded:
		// Check if it's a map - expand it
		if newMap, isMap := newVal.(map[string]interface{}); isMap && len(newMap) > 0 {
			b.WriteString(OpCreateStyle.Render(indentStr + "+ "))
			b.WriteString(OpCreateStyle.Render(key + ":"))
			b.WriteString("\n")
			b.WriteString(r.renderObjectExpanded(newMap, OpCreateStyle, "+", indent+1))
		} else if newArr, isArr := newVal.([]interface{}); isArr && len(newArr) > 0 {
			// Expand arrays too
			b.WriteString(OpCreateStyle.Render(indentStr + "+ "))
			b.WriteString(OpCreateStyle.Render(key + ":"))
			b.WriteString("\n")
			b.WriteString(r.renderArrayExpanded(newArr, OpCreateStyle, "+", indent+1))
		} else {
			// Green + for added
			b.WriteString(OpCreateStyle.Render(indentStr + "+ "))
			b.WriteString(OpCreateStyle.Render(key + ": "))
			b.WriteString(formatDiffValue(newVal, OpCreateStyle, r.maxWidth, indent))
			b.WriteString("\n")
		}

	case DiffRemoved:
		// Check if it's a map - expand it
		if oldMap, isMap := oldVal.(map[string]interface{}); isMap && len(oldMap) > 0 {
			b.WriteString(OpDeleteStyle.Render(indentStr + "- "))
			b.WriteString(OpDeleteStyle.Render(key + ":"))
			b.WriteString("\n")
			b.WriteString(r.renderObjectExpanded(oldMap, OpDeleteStyle, "-", indent+1))
		} else if oldArr, isArr := oldVal.([]interface{}); isArr && len(oldArr) > 0 {
			// Expand arrays too
			b.WriteString(OpDeleteStyle.Render(indentStr + "- "))
			b.WriteString(OpDeleteStyle.Render(key + ":"))
			b.WriteString("\n")
			b.WriteString(r.renderArrayExpanded(oldArr, OpDeleteStyle, "-", indent+1))
		} else {
			// Red - for removed
			b.WriteString(OpDeleteStyle.Render(indentStr + "- "))
			b.WriteString(OpDeleteStyle.Render(key + ": "))
			b.WriteString(formatDiffValue(oldVal, OpDeleteStyle, r.maxWidth, indent))
			b.WriteString("\n")
		}

	case DiffModified:
		// Check if both are maps - if so, recurse
		oldMap, oldIsMap := oldVal.(map[string]interface{})
		newMap, newIsMap := newVal.(map[string]interface{})

		if oldIsMap && newIsMap {
			// Recurse into nested maps
			b.WriteString(OpUpdateStyle.Render(indentStr + "~ "))
			b.WriteString(OpUpdateStyle.Render(key + ":"))
			b.WriteString("\n")
			b.WriteString(r.renderDiffMap(oldMap, newMap, indent+1))
		} else {
			// Check if both are arrays - if so, show element-level diff
			oldArr, oldIsArr := oldVal.([]interface{})
			newArr, newIsArr := newVal.([]interface{})

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
		// Check if it's a map - expand it
		if newMap, isMap := newVal.(map[string]interface{}); isMap && len(newMap) > 0 {
			b.WriteString(DimStyle.Render(indentStr + "  "))
			b.WriteString(DimStyle.Render(key + ":"))
			b.WriteString("\n")
			b.WriteString(r.renderObjectExpanded(newMap, DimStyle, " ", indent+1))
		} else if newArr, isArr := newVal.([]interface{}); isArr && len(newArr) > 0 {
			// Expand arrays too for unchanged
			b.WriteString(DimStyle.Render(indentStr + "  "))
			b.WriteString(DimStyle.Render(key + ":"))
			b.WriteString("\n")
			b.WriteString(r.renderArrayExpanded(newArr, DimStyle, " ", indent+1))
		} else {
			// Dimmed for unchanged
			b.WriteString(DimStyle.Render(indentStr + "  "))
			b.WriteString(DimStyle.Render(key + ": "))
			b.WriteString(formatDiffValue(newVal, DimStyle, r.maxWidth, indent))
			b.WriteString("\n")
		}
	}

	return b.String()
}

// renderArrayDiff renders a diff between two arrays showing element-level changes
func (r *DiffRenderer) renderArrayDiff(oldArr, newArr []interface{}, indent int) string {
	var b strings.Builder
	indentStr := strings.Repeat("  ", indent)

	maxLen := len(oldArr)
	if len(newArr) > maxLen {
		maxLen = len(newArr)
	}

	for i := 0; i < maxLen; i++ {
		hasOld := i < len(oldArr)
		hasNew := i < len(newArr)

		if hasOld && hasNew {
			oldVal := oldArr[i]
			newVal := newArr[i]

			if valuesEqual(oldVal, newVal) {
				// Unchanged element
				b.WriteString(DimStyle.Render(fmt.Sprintf("%s  [%d]: ", indentStr, i)))
				b.WriteString(formatDiffValue(oldVal, DimStyle, r.maxWidth, indent+1))
				b.WriteString("\n")
			} else {
				// Check if both are maps - recurse
				oldMap, oldIsMap := oldVal.(map[string]interface{})
				newMap, newIsMap := newVal.(map[string]interface{})

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
		} else if hasOld {
			// Removed element (old array was longer)
			b.WriteString(OpDeleteStyle.Render(fmt.Sprintf("%s- [%d]: ", indentStr, i)))
			b.WriteString(formatDiffValue(oldArr[i], OpDeleteStyle, r.maxWidth, indent+1))
			b.WriteString("\n")
		} else if hasNew {
			// Added element (new array is longer)
			b.WriteString(OpCreateStyle.Render(fmt.Sprintf("%s+ [%d]: ", indentStr, i)))
			b.WriteString(formatDiffValue(newArr[i], OpCreateStyle, r.maxWidth, indent+1))
			b.WriteString("\n")
		}
	}

	return b.String()
}

// renderObjectExpanded renders all keys of an object with consistent styling
func (r *DiffRenderer) renderObjectExpanded(obj map[string]interface{}, style lipgloss.Style, prefix string, indent int) string {
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
		val := obj[key]

		// Check if value is a nested map
		if nestedMap, isMap := val.(map[string]interface{}); isMap && len(nestedMap) > 0 {
			b.WriteString(style.Render(indentStr + prefix + " "))
			b.WriteString(style.Render(key + ":"))
			b.WriteString("\n")
			b.WriteString(r.renderObjectExpanded(nestedMap, style, prefix, indent+1))
		} else if nestedArr, isArr := val.([]interface{}); isArr && len(nestedArr) > 0 {
			b.WriteString(style.Render(indentStr + prefix + " "))
			b.WriteString(style.Render(key + ":"))
			b.WriteString("\n")
			b.WriteString(r.renderArrayExpanded(nestedArr, style, prefix, indent+1))
		} else {
			b.WriteString(style.Render(indentStr + prefix + " "))
			b.WriteString(style.Render(key + ": "))
			b.WriteString(formatDiffValue(val, style, r.maxWidth, indent))
			b.WriteString("\n")
		}
	}

	return b.String()
}

// renderArrayExpanded renders all elements of an array with consistent styling
func (r *DiffRenderer) renderArrayExpanded(arr []interface{}, style lipgloss.Style, prefix string, indent int) string {
	var b strings.Builder
	indentStr := strings.Repeat("  ", indent)

	for i, val := range arr {
		// Check if value is a nested map
		if nestedMap, isMap := val.(map[string]interface{}); isMap && len(nestedMap) > 0 {
			b.WriteString(style.Render(fmt.Sprintf("%s%s [%d]:", indentStr, prefix, i)))
			b.WriteString("\n")
			b.WriteString(r.renderObjectExpanded(nestedMap, style, prefix, indent+1))
		} else if nestedArr, isArr := val.([]interface{}); isArr && len(nestedArr) > 0 {
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
