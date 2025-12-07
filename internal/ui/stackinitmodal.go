package ui

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/rfhold/p5/internal/pulumi"
)

// StackInitModal wraps StepModal with stack initialization-specific logic
type StackInitModal struct {
	*StepModal

	// Backend info from pulumi whoami
	backendUser string
	backendURL  string

	// Stack files found in workspace
	stackFiles []pulumi.StackFileInfo

	// Track which stack has encryption configured
	stacksWithEncryption map[string]bool

	// Default secrets providers
	defaultProviders []StepSuggestion

	// Auth environment from plugins (may contain PULUMI_CONFIG_PASSPHRASE)
	authEnv map[string]string
}

const (
	stepStackName       = 0
	stepSecretsProvider = 1
	stepPassphrase      = 2
)

// NewStackInitModal creates a new stack init modal
func NewStackInitModal() *StackInitModal {
	m := &StackInitModal{
		StepModal:            NewStepModal("Initialize Stack"),
		stacksWithEncryption: make(map[string]bool),
		defaultProviders: []StepSuggestion{
			{ID: "passphrase", Label: "passphrase", Description: "Default passphrase-based encryption"},
			{ID: "awskms://alias/pulumi", Label: "awskms://alias/pulumi", Description: "AWS KMS"},
			{ID: "azurekeyvault://", Label: "azurekeyvault://...", Description: "Azure Key Vault"},
			{ID: "gcpkms://", Label: "gcpkms://...", Description: "Google Cloud KMS"},
			{ID: "hashivault://", Label: "hashivault://...", Description: "HashiCorp Vault"},
		},
	}

	// Configure the steps
	m.configureSteps()

	return m
}

// configureSteps sets up the modal steps
func (m *StackInitModal) configureSteps() {
	steps := []StepModalStep{
		{
			Title:            "Select or enter stack name",
			InputLabel:       "Stack name",
			InputPlaceholder: "Enter stack name...",
		},
		{
			Title:            "Select secrets provider",
			InputLabel:       "Provider URL",
			InputPlaceholder: "Enter provider URL...",
		},
		{
			Title:            "Enter passphrase",
			InputLabel:       "Passphrase",
			InputPlaceholder: "Enter passphrase for encrypting secrets...",
			PasswordMode:     true,
		},
	}

	m.SetSteps(steps)
}

// Show shows the modal and resets state
func (m *StackInitModal) Show() {
	m.StepModal.Show()
	m.configureSteps()
	m.updateBackendInfo()
}

// SetBackendInfo sets the backend connection information
func (m *StackInitModal) SetBackendInfo(user, url string) {
	m.backendUser = user
	m.backendURL = url
	m.updateBackendInfo()
}

// SetAuthEnv sets the auth environment from plugins
func (m *StackInitModal) SetAuthEnv(env map[string]string) {
	m.authEnv = env
}

// SetStackFiles sets the available stack files
func (m *StackInitModal) SetStackFiles(files []pulumi.StackFileInfo) {
	m.stackFiles = files
	m.stacksWithEncryption = make(map[string]bool)

	// Build suggestions from stack files
	suggestions := make([]StepSuggestion, 0, len(files))
	for _, f := range files {
		s := StepSuggestion{
			ID:     f.Name,
			Label:  f.Name,
			Source: "from Pulumi." + f.Name + ".yaml",
		}
		if f.HasEncryption {
			s.Warning = "has existing encryption"
			m.stacksWithEncryption[f.Name] = true
		}
		suggestions = append(suggestions, s)
	}

	m.SetStepSuggestions(stepStackName, suggestions)
	m.updateSecretsProviderSuggestions()
}

// updateBackendInfo updates the info lines for step 1
func (m *StackInitModal) updateBackendInfo() {
	info := []InfoLine{}
	if m.backendURL != "" {
		info = append(info, InfoLine{Label: "Backend", Value: m.backendURL})
	}
	if m.backendUser != "" {
		info = append(info, InfoLine{Label: "User", Value: m.backendUser})
	}
	m.SetStepInfoLines(stepStackName, info)
}

