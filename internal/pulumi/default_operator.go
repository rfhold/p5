package pulumi

import "context"

// DefaultStackOperator wraps the existing free functions to implement StackOperator.
// It owns the event channels and returns receive-only channels to callers.
type DefaultStackOperator struct{}

// NewStackOperator creates a new DefaultStackOperator.
func NewStackOperator() *DefaultStackOperator {
	return &DefaultStackOperator{}
}

// Preview runs a preview and returns a channel of events.
// The opType determines which preview variant to run (up, refresh, destroy).
func (d *DefaultStackOperator) Preview(ctx context.Context, workDir, stackName string, opType OperationType, opts OperationOptions) <-chan PreviewEvent {
	ch := make(chan PreviewEvent)
	go func() {
		switch opType {
		case OperationRefresh:
			RunRefreshPreview(ctx, workDir, stackName, opts, ch)
		case OperationDestroy:
			RunDestroyPreview(ctx, workDir, stackName, opts, ch)
		default:
			// Default to up preview for OperationUp
			RunUpPreview(ctx, workDir, stackName, opts, ch)
		}
	}()
	return ch
}

// Up executes pulumi up and returns a channel of events.
func (d *DefaultStackOperator) Up(ctx context.Context, workDir, stackName string, opts OperationOptions) <-chan OperationEvent {
	ch := make(chan OperationEvent)
	go func() {
		RunUp(ctx, workDir, stackName, opts, ch)
	}()
	return ch
}

// Refresh executes pulumi refresh and returns a channel of events.
func (d *DefaultStackOperator) Refresh(ctx context.Context, workDir, stackName string, opts OperationOptions) <-chan OperationEvent {
	ch := make(chan OperationEvent)
	go func() {
		RunRefresh(ctx, workDir, stackName, opts, ch)
	}()
	return ch
}

// Destroy executes pulumi destroy and returns a channel of events.
func (d *DefaultStackOperator) Destroy(ctx context.Context, workDir, stackName string, opts OperationOptions) <-chan OperationEvent {
	ch := make(chan OperationEvent)
	go func() {
		RunDestroy(ctx, workDir, stackName, opts, ch)
	}()
	return ch
}

// Compile-time interface compliance check
var _ StackOperator = (*DefaultStackOperator)(nil)
