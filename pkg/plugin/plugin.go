// Package plugin provides shared types and helpers for p5 plugin authors.
// Plugin authors should import this package and implement the AuthPlugin interface.
//
// This package is the canonical source for shared plugin types used by both
// the p5 host and external plugin implementations.
package plugin

import (
	"context"
	"fmt"

	goplugin "github.com/hashicorp/go-plugin"
	"github.com/rfhold/p5/internal/plugins/proto"
	"google.golang.org/grpc"
)

// Re-export proto types for plugin authors
type (
	// AuthenticateRequest is the request sent to the Authenticate RPC
	AuthenticateRequest = proto.AuthenticateRequest
	// AuthenticateResponse is the response from the Authenticate RPC
	AuthenticateResponse = proto.AuthenticateResponse
	// ImportSuggestionsRequest is the request sent to the GetImportSuggestions RPC
	ImportSuggestionsRequest = proto.ImportSuggestionsRequest
	// ImportSuggestionsResponse is the response from the GetImportSuggestions RPC
	ImportSuggestionsResponse = proto.ImportSuggestionsResponse
	// ImportSuggestion represents a single import suggestion
	ImportSuggestion = proto.ImportSuggestion
)

// AuthPlugin is the interface that plugins must implement.
// This is the canonical definition used by both host and plugins.
type AuthPlugin interface {
	// Authenticate performs authentication and returns environment variables
	Authenticate(ctx context.Context, req *AuthenticateRequest) (*AuthenticateResponse, error)
}

// ImportHelperPlugin is an optional interface that plugins can implement
// to provide import ID suggestions for resources.
type ImportHelperPlugin interface {
	// GetImportSuggestions returns import ID suggestions for a resource.
	// Plugins should return CanProvide: false if they don't handle the resource type.
	GetImportSuggestions(ctx context.Context, req *ImportSuggestionsRequest) (*ImportSuggestionsResponse, error)
}

// Handshake is the handshake config for plugins.
// Both the host and plugin must agree on this configuration.
// This is the canonical definition - do not duplicate elsewhere.
var Handshake = goplugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "P5_PLUGIN",
	MagicCookieValue: "v0",
}

// PluginMap is the map of plugins we can dispense.
// This is the canonical definition used by both host and plugins.
var PluginMap = map[string]goplugin.Plugin{
	"auth":          &AuthPluginGRPC{},
	"import_helper": &ImportHelperPluginGRPC{},
}

// SuccessResponse creates a successful authentication response.
// This helper can be used by both builtin and external plugins.
func SuccessResponse(env map[string]string, ttlSeconds int32) *AuthenticateResponse {
	return &AuthenticateResponse{
		Success:    true,
		Env:        env,
		TtlSeconds: ttlSeconds,
	}
}

// ErrorResponse creates an error authentication response with format string support.
// This helper can be used by both builtin and external plugins.
func ErrorResponse(format string, args ...any) *AuthenticateResponse {
	return &AuthenticateResponse{
		Success: false,
		Error:   fmt.Sprintf(format, args...),
	}
}

// ImportSuggestionsNotSupported returns a response indicating the plugin doesn't handle this resource type.
func ImportSuggestionsNotSupported() *ImportSuggestionsResponse {
	return &ImportSuggestionsResponse{CanProvide: false}
}

// ImportSuggestionsSuccess creates a successful import suggestions response.
func ImportSuggestionsSuccess(suggestions []*ImportSuggestion) *ImportSuggestionsResponse {
	return &ImportSuggestionsResponse{
		CanProvide:  true,
		Suggestions: suggestions,
	}
}

// ImportSuggestionsError creates an error import suggestions response.
func ImportSuggestionsError(format string, args ...any) *ImportSuggestionsResponse {
	return &ImportSuggestionsResponse{
		CanProvide: true, // We can provide, but encountered an error
		Error:      fmt.Sprintf(format, args...),
	}
}

// NewImportSuggestion creates a new import suggestion.
func NewImportSuggestion(id, label, description string) *ImportSuggestion {
	return &ImportSuggestion{
		Id:          id,
		Label:       label,
		Description: description,
	}
}

