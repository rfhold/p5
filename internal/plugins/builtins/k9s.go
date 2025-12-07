package builtins

import (
	"context"
	"encoding/json"
	"os"
	"strings"

	"github.com/rfhold/p5/internal/plugins"
	"github.com/rfhold/p5/internal/plugins/proto"
	"github.com/rfhold/p5/pkg/plugin"
)

func init() {
	plugins.RegisterBuiltin(&K9sPlugin{
		BuiltinPluginBase: plugins.NewBuiltinPluginBase("k9s"),
	})
}

// K9sPlugin provides resource opening capabilities for Kubernetes resources
// by launching k9s with the appropriate context and navigating to the resource.
type K9sPlugin struct {
	plugins.BuiltinPluginBase
}

// Authenticate returns a no-op success response.
// This plugin is primarily for resource opening, not auth.
func (p *K9sPlugin) Authenticate(ctx context.Context, req *proto.AuthenticateRequest) (*proto.AuthenticateResponse, error) {
	return plugins.SuccessResponse(nil, 0), nil
}

// GetSupportedOpenTypes returns regex patterns for Kubernetes resource types.
func (p *K9sPlugin) GetSupportedOpenTypes(ctx context.Context, req *plugin.SupportedOpenTypesRequest) (*plugin.SupportedOpenTypesResponse, error) {
	return plugin.SupportedOpenTypesPatterns(
		`^kubernetes:.*`,
	), nil
}

// OpenResource returns the k9s command to open a Kubernetes resource.
func (p *K9sPlugin) OpenResource(ctx context.Context, req *plugin.OpenResourceRequest) (*plugin.OpenResourceResponse, error) {
	// Parse resource type to extract kind
	// Format: kubernetes:GROUP/VERSION:KIND (e.g., "kubernetes:core/v1:Pod", "kubernetes:apps/v1:Deployment")
	kind := extractK8sKind(req.ResourceType)
	if kind == "" {
		return plugin.OpenNotSupported(), nil
	}

	// Build k9s command arguments
	args := []string{}
	env := make(map[string]string)

	// Get kubeconfig - priority: provider inputs > stack config > program config
	kubeconfig := req.ProviderInputs["kubeconfig"]
	if kubeconfig == "" {
		kubeconfig = req.StackConfig["kubernetes:kubeconfig"]
	}
	if kubeconfig == "" {
		kubeconfig = req.ProgramConfig["kubernetes:kubeconfig"]
	}

	// Handle kubeconfig: could be file path or content
	if kubeconfig != "" {
		if isKubeconfigContent(kubeconfig) {
			// It's YAML/JSON content - write to temp file
			tmpFile, err := os.CreateTemp("", "p5-kubeconfig-*.yaml")
			if err == nil {
				tmpFile.WriteString(kubeconfig)
				tmpFile.Close()
				// Note: This temp file will persist until k9s exits
				// k9s runs in foreground so cleanup would happen after
				args = append(args, "--kubeconfig", tmpFile.Name())
			}
		} else {
			// It's a file path
			args = append(args, "--kubeconfig", kubeconfig)
		}
	}

	// Get context - priority: provider inputs > stack config > program config
	kubeContext := req.ProviderInputs["context"]
	if kubeContext == "" {
		kubeContext = req.StackConfig["kubernetes:context"]
	}
	if kubeContext == "" {
		kubeContext = req.ProgramConfig["kubernetes:context"]
	}
	if kubeContext != "" {
		args = append(args, "--context", kubeContext)
	}

	// Get namespace - priority: resource metadata > provider inputs > stack config > program config
	// Kubernetes resources store namespace in metadata.namespace (JSON serialized)
	namespace := extractK8sNamespace(req.Inputs["metadata"])
	if namespace == "" {
		namespace = req.ProviderInputs["namespace"]
	}
	if namespace == "" {
		namespace = req.StackConfig["kubernetes:namespace"]
	}
	if namespace == "" {
		namespace = req.ProgramConfig["kubernetes:namespace"]
	}
	if namespace != "" {
		args = append(args, "--namespace", namespace)
	}

	// Use --command to navigate to the specific resource type
	// k9s --command <resource-kind> opens k9s directly to that resource view
	args = append(args, "--command", kind)

	// Pass through auth environment if provided
	for k, v := range req.AuthEnv {
		env[k] = v
	}

	return plugin.OpenExecResponse("k9s", args, env), nil
}

// extractK8sKind extracts the Kubernetes kind from a Pulumi resource type.

func extractK8sKind(resourceType string) string {
	// Must start with "kubernetes:"
	if !strings.HasPrefix(resourceType, "kubernetes:") {
		return ""
	}

	// Split by ":"
	parts := strings.Split(resourceType, ":")
	if len(parts) < 3 {
		return ""
	}

	// The kind is the last part
	kind := parts[len(parts)-1]
	return strings.ToLower(kind)
}

// isKubeconfigContent checks if the string looks like kubeconfig content rather than a file path.
func isKubeconfigContent(s string) bool {
	return strings.Contains(s, "apiVersion:") || strings.Contains(s, `"apiVersion"`)
}

// extractK8sNamespace extracts the namespace from a Kubernetes metadata JSON string.
// The metadata field is serialized as JSON, e.g., {"name":"foo","namespace":"default"}
func extractK8sNamespace(metadataJSON string) string {
	if metadataJSON == "" {
		return ""
	}

	var metadata struct {
		Namespace string `json:"namespace"`
	}
	if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
		return ""
	}
	return metadata.Namespace
}
