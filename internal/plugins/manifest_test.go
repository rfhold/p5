package plugins

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMergeConfigs_GlobalOnly verifies merging with only global config.
func TestMergeConfigs_GlobalOnly(t *testing.T) {
	global := &GlobalConfig{
		Plugins: map[string]PluginConfig{
			"aws": {Cmd: "/usr/bin/aws-plugin", Args: []string{"--verbose"}},
		},
	}

	result := MergeConfigs(global, nil)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Plugins) != 1 {
		t.Errorf("expected 1 plugin, got %d", len(result.Plugins))
	}
	if result.Plugins["aws"].Cmd != "/usr/bin/aws-plugin" {
		t.Errorf("expected Cmd=%q, got %q", "/usr/bin/aws-plugin", result.Plugins["aws"].Cmd)
	}
	if len(result.Plugins["aws"].Args) != 1 || result.Plugins["aws"].Args[0] != "--verbose" {
		t.Errorf("expected Args=[--verbose], got %v", result.Plugins["aws"].Args)
	}
}

// TestMergeConfigs_ProgramOnly verifies merging with only program config.
func TestMergeConfigs_ProgramOnly(t *testing.T) {
	program := &P5Config{
		Plugins: map[string]PluginConfig{
			"kubernetes": {Cmd: "/usr/bin/k8s-plugin"},
		},
	}

	result := MergeConfigs(nil, program)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Plugins) != 1 {
		t.Errorf("expected 1 plugin, got %d", len(result.Plugins))
	}
	if result.Plugins["kubernetes"].Cmd != "/usr/bin/k8s-plugin" {
		t.Errorf("expected Cmd=%q, got %q", "/usr/bin/k8s-plugin", result.Plugins["kubernetes"].Cmd)
	}
}

// TestMergeConfigs_OverrideCmd verifies program cmd overrides global.
func TestMergeConfigs_OverrideCmd(t *testing.T) {
	global := &GlobalConfig{
		Plugins: map[string]PluginConfig{
			"aws": {Cmd: "/global/aws-plugin"},
		},
	}
	program := &P5Config{
		Plugins: map[string]PluginConfig{
			"aws": {Cmd: "/program/aws-plugin"},
		},
	}

	result := MergeConfigs(global, program)

	if result.Plugins["aws"].Cmd != "/program/aws-plugin" {
		t.Errorf("expected program Cmd to override global, got %q", result.Plugins["aws"].Cmd)
	}
}

// TestMergeConfigs_OverrideArgs verifies program args overrides global.
func TestMergeConfigs_OverrideArgs(t *testing.T) {
	global := &GlobalConfig{
		Plugins: map[string]PluginConfig{
			"aws": {Cmd: "/aws", Args: []string{"--global"}},
		},
	}
	program := &P5Config{
		Plugins: map[string]PluginConfig{
			"aws": {Args: []string{"--program", "--extra"}},
		},
	}

	result := MergeConfigs(global, program)

	// Program args should completely override global args
	if len(result.Plugins["aws"].Args) != 2 {
		t.Errorf("expected 2 args, got %d", len(result.Plugins["aws"].Args))
	}
	if result.Plugins["aws"].Args[0] != "--program" {
		t.Errorf("expected Args[0]=%q, got %q", "--program", result.Plugins["aws"].Args[0])
	}
	// Cmd should be preserved from global since program didn't override it
	if result.Plugins["aws"].Cmd != "/aws" {
		t.Errorf("expected Cmd preserved from global, got %q", result.Plugins["aws"].Cmd)
	}
}

// TestMergeConfigs_MergeConfigMaps verifies nested config maps merge (program overrides).
func TestMergeConfigs_MergeConfigMaps(t *testing.T) {
	global := &GlobalConfig{
		Plugins: map[string]PluginConfig{
			"aws": {
				Cmd: "/aws",
				Config: map[string]any{
					"region":  "us-east-1",
					"profile": "default",
				},
			},
		},
	}
	program := &P5Config{
		Plugins: map[string]PluginConfig{
			"aws": {
				Config: map[string]any{
					"region": "us-west-2", // Override region
					"role":   "my-role",   // Add new key
				},
			},
		},
	}

	result := MergeConfigs(global, program)

	cfg := result.Plugins["aws"].Config
	if cfg["region"] != "us-west-2" {
		t.Errorf("expected region=%q, got %q", "us-west-2", cfg["region"])
	}
	if cfg["profile"] != "default" {
		t.Errorf("expected profile=%q (preserved from global), got %q", "default", cfg["profile"])
	}
	if cfg["role"] != "my-role" {
		t.Errorf("expected role=%q (added from program), got %q", "my-role", cfg["role"])
	}
}

