package application

import (
	"context"
	"testing"

	"github.com/20uf/devcli/internal/deployment/domain"
)

// Mock tracker for testing
type mockTracker struct {
	tracked map[string]domain.TrackedDeployment
}

func newMockTracker() *mockTracker {
	return &mockTracker{
		tracked: make(map[string]domain.TrackedDeployment),
	}
}

func (m *mockTracker) Save(ctx context.Context, td domain.TrackedDeployment) error {
	m.tracked[td.ID()] = td
	return nil
}

func (m *mockTracker) List(ctx context.Context) ([]domain.TrackedDeployment, error) {
	var result []domain.TrackedDeployment
	for _, td := range m.tracked {
		result = append(result, td)
	}
	return result, nil
}

func (m *mockTracker) GetByID(ctx context.Context, id string) (*domain.TrackedDeployment, error) {
	if td, ok := m.tracked[id]; ok {
		return &td, nil
	}
	return nil, nil
}

func (m *mockTracker) Remove(ctx context.Context, id string) error {
	delete(m.tracked, id)
	return nil
}

func (m *mockTracker) ListActive(ctx context.Context) ([]domain.TrackedDeployment, error) {
	var active []domain.TrackedDeployment
	for _, td := range m.tracked {
		if td.IsActive() {
			active = append(active, td)
		}
	}
	return active, nil
}

func (m *mockTracker) Cleanup(ctx context.Context, maxAge int64) (int, error) {
	// For testing, just remove everything
	removed := len(m.tracked)
	m.tracked = make(map[string]domain.TrackedDeployment)
	return removed, nil
}

// Mock run repository for testing
type mockRunRepo struct{}

func (m *mockRunRepo) CreateRun(ctx context.Context, deployment domain.Deployment) (*domain.Run, error) {
	run := domain.NewRun("run-1", 1, domain.RunStatusQueued, "main", "https://github.com")
	return &run, nil
}

func (m *mockRunRepo) GetRun(ctx context.Context, runID string) (*domain.Run, error) {
	run := domain.NewRun(runID, 1, domain.RunStatusInProgress, "main", "https://github.com")
	return &run, nil
}

func (m *mockRunRepo) UpdateRunStatus(ctx context.Context, runID string, status domain.RunStatus) error {
	return nil
}

func (m *mockRunRepo) UpdateRunConclusion(ctx context.Context, runID string, conclusion domain.RunConclusion) error {
	return nil
}

func (m *mockRunRepo) GetRunLogs(ctx context.Context, runID string) (string, error) {
	return "Sample logs", nil
}

// Test: StatusOrchestrator initialization
func TestStatusOrchestrator_Init(t *testing.T) {
	tracker := newMockTracker()
	runs := &mockRunRepo{}
	orchestrator := NewStatusOrchestrator(tracker, runs)

	if orchestrator == nil {
		t.Errorf("Orchestrator should not be nil")
	}

	t.Log("✓ StatusOrchestrator initialized")
}

// Test: Track deployment
func TestStatusOrchestrator_TrackDeployment(t *testing.T) {
	ctx := context.Background()
	tracker := newMockTracker()
	runs := &mockRunRepo{}
	orchestrator := NewStatusOrchestrator(tracker, runs)

	workflow, _ := domain.NewWorkflow("deploy.yml")
	td, err := orchestrator.TrackDeployment(ctx, "run-1", workflow, "main", "owner/repo")

	if err != nil {
		t.Errorf("Failed to track deployment: %v", err)
	}

	if td.ID() != "run-1" {
		t.Errorf("Deployment ID mismatch")
	}

	if !td.IsActive() {
		t.Errorf("New deployment should be active")
	}

	t.Log("✓ Deployment tracked successfully")
}

// Test: List tracked deployments
func TestStatusOrchestrator_ListTracked(t *testing.T) {
	ctx := context.Background()
	tracker := newMockTracker()
	runs := &mockRunRepo{}
	orchestrator := NewStatusOrchestrator(tracker, runs)

	workflow, _ := domain.NewWorkflow("deploy.yml")

	// Track multiple deployments
	_, _ = orchestrator.TrackDeployment(ctx, "run-1", workflow, "main", "owner/repo")
	_, _ = orchestrator.TrackDeployment(ctx, "run-2", workflow, "develop", "owner/repo")

	// List all
	tracked, err := orchestrator.ListTracked(ctx)

	if err != nil {
		t.Errorf("Failed to list tracked: %v", err)
	}

	if len(tracked) != 2 {
		t.Errorf("Expected 2 tracked deployments, got %d", len(tracked))
	}

	t.Log("✓ Listed tracked deployments")
}

