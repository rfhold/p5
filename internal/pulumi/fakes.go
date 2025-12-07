package pulumi

import (
	"context"
)

// FakeStackOperator implements StackOperator for testing.
// Configure behavior via function fields, and track calls via the Calls struct.
type FakeStackOperator struct {
	// PreviewFunc optionally configures Preview behavior.
	// If nil, returns an empty closed channel.
	PreviewFunc func(ctx context.Context, workDir, stackName string, opType OperationType, opts OperationOptions) <-chan PreviewEvent

	// UpFunc optionally configures Up behavior.
	UpFunc func(ctx context.Context, workDir, stackName string, opts OperationOptions) <-chan OperationEvent

	// RefreshFunc optionally configures Refresh behavior.
	RefreshFunc func(ctx context.Context, workDir, stackName string, opts OperationOptions) <-chan OperationEvent

	// DestroyFunc optionally configures Destroy behavior.
	DestroyFunc func(ctx context.Context, workDir, stackName string, opts OperationOptions) <-chan OperationEvent

	// Calls tracks all method invocations for assertions.
	Calls struct {
		Preview []PreviewCall
		Up      []OperationCall
		Refresh []OperationCall
		Destroy []OperationCall
	}
}

// PreviewCall records a call to Preview.
type PreviewCall struct {
	WorkDir   string
	StackName string
	OpType    OperationType
	Opts      OperationOptions
}

// OperationCall records a call to Up, Refresh, or Destroy.
type OperationCall struct {
	WorkDir   string
	StackName string
	Opts      OperationOptions
}

func (f *FakeStackOperator) Preview(ctx context.Context, workDir, stackName string, opType OperationType, opts OperationOptions) <-chan PreviewEvent {
	f.Calls.Preview = append(f.Calls.Preview, PreviewCall{workDir, stackName, opType, opts})
	if f.PreviewFunc != nil {
		return f.PreviewFunc(ctx, workDir, stackName, opType, opts)
	}
	ch := make(chan PreviewEvent)
	close(ch)
	return ch
}

func (f *FakeStackOperator) Up(ctx context.Context, workDir, stackName string, opts OperationOptions) <-chan OperationEvent {
	f.Calls.Up = append(f.Calls.Up, OperationCall{workDir, stackName, opts})
	if f.UpFunc != nil {
		return f.UpFunc(ctx, workDir, stackName, opts)
	}
	ch := make(chan OperationEvent)
	close(ch)
	return ch
}

func (f *FakeStackOperator) Refresh(ctx context.Context, workDir, stackName string, opts OperationOptions) <-chan OperationEvent {
	f.Calls.Refresh = append(f.Calls.Refresh, OperationCall{workDir, stackName, opts})
	if f.RefreshFunc != nil {
		return f.RefreshFunc(ctx, workDir, stackName, opts)
	}
	ch := make(chan OperationEvent)
	close(ch)
	return ch
}

func (f *FakeStackOperator) Destroy(ctx context.Context, workDir, stackName string, opts OperationOptions) <-chan OperationEvent {
	f.Calls.Destroy = append(f.Calls.Destroy, OperationCall{workDir, stackName, opts})
	if f.DestroyFunc != nil {
		return f.DestroyFunc(ctx, workDir, stackName, opts)
	}
	ch := make(chan OperationEvent)
	close(ch)
	return ch
}

// WithPreviewEvents is a helper that configures PreviewFunc to return the given events.
func (f *FakeStackOperator) WithPreviewEvents(events ...PreviewEvent) *FakeStackOperator {
	f.PreviewFunc = func(ctx context.Context, workDir, stackName string, opType OperationType, opts OperationOptions) <-chan PreviewEvent {
		ch := make(chan PreviewEvent, len(events))
		for _, e := range events {
			ch <- e
		}
		close(ch)
		return ch
	}
	return f
}

// WithOperationEvents is a helper that configures all operation funcs to return the given events.
func (f *FakeStackOperator) WithOperationEvents(events ...OperationEvent) *FakeStackOperator {
	makeFunc := func() func(ctx context.Context, workDir, stackName string, opts OperationOptions) <-chan OperationEvent {
		return func(ctx context.Context, workDir, stackName string, opts OperationOptions) <-chan OperationEvent {
			ch := make(chan OperationEvent, len(events))
			for i := range events {
				ch <- events[i]
			}
			close(ch)
			return ch
		}
	}
	f.UpFunc = makeFunc()
	f.RefreshFunc = makeFunc()
	f.DestroyFunc = makeFunc()
	return f
}

