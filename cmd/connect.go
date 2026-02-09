package cmd

import (
	"fmt"
	"os"

	awsutil "github.com/20uf/devcli/internal/aws"
	"github.com/20uf/devcli/internal/ecs"
	"github.com/20uf/devcli/internal/ui"
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
	Long:  "Discover ECS clusters, services, tasks and containers dynamically, then open an interactive shell.",
	RunE:  runConnect,
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
	// 0. Check required dependencies
	if err := awsutil.CheckDependencies(); err != nil {
		return err
	}

	// 1. Select AWS profile
	profile, err := selectProfile()
	if err != nil {
		return err
	}

	// 2. Ensure SSO session is valid (auto-login if expired)
	if err := awsutil.EnsureSSOLogin(profile); err != nil {
		return err
	}

	client, err := ecs.NewClient(profile, flagRegion)
	if err != nil {
		return fmt.Errorf("failed to create AWS client: %w", err)
	}

	// 1. Select cluster
	cluster, err := selectCluster(client)
	if err != nil {
		return err
	}

	// 2. Select service
	service, err := selectService(client, cluster)
	if err != nil {
		return err
	}

	// 3. Get a running task
	task, err := client.GetRunningTask(cmd.Context(), cluster, service)
	if err != nil {
		return fmt.Errorf("no running task found: %w", err)
	}

	// 4. Select container
	container, err := selectContainer(client, cmd, cluster, task)
	if err != nil {
		return err
	}

	// 5. Determine shell
	shell := resolveShell()

	// 6. Execute
	fmt.Printf("Connecting to %s/%s/%s (%s)...\n", cluster, service, container, shell)
	return client.ExecInteractive(cmd.Context(), cluster, task, container, shell, profile)
}

func selectCluster(client *ecs.Client) (string, error) {
	if flagCluster != "" {
		return flagCluster, nil
	}

	clusters, err := client.ListClusters(rootCmd.Context())
	if err != nil {
		return "", fmt.Errorf("failed to list clusters: %w", err)
	}

	if len(clusters) == 0 {
		return "", fmt.Errorf("no ECS clusters found")
	}

	selected, err := ui.Select("Select cluster", clusters)
	if err != nil {
		os.Exit(0)
	}

	return selected, nil
}

func selectService(client *ecs.Client, cluster string) (string, error) {
	if flagService != "" {
		return flagService, nil
	}

	services, err := client.ListServices(rootCmd.Context(), cluster)
	if err != nil {
		return "", fmt.Errorf("failed to list services: %w", err)
	}

	if len(services) == 0 {
		return "", fmt.Errorf("no services found in cluster %s", cluster)
	}

	selected, err := ui.Select("Select service", services)
	if err != nil {
		os.Exit(0)
	}

	return selected, nil
}

func selectContainer(client *ecs.Client, cmd *cobra.Command, cluster, task string) (string, error) {
	if flagContainer != "" {
		return flagContainer, nil
	}

	containers, err := client.ListContainers(cmd.Context(), cluster, task)
	if err != nil {
		return "", fmt.Errorf("failed to list containers: %w", err)
	}

	if len(containers) == 0 {
		return "", fmt.Errorf("no containers found in task %s", task)
	}

	// Auto-select "php" if present
	for _, c := range containers {
		if c == "php" {
			fmt.Println("Auto-selected container: php")
			return "php", nil
		}
	}

	if len(containers) == 1 {
		fmt.Printf("Auto-selected container: %s\n", containers[0])
		return containers[0], nil
	}

	selected, err := ui.Select("Select container", containers)
	if err != nil {
		os.Exit(0)
	}

	return selected, nil
}

func selectProfile() (string, error) {
	if flagProfile != "" {
		return flagProfile, nil
	}

	profiles, err := awsutil.ListProfiles()
	if err != nil {
		return "", fmt.Errorf("failed to list AWS profiles: %w", err)
	}

	if len(profiles) == 0 {
		return "", fmt.Errorf("no AWS profiles found in ~/.aws/config")
	}

	if len(profiles) == 1 {
		fmt.Printf("Using AWS profile: %s\n", profiles[0])
		return profiles[0], nil
	}

	selected, err := ui.Select("Select AWS profile", profiles)
	if err != nil {
		os.Exit(0)
	}

	return selected, nil
}

func resolveShell() string {
	if flagShell != "" {
		return flagShell
	}
	return "su -s /bin/sh www-data"
}
