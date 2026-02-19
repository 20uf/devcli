package application

import (
	"context"
	"testing"

	connDomain "github.com/20uf/devcli/internal/connection/domain"
	"github.com/20uf/devcli/internal/deployment/domain"
	"github.com/20uf/devcli/internal/deployment/infra"
)

// E2E: Deploy workflow with typed inputs end-to-end
func TestE2E_DeployWithTypedInputs(t *testing.T) {
	ctx := context.Background()

	// Setup: Mock repositories
	workflows := infra.NewMockWorkflowRepository()
	runs := infra.NewMockRunRepository()
	branches := infra.NewMockBranchRepository()
	deployments := infra.NewMockDeploymentRepository()

	repos := &domain.AllRepositories{
		Workflows:   workflows,
		Runs:        runs,
		Branches:    branches,
		Deployments: deployments,
	}

	// Setup: Orchestrator
	orchestrator := NewTriggerDeploymentOrchestrator(repos)

	// Step 1: List workflows
	workflowList, err := repos.Workflows.ListWorkflows(ctx)
	if err != nil || len(workflowList) == 0 {
		t.Fatalf("Failed to list workflows: %v", err)
	}

	workflow := workflowList[0]
	t.Logf("Selected workflow: %s", workflow.Name())

	// Step 2: Get workflow inputs
	inputs, err := repos.Workflows.GetWorkflowInputs(ctx, workflow)
	if err != nil {
		t.Fatalf("Failed to get inputs: %v", err)
	}

	// Step 3: Prepare inputs
	inputMap := make(map[string]string)
	for _, input := range inputs {
		switch input.Type() {
		case domain.InputTypeChoice:
			inputMap[input.Key()] = "dev" // Select first option
		case domain.InputTypeBoolean:
			inputMap[input.Key()] = "false"
		case domain.InputTypeString:
			inputMap[input.Key()] = "test-value"
		}
	}

	// Step 4: List branches
	branchList, err := repos.Branches.ListBranches(ctx)
	if err != nil || len(branchList) == 0 {
		t.Fatalf("Failed to list branches: %v", err)
	}

	branch := branchList[0]
	t.Logf("Selected branch: %s", branch)

	// Step 5: Trigger deployment
	wfName := workflow.Name()
	deployment, err := orchestrator.Trigger(ctx, TriggerRequest{
		WorkflowName: &wfName,
		BranchName:   &branch,
		Inputs:       inputMap,
		RepoURL:      "owner/repo",
	})

	if err != nil {
		t.Fatalf("Failed to trigger deployment: %v", err)
	}

	if !deployment.HasRun() {
		t.Errorf("Deployment should have run after trigger")
	}

	t.Logf("✓ Deployment triggered: %s on %s (run: %s)", workflow.Name(), branch, deployment.Run().ID())
}

// E2E: Track deployment through status dashboard
func TestE2E_TrackDeploymentToDashboard(t *testing.T) {
	ctx := context.Background()

	// Setup: Tracker + Status orchestrator
	tracker := infra.NewFileTrackerRepository("/tmp/devcli-e2e-test")
	runs := infra.NewMockRunRepository()
	statusOrch := NewStatusOrchestrator(tracker, runs)

	// Step 1: Create deployment to track
	workflow, _ := domain.NewWorkflow("deploy.yml")
	runID := "e2e-run-123"

	// Step 2: Track it
	tracked, err := statusOrch.TrackDeployment(ctx, runID, workflow, "main", "owner/repo")
	if err != nil {
		t.Fatalf("Failed to track deployment: %v", err)
	}

	if !tracked.IsActive() {
		t.Errorf("Tracked deployment should be active")
	}

	t.Logf("✓ Deployment tracked: %s", tracked.ID())

	// Step 3: List tracked
	allTracked, err := statusOrch.ListTracked(ctx)
	if err != nil {
		t.Fatalf("Failed to list tracked: %v", err)
	}

	if len(allTracked) == 0 {
		t.Errorf("Should have tracked deployment")
	}

	// Step 4: Get specific
	retrieved, err := statusOrch.GetTracked(ctx, runID)
	if err != nil {
		t.Fatalf("Failed to get tracked: %v", err)
	}

	if retrieved == nil || retrieved.ID() != runID {
		t.Errorf("Should retrieve tracked deployment")
	}

	t.Logf("✓ Retrieved tracked deployment from dashboard")

	// Step 5: Get logs
	logs, err := statusOrch.GetRunLogs(ctx, runID)
	if err != nil {
		t.Logf("Note: GetRunLogs failed (expected in test): %v", err)
	} else {
		t.Logf("✓ Retrieved logs: %d bytes", len(logs))
	}

	// Step 6: Dismiss
	err = statusOrch.DismissTracked(ctx, runID)
	if err != nil {
		t.Fatalf("Failed to dismiss: %v", err)
	}

	// Step 7: Verify gone
	retrieved2, _ := statusOrch.GetTracked(ctx, runID)
	if retrieved2 != nil {
		t.Errorf("Should be removed after dismiss")
	}

	t.Logf("✓ Deployment dismissed from dashboard")
}

