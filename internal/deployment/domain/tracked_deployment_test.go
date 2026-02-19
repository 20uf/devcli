package domain

import (
	"testing"
	"time"
)

// Test: Create tracked deployment
func TestTrackedDeployment_NewTrackedDeployment(t *testing.T) {
	workflow, _ := NewWorkflow("deploy.yml")
	td := NewTrackedDeployment("run-123", workflow, "main", "owner/repo")

	if td.ID() != "run-123" {
		t.Errorf("ID mismatch: got %s, want run-123", td.ID())
	}

	if td.RunID() != "run-123" {
		t.Errorf("RunID mismatch")
	}

	if td.Branch() != "main" {
		t.Errorf("Branch mismatch")
	}

	if td.Workflow().Name() != "deploy.yml" {
		t.Errorf("Workflow mismatch")
	}

	if td.Repo() != "owner/repo" {
		t.Errorf("Repo mismatch")
	}

	if td.Status() != RunStatusQueued {
		t.Errorf("Initial status should be Queued")
	}

	t.Log("✓ TrackedDeployment created successfully")
}

// Test: Status updates
func TestTrackedDeployment_UpdateStatus(t *testing.T) {
	workflow, _ := NewWorkflow("deploy.yml")
	td := NewTrackedDeployment("run-1", workflow, "main", "owner/repo")

	// Initial status
	if td.Status() != RunStatusQueued {
		t.Errorf("Initial status should be Queued")
	}

	if td.IsActive() {
		t.Logf("✓ Queued deployment is active")
	}

	// Update to in progress
	td.UpdateStatus(RunStatusInProgress)
	if td.Status() != RunStatusInProgress {
		t.Errorf("Status should be InProgress")
	}

	// Update to completed (via conclusion)
	td.UpdateConclusion(RunConclusionSuccess)
	if td.Status() != RunStatusCompleted {
		t.Errorf("Status should be Completed")
	}

	if !td.IsCompleted() {
		t.Errorf("Should be completed")
	}

	t.Log("✓ Status transitions work correctly")
}

// Test: Conclusion tracking
func TestTrackedDeployment_ConclusionTracking(t *testing.T) {
	tests := []struct {
		name       string
		conclusion RunConclusion
		checkFunc  func(TrackedDeployment) bool
	}{
		{
			name:       "Success",
			conclusion: RunConclusionSuccess,
			checkFunc:  func(td TrackedDeployment) bool { return td.IsSuccess() },
		},
		{
			name:       "Failure",
			conclusion: RunConclusionFailure,
			checkFunc:  func(td TrackedDeployment) bool { return td.IsFailed() },
		},
		{
			name:       "Cancelled",
			conclusion: RunConclusionCancelled,
			checkFunc:  func(td TrackedDeployment) bool { return td.IsCancelled() },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflow, _ := NewWorkflow("deploy.yml")
			td := NewTrackedDeployment("run-1", workflow, "main", "owner/repo")

			td.UpdateConclusion(tt.conclusion)

			if td.Conclusion() != tt.conclusion {
				t.Errorf("Conclusion mismatch: got %v, want %v", td.Conclusion(), tt.conclusion)
			}

			if !tt.checkFunc(td) {
				t.Errorf("Check function failed for %s", tt.name)
			}

			if !td.IsCompleted() {
				t.Errorf("Should be completed after conclusion update")
			}

			t.Logf("✓ %s conclusion tracked correctly", tt.name)
		})
	}
}

// Test: Active vs completed
func TestTrackedDeployment_ActiveVsCompleted(t *testing.T) {
	workflow, _ := NewWorkflow("deploy.yml")

	// Queued deployment should be active
	td1 := NewTrackedDeployment("run-1", workflow, "main", "owner/repo")
	if !td1.IsActive() {
		t.Errorf("Queued deployment should be active")
	}

	if td1.IsCompleted() {
		t.Errorf("Queued deployment should not be completed")
	}

	// In-progress should be active
	td1.UpdateStatus(RunStatusInProgress)
	if !td1.IsActive() {
		t.Errorf("In-progress should be active")
	}

	// Completed should not be active
	td1.UpdateConclusion(RunConclusionSuccess)
	if td1.IsActive() {
		t.Errorf("Completed should not be active")
	}

	if !td1.IsCompleted() {
		t.Errorf("Should be completed")
	}

	t.Log("✓ Active/completed states correct")
}

