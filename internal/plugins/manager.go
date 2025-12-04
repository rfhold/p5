package plugins

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AuthContext holds the current authentication context for change detection
type AuthContext struct {
	WorkDir     string
	StackName   string
	ProgramName string
	// ConfigHashes stores hash of (program + stack) config per plugin for change detection
	ConfigHashes map[string]string
}

// Manager handles plugin lifecycle and credential management
type Manager struct {
	mu          sync.RWMutex
	plugins     map[string]*Plugin
	credentials map[string]*Credentials
	cacheDir    string

	// Track current context for change detection
	currentContext *AuthContext
	// Store merged config for refresh trigger checks
	mergedConfig *P5Config
	// Path to global config file (for logging/debugging)
	globalConfigPath string
	// Launch directory (for finding p5.toml)
	launchDir string
}

// Credentials holds the result of a plugin authentication
type Credentials struct {
	PluginName string
	Env        map[string]string
	ExpiresAt  time.Time // Zero time means never expires (TTL = -1 means always refresh)
	AlwaysCall bool      // True if TTL was -1
}

// IsExpired returns true if the credentials have expired
func (c *Credentials) IsExpired() bool {
	if c.AlwaysCall {
		return true // Always considered expired, so always re-auth
	}
	if c.ExpiresAt.IsZero() {
		return false // Never expires
	}
	return time.Now().After(c.ExpiresAt)
}

// NewManager creates a new plugin manager
// launchDir is the directory p5 was launched from (used to find p5.toml)
func NewManager(launchDir string) (*Manager, error) {
	cacheDir, err := GetCacheDir()
	if err != nil {
		return nil, err
	}

	return &Manager{
		plugins:     make(map[string]*Plugin),
		credentials: make(map[string]*Credentials),
		cacheDir:    cacheDir,
		launchDir:   launchDir,
	}, nil
}

// LoadPlugins loads all plugins defined in the p5 config
func (m *Manager) LoadPlugins(ctx context.Context, p5Config *P5Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Close any existing plugins
	for _, p := range m.plugins {
		p.Close(ctx)
	}
	m.plugins = make(map[string]*Plugin)

	// Load each plugin
	for name, pluginConfig := range p5Config.Plugins {
		if err := m.loadPlugin(ctx, name, pluginConfig); err != nil {
			return fmt.Errorf("failed to load plugin %s: %w", name, err)
		}
	}

	return nil
}

// loadPlugin loads a single plugin
func (m *Manager) loadPlugin(ctx context.Context, name string, config PluginConfig) error {
	// Parse the source
	source, err := ParseSource(config.Source)
	if err != nil {
		return fmt.Errorf("invalid source: %w", err)
	}

	// Download/locate the plugin files
	files, err := source.Download(ctx, m.cacheDir)
	if err != nil {
		return fmt.Errorf("failed to download plugin: %w", err)
	}

	// Load the manifest
	manifest, err := LoadManifest(files.ManifestPath)
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	// Load the plugin
	plugin, err := LoadPlugin(ctx, name, files.WasmPath, manifest)
	if err != nil {
		return fmt.Errorf("failed to load plugin: %w", err)
	}

	m.plugins[name] = plugin
	return nil
}

// AuthenticateResult holds the result of an authentication attempt
type AuthenticateResult struct {
	PluginName  string
	Credentials *Credentials
	Error       error
}

// AuthenticateAll runs authentication for all plugins in parallel
func (m *Manager) AuthenticateAll(ctx context.Context, programName, stackName string, p5Config *P5Config, workDir string) ([]AuthenticateResult, error) {
	m.mu.RLock()
	plugins := make(map[string]*Plugin, len(m.plugins))
	for k, v := range m.plugins {
		plugins[k] = v
	}
	m.mu.RUnlock()

	if len(plugins) == 0 {
		return nil, nil
	}

	// Run authentication in parallel
	var wg sync.WaitGroup
	results := make(chan AuthenticateResult, len(plugins))
	configHashes := make(map[string]string)
	var configHashesMu sync.Mutex

	for name, plugin := range plugins {
		// Check if we have valid cached credentials
		m.mu.RLock()
		creds, hasCreds := m.credentials[name]
		m.mu.RUnlock()

		if hasCreds && !creds.IsExpired() {
			// Use cached credentials
			results <- AuthenticateResult{
				PluginName:  name,
				Credentials: creds,
			}
			continue
		}

		// Need to authenticate
		wg.Add(1)
		go func(name string, plugin *Plugin) {
			defer wg.Done()

			result, hash := m.authenticateWithHash(ctx, name, plugin, programName, stackName, p5Config, workDir)
			if hash != "" {
				configHashesMu.Lock()
				configHashes[name] = hash
				configHashesMu.Unlock()
			}
			results <- result
		}(name, plugin)
	}

	// Wait for all authentications to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var allResults []AuthenticateResult
	for result := range results {
		allResults = append(allResults, result)

		// Cache successful credentials
		if result.Error == nil && result.Credentials != nil {
			m.mu.Lock()
			m.credentials[result.PluginName] = result.Credentials
			m.mu.Unlock()
		}
	}

	// Update context with new config hashes
	m.UpdateContext(workDir, stackName, programName, configHashes)

	return allResults, nil
}

