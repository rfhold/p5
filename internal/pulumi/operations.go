package pulumi

import (
	"context"
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/auto/events"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optdestroy"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optrefresh"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
)

// RunUp executes pulumi up
func RunUp(ctx context.Context, workDir, stackName string, opts OperationOptions, eventCh chan<- OperationEvent) {
	defer close(eventCh)

	stack, err := selectStack(ctx, workDir, stackName, opts.Env)
	if err != nil {
		eventCh <- OperationEvent{Error: err, Done: true}
		return
	}

	pulumiEvents := make(chan events.EngineEvent)

	go processOperationEvents(pulumiEvents, eventCh, OperationModeStandard)

	upOpts := []optup.Option{optup.EventStreams(pulumiEvents)}
	if len(opts.Targets) > 0 {
		upOpts = append(upOpts, optup.Target(opts.Targets))
	}
	if len(opts.Replaces) > 0 {
		upOpts = append(upOpts, optup.Replace(opts.Replaces))
	}

	_, err = stack.Up(ctx, upOpts...)
	if err != nil {
		eventCh <- OperationEvent{Error: fmt.Errorf("up failed: %w", err), Done: true}
		return
	}

	eventCh <- OperationEvent{Done: true}
}

// RunRefresh executes pulumi refresh
func RunRefresh(ctx context.Context, workDir, stackName string, opts OperationOptions, eventCh chan<- OperationEvent) {
	defer close(eventCh)

	stack, err := selectStack(ctx, workDir, stackName, opts.Env)
	if err != nil {
		eventCh <- OperationEvent{Error: err, Done: true}
		return
	}

	pulumiEvents := make(chan events.EngineEvent)

	go processOperationEvents(pulumiEvents, eventCh, OperationModeStandard)

	refreshOpts := []optrefresh.Option{optrefresh.EventStreams(pulumiEvents)}
	if len(opts.Targets) > 0 {
		refreshOpts = append(refreshOpts, optrefresh.Target(opts.Targets))
	}

	_, err = stack.Refresh(ctx, refreshOpts...)
	if err != nil {
		eventCh <- OperationEvent{Error: fmt.Errorf("refresh failed: %w", err), Done: true}
		return
	}

	eventCh <- OperationEvent{Done: true}
}

// RunDestroy executes pulumi destroy
func RunDestroy(ctx context.Context, workDir, stackName string, opts OperationOptions, eventCh chan<- OperationEvent) {
	defer close(eventCh)

	stack, err := selectStack(ctx, workDir, stackName, opts.Env)
	if err != nil {
		eventCh <- OperationEvent{Error: err, Done: true}
		return
	}

	pulumiEvents := make(chan events.EngineEvent)

	go processOperationEvents(pulumiEvents, eventCh, OperationModeDestroy)

	destroyOpts := []optdestroy.Option{optdestroy.EventStreams(pulumiEvents)}
	if len(opts.Targets) > 0 {
		destroyOpts = append(destroyOpts, optdestroy.Target(opts.Targets))
	}

	_, err = stack.Destroy(ctx, destroyOpts...)
	if err != nil {
		eventCh <- OperationEvent{Error: fmt.Errorf("destroy failed: %w", err), Done: true}
		return
	}

	eventCh <- OperationEvent{Done: true}
}