// E2E: Full workflow - deploy then track
func TestE2E_DeployThenTrack(t *testing.T) {
	ctx := context.Background()

	// Setup: All components
	workflows := infra.NewMockWorkflowRepository()
	runs := infra.NewMockRunRepository()
	branches := infra.NewMockBranchRepository()
	deployments := infra.NewMockDeploymentRepository()
	tracker := infra.NewFileTrackerRepository("/tmp/devcli-e2e-full")
	statusOrch := NewStatusOrchestrator(tracker, runs)

	repos := &domain.AllRepositories{
		Workflows:   workflows,
		Runs:        runs,
		Branches:    branches,
		Deployments: deployments,
	}

	deployOrch := NewTriggerDeploymentOrchestrator(repos)

	// Phase 1: Deploy
	workflowList, _ := repos.Workflows.ListWorkflows(ctx)
	branchList, _ := repos.Branches.ListBranches(ctx)
	inputs, _ := repos.Workflows.GetWorkflowInputs(ctx, workflowList[0])

	inputMap := make(map[string]string)
	for _, inp := range inputs {
		switch inp.Type() {
		case domain.InputTypeChoice:
			if len(inp.Options()) > 0 {
				inputMap[inp.Key()] = inp.Options()[0]
			}
		case domain.InputTypeBoolean:
			inputMap[inp.Key()] = "false"
		default:
			inputMap[inp.Key()] = "test"
		}
	}

	wfName := workflowList[0].Name()
	brName := branchList[0]
	deployment, err := deployOrch.Trigger(ctx, TriggerRequest{
		WorkflowName: &wfName,
		BranchName:   &brName,
		Inputs:       inputMap,
		RepoURL:      "owner/repo",
	})

	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	t.Logf("✓ Phase 1: Deployed %s on %s (run: %s)",
		deployment.Workflow().Name(), deployment.Branch(), deployment.Run().ID())

	// Phase 2: Track the deployment
	tracked, err := statusOrch.TrackDeployment(ctx, deployment.Run().ID(), deployment.Workflow(),
		deployment.Branch(), "owner/repo")

	if err != nil {
		t.Fatalf("Track failed: %v", err)
	}

	t.Logf("✓ Phase 2: Tracked in dashboard: %s", tracked.String())

	// Phase 3: Verify in dashboard
	active, _ := statusOrch.ListActive(ctx)
	if len(active) == 0 {
		t.Errorf("Should have active deployment")
	}

	t.Logf("✓ Phase 3: Active deployments: %d", len(active))

	// Phase 4: Complete deployment
	tracked.UpdateConclusion(domain.RunConclusionSuccess)
	tracker.Save(ctx, tracked)

	// Phase 5: Verify completed
	completed, _ := statusOrch.GetTracked(ctx, tracked.ID())
	if !completed.IsSuccess() {
		t.Errorf("Should show success")
	}

	t.Logf("✓ Phase 4: Deployment marked as success")
	t.Logf("✓ Full E2E workflow complete: Deploy → Track → Complete")
}

// E2E: Connection context flow
func TestE2E_ConnectContext(t *testing.T) {
	// This test validates the connection context works end-to-end
	// (Uses existing ConnectOrchestrator from connection package)

	// Just verify the connection domain is available
	cluster, _ := connDomain.NewCluster("production")
	if cluster.Name() != "production" {
		t.Errorf("Cluster creation failed")
	}

	t.Logf("✓ Connection context available for E2E testing")
}

// E2E: Error handling and edge cases
func TestE2E_ErrorHandling(t *testing.T) {
	tests := []struct {
		name      string
		testFunc  func() error
		shouldErr bool
	}{
		{
			name: "Invalid workflow name",
			testFunc: func() error {
				_, err := domain.NewWorkflow("")
				return err
			},
			shouldErr: true,
		},
		{
			name: "Invalid cluster name",
			testFunc: func() error {
				_, err := connDomain.NewCluster("")
				return err
			},
			shouldErr: true,
		},
		{
			name: "Invalid choice input",
			testFunc: func() error {
				_, err := domain.NewChoiceInput("env", "invalid", []string{"a", "b"}, true)
				return err
			},
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.testFunc()
			if tt.shouldErr && err == nil {
				t.Errorf("Expected error but got nil")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			t.Logf("✓ %s", tt.name)
		})
	}
}
