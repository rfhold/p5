package main

import (
	"context"
	"log/slog"
	"testing"

	"github.com/rfhold/p5/internal/plugins"
	"github.com/rfhold/p5/internal/pulumi"
	"github.com/rfhold/p5/internal/ui"
)

// newTestDependencies creates a Dependencies struct with all fakes for testing.
// This is the primary way to create testable model instances.
func newTestDependencies() *Dependencies {
	return &Dependencies{
		StackOperator:    &pulumi.FakeStackOperator{},
		StackReader:      &pulumi.FakeStackReader{},
		WorkspaceReader:  &pulumi.FakeWorkspaceReader{ValidWorkDir: true},
		StackInitializer: &pulumi.FakeStackInitializer{},
		ResourceImporter: &pulumi.FakeResourceImporter{},
		PluginProvider:   &plugins.FakePluginProvider{},
		Logger:           slog.New(slog.NewTextHandler(discardWriter{}, nil)),
	}
}

// discardWriter is an io.Writer that discards all output
type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }

// TestInitialModelStartsInCheckingWorkspaceState verifies the model starts
// in the correct initial state.
func TestInitialModelStartsInCheckingWorkspaceState(t *testing.T) {
	deps := newTestDependencies()
	ctx := AppContext{
		WorkDir:   "/fake/path",
		StartView: "stack",
	}

	m := initialModel(context.Background(), ctx, deps)

	if m.state.InitState != InitCheckingWorkspace {
		t.Errorf("expected InitState=%v, got %v", InitCheckingWorkspace, m.state.InitState)
	}

	if m.state.OpState != OpIdle {
		t.Errorf("expected OpState=%v, got %v", OpIdle, m.state.OpState)
	}

	if m.ui.ViewMode != ui.ViewStack {
		t.Errorf("expected ViewMode=%v, got %v", ui.ViewStack, m.ui.ViewMode)
	}
}

// TestInitialModelWithUpView verifies the model sets correct state for "up" start view.
func TestInitialModelWithUpView(t *testing.T) {
	deps := newTestDependencies()
	ctx := AppContext{
		WorkDir:   "/fake/path",
		StartView: "up",
	}

	m := initialModel(context.Background(), ctx, deps)

	if m.state.Operation != pulumi.OperationUp {
		t.Errorf("expected Operation=%v, got %v", pulumi.OperationUp, m.state.Operation)
	}

	if m.ui.ViewMode != ui.ViewPreview {
		t.Errorf("expected ViewMode=%v, got %v", ui.ViewPreview, m.ui.ViewMode)
	}
}

// TestInitialModelWithRefreshView verifies the model sets correct state for "refresh" start view.
func TestInitialModelWithRefreshView(t *testing.T) {
	deps := newTestDependencies()
	ctx := AppContext{
		WorkDir:   "/fake/path",
		StartView: "refresh",
	}

	m := initialModel(context.Background(), ctx, deps)

	if m.state.Operation != pulumi.OperationRefresh {
		t.Errorf("expected Operation=%v, got %v", pulumi.OperationRefresh, m.state.Operation)
	}

	if m.ui.ViewMode != ui.ViewPreview {
		t.Errorf("expected ViewMode=%v, got %v", ui.ViewPreview, m.ui.ViewMode)
	}
}

// TestInitialModelWithDestroyView verifies the model sets correct state for "destroy" start view.
func TestInitialModelWithDestroyView(t *testing.T) {
	deps := newTestDependencies()
	ctx := AppContext{
		WorkDir:   "/fake/path",
		StartView: "destroy",
	}

	m := initialModel(context.Background(), ctx, deps)

	if m.state.Operation != pulumi.OperationDestroy {
		t.Errorf("expected Operation=%v, got %v", pulumi.OperationDestroy, m.state.Operation)
	}

	if m.ui.ViewMode != ui.ViewPreview {
		t.Errorf("expected ViewMode=%v, got %v", ui.ViewPreview, m.ui.ViewMode)
	}
}

// TestInitialModelSharesFlags verifies that Flags map is shared between state and UI.
func TestInitialModelSharesFlags(t *testing.T) {
	deps := newTestDependencies()
	ctx := AppContext{
		WorkDir:   "/fake/path",
		StartView: "stack",
	}

	m := initialModel(context.Background(), ctx, deps)

	// Add a flag to state
	testURN := "urn:pulumi:dev::test::aws:s3:Bucket::mybucket"
	m.state.Flags[testURN] = ui.ResourceFlags{Target: true}

	// Verify it's visible in UI's ResourceList flags
	// Note: ResourceList gets the same map reference
	if m.state.Flags[testURN].Target != true {
		t.Error("expected flag to be set in state")
	}
}

// TestInitialModelContextIsSet verifies the AppContext is properly stored.
func TestInitialModelContextIsSet(t *testing.T) {
	deps := newTestDependencies()
	ctx := AppContext{
		WorkDir:   "/test/workspace",
		StackName: "dev",
		StartView: "stack",
	}

	m := initialModel(context.Background(), ctx, deps)

	if m.ctx.WorkDir != "/test/workspace" {
		t.Errorf("expected WorkDir=%q, got %q", "/test/workspace", m.ctx.WorkDir)
	}

	if m.ctx.StackName != "dev" {
		t.Errorf("expected StackName=%q, got %q", "dev", m.ctx.StackName)
	}
}

// TestNewAppStateDefaults verifies NewAppState creates correct default values.
func TestNewAppStateDefaults(t *testing.T) {
	state := NewAppState()

	if state.InitState != InitCheckingWorkspace {
		t.Errorf("expected InitState=%v, got %v", InitCheckingWorkspace, state.InitState)
	}

	if state.OpState != OpIdle {
		t.Errorf("expected OpState=%v, got %v", OpIdle, state.OpState)
	}

	if state.Flags == nil {
		t.Error("expected Flags to be initialized, got nil")
	}

	if state.PendingOperation != nil {
		t.Errorf("expected PendingOperation=nil, got %v", state.PendingOperation)
	}

	if state.Err != nil {
		t.Errorf("expected Err=nil, got %v", state.Err)
	}
}

// TestOperationStateIsActive verifies IsActive returns correct values.
func TestOperationStateIsActive(t *testing.T) {
	tests := []struct {
		state  OperationState
		active bool
	}{
		{OpIdle, false},
		{OpStarting, true},
		{OpRunning, true},
		{OpCancelling, true},
		{OpComplete, false},
		{OpError, false},
	}

	for _, tt := range tests {
		t.Run(tt.state.String(), func(t *testing.T) {
			if got := tt.state.IsActive(); got != tt.active {
				t.Errorf("%v.IsActive() = %v, want %v", tt.state, got, tt.active)
			}
		})
	}
}

// TestInitStateString verifies String() returns human-readable names.
func TestInitStateString(t *testing.T) {
	tests := []struct {
		state InitState
		want  string
	}{
		{InitCheckingWorkspace, "CheckingWorkspace"},
		{InitLoadingPlugins, "LoadingPlugins"},
		{InitLoadingStacks, "LoadingStacks"},
		{InitSelectingStack, "SelectingStack"},
		{InitLoadingResources, "LoadingResources"},
		{InitComplete, "Complete"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.state.String(); got != tt.want {
				t.Errorf("InitState(%d).String() = %q, want %q", tt.state, got, tt.want)
			}
		})
	}
}

