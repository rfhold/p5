package builtins

import (
	"context"
	"testing"

	"github.com/rfhold/p5/internal/plugins"
	"github.com/rfhold/p5/internal/plugins/proto"
	"github.com/rfhold/p5/pkg/plugin"
)

// =============================================================================
// extractK8sKind Tests
// =============================================================================

func TestExtractK8sKind_ValidResourceTypes(t *testing.T) {
	tests := []struct {
		resourceType string
		expected     string
	}{
		{"kubernetes:core/v1:Pod", "pod"},
		{"kubernetes:core/v1:Service", "service"},
		{"kubernetes:core/v1:ConfigMap", "configmap"},
		{"kubernetes:core/v1:Secret", "secret"},
		{"kubernetes:core/v1:Namespace", "namespace"},
		{"kubernetes:apps/v1:Deployment", "deployment"},
		{"kubernetes:apps/v1:StatefulSet", "statefulset"},
		{"kubernetes:apps/v1:DaemonSet", "daemonset"},
		{"kubernetes:networking.k8s.io/v1:Ingress", "ingress"},
		{"kubernetes:networking.k8s.io/v1:NetworkPolicy", "networkpolicy"},
		{"kubernetes:batch/v1:Job", "job"},
		{"kubernetes:batch/v1:CronJob", "cronjob"},
		{"kubernetes:rbac.authorization.k8s.io/v1:Role", "role"},
		{"kubernetes:rbac.authorization.k8s.io/v1:ClusterRole", "clusterrole"},
		{"kubernetes:cert-manager.io/v1:Certificate", "certificate"},
		{"kubernetes:autoscaling/v2:HorizontalPodAutoscaler", "horizontalpodautoscaler"},
	}

	for _, tc := range tests {
		t.Run(tc.resourceType, func(t *testing.T) {
			result := extractK8sKind(tc.resourceType)
			if result != tc.expected {
				t.Errorf("extractK8sKind(%q) = %q, want %q", tc.resourceType, result, tc.expected)
			}
		})
	}
}

func TestExtractK8sKind_InvalidResourceTypes(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
	}{
		{"not_kubernetes", "aws:ec2/instance:Instance"},
		{"missing_kind", "kubernetes:core/v1"},
		{"no_prefix", "core/v1:Pod"},
		{"empty", ""},
		{"only_kubernetes", "kubernetes:"},
		{"wrong_prefix", "k8s:core/v1:Pod"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := extractK8sKind(tc.resourceType)
			if result != "" {
				t.Errorf("extractK8sKind(%q) = %q, want empty string", tc.resourceType, result)
			}
		})
	}
}

// =============================================================================
// extractK8sNamespace Tests
// =============================================================================

func TestExtractK8sNamespace_ValidJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "with_namespace",
			input:    `{"name":"my-pod","namespace":"default"}`,
			expected: "default",
		},
		{
			name:     "different_namespace",
			input:    `{"name":"my-pod","namespace":"kube-system"}`,
			expected: "kube-system",
		},
		{
			name:     "complex_namespace",
			input:    `{"name":"my-pod","namespace":"ai-inference","labels":{"app":"test"}}`,
			expected: "ai-inference",
		},
		{
			name:     "no_namespace_field",
			input:    `{"name":"my-pod"}`,
			expected: "",
		},
		{
			name:     "empty_namespace",
			input:    `{"name":"my-pod","namespace":""}`,
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := extractK8sNamespace(tc.input)
			if result != tc.expected {
				t.Errorf("extractK8sNamespace(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestExtractK8sNamespace_InvalidJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty_string", ""},
		{"invalid_json", "not-json"},
		{"incomplete_json", `{"name":"test"`},
		{"array_not_object", `["namespace","default"]`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := extractK8sNamespace(tc.input)
			if result != "" {
				t.Errorf("extractK8sNamespace(%q) = %q, want empty string", tc.input, result)
			}
		})
	}
}

// =============================================================================
// isKubeconfigContent Tests
// =============================================================================

func TestIsKubeconfigContent_YAMLContent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name: "yaml_kubeconfig",
			input: `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://example.com
  name: my-cluster`,
			expected: true,
		},
		{
			name:     "yaml_apiVersion_line",
			input:    "apiVersion: v1",
			expected: true,
		},
		{
			name:     "file_path",
			input:    "/home/user/.kube/config",
			expected: false,
		},
		{
			name:     "relative_path",
			input:    "~/.kube/config",
			expected: false,
		},
		{
			name:     "windows_path",
			input:    `C:\Users\user\.kube\config`,
			expected: false,
		},
		{
			name:     "empty",
			input:    "",
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := isKubeconfigContent(tc.input)
			if result != tc.expected {
				t.Errorf("isKubeconfigContent(%q) = %v, want %v", tc.input, result, tc.expected)
			}
		})
	}
}

