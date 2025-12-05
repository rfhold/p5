package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// joinWithSeparator joins strings with a separator
func joinWithSeparator(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += sep + parts[i]
	}
	return result
}

// placeOverlay places an overlay string at the specified x,y position on the background
func placeOverlay(x, y int, overlay, background string) string {
	bgLines := strings.Split(background, "\n")
	overlayLines := strings.Split(overlay, "\n")

	for i, overlayLine := range overlayLines {
		bgIdx := y + i
		if bgIdx < 0 || bgIdx >= len(bgLines) {
			continue
		}

		bgLine := bgLines[bgIdx]

		// Truncate background line to x visual width and append overlay
		truncatedBg := truncateToWidth(bgLine, x)
		// Pad if needed
		currentWidth := lipgloss.Width(truncatedBg)
		if currentWidth < x {
			truncatedBg += strings.Repeat(" ", x-currentWidth)
		}

		bgLines[bgIdx] = truncatedBg + overlayLine
	}

	return strings.Join(bgLines, "\n")
}

// truncateToWidth truncates a string (which may contain ANSI codes) to the given visual width
func truncateToWidth(s string, width int) string {
	if width <= 0 {
		return ""
	}

	// Use lipgloss to handle ANSI-aware truncation
	style := lipgloss.NewStyle().MaxWidth(width)
	return style.Render(s)
}
