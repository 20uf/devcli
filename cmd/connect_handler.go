package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/20uf/devcli/internal/connection/application"
	"github.com/20uf/devcli/internal/connection/domain"
	"github.com/20uf/devcli/internal/connection/infra"
	"github.com/20uf/devcli/internal/history"
	"github.com/20uf/devcli/internal/ui"
	"github.com/aws/aws-sdk-go-v2/config"
	ecsv2 "github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/spf13/cobra"
)

// ConnectHandler bridges the CLI layer and domain layer.
// It orchestrates user input (UI) with business logic (domain).
type ConnectHandler struct {
	orchestrator *application.ConnectOrchestrator
	repos        *domain.AllRepositories
	history      *history.Store
	profile      string // AWS profile for SSO
}

// NewConnectHandler creates a handler with all dependencies wired.
func NewConnectHandler(ctx context.Context, profile, region string) (*ConnectHandler, error) {
	// Auto-detect default profile if not provided
	if profile == "" {
		profile = detectDefaultProfile()
	}

	// Step 1: Initialize AWS SDK
	var opts []func(*config.LoadOptions) error
	if profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(profile))
	}
	if region != "" {
		opts = append(opts, config.WithRegion(region))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS config: %w", err)
	}

	ecsClient := ecsv2.NewFromConfig(cfg)

	// Step 2: Create repositories (infrastructure layer)
	repos := &domain.AllRepositories{
		Clusters:    infra.NewECSClusterRepository(ecsClient),
		Services:    infra.NewECSServiceRepository(ecsClient),
		Tasks:       infra.NewECSTaskRepository(ecsClient),
		Connections: &infra.NoOpConnectionRepository{}, // TODO: use FileConnectionRepository
	}

	// Step 3: Load history for replay
	hist, _ := history.Load()

	return &ConnectHandler{
		orchestrator: application.NewConnectOrchestrator(repos),
		repos:        repos,
		history:      hist,
		profile:      profile,
	}, nil
}

// Handle orchestrates the complete connection flow.
// flagXxx parameters can be empty (user will select) or populated (non-interactive).
func (h *ConnectHandler) Handle(cmd *cobra.Command, clusterFlag, serviceFlag, containerFlag, shellFlag string) error {
	ctx := cmd.Context()

	// Ensure SSO is authenticated before proceeding
	if err := h.ensureSSO(ctx); err != nil {
		return err
	}

	// Non-interactive mode: all flags provided
	if clusterFlag != "" && serviceFlag != "" && containerFlag != "" {
		conn, err := h.orchestrator.Connect(ctx, application.ConnectRequest{
			ClusterName:   &clusterFlag,
			ServiceName:   &serviceFlag,
			ContainerName: &containerFlag,
			ShellCommand:  shellFlag,
		})
		if err != nil {
			return err
		}
		return h.executeConnection(ctx, conn)
	}

	// Interactive mode: guide user through selection
	return h.interactiveFlow(ctx, clusterFlag, serviceFlag, containerFlag, shellFlag)
}

