package cmd

import (
	"fmt"

	"github.com/20uf/devcli/internal/updater"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update devcli to the latest version",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Checking for updates...")

		latest, hasUpdate, err := updater.Check(appVersion)
		if err != nil {
			return fmt.Errorf("failed to check for updates: %w", err)
		}

		if !hasUpdate {
			fmt.Printf("Already up to date (%s)\n", appVersion)
			return nil
		}

		fmt.Printf("New version available: %s (current: %s)\n", latest, appVersion)

		if err := updater.Apply(latest); err != nil {
			return fmt.Errorf("failed to update: %w", err)
		}

		fmt.Printf("Updated to %s successfully!\n", latest)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
