package cmd

import (
	"fmt"
	"os"
	"sync"

	"github.com/20uf/devcli/internal/ui"
	"github.com/20uf/devcli/internal/updater"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "devcli",
	Short: "CLI for interactive access to AWS ECS containers",
	Long: `A developer CLI to dynamically discover and connect to AWS ECS Fargate containers.

Available commands:
  connect       Connect to an ECS container interactively
  deploy        Trigger a GitHub Actions deployment workflow
  update        Update devcli to the latest version
  version       Print version information
  completion    Generate or install shell completion

Use "devcli <command> --help" for more information about a command.`,
	Run: func(cmd *cobra.Command, args []string) {
		ui.PrintBanner(appVersion)
		cmd.Help() //nolint:errcheck
	},
}

var (
	updateNotice string
	updateOnce   sync.Once
)

func Execute() {
	// Non-blocking update check in background
	var wg sync.WaitGroup
	if appVersion != "dev" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			checkForUpdate()
		}()
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Wait for update check and display notice if available
	wg.Wait()
	if updateNotice != "" {
		fmt.Fprintln(os.Stderr, updateNotice)
	}
}

func checkForUpdate() {
	latest, hasUpdate, err := updater.Check(appVersion, true)
	if err != nil || !hasUpdate {
		return
	}

	updateOnce.Do(func() {
		updateNotice = fmt.Sprintf(
			"\n%s %s → %s\n%s\n",
			ui.WarningStyle.Render("Update available:"),
			ui.MutedStyle.Render(appVersion),
			ui.SuccessStyle.Render(latest),
			ui.MutedStyle.Render("Run \"devcli update --pre-release\" to update."),
		)
	})
}
