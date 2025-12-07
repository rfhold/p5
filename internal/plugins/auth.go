package plugins

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rfhold/p5/internal/plugins/proto"
)

var ErrAuthenticationFailed = errors.New("authentication failed")

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

// AuthenticateResult holds the result of an authentication attempt
type AuthenticateResult struct {
	PluginName  string
	Credentials *Credentials
	Error       error
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

// AuthenticateAll runs authentication for all plugins in parallel
func (m *Manager) AuthenticateAll(ctx context.Context, programName, stackName string, p5Config *P5Config, workDir string) ([]AuthenticateResult, error) {
	m.mu.RLock()
	plugins := make(map[string]*PluginInstance, len(m.plugins))
	maps.Copy(plugins, m.plugins)
	m.mu.RUnlock()

	if len(plugins) == 0 {
		return nil, nil
	}

	// Run authentication in parallel
	var wg sync.WaitGroup
	results := make(chan AuthenticateResult, len(plugins))
	configHashes := make(map[string]string)
	var configHashesMu sync.Mutex

	for name, pluginInst := range plugins {
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
		go func(name string, pluginInst *PluginInstance) {
			defer wg.Done()

			result, hash := m.authenticateWithHash(ctx, name, pluginInst, programName, stackName, p5Config, workDir)
			if hash != "" {
				configHashesMu.Lock()
				configHashes[name] = hash
				configHashesMu.Unlock()
			}
			results <- result
		}(name, pluginInst)
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
func (m *Manager) authenticateWithHash(ctx context.Context, name string, pluginInst *PluginInstance, programName, stackName string, p5Config *P5Config, workDir string) (result AuthenticateResult, configHash string) {
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

	// Convert configs to string maps for gRPC
	programConfigStr := convertToStringMap(programConfig)
	stackConfigStr := convertToStringMap(stackConfig)

	// Call the plugin
	req := &proto.AuthenticateRequest{
		ProgramConfig: programConfigStr,
		StackConfig:   stackConfigStr,
		StackName:     stackName,
		ProgramName:   programName,
	}

	resp, err := pluginInst.auth.Authenticate(ctx, req)
	if err != nil {
		return AuthenticateResult{
			PluginName: name,
			Error:      err,
		}, cfgHash
	}

	if !resp.Success {
		return AuthenticateResult{
			PluginName: name,
			Error:      fmt.Errorf("%w: %s", ErrAuthenticationFailed, resp.Error),
		}, cfgHash
	}

	// Calculate expiration
	creds := &Credentials{
		PluginName: name,
		Env:        resp.Env,
	}

	if resp.TtlSeconds < 0 {
		creds.AlwaysCall = true
	} else if resp.TtlSeconds > 0 {
		creds.ExpiresAt = time.Now().Add(time.Duration(resp.TtlSeconds) * time.Second)
	}
	// TtlSeconds == 0 means never expires (ExpiresAt stays zero)

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
			maps.Copy(env, creds.Env)
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
	// Load global config from p5.toml (git root or workDir)
	// Use workDir instead of launchDir so that when a workspace is selected via UI,
	// we find p5.toml relative to that workspace rather than where p5 was launched
	globalConfig, globalPath, err := LoadGlobalConfig(workDir)
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
func hashConfig(programConfig, stackConfig map[string]any) string {
	combined := map[string]any{
		"program": programConfig,
		"stack":   stackConfig,
	}
	data, _ := json.Marshal(combined)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:8]) // Use first 8 bytes for brevity
}

// convertToStringMap converts a map[string]any to map[string]string
func convertToStringMap(m map[string]any) map[string]string {
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
