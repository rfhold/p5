package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
)

// getMapValue safely gets a value from a map
func getMapValue(m map[string]interface{}, key string) (interface{}, bool) {
	if m == nil {
		return nil, false
	}
	val, ok := m[key]
	return val, ok
}

// valuesEqual compares two values for equality
func valuesEqual(a, b interface{}) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Compare maps recursively
	aMap, aIsMap := a.(map[string]interface{})
	bMap, bIsMap := b.(map[string]interface{})
	if aIsMap && bIsMap {
		if len(aMap) != len(bMap) {
			return false
		}
		for k, av := range aMap {
			bv, ok := bMap[k]
			if !ok || !valuesEqual(av, bv) {
				return false
			}
		}
		return true
	}

	// Compare slices
	aSlice, aIsSlice := a.([]interface{})
	bSlice, bIsSlice := b.([]interface{})
	if aIsSlice && bIsSlice {
		if len(aSlice) != len(bSlice) {
			return false
		}
		for i := range aSlice {
			if !valuesEqual(aSlice[i], bSlice[i]) {
				return false
			}
		}
		return true
	}

	// Use fmt for simple comparison
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

// sortStrings sorts a slice of strings in place
func sortStrings(s []string) {
	for i := 0; i < len(s)-1; i++ {
		for j := i + 1; j < len(s); j++ {
			if s[i] > s[j] {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
}

// truncateMiddle truncates a string in the middle if it exceeds maxLen,
// keeping the beginning and end visible with "..." in the middle.
// Returns the original string if it fits within maxLen.
func truncateMiddle(s string, maxLen int) string {
	if len(s) <= maxLen || maxLen < 5 {
		return s
	}

	// Reserve 3 chars for "..."
	// Split remaining space between start and end, favoring the end slightly
	remaining := maxLen - 3
	endLen := (remaining + 1) / 2
	startLen := remaining - endLen

	return s[:startLen] + "***" + s[len(s)-endLen:]
}

// RenderCenteredLoading renders a loading state with spinner centered in the given dimensions
func RenderCenteredLoading(spin spinner.Model, msg string, width, height int) string {
	if msg == "" {
		msg = "Loading..."
	}
	content := fmt.Sprintf("%s %s", spin.View(), msg)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)
}

// RenderCenteredMessage renders a dim message centered in the given dimensions
func RenderCenteredMessage(msg string, width, height int) string {
	content := DimStyle.Render(msg)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)
}

// RenderPaddedError renders an error with padding
func RenderPaddedError(err error) string {
	paddedStyle := lipgloss.NewStyle().Padding(1, 2)
	errMsg := ErrorStyle.Render(fmt.Sprintf("Error: %v", err))
	return paddedStyle.Render(errMsg)
}

// FormatTime parses an RFC3339 time string and formats it with the given format.
// Returns the original string if parsing fails.
func FormatTime(timeStr string, format string) string {
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return timeStr
	}
	return t.Format(format)
}

// FormatTimeStyled parses an RFC3339 time string and formats it with styling.
// If parsing fails and the string is longer than maxLen, it truncates the string.
// Pass maxLen <= 0 to skip truncation on parse failure.
func FormatTimeStyled(timeStr string, format string, maxLen int, style lipgloss.Style) string {
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		if maxLen > 0 && len(timeStr) > maxLen {
			return style.Render(timeStr[:maxLen])
		}
		return style.Render(timeStr)
	}
	return style.Render(t.Format(format))
}

