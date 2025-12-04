package plugins

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

// Manifest describes a plugin's metadata and security permissions
type Manifest struct {
	// Plugin metadata
	Name        string `yaml:"name"`
	Version     string `yaml:"version"`
	Description string `yaml:"description,omitempty"`

	// Security permissions
	Permissions Permissions `yaml:"permissions"`

	// Whether this plugin can request browser-based auth
	Interactive bool `yaml:"interactive,omitempty"`

	// Config schema for documentation/validation
	Config ConfigSchema `yaml:"config,omitempty"`
}

// Permissions defines what the plugin is allowed to do
type Permissions struct {
	// HTTP hosts the plugin can make requests to (supports wildcards like *.okta.com)
	HTTP []string `yaml:"http,omitempty"`
	// Environment variables the plugin is allowed to set
	Env []string `yaml:"env,omitempty"`
}

// ConfigSchema defines expected configuration structure
type ConfigSchema struct {
	// Program-level config (from Pulumi.yaml)
	Program map[string]ConfigField `yaml:"program,omitempty"`
	// Stack-level config (from Pulumi.{stack}.yaml)
	Stack map[string]ConfigField `yaml:"stack,omitempty"`
}

// ConfigField describes a configuration field
type ConfigField struct {
	Type        string `yaml:"type"`
	Required    bool   `yaml:"required,omitempty"`
	Description string `yaml:"description,omitempty"`
	Default     string `yaml:"default,omitempty"`
}

// LoadManifest loads a manifest from a YAML file
func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	if err := manifest.Validate(); err != nil {
		return nil, fmt.Errorf("invalid manifest: %w", err)
	}

	return &manifest, nil
}

// Validate checks that the manifest is valid
func (m *Manifest) Validate() error {
	if m.Name == "" {
		return fmt.Errorf("name is required")
	}
	if m.Version == "" {
		return fmt.Errorf("version is required")
	}
	return nil
}

// IsHTTPAllowed checks if the given URL host is allowed by the manifest
func (m *Manifest) IsHTTPAllowed(host string) bool {
	for _, pattern := range m.Permissions.HTTP {
		if matchHost(pattern, host) {
			return true
		}
	}
	return false
}

// IsEnvAllowed checks if the given environment variable is allowed by the manifest
func (m *Manifest) IsEnvAllowed(envVar string) bool {
	for _, allowed := range m.Permissions.Env {
		if allowed == envVar {
			return true
		}
	}
	return false
}

// matchHost checks if a host matches a pattern (supports * wildcard prefix)
// Examples:
//   - "*.okta.com" matches "myorg.okta.com"
//   - "sts.amazonaws.com" matches "sts.amazonaws.com"
//   - "sts.*.amazonaws.com" matches "sts.us-west-2.amazonaws.com"
func matchHost(pattern, host string) bool {
	// Exact match
	if pattern == host {
		return true
	}

	// Wildcard prefix match (e.g., *.okta.com)
	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[1:] // Remove the *
		return strings.HasSuffix(host, suffix)
	}

	// Wildcard in middle (e.g., sts.*.amazonaws.com)
	if strings.Contains(pattern, "*") {
		parts := strings.Split(pattern, "*")
		if len(parts) == 2 {
			return strings.HasPrefix(host, parts[0]) && strings.HasSuffix(host, parts[1])
		}
	}

	return false
}

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
	// Source is the plugin location (git URL, HTTP URL, or local path)
	Source string `yaml:"source" toml:"source"`
	// Config is the program-level configuration
	Config map[string]interface{} `yaml:"config,omitempty" toml:"config,omitempty"`
	// Refresh controls when credentials should be refreshed
	Refresh *RefreshTrigger `yaml:"refresh,omitempty" toml:"refresh,omitempty"`
}

// P5Config represents the p5 configuration section in Pulumi.yaml
type P5Config struct {
	Plugins map[string]PluginConfig `yaml:"plugins,omitempty"`
}

// LoadP5Config loads p5 configuration from a Pulumi.yaml file
func LoadP5Config(pulumiYamlPath string) (*P5Config, error) {
	data, err := os.ReadFile(pulumiYamlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Pulumi.yaml: %w", err)
	}

	var raw map[string]interface{}
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

// LoadStackPluginConfig loads stack-level plugin configuration from Pulumi.{stack}.yaml
func LoadStackPluginConfig(workDir, stackName, pluginName string) (map[string]interface{}, error) {
	// Try both .yaml and .yml extensions
	stackConfigPath := filepath.Join(workDir, fmt.Sprintf("Pulumi.%s.yaml", stackName))
	if _, err := os.Stat(stackConfigPath); os.IsNotExist(err) {
		stackConfigPath = filepath.Join(workDir, fmt.Sprintf("Pulumi.%s.yml", stackName))
	}

	data, err := os.ReadFile(stackConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No stack config is fine
		}
		return nil, fmt.Errorf("failed to read stack config: %w", err)
	}

	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse stack config: %w", err)
	}

	// Look for config -> p5:plugins:{pluginName}
	configRaw, ok := raw["config"].(map[string]interface{})
	if !ok {
		return nil, nil
	}

	key := fmt.Sprintf("p5:plugins:%s", pluginName)
	pluginConfig, ok := configRaw[key].(map[string]interface{})
	if !ok {
		return nil, nil
	}

	return pluginConfig, nil
}

// GlobalConfig represents the p5.toml global configuration
type GlobalConfig struct {
	Plugins map[string]PluginConfig `toml:"plugins"`
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
		return program
	}

	merged := &P5Config{
		Plugins: make(map[string]PluginConfig),
	}

	// Start with global config
	for name, cfg := range global.Plugins {
		merged.Plugins[name] = cfg
	}

	// Override with program config
	for name, cfg := range program.Plugins {
		if existing, ok := merged.Plugins[name]; ok {
			// Merge configs - program config takes precedence
			mergedPlugin := existing

			// Source from program overrides global
			if cfg.Source != "" {
				mergedPlugin.Source = cfg.Source
			}

			// Merge config maps (program values override global)
			if mergedPlugin.Config == nil {
				mergedPlugin.Config = make(map[string]interface{})
			}
			for k, v := range cfg.Config {
				mergedPlugin.Config[k] = v
			}

			// Refresh settings from program override global
			if cfg.Refresh != nil {
				mergedPlugin.Refresh = cfg.Refresh
			}

			merged.Plugins[name] = mergedPlugin
		} else {
			// New plugin from program config
			merged.Plugins[name] = cfg
		}
	}

	return merged
}
