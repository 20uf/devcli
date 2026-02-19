package application

import (
	"context"
	"testing"

	"github.com/20uf/devcli/internal/deployment/domain"
)

// Mocks for testing

type MockWorkflowRepository struct {
	workflows []domain.Workflow
	inputs    map[string][]domain.Input
	err       error
}

func (m *MockWorkflowRepository) ListWorkflows(ctx context.Context) ([]domain.Workflow, error) {
	return m.workflows, m.err
}

func (m *MockWorkflowRepository) GetWorkflow(ctx context.Context, name string) (*domain.Workflow, error) {
	for _, w := range m.workflows {
		if w.Name() == name {
			return &w, nil
		}
	}
	return nil, domain.ErrWorkflowNotFound
}

func (m *MockWorkflowRepository) GetWorkflowInputs(ctx context.Context, workflow domain.Workflow) ([]domain.Input, error) {
	if inputs, ok := m.inputs[workflow.Name()]; ok {
		return inputs, nil
	}
	return []domain.Input{}, nil
}

type MockRunRepository struct {
	runs map[string]domain.Run
	err  error
}

func (m *MockRunRepository) CreateRun(ctx context.Context, deployment domain.Deployment) (*domain.Run, error) {
	run := domain.NewRun("run-123", 42, domain.RunStatusQueued, deployment.Branch(), "https://github.com/example")
	m.runs["run-123"] = run
	return &run, m.err
}

func (m *MockRunRepository) GetRun(ctx context.Context, runID string) (*domain.Run, error) {
	if run, ok := m.runs[runID]; ok {
		return &run, nil
	}
	return nil, domain.ErrNoRunFound
}

func (m *MockRunRepository) UpdateRunStatus(ctx context.Context, runID string, status domain.RunStatus) error {
	return nil
}

func (m *MockRunRepository) UpdateRunConclusion(ctx context.Context, runID string, conclusion domain.RunConclusion) error {
	return nil
}

func (m *MockRunRepository) GetRunLogs(ctx context.Context, runID string) (string, error) {
	return "logs...", nil
}

type MockBranchRepository struct {
	branches      []string
	defaultBranch string
	err           error
}

func (m *MockBranchRepository) ListBranches(ctx context.Context) ([]string, error) {
	return m.branches, m.err
}

func (m *MockBranchRepository) GetDefaultBranch(ctx context.Context) (string, error) {
	if m.defaultBranch != "" {
		return m.defaultBranch, nil
	}
	return "main", m.err
}

type MockDeploymentRepository struct {
	deployments []*domain.Deployment
	err         error
}

func (m *MockDeploymentRepository) Save(ctx context.Context, deployment domain.Deployment) error {
	m.deployments = append(m.deployments, &deployment)
	return m.err
}

func (m *MockDeploymentRepository) FindByID(ctx context.Context, id string) (*domain.Deployment, error) {
	return nil, nil
}

func (m *MockDeploymentRepository) FindRecent(ctx context.Context, limit int) ([]domain.Deployment, error) {
	return []domain.Deployment{}, nil
}

// Tests

func TestTriggerDeploymentOrchestrator_Trigger_Success(t *testing.T) {
	// Arrange
	workflow, _ := domain.NewWorkflow("deploy.yml")
	repos := &domain.AllRepositories{
		Workflows: &MockWorkflowRepository{
			workflows: []domain.Workflow{workflow},
			inputs: map[string][]domain.Input{
				"deploy.yml": {},
			},
		},
		Branches: &MockBranchRepository{
			defaultBranch: "main",
		},
		Runs: &MockRunRepository{
			runs: make(map[string]domain.Run),
		},
		Deployments: &MockDeploymentRepository{},
	}

	orchestrator := NewTriggerDeploymentOrchestrator(repos)
	ctx := context.Background()

	// Act
	deployment, err := orchestrator.Trigger(ctx, TriggerRequest{
		RepoURL: "https://github.com/example/repo",
	})

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if deployment.Workflow().Name() != "deploy.yml" {
		t.Errorf("expected workflow 'deploy.yml', got '%s'", deployment.Workflow().Name())
	}

	if deployment.Branch() != "main" {
		t.Errorf("expected branch 'main', got '%s'", deployment.Branch())
	}

	if !deployment.HasRun() {
		t.Errorf("deployment should have a run")
	}
}

func TestTriggerDeploymentOrchestrator_SelectWorkflow_Explicit(t *testing.T) {
	// Arrange
	workflow, _ := domain.NewWorkflow("deploy.yml")
	repos := &domain.AllRepositories{
		Workflows: &MockWorkflowRepository{
			workflows: []domain.Workflow{workflow},
		},
	}
	orchestrator := NewTriggerDeploymentOrchestrator(repos)

	// Act
	name := "deploy.yml"
	selected, err := orchestrator.SelectWorkflow(context.Background(), SelectWorkflowRequest{
		WorkflowName: &name,
	})

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if selected.Name() != "deploy.yml" {
		t.Errorf("expected 'deploy.yml', got '%s'", selected.Name())
	}
}

