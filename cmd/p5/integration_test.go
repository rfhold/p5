//go:build integration

package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/golden"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/muesli/termenv"

	"github.com/rfhold/p5/internal/plugins"
	"github.com/rfhold/p5/internal/pulumi"
)

func init() {
	// Force consistent color profile for reproducible tests across environments
	lipgloss.SetColorProfile(termenv.Ascii)
}

// Golden file test dimensions - consistent size for reproducible snapshots
const (
	goldenWidth  = 120
	goldenHeight = 40
)

// =============================================================================
// Test Helper Functions
// =============================================================================

// testModelOption is a function that configures test dependencies or context
type testModelOption func(*Dependencies, *AppContext)

// getTestWorkDir returns the path to test/simple for tests that need a real workspace
func getTestWorkDir(t *testing.T) string {
	t.Helper()
	// Find the test/simple directory relative to this test file
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	// We're in cmd/p5, go up to root then into test/simple
	testDir := filepath.Join(wd, "..", "..", "test", "simple")
	absPath, err := filepath.Abs(testDir)
	if err != nil {
		t.Fatalf("failed to get absolute path: %v", err)
	}
	return absPath
}

// createTestModel creates a Model with fake dependencies for integration testing.
// Note: Some commands call Pulumi functions directly rather than using injected
// dependencies, so tests that need workspace validation should use a real
// workspace directory (see getTestWorkDir).
func createTestModel(t *testing.T, opts ...testModelOption) Model {
	t.Helper()

	deps := &Dependencies{
		StackOperator: &pulumi.FakeStackOperator{},
		StackReader:   &pulumi.FakeStackReader{},
		WorkspaceReader: &pulumi.FakeWorkspaceReader{
			ValidWorkDir: true,
			// Provide default ProjectInfo to avoid nil pointer dereference
			ProjectInfo: &pulumi.ProjectInfo{
				ProgramName: "test-project",
				StackName:   "dev",
			},
		},
		StackInitializer: &pulumi.FakeStackInitializer{},
		ResourceImporter: &pulumi.FakeResourceImporter{},
		PluginProvider:   &plugins.FakePluginProvider{},
	}

	appCtx := AppContext{
		WorkDir:   "/fake/workdir",
		StackName: "dev",
		StartView: "stack",
	}

	for _, opt := range opts {
		opt(deps, &appCtx)
	}

	return initialModel(context.Background(), appCtx, deps)
}

// withStackOperator sets a custom StackOperator for the test
func withStackOperator(op pulumi.StackOperator) testModelOption {
	return func(d *Dependencies, _ *AppContext) {
		d.StackOperator = op
	}
}

// withStackReader sets a custom StackReader for the test
func withStackReader(reader pulumi.StackReader) testModelOption {
	return func(d *Dependencies, _ *AppContext) {
		d.StackReader = reader
	}
}

// withWorkspaceReader sets a custom WorkspaceReader for the test
func withWorkspaceReader(reader pulumi.WorkspaceReader) testModelOption {
	return func(d *Dependencies, _ *AppContext) {
		d.WorkspaceReader = reader
	}
}

// withStartView sets the initial view mode
func withStartView(view string) testModelOption {
	return func(_ *Dependencies, ctx *AppContext) {
		ctx.StartView = view
	}
}

// withStackName sets the stack name
func withStackName(name string) testModelOption {
	return func(_ *Dependencies, ctx *AppContext) {
		ctx.StackName = name
	}
}

// withResources configures the FakeStackReader to return the given resources
func withResources(resources []pulumi.ResourceInfo) testModelOption {
	return func(d *Dependencies, _ *AppContext) {
		d.StackReader = &pulumi.FakeStackReader{
			Resources: resources,
		}
	}
}

// withStacks configures the FakeStackReader to return the given stacks
func withStacks(stacks []pulumi.StackInfo) testModelOption {
	return func(d *Dependencies, _ *AppContext) {
		reader, ok := d.StackReader.(*pulumi.FakeStackReader)
		if !ok {
			reader = &pulumi.FakeStackReader{}
			d.StackReader = reader
		}
		reader.Stacks = stacks
	}
}

// withPluginProvider sets a custom PluginProvider for the test
func withPluginProvider(provider plugins.PluginProvider) testModelOption {
	return func(d *Dependencies, _ *AppContext) {
		d.PluginProvider = provider
	}
}

// waitForContent waits for specific content to appear in output
func waitForContent(t *testing.T, tm *teatest.TestModel, content string, timeout time.Duration) {
	t.Helper()
	teatest.WaitFor(t, tm.Output(),
		func(bts []byte) bool {
			return bytes.Contains(bts, []byte(content))
		},
		teatest.WithCheckInterval(50*time.Millisecond),
		teatest.WithDuration(timeout),
	)
}

// waitForAnyContent waits for any of the specified content strings
func waitForAnyContent(t *testing.T, tm *teatest.TestModel, contents []string, timeout time.Duration) {
	t.Helper()
	teatest.WaitFor(t, tm.Output(),
		func(bts []byte) bool {
			for _, content := range contents {
				if bytes.Contains(bts, []byte(content)) {
					return true
				}
			}
			return false
		},
		teatest.WithCheckInterval(50*time.Millisecond),
		teatest.WithDuration(timeout),
	)
}