// interactiveFlow guides user through cluster → service → task → container selection.
func (h *ConnectHandler) interactiveFlow(ctx context.Context, clusterFlag, serviceFlag, containerFlag, shellFlag string) error {
	// Step 0: Show history if no flags
	if clusterFlag == "" && serviceFlag == "" && containerFlag == "" {
		if histConn, err := h.showHistoryMenu(); err == nil && histConn != nil {
			ui.PrintStep("↻", fmt.Sprintf("Replaying: %s", histConn.String()))
			return h.executeConnection(ctx, *histConn)
		}
		// User selected "New connection" or pressed ESC, continue to interactive flow
	}

	// Step 1: Select cluster
	clusters, err := h.repos.Clusters.ListClusters(ctx)
	if err != nil {
		return err
	}

	clusterNames := make([]string, len(clusters))
	for i, c := range clusters {
		clusterNames[i] = c.Name()
	}

	if clusterFlag != "" {
		// Use provided cluster
		clusterNames = []string{clusterFlag}
	}

	selectedClusterName, err := ui.Select("Select cluster", clusterNames)
	if err != nil {
		ui.PrintWarning("Cancelled - returning to menu")
		return nil // User pressed ESC
	}

	cluster, _ := domain.NewCluster(selectedClusterName)

	// Step 2: Select service
	services, err := h.repos.Services.ListServices(ctx, cluster)
	if err != nil {
		return err
	}

	serviceNames := make([]string, len(services))
	for i, s := range services {
		serviceNames[i] = s.Name()
	}

	if serviceFlag != "" {
		serviceNames = []string{serviceFlag}
	}

	selectedServiceName, err := ui.Select("Select service", serviceNames)
	if err != nil {
		return nil // User pressed ESC
	}

	service, _ := domain.NewService(selectedServiceName)

	// Step 3: Get running task
	task, err := h.repos.Tasks.GetRunningTask(ctx, cluster, service)
	if err != nil {
		ui.PrintWarning(fmt.Sprintf("No running task for %s: %s", service.Name(), err))
		return nil
	}

	// Step 4: Select container (with auto-selection logic from domain)
	container, _ := task.SelectContainer()

	// If multiple containers and no preference, show selection
	if len(task.Containers()) > 1 && !container.IsPreferred() {
		if containerFlag != "" {
			container, _ = domain.NewContainer(containerFlag)
		} else {
			containerNames := make([]string, len(task.Containers()))
			for i, c := range task.Containers() {
				containerNames[i] = c.Name()
			}

			selectedContainerName, err := ui.Select("Select container", containerNames)
			if err != nil {
				return nil // User pressed ESC
			}

			container, _ = domain.NewContainer(selectedContainerName)
		}
	}

	// Step 5: Initiate and execute connection
	conn, err := h.orchestrator.InitiateConnection(ctx, application.InitiateConnectionRequest{
		Cluster:      cluster,
		Service:      service,
		Task:         task,
		Container:    container,
		ShellCommand: h.resolveShell(shellFlag),
	})
	if err != nil {
		return err
	}

	return h.executeConnection(ctx, conn)
}

// executeConnection saves to history and executes the AWS CLI command.
func (h *ConnectHandler) executeConnection(ctx context.Context, conn domain.Connection) error {
	// Save to history for replay
	if h.history != nil {
		label := conn.String()
		h.history.Add("connect", label, []string{
			"--cluster", conn.Cluster().Name(),
			"--service", conn.Service().Name(),
			"--container", conn.Container().Name(),
		})
		h.history.Save() //nolint:errcheck
	}

	ui.PrintStep("▶", fmt.Sprintf("Connecting to %s", conn.String()))

	// Execute AWS CLI command via ECS Exec
	// Build AWS SSM session command for ECS container
	args := []string{
		"ecs", "execute-command",
		"--cluster", conn.Cluster().Name(),
		"--task", conn.Task().ID(),
		"--container", conn.Container().Name(),
		"--interactive",
		"--command", conn.ShellCommand(),
	}

	// Add profile if specified
	if h.profile != "" {
		args = append(args, "--profile", h.profile)
	}

	cmd := exec.Command("aws", args...)

	// Attach stdin/stdout/stderr for interactive session
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Execute and return result
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}

	return nil
}

// showHistoryMenu displays recent connections for replay.
func (h *ConnectHandler) showHistoryMenu() (*domain.Connection, error) {
	if h.history == nil {
		return nil, nil
	}

	labels := h.history.Labels("connect")
	if len(labels) == 0 {
		return nil, nil
	}

	if len(labels) > 10 {
		labels = labels[:10]
	}

	labels = append([]string{"+ New connection"}, labels...)
	selected, err := ui.Select("Recent connections", labels)
	if err != nil {
		return nil, err // User pressed ESC
	}

	if selected == "+ New connection" {
		return nil, nil // Signal to start fresh
	}

	// Extract label prefix (remove timestamp)
	labelPrefix := selected[:strings.LastIndex(selected, " (")]

	// Find the history entry
	entry := h.history.FindByLabel("connect", labelPrefix)
	if entry == nil {
		return nil, nil
	}

	// Parse cluster/service from history args
	helper := infra.NewIntegrationHelper(entry.Command, entry.Label, entry.Args)
	_, clusterName, serviceName, containerName, shell := helper.ParseConnectionArgs()

	// Reconstruct domain objects
	cluster, err := domain.NewCluster(clusterName)
	if err != nil {
		return nil, err
	}

	service, err := domain.NewService(serviceName)
	if err != nil {
		return nil, err
	}

	container, err := domain.NewContainer(containerName)
	if err != nil {
		return nil, err
	}

	// Fetch REAL running task from AWS (not reconstructed)
	task, err := h.repos.Tasks.GetRunningTask(context.Background(), cluster, service)
	if err != nil {
		ui.PrintWarning(fmt.Sprintf("No running task found for %s/%s: %s", clusterName, serviceName, err))
		return nil, nil
	}

	// Create connection with real task
	conn, err := domain.NewConnection(
		fmt.Sprintf("replay-%s", task.ID()),
		cluster,
		service,
		task,
		container,
		shell,
	)
	if err != nil {
		return nil, err
	}

	return &conn, nil
}

