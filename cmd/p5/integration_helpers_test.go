//go:build integration

package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/golden"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/muesli/termenv"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"

	"github.com/rfhold/p5/internal/plugins"
	"github.com/rfhold/p5/internal/pulumi"
)

func init() {
	// Force consistent color profile for reproducible tests across environments
	lipgloss.SetColorProfile(termenv.Ascii)
}

const (
	goldenWidth  = 120
	goldenHeight = 40
)

type testModelOption func(*Dependencies, *AppContext)

type outputCapture struct {
	tm       *teatest.TestModel
	captured bytes.Buffer
	mu       sync.Mutex
}

type testHarness struct {
	t       *testing.T
	tm      *teatest.TestModel
	capture *outputCapture
}

type TestEnvironment struct {
	t          *testing.T
	WorkDir    string
	BackendDir string
	StackName  string
	Env        map[string]string
	cleaned    bool
}

type TestEnvOption func(*TestEnvironment)

func WithStackName(name string) TestEnvOption {
	return func(te *TestEnvironment) {
		te.StackName = name
	}
}

func WithExistingStacks(names ...string) TestEnvOption {
	return func(te *TestEnvironment) {
		for _, name := range names {
			configPath := filepath.Join(te.WorkDir, fmt.Sprintf("Pulumi.%s.yaml", name))
			if err := os.WriteFile(configPath, []byte("config: {}\n"), 0644); err != nil {
				te.t.Fatalf("failed to create stack config %s: %v", name, err)
			}
		}
	}
}

func getTestWorkDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	testDir := filepath.Join(wd, "..", "..", "test", "simple")
	absPath, err := filepath.Abs(testDir)
	if err != nil {
		t.Fatalf("failed to get absolute path: %v", err)
	}
	return absPath
}

