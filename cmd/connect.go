package cmd

import (
	"fmt"

	awsutil "github.com/20uf/devcli/internal/aws"
	"github.com/spf13/cobra"
)

var (
	flagCluster   string
	flagService   string
	flagContainer string
	flagShell     string
	flagProfile   string
	flagRegion    string
)

var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Connect to an ECS container interactively",
	Long: `Discover ECS clusters, services, tasks and containers dynamically, then open an interactive shell.

Examples:
  devcli connect                                         Interactive selection
  devcli connect --profile dev --cluster my-cluster      Partial flags
  devcli connect --profile dev --cluster c --service s   Full non-interactive
  devcli connect --shell /bin/bash                       Custom shell`,
	RunE: runConnect,
}

func init() {
	connectCmd.Flags().StringVar(&flagCluster, "cluster", "", "ECS cluster name or ARN (skip selection)")
	connectCmd.Flags().StringVar(&flagService, "service", "", "ECS service name (skip selection)")
	connectCmd.Flags().StringVar(&flagContainer, "container", "", "Container name (skip selection)")
	connectCmd.Flags().StringVar(&flagShell, "shell", "", "Shell command (default: auto-detect)")
	connectCmd.Flags().StringVar(&flagProfile, "profile", "", "AWS profile to use")
	connectCmd.Flags().StringVar(&flagRegion, "region", "", "AWS region to use")
	rootCmd.AddCommand(connectCmd)
}

func runConnect(cmd *cobra.Command, args []string) error {
	if err := awsutil.CheckDependencies(); err != nil {
		return err
	}

	// Create handler (wires all dependencies: domain + repos + UI)
	handler, err := NewConnectHandler(cmd.Context(), flagProfile, flagRegion)
	if err != nil {
		return fmt.Errorf("failed to initialize connection handler: %w", err)
	}

	// Orchestrate the connection flow
	// Handler manages: cluster selection → service → task → container → execution
	return handler.Handle(cmd, flagCluster, flagService, flagContainer, flagShell)
}
