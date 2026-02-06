# US-008: Actions on Tracked Runs

## Context

Once a run is selected in the dashboard, the user can perform
actions to interact with the workflow (logs, browser, dismiss).

## Acceptance Criteria

- [ ] User selects a tracked run
- [ ] System displays action menu
- [ ] "Stream logs" → displays live logs (`gh run watch`)
- [ ] "View in browser" → opens run in browser
- [ ] "View full logs" → after completion
- [ ] "Dismiss" → stop tracking and remove from list
- [ ] Live logs remain until completion

## Examples

```bash
# Stream live
$ devcli status
[Select run]
[Select "Stream logs"]
✓ run #42 [in_progress]
[Live output...]
✓ Workflow completed

# Dismiss
$ devcli status
[Select run]
[Select "Dismiss"]
⊘ Run dismissed
```

## User Journey

1. User selects run in dashboard
2. Action menu displayed
3. Depending on action:
   - Stream: stays connected until completion
   - Browser: opens external tab
   - Dismiss: removes and returns to dashboard

## Files Impacted

- CLI: `cmd/status.go` (showRunActions)
- Infra: `internal/deployment/infra/github_run_repository.go` (GetRunLogs)
- Tests: `cmd/status_test.go` (integration)

## Constraints

- Timeout on stream if workflow > 1h
- Immediate dismiss, no confirmation
- Logs accessible 90 days after completion (GitHub limit)

