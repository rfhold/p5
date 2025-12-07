package main

import (
	"fmt"
	"os"

	"github.com/rfhold/p5/internal/plugins"
	"github.com/rfhold/p5/internal/pulumi"
)

// Dependencies holds all external dependencies for the application.
// These can be replaced with test doubles for unit testing.
type Dependencies struct {
	StackOperator    pulumi.StackOperator
	StackReader      pulumi.StackReader
	WorkspaceReader  pulumi.WorkspaceReader
	StackInitializer pulumi.StackInitializer
	ResourceImporter pulumi.ResourceImporter
	PluginProvider   plugins.PluginProvider
	Env              map[string]string // Environment variables to pass to Pulumi
}

// NewProductionDependencies creates dependencies configured for production use.
// workDir is used to initialize the plugin manager for p5.toml discovery.
func NewProductionDependencies(workDir string) (*Dependencies, error) {
	pluginMgr, err := plugins.NewManager(workDir)
	if err != nil {
		// Log but don't fail - plugins are optional
		fmt.Fprintf(os.Stderr, "Warning: failed to initialize plugin manager: %v\n", err)
		// Continue with nil plugin manager - app should still work without plugins
	}

	return &Dependencies{
		StackOperator:    pulumi.NewStackOperator(),
		StackReader:      pulumi.NewStackReader(),
		WorkspaceReader:  pulumi.NewWorkspaceReader(),
		StackInitializer: pulumi.NewStackInitializer(),
		ResourceImporter: pulumi.NewResourceImporter(),
		PluginProvider:   pluginMgr,
	}, nil
}
