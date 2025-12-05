package pulumi

import "context"

// DefaultStackReader wraps the existing free functions to implement StackReader.
type DefaultStackReader struct{}

// NewStackReader creates a new DefaultStackReader.
func NewStackReader() *DefaultStackReader {
	return &DefaultStackReader{}
}

// GetResources returns all resources in the stack.
func (d *DefaultStackReader) GetResources(ctx context.Context, workDir, stackName string) ([]ResourceInfo, error) {
	return GetStackResources(ctx, workDir, stackName)
}

// GetHistory returns stack update history.
// pageSize is the number of entries per page, page is 1-indexed.
func (d *DefaultStackReader) GetHistory(ctx context.Context, workDir, stackName string, pageSize, page int) ([]UpdateSummary, error) {
	return GetStackHistory(ctx, workDir, stackName, pageSize, page)
}

// GetStacks returns available stacks for a workspace.
func (d *DefaultStackReader) GetStacks(ctx context.Context, workDir string) ([]StackInfo, error) {
	return ListStacks(ctx, workDir)
}

// SelectStack sets the specified stack as current.
func (d *DefaultStackReader) SelectStack(ctx context.Context, workDir, stackName string) error {
	return SelectStack(ctx, workDir, stackName)
}

// Compile-time interface compliance check
var _ StackReader = (*DefaultStackReader)(nil)
