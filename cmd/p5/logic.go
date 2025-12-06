package main

import (
	"path/filepath"

	"github.com/rfhold/p5/internal/plugins"
	"github.com/rfhold/p5/internal/pulumi"
	"github.com/rfhold/p5/internal/ui"
)

// PreviewEventResult holds the result of processing a preview event
type PreviewEventResult struct {
	// State changes
	NewOpState OperationState
	InitDone   bool // True if init should transition to InitComplete
	HasError   bool
	Error      error

	// Resource item to add (nil if none)
	Item *ui.ResourceItem
}

// ProcessPreviewEvent processes a preview event and returns state changes.
// This is a pure function - no side effects.
func ProcessPreviewEvent(event pulumi.PreviewEvent, currentOpState OperationState, initState InitState) PreviewEventResult {
	result := PreviewEventResult{
		NewOpState: currentOpState,
	}

	// First event transitions from Starting to Running
	if currentOpState == OpStarting {
		result.NewOpState = OpRunning
	}

	if event.Error != nil {
		result.NewOpState = OpError
		result.HasError = true
		result.Error = event.Error
		// Mark init complete even on error - we're done initializing
		if initState == InitLoadingResources {
			result.InitDone = true
		}
		return result
	}

	if event.Done {
		result.NewOpState = OpComplete
		// Mark init complete when preview finishes
		if initState == InitLoadingResources {
			result.InitDone = true
		}
		return result
	}

	if event.Step != nil {
		result.Item = convertPreviewStepToItem(event.Step)
	}

	return result
}

// convertPreviewStepToItem converts a PreviewStep to a ResourceItem.
// This handles the old/new state merging logic.
func convertPreviewStepToItem(step *pulumi.PreviewStep) *ui.ResourceItem {
	// New state inputs/outputs (for create/update)
	inputs := step.Inputs
	outputs := step.Outputs

	// Old state (for updates/deletes) - used for diff view
	var oldInputs, oldOutputs map[string]interface{}
	if step.Old != nil {
		oldInputs = step.Old.Inputs
		oldOutputs = step.Old.Outputs
		// For delete ops, use old as current since new doesn't exist
		if inputs == nil {
			inputs = step.Old.Inputs
		}
		if outputs == nil {
			outputs = step.Old.Outputs
		}
	}

	return &ui.ResourceItem{
		URN:        step.URN,
		Type:       step.Type,
		Name:       step.Name,
		Op:         step.Op,
		Status:     ui.StatusNone,
		Parent:     step.Parent,
		Inputs:     inputs,
		Outputs:    outputs,
		OldInputs:  oldInputs,
		OldOutputs: oldOutputs,
	}
}

// OperationEventResult holds the result of processing an operation event
type OperationEventResult struct {
	// State changes
	NewOpState OperationState
	HasError   bool
	Error      error
	Done       bool // True if operation is complete

	// Resource item to add/update (nil if none)
	Item *ui.ResourceItem
}

// ProcessOperationEvent processes an operation event and returns state changes.
// This is a pure function - no side effects.
func ProcessOperationEvent(event pulumi.OperationEvent, currentOpState OperationState) OperationEventResult {
	result := OperationEventResult{
		NewOpState: currentOpState,
	}

	// First event transitions from Starting to Running
	if currentOpState == OpStarting {
		result.NewOpState = OpRunning
	}

	if event.Error != nil {
		result.NewOpState = OpError
		result.HasError = true
		result.Error = event.Error
		return result
	}

	if event.Done {
		// Determine final state based on whether we were cancelling
		result.NewOpState = OpComplete
		result.Done = true
		return result
	}

	// Add items as events stream in
	if event.URN != "" {
		result.Item = convertOperationEventToItem(event)
	}

	return result
}

// convertOperationEventToItem converts an OperationEvent to a ResourceItem.
func convertOperationEventToItem(event pulumi.OperationEvent) *ui.ResourceItem {
	var status ui.ItemStatus
	switch event.Status {
	case pulumi.StepPending:
		status = ui.StatusPending
	case pulumi.StepRunning:
		status = ui.StatusRunning
	case pulumi.StepSuccess:
		status = ui.StatusSuccess
	case pulumi.StepFailed:
		status = ui.StatusFailed
	}

	return &ui.ResourceItem{
		URN:        event.URN,
		Type:       event.Type,
		Name:       event.Name,
		Op:         event.Op,
		Parent:     event.Parent,
		Status:     status,
		Inputs:     event.Inputs,
		Outputs:    event.Outputs,
		OldInputs:  event.OldInputs,
		OldOutputs: event.OldOutputs,
	}
}

