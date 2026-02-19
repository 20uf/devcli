package application

import (
	"context"
	"fmt"

	"github.com/20uf/devcli/internal/deployment/domain"
)

// TriggerDeploymentOrchestrator is the main use case for triggering a deployment.
// It orchestrates the domain logic: select workflow → get inputs → collect values → trigger run.
// This application service is framework-agnostic and fully testable.
type TriggerDeploymentOrchestrator struct {
	repos *domain.AllRepositories
}

// NewTriggerDeploymentOrchestrator creates a new orchestrator service.
func NewTriggerDeploymentOrchestrator(repos *domain.AllRepositories) *TriggerDeploymentOrchestrator {
	return &TriggerDeploymentOrchestrator{repos: repos}
}

// SelectWorkflowRequest represents the request to select a workflow.
type SelectWorkflowRequest struct {
	WorkflowName *string // If provided, skip selection
}

// SelectWorkflow selects a workflow from available options.
func (o *TriggerDeploymentOrchestrator) SelectWorkflow(ctx context.Context, req SelectWorkflowRequest) (domain.Workflow, error) {
	if req.WorkflowName != nil && *req.WorkflowName != "" {
		// Direct selection: validate that the workflow exists
		return domain.NewWorkflow(*req.WorkflowName)
	}

	// List available workflows
	workflows, err := o.repos.Workflows.ListWorkflows(ctx)
	if err != nil {
		return domain.Workflow{}, err
	}

	if len(workflows) == 0 {
		return domain.Workflow{}, domain.ErrNoWorkflowFound
	}

	// Return the first workflow; UI layer will handle multi-selection if needed
	return workflows[0], nil
}

// SelectBranchRequest represents the request to select a branch.
type SelectBranchRequest struct {
	BranchName *string // If provided, skip selection
}

// SelectBranch selects a branch to run the workflow on.
// If provided in request, returns it; otherwise uses default branch.
func (o *TriggerDeploymentOrchestrator) SelectBranch(ctx context.Context, req SelectBranchRequest) (string, error) {
	if req.BranchName != nil && *req.BranchName != "" {
		return *req.BranchName, nil
	}

	defaultBranch, err := o.repos.Branches.GetDefaultBranch(ctx)
	if err != nil {
		return "", err
	}

	return defaultBranch, nil
}

// GetWorkflowInputsRequest represents the request to get workflow inputs.
type GetWorkflowInputsRequest struct {
	Workflow domain.Workflow
}

// GetWorkflowInputs retrieves the inputs required by a workflow.
func (o *TriggerDeploymentOrchestrator) GetWorkflowInputs(ctx context.Context, req GetWorkflowInputsRequest) ([]domain.Input, error) {
	inputs, err := o.repos.Workflows.GetWorkflowInputs(ctx, req.Workflow)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow inputs: %w", err)
	}

	return inputs, nil
}

// PrepareDeploymentRequest represents a request to prepare a deployment.
type PrepareDeploymentRequest struct {
	Workflow domain.Workflow
	Branch   string
	Inputs   []domain.Input
	RepoURL  string
}

// PrepareDeployment creates and validates a deployment.
// Adds inputs and validates them before execution.
func (o *TriggerDeploymentOrchestrator) PrepareDeployment(ctx context.Context, req PrepareDeploymentRequest) (domain.Deployment, error) {
	deployment, err := domain.NewDeployment(
		fmt.Sprintf("dep-%d", ctx.Value("requestID")),
		req.Workflow,
		req.Branch,
		req.RepoURL,
	)
	if err != nil {
		return domain.Deployment{}, err
	}

	for _, input := range req.Inputs {
		if err := deployment.AddInput(input); err != nil {
			return domain.Deployment{}, fmt.Errorf("failed to add input %s: %w", input.Key(), err)
		}
	}

	if err := deployment.ValidateInputs(); err != nil {
		return domain.Deployment{}, fmt.Errorf("input validation failed: %w", err)
	}

	return deployment, nil
}

// ExecuteDeploymentRequest represents a request to execute a deployment.
type ExecuteDeploymentRequest struct {
	Deployment domain.Deployment
}

// ExecuteDeployment triggers the workflow run.
func (o *TriggerDeploymentOrchestrator) ExecuteDeployment(ctx context.Context, req ExecuteDeploymentRequest) (domain.Deployment, error) {
	// Trigger the run
	run, err := o.repos.Runs.CreateRun(ctx, req.Deployment)
	if err != nil {
		return domain.Deployment{}, fmt.Errorf("failed to create run: %w", err)
	}

	// Store the run in the deployment
	deployment := req.Deployment
	if run != nil {
		deployment.SetRun(*run)
	}

	// Save for history/replay
	_ = o.repos.Deployments.Save(ctx, deployment)

	return deployment, nil
}

// TriggerRequest represents a complete deployment trigger request.
type TriggerRequest struct {
	WorkflowName *string
	BranchName   *string
	Inputs       map[string]string // User-provided input values
	RepoURL      string
}

// Trigger orchestrates the complete deployment flow.
// UseCase: select workflow → select branch → validate inputs → create deployment → execute.
func (o *TriggerDeploymentOrchestrator) Trigger(ctx context.Context, req TriggerRequest) (domain.Deployment, error) {
	workflow, err := o.SelectWorkflow(ctx, SelectWorkflowRequest{WorkflowName: req.WorkflowName})
	if err != nil {
		return domain.Deployment{}, fmt.Errorf("workflow selection failed: %w", err)
	}

	branch, err := o.SelectBranch(ctx, SelectBranchRequest{BranchName: req.BranchName})
	if err != nil {
		return domain.Deployment{}, fmt.Errorf("branch selection failed: %w", err)
	}

	inputs, err := o.GetWorkflowInputs(ctx, GetWorkflowInputsRequest{Workflow: workflow})
	if err != nil {
		return domain.Deployment{}, fmt.Errorf("failed to get inputs: %w", err)
	}

	for i := range inputs {
		if val, ok := req.Inputs[inputs[i].Key()]; ok {
			if err := inputs[i].SetValue(val); err != nil {
				return domain.Deployment{}, fmt.Errorf("input %s validation failed: %w", inputs[i].Key(), err)
			}
		}
	}

	deployment, err := o.PrepareDeployment(ctx, PrepareDeploymentRequest{
		Workflow: workflow,
		Branch:   branch,
		Inputs:   inputs,
		RepoURL:  req.RepoURL,
	})
	if err != nil {
		return domain.Deployment{}, fmt.Errorf("deployment preparation failed: %w", err)
	}

	return o.ExecuteDeployment(ctx, ExecuteDeploymentRequest{Deployment: deployment})
}
