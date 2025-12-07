package pulumi

import (
	"context"
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/auto/events"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optpreview"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optrefresh"
)

// RunPreview runs a pulumi preview and streams events to the channel
// If stackName is empty, it will use the currently selected stack
func RunPreview(ctx context.Context, workDir, stackName string, eventCh chan<- PreviewEvent) {
	RunUpPreview(ctx, workDir, stackName, OperationOptions{}, eventCh)
}

// RunUpPreview runs a pulumi up preview with options
func RunUpPreview(ctx context.Context, workDir, stackName string, opts OperationOptions, eventCh chan<- PreviewEvent) {
	defer close(eventCh)

	stack, err := selectStack(ctx, workDir, stackName, opts.Env)
	if err != nil {
		eventCh <- PreviewEvent{Error: err}
		return
	}

	// Create event channel for Pulumi
	pulumiEvents := make(chan events.EngineEvent)

	// Process Pulumi events and forward to our channel
	go processPreviewEvents(pulumiEvents, eventCh)

	// Build preview options
	previewOpts := []optpreview.Option{optpreview.EventStreams(pulumiEvents)}
	if len(opts.Targets) > 0 {
		previewOpts = append(previewOpts, optpreview.Target(opts.Targets))
	}
	if len(opts.Replaces) > 0 {
		previewOpts = append(previewOpts, optpreview.Replace(opts.Replaces))
	}

	// Run preview
	_, err = stack.Preview(ctx, previewOpts...)
	if err != nil {
		eventCh <- PreviewEvent{Error: fmt.Errorf("preview failed: %w", err)}
		return
	}

	// Signal completion
	eventCh <- PreviewEvent{Done: true}
}

// RunRefreshPreview runs a pulumi refresh preview
func RunRefreshPreview(ctx context.Context, workDir, stackName string, opts OperationOptions, eventCh chan<- PreviewEvent) {
	defer close(eventCh)

	stack, err := selectStack(ctx, workDir, stackName, opts.Env)
	if err != nil {
		eventCh <- PreviewEvent{Error: err}
		return
	}

	pulumiEvents := make(chan events.EngineEvent)

	go processPreviewEvents(pulumiEvents, eventCh)

	refreshOpts := []optrefresh.Option{optrefresh.EventStreams(pulumiEvents)}
	if len(opts.Targets) > 0 {
		refreshOpts = append(refreshOpts, optrefresh.Target(opts.Targets))
	}

	// Refresh with ExpectNoChanges to preview only
	_, err = stack.Refresh(ctx, refreshOpts...)
	if err != nil {
		eventCh <- PreviewEvent{Error: fmt.Errorf("refresh preview failed: %w", err)}
		return
	}

	eventCh <- PreviewEvent{Done: true}
}

// RunDestroyPreview runs a pulumi destroy preview
func RunDestroyPreview(ctx context.Context, workDir, stackName string, opts OperationOptions, eventCh chan<- PreviewEvent) {
	defer close(eventCh)

	resolvedStackName, err := resolveStackName(ctx, workDir, stackName, opts.Env)
	if err != nil {
		eventCh <- PreviewEvent{Error: err}
		return
	}

	// For a true destroy preview, we need to use a different approach
	// since the automation API doesn't have a destroy preview
	// For now, we'll just mark all resources as delete operations
	resources, err := GetStackResources(ctx, workDir, resolvedStackName, opts.Env)
	if err != nil {
		eventCh <- PreviewEvent{Error: fmt.Errorf("failed to get stack resources: %w", err)}
		return
	}

	for _, r := range resources {
		// Skip the stack itself
		if r.Type == "pulumi:pulumi:Stack" {
			continue
		}
		step := &PreviewStep{
			URN:    r.URN,
			Op:     OpDelete,
			Type:   r.Type,
			Name:   r.Name,
			Parent: r.Parent,
			// For delete, current state is the "old" state
			Old: &StepState{
				Inputs:  r.Inputs,
				Outputs: r.Outputs,
			},
		}
		eventCh <- PreviewEvent{Step: step}
	}

	eventCh <- PreviewEvent{Done: true}
}
