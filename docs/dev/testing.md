# Testing

## Test Types

### Unit Tests
Standard Go tests without build tags.
```bash
go test ./...
```

### Integration Tests
Require `-tags=integration` build tag. Located in `cmd/p5/*_integration_test.go`.
```bash
go test -tags=integration ./...
```

Integration tests use `teatest` for TUI testing with mock Pulumi dependencies.

### Golden File Tests
UI component snapshots in `testdata/*.golden`. Used for visual regression testing of UI components.

```bash
# Run with update flag to regenerate snapshots
go test ./internal/ui -update
```

## Running Tests

```bash
go test ./...                      # Unit tests only
go test -tags=integration ./...    # All tests
./scripts/coverage.sh              # Coverage report
./scripts/coverage.sh -html        # Open HTML report
```

## Test Structure

### UI Component Tests
Located in `internal/ui/ui_test.go`. Test individual UI components (header, modals, lists) using golden file comparisons.

### Integration Test Helpers
`cmd/p5/integration_helpers_test.go` provides:
- `testModel()` - Creates model with fake dependencies
- `FakeStackOperator`, `FakeStackReader`, etc. - Mock implementations
- Golden file test utilities

### Fake Implementations
`internal/pulumi/fakes.go` provides mock implementations of all Pulumi interfaces for testing without actual Pulumi operations.

## Golden File Format

Golden files capture expected terminal output. File naming convention:
```
testdata/TestFunctionName_SubTest.golden
```

Update snapshots only when UI changes are intentional:
```bash
go test ./internal/ui -update -run TestHeader
```