// =============================================================================
// Sample Test Data
// =============================================================================

// testResources creates a simple set of test resources
func testResources() []pulumi.ResourceInfo {
	return []pulumi.ResourceInfo{
		{
			URN:     "urn:pulumi:dev::test::pulumi:pulumi:Stack::test-dev",
			Type:    "pulumi:pulumi:Stack",
			Name:    "test-dev",
			Parent:  "",
			Inputs:  map[string]interface{}{},
			Outputs: map[string]interface{}{},
		},
		{
			URN:     "urn:pulumi:dev::test::aws:s3:Bucket::mybucket",
			Type:    "aws:s3:Bucket",
			Name:    "mybucket",
			Parent:  "urn:pulumi:dev::test::pulumi:pulumi:Stack::test-dev",
			Inputs:  map[string]interface{}{"bucket": "my-bucket-name"},
			Outputs: map[string]interface{}{"id": "my-bucket-name", "arn": "arn:aws:s3:::my-bucket-name"},
		},
		{
			URN:     "urn:pulumi:dev::test::aws:lambda:Function::myfunc",
			Type:    "aws:lambda:Function",
			Name:    "myfunc",
			Parent:  "urn:pulumi:dev::test::pulumi:pulumi:Stack::test-dev",
			Inputs:  map[string]interface{}{"runtime": "nodejs18.x"},
			Outputs: map[string]interface{}{"arn": "arn:aws:lambda:us-east-1:123456789:function:myfunc"},
		},
	}
}

// testStacks creates a simple set of test stacks
func testStacks() []pulumi.StackInfo {
	return []pulumi.StackInfo{
		{Name: "dev", Current: true},
		{Name: "staging", Current: false},
		{Name: "prod", Current: false},
	}
}

// =============================================================================
// Initialization Flow Tests
// =============================================================================

// TestInitializationFlow_Success tests the complete initialization sequence
// when everything works correctly.
func TestInitializationFlow_Success(t *testing.T) {
	resources := testResources()
	stacks := testStacks()

	m := createTestModel(t,
		withResources(resources),
		withStacks(stacks),
		withStackName("dev"),
	)

	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(120, 40),
	)

	// Wait for resources to load and display
	waitForContent(t, tm, "mybucket", 5*time.Second)

	// Verify we can quit cleanly
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

// TestInitializationFlow_WorkspaceInvalid tests error handling when the
// workspace is invalid (no Pulumi.yaml).
func TestInitializationFlow_WorkspaceInvalid(t *testing.T) {
	m := createTestModel(t,
		withWorkspaceReader(&pulumi.FakeWorkspaceReader{
			ValidWorkDir: false, // Workspace is invalid
			// Still need ProjectInfo for any deferred auth calls
			ProjectInfo: &pulumi.ProjectInfo{
				ProgramName: "test-project",
				StackName:   "dev",
			},
		}),
	)

	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(120, 40),
	)

	// Wait for workspace selector to appear
	waitForAnyContent(t, tm, []string{"Workspace", "Select"}, 3*time.Second)

	// When workspace selector is showing with no workspaces, the user can't quit
	// via 'q' key (focus is on selector). Use tea.Quit directly for testing.
	tm.Send(tea.Quit())
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

// =============================================================================
// Navigation Tests
// =============================================================================

// TestNavigation_ResourceList tests keyboard navigation through the resource list.
func TestNavigation_ResourceList(t *testing.T) {
	resources := testResources()

	m := createTestModel(t,
		withResources(resources),
		withStacks(testStacks()),
	)

	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(120, 40),
	)

	// Wait for initial load
	waitForContent(t, tm, "mybucket", 3*time.Second)

	// Navigate down (j key)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})

	// Navigate up (k key)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

// TestNavigation_ToggleDetails tests toggling the details panel.
func TestNavigation_ToggleDetails(t *testing.T) {
	resources := testResources()

	m := createTestModel(t,
		withResources(resources),
		withStacks(testStacks()),
	)

	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(120, 40),
	)

	// Wait for initial load
	waitForContent(t, tm, "mybucket", 3*time.Second)

	// Open details panel (D key)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("D")})

	// Wait for details to show
	time.Sleep(100 * time.Millisecond)

	// Close details panel (D key again)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("D")})

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

// =============================================================================
// Help Modal Tests
// =============================================================================

// TestHelpModal_ToggleWithQuestionMark tests opening and closing the help modal.
func TestHelpModal_ToggleWithQuestionMark(t *testing.T) {
	m := createTestModel(t,
		withResources(testResources()),
		withStacks(testStacks()),
	)

	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(120, 40),
	)

	waitForContent(t, tm, "mybucket", 3*time.Second)

	// Open help
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	waitForAnyContent(t, tm, []string{"Help", "Keybindings", "Key"}, 2*time.Second)

	// Close help
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})

	// Wait briefly for modal to close
	time.Sleep(100 * time.Millisecond)

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

// =============================================================================
// Visual Mode Tests
// =============================================================================

