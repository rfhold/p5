package plugins

import (
	p5plugin "github.com/rfhold/p5/pkg/plugin"
)

// Re-export shared types from pkg/plugin for internal use.
// This ensures a single source of truth for these definitions.
var (
	// Handshake is re-exported from pkg/plugin
	Handshake = p5plugin.Handshake
	// PluginMap is re-exported from pkg/plugin
	PluginMap = p5plugin.PluginMap
)

// Re-export gRPC types from pkg/plugin.
// These are the canonical implementations - do not duplicate.
type (
	// AuthPluginGRPC is the implementation of goplugin.GRPCPlugin for AuthPlugin
	AuthPluginGRPC = p5plugin.AuthPluginGRPC
	// GRPCClient is the client-side implementation of AuthPlugin over gRPC
	GRPCClient = p5plugin.GRPCClient
	// GRPCServer is the server-side implementation that wraps the actual plugin
	GRPCServer = p5plugin.GRPCServer
	// ImportHelperPluginGRPC is the implementation of goplugin.GRPCPlugin for ImportHelperPlugin
	ImportHelperPluginGRPC = p5plugin.ImportHelperPluginGRPC
	// ImportHelperGRPCClient is the client-side implementation of ImportHelperPlugin over gRPC
	ImportHelperGRPCClient = p5plugin.ImportHelperGRPCClient
	// ImportHelperGRPCServer is the server-side implementation that wraps the actual import helper plugin
	ImportHelperGRPCServer = p5plugin.ImportHelperGRPCServer
	// ResourceOpenerPluginGRPC is the implementation of goplugin.GRPCPlugin for ResourceOpenerPlugin
	ResourceOpenerPluginGRPC = p5plugin.ResourceOpenerPluginGRPC
	// ResourceOpenerGRPCClient is the client-side implementation of ResourceOpenerPlugin over gRPC
	ResourceOpenerGRPCClient = p5plugin.ResourceOpenerGRPCClient
	// ResourceOpenerGRPCServer is the server-side implementation that wraps the actual resource opener plugin
	ResourceOpenerGRPCServer = p5plugin.ResourceOpenerGRPCServer
)