// TestMergeConfigs_OverrideRefresh verifies refresh triggers from program override global.
func TestMergeConfigs_OverrideRefresh(t *testing.T) {
	globalTrue := true
	programFalse := false

	global := &GlobalConfig{
		Plugins: map[string]PluginConfig{
			"aws": {
				Cmd:     "/aws",
				Refresh: &RefreshTrigger{OnWorkspaceChange: &globalTrue},
			},
		},
	}
	program := &P5Config{
		Plugins: map[string]PluginConfig{
			"aws": {
				Refresh: &RefreshTrigger{OnWorkspaceChange: &programFalse},
			},
		},
	}

	result := MergeConfigs(global, program)

	if result.Plugins["aws"].Refresh == nil {
		t.Fatal("expected Refresh to be set")
	}
	if result.Plugins["aws"].Refresh.ShouldRefreshOnWorkspaceChange() != false {
		t.Error("expected program Refresh to override global")
	}
}

// TestMergeConfigs_OverrideImportHelper verifies import helper bool override.
func TestMergeConfigs_OverrideImportHelper(t *testing.T) {
	global := &GlobalConfig{
		Plugins: map[string]PluginConfig{
			"aws": {Cmd: "/aws", ImportHelper: false},
		},
	}
	program := &P5Config{
		Plugins: map[string]PluginConfig{
			"aws": {ImportHelper: true},
		},
	}

	result := MergeConfigs(global, program)

	if !result.Plugins["aws"].ImportHelper {
		t.Error("expected ImportHelper=true from program")
	}
}

// TestMergeConfigs_NilInputs verifies handling of nil global and program.
func TestMergeConfigs_NilInputs(t *testing.T) {
	result := MergeConfigs(nil, nil)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Plugins == nil {
		t.Error("expected Plugins map to be initialized")
	}
	if len(result.Plugins) != 0 {
		t.Errorf("expected 0 plugins, got %d", len(result.Plugins))
	}
}

// TestMergeConfigs_EmptyPlugins verifies empty plugin maps.
func TestMergeConfigs_EmptyPlugins(t *testing.T) {
	global := &GlobalConfig{Plugins: make(map[string]PluginConfig)}
	program := &P5Config{Plugins: make(map[string]PluginConfig)}

	result := MergeConfigs(global, program)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Plugins) != 0 {
		t.Errorf("expected 0 plugins, got %d", len(result.Plugins))
	}
}

// TestMergeConfigs_AddNewPluginFromProgram verifies program can add new plugins not in global.
func TestMergeConfigs_AddNewPluginFromProgram(t *testing.T) {
	global := &GlobalConfig{
		Plugins: map[string]PluginConfig{
			"aws": {Cmd: "/aws"},
		},
	}
	program := &P5Config{
		Plugins: map[string]PluginConfig{
			"kubernetes": {Cmd: "/k8s"},
		},
	}

	result := MergeConfigs(global, program)

	if len(result.Plugins) != 2 {
		t.Errorf("expected 2 plugins (aws + kubernetes), got %d", len(result.Plugins))
	}
	if result.Plugins["aws"].Cmd != "/aws" {
		t.Error("expected aws plugin from global")
	}
	if result.Plugins["kubernetes"].Cmd != "/k8s" {
		t.Error("expected kubernetes plugin from program")
	}
}

// TestRefreshTrigger_DefaultValues verifies nil trigger uses defaults.
func TestRefreshTrigger_DefaultValues(t *testing.T) {
	var trigger *RefreshTrigger = nil

	// Defaults: workspace=true, stack=true, config=false
	if !trigger.ShouldRefreshOnWorkspaceChange() {
		t.Error("expected default ShouldRefreshOnWorkspaceChange=true")
	}
	if !trigger.ShouldRefreshOnStackChange() {
		t.Error("expected default ShouldRefreshOnStackChange=true")
	}
	if trigger.ShouldRefreshOnConfigChange() {
		t.Error("expected default ShouldRefreshOnConfigChange=false")
	}
}