func TestIsKubeconfigContent_JSONContent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "json_kubeconfig",
			input:    `{"apiVersion":"v1","kind":"Config","clusters":[]}`,
			expected: true,
		},
		{
			name:     "json_apiVersion_only",
			input:    `{"apiVersion":"v1"}`,
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := isKubeconfigContent(tc.input)
			if result != tc.expected {
				t.Errorf("isKubeconfigContent(%q) = %v, want %v", tc.input, result, tc.expected)
			}
		})
	}
}

// =============================================================================
// K9sPlugin Registration Tests
// =============================================================================

func TestK9sPlugin_Name(t *testing.T) {
	p := &K9sPlugin{
		BuiltinPluginBase: plugins.NewBuiltinPluginBase("k9s"),
	}

	if p.Name() != "k9s" {
		t.Errorf("expected Name=%q, got %q", "k9s", p.Name())
	}
}

func TestK9sPlugin_Authenticate(t *testing.T) {
	p := &K9sPlugin{
		BuiltinPluginBase: plugins.NewBuiltinPluginBase("k9s"),
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
// GetSupportedOpenTypes Tests
// =============================================================================

func TestK9sPlugin_GetSupportedOpenTypes(t *testing.T) {
	p := &K9sPlugin{
		BuiltinPluginBase: plugins.NewBuiltinPluginBase("k9s"),
	}

	ctx := context.Background()
	req := &plugin.SupportedOpenTypesRequest{}

	resp, err := p.GetSupportedOpenTypes(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.ResourceTypePatterns) == 0 {
		t.Fatal("expected at least one pattern")
	}

	// Should contain kubernetes pattern
	found := false
	for _, pattern := range resp.ResourceTypePatterns {
		if pattern == `^kubernetes:.*` {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected pattern ^kubernetes:.* in %v", resp.ResourceTypePatterns)
	}
}

// =============================================================================
// OpenResource Tests
// =============================================================================

func TestK9sPlugin_OpenResource_ValidKubernetesResource(t *testing.T) {
	p := &K9sPlugin{
		BuiltinPluginBase: plugins.NewBuiltinPluginBase("k9s"),
	}

	ctx := context.Background()
	req := &plugin.OpenResourceRequest{
		ResourceType: "kubernetes:core/v1:Pod",
		ResourceName: "my-pod",
		ResourceUrn:  "urn:pulumi:dev::project::kubernetes:core/v1:Pod::my-pod",
		Inputs:       map[string]string{"metadata": `{"name":"my-pod","namespace":"default"}`},
	}

	resp, err := p.OpenResource(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.CanOpen {
		t.Error("expected CanOpen=true")
	}
	if resp.Action == nil {
		t.Fatal("expected Action to be set")
	}
	if resp.Action.Command != "k9s" {
		t.Errorf("expected Command=%q, got %q", "k9s", resp.Action.Command)
	}

	// Check that --command pod is in args
	foundCommand := false
	for i, arg := range resp.Action.Args {
		if arg == "--command" && i+1 < len(resp.Action.Args) && resp.Action.Args[i+1] == "pod" {
			foundCommand = true
			break
		}
	}
	if !foundCommand {
		t.Errorf("expected --command pod in args: %v", resp.Action.Args)
	}

	// Check that --namespace default is in args
	foundNamespace := false
	for i, arg := range resp.Action.Args {
		if arg == "--namespace" && i+1 < len(resp.Action.Args) && resp.Action.Args[i+1] == "default" {
			foundNamespace = true
			break
		}
	}
	if !foundNamespace {
		t.Errorf("expected --namespace default in args: %v", resp.Action.Args)
	}
}

func TestK9sPlugin_OpenResource_NotSupported(t *testing.T) {
	p := &K9sPlugin{
		BuiltinPluginBase: plugins.NewBuiltinPluginBase("k9s"),
	}

	tests := []struct {
		name         string
		resourceType string
	}{
		{"aws_resource", "aws:ec2/instance:Instance"},
		{"azure_resource", "azure:compute:VirtualMachine"},
		{"empty_type", ""},
		{"invalid_kubernetes", "kubernetes:"},
		{"kubernetes_no_kind", "kubernetes:core/v1"},
	}

	ctx := context.Background()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := &plugin.OpenResourceRequest{
				ResourceType: tc.resourceType,
				ResourceName: "test",
			}

			resp, err := p.OpenResource(ctx, req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.CanOpen {
				t.Errorf("expected CanOpen=false for %q", tc.resourceType)
			}
		})
	}
}

func TestK9sPlugin_OpenResource_WithContext(t *testing.T) {
	p := &K9sPlugin{
		BuiltinPluginBase: plugins.NewBuiltinPluginBase("k9s"),
	}

	tests := []struct {
		name            string
		providerInputs  map[string]string
		stackConfig     map[string]string
		programConfig   map[string]string
		expectedContext string
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
		{
			name:            "no_context",
			providerInputs:  map[string]string{},
			stackConfig:     map[string]string{},
			programConfig:   map[string]string{},
			expectedContext: "",
		},
	}

	ctx := context.Background()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := &plugin.OpenResourceRequest{
				ResourceType:   "kubernetes:apps/v1:Deployment",
				ResourceName:   "my-deployment",
				ProviderInputs: tc.providerInputs,
				StackConfig:    tc.stackConfig,
				ProgramConfig:  tc.programConfig,
			}

			resp, err := p.OpenResource(ctx, req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !resp.CanOpen {
				t.Fatal("expected CanOpen=true")
			}

			// Check for --context in args
			foundContext := ""
			for i, arg := range resp.Action.Args {
				if arg == "--context" && i+1 < len(resp.Action.Args) {
					foundContext = resp.Action.Args[i+1]
					break
				}
			}

			if foundContext != tc.expectedContext {
				t.Errorf("expected context=%q, got %q in args: %v", tc.expectedContext, foundContext, resp.Action.Args)
			}
		})
	}
}