// Test: List active deployments
func TestStatusOrchestrator_ListActive(t *testing.T) {
	ctx := context.Background()
	tracker := newMockTracker()
	runs := &mockRunRepo{}
	orchestrator := NewStatusOrchestrator(tracker, runs)

	workflow, _ := domain.NewWorkflow("deploy.yml")

	// Track and complete one deployment
	td1, _ := orchestrator.TrackDeployment(ctx, "run-1", workflow, "main", "owner/repo")
	_, _ = orchestrator.TrackDeployment(ctx, "run-2", workflow, "develop", "owner/repo")

	// Complete one
	td1.UpdateConclusion(domain.RunConclusionSuccess)
	_ = tracker.Save(ctx, td1)

	// List active
	active, err := orchestrator.ListActive(ctx)

	if err != nil {
		t.Errorf("Failed to list active: %v", err)
	}

	if len(active) != 1 {
		t.Errorf("Expected 1 active deployment, got %d", len(active))
	}

	if active[0].ID() != "run-2" {
		t.Errorf("Wrong deployment is active")
	}

	t.Log("✓ Listed active deployments")
}

// Test: Get tracked deployment
func TestStatusOrchestrator_GetTracked(t *testing.T) {
	ctx := context.Background()
	tracker := newMockTracker()
	runs := &mockRunRepo{}
	orchestrator := NewStatusOrchestrator(tracker, runs)

	workflow, _ := domain.NewWorkflow("deploy.yml")
	_, _ = orchestrator.TrackDeployment(ctx, "run-1", workflow, "main", "owner/repo")

	td, err := orchestrator.GetTracked(ctx, "run-1")

	if err != nil {
		t.Errorf("Failed to get tracked: %v", err)
	}

	if td == nil {
		t.Fatalf("Tracked deployment should not be nil")
	}

	if td.ID() != "run-1" {
		t.Errorf("ID mismatch")
	}

	t.Log("✓ Got tracked deployment")
}

// Test: Dismiss tracked deployment
func TestStatusOrchestrator_DismissTracked(t *testing.T) {
	ctx := context.Background()
	tracker := newMockTracker()
	runs := &mockRunRepo{}
	orchestrator := NewStatusOrchestrator(tracker, runs)

	workflow, _ := domain.NewWorkflow("deploy.yml")
	_, _ = orchestrator.TrackDeployment(ctx, "run-1", workflow, "main", "owner/repo")

	// Dismiss it
	err := orchestrator.DismissTracked(ctx, "run-1")
	if err != nil {
		t.Errorf("Failed to dismiss: %v", err)
	}

	// Verify it's gone
	td, _ := orchestrator.GetTracked(ctx, "run-1")
	if td != nil {
		t.Errorf("Deployment should be removed")
	}

	t.Log("✓ Dismissed tracked deployment")
}

// Test: Get run logs
func TestStatusOrchestrator_GetRunLogs(t *testing.T) {
	ctx := context.Background()
	tracker := newMockTracker()
	runs := &mockRunRepo{}
	orchestrator := NewStatusOrchestrator(tracker, runs)

	logs, err := orchestrator.GetRunLogs(ctx, "run-1")

	if err != nil {
		t.Errorf("Failed to get logs: %v", err)
	}

	if logs != "Sample logs" {
		t.Errorf("Logs mismatch: %s", logs)
	}

	t.Log("✓ Got run logs")
}

// Test: Deployment lifecycle
func TestStatusOrchestrator_DeploymentLifecycle(t *testing.T) {
	ctx := context.Background()
	tracker := newMockTracker()
	runs := &mockRunRepo{}
	orchestrator := NewStatusOrchestrator(tracker, runs)

	workflow, _ := domain.NewWorkflow("deploy.yml")

	// 1. Track
	td, _ := orchestrator.TrackDeployment(ctx, "run-1", workflow, "main", "owner/repo")
	if !td.IsActive() {
		t.Errorf("Should be active after tracking")
	}

	// 2. Get
	td2, _ := orchestrator.GetTracked(ctx, "run-1")
	if td2.ID() != "run-1" {
		t.Errorf("Should retrieve tracked deployment")
	}

	// 3. Complete it
	td2.UpdateConclusion(domain.RunConclusionSuccess)
	_ = tracker.Save(ctx, *td2)

	// 4. Verify not in active
	active, _ := orchestrator.ListActive(ctx)
	if len(active) > 0 {
		t.Errorf("Completed should not be active")
	}

	// 5. Dismiss
	_ = orchestrator.DismissTracked(ctx, "run-1")

	// 6. Verify gone
	td3, _ := orchestrator.GetTracked(ctx, "run-1")
	if td3 != nil {
		t.Errorf("Should be removed after dismiss")
	}

	t.Log("✓ Deployment lifecycle complete")
}
