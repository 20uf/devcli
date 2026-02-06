package cmd

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/20uf/devcli/internal/deployment/application"
	"github.com/20uf/devcli/internal/deployment/domain"
	"github.com/20uf/devcli/internal/deployment/infra"
	"github.com/20uf/devcli/internal/history"
	"github.com/20uf/devcli/internal/ui"
	"github.com/spf13/cobra"
)

// DeployHandler bridges the CLI layer and domain layer for deployments.
type DeployHandler struct {
	orchestrator *application.TriggerDeploymentOrchestrator
	repos        *domain.AllRepositories
	history      *history.Store
}

// NewDeployHandler creates a handler with all dependencies wired.
func NewDeployHandler(ctx context.Context, repoURL string) (*DeployHandler, error) {
	repos := infra.CreateRepositories(repoURL)

	hist, _ := history.Load()

	return &DeployHandler{
		orchestrator: application.NewTriggerDeploymentOrchestrator(repos),
		repos:        repos,
		history:      hist,
	}, nil
}

// Handle orchestrates the complete deployment flow.
func (h *DeployHandler) Handle(
	cmd *cobra.Command,
	workflowFlag string,
	branchFlag string,
	inputFlags []string,
	watchFlag bool,
	repoURL string,
) error {
	ctx := cmd.Context()

	// Verify gh CLI is installed
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("GitHub CLI (gh) is required.\n  Install: https://cli.github.com/")
	}

	// Non-interactive mode: all flags provided
	if repoURL != "" && workflowFlag != "" && branchFlag != "" {
		realHandler, err := NewDeployHandler(ctx, repoURL)
		if err != nil {
			return err
		}
		inputs := parseInputFlags(inputFlags)
		deployment, err := realHandler.orchestrator.Trigger(ctx, application.TriggerRequest{
			WorkflowName: &workflowFlag,
			BranchName:   &branchFlag,
			Inputs:       inputs,
			RepoURL:      "",
		})
		if err != nil {
			return err
		}
		return realHandler.executeDeployment(ctx, deployment, watchFlag)
	}

	// Interactive mode: guide user through selection
	return h.interactiveFlow(ctx, workflowFlag, branchFlag, inputFlags, watchFlag)
}