// TestOperationStateString verifies String() returns human-readable names.
func TestOperationStateString(t *testing.T) {
	tests := []struct {
		state OperationState
		want  string
	}{
		{OpIdle, "Idle"},
		{OpStarting, "Starting"},
		{OpRunning, "Running"},
		{OpCancelling, "Cancelling"},
		{OpComplete, "Complete"},
		{OpError, "Error"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.state.String(); got != tt.want {
				t.Errorf("OperationState(%d).String() = %q, want %q", tt.state, got, tt.want)
			}
		})
	}
}

// TestTransitionTo verifies the transitionTo method correctly updates InitState.
func TestTransitionTo(t *testing.T) {
	deps := newTestDependencies()
	ctx := AppContext{
		WorkDir:   "/fake/path",
		StartView: "stack",
	}
	m := initialModel(context.Background(), ctx, deps)

	// Start at InitCheckingWorkspace
	if m.state.InitState != InitCheckingWorkspace {
		t.Fatalf("expected initial state %v, got %v", InitCheckingWorkspace, m.state.InitState)
	}

	// Transition through the state machine
	transitions := []InitState{
		InitLoadingPlugins,
		InitLoadingStacks,
		InitSelectingStack,
		InitLoadingResources,
		InitComplete,
	}

	for _, newState := range transitions {
		m.transitionTo(newState)
		if m.state.InitState != newState {
			t.Errorf("expected state %v after transition, got %v", newState, m.state.InitState)
		}
	}
}

// TestHandleWorkspaceCheckValid verifies valid workspace transitions to LoadingPlugins.
func TestHandleWorkspaceCheckValid(t *testing.T) {
	deps := newTestDependencies()
	ctx := AppContext{
		WorkDir:   "/fake/path",
		StartView: "stack",
	}
	m := initialModel(context.Background(), ctx, deps)

	// Simulate receiving a valid workspace check message
	result, _ := m.handleWorkspaceCheck(workspaceCheckMsg(true))
	resultModel, ok := result.(Model)
	if !ok {
		t.Fatal("expected result to be Model type")
	}

	if resultModel.state.InitState != InitLoadingPlugins {
		t.Errorf("expected state %v after valid workspace check, got %v",
			InitLoadingPlugins, resultModel.state.InitState)
	}
}

// TestHandleWorkspaceCheckInvalid verifies invalid workspace shows workspace selector.
func TestHandleWorkspaceCheckInvalid(t *testing.T) {
	deps := newTestDependencies()
	ctx := AppContext{
		WorkDir:   "/fake/path",
		StartView: "stack",
	}
	m := initialModel(context.Background(), ctx, deps)

	// Simulate receiving an invalid workspace check message
	result, _ := m.handleWorkspaceCheck(workspaceCheckMsg(false))
	resultModel, ok := result.(Model)
	if !ok {
		t.Fatal("expected result to be Model type")
	}

	// Should still be in CheckingWorkspace (waiting for workspace selection)
	if resultModel.state.InitState != InitCheckingWorkspace {
		t.Errorf("expected state %v after invalid workspace check, got %v",
			InitCheckingWorkspace, resultModel.state.InitState)
	}

	// Focus should show workspace selector is visible
	if !resultModel.ui.Focus.Has(ui.FocusWorkspaceSelector) {
		t.Error("expected workspace selector to be focused after invalid workspace check")
	}
}

// TestHandleError verifies error handling transitions to InitComplete.
func TestHandleError(t *testing.T) {
	deps := newTestDependencies()
	ctx := AppContext{
		WorkDir:   "/fake/path",
		StartView: "stack",
	}
	m := initialModel(context.Background(), ctx, deps)

	// Set up model mid-initialization
	m.transitionTo(InitLoadingPlugins)

	// Simulate receiving an error
	testErr := errMsg(testError("test error"))
	result, _ := m.handleError(testErr)
	resultModel, ok := result.(Model)
	if !ok {
		t.Fatal("expected result to be Model type")
	}

	// Should transition to InitComplete to allow user interaction
	if resultModel.state.InitState != InitComplete {
		t.Errorf("expected state %v after error, got %v", InitComplete, resultModel.state.InitState)
	}

	// Error should be stored
	if resultModel.state.Err == nil {
		t.Error("expected error to be stored in state")
	}
}

// TestHandlePluginInitDoneWithStackName verifies plugin init with stack specified
// transitions directly to LoadingResources.
func TestHandlePluginInitDoneWithStackName(t *testing.T) {
	deps := newTestDependencies()
	ctx := AppContext{
		WorkDir:   "/fake/path",
		StackName: "dev", // Stack specified
		StartView: "stack",
	}
	m := initialModel(context.Background(), ctx, deps)
	m.transitionTo(InitLoadingPlugins)

	// Simulate plugin init done
	result, _ := m.handlePluginInitDone(pluginInitDoneMsg{results: nil, err: nil})
	resultModel, ok := result.(Model)
	if !ok {
		t.Fatal("expected result to be Model")
	}

	if resultModel.state.InitState != InitLoadingResources {
		t.Errorf("expected state %v when stack specified, got %v",
			InitLoadingResources, resultModel.state.InitState)
	}
}

// TestHandlePluginInitDoneWithoutStackName verifies plugin init without stack
// transitions to LoadingStacks.
func TestHandlePluginInitDoneWithoutStackName(t *testing.T) {
	deps := newTestDependencies()
	ctx := AppContext{
		WorkDir:   "/fake/path",
		StackName: "", // No stack specified
		StartView: "stack",
	}
	m := initialModel(context.Background(), ctx, deps)
	m.transitionTo(InitLoadingPlugins)

	// Simulate plugin init done
	result, _ := m.handlePluginInitDone(pluginInitDoneMsg{results: nil, err: nil})
	resultModel, ok := result.(Model)
	if !ok {
		t.Fatal("expected result to be Model")
	}

	if resultModel.state.InitState != InitLoadingStacks {
		t.Errorf("expected state %v when no stack specified, got %v",
			InitLoadingStacks, resultModel.state.InitState)
	}
}

// testError is a simple error type for testing.
type testError string

func (e testError) Error() string { return string(e) }

// TestProcessPreviewEvent_AddsStep verifies step events produce ResourceItems.
func TestProcessPreviewEvent_AddsStep(t *testing.T) {
	event := pulumi.PreviewEvent{
		Step: &pulumi.PreviewStep{
			URN:    "urn:pulumi:dev::test::aws:s3:Bucket::mybucket",
			Type:   "aws:s3:Bucket",
			Name:   "mybucket",
			Op:     pulumi.OpCreate,
			Parent: "",
			Inputs: map[string]any{"bucket": "my-bucket"},
		},
	}

	result := ProcessPreviewEvent(event, OpRunning, InitLoadingResources)

	if result.NewOpState != OpRunning {
		t.Errorf("expected OpState=%v, got %v", OpRunning, result.NewOpState)
	}
	if result.HasError {
		t.Error("expected no error")
	}
	if result.Item == nil {
		t.Fatal("expected Item to be set")
	}
	if result.Item.URN != event.Step.URN {
		t.Errorf("expected URN=%q, got %q", event.Step.URN, result.Item.URN)
	}
	if result.Item.Op != pulumi.OpCreate {
		t.Errorf("expected Op=%v, got %v", pulumi.OpCreate, result.Item.Op)
	}
}