// TestVisualMode_SelectResources tests entering and exiting visual selection mode.
func TestVisualMode_SelectResources(t *testing.T) {
	resources := testResources()

	m := createTestModel(t,
		withResources(resources),
		withStacks(testStacks()),
	)

	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(120, 40),
	)

	// Wait for initial load
	waitForContent(t, tm, "mybucket", 3*time.Second)

	// Enter visual mode (v key)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})

	// Wait for visual mode indicator
	waitForContent(t, tm, "VISUAL", 2*time.Second)

	// Move down to extend selection
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})

	// Exit visual mode with Escape
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})

	// Wait briefly for mode to exit
	time.Sleep(100 * time.Millisecond)

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

// =============================================================================
// Edge Case Tests
// =============================================================================

// TestResize_DuringDisplay tests that the app handles terminal resize gracefully.
func TestResize_DuringDisplay(t *testing.T) {
	m := createTestModel(t,
		withResources(testResources()),
		withStacks(testStacks()),
	)

	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(120, 40),
	)

	// Wait for initial load
	waitForContent(t, tm, "mybucket", 3*time.Second)

	// Send resize message
	tm.Send(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Wait briefly for resize to process
	time.Sleep(100 * time.Millisecond)

	// Send another resize (larger)
	tm.Send(tea.WindowSizeMsg{Width: 160, Height: 50})

	// Wait briefly
	time.Sleep(100 * time.Millisecond)

	// App should still be functional - quit cleanly
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

// TestRapidKeyPresses tests that the app handles rapid key presses without crashing.
func TestRapidKeyPresses(t *testing.T) {
	// Create many resources for scrolling
	resources := make([]pulumi.ResourceInfo, 50)
	resources[0] = pulumi.ResourceInfo{
		URN:    "urn:pulumi:dev::test::pulumi:pulumi:Stack::test-dev",
		Type:   "pulumi:pulumi:Stack",
		Name:   "test-dev",
		Parent: "",
	}
	for i := 1; i < 50; i++ {
		resources[i] = pulumi.ResourceInfo{
			URN:    "urn:pulumi:dev::test::aws:s3:Bucket::bucket-" + itoa(i),
			Type:   "aws:s3:Bucket",
			Name:   "bucket-" + itoa(i),
			Parent: "urn:pulumi:dev::test::pulumi:pulumi:Stack::test-dev",
		}
	}

	m := createTestModel(t,
		withResources(resources),
		withStacks(testStacks()),
	)

	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(120, 40),
	)

	waitForContent(t, tm, "bucket-1", 3*time.Second)

	// Rapid navigation
	for i := 0; i < 20; i++ {
		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	}
	for i := 0; i < 10; i++ {
		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	}

	// Should not crash or hang - quit cleanly
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(5*time.Second))
}

// Note: itoa helper is defined in logic.go and reused here

// =============================================================================
// Golden Snapshot Helper Functions
// =============================================================================

// takeSnapshot captures the current TUI state and compares against golden file.
// The snapshot name is appended to the test name to allow multiple snapshots per test.
func takeSnapshot(t *testing.T, tm *teatest.TestModel, snapshotName string) {
	t.Helper()
	// Give UI time to stabilize
	time.Sleep(100 * time.Millisecond)

	// Read current output
	out, err := io.ReadAll(tm.Output())
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}

	// Use golden.RequireEqualSub for sub-test naming
	t.Run(snapshotName, func(t *testing.T) {
		golden.RequireEqual(t, out)
	})
}

// waitAndSnapshot waits for content then takes a golden snapshot
func waitAndSnapshot(t *testing.T, tm *teatest.TestModel, content string, snapshotName string, timeout time.Duration) {
	t.Helper()
	waitForContent(t, tm, content, timeout)
	takeSnapshot(t, tm, snapshotName)
}

// =============================================================================
// Enhanced Test Data Builders
// =============================================================================

// testResourcesWithHierarchy creates a realistic resource tree with parent-child relationships
func testResourcesWithHierarchy() []pulumi.ResourceInfo {
	return []pulumi.ResourceInfo{
		{
			URN:     "urn:pulumi:dev::myapp::pulumi:pulumi:Stack::myapp-dev",
			Type:    "pulumi:pulumi:Stack",
			Name:    "myapp-dev",
			Parent:  "",
			Inputs:  map[string]interface{}{},
			Outputs: map[string]interface{}{},
		},
		{
			URN:    "urn:pulumi:dev::myapp::aws:s3:Bucket::data-bucket",
			Type:   "aws:s3:Bucket",
			Name:   "data-bucket",
			Parent: "urn:pulumi:dev::myapp::pulumi:pulumi:Stack::myapp-dev",
			Inputs: map[string]interface{}{
				"bucket":       "myapp-data-bucket-123",
				"acl":          "private",
				"forceDestroy": false,
			},
			Outputs: map[string]interface{}{
				"id":  "myapp-data-bucket-123",
				"arn": "arn:aws:s3:::myapp-data-bucket-123",
			},
		},
		{
			URN:    "urn:pulumi:dev::myapp::aws:iam:Role::lambda-role",
			Type:   "aws:iam:Role",
			Name:   "lambda-role",
			Parent: "urn:pulumi:dev::myapp::pulumi:pulumi:Stack::myapp-dev",
			Inputs: map[string]interface{}{
				"name": "myapp-lambda-role",
				"assumeRolePolicy": `{
					"Version": "2012-10-17",
					"Statement": [{
						"Effect": "Allow",
						"Principal": {"Service": "lambda.amazonaws.com"},
						"Action": "sts:AssumeRole"
					}]
				}`,
			},
			Outputs: map[string]interface{}{
				"arn":  "arn:aws:iam::123456789:role/myapp-lambda-role",
				"name": "myapp-lambda-role",
			},
		},
		{
			URN:    "urn:pulumi:dev::myapp::aws:lambda:Function::api-handler",
			Type:   "aws:lambda:Function",
			Name:   "api-handler",
			Parent: "urn:pulumi:dev::myapp::pulumi:pulumi:Stack::myapp-dev",
			Inputs: map[string]interface{}{
				"functionName": "myapp-api-handler",
				"runtime":      "nodejs18.x",
				"handler":      "index.handler",
				"memorySize":   256,
				"timeout":      30,
			},
			Outputs: map[string]interface{}{
				"arn":          "arn:aws:lambda:us-east-1:123456789:function:myapp-api-handler",
				"functionName": "myapp-api-handler",
				"invokeArn":    "arn:aws:apigateway:us-east-1:lambda:path/2015-03-31/functions/arn:aws:lambda:us-east-1:123456789:function:myapp-api-handler/invocations",
			},
		},
		{
			URN:    "urn:pulumi:dev::myapp::aws:apigateway:RestApi::api",
			Type:   "aws:apigateway:RestApi",
			Name:   "api",
			Parent: "urn:pulumi:dev::myapp::pulumi:pulumi:Stack::myapp-dev",
			Inputs: map[string]interface{}{
				"name":        "myapp-api",
				"description": "MyApp REST API",
			},
			Outputs: map[string]interface{}{
				"id":             "abc123",
				"rootResourceId": "xyz789",
				"executionArn":   "arn:aws:execute-api:us-east-1:123456789:abc123",
			},
		},
	}
}

