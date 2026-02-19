package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/20uf/devcli/internal/deployment/domain"
	"github.com/20uf/devcli/internal/verbose"
)

// GitHubRunRepository implements RunRepository using GitHub API via gh CLI.
type GitHubRunRepository struct {
	repoURL string
}

// NewGitHubRunRepository creates a new GitHub run repository.
func NewGitHubRunRepository(repoURL string) *GitHubRunRepository {
	return &GitHubRunRepository{
		repoURL: repoURL,
	}
}

// CreateRun triggers a new workflow run and returns the created run.
func (r *GitHubRunRepository) CreateRun(ctx context.Context, deployment domain.Deployment) (*domain.Run, error) {
	var inputParams []string
	for _, input := range deployment.Inputs() {
		inputParams = append(inputParams, fmt.Sprintf("%s=%s", input.Key(), input.Value()))
	}

	// Trigger workflow via gh CLI: gh workflow run <workflow> [-r branch] [--input <key=value>...]
	args := []string{"workflow", "run", deployment.Workflow().Name(), "--repo", r.repoURL}

	if deployment.Branch() != "" {
		args = append(args, "-r", deployment.Branch())
	}

	for _, param := range inputParams {
		args = append(args, "--input", param)
	}

	cmd := verbose.Cmd(exec.CommandContext(ctx, "gh", args...))
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to trigger workflow: %w", err)
	}

	// Wait a moment for the run to appear in GitHub
	time.Sleep(2 * time.Second)

	runID, err := r.getLatestRunID(ctx, deployment.Workflow().Name())
	if err != nil {
		return nil, fmt.Errorf("failed to get run ID: %w", err)
	}

	return r.GetRun(ctx, runID)
}

// GetRun retrieves a specific run by ID.
func (r *GitHubRunRepository) GetRun(ctx context.Context, runID string) (*domain.Run, error) {
	cmd := verbose.Cmd(exec.CommandContext(ctx, "gh", "run", "view", runID,
		"--repo", r.repoURL,
		"--json", "databaseId,status,conclusion,createdAt,updatedAt"))

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch run: %w", err)
	}

	var runData struct {
		DatabaseID string `json:"databaseId"`
		Status     string `json:"status"`
		Conclusion string `json:"conclusion"`
		CreatedAt  string `json:"createdAt"`
		UpdatedAt  string `json:"updatedAt"`
	}

	if err := json.Unmarshal(out, &runData); err != nil {
		return nil, fmt.Errorf("failed to parse run data: %w", err)
	}

	status := stringToRunStatus(runData.Status)
	conclusion := stringToRunConclusion(runData.Conclusion)

	// Note: We use a simplified approach - would need workflow name in practice
	run := domain.NewRun(runID, 0, status, "", r.repoURL)

	if conclusion != "" {
		_ = r.UpdateRunConclusion(ctx, runID, conclusion)
	}

	return &run, nil
}

// UpdateRunStatus updates the status of a run.
func (r *GitHubRunRepository) UpdateRunStatus(ctx context.Context, runID string, status domain.RunStatus) error {
	// Status is read-only from GitHub API - we only fetch it
	// This is a no-op in production but needed for interface compliance
	return nil
}

// UpdateRunConclusion updates the conclusion of a run.
func (r *GitHubRunRepository) UpdateRunConclusion(ctx context.Context, runID string, conclusion domain.RunConclusion) error {
	// Conclusion is read-only from GitHub API - we only fetch it
	// This is a no-op in production but needed for interface compliance
	return nil
}

// GetRunLogs retrieves the logs for a run.
func (r *GitHubRunRepository) GetRunLogs(ctx context.Context, runID string) (string, error) {
	cmd := verbose.Cmd(exec.CommandContext(ctx, "gh", "run", "view", runID,
		"--repo", r.repoURL,
		"--log"))

	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to fetch logs: %w", err)
	}

	return string(out), nil
}

// getLatestRunID fetches the most recent run ID for a workflow.
func (r *GitHubRunRepository) getLatestRunID(ctx context.Context, workflowName string) (string, error) {
	cmd := verbose.Cmd(exec.CommandContext(ctx, "gh", "run", "list",
		"--repo", r.repoURL,
		"--workflow", workflowName,
		"--limit", "1",
		"--json", "databaseId",
		"-q", ".[0].databaseId"))

	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	id := strings.TrimSpace(string(out))
	if id == "" {
		return "", fmt.Errorf("no run found")
	}

	return strings.Trim(id, `"`), nil
}

// stringToRunStatus converts GitHub status strings to domain.RunStatus.
func stringToRunStatus(s string) domain.RunStatus {
	switch s {
	case "queued":
		return domain.RunStatusQueued
	case "in_progress":
		return domain.RunStatusInProgress
	case "completed":
		return domain.RunStatusCompleted
	default:
		return domain.RunStatusQueued
	}
}

// stringToRunConclusion converts GitHub conclusion strings to domain.RunConclusion.
func stringToRunConclusion(s string) domain.RunConclusion {
	switch s {
	case "success":
		return domain.RunConclusionSuccess
	case "failure":
		return domain.RunConclusionFailure
	case "cancelled":
		return domain.RunConclusionCancelled
	case "skipped":
		return domain.RunConclusionSkipped
	default:
		return ""
	}
}
