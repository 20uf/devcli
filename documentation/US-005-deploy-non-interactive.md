# US-005: Deploy Without Interactive Selection

## Context

User knows the workflow and branch; wants to trigger deployment directly
without going through interactive selection.

## Acceptance Criteria

- [ ] User passes `--workflow` + `--branch` + optional `--input`
- [ ] System shows **no prompts**
- [ ] Deployment triggers directly if valid
- [ ] Clear error if workflow doesn't exist or input is invalid
- [ ] Typed inputs validated at launch time (not after selection)

## Example

```bash
$ devcli deploy --workflow deploy.yml --branch main --input environment=prod
▶ Triggering deploy.yml on main
  environment: prod
✓ Workflow triggered: run 42
```

## User Journey

1. User executes command with all flags
2. DeployHandler detects all flags are present
3. Orchestrator validates inputs
4. Workflow triggered directly
5. Result displayed

## Files Impacted

- App: `internal/deployment/application/trigger_service.go` (ready)
- CLI: `cmd/deploy_handler.go` (Handle method)
- Tests: `cmd/deploy_handler_test.go` (NonInteractive_AllFlags)

## Constraints

- Missing inputs = error before trigger
- Invalid inputs = error before trigger
- No fallback to interactive mode

