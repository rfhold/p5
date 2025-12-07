package plugins

import (
	"context"
)

// FakePluginProvider implements PluginProvider for testing.
// Configure behavior via function fields and default return values.
type FakePluginProvider struct {
	// AuthProvider methods
	GetMergedAuthEnvFunc         func() map[string]string
	GetAllEnvFunc                func() map[string]string
	ApplyEnvToProcessFunc        func()
	GetCredentialsSummaryFunc    func() []CredentialsSummary
	InvalidateCredentialsFunc    func(pluginName string)
	InvalidateAllCredentialsFunc func()

	// ImportHelper methods
	GetImportSuggestionsFunc func(ctx context.Context, req *ImportSuggestionsRequest) ([]*AggregatedImportSuggestion, error)
	HasImportHelpersFunc     func() bool

	// ResourceOpener methods
	OpenResourceFunc       func(ctx context.Context, req *OpenResourceRequest) (*OpenResourceResponse, string, error)
	HasResourceOpenersFunc func() bool

	// PluginProvider methods
	InitializeFunc                      func(ctx context.Context, workDir, programName, stackName string) ([]AuthenticateResult, error)
	CloseFunc                           func(ctx context.Context)
	GetMergedConfigFunc                 func() *P5Config
	ShouldRefreshCredentialsFunc        func(pluginName string, newWorkDir, newStackName, newProgramName string, newProgramConfig, newStackConfig map[string]any) bool
	InvalidateCredentialsForContextFunc func(workDir, stackName, programName string, p5Config *P5Config)
	AuthenticateAllFunc                 func(ctx context.Context, programName, stackName string, p5Config *P5Config, workDir string) ([]AuthenticateResult, error)

	// Default return values
	AuthEnv              map[string]string
	AllEnv               map[string]string
	CredentialsSummary   []CredentialsSummary
	ImportSuggestions    []*AggregatedImportSuggestion
	HasImportHelper      bool
	OpenResourceResponse *OpenResourceResponse
	OpenResourcePlugin   string
	HasResourceOpener    bool
	AuthResults          []AuthenticateResult
	MergedConfig         *P5Config
	ShouldRefresh        bool

	// Calls tracks all method invocations.
	Calls struct {
		GetMergedAuthEnv                int
		GetAllEnv                       int
		ApplyEnvToProcess               int
		GetCredentialsSummary           int
		InvalidateCredentials           []string
		InvalidateAllCredentials        int
		GetImportSuggestions            []*ImportSuggestionsRequest
		HasImportHelpers                int
		OpenResource                    []*OpenResourceRequest
		HasResourceOpeners              int
		Initialize                      []InitializeCall
		Close                           int
		GetMergedConfig                 int
		ShouldRefreshCredentials        []ShouldRefreshCredentialsCall
		InvalidateCredentialsForContext []InvalidateCredentialsForContextCall
		AuthenticateAll                 []AuthenticateAllCall
	}
}

type InitializeCall struct {
	WorkDir     string
	ProgramName string
	StackName   string
}

type ShouldRefreshCredentialsCall struct {
	PluginName       string
	NewWorkDir       string
	NewStackName     string
	NewProgramName   string
	NewProgramConfig map[string]any
	NewStackConfig   map[string]any
}

type InvalidateCredentialsForContextCall struct {
	WorkDir     string
	StackName   string
	ProgramName string
	P5Config    *P5Config
}

type AuthenticateAllCall struct {
	ProgramName string
	StackName   string
	P5Config    *P5Config
	WorkDir     string
}

// AuthProvider interface implementation

func (f *FakePluginProvider) GetMergedAuthEnv() map[string]string {
	f.Calls.GetMergedAuthEnv++
	if f.GetMergedAuthEnvFunc != nil {
		return f.GetMergedAuthEnvFunc()
	}
	if f.AuthEnv == nil {
		return make(map[string]string)
	}
	return f.AuthEnv
}

func (f *FakePluginProvider) GetAllEnv() map[string]string {
	f.Calls.GetAllEnv++
	if f.GetAllEnvFunc != nil {
		return f.GetAllEnvFunc()
	}
	if f.AllEnv == nil {
		return make(map[string]string)
	}
	return f.AllEnv
}

func (f *FakePluginProvider) ApplyEnvToProcess() {
	f.Calls.ApplyEnvToProcess++
	if f.ApplyEnvToProcessFunc != nil {
		f.ApplyEnvToProcessFunc()
	}
}

func (f *FakePluginProvider) GetCredentialsSummary() []CredentialsSummary {
	f.Calls.GetCredentialsSummary++
	if f.GetCredentialsSummaryFunc != nil {
		return f.GetCredentialsSummaryFunc()
	}
	return f.CredentialsSummary
}

func (f *FakePluginProvider) InvalidateCredentials(pluginName string) {
	f.Calls.InvalidateCredentials = append(f.Calls.InvalidateCredentials, pluginName)
	if f.InvalidateCredentialsFunc != nil {
		f.InvalidateCredentialsFunc(pluginName)
	}
}