// authenticateWithHash runs authentication for a single plugin and returns the config hash
func (m *Manager) authenticateWithHash(ctx context.Context, name string, plugin *Plugin, programName, stackName string, p5Config *P5Config, workDir string) (AuthenticateResult, string) {
	// Get program-level config
	programConfig := make(map[string]any)
	if pluginCfg, ok := p5Config.Plugins[name]; ok {
		programConfig = pluginCfg.Config
	}

	// Get stack-level config
	stackConfig, err := LoadStackPluginConfig(workDir, stackName, name)
	if err != nil {
		return AuthenticateResult{
			PluginName: name,
			Error:      fmt.Errorf("failed to load stack config: %w", err),
		}, ""
	}
	if stackConfig == nil {
		stackConfig = make(map[string]any)
	}

	// Calculate config hash for change detection
	cfgHash := hashConfig(programConfig, stackConfig)

	// Convert configs to string maps for TinyGo WASM compatibility
	programConfigStr := convertToStringMap(programConfig)
	stackConfigStr := convertToStringMap(stackConfig)

	// Call the plugin
	input := AuthInput{
		ProgramConfig: programConfigStr,
		StackConfig:   stackConfigStr,
		StackName:     stackName,
		ProgramName:   programName,
	}

	output, err := plugin.Authenticate(ctx, input)
	if err != nil {
		return AuthenticateResult{
			PluginName: name,
			Error:      err,
		}, cfgHash
	}

	if !output.Success {
		return AuthenticateResult{
			PluginName: name,
			Error:      fmt.Errorf("authentication failed: %s", output.Error),
		}, cfgHash
	}

	// Calculate expiration
	creds := &Credentials{
		PluginName: name,
		Env:        output.Env,
	}

	if output.TTLSeconds < 0 {
		creds.AlwaysCall = true
	} else if output.TTLSeconds > 0 {
		creds.ExpiresAt = time.Now().Add(time.Duration(output.TTLSeconds) * time.Second)
	}
	// TTLSeconds == 0 means never expires (ExpiresAt stays zero)

	return AuthenticateResult{
		PluginName:  name,
		Credentials: creds,
	}, cfgHash
}

// GetAllEnv returns all environment variables from all valid credentials
func (m *Manager) GetAllEnv() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	env := make(map[string]string)
	for _, creds := range m.credentials {
		if !creds.IsExpired() || creds.AlwaysCall {
			for k, v := range creds.Env {
				env[k] = v
			}
		}
	}
	return env
}

// ApplyEnvToProcess sets all credential env vars in the current process environment
// This allows subsequent Pulumi operations (which use os.Environ) to inherit them
func (m *Manager) ApplyEnvToProcess() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, creds := range m.credentials {
		if !creds.IsExpired() || creds.AlwaysCall {
			for k, v := range creds.Env {
				os.Setenv(k, v)
			}
		}
	}
}

// InvalidateCredentials marks credentials for a specific plugin as expired
func (m *Manager) InvalidateCredentials(pluginName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.credentials, pluginName)
}

// InvalidateAllCredentials clears all cached credentials
func (m *Manager) InvalidateAllCredentials() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.credentials = make(map[string]*Credentials)
}

// Close cleans up all plugin resources
func (m *Manager) Close(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, p := range m.plugins {
		p.Close(ctx)
	}
	m.plugins = make(map[string]*Plugin)
	m.credentials = make(map[string]*Credentials)
}

// CredentialsSummary returns a summary of current credentials for display
type CredentialsSummary struct {
	PluginName string
	EnvVars    []string  // List of env var names (not values for security)
	ExpiresAt  time.Time // Zero if never expires
	AlwaysCall bool
	HasError   bool
	Error      string
}

// GetCredentialsSummary returns a summary of all credentials for UI display
func (m *Manager) GetCredentialsSummary() []CredentialsSummary {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var summaries []CredentialsSummary
	for name, creds := range m.credentials {
		envVars := make([]string, 0, len(creds.Env))
		for k := range creds.Env {
			envVars = append(envVars, k)
		}

		summaries = append(summaries, CredentialsSummary{
			PluginName: name,
			EnvVars:    envVars,
			ExpiresAt:  creds.ExpiresAt,
			AlwaysCall: creds.AlwaysCall,
		})
	}

	return summaries
}