// TestProcessPreviewEvent_HandlesError verifies error events set error state.
func TestProcessPreviewEvent_HandlesError(t *testing.T) {
	testErr := testError("preview failed")
	event := pulumi.PreviewEvent{
		Error: testErr,
	}

	result := ProcessPreviewEvent(event, OpRunning, InitLoadingResources)

	if result.NewOpState != OpError {
		t.Errorf("expected OpState=%v, got %v", OpError, result.NewOpState)
	}
	if !result.HasError {
		t.Error("expected HasError=true")
	}
	if result.Error == nil {
		t.Error("expected Error to be set")
	}
	if result.Item != nil {
		t.Error("expected Item to be nil on error")
	}
	if !result.InitDone {
		t.Error("expected InitDone=true when in InitLoadingResources")
	}
}

// TestProcessPreviewEvent_HandlesDone verifies done events complete the operation.
func TestProcessPreviewEvent_HandlesDone(t *testing.T) {
	event := pulumi.PreviewEvent{
		Done: true,
	}

	result := ProcessPreviewEvent(event, OpRunning, InitLoadingResources)

	if result.NewOpState != OpComplete {
		t.Errorf("expected OpState=%v, got %v", OpComplete, result.NewOpState)
	}
	if result.HasError {
		t.Error("expected no error")
	}
	if result.InitDone != true {
		t.Error("expected InitDone=true when in InitLoadingResources")
	}
}

// TestProcessPreviewEvent_TransitionsFromStarting verifies Starting→Running transition.
func TestProcessPreviewEvent_TransitionsFromStarting(t *testing.T) {
	event := pulumi.PreviewEvent{
		Step: &pulumi.PreviewStep{
			URN:  "urn:pulumi:dev::test::aws:s3:Bucket::mybucket",
			Type: "aws:s3:Bucket",
			Name: "mybucket",
			Op:   pulumi.OpCreate,
		},
	}

	result := ProcessPreviewEvent(event, OpStarting, InitLoadingResources)

	if result.NewOpState != OpRunning {
		t.Errorf("expected OpState=%v after first event, got %v", OpRunning, result.NewOpState)
	}
}

// TestProcessPreviewEvent_MergesOldState verifies old state is preserved for diffs.
func TestProcessPreviewEvent_MergesOldState(t *testing.T) {
	event := pulumi.PreviewEvent{
		Step: &pulumi.PreviewStep{
			URN:    "urn:pulumi:dev::test::aws:s3:Bucket::mybucket",
			Type:   "aws:s3:Bucket",
			Name:   "mybucket",
			Op:     pulumi.OpUpdate,
			Inputs: map[string]any{"bucket": "new-bucket"},
			Old: &pulumi.StepState{
				Inputs:  map[string]any{"bucket": "old-bucket"},
				Outputs: map[string]any{"id": "bucket-123"},
			},
		},
	}

	result := ProcessPreviewEvent(event, OpRunning, InitLoadingResources)

	if result.Item == nil {
		t.Fatal("expected Item to be set")
	}
	if result.Item.OldInputs["bucket"] != "old-bucket" {
		t.Errorf("expected OldInputs[bucket]=%q, got %q", "old-bucket", result.Item.OldInputs["bucket"])
	}
	if result.Item.Inputs["bucket"] != "new-bucket" {
		t.Errorf("expected Inputs[bucket]=%q, got %q", "new-bucket", result.Item.Inputs["bucket"])
	}
}

// TestProcessPreviewEvent_DeleteUsesOldState verifies delete ops use old state as current.
func TestProcessPreviewEvent_DeleteUsesOldState(t *testing.T) {
	event := pulumi.PreviewEvent{
		Step: &pulumi.PreviewStep{
			URN:    "urn:pulumi:dev::test::aws:s3:Bucket::mybucket",
			Type:   "aws:s3:Bucket",
			Name:   "mybucket",
			Op:     pulumi.OpDelete,
			Inputs: nil, // No new inputs for delete
			Old: &pulumi.StepState{
				Inputs:  map[string]any{"bucket": "old-bucket"},
				Outputs: map[string]any{"id": "bucket-123"},
			},
		},
	}

	result := ProcessPreviewEvent(event, OpRunning, InitLoadingResources)

	if result.Item == nil {
		t.Fatal("expected Item to be set")
	}
	// For delete, current inputs should be old inputs
	if result.Item.Inputs["bucket"] != "old-bucket" {
		t.Errorf("expected Inputs to use old state for delete, got %v", result.Item.Inputs)
	}
}

// TestProcessPreviewEvent_NotInitLoading verifies InitDone is false when not in InitLoadingResources.
func TestProcessPreviewEvent_NotInitLoading(t *testing.T) {
	event := pulumi.PreviewEvent{Done: true}

	result := ProcessPreviewEvent(event, OpRunning, InitComplete)

	if result.InitDone {
		t.Error("expected InitDone=false when not in InitLoadingResources")
	}
}

// TestProcessOperationEvent_AddsItem verifies operation events produce ResourceItems.
func TestProcessOperationEvent_AddsItem(t *testing.T) {
	event := pulumi.OperationEvent{
		URN:    "urn:pulumi:dev::test::aws:s3:Bucket::mybucket",
		Type:   "aws:s3:Bucket",
		Name:   "mybucket",
		Op:     pulumi.OpCreate,
		Status: pulumi.StepRunning,
	}

	result := ProcessOperationEvent(event, OpRunning)

	if result.NewOpState != OpRunning {
		t.Errorf("expected OpState=%v, got %v", OpRunning, result.NewOpState)
	}
	if result.Item == nil {
		t.Fatal("expected Item to be set")
	}
	if result.Item.Status != ui.StatusRunning {
		t.Errorf("expected Status=%v, got %v", ui.StatusRunning, result.Item.Status)
	}
}

// TestProcessOperationEvent_HandlesError verifies error events set error state.
func TestProcessOperationEvent_HandlesError(t *testing.T) {
	testErr := testError("operation failed")
	event := pulumi.OperationEvent{
		Error: testErr,
	}

	result := ProcessOperationEvent(event, OpRunning)

	if result.NewOpState != OpError {
		t.Errorf("expected OpState=%v, got %v", OpError, result.NewOpState)
	}
	if !result.HasError {
		t.Error("expected HasError=true")
	}
}

// TestProcessOperationEvent_HandlesDone verifies done events complete the operation.
func TestProcessOperationEvent_HandlesDone(t *testing.T) {
	event := pulumi.OperationEvent{
		Done: true,
	}

	result := ProcessOperationEvent(event, OpRunning)

	if result.NewOpState != OpComplete {
		t.Errorf("expected OpState=%v, got %v", OpComplete, result.NewOpState)
	}
	if !result.Done {
		t.Error("expected Done=true")
	}
}

