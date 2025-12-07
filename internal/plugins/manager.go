package plugins

import (
	"context"
	"maps"
	"sync"
)

// Compile-time check that Manager implements PluginProvider
var _ PluginProvider = (*Manager)(nil)

// Manager handles plugin lifecycle and credential management
type Manager struct {
	mu          sync.RWMutex
	plugins     map[string]*PluginInstance
	credentials map[string]*Credentials

	// Track current context for change detection
	currentContext *AuthContext
	// Store merged config for refresh trigger checks
	mergedConfig *P5Config
	// Path to global config file (for logging/debugging)
	globalConfigPath string
	// Launch directory (for finding p5.toml)
	launchDir string
}

// NewManager creates a new plugin manager
// launchDir is the directory p5 was launched from (used to find p5.toml)
func NewManager(launchDir string) (*Manager, error) {
	return &Manager{
		plugins:     make(map[string]*PluginInstance),
		credentials: make(map[string]*Credentials),
		launchDir:   launchDir,
	}, nil
}

// Close cleans up all plugin resources
func (m *Manager) Close(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, p := range m.plugins {
		p.Close()
	}
	m.plugins = make(map[string]*PluginInstance)
	m.credentials = make(map[string]*Credentials)
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

// AggregatedImportSuggestion includes the source plugin name
type AggregatedImportSuggestion struct {
	PluginName string
	Suggestion *ImportSuggestion
}

// GetImportSuggestions queries all enabled import helper plugins for suggestions
func (m *Manager) GetImportSuggestions(ctx context.Context, req *ImportSuggestionsRequest) ([]*AggregatedImportSuggestion, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var results []*AggregatedImportSuggestion

	for name, instance := range m.plugins {
		if !instance.HasImportHelper() {
			continue
		}

		// Build the request with auth env if configured
		pluginReq := req

		// If use_auth_env is enabled for this plugin, populate auth_env
		if config, ok := m.mergedConfig.Plugins[name]; ok && config.UseAuthEnv {
			// Clone the request and add auth env
			pluginReq = &ImportSuggestionsRequest{
				ResourceType:   req.ResourceType,
				ResourceName:   req.ResourceName,
				ResourceUrn:    req.ResourceUrn,
				ParentUrn:      req.ParentUrn,
				Inputs:         req.Inputs,
				ProgramConfig:  req.ProgramConfig,
				StackConfig:    req.StackConfig,
				StackName:      req.StackName,
				ProgramName:    req.ProgramName,
				AuthEnv:        m.getMergedAuthEnvLocked(),
				ProviderUrn:    req.ProviderUrn,
				ProviderInputs: req.ProviderInputs,
			}
		}

		resp, err := instance.importHelper.GetImportSuggestions(ctx, pluginReq)
		if err != nil {
			// Log error but continue with other plugins
			continue
		}

		// Skip if plugin can't provide suggestions for this resource type
		if !resp.CanProvide {
			continue
		}

		// Skip if there was an error
		if resp.Error != "" {
			continue
		}

		// Add suggestions with plugin name
		for _, suggestion := range resp.Suggestions {
			results = append(results, &AggregatedImportSuggestion{
				PluginName: name,
				Suggestion: suggestion,
			})
		}
	}

	return results, nil
}

// GetMergedAuthEnv returns all auth environment variables from all plugins
func (m *Manager) GetMergedAuthEnv() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.getMergedAuthEnvLocked()
}

// getMergedAuthEnvLocked returns all auth environment variables from all plugins (must hold lock)
func (m *Manager) getMergedAuthEnvLocked() map[string]string {
	env := make(map[string]string)
	for _, creds := range m.credentials {
		if creds != nil && creds.Env != nil {
			maps.Copy(env, creds.Env)
		}
	}
	return env
}

// HasImportHelpers returns true if any plugin has import helper capability enabled
func (m *Manager) HasImportHelpers() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, instance := range m.plugins {
		if instance.HasImportHelper() {
			return true
		}
	}
	return false
}

// HasResourceOpeners returns true if any plugin has resource opener capability enabled
func (m *Manager) HasResourceOpeners() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, instance := range m.plugins {
		if instance.HasResourceOpener() {
			return true
		}
	}
	return false
}

// OpenResource queries all enabled resource opener plugins to get an action for opening the resource.
// Returns the first plugin that can handle the resource type, or nil if none can.
func (m *Manager) OpenResource(ctx context.Context, req *OpenResourceRequest) (resp *OpenResourceResponse, pluginName string, err error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for name, instance := range m.plugins {
		if !instance.HasResourceOpener() {
			continue
		}

		// Build the request with auth env if configured
		pluginReq := req

		// If use_auth_env is enabled for this plugin, populate auth_env
		if config, ok := m.mergedConfig.Plugins[name]; ok && config.UseAuthEnv {
			// Clone the request and add auth env
			pluginReq = &OpenResourceRequest{
				ResourceType:   req.ResourceType,
				ResourceName:   req.ResourceName,
				ResourceUrn:    req.ResourceUrn,
				ProviderUrn:    req.ProviderUrn,
				ProviderInputs: req.ProviderInputs,
				Inputs:         req.Inputs,
				Outputs:        req.Outputs,
				ProgramConfig:  req.ProgramConfig,
				StackConfig:    req.StackConfig,
				StackName:      req.StackName,
				ProgramName:    req.ProgramName,
				AuthEnv:        m.getMergedAuthEnvLocked(),
			}
		}

		resp, err := instance.resourceOpener.OpenResource(ctx, pluginReq)
		if err != nil {
			// Log error but continue with other plugins
			continue
		}

		// Skip if plugin can't open this resource type
		if !resp.CanOpen {
			continue
		}

		// Return first plugin that can open the resource
		return resp, name, nil
	}

	// No plugin can open this resource
	return nil, "", nil
}

// Initialize loads and authenticates plugins based on the current context.
// This is an alias for LoadAndAuthenticate to satisfy the PluginProvider interface.
func (m *Manager) Initialize(ctx context.Context, workDir, programName, stackName string) ([]AuthenticateResult, error) {
	return m.LoadAndAuthenticate(ctx, workDir, programName, stackName)
}
