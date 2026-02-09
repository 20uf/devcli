package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/20uf/devcli/internal/ui"
	"github.com/spf13/cobra"
)

var (
	flagRepo     string
	flagWorkflow string
	flagBranch   string
	flagInputs   []string
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Trigger a GitHub Actions deployment workflow",
	Long:  "List and trigger GitHub Actions workflows via the gh CLI.",
	RunE:  runDeploy,
}

func init() {
	deployCmd.Flags().StringVar(&flagRepo, "repo", "", "GitHub repository (owner/repo)")
	deployCmd.Flags().StringVar(&flagWorkflow, "workflow", "", "Workflow file name or ID (skip selection)")
	deployCmd.Flags().StringVar(&flagBranch, "branch", "", "Branch to run the workflow on (default: main)")
	deployCmd.Flags().StringSliceVar(&flagInputs, "input", nil, "Workflow inputs as key=value pairs")
	rootCmd.AddCommand(deployCmd)
}

type ghWorkflow struct {
	Name  string `json:"name"`
	ID    int    `json:"id"`
	Path  string `json:"path"`
	State string `json:"state"`
}

func runDeploy(cmd *cobra.Command, args []string) error {
	// Check gh is installed
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("GitHub CLI (gh) is required.\n  Install: https://cli.github.com/")
	}

	// Determine repo
	repo, err := resolveRepo()
	if err != nil {
		return err
	}

	// Select workflow
	workflow, err := selectWorkflow(repo)
	if err != nil {
		return err
	}

	// Determine branch
	branch := flagBranch
	if branch == "" {
		branch = "main"
	}

	// Build command
	ghArgs := []string{"workflow", "run", workflow, "--repo", repo, "--ref", branch}

	for _, input := range flagInputs {
		ghArgs = append(ghArgs, "--field", input)
	}

	fmt.Printf("Triggering workflow %q on %s (branch: %s)...\n", workflow, repo, branch)

	c := exec.Command("gh", ghArgs...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	if err := c.Run(); err != nil {
		return fmt.Errorf("failed to trigger workflow: %w", err)
	}

	fmt.Println("Workflow triggered successfully.")
	fmt.Printf("View runs: gh run list --repo %s --workflow %s\n", repo, workflow)
	return nil
}

func resolveRepo() (string, error) {
	if flagRepo != "" {
		return flagRepo, nil
	}

	// Try to detect from current git repo
	out, err := exec.Command("gh", "repo", "view", "--json", "nameWithOwner", "-q", ".nameWithOwner").Output()
	if err == nil {
		repo := strings.TrimSpace(string(out))
		if repo != "" {
			return repo, nil
		}
	}

	return "", fmt.Errorf("could not detect repository. Use --repo owner/repo")
}

func selectWorkflow(repo string) (string, error) {
	if flagWorkflow != "" {
		return flagWorkflow, nil
	}

	out, err := exec.Command("gh", "workflow", "list", "--repo", repo, "--json", "name,id,path,state").Output()
	if err != nil {
		return "", fmt.Errorf("failed to list workflows: %w", err)
	}

	var workflows []ghWorkflow
	if err := json.Unmarshal(out, &workflows); err != nil {
		return "", fmt.Errorf("failed to parse workflows: %w", err)
	}

	// Filter active workflows
	var active []ghWorkflow
	for _, w := range workflows {
		if w.State == "active" {
			active = append(active, w)
		}
	}

	if len(active) == 0 {
		return "", fmt.Errorf("no active workflows found in %s", repo)
	}

	// Build display names
	options := make([]string, len(active))
	for i, w := range active {
		options[i] = fmt.Sprintf("%s (%s)", w.Name, extractWorkflowFile(w.Path))
	}

	selected, err := ui.Select("Select workflow to trigger", options)
	if err != nil {
		os.Exit(0)
	}

	// Find matching workflow and return its file name
	for i, opt := range options {
		if opt == selected {
			return extractWorkflowFile(active[i].Path), nil
		}
	}

	return "", fmt.Errorf("workflow not found")
}

func extractWorkflowFile(path string) string {
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}