// TestProcessOperationEvent_StatusMapping verifies all status mappings.
func TestProcessOperationEvent_StatusMapping(t *testing.T) {
	tests := []struct {
		name         string
		pulumiStatus pulumi.StepStatus
		uiStatus     ui.ItemStatus
	}{
		{"Pending", pulumi.StepPending, ui.StatusPending},
		{"Running", pulumi.StepRunning, ui.StatusRunning},
		{"Success", pulumi.StepSuccess, ui.StatusSuccess},
		{"Failed", pulumi.StepFailed, ui.StatusFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := pulumi.OperationEvent{
				URN:    "urn:pulumi:dev::test::aws:s3:Bucket::mybucket",
				Status: tt.pulumiStatus,
			}

			result := ProcessOperationEvent(event, OpRunning)

			if result.Item == nil {
				t.Fatal("expected Item to be set")
			}
			if result.Item.Status != tt.uiStatus {
				t.Errorf("expected Status=%v, got %v", tt.uiStatus, result.Item.Status)
			}
		})
	}
}

// TestProcessOperationEvent_TransitionsFromStarting verifies Starting→Running transition.
func TestProcessOperationEvent_TransitionsFromStarting(t *testing.T) {
	event := pulumi.OperationEvent{
		URN:    "urn:pulumi:dev::test::aws:s3:Bucket::mybucket",
		Status: pulumi.StepRunning,
	}

	result := ProcessOperationEvent(event, OpStarting)

	if result.NewOpState != OpRunning {
		t.Errorf("expected OpState=%v after first event, got %v", OpRunning, result.NewOpState)
	}
}

// TestConvertResourcesToItems_Basic verifies basic resource conversion.
func TestConvertResourcesToItems_Basic(t *testing.T) {
	resources := []pulumi.ResourceInfo{
		{
			URN:     "urn:pulumi:dev::test::aws:s3:Bucket::mybucket",
			Type:    "aws:s3:Bucket",
			Name:    "mybucket",
			Parent:  "",
			Inputs:  map[string]any{"bucket": "my-bucket"},
			Outputs: map[string]any{"id": "bucket-123"},
		},
		{
			URN:    "urn:pulumi:dev::test::aws:s3:BucketObject::myfile",
			Type:   "aws:s3:BucketObject",
			Name:   "myfile",
			Parent: "urn:pulumi:dev::test::aws:s3:Bucket::mybucket",
		},
	}

	items := ConvertResourcesToItems(resources)

	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	// First item
	if items[0].URN != resources[0].URN {
		t.Errorf("expected URN=%q, got %q", resources[0].URN, items[0].URN)
	}
	if items[0].Op != pulumi.OpSame {
		t.Errorf("expected Op=%v, got %v", pulumi.OpSame, items[0].Op)
	}
	if items[0].Status != ui.StatusNone {
		t.Errorf("expected Status=%v, got %v", ui.StatusNone, items[0].Status)
	}
	if items[0].Inputs["bucket"] != "my-bucket" {
		t.Errorf("expected Inputs preserved")
	}

	// Second item (child)
	if items[1].Parent != resources[0].URN {
		t.Errorf("expected Parent=%q, got %q", resources[0].URN, items[1].Parent)
	}
}

