package domain

import "context"

// ClusterRepository defines the interface for accessing ECS clusters.
type ClusterRepository interface {
	// ListClusters returns all available ECS clusters, sorted by name.
	ListClusters(ctx context.Context) ([]Cluster, error)
}

// ServiceRepository defines the interface for accessing ECS services.
type ServiceRepository interface {
	// ListServices returns all services in a given cluster, sorted by name.
	ListServices(ctx context.Context, cluster Cluster) ([]Service, error)
}

// TaskRepository defines the interface for accessing ECS tasks.
type TaskRepository interface {
	// GetRunningTask returns the first running task for a given service.
	// Returns ErrNoTaskFound if no task is running.
	GetRunningTask(ctx context.Context, cluster Cluster, service Service) (Task, error)
}

// ConnectionRepository defines the interface for persisting connections.
// Used to save and retrieve connections for replay functionality.
type ConnectionRepository interface {
	// Save persists a connection record.
	Save(ctx context.Context, conn Connection) error

	// FindByLabel retrieves a connection by its label.
	FindByLabel(ctx context.Context, label string) (*Connection, error)

	// FindRecent retrieves the N most recent connections.
	FindRecent(ctx context.Context, limit int) ([]Connection, error)
}

// AllRepositories bundles all repositories needed for the connection context.
// This is used as a parameter in application services.
type AllRepositories struct {
	Clusters   ClusterRepository
	Services   ServiceRepository
	Tasks      TaskRepository
	Connections ConnectionRepository
}
