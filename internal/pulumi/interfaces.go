package pulumi

import "context"

// StackOperator handles stack mutation operations (preview, up, refresh, destroy).
// Implementations own the event channels and return receive-only channels.
type StackOperator interface {
	// Preview runs a preview and returns a channel of events.
	// The opType determines which preview variant to run (up, refresh, destroy).
	Preview(ctx context.Context, workDir, stackName string, opType OperationType, opts OperationOptions) <-chan PreviewEvent

	// Up executes pulumi up and returns a channel of events.
	Up(ctx context.Context, workDir, stackName string, opts OperationOptions) <-chan OperationEvent

	// Refresh executes pulumi refresh and returns a channel of events.
	Refresh(ctx context.Context, workDir, stackName string, opts OperationOptions) <-chan OperationEvent

	// Destroy executes pulumi destroy and returns a channel of events.
	Destroy(ctx context.Context, workDir, stackName string, opts OperationOptions) <-chan OperationEvent
}

// StackReader handles read-only stack queries.
type StackReader interface {
	// GetResources returns all resources in the stack.
	GetResources(ctx context.Context, workDir, stackName string) ([]ResourceInfo, error)

	// GetHistory returns stack update history.
	// pageSize is the number of entries per page, page is 1-indexed.
	GetHistory(ctx context.Context, workDir, stackName string, pageSize, page int) ([]UpdateSummary, error)

	// GetStacks returns available stacks for a workspace.
	GetStacks(ctx context.Context, workDir string) ([]StackInfo, error)

	// SelectStack sets the specified stack as current.
	SelectStack(ctx context.Context, workDir, stackName string) error
}

// WorkspaceReader handles workspace-level queries.
type WorkspaceReader interface {
	// GetProjectInfo returns project metadata.
	GetProjectInfo(ctx context.Context, workDir, stackName string) (*ProjectInfo, error)

	// FindWorkspaces finds Pulumi workspaces in a directory tree.
	FindWorkspaces(startDir, currentWorkDir string) ([]WorkspaceInfo, error)

	// IsWorkspace checks if the given directory is a valid Pulumi workspace.
	IsWorkspace(dir string) bool

	// GetWhoAmI returns the current backend user and URL.
	GetWhoAmI(ctx context.Context, workDir string) (*WhoAmIInfo, error)

	// ListStackFiles finds all Pulumi.<stack>.yaml files in the workspace.
	ListStackFiles(workDir string) ([]StackFileInfo, error)
}

// StackInitializer handles stack creation.
type StackInitializer interface {
	// InitStack creates a new stack with the given configuration.
	InitStack(ctx context.Context, workDir, stackName string, opts InitStackOptions) error
}

// ResourceImporter handles resource import operations.
type ResourceImporter interface {
	// Import imports an external resource into the stack.
	// parentURN is optional - if provided, the resource will be imported as a child of this resource.
	Import(ctx context.Context, workDir, stackName, resourceType, resourceName, importID, parentURN string, opts ImportOptions) (*CommandResult, error)

	// StateDelete removes a resource from state without deleting the actual resource.
	StateDelete(ctx context.Context, workDir, stackName, urn string, opts StateDeleteOptions) (*CommandResult, error)
}
