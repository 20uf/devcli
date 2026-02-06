package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "devcli",
	Short: "CLI for interactive access to AWS ECS containers",
	Long:  "A developer CLI to dynamically discover and connect to AWS ECS Fargate containers.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
