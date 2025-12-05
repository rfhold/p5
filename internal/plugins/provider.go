package plugins

import "context"

// AuthProvider handles authentication credentials from plugins.
type AuthProvider interface {
	// GetMergedAuthEnv returns combined auth environment variables from all plugins.
	GetMergedAuthEnv() map[string]string

	// GetAllEnv returns all environment variables from all valid credentials.
	GetAllEnv() map[string]string

	// ApplyEnvToProcess sets all credential env vars in the current process environment.
	ApplyEnvToProcess()

	// GetCredentialsSummary returns a summary of all credentials for UI display.
	GetCredentialsSummary() []CredentialsSummary

	// InvalidateCredentials marks credentials for a specific plugin as expired.
	InvalidateCredentials(pluginName string)

	// InvalidateAllCredentials clears all cached credentials.
	InvalidateAllCredentials()
}

// ImportHelper provides import ID suggestions.
type ImportHelper interface {
	// GetImportSuggestions queries plugins for import ID suggestions.
	GetImportSuggestions(ctx context.Context, req *ImportSuggestionsRequest) ([]*AggregatedImportSuggestion, error)

	// HasImportHelpers returns true if any plugin provides import suggestions.
	HasImportHelpers() bool
}

// PluginProvider combines all plugin capabilities needed by the application.
// This is the main interface used by the TUI to interact with the plugin system.
type PluginProvider interface {
	AuthProvider
	ImportHelper

	// Initialize loads and authenticates plugins based on the current context.
	// This is a convenience method that loads plugins from config and authenticates.
	Initialize(ctx context.Context, workDir, programName, stackName string) ([]AuthenticateResult, error)

	// Close cleans up plugin resources.
	Close(ctx context.Context)

	// GetMergedConfig returns the merged plugin configuration.
	GetMergedConfig() *P5Config

	// ShouldRefreshCredentials determines if credentials should be refreshed for a plugin
	// based on context changes and refresh trigger settings.
	ShouldRefreshCredentials(pluginName string, newWorkDir, newStackName, newProgramName string, newProgramConfig, newStackConfig map[string]any) bool

	// InvalidateCredentialsForContext invalidates credentials based on context change
	// and plugin refresh trigger settings.
	InvalidateCredentialsForContext(workDir, stackName, programName string, p5Config *P5Config)

	// AuthenticateAll runs authentication for all loaded plugins.
	AuthenticateAll(ctx context.Context, programName, stackName string, p5Config *P5Config, workDir string) ([]AuthenticateResult, error)
}
