package cmd

import (
	"errors"
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
		showHome(cmd)
	},
}

var (
	updateNotice string
	updateOnce   sync.Once
)

func showHome(cmd *cobra.Command) {
	// Print banner with inline update check
	var checkFn func() (string, bool, error)
	if appVersion != "dev" {
		checkFn = func() (string, bool, error) {
			return updater.Check(appVersion, false)
		}
	}

	result := ui.PrintBannerWithUpdateCheck(appVersion, checkFn)

	// If update available, invite user to update
	if result != nil && result.HasUpdate {
		confirmed, err := ui.Confirm(fmt.Sprintf("Update to v%s?", result.Latest))
		if err == nil && confirmed {
			fmt.Println()
			if err := updater.Apply(result.Latest); err != nil {
				ui.PrintError(fmt.Sprintf("Update failed: %s", err))
			} else {
				ui.PrintSuccess(fmt.Sprintf("Updated to v%s!", result.Latest))
			}
			fmt.Println()
		}
	}

	// Interactive command selection loop
	commands := []ui.SelectOption{
		{Display: "connect    Connect to an ECS container interactively", Value: "connect"},
		{Display: "deploy     Trigger a GitHub Actions deployment workflow", Value: "deploy"},
		{Display: "update     Update devcli to the latest version", Value: "update"},
		{Display: "version    Print version information", Value: "version"},
	}

	for {
		selected, err := ui.SelectWithOptions("Available Commands", commands)
		if err != nil {
			return // ESC at home = exit
		}

		fmt.Println()

		subcmd, _, findErr := cmd.Root().Find([]string{selected})
		if findErr != nil {
			ui.PrintError(fmt.Sprintf("Command not found: %s", selected))
			continue
		}

		var runErr error
		if subcmd.RunE != nil {
			runErr = subcmd.RunE(subcmd, []string{})
		} else if subcmd.Run != nil {
			subcmd.Run(subcmd, []string{})
		}

		if runErr != nil && !errors.Is(runErr, ui.ErrUserAbort) {
			ui.PrintError(runErr.Error())
		}

		fmt.Println()
	}
}

func Execute() {
	// Background update check only for direct subcommand usage
	var wg sync.WaitGroup
	if appVersion != "dev" && len(os.Args) > 1 {
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

	wg.Wait()
	if updateNotice != "" {
		fmt.Fprintln(os.Stderr, updateNotice)
	}
}

func checkForUpdate() {
	latest, hasUpdate, err := updater.Check(appVersion, false)
	if err != nil || !hasUpdate {
		return
	}

	updateOnce.Do(func() {
		updateNotice = fmt.Sprintf(
			"\n%s %s → %s\n%s\n",
			ui.WarningStyle.Render("Update available:"),
			ui.MutedStyle.Render(appVersion),
			ui.SuccessStyle.Render(latest),
			ui.MutedStyle.Render("Run \"devcli update\" to update."),
		)
	})
}
