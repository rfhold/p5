package builtins

import (
	"context"
	"testing"

	"github.com/rfhold/p5/internal/plugins"
	"github.com/rfhold/p5/internal/plugins/proto"
	"github.com/rfhold/p5/pkg/plugin"
)

// =============================================================================
// buildKubectlResource Tests
// =============================================================================

func TestBuildKubectlResource_CoreAPI(t *testing.T) {
	tests := []struct {
		apiVersion string
		kind       string
		expected   string
	}{
		{"v1", "Namespace", "namespace"},
		{"v1", "Pod", "pod"},
		{"v1", "Service", "service"},
		{"v1", "ConfigMap", "configmap"},
		{"v1", "Secret", "secret"},
		{"v1", "PersistentVolumeClaim", "persistentvolumeclaim"},
	}

	for _, tc := range tests {
		t.Run(tc.kind, func(t *testing.T) {
			result := buildKubectlResource(tc.apiVersion, tc.kind)
			if result != tc.expected {
				t.Errorf("buildKubectlResource(%q, %q) = %q, want %q", tc.apiVersion, tc.kind, result, tc.expected)
			}
		})
	}
}

func TestBuildKubectlResource_AppsAPI(t *testing.T) {
	tests := []struct {
		apiVersion string
		kind       string
		expected   string
	}{
		{"apps/v1", "Deployment", "deployment.apps"},
		{"apps/v1", "StatefulSet", "statefulset.apps"},
		{"apps/v1", "DaemonSet", "daemonset.apps"},
		{"apps/v1", "ReplicaSet", "replicaset.apps"},
	}

	for _, tc := range tests {
		t.Run(tc.kind, func(t *testing.T) {
			result := buildKubectlResource(tc.apiVersion, tc.kind)
			if result != tc.expected {
				t.Errorf("buildKubectlResource(%q, %q) = %q, want %q", tc.apiVersion, tc.kind, result, tc.expected)
			}
		})
	}
}

func TestBuildKubectlResource_ExtendedAPIs(t *testing.T) {
	tests := []struct {
		apiVersion string
		kind       string
		expected   string
	}{
		{"networking.k8s.io/v1", "Ingress", "ingress.networking.k8s.io"},
		{"networking.k8s.io/v1", "NetworkPolicy", "networkpolicy.networking.k8s.io"},
		{"batch/v1", "Job", "job.batch"},
		{"batch/v1", "CronJob", "cronjob.batch"},
		{"autoscaling/v2", "HorizontalPodAutoscaler", "horizontalpodautoscaler.autoscaling"},
		{"rbac.authorization.k8s.io/v1", "Role", "role.rbac.authorization.k8s.io"},
		{"rbac.authorization.k8s.io/v1", "ClusterRole", "clusterrole.rbac.authorization.k8s.io"},
		{"cert-manager.io/v1", "Certificate", "certificate.cert-manager.io"},
		{"cert-manager.io/v1", "ClusterIssuer", "clusterissuer.cert-manager.io"},
	}

	for _, tc := range tests {
		t.Run(tc.kind, func(t *testing.T) {
			result := buildKubectlResource(tc.apiVersion, tc.kind)
			if result != tc.expected {
				t.Errorf("buildKubectlResource(%q, %q) = %q, want %q", tc.apiVersion, tc.kind, result, tc.expected)
			}
		})
	}
}

// =============================================================================
// clusterScopedKinds Tests
// =============================================================================

func TestClusterScopedKinds_Known(t *testing.T) {
	knownClusterScoped := []string{
		"Namespace",
		"Node",
		"PersistentVolume",
		"ClusterRole",
		"ClusterRoleBinding",
		"StorageClass",
		"IngressClass",
		"ClusterIssuer",
	}

	for _, kind := range knownClusterScoped {
		t.Run(kind, func(t *testing.T) {
			if !clusterScopedKinds[kind] {
				t.Errorf("expected %q to be cluster-scoped", kind)
			}
		})
	}
}

