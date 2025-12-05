package plugins

import (
	"testing"
)

// =============================================================================
// ShouldRefreshCredentials Tests
// =============================================================================

// Note: These tests use a Manager directly to test the ShouldRefreshCredentials
// method. We can't easily extract the pure decision logic without significant
// refactoring since it's tightly coupled to the Manager's internal state.

// TestShouldRefreshCredentials_NoCurrentContext verifies refresh when no context exists.
func TestShouldRefreshCredentials_NoCurrentContext(t *testing.T) {
	m := &Manager{
		credentials:    make(map[string]*Credentials),
		currentContext: nil, // No current context
		mergedConfig:   nil,
	}

	// When there's no current context, should always refresh
	should := m.ShouldRefreshCredentials("aws", "/new/work/dir", "new-stack", "my-program", nil, nil)

	if !should {
		t.Error("expected ShouldRefreshCredentials=true when no current context")
	}
}

// TestShouldRefreshCredentials_WorkspaceChanged verifies refresh on workspace change.
func TestShouldRefreshCredentials_WorkspaceChanged(t *testing.T) {
	m := &Manager{
		credentials: make(map[string]*Credentials),
		currentContext: &AuthContext{
			WorkDir:      "/old/work/dir",
			StackName:    "dev",
			ConfigHashes: make(map[string]string),
		},
		mergedConfig: nil, // Default refresh triggers
	}

	// Workspace changed
	should := m.ShouldRefreshCredentials("aws", "/new/work/dir", "dev", "my-program", nil, nil)

	if !should {
		t.Error("expected ShouldRefreshCredentials=true when workspace changed (default trigger)")
	}
}

// TestShouldRefreshCredentials_StackChanged verifies refresh on stack change.
func TestShouldRefreshCredentials_StackChanged(t *testing.T) {
	m := &Manager{
		credentials: make(map[string]*Credentials),
		currentContext: &AuthContext{
			WorkDir:      "/work/dir",
			StackName:    "dev",
			ConfigHashes: make(map[string]string),
		},
		mergedConfig: nil, // Default refresh triggers
	}

	// Stack changed
	should := m.ShouldRefreshCredentials("aws", "/work/dir", "prod", "my-program", nil, nil)

	if !should {
		t.Error("expected ShouldRefreshCredentials=true when stack changed (default trigger)")
	}
}

// TestShouldRefreshCredentials_NoChange verifies no refresh when nothing changed.
func TestShouldRefreshCredentials_NoChange(t *testing.T) {
	m := &Manager{
		credentials: make(map[string]*Credentials),
		currentContext: &AuthContext{
			WorkDir:      "/work/dir",
			StackName:    "dev",
			ConfigHashes: make(map[string]string),
		},
		mergedConfig: nil,
	}

	// Same workspace and stack
	should := m.ShouldRefreshCredentials("aws", "/work/dir", "dev", "my-program", nil, nil)

	if should {
		t.Error("expected ShouldRefreshCredentials=false when nothing changed")
	}
}

// TestShouldRefreshCredentials_WorkspaceChangeDisabled verifies no refresh when trigger disabled.
func TestShouldRefreshCredentials_WorkspaceChangeDisabled(t *testing.T) {
	falseVal := false
	m := &Manager{
		credentials: make(map[string]*Credentials),
		currentContext: &AuthContext{
			WorkDir:      "/old/work/dir",
			StackName:    "dev",
			ConfigHashes: make(map[string]string),
		},
		mergedConfig: &P5Config{
			Plugins: map[string]PluginConfig{
				"aws": {
					Refresh: &RefreshTrigger{
						OnWorkspaceChange: &falseVal, // Disabled
					},
				},
			},
		},
	}

	// Workspace changed but trigger disabled
	should := m.ShouldRefreshCredentials("aws", "/new/work/dir", "dev", "my-program", nil, nil)

	if should {
		t.Error("expected ShouldRefreshCredentials=false when workspace trigger disabled")
	}
}

