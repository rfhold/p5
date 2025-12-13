package ui

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/golden"
	"github.com/rfhold/p5/internal/pulumi"
)

// Test dimensions for consistent golden file output
const (
	testWidth  = 80
	testHeight = 24
)

func TestHeader_Loading(t *testing.T) {
	h := NewHeader()
	h.SetWidth(testWidth)
	// Loading state is the default

	golden.RequireEqual(t, []byte(h.View()))
}

func TestHeader_WithData(t *testing.T) {
	h := NewHeader()
	h.SetWidth(testWidth)
	h.SetData(&HeaderData{
		ProgramName: "my-app",
		StackName:   "dev",
		Runtime:     "go",
	})

	golden.RequireEqual(t, []byte(h.View()))
}

func TestHeader_WithError(t *testing.T) {
	h := NewHeader()
	h.SetWidth(testWidth)
	h.SetError(errTest)

	golden.RequireEqual(t, []byte(h.View()))
}

func TestHeader_StackView(t *testing.T) {
	h := NewHeader()
	h.SetWidth(testWidth)
	h.SetData(&HeaderData{
		ProgramName: "my-app",
		StackName:   "dev",
		Runtime:     "go",
	})
	h.SetViewMode(ViewStack)
	h.SetSummary(ResourceSummary{
		Total: 10,
		Same:  10,
	}, HeaderDone)

	golden.RequireEqual(t, []byte(h.View()))
}

func TestHeader_PreviewRunning(t *testing.T) {
	h := NewHeader()
	h.SetWidth(testWidth)
	h.SetData(&HeaderData{
		ProgramName: "my-app",
		StackName:   "dev",
		Runtime:     "go",
	})
	h.SetViewMode(ViewPreview)
	h.SetOperation(OperationUp)
	h.SetSummary(ResourceSummary{
		Total:  5,
		Create: 2,
		Update: 1,
		Delete: 1,
	}, HeaderRunning)

	golden.RequireEqual(t, []byte(h.View()))
}

func TestHeader_PreviewDone(t *testing.T) {
	h := NewHeader()
	h.SetWidth(testWidth)
	h.SetData(&HeaderData{
		ProgramName: "my-app",
		StackName:   "dev",
		Runtime:     "go",
	})
	h.SetViewMode(ViewPreview)
	h.SetOperation(OperationUp)
	h.SetSummary(ResourceSummary{
		Total:   5,
		Create:  2,
		Update:  1,
		Delete:  1,
		Replace: 1,
	}, HeaderDone)

	golden.RequireEqual(t, []byte(h.View()))
}

func TestHeader_PreviewNoChanges(t *testing.T) {
	h := NewHeader()
	h.SetWidth(testWidth)
	h.SetData(&HeaderData{
		ProgramName: "my-app",
		StackName:   "dev",
		Runtime:     "go",
	})
	h.SetViewMode(ViewPreview)
	h.SetOperation(OperationRefresh)
	h.SetSummary(ResourceSummary{
		Total: 0,
	}, HeaderDone)

	golden.RequireEqual(t, []byte(h.View()))
}

func TestHeader_HistoryView(t *testing.T) {
	h := NewHeader()
	h.SetWidth(testWidth)
	h.SetData(&HeaderData{
		ProgramName: "my-app",
		StackName:   "dev",
		Runtime:     "go",
	})
	h.SetViewMode(ViewHistory)
	h.SetSummary(ResourceSummary{
		Total: 15,
	}, HeaderDone)

	golden.RequireEqual(t, []byte(h.View()))
}

func TestResourceList_Empty(t *testing.T) {
	r := NewResourceList(make(map[string]ResourceFlags))
	r.SetSize(testWidth, testHeight)
	r.SetItems([]ResourceItem{})

	golden.RequireEqual(t, []byte(r.View()))
}

func TestResourceList_Loading(t *testing.T) {
	r := NewResourceList(make(map[string]ResourceFlags))
	r.SetSize(testWidth, testHeight)
	r.SetLoading(true, "Loading resources...")

	golden.RequireEqual(t, []byte(r.View()))
}

func TestResourceList_SingleItem(t *testing.T) {
	flags := make(map[string]ResourceFlags)
	r := NewResourceList(flags)
	r.SetSize(testWidth, testHeight)
	r.SetItems([]ResourceItem{
		{
			URN:  "urn:pulumi:dev::my-app::aws:s3/bucket:Bucket::my-bucket",
			Type: "aws:s3/bucket:Bucket",
			Name: "my-bucket",
			Op:   OpCreate,
		},
	})

	golden.RequireEqual(t, []byte(r.View()))
}