// Serve starts the plugin server with the given implementation.
// This should be called from the plugin's main() function.
//
// Example:
//
//	func main() {
//	    plugin.Serve(&MyPlugin{})
//	}
func Serve(impl AuthPlugin) {
	plugins := map[string]goplugin.Plugin{
		"auth": &AuthPluginGRPC{Impl: impl},
	}

	// If the plugin also implements ImportHelperPlugin, register it
	if importHelper, ok := impl.(ImportHelperPlugin); ok {
		plugins["import_helper"] = &ImportHelperPluginGRPC{Impl: importHelper}
	}

	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: Handshake,
		Plugins:         plugins,
		GRPCServer:      goplugin.DefaultGRPCServer,
	})
}

// AuthPluginGRPC is the implementation of goplugin.GRPCPlugin for AuthPlugin
type AuthPluginGRPC struct {
	goplugin.Plugin
	// Impl is the actual plugin implementation
	Impl AuthPlugin
}

// GRPCServer registers the gRPC server (plugin side)
func (p *AuthPluginGRPC) GRPCServer(broker *goplugin.GRPCBroker, s *grpc.Server) error {
	proto.RegisterAuthPluginServer(s, &GRPCServer{Impl: p.Impl})
	return nil
}

// GRPCClient returns the gRPC client (host side)
func (p *AuthPluginGRPC) GRPCClient(ctx context.Context, broker *goplugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &GRPCClient{client: proto.NewAuthPluginClient(c)}, nil
}

// GRPCClient is the client-side implementation of AuthPlugin over gRPC
type GRPCClient struct {
	client proto.AuthPluginClient
}

// Authenticate calls the plugin's Authenticate RPC
func (c *GRPCClient) Authenticate(ctx context.Context, req *AuthenticateRequest) (*AuthenticateResponse, error) {
	return c.client.Authenticate(ctx, req)
}

// GRPCServer is the server-side implementation that wraps the actual plugin
type GRPCServer struct {
	proto.UnimplementedAuthPluginServer
	Impl AuthPlugin
}

// Authenticate handles the Authenticate RPC
func (s *GRPCServer) Authenticate(ctx context.Context, req *AuthenticateRequest) (*AuthenticateResponse, error) {
	return s.Impl.Authenticate(ctx, req)
}

// ImportHelperPluginGRPC is the implementation of goplugin.GRPCPlugin for ImportHelperPlugin
type ImportHelperPluginGRPC struct {
	goplugin.Plugin
	// Impl is the actual plugin implementation
	Impl ImportHelperPlugin
}

// GRPCServer registers the gRPC server (plugin side)
func (p *ImportHelperPluginGRPC) GRPCServer(broker *goplugin.GRPCBroker, s *grpc.Server) error {
	proto.RegisterImportHelperPluginServer(s, &ImportHelperGRPCServer{Impl: p.Impl})
	return nil
}

// GRPCClient returns the gRPC client (host side)
func (p *ImportHelperPluginGRPC) GRPCClient(ctx context.Context, broker *goplugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &ImportHelperGRPCClient{client: proto.NewImportHelperPluginClient(c)}, nil
}

// ImportHelperGRPCClient is the client-side implementation of ImportHelperPlugin over gRPC
type ImportHelperGRPCClient struct {
	client proto.ImportHelperPluginClient
}

// GetImportSuggestions calls the plugin's GetImportSuggestions RPC
func (c *ImportHelperGRPCClient) GetImportSuggestions(ctx context.Context, req *ImportSuggestionsRequest) (*ImportSuggestionsResponse, error) {
	return c.client.GetImportSuggestions(ctx, req)
}

// ImportHelperGRPCServer is the server-side implementation that wraps the actual plugin
type ImportHelperGRPCServer struct {
	proto.UnimplementedImportHelperPluginServer
	Impl ImportHelperPlugin
}

// GetImportSuggestions handles the GetImportSuggestions RPC
func (s *ImportHelperGRPCServer) GetImportSuggestions(ctx context.Context, req *ImportSuggestionsRequest) (*ImportSuggestionsResponse, error) {
	return s.Impl.GetImportSuggestions(ctx, req)
}