// TestShouldRefreshCredentials_StackChangeDisabled verifies no refresh when stack trigger disabled.
func TestShouldRefreshCredentials_StackChangeDisabled(t *testing.T) {
	falseVal := false
	m := &Manager{
		credentials: make(map[string]*Credentials),
		currentContext: &AuthContext{
			WorkDir:      "/work/dir",
			StackName:    "dev",
			ConfigHashes: make(map[string]string),
		},
		mergedConfig: &P5Config{
			Plugins: map[string]PluginConfig{
				"aws": {
					Refresh: &RefreshTrigger{
						OnStackChange: &falseVal, // Disabled
					},
				},
			},
		},
	}

	// Stack changed but trigger disabled
	should := m.ShouldRefreshCredentials("aws", "/work/dir", "prod", "my-program", nil, nil)

	if should {
		t.Error("expected ShouldRefreshCredentials=false when stack trigger disabled")
	}
}

// TestShouldRefreshCredentials_ConfigChangeOnly verifies refresh only when config changed.
func TestShouldRefreshCredentials_ConfigChangeOnly(t *testing.T) {
	trueVal := true
	m := &Manager{
		credentials: make(map[string]*Credentials),
		currentContext: &AuthContext{
			WorkDir:      "/old/work/dir",
			StackName:    "dev",
			ConfigHashes: map[string]string{"aws": "old-hash"},
		},
		mergedConfig: &P5Config{
			Plugins: map[string]PluginConfig{
				"aws": {
					Refresh: &RefreshTrigger{
						OnWorkspaceChange: &trueVal,
						OnConfigChange:    &trueVal, // Require config change too
					},
				},
			},
		},
	}

	// Workspace changed but same config - should NOT refresh
	sameConfig := map[string]any{"region": "us-east-1"}
	m.currentContext.ConfigHashes["aws"] = hashConfig(sameConfig, nil)

	should := m.ShouldRefreshCredentials("aws", "/new/work/dir", "dev", "my-program", sameConfig, nil)

	if should {
		t.Error("expected ShouldRefreshCredentials=false when config unchanged and OnConfigChange=true")
	}

	// Now with different config - should refresh
	differentConfig := map[string]any{"region": "us-west-2"}
	should = m.ShouldRefreshCredentials("aws", "/new/work/dir", "dev", "my-program", differentConfig, nil)

	if !should {
		t.Error("expected ShouldRefreshCredentials=true when config changed")
	}
}

// =============================================================================
// UpdateContext Tests
// =============================================================================

// TestUpdateContext_SetsFields verifies UpdateContext properly updates manager state.
func TestUpdateContext_SetsFields(t *testing.T) {
	m := &Manager{
		credentials: make(map[string]*Credentials),
	}

	hashes := map[string]string{"aws": "hash123"}
	m.UpdateContext("/work/dir", "dev", "my-program", hashes)

	if m.currentContext == nil {
		t.Fatal("expected currentContext to be set")
	}
	if m.currentContext.WorkDir != "/work/dir" {
		t.Errorf("expected WorkDir=%q, got %q", "/work/dir", m.currentContext.WorkDir)
	}
	if m.currentContext.StackName != "dev" {
		t.Errorf("expected StackName=%q, got %q", "dev", m.currentContext.StackName)
	}
	if m.currentContext.ProgramName != "my-program" {
		t.Errorf("expected ProgramName=%q, got %q", "my-program", m.currentContext.ProgramName)
	}
	if m.currentContext.ConfigHashes["aws"] != "hash123" {
		t.Errorf("expected ConfigHashes[aws]=%q, got %q", "hash123", m.currentContext.ConfigHashes["aws"])
	}
}

// =============================================================================
// InvalidateCredentialsForContext Tests
// =============================================================================

// TestInvalidateCredentialsForContext_NoCurrentContext verifies no panic with nil context.
func TestInvalidateCredentialsForContext_NoCurrentContext(t *testing.T) {
	m := &Manager{
		credentials:    make(map[string]*Credentials),
		currentContext: nil,
	}

	// Should not panic
	m.InvalidateCredentialsForContext("/work/dir", "dev", "my-program", nil)

	// No assertion needed - just verify no panic
}

