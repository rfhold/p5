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
	itemCount := r.effectiveItemCount()

	// Handle filter with no matches
	if r.filter.Applied() && itemCount == 0 {
		var b strings.Builder
		b.WriteString(DimStyle.Render("No matches"))
		b.WriteString("\n\n")
		b.WriteString(RenderFilterBar(&r.filter, 0, len(r.visibleIdx), r.Width()))
		paddedStyle := lipgloss.NewStyle().Padding(1, 2)
		return paddedStyle.Render(b.String())
	}

	if len(r.visibleIdx) == 0 {
		return RenderCenteredMessage("No resources", r.Width(), r.Height())
	}

	var b strings.Builder
	visible := r.visibleHeight()
	endIdx := min(r.scrollOffset+visible, itemCount)

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
		visIdx := r.effectiveIndex(i)
		if visIdx < 0 || visIdx >= len(r.visibleIdx) {
			continue
		}
		itemIdx := r.visibleIdx[visIdx]
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
		totalLines := itemCount
		hint := DimStyle.Render(fmt.Sprintf("  [%d-%d/%d]", startLine, endLine, totalLines))
		scrollHint := RenderScrollHint(canScrollUp, canScrollDown, " ")
		b.WriteString(hint)
		if scrollHint != "" {
			b.WriteString(" ")
			b.WriteString(scrollHint)
		}
		b.WriteString("\n")
	}

	// Add filter bar at bottom when active or applied
	if r.filter.ActiveOrApplied() {
		filterBar := RenderFilterBar(&r.filter, itemCount, len(r.visibleIdx), r.Width())
		b.WriteString(filterBar)
		b.WriteString("\n")
	}

	paddedStyle := lipgloss.NewStyle().Padding(1, 2)
	return paddedStyle.Render(b.String())
}

type opSymbolInfo struct {
	symbol string
	style  lipgloss.Style
}

func getOpSymbolInfo(op ResourceOp) opSymbolInfo {
	switch op {
	case OpCreate:
		return opSymbolInfo{"+", OpCreateStyle}
	case OpUpdate:
		return opSymbolInfo{"~", OpUpdateStyle}
	case OpDelete:
		return opSymbolInfo{"-", OpDeleteStyle}
	case OpReplace, OpCreateReplace, OpDeleteReplace:
		return opSymbolInfo{"+-", OpReplaceStyle}
	case OpRefresh:
		return opSymbolInfo{"↻", OpRefreshStyle}
	default:
		return opSymbolInfo{" ", DimStyle}
	}
}

type renderStyles struct {
	op, dim, value, cursor               lipgloss.Style
	flagTarget, flagReplace, flagExclude lipgloss.Style
	flagProtect                          lipgloss.Style
	tree                                 lipgloss.Style
	bg                                   lipgloss.Color
	hasBackground                        bool
}

func newRenderStyles(opStyle lipgloss.Style, isFlashing, isSelected bool) renderStyles {
	rs := renderStyles{
		op:          opStyle,
		dim:         DimStyle,
		value:       ValueStyle,
		cursor:      CursorStyle,
		flagTarget:  FlagTargetStyle,
		flagReplace: FlagReplaceStyle,
		flagExclude: FlagExcludeStyle,
		flagProtect: FlagProtectStyle,
		tree:        TreeLineStyle,
	}

	if isFlashing {
		rs.bg = ColorFlash
		rs.hasBackground = true
	} else if isSelected {
		rs.bg = ColorSelection
		rs.hasBackground = true
	}

	if rs.hasBackground {
		rs.op = rs.op.Background(rs.bg)
		rs.dim = rs.dim.Background(rs.bg)
		rs.value = rs.value.Background(rs.bg)
		rs.cursor = rs.cursor.Background(rs.bg)
		rs.flagTarget = rs.flagTarget.Background(rs.bg)
		rs.flagReplace = rs.flagReplace.Background(rs.bg)
		rs.flagExclude = rs.flagExclude.Background(rs.bg)
		rs.flagProtect = rs.flagProtect.Background(rs.bg)
		rs.tree = rs.tree.Background(rs.bg)
	}

	return rs
}

