package application

import (
	"context"
	"fmt"

	"github.com/20uf/devcli/internal/connection/domain"
)

// ConnectOrchestrator is the main use case for initiating a connection to an ECS container.
// It orchestrates the domain logic: selecting cluster → service → task → container.
// This application service is framework-agnostic and fully testable.
type ConnectOrchestrator struct {
	repos *domain.AllRepositories
}

// NewConnectOrchestrator creates a new orchestrator service.
func NewConnectOrchestrator(repos *domain.AllRepositories) *ConnectOrchestrator {
	return &ConnectOrchestrator{repos: repos}
}

// SelectClusterRequest represents the request to select a cluster.
type SelectClusterRequest struct {
	// If ClusterName is provided, skip selection and use this directly
	ClusterName *string
}

// SelectCluster selects a cluster, either from the provided name or by listing available clusters.
func (o *ConnectOrchestrator) SelectCluster(ctx context.Context, req SelectClusterRequest) (domain.Cluster, error) {
	if req.ClusterName != nil && *req.ClusterName != "" {
		// Direct selection: validate that the cluster exists
		return domain.NewCluster(*req.ClusterName)
	}

	// List available clusters
	clusters, err := o.repos.Clusters.ListClusters(ctx)
	if err != nil {
		return domain.Cluster{}, err
	}

	if len(clusters) == 0 {
		return domain.Cluster{}, domain.ErrNoClusterFound
	}

	// Return the first cluster; UI layer will handle multi-selection if needed
	return clusters[0], nil
}

// SelectServiceRequest represents the request to select a service.
type SelectServiceRequest struct {
	Cluster     domain.Cluster
	ServiceName *string // If provided, skip selection
}

// SelectService selects a service within a cluster.
func (o *ConnectOrchestrator) SelectService(ctx context.Context, req SelectServiceRequest) (domain.Service, error) {
	if req.ServiceName != nil && *req.ServiceName != "" {
		return domain.NewService(*req.ServiceName)
	}

	services, err := o.repos.Services.ListServices(ctx, req.Cluster)
	if err != nil {
		return domain.Service{}, err
	}

	if len(services) == 0 {
		return domain.Service{}, domain.ErrNoServiceFound
	}

	return services[0], nil
}

// SelectTaskRequest represents the request to select a task.
type SelectTaskRequest struct {
	Cluster domain.Cluster
	Service domain.Service
}

// SelectTask selects a running task for a service.
func (o *ConnectOrchestrator) SelectTask(ctx context.Context, req SelectTaskRequest) (domain.Task, error) {
	task, err := o.repos.Tasks.GetRunningTask(ctx, req.Cluster, req.Service)
	if err != nil {
		return domain.Task{}, fmt.Errorf("no running task for service %s: %w", req.Service.Name(), err)
	}
	return task, nil
}

// SelectContainerRequest represents the request to select a container.
type SelectContainerRequest struct {
	Task          domain.Task
	ContainerName *string // If provided, skip selection and auto-detection
}

// SelectContainer selects a container within a task.
// Strategy:
// 1. If ContainerName is provided, use it directly
// 2. If task has a preferred container (php, app, web, api), use it
// 3. If task has only one container, use it
// 4. Otherwise, delegate to UI layer (return all containers)
func (o *ConnectOrchestrator) SelectContainer(ctx context.Context, req SelectContainerRequest) (domain.Container, error) {
	if req.ContainerName != nil && *req.ContainerName != "" {
		return domain.NewContainer(*req.ContainerName)
	}

	// Auto-select a container using domain logic
	container, err := req.Task.SelectContainer()
	if err != nil {
		return domain.Container{}, err
	}

	return container, nil
}

// InitiateConnectionRequest represents a complete connection request.
type InitiateConnectionRequest struct {
	Cluster      domain.Cluster
	Service      domain.Service
	Task         domain.Task
	Container    domain.Container
	ShellCommand string
}

// InitiateConnection creates and prepares a connection for execution.
// This doesn't execute the connection; it just validates and returns it.
func (o *ConnectOrchestrator) InitiateConnection(ctx context.Context, req InitiateConnectionRequest) (domain.Connection, error) {
	if req.ShellCommand == "" {
		req.ShellCommand = "su -s /bin/sh www-data" // Default shell
	}

	conn, err := domain.NewConnection(
		fmt.Sprintf("conn-%d", ctx.Value("requestID")), // Simple ID; UI layer should provide UUID
		req.Cluster,
		req.Service,
		req.Task,
		req.Container,
		req.ShellCommand,
	)
	if err != nil {
		return domain.Connection{}, err
	}

	// Save for history/replay
	_ = o.repos.Connections.Save(ctx, conn)

	return conn, nil
}

// ConnectRequest represents a full connect request with all options.
type ConnectRequest struct {
	ClusterName   *string
	ServiceName   *string
	ContainerName *string
	ShellCommand  string
}

// Connect is the main orchestration flow: cluster → service → task → container.
// This is a complete use case that guides through the entire selection process.
func (o *ConnectOrchestrator) Connect(ctx context.Context, req ConnectRequest) (domain.Connection, error) {
	// Step 1: Select cluster
	cluster, err := o.SelectCluster(ctx, SelectClusterRequest{ClusterName: req.ClusterName})
	if err != nil {
		return domain.Connection{}, fmt.Errorf("cluster selection failed: %w", err)
	}

	// Step 2: Select service
	service, err := o.SelectService(ctx, SelectServiceRequest{
		Cluster:     cluster,
		ServiceName: req.ServiceName,
	})
	if err != nil {
		return domain.Connection{}, fmt.Errorf("service selection failed: %w", err)
	}

	// Step 3: Select task
	task, err := o.SelectTask(ctx, SelectTaskRequest{
		Cluster: cluster,
		Service: service,
	})
	if err != nil {
		return domain.Connection{}, fmt.Errorf("task selection failed: %w", err)
	}

	// Step 4: Select container
	container, err := o.SelectContainer(ctx, SelectContainerRequest{
		Task:          task,
		ContainerName: req.ContainerName,
	})
	if err != nil {
		return domain.Connection{}, fmt.Errorf("container selection failed: %w", err)
	}

	// Step 5: Initiate connection
	return o.InitiateConnection(ctx, InitiateConnectionRequest{
		Cluster:      cluster,
		Service:      service,
		Task:         task,
		Container:    container,
		ShellCommand: req.ShellCommand,
	})
}
