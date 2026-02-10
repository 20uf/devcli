package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/20uf/devcli/internal/history"
	"github.com/20uf/devcli/internal/ui"
	"github.com/20uf/devcli/internal/verbose"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
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

type repoInfo struct {
	NameWithOwner string `json:"nameWithOwner"`
	Description   string `json:"description"`
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
			if len(labels) > 10 {
				labels = labels[:10]
			}
			labels = append([]string{"+ New deployment"}, labels...)
			selected, err := ui.Select("Deploy", labels)
			if err != nil {
				return err
			}
			if selected != "+ New deployment" {
				label := selected[:strings.LastIndex(selected, " (")]
				entry := hist.FindByLabel("deploy", label)
				if entry != nil {
					return executeDeployFromHistory(entry)
				}
			}
		}
	}

	// Step-based navigation: ESC goes back to previous step
	var owner, repo, workflow, workflowName, branch string
	var workflowInputValues []string

	step := 0
	if flagRepo != "" {
		repo = flagRepo
		step = 2 // skip owner + repo selection
	}

	for {
		switch step {
		case 0: // Select owner
			o, err := selectOwner()
			if err != nil {
				return err // ESC → back to home
			}
			owner = o
			step++

		case 1: // Select repo
			r, err := selectRepoForOwner(owner)
			if err != nil {
				step = 0 // ESC → back to owner
				continue
			}
			repo = r
			step++

		case 2: // Select workflow
			w, wn, err := selectDeployWorkflow(repo)
			if err != nil {
				if flagRepo != "" {
					return err // can't go back if repo was a flag
				}
				step = 1 // ESC → back to repo
				continue
			}
			workflow = w
			workflowName = wn
			step++

		case 3: // Workflow inputs (if any)
			if len(flagInputs) > 0 {
				// Inputs provided via flags, skip interactive
				workflowInputValues = flagInputs
				step++
				continue
			}

			inputs, err := fetchWorkflowInputs(repo, workflow)
			if err != nil {
				verbose.Log("could not fetch workflow inputs: %s", err)
				// Not fatal — workflow may not have inputs
				workflowInputValues = nil
				step++
				continue
			}

			if len(inputs) == 0 {
				workflowInputValues = nil
				step++
				continue
			}

			ui.PrintStep("◆", "Workflow inputs")
			values, err := promptWorkflowInputs(inputs)
			if err != nil {
				step = 2 // ESC → back to workflow
				continue
			}
			workflowInputValues = values
			step++

		case 4: // Select branch
			b, err := selectBranch(repo)
			if err != nil {
				step = 3 // ESC → back to inputs
				continue
			}
			branch = b
			step++

		case 5: // Trigger
			label := fmt.Sprintf("%s/%s @ %s", repo, workflowName, branch)
			deployArgs := []string{"--repo", repo, "--workflow", workflow, "--branch", branch}
			for _, input := range workflowInputValues {
				deployArgs = append(deployArgs, "--input", input)
			}

			if err := triggerWorkflowWithInputs(repo, workflow, branch, workflowInputValues); err != nil {
				return err
			}

			if hist != nil {
				hist.Add("deploy", label, deployArgs)
				hist.Save() //nolint:errcheck
			}

			if flagWatch {
				return watchLatestRun(repo, workflow)
			}
			return nil
		}
	}
}

func selectOwner() (string, error) {
	owners := listOwners()
	if len(owners) == 0 {
		return "", fmt.Errorf("could not determine GitHub user/orgs")
	}
	if len(owners) == 1 {
		return owners[0], nil
	}
	return ui.Select("Select owner", owners)
}