// FakeStackReader implements StackReader for testing.
type FakeStackReader struct {
	// GetResourcesFunc optionally configures GetResources behavior.
	GetResourcesFunc func(ctx context.Context, workDir, stackName string, opts ReadOptions) ([]ResourceInfo, error)

	// GetHistoryFunc optionally configures GetHistory behavior.
	GetHistoryFunc func(ctx context.Context, workDir, stackName string, pageSize, page int, opts ReadOptions) ([]UpdateSummary, error)

	// GetStacksFunc optionally configures GetStacks behavior.
	GetStacksFunc func(ctx context.Context, workDir string, opts ReadOptions) ([]StackInfo, error)

	// SelectStackFunc optionally configures SelectStack behavior.
	SelectStackFunc func(ctx context.Context, workDir, stackName string, opts ReadOptions) error

	// Default return values (used when funcs are nil)
	Resources []ResourceInfo
	History   []UpdateSummary
	Stacks    []StackInfo

	// Calls tracks all method invocations.
	Calls struct {
		GetResources []GetResourcesCall
		GetHistory   []GetHistoryCall
		GetStacks    []GetStacksCall
		SelectStack  []SelectStackCall
	}
}

type GetResourcesCall struct {
	WorkDir   string
	StackName string
	Opts      ReadOptions
}

type GetHistoryCall struct {
	WorkDir   string
	StackName string
	PageSize  int
	Page      int
	Opts      ReadOptions
}

type GetStacksCall struct {
	WorkDir string
	Opts    ReadOptions
}

type SelectStackCall struct {
	WorkDir   string
	StackName string
	Opts      ReadOptions
}

func (f *FakeStackReader) GetResources(ctx context.Context, workDir, stackName string, opts ReadOptions) ([]ResourceInfo, error) {
	f.Calls.GetResources = append(f.Calls.GetResources, GetResourcesCall{workDir, stackName, opts})
	if f.GetResourcesFunc != nil {
		return f.GetResourcesFunc(ctx, workDir, stackName, opts)
	}
	return f.Resources, nil
}

func (f *FakeStackReader) GetHistory(ctx context.Context, workDir, stackName string, pageSize, page int, opts ReadOptions) ([]UpdateSummary, error) {
	f.Calls.GetHistory = append(f.Calls.GetHistory, GetHistoryCall{workDir, stackName, pageSize, page, opts})
	if f.GetHistoryFunc != nil {
		return f.GetHistoryFunc(ctx, workDir, stackName, pageSize, page, opts)
	}
	return f.History, nil
}

func (f *FakeStackReader) GetStacks(ctx context.Context, workDir string, opts ReadOptions) ([]StackInfo, error) {
	f.Calls.GetStacks = append(f.Calls.GetStacks, GetStacksCall{workDir, opts})
	if f.GetStacksFunc != nil {
		return f.GetStacksFunc(ctx, workDir, opts)
	}
	return f.Stacks, nil
}

func (f *FakeStackReader) SelectStack(ctx context.Context, workDir, stackName string, opts ReadOptions) error {
	f.Calls.SelectStack = append(f.Calls.SelectStack, SelectStackCall{workDir, stackName, opts})
	if f.SelectStackFunc != nil {
		return f.SelectStackFunc(ctx, workDir, stackName, opts)
	}
	return nil
}

// FakeWorkspaceReader implements WorkspaceReader for testing.
type FakeWorkspaceReader struct {
	// GetProjectInfoFunc optionally configures GetProjectInfo behavior.
	GetProjectInfoFunc func(ctx context.Context, workDir, stackName string, opts ReadOptions) (*ProjectInfo, error)

	// FindWorkspacesFunc optionally configures FindWorkspaces behavior.
	FindWorkspacesFunc func(startDir, currentWorkDir string) ([]WorkspaceInfo, error)

	// IsWorkspaceFunc optionally configures IsWorkspace behavior.
	IsWorkspaceFunc func(dir string) bool

	// GetWhoAmIFunc optionally configures GetWhoAmI behavior.
	GetWhoAmIFunc func(ctx context.Context, workDir string, opts ReadOptions) (*WhoAmIInfo, error)

	// ListStackFilesFunc optionally configures ListStackFiles behavior.
	ListStackFilesFunc func(workDir string) ([]StackFileInfo, error)

	// Default return values
	ProjectInfo  *ProjectInfo
	Workspaces   []WorkspaceInfo
	ValidWorkDir bool // Default for IsWorkspace
	WhoAmI       *WhoAmIInfo
	StackFiles   []StackFileInfo

	// Calls tracks all method invocations.
	Calls struct {
		GetProjectInfo []GetProjectInfoCall
		FindWorkspaces []FindWorkspacesCall
		IsWorkspace    []string
		GetWhoAmI      []GetWhoAmICall
		ListStackFiles []string
	}
}

type GetProjectInfoCall struct {
	WorkDir   string
	StackName string
	Opts      ReadOptions
}

type FindWorkspacesCall struct {
	StartDir       string
	CurrentWorkDir string
}

type GetWhoAmICall struct {
	WorkDir string
	Opts    ReadOptions
}

func (f *FakeWorkspaceReader) GetProjectInfo(ctx context.Context, workDir, stackName string, opts ReadOptions) (*ProjectInfo, error) {
	f.Calls.GetProjectInfo = append(f.Calls.GetProjectInfo, GetProjectInfoCall{workDir, stackName, opts})
	if f.GetProjectInfoFunc != nil {
		return f.GetProjectInfoFunc(ctx, workDir, stackName, opts)
	}
	return f.ProjectInfo, nil
}