func TestClusterScopedKinds_NotClusterScoped(t *testing.T) {
	namespacedKinds := []string{
		"Pod",
		"Deployment",
		"Service",
		"ConfigMap",
		"Secret",
		"Ingress",
	}

	for _, kind := range namespacedKinds {
		t.Run(kind, func(t *testing.T) {
			if clusterScopedKinds[kind] {
				t.Errorf("expected %q to NOT be cluster-scoped", kind)
			}
		})
	}
}

// =============================================================================
// KubernetesPlugin Registration Tests
// =============================================================================

func TestKubernetesPlugin_Name(t *testing.T) {
	p := &KubernetesPlugin{
		BuiltinPluginBase: plugins.NewBuiltinPluginBase("kubernetes"),
	}

	if p.Name() != "kubernetes" {
		t.Errorf("expected Name=%q, got %q", "kubernetes", p.Name())
	}
}

func TestKubernetesPlugin_Authenticate(t *testing.T) {
	p := &KubernetesPlugin{
		BuiltinPluginBase: plugins.NewBuiltinPluginBase("kubernetes"),
	}

	ctx := context.Background()
	req := &proto.AuthenticateRequest{
		ProgramConfig: map[string]string{},
		StackConfig:   map[string]string{},
	}

	resp, err := p.Authenticate(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Error("expected Success=true")
	}
}

// =============================================================================
// GetImportSuggestions Tests
// =============================================================================

func TestKubernetesPlugin_GetImportSuggestions_NotSupported(t *testing.T) {
	p := &KubernetesPlugin{
		BuiltinPluginBase: plugins.NewBuiltinPluginBase("kubernetes"),
	}

	tests := []struct {
		name   string
		inputs map[string]string
	}{
		{
			name:   "missing_apiVersion",
			inputs: map[string]string{"kind": "Pod"},
		},
		{
			name:   "missing_kind",
			inputs: map[string]string{"apiVersion": "v1"},
		},
		{
			name:   "empty_inputs",
			inputs: map[string]string{},
		},
		{
			name:   "empty_values",
			inputs: map[string]string{"apiVersion": "", "kind": ""},
		},
	}

	ctx := context.Background()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := &plugin.ImportSuggestionsRequest{
				ResourceType: "kubernetes:core/v1:Pod",
				Inputs:       tc.inputs,
			}

			resp, err := p.GetImportSuggestions(ctx, req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.CanProvide {
				t.Error("expected CanProvide=false for missing apiVersion/kind")
			}
		})
	}
}

func TestKubernetesPlugin_GetImportSuggestions_ContextPriority(t *testing.T) {
	// This tests that context is resolved in the correct priority order:
	// provider inputs > stack config > program config
	// We can't actually run kubectl, but we can verify the logic is correct
	// by checking the function doesn't error with valid inputs

	p := &KubernetesPlugin{
		BuiltinPluginBase: plugins.NewBuiltinPluginBase("kubernetes"),
	}

	tests := []struct {
		name            string
		providerInputs  map[string]string
		stackConfig     map[string]string
		programConfig   map[string]string
		expectedContext string // For documentation, can't actually verify without kubectl
	}{
		{
			name:            "provider_context_wins",
			providerInputs:  map[string]string{"context": "provider-ctx"},
			stackConfig:     map[string]string{"kubernetes:context": "stack-ctx"},
			programConfig:   map[string]string{"kubernetes:context": "program-ctx"},
			expectedContext: "provider-ctx",
		},
		{
			name:            "stack_context_fallback",
			providerInputs:  map[string]string{},
			stackConfig:     map[string]string{"kubernetes:context": "stack-ctx"},
			programConfig:   map[string]string{"kubernetes:context": "program-ctx"},
			expectedContext: "stack-ctx",
		},
		{
			name:            "program_context_fallback",
			providerInputs:  map[string]string{},
			stackConfig:     map[string]string{},
			programConfig:   map[string]string{"kubernetes:context": "program-ctx"},
			expectedContext: "program-ctx",
		},
	}

	// We just verify the function doesn't panic with various configs
	// (kubectl will fail but that's expected without a real cluster)
	ctx := context.Background()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := &plugin.ImportSuggestionsRequest{
				ResourceType:   "kubernetes:core/v1:Namespace",
				Inputs:         map[string]string{"apiVersion": "v1", "kind": "Namespace"},
				ProviderInputs: tc.providerInputs,
				StackConfig:    tc.stackConfig,
				ProgramConfig:  tc.programConfig,
			}

			// This will fail because kubectl isn't available/configured,
			// but we're testing that the function handles config correctly
			resp, err := p.GetImportSuggestions(ctx, req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Response should indicate kubectl failed (not that we don't support it)
			if !resp.CanProvide {
				t.Error("expected CanProvide=true (even with error)")
			}
		})
	}
}

