package plugins

// AuthContext holds the current authentication context for change detection
type AuthContext struct {
	WorkDir     string
	StackName   string
	ProgramName string
	// ConfigHashes stores hash of (program + stack) config per plugin for change detection
	ConfigHashes map[string]string
}

// ShouldRefreshCredentials determines if credentials should be refreshed for a plugin
// based on context changes and refresh trigger settings
func (m *Manager) ShouldRefreshCredentials(pluginName string, newWorkDir, newStackName, newProgramName string, newProgramConfig, newStackConfig map[string]any) bool {
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
