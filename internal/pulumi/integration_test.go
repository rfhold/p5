//go:build integration

package pulumi

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pulumi/pulumi/sdk/v3/go/auto"
)

// TestStack wraps a Pulumi stack with cleanup functionality
type TestStack struct {
	Stack      auto.Stack
	WorkDir    string // The project directory (with Pulumi.yaml)
	BackendDir string // The state backend directory
	t          *testing.T
	cleaned    bool
}

// SetupTestStack creates an isolated test stack using a local source project.
// It copies the specified fixture from testdata to a temp directory and
// creates a stack with a local file backend for isolation.
func SetupTestStack(t *testing.T, fixture string) *TestStack {
	t.Helper()

	ctx := context.Background()

	// Get the testdata fixture path
	fixturePath := filepath.Join("testdata", fixture)
	if _, err := os.Stat(fixturePath); os.IsNotExist(err) {
		t.Fatalf("fixture %s does not exist", fixturePath)
	}

	// Create temporary directory for the project
	workDir, err := os.MkdirTemp("", "pulumi-test-project-*")
	if err != nil {
		t.Fatalf("failed to create temp project dir: %v", err)
	}

	// Copy fixture to temp directory
	if err := copyDir(fixturePath, workDir); err != nil {
		os.RemoveAll(workDir)
		t.Fatalf("failed to copy fixture: %v", err)
	}

	// Create separate temp directory for state backend
	backendDir, err := os.MkdirTemp("", "pulumi-test-state-*")
	if err != nil {
		os.RemoveAll(workDir)
		t.Fatalf("failed to create temp state dir: %v", err)
	}

	// Generate unique stack name
	stackName := fmt.Sprintf("test-%s", time.Now().Format("150405"))

	// Environment variables for local backend
	env := map[string]string{
		"PULUMI_BACKEND_URL":       "file://" + backendDir,
		"PULUMI_CONFIG_PASSPHRASE": "test-passphrase-12345",
	}

	// Create the stack using local source
	s, err := auto.NewStackLocalSource(ctx, stackName, workDir,
		auto.EnvVars(env),
		auto.SecretsProvider("passphrase"),
	)
	if err != nil {
		os.RemoveAll(workDir)
		os.RemoveAll(backendDir)
		t.Fatalf("failed to create test stack: %v", err)
	}

	ts := &TestStack{
		Stack:      s,
		WorkDir:    workDir,
		BackendDir: backendDir,
		t:          t,
	}

	// Register cleanup
	t.Cleanup(ts.Cleanup)

	return ts
}

// Env returns the environment variables needed for operations
func (ts *TestStack) Env() map[string]string {
	return map[string]string{
		"PULUMI_BACKEND_URL":       "file://" + ts.BackendDir,
		"PULUMI_CONFIG_PASSPHRASE": "test-passphrase-12345",
	}
}

// Cleanup destroys the stack and removes temporary files
func (ts *TestStack) Cleanup() {
	if ts.cleaned {
		return
	}
	ts.cleaned = true

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Destroy resources (ignore errors - may not have any)
	_, _ = ts.Stack.Destroy(ctx)

	// Remove stack from backend
	_ = ts.Stack.Workspace().RemoveStack(ctx, ts.Stack.Name())

	// Remove temp directories
	os.RemoveAll(ts.WorkDir)
	os.RemoveAll(ts.BackendDir)
}

// Name returns the stack's full name
func (ts *TestStack) Name() string {
	return ts.Stack.Name()
}

// copyDir recursively copies a directory
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate destination path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		return copyFile(path, dstPath)
	})
}

// copyFile copies a single file
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// CollectPreviewEvents collects all events from a preview channel into a slice
func CollectPreviewEvents(ch <-chan PreviewEvent) []PreviewEvent {
	var events []PreviewEvent
	for event := range ch {
		events = append(events, event)
	}
	return events
}

// CollectOperationEvents collects all events from an operation channel into a slice
func CollectOperationEvents(ch <-chan OperationEvent) []OperationEvent {
	var events []OperationEvent
	for event := range ch {
		events = append(events, event)
	}
	return events
}