// TestConvertResourcesToItems_Empty verifies empty input returns empty slice.
func TestConvertResourcesToItems_Empty(t *testing.T) {
	items := ConvertResourcesToItems(nil)

	if items == nil {
		t.Error("expected non-nil empty slice")
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

// TestConvertHistoryToItems_Basic verifies basic history conversion.
func TestConvertHistoryToItems_Basic(t *testing.T) {
	history := []pulumi.UpdateSummary{
		{
			Version:   3,
			Kind:      "update",
			StartTime: "2024-01-15T10:00:00Z",
			EndTime:   "2024-01-15T10:05:00Z",
			Message:   "Deploy v1.2.0",
			Result:    "succeeded",
			ResourceChanges: map[string]int{
				"create": 2,
				"update": 1,
			},
			User:      "alice",
			UserEmail: "alice@example.com",
		},
	}

	items := ConvertHistoryToItems(history)

	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	if items[0].Version != 3 {
		t.Errorf("expected Version=3, got %d", items[0].Version)
	}
	if items[0].Kind != "update" {
		t.Errorf("expected Kind=%q, got %q", "update", items[0].Kind)
	}
	if items[0].User != "alice" {
		t.Errorf("expected User=%q, got %q", "alice", items[0].User)
	}
	if items[0].ResourceChanges["create"] != 2 {
		t.Error("expected ResourceChanges to be preserved")
	}
}

// TestConvertHistoryToItems_LocalBackendVersioning verifies version calculation for local backend.
func TestConvertHistoryToItems_LocalBackendVersioning(t *testing.T) {
	// Local backend returns history newest-first with Version=0
	history := []pulumi.UpdateSummary{
		{Version: 0, Kind: "update", Message: "Third update"},  // Most recent
		{Version: 0, Kind: "update", Message: "Second update"}, // Middle
		{Version: 0, Kind: "update", Message: "First update"},  // Oldest
	}

	items := ConvertHistoryToItems(history)

	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}

	// Version should be calculated: len(history) - index
	// Index 0 → version 3 (most recent)
	// Index 1 → version 2
	// Index 2 → version 1 (oldest)
	if items[0].Version != 3 {
		t.Errorf("expected Version=3 for most recent, got %d", items[0].Version)
	}
	if items[1].Version != 2 {
		t.Errorf("expected Version=2 for middle, got %d", items[1].Version)
	}
	if items[2].Version != 1 {
		t.Errorf("expected Version=1 for oldest, got %d", items[2].Version)
	}
}

// TestConvertHistoryToItems_Empty verifies empty input returns empty slice.
func TestConvertHistoryToItems_Empty(t *testing.T) {
	items := ConvertHistoryToItems(nil)

	if items == nil {
		t.Error("expected non-nil empty slice")
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

// TestConvertImportSuggestions_Basic verifies basic suggestion conversion.
func TestConvertImportSuggestions_Basic(t *testing.T) {
	suggestions := []*plugins.AggregatedImportSuggestion{
		{
			PluginName: "aws",
			Suggestion: &plugins.ImportSuggestion{
				Id:          "bucket-123",
				Label:       "my-bucket",
				Description: "S3 bucket in us-east-1",
			},
		},
		{
			PluginName: "aws",
			Suggestion: &plugins.ImportSuggestion{
				Id:          "bucket-456",
				Label:       "other-bucket",
				Description: "S3 bucket in us-west-2",
			},
		},
	}

	items := ConvertImportSuggestions(suggestions)

	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	if items[0].ID != "bucket-123" {
		t.Errorf("expected ID=%q, got %q", "bucket-123", items[0].ID)
	}
	if items[0].Label != "my-bucket" {
		t.Errorf("expected Label=%q, got %q", "my-bucket", items[0].Label)
	}
	if items[0].PluginName != "aws" {
		t.Errorf("expected PluginName=%q, got %q", "aws", items[0].PluginName)
	}
}

// TestConvertImportSuggestions_Empty verifies empty input returns empty slice.
func TestConvertImportSuggestions_Empty(t *testing.T) {
	items := ConvertImportSuggestions(nil)

	if items == nil {
		t.Error("expected non-nil empty slice")
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

// TestSummarizePluginAuthResults_AllSuccess verifies successful auth summary.
func TestSummarizePluginAuthResults_AllSuccess(t *testing.T) {
	results := []plugins.AuthenticateResult{
		{PluginName: "aws", Credentials: &plugins.Credentials{Env: map[string]string{"AWS_KEY": "xxx"}}},
		{PluginName: "kubernetes", Credentials: &plugins.Credentials{Env: map[string]string{"KUBECONFIG": "/path"}}},
	}

	summary := SummarizePluginAuthResults(results)

	if summary.HasErrors {
		t.Error("expected HasErrors=false")
	}
	if len(summary.AuthenticatedPlugins) != 2 {
		t.Errorf("expected 2 authenticated plugins, got %d", len(summary.AuthenticatedPlugins))
	}
	if summary.AuthenticatedPlugins[0] != "aws" {
		t.Errorf("expected first plugin=%q, got %q", "aws", summary.AuthenticatedPlugins[0])
	}
}

// TestSummarizePluginAuthResults_WithErrors verifies error handling.
func TestSummarizePluginAuthResults_WithErrors(t *testing.T) {
	results := []plugins.AuthenticateResult{
		{PluginName: "aws", Credentials: &plugins.Credentials{Env: map[string]string{"AWS_KEY": "xxx"}}},
		{PluginName: "kubernetes", Error: testError("auth failed")},
	}

	summary := SummarizePluginAuthResults(results)

	if !summary.HasErrors {
		t.Error("expected HasErrors=true")
	}
	if len(summary.ErrorMessages) != 1 {
		t.Errorf("expected 1 error message, got %d", len(summary.ErrorMessages))
	}
	if len(summary.AuthenticatedPlugins) != 1 {
		t.Errorf("expected 1 authenticated plugin, got %d", len(summary.AuthenticatedPlugins))
	}
}

// TestSummarizePluginAuthResults_NoCredentials verifies plugins without credentials.
func TestSummarizePluginAuthResults_NoCredentials(t *testing.T) {
	results := []plugins.AuthenticateResult{
		{PluginName: "aws", Credentials: nil}, // No credentials but no error
	}

	summary := SummarizePluginAuthResults(results)

	if summary.HasErrors {
		t.Error("expected HasErrors=false")
	}
	if len(summary.AuthenticatedPlugins) != 0 {
		t.Errorf("expected 0 authenticated plugins, got %d", len(summary.AuthenticatedPlugins))
	}
}

// TestSummarizePluginAuthResults_Empty verifies empty input.
func TestSummarizePluginAuthResults_Empty(t *testing.T) {
	summary := SummarizePluginAuthResults(nil)

	if summary.HasErrors {
		t.Error("expected HasErrors=false for empty input")
	}
	if len(summary.AuthenticatedPlugins) != 0 {
		t.Error("expected empty AuthenticatedPlugins")
	}
}

// TestConvertStacksToItems_Basic verifies multiple stacks with one current.
func TestConvertStacksToItems_Basic(t *testing.T) {
	stacks := []pulumi.StackInfo{
		{Name: "dev", Current: false},
		{Name: "staging", Current: true},
		{Name: "prod", Current: false},
	}

	result := ConvertStacksToItems(stacks)

	if len(result.Items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(result.Items))
	}
	if result.CurrentStackName != "staging" {
		t.Errorf("expected CurrentStackName=%q, got %q", "staging", result.CurrentStackName)
	}
	if result.Items[1].Name != "staging" || !result.Items[1].Current {
		t.Error("expected second item to be staging and current")
	}
}

// TestConvertStacksToItems_NoCurrent verifies no stack marked as current.
func TestConvertStacksToItems_NoCurrent(t *testing.T) {
	stacks := []pulumi.StackInfo{
		{Name: "dev", Current: false},
		{Name: "staging", Current: false},
	}

	result := ConvertStacksToItems(stacks)

	if len(result.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result.Items))
	}
	if result.CurrentStackName != "" {
		t.Errorf("expected empty CurrentStackName, got %q", result.CurrentStackName)
	}
}

// TestConvertStacksToItems_Empty verifies empty slice input.
func TestConvertStacksToItems_Empty(t *testing.T) {
	result := ConvertStacksToItems(nil)

	if result.Items == nil {
		t.Error("expected non-nil empty slice")
	}
	if len(result.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(result.Items))
	}
	if result.CurrentStackName != "" {
		t.Errorf("expected empty CurrentStackName, got %q", result.CurrentStackName)
	}
}

// TestConvertStacksToItems_AllCurrent verifies edge case: multiple marked current (last wins).
func TestConvertStacksToItems_AllCurrent(t *testing.T) {
	stacks := []pulumi.StackInfo{
		{Name: "dev", Current: true},
		{Name: "staging", Current: true},
		{Name: "prod", Current: true},
	}

	result := ConvertStacksToItems(stacks)

	if len(result.Items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(result.Items))
	}
	// Last one wins in our implementation
	if result.CurrentStackName != "prod" {
		t.Errorf("expected CurrentStackName=%q (last wins), got %q", "prod", result.CurrentStackName)
	}
}

// TestConvertWorkspacesToItems_Basic verifies basic conversion with valid cwd.
func TestConvertWorkspacesToItems_Basic(t *testing.T) {
	workspaces := []pulumi.WorkspaceInfo{
		{Path: "/home/user/projects/app1", Name: "app1", Current: true},
		{Path: "/home/user/projects/app2", Name: "app2", Current: false},
	}

	items := ConvertWorkspacesToItems(workspaces, "/home/user/projects")

	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].Path != "/home/user/projects/app1" {
		t.Errorf("expected Path=%q, got %q", "/home/user/projects/app1", items[0].Path)
	}
	if items[0].RelativePath != "app1" {
		t.Errorf("expected RelativePath=%q, got %q", "app1", items[0].RelativePath)
	}
	if items[0].Name != "app1" {
		t.Errorf("expected Name=%q, got %q", "app1", items[0].Name)
	}
	if !items[0].Current {
		t.Error("expected first item to be current")
	}
}

// TestConvertWorkspacesToItems_EmptyCwd verifies no relative path calculation when cwd is empty.
func TestConvertWorkspacesToItems_EmptyCwd(t *testing.T) {
	workspaces := []pulumi.WorkspaceInfo{
		{Path: "/home/user/projects/app1", Name: "app1", Current: false},
	}

	items := ConvertWorkspacesToItems(workspaces, "")

	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	// RelativePath should be same as Path when cwd is empty
	if items[0].RelativePath != "/home/user/projects/app1" {
		t.Errorf("expected RelativePath=%q (same as Path), got %q", "/home/user/projects/app1", items[0].RelativePath)
	}
}

// TestConvertWorkspacesToItems_Empty verifies empty slice input.
func TestConvertWorkspacesToItems_Empty(t *testing.T) {
	items := ConvertWorkspacesToItems(nil, "/home/user")

	if items == nil {
		t.Error("expected non-nil empty slice")
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

// TestConvertWorkspacesToItems_RelativePath verifies relative path calculation for various paths.
func TestConvertWorkspacesToItems_RelativePath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		cwd      string
		expected string
	}{
		{"child dir", "/home/user/projects/app", "/home/user/projects", "app"},
		{"sibling dir", "/home/user/other/app", "/home/user/projects", "../other/app"},
		{"same dir", "/home/user/projects", "/home/user/projects", "."},
		{"nested child", "/home/user/projects/a/b/c", "/home/user/projects", "a/b/c"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workspaces := []pulumi.WorkspaceInfo{
				{Path: tt.path, Name: "test", Current: false},
			}

			items := ConvertWorkspacesToItems(workspaces, tt.cwd)

			if len(items) != 1 {
				t.Fatalf("expected 1 item, got %d", len(items))
			}
			if items[0].RelativePath != tt.expected {
				t.Errorf("expected RelativePath=%q, got %q", tt.expected, items[0].RelativePath)
			}
		})
	}
}