func TestResourceList_MultipleOps(t *testing.T) {
	flags := make(map[string]ResourceFlags)
	r := NewResourceList(flags)
	r.SetSize(testWidth, testHeight)
	r.SetItems([]ResourceItem{
		{
			URN:  "urn:pulumi:dev::my-app::aws:s3/bucket:Bucket::bucket-1",
			Type: "aws:s3/bucket:Bucket",
			Name: "bucket-1",
			Op:   OpCreate,
		},
		{
			URN:  "urn:pulumi:dev::my-app::aws:s3/bucket:Bucket::bucket-2",
			Type: "aws:s3/bucket:Bucket",
			Name: "bucket-2",
			Op:   OpUpdate,
		},
		{
			URN:  "urn:pulumi:dev::my-app::aws:s3/bucket:Bucket::bucket-3",
			Type: "aws:s3/bucket:Bucket",
			Name: "bucket-3",
			Op:   OpDelete,
		},
		{
			URN:  "urn:pulumi:dev::my-app::aws:s3/bucket:Bucket::bucket-4",
			Type: "aws:s3/bucket:Bucket",
			Name: "bucket-4",
			Op:   OpReplace,
		},
		{
			URN:  "urn:pulumi:dev::my-app::aws:s3/bucket:Bucket::bucket-5",
			Type: "aws:s3/bucket:Bucket",
			Name: "bucket-5",
			Op:   OpSame,
		},
	})

	golden.RequireEqual(t, []byte(r.View()))
}

func TestResourceList_WithFlags(t *testing.T) {
	flags := make(map[string]ResourceFlags)
	flags["urn:pulumi:dev::my-app::aws:s3/bucket:Bucket::bucket-1"] = ResourceFlags{Target: true}
	flags["urn:pulumi:dev::my-app::aws:s3/bucket:Bucket::bucket-2"] = ResourceFlags{Replace: true}
	flags["urn:pulumi:dev::my-app::aws:s3/bucket:Bucket::bucket-3"] = ResourceFlags{Exclude: true}
	flags["urn:pulumi:dev::my-app::aws:s3/bucket:Bucket::bucket-4"] = ResourceFlags{Target: true, Replace: true}

	r := NewResourceList(flags)
	r.SetSize(testWidth, testHeight)
	r.SetItems([]ResourceItem{
		{
			URN:  "urn:pulumi:dev::my-app::aws:s3/bucket:Bucket::bucket-1",
			Type: "aws:s3/bucket:Bucket",
			Name: "bucket-1",
			Op:   OpCreate,
		},
		{
			URN:  "urn:pulumi:dev::my-app::aws:s3/bucket:Bucket::bucket-2",
			Type: "aws:s3/bucket:Bucket",
			Name: "bucket-2",
			Op:   OpUpdate,
		},
		{
			URN:  "urn:pulumi:dev::my-app::aws:s3/bucket:Bucket::bucket-3",
			Type: "aws:s3/bucket:Bucket",
			Name: "bucket-3",
			Op:   OpSame,
		},
		{
			URN:  "urn:pulumi:dev::my-app::aws:s3/bucket:Bucket::bucket-4",
			Type: "aws:s3/bucket:Bucket",
			Name: "bucket-4",
			Op:   OpReplace,
		},
	})

	golden.RequireEqual(t, []byte(r.View()))
}

func TestResourceList_WithStatus(t *testing.T) {
	flags := make(map[string]ResourceFlags)
	r := NewResourceList(flags)
	r.SetSize(testWidth, testHeight)
	r.SetItems([]ResourceItem{
		{
			URN:    "urn:pulumi:dev::my-app::aws:s3/bucket:Bucket::bucket-1",
			Type:   "aws:s3/bucket:Bucket",
			Name:   "bucket-1",
			Op:     OpCreate,
			Status: StatusSuccess,
		},
		{
			URN:       "urn:pulumi:dev::my-app::aws:s3/bucket:Bucket::bucket-2",
			Type:      "aws:s3/bucket:Bucket",
			Name:      "bucket-2",
			Op:        OpUpdate,
			Status:    StatusRunning,
			CurrentOp: OpUpdate,
		},
		{
			URN:    "urn:pulumi:dev::my-app::aws:s3/bucket:Bucket::bucket-3",
			Type:   "aws:s3/bucket:Bucket",
			Name:   "bucket-3",
			Op:     OpDelete,
			Status: StatusPending,
		},
		{
			URN:    "urn:pulumi:dev::my-app::aws:s3/bucket:Bucket::bucket-4",
			Type:   "aws:s3/bucket:Bucket",
			Name:   "bucket-4",
			Op:     OpCreate,
			Status: StatusFailed,
		},
	})

	golden.RequireEqual(t, []byte(r.View()))
}

