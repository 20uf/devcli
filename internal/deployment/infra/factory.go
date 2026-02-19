package infra

import (
	"os"
	"path/filepath"

	"github.com/20uf/devcli/internal/deployment/domain"
)

// CreateRepositories creates and returns all deployment repositories.
// Uses GitHub API via gh CLI for real implementations.
func CreateRepositories(repoURL string) *domain.AllRepositories {
	return &domain.AllRepositories{
		Workflows:   NewGitHubWorkflowRepository(repoURL),
		Runs:        NewGitHubRunRepository(repoURL),
		Branches:    NewGitHubBranchRepository(repoURL),
		Deployments: NewFileDeploymentRepository(getDeploymentStorePath()),
	}
}

// CreateMockRepositories creates mock implementations for testing.
// Uses in-memory implementations that don't require GitHub access.
func CreateMockRepositories() *domain.AllRepositories {
	return &domain.AllRepositories{
		Workflows:   NewMockWorkflowRepository(),
		Runs:        NewMockRunRepository(),
		Branches:    NewMockBranchRepository(),
		Deployments: NewMockDeploymentRepository(),
	}
}

// getDeploymentStorePath returns the path where deployments are stored locally.
func getDeploymentStorePath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".devcli", "deployments")
}