func TestIntegration_Preview_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	t.Parallel()

	ts := SetupTestStack(t, "simple")
	ctx := context.Background()

	operator := NewStackOperator()

	eventCh := operator.Preview(ctx, ts.WorkDir, ts.Name(), OperationUp, OperationOptions{Env: ts.Env()})
	events := CollectPreviewEvents(eventCh)

	// Verify we got events
	if len(events) == 0 {
		t.Fatal("expected preview events, got none")
	}

	// Verify we got a completion event (Done=true)
	var foundDone bool
	for _, e := range events {
		if e.Done {
			foundDone = true
			if e.Error != nil {
				t.Errorf("preview should succeed, got error: %v", e.Error)
			}
		}
	}
	if !foundDone {
		t.Error("expected Done event")
	}
}

func TestIntegration_Preview_WithChanges(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	t.Parallel()

	ts := SetupTestStack(t, "simple")
	ctx := context.Background()

	operator := NewStackOperator()

	eventCh := operator.Preview(ctx, ts.WorkDir, ts.Name(), OperationUp, OperationOptions{Env: ts.Env()})
	events := CollectPreviewEvents(eventCh)

	// Find step events with create operations
	var createSteps int
	for _, e := range events {
		if e.Step != nil && e.Step.Op == OpCreate {
			createSteps++
		}
	}

	// Should have at least one create (the random ID)
	if createSteps < 1 {
		t.Errorf("expected at least 1 create step, got %d", createSteps)
	}
}

func TestIntegration_Preview_Cancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	t.Parallel()

	ts := SetupTestStack(t, "simple")

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	operator := NewStackOperator()

	eventCh := operator.Preview(ctx, ts.WorkDir, ts.Name(), OperationUp, OperationOptions{Env: ts.Env()})

	// Cancel immediately
	cancel()

	// Drain events - should complete quickly
	done := make(chan struct{})
	go func() {
		CollectPreviewEvents(eventCh)
		close(done)
	}()

	select {
	case <-done:
		// Success - channel closed
	case <-time.After(30 * time.Second):
		t.Fatal("preview did not cancel in time")
	}
}

func TestIntegration_Up_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	t.Parallel()

	ts := SetupTestStack(t, "simple")
	ctx := context.Background()

	operator := NewStackOperator()

	eventCh := operator.Up(ctx, ts.WorkDir, ts.Name(), OperationOptions{Env: ts.Env()})
	events := CollectOperationEvents(eventCh)

	// Verify completion
	var foundDone bool
	for _, e := range events {
		if e.Done {
			foundDone = true
			if e.Error != nil {
				t.Errorf("up should succeed, got error: %v", e.Error)
			}
		}
	}

	if !foundDone {
		t.Error("expected Done event")
	}
}

func TestIntegration_Up_ThenDestroy(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	t.Parallel()

	ts := SetupTestStack(t, "simple")
	ctx := context.Background()

	operator := NewStackOperator()

	// First, run up
	upCh := operator.Up(ctx, ts.WorkDir, ts.Name(), OperationOptions{Env: ts.Env()})
	CollectOperationEvents(upCh)

	// Now destroy
	destroyCh := operator.Destroy(ctx, ts.WorkDir, ts.Name(), OperationOptions{Env: ts.Env()})
	events := CollectOperationEvents(destroyCh)

	// Verify destruction completed
	var foundDelete bool
	for _, e := range events {
		if e.Op == OpDelete {
			foundDelete = true
		}
	}
	if !foundDelete {
		t.Error("expected delete operations")
	}
}

func TestIntegration_Up_Idempotent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	t.Parallel()

	ts := SetupTestStack(t, "simple")
	ctx := context.Background()

	operator := NewStackOperator()

	// First up
	upCh1 := operator.Up(ctx, ts.WorkDir, ts.Name(), OperationOptions{Env: ts.Env()})
	CollectOperationEvents(upCh1)

	// Second up - should have no changes
	upCh2 := operator.Up(ctx, ts.WorkDir, ts.Name(), OperationOptions{Env: ts.Env()})
	events := CollectOperationEvents(upCh2)

	// Count actual changes (not same operations)
	var changes int
	for _, e := range events {
		if e.URN != "" && e.Op != OpSame && e.Op != "" {
			changes++
		}
	}
	if changes > 0 {
		t.Errorf("second up should have no changes, got %d", changes)
	}
}