func TestKubernetesPlugin_GetImportSuggestions_NamespacePriority(t *testing.T) {
	// Tests namespace resolution priority:
	// resource metadata > provider inputs > stack config > program config

	p := &KubernetesPlugin{
		BuiltinPluginBase: plugins.NewBuiltinPluginBase("kubernetes"),
	}

	tests := []struct {
		name              string
		inputs            map[string]string
		providerInputs    map[string]string
		stackConfig       map[string]string
		programConfig     map[string]string
		expectedNamespace string // For documentation
	}{
		{
			name:              "metadata_namespace_wins",
			inputs:            map[string]string{"apiVersion": "v1", "kind": "Pod", "metadata": `{"namespace":"meta-ns"}`},
			providerInputs:    map[string]string{"namespace": "provider-ns"},
			stackConfig:       map[string]string{"kubernetes:namespace": "stack-ns"},
			programConfig:     map[string]string{"kubernetes:namespace": "program-ns"},
			expectedNamespace: "meta-ns",
		},
		{
			name:              "provider_namespace_fallback",
			inputs:            map[string]string{"apiVersion": "v1", "kind": "Pod", "metadata": `{}`},
			providerInputs:    map[string]string{"namespace": "provider-ns"},
			stackConfig:       map[string]string{"kubernetes:namespace": "stack-ns"},
			programConfig:     map[string]string{"kubernetes:namespace": "program-ns"},
			expectedNamespace: "provider-ns",
		},
		{
			name:              "stack_namespace_fallback",
			inputs:            map[string]string{"apiVersion": "v1", "kind": "Pod"},
			providerInputs:    map[string]string{},
			stackConfig:       map[string]string{"kubernetes:namespace": "stack-ns"},
			programConfig:     map[string]string{"kubernetes:namespace": "program-ns"},
			expectedNamespace: "stack-ns",
		},
		{
			name:              "program_namespace_fallback",
			inputs:            map[string]string{"apiVersion": "v1", "kind": "Pod"},
			providerInputs:    map[string]string{},
			stackConfig:       map[string]string{},
			programConfig:     map[string]string{"kubernetes:namespace": "program-ns"},
			expectedNamespace: "program-ns",
		},
	}

	ctx := context.Background()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := &plugin.ImportSuggestionsRequest{
				ResourceType:   "kubernetes:core/v1:Pod",
				Inputs:         tc.inputs,
				ProviderInputs: tc.providerInputs,
				StackConfig:    tc.stackConfig,
				ProgramConfig:  tc.programConfig,
			}

			// This will fail because kubectl isn't available/configured,
			// but we're testing that the function handles config correctly
			resp, err := p.GetImportSuggestions(ctx, req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Response should indicate kubectl failed (not that we don't support it)
			if !resp.CanProvide {
				t.Error("expected CanProvide=true (even with error)")
			}
		})
	}
}

func TestKubernetesPlugin_GetImportSuggestions_ClusterScoped(t *testing.T) {
	// Tests that cluster-scoped resources don't include namespace args

	p := &KubernetesPlugin{
		BuiltinPluginBase: plugins.NewBuiltinPluginBase("kubernetes"),
	}

	ctx := context.Background()

	req := &plugin.ImportSuggestionsRequest{
		ResourceType: "kubernetes:core/v1:Namespace",
		Inputs:       map[string]string{"apiVersion": "v1", "kind": "Namespace"},
		// Even with namespace config, it should be ignored for cluster-scoped resources
		StackConfig: map[string]string{"kubernetes:namespace": "should-be-ignored"},
	}

	resp, err := p.GetImportSuggestions(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// We can only verify the function completes without error
	// kubectl will fail but namespace shouldn't be in the args
	if !resp.CanProvide {
		t.Error("expected CanProvide=true (even with error)")
	}
}
