package pulumi

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/events"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optdestroy"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optpreview"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optrefresh"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
)

// ProjectInfo holds project and stack information
type ProjectInfo struct {
	ProgramName string
	Description string
	Runtime     string
	StackName   string
}

// FetchProjectInfo loads project info from the specified directory
// If stackName is empty, it will use the currently selected stack
func FetchProjectInfo(ctx context.Context, workDir string, stackName string) (*ProjectInfo, error) {
	// Create a local workspace
	ws, err := auto.NewLocalWorkspace(ctx, auto.WorkDir(workDir))
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	// Get project settings
	project, err := ws.ProjectSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get project settings: %w", err)
	}

	// Get runtime as string
	runtime := ""
	if project.Runtime.Name() != "" {
		runtime = project.Runtime.Name()
	}

	// Try to get current stack if not specified
	resolvedStackName := stackName
	if resolvedStackName == "" {
		stacks, err := ws.ListStacks(ctx)
		if err == nil && len(stacks) > 0 {
			for _, s := range stacks {
				if s.Current {
					resolvedStackName = s.Name
					break
				}
			}
			if resolvedStackName == "" {
				resolvedStackName = stacks[0].Name
			}
		}
	}

	description := ""
	if project.Description != nil {
		description = *project.Description
	}

	return &ProjectInfo{
		ProgramName: project.Name.String(),
		Description: description,
		Runtime:     runtime,
		StackName:   resolvedStackName,
	}, nil
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

// ResourceInfo for stack resources
type ResourceInfo struct {
	URN      string
	Type     string
	Name     string
	Provider string
	Parent   string                 // Parent resource URN (empty for root resources)
	Inputs   map[string]interface{} // Resource inputs/args
	Outputs  map[string]interface{} // Resource outputs
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

// ListStacks returns all available stacks in the workspace
func ListStacks(ctx context.Context, workDir string) ([]StackInfo, error) {
	ws, err := auto.NewLocalWorkspace(ctx, auto.WorkDir(workDir))
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	stacks, err := ws.ListStacks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list stacks: %w", err)
	}

	result := make([]StackInfo, 0, len(stacks))
	for _, s := range stacks {
		result = append(result, StackInfo{
			Name:    s.Name,
			Current: s.Current,
		})
	}
	return result, nil
}

// SelectStack sets the specified stack as current
func SelectStack(ctx context.Context, workDir string, stackName string) error {
	_, err := auto.SelectStackLocalSource(ctx, stackName, workDir)
	if err != nil {
		return fmt.Errorf("failed to select stack: %w", err)
	}
	return nil
}

// resolveStackName resolves the stack name, using current stack if empty
func resolveStackName(ctx context.Context, workDir string, stackName string) (string, error) {
	if stackName != "" {
		return stackName, nil
	}

	ws, err := auto.NewLocalWorkspace(ctx, auto.WorkDir(workDir))
	if err != nil {
		return "", fmt.Errorf("failed to create workspace: %w", err)
	}
	stacks, err := ws.ListStacks(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to list stacks: %w", err)
	}
	for _, s := range stacks {
		if s.Current {
			return s.Name, nil
		}
	}
	if len(stacks) > 0 {
		return stacks[0].Name, nil
	}
	return "", fmt.Errorf("no stacks found")
}