func TestIntegration_Refresh_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	t.Parallel()

	ts := SetupTestStack(t, "simple")
	ctx := context.Background()

	operator := NewStackOperator()

	// First deploy
	upCh := operator.Up(ctx, ts.WorkDir, ts.Name(), OperationOptions{Env: ts.Env()})
	CollectOperationEvents(upCh)

	// Then refresh
	refreshCh := operator.Refresh(ctx, ts.WorkDir, ts.Name(), OperationOptions{Env: ts.Env()})
	events := CollectOperationEvents(refreshCh)

	// Should complete successfully
	var foundDone bool
	for _, e := range events {
		if e.Done {
			foundDone = true
			if e.Error != nil {
				t.Errorf("refresh should succeed, got error: %v", e.Error)
			}
		}
	}
	if !foundDone {
		t.Error("expected Done event")
	}
}

func TestIntegration_GetResources_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	// Not parallel - we're setting env vars
	ts := SetupTestStack(t, "simple")
	ctx := context.Background()

	// Set env vars for the reader (it doesn't have an Env option)
	for k, v := range ts.Env() {
		os.Setenv(k, v)
		defer os.Unsetenv(k)
	}

	reader := NewStackReader()

	resources, err := reader.GetResources(ctx, ts.WorkDir, ts.Name())
	if err != nil {
		t.Fatalf("GetResources failed: %v", err)
	}

	// A newly created stack with no deployments has 0 resources
	if len(resources) != 0 {
		t.Errorf("expected 0 resources for empty stack, got %d", len(resources))
	}
}

func TestIntegration_GetResources_AfterUp(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	// Not parallel - we're setting env vars
	ts := SetupTestStack(t, "multi")
	ctx := context.Background()

	// Set env vars for the reader (it doesn't have an Env option)
	for k, v := range ts.Env() {
		os.Setenv(k, v)
		defer os.Unsetenv(k)
	}

	// Deploy first
	operator := NewStackOperator()
	upCh := operator.Up(ctx, ts.WorkDir, ts.Name(), OperationOptions{Env: ts.Env()})
	CollectOperationEvents(upCh)

	// Now read
	reader := NewStackReader()
	resources, err := reader.GetResources(ctx, ts.WorkDir, ts.Name())
	if err != nil {
		t.Fatalf("GetResources failed: %v", err)
	}

	// Should have stack + 2 resources (RandomId + RandomString)
	if len(resources) < 3 {
		t.Errorf("expected at least 3 resources, got %d", len(resources))
	}

	// Verify resource types
	var foundRandomId, foundRandomString bool
	for _, r := range resources {
		if r.Type == "random:index/randomId:RandomId" {
			foundRandomId = true
		}
		if r.Type == "random:index/randomString:RandomString" {
			foundRandomString = true
		}
	}
	if !foundRandomId {
		t.Error("expected to find RandomId resource")
	}
	if !foundRandomString {
		t.Error("expected to find RandomString resource")
	}
}

func TestIntegration_IsWorkspace_Valid(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	reader := NewWorkspaceReader()

	// Use the testdata/simple directory which has a real Pulumi.yaml
	valid := reader.IsWorkspace("testdata/simple")
	if !valid {
		t.Error("expected testdata/simple to be a valid workspace")
	}
}

func TestIntegration_IsWorkspace_Invalid(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	reader := NewWorkspaceReader()

	// Create temp directory without Pulumi.yaml
	tmpDir, err := os.MkdirTemp("", "invalid-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	valid := reader.IsWorkspace(tmpDir)
	if valid {
		t.Error("expected temp directory without Pulumi.yaml to be invalid")
	}
}

func TestIntegration_History_AfterOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	// Not parallel - we're setting env vars
	ts := SetupTestStack(t, "simple")
	ctx := context.Background()

	// Set env vars for the reader (it doesn't have an Env option)
	for k, v := range ts.Env() {
		os.Setenv(k, v)
		defer os.Unsetenv(k)
	}

	operator := NewStackOperator()

	// Run up
	upCh := operator.Up(ctx, ts.WorkDir, ts.Name(), OperationOptions{Env: ts.Env()})
	CollectOperationEvents(upCh)

	// Check history
	reader := NewStackReader()
	history, err := reader.GetHistory(ctx, ts.WorkDir, ts.Name(), 10, 1)
	if err != nil {
		t.Fatalf("GetHistory failed: %v", err)
	}

	if len(history) == 0 {
		t.Error("expected at least one history entry")
	}

	// Most recent should be update
	if len(history) > 0 && history[0].Kind != "update" {
		t.Errorf("expected Kind=update, got %s", history[0].Kind)
	}
}
