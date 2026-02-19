package infra

import (
	"context"

	"github.com/20uf/devcli/internal/deployment/domain"
)

// MockWorkflowRepository is a mock implementation for testing.
type MockWorkflowRepository struct{}

func NewMockWorkflowRepository() *MockWorkflowRepository {
	return &MockWorkflowRepository{}
}

func (m *MockWorkflowRepository) ListWorkflows(ctx context.Context) ([]domain.Workflow, error) {
	w1, _ := domain.NewWorkflow("deploy.yml")
	w2, _ := domain.NewWorkflow("test.yml")
	return []domain.Workflow{w1, w2}, nil
}

func (m *MockWorkflowRepository) GetWorkflow(ctx context.Context, name string) (*domain.Workflow, error) {
	w, err := domain.NewWorkflow(name)
	return &w, err
}

func (m *MockWorkflowRepository) GetWorkflowInputs(ctx context.Context, workflow domain.Workflow) ([]domain.Input, error) {
	if workflow.Name() == "deploy.yml" {
		env, _ := domain.NewChoiceInput("environment", "", []string{"dev", "staging", "prod"}, true)
		skip, _ := domain.NewInput("skip_tests", domain.InputTypeBoolean, "false", false)
		return []domain.Input{env, skip}, nil
	}
	return []domain.Input{}, nil
}

// MockRunRepository is a mock implementation for testing.
type MockRunRepository struct{}

func NewMockRunRepository() *MockRunRepository {
	return &MockRunRepository{}
}

func (m *MockRunRepository) CreateRun(ctx context.Context, deployment domain.Deployment) (*domain.Run, error) {
	run := domain.NewRun("run-123", 42, domain.RunStatusQueued, deployment.Branch(), "https://github.com/example")
	return &run, nil
}

func (m *MockRunRepository) GetRun(ctx context.Context, runID string) (*domain.Run, error) {
	run := domain.NewRun(runID, 42, domain.RunStatusInProgress, "main", "https://github.com")
	return &run, nil
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

// MockBranchRepository is a mock implementation for testing.
type MockBranchRepository struct{}

func NewMockBranchRepository() *MockBranchRepository {
	return &MockBranchRepository{}
}

func (m *MockBranchRepository) ListBranches(ctx context.Context) ([]string, error) {
	return []string{"main", "develop", "feature-x"}, nil
}

func (m *MockBranchRepository) GetDefaultBranch(ctx context.Context) (string, error) {
	return "main", nil
}

// MockDeploymentRepository is a mock implementation for testing.
type MockDeploymentRepository struct{}

func NewMockDeploymentRepository() *MockDeploymentRepository {
	return &MockDeploymentRepository{}
}

func (m *MockDeploymentRepository) Save(ctx context.Context, deployment domain.Deployment) error {
	return nil
}

func (m *MockDeploymentRepository) FindByID(ctx context.Context, id string) (*domain.Deployment, error) {
	return nil, nil
}

func (m *MockDeploymentRepository) FindRecent(ctx context.Context, limit int) ([]domain.Deployment, error) {
	return []domain.Deployment{}, nil
}