// interactiveFlow guides user through repository → workflow → branch → inputs → trigger.
func (h *DeployHandler) interactiveFlow(
	ctx context.Context,
	workflowFlag string,
	branchFlag string,
	inputFlags []string,
	watchFlag bool,
) error {
	// Step 0: Show history if no flags
	if workflowFlag == "" && branchFlag == "" {
		if histDep, err := h.showHistoryMenu(); err == nil && histDep != nil {
			ui.PrintStep("↻", fmt.Sprintf("Replaying: %s", histDep.String()))
			return h.executeDeployment(ctx, *histDep, watchFlag)
		}
		// User selected "New deployment" or pressed ESC, continue
	}

	// Step 1: Try to select organization, fallback to manual input
	var selectedOrg string
	organizations, err := listOrganizations()

	if err != nil || len(organizations) == 0 {
		// Fallback: ask user to enter organization manually
		ui.PrintWarning("Unable to list organizations - enter manually")
		selectedOrg, err = ui.Input("Enter organization", "myorg")
		if err != nil {
			return err
		}
		if selectedOrg == "" {
			return fmt.Errorf("organization is required")
		}
	} else {
		// Normal: select from list
		selectedOrg, err = ui.Select("Select organization", organizations)
		if err != nil {
			ui.PrintWarning("Cancelled - returning to menu")
			return nil
		}
	}

	// Step 2: Select repository (from selected organization)
	repositories, err := listRepositoriesByOrg(selectedOrg)
	if err != nil {
		return fmt.Errorf("failed to list repositories for %s: %w", selectedOrg, err)
	}

	if len(repositories) == 0 {
		return fmt.Errorf("no repositories found in %s", selectedOrg)
	}

	selectedRepo, err := ui.Select("Select repository", repositories)
	if err != nil {
		ui.PrintWarning("Cancelled - returning to menu")
		return nil
	}

	// Step 2: Create handler with selected repository
	realHandler, err := NewDeployHandler(ctx, selectedRepo)
	if err != nil {
		return fmt.Errorf("failed to initialize deployment handler for %s: %w", selectedRepo, err)
	}

	// Step 3: Select workflow
	workflows, err := realHandler.repos.Workflows.ListWorkflows(ctx)
	if err != nil {
		return fmt.Errorf("failed to list workflows: %w", err)
	}

	workflowNames := make([]string, len(workflows))
	for i, w := range workflows {
		workflowNames[i] = w.Name()
	}

	if workflowFlag != "" {
		workflowNames = []string{workflowFlag}
	}

	selectedWorkflowName, err := ui.Select("Select workflow", workflowNames)
	if err != nil {
		return nil
	}

	workflow, _ := domain.NewWorkflow(selectedWorkflowName)

	// Step 4: Get workflow inputs (typed!)
	inputs, err := realHandler.repos.Workflows.GetWorkflowInputs(ctx, workflow)
	if err != nil {
		return err
	}

	// Step 5: Select branch
	branches, err := listBranches(selectedOrg, selectedRepo)
	if err != nil {
		return fmt.Errorf("failed to list branches for %s/%s: %w", selectedOrg, selectedRepo, err)
	}

	if branchFlag != "" {
		branches = []string{branchFlag}
	}

	selectedBranch, err := ui.Select("Select branch", branches)
	if err != nil {
		return nil
	}

	// Step 6: Collect input values (with type validation)
	if len(inputs) > 0 {
		inputs, err = realHandler.collectInputs(ctx, inputs, inputFlags)
		if err != nil {
			return err
		}
	}

	// Step 7: Prepare and execute deployment
	inputMap := realHandler.inputsToMap(inputs)
	deployment, err := realHandler.orchestrator.Trigger(ctx, application.TriggerRequest{
		WorkflowName: &selectedWorkflowName,
		BranchName:   &selectedBranch,
		Inputs:       inputMap,
		RepoURL:      "",
	})
	if err != nil {
		return err
	}

	return realHandler.executeDeployment(ctx, deployment, watchFlag)
}

// listOrganizations retrieves user's organizations using gh CLI.
func listOrganizations() ([]string, error) {
	cmd := exec.Command("gh", "api", "user/orgs", "--jq", ".[].login")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	orgs := []string{}
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line != "" {
			orgs = append(orgs, line)
		}
	}

	return orgs, nil
}

// listRepositoriesByOrg retrieves repositories for a specific organization.
func listRepositoriesByOrg(org string) ([]string, error) {
	cmd := exec.Command("gh", "repo", "list", org, "--limit", "50", "--json", "nameWithOwner", "-q", ".[].nameWithOwner")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	repos := []string{}
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line != "" {
			repos = append(repos, line)
		}
	}

	return repos, nil
}

// listBranches retrieves the 50 most recently active branches.
func listBranches(org, repo string) ([]string, error) {
	// repo might be "org/name" format from listRepositoriesByOrg
	fullRepo := repo
	if !strings.Contains(repo, "/") {
		fullRepo = org + "/" + repo
	}

	// Fetch branches sorted by latest commit date (most active first)
	cmd := exec.Command(
		"gh", "api", "repos/"+fullRepo+"/branches",
		"--jq", "sort_by(.commit.date) | reverse | .[0:50] | .[] | .name",
	)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	branches := []string{}
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line != "" {
			branches = append(branches, line)
		}
	}

	return branches, nil
}

