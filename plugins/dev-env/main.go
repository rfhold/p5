package main

import (
	"encoding/json"

	"github.com/extism/go-pdk"
)

// AuthInput is the input passed to the plugin's authenticate function
// Using map[string]string for TinyGo WASM compatibility
type AuthInput struct {
	ProgramConfig map[string]string `json:"program_config"`
	StackConfig   map[string]string `json:"stack_config"`
	StackName     string            `json:"stack_name"`
	ProgramName   string            `json:"program_name"`
}

// AuthOutput is the output from the plugin's authenticate function
type AuthOutput struct {
	Success    bool              `json:"success"`
	Env        map[string]string `json:"env,omitempty"`
	TTLSeconds int               `json:"ttl_seconds,omitempty"`
	Error      string            `json:"error,omitempty"`
}

//go:wasmexport authenticate
func authenticate() int32 {
	// Get raw input bytes
	inputBytes := pdk.Input()

	var input AuthInput
	if err := json.Unmarshal(inputBytes, &input); err != nil {
		outputError("failed to parse input: " + err.Error())
		return 1
	}

	// Get values from config, with defaults
	backendURL := getStringConfig(input.ProgramConfig, "backendUrl", "file://~/.pulumi")
	passphrase := getStringConfig(input.ProgramConfig, "passphrase", "")

	// Allow stack config to override
	if stackBackend := getStringConfig(input.StackConfig, "backendUrl", ""); stackBackend != "" {
		backendURL = stackBackend
	}
	if stackPass := getStringConfig(input.StackConfig, "passphrase", ""); stackPass != "" {
		passphrase = stackPass
	}

	// Build env vars
	env := make(map[string]string)

	if backendURL != "" {
		env["PULUMI_BACKEND_URL"] = backendURL
	}

	// PULUMI_CONFIG_PASSPHRASE can be empty string (for local dev with no encryption)
	env["PULUMI_CONFIG_PASSPHRASE"] = passphrase

	output := AuthOutput{
		Success:    true,
		Env:        env,
		TTLSeconds: 0, // Never expires - these are static values
	}

	outputBytes, err := json.Marshal(output)
	if err != nil {
		outputError("failed to marshal output: " + err.Error())
		return 1
	}

	pdk.Output(outputBytes)
	return 0
}

func getStringConfig(config map[string]string, key, defaultVal string) string {
	if config == nil {
		return defaultVal
	}
	if val, ok := config[key]; ok {
		return val
	}
	return defaultVal
}

func outputError(msg string) {
	output := AuthOutput{
		Success: false,
		Error:   msg,
	}
	outputBytes, _ := json.Marshal(output)
	pdk.Output(outputBytes)
}

func main() {}