func TestResourceList_TreeStructure(t *testing.T) {
	flags := make(map[string]ResourceFlags)
	r := NewResourceList(flags)
	r.SetSize(testWidth, testHeight)
	r.SetItems([]ResourceItem{
		{
			URN:  "urn:pulumi:dev::my-app::pulumi:pulumi:Stack::my-stack",
			Type: "pulumi:pulumi:Stack",
			Name: "my-stack",
			Op:   OpSame,
		},
		{
			URN:    "urn:pulumi:dev::my-app::my:component:Component::parent",
			Type:   "my:component:Component",
			Name:   "parent",
			Op:     OpSame,
			Parent: "urn:pulumi:dev::my-app::pulumi:pulumi:Stack::my-stack",
		},
		{
			URN:    "urn:pulumi:dev::my-app::aws:s3/bucket:Bucket::child-1",
			Type:   "aws:s3/bucket:Bucket",
			Name:   "child-1",
			Op:     OpCreate,
			Parent: "urn:pulumi:dev::my-app::my:component:Component::parent",
		},
		{
			URN:    "urn:pulumi:dev::my-app::aws:s3/bucket:Bucket::child-2",
			Type:   "aws:s3/bucket:Bucket",
			Name:   "child-2",
			Op:     OpUpdate,
			Parent: "urn:pulumi:dev::my-app::my:component:Component::parent",
		},
	})

	golden.RequireEqual(t, []byte(r.View()))
}

func TestHelpDialog_View(t *testing.T) {
	h := NewHelpDialog()
	h.SetSize(testWidth, testHeight)

	golden.RequireEqual(t, []byte(h.View()))
}

func TestToast_Hidden(t *testing.T) {
	toast := NewToast()
	golden.RequireEqual(t, []byte(toast.View(testWidth)))
}

func TestToast_Visible(t *testing.T) {
	toast := NewToast()
	toast.Show("Copied to clipboard!")
	golden.RequireEqual(t, []byte(toast.View(testWidth)))
}

func TestDiffRenderer_Create(t *testing.T) {
	r := NewDiffRenderer(testWidth)
	resource := &ResourceItem{
		Op: OpCreate,
		Inputs: map[string]any{
			"name":   "my-bucket",
			"region": "us-west-2",
			"tags": map[string]any{
				"env": "dev",
			},
		},
		Outputs: map[string]any{
			"id":  "bucket-12345",
			"arn": "arn:aws:s3:::my-bucket",
		},
	}

	golden.RequireEqual(t, []byte(r.RenderCombinedProperties(resource)))
}

func TestDiffRenderer_Delete(t *testing.T) {
	r := NewDiffRenderer(testWidth)
	resource := &ResourceItem{
		Op: OpDelete,
		OldInputs: map[string]any{
			"name":   "my-bucket",
			"region": "us-west-2",
		},
		OldOutputs: map[string]any{
			"id":  "bucket-12345",
			"arn": "arn:aws:s3:::my-bucket",
		},
	}

	golden.RequireEqual(t, []byte(r.RenderCombinedProperties(resource)))
}

func TestDiffRenderer_Update(t *testing.T) {
	r := NewDiffRenderer(testWidth)
	resource := &ResourceItem{
		Op: OpUpdate,
		OldInputs: map[string]any{
			"name":   "my-bucket",
			"region": "us-west-2",
			"tags": map[string]any{
				"env": "dev",
			},
		},
		Inputs: map[string]any{
			"name":   "my-bucket",
			"region": "us-west-2",
			"tags": map[string]any{
				"env": "prod",
			},
		},
		OldOutputs: map[string]any{
			"id": "bucket-12345",
		},
		Outputs: map[string]any{
			"id": "bucket-12345",
		},
	}

	golden.RequireEqual(t, []byte(r.RenderCombinedProperties(resource)))
}

func TestDiffRenderer_UpdateAddRemoveKeys(t *testing.T) {
	r := NewDiffRenderer(testWidth)
	resource := &ResourceItem{
		Op: OpUpdate,
		OldInputs: map[string]any{
			"name":      "my-bucket",
			"oldField":  "will-be-removed",
			"unchanged": "stays-same",
		},
		Inputs: map[string]any{
			"name":      "my-bucket-renamed",
			"newField":  "just-added",
			"unchanged": "stays-same",
		},
	}

	golden.RequireEqual(t, []byte(r.RenderCombinedProperties(resource)))
}

func TestDiffRenderer_ArrayDiff(t *testing.T) {
	r := NewDiffRenderer(testWidth)
	resource := &ResourceItem{
		Op: OpUpdate,
		OldInputs: map[string]any{
			"ports": []any{80, 443},
		},
		Inputs: map[string]any{
			"ports": []any{80, 443, 8080},
		},
	}

	golden.RequireEqual(t, []byte(r.RenderCombinedProperties(resource)))
}

func TestDiffRenderer_NoProperties(t *testing.T) {
	r := NewDiffRenderer(testWidth)
	resource := &ResourceItem{
		Op: OpSame,
	}

	golden.RequireEqual(t, []byte(r.RenderCombinedProperties(resource)))
}

func TestDetailPanel_NotVisible(t *testing.T) {
	d := NewDetailPanel()
	d.SetSize(testWidth, testHeight)
	// Not visible by default

	golden.RequireEqual(t, []byte(d.View()))
}

func TestDetailPanel_NoResource(t *testing.T) {
	d := NewDetailPanel()
	d.SetSize(testWidth, testHeight)
	d.Show()

	golden.RequireEqual(t, []byte(d.View()))
}