// collectInputs guides user through providing typed input values.
func (h *DeployHandler) collectInputs(ctx context.Context, inputs []domain.Input, flags []string) ([]domain.Input, error) {
	flagMap := parseInputFlags(flags)

	for i, input := range inputs {
		// Check if value was provided via flag
		if val, ok := flagMap[input.Key()]; ok {
			if err := input.SetValue(val); err != nil {
				return nil, fmt.Errorf("input %s validation failed: %w", input.Key(), err)
			}
			inputs[i] = input
			continue
		}

		// Prompt user based on input type
		switch input.Type() {
		case domain.InputTypeChoice:
			selectedValue, err := ui.Select(
				fmt.Sprintf("Select %s", input.Key()),
				input.Options(),
			)
			if err != nil {
				return nil, err
			}
			if err := input.SetValue(selectedValue); err != nil {
				return nil, fmt.Errorf("input %s validation failed: %w", input.Key(), err)
			}

		case domain.InputTypeBoolean:
			confirmed, err := ui.Confirm(fmt.Sprintf("Enable %s?", input.Key()))
			if err != nil {
				return nil, err
			}
			value := "false"
			if confirmed {
				value = "true"
			}
			if err := input.SetValue(value); err != nil {
				return nil, fmt.Errorf("input %s validation failed: %w", input.Key(), err)
			}

		case domain.InputTypeString:
			value, err := ui.Input(fmt.Sprintf("Enter %s", input.Key()), "")
			if err != nil {
				return nil, err
			}
			if err := input.SetValue(value); err != nil {
				return nil, fmt.Errorf("input %s validation failed: %w", input.Key(), err)
			}

		default:
			value, err := ui.Input(fmt.Sprintf("Enter %s", input.Key()), "")
			if err != nil {
				return nil, err
			}
			if err := input.SetValue(value); err != nil {
				return nil, fmt.Errorf("input %s validation failed: %w", input.Key(), err)
			}
		}

		inputs[i] = input
	}

	return inputs, nil
}

// executeDeployment saves to history and executes the workflow trigger.
func (h *DeployHandler) executeDeployment(ctx context.Context, deployment domain.Deployment, watch bool) error {
	// Save to history
	if h.history != nil {
		label := deployment.Workflow().Name()
		args := []string{"--workflow", deployment.Workflow().Name(), "--branch", deployment.Branch()}

		for _, input := range deployment.Inputs() {
			args = append(args, "--input", fmt.Sprintf("%s=%s", input.Key(), input.Value()))
		}

		h.history.Add("deploy", label, args)
		h.history.Save() //nolint:errcheck
	}

	ui.PrintStep("▶", fmt.Sprintf("Triggering %s on %s", deployment.Workflow().Name(), deployment.Branch()))

	if deployment.HasRun() {
		ui.PrintSuccess(fmt.Sprintf("Workflow triggered: run %s", deployment.Run().ID()))

		if watch {
			ui.PrintInfo("Deployment tracking", "View progress with: devcli status")
		}
	}

	return nil
}

// showHistoryMenu displays recent deployments for replay.
func (h *DeployHandler) showHistoryMenu() (*domain.Deployment, error) {
	if h.history == nil {
		return nil, nil
	}

	labels := h.history.Labels("deploy")
	if len(labels) == 0 {
		return nil, nil
	}

	if len(labels) > 10 {
		labels = labels[:10]
	}

	labels = append([]string{"+ New deployment"}, labels...)
	selected, err := ui.Select("Recent deployments", labels)
	if err != nil {
		return nil, err
	}

	if selected == "+ New deployment" {
		return nil, nil
	}

	labelPrefix := selected[:strings.LastIndex(selected, " (")]
	entry := h.history.FindByLabel("deploy", labelPrefix)
	if entry == nil {
		return nil, nil
	}

	return nil, nil
}

// Helper functions

func parseInputFlags(flags []string) map[string]string {
	inputs := make(map[string]string)
	for _, flag := range flags {
		parts := strings.Split(flag, "=")
		if len(parts) == 2 {
			inputs[parts[0]] = parts[1]
		}
	}
	return inputs
}

func (h *DeployHandler) inputsToMap(inputs []domain.Input) map[string]string {
	result := make(map[string]string)
	for _, input := range inputs {
		result[input.Key()] = input.Value()
	}
	return result
}
