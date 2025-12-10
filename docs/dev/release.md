# Release

## Tools

Native Go tools defined in `go.mod`:
```bash
go tool svu          # Semantic versioning
go tool goreleaser   # Build and release
```

## Versioning

Uses semantic versioning via `svu`:
```bash
go tool svu next     # Suggest next version
go tool svu patch    # Next patch version
go tool svu minor    # Next minor version
go tool svu major    # Next major version
```

## Release Process

1. Ensure all tests pass:
   ```bash
   go test -tags=integration ./...
   ```

2. Update CHANGELOG.md with release notes

3. Tag release:
   ```bash
   git tag v$(go tool svu next)
   git push --tags
   ```

4. GoReleaser handles the rest via CI:
   - Builds binaries for multiple platforms
   - Creates GitHub release
   - Publishes release notes

## Configuration

- `.goreleaser.yaml` - GoReleaser configuration
- `CHANGELOG.md` - Release notes

## Scripts

- `./scripts/release.sh` - Release automation script
- `./scripts/vhs.sh` - Demo GIF generation for releases

## Demo Recording

The demo GIF is recorded using VHS:
```bash
./scripts/vhs.sh              # Full build and record
./scripts/vhs.sh --build-only # Build Docker image
./scripts/vhs.sh --run-only   # Record (image must exist)
```

Configuration in `demo.tape`.
