package builtins

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rfhold/p5/internal/plugins"
	"github.com/rfhold/p5/internal/plugins/proto"
)

func init() {
	plugins.RegisterBuiltin(&EnvPlugin{
		BuiltinPluginBase: plugins.NewBuiltinPluginBase("env"),
	})
}

// EnvPlugin provides environment variables from .env files, static config, or command execution.
type EnvPlugin struct {
	plugins.BuiltinPluginBase
}

// EnvSource represents a single source of environment variables
type EnvSource struct {
	// Type is one of: "file", "static", "exec"
	Type string `json:"type"`
	// Path is the file path for "file" type (relative to workdir or absolute)
	Path string `json:"path,omitempty"`
	// Vars holds static variables for "static" type
	Vars map[string]string `json:"vars,omitempty"`
	// Cmd is the command for "exec" type
	Cmd string `json:"cmd,omitempty"`
	// Args are arguments for the "exec" command
	Args []string `json:"args,omitempty"`
	// Dir is the working directory for "exec" type (optional)
	Dir string `json:"dir,omitempty"`
}

// Authenticate processes environment sources and returns merged env vars
func (e *EnvPlugin) Authenticate(ctx context.Context, req *proto.AuthenticateRequest) (*proto.AuthenticateResponse, error) {
	// Parse sources from config
	// Config can have:
	//   - "sources": array of EnvSource objects
	//   - Single source fields at top level for simple cases

	sources, err := e.parseSources(req.ProgramConfig, req.StackConfig)
	if err != nil {
		return plugins.ErrorResponse("failed to parse sources: %v", err), nil
	}

	if len(sources) == 0 {
		return plugins.ErrorResponse("no env sources configured"), nil
	}

	// Process each source and merge env vars
	env := make(map[string]string)
	for i, src := range sources {
		srcEnv, err := e.processSource(ctx, src)
		if err != nil {
			return plugins.ErrorResponse("source %d (%s): %v", i, src.Type, err), nil
		}
		// Merge - later sources override earlier ones
		for k, v := range srcEnv {
			env[k] = v
		}
	}

	// Return with TTL of 0 (never expires, will reload on stack/workspace change)
	return plugins.SuccessResponse(env, 0), nil
}

// parseSources extracts EnvSource configs from program and stack config
// Sources are processed in order: simple format first, then sources array
// This allows global config (p5.toml) using simple format to be extended by
// program config (Pulumi.yaml) using sources array format
func (e *EnvPlugin) parseSources(programConfig, stackConfig map[string]string) ([]EnvSource, error) {
	var sources []EnvSource

	// First, check for simple single-source config at top level (type/vars/path/cmd)
	// This is typically from p5.toml global config
	if src := e.parseSimpleSource(programConfig); src != nil {
		sources = append(sources, *src)
	}

	// Then parse "sources" array from program config (extends simple format)
	if sourcesJSON, ok := programConfig["sources"]; ok {
		var parsed []EnvSource
		if err := json.Unmarshal([]byte(sourcesJSON), &parsed); err != nil {
			return nil, fmt.Errorf("invalid sources config: %w", err)
		}
		sources = append(sources, parsed...)
	}

	// Stack config simple format
	if src := e.parseSimpleSource(stackConfig); src != nil {
		sources = append(sources, *src)
	}

	// Stack config sources array (override/extend program sources)
	if sourcesJSON, ok := stackConfig["sources"]; ok {
		var parsed []EnvSource
		if err := json.Unmarshal([]byte(sourcesJSON), &parsed); err != nil {
			return nil, fmt.Errorf("invalid stack sources config: %w", err)
		}
		sources = append(sources, parsed...)
	}

	return sources, nil
}

// parseSimpleSource checks for a simple single-source config
func (e *EnvPlugin) parseSimpleSource(config map[string]string) *EnvSource {
	srcType, ok := config["type"]
	if !ok {
		// Try to infer type from other fields
		if _, hasPath := config["path"]; hasPath {
			srcType = "file"
		} else if _, hasCmd := config["cmd"]; hasCmd {
			srcType = "exec"
		} else if _, hasVars := config["vars"]; hasVars {
			srcType = "static"
		} else {
			return nil
		}
	}

	src := &EnvSource{Type: srcType}

	switch srcType {
	case "file":
		src.Path = config["path"]
	case "static":
		if varsJSON, ok := config["vars"]; ok {
			var vars map[string]string
			if err := json.Unmarshal([]byte(varsJSON), &vars); err == nil {
				src.Vars = vars
			}
		}
	case "exec":
		src.Cmd = config["cmd"]
		if argsJSON, ok := config["args"]; ok {
			var args []string
			if err := json.Unmarshal([]byte(argsJSON), &args); err == nil {
				src.Args = args
			}
		}
		src.Dir = config["dir"]
	}

	return src
}

// processSource loads env vars from a single source
func (e *EnvPlugin) processSource(ctx context.Context, src EnvSource) (map[string]string, error) {
	switch src.Type {
	case "file":
		return e.loadFromFile(src.Path)
	case "static":
		return src.Vars, nil
	case "exec":
		return e.loadFromExec(ctx, src)
	default:
		return nil, fmt.Errorf("unknown source type: %s", src.Type)
	}
}

// loadFromFile reads a .env file and parses it
func (e *EnvPlugin) loadFromFile(path string) (map[string]string, error) {
	if path == "" {
		return nil, fmt.Errorf("file path is required")
	}

	// Expand ~ to home directory
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(home, path[2:])
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return parseEnvFormat(data)
}

// loadFromExec runs a command and parses its stdout as .env format
func (e *EnvPlugin) loadFromExec(ctx context.Context, src EnvSource) (map[string]string, error) {
	if src.Cmd == "" {
		return nil, fmt.Errorf("cmd is required for exec source")
	}

	cmd := exec.CommandContext(ctx, src.Cmd, src.Args...)
	if src.Dir != "" {
		cmd.Dir = src.Dir
	}

	// Capture stdout
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("command failed: %w (stderr: %s)", err, stderr.String())
	}

	return parseEnvFormat(stdout.Bytes())
}

// parseEnvFormat parses .env format (KEY=VALUE lines)
func parseEnvFormat(data []byte) (map[string]string, error) {
	env := make(map[string]string)
	scanner := bufio.NewScanner(bytes.NewReader(data))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE
		idx := strings.Index(line, "=")
		if idx == -1 {
			continue // Skip lines without =
		}

		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])

		// Remove surrounding quotes if present
		value = unquote(value)

		if key != "" {
			env[key] = value
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to parse env format: %w", err)
	}

	return env, nil
}

// unquote removes surrounding quotes from a string
func unquote(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}