// TestRefreshTrigger_ExplicitTrue verifies explicitly set to true.
func TestRefreshTrigger_ExplicitTrue(t *testing.T) {
	trueVal := true
	trigger := &RefreshTrigger{
		OnWorkspaceChange: &trueVal,
		OnStackChange:     &trueVal,
		OnConfigChange:    &trueVal,
	}

	if !trigger.ShouldRefreshOnWorkspaceChange() {
		t.Error("expected ShouldRefreshOnWorkspaceChange=true")
	}
	if !trigger.ShouldRefreshOnStackChange() {
		t.Error("expected ShouldRefreshOnStackChange=true")
	}
	if !trigger.ShouldRefreshOnConfigChange() {
		t.Error("expected ShouldRefreshOnConfigChange=true")
	}
}

// TestRefreshTrigger_ExplicitFalse verifies explicitly set to false.
func TestRefreshTrigger_ExplicitFalse(t *testing.T) {
	falseVal := false
	trigger := &RefreshTrigger{
		OnWorkspaceChange: &falseVal,
		OnStackChange:     &falseVal,
		OnConfigChange:    &falseVal,
	}

	if trigger.ShouldRefreshOnWorkspaceChange() {
		t.Error("expected ShouldRefreshOnWorkspaceChange=false")
	}
	if trigger.ShouldRefreshOnStackChange() {
		t.Error("expected ShouldRefreshOnStackChange=false")
	}
	if trigger.ShouldRefreshOnConfigChange() {
		t.Error("expected ShouldRefreshOnConfigChange=false")
	}
}

// TestRefreshTrigger_PartialNil verifies partial nil fields use defaults.
func TestRefreshTrigger_PartialNil(t *testing.T) {
	falseVal := false
	trigger := &RefreshTrigger{
		OnWorkspaceChange: nil, // Should default to true
		OnStackChange:     &falseVal,
		OnConfigChange:    nil, // Should default to false
	}

	if !trigger.ShouldRefreshOnWorkspaceChange() {
		t.Error("expected ShouldRefreshOnWorkspaceChange=true (default)")
	}
	if trigger.ShouldRefreshOnStackChange() {
		t.Error("expected ShouldRefreshOnStackChange=false (explicit)")
	}
	if trigger.ShouldRefreshOnConfigChange() {
		t.Error("expected ShouldRefreshOnConfigChange=false (default)")
	}
}

// TestLoadP5Config_Valid verifies loading a valid Pulumi.yaml with p5 section.
func TestLoadP5Config_Valid(t *testing.T) {
	testdataDir := "testdata"
	config, err := LoadP5Config(filepath.Join(testdataDir, "valid-pulumi.yaml"))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if config == nil {
		t.Fatal("expected non-nil config")
	}
	if len(config.Plugins) != 2 {
		t.Errorf("expected 2 plugins, got %d", len(config.Plugins))
	}

	// Check aws plugin
	aws, ok := config.Plugins["aws"]
	if !ok {
		t.Fatal("expected aws plugin to exist")
	}
	if aws.Cmd != "/usr/bin/aws-plugin" {
		t.Errorf("expected aws Cmd=%q, got %q", "/usr/bin/aws-plugin", aws.Cmd)
	}
	if len(aws.Args) != 1 || aws.Args[0] != "--verbose" {
		t.Errorf("expected aws Args=[--verbose], got %v", aws.Args)
	}
	if aws.Config["region"] != "us-east-1" {
		t.Errorf("expected aws config region=%q, got %q", "us-east-1", aws.Config["region"])
	}
	if aws.Refresh == nil {
		t.Error("expected aws Refresh to be set")
	} else {
		if !aws.Refresh.ShouldRefreshOnWorkspaceChange() {
			t.Error("expected aws OnWorkspaceChange=true")
		}
		if aws.Refresh.ShouldRefreshOnStackChange() {
			t.Error("expected aws OnStackChange=false")
		}
	}

	// Check kubernetes plugin
	k8s, ok := config.Plugins["kubernetes"]
	if !ok {
		t.Fatal("expected kubernetes plugin to exist")
	}
	if !k8s.ImportHelper {
		t.Error("expected kubernetes ImportHelper=true")
	}
}