// LoadAndAuthenticate is a convenience method that loads plugins and authenticates
func (m *Manager) LoadAndAuthenticate(ctx context.Context, workDir, programName, stackName string) ([]AuthenticateResult, error) {
	// Load global config from p5.toml (git root or launch dir)
	globalConfig, globalPath, err := LoadGlobalConfig(m.launchDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load global config: %w", err)
	}
	m.globalConfigPath = globalPath

	// Load p5 config from Pulumi.yaml
	pulumiYamlPath := filepath.Join(workDir, "Pulumi.yaml")
	p5Config, err := LoadP5Config(pulumiYamlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load p5 config: %w", err)
	}

	// Merge configs (global as base, program overrides)
	mergedConfig := MergeConfigs(globalConfig, p5Config)
	m.mergedConfig = mergedConfig

	if len(mergedConfig.Plugins) == 0 {
		return nil, nil // No plugins configured
	}

	// Load plugins
	if err := m.LoadPlugins(ctx, mergedConfig); err != nil {
		return nil, err
	}

	// Authenticate
	return m.AuthenticateAll(ctx, programName, stackName, mergedConfig, workDir)
}

// hashConfig creates a hash of the plugin configuration for change detection
func hashConfig(programConfig, stackConfig map[string]interface{}) string {
	combined := map[string]interface{}{
		"program": programConfig,
		"stack":   stackConfig,
	}
	data, _ := json.Marshal(combined)
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash[:8]) // Use first 8 bytes for brevity
}

// convertToStringMap converts a map[string]interface{} to map[string]string
// This is needed for TinyGo WASM compatibility (TinyGo has issues with interface{} in JSON)
func convertToStringMap(m map[string]interface{}) map[string]string {
	result := make(map[string]string)
	for k, v := range m {
		switch val := v.(type) {
		case string:
			result[k] = val
		case int, int64, float64, bool:
			result[k] = fmt.Sprintf("%v", val)
		default:
			// For complex types, marshal to JSON
			if data, err := json.Marshal(val); err == nil {
				result[k] = string(data)
			}
		}
	}
	return result
}

// ShouldRefreshCredentials determines if credentials should be refreshed for a plugin
// based on context changes and refresh trigger settings
func (m *Manager) ShouldRefreshCredentials(pluginName string, newWorkDir, newStackName, newProgramName string, newProgramConfig, newStackConfig map[string]interface{}) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// No current context means we haven't authenticated yet
	if m.currentContext == nil {
		return true
	}

	// Get plugin config to check refresh triggers
	var refreshTrigger *RefreshTrigger
	if m.mergedConfig != nil {
		if cfg, ok := m.mergedConfig.Plugins[pluginName]; ok {
			refreshTrigger = cfg.Refresh
		}
	}

	// Check workspace change
	workspaceChanged := m.currentContext.WorkDir != newWorkDir
	if workspaceChanged {
		if refreshTrigger.ShouldRefreshOnWorkspaceChange() {
			// If OnConfigChange is also set, only refresh if config changed
			if refreshTrigger.ShouldRefreshOnConfigChange() {
				newHash := hashConfig(newProgramConfig, newStackConfig)
				oldHash := m.currentContext.ConfigHashes[pluginName]
				return newHash != oldHash
			}
			return true
		}
	}

	// Check stack change
	stackChanged := m.currentContext.StackName != newStackName
	if stackChanged {
		if refreshTrigger.ShouldRefreshOnStackChange() {
			// If OnConfigChange is also set, only refresh if config changed
			if refreshTrigger.ShouldRefreshOnConfigChange() {
				newHash := hashConfig(newProgramConfig, newStackConfig)
				oldHash := m.currentContext.ConfigHashes[pluginName]
				return newHash != oldHash
			}
			return true
		}
	}

	// If neither workspace nor stack changed, check if config changed (for TTL-based refresh)
	// This is handled by the normal TTL logic, not here

	return false
}

// InvalidateCredentialsForContext invalidates credentials based on context change
// and plugin refresh trigger settings
func (m *Manager) InvalidateCredentialsForContext(workDir, stackName, programName string, p5Config *P5Config) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.currentContext == nil {
		return
	}

	workspaceChanged := m.currentContext.WorkDir != workDir
	stackChanged := m.currentContext.StackName != stackName

	// Check each plugin's refresh triggers
	for pluginName := range m.credentials {
		var refreshTrigger *RefreshTrigger
		if p5Config != nil {
			if cfg, ok := p5Config.Plugins[pluginName]; ok {
				refreshTrigger = cfg.Refresh
			}
		}

		shouldInvalidate := false

		if workspaceChanged && refreshTrigger.ShouldRefreshOnWorkspaceChange() {
			shouldInvalidate = true
		}

		if stackChanged && refreshTrigger.ShouldRefreshOnStackChange() {
			shouldInvalidate = true
		}

		if shouldInvalidate {
			delete(m.credentials, pluginName)
		}
	}
}

// UpdateContext updates the current authentication context
func (m *Manager) UpdateContext(workDir, stackName, programName string, configHashes map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.currentContext = &AuthContext{
		WorkDir:      workDir,
		StackName:    stackName,
		ProgramName:  programName,
		ConfigHashes: configHashes,
	}
}

// GetMergedConfig returns the current merged configuration
func (m *Manager) GetMergedConfig() *P5Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.mergedConfig
}

// GetGlobalConfigPath returns the path to the loaded global config file
func (m *Manager) GetGlobalConfigPath() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.globalConfigPath
}
