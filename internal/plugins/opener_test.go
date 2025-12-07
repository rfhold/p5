package plugins

import (
	"context"
	"testing"

	"github.com/rfhold/p5/internal/plugins/proto"
	"github.com/rfhold/p5/pkg/plugin"
)

func TestOpenNotSupported(t *testing.T) {
	resp := OpenNotSupported()
	if resp.CanOpen {
		t.Error("expected CanOpen=false")
	}
}

func TestOpenBrowserResponse(t *testing.T) {
	url := "https://console.example.com/resource/123"
	resp := OpenBrowserResponse(url)

	if !resp.CanOpen {
		t.Error("expected CanOpen=true")
	}
	if resp.Action == nil {
		t.Fatal("expected Action to be set")
	}
	if resp.Action.Type != proto.OpenActionType_OPEN_ACTION_TYPE_BROWSER {
		t.Errorf("expected Type=BROWSER, got %v", resp.Action.Type)
	}
	if resp.Action.Url != url {
		t.Errorf("expected Url=%q, got %q", url, resp.Action.Url)
	}
}

func TestOpenExecResponse(t *testing.T) {
	cmd := "kubectl"
	args := []string{"exec", "-it", "pod-name", "--", "bash"}
	env := map[string]string{"KUBECONFIG": "/custom/config"}

	resp := OpenExecResponse(cmd, args, env)

	if !resp.CanOpen {
		t.Error("expected CanOpen=true")
	}
	if resp.Action == nil {
		t.Fatal("expected Action to be set")
	}
	if resp.Action.Type != proto.OpenActionType_OPEN_ACTION_TYPE_EXEC {
		t.Errorf("expected Type=EXEC, got %v", resp.Action.Type)
	}
	if resp.Action.Command != cmd {
		t.Errorf("expected Command=%q, got %q", cmd, resp.Action.Command)
	}
	if len(resp.Action.Args) != len(args) {
		t.Errorf("expected %d args, got %d", len(args), len(resp.Action.Args))
	}
	for i, arg := range args {
		if resp.Action.Args[i] != arg {
			t.Errorf("expected Args[%d]=%q, got %q", i, arg, resp.Action.Args[i])
		}
	}
	if resp.Action.Env["KUBECONFIG"] != "/custom/config" {
		t.Errorf("expected Env[KUBECONFIG]=%q, got %q", "/custom/config", resp.Action.Env["KUBECONFIG"])
	}
}

func TestOpenError(t *testing.T) {
	resp := OpenError("failed to open: %s", "test error")

	if !resp.CanOpen {
		t.Error("expected CanOpen=true (error is reported, but plugin tried)")
	}
	if resp.Error == "" {
		t.Error("expected Error to be set")
	}
	if resp.Error != "failed to open: test error" {
		t.Errorf("expected Error=%q, got %q", "failed to open: test error", resp.Error)
	}
}

func TestSupportedOpenTypesPatterns(t *testing.T) {
	patterns := []string{`^kubernetes:.*`, `^aws:ec2/.*`}
	resp := SupportedOpenTypesPatterns(patterns...)

	if len(resp.ResourceTypePatterns) != len(patterns) {
		t.Errorf("expected %d patterns, got %d", len(patterns), len(resp.ResourceTypePatterns))
	}
	for i, pattern := range patterns {
		if resp.ResourceTypePatterns[i] != pattern {
			t.Errorf("expected pattern[%d]=%q, got %q", i, pattern, resp.ResourceTypePatterns[i])
		}
	}
}

