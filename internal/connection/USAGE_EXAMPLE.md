# Connection Context - Usage Examples

This document shows practical examples of using the refactored Connection Context.

## 1. Unit Testing (No AWS dependency)

```go
package main

import (
    "context"
    "testing"
    "github.com/20uf/devcli/internal/connection/application"
    "github.com/20uf/devcli/internal/connection/domain"
)

// Mock repositories for testing
type MockClusterRepository struct {
    clusters []domain.Cluster
}

func (m *MockClusterRepository) ListClusters(ctx context.Context) ([]domain.Cluster, error) {
    return m.clusters, nil
}

// Test the complete connection flow without AWS
func TestConnectionFlow(t *testing.T) {
    // Arrange: Create test data
    cluster, _ := domain.NewCluster("production")
    service, _ := domain.NewService("api")
    container, _ := domain.NewContainer("php")
    task := domain.NewTask("task-123", []domain.Container{container}, domain.TaskStatusRunning)

    // Create mocked repositories
    repos := &domain.AllRepositories{
        Clusters: &MockClusterRepository{clusters: []domain.Cluster{cluster}},
        Services: mockServices(service),
        Tasks: mockTasks(task),
        Connections: mockConnections(),
    }

    // Act: Use the orchestrator (pure business logic)
    orch := application.NewConnectOrchestrator(repos)
    conn, err := orch.Connect(context.Background(), application.ConnectRequest{})

    // Assert: Verify result
    if err != nil {
        t.Fatalf("expected no error: %v", err)
    }
    if conn.Container().Name() != "php" {
        t.Errorf("expected container 'php', got '%s'", conn.Container().Name())
    }
}
```

## 2. CLI Integration

```go
// In cmd/connect.go
func runConnect(cmd *cobra.Command, args []string) error {
    // Create adapter (wires AWS SDK + repositories)
    adapter, err := infra.NewCLIAdapter(cmd.Context(), flagProfile, flagRegion)
    if err != nil {
        return err
    }

    // Call the orchestrator with user input
    conn, err := adapter.Connect(
        cmd.Context(),
        flagCluster,      // May be empty (user will select)
        flagService,      // May be empty (user will select)
        flagContainer,    // May be empty (auto-select or user picks)
        resolveShell(),
    )
    if err != nil {
        return err
    }

    // Execute the connection
    return executeConnection(cmd.Context(), conn)
}
```

## 3. Container Auto-Selection Logic

The domain handles smart container selection:

```go
// Task with multiple containers
php := domain.NewContainer("php")
nginx := domain.NewContainer("nginx")
task := domain.NewTask("task-123",
    []domain.Container{nginx, php},
    domain.TaskStatusRunning)

// Auto-select: "php" is preferred
container, _ := task.SelectContainer()
// Result: container.Name() == "php"

// Find specific container by name
php, _ := task.FindContainerByName("php")

// Check if container is preferred
if php.IsPreferred() {
    // "php" is in the preferred list: ["php", "app", "web", "api"]
}
```

## 4. Type-Safe Connection Building

```go
// Create value objects (validated)
cluster, err := domain.NewCluster("production")
if err != nil {
    // Empty cluster names are rejected
}

service, err := domain.NewService("api")
if err != nil {
    // Empty service names are rejected
}

container, err := domain.NewContainer("php")
if err != nil {
    // Empty container names are rejected
}

// Build a task
task := domain.NewTask(
    "task-arn",
    []domain.Container{container},
    domain.TaskStatusRunning,
)

// Create a connection (aggregate root guards invariants)
conn, err := domain.NewConnection(
    "conn-123",
    cluster,
    service,
    task,
    container,
    "su -s /bin/sh www-data",
)
if err != nil {
    // Connection is invalid if container doesn't exist in task
}

// Use the connection
fmt.Printf("Connecting to: %s\n", conn.String())
// Output: "Connecting to: production/api/php"
```

## 5. Repository Implementation

### Implementing for a New Cloud Provider (GCP)

