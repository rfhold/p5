package pulumi

import (
	"os"
	"path/filepath"
	"strings"
)

// IsWorkspace checks if the given directory is a valid Pulumi workspace
// (contains Pulumi.yaml or Pulumi.yml)
func IsWorkspace(dir string) bool {
	yamlPath := filepath.Join(dir, "Pulumi.yaml")
	if _, err := os.Stat(yamlPath); err == nil {
		return true
	}
	ymlPath := filepath.Join(dir, "Pulumi.yml")
	if _, err := os.Stat(ymlPath); err == nil {
		return true
	}
	return false
}

// FindWorkspaces searches for Pulumi.yaml files starting from the given directory
// and returns a list of workspace paths. It searches recursively down the directory tree.
func FindWorkspaces(startDir, currentWorkDir string) ([]WorkspaceInfo, error) {
	var workspaces []WorkspaceInfo

	// Resolve absolute paths for comparison
	absStart, err := filepath.Abs(startDir)
	if err != nil {
		return nil, err
	}

	absCurrent := ""
	if currentWorkDir != "" {
		absCurrent, err = filepath.Abs(currentWorkDir)
		if err != nil {
			absCurrent = ""
		}
	}

	err = filepath.Walk(absStart, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip directories we can't access
			if info != nil && info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip hidden directories and common non-project directories
		if info.IsDir() {
			name := info.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == "__pycache__" {
				return filepath.SkipDir
			}
		}

		// Check for Pulumi.yaml or Pulumi.yml
		if !info.IsDir() && (info.Name() == "Pulumi.yaml" || info.Name() == "Pulumi.yml") {
			dir := filepath.Dir(path)

			// Try to get project name from the file
			projectName := filepath.Base(dir)
			if name, err := getProjectName(path); err == nil && name != "" {
				projectName = name
			}

			workspaces = append(workspaces, WorkspaceInfo{
				Path:    dir,
				Name:    projectName,
				Current: dir == absCurrent,
			})
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return workspaces, nil
}

// getProjectName reads the project name from a Pulumi.yaml file
func getProjectName(pulumiYamlPath string) (string, error) {
	data, err := os.ReadFile(pulumiYamlPath)
	if err != nil {
		return "", err
	}

	// Simple YAML parsing - just look for "name:" line
	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if name, ok := strings.CutPrefix(line, "name:"); ok {
			name = strings.TrimSpace(name)
			// Remove quotes if present
			name = strings.Trim(name, "\"'")
			return name, nil
		}
	}

	return "", nil
}
