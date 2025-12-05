package pulumi

import (
	"context"
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/auto"
)

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

// selectStack handles the common stack selection boilerplate
// It resolves the stack name and creates workspace options with environment variables
func selectStack(ctx context.Context, workDir, stackName string, env map[string]string) (*auto.Stack, error) {
	resolvedStackName, err := resolveStackName(ctx, workDir, stackName)
	if err != nil {
		return nil, err
	}

	wsOpts := []auto.LocalWorkspaceOption{auto.WorkDir(workDir)}
	if len(env) > 0 {
		wsOpts = append(wsOpts, auto.EnvVars(env))
	}

	stack, err := auto.SelectStackLocalSource(ctx, resolvedStackName, workDir, wsOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to select stack: %w", err)
	}

	return &stack, nil
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
