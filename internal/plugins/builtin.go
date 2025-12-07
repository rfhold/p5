package plugins

import (
	"context"

	p5plugin "github.com/rfhold/p5/pkg/plugin"

	"github.com/rfhold/p5/internal/plugins/proto"
)

// BuiltinPlugin is an AuthPlugin that runs in-process (no subprocess/gRPC)
type BuiltinPlugin interface {
	AuthPlugin
	// Name returns the plugin's registered name (e.g., "env")
	Name() string
}

// BuiltinImportHelperPlugin is for builtin plugins that also provide import suggestions
type BuiltinImportHelperPlugin interface {
	BuiltinPlugin
	ImportHelperPlugin
}

// BuiltinResourceOpenerPlugin is for builtin plugins that also provide resource opening capabilities
type BuiltinResourceOpenerPlugin interface {
	BuiltinPlugin
	ResourceOpenerPlugin
}

// builtinRegistry holds all registered builtin plugins
var builtinRegistry = make(map[string]BuiltinPlugin)

// RegisterBuiltin registers a builtin plugin by name
func RegisterBuiltin(plugin BuiltinPlugin) {
	builtinRegistry[plugin.Name()] = plugin
}

// GetBuiltin returns a builtin plugin by name, or nil if not found
func GetBuiltin(name string) BuiltinPlugin {
	return builtinRegistry[name]
}

// IsBuiltin returns true if a plugin name refers to a builtin plugin
func IsBuiltin(name string) bool {
	_, ok := builtinRegistry[name]
	return ok
}

// ListBuiltins returns all registered builtin plugin names
func ListBuiltins() []string {
	names := make([]string, 0, len(builtinRegistry))
	for name := range builtinRegistry {
		names = append(names, name)
	}
	return names
}

// BuiltinPluginInstance wraps a builtin plugin to satisfy the plugin instance interface
type BuiltinPluginInstance struct {
	name   string
	plugin BuiltinPlugin
}

// Authenticate delegates to the builtin plugin
func (b *BuiltinPluginInstance) Authenticate(ctx context.Context, req *proto.AuthenticateRequest) (*proto.AuthenticateResponse, error) {
	return b.plugin.Authenticate(ctx, req)
}

// NewBuiltinPluginInstance creates a new instance wrapper for a builtin plugin
func NewBuiltinPluginInstance(name string, plugin BuiltinPlugin) *BuiltinPluginInstance {
	return &BuiltinPluginInstance{
		name:   name,
		plugin: plugin,
	}
}

// BuiltinPluginBase provides common functionality for builtin plugins
type BuiltinPluginBase struct {
	name string
}

// Name returns the plugin name
func (b *BuiltinPluginBase) Name() string {
	return b.name
}

// NewBuiltinPluginBase creates a new base for builtin plugins
func NewBuiltinPluginBase(name string) BuiltinPluginBase {
	return BuiltinPluginBase{name: name}
}

// SuccessResponse is re-exported from pkg/plugin for internal use.
var SuccessResponse = p5plugin.SuccessResponse

// ErrorResponse is re-exported from pkg/plugin for internal use.
var ErrorResponse = p5plugin.ErrorResponse
