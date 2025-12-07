package plugins

import (
	"context"
	"errors"
	"fmt"
	"os/exec"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
)

var (
	ErrBuiltinPluginNotFound    = errors.New("builtin plugin not found")
	ErrExternalPluginCmdMissing = errors.New("cmd is required for external plugins (not a builtin)")
	ErrNotAuthPlugin            = errors.New("plugin does not implement AuthPlugin interface")
)

// PluginInstance holds a running plugin client and its interface
type PluginInstance struct {
	name           string
	client         *plugin.Client // nil for builtin plugins
	auth           AuthPlugin
	importHelper   ImportHelperPlugin   // nil if not supported or not enabled
	resourceOpener ResourceOpenerPlugin // nil if not supported or not enabled
	builtin        bool                 // true if this is a builtin plugin
}

// HasImportHelper returns true if this plugin provides import suggestions
func (p *PluginInstance) HasImportHelper() bool {
	return p.importHelper != nil
}

// HasResourceOpener returns true if this plugin provides resource opening capabilities
func (p *PluginInstance) HasResourceOpener() bool {
	return p.resourceOpener != nil
}

// Close shuts down the plugin
func (p *PluginInstance) Close() {
	// Only external plugins have a client to kill
	if p.client != nil {
		p.client.Kill()
	}
}

// LoadPlugins loads all plugins defined in the p5 config
func (m *Manager) LoadPlugins(ctx context.Context, p5Config *P5Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Close any existing plugins
	for _, p := range m.plugins {
		p.Close()
	}
	m.plugins = make(map[string]*PluginInstance)

	// Load each plugin
	for name, pluginConfig := range p5Config.Plugins {
		// Check if this is a builtin plugin
		if IsBuiltin(name) {
			if err := m.loadBuiltinPlugin(name, pluginConfig); err != nil {
				return fmt.Errorf("failed to load builtin plugin %s: %w", name, err)
			}
		} else {
			if err := m.loadPlugin(ctx, name, pluginConfig); err != nil {
				return fmt.Errorf("failed to load plugin %s: %w", name, err)
			}
		}
	}

	return nil
}

// loadBuiltinPlugin loads a builtin plugin by name
func (m *Manager) loadBuiltinPlugin(name string, config PluginConfig) error {
	builtinPlugin := GetBuiltin(name)
	if builtinPlugin == nil {
		return fmt.Errorf("%w: %s", ErrBuiltinPluginNotFound, name)
	}

	instance := &PluginInstance{
		name:    name,
		client:  nil,
		auth:    builtinPlugin,
		builtin: true,
	}

	// Check if plugin implements ImportHelperPlugin and is enabled
	if config.ImportHelper {
		if importHelper, ok := builtinPlugin.(ImportHelperPlugin); ok {
			instance.importHelper = importHelper
		}
	}

	// Check if plugin implements ResourceOpenerPlugin and is enabled
	if config.ResourceOpener {
		if resourceOpener, ok := builtinPlugin.(ResourceOpenerPlugin); ok {
			instance.resourceOpener = resourceOpener
		}
	}

	m.plugins[name] = instance
	return nil
}

// loadPlugin loads a single external plugin using go-plugin
func (m *Manager) loadPlugin(ctx context.Context, name string, config PluginConfig) error {
	if config.Cmd == "" {
		return fmt.Errorf("plugin %s: %w", name, ErrExternalPluginCmdMissing)
	}

	// Create a logger that discards output (plugins should be quiet)
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "plugin",
		Output: hclog.DefaultOutput,
		Level:  hclog.Warn,
	})

	// Build the command
	cmd := exec.CommandContext(ctx, config.Cmd, config.Args...) //nolint:gosec // G204: Plugin command comes from user config

	// Create the plugin client
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: Handshake,
		Plugins:         PluginMap,
		Cmd:             cmd,
		Logger:          logger,
		AllowedProtocols: []plugin.Protocol{
			plugin.ProtocolGRPC,
		},
	})

	// Connect to the plugin
	rpcClient, err := client.Client()
	if err != nil {
		client.Kill()
		return fmt.Errorf("failed to connect to plugin: %w", err)
	}

	// Request the auth plugin
	raw, err := rpcClient.Dispense("auth")
	if err != nil {
		client.Kill()
		return fmt.Errorf("failed to dispense auth plugin: %w", err)
	}

	authPlugin, ok := raw.(AuthPlugin)
	if !ok {
		client.Kill()
		return ErrNotAuthPlugin
	}

	instance := &PluginInstance{
		name:   name,
		client: client,
		auth:   authPlugin,
	}

	// Try to load import helper if enabled in config
	if config.ImportHelper {
		rawImportHelper, err := rpcClient.Dispense("import_helper")
		if err == nil {
			if importHelper, ok := rawImportHelper.(ImportHelperPlugin); ok {
				instance.importHelper = importHelper
			}
		}
		// If dispensing fails, just continue without import helper capability
	}

	// Try to load resource opener if enabled in config
	if config.ResourceOpener {
		rawResourceOpener, err := rpcClient.Dispense("resource_opener")
		if err == nil {
			if resourceOpener, ok := rawResourceOpener.(ResourceOpenerPlugin); ok {
				instance.resourceOpener = resourceOpener
			}
		}
		// If dispensing fails, just continue without resource opener capability
	}

	m.plugins[name] = instance
	return nil
}
