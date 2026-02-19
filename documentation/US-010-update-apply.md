# US-010: Apply Update

## Context

After detection of a newer version, the user accepts the update
and the system downloads and installs the new binary.

## Acceptance Criteria

- [ ] User confirms "Update to v0.11.0? (Y/n)"
- [ ] System downloads release from GitHub
- [ ] Replaces current binary
- [ ] Displays confirmation and version
- [ ] Next session uses new version
- [ ] Rollback possible on error

## Examples

```bash
$ devcli update
New version available: v0.11.0
Update to v0.11.0? (Y/n) y
  Downloading v0.11.0...
  ✓ Downloaded
  ✓ Installed
✓ Updated to v0.11.0!

$ devcli version
devcli v0.11.0
```

## User Journey

1. Version check performed (US-009)
2. User accepts update
3. Download binary from GitHub
4. Verify checksum (optional)
5. Replace executable
6. Confirmation

## Files Impacted

- CLI: `cmd/update.go`
- Infra: `internal/updater/updater.go` (Apply)
- Tests: `cmd/update_test.go`

## Technical Notes

- Download from GitHub releases assets
- Preserve permissions (~755 for executable)
- Backup old binary before replacement
- Atomicity: rename after complete success