// detectDefaultProfile finds a default AWS profile for SSO.
// Priority: playiad-dev > playiad-testing > playiad-preprod > first SSO profile
func detectDefaultProfile() string {
	priorities := []string{"playiad-dev", "playiad-testing", "playiad-preprod"}

	// Check prioritized profiles
	for _, p := range priorities {
		if isValidProfile(p) {
			return p
		}
	}

	// If none found, try to find any SSO profile
	cfgFile := os.ExpandEnv("$HOME/.aws/config")
	data, err := os.ReadFile(cfgFile)
	if err != nil {
		return "" // No config file, return empty
	}

	// Simple check: if file contains sso_start_url, we have SSO configured
	if strings.Contains(string(data), "sso_start_url") {
		// Return first profile section found
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "[profile ") && strings.HasSuffix(line, "]") {
				profile := strings.TrimPrefix(line, "[profile ")
				profile = strings.TrimSuffix(profile, "]")
				if isValidProfile(profile) {
					return profile
				}
			}
		}
	}

	return ""
}

// isValidProfile checks if a profile exists in AWS config
func isValidProfile(profileName string) bool {
	cfgFile := os.ExpandEnv("$HOME/.aws/config")
	data, err := os.ReadFile(cfgFile)
	if err != nil {
		return false
	}

	content := string(data)
	searchStr := fmt.Sprintf("[profile %s]", profileName)
	// Special case for default profile
	if profileName == "default" {
		return strings.Contains(content, "[default]")
	}
	return strings.Contains(content, searchStr)
}

// ensureSSO verifies AWS SSO authentication and prompts for login if needed.
func (h *ConnectHandler) ensureSSO(ctx context.Context) error {
	if h.profile == "" {
		return fmt.Errorf("no AWS profile found - configure SSO with: aws configure sso")
	}

	// Show loader while checking SSO
	spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	spinnerIdx := 0
	done := make(chan bool)
	go func() {
		for {
			select {
			case <-done:
				return
			default:
				fmt.Printf("\r%s Checking AWS credentials...", ui.MutedStyle.Render(spinner[spinnerIdx%len(spinner)]))
				spinnerIdx++
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	// Check if SSO credentials are valid by attempting a test AWS call
	checkCmd := exec.CommandContext(ctx, "aws", "sts", "get-caller-identity", "--profile", h.profile)
	checkCmd.Stdout = nil
	checkCmd.Stderr = nil

	err := checkCmd.Run()
	close(done)
	fmt.Print("\r\033[K") // Clear line

	if err == nil {
		return nil // Already authenticated
	}

	// SSO not authenticated, prompt user to login
	ui.PrintStep("🔐", fmt.Sprintf("AWS SSO authentication required for profile: %s", h.profile))
	ui.PrintInfo("Opening browser", "Authenticate with your AWS organization")

	// Launch SSO login
	loginCmd := exec.Command("aws", "sso", "login", "--profile", h.profile)
	loginCmd.Stdin = os.Stdin
	loginCmd.Stdout = os.Stdout
	loginCmd.Stderr = os.Stderr

	if err := loginCmd.Run(); err != nil {
		return fmt.Errorf("SSO login failed: %w", err)
	}

	// Verify authentication succeeded
	verifyCmd := exec.CommandContext(ctx, "aws", "sts", "get-caller-identity", "--profile", h.profile)
	verifyCmd.Stdout = nil
	verifyCmd.Stderr = nil

	if err := verifyCmd.Run(); err != nil {
		return fmt.Errorf("SSO authentication verification failed: %w", err)
	}

	ui.PrintSuccess(fmt.Sprintf("SSO authenticated - Profile: %s", h.profile))
	return nil
}

// resolveShell returns the shell command to use.
func (h *ConnectHandler) resolveShell(flagShell string) string {
	if flagShell != "" {
		return flagShell
	}
	return "su -s /bin/sh www-data"
}
