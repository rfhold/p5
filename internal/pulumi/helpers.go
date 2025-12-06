package pulumi

import (
	"github.com/pulumi/pulumi/sdk/v3/go/auto/events"
	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
)

// ExtractResourceName gets the resource name from a URN.
// URN format: urn:pulumi:stack::project::type::name
func ExtractResourceName(urn string) string {
	for i := len(urn) - 1; i >= 0; i-- {
		if i > 0 && urn[i-1:i+1] == "::" {
			return urn[i+1:]
		}
	}
	return urn
}

// extractParent gets the parent URN from step metadata.
func extractParent(meta apitype.StepEventMetadata) string {
	if meta.New != nil && meta.New.Parent != "" {
		return meta.New.Parent
	}
	if meta.Old != nil && meta.Old.Parent != "" {
		return meta.Old.Parent
	}
	return ""
}

// processPreviewEvents handles event processing for preview operations.
func processPreviewEvents(pulumiEvents <-chan events.EngineEvent, eventCh chan<- PreviewEvent) {
	for e := range pulumiEvents {
		if e.ResourcePreEvent != nil {
			meta := e.ResourcePreEvent.Metadata
			step := &PreviewStep{
				URN:    meta.URN,
				Op:     ResourceOp(meta.Op),
				Type:   meta.Type,
				Name:   ExtractResourceName(meta.URN),
				Parent: extractParent(meta),
			}
			if meta.New != nil {
				step.Inputs = meta.New.Inputs
				step.Outputs = meta.New.Outputs
			}
			if meta.Old != nil {
				step.Old = &StepState{
					Inputs:  meta.Old.Inputs,
					Outputs: meta.Old.Outputs,
				}
			}
			eventCh <- PreviewEvent{Step: step}
		}
		if e.ResOutputsEvent != nil {
			meta := e.ResOutputsEvent.Metadata
			step := &PreviewStep{
				URN:  meta.URN,
				Op:   ResourceOp(meta.Op),
				Type: meta.Type,
				Name: ExtractResourceName(meta.URN),
			}
			if meta.New != nil {
				step.Outputs = meta.New.Outputs
			}
			eventCh <- PreviewEvent{Step: step}
		}
	}
}

// processOperationEvents handles event processing for operations (up, refresh, destroy).
func processOperationEvents(pulumiEvents <-chan events.EngineEvent, eventCh chan<- OperationEvent, mode OperationEventMode) {
	for e := range pulumiEvents {
		if e.ResourcePreEvent != nil {
			meta := e.ResourcePreEvent.Metadata
			ev := OperationEvent{
				URN:    meta.URN,
				Op:     ResourceOp(meta.Op),
				Type:   meta.Type,
				Name:   ExtractResourceName(meta.URN),
				Parent: extractParent(meta),
				Status: StepRunning,
			}

			switch mode {
			case OperationModeDestroy:
				if meta.Old != nil {
					ev.Inputs = meta.Old.Inputs
					ev.Outputs = meta.Old.Outputs
				}
			default: // OperationModeStandard
				if meta.New != nil {
					ev.Inputs = meta.New.Inputs
				}
				if meta.Old != nil {
					ev.OldInputs = meta.Old.Inputs
					ev.OldOutputs = meta.Old.Outputs
				}
			}
			eventCh <- ev
		}
		if e.ResOutputsEvent != nil {
			meta := e.ResOutputsEvent.Metadata
			ev := OperationEvent{
				URN:    meta.URN,
				Op:     ResourceOp(meta.Op),
				Type:   meta.Type,
				Name:   ExtractResourceName(meta.URN),
				Parent: extractParent(meta),
				Status: StepSuccess,
			}
			if mode != OperationModeDestroy && meta.New != nil {
				ev.Outputs = meta.New.Outputs
			}
			eventCh <- ev
		}
		if e.DiagnosticEvent != nil && e.DiagnosticEvent.Severity == "error" {
			eventCh <- OperationEvent{
				Message: e.DiagnosticEvent.Message,
				Status:  StepFailed,
			}
		}
	}
}
