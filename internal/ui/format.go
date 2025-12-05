package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Pulumi uses specific sentinel UUIDs to represent unknown/computed values.
// These are defined in github.com/pulumi/pulumi/sdk/v3/go/common/resource/plugin/rpc.go
var pulumiUnknownSentinels = map[string]bool{
	"1c4a061d-8072-4f0a-a4cb-0ff528b18fe7": true, // UnknownBoolValue
	"3eeb2bf0-c639-47a8-9e75-3b44932eb421": true, // UnknownNumberValue
	"04da6b54-80e4-46f7-96ec-b56ff0331ba9": true, // UnknownStringValue
	"6a19a0b0-7e62-4c92-b797-7f8e31da9cc2": true, // UnknownArrayValue
	"030794c1-ac77-496b-92df-f27374a8bd58": true, // UnknownAssetValue
	"e48ece36-62e2-4504-bad9-02848725956a": true, // UnknownArchiveValue
	"dd056dcd-154b-4c76-9bd3-c8f88648b5ff": true, // UnknownObjectValue
}

// isComputedPlaceholder checks if a string is a Pulumi sentinel value for unknown/computed values.
// Unlike the previous implementation that treated any UUID as a computed placeholder,
// this only matches the specific sentinel UUIDs that Pulumi uses internally.
func isComputedPlaceholder(s string) bool {
	return pulumiUnknownSentinels[s]
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
		if maxLen < DefaultMaxStringLength {
			maxLen = DefaultMaxStringLength
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
		maxLen := maxWidth - (indent * 2) - MinFormattedStringLength
		if maxLen < MinFormattedStringLength {
			maxLen = MinFormattedStringLength
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
		if len(v) > ArrayItemTruncateLength {
			return fmt.Sprintf("%q...", v[:ArrayItemTruncateDisplay])
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
