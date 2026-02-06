package cmd

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"
)

var (
	flagRepo     string
	flagWorkflow string
	flagBranch   string
	flagInputs   []string
	flagWatch    bool
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Trigger a GitHub Actions deployment workflow",
	Long: `Trigger a GitHub Actions deployment workflow via the gh CLI.

Examples:
  devcli deploy                                          Interactive selection
  devcli deploy --repo owner/repo --workflow deploy.yml  Non-interactive
  devcli deploy --branch feature-x --watch               Deploy and stream logs
  devcli deploy --input environment=prod --input v=1.2   With workflow inputs`,
	RunE: runDeploy,
}

func init() {
	deployCmd.Flags().StringVar(&flagRepo, "repo", "", "GitHub repository (owner/repo)")
	deployCmd.Flags().StringVar(&flagWorkflow, "workflow", "", "Workflow file name or ID")
	deployCmd.Flags().StringVar(&flagBranch, "branch", "", "Branch to run the workflow on")
	deployCmd.Flags().StringSliceVar(&flagInputs, "input", nil, "Workflow inputs (key=value)")
	deployCmd.Flags().BoolVar(&flagWatch, "watch", false, "Watch workflow run and stream logs")
	rootCmd.AddCommand(deployCmd)
}

func runDeploy(cmd *cobra.Command, args []string) error {
	// Check gh CLI is installed
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("GitHub CLI (gh) is required.\n  Install: https://cli.github.com/")
	}

	// Create handler (wires all dependencies: domain + repos + UI)
	handler, err := NewDeployHandler(cmd.Context(), flagRepo)
	if err != nil {
		return fmt.Errorf("failed to initialize deployment handler: %w", err)
	}

	// Orchestrate the deployment flow
	// Handler manages: workflow selection → branch → inputs → trigger → (optional) watch
	return handler.Handle(cmd, flagWorkflow, flagBranch, flagInputs, flagWatch, flagRepo)
}
