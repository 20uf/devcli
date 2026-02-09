package cmd

import (
	"fmt"
	"os"
	"sync"

	"github.com/20uf/devcli/internal/updater"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "devcli",
	Short: "CLI for interactive access to AWS ECS containers",
	Long:  "A developer CLI to dynamically discover and connect to AWS ECS Fargate containers.",
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
	// Check pre-releases first (since we're likely on a pre-release),
	// then stable releases
	latest, hasUpdate, err := updater.Check(appVersion, true)
	if err != nil || !hasUpdate {
		return
	}

	updateOnce.Do(func() {
		updateNotice = fmt.Sprintf(
			"\nA new version of devcli is available: %s (current: %s)\nRun \"devcli update --pre-release\" to update.\n",
			latest, appVersion,
		)
	})
}