func TestManager_OpenResource_NoPlugins(t *testing.T) {
	mgr, _ := NewManager("")
	mgr.mergedConfig = &P5Config{Plugins: make(map[string]PluginConfig)}

	ctx := context.Background()
	req := &OpenResourceRequest{
		ResourceType: "kubernetes:core/v1:Pod",
		ResourceName: "my-pod",
	}

	resp, pluginName, err := mgr.OpenResource(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != nil {
		t.Error("expected nil response when no plugins")
	}
	if pluginName != "" {
		t.Errorf("expected empty plugin name, got %q", pluginName)
	}
}

func TestManager_HasResourceOpeners_NoPlugins(t *testing.T) {
	mgr, _ := NewManager("")

	if mgr.HasResourceOpeners() {
		t.Error("expected HasResourceOpeners=false when no plugins")
	}
}

func TestFakePluginProvider_OpenResource_Default(t *testing.T) {
	fake := &FakePluginProvider{
		OpenResourceResponse: plugin.OpenExecResponse("k9s", []string{"--command", "pod"}, nil),
		OpenResourcePlugin:   "k9s",
	}

	ctx := context.Background()
	req := &OpenResourceRequest{
		ResourceType: "kubernetes:core/v1:Pod",
		ResourceName: "my-pod",
	}

	resp, pluginName, err := fake.OpenResource(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.CanOpen {
		t.Error("expected CanOpen=true")
	}
	if pluginName != "k9s" {
		t.Errorf("expected pluginName=%q, got %q", "k9s", pluginName)
	}
	if len(fake.Calls.OpenResource) != 1 {
		t.Errorf("expected 1 call, got %d", len(fake.Calls.OpenResource))
	}
}

func TestFakePluginProvider_OpenResource_CustomFunc(t *testing.T) {
	var capturedReq *OpenResourceRequest
	fake := &FakePluginProvider{
		OpenResourceFunc: func(ctx context.Context, req *OpenResourceRequest) (*OpenResourceResponse, string, error) {
			capturedReq = req
			if req.ResourceType == "kubernetes:core/v1:Pod" {
				return plugin.OpenExecResponse("k9s", []string{"--command", "pod"}, nil), "k9s", nil
			}
			return plugin.OpenNotSupported(), "", nil
		},
	}

	ctx := context.Background()
	req := &OpenResourceRequest{
		ResourceType: "kubernetes:core/v1:Pod",
		ResourceName: "my-pod",
		Inputs:       map[string]string{"metadata": `{"namespace":"default"}`},
	}

	resp, pluginName, err := fake.OpenResource(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.CanOpen {
		t.Error("expected CanOpen=true")
	}
	if pluginName != "k9s" {
		t.Errorf("expected pluginName=%q, got %q", "k9s", pluginName)
	}
	if capturedReq == nil || capturedReq.ResourceType != "kubernetes:core/v1:Pod" {
		t.Error("expected request to be captured")
	}
}

func TestFakePluginProvider_HasResourceOpeners(t *testing.T) {
	tests := []struct {
		name     string
		value    bool
		expected bool
	}{
		{"has_openers", true, true},
		{"no_openers", false, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fake := &FakePluginProvider{
				HasResourceOpener: tc.value,
			}

			result := fake.HasResourceOpeners()
			if result != tc.expected {
				t.Errorf("expected HasResourceOpeners=%v, got %v", tc.expected, result)
			}
			if fake.Calls.HasResourceOpeners != 1 {
				t.Errorf("expected 1 call, got %d", fake.Calls.HasResourceOpeners)
			}
		})
	}
}

func TestOpenActionType_Browser(t *testing.T) {
	if proto.OpenActionType_OPEN_ACTION_TYPE_BROWSER != 1 {
		t.Error("expected BROWSER type to be 1")
	}
}

func TestOpenActionType_Exec(t *testing.T) {
	if proto.OpenActionType_OPEN_ACTION_TYPE_EXEC != 2 {
		t.Error("expected EXEC type to be 2")
	}
}

func TestOpenActionType_Unspecified(t *testing.T) {
	if proto.OpenActionType_OPEN_ACTION_TYPE_UNSPECIFIED != 0 {
		t.Error("expected UNSPECIFIED type to be 0")
	}
}
