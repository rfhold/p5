package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
)

// ViewMode represents the current view
type ViewMode int

const (
	ViewStack ViewMode = iota
	ViewPreview
	ViewExecute
	ViewHistory
)

func (v ViewMode) String() string {
	switch v {
	case ViewStack:
		return "Stack"
	case ViewPreview:
		return "Preview"
	case ViewExecute:
		return "Execute"
	case ViewHistory:
		return "History"
	default:
		return "Unknown"
	}
}

// HeaderData contains the data displayed in the header
type HeaderData struct {
	ProgramName string
	StackName   string
	Runtime     string
}

// Header renders the top header bar
type Header struct {
	spinner   spinner.Model
	data      *HeaderData
	summary   *ResourceSummary
	viewMode  ViewMode
	operation OperationType
	state     HeaderState
	err       error
	loading   bool
	width     int
}

// HeaderState represents the current state of the header
type HeaderState int

const (
	HeaderLoading HeaderState = iota
	HeaderRunning
	HeaderDone
	HeaderError
)

// NewHeader creates a new header component
func NewHeader() Header {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorPrimary)

	return Header{
		spinner:  s,
		loading:  true,
		viewMode: ViewStack,
	}
}

// SetData sets the header data
func (h *Header) SetData(data *HeaderData) {
	h.data = data
	h.loading = false
}

// SetError sets an error state
func (h *Header) SetError(err error) {
	h.err = err
	h.loading = false
	h.state = HeaderError
}

// SetWidth sets the header width
func (h *Header) SetWidth(width int) {
	h.width = width
}

// SetViewMode sets the current view mode
func (h *Header) SetViewMode(mode ViewMode) {
	h.viewMode = mode
}

// SetOperation sets the current operation type
func (h *Header) SetOperation(op OperationType) {
	h.operation = op
}

// SetSummary updates the resource summary in the header
func (h *Header) SetSummary(summary ResourceSummary, state HeaderState) {
	h.summary = &summary
	h.state = state
}

// SetPreviewSummary is a compatibility method that converts PreviewSummary to ResourceSummary
func (h *Header) SetPreviewSummary(summary PreviewSummary, state PreviewState) {
	h.summary = &ResourceSummary{
		Total:   summary.Total,
		Create:  summary.Create,
		Update:  summary.Update,
		Delete:  summary.Delete,
		Replace: summary.Replace,
	}
	// Convert preview state to header state
	switch state {
	case PreviewLoading:
		h.state = HeaderLoading
	case PreviewRunning:
		h.state = HeaderRunning
	case PreviewDone:
		h.state = HeaderDone
	case PreviewError:
		h.state = HeaderError
	}
}

// IsLoading returns whether the header is in loading state
func (h *Header) IsLoading() bool {
	return h.loading || h.state == HeaderLoading || h.state == HeaderRunning
}

// Spinner returns the spinner model for updates
func (h *Header) Spinner() spinner.Model {
	return h.spinner
}

// SetSpinner updates the spinner model
func (h *Header) SetSpinner(s spinner.Model) {
	h.spinner = s
}

// View renders the header
func (h *Header) View() string {
	var topRow string
	var bottomRow string

	if h.loading {
		topRow = fmt.Sprintf("%s Loading...", h.spinner.View())
	} else if h.err != nil {
		topRow = ErrorStyle.Render(fmt.Sprintf("Error: %v", h.err))
	} else if h.data != nil {
		program := fmt.Sprintf("%s %s",
			LabelStyle.Render("Program:"),
			ValueStyle.Render(h.data.ProgramName))

		stack := fmt.Sprintf("%s %s",
			LabelStyle.Render("Stack:"),
			ValueStyle.Render(orDefault(h.data.StackName, "(none)")))

		runtime := fmt.Sprintf("%s %s",
			LabelStyle.Render("Runtime:"),
			ValueStyle.Render(orDefault(h.data.Runtime, "?")))

		topRow = lipgloss.JoinHorizontal(lipgloss.Center,
			program,
			DimStyle.Render("  │  "),
			stack,
			DimStyle.Render("  │  "),
			runtime,
		)
	}

	// Render view mode and summary row
	bottomRow = h.renderSummaryRow()

	// Combine rows
	content := topRow
	if bottomRow != "" {
		content = lipgloss.JoinVertical(lipgloss.Left, topRow, bottomRow)
	}

	return BoxStyle.Width(h.width - 2).Render(content)
}

// renderSummaryRow renders the view mode and summary line
func (h *Header) renderSummaryRow() string {
	var parts []string

	// View mode label
	viewLabel := h.viewMode.String()
	if h.viewMode != ViewStack && h.viewMode != ViewHistory {
		viewLabel = fmt.Sprintf("%s %s", h.viewMode.String(), h.operation.String())
	}

	// Status indicator
	switch h.state {
	case HeaderLoading:
		parts = append(parts, fmt.Sprintf("%s %s", h.spinner.View(), DimStyle.Render("Loading...")))
		return strings.Join(parts, "  ")
	case HeaderRunning:
		parts = append(parts, fmt.Sprintf("%s %s", h.spinner.View(), ViewLabelStyle.Render(viewLabel)))
	case HeaderDone:
		parts = append(parts, ViewLabelStyle.Render(viewLabel))
	case HeaderError:
		parts = append(parts, ErrorStyle.Render(viewLabel+" failed"))
		return strings.Join(parts, "  ")
	default:
		// For stack view with no operation yet
		if h.viewMode == ViewStack {
			parts = append(parts, ViewLabelStyle.Render(viewLabel))
		}
	}

	// Summary counts
	if h.summary != nil {
		total := h.summary.Create + h.summary.Update + h.summary.Delete + h.summary.Replace + h.summary.Refresh
		// For stack view, always show the resource count
		if h.viewMode == ViewStack {
			parts = append(parts, DimStyle.Render(fmt.Sprintf("%d resources", h.summary.Total)))
		} else if h.viewMode == ViewHistory {
			// For history view, show number of history entries
			parts = append(parts, DimStyle.Render(fmt.Sprintf("%d updates", h.summary.Total)))
		} else if total == 0 && h.state == HeaderDone {
			parts = append(parts, DimStyle.Render("No changes"))
		} else if total > 0 {
			var countParts []string
			if h.summary.Create > 0 {
				countParts = append(countParts, OpCreateStyle.Render(fmt.Sprintf("+%d", h.summary.Create)))
			}
			if h.summary.Update > 0 {
				countParts = append(countParts, OpUpdateStyle.Render(fmt.Sprintf("~%d", h.summary.Update)))
			}
			if h.summary.Replace > 0 {
				countParts = append(countParts, OpReplaceStyle.Render(fmt.Sprintf("±%d", h.summary.Replace)))
			}
			if h.summary.Delete > 0 {
				countParts = append(countParts, OpDeleteStyle.Render(fmt.Sprintf("-%d", h.summary.Delete)))
			}
			if h.summary.Refresh > 0 {
				countParts = append(countParts, OpRefreshStyle.Render(fmt.Sprintf("↻%d", h.summary.Refresh)))
			}
			if len(countParts) > 0 {
				parts = append(parts, strings.Join(countParts, " "))
			}
		}
	}

	return strings.Join(parts, "  ")
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
