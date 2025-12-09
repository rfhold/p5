package pulumi

import (
	"context"
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/auto/events"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optdestroy"
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
	if len(opts.Excludes) > 0 {
		previewOpts = append(previewOpts, optpreview.Exclude(opts.Excludes))
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

// RunRefreshPreview runs a pulumi refresh preview (dry-run)
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
	if len(opts.Excludes) > 0 {
		refreshOpts = append(refreshOpts, optrefresh.Exclude(opts.Excludes))
	}

	// Use PreviewRefresh for dry-run (requires Pulumi CLI >= 3.105.0)
	_, err = stack.PreviewRefresh(ctx, refreshOpts...)
	if err != nil {
		eventCh <- PreviewEvent{Error: fmt.Errorf("refresh preview failed: %w", err)}
		return
	}

	eventCh <- PreviewEvent{Done: true}
}

// RunDestroyPreview runs a pulumi destroy preview
func RunDestroyPreview(ctx context.Context, workDir, stackName string, opts OperationOptions, eventCh chan<- PreviewEvent) {
	defer close(eventCh)

	stack, err := selectStack(ctx, workDir, stackName, opts.Env)
	if err != nil {
		eventCh <- PreviewEvent{Error: err}
		return
	}

	pulumiEvents := make(chan events.EngineEvent)

	go processPreviewEvents(pulumiEvents, eventCh)

	destroyOpts := []optdestroy.Option{optdestroy.EventStreams(pulumiEvents)}
	if len(opts.Targets) > 0 {
		destroyOpts = append(destroyOpts, optdestroy.Target(opts.Targets))
	}
	if len(opts.Excludes) > 0 {
		destroyOpts = append(destroyOpts, optdestroy.Exclude(opts.Excludes))
	}

	_, err = stack.PreviewDestroy(ctx, destroyOpts...)
	if err != nil {
		eventCh <- PreviewEvent{Error: fmt.Errorf("destroy preview failed: %w", err)}
		return
	}

	eventCh <- PreviewEvent{Done: true}
}
