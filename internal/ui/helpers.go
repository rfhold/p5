package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
)

// getMapValue safely gets a value from a map
func getMapValue(m map[string]any, key string) (any, bool) {
	if m == nil {
		return nil, false
	}
	val, ok := m[key]
	return val, ok
}

// valuesEqual compares two values for equality
func valuesEqual(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Compare maps recursively
	aMap, aIsMap := a.(map[string]any)
	bMap, bIsMap := b.(map[string]any)
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
	aSlice, aIsSlice := a.([]any)
	bSlice, bIsSlice := b.([]any)
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
	for i := range len(s) - 1 {
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
func FormatTime(timeStr, format string) string {
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return timeStr
	}
	return t.Format(format)
}

// FormatTimeStyled parses an RFC3339 time string and formats it with styling.
// If parsing fails and the string is longer than maxLen, it truncates the string.
// Pass maxLen <= 0 to skip truncation on parse failure.
func FormatTimeStyled(timeStr, format string, maxLen int, style lipgloss.Style) string {
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
	switch {
	case canScrollUp && canScrollDown:
		return ScrollIndicatorStyle.Render(padding + "▲▼ more")
	case canScrollUp:
		return ScrollIndicatorStyle.Render(padding + "▲ more above")
	case canScrollDown:
		return ScrollIndicatorStyle.Render(padding + "▼ more below")
	default:
		return ""
	}
}

// ResourceChangesFormat specifies the output format for RenderResourceChanges
type ResourceChangesFormat int

const (
	// ResourceChangesCompact renders as: +2 ~1 ±1 -0
	ResourceChangesCompact ResourceChangesFormat = iota
	// ResourceChangesExpanded renders as: + 2 created, ~ 1 updated, etc.
	ResourceChangesExpanded
)

// RenderResourceChanges renders a map of resource changes with appropriate styling.
// The changes map typically has keys: "create", "update", "delete", "replace", "same".
func RenderResourceChanges(changes map[string]int, format ResourceChangesFormat) string {
	if len(changes) == 0 {
		return DimStyle.Render("no changes")
	}

	create := changes["create"]
	update := changes["update"]
	del := changes["delete"]
	replace := changes["replace"]
	same := changes["same"]

	var parts []string

	switch format {
	case ResourceChangesCompact:
		if create > 0 {
			parts = append(parts, OpCreateStyle.Render(fmt.Sprintf("+%d", create)))
		}
		if update > 0 {
			parts = append(parts, OpUpdateStyle.Render(fmt.Sprintf("~%d", update)))
		}
		if replace > 0 {
			parts = append(parts, OpReplaceStyle.Render(fmt.Sprintf("±%d", replace)))
		}
		if del > 0 {
			parts = append(parts, OpDeleteStyle.Render(fmt.Sprintf("-%d", del)))
		}

		if len(parts) == 0 {
			if same > 0 {
				return DimStyle.Render(fmt.Sprintf("%d unchanged", same))
			}
			return DimStyle.Render("no changes")
		}
		return strings.Join(parts, " ")

	case ResourceChangesExpanded:
		if create > 0 {
			parts = append(parts, OpCreateStyle.Render(fmt.Sprintf("  + %d created", create)))
		}
		if update > 0 {
			parts = append(parts, OpUpdateStyle.Render(fmt.Sprintf("  ~ %d updated", update)))
		}
		if replace > 0 {
			parts = append(parts, OpReplaceStyle.Render(fmt.Sprintf("  ± %d replaced", replace)))
		}
		if del > 0 {
			parts = append(parts, OpDeleteStyle.Render(fmt.Sprintf("  - %d deleted", del)))
		}
		if same > 0 {
			parts = append(parts, DimStyle.Render(fmt.Sprintf("  = %d unchanged", same)))
		}

		if len(parts) == 0 {
			return DimStyle.Render("no changes")
		}
		return strings.Join(parts, "\n")
	}

	return DimStyle.Render("no changes")
}

// CursorState holds cursor and scroll state for list components
type CursorState struct {
	Cursor       int
	ScrollOffset int
}

// MoveCursor moves the cursor by delta, clamping to valid range [0, itemCount-1].
// Returns the new cursor position.
func MoveCursor(cursor, delta, itemCount int) int {
	cursor += delta
	if cursor < 0 {
		cursor = 0
	}
	if itemCount > 0 && cursor >= itemCount {
		cursor = itemCount - 1
	}
	return cursor
}

// EnsureCursorVisible adjusts scroll offset to keep cursor visible within the viewport.
// Returns the new scroll offset.
func EnsureCursorVisible(cursor, scrollOffset, itemCount, visibleHeight int) int {
	if itemCount == 0 {
		return 0
	}

	// Scroll up if cursor is above visible area
	if cursor < scrollOffset {
		scrollOffset = cursor
	}

	// Scroll down if cursor is below visible area
	if cursor >= scrollOffset+visibleHeight {
		scrollOffset = cursor - visibleHeight + 1
	}

	// Clamp scroll offset
	maxScroll := max(itemCount-visibleHeight, 0)
	if scrollOffset > maxScroll {
		scrollOffset = maxScroll
	}
	if scrollOffset < 0 {
		scrollOffset = 0
	}

	return scrollOffset
}

// CalculateVisibleHeight returns the number of lines available for items.
// Accounts for padding and scroll indicators if content is scrollable.
func CalculateVisibleHeight(totalHeight, itemCount, padding int) int {
	// Base height minus padding
	baseHeight := max(totalHeight-padding, 1)

	// Check if scrollable (more items than base height)
	if itemCount > baseHeight {
		// Reserve 2 lines for scroll indicators
		height := max(baseHeight-2, 1)
		return height
	}

	return baseHeight
}

// IsScrollable returns true if there are more items than can fit in the base height.
func IsScrollable(totalHeight, itemCount, padding int) bool {
	baseHeight := max(totalHeight-padding, 1)
	return itemCount > baseHeight
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
	contentHeight := max(
		// header + blank line + border(2) + padding(2)
		panelHeight-headerHeight-5, 1)

	// Apply scrolling to content
	contentLines := strings.Split(params.Content, "\n")
	scrollOffset := params.ScrollOffset
	if scrollOffset >= len(contentLines) {
		scrollOffset = max(len(contentLines)-1, 0)
	}

	// Get visible portion
	endIdx := min(scrollOffset+contentHeight, len(contentLines))
	visibleLines := contentLines[scrollOffset:endIdx]
	visibleContent := strings.Join(visibleLines, "\n")

	// Add scroll indicator if needed
	if len(contentLines) > contentHeight {
		scrollInfo := DimStyle.Render(fmt.Sprintf(" [%d/%d]", scrollOffset+1, len(contentLines)))
		header += scrollInfo
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