// GetStackResources returns the currently deployed resources in the stack
func GetStackResources(ctx context.Context, workDir, stackName string) ([]ResourceInfo, error) {
	resolvedStackName, err := resolveStackName(ctx, workDir, stackName)
	if err != nil {
		return nil, err
	}

	stack, err := auto.SelectStackLocalSource(ctx, resolvedStackName, workDir)
	if err != nil {
		return nil, fmt.Errorf("failed to select stack: %w", err)
	}

	// Export the stack state
	state, err := stack.Export(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to export stack: %w", err)
	}

	// Parse the deployment to get resources with inputs and outputs
	var deployment struct {
		Resources []struct {
			URN      string                 `json:"urn"`
			Type     string                 `json:"type"`
			Provider string                 `json:"provider"`
			Parent   string                 `json:"parent"`
			Inputs   map[string]interface{} `json:"inputs"`
			Outputs  map[string]interface{} `json:"outputs"`
		} `json:"resources"`
	}

	if err := json.Unmarshal(state.Deployment, &deployment); err != nil {
		return nil, fmt.Errorf("failed to parse deployment: %w", err)
	}

	resources := make([]ResourceInfo, 0, len(deployment.Resources))
	for _, r := range deployment.Resources {
		resources = append(resources, ResourceInfo{
			URN:      r.URN,
			Type:     r.Type,
			Name:     extractResourceName(r.URN),
			Provider: r.Provider,
			Parent:   r.Parent,
			Inputs:   r.Inputs,
			Outputs:  r.Outputs,
		})
	}

	return resources, nil
}

// RunPreview runs a pulumi preview and streams events to the channel
// If stackName is empty, it will use the currently selected stack
func RunPreview(ctx context.Context, workDir string, stackName string, eventCh chan<- PreviewEvent) {
	RunUpPreview(ctx, workDir, stackName, OperationOptions{}, eventCh)
}