// TestInvalidateCredentialsForContext_WorkspaceChanged verifies credentials invalidated on workspace change.
func TestInvalidateCredentialsForContext_WorkspaceChanged(t *testing.T) {
	m := &Manager{
		credentials: map[string]*Credentials{
			"aws": {PluginName: "aws", Env: map[string]string{"AWS_KEY": "xxx"}},
		},
		currentContext: &AuthContext{
			WorkDir:   "/old/work/dir",
			StackName: "dev",
		},
	}

	// Workspace changed - default trigger should invalidate
	m.InvalidateCredentialsForContext("/new/work/dir", "dev", "my-program", nil)

	if _, ok := m.credentials["aws"]; ok {
		t.Error("expected aws credentials to be invalidated on workspace change")
	}
}

// TestInvalidateCredentialsForContext_StackChanged verifies credentials invalidated on stack change.
func TestInvalidateCredentialsForContext_StackChanged(t *testing.T) {
	m := &Manager{
		credentials: map[string]*Credentials{
			"aws": {PluginName: "aws", Env: map[string]string{"AWS_KEY": "xxx"}},
		},
		currentContext: &AuthContext{
			WorkDir:   "/work/dir",
			StackName: "dev",
		},
	}

	// Stack changed - default trigger should invalidate
	m.InvalidateCredentialsForContext("/work/dir", "prod", "my-program", nil)

	if _, ok := m.credentials["aws"]; ok {
		t.Error("expected aws credentials to be invalidated on stack change")
	}
}

// TestInvalidateCredentialsForContext_NoChange verifies credentials preserved when nothing changed.
func TestInvalidateCredentialsForContext_NoChange(t *testing.T) {
	m := &Manager{
		credentials: map[string]*Credentials{
			"aws": {PluginName: "aws", Env: map[string]string{"AWS_KEY": "xxx"}},
		},
		currentContext: &AuthContext{
			WorkDir:   "/work/dir",
			StackName: "dev",
		},
	}

	// Same workspace and stack
	m.InvalidateCredentialsForContext("/work/dir", "dev", "my-program", nil)

	if _, ok := m.credentials["aws"]; !ok {
		t.Error("expected aws credentials to be preserved when nothing changed")
	}
}

// TestInvalidateCredentialsForContext_SelectiveInvalidation verifies only affected plugins invalidated.
func TestInvalidateCredentialsForContext_SelectiveInvalidation(t *testing.T) {
	falseVal := false
	trueVal := true

	m := &Manager{
		credentials: map[string]*Credentials{
			"aws":        {PluginName: "aws", Env: map[string]string{"AWS_KEY": "xxx"}},
			"kubernetes": {PluginName: "kubernetes", Env: map[string]string{"KUBECONFIG": "/path"}},
		},
		currentContext: &AuthContext{
			WorkDir:   "/old/work/dir",
			StackName: "dev",
		},
	}

	// aws: workspace trigger disabled, kubernetes: workspace trigger enabled
	p5Config := &P5Config{
		Plugins: map[string]PluginConfig{
			"aws": {
				Refresh: &RefreshTrigger{OnWorkspaceChange: &falseVal},
			},
			"kubernetes": {
				Refresh: &RefreshTrigger{OnWorkspaceChange: &trueVal},
			},
		},
	}

	// Workspace changed
	m.InvalidateCredentialsForContext("/new/work/dir", "dev", "my-program", p5Config)

	// aws should be preserved (trigger disabled)
	if _, ok := m.credentials["aws"]; !ok {
		t.Error("expected aws credentials to be preserved (trigger disabled)")
	}

	// kubernetes should be invalidated (trigger enabled)
	if _, ok := m.credentials["kubernetes"]; ok {
		t.Error("expected kubernetes credentials to be invalidated (trigger enabled)")
	}
}
