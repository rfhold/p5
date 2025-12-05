package builtins

import (
	"context"
	"strings"

	"github.com/rfhold/p5/internal/plugins"
	"github.com/rfhold/p5/internal/plugins/proto"
	"github.com/rfhold/p5/pkg/plugin"
)

func init() {
	plugins.RegisterBuiltin(&CloudflarePlugin{
		BuiltinPluginBase: plugins.NewBuiltinPluginBase("cloudflare"),
	})
}

// CloudflarePlugin provides import suggestions for Cloudflare resources.
// Currently returns dummy data for testing purposes.
type CloudflarePlugin struct {
	plugins.BuiltinPluginBase
}

// Authenticate returns a no-op success response.
// This plugin is primarily for import help, not auth.
func (p *CloudflarePlugin) Authenticate(ctx context.Context, req *proto.AuthenticateRequest) (*proto.AuthenticateResponse, error) {
	return plugins.SuccessResponse(nil, 0), nil
}

// GetImportSuggestions returns import ID suggestions for Cloudflare resources.
// Currently returns dummy data for testing purposes.
func (p *CloudflarePlugin) GetImportSuggestions(ctx context.Context, req *plugin.ImportSuggestionsRequest) (*plugin.ImportSuggestionsResponse, error) {
	// Check if this is a Cloudflare resource
	if !strings.HasPrefix(req.ResourceType, "cloudflare:") {
		return plugin.ImportSuggestionsNotSupported(), nil
	}

	// Return dummy suggestions based on resource type
	var suggestions []*plugin.ImportSuggestion
	return plugin.ImportSuggestionsSuccess(suggestions), nil
}