// CalculateDuration calculates the duration between two RFC3339 time strings
// and returns a human-readable string. Returns empty string on parse error.
func CalculateDuration(startStr, endStr string) string {
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

// ScrollIndicatorConfig configures the scroll indicator rendering
type ScrollIndicatorConfig struct {
	Padding       string // Padding before the arrow (default: "  ")
	IncludeMore   bool   // Whether to include "more" text after the arrow
	TrailingSpace bool   // Whether to reserve space when indicator is not shown (prevents layout jumps)
}

// DefaultScrollConfig returns the default scroll indicator configuration
func DefaultScrollConfig() ScrollIndicatorConfig {
	return ScrollIndicatorConfig{
		Padding:       "  ",
		IncludeMore:   false,
		TrailingSpace: true,
	}
}

// RenderScrollUpIndicator renders an up scroll indicator with proper spacing.
// Returns styled arrow if canScroll is true, otherwise returns empty spacing.
func RenderScrollUpIndicator(canScroll bool) string {
	return RenderScrollUpIndicatorWithConfig(canScroll, DefaultScrollConfig())
}

// RenderScrollDownIndicator renders a down scroll indicator with proper spacing.
// Returns styled arrow if canScroll is true, otherwise returns empty spacing.
func RenderScrollDownIndicator(canScroll bool) string {
	return RenderScrollDownIndicatorWithConfig(canScroll, DefaultScrollConfig())
}

// RenderScrollUpIndicatorWithConfig renders an up scroll indicator with custom configuration.
func RenderScrollUpIndicatorWithConfig(canScroll bool, config ScrollIndicatorConfig) string {
	if canScroll {
		arrow := config.Padding + "▲"
		if config.IncludeMore {
			arrow += " more"
		}
		return ScrollIndicatorStyle.Render(arrow) + "\n"
	}
	if config.TrailingSpace {
		// Calculate equivalent spacing
		spaceLen := len(config.Padding) + 1 // padding + arrow
		if config.IncludeMore {
			spaceLen += 5 // " more"
		}
		return strings.Repeat(" ", spaceLen) + "\n"
	}
	return ""
}

// RenderScrollDownIndicatorWithConfig renders a down scroll indicator with custom configuration.
func RenderScrollDownIndicatorWithConfig(canScroll bool, config ScrollIndicatorConfig) string {
	if canScroll {
		arrow := config.Padding + "▼"
		if config.IncludeMore {
			arrow += " more"
		}
		return ScrollIndicatorStyle.Render(arrow) + "\n"
	}
	if config.TrailingSpace {
		// Calculate equivalent spacing
		spaceLen := len(config.Padding) + 1 // padding + arrow
		if config.IncludeMore {
			spaceLen += 5 // " more"
		}
		return strings.Repeat(" ", spaceLen) + "\n"
	}
	return ""
}

// RenderScrollIndicators renders both up and down scroll indicators based on current scroll state.
// This is the unified method for rendering scroll indicators throughout the UI.
// Returns upIndicator, downIndicator strings (both include newlines when non-empty).
func RenderScrollIndicators(isScrollable, canScrollUp, canScrollDown bool, config ScrollIndicatorConfig) (upIndicator, downIndicator string) {
	if !isScrollable {
		return "", ""
	}
	upIndicator = RenderScrollUpIndicatorWithConfig(canScrollUp, config)
	downIndicator = RenderScrollDownIndicatorWithConfig(canScrollDown, config)
	return upIndicator, downIndicator
}

// RenderScrollHint renders a text-based scroll hint for dialogs/modals.
// This uses text like "↑↓ more" rather than arrow symbols for inline hints.
func RenderScrollHint(canScrollUp, canScrollDown bool, padding string) string {
	if canScrollUp && canScrollDown {
		return ScrollIndicatorStyle.Render(padding + "▲▼ more")
	} else if canScrollUp {
		return ScrollIndicatorStyle.Render(padding + "▲ more above")
	} else if canScrollDown {
		return ScrollIndicatorStyle.Render(padding + "▼ more below")
	}
	return ""
}

// DetailPanelContent holds the parameters for rendering a detail panel
type DetailPanelContent struct {
	Header       string // Header text (e.g., resource name)
	Content      string // Main content to display
	Width        int    // Panel width
	Height       int    // Panel height
	ScrollOffset int    // Current scroll position
}

// DetailPanelResult holds the result of rendering a detail panel
type DetailPanelResult struct {
	Rendered        string   // The fully rendered panel
	VisibleLines    []string // The visible content lines (for selection extraction)
	NewScrollOffset int      // Adjusted scroll offset (clamped to valid range)
}

// RenderDetailPanel renders a scrollable detail panel with header and content.
// This consolidates the common rendering logic between DetailPanel and HistoryDetailPanel.
func RenderDetailPanel(params DetailPanelContent) DetailPanelResult {
	panelWidth := params.Width
	panelHeight := params.Height

	// Build header
	header := LabelStyle.Render(params.Header)

	// Calculate content height (subtract header, blank line, border, padding)
	headerHeight := lipgloss.Height(header)
	contentHeight := panelHeight - headerHeight - 5 // header + blank line + border(2) + padding(2)
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Apply scrolling to content
	contentLines := strings.Split(params.Content, "\n")
	scrollOffset := params.ScrollOffset
	if scrollOffset >= len(contentLines) {
		scrollOffset = len(contentLines) - 1
		if scrollOffset < 0 {
			scrollOffset = 0
		}
	}

	// Get visible portion
	endIdx := scrollOffset + contentHeight
	if endIdx > len(contentLines) {
		endIdx = len(contentLines)
	}
	visibleLines := contentLines[scrollOffset:endIdx]
	visibleContent := strings.Join(visibleLines, "\n")

	// Add scroll indicator if needed
	if len(contentLines) > contentHeight {
		scrollInfo := DimStyle.Render(fmt.Sprintf(" [%d/%d]", scrollOffset+1, len(contentLines)))
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

	return DetailPanelResult{
		Rendered:        panelStyle.Render(body),
		VisibleLines:    visibleLines,
		NewScrollOffset: scrollOffset,
	}
}