func TestTriggerDeploymentOrchestrator_GetWorkflowInputs_WithTypedInputs(t *testing.T) {
	// Arrange
	workflow, _ := domain.NewWorkflow("deploy.yml")

	// Create typed inputs
	envInput, _ := domain.NewChoiceInput("environment", "dev", []string{"dev", "staging", "prod"}, true)
	flakyInput, _ := domain.NewInput("enable_flaky_tests", domain.InputTypeBoolean, "false", false)

	repos := &domain.AllRepositories{
		Workflows: &MockWorkflowRepository{
			inputs: map[string][]domain.Input{
				"deploy.yml": {envInput, flakyInput},
			},
		},
	}

	orchestrator := NewTriggerDeploymentOrchestrator(repos)

	// Act
	inputs, err := orchestrator.GetWorkflowInputs(context.Background(), GetWorkflowInputsRequest{
		Workflow: workflow,
	})

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(inputs) != 2 {
		t.Errorf("expected 2 inputs, got %d", len(inputs))
	}

	if inputs[0].Type() != domain.InputTypeChoice {
		t.Errorf("expected choice type, got %s", inputs[0].Type())
	}

	if inputs[1].Type() != domain.InputTypeBoolean {
		t.Errorf("expected boolean type, got %s", inputs[1].Type())
	}
}

func TestTriggerDeploymentOrchestrator_PrepareDeployment_InputValidation(t *testing.T) {
	// Arrange
	workflow, _ := domain.NewWorkflow("deploy.yml")

	envInput, _ := domain.NewChoiceInput("environment", "", []string{"dev", "staging", "prod"}, true)
	_ = envInput.SetValue("prod") // Valid choice

	repos := &domain.AllRepositories{
		Deployments: &MockDeploymentRepository{},
	}
	orchestrator := NewTriggerDeploymentOrchestrator(repos)

	// Act
	deployment, err := orchestrator.PrepareDeployment(context.Background(), PrepareDeploymentRequest{
		Workflow: workflow,
		Branch:   "main",
		Inputs:   []domain.Input{envInput},
		RepoURL:  "https://github.com/example",
	})

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(deployment.Inputs()) != 1 {
		t.Errorf("expected 1 input, got %d", len(deployment.Inputs()))
	}
}

// Acceptance Test: User triggers a deployment with typed inputs
func TestAcceptance_TriggerDeploymentWithInputs(t *testing.T) {
	// Scenario: Developer triggers a deployment to production with custom configuration

	// Arrange
	workflow, _ := domain.NewWorkflow("deploy.yml")

	// Define typed inputs
	envInput, _ := domain.NewChoiceInput("environment", "prod", []string{"dev", "staging", "prod"}, true)
	skipTestsInput, _ := domain.NewInput("skip_tests", domain.InputTypeBoolean, "false", false)

	repos := &domain.AllRepositories{
		Workflows: &MockWorkflowRepository{
			workflows: []domain.Workflow{workflow},
			inputs: map[string][]domain.Input{
				"deploy.yml": {envInput, skipTestsInput},
			},
		},
		Branches: &MockBranchRepository{
			defaultBranch: "main",
		},
		Runs: &MockRunRepository{
			runs: make(map[string]domain.Run),
		},
		Deployments: &MockDeploymentRepository{},
	}

	orchestrator := NewTriggerDeploymentOrchestrator(repos)

	// Act: Execute deployment with inputs
	deployment, err := orchestrator.Trigger(context.Background(), TriggerRequest{
		WorkflowName: strPtr("deploy.yml"),
		BranchName:   strPtr("main"),
		Inputs: map[string]string{
			"environment": "prod",
			"skip_tests":  "true",
		},
		RepoURL: "https://github.com/example/repo",
	})

	// Assert
	if err != nil {
		t.Fatalf("deployment failed: %v", err)
	}

	if deployment.Workflow().Name() != "deploy.yml" {
		t.Errorf("expected workflow 'deploy.yml'")
	}

	if deployment.Branch() != "main" {
		t.Errorf("expected branch 'main'")
	}

	// Verify inputs were properly set
	if len(deployment.Inputs()) != 2 {
		t.Errorf("expected 2 inputs, got %d", len(deployment.Inputs()))
	}

	// Verify the run was created
	if !deployment.HasRun() {
		t.Errorf("deployment should have a run")
	}

	// Verify the run is in queued state
	if deployment.Run().Status() != domain.RunStatusQueued {
		t.Errorf("expected queued status, got %s", deployment.Run().Status())
	}
}

// Helper
func strPtr(s string) *string {
	return &s
}