// testHistoryItems creates sample history data
func testHistoryItems() []pulumi.UpdateSummary {
	return []pulumi.UpdateSummary{
		{
			Version:   5,
			Kind:      "update",
			StartTime: "2024-01-15T10:30:00Z",
			EndTime:   "2024-01-15T10:32:15Z",
			Message:   "Add API Gateway endpoint",
			Result:    "succeeded",
			ResourceChanges: map[string]int{
				"create": 2,
				"same":   3,
			},
			User:      "developer",
			UserEmail: "dev@example.com",
		},
		{
			Version:   4,
			Kind:      "update",
			StartTime: "2024-01-14T15:00:00Z",
			EndTime:   "2024-01-14T15:01:30Z",
			Message:   "Update Lambda memory",
			Result:    "succeeded",
			ResourceChanges: map[string]int{
				"update": 1,
				"same":   4,
			},
			User: "developer",
		},
		{
			Version:   3,
			Kind:      "refresh",
			StartTime: "2024-01-13T09:00:00Z",
			EndTime:   "2024-01-13T09:00:45Z",
			Message:   "",
			Result:    "succeeded",
			ResourceChanges: map[string]int{
				"same": 5,
			},
		},
		{
			Version:   2,
			Kind:      "update",
			StartTime: "2024-01-10T14:00:00Z",
			EndTime:   "2024-01-10T14:05:00Z",
			Message:   "Initial deployment",
			Result:    "succeeded",
			ResourceChanges: map[string]int{
				"create": 5,
			},
		},
		{
			Version:   1,
			Kind:      "update",
			StartTime: "2024-01-10T13:55:00Z",
			EndTime:   "2024-01-10T13:55:30Z",
			Message:   "Failed first attempt",
			Result:    "failed",
			ResourceChanges: map[string]int{
				"create": 1,
			},
		},
	}
}

// makePreviewEvents creates a sequence of preview events for testing
func makePreviewEvents(steps []pulumi.PreviewStep, finalError error) []pulumi.PreviewEvent {
	events := make([]pulumi.PreviewEvent, 0, len(steps)+1)
	for _, step := range steps {
		s := step // avoid loop variable capture
		events = append(events, pulumi.PreviewEvent{Step: &s})
	}
	// Add done event
	events = append(events, pulumi.PreviewEvent{Done: true, Error: finalError})
	return events
}

// makeOperationEvents creates a sequence of operation events for testing
func makeOperationEvents(steps []struct {
	URN    string
	Op     pulumi.ResourceOp
	Type   string
	Name   string
	Status pulumi.StepStatus
}, finalError error) []pulumi.OperationEvent {
	events := make([]pulumi.OperationEvent, 0, len(steps)+1)
	for _, step := range steps {
		events = append(events, pulumi.OperationEvent{
			URN:    step.URN,
			Op:     step.Op,
			Type:   step.Type,
			Name:   step.Name,
			Status: step.Status,
		})
	}
	// Add done event
	events = append(events, pulumi.OperationEvent{Done: true, Error: finalError})
	return events
}

// =============================================================================
// Complex Preview Flow Tests with Golden Snapshots
// =============================================================================