// updateSecretsProviderSuggestions builds the secrets provider suggestions list
func (m *StackInitModal) updateSecretsProviderSuggestions() {
	// Collect unique providers from existing stack files
	seenProviders := make(map[string]bool)
	var suggestions []StepSuggestion

	// Add providers found in stack files first
	for _, f := range m.stackFiles {
		if f.SecretsProvider != "" && !seenProviders[f.SecretsProvider] {
			seenProviders[f.SecretsProvider] = true
			suggestions = append(suggestions, StepSuggestion{
				ID:     f.SecretsProvider,
				Label:  f.SecretsProvider,
				Source: "from Pulumi." + f.Name + ".yaml",
			})
		}
	}

	// Add default providers that haven't been seen
	for _, p := range m.defaultProviders {
		if !seenProviders[p.ID] {
			suggestions = append(suggestions, p)
		}
	}

	m.SetStepSuggestions(stepSecretsProvider, suggestions)
}

// Update handles key events and manages step transitions
func (m *StackInitModal) Update(msg tea.KeyMsg) (StepModalAction, tea.Cmd) {
	action, cmd := m.StepModal.Update(msg)

	// Handle step transitions
	if action == StepModalActionNext {
		m.onStepTransition()
	}

	return action, cmd
}

// onStepTransition handles updates needed when moving between steps
func (m *StackInitModal) onStepTransition() {
	currentStep := m.CurrentStep()

	switch currentStep {
	case stepSecretsProvider:
		// Update info for step 2 with selected stack
		stackName := m.GetResult(stepStackName)
		info := []InfoLine{
			{Label: "Stack", Value: stackName},
		}
		m.SetStepInfoLines(stepSecretsProvider, info)

		// Set warning if stack has existing encryption
		if m.stacksWithEncryption[stackName] {
			m.SetStepWarning(stepSecretsProvider,
				"Stack '"+stackName+"' already has encryption configured. Re-initializing may cause issues with existing secrets.")
		} else {
			m.SetStepWarning(stepSecretsProvider, "")
		}

	case stepPassphrase:
		// Update info for step 3
		stackName := m.GetResult(stepStackName)
		provider := m.GetResult(stepSecretsProvider)
		info := []InfoLine{
			{Label: "Stack", Value: stackName},
			{Label: "Secrets Provider", Value: provider},
		}
		m.SetStepInfoLines(stepPassphrase, info)
	}
}

// NeedsPassphrase returns true if the passphrase step should be shown
func (m *StackInitModal) NeedsPassphrase() bool {
	provider := m.GetResult(stepSecretsProvider)
	// Passphrase provider needs a passphrase, unless env var is set
	// Note: empty string "" is a valid passphrase, so we check if the key exists at all
	if provider == "passphrase" || provider == "" {
		// Check system environment first
		if _, exists := os.LookupEnv("PULUMI_CONFIG_PASSPHRASE"); exists {
			return false
		}
		// Check auth plugin environment
		if m.authEnv != nil {
			if _, exists := m.authEnv["PULUMI_CONFIG_PASSPHRASE"]; exists {
				return false
			}
		}
		return true
	}
	return false
}

// ShouldSkipPassphrase returns true if we should skip the passphrase step
func (m *StackInitModal) ShouldSkipPassphrase() bool {
	return !m.NeedsPassphrase()
}

// GetStackName returns the selected/entered stack name
func (m *StackInitModal) GetStackName() string {
	return m.GetResult(stepStackName)
}

// GetSecretsProvider returns the selected/entered secrets provider
func (m *StackInitModal) GetSecretsProvider() string {
	return m.GetResult(stepSecretsProvider)
}

// GetPassphrase returns the entered passphrase
func (m *StackInitModal) GetPassphrase() string {
	return m.GetResult(stepPassphrase)
}

// IsComplete returns true if all required steps have been completed
func (m *StackInitModal) IsComplete() bool {
	// Need stack name
	if m.GetStackName() == "" {
		return false
	}
	// Need secrets provider
	if m.GetSecretsProvider() == "" {
		return false
	}
	// Need passphrase if using passphrase provider without env var
	if m.NeedsPassphrase() && m.GetPassphrase() == "" {
		return false
	}
	return true
}