// TestLoadP5Config_NoP5Section verifies loading a Pulumi.yaml without p5 section returns empty config.
func TestLoadP5Config_NoP5Section(t *testing.T) {
	testdataDir := "testdata"
	config, err := LoadP5Config(filepath.Join(testdataDir, "no-p5-pulumi.yaml"))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if config == nil {
		t.Fatal("expected non-nil config")
	}
	if len(config.Plugins) != 0 {
		t.Errorf("expected 0 plugins for no-p5 file, got %d", len(config.Plugins))
	}
}

// TestLoadP5Config_InvalidYAML verifies error on invalid YAML.
func TestLoadP5Config_InvalidYAML(t *testing.T) {
	testdataDir := "testdata"
	_, err := LoadP5Config(filepath.Join(testdataDir, "invalid.yaml"))

	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

// TestLoadP5Config_FileNotFound verifies error when file doesn't exist.
func TestLoadP5Config_FileNotFound(t *testing.T) {
	_, err := LoadP5Config("nonexistent-file.yaml")

	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

// LoadStackPluginConfig Tests

// TestLoadStackPluginConfig_Valid verifies loading valid stack config.
func TestLoadStackPluginConfig_Valid(t *testing.T) {
	testdataDir := "testdata"
	result, err := LoadStackPluginConfig(testdataDir, "dev", "aws")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Config["region"] != "us-west-2" {
		t.Errorf("expected region=%q, got %q", "us-west-2", result.Config["region"])
	}
	if result.Config["account"] != "123456789" {
		t.Errorf("expected account=%q, got %q", "123456789", result.Config["account"])
	}
}

// TestLoadStackPluginConfig_NoFile verifies empty config returned when stack file doesn't exist.
func TestLoadStackPluginConfig_NoFile(t *testing.T) {
	testdataDir := "testdata"
	result, err := LoadStackPluginConfig(testdataDir, "nonexistent", "aws")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Config) != 0 {
		t.Errorf("expected empty config for non-existent stack, got %v", result.Config)
	}
}

// TestLoadStackPluginConfig_NoP5Plugins verifies empty config returned when no p5:plugins section.
func TestLoadStackPluginConfig_NoP5Plugins(t *testing.T) {
	// Create a temp stack config without p5:plugins
	tmpDir := t.TempDir()
	stackContent := []byte("config:\n  other:key: value\n")
	err := os.WriteFile(filepath.Join(tmpDir, "Pulumi.test.yaml"), stackContent, 0o600)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	result, err := LoadStackPluginConfig(tmpDir, "test", "aws")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Config) != 0 {
		t.Errorf("expected empty config when no p5:plugins, got %v", result.Config)
	}
}

// TestLoadStackPluginConfig_PluginNotFound verifies empty config returned when plugin not in config.
func TestLoadStackPluginConfig_PluginNotFound(t *testing.T) {
	testdataDir := "testdata"
	result, err := LoadStackPluginConfig(testdataDir, "dev", "nonexistent-plugin")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Config) != 0 {
		t.Errorf("expected empty config for unknown plugin, got %v", result.Config)
	}
}

// TestLoadStackPluginConfig_SecretsProvider verifies secretsprovider is extracted.
func TestLoadStackPluginConfig_SecretsProvider(t *testing.T) {
	tmpDir := t.TempDir()
	stackContent := []byte(`secretsprovider: awskms://alias/my-key
config:
  p5:plugins:
    aws:
      config:
        region: us-east-1
`)
	err := os.WriteFile(filepath.Join(tmpDir, "Pulumi.test.yaml"), stackContent, 0o600)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	result, err := LoadStackPluginConfig(tmpDir, "test", "aws")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.SecretsProvider != "awskms://alias/my-key" {
		t.Errorf("expected secretsprovider=%q, got %q", "awskms://alias/my-key", result.SecretsProvider)
	}
	if result.Config["region"] != "us-east-1" {
		t.Errorf("expected region=%q, got %q", "us-east-1", result.Config["region"])
	}
}

