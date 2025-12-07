package pulumi

import "context"

// DefaultStackReader wraps the existing free functions to implement StackReader.
type DefaultStackReader struct{}

// NewStackReader creates a new DefaultStackReader.
func NewStackReader() *DefaultStackReader {
	return &DefaultStackReader{}
}

// GetResources returns all resources in the stack.
func (d *DefaultStackReader) GetResources(ctx context.Context, workDir, stackName string, opts ReadOptions) ([]ResourceInfo, error) {
	return GetStackResources(ctx, workDir, stackName, opts.Env)
}

// GetHistory returns stack update history.
// pageSize is the number of entries per page, page is 1-indexed.
func (d *DefaultStackReader) GetHistory(ctx context.Context, workDir, stackName string, pageSize, page int, opts ReadOptions) ([]UpdateSummary, error) {
	return GetStackHistory(ctx, workDir, stackName, pageSize, page, opts.Env)
}

// GetStacks returns available stacks for a workspace.
func (d *DefaultStackReader) GetStacks(ctx context.Context, workDir string, opts ReadOptions) ([]StackInfo, error) {
	return ListStacks(ctx, workDir, opts.Env)
}

// SelectStack sets the specified stack as current.
func (d *DefaultStackReader) SelectStack(ctx context.Context, workDir, stackName string, opts ReadOptions) error {
	return SelectStack(ctx, workDir, stackName, opts.Env)
}

// Compile-time interface compliance check
var _ StackReader = (*DefaultStackReader)(nil)
