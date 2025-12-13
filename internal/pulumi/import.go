package pulumi

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/pulumi/pulumi/sdk/v3/go/auto/optimport"
)

// execCommand creates an exec.Command - can be mocked for testing
var execCommand = execCommandImpl

func execCommandImpl(ctx context.Context, name string, args ...string) *execCmd {
	cmd := &execCmd{
		name: name,
		args: args,
		ctx:  ctx,
	}
	return cmd
}

// execCmd wraps os/exec.Cmd for easier testing
type execCmd struct {
	name string
	args []string
	ctx  context.Context
	Dir  string
	Env  []string
}

func (c *execCmd) CombinedOutput() ([]byte, error) {
	cmd := exec.CommandContext(c.ctx, c.name, c.args...) //nolint:gosec // G204: Pulumi CLI command execution
	cmd.Dir = c.Dir
	cmd.Env = c.Env
	return cmd.CombinedOutput()
}

// runPulumiCommand executes a pulumi CLI command with environment variables
func runPulumiCommand(ctx context.Context, workDir string, env map[string]string, args ...string) (string, error) {
	cmd := execCommand(ctx, "pulumi", args...)
	cmd.Dir = workDir

	if len(env) > 0 {
		cmdEnv := os.Environ()
		for k, v := range env {
			cmdEnv = append(cmdEnv, k+"="+v)
		}
		cmd.Env = cmdEnv
	}

	output, err := cmd.CombinedOutput()
	return string(output), err
}

// ImportResource imports an existing resource into the Pulumi state using the SDK
// resourceType is the Pulumi resource type (e.g., "aws:s3/bucket:Bucket")
// resourceName is the logical name for the resource in Pulumi
// importID is the provider-specific ID of the existing resource to import
// parentURN is optional - if provided, the resource will be imported as a child of this resource
func ImportResource(ctx context.Context, workDir, stackName, resourceType, resourceName, importID, parentURN string, opts ImportOptions) (*CommandResult, error) {
	stack, err := selectStack(ctx, workDir, stackName, opts.Env)
	if err != nil {
		return nil, err
	}

	resource := &optimport.ImportResource{
		Type:   resourceType,
		Name:   resourceName,
		ID:     importID,
		Parent: parentURN,
	}

	var output bytes.Buffer
	_, err = stack.ImportResources(ctx,
		optimport.Resources([]*optimport.ImportResource{resource}),
		optimport.Protect(false),
		optimport.GenerateCode(false),
		optimport.ProgressStreams(&output),
		optimport.ErrorProgressStreams(&output),
	)
	if err != nil {
		return &CommandResult{
			Success: false,
			Output:  output.String(),
			Error:   fmt.Errorf("import failed: %w", err),
		}, nil
	}

	return &CommandResult{
		Success: true,
		Output:  output.String(),
	}, nil
}

// DeleteFromState removes a resource from the Pulumi state without deleting the actual resource
// urn is the full URN of the resource to remove from state
func DeleteFromState(ctx context.Context, workDir, stackName, urn string, opts StateDeleteOptions) (*CommandResult, error) {
	resolvedStackName, err := resolveStackName(ctx, workDir, stackName, opts.Env)
	if err != nil {
		return nil, err
	}

	// Build the pulumi state delete command
	// Format: pulumi state delete <urn> --stack <stack> --yes
	args := []string{
		"state",
		"delete",
		urn,
		"--stack", resolvedStackName,
		"--yes", // Auto-confirm
	}

	output, err := runPulumiCommand(ctx, workDir, opts.Env, args...)
	if err != nil {
		return &CommandResult{
			Success: false,
			Output:  output,
			Error:   fmt.Errorf("state delete failed: %w\n%s", err, output),
		}, nil
	}

	return &CommandResult{
		Success: true,
		Output:  output,
	}, nil
}

// ProtectResource marks a resource as protected in the Pulumi state
// Protected resources cannot be destroyed without first being unprotected
func ProtectResource(ctx context.Context, workDir, stackName, urn string, opts StateProtectOptions) (*CommandResult, error) {
	resolvedStackName, err := resolveStackName(ctx, workDir, stackName, opts.Env)
	if err != nil {
		return nil, err
	}

	// Build the pulumi state protect command
	// Format: pulumi state protect <urn> --stack <stack> --yes
	args := []string{
		"state",
		"protect",
		urn,
		"--stack", resolvedStackName,
		"--yes", // Auto-confirm
	}

	output, err := runPulumiCommand(ctx, workDir, opts.Env, args...)
	if err != nil {
		return &CommandResult{
			Success: false,
			Output:  output,
			Error:   fmt.Errorf("state protect failed: %w\n%s", err, output),
		}, nil
	}

	return &CommandResult{
		Success: true,
		Output:  output,
	}, nil
}

// UnprotectResource removes the protected flag from a resource in the Pulumi state
// This allows the resource to be destroyed
func UnprotectResource(ctx context.Context, workDir, stackName, urn string, opts StateProtectOptions) (*CommandResult, error) {
	resolvedStackName, err := resolveStackName(ctx, workDir, stackName, opts.Env)
	if err != nil {
		return nil, err
	}

	// Build the pulumi state unprotect command
	// Format: pulumi state unprotect <urn> --stack <stack> --yes
	args := []string{
		"state",
		"unprotect",
		urn,
		"--stack", resolvedStackName,
		"--yes", // Auto-confirm
	}

	output, err := runPulumiCommand(ctx, workDir, opts.Env, args...)
	if err != nil {
		return &CommandResult{
			Success: false,
			Output:  output,
			Error:   fmt.Errorf("state unprotect failed: %w\n%s", err, output),
		}, nil
	}

	return &CommandResult{
		Success: true,
		Output:  output,
	}, nil
}
