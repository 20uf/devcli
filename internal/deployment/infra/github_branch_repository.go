package infra

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/20uf/devcli/internal/verbose"
)

// GitHubBranchRepository implements BranchRepository using GitHub API via gh CLI.
type GitHubBranchRepository struct {
	repoURL string
}

// NewGitHubBranchRepository creates a new GitHub branch repository.
func NewGitHubBranchRepository(repoURL string) *GitHubBranchRepository {
	return &GitHubBranchRepository{
		repoURL: repoURL,
	}
}

// ListBranches returns all branches in the repository.
func (r *GitHubBranchRepository) ListBranches(ctx context.Context) ([]string, error) {
	cmd := verbose.Cmd(exec.CommandContext(ctx, "gh", "branch", "list",
		"--repo", r.repoURL,
		"--json", "name",
		"-q", ".[].name"))

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}

	var branches []string
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Remove quotes if present (gh CLI output format)
		line = strings.Trim(line, `"`)
		branches = append(branches, line)
	}

	if len(branches) == 0 {
		return nil, fmt.Errorf("no branches found in repository")
	}

	return branches, nil
}

// GetDefaultBranch returns the default branch of the repository.
func (r *GitHubBranchRepository) GetDefaultBranch(ctx context.Context) (string, error) {
	cmd := verbose.Cmd(exec.CommandContext(ctx, "gh", "api",
		fmt.Sprintf("repos/%s", r.repoURL),
		"--jq", ".default_branch"))

	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get default branch: %w", err)
	}

	branch := strings.TrimSpace(string(out))
	branch = strings.Trim(branch, `"`)

	if branch == "" {
		return "", fmt.Errorf("no default branch found")
	}

	return branch, nil
}