func createTestModel(t *testing.T, opts ...testModelOption) Model {
	t.Helper()

	deps := &Dependencies{
		StackOperator: &pulumi.FakeStackOperator{},
		StackReader:   &pulumi.FakeStackReader{},
		WorkspaceReader: &pulumi.FakeWorkspaceReader{
			ValidWorkDir: true,
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

func withStackOperator(op pulumi.StackOperator) testModelOption {
	return func(d *Dependencies, _ *AppContext) {
		d.StackOperator = op
	}
}

func withStackReader(reader pulumi.StackReader) testModelOption {
	return func(d *Dependencies, _ *AppContext) {
		d.StackReader = reader
	}
}

func withWorkspaceReader(reader pulumi.WorkspaceReader) testModelOption {
	return func(d *Dependencies, _ *AppContext) {
		d.WorkspaceReader = reader
	}
}

func withStartView(view string) testModelOption {
	return func(_ *Dependencies, ctx *AppContext) {
		ctx.StartView = view
	}
}

func withStackName(name string) testModelOption {
	return func(_ *Dependencies, ctx *AppContext) {
		ctx.StackName = name
	}
}

func withResources(resources []pulumi.ResourceInfo) testModelOption {
	return func(d *Dependencies, _ *AppContext) {
		d.StackReader = &pulumi.FakeStackReader{
			Resources: resources,
		}
	}
}

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

func withPluginProvider(provider plugins.PluginProvider) testModelOption {
	return func(d *Dependencies, _ *AppContext) {
		d.PluginProvider = provider
	}
}

func newOutputCapture(tm *teatest.TestModel) *outputCapture {
	return &outputCapture{tm: tm}
}

func (oc *outputCapture) Read(p []byte) (n int, err error) {
	n, err = oc.tm.Output().Read(p)
	if n > 0 {
		oc.mu.Lock()
		oc.captured.Write(p[:n])
		oc.mu.Unlock()
	}
	return n, err
}

func (oc *outputCapture) AllOutput() []byte {
	remaining, _ := io.ReadAll(oc.tm.Output())
	if len(remaining) > 0 {
		oc.mu.Lock()
		oc.captured.Write(remaining)
		oc.mu.Unlock()
	}
	oc.mu.Lock()
	defer oc.mu.Unlock()
	return oc.captured.Bytes()
}

func newTestHarness(t *testing.T, m Model) *testHarness {
	t.Helper()
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(goldenWidth, goldenHeight),
	)
	oc := newOutputCapture(tm)
	return &testHarness{t: t, tm: tm, capture: oc}
}

func (h *testHarness) Send(msg tea.Msg) {
	h.tm.Send(msg)
}

func (h *testHarness) WaitFor(content string, timeout time.Duration) {
	h.t.Helper()
	teatest.WaitFor(h.t, h.capture,
		func(bts []byte) bool {
			return bytes.Contains(bts, []byte(content))
		},
		teatest.WithCheckInterval(50*time.Millisecond),
		teatest.WithDuration(timeout),
	)
}

func (h *testHarness) WaitForAny(contents []string, timeout time.Duration) {
	h.t.Helper()
	teatest.WaitFor(h.t, h.capture,
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

func (h *testHarness) Snapshot(name string) {
	h.t.Helper()
	// Wait for UI to stabilize - operations may still be updating
	time.Sleep(500 * time.Millisecond)
	out := h.capture.AllOutput()
	// Extract only the last frame to avoid capturing spinner animation differences
	lastFrame := extractLastFrame(out)
	// Normalize dynamic content (temp paths, random values)
	normalized := normalizeDynamicContent(string(lastFrame))
	h.t.Run(name, func(t *testing.T) {
		golden.RequireEqual(t, []byte(normalized))
	})
}

func (h *testHarness) FinalSnapshot(name string) {
	h.t.Helper()
	time.Sleep(100 * time.Millisecond)
	// Send quit to allow FinalModel to complete
	h.tm.Send(tea.Quit())
	finalModel := h.tm.FinalModel(h.t, teatest.WithFinalTimeout(5*time.Second))
	view := normalizeSpinners(finalModel.View())
	// Normalize dynamic content (temp paths, random values)
	view = normalizeDynamicContent(view)
	h.t.Run(name, func(t *testing.T) {
		golden.RequireEqual(t, []byte(view))
	})
}

func extractLastFrame(output []byte) []byte {
	// Bubbletea redraws by moving cursor up N lines (ESC[<N>A) and redrawing
	// Find the last cursor-up sequence which marks the start of the final frame
	str := string(output)

	// Look for ESC[<digits>A pattern (cursor up) - this is how bubbletea redraws
	// We want to find the last occurrence and take everything after it
	lastCursorUp := -1
	for i := len(str) - 1; i >= 0; i-- {
		if i >= 2 && str[i] == 'A' {
			// Look backwards for digits and ESC[
			j := i - 1
			for j > 0 && str[j] >= '0' && str[j] <= '9' {
				j--
			}
			if j >= 1 && str[j] == '[' && str[j-1] == '\x1b' {
				lastCursorUp = j - 1
				break
			}
		}
	}

	var result string
	if lastCursorUp > 0 {
		result = str[lastCursorUp:]
	} else {
		result = str
	}

	// Normalize spinner characters to avoid timing-based test failures
	// Braille spinner chars: ⣾ ⣷ ⣯ ⣟ ⡿ ⢿ ⣻ ⣽
	result = normalizeSpinners(result)

	return []byte(result)
}

func normalizeSpinners(s string) string {
	spinnerChars := []string{"⣾", "⣷", "⣯", "⣟", "⡿", "⢿", "⣻", "⣽"}
	for _, char := range spinnerChars {
		s = replaceAll(s, char, "◐")
	}
	return s
}

func normalizeDynamicContent(s string) string {
	result := s

	// Normalize temp directory paths in backend URLs with length preservation
	// to avoid spacing differences in formatted output
	result = normalizeLengthPreserving(result, `file:///tmp/p5-test-backend-[A-Za-z0-9]+`, "file:///tmp/p5-test-backend-XXXXXX")

	// Normalize Stack resource type display - Pulumi sometimes returns the full type
	// "pulumi:pulumi:Stack" and sometimes just the stack name (e.g., "simple-test").
	// This normalizes the stack resource line to use the consistent pulumi:pulumi:Stack format.
	// The stack resource is identified by having the same type and name (both are project-stack format)
	result = normalizeStackResourceType(result)

	// Normalize random hex values (16 hex chars) - common in resource IDs
	// Only normalize values that appear after specific property names
	result = normalizePattern(result, `(baseId|hex):\s*"[a-f0-9]{16}"`, `$1: "<HEX-16-CHARS>"`)

	// Normalize base64 values (including standalone 'id' for RandomId resources)
	result = normalizeLengthPreserving(result, `(b64Std|b64Url|id):\s*"[A-Za-z0-9+/=_-]+"`, `$1: "<B64>"`)

	// Normalize decimal IDs - preserve approximate length
	result = normalizeLengthPreserving(result, `dec:\s*"\d+"`, `dec: "<DECIMAL-ID>"`)

	// Normalize result strings that look like random passwords/strings
	result = normalizeLengthPreserving(result, `result:\s*"[^"]{10,}"`, `result: "<RANDOM-VALUE>"`)

	// Normalize history timestamps (format: YYYY-MM-DD HH:MM)
	// These vary based on when the test runs
	result = normalizePattern(result, `\d{4}-\d{2}-\d{2} \d{2}:\d{2}`, `YYYY-MM-DD HH:MM`)

	// Normalize trailing whitespace on each line to avoid width-related differences
	result = normalizeTrailingWhitespace(result)

	return result
}

func normalizePattern(s, pattern, replacement string) string {
	re := regexp.MustCompile(pattern)
	return re.ReplaceAllString(s, replacement)
}

func normalizeLengthPreserving(s, pattern, replacement string) string {
	re := regexp.MustCompile(pattern)
	return re.ReplaceAllStringFunc(s, func(match string) string {
		if len(replacement) >= len(match) {
			return replacement[:len(match)]
		}
		// Pad replacement to match original length
		return replacement + strings.Repeat(" ", len(match)-len(replacement))
	})
}

// normalizeStackResourceType normalizes the Stack resource type display.
// Pulumi sometimes returns the full type "pulumi:pulumi:Stack" and sometimes
// just the stack name (e.g., "simple-test"). This identifies stack resources
// by the pattern where type and name are identical and in project-stack format,
// then normalizes to the consistent "pulumi:pulumi:Stack" format.
func normalizeStackResourceType(s string) string {
	// Match patterns like "[op] word-word  word-word" where both words are the same
	// Op symbols: [ ], [+], [~], [-], [↻], [+-]
	re := regexp.MustCompile(`(\[[ +~\-↻]+\]) (\w+-\w+)  (\w+-\w+)`)
	return re.ReplaceAllStringFunc(s, func(match string) string {
		parts := re.FindStringSubmatch(match)
		if len(parts) == 4 {
			op := parts[1]
			typeStr := parts[2]
			name := parts[3]
			// If type and name are the same, it's a stack resource displayed with its name as type
			if typeStr == name {
				return op + " pulumi:pulumi:Stack  " + name
			}
		}
		return match
	})
}

// normalizeTrailingWhitespace removes trailing whitespace from each line
// to avoid differences caused by terminal width or padding variations.
func normalizeTrailingWhitespace(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	return strings.Join(lines, "\n")
}

func replaceAll(s, old, new string) string {
	result := s
	for {
		i := indexOf(result, old)
		if i < 0 {
			break
		}
		result = result[:i] + new + result[i+len(old):]
	}
	return result
}

func indexOf(s, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func (h *testHarness) WaitAndSnapshot(content string, name string, timeout time.Duration) {
	h.t.Helper()
	h.WaitFor(content, timeout)
	h.Snapshot(name)
}

func (h *testHarness) Quit(timeout time.Duration) {
	h.tm.Send(tea.Quit())
	h.tm.WaitFinished(h.t, teatest.WithFinalTimeout(timeout))
}

func (h *testHarness) QuitWithKey(timeout time.Duration) {
	h.tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	h.tm.WaitFinished(h.t, teatest.WithFinalTimeout(timeout))
}

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

func waitForContentWithCapture(t *testing.T, tm *teatest.TestModel, content string, timeout time.Duration) *outputCapture {
	t.Helper()
	oc := newOutputCapture(tm)
	teatest.WaitFor(t, oc,
		func(bts []byte) bool {
			return bytes.Contains(bts, []byte(content))
		},
		teatest.WithCheckInterval(50*time.Millisecond),
		teatest.WithDuration(timeout),
	)
	return oc
}

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

func waitForAnyContentWithCapture(t *testing.T, tm *teatest.TestModel, contents []string, timeout time.Duration) *outputCapture {
	t.Helper()
	oc := newOutputCapture(tm)
	teatest.WaitFor(t, oc,
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
	return oc
}

func takeSnapshot(t *testing.T, tm *teatest.TestModel, snapshotName string) {
	t.Helper()
	time.Sleep(100 * time.Millisecond)

	out, err := io.ReadAll(tm.Output())
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}

	t.Run(snapshotName, func(t *testing.T) {
		golden.RequireEqual(t, out)
	})
}

func takeSnapshotFromCapture(t *testing.T, oc *outputCapture, snapshotName string) {
	t.Helper()
	time.Sleep(100 * time.Millisecond)

	out := oc.AllOutput()

	t.Run(snapshotName, func(t *testing.T) {
		golden.RequireEqual(t, out)
	})
}

func waitAndSnapshot(t *testing.T, tm *teatest.TestModel, content string, snapshotName string, timeout time.Duration) {
	t.Helper()
	oc := waitForContentWithCapture(t, tm, content, timeout)
	takeSnapshotFromCapture(t, oc, snapshotName)
}

func SetupTestEnv(t *testing.T, fixture string, opts ...TestEnvOption) *TestEnvironment {
	t.Helper()

	fixturePath := filepath.Join("testdata", "fixtures", fixture)
	if _, err := os.Stat(fixturePath); os.IsNotExist(err) {
		t.Fatalf("fixture %s does not exist at %s", fixture, fixturePath)
	}

	workDir, err := os.MkdirTemp("", "p5-test-project-*")
	if err != nil {
		t.Fatalf("failed to create temp project dir: %v", err)
	}

	if err := copyDir(fixturePath, workDir); err != nil {
		os.RemoveAll(workDir)
		t.Fatalf("failed to copy fixture: %v", err)
	}

	// Use a fixed-length backend path to ensure consistent UI width in golden tests.
	// The path "/tmp/p5-test-backend-XXXXXX" is exactly 27 characters (without file://).
	// We create it within /tmp for a predictable base path length.
	backendDir, err := os.MkdirTemp("/tmp", "p5-test-backend-")
	if err != nil {
		os.RemoveAll(workDir)
		t.Fatalf("failed to create temp state dir: %v", err)
	}

	stackName := "test"

	env := map[string]string{
		"PULUMI_BACKEND_URL":       "file://" + backendDir,
		"PULUMI_CONFIG_PASSPHRASE": "test-passphrase-12345",
	}

	te := &TestEnvironment{
		t:          t,
		WorkDir:    workDir,
		BackendDir: backendDir,
		StackName:  stackName,
		Env:        env,
	}

	for _, opt := range opts {
		opt(te)
	}

	t.Cleanup(te.Cleanup)

	return te
}

func (te *TestEnvironment) Cleanup() {
	if te.cleaned {
		return
	}
	te.cleaned = true

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if te.StackName != "" {
		ws, err := auto.NewLocalWorkspace(ctx,
			auto.WorkDir(te.WorkDir),
			auto.EnvVars(te.Env),
		)
		if err == nil {
			stack, err := auto.SelectStack(ctx, te.StackName, ws)
			if err == nil {
				_, _ = stack.Destroy(ctx)
				_ = ws.RemoveStack(ctx, te.StackName)
			}
		}
	}

	os.RemoveAll(te.WorkDir)
	os.RemoveAll(te.BackendDir)
}

func (te *TestEnvironment) CreateStack(ctx context.Context) error {
	_, err := auto.NewStackLocalSource(ctx, te.StackName, te.WorkDir,
		auto.EnvVars(te.Env),
		auto.SecretsProvider("passphrase"),
	)
	return err
}

func (te *TestEnvironment) DeployStack(ctx context.Context) error {
	stack, err := auto.SelectStackLocalSource(ctx, te.StackName, te.WorkDir,
		auto.EnvVars(te.Env),
	)
	if err != nil {
		return fmt.Errorf("failed to select stack: %w", err)
	}

	_, err = stack.Up(ctx)
	if err != nil {
		return fmt.Errorf("failed to deploy stack: %w", err)
	}

	return nil
}

func (te *TestEnvironment) CreateModel(startView string) Model {
	deps := &Dependencies{
		StackOperator:    pulumi.NewStackOperator(),
		StackReader:      pulumi.NewStackReader(),
		WorkspaceReader:  pulumi.NewWorkspaceReader(),
		StackInitializer: pulumi.NewStackInitializer(),
		ResourceImporter: pulumi.NewResourceImporter(),
		PluginProvider:   &plugins.FakePluginProvider{},
		Env:              te.Env,
	}

	appCtx := AppContext{
		WorkDir:   te.WorkDir,
		StackName: te.StackName,
		StartView: startView,
	}

	return initialModel(context.Background(), appCtx, deps)
}

func (te *TestEnvironment) ReadOptions() pulumi.ReadOptions {
	return pulumi.ReadOptions{Env: te.Env}
}

func (te *TestEnvironment) OperationOptions() pulumi.OperationOptions {
	return pulumi.OperationOptions{Env: te.Env}
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

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

func testStacks() []pulumi.StackInfo {
	return []pulumi.StackInfo{
		{Name: "dev", Current: true},
		{Name: "staging", Current: false},
		{Name: "prod", Current: false},
	}
}

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

func makePreviewEvents(steps []pulumi.PreviewStep, finalError error) []pulumi.PreviewEvent {
	events := make([]pulumi.PreviewEvent, 0, len(steps)+1)
	for _, step := range steps {
		s := step
		events = append(events, pulumi.PreviewEvent{Step: &s})
	}
	events = append(events, pulumi.PreviewEvent{Done: true, Error: finalError})
	return events
}

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
	events = append(events, pulumi.OperationEvent{Done: true, Error: finalError})
	return events
}
