package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// PluginFiles contains the paths to the downloaded plugin files
type PluginFiles struct {
	WasmPath     string
	ManifestPath string
}

// Source represents a plugin source that can download plugins
type Source interface {
	// Download fetches the plugin files to the cache directory
	Download(ctx context.Context, cacheDir string) (*PluginFiles, error)
	// CacheKey returns a unique key for caching this source
	CacheKey() string
}

// ParseSource parses a source string and returns the appropriate Source implementation
func ParseSource(source string) (Source, error) {
	// Local file path
	if strings.HasPrefix(source, "file://") {
		return &LocalSource{Path: strings.TrimPrefix(source, "file://")}, nil
	}
	if strings.HasPrefix(source, "./") || strings.HasPrefix(source, "/") || strings.HasPrefix(source, "../") {
		return &LocalSource{Path: source}, nil
	}

	// HTTP/HTTPS URL
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		return &HTTPSource{URL: source}, nil
	}

	// Git URL (github.com/org/repo/path@version)
	if strings.HasPrefix(source, "github.com/") {
		return parseGitHubSource(source)
	}

	return nil, fmt.Errorf("unsupported source format: %s", source)
}

// LocalSource handles local filesystem plugin sources
type LocalSource struct {
	Path string
}

func (s *LocalSource) CacheKey() string {
	abs, err := filepath.Abs(s.Path)
	if err != nil {
		return s.Path
	}
	return "local:" + abs
}

func (s *LocalSource) Download(ctx context.Context, cacheDir string) (*PluginFiles, error) {
	// For local sources, just resolve the paths
	absPath, err := filepath.Abs(s.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	// Check if it's a directory or a .wasm file
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat path: %w", err)
	}

	var wasmPath, manifestPath string
	if info.IsDir() {
		// Look for .wasm and .manifest.yaml files in the directory
		entries, err := os.ReadDir(absPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read directory: %w", err)
		}

		for _, entry := range entries {
			name := entry.Name()
			if strings.HasSuffix(name, ".wasm") && wasmPath == "" {
				wasmPath = filepath.Join(absPath, name)
			}
			if strings.HasSuffix(name, ".manifest.yaml") || strings.HasSuffix(name, ".manifest.yml") {
				manifestPath = filepath.Join(absPath, name)
			}
		}
	} else if strings.HasSuffix(absPath, ".wasm") {
		wasmPath = absPath
		// Look for manifest alongside wasm file
		base := strings.TrimSuffix(absPath, ".wasm")
		for _, ext := range []string{".manifest.yaml", ".manifest.yml"} {
			if _, err := os.Stat(base + ext); err == nil {
				manifestPath = base + ext
				break
			}
		}
	}

	if wasmPath == "" {
		return nil, fmt.Errorf("no .wasm file found in %s", absPath)
	}
	if manifestPath == "" {
		return nil, fmt.Errorf("no manifest file found for %s", absPath)
	}

	return &PluginFiles{
		WasmPath:     wasmPath,
		ManifestPath: manifestPath,
	}, nil
}

// HTTPSource handles HTTP/HTTPS plugin sources
type HTTPSource struct {
	URL string
}

func (s *HTTPSource) CacheKey() string {
	return "http:" + s.URL
}

func (s *HTTPSource) Download(ctx context.Context, cacheDir string) (*PluginFiles, error) {
	// Create cache directory for this source
	pluginDir := filepath.Join(cacheDir, sanitizeFilename(s.URL))
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	wasmPath := filepath.Join(pluginDir, "plugin.wasm")
	manifestPath := filepath.Join(pluginDir, "plugin.manifest.yaml")

	// Download .wasm file
	if err := downloadFile(ctx, s.URL, wasmPath); err != nil {
		return nil, fmt.Errorf("failed to download wasm: %w", err)
	}

	// Download manifest (assume it's next to the wasm with .manifest.yaml suffix)
	manifestURL := strings.TrimSuffix(s.URL, ".wasm") + ".manifest.yaml"
	if err := downloadFile(ctx, manifestURL, manifestPath); err != nil {
		return nil, fmt.Errorf("failed to download manifest: %w", err)
	}

	return &PluginFiles{
		WasmPath:     wasmPath,
		ManifestPath: manifestPath,
	}, nil
}