```go
// internal/connection/infra/gcp_mapper.go
package infra

import "github.com/20uf/devcli/internal/connection/domain"

type GCPMapper struct{}

func (m *GCPMapper) MapRunInstanceToTask(instance *compute.Instance) (domain.Task, error) {
    // Extract containers from instance
    containers := []domain.Container{}
    for _, c := range instance.Containers {
        container, _ := domain.NewContainer(c.Name)
        containers = append(containers, container)
    }
    return domain.NewTask(instance.ID, containers, domain.TaskStatusRunning), nil
}

// internal/connection/infra/gcp_repository.go
type GCPTaskRepository struct {
    client *compute.Client
    mapper *GCPMapper
}

func (r *GCPTaskRepository) GetRunningTask(ctx context.Context,
    cluster domain.Cluster,
    service domain.Service) (domain.Task, error) {

    // Query GCP Compute Engine instead of ECS
    instance, err := r.client.GetInstance(ctx, cluster.Name(), service.Name())
    if err != nil {
        return domain.Task{}, domain.ErrNoTaskFound
    }

    return r.mapper.MapRunInstanceToTask(instance)
}

// Now you can use GCP instead of AWS with the same domain logic:
repos := &domain.AllRepositories{
    Clusters: gcp.NewGCPClusterRepository(gcpClient),
    Services: gcp.NewGCPServiceRepository(gcpClient),
    Tasks: gcp.NewGCPTaskRepository(gcpClient),      // ← GCP instead of ECS
}

orch := application.NewConnectOrchestrator(repos)
conn, _ := orch.Connect(ctx, request)
// Same domain logic, different cloud provider ✓
```

## 6. Error Handling

```go
import "github.com/20uf/devcli/internal/connection/domain"

conn, err := orchestrator.Connect(ctx, request)
if err != nil {
    switch err {
    case domain.ErrNoClusterFound:
        ui.PrintError("No ECS clusters available")
    case domain.ErrNoServiceFound:
        ui.PrintError("No services in this cluster")
    case domain.ErrNoTaskFound:
        ui.PrintError("No running tasks for this service")
    case domain.ErrNoContainerFound:
        ui.PrintError("No containers in this task")
    default:
        ui.PrintError(fmt.Sprintf("Error: %v", err))
    }
    return err
}
```

## 7. Replaying Connections from History

```go
// Get recent connections
recentConnections, _ := repos.Connections.FindRecent(ctx, 10)

for _, conn := range recentConnections {
    fmt.Printf("- %s (created at %s)\n",
        conn.String(),
        conn.CreatedAt().Format("2006-01-02 15:04:05"),
    )
}

// Replay a specific connection
savedConn, _ := repos.Connections.FindByLabel(ctx, "production/api/php")
if savedConn != nil {
    return executeConnection(ctx, *savedConn)
}
```

## 8. Orchestrator Step-by-Step

You can also use the orchestrator step-by-step for more control:

```go
ctx := context.Background()

// Step 1: Select cluster
cluster, err := orchestrator.SelectCluster(ctx,
    application.SelectClusterRequest{
        ClusterName: nil, // nil = user will select
    })

// Step 2: Select service
service, err := orchestrator.SelectService(ctx,
    application.SelectServiceRequest{
        Cluster: cluster,
        ServiceName: nil,
    })

// Step 3: Get task
task, err := orchestrator.SelectTask(ctx,
    application.SelectTaskRequest{
        Cluster: cluster,
        Service: service,
    })

// Step 4: Select container
container, err := orchestrator.SelectContainer(ctx,
    application.SelectContainerRequest{
        Task: task,
        ContainerName: nil, // nil = auto-select based on domain logic
    })

// Step 5: Initiate connection
conn, err := orchestrator.InitiateConnection(ctx,
    application.InitiateConnectionRequest{
        Cluster: cluster,
        Service: service,
        Task: task,
        Container: container,
        ShellCommand: "su -s /bin/sh www-data",
    })
```

---

## Key Takeaways

✅ **Domain logic is pure** - no AWS, no Cobra, no I/O dependencies
✅ **Easy to test** - use mocks for repositories
✅ **Easy to extend** - add new cloud providers with new repositories
✅ **Type-safe** - cluster/service/container are value objects, not strings
✅ **Self-documenting** - code reads like business logic ("select cluster → service → task → container")

The domain never knows about AWS, GitHub, Cobra, or any framework. It's pure business logic that can be reused anywhere.
