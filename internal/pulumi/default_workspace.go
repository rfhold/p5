package pulumi

import "context"

// DefaultWorkspaceReader wraps the existing free functions to implement WorkspaceReader.
type DefaultWorkspaceReader struct{}

// NewWorkspaceReader creates a new DefaultWorkspaceReader.
func NewWorkspaceReader() *DefaultWorkspaceReader {
	return &DefaultWorkspaceReader{}
}

// GetProjectInfo returns project metadata.
func (d *DefaultWorkspaceReader) GetProjectInfo(ctx context.Context, workDir, stackName string) (*ProjectInfo, error) {
	return FetchProjectInfo(ctx, workDir, stackName)
}

// FindWorkspaces finds Pulumi workspaces in a directory tree.
func (d *DefaultWorkspaceReader) FindWorkspaces(startDir, currentWorkDir string) ([]WorkspaceInfo, error) {
	return FindWorkspaces(startDir, currentWorkDir)
}

// IsWorkspace checks if the given directory is a valid Pulumi workspace.
func (d *DefaultWorkspaceReader) IsWorkspace(dir string) bool {
	return IsWorkspace(dir)
}

// GetWhoAmI returns the current backend user and URL.
func (d *DefaultWorkspaceReader) GetWhoAmI(ctx context.Context, workDir string) (*WhoAmIInfo, error) {
	return GetWhoAmI(ctx, workDir)
}

// ListStackFiles finds all Pulumi.<stack>.yaml files in the workspace.
func (d *DefaultWorkspaceReader) ListStackFiles(workDir string) ([]StackFileInfo, error) {
	return ListStackFiles(workDir)
}

// Compile-time interface compliance check
var _ WorkspaceReader = (*DefaultWorkspaceReader)(nil)