// ConvertResourcesToItems converts pulumi ResourceInfo slice to UI ResourceItems.
// This is used when loading stack resources.
func ConvertResourcesToItems(resources []pulumi.ResourceInfo) []ui.ResourceItem {
	items := make([]ui.ResourceItem, 0, len(resources))
	for _, r := range resources {
		items = append(items, ui.ResourceItem{
			URN:     r.URN,
			Type:    r.Type,
			Name:    r.Name,
			Op:      pulumi.OpSame, // Stack view shows existing resources
			Status:  ui.StatusNone,
			Parent:  r.Parent,
			Inputs:  r.Inputs,
			Outputs: r.Outputs,
		})
	}
	return items
}

// ConvertHistoryToItems converts pulumi UpdateSummary slice to UI HistoryItems.
// For local backends where Version may be 0, it calculates version from index.
func ConvertHistoryToItems(history []pulumi.UpdateSummary) []ui.HistoryItem {
	items := make([]ui.HistoryItem, 0, len(history))
	for i, h := range history {
		version := h.Version
		// Pulumi local backend doesn't track version numbers, so use index
		// History is returned newest-first, so index 0 = most recent
		if version == 0 {
			version = len(history) - i
		}
		items = append(items, ui.HistoryItem{
			Version:         version,
			Kind:            h.Kind,
			StartTime:       h.StartTime,
			EndTime:         h.EndTime,
			Message:         h.Message,
			Result:          h.Result,
			ResourceChanges: h.ResourceChanges,
			User:            h.User,
			UserEmail:       h.UserEmail,
		})
	}
	return items
}

// ConvertImportSuggestions converts plugin import suggestions to UI format.
func ConvertImportSuggestions(suggestions []*plugins.AggregatedImportSuggestion) []ui.ImportSuggestion {
	items := make([]ui.ImportSuggestion, 0, len(suggestions))
	for _, s := range suggestions {
		items = append(items, ui.ImportSuggestion{
			ID:          s.Suggestion.Id,
			Label:       s.Suggestion.Label,
			Description: s.Suggestion.Description,
			PluginName:  s.PluginName,
		})
	}
	return items
}

// StacksConversionResult holds the result of converting stacks
type StacksConversionResult struct {
	Items            []ui.StackItem
	CurrentStackName string
}

// ConvertStacksToItems converts pulumi StackInfo slice to UI StackItems.
// Returns the converted items and the name of the current stack (if any).
func ConvertStacksToItems(stacks []pulumi.StackInfo) StacksConversionResult {
	result := StacksConversionResult{
		Items: make([]ui.StackItem, 0, len(stacks)),
	}
	for _, s := range stacks {
		result.Items = append(result.Items, ui.StackItem{
			Name:    s.Name,
			Current: s.Current,
		})
		if s.Current {
			result.CurrentStackName = s.Name
		}
	}
	return result
}

// ConvertWorkspacesToItems converts pulumi WorkspaceInfo slice to UI WorkspaceItems.
// cwd is used to compute relative paths; pass empty string to skip relative path calculation.
func ConvertWorkspacesToItems(workspaces []pulumi.WorkspaceInfo, cwd string) []ui.WorkspaceItem {
	items := make([]ui.WorkspaceItem, 0, len(workspaces))
	for _, w := range workspaces {
		relPath := w.Path
		if cwd != "" {
			if rel, err := filepath.Rel(cwd, w.Path); err == nil {
				relPath = rel
			}
		}
		items = append(items, ui.WorkspaceItem{
			Path:         w.Path,
			RelativePath: relPath,
			Name:         w.Name,
			Current:      w.Current,
		})
	}
	return items
}

// CanImportResource determines if the current selection can be imported.
// Import is only valid for create operations in preview view.
func CanImportResource(viewMode ui.ViewMode, selectedItem *ui.ResourceItem) bool {
	if viewMode != ui.ViewPreview {
		return false
	}
	if selectedItem == nil {
		return false
	}
	return selectedItem.Op == pulumi.OpCreate
}

// CanDeleteFromState determines if the current selection can be deleted from state.
// State delete is only valid in stack view and not for the root stack resource.
func CanDeleteFromState(viewMode ui.ViewMode, selectedItem *ui.ResourceItem) bool {
	if viewMode != ui.ViewStack {
		return false
	}
	if selectedItem == nil {
		return false
	}
	// Cannot delete the root stack resource
	return selectedItem.Type != "pulumi:pulumi:Stack"
}

