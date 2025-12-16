package ui

import "github.com/charmbracelet/lipgloss"

// Color palette (Tokyo Night)
var (
	ColorPrimary           = lipgloss.Color("#7aa2f7")
	ColorSecondary         = lipgloss.Color("#bb9af7")
	ColorText              = lipgloss.Color("#c0caf5")
	ColorDim               = lipgloss.Color("#565f89")
	ColorError             = lipgloss.Color("#f7768e")
	ColorBg                = lipgloss.Color("#1a1b26")
	ColorSelection         = lipgloss.Color("#283457") // subtle selection highlight (visual range)
	ColorDiscreteSelection = lipgloss.Color("#3d4f2f") // discrete selection (green-ish)
	ColorBothSelection     = lipgloss.Color("#4a3f5c") // both visual and discrete (purple-ish)
	ColorFlash             = lipgloss.Color("#3d59a1") // brighter flash highlight

	// Operation colors
	ColorCreate  = lipgloss.Color("#9ece6a") // green
	ColorUpdate  = lipgloss.Color("#e0af68") // yellow/orange
	ColorDelete  = lipgloss.Color("#f7768e") // red
	ColorReplace = lipgloss.Color("#bb9af7") // purple
	ColorRefresh = lipgloss.Color("#7dcfff") // cyan
	ColorSuccess = lipgloss.Color("#9ece6a") // green (same as create)

	// Flag colors
	ColorTarget  = lipgloss.Color("#7dcfff") // cyan
	ColorExclude = lipgloss.Color("#f7768e") // red (same as error/delete)
	ColorProtect = lipgloss.Color("#f5a623") // masterlock yellow
)

// Styles
var (
	// Text styles
	LabelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSecondary)

	ValueStyle = lipgloss.NewStyle().
			Foreground(ColorText)

	DimStyle = lipgloss.NewStyle().
			Foreground(ColorDim)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorError)

	// Box styles
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorDim).
			Padding(0, 1)

	// Dialog styles
	DialogStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(1, 2)

	DialogTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorPrimary).
				MarginBottom(1)

	// Operation styles
	OpCreateStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorCreate)

	OpUpdateStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorUpdate)

	OpDeleteStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorDelete)

	OpReplaceStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorReplace)

	OpRefreshStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorRefresh)

	// Execution status styles
	StatusPendingStyle = lipgloss.NewStyle().Foreground(ColorDim)
	StatusRunningStyle = lipgloss.NewStyle().Foreground(ColorPrimary)
	StatusSuccessStyle = lipgloss.NewStyle().Foreground(ColorSuccess)
	StatusFailedStyle  = lipgloss.NewStyle().Foreground(ColorError)

	// Scroll indicator styles - bright cyan for high visibility
	ScrollIndicatorStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorRefresh) // Use bright cyan for better visibility

	// Cursor and selection styles
	CursorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary)

	SelectionStyle = lipgloss.NewStyle().
			Background(ColorSelection)

	// Flag badge styles
	FlagTargetStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorTarget)

	FlagReplaceStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorReplace)

	FlagExcludeStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorExclude)

	FlagProtectStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorProtect)

	// View mode label styles
	ViewLabelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary)

	// Tree connector style for component resources
	TreeLineStyle = lipgloss.NewStyle().
			Foreground(ColorDim)
)

// Status icons
const (
	IconPending = "○"
	IconRunning = "◐" // or use spinner
	IconSuccess = "✓"
	IconFailed  = "✗"
)

// Layout constants for UI components
const (
	// Text input widths
	DefaultInputWidth = 50

	// Modal and dialog dimensions
	DefaultModalMaxHeight  = 20
	DefaultDialogMaxWidth  = 80
	DefaultDialogMaxHeight = 30
	MinContentWidth        = 20
	MinContentHeight       = 5
	DialogPaddingAllowance = 6  // Padding around dialog content
	DialogChromeAllowance  = 10 // Title, summary, footer space

	// Resource list rendering
	DefaultMaxTypeLength = 50
	MinTypeLength        = 20

	// Text formatting
	DefaultMaxStringLength   = 30
	MinFormattedStringLength = 20
	ArrayItemTruncateLength  = 30
	ArrayItemTruncateDisplay = 27 // Length to show before "..."
)

// RenderOp renders a resource operation with appropriate styling
func RenderOp(op ResourceOp) string {
	switch op {
	case OpCreate:
		return OpCreateStyle.Render("create")
	case OpUpdate:
		return OpUpdateStyle.Render("update")
	case OpDelete:
		return OpDeleteStyle.Render("delete")
	case OpReplace:
		return OpReplaceStyle.Render("replace")
	case OpCreateReplace:
		return OpCreateStyle.Render("create-replacement")
	case OpDeleteReplace:
		return OpDeleteStyle.Render("delete-replaced")
	case OpRefresh:
		return OpRefreshStyle.Render("refresh")
	case OpSame:
		return DimStyle.Render("unchanged")
	default:
		return DimStyle.Render(string(op))
	}
}

// RenderStatus renders a status with appropriate styling
func RenderStatus(status ItemStatus) string {
	switch status {
	case StatusNone:
		return DimStyle.Render("none")
	case StatusPending:
		return StatusPendingStyle.Render("pending")
	case StatusRunning:
		return StatusRunningStyle.Render("running")
	case StatusSuccess:
		return StatusSuccessStyle.Render("success")
	case StatusFailed:
		return StatusFailedStyle.Render("failed")
	default:
		return DimStyle.Render("unknown")
	}
}

// RenderHistoryKind renders a history operation kind with appropriate styling
func RenderHistoryKind(kind string) string {
	switch kind {
	case "update":
		return OpCreateStyle.Render("update")
	case "refresh":
		return OpRefreshStyle.Render("refresh")
	case "destroy":
		return OpDeleteStyle.Render("destroy")
	case "preview":
		return DimStyle.Render("preview")
	default:
		return DimStyle.Render(kind)
	}
}

// RenderHistoryResult renders a history operation result with appropriate styling
func RenderHistoryResult(result string) string {
	switch result {
	case "succeeded":
		return StatusSuccessStyle.Render("succeeded")
	case "failed":
		return StatusFailedStyle.Render("failed")
	case "in-progress":
		return StatusRunningStyle.Render("in-progress")
	default:
		return DimStyle.Render(result)
	}
}
