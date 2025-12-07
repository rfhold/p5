package pulumi

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"gopkg.in/yaml.v3"
)

var ErrNoStacksFound = errors.New("no stacks found")

// FetchProjectInfo loads project info from the specified directory
// If stackName is empty, it will use the currently selected stack
func FetchProjectInfo(ctx context.Context, workDir, stackName string, env map[string]string) (*ProjectInfo, error) {
	// Create a local workspace
	wsOpts := []auto.LocalWorkspaceOption{auto.WorkDir(workDir)}
	if len(env) > 0 {
		wsOpts = append(wsOpts, auto.EnvVars(env))
	}
	ws, err := auto.NewLocalWorkspace(ctx, wsOpts...)
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
		resolvedStackName = findCurrentStack(ctx, ws)
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
	resolvedStackName, err := resolveStackName(ctx, workDir, stackName, env)
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
func ListStacks(ctx context.Context, workDir string, env map[string]string) ([]StackInfo, error) {
	wsOpts := []auto.LocalWorkspaceOption{auto.WorkDir(workDir)}
	if len(env) > 0 {
		wsOpts = append(wsOpts, auto.EnvVars(env))
	}
	ws, err := auto.NewLocalWorkspace(ctx, wsOpts...)
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
func SelectStack(ctx context.Context, workDir, stackName string, env map[string]string) error {
	wsOpts := []auto.LocalWorkspaceOption{auto.WorkDir(workDir)}
	if len(env) > 0 {
		wsOpts = append(wsOpts, auto.EnvVars(env))
	}
	_, err := auto.SelectStackLocalSource(ctx, stackName, workDir, wsOpts...)
	if err != nil {
		return fmt.Errorf("failed to select stack: %w", err)
	}
	return nil
}

// resolveStackName resolves the stack name, using current stack if empty
func resolveStackName(ctx context.Context, workDir, stackName string, env map[string]string) (string, error) {
	if stackName != "" {
		return stackName, nil
	}

	wsOpts := []auto.LocalWorkspaceOption{auto.WorkDir(workDir)}
	if len(env) > 0 {
		wsOpts = append(wsOpts, auto.EnvVars(env))
	}
	ws, err := auto.NewLocalWorkspace(ctx, wsOpts...)
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
	return "", ErrNoStacksFound
}

// WhoAmIInfo contains backend connection information
type WhoAmIInfo struct {
	User string
	URL  string
}

// GetWhoAmI returns the current backend user and URL
func GetWhoAmI(ctx context.Context, workDir string, env map[string]string) (*WhoAmIInfo, error) {
	wsOpts := []auto.LocalWorkspaceOption{auto.WorkDir(workDir)}
	if len(env) > 0 {
		wsOpts = append(wsOpts, auto.EnvVars(env))
	}
	ws, err := auto.NewLocalWorkspace(ctx, wsOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	whoami, err := ws.WhoAmIDetails(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get whoami: %w", err)
	}

	return &WhoAmIInfo{
		User: whoami.User,
		URL:  whoami.URL,
	}, nil
}

// StackFileInfo describes a stack config file
type StackFileInfo struct {
	Name            string
	FilePath        string
	SecretsProvider string
	HasEncryption   bool
}

// ListStackFiles finds all Pulumi.<stack>.yaml files in the workspace
// and extracts secrets provider configuration from each
func ListStackFiles(workDir string) ([]StackFileInfo, error) {
	pattern := filepath.Join(workDir, "Pulumi.*.yaml")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to glob stack files: %w", err)
	}

	var files []StackFileInfo
	for _, match := range matches {
		filename := filepath.Base(match)
		// Extract stack name from Pulumi.<stack>.yaml
		if !strings.HasPrefix(filename, "Pulumi.") || !strings.HasSuffix(filename, ".yaml") {
			continue
		}
		stackName := strings.TrimPrefix(filename, "Pulumi.")
		stackName = strings.TrimSuffix(stackName, ".yaml")

		info := parseStackFileInfo(stackName, match)
		files = append(files, info)
	}

	return files, nil
}

// InitStackOptions contains options for stack initialization
type InitStackOptions struct {
	SecretsProvider string
	Passphrase      string            // For passphrase-based secrets provider
	Env             map[string]string // Additional environment variables
}

// InitStack creates a new stack with the given configuration
func InitStack(ctx context.Context, workDir, stackName string, opts InitStackOptions) error {
	wsOpts := []auto.LocalWorkspaceOption{auto.WorkDir(workDir)}

	// Build env vars
	env := make(map[string]string)
	maps.Copy(env, opts.Env)

	// Set passphrase if provided
	if opts.Passphrase != "" {
		env["PULUMI_CONFIG_PASSPHRASE"] = opts.Passphrase
	}

	// Set secrets provider if provided
	if opts.SecretsProvider != "" {
		wsOpts = append(wsOpts, auto.SecretsProvider(opts.SecretsProvider))
	}

	if len(env) > 0 {
		wsOpts = append(wsOpts, auto.EnvVars(env))
	}

	// Create the stack
	_, err := auto.NewStackLocalSource(ctx, stackName, workDir, wsOpts...)
	if err != nil {
		return fmt.Errorf("failed to create stack: %w", err)
	}

	return nil
}

func findCurrentStack(ctx context.Context, ws auto.Workspace) string {
	stacks, err := ws.ListStacks(ctx)
	if err != nil || len(stacks) == 0 {
		return ""
	}

	for _, s := range stacks {
		if s.Current {
			return s.Name
		}
	}
	return stacks[0].Name
}

func parseStackFileInfo(stackName, filePath string) StackFileInfo {
	info := StackFileInfo{
		Name:     stackName,
		FilePath: filePath,
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return info
	}

	var config map[string]any
	if yaml.Unmarshal(data, &config) != nil {
		return info
	}

	if sp, ok := config["secretsprovider"].(string); ok {
		info.SecretsProvider = sp
	}

	_, hasSalt := config["encryptionsalt"]
	_, hasKey := config["encryptedkey"]
	info.HasEncryption = hasSalt || hasKey

	return info
}
