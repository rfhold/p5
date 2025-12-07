package pulumi

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pulumi/pulumi/sdk/v3/go/auto"
)

// GetStackResources returns the currently deployed resources in the stack
func GetStackResources(ctx context.Context, workDir, stackName string, env map[string]string) ([]ResourceInfo, error) {
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

	// Export the stack state
	state, err := stack.Export(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to export stack: %w", err)
	}

	// Parse the deployment to get resources with inputs and outputs
	var deployment struct {
		Resources []struct {
			URN      string         `json:"urn"`
			Type     string         `json:"type"`
			Provider string         `json:"provider"`
			Parent   string         `json:"parent"`
			Inputs   map[string]any `json:"inputs"`
			Outputs  map[string]any `json:"outputs"`
		} `json:"resources"`
	}

	if err := json.Unmarshal(state.Deployment, &deployment); err != nil {
		return nil, fmt.Errorf("failed to parse deployment: %w", err)
	}

	// First pass: build provider inputs map (provider URN -> inputs)
	providerInputs := make(map[string]map[string]any)
	for _, r := range deployment.Resources {
		// Provider resources have type like "pulumi:providers:kubernetes"
		if strings.HasPrefix(r.Type, "pulumi:providers:") {
			providerInputs[r.URN] = r.Inputs
		}
	}

	// Second pass: build resource list with provider inputs
	resources := make([]ResourceInfo, 0, len(deployment.Resources))
	for _, r := range deployment.Resources {
		info := ResourceInfo{
			URN:      r.URN,
			Type:     r.Type,
			Name:     ExtractResourceName(r.URN),
			Provider: r.Provider,
			Parent:   r.Parent,
			Inputs:   r.Inputs,
			Outputs:  r.Outputs,
		}

		// Look up provider inputs if this resource has a provider reference
		if r.Provider != "" {
			providerURN := extractProviderURN(r.Provider)
			if inputs, ok := providerInputs[providerURN]; ok {
				info.ProviderInputs = inputs
			}
		}

		resources = append(resources, info)
	}

	return resources, nil
}

// extractProviderURN extracts the URN from a provider reference string.
// Provider references are in format "URN::ID" where ID is a UUID.
// Example: "urn:pulumi:dev::proj::pulumi:providers:kubernetes::my-k8s::a1b2c3d4-..." -> "urn:pulumi:dev::proj::pulumi:providers:kubernetes::my-k8s"
func extractProviderURN(providerRef string) string {
	if providerRef == "" {
		return ""
	}
	// Find the last "::" followed by what looks like a UUID (contains hyphens, hex chars)
	// The URN ends at the provider name, then ::ID follows
	// Simple approach: split by "::" and check if last part looks like an ID
	parts := strings.Split(providerRef, "::")
	if len(parts) < 2 {
		return providerRef
	}
	// Check if last part looks like a UUID (common format: 8-4-4-4-12 hex chars)
	lastPart := parts[len(parts)-1]
	if looksLikeUUID(lastPart) {
		return strings.Join(parts[:len(parts)-1], "::")
	}
	return providerRef
}

// looksLikeUUID checks if a string looks like a UUID
func looksLikeUUID(s string) bool {
	// UUIDs are 36 chars: 8-4-4-4-12 with hyphens, or 32 hex chars without
	if len(s) == 36 && strings.Count(s, "-") == 4 {
		return true
	}
	if len(s) == 32 {
		for _, c := range s {
			if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
				return false
			}
		}
		return true
	}
	return false
}
