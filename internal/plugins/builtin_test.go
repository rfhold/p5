package plugins

import (
	"context"
	"sort"
	"testing"

	"github.com/rfhold/p5/internal/plugins/proto"
)

// mockBuiltinPlugin is a test implementation of BuiltinPlugin
type mockBuiltinPlugin struct {
	BuiltinPluginBase
}

func (m *mockBuiltinPlugin) Authenticate(ctx context.Context, req *proto.AuthenticateRequest) (*proto.AuthenticateResponse, error) {
	return SuccessResponse(map[string]string{"TEST_VAR": "test_value"}, 3600), nil
}

// TestIsBuiltin_Registered verifies returns true for registered plugins.
func TestIsBuiltin_Registered(t *testing.T) {
	// Save current registry and restore after test
	originalRegistry := builtinRegistry
	defer func() { builtinRegistry = originalRegistry }()

	// Create fresh registry for test
	builtinRegistry = make(map[string]BuiltinPlugin)

	// Register a test plugin
	testPlugin := &mockBuiltinPlugin{BuiltinPluginBase: NewBuiltinPluginBase("test-plugin")}
	RegisterBuiltin(testPlugin)

	if !IsBuiltin("test-plugin") {
		t.Error("expected IsBuiltin=true for registered plugin")
	}
}

// TestIsBuiltin_NotRegistered verifies returns false for unknown plugins.
func TestIsBuiltin_NotRegistered(t *testing.T) {
	// Save current registry and restore after test
	originalRegistry := builtinRegistry
	defer func() { builtinRegistry = originalRegistry }()

	// Create fresh registry for test
	builtinRegistry = make(map[string]BuiltinPlugin)

	if IsBuiltin("nonexistent-plugin") {
		t.Error("expected IsBuiltin=false for unregistered plugin")
	}
}

// TestGetBuiltin_Found verifies returns plugin when found.
func TestGetBuiltin_Found(t *testing.T) {
	// Save current registry and restore after test
	originalRegistry := builtinRegistry
	defer func() { builtinRegistry = originalRegistry }()

	// Create fresh registry for test
	builtinRegistry = make(map[string]BuiltinPlugin)

	// Register a test plugin
	testPlugin := &mockBuiltinPlugin{BuiltinPluginBase: NewBuiltinPluginBase("test-plugin")}
	RegisterBuiltin(testPlugin)

	plugin := GetBuiltin("test-plugin")
	if plugin == nil {
		t.Fatal("expected GetBuiltin to return plugin")
	}
	if plugin.Name() != "test-plugin" {
		t.Errorf("expected plugin Name=%q, got %q", "test-plugin", plugin.Name())
	}
}

// TestGetBuiltin_NotFound verifies returns nil for unknown plugins.
func TestGetBuiltin_NotFound(t *testing.T) {
	// Save current registry and restore after test
	originalRegistry := builtinRegistry
	defer func() { builtinRegistry = originalRegistry }()

	// Create fresh registry for test
	builtinRegistry = make(map[string]BuiltinPlugin)

	plugin := GetBuiltin("nonexistent-plugin")
	if plugin != nil {
		t.Error("expected GetBuiltin to return nil for unregistered plugin")
	}
}

// TestListBuiltins_All verifies returns all registered plugin names.
func TestListBuiltins_All(t *testing.T) {
	// Save current registry and restore after test
	originalRegistry := builtinRegistry
	defer func() { builtinRegistry = originalRegistry }()

	// Create fresh registry for test
	builtinRegistry = make(map[string]BuiltinPlugin)

	// Register multiple test plugins
	plugin1 := &mockBuiltinPlugin{BuiltinPluginBase: NewBuiltinPluginBase("plugin-a")}
	plugin2 := &mockBuiltinPlugin{BuiltinPluginBase: NewBuiltinPluginBase("plugin-b")}
	plugin3 := &mockBuiltinPlugin{BuiltinPluginBase: NewBuiltinPluginBase("plugin-c")}
	RegisterBuiltin(plugin1)
	RegisterBuiltin(plugin2)
	RegisterBuiltin(plugin3)

	names := ListBuiltins()

	if len(names) != 3 {
		t.Errorf("expected 3 plugins, got %d", len(names))
	}

	// Sort for deterministic comparison
	sort.Strings(names)
	expected := []string{"plugin-a", "plugin-b", "plugin-c"}
	for i, name := range names {
		if name != expected[i] {
			t.Errorf("expected names[%d]=%q, got %q", i, expected[i], name)
		}
	}
}

// TestListBuiltins_Empty verifies returns empty slice when no plugins registered.
func TestListBuiltins_Empty(t *testing.T) {
	// Save current registry and restore after test
	originalRegistry := builtinRegistry
	defer func() { builtinRegistry = originalRegistry }()

	// Create fresh registry for test
	builtinRegistry = make(map[string]BuiltinPlugin)

	names := ListBuiltins()

	if len(names) != 0 {
		t.Errorf("expected 0 plugins, got %d", len(names))
	}
}

// TestBuiltinPluginBase_Name verifies Name returns correct value.
func TestBuiltinPluginBase_Name(t *testing.T) {
	base := NewBuiltinPluginBase("my-plugin")

	if base.Name() != "my-plugin" {
		t.Errorf("expected Name=%q, got %q", "my-plugin", base.Name())
	}
}

// TestBuiltinPluginInstance_Authenticate verifies delegation to underlying plugin.
func TestBuiltinPluginInstance_Authenticate(t *testing.T) {
	plugin := &mockBuiltinPlugin{BuiltinPluginBase: NewBuiltinPluginBase("test")}
	instance := NewBuiltinPluginInstance("test", plugin)

	ctx := context.Background()
	req := &proto.AuthenticateRequest{
		ProgramConfig: map[string]string{},
		StackConfig:   map[string]string{},
	}

	resp, err := instance.Authenticate(ctx, req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if !resp.Success {
		t.Error("expected Success=true")
	}
	if resp.Env["TEST_VAR"] != "test_value" {
		t.Errorf("expected Env[TEST_VAR]=%q, got %q", "test_value", resp.Env["TEST_VAR"])
	}
}
