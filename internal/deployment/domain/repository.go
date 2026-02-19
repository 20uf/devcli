package domain

import "context"

// WorkflowRepository defines the interface for accessing workflows.
type WorkflowRepository interface {
	// ListWorkflows returns all available workflows in the repository.
	ListWorkflows(ctx context.Context) ([]Workflow, error)

	// GetWorkflow retrieves a specific workflow by name.
	GetWorkflow(ctx context.Context, name string) (*Workflow, error)

	// GetWorkflowInputs retrieves the typed inputs required by a workflow.
	GetWorkflowInputs(ctx context.Context, workflow Workflow) ([]Input, error)
}

// RunRepository defines the interface for accessing and managing runs.
type RunRepository interface {
	// CreateRun triggers a new workflow run and returns the created run.
	CreateRun(ctx context.Context, deployment Deployment) (*Run, error)

	// GetRun retrieves a specific run by ID.
	GetRun(ctx context.Context, runID string) (*Run, error)

	// UpdateRunStatus updates the status of a run.
	UpdateRunStatus(ctx context.Context, runID string, status RunStatus) error

	// UpdateRunConclusion updates the conclusion of a run.
	UpdateRunConclusion(ctx context.Context, runID string, conclusion RunConclusion) error

	// GetRunLogs retrieves the logs for a run.
	GetRunLogs(ctx context.Context, runID string) (string, error)
}

// BranchRepository defines the interface for accessing branch information.
type BranchRepository interface {
	// ListBranches returns all branches in the repository.
	ListBranches(ctx context.Context) ([]string, error)

	// GetDefaultBranch returns the default branch.
	GetDefaultBranch(ctx context.Context) (string, error)
}

// DeploymentRepository defines the interface for persisting deployments.
type DeploymentRepository interface {
	// Save persists a deployment record.
	Save(ctx context.Context, deployment Deployment) error

	// FindByID retrieves a deployment by its ID.
	FindByID(ctx context.Context, id string) (*Deployment, error)

	// FindRecent retrieves recent deployments.
	FindRecent(ctx context.Context, limit int) ([]Deployment, error)
}

// AllRepositories bundles all repositories needed for the deployment context.
type AllRepositories struct {
	Workflows   WorkflowRepository
	Runs        RunRepository
	Branches    BranchRepository
	Deployments DeploymentRepository
}