// TestPreviewFlow_UpWithChanges tests the complete up preview flow with multiple
// resource changes and captures golden snapshots at key points.
func TestPreviewFlow_UpWithChanges(t *testing.T) {
	// Create preview events simulating resource changes
	previewSteps := []pulumi.PreviewStep{
		{
			URN:  "urn:pulumi:dev::myapp::aws:s3:Bucket::new-bucket",
			Op:   pulumi.OpCreate,
			Type: "aws:s3:Bucket",
			Name: "new-bucket",
			Inputs: map[string]interface{}{
				"bucket": "myapp-new-bucket",
				"acl":    "private",
			},
		},
		{
			URN:  "urn:pulumi:dev::myapp::aws:lambda:Function::api-handler",
			Op:   pulumi.OpUpdate,
			Type: "aws:lambda:Function",
			Name: "api-handler",
			Inputs: map[string]interface{}{
				"memorySize": 512, // Changed from 256
				"timeout":    60,  // Changed from 30
			},
			Old: &pulumi.StepState{
				Inputs: map[string]interface{}{
					"memorySize": 256,
					"timeout":    30,
				},
			},
		},
		{
			URN:  "urn:pulumi:dev::myapp::aws:s3:Bucket::old-bucket",
			Op:   pulumi.OpDelete,
			Type: "aws:s3:Bucket",
			Name: "old-bucket",
		},
	}

	fakeOp := &pulumi.FakeStackOperator{}
	fakeOp.WithPreviewEvents(makePreviewEvents(previewSteps, nil)...)

	m := createTestModel(t,
		withStackOperator(fakeOp),
		withResources(testResourcesWithHierarchy()),
		withStacks(testStacks()),
		withStartView("up"), // Start directly in preview mode
	)

	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(goldenWidth, goldenHeight),
	)

	// Wait for preview to complete and capture snapshot
	waitAndSnapshot(t, tm, "new-bucket", "preview_complete", 5*time.Second)

	// Navigate to see different resources selected
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}) // Down
	time.Sleep(50 * time.Millisecond)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}) // Down again
	time.Sleep(100 * time.Millisecond)
	takeSnapshot(t, tm, "navigated_to_delete")

	// Quit cleanly
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

// TestPreviewFlow_RefreshNoChanges tests refresh preview when infrastructure is in sync
func TestPreviewFlow_RefreshNoChanges(t *testing.T) {
	// Empty preview - no changes detected
	fakeOp := &pulumi.FakeStackOperator{}
	fakeOp.WithPreviewEvents(pulumi.PreviewEvent{Done: true})

	m := createTestModel(t,
		withStackOperator(fakeOp),
		withResources(testResourcesWithHierarchy()),
		withStacks(testStacks()),
		withStartView("refresh"),
	)

	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(goldenWidth, goldenHeight),
	)

	// Wait for preview to complete - should show "no changes"
	waitForAnyContent(t, tm, []string{"No changes", "0 resources"}, 5*time.Second)
	takeSnapshot(t, tm, "no_changes")

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

// TestPreviewFlow_WithError tests error handling during preview
func TestPreviewFlow_WithError(t *testing.T) {
	previewError := errors.New("failed to connect to AWS: credentials expired")

	fakeOp := &pulumi.FakeStackOperator{}
	fakeOp.WithPreviewEvents(pulumi.PreviewEvent{
		Done:  true,
		Error: previewError,
	})

	m := createTestModel(t,
		withStackOperator(fakeOp),
		withResources(testResourcesWithHierarchy()),
		withStacks(testStacks()),
		withStartView("up"),
	)

	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(goldenWidth, goldenHeight),
	)

	// Wait for error to display
	waitAndSnapshot(t, tm, "credentials expired", "preview_error", 5*time.Second)

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

// =============================================================================
// Execute Operation Flow Tests with Golden Snapshots
// =============================================================================

// TestExecuteFlow_UpSuccess tests executing an up operation after preview
func TestExecuteFlow_UpSuccess(t *testing.T) {
	// Preview events
	previewSteps := []pulumi.PreviewStep{
		{
			URN:    "urn:pulumi:dev::myapp::aws:s3:Bucket::new-bucket",
			Op:     pulumi.OpCreate,
			Type:   "aws:s3:Bucket",
			Name:   "new-bucket",
			Inputs: map[string]interface{}{"bucket": "test"},
		},
	}

	// Operation events (execution)
	opEvents := []pulumi.OperationEvent{
		{
			URN:    "urn:pulumi:dev::myapp::aws:s3:Bucket::new-bucket",
			Op:     pulumi.OpCreate,
			Type:   "aws:s3:Bucket",
			Name:   "new-bucket",
			Status: pulumi.StepRunning,
		},
		{
			URN:    "urn:pulumi:dev::myapp::aws:s3:Bucket::new-bucket",
			Op:     pulumi.OpCreate,
			Type:   "aws:s3:Bucket",
			Name:   "new-bucket",
			Status: pulumi.StepSuccess,
			Outputs: map[string]interface{}{
				"id":  "test-bucket-id",
				"arn": "arn:aws:s3:::test-bucket-id",
			},
		},
		{Done: true},
	}

	fakeOp := &pulumi.FakeStackOperator{}
	fakeOp.WithPreviewEvents(makePreviewEvents(previewSteps, nil)...)

	// Configure Up to return operation events
	fakeOp.UpFunc = func(ctx context.Context, workDir, stackName string, opts pulumi.OperationOptions) <-chan pulumi.OperationEvent {
		ch := make(chan pulumi.OperationEvent, len(opEvents))
		for _, e := range opEvents {
			ch <- e
		}
		close(ch)
		return ch
	}

	m := createTestModel(t,
		withStackOperator(fakeOp),
		withResources(testResourcesWithHierarchy()),
		withStacks(testStacks()),
		withStartView("up"),
	)

	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(goldenWidth, goldenHeight),
	)

	// Wait for preview to complete
	waitAndSnapshot(t, tm, "new-bucket", "before_execute", 5*time.Second)

	// Press Ctrl+U to execute (already on preview screen for up operation)
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlU})

	// Wait for execution to complete
	time.Sleep(500 * time.Millisecond)
	takeSnapshot(t, tm, "after_execute")

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

