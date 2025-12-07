# Contributing

## Tests

### Types
- **Unit tests**: Standard `go test ./...`
- **Integration tests**: Require `-tags=integration` build tag
- **Golden file tests**: UI component snapshots in `testdata/*.golden`

### Running
```bash
go test ./...                      # Unit tests only
go test -tags=integration ./...    # All tests
./scripts/coverage.sh              # Coverage report
./scripts/coverage.sh -html        # Open HTML report
```

### Golden Files
Update snapshots with `-update` flag when UI changes are intentional.

## Tools
Native Go tools are defined in `go.mod` under the `tool` directive. Run with `go tool <name>`:
- `go tool svu` - semantic versioning
- `go tool goreleaser` - releases

Run `go generate ./...` for code generation (e.g., protobuf).

## Linting
```bash
go tool golangci-lint run ./...
go tool golangci-lint run --fix ./...  # Auto-fix issues
```

Configuration is in `.golangci.yml`.

## Commits
Use conventional commits (e.g., `feat:`, `fix:`, `docs:`, `test:`, `chore:`, `ci:`).
