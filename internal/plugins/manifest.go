package plugins

import (
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

// RefreshTrigger defines when credentials should be refreshed
type RefreshTrigger struct {
	// OnWorkspaceChange triggers credential refresh when workspace changes
	// Default: true
	OnWorkspaceChange *bool `yaml:"onWorkspaceChange,omitempty" toml:"onWorkspaceChange,omitempty"`
	// OnStackChange triggers credential refresh when stack changes
	// Default: true
	OnStackChange *bool `yaml:"onStackChange,omitempty" toml:"onStackChange,omitempty"`
	// OnConfigChange triggers credential refresh only when plugin config changes
	// (both program and stack config are compared)
	// Default: false - when true, workspace/stack changes only refresh if config differs
	OnConfigChange *bool `yaml:"onConfigChange,omitempty" toml:"onConfigChange,omitempty"`
}

// ShouldRefreshOnWorkspaceChange returns whether to refresh on workspace change
func (r *RefreshTrigger) ShouldRefreshOnWorkspaceChange() bool {
	if r == nil || r.OnWorkspaceChange == nil {
		return true // default
	}
	return *r.OnWorkspaceChange
}

// ShouldRefreshOnStackChange returns whether to refresh on stack change
func (r *RefreshTrigger) ShouldRefreshOnStackChange() bool {
	if r == nil || r.OnStackChange == nil {
		return true // default
	}
	return *r.OnStackChange
}

// ShouldRefreshOnConfigChange returns whether to only refresh when config changes
func (r *RefreshTrigger) ShouldRefreshOnConfigChange() bool {
	if r == nil || r.OnConfigChange == nil {
		return false // default
	}
	return *r.OnConfigChange
}

// PluginConfig represents the configuration for a plugin from Pulumi.yaml or p5.toml
type PluginConfig struct {
	// Cmd is the command to run the plugin (path to executable)
	Cmd string `yaml:"cmd" toml:"cmd"`
	// Args are optional arguments to pass to the plugin command
	Args []string `yaml:"args,omitempty" toml:"args,omitempty"`
	// Config is the program-level configuration
	Config map[string]any `yaml:"config,omitempty" toml:"config,omitempty"`
	// Refresh controls when credentials should be refreshed
	Refresh *RefreshTrigger `yaml:"refresh,omitempty" toml:"refresh,omitempty"`

	// Import helper settings
	// ImportHelper enables the import helper capability for this plugin (default: false)
	ImportHelper bool `yaml:"import_helper,omitempty" toml:"import_helper,omitempty"`
	// UseAuthEnv passes merged auth environment variables to import helper requests (default: false)
	UseAuthEnv bool `yaml:"use_auth_env,omitempty" toml:"use_auth_env,omitempty"`

	// Resource opener settings
	// ResourceOpener enables the resource opener capability for this plugin (default: false)
	ResourceOpener bool `yaml:"resource_opener,omitempty" toml:"resource_opener,omitempty"`
}

// P5Config represents the p5 configuration section in Pulumi.yaml
type P5Config struct {
	Plugins map[string]PluginConfig `yaml:"plugins,omitempty"`
	// Order specifies the execution order for plugin authentication.
	// Plugins are authenticated sequentially in this order.
	// Plugins not listed in order will run after ordered plugins (in non-deterministic order).
	Order []string `yaml:"order,omitempty" toml:"order,omitempty"`
}

// LoadP5Config loads p5 configuration from a Pulumi.yaml file
func LoadP5Config(pulumiYamlPath string) (*P5Config, error) {
	data, err := os.ReadFile(pulumiYamlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Pulumi.yaml: %w", err)
	}

	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse Pulumi.yaml: %w", err)
	}

	// Extract p5 section
	p5Raw, ok := raw["p5"]
	if !ok {
		// No p5 config, return empty
		return &P5Config{}, nil
	}

	// Re-marshal and unmarshal the p5 section
	p5Data, err := yaml.Marshal(p5Raw)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal p5 config: %w", err)
	}

	var p5Config P5Config
	if err := yaml.Unmarshal(p5Data, &p5Config); err != nil {
		return nil, fmt.Errorf("failed to parse p5 config: %w", err)
	}

	return &p5Config, nil
}

// StackPluginConfigResult holds the result of loading stack plugin configuration
type StackPluginConfigResult struct {
	Config          map[string]any
	SecretsProvider string
}

// LoadStackPluginConfig loads stack-level plugin configuration from Pulumi.{stack}.yaml
func LoadStackPluginConfig(workDir, stackName, pluginName string) (*StackPluginConfigResult, error) {
	// Try both .yaml and .yml extensions
	stackConfigPath := filepath.Join(workDir, fmt.Sprintf("Pulumi.%s.yaml", stackName))
	if _, err := os.Stat(stackConfigPath); os.IsNotExist(err) {
		stackConfigPath = filepath.Join(workDir, fmt.Sprintf("Pulumi.%s.yml", stackName))
	}

	data, err := os.ReadFile(stackConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &StackPluginConfigResult{Config: map[string]any{}}, nil // No stack config is fine
		}
		return nil, fmt.Errorf("failed to read stack config: %w", err)
	}

	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse stack config: %w", err)
	}

	result := &StackPluginConfigResult{
		Config: map[string]any{},
	}

	// Extract secretsprovider if present
	if sp, ok := raw["secretsprovider"].(string); ok {
		result.SecretsProvider = sp
	}

	// Look for config -> p5:plugins -> {pluginName}
	configRaw, ok := raw["config"].(map[string]any)
	if !ok {
		return result, nil
	}

	p5Plugins, ok := configRaw["p5:plugins"].(map[string]any)
	if !ok {
		return result, nil
	}

	pluginConfig, ok := p5Plugins[pluginName].(map[string]any)
	if !ok {
		return result, nil
	}

	// Look for .config nesting to match global and program config structure
	// config -> p5:plugins -> {pluginName} -> config
	if nestedConfig, ok := pluginConfig["config"].(map[string]any); ok {
		result.Config = nestedConfig
	} else {
		result.Config = pluginConfig
	}

	return result, nil
}

