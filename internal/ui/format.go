package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// isComputedPlaceholder checks if a string looks like a Pulumi computed value placeholder (UUID)
func isComputedPlaceholder(s string) bool {
	// Pulumi uses UUID v4 format for computed placeholders: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	if len(s) != 36 {
		return false
	}
	// Check format: 8-4-4-4-12 with hyphens at positions 8, 13, 18, 23
	if s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return false
	}
	// Check all other chars are hex digits
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			continue
		}
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// formatDiffValue formats a value for display in the diff view
func formatDiffValue(value interface{}, style lipgloss.Style, maxWidth int, indent int) string {
	if value == nil {
		return style.Render("null")
	}

	switch v := value.(type) {
	case map[string]interface{}:
		if len(v) == 0 {
			return style.Render("{}")
		}
		// For inline display, show abbreviated - expanded view handled separately
		return style.Render(fmt.Sprintf("{...%d keys}", len(v)))

	case []interface{}:
		if len(v) == 0 {
			return style.Render("[]")
		}
		// Format array items inline
		var items []string
		for _, item := range v {
			items = append(items, formatArrayItem(item))
		}
		result := "[" + strings.Join(items, ", ") + "]"
		// Truncate if too long
		maxLen := maxWidth - (indent * 2)
		if maxLen < 30 {
			maxLen = 30
		}
		if len(result) > maxLen {
			return style.Render(result[:maxLen-3] + "...")
		}
		return style.Render(result)

	case string:
		// Check for Pulumi computed value placeholder (UUID)
		if isComputedPlaceholder(v) {
			// Show computed values with update style to indicate they will change
			return OpUpdateStyle.Render("~[computed]")
		}
		// Truncate long strings
		maxLen := maxWidth - (indent * 2) - 20
		if maxLen < 20 {
			maxLen = 20
		}
		if len(v) > maxLen {
			return style.Render(fmt.Sprintf("%q...", v[:maxLen-3]))
		}
		return style.Render(fmt.Sprintf("%q", v))

	case bool:
		if v {
			return style.Render("true")
		}
		return style.Render("false")

	case float64:
		// Check if it's actually an integer
		if v == float64(int64(v)) {
			return style.Render(fmt.Sprintf("%d", int64(v)))
		}
		return style.Render(fmt.Sprintf("%v", v))

	default:
		return style.Render(fmt.Sprintf("%v", v))
	}
}

// formatArrayItem formats a single array item for inline display
func formatArrayItem(item interface{}) string {
	switch v := item.(type) {
	case map[string]interface{}:
		if len(v) == 0 {
			return "{}"
		}
		return fmt.Sprintf("{...%d keys}", len(v))
	case []interface{}:
		if len(v) == 0 {
			return "[]"
		}
		return fmt.Sprintf("[...%d items]", len(v))
	case string:
		// Truncate long strings in arrays
		if len(v) > 30 {
			return fmt.Sprintf("%q...", v[:27])
		}
		return fmt.Sprintf("%q", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	case float64:
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%v", v)
	case nil:
		return "null"
	default:
		return fmt.Sprintf("%v", v)
	}
}

// wrapText wraps text to fit within maxWidth
func wrapText(text string, maxWidth int) string {
	if maxWidth <= 0 {
		maxWidth = 40
	}
	if len(text) <= maxWidth {
		return text
	}

	var lines []string
	for len(text) > maxWidth {
		lines = append(lines, text[:maxWidth])
		text = text[maxWidth:]
	}
	if len(text) > 0 {
		lines = append(lines, text)
	}
	return strings.Join(lines, "\n")
}