func selectRepoForOwner(owner string) (string, error) {
	ui.PrintStep("◆", fmt.Sprintf("Organization: %s", owner))

	// Try to detect current repo
	var currentRepo string
	out, err := verbose.Cmd(exec.Command("gh", "repo", "view", "--json", "nameWithOwner", "-q", ".nameWithOwner")).Output()
	if err == nil {
		currentRepo = strings.TrimSpace(string(out))
	}

	repos, err := listReposForOwner(owner)
	if err != nil || len(repos) == 0 {
		ui.PrintWarning(fmt.Sprintf("Could not list repositories for %s", owner))
		// Use Select with manual entry option so ESC works for back navigation
		manualChoice, err := ui.Select("Repository not listed", []string{"Enter repository manually"})
		if err != nil {
			return "", err // ESC → back to owner
		}
		if manualChoice == "Enter repository manually" {
			repo, err := ui.Input("Repository (owner/repo)", owner+"/")
			if err != nil {
				return "", err
			}
			if repo == "" {
				return "", fmt.Errorf("no repository specified")
			}
			return repo, nil
		}
	}

	// Build options: strip owner prefix, add description
	prefix := owner + "/"
	maxNameLen := 0
	for _, r := range repos {
		name := strings.TrimPrefix(r.NameWithOwner, prefix)
		if r.NameWithOwner == currentRepo {
			name += " *"
		}
		if len(name) > maxNameLen {
			maxNameLen = len(name)
		}
	}

	var options []ui.SelectOption
	for _, r := range repos {
		name := strings.TrimPrefix(r.NameWithOwner, prefix)
		if r.NameWithOwner == currentRepo {
			name += " *"
		}
		display := name
		if r.Description != "" {
			desc := r.Description
			if len(desc) > 50 {
				desc = desc[:47] + "..."
			}
			display = fmt.Sprintf("%-*s  %s", maxNameLen+1, name, desc)
		}
		options = append(options, ui.SelectOption{
			Display: display,
			Value:   r.NameWithOwner,
		})
	}

	return ui.SelectWithOptions("Select repository", options)
}

func replayLast(hist *history.Store) error {
	labels := hist.Labels("deploy")
	if len(labels) == 0 {
		return fmt.Errorf("no deployment history found")
	}

	label := labels[0][:strings.LastIndex(labels[0], " (")]
	entry := hist.FindByLabel("deploy", label)
	if entry == nil {
		return fmt.Errorf("could not find last deployment")
	}

	return executeDeployFromHistory(entry)
}

func executeDeployFromHistory(entry *history.Entry) error {
	var repo, workflow, branch string
	var inputs []string
	for i := 0; i < len(entry.Args)-1; i += 2 {
		switch entry.Args[i] {
		case "--repo":
			repo = entry.Args[i+1]
		case "--workflow":
			workflow = entry.Args[i+1]
		case "--branch":
			branch = entry.Args[i+1]
		case "--input":
			inputs = append(inputs, entry.Args[i+1])
		}
	}

	if repo == "" || workflow == "" || branch == "" {
		return fmt.Errorf("incomplete history entry")
	}

	ui.PrintStep("↻", fmt.Sprintf("Replaying: %s", entry.Label))
	if err := triggerWorkflowWithInputs(repo, workflow, branch, inputs); err != nil {
		return err
	}

	if flagWatch {
		return watchLatestRun(repo, workflow)
	}
	return nil
}

func listReposForOwner(owner string) ([]repoInfo, error) {
	args := []string{"repo", "list", "--json", "nameWithOwner,description", "--limit", "10"}
	if owner != "" {
		args = append(args, owner)
	}
	out, err := verbose.Cmd(exec.Command("gh", args...)).Output()
	if err != nil {
		return nil, err
	}
	var repos []repoInfo
	if err := json.Unmarshal(out, &repos); err != nil {
		return nil, err
	}
	return repos, nil
}

