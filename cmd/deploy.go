package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/20uf/devcli/internal/history"
	"github.com/20uf/devcli/internal/ui"
	"github.com/spf13/cobra"
)

var (
	flagRepo     string
	flagWorkflow string
	flagBranch   string
	flagInputs   []string
	flagWatch    bool
	flagLast     bool
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Trigger a GitHub Actions deployment workflow",
	Long: `Trigger a GitHub Actions deployment workflow via the gh CLI.

Examples:
  devcli deploy                                          Interactive selection
  devcli deploy --last                                   Replay last deployment
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
	deployCmd.Flags().BoolVar(&flagLast, "last", false, "Replay last deployment")
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

	// Load history
	hist, _ := history.Load()

	// Replay last deployment
	if flagLast && hist != nil {
		return replayLast(hist)
	}

	// Show history if no flags provided
	if flagRepo == "" && flagWorkflow == "" && flagBranch == "" && hist != nil {
		labels := hist.Labels("deploy")
		if len(labels) > 0 {
			labels = append([]string{"+ New deployment"}, labels...)
			selected, err := ui.Select("Deploy", labels)
			if err != nil {
				os.Exit(0)
			}
			if selected != "+ New deployment" {
				// Extract label before the timestamp
				label := selected[:strings.LastIndex(selected, " (")]
				entry := hist.FindByLabel("deploy", label)
				if entry != nil {
					return executeDeployFromHistory(entry)
				}
			}
		}
	}

	// 1. Select repository
	repo, err := selectRepo()
	if err != nil {
		return err
	}

	// 2. Select workflow
	workflow, workflowName, err := selectDeployWorkflow(repo)
	if err != nil {
		return err
	}

	// 3. Select branch
	branch, err := selectBranch(repo)
	if err != nil {
		return err
	}

	// 4. Trigger
	label := fmt.Sprintf("%s/%s @ %s", repo, workflowName, branch)
	deployArgs := []string{"--repo", repo, "--workflow", workflow, "--branch", branch}
	for _, input := range flagInputs {
		deployArgs = append(deployArgs, "--input", input)
	}

	if err := triggerWorkflow(repo, workflow, branch); err != nil {
		return err
	}

	// Save to history
	if hist != nil {
		hist.Add("deploy", label, deployArgs)
		hist.Save() //nolint:errcheck
	}

	// Watch logs if requested
	if flagWatch {
		return watchLatestRun(repo, workflow)
	}

	return nil
}

func replayLast(hist *history.Store) error {
	labels := hist.Labels("deploy")
	if len(labels) == 0 {
		return fmt.Errorf("no deployment history found")
	}

	// Get the most recent entry
	label := labels[0][:strings.LastIndex(labels[0], " (")]
	entry := hist.FindByLabel("deploy", label)
	if entry == nil {
		return fmt.Errorf("could not find last deployment")
	}

	return executeDeployFromHistory(entry)
}

func executeDeployFromHistory(entry *history.Entry) error {
	var repo, workflow, branch string
	for i := 0; i < len(entry.Args)-1; i += 2 {
		switch entry.Args[i] {
		case "--repo":
			repo = entry.Args[i+1]
		case "--workflow":
			workflow = entry.Args[i+1]
		case "--branch":
			branch = entry.Args[i+1]
		}
	}

	if repo == "" || workflow == "" || branch == "" {
		return fmt.Errorf("incomplete history entry")
	}

	ui.PrintStep("↻", fmt.Sprintf("Replaying: %s", entry.Label))
	if err := triggerWorkflow(repo, workflow, branch); err != nil {
		return err
	}

	if flagWatch {
		return watchLatestRun(repo, workflow)
	}
	return nil
}

func listReposForOwner(owner string) []string {
	args := []string{"repo", "list", "--json", "nameWithOwner", "--limit", "100", "-q", ".[].nameWithOwner"}
	if owner != "" {
		args = append(args, owner)
	}
	out, err := exec.Command("gh", args...).Output()
	if err != nil {
		return nil
	}
	var repos []string
	for _, r := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		r = strings.TrimSpace(r)
		if r != "" {
			repos = append(repos, r)
		}
	}
	return repos
}

func listOwners() []string {
	// Get current user
	userOut, err := exec.Command("gh", "api", "user", "--jq", ".login").Output()
	if err != nil {
		return nil
	}
	user := strings.TrimSpace(string(userOut))

	owners := []string{user}

	// Get organizations
	orgsOut, err := exec.Command("gh", "api", "user/orgs", "--jq", ".[].login").Output()
	if err == nil {
		for _, org := range strings.Split(strings.TrimSpace(string(orgsOut)), "\n") {
			org = strings.TrimSpace(org)
			if org != "" {
				owners = append(owners, org)
			}
		}
	}

	return owners
}

func selectRepo() (string, error) {
	if flagRepo != "" {
		return flagRepo, nil
	}

	// Try to detect from current git repo
	var currentRepo string
	out, err := exec.Command("gh", "repo", "view", "--json", "nameWithOwner", "-q", ".nameWithOwner").Output()
	if err == nil {
		currentRepo = strings.TrimSpace(string(out))
	}

	// List owners (user + orgs)
	owners := listOwners()

	// Select owner first
	var selectedOwner string
	if len(owners) > 1 {
		selected, err := ui.Select("Select owner", owners)
		if err != nil {
			os.Exit(0)
		}
		selectedOwner = selected
	} else if len(owners) == 1 {
		selectedOwner = owners[0]
	}

	// List repos for the selected owner
	repos := listReposForOwner(selectedOwner)
	if len(repos) == 0 {
		return "", fmt.Errorf("no repositories found for %s. Use --repo owner/repo", selectedOwner)
	}

	// Put current repo first if it belongs to the selected owner
	var options []string
	if currentRepo != "" && strings.HasPrefix(currentRepo, selectedOwner+"/") {
		options = append(options, currentRepo+" (current)")
		for _, r := range repos {
			if r != currentRepo {
				options = append(options, r)
			}
		}
	} else {
		options = repos
	}

	selected, err := ui.Select("Select repository", options)
	if err != nil {
		os.Exit(0)
	}

	return strings.TrimSuffix(selected, " (current)"), nil
}

func selectDeployWorkflow(repo string) (fileName, displayName string, err error) {
	if flagWorkflow != "" {
		return flagWorkflow, flagWorkflow, nil
	}

	out, err := exec.Command("gh", "workflow", "list", "--repo", repo, "--json", "name,id,path,state").Output()
	if err != nil {
		return "", "", fmt.Errorf("failed to list workflows: %w", err)
	}

	var workflows []ghWorkflow
	if err := json.Unmarshal(out, &workflows); err != nil {
		return "", "", fmt.Errorf("failed to parse workflows: %w", err)
	}

	var active []ghWorkflow
	for _, w := range workflows {
		if w.State == "active" {
			active = append(active, w)
		}
	}

	if len(active) == 0 {
		return "", "", fmt.Errorf("no active workflows found in %s", repo)
	}

	options := make([]string, len(active))
	for i, w := range active {
		options[i] = fmt.Sprintf("%s (%s)", w.Name, extractWorkflowFile(w.Path))
	}

	selected, err := ui.Select("Select workflow", options)
	if err != nil {
		os.Exit(0)
	}

	for i, opt := range options {
		if opt == selected {
			return extractWorkflowFile(active[i].Path), active[i].Name, nil
		}
	}

	return "", "", fmt.Errorf("workflow not found")
}

func selectBranch(repo string) (string, error) {
	if flagBranch != "" {
		return flagBranch, nil
	}

	// Get branches from the repo
	out, err := exec.Command("gh", "api", fmt.Sprintf("repos/%s/branches", repo),
		"--jq", ".[].name", "--paginate").Output()
	if err != nil {
		// Fallback to manual input
		branch, err := ui.Input("Branch name", "main")
		if err != nil {
			os.Exit(0)
		}
		if branch == "" {
			return "main", nil
		}
		return branch, nil
	}

	branches := strings.Split(strings.TrimSpace(string(out)), "\n")
	var cleaned []string
	for _, b := range branches {
		b = strings.TrimSpace(b)
		if b != "" {
			cleaned = append(cleaned, b)
		}
	}

	if len(cleaned) == 0 {
		return "main", nil
	}

	selected, err := ui.Select("Select branch", cleaned)
	if err != nil {
		os.Exit(0)
	}

	return selected, nil
}

func triggerWorkflow(repo, workflow, branch string) error {
	ghArgs := []string{"workflow", "run", workflow, "--repo", repo, "--ref", branch}

	for _, input := range flagInputs {
		ghArgs = append(ghArgs, "--field", input)
	}

	ui.PrintStep("▶", fmt.Sprintf("Triggering %s on %s (branch: %s)", workflow, repo, branch))

	c := exec.Command("gh", ghArgs...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	if err := c.Run(); err != nil {
		return fmt.Errorf("failed to trigger workflow: %w", err)
	}

	ui.PrintSuccess("Workflow triggered successfully")
	return nil
}

func watchLatestRun(repo, workflow string) error {
	ui.PrintStep("◉", "Waiting for workflow run to start...")

	// Wait a moment for the run to be created
	time.Sleep(3 * time.Second)

	// Get the latest run ID
	out, err := exec.Command("gh", "run", "list",
		"--repo", repo,
		"--workflow", workflow,
		"--limit", "1",
		"--json", "databaseId",
		"-q", ".[0].databaseId").Output()
	if err != nil {
		return fmt.Errorf("failed to get run ID: %w", err)
	}

	runID := strings.TrimSpace(string(out))
	if runID == "" {
		return fmt.Errorf("no run found")
	}

	ui.PrintStep("◉", fmt.Sprintf("Streaming logs for run #%s", runID))
	fmt.Println(ui.BoxStyle.Render("Press Ctrl+C to stop watching"))
	fmt.Println()

	c := exec.Command("gh", "run", "watch", runID, "--repo", repo, "--exit-status")
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	if err := c.Run(); err != nil {
		ui.PrintError(fmt.Sprintf("Workflow run failed (run #%s)", runID))
		fmt.Printf("\nView full logs: gh run view %s --repo %s --log\n", runID, repo)
		return err
	}

	ui.PrintSuccess(fmt.Sprintf("Workflow run #%s completed successfully", runID))
	return nil
}

func extractWorkflowFile(path string) string {
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}