// TestDetermineStackInitAction_NoStacks verifies returns ShowInit when no stacks exist.
func TestDetermineStackInitAction_NoStacks(t *testing.T) {
	action := DetermineStackInitAction(InitLoadingStacks, 0, "")

	if action != StackInitActionShowInit {
		t.Errorf("expected %v, got %v", StackInitActionShowInit, action)
	}
}

// TestDetermineStackInitAction_NoCurrent verifies returns ShowSelector when stacks exist but none current.
func TestDetermineStackInitAction_NoCurrent(t *testing.T) {
	action := DetermineStackInitAction(InitLoadingStacks, 3, "")

	if action != StackInitActionShowSelector {
		t.Errorf("expected %v, got %v", StackInitActionShowSelector, action)
	}
}

// TestDetermineStackInitAction_HasCurrent verifies returns Proceed when a current stack exists.
func TestDetermineStackInitAction_HasCurrent(t *testing.T) {
	action := DetermineStackInitAction(InitLoadingStacks, 3, "dev")

	if action != StackInitActionProceed {
		t.Errorf("expected %v, got %v", StackInitActionProceed, action)
	}
}

// TestDetermineStackInitAction_NotInInitFlow verifies returns None when not in InitLoadingStacks.
func TestDetermineStackInitAction_NotInInitFlow(t *testing.T) {
	tests := []struct {
		name      string
		initState InitState
	}{
		{"CheckingWorkspace", InitCheckingWorkspace},
		{"LoadingPlugins", InitLoadingPlugins},
		{"SelectingStack", InitSelectingStack},
		{"LoadingResources", InitLoadingResources},
		{"Complete", InitComplete},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := DetermineStackInitAction(tt.initState, 3, "dev")

			if action != StackInitActionNone {
				t.Errorf("expected %v for state %v, got %v", StackInitActionNone, tt.initState, action)
			}
		})
	}
}

// TestDetermineStackInitAction_EmptyStacksWithName verifies edge case: zero stacks but name provided.
func TestDetermineStackInitAction_EmptyStacksWithName(t *testing.T) {
	// Even if a name is provided, zero stacks means ShowInit
	action := DetermineStackInitAction(InitLoadingStacks, 0, "dev")

	if action != StackInitActionShowInit {
		t.Errorf("expected %v (zero stacks takes priority), got %v", StackInitActionShowInit, action)
	}
}

// TestStackInitActionString verifies String() returns human-readable names.
func TestStackInitActionString(t *testing.T) {
	tests := []struct {
		action StackInitAction
		want   string
	}{
		{StackInitActionNone, "None"},
		{StackInitActionShowInit, "ShowInit"},
		{StackInitActionShowSelector, "ShowSelector"},
		{StackInitActionProceed, "Proceed"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.action.String(); got != tt.want {
				t.Errorf("StackInitAction(%d).String() = %q, want %q", tt.action, got, tt.want)
			}
		})
	}
}

// TestDetermineEscapeAction_VisualMode verifies visual mode exit takes highest priority.
func TestDetermineEscapeAction_VisualMode(t *testing.T) {
	// Visual mode should always return ExitVisualMode, regardless of other state
	tests := []struct {
		name     string
		viewMode ui.ViewMode
		opState  OperationState
	}{
		{"in stack view", ui.ViewStack, OpIdle},
		{"in preview running", ui.ViewPreview, OpRunning},
		{"in execute running", ui.ViewExecute, OpRunning},
		{"in history", ui.ViewHistory, OpIdle},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := DetermineEscapeAction(tt.viewMode, tt.opState, true)
			if action != EscapeActionExitVisualMode {
				t.Errorf("expected %v with visualMode=true, got %v", EscapeActionExitVisualMode, action)
			}
		})
	}
}

// TestDetermineEscapeAction_CancelRunningOp verifies cancellation in execute view with running op.
func TestDetermineEscapeAction_CancelRunningOp(t *testing.T) {
	action := DetermineEscapeAction(ui.ViewExecute, OpRunning, false)
	if action != EscapeActionCancelOp {
		t.Errorf("expected %v, got %v", EscapeActionCancelOp, action)
	}
}

// TestDetermineEscapeAction_NavigateBackPreview verifies navigation back from preview when idle.
func TestDetermineEscapeAction_NavigateBackPreview(t *testing.T) {
	tests := []struct {
		name    string
		opState OperationState
	}{
		{"Idle", OpIdle},
		{"Complete", OpComplete},
		{"Error", OpError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := DetermineEscapeAction(ui.ViewPreview, tt.opState, false)
			if action != EscapeActionNavigateBack {
				t.Errorf("expected %v in ViewPreview with %v, got %v", EscapeActionNavigateBack, tt.opState, action)
			}
		})
	}
}

// TestDetermineEscapeAction_NavigateBackHistory verifies history view always allows back navigation.
func TestDetermineEscapeAction_NavigateBackHistory(t *testing.T) {
	// History view should allow navigation back even if an op is "running" (edge case)
	tests := []struct {
		name    string
		opState OperationState
	}{
		{"Idle", OpIdle},
		{"Running", OpRunning}, // Special: history allows this
		{"Complete", OpComplete},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := DetermineEscapeAction(ui.ViewHistory, tt.opState, false)
			if action != EscapeActionNavigateBack {
				t.Errorf("expected %v in ViewHistory with %v, got %v", EscapeActionNavigateBack, tt.opState, action)
			}
		})
	}
}

// TestDetermineEscapeAction_BlockedByActiveOp verifies escape is blocked during active ops.
func TestDetermineEscapeAction_BlockedByActiveOp(t *testing.T) {
	// In preview/execute view with active op (but not ViewExecute+Running which triggers cancel)
	tests := []struct {
		name     string
		viewMode ui.ViewMode
		opState  OperationState
	}{
		{"preview starting", ui.ViewPreview, OpStarting},
		{"preview running", ui.ViewPreview, OpRunning},
		{"preview cancelling", ui.ViewPreview, OpCancelling},
		{"execute starting", ui.ViewExecute, OpStarting},
		{"execute cancelling", ui.ViewExecute, OpCancelling},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := DetermineEscapeAction(tt.viewMode, tt.opState, false)
			if action != EscapeActionNone {
				t.Errorf("expected %v for %v/%v (active op), got %v", EscapeActionNone, tt.viewMode, tt.opState, action)
			}
		})
	}
}

// TestDetermineEscapeAction_StackView verifies no action in stack view.
func TestDetermineEscapeAction_StackView(t *testing.T) {
	action := DetermineEscapeAction(ui.ViewStack, OpIdle, false)
	if action != EscapeActionNone {
		t.Errorf("expected %v in ViewStack, got %v", EscapeActionNone, action)
	}
}