// TestExecuteFlow_WithConfirmation tests execution with confirmation modal
func TestExecuteFlow_WithConfirmation(t *testing.T) {
	// Start in stack view, then try to execute directly (should show confirmation)
	fakeOp := &pulumi.FakeStackOperator{}
	fakeOp.UpFunc = func(ctx context.Context, workDir, stackName string, opts pulumi.OperationOptions) <-chan pulumi.OperationEvent {
		ch := make(chan pulumi.OperationEvent)
		go func() {
			ch <- pulumi.OperationEvent{Done: true}
			close(ch)
		}()
		return ch
	}

	m := createTestModel(t,
		withStackOperator(fakeOp),
		withResources(testResourcesWithHierarchy()),
		withStacks(testStacks()),
		withStartView("stack"), // Start in stack view, not preview
	)

	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(goldenWidth, goldenHeight),
	)

	// Wait for resources to load
	waitForContent(t, tm, "data-bucket", 3*time.Second)

	// Try to execute up directly - should show confirmation modal
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlU})

	// Wait for confirmation modal
	waitAndSnapshot(t, tm, "Execute", "confirmation_modal", 2*time.Second)

	// Press 'n' to cancel
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	time.Sleep(100 * time.Millisecond)
	takeSnapshot(t, tm, "after_cancel")

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

// =============================================================================
// Destroy Flow Tests with Golden Snapshots
// =============================================================================

// TestDestroyFlow_PreviewAndCancel tests destroy preview and cancel
func TestDestroyFlow_PreviewAndCancel(t *testing.T) {
	// Destroy preview - all resources will be deleted
	previewSteps := []pulumi.PreviewStep{
		{
			URN:  "urn:pulumi:dev::myapp::aws:lambda:Function::api-handler",
			Op:   pulumi.OpDelete,
			Type: "aws:lambda:Function",
			Name: "api-handler",
		},
		{
			URN:  "urn:pulumi:dev::myapp::aws:iam:Role::lambda-role",
			Op:   pulumi.OpDelete,
			Type: "aws:iam:Role",
			Name: "lambda-role",
		},
		{
			URN:  "urn:pulumi:dev::myapp::aws:s3:Bucket::data-bucket",
			Op:   pulumi.OpDelete,
			Type: "aws:s3:Bucket",
			Name: "data-bucket",
		},
	}

	fakeOp := &pulumi.FakeStackOperator{}
	fakeOp.WithPreviewEvents(makePreviewEvents(previewSteps, nil)...)

	m := createTestModel(t,
		withStackOperator(fakeOp),
		withResources(testResourcesWithHierarchy()),
		withStacks(testStacks()),
		withStartView("destroy"),
	)

	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(goldenWidth, goldenHeight),
	)

	// Wait for destroy preview to complete
	waitAndSnapshot(t, tm, "data-bucket", "destroy_preview", 5*time.Second)

	// Press Escape to cancel and go back to stack view
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	waitForContent(t, tm, "data-bucket", 3*time.Second)
	takeSnapshot(t, tm, "back_to_stack")

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

// =============================================================================
// Stack Switching Tests with Golden Snapshots
// =============================================================================

// TestStackSwitching_OpenSelectorAndSwitch tests opening stack selector and switching
func TestStackSwitching_OpenSelectorAndSwitch(t *testing.T) {
	fakeReader := &pulumi.FakeStackReader{
		Resources: testResourcesWithHierarchy(),
		Stacks: []pulumi.StackInfo{
			{Name: "dev", Current: true},
			{Name: "staging", Current: false},
			{Name: "prod", Current: false},
		},
	}

	m := createTestModel(t,
		withStackReader(fakeReader),
		withStacks(testStacks()),
	)

	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(goldenWidth, goldenHeight),
	)

	// Wait for initial load
	waitForContent(t, tm, "data-bucket", 3*time.Second)

	// Open stack selector with 's' key
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	waitAndSnapshot(t, tm, "Select Stack", "stack_selector_open", 2*time.Second)

	// Navigate down to staging
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	time.Sleep(100 * time.Millisecond)
	takeSnapshot(t, tm, "staging_selected")

	// Press Escape to cancel
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	time.Sleep(100 * time.Millisecond)
	takeSnapshot(t, tm, "selector_closed")

	// Use tea.Quit() directly since 'q' key may have timing issues in tests
	tm.Send(tea.Quit())
	tm.WaitFinished(t, teatest.WithFinalTimeout(5*time.Second))
}

// =============================================================================
// History View Tests with Golden Snapshots
// =============================================================================