// GlobalConfig represents the p5.toml global configuration
type GlobalConfig struct {
	Plugins map[string]PluginConfig `toml:"plugins"`
	// Order specifies the execution order for plugin authentication.
	// Plugins are authenticated sequentially in this order.
	// Plugins not listed in order will run after ordered plugins (in non-deterministic order).
	Order []string `toml:"order,omitempty"`
}

// LoadGlobalConfig loads p5.toml from either git root or launch directory
// Priority: git root > launch directory
func LoadGlobalConfig(launchDir string) (*GlobalConfig, string, error) {
	// Try git root first
	gitRoot, err := findGitRoot(launchDir)
	if err == nil && gitRoot != "" {
		configPath := filepath.Join(gitRoot, "p5.toml")
		if _, err := os.Stat(configPath); err == nil {
			config, err := loadGlobalConfigFile(configPath)
			if err != nil {
				return nil, "", fmt.Errorf("failed to load %s: %w", configPath, err)
			}
			return config, configPath, nil
		}
	}

	// Fall back to launch directory
	configPath := filepath.Join(launchDir, "p5.toml")
	if _, err := os.Stat(configPath); err == nil {
		config, err := loadGlobalConfigFile(configPath)
		if err != nil {
			return nil, "", fmt.Errorf("failed to load %s: %w", configPath, err)
		}
		return config, configPath, nil
	}

	// No p5.toml found, return empty config
	return &GlobalConfig{Plugins: make(map[string]PluginConfig)}, "", nil
}

// loadGlobalConfigFile loads a p5.toml file
func loadGlobalConfigFile(path string) (*GlobalConfig, error) {
	var config GlobalConfig
	if _, err := toml.DecodeFile(path, &config); err != nil {
		return nil, err
	}
	if config.Plugins == nil {
		config.Plugins = make(map[string]PluginConfig)
	}
	return &config, nil
}

// findGitRoot finds the git repository root from the given directory
func findGitRoot(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// MergeConfigs merges global config (p5.toml) with program config (Pulumi.yaml)
// Program config takes precedence over global config
func MergeConfigs(global *GlobalConfig, program *P5Config) *P5Config {
	if program == nil {
		program = &P5Config{Plugins: make(map[string]PluginConfig)}
	}
	if global == nil || len(global.Plugins) == 0 {
		// Still need to handle order even if no global plugins
		return program
	}

	merged := &P5Config{
		Plugins: make(map[string]PluginConfig),
	}

	// Start with global config
	maps.Copy(merged.Plugins, global.Plugins)

	// Override with program config
	for name, cfg := range program.Plugins {
		if existing, ok := merged.Plugins[name]; ok {
			merged.Plugins[name] = mergePluginConfig(existing, cfg)
		} else {
			merged.Plugins[name] = cfg
		}
	}

	// Order: program config takes precedence if specified, otherwise use global
	if len(program.Order) > 0 {
		merged.Order = program.Order
	} else if len(global.Order) > 0 {
		merged.Order = global.Order
	}

	return merged
}

// GetOrderedPluginNames returns plugin names in execution order.
// Plugins specified in Order come first (in that order), followed by
// any remaining plugins not in the order list (in non-deterministic order).
func (c *P5Config) GetOrderedPluginNames() []string {
	if c == nil || len(c.Plugins) == 0 {
		return nil
	}

	// Track which plugins we've added
	seen := make(map[string]bool)
	var result []string

	// First, add plugins in the specified order (if they exist in Plugins map)
	for _, name := range c.Order {
		if _, exists := c.Plugins[name]; exists && !seen[name] {
			result = append(result, name)
			seen[name] = true
		}
	}

	// Then add any remaining plugins not in the order list
	for name := range c.Plugins {
		if !seen[name] {
			result = append(result, name)
		}
	}

	return result
}

func mergePluginConfig(base, override PluginConfig) PluginConfig {
	if override.Cmd != "" {
		base.Cmd = override.Cmd
	}
	if len(override.Args) > 0 {
		base.Args = override.Args
	}
	if base.Config == nil {
		base.Config = make(map[string]any)
	}
	maps.Copy(base.Config, override.Config)
	if override.Refresh != nil {
		base.Refresh = override.Refresh
	}
	if override.ImportHelper {
		base.ImportHelper = override.ImportHelper
	}
	if override.UseAuthEnv {
		base.UseAuthEnv = override.UseAuthEnv
	}
	if override.ResourceOpener {
		base.ResourceOpener = override.ResourceOpener
	}
	return base
}