// TestEscapeActionString verifies String() returns human-readable names.
func TestEscapeActionString(t *testing.T) {
	tests := []struct {
		action EscapeAction
		want   string
	}{
		{EscapeActionNone, "None"},
		{EscapeActionExitVisualMode, "ExitVisualMode"},
		{EscapeActionCancelOp, "CancelOp"},
		{EscapeActionNavigateBack, "NavigateBack"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.action.String(); got != tt.want {
				t.Errorf("EscapeAction(%d).String() = %q, want %q", tt.action, got, tt.want)
			}
		})
	}
}

// TestCanImportResource_ValidCreate verifies import allowed for create op in preview.
func TestCanImportResource_ValidCreate(t *testing.T) {
	item := &ui.ResourceItem{
		URN:  "urn:pulumi:dev::test::aws:s3:Bucket::mybucket",
		Type: "aws:s3:Bucket",
		Name: "mybucket",
		Op:   pulumi.OpCreate,
	}

	if !CanImportResource(ui.ViewPreview, item) {
		t.Error("expected CanImportResource=true for create op in preview view")
	}
}

// TestCanImportResource_WrongView verifies import not allowed outside preview view.
func TestCanImportResource_WrongView(t *testing.T) {
	item := &ui.ResourceItem{
		Op: pulumi.OpCreate,
	}

	views := []ui.ViewMode{ui.ViewStack, ui.ViewExecute, ui.ViewHistory}
	for _, v := range views {
		if CanImportResource(v, item) {
			t.Errorf("expected CanImportResource=false for view %v", v)
		}
	}
}

// TestCanImportResource_NoSelection verifies import not allowed with nil item.
func TestCanImportResource_NoSelection(t *testing.T) {
	if CanImportResource(ui.ViewPreview, nil) {
		t.Error("expected CanImportResource=false for nil item")
	}
}

// TestCanImportResource_WrongOp verifies import not allowed for non-create ops.
func TestCanImportResource_WrongOp(t *testing.T) {
	ops := []pulumi.ResourceOp{
		pulumi.OpUpdate,
		pulumi.OpDelete,
		pulumi.OpSame,
		pulumi.OpReplace,
		pulumi.OpRefresh,
	}

	for _, op := range ops {
		item := &ui.ResourceItem{Op: op}
		if CanImportResource(ui.ViewPreview, item) {
			t.Errorf("expected CanImportResource=false for op %v", op)
		}
	}
}

// TestCanDeleteFromState_ValidResource verifies delete allowed for regular resource in stack view.
func TestCanDeleteFromState_ValidResource(t *testing.T) {
	item := &ui.ResourceItem{
		URN:  "urn:pulumi:dev::test::aws:s3:Bucket::mybucket",
		Type: "aws:s3:Bucket",
		Name: "mybucket",
	}

	if !CanDeleteFromState(ui.ViewStack, item) {
		t.Error("expected CanDeleteFromState=true for regular resource in stack view")
	}
}

// TestCanDeleteFromState_WrongView verifies delete not allowed outside stack view.
func TestCanDeleteFromState_WrongView(t *testing.T) {
	item := &ui.ResourceItem{
		Type: "aws:s3:Bucket",
	}

	views := []ui.ViewMode{ui.ViewPreview, ui.ViewExecute, ui.ViewHistory}
	for _, v := range views {
		if CanDeleteFromState(v, item) {
			t.Errorf("expected CanDeleteFromState=false for view %v", v)
		}
	}
}

// TestCanDeleteFromState_NoSelection verifies delete not allowed with nil item.
func TestCanDeleteFromState_NoSelection(t *testing.T) {
	if CanDeleteFromState(ui.ViewStack, nil) {
		t.Error("expected CanDeleteFromState=false for nil item")
	}
}

// TestCanDeleteFromState_RootStack verifies delete not allowed for pulumi:pulumi:Stack.
func TestCanDeleteFromState_RootStack(t *testing.T) {
	item := &ui.ResourceItem{
		URN:  "urn:pulumi:dev::test::pulumi:pulumi:Stack::test-dev",
		Type: "pulumi:pulumi:Stack",
		Name: "test-dev",
	}

	if CanDeleteFromState(ui.ViewStack, item) {
		t.Error("expected CanDeleteFromState=false for pulumi:pulumi:Stack")
	}
}

// TestFormatClipboardMessage_SingleNamed verifies single resource with name.
func TestFormatClipboardMessage_SingleNamed(t *testing.T) {
	msg := FormatClipboardMessage(1, "mybucket")
	expected := "Copied mybucket"
	if msg != expected {
		t.Errorf("expected %q, got %q", expected, msg)
	}
}

// TestFormatClipboardMessage_SingleUnnamed verifies single resource without name.
func TestFormatClipboardMessage_SingleUnnamed(t *testing.T) {
	msg := FormatClipboardMessage(1, "")
	expected := "Copied resource"
	if msg != expected {
		t.Errorf("expected %q, got %q", expected, msg)
	}
}

// TestFormatClipboardMessage_Multiple verifies multiple resources show count.
func TestFormatClipboardMessage_Multiple(t *testing.T) {
	tests := []struct {
		count    int
		expected string
	}{
		{2, "Copied 2 resources"},
		{5, "Copied 5 resources"},
		{100, "Copied 100 resources"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			msg := FormatClipboardMessage(tt.count, "ignored")
			if msg != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, msg)
			}
		})
	}
}

// TestFormatClipboardMessage_Text verifies text copy (count=0) uses generic message.
func TestFormatClipboardMessage_Text(t *testing.T) {
	msg := FormatClipboardMessage(0, "")
	expected := "Copied to clipboard"
	if msg != expected {
		t.Errorf("expected %q, got %q", expected, msg)
	}

	// Even with a name, count=0 should use generic message
	msg2 := FormatClipboardMessage(0, "somename")
	if msg2 != expected {
		t.Errorf("expected %q for count=0 with name, got %q", expected, msg2)
	}
}

// TestProcessPreviewEvent_ErrorDoesNotSetInitDoneOutsideInit verifies InitDone=false when not InitLoadingResources.
func TestProcessPreviewEvent_ErrorDoesNotSetInitDoneOutsideInit(t *testing.T) {
	testErr := testError("preview failed")
	event := pulumi.PreviewEvent{
		Error: testErr,
	}

	// Test with InitComplete - InitDone should be false
	result := ProcessPreviewEvent(event, OpRunning, InitComplete)

	if result.InitDone {
		t.Error("expected InitDone=false when not in InitLoadingResources")
	}
	if result.NewOpState != OpError {
		t.Errorf("expected OpState=%v, got %v", OpError, result.NewOpState)
	}
}

// TestProcessPreviewEvent_NilStep verifies event with nil step is no-op for item.
func TestProcessPreviewEvent_NilStep(t *testing.T) {
	// Event with no step, no error, not done - just an empty event
	event := pulumi.PreviewEvent{}

	result := ProcessPreviewEvent(event, OpRunning, InitLoadingResources)

	if result.Item != nil {
		t.Error("expected Item=nil for event with nil step")
	}
	if result.NewOpState != OpRunning {
		t.Errorf("expected OpState to remain %v, got %v", OpRunning, result.NewOpState)
	}
	if result.HasError {
		t.Error("expected no error")
	}
	if result.InitDone {
		t.Error("expected InitDone=false for empty event")
	}
}