func (f *FakePluginProvider) InvalidateAllCredentials() {
	f.Calls.InvalidateAllCredentials++
	if f.InvalidateAllCredentialsFunc != nil {
		f.InvalidateAllCredentialsFunc()
	}
}

// ImportHelper interface implementation

func (f *FakePluginProvider) GetImportSuggestions(ctx context.Context, req *ImportSuggestionsRequest) ([]*AggregatedImportSuggestion, error) {
	f.Calls.GetImportSuggestions = append(f.Calls.GetImportSuggestions, req)
	if f.GetImportSuggestionsFunc != nil {
		return f.GetImportSuggestionsFunc(ctx, req)
	}
	return f.ImportSuggestions, nil
}

func (f *FakePluginProvider) HasImportHelpers() bool {
	f.Calls.HasImportHelpers++
	if f.HasImportHelpersFunc != nil {
		return f.HasImportHelpersFunc()
	}
	return f.HasImportHelper
}

// ResourceOpener interface implementation

func (f *FakePluginProvider) OpenResource(ctx context.Context, req *OpenResourceRequest) (resp *OpenResourceResponse, pluginName string, err error) {
	f.Calls.OpenResource = append(f.Calls.OpenResource, req)
	if f.OpenResourceFunc != nil {
		return f.OpenResourceFunc(ctx, req)
	}
	return f.OpenResourceResponse, f.OpenResourcePlugin, nil
}

func (f *FakePluginProvider) HasResourceOpeners() bool {
	f.Calls.HasResourceOpeners++
	if f.HasResourceOpenersFunc != nil {
		return f.HasResourceOpenersFunc()
	}
	return f.HasResourceOpener
}

// PluginProvider interface implementation

func (f *FakePluginProvider) Initialize(ctx context.Context, workDir, programName, stackName string) ([]AuthenticateResult, error) {
	f.Calls.Initialize = append(f.Calls.Initialize, InitializeCall{workDir, programName, stackName})
	if f.InitializeFunc != nil {
		return f.InitializeFunc(ctx, workDir, programName, stackName)
	}
	return f.AuthResults, nil
}

func (f *FakePluginProvider) Close(ctx context.Context) {
	f.Calls.Close++
	if f.CloseFunc != nil {
		f.CloseFunc(ctx)
	}
}

func (f *FakePluginProvider) GetMergedConfig() *P5Config {
	f.Calls.GetMergedConfig++
	if f.GetMergedConfigFunc != nil {
		return f.GetMergedConfigFunc()
	}
	if f.MergedConfig == nil {
		return &P5Config{Plugins: make(map[string]PluginConfig)}
	}
	return f.MergedConfig
}

func (f *FakePluginProvider) ShouldRefreshCredentials(pluginName, newWorkDir, newStackName, newProgramName string, newProgramConfig, newStackConfig map[string]any) bool {
	f.Calls.ShouldRefreshCredentials = append(f.Calls.ShouldRefreshCredentials, ShouldRefreshCredentialsCall{
		PluginName:       pluginName,
		NewWorkDir:       newWorkDir,
		NewStackName:     newStackName,
		NewProgramName:   newProgramName,
		NewProgramConfig: newProgramConfig,
		NewStackConfig:   newStackConfig,
	})
	if f.ShouldRefreshCredentialsFunc != nil {
		return f.ShouldRefreshCredentialsFunc(pluginName, newWorkDir, newStackName, newProgramName, newProgramConfig, newStackConfig)
	}
	return f.ShouldRefresh
}

func (f *FakePluginProvider) InvalidateCredentialsForContext(workDir, stackName, programName string, p5Config *P5Config) {
	f.Calls.InvalidateCredentialsForContext = append(f.Calls.InvalidateCredentialsForContext, InvalidateCredentialsForContextCall{
		WorkDir:     workDir,
		StackName:   stackName,
		ProgramName: programName,
		P5Config:    p5Config,
	})
	if f.InvalidateCredentialsForContextFunc != nil {
		f.InvalidateCredentialsForContextFunc(workDir, stackName, programName, p5Config)
	}
}

func (f *FakePluginProvider) AuthenticateAll(ctx context.Context, programName, stackName string, p5Config *P5Config, workDir string) ([]AuthenticateResult, error) {
	f.Calls.AuthenticateAll = append(f.Calls.AuthenticateAll, AuthenticateAllCall{
		ProgramName: programName,
		StackName:   stackName,
		P5Config:    p5Config,
		WorkDir:     workDir,
	})
	if f.AuthenticateAllFunc != nil {
		return f.AuthenticateAllFunc(ctx, programName, stackName, p5Config, workDir)
	}
	return f.AuthResults, nil
}

// Compile-time interface compliance check
var _ PluginProvider = (*FakePluginProvider)(nil)
