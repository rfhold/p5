package pulumi

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/auto"
)

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
			Name:     ExtractResourceName(r.URN),
			Provider: r.Provider,
			Parent:   r.Parent,
			Inputs:   r.Inputs,
			Outputs:  r.Outputs,
		})
	}

	return resources, nil
}