// EscapeAction represents the action to take when escape is pressed
type EscapeAction int

const (
	EscapeActionNone           EscapeAction = iota // Do nothing
	EscapeActionExitVisualMode                     // Exit visual selection mode
	EscapeActionCancelOp                           // Cancel running operation
	EscapeActionNavigateBack                       // Navigate back to stack view
)

// String returns a human-readable name for the action.
func (a EscapeAction) String() string {
	switch a {
	case EscapeActionNone:
		return "None"
	case EscapeActionExitVisualMode:
		return "ExitVisualMode"
	case EscapeActionCancelOp:
		return "CancelOp"
	case EscapeActionNavigateBack:
		return "NavigateBack"
	default:
		return "Unknown"
	}
}

// DetermineEscapeAction determines what action to take when escape is pressed.
// This is a pure function that examines the current state without side effects.
func DetermineEscapeAction(viewMode ui.ViewMode, opState OperationState, visualMode bool) EscapeAction {
	// Cancel visual mode first (highest priority)
	if visualMode {
		return EscapeActionExitVisualMode
	}

	// Cancel running execution in execute view
	if viewMode == ui.ViewExecute && opState == OpRunning {
		return EscapeActionCancelOp
	}

	// Navigate back from preview, history, or completed execution
	if viewMode == ui.ViewPreview || viewMode == ui.ViewExecute || viewMode == ui.ViewHistory {
		if !opState.IsActive() || viewMode == ui.ViewHistory {
			return EscapeActionNavigateBack
		}
	}

	return EscapeActionNone
}

// StackInitAction represents the action to take after loading stacks
type StackInitAction int

const (
	StackInitActionNone         StackInitAction = iota // Not in init flow
	StackInitActionShowInit                            // Show stack init modal (no stacks exist)
	StackInitActionShowSelector                        // Show stack selector (stacks exist, none current)
	StackInitActionProceed                             // Proceed with current stack
)

// String returns a human-readable name for the action.
func (a StackInitAction) String() string {
	switch a {
	case StackInitActionNone:
		return "None"
	case StackInitActionShowInit:
		return "ShowInit"
	case StackInitActionShowSelector:
		return "ShowSelector"
	case StackInitActionProceed:
		return "Proceed"
	default:
		return "Unknown"
	}
}

// DetermineStackInitAction determines what action to take based on loaded stacks.
// Only returns a meaningful action when initState is InitLoadingStacks.
func DetermineStackInitAction(initState InitState, stackCount int, currentStackName string) StackInitAction {
	if initState != InitLoadingStacks {
		return StackInitActionNone
	}

	if stackCount == 0 {
		return StackInitActionShowInit
	}

	if currentStackName == "" {
		return StackInitActionShowSelector
	}

	return StackInitActionProceed
}

// PluginAuthSummary summarizes the results of plugin authentication
type PluginAuthSummary struct {
	// AuthenticatedPlugins is the list of plugins that provided credentials
	AuthenticatedPlugins []string
	// ErrorMessages is the list of error messages from failed plugins
	ErrorMessages []string
	// HasErrors is true if any plugin failed
	HasErrors bool
}

// SummarizePluginAuthResults processes plugin auth results into a summary.
func SummarizePluginAuthResults(results []plugins.AuthenticateResult) PluginAuthSummary {
	summary := PluginAuthSummary{}

	for _, result := range results {
		if result.Error != nil {
			summary.HasErrors = true
			summary.ErrorMessages = append(summary.ErrorMessages, result.PluginName+": "+result.Error.Error())
		} else if result.Credentials != nil && len(result.Credentials.Env) > 0 {
			summary.AuthenticatedPlugins = append(summary.AuthenticatedPlugins, result.PluginName)
		}
	}

	return summary
}

// FormatClipboardMessage formats a toast message for clipboard operations.
// count is the number of resources copied:
//   - count == 1: single resource, uses selectedItemName if provided
//   - count > 1: multiple resources, shows count
//   - count == 0: text copy (from details panel)
func FormatClipboardMessage(count int, selectedItemName string) string {
	switch {
	case count == 1:
		if selectedItemName != "" {
			return "Copied " + selectedItemName
		}
		return "Copied resource"
	case count > 1:
		return "Copied " + itoa(count) + " resources"
	default:
		return "Copied to clipboard"
	}
}

// itoa is a simple int-to-string without importing strconv.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	if i < 0 {
		return "-" + itoa(-i)
	}
	var digits []byte
	for i > 0 {
		digits = append([]byte{byte('0' + i%10)}, digits...)
		i /= 10
	}
	return string(digits)
}
