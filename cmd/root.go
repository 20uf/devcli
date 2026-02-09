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
	Short: "Focus on coding, not on tooling.",
	Long:  `Devcli is a modular CLI toolbox to manage your dev environment, workflows, and infrastructure interactions.`,
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
