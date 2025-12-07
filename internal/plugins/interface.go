package plugins

import (
	p5plugin "github.com/rfhold/p5/pkg/plugin"
)

// AuthPlugin is the interface that plugins must implement.
// This is re-exported from pkg/plugin for internal use.
type AuthPlugin = p5plugin.AuthPlugin

// ImportHelperPlugin is an optional interface that plugins can implement
// to provide import ID suggestions for resources.
// This is re-exported from pkg/plugin for internal use.
type ImportHelperPlugin = p5plugin.ImportHelperPlugin

// ResourceOpenerPlugin is an optional interface that plugins can implement
// to provide resource opening capabilities (browser URLs or alternate screen programs).
// This is re-exported from pkg/plugin for internal use.
type ResourceOpenerPlugin = p5plugin.ResourceOpenerPlugin

// Re-export import suggestion types from pkg/plugin for internal use.
type (
	ImportSuggestionsRequest  = p5plugin.ImportSuggestionsRequest
	ImportSuggestionsResponse = p5plugin.ImportSuggestionsResponse
	ImportSuggestion          = p5plugin.ImportSuggestion
)

// Re-export resource opener types from pkg/plugin for internal use.
type (
	SupportedOpenTypesRequest  = p5plugin.SupportedOpenTypesRequest
	SupportedOpenTypesResponse = p5plugin.SupportedOpenTypesResponse
	OpenResourceRequest        = p5plugin.OpenResourceRequest
	OpenResourceResponse       = p5plugin.OpenResourceResponse
	OpenAction                 = p5plugin.OpenAction
	OpenActionType             = p5plugin.OpenActionType
)

// Re-export import suggestion helper functions from pkg/plugin for internal use.
var (
	ImportSuggestionsNotSupported = p5plugin.ImportSuggestionsNotSupported
	ImportSuggestionsSuccess      = p5plugin.ImportSuggestionsSuccess
	ImportSuggestionsError        = p5plugin.ImportSuggestionsError
	NewImportSuggestion           = p5plugin.NewImportSuggestion
)

// Re-export resource opener helper functions from pkg/plugin for internal use.
var (
	OpenNotSupported           = p5plugin.OpenNotSupported
	OpenBrowserResponse        = p5plugin.OpenBrowserResponse
	OpenExecResponse           = p5plugin.OpenExecResponse
	OpenError                  = p5plugin.OpenError
	SupportedOpenTypesPatterns = p5plugin.SupportedOpenTypesPatterns
)