// RunUpPreview runs a pulumi up preview with options
func RunUpPreview(ctx context.Context, workDir string, stackName string, opts OperationOptions, eventCh chan<- PreviewEvent) {
	defer close(eventCh)

	resolvedStackName, err := resolveStackName(ctx, workDir, stackName)
	if err != nil {
		eventCh <- PreviewEvent{Error: err}
		return
	}

	// Build workspace options with env vars
	wsOpts := []auto.LocalWorkspaceOption{auto.WorkDir(workDir)}
	if len(opts.Env) > 0 {
		wsOpts = append(wsOpts, auto.EnvVars(opts.Env))
	}

	// Create/select stack from local source with env vars
	stack, err := auto.SelectStackLocalSource(ctx, resolvedStackName, workDir, wsOpts...)
	if err != nil {
		eventCh <- PreviewEvent{Error: fmt.Errorf("failed to select stack: %w", err)}
		return
	}

	// Create event channel for Pulumi
	pulumiEvents := make(chan events.EngineEvent)

	// Process Pulumi events and forward to our channel
	go func() {
		for e := range pulumiEvents {
			if e.ResourcePreEvent != nil {
				meta := e.ResourcePreEvent.Metadata
				step := &PreviewStep{
					URN:    meta.URN,
					Op:     ResourceOp(meta.Op),
					Type:   meta.Type,
					Name:   extractResourceName(meta.URN),
					Parent: extractParent(meta),
				}
				// Extract inputs/outputs from new state
				if meta.New != nil {
					step.Inputs = meta.New.Inputs
					step.Outputs = meta.New.Outputs
				}
				// Extract old state for updates/deletes
				if meta.Old != nil {
					step.Old = &StepState{
						Inputs:  meta.Old.Inputs,
						Outputs: meta.Old.Outputs,
					}
				}
				eventCh <- PreviewEvent{Step: step}
			}
			// Also handle ResOutputsEvent to capture computed outputs (especially for stack)
			if e.ResOutputsEvent != nil {
				meta := e.ResOutputsEvent.Metadata
				step := &PreviewStep{
					URN:  meta.URN,
					Op:   ResourceOp(meta.Op),
					Type: meta.Type,
					Name: extractResourceName(meta.URN),
				}
				if meta.New != nil {
					step.Outputs = meta.New.Outputs
				}
				eventCh <- PreviewEvent{Step: step}
			}
		}
	}()

	// Build preview options
	previewOpts := []optpreview.Option{optpreview.EventStreams(pulumiEvents)}
	if len(opts.Targets) > 0 {
		previewOpts = append(previewOpts, optpreview.Target(opts.Targets))
	}
	if len(opts.Replaces) > 0 {
		previewOpts = append(previewOpts, optpreview.Replace(opts.Replaces))
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

// RunRefreshPreview runs a pulumi refresh preview
func RunRefreshPreview(ctx context.Context, workDir string, stackName string, opts OperationOptions, eventCh chan<- PreviewEvent) {
	defer close(eventCh)

	resolvedStackName, err := resolveStackName(ctx, workDir, stackName)
	if err != nil {
		eventCh <- PreviewEvent{Error: err}
		return
	}

	// Build workspace options with env vars
	wsOpts := []auto.LocalWorkspaceOption{auto.WorkDir(workDir)}
	if len(opts.Env) > 0 {
		wsOpts = append(wsOpts, auto.EnvVars(opts.Env))
	}

	stack, err := auto.SelectStackLocalSource(ctx, resolvedStackName, workDir, wsOpts...)
	if err != nil {
		eventCh <- PreviewEvent{Error: fmt.Errorf("failed to select stack: %w", err)}
		return
	}

	pulumiEvents := make(chan events.EngineEvent)

	go func() {
		for e := range pulumiEvents {
			if e.ResourcePreEvent != nil {
				meta := e.ResourcePreEvent.Metadata
				step := &PreviewStep{
					URN:    meta.URN,
					Op:     ResourceOp(meta.Op),
					Type:   meta.Type,
					Name:   extractResourceName(meta.URN),
					Parent: extractParent(meta),
				}
				if meta.New != nil {
					step.Inputs = meta.New.Inputs
					step.Outputs = meta.New.Outputs
				}
				if meta.Old != nil {
					step.Old = &StepState{
						Inputs:  meta.Old.Inputs,
						Outputs: meta.Old.Outputs,
					}
				}
				eventCh <- PreviewEvent{Step: step}
			}
			// Also handle ResOutputsEvent to capture computed outputs (especially for stack)
			if e.ResOutputsEvent != nil {
				meta := e.ResOutputsEvent.Metadata
				step := &PreviewStep{
					URN:  meta.URN,
					Op:   ResourceOp(meta.Op),
					Type: meta.Type,
					Name: extractResourceName(meta.URN),
				}
				if meta.New != nil {
					step.Outputs = meta.New.Outputs
				}
				eventCh <- PreviewEvent{Step: step}
			}
		}
	}()

	refreshOpts := []optrefresh.Option{optrefresh.EventStreams(pulumiEvents)}
	if len(opts.Targets) > 0 {
		refreshOpts = append(refreshOpts, optrefresh.Target(opts.Targets))
	}

	// Refresh with ExpectNoChanges to preview only
	_, err = stack.Refresh(ctx, refreshOpts...)
	if err != nil {
		eventCh <- PreviewEvent{Error: fmt.Errorf("refresh preview failed: %w", err)}
		return
	}

	eventCh <- PreviewEvent{Done: true}
}

// RunDestroyPreview runs a pulumi destroy preview
func RunDestroyPreview(ctx context.Context, workDir string, stackName string, opts OperationOptions, eventCh chan<- PreviewEvent) {
	defer close(eventCh)

	resolvedStackName, err := resolveStackName(ctx, workDir, stackName)
	if err != nil {
		eventCh <- PreviewEvent{Error: err}
		return
	}

	// For a true destroy preview, we need to use a different approach
	// since the automation API doesn't have a destroy preview
	// For now, we'll just mark all resources as delete operations
	resources, err := GetStackResources(ctx, workDir, resolvedStackName)
	if err != nil {
		eventCh <- PreviewEvent{Error: fmt.Errorf("failed to get stack resources: %w", err)}
		return
	}

	for _, r := range resources {
		// Skip the stack itself
		if r.Type == "pulumi:pulumi:Stack" {
			continue
		}
		step := &PreviewStep{
			URN:    r.URN,
			Op:     OpDelete,
			Type:   r.Type,
			Name:   r.Name,
			Parent: r.Parent,
			// For delete, current state is the "old" state
			Old: &StepState{
				Inputs:  r.Inputs,
				Outputs: r.Outputs,
			},
		}
		eventCh <- PreviewEvent{Step: step}
	}

	eventCh <- PreviewEvent{Done: true}
}

// RunUp executes pulumi up
func RunUp(ctx context.Context, workDir string, stackName string, opts OperationOptions, eventCh chan<- OperationEvent) {
	defer close(eventCh)

	resolvedStackName, err := resolveStackName(ctx, workDir, stackName)
	if err != nil {
		eventCh <- OperationEvent{Error: err, Done: true}
		return
	}

	// Build workspace options with env vars
	wsOpts := []auto.LocalWorkspaceOption{auto.WorkDir(workDir)}
	if len(opts.Env) > 0 {
		wsOpts = append(wsOpts, auto.EnvVars(opts.Env))
	}

	stack, err := auto.SelectStackLocalSource(ctx, resolvedStackName, workDir, wsOpts...)
	if err != nil {
		eventCh <- OperationEvent{Error: fmt.Errorf("failed to select stack: %w", err), Done: true}
		return
	}

	pulumiEvents := make(chan events.EngineEvent)

	go func() {
		for e := range pulumiEvents {
			if e.ResourcePreEvent != nil {
				meta := e.ResourcePreEvent.Metadata
				ev := OperationEvent{
					URN:    meta.URN,
					Op:     ResourceOp(meta.Op),
					Type:   meta.Type,
					Name:   extractResourceName(meta.URN),
					Status: StepRunning,
				}
				if meta.New != nil {
					ev.Inputs = meta.New.Inputs
				}
				if meta.Old != nil {
					ev.OldInputs = meta.Old.Inputs
					ev.OldOutputs = meta.Old.Outputs
				}
				eventCh <- ev
			}
			if e.ResOutputsEvent != nil {
				meta := e.ResOutputsEvent.Metadata
				ev := OperationEvent{
					URN:    meta.URN,
					Op:     ResourceOp(meta.Op),
					Type:   meta.Type,
					Name:   extractResourceName(meta.URN),
					Status: StepSuccess,
				}
				if meta.New != nil {
					ev.Outputs = meta.New.Outputs
				}
				eventCh <- ev
			}
			if e.DiagnosticEvent != nil && e.DiagnosticEvent.Severity == "error" {
				eventCh <- OperationEvent{
					Message: e.DiagnosticEvent.Message,
					Status:  StepFailed,
				}
			}
		}
	}()

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
func RunRefresh(ctx context.Context, workDir string, stackName string, opts OperationOptions, eventCh chan<- OperationEvent) {
	defer close(eventCh)

	resolvedStackName, err := resolveStackName(ctx, workDir, stackName)
	if err != nil {
		eventCh <- OperationEvent{Error: err, Done: true}
		return
	}

	// Build workspace options with env vars
	wsOpts := []auto.LocalWorkspaceOption{auto.WorkDir(workDir)}
	if len(opts.Env) > 0 {
		wsOpts = append(wsOpts, auto.EnvVars(opts.Env))
	}

	stack, err := auto.SelectStackLocalSource(ctx, resolvedStackName, workDir, wsOpts...)
	if err != nil {
		eventCh <- OperationEvent{Error: fmt.Errorf("failed to select stack: %w", err), Done: true}
		return
	}

	pulumiEvents := make(chan events.EngineEvent)

	go func() {
		for e := range pulumiEvents {
			if e.ResourcePreEvent != nil {
				meta := e.ResourcePreEvent.Metadata
				ev := OperationEvent{
					URN:    meta.URN,
					Op:     ResourceOp(meta.Op),
					Type:   meta.Type,
					Name:   extractResourceName(meta.URN),
					Status: StepRunning,
				}
				if meta.New != nil {
					ev.Inputs = meta.New.Inputs
				}
				if meta.Old != nil {
					ev.OldInputs = meta.Old.Inputs
					ev.OldOutputs = meta.Old.Outputs
				}
				eventCh <- ev
			}
			if e.ResOutputsEvent != nil {
				meta := e.ResOutputsEvent.Metadata
				ev := OperationEvent{
					URN:    meta.URN,
					Op:     ResourceOp(meta.Op),
					Type:   meta.Type,
					Name:   extractResourceName(meta.URN),
					Status: StepSuccess,
				}
				if meta.New != nil {
					ev.Outputs = meta.New.Outputs
				}
				eventCh <- ev
			}
			if e.DiagnosticEvent != nil && e.DiagnosticEvent.Severity == "error" {
				eventCh <- OperationEvent{
					Message: e.DiagnosticEvent.Message,
					Status:  StepFailed,
				}
			}
		}
	}()

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
func RunDestroy(ctx context.Context, workDir string, stackName string, opts OperationOptions, eventCh chan<- OperationEvent) {
	defer close(eventCh)

	resolvedStackName, err := resolveStackName(ctx, workDir, stackName)
	if err != nil {
		eventCh <- OperationEvent{Error: err, Done: true}
		return
	}

	// Build workspace options with env vars
	wsOpts := []auto.LocalWorkspaceOption{auto.WorkDir(workDir)}
	if len(opts.Env) > 0 {
		wsOpts = append(wsOpts, auto.EnvVars(opts.Env))
	}

	stack, err := auto.SelectStackLocalSource(ctx, resolvedStackName, workDir, wsOpts...)
	if err != nil {
		eventCh <- OperationEvent{Error: fmt.Errorf("failed to select stack: %w", err), Done: true}
		return
	}

	pulumiEvents := make(chan events.EngineEvent)

	go func() {
		for e := range pulumiEvents {
			if e.ResourcePreEvent != nil {
				meta := e.ResourcePreEvent.Metadata
				ev := OperationEvent{
					URN:    meta.URN,
					Op:     ResourceOp(meta.Op),
					Type:   meta.Type,
					Name:   extractResourceName(meta.URN),
					Status: StepRunning,
				}
				// For destroy, Old has the current state being deleted
				if meta.Old != nil {
					ev.Inputs = meta.Old.Inputs
					ev.Outputs = meta.Old.Outputs
				}
				eventCh <- ev
			}
			if e.ResOutputsEvent != nil {
				meta := e.ResOutputsEvent.Metadata
				ev := OperationEvent{
					URN:    meta.URN,
					Op:     ResourceOp(meta.Op),
					Type:   meta.Type,
					Name:   extractResourceName(meta.URN),
					Status: StepSuccess,
				}
				eventCh <- ev
			}
			if e.DiagnosticEvent != nil && e.DiagnosticEvent.Severity == "error" {
				eventCh <- OperationEvent{
					Message: e.DiagnosticEvent.Message,
					Status:  StepFailed,
				}
			}
		}
	}()

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

// extractResourceName gets the resource name from a URN
// URN format: urn:pulumi:stack::project::type::name
func extractResourceName(urn string) string {
	// Find the last :: and return everything after it
	for i := len(urn) - 1; i >= 0; i-- {
		if i > 0 && urn[i-1:i+1] == "::" {
			return urn[i+1:]
		}
	}
	return urn
}

// extractParent gets the parent URN from step metadata
// Prefers New state (for creates) but falls back to Old state (for updates/deletes)
func extractParent(meta apitype.StepEventMetadata) string {
	if meta.New != nil && meta.New.Parent != "" {
		return meta.New.Parent
	}
	if meta.Old != nil && meta.Old.Parent != "" {
		return meta.Old.Parent
	}
	return ""
}

// IsWorkspace checks if the given directory is a valid Pulumi workspace
// (contains Pulumi.yaml or Pulumi.yml)
func IsWorkspace(dir string) bool {
	yamlPath := filepath.Join(dir, "Pulumi.yaml")
	if _, err := os.Stat(yamlPath); err == nil {
		return true
	}
	ymlPath := filepath.Join(dir, "Pulumi.yml")
	if _, err := os.Stat(ymlPath); err == nil {
		return true
	}
	return false
}

// FindWorkspaces searches for Pulumi.yaml files starting from the given directory
// and returns a list of workspace paths. It searches recursively down the directory tree.
func FindWorkspaces(startDir string, currentWorkDir string) ([]WorkspaceInfo, error) {
	var workspaces []WorkspaceInfo

	// Resolve absolute paths for comparison
	absStart, err := filepath.Abs(startDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve start directory: %w", err)
	}

	absCurrent := ""
	if currentWorkDir != "" {
		absCurrent, err = filepath.Abs(currentWorkDir)
		if err != nil {
			absCurrent = ""
		}
	}

	err = filepath.Walk(absStart, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip directories we can't access
			if info != nil && info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip hidden directories and common non-project directories
		if info.IsDir() {
			name := info.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == "__pycache__" {
				return filepath.SkipDir
			}
		}

		// Check for Pulumi.yaml or Pulumi.yml
		if !info.IsDir() && (info.Name() == "Pulumi.yaml" || info.Name() == "Pulumi.yml") {
			dir := filepath.Dir(path)

			// Try to get project name from the file
			projectName := filepath.Base(dir)
			if name, err := getProjectName(path); err == nil && name != "" {
				projectName = name
			}

			workspaces = append(workspaces, WorkspaceInfo{
				Path:    dir,
				Name:    projectName,
				Current: dir == absCurrent,
			})
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to search for workspaces: %w", err)
	}

	return workspaces, nil
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

// GetStackHistory returns the history of updates for a stack
func GetStackHistory(ctx context.Context, workDir, stackName string, pageSize, page int) ([]UpdateSummary, error) {
	resolvedStackName, err := resolveStackName(ctx, workDir, stackName)
	if err != nil {
		return nil, err
	}

	stack, err := auto.SelectStackLocalSource(ctx, resolvedStackName, workDir)
	if err != nil {
		return nil, fmt.Errorf("failed to select stack: %w", err)
	}

	history, err := stack.History(ctx, pageSize, page)
	if err != nil {
		return nil, fmt.Errorf("failed to get stack history: %w", err)
	}

	result := make([]UpdateSummary, 0, len(history))
	for _, h := range history {
		summary := UpdateSummary{
			Version:   h.Version,
			Kind:      h.Kind,
			StartTime: h.StartTime,
			Message:   h.Message,
			Result:    h.Result,
		}
		if h.EndTime != nil {
			summary.EndTime = *h.EndTime
		}
		if h.ResourceChanges != nil {
			summary.ResourceChanges = make(map[string]int)
			for k, v := range *h.ResourceChanges {
				summary.ResourceChanges[k] = v
			}
		}
		// Extract user info from environment
		if h.Environment != nil {
			if author, ok := h.Environment["git.author"]; ok && author != "" {
				summary.User = author
			} else if committer, ok := h.Environment["git.committer"]; ok && committer != "" {
				summary.User = committer
			}
			if email, ok := h.Environment["git.author.email"]; ok && email != "" {
				summary.UserEmail = email
			} else if email, ok := h.Environment["git.committer.email"]; ok && email != "" {
				summary.UserEmail = email
			}
		}
		result = append(result, summary)
	}

	return result, nil
}

// getProjectName reads the project name from a Pulumi.yaml file
func getProjectName(pulumiYamlPath string) (string, error) {
	data, err := os.ReadFile(pulumiYamlPath)
	if err != nil {
		return "", err
	}

	// Simple YAML parsing - just look for "name:" line
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "name:") {
			name := strings.TrimPrefix(line, "name:")
			name = strings.TrimSpace(name)
			// Remove quotes if present
			name = strings.Trim(name, "\"'")
			return name, nil
		}
	}

	return "", nil
}