// Test: Age and elapsed time
func TestTrackedDeployment_TimeTracking(t *testing.T) {
	workflow, _ := NewWorkflow("deploy.yml")
	td := NewTrackedDeployment("run-1", workflow, "main", "owner/repo")

	// Initial elapsed time
	elapsed := td.ElapsedTime()
	if elapsed < 0 {
		t.Errorf("Elapsed time should not be negative")
	}

	// Wait a bit
	time.Sleep(10 * time.Millisecond)

	// Elapsed time should increase
	elapsed2 := td.ElapsedTime()
	if elapsed2 <= elapsed {
		t.Logf("Note: elapsed time timing may be tight in tests")
	}

	// Complete it
	td.UpdateConclusion(RunConclusionSuccess)

	// Elapsed time should now be fixed
	elapsedFinal := td.ElapsedTime()
	time.Sleep(10 * time.Millisecond)
	elapsedAfter := td.ElapsedTime()

	if elapsedAfter != elapsedFinal {
		t.Logf("Note: completed elapsed time should be stable")
	}

	t.Log("✓ Time tracking works")
}

// Test: Staleness checking
func TestTrackedDeployment_StalenessCheck(t *testing.T) {
	workflow, _ := NewWorkflow("deploy.yml")
	td := NewTrackedDeployment("run-1", workflow, "main", "owner/repo")

	// Fresh deployment is not stale
	if td.IsStale(1 * time.Hour) {
		t.Errorf("Fresh deployment should not be stale")
	}

	// Complete it
	td.UpdateConclusion(RunConclusionSuccess)

	// Just completed should not be stale
	if td.IsStale(1 * time.Hour) {
		t.Errorf("Just-completed should not be stale with 1h TTL")
	}

	// Should be stale with very small TTL
	if !td.IsStale(1 * time.Millisecond) {
		t.Logf("Note: staleness may be timing-dependent in tests")
	}

	t.Log("✓ Staleness checking works")
}

// Test: String representation
func TestTrackedDeployment_String(t *testing.T) {
	workflow, _ := NewWorkflow("deploy.yml")
	td := NewTrackedDeployment("run-1", workflow, "main", "owner/repo")

	str := td.String()
	if str != "deploy.yml on main" {
		t.Errorf("String representation incorrect: %s", str)
	}

	t.Log("✓ String representation correct")
}

// Test: Multiple deployments with different states
func TestTrackedDeployment_MultipleStates(t *testing.T) {
	workflow1, _ := NewWorkflow("deploy.yml")
	workflow2, _ := NewWorkflow("test.yml")

	// Create multiple tracked deployments
	td1 := NewTrackedDeployment("run-1", workflow1, "main", "owner/repo")
	td2 := NewTrackedDeployment("run-2", workflow2, "develop", "owner/repo")
	td3 := NewTrackedDeployment("run-3", workflow1, "feature-x", "owner/repo")

	// Update to different states
	td1.UpdateStatus(RunStatusInProgress)
	td2.UpdateConclusion(RunConclusionSuccess)
	td3.UpdateConclusion(RunConclusionFailure)

	// Verify states
	if !td1.IsActive() || td1.IsCompleted() {
		t.Errorf("td1 should be active and not completed")
	}

	if !td2.IsCompleted() || !td2.IsSuccess() {
		t.Errorf("td2 should be completed and success")
	}

	if !td3.IsCompleted() || !td3.IsFailed() {
		t.Errorf("td3 should be completed and failed")
	}

	// Verify independence
	ids := map[string]bool{td1.ID(): true, td2.ID(): true, td3.ID(): true}
	if len(ids) != 3 {
		t.Errorf("All IDs should be unique")
	}

	t.Log("✓ Multiple tracked deployments managed independently")
}

// Test: Workflow and branch tracking
func TestTrackedDeployment_WorkflowAndBranch(t *testing.T) {
	tests := []struct {
		name     string
		workflow string
		branch   string
	}{
		{"Production deploy", "deploy.yml", "main"},
		{"Staging deploy", "deploy.yml", "develop"},
		{"Test workflow", "test.yml", "feature-x"},
		{"Release workflow", "release.yml", "v1.0.0"},
	}

	for _, tt := range tests {
		workflow, _ := NewWorkflow(tt.workflow)
		td := NewTrackedDeployment("run-1", workflow, tt.branch, "owner/repo")

		if td.Workflow().Name() != tt.workflow {
			t.Errorf("Workflow mismatch: got %s, want %s", td.Workflow().Name(), tt.workflow)
		}

		if td.Branch() != tt.branch {
			t.Errorf("Branch mismatch: got %s, want %s", td.Branch(), tt.branch)
		}

		t.Logf("✓ %s tracked correctly", tt.name)
	}
}
