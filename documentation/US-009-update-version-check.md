# US-009: Check Available Updates

## Context

User wants to know if there's a newer version of devcli
available on GitHub releases.

## Acceptance Criteria

- [ ] Command `devcli update` checks GitHub releases
- [ ] Compares local vs remote
- [ ] Displays available version if newer
- [ ] Shows "already up to date" otherwise
- [ ] Support `--pre-release` flag for alpha/beta
- [ ] Graceful timeout if GitHub inaccessible

## Examples

```bash
# Up to date
$ devcli update
Already up to date (v0.10.1)

# Update available
$ devcli update
New version available: v0.11.0 (current: v0.10.1)
Update to v0.11.0? (Y/n)

# With pre-release
$ devcli update --pre-release
New version available: v0.11.0-beta.1
```

## User Journey

1. User runs `devcli update`
2. System calls GitHub API for latest release
3. Compares versions (semantic versioning)
4. Displays result

## Files Impacted

- CLI: `cmd/update.go`
- Infra: `internal/updater/updater.go` (Check)
- Tests: `cmd/update_test.go`

## Technical Notes

- GitHub releases API: `GET /repos/{owner}/{repo}/releases/latest`
- Pre-release flag includes versions ~-alpha, -beta, -rc
- Semantics: v1.2.3 > v1.2.2

