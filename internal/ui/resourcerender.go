package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// View renders the resource list component
func (r *ResourceList) View() string {
	if rendered, handled := r.RenderLoadingState(); handled {
		return rendered
	}
	return r.renderItems()
}

func (r *ResourceList) renderItems() string {
	if len(r.visibleIdx) == 0 {
		return RenderCenteredMessage("No resources", r.Width(), r.Height())
	}

	var b strings.Builder
	visible := r.visibleHeight()
	endIdx := r.scrollOffset + visible
	if endIdx > len(r.visibleIdx) {
		endIdx = len(r.visibleIdx)
	}

	// Check if content is scrollable at all
	scrollable := r.isScrollable()
	canScrollUp := !r.AtTop()
	canScrollDown := !r.AtBottom()

	// Determine visual selection range
	visualStart, visualEnd := -1, -1
	if r.visualMode {
		visualStart, visualEnd = r.visualStart, r.cursor
		if visualStart > visualEnd {
			visualStart, visualEnd = visualEnd, visualStart
		}
	}

	for i := r.scrollOffset; i < endIdx; i++ {
		itemIdx := r.visibleIdx[i]
		item := r.items[itemIdx]

		isCursor := i == r.cursor
		isSelected := r.visualMode && i >= visualStart && i <= visualEnd
		isFlashing := r.flashing && (r.flashAll || i == r.flashIdx)

		// Build ancestorIsLast by tracing back through parent chain
		ancestorIsLast := r.buildAncestorIsLast(itemIdx)

		line := r.renderItem(item, isCursor, isSelected, isFlashing, ancestorIsLast)
		b.WriteString(line)
		b.WriteString("\n")
	}

	// Add line count hint and scroll arrows at bottom (import modal style)
	if scrollable {
		startLine := r.scrollOffset + 1
		endLine := endIdx
		totalLines := len(r.visibleIdx)
		hint := DimStyle.Render(fmt.Sprintf("  [%d-%d/%d]", startLine, endLine, totalLines))
		scrollHint := RenderScrollHint(canScrollUp, canScrollDown, " ")
		b.WriteString(hint)
		if scrollHint != "" {
			b.WriteString(" ")
			b.WriteString(scrollHint)
		}
		b.WriteString("\n")
	}

	paddedStyle := lipgloss.NewStyle().Padding(1, 2)
	return paddedStyle.Render(b.String())
}

func (r *ResourceList) renderItem(item ResourceItem, isCursor, isSelected, isFlashing bool, ancestorIsLast []bool) string {
	// Op symbol and color
	var symbol string
	var opStyle lipgloss.Style

	switch item.Op {
	case OpCreate:
		symbol = "+"
		opStyle = OpCreateStyle
	case OpUpdate:
		symbol = "~"
		opStyle = OpUpdateStyle
	case OpDelete:
		symbol = "-"
		opStyle = OpDeleteStyle
	case OpReplace, OpCreateReplace, OpDeleteReplace:
		symbol = "+-"
		opStyle = OpReplaceStyle
	case OpRefresh:
		symbol = "↻"
		opStyle = OpRefreshStyle
	case OpSame:
		symbol = " "
		opStyle = DimStyle
	default:
		symbol = " "
		opStyle = DimStyle
	}

	// If selected or flashing, we need to apply background to each styled segment
	dimStyle := DimStyle
	valueStyle := ValueStyle
	cursorStyle := CursorStyle
	targetStyle := FlagTargetStyle
	replaceStyle := FlagReplaceStyle
	excludeStyle := FlagExcludeStyle
	treeStyle := TreeLineStyle

	// Determine background color
	var bg lipgloss.Color
	hasBackground := false
	if isFlashing {
		bg = ColorFlash
		hasBackground = true
	} else if isSelected {
		bg = ColorSelection
		hasBackground = true
	}

	if hasBackground {
		opStyle = opStyle.Background(bg)
		dimStyle = dimStyle.Background(bg)
		valueStyle = valueStyle.Background(bg)
		cursorStyle = cursorStyle.Background(bg)
		targetStyle = targetStyle.Background(bg)
		replaceStyle = replaceStyle.Background(bg)
		excludeStyle = excludeStyle.Background(bg)
		treeStyle = treeStyle.Background(bg)
	}

	// Cursor indicator
	cursor := "  "
	if isCursor {
		cursor = cursorStyle.Render("> ")
	} else if hasBackground {
		// Need to style the spaces too for consistent background
		cursor = lipgloss.NewStyle().Background(bg).Render("  ")
	}

	// Build tree prefix for nested items
	treePrefix := ""
	if item.Depth > 0 {
		var treeParts []string
		// For each ancestor level, draw vertical line or space
		for i := 0; i < item.Depth-1; i++ {
			if i < len(ancestorIsLast) && ancestorIsLast[i] {
				// Ancestor was last child, no vertical line needed
				if hasBackground {
					treeParts = append(treeParts, lipgloss.NewStyle().Background(bg).Render("   "))
				} else {
					treeParts = append(treeParts, "   ")
				}
			} else {
				// Draw vertical line
				treeParts = append(treeParts, treeStyle.Render("│  "))
			}
		}
		// Draw the connector for this item
		if item.IsLast {
			treeParts = append(treeParts, treeStyle.Render("└─ "))
		} else {
			treeParts = append(treeParts, treeStyle.Render("├─ "))
		}
		treePrefix = strings.Join(treeParts, "")
	}

	// Status icon for execution
	statusIcon := r.renderStatusIcon(item.Status, item.Op, item.CurrentOp)
	if statusIcon != "" {
		statusIcon = " " + statusIcon
	}

	// Format: > [tree] [+] type  name  [T][R][E]  status
	opStr := opStyle.Render(fmt.Sprintf("[%s]", symbol))

	// Calculate max width for type to prevent overflow
	// Account for: cursor(2) + tree prefix(3*depth) + op(4) + spacing(3) + name + badges + status
	// Use a reasonable max of 50 chars for type, but adjust based on available width
	maxTypeLen := 50
	if r.Width() > 0 {
		// Estimate other elements: cursor(2) + op(4) + spaces(3) + name(~20) + badges(~12) + status(~20) + padding(4)
		treePrefixLen := item.Depth * 3
		otherElements := 2 + treePrefixLen + 4 + 3 + len(item.Name) + 12 + 20 + 4
		available := r.Width() - otherElements
		if available > 20 && available < maxTypeLen {
			maxTypeLen = available
		}
	}
	truncatedType := truncateMiddle(item.Type, maxTypeLen)
	typeStr := dimStyle.Render(truncatedType)
	nameStr := valueStyle.Render(item.Name)

	// Build flag badges
	flags := r.flags[item.URN]
	var badges []string
	if flags.Target {
		badges = append(badges, targetStyle.Render("[T]"))
	}
	if flags.Replace {
		badges = append(badges, replaceStyle.Render("[R]"))
	}
	if flags.Exclude {
		badges = append(badges, excludeStyle.Render("[E]"))
	}
	badgeStr := ""
	if len(badges) > 0 {
		if hasBackground {
			badgeStr = lipgloss.NewStyle().Background(bg).Render("  ") + strings.Join(badges, "")
		} else {
			badgeStr = "  " + strings.Join(badges, "")
		}
	}

	// Build the line with styled separators for background highlight
	var line string
	if hasBackground {
		bgStyle := lipgloss.NewStyle().Background(bg)
		line = fmt.Sprintf("%s%s%s%s%s%s%s%s%s", cursor, treePrefix, opStr, bgStyle.Render(" "), typeStr, bgStyle.Render("  "), nameStr, badgeStr, statusIcon)
	} else {
		line = fmt.Sprintf("%s%s%s %s  %s%s%s", cursor, treePrefix, opStr, typeStr, nameStr, badgeStr, statusIcon)
	}

	return line
}