func TestDetailPanel_WithResource(t *testing.T) {
	d := NewDetailPanel()
	d.SetSize(testWidth, testHeight)
	d.Show()
	d.SetResource(&ResourceItem{
		URN:    "urn:pulumi:dev::my-app::aws:s3/bucket:Bucket::my-bucket",
		Type:   "aws:s3/bucket:Bucket",
		Name:   "my-bucket",
		Op:     OpCreate,
		Status: StatusPending,
		Inputs: map[string]any{
			"bucketName": "my-bucket",
			"region":     "us-west-2",
		},
		Outputs: map[string]any{
			"id":  "bucket-12345",
			"arn": "arn:aws:s3:::my-bucket",
		},
	})

	golden.RequireEqual(t, []byte(d.View()))
}

func TestDetailPanel_WithRunningStatus(t *testing.T) {
	d := NewDetailPanel()
	d.SetSize(testWidth, testHeight)
	d.Show()
	d.SetResource(&ResourceItem{
		URN:       "urn:pulumi:dev::my-app::aws:s3/bucket:Bucket::my-bucket",
		Type:      "aws:s3/bucket:Bucket",
		Name:      "my-bucket",
		Op:        OpUpdate,
		Status:    StatusRunning,
		CurrentOp: OpUpdate,
		OldInputs: map[string]any{
			"bucketName": "old-bucket",
		},
		Inputs: map[string]any{
			"bucketName": "new-bucket",
		},
	})

	golden.RequireEqual(t, []byte(d.View()))
}

func TestConfirmModal_Basic(t *testing.T) {
	m := NewConfirmModal()
	m.SetSize(testWidth, testHeight)
	m.Show("Confirm Action", "Are you sure you want to proceed?", "")

	golden.RequireEqual(t, []byte(m.View()))
}

func TestConfirmModal_WithWarning(t *testing.T) {
	m := NewConfirmModal()
	m.SetSize(testWidth, testHeight)
	m.Show("Delete Resource", "This will permanently delete the resource.", "This action cannot be undone!")

	golden.RequireEqual(t, []byte(m.View()))
}

func TestConfirmModal_CustomLabels(t *testing.T) {
	m := NewConfirmModal()
	m.SetSize(testWidth, testHeight)
	m.SetLabels("No, keep it", "Yes, delete")
	m.SetKeys("n", "d")
	m.Show("Confirm Delete", "Delete this item?", "")

	golden.RequireEqual(t, []byte(m.View()))
}

func TestConfirmModal_Unprotect(t *testing.T) {
	m := NewConfirmModal()
	m.SetSize(testWidth, testHeight)
	m.SetLabels("Cancel", "Unprotect")
	m.ShowWithContext(
		"Unprotect Resource",
		"Remove protection from 'my-bucket'?\n\nType: aws:s3/bucket:Bucket",
		"This will allow the resource to be destroyed.",
		"urn:pulumi:dev::app::aws:s3/bucket:Bucket::my-bucket",
		"my-bucket",
		"aws:s3/bucket:Bucket",
	)

	golden.RequireEqual(t, []byte(m.View()))
}

func TestErrorModal_Basic(t *testing.T) {
	m := NewErrorModal()
	m.SetSize(testWidth, testHeight)
	m.Show("Operation Failed", "The update operation failed", "Error: resource not found\n\nStack trace:\n  at createBucket()\n  at main()")

	golden.RequireEqual(t, []byte(m.View()))
}

func TestErrorModal_LongDetails(t *testing.T) {
	m := NewErrorModal()
	m.SetSize(testWidth, testHeight)

	// Create long details that will require scrolling
	var details strings.Builder
	for i := 1; i <= 30; i++ {
		details.WriteString(fmt.Sprintf("Line %d: Error detail information\n", i))
	}

	m.Show("Multiple Errors", "Several errors occurred during operation", details.String())

	golden.RequireEqual(t, []byte(m.View()))
}

func TestHistoryList_Empty(t *testing.T) {
	h := NewHistoryList()
	h.SetSize(testWidth, testHeight)
	h.SetItems([]HistoryItem{})

	golden.RequireEqual(t, []byte(h.View()))
}

func TestHistoryList_Loading(t *testing.T) {
	h := NewHistoryList()
	h.SetSize(testWidth, testHeight)
	h.SetLoading(true, "Loading history...")

	golden.RequireEqual(t, []byte(h.View()))
}

func TestHistoryList_SingleItem(t *testing.T) {
	h := NewHistoryList()
	h.SetSize(testWidth, testHeight)
	h.SetItems([]HistoryItem{
		{
			Version:   1,
			Kind:      "update",
			StartTime: "2024-01-15T10:30:00Z",
			EndTime:   "2024-01-15T10:35:00Z",
			Result:    "succeeded",
			User:      "developer",
			Message:   "Initial deployment",
			ResourceChanges: map[string]int{
				"create": 5,
			},
		},
	})

	golden.RequireEqual(t, []byte(h.View()))
}

