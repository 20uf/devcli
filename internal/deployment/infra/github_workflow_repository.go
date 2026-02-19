package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/20uf/devcli/internal/deployment/domain"
	"github.com/20uf/devcli/internal/verbose"
)

// GitHubWorkflowRepository implements WorkflowRepository using GitHub API via gh CLI.
type GitHubWorkflowRepository struct {
	repoURL string
}

// NewGitHubWorkflowRepository creates a new GitHub workflow repository.
func NewGitHubWorkflowRepository(repoURL string) *GitHubWorkflowRepository {
	return &GitHubWorkflowRepository{
		repoURL: repoURL,
	}
}

// ListWorkflows fetches available workflows from GitHub.
func (r *GitHubWorkflowRepository) ListWorkflows(ctx context.Context) ([]domain.Workflow, error) {
	// Use gh CLI to list workflows as JSON
	cmd := verbose.Cmd(exec.CommandContext(ctx, "gh", "workflow", "list",
		"--repo", r.repoURL,
		"--json", "name",
		"-q", ".[].name"))

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list workflows: %w", err)
	}

	var workflows []domain.Workflow
	names := strings.Split(strings.TrimSpace(string(out)), "\n")

	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		// Remove quotes if present (gh CLI output format)
		name = strings.Trim(name, `"`)

		workflow, err := domain.NewWorkflow(name)
		if err != nil {
			continue
		}
		workflows = append(workflows, workflow)
	}

	if len(workflows) == 0 {
		return nil, fmt.Errorf("no workflows found in repository")
	}

	return workflows, nil
}

// GetWorkflow retrieves a specific workflow by name.
func (r *GitHubWorkflowRepository) GetWorkflow(ctx context.Context, name string) (*domain.Workflow, error) {
	workflow, err := domain.NewWorkflow(name)
	if err != nil {
		return nil, fmt.Errorf("invalid workflow name: %w", err)
	}

	return &workflow, nil
}

// GetWorkflowInputs retrieves typed inputs required by a workflow.
// Parses workflow YAML via GitHub API to extract workflow_dispatch inputs.
func (r *GitHubWorkflowRepository) GetWorkflowInputs(ctx context.Context, workflow domain.Workflow) ([]domain.Input, error) {
	// GitHub API: GET /repos/{owner}/{repo}/actions/workflows/{workflow_id}
	// We use gh API to fetch the workflow and parse its inputs

	cmd := verbose.Cmd(exec.CommandContext(ctx, "gh", "api",
		fmt.Sprintf("repos/%s/actions/workflows/%s", r.repoURL, workflow.Name()),
		"--jq", ".on.workflow_dispatch.inputs"))

	out, err := cmd.Output()
	if err != nil {
		// Workflow doesn't exist or has no workflow_dispatch inputs
		return []domain.Input{}, nil
	}

	var inputsData map[string]interface{}
	if err := json.Unmarshal(out, &inputsData); err != nil {
		return []domain.Input{}, nil
	}

	var inputs []domain.Input

	for key, val := range inputsData {
		inputMap, ok := val.(map[string]interface{})
		if !ok {
			continue
		}

		var inputType domain.InputType
		if t, ok := inputMap["type"].(string); ok {
			switch t {
			case "choice":
				inputType = domain.InputTypeChoice
			case "boolean":
				inputType = domain.InputTypeBoolean
			default:
				inputType = domain.InputTypeString
			}
		} else {
			inputType = domain.InputTypeString
		}

		required := true
		if r, ok := inputMap["required"].(bool); ok {
			required = r
		}

		defaultVal := ""
		if d, ok := inputMap["default"].(string); ok {
			defaultVal = d
		}

		if inputType == domain.InputTypeChoice {
			var options []string
			if opts, ok := inputMap["options"].([]interface{}); ok {
				for _, opt := range opts {
					if optStr, ok := opt.(string); ok {
						options = append(options, optStr)
					}
				}
			}

			input, err := domain.NewChoiceInput(key, defaultVal, options, required)
			if err == nil {
				inputs = append(inputs, input)
			}
		} else {
			input, err := domain.NewInput(key, inputType, defaultVal, required)
			if err == nil {
				inputs = append(inputs, input)
			}
		}
	}

	return inputs, nil
}
