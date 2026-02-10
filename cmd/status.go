package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/20uf/devcli/internal/tracker"
	"github.com/20uf/devcli/internal/ui"
	"github.com/20uf/devcli/internal/verbose"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Live dashboard for tracked workflow runs",
	Long: `View and follow your deployment runs in real time.

Tracked runs persist across sessions. After triggering a deploy, it appears
here automatically. You can watch logs, open in browser, or dismiss runs.

Examples:
  devcli status             Open the live dashboard`,
	RunE: runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("GitHub CLI (gh) is required.\n  Install: https://cli.github.com/")
	}

	store, err := tracker.Load()
	if err != nil {
		return fmt.Errorf("failed to load tracker: %w", err)
	}

	store.Cleanup()

	if len(store.Runs) == 0 {
		ui.PrintWarning("No tracked deployments")
		fmt.Println(ui.MutedStyle.Render("  Trigger a deploy with `devcli deploy` — it will appear here automatically."))
		return nil
	}

	return showDashboard(store)
}

func showDashboard(store *tracker.Store) error {
	for {
		// Refresh statuses from GitHub
		refreshRunStatuses(store)
		store.Save() //nolint:errcheck

		runs := store.All()
		if len(runs) == 0 {
			ui.PrintSuccess("All deployments completed!")
			return nil
		}

		// Build options
		options := make([]ui.SelectOption, 0, len(runs)+2)
		for _, r := range runs {
			icon := runStatusIcon(r.Status, r.Conclusion)
			elapsed := time.Since(r.StartedAt).Truncate(time.Second)
			display := fmt.Sprintf("%s  %s  (%s)  %s", icon, r.Label, r.Branch, ui.MutedStyle.Render(elapsed.String()))
			options = append(options, ui.SelectOption{
				Display: display,
				Value:   r.RunID,
			})
		}
		options = append(options, ui.SelectOption{
			Display: ui.MutedStyle.Render("↻  Refresh"),
			Value:   "__refresh",
		})
		options = append(options, ui.SelectOption{
			Display: ui.MutedStyle.Render("←  Back"),
			Value:   "__back",
		})

		selected, err := ui.SelectWithOptions("Tracked Deployments", options)
		if err != nil {
			return nil // ESC → back to home
		}

		if selected == "__refresh" {
			fmt.Print("\r\033[K")
			continue
		}
		if selected == "__back" {
			return nil
		}

		// Find the selected run
		var run *tracker.Run
		for i := range runs {
			if runs[i].RunID == selected {
				run = &runs[i]
				break
			}
		}
		if run == nil {
			continue
		}

		// Show actions for this run
		actionErr := showRunActions(store, run)
		if actionErr != nil {
			continue // ESC → back to list
		}
	}
}

func showRunActions(store *tracker.Store, run *tracker.Run) error {
	actions := []string{"Stream logs (watch)", "View in browser"}

	if run.Status == "completed" {
		actions = append(actions, "View full logs")
	}
	actions = append(actions, "Dismiss (stop tracking)")
	actions = append(actions, "Back to dashboard")

	action, err := ui.Select(fmt.Sprintf("Run #%s", run.RunID), actions)
	if err != nil {
		return err
	}

	switch action {
	case "Stream logs (watch)":
		c := verbose.Cmd(exec.Command("gh", "run", "watch", run.RunID, "--repo", run.Repo, "--exit-status"))
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr

		if err := c.Run(); err != nil {
			ui.PrintError(fmt.Sprintf("Workflow run failed (#%s)", run.RunID))
		} else {
			ui.PrintSuccess(fmt.Sprintf("Workflow run #%s completed!", run.RunID))
		}
		// Refresh status after watching
		refreshSingleRun(store, run.RunID, run.Repo)
		store.Save() //nolint:errcheck

	case "View in browser":
		verbose.Cmd(exec.Command("gh", "run", "view", run.RunID, "--repo", run.Repo, "--web")).Run() //nolint:errcheck

	case "View full logs":
		c := verbose.Cmd(exec.Command("gh", "run", "view", run.RunID, "--repo", run.Repo, "--log"))
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		c.Run() //nolint:errcheck

	case "Dismiss (stop tracking)":
		store.Remove(run.RunID)
		store.Save() //nolint:errcheck
		ui.PrintStep("⊘", "Run dismissed")

	case "Back to dashboard":
		// no-op, will loop
	}

	return nil
}

func refreshRunStatuses(store *tracker.Store) {
	for i := range store.Runs {
		r := &store.Runs[i]
		if r.Status == "completed" {
			continue
		}
		refreshSingleRun(store, r.RunID, r.Repo)
	}
}

func refreshSingleRun(store *tracker.Store, runID, repo string) {
	out, err := verbose.Cmd(exec.Command("gh", "run", "view", runID,
		"--repo", repo,
		"--json", "status,conclusion")).Output()
	if err != nil {
		return
	}

	var result struct {
		Status     string `json:"status"`
		Conclusion string `json:"conclusion"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return
	}

	store.Update(runID, result.Status, result.Conclusion)
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

// findLatestRunID finds the most recent run ID for a workflow after trigger.
func findLatestRunID(repo, workflow string) (string, error) {
	// Wait a moment for the run to appear
	time.Sleep(2 * time.Second)

	out, err := verbose.Cmd(exec.Command("gh", "run", "list",
		"--repo", repo,
		"--workflow", workflow,
		"--limit", "1",
		"--json", "databaseId",
		"-q", ".[0].databaseId")).Output()
	if err != nil {
		return "", err
	}

	id := strings.TrimSpace(string(out))
	if id == "" {
		return "", fmt.Errorf("no run found")
	}
	return id, nil
}
