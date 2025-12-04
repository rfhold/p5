package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"

	extism "github.com/extism/go-sdk"
)

// AuthInput is the input passed to the plugin's authenticate function
// Note: Config maps use string values for TinyGo WASM compatibility
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
	TTLSeconds int               `json:"ttl_seconds,omitempty"` // -1 means always call
	Error      string            `json:"error,omitempty"`
}

// Plugin represents a loaded WASM plugin with its manifest
type Plugin struct {
	name     string
	manifest *Manifest
	plugin   *extism.Plugin
}

// HTTPRequest represents an HTTP request from the plugin
type HTTPRequest struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
}

// HTTPResponse represents an HTTP response to the plugin
type HTTPResponse struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       string            `json:"body,omitempty"`
	Error      string            `json:"error,omitempty"`
}

// OpenBrowserRequest represents a request to open a browser
type OpenBrowserRequest struct {
	URL string `json:"url"`
}

// WaitForCallbackRequest represents a request to wait for an OAuth callback
type WaitForCallbackRequest struct {
	Port           int    `json:"port"`
	Path           string `json:"path"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}

// WaitForCallbackResponse represents the OAuth callback response
type WaitForCallbackResponse struct {
	QueryParams map[string]string `json:"query_params,omitempty"`
	Error       string            `json:"error,omitempty"`
}

// LoadPlugin loads a WASM plugin with the given manifest
func LoadPlugin(ctx context.Context, name string, wasmPath string, manifest *Manifest) (*Plugin, error) {
	// Create host functions
	hostFuncs := createHostFunctions(manifest)

	// Configure the plugin
	config := extism.PluginConfig{
		EnableWasi: true,
	}

	// Create the plugin
	wasmManifest := extism.Manifest{
		Wasm: []extism.Wasm{
			extism.WasmFile{Path: wasmPath},
		},
		// Note: Extism's AllowedHosts is for their built-in HTTP, but we use custom host functions
	}

	plugin, err := extism.NewPlugin(ctx, wasmManifest, config, hostFuncs)
	if err != nil {
		return nil, fmt.Errorf("failed to create plugin: %w", err)
	}

	return &Plugin{
		name:     name,
		manifest: manifest,
		plugin:   plugin,
	}, nil
}

// Authenticate calls the plugin's authenticate function
func (p *Plugin) Authenticate(ctx context.Context, input AuthInput) (*AuthOutput, error) {
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input: %w", err)
	}

	// Call the plugin function
	_, outputBytes, err := p.plugin.Call("authenticate", inputJSON)
	if err != nil {
		return nil, fmt.Errorf("plugin call failed: %w", err)
	}

	var output AuthOutput
	if err := json.Unmarshal(outputBytes, &output); err != nil {
		return nil, fmt.Errorf("failed to unmarshal output: %w", err)
	}

	// Validate the output against the manifest
	for envVar := range output.Env {
		if !p.manifest.IsEnvAllowed(envVar) {
			return nil, fmt.Errorf("plugin tried to set disallowed env var: %s", envVar)
		}
	}

	return &output, nil
}

// Close releases plugin resources
func (p *Plugin) Close(ctx context.Context) {
	if p.plugin != nil {
		p.plugin.Close(ctx)
	}
}

// Name returns the plugin name
func (p *Plugin) Name() string {
	return p.name
}

// Manifest returns the plugin manifest
func (p *Plugin) Manifest() *Manifest {
	return p.manifest
}

// createHostFunctions creates the host functions available to plugins
func createHostFunctions(manifest *Manifest) []extism.HostFunction {
	return []extism.HostFunction{
		createHTTPRequestFunction(manifest),
		createOpenBrowserFunction(manifest),
		createWaitForCallbackFunction(),
	}
}

// createHTTPRequestFunction creates the http_request host function
func createHTTPRequestFunction(manifest *Manifest) extism.HostFunction {
	return extism.NewHostFunctionWithStack(
		"http_request",
		func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
			// Read input from plugin memory
			inputOffset := stack[0]
			inputBytes, err := p.ReadBytes(inputOffset)
			if err != nil {
				writeErrorResponse(p, stack, fmt.Sprintf("failed to read input: %v", err))
				return
			}

			var req HTTPRequest
			if err := json.Unmarshal(inputBytes, &req); err != nil {
				writeErrorResponse(p, stack, fmt.Sprintf("failed to parse request: %v", err))
				return
			}

			// Parse URL and check against allowed hosts
			parsedURL, err := url.Parse(req.URL)
			if err != nil {
				writeErrorResponse(p, stack, fmt.Sprintf("invalid URL: %v", err))
				return
			}

			if !manifest.IsHTTPAllowed(parsedURL.Host) {
				writeErrorResponse(p, stack, fmt.Sprintf("host not allowed: %s", parsedURL.Host))
				return
			}

			// Make the HTTP request
			httpReq, err := http.NewRequestWithContext(ctx, req.Method, req.URL, strings.NewReader(req.Body))
			if err != nil {
				writeErrorResponse(p, stack, fmt.Sprintf("failed to create request: %v", err))
				return
			}

			for k, v := range req.Headers {
				httpReq.Header.Set(k, v)
			}

			resp, err := http.DefaultClient.Do(httpReq)
			if err != nil {
				writeErrorResponse(p, stack, fmt.Sprintf("request failed: %v", err))
				return
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				writeErrorResponse(p, stack, fmt.Sprintf("failed to read response: %v", err))
				return
			}

			// Build response
			httpResp := HTTPResponse{
				StatusCode: resp.StatusCode,
				Headers:    make(map[string]string),
				Body:       string(body),
			}
			for k, v := range resp.Header {
				if len(v) > 0 {
					httpResp.Headers[k] = v[0]
				}
			}

			respBytes, _ := json.Marshal(httpResp)
			offset, err := p.WriteBytes(respBytes)
			if err != nil {
				writeErrorResponse(p, stack, fmt.Sprintf("failed to write response: %v", err))
				return
			}
			stack[0] = offset
		},
		[]extism.ValueType{extism.ValueTypeI64},
		[]extism.ValueType{extism.ValueTypeI64},
	)
}

// createOpenBrowserFunction creates the open_browser host function
func createOpenBrowserFunction(manifest *Manifest) extism.HostFunction {
	return extism.NewHostFunctionWithStack(
		"open_browser",
		func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
			// Only allow if plugin is marked as interactive
			if !manifest.Interactive {
				writeErrorResponse(p, stack, "plugin is not marked as interactive")
				return
			}

			// Read URL from plugin memory
			inputOffset := stack[0]
			inputBytes, err := p.ReadBytes(inputOffset)
			if err != nil {
				writeErrorResponse(p, stack, fmt.Sprintf("failed to read input: %v", err))
				return
			}

			var req OpenBrowserRequest
			if err := json.Unmarshal(inputBytes, &req); err != nil {
				writeErrorResponse(p, stack, fmt.Sprintf("failed to parse request: %v", err))
				return
			}

			// Validate the URL host is allowed
			parsedURL, err := url.Parse(req.URL)
			if err != nil {
				writeErrorResponse(p, stack, fmt.Sprintf("invalid URL: %v", err))
				return
			}

			if !manifest.IsHTTPAllowed(parsedURL.Host) {
				writeErrorResponse(p, stack, fmt.Sprintf("host not allowed: %s", parsedURL.Host))
				return
			}

			// Open the browser
			if err := openBrowser(req.URL); err != nil {
				writeErrorResponse(p, stack, fmt.Sprintf("failed to open browser: %v", err))
				return
			}

			// Return success (empty response)
			respBytes, _ := json.Marshal(map[string]string{"status": "ok"})
			offset, err := p.WriteBytes(respBytes)
			if err != nil {
				writeErrorResponse(p, stack, fmt.Sprintf("failed to write response: %v", err))
				return
			}
			stack[0] = offset
		},
		[]extism.ValueType{extism.ValueTypeI64},
		[]extism.ValueType{extism.ValueTypeI64},
	)
}

// createWaitForCallbackFunction creates the wait_for_callback host function
func createWaitForCallbackFunction() extism.HostFunction {
	return extism.NewHostFunctionWithStack(
		"wait_for_callback",
		func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
			// Read request from plugin memory
			inputOffset := stack[0]
			inputBytes, err := p.ReadBytes(inputOffset)
			if err != nil {
				writeErrorResponse(p, stack, fmt.Sprintf("failed to read input: %v", err))
				return
			}

			var req WaitForCallbackRequest
			if err := json.Unmarshal(inputBytes, &req); err != nil {
				writeErrorResponse(p, stack, fmt.Sprintf("failed to parse request: %v", err))
				return
			}

			// Set reasonable defaults
			if req.TimeoutSeconds <= 0 {
				req.TimeoutSeconds = 120
			}
			if req.Path == "" {
				req.Path = "/callback"
			}

			// Start a local HTTP server to receive the callback
			result, err := waitForOAuthCallback(ctx, req.Port, req.Path, time.Duration(req.TimeoutSeconds)*time.Second)
			if err != nil {
				writeErrorResponse(p, stack, fmt.Sprintf("callback failed: %v", err))
				return
			}

			resp := WaitForCallbackResponse{
				QueryParams: result,
			}
			respBytes, _ := json.Marshal(resp)
			offset, err := p.WriteBytes(respBytes)
			if err != nil {
				writeErrorResponse(p, stack, fmt.Sprintf("failed to write response: %v", err))
				return
			}
			stack[0] = offset
		},
		[]extism.ValueType{extism.ValueTypeI64},
		[]extism.ValueType{extism.ValueTypeI64},
	)
}

// writeErrorResponse writes an error response back to the plugin
func writeErrorResponse(p *extism.CurrentPlugin, stack []uint64, errMsg string) {
	resp := map[string]string{"error": errMsg}
	respBytes, _ := json.Marshal(resp)
	offset, err := p.WriteBytes(respBytes)
	if err != nil {
		// If we can't write the error, just return 0
		stack[0] = 0
		return
	}
	stack[0] = offset
}

// openBrowser opens a URL in the default browser
func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}

// waitForOAuthCallback starts a local HTTP server and waits for the OAuth callback
func waitForOAuthCallback(ctx context.Context, port int, path string, timeout time.Duration) (map[string]string, error) {
	resultCh := make(chan map[string]string, 1)
	errCh := make(chan error, 1)

	// Create listener
	addr := fmt.Sprintf("localhost:%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	// Create server with handler
	mux := http.NewServeMux()
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		// Extract query parameters
		params := make(map[string]string)
		for k, v := range r.URL.Query() {
			if len(v) > 0 {
				params[k] = v[0]
			}
		}

		// Send success response to browser
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `<!DOCTYPE html>
<html>
<head><title>Authentication Complete</title></head>
<body>
<h1>Authentication Complete</h1>
<p>You can close this window and return to the terminal.</p>
<script>window.close();</script>
</body>
</html>`)

		// Send result
		select {
		case resultCh <- params:
		default:
		}
	})

	server := &http.Server{Handler: mux}

	// Start server in background
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			select {
			case errCh <- err:
			default:
			}
		}
	}()

	// Wait for result, timeout, or context cancellation
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		server.Shutdown(shutdownCtx)
	}()

	select {
	case result := <-resultCh:
		return result, nil
	case err := <-errCh:
		return nil, err
	case <-ctx.Done():
		return nil, fmt.Errorf("timeout waiting for callback")
	}
}