// GitHubSource handles GitHub release asset plugin sources
type GitHubSource struct {
	Owner   string
	Repo    string
	Path    string // Path within the repo (e.g., "plugins/okta-aws")
	Version string // Tag/version (e.g., "v1.0.0")
}

func parseGitHubSource(source string) (*GitHubSource, error) {
	// Format: github.com/owner/repo/path@version
	// Example: github.com/myorg/p5-plugins/okta-aws@v1.0.0

	// Remove github.com/ prefix
	rest := strings.TrimPrefix(source, "github.com/")

	// Split by @
	var version string
	if idx := strings.LastIndex(rest, "@"); idx != -1 {
		version = rest[idx+1:]
		rest = rest[:idx]
	}

	// Split path components
	parts := strings.SplitN(rest, "/", 3)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid github source: %s (expected github.com/owner/repo[/path][@version])", source)
	}

	gs := &GitHubSource{
		Owner:   parts[0],
		Repo:    parts[1],
		Version: version,
	}
	if len(parts) == 3 {
		gs.Path = parts[2]
	}

	return gs, nil
}

func (s *GitHubSource) CacheKey() string {
	return fmt.Sprintf("github:%s/%s/%s@%s", s.Owner, s.Repo, s.Path, s.Version)
}

func (s *GitHubSource) Download(ctx context.Context, cacheDir string) (*PluginFiles, error) {
	// Create cache directory for this version
	versionDir := s.Version
	if versionDir == "" {
		versionDir = "latest"
	}
	pluginDir := filepath.Join(cacheDir, s.Owner, s.Repo, s.Path, versionDir)

	// Check if already cached
	wasmPath := filepath.Join(pluginDir, "plugin.wasm")
	manifestPath := filepath.Join(pluginDir, "plugin.manifest.yaml")

	if _, err := os.Stat(wasmPath); err == nil {
		if _, err := os.Stat(manifestPath); err == nil {
			// Already cached
			return &PluginFiles{
				WasmPath:     wasmPath,
				ManifestPath: manifestPath,
			}, nil
		}
	}

	// Need to download
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Determine the release tag
	tag := s.Version
	if tag == "" {
		// Get latest release
		latestTag, err := s.getLatestRelease(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get latest release: %w", err)
		}
		tag = latestTag
	}

	// Construct asset names based on path
	assetBase := filepath.Base(s.Path)
	if assetBase == "" || assetBase == "." {
		assetBase = s.Repo
	}

	wasmAsset := assetBase + ".wasm"
	manifestAsset := assetBase + ".manifest.yaml"

	// Download release assets
	if err := s.downloadReleaseAsset(ctx, tag, wasmAsset, wasmPath); err != nil {
		return nil, fmt.Errorf("failed to download wasm asset: %w", err)
	}

	if err := s.downloadReleaseAsset(ctx, tag, manifestAsset, manifestPath); err != nil {
		// Try .yml extension
		manifestAsset = assetBase + ".manifest.yml"
		if err := s.downloadReleaseAsset(ctx, tag, manifestAsset, manifestPath); err != nil {
			return nil, fmt.Errorf("failed to download manifest asset: %w", err)
		}
	}

	return &PluginFiles{
		WasmPath:     wasmPath,
		ManifestPath: manifestPath,
	}, nil
}

func (s *GitHubSource) getLatestRelease(ctx context.Context) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", s.Owner, s.Repo)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	return release.TagName, nil
}

func (s *GitHubSource) downloadReleaseAsset(ctx context.Context, tag, assetName, destPath string) error {
	// First, get the release by tag
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/tags/%s", s.Owner, s.Repo, tag)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub API returned %d for tag %s", resp.StatusCode, tag)
	}

	var release struct {
		Assets []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return err
	}

	// Find the asset
	var downloadURL string
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		return fmt.Errorf("asset %s not found in release %s", assetName, tag)
	}

	return downloadFile(ctx, downloadURL, destPath)
}

// downloadFile downloads a URL to a local file
func downloadFile(ctx context.Context, url, destPath string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// sanitizeFilename converts a URL to a safe filename
func sanitizeFilename(s string) string {
	// Replace URL-unfriendly chars
	re := regexp.MustCompile(`[^a-zA-Z0-9_.-]`)
	return re.ReplaceAllString(s, "_")
}

// GetCacheDir returns the plugin cache directory
func GetCacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".p5", "plugins"), nil
}
