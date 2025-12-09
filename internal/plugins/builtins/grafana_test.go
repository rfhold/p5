package builtins

import (
	"context"
	"slices"
	"testing"

	"github.com/rfhold/p5/internal/plugins"
	"github.com/rfhold/p5/internal/plugins/proto"
	"github.com/rfhold/p5/pkg/plugin"
)

func TestGrafanaPlugin_Name(t *testing.T) {
	p := &GrafanaPlugin{
		BuiltinPluginBase: plugins.NewBuiltinPluginBase("grafana"),
	}

	if p.Name() != "grafana" {
		t.Errorf("expected Name=%q, got %q", "grafana", p.Name())
	}
}

func TestGrafanaPlugin_Authenticate(t *testing.T) {
	p := &GrafanaPlugin{
		BuiltinPluginBase: plugins.NewBuiltinPluginBase("grafana"),
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

func TestGrafanaPlugin_GetSupportedOpenTypes(t *testing.T) {
	p := &GrafanaPlugin{
		BuiltinPluginBase: plugins.NewBuiltinPluginBase("grafana"),
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

	if !slices.Contains(resp.ResourceTypePatterns, `^grafana:onCall/.*`) {
		t.Errorf("expected pattern ^grafana:onCall/.* in %v", resp.ResourceTypePatterns)
	}

	if !slices.Contains(resp.ResourceTypePatterns, `^grafana:oss/team:Team$`) {
		t.Errorf("expected pattern ^grafana:oss/team:Team$ in %v", resp.ResourceTypePatterns)
	}

	if !slices.Contains(resp.ResourceTypePatterns, `^grafana:alerting/ruleGroup:RuleGroup$`) {
		t.Errorf("expected pattern ^grafana:alerting/ruleGroup:RuleGroup$ in %v", resp.ResourceTypePatterns)
	}

	if !slices.Contains(resp.ResourceTypePatterns, `^grafana:oss/dashboard:Dashboard$`) {
		t.Errorf("expected pattern ^grafana:oss/dashboard:Dashboard$ in %v", resp.ResourceTypePatterns)
	}

	if !slices.Contains(resp.ResourceTypePatterns, `^grafana:alerting/contactPoint:ContactPoint$`) {
		t.Errorf("expected pattern ^grafana:alerting/contactPoint:ContactPoint$ in %v", resp.ResourceTypePatterns)
	}

	if !slices.Contains(resp.ResourceTypePatterns, `^grafana:alerting/muteTiming:MuteTiming$`) {
		t.Errorf("expected pattern ^grafana:alerting/muteTiming:MuteTiming$ in %v", resp.ResourceTypePatterns)
	}

	if !slices.Contains(resp.ResourceTypePatterns, `^grafana:alerting/notificationPolicy:NotificationPolicy$`) {
		t.Errorf("expected pattern ^grafana:alerting/notificationPolicy:NotificationPolicy$ in %v", resp.ResourceTypePatterns)
	}

	if !slices.Contains(resp.ResourceTypePatterns, `^grafana:cloud/accessPolicy:AccessPolicy$`) {
		t.Errorf("expected pattern ^grafana:cloud/accessPolicy:AccessPolicy$ in %v", resp.ResourceTypePatterns)
	}

	if !slices.Contains(resp.ResourceTypePatterns, `^grafana:cloud/accessPolicyToken:AccessPolicyToken$`) {
		t.Errorf("expected pattern ^grafana:cloud/accessPolicyToken:AccessPolicyToken$ in %v", resp.ResourceTypePatterns)
	}

	if !slices.Contains(resp.ResourceTypePatterns, `^grafana:cloud/stackServiceAccount:StackServiceAccount$`) {
		t.Errorf("expected pattern ^grafana:cloud/stackServiceAccount:StackServiceAccount$ in %v", resp.ResourceTypePatterns)
	}

	if !slices.Contains(resp.ResourceTypePatterns, `^grafana:cloud/stackServiceAccountToken:StackServiceAccountToken$`) {
		t.Errorf("expected pattern ^grafana:cloud/stackServiceAccountToken:StackServiceAccountToken$ in %v", resp.ResourceTypePatterns)
	}
}

func TestGrafanaPlugin_OpenResource_EscalationChain(t *testing.T) {
	p := &GrafanaPlugin{
		BuiltinPluginBase: plugins.NewBuiltinPluginBase("grafana"),
	}

	ctx := context.Background()
	req := &plugin.OpenResourceRequest{
		ResourceType:   "grafana:onCall/escalationChain:EscalationChain",
		ResourceName:   "my-escalation-chain",
		ProviderInputs: map[string]string{"url": "https://example.grafana.net"},
		Outputs:        map[string]string{"id": "FBWUTTQDSMZCM"},
	}

	resp, err := p.OpenResource(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.CanOpen {
		t.Errorf("expected CanOpen=true, got error: %s", resp.Error)
	}
	if resp.Action == nil {
		t.Fatal("expected Action to be set")
	}
	if resp.Action.Type != proto.OpenActionType_OPEN_ACTION_TYPE_BROWSER {
		t.Errorf("expected browser action, got %v", resp.Action.Type)
	}

	expected := "https://example.grafana.net/a/grafana-irm-app/escalations/FBWUTTQDSMZCM"
	if resp.Action.Url != expected {
		t.Errorf("expected URL=%q, got %q", expected, resp.Action.Url)
	}
}

func TestGrafanaPlugin_OpenResource_Escalation(t *testing.T) {
	p := &GrafanaPlugin{
		BuiltinPluginBase: plugins.NewBuiltinPluginBase("grafana"),
	}

	ctx := context.Background()
	req := &plugin.OpenResourceRequest{
		ResourceType:   "grafana:onCall/escalation:Escalation",
		ResourceName:   "my-escalation",
		ProviderInputs: map[string]string{"url": "https://example.grafana.net"},
		Inputs:         map[string]string{"escalationChainId": "FBWUTTQDSMZCM"},
	}

	resp, err := p.OpenResource(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.CanOpen {
		t.Errorf("expected CanOpen=true, got error: %s", resp.Error)
	}

	expected := "https://example.grafana.net/a/grafana-irm-app/escalations/FBWUTTQDSMZCM"
	if resp.Action.Url != expected {
		t.Errorf("expected URL=%q, got %q", expected, resp.Action.Url)
	}
}

func TestGrafanaPlugin_OpenResource_Integration(t *testing.T) {
	p := &GrafanaPlugin{
		BuiltinPluginBase: plugins.NewBuiltinPluginBase("grafana"),
	}

	ctx := context.Background()
	req := &plugin.OpenResourceRequest{
		ResourceType:   "grafana:onCall/integration:Integration",
		ResourceName:   "my-integration",
		ProviderInputs: map[string]string{"url": "https://example.grafana.net"},
		Outputs:        map[string]string{"id": "C9FCZIDCSTFAB"},
	}

	resp, err := p.OpenResource(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.CanOpen {
		t.Errorf("expected CanOpen=true, got error: %s", resp.Error)
	}

	expected := "https://example.grafana.net/a/grafana-irm-app/integrations/C9FCZIDCSTFAB"
	if resp.Action.Url != expected {
		t.Errorf("expected URL=%q, got %q", expected, resp.Action.Url)
	}
}

func TestGrafanaPlugin_OpenResource_OnCallShift(t *testing.T) {
	p := &GrafanaPlugin{
		BuiltinPluginBase: plugins.NewBuiltinPluginBase("grafana"),
	}

	ctx := context.Background()
	req := &plugin.OpenResourceRequest{
		ResourceType:   "grafana:onCall/onCallShift:OnCallShift",
		ResourceName:   "my-shift",
		ProviderInputs: map[string]string{"url": "https://example.grafana.net"},
	}

	resp, err := p.OpenResource(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.CanOpen {
		t.Errorf("expected CanOpen=true, got error: %s", resp.Error)
	}

	expected := "https://example.grafana.net/a/grafana-irm-app/schedules"
	if resp.Action.Url != expected {
		t.Errorf("expected URL=%q, got %q", expected, resp.Action.Url)
	}
}

func TestGrafanaPlugin_OpenResource_Route(t *testing.T) {
	p := &GrafanaPlugin{
		BuiltinPluginBase: plugins.NewBuiltinPluginBase("grafana"),
	}

	ctx := context.Background()
	req := &plugin.OpenResourceRequest{
		ResourceType:   "grafana:onCall/route:Route",
		ResourceName:   "my-route",
		ProviderInputs: map[string]string{"url": "https://example.grafana.net"},
		Inputs:         map[string]string{"integrationId": "C2A9G1V9UE92C"},
	}

	resp, err := p.OpenResource(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.CanOpen {
		t.Errorf("expected CanOpen=true, got error: %s", resp.Error)
	}

	expected := "https://example.grafana.net/a/grafana-irm-app/integrations/C2A9G1V9UE92C"
	if resp.Action.Url != expected {
		t.Errorf("expected URL=%q, got %q", expected, resp.Action.Url)
	}
}

func TestGrafanaPlugin_OpenResource_Schedule(t *testing.T) {
	p := &GrafanaPlugin{
		BuiltinPluginBase: plugins.NewBuiltinPluginBase("grafana"),
	}

	ctx := context.Background()
	req := &plugin.OpenResourceRequest{
		ResourceType:   "grafana:onCall/schedule:Schedule",
		ResourceName:   "my-schedule",
		ProviderInputs: map[string]string{"url": "https://example.grafana.net"},
		Outputs:        map[string]string{"id": "S1PCAFH2AAWNA"},
	}

	resp, err := p.OpenResource(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.CanOpen {
		t.Errorf("expected CanOpen=true, got error: %s", resp.Error)
	}

	expected := "https://example.grafana.net/a/grafana-irm-app/schedules/S1PCAFH2AAWNA"
	if resp.Action.Url != expected {
		t.Errorf("expected URL=%q, got %q", expected, resp.Action.Url)
	}
}

func TestGrafanaPlugin_OpenResource_Team(t *testing.T) {
	p := &GrafanaPlugin{
		BuiltinPluginBase: plugins.NewBuiltinPluginBase("grafana"),
	}

	ctx := context.Background()
	req := &plugin.OpenResourceRequest{
		ResourceType:   "grafana:oss/team:Team",
		ResourceName:   "my-team",
		ProviderInputs: map[string]string{"url": "https://example.grafana.net"},
		Outputs:        map[string]string{"teamUid": "eew3dbktwt7nkd"},
	}

	resp, err := p.OpenResource(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.CanOpen {
		t.Errorf("expected CanOpen=true, got error: %s", resp.Error)
	}

	expected := "https://example.grafana.net/org/teams/edit/eew3dbktwt7nkd"
	if resp.Action.Url != expected {
		t.Errorf("expected URL=%q, got %q", expected, resp.Action.Url)
	}
}

func TestGrafanaPlugin_OpenResource_RuleGroup(t *testing.T) {
	p := &GrafanaPlugin{
		BuiltinPluginBase: plugins.NewBuiltinPluginBase("grafana"),
	}

	ctx := context.Background()
	req := &plugin.OpenResourceRequest{
		ResourceType:   "grafana:alerting/ruleGroup:RuleGroup",
		ResourceName:   "default",
		ProviderInputs: map[string]string{"url": "https://example.grafana.net"},
		Inputs: map[string]string{
			"folderUid": "my-alerts-folder",
			"name":      "my-alert-rule-group",
		},
	}

	resp, err := p.OpenResource(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.CanOpen {
		t.Errorf("expected CanOpen=true, got error: %s", resp.Error)
	}

	expected := "https://example.grafana.net/alerting/grafana/namespaces/my-alerts-folder/groups/my-alert-rule-group/view"
	if resp.Action.Url != expected {
		t.Errorf("expected URL=%q, got %q", expected, resp.Action.Url)
	}
}

func TestGrafanaPlugin_OpenResource_RuleGroup_MissingFolder(t *testing.T) {
	p := &GrafanaPlugin{
		BuiltinPluginBase: plugins.NewBuiltinPluginBase("grafana"),
	}

	ctx := context.Background()
	req := &plugin.OpenResourceRequest{
		ResourceType:   "grafana:alerting/ruleGroup:RuleGroup",
		ResourceName:   "default",
		ProviderInputs: map[string]string{"url": "https://example.grafana.net"},
		Inputs: map[string]string{
			"name": "my-alert-rule-group",
		},
	}

	resp, err := p.OpenResource(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Error == "" {
		t.Error("expected error message when folderUid is missing")
	}
}

func TestGrafanaPlugin_OpenResource_RuleGroup_MissingName(t *testing.T) {
	p := &GrafanaPlugin{
		BuiltinPluginBase: plugins.NewBuiltinPluginBase("grafana"),
	}

	ctx := context.Background()
	req := &plugin.OpenResourceRequest{
		ResourceType:   "grafana:alerting/ruleGroup:RuleGroup",
		ResourceName:   "default",
		ProviderInputs: map[string]string{"url": "https://example.grafana.net"},
		Inputs: map[string]string{
			"folderUid": "my-alerts-folder",
		},
	}

	resp, err := p.OpenResource(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Error == "" {
		t.Error("expected error message when name is missing")
	}
}

func TestGrafanaPlugin_OpenResource_NotSupported(t *testing.T) {
	p := &GrafanaPlugin{
		BuiltinPluginBase: plugins.NewBuiltinPluginBase("grafana"),
	}

	tests := []struct {
		name         string
		resourceType string
	}{
		{"aws_resource", "aws:ec2/instance:Instance"},
		{"kubernetes_resource", "kubernetes:core/v1:Pod"},
		{"empty_type", ""},
		{"unknown_grafana_type", "grafana:unknown/type:Unknown"},
	}

	ctx := context.Background()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := &plugin.OpenResourceRequest{
				ResourceType:   tc.resourceType,
				ResourceName:   "test",
				ProviderInputs: map[string]string{"url": "https://example.grafana.net"},
			}

			resp, err := p.OpenResource(ctx, req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.CanOpen && resp.Error == "" {
				t.Errorf("expected CanOpen=false for %q", tc.resourceType)
			}
		})
	}
}

func TestGrafanaPlugin_OpenResource_URLPriority(t *testing.T) {
	p := &GrafanaPlugin{
		BuiltinPluginBase: plugins.NewBuiltinPluginBase("grafana"),
	}

	tests := []struct {
		name           string
		providerInputs map[string]string
		stackConfig    map[string]string
		programConfig  map[string]string
		expectedURL    string
	}{
		{
			name:           "provider_wins",
			providerInputs: map[string]string{"url": "https://provider.grafana.net"},
			stackConfig:    map[string]string{"grafana:url": "https://stack.grafana.net"},
			programConfig:  map[string]string{"grafana:url": "https://program.grafana.net"},
			expectedURL:    "https://provider.grafana.net/a/grafana-irm-app/schedules/TEST123",
		},
		{
			name:           "stack_fallback",
			providerInputs: map[string]string{},
			stackConfig:    map[string]string{"grafana:url": "https://stack.grafana.net"},
			programConfig:  map[string]string{"grafana:url": "https://program.grafana.net"},
			expectedURL:    "https://stack.grafana.net/a/grafana-irm-app/schedules/TEST123",
		},
		{
			name:           "program_fallback",
			providerInputs: map[string]string{},
			stackConfig:    map[string]string{},
			programConfig:  map[string]string{"grafana:url": "https://program.grafana.net"},
			expectedURL:    "https://program.grafana.net/a/grafana-irm-app/schedules/TEST123",
		},
	}

	ctx := context.Background()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := &plugin.OpenResourceRequest{
				ResourceType:   "grafana:onCall/schedule:Schedule",
				ResourceName:   "my-schedule",
				ProviderInputs: tc.providerInputs,
				StackConfig:    tc.stackConfig,
				ProgramConfig:  tc.programConfig,
				Outputs:        map[string]string{"id": "TEST123"},
			}

			resp, err := p.OpenResource(ctx, req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !resp.CanOpen {
				t.Errorf("expected CanOpen=true, got error: %s", resp.Error)
			}

			if resp.Action.Url != tc.expectedURL {
				t.Errorf("expected URL=%q, got %q", tc.expectedURL, resp.Action.Url)
			}
		})
	}
}

func TestGrafanaPlugin_OpenResource_MissingConfig(t *testing.T) {
	p := &GrafanaPlugin{
		BuiltinPluginBase: plugins.NewBuiltinPluginBase("grafana"),
	}

	tests := []struct {
		name           string
		resourceType   string
		providerInputs map[string]string
		expectedError  string
	}{
		{
			name:           "missing_grafana_url_for_oncall",
			resourceType:   "grafana:onCall/schedule:Schedule",
			providerInputs: map[string]string{},
			expectedError:  "grafana url not configured",
		},
		{
			name:           "missing_grafana_url_for_team",
			resourceType:   "grafana:oss/team:Team",
			providerInputs: map[string]string{},
			expectedError:  "grafana url not configured",
		},
	}

	ctx := context.Background()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := &plugin.OpenResourceRequest{
				ResourceType:   tc.resourceType,
				ResourceName:   "test",
				ProviderInputs: tc.providerInputs,
				Outputs:        map[string]string{"id": "TEST123", "teamUid": "test-uid"},
			}

			resp, err := p.OpenResource(ctx, req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if resp.Error == "" {
				t.Errorf("expected error %q, got none", tc.expectedError)
			}
		})
	}
}

func TestGrafanaPlugin_OpenResource_ContactPoint(t *testing.T) {
	p := &GrafanaPlugin{
		BuiltinPluginBase: plugins.NewBuiltinPluginBase("grafana"),
	}

	ctx := context.Background()
	req := &plugin.OpenResourceRequest{
		ResourceType:   "grafana:alerting/contactPoint:ContactPoint",
		ResourceName:   "default",
		ProviderInputs: map[string]string{"url": "https://example.grafana.net"},
		Inputs:         map[string]string{"name": "OnCall"},
	}

	resp, err := p.OpenResource(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.CanOpen {
		t.Errorf("expected CanOpen=true, got error: %s", resp.Error)
	}

	expected := "https://example.grafana.net/alerting/notifications"
	if resp.Action.Url != expected {
		t.Errorf("expected URL=%q, got %q", expected, resp.Action.Url)
	}
}

func TestGrafanaPlugin_OpenResource_MuteTiming(t *testing.T) {
	p := &GrafanaPlugin{
		BuiltinPluginBase: plugins.NewBuiltinPluginBase("grafana"),
	}

	ctx := context.Background()
	req := &plugin.OpenResourceRequest{
		ResourceType:   "grafana:alerting/muteTiming:MuteTiming",
		ResourceName:   "sunday-blackout",
		ProviderInputs: map[string]string{"url": "https://example.grafana.net"},
		Inputs:         map[string]string{"name": "Sunday Alert Blackout"},
	}

	resp, err := p.OpenResource(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.CanOpen {
		t.Errorf("expected CanOpen=true, got error: %s", resp.Error)
	}

	expected := "https://example.grafana.net/alerting/routes?tab=time_intervals"
	if resp.Action.Url != expected {
		t.Errorf("expected URL=%q, got %q", expected, resp.Action.Url)
	}
}

func TestGrafanaPlugin_OpenResource_NotificationPolicy(t *testing.T) {
	p := &GrafanaPlugin{
		BuiltinPluginBase: plugins.NewBuiltinPluginBase("grafana"),
	}

	ctx := context.Background()
	req := &plugin.OpenResourceRequest{
		ResourceType:   "grafana:alerting/notificationPolicy:NotificationPolicy",
		ResourceName:   "default",
		ProviderInputs: map[string]string{"url": "https://example.grafana.net"},
		Inputs:         map[string]string{"contactPoint": "OnCall"},
	}

	resp, err := p.OpenResource(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.CanOpen {
		t.Errorf("expected CanOpen=true, got error: %s", resp.Error)
	}

	expected := "https://example.grafana.net/alerting/routes"
	if resp.Action.Url != expected {
		t.Errorf("expected URL=%q, got %q", expected, resp.Action.Url)
	}
}

func TestGrafanaPlugin_OpenResource_AccessPolicy(t *testing.T) {
	p := &GrafanaPlugin{
		BuiltinPluginBase: plugins.NewBuiltinPluginBase("grafana"),
	}

	ctx := context.Background()
	req := &plugin.OpenResourceRequest{
		ResourceType:   "grafana:cloud/accessPolicy:AccessPolicy",
		ResourceName:   "cicd",
		ProviderInputs: map[string]string{},
		StackConfig:    map[string]string{"grafana:cloudOrgSlug": "myorg"},
		Inputs:         map[string]string{"name": "cicd", "region": "prod-us-east-0"},
		Outputs:        map[string]string{"policyId": "fd4b4560-8648-4039-9b07-aafd16842dc9"},
	}

	resp, err := p.OpenResource(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.CanOpen {
		t.Errorf("expected CanOpen=true, got error: %s", resp.Error)
	}

	expected := "https://grafana.com/orgs/myorg/access-policies"
	if resp.Action.Url != expected {
		t.Errorf("expected URL=%q, got %q", expected, resp.Action.Url)
	}
}

func TestGrafanaPlugin_OpenResource_AccessPolicy_NoOrgSlug(t *testing.T) {
	p := &GrafanaPlugin{
		BuiltinPluginBase: plugins.NewBuiltinPluginBase("grafana"),
	}

	ctx := context.Background()
	req := &plugin.OpenResourceRequest{
		ResourceType:   "grafana:cloud/accessPolicy:AccessPolicy",
		ResourceName:   "cicd",
		ProviderInputs: map[string]string{},
		Inputs:         map[string]string{"name": "cicd", "region": "prod-us-east-0"},
		Outputs:        map[string]string{"policyId": "fd4b4560-8648-4039-9b07-aafd16842dc9"},
	}

	resp, err := p.OpenResource(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.CanOpen {
		t.Errorf("expected CanOpen=true, got error: %s", resp.Error)
	}

	expected := "https://grafana.com/orgs/access-policies"
	if resp.Action.Url != expected {
		t.Errorf("expected URL=%q, got %q", expected, resp.Action.Url)
	}
}

func TestGrafanaPlugin_OpenResource_AccessPolicyToken(t *testing.T) {
	p := &GrafanaPlugin{
		BuiltinPluginBase: plugins.NewBuiltinPluginBase("grafana"),
	}

	ctx := context.Background()
	req := &plugin.OpenResourceRequest{
		ResourceType:   "grafana:cloud/accessPolicyToken:AccessPolicyToken",
		ResourceName:   "cicd-2025-12-01",
		ProviderInputs: map[string]string{},
		ProgramConfig:  map[string]string{"grafana:cloudOrgSlug": "myorg"},
		Inputs:         map[string]string{"accessPolicyId": "fd4b4560-8648-4039-9b07-aafd16842dc9", "region": "prod-us-east-0"},
	}

	resp, err := p.OpenResource(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.CanOpen {
		t.Errorf("expected CanOpen=true, got error: %s", resp.Error)
	}

	expected := "https://grafana.com/orgs/myorg/access-policies"
	if resp.Action.Url != expected {
		t.Errorf("expected URL=%q, got %q", expected, resp.Action.Url)
	}
}

func TestGrafanaPlugin_OpenResource_StackServiceAccount(t *testing.T) {
	p := &GrafanaPlugin{
		BuiltinPluginBase: plugins.NewBuiltinPluginBase("grafana"),
	}

	ctx := context.Background()
	req := &plugin.OpenResourceRequest{
		ResourceType:   "grafana:cloud/stackServiceAccount:StackServiceAccount",
		ResourceName:   "cicd",
		ProviderInputs: map[string]string{"url": "https://example.grafana.net"},
		Inputs:         map[string]string{"name": "my-service-account", "stackSlug": "example"},
		Outputs:        map[string]string{"id": "example:18"},
	}

	resp, err := p.OpenResource(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.CanOpen {
		t.Errorf("expected CanOpen=true, got error: %s", resp.Error)
	}

	expected := "https://example.grafana.net/org/serviceaccounts/18"
	if resp.Action.Url != expected {
		t.Errorf("expected URL=%q, got %q", expected, resp.Action.Url)
	}
}

func TestGrafanaPlugin_OpenResource_StackServiceAccountToken(t *testing.T) {
	p := &GrafanaPlugin{
		BuiltinPluginBase: plugins.NewBuiltinPluginBase("grafana"),
	}

	ctx := context.Background()
	req := &plugin.OpenResourceRequest{
		ResourceType:   "grafana:cloud/stackServiceAccountToken:StackServiceAccountToken",
		ResourceName:   "my-token",
		ProviderInputs: map[string]string{"url": "https://example.grafana.net"},
		Inputs:         map[string]string{"serviceAccountId": "example:18", "stackSlug": "example"},
	}

	resp, err := p.OpenResource(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.CanOpen {
		t.Errorf("expected CanOpen=true, got error: %s", resp.Error)
	}

	expected := "https://example.grafana.net/org/serviceaccounts/18"
	if resp.Action.Url != expected {
		t.Errorf("expected URL=%q, got %q", expected, resp.Action.Url)
	}
}

func TestGrafanaPlugin_OpenResource_StackServiceAccountToken_NoServiceAccountId(t *testing.T) {
	p := &GrafanaPlugin{
		BuiltinPluginBase: plugins.NewBuiltinPluginBase("grafana"),
	}

	ctx := context.Background()
	req := &plugin.OpenResourceRequest{
		ResourceType:   "grafana:cloud/stackServiceAccountToken:StackServiceAccountToken",
		ResourceName:   "my-token",
		ProviderInputs: map[string]string{"url": "https://example.grafana.net"},
		Inputs:         map[string]string{"stackSlug": "example"},
	}

	resp, err := p.OpenResource(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.CanOpen {
		t.Errorf("expected CanOpen=true, got error: %s", resp.Error)
	}

	expected := "https://example.grafana.net/org/serviceaccounts"
	if resp.Action.Url != expected {
		t.Errorf("expected URL=%q, got %q", expected, resp.Action.Url)
	}
}

func TestGrafanaPlugin_OpenResource_Dashboard(t *testing.T) {
	p := &GrafanaPlugin{
		BuiltinPluginBase: plugins.NewBuiltinPluginBase("grafana"),
	}

	ctx := context.Background()
	req := &plugin.OpenResourceRequest{
		ResourceType:   "grafana:oss/dashboard:Dashboard",
		ResourceName:   "my-dashboard",
		ProviderInputs: map[string]string{"url": "https://example.grafana.net"},
		Inputs:         map[string]string{"folder": "my-folder"},
		Outputs:        map[string]string{"url": "https://example.grafana.net/d/my-dashboard/my-dashboard-title", "uid": "my-dashboard"},
	}

	resp, err := p.OpenResource(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.CanOpen {
		t.Errorf("expected CanOpen=true, got error: %s", resp.Error)
	}

	expected := "https://example.grafana.net/d/my-dashboard/my-dashboard-title"
	if resp.Action.Url != expected {
		t.Errorf("expected URL=%q, got %q", expected, resp.Action.Url)
	}
}

func TestGrafanaPlugin_OpenResource_Dashboard_MissingURL(t *testing.T) {
	p := &GrafanaPlugin{
		BuiltinPluginBase: plugins.NewBuiltinPluginBase("grafana"),
	}

	ctx := context.Background()
	req := &plugin.OpenResourceRequest{
		ResourceType:   "grafana:oss/dashboard:Dashboard",
		ResourceName:   "my-dashboard",
		ProviderInputs: map[string]string{"url": "https://example.grafana.net"},
		Outputs:        map[string]string{"uid": "my-dashboard"},
	}

	resp, err := p.OpenResource(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Error == "" {
		t.Error("expected error message when url is missing from outputs")
	}
}

func TestGrafanaPlugin_OpenResource_TrailingSlashRemoval(t *testing.T) {
	p := &GrafanaPlugin{
		BuiltinPluginBase: plugins.NewBuiltinPluginBase("grafana"),
	}

	ctx := context.Background()
	req := &plugin.OpenResourceRequest{
		ResourceType:   "grafana:onCall/schedule:Schedule",
		ResourceName:   "my-schedule",
		ProviderInputs: map[string]string{"url": "https://example.grafana.net/"},
		Outputs:        map[string]string{"id": "TEST123"},
	}

	resp, err := p.OpenResource(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.CanOpen {
		t.Errorf("expected CanOpen=true, got error: %s", resp.Error)
	}

	expected := "https://example.grafana.net/a/grafana-irm-app/schedules/TEST123"
	if resp.Action.Url != expected {
		t.Errorf("expected URL=%q (trailing slash removed), got %q", expected, resp.Action.Url)
	}
}
