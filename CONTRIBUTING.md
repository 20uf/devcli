# Contributing to Devcli

## Contribution Workflow

Every feature or bug fix follows this process:

### 1. Create a User Story (US)

**File:** `.jira/US-XXX-short-description.md`

**Template:**

```markdown
# US-XXX: Feature Title (business language)

## Context
[Why this feature? What user need?]

## Acceptance Criteria
- [ ] AC1: First acceptance criterion (observable)
- [ ] AC2: Second acceptance criterion (observable)
- [ ] AC3: Edge case or constraint (observable)

## User Journey
[Main scenario with steps]

## Files Impacted
- Domain: path/to/file.go
- Infra: path/to/file.go
- Tests: path/to/*_test.go

## Technical Notes
[If complexity, implementation choices, dependencies]
```

### 2. TDD Implementation

```
RED   → Write failing tests
GREEN → Minimal code to pass tests
TEST  → Verify everything passes
CLEAN → Refactor if needed
DOCS  → Document in the US
```

### 3. Quality Checklist

- [ ] Tests created and passing
- [ ] Unnecessary comments removed
- [ ] Code in English, business logic documented in specs
- [ ] Interfaces/contracts documented (only comments needed)
- [ ] No dead code/temporary mocks
- [ ] Architecture respected (Domain → App → Infra → CLI)

### 4. Documentation Cleanup

**Keep:**
- Architecture diagrams
- Acceptance criteria
- User journeys
- Technical decisions

**Remove:**
- Status comments ("TODO", "WIP", "DONE")
- Exhaustive file listings
- Redundant summaries
- Session logs

### 5. Validation

```bash
# Tests
go test ./internal/... -v

# Build
go build ./cmd/...

# Linting
go fmt ./...
```

---

## Business Language

**Use for documentation:**

| Domain | Term | Example |
|--------|------|---------|
| Deployment | Orchestrate | "Orchestrate workflow selection" |
| Deployment | Type Safety | "Validate typed inputs (choice, boolean, string)" |
| Connection | Smart Selection | "Auto-select php container if available" |
| Status | Tracking | "Track workflow execution" |
| Storage | Persistence | "Persist runs as JSON" |

**Avoid:**
- Technical jargon without context
- Undefined acronyms
- Implementation details instead of business intent

---

## Minimal US Structure

### For small feature (< 1h)

```markdown
# US-XXX: Action

## Acceptance
- [ ] User can do X
- [ ] System displays Y
- [ ] Error Z is handled

## Example
$ devcli command --flag
✓ Result shown
```

### For medium feature (1-3h)

```markdown
# US-XXX: Feature

## Context
Why this feature.

## Acceptance
- [ ] AC1: happy path
- [ ] AC2: alternative path
- [ ] AC3: error handling

## User Journey
[Steps]

## Files
- Domain: entity.go
- Tests: entity_test.go
```

### For major feature (> 3h)

**Create multiple US instead of one large one.** Example:
- US-XXX-01: Domain entity
- US-XXX-02: Repository interface
- US-XXX-03: Implementation
- US-XXX-04: Integration tests

---

## Identifying Missing US

Ask these questions:

1. **Does each use case have a US?**
   - Connect interactive ✅ (US-001)
   - Connect flags ✅ (US-002)
   - Connect auto-select ✅ (US-003)
   - Deploy interactive ✅ (US-004)
   - Deploy flags ❌ **MISSING**
   - Deploy inputs ✅ (US-004)
   - Status dashboard ✅ (US-007)
   - Status actions ✅ (US-008)
   - Update check ❌ **MISSING**
   - Update apply ❌ **MISSING**

2. **Does each error/edge case have acceptance criteria?**

3. **Are non-obvious decisions documented?**

---

## Code Cleanup

### Comments to KEEP

```go
// RunStatus represents the lifecycle state of a deployment run.
type RunStatus string

// IsActive checks if the deployment is in-progress or queued.
func (td TrackedDeployment) IsActive() bool
```

### Comments to REMOVE

```go
// Get the run (❌ Obvious)
run, _ := ...

// TODO: implement this (❌ Not done, create a US)
// Example: (❌ Example code commented out)
// Step 1: init (❌ Too verbose)
```

### Recommended Pattern

```go
// Interface to abstract persistence of tracked runs.
// Allows changing storage (file, DB, remote) without touching domain.
type TrackerRepository interface {
    Save(ctx context.Context, td TrackedDeployment) error
    List(ctx context.Context) ([]TrackedDeployment, error)
}
```

---

## Session Process

**Before coding, each session:**

1. ✅ **Identify missing US**
2. ✅ **Create US** (business model)
3. ✅ **Plan order** (dependencies)
4. ✅ **Code with TDD**
5. ✅ **Clean code + docs**
6. ✅ **Validate checklist**

**Before ending a session:**

```bash
# Verify
go test ./internal/... -v     # Tests
go fmt ./...                   # Format
go build ./cmd/...             # Build

# Documentation
# - Update existing US
# - Create missing US
# - Remove unnecessary files
# - Validate business language
```

---

## Complete Example

### Before Contribution

**State:** Feature missing
**Need:** Deploy without listing workflows

### Step 1: Create US

```markdown
# US-005: Deploy with specific workflow

## Context
User knows the workflow name and wants to deploy directly
without interactive selection.

## Acceptance
- [ ] User passes --workflow=deploy.yml
- [ ] System does not show workflow list
- [ ] Deployment triggers directly
- [ ] Error if workflow does not exist

## Example
$ devcli deploy --workflow deploy.yml --branch main
▶ Deploying deploy.yml on main
✓ Triggered: run #42
```

### Step 2: Implement with TDD

```go
// Test fails first
func TestDeploy_NonInteractive_WorkflowFlag(t *testing.T) {
    handler, _ := NewDeployHandler(ctx, "owner/repo")
    err := handler.Handle(cmd, "deploy.yml", "main", nil, false)
    // Assertion: no prompts, direct deployment
}

// Implement minimal code
func (h *DeployHandler) Handle(..., workflow string, ...) error {
    if workflow != "" {
        return h.triggerDirect(workflow)
    }
    return h.interactiveFlow()
}
```

### Step 3: Clean up

- ✅ Remove "// TODO" comments
- ✅ Verify business language in docs
- ✅ No temporary mocks in production code

### Step 4: Validate

```bash
go test ./cmd -run Deploy -v   # ✅ Passes
go build ./cmd/...              # ✅ Compiles
```

### Step 5: Update documentation

- Add AC to US-005
- Remove redundant summaries
- Update FEATURES_REGISTRY

---

## Frequently Asked Questions

**Q: How many US per session?**
A: As many as are complete (code + tests + docs). Usually 2-4.

**Q: When to create a sub-US?**
A: If effort > 2-3h. Split by dependencies or context.

**Q: How to name US?**
A: `US-NNN-verb-noun` in business language.
Example: `US-006-trigger-workflow-with-inputs`

**Q: Temporary mocks in tests?**
A: Keep in `*_test.go` or `infra/mock_*.go`, not in production.

---

## Final Checklist

Before finishing:

- [ ] All US created (business language)
- [ ] Tests written and passing
- [ ] Code cleaned (comments, imports)
- [ ] Documentation in business language
- [ ] Build + tests pass
- [ ] No unnecessary tracking files
- [ ] Ready for production or next session

---

**Established procedure for every contribution.** 🎯