func (f *FakeWorkspaceReader) FindWorkspaces(startDir, currentWorkDir string) ([]WorkspaceInfo, error) {
	f.Calls.FindWorkspaces = append(f.Calls.FindWorkspaces, FindWorkspacesCall{startDir, currentWorkDir})
	if f.FindWorkspacesFunc != nil {
		return f.FindWorkspacesFunc(startDir, currentWorkDir)
	}
	return f.Workspaces, nil
}

func (f *FakeWorkspaceReader) IsWorkspace(dir string) bool {
	f.Calls.IsWorkspace = append(f.Calls.IsWorkspace, dir)
	if f.IsWorkspaceFunc != nil {
		return f.IsWorkspaceFunc(dir)
	}
	return f.ValidWorkDir
}

func (f *FakeWorkspaceReader) GetWhoAmI(ctx context.Context, workDir string, opts ReadOptions) (*WhoAmIInfo, error) {
	f.Calls.GetWhoAmI = append(f.Calls.GetWhoAmI, GetWhoAmICall{workDir, opts})
	if f.GetWhoAmIFunc != nil {
		return f.GetWhoAmIFunc(ctx, workDir, opts)
	}
	return f.WhoAmI, nil
}

func (f *FakeWorkspaceReader) ListStackFiles(workDir string) ([]StackFileInfo, error) {
	f.Calls.ListStackFiles = append(f.Calls.ListStackFiles, workDir)
	if f.ListStackFilesFunc != nil {
		return f.ListStackFilesFunc(workDir)
	}
	return f.StackFiles, nil
}

// FakeStackInitializer implements StackInitializer for testing.
type FakeStackInitializer struct {
	// InitStackFunc optionally configures InitStack behavior.
	InitStackFunc func(ctx context.Context, workDir, stackName string, opts InitStackOptions) error

	// Error is the default error to return (nil for success).
	Error error

	// Calls tracks all method invocations.
	Calls struct {
		InitStack []InitStackCall
	}
}

type InitStackCall struct {
	WorkDir   string
	StackName string
	Opts      InitStackOptions
}

func (f *FakeStackInitializer) InitStack(ctx context.Context, workDir, stackName string, opts InitStackOptions) error {
	f.Calls.InitStack = append(f.Calls.InitStack, InitStackCall{workDir, stackName, opts})
	if f.InitStackFunc != nil {
		return f.InitStackFunc(ctx, workDir, stackName, opts)
	}
	return f.Error
}

// FakeResourceImporter implements ResourceImporter for testing.
type FakeResourceImporter struct {
	// ImportFunc optionally configures Import behavior.
	ImportFunc func(ctx context.Context, workDir, stackName, resourceType, resourceName, importID, parentURN string, opts ImportOptions) (*CommandResult, error)

	// StateDeleteFunc optionally configures StateDelete behavior.
	StateDeleteFunc func(ctx context.Context, workDir, stackName, urn string, opts StateDeleteOptions) (*CommandResult, error)

	// Default return values
	ImportResult      *CommandResult
	StateDeleteResult *CommandResult

	// Calls tracks all method invocations.
	Calls struct {
		Import      []ImportCall
		StateDelete []StateDeleteCall
	}
}

type ImportCall struct {
	WorkDir      string
	StackName    string
	ResourceType string
	ResourceName string
	ImportID     string
	ParentURN    string
	Opts         ImportOptions
}

type StateDeleteCall struct {
	WorkDir   string
	StackName string
	URN       string
	Opts      StateDeleteOptions
}

func (f *FakeResourceImporter) Import(ctx context.Context, workDir, stackName, resourceType, resourceName, importID, parentURN string, opts ImportOptions) (*CommandResult, error) {
	f.Calls.Import = append(f.Calls.Import, ImportCall{workDir, stackName, resourceType, resourceName, importID, parentURN, opts})
	if f.ImportFunc != nil {
		return f.ImportFunc(ctx, workDir, stackName, resourceType, resourceName, importID, parentURN, opts)
	}
	if f.ImportResult != nil {
		return f.ImportResult, nil
	}
	return &CommandResult{Success: true}, nil
}

func (f *FakeResourceImporter) StateDelete(ctx context.Context, workDir, stackName, urn string, opts StateDeleteOptions) (*CommandResult, error) {
	f.Calls.StateDelete = append(f.Calls.StateDelete, StateDeleteCall{workDir, stackName, urn, opts})
	if f.StateDeleteFunc != nil {
		return f.StateDeleteFunc(ctx, workDir, stackName, urn, opts)
	}
	if f.StateDeleteResult != nil {
		return f.StateDeleteResult, nil
	}
	return &CommandResult{Success: true}, nil
}

// Compile-time interface compliance checks
var (
	_ StackOperator    = (*FakeStackOperator)(nil)
	_ StackReader      = (*FakeStackReader)(nil)
	_ WorkspaceReader  = (*FakeWorkspaceReader)(nil)
	_ StackInitializer = (*FakeStackInitializer)(nil)
	_ ResourceImporter = (*FakeResourceImporter)(nil)
)
