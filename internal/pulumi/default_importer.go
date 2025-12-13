package pulumi

import "context"

// DefaultResourceImporter wraps the existing free functions to implement ResourceImporter.
type DefaultResourceImporter struct{}

// NewResourceImporter creates a new DefaultResourceImporter.
func NewResourceImporter() *DefaultResourceImporter {
	return &DefaultResourceImporter{}
}

// Import imports an external resource into the stack.
// parentURN is optional - if provided, the resource will be imported as a child of this resource.
func (d *DefaultResourceImporter) Import(ctx context.Context, workDir, stackName, resourceType, resourceName, importID, parentURN string, opts ImportOptions) (*CommandResult, error) {
	return ImportResource(ctx, workDir, stackName, resourceType, resourceName, importID, parentURN, opts)
}

// StateDelete removes a resource from state without deleting the actual resource.
func (d *DefaultResourceImporter) StateDelete(ctx context.Context, workDir, stackName, urn string, opts StateDeleteOptions) (*CommandResult, error) {
	return DeleteFromState(ctx, workDir, stackName, urn, opts)
}

// Protect marks a resource as protected, preventing it from being destroyed.
func (d *DefaultResourceImporter) Protect(ctx context.Context, workDir, stackName, urn string, opts StateProtectOptions) (*CommandResult, error) {
	return ProtectResource(ctx, workDir, stackName, urn, opts)
}

// Unprotect removes the protected flag from a resource, allowing it to be destroyed.
func (d *DefaultResourceImporter) Unprotect(ctx context.Context, workDir, stackName, urn string, opts StateProtectOptions) (*CommandResult, error) {
	return UnprotectResource(ctx, workDir, stackName, urn, opts)
}

// Compile-time interface compliance check
var _ ResourceImporter = (*DefaultResourceImporter)(nil)

// DefaultStackInitializer wraps the existing InitStack function to implement StackInitializer.
type DefaultStackInitializer struct{}

// NewStackInitializer creates a new DefaultStackInitializer.
func NewStackInitializer() *DefaultStackInitializer {
	return &DefaultStackInitializer{}
}

// InitStack creates a new stack with the given configuration.
func (d *DefaultStackInitializer) InitStack(ctx context.Context, workDir, stackName string, opts InitStackOptions) error {
	return InitStack(ctx, workDir, stackName, opts)
}

// Compile-time interface compliance check
var _ StackInitializer = (*DefaultStackInitializer)(nil)