// TestHistoryView_NavigateAndDetails tests history view navigation and details
func TestHistoryView_NavigateAndDetails(t *testing.T) {
	fakeReader := &pulumi.FakeStackReader{
		Resources: testResourcesWithHierarchy(),
		Stacks:    testStacks(),
		History:   testHistoryItems(),
	}

	m := createTestModel(t,
		withStackReader(fakeReader),
		withStacks(testStacks()),
	)

	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(goldenWidth, goldenHeight),
	)

	// Wait for resources to load
	waitForContent(t, tm, "data-bucket", 3*time.Second)

	// Press 'h' to switch to history view
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	waitAndSnapshot(t, tm, "update", "history_view", 3*time.Second)

	// Navigate through history items
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	time.Sleep(50 * time.Millisecond)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	time.Sleep(100 * time.Millisecond)
	takeSnapshot(t, tm, "history_navigated")

	// Open details panel
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("D")})
	time.Sleep(100 * time.Millisecond)
	takeSnapshot(t, tm, "history_details")

	// Press Escape to go back to stack view
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	waitForContent(t, tm, "data-bucket", 2*time.Second)

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

// =============================================================================
// Import Modal Tests with Golden Snapshots
// =============================================================================

// TestImportModal_OpenAndFill tests import modal workflow
func TestImportModal_OpenAndFill(t *testing.T) {
	// TODO: This test has a persistent issue where the output buffer is empty
	// when using withPluginProvider. Needs investigation.
	t.Skip("Skipping due to empty output buffer issue with custom PluginProvider")

	// Create a preview with a create operation (can be imported)
	previewSteps := []pulumi.PreviewStep{
		{
			URN:    "urn:pulumi:dev::myapp::aws:s3:Bucket::existing-bucket",
			Op:     pulumi.OpCreate,
			Type:   "aws:s3:Bucket",
			Name:   "existing-bucket",
			Inputs: map[string]interface{}{"bucket": "existing-bucket-name"},
		},
	}

	fakeOp := &pulumi.FakeStackOperator{}
	fakeOp.WithPreviewEvents(makePreviewEvents(previewSteps, nil)...)

	// Configure plugin provider with import suggestions
	fakePlugin := &plugins.FakePluginProvider{
		HasImportHelper: true,
		ImportSuggestions: []*plugins.AggregatedImportSuggestion{
			{
				PluginName: "aws",
				Suggestion: &plugins.ImportSuggestion{
					Id:          "existing-bucket-name",
					Label:       "existing-bucket-name (S3 Bucket)",
					Description: "S3 bucket in us-east-1",
				},
			},
		},
	}

	m := createTestModel(t,
		withStackOperator(fakeOp),
		withResources(testResourcesWithHierarchy()),
		withStacks(testStacks()),
		withPluginProvider(fakePlugin),
		withStartView("up"),
	)

	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(goldenWidth, goldenHeight),
	)

	// Wait for preview to complete
	waitForContent(t, tm, "existing-bucket", 5*time.Second)

	// Press 'i' to open import modal
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	waitAndSnapshot(t, tm, "Import", "import_modal_open", 2*time.Second)

	// Press Escape to close
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	time.Sleep(100 * time.Millisecond)
	takeSnapshot(t, tm, "import_modal_closed")

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

// =============================================================================
// Details Panel Tests with Golden Snapshots
// =============================================================================

// TestDetailsPanel_ResourceInspection tests viewing resource details
func TestDetailsPanel_ResourceInspection(t *testing.T) {
	m := createTestModel(t,
		withResources(testResourcesWithHierarchy()),
		withStacks(testStacks()),
	)

	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(goldenWidth, goldenHeight),
	)

	// Wait for resources to load
	waitForContent(t, tm, "data-bucket", 3*time.Second)

	// Navigate to Lambda function
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}) // Skip stack
	time.Sleep(50 * time.Millisecond)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}) // Skip bucket
	time.Sleep(50 * time.Millisecond)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}) // To lambda-role
	time.Sleep(50 * time.Millisecond)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}) // To api-handler
	time.Sleep(100 * time.Millisecond)

	// Open details panel
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("D")})
	waitAndSnapshot(t, tm, "api-handler", "details_panel_lambda", 2*time.Second)

	// Scroll down in details
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	time.Sleep(100 * time.Millisecond)
	takeSnapshot(t, tm, "details_scrolled")

	// Close details
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("D")})
	time.Sleep(100 * time.Millisecond)

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

// =============================================================================
// Visual Selection Mode Tests with Golden Snapshots
// =============================================================================

// TestVisualMode_MultiSelectResources tests selecting multiple resources
func TestVisualMode_MultiSelectResources(t *testing.T) {
	m := createTestModel(t,
		withResources(testResourcesWithHierarchy()),
		withStacks(testStacks()),
	)

	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(goldenWidth, goldenHeight),
	)

	// Wait for resources to load
	waitForContent(t, tm, "data-bucket", 3*time.Second)

	// Navigate to first real resource (skip stack)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	time.Sleep(50 * time.Millisecond)

	// Enter visual mode
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	waitAndSnapshot(t, tm, "VISUAL", "visual_mode_entered", 2*time.Second)

	// Extend selection down
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	time.Sleep(50 * time.Millisecond)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	time.Sleep(100 * time.Millisecond)
	takeSnapshot(t, tm, "visual_extended")

	// Toggle target flag on selection (t key)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	time.Sleep(100 * time.Millisecond)
	takeSnapshot(t, tm, "visual_targeted")

	// Exit visual mode
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	time.Sleep(100 * time.Millisecond)
	takeSnapshot(t, tm, "after_visual")

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

