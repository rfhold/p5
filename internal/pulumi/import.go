package pulumi

import (
	"context"
	"fmt"
	"os"
	"os/exec"
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
	cmd := exec.CommandContext(c.ctx, c.name, c.args...)
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

// ImportResource imports an existing resource into the Pulumi state using the CLI
// resourceType is the Pulumi resource type (e.g., "aws:s3/bucket:Bucket")
// resourceName is the logical name for the resource in Pulumi
// importID is the provider-specific ID of the existing resource to import
// parentURN is optional - if provided, the resource will be imported as a child of this resource
func ImportResource(ctx context.Context, workDir, stackName string, resourceType, resourceName, importID, parentURN string, opts ImportOptions) (*CommandResult, error) {
	resolvedStackName, err := resolveStackName(ctx, workDir, stackName)
	if err != nil {
		return nil, err
	}

	// Build the pulumi import command
	// Format: pulumi import <type> <name> <id> [--parent <parent-urn>] --yes
	args := []string{
		"import",
		resourceType,
		resourceName,
		importID,
		"--stack", resolvedStackName,
		"--yes",                 // Auto-confirm
		"--skip-preview",        // Skip the preview
		"--protect=false",       // Don't protect the imported resource
		"--generate-code=false", // Skip code generation (avoids issues with --parent flag)
	}

	// Add parent URN if provided
	if parentURN != "" {
		args = append(args, "--parent", parentURN)
	}

	output, err := runPulumiCommand(ctx, workDir, opts.Env, args...)
	if err != nil {
		return &CommandResult{
			Success: false,
			Output:  output,
			Error:   fmt.Errorf("import failed: %w\n%s", err, output),
		}, nil
	}

	return &CommandResult{
		Success: true,
		Output:  output,
	}, nil
}

// DeleteFromState removes a resource from the Pulumi state without deleting the actual resource
// urn is the full URN of the resource to remove from state
func DeleteFromState(ctx context.Context, workDir, stackName string, urn string, opts StateDeleteOptions) (*CommandResult, error) {
	resolvedStackName, err := resolveStackName(ctx, workDir, stackName)
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
