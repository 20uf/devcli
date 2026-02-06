package infra

import (
	"context"
	"fmt"

	"github.com/20uf/devcli/internal/connection/application"
	"github.com/20uf/devcli/internal/connection/domain"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	ecsv2 "github.com/aws/aws-sdk-go-v2/service/ecs"
)

// CLIAdapter wires the infrastructure, domain, and application layers together.
// It's responsible for creating all repositories and the orchestrator.
// This is the composition root for the connection context.
type CLIAdapter struct {
	orchestrator *application.ConnectOrchestrator
	repos        *domain.AllRepositories
}

// NewCLIAdapter creates a new CLI adapter with all wired dependencies.
// It initializes AWS SDK, creates repositories, and builds the orchestrator.
func NewCLIAdapter(ctx context.Context, profile, region string) (*CLIAdapter, error) {
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

	// Step 2: Create all repositories
	repos := &domain.AllRepositories{
		Clusters:    NewECSClusterRepository(ecsClient),
		Services:    NewECSServiceRepository(ecsClient),
		Tasks:       NewECSTaskRepository(ecsClient),
		Connections: &NoOpConnectionRepository{}, // Placeholder for now
	}

	// Try to load file repository (optional, for history)
	if fileRepo, err := NewFileConnectionRepository(); err == nil {
		repos.Connections = fileRepo
	}

	// Step 3: Create the orchestrator
	orchestrator := application.NewConnectOrchestrator(repos)

	return &CLIAdapter{
		orchestrator: orchestrator,
		repos:        repos,
	}, nil
}

// Connect orchestrates a connection to an ECS container.
// This is the main entry point from the CLI layer.
func (a *CLIAdapter) Connect(ctx context.Context, clusterName, serviceName, containerName, shellCommand string) (domain.Connection, error) {
	req := application.ConnectRequest{
		ClusterName:   toPtr(clusterName),
		ServiceName:   toPtr(serviceName),
		ContainerName: toPtr(containerName),
		ShellCommand:  shellCommand,
	}

	return a.orchestrator.Connect(ctx, req)
}

// SelectClusterInteractive lists clusters for user selection.
// Returns the cluster name.
func (a *CLIAdapter) SelectClusterInteractive(ctx context.Context) (string, error) {
	clusters, err := a.repos.Clusters.ListClusters(ctx)
	if err != nil {
		return "", err
	}

	if len(clusters) == 0 {
		return "", domain.ErrNoClusterFound
	}

	// In a real CLI, you'd use UI selector here
	// For now, just return the first one
	return clusters[0].Name(), nil
}

// SelectServiceInteractive lists services in a cluster.
func (a *CLIAdapter) SelectServiceInteractive(ctx context.Context, clusterName string) (string, error) {
	cluster, err := domain.NewCluster(clusterName)
	if err != nil {
		return "", err
	}

	services, err := a.repos.Services.ListServices(ctx, cluster)
	if err != nil {
		return "", err
	}

	if len(services) == 0 {
		return "", domain.ErrNoServiceFound
	}

	// In a real CLI, you'd use UI selector here
	return services[0].Name(), nil
}

// GetConnections retrieves recent connections for replay.
func (a *CLIAdapter) GetConnections(ctx context.Context, limit int) ([]domain.Connection, error) {
	return a.repos.Connections.FindRecent(ctx, limit)
}

// NoOpConnectionRepository is a stub that does nothing.
// Used as a placeholder when file repository fails.
type NoOpConnectionRepository struct{}

func (r *NoOpConnectionRepository) Save(ctx context.Context, conn domain.Connection) error {
	return nil
}

func (r *NoOpConnectionRepository) FindByLabel(ctx context.Context, label string) (*domain.Connection, error) {
	return nil, nil
}

func (r *NoOpConnectionRepository) FindRecent(ctx context.Context, limit int) ([]domain.Connection, error) {
	return []domain.Connection{}, nil
}

// Helper: convert string to *string
func toPtr(s string) *string {
	if s == "" {
		return nil
	}
	return aws.String(s)
}
