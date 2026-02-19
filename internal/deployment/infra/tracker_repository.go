package infra

import "context"

import "github.com/20uf/devcli/internal/deployment/domain"

// TrackerRepository defines the interface for persisting and retrieving tracked deployments.
type TrackerRepository interface {
	// Save persists a tracked deployment.
	Save(ctx context.Context, tracked domain.TrackedDeployment) error

	// List retrieves all tracked deployments.
	List(ctx context.Context) ([]domain.TrackedDeployment, error)

	// GetByID retrieves a specific tracked deployment by ID.
	GetByID(ctx context.Context, id string) (*domain.TrackedDeployment, error)

	// Remove removes a tracked deployment.
	Remove(ctx context.Context, id string) error

	// ListActive retrieves only active (in-progress or queued) deployments.
	ListActive(ctx context.Context) ([]domain.TrackedDeployment, error)

	// Cleanup removes stale deployments (completed and older than TTL).
	Cleanup(ctx context.Context, maxAge int64) (removed int, err error)
}