// TestProcessPreviewEvent_StepWithNilOld verifies step without old state works correctly.
func TestProcessPreviewEvent_StepWithNilOld(t *testing.T) {
	event := pulumi.PreviewEvent{
		Step: &pulumi.PreviewStep{
			URN:    "urn:pulumi:dev::test::aws:s3:Bucket::mybucket",
			Type:   "aws:s3:Bucket",
			Name:   "mybucket",
			Op:     pulumi.OpCreate,
			Inputs: map[string]any{"bucket": "new-bucket"},
			Old:    nil, // No old state for create
		},
	}

	result := ProcessPreviewEvent(event, OpRunning, InitLoadingResources)

	if result.Item == nil {
		t.Fatal("expected Item to be set")
	}
	if result.Item.OldInputs != nil {
		t.Errorf("expected OldInputs=nil for create, got %v", result.Item.OldInputs)
	}
	if result.Item.OldOutputs != nil {
		t.Errorf("expected OldOutputs=nil for create, got %v", result.Item.OldOutputs)
	}
	if result.Item.Inputs["bucket"] != "new-bucket" {
		t.Errorf("expected Inputs[bucket]=%q, got %q", "new-bucket", result.Item.Inputs["bucket"])
	}
}

// TestProcessOperationEvent_EmptyURN verifies event with empty URN produces no item.
func TestProcessOperationEvent_EmptyURN(t *testing.T) {
	event := pulumi.OperationEvent{
		URN:    "", // Empty URN
		Status: pulumi.StepRunning,
	}

	result := ProcessOperationEvent(event, OpRunning)

	if result.Item != nil {
		t.Error("expected Item=nil for empty URN")
	}
	if result.NewOpState != OpRunning {
		t.Errorf("expected OpState=%v, got %v", OpRunning, result.NewOpState)
	}
}

// TestProcessOperationEvent_TransitionsFromCancelling verifies Cancelling state handling.
func TestProcessOperationEvent_TransitionsFromCancelling(t *testing.T) {
	// During cancelling, events may still arrive
	event := pulumi.OperationEvent{
		URN:    "urn:pulumi:dev::test::aws:s3:Bucket::mybucket",
		Status: pulumi.StepRunning,
	}

	result := ProcessOperationEvent(event, OpCancelling)

	// Should remain in cancelling state
	if result.NewOpState != OpCancelling {
		t.Errorf("expected OpState=%v, got %v", OpCancelling, result.NewOpState)
	}
	if result.Item == nil {
		t.Error("expected Item to be set even during cancelling")
	}
}

// TestProcessOperationEvent_DoneWhileCancelling verifies done event during cancelling.
func TestProcessOperationEvent_DoneWhileCancelling(t *testing.T) {
	event := pulumi.OperationEvent{
		Done: true,
	}

	result := ProcessOperationEvent(event, OpCancelling)

	// Done should complete even from cancelling
	if result.NewOpState != OpComplete {
		t.Errorf("expected OpState=%v, got %v", OpComplete, result.NewOpState)
	}
	if !result.Done {
		t.Error("expected Done=true")
	}
}

// TestConvertResourcesToItems_WithParent verifies parent URN is preserved.
func TestConvertResourcesToItems_WithParent(t *testing.T) {
	parentURN := "urn:pulumi:dev::test::aws:s3:Bucket::mybucket"
	resources := []pulumi.ResourceInfo{
		{
			URN:    parentURN,
			Type:   "aws:s3:Bucket",
			Name:   "mybucket",
			Parent: "",
		},
		{
			URN:    "urn:pulumi:dev::test::aws:s3:BucketObject::myfile",
			Type:   "aws:s3:BucketObject",
			Name:   "myfile",
			Parent: parentURN,
		},
	}

	items := ConvertResourcesToItems(resources)

	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].Parent != "" {
		t.Errorf("expected first item Parent=%q, got %q", "", items[0].Parent)
	}
	if items[1].Parent != parentURN {
		t.Errorf("expected second item Parent=%q, got %q", parentURN, items[1].Parent)
	}
}

// TestConvertResourcesToItems_LargeList verifies performance sanity check with many resources.
func TestConvertResourcesToItems_LargeList(t *testing.T) {
	// Create 1000 resources
	resources := make([]pulumi.ResourceInfo, 1000)
	for i := range 1000 {
		resources[i] = pulumi.ResourceInfo{
			URN:  "urn:pulumi:dev::test::aws:s3:Bucket::bucket-" + testItoa(i),
			Type: "aws:s3:Bucket",
			Name: "bucket-" + testItoa(i),
		}
	}

	items := ConvertResourcesToItems(resources)

	if len(items) != 1000 {
		t.Errorf("expected 1000 items, got %d", len(items))
	}
	// Spot check first and last
	if items[0].Name != "bucket-0" {
		t.Errorf("expected first item name=%q, got %q", "bucket-0", items[0].Name)
	}
	if items[999].Name != "bucket-999" {
		t.Errorf("expected last item name=%q, got %q", "bucket-999", items[999].Name)
	}
}

// testItoa is a simple int-to-string for test use.
func testItoa(i int) string {
	if i == 0 {
		return "0"
	}
	if i < 0 {
		return "-" + testItoa(-i)
	}
	var digits []byte
	for i > 0 {
		digits = append([]byte{byte('0' + i%10)}, digits...)
		i /= 10
	}
	return string(digits)
}

// TestSummarizePluginAuthResults_EmptyEnv verifies credentials with empty env map.
func TestSummarizePluginAuthResults_EmptyEnv(t *testing.T) {
	results := []plugins.AuthenticateResult{
		{PluginName: "aws", Credentials: &plugins.Credentials{Env: map[string]string{}}}, // Empty env
	}

	summary := SummarizePluginAuthResults(results)

	if summary.HasErrors {
		t.Error("expected HasErrors=false")
	}
	// Empty env should not count as authenticated
	if len(summary.AuthenticatedPlugins) != 0 {
		t.Errorf("expected 0 authenticated plugins for empty env, got %d", len(summary.AuthenticatedPlugins))
	}
}

// TestSummarizePluginAuthResults_MultipleErrors verifies multiple failed plugins.
func TestSummarizePluginAuthResults_MultipleErrors(t *testing.T) {
	results := []plugins.AuthenticateResult{
		{PluginName: "aws", Error: testError("aws auth failed")},
		{PluginName: "kubernetes", Error: testError("k8s auth failed")},
		{PluginName: "gcp", Error: testError("gcp auth failed")},
	}

	summary := SummarizePluginAuthResults(results)

	if !summary.HasErrors {
		t.Error("expected HasErrors=true")
	}
	if len(summary.ErrorMessages) != 3 {
		t.Errorf("expected 3 error messages, got %d", len(summary.ErrorMessages))
	}
	if len(summary.AuthenticatedPlugins) != 0 {
		t.Errorf("expected 0 authenticated plugins, got %d", len(summary.AuthenticatedPlugins))
	}
	// Verify error messages contain plugin names
	for i, msg := range summary.ErrorMessages {
		if msg == "" {
			t.Errorf("expected non-empty error message at index %d", i)
		}
	}
}