func TestK9sPlugin_OpenResource_WithNamespace(t *testing.T) {
	p := &K9sPlugin{
		BuiltinPluginBase: plugins.NewBuiltinPluginBase("k9s"),
	}

	tests := []struct {
		name              string
		inputs            map[string]string
		providerInputs    map[string]string
		stackConfig       map[string]string
		programConfig     map[string]string
		expectedNamespace string
	}{
		{
			name:              "metadata_namespace_wins",
			inputs:            map[string]string{"metadata": `{"namespace":"meta-ns"}`},
			providerInputs:    map[string]string{"namespace": "provider-ns"},
			stackConfig:       map[string]string{"kubernetes:namespace": "stack-ns"},
			programConfig:     map[string]string{"kubernetes:namespace": "program-ns"},
			expectedNamespace: "meta-ns",
		},
		{
			name:              "provider_namespace_fallback",
			inputs:            map[string]string{"metadata": `{}`},
			providerInputs:    map[string]string{"namespace": "provider-ns"},
			stackConfig:       map[string]string{"kubernetes:namespace": "stack-ns"},
			programConfig:     map[string]string{"kubernetes:namespace": "program-ns"},
			expectedNamespace: "provider-ns",
		},
		{
			name:              "stack_namespace_fallback",
			inputs:            map[string]string{},
			providerInputs:    map[string]string{},
			stackConfig:       map[string]string{"kubernetes:namespace": "stack-ns"},
			programConfig:     map[string]string{"kubernetes:namespace": "program-ns"},
			expectedNamespace: "stack-ns",
		},
		{
			name:              "program_namespace_fallback",
			inputs:            map[string]string{},
			providerInputs:    map[string]string{},
			stackConfig:       map[string]string{},
			programConfig:     map[string]string{"kubernetes:namespace": "program-ns"},
			expectedNamespace: "program-ns",
		},
		{
			name:              "no_namespace",
			inputs:            map[string]string{},
			providerInputs:    map[string]string{},
			stackConfig:       map[string]string{},
			programConfig:     map[string]string{},
			expectedNamespace: "",
		},
	}

	ctx := context.Background()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := &plugin.OpenResourceRequest{
				ResourceType:   "kubernetes:apps/v1:Deployment",
				ResourceName:   "my-deployment",
				Inputs:         tc.inputs,
				ProviderInputs: tc.providerInputs,
				StackConfig:    tc.stackConfig,
				ProgramConfig:  tc.programConfig,
			}

			resp, err := p.OpenResource(ctx, req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !resp.CanOpen {
				t.Fatal("expected CanOpen=true")
			}

			// Check for --namespace in args
			foundNamespace := ""
			for i, arg := range resp.Action.Args {
				if arg == "--namespace" && i+1 < len(resp.Action.Args) {
					foundNamespace = resp.Action.Args[i+1]
					break
				}
			}

			if foundNamespace != tc.expectedNamespace {
				t.Errorf("expected namespace=%q, got %q in args: %v", tc.expectedNamespace, foundNamespace, resp.Action.Args)
			}
		})
	}
}