func TestHistoryList_MultipleItems(t *testing.T) {
	h := NewHistoryList()
	h.SetSize(testWidth, testHeight)
	h.SetItems([]HistoryItem{
		{
			Version:   3,
			Kind:      "update",
			StartTime: "2024-01-17T14:00:00Z",
			Result:    "succeeded",
			User:      "developer",
			ResourceChanges: map[string]int{
				"update": 2,
				"same":   3,
			},
		},
		{
			Version:   2,
			Kind:      "preview",
			StartTime: "2024-01-16T09:00:00Z",
			Result:    "succeeded",
			User:      "developer",
		},
		{
			Version:   1,
			Kind:      "update",
			StartTime: "2024-01-15T10:30:00Z",
			Result:    "failed",
			User:      "developer",
			Message:   "Initial deployment",
			ResourceChanges: map[string]int{
				"create": 5,
			},
		},
	})

	golden.RequireEqual(t, []byte(h.View()))
}

func TestHistoryList_DifferentKinds(t *testing.T) {
	h := NewHistoryList()
	h.SetSize(testWidth, testHeight)
	h.SetItems([]HistoryItem{
		{Version: 4, Kind: "destroy", StartTime: "2024-01-20T10:00:00Z", Result: "succeeded"},
		{Version: 3, Kind: "refresh", StartTime: "2024-01-19T10:00:00Z", Result: "succeeded"},
		{Version: 2, Kind: "preview", StartTime: "2024-01-18T10:00:00Z", Result: "succeeded"},
		{Version: 1, Kind: "update", StartTime: "2024-01-17T10:00:00Z", Result: "in-progress"},
	})

	golden.RequireEqual(t, []byte(h.View()))
}

func TestHistoryDetailPanel_NotVisible(t *testing.T) {
	d := NewHistoryDetailPanel()
	d.SetSize(testWidth, testHeight)

	golden.RequireEqual(t, []byte(d.View()))
}

func TestHistoryDetailPanel_NoItem(t *testing.T) {
	d := NewHistoryDetailPanel()
	d.SetSize(testWidth, testHeight)
	d.Show()

	golden.RequireEqual(t, []byte(d.View()))
}

func TestHistoryDetailPanel_WithItem(t *testing.T) {
	d := NewHistoryDetailPanel()
	d.SetSize(testWidth, testHeight)
	d.Show()
	d.SetItem(&HistoryItem{
		Version:   5,
		Kind:      "update",
		StartTime: "2024-01-15T10:30:00Z",
		EndTime:   "2024-01-15T10:35:00Z",
		Result:    "succeeded",
		User:      "developer",
		UserEmail: "dev@example.com",
		Message:   "Add new S3 bucket for static assets",
		ResourceChanges: map[string]int{
			"create": 2,
			"update": 1,
			"same":   5,
		},
	})

	golden.RequireEqual(t, []byte(d.View()))
}

func TestHistoryDetailPanel_FailedUpdate(t *testing.T) {
	d := NewHistoryDetailPanel()
	d.SetSize(testWidth, testHeight)
	d.Show()
	d.SetItem(&HistoryItem{
		Version:   3,
		Kind:      "update",
		StartTime: "2024-01-15T10:30:00Z",
		EndTime:   "2024-01-15T10:31:00Z",
		Result:    "failed",
		User:      "developer",
		Message:   "Failed to create resource",
		ResourceChanges: map[string]int{
			"create": 0,
			"delete": 1,
		},
	})

	golden.RequireEqual(t, []byte(d.View()))
}

func TestImportModal_Basic(t *testing.T) {
	m := NewImportModal()
	m.SetSize(testWidth, testHeight)
	m.Show("aws:s3/bucket:Bucket", "my-bucket", "urn:pulumi:dev::app::aws:s3/bucket:Bucket::my-bucket", "")
	m.SetSuggestions([]ImportSuggestion{}) // No suggestions

	golden.RequireEqual(t, []byte(m.View()))
}

func TestImportModal_WithSuggestions(t *testing.T) {
	m := NewImportModal()
	m.SetSize(testWidth, testHeight)
	m.Show("aws:s3/bucket:Bucket", "my-bucket", "urn:pulumi:dev::app::aws:s3/bucket:Bucket::my-bucket", "")
	m.SetSuggestions([]ImportSuggestion{
		{ID: "bucket-123", Label: "bucket-123", Description: "Production bucket", PluginName: "aws"},
		{ID: "bucket-456", Label: "bucket-456", Description: "Staging bucket", PluginName: "aws"},
		{ID: "bucket-789", Label: "bucket-789", Description: "Dev bucket", PluginName: "aws"},
	})

	golden.RequireEqual(t, []byte(m.View()))
}

func TestImportModal_Loading(t *testing.T) {
	m := NewImportModal()
	m.SetSize(testWidth, testHeight)
	m.Show("aws:s3/bucket:Bucket", "my-bucket", "urn:pulumi:dev::app::aws:s3/bucket:Bucket::my-bucket", "")
	m.SetLoadingSuggestions(true)

	golden.RequireEqual(t, []byte(m.View()))
}

