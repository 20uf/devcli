package application

import (
	"context"
	"testing"

	"github.com/20uf/devcli/internal/connection/domain"
)

// Mocks for testing

// MockClusterRepository is a stub implementation for testing.
type MockClusterRepository struct {
	clusters []domain.Cluster
	err      error
}

func (m *MockClusterRepository) ListClusters(ctx context.Context) ([]domain.Cluster, error) {
	return m.clusters, m.err
}

// MockServiceRepository is a stub implementation for testing.
type MockServiceRepository struct {
	services []domain.Service
	err      error
}

func (m *MockServiceRepository) ListServices(ctx context.Context, cluster domain.Cluster) ([]domain.Service, error) {
	return m.services, m.err
}

// MockTaskRepository is a stub implementation for testing.
type MockTaskRepository struct {
	task domain.Task
	err  error
}

func (m *MockTaskRepository) GetRunningTask(ctx context.Context, cluster domain.Cluster, service domain.Service) (domain.Task, error) {
	return m.task, m.err
}

// MockConnectionRepository is a stub implementation for testing.
type MockConnectionRepository struct {
	saved []*domain.Connection
	err   error
}

func (m *MockConnectionRepository) Save(ctx context.Context, conn domain.Connection) error {
	m.saved = append(m.saved, &conn)
	return m.err
}

func (m *MockConnectionRepository) FindByLabel(ctx context.Context, label string) (*domain.Connection, error) {
	return nil, nil
}

func (m *MockConnectionRepository) FindRecent(ctx context.Context, limit int) ([]domain.Connection, error) {
	return nil, nil
}

// Test: Full connection flow
func TestConnectOrchestrator_Connect_Success(t *testing.T) {
	// Arrange: Set up test data
	cluster, _ := domain.NewCluster("production")
	service, _ := domain.NewService("api")
	container, _ := domain.NewContainer("php")
	task := domain.NewTask("task-123", []domain.Container{container}, domain.TaskStatusRunning)

	repos := &domain.AllRepositories{
		Clusters: &MockClusterRepository{
			clusters: []domain.Cluster{cluster},
		},
		Services: &MockServiceRepository{
			services: []domain.Service{service},
		},
		Tasks: &MockTaskRepository{
			task: task,
		},
		Connections: &MockConnectionRepository{},
	}

	orchestrator := NewConnectOrchestrator(repos)
	ctx := context.Background()

	// Act: Execute the connection flow
	conn, err := orchestrator.Connect(ctx, ConnectRequest{})

	// Assert: Verify the result
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if conn.Cluster().Name() != "production" {
		t.Errorf("expected cluster 'production', got '%s'", conn.Cluster().Name())
	}

	if conn.Service().Name() != "api" {
		t.Errorf("expected service 'api', got '%s'", conn.Service().Name())
	}

	if conn.Container().Name() != "php" {
		t.Errorf("expected container 'php', got '%s'", conn.Container().Name())
	}

	expectedShell := "su -s /bin/sh www-data"
	if conn.ShellCommand() != expectedShell {
		t.Errorf("expected shell '%s', got '%s'", expectedShell, conn.ShellCommand())
	}

	// Connection is created but not yet executed (that happens in infra layer)
	if conn.IsInitiated() {
		t.Errorf("connection should not be marked as initiated yet (execution happens in infra)")
	}
}

// Test: Container auto-selection (prefers "php")
func TestConnectOrchestrator_SelectContainer_PreferredContainer(t *testing.T) {
	// Arrange: Task with multiple containers, "php" should be preferred
	php, _ := domain.NewContainer("php")
	nginx, _ := domain.NewContainer("nginx")
	task := domain.NewTask("task-123", []domain.Container{nginx, php}, domain.TaskStatusRunning)

	orchestrator := &ConnectOrchestrator{}

	// Act: Select container from task
	selected, err := orchestrator.SelectContainer(context.Background(), SelectContainerRequest{
		Task: task,
	})

	// Assert: "php" should be selected
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if selected.Name() != "php" {
		t.Errorf("expected 'php' container, got '%s'", selected.Name())
	}
}

// Test: Container fallback when no preferred container
func TestConnectOrchestrator_SelectContainer_SingleContainer(t *testing.T) {
	// Arrange: Task with a single non-preferred container
	app, _ := domain.NewContainer("app")
	task := domain.NewTask("task-123", []domain.Container{app}, domain.TaskStatusRunning)

	orchestrator := &ConnectOrchestrator{}

	// Act
	selected, err := orchestrator.SelectContainer(context.Background(), SelectContainerRequest{
		Task: task,
	})

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if selected.Name() != "app" {
		t.Errorf("expected 'app' container, got '%s'", selected.Name())
	}
}

// Test: Explicit container selection
func TestConnectOrchestrator_SelectContainer_Explicit(t *testing.T) {
	// Arrange
	nginx, _ := domain.NewContainer("nginx")
	task := domain.NewTask("task-123", []domain.Container{nginx}, domain.TaskStatusRunning)
	containerName := "nginx"

	orchestrator := &ConnectOrchestrator{}

	// Act
	selected, err := orchestrator.SelectContainer(context.Background(), SelectContainerRequest{
		Task:          task,
		ContainerName: &containerName,
	})

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if selected.Name() != "nginx" {
		t.Errorf("expected 'nginx', got '%s'", selected.Name())
	}
}