func (r *ResourceList) renderStatusIcon(status ItemStatus, op ResourceOp, currentOp ResourceOp) string {
	switch status {
	case StatusPending:
		return StatusPendingStyle.Render("pending")
	case StatusRunning:
		// Show what step is currently running with appropriate color
		return r.getRunningStatusText(currentOp)
	case StatusSuccess:
		// Show completed status with appropriate color based on final op
		return r.getCompletedStatusText(op)
	case StatusFailed:
		return StatusFailedStyle.Render("failed")
	default:
		return ""
	}
}

// getRunningStatusText returns a color-coded status text for a running operation
func (r *ResourceList) getRunningStatusText(op ResourceOp) string {
	switch op {
	case OpCreate:
		return OpCreateStyle.Render("creating...")
	case OpUpdate:
		return OpUpdateStyle.Render("updating...")
	case OpDelete:
		return OpDeleteStyle.Render("deleting...")
	case OpReplace:
		return OpReplaceStyle.Render("replacing...")
	case OpCreateReplace:
		return OpCreateStyle.Render("creating replacement...")
	case OpDeleteReplace:
		return OpDeleteStyle.Render("deleting original...")
	case OpRefresh:
		return OpRefreshStyle.Render("refreshing...")
	case OpRead:
		return StatusRunningStyle.Render("reading...")
	default:
		return StatusRunningStyle.Render("running...")
	}
}

// getCompletedStatusText returns a color-coded status text for a completed operation
func (r *ResourceList) getCompletedStatusText(op ResourceOp) string {
	switch op {
	case OpCreate:
		return OpCreateStyle.Render("created")
	case OpUpdate:
		return OpUpdateStyle.Render("updated")
	case OpDelete:
		return OpDeleteStyle.Render("deleted")
	case OpReplace, OpCreateReplace, OpDeleteReplace:
		return OpReplaceStyle.Render("replaced")
	case OpRefresh:
		return OpRefreshStyle.Render("refreshed")
	case OpRead:
		return StatusSuccessStyle.Render("read")
	case OpSame:
		return StatusSuccessStyle.Render("unchanged")
	default:
		return StatusSuccessStyle.Render("done")
	}
}

// FlashClearMsg is sent to clear the flash highlight
type FlashClearMsg struct{}

// ClearFlash clears the flash highlight
func (r *ResourceList) ClearFlash() {
	r.flashing = false
	r.flashIdx = -1
	r.flashAll = false
}