func TestImportModal_WithError(t *testing.T) {
	m := NewImportModal()
	m.SetSize(testWidth, testHeight)
	m.Show("aws:s3/bucket:Bucket", "my-bucket", "urn:pulumi:dev::app::aws:s3/bucket:Bucket::my-bucket", "")
	m.SetSuggestions([]ImportSuggestion{})
	m.SetError(errors.New("invalid import ID format"))

	golden.RequireEqual(t, []byte(m.View()))
}

// testSelectorItem implements SelectorItem for testing
type testSelectorItem struct {
	name    string
	current bool
}

func (t testSelectorItem) Label() string   { return t.name }
func (t testSelectorItem) IsCurrent() bool { return t.current }

func TestSelectorDialog_Empty(t *testing.T) {
	s := NewSelectorDialog[testSelectorItem]("Select Item")
	s.SetSize(testWidth, testHeight)
	s.Show()
	s.SetItems([]testSelectorItem{})

	golden.RequireEqual(t, []byte(s.View()))
}

func TestSelectorDialog_Loading(t *testing.T) {
	s := NewSelectorDialog[testSelectorItem]("Select Item")
	s.SetSize(testWidth, testHeight)
	s.Show()
	s.SetLoading(true)

	golden.RequireEqual(t, []byte(s.View()))
}

func TestSelectorDialog_WithItems(t *testing.T) {
	s := NewSelectorDialog[testSelectorItem]("Select Item")
	s.SetSize(testWidth, testHeight)
	s.Show()
	s.SetItems([]testSelectorItem{
		{name: "item-1", current: false},
		{name: "item-2", current: true},
		{name: "item-3", current: false},
	})

	golden.RequireEqual(t, []byte(s.View()))
}

func TestSelectorDialog_WithError(t *testing.T) {
	s := NewSelectorDialog[testSelectorItem]("Select Item")
	s.SetSize(testWidth, testHeight)
	s.Show()
	s.SetError(errors.New("failed to load items"))

	golden.RequireEqual(t, []byte(s.View()))
}

func TestSelectorDialog_ManyItems(t *testing.T) {
	s := NewSelectorDialog[testSelectorItem]("Select Item")
	s.SetSize(testWidth, testHeight)
	s.Show()

	items := make([]testSelectorItem, 15)
	for i := range items {
		items[i] = testSelectorItem{name: fmt.Sprintf("item-%d", i+1), current: i == 5}
	}
	s.SetItems(items)

	golden.RequireEqual(t, []byte(s.View()))
}

func TestStackSelector_Empty(t *testing.T) {
	s := NewStackSelector()
	s.SetSize(testWidth, testHeight)
	s.Show()
	s.SetStacks([]StackItem{})

	golden.RequireEqual(t, []byte(s.View()))
}

func TestStackSelector_WithStacks(t *testing.T) {
	s := NewStackSelector()
	s.SetSize(testWidth, testHeight)
	s.Show()
	s.SetStacks([]StackItem{
		{Name: "dev", Current: true},
		{Name: "staging", Current: false},
		{Name: "production", Current: false},
	})

	golden.RequireEqual(t, []byte(s.View()))
}

func TestStackSelector_NoNewOption(t *testing.T) {
	s := NewStackSelector()
	s.SetSize(testWidth, testHeight)
	s.SetShowNewOption(false)
	s.Show()
	s.SetStacks([]StackItem{
		{Name: "dev", Current: true},
		{Name: "staging", Current: false},
	})

	golden.RequireEqual(t, []byte(s.View()))
}

func TestWorkspaceSelector_Empty(t *testing.T) {
	s := NewWorkspaceSelector()
	s.SetSize(testWidth, testHeight)
	s.Show()
	s.SetWorkspaces([]WorkspaceItem{})

	golden.RequireEqual(t, []byte(s.View()))
}

func TestWorkspaceSelector_WithWorkspaces(t *testing.T) {
	s := NewWorkspaceSelector()
	s.SetSize(testWidth, testHeight)
	s.Show()
	s.SetWorkspaces([]WorkspaceItem{
		{Name: "my-app", Path: "/home/user/projects/my-app", RelativePath: "./my-app", Current: true},
		{Name: "another-app", Path: "/home/user/projects/another-app", RelativePath: "./another-app", Current: false},
		{Name: "third-app", Path: "/home/user/projects/third-app", RelativePath: "./third-app", Current: false},
	})

	golden.RequireEqual(t, []byte(s.View()))
}

func TestStepModal_SingleStep(t *testing.T) {
	m := NewStepModal("Configure")
	m.SetSize(testWidth, testHeight)
	m.SetSteps([]StepModalStep{
		{
			Title:            "Enter Name",
			InputLabel:       "Name",
			InputPlaceholder: "Enter a name...",
		},
	})
	m.Show()

	golden.RequireEqual(t, []byte(m.View()))
}