// LoadGlobalConfig Tests

// TestLoadGlobalConfig_Valid verifies loading p5.toml.
func TestLoadGlobalConfig_Valid(t *testing.T) {
	// Use loadGlobalConfigFile directly to avoid git root lookup
	// which would find the repo's p5.toml instead of testdata/p5.toml
	testdataDir := "testdata"
	configPath := filepath.Join(testdataDir, "p5.toml")
	config, err := loadGlobalConfigFile(configPath)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if config == nil {
		t.Fatal("expected non-nil config")
	}
	if len(config.Plugins) != 2 {
		t.Errorf("expected 2 plugins, got %d", len(config.Plugins))
	}

	// Check aws plugin
	aws, ok := config.Plugins["aws"]
	if !ok {
		t.Fatal("expected aws plugin")
	}
	if aws.Cmd != "/global/aws-plugin" {
		t.Errorf("expected aws Cmd=%q, got %q", "/global/aws-plugin", aws.Cmd)
	}
	if !aws.ImportHelper {
		t.Error("expected aws ImportHelper=true")
	}
	if aws.Config["region"] != "eu-west-1" {
		t.Errorf("expected aws config region=%q, got %q", "eu-west-1", aws.Config["region"])
	}

	// Check cloudflare plugin
	cf, ok := config.Plugins["cloudflare"]
	if !ok {
		t.Fatal("expected cloudflare plugin")
	}
	if cf.Cmd != "cloudflare-plugin" {
		t.Errorf("expected cloudflare Cmd=%q, got %q", "cloudflare-plugin", cf.Cmd)
	}
}

// TestLoadGlobalConfig_FallbackToLaunchDir verifies loading from launch directory when not in git repo.
func TestLoadGlobalConfig_FallbackToLaunchDir(t *testing.T) {
	// Create temp directory with p5.toml (outside git repo)
	tmpDir := t.TempDir()
	content := []byte("[plugins.test]\ncmd = \"test-plugin\"\n")
	err := os.WriteFile(filepath.Join(tmpDir, "p5.toml"), content, 0o600)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	config, path, err := LoadGlobalConfig(tmpDir)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if config == nil {
		t.Fatal("expected non-nil config")
	}
	if path == "" {
		t.Error("expected non-empty config path")
	}
	if len(config.Plugins) != 1 {
		t.Errorf("expected 1 plugin, got %d", len(config.Plugins))
	}
	if config.Plugins["test"].Cmd != "test-plugin" {
		t.Errorf("expected test plugin cmd=%q, got %q", "test-plugin", config.Plugins["test"].Cmd)
	}
}

// TestLoadGlobalConfig_NotFound verifies empty config when p5.toml doesn't exist.
func TestLoadGlobalConfig_NotFound(t *testing.T) {
	tmpDir := t.TempDir() // Empty directory, no p5.toml
	config, path, err := LoadGlobalConfig(tmpDir)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if config == nil {
		t.Fatal("expected non-nil config")
	}
	if path != "" {
		t.Errorf("expected empty path for non-existent config, got %q", path)
	}
	if len(config.Plugins) != 0 {
		t.Errorf("expected 0 plugins for non-existent config, got %d", len(config.Plugins))
	}
}

// TestLoadGlobalConfig_InvalidTOML verifies error on invalid TOML.
func TestLoadGlobalConfig_InvalidTOML(t *testing.T) {
	tmpDir := t.TempDir()
	// Create invalid TOML file
	invalidContent := []byte("this is not valid toml [[[")
	err := os.WriteFile(filepath.Join(tmpDir, "p5.toml"), invalidContent, 0o600)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	_, _, err = LoadGlobalConfig(tmpDir)

	if err == nil {
		t.Error("expected error for invalid TOML")
	}
}

// loadGlobalConfigFile Tests

// TestLoadGlobalConfigFile_NilPlugins verifies nil plugins map is initialized.
func TestLoadGlobalConfigFile_NilPlugins(t *testing.T) {
	tmpDir := t.TempDir()
	// Create TOML file with no plugins section
	content := []byte("# Empty config\n")
	configPath := filepath.Join(tmpDir, "p5.toml")
	err := os.WriteFile(configPath, content, 0o600)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	config, err := loadGlobalConfigFile(configPath)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if config.Plugins == nil {
		t.Error("expected Plugins map to be initialized, not nil")
	}
}