func (r *ResourceList) buildFlagBadges(urn string, styles renderStyles) string {
	flags := r.flags[urn]
	var badges []string
	if flags.Target {
		badges = append(badges, styles.flagTarget.Render("[T]"))
	}
	if flags.Replace {
		badges = append(badges, styles.flagReplace.Render("[R]"))
	}
	if flags.Exclude {
		badges = append(badges, styles.flagExclude.Render("[E]"))
	}
	if len(badges) == 0 {
		return ""
	}
	if styles.hasBackground {
		return lipgloss.NewStyle().Background(styles.bg).Render("  ") + strings.Join(badges, "")
	}
	return "  " + strings.Join(badges, "")
}

func buildProtectBadge(protected bool, styles renderStyles) string {
	if !protected {
		return ""
	}
	if styles.hasBackground {
		return lipgloss.NewStyle().Background(styles.bg).Render("  ") + styles.flagProtect.Render("[Protected]")
	}
	return "  " + styles.flagProtect.Render("[Protected]")
}

func (r *ResourceList) renderItem(item ResourceItem, isCursor, isSelected, isFlashing bool, ancestorIsLast []bool) string {
	opInfo := getOpSymbolInfo(item.Op)
	styles := newRenderStyles(opInfo.style, isFlashing, isSelected)

	cursor := r.renderCursor(isCursor, styles)
	treePrefix := buildTreePrefix(item, ancestorIsLast, styles.hasBackground, styles.bg, styles.tree)
	statusIcon := r.renderStatusIcon(item.Status, item.Op, item.CurrentOp)
	if statusIcon != "" {
		statusIcon = " " + statusIcon
	}

	opStr := styles.op.Render(fmt.Sprintf("[%s]", opInfo.symbol))
	maxTypeLen := r.calculateMaxTypeLen(item)
	typeStr := styles.dim.Render(truncateMiddle(item.Type, maxTypeLen))
	nameStr := styles.value.Render(item.Name)
	protectBadge := buildProtectBadge(item.Protected, styles)
	flagBadges := r.buildFlagBadges(item.URN, styles)

	if styles.hasBackground {
		bgStyle := lipgloss.NewStyle().Background(styles.bg)
		return fmt.Sprintf("%s%s%s%s%s%s%s%s%s%s", cursor, treePrefix, opStr, bgStyle.Render(" "), typeStr, bgStyle.Render("  "), nameStr, protectBadge, flagBadges, statusIcon)
	}
	return fmt.Sprintf("%s%s%s %s  %s%s%s%s", cursor, treePrefix, opStr, typeStr, nameStr, protectBadge, flagBadges, statusIcon)
}

func (r *ResourceList) renderCursor(isCursor bool, styles renderStyles) string {
	if isCursor {
		return styles.cursor.Render("> ")
	}
	if styles.hasBackground {
		return lipgloss.NewStyle().Background(styles.bg).Render("  ")
	}
	return "  "
}

func (r *ResourceList) calculateMaxTypeLen(item ResourceItem) int {
	maxTypeLen := DefaultMaxTypeLength
	if r.Width() > 0 {
		treePrefixLen := item.Depth * 3
		otherElements := 2 + treePrefixLen + 4 + 3 + len(item.Name) + 12 + 20 + 4
		available := r.Width() - otherElements
		if available > MinTypeLength && available < maxTypeLen {
			maxTypeLen = available
		}
	}
	return maxTypeLen
}

func (r *ResourceList) renderStatusIcon(status ItemStatus, op, currentOp ResourceOp) string {
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

func buildTreePrefix(item ResourceItem, ancestorIsLast []bool, hasBackground bool, bg lipgloss.Color, treeStyle lipgloss.Style) string {
	if item.Depth == 0 {
		return ""
	}

	var treeParts []string
	for i := range item.Depth - 1 {
		treeParts = append(treeParts, buildAncestorSegment(i, ancestorIsLast, hasBackground, bg, treeStyle))
	}

	if item.IsLast {
		treeParts = append(treeParts, treeStyle.Render("└─ "))
	} else {
		treeParts = append(treeParts, treeStyle.Render("├─ "))
	}
	return strings.Join(treeParts, "")
}

func buildAncestorSegment(i int, ancestorIsLast []bool, hasBackground bool, bg lipgloss.Color, treeStyle lipgloss.Style) string {
	if i < len(ancestorIsLast) && ancestorIsLast[i] {
		if hasBackground {
			return lipgloss.NewStyle().Background(bg).Render("   ")
		}
		return "   "
	}
	return treeStyle.Render("│  ")
}
