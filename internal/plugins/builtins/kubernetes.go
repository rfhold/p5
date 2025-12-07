package builtins

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"

	"github.com/rfhold/p5/internal/plugins"
	"github.com/rfhold/p5/internal/plugins/proto"
	"github.com/rfhold/p5/pkg/plugin"
)

func init() {
	plugins.RegisterBuiltin(&KubernetesPlugin{
		BuiltinPluginBase: plugins.NewBuiltinPluginBase("kubernetes"),
	})
}

// KubernetesPlugin provides import suggestions for Kubernetes resources
// by querying kubectl for existing resources.
type KubernetesPlugin struct {
	plugins.BuiltinPluginBase
}

// Authenticate returns a no-op success response.
// This plugin is primarily for import help, not auth.
func (p *KubernetesPlugin) Authenticate(ctx context.Context, req *proto.AuthenticateRequest) (*proto.AuthenticateResponse, error) {
	return plugins.SuccessResponse(nil, 0), nil
}

// kubeResource represents a Kubernetes resource from kubectl output
type kubeResource struct {
	Metadata struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace,omitempty"`
	} `json:"metadata"`
}

// kubeResourceList represents the list response from kubectl
type kubeResourceList struct {
	Items []kubeResource `json:"items"`
}

// clusterScopedKinds lists Kubernetes kinds that don't have a namespace.
var clusterScopedKinds = map[string]bool{
	"Namespace":          true,
	"Node":               true,
	"PersistentVolume":   true,
	"ClusterRole":        true,
	"ClusterRoleBinding": true,
	"StorageClass":       true,
	"IngressClass":       true,
	"ClusterIssuer":      true, // cert-manager CRD
}

// buildKubectlResource builds a kubectl resource identifier from apiVersion and kind.

func buildKubectlResource(apiVersion, kind string) string {
	kindLower := strings.ToLower(kind)

	// Extract API group from apiVersion (e.g., "apps/v1" -> "apps", "v1" -> "")
	if idx := strings.Index(apiVersion, "/"); idx > 0 {
		apiGroup := apiVersion[:idx]
		return kindLower + "." + apiGroup
	}

	// Core API (just "v1") - no group suffix
	return kindLower
}

// GetImportSuggestions returns import ID suggestions for Kubernetes resources
func (p *KubernetesPlugin) GetImportSuggestions(ctx context.Context, req *plugin.ImportSuggestionsRequest) (*plugin.ImportSuggestionsResponse, error) {
	// All Kubernetes resources have apiVersion and kind in their inputs
	apiVersion := req.Inputs["apiVersion"]
	kind := req.Inputs["kind"]
	if apiVersion == "" || kind == "" {
		return plugin.ImportSuggestionsNotSupported(), nil
	}

	kubeKind := buildKubectlResource(apiVersion, kind)
	isClusterScoped := clusterScopedKinds[kind]

	// Build kubectl command
	args := []string{"get", kubeKind, "-o", "json"}

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

	// Handle kubeconfig from provider
	var tempKubeconfig string
	kubeconfig := req.ProviderInputs["kubeconfig"]
	if kubeconfig != "" {
		// kubeconfig could be file path or content
		// If it looks like YAML/JSON content, write to temp file
		if strings.Contains(kubeconfig, "apiVersion:") || strings.Contains(kubeconfig, "\"apiVersion\"") {
			tmpFile, err := os.CreateTemp("", "p5-kubeconfig-*.yaml")
			if err == nil {
				_, _ = tmpFile.WriteString(kubeconfig)
				tmpFile.Close()
				tempKubeconfig = tmpFile.Name()
				defer os.Remove(tempKubeconfig)
				args = append(args, "--kubeconfig", tempKubeconfig)
			}
		} else {
			// Assume it's a file path
			args = append(args, "--kubeconfig", kubeconfig)
		}
	}

	// Add namespace for namespaced resources
	if !isClusterScoped {
		args = appendNamespaceArgs(args, req)
	}

	// Run kubectl
	cmd := exec.CommandContext(ctx, "kubectl", args...)

	// Pass through auth environment if provided
	if len(req.AuthEnv) > 0 {
		env := cmd.Environ()
		for k, v := range req.AuthEnv {
			env = append(env, k+"="+v)
		}
		cmd.Env = env
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// kubectl failed - might not have access or resource type doesn't exist
		return plugin.ImportSuggestionsError("kubectl failed: %s", stderr.String()), nil
	}

	// Parse the JSON output
	var resources kubeResourceList
	if err := json.Unmarshal(stdout.Bytes(), &resources); err != nil {
		return plugin.ImportSuggestionsError("failed to parse kubectl output: %v", err), nil
	}

	// Convert to suggestions
	suggestions := make([]*plugin.ImportSuggestion, 0, len(resources.Items))
	for _, item := range resources.Items {
		var importID, description string

		switch {
		case isClusterScoped:
			// Cluster-scoped: just the name
			importID = item.Metadata.Name
			description = "Cluster resource"
		case item.Metadata.Namespace != "":
			// Namespaced: namespace/name format
			importID = item.Metadata.Namespace + "/" + item.Metadata.Name
			description = "Namespace: " + item.Metadata.Namespace
		default:
			// Fallback to just name
			importID = item.Metadata.Name
			description = ""
		}

		suggestions = append(suggestions, plugin.NewImportSuggestion(
			importID,
			item.Metadata.Name,
			description,
		))
	}

	return plugin.ImportSuggestionsSuccess(suggestions), nil
}

func appendNamespaceArgs(args []string, req *plugin.ImportSuggestionsRequest) []string {
	namespace := resolveK8sNamespace(req)
	if namespace != "" {
		return append(args, "-n", namespace)
	}
	return append(args, "--all-namespaces")
}

func resolveK8sNamespace(req *plugin.ImportSuggestionsRequest) string {
	// Priority: resource metadata > provider inputs > stack config > program config
	if ns := extractK8sNamespace(req.Inputs["metadata"]); ns != "" {
		return ns
	}
	if ns := req.ProviderInputs["namespace"]; ns != "" {
		return ns
	}
	if ns := req.StackConfig["kubernetes:namespace"]; ns != "" {
		return ns
	}
	return req.ProgramConfig["kubernetes:namespace"]
}