func TestK9sPlugin_OpenResource_WithKubeconfig(t *testing.T) {
	p := &K9sPlugin{
		BuiltinPluginBase: plugins.NewBuiltinPluginBase("k9s"),
	}

	tests := []struct {
		name             string
		providerInputs   map[string]string
		stackConfig      map[string]string
		programConfig    map[string]string
		expectKubeconfig bool
		expectTempFile   bool // If kubeconfig is content, it creates a temp file
	}{
		{
			name:             "kubeconfig_file_path",
			providerInputs:   map[string]string{"kubeconfig": "/home/user/.kube/config"},
			expectKubeconfig: true,
			expectTempFile:   false,
		},
		{
			name:             "kubeconfig_yaml_content",
			providerInputs:   map[string]string{"kubeconfig": "apiVersion: v1\nkind: Config"},
			expectKubeconfig: true,
			expectTempFile:   true,
		},
		{
			name:             "stack_config_kubeconfig",
			stackConfig:      map[string]string{"kubernetes:kubeconfig": "/etc/kubernetes/admin.conf"},
			expectKubeconfig: true,
			expectTempFile:   false,
		},
		{
			name:             "program_config_kubeconfig",
			programConfig:    map[string]string{"kubernetes:kubeconfig": "~/.kube/config"},
			expectKubeconfig: true,
			expectTempFile:   false,
		},
		{
			name:             "no_kubeconfig",
			expectKubeconfig: false,
		},
	}

	ctx := context.Background()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := &plugin.OpenResourceRequest{
				ResourceType:   "kubernetes:core/v1:Pod",
				ResourceName:   "my-pod",
				ProviderInputs: tc.providerInputs,
				StackConfig:    tc.stackConfig,
				ProgramConfig:  tc.programConfig,
			}

			resp, err := p.OpenResource(ctx, req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !resp.CanOpen {
				t.Fatal("expected CanOpen=true")
			}

			// Check for --kubeconfig in args
			foundKubeconfig := false
			for _, arg := range resp.Action.Args {
				if arg == "--kubeconfig" {
					foundKubeconfig = true
					break
				}
			}

			if foundKubeconfig != tc.expectKubeconfig {
				t.Errorf("expected kubeconfig=%v, found=%v in args: %v", tc.expectKubeconfig, foundKubeconfig, resp.Action.Args)
			}
		})
	}
}

func TestK9sPlugin_OpenResource_AuthEnvPassthrough(t *testing.T) {
	p := &K9sPlugin{
		BuiltinPluginBase: plugins.NewBuiltinPluginBase("k9s"),
	}

	ctx := context.Background()
	req := &plugin.OpenResourceRequest{
		ResourceType: "kubernetes:core/v1:Pod",
		ResourceName: "my-pod",
		AuthEnv: map[string]string{
			"KUBECONFIG":       "/custom/kubeconfig",
			"AWS_PROFILE":      "my-profile",
			"GOOGLE_AUTH_JSON": `{"type":"service_account"}`,
		},
	}

	resp, err := p.OpenResource(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.CanOpen {
		t.Fatal("expected CanOpen=true")
	}

	// Verify auth env is passed through
	if resp.Action.Env == nil {
		t.Fatal("expected Env to be set")
	}

	for key, expectedValue := range req.AuthEnv {
		if resp.Action.Env[key] != expectedValue {
			t.Errorf("expected Env[%q]=%q, got %q", key, expectedValue, resp.Action.Env[key])
		}
	}
}

func TestK9sPlugin_OpenResource_ResourceKinds(t *testing.T) {
	p := &K9sPlugin{
		BuiltinPluginBase: plugins.NewBuiltinPluginBase("k9s"),
	}

	tests := []struct {
		resourceType string
		expectedKind string
	}{
		{"kubernetes:core/v1:Pod", "pod"},
		{"kubernetes:apps/v1:Deployment", "deployment"},
		{"kubernetes:apps/v1:StatefulSet", "statefulset"},
		{"kubernetes:networking.k8s.io/v1:Ingress", "ingress"},
		{"kubernetes:batch/v1:CronJob", "cronjob"},
	}

	ctx := context.Background()

	for _, tc := range tests {
		t.Run(tc.resourceType, func(t *testing.T) {
			req := &plugin.OpenResourceRequest{
				ResourceType: tc.resourceType,
				ResourceName: "test-resource",
			}

			resp, err := p.OpenResource(ctx, req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !resp.CanOpen {
				t.Fatal("expected CanOpen=true")
			}

			// Check that --command <kind> is in args
			foundKind := ""
			for i, arg := range resp.Action.Args {
				if arg == "--command" && i+1 < len(resp.Action.Args) {
					foundKind = resp.Action.Args[i+1]
					break
				}
			}

			if foundKind != tc.expectedKind {
				t.Errorf("expected --command %q, got --command %q in args: %v", tc.expectedKind, foundKind, resp.Action.Args)
			}
		})
	}
}
