package cmd

import (
	"fmt"
	"strings"

	awsutil "github.com/20uf/devcli/internal/aws"
	"github.com/20uf/devcli/internal/ecs"
	"github.com/20uf/devcli/internal/history"
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
	Long: `Discover ECS clusters, services, tasks and containers dynamically, then open an interactive shell.

Examples:
  devcli connect                                         Interactive selection
  devcli connect --profile dev --cluster my-cluster      Partial flags
  devcli connect --profile dev --cluster c --service s   Full non-interactive
  devcli connect --shell /bin/bash                       Custom shell`,
	RunE: runConnect,
}

var flagConnectLast bool

func init() {
	connectCmd.Flags().StringVar(&flagCluster, "cluster", "", "ECS cluster name or ARN (skip selection)")
	connectCmd.Flags().StringVar(&flagService, "service", "", "ECS service name (skip selection)")
	connectCmd.Flags().StringVar(&flagContainer, "container", "", "Container name (skip selection)")
	connectCmd.Flags().StringVar(&flagShell, "shell", "", "Shell command (default: auto-detect)")
	connectCmd.Flags().StringVar(&flagProfile, "profile", "", "AWS profile to use")
	connectCmd.Flags().StringVar(&flagRegion, "region", "", "AWS region to use")
	connectCmd.Flags().BoolVar(&flagConnectLast, "last", false, "Replay last connection")
	rootCmd.AddCommand(connectCmd)
}

func runConnect(cmd *cobra.Command, args []string) error {
	// 0. Check required dependencies
	if err := awsutil.CheckDependencies(); err != nil {
		return err
	}

	// Replay last or show history
	if flagConnectLast {
		return replayLastConnect()
	}

	// Show history if no flags
	if flagProfile == "" && flagCluster == "" && flagService == "" {
		entry, err := showConnectHistory()
		if err != nil {
			return err
		}
		if entry != nil {
			return replayConnectEntry(entry)
		}
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

	// 6. Save to history
	hist, _ := history.Load()
	if hist != nil {
		label := fmt.Sprintf("%s → %s/%s/%s", profile, cluster, service, container)
		hist.Add("connect", label, []string{
			"--profile", profile, "--cluster", cluster,
			"--service", service, "--container", container,
		})
		hist.Save() //nolint:errcheck
	}

	// 7. Execute
	ui.PrintStep("▶", fmt.Sprintf("Connecting to %s/%s/%s", cluster, service, container))
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
		return "", err
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
		return "", err
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
		return "", err
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
		return "", err
	}

	return selected, nil
}

func resolveShell() string {
	if flagShell != "" {
		return flagShell
	}
	return "su -s /bin/sh www-data"
}

func showConnectHistory() (*history.Entry, error) {
	hist, err := history.Load()
	if err != nil || hist == nil {
		return nil, nil
	}

	labels := hist.Labels("connect")
	if len(labels) == 0 {
		return nil, nil
	}

	// Keep only the 10 most recent
	if len(labels) > 10 {
		labels = labels[:10]
	}

	labels = append([]string{"+ New connection"}, labels...)
	selected, err := ui.Select("Recent connections", labels)
	if err != nil {
		return nil, err
	}

	if selected == "+ New connection" {
		return nil, nil
	}

	label := selected[:strings.LastIndex(selected, " (")]
	return hist.FindByLabel("connect", label), nil
}

func replayLastConnect() error {
	hist, err := history.Load()
	if err != nil {
		return fmt.Errorf("no connection history found")
	}

	labels := hist.Labels("connect")
	if len(labels) == 0 {
		return fmt.Errorf("no connection history found")
	}

	label := labels[0][:strings.LastIndex(labels[0], " (")]
	entry := hist.FindByLabel("connect", label)
	if entry == nil {
		return fmt.Errorf("could not find last connection")
	}

	return replayConnectEntry(entry)
}

func replayConnectEntry(entry *history.Entry) error {
	var profile, cluster, service, container string
	for i := 0; i < len(entry.Args)-1; i += 2 {
		switch entry.Args[i] {
		case "--profile":
			profile = entry.Args[i+1]
		case "--cluster":
			cluster = entry.Args[i+1]
		case "--service":
			service = entry.Args[i+1]
		case "--container":
			container = entry.Args[i+1]
		}
	}

	ui.PrintStep("↻", fmt.Sprintf("Replaying: %s", entry.Label))

	if err := awsutil.EnsureSSOLogin(profile); err != nil {
		return err
	}

	client, err := ecs.NewClient(profile, flagRegion)
	if err != nil {
		return fmt.Errorf("failed to create AWS client: %w", err)
	}

	task, err := client.GetRunningTask(rootCmd.Context(), cluster, service)
	if err != nil {
		return fmt.Errorf("no running task found: %w", err)
	}

	shell := resolveShell()
	ui.PrintStep("▶", fmt.Sprintf("Connecting to %s/%s/%s", cluster, service, container))
	return client.ExecInteractive(rootCmd.Context(), cluster, task, container, shell, profile)
}