// Test: No clusters available
func TestConnectOrchestrator_SelectCluster_NoClusters(t *testing.T) {
	// Arrange
	repos := &domain.AllRepositories{
		Clusters: &MockClusterRepository{
			clusters: []domain.Cluster{},
			err:      domain.ErrNoClusterFound,
		},
	}

	orchestrator := NewConnectOrchestrator(repos)

	// Act
	_, err := orchestrator.SelectCluster(context.Background(), SelectClusterRequest{})

	// Assert
	if err != domain.ErrNoClusterFound {
		t.Errorf("expected ErrNoClusterFound, got %v", err)
	}
}

// Test: No services in cluster
func TestConnectOrchestrator_SelectService_NoServices(t *testing.T) {
	// Arrange
	cluster, _ := domain.NewCluster("production")
	repos := &domain.AllRepositories{
		Services: &MockServiceRepository{
			services: []domain.Service{},
			err:      domain.ErrNoServiceFound,
		},
	}

	orchestrator := NewConnectOrchestrator(repos)

	// Act
	_, err := orchestrator.SelectService(context.Background(), SelectServiceRequest{
		Cluster: cluster,
	})

	// Assert
	if err != domain.ErrNoServiceFound {
		t.Errorf("expected ErrNoServiceFound, got %v", err)
	}
}

// Test: No running tasks
func TestConnectOrchestrator_SelectTask_NoRunningTasks(t *testing.T) {
	// Arrange
	cluster, _ := domain.NewCluster("production")
	service, _ := domain.NewService("api")
	repos := &domain.AllRepositories{
		Tasks: &MockTaskRepository{
			err: domain.ErrNoTaskFound,
		},
	}

	orchestrator := NewConnectOrchestrator(repos)

	// Act
	_, err := orchestrator.SelectTask(context.Background(), SelectTaskRequest{
		Cluster: cluster,
		Service: service,
	})

	// Assert
	if err == nil {
		t.Errorf("expected error, got nil")
	}
}

// Test: Invalid connection (missing shell command)
func TestConnectOrchestrator_InitiateConnection_InvalidShell(t *testing.T) {
	// Arrange
	cluster, _ := domain.NewCluster("production")
	service, _ := domain.NewService("api")
	container, _ := domain.NewContainer("php")
	task := domain.NewTask("task-123", []domain.Container{container}, domain.TaskStatusRunning)

	orchestrator := &ConnectOrchestrator{
		repos: &domain.AllRepositories{
			Connections: &MockConnectionRepository{},
		},
	}

	// The default shell is applied, so this should succeed
	req := InitiateConnectionRequest{
		Cluster:   cluster,
		Service:   service,
		Task:      task,
		Container: container,
		// ShellCommand left empty, should use default
	}

	// Act
	conn, err := orchestrator.InitiateConnection(context.Background(), req)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if conn.ShellCommand() == "" {
		t.Errorf("shell command should not be empty")
	}
}

// Acceptance Test: User connects to a container in a running service
func TestAcceptance_ConnectToContainer(t *testing.T) {
	// Scenario: A developer wants to connect to a PHP container running in the "api" service
	// in the "production" cluster.

	// Arrange: Set up the infrastructure
	cluster, _ := domain.NewCluster("production")
	service, _ := domain.NewService("api")
	phpContainer, _ := domain.NewContainer("php")
	nginxContainer, _ := domain.NewContainer("nginx")
	task := domain.NewTask(
		"arn:aws:ecs:us-east-1:123456789:task/production/abcd1234",
		[]domain.Container{nginxContainer, phpContainer},
		domain.TaskStatusRunning,
	)

	repos := &domain.AllRepositories{
		Clusters: &MockClusterRepository{
			clusters: []domain.Cluster{cluster},
		},
		Services: &MockServiceRepository{
			services: []domain.Service{service},
		},
		Tasks: &MockTaskRepository{
			task: task,
		},
		Connections: &MockConnectionRepository{},
	}

	orchestrator := NewConnectOrchestrator(repos)

	// Act: Execute the full connection flow
	conn, err := orchestrator.Connect(context.Background(), ConnectRequest{})

	// Assert: Verify the connection is properly configured
	if err != nil {
		t.Fatalf("connection failed: %v", err)
	}

	// The orchestrator should select the correct resources
	if conn.Cluster().Name() != "production" {
		t.Errorf("expected cluster 'production'")
	}

	if conn.Service().Name() != "api" {
		t.Errorf("expected service 'api'")
	}

	// It should prefer the "php" container over "nginx"
	if conn.Container().Name() != "php" {
		t.Errorf("expected container 'php' (preferred)")
	}

	// It should have a shell command configured
	if conn.ShellCommand() == "" {
		t.Errorf("shell command should be configured")
	}

	// The connection should be saved to history
	connRepo := repos.Connections.(*MockConnectionRepository)
	if len(connRepo.saved) != 1 {
		t.Errorf("expected 1 saved connection, got %d", len(connRepo.saved))
	}
}
