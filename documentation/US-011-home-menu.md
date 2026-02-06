# US-011: Home Menu

## Context

On startup without arguments, devcli displays an interactive menu that allows
navigation between main commands.

## Acceptance Criteria

- [ ] `devcli` without args → displays menu
- [ ] Options: connect, deploy, status, update, version
- [ ] Interactive selection with arrow keys
- [ ] Badge "in progress" on status if deployments active
- [ ] ESC → exit cleanly
- [ ] Return to menu after execution

## Example

```bash
$ devcli

 devcli v0.10.1
 Focus on coding, not tooling

Available Commands
▸ connect    Connect to an ECS container interactively
  deploy     Trigger a GitHub Actions deployment workflow
  status     Deployments in progress (2)
  update     Update devcli to the latest version
  version    Print version information

[Select command]
```

## User Journey

1. User runs `devcli`
2. System displays banner + menu
3. User selects command
4. Command executed
5. Return to menu
6. Or ESC to exit

## Files Impacted

- CLI: `cmd/root.go` (showHome)
- App: `cmd/root.go` (home menu loop)
- Tests: `cmd/root_test.go`

## Technical Notes

- "in progress" badge counts active deployments
- Update check in background (non-blocking)
- Tracker loaded to count active runs