// GetOrderedPluginNames Tests

// TestGetOrderedPluginNames_NoOrder verifies all plugins returned when no order specified.
func TestGetOrderedPluginNames_NoOrder(t *testing.T) {
	config := &P5Config{
		Plugins: map[string]PluginConfig{
			"aws":        {Cmd: "/aws"},
			"kubernetes": {Cmd: "/k8s"},
			"cloudflare": {Cmd: "/cf"},
		},
	}

	names := config.GetOrderedPluginNames()

	if len(names) != 3 {
		t.Errorf("expected 3 plugins, got %d", len(names))
	}

	// All plugins should be present (order is non-deterministic without Order field)
	found := make(map[string]bool)
	for _, name := range names {
		found[name] = true
	}
	for expected := range config.Plugins {
		if !found[expected] {
			t.Errorf("expected plugin %q to be in result", expected)
		}
	}
}

// TestGetOrderedPluginNames_WithOrder verifies plugins are returned in specified order.
func TestGetOrderedPluginNames_WithOrder(t *testing.T) {
	config := &P5Config{
		Plugins: map[string]PluginConfig{
			"aws":        {Cmd: "/aws"},
			"kubernetes": {Cmd: "/k8s"},
			"cloudflare": {Cmd: "/cf"},
		},
		Order: []string{"cloudflare", "aws", "kubernetes"},
	}

	names := config.GetOrderedPluginNames()

	if len(names) != 3 {
		t.Errorf("expected 3 plugins, got %d", len(names))
	}

	// Verify exact order
	expected := []string{"cloudflare", "aws", "kubernetes"}
	for i, name := range names {
		if name != expected[i] {
			t.Errorf("expected names[%d]=%q, got %q", i, expected[i], name)
		}
	}
}

// TestGetOrderedPluginNames_PartialOrder verifies ordered plugins come first, then remaining.
func TestGetOrderedPluginNames_PartialOrder(t *testing.T) {
	config := &P5Config{
		Plugins: map[string]PluginConfig{
			"aws":        {Cmd: "/aws"},
			"kubernetes": {Cmd: "/k8s"},
			"cloudflare": {Cmd: "/cf"},
			"gcp":        {Cmd: "/gcp"},
		},
		Order: []string{"cloudflare", "aws"}, // Only 2 of 4 plugins ordered
	}

	names := config.GetOrderedPluginNames()

	if len(names) != 4 {
		t.Errorf("expected 4 plugins, got %d", len(names))
	}

	// First two should be in order
	if names[0] != "cloudflare" {
		t.Errorf("expected names[0]=%q, got %q", "cloudflare", names[0])
	}
	if names[1] != "aws" {
		t.Errorf("expected names[1]=%q, got %q", "aws", names[1])
	}

	// Remaining should be present (order non-deterministic)
	remaining := make(map[string]bool)
	for _, name := range names[2:] {
		remaining[name] = true
	}
	if !remaining["kubernetes"] {
		t.Error("expected kubernetes in remaining plugins")
	}
	if !remaining["gcp"] {
		t.Error("expected gcp in remaining plugins")
	}
}

// TestGetOrderedPluginNames_OrderWithNonExistent verifies non-existent plugins in order are ignored.
func TestGetOrderedPluginNames_OrderWithNonExistent(t *testing.T) {
	config := &P5Config{
		Plugins: map[string]PluginConfig{
			"aws":        {Cmd: "/aws"},
			"kubernetes": {Cmd: "/k8s"},
		},
		Order: []string{"nonexistent", "aws", "also-missing", "kubernetes"},
	}

	names := config.GetOrderedPluginNames()

	if len(names) != 2 {
		t.Errorf("expected 2 plugins, got %d", len(names))
	}

	// Should only include plugins that exist, in order
	if names[0] != "aws" {
		t.Errorf("expected names[0]=%q, got %q", "aws", names[0])
	}
	if names[1] != "kubernetes" {
		t.Errorf("expected names[1]=%q, got %q", "kubernetes", names[1])
	}
}

