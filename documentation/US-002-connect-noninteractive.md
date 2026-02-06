# US-002: Connect to ECS Container (Non-Interactive Mode)

**Status:** ✅ WORKING (integrated, needs handler tests)
**Priority:** P1
**Effort:** Already complete

---

## User Story

**As a** CI/CD pipeline or script
**I want to** connect to a container using command-line flags
**So that** I can automate container access without interactive prompts

---

## Acceptance Criteria

### AC1: All Flags Provided
```gherkin
Given all flags are provided: --cluster, --service, --container, --shell
When system validates inputs
Then connection proceeds without any UI prompts
And no history menu is shown
```

### AC2: Flag Validation
```gherkin
Given flags are provided
When system validates
Then invalid cluster/service/container returns error
And error occurs before connection attempt
```

### AC3: Partial Flags
```gherkin
Given only --cluster flag provided
When --service and --container are missing
Then system prompts for missing selections interactively
```

---

## Command Examples

```bash
# Non-interactive: all flags
devcli connect --cluster prod --service api --container php --shell bash

# Partial: cluster + service, select container interactively
devcli connect --cluster prod --service api

# Full flags from history replay
devcli connect --cluster $(last-cluster) --service $(last-service)
```

---

## Test Cases

- [ ] All flags provided → direct connection
- [ ] Invalid cluster → error
- [ ] Partial flags → interactive for missing
- [ ] Flag parsing errors → clear error message

---

## Files Impacted

- `cmd/connect.go` (flag parsing)
- `cmd/connect_handler.go` (Handle method logic)
- Tests: `cmd/connect_handler_test.go` (integration)

---

## Implementation Status

✅ Connected to domain (ConnectOrchestrator)
⏳ Handler tests needed (1/7 test)

## See Also

- [US-001-connect-interactive.md](US-001-connect-interactive.md) - Interactive mode
- [US-003-connect-container-autoselect.md](US-003-connect-container-autoselect.md) - Auto-select logic
- [IMPLEMENTATION_PLAN.md](IMPLEMENTATION_PLAN.md) - Phase 2: Handler tests
