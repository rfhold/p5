package main

import (
	"github.com/rfhold/p5/internal/plugins"
	"github.com/rfhold/p5/internal/pulumi"
)

// Messages for data fetching
type projectInfoMsg *pulumi.ProjectInfo
type errMsg error
type previewEventMsg pulumi.PreviewEvent
type operationEventMsg pulumi.OperationEvent
type stackResourcesMsg []pulumi.ResourceInfo
type stacksListMsg struct {
	Stacks []pulumi.StackInfo
	Files  []pulumi.StackFileInfo
}
type stackSelectedMsg string
type workspacesListMsg []pulumi.WorkspaceInfo
type workspaceSelectedMsg string
type workspaceCheckMsg bool // true if current dir is a valid workspace
type stackHistoryMsg []pulumi.UpdateSummary
type importResultMsg *pulumi.CommandResult
type stateDeleteResultMsg *pulumi.CommandResult
type protectResultMsg struct {
	Result    *pulumi.CommandResult
	Protected bool   // true if protecting, false if unprotecting
	URN       string // the resource URN
	Name      string // the resource name (for toast message)
}

// Plugin-related messages
type pluginAuthResultMsg []plugins.AuthenticateResult
type pluginAuthErrorMsg error

// authCompleteMsg is sent when plugin authentication completes (success or error)
// This message always releases the auth busy lock and executes pending operations
type authCompleteMsg struct {
	results []plugins.AuthenticateResult
	err     error
}

// pluginInitDoneMsg is sent when initial plugin auth completes
type pluginInitDoneMsg struct {
	results []plugins.AuthenticateResult
	err     error
}

// initPreviewMsg is sent to start a preview from Init
type initPreviewMsg struct {
	op pulumi.OperationType
	ch <-chan pulumi.PreviewEvent
}

// Import suggestion messages
type importSuggestionsMsg []*plugins.AggregatedImportSuggestion
type importSuggestionsErrMsg error

// Stack init messages
type whoAmIMsg *pulumi.WhoAmIInfo
type stackFilesMsg []pulumi.StackFileInfo
type stackInitResultMsg struct {
	StackName string
	Error     error
}

// Open resource messages
type openResourceActionMsg struct {
	Response   *plugins.OpenResourceResponse
	PluginName string
}
type openResourceErrMsg error
type openResourceExecDoneMsg struct {
	Error error
}
