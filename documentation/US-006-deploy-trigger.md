# US-006: Trigger GitHub Workflow

## Context

Once inputs are selected and validated, the system must send the request
to GitHub to execute the workflow on the specified branch.

## Acceptance Criteria

- [ ] User confirms parameters
- [ ] System sends GitHub request (via `gh workflow run`)
- [ ] Run ID received and stored
- [ ] Confirmation displayed with run number
- [ ] Deployment saved for replay
- [ ] GitHub errors propagated clearly

## Example

```bash
$ devcli deploy --workflow deploy.yml --branch main
▶ Triggering deploy.yml on main
  environment: prod
  skip_tests: false
✓ Workflow triggered: run 42
```

## User Journey

1. RunRepository.CreateRun() called
2. GitHub receives workflow_dispatch
3. Run created and returns run ID
4. Deployment persisted in history
5. Positive feedback to user

## Files Impacted

- Infra: `internal/deployment/infra/github_run_repository.go` (CreateRun)
- App: `internal/deployment/application/trigger_service.go` (orchestration)
- Tests: `internal/deployment/application/e2e_test.go` (E2E: Deploy)

## Technical Notes

- Wait 2s for run to appear in GitHub API
- Retrieve run ID via `gh run list --limit 1`
- New Run entity created with Queued status