func TestStepModal_MultiStep(t *testing.T) {
	m := NewStepModal("Setup Wizard")
	m.SetSize(testWidth, testHeight)
	m.SetSteps([]StepModalStep{
		{
			Title:            "Step 1: Name",
			InputLabel:       "Name",
			InputPlaceholder: "Enter name...",
		},
		{
			Title:            "Step 2: Region",
			InputLabel:       "Region",
			InputPlaceholder: "Enter region...",
		},
		{
			Title:            "Step 3: Confirm",
			InputLabel:       "Type 'yes' to confirm",
			InputPlaceholder: "yes",
		},
	})
	m.Show()

	golden.RequireEqual(t, []byte(m.View()))
}

func TestStepModal_WithInfoLines(t *testing.T) {
	m := NewStepModal("Configure Resource")
	m.SetSize(testWidth, testHeight)
	m.SetSteps([]StepModalStep{
		{
			Title: "Select Option",
			InfoLines: []InfoLine{
				{Label: "Resource", Value: "my-bucket"},
				{Label: "Type", Value: "aws:s3/bucket:Bucket"},
			},
			InputLabel:       "Option",
			InputPlaceholder: "Enter option...",
		},
	})
	m.Show()

	golden.RequireEqual(t, []byte(m.View()))
}

func TestStepModal_WithSuggestions(t *testing.T) {
	m := NewStepModal("Select Provider")
	m.SetSize(testWidth, testHeight)
	m.SetSteps([]StepModalStep{
		{
			Title: "Choose Provider",
			Suggestions: []StepSuggestion{
				{ID: "aws", Label: "AWS", Description: "Amazon Web Services"},
				{ID: "gcp", Label: "GCP", Description: "Google Cloud Platform"},
				{ID: "azure", Label: "Azure", Description: "Microsoft Azure"},
			},
			InputLabel:       "Provider",
			InputPlaceholder: "Enter provider...",
		},
	})
	m.Show()

	golden.RequireEqual(t, []byte(m.View()))
}

func TestStepModal_WithWarning(t *testing.T) {
	m := NewStepModal("Dangerous Action")
	m.SetSize(testWidth, testHeight)
	m.SetSteps([]StepModalStep{
		{
			Title:            "Confirm Action",
			Warning:          "This will delete all data and cannot be undone!",
			InputLabel:       "Confirmation",
			InputPlaceholder: "Type 'delete' to confirm...",
		},
	})
	m.Show()

	golden.RequireEqual(t, []byte(m.View()))
}

func TestStepModal_PasswordMode(t *testing.T) {
	m := NewStepModal("Enter Credentials")
	m.SetSize(testWidth, testHeight)
	m.SetSteps([]StepModalStep{
		{
			Title:            "Enter Password",
			InputLabel:       "Password",
			InputPlaceholder: "Enter password...",
			PasswordMode:     true,
		},
	})
	m.Show()

	golden.RequireEqual(t, []byte(m.View()))
}

func TestStackInitModal_Initial(t *testing.T) {
	m := NewStackInitModal()
	m.SetSize(testWidth, testHeight)
	m.SetBackendInfo("user@example.com", "https://api.pulumi.com")
	m.Show()

	golden.RequireEqual(t, []byte(m.View()))
}

func TestStackInitModal_WithStackFiles(t *testing.T) {
	m := NewStackInitModal()
	m.SetSize(testWidth, testHeight)
	m.SetBackendInfo("user@example.com", "file://~")
	m.Show()
	m.SetStackFiles([]pulumi.StackFileInfo{
		{Name: "dev", HasEncryption: false, SecretsProvider: ""},
		{Name: "staging", HasEncryption: true, SecretsProvider: "awskms://alias/pulumi"},
		{Name: "prod", HasEncryption: true, SecretsProvider: "awskms://alias/pulumi"},
	})

	golden.RequireEqual(t, []byte(m.View()))
}

// errTest is a simple test error
type testError struct{}

func (e testError) Error() string {
	return "test error"
}

var errTest = testError{}

func TestFilterState_Basic(t *testing.T) {
	f := NewFilterState()

	// Initially inactive
	if f.Active() {
		t.Error("filter should be inactive initially")
	}
	if f.Text() != "" {
		t.Error("filter text should be empty initially")
	}

	// Activate
	f.Activate()
	if !f.Active() {
		t.Error("filter should be active after Activate()")
	}

	// Deactivate
	f.Deactivate()
	if f.Active() {
		t.Error("filter should be inactive after Deactivate()")
	}
}

func TestFilterState_Matches(t *testing.T) {
	f := NewFilterState()

	// Empty filter matches everything
	if !f.Matches("anything") {
		t.Error("empty filter should match any text")
	}
	if !f.MatchesAny("one", "two", "three") {
		t.Error("empty filter should match any of the texts")
	}

	// Activate and set filter text
	f.Activate()
	f.input.SetValue("bucket")

	// Case-insensitive matching
	if !f.Matches("my-bucket") {
		t.Error("filter should match 'my-bucket'")
	}
	if !f.Matches("MY-BUCKET") {
		t.Error("filter should match 'MY-BUCKET' (case-insensitive)")
	}
	if !f.Matches("Bucket-123") {
		t.Error("filter should match 'Bucket-123' (case-insensitive)")
	}
	if f.Matches("my-table") {
		t.Error("filter should not match 'my-table'")
	}

	// MatchesAny
	if !f.MatchesAny("table", "bucket", "queue") {
		t.Error("filter should match when any text matches")
	}
	if f.MatchesAny("table", "queue", "topic") {
		t.Error("filter should not match when no text matches")
	}
}

