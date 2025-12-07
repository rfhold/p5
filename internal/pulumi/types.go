package pulumi

// ProjectInfo holds project and stack information
type ProjectInfo struct {
	ProgramName string
	Description string
	Runtime     string
	StackName   string
}

// ResourceOp represents a resource operation type
type ResourceOp string

const (
	OpCreate        ResourceOp = "create"
	OpUpdate        ResourceOp = "update"
	OpDelete        ResourceOp = "delete"
	OpSame          ResourceOp = "same"
	OpReplace       ResourceOp = "replace"
	OpCreateReplace ResourceOp = "create-replacement"
	OpDeleteReplace ResourceOp = "delete-replaced"
	OpRead          ResourceOp = "read"
	OpRefresh       ResourceOp = "refresh"
)

// PreviewStep represents a single resource operation in the preview
type PreviewStep struct {
	URN     string
	Op      ResourceOp
	Type    string
	Name    string
	Parent  string
	Inputs  map[string]interface{} // New state inputs (for create/update)
	Outputs map[string]interface{} // New state outputs (for create/update)
	Old     *StepState             // Old state (for update/delete)
}

// StepState holds resource state for old/new comparison
type StepState struct {
	Inputs  map[string]interface{}
	Outputs map[string]interface{}
}

// PreviewEvent is sent for each resource during preview
type PreviewEvent struct {
	Step  *PreviewStep
	Error error
	Done  bool
}

// PreviewSummary contains the final counts
type PreviewSummary struct {
	Create  int
	Update  int
	Delete  int
	Same    int
	Replace int
}

// OperationType for unified handling
type OperationType int

const (
	OperationUp OperationType = iota
	OperationRefresh
	OperationDestroy
)

func (o OperationType) String() string {
	switch o {
	case OperationUp:
		return "Up"
	case OperationRefresh:
		return "Refresh"
	case OperationDestroy:
		return "Destroy"
	default:
		return "Unknown"
	}
}

// OperationOptions for both preview and execution
type OperationOptions struct {
	Targets  []string          // --target URNs
	Replaces []string          // --replace URNs (up only)
	Env      map[string]string // Environment variables to set for the operation
}

// OperationEvent unified event type for execution
type OperationEvent struct {
	URN        string     // Resource being operated on
	Op         ResourceOp // Operation type
	Type       string     // Resource type
	Name       string     // Resource name
	Parent     string     // Parent URN for component hierarchy
	Status     StepStatus // pending/running/success/failed
	Error      error
	Done       bool
	Message    string                 // Diagnostic/log message
	Inputs     map[string]interface{} // Resource inputs (from ResourcePreEvent)
	Outputs    map[string]interface{} // Resource outputs (from ResOutputsEvent)
	OldInputs  map[string]interface{} // Previous inputs (for updates/deletes)
	OldOutputs map[string]interface{} // Previous outputs (for updates/deletes)
}

// StepStatus represents execution progress status
type StepStatus int

const (
	StepPending StepStatus = iota
	StepRunning
	StepSuccess
	StepFailed
)

// OperationEventMode determines how to extract state from operation events
type OperationEventMode int

const (
	// OperationModeStandard uses New.Inputs for inputs, Old for old state (up/refresh)
	OperationModeStandard OperationEventMode = iota
	// OperationModeDestroy uses Old state for inputs/outputs (destroy)
	OperationModeDestroy
)

// ResourceInfo for stack resources
type ResourceInfo struct {
	URN            string
	Type           string
	Name           string
	Provider       string
	Parent         string                 // Parent resource URN (empty for root resources)
	Inputs         map[string]interface{} // Resource inputs/args
	Outputs        map[string]interface{} // Resource outputs
	ProviderInputs map[string]interface{} // Configuration from the provider resource
}

// StackInfo holds information about a stack
type StackInfo struct {
	Name    string
	Current bool
}

// WorkspaceInfo holds information about a Pulumi workspace (project)
type WorkspaceInfo struct {
	Path    string // Absolute path to the directory containing Pulumi.yaml
	Name    string // Project name from Pulumi.yaml
	Current bool   // True if this is the currently selected workspace
}

// UpdateSummary represents a historical stack update
type UpdateSummary struct {
	Version         int
	Kind            string // "update", "preview", "refresh", "destroy"
	StartTime       string
	EndTime         string
	Message         string
	Result          string         // "succeeded", "failed", "in-progress"
	ResourceChanges map[string]int // e.g., {"create": 2, "update": 1}
	// Git/user info from Environment
	User      string // git.author or git.committer
	UserEmail string // git.author.email or git.committer.email
}

// CommandResult contains the result of a CLI command operation (import, state delete, etc.)
type CommandResult struct {
	Success bool
	Output  string
	Error   error
}

// ImportOptions for importing a resource
type ImportOptions struct {
	Env map[string]string // Environment variables to set for the operation
}

// StateDeleteOptions for deleting a resource from state
type StateDeleteOptions struct {
	Env map[string]string // Environment variables to set for the operation
}

// History pagination defaults
const (
	// DefaultHistoryPageSize is the default number of history entries to fetch
	DefaultHistoryPageSize = 50
	// DefaultHistoryPage is the default page number (1-indexed)
	DefaultHistoryPage = 1
)