func listOwners() []string {
	userOut, err := verbose.Cmd(exec.Command("gh", "api", "user", "--jq", ".login")).Output()
	if err != nil {
		return nil
	}
	user := strings.TrimSpace(string(userOut))

	owners := []string{user}

	orgsOut, err := verbose.Cmd(exec.Command("gh", "api", "user/orgs", "--jq", ".[].login")).Output()
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

func selectDeployWorkflow(repo string) (fileName, displayName string, err error) {
	if flagWorkflow != "" {
		return flagWorkflow, flagWorkflow, nil
	}

	out, err := verbose.Cmd(exec.Command("gh", "workflow", "list", "--repo", repo, "--json", "name,id,path,state")).Output()
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
		return "", "", err
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

	out, err := verbose.Cmd(exec.Command("gh", "api", fmt.Sprintf("repos/%s/branches", repo),
		"--jq", ".[].name", "--paginate")).Output()
	if err != nil {
		branch, err := ui.Input("Branch name", "main")
		if err != nil {
			return "", err
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

	return ui.Select("Select branch", cleaned)
}

func triggerWorkflowWithInputs(repo, workflow, branch string, inputs []string) error {
	ghArgs := []string{"workflow", "run", workflow, "--repo", repo, "--ref", branch}

	for _, input := range inputs {
		ghArgs = append(ghArgs, "--field", input)
	}

	ui.PrintStep("▶", fmt.Sprintf("Triggering %s on %s (branch: %s)", workflow, repo, branch))

	c := verbose.Cmd(exec.Command("gh", ghArgs...))
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

	time.Sleep(3 * time.Second)

	out, err := verbose.Cmd(exec.Command("gh", "run", "list",
		"--repo", repo,
		"--workflow", workflow,
		"--limit", "1",
		"--json", "databaseId",
		"-q", ".[0].databaseId")).Output()
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

	watchCmd := verbose.Cmd(exec.Command("gh", "run", "watch", runID, "--repo", repo, "--exit-status"))
	watchCmd.Stdin = os.Stdin
	watchCmd.Stdout = os.Stdout
	watchCmd.Stderr = os.Stderr

	if err := watchCmd.Run(); err != nil {
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

// workflowInput represents a single input from workflow_dispatch.
type workflowInput struct {
	Description string   `yaml:"description"`
	Required    bool     `yaml:"required"`
	Default     string   `yaml:"default"`
	Type        string   `yaml:"type"`
	Options     []string `yaml:"options"`
}

// workflowFile represents the relevant parts of a GitHub Actions workflow YAML.
type workflowFile struct {
	On struct {
		WorkflowDispatch struct {
			Inputs map[string]workflowInput `yaml:"inputs"`
		} `yaml:"workflow_dispatch"`
	} `yaml:"on"`
}

// fetchWorkflowInputs retrieves the workflow file from GitHub and parses its inputs.
func fetchWorkflowInputs(repo, workflowFileName string) (map[string]workflowInput, error) {
	path := fmt.Sprintf(".github/workflows/%s", workflowFileName)
	verbose.Log("fetching workflow file: %s from %s", path, repo)

	out, err := verbose.Cmd(exec.Command("gh", "api",
		fmt.Sprintf("repos/%s/contents/%s", repo, path),
		"--jq", ".content")).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch workflow file: %w", err)
	}

	content := strings.TrimSpace(string(out))
	decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(content, "\n", ""))
	if err != nil {
		return nil, fmt.Errorf("failed to decode workflow file: %w", err)
	}

	var wf workflowFile
	if err := yaml.Unmarshal(decoded, &wf); err != nil {
		return nil, fmt.Errorf("failed to parse workflow YAML: %w", err)
	}

	return wf.On.WorkflowDispatch.Inputs, nil
}

// promptWorkflowInputs interactively prompts the user for each workflow input.
func promptWorkflowInputs(inputs map[string]workflowInput) ([]string, error) {
	if len(inputs) == 0 {
		return nil, nil
	}

	// Collect input names in a stable order
	var names []string
	for name := range inputs {
		names = append(names, name)
	}
	// Sort for consistent ordering
	sort.Strings(names)

	var result []string
	for _, name := range names {
		input := inputs[name]
		label := name
		if input.Description != "" {
			label = fmt.Sprintf("%s (%s)", name, input.Description)
		}

		var value string
		var err error

		if input.Type == "choice" && len(input.Options) > 0 {
			// Show select for choice inputs
			options := input.Options
			value, err = ui.Select(label, options)
		} else if input.Type == "boolean" {
			confirmed, confirmErr := ui.Confirm(label)
			if confirmErr != nil {
				return nil, confirmErr
			}
			if confirmed {
				value = "true"
			} else {
				value = "false"
			}
			err = nil
		} else {
			// Text input with default as placeholder
			placeholder := input.Default
			if placeholder == "" {
				placeholder = ""
			}
			value, err = ui.Input(label, placeholder)
		}

		if err != nil {
			return nil, err
		}

		if value == "" && input.Default != "" {
			value = input.Default
		}

		if value != "" {
			result = append(result, fmt.Sprintf("%s=%s", name, value))
		}
	}

	return result, nil
}

