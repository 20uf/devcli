package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/20uf/devcli/internal/ui"
	"github.com/20uf/devcli/internal/verbose"
	"github.com/spf13/cobra"
)

var flagStatusRepo string

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "View GitHub Actions workflow run status",
	Long: `View recent GitHub Actions workflow runs and stream their logs.

Examples:
  devcli status                              Interactive selection
  devcli status --repo owner/repo            Skip repo selection`,
	RunE: runStatus,
}

func init() {
	statusCmd.Flags().StringVar(&flagStatusRepo, "repo", "", "GitHub repository (owner/repo)")
	rootCmd.AddCommand(statusCmd)
}

type ghRun struct {
	DatabaseID int    `json:"databaseId"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	Branch     string `json:"headBranch"`
	Event      string `json:"event"`
	Number     int    `json:"number"`
	URL        string `json:"url"`
	CreatedAt  string `json:"createdAt"`
}

func runStatus(cmd *cobra.Command, args []string) error {
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("GitHub CLI (gh) is required.\n  Install: https://cli.github.com/")
	}

	var repo string

	step := 0
	if flagStatusRepo != "" {
		repo = flagStatusRepo
		step = 2
	}

	for {
		switch step {
		case 0: // Select owner
			o, err := selectOwner()
			if err != nil {
				return err
			}
			_ = o
			step++
			// We need the owner for repo selection, reuse from deploy
			repo = "" // will be set in step 1

			// Actually, re-select repo via owner
			r, err := selectRepoForOwner(o)
			if err != nil {
				step = 0
				continue
			}
			repo = r
			step = 2
			continue

		case 2: // List runs
			runs, err := listWorkflowRuns(repo)
			if err != nil {
				return fmt.Errorf("failed to list workflow runs: %w", err)
			}

			if len(runs) == 0 {
				ui.PrintWarning("No recent workflow runs found")
				return nil
			}

			options := make([]ui.SelectOption, len(runs))
			for i, r := range runs {
				icon := runStatusIcon(r.Status, r.Conclusion)
				display := fmt.Sprintf("%s  #%d %s (%s) [%s]",
					icon, r.Number, r.Name, r.Branch, r.CreatedAt[:10])
				options[i] = ui.SelectOption{
					Display: display,
					Value:   fmt.Sprintf("%d", r.DatabaseID),
				}
			}

			selected, err := ui.SelectWithOptions("Recent workflow runs", options)
			if err != nil {
				if flagStatusRepo != "" {
					return err
				}
				step = 0
				continue
			}

			// Show run details and offer actions
			step = 3
			_ = selected

			action, err := ui.Select("Action", []string{
				"Stream logs (watch)",
				"View in browser",
				"Back to runs",
			})
			if err != nil {
				step = 2
				continue
			}

			switch action {
			case "Stream logs (watch)":
				verbose.Log("gh run watch %s --repo %s", selected, repo)
				c := verbose.Cmd(exec.Command("gh", "run", "watch", selected, "--repo", repo, "--exit-status"))
				c.Stdin = os.Stdin
				c.Stdout = os.Stdout
				c.Stderr = os.Stderr

				if err := c.Run(); err != nil {
					ui.PrintError(fmt.Sprintf("Workflow run failed (#%s)", selected))
					fmt.Printf("\nView full logs: gh run view %s --repo %s --log\n", selected, repo)
				} else {
					ui.PrintSuccess(fmt.Sprintf("Workflow run #%s completed successfully", selected))
				}
				return nil

			case "View in browser":
				verbose.Cmd(exec.Command("gh", "run", "view", selected, "--repo", repo, "--web")).Run() //nolint:errcheck
				return nil

			case "Back to runs":
				step = 2
				continue
			}

		default:
			return nil
		}
	}
}

func listWorkflowRuns(repo string) ([]ghRun, error) {
	out, err := verbose.Cmd(exec.Command("gh", "run", "list",
		"--repo", repo,
		"--limit", "15",
		"--json", "databaseId,name,status,conclusion,headBranch,event,number,url,createdAt")).Output()
	if err != nil {
		return nil, err
	}

	var runs []ghRun
	if err := json.Unmarshal(out, &runs); err != nil {
		return nil, err
	}
	return runs, nil
}

func runStatusIcon(status, conclusion string) string {
	switch {
	case status == "in_progress":
		return ui.WarningStyle.Render("◉")
	case status == "queued" || status == "waiting":
		return ui.MutedStyle.Render("○")
	case conclusion == "success":
		return ui.SuccessStyle.Render("✓")
	case conclusion == "failure":
		return ui.ErrorStyle.Render("✗")
	case conclusion == "cancelled":
		return ui.MutedStyle.Render("⊘")
	default:
		return ui.MutedStyle.Render("·")
	}
}

func extractRunID(display string) string {
	// Extract #NNN from display
	start := strings.Index(display, "#")
	if start < 0 {
		return display
	}
	end := strings.Index(display[start:], " ")
	if end < 0 {
		return display[start+1:]
	}
	return display[start+1 : start+end]
}