// TestGetOrderedPluginNames_DuplicatesInOrder verifies duplicates in order are handled.
func TestGetOrderedPluginNames_DuplicatesInOrder(t *testing.T) {
	config := &P5Config{
		Plugins: map[string]PluginConfig{
			"aws":        {Cmd: "/aws"},
			"kubernetes": {Cmd: "/k8s"},
		},
		Order: []string{"aws", "kubernetes", "aws"}, // aws appears twice
	}

	names := config.GetOrderedPluginNames()

	if len(names) != 2 {
		t.Errorf("expected 2 plugins (no duplicates), got %d", len(names))
	}

	// Should only include each plugin once
	if names[0] != "aws" {
		t.Errorf("expected names[0]=%q, got %q", "aws", names[0])
	}
	if names[1] != "kubernetes" {
		t.Errorf("expected names[1]=%q, got %q", "kubernetes", names[1])
	}
}

// TestGetOrderedPluginNames_EmptyPlugins verifies nil returned for empty plugins.
func TestGetOrderedPluginNames_EmptyPlugins(t *testing.T) {
	config := &P5Config{
		Plugins: map[string]PluginConfig{},
		Order:   []string{"aws"},
	}

	names := config.GetOrderedPluginNames()

	if names != nil {
		t.Errorf("expected nil for empty plugins, got %v", names)
	}
}

// TestGetOrderedPluginNames_NilConfig verifies nil returned for nil config.
func TestGetOrderedPluginNames_NilConfig(t *testing.T) {
	var config *P5Config

	names := config.GetOrderedPluginNames()

	if names != nil {
		t.Errorf("expected nil for nil config, got %v", names)
	}
}

// MergeConfigs Order Tests

// TestMergeConfigs_OrderFromGlobal verifies order is taken from global when program has none.
func TestMergeConfigs_OrderFromGlobal(t *testing.T) {
	global := &GlobalConfig{
		Plugins: map[string]PluginConfig{
			"aws":        {Cmd: "/aws"},
			"kubernetes": {Cmd: "/k8s"},
		},
		Order: []string{"kubernetes", "aws"},
	}
	program := &P5Config{
		Plugins: map[string]PluginConfig{},
	}

	result := MergeConfigs(global, program)

	if len(result.Order) != 2 {
		t.Errorf("expected 2 items in order, got %d", len(result.Order))
	}
	if result.Order[0] != "kubernetes" {
		t.Errorf("expected Order[0]=%q, got %q", "kubernetes", result.Order[0])
	}
	if result.Order[1] != "aws" {
		t.Errorf("expected Order[1]=%q, got %q", "aws", result.Order[1])
	}
}

// TestMergeConfigs_OrderFromProgram verifies program order overrides global.
func TestMergeConfigs_OrderFromProgram(t *testing.T) {
	global := &GlobalConfig{
		Plugins: map[string]PluginConfig{
			"aws":        {Cmd: "/aws"},
			"kubernetes": {Cmd: "/k8s"},
		},
		Order: []string{"kubernetes", "aws"},
	}
	program := &P5Config{
		Plugins: map[string]PluginConfig{},
		Order:   []string{"aws", "kubernetes"}, // Different order
	}

	result := MergeConfigs(global, program)

	if len(result.Order) != 2 {
		t.Errorf("expected 2 items in order, got %d", len(result.Order))
	}
	// Program order should win
	if result.Order[0] != "aws" {
		t.Errorf("expected Order[0]=%q (from program), got %q", "aws", result.Order[0])
	}
	if result.Order[1] != "kubernetes" {
		t.Errorf("expected Order[1]=%q (from program), got %q", "kubernetes", result.Order[1])
	}
}

// TestMergeConfigs_NoOrder verifies no order when neither config has one.
func TestMergeConfigs_NoOrder(t *testing.T) {
	global := &GlobalConfig{
		Plugins: map[string]PluginConfig{
			"aws": {Cmd: "/aws"},
		},
	}
	program := &P5Config{
		Plugins: map[string]PluginConfig{},
	}

	result := MergeConfigs(global, program)

	if len(result.Order) != 0 {
		t.Errorf("expected empty order, got %v", result.Order)
	}
}