func TestFilterState_EscapeBehavior(t *testing.T) {
	f := NewFilterState()
	f.Activate()
	f.input.SetValue("test")

	// Escape exits filter mode but keeps text applied
	escKey := tea.KeyMsg{Type: tea.KeyEscape}
	_, handled := f.Update(escKey)
	if !handled {
		t.Error("escape should be handled")
	}
	if f.Active() {
		t.Error("escape should deactivate filter")
	}
	if f.Text() != "test" {
		t.Error("escape should keep filter text applied")
	}

	// Re-activating filter should reset the text
	f.Activate()
	if f.Text() != "" {
		t.Error("re-activating filter should reset text")
	}
	if !f.Active() {
		t.Error("filter should be active after re-activation")
	}
}

func TestFilterState_EnterBehavior(t *testing.T) {
	f := NewFilterState()
	f.Activate()
	f.input.SetValue("test")

	// Enter exits filter mode but keeps text
	enterKey := tea.KeyMsg{Type: tea.KeyEnter}
	_, handled := f.Update(enterKey)
	if !handled {
		t.Error("enter should be handled")
	}
	if f.Active() {
		t.Error("enter should deactivate filter mode")
	}
	if f.Text() != "test" {
		t.Error("enter should keep filter text applied")
	}
}

func TestResourceList_Filter(t *testing.T) {
	flags := make(map[string]ResourceFlags)
	r := NewResourceList(flags)
	r.SetSize(testWidth, testHeight)
	r.SetItems([]ResourceItem{
		{URN: "urn:1", Type: "aws:s3/bucket:Bucket", Name: "my-bucket", Op: OpCreate},
		{URN: "urn:2", Type: "aws:dynamodb/table:Table", Name: "my-table", Op: OpUpdate},
		{URN: "urn:3", Type: "aws:s3/bucket:Bucket", Name: "other-bucket", Op: OpSame},
		{URN: "urn:4", Type: "aws:lambda/function:Function", Name: "my-function", Op: OpDelete},
	})

	// Simulate pressing "/" to activate filter
	r.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	// Type "bucket" by simulating key presses
	for _, char := range "bucket" {
		r.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}})
	}

	golden.RequireEqual(t, []byte(r.View()))
}

func TestResourceList_FilterNoMatches(t *testing.T) {
	flags := make(map[string]ResourceFlags)
	r := NewResourceList(flags)
	r.SetSize(testWidth, testHeight)
	r.SetItems([]ResourceItem{
		{URN: "urn:1", Type: "aws:s3/bucket:Bucket", Name: "my-bucket", Op: OpCreate},
		{URN: "urn:2", Type: "aws:dynamodb/table:Table", Name: "my-table", Op: OpUpdate},
	})

	// Simulate pressing "/" to activate filter
	r.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	// Type "nonexistent" by simulating key presses
	for _, char := range "nonexistent" {
		r.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}})
	}

	golden.RequireEqual(t, []byte(r.View()))
}

func TestSelectorDialog_Filter(t *testing.T) {
	s := NewSelectorDialog[testSelectorItem]("Select Item")
	s.SetSize(testWidth, testHeight)
	s.Show()
	s.SetItems([]testSelectorItem{
		{name: "dev-bucket", current: false},
		{name: "staging-bucket", current: false},
		{name: "prod-table", current: true},
		{name: "prod-queue", current: false},
	})

	// Simulate pressing "/" to activate filter
	s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	// Type "bucket" by simulating key presses
	for _, char := range "bucket" {
		s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}})
	}

	golden.RequireEqual(t, []byte(s.View()))
}

func TestHistoryList_Filter(t *testing.T) {
	h := NewHistoryList()
	h.SetSize(testWidth, testHeight)
	h.SetItems([]HistoryItem{
		{Version: 1, Kind: "update", StartTime: "2024-01-15T10:00:00Z", Result: "succeeded", User: "dev"},
		{Version: 2, Kind: "preview", StartTime: "2024-01-16T10:00:00Z", Result: "succeeded", User: "admin"},
		{Version: 3, Kind: "update", StartTime: "2024-01-17T10:00:00Z", Result: "failed", User: "dev"},
		{Version: 4, Kind: "destroy", StartTime: "2024-01-18T10:00:00Z", Result: "succeeded", User: "admin"},
	})

	// Simulate pressing "/" to activate filter
	h.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	// Type "update" by simulating key presses
	for _, char := range "update" {
		h.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}})
	}

	golden.RequireEqual(t, []byte(h.View()))
}