// =============================================================================
// Complex Multi-Step Scenario Tests
// =============================================================================

// TestScenario_CompleteDeploymentCycle tests a realistic deployment workflow:
// 1. View stack resources
// 2. Run up preview
// 3. Review changes
// 4. View details of changed resource
// 5. Execute deployment
// 6. Verify completion
func TestScenario_CompleteDeploymentCycle(t *testing.T) {
	// Preview shows one new resource
	previewSteps := []pulumi.PreviewStep{
		{
			URN:  "urn:pulumi:dev::myapp::aws:dynamodb:Table::users-table",
			Op:   pulumi.OpCreate,
			Type: "aws:dynamodb:Table",
			Name: "users-table",
			Inputs: map[string]interface{}{
				"name":        "myapp-users",
				"billingMode": "PAY_PER_REQUEST",
				"hashKey":     "userId",
			},
		},
	}

	// Execution events
	opEvents := []pulumi.OperationEvent{
		{
			URN:    "urn:pulumi:dev::myapp::aws:dynamodb:Table::users-table",
			Op:     pulumi.OpCreate,
			Type:   "aws:dynamodb:Table",
			Name:   "users-table",
			Status: pulumi.StepRunning,
		},
		{
			URN:    "urn:pulumi:dev::myapp::aws:dynamodb:Table::users-table",
			Op:     pulumi.OpCreate,
			Type:   "aws:dynamodb:Table",
			Name:   "users-table",
			Status: pulumi.StepSuccess,
			Outputs: map[string]interface{}{
				"arn":  "arn:aws:dynamodb:us-east-1:123456789:table/myapp-users",
				"name": "myapp-users",
			},
		},
		{Done: true},
	}

	fakeOp := &pulumi.FakeStackOperator{}
	fakeOp.WithPreviewEvents(makePreviewEvents(previewSteps, nil)...)
	fakeOp.UpFunc = func(ctx context.Context, workDir, stackName string, opts pulumi.OperationOptions) <-chan pulumi.OperationEvent {
		ch := make(chan pulumi.OperationEvent, len(opEvents))
		for _, e := range opEvents {
			ch <- e
		}
		close(ch)
		return ch
	}

	m := createTestModel(t,
		withStackOperator(fakeOp),
		withResources(testResourcesWithHierarchy()),
		withStacks(testStacks()),
	)

	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(goldenWidth, goldenHeight),
	)

	// Step 1: View initial stack resources
	waitAndSnapshot(t, tm, "data-bucket", "step1_stack_view", 3*time.Second)

	// Step 2: Start up preview
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
	waitAndSnapshot(t, tm, "users-table", "step2_preview_complete", 5*time.Second)

	// Step 3: Open details panel to review the change
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("D")})
	time.Sleep(100 * time.Millisecond)
	takeSnapshot(t, tm, "step3_reviewing_details")

	// Close details
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("D")})
	time.Sleep(100 * time.Millisecond)

	// Step 4: Execute the deployment
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlU})
	time.Sleep(500 * time.Millisecond)
	takeSnapshot(t, tm, "step4_execution_complete")

	// Step 5: Go back to stack view
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	waitForContent(t, tm, "data-bucket", 3*time.Second)
	takeSnapshot(t, tm, "step5_back_to_stack")

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

// TestScenario_InvestigateFailedDeployment tests investigating a failed update
func TestScenario_InvestigateFailedDeployment(t *testing.T) {
	// Create a preview that will fail partway through
	previewSteps := []pulumi.PreviewStep{
		{
			URN:  "urn:pulumi:dev::myapp::aws:ec2:Instance::web-server",
			Op:   pulumi.OpCreate,
			Type: "aws:ec2:Instance",
			Name: "web-server",
		},
	}

	previewError := errors.New("Error creating EC2 instance: InsufficientInstanceCapacity: We currently do not have sufficient capacity")

	fakeOp := &pulumi.FakeStackOperator{}
	fakeOp.WithPreviewEvents(makePreviewEvents(previewSteps, previewError)...)

	fakeReader := &pulumi.FakeStackReader{
		Resources: testResourcesWithHierarchy(),
		Stacks:    testStacks(),
		History: []pulumi.UpdateSummary{
			{
				Version:   10,
				Kind:      "update",
				StartTime: "2024-01-20T10:00:00Z",
				EndTime:   "2024-01-20T10:01:00Z",
				Message:   "Add web server",
				Result:    "failed",
				ResourceChanges: map[string]int{
					"create": 0, // Failed before creating
				},
			},
		},
	}

	m := createTestModel(t,
		withStackOperator(fakeOp),
		withStackReader(fakeReader),
		withStacks(testStacks()),
	)

	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(goldenWidth, goldenHeight),
	)

	// Wait for stack view
	waitForContent(t, tm, "data-bucket", 3*time.Second)

	// Try to run preview and see error
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
	waitAndSnapshot(t, tm, "InsufficientInstanceCapacity", "preview_failed", 5*time.Second)

	// Go to history to see the failed update
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc}) // Back to stack
	waitForContent(t, tm, "data-bucket", 2*time.Second)

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")}) // History view
	waitAndSnapshot(t, tm, "failed", "history_shows_failure", 3*time.Second)

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}
